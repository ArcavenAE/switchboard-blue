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
	// BC-2.04.002 PC-3 mandates the exact two-sentence canonical message.
	const wantLogSubstr = "tmux control mode unavailable; using PTY proxy mode. Functionality limited: no structured session metadata, no content-type detection."

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

	// BC-2.04.002 EC-004: the full operator-facing guidance MUST be emitted via
	// the configured logger. Failure is never silent (invariant 3).
	if !log.HasLine("Install 'openpty' or check device permissions") {
		t.Errorf("logger did not emit BC-2.04.002 EC-004 guidance text; got: %v", log.Lines())
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

// TestPTYProxy_EC002_TmuxNotFound_ViaSentinel verifies that wrapping
// ErrControlModeBinaryNotFound directly (the production path from
// defaultExecFn) causes controlModeFailureLogMsg to emit the canonical
// EC-002 message via the errors.Is branch, not via the string-match
// fallback exercised by TestPTYProxy_EC002_TmuxNotFound.
//
// Traces to:
//
//	BC-2.04.002 EC-002 (tmux binary not present)
//	pass-6 L-04 (cover errors.Is branch in controlModeFailureLogMsg)
func TestPTYProxy_EC002_TmuxNotFound_ViaSentinel(t *testing.T) {
	t.Parallel()

	master := newFakePTYMaster()
	log := &fakeLogCapture{}

	// Wrap ErrControlModeBinaryNotFound directly — this hits the
	// errors.Is(err, ErrControlModeBinaryNotFound) branch in
	// controlModeFailureLogMsg, not the strings.Contains fallback.
	sentinelErr := fmt.Errorf("%w: %w: synthetic LookPath failure",
		tmux.ErrControlModeUnavailable,
		tmux.ErrControlModeBinaryNotFound,
	)

	sc := newTestSessionConnector(
		t,
		[]tmux.Option{
			fakeExecFuncErr(sentinelErr),
		},
		[]tmux.PTYProxyOption{
			fakePTYAlloc(master, 33333),
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
		t.Fatalf("Connect: %v; want nil (PTY fallback should succeed for EC-002 via sentinel)", err)
	}

	if !sc.InPTYMode() {
		t.Error("InPTYMode() = false; want true after tmux-not-found via sentinel (EC-002)")
	}

	// EC-002 log requirement via errors.Is path: "tmux binary not found; using PTY proxy"
	if !log.HasLine("tmux binary not found; using PTY proxy") {
		t.Errorf("log lines = %v; want line containing 'tmux binary not found; using PTY proxy' (BC-2.04.002 EC-002 via errors.Is sentinel)", log.Lines())
	}
}

// -- EC-003: mid-session control mode loss (3 reconnect attempts) -----------

// TestSessionConnector_MidSessionFallback_ReconnectAttempts verifies that
// when tmux control mode drops mid-session (ErrControlModeDropped via Err()),
// the SessionConnector calls the injected factory exactly maxReconnectAttempts
// (3) times before switching to PTY proxy mode (BC-2.04.002 EC-003; ADR-010).
//
// Fix for C-001: the previous version defined reconnectFn but never wired it
// into the production path — the connector used sc.ctrl.Connect internally.
// This version uses WithControlModeFactory to inject a counting factory so
// that the 3-attempt assertion exercises the real reconnect loop.
//
// Hermetic: initial control-mode stream EOF triggers ErrControlModeDropped;
// the injected factory returns ErrControlModeUnavailable on every call;
// PTY alloc succeeds. No real tmux or PTY forked.
func TestSessionConnector_MidSessionFallback_ReconnectAttempts(t *testing.T) {
	t.Parallel()

	master := newFakePTYMaster()
	log := &fakeLogCapture{}

	// Track how many times the factory is invoked during the reconnect loop.
	var factoryMu sync.Mutex
	factoryCalls := 0
	factory := func(_ context.Context) (*tmux.ControlMode, error) {
		factoryMu.Lock()
		factoryCalls++
		factoryMu.Unlock()
		// All reconnects fail — forces eventual PTY fallback per BC-2.04.002 EC-003.
		return nil, fmt.Errorf("%w: reconnect attempt failed (synthetic)", tmux.ErrControlModeUnavailable)
	}

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
	// WithControlModeFactory injects the counting factory into watchAndFallback
	// so that reconnect attempts are observable (C-001 fix).
	sc := tmux.NewSessionConnector(ctrl, ptyProxy, tmux.WithControlModeFactory(factory))

	t.Cleanup(func() {
		_ = sc.Close()
		_ = master.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initial connect succeeds (fake stream is valid).
	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v; want nil (initial connect must succeed)", err)
	}

	// Wait for mid-session fallback to complete. The fake stream has an
	// immediate EOF, so ErrControlModeDropped is dispatched and watched.
	// After 3 failed factory calls, PTY fallback activates.
	deadline := time.Now().Add(2 * time.Second)
	for !sc.InPTYMode() {
		if time.Now().After(deadline) {
			t.Fatal("InPTYMode() never became true within 2s; mid-session PTY fallback did not activate")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// PRIMARY ASSERTION (BC-2.04.002 EC-003): factory called exactly 3 times.
	factoryMu.Lock()
	got := factoryCalls
	factoryMu.Unlock()
	if got != 3 {
		t.Errorf("factory called %d times; want exactly 3 (BC-2.04.002 EC-003 maxReconnectAttempts)", got)
	}

	// EC-003 log requirement: "tmux control mode lost; falling back to PTY proxy"
	if !log.HasLine("tmux control mode lost; falling back to PTY proxy") {
		t.Errorf("log lines = %v; want line containing 'tmux control mode lost; falling back to PTY proxy' (BC-2.04.002 EC-003)", log.Lines())
	}
}

// -- No auto-upgrade from PTY back to control mode -------------------------

// TestSessionConnector_NoAutoUpgrade_AfterFallback verifies that once the
// SessionConnector enters PTY proxy mode, the injected factory is NOT called
// again — no background goroutine retries control mode after fallback
// (BC-2.04.002 invariant; ADR-010 "no auto-upgrade"; S-3.01b task 9).
//
// Fix for M-003: the previous version relied on a 100ms time.Sleep to
// "observe" that InPTYMode stayed true — this is a weak temporal assertion
// (sleep proves nothing about long-term invariance). This version uses
// WithControlModeFactory to inject a counting factory. After PTY fallback
// is confirmed, we wait an additional period and assert the factory call
// count has not increased. A sticky call count proves the no-auto-upgrade
// invariant structurally.
//
// Strategy: drive into PTY mode via a mid-session EOF (same as EC-003 test).
// Once InPTYMode() is true, snapshot factoryCalls. Sleep 200ms to give any
// hypothetical "auto-upgrade" goroutine time to fire. Assert factory call
// count is unchanged. Assert InPTYMode() is still true.
//
// Hermetic: fake initial control-mode stream; factory always fails; PTY
// alloc succeeds. No real tmux or PTY forked.
func TestSessionConnector_NoAutoUpgrade_AfterFallback(t *testing.T) {
	t.Parallel()

	master := newFakePTYMaster()
	log := &fakeLogCapture{}

	// Count every factory invocation. Post-fallback count must not increase.
	var factoryMu sync.Mutex
	factoryCalls := 0
	factory := func(_ context.Context) (*tmux.ControlMode, error) {
		factoryMu.Lock()
		factoryCalls++
		factoryMu.Unlock()
		return nil, fmt.Errorf("%w: synthetic reconnect failure", tmux.ErrControlModeUnavailable)
	}

	// Initial stream EOFs immediately to trigger ErrControlModeDropped.
	initialStream := fakeControlOutput(
		"%begin 2000000000 0 1",
		"%end 2000000000 0 1",
	)

	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)
	ctrl := tmux.New(pub, ds, fakeExecFunc(initialStream))
	ptyProxy := tmux.NewPTYProxy(pub, ds,
		fakePTYAlloc(master, 44444),
		tmux.WithLogger(log),
	)
	sc := tmux.NewSessionConnector(ctrl, ptyProxy, tmux.WithControlModeFactory(factory))

	t.Cleanup(func() {
		_ = sc.Close()
		_ = master.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v; want nil (initial connect must succeed)", err)
	}

	// Wait for PTY fallback to engage (same poll pattern as EC-003 test).
	deadline := time.Now().Add(2 * time.Second)
	for !sc.InPTYMode() {
		if time.Now().After(deadline) {
			t.Fatal("InPTYMode() never became true within 2s; PTY fallback did not activate")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Snapshot factory call count at the moment fallback is confirmed.
	factoryMu.Lock()
	callsAtFallback := factoryCalls
	factoryMu.Unlock()

	// Give any hypothetical auto-upgrade goroutine time to fire.
	// ADR-010 "no auto-upgrade" means this window must remain quiet.
	time.Sleep(200 * time.Millisecond)

	// STRUCTURAL ASSERTION (ADR-010; BC-2.04.002 invariant): factory must NOT
	// be called again after PTY fallback is active.
	factoryMu.Lock()
	callsAfterWait := factoryCalls
	factoryMu.Unlock()
	if callsAfterWait != callsAtFallback {
		t.Errorf(
			"factory called %d additional times after PTY fallback; want 0 (ADR-010 no-auto-upgrade)",
			callsAfterWait-callsAtFallback,
		)
	}

	// InPTYMode must remain sticky true (BC-2.04.002 invariant; S-3.01b task 9).
	if !sc.InPTYMode() {
		t.Error("InPTYMode() = false after fallback + 200ms wait; expected sticky true (ADR-010)")
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

// -- M-003: SessionConnector.Err() channel -----------------------------------

// TestSessionConnector_Err_MidSessionPTYFallbackFail verifies that when
// tmux control mode drops mid-session (ErrControlModeDropped) AND the
// subsequent PTY fallback also fails (ErrPTYDeviceUnavailable), the
// SessionConnector surfaces the failure via the Err() channel
// (M-003; BC-2.04.002 invariant 3 — never silent).
//
// This exercises the "both paths down" undefined-state scenario where
// the operator must be notified that the access node has lost all viable
// connection paths.
//
// Hermetic: initial control-mode stream EOF triggers ErrControlModeDropped
// (no factory → immediate PTY fallback); PTY alloc returns ErrPTYDeviceUnavailable.
// No real tmux or PTY forked.
func TestSessionConnector_Err_MidSessionPTYFallbackFail(t *testing.T) {
	t.Parallel()

	log := &fakeLogCapture{}

	// Initial stream EOFs immediately to trigger ErrControlModeDropped
	// without any reconnect factory (nil factory → immediate PTY fallback).
	initialStream := fakeControlOutput(
		"%begin 3000000000 0 1",
		"%end 3000000000 0 1",
		// EOF immediately follows.
	)

	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)
	ctrl := tmux.New(pub, ds, fakeExecFunc(initialStream))

	// PTY alloc always fails — simulates no PTY device available.
	ptyProxy := tmux.NewPTYProxy(pub, ds,
		fakePTYAllocErr(fmt.Errorf("%w: /dev/ptmx: no such device", tmux.ErrPTYDeviceUnavailable)),
		tmux.WithLogger(log),
	)

	// No factory → watchAndFallback falls back to PTY immediately on drop.
	sc := tmux.NewSessionConnector(ctrl, ptyProxy)
	t.Cleanup(func() { _ = sc.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initial connect succeeds (fake stream is valid).
	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v; want nil (initial control mode must succeed)", err)
	}

	// Wait for the Err() channel to receive the PTY failure.
	// The fake stream has an immediate EOF, so ErrControlModeDropped is
	// dispatched; watchAndFallback attempts PTY connect (which fails);
	// then sends ErrPTYDeviceUnavailable to Err().
	select {
	case err, ok := <-sc.Err():
		if !ok {
			// Channel closed without an error — only valid if Close() raced.
			// In this test, Close() has not been called yet, so this is a failure.
			t.Fatal("Err() channel closed without an error; want ErrPTYDeviceUnavailable")
		}
		if !errors.Is(err, tmux.ErrPTYDeviceUnavailable) {
			t.Errorf("Err() = %v; want errors.Is(_, ErrPTYDeviceUnavailable)", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Err() channel did not receive an error within 2s; mid-session fallback failure not surfaced")
	}
}

// TestSessionConnector_Err_ClosedOnNormalShutdown verifies that the Err()
// channel is closed (not written to) on a graceful Close(), so callers
// ranging over it unblock without receiving an error (M-003 normal-path).
func TestSessionConnector_Err_ClosedOnNormalShutdown(t *testing.T) {
	t.Parallel()

	master := newFakePTYMaster()
	initialStream := fakeControlOutput(
		"%begin 4000000000 0 1",
		"%end 4000000000 0 1",
	)

	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)
	ctrl := tmux.New(pub, ds, fakeExecFunc(initialStream))
	ptyProxy := tmux.NewPTYProxy(pub, ds, fakePTYAlloc(master, 55555))
	sc := tmux.NewSessionConnector(ctrl, ptyProxy)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v; want nil", err)
	}

	// Graceful Close — Err() channel must close cleanly.
	if err := sc.Close(); err != nil {
		t.Logf("Close: %v (non-nil close error acceptable)", err)
	}
	_ = master.Close()

	// Assert Err() is closed by draining it in a non-blocking way.
	// After Close(), the channel must be closed (ok == false), not blocking.
	select {
	case err, ok := <-sc.Err():
		if ok {
			t.Errorf("Err() sent %v after graceful Close; want channel closed with no error", err)
		}
		// ok == false: channel is closed, no error — correct.
	case <-time.After(500 * time.Millisecond):
		t.Error("Err() channel not closed within 500ms after sc.Close(); want closed on graceful shutdown")
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
