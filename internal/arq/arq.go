// Package arq implements downstream Automatic Repeat reQuest (ARQ) with
// piggybacked ACK/SACK and TLPKTDROP for the Switchboard routing engine
// (BC-2.02.005, BC-2.02.006).
//
// The console receiver maintains a sliding-window reorder buffer and
// delivers downstream frames to the terminal in sequence order. The access
// node sender maintains a send buffer and retransmits frames whose sequence
// numbers are reported missing in the SACK bitmap.
//
// SACK bitmap location: the 8-byte SACK field lives in the channel header
// when SACK_present=1 (flags bit 2). It is never placed in the outer header
// payload (ARCH-02; fixes spec drift F-P8-007).
//
// When a downstream frame exceeds its deadline (configurable; default
// 2 × tick_interval) the access node fires TLPKTDROP: retransmits cease,
// a degradation event is sent on the DegradationEvents channel, and the
// console advances past the dropped sequence (BC-2.02.006).
//
// ARQ resync on reconnect (ADR-005): on path failover the receiver sends a
// RESYNC signal requesting retransmit from last_acked_seq+1. In-flight
// frames during failover are lost; the SACK bitmap drives recovery.
//
// This package is pure-core: no I/O. TLPKTDROP signals via channel not OS
// calls. All side-effects (timers, network) are owned by the caller.
//
// math/bits is used for SACK bitmap population count (VP-052).
package arq

import (
	"errors"
	"math/bits"
	"sync"
	"time"
)

// SACKBitmapBytes is the size of the SACK bitmap field in the channel header
// when SACK_present=1 (ARCH-02 §3.2: 8 bytes = 64-bit bitmap).
const SACKBitmapBytes = 8

// DefaultDropTimeoutMultiplier is the coefficient applied to tick_interval to
// derive the TLPKTDROP deadline (ARCH-03: 2 × tick_interval).
const DefaultDropTimeoutMultiplier = 2

// ErrDuplicateSequence is returned by OnAck when the caller attempts to
// acknowledge a sequence number that has already been acknowledged and
// delivered (AC-001: no frame is delivered twice).
var ErrDuplicateSequence = errors.New("arq: duplicate sequence number already acknowledged")

// ErrSequenceNotInFlight is returned by TLPKTDROP when the supplied sequence
// number is not present in the retransmit queue — either it was already
// acknowledged or it was never sent.
var ErrSequenceNotInFlight = errors.New("arq: sequence not in retransmit queue")

// ErrFrameNotOverdue is returned by TLPKTDROP when the frame identified by
// overdue_seq has not yet exceeded its configured deadline. Only frames past
// their deadline may be dropped (AC-005).
var ErrFrameNotOverdue = errors.New("arq: frame has not exceeded drop deadline")

// DegradationEvent is sent on ARQ.DegradationEvents when TLPKTDROP fires for
// a frame. It carries the dropped sequence number so the metrics layer can
// track TLPKTDROP rate per session (BC-2.02.006 invariant 3, BC-2.06.001).
type DegradationEvent struct {
	// DroppedSeq is the channel sequence number of the frame that was
	// terminated by TLPKTDROP.
	DroppedSeq uint32
}

// inFlightFrame tracks a frame in the sender's retransmit queue.
type inFlightFrame struct {
	seq      uint32
	payload  []byte
	deadline time.Time
}

// recvEntry is a slot in the receiver's reorder buffer.
type recvEntry struct {
	seq     uint32
	payload []byte
}

// ARQ is the state machine for downstream ARQ with piggybacked ACK/SACK and
// TLPKTDROP. It is safe for concurrent use.
//
// Receiver role (console): OnAck advances the cumulative acknowledgment,
// processes the SACK bitmap, and delivers frames in order via DeliveredFrames.
//
// Sender role (access node): TLPKTDROP terminates an overdue frame, signals
// degradation via DegradationEvents, and removes the frame from the retransmit
// queue.
//
// Zero value is not usable; construct via New.
type ARQ struct {
	mu sync.Mutex

	// dropTimeout is the per-frame deadline. Frames past this duration since
	// their send time are eligible for TLPKTDROP (AC-005).
	dropTimeout time.Duration

	// nextExpected is the next sequence number the receiver expects to deliver
	// in-order (cumulative ACK pointer).
	nextExpected uint32

	// acked is the set of sequence numbers that have been acknowledged and
	// delivered. Used to enforce no-duplicate-delivery (AC-001).
	acked map[uint32]struct{}

	// reorderBuf holds out-of-order received frames awaiting in-order delivery
	// (AC-002).
	reorderBuf []recvEntry

	// inFlight is the sender's retransmit queue: frames sent but not yet ACKed.
	inFlight map[uint32]*inFlightFrame

	// DeliveredFrames receives frames in order as they are ready for terminal
	// output. Callers drain this channel to consume delivered payloads.
	DeliveredFrames chan []byte

	// DegradationEvents receives a DegradationEvent each time TLPKTDROP fires.
	// The metrics layer reads this channel to update the quality indicator
	// (BC-2.02.006 invariant 3).
	DegradationEvents chan DegradationEvent
}

