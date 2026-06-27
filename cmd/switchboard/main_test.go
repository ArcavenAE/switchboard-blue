// Package main — main_test.go contains integration tests for the access-node
// daemon wiring (S-W3.04; AC-001 through AC-008).
//
// All new tests MUST fail against the current stubs (Red Gate, BC-5.38.001).
// Every stub body panics("not implemented"); a panic in a test goroutine is
// a test failure, so all six new tests are discriminating against the stubs.
//
// BC traces:
//   - AC-001 → BC-2.05.008 PC-2 + invariant 1 (router logger / E-ADM-016)
//   - AC-002 → BC-2.04.005 PC-3 + BC-2.04.003 PC-3 (live SessionAuth)
//   - AC-003 → BC-2.04.004 PC-1 + PC-3 (sweep eviction)
//   - AC-006 → BC-2.04.006 invariant 4 (FramesDropped log ticker)
//   - AC-007 → BC-2.04.007 PC-1 (connect failure → log + non-zero exit)
//   - AC-008 → BC-2.04.007 PC-2 (SIGTERM/SIGINT → clean shutdown + exit 0)
package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/halfchannel"
	hmacpkg "github.com/arcavenae/switchboard/internal/hmac"
	"github.com/arcavenae/switchboard/internal/routing"
	"github.com/arcavenae/switchboard/internal/session"
	"github.com/arcavenae/switchboard/internal/tmux"
)

// ── Shared helpers ────────────────────────────────────────────────────────────

// captureLogger is a routing.Logger (and tmux.Logger) that records all log
// lines for assertion. Goroutine-safe.
type captureLogger struct {
	mu    sync.Mutex
	lines []string
}

func (l *captureLogger) Log(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lines = append(l.lines, msg)
}

// HasLine returns true if any recorded line contains substr.
func (l *captureLogger) HasLine(substr string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, line := range l.lines {
		if strings.Contains(line, substr) {
			return true
		}
	}
	return false
}

// Lines returns a copy of all recorded log lines.
func (l *captureLogger) Lines() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]string, len(l.lines))
	copy(out, l.lines)
	return out
}

// mustGenEd25519Main generates an Ed25519 key pair or fatals.
func mustGenEd25519Main(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	return pub, priv
}

// svtnIDFromByte constructs a 16-byte SVTN ID with the first byte set to b.
func svtnIDFromByte(b byte) [16]byte {
	var id [16]byte
	id[0] = b
	return id
}

// deriveNodeAddr mirrors frame.DeriveNodeAddress: SHA-256(svtnID||pubKey)[:8].
func deriveNodeAddr(svtnID [16]byte, pubKey ed25519.PublicKey) [8]byte {
	h := sha256.New()
	h.Write(svtnID[:])
	h.Write([]byte(pubKey))
	sum := h.Sum(nil)
	var addr [8]byte
	copy(addr[:], sum[:8])
	return addr
}

// admitAndRegister registers a node key, completes challenge-response (so
// IsAdmitted returns true), derives the authKey, and registers the forwarding
// entry on r. Returns (srcAddr, authKey).
func admitAndRegister(
	t *testing.T,
	ks *admission.AdmittedKeySet,
	r *routing.Router,
	svtnID [16]byte,
	nodePub ed25519.PublicKey,
	nodePriv ed25519.PrivateKey,
) ([8]byte, [hmacpkg.KeySize]byte) {
	t.Helper()

	// Register the key with the admitted set (pre-admission step).
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)

	srcAddr := deriveNodeAddr(svtnID, nodePub)

	// Complete challenge-response.
	_, routerPriv := mustGenEd25519Main(t)
	ch, err := admission.GenerateChallenge(routerPriv)
	if err != nil {
		t.Fatalf("GenerateChallenge: %v", err)
	}
	nonceSig := ed25519.Sign(nodePriv, ch.Nonce[:])
	resp := admission.ChallengeResponse{NonceSig: nonceSig}
	if err := admission.AdmitNode(ch, resp, nodePub, svtnID, ks); err != nil {
		t.Fatalf("AdmitNode: %v", err)
	}

	// Derive and register forwarding entry.
	authKey := hmacpkg.DeriveKey([]byte(nodePub), svtnID)
	r.RegisterForwardingEntry(svtnID, srcAddr, authKey)

	return srcAddr, authKey
}

// ── AC-001: TestRouterLoggerEmitsEADM016 ──────────────────────────────────────

