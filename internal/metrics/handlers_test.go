package metrics_test

import (
	"testing"
)

// TestDaemonPathsList_HandlerRegistered verifies that the paths.list RPC handler
// is registered in the mgmt dispatch and returns a PathsListResponse conforming
// to the BC-2.06.003 PC-1 schema with at least one entry.
// AC-001; BC-2.06.003 PC-1.
func TestDaemonPathsList_HandlerRegistered(t *testing.T) {
	t.Fatal("TODO: S-W5.04 AC-001 not yet written")
}

// TestPathEntry_RTTValueSerialization verifies RTTValue.MarshalJSON union behaviour:
//
//	row (a) SampleCount=0  → "pending"
//	row (b) SampleCount=9  → "pending"
//	row (c) SampleCount=10 → float64
//	row (d) SampleCount=100 → float64
//
// AC-002; BC-2.06.003 PC-1, EC-003.
func TestPathEntry_RTTValueSerialization(t *testing.T) {
	t.Fatal("TODO: S-W5.04 AC-002 not yet written")
}

// TestPathEntry_StatusFromDegraded verifies that PathEntry.Status is set to
// "degraded" when PathSnapshot.Degraded==true and "ok" otherwise.
// AC-003; BC-2.06.001.
func TestPathEntry_StatusFromDegraded(t *testing.T) {
	t.Fatal("TODO: S-W5.04 AC-003 not yet written")
}

// TestDaemonRouterMetrics_HandlerRegistered verifies that the router.metrics RPC
// handler is registered and returns a RouterMetricsResponse conforming to the
// BC-2.06.003 PC-2 schema.
// AC-004; BC-2.06.003 PC-2.
func TestDaemonRouterMetrics_HandlerRegistered(t *testing.T) {
	t.Fatal("TODO: S-W5.04 AC-004 not yet written")
}

// TestDaemonRouterStatus_HandlerRegistered verifies that the router.status RPC
// handler is registered and returns response fields matching the paths.list shape
// plus a "quality" summary field.
// AC-005; BC-2.06.003 PC-3.
func TestDaemonRouterStatus_HandlerRegistered(t *testing.T) {
	t.Fatal("TODO: S-W5.04 AC-005 not yet written")
}

// TestDaemonRouterStatus_FailedAndPendingPrecedence verifies the failed+pending
// precedence ruling (S502-DEFER-3; BC-2.06.003 v1.8 EC-007):
//
//	row (a) Degraded=true  + SampleCount=5  → quality "pending", status "failed"
//	row (b) Degraded=true  + SampleCount=10 → quality derived from p99 (not pending)
//	row (c) Degraded=false + SampleCount=5  → quality "pending", status "active"
//
// AC-005a; BC-2.06.003 v1.8 EC-007; S502-DEFER-3.
func TestDaemonRouterStatus_FailedAndPendingPrecedence(t *testing.T) {
	t.Fatal("TODO: S-W5.04 AC-005a not yet written")
}
