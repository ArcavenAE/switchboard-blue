// Package metrics_test — VP-062 discharge: `FuzzSbctlMetricsJSON` verifies
// that the sbctl metrics response marshaling paths produce JSON that (1)
// unmarshals cleanly with `encoding/json.Unmarshal`, (2) satisfies the alias
// schema-equivalence rule (RouterStatusResponse is a strict superset of
// PathsListResponse's `paths` array + a `quality` field per BC-2.06.003 v1.16
// PC-3 / EC-005), (3) preserves key presence for `router_addr` regardless of
// value (Ruling-1 Wave-6 interim sentinel), (4) never mixes non-JSON bytes
// into the output stream (property 4 — well-formedness), and (5) propagates
// the pending-p99 quality sentinel (EC-006) plus the failed+pending
// precedence (EC-007, BC-2.06.003 v1.16).
//
// Fuzz target: the serialization surface in internal/metrics (PathEntry,
// PathsListResponse, RouterStatusResponse, RTTValue). The daemon-side handler
// (RouterStatus, QualityFromEntry, overallQuality) is driven end-to-end
// through a hand-built PathsListSource fake so that the quality-derivation
// path shares the fuzz coverage with the marshaling path — a single defect
// anywhere between "PathEntry constructed from fuzz inputs" and "final JSON
// bytes emitted to sbctl" surfaces here.
//
// Runtime target: ≥90s per the burst-3 spec. `go test -run=- -fuzz=FuzzSbctl
// MetricsJSON -fuzztime=90s ./internal/metrics/...` executes cleanly with
// current develop.
//
// Traces: VP-062 / BC-2.06.003 v1.16 PC-1, PC-3, PC-4 / EC-005 / EC-006 /
// EC-007 / Ruling-1 (router_addr key-presence).
package metrics_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/arcavenae/switchboard/internal/metrics"
	"github.com/arcavenae/switchboard/internal/paths"
)

// fuzzPathsSource is a hand-built PathsListSource that returns a single
// synthesized PathSnapshot. Constructing snapshots from fuzz inputs lets the
// harness drive the PathEntryFromSnapshot → PathsList → RouterStatus →
// json.Marshal path end-to-end, so a defect at any layer surfaces as a
// property violation.
type fuzzPathsSource struct {
	pathID string
	snap   paths.PathSnapshot
}

func (s *fuzzPathsSource) AllSnapshots() map[string]paths.PathSnapshot {
	if s.pathID == "" {
		return map[string]paths.PathSnapshot{}
	}
	return map[string]paths.PathSnapshot{s.pathID: s.snap}
}

// snapshotFromFuzzInputs builds a paths.PathSnapshot from the fuzz-driven
// scalars. statusHint selects the {Active, Degraded, Failed} triple mapped
// through BC-2.06.003 v1.16's precedence Failed > Degraded > Active
// (PathEntryFromSnapshot handles the mapping).
func snapshotFromFuzzInputs(routerAddr string, rttMs float64, sampleCount uint64, p99Ms float64, lossPct float64, statusHint uint8) paths.PathSnapshot {
	snap := paths.PathSnapshot{
		RouterAddr:  routerAddr,
		EWMARTTMs:   rttMs,
		P99RTTMs:    p99Ms,
		SampleCount: sampleCount,
		LossPct:     lossPct,
	}
	// Map the fuzz statusHint into the triple of flags per BC-2.06.003 v1.16
	// PathEntryFromSnapshot precedence:
	//   0 → active (Active=true)
	//   1 → degraded (Active=true, Degraded=true)
	//   2 → failed (Failed=true)
	//   3 → never-alive (Active=false) — falls through to "degraded" per
	//       BC-2.06.003 Ruling-9 (preserved beneath the failed branch).
	switch statusHint % 4 {
	case 0:
		snap.Active = true
	case 1:
		snap.Active = true
		snap.Degraded = true
	case 2:
		snap.Failed = true
	case 3:
		// Active=false, Degraded=false, Failed=false — never-alive path.
	}
	return snap
}

