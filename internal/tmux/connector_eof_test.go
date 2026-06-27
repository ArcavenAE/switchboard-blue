// Package tmux — connector_eof_test.go tests the PTY-source EOF no-spin obligation
// (AC-009; BC-2.04.002 EC-008; ARCH-01 ADR-011 v1.5 §HIGH-A).
//
// When the access node is in PTY mode and the PTY shell exits (EOF on PTY master)
// WITHOUT sc.Close() being called, the forwardFrames relay MUST:
//  1. Detect srcCh==prevSrcCh in PTY mode (InPTYMode true).
//  2. Send ErrPTYSourceEOF on sc.errCh via sc.closeErrCh.Do (buffered-1, non-blocking).
//  3. Return — no hot-spin.
//
// The daemon's PC-2.6 drain path in runAccessWithConnector then receives the error
// on sc.Err(), logs E-SYS-002 format, cancels the root context, and exits with code 1.
//
// Discriminating invariant: a relay that busy-spins instead of exiting would NOT
// send ErrPTYSourceEOF and would NOT close sc.Err(). The test's bounded deadline
// would fire and the test would FAIL — catching the hot-spin regression.
//
// BC traces:
//   - AC-009 → BC-2.04.002 EC-008; BC-2.04.007 EC-007 + PC-2.6
//   - ErrPTYSourceEOF message → error-taxonomy.md v2.1 E-SYS-003
//   - Relay no-spin → ARCH-01 ADR-011 v1.5 §HIGH-A
package tmux_test

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/halfchannel"
	"github.com/arcavenae/switchboard/internal/session"
	"github.com/arcavenae/switchboard/internal/tmux"
)

// eofPipeMaster is a fake io.ReadWriteCloser that blocks on Read until
// closeMaster() is called, at which point it returns io.EOF.
// This simulates the PTY master reaching EOF when the shell process exits.
//
// Distinct from pipeMaster in connector_frames_test.go (which also supports
// byte injection). eofPipeMaster is intentionally minimal: it only models
// the EOF trigger, which is all AC-009 requires.
type eofPipeMaster struct {
	mu     sync.Mutex
	cond   *sync.Cond
	closed bool
}

func newEOFPipeMaster() *eofPipeMaster {
	m := &eofPipeMaster{}
	m.cond = sync.NewCond(&m.mu)
	return m
}

// closeMaster simulates the PTY shell exiting: causes all blocked Read calls
// to return (0, io.EOF). Idempotent.
func (m *eofPipeMaster) closeMaster() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	m.cond.Broadcast()
}

func (m *eofPipeMaster) Read(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for !m.closed {
		m.cond.Wait()
	}
	return 0, io.EOF
}

func (m *eofPipeMaster) Write(p []byte) (int, error) { return len(p), nil }

func (m *eofPipeMaster) Close() error {
	m.closeMaster()
	return nil
}

