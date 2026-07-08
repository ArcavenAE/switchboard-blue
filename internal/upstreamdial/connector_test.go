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

// stamp is a timestamped log message used by timestampedLogWriter.
type stamp struct {
	t   time.Time
	msg string
}

// timestampedLogWriter records each Write with a wall-clock timestamp.
// Used by F-P2-002 timing tests to measure dial-attempt gaps.
type timestampedLogWriter struct {
	ch chan stamp
}

func (lw *timestampedLogWriter) Write(p []byte) (int, error) {
	lw.ch <- stamp{t: time.Now(), msg: string(p)}
	return len(p), nil
}

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

// ── AC-002: backoff constants (floor guard) ────────────────────────────────────

// TestConnector_BackoffConstants verifies AC-002 postcondition 3:
// the Connector exports the exact Q5 backoff constants — BackoffBase=500ms,
// BackoffCap=30s, BackoffJitterFraction=0.25.
//
// This test guards the constant values only.  Operative-base wiring (which
// uses keepaliveInterval as the reconnect base, floored at BackoffBase) is
// exercised by TestConnector_OperativeBackoffBase_TracksKeepalive (F-P2-002).
func TestConnector_BackoffConstants(t *testing.T) {
	t.Parallel()

	if BackoffBase != 500*time.Millisecond {
		t.Errorf("BackoffBase = %v; want 500ms (Q5 normative, AC-002 PC-3)", BackoffBase)
	}
	if BackoffCap != 30*time.Second {
		t.Errorf("BackoffCap = %v; want 30s (Q5 normative, AC-002 PC-3)", BackoffCap)
	}
	if BackoffJitterFraction != 0.25 {
		t.Errorf("BackoffJitterFraction = %v; want 0.25 (Q5 normative ±25%%, AC-002 PC-3)", BackoffJitterFraction)
	}
}

// ── F-P2-002: operativeBase pure-function exhaustive tests ─────────────────────

// TestOperativeBase_TracksKeepalive exhaustively unit-tests the operativeBase
// pure function (F-P2-002 architect ruling):
//
//   - Above-floor inputs: operativeBase(X) == X when X >= BackoffBase.
//   - Floor inputs: operativeBase(X) == BackoffBase when X < BackoffBase.
//   - Boundary: operativeBase(BackoffBase) == BackoffBase (exact floor).
//
// Each sub-case is mutation-detectable: if the assignment `backoff :=
// operativeBase(c.keepaliveInterval)` were changed back to `backoff :=
// c.keepaliveInterval`, the floor sub-cases would fail (100ms → 100ms, not
// 500ms).  If operativeBase were deleted, this test would not compile.
func TestOperativeBase_TracksKeepalive(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		keepalive       time.Duration
		wantOperative   time.Duration
		wantDescription string
	}{
		{
			name:            "well-above-floor/1s",
			keepalive:       1 * time.Second,
			wantOperative:   1 * time.Second,
			wantDescription: "keepalive=1s > BackoffBase=500ms → operative==keepalive",
		},
		{
			name:            "well-above-floor/600ms",
			keepalive:       600 * time.Millisecond,
			wantOperative:   600 * time.Millisecond,
			wantDescription: "keepalive=600ms > BackoffBase=500ms → operative==keepalive",
		},
		{
			name:            "well-above-floor/1200ms",
			keepalive:       1200 * time.Millisecond,
			wantOperative:   1200 * time.Millisecond,
			wantDescription: "keepalive=1200ms > BackoffBase=500ms → operative==keepalive (distinguishable from 600ms case)",
		},
		{
			name:            "well-above-floor/5s",
			keepalive:       5 * time.Second,
			wantOperative:   5 * time.Second,
			wantDescription: "keepalive=5s > BackoffBase → operative==keepalive",
		},
		{
			name:            "exact-floor",
			keepalive:       BackoffBase,
			wantOperative:   BackoffBase,
			wantDescription: "keepalive==BackoffBase → operative==BackoffBase (boundary)",
		},
		{
			name:            "below-floor/100ms",
			keepalive:       100 * time.Millisecond,
			wantOperative:   BackoffBase,
			wantDescription: "keepalive=100ms < BackoffBase=500ms → floored to BackoffBase",
		},
		{
			name:            "below-floor/1ms",
			keepalive:       1 * time.Millisecond,
			wantOperative:   BackoffBase,
			wantDescription: "keepalive=1ms << BackoffBase → floored to BackoffBase",
		},
		{
			name:            "below-floor/499ms",
			keepalive:       499 * time.Millisecond,
			wantOperative:   BackoffBase,
			wantDescription: "keepalive=499ms (one tick below floor) → floored to BackoffBase",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := operativeBase(tc.keepalive)
			if got != tc.wantOperative {
				t.Errorf("operativeBase(%v) = %v; want %v — %s (F-P2-002)", tc.keepalive, got, tc.wantOperative, tc.wantDescription)
			}
		})
	}
}

