// Package tmux — connector_toctou_test.go tests the deterministic TOCTOU
// regression guard for the forwardFrames relay (ADR-011 v1.6 §HIGH-A;
// ARCH-01 T2 Obligation; BC-2.04.002 EC-003).
//
// This file is in package tmux (not tmux_test) because the test must set
// sc.swapBarrier, which is an unexported field on SessionConnector.
//
// The TOCTOU race (fixed in commit 5ffac2d) occurred when forwardFrames read
// sc.active and sc.inPTYMode under two separate sc.mu acquisitions:
//
//  1. Read sc.active → ctrl (ctrl.frames closed)
//  2. A ctrl→PTY swap lands between 1 and 2
//  3. Read sc.inPTYMode → true (from the swap)
//
// The result: srcCh == prevSrcCh AND inPTY == true → relay incorrectly
// signals ErrPTYSourceEOF and exits, even though PTY frames are available.
//
// The v1.6 fix (activeSourceSnapshot) reads {source, srcCh, inPTYMode} under
// a SINGLE sc.mu hold, so the snapshot is always self-consistent.
//
// The swapBarrier seam enables a deterministic interleaving test without a
// data race:
//
//	sc.swapBarrier is set BEFORE sc.Connect() — before the relay goroutine
//	starts — satisfying the Go memory model §"Goroutine creation" guarantee
//	(the write happens-before the goroutine start).
//
// Sequence exercised:
//
//  1. sc.swapBarrier = make(chan struct{}) set before sc.Connect().
//  2. sc.Connect() starts forwardFrames; relay immediately blocks at the
//     swapBarrier (snapshot taken under sc.mu, barrier receive blocks before
//     returning the snapshot to forwardFrames).
//  3. While relay is blocked: close ctrlPW (EOF) → ErrControlModeDropped →
//     watchAndFallback → pty.Connect() → sc.active = sc.pty; sc.inPTYMode =
//     true (swap lands under sc.mu).
//  4. Release swapBarrier (close it): relay resumes with stale snapshot
//     {ctrl.frames, inPTY=false} (the atomic snapshot captured inPTY before
//     the swap; the swap is invisible to this snapshot).
//  5. Relay: prevSrcCh == nil → srcCh != prevSrcCh → sets prevSrcCh =
//     ctrl.frames → for f := range ctrl.frames (already closed) exits.
//  6. Relay loops: fresh snapshot {pty.frames, inPTY=true} →
//     srcCh != prevSrcCh (pty.frames is a NEW channel, ≠ ctrl.frames) →
//     relay starts ranging over PTY source.
//  7. PTY frame arrives on sc.Frames() — no ErrPTYSourceEOF misfire.
//
// Discriminating: a broken implementation that reads sc.inPTYMode under a
// SEPARATE lock acquisition after the barrier releases would observe
// inPTY=true (the swap has already landed) while srcCh == ctrl.frames.
// The condition srcCh == prevSrcCh AND inPTY == true would fire ErrPTYSourceEOF,
// which this test would detect at the sc.Err() case.
package tmux

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/halfchannel"
	"github.com/arcavenae/switchboard/internal/session"
)

// toctouNopCloser wraps an io.Reader with a no-op Close for injection as the
// fake tmux subprocess stdout (avoids forking a real tmux process).
type toctouNopCloser struct{ io.Reader }

func (toctouNopCloser) Close() error { return nil }

// toctouNopWriteCloser is a no-op io.WriteCloser for the fake tmux stdin.
// ControlMode.Connect writes list-sessions to stdin; the fake discards it.
type toctouNopWriteCloser struct{}

func (toctouNopWriteCloser) Write(p []byte) (int, error) { return len(p), nil }
func (toctouNopWriteCloser) Close() error                { return nil }

// toctouClosedNilChan returns a pre-closed nil-classification channel for
// test fakes that do not exercise the classification path.
func toctouClosedNilChan() <-chan error {
	ch := make(chan error, 1)
	close(ch)
	return ch
}

