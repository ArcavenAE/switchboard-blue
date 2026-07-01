package metrics_test

import (
	"testing"
)

// TestVP047_SbctlPathsList_EndToEnd is the VP-047 integration test.
// It spins up a daemon with two synthetic paths (one pending, one green),
// runs "sbctl paths list --json", and asserts all required fields are present
// and non-null per BC-2.06.003 PC-1:
//   - path_id, router_addr, rtt_ms, rtt_p99_ms (float64 or "pending"), loss_pct, status
//
// The pending path has SampleCount < 10; the green path has SampleCount ≥ 10.
//
// AC-006; VP-047; BC-2.06.003 PC-1.
func TestVP047_SbctlPathsList_EndToEnd(t *testing.T) {
	t.Fatal("TODO: S-W5.04 AC-006 / VP-047 not yet written")
}