// Config carries construction parameters for New.
type Config struct {
	// DropTimeout is the per-frame deadline after which TLPKTDROP fires.
	// Defaults to DefaultDropTimeoutMultiplier × TickInterval when zero.
	DropTimeout time.Duration

	// TickInterval is the half-channel tick period. Used to derive DropTimeout
	// when DropTimeout is zero.
	TickInterval time.Duration

	// DeliveredBufSize is the buffer depth of the DeliveredFrames channel.
	// Zero means unbuffered.
	DeliveredBufSize int

	// DegradationBufSize is the buffer depth of the DegradationEvents channel.
	// Zero means unbuffered.
	DegradationBufSize int
}

// New constructs an ARQ state machine with the given configuration.
func New(cfg Config) *ARQ {
	dropTimeout := cfg.DropTimeout
	if dropTimeout == 0 {
		dropTimeout = time.Duration(DefaultDropTimeoutMultiplier) * cfg.TickInterval
	}
	return &ARQ{
		dropTimeout:       dropTimeout,
		acked:             make(map[uint32]struct{}),
		inFlight:          make(map[uint32]*inFlightFrame),
		DeliveredFrames:   make(chan []byte, cfg.DeliveredBufSize),
		DegradationEvents: make(chan DegradationEvent, cfg.DegradationBufSize),
	}
}

// OnAck advances the receiver state machine using a cumulative ACK and SACK
// bitmap read from the channel header.
//
// ackSeq is the cumulative acknowledgment: the sender has received all frames
// up to and including ackSeq.
//
// sackBitmap is the 8-byte (64-bit) SACK bitmap from the channel header
// (SACK_present=1, flags bit 2). Each bit i at offset above ackSeq indicates
// that frame ackSeq+1+i has been received out of order.
//
// Frames newly made deliverable in-order are sent on DeliveredFrames.
// Duplicate sequence numbers are ignored (idempotent per EC-001).
//
// Not yet implemented — this is a stub.
func (a *ARQ) OnAck(ackSeq uint32, sackBitmap [SACKBitmapBytes]byte) error {
	panic("not yet implemented")
}

// SACKFromChannelHeader extracts the 8-byte SACK bitmap from a channel header
// byte slice. The SACK field is present only when SACK_present=1 (flags byte
// bit 2). channelHeader must be at least 20 bytes when SACK is present (the
// base 12-byte header plus the 8-byte conditional SACK field per ARCH-02 §3.2).
//
// Returns the bitmap and true if SACK_present=1, or a zero bitmap and false
// if the flag is clear.
//
// Not yet implemented — this is a stub.
func SACKFromChannelHeader(channelHeader []byte) ([SACKBitmapBytes]byte, bool, error) {
	panic("not yet implemented")
}

// EnqueueSend records a newly-sent frame in the sender's retransmit queue and
// sets its deadline to now+dropTimeout. The frame is eligible for TLPKTDROP
// once the deadline passes.
//
// Not yet implemented — this is a stub.
func (a *ARQ) EnqueueSend(seq uint32, payload []byte, now time.Time) {
	panic("not yet implemented")
}

// TLPKTDROP terminates the overdue frame identified by overdueSeq. It removes
// the frame from the retransmit queue and sends a DegradationEvent on
// DegradationEvents.
//
// Returns ErrSequenceNotInFlight if overdueSeq is not in the retransmit queue.
// Returns ErrFrameNotOverdue if the frame has not yet passed its deadline
// (AC-005: only overdue frames may be dropped).
//
// Not yet implemented — this is a stub.
func (a *ARQ) TLPKTDROP(overdueSeq uint32, now time.Time) error {
	panic("not yet implemented")
}

// SACKPopCount returns the number of set bits in the SACK bitmap. Used by
// tests to assert the bitmap accurately reflects received/missing frames
// (VP-052).
//
// Uses math/bits.OnesCount64 via encoding/binary for a one-line body.
func SACKPopCount(bitmap [SACKBitmapBytes]byte) int {
	return bits.OnesCount64(bitmapToUint64(bitmap))
}

// bitmapToUint64 converts the 8-byte big-endian SACK bitmap to a uint64.
func bitmapToUint64(b [SACKBitmapBytes]byte) uint64 {
	return uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
		uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
}
