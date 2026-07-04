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

// TestPTYProxy_EC001_OldTmuxVersion_ViaAsyncClassify verifies the async
// classification path for EC-001: Connect succeeds (stdout opens), but the
// classifier later delivers ErrControlModeUnsupportedFlag via classifyCh.
// dispatchLoop must surface ErrControlModeUnsupportedFlag on ctrl.Err()
// (not ErrControlModeDropped) per the pass-7 H-001/M-001 fix.
//
// This test is a regression guard for the production async path; the existing
// TestPTYProxy_EC001_OldTmuxVersion covers the sync error-from-execFn path.
//
// Hermetic: ControlMode is constructed directly (not via SessionConnector)
// to isolate the classification-supersedes-drop contract at the ControlMode
// layer, independent of watchAndFallback routing logic.
func TestPTYProxy_EC001_OldTmuxVersion_ViaAsyncClassify(t *testing.T) {
	t.Parallel()

	// classifyCh is pre-populated with ErrControlModeUnsupportedFlag and closed.
	// stdout is empty → immediate EOF → dispatchLoop hits unexpected-exit path.
	// With the pass-7 fix, classifyCh is drained before writing to errCh.
	classifyCh := make(chan error, 1)
	classifyCh <- tmux.ErrControlModeUnsupportedFlag
	close(classifyCh)

	fake := tmux.WithExecFunc(func(_ context.Context) (io.WriteCloser, io.ReadCloser, <-chan error, error) {
		return nopWriteCloser{}, nopCloser{strings.NewReader("")}, classifyCh, nil
	})

	cm, _ := newTestControlWithOpts(t, fake) //nolint:dogsled // publisher unused here
	t.Cleanup(func() { _ = cm.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cm.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v; want nil (control mode starts OK; classification is async)", err)
	}

	// Drain ctrl.Err(). With the pass-7 fix, classification supersedes the drop
	// sentinel, so we expect ErrControlModeUnsupportedFlag — not ErrControlModeDropped.
	select {
	case err := <-cm.Err():
		if !errors.Is(err, tmux.ErrControlModeUnsupportedFlag) {
			t.Errorf(
				"ctrl.Err() = %v; want ErrControlModeUnsupportedFlag "+
					"(classification must supersede ErrControlModeDropped; pass-7 H-001/M-001)",
				err,
			)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no error on ctrl.Err() within 2s; async classification did not fire")
	}
}

// TestSessionConnector_AsyncEC001_PTYFallback verifies that when ctrl.Err()
// delivers ErrControlModeUnsupportedFlag (EC-001 async path — subprocess
// started then exited with flag-rejection classification), SessionConnector:
//
//  1. Emits the BC-2.04.002 EC-001 log message "tmux version does not support -CC flag"
//  2. Invokes PTY fallback immediately (no reconnect attempts)
//  3. Sets sc.inPTYMode = true
//
// This is the pass-8 H-001 regression guard — prior to fix, watchAndFallback
// filtered ErrControlModeUnsupportedFlag out silently (continue on
// !ErrControlModeDropped).
//
// Hermetic: WithExecFunc injects a fake that delivers ErrControlModeUnsupportedFlag
// via the classifyCh with empty stdout (no real tmux or PTY).
func TestSessionConnector_AsyncEC001_PTYFallback(t *testing.T) {
	t.Parallel()

	// classifyCh delivers ErrControlModeUnsupportedFlag and is then closed,
	// matching the production dispatchLoop exit path (pass-7).
	classifyCh := make(chan error, 1)
	classifyCh <- tmux.ErrControlModeUnsupportedFlag
	close(classifyCh)

	fake := tmux.WithExecFunc(func(_ context.Context) (io.WriteCloser, io.ReadCloser, <-chan error, error) {
		// Empty stdout → immediate EOF → dispatchLoop hits unexpected-exit path,
		// which drains classifyCh and delivers ErrControlModeUnsupportedFlag.
		return nopWriteCloser{}, nopCloser{strings.NewReader("")}, classifyCh, nil
	})

	master := newFakePTYMaster()
	log := &fakeLogCapture{}

	// Factory should NEVER be called for EC-001 (immediate fallback, no reconnect).
	var factoryMu sync.Mutex
	factoryCalls := 0
	factory := func(_ context.Context) (*tmux.ControlMode, error) {
		factoryMu.Lock()
		factoryCalls++
		factoryMu.Unlock()
		return nil, fmt.Errorf("factory must not be called for EC-001")
	}

	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)
	ctrl := tmux.New(pub, ds, fake)
	ptyProxy := tmux.NewPTYProxy(pub, ds,
		fakePTYAlloc(master, 99999),
		tmux.WithLogger(log),
	)
	sc := tmux.NewSessionConnector(ctrl, ptyProxy, tmux.WithControlModeFactory(factory))
	t.Cleanup(func() {
		_ = sc.Close()
		_ = master.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initial connect succeeds (the exec func returns nil error; fake stream
	// triggers async classification).
	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v; want nil (control mode starts OK; EC-001 classification is async)", err)
	}

	// Wait for PTY fallback to engage (poll InPTYMode with deadline).
	deadline := time.Now().Add(2 * time.Second)
	for !sc.InPTYMode() {
		if time.Now().After(deadline) {
			t.Fatal("PTY fallback did not engage within 2s after EC-001 async classification")
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Assert factory was NOT called (immediate fallback, no reconnect).
	factoryMu.Lock()
	got := factoryCalls
	factoryMu.Unlock()
	if got != 0 {
		t.Errorf("factory called %d times; want 0 for EC-001 (no reconnect — same binary will reject again)", got)
	}

	// Assert log contains the EC-001 canonical message.
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

// -- L-003 (pass-9): successful factory reconnect path ----------------------

// TestSessionConnector_FactoryReconnectSucceeds verifies the BC-2.04.002 EC-003
// happy path: mid-session control mode drop → factory returns a freshly-
// Connected ControlMode → watchAndFallback swaps in the new ctrl + spawns
// recursive watcher → sc.InPTYMode() remains false (still in control mode).
//
// Closes the L-003 (pass-9) coverage gap.
func TestSessionConnector_FactoryReconnectSucceeds(t *testing.T) {
	t.Parallel()

	// Track factory calls.
	var factoryMu sync.Mutex
	factoryCalls := 0

	// The factory returns a NEW ControlMode backed by a pipe that stays open
	// (no EOF) for the duration of the test.  The pipe's write end is kept
	// alive via t.Cleanup so the new ctrl never drops within the test window.
	factory := func(ctx context.Context) (*tmux.ControlMode, error) {
		factoryMu.Lock()
		factoryCalls++
		factoryMu.Unlock()

		// Pipe: pr is the fake stdout read by ControlMode; pw is held open.
		pr, pw := io.Pipe()
		t.Cleanup(func() { _ = pw.Close() })

		// The new ctrl's classifyCh is never written to — just pre-closed so the
		// dispatcher doesn't block waiting for classification.
		newClassifyCh := make(chan error)
		close(newClassifyCh)

		newExec := tmux.WithExecFunc(func(_ context.Context) (io.WriteCloser, io.ReadCloser, <-chan error, error) {
			return nopWriteCloser{}, pr, newClassifyCh, nil
		})

		keys := admission.NewAdmittedKeySet()
		pub := session.NewPublisher(keys)
		ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)
		newCM := tmux.New(pub, ds, newExec)
		if err := newCM.Connect(ctx); err != nil {
			_ = pr.Close()
			return nil, fmt.Errorf("factory newCM.Connect: %w", err)
		}
		return newCM, nil
	}

	// Initial ctrl: stream with a minimal valid header that EOFs immediately,
	// triggering ErrControlModeDropped via the dispatchLoop unexpected-exit path.
	initialStream := fakeControlOutput(
		"%begin 5000000000 0 1",
		"%end 5000000000 0 1",
		// EOF immediately follows — causes ErrControlModeDropped.
	)

	master := newFakePTYMaster()
	log := &fakeLogCapture{}

	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)
	ctrl := tmux.New(pub, ds, fakeExecFunc(initialStream))
	ptyProxy := tmux.NewPTYProxy(pub, ds,
		fakePTYAlloc(master, 66666),
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
		t.Fatalf("sc.Connect: %v", err)
	}

	// Wait for factory to be called exactly once (initial drop → reconnect succeeds).
	deadline := time.Now().Add(2 * time.Second)
	for {
		factoryMu.Lock()
		calls := factoryCalls
		factoryMu.Unlock()
		if calls >= 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("factory was not called within 2s")
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Give watchAndFallback a moment to complete the swap and spawn the
	// recursive watcher on the new ctrl.
	time.Sleep(50 * time.Millisecond)

	// PRIMARY ASSERTION: successful reconnect means we stayed in control mode.
	if sc.InPTYMode() {
		t.Error("sc.InPTYMode() = true after successful factory reconnect; want false (still in control mode)")
	}

	// SECONDARY ASSERTION: factory called exactly once (one successful reconnect
	// — no further retries needed).
	factoryMu.Lock()
	got := factoryCalls
	factoryMu.Unlock()
	if got != 1 {
		t.Errorf("factoryCalls = %d after successful reconnect; want 1", got)
	}
}

// -- SendInput coverage (F-C-5 pass-3) -----------------------------------------

// capturingPTYMaster is a fake PTY master that records Write calls for
// assertion in PTYProxy.SendInput tests. Read blocks until Close is called
// (so the ioRelay goroutine stays alive during the test).
type capturingPTYMaster struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	cond   *sync.Cond
	closed bool
}

func newCapturingPTYMaster() *capturingPTYMaster {
	m := &capturingPTYMaster{}
	m.cond = sync.NewCond(&m.mu)
	return m
}

func (m *capturingPTYMaster) Write(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.buf.Write(p)
}

func (m *capturingPTYMaster) Read(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for !m.closed {
		m.cond.Wait()
	}
	return 0, io.EOF
}

func (m *capturingPTYMaster) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	m.cond.Broadcast()
	return nil
}

func (m *capturingPTYMaster) Written() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]byte, m.buf.Len())
	copy(out, m.buf.Bytes())
	return out
}

