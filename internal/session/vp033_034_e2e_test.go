//go:build integration

// Package session_test VP-033 + VP-034 e2e tests.
//
// These tests discharge VP-033 (Console.Attach/Detach lifecycle) and VP-034
// (multi-console fan-out) using the internal/testenv in-process rig.
//
// Build tag: integration (requires testenv package — not part of unit test run).
package session_test

import (
	"context"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/testenv"
)

// TestE2E_Console_AttachDetachLifecycle verifies the full attach/detach
// lifecycle across a running router + access + console stack.
//
// Traces to: VP-033, BC-2.04.003, BC-2.04.004
func TestE2E_Console_AttachDetachLifecycle(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	env := testenv.New(t, ctx)
	t.Cleanup(env.Close)

	sessionID := env.CreateSession(t)

	// --- Attach phase ---
	console := env.AttachConsole(t, sessionID)

	// Generate downstream output by sending a keystroke.
	env.SendKeystroke(t, sessionID, "echo hello\n")

	// Wait for downstream frames to arrive at the console.
	frames := console.CollectFrames(t, 2*time.Second)
	if len(frames) == 0 {
		t.Fatal("expected downstream frames after attach; received none")
	}

	// --- Detach phase ---
	console.Detach(t)

	// Allow any in-flight frames to drain.
	time.Sleep(200 * time.Millisecond)
	beforeCount := len(console.CollectFrames(t, 0))

	// Generate more output on the session.
	env.SendKeystroke(t, sessionID, "echo world\n")
	time.Sleep(500 * time.Millisecond)

	framesAfterDetach := console.CollectFrames(t, 0)
	if len(framesAfterDetach) > beforeCount {
		t.Errorf("frames received after detach: expected 0 new frames, got %d",
			len(framesAfterDetach)-beforeCount)
	}

	// Session must still be alive on the access node.
	if !env.SessionAlive(t, sessionID) {
		t.Error("session was terminated after console detach; expected it to survive")
	}
}

// TestE2E_MultiConsole_FanOut verifies that two simultaneously attached
// consoles both receive all downstream frames from a session.
//
// Traces to: VP-034, BC-2.04.006
func TestE2E_MultiConsole_FanOut(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	env := testenv.New(t, ctx)
	t.Cleanup(env.Close)

	sessionID := env.CreateSession(t)

	console1 := env.AttachConsole(t, sessionID)
	console2 := env.AttachConsole(t, sessionID)

	// Generate downstream output.
	const lines = 5
	for i := 0; i < lines; i++ {
		env.SendKeystroke(t, sessionID, "echo line\n")
		time.Sleep(50 * time.Millisecond)
	}

	// Allow frames to propagate.
	time.Sleep(500 * time.Millisecond)

	frames1 := console1.CollectFrames(t, 0)
	frames2 := console2.CollectFrames(t, 0)

	if len(frames1) == 0 {
		t.Error("console1 received no frames")
	}
	if len(frames2) == 0 {
		t.Error("console2 received no frames")
	}

	// Both consoles must have received the same number of frames.
	if len(frames1) != len(frames2) {
		t.Errorf("frame count mismatch: console1=%d console2=%d", len(frames1), len(frames2))
	}
}