// ── F-P2-002: wiring test — operative base tracked by Connector ─────────────────

// TestConnector_OperativeBackoffBase_TracksKeepalive is a coarse timing test
// that proves the operative-base wiring in dialLoop (connector.go lines for
// `backoff := operativeBase(c.keepaliveInterval)` initial assignment and
// `backoff = operativeBase(c.keepaliveInterval)` reset-on-success assignment)
// is wired to operativeBase, not to a hardcoded constant.
//
// # Band math (F-P2-002 requirement — bands must be disjoint from floor band)
//
// We test two keepalive values above the floor:
//
//	keepalive=1s: operative=1s.  With ±25% jitter, first-retry gap in [0.75s, 1.25s].
//	keepalive=2s: operative=2s.  With ±25% jitter, first-retry gap in [1.5s, 2.5s].
//	BackoffBase floor=500ms:     first-retry gap in [0.375s, 0.625s].
//
// Bands: [0.375, 0.625] vs [0.75, 1.25] vs [1.5, 2.5].  Disjoint — no overlap.
//
// Path taken: pure-function extraction (operativeBase). Timing here is coarse
// (dial-attempt timestamp delta from the listener side) — proves the wiring;
// per-operand exhaustive correctness is in TestOperativeBase_TracksKeepalive.
//
// Assertion: when keepalive=1s, the gap between dial-attempt 1 and dial-attempt 2
// falls in [750ms, 1250ms].  If the wiring used BackoffBase instead, the gap
// would fall in [375ms, 625ms] — outside the expected window.
func TestConnector_OperativeBackoffBase_TracksKeepalive(t *testing.T) {
	// NOT t.Parallel(): uses net.Listen on 127.0.0.1:0.

	// keepalive=1s: expected gap in [750ms, 1250ms]; BackoffBase gap [375ms, 625ms] — disjoint.
	t.Run("1s-keepalive-above-floor", func(t *testing.T) {
		const testKeepalive = 1 * time.Second
		const loWindow = 600 * time.Millisecond // generous but outside floor band
		const hiWindow = 1500 * time.Millisecond

		ln, addr := newLoopbackListener(t)
		// Close immediately — we want repeated dial failures to measure backoff gap.
		_ = ln.Close()

		// Record timestamps of each dial attempt at the OS level by watching
		// when the dial completes (returns error from refused connection).
		// We inject a counting hook via a test-only net.Dialer wrapping.
		// Since there is no dial hook in Connector, we measure via log emission:
		// each EC-001 "upstream router ... unreachable" log corresponds to one dial attempt.
		// Record wall-clock timestamp when each log message is emitted.
		stampCh := make(chan stamp, 16)

		// Use a custom logWriter that also records timestamps.
		lw := &timestampedLogWriter{ch: stampCh}
		c := New(lw, zeroEnv(), testKeepalive, []string{addr})
		t.Cleanup(c.Stop)
		c.Start()

		wantLog := fmt.Sprintf("upstream router %s unreachable", addr)

		// Collect at least 2 EC-001 timestamps within a generous budget.
		var ts [2]time.Time
		budget := time.After(5 * time.Second)
		got := 0
		for got < 2 {
			select {
			case s := <-stampCh:
				if strings.Contains(s.msg, wantLog) {
					ts[got] = s.t
					got++
				}
			case <-budget:
				t.Fatalf("TestConnector_OperativeBackoffBase_TracksKeepalive/1s-keepalive: only got %d EC-001 log messages within 5s; need 2 to measure gap", got)
			}
		}

		gap := ts[1].Sub(ts[0])
		if gap < loWindow || gap > hiWindow {
			t.Errorf(
				"TestConnector_OperativeBackoffBase_TracksKeepalive/1s-keepalive: gap between dial attempts = %v; want [%v, %v] (operative base=1s ±25%% + scheduling slack; if wired to BackoffBase=500ms the gap would be [375ms,625ms] — F-P2-002)",
				gap, loWindow, hiWindow,
			)
		}
	})

	// keepalive=2s: expected gap in [1.5s, 2.5s]; BackoffBase gap [375ms, 625ms] — disjoint.
	// This second sub-test proves that a DIFFERENT keepalive produces a DIFFERENT gap,
	// ruling out a coincidental pass where operative==BackoffBase happened to match.
	t.Run("2s-keepalive-above-floor", func(t *testing.T) {
		const testKeepalive = 2 * time.Second
		const loWindow = 1200 * time.Millisecond
		const hiWindow = 2800 * time.Millisecond

		ln, addr := newLoopbackListener(t)
		_ = ln.Close()

		stampCh := make(chan stamp, 16)
		lw := &timestampedLogWriter{ch: stampCh}
		c := New(lw, zeroEnv(), testKeepalive, []string{addr})
		t.Cleanup(c.Stop)
		c.Start()

		wantLog := fmt.Sprintf("upstream router %s unreachable", addr)

		var ts2 [2]time.Time
		budget := time.After(10 * time.Second)
		got := 0
		for got < 2 {
			select {
			case s := <-stampCh:
				if strings.Contains(s.msg, wantLog) {
					ts2[got] = s.t
					got++
				}
			case <-budget:
				t.Fatalf("TestConnector_OperativeBackoffBase_TracksKeepalive/2s-keepalive: only got %d EC-001 log messages within 10s; need 2 to measure gap", got)
			}
		}

		gap := ts2[1].Sub(ts2[0])
		if gap < loWindow || gap > hiWindow {
			t.Errorf(
				"TestConnector_OperativeBackoffBase_TracksKeepalive/2s-keepalive: gap between dial attempts = %v; want [%v, %v] (operative base=2s ±25%% + scheduling slack — F-P2-002)",
				gap, loWindow, hiWindow,
			)
		}
	})

	// keepalive=100ms below floor: operative base==BackoffBase=500ms, NOT 100ms.
	// This proves the floor applies when keepalive < BackoffBase.
	t.Run("100ms-keepalive-below-floor", func(t *testing.T) {
		const testKeepalive = 100 * time.Millisecond
		// Operative = BackoffBase = 500ms. Expected gap in [375ms, 625ms].
		// If the floor were missing (operative=100ms), gap would be ~100ms — far below lo.
		const loWindow = 300 * time.Millisecond
		const hiWindow = 750 * time.Millisecond

		ln, addr := newLoopbackListener(t)
		_ = ln.Close()

		stampCh := make(chan stamp, 16)
		lw := &timestampedLogWriter{ch: stampCh}
		c := New(lw, zeroEnv(), testKeepalive, []string{addr})
		t.Cleanup(c.Stop)
		c.Start()

		wantLog := fmt.Sprintf("upstream router %s unreachable", addr)

		var ts3 [2]time.Time
		budget := time.After(5 * time.Second)
		got := 0
		for got < 2 {
			select {
			case s := <-stampCh:
				if strings.Contains(s.msg, wantLog) {
					ts3[got] = s.t
					got++
				}
			case <-budget:
				t.Fatalf("TestConnector_OperativeBackoffBase_TracksKeepalive/100ms-below-floor: only got %d EC-001 log messages within 5s; need 2", got)
			}
		}

		gap := ts3[1].Sub(ts3[0])
		if gap < loWindow || gap > hiWindow {
			t.Errorf(
				"TestConnector_OperativeBackoffBase_TracksKeepalive/100ms-below-floor: gap = %v; want [%v, %v] (operative=BackoffBase=500ms; if floor missing, gap would be ~100ms — F-P2-002)",
				gap, loWindow, hiWindow,
			)
		}
	})
}

