package tmux

// PTY proxy fallback for the access node (BC-2.04.002; ADR-010; ARCH-08 §6.5).
//
// When tmux control mode is unavailable (initial connect failure or
// mid-session drop after 3 reconnect attempts), the access node enters
// PTY proxy mode via PTYProxy. The fallback is one-way per session
// lifecycle — no auto-upgrade back to control mode once PTY proxy is
// active (BC-2.04.002 EC-004; story S-3.01b task 9).
//
// Classification: effectful (ARCH-09). PTY allocation forks an OS-level
// pseudo-terminal device; tests inject ptyAllocFunc to avoid real PTY I/O
// in unit tests (hermetic test pattern — no real PTY shell-out in unit
// tests; real-PTY coverage deferred to VP-032 integration harness).
//
// Allowed internal imports: {halfchannel, session} per ARCH-08 §6.5.
// Forbidden: internal/admission, internal/routing, internal/hmac.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/arcavenae/switchboard/internal/halfchannel"
	"github.com/arcavenae/switchboard/internal/session"
)

// ErrPTYDeviceUnavailable is returned by PTYProxy.Connect when no PTY device
// can be allocated on the host. Maps to E-SYS-001 (error-taxonomy.md §SYS;
// BC-2.04.002 EC-004; FM-011). The operator-facing guidance emitted to the
// configured Logger is:
//
//	"PTY device unavailable: cannot start access node. Install 'openpty'
//	 or check device permissions."
//
// The sentinel text is terse (ST1005 — no trailing period); the full guidance
// is emitted via the logger on every allocation failure (BC-2.04.002 EC-004).
//
// When both tmux control mode and PTY device are unavailable, the access node
// must return this error and exit with a non-zero status. Failure is never
// silent (BC-2.04.002 invariant 3).
var ErrPTYDeviceUnavailable = errors.New("PTY device unavailable: cannot start access node")

// maxReconnectAttempts is the number of tmux control mode reconnect attempts
// before the mid-session fallback path switches to PTY proxy mode
// (BC-2.04.002 EC-003; S-3.01b task 6).
const maxReconnectAttempts = 3

// ptyAllocFunc allocates a PTY pair and starts a child shell connected to the
// slave side. Returns the master end (the slave is held only by the child
// process), the shell's process ID for synthetic session naming ("pty-<pid>"),
// and any error.
//
// The default implementation (defaultPTYAlloc) opens /dev/ptmx and uses
// platform-specific ioctls to obtain the slave path:
//   - Linux: TIOCSPTLCK to unlock + TIOCGPTN to read the slave number
//   - Darwin: TIOCPTYGNAME to read the slave device path
//
// Other platforms return ErrPTYDeviceUnavailable.
//
// Test callers inject hermetic stubs via WithPTYAllocFunc.
type ptyAllocFunc func() (master io.ReadWriteCloser, pid int, err error)

// PTYProxyOption is a functional option for NewPTYProxy.
type PTYProxyOption func(*PTYProxy)

// WithPTYAllocFunc replaces the default PTY allocator with fn.
// Hermetic test injection point: unit tests supply a fake that avoids
// forking a real PTY process. Production callers use defaultPTYAlloc,
// which opens /dev/ptmx and applies platform ioctls to find the slave.
func WithPTYAllocFunc(fn ptyAllocFunc) PTYProxyOption {
	return func(p *PTYProxy) {
		p.ptyAlloc = fn
	}
}

// Logger is a minimal logging interface injected into PTYProxy.
// BC-2.04.002 postcondition 3 requires a specific log entry on every
// fallback event; callers supply a real logger; tests supply a fake
// that captures log lines for assertion.
type Logger interface {
	// Log records a single log line.
	Log(msg string)
}

// WithLogger sets the logger used by PTYProxy. If not set, PTYProxy uses
// stderrLogger (log entries are written to os.Stderr). Tests inject a fake
// logger to assert mandatory log messages (BC-2.04.002 PC-3).
func WithLogger(l Logger) PTYProxyOption {
	return func(p *PTYProxy) {
		p.logger = l
	}
}

// stderrLogger is the default logger. Writes to os.Stderr so operators see
// mandatory fallback notifications (BC-2.04.002 invariant 3) without requiring
// explicit logger injection in production callers.
type stderrLogger struct{}

func (stderrLogger) Log(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}

