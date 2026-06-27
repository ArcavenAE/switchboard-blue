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
// their per-instance frame channel. activeFrSource returns this interface so the
// relay goroutine does not need to know which concrete mode is active.
type framesSource interface {
	Frames() <-chan halfchannel.ChannelFrame
}

// activeFrSource returns the active frame source under sc.mu. The relay
// goroutine calls this after a mode switch to re-read the new source.
//
// Returns nil if sc.active is nil or does not implement framesSource.
func (sc *SessionConnector) activeFrSource() framesSource {
	sc.mu.Lock()
	active := sc.active
	closed := sc.closed
	sc.mu.Unlock()

	if closed || active == nil {
		return nil
	}
	src, _ := active.(framesSource)
	return src
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
// amended v1.4).
//
// Outer loop: select on ctx.Done() first — exits cleanly on Close without
// busy-spinning when a mode swap is in flight (ARCH-01 v1.4 §Relay busy-spin
// guard). On source-channel close, if the re-read source is identical to the
// just-closed source (swap has not landed yet), yields via runtime.Gosched()
// before retrying.
//
// Inner loop: ranges over the current source channel; writes each frame
// non-blocking to sc.frames. Drops on full, incrementing sc.relayDropped
// atomically (ARCH-01 v1.4 §Relay-drop counter contract).
//
// Exits when ctx is cancelled or sc is closed (activeFrSource returns nil).
func (sc *SessionConnector) forwardFrames(ctx context.Context) {
	// Close sc.frames exactly once on exit, so range-consumers unblock cleanly.
	defer sc.closeForwardFrames.Do(func() { close(sc.frames) })

	var prevSrcCh <-chan halfchannel.ChannelFrame

	for {
		// ARCH-01 v1.4 §Relay busy-spin guard: check ctx.Done() before
		// re-reading the active source — prevents spinning if Close races
		// with a mode swap.
		select {
		case <-ctx.Done():
			return
		default:
		}

		src := sc.activeFrSource()
		if src == nil {
			// Connector closed or no active source — exit.
			return
		}
		srcCh := src.Frames()

		// ARCH-01 v1.5 §HIGH-A + v1.4 §Relay busy-spin guard: if the returned
		// source channel is the same already-closed channel as the previous
		// iteration, discriminate by mode:
		//   PTY mode (inPTYMode true): no swap will arrive — the PTY shell
		//     process has exited. Signal session-fatal backend loss via sc.Err()
		//     and exit the relay.
		//   Control mode (inPTYMode false): a mode swap may be in flight
		//     (ctrl → PTY via watchAndFallback). Yield and retry so the swap
		//     can land under sc.mu.
		if srcCh == prevSrcCh {
			if sc.InPTYMode() {
				// Terminal source EOF — no swap coming. Signal fatal backend loss
				// (BC-2.04.002 EC-008; E-SYS-003). closeErrCh.Do guards against
				// double-close with sc.Close() or watchAndFallback.
				sc.closeErrCh.Do(func() {
					select {
					case sc.errCh <- ErrPTYSourceEOF:
					default:
					}
					close(sc.errCh)
				})
				return
			}
			// Control mode: swap is in flight. Yield and retry.
			runtime.Gosched()
			continue
		}
		prevSrcCh = srcCh

		// Range over the current source channel. When it closes (mode switch or
		// Close), the for-range exits and we re-read the active source.
		for f := range srcCh {
			select {
			case sc.frames <- f:
			default:
				// EC-003 / ARCH-01 v1.4: sc.frames full — drop frame, increment
				// relay-level counter (distinct from ConsoleSet-level drops).
				atomic.AddUint64(&sc.relayDropped, 1)
			}
		}

		// Source channel closed — check if we should exit or switch sources.
		sc.mu.Lock()
		closed := sc.closed
		sc.mu.Unlock()
		if closed {
			return
		}
		// Loop: re-read activeFrSource() for the new backend (PTY after ctrl drop).
	}
}
