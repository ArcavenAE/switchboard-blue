package metrics_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/arcavenae/switchboard/internal/metrics"
	"github.com/arcavenae/switchboard/internal/paths"
)

// ── fakes ──────────────────────────────────────────────────────────────────

// fakePathsListSource implements metrics.PathsListSource for tests.
type fakePathsListSource struct {
	snaps map[string]paths.PathSnapshot
}

func (f *fakePathsListSource) AllSnapshots() map[string]paths.PathSnapshot {
	out := make(map[string]paths.PathSnapshot, len(f.snaps))
	for k, v := range f.snaps {
		out[k] = v
	}
	return out
}

// fakeRouterMetricsSource implements metrics.RouterMetricsSource for tests.
type fakeRouterMetricsSource struct {
	metrics map[string]metrics.RouterMetricsResponse
}

func (f *fakeRouterMetricsSource) SVTNMetrics(svtnID string) (metrics.RouterMetricsResponse, error) {
	m, ok := f.metrics[svtnID]
	if !ok {
		return metrics.RouterMetricsResponse{}, &rpcError{code: "E-RPC-011", message: "SVTN not found: " + svtnID}
	}
	return m, nil
}

// rpcError is a test-local error type that carries the E-RPC-011 code.
type rpcError struct {
	code    string
	message string
}

func (e *rpcError) Error() string { return e.message }

// ── AC-001: TestDaemonPathsList_HandlerRegistered ──────────────────────────

// TestDaemonPathsList_HandlerRegistered verifies that PathsList returns a
// PathsListResponse with at least one PathEntry when the source has a path,
// and that the entry fields are populated from the snapshot.
//
// AC-001; BC-2.06.003 PC-1.
func TestDaemonPathsList_HandlerRegistered(t *testing.T) {
	t.Parallel()

	snap := paths.PathSnapshot{
		EWMARTTMs:   15.0,
		LossPct:     0.1,
		Active:      true,
		Degraded:    false,
		P99RTTMs:    22.0,
		SampleCount: 10,
	}
	src := &fakePathsListSource{
		snaps: map[string]paths.PathSnapshot{
			"path-abc": snap,
		},
	}

	resp, err := metrics.PathsList(context.Background(), nil, src)
	if err != nil {
		t.Fatalf("PathsList returned unexpected error: %v", err)
	}
	if len(resp.Paths) != 1 {
		t.Fatalf("expected 1 path in response; got %d", len(resp.Paths))
	}

	entry := resp.Paths[0]
	if entry.PathID != "path-abc" {
		t.Errorf("path_id: got %q; want %q", entry.PathID, "path-abc")
	}
	if entry.RTTMs != 15.0 {
		t.Errorf("rtt_ms: got %v; want 15.0", entry.RTTMs)
	}
	if entry.LossPct != 0.1 {
		t.Errorf("loss_pct: got %v; want 0.1", entry.LossPct)
	}
	if entry.Status == "" {
		t.Errorf("status: empty string; want non-empty")
	}
}

// TestDaemonPathsList_EmptySource verifies EC-001: no paths → empty list + message.
//
// AC-001; BC-2.06.003 EC-001.
func TestDaemonPathsList_EmptySource(t *testing.T) {
	t.Parallel()

	src := &fakePathsListSource{snaps: map[string]paths.PathSnapshot{}}
	resp, err := metrics.PathsList(context.Background(), nil, src)
	if err != nil {
		t.Fatalf("PathsList returned unexpected error: %v", err)
	}
	if len(resp.Paths) != 0 {
		t.Errorf("expected empty paths slice; got %d entries", len(resp.Paths))
	}
	if resp.Message != "no active paths" {
		t.Errorf("message: got %q; want %q", resp.Message, "no active paths")
	}
}

// ── AC-002: TestPathEntry_RTTValueSerialization ────────────────────────────