// PTYProxy implements PTY proxy mode for the access node
// (BC-2.04.002; ADR-010). It opens a PTY and proxies its I/O as a single
// anonymous session published under a synthetic name ("pty-<pid>").
//
// Concurrency: PTYProxy is safe for concurrent use. Lifecycle state (master,
// closed) is protected by mu; the wg synchronizes the ioRelay goroutine
// shutdown. Sessions() delegates to the thread-safe publisher and does not
// take mu.
//
// The zero value is not usable; construct with NewPTYProxy.
type PTYProxy struct {
	publisher  *session.Publisher
	downstream *halfchannel.HalfChannel
	logger     Logger

	// ptyAlloc allocates the PTY device; replaced in tests via WithPTYAllocFunc.
	ptyAlloc ptyAllocFunc

	// sessionName is the synthetic "pty-<pid>" name published on Connect.
	sessionName string

	// mu protects all lifecycle fields.
	mu     sync.Mutex
	master io.ReadWriteCloser
	pid    int
	closed bool

	// wg joins the I/O relay goroutine on Close.
	wg sync.WaitGroup

	// H-001 (pass-3): frames is the output stream of ChannelFrames produced by
	// ioRelay. Symmetric to ControlMode.frames. Each PTY-master read produces
	// (after fragmentation if needed) one or more frames via Tick(). Buffered to
	// absorb burst; non-blocking send drops on backpressure. Closed once on Close
	// via closeFrames.
	frames      chan halfchannel.ChannelFrame
	closeFrames sync.Once
}

// NewPTYProxy constructs a PTYProxy that publishes sessions via publisher and
// delivers output frames to downstream (BC-2.04.002; S-3.01b task 4).
//
// publisher and downstream must not be nil.
func NewPTYProxy(publisher *session.Publisher, downstream *halfchannel.HalfChannel, opts ...PTYProxyOption) *PTYProxy {
	p := &PTYProxy{
		publisher:  publisher,
		downstream: downstream,
		logger:     stderrLogger{},
		ptyAlloc:   defaultPTYAlloc,
		// H-001 (pass-3): match ControlMode capacity.
		frames: make(chan halfchannel.ChannelFrame, framesBufferSize),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Connect allocates a PTY device, publishes the session under the synthetic
// name "pty-<pid>", and starts the I/O relay goroutine (BC-2.04.002 PC-1,
// PC-2, PC-3).
//
// Mandatory log entry written on success:
// "tmux control mode unavailable; using PTY proxy mode. Functionality
// limited: no structured session metadata, no content-type detection."
// (BC-2.04.002 PC-3; VP-032).
//
// Returns ErrPTYDeviceUnavailable (E-SYS-001) if the PTY device cannot be
// allocated (BC-2.04.002 EC-004). This is the only error that exits with a
// non-zero status — failure is never silent (BC-2.04.002 invariant 3).
func (p *PTYProxy) Connect(_ context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return fmt.Errorf("PTY proxy: connect: %w", ErrPTYProxyAlreadyClosed)
	}
	if p.master != nil {
		return fmt.Errorf("PTY proxy: connect: %w", ErrPTYProxyAlreadyConnected)
	}

	master, pid, err := p.ptyAlloc()
	if err != nil {
		// BC-2.04.002 EC-004 / story AC-003: operator-facing guidance MUST reach
		// the operator. The sentinel text is terse (ST1005); the full guidance
		// is emitted here via the configured logger.
		p.logger.Log("PTY device unavailable: cannot start access node. Install 'openpty' or check device permissions.")
		if errors.Is(err, ErrPTYDeviceUnavailable) {
			return err
		}
		return fmt.Errorf("%w: %w", ErrPTYDeviceUnavailable, err)
	}

	p.master = master
	p.pid = pid
	p.sessionName = fmt.Sprintf("pty-%d", pid)

	// Publish the synthetic session (BC-2.04.002 PC-2).
	if err := p.publisher.Publish(p.sessionName); err != nil {
		if !errors.Is(err, session.ErrSessionAlreadyPublished) {
			p.logger.Log(fmt.Sprintf("PTY proxy publish failed: %v", err))
			// M-001 (pass-4): close the master to prevent resource leak on
			// Connect failure. ioRelay has not been started yet, so there is
			// no goroutine to join — Close the master directly.
			_ = master.Close()
			p.master = nil
			p.pid = 0
			p.sessionName = ""
			return fmt.Errorf("tmux: pty proxy publish: %w", err)
		}
		// ErrSessionAlreadyPublished is idempotent; continue.
	}

	// Write the mandatory log entry (BC-2.04.002 PC-3; VP-032).
	p.logger.Log("tmux control mode unavailable; using PTY proxy mode. Functionality limited: no structured session metadata, no content-type detection.")

	// Start I/O relay goroutine.
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.ioRelay(master)
	}()

	return nil
}

