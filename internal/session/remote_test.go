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
// Unit-level tests: each test constructs a per-test ConsoleServer with an
// isolated ConsoleState (L2-T1: no shared package-level state; no parallel
// data race). End-to-end verification through a real mgmt.Server lives in
// cmd/switchboard/console_handlers_e2e_test.go (VP-050; L2-T5).
package session_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"testing"

	"github.com/arcavenae/switchboard/internal/session"
)

// stubRegistry is a minimal SessionRegistry that returns true for a fixed set
// of session names. Used by unit tests to avoid a live Publisher dependency.
type stubRegistry struct {
	known map[string]struct{}
}

func newStubRegistry(names ...string) *stubRegistry {
	m := make(map[string]struct{}, len(names))
	for _, n := range names {
		m[n] = struct{}{}
	}
	return &stubRegistry{known: m}
}

func (s *stubRegistry) Exists(name string) bool {
	_, ok := s.known[name]
	return ok
}

// newTestConsoleServer returns a ConsoleServer + ConsoleState pair for a single
// test. The state is reset on t.Cleanup (L2-T1: per-test isolation, no race).
func newTestConsoleServer(t *testing.T, reg session.SessionRegistry) *session.ConsoleServer {
	t.Helper()
	state := session.NewConsoleState()
	t.Cleanup(func() {
		// State is per-test; nothing to reset — GC handles it.
	})
	return session.NewConsoleServer(reg, state)
}

// TestHandleConsoleAttach_Success verifies that HandleConsoleAttach returns the
// correct ConsoleAttachResponse for a known session (BC-2.08.001 PC-1; AC-001).
func TestHandleConsoleAttach_Success(t *testing.T) {
	t.Parallel()

	reg := newStubRegistry("agent-01", "agent-02")
	cs := newTestConsoleServer(t, reg)
	ctx := context.Background()

	resp, err := cs.HandleConsoleAttach(ctx, session.ConsoleAttachRequest{SessionName: "agent-01"})
	if err != nil {
		t.Fatalf("BC-2.08.001 PC-1 — HandleConsoleAttach: unexpected error: %v", err)
	}
	if resp.SessionName != "agent-01" {
		t.Errorf("BC-2.08.001 PC-1 — ConsoleAttachResponse.SessionName: got %q; want %q",
			resp.SessionName, "agent-01")
	}
}

// TestHandleConsoleDetach_Success verifies that HandleConsoleDetach returns the
// correct ConsoleDetachResponse after a successful attach (BC-2.08.001 PC-2; AC-002).
func TestHandleConsoleDetach_Success(t *testing.T) {
	t.Parallel()

	reg := newStubRegistry("agent-01")
	cs := newTestConsoleServer(t, reg)
	ctx := context.Background()

	// Set up: attach first.
	if _, err := cs.HandleConsoleAttach(ctx, session.ConsoleAttachRequest{SessionName: "agent-01"}); err != nil {
		t.Fatalf("BC-2.08.001 PC-2 setup — HandleConsoleAttach: %v", err)
	}

	// Detach.
	resp, err := cs.HandleConsoleDetach(ctx, session.ConsoleDetachRequest{})
	if err != nil {
		t.Fatalf("BC-2.08.001 PC-2 — HandleConsoleDetach: unexpected error: %v", err)
	}
	if resp.SessionName != "agent-01" {
		t.Errorf("BC-2.08.001 PC-2 — ConsoleDetachResponse.SessionName: got %q; want %q",
			resp.SessionName, "agent-01")
	}
}

// TestHandleConsoleSwitch_Success verifies that HandleConsoleSwitch correctly
// transitions to the target session (BC-2.08.001 PC-3; AC-003; L1-C3).
//
// L1-C3 assertion: after a successful switch, ConsoleServer internal state
// tracks the new session name (verified by a subsequent detach returning the
// new name rather than "").
func TestHandleConsoleSwitch_Success(t *testing.T) {
	t.Parallel()

	reg := newStubRegistry("agent-01", "agent-02")
	cs := newTestConsoleServer(t, reg)
	ctx := context.Background()

	// Set up: attach to agent-01.
	if _, err := cs.HandleConsoleAttach(ctx, session.ConsoleAttachRequest{SessionName: "agent-01"}); err != nil {
		t.Fatalf("BC-2.08.001 PC-3 setup — HandleConsoleAttach: %v", err)
	}

	// Switch to agent-02.
	switchResp, err := cs.HandleConsoleSwitch(ctx, session.ConsoleSwitchRequest{SessionName: "agent-02"})
	if err != nil {
		t.Fatalf("BC-2.08.001 PC-3 — HandleConsoleSwitch: unexpected error: %v", err)
	}
	if switchResp.SessionName != "agent-02" {
		t.Errorf("BC-2.08.001 PC-3 — ConsoleSwitchResponse.SessionName: got %q; want %q",
			switchResp.SessionName, "agent-02")
	}

	// L1-C3 assertion: state must now track agent-02 (not "").
	// A subsequent detach must return agent-02, proving state was set to agent-02.
	detachResp, err := cs.HandleConsoleDetach(ctx, session.ConsoleDetachRequest{})
	if err != nil {
		t.Fatalf("BC-2.08.001 PC-3 L1-C3 — post-switch detach: unexpected error: %v", err)
	}
	if detachResp.SessionName != "agent-02" {
		t.Errorf("BC-2.08.001 PC-3 L1-C3 — post-switch detach.SessionName: got %q; want %q (L1-C3: state must track new session)",
			detachResp.SessionName, "agent-02")
	}
}

