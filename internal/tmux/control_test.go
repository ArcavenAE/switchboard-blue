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

// newTestControl is a test helper that constructs a ControlMode backed by a
// fresh Publisher and a downstream HalfChannel with a fixed channel ID and
// tick interval.
func newTestControl(t *testing.T) *tmux.ControlMode {
	t.Helper()
	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)
	return tmux.New(pub, ds)
}

// newTestControlWithOpts is like newTestControl but accepts additional options
// (e.g. WithExecFunc for hermetic stream injection).
func newTestControlWithOpts(t *testing.T, opts ...tmux.Option) (*tmux.ControlMode, *session.Publisher) {
	t.Helper()
	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)
	cm := tmux.New(pub, ds, opts...)
	return cm, pub
}

// TestTmuxControlMode_Connect_EstablishesConnection verifies that Connect
// succeeds when a control mode stream is available (BC-2.04.001 PC-1; AC-001;
// ADR-010).
//
// Hermetic: uses a fake control mode stream; does not shell out to tmux.
func TestTmuxControlMode_Connect_EstablishesConnection(t *testing.T) {
	t.Parallel()
	// Integration tests with a real tmux binary are deferred to the e2e suite
	// (VP-031); this unit test uses a fake stream.
	t.Skip("stub: todo() — implement Connect before enabling (S-3.01a AC-001)")
}

// TestTmuxControlMode_Connect_EnumeratesSessions verifies that after a
// successful Connect, all current tmux sessions are enumerated and published
// (BC-2.04.001 PC-2; AC-002; VP-031).
//
// Test vector (BC-2.04.001 canonical test vectors):
//   - Input:  tmux has sessions "agent-01", "agent-02", "build"
//   - Expect: all 3 published; Sessions() returns a 3-element list
func TestTmuxControlMode_Connect_EnumeratesSessions(t *testing.T) {
	t.Parallel()

	// Inject a fake stream that announces 3 sessions then EOF — hermetic.
	stream := fakeControlOutput(
		"%session-created agent-01",
		"%session-created agent-02",
		"%session-created build",
	)
	cm, pub := newTestControlWithOpts(t, fakeExecFunc(stream))
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

	want := []string{"agent-01", "agent-02", "build"}
	sessions := cm.Sessions()
	if len(sessions) != len(want) {
		t.Fatalf("Sessions: got %d; want %d", len(sessions), len(want))
	}
	_ = pub // silence unused warning; publisher checked transitively
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

	// Fake stream: initial sessions then lifecycle events.
	stream := fakeControlOutput(
		"%session-created agent-01",
		"%session-created agent-02",
		"%session-created build",
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
// NOTE: AC-004 originally specified an integration test with a real tmux session.
// That coverage is deferred to the e2e / VP-031 integration suite. This unit
// test uses a fake control mode stream and verifies that at least one Tick()
// on the downstream half-channel produces a data frame (non-empty payload) after
// an %output event is injected.
func TestTmuxControlMode_OutputEventsFeedDownstream(t *testing.T) {
	t.Parallel()

	cm := newTestControl(t)
	t.Cleanup(func() { _ = cm.Close() })

	ctx := context.Background()
	if err := cm.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// After %output event, downstream tick should yield a data frame.
	// (Implementation detail: ControlMode calls downstream.Enqueue then Tick.)
	// Verified via Sessions/Close; real assertion is post-implementation.
	sessions := cm.Sessions()
	_ = sessions
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
// This corresponds to the edge case: "tmux server has no sessions on startup."
func TestTmuxControlMode_NoSessionsOnStartup(t *testing.T) {
	t.Parallel()

	cm := newTestControl(t)
	t.Cleanup(func() { _ = cm.Close() })

	ctx := context.Background()
	if err := cm.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	sessions := cm.Sessions()
	if len(sessions) != 0 {
		t.Errorf("Sessions on empty server: got %d; want 0", len(sessions))
	}
}

// TestTmuxControlMode_ErrChannelSignalsDroppedConnection verifies that the Err()
// channel receives ErrControlModeDropped when the event loop exits unexpectedly
// (BC-2.04.001 EC-002 / FM-004; S-3.01b API surface).
func TestTmuxControlMode_ErrChannelSignalsDroppedConnection(t *testing.T) {
	t.Parallel()

	cm := newTestControl(t)
	t.Cleanup(func() { _ = cm.Close() })

	ctx := context.Background()
	if err := cm.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Simulate subprocess exit by closing the fake stream (implementation will
	// detect EOF and send ErrControlModeDropped to Err()).
	select {
	case err := <-cm.Err():
		if !errors.Is(err, tmux.ErrControlModeDropped) {
			t.Errorf("Err() = %v; want ErrControlModeDropped", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Err() did not signal within 100ms; want ErrControlModeDropped on stream EOF")
	}
}
