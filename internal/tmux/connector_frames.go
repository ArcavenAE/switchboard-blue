// Package tmux — connector_frames.go implements SessionConnector.Frames(), the
// forwarding-channel API for failover-stable frame delivery (ADR-011; S-W3.04
// AC-004; drift W3-R2-M4).
//
// Design rationale: see ARCH-01 ADR-011. The caller receives one stable channel
// for the lifetime of the session; the relay goroutine (forwardFrames) re-plumbs
// internally on ctrl→PTY mode switch without disturbing the consumer.
//
// Stub: Frames() compiles and returns sc.frames, but the relay goroutine is not
// started (panic("not implemented")). Tests exercising failover will fail (Red
// Gate — BC-5.38.001).
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
// STUB: panics (S-W3.04 AC-004). The real body must type-assert sc.active to
// framesSource and return it; that assertion is non-trivial (may panic if active
// is nil or wrong type — needs guard logic). Keeping as panic per BC-5.38.001.
func (sc *SessionConnector) activeFrSource() framesSource {
	panic("not implemented: SessionConnector.activeFrSource (S-W3.04 AC-004)")
}

// Frames returns the stable forwarding channel for downstream frame consumers
// (ADR-011; AC-004; AC-005). The channel is buffered to framesBufferSize.
//
// The consumer calls Frames() once after sc.Connect(ctx) succeeds and ranges
// over the returned channel for the session lifetime. When sc.Close() is called,
// the channel is closed exactly once via closeForwardFrames (sync.Once).
//
// STUB: the relay goroutine that feeds sc.frames is not yet started; this method
// compiles but the forwarding relay is unimplemented. Callers will see sc.frames
// remain empty across a ctrl→PTY failover — TestSessionConnectorFramesSurviveFailover
// will fail (Red Gate, BC-5.38.001 / S-W3.04 AC-004).
func (sc *SessionConnector) Frames() <-chan halfchannel.ChannelFrame {
	panic("not implemented: SessionConnector.Frames() relay goroutine (S-W3.04 AC-004)")
}

// forwardFrames is the relay goroutine body (ADR-011 §Concurrency contract).
// It ranges over the current active mode's Frames() channel, writing each frame
// non-blocking to sc.frames (drop on full). On source-channel close it
// re-reads sc.activeFrSource() and continues ranging over the new source.
// Exits when sc.frames is closed.
//
// STUB: not implemented — started as a goroutine in Connect once implemented.
func (sc *SessionConnector) forwardFrames() {
	panic("not implemented: SessionConnector.forwardFrames relay goroutine (S-W3.04 AC-004)")
}
