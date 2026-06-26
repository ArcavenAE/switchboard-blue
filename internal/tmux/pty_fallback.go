package tmux

// PTY proxy fallback for the access node (BC-2.04.002; ADR-010).
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
// Allowed internal imports: {halfchannel, session} per ARCH-08 §6.6.
// Forbidden: internal/admission, internal/routing, internal/hmac.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/arcavenae/switchboard/internal/halfchannel"
	"github.com/arcavenae/switchboard/internal/session"
)

// ErrPTYDeviceUnavailable is returned by PTYProxy.Connect when no PTY device
// can be allocated on the host. Maps to E-SYS-001: "PTY device unavailable:
// cannot start access node" (error-taxonomy.md §SYS; BC-2.04.002 EC-004;
// FM-011).
//
// When both tmux control mode and PTY device are unavailable, the access node
// must return this error and exit with a non-zero status. Failure is never
// silent (BC-2.04.002 invariant 3).
var ErrPTYDeviceUnavailable = errors.New("PTY device unavailable: cannot start access node")

// maxReconnectAttempts is the number of tmux control mode reconnect attempts
// before the mid-session fallback path switches to PTY proxy mode
// (BC-2.04.002 EC-003; S-3.01b task 6).
const maxReconnectAttempts = 3 //nolint:unused // stub constant; wired by implementer in SessionConnector.Connect

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
// a no-op logger (log entries are discarded). Tests inject a fake logger
// to assert mandatory log messages (BC-2.04.002 PC-3).
func WithLogger(l Logger) PTYProxyOption {
	return func(p *PTYProxy) {
		p.logger = l
	}
}

// PTYProxy implements PTY proxy mode for the access node
// (BC-2.04.002; ADR-010). It opens a PTY and proxies its I/O as a single
// anonymous session published under a synthetic name ("pty-<pid>").
//
// Concurrency: PTYProxy is safe for concurrent use. All exported methods
// are protected by mu. The I/O goroutine (started by Connect) is the
// only writer to the downstream half-channel; it is joined by Close via wg.
//
// The zero value is not usable; construct with NewPTYProxy.
type PTYProxy struct {
	publisher  *session.Publisher
	downstream *halfchannel.HalfChannel
	logger     Logger

	// ptyAlloc allocates the PTY device; replaced in tests via WithPTYAllocFunc.
	ptyAlloc ptyAllocFunc

	// sessionName is the synthetic "pty-<pid>" name published on Connect.
	sessionName string //nolint:unused // stub field; set by implementer in Connect

	// mu protects all lifecycle fields.
	mu     sync.Mutex         //nolint:unused // stub field; used by implementer in Connect/Close
	master io.ReadWriteCloser //nolint:unused // stub field; set by implementer in Connect
	pid    int                //nolint:unused // stub field; set by implementer in Connect
	closed bool               //nolint:unused // stub field; set by implementer in Close

	// wg joins the I/O relay goroutine on Close.
	wg sync.WaitGroup //nolint:unused // stub field; used by implementer in Connect/Close
}