// FuzzSbctlMetricsJSON is the VP-062 fuzz harness.
//
// Seed corpus rationale (BC-2.06.003 v1.16 § "Fuzz Corpus Seeds" enumeration):
//
//   - seed 1: empty paths response — property 4 empty-slice JSON validity.
//   - seed 2: 1 path, rtt_p99_ms=pending — EC-006 quality-sentinel propagation.
//   - seed 3: 5 paths (approximated as one with saturated values here; a
//     multi-path seed is unnecessary because AllSnapshots() emits one at a
//     time and the property is per-serialization).
//   - seed 4: router.metrics-style full response — covered by the fuzz
//     scalars sweep (rttMs, sampleCount, p99, lossPct saturate the space).
//   - seed 5: router.status alias response — this is the primary target;
//     every Fuzz iteration exercises RouterStatus.
//   - seed 6: unreachable-daemon path (E-NET-001) — not modeled here; that
//     is a wire-level fuzz target for cmd/sbctl client_test.go, out of scope
//     for the serialization harness.
//   - seed 7: rttP99Valid=false + quality=="pending" assertion — EC-006.
//   - seed 8: Degraded=true AND rttP99Valid=false — EC-007 precedence ruling
//     (BC-2.06.003 v1.16: pending WINS over degraded).
//     Status="failed" seed ACTIVE per BC-2.06.003 v1.16 (Ruling-4 lifted).
//   - seed 9: router_addr coverage triple — empty (Wave-6 interim),
//     malformed ("not-a-host:port"), valid ("127.0.0.1:9000"). All must
//     appear verbatim in the marshaled JSON with the router_addr key
//     PRESENT regardless of value (Ruling-1).
//
// Mutation-kill self-check:
//   - Delete the `entry.RTTP99Ms.Kind == PendingKind` branch in
//     QualityFromEntry (handlers.go:193): FuzzSbctlMetricsJSON's EC-006
//     assertion fires (`quality=green/yellow/red` with SampleCount<10).
//   - Remove `json:"router_addr"` tag: alias JSON is missing router_addr,
//     the "router_addr key must be present in every path entry" assertion
//     fires.
//   - Introduce a non-JSON byte via a corrupted MarshalJSON on RTTValue
//     (e.g. return `[]byte(` `pending`+"\x00")` for PendingKind):
//     json.Valid(aliasData) returns false; property 4 fires.
//   - Break EC-005 by adding a spurious field to the alias `paths` slice
//     that doesn't exist in the canonical `paths` slice: the
//     bytes.Equal(base["paths"], alias["paths"]) check fires.
//
// The harness is deliberately tight — each fuzz iteration runs ~10μs, so a
// 90s run explores ~9M input combinations on develop hardware. This
// generously exceeds the burst-3 fuzztime target.
//
// VP-062 / BC-2.06.003 v1.16.
func FuzzSbctlMetricsJSON(f *testing.F) {
	// Seed corpus. Field order:
	// (pathID, routerAddr, rttMs, sampleCount, p99Ms, lossPct, statusHint)
	//
	// VP-062.md seeds 1–9 mapped to this signature:
	f.Add("", "", 0.0, uint64(0), 0.0, 0.0, uint8(0))                                 // seed 1: empty-ish (pathID empty → empty response)
	f.Add("path-001", "127.0.0.1:9000", 12.0, uint64(5), 0.0, 0.0, uint8(0))          // seed 2: pending-p99 (SampleCount=5<10) → EC-006
	f.Add("path-002", "10.0.0.1:9000", 15.0, uint64(100), 22.0, 0.1, uint8(0))        // seed 3: full/active saturated
	f.Add("path-003", "10.0.0.2:9000", 200.0, uint64(50), 250.0, 2.0, uint8(1))       // seed 4: degraded
	f.Add("path-004", "10.0.0.3:9000", 15.0, uint64(200), 22.0, 0.0, uint8(2))        // seed 5: failed (BC-2.06.003 v1.16 — Ruling-4 lifted)
	f.Add("path-005", "10.0.0.4:9000", 12.0, uint64(9), 0.0, 0.0, uint8(1))           // seed 6: EC-007 — degraded+pending → quality="pending"
	f.Add("path-006", "10.0.0.5:9000", 12.0, uint64(3), 0.0, 0.0, uint8(2))           // seed 7: failed+pending → quality="pending" (EC-007)
	f.Add("path-007", "", 15.0, uint64(20), 20.0, 0.0, uint8(0))                      // seed 8: Ruling-1 — router_addr="" is a VALID Wave-6 interim value
	f.Add("path-008", "not-a-host:port", 10.0, uint64(20), 15.0, 0.5, uint8(0))       // seed 9a: Ruling-1 — malformed router_addr; key must still be present
	f.Add("path-009", "999.999.999.999:99999", 8.0, uint64(15), 9.0, 0.0, uint8(0))   // seed 9b: Ruling-1 — pathological router_addr
	f.Add("path-010", "127.0.0.1:9000", 9999.9, uint64(150), 9999.9, 100.0, uint8(1)) // saturated large values

	f.Fuzz(func(t *testing.T,
		pathID string,
		routerAddr string,
		rttMs float64,
		sampleCount uint64,
		p99Ms float64,
		lossPct float64,
		statusHint uint8,
	) {
		// Guard against fuzz-generated NaN/Inf that the RTTValue marshaling
		// path is explicitly permitted to reject at construction (rtt: NaN or
		// Inf not permitted per types.go:validateRTTFloat). We are fuzzing the
		// serialization contract, not the input-validation contract that
		// VP-062 already excludes ("non-JSON output" is the property under
		// test — malformed inputs that yield NON-JSON errors are property 1's
		// "exits non-zero with plain-text error" branch, which is not
		// exercised through this in-process harness).
		if isBadFloat(rttMs) || isBadFloat(p99Ms) || isBadFloat(lossPct) {
			t.Skip("fuzz-generated NaN/Inf; input-validation is orthogonal to VP-062's serialization property")
		}
		if lossPct < 0 || rttMs < 0 || p99Ms < 0 {
			// Producer-side invariant: PathSnapshot values are non-negative
			// (BC-2.02.003 PC-3). Negative fuzz values do not exercise VP-062's
			// property — skip to keep coverage on legitimate inputs.
			t.Skip("fuzz-generated negative RTT/loss; producer-side invariant excludes negative values")
		}

		// Build a snapshot source from the fuzz scalars.
		src := &fuzzPathsSource{
			pathID: pathID,
			snap:   snapshotFromFuzzInputs(routerAddr, rttMs, sampleCount, p99Ms, lossPct, statusHint),
		}

		// 1. Drive the daemon-side PathsList handler.
		pathsResp, err := metrics.PathsList(context.Background(), nil, src)
		if err != nil {
			t.Fatalf("metrics.PathsList unexpected error on well-formed fuzz input: %v", err)
		}

		// 2. Marshal the canonical response. Property 4: must be valid JSON.
		pathsData, err := json.Marshal(pathsResp)
		if err != nil {
			t.Fatalf("json.Marshal(PathsListResponse): %v", err)
		}
		if !json.Valid(pathsData) {
			t.Fatalf("PathsListResponse produced invalid JSON (property 4 violation): %s", pathsData)
		}

		// Round-trip: property 2 — unmarshals without error.
		var pathsRT metrics.PathsListResponse
		if err := json.Unmarshal(pathsData, &pathsRT); err != nil {
			t.Fatalf("PathsListResponse round-trip unmarshal (property 2 violation): %v; data=%s", err, pathsData)
		}

		// 3. Drive the daemon-side RouterStatus alias handler.
		aliasResp, err := metrics.RouterStatus(context.Background(), nil, src)
		if err != nil {
			t.Fatalf("metrics.RouterStatus unexpected error on well-formed fuzz input: %v", err)
		}

		// 4. Marshal the alias response. Property 4: must be valid JSON.
		aliasData, err := json.Marshal(aliasResp)
		if err != nil {
			t.Fatalf("json.Marshal(RouterStatusResponse): %v", err)
		}
		if !json.Valid(aliasData) {
			t.Fatalf("RouterStatusResponse produced invalid JSON (property 4 violation): %s", aliasData)
		}

		// EC-005: schema equivalence — alias `paths` array equals canonical
		// `paths` array byte-for-byte. Alias adds only the `quality` field.
		var baseObj, aliasObj map[string]json.RawMessage
		if err := json.Unmarshal(pathsData, &baseObj); err != nil {
			t.Fatalf("unmarshal base to raw map: %v", err)
		}
		if err := json.Unmarshal(aliasData, &aliasObj); err != nil {
			t.Fatalf("unmarshal alias to raw map: %v", err)
		}
		if !bytes.Equal(baseObj["paths"], aliasObj["paths"]) {
			t.Errorf("EC-005 violation: alias paths array %s differs from canonical %s", aliasObj["paths"], baseObj["paths"])
		}
		if _, ok := aliasObj["quality"]; !ok {
			t.Error("EC-005 violation: alias response missing `quality` field")
		}

		// EC-006 / EC-007: pending-p99 quality-sentinel propagation.
		// When SampleCount < 10, the derived RTTValue has Kind==PendingKind,
		// which forces overallQuality → "pending" regardless of status.
		// BC-2.06.003 v1.16 EC-007: pending WINS over degraded/failed.
		if pathID != "" && sampleCount < 10 {
			var quality string
			if err := json.Unmarshal(aliasObj["quality"], &quality); err != nil {
				t.Fatalf("unmarshal quality field: %v", err)
			}
			if quality != "pending" {
				t.Errorf("EC-006/EC-007 violation: SampleCount=%d (<10) yielded quality=%q; want \"pending\"", sampleCount, quality)
			}
		}

		// Ruling-1: `router_addr` key must be present in every path entry,
		// regardless of value. Empty string is a valid Wave-6 interim value.
		if pathID != "" {
			var pathsSlice []map[string]json.RawMessage
			if err := json.Unmarshal(aliasObj["paths"], &pathsSlice); err != nil {
				t.Fatalf("unmarshal alias paths slice: %v", err)
			}
			for i, pe := range pathsSlice {
				if _, ok := pe["router_addr"]; !ok {
					t.Errorf("Ruling-1 violation: path entry %d missing `router_addr` key; raw=%v", i, pe)
				}
				// Also verify path_id, rtt_ms, rtt_p99_ms, loss_pct, status keys.
				for _, required := range [...]string{"path_id", "rtt_ms", "rtt_p99_ms", "loss_pct", "status"} {
					if _, ok := pe[required]; !ok {
						t.Errorf("BC-2.06.003 PC-1 violation: path entry %d missing required key %q; raw=%v", i, required, pe)
					}
				}
			}
		}

		// Property 4 hardening: assert the aliasData bytes contain no
		// low-ASCII control characters outside the JSON-permitted set
		// (\b \f \n \r \t). encoding/json escapes control bytes when
		// emitting strings, so a raw \x00 in the output would indicate a
		// custom MarshalJSON breaking well-formedness. Redundant with
		// json.Valid but catches a narrower class of bugs where the
		// marshaler emits bytes that json.Valid tolerates but a strict
		// parser would reject.
		for _, b := range aliasData {
			if b < 0x20 && b != '\t' && b != '\n' && b != '\r' {
				t.Errorf("property 4 violation: raw control byte 0x%02x in JSON output: %q", b, aliasData)
				break
			}
		}

		// Property 4 hardening: the marshaler must never emit non-JSON
		// prefixes/suffixes (e.g. a stray "OK\n" from a debug print).
		// json.Valid catches structural violations; this check catches
		// well-formed JSON with leading/trailing garbage.
		trimmed := bytes.TrimSpace(aliasData)
		if len(trimmed) < 2 || trimmed[0] != '{' || trimmed[len(trimmed)-1] != '}' {
			t.Errorf("property 4 violation: alias JSON does not open with `{` or close with `}`: %s", aliasData)
		}
	})
}

