// Package tmux — connector_toctou_misclass_test.go
//
// Deterministic regression test for the TOCTOU misclassification branch in
// forwardFrames (ARCH-01 v1.6 §ADR-011 Obligation T2; BC-2.04.002 EC-003,
// EC-002).
//
// # Background
//
// The TOCTOU bug (fixed in commit 5ffac2d) occurred when forwardFrames read
// {sc.active, srcCh} and {sc.inPTYMode} under two SEPARATE sc.mu acquisitions.
// A ctrl→PTY swap straddling those two reads produced a self-inconsistent
// "snapshot": srcCh still pointed to the already-closed ctrl.frames channel
// (stale, from lock 1), while inPTY was already true (fresh, from lock 2).
// The relay then hit the misclassification branch:
//
//	srcCh == prevSrcCh (both = closed ctrl.frames) AND inPTY == true
//	→ treated as terminal PTY-source EOF → ErrPTYSourceEOF fired → relay exited
//
// even though the PTY was alive and delivering frames (~20% failure rate in
// the 50-iteration probabilistic test, TestForwardFramesTOCTOUCount50).
//
// The v1.6 fix moved all three reads ({active, srcCh, inPTYMode}) under a
// SINGLE sc.mu hold in activeSourceSnapshot(), making the snapshot
// self-consistent and rendering the straddle impossible.
//
// # What the existing deterministic test does NOT cover
//
// TestForwardFramesTOCTOURegressionDeterministic (connector_toctou_test.go:178)
// gates the FIRST relay iteration via swapBarrier. On the first iteration
// prevSrcCh is nil, so srcCh != prevSrcCh always holds — the relay takes the
// re-subscribe path, never reaching the misclassification branch. That test
// validates snapshot consistency but does not exercise the
// srcCh == prevSrcCh && inPTY == true branch documented in forwardFrames.
//
// # This test
//
// TestForwardFramesTOCTOUMisclassificationBranchDeterministic exercises the
// SECOND relay iteration (prevSrcCh == ctrl.frames) and enforces:
//
//  1. prevSrcCh is SET to ctrl.frames (first iteration complete).
//  2. The relay's second snapshot lock for sc.active is taken WHILE sc.active=ctrl
//     (ptyAlloc is blocking → swap cannot complete until the test signals).
//  3. Only AFTER the relay has captured sc.active=ctrl does the test trigger
//     the swap (via ptyAllocReady) and wait for inPTYMode=true.
//  4. The test then releases swapBarrier.
//
// With the FIXED (atomic) snapshot: step 4 returns {ctrl.frames, inPTY=false}
// (the snapshot was taken before the swap, atomically consistent). The relay
// yields, loops, and takes a fresh snapshot {pty.frames, true} → re-subscribes
// → delivers the PTY frame. PASS.
//
// With the BUGGY two-lock form (scratch edit — see Red-gate section):
// swapBarrier is positioned BETWEEN lock1 (reading sc.active) and lock2
// (reading sc.inPTYMode). lock2 runs after the barrier is released (after
// InPTYMode=true is confirmed), so lock2 reads inPTYMode=true. This produces
// the misclassifying {ctrl.frames, inPTY=true} pair: srcCh==prevSrcCh AND
// inPTY → ErrPTYSourceEOF. Test FAILS deterministically on every run.
//
// # Barrier strategy
//
// Two test-only seam channels are used:
//
//   - swapBarrier (unbuffered): blocks the relay in activeSourceSnapshot after
//     the sc.active lock. The relay blocks here while the test orchestrates the
//     swap and confirms InPTYMode.
//
//   - swapBarrier2 (buffered N): non-blocking send fired by activeSourceSnapshot
//     AFTER sc.active has been read under sc.mu, BEFORE swapBarrier blocks.
//     The test receives this signal to know sc.active was captured (as ctrl,
//     because ptyAlloc is blocking the swap). The test then safely sends
//     ptyAllocReady and waits InPTYMode before releasing swapBarrier.
//
// The ptyAllocFunc blocks until the test sends ptyAllocReady. This guarantees
// sc.active=ctrl when the relay's lock runs, because the swap (sc.active=pty)
// cannot complete until ptyAlloc returns.
//
// # Phase overview
//
//	Phase 0 — setup:   barrier, barrier2, ptyAllocReady channels installed
//	Phase 1 — first:   swapBarrier2 signal 1 received → first barrier released
//	                   relay acts on {ctrl, false} → prevSrcCh=ctrl.frames
//	Phase 2 — second:  swapBarrier2 signal 2 received → sc.active=ctrl confirmed
//	                   ptyAllocReady sent → swap fires → InPTYMode=true
//	                   barrier released → (fixed) {ctrl,false}→yield; (buggy) {ctrl,true}→FAIL
//	Phase 3 — deliver: barrier closed → relay finds {pty,true} → frame delivered
//
// # Red-gate validation (buggy two-lock form)
//
// To verify this is a genuine discriminator, temporarily scratch-edit
// connector_frames.go to the pre-5ffac2d two-lock form, positioning the
// swapBarrier receive BETWEEN lock1 and lock2, and firing swapBarrier2 after
// lock1:
//
//	sc.mu.Lock()
//	active := sc.active; /* build src, srcCh */ ; sc.mu.Unlock()
//	if sc.swapBarrier2 != nil { select { case sc.swapBarrier2 <- struct{}{}: default: } }
//	if sc.swapBarrier  != nil { <-sc.swapBarrier }    // ← barrier BETWEEN locks
//	sc.mu.Lock(); inPTY := sc.inPTYMode; sc.mu.Unlock()
//	return src, srcCh, inPTY
//
// With this placement: phase 2 receives signal 2 (lock1 done, sc.active=ctrl),
// triggers the swap, waits InPTYMode=true, releases barrier → lock2 reads
// inPTYMode=true → {ctrl.frames, true} → ErrPTYSourceEOF → test FAILS
// deterministically every run.
//
// Revert the scratch edit before committing — the net diff must contain only
// this test file and the pty_fallback.go swapBarrier2 field addition.
//
// # Race-safety
//
// swapBarrier and swapBarrier2 are written once before sc.Connect() starts the
// relay goroutine. Go memory model §"Goroutine creation" guarantees the relay
// sees both writes.
//
// # Relationship to other TOCTOU tests
//
//   - TestForwardFramesTOCTOURegressionDeterministic: first snapshot,
//     prevSrcCh==nil path, snapshot-consistency.
//   - TestForwardFramesTOCTOUMisclassificationBranchDeterministic (this test):
//     second snapshot, prevSrcCh==ctrl.frames path, misclassification branch.
//   - TestForwardFramesTOCTOUCount50: probabilistic 50-iteration stress.
package tmux

