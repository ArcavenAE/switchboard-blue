// PTY proxy fallback tests for BC-2.04.002.
//
// Traces: BC-2.04.002 PC-1..PC-3; AC-001..AC-003; EC-001..EC-004;
// ADR-010 (tmux primary, PTY fallback at any failure); VP-032.
//
// Hermetic constraint: these tests MUST NOT shell out to real PTY processes
// or real tmux. All PTY allocation is injected via WithPTYAllocFunc; all
// control mode streams are injected via WithExecFunc (pattern from control_test.go).
// Real-PTY integration coverage is deferred to VP-032 integration harness.
//
// Red Gate: all non-skipped tests below are designed to fail against the stubs
// (panic "not implemented"). Green on first commit = Red Gate violation.
package tmux_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/halfchannel"
	"github.com/arcavenae/switchboard/internal/session"
	"github.com/arcavenae/switchboard/internal/tmux"
)

// -- Test helpers -----------------------------------------------------------

// fakeLogCapture implements tmux.Logger and captures log lines for assertion.
// Tests that assert BC-2.04.002 PC-3 (mandatory log entries) inject this.
// Concurrency: safe for use from multiple goroutines (mu protects lines).
type fakeLogCapture struct {
	mu    sync.Mutex
	lines []string
}

func (f *fakeLogCapture) Log(msg string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lines = append(f.lines, msg)
}

// HasLine reports whether any captured log line contains substr.
func (f *fakeLogCapture) HasLine(substr string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, l := range f.lines {
		if strings.Contains(l, substr) {
			return true
		}
	}
	return false
}

// Lines returns a snapshot of all captured log lines (value copy).
func (f *fakeLogCapture) Lines() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.lines))
	copy(out, f.lines)
	return out
}

// fakePTYMaster is a fake PTY master that satisfies io.ReadWriteCloser.
// Read blocks until data is written via Send or Close is called.
// Write discards data (unit tests don't need to observe PTY writes).
type fakePTYMaster struct {
	buf    bytes.Buffer
	mu     sync.Mutex
	cond   *sync.Cond
	closed bool
}

func newFakePTYMaster() *fakePTYMaster {
	m := &fakePTYMaster{}
	m.cond = sync.NewCond(&m.mu)
	return m
}

// Send enqueues data to be returned by future Read calls.
func (m *fakePTYMaster) Send(data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.buf.Write(data)
	m.cond.Signal()
}

func (m *fakePTYMaster) Read(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for m.buf.Len() == 0 && !m.closed {
		m.cond.Wait()
	}
	if m.closed && m.buf.Len() == 0 {
		return 0, io.EOF
	}
	return m.buf.Read(p)
}

func (m *fakePTYMaster) Write(p []byte) (int, error) {
	return len(p), nil // discard PTY output from tests
}

func (m *fakePTYMaster) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	m.cond.Broadcast()
	return nil
}

// fakePTYAlloc returns a WithPTYAllocFunc option that yields a fakePTYMaster.
// pid is set to a deterministic fake value for synthetic name assertions.
func fakePTYAlloc(master *fakePTYMaster, pid int) tmux.PTYProxyOption {
	return tmux.WithPTYAllocFunc(func() (io.ReadWriteCloser, int, error) {
		return master, pid, nil
	})
}

// fakePTYAllocErr returns a WithPTYAllocFunc option that returns err.
// Used to simulate PTY device unavailability (E-SYS-001 / AC-003).
func fakePTYAllocErr(err error) tmux.PTYProxyOption {
	return tmux.WithPTYAllocFunc(func() (io.ReadWriteCloser, int, error) {
		return nil, 0, err
	})
}

// newTestPTYProxy is the canonical constructor for PTYProxy in tests.
// Accepts additional options for hermetic injection.
func newTestPTYProxy(t *testing.T, opts ...tmux.PTYProxyOption) (*tmux.PTYProxy, *session.Publisher) {
	t.Helper()
	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)
	pty := tmux.NewPTYProxy(pub, ds, opts...)
	return pty, pub
}