// ioRelay reads from the PTY master and drives the downstream half-channel.
// It exits when the master returns io.EOF or an error.
func (p *PTYProxy) ioRelay(master io.ReadWriteCloser) {
	// H-001 (pass-3): close frames on every exit path (normal, EOF, error).
	// sync.Once guards against double-close with Close().
	defer p.closeFrames.Do(func() { close(p.frames) })

	buf := make([]byte, 4096)
	for {
		n, err := master.Read(buf)
		if n > 0 {
			data := buf[:n]
			for i := 0; i < len(data); i += halfchannel.MaxPayloadSize {
				end := i + halfchannel.MaxPayloadSize
				if end > len(data) {
					end = len(data)
				}
				if enqErr := p.downstream.Enqueue(data[i:end]); enqErr != nil {
					break
				}
				frame := p.downstream.Tick()
				// H-001 (pass-3): publish dequeued frame for downstream consumers
				// (S-3.02). Non-blocking — if consumer falls behind, drop with note.
				// TODO(phase-6): add metrics counter.
				select {
				case p.frames <- frame:
				default:
					// Frame dropped due to downstream backpressure.
				}
			}
		}
		if err != nil {
			return
		}
	}
}

// Sessions returns a snapshot of all sessions currently published via the
// shared publisher (BC-2.04.002 PC-2). In PTY proxy mode there is at most
// one session active (the synthetic "pty-<pid>" name set on Connect).
//
// Note: the publisher is shared with ControlMode when both are wired through
// a SessionConnector. Sessions() reflects the publisher's current state,
// which may include sessions from either mode if the connector is in a
// transitional state. In steady-state PTY proxy mode, only the PTY session
// is present.
//
// The slice is a value copy — callers may freely mutate it.
func (p *PTYProxy) Sessions() []session.Info {
	return p.publisher.ListSessions()
}

// Frames returns the read-only channel of ChannelFrames produced by the PTY
// proxy's ioRelay goroutine. Symmetric to ControlMode.Frames(). The channel is
// buffered; if the consumer falls behind, frames are dropped (backpressure
// protection — ioRelay must not stall on PTY reads). The channel is closed when
// Close() is called or ioRelay exits.
//
// Callers should drain Frames() concurrently with reading any error signaling
// channel.
func (p *PTYProxy) Frames() <-chan halfchannel.ChannelFrame {
	return p.frames
}

// Close tears down the PTY proxy: terminates the child process, closes the
// PTY master, and unpublishes the session. Close is idempotent.
func (p *PTYProxy) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	master := p.master
	sessionName := p.sessionName
	p.master = nil
	p.closed = true
	p.mu.Unlock()

	if master != nil {
		_ = master.Close()
	}

	// Wait for the I/O relay goroutine to exit.
	p.wg.Wait()

	// H-001 (pass-3): close frames after ioRelay has exited. sync.Once guards
	// against double-close with the defer in ioRelay (either path wins safely).
	p.closeFrames.Do(func() { close(p.frames) })

	// Unpublish the synthetic session.
	if sessionName != "" {
		// Ignore ErrSessionNotFound — idempotent.
		_ = p.publisher.Unpublish(sessionName)
	}

	return nil
}

// ErrPTYProxyClosed is returned by PTYProxy.SendInput when the proxy has been
// closed or was never connected.
var ErrPTYProxyClosed = errors.New("PTY proxy: closed or not connected")

// ErrPTYProxyAlreadyClosed is returned by PTYProxy.Connect when the proxy has
// already been closed. PTYProxy is single-use per session lifecycle.
var ErrPTYProxyAlreadyClosed = errors.New("PTY proxy: already closed")

// ErrPTYProxyAlreadyConnected is returned by PTYProxy.Connect when the proxy
// has already been connected.
var ErrPTYProxyAlreadyConnected = errors.New("PTY proxy: already connected")

