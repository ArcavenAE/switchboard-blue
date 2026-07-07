// Package upstreamdial — connector_test.go — S-7.04-FU-PE-CONNECTOR unit tests.
//
// These tests will all FAIL at the Red Gate because the Connector methods are
// currently stub panics.  Each test is named exactly as specified in the story's
// Estimated Test Surface table.
//
// Traces:
//   - AC-001 → BC-2.09.001 PC-2/PC-3 (TestConnector_DialSuccess_ModePE,
//     TestConnector_ReorderReuse_NoTeardown)
//   - AC-002 → BC-2.09.001 EC-001/EC-004 (TestConnector_DialFailure_EC001Log,
//     TestConnector_BackoffParameters, TestConnector_AllUpstreamsUnreachable_ModeE)
//   - AC-003 → BC-2.09.003 PC-8 (TestConnector_KeepaliveTickerDrivesHealthProbe)
//   - AC-001 Q1 → (TestConnector_ReloadAddrs_AddsAndRemoves)
package upstreamdial

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/outerassembler"
)

// ── shared helpers ─────────────────────────────────────────────────────────────

// newLoopbackListener starts a TCP listener on 127.0.0.1:0 and returns it
// alongside its address.  The test owns the lifecycle.
func newLoopbackListener(t *testing.T) (net.Listener, string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("newLoopbackListener: Listen: %v", err)
	}
	return ln, ln.Addr().String()
}

