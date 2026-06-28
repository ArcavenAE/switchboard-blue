// Package replay implements the upstream idempotent replay window for the
// Switchboard access node receiver. It is pure-core: no goroutines, no timers,
// no I/O. The effectful layer drives OnUpstream() on its own schedule (ARCH-09
// purity boundary).
//
// The receiver deduplicates keystroke frames by sequence number (chan_seq) and
// delivers them in order. Out-of-order frames are buffered until the missing
// predecessor(s) arrive. Frames whose sequence number falls outside the
// configured sliding window are discarded without error.
//
// This file contains stubs only. All non-trivial function bodies are
// not yet implemented — they will be filled in by the implementer.
// Tests calling these functions will fail until the real logic is added.
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
// upstream keystroke frames. It maintains a ring-style seen-set of the last N
// sequence numbers and a pending map for out-of-order arrivals.
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
//
// Not yet implemented: the real constructor body is a stub.
func New(windowSize uint32, deliver DeliverFunc) *Replay {
	if windowSize < 1 {
		panic("replay: New: windowSize must be >= 1")
	}
	if deliver == nil {
		panic("replay: New: deliver must not be nil")
	}
	panic("replay: New: not yet implemented")
}

// OnUpstream processes one incoming upstream frame from the network layer.
//
// Behaviour (not yet implemented — Red Gate):
//   - If frame.Seq has already been delivered (within the current window),
//     returns ErrAlreadyDelivered without calling deliver.
//   - If frame.Seq is older than the window (seq < nextSeq - windowSize),
//     the frame is silently discarded and nil is returned.
//   - If frame.Seq is the next expected sequence number, deliver is called
//     immediately; then any buffered in-order successors are drained.
//   - If frame.Seq is ahead of the next expected sequence number, the frame
//     is held in the pending buffer until its predecessors arrive.
//
// Returns ErrAlreadyDelivered on duplicate delivery; nil in all other cases.
func (r *Replay) OnUpstream(f Frame) error {
	panic("replay: OnUpstream: not yet implemented")
}

// WindowSize returns the configured window size.
func (r *Replay) WindowSize() uint32 {
	panic("replay: WindowSize: not yet implemented")
}

// NextSeq returns the next in-order sequence number the receiver is waiting
// for. Useful for testing and observability.
func (r *Replay) NextSeq() uint32 {
	panic("replay: NextSeq: not yet implemented")
}