// TestPathEntry_RTTValueSerialization verifies RTTValue.MarshalJSON union semantics:
//
//	row (a) SampleCount=0   → "pending" string in JSON
//	row (b) SampleCount=9   → "pending" string in JSON
//	row (c) SampleCount=10  → float64 numeric in JSON
//	row (d) SampleCount=100 → float64 numeric in JSON
//
// AC-002; BC-2.06.003 PC-1, EC-003.
func TestPathEntry_RTTValueSerialization(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		rttValue    metrics.RTTValue
		wantPending bool    // true → JSON must be the string "pending"
		wantFloat   float64 // only checked when wantPending==false
	}{
		{
			name:        "row_a_count_0",
			rttValue:    metrics.RTTValue{Kind: metrics.PendingKind, SampleCount: 0},
			wantPending: true,
		},
		{
			name:        "row_b_count_9",
			rttValue:    metrics.RTTValue{Kind: metrics.PendingKind, Value: 42.5, SampleCount: 9},
			wantPending: true,
		},
		{
			name:        "row_c_count_10",
			rttValue:    metrics.RTTValue{Kind: metrics.FloatKind, Value: 22.0, SampleCount: 10},
			wantPending: false,
			wantFloat:   22.0,
		},
		{
			name:        "row_d_count_100",
			rttValue:    metrics.RTTValue{Kind: metrics.FloatKind, Value: 68.3, SampleCount: 100},
			wantPending: false,
			wantFloat:   68.3,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			v := tc.rttValue
			data, err := json.Marshal(v)
			if err != nil {
				t.Fatalf("MarshalJSON error: %v", err)
			}

			raw := string(data)
			if tc.wantPending {
				if raw != `"pending"` {
					t.Errorf("MarshalJSON(%+v): got %s; want \"pending\"", v, raw)
				}
				// L2 F-C2: discriminating oracle — value must not leak into JSON when pending.
				// Row_b has valueMs=42.5 with SampleCount=9; confirm 42.5 is suppressed.
				var f float64
				if jsonErr := json.Unmarshal(data, &f); jsonErr == nil {
					t.Errorf("MarshalJSON(%+v): produced float64 %v; pending must suppress ValueMs", v, f)
				}
				return
			}
			// Expect a numeric JSON value (float64).
			var got float64
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("expected float64 JSON value; got %s; unmarshal error: %v", raw, err)
			}
			if got != tc.wantFloat {
				t.Errorf("MarshalJSON(%+v): got float %v; want %v", v, got, tc.wantFloat)
			}
		})
	}
}

// TestRTTValue_RoundTrip verifies that Marshal→Unmarshal→Marshal is stable:
// marshalling an RTTValue, unmarshalling the JSON, then marshalling again
// produces the same JSON output. This guards against lossy decode.
//
// F-P1L1-007; BC-2.06.003 PC-1.
func TestRTTValue_RoundTrip(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		v           metrics.RTTValue
		wantPending bool
		wantFloat   float64
	}{
		{
			name:        "pending_count_0",
			v:           metrics.RTTValue{Kind: metrics.PendingKind, SampleCount: 0},
			wantPending: true,
		},
		{
			name: "pending_count_9_nonzero_value",
			// PendingKind: Value field is irrelevant but preserved.
			v:           metrics.RTTValue{Kind: metrics.PendingKind, Value: 42.5, SampleCount: 9},
			wantPending: true,
		},
		{
			name:        "float_count_10",
			v:           metrics.RTTValue{Kind: metrics.FloatKind, Value: 22.0, SampleCount: 10},
			wantPending: false,
			wantFloat:   22.0,
		},
		{
			name:        "float_count_100",
			v:           metrics.RTTValue{Kind: metrics.FloatKind, Value: 68.3, SampleCount: 100},
			wantPending: false,
			wantFloat:   68.3,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// First marshal.
			j1, err := json.Marshal(tc.v)
			if err != nil {
				t.Fatalf("first Marshal: %v", err)
			}

			// Unmarshal into a fresh RTTValue.
			var decoded metrics.RTTValue
			if err := json.Unmarshal(j1, &decoded); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}

			// Second marshal.
			j2, err := json.Marshal(decoded)
			if err != nil {
				t.Fatalf("second Marshal: %v", err)
			}

			// Round-trip stability: both JSON representations must be identical.
			if string(j1) != string(j2) {
				t.Errorf("round-trip unstable: first=%s second=%s", j1, j2)
			}

			// Pending cases: decoded must still be pending (SampleCount < 10).
			if tc.wantPending {
				if decoded.SampleCount >= 10 {
					t.Errorf("pending round-trip: decoded SampleCount=%d; want <10 (still pending)", decoded.SampleCount)
				}
				if string(j2) != `"pending"` {
					t.Errorf("pending round-trip: j2=%s; want \"pending\"", j2)
				}
			} else {
				// Float cases: decoded Kind must be FloatKind, Value must match.
				if decoded.Kind != metrics.FloatKind {
					t.Errorf("float round-trip: decoded Kind=%v; want FloatKind", decoded.Kind)
				}
				if decoded.Value != tc.wantFloat {
					t.Errorf("float round-trip: decoded Value=%v; want %v", decoded.Value, tc.wantFloat)
				}
				// And SampleCount must be ≥ 10 (preserved as valid).
				if decoded.SampleCount < 10 {
					t.Errorf("float round-trip: decoded SampleCount=%d; want ≥10 (valid float)", decoded.SampleCount)
				}
			}
		})
	}
}

// ── AC-003: TestPathEntry_StatusFromDegraded ──────────────────────────────

