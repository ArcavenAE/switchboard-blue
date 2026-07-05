// Package arqsend is the boundary-layer sender that turns ARQ retransmit
// decisions into HMAC-authenticated wire bytes. It closes BC-2.02.005 PC-3
// (on gap detection the access node retransmits the missing content) and
// upholds BC-2.02.005 PC-5 (the retransmit carries the ORIGINAL payload but
// under a NEW frame sequence number — QUIC model).
//
// # Layering (ARCH-09)
//
// arqsend is effectful only in the composition sense: it holds an *arq.ARQ
// handle and it invokes a caller-supplied Dispatch callback with the wire
// bytes. It performs no network I/O, no goroutines, no clocks. The Dispatch
// callback owns wire delivery (multipath.Send is a typical composition
// target); arqsend owns the decision-to-wire seam:
//
//  1. Read the original payload for oldSeq via arq.PayloadForInFlight.
//  2. Compose the wire bytes with outerassembler.Assemble, stamping the
//     channel-header ChanSeq to newSeq.
//  3. Call dispatch(wire). On error, return without mutating ARQ state
//     (no orphan queue entries — the retransmit is re-tryable on the next
//     GapsToRetransmit pass).
//  4. On dispatch success, EnqueueSend(newSeq, payload, now) then
//     RemoveInFlight(oldSeq) — the old queue entry is released now that
//     its content lives under the new seq.
//
// # Concurrency
//
// A single Retransmitter is not safe for concurrent use — it holds an
// *arq.ARQ which is documented pure-core, single-writer. The effectful
// scheduling layer must ensure Retransmit is driven from one goroutine
// (or under external synchronisation) per Retransmitter instance. Two
// Retransmitters over independent ARQ handles are independent and may
// run in parallel.
package arqsend

import (
	"errors"
	"fmt"
	"time"

	"github.com/arcavenae/switchboard/internal/arq"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/halfchannel"
	"github.com/arcavenae/switchboard/internal/outerassembler"
)

// ErrSequenceNotInFlight is returned by Retransmit when oldSeq is not
// present in the ARQ in-flight queue. This is the "nothing to retransmit"
// signal; callers walking a GapsToRetransmit list may treat it as benign
// (another writer already resolved the gap) or as a bug depending on the
// caller's contract with ARQ state.
var ErrSequenceNotInFlight = errors.New("arqsend: oldSeq not in flight")

// Dispatch is the caller-supplied wire-writer. It is called exactly once
// per successful Retransmit with the assembled wire bytes. The Dispatch
// implementation owns the network path; a typical composition invokes
// multipath.Send inside dispatch to fan out on the two fastest paths
// (BC-2.02.001).
//
// The wire slice passed to Dispatch is owned by the caller after the call
// returns — arqsend retains no reference. Dispatch MAY hold the slice.
//
// A non-nil error return causes Retransmit to return that error wrapped
// with context and to leave ARQ state unmodified.
type Dispatch func(wire []byte) error

// Retransmitter composes an *arq.ARQ, an Envelope for HMAC-authenticated
// assembly, and an optional channel-id override into a single seam. It is
// constructed via New and configured with functional options.
type Retransmitter struct {
	arq    *arq.ARQ
	env    outerassembler.Envelope
	chanID uint32
	flags  byte
}

// Option configures a Retransmitter at construction time.
type Option func(*Retransmitter)

// WithChanID sets the channel identifier stamped into the retransmitted
// frame's channel header. Zero (the default) is a legal ChanID; use this
// option when the effectful caller has a non-zero session channel binding.
func WithChanID(chanID uint32) Option {
	return func(r *Retransmitter) { r.chanID = chanID }
}

// WithFlags sets the channel-header flag byte for retransmitted frames.
// The default is zero (no SACK, no FEC, no ARQ_req bit). Callers that
// want to piggy-back a SACK bitmap onto a retransmit MUST set
// outerassembler.FlagSACKPresent and pass the bitmap via WithSACKBitmap
// (not implemented yet — retransmits carry data, not ACKs; a piggy-back
// path is a future extension).
func WithFlags(flags byte) Option {
	return func(r *Retransmitter) { r.flags = flags }
}

// New constructs a Retransmitter with the given ARQ handle and Envelope.
// The Envelope carries the SVTNID + addresses + FrameAuthKey used by
// outerassembler.Assemble; the same key must be registered in the
// receive-side forwarding table for the retransmitted frame to verify.
func New(a *arq.ARQ, env outerassembler.Envelope, opts ...Option) *Retransmitter {
	r := &Retransmitter{
		arq:   a,
		env:   env,
		flags: 0,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Retransmit emits the payload currently held under oldSeq as a fresh
// wire frame stamped with newSeq (BC-2.02.005 PC-3 + PC-5). The wire
// bytes are handed to dispatch; on dispatch success the ARQ state
// transitions atomically-in-effect: newSeq is enqueued under EnqueueSend
// with the caller-supplied deadline anchor `now`, and oldSeq is released.
//
// Preconditions:
//   - oldSeq is in the ARQ in-flight queue (otherwise ErrSequenceNotInFlight).
//   - dispatch is non-nil.
//
// Postconditions (dispatch returned nil):
//   - The wire bytes emitted to dispatch are outerassembler.Assemble output
//     with ChannelHeader.ChanSeq == newSeq (never oldSeq — BC-2.02.005 PC-5).
//   - arq.InFlightContains(oldSeq) == false.
//   - arq.InFlightContains(newSeq) == true.
//   - arq.PayloadForInFlight(newSeq) byte-equals the original payload.
//
// Postconditions (dispatch returned an error):
//   - Returned error wraps the dispatch error via errors.Is.
//   - arq.InFlightContains(oldSeq) == true (state preserved; retryable).
//   - arq.InFlightContains(newSeq) == false (no orphan entry).
func (r *Retransmitter) Retransmit(oldSeq, newSeq uint32, now time.Time, dispatch Dispatch) error {
	payload := r.arq.PayloadForInFlight(oldSeq)
	if payload == nil {
		return fmt.Errorf("oldSeq=%d: %w", oldSeq, ErrSequenceNotInFlight)
	}

	cf := halfchannel.ChannelFrame{
		ChanID:    r.chanID,
		ChanSeq:   newSeq,
		FrameType: frame.FrameTypeData,
		Flags:     r.flags,
		Payload:   payload,
	}

	wire, err := outerassembler.Assemble(cf, [outerassembler.SACKBitmapSize]byte{}, r.env)
	if err != nil {
		return fmt.Errorf("assemble retransmit oldSeq=%d newSeq=%d: %w", oldSeq, newSeq, err)
	}

	if err := dispatch(wire); err != nil {
		// State preserved: the retransmit is re-tryable. Do NOT enqueue
		// newSeq and do NOT release oldSeq (BC-2.02.005 no-orphan-state).
		return fmt.Errorf("dispatch retransmit oldSeq=%d newSeq=%d: %w", oldSeq, newSeq, err)
	}

	// Dispatch succeeded — commit the QUIC-model transition:
	// newSeq inherits the payload, oldSeq is released.
	r.arq.EnqueueSend(newSeq, payload, now)
	r.arq.RemoveInFlight(oldSeq)

	return nil
}
