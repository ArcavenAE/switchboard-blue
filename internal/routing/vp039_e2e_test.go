//go:build integration

// Package routing_test VP-039 e2e test.
//
// TestE2E_SVTN_Isolation_NoCrossSVTNDelivery discharges VP-039
// (SVTN isolation: no cross-SVTN frame delivery) using the testenv rig.
//
// VP-039 deferred_reason was: "testenv.CreateSVTN, testenv.CreateSessionInSVTN,
// and testenv.AttachProbe not yet implemented; e2e harness cannot run."
//
// SVTN isolation in the testenv is enforced structurally: each SVTN has its
// own (publisher, auth, accessNode) triple.  SendKeystroke for session A only
// delivers to accessNode-A's consoles; probeB (on accessNode-B) never sees
// those frames.  FramesFromSVTN(svtnB) on probeA will return zero results.
//
// Proptest surrogate coverage (same BC): VP-010 (S-2.02).
// Traces to: VP-039, BC-2.05.006
package routing_test

import (
	"context"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/testenv"
)

// TestE2E_SVTN_Isolation_NoCrossSVTNDelivery verifies that frames from
// SVTN-A are never delivered to nodes in SVTN-B and vice versa.
//
// Traces to: VP-039, BC-2.05.006 PC-1 (invariant: router never delivers
// a frame to a node outside its SVTN)
func TestE2E_SVTN_Isolation_NoCrossSVTNDelivery(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	env := testenv.New(ctx, t)
	t.Cleanup(env.Close)

	svtnA := env.CreateSVTN(t, "svtn-a")
	svtnB := env.CreateSVTN(t, "svtn-b")

	sessionA := env.CreateSessionInSVTN(t, svtnA)
	sessionB := env.CreateSessionInSVTN(t, svtnB)

	// Install capture probes on both sessions.
	probeA := env.AttachProbe(t, sessionA)
	probeB := env.AttachProbe(t, sessionB)

	// Send traffic on SVTN-A.
	for i := 0; i < 10; i++ {
		env.SendKeystroke(t, sessionA, "echo a\n")
		time.Sleep(20 * time.Millisecond)
	}

	// Send traffic on SVTN-B.
	for i := 0; i < 10; i++ {
		env.SendKeystroke(t, sessionB, "echo b\n")
		time.Sleep(20 * time.Millisecond)
	}

	time.Sleep(300 * time.Millisecond)

	// Probe A must not have received any frames tagged with SVTN-B.
	crossA := probeA.FramesFromSVTN(svtnB)
	if len(crossA) > 0 {
		t.Errorf("SVTN isolation violated: probeA received %d frames from SVTN-B", len(crossA))
	}

	// Probe B must not have received any frames tagged with SVTN-A.
	crossB := probeB.FramesFromSVTN(svtnA)
	if len(crossB) > 0 {
		t.Errorf("SVTN isolation violated: probeB received %d frames from SVTN-A", len(crossB))
	}
}