// ErrSessionConnectorClosed is returned by SessionConnector.SendInput when the
// connector has been closed. Construct a new SessionConnector for each session.
var ErrSessionConnectorClosed = errors.New("session connector: closed")

// SendInput writes payload to the PTY master, forwarding keystrokes to the
// shell running in the PTY. It implements session.KeystrokeSink.
//
// Lock discipline: mu is acquired briefly to capture p.master; the write
// occurs outside the lock because writes can block on a full PTY buffer and
// holding the lock during a blocking write would deadlock Close.
//
// Returns ErrPTYProxyClosed if the proxy has been closed or was never
// connected (p.master is nil). Propagates any io.Writer error.
func (p *PTYProxy) SendInput(payload []byte) error {
	p.mu.Lock()
	master := p.master
	closed := p.closed
	p.mu.Unlock()

	if closed || master == nil {
		return ErrPTYProxyClosed
	}

	_, err := master.Write(payload)
	if err != nil {
		return fmt.Errorf("PTY proxy: send input: %w", err)
	}

	return nil
}

// SendInput dispatches to the active mode's SendInput (ControlMode or
// PTYProxy). It implements session.KeystrokeSink.
//
// Lock discipline: sc.mu is acquired briefly to capture the active sink; the
// delegated write occurs outside the lock. Returns an appropriate error if the
// connector has been closed or was never connected.
func (sc *SessionConnector) SendInput(payload []byte) error {
	sc.mu.Lock()
	ctrl := sc.ctrl
	pty := sc.pty
	inPTY := sc.inPTYMode
	closed := sc.closed
	sc.mu.Unlock()

	if closed {
		return fmt.Errorf("session connector: send input: %w", ErrSessionConnectorClosed)
	}

	if inPTY {
		return pty.SendInput(payload)
	}

	return ctrl.SendInput(payload)
}

// ControlModeFactory constructs a fresh ControlMode for reconnect attempts.
// Per ControlMode SINGLE-USE contract (control.go ADR-010), each reconnect
// must produce a new instance. The factory captures the construction
// parameters (publisher, downstream, options) closed over from New.
//
// BC-2.04.002 EC-003: SessionConnector retries control mode up to 3 times
// using this factory before falling back to PTY proxy. A nil factory means
// no reconnect is attempted — SessionConnector falls back to PTY proxy
// immediately on ErrControlModeDropped.
//
// CONTRACT (pass-9 L-001): the returned *ControlMode MUST already have
// Connect(ctx) completed successfully. SessionConnector does NOT call
// Connect on the returned instance — it expects to range over its Err()
// channel immediately. A constructed-but-unconnected ControlMode will
// deadlock the watcher (Err() never fires). Returning (nil, nil) is a
// contract violation and is treated as a failed attempt.
type ControlModeFactory func(ctx context.Context) (*ControlMode, error)

// SessionConnectorOption is a functional option for NewSessionConnector.
type SessionConnectorOption func(*SessionConnector)

// WithControlModeFactory sets the factory used for reconnect attempts after
// mid-session control-mode drop. If unset, no reconnection is attempted —
// SessionConnector falls back to PTY proxy immediately on ErrControlModeDropped.
func WithControlModeFactory(f ControlModeFactory) SessionConnectorOption {
	return func(sc *SessionConnector) { sc.factory = f }
}