// TestPathEntry_StatusFromDegraded verifies that PathEntryFromSnapshot derives
// PathEntry.Status from PathSnapshot.Active and PathSnapshot.Degraded per
// BC-2.06.003 v1.10 PC-1 (status enum retracted to {active, degraded} per Ruling-4):
//
//	Active=false → "degraded" (liveness failure maps to "degraded" in Wave 6;
//	  "failed" is reserved for S-BL.PATH-FAILED-STATUS, Wave-7)
//	Active=true, Degraded=true → "degraded"
//	Active=true, Degraded=false → "active"
//
// AC-003; BC-2.06.001; BC-2.06.003 v1.10 PC-1; Ruling-4 (wave-6-tranche-a-scope-rulings.md).
func TestPathEntry_StatusFromDegraded(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		active     bool
		degraded   bool
		wantStatus string
	}{
		// Active=false maps to "degraded" in Wave 6; "failed" is reserved per Ruling-4.
		{name: "active_false_is_degraded", active: false, degraded: false, wantStatus: "degraded"},
		{name: "active_degraded_is_degraded", active: true, degraded: true, wantStatus: "degraded"},
		{name: "active_ok_is_active", active: true, degraded: false, wantStatus: "active"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			snap := paths.PathSnapshot{
				EWMARTTMs:   10.0,
				LossPct:     0.0,
				Active:      tc.active,
				Degraded:    tc.degraded,
				P99RTTMs:    10.0,
				SampleCount: 20,
			}
			entry := metrics.PathEntryFromSnapshot("pid", "host:9000", snap)
			if entry.Status != tc.wantStatus {
				t.Errorf("status: got %q; want %q (active=%v degraded=%v)",
					entry.Status, tc.wantStatus, tc.active, tc.degraded)
			}
		})
	}
}

// ── AC-004: TestDaemonRouterMetrics_HandlerRegistered ─────────────────────

// TestDaemonRouterMetrics_HandlerRegistered verifies that RouterMetrics returns
// a RouterMetricsResponse with the correct fields for a known SVTN.
//
// AC-004; BC-2.06.003 PC-2.
func TestDaemonRouterMetrics_HandlerRegistered(t *testing.T) {
	t.Parallel()

	want := metrics.RouterMetricsResponse{
		FrameCount:       1000,
		HMACFailCount:    5,
		DropCacheHits:    12,
		PathDistribution: map[string]uint64{"path-1": 600, "path-2": 400},
	}
	src := &fakeRouterMetricsSource{
		metrics: map[string]metrics.RouterMetricsResponse{"svtn-xyz": want},
	}

	// F-P1L1-001: use canonical wire key "svtn_id" (matches sbctl/router_metrics.go).
	args, _ := json.Marshal(map[string]string{"svtn_id": "svtn-xyz"})
	resp, err := metrics.RouterMetrics(context.Background(), json.RawMessage(args), src)
	if err != nil {
		t.Fatalf("RouterMetrics returned unexpected error: %v", err)
	}
	if resp.FrameCount != want.FrameCount {
		t.Errorf("frame_count: got %d; want %d", resp.FrameCount, want.FrameCount)
	}
	if resp.HMACFailCount != want.HMACFailCount {
		t.Errorf("hmac_fail_count: got %d; want %d", resp.HMACFailCount, want.HMACFailCount)
	}
	if resp.DropCacheHits != want.DropCacheHits {
		t.Errorf("drop_cache_hits: got %d; want %d", resp.DropCacheHits, want.DropCacheHits)
	}
}

// TestDaemonRouterMetrics_SVTNNotFound verifies E-RPC-011 on unknown SVTN.
//
// AC-004; BC-2.06.003 EC-004 (via E-RPC-011).
func TestDaemonRouterMetrics_SVTNNotFound(t *testing.T) {
	t.Parallel()

	src := &fakeRouterMetricsSource{metrics: map[string]metrics.RouterMetricsResponse{}}
	// F-P1L1-001: use canonical wire key "svtn_id".
	args, _ := json.Marshal(map[string]string{"svtn_id": "missing-svtn"})
	_, err := metrics.RouterMetrics(context.Background(), json.RawMessage(args), src)
	if err == nil {
		t.Fatal("expected error for unknown SVTN; got nil")
	}
	// F-P1L1-006: verify the error carries E-RPC-011 code (AC-004; BC-2.06.003 PC-2).
	rpcErr, ok := err.(*rpcError)
	if !ok {
		t.Fatalf("expected *rpcError; got %T: %v", err, err)
	}
	if rpcErr.code != "E-RPC-011" {
		t.Errorf("SVTN-not-found error code: got %q; want \"E-RPC-011\"", rpcErr.code)
	}
}

