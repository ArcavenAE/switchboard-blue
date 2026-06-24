// Package halfchannel implements the timeslice clock state machine for one
// directional half of a Switchboard channel (BC-2.01.001, BC-2.01.002,
// BC-2.01.003). The package is pure-core: no goroutines, no timers, no I/O.
// The effectful layer drives Tick() on its own schedule (ARCH-09).
package halfchannel

import (
	"errors"
	"time"

	"github.com/arcavenae/switchboard/internal/frame"
)

// FrameTypeData and FrameTypeEmptyTick are aliases of the canonical wire-format
// constants defined in internal/frame (ARCH-02 §3.1). Using aliases instead of
// local literals ensures these values stay in sync with the wire-format source
// of truth automatically — no "keep in sync" comment is needed.
const (
	FrameTypeData      = frame.FrameTypeData
	FrameTypeEmptyTick = frame.FrameTypeEmptyTick
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

// MinTickInterval and MaxTickInterval are the canonical bounds for tick
// scheduling per ADR-008. They are exported as documentary constants for
// the effectful scheduling layer to use in its own range check; the
// pure-core HalfChannel does NOT validate `tickInterval` against these
// bounds inside New (purity: a constructor that panics is a code smell).
// Callers MUST keep their configured interval within [MinTickInterval,
// MaxTickInterval] — exceeding the bounds is undefined behavior at this
// layer.
const (
	MinTickInterval = 5 * time.Millisecond
	MaxTickInterval = 50 * time.Millisecond
)

// ChannelFrame is the output of a single Tick() call: the channel-header
// fields and the payload bytes for that timeslice. The outer header
// (OuterHeader from internal/frame) is assembled by the effectful network
// layer, not here (ARCH-09 purity boundary; BC-2.01.005 invariant 1).
//
// FrameType is set to frame.FrameTypeData (0x01) when Payload is non-nil, or
// frame.FrameTypeEmptyTick (0x02) when Payload is nil (BC-2.01.002
// postcondition 1–2). The outer-assembler reads FrameType and sets
// OuterHeader.frame_type accordingly.
//
// Flags bit layout (ARCH-02 §3.2):
//
//	bit 0 — FEC_present
//	bit 1 — ARQ_req
//	bit 2 — SACK_present
type ChannelFrame struct {
	ChanID    uint32
	ChanSeq   uint32
	FrameType byte
	Flags     byte
	Payload   []byte
}

// ErrEmptyPayload is returned by Enqueue when a nil or zero-length payload is
// passed (BC-2.01.002 precondition 4).
var ErrEmptyPayload = errors.New("enqueue: payload must not be empty")

// HalfChannel is a pure-core timeslice clock state machine for one
// directional half of a Switchboard session channel.
//
// Concurrency: HalfChannel is not safe for concurrent use. The effectful
// scheduling layer MUST ensure Tick() and Enqueue() are called from a
// single goroutine or under external synchronisation.
type HalfChannel struct {
	chanID       uint32
	direction    Direction
	seq          uint32
	pending      [][]byte
	tickInterval time.Duration
}

// New constructs a HalfChannel with the given channel identifier, direction,
// and nominal tick interval. The tick interval is stored for external
// reference (e.g. benchmark scheduling); HalfChannel never calls time.Now()
// or time.Sleep() internally (ARCH-09 purity).
func New(chanID uint32, direction Direction, tickInterval time.Duration) *HalfChannel {
	return &HalfChannel{
		chanID:       chanID,
		direction:    direction,
		tickInterval: tickInterval,
	}
}

// Tick advances the state machine by one timeslice and returns exactly one
// ChannelFrame (AC-001). When the pending queue is empty the returned frame
// has nil Payload and FrameType = FrameTypeEmptyTick (AC-002, BC-2.01.002
// postcondition 1–2). When payload is available, FrameType = FrameTypeData.
// Each call increments seq by 1 (AC-004, BC-2.01.001 postcondition 5).
func (h *HalfChannel) Tick() ChannelFrame {
	h.seq++

	var payload []byte
	var frameType byte = FrameTypeEmptyTick
	if len(h.pending) > 0 {
		payload = h.pending[0]
		// Nil the freed slot before reslicing to allow GC of large payloads.
		h.pending[0] = nil
		h.pending = h.pending[1:]
		frameType = FrameTypeData
	}

	return ChannelFrame{
		ChanID:    h.chanID,
		ChanSeq:   h.seq,
		FrameType: frameType,
		Flags:     0,
		Payload:   payload,
	}
}

// Enqueue appends payload to the pending queue for emission on subsequent
// ticks. The payload is not copied; the caller must not mutate it after
// passing it to Enqueue. Returns ErrEmptyPayload if payload is nil or has
// zero length (BC-2.01.002 precondition 4).
func (h *HalfChannel) Enqueue(payload []byte) error {
	if len(payload) == 0 {
		return ErrEmptyPayload
	}

	h.pending = append(h.pending, payload)

	return nil
}

// Seq returns the current sequence counter value. The counter starts at 0
// and is incremented by each Tick() call (BC-2.01.001 postcondition 5).
func (h *HalfChannel) Seq() uint32 {
	return h.seq
}

// TickInterval returns the nominal tick period passed to New. Exposed so the
// effectful scheduling layer can read back the configured interval without
// accessing unexported fields.
func (h *HalfChannel) TickInterval() time.Duration {
	return h.tickInterval
}

// Direction returns the channel direction (Upstream or Downstream) configured
// at construction. The pure-core HalfChannel does not behave differently by
// direction; the field exists so effectful upstream code can route by direction
// without re-threading the value.
func (h *HalfChannel) Direction() Direction {
	return h.direction
}
