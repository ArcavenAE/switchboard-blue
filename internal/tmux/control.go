// Package tmux implements tmux control mode integration for the access node
// (ARCH-08 §6.6 position 7; BC-2.04.001; ADR-010).
//
// Classification: effectful (ARCH-09). This package spawns the tmux subprocess
// via os/exec, consumes the tmux control mode event stream, drives the
// downstream half-channel tick loop, and surfaces a session lifecycle event
// channel to callers (S-3.01b reads the Err() channel for fallback signalling).
//
// Allowed internal imports: {halfchannel, session} per ARCH-08 §6.6.
// Forbidden: internal/admission, internal/routing (ARCH-08 §6.6 forbidden edges).
package tmux

import (
	"context"
	"errors"
	"io"

	"github.com/arcavenae/switchboard/internal/halfchannel"
	"github.com/arcavenae/switchboard/internal/session"
)

// ErrControlModeUnavailable is returned by Connect when the tmux binary is not
// found in PATH (BC-2.04.001 EC-004 / FM-011; log: "tmux not found; using PTY
// fallback"). FM-004 explicitly assigns no catalog code to this degradation
// signal; the sentinel here is provided for errors.Is checks in S-3.01b.
var ErrControlModeUnavailable = errors.New("tmux: control mode unavailable")

// ErrControlModeDropped signals that a previously established control mode
// connection has been lost mid-session (BC-2.04.001 EC-002 / FM-004).
// Callers (S-3.01b fallback path) receive this via the Err() channel.
var ErrControlModeDropped = errors.New("tmux: control mode connection dropped")

// ControlMode manages a single tmux control mode connection and bridges
// control mode events to the session publisher and downstream half-channel.
//
// The zero value is not usable; construct with New.
//
// Concurrency: ControlMode spawns exactly one internal goroutine (the event
// dispatch loop) when Connect succeeds. All exported methods are safe to call
// from any goroutine.
type ControlMode struct {
	publisher  *session.Publisher       //nolint:unused // Red Gate stub — used post-implementation
	downstream *halfchannel.HalfChannel //nolint:unused // Red Gate stub — used post-implementation
	errCh      chan error               //nolint:unused // Red Gate stub — used post-implementation
	cancel     context.CancelFunc       //nolint:unused // Red Gate stub — used post-implementation
	// proc holds the running tmux process handle (set by Connect).
	// Type is io.ReadCloser to allow fake injection in tests (hermetic; no
	// direct reference to os.Process which would require a real tmux binary).
	proc io.ReadCloser //nolint:unused // Red Gate stub — used post-implementation
}

// New constructs a ControlMode that publishes sessions via publisher and
// delivers output frames to downstream (BC-2.04.001; S-3.01a task 5+6).
//
// publisher and downstream must not be nil.
func New(publisher *session.Publisher, downstream *halfchannel.HalfChannel) *ControlMode {
	todo() // TODO(S-3.01a): implement; init errCh (unbuffered — no buffer without justification)
	return nil
}

// Connect launches `tmux -C` as a subprocess and begins the control mode event
// subscription loop (BC-2.04.001 PC-1; AC-001).
//
// On success: the event loop goroutine is running and all current sessions are
// enumerated and published (BC-2.04.001 PC-2; AC-002).
//
// Returns ErrControlModeUnavailable if tmux is not found in PATH (EC-004).
// Returns a wrapped error for any other subprocess launch failure.
//
// Connect is idempotent — calling Connect on an already-connected ControlMode
// is an error (caller bug; the caller must Close first).
func (c *ControlMode) Connect(ctx context.Context) error {
	todo() // TODO(S-3.01a): implement per BC-2.04.001 PC-1; AC-001
	return nil
}

// Sessions returns a snapshot of all currently published tmux sessions
// (BC-2.04.001 PC-2; VP-031).
//
// The slice is a value copy — callers may freely mutate it.
func (c *ControlMode) Sessions() []session.Info {
	todo() // TODO(S-3.01a): implement per BC-2.04.001 PC-2; VP-031
	return nil
}

// Err returns the error channel that receives a non-nil error when the
// control mode connection is lost (BC-2.04.001 EC-002; S-3.01b API surface).
//
// The channel is closed and sends ErrControlModeDropped when the event loop
// exits unexpectedly. S-3.01b reads this channel to trigger PTY fallback.
//
// The channel is never written to by the caller.
func (c *ControlMode) Err() <-chan error {
	todo() // TODO(S-3.01a): implement; return c.errCh
	return nil
}

// Close shuts down the event loop and terminates the tmux subprocess.
// It is safe to call Close multiple times; subsequent calls are no-ops.
func (c *ControlMode) Close() error {
	todo() // TODO(S-3.01a): implement; cancel ctx, drain errCh, close proc
	return nil
}

// dispatchLoop is the internal goroutine started by Connect. It reads control
// mode protocol lines from the tmux subprocess stdout, dispatches them to the
// appropriate handler, and drives the downstream half-channel Tick loop.
//
// Protocol events handled (BC-2.04.001 trigger + PC-3, PC-4, PC-5):
//   - %begin / %end     — command response delimiters
//   - %sessions-changed — re-enumerate sessions
//   - %session-created  — AC-003 (PC-3)
//   - %session-closed   — AC-003 (PC-4)
//   - %output           — AC-004 (PC-5): feed downstream half-channel
//
// dispatchLoop is unexported; it is started only by Connect and exits when the
// subprocess exits or ctx is cancelled.
//
//nolint:unused,unparam // Red Gate stub — ctx and r both used post-implementation
func (c *ControlMode) dispatchLoop(ctx context.Context, r io.Reader) {
	todo() // TODO(S-3.01a): implement event dispatch; drive c.downstream.Tick()
}

// todo is a package-local helper that panics with a "not implemented" message.
// Its sole purpose is to satisfy the Red Gate discipline (BC-5.38.001): every
// non-trivial stub body calls todo() so that tests fail immediately rather than
// returning silent zero values.
//
// BC-5.38.005 self-check: "If I include this real implementation, will the test
// for this function pass trivially without any implementer work?" — yes for every
// function above; all use todo().
func todo() {
	panic("not implemented") //nolint:forbidigo // Red Gate stub — implementer replaces with real body
}
