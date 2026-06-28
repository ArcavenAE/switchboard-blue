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
// This package is pure-core: no I/O, no goroutines, no timers. OnAck returns
// deliverable frames synchronously; the caller's tick loop routes them to the
// terminal. All side-effects (timers, network) are owned by the caller.
// Single-writer contract: OnAck and TLPKTDROP must be called from a single
// goroutine (the half-channel tick loop). Concurrent calls are not safe.
//
// math/bits is used for SACK bitmap population count (BC-2.02.005 SACK-accuracy clause; VP-019/VP-020).
package arq

import (
	"fmt"
	"math/bits"
	"sort"
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

// SackWindowSize is the number of sequence positions covered by the SACK
// bitmap (64 bits = 64 positions above ackSeq). Frames outside this window
// are not retained in reorderBuf (BC-2.02.005 precondition 3).
const SackWindowSize = 64

// sackWindowSize is the package-internal alias for SackWindowSize.
const sackWindowSize = SackWindowSize

// ErrAckOutOfWindow is returned by OnAck when the cumulative ACK sequence
// number falls outside the valid window: ackSeq must satisfy
// ackSeq - nextExpected <= sackWindowSize (64). An out-of-window ackSeq is
// a protocol-illegal frame; the caller should log and discard it.
// Traces to: BC-2.02.005 PC-3, EC-005; RULING-003.
var ErrAckOutOfWindow = fmt.Errorf("arq: cumulative ACK out of window")

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
	//
	// The zero value (DroppedSeq == 0) is the "no event" sentinel. 0 is never
	// a valid frame sequence number because chan_seq is reserved to start at 1
	// per ARCH-02 §chan_seq (RULING-001 §R1); the sentinel therefore cannot
	// collide with a real dropped sequence.
	DroppedSeq uint32
}

// inFlightFrame tracks a frame in the sender's retransmit queue.
type inFlightFrame struct {
	seq      uint32
	payload  []byte
	deadline time.Time
}

// ARQ is the state machine for downstream ARQ with piggybacked ACK/SACK and
// TLPKTDROP. It is driven by a single half-channel tick loop.
//
// Single-writer contract: OnAck and TLPKTDROP MUST be called from a single
// goroutine. Concurrent calls are NOT safe. There are no internal goroutines
// or mutexes — this is a pure-core state machine (see package doc).
//
// Receiver role (console): OnAck advances the cumulative acknowledgment,
// processes the SACK bitmap, and returns frames in delivery order.
//
// Sender role (access node): TLPKTDROP terminates an overdue frame, returns
// a DegradationEvent, and signals degradation via DegradationEvents channel.
//
// Zero value is not usable; construct via New.
type ARQ struct {
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
	//
	// NOTE: inFlight has no window bound in this package. The caller (sender-side
	// wiring, S-5.01) is responsible for bounding the number of concurrent
	// EnqueueSend calls to the negotiated ARQ window size. Unbounded EnqueueSend
	// calls without corresponding OnAck calls will grow this map without limit.
	// Window enforcement is deferred to S-5.01 (BC-2.02.005 precondition 3).
	inFlight map[uint32]*inFlightFrame

	// DegradationEvents receives a DegradationEvent each time TLPKTDROP fires.
	// The metrics layer reads this channel to update the quality indicator
	// (BC-2.02.006 invariant 3). Always buffered (minimum 1; see New).
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

	// DegradationBufSize is the buffer depth of the DegradationEvents channel.
	// Zero or negative defaults to a minimum buffer of 1 — the DegradationEvents
	// channel is always buffered so TLPKTDROP never blocks the caller when the
	// metrics layer is momentarily behind.
	DegradationBufSize int
}

