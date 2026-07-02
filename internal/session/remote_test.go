// Package session_test — remote.go RPC handler tests for sbctl console
// commands (BC-2.08.001; S-7.03).
//
// Traceability:
//
//	BC-2.08.001 — Console Remotely Controllable via sbctl
//	AC-001      — HandleConsoleAttach (attach)
//	AC-002      — HandleConsoleDetach (detach)
//	AC-003      — HandleConsoleSwitch (switch)
//	VP-050      — Console remote control end-to-end verification
//
// Red Gate: all tests MUST fail before implementation (handlers panic).
package session_test

import (
	"context"
	"errors"
	"testing"

	"github.com/arcavenae/switchboard/internal/session"
)

// TestConsoleRemote_E2E_VP050 is the integration test for VP-050:
// console remote control end-to-end verification covering the full
// AC-001/AC-002/AC-003 scenarios through HandleConsoleAttach,
// HandleConsoleDetach, and HandleConsoleSwitch.
//
// VP-050 — console remotely controllable via mgmt-plane Unix socket.
// BC-2.08.001 PC-1/PC-2/PC-3.
//
// The test exercises the full attach→detach→switch cycle:
// - AC-001: HandleConsoleAttach returns ConsoleAttachResponse.
// - AC-002: HandleConsoleDetach returns ConsoleDetachResponse.
// - AC-003: HandleConsoleSwitch returns ConsoleSwitchResponse.
// Failures surface through handler panics (Red Gate) until implementation lands.
func TestConsoleRemote_E2E_VP050(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// AC-001: HandleConsoleAttach — attach to a session.
	// BC-2.08.001 PC-1.
	attachResp, err := session.HandleConsoleAttach(ctx, session.ConsoleAttachRequest{
		SessionName: "agent-01",
	})
	if err != nil {
		t.Fatalf("VP-050 AC-001 — HandleConsoleAttach: %v", err)
	}
	if attachResp.SessionName != "agent-01" {
		t.Errorf("VP-050 AC-001 — ConsoleAttachResponse.SessionName: got %q; want %q",
			attachResp.SessionName, "agent-01")
	}

	// AC-002: HandleConsoleDetach — detach from the attached session.
	// BC-2.08.001 PC-2.
	detachResp, err := session.HandleConsoleDetach(ctx, session.ConsoleDetachRequest{})
	if err != nil {
		t.Fatalf("VP-050 AC-002 — HandleConsoleDetach: %v", err)
	}
	if detachResp.SessionName == "" {
		t.Error("VP-050 AC-002 — ConsoleDetachResponse.SessionName must be non-empty")
	}

	// AC-003: HandleConsoleSwitch — re-attach to another session atomically.
	// BC-2.08.001 PC-3.
	// Note: This calls switch after detach, so the detach leg will encounter
	// ErrConsoleNotAttached unless the implementation tracks state correctly.
	// For VP-050 e2e, we first attach again and then switch.
	_, err = session.HandleConsoleAttach(ctx, session.ConsoleAttachRequest{
		SessionName: "agent-01",
	})
	if err != nil {
		t.Fatalf("VP-050 AC-003 setup — HandleConsoleAttach: %v", err)
	}
	switchResp, err := session.HandleConsoleSwitch(ctx, session.ConsoleSwitchRequest{
		SessionName: "agent-02",
	})
	if err != nil {
		t.Fatalf("VP-050 AC-003 — HandleConsoleSwitch: %v", err)
	}
	if switchResp.SessionName != "agent-02" {
		t.Errorf("VP-050 AC-003 — ConsoleSwitchResponse.SessionName: got %q; want %q",
			switchResp.SessionName, "agent-02")
	}
}

// TestHandleConsoleAttach_UnknownSession verifies that HandleConsoleAttach
// returns ErrSessionNotFound (E-SES-001) when the named session does not exist.
//
// BC-2.08.001 PC-1; AC-001; EC-001.
func TestHandleConsoleAttach_UnknownSession(t *testing.T) {
	t.Parallel()

	// BC-2.08.001 PC-1 / EC-001 — unknown session must return ErrSessionNotFound.
	ctx := context.Background()

	_, err := session.HandleConsoleAttach(ctx, session.ConsoleAttachRequest{
		SessionName: "does-not-exist",
	})

	if err == nil {
		t.Fatal("BC-2.08.001 PC-1 EC-001 — HandleConsoleAttach unknown session: want non-nil error; got nil")
	}
	if !errors.Is(err, session.ErrSessionNotFound) {
		t.Errorf("BC-2.08.001 PC-1 EC-001 — HandleConsoleAttach unknown session: "+
			"want errors.Is(err, ErrSessionNotFound); got: %v", err)
	}
}

// TestHandleConsoleDetach_NotAttached verifies that HandleConsoleDetach
// returns ErrConsoleNotAttached (E-SES-004) when no session is attached.
//
// BC-2.08.001 PC-2; AC-002; EC-002.
func TestHandleConsoleDetach_NotAttached(t *testing.T) {
	t.Parallel()

	// BC-2.08.001 PC-2 / EC-002 — not attached must return ErrConsoleNotAttached.
	ctx := context.Background()

	_, err := session.HandleConsoleDetach(ctx, session.ConsoleDetachRequest{})

	if err == nil {
		t.Fatal("BC-2.08.001 PC-2 EC-002 — HandleConsoleDetach not attached: want non-nil error; got nil")
	}
	if !errors.Is(err, session.ErrConsoleNotAttached) {
		t.Errorf("BC-2.08.001 PC-2 EC-002 — HandleConsoleDetach not attached: "+
			"want errors.Is(err, ErrConsoleNotAttached); got: %v", err)
	}
}

// TestHandleConsoleSwitch_UnknownSession verifies that HandleConsoleSwitch
// returns ErrSessionNotFound (E-SES-001) when the target session does not exist.
//
// BC-2.08.001 PC-3; AC-003; EC-001.
func TestHandleConsoleSwitch_UnknownSession(t *testing.T) {
	t.Parallel()

	// BC-2.08.001 PC-3 / EC-001 — unknown target session must return ErrSessionNotFound.
	ctx := context.Background()

	_, err := session.HandleConsoleSwitch(ctx, session.ConsoleSwitchRequest{
		SessionName: "does-not-exist",
	})

	if err == nil {
		t.Fatal("BC-2.08.001 PC-3 EC-001 — HandleConsoleSwitch unknown session: want non-nil error; got nil")
	}
	if !errors.Is(err, session.ErrSessionNotFound) {
		t.Errorf("BC-2.08.001 PC-3 EC-001 — HandleConsoleSwitch unknown session: "+
			"want errors.Is(err, ErrSessionNotFound); got: %v", err)
	}
}