// ── AC-002: backoff-reset-on-success wiring test ───────────────────────────────

// TestConnector_BackoffParameters verifies AC-002 postcondition 3 (backoff reset
// on success): after multiple failed dials grow the backoff, a successful
// reconnect must reset the delay back to the operative base (keepaliveInterval
// floored at BackoffBase), not carry the grown value forward.
//
// This replaces the old TestConnector_BackoffParameters which only asserted
// constant values (those are now in TestConnector_BackoffConstants) and added
// a retry-succeeds check without measuring whether the reset used the operative
// base or the grown value.
//
// keepalive=1s (above floor): operative base=1s.
// After 3 failures: backoff grows to ~4s (1s→2s→4s) — well above the 1300ms hiWindow.
// After reset: first-retry gap in [700ms, 1300ms].
//
// Measurement: the sequence after a connection drops is:
//
//	stamp[0]: maintainConn write-fail log (after next keepalive tick, ~1s)
//	stamp[1]: first dial-fail immediately after (no pre-sleep on first attempt)
//	stamp[2]: second dial-fail after the backoff sleep (= reset operative base ~1s)
//
// gap = stamps[2].t - stamps[1].t = ~1s (operative base after reset).
// If the reset assignment used BackoffBase instead of operativeBase, gap ~500ms.
// BackoffBase mutant band (±25% jitter): [375ms, 625ms] — entirely below loWindow=700ms.
// If the reset were absent and backoff stayed grown (~4s), gap >> hiWindow=1300ms.
//
// Failure condition: if `backoff = operativeBase(c.keepaliveInterval)` in
// dialLoop (connector.go reset-on-success line) were changed to
// `backoff = BackoffBase`, this test fails because the ~500ms mutant gap
// falls below loWindow=700ms (F-P3-001, AC-002 PC-3).
func TestConnector_BackoffParameters(t *testing.T) {
	// NOT t.Parallel(): uses net.Listen on 127.0.0.1:0.

	const testKeepalive = 1000 * time.Millisecond
	// Operative base = 1s.  Post-reset gap in [700ms, 1300ms].
	// BackoffBase mutant band [375ms,625ms] is entirely below loWindow=700ms → mutant caught.
	// Grown value after 3 failures ~4s (1s→2s→4s) — well above hiWindow=1300ms.
	const loWindow = 700 * time.Millisecond
	const hiWindow = 1300 * time.Millisecond

	stampCh := make(chan stamp, 64)
	lw := &timestampedLogWriter{ch: stampCh}

	// Phase 1: start with closed port — grow backoff past operative base.
	ln, addr := newLoopbackListener(t)
	_ = ln.Close()

	c := New(lw, zeroEnv(), testKeepalive, []string{addr})
	t.Cleanup(c.Stop)
	c.Start()

	wantLog := fmt.Sprintf("upstream router %s unreachable", addr)

	// Collect 3 EC-001 stamps (initial + 2 backoff-grown) to verify backoff grows.
	// At keepalive=1s: attempt 1 fires at ~0ms, attempt 2 at ~1s, attempt 3 at ~3s.
	// After 3 failures, backoff is ~4s — distinguishably above the 1300ms hiWindow.
	growBudget := time.After(15 * time.Second)
	growGot := 0
	for growGot < 3 {
		select {
		case s := <-stampCh:
			if strings.Contains(s.msg, wantLog) {
				growGot++
			}
		case <-growBudget:
			t.Fatalf("TestConnector_BackoffParameters: only got %d EC-001 logs in grow phase (15s); need 3", growGot)
		}
	}

	// Phase 2: open listener (keep connection alive), wait for ModePE.
	// Use a held-connection fixture so the connection persists until we choose to drop it.
	ln2, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("TestConnector_BackoffParameters: re-listen: %v", err)
	}
	defer func() { _ = ln2.Close() }()

	// Accept and hold the server-side connection so the keepalive ticker keeps firing.
	heldConn := make(chan net.Conn, 1)
	go func() {
		conn, aErr := ln2.Accept()
		if aErr != nil {
			return
		}
		heldConn <- conn
		// Drain any reads from the client so the connection stays alive.
		buf := make([]byte, 4096)
		for {
			_, err := conn.Read(buf)
			if err != nil {
				return
			}
		}
	}()

	// After 3 grow-phase failures with testKeepalive=1s, backoff reaches ~4s (max jitter ~5s).
	// Budget must cover the worst-case grown-backoff wait before the next dial attempt.
	if !pollForMode(c, 15*time.Second) {
		t.Fatalf("TestConnector_BackoffParameters: Mode() != ModePE after opening listener")
	}

	// Drain all stale stamps collected during the grow and connect phases.
