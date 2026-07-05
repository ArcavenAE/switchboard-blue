// Package outerassembler composes ChannelFrame + OuterHeader values into
// on-the-wire byte sequences and their inverse decode (Story S-BL.OA).
//
// # Scope
//
// This package handles two responsibilities the effectful send-path needs
// but that must remain testable as pure functions:
//
//  1. Channel-header wire-format codec (BC-2.01.005; ARCH-02 §3.2 —
//     12/20-byte layout including the conditional 8-byte SACK bitmap).
//  2. Composition: bind a ChannelFrame (from internal/halfchannel.Tick)
//     to an OuterHeader (from internal/frame), compute the outer-header
//     HMAC over the zeroed-tag header || channel_header || payload
//     concatenation, and emit the final wire bytes suitable for
//     netingress.ReadFrame → routing.RouteFrame consumption.
//
// # Out of scope (still anchored to future stories)
//
//   - Egress transport (TCP/UDP dial, framing on the wire).
//   - ARQ retransmit TX-side buffering (S-4.03 / S-BL.ARQ-TX).
//   - FEC group assembly (BC-2.01.005 EC-004; future story).
//   - Discovery wire format (internal/discovery).
//   - Live daemon send-path wiring (cmd/switchboard).
//   - ADR-005 RESYNC frame emission and reconnect state machine.
//     Note: STATE.md line 89 re-anchors "ADR-005 resync wire-mechanics"
//     to S-BL.OA, but the RESYNC frame is a control-frame type with its
//     own state machine (RESYNC emission, reconnect trigger, replay from
//     last_acked_seq+1) that is orthogonal to mechanical channel-header
//     serialization. This story delivers the wire-mechanics primitive
//     (encode/decode a channel header, compose it into a wire frame,
//     compute the HMAC that satisfies routing.RouteFrame) that a future
//     RESYNC frame will use. See DELIVERY.md "ADR-005 resync
//     disposition" for the reasoning.
//
// # Classification (ARCH-09)
//
// pure-core: all exported functions are deterministic transformations
// over inputs. No I/O, no globals, no goroutines, no clocks.
//
// # Import constraints (ARCH-08 §6)
//
// Position 8 (append after tmux in the topological order). Imports:
//
//	{frame, hmac, halfchannel}
//
// Justification: this package is the composition point where a
// ChannelFrame (halfchannel) is bound to an OuterHeader (frame) and
// authenticated with the frame_auth_key (hmac). It has no downstream
// state; consumers pass it structs and receive bytes. It does NOT
// import admission, routing, session, or any effectful package —
// composition happens above and calls into here.
package outerassembler

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// ChannelHeaderFixedSize is the fixed-layout portion of the channel header
// (BC-2.01.005 PC-3): chan_id(4) + chan_seq(4) + flags(1) + reserved(3) = 12.
const ChannelHeaderFixedSize = 12

// SACKBitmapSize is the byte length of the conditional SACK bitmap
// (BC-2.01.005 PC-3 row 5): 8 bytes covering 64 sequence slots. Present
// only when the SACK_present flag bit is set.
const SACKBitmapSize = 8

// ChannelHeaderWithSACKSize is the wire size of the channel header when
// the SACK_present flag is set.
const ChannelHeaderWithSACKSize = ChannelHeaderFixedSize + SACKBitmapSize

// Channel-header flag bits (ARCH-02 §3.2 / BC-2.01.005 PC-3 row 3).
const (
	FlagFECPresent  byte = 1 << 0 // bit 0
	FlagARQReq      byte = 1 << 1 // bit 1
	FlagSACKPresent byte = 1 << 2 // bit 2
)

// ErrChannelHeaderTruncated is returned by DecodeChannelHeader when the
// input slice is shorter than the layout demands (12 bytes fixed, or 20
// when SACK_present=1). Traces to E-PRT-003 / BC-2.01.005 EC-002.
var ErrChannelHeaderTruncated = errors.New("outerassembler: channel header truncated")

// ErrChannelHeaderReservedNonZero is returned by DecodeChannelHeader when
// any of the 3 reserved bytes (offsets 9..11) is not zero. Enforces
// BC-2.01.005 PC-3 row 4 ("reserved: must be zero").
var ErrChannelHeaderReservedNonZero = errors.New("outerassembler: channel header reserved bytes must be zero")

