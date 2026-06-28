// Package replay implements the upstream idempotent replay window for the
// Switchboard access node receiver. It is pure-core: no goroutines, no timers,
// no I/O. The effectful layer drives OnUpstream() on its own schedule (ARCH-09
// purity boundary).
//
// The receiver deduplicates keystroke frames by sequence number (chan_seq) and
// delivers them in order. Out-of-order frames are buffered until the missing
// predecessor(s) arrive. Frames whose sequence number falls outside the
// configured sliding window are discarded without error.
package replay

import "errors"

// ErrAlreadyDelivered is returned by OnUpstream when the frame's sequence
// number has already been delivered to the session. Callers MUST treat this
// as a non-fatal, expected condition during replay recovery.
var ErrAlreadyDelivered = errors.New("replay: sequence already delivered")

// Frame is a single upstream keystroke frame as seen by the receiver. The
// payload is opaque to the replay layer (SSH-encrypted end-to-end per
// ARCH-03 §Upstream Idempotent Replay). Only Seq and Payload are inspected.
type Frame struct {
	// Seq is the monotonically increasing channel sequence number assigned by
	// the sender. Sequence numbers start at 1; 0 is treated as unset.
	Seq uint32

	// Payload carries the keystroke content. The replay layer does not decode
	// or inspect the bytes; it passes them to the DeliverFunc verbatim.
	Payload []byte
}

// DeliverFunc is called by Replay to hand a fully ordered, deduplicated frame
// to the consumer (typically the tmux session write path). The consumer owns
// the payload slice after this call returns.
//
// DeliverFunc is called in sequence order: if frames 1, 2, 3 arrive in order
// 1, 3, 2 then DeliverFunc is called as (1), then (2), then (3) once 2
// arrives. The function must not block for extended periods; doing so stalls
// the in-order delivery queue.
type DeliverFunc func(f Frame)

// Replay is a pure-core sliding-window deduplication and reorder buffer for
// upstream keystroke frames. It maintains a seen-set of the last N sequence
// numbers and a pending map for out-of-order arrivals.
//
// Concurrency: Replay is not safe for concurrent use. The effectful scheduling
// layer MUST ensure OnUpstream() is called from a single goroutine or under
// external synchronisation.
type Replay struct {
	windowSize uint32
	deliver    DeliverFunc
	// nextSeq is the next in-order sequence number expected for delivery.
	nextSeq uint32
	// seen tracks which sequence numbers within the current window have been
	// processed. The key is seq; value is a sentinel true when delivered.
	seen map[uint32]bool
	// pending holds out-of-order frames that arrived before their predecessor.
	// Keyed by sequence number.
	pending map[uint32]Frame
}

// New constructs a Replay with the given window size and delivery callback.
// windowSize must be >= 1; deliver must not be nil. Both are preconditions —
// violating them is a programming error and will panic.
func New(windowSize uint32, deliver DeliverFunc) *Replay {
	if windowSize < 1 {
		panic("replay: New: windowSize must be >= 1")
	}
	if deliver == nil {
		panic("replay: New: deliver must not be nil")
	}
	return &Replay{
		windowSize: windowSize,
		deliver:    deliver,
		nextSeq:    1,
		seen:       make(map[uint32]bool),
		pending:    make(map[uint32]Frame),
	}
}