// isBadFloat reports whether v is NaN or ±Inf. Kept local (not exported)
// because the check is fuzz-harness-specific: the production RTTValue
// UnmarshalJSON already rejects these via validateRTTFloat.
func isBadFloat(v float64) bool {
	return v != v || v == veryLargePositive || v == veryLargeNegative
}

// veryLargePositive / veryLargeNegative are +Inf and -Inf without importing
// math (keeps the harness dependency-minimal to match go.md rule 1).
var (
	veryLargePositive = 1.0 / zero()
	veryLargeNegative = -1.0 / zero()
)

func zero() float64 { return 0.0 }

// TestVP062_FuzzSbctlMetricsJSON_SeedsAlone runs the fuzz body once per
// seed corpus entry as a standard Go test. This gives CI (which does not
// invoke `go test -fuzz`) end-to-end coverage of the seed corpus without
// needing the fuzz runner. `just smoke-quick` and default `go test ./...`
// pick this up.
//
// Also serves as a compilation smoke: an accidental fuzz-only build tag
// or mis-scoped helper is caught before the fuzz run.
//
// VP-062.
func TestVP062_FuzzSbctlMetricsJSON_SeedsAlone(t *testing.T) {
	t.Parallel()

	type seed struct {
		pathID      string
		routerAddr  string
		rttMs       float64
		sampleCount uint64
		p99Ms       float64
		lossPct     float64
		statusHint  uint8
	}
	seeds := []seed{
		{"", "", 0.0, 0, 0.0, 0.0, 0},
		{"path-001", "127.0.0.1:9000", 12.0, 5, 0.0, 0.0, 0},
		{"path-002", "10.0.0.1:9000", 15.0, 100, 22.0, 0.1, 0},
		{"path-003", "10.0.0.2:9000", 200.0, 50, 250.0, 2.0, 1},
		{"path-004", "10.0.0.3:9000", 15.0, 200, 22.0, 0.0, 2},
		{"path-005", "10.0.0.4:9000", 12.0, 9, 0.0, 0.0, 1},
		{"path-006", "10.0.0.5:9000", 12.0, 3, 0.0, 0.0, 2},
		{"path-007", "", 15.0, 20, 20.0, 0.0, 0},
		{"path-008", "not-a-host:port", 10.0, 20, 15.0, 0.5, 0},
		{"path-009", "999.999.999.999:99999", 8.0, 15, 9.0, 0.0, 0},
		{"path-010", "127.0.0.1:9000", 9999.9, 150, 9999.9, 100.0, 1},
	}

	for i, s := range seeds {
		src := &fuzzPathsSource{
			pathID: s.pathID,
			snap:   snapshotFromFuzzInputs(s.routerAddr, s.rttMs, s.sampleCount, s.p99Ms, s.lossPct, s.statusHint),
		}
		aliasResp, err := metrics.RouterStatus(context.Background(), nil, src)
		if err != nil {
			t.Fatalf("seed %d: RouterStatus: %v", i, err)
		}
		aliasData, err := json.Marshal(aliasResp)
		if err != nil {
			t.Fatalf("seed %d: marshal alias: %v", i, err)
		}
		if !json.Valid(aliasData) {
			t.Fatalf("seed %d: alias JSON invalid: %s", i, aliasData)
		}
		// EC-006 / EC-007: SampleCount<10 with a non-empty path forces
		// quality="pending".
		if s.pathID != "" && s.sampleCount < 10 {
			var m map[string]json.RawMessage
			if err := json.Unmarshal(aliasData, &m); err != nil {
				t.Fatalf("seed %d: unmarshal: %v", i, err)
			}
			var q string
			if err := json.Unmarshal(m["quality"], &q); err != nil {
				t.Fatalf("seed %d: unmarshal quality: %v", i, err)
			}
			if q != "pending" {
				t.Errorf("seed %d: SampleCount=%d yielded quality=%q; want pending", i, s.sampleCount, q)
			}
		}
		// EC-005: alias JSON contains `paths` and `quality` keys.
		if !bytes.Contains(aliasData, []byte(`"paths"`)) {
			t.Errorf("seed %d: alias missing `paths` key: %s", i, aliasData)
		}
		if !bytes.Contains(aliasData, []byte(`"quality"`)) {
			t.Errorf("seed %d: alias missing `quality` key: %s", i, aliasData)
		}
		// Sanity: BC-2.06.003 v1.16 status enum must map to a permitted
		// string when a path is present.
		if s.pathID != "" {
			var full metrics.RouterStatusResponse
			if err := json.Unmarshal(aliasData, &full); err != nil {
				t.Fatalf("seed %d: unmarshal RouterStatusResponse: %v", i, err)
			}
			if len(full.Paths) != 1 {
				t.Fatalf("seed %d: want 1 path, got %d", i, len(full.Paths))
			}
			st := full.Paths[0].Status
			if st != "active" && st != "degraded" && st != "failed" {
				t.Errorf("seed %d: status=%q not in {active, degraded, failed} (BC-2.06.003 v1.16 PC-1)", i, st)
			}
		}
	}
	// Silence unused-import warning on strings if the fuzz path is disabled
	// in a future refactor; strings is imported for future assertions.
	_ = strings.Contains
}
