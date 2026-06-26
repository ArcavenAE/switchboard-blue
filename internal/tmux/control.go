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
	"bytes"
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

// ErrAlreadyConnected is returned by Connect if the ControlMode has already
// been connected. ControlMode instances are SINGLE-USE: after Close (or after
// dispatchLoop has exited on stream EOF), the instance is terminal. To
// reconnect, construct a new ControlMode via tmux.New(...).
var ErrAlreadyConnected = errors.New("tmux: control mode already connected")

// ErrControlModeClosed is returned by Connect if the ControlMode has been
// Closed. ControlMode is single-use; reconnect requires a new instance via
// tmux.New(...). M-2 (pass-12): enforces the documented SINGLE-USE contract.
var ErrControlModeClosed = errors.New("tmux: control mode is closed (single-use)")

// ErrControlModeUnsupportedFlag is returned by Connect if tmux rejects the
// -C / -CC flag (older tmux versions that do not support control mode).
// TODO(S-3.01b H-003): detection requires stderr capture during subprocess
// launch; deferred to a future story. The sentinel is defined now so that
// callers can use errors.Is once detection is wired in.
var ErrControlModeUnsupportedFlag = errors.New("tmux: control mode flag not supported")

// ErrControlModeBinaryNotFound is returned by Connect if the tmux binary
// is not found in PATH. Wraps the underlying exec.LookPath error via %w.
var ErrControlModeBinaryNotFound = errors.New("tmux: binary not found in PATH")

// execFunc is the function signature for spawning the tmux subprocess.
// Replacing it via WithExecFunc allows hermetic unit tests to inject a fake
// control mode stream. Hermetic test injection point: tests use WithExecFunc
// to substitute the process spawn; production callers use the default
// exec.LookPath("tmux") + exec.CommandContext path.
//
// H-03: returns both stdin (for writing commands) and stdout (for reading
// events), so Connect can actually send commands like list-sessions. Tests
// that use fakeExecFunc must adapt to this signature.
//
// M-002/L-004 (pass-4): classifyCh delivers exactly one classification result
// after the subprocess exits — either ErrControlModeUnsupportedFlag (flag
// rejection detected in stderr) or nil (clean exit or unrecognized stderr).
// The channel is buffered(1) and closed after the result is sent, so callers
// can drain it without blocking. Connect forwards a non-nil classification to
// the Err() channel so callers can distinguish ErrControlModeUnsupportedFlag
// from ErrControlModeDropped.
// Test fakes that do not care about classification may return a closed nil channel.
type execFunc func(ctx context.Context) (stdin io.WriteCloser, stdout io.ReadCloser, classifyCh <-chan error, err error)

// ControlMode manages a single tmux control mode connection and bridges
// control mode events to the session publisher and downstream half-channel.
//
// The zero value is not usable; construct with New.
//
// Concurrency: ControlMode spawns exactly one internal goroutine (the event
// dispatch loop) when Connect succeeds. All exported methods are safe to call
// from any goroutine.
//
// mu protects all lifecycle fields (proc, stdin, cancel). Channel fields
// (errCh, closeErrCh, frames, closeFrames) are safe by their own Go
// concurrency contracts.
type ControlMode struct {
	publisher  *session.Publisher
	downstream *halfchannel.HalfChannel
	// errCh is buffered with 1 so dispatchLoop can always send without blocking
	// on Close. One error is the maximum meaningful signal (dropped = fatal).
	errCh      chan error
	closeErrCh sync.Once

	// frames is the output stream of ChannelFrames produced by the dispatch
	// loop (F-PASS7-H-001, pass-7). Each %output event yields one or more
	// frames via M-1 fragmentation. Callers drain via Frames(). Buffered to
	// absorb burst latency; on overflow, frames are dropped with a non-blocking
	// select (backpressure protection — dispatchLoop must not stall on a slow
	// consumer). The channel is closed when dispatchLoop exits.
	frames      chan halfchannel.ChannelFrame
	closeFrames sync.Once // guards close(frames) against double-close

	// mu protects lifecycle fields below (proc, stdin, cancel, closed).
	mu     sync.Mutex
	cancel context.CancelFunc
	proc   io.ReadCloser
	stdin  io.WriteCloser
	closed bool // M-2 (pass-12): true after Close; Connect rejects with ErrControlModeClosed

	// wg joins dispatchLoop on Close to provide a hard lifecycle boundary.
	// After Close returns, dispatchLoop has exited and c.publisher /
	// c.downstream are no longer accessed by ControlMode (M-2, pass-5).
	wg sync.WaitGroup

	// execFn spawns the tmux subprocess; replaced in tests via WithExecFunc.
	execFn execFunc
}

