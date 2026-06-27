// Package tmux — connector_frames.go implements SessionConnector.Frames(), the
// forwarding-channel API for failover-stable frame delivery (ADR-011; S-W3.04
// AC-004; drift W3-R2-M4).
//
// Design rationale: see ARCH-01 ADR-011. The caller receives one stable channel
// for the lifetime of the session; the relay goroutine (forwardFrames) re-plumbs
// internally on ctrl→PTY mode switch without disturbing the consumer.
package tmux

import "github.com/arcavenae/switchboard/internal/halfchannel"

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

// startForwardFrames starts the relay goroutine and registers it with sc.wg.
// Called from Connect after the active mode is set.
func (sc *SessionConnector) startForwardFrames() {
	sc.wg.Add(1)
	go func() {
		defer sc.wg.Done()
		sc.forwardFrames()
	}()
}

// forwardFrames is the relay goroutine body (ADR-011 §Concurrency contract).
// It ranges over the current active mode's Frames() channel, writing each frame
// non-blocking to sc.frames (drop on full). On source-channel close it
// re-reads sc.activeFrSource() and continues ranging over the new source.
// Exits when sc is closed (activeFrSource returns nil).
func (sc *SessionConnector) forwardFrames() {
	// Close sc.frames exactly once on exit, so range-consumers unblock cleanly.
	defer sc.closeForwardFrames.Do(func() { close(sc.frames) })

	for {
		src := sc.activeFrSource()
		if src == nil {
			// Connector closed or no active source — exit.
			return
		}
		srcCh := src.Frames()

		// Range over the current source channel. When it closes (mode switch or
		// Close), the for-range exits and we re-read the active source.
		for f := range srcCh {
			select {
			case sc.frames <- f:
			default:
				// EC-003: sc.frames full — drop frame, do not block.
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