// ── AC-005: TestDaemonRouterStatus_HandlerRegistered ─────────────────────

// TestDaemonRouterStatus_HandlerRegistered verifies that RouterStatus returns
// a RouterStatusResponse with a Quality field and the same path structure as
// PathsListResponse.
//
// AC-005; BC-2.06.003 PC-3.
func TestDaemonRouterStatus_HandlerRegistered(t *testing.T) {
	t.Parallel()

	snap := paths.PathSnapshot{
		EWMARTTMs:   15.0,
		LossPct:     0.0,
		Active:      true,
		Degraded:    false,
		P99RTTMs:    15.0,
		SampleCount: 20, // ≥10: green
	}
	src := &fakePathsListSource{
		snaps: map[string]paths.PathSnapshot{"path-1": snap},
	}

	resp, err := metrics.RouterStatus(context.Background(), nil, src)
	if err != nil {
		t.Fatalf("RouterStatus returned unexpected error: %v", err)
	}
	if len(resp.Paths) != 1 {
		t.Errorf("expected 1 path; got %d", len(resp.Paths))
	}
	// L2 finding: sharpen from "any valid quality" to exact expected value.
	// Input: SampleCount=20 (≥10), P99RTTMs=15ms, loss=0.0 → Classify → green.
	// An implementation that returns any other value for this input is wrong.
	if resp.Quality != "green" {
		t.Errorf("quality: got %q; want \"green\" (SampleCount=20, p99=15ms, loss=0 → green band)", resp.Quality)
	}
}

// ── AC-005a: TestDaemonRouterStatus_QualityStatusIndependence ────────────

// TestDaemonRouterStatus_QualityStatusIndependence verifies S502-DEFER-3 /
// BC-2.06.003 v1.10 EC-007: quality and status are ORTHOGONAL fields.
// When a path has Active==false (liveness failure → status:"degraded" per Ruling-4)
// AND SampleCount<10 (p99 indeterminate → rtt_p99_ms:"pending"), the quality
// field MUST be "pending" — independent of the status field.
// Quality enum is {green,yellow,red,pending}; status enum is {active,degraded}.
// "failed" MUST NOT appear in either field in Wave 6 (Ruling-4; S-BL.PATH-FAILED-STATUS).
//
// F-P1L3-001: renamed from TestDaemonRouterStatus_FailedAndPendingPrecedence
// to reflect that quality/status orthogonality is the invariant under test.
//
// AC-005a; BC-2.06.003 v1.10 EC-007; S502-DEFER-3; Ruling-4.
func TestDaemonRouterStatus_QualityStatusIndependence(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		degraded    bool
		active      bool
		sampleCount uint64
		p99RTTMs    float64
		wantQuality string
		wantStatus  string
	}{
		{
			name:        "row_a_degraded_and_pending",
			degraded:    true,
			active:      false, // liveness failure → "degraded" per Ruling-4 (not "failed")
			sampleCount: 5,     // <10 → p99 pending
			p99RTTMs:    0,
			wantQuality: "pending",
			wantStatus:  "degraded",
		},
		{
			name:        "row_b_degraded_and_sufficient_samples",
			degraded:    true,
			active:      false, // liveness failure → "degraded" per Ruling-4
			sampleCount: 10,    // ≥10 → quality derived from p99
			p99RTTMs:    250.0, // 250ms → yellow or red depending on classify
			wantQuality: "",    // not "pending" — verified via != "pending" check
			wantStatus:  "degraded",
		},
		{
			name:        "row_c_healthy_pending",
			degraded:    false,
			active:      true,
			sampleCount: 5, // <10 → pending
			p99RTTMs:    0,
			wantQuality: "pending",
			wantStatus:  "active",
		},
		{
			name:        "row_d_green_sufficient_samples",
			degraded:    false,
			active:      true,
			sampleCount: 15,
			p99RTTMs:    15.0, // 15ms → green
			wantQuality: "green",
			wantStatus:  "active",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			snap := paths.PathSnapshot{
				EWMARTTMs:   tc.p99RTTMs, // reuse for simplicity
				LossPct:     0.0,
				Active:      tc.active,
				Degraded:    tc.degraded,
				P99RTTMs:    tc.p99RTTMs,
				SampleCount: tc.sampleCount,
			}
			src := &fakePathsListSource{
				snaps: map[string]paths.PathSnapshot{"path-x": snap},
			}

			resp, err := metrics.RouterStatus(context.Background(), nil, src)
			if err != nil {
				t.Fatalf("RouterStatus error: %v", err)
			}
			if len(resp.Paths) != 1 {
				t.Fatalf("expected 1 path; got %d", len(resp.Paths))
			}

			entry := resp.Paths[0]
			// Verify status field.
			if tc.wantStatus != "" && entry.Status != tc.wantStatus {
				t.Errorf("status: got %q; want %q", entry.Status, tc.wantStatus)
			}

			// Verify quality field.
			if tc.wantQuality == "pending" && resp.Quality != "pending" {
				t.Errorf("quality: got %q; want %q (pending-p99 must win over liveness state)", resp.Quality, "pending")
			}
			if tc.name == "row_b_degraded_and_sufficient_samples" {
				// When samples ≥ 10 and degraded, quality is NOT pending.
				if resp.Quality == "pending" {
					t.Errorf("quality: got %q; want non-pending (sufficient samples, liveness degraded)", resp.Quality)
				}
				// "failed" is not a valid quality value (Ruling-4; status enum is {active,degraded}).
				if resp.Quality == "failed" {
					t.Errorf("quality: got %q; \"failed\" is not a valid quality enum value", resp.Quality)
				}
				// L2 F-C3: discriminating oracle — p99RTTMs=250ms, loss=0%
				// classifies as yellow (100ms < 250ms ≤ 500ms). Confirm exactly.
				if resp.Quality != "yellow" {
					t.Errorf("quality: got %q; want \"yellow\" (p99=250ms, loss=0%% → yellow band)", resp.Quality)
				}
			}
			if tc.wantQuality == "green" && resp.Quality != "green" {
				t.Errorf("quality: got %q; want %q", resp.Quality, "green")
			}
		})
	}
}

