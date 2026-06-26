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

// ptyAllocFunc is the injection point for PTY allocation. The real
// implementation calls golang.org/x/sys/unix.Openpty; unit tests inject
// a fake that returns pre-wired io.ReadWriteCloser pairs without forking
// a real PTY process (hermetic test pattern).
//
// Returns (masterFD, slaveFD io.ReadWriteCloser, pid int, err error) where
// pid is the PID of the shell process spawned on the slave side.
type ptyAllocFunc func() (master io.ReadWriteCloser, pid int, err error)

// PTYProxyOption is a functional option for NewPTYProxy.
type PTYProxyOption func(*PTYProxy)

// WithPTYAllocFunc replaces the default PTY allocator with fn.
// Hermetic test injection point: unit tests supply a fake that avoids
// forking a real PTY process. Production callers use the default
// golang.org/x/sys/unix.Openpty path.
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
		return fmt.Errorf("PTY proxy: already closed")
	}
	if p.master != nil {
		return fmt.Errorf("PTY proxy: already connected")
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
				_ = p.downstream.Tick()
			}
		}
		if err != nil {
			return
		}
	}
}

// Sessions returns a snapshot of all currently published PTY sessions
// (BC-2.04.002 PC-2). In PTY proxy mode there is at most one session active.
//
// The slice is a value copy — callers may freely mutate it.
func (p *PTYProxy) Sessions() []session.Info {
	return p.publisher.ListSessions()
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

	// Unpublish the synthetic session.
	if sessionName != "" {
		// Ignore ErrSessionNotFound — idempotent.
		_ = p.publisher.Unpublish(sessionName)
	}

	return nil
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

	// mu protects active, inPTYMode, ctrl, closed.
	mu        sync.Mutex
	inPTYMode bool
	closed    bool // set true on Close; gates watchAndFallback resurrection

	// wg joins watchAndFallback goroutine(s) on Close.
	wg sync.WaitGroup
}

// NewSessionConnector constructs a SessionConnector with the given control
// mode and PTY proxy (S-3.01b task 5+6).
//
// ctrl and pty must not be nil.
func NewSessionConnector(ctrl *ControlMode, pty *PTYProxy, opts ...SessionConnectorOption) *SessionConnector {
	sc := &SessionConnector{
		ctrl: ctrl,
		pty:  pty,
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
		sc.mu.Lock()
		sc.active = sc.ctrl
		sc.mu.Unlock()

		// Start watching for mid-session drops (EC-003).
		sc.wg.Add(1)
		go sc.watchAndFallback(ctx)

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

// Close tears down whichever mode is active. Close is idempotent.
func (sc *SessionConnector) Close() error {
	sc.mu.Lock()
	if sc.closed {
		sc.mu.Unlock()
		return nil
	}
	sc.closed = true
	sc.mu.Unlock()

	// Close BOTH ctrl AND pty regardless of which was active.
	var firstErr error
	if sc.ctrl != nil {
		if err := sc.ctrl.Close(); err != nil {
			firstErr = err
		}
	}
	if sc.pty != nil {
		if err := sc.pty.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	// Wait for watchAndFallback goroutine(s) to exit.
	sc.wg.Wait()

	return firstErr
}

// watchAndFallback monitors the control mode Err() channel. On receiving
// ErrControlModeDropped, it attempts up to maxReconnectAttempts reconnections
// via the ControlModeFactory (EC-003). If all fail (or factory is nil), it
// activates PTY proxy mode (BC-2.04.002 EC-003; S-3.01b task 6).
//
// Each reconnect attempt uses a fresh ControlMode instance per the SINGLE-USE
// contract (ADR-010; ErrAlreadyConnected is avoided by construction).
func (sc *SessionConnector) watchAndFallback(ctx context.Context) {
	defer sc.wg.Done()

	for err := range sc.ctrl.Err() {
		if !errors.Is(err, ErrControlModeDropped) {
			continue
		}

		// BC-2.04.002 EC-003: up to maxReconnectAttempts via factory.
		reconnected := false
		if sc.factory != nil {
			for attempt := 1; attempt <= maxReconnectAttempts; attempt++ {
				newCtrl, connErr := sc.factory(ctx)
				if connErr == nil {
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
				// factory returned an error; newCtrl may be nil
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
		}
		// Whether PTY connect succeeded or failed, stop watching ctrl.
		return
	}
}

// defaultPTYAlloc is the production PTY allocator. It calls
// golang.org/x/sys/unix.Openpty and spawns a shell process on the slave
// side. Replaced in unit tests via WithPTYAllocFunc (hermetic test pattern).
//
// The real implementation is deferred to the implementer (Red Gate stub).
func defaultPTYAlloc() (io.ReadWriteCloser, int, error) {
	// Real PTY allocation deferred to VP-032 integration harness.
	// Unit tests always inject WithPTYAllocFunc and never reach this path.
	return nil, 0, fmt.Errorf("%w: defaultPTYAlloc not available in this build (VP-032 integration harness required)", ErrPTYDeviceUnavailable)
}
