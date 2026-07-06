//go:build integration

// Package admission_test VP-036 e2e test.
//
// TestE2E_Session_ContinuityAcrossIPChange discharges VP-036
// (session continuity across IP address change) using the testenv rig.
//
// VP-036 deferred_reason was: "internal/testenv.ConnectWithSourceIP not yet
// implemented; t.Skip placeholder … in reauth_test.go"
//
// This file provides the integration-tagged harness.  The unit-scope
// node-address-stability test (TestSessionContinuity_NodeAddressStableAfterReauth)
// remains in admission_test.go and covers BC-2.01.007 invariant 3.
package admission_test

import (
	"context"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/testenv"
)

// TestE2E_Session_ContinuityAcrossIPChange verifies that a session survives
// a source IP address change after re-authentication.
//
// Traces to: VP-036, BC-2.01.007 (session identity = channel_id + node_addr;
// IP is not part of session identity)
func TestE2E_Session_ContinuityAcrossIPChange(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	env := testenv.New(t, ctx)
	t.Cleanup(env.Close)

	// Establish initial session from IP A.
	creds := env.GenerateCredentials(t)
	conn1 := env.ConnectWithSourceIP(t, "192.0.2.1", creds)
	sessionID := conn1.SessionID()

	// Confirm session is active — send a keystroke and collect frames.
	env.SendKeystroke(t, sessionID, "echo before\n")
	frames1 := conn1.CollectFrames(t, 2*time.Second)
	if len(frames1) == 0 {
		t.Fatal("no frames received before IP change")
	}

	// Simulate IP address change: close old connection, reconnect from IP B.
	conn1.Close()
	time.Sleep(100 * time.Millisecond)

	conn2 := env.ConnectWithSourceIP(t, "192.0.2.2", creds)
	t.Cleanup(conn2.Close)

	// Session ID must be the same — IP change must not break session identity.
	if conn2.SessionID() != sessionID {
		t.Errorf("session ID changed after IP change: before=%s after=%s",
			sessionID, conn2.SessionID())
	}

	// Traffic must resume on the new connection.
	env.SendKeystroke(t, sessionID, "echo after\n")
	frames2 := conn2.CollectFrames(t, 3*time.Second)
	if len(frames2) == 0 {
		t.Error("no frames received after IP change and re-authentication")
	}
}