// ── QualityFromEntry direct tests ──────────────────────────────────────────

// TestQualityFromEntry_PendingWhenSampleCountLow verifies that QualityFromEntry
// returns "pending" when RTTP99Ms.SampleCount < 10.
//
// BC-2.06.003 EC-006, EC-007.
func TestQualityFromEntry_PendingWhenSampleCountLow(t *testing.T) {
	t.Parallel()

	entry := metrics.PathEntry{
		PathID:     "p",
		RouterAddr: "h:9000",
		RTTMs:      50.0,
		RTTP99Ms:   metrics.RTTValue{Kind: metrics.PendingKind, SampleCount: 9},
		LossPct:    0.0,
		Status:     "active",
	}
	got := metrics.QualityFromEntry(entry)
	if got != "pending" {
		t.Errorf("QualityFromEntry with SampleCount=9: got %q; want \"pending\"", got)
	}
}

// TestQualityFromEntry_PendingWinsOverDegraded verifies EC-007 directly:
// when status indicates a non-healthy path AND SampleCount<10, quality MUST be "pending".
//
// Note: in Wave 6, PathEntryFromSnapshot never emits status="failed" (Ruling-4;
// BC-2.06.003 v1.10 PC-1). This test exercises QualityFromEntry robustness for
// any non-active status value passed to it directly.
//
// BC-2.06.003 v1.10 EC-007; S502-DEFER-3; Ruling-4.
func TestQualityFromEntry_PendingWinsOverDegraded(t *testing.T) {
	t.Parallel()

	entry := metrics.PathEntry{
		PathID:     "p",
		RouterAddr: "h:9000",
		RTTMs:      0.0,
		RTTP99Ms:   metrics.RTTValue{Kind: metrics.PendingKind, SampleCount: 5},
		LossPct:    0.0,
		Status:     "degraded",
	}
	got := metrics.QualityFromEntry(entry)
	if got != "pending" {
		t.Errorf("QualityFromEntry(status=degraded, SampleCount=5): got %q; want \"pending\" (EC-007 precedence)", got)
	}
	if got == "failed" {
		t.Errorf("quality %q is not a valid enum value; \"failed\" must never appear in the quality field", got)
	}
}

// TestQualityFromEntry_GreenWithSufficientSamples verifies the green path.
//
// BC-2.06.003 PC-3.
func TestQualityFromEntry_GreenWithSufficientSamples(t *testing.T) {
	t.Parallel()

	entry := metrics.PathEntry{
		PathID:     "p",
		RouterAddr: "h:9000",
		RTTMs:      15.0,
		RTTP99Ms:   metrics.RTTValue{Kind: metrics.FloatKind, Value: 15.0, SampleCount: 20},
		LossPct:    0.0,
		Status:     "active",
	}
	got := metrics.QualityFromEntry(entry)
	if got != "green" {
		t.Errorf("QualityFromEntry(p99=15ms, loss=0): got %q; want \"green\"", got)
	}
}