// Option is a functional option for New.
type Option func(*ControlMode)

// WithExecFunc replaces the default os/exec-based tmux launcher with fn.
// Hermetic test injection point: tests use this to substitute the process
// spawn without shelling out to real tmux.
//
// H-03: fn now returns (stdin, stdout, classifyCh, err) so Connect can write
// commands and monitor post-start classification. Test fakes that do not
// exercise classification must return a pre-closed nil channel for classifyCh.
func WithExecFunc(fn func(ctx context.Context) (io.WriteCloser, io.ReadCloser, <-chan error, error)) Option {
	return func(c *ControlMode) {
		c.execFn = fn
	}
}

// defaultExecFn is the production exec function that launches `tmux -C`.
//
// H-01: reaps the subprocess in a goroutine via cmd.Wait() to avoid zombies.
// H-03: returns both stdin and stdout so Connect can write commands.
// M-002/L-004 (pass-4): adds sync.WaitGroup to synchronize the stderr drain
// goroutine with the reaper, ensuring the full stderr capture is available for
// ClassifyStderr before classifyCh is written. Returns classifyCh (buffered 1)
// that receives ErrControlModeUnsupportedFlag on flag-rejection, or nil on clean
// exit, then is closed. Connect forwards non-nil classification to Err().
func defaultExecFn(ctx context.Context) (io.WriteCloser, io.ReadCloser, <-chan error, error) {
	path, err := exec.LookPath("tmux")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%w: %w: %w", ErrControlModeUnavailable, ErrControlModeBinaryNotFound, err)
	}

	cmd := exec.CommandContext(ctx, path, "-C")

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("tmux: stdin pipe: %w", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdinPipe.Close()
		return nil, nil, nil, fmt.Errorf("tmux: stdout pipe: %w", err)
	}

	// M-001 (pass-3): capture stderr to detect -C flag rejection.
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		_ = stdinPipe.Close()
		_ = stdoutPipe.Close()
		return nil, nil, nil, fmt.Errorf("tmux: stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = stdinPipe.Close()
		_ = stdoutPipe.Close()
		_ = stderrPipe.Close()
		return nil, nil, nil, fmt.Errorf("%w: %w", ErrControlModeUnavailable, err)
	}

	// M-002 (pass-4): synchronize the stderr drain goroutine with the reaper
	// via sync.WaitGroup so that stderrBuf.String() is safe to read in the
	// reaper only after all bytes have been copied from stderrPipe.
	// Without this, cmd.Wait() returns before io.Copy finishes (StderrPipe
	// transfers responsibility for draining to the caller; Wait only closes
	// the write-end of the pipe, it does not join the caller's drain goroutine).
	var stderrBuf bytes.Buffer
	var drainWG sync.WaitGroup
	drainWG.Add(1)
	go func() {
		defer drainWG.Done()
		_, _ = io.Copy(&stderrBuf, stderrPipe)
	}()

	// classifyCh delivers the post-exit classification to Connect.
	// Buffered 1 so the reaper goroutine never blocks on a non-draining caller.
	classifyCh := make(chan error, 1)

	// H-01: reap the subprocess; classify stderr on abnormal exit.
	go func() {
		waitErr := cmd.Wait()
		// M-002 (pass-4): wait for the drain goroutine to complete before reading
		// stderrBuf. This is the critical synchronization point.
		drainWG.Wait()
		if waitErr != nil {
			captured := stderrBuf.String()
			// L-004 (pass-4): emit ErrControlModeUnsupportedFlag via classifyCh
			// so Connect can forward it to the Err() channel. Non-nil classification
			// supersedes ErrControlModeDropped from the dispatchLoop EOF path.
			if classified := ClassifyStderr(captured); classified != nil {
				classifyCh <- classified
			}
		}
		close(classifyCh)
	}()

	return stdinPipe, stdoutPipe, classifyCh, nil
}