// TestPTYProxy_SendInput_HappyPath verifies that PTYProxy.SendInput writes the
// payload to the PTY master when the proxy is connected (F-C-5;
// session.KeystrokeSink contract).
//
// Hermetic: WithPTYAllocFunc injects a capturingPTYMaster so we can observe
// the bytes written. No real PTY.
func TestPTYProxy_SendInput_HappyPath(t *testing.T) {
	t.Parallel()

	master := newCapturingPTYMaster()
	t.Cleanup(func() { _ = master.Close() })

	pty, _ := newTestPTYProxy(t,
		tmux.WithPTYAllocFunc(func() (io.ReadWriteCloser, int, error) {
			return master, 12345, nil
		}),
	)
	t.Cleanup(func() { _ = pty.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := pty.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	payload := []byte("hello\r")
	if err := pty.SendInput(payload); err != nil {
		t.Fatalf("SendInput: unexpected error: %v", err)
	}

	got := master.Written()
	if string(got) != string(payload) {
		t.Errorf("master.Written() = %q; want %q", got, payload)
	}
}

// TestPTYProxy_SendInput_AfterClose_ReturnsSentinel verifies that
// PTYProxy.SendInput returns ErrPTYProxyClosed after the proxy has been closed
// (F-C-5; F-H-1 sentinel inspection via errors.Is).
func TestPTYProxy_SendInput_AfterClose_ReturnsSentinel(t *testing.T) {
	t.Parallel()

	master := newFakePTYMaster()
	pty, _ := newTestPTYProxy(t, fakePTYAlloc(master, 9999))
	t.Cleanup(func() { _ = master.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := pty.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if err := pty.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	err := pty.SendInput([]byte("after-close"))
	if !errors.Is(err, tmux.ErrPTYProxyClosed) {
		t.Errorf("SendInput after Close: got %v; want errors.Is(_, ErrPTYProxyClosed)", err)
	}
}

// TestSessionConnector_SendInput_ControlMode_Dispatches verifies that when
// SessionConnector is in control mode (not PTY mode), SendInput delegates to
// the active ControlMode's SendInput (F-C-5; session.KeystrokeSink contract).
//
// Hermetic: WithExecFunc injects a capturingWriteCloser as stdin; WithPTYAllocFunc
// is unused (stays in control mode). Asserts the payload arrives at the fake stdin.
func TestSessionConnector_SendInput_ControlMode_Dispatches(t *testing.T) {
	t.Parallel()

	stdin := &bytes.Buffer{}
	var stdinMu sync.Mutex
	var stdinClosed bool

	captureExec := tmux.WithExecFunc(func(_ context.Context) (io.WriteCloser, io.ReadCloser, <-chan error, error) {
		return &bufWriteCloser{buf: stdin, mu: &stdinMu, closed: &stdinClosed},
			nopCloser{strings.NewReader(
				"%begin 9000000000 0 1\n%end 9000000000 0 1\n",
			)},
			closedNilChan(),
			nil
	})

	master := newFakePTYMaster()
	log := &fakeLogCapture{}

	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)
	ctrl := tmux.New(pub, ds, captureExec)
	ptyProxy := tmux.NewPTYProxy(pub, ds,
		fakePTYAlloc(master, 77777),
		tmux.WithLogger(log),
	)
	sc := tmux.NewSessionConnector(ctrl, ptyProxy)
	t.Cleanup(func() {
		_ = sc.Close()
		_ = master.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Must be in control mode (not PTY mode).
	if sc.InPTYMode() {
		t.Fatal("InPTYMode() = true; want false (control mode should be active)")
	}

	payload := []byte("ctrl-mode-input\r")
	if err := sc.SendInput(payload); err != nil {
		t.Fatalf("SendInput (control mode): unexpected error: %v", err)
	}

	stdinMu.Lock()
	got := make([]byte, stdin.Len())
	copy(got, stdin.Bytes())
	stdinMu.Unlock()

	// stdin also receives the list-sessions command written by Connect.
	// Assert the payload was written (it must be present in stdin output).
	if !strings.Contains(string(got), string(payload)) {
		t.Errorf("stdin.Written() = %q; want to contain %q", got, payload)
	}
}

// TestSessionConnector_SendInput_AfterClose_ReturnsSentinel verifies that
// SessionConnector.SendInput returns ErrSessionConnectorClosed after Close
// (F-C-5; F-H-1 sentinel inspection via errors.Is).
func TestSessionConnector_SendInput_AfterClose_ReturnsSentinel(t *testing.T) {
	t.Parallel()

	master := newFakePTYMaster()
	sc := newTestSessionConnector(
		t,
		[]tmux.Option{
			fakeExecFuncErr(fmt.Errorf("%w: tmux not found", tmux.ErrControlModeUnavailable)),
		},
		[]tmux.PTYProxyOption{
			fakePTYAlloc(master, 88888),
		},
	)
	t.Cleanup(func() {
		_ = sc.Close()
		_ = master.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if err := sc.Close(); err != nil {
		t.Logf("Close: %v (non-nil close acceptable)", err)
	}

	err := sc.SendInput([]byte("after-close"))
	if !errors.Is(err, tmux.ErrSessionConnectorClosed) {
		t.Errorf("SendInput after Close: got %v; want errors.Is(_, ErrSessionConnectorClosed)", err)
	}
}

// bufWriteCloser is a goroutine-safe io.WriteCloser wrapping a bytes.Buffer.
// Used in TestSessionConnector_SendInput_ControlMode_Dispatches to capture
// stdin writes from ControlMode.SendInput.
type bufWriteCloser struct {
	buf    *bytes.Buffer
	mu     *sync.Mutex
	closed *bool
}

func (b *bufWriteCloser) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *bufWriteCloser) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	*b.closed = true
	return nil
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