// TestQualityFromEntry_NeverEmitsFailed verifies that "failed" never appears
// as a quality value under any combination of inputs.
//
// BC-2.06.003 PC-3; S502-DEFER-3 invariant.
func TestQualityFromEntry_NeverEmitsFailed(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		sampleCount uint64
		p99Ms       float64
		status      string
	}{
		{"active_low_samples", 5, 0, "active"},
		{"active_high_samples_green", 20, 15.0, "active"},
		{"degraded_low_samples", 5, 0, "degraded"},
		{"failed_low_samples", 5, 0, "failed"},
		{"failed_high_samples", 20, 600.0, "failed"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Derive Kind from sampleCount per the same logic as PathEntryFromSnapshot.
			kind := metrics.PendingKind
			if tc.sampleCount >= 10 {
				kind = metrics.FloatKind
			}
			entry := metrics.PathEntry{
				RTTP99Ms: metrics.RTTValue{Kind: kind, Value: tc.p99Ms, SampleCount: tc.sampleCount},
				Status:   tc.status,
			}
			got := metrics.QualityFromEntry(entry)
			if got == "failed" {
				t.Errorf("QualityFromEntry returned %q; \"failed\" is not a valid quality enum value", got)
			}
		})
	}
}

// ── Pass-2 L1/L2 additional tests ─────────────────────────────────────────────

// TestPathsList_DiscriminatingStatusOracle verifies that when Degraded=false and
// SampleCount≥10 the handler emits status="active" — not any other value.
// Prevents a dead-code path where the implementation returns hardcoded "active"
// regardless of input (F-P2L1-005 discriminating oracle).
//
// BC-2.06.003 PC-1; AC-003.
func TestPathsList_DiscriminatingStatusOracle(t *testing.T) {
	t.Parallel()

	// Row 1: Degraded=false, SampleCount=10 → status must be exactly "active".
	snap1 := paths.PathSnapshot{
		EWMARTTMs:   20.0,
		LossPct:     0.0,
		Active:      true,
		Degraded:    false,
		P99RTTMs:    20.0,
		SampleCount: 10,
	}
	entry1 := metrics.PathEntryFromSnapshot("p1", "", snap1)
	if entry1.Status != "active" {
		t.Errorf("Degraded=false, SampleCount=10: status=%q; want \"active\"", entry1.Status)
	}

	// Row 2: Degraded=true, SampleCount=10 → status must be "degraded" (not "active").
	// If the implementation hardcodes "active", this row will catch it.
	snap2 := paths.PathSnapshot{
		EWMARTTMs:   250.0,
		LossPct:     0.0,
		Active:      true,
		Degraded:    true,
		P99RTTMs:    250.0,
		SampleCount: 10,
	}
	entry2 := metrics.PathEntryFromSnapshot("p2", "", snap2)
	if entry2.Status == "active" {
		t.Errorf("Degraded=true, SampleCount=10: status=%q; must NOT be \"active\" when path is degraded", entry2.Status)
	}
	if entry2.Status != "degraded" {
		t.Errorf("Degraded=true, SampleCount=10: status=%q; want \"degraded\"", entry2.Status)
	}
}

// TestRouterMetrics_MalformedArgsDecode verifies that RouterMetrics returns a
// decode error carrying E-RPC-002 (not a panic) when given garbage bytes as args.
//
// F-P2L2 malformed-args path; BC-2.06.003 v1.10 PC-2.
func TestRouterMetrics_MalformedArgsDecode(t *testing.T) {
	t.Parallel()

	src := &fakeRouterMetricsSource{metrics: map[string]metrics.RouterMetricsResponse{}}
	garbage := json.RawMessage([]byte{0xFF, 0xFE, 0x01, 0x02})
	_, err := metrics.RouterMetrics(context.Background(), garbage, src)
	if err == nil {
		t.Fatal("RouterMetrics with garbage args: expected error; got nil")
	}
	// Sharpen oracle: error MUST carry E-RPC-002 (malformed args error code per BC-2.06.003 PC-2).
	// An implementation that panics or returns a non-E-RPC-002 error fails this oracle.
	if !containsAny(err.Error(), "E-RPC-002") {
		t.Errorf("RouterMetrics garbage args: error %q; want E-RPC-002 prefix (BC-2.06.003 PC-2 malformed-args)", err.Error())
	}
}

