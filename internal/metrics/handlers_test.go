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
		sampleCount uint64
		valueMs     float64
		wantPending bool    // true → JSON must be the string "pending"
		wantFloat   float64 // only checked when wantPending==false
	}{
		{name: "row_a_count_0", sampleCount: 0, valueMs: 0, wantPending: true},
		{name: "row_b_count_9", sampleCount: 9, valueMs: 42.5, wantPending: true},
		{name: "row_c_count_10", sampleCount: 10, valueMs: 22.0, wantPending: false, wantFloat: 22.0},
		{name: "row_d_count_100", sampleCount: 100, valueMs: 68.3, wantPending: false, wantFloat: 68.3},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			v := metrics.RTTValue{ValueMs: tc.valueMs, SampleCount: tc.sampleCount}
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
			v:           metrics.RTTValue{ValueMs: 0, SampleCount: 0},
			wantPending: true,
		},
		{
			name:        "pending_count_9_nonzero_value",
			v:           metrics.RTTValue{ValueMs: 42.5, SampleCount: 9},
			wantPending: true,
		},
		{
			name:        "float_count_10",
			v:           metrics.RTTValue{ValueMs: 22.0, SampleCount: 10},
			wantPending: false,
			wantFloat:   22.0,
		},
		{
			name:        "float_count_100",
			v:           metrics.RTTValue{ValueMs: 68.3, SampleCount: 100},
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
				// Float cases: decoded ValueMs must match the input.
				if decoded.ValueMs != tc.wantFloat {
					t.Errorf("float round-trip: decoded ValueMs=%v; want %v", decoded.ValueMs, tc.wantFloat)
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
// PathEntry.Status from PathSnapshot.Active and PathSnapshot.Degraded:
//
//	Active=false → "failed"
//	Active=true, Degraded=true → "degraded"
//	Active=true, Degraded=false → "active"
//
// AC-003; BC-2.06.001; BC-2.06.003 PC-1.
func TestPathEntry_StatusFromDegraded(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		active     bool
		degraded   bool
		wantStatus string
	}{
		{name: "active_false_is_failed", active: false, degraded: false, wantStatus: "failed"},
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
	validQualities := map[string]bool{"green": true, "yellow": true, "red": true, "pending": true}
	if !validQualities[resp.Quality] {
		t.Errorf("quality %q is not a valid enum value (green|yellow|red|pending)", resp.Quality)
	}
	// Quality field must be present and non-empty (structural requirement).
	if resp.Quality == "" {
		t.Error("quality field must not be empty")
	}
}

// ── AC-005a: TestDaemonRouterStatus_QualityStatusIndependence ────────────

// TestDaemonRouterStatus_QualityStatusIndependence verifies S502-DEFER-3 /
// BC-2.06.003 v1.8 EC-007: quality and status are ORTHOGONAL fields.
// When a path has Active==false (liveness failure → status:"failed") AND
// SampleCount<10 (p99 indeterminate → rtt_p99_ms:"pending"), the quality
// field MUST be "pending" — independent of the status field.
// Quality enum is {green,yellow,red,pending}; "failed" is not a valid quality value.
//
// F-P1L3-001: renamed from TestDaemonRouterStatus_FailedAndPendingPrecedence
// to reflect that quality/status orthogonality is the invariant under test.
//
// AC-005a; BC-2.06.003 v1.8 EC-007; S502-DEFER-3.
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
			active:      false, // ≥3 missed keepalives → "failed"
			sampleCount: 5,     // <10 → p99 pending
			p99RTTMs:    0,
			wantQuality: "pending",
			wantStatus:  "failed",
		},
		{
			name:        "row_b_degraded_and_sufficient_samples",
			degraded:    true,
			active:      false, // failed
			sampleCount: 10,    // ≥10 → quality derived from p99
			p99RTTMs:    250.0, // 250ms → yellow or red depending on classify
			wantQuality: "",    // not "pending" — verified via != "pending" check
			wantStatus:  "failed",
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
				// When samples ≥ 10 and degraded+failed, quality is NOT pending.
				if resp.Quality == "pending" {
					t.Errorf("quality: got %q; want non-pending (sufficient samples, liveness failed)", resp.Quality)
				}
				// "failed" is not a valid quality value.
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
		RTTP99Ms:   metrics.RTTValue{ValueMs: 0, SampleCount: 9},
		LossPct:    0.0,
		Status:     "active",
	}
	got := metrics.QualityFromEntry(entry)
	if got != "pending" {
		t.Errorf("QualityFromEntry with SampleCount=9: got %q; want \"pending\"", got)
	}
}

// TestQualityFromEntry_PendingWinsOverFailed verifies EC-007 directly:
// when Status=="failed" AND SampleCount<10, quality MUST be "pending".
//
// BC-2.06.003 v1.8 EC-007; S502-DEFER-3.
func TestQualityFromEntry_PendingWinsOverFailed(t *testing.T) {
	t.Parallel()

	entry := metrics.PathEntry{
		PathID:     "p",
		RouterAddr: "h:9000",
		RTTMs:      0.0,
		RTTP99Ms:   metrics.RTTValue{ValueMs: 0, SampleCount: 5},
		LossPct:    0.0,
		Status:     "failed",
	}
	got := metrics.QualityFromEntry(entry)
	if got != "pending" {
		t.Errorf("QualityFromEntry(status=failed, SampleCount=5): got %q; want \"pending\" (EC-007 precedence)", got)
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
		RTTP99Ms:   metrics.RTTValue{ValueMs: 15.0, SampleCount: 20},
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
			entry := metrics.PathEntry{
				RTTP99Ms: metrics.RTTValue{ValueMs: tc.p99Ms, SampleCount: tc.sampleCount},
				Status:   tc.status,
			}
			got := metrics.QualityFromEntry(entry)
			if got == "failed" {
				t.Errorf("QualityFromEntry returned %q; \"failed\" is not a valid quality enum value", got)
			}
		})
	}
}
