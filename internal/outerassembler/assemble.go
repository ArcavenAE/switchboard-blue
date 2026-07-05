package outerassembler

import (
	"errors"
	"fmt"

	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/halfchannel"
	"github.com/arcavenae/switchboard/internal/hmac"
)

// ErrPayloadTooLarge is returned by Assemble when
// channel_header_size + len(cf.Payload) would exceed the uint16 limit of
// OuterHeader.PayloadLen (BC-2.01.004 postcondition 2, BC-2.01.002 PC5).
//
// This is a defensive wire-format re-check. halfchannel.Enqueue enforces
// the same bound on the ingest side (BC-2.01.002 PC5, error surfaced as
// halfchannel.ErrPayloadTooLarge), but the assembler cannot rely on
// upstream enforcement: callers can construct ChannelFrame values
// directly (e.g., control-frame emitters, tests). The invariant that
// wire.PayloadLen equals channel_header_size + len(payload) must hold at
// the point of encoding, so we re-check here.
var ErrPayloadTooLarge = errors.New("outerassembler: channel_header_size + payload exceeds uint16 max")

// Envelope carries the outer-header fields the assembler needs beyond
// those already present in a ChannelFrame. Grouping them into a single
// struct prevents src/dst swap and svtn/key mismatch bugs that a
// positional 4-argument function would silently accept.
//
// FrameAuthKey is the per-(node, SVTN) HMAC key derived from
// hmac.DeriveKey — the same [32]byte value routing.ForwardingEntry
// stores. The caller owns key management; the assembler only reads.
type Envelope struct {
	SVTNID       [16]byte
	SrcAddr      [8]byte
	DstAddr      [8]byte
	FrameAuthKey [hmac.KeySize]byte
}

// Assemble composes a ChannelFrame and its Envelope into a single wire
// byte sequence suitable for netingress.ReadFrame consumption + downstream
// routing.RouteFrame authentication.
//
// The output layout, matching ARCH-02 §3 and consumed byte-for-byte by
// the ingress + routing side of the receiver:
//
//	bytes 0..43    outer header (BC-2.01.004; frame.OuterHeaderSize)
//	bytes 44..    channel header (BC-2.01.005; 12 or 20 bytes)
//	bytes ...     application payload (cf.Payload)
//
// PayloadLen (bytes 2-3 of the outer header) equals
// channel_header_size + len(cf.Payload) — the router-visible sizing that
// makes the wire frame self-delimiting for netingress.ReadFrame.
//
// The HMAC tag (bytes 36..43) is computed AFTER the rest of the frame is
// materialised, over (zeroed-tag outer_header || channel_header ||
// payload) with env.FrameAuthKey. This exactly matches the verifier in
// internal/routing/routing.go::verifyFrameHMAC — so a wire frame emitted
// here is verifiable by the routing layer without any further coupling.
//
// The sackBitmap argument is respected only when cf.Flags has
// FlagSACKPresent set; otherwise the bitmap bytes are omitted from the
// wire encoding and the argument is ignored. ChannelFrame does not
// currently carry a SACK bitmap field (halfchannel is pure-core and
// single-writer; upstream ARQ receivers hand the bitmap to the assembler
// out of band), so passing the bitmap as a separate parameter avoids a
// downstream schema change to halfchannel.
//
// Returns ErrPayloadTooLarge (wrapped for context) when the channel
// header + payload would exceed the uint16 limit of PayloadLen.
//
// Assemble is a pure function: same inputs produce the same output; no
// I/O, no clocks, no globals.
func Assemble(cf halfchannel.ChannelFrame, sackBitmap [SACKBitmapSize]byte, env Envelope) ([]byte, error) {
	// Compose the channel header — inject the SACK bitmap only when the
	// flag says so. Flags come from ChannelFrame; sackBitmap comes from
	// the ARQ receiver via the caller.
	chdr := ChannelHeader{
		ChanID:  cf.ChanID,
		ChanSeq: cf.ChanSeq,
		Flags:   cf.Flags,
	}
	if cf.Flags&FlagSACKPresent != 0 {
		chdr.SACKBitmap = sackBitmap
	}
	chdrBytes := EncodeChannelHeader(chdr)

	// Defensive re-check: channel_header_size + len(payload) must fit in
	// uint16. halfchannel.Enqueue enforces this at ingest but the
	// assembler MUST NOT rely on upstream — see ErrPayloadTooLarge doc.
	total := len(chdrBytes) + len(cf.Payload)
	if total > int(^uint16(0)) {
		return nil, fmt.Errorf("channel_header_size=%d + payload=%d > %d: %w",
			len(chdrBytes), len(cf.Payload), int(^uint16(0)), ErrPayloadTooLarge)
	}

	// Build the outer header with HMACTag zeroed for MAC computation
	// (routing.verifyFrameHMAC reconstructs this exact shape by zeroing
	// the wire tag before recomputing).
	hdr := frame.OuterHeader{
		Version:    frame.VersionByte,
		FrameType:  cf.FrameType,
		PayloadLen: uint16(total),
		SVTNID:     env.SVTNID,
		SrcAddr:    env.SrcAddr,
		DstAddr:    env.DstAddr,
		// HMACTag left zero for the initial encode.
	}
	hdrForMAC := frame.EncodeOuterHeader(hdr)

	// Concatenate zeroed-tag header || channel header || payload as the
	// MAC message. This is byte-for-byte the same shape
	// routing.verifyFrameHMAC computes on the receive side, so a
	// receiver that registers env.FrameAuthKey for env.SrcAddr succeeds
	// on verify.
	msg := make([]byte, len(hdrForMAC)+len(chdrBytes)+len(cf.Payload))
	copy(msg, hdrForMAC[:])
	copy(msg[len(hdrForMAC):], chdrBytes)
	copy(msg[len(hdrForMAC)+len(chdrBytes):], cf.Payload)

	tag := hmac.ComputeHMAC(env.FrameAuthKey[:], msg)

	// Inject the tag into the wire bytes at offsets 36..43. We assemble
	// the final wire buffer in-place by copying msg (which already has
	// the correct layout) and overwriting the tag region.
	wire := msg // reuse the backing array; msg is not referenced again
	copy(wire[36:44], tag[:])

	return wire, nil
}