// controlExecWithReader returns a WithExecFunc option that yields the given
// io.Reader as the ctrl subprocess stdout. Tests control EOF timing by closing
// the underlying io.Pipe write end.
func controlExecWithReader(r io.Reader) Option {
	return WithExecFunc(func(_ context.Context) (io.WriteCloser, io.ReadCloser, <-chan error, error) {
		return toctouNopWriteCloser{}, toctouNopCloser{r}, toctouClosedNilChan(), nil
	})
}

// toctouPipeMaster is a fake io.ReadWriteCloser for injecting PTY bytes.
// Read blocks until bytes are injected via inject() or Close is called.
type toctouPipeMaster struct {
	mu     sync.Mutex
	cond   *sync.Cond
	buf    []byte
	closed bool
}

func newToctouPipeMaster() *toctouPipeMaster {
	m := &toctouPipeMaster{}
	m.cond = sync.NewCond(&m.mu)
	return m
}

func (m *toctouPipeMaster) inject(p []byte) {
	m.mu.Lock()
	m.buf = append(m.buf, p...)
	m.cond.Broadcast()
	m.mu.Unlock()
}

func (m *toctouPipeMaster) Read(p []byte) (int, error) {
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

func (m *toctouPipeMaster) Write(p []byte) (int, error) { return len(p), nil }

func (m *toctouPipeMaster) Close() error {
	m.mu.Lock()
	m.closed = true
	m.cond.Broadcast()
	m.mu.Unlock()
	return nil
}

// TestForwardFramesTOCTOURegressionDeterministic — T2 TOCTOU regression guard
// (ADR-011 v1.6; ARCH-01 §"Test obligations" T2 Option B; BC-2.04.002 EC-003)
//
// Exercises the exact race window that commit 5ffac2d closed. The swapBarrier
// forces the relay to act on a snapshot taken before a ctrl→PTY swap lands,
// confirming that the atomic snapshot (not the pre-v1.6 two-lock read) prevents
// misclassification as ErrPTYSourceEOF.
//
// Race-safety: sc.swapBarrier is written once, before sc.Connect() starts the
// relay goroutine. Subsequent reads by the relay happen-after the goroutine
// creation (Go memory model §"Goroutine creation"). The barrier channel is
// released by close() from the test goroutine; channel close is safe for
// concurrent receive.
func TestForwardFramesTOCTOURegressionDeterministic(t *testing.T) {
	// NOT t.Parallel(): goroutine lifecycle depends on sequential phase completion.

	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)

	// ctrlPR / ctrlPW: test-controlled ctrl subprocess stdout pipe.
	// Writing %begin/%end lets ctrl.Connect succeed; closing ctrlPW triggers
	// EOF → ErrControlModeDropped → watchAndFallback → PTY fallback.
	ctrlPR, ctrlPW := io.Pipe()
	t.Cleanup(func() { _ = ctrlPW.Close() })
	t.Cleanup(func() { _ = ctrlPR.Close() })

	ctrl := New(pub, ds, controlExecWithReader(ctrlPR))

	// PTY master that allows test-controlled byte injection post-swap.
	ptyMaster := newToctouPipeMaster()
	pty := NewPTYProxy(pub, ds,
		WithPTYAllocFunc(func() (io.ReadWriteCloser, int, error) {
			return ptyMaster, 9999, nil
		}),
	)

	// No ControlModeFactory → immediate PTY fallback on ErrControlModeDropped.
	sc := NewSessionConnector(ctrl, pty)
	t.Cleanup(func() {
		_ = ptyMaster.Close()
		_ = sc.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Step 1: install the swap barrier BEFORE sc.Connect() — this is the only
	// race-safe write point because the relay goroutine has not yet started.
	// The Go memory model guarantees the relay sees this write when it starts
	// (goroutine creation happens-after all prior writes in the creating goroutine).
	barrier := make(chan struct{})
	sc.swapBarrier = barrier

	// Step 2: write a valid %begin/%end block concurrently so ctrl.Connect
	// succeeds (the pipe reader blocks until data arrives).
	writeErrCh := make(chan error, 1)
	go func() {
		_, err := io.WriteString(ctrlPW, "%begin 8000000000 0 1\n%end 8000000000 0 1\n")
		writeErrCh <- err
	}()

	// Step 3: Connect — ctrl.Connect reads the header and succeeds, then
	// startForwardFrames launches the relay goroutine. The relay immediately
	// calls activeSourceSnapshot, takes the {ctrl.frames, inPTY=false} snapshot
	// under sc.mu, releases sc.mu, then blocks at <-sc.swapBarrier.
	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("sc.Connect: %v", err)
	}
	if err := <-writeErrCh; err != nil {
		t.Fatalf("pipe header write: %v", err)
	}

	// Verify ctrl mode is active (not PTY yet).
	if sc.InPTYMode() {
		t.Fatal("sc.InPTYMode() = true immediately after Connect; want false (ctrl path)")
	}

	// Step 4: trigger ctrl EOF — ErrControlModeDropped → watchAndFallback →
	// pty.Connect() → sc.active = sc.pty; sc.inPTYMode = true (under sc.mu).
	// The relay is blocked at the barrier and cannot observe this swap until
	// the barrier is released.
	_ = ctrlPW.Close()

	// Step 5: wait for the swap to complete (InPTYMode == true).
	// watchAndFallback sets inPTYMode=true under sc.mu before the relay
	// resumes from the barrier. Bounded: 3s.
	swapDeadline := time.After(3 * time.Second)
	for !sc.InPTYMode() {
		select {
		case <-swapDeadline:
			t.Fatal("InPTYMode() never became true within 3s after ctrl EOF; " +
				"watchAndFallback must activate PTY on ErrControlModeDropped (EC-003)")
		default:
		}
	}

	// Step 6: release the barrier. The relay resumes with its stale atomic
	// snapshot {ctrl.frames, inPTY=false}. Because the snapshot is self-
	// consistent (both fields from the same sc.mu hold), inPTY=false correctly
	// reflects the state AT THE TIME OF THE SNAPSHOT — before the swap landed.
	//
	// The relay path: prevSrcCh==nil → srcCh != prevSrcCh → set prevSrcCh =
	// ctrl.frames → range ctrl.frames (already closed, exits immediately) →
	// loop → fresh snapshot {pty.frames, inPTY=true} → srcCh != prevSrcCh
	// (pty.frames is a NEW object, guaranteed ≠ ctrl.frames) → range PTY.
	//
	// The broken (pre-v1.6) path: relay re-reads inPTYMode under a second lock
	// after barrier — inPTY=true (swap landed) while srcCh == ctrl.frames ==
	// prevSrcCh → fires ErrPTYSourceEOF. This test detects that regression.
	close(barrier)

	// Step 7: inject a PTY frame after the barrier is released.
	ptyMaster.inject([]byte("toctou-t2-frame"))

	// Step 8: assert the PTY frame arrives on sc.Frames() within a bounded
	// deadline. This confirms:
	//   - sc.frames was NOT prematurely closed (no ErrPTYSourceEOF misfire)
	//   - forwardFrames re-subscribed to the PTY source after the barrier
	//   - the ADR-011 re-subscribe invariant holds across the deterministic race
	const frameDeadline = 2 * time.Second
	select {
	case f, ok := <-sc.Frames():
		if !ok {
			t.Fatal("sc.Frames() channel closed prematurely after TOCTOU interleaving; " +
				"ErrPTYSourceEOF was incorrectly fired (TOCTOU regression detected)")
		}
		_ = f // delivery — not content — is the assertion (ADR-011)
	case err, ok := <-sc.Err():
		if ok && errors.Is(err, ErrPTYSourceEOF) {
			t.Fatalf("sc.Err() delivered ErrPTYSourceEOF after TOCTOU interleaving; "+
				"relay misclassified stale ctrl snapshot as PTY terminal EOF "+
				"(TOCTOU regression — atomic snapshot fix in commit 5ffac2d lost): %v", err)
		}
		t.Fatalf("sc.Err() delivered unexpected error after TOCTOU interleaving: %v (ok=%v)", err, ok)
	case <-time.After(frameDeadline):
		t.Fatalf("no PTY frame on sc.Frames() within %v after barrier release; "+
			"forwardFrames did not re-subscribe to PTY source after TOCTOU interleaving "+
			"(ADR-011 re-subscribe invariant)", frameDeadline)
	}
}

// TestForwardFramesTOCTOUCount50 — T2 stress variant (ARCH-01 v1.6 §T2 Option A)
//
// Runs the EC-002 failover scenario 50 times to catch probabilistic TOCTOU
// regressions that the deterministic test may not reach under a particular
// scheduler. Any single failure indicates a TOCTOU regression in the
// activeSourceSnapshot logic (srcCh/inPTYMode snapshot consistency).
//
// Each iteration is independent: fresh sc, ctrl, pty, and pipe master.
func TestForwardFramesTOCTOUCount50(t *testing.T) {
	// NOT t.Parallel() at outer level — 50 goroutine-heavy subtests in parallel
	// would saturate the scheduler and introduce false timeouts.
	for i := range 50 {
		t.Run("", func(t *testing.T) {
			keys := admission.NewAdmittedKeySet()
			pub := session.NewPublisher(keys)
			ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)

			// Ctrl stream: one valid block then immediate EOF → triggers failover.
			ctrlStream := strings.NewReader(
				"%begin 7000000000 0 1\n%end 7000000000 0 1\n",
			)
			ctrl := New(pub, ds, controlExecWithReader(ctrlStream))

			ptypipe := newToctouPipeMaster()
			pty := NewPTYProxy(pub, ds,
				WithPTYAllocFunc(func() (io.ReadWriteCloser, int, error) {
					return ptypipe, 5001 + i, nil
				}),
			)

			sc := NewSessionConnector(ctrl, pty)
			t.Cleanup(func() {
				_ = ptypipe.Close()
				_ = sc.Close()
			})

			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			ch := sc.Frames()

			if err := sc.Connect(ctx); err != nil {
				t.Fatalf("iter %d: sc.Connect: %v", i, err)
			}

			// Wait for PTY fallback (ctrl EOF → watchAndFallback → inPTYMode=true).
			deadline := time.After(3 * time.Second)
			for !sc.InPTYMode() {
				select {
				case <-deadline:
					t.Fatalf("iter %d: InPTYMode() never true within 3s", i)
				default:
				}
			}

			// Drain any ctrl-phase frames so channel is ready for PTY assertion.
			for {
				select {
				case <-ch:
				default:
					goto drained
				}
			}
		drained:

			// Inject a PTY frame and assert it arrives on the SAME channel.
			// A TOCTOU regression would fire ErrPTYSourceEOF or close sc.frames
			// instead, causing the ok==false or sc.Err() branches to fire.
			ptypipe.inject([]byte("toctou-count50-frame"))

			select {
			case f, ok := <-ch:
				if !ok {
					t.Fatalf("iter %d: sc.Frames() closed prematurely — "+
						"ErrPTYSourceEOF misfire (TOCTOU regression)", i)
				}
				_ = f
			case err, ok := <-sc.Err():
				if ok && errors.Is(err, ErrPTYSourceEOF) {
					t.Fatalf("iter %d: ErrPTYSourceEOF on sc.Err() — "+
						"TOCTOU regression in activeSourceSnapshot", i)
				}
				t.Fatalf("iter %d: sc.Err() delivered unexpected error: %v", i, err)
			case <-time.After(2 * time.Second):
				t.Fatalf("iter %d: no PTY frame within 2s after swap "+
					"(ADR-011 re-subscribe invariant)", i)
			}
		})
	}
}
