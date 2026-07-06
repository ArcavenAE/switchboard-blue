//go:build integration

// Package multipath_test VP-040 e2e test.
//
// TestE2E_Multipath_FailoverRecovery discharges the harness portion of VP-040
// (multipath failover recovery < 2s / NFR-003).
//
// Proof status: PARTIAL — CloseRouterConnection marks router[0] as closed
// and subsequent SendKeystroke + CollectFrames continue to succeed via the
// surviving router, confirming the in-process recovery path.  The wall-clock
// < 2s claim requires the production multipath dispatch layer (S-4.01 + its
// path-selection loop); the testenv loop provides the structural harness.
//
// Traces to: VP-040, BC-2.02.003, NFR-003
package multipath_test

import (
	"context"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/testenv"
)

// TestE2E_Multipath_FailoverRecovery verifies that after one router connection
// is closed, traffic resumes on the remaining router within 2 seconds.
//
// Traces to: VP-040, BC-2.02.003, NFR-003
func TestE2E_Multipath_FailoverRecovery(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	// Two-router topology; session uses both paths.
	env := testenv.NewWithRouters(t, ctx, 2)
	t.Cleanup(env.Close)

	sessionID := env.CreateSession(t)
	console := env.AttachConsole(t, sessionID)

	// Wait for both paths to be active.
	if err := env.WaitForPaths(t, sessionID, 2, 5*time.Second); err != nil {
		t.Fatalf("did not reach 2 active paths: %v", err)
	}

	// Confirm traffic flowing.
	env.SendKeystroke(t, sessionID, "echo pre-fail\n")
	pre := console.CollectFrames(t, 2*time.Second)
	if len(pre) == 0 {
		t.Fatal("no frames before path failure")
	}

	// Kill path 0 (close its router connection).
	failAt := time.Now()
	env.CloseRouterConnection(t, 0)

	// Poll until traffic resumes on the surviving router, up to 2s.
	const maxRecovery = 2 * time.Second
	deadline := failAt.Add(maxRecovery)
	var recovered bool
	for time.Now().Before(deadline) {
		env.SendKeystroke(t, sessionID, "echo post-fail\n")
		post := env.CollectFrames(t, sessionID, 200*time.Millisecond)
		if len(post) > 0 {
			recovered = true
			break
		}
	}

	if !recovered {
		elapsed := time.Since(failAt)
		t.Errorf("multipath failover took longer than %v (elapsed %v)", maxRecovery, elapsed)
	}
}
