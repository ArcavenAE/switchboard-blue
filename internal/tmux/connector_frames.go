// Package tmux — connector_frames.go implements SessionConnector.Frames(), the
// forwarding-channel API for failover-stable frame delivery (ADR-011; S-W3.04
// AC-004; drift W3-R2-M4).
//
// Design rationale: see ARCH-01 ADR-011. The caller receives one stable channel
// for the lifetime of the session; the relay goroutine (forwardFrames) re-plumbs
// internally on ctrl→PTY mode switch without disturbing the consumer.
package tmux

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"

	"github.com/arcavenae/switchboard/internal/halfchannel"
)

// ErrPTYSourceEOF is sent on sc.Err() when the forwardFrames relay detects EOF
// on the active PTY source channel without a prior sc.Close() call. This is
// E-SYS-003 (error-taxonomy.md §SYS; ARCH-01 ADR-011 v1.5 §HIGH-A;
// BC-2.04.002 EC-008).
//
// The sentinel string satisfies ST1005 (no trailing punctuation). Callers use
// errors.Is to detect this condition. It flows through the sc.Err() drain path
// in runAccess and is surfaced to the operator as E-SYS-002 format:
//
//	"fatal: cannot connect to session backend: session connector: PTY source EOF"
var ErrPTYSourceEOF = errors.New("session connector: PTY source EOF")

// framesSource is the interface satisfied by both ControlMode and PTYProxy for
// their per-instance frame channel. activeSourceSnapshot uses this interface so
// the relay goroutine does not need to know which concrete mode is active.
type framesSource interface {
	Frames() <-chan halfchannel.ChannelFrame
}

// activeSourceSnapshot returns the current active source, its Frames() channel,
// and whether the connector is in PTY mode — all read under a single sc.mu hold
// (ARCH-01 ADR-011 v1.6 §HIGH-A TOCTOU fix).
//
// src.Frames() is called inside the lock because src is the interface value
// captured from sc.active; calling it outside would allow watchAndFallback to
// replace sc.active between the two reads, breaking the soundness proof
// (pty.frames != ctrl.frames is the discriminator — it must be stable).
//
// Returns (nil, nil, false) if the connector is closed or has no active source.
//
// swapBarrier (test-only deterministic interleaving seam; nil in production):
// if sc.swapBarrier is set, the snapshot is held at this point — outside the
// lock, after all three fields are read — so a test goroutine can complete a
// ctrl→PTY swap before the relay acts on the snapshot. This confirms that the
// atomic snapshot (not a two-lock read) prevents misclassification.
func (sc *SessionConnector) activeSourceSnapshot() (framesSource, <-chan halfchannel.ChannelFrame, bool) {
	sc.mu.Lock()
	closed := sc.closed
	active := sc.active
	inPTY := sc.inPTYMode
	var src framesSource
	var srcCh <-chan halfchannel.ChannelFrame
	if !closed && active != nil {
		if s, ok := active.(framesSource); ok {
			src = s
			srcCh = s.Frames() // called inside the lock — safe, pure accessor
		}
	}
	sc.mu.Unlock()

	if src == nil {
		return nil, nil, false
	}

	// Test-only deterministic interleaving seam. swapBarrier is always nil in
	// production; the nil check is the only cost (branch prediction friendly).
	if sc.swapBarrier != nil {
		<-sc.swapBarrier
	}

	return src, srcCh, inPTY
}

// Frames returns the stable forwarding channel for downstream frame consumers
// (ADR-011; AC-004; AC-005). The channel is buffered to framesBufferSize.
//
// The consumer calls Frames() once after sc.Connect(ctx) succeeds and ranges
// over the returned channel for the session lifetime. When sc.Close() is called,
// the channel is closed exactly once via closeForwardFrames (sync.Once).
//
// The relay goroutine (forwardFrames) is started by Connect; callers should call
// Frames() before or after Connect — the channel is the same either way.
func (sc *SessionConnector) Frames() <-chan halfchannel.ChannelFrame {
	return sc.frames
}

// RelayDropped returns the cumulative count of frames dropped at the relay
// layer — frames that could not be written to sc.frames because the channel
// was full (non-blocking select default branch in forwardFrames).
//
// This is distinct from AccessNode.FramesDropped() which counts drops at the
// ConsoleSet fan-out layer. Both MUST be reported in the AC-006 log line.
// ARCH-01 v1.4 §Relay-drop counter contract; BC-2.04.006 v1.4 Inv-4.
func (sc *SessionConnector) RelayDropped() uint64 {
	return atomic.LoadUint64(&sc.relayDropped)
}