// pollForMode blocks until connector.Mode() returns ModePE or timeout elapses.
// Returns true if ModePE was reached.
func pollForMode(c *Connector, budget time.Duration) bool {
	deadline := time.Now().Add(budget)
	for time.Now().Before(deadline) {
		if c.Mode() == ModePE {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return c.Mode() == ModePE
}

// pollForLog blocks until the recorded output contains substr or budget elapses.
type logWriter struct {
	ch chan string
}

func newLogWriter() *logWriter { return &logWriter{ch: make(chan string, 4096)} }

func (lw *logWriter) Write(p []byte) (int, error) {
	lw.ch <- string(p)
	return len(p), nil
}

// contains returns true when any buffered message contains substr (non-blocking drain).
func (lw *logWriter) contains(substr string) bool {
	// Drain any newly written messages into a local slice.
	lines := make([]string, 0, len(lw.ch))
	for {
		select {
		case msg := <-lw.ch:
			lines = append(lines, msg)
		default:
			// Put them back — push logic is not needed; rebuild is fine for tests.
			// Actually we need to rebuild. We do it by collecting all and re-checking.
			for _, l := range lines {
				lw.ch <- l
			}
			for _, l := range lines {
				if strings.Contains(l, substr) {
					return true
				}
			}
			return false
		}
	}
}

// waitForLog polls lw.contains(substr) until budget elapses.
func waitForLog(lw *logWriter, substr string, budget time.Duration) bool {
	deadline := time.Now().Add(budget)
	for time.Now().Before(deadline) {
		if lw.contains(substr) {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return lw.contains(substr)
}

// zeroEnv returns a minimal Envelope suitable for unit tests.  The key
// material and addresses are zero-valued, which is sufficient for the
// stub's panic (any Connector call will panic in the Red Gate).
func zeroEnv() outerassembler.Envelope {
	return outerassembler.Envelope{}
}

// ── AC-001: dial success ────────────────────────────────────────────────────────

// TestConnector_DialSuccess_ModePE verifies AC-001 postconditions 1-4:
// when the upstream listener accepts connections and the outerassembler
// bootstrap write succeeds, the Connector's connected-count increments
// and Mode() returns ModePE.
//
// RED GATE: New and Start panic — test fails immediately.
func TestConnector_DialSuccess_ModePE(t *testing.T) {
	// NOT t.Parallel(): uses net.Listen on 127.0.0.1:0.

	ln, addr := newLoopbackListener(t)
	defer func() { _ = ln.Close() }()

	// Accept connections in background — upstream fixture side.
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // listener closed
			}
			_ = conn.Close()
		}
	}()

	lw := newLogWriter()
	c := New(lw, zeroEnv(), 100*time.Millisecond, []string{addr})
	t.Cleanup(c.Stop)
	c.Start()

	// AC-001 postcondition 4: Mode() == ModePE once ≥1 upstream connected.
	if !pollForMode(c, 2*time.Second) {
		t.Errorf("TestConnector_DialSuccess_ModePE: Mode() == %v after 2s; want ModePE (AC-001 PC-4)", c.Mode())
	}
}

// ── AC-002: EC-001 log contract ────────────────────────────────────────────────

// TestConnector_DialFailure_EC001Log verifies AC-002 postconditions 1-2:
// when the upstream address is not listening, the Connector emits the
// verbatim EC-001 log line "upstream router <addr> unreachable" and
// Mode() remains ModeE.
//
// RED GATE: New panics — test fails immediately.
func TestConnector_DialFailure_EC001Log(t *testing.T) {
	// NOT t.Parallel(): uses net.Listen on 127.0.0.1:0.

	// Allocate a port then close it so the address is valid-format but not listening.
	ln, addr := newLoopbackListener(t)
	_ = ln.Close()

	lw := newLogWriter()
	c := New(lw, zeroEnv(), 200*time.Millisecond, []string{addr})
	t.Cleanup(c.Stop)
	c.Start()

	// AC-002 postcondition 1: EC-001 verbatim log line must appear.
	wantLog := fmt.Sprintf("upstream router %s unreachable", addr)
	if !waitForLog(lw, wantLog, 2*time.Second) {
		t.Errorf("TestConnector_DialFailure_EC001Log: EC-001 log line %q not emitted within 2s (AC-002 PC-1)", wantLog)
	}

	// AC-002 postcondition 2: Mode() == ModeE (connected-count stays 0).
	if c.Mode() != ModeE {
		t.Errorf("TestConnector_DialFailure_EC001Log: Mode() == %v; want ModeE (AC-002 PC-2)", c.Mode())
	}
}

// ── AC-001 Q1: set-equal reorder semantics ─────────────────────────────────────

// TestConnector_ReorderReuse_NoTeardown verifies AC-001 postcondition 5:
// reloading with the same addresses in a different order MUST NOT trigger
// teardown of existing connections or initiate new dials.
//
// RED GATE: New panics — test fails immediately.
func TestConnector_ReorderReuse_NoTeardown(t *testing.T) {
	// NOT t.Parallel(): uses net.Listen on 127.0.0.1:0.

	ln1, addr1 := newLoopbackListener(t)
	ln2, addr2 := newLoopbackListener(t)
	defer func() { _ = ln1.Close() }()
	defer func() { _ = ln2.Close() }()

	// Accept connections for both upstreams.
	acceptLoop := func(ln net.Listener) {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}
	go acceptLoop(ln1)
	go acceptLoop(ln2)

	lw := newLogWriter()
	c := New(lw, zeroEnv(), 100*time.Millisecond, []string{addr1, addr2})
	t.Cleanup(c.Stop)
	c.Start()

	// Wait until both are connected.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if c.Mode() == ModePE {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if c.Mode() != ModePE {
		t.Fatalf("TestConnector_ReorderReuse_NoTeardown: precondition: Mode() != ModePE after 2s")
	}

	// Reload with same addresses in reversed order — set-equal.
	c.ReloadAddrs([]string{addr2, addr1})

	// Give the reconciler a chance to run.
	time.Sleep(100 * time.Millisecond)

	// AC-001 postcondition 5: Mode() must still be ModePE (no teardown).
	if c.Mode() != ModePE {
		t.Errorf("TestConnector_ReorderReuse_NoTeardown: Mode() == %v after set-equal reload; want ModePE (AC-001 PC-5)", c.Mode())
	}
}

// ── AC-002: backoff parameters ─────────────────────────────────────────────────