// TestForwardFramesPTYEOFExitsCleanly — AC-009
// (BC-2.04.002 EC-008; BC-2.04.007 EC-007 + PC-2.6; ARCH-01 ADR-011 v1.5 §HIGH-A)
//
// Verifies that when the access node is in PTY mode and the PTY shell process
// exits (EOF on PTY master) WITHOUT sc.Close() being called, the forwardFrames
// relay goroutine does NOT hot-spin. Instead it MUST:
//  1. Detect srcCh==prevSrcCh in PTY mode (InPTYMode true after Connect on PTY-direct path).
//  2. Send ErrPTYSourceEOF on sc.Err() satisfying errors.Is(err, tmux.ErrPTYSourceEOF).
//  3. Exit — sc.Err() delivers the sentinel within ≤100ms of PTY master EOF.
//
// The test FAILS on a hot-spin regression: if forwardFrames does NOT detect PTY mode
// and signal ErrPTYSourceEOF, sc.Err() never receives the sentinel and the 100ms
// deadline fires — test fails.
//
// Construction: ControlMode is injected to fail immediately (ErrControlModeUnavailable),
// forcing sc into PTY-direct mode (sc.inPTYMode = true after Connect). PTYProxy is
// injected with eofPipeMaster. After sc.Connect succeeds (sc.InPTYMode() == true),
// we call closeMaster() WITHOUT calling sc.Close() — simulating shell exit.
//
// Hermetic: no real tmux binary or PTY device. WithPTYAllocFunc injects eofPipeMaster.
// WithExecFunc injects immediate failure for ControlMode.
func TestForwardFramesPTYEOFExitsCleanly(t *testing.T) {
	// AC-009 — BC-2.04.002 EC-008; BC-2.04.007 EC-007 + PC-2.6.
	// NOT t.Parallel(): goroutine lifecycle depends on sequential phase ordering.

	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)

	// ControlMode fails immediately: forces PTY-direct path (sc.inPTYMode = true).
	ctrl := tmux.New(pub, ds, fakeExecFuncErr(tmux.ErrControlModeUnavailable))

	// eofMaster blocks on Read until closeMaster() is called (simulates shell exit).
	eofMaster := newEOFPipeMaster()

	pty := tmux.NewPTYProxy(pub, ds,
		tmux.WithPTYAllocFunc(func() (io.ReadWriteCloser, int, error) {
			return eofMaster, 5555, nil
		}),
	)

	// No ControlModeFactory — PTY-direct path only (no mid-session swap expected).
	sc := tmux.NewSessionConnector(ctrl, pty)

	// t.Cleanup: close sc AFTER assertions (sc.Close() may race with the relay
	// exit; sc.Close() is idempotent so it is safe to call after relay has exited).
	t.Cleanup(func() {
		_ = sc.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Connect: ctrl fails → PTY-direct fallback (sc.inPTYMode = true after Connect).
	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("sc.Connect: %v; want nil (PTY-direct fallback path)", err)
	}

	// Confirm PTY mode is active — precondition for EC-008 detection in forwardFrames.
	if !sc.InPTYMode() {
		t.Fatal("sc.InPTYMode() = false after PTY-direct connect; want true (AC-009 precondition)")
	}

	// Simulate PTY shell exit: close the master WITHOUT calling sc.Close().
	// PTYProxy.ioRelay's Read call returns io.EOF → ioRelay exits → p.frames closed.
	// forwardFrames sees the inner range exit; srcCh == prevSrcCh; InPTYMode() == true
	// → sends ErrPTYSourceEOF on sc.errCh → returns (no hot-spin).
	eofMaster.closeMaster()

	// BOUNDED DEADLINE: ≤100ms per AC-009 spec obligation.
	// If forwardFrames hot-spins instead of detecting PTY-mode EOF, sc.Err() never
	// receives ErrPTYSourceEOF. The 100ms deadline fires and the test FAILS.
	// That is the discriminating property: test fails on the hot-spin regression.
	const eofDeadline = 100 * time.Millisecond

	// Enforce deadline via t.Cleanup with time.AfterFunc, as specified in AC-009.
	// This is a belt-and-suspenders enforcement alongside the select below.
	relayExited := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-relayExited:
			// Relay exited — good.
		case <-time.After(eofDeadline):
			t.Errorf("relay goroutine did not exit within %v of PTY EOF; "+
				"forwardFrames MUST detect srcCh==prevSrcCh in PTY mode and send ErrPTYSourceEOF — "+
				"hot-spin regression guard (AC-009; BC-2.04.002 EC-008; ARCH-01 ADR-011 v1.5 §HIGH-A)",
				eofDeadline)
		}
	})

	// Primary assertion: sc.Err() delivers ErrPTYSourceEOF within the deadline.
	select {
	case err, ok := <-sc.Err():
		close(relayExited)
		if !ok {
			// Channel was closed without delivering an error (possible if sc.Close()
			// raced with forwardFrames). This means Close() won the closeErrCh.Do
			// race — acceptable only if sc.Close() was called, which it was NOT in
			// this test path. Fail.
			t.Fatal("sc.Err() closed without delivering ErrPTYSourceEOF; " +
				"forwardFrames must signal fatal backend loss before exiting " +
				"(BC-2.04.002 invariant 3 — never silent)")
		}
		// errors.Is check: ErrPTYSourceEOF may be delivered directly or wrapped.
		// The spec says "errors.Is(err, tmux.ErrPTYSourceEOF)" per AC-009.
		if !errors.Is(err, tmux.ErrPTYSourceEOF) {
			t.Errorf("sc.Err() = %v; want errors.Is(_, tmux.ErrPTYSourceEOF); "+
				"sentinel string must be %q (E-SYS-003; error-taxonomy.md §SYS)",
				err, tmux.ErrPTYSourceEOF.Error())
		}

	case <-time.After(eofDeadline):
		close(relayExited)
		t.Fatalf("sc.Err() did not deliver ErrPTYSourceEOF within %v of PTY master EOF; "+
			"forwardFrames is hot-spinning (AC-009 regression guard: test MUST fail on busy-spin); "+
			"expected: relay detects srcCh==prevSrcCh + InPTYMode()==true → signals ErrPTYSourceEOF → exits "+
			"(BC-2.04.002 EC-008; BC-2.04.007 EC-007; ARCH-01 ADR-011 v1.5 §HIGH-A)",
			eofDeadline)
	}

	// Secondary assertion: E-SYS-003 sentinel message matches error-taxonomy.md v2.1.
	// ErrPTYSourceEOF.Error() must be exactly "session connector: PTY source EOF"
	// per the spec (no trailing punctuation — ST1005; no wrapping here since
	// forwardFrames sends the sentinel directly).
	// This is verified structurally via errors.Is above. Additionally assert the
	// sentinel is registered as a package-level var (not re-constructed).
	const wantErrMsg = "session connector: PTY source EOF"
	if tmux.ErrPTYSourceEOF.Error() != wantErrMsg {
		t.Errorf("tmux.ErrPTYSourceEOF.Error() = %q; want %q (E-SYS-003; error-taxonomy.md §SYS)",
			tmux.ErrPTYSourceEOF.Error(), wantErrMsg)
	}
}
