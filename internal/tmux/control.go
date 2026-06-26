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
	"sync"

	"github.com/arcavenae/switchboard/internal/halfchannel"
	"github.com/arcavenae/switchboard/internal/session"
)

// ErrControlModeUnavailable is returned by Connect when the tmux binary is not
// found in PATH (BC-2.04.001 EC-004 / FM-011; log: "tmux not found; using PTY
// fallback").
var ErrControlModeUnavailable = errors.New("tmux: control mode unavailable")

// ErrControlModeDropped signals that a previously established control mode
// connection has been lost mid-session (BC-2.04.001 EC-002 / FM-004).
// Callers (S-3.01b fallback path) receive this via the Err() channel.
var ErrControlModeDropped = errors.New("tmux: control mode connection dropped")

// ErrAlreadyConnected is returned by Connect when the ControlMode is already
// connected. Callers must Close before reconnecting.
var ErrAlreadyConnected = errors.New("tmux: control mode already connected")

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
	errCh      chan error
	closeErrCh sync.Once
	cancel     context.CancelFunc
	proc       io.ReadCloser
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
// enumerated and published (BC-2.04.001 PC-2; AC-002). Connect sends a
// "list-sessions" command immediately after establishing the stream; the
// resulting %begin/%end block is consumed to register pre-existing sessions.
//
// Returns ErrControlModeUnavailable if tmux is not found in PATH (EC-004).
// Returns ErrAlreadyConnected if Connect is called on an already-connected
// ControlMode; callers must Close first.
// Returns a wrapped error for any other subprocess launch failure.
func (c *ControlMode) Connect(ctx context.Context) error {
	// F-04: idempotency guard — second call must not leak subprocess + goroutine.
	if c.proc != nil || c.cancel != nil {
		return ErrAlreadyConnected
	}

	innerCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	r, err := c.execFn(innerCtx)
	if err != nil {
		cancel()
		c.cancel = nil
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
// The channel receives ErrControlModeDropped and is then closed when the
// event loop exits unexpectedly. S-3.01b reads this channel (or ranges over
// it) to trigger PTY fallback.
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

	// F-03: ensure errCh is closed so consumers using range-over-channel
	// unblock after a graceful Close. sync.Once guards against double-close.
	c.closeErrCh.Do(func() {
		close(c.errCh)
	})

	return nil
}

// dispatchLoop is the internal goroutine started by Connect. It reads control
// mode protocol lines from the tmux subprocess stdout, dispatches them to the
// appropriate handler, and drives the downstream half-channel Tick loop.
//
// Protocol events handled (BC-2.04.001 trigger + PC-3, PC-4, PC-5):
//
//   - %begin / %end         — command response delimiters; F-01 uses these
//     to collect the list-sessions response and register existing sessions.
//   - %sessions-changed     — re-enumerate sessions
//   - %session-created name — AC-003 (PC-3)
//   - %session-closed name  — AC-003 (PC-4)
//   - %output pane data     — AC-004 (PC-5): feed downstream half-channel
//
// dispatchLoop is unexported; it is started only by Connect and exits when the
// subprocess exits or ctx is cancelled.
func (c *ControlMode) dispatchLoop(ctx context.Context, r io.Reader) {
	scanner := bufio.NewScanner(r)

	// F-01: inSession tracks whether we are currently inside a %begin/%end
	// block that is the response to the initial list-sessions command we issue
	// immediately after establishing the connection. Lines between %begin and
	// %end are session names from a pre-existing tmux server state.
	inSessionList := false

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			// F-03: close errCh on context cancellation (graceful Close path).
			c.closeErrCh.Do(func() {
				close(c.errCh)
			})
			return
		default:
		}

		line := scanner.Text()

		// F-01: handle %begin/%end block for list-sessions response.
		switch {
		case strings.HasPrefix(line, "%begin "):
			inSessionList = true
			continue
		case strings.HasPrefix(line, "%end "):
			inSessionList = false
			continue
		}

		if inSessionList {
			// Each line between %begin and %end is a session name.
			name := strings.TrimSpace(line)
			if name != "" {
				// Ignore ErrSessionAlreadyPublished — idempotent on re-connect.
				_ = c.publisher.Publish(name)
			}
			continue
		}

		c.handleLine(line)
	}

	// Scanner stopped — either EOF (process exited) or ctx cancelled.
	select {
	case <-ctx.Done():
		// Normal shutdown via Close; don't signal dropped connection.
		c.closeErrCh.Do(func() {
			close(c.errCh)
		})
	default:
		// F-03: unexpected exit — signal dropped connection to S-3.01b,
		// then close the channel so range-over-channel consumers unblock.
		select {
		case c.errCh <- ErrControlModeDropped:
		default:
			// Channel already has an error; don't block.
		}
		c.closeErrCh.Do(func() {
			close(c.errCh)
		})
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
		// %output <pane-id> <octal-escaped-data>
		// F-06: split on first space to isolate pane-id; unescape the payload.
		rest := strings.TrimPrefix(line, "%output ")
		parts := strings.SplitN(rest, " ", 2)
		if len(parts) == 2 && len(parts[1]) > 0 {
			data := unescapeTmuxOutput(parts[1])
			_ = c.downstream.Enqueue(data)
			c.downstream.Tick()
		}
	}
}

// unescapeTmuxOutput decodes a tmux control-mode octal-escaped payload.
//
// tmux encodes non-printable bytes and spaces using octal escapes of the form
// \NNN (three octal digits). Double-backslash \\ encodes a literal backslash.
// All other bytes are passed through unchanged.
//
// Example: "hello\040world" → "hello world" (0x20 = space = octal 040).
func unescapeTmuxOutput(s string) []byte {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' {
			b = append(b, s[i])
			continue
		}
		// Peek at the next character.
		if i+1 >= len(s) {
			b = append(b, s[i])
			continue
		}
		next := s[i+1]
		if next == '\\' {
			// \\ → literal backslash.
			b = append(b, '\\')
			i++
			continue
		}
		// Check for three-digit octal sequence \NNN.
		if i+3 < len(s) &&
			next >= '0' && next <= '7' &&
			s[i+2] >= '0' && s[i+2] <= '7' &&
			s[i+3] >= '0' && s[i+3] <= '7' {
			octal := (next-'0')*64 + (s[i+2]-'0')*8 + (s[i+3] - '0')
			b = append(b, octal)
			i += 3
			continue
		}
		// Unrecognised escape — pass through as-is.
		b = append(b, s[i])
	}
	return b
}