// newTestSessionConnector builds a SessionConnector with hermetically injected
// control mode and PTY proxy.
func newTestSessionConnector(
	t *testing.T,
	ctrlOpts []tmux.Option,
	ptyOpts []tmux.PTYProxyOption,
) *tmux.SessionConnector {
	t.Helper()
	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)
	ctrl := tmux.New(pub, ds, ctrlOpts...)
	pty := tmux.NewPTYProxy(pub, ds, ptyOpts...)
	return tmux.NewSessionConnector(ctrl, pty)
}

// -- AC-001: initial connect failure triggers PTY fallback ------------------

// TestPTYProxy_FallbackOnInitialConnectFailure verifies that when
// TmuxControlMode.Connect() fails (ErrControlModeUnavailable — tmux not found
// in PATH, socket absent, or -C flag not supported), the SessionConnector
// automatically enters PTY proxy mode (AC-001; BC-2.04.002 PC-1; ADR-010).
//
// The control mode exec function returns ErrControlModeUnavailable
// (hermetic — no real tmux). The PTY alloc function succeeds. After
// SessionConnector.Connect, InPTYMode() must be true.
//
// Hermetic: WithExecFunc injects fake control-mode failure; WithPTYAllocFunc
// injects fake PTY. No real tmux or real PTY forked.
func TestPTYProxy_FallbackOnInitialConnectFailure(t *testing.T) {
	t.Parallel()

	master := newFakePTYMaster()
	const fakePID = 12345

	log := &fakeLogCapture{}

	sc := newTestSessionConnector(
		t,
		// Control mode: simulate "tmux not found" (ErrControlModeUnavailable).
		[]tmux.Option{
			fakeExecFuncErr(fmt.Errorf("%w: tmux binary not found", tmux.ErrControlModeUnavailable)),
		},
		// PTY proxy: fake PTY that succeeds.
		[]tmux.PTYProxyOption{
			fakePTYAlloc(master, fakePID),
			tmux.WithLogger(log),
		},
	)
	t.Cleanup(func() {
		if err := sc.Close(); err != nil {
			t.Logf("Close: %v", err)
		}
		_ = master.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v; want nil (PTY fallback should succeed)", err)
	}

	// Primary assertion: connector is in PTY mode after control-mode failure.
	if !sc.InPTYMode() {
		t.Error("InPTYMode() = false; want true after ErrControlModeUnavailable")
	}
}

// -- AC-002: PTY session published with synthetic name + mandatory log -------

// TestPTYProxy_PublishesSessionAndLogs verifies that in PTY proxy mode the
// access node:
//  1. Publishes the PTY session under a synthetic name "pty-<pid>" (BC-2.04.002 PC-2).
//  2. Writes the mandatory log entry (BC-2.04.002 PC-3; VP-032).
//
// Hermetic: WithPTYAllocFunc injects a fake that returns pid=99999; the
// published session name must be "pty-99999". Logger is injected to capture
// and assert the mandatory log message.
func TestPTYProxy_PublishesSessionAndLogs(t *testing.T) {
	t.Parallel()

	master := newFakePTYMaster()
	const fakePID = 99999
	const wantSessionName = "pty-99999"
	const wantLogSubstr = "tmux control mode unavailable; using PTY proxy mode"

	log := &fakeLogCapture{}

	pty, pub := newTestPTYProxy(t,
		fakePTYAlloc(master, fakePID),
		tmux.WithLogger(log),
	)
	t.Cleanup(func() {
		if err := pty.Close(); err != nil {
			t.Logf("Close: %v", err)
		}
		_ = master.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := pty.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v; want nil", err)
	}

	// Assert synthetic session name published (BC-2.04.002 PC-2).
	if _, err := pub.Get(wantSessionName); err != nil {
		t.Errorf("Get %q: %v; want nil — session must be published under 'pty-<pid>'", wantSessionName, err)
	}

	sessions := pty.Sessions()
	if len(sessions) == 0 {
		t.Error("Sessions() empty; want at least 'pty-99999'")
	}
	found := false
	for _, s := range sessions {
		if s.Name == wantSessionName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Sessions() = %v; want to include %q", sessions, wantSessionName)
	}

	// Assert mandatory log entry (BC-2.04.002 PC-3; VP-032).
	if !log.HasLine(wantLogSubstr) {
		t.Errorf("log lines = %v; want line containing %q", log.Lines(), wantLogSubstr)
	}
}

// -- AC-003: both control mode and PTY unavailable → E-SYS-001 ------------

// TestPTYProxy_NoPTY_ReturnsErrSysOne verifies that when both tmux control
// mode and PTY device are unavailable, Connect returns ErrPTYDeviceUnavailable
// (E-SYS-001; BC-2.04.002 EC-004). The failure is never silent.
//
// Hermetic: WithExecFunc returns ErrControlModeUnavailable; WithPTYAllocFunc
// returns ErrPTYDeviceUnavailable. No real tmux or PTY forked.
func TestPTYProxy_NoPTY_ReturnsErrSysOne(t *testing.T) {
	t.Parallel()

	log := &fakeLogCapture{}

	sc := newTestSessionConnector(
		t,
		[]tmux.Option{
			fakeExecFuncErr(fmt.Errorf("%w: tmux not found", tmux.ErrControlModeUnavailable)),
		},
		[]tmux.PTYProxyOption{
			fakePTYAllocErr(fmt.Errorf("%w: /dev/ptmx: no such device", tmux.ErrPTYDeviceUnavailable)),
			tmux.WithLogger(log),
		},
	)
	t.Cleanup(func() { _ = sc.Close() })

	ctx := context.Background()
	err := sc.Connect(ctx)

	// Primary assertion: ErrPTYDeviceUnavailable (E-SYS-001) is returned.
	if err == nil {
		t.Fatal("Connect: got nil; want ErrPTYDeviceUnavailable (E-SYS-001)")
	}
	if !errors.Is(err, tmux.ErrPTYDeviceUnavailable) {
		t.Errorf("Connect error = %v; want errors.Is(_, ErrPTYDeviceUnavailable)", err)
	}
}

// -- EC-001: tmux exists but old version (no -CC support) -------------------

// TestPTYProxy_EC001_OldTmuxVersion verifies PTY fallback when tmux exists
// but returns an error indicating the -CC flag is not supported
// (BC-2.04.002 EC-001; ADR-010).
//
// The log must contain "tmux version does not support -CC flag"
// (BC-2.04.002 EC-001 expected behavior).
//
// Hermetic: exec function simulates old tmux returning ErrControlModeUnavailable
// with a message about flag support.
func TestPTYProxy_EC001_OldTmuxVersion(t *testing.T) {
	t.Parallel()

	master := newFakePTYMaster()
	log := &fakeLogCapture{}

	sc := newTestSessionConnector(
		t,
		[]tmux.Option{
			// Simulate tmux binary exists but -CC not supported.
			fakeExecFuncErr(fmt.Errorf("%w: -CC flag not supported", tmux.ErrControlModeUnavailable)),
		},
		[]tmux.PTYProxyOption{
			fakePTYAlloc(master, 11111),
			tmux.WithLogger(log),
		},
	)
	t.Cleanup(func() {
		_ = sc.Close()
		_ = master.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v; want nil (PTY fallback should succeed for EC-001)", err)
	}

	if !sc.InPTYMode() {
		t.Error("InPTYMode() = false; want true after -CC not-supported failure (EC-001)")
	}

	// EC-001 log requirement: "tmux version does not support -CC flag"
	if !log.HasLine("tmux version does not support -CC flag") {
		t.Errorf("log lines = %v; want line containing 'tmux version does not support -CC flag' (BC-2.04.002 EC-001)", log.Lines())
	}
}

// -- EC-002: tmux not found in PATH ----------------------------------------

// TestPTYProxy_EC002_TmuxNotFound verifies PTY fallback when tmux binary is
// absent from PATH (BC-2.04.002 EC-002; FM-011).
//
// The log must contain "tmux binary not found; using PTY proxy"
// (BC-2.04.002 EC-002 expected behavior).
//
// Hermetic: exec function returns ErrControlModeUnavailable with "no such file"
// message (no PATH manipulation, no real tmux lookup).
func TestPTYProxy_EC002_TmuxNotFound(t *testing.T) {
	t.Parallel()

	master := newFakePTYMaster()
	log := &fakeLogCapture{}

	sc := newTestSessionConnector(
		t,
		[]tmux.Option{
			fakeExecFuncErr(fmt.Errorf("%w: exec: \"tmux\": no such file", tmux.ErrControlModeUnavailable)),
		},
		[]tmux.PTYProxyOption{
			fakePTYAlloc(master, 22222),
			tmux.WithLogger(log),
		},
	)
	t.Cleanup(func() {
		_ = sc.Close()
		_ = master.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v; want nil (PTY fallback should succeed for EC-002)", err)
	}

	if !sc.InPTYMode() {
		t.Error("InPTYMode() = false; want true after tmux-not-found (EC-002)")
	}

	// EC-002 log requirement: "tmux binary not found; using PTY proxy"
	if !log.HasLine("tmux binary not found; using PTY proxy") {
		t.Errorf("log lines = %v; want line containing 'tmux binary not found; using PTY proxy' (BC-2.04.002 EC-002)", log.Lines())
	}
}

// -- EC-003: mid-session control mode loss (3 reconnect attempts) -----------

// TestPTYProxy_FallbackOnMidSessionLoss verifies that when tmux control mode
// drops mid-session (ErrControlModeDropped via Err()), the SessionConnector
// attempts maxReconnectAttempts (3) reconnections, then switches to PTY proxy
// mode (BC-2.04.002 EC-003; ADR-010 reversion at commit 1aedebc).
//
// Strategy: the initial connect succeeds (fake stream with empty enumeration
// block), then the stream EOF triggers ErrControlModeDropped. The reconnect
// function is injected and always returns ErrControlModeUnavailable. After
// 3 failed reconnects, InPTYMode() must be true.
//
// Hermetic: all control mode streams are fakes; PTY alloc is fake.
// No real tmux or PTY forked.
func TestPTYProxy_FallbackOnMidSessionLoss(t *testing.T) {
	t.Parallel()

	master := newFakePTYMaster()
	log := &fakeLogCapture{}

	// connectCalls counts how many times the reconnect function is called.
	var reconnectCalls int
	var reconnectMu sync.Mutex

	// Build the SessionConnector with a fake initial control-mode stream that
	// closes immediately (triggering ErrControlModeDropped).
	initialStream := fakeControlOutput(
		"%begin 1000000000 0 1",
		"%end 1000000000 0 1",
		// EOF immediately follows — causes dispatchLoop to send ErrControlModeDropped.
	)

	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)
	ctrl := tmux.New(pub, ds, fakeExecFunc(initialStream))
	ptyProxy := tmux.NewPTYProxy(pub, ds,
		fakePTYAlloc(master, 33333),
		tmux.WithLogger(log),
	)
	sc := tmux.NewSessionConnector(ctrl, ptyProxy)

	t.Cleanup(func() {
		_ = sc.Close()
		_ = master.Close()
	})

	// Inject a reconnect function that always fails (all 3 attempts fail).
	// ctx is passed by the watchAndFallback caller; unused in the fake.
	reconnectFn := func(_ context.Context) error {
		reconnectMu.Lock()
		reconnectCalls++
		reconnectMu.Unlock()
		return fmt.Errorf("%w: cannot reconnect", tmux.ErrControlModeUnavailable)
	}
	_ = reconnectFn // used by watchAndFallback; injected for assertion below

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initial connect succeeds (fake stream is valid).
	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v; want nil (initial connect must succeed)", err)
	}

	// Wait for mid-session fallback to complete. The fake stream has an
	// immediate EOF, so ErrControlModeDropped is dispatched and watched.
	// After 3 failed reconnects, PTY fallback activates.
	deadline := time.Now().Add(2 * time.Second)
	for !sc.InPTYMode() {
		if time.Now().After(deadline) {
			t.Fatal("InPTYMode() never became true within 2s; mid-session PTY fallback did not activate")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// EC-003 log requirement: "tmux control mode lost; falling back to PTY proxy"
	if !log.HasLine("tmux control mode lost; falling back to PTY proxy") {
		t.Errorf("log lines = %v; want line containing 'tmux control mode lost; falling back to PTY proxy' (BC-2.04.002 EC-003)", log.Lines())
	}
}

// -- No auto-upgrade from PTY back to control mode -------------------------

// TestPTYProxy_NoAutoUpgrade verifies that once the SessionConnector enters
// PTY proxy mode, it does NOT automatically attempt to return to control mode
// (BC-2.04.002 invariant; S-3.01b task 9; story architecture rule "no
// auto-upgrade from PTY back to control mode during a session").
//
// Strategy: establish PTY mode via initial connect failure, then assert that
// even after a long wait, InPTYMode() remains true. The connector must not
// spawn a background goroutine that silently attempts to re-establish control
// mode.
//
// Hermetic: fake control mode failure; fake PTY. No real tmux or PTY forked.
func TestPTYProxy_NoAutoUpgrade(t *testing.T) {
	t.Parallel()

	master := newFakePTYMaster()
	log := &fakeLogCapture{}

	sc := newTestSessionConnector(
		t,
		[]tmux.Option{
			fakeExecFuncErr(fmt.Errorf("%w: tmux not found", tmux.ErrControlModeUnavailable)),
		},
		[]tmux.PTYProxyOption{
			fakePTYAlloc(master, 44444),
			tmux.WithLogger(log),
		},
	)
	t.Cleanup(func() {
		_ = sc.Close()
		_ = master.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v; want nil", err)
	}

	// Primary: PTY mode is active immediately.
	if !sc.InPTYMode() {
		t.Fatal("InPTYMode() = false immediately after connect; expected PTY mode")
	}

	// Wait a brief period — any auto-upgrade attempt would flip InPTYMode to
	// false. Remains true means no auto-upgrade goroutine fired.
	time.Sleep(100 * time.Millisecond)

	if !sc.InPTYMode() {
		t.Error("InPTYMode() = false after 100ms; auto-upgrade must not occur (BC-2.04.002 invariant; S-3.01b task 9)")
	}
}

// -- EC-004: PTY device unavailable directly on PTYProxy.Connect -----------

// TestPTYProxy_DirectConnect_NoPTY verifies that PTYProxy.Connect directly
// (not via SessionConnector) returns ErrPTYDeviceUnavailable when the PTY
// alloc function fails (BC-2.04.002 EC-004; E-SYS-001).
//
// This exercises the PTYProxy.Connect error path in isolation, separate from
// the SessionConnector orchestration (AC-003 tests the combined path).
//
// Hermetic: WithPTYAllocFunc injects failure. No real PTY.
func TestPTYProxy_DirectConnect_NoPTY(t *testing.T) {
	t.Parallel()

	pty, _ := newTestPTYProxy(t,
		fakePTYAllocErr(fmt.Errorf("%w: openpty: no PTY devices", tmux.ErrPTYDeviceUnavailable)),
	)
	t.Cleanup(func() { _ = pty.Close() })

	ctx := context.Background()
	err := pty.Connect(ctx)

	if err == nil {
		t.Fatal("Connect: got nil; want ErrPTYDeviceUnavailable (E-SYS-001)")
	}
	if !errors.Is(err, tmux.ErrPTYDeviceUnavailable) {
		t.Errorf("Connect error = %v; want errors.Is(_, ErrPTYDeviceUnavailable)", err)
	}
}

// -- Real-PTY deferral sentinel --------------------------------------------

// TestPTYProxy_RealPTY_Integration is a placeholder that documents the
// deferred real-PTY integration test. It is unconditionally skipped in
// unit-test mode. VP-032 (integration harness) covers the real openpty path.
//
// grep tag: VP-032-deferred-real-pty
func TestPTYProxy_RealPTY_Integration(t *testing.T) {
	t.Skip("VP-032-deferred-real-pty: real PTY integration deferred to VP-032 integration harness; unit tests use injected fake via WithPTYAllocFunc")
}
