// Package halfchannel implements the timeslice clock state machine for one
// directional half of a Switchboard channel (BC-2.01.001, BC-2.01.002,
// BC-2.01.003). The package is pure-core: no goroutines, no timers, no I/O.
// The effectful layer drives Tick() on its own schedule (ARCH-09).
package halfchannel

import (
	"time"

	// frame is the only internal dependency permitted by ARCH-08 topological
	// order position 7. FrameTypeData and FrameTypeEmptyTick are used by the
	// implementer when populating ChannelFrame.Flags and related fields.
	_ "github.com/arcavenae/switchboard/internal/frame"
)

// Direction identifies which half of a bidirectional channel this instance
// represents.
type Direction uint8

const (
	// Upstream is the client-to-server direction.
	Upstream Direction = iota
	// Downstream is the server-to-client direction.
	Downstream
)

// Tick interval bounds per ADR-008. The effectful scheduling layer MUST keep
// its timer period within [MinTickInterval, MaxTickInterval]. The pure-core
// HalfChannel itself never reads a wall clock.
const (
	MinTickInterval = 5 * time.Millisecond
	MaxTickInterval = 50 * time.Millisecond
)

// ChannelFrame is the output of a single Tick() call: the channel-header
// fields and the payload bytes for that timeslice. The outer header
// (OuterHeader from internal/frame) is assembled by the effectful network
// layer, not here (ARCH-09 purity boundary; BC-2.01.005 invariant 1).
//
// Flags bit layout (ARCH-02 §3.2):
//
//	bit 0 — FEC_present
//	bit 1 — ARQ_req
//	bit 2 — SACK_present
type ChannelFrame struct {
	ChanID  uint32
	ChanSeq uint32
	Flags   byte
	Payload []byte
}

// HalfChannel is a pure-core timeslice clock state machine for one
// directional half of a Switchboard session channel.
//
// Concurrency: HalfChannel is not safe for concurrent use. The effectful
// scheduling layer MUST ensure Tick() and Enqueue() are called from a
// single goroutine or under external synchronisation.
type HalfChannel struct {
	chanID       uint32        //nolint:unused // used by implementer in Tick/Seq/Enqueue
	direction    Direction     //nolint:unused // used by implementer in Tick
	seq          uint32        //nolint:unused // used by implementer in Tick/Seq
	pending      [][]byte      //nolint:unused // used by implementer in Enqueue/Tick
	tickInterval time.Duration //nolint:unused // used by implementer in TickInterval
	mtu          int           //nolint:unused // used by implementer in Tick for payload truncation
}

// New constructs a HalfChannel with the given channel identifier, direction,
// and nominal tick interval. The tick interval is stored for external
// reference (e.g. benchmark scheduling); HalfChannel never calls time.Now()
// or time.Sleep() internally (ARCH-09 purity).
//
// mtu is the maximum payload size in bytes for a single emitted frame. Pass
// 0 to use a caller-defined default (the implementer should define the
// sentinel).
func New(chanID uint32, direction Direction, tickInterval time.Duration) *HalfChannel {
	panic("not implemented: S-1.02 New")
}

// Tick advances the state machine by one timeslice and returns exactly one
// ChannelFrame (AC-001). When the pending queue is empty the returned frame
// has zero-length Payload (AC-002, BC-2.01.002 postcondition 1). Each call
// increments seq by 1 (AC-004, BC-2.01.001 postcondition 5).
func (h *HalfChannel) Tick() ChannelFrame {
	panic("not implemented: S-1.02 HalfChannel.Tick")
}

// Enqueue appends payload to the pending queue for emission on subsequent
// ticks. The payload is not copied; the caller must not mutate it after
// passing it to Enqueue. Returns an error if payload is nil.
func (h *HalfChannel) Enqueue(payload []byte) error {
	panic("not implemented: S-1.02 HalfChannel.Enqueue")
}

// Seq returns the current sequence counter value. The counter starts at 0
// and is incremented by each Tick() call (BC-2.01.001 postcondition 5).
func (h *HalfChannel) Seq() uint32 {
	panic("not implemented: S-1.02 HalfChannel.Seq")
}

// TickInterval returns the nominal tick period passed to New. Exposed so the
// effectful scheduling layer can read back the configured interval without
// accessing unexported fields.
func (h *HalfChannel) TickInterval() time.Duration {
	panic("not implemented: S-1.02 HalfChannel.TickInterval")
}
