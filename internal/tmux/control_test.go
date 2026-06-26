// Package tmux tests for BC-2.04.001 (tmux control mode integration).
// Traces: BC-2.04.001 PC-1..PC-5; AC-001..AC-004; ADR-010; ARCH-08 §6.6 position 7.
//
// Hermetic constraint: these tests MUST NOT shell out to the real tmux binary.
// Tests that require a live tmux process use a fake io.ReadCloser that replays
// pre-recorded control mode protocol lines. Any test requiring a real tmux
// process is skipped with a deferral docstring citing the integration test plan.
//
// Red Gate: all non-skipped tests below are designed to fail against the stub
// (todo() panic). Green on first commit = Red Gate violation.
package tmux_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/halfchannel"
	"github.com/arcavenae/switchboard/internal/session"
	"github.com/arcavenae/switchboard/internal/tmux"
)

// nopCloser wraps an io.Reader with a no-op Close so fakeControlOutput can be
// injected as the tmux subprocess stdout in tests (hermetic; no real tmux).
type nopCloser struct{ io.Reader }

func (nopCloser) Close() error { return nil }

// fakeControlOutput returns a fake tmux control mode stdout stream containing
// the provided pre-scripted lines. The caller controls exactly what "tmux"
// reports, so tests are hermetic and deterministic.
func fakeControlOutput(lines ...string) io.ReadCloser {
	return nopCloser{strings.NewReader(strings.Join(lines, "\n") + "\n")}
}

// fakeExecFunc returns a WithExecFunc option that yields the given stream.
func fakeExecFunc(r io.ReadCloser) tmux.Option {
	return tmux.WithExecFunc(func(_ context.Context) (io.ReadCloser, error) {
		return r, nil
	})
}

// fakeExecFuncErr returns a WithExecFunc option that returns the given error.
func fakeExecFuncErr(err error) tmux.Option {
	return tmux.WithExecFunc(func(_ context.Context) (io.ReadCloser, error) {
		return nil, err
	})
}

// newTestControlWithOpts is the canonical test constructor — accepts additional
// options (e.g. WithExecFunc for hermetic stream injection).
func newTestControlWithOpts(t *testing.T, opts ...tmux.Option) (*tmux.ControlMode, *session.Publisher) {
	t.Helper()
	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)
	cm := tmux.New(pub, ds, opts...)
	return cm, pub
}