import (
	"context"
	"errors"
	"io"
	"runtime"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/halfchannel"
	"github.com/arcavenae/switchboard/internal/session"
)

// TestForwardFramesTOCTOUMisclassificationBranchDeterministic is the
// deterministic regression detector for the srcCh==prevSrcCh && inPTY==true
// misclassification branch in forwardFrames.
//
// ARCH-01 v1.6 §ADR-011 T2; BC-2.04.002 EC-003, EC-002.
//
// Complements TestForwardFramesTOCTOURegressionDeterministic (first-iteration
// snapshot-consistency; prevSrcCh==nil) and TestForwardFramesTOCTOUCount50
// (probabilistic 50-iteration stress). This test specifically exercises the
// second-iteration path (prevSrcCh==ctrl.frames) using the two-seam barrier
// protocol (swapBarrier + swapBarrier2) and ptyAlloc blocking to guarantee
// that sc.active=ctrl is captured before the swap fires.
func TestForwardFramesTOCTOUMisclassificationBranchDeterministic(t *testing.T) {
	// NOT t.Parallel(): goroutine lifecycle depends on sequential phase completion.

	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)

	// ctrlPR/ctrlPW: test-controlled ctrl subprocess stdout pipe.
	// Writing %begin/%end lets ctrl.Connect succeed.
	// Closing ctrlPW triggers EOF → ctrl.frames closes → relay's first range
	// exits → relay loops for second snapshot.
	ctrlPR, ctrlPW := io.Pipe()
	t.Cleanup(func() { _ = ctrlPW.Close() })
	t.Cleanup(func() { _ = ctrlPR.Close() })

	ctrl := New(pub, ds, controlExecWithReader(ctrlPR))

	// ptyAllocReady gates pty.Connect → swap. The allocFunc blocks until the
	// test sends on ptyAllocReady, guaranteeing sc.active=ctrl when the relay's
	// lock runs (because the swap cannot land until ptyAlloc returns).
	ptyAllocReady := make(chan struct{})
	ptyMaster := newToctouPipeMaster()
	pty := NewPTYProxy(pub, ds,
		WithPTYAllocFunc(func() (io.ReadWriteCloser, int, error) {
			<-ptyAllocReady // block until test signals
			return ptyMaster, 8888, nil
		}),
	)

	sc := NewSessionConnector(ctrl, pty)
	t.Cleanup(func() {
		_ = ptyMaster.Close()
		_ = sc.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Install barriers before sc.Connect() so the relay goroutine sees them at
	// startup (Go memory model §"Goroutine creation").
	//
	// barrier (swapBarrier, unbuffered): blocks the relay in activeSourceSnapshot
	// after sc.active has been read under sc.mu. The relay blocks here; the test
	// controls each release.
	//
	// barrier2 (swapBarrier2, buffered): non-blocking signal from relay →
	// activeSourceSnapshot fires after sc.active is read. The test receives this
	// to know sc.active was captured (as ctrl, while ptyAlloc is blocking) before
	// triggering the swap.
	barrier := make(chan struct{})
	barrier2 := make(chan struct{}, 16) // buffered to absorb spurious yield iterations
	sc.swapBarrier = barrier
	sc.swapBarrier2 = barrier2

	// Write a valid %begin/%end block so ctrl.Connect succeeds.
	writeErrCh := make(chan error, 1)
	go func() {
		_, err := io.WriteString(ctrlPW, "%begin 9000000000 0 1\n%end 9000000000 0 1\n")
		writeErrCh <- err
	}()

	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("sc.Connect: %v", err)
	}
	if err := <-writeErrCh; err != nil {
		t.Fatalf("pipe header write: %v", err)
	}

	if sc.InPTYMode() {
		t.Fatal("sc.InPTYMode() = true immediately after Connect; want false (ctrl path)")
	}

	// Phase 1 — first iteration: establish prevSrcCh = ctrl.frames.
	//
	// Step 1a: trigger ctrl EOF. ctrl.frames will close and watchAndFallback will
	// call pty.Connect → ptyAllocFunc → BLOCKS on ptyAllocReady. The swap cannot
	// complete until ptyAllocReady is sent, so sc.active remains ctrl.
	_ = ctrlPW.Close()

	// Step 1b: wait for barrier2 signal 1 — relay's first snapshot has captured
	// sc.active (as ctrl) under sc.mu and is about to block on swapBarrier.
	const signalTimeout = 5 * time.Second
	select {
	case <-barrier2:
		// First snapshot sc.active read complete (sc.active=ctrl, ptyAlloc blocking).
	case <-time.After(signalTimeout):
		t.Fatalf("no barrier2 signal within %v for first snapshot; "+
			"relay goroutine did not start or activeSourceSnapshot did not fire", signalTimeout)
	}

	// Step 1c: release first snapshot. The relay was blocked at swapBarrier after
	// reading {ctrl.frames, inPTY=false} (ptyAlloc blocking, so inPTYMode=false).
	// After this send the relay acts on the snapshot: prevSrcCh=nil →
	// srcCh(ctrl.frames) != prevSrcCh → sets prevSrcCh=ctrl.frames → ranges
	// ctrl.frames. ctrl.frames will close (triggered by ctrlPW.Close above), so
	// the range exits and the relay loops.
	const sendTimeout = 3 * time.Second
	select {
	case barrier <- struct{}{}:
		// First snapshot released. relay will range ctrl.frames, exit, loop.
	case <-time.After(sendTimeout):
		t.Fatalf("relay did not reach first swapBarrier within %v; "+
			"forwardFrames not running or swapBarrier seam not installed", sendTimeout)
	}

	// Phase 2 — second iteration: force the misclassification-branch conditions.
	//
	// The relay has ranged ctrl.frames (now closed) and loops. The second call to
	// activeSourceSnapshot will read sc.active=ctrl (swap still blocked) and fire
	// barrier2 signal 2 before blocking on swapBarrier again.
	//
	// Step 2a: wait for barrier2 signal 2 — relay's second snapshot has captured
	// sc.active=ctrl under sc.mu. prevSrcCh is now ctrl.frames (set in phase 1).
	// This is the critical moment: sc.active=ctrl AND prevSrcCh=ctrl.frames.
	select {
	case <-barrier2:
		// Second snapshot sc.active read complete (sc.active=ctrl, ptyAlloc blocking).
		// Relay is now blocked at swapBarrier:
		//   fixed form: after single lock — full snapshot {ctrl.frames,false} determined
		//   buggy form: after lock1 — lock2 (reading inPTYMode) has NOT yet run
	case <-time.After(signalTimeout):
		t.Fatalf("no barrier2 signal within %v for second snapshot; "+
			"relay did not loop after first iteration or ctrl.frames did not close", signalTimeout)
	}

	// Step 2b: release ptyAlloc → swap fires.
	// Now that sc.active=ctrl has been captured by the relay's lock, it is safe
	// to complete the swap. ptyAlloc returns → watchAndFallback sets
	// sc.active=pty, sc.inPTYMode=true under sc.mu.
	ptyAllocReady <- struct{}{}

	// Step 2c: wait for the ctrl→PTY swap to complete.
	// watchAndFallback sets inPTYMode=true under sc.mu. Only after this is
	// confirmed does releasing swapBarrier have the desired effect:
	//   fixed form: snapshot is already {ctrl,false} — releasing now has no
	//     additional effect on the snapshot (it was determined before the swap).
	//   buggy form: lock2 runs AFTER this, so it reads inPTYMode=true → {ctrl,true}
	//     → misclassification branch → ErrPTYSourceEOF.
	const swapTimeout = 5 * time.Second
	swapDone := time.After(swapTimeout)
	for !sc.InPTYMode() {
		select {
		case <-swapDone:
			t.Fatal("InPTYMode() never became true within 5s after ptyAllocReady; " +
				"watchAndFallback must activate PTY on ErrControlModeDropped (EC-003)")
		default:
			runtime.Gosched()
		}
	}

	// Step 2d: release swapBarrier for the second snapshot.
	//   fixed form: relay returns {ctrl.frames, inPTY=false} (pre-swap snapshot).
	//     srcCh==prevSrcCh && !inPTY → yield (Gosched; continue).
	//   buggy form: lock2 runs now and reads inPTYMode=true.
	//     Returns {ctrl.frames, inPTY=true}: srcCh==prevSrcCh AND inPTY →
	//     ErrPTYSourceEOF fired → relay exits. Test detects failure.
	select {
	case barrier <- struct{}{}:
		// Second snapshot released.
	case <-time.After(sendTimeout):
		t.Fatalf("relay did not reach second swapBarrier within %v", sendTimeout)
	}

	// Phase 3 — deliver: close barrier so the relay can advance past any yield
	// iterations and find the {pty.frames, inPTY=true} snapshot.
	// (fixed form only: buggy form has already exited via ErrPTYSourceEOF.)
	close(barrier)

	// Inject a PTY frame so the relay can deliver it after re-subscribing.
	ptyMaster.inject([]byte("misclass-branch-frame"))

	// Assert: PTY frame arrives on sc.Frames() without ErrPTYSourceEOF.
	//
	// PASS (fixed code): relay yields on {ctrl.frames,false}, loops, takes
	// {pty.frames,true} → srcCh≠prevSrcCh → re-subscribes → frame delivered.
	//
	// FAIL (buggy two-lock scratch): relay fires ErrPTYSourceEOF →
	// sc.Err() delivers it → sc.frames closes → test detects misclassification.
	const frameDeadline = 3 * time.Second
	select {
	case f, ok := <-sc.Frames():
		if !ok {
			t.Fatal("sc.Frames() closed prematurely after second-iteration TOCTOU " +
				"interleaving — ErrPTYSourceEOF was incorrectly fired " +
				"(misclassification-branch regression: srcCh==prevSrcCh && inPTY==true " +
				"reached in forwardFrames; atomic snapshot fix lost)")
		}
		_ = f // frame delivery — not content — is the assertion (ADR-011)
	case err, ok := <-sc.Err():
		if ok && errors.Is(err, ErrPTYSourceEOF) {
			t.Fatalf("ErrPTYSourceEOF on sc.Err() after second-iteration TOCTOU "+
				"interleaving: relay entered srcCh==prevSrcCh && inPTY==true branch "+
				"and misclassified the ctrl-channel EOF as terminal PTY-source EOF "+
				"(TOCTOU misclassification regression — atomic snapshot fix in commit "+
				"5ffac2d lost): %v", err)
		}
		t.Fatalf("unexpected error on sc.Err() after second-iteration TOCTOU "+
			"interleaving: %v (ok=%v)", err, ok)
	case <-time.After(frameDeadline):
		t.Fatalf("no PTY frame on sc.Frames() within %v after barrier release; "+
			"forwardFrames did not re-subscribe to PTY source after second-iteration "+
			"TOCTOU interleaving (ADR-011 re-subscribe invariant violated)", frameDeadline)
	}
}