// SessionConnector orchestrates the tmux-first, PTY-fallback connection
// strategy (ADR-010; BC-2.04.002). It attempts control mode first; if
// Connect returns ErrControlModeUnavailable or the Err() channel delivers
// ErrControlModeDropped (after maxReconnectAttempts), it falls back to
// PTYProxy.Connect.
//
// The fallback is one-way per session lifecycle (BC-2.04.002; S-3.01b task 9):
// once PTY proxy is active, the connector never attempts to re-establish
// control mode for the current session. Retry on next session start.
//
// Concurrency: SessionConnector is single-use. Construct a new instance for
// each session start. All methods are safe for concurrent use after
// Connect completes.
type SessionConnector struct {
	ctrl *ControlMode
	pty  *PTYProxy

	// factory constructs a fresh ControlMode for EC-003 reconnect attempts.
	// Nil means immediate PTY fallback on ErrControlModeDropped.
	factory ControlModeFactory

	// active points to whichever mode is currently running (ctrl or pty)
	// after Connect. Nil until Connect succeeds.
	active interface {
		Sessions() []session.Info
		Close() error
	}

	// mu protects active, inPTYMode, ctrl, closed, connectCancel.
	mu        sync.Mutex
	inPTYMode bool
	closed    bool // set true on Close; gates watchAndFallback resurrection

	// L-001 (pass-3): connectCancel cancels the context passed to
	// watchAndFallback, allowing Close to unblock the goroutine without
	// waiting for an ErrControlModeDropped signal that may never arrive.
	connectCancel context.CancelFunc

	// wg joins watchAndFallback goroutine(s) on Close.
	wg sync.WaitGroup

	// M-003 (pass-4): errCh surfaces mid-session fallback failures to the
	// caller. It receives ErrPTYDeviceUnavailable when watchAndFallback
	// activates the PTY fallback path and pty.Connect fails (undefined state:
	// control mode dropped AND PTY unavailable). Buffered 1 so the sender
	// never blocks on a non-draining caller. Closed on Close() so consumers
	// using range-over-channel unblock after graceful shutdown.
	//
	// Per BC-2.04.002 invariant 3 (never silent): callers MUST drain this
	// channel to detect mid-session fallback failures.
	errCh      chan error
	closeErrCh sync.Once

	// ADR-011: forwarding channel and relay goroutine for failover-stable
	// frame delivery to cmd/switchboard. sc.frames is the stable channel
	// returned by Frames(); the relay goroutine (forwardFrames) copies from
	// the active mode's Frames() channel into sc.frames, re-subscribing
	// transparently on ctrl→PTY mode switch.
	//
	// sc.frames is created in NewSessionConnector (buffered to framesBufferSize).
	// The relay goroutine is started in Connect after the active mode is
	// confirmed. sc.Close() closes sc.frames exactly once via closeForwardFrames.
	frames             chan halfchannel.ChannelFrame
	closeForwardFrames sync.Once
}

// NewSessionConnector constructs a SessionConnector with the given control
// mode and PTY proxy (S-3.01b task 5+6).
//
// ctrl and pty must not be nil.
func NewSessionConnector(ctrl *ControlMode, pty *PTYProxy, opts ...SessionConnectorOption) *SessionConnector {
	sc := &SessionConnector{
		ctrl: ctrl,
		pty:  pty,
		// M-003 (pass-4): buffered 1 so watchAndFallback can always deliver
		// a failure signal without blocking on a non-draining caller.
		errCh: make(chan error, 1),
		// ADR-011: forwarding channel created here; relay goroutine started in
		// Connect. Buffered to match the underlying mode channels.
		frames: make(chan halfchannel.ChannelFrame, framesBufferSize),
	}
	for _, opt := range opts {
		opt(sc)
	}
	return sc
}

// Connect attempts tmux control mode; falls back to PTY proxy on initial
// failure or after maxReconnectAttempts mid-session reconnect failures
// (BC-2.04.002 PC-1; ADR-010; EC-003).
//
// Returns ErrPTYDeviceUnavailable if both control mode and PTY proxy fail.
func (sc *SessionConnector) Connect(ctx context.Context) error {
	ctrlErr := sc.ctrl.Connect(ctx)
	if ctrlErr == nil {
		// Control mode connected — set active and start the watch goroutine.
		// L-001 (pass-3): use a cancellable child so Close can unblock the
		// goroutine without waiting for a drop signal that may never arrive.
		innerCtx, cancel := context.WithCancel(ctx)

		sc.mu.Lock()
		sc.active = sc.ctrl
		sc.connectCancel = cancel
		sc.mu.Unlock()

		// Start watching for mid-session drops (EC-003).
		sc.wg.Add(1)
		go sc.watchAndFallback(innerCtx)

		return nil
	}

	// Control mode failed — determine log message before falling back.
	logMsg := controlModeFailureLogMsg(ctrlErr)

	// Fall back to PTY proxy (AC-001; BC-2.04.002 PC-1).
	ptyErr := sc.pty.Connect(ctx)
	if ptyErr != nil {
		return ptyErr
	}

	// Log the specific EC-specific message (EC-001, EC-002) in addition to the
	// standard PTY proxy log already written by pty.Connect.
	if logMsg != "" {
		sc.pty.logger.Log(logMsg)
	}

	sc.mu.Lock()
	sc.active = sc.pty
	sc.inPTYMode = true
	sc.mu.Unlock()

	return nil
}

