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
	"fmt"
	"math/bits"
	"sort"
	"sync"
	"time"
)

// SACKBitmapBytes is the size of the SACK bitmap field in the channel header
// when SACK_present=1 (ARCH-02 §3.2: 8 bytes = 64-bit bitmap).
const SACKBitmapBytes = 8

// DefaultDropTimeoutMultiplier is the coefficient applied to tick_interval to
// derive the TLPKTDROP deadline (ARCH-03: 2 × tick_interval).
const DefaultDropTimeoutMultiplier = 2

// channelHeaderBaseLen is the minimum channel header length (bytes 0–11).
const channelHeaderBaseLen = 12

// channelHeaderSACKLen is the full channel header length when SACK_present=1
// (base 12 bytes + 8-byte SACK field).
const channelHeaderSACKLen = 20

// sackPresentMask is the channel-header flags bit that signals SACK_present
// (ARCH-02 §3.2: flags byte at offset 8, bit 2).
const sackPresentMask = byte(0x04)

// sackWindowSize is the number of sequence positions covered by the SACK
// bitmap (64 bits = 64 positions above ackSeq). Frames outside this window
// are not retained in reorderBuf (BC-2.02.005 precondition 3).
const sackWindowSize = 64

// ErrSequenceNotInFlight is returned by TLPKTDROP when the supplied sequence
// number is not present in the retransmit queue — either it was already
// acknowledged or it was never sent.
var ErrSequenceNotInFlight = fmt.Errorf("arq: sequence not in retransmit queue")

// ErrFrameNotOverdue is returned by TLPKTDROP when the frame identified by
// overdue_seq has not yet exceeded its configured deadline. Only frames past
// their deadline may be dropped (AC-005).
var ErrFrameNotOverdue = fmt.Errorf("arq: frame has not exceeded drop deadline")

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