// startForwardFrames starts the relay goroutine and registers it with sc.wg.
// Called from Connect after the active mode is set. ctx is the innerCtx from
// Connect (derived from connectCancel) so forwardFrames can observe ctx.Done()
// and exit cleanly on Close without busy-spinning.
func (sc *SessionConnector) startForwardFrames(ctx context.Context) {
	sc.wg.Add(1)
	go func() {
		defer sc.wg.Done()
		sc.forwardFrames(ctx)
	}()
}

// forwardFrames is the relay goroutine body (ADR-011 §Concurrency contract,
// amended v1.6).
//
// Outer loop: select on ctx.Done() first — exits cleanly on Close without
// busy-spinning (ARCH-01 v1.4 §Relay busy-spin guard, retained). Then takes
// an atomic snapshot {src, srcCh, inPTY} via activeSourceSnapshot() — a single
// sc.mu hold so source identity, channel, and mode flag are mutually consistent
// (ARCH-01 ADR-011 v1.6 §HIGH-A TOCTOU fix).
//
// Post-inner-range discrimination (ARCH-01 v1.6 §(d)):
//
//	srcCh == prevSrcCh AND inPTY → terminal PTY-source EOF (sound, see soundness
//	  proof in ARCH-01 v1.6 §(c)): signal ErrPTYSourceEOF and exit.
//	srcCh == prevSrcCh AND !inPTY → ctrl→PTY swap in flight: yield and retry.
//	srcCh != prevSrcCh → new source has landed: range over it.
//
// Inner loop: ranges over the current source channel; writes each frame
// non-blocking to sc.frames. Drops on full, incrementing sc.relayDropped
// atomically (ARCH-01 v1.4 §Relay-drop counter contract).
//
// Exits when ctx is cancelled or sc is closed (activeSourceSnapshot returns nil).
func (sc *SessionConnector) forwardFrames(ctx context.Context) {
	// Close sc.frames exactly once on exit, so range-consumers unblock cleanly.
	defer sc.closeForwardFrames.Do(func() { close(sc.frames) })

	var prevSrcCh <-chan halfchannel.ChannelFrame

	for {
		// ARCH-01 v1.4 §Relay busy-spin guard: check ctx.Done() before re-reading
		// the active source — exits cleanly when Close cancels innerCtx.
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Atomic snapshot: {src, srcCh, inPTY} from one sc.mu hold.
		// src.Frames() is called inside the lock (see activeSourceSnapshot).
		_, srcCh, inPTY := sc.activeSourceSnapshot()
		if srcCh == nil {
			// Connector closed or no active source — exit.
			return
		}

		// Discriminate same-closed-channel case (ARCH-01 v1.6 §(d)).
		if srcCh == prevSrcCh {
			if inPTY {
				// Terminal PTY-source EOF: PTY swap already landed (inPTY=true
				// from same snapshot as srcCh), and pty.frames closed (same
				// channel as prevSrcCh). No further swap will arrive.
				// Signal session-fatal backend loss (BC-2.04.002 EC-008; E-SYS-003).
				// closeErrCh.Do guards against double-close with sc.Close() and
				// watchAndFallback.
				sc.closeErrCh.Do(func() {
					select {
					case sc.errCh <- ErrPTYSourceEOF:
					default:
					}
					close(sc.errCh)
				})
				return
			}
			// Control mode: the snapshot shows inPTY=false, meaning watchAndFallback
			// has not yet set sc.inPTYMode=true. A ctrl→PTY swap may be in flight.
			// Yield and retry — the swap will update sc.active and sc.inPTYMode
			// atomically under sc.mu on the next snapshot.
			runtime.Gosched()
			continue
		}
		prevSrcCh = srcCh

		// Range over the current source channel. When it closes (mode switch or
		// Close), the for-range exits and we loop back for a fresh snapshot.
		for f := range srcCh {
			select {
			case sc.frames <- f:
			default:
				// EC-003 / ARCH-01 v1.4: sc.frames full — drop frame, increment
				// relay-level counter (distinct from ConsoleSet-level drops).
				atomic.AddUint64(&sc.relayDropped, 1)
			}
		}
		// Loop: fresh snapshot at top — activeSourceSnapshot handles closed check.
	}
}