// TestHandleConsoleAttach_UnknownSession verifies that HandleConsoleAttach
// returns ErrSessionNotFound (E-SES-001) when the named session does not exist.
//
// BC-2.08.001 PC-1; AC-001; EC-001.
func TestHandleConsoleAttach_UnknownSession(t *testing.T) {
	t.Parallel()

	reg := newStubRegistry("agent-01") // does-not-exist is not registered
	cs := newTestConsoleServer(t, reg)
	ctx := context.Background()

	_, err := cs.HandleConsoleAttach(ctx, session.ConsoleAttachRequest{
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

	reg := newStubRegistry("agent-01")
	cs := newTestConsoleServer(t, reg)
	ctx := context.Background()

	_, err := cs.HandleConsoleDetach(ctx, session.ConsoleDetachRequest{})

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

	reg := newStubRegistry("agent-01")
	cs := newTestConsoleServer(t, reg)
	ctx := context.Background()

	_, err := cs.HandleConsoleSwitch(ctx, session.ConsoleSwitchRequest{
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

// TestHandleConsoleSwitch_NotAttached verifies that HandleConsoleSwitch
// returns ErrConsoleNotAttached (E-SES-004) when no session is currently attached,
// even when the target session is known.
//
// BC-2.08.001 PC-3; AC-003; EC-002.
func TestHandleConsoleSwitch_NotAttached(t *testing.T) {
	t.Parallel()

	reg := newStubRegistry("agent-01", "agent-02")
	cs := newTestConsoleServer(t, reg)
	ctx := context.Background()

	// No attach first — switch must fail with ErrConsoleNotAttached.
	_, err := cs.HandleConsoleSwitch(ctx, session.ConsoleSwitchRequest{
		SessionName: "agent-02",
	})

	if err == nil {
		t.Fatal("BC-2.08.001 PC-3 EC-002 — HandleConsoleSwitch not attached: want non-nil error; got nil")
	}
	if !errors.Is(err, session.ErrConsoleNotAttached) {
		t.Errorf("BC-2.08.001 PC-3 EC-002 — HandleConsoleSwitch not attached: "+
			"want errors.Is(err, ErrConsoleNotAttached); got: %v", err)
	}
}

// TestConsoleState_ConcurrentAttachDetachSwitchIsRaceFree exercises ConsoleState
// under concurrent Attach, Detach, and Switch calls to verify there are no data
// races on the internal mutex-protected state (F-P2L2-001; L2-T1).
//
// The test spawns N=100 goroutines. Each goroutine performs a randomised sequence
// of operations against a shared ConsoleState. No assertion on intermediate values
// is made — races are the target, detected by `go test -race`. The only value
// assertion is at the end: ConsoleState.Current() must return either "" or one of
// the known session names (not a torn write).
//
// Run via: just test-race (go test -race ./...)
func TestConsoleState_ConcurrentAttachDetachSwitchIsRaceFree(t *testing.T) {
	t.Parallel()

	const goroutines = 100

	// Build N session names and a registry that knows all of them.
	sessionNames := make([]string, goroutines)
	for i := range goroutines {
		sessionNames[i] = fmt.Sprintf("sess-%d", i)
	}
	reg := newStubRegistry(sessionNames...)

	state := session.NewConsoleState()
	srv := session.NewConsoleServer(reg, state)
	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(idx int) {
			defer wg.Done()

			// Use a per-goroutine random source so goroutines don't share state.
			//nolint:gosec // not used for cryptography; local test entropy only
			r := rand.New(rand.NewSource(int64(idx)))

			for range 30 {
				switch r.Intn(3) {
				case 0: // Attach
					name := sessionNames[r.Intn(len(sessionNames))]
					_, _ = srv.HandleConsoleAttach(ctx, session.ConsoleAttachRequest{SessionName: name})
				case 1: // Detach
					_, _ = srv.HandleConsoleDetach(ctx, session.ConsoleDetachRequest{})
				case 2: // Switch
					name := sessionNames[r.Intn(len(sessionNames))]
					_, _ = srv.HandleConsoleSwitch(ctx, session.ConsoleSwitchRequest{SessionName: name})
				}
			}
		}(i)
	}

	wg.Wait()

	// Post-condition: Current() must return "" or a known session name — no torn write.
	got := state.Current()
	if got == "" {
		return // valid: no session attached after all operations
	}
	known := make(map[string]struct{}, len(sessionNames))
	for _, n := range sessionNames {
		known[n] = struct{}{}
	}
	if _, ok := known[got]; !ok {
		t.Errorf("ConsoleState.Current() returned unknown session %q after concurrent operations (torn write?)", got)
	}
}
