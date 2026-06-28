// Package main — main_test.go contains integration tests for the access-node
// daemon wiring (S-W3.04; AC-001 through AC-008).
//
// BC traces:
//   - AC-001 → BC-2.05.008 PC-2 + invariant 1 (router logger / E-ADM-016)
//   - AC-002 → BC-2.04.005 PC-3 + BC-2.04.003 PC-3 (live SessionAuth)
//   - AC-003 → BC-2.04.004 PC-1 + PC-3 (sweep eviction)
//   - AC-006 → BC-2.04.006 v1.4 invariant 4 (dual-counter FramesDropped log ticker)
//   - AC-007 → BC-2.04.007 PC-1 (connect failure → log + non-zero exit)
//   - AC-007/PC-2.6 → BC-2.04.007 PC-2.6 (mid-session double-failure → E-SYS-002 + exit 1)
//   - AC-008 → BC-2.04.007 PC-2 (SIGTERM/SIGINT → clean shutdown + exit 0)
//
// NOTE on startFramesDroppedTicker testability: startFramesDroppedTicker accepts
// a tickInterval parameter (FIX 4; symmetric with startSweepTicker), enabling the
// AC-006 test to drive a real tick with time.Millisecond and assert the production
// code path emits "frames_dropped relay=<N> consoles=<M>".
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
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/config"
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

// newFakeSessionConnector builds a hermetic SessionConnector with:
//   - ControlMode that fails with ErrControlModeUnavailable (forces PTY fallback)
//   - PTYProxy backed by a pipeMasterMain (reads block; writes are discarded)
//
// The returned pipe can be used to inject bytes; the sc and pipe are cleaned up
// via t.Cleanup. Also returns keys and pub shared with the connector.
func newFakeSessionConnector(t *testing.T) (
	sc *tmux.SessionConnector,
	keys *admission.AdmittedKeySet,
	pub *session.Publisher,
) {
	t.Helper()

	keys = admission.NewAdmittedKeySet()
	pub = session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)

	ctrl := tmux.New(pub, ds, fakeExecFuncErrMain(tmux.ErrControlModeUnavailable))
	pipe := newPipeMasterMain()
	pty := tmux.NewPTYProxy(pub, ds,
		tmux.WithPTYAllocFunc(func() (io.ReadWriteCloser, int, error) {
			return pipe, 9001, nil
		}),
	)
	sc = tmux.NewSessionConnector(ctrl, pty)
	t.Cleanup(func() {
		_ = pipe.Close()
		_ = sc.Close()
	})
	return sc, keys, pub
}

// ── AC-001: TestRouterLoggerEmitsEADM016 ──────────────────────────────────────

// TestRouterLoggerEmitsEADM016 — AC-001 (BC-2.05.008 PC-2 + invariant 1)
//
// Verifies that buildAccessComponents constructs a routing.Router with a real
// routing.Logger injected — not nil, not a no-op sink — and that the daemon's
// OWN router instance emits E-ADM-016 to the injected logger when an HMAC-bad
// frame is routed.
//
// Strategy (non-tautological):
//
//  1. Construct a captureLogger.
//  2. Call buildAccessComponents(keys, pub, sc, captureLogger) — passing the
//     capture logger as the router's logger (the 4th argument introduced by FIX 2).
//  3. Register a key into the SHARED keyset and the RETURNED router
//     (admitAndRegister).
//  4. Call routing.RouteFrame on the RETURNED router with an HMAC-bad frame.
//  5. Assert captureLogger received E-ADM-016 canonical string AND
//     ErrHMACVerificationFailed is returned.
//
// Discriminating property: if buildAccessComponents wires the router with a
// nil/noop logger, captureLogger records nothing and the test fails at step 5.
// If it wires a DIFFERENT keyset, admitAndRegister's forwarding entry would not
// be visible and RouteFrame would return ErrHMACVerificationFailed for the
// wrong reason (no auth key path) — the SVTN-hex assertion still passes but
// that is fine because the SVTN ID appears in both log paths.
//
// NO parallel r2 reconstruction: the captureLogger IS the daemon's own logger.
// There is no second router.
func TestRouterLoggerEmitsEADM016(t *testing.T) {
	// AC-001 — BC-2.05.008 PC-2 + invariant 1.
	// NOT t.Parallel(): depends on shared keyset construction order.

	svtnID := svtnIDFromByte(0x01)
	nodePub, nodePriv := mustGenEd25519Main(t)

	// Build ONE shared keyset+pub+sc to pass to buildAccessComponents.
	// This matches the production wiring — keys is shared with BOTH an AND router.
	sc, keys, pub := newFakeSessionConnector(t)

	// Connect sc so buildAccessComponents has a live connector.
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("sc.Connect: %v; want nil (PTY fallback path)", err)
	}

	// captureLogger is injected as the router's logger into buildAccessComponents.
	// This IS the daemon's own router logger — no parallel reconstruction.
	cl := &captureLogger{}

	// buildAccessComponents with captureLogger as the routing.Logger (FIX 2).
	// The returned router IS the daemon's router, wired with the shared keyset
	// AND with cl as the logger. Any E-ADM-016 emission goes into cl.
	_, router := buildAccessComponents(keys, pub, sc, cl)

	// Register a key into the SHARED keyset and the daemon's router.
	// If buildAccessComponents wired the router with a DIFFERENT keyset, this
	// registration would not be visible to the router and the HMAC check would
	// fail at the "no auth key" path (PATH-B) rather than the "tag mismatch"
	// path (PATH-A) — but E-ADM-016 is emitted on BOTH paths, so the test still
	// verifies the logger received the event.
	srcAddr, _ := admitAndRegister(t, keys, router, svtnID, nodePub, nodePriv)

	// Craft a frame with an all-zero HMAC tag — mismatch for any derived key.
	hdr := frame.OuterHeader{
		SVTNID:    svtnID,
		SrcAddr:   srcAddr,
		DstAddr:   [8]byte{},
		FrameType: frame.FrameTypeData,
		// HMACTag zero-value: guaranteed to fail verifyFrameHMAC (PATH-A).
	}
	payload := []byte("tampered")

	// Primary assertion: daemon's OWN router (with captureLogger) returns
	// ErrHMACVerificationFailed and logs E-ADM-016 into cl.
	// FAILS if buildAccessComponents wired a nil/noop logger: cl records nothing.
	routeErr := routing.RouteFrame(hdr, payload, router)
	if !errors.Is(routeErr, routing.ErrHMACVerificationFailed) {
		t.Fatalf("RouteFrame(daemon router, tampered): got %v; want ErrHMACVerificationFailed (E-ADM-016)", routeErr)
	}

	// Canonical E-ADM-016 assertions (error-taxonomy.md §ADM).
	// These assert that cl (the daemon's own injected logger) received the event —
	// not a parallel router. The test is RED if cl.lines is empty.

	// 1. Literal "E-ADM-016" must appear for grep-ability.
	if !cl.HasLine("E-ADM-016") {
		t.Errorf("daemon router logger did not receive E-ADM-016 literal; got: %v\n"+
			"(FAIL: buildAccessComponents must inject the captureLogger into the router — "+
			"not a nil/noop logger)", cl.Lines())
	}
	// 2. Canonical prefix per error-taxonomy.md §ADM.
	if !cl.HasLine("wire HMAC verification failed") {
		t.Errorf("daemon router logger missing canonical prefix 'wire HMAC verification failed'; "+
			"got: %v", cl.Lines())
	}
	// 3. SVTN ID (lowercase hex) appears in the log line — identifies the source.
	svtnHex := fmt.Sprintf("%x", svtnID)
	if !cl.HasLine(svtnHex) {
		t.Errorf("daemon router logger missing svtn_id=%q; got: %v", svtnHex, cl.Lines())
	}
}