// containsAny reports whether s contains any of the substrings.
func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(sub) > 0 {
			// manual contains check to avoid importing strings
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

// TestVP047_FieldSwapOracle verifies that path_id and router_addr are not
// swapped in the PathEntry serialization. Seeds two paths with
// non-overlapping character sets so a field cross-contamination would be
// detectable.
//
// VP-047 field-swap oracle (F-P2L2); AC-006; BC-2.06.003 PC-1.
func TestVP047_FieldSwapOracle(t *testing.T) {
	t.Parallel()

	// path_id uses only digits; router_addr uses only alpha chars.
	// If the fields were swapped, the digit-only string would appear in
	// router_addr and the alpha-only string in path_id.
	pathID := "000111222"
	routerAddr := "abcdefghi"

	snap := paths.PathSnapshot{
		EWMARTTMs:   10.0,
		LossPct:     0.0,
		Active:      true,
		Degraded:    false,
		P99RTTMs:    10.0,
		SampleCount: 10,
	}
	entry := metrics.PathEntryFromSnapshot(pathID, routerAddr, snap)

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal PathEntry: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}

	// path_id must contain only digits.
	var gotPathID string
	if err := json.Unmarshal(raw["path_id"], &gotPathID); err != nil {
		t.Fatalf("unmarshal path_id: %v", err)
	}
	if gotPathID != pathID {
		t.Errorf("path_id: got %q; want %q (possible field swap)", gotPathID, pathID)
	}

	// router_addr must contain only alpha chars.
	var gotRouterAddr string
	if err := json.Unmarshal(raw["router_addr"], &gotRouterAddr); err != nil {
		t.Fatalf("unmarshal router_addr: %v", err)
	}
	if gotRouterAddr != routerAddr {
		t.Errorf("router_addr: got %q; want %q (possible field swap)", gotRouterAddr, routerAddr)
	}
}

// TestEC006_DegradedAndPendingRow verifies EC-006:
// Degraded=true AND SampleCount<10 → status="degraded" AND rtt_p99_ms="pending"
// AND quality="pending". This is the composite row test.
//
// BC-2.06.003 EC-006; AC-005a.
func TestEC006_DegradedAndPendingRow(t *testing.T) {
	t.Parallel()

	snap := paths.PathSnapshot{
		EWMARTTMs:   300.0,
		LossPct:     0.0,
		Active:      true, // liveness ok
		Degraded:    true, // EWMA RTT > 200ms threshold
		P99RTTMs:    0.0,
		SampleCount: 5, // <10 → p99 pending
	}
	src := &fakePathsListSource{
		snaps: map[string]paths.PathSnapshot{"path-ec006": snap},
	}

	resp, err := metrics.RouterStatus(context.Background(), nil, src)
	if err != nil {
		t.Fatalf("RouterStatus error: %v", err)
	}
	if len(resp.Paths) != 1 {
		t.Fatalf("expected 1 path; got %d", len(resp.Paths))
	}

	entry := resp.Paths[0]

	// EC-006: status must be "degraded".
	if entry.Status != "degraded" {
		t.Errorf("EC-006: status=%q; want \"degraded\" (Degraded=true)", entry.Status)
	}

	// EC-006: rtt_p99_ms must be "pending" (SampleCount<10).
	p99JSON, err := json.Marshal(entry.RTTP99Ms)
	if err != nil {
		t.Fatalf("marshal rtt_p99_ms: %v", err)
	}
	if string(p99JSON) != `"pending"` {
		t.Errorf("EC-006: rtt_p99_ms=%s; want \"pending\" (SampleCount<10)", p99JSON)
	}

	// EC-006: quality must be "pending" (SampleCount<10 takes precedence, EC-007).
	if resp.Quality != "pending" {
		t.Errorf("EC-006: quality=%q; want \"pending\" (EC-007 pending-p99 precedence)", resp.Quality)
	}
}

// TestRouterMetrics_MissingRequiredSVTN verifies Fix 6: router.metrics returns
// an E-RPC-* error when svtn_id is absent or empty.
//
// Fix 6; BC-2.06.003 PC-2.
func TestRouterMetrics_MissingRequiredSVTN(t *testing.T) {
	t.Parallel()

	src := &fakeRouterMetricsSource{metrics: map[string]metrics.RouterMetricsResponse{}}

	cases := []struct {
		name string
		args json.RawMessage
	}{
		{name: "nil_args", args: nil},
		{name: "empty_object", args: json.RawMessage(`{}`)},
		{name: "empty_svtn_id", args: json.RawMessage(`{"svtn_id":""}`)},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := metrics.RouterMetrics(context.Background(), tc.args, src)
			if err == nil {
				t.Fatalf("router.metrics with %s: expected error for missing svtn_id; got nil", tc.name)
			}
			if !containsAny(err.Error(), "E-RPC", "svtn_id", "required") {
				t.Errorf("router.metrics %s: error %q; want E-RPC-* or svtn_id mention", tc.name, err.Error())
			}
		})
	}
}

// ── EC-002: All paths pending ─────────────────────────────────────────────────

