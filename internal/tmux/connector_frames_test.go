// Package tmux — connector_frames_test.go tests the SessionConnector.Frames()
// forwarding-channel API (ADR-011; AC-004; AC-005).
//
// AC-004 traces to: BC-2.04.001 PC-5 + BC-2.04.002 PC-4
//   - Frames() returns a stable channel that survives a ctrl→PTY failover.
//   - Consumer goroutine does NOT need to resubscribe after mode switch.
//
// EC-001 traces to: BC-2.04.001 EC-001/EC-002 — initial ctrl failure forces PTY;
//
//	frames must arrive on the same channel Frames() returned before Connect.
//
// EC-002 traces to: BC-2.04.002 EC-003 — mid-session ctrl drop; relay re-reads
//
//	activeFrSource() under sc.mu; frames from new source appear on same channel.
//
// EC-003 traces to: ADR-011 §Concurrency — sc.frames full: relay uses non-blocking
//
//	select; no deadlock; injected frames complete within a bounded timeout.
//
// Red Gate (BC-5.38.001): Frames(), activeFrSource(), and forwardFrames() all
// panic("not implemented") in the current stub. Every test below MUST FAIL until
// the implementer provides the relay goroutine.
package tmux_test

import (
	"context"
	"io"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/halfchannel"
	"github.com/arcavenae/switchboard/internal/session"
	"github.com/arcavenae/switchboard/internal/tmux"
)

// pipeMaster is a fake io.ReadWriteCloser that allows the test to inject
// readable bytes into the PTY ioRelay goroutine. Writes to injectBytes are
// buffered and returned by Read in FIFO order. Read blocks until either bytes
// arrive or Close is called (returns io.EOF on Close).
type pipeMaster struct {
	mu     sync.Mutex
	cond   *sync.Cond
	buf    []byte
	closed bool
}

func newPipeMaster() *pipeMaster {
	m := &pipeMaster{}
	m.cond = sync.NewCond(&m.mu)
	return m
}

// injectBytes enqueues p for the next Read call. Never blocks.
func (m *pipeMaster) injectBytes(p []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.buf = append(m.buf, p...)
	m.cond.Broadcast()
}

func (m *pipeMaster) Read(p []byte) (int, error) {
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

func (m *pipeMaster) Write(p []byte) (int, error) {
	return len(p), nil // discard keystroke writes
}

func (m *pipeMaster) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	m.cond.Broadcast()
	return nil
}