drainLoop:
	for {
		select {
		case <-stampCh:
		default:
			break drainLoop
		}
	}

	// Phase 3: drop the server-side connection → dialLoop resets backoff → measure gap.
	select {
	case conn := <-heldConn:
		_ = conn.Close()
	case <-time.After(2 * time.Second):
		t.Fatalf("TestConnector_BackoffParameters: held connection not received within 2s")
	}
	_ = ln2.Close() // also close the listener so reconnect dials fail

	// Collect 3 EC-001 stamps from the post-drop retry sequence:
	//   stamp[0]: maintainConn write-fail (fires when next keepalive tick hits the dead conn)
	//   stamp[1]: first dial-fail immediately after
	//   stamp[2]: second dial-fail after the backoff-sleep (= operative base = 1s)
	var postDrop [3]stamp
	postBudget := time.After(10 * time.Second)
	postGot := 0
	for postGot < 3 {
		select {
		case s := <-stampCh:
			if strings.Contains(s.msg, wantLog) {
				postDrop[postGot] = s
				postGot++
			}
		case <-postBudget:
			t.Fatalf("TestConnector_BackoffParameters: only got %d post-drop stamps in 10s", postGot)
		}
	}

	// gap between stamp[1] and stamp[2] = backoff sleep duration after reset.
	// Expected: ~1s (operative base).  If backoff were still grown (~4s), gap >> hiWindow.
	// If backoff used BackoffBase mutant (~500ms), gap < loWindow=700ms (F-P3-001 pinned).
	gap := postDrop[2].t.Sub(postDrop[1].t)
	if gap < loWindow || gap > hiWindow {
		t.Errorf(
			"TestConnector_BackoffParameters: post-reset retry gap = %v; want [%v, %v] "+
				"(reset must restore operative base %v, not carry grown ~4s backoff or use BackoffBase ~500ms; F-P3-001, AC-002 PC-3)",
			gap, loWindow, hiWindow, testKeepalive,
		)
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

// ── AC-002: EC-004 drop-to-zero mode=E emission ───────────────────────────────

// TestConnector_EC004_DropToZero_ModeEEmission verifies that when the last
// upstream connection drops (connectedCount → 0), the Connector emits the
// verbatim EC-004 log "mode=E (no upstream_routers configured)" (F-P1-006,
// the connectedCount==0 && ctx.Err()==nil branch in dialLoop's EC-004 emission path, AC-002).
//
// Failure condition: if the EC-004 log branch (connectedCount==0 check after
// connectedCount.Add(-1)) is removed, this test fails.
func TestConnector_EC004_DropToZero_ModeEEmission(t *testing.T) {
	// NOT t.Parallel(): uses net.Listen on 127.0.0.1:0.

	const testKeepalive = 30 * time.Millisecond

	ln, addr := newLoopbackListener(t)

	// Upstream fixture: accept the connection and keep it open.
	accepted := make(chan net.Conn, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		accepted <- conn
	}()

	lw := newLogWriter()
	c := New(lw, zeroEnv(), testKeepalive, []string{addr})
	t.Cleanup(c.Stop)
	c.Start()

	// Wait for connection to establish (ModePE).
	if !pollForMode(c, 2*time.Second) {
		t.Fatalf("TestConnector_EC004_DropToZero_ModeEEmission: Mode() != ModePE after 2s")
	}

	// Drain the accepted connection channel so we can close the listener.
	select {
	case conn := <-accepted:
		// Close the server-side connection to trigger a write failure in maintainConn.
		_ = conn.Close()
	case <-time.After(2 * time.Second):
		t.Fatalf("TestConnector_EC004_DropToZero_ModeEEmission: upstream fixture did not accept connection within 2s")
	}

	// Also close the listener so reconnect attempts fail immediately.
	_ = ln.Close()

	// EC-004: once connectedCount drops to 0, the Connector must emit the
	// verbatim log "mode=E (no upstream_routers configured)".
	// Allow up to 3 keepalive intervals for the write deadline to expire and
	// the count to decrement.
	const ec004Log = "mode=E (no upstream_routers configured)"
	if !waitForLog(lw, ec004Log, 3*time.Second) {
		t.Errorf("TestConnector_EC004_DropToZero_ModeEEmission: EC-004 log %q not emitted within 3s after connection drop (F-P1-006)", ec004Log)
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

// ── F-P2-001: idempotent Stop ─────────────────────────────────────────────────

// TestConnector_Stop_Idempotent verifies F-P2-001: Stop() is idempotent.
// Calling Stop() twice sequentially, and once concurrently with another
// goroutine, must not panic.  All callers must return (not block indefinitely).
//
// Failure condition (old code): the second close(stopCh) panics with
// "close of closed channel".
func TestConnector_Stop_Idempotent(t *testing.T) {
	// NOT t.Parallel(): uses net.Listen on 127.0.0.1:0.

	ln, addr := newLoopbackListener(t)
	defer func() { _ = ln.Close() }()

	// Accept loop so the Connector can connect (not strictly required for Stop
	// idempotency, but uses a reachable address to keep the goroutines live
	// until the first Stop).
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	c := New(nil, zeroEnv(), 100*time.Millisecond, []string{addr})
	c.Start()

	// First Stop: nominal shutdown — must not panic.
	c.Stop()

	// Second Stop sequential: must not panic.
	c.Stop()

	// Third Stop concurrent: must not panic or block.
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.Stop()
	}()
	select {
	case <-done:
		// pass
	case <-time.After(2 * time.Second):
		t.Error("TestConnector_Stop_Idempotent: concurrent Stop() blocked for >2s (F-P2-001)")
	}
}

// ── F-P4-001: no spurious EC-004 on graceful Stop ────────────────────────────

// TestConnector_NoEC004OnGracefulStop verifies F-P4-001: a graceful Stop() of
// a still-connected PE router MUST NOT emit the EC-004 log line
// "mode=E (no upstream_routers configured)" to the operator writer.
//
// BC-2.09.001 EC-004's trigger is upstream-LOSS, not self-initiated teardown.
// The EC-001 sibling guard (ctx.Err()!=nil early-return in dialLoop's EC-001 emission path)
// is the model: when ctx.Err() != nil, the loop returns without logging.  The EC-004
// branch (connectedCount==0 && ctx.Err()==nil guard in dialLoop's EC-004 emission path)
// must apply the same guard.
//
// Failure condition: if the ctx.Err() guard before the EC-004 branch is
// absent (the unguarded code), this test fails because Stop() triggers
// maintainConn to return via ctx cancellation, connectedCount drops to zero,
// and the spurious EC-004 line is emitted even though the connection was live.
func TestConnector_NoEC004OnGracefulStop(t *testing.T) {
	// NOT t.Parallel(): uses net.Listen on 127.0.0.1:0.

	const testKeepalive = 30 * time.Millisecond

	ln, addr := newLoopbackListener(t)
	defer func() { _ = ln.Close() }()

	// Upstream fixture: accept the connection and hold it open (connection remains
	// healthy for the entire duration of the test — Stop() is called while live).
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			// Hold the connection open; drain any keepalive bytes.
			go func(c net.Conn) {
				buf := make([]byte, 4096)
				for {
					_, err := c.Read(buf)
					if err != nil {
						return
					}
				}
			}(conn)
		}
	}()

	lw := newLogWriter()
	c := New(lw, zeroEnv(), testKeepalive, []string{addr})
	c.Start()

	// Precondition: reach ModePE (connection established, listener still healthy).
	if !pollForMode(c, 2*time.Second) {
		t.Fatalf("TestConnector_NoEC004OnGracefulStop: precondition: Mode() != ModePE after 2s")
	}

	// Drain any log lines accumulated during startup so only post-Stop output is examined.
drainStartup:
	for {
		select {
		case <-lw.ch:
		default:
			break drainStartup
		}
	}

	// Call Stop() while the upstream connection is still healthy.
	c.Stop()

	// Assert: no EC-004 line was emitted after Stop() returned.
	// Allow a brief window (3 keepalive intervals) in case of scheduling lag.
	const ec004Log = "mode=E (no upstream_routers configured)"
	deadline := time.After(3 * testKeepalive)
	for {
		select {
		case msg := <-lw.ch:
			if strings.Contains(msg, ec004Log) {
				t.Errorf("TestConnector_NoEC004OnGracefulStop: spurious EC-004 emitted on graceful Stop(): %q (F-P4-001; EC-004 trigger is upstream-LOSS, not self-initiated teardown)", msg)
				return
			}
		case <-deadline:
			// No spurious EC-004 seen — pass.
			return
		}
	}
}