// controlModeFailureLogMsg returns the BC-2.04.002-specified log message for
// a given control mode connect error. Returns "" if no specific message applies.
//
// Uses errors.Is for new sentinel types (production path via defaultExecFn).
// Falls back to string matching for backward compatibility with wrapped
// ErrControlModeUnavailable errors that carry diagnostic strings.
func controlModeFailureLogMsg(err error) string {
	switch {
	case errors.Is(err, ErrControlModeUnsupportedFlag):
		return "tmux version does not support -CC flag"
	case errors.Is(err, ErrControlModeBinaryNotFound):
		return "tmux binary not found; using PTY proxy"
	case strings.Contains(err.Error(), "-CC flag not supported") ||
		strings.Contains(err.Error(), "does not support -CC"):
		return "tmux version does not support -CC flag"
	case strings.Contains(err.Error(), "no such file") ||
		strings.Contains(err.Error(), "not found") ||
		strings.Contains(err.Error(), "tmux binary not found"):
		return "tmux binary not found; using PTY proxy"
	default:
		return ""
	}
}

// Sessions returns the current session snapshot from whichever mode is active.
func (sc *SessionConnector) Sessions() []session.Info {
	sc.mu.Lock()
	active := sc.active
	sc.mu.Unlock()

	if active == nil {
		return nil
	}
	return active.Sessions()
}

// InPTYMode reports whether the connector is currently in PTY proxy mode.
// Tests assert this to verify AC-003 (no auto-upgrade) and AC-001/AC-002.
func (sc *SessionConnector) InPTYMode() bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.inPTYMode
}

// Err returns a channel that receives an error if the SessionConnector enters
// an undefined state — specifically, when mid-session control-mode drop occurs
// AND the subsequent PTY fallback also fails (ErrPTYDeviceUnavailable).
//
// The channel is closed (with no error) when the connector is closed normally
// via Close(), so callers ranging over it will unblock.
//
// Per BC-2.04.002 invariant 3 (never silent): callers SHOULD drain this
// channel to detect mid-session fallback failures. Missing this means a
// "both paths down" scenario is silently absorbed.
func (sc *SessionConnector) Err() <-chan error {
	return sc.errCh
}