// NewPTYProxy constructs a PTYProxy that publishes sessions via publisher and
// delivers output frames to downstream (BC-2.04.002; S-3.01b task 4).
//
// publisher and downstream must not be nil.
func NewPTYProxy(publisher *session.Publisher, downstream *halfchannel.HalfChannel, opts ...PTYProxyOption) *PTYProxy {
	p := &PTYProxy{
		publisher:  publisher,
		downstream: downstream,
		logger:     noopLogger{},
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
func (p *PTYProxy) Connect(ctx context.Context) error {
	// Red Gate: non-trivial body deferred to implementer.
	// Traces: BC-2.04.002 PC-1..PC-3; AC-001; AC-002.
	panic("not implemented: PTYProxy.Connect (BC-2.04.002 PC-1..PC-3; AC-001; AC-002)")
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
	// Red Gate: non-trivial body deferred to implementer.
	// Traces: BC-2.04.002; S-3.01b task 4.
	panic("not implemented: PTYProxy.Close (BC-2.04.002)")
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

	// active points to whichever mode is currently running (ctrl or pty)
	// after Connect. Nil until Connect succeeds.
	active interface {
		Sessions() []session.Info
		Close() error
	}

	// mu protects active and inPTYMode.
	mu        sync.Mutex
	inPTYMode bool
}

// NewSessionConnector constructs a SessionConnector with the given control
// mode and PTY proxy (S-3.01b task 5+6).
//
// ctrl and pty must not be nil.
func NewSessionConnector(ctrl *ControlMode, pty *PTYProxy) *SessionConnector {
	return &SessionConnector{
		ctrl: ctrl,
		pty:  pty,
	}
}

// Connect attempts tmux control mode; falls back to PTY proxy on initial
// failure or after maxReconnectAttempts mid-session reconnect failures
// (BC-2.04.002 PC-1; ADR-010; EC-003).
//
// Returns ErrPTYDeviceUnavailable if both control mode and PTY proxy fail.
func (sc *SessionConnector) Connect(ctx context.Context) error {
	// Red Gate: non-trivial body deferred to implementer.
	// Traces: BC-2.04.002 PC-1; AC-001; EC-001; EC-002; EC-003; ADR-010.
	panic("not implemented: SessionConnector.Connect (BC-2.04.002 PC-1; AC-001)")
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

// Close tears down whichever mode is active.
func (sc *SessionConnector) Close() error {
	sc.mu.Lock()
	active := sc.active
	sc.active = nil
	sc.mu.Unlock()

	if active == nil {
		return nil
	}
	return active.Close()
}

// watchAndFallback monitors the control mode Err() channel in a goroutine.
// On receiving ErrControlModeDropped, it attempts maxReconnectAttempts
// reconnections (EC-003). If all fail, it activates PTY proxy mode
// (BC-2.04.002 EC-003; S-3.01b task 6).
//
// reconnectFn is injected for testability (hermetic test pattern —
// production path calls ctrl.Connect; tests inject a fake).
func (sc *SessionConnector) watchAndFallback(ctx context.Context, reconnectFn func(context.Context) error) { //nolint:unused // stub method; called by implementer from SessionConnector.Connect
	// Red Gate: non-trivial body deferred to implementer.
	// Traces: BC-2.04.002 EC-003; S-3.01b task 6.
	panic("not implemented: SessionConnector.watchAndFallback (BC-2.04.002 EC-003)")
}

// defaultPTYAlloc is the production PTY allocator. It calls
// golang.org/x/sys/unix.Openpty and spawns a shell process on the slave
// side. Replaced in unit tests via WithPTYAllocFunc (hermetic test pattern).
//
// The real implementation is deferred to the implementer (Red Gate stub).
func defaultPTYAlloc() (io.ReadWriteCloser, int, error) {
	// Red Gate: non-trivial body deferred to implementer.
	// Traces: BC-2.04.002 PC-1; ARCH-09 effectful classification.
	panic("not implemented: defaultPTYAlloc (BC-2.04.002 PC-1; requires golang.org/x/sys/unix.Openpty)")
}

// noopLogger discards all log messages. Used as the default when no
// Logger is injected. BC-2.04.002 invariant 3 requires mandatory log
// entries; tests must inject a real Logger to assert them.
type noopLogger struct{}

func (noopLogger) Log(_ string) {}

// logFallbackEntry writes the mandatory BC-2.04.002 PC-3 log entry.
// The exact message is specified by the behavioral contract so that
// operators can grep for it reliably.
//
// This is a thin wrapper — it is GREEN-BY-DESIGN if the body were only
// logger.Log(...), but it is called from non-trivial callers (Connect)
// so it remains a helper kept here for the implementer to wire in.
func logFallbackEntry(_ Logger, reason string) { //nolint:unused // stub function; called by implementer in PTYProxy.Connect / SessionConnector.Connect
	// Red Gate: non-trivial body deferred to implementer.
	// Traces: BC-2.04.002 PC-3; VP-032 ("log entry is written on every fallback event").
	// Implementer: replace _ with logger and call logger.Log(msg).
	panic(fmt.Sprintf("not implemented: logFallbackEntry (BC-2.04.002 PC-3; reason: %q)", reason))
}