// framesBufferSize is the capacity of the frames channel. Large enough to
// absorb tmux burst output (BC-2.04.001 PC-5) without backpressure stalling
// dispatchLoop; small enough that memory overhead is negligible.
const framesBufferSize = 256

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
		errCh: make(chan error, 1),
		// F-PASS7-H-001 (pass-7): buffered output stream; drained via Frames().
		frames: make(chan halfchannel.ChannelFrame, framesBufferSize),
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
// Returns ErrControlModeClosed if the instance has been Closed — ControlMode
// is single-use; reconnect requires a new instance via tmux.New(...).
// Returns a wrapped error for any other subprocess launch failure.
func (c *ControlMode) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// M-2 (pass-12): single-use guard — reject Connect after Close.
	if c.closed {
		return ErrControlModeClosed
	}

	// F-04: idempotency guard — second call must not leak subprocess + goroutine.
	if c.proc != nil || c.cancel != nil {
		return ErrAlreadyConnected
	}

	innerCtx, cancel := context.WithCancel(ctx)

	stdinPipe, stdoutPipe, classifyCh, err := c.execFn(innerCtx)
	if err != nil {
		cancel()
		if errors.Is(err, ErrControlModeUnavailable) {
			return err
		}
		return fmt.Errorf("tmux: connect: %w", err)
	}

	c.cancel = cancel
	c.proc = stdoutPipe
	c.stdin = stdinPipe

	// H-03: send the list-sessions command so tmux emits a %begin/%end block
	// containing pre-existing session names (BC-2.04.001 PC-2; F-01 fix).
	// -F #{session_name} formats each session as its name only (one per line).
	if _, err := io.WriteString(stdinPipe, "list-sessions -F #{session_name}\n"); err != nil {
		_ = stdinPipe.Close()
		_ = stdoutPipe.Close()
		cancel()
		c.cancel = nil
		c.proc = nil
		c.stdin = nil
		return fmt.Errorf("tmux: list-sessions write: %w", err)
	}

	// M-2 (pass-5): register goroutine with wg before spawning so that
	// Close can call wg.Wait() to guarantee dispatchLoop has exited before
	// returning. This provides the happens-before guarantee for all accesses
	// to c.publisher and c.downstream from the goroutine.
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.dispatchLoop(innerCtx, stdoutPipe)
	}()

	// L-004 (pass-4): if execFn returns a classification channel, monitor it
	// in a goroutine. A non-nil classification (ErrControlModeUnsupportedFlag)
	// supersedes the ErrControlModeDropped that dispatchLoop emits on EOF —
	// forward it to errCh so callers get the more specific sentinel.
	if classifyCh != nil {
		go func() {
			if classified, ok := <-classifyCh; ok && classified != nil {
				c.closeErrCh.Do(func() {
					select {
					case c.errCh <- classified:
					default:
					}
					close(c.errCh)
				})
			}
		}()
	}

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

// Frames returns the read-only channel of ChannelFrames produced by the
// control mode dispatch loop (F-PASS7-H-001, pass-7). Each %output event
// yields one or more frames via M-1 fragmentation. The channel is buffered;
// if the consumer falls behind, frames are dropped (backpressure protection —
// dispatchLoop must not stall on a slow drain). The channel is closed when
// dispatchLoop exits (via Close or unexpected EOF).
//
// Callers should drain Frames() concurrently with reading Err() to receive
// both data and lifecycle signals.
func (c *ControlMode) Frames() <-chan halfchannel.ChannelFrame {
	return c.frames
}