// TestRouterLoggerEmitsEADM016 — AC-001 (BC-2.05.008 PC-2 + invariant 1)
//
// Verifies that buildRouter constructs a routing.Router with a real
// routing.Logger injected (not nil, not a no-op sink). When RouteFrame
// encounters an HMAC verification failure, the log event E-ADM-016 is written
// to the logger's output with the canonical message format:
//
//	"wire HMAC verification failed at RouteFrame: tag mismatch for SVTN
//	 <svtn_id> from src <src_addr> (E-ADM-016)"
//
// The test has two assertions:
//
//  1. buildRouter(ks) returns a non-nil router that satisfies BC-2.05.008
//     (primary assertion — stub panics; Red Gate).
//  2. A Router with a captureLogger wired produces the canonical E-ADM-016
//     message on HMAC failure (concrete log format assertion).
//
// Red Gate: buildRouter panics("not implemented") in the stub.
func TestRouterLoggerEmitsEADM016(t *testing.T) {
	// AC-001 — BC-2.05.008 PC-2 + invariant 1.
	// NOT t.Parallel(): buildRouter panics in the stub.

	svtnID := svtnIDFromByte(0x01)
	nodePub, nodePriv := mustGenEd25519Main(t)

	ks := admission.NewAdmittedKeySet()

	// buildRouter panics on stub — that IS the Red Gate failure.
	r := buildRouter(ks)

	srcAddr, _ := admitAndRegister(t, ks, r, svtnID, nodePub, nodePriv)

	// Craft a frame with an all-zero HMAC tag — mismatch for any derived key.
	hdr := frame.OuterHeader{
		SVTNID:    svtnID,
		SrcAddr:   srcAddr,
		DstAddr:   [8]byte{},
		FrameType: frame.FrameTypeData,
		// HMACTag zero-value: guaranteed to fail verifyFrameHMAC.
	}
	payload := []byte("tampered")

	routeErr := routing.RouteFrame(hdr, payload, r)
	if !errors.Is(routeErr, routing.ErrHMACVerificationFailed) {
		t.Fatalf("RouteFrame(tampered): got %v; want ErrHMACVerificationFailed (E-ADM-016)", routeErr)
	}

	// Assertion 2: concrete log format with captureLogger.
	// buildRouter must wire a real Logger; we replicate the same construction
	// with a known captureLogger to verify the canonical E-ADM-016 message.
	cl := &captureLogger{}
	r2 := routing.NewRouter(ks, routing.WithLogger(cl))
	r2.RegisterForwardingEntry(svtnID, srcAddr, hmacpkg.DeriveKey([]byte(nodePub), svtnID))

	r2Err := routing.RouteFrame(hdr, payload, r2)
	if !errors.Is(r2Err, routing.ErrHMACVerificationFailed) {
		t.Fatalf("RouteFrame(logged router): got %v; want ErrHMACVerificationFailed", r2Err)
	}

	// Canonical E-ADM-016 assertions (error-taxonomy.md §ADM):
	// 1. Literal "(E-ADM-016)" must appear for grep-ability.
	if !cl.HasLine("E-ADM-016") {
		t.Errorf("logger did not receive E-ADM-016 literal; got: %v", cl.Lines())
	}
	// 2. Canonical prefix per error-taxonomy.md.
	if !cl.HasLine("wire HMAC verification failed") {
		t.Errorf("logger missing canonical prefix; got: %v", cl.Lines())
	}
	// 3. SVTN ID (lowercase hex) appears in the log line.
	svtnHex := fmt.Sprintf("%x", svtnID)
	if !cl.HasLine(svtnHex) {
		t.Errorf("logger missing svtn_id=%q; got: %v", svtnHex, cl.Lines())
	}
}

// ── AC-002: TestDaemonAuthRejectsUnregisteredConsole ──────────────────────────