// ARQ is the state machine for downstream ARQ with piggybacked ACK/SACK and
// TLPKTDROP. It is driven by a single half-channel tick loop and is NOT safe
// for concurrent OnAck or TLPKTDROP calls from multiple goroutines.
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

	// nextExpected is the cumulative delivery pointer: we have delivered all
	// frames with seq <= nextExpected. The next frame to deliver in order is
	// nextExpected+1.
	nextExpected uint32

	// reorderBuf holds out-of-order received frames awaiting in-order delivery
	// (AC-002). Keyed by sequence number. Only frames within the SACK window
	// (ackSeq+1 .. ackSeq+64) are retained.
	reorderBuf map[uint32][]byte

	// inFlight is the sender's retransmit queue: frames sent but not yet ACKed.
	inFlight map[uint32]*inFlightFrame

	// prevDelivery is the "done" signal from the previously launched delivery
	// goroutine. Each OnAck call that produces frames launches a goroutine
	// that waits on prevDelivery before writing to DeliveredFrames, ensuring
	// global FIFO delivery order even though individual OnAck calls return
	// immediately (non-blocking). Guarded by mu.
	prevDelivery chan struct{}

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
	// prevDelivery is pre-signalled so the first delivery goroutine starts
	// immediately without waiting.
	ready := make(chan struct{}, 1)
	ready <- struct{}{}
	return &ARQ{
		dropTimeout:       dropTimeout,
		reorderBuf:        make(map[uint32][]byte),
		inFlight:          make(map[uint32]*inFlightFrame),
		prevDelivery:      ready,
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
func (a *ARQ) OnAck(ackSeq uint32, sackBitmap [SACKBitmapBytes]byte) error {
	a.mu.Lock()

	// Collect frames to deliver. All payloads are deep-copied above (in
	// payloadFor / the SACK clone), so the goroutine below owns the slice
	// and touches no other shared state.
	var toDeliver [][]byte

	// Step 1: deliver all frames from nextExpected+1 through ackSeq in order.
	// Frames with no known payload (not in inFlight or reorderBuf) are skipped
	// for delivery but still advance nextExpected — the cumulative ACK tells us
	// the remote side received them; we only surface what we have locally.
	for seq := a.nextExpected + 1; seq <= ackSeq; seq++ {
		payload := a.payloadFor(seq)
		a.nextExpected = seq
		// Remove from inFlight once delivered (or skipped).
		delete(a.inFlight, seq)
		delete(a.reorderBuf, seq)
		if payload != nil {
			toDeliver = append(toDeliver, payload)
		}
	}

	// Step 2: process SACK bitmap — buffer out-of-order frames for later flush.
	// Bit i (MSB-first, bit 0 = MSB of byte 0) = seq ackSeq+1+i.
	// Only retain frames within the SACK window [ackSeq+1 .. ackSeq+64] to
	// prevent unbounded reorderBuf growth (BC-2.02.005 precondition 3).
	u := bitmapToUint64(sackBitmap)
	for i := 0; i < sackWindowSize; i++ {
		if u&(uint64(1)<<(63-i)) != 0 {
			seq := ackSeq + 1 + uint32(i)
			if seq > a.nextExpected {
				if _, buffered := a.reorderBuf[seq]; !buffered {
					if f, ok := a.inFlight[seq]; ok {
						// Clone payload to decouple buffer from inFlight.
						p := make([]byte, len(f.payload))
						copy(p, f.payload)
						a.reorderBuf[seq] = p
					}
				}
			}
		}
	}

	// Step 3: flush consecutive frames from the reorder buffer.
	for {
		next := a.nextExpected + 1
		if p, ok := a.reorderBuf[next]; ok {
			toDeliver = append(toDeliver, p)
			a.nextExpected = next
			delete(a.reorderBuf, next)
			delete(a.inFlight, next)
		} else {
			break
		}
	}

	if len(toDeliver) == 0 {
		a.mu.Unlock()
		return nil
	}

	// Chain delivery goroutines to preserve global FIFO order of DeliveredFrames
	// while keeping OnAck non-blocking (the caller must not block on a partially-
	// drained channel — DeliveredBufSize may be smaller than the flush batch).
	prev := a.prevDelivery
	next := make(chan struct{}, 1)
	a.prevDelivery = next
	a.mu.Unlock()

	go func(frames [][]byte, prev, next chan struct{}) {
		<-prev // wait for the previous batch to finish delivering
		for _, p := range frames {
			a.DeliveredFrames <- p
		}
		next <- struct{}{} // signal the next batch it may start
	}(toDeliver, prev, next)

	return nil
}

// payloadFor returns the payload for seq, checking reorderBuf then inFlight.
// Returns nil if not found. Must be called with a.mu held.
func (a *ARQ) payloadFor(seq uint32) []byte {
	if p, ok := a.reorderBuf[seq]; ok {
		return p
	}
	if f, ok := a.inFlight[seq]; ok {
		p := make([]byte, len(f.payload))
		copy(p, f.payload)
		return p
	}
	return nil
}

// SACKFromChannelHeader extracts the 8-byte SACK bitmap from a channel header
// byte slice. The SACK field is present only when SACK_present=1 (flags byte
// bit 2). channelHeader must be at least 20 bytes when SACK is present (the
// base 12-byte header plus the 8-byte conditional SACK field per ARCH-02 §3.2).
//
// Returns the bitmap and true if SACK_present=1, or a zero bitmap and false
// if the flag is clear.
func SACKFromChannelHeader(channelHeader []byte) ([SACKBitmapBytes]byte, bool, error) {
	if len(channelHeader) < channelHeaderBaseLen {
		return [SACKBitmapBytes]byte{}, false, fmt.Errorf(
			"arq: channel header too short: need %d bytes, got %d",
			channelHeaderBaseLen, len(channelHeader),
		)
	}

	flags := channelHeader[8]
	if flags&sackPresentMask == 0 {
		return [SACKBitmapBytes]byte{}, false, nil
	}

	// SACK_present=1: the 8-byte bitmap must be present at bytes 12–19.
	if len(channelHeader) < channelHeaderSACKLen {
		return [SACKBitmapBytes]byte{}, false, fmt.Errorf(
			"arq: channel header too short for SACK field: need %d bytes, got %d",
			channelHeaderSACKLen, len(channelHeader),
		)
	}

	var bitmap [SACKBitmapBytes]byte
	copy(bitmap[:], channelHeader[12:20])
	return bitmap, true, nil
}

// EnqueueSend records a newly-sent frame in the sender's retransmit queue and
// sets its deadline to now+dropTimeout. The frame is eligible for TLPKTDROP
// once the deadline passes.
func (a *ARQ) EnqueueSend(seq uint32, payload []byte, now time.Time) {
	p := make([]byte, len(payload))
	copy(p, payload)

	a.mu.Lock()
	a.inFlight[seq] = &inFlightFrame{
		seq:      seq,
		payload:  p,
		deadline: now.Add(a.dropTimeout),
	}
	a.mu.Unlock()
}

// TLPKTDROP terminates the overdue frame identified by overdueSeq. It removes
// the frame from the retransmit queue and sends a DegradationEvent on
// DegradationEvents.
//
// Only the overdue frame's content is abandoned (BC-2.02.006 postcondition 5).
// Frames below overdueSeq that are still undelivered remain in-flight and are
// deliverable via subsequent OnAck calls.
//
// nextExpected is advanced only when overdueSeq == nextExpected+1 (the dropped
// frame was the next expected frame in sequence). If overdueSeq is ahead of
// nextExpected+1, lower undelivered frames are preserved.
//
// Returns ErrSequenceNotInFlight if overdueSeq is not in the retransmit queue.
// Returns ErrFrameNotOverdue if the frame has not yet passed its deadline
// (AC-005: only overdue frames may be dropped). The deadline check is
// exclusive: now must be strictly after the deadline (now.After(deadline)).
func (a *ARQ) TLPKTDROP(overdueSeq uint32, now time.Time) error {
	a.mu.Lock()

	f, ok := a.inFlight[overdueSeq]
	if !ok {
		a.mu.Unlock()
		return ErrSequenceNotInFlight
	}

	// Deadline is exclusive: the frame must be strictly past its deadline.
	if !now.After(f.deadline) {
		a.mu.Unlock()
		return ErrFrameNotOverdue
	}

	// Remove from retransmit queue.
	delete(a.inFlight, overdueSeq)
	delete(a.reorderBuf, overdueSeq)

	// Advance nextExpected only when overdueSeq is the immediate next frame.
	// This preserves lower undelivered frames (BC-2.02.006 PC5: only the
	// overdue frame's content is abandoned).
	if overdueSeq == a.nextExpected+1 {
		a.nextExpected = overdueSeq
	}

	a.mu.Unlock()

	// Send degradation event outside the lock.
	a.DegradationEvents <- DegradationEvent{DroppedSeq: overdueSeq}
	return nil
}

// GapsToRetransmit returns the sequence numbers of in-flight frames that are
// unacknowledged — neither cumulatively ACKed (seq <= ackSeq) nor marked
// received in the SACK bitmap — in ascending order (BC-2.02.005 PC2).
// SACK bitmap covers positions ackSeq+1..ackSeq+64 (bit 0 = MSB of byte 0 = ackSeq+1).
// In-flight frames at seq >= ackSeq+65 (outside the bitmap window) are also gaps.
func (a *ARQ) GapsToRetransmit(ackSeq uint32, sackBitmap [SACKBitmapBytes]byte) []uint32 {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.inFlight) == 0 {
		return nil
	}

	// Build a set of seqs covered by the SACK bitmap (received out-of-order).
	u := bitmapToUint64(sackBitmap)
	sacked := make(map[uint32]bool, bits.OnesCount64(u))
	for i := 0; i < sackWindowSize; i++ {
		if u&(uint64(1)<<(63-i)) != 0 {
			sacked[ackSeq+1+uint32(i)] = true
		}
	}

	// Collect in-flight frames that are gaps: not cumulatively ACKed and not
	// in the SACK bitmap.
	var gaps []uint32
	for seq := range a.inFlight {
		if seq <= ackSeq {
			// Cumulatively ACKed — not a gap.
			continue
		}
		if sacked[seq] {
			// In SACK bitmap — receiver already has it — not a gap.
			continue
		}
		gaps = append(gaps, seq)
	}

	sort.Slice(gaps, func(i, j int) bool { return gaps[i] < gaps[j] })
	return gaps
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