// Close cancels the control mode connection and waits for the dispatch loop
// to exit. After Close returns, the dispatch goroutine has exited and the
// publisher/downstream are no longer accessed by ControlMode.
// Close is idempotent; multiple calls are no-ops after the first.
func (c *ControlMode) Close() error {
	c.mu.Lock()

	cancel := c.cancel
	stdin := c.stdin
	proc := c.proc
	c.cancel = nil
	c.stdin = nil
	c.proc = nil
	c.closed = true // M-2 (pass-12): enforce single-use; Connect rejects after Close

	c.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	if stdin != nil {
		_ = stdin.Close()
	}

	// M-2 (pass-5): block until dispatchLoop has fully exited before closing
	// proc. This is important: closing proc (the stdout ReadCloser) would cause
	// scanner.Scan to return immediately with an error, racing with the natural
	// exit path. We want dispatchLoop to observe ctx cancellation and exit
	// naturally, then we clean up. The mutex is released above so dispatchLoop
	// can complete without deadlock.
	c.wg.Wait()

	if proc != nil {
		_ = proc.Close()
	}

	// F-03: ensure errCh is closed so consumers using range-over-channel
	// unblock after a graceful Close. sync.Once guards against double-close.
	c.closeErrCh.Do(func() {
		close(c.errCh)
	})

	// F-PASS7-H-001 (pass-7): close frames so Frames() consumers unblock.
	c.closeFrames.Do(func() {
		close(c.frames)
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
//   - %session-created name — AC-003 (PC-3)
//   - %session-closed name  — AC-003 (PC-4)
//   - %output pane data     — AC-004 (PC-5): feed downstream half-channel
//
// Not handled here (future work):
//   - %sessions-changed — would trigger full re-enumeration; deferred to a
//     future story (currently we rely on per-event create/closed signals
//     which suffice for BC-2.04.001 PC-3/PC-4).
//
// dispatchLoop is unexported; it is started only by Connect and exits when the
// subprocess exits or ctx is cancelled.
func (c *ControlMode) dispatchLoop(ctx context.Context, r io.Reader) {
	// M-1 (pass-12): close frames on every exit path (normal return, ctx
	// cancellation, %error, panic). sync.Once guards against double-close with
	// Close(). Without this defer, the inline ctx.Done() arm returned without
	// closing frames, causing range-consumers to block forever.
	defer c.closeFrames.Do(func() { close(c.frames) })

	scanner := bufio.NewScanner(r)
	// H-1 (pass-5): raise scanner token buffer to 2 MiB so that large %output
	// lines (e.g. a terminal repaint of a 100 KiB scrollback) are not silently
	// truncated by the default 64 KiB limit (bufio.ErrTooLong causes scanner.Scan
	// to return false, which dispatchLoop previously misread as an unexpected drop).
	const maxTmuxOutputLine = 2 * 1024 * 1024 // 2 MiB
	scanner.Buffer(make([]byte, 0, 64*1024), maxTmuxOutputLine)

	// F-01: inSession tracks whether we are currently inside a %begin/%end
	// block that is the response to the initial list-sessions command we issue
	// immediately after establishing the connection. Lines between %begin and
	// %end are session names from a pre-existing tmux server state.
	inSessionList := false

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			// H-02: close errCh on context cancellation (graceful Close path).
			// Use sync.Once atomically — both send decision and close happen
			// inside the same Do invocation to prevent send-on-closed-channel.
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
		case strings.HasPrefix(line, "%error "):
			// tmux command failure (e.g., list-sessions failed; server starting;
			// permissions). BC-2.04.001 EC-001: control-mode init failure →
			// ErrControlModeDropped → PTY fallback (ADR-010).
			// inSessionList reset is implicit — we return immediately below.
			c.closeErrCh.Do(func() {
				select {
				case c.errCh <- ErrControlModeDropped:
				default:
					// Channel already has an error; don't block.
				}
				close(c.errCh)
			})
			// frames closed via defer at top of dispatchLoop.
			return
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
	// H-1 (pass-5): consume scanner.Err() for debuggability. ErrTooLong would
	// mean a line exceeded the 2 MiB buffer; other errors indicate I/O failure.
	// We don't currently log (no logger injected), but the read prevents the
	// result from being silently discarded and documents intent for future use.
	_ = scanner.Err()
	select {
	case <-ctx.Done():
		// Normal shutdown via Close; don't signal dropped connection.
		// H-02: sync.Once guards close; no send on graceful exit.
		c.closeErrCh.Do(func() {
			close(c.errCh)
		})
	default:
		// F-03: unexpected exit — signal dropped connection to S-3.01b,
		// then close the channel so range-over-channel consumers unblock.
		//
		// H-02: perform the send and close atomically inside sync.Once to
		// prevent a concurrent Close() from closing the channel while we
		// attempt to send, which would panic. If Close() wins the Once,
		// this Do is a no-op — the send is skipped entirely (acceptable:
		// Close winning means the shutdown was not truly unexpected).
		c.closeErrCh.Do(func() {
			select {
			case c.errCh <- ErrControlModeDropped:
			default:
				// Channel already has an error; don't block.
			}
			close(c.errCh)
		})
	}

	// frames closed via defer at top of dispatchLoop (M-1, pass-12).
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
			if len(data) == 0 {
				return
			}
			// M-1 (pass-5): fragment payloads larger than MaxPayloadSize to
			// preserve BC-2.04.001 PC-5. Enqueue rejects payloads above
			// halfchannel.MaxPayloadSize (65515 bytes); split into chunks so
			// that arbitrarily large terminal output reaches the downstream.
			for i := 0; i < len(data); i += halfchannel.MaxPayloadSize {
				end := i + halfchannel.MaxPayloadSize
				if end > len(data) {
					end = len(data)
				}
				if err := c.downstream.Enqueue(data[i:end]); err != nil {
					// Chunk is within MaxPayloadSize after fragmentation;
					// an error here is unexpected. Break to preserve partial
					// delivery semantics rather than silently dropping.
					break
				}
				frame := c.downstream.Tick()
				// F-PASS7-H-001 (pass-7): publish the dequeued frame to the
				// Frames channel. Non-blocking: if the consumer is slow, drop
				// the frame rather than stalling dispatchLoop (backpressure
				// would block tmux output processing).
				// TODO(phase-6): structured logging on drop.
				select {
				case c.frames <- frame:
				default:
					// Frame dropped due to downstream backpressure.
				}
			}
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