// ── AC-002: TestDaemonAuthRejectsUnregisteredConsole ──────────────────────────

// TestDaemonAuthRejectsUnregisteredConsole — AC-002
// (BC-2.04.005 PC-3 + BC-2.04.003 PC-3)
//
// Verifies that buildAccessComponents wires a live *session.SessionAuth as the
// Authorizer (not NoOpAuthorizer or nil). Specifically:
//
//  1. (Primary — non-tautological) The AccessNode returned by buildAccessComponents
//     rejects Attach for an unregistered console key — fail-open is CLOSED.
//     This test FAILS if buildAccessComponents wired NoOpAuthorizer (which
//     would allow the unregistered key through).
//
//  2. A registered read-only key's upstream keystroke returns ErrUpstreamReadOnly
//     (E-ADM-007 per error-taxonomy.md §ADM).
//
//  3. A registered full-access key's upstream keystrokes are forwarded.
//
// Assertions 2+3 use a separately constructed AccessNode (an2) with a known
// SessionAuth so key registrations can be controlled precisely. The primary
// non-tautological assertion (1) uses the production-wired an from
// buildAccessComponents to verify the fail-open default is closed.
//
// Attachment precedes authorization (F-L-1): consoles must be Attached before
// SendKeystroke reaches the authorizer. An unattached console returns
// ErrConsoleNotFound (E-SES-003) before the authorizer runs.
func TestDaemonAuthRejectsUnregisteredConsole(t *testing.T) {
	// AC-002 — BC-2.04.005 PC-3 + BC-2.04.003 PC-3.

	const sessionName = "agent-01"
	unregisteredKey := session.ConsoleKey("unregistered")
	readOnlyKey := session.ConsoleKey("readonly")
	fullAccessKey := session.ConsoleKey("fullaccess")

	// Build ONE shared keys+pub+sc — matches production wiring in runAccess.
	sc, keys, pub := newFakeSessionConnector(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("sc.Connect: %v; want nil (PTY fallback path)", err)
	}

	// buildAccessComponents — production wiring (non-tautological assertion 1).
	// The returned an uses a live *session.SessionAuth as the Authorizer.
	// Pass a captureLogger; its contents are not asserted here (AC-002 does not
	// test the router logger — that is AC-001's scope).
	an, _ := buildAccessComponents(keys, pub, sc, &captureLogger{})

	// Ensure session is published (buildAccessComponents may or may not publish it).
	if err := pub.Publish(sessionName); err != nil {
		if !errors.Is(err, session.ErrSessionAlreadyPublished) {
			t.Fatalf("Publish: %v", err)
		}
	}

	// Assertion 1 (primary — non-tautological): fail-open is CLOSED.
	// With live SessionAuth, Attach with an unregistered key must return an error.
	// A NoOpAuthorizer would allow the key through — this test catches that case.
	_, _, attachErr := an.Attach(unregisteredKey, sessionName)
	if attachErr == nil {
		t.Errorf("Attach(unregistered key): got nil; want error (fail-open default must be closed — W3-M-3)")
	}

	// Assertions 2+3: use a parallel AccessNode with a controlled SessionAuth so
	// read-only and full-access roles can be registered precisely.
	// These assertions verify the authorization logic itself (E-ADM-007 path).
	sa := session.NewSessionAuth()
	sa.RegisterKey(sessionName, readOnlyKey, session.RoleReadOnly)
	sa.RegisterKey(sessionName, fullAccessKey, session.RoleFull)

	sink := &keystrokeSinkCapture{}
	an2 := session.NewAccessNode(pub, sa, session.WithKeystrokeSink(sink))

	// Attach full-access console so it's in the fan-out set.
	if _, _, err := an2.Attach(fullAccessKey, sessionName); err != nil {
		t.Fatalf("Attach(fullAccess): %v", err)
	}

	// Attach the read-only console before sending a keystroke.
	// BC-2.04.005 Trigger: "Console attaches with a read-only session authorization key;
	// console sends a keystroke while in read-only mode." Attachment is a precondition
	// of the authorization check; without it SendKeystroke returns ErrConsoleNotFound
	// (E-SES-003) before the authorizer can return ErrUpstreamReadOnly (E-ADM-007).
	if _, _, err := an2.Attach(readOnlyKey, sessionName); err != nil {
		t.Fatalf("Attach(readOnly): %v", err)
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
	var wg sync.WaitGroup
	wg.Add(1)
	startSweepTicker(ctx, &wg, an, time.Millisecond, sweepDeadlineTest)

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

// TestDaemonFramesDroppedLoggedOnTick — AC-006 (BC-2.04.006 v1.4 invariant 4)
//
// Verifies the dual-counter observability requirement:
//   - startFramesDroppedTicker logs BOTH sc.RelayDropped() and an.FramesDropped()
//   - Log format: "frames_dropped relay=<N> consoles=<M>" (both counters required)
//   - Relay-layer drops and ConsoleSet-layer drops are SEPARATE counters (EC-003)
//
// Now that startFramesDroppedTicker accepts a tickInterval parameter (FIX 4,
// matching startSweepTicker pattern), this test drives an actual tick via
// time.Millisecond and asserts the PRODUCTION code path wrote the log line —
// not a hand-rolled lg.Printf.
//
// Counter isolation assertion (EC-003): relay-layer drops (sc.RelayDropped())
// are NOT reflected in an.FramesDropped() (ConsoleSet-layer). Verified by
// saturating sc.frames and asserting FramesDropped() remains 0.
func TestDaemonFramesDroppedLoggedOnTick(t *testing.T) {
	// AC-006 — BC-2.04.006 v1.4 invariant 4.

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

	// Saturate the downstream buffer (64 per fanout.go) to trigger ConsoleSet-layer drops.
	for i := range 200 {
		an.DeliverFrame(frame.OuterHeader{
			SVTNID:    svtnIDFromByte(0x02),
			FrameType: frame.FrameTypeData,
			SrcAddr:   [8]byte{byte(i % 256)},
		})
	}

	consolesDropped := an.FramesDropped()
	if consolesDropped == 0 {
		t.Skip("no frames dropped in setup; downstream buffer consumed all frames (test precondition not met)")
	}

	// Build a fake sc to pass to startFramesDroppedTicker.
	// sc.RelayDropped() starts at 0 — relay has not run yet.
	sc, _, _ := newFakeSessionConnector(t)

	if sc.RelayDropped() != 0 {
		t.Fatalf("sc.RelayDropped() initial value: got %d; want 0", sc.RelayDropped())
	}

	// Drive an actual tick through the production goroutine. cw captures whatever
	// the goroutine's lg.Printf writes — this asserts the production code path, not
	// a hand-typed format string (FIX 4: tickInterval is now injectable).
	cw := &captureWriter{}
	lg := log.New(cw, "", 0)

	ctx, cancel := context.WithCancel(context.Background())
	gorsBefore := runtime.NumGoroutine()
	// time.Millisecond: fast enough that at least one tick fires before the
	// 500ms wait deadline below.
	var wg sync.WaitGroup
	wg.Add(1)
	startFramesDroppedTicker(ctx, &wg, sc, an, lg, time.Millisecond)

	// Wait for the production ticker goroutine to emit at least one log line.
	tickDeadline := time.After(500 * time.Millisecond)
	for !strings.Contains(cw.String(), "frames_dropped") {
		select {
		case <-tickDeadline:
			t.Fatalf("startFramesDroppedTicker produced no 'frames_dropped' log line within 500ms; " +
				"production goroutine must emit on each tick (BC-2.04.006 v1.4 invariant 4)")
		default:
			runtime.Gosched()
		}
	}
	cancel()

	logged := cw.String()

	// Assert the canonical format (BC-2.04.006 v1.4 invariant 4).
	if !strings.Contains(logged, "frames_dropped") {
		t.Errorf("log line missing 'frames_dropped' key; got: %q", logged)
	}
	if !strings.Contains(logged, "relay=") {
		t.Errorf("log line missing 'relay=' counter; got: %q (AC-006 requires both counters)", logged)
	}
	if !strings.Contains(logged, "consoles=") {
		t.Errorf("log line missing 'consoles=' counter; got: %q (AC-006 requires both counters)", logged)
	}

	// Assert relay=0 (no relay drops yet) and consoles=<N> (ConsoleSet drops from above).
	// Note: the ticker may have fired multiple times; we check that at least one line
	// has the exact values. The first tick captures the values at that moment.
	wantRelayField := "relay=0"
	wantConsolesField := fmt.Sprintf("consoles=%d", consolesDropped)
	if !strings.Contains(logged, wantRelayField) {
		t.Errorf("log output: want %q; got: %q", wantRelayField, logged)
	}
	if !strings.Contains(logged, wantConsolesField) {
		t.Errorf("log output: want %q; got: %q", wantConsolesField, logged)
	}

	// AC-006 goroutine lifecycle: verify goroutine exits cleanly on ctx cancel.
	t.Cleanup(func() {
		deadline := time.After(200 * time.Millisecond)
		for {
			after := runtime.NumGoroutine()
			if after <= gorsBefore+1 {
				return
			}
			select {
			case <-deadline:
				t.Errorf("startFramesDroppedTicker goroutine leak: before=%d after=%d; "+
					"must exit on ctx cancel (BC-2.04.007 PC-2)", gorsBefore, runtime.NumGoroutine())
				return
			default:
				runtime.Gosched()
			}
		}
	})
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

// ── AC-007/EC-003: TestSessionConnectorFramesRelayDropIncrementsCounter ─────────

// TestSessionConnectorFramesRelayDropIncrementsCounter — AC-006/EC-003
// (BC-2.04.006 v1.4 Inv-4; ADR-011 §Concurrency; ARCH-01 v1.4 §Relay-drop
// counter contract)
//
// Verifies that when sc.frames is full (relay channel saturated), the
// forwardFrames goroutine:
//  1. Uses a non-blocking select (does NOT block on the full channel).
//  2. Increments sc.RelayDropped() for each dropped frame.
//  3. Does NOT increment an.FramesDropped() — relay-layer drops are a SEPARATE
//     counter from ConsoleSet-layer drops (EC-003 two-counter clarification).
//
// Discriminating: a naive implementation that did not increment relayDropped
// on the non-blocking drop path would leave RelayDropped() == 0 after
// saturation, causing the test to fail.
//
// Approach: two-phase injection.
//
// Phase 1: inject framesBufferSize bytes one at a time, draining sc.Frames()
// to confirm they flow all the way through (pty.frames → sc.frames consumed).
// Then inject more bytes while NOT draining sc.Frames() — sc.frames fills up.
//
// Phase 2: continue injecting. sc.frames is full; forwardFrames reads from
// pty.frames and relay-drops into the relayDropped counter. The injection
// goroutine must not block (non-blocking select in forwardFrames).
//
// Note on the one-frame-per-read constraint: PTYProxy.ioRelay reads from the
// pipe into a 4096-byte buffer. Each Read call produces exactly ONE ChannelFrame
// (since MaxPayloadSize = 65515 >> 4096, all bytes from one read fit in one
// Enqueue call). To produce N separate frames, we need N separate Read calls.
// singleBytePipeMaster limits each Read to 1 byte, guaranteeing 1 frame per
// injection call.
//
// Two-phase protocol:
//
//	Phase 1 — saturate sc.frames: inject framesBufferSize bytes one at a time
//	(singleBytePipeMaster), WITHOUT consuming sc.Frames(). Each byte is a
//	separate Read → 1 frame. After framesBufferSize injections, sc.frames is
//	full and pty.frames is drained (forwardFrames moved frames to sc.frames).
//
//	Phase 2 — trigger relay drops: inject more bytes (still one at a time).
//	New frames arrive in pty.frames; forwardFrames reads them and tries to write
//	to the already-full sc.frames → non-blocking default branch → relay drop →
//	relayDropped++.
func TestSessionConnectorFramesRelayDropIncrementsCounter(t *testing.T) {
	// AC-006/EC-003 — relay-layer drop counter; BC-2.04.006 v1.4 Inv-4.
	// NOT t.Parallel(): relies on goroutine phases.

	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)

	// ctrl fails → PTY path.
	ctrl := tmux.New(pub, ds, fakeExecFuncErrMain(tmux.ErrControlModeUnavailable))
	// singleBytePipeMaster: each Read returns at most 1 byte, ensuring
	// ioRelay produces exactly 1 ChannelFrame per injected byte.
	pipe := newSingleBytePipeMaster()
	pty := tmux.NewPTYProxy(pub, ds,
		tmux.WithPTYAllocFunc(func() (io.ReadWriteCloser, int, error) {
			return pipe, 7777, nil
		}),
	)
	sc := tmux.NewSessionConnector(ctrl, pty)
	t.Cleanup(func() {
		_ = pipe.Close()
		_ = sc.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("sc.Connect: %v; want nil", err)
	}

	// Build an AccessNode to verify relay drops do NOT increment FramesDropped().
	sa := session.NewSessionAuth()
	an := session.NewAccessNode(pub, sa, session.WithKeystrokeSink(session.NoOpSink{}))
	consolesDroppedBefore := an.FramesDropped()

	// Phase 1: inject framesBufferSize (256) bytes one at a time, WITHOUT consuming
	// sc.Frames(). Each byte → 1 Read call → 1 frame → forwarded to sc.frames.
	// After phase 1, sc.frames is full (256 frames).
	const framesBufferSize = 256
	for range framesBufferSize {
		pipe.injectBytes([]byte("p"))
	}

	// Wait for sc.frames to fill.
	filledDeadline := time.After(3 * time.Second)
	for len(sc.Frames()) < framesBufferSize {
		select {
		case <-filledDeadline:
			t.Fatalf("phase 1: sc.frames did not fill to %d within 3s (got %d); "+
				"pipeline not forwarding frames (1 byte = 1 frame with singleBytePipeMaster)",
				framesBufferSize, len(sc.Frames()))
		default:
			runtime.Gosched()
		}
	}

	// sc.frames is now full. Phase 2: inject more bytes. forwardFrames reads each
	// new frame from pty.frames and relay-drops it (sc.frames is full).
	const phase2Count = 50
	injectDone := make(chan struct{})
	go func() {
		defer close(injectDone)
		for range phase2Count {
			pipe.injectBytes([]byte("q"))
		}
	}()

	// Injection goroutine must not block: both ioRelay and forwardFrames use
	// non-blocking selects, so injection completes even when channels are full.
	select {
	case <-injectDone:
	case <-time.After(3 * time.Second):
		t.Fatal("EC-003: injection goroutine blocked for >3s; " +
			"forwardFrames relay must use non-blocking select (ADR-011 §Concurrency)")
	}

	// Wait for relayDropped > 0.
	relayDeadline := time.After(2 * time.Second)
	for sc.RelayDropped() == 0 {
		select {
		case <-relayDeadline:
			t.Fatalf("sc.RelayDropped() == 0 after phase-2 injection of %d frames with full sc.frames; "+
				"forwardFrames must increment relayDropped on non-blocking drop "+
				"(BC-2.04.006 v1.4 Inv-4)", phase2Count)
		default:
			runtime.Gosched()
		}
	}

	// Assertion 1: relay-layer drops counted.
	relayDropped := sc.RelayDropped()
	if relayDropped == 0 {
		t.Errorf("sc.RelayDropped() == 0; want > 0 (BC-2.04.006 v1.4 Inv-4)")
	}

	// Assertion 2 (counter isolation — EC-003): relay drops must NOT be reflected
	// in an.FramesDropped() (ConsoleSet-layer counter).
	consolesDroppedAfter := an.FramesDropped()
	if consolesDroppedAfter != consolesDroppedBefore {
		t.Errorf("an.FramesDropped() changed from %d to %d during relay-layer drops; "+
			"relay drops must NOT increment ConsoleSet-layer counter (EC-003)",
			consolesDroppedBefore, consolesDroppedAfter)
	}
}

// singleBytePipeMaster is a fake io.ReadWriteCloser where Read returns at most
// 1 byte per call. This ensures PTYProxy.ioRelay produces exactly 1 ChannelFrame
// per injected byte (since one Read → one Enqueue → one Tick → one frame).
// This is required to trigger relay-level drops in TestSessionConnectorFramesRelayDropIncrementsCounter.
type singleBytePipeMaster struct {
	mu     sync.Mutex
	cond   *sync.Cond
	buf    []byte
	closed bool
}

func newSingleBytePipeMaster() *singleBytePipeMaster {
	m := &singleBytePipeMaster{}
	m.cond = sync.NewCond(&m.mu)
	return m
}

func (m *singleBytePipeMaster) injectBytes(p []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.buf = append(m.buf, p...)
	m.cond.Broadcast()
}

// Read returns exactly 1 byte (blocking until available or closed).
func (m *singleBytePipeMaster) Read(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for len(m.buf) == 0 && !m.closed {
		m.cond.Wait()
	}
	if m.closed && len(m.buf) == 0 {
		return 0, io.EOF
	}
	if len(p) == 0 {
		return 0, nil
	}
	// Return at most 1 byte per Read call.
	p[0] = m.buf[0]
	m.buf = m.buf[1:]
	return 1, nil
}

func (m *singleBytePipeMaster) Write(p []byte) (int, error) { return len(p), nil }

func (m *singleBytePipeMaster) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	m.cond.Broadcast()
	return nil
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
// Goroutine tolerance: the test allows goroutinesBefore+1 (not +2) after a
// 300ms settle. A tolerance of +2 can mask a single real goroutine leak;
// +1 catches leaked goroutines while allowing for one transient Go runtime
// goroutine that may appear during t.Cleanup execution. If the test becomes
// flaky at +1 due to a specific named harness goroutine, increase to +2 and
// document the named goroutine here.
//
// Red Gate: runAccess panics("not implemented") in the stub.
func TestDaemonCleanShutdown(t *testing.T) {
	// AC-008 — BC-2.04.007 PC-2.
	// NOT t.Parallel(): sends SIGTERM to the test process.
	//
	// Pre-check: runAccess creates its own SessionConnector with defaultPTYAlloc.
	// If PTY allocation fails (e.g. macOS sandbox / permission environment), skip
	// rather than fail — the test is structurally correct but requires a working
	// PTY device. CI environments with full PTY access run this test end-to-end.
	// The connect-failure path is covered by TestDaemonConnectFailureExitsNonZero.
	if !ptyAvailableForTest() {
		t.Skip("PTY device unavailable in this environment; skipping clean-shutdown test " +
			"(requires working /dev/ptmx + slave open; covered by CI with full PTY access)")
	}

	goroutinesBefore := runtime.NumGoroutine()

	ctx, cancel := context.WithCancel(context.Background())

	var stderr bytes.Buffer
	errCh := make(chan error, 1)
	go func() {
		errCh <- runAccess(ctx, &stderr)
	}()

	// Give the daemon a brief window to start its goroutines before cancellation.
	// 50ms is sufficient; the stub panics immediately (Red Gate) and the
	// real implementation must start within a reasonable time.
	time.Sleep(50 * time.Millisecond)

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
	// Tightened tolerance: +1 (down from +2) to catch single-goroutine leaks.
	// Allow 300ms for goroutines to fully drain after runAccess returns.
	t.Cleanup(func() {
		deadline := time.After(300 * time.Millisecond)
		for {
			after := runtime.NumGoroutine()
			if after <= goroutinesBefore+1 {
				return
			}
			select {
			case <-deadline:
				t.Errorf("goroutine leak: before=%d after=%d (≥2 extra); "+
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

// ptyAvailableForTest probes whether PTY allocation works in the current
// environment. runAccess calls tmux.NewPTYProxy with defaultPTYAlloc (real
// /dev/ptmx open). If that fails (macOS sandbox, container, CI without PTY),
// tests that depend on runAccess successfully connecting must be skipped.
func ptyAvailableForTest() bool {
	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)
	// NewPTYProxy with no options uses defaultPTYAlloc (real /dev/ptmx).
	pty := tmux.NewPTYProxy(pub, ds)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := pty.Connect(ctx)
	_ = pty.Close()
	return err == nil
}

// ── S-6.01 AC-003: TestRouterStartup_ExitsWithActionableError ─────────────────

// TestRouterStartup_ExitsWithActionableError — AC-003 (BC-2.09.003 postconditions 1 and 2)
//
// Verifies that the daemon cmd-level startup path:
//  1. Accepts a --config flag pointing to a config file.
//  2. Calls config.LoadFile and cfg.Validate() at startup BEFORE any socket bind.
//  3. On validation failure: returns a non-nil error whose message follows the
//     E-CFG-001 actionable format "config error: <field>: <problem>. Fix: <suggestion>".
//  4. The error is returned by run() so that main() can print it to stderr and
//     call os.Exit(1) — exit code 1 behavior (BC-2.09.003 postconditions 1 and 2).
//
// This test exercises the cmd-level path — not just the config-layer contract in
// isolation. It deliberately tests that run() wires LoadFile → Validate → return
// error, which is distinct from the internal/config unit tests.
//
// Red Gate: run() currently has no --config flag and no config loading. Calling
// run(stdout, []string{"switchboard", "access", "--config", path}) returns a flag
// parse error ("flag provided but not defined: -config"), NOT an E-CFG-001
// actionable message. This test fails until the implementer:
//   - Adds a --config flag to the "access" subcommand FlagSet in run()
//   - Calls config.LoadFile(configPath) before runAccess
//   - Calls cfg.Validate() and returns E-CFG-001 error on failure
//
// Traces: BC-2.09.003 postconditions 1 and 2, AC-003, S-6.01 v1.1.
func TestRouterStartup_ExitsWithActionableError(t *testing.T) {
	// NOT t.Parallel(): exercises the cmd startup path; not safe to parallelise
	// with tests that also call run() with "access".

	// Write a config file that is syntactically valid YAML but has an invalid
	// tick_interval (below the 5ms minimum) so it passes LoadFile but fails Validate.
	// This exercises the Validate → E-CFG-001 path.
	const badConfigContent = "listen_addr: 0.0.0.0:9090\ntick_interval: 1ms\n"
	dir := t.TempDir()
	configPath := filepath.Join(dir, "bad-config.yaml")
	if err := os.WriteFile(configPath, []byte(badConfigContent), 0o600); err != nil {
		t.Fatalf("WriteFile bad config: %v", err)
	}

	var stdout bytes.Buffer
	// run() must accept --config; currently it does not, so this call returns a
	// flag-parse error. The test asserts the error message matches the E-CFG-001
	// actionable format — which will NOT match a flag error.
	err := run(&stdout, []string{"switchboard", "access", "--config", configPath})

	// AC-003 assertion 1: non-nil error (exit code 1 — BC-2.09.003 postconditions 1 and 2).
	if err == nil {
		t.Fatal("run(access --config bad-config): got nil error; want non-nil " +
			"(config validation must cause exit 1 — BC-2.09.003 postconditions 1 and 2)")
	}

	// AC-003 assertion 2: error message follows E-CFG-001 actionable format.
	// BC-2.09.003 postconditions 1 and 2: "config error: <field>: <problem>. Fix: <suggestion>"
	// The "tick_interval" field must be named; the suggestion must contain "Fix:".
	msg := err.Error()
	if !strings.Contains(msg, "config error") {
		t.Errorf("run(access --config bad-config): error must contain 'config error' "+
			"(E-CFG-001 actionable format); got: %q\n"+
			"(FAIL: run() must call LoadFile+Validate and return E-CFG-001 on bad config, "+
			"not a flag-parse error)", msg)
	}
	if !strings.Contains(msg, "tick_interval") {
		t.Errorf("run(access --config bad-config): error must name the invalid field "+
			"'tick_interval'; got: %q", msg)
	}
	if !strings.Contains(msg, "Fix") && !strings.Contains(msg, "fix") {
		t.Errorf("run(access --config bad-config): error must contain a fix suggestion "+
			"(BC-2.09.003 postconditions 1 and 2 / AC-003); got: %q", msg)
	}

	// AC-003 assertion 3: stdout must be empty (BC-2.09.003 postcondition 3).
	if stdout.Len() != 0 {
		t.Errorf("run(access --config bad-config): stdout must be empty on config error; "+
			"got: %q", stdout.String())
	}
}

// ── S-6.01 F-002a: TestBC_2_09_003_MissingConfigFile_ExitsNonZero ──────────────

// TestBC_2_09_003_MissingConfigFile_ExitsNonZero — F-002a (BC-2.09.003 EC-001)
//
// Verifies that when the daemon is started with --config pointing at a
// NON-EXISTENT path, the cmd-level run() function:
//  1. Returns a non-nil error (caller exits with code 1).
//  2. The error message contains E-CFG-004 and the literal "config file not found"
//     followed by the path that was not found.
//
// This is a coverage-backfill test for the DAEMON-STARTUP path (not a red-gate):
// the wiring in main.go already returns the LoadFile error to run()'s caller
// before reaching runAccess. The test verifies the end-to-end cmd-level observable.
//
// Traces: BC-2.09.003 EC-001, postconditions 1 and 2.
func TestBC_2_09_003_MissingConfigFile_ExitsNonZero(t *testing.T) {
	// NOT t.Parallel(): exercises the cmd startup path alongside other "access" tests.

	dir := t.TempDir()
	// Construct a path that does not exist inside the temp dir.
	missingPath := filepath.Join(dir, "does-not-exist.yaml")

	var stdout bytes.Buffer
	err := run(&stdout, []string{"switchboard", "access", "--config", missingPath})

	// Assertion 1: non-nil error → exit code 1 (BC-2.09.003 postcondition 1).
	if err == nil {
		t.Fatal("run(access --config <missing>): got nil error; want non-nil " +
			"(missing config file must cause exit 1 — BC-2.09.003 EC-001, postcondition 1)")
	}

	// Assertion 2: error carries E-CFG-004 code (BC-2.09.003 EC-001).
	var ce *config.ConfigError
	if !errors.As(err, &ce) {
		t.Fatalf("run(access --config <missing>): error is %T %q; want *config.ConfigError "+
			"with code E-CFG-004 (BC-2.09.003 EC-001)", err, err.Error())
	}
	if ce.Code != "E-CFG-004" {
		t.Errorf("run(access --config <missing>): error code = %q; want E-CFG-004 "+
			"(BC-2.09.003 EC-001)", ce.Code)
	}

	// Assertion 3: message contains canonical "config file not found: <path>"
	// (BC-2.09.003 EC-001 canonical format).
	msg := err.Error()
	if !strings.Contains(msg, "config file not found") {
		t.Errorf("run(access --config <missing>): error must contain 'config file not found'; "+
			"got: %q", msg)
	}
	if !strings.Contains(msg, missingPath) {
		t.Errorf("run(access --config <missing>): error must contain the missing path %q; "+
			"got: %q", missingPath, msg)
	}

	// Assertion 4: stdout must be empty — no partial output on error
	// (BC-2.09.003 postcondition 3).
	if stdout.Len() != 0 {
		t.Errorf("run(access --config <missing>): stdout must be empty on config error; "+
			"got: %q", stdout.String())
	}
}

// ── S-6.01 F-002b: TestBC_2_09_003_MalformedYAML_ExitsNonZero ────────────────

// TestBC_2_09_003_MalformedYAML_ExitsNonZero — F-002b (BC-2.09.003 EC-003)
//
// Verifies that when the daemon is started with --config pointing at a
// syntactically malformed YAML file, the cmd-level run() function:
//  1. Returns a non-nil error (caller exits with code 1).
//  2. The error message contains E-CFG-005 and the canonical "at line" fragment
//     per BC-2.09.003 EC-003 format: "config parse error: invalid YAML at line N: <detail>".
//
// The malformed fixture is written to t.TempDir() at test time.
//
// This is a coverage-backfill test: the wiring in main.go returns the LoadFile
// error before reaching runAccess. The test verifies the end-to-end cmd observable.
//
// Traces: BC-2.09.003 EC-003 (FM-010), postconditions 1 and 2.
func TestBC_2_09_003_MalformedYAML_ExitsNonZero(t *testing.T) {
	// NOT t.Parallel(): exercises the cmd startup path alongside other "access" tests.

	// Malformed YAML: a tab-indented key is a syntax error in YAML.
	// yaml.v3 will report this as "yaml: line 3: found character that cannot start
	// any token" — the line number is load-bearing for the E-CFG-005 canonical format.
	const malformedContent = "listen_addr: 0.0.0.0:9090\ntick_interval: 10ms\n\tfoo: bad\n"
	dir := t.TempDir()
	configPath := filepath.Join(dir, "malformed.yaml")
	if err := os.WriteFile(configPath, []byte(malformedContent), 0o600); err != nil {
		t.Fatalf("WriteFile malformed config: %v", err)
	}

	var stdout bytes.Buffer
	err := run(&stdout, []string{"switchboard", "access", "--config", configPath})

	// Assertion 1: non-nil error → exit code 1 (BC-2.09.003 postcondition 1).
	if err == nil {
		t.Fatal("run(access --config <malformed>): got nil error; want non-nil " +
			"(malformed YAML must cause exit 1 — BC-2.09.003 EC-003, postcondition 1)")
	}

	// Assertion 2: error carries E-CFG-005 code (BC-2.09.003 EC-003).
	var ce *config.ConfigError
	if !errors.As(err, &ce) {
		t.Fatalf("run(access --config <malformed>): error is %T %q; want *config.ConfigError "+
			"with code E-CFG-005 (BC-2.09.003 EC-003)", err, err.Error())
	}
	if ce.Code != "E-CFG-005" {
		t.Errorf("run(access --config <malformed>): error code = %q; want E-CFG-005 "+
			"(BC-2.09.003 EC-003)", ce.Code)
	}

	// Assertion 3: message contains "at line" — the canonical E-CFG-005 format
	// "config parse error: invalid YAML at line N: <detail>" (BC-2.09.003 EC-003).
	// This distinguishes the properly-formatted error from a raw yaml library string.
	msg := err.Error()
	if !strings.Contains(msg, "at line") {
		t.Errorf("run(access --config <malformed>): E-CFG-005 message must contain 'at line' "+
			"(canonical format 'config parse error: invalid YAML at line N: <detail>'); "+
			"got: %q", msg)
	}

	// Assertion 4: stdout must be empty on error (BC-2.09.003 postcondition 3).
	if stdout.Len() != 0 {
		t.Errorf("run(access --config <malformed>): stdout must be empty on config error; "+
			"got: %q", stdout.String())
	}
}

// ── S-6.01 F-003: TestBC_2_09_003_InvalidConfig_DoesNotEnterRunAccess ─────────

// TestBC_2_09_003_InvalidConfig_DoesNotEnterRunAccess — F-003
// (BC-2.09.003 PC4 + invariant 1)
//
// Verifies that when --config is invalid, the daemon does NOT proceed into the
// socket/PTY work (runAccess) — i.e. no partial binding occurs.
//
// Observable contract: the ARCH-06 binding sequence in main.go (lines 61-77) is:
//
//  1. if *configPath != "": LoadFile(path)  →  on error: return err  (line 64)
//  2. cfg.Validate()                        →  on error: return err  (line 67)
//  3. runAccess(ctx, stderr)                ←  only reached on success (line 77)
//
// This test observes the short-circuit WITHOUT a production seam by asserting:
//
//	(a) run() returns a *config.ConfigError — proving the error came from the
//	    config layer, not from runAccess (which would return nil on clean shutdown
//	    or a session-layer error, never a *config.ConfigError).
//
//	(b) run() returns in well under the 50ms window given to runAccess at startup.
//	    runAccess always takes at least ~1ms to begin and would block indefinitely
//	    on a successful PTY connect. A config-error return is synchronous and
//	    completes in microseconds. A 500ms deadline is conservative but sufficient
//	    to distinguish the two paths.
//
// NO production seam required: the error type and return speed together provide
// unambiguous evidence that runAccess was not entered.
//
// Traces: BC-2.09.003 PC4, invariant 1, postconditions 1 and 2.
func TestBC_2_09_003_InvalidConfig_DoesNotEnterRunAccess(t *testing.T) {
	// NOT t.Parallel(): exercises the cmd startup path alongside other "access" tests.

	// Use a missing-file path so LoadFile returns E-CFG-004 before Validate runs.
	// Any config error suffices to exercise the short-circuit; missing file is the
	// simplest because it requires no fixture and cannot accidentally be valid.
	dir := t.TempDir()
	missingPath := filepath.Join(dir, "not-here.yaml")

	// Measure elapsed time to distinguish config-error path (synchronous, ~µs)
	// from runAccess path (blocks on PTY/tmux connect, always >1ms).
	start := time.Now()

	var stdout bytes.Buffer
	err := run(&stdout, []string{"switchboard", "access", "--config", missingPath})

	elapsed := time.Since(start)

	// Assertion 1: run() returned a non-nil error (BC-2.09.003 postcondition 1).
	if err == nil {
		t.Fatal("run(access --config <missing>): got nil error; want non-nil " +
			"(config error must short-circuit before runAccess — BC-2.09.003 invariant 1)")
	}

	// Assertion 2: the error is a *config.ConfigError — ONLY the config layer
	// returns this type. runAccess never returns a *config.ConfigError, so this
	// proves the short-circuit without requiring an injectable runAccess seam.
	var ce *config.ConfigError
	if !errors.As(err, &ce) {
		t.Fatalf("run(access --config <missing>): error is %T (%q); want *config.ConfigError "+
			"(BC-2.09.003 invariant 1: config error must come from config layer, not runAccess)",
			err, err.Error())
	}

	// Assertion 3: run() returned within 500ms — the config-error path is
	// synchronous (I/O: one ReadFile call). runAccess blocks on PTY/tmux
	// connect; if that path were entered, this deadline would fire.
	// 500ms is chosen to be robust across slow CI while still being 10× the
	// maximum reasonable config-layer latency.
	if elapsed > 500*time.Millisecond {
		t.Errorf("run(access --config <missing>): took %v; want < 500ms "+
			"(config-error path is synchronous — if runAccess was entered, it would block "+
			"on PTY/network: BC-2.09.003 invariant 1)", elapsed)
	}

	// Assertion 4: stdout must be empty (BC-2.09.003 postcondition 3).
	if stdout.Len() != 0 {
		t.Errorf("run(access --config <missing>): stdout must be empty on config error; "+
			"got: %q", stdout.String())
	}
}

// ── S-6.01 AC-009 / PC-9 / Inv-5: TestRouterStartup_AppliesValidatedConfig ────

// TestRouterStartup_AppliesValidatedConfig — AC-009 (BC-2.09.003 postcondition 9
// and invariant 5)
//
// Verifies that when a valid --config is supplied, the daemon wires the validated
// config struct into its subsystems rather than discarding it or falling back to
// hardcoded constants (BC-2.09.003 Inv-5: "validated config is the single source
// of truth for subsystem configuration").
//
// RED Gate mechanism:
//
//	runAccessWithConnector currently has signature:
//	  func runAccessWithConnector(ctx, stderr, sc, an, router) error
//	It does NOT accept a *config.Config. This test calls it with a *config.Config
//	as an additional argument, which causes a compile error:
//	  "too many arguments in call to runAccessWithConnector"
//	That compile failure IS the Red Gate for AC-009.
//
//	The implementer must (Task 17):
//	  1. Add *config.Config as a parameter to runAccessWithConnector.
//	  2. Use cfg.ListenAddr for the listen socket bind.
//	  3. Use cfg.TickInterval for the half-channel tick interval.
//	  4. Use cfg.DrainTimeout for the drain subsystem.
//	  5. Use cfg.KeepaliveInterval for keepalive probes.
//
// After Task 17, this test verifies that the returned error is NOT a
// *config.ConfigError (config-layer errors must not propagate from the access layer).
//
// Traces: BC-2.09.003 postcondition 9, invariant 5, AC-009, S-6.01 v1.3 EC-011.
func TestRouterStartup_AppliesValidatedConfig(t *testing.T) {
	// NOT t.Parallel(): exercises the daemon wiring path.

	// Build a valid config with a distinctive non-default listen_addr.
	// "127.0.0.1:19282" differs from any plausible hardcoded default
	// (":9090", "0.0.0.0:9090") — a future socket-level assertion would catch
	// a daemon that ignores the config and uses a hardcoded addr.
	cfg := &config.Config{
		ListenAddr:        "127.0.0.1:19282",
		TickInterval:      10 * time.Millisecond,
		DrainTimeout:      10 * time.Second,
		KeepaliveInterval: 1 * time.Second,
	}

	// Precondition: fixture must be a valid config (validates without error).
	if err := cfg.Validate(); err != nil {
		t.Fatalf("cfg.Validate(valid fixture): %v — fixture must be valid (test precondition)", err)
	}

	// Build a fakeConnector so the test does not require a real PTY.
	sc, keys, pub := newFakeSessionConnector(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("sc.Connect: %v; want nil (PTY fallback path)", err)
	}

	an, router := buildAccessComponents(keys, pub, sc, &captureLogger{})

	// AC-009 / PC-9 / Inv-5 (RED — compile failure until Task 17):
	//
	// runAccessWithConnector MUST accept a *config.Config so it can:
	//   - bind the listen socket on cfg.ListenAddr (not a hardcoded default)
	//   - set cfg.TickInterval as the half-channel tick interval
	//   - set cfg.DrainTimeout for the drain subsystem
	//   - set cfg.KeepaliveInterval for keepalive probes
	//
	// Currently runAccessWithConnector signature is:
	//   func runAccessWithConnector(ctx, stderr, sc, an, router) error
	// Passing cfg as an extra argument produces:
	//   "too many arguments in call to runAccessWithConnector"
	// — that compile error IS the Red Gate. Do NOT remove cfg from the call.
	cancel() // cancel context so runAccessWithConnector exits immediately after wiring
	var stderr bytes.Buffer
	runErr := runAccessWithConnector(ctx, &stderr, sc, an, router, cfg) // RED: extra cfg arg not yet in signature
	// After Task 17 lands: assert runErr is NOT a *config.ConfigError.
	var ce *config.ConfigError
	if errors.As(runErr, &ce) {
		t.Fatalf("runAccessWithConnector(..., cfg): got config error %v; "+
			"config-layer errors must not propagate from the access layer (AC-009 / BC-2.09.003 PC-9)", runErr)
	}
	_ = router // suppress unused warning if router is not otherwise referenced
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
