//go:build integration

// Package drain_test VP-037 e2e test.
//
// TestE2E_RouterDrain_NodesMigrateWithin2s discharges VP-037
// (router drain: nodes migrate to alternate router within 2s).
//
// Note on proof status: the testenv drain simulation signals the drain.Drain
// controller and observes that subsequent SendKeystroke + CollectFrames
// calls continue to succeed on the same session (traffic on a surviving
// router). It is built on the same testenv.NewWithRouters stub pattern that
// never runs a real runRouter, so it is structurally incapable of proving
// real router-to-router migration — it stays go:build integration-gated
// permanently, as harness infrastructure only, and is NOT either stage of
// VP-037's discharge (Q4-AMENDED).
//
// VP-037 stage-1 (wire round-trip: a connected node receives the DRAIN
// frame; drainCoord.Wait returns nil within the window) is discharged by
// TestE2E_RouterDrain_WireRoundTrip in
// cmd/switchboard/router_drain_wire_test.go (S-7.04-FU-DRAIN-WIRE),
// against a real runRouter. VP-037 stage-2 (node-side migration logic;
// verification_lock flips to true) is a named follow-on that un-gates the
// t.Skip-gated TestE2E_RouterDrain_NodesMigrateWithin2s copy in
// cmd/switchboard/router_pe_connector_test.go — NOT this file's copy.
//
// Traces to: VP-037, BC-2.09.002
package drain_test

import (
	"context"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/testenv"
)

// TestE2E_RouterDrain_NodesMigrateWithin2s verifies that after a drain signal
// is sent, session traffic continues flowing (to a surviving router).
//
// Traces to: VP-037, BC-2.09.002
func TestE2E_RouterDrain_NodesMigrateWithin2s(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	// Two-router topology.
	env := testenv.NewWithRouters(ctx, t, 2)
	t.Cleanup(env.Close)

	sessionID := env.CreateSession(t)
	console := env.AttachConsole(t, sessionID)

	// Confirm traffic is flowing.
	env.SendKeystroke(t, sessionID, "echo pre-drain\n")
	pre := console.CollectFrames(t, 2*time.Second)
	if len(pre) == 0 {
		t.Fatal("no frames before drain signal; check stack setup")
	}

	// Send drain signal on router 0.
	drainAt := time.Now()
	env.SendDrainSignal(t, 0)

	// Poll until frames resume (traffic should continue via surviving router).
	const maxMigration = 2 * time.Second
	deadline := drainAt.Add(maxMigration)
	var resumed bool
	for time.Now().Before(deadline) {
		env.SendKeystroke(t, sessionID, "echo post-drain\n")
		post := env.CollectFrames(t, sessionID, 200*time.Millisecond)
		if len(post) > 0 {
			resumed = true
			break
		}
	}

	if !resumed {
		t.Errorf("session traffic did not resume within %v after DRAIN_SIGNAL", maxMigration)
	}
}