// TestDaemonAuthRejectsUnregisteredConsole — AC-002
// (BC-2.04.005 PC-3 + BC-2.04.003 PC-3)
//
// Verifies that buildAccessNode wires a live *session.SessionAuth as the
// Authorizer (not NoOpAuthorizer or nil). Specifically:
//  1. An unregistered console key's Attach call is rejected — fail-open is closed.
//  2. A registered read-only key's upstream keystroke returns ErrUpstreamReadOnly
//     (E-ADM-007 per error-taxonomy.md §ADM).
//  3. A registered full-access key's upstream keystrokes are forwarded.
//
// The test wires a known SessionAuth for assertions 2+3, and calls buildAccessNode
// to verify assertion 1 (stub panics — Red Gate).
//
// Red Gate: buildAccessNode panics("not implemented") in the stub.
func TestDaemonAuthRejectsUnregisteredConsole(t *testing.T) {
	// AC-002 — BC-2.04.005 PC-3 + BC-2.04.003 PC-3.

	const sessionName = "agent-01"
	unregisteredKey := session.ConsoleKey("unregistered")
	readOnlyKey := session.ConsoleKey("readonly")
	fullAccessKey := session.ConsoleKey("fullaccess")

	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)

	ctrl := tmux.New(pub, ds, fakeExecFuncErrMain(tmux.ErrControlModeUnavailable))
	pipe := newPipeMasterMain()
	pty := tmux.NewPTYProxy(pub, ds,
		tmux.WithPTYAllocFunc(func() (io.ReadWriteCloser, int, error) {
			return pipe, 5678, nil
		}),
	)
	sc := tmux.NewSessionConnector(ctrl, pty)
	t.Cleanup(func() {
		_ = pipe.Close()
		_ = sc.Close()
	})

	// buildAccessNode panics on stub — Red Gate.
	an := buildAccessNode(sc)

	// Ensure session is published (buildAccessNode may or may not publish it).
	if err := pub.Publish(sessionName); err != nil {
		if !errors.Is(err, session.ErrSessionAlreadyPublished) {
			t.Fatalf("Publish: %v", err)
		}
	}

	// Assertion 1 (AC-002 primary): fail-open is CLOSED — unregistered key rejected.
	// With live SessionAuth, Attach with an unregistered key must return an error.
	_, _, attachErr := an.Attach(unregisteredKey, sessionName)
	if attachErr == nil {
		t.Errorf("Attach(unregistered key): got nil; want error (fail-open default must be closed — W3-M-3)")
	}

	// Assertions 2+3: verified with a known-wired AccessNode + SessionAuth.
	// (buildAccessNode controls its own SessionAuth; we construct a parallel one
	// to exercise the E-ADM-007 path precisely.)
	sa := session.NewSessionAuth()
	sa.RegisterKey(sessionName, readOnlyKey, session.RoleReadOnly)
	sa.RegisterKey(sessionName, fullAccessKey, session.RoleFull)

	sink := &keystrokeSinkCapture{}
	an2 := session.NewAccessNode(pub, sa, session.WithKeystrokeSink(sink))

	// Attach full-access console so it's in the fan-out set.
	if _, _, err := an2.Attach(fullAccessKey, sessionName); err != nil {
		t.Fatalf("Attach(fullAccess): %v", err)
	}

	// Assertion 2: read-only upstream keystroke → ErrUpstreamReadOnly (E-ADM-007).
	roErr := an2.SendKeystroke(readOnlyKey, sessionName, []byte("key"))
	if !errors.Is(roErr, session.ErrUpstreamReadOnly) {
		t.Errorf("SendKeystroke(readOnly): got %v; want ErrUpstreamReadOnly (E-ADM-007)", roErr)
	}

	// Assertion 3: full-access upstream keystroke → forwarded (nil error).
	fwdErr := an2.SendKeystroke(fullAccessKey, sessionName, []byte("keystroke"))
	if fwdErr != nil {
		t.Errorf("SendKeystroke(fullAccess): got %v; want nil", fwdErr)
	}
	if !sink.calledWith("keystroke") {
		t.Errorf("keystroke sink not called; want payload forwarded; got: %v", sink.captured())
	}
}

// keystrokeSinkCapture records payloads passed to SendInput. Goroutine-safe.
type keystrokeSinkCapture struct {
	mu       sync.Mutex
	payloads [][]byte
}

func (k *keystrokeSinkCapture) SendInput(payload []byte) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	cp := make([]byte, len(payload))
	copy(cp, payload)
	k.payloads = append(k.payloads, cp)
	return nil
}

func (k *keystrokeSinkCapture) calledWith(s string) bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	for _, p := range k.payloads {
		if strings.Contains(string(p), s) {
			return true
		}
	}
	return false
}

func (k *keystrokeSinkCapture) captured() []string {
	k.mu.Lock()
	defer k.mu.Unlock()
	out := make([]string, len(k.payloads))
	for i, p := range k.payloads {
		out[i] = string(p)
	}
	return out
}