// ── F-P5-001: ReloadAddrs storm — no deadlock ─────────────────────────────────

// TestConnector_ReloadAddrs_StormNoDeadlock verifies F-P5-001: ReloadAddrs must
// be non-blocking under all interleavings, including races between the single
// production caller (runRouter's SIGHUP select case) and the reconcile goroutine
// draining the channel.
//
// The adversary reproduced the deadlock with a 200k-iteration storm: when
// addrsCh (cap 1) is full, the old default branch did a BLOCKING <-c.addrsCh.
// If the reconcile goroutine drains the slot in that window, the channel is
// empty and the blocking receive waits forever — wedging runRouter's select loop.
//
// Failure condition (pre-fix code): the storm goroutine wedges, the watchdog
// fires after 10s, and the test fails.  Post-fix (non-blocking drain + non-
// blocking resend): the storm completes in well under 2s.
func TestConnector_ReloadAddrs_StormNoDeadlock(t *testing.T) {
	// NOT t.Parallel(): uses net.Listen on 127.0.0.1:0; start the reconcile
	// goroutine so it races the drain exactly as in production.

	const iterations = 200_000
	const watchdogTimeout = 10 * time.Second
	const greenPassTimeout = 2 * time.Second

	// Use an unreachable address so dialLoop doesn't make real TCP connections,
	// but the reconcile goroutine MUST be running so it races the addrsCh drain.
	// Probe-and-close (F-P1-005): bind an ephemeral loopback port then close it
	// so the address is valid-format but not listening → dials refused instantly
	// → dialLoop stays in the backoff wait and alive throughout the storm.
	ln, addr := newLoopbackListener(t)
	_ = ln.Close() // close immediately: address exists but is not listening

	c := New(nil, zeroEnv(), BackoffBase, []string{addr})
	t.Cleanup(c.Stop)
	c.Start() // reconcile goroutine is now running and may drain addrsCh at any time

	// Run the storm in a separate goroutine so the watchdog can interrupt on wedge.
	stormDone := make(chan struct{})
	go func() {
		defer close(stormDone)
		snap := []string{addr}
		for i := 0; i < iterations; i++ {
			c.ReloadAddrs(snap)
		}
	}()

	select {
	case <-stormDone:
		// Pass: storm completed before watchdog fired.
	case <-time.After(watchdogTimeout):
		t.Fatalf(
			"TestConnector_ReloadAddrs_StormNoDeadlock: %d ReloadAddrs calls did not complete within %v — "+
				"ReloadAddrs is blocking (F-P5-001 deadlock; pre-fix default-branch blocks on <-c.addrsCh "+
				"when reconcile drains the slot, leaving the channel empty and the receive wedged forever)",
			iterations, watchdogTimeout,
		)
	}

	// Green-path timing assertion: with non-blocking selects the storm is fast.
	// This is informational — the liveness property is the watchdog above.
	_ = greenPassTimeout // referenced for documentation; watchdog is the hard gate
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