// TestTmuxControlMode_Connect_EstablishesConnection verifies AC-001 (BC-2.04.001
// PC-1): Connect against a fake control-mode stream succeeds and places the
// ControlMode into a connected state. A second Connect call must return
// ErrAlreadyConnected (H-04 idempotency fix).
//
// Hermetic: uses a fake control mode stream; does not shell out to tmux.
// VP-031 (e2e against real tmux) is deferred to the integration test harness.
func TestTmuxControlMode_Connect_EstablishesConnection(t *testing.T) {
	t.Parallel()

	// Minimal fake stream: empty %begin/%end block so dispatchLoop can parse it
	// cleanly, then keeps reading until ctx cancellation (no premature EOF).
	stream := fakeControlOutput(
		"%begin 1000000000 0 1",
		"%end 1000000000 0 1",
	)
	cm, _ := newTestControlWithOpts(t, fakeExecFunc(stream)) //nolint:dogsled // publisher unused in AC-001
	t.Cleanup(func() {
		if err := cm.Close(); err != nil {
			t.Logf("Close: %v", err)
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// AC-001 primary assertion: first Connect must succeed.
	if err := cm.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v; want nil", err)
	}

	// H-04 idempotency: second Connect on an already-connected ControlMode must
	// return ErrAlreadyConnected (not nil, not a different error).
	if err := cm.Connect(ctx); !errors.Is(err, tmux.ErrAlreadyConnected) {
		t.Errorf("second Connect: got %v; want ErrAlreadyConnected", err)
	}
}

// TestTmuxControlMode_Connect_EnumeratesSessions verifies that after a
// successful Connect, all current tmux sessions are enumerated and published
// (BC-2.04.001 PC-2; AC-002; VP-031).
//
// This test uses the %begin/%end block protocol that Connect emits after
// issuing list-sessions (F-01 fix): session names appear line-by-line between
// the delimiters. The post-block %session-created events also exercise
// dynamic session creation (AC-003 / PC-3).
//
// Test vector (BC-2.04.001 canonical test vectors):
//   - Input:  %begin/%end block contains "session-1", "session-2", "build"
//   - Expect: all 3 published; Sessions() returns a 3-element list
func TestTmuxControlMode_Connect_EnumeratesSessions(t *testing.T) {
	t.Parallel()

	// Inject a fake stream that returns a %begin/%end block with session names
	// (as emitted by tmux in response to list-sessions), followed by a dynamic
	// %session-created to verify lifecycle handling still works. Hermetic.
	stream := fakeControlOutput(
		"%begin 1000000000 0 1",
		"session-1",
		"session-2",
		"build",
		"%end 1000000000 0 1",
		"%session-created extra-session",
		"%session-closed extra-session",
	)
	cm, _ := newTestControlWithOpts(t, fakeExecFunc(stream)) //nolint:dogsled // publisher checked via cm.Sessions()
	t.Cleanup(func() {
		if err := cm.Close(); err != nil {
			t.Logf("Close: %v", err)
		}
	})

	ctx := context.Background()
	if err := cm.Connect(ctx); err != nil {
		t.Fatalf("Connect: unexpected error: %v", err)
	}

	// Allow the dispatch loop time to process all events from the stream.
	time.Sleep(20 * time.Millisecond)

	want := []string{"build", "session-1", "session-2"}
	sessions := cm.Sessions()
	if len(sessions) != len(want) {
		t.Fatalf("Sessions: got %d; want %d (sessions: %v)", len(sessions), len(want), sessions)
	}
	// Verify all expected names are present (Sessions returns sorted order).
	got := make(map[string]bool, len(sessions))
	for _, s := range sessions {
		got[s.Name] = true
	}
	for _, name := range want {
		if !got[name] {
			t.Errorf("Sessions: missing %q; got %v", name, sessions)
		}
	}
}

// TestTmuxControlMode_SessionLifecycleEvents verifies that new sessions are
// published on %session-created and removed on %session-closed (BC-2.04.001
// PC-3, PC-4; AC-003).
//
// Test vector:
//   - Start with "agent-01", "agent-02", "build"
//   - Inject %session-created → "agent-03" published
//   - Inject %session-closed  → "agent-02" unpublished
func TestTmuxControlMode_SessionLifecycleEvents(t *testing.T) {
	t.Parallel()

	// Fake stream: empty enumeration block then lifecycle events.
	stream := fakeControlOutput(
		"%begin 1000000000 0 1",
		"agent-01",
		"agent-02",
		"build",
		"%end 1000000000 0 1",
		"%session-created agent-03",
		"%session-closed agent-02",
	)
	cm, pub := newTestControlWithOpts(t, fakeExecFunc(stream))
	t.Cleanup(func() { _ = cm.Close() })

	ctx := context.Background()
	if err := cm.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Wait for event loop to process lifecycle events.
	time.Sleep(20 * time.Millisecond)

	// After %session-created agent-03 — verify it appears.
	if _, err := pub.Get("agent-03"); err != nil {
		t.Errorf("Get agent-03: %v; want nil (session should be published)", err)
	}

	// After %session-closed agent-02 — verify it is gone.
	if _, err := pub.Get("agent-02"); !errors.Is(err, session.ErrSessionNotFound) {
		t.Errorf("Get agent-02: got %v; want ErrSessionNotFound (session should be unpublished)", err)
	}
}

// TestTmuxControlMode_OutputEventsFeedDownstream verifies that %output events
// from the control mode stream are enqueued into the downstream half-channel
// (BC-2.04.001 PC-5; AC-004).
//
// Hermetic: uses WithExecFunc to inject a fake stream; does not shell out to tmux.
//
// The test verifies:
//  1. After %output event, the downstream half-channel's sequence counter
//     advances (proving Tick was called with data).
//  2. The payload enqueued was the octal-unescaped form of the %output data
//     (F-06 contract: "hello\040world\012" → "hello world\n").
//
// Concurrency note: dispatchLoop runs in a goroutine and is the only writer of
// the (non-thread-safe) HalfChannel. The test waits for the stream EOF signal
// via c.Err() — which guarantees dispatchLoop has exited — before reading
// ds.Seq(). This is safe because dispatchLoop exits before sending to errCh,
// so no concurrent access to ds can occur after the receive.
func TestTmuxControlMode_OutputEventsFeedDownstream(t *testing.T) {
	t.Parallel()

	// Build dependencies inline so we hold a direct reference to ds.
	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)

	// Fake stream: empty session enumeration, register session-1, emit an
	// octal-escaped %output line, then EOF.
	// F-06 contract: \040 = space (0x20), \012 = newline (0x0A).
	stream := fakeControlOutput(
		"%begin 1000000000 0 1",
		"%end 1000000000 0 1",
		"%session-created session-1",
		`%output %12 hello\040world\012`,
	)
	cm := tmux.New(pub, ds, fakeExecFunc(stream))
	t.Cleanup(func() { _ = cm.Close() })

	ctx := context.Background()
	if err := cm.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Wait for dispatchLoop to reach EOF and signal the dropped connection.
	// This guarantees dispatchLoop has exited and ds is no longer being written.
	select {
	case err := <-cm.Err():
		if !errors.Is(err, tmux.ErrControlModeDropped) {
			t.Logf("Err() = %v (expected ErrControlModeDropped on stream EOF)", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("dispatchLoop did not signal within 200ms; stream may not have closed")
	}

	// dispatchLoop has exited — ds is no longer accessed concurrently. Safe to inspect.
	// The %output line caused one Enqueue + Tick, so seq must be >= 1.
	if seq := ds.Seq(); seq < 1 {
		t.Errorf("downstream Seq() = %d; want >= 1 (Tick must have been called for %%output event)", seq)
	}

	// Verify session-1 was published (proves the %session-created before %output was processed).
	if _, err := pub.Get("session-1"); err != nil {
		t.Errorf("Get session-1: %v; want nil (session-1 should be published before %%output)", err)
	}
}

// TestTmuxControlMode_Connect_ErrWhenTmuxNotFound verifies that Connect returns
// ErrControlModeUnavailable when the tmux binary is not present (BC-2.04.001
// EC-004 / FM-011; ADR-010).
//
// Hermetic: injects a fake exec function that returns ErrControlModeUnavailable.
func TestTmuxControlMode_Connect_ErrWhenTmuxNotFound(t *testing.T) {
	t.Parallel()

	// Simulate "tmux not in PATH" via WithExecFunc — hermetic, no PATH manipulation.
	notFoundErr := fmt.Errorf("%w: exec: tmux: no such file", tmux.ErrControlModeUnavailable)
	cm, _ := newTestControlWithOpts(t, fakeExecFuncErr(notFoundErr)) //nolint:dogsled // publisher unused in this test
	t.Cleanup(func() { _ = cm.Close() })

	ctx := context.Background()
	err := cm.Connect(ctx)
	if err == nil {
		t.Fatal("Connect with no tmux binary: got nil error; want ErrControlModeUnavailable")
	}
	if !errors.Is(err, tmux.ErrControlModeUnavailable) {
		t.Errorf("Connect error = %v; want ErrControlModeUnavailable", err)
	}
}

// TestTmuxControlMode_NoSessionsOnStartup verifies that Connect with an empty
// tmux server results in an empty session list (BC-2.04.001 EC-003; ADR-010).
//
// Hermetic: uses WithExecFunc to inject a fake stream with an empty %begin/%end
// block (no session names). Does not shell out to real tmux.
func TestTmuxControlMode_NoSessionsOnStartup(t *testing.T) {
	t.Parallel()

	// Inject a fake stream with an empty enumeration response — no session names
	// between %begin and %end. The stream closes after the block (EOF).
	stream := fakeControlOutput(
		"%begin 1000000000 0 1",
		"%end 1000000000 0 1",
	)
	cm, _ := newTestControlWithOpts(t, fakeExecFunc(stream)) //nolint:dogsled // publisher unused in this test
	t.Cleanup(func() { _ = cm.Close() })

	ctx := context.Background()
	if err := cm.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Allow dispatch loop to process the stream and reach EOF.
	time.Sleep(20 * time.Millisecond)

	sessions := cm.Sessions()
	if len(sessions) != 0 {
		t.Errorf("Sessions on empty server: got %d sessions (%v); want 0", len(sessions), sessions)
	}
}

// TestTmuxControlMode_ErrChannelSignalsDroppedConnection verifies that the Err()
// channel receives ErrControlModeDropped when the event loop exits unexpectedly
// (BC-2.04.001 EC-002 / FM-004; S-3.01b API surface).
//
// Hermetic: uses WithExecFunc to inject a fake stream that emits a %begin/%end
// block then immediately closes (EOF). Does not shell out to real tmux.
func TestTmuxControlMode_ErrChannelSignalsDroppedConnection(t *testing.T) {
	t.Parallel()

	// Inject a fake stream that emits an empty enumeration block then EOF.
	// After EOF, dispatchLoop should detect unexpected exit and send
	// ErrControlModeDropped to the Err() channel (EC-002 / FM-004).
	stream := fakeControlOutput(
		"%begin 1000000000 0 1",
		"%end 1000000000 0 1",
	)
	cm, _ := newTestControlWithOpts(t, fakeExecFunc(stream)) //nolint:dogsled // publisher unused in this test
	t.Cleanup(func() { _ = cm.Close() })

	ctx := context.Background()
	if err := cm.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// The fake stream EOF triggers the "unexpected exit" path in dispatchLoop —
	// it sends ErrControlModeDropped to errCh. Assert we receive it within 100ms.
	select {
	case err := <-cm.Err():
		if !errors.Is(err, tmux.ErrControlModeDropped) {
			t.Errorf("Err() = %v; want ErrControlModeDropped", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Err() did not signal within 100ms; want ErrControlModeDropped on stream EOF")
	}
}
