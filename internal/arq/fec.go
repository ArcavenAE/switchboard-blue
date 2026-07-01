// Package arq — FEC extension.
//
// fec.go provides XOR parity FEC for single-loss recovery within an ARQ group
// (BC-2.02.007 v1.2). The Encoder produces one parity frame per group of N
// data frames; the Decoder recovers the single missing frame using XOR across
// the group + parity.
//
// The parity frame carries frame_type=fec=0x05 (internal/frame.FrameTypeFec)
// in its outer header. Do NOT redefine the constant locally — import
// internal/frame and use frame.FrameTypeFec (ARCH-08 position 9: arq→frame
// import is legal; F-P8-008).
//
// ErrTooManyLosses is a package-local sentinel consumed by the ARQ retransmit
// path (AC-004); it has no operator-visible surface and no E-FEC-* taxonomy
// slot (AC-003).
package arq

import (
	"errors"

	"github.com/arcavenae/switchboard/internal/frame"
)

// DefaultFECGroupSize is the default number of data frames per FEC group
// (ADR-002). One parity frame is emitted per complete group of this size.
// Configurable via FECConfig.GroupSize.
const DefaultFECGroupSize = 4

// ErrTooManyLosses is returned by Decoder.Recover when more than one frame
// is missing from the group. XOR parity can only recover a single loss; the
// caller MUST invoke the ARQ SACK/retransmit path on receiving this error
// (BC-2.02.007 PC-4, VP-043; AC-003, AC-004).
//
// This sentinel is package-local: it is consumed internally by the ARQ layer
// and must not be exposed via RPC or operator messaging (AC-003).
var ErrTooManyLosses = errors.New("fec: too many losses in group")

// FECConfig carries construction parameters for NewEncoder and NewDecoder.
type FECConfig struct {
	// GroupSize is the number of data frames per FEC group. Defaults to
	// DefaultFECGroupSize (4) when zero (ADR-002).
	GroupSize int
}

// Encoder accumulates data frame payloads and emits a parity frame for each
// complete group of GroupSize frames. Incomplete last groups (fewer than
// GroupSize frames at session close) do not produce a parity frame
// (BC-2.02.007 EC-001; AC-005).
//
// Zero value is not usable; construct via NewEncoder.
type Encoder struct {
	groupSize int
}

// NewEncoder constructs an Encoder with the given configuration.
func NewEncoder(cfg FECConfig) *Encoder {
	gs := cfg.GroupSize
	if gs <= 0 {
		gs = DefaultFECGroupSize
	}
	return &Encoder{
		groupSize: gs,
	}
}

// AddFrame appends a data frame payload to the current group. When the group
// reaches GroupSize frames it is complete: AddFrame returns the XOR parity
// payload (to be wrapped in a parity frame with
// frame_type=frame.FrameTypeFec=0x05) and resets the internal buffer for the
// next group. When the group is not yet complete, parityPayload is nil.
//
// Callers must set the outer header FrameType to frame.FrameTypeFec when
// transmitting the returned parity payload (AC-001).
//
// EC-001 / AC-005: if the session ends before a full group is collected, the
// caller should discard any partial group; no parity is emitted. The caller
// uses Flush to detect and discard an incomplete group.
func (e *Encoder) AddFrame(payload []byte) (parityPayload []byte) {
	panic("unimplemented")
}

// Flush reports whether the Encoder holds an incomplete last group (fewer than
// GroupSize frames since the last complete group or construction). When true,
// the caller must pass the buffered frames to ARQ for normal handling and must
// NOT emit a parity frame (BC-2.02.007 EC-001; AC-005).
func (e *Encoder) Flush() (incomplete [][]byte, hasIncomplete bool) {
	panic("unimplemented")
}

// GroupSize returns the configured FEC group size.
func (e *Encoder) GroupSize() int {
	return e.groupSize
}

// Decoder recovers a missing data frame from a complete FEC group.
//
// Zero value is not usable; construct via NewDecoder.
type Decoder struct {
	groupSize int
}

// NewDecoder constructs a Decoder with the given configuration.
func NewDecoder(cfg FECConfig) *Decoder {
	gs := cfg.GroupSize
	if gs <= 0 {
		gs = DefaultFECGroupSize
	}
	return &Decoder{
		groupSize: gs,
	}
}

// Recover reconstructs the single missing data frame from a group using XOR
// parity.
//
// group contains the data frame payloads received; missing positions are
// represented by nil entries. parityPayload is the parity frame payload
// extracted from the frame with frame_type=frame.FrameTypeFec=0x05.
//
// Returns the recovered payload when exactly one entry in group is nil
// (AC-002).
//
// Returns ErrTooManyLosses when more than one entry in group is nil — XOR
// parity cannot recover multiple losses. The caller MUST NOT drop the frame
// group silently on ErrTooManyLosses; it MUST invoke the ARQ SACK/retransmit
// path (AC-003, AC-004; BC-2.02.007 PC-4).
//
// The parity frame type constant is frame.FrameTypeFec (=0x05); this reference
// ensures the import is live and the canonical value is used throughout.
func (d *Decoder) Recover(group [][]byte, parityPayload []byte) ([]byte, error) {
	// Reference frame.FrameTypeFec to satisfy the import requirement (ARCH-08;
	// F-P8-008): the parity frame outer header carries this frame type.
	_ = frame.FrameTypeFec
	panic("unimplemented")
}