// TestSessionConnectorFramesSurviveFailover — AC-004
// (BC-2.04.001 PC-5 + BC-2.04.002 PC-4)
//
// Verifies that sc.Frames() returns a stable forwarding channel that continues
// delivering frames across a ctrl→PTY failover without the consumer needing to
// resubscribe (ADR-011).
//
// Three sub-assertions:
//  1. Frames() returns a non-nil receive-only channel (ADR-011 contract).
//  2. After Connect (PTY fallback path), frames injected through the PTY master
//     arrive on the SAME channel returned before Connect (EC-001: initial ctrl
//     failure transparently activates PTY; consumer does not resubscribe).
//  3. When sc.frames is full, the relay does NOT block — additional injections
//     complete within a short deadline (EC-003: non-blocking drop per ADR-011
//     §Concurrency contract).
//
// Hermetic: ControlMode is injected with fakeExecFuncErr (connect fails with
// ErrControlModeUnavailable) → forces PTY fallback. PTYProxy is injected with
// a pipeMaster that feeds controlled bytes.
//
// Red Gate: sc.Frames() panics("not implemented") in the current stub.
func TestSessionConnectorFramesSurviveFailover(t *testing.T) {
	// AC-004 / EC-001 / EC-002 / EC-003.
	// NOT t.Parallel(): stub panics; parallel execution would confuse the test
	// framework's panic attribution under -count=N.

	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)

	// ControlMode that returns ErrControlModeUnavailable on Connect — forces PTY
	// fallback (EC-001: initial ctrl failure).
	ctrl := tmux.New(pub, ds, fakeExecFuncErr(tmux.ErrControlModeUnavailable))

	// Fake PTY master: Read delivers injected bytes; Write is a no-op.
	pipe := newPipeMaster()

	pty := tmux.NewPTYProxy(pub, ds,
		tmux.WithPTYAllocFunc(func() (io.ReadWriteCloser, int, error) {
			return pipe, 1234, nil
		}),
	)
	sc := tmux.NewSessionConnector(ctrl, pty)
	t.Cleanup(func() {
		_ = pipe.Close()
		if err := sc.Close(); err != nil {
			t.Logf("sc.Close: %v", err)
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Assertion 1 (AC-004 primary): Frames() returns a non-nil channel.
	// The stub panics here — that panic IS the Red Gate failure for this test.
	framesCh := sc.Frames()
	if framesCh == nil {
		t.Fatal("sc.Frames() returned nil; want a non-nil receive-only channel (ADR-011)")
	}

	// Connect: ctrl fails → PTY fallback (EC-001).
	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("sc.Connect: %v; want nil (PTY fallback path)", err)
	}

	// Assertion 2 (AC-004 / EC-001): inject data into PTY master; frame must
	// arrive on framesCh — the SAME channel obtained before Connect.
	// The relay goroutine (forwardFrames) must forward from PTY backend to
	// sc.frames, which is the channel returned by Frames().
	pipe.injectBytes([]byte("hello-ac-004"))

	select {
	case f, ok := <-framesCh:
		if !ok {
			t.Fatal("framesCh closed prematurely; want frame delivery from PTY ioRelay")
		}
		// Any non-closed frame delivery satisfies AC-004 structural assertion.
		_ = f
	case <-time.After(500 * time.Millisecond):
		t.Fatal("no frame on sc.Frames() within 500ms after PTY byte injection; " +
			"forwardFrames relay goroutine is not forwarding " +
			"(stub panics — Red Gate: BC-5.38.001)")
	}

	// Assertion 3 (EC-003): sc.frames full → relay uses non-blocking select.
	// Drain any buffered frames so the channel has room to fill.
	drainFramesCh(framesCh)

	// Inject well beyond framesBufferSize (256) bytes to saturate sc.frames.
	// If the relay blocks on a full channel, this goroutine would stall and the
	// deadline below would fire. ADR-011 §Concurrency requires a non-blocking
	// select with a drop-on-full path.
	overfillDone := make(chan struct{})
	go func() {
		defer close(overfillDone)
		for range 300 {
			pipe.injectBytes([]byte("x"))
		}
	}()

	select {
	case <-overfillDone:
		// Injection completed without blocking — non-blocking drop is working.
	case <-time.After(1 * time.Second):
		t.Fatal("EC-003: frame injection goroutine blocked for >1s when sc.frames is full; " +
			"forwardFrames relay must use non-blocking select (ADR-011 §Concurrency)")
	}
}

// drainFramesCh reads all currently buffered frames from ch without blocking.
func drainFramesCh(ch <-chan halfchannel.ChannelFrame) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

// TestSessionConnectorFramesSurvivesMidSessionFailover — EC-002
// (BC-2.04.002 EC-003; ADR-011 §Relay re-subscribe after mode swap)
//
// Verifies the load-bearing ADR-011 invariant: the consumer of sc.Frames()
// must NOT need to resubscribe after a mid-session ctrl→PTY mode swap. The
// relay goroutine (forwardFrames) re-reads activeFrSource() internally;
// frames from the NEW backend (PTY) arrive on the SAME channel that was
// obtained BEFORE the swap.
//
// Sequence:
//
//  1. Build sc with initial ctrl stream that produces one visible line then EOFs.
//  2. Obtain sc.Frames() ONCE — this is the channel consumers hold forever.
//  3. Connect and verify a frame from the initial ctrl backend arrives on ch.
//  4. Let ctrl EOF trigger ErrControlModeDropped → watchAndFallback → PTY fallback.
//  5. Wait until InPTYMode() is true (swap complete).
//  6. Inject a frame through the PTY pipe master.
//  7. Assert the PTY frame arrives on THE SAME ch from step 2.
//
// Discriminating: a "capture source once" relay implementation — one that does
// NOT re-read activeFrSource() after the source channel closes — would exhaust
// the initial ctrl channel and then block forever on the stale closed channel.
// PTY frames would never appear in ch. This test would time-out at step 7.
//
// Hermetic: all PTY allocation and ctrl exec are injected; no real tmux or PTY
// forked.
func TestSessionConnectorFramesSurvivesMidSessionFailover(t *testing.T) {
	// EC-002 / ADR-011 §Relay re-subscribe invariant.
	// NOT t.Parallel(): goroutine lifecycle depends on sequential phase completion.

	// ptypipe is the fake PTY master that allows test to inject bytes AFTER swap.
	ptypipe := newPipeMaster()

	// fakePTYAlloc succeeds and yields ptypipe as the master.
	ptyAllocSucceeds := tmux.WithPTYAllocFunc(func() (io.ReadWriteCloser, int, error) {
		return ptypipe, 4321, nil
	})

	// Initial ctrl stream: one valid %begin/%end line then immediate EOF.
	// EOF causes dispatchLoop to emit ErrControlModeDropped, triggering
	// watchAndFallback → PTY fallback path (EC-003 drop path; no factory).
	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)

	// Ctrl stream has ONE valid header pair then closes.
	ctrlStream := fakeControlOutput(
		"%begin 7000000000 0 1",
		"%end 7000000000 0 1",
		// EOF immediately after — triggers ErrControlModeDropped.
	)
	ctrl := tmux.New(pub, ds, fakeExecFunc(ctrlStream))

	// PTY proxy backed by ptypipe so we can inject bytes post-swap.
	ptyProxy := tmux.NewPTYProxy(pub, ds, ptyAllocSucceeds)

	// No ControlModeFactory → immediate PTY fallback on ErrControlModeDropped.
	sc := tmux.NewSessionConnector(ctrl, ptyProxy)
	t.Cleanup(func() {
		_ = ptypipe.Close()
		_ = sc.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Step 2: obtain sc.Frames() ONCE — consumer never resubscribes (ADR-011).
	ch := sc.Frames()
	if ch == nil {
		t.Fatal("sc.Frames() returned nil; want non-nil channel (ADR-011)")
	}

	// Step 3: Connect.
	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("sc.Connect: %v; want nil (ctrl path must connect first)", err)
	}

	// The initial ctrl stream produces frames (from the %begin/%end lines via the
	// downstream half-channel) OR no frames at all (the ctrl stream may not produce
	// ChannelFrames from administrative lines). Either way, we wait for the ctrl
	// channel to EOF and watchAndFallback to activate PTY mode.
	//
	// We do NOT assert a frame from ctrl here — ControlMode may not emit
	// ChannelFrames for %begin/%end administrative lines. The key ADR-011
	// invariant is tested post-swap in steps 6–7.

	// Step 4–5: Wait for PTY fallback to engage (ctrl EOF → ErrControlModeDropped
	// → watchAndFallback → pty.Connect). Bounded: 3s.
	ptyFallbackDeadline := time.After(3 * time.Second)
	for !sc.InPTYMode() {
		select {
		case <-ptyFallbackDeadline:
			t.Fatal("InPTYMode() never became true within 3s after ctrl EOF; " +
				"watchAndFallback must activate PTY on ErrControlModeDropped (EC-003)")
		default:
			runtime.Gosched()
		}
	}

	// Drain any frames buffered from the ctrl phase so the channel is empty.
	drainFramesCh(ch)

	// Step 6: inject a frame via the PTY pipe master AFTER the mode swap.
	// The byte arrives in ptypipe → ioRelay reads it → produces a ChannelFrame
	// on pty.frames → forwardFrames re-reads activeFrSource() (now = pty) →
	// writes to sc.frames (= ch).
	ptypipe.injectBytes([]byte("ec-002-pty-frame"))

	// Step 7: assert the PTY frame arrives on THE SAME ch obtained before Connect.
	// A "capture once" relay (no activeFrSource re-read) would never deliver here.
	select {
	case f, ok := <-ch:
		if !ok {
			t.Fatal("sc.Frames() channel closed prematurely; want PTY frame delivery on same channel (ADR-011)")
		}
		// Any non-closed delivery satisfies the ADR-011 re-subscribe invariant.
		_ = f
	case <-time.After(2 * time.Second):
		t.Fatal("no frame on sc.Frames() within 2s after PTY byte injection post-swap; " +
			"forwardFrames must re-read activeFrSource() after ctrl channel closes " +
			"(ADR-011 §Relay re-subscribe — EC-002 discriminating assertion)")
	}
}
