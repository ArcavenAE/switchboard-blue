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
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

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

// execFunc is the function signature for spawning the tmux subprocess.
// Replacing it via WithExecFunc allows hermetic unit tests to inject a fake
// control mode stream (BC-5.38.001; test hermetic constraint).
type execFunc func(ctx context.Context) (io.ReadCloser, error)

// ControlMode manages a single tmux control mode connection and bridges
// control mode events to the session publisher and downstream half-channel.
//
// The zero value is not usable; construct with New.
//
// Concurrency: ControlMode spawns exactly one internal goroutine (the event
// dispatch loop) when Connect succeeds. All exported methods are safe to call
// from any goroutine.
type ControlMode struct {
	publisher  *session.Publisher
	downstream *halfchannel.HalfChannel
	// errCh is buffered with 1 so dispatchLoop can always send without blocking
	// on Close. One error is the maximum meaningful signal (dropped = fatal).
	errCh  chan error
	cancel context.CancelFunc
	proc   io.ReadCloser
	// execFn spawns the tmux subprocess; replaced in tests via WithExecFunc.
	execFn execFunc
}

// Option is a functional option for New.
type Option func(*ControlMode)

// WithExecFunc replaces the default os/exec-based tmux launcher with fn.
// This is the injection point for hermetic unit tests (BC-5.38.001; test
// hermetic constraint: MUST NOT shell out to real tmux).
func WithExecFunc(fn func(ctx context.Context) (io.ReadCloser, error)) Option {
	return func(c *ControlMode) {
		c.execFn = fn
	}
}

// defaultExecFn is the production exec function that launches `tmux -C`.
func defaultExecFn(ctx context.Context) (io.ReadCloser, error) {
	path, err := exec.LookPath("tmux")
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrControlModeUnavailable, err)
	}

	cmd := exec.CommandContext(ctx, path, "-C")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("tmux: stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrControlModeUnavailable, err)
	}

	return stdout, nil
}

// New constructs a ControlMode that publishes sessions via publisher and
// delivers output frames to downstream (BC-2.04.001; S-3.01a task 5+6).
//
// publisher and downstream must not be nil.
func New(publisher *session.Publisher, downstream *halfchannel.HalfChannel, opts ...Option) *ControlMode {
	c := &ControlMode{
		publisher:  publisher,
		downstream: downstream,
		// Buffer 1 so dispatchLoop can always deliver the drop signal without
		// blocking, even if the caller has not yet called Err().
		errCh:  make(chan error, 1),
		execFn: defaultExecFn,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
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
	innerCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	r, err := c.execFn(innerCtx)
	if err != nil {
		cancel()
		if errors.Is(err, ErrControlModeUnavailable) {
			return err
		}
		return fmt.Errorf("tmux: connect: %w", err)
	}

	c.proc = r

	go c.dispatchLoop(innerCtx, r)

	return nil
}

// Sessions returns a snapshot of all currently published tmux sessions
// (BC-2.04.001 PC-2; VP-031).
//
// The slice is a value copy — callers may freely mutate it.
func (c *ControlMode) Sessions() []session.Info {
	return c.publisher.ListSessions()
}

// Err returns the error channel that receives a non-nil error when the
// control mode connection is lost (BC-2.04.001 EC-002; S-3.01b API surface).
//
// The channel is closed and sends ErrControlModeDropped when the event loop
// exits unexpectedly. S-3.01b reads this channel to trigger PTY fallback.
//
// The channel is never written to by the caller.
func (c *ControlMode) Err() <-chan error {
	return c.errCh
}

// Close shuts down the event loop and terminates the tmux subprocess.
// It is safe to call Close multiple times; subsequent calls are no-ops.
func (c *ControlMode) Close() error {
	if c.cancel != nil {
		c.cancel()
	}

	if c.proc != nil {
		_ = c.proc.Close()
	}

	return nil
}

// dispatchLoop is the internal goroutine started by Connect. It reads control
// mode protocol lines from the tmux subprocess stdout, dispatches them to the
// appropriate handler, and drives the downstream half-channel Tick loop.
//
// Protocol events handled (BC-2.04.001 trigger + PC-3, PC-4, PC-5):
//
//   - %begin / %end         — command response delimiters
//   - %sessions-changed     — re-enumerate sessions
//   - %session-created name — AC-003 (PC-3)
//   - %session-closed name  — AC-003 (PC-4)
//   - %output pane data     — AC-004 (PC-5): feed downstream half-channel
//
// dispatchLoop is unexported; it is started only by Connect and exits when the
// subprocess exits or ctx is cancelled.
func (c *ControlMode) dispatchLoop(ctx context.Context, r io.Reader) {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		c.handleLine(line)
	}

	// Scanner stopped — either EOF (process exited) or ctx cancelled.
	select {
	case <-ctx.Done():
		// Normal shutdown via Close; don't signal dropped connection.
	default:
		// Unexpected exit — signal dropped connection to S-3.01b.
		select {
		case c.errCh <- ErrControlModeDropped:
		default:
			// Channel already has an error; don't block.
		}
	}
}

// handleLine processes a single control mode protocol line.
func (c *ControlMode) handleLine(line string) {
	switch {
	case strings.HasPrefix(line, "%session-created "):
		name := strings.TrimPrefix(line, "%session-created ")
		name = strings.TrimSpace(name)
		if name != "" {
			// Ignore ErrSessionAlreadyPublished — idempotent on re-connect.
			_ = c.publisher.Publish(name)
		}

	case strings.HasPrefix(line, "%session-closed "):
		name := strings.TrimPrefix(line, "%session-closed ")
		name = strings.TrimSpace(name)
		if name != "" {
			// Ignore ErrSessionNotFound — idempotent.
			_ = c.publisher.Unpublish(name)
		}

	case strings.HasPrefix(line, "%output "):
		// %output <pane-id> <data>
		// Feed the raw data bytes into the downstream half-channel.
		rest := strings.TrimPrefix(line, "%output ")
		// rest = "<pane-id> <data>"; skip pane-id, take data.
		parts := strings.SplitN(rest, " ", 2)
		if len(parts) == 2 && len(parts[1]) > 0 {
			_ = c.downstream.Enqueue([]byte(parts[1]))
			c.downstream.Tick()
		}
	}
}