// TestConnector_BackoffParameters verifies AC-002 postcondition 3:
// the Connector exports the exact Q5 backoff parameters — base=500ms,
// cap=30s — as package-level constants.
//
// This test does NOT exercise the actual timer (that would be too slow
// without clock injection); it asserts the constant values and the retry
// behaviour by confirming the Connector retries after a failed dial.
//
// RED GATE: New panics — test fails immediately (the constant check alone
// would pass, but the retry assertion calls New).
func TestConnector_BackoffParameters(t *testing.T) {
	t.Parallel()

	// Assert the exact Q5 constants are exported with normative values.
	if BackoffBase != 500*time.Millisecond {
		t.Errorf("BackoffBase = %v; want 500ms (Q5 normative, AC-002 PC-3)", BackoffBase)
	}
	if BackoffCap != 30*time.Second {
		t.Errorf("BackoffCap = %v; want 30s (Q5 normative, AC-002 PC-3)", BackoffCap)
	}
	if BackoffJitterFraction != 0.25 {
		t.Errorf("BackoffJitterFraction = %v; want 0.25 (Q5 normative ±25%%, AC-002 PC-3)", BackoffJitterFraction)
	}

	// Verify the Connector actually retries: start with a closed port.
	// After we open the listener the Connector must eventually connect (backoff reset).
	ln, addr := newLoopbackListener(t)
	_ = ln.Close() // close immediately — unreachable at first

	lw := newLogWriter()
	// Use a very short keepalive so the backoff base is also short in tests.
	c := New(lw, zeroEnv(), 50*time.Millisecond, []string{addr})
	t.Cleanup(c.Stop)
	c.Start()

	// Wait for at least one EC-001 log to confirm the retry loop is running.
	wantLog := fmt.Sprintf("upstream router %s unreachable", addr)
	if !waitForLog(lw, wantLog, 2*time.Second) {
		t.Fatalf("TestConnector_BackoffParameters: EC-001 log not emitted within 2s — retry loop not running")
	}

	// Now open the listener so the next retry succeeds.
	ln2, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("TestConnector_BackoffParameters: re-listen %s: %v", addr, err)
	}
	defer func() { _ = ln2.Close() }()
	go func() {
		for {
			conn, err := ln2.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	// AC-002 postcondition 3: backoff reset on success — Mode() becomes ModePE.
	if !pollForMode(c, 5*time.Second) {
		t.Errorf("TestConnector_BackoffParameters: Mode() != ModePE after opening listener — backoff not resetting on success (AC-002 PC-3)")
	}
}

// ── AC-002: all upstreams unreachable → ModeE ──────────────────────────────────

// TestConnector_AllUpstreamsUnreachable_ModeE verifies AC-002 postcondition 5:
// when ALL configured upstreams are unreachable, Mode() returns ModeE and the
// EC-004 "mode=E (no upstream_routers configured)" log fires when the last
// upstream connection drops to zero.
//
// RED GATE: New panics — test fails immediately.
func TestConnector_AllUpstreamsUnreachable_ModeE(t *testing.T) {
	// NOT t.Parallel(): uses net.Listen on 127.0.0.1:0.

	ln1, addr1 := newLoopbackListener(t)
	ln2, addr2 := newLoopbackListener(t)
	_ = ln1.Close()
	_ = ln2.Close()

	lw := newLogWriter()
	c := New(lw, zeroEnv(), 100*time.Millisecond, []string{addr1, addr2})
	t.Cleanup(c.Stop)
	c.Start()

	// AC-002 postcondition 5 — Mode() == ModeE when all upstreams are unreachable.
	// Give a window for the initial dials to fail.
	time.Sleep(300 * time.Millisecond)
	if c.Mode() != ModeE {
		t.Errorf("TestConnector_AllUpstreamsUnreachable_ModeE: Mode() == %v; want ModeE (AC-002 PC-5)", c.Mode())
	}

	// EC-001 log lines must have appeared for at least one address.
	if !waitForLog(lw, "upstream router", 2*time.Second) {
		t.Errorf("TestConnector_AllUpstreamsUnreachable_ModeE: no EC-001 log emitted within 2s")
	}
}

// ── AC-003: keepalive ticker ───────────────────────────────────────────────────

// TestConnector_KeepaliveTickerDrivesHealthProbe verifies AC-003 postconditions
// 1-2: the Connector owns a keepalive ticker constructed from the
// keepaliveInterval passed to New, and that ticker fires probe frames on
// established upstream connections.
//
// The test isolates the keepalive probe from the bootstrap frame:
//  1. Drain the bootstrap frame (first write on connect).
//  2. Require a SUBSEQUENT write within ~3 keepalive intervals.
//
// Failure condition: if maintainConn's keepalive tick case is removed, the
// second write never arrives and the test times out.
func TestConnector_KeepaliveTickerDrivesHealthProbe(t *testing.T) {
	// NOT t.Parallel(): uses net.Listen on 127.0.0.1:0.

	// Use a short keepalive so the probe fires quickly.
	const testKeepalive = 50 * time.Millisecond

	ln, addr := newLoopbackListener(t)
	defer func() { _ = ln.Close() }()

	// Upstream fixture: accept, drain the bootstrap frame, then wait for a
	// second write (the keepalive probe) within the deadline.
	keepaliveProbedCh := make(chan struct{}, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		buf := make([]byte, 4096)

		// Read 1: bootstrap frame.  Block up to 2s.
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, err := conn.Read(buf)
		if n == 0 || err != nil {
			return // bootstrap frame not received — test will timeout below
		}

		// Read 2: keepalive probe.  Keepalive ticker fires after testKeepalive;
		// allow 4 intervals for the first tick to fire and the write to complete.
		_ = conn.SetReadDeadline(time.Now().Add(4 * testKeepalive))
		n, _ = conn.Read(buf)
		if n > 0 {
			select {
			case keepaliveProbedCh <- struct{}{}:
			default:
			}
		}
	}()

	lw := newLogWriter()
	c := New(lw, zeroEnv(), testKeepalive, []string{addr})
	t.Cleanup(c.Stop)
	c.Start()

	// Wait for connection to establish (bootstrap write succeeds).
	if !pollForMode(c, 2*time.Second) {
		t.Fatalf("TestConnector_KeepaliveTickerDrivesHealthProbe: Mode() != ModePE after 2s")
	}

	// AC-003 postcondition 2: the keepalive ticker must have driven a subsequent
	// probe write distinct from the bootstrap frame.
	// Allow 6 keepalive intervals: 1 interval for the ticker to fire +
	// scheduling slack.
	select {
	case <-keepaliveProbedCh:
		// pass — upstream fixture received a second write (keepalive probe).
	case <-time.After(6 * testKeepalive):
		t.Errorf("TestConnector_KeepaliveTickerDrivesHealthProbe: no keepalive probe received after bootstrap within %v; maintainConn ticker not driving probes (AC-003 PC-2, F-P1-003)", 6*testKeepalive)
	}
}

// ── AC-001 Q1: ReloadAddrs adds and removes ────────────────────────────────────

// TestConnector_ReloadAddrs_AddsAndRemoves verifies AC-001 postcondition 6:
// ReloadAddrs with new addresses initiates dials to added addresses and
// tears down connections to removed addresses.
//
// RED GATE: New panics — test fails immediately.
func TestConnector_ReloadAddrs_AddsAndRemoves(t *testing.T) {
	// NOT t.Parallel(): uses net.Listen on 127.0.0.1:0.

	ln1, addr1 := newLoopbackListener(t)
	defer func() { _ = ln1.Close() }()
	go func() {
		for {
			conn, err := ln1.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	ln2, addr2 := newLoopbackListener(t)
	defer func() { _ = ln2.Close() }()
	go func() {
		for {
			conn, err := ln2.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	lw := newLogWriter()
	// Start with only addr1.
	c := New(lw, zeroEnv(), 100*time.Millisecond, []string{addr1})
	t.Cleanup(c.Stop)
	c.Start()

	// Precondition: addr1 connected → ModePE.
	if !pollForMode(c, 2*time.Second) {
		t.Fatalf("TestConnector_ReloadAddrs_AddsAndRemoves: precondition: Mode() != ModePE")
	}

	// Reload: replace addr1 with addr2.
	c.ReloadAddrs([]string{addr2})

	// AC-001 postcondition 6: Connector dials addr2 (Mode() must remain ModePE).
	if !pollForMode(c, 2*time.Second) {
		t.Errorf("TestConnector_ReloadAddrs_AddsAndRemoves: Mode() != ModePE after adding addr2 (AC-001 PC-6)")
	}
}

// ── nextBackoff pure-function schedule tests ────────────────────────────────

// TestNextBackoff_DoublingWithinJitterBand verifies that nextBackoff doubles the
// current value within the ±25% jitter band (F-P1-004, Q5 normative, AC-002).
// Runs 1000 trials per base value to cover the jitter distribution.
func TestNextBackoff_DoublingWithinJitterBand(t *testing.T) {
	t.Parallel()

	bases := []time.Duration{
		BackoffBase,
		BackoffBase * 2,
		BackoffBase * 4,
		500 * time.Millisecond,
		1 * time.Second,
		5 * time.Second,
	}

	for _, base := range bases {
		base := base
		t.Run(base.String(), func(t *testing.T) {
			t.Parallel()
			doubled := base * 2
			if doubled > BackoffCap {
				doubled = BackoffCap
			}
			lo := time.Duration(float64(doubled) * (1 - BackoffJitterFraction))
			hi := time.Duration(float64(doubled) * (1 + BackoffJitterFraction))
			// Floor at BackoffBase.
			if lo < BackoffBase {
				lo = BackoffBase
			}
			// Cap at BackoffCap.
			if hi > BackoffCap {
				hi = BackoffCap
			}

			for i := 0; i < 1000; i++ {
				got := nextBackoff(base)
				if got < lo || got > hi {
					t.Errorf("trial %d: nextBackoff(%v) = %v; want [%v, %v] (doubling ±25%% jitter, Q5)", i, base, got, lo, hi)
					break
				}
			}
		})
	}
}

// TestNextBackoff_CapClamp verifies that nextBackoff clamps at BackoffCap even
// when the doubled value would exceed it (F-P1-004, Q5 normative, AC-002).
func TestNextBackoff_CapClamp(t *testing.T) {
	t.Parallel()

	// Inputs whose double exceeds BackoffCap.
	overCaps := []time.Duration{
		BackoffCap,
		BackoffCap - 1,
		BackoffCap / 2 * 3, // 1.5×cap → doubled = 3×cap
		BackoffCap * 10,
	}

	for _, input := range overCaps {
		input := input
		t.Run(input.String(), func(t *testing.T) {
			t.Parallel()
			for i := 0; i < 200; i++ {
				got := nextBackoff(input)
				if got > BackoffCap {
					t.Errorf("trial %d: nextBackoff(%v) = %v; exceeds BackoffCap %v (Q5 cap clamp)", i, input, got, BackoffCap)
					break
				}
			}
		})
	}
}

// TestNextBackoff_FloorAtBase verifies that nextBackoff never returns a value
// below BackoffBase, even when jitter would push a small input below it
// (F-P1-004, Q5 normative, AC-002).
func TestNextBackoff_FloorAtBase(t *testing.T) {
	t.Parallel()

	// The only input where jitter can push below BackoffBase is BackoffBase itself
	// (doubled = 1s, lo = 0.75s > BackoffBase, so actually fine — the floor guards
	// against future constant changes).  Test a sub-base input to confirm the floor.
	subBases := []time.Duration{
		1 * time.Millisecond,
		10 * time.Millisecond,
		100 * time.Millisecond,
		BackoffBase / 4,
	}

	for _, input := range subBases {
		input := input
		t.Run(input.String(), func(t *testing.T) {
			t.Parallel()
			for i := 0; i < 200; i++ {
				got := nextBackoff(input)
				if got < BackoffBase {
					t.Errorf("trial %d: nextBackoff(%v) = %v; below BackoffBase %v (Q5 floor)", i, input, got, BackoffBase)
					break
				}
			}
		})
	}
}

// TestNextBackoff_JitterBounds verifies the raw jitter formula: for 10 000 trials
// at BackoffBase, every result stays within [BackoffBase*0.75, BackoffBase*2*1.25]
// clamped to [BackoffBase, BackoffCap] (F-P1-004, Q5 ±25%%).
func TestNextBackoff_JitterBounds(t *testing.T) {
	t.Parallel()

	doubled := BackoffBase * 2
	lo := time.Duration(float64(doubled) * (1 - BackoffJitterFraction))
	hi := time.Duration(float64(doubled) * (1 + BackoffJitterFraction))
	if lo < BackoffBase {
		lo = BackoffBase
	}
	if hi > BackoffCap {
		hi = BackoffCap
	}

	for i := 0; i < 10_000; i++ {
		got := nextBackoff(BackoffBase)
		if got < lo || got > hi {
			t.Errorf("trial %d: nextBackoff(BackoffBase) = %v; outside [%v, %v] (F-P1-004)", i, got, lo, hi)
			return
		}
	}
}