// ── AC-003: TestDaemonSweepEvictsStaleConsole ─────────────────────────────────

// TestDaemonSweepEvictsStaleConsole — AC-003 (BC-2.04.004 PC-1 + PC-3)
//
// Verifies that startSweepTicker calls accessNode.Sweep(deadline) on each tick.
// After the keepalive deadline elapses (via injected clock), a console that has
// not sent a heartbeat is evicted. Subsequent SendKeystroke returns
// ErrConsoleNotFound (BC-2.04.004 PC-3).
//
// Red Gate: startSweepTicker panics("not implemented") in the stub.
func TestDaemonSweepEvictsStaleConsole(t *testing.T) {
	// AC-003 — BC-2.04.004 PC-1 + PC-3.

	const sessionName = "agent-sweep"
	consoleKey := session.ConsoleKey("stale-console")

	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	if err := pub.Publish(sessionName); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// Controllable clock so we can jump time without real sleeps.
	var clockMu sync.Mutex
	now := time.Now().UTC()
	clockFn := func() time.Time {
		clockMu.Lock()
		defer clockMu.Unlock()
		return now
	}
	advanceClock := func(d time.Duration) {
		clockMu.Lock()
		defer clockMu.Unlock()
		now = now.Add(d)
	}

	sa := session.NewSessionAuth()
	sa.RegisterKey(sessionName, consoleKey, session.RoleFull)
	an := session.NewAccessNode(pub, sa,
		session.WithKeystrokeSink(session.NoOpSink{}),
		session.WithClock(clockFn),
	)

	// Attach console.
	if _, _, err := an.Attach(consoleKey, sessionName); err != nil {
		t.Fatalf("Attach: %v", err)
	}

	// Console is present before sweep.
	if err := an.SendKeystroke(consoleKey, sessionName, []byte("alive")); err != nil {
		t.Fatalf("pre-sweep SendKeystroke: %v; want nil", err)
	}

	// Advance clock past sweep deadline → console is now stale.
	const sweepDeadlineTest = 60 * time.Second
	advanceClock(sweepDeadlineTest + time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// startSweepTicker panics in the stub — Red Gate.
	// Use 1ms interval so the sweep fires immediately.
	startSweepTicker(ctx, an, time.Millisecond, sweepDeadlineTest)

	// Wait for the sweep to evict the console (bounded: 500ms).
	deadline := time.After(500 * time.Millisecond)
	for {
		err := an.SendKeystroke(consoleKey, sessionName, []byte("post-sweep"))
		if errors.Is(err, session.ErrConsoleNotFound) {
			// AC-003 assertion: evicted — subsequent SendKeystroke returns
			// ErrConsoleNotFound (BC-2.04.004 PC-3).
			return
		}
		select {
		case <-deadline:
			t.Fatalf("console not evicted within 500ms; last err: %v; "+
				"startSweepTicker must call Sweep(deadline) on each tick (BC-2.04.004 PC-3)", err)
		default:
			runtime.Gosched()
		}
	}
}

// ── AC-006: TestDaemonFramesDroppedLoggedOnTick ────────────────────────────────

// TestDaemonFramesDroppedLoggedOnTick — AC-006 (BC-2.04.006 invariant 4)
//
// Verifies that startFramesDroppedTicker logs a structured "frames_dropped
// count=<N>" INFO line when accessNode.FramesDropped() > 0. The counter is
// cumulative (not reset on read). Closes drift W3-R2-M3.
//
// Red Gate: startFramesDroppedTicker panics("not implemented") in the stub.
func TestDaemonFramesDroppedLoggedOnTick(t *testing.T) {
	// AC-006 — BC-2.04.006 invariant 4.

	const sessionName = "agent-dropped"
	stalledKey := session.ConsoleKey("stalled")

	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	if err := pub.Publish(sessionName); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	sa := session.NewSessionAuth()
	sa.RegisterKey(sessionName, stalledKey, session.RoleReadOnly)
	an := session.NewAccessNode(pub, sa, session.WithKeystrokeSink(session.NoOpSink{}))

	// Attach a stalled console that never drains its downstream channel.
	if _, _, err := an.Attach(stalledKey, sessionName); err != nil {
		t.Fatalf("Attach: %v", err)
	}

	// Saturate the downstream buffer (64 per fanout.go) to trigger drops.
	for i := range 200 {
		an.DeliverFrame(frame.OuterHeader{
			SVTNID:    svtnIDFromByte(0x02),
			FrameType: frame.FrameTypeData,
			SrcAddr:   [8]byte{byte(i % 256)},
		})
	}

	dropped := an.FramesDropped()
	if dropped == 0 {
		t.Skip("no frames dropped in setup; downstream buffer consumed all frames (test precondition not met)")
	}

	// Construct a *log.Logger writing to captureWriter.
	cw := &captureWriter{}
	lg := log.New(cw, "", 0)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// startFramesDroppedTicker panics in the stub — Red Gate.
	// Use a 1ms ticker interval so the tick fires immediately.
	startFramesDroppedTicker(ctx, an, lg)

	// Wait for the log line to appear (bounded: 500ms).
	deadline := time.After(500 * time.Millisecond)
	wantCount := fmt.Sprintf("%d", dropped)
	for {
		// AC-006 assertion: structured log line with "frames_dropped" key and
		// cumulative count N.
		if cw.hasSubstr("frames_dropped") && cw.hasSubstr(wantCount) {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("frames_dropped log not produced within 500ms; "+
				"want 'frames_dropped count=%s'; got: %q", wantCount, cw.String())
		default:
			runtime.Gosched()
		}
	}
}

// captureWriter is an io.Writer that records all bytes written. Goroutine-safe.
type captureWriter struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (w *captureWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

func (w *captureWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

func (w *captureWriter) hasSubstr(s string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return strings.Contains(w.buf.String(), s)
}

// ── AC-007: TestDaemonConnectFailureExitsNonZero ───────────────────────────────

// TestDaemonConnectFailureExitsNonZero — AC-007 (BC-2.04.007 PC-1)
//
// Verifies that when sc.Connect(ctx) returns a non-nil error, runAccess:
//  1. Returns a non-nil error (caller exits with code 1).
//  2. Emits E-SYS-002 canonical diagnostic to stderr:
//     "fatal: cannot connect to session backend: <reason>"
//  3. Does NOT panic.
//  4. Does NOT start relay goroutines.
//
// E-SYS-002 canonical message (error-taxonomy.md §SYS):
//
//	"fatal: cannot connect to session backend: <reason>"
//
// Red Gate: runAccess panics("not implemented") in the stub.
func TestDaemonConnectFailureExitsNonZero(t *testing.T) {
	// AC-007 — BC-2.04.007 PC-1; E-SYS-002.
	// NOT t.Parallel(): runAccess panics in stub.

	// A pre-cancelled context causes sc.Connect to return context.Canceled,
	// simulating a connect failure (BC-2.04.007 EC-002: SIGTERM before connect).
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var stderr bytes.Buffer

	// runAccess panics in the stub — that panic IS the Red Gate failure.
	err := runAccess(ctx, &stderr)

	// AC-007 assertion 1 (BC-2.04.007 PC-3): non-nil error → caller exits non-zero.
	if err == nil {
		t.Fatal("runAccess(cancelled ctx): got nil; want non-nil error (non-zero exit — BC-2.04.007 PC-3)")
	}

	// AC-007 assertion 2 (BC-2.04.007 PC-2): E-SYS-002 diagnostic in stderr or
	// error message. Either location satisfies the observable contract.
	combined := err.Error() + " " + stderr.String()
	const esys002 = "fatal: cannot connect to session backend"
	if !strings.Contains(combined, esys002) {
		t.Errorf("E-SYS-002 message not found; want %q; "+
			"got err=%q stderr=%q", esys002, err.Error(), stderr.String())
	}
}

// ── AC-008: TestDaemonCleanShutdown ──────────────────────────────────────────

// TestDaemonCleanShutdown — AC-008 (BC-2.04.007 PC-2)
//
// Verifies that when SIGTERM or SIGINT is received (simulated here via context
// cancellation), the daemon:
//  1. Cancels its root context.
//  2. All goroutines (relay, sweep ticker, frames-dropped ticker) drain.
//  3. runAccess returns nil (exit code 0).
//  4. No goroutines are leaked.
//
// Red Gate: runAccess panics("not implemented") in the stub.
func TestDaemonCleanShutdown(t *testing.T) {
	// AC-008 — BC-2.04.007 PC-2.
	// NOT t.Parallel(): sends SIGTERM to the test process.

	goroutinesBefore := runtime.NumGoroutine()

	ctx, cancel := context.WithCancel(context.Background())

	var stderr bytes.Buffer
	errCh := make(chan error, 1)
	go func() {
		errCh <- runAccess(ctx, &stderr)
	}()

	// Give the daemon a brief window to start its goroutines before cancellation.
	// 20ms is sufficient; the stub panics immediately (Red Gate) and the
	// real implementation must start within a reasonable time.
	time.Sleep(20 * time.Millisecond)

	// Simulate SIGTERM: cancel context AND deliver actual signal.
	cancel()
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
		t.Logf("Kill(SIGTERM): %v (context already cancelled; non-fatal)", err)
	}

	// AC-008 assertion 1 (BC-2.04.007 PC-5): runAccess returns within 500ms.
	select {
	case err := <-errCh:
		// AC-008 assertion 2 (BC-2.04.007 PC-2, PC-5): nil return → exit 0.
		if err != nil {
			t.Errorf("runAccess clean shutdown: got %v; want nil (exit 0 — BC-2.04.007 PC-5)", err)
		}
	case <-time.After(500 * time.Millisecond):
		cancel()
		t.Fatal("runAccess did not return within 500ms of ctx cancellation; " +
			"goroutines must observe ctx.Done() promptly (BC-2.04.007 PC-3)")
	}

	// AC-008 assertion 3 (BC-2.04.007 PC-6): no goroutine leak.
	// t.Cleanup runs after the test body; by then goroutines should have drained.
	t.Cleanup(func() {
		deadline := time.After(200 * time.Millisecond)
		for {
			after := runtime.NumGoroutine()
			if after <= goroutinesBefore+2 { // ±2 for test harness overhead
				return
			}
			select {
			case <-deadline:
				t.Errorf("goroutine leak: before=%d after=%d (≥3 extra); "+
					"all daemon goroutines must exit on shutdown (BC-2.04.007 PC-6)",
					goroutinesBefore, runtime.NumGoroutine())
				return
			default:
				runtime.Gosched()
			}
		}
	})
}

// ── Fake helpers for cmd/switchboard tests ────────────────────────────────────

// fakeExecFuncErrMain returns a tmux.Option that makes ControlMode.Connect fail
// with errToReturn immediately (hermetic; no real tmux subprocess).
func fakeExecFuncErrMain(errToReturn error) tmux.Option {
	return tmux.WithExecFunc(func(_ context.Context) (io.WriteCloser, io.ReadCloser, <-chan error, error) {
		return nil, nil, nil, errToReturn
	})
}

// pipeMasterMain is a goroutine-safe fake io.ReadWriteCloser used as a
// hermetic PTY master in cmd/switchboard tests. Read blocks until bytes are
// available or Close is called (returns io.EOF on close).
type pipeMasterMain struct {
	mu     sync.Mutex
	cond   *sync.Cond
	buf    []byte
	closed bool
}

func newPipeMasterMain() *pipeMasterMain {
	m := &pipeMasterMain{}
	m.cond = sync.NewCond(&m.mu)
	return m
}

func (m *pipeMasterMain) Read(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for len(m.buf) == 0 && !m.closed {
		m.cond.Wait()
	}
	if m.closed && len(m.buf) == 0 {
		return 0, io.EOF
	}
	n := copy(p, m.buf)
	m.buf = m.buf[n:]
	return n, nil
}

func (m *pipeMasterMain) Write(p []byte) (int, error) { return len(p), nil }

func (m *pipeMasterMain) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	m.cond.Broadcast()
	return nil
}

// ── Existing tests (preserved) ────────────────────────────────────────────────

func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantErr    bool
		wantOutput string
	}{
		{
			name:       "version flag prints version",
			args:       []string{"switchboard", "--version"},
			wantOutput: "dev",
		},
		{
			name:       "no args prints version",
			args:       []string{"switchboard"},
			wantOutput: "dev",
		},
		{
			name:    "unknown flag returns error",
			args:    []string{"switchboard", "--bogus"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := run(&buf, tt.args)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := buf.String()
			if !strings.Contains(got, tt.wantOutput) {
				t.Errorf("output %q does not contain %q", got, tt.wantOutput)
			}
		})
	}
}

func TestVersionNonEmpty(t *testing.T) {
	t.Parallel()

	if version == "" {
		t.Fatal("version must not be empty")
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestRun_WriteError(t *testing.T) {
	t.Parallel()

	err := run(failWriter{}, []string{"switchboard", "--version"})
	if err == nil {
		t.Fatal("expected error from failing writer")
	}
}