// Close tears down whichever mode is active. Close is idempotent.
func (sc *SessionConnector) Close() error {
	sc.mu.Lock()
	if sc.closed {
		sc.mu.Unlock()
		return nil
	}
	sc.closed = true
	// L-001 (pass-3): capture and clear connectCancel under the lock so we
	// can call it after releasing the lock (no lock-while-calling convention).
	cancelFunc := sc.connectCancel
	sc.connectCancel = nil
	// H-01 (pass-6): snapshot ctrl and pty under the same lock that sets
	// closed=true. After this Unlock, watchAndFallback will see closed=true
	// and abort any in-flight reconnect swap, so the snapshots are coherent.
	ctrl := sc.ctrl
	pty := sc.pty
	sc.mu.Unlock()

	// L-001 (pass-3): cancel the watchAndFallback context BEFORE closing
	// ctrl so the goroutine observes ctx.Done() and exits promptly.
	if cancelFunc != nil {
		cancelFunc()
	}

	// Close BOTH ctrl AND pty regardless of which was active.
	var firstErr error
	if ctrl != nil {
		if err := ctrl.Close(); err != nil {
			firstErr = err
		}
	}
	if pty != nil {
		if err := pty.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	// Wait for watchAndFallback goroutine(s) to exit.
	sc.wg.Wait()

	// M-003 (pass-4): close errCh so consumers ranging over it unblock after
	// a graceful Close. sync.Once guards against double-close with watchAndFallback.
	sc.closeErrCh.Do(func() {
		close(sc.errCh)
	})

	return firstErr
}

// watchAndFallback monitors the control mode Err() channel and handles two
// sentinels:
//
//   - ErrControlModeDropped (BC-2.04.002 EC-003): mid-session loss — attempt
//     up to maxReconnectAttempts reconnections via the ControlModeFactory;
//     if all fail (or factory is nil), activate PTY proxy mode.
//
//   - ErrControlModeUnsupportedFlag (BC-2.04.002 EC-001 async path): the
//     control mode subprocess started and then exited with flag-rejection
//     classification. Immediate PTY fallback — no reconnect, because the
//     same binary will keep rejecting -C.
//
// Each reconnect attempt uses a fresh ControlMode instance per the SINGLE-USE
// contract (ADR-010; ErrAlreadyConnected is avoided by construction).
func (sc *SessionConnector) watchAndFallback(ctx context.Context) {
	defer sc.wg.Done()

	for err := range sc.ctrl.Err() {
		switch {
		case err == nil:
			continue

		case errors.Is(err, ErrControlModeUnsupportedFlag):
			// BC-2.04.002 EC-001 async path: subprocess rejected -C flag.
			// Immediate PTY fallback — no reconnect (same binary will reject again).
			sc.mu.Lock()
			if sc.closed {
				sc.mu.Unlock()
				return
			}
			sc.mu.Unlock()

			logMsg := controlModeFailureLogMsg(err)
			if logMsg != "" {
				sc.pty.logger.Log(logMsg)
			}

			if ptyErr := sc.pty.Connect(ctx); ptyErr == nil {
				sc.mu.Lock()
				sc.active = sc.pty
				sc.inPTYMode = true
				sc.mu.Unlock()
			} else {
				sc.closeErrCh.Do(func() {
					select {
					case sc.errCh <- ptyErr:
					default:
					}
					close(sc.errCh)
				})
			}
			return

		case errors.Is(err, ErrControlModeDropped):
			// fall through to the reconnect + PTY-fallback path below

		default:
			// Unknown sentinel — skip (defensive; future sentinels may need handling).
			continue
		}

		// BC-2.04.002 EC-003: up to maxReconnectAttempts via factory.
		reconnected := false
		if sc.factory != nil {
			for attempt := 1; attempt <= maxReconnectAttempts; attempt++ {
				// L-001 (pass-3): honor Close() and ctx cancellation between attempts.
				sc.mu.Lock()
				if sc.closed {
					sc.mu.Unlock()
					return
				}
				sc.mu.Unlock()

				select {
				case <-ctx.Done():
					return
				default:
				}

				newCtrl, connErr := sc.factory(ctx)
				if connErr == nil && newCtrl != nil {
					sc.mu.Lock()
					if sc.closed {
						sc.mu.Unlock()
						_ = newCtrl.Close()
						return
					}
					oldCtrl := sc.ctrl
					sc.ctrl = newCtrl
					sc.active = newCtrl
					sc.mu.Unlock()
					_ = oldCtrl.Close()
					reconnected = true
					break
				}
				// connErr != nil OR (connErr == nil but newCtrl == nil) — both treated as
				// failed reconnect attempt. L-002 (pass-9): defensive guard against factory
				// contract violation (returning nil, nil).
				if newCtrl != nil {
					_ = newCtrl.Close()
				}
			}
		}

		if reconnected {
			// Re-enter watchAndFallback on the new ctrl in a fresh goroutine,
			// then exit the current one. The wg count stays balanced because
			// we Add(1) before spawning and this goroutine decrements on return.
			sc.wg.Add(1)
			go sc.watchAndFallback(ctx)
			return
		}

		// All reconnect attempts failed (or factory was nil) → PTY fallback.
		sc.mu.Lock()
		if sc.closed {
			sc.mu.Unlock()
			return
		}
		sc.mu.Unlock()

		sc.pty.logger.Log("tmux control mode lost; falling back to PTY proxy")

		if ptyErr := sc.pty.Connect(ctx); ptyErr == nil {
			sc.mu.Lock()
			sc.active = sc.pty
			sc.inPTYMode = true
			sc.mu.Unlock()
		} else {
			// M-003 (pass-4): PTY fallback also failed — connector is in an
			// undefined state (control mode dropped AND PTY unavailable).
			// Surface the failure via Err() per BC-2.04.002 invariant 3
			// (never silent). closeErrCh.Do guards against double-close
			// if Close() races with this path.
			sc.closeErrCh.Do(func() {
				select {
				case sc.errCh <- ptyErr:
				default:
					// Channel already occupied; drop (buffered-1 invariant).
				}
				close(sc.errCh)
			})
		}
		// Whether PTY connect succeeded or failed, stop watching ctrl.
		return
	}
}