// TestEC002_AllPathsPending verifies EC-002: when all paths have SampleCount<10,
// every PathEntry.rtt_p99_ms value MUST be the string "pending" in JSON.
//
// BC-2.06.003 EC-002; AC-002.
func TestEC002_AllPathsPending(t *testing.T) {
	t.Parallel()

	// Three paths, all with SampleCount<10 → all rtt_p99_ms must be "pending".
	snaps := map[string]paths.PathSnapshot{
		"path-p1": {EWMARTTMs: 10.0, LossPct: 0.0, Active: true, Degraded: false, P99RTTMs: 0.0, SampleCount: 0},
		"path-p2": {EWMARTTMs: 20.0, LossPct: 0.0, Active: true, Degraded: false, P99RTTMs: 0.0, SampleCount: 5},
		"path-p3": {EWMARTTMs: 15.0, LossPct: 0.1, Active: true, Degraded: false, P99RTTMs: 0.0, SampleCount: 9},
	}
	src := &fakePathsListSource{snaps: snaps}

	resp, err := metrics.PathsList(context.Background(), nil, src)
	if err != nil {
		t.Fatalf("PathsList error: %v", err)
	}
	if len(resp.Paths) != 3 {
		t.Fatalf("expected 3 paths; got %d", len(resp.Paths))
	}

	for _, entry := range resp.Paths {
		p99JSON, err := json.Marshal(entry.RTTP99Ms)
		if err != nil {
			t.Fatalf("marshal rtt_p99_ms for %s: %v", entry.PathID, err)
		}
		// EC-002: EVERY entry with SampleCount<10 must emit "pending".
		if string(p99JSON) != `"pending"` {
			t.Errorf("EC-002: path %s rtt_p99_ms=%s; want \"pending\" (SampleCount<10)", entry.PathID, p99JSON)
		}
	}
}

// ── Status enum closure ───────────────────────────────────────────────────────

// TestPathEntry_StatusEnumClosed verifies that PathEntryFromSnapshot never emits
// a status value outside {active, degraded} for any combination of inputs.
// "failed" MUST NOT appear per BC-2.06.003 v1.10 PC-1 Ruling-4.
//
// BC-2.06.003 v1.10 PC-1; Ruling-4 (wave-6-tranche-a-scope-rulings.md).
func TestPathEntry_StatusEnumClosed(t *testing.T) {
	t.Parallel()

	validStatuses := map[string]bool{"active": true, "degraded": true}

	cases := []struct {
		name     string
		active   bool
		degraded bool
	}{
		{"active_true_degraded_false", true, false},
		{"active_true_degraded_true", true, true},
		{"active_false_degraded_false", false, false},
		{"active_false_degraded_true", false, true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			snap := paths.PathSnapshot{
				EWMARTTMs:   10.0,
				LossPct:     0.0,
				Active:      tc.active,
				Degraded:    tc.degraded,
				P99RTTMs:    10.0,
				SampleCount: 20,
			}
			entry := metrics.PathEntryFromSnapshot("p", "", snap)
			if !validStatuses[entry.Status] {
				t.Errorf("status enum violation: got %q; valid values are {active, degraded} only (BC-2.06.003 v1.10 PC-1, Ruling-4)", entry.Status)
			}
		})
	}
}

// TestRTTValue_JSONShapeExact verifies the exact JSON wire shape of RTTValue.
// Pending → `"pending"` (JSON string); float → bare float64 (no wrapper object).
// This guards against encoding drift where the shape changes but .Value() still works.
//
// Pass-3 L2 finding: RTTValue round-trip tightening.
// BC-2.06.003 v1.10 PC-1, EC-003.
func TestRTTValue_JSONShapeExact(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		v         metrics.RTTValue
		wantShape string // exact JSON bytes
	}{
		{
			name:      "pending_kind_zero_value",
			v:         metrics.RTTValue{Kind: metrics.PendingKind, Value: 0, SampleCount: 0},
			wantShape: `"pending"`,
		},
		{
			name:      "pending_kind_nonzero_value_suppressed",
			v:         metrics.RTTValue{Kind: metrics.PendingKind, Value: 99.9, SampleCount: 9},
			wantShape: `"pending"`, // value MUST be suppressed when Kind==PendingKind
		},
		{
			name:      "float_kind_integer_ms",
			v:         metrics.RTTValue{Kind: metrics.FloatKind, Value: 42, SampleCount: 10},
			wantShape: `42`, // JSON number, no quotes, no object wrapper
		},
		{
			name:      "float_kind_fractional_ms",
			v:         metrics.RTTValue{Kind: metrics.FloatKind, Value: 68.3, SampleCount: 100},
			wantShape: `68.3`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tc.v)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if string(data) != tc.wantShape {
				t.Errorf("JSON shape: got %s; want %s (exact wire shape required by BC-2.06.003 v1.10 PC-1)", data, tc.wantShape)
			}
		})
	}
}