// ChannelHeader is the endpoint-visible metadata block that follows the
// 44-byte outer header in every frame (BC-2.01.005; ARCH-02 §3.2).
//
// Routers never parse this struct — they forward the whole payload region
// (channel header + application payload) opaquely per BC-2.01.005 PC-2.
// The type exists for endpoint composition/parsing only.
type ChannelHeader struct {
	// ChanID identifies the half-channel (upstream vs downstream).
	// (BC-2.01.005 PC-3 row 1.)
	ChanID uint32
	// ChanSeq is the per-half-channel sequence number, +1 per tick
	// (BC-2.01.005 PC-3 row 2; RULING-001: starts at 1, skips 0 on wrap).
	ChanSeq uint32
	// Flags is a bitfield: bit 0 = FEC_present, bit 1 = ARQ_req,
	// bit 2 = SACK_present (BC-2.01.005 PC-3 row 3).
	Flags byte
	// SACKBitmap is the conditional 8-byte SACK acknowledgement bitmap
	// (BC-2.01.005 PC-3 row 5). Present in the wire encoding only when
	// (Flags & FlagSACKPresent) != 0; otherwise the field is ignored on
	// encode and left zero on decode.
	SACKBitmap [SACKBitmapSize]byte
}

// ChannelHeaderSize returns the on-wire byte length of a channel header
// with the given flags: 12 for the fixed layout, 20 when SACK_present=1.
// Pure function, safe for concurrent use.
func ChannelHeaderSize(flags byte) int {
	if flags&FlagSACKPresent != 0 {
		return ChannelHeaderWithSACKSize
	}
	return ChannelHeaderFixedSize
}

// EncodeChannelHeader serialises h into 12 or 20 bytes in big-endian wire
// order per BC-2.01.005 PC-3:
//
//	offset 0..3    chan_id      u32 big-endian
//	offset 4..7    chan_seq     u32 big-endian
//	offset 8       flags        u8  (bit 0 FEC / bit 1 ARQ / bit 2 SACK)
//	offset 9..11   reserved     3 zero bytes
//	offset 12..19  sack_bitmap  8 bytes, present iff Flags&FlagSACKPresent
//
// The returned slice is a freshly allocated backing array; callers own it.
func EncodeChannelHeader(h ChannelHeader) []byte {
	size := ChannelHeaderSize(h.Flags)
	b := make([]byte, size)
	binary.BigEndian.PutUint32(b[0:4], h.ChanID)
	binary.BigEndian.PutUint32(b[4:8], h.ChanSeq)
	b[8] = h.Flags
	// b[9..11] left zero (reserved).
	if size == ChannelHeaderWithSACKSize {
		copy(b[12:20], h.SACKBitmap[:])
	}
	return b
}

// DecodeChannelHeader parses a channel header from the start of b.
//
// Returns:
//   - the parsed ChannelHeader,
//   - the number of bytes consumed (12 or 20 depending on flags),
//   - a non-nil error wrapping ErrChannelHeaderTruncated when b is
//     shorter than the layout demands, or ErrChannelHeaderReservedNonZero
//     when any of the 3 reserved bytes at offsets 9..11 is non-zero
//     (BC-2.01.005 PC-3 row 4).
//
// Extra bytes past the channel-header region are left in b for the caller
// (the application payload follows the channel header in wire order).
func DecodeChannelHeader(b []byte) (ChannelHeader, error) {
	h, _, err := DecodeChannelHeaderN(b)
	return h, err
}

// DecodeChannelHeaderN is like DecodeChannelHeader but also returns the
// number of bytes consumed, so callers reading from a longer buffer can
// slice out the payload region without recomputing the header size.
func DecodeChannelHeaderN(b []byte) (ChannelHeader, int, error) {
	if len(b) < ChannelHeaderFixedSize {
		return ChannelHeader{}, 0, fmt.Errorf("channel header truncated: expected %d bytes, got %d: %w",
			ChannelHeaderFixedSize, len(b), ErrChannelHeaderTruncated)
	}
	// Reserved bytes must be zero (BC-2.01.005 PC-3 row 4).
	if b[9] != 0 || b[10] != 0 || b[11] != 0 {
		return ChannelHeader{}, 0, fmt.Errorf("reserved bytes = 0x%02x 0x%02x 0x%02x: %w",
			b[9], b[10], b[11], ErrChannelHeaderReservedNonZero)
	}
	var h ChannelHeader
	h.ChanID = binary.BigEndian.Uint32(b[0:4])
	h.ChanSeq = binary.BigEndian.Uint32(b[4:8])
	h.Flags = b[8]
	size := ChannelHeaderSize(h.Flags)
	if len(b) < size {
		return ChannelHeader{}, 0, fmt.Errorf("channel header truncated (SACK_present=1): expected %d bytes, got %d: %w",
			size, len(b), ErrChannelHeaderTruncated)
	}
	if size == ChannelHeaderWithSACKSize {
		copy(h.SACKBitmap[:], b[12:20])
	}
	return h, size, nil
}