// New constructs an ARQ state machine with the given configuration.
func New(cfg Config) *ARQ {
	dropTimeout := cfg.DropTimeout
	if dropTimeout == 0 {
		dropTimeout = time.Duration(DefaultDropTimeoutMultiplier) * cfg.TickInterval
	}
	degBufSize := cfg.DegradationBufSize
	if degBufSize < 1 {
		degBufSize = 1
	}
	return &ARQ{
		dropTimeout:       dropTimeout,
		reorderBuf:        make(map[uint32][]byte),
		inFlight:          make(map[uint32]*inFlightFrame),
		DegradationEvents: make(chan DegradationEvent, degBufSize),
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
// Returns the ordered slice of frame payloads ready for terminal output.
// The returned slices are owned by the caller; ARQ retains no reference to
// them after return. The caller's tick loop forwards them to the terminal.
//
// Duplicate sequence numbers are ignored (idempotent per EC-001).
//
// Concurrency: must be called from a single goroutine (the tick loop).
// Not safe for concurrent calls.
func (a *ARQ) OnAck(ackSeq uint32, sackBitmap [SACKBitmapBytes]byte) ([][]byte, error) {
	var toDeliver [][]byte

	// Record the pre-call nextExpected so Step 3 can guard the flush: reorderBuf
	// frames are only deliverable once the cumulative ACK has advanced past at
	// least one sequence (prevNextExpected < ackSeq). Without this guard, SACK
	// frames buffered in Step 2 would be immediately flushed in Step 3 even when
	// no cumulative progress occurred (e.g. OnAck(0, allBitsSet)).
	prevNextExpected := a.nextExpected

	// Validate ackSeq is within one ARQ window of nextExpected.
	// ackSeq is wire-derived (peer/attacker-controlled). A legal cumulative ACK
	// advances at most sackWindowSize (64) positions. An out-of-window value
	// would drive the Step-1 loop for up to 2^32 iterations — a per-frame DoS.
	// Reject without iterating (RULING-003; BC-2.02.005 PC-3, EC-005).
	//
	// The subtraction is unsigned: if ackSeq < nextExpected the result wraps to
	// a large uint32 (> sackWindowSize), so the guard also correctly rejects
	// stale (already-ACKed) values without a separate comparison.
	if ackSeq-a.nextExpected > sackWindowSize {
		return nil, ErrAckOutOfWindow
	}

	// Step 1: deliver all frames from nextExpected+1 through ackSeq in order.
	// Frames with no known payload (not in inFlight or reorderBuf) are skipped
	// for delivery but still advance nextExpected — the cumulative ACK tells us
	// the remote side received them; we only surface what we have locally.
	// Advancing past locally-absent seqs is correct and intended: see BC-2.02.005
	// invariant 4 and PC-4 scope note (F-H4 ruling, disposition A).
	//
	// Note: this loop does not implement RFC-1982-style serial-number arithmetic
	// for uint32 wraparound (MaxUint32 → 1). 32-bit sequence wraparound is out
	// of MVP scope per RULING-001 §R2: sessions are bounded well below the
	// ~49–497-day wrap interval at any normal tick rate (BC-2.02.002 EC-004).
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

	// Step 3: flush consecutive frames from the reorder buffer, but only when
	// the cumulative ACK advanced in Step 1 (ackSeq > prevNextExpected). SACK
	// frames are buffered speculatively in Step 2; delivery only occurs once
	// the cumulative pointer has reached the frame immediately below them.
	if ackSeq > prevNextExpected {
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
	}

	return toDeliver, nil
}

// payloadFor returns the payload for seq, checking reorderBuf then inFlight.
// Returns nil if not found. The returned slice is a fully-owned copy; ARQ
// retains no reference to it after return (satisfies go.md rule 12).
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

	a.inFlight[seq] = &inFlightFrame{
		seq:      seq,
		payload:  p,
		deadline: now.Add(a.dropTimeout),
	}
}

// TLPKTDROP terminates the overdue frame identified by overdueSeq.
//
// Returns ErrSequenceNotInFlight if overdueSeq is not in the retransmit queue.
// Returns ErrFrameNotOverdue if the frame has not exceeded its deadline (AC-005).
//
// On success, sends a DegradationEvent to DegradationEvents (non-blocking;
// the channel must be buffered — see New). The caller must drain
// DegradationEvents on its tick loop. The dropped sequence is also returned
// as a DegradationEvent value for callers that prefer synchronous inspection.
//
// The zero value of DegradationEvent (DroppedSeq == 0) is the "no event"
// sentinel returned on error paths (seq 0 is never a valid in-flight seq).
//
// Only the overdue frame's content is abandoned (BC-2.02.006 postcondition 5).
// Frames below overdueSeq that are still undelivered remain in-flight and are
// deliverable via subsequent OnAck calls.
//
// nextExpected is advanced only when overdueSeq == nextExpected+1 (the dropped
// frame was the next expected frame in sequence). If overdueSeq is ahead of
// nextExpected+1, lower undelivered frames are preserved (C-1 fix).
//
// The deadline check is exclusive: now must be strictly after the deadline
// (now.After(deadline)).
//
// Concurrency: must be called from a single goroutine (the tick loop).
func (a *ARQ) TLPKTDROP(overdueSeq uint32, now time.Time) (DegradationEvent, error) {
	f, ok := a.inFlight[overdueSeq]
	if !ok {
		return DegradationEvent{}, ErrSequenceNotInFlight
	}

	// Deadline is exclusive: the frame must be strictly past its deadline.
	if !now.After(f.deadline) {
		return DegradationEvent{}, ErrFrameNotOverdue
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

	ev := DegradationEvent{DroppedSeq: overdueSeq}

	// Send non-blocking to the buffered DegradationEvents channel. The channel
	// is always buffered (minimum 1; see New) so this select is safe as long as
	// the caller drains DegradationEvents at tick rate.
	select {
	case a.DegradationEvents <- ev:
	default:
		// Channel full — caller is not draining. The return value still carries
		// the event; the metrics layer will observe it on next drain cycle.
	}

	return ev, nil
}

// GapsToRetransmit returns the sequence numbers of in-flight frames that are
// unacknowledged — neither cumulatively ACKed (seq <= ackSeq) nor marked
// received in the SACK bitmap — in ascending order (BC-2.02.005 PC2).
// SACK bitmap covers positions ackSeq+1..ackSeq+64 (bit 0 = MSB of byte 0 = ackSeq+1).
// In-flight frames at seq >= ackSeq+65 (outside the bitmap window) are also gaps.
func (a *ARQ) GapsToRetransmit(ackSeq uint32, sackBitmap [SACKBitmapBytes]byte) []uint32 {
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
// (BC-2.02.005 SACK-accuracy clause; VP-019/VP-020).
//
// Uses math/bits.OnesCount64 via encoding/binary for a one-line body.
func SACKPopCount(bitmap [SACKBitmapBytes]byte) int {
	return bits.OnesCount64(bitmapToUint64(bitmap))
}

// SetNextExpected sets the cumulative delivery pointer directly. This method
// exists to support tests that need to construct ARQ state without driving
// the state machine through a full sequence of ACKs (RULING-003 red-gate tests).
func (a *ARQ) SetNextExpected(seq uint32) {
	a.nextExpected = seq
}

// NextExpected returns the current cumulative delivery pointer. Used by tests
// to assert that OnAck does not mutate state on an out-of-window rejection
// (RULING-003; BC-2.02.005 PC-3).
func (a *ARQ) NextExpected() uint32 {
	return a.nextExpected
}

// InFlightContains reports whether seq is present in the sender's retransmit
// queue. Used by tests to assert that out-of-window rejection leaves inFlight
// unmodified (RULING-003; BC-2.02.005 PC-3).
func (a *ARQ) InFlightContains(seq uint32) bool {
	_, ok := a.inFlight[seq]
	return ok
}

// bitmapToUint64 converts the 8-byte big-endian SACK bitmap to a uint64.
func bitmapToUint64(b [SACKBitmapBytes]byte) uint64 {
	return uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
		uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
}