// OnUpstream processes one incoming upstream frame from the network layer.
//
// Frame classification uses a wrap-safe forward distance: dist = seq - nextSeq
// (uint32 modular subtraction). The four cases in delivery order are:
//
//   - If frame.Seq is 0 (the reserved sentinel per RULING-001), the frame is
//     silently discarded and nil is returned. Senders MUST skip 0 on wrap.
//   - If frame.Seq is already in the seen set, returns ErrAlreadyDelivered
//     without calling deliver.
//   - If dist == 0 (seq == nextSeq), deliver is called immediately; then any
//     buffered in-order successors are drained.
//   - If 0 < dist < windowSize, the frame is in the forward window: it is
//     held in the pending buffer until its predecessors arrive.
//   - If dist >= windowSize, the frame is out-of-window and silently discarded
//     (nil returned). This single branch covers both far-future frames (dist is
//     a small integer > windowSize) and past/too-old frames (a sequence number
//     behind nextSeq produces a large modular distance ≈ 2^32 − k, which
//     exceeds any practical windowSize). BC-2.02.004 invariant 3.
//
// Returns ErrAlreadyDelivered on duplicate delivery; nil in all other cases.
func (r *Replay) OnUpstream(f Frame) error {
	seq := f.Seq

	// Seq=0 is the unset/invalid sentinel — discard silently without delivery.
	// Per RULING-001 §R1: chan_seq starts at 1; 0 is reserved and never a valid
	// wire-frame sequence number. Senders MUST skip 0 on wrap (MaxUint32 → 1).
	if seq == 0 {
		return nil
	}

	// Already delivered: seq is in the seen set.
	if r.seen[seq] {
		return ErrAlreadyDelivered
	}

	// Classify seq by its wrap-safe forward distance from nextSeq.
	//
	// uint32 subtraction wraps correctly mod 2^32, so dist = seq - r.nextSeq gives
	// the true forward gap for any uint32 pair:
	//   dist == 0              → in-order: deliver immediately (handled above)
	//   0 < dist < windowSize  → in-window future frame: buffer in pending
	//   dist >= windowSize     → out-of-window: covers BOTH far-future frames
	//                            (dist is a small number > windowSize) AND past/"too-old"
	//                            frames (dist ≈ 2^32 - k, which is >> windowSize).
	//
	// This single check replaces the old separate non-wrap-safe lower-bound guard
	// (r.nextSeq > r.windowSize && seq < r.nextSeq-r.windowSize), which failed near
	// the uint32 boundary because integer-sense past frames with wrapped seq values
	// were evaluated before the distance check could classify them correctly.

	// In-order: deliver immediately and drain any buffered successors.
	if seq == r.nextSeq {
		r.deliverAndDrain(f)
		return nil
	}

	dist := seq - r.nextSeq
	if dist > 0 && dist < r.windowSize {
		// In-window future frame: buffer until predecessor(s) arrive.
		r.pending[seq] = f
		return nil
	}

	// dist >= windowSize: out-of-window (too far ahead or too old). Discard silently.
	// seq < nextSeq and not in seen (evicted after window slide) also lands here.
	return nil
}

// deliverAndDrain delivers the given frame and then drains any pending frames
// that have now become in-order.
func (r *Replay) deliverAndDrain(f Frame) {
	r.deliver(f)
	r.seen[f.Seq] = true
	r.nextSeq++
	// Per RULING-001 §R1: senders skip seq=0 on wrap (MaxUint32 → 1). The receiver
	// mirrors that skip so it never waits for the reserved seq=0 frame.
	if r.nextSeq == 0 {
		r.nextSeq = 1
	}

	// Evict entries that have fallen outside the window.
	r.evictOldSeen()

	// Drain any pending frames that are now in order.
	for {
		next, ok := r.pending[r.nextSeq]
		if !ok {
			break
		}
		delete(r.pending, r.nextSeq)
		r.deliver(next)
		r.seen[next.Seq] = true
		r.nextSeq++
		// Per RULING-001 §R1: skip seq=0 on wrap.
		if r.nextSeq == 0 {
			r.nextSeq = 1
		}
		r.evictOldSeen()
	}
}

// evictOldSeen removes entries from the seen map that are now outside the
// sliding window. This keeps memory bounded at O(windowSize).
func (r *Replay) evictOldSeen() {
	// Window covers [nextSeq-windowSize, nextSeq-1]. Evict anything older.
	if r.nextSeq <= r.windowSize {
		return
	}
	evictBefore := r.nextSeq - r.windowSize
	// We only need to check the one entry that just fell out.
	// Since nextSeq advances by 1 at a time, evictBefore advances by 1 at a time,
	// so we only need to delete evictBefore-1 = nextSeq - windowSize - 1.
	// But to be safe during drain (multiple advances), we clean up anything old.
	// Since we call this after each advance, at most one entry needs eviction.
	delete(r.seen, evictBefore-1)
}

// WindowSize returns the configured window size.
func (r *Replay) WindowSize() uint32 {
	return r.windowSize
}

// NextSeq returns the next in-order sequence number the receiver is waiting
// for. Useful for testing and observability.
func (r *Replay) NextSeq() uint32 {
	return r.nextSeq
}
