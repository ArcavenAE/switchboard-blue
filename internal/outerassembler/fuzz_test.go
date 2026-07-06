// Fuzz harnesses for Phase 6 formal hardening.
//
// outerassembler is the wire-frame composition point. Its codec pair
// (EncodeChannelHeader / DecodeChannelHeader) and its Assemble function must
// round-trip byte-exact against netingress.ReadFrame — an attacker who can
// construct a ChannelFrame and Envelope must not be able to cause the
// decoder to panic, mis-size a payload, or accept a truncated header.
//
// This file adds two fuzz targets:
//
//   - FuzzChannelHeaderRoundTrip — DecodeChannelHeader over arbitrary bytes;
//     when decode succeeds, EncodeChannelHeader(decoded) must reproduce the
//     first N bytes of the input exactly (canonical form).
//   - FuzzAssembleReadFrameRoundTrip — Assemble produces bytes that
//     netingress.ReadFrame parses back into an OuterHeader whose SVTNID /
//     addresses match the Envelope, whose FrameType matches the ChannelFrame,
//     and whose PayloadLen equals ChannelHeaderSize(cf.Flags) + len(cf.Payload).
package outerassembler_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/halfchannel"
	"github.com/arcavenae/switchboard/internal/hmac"
	"github.com/arcavenae/switchboard/internal/netingress"
	"github.com/arcavenae/switchboard/internal/outerassembler"
)

// FuzzChannelHeaderRoundTrip asserts the codec invariant:
//
//	decoded, n, err := DecodeChannelHeaderN(b)
//	err == nil ⇒ EncodeChannelHeader(decoded) == b[:n]
//
// This catches drift between encoder and decoder — every value the decoder
// accepts must be a value the encoder can produce with the same layout.
func FuzzChannelHeaderRoundTrip(f *testing.F) {
	// Fixed 12-byte header, all flags zero.
	seed12 := []byte{
		0x11, 0x22, 0x33, 0x44, // chan_id
		0x55, 0x66, 0x77, 0x88, // chan_seq
		0x00,             // flags
		0x00, 0x00, 0x00, // reserved
	}
	f.Add(seed12)

	// 20-byte header, SACK_present=1, bitmap set.
	seed20 := []byte{
		0x00, 0x00, 0x00, 0x01, // chan_id
		0x00, 0x00, 0x00, 0x07, // chan_seq
		outerassembler.FlagSACKPresent, // flags
		0x00, 0x00, 0x00,               // reserved
		0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE, // sack_bitmap
	}
	f.Add(seed20)

	// Reserved bytes non-zero (must reject).
	seedBadReserved := append([]byte{}, seed12...)
	seedBadReserved[10] = 0x01
	f.Add(seedBadReserved)

	// Truncated 20-byte header (SACK_present=1 but only 12 bytes present).
	f.Add(append(seed12[:8], outerassembler.FlagSACKPresent, 0, 0, 0))

	// All zero.
	f.Add(make([]byte, 12))

	// Empty.
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		h, n, err := outerassembler.DecodeChannelHeaderN(data)
		if err != nil {
			// Any error must chain to a documented sentinel.
			if !errors.Is(err, outerassembler.ErrChannelHeaderTruncated) &&
				!errors.Is(err, outerassembler.ErrChannelHeaderReservedNonZero) {
				t.Fatalf("unclassified error from DecodeChannelHeaderN: %v", err)
			}
			return
		}

		// Success: n must be 12 or 20 depending on the SACK flag.
		want := outerassembler.ChannelHeaderSize(h.Flags)
		if n != want {
			t.Fatalf("DecodeChannelHeaderN returned n=%d but ChannelHeaderSize(flags)=%d", n, want)
		}

		// The decoder consumed exactly n bytes; re-encode must match those n bytes.
		reencoded := outerassembler.EncodeChannelHeader(h)
		if !bytes.Equal(reencoded, data[:n]) {
			t.Fatalf("codec round-trip mismatch:\n  input  = %x\n  re-enc = %x", data[:n], reencoded)
		}
	})
}

// FuzzAssembleReadFrameRoundTrip asserts the Assemble → ReadFrame pipeline
// invariant: bytes produced by Assemble parse cleanly through
// netingress.ReadFrame and expose the header/payload contract Assemble was
// asked to produce.
//
// The fuzz corpus is the inputs to Assemble (payload bytes, flag byte,
// per-channel identifiers). Since Assemble takes structured inputs, we
// bound them via the fuzz-supplied byte slice to constrain payload size
// and derive flag/chan values deterministically.
func FuzzAssembleReadFrameRoundTrip(f *testing.F) {
	// Seed: payload + flags encoded into one byte slice.
	f.Add([]byte("hello switchboard"), byte(0), uint32(42), uint32(1), byte(frame.FrameTypeData))
	f.Add([]byte{}, byte(0), uint32(1), uint32(1), byte(frame.FrameTypeEmptyTick))
	f.Add([]byte{0xDE, 0xAD, 0xBE, 0xEF}, byte(outerassembler.FlagSACKPresent), uint32(99), uint32(7), byte(frame.FrameTypeData))
	f.Add([]byte{0x00}, byte(outerassembler.FlagARQReq), uint32(0), uint32(1), byte(frame.FrameTypeArq))

	// Large payload near uint16 boundary.
	f.Add(bytes.Repeat([]byte{0xAB}, 60000), byte(0), uint32(1), uint32(100), byte(frame.FrameTypeData))

	env := outerassembler.Envelope{
		SVTNID:       [16]byte{0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F},
		SrcAddr:      [8]byte{0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27},
		DstAddr:      [8]byte{0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37},
		FrameAuthKey: [hmac.KeySize]byte{0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49, 0x4A, 0x4B, 0x4C, 0x4D, 0x4E, 0x4F, 0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57, 0x58, 0x59, 0x5A, 0x5B, 0x5C, 0x5D, 0x5E, 0x5F},
	}

	f.Fuzz(func(t *testing.T, payload []byte, flags byte, chanID, chanSeq uint32, ftByte byte) {
		// Constrain FrameType to canonical values (parser rejects reserved).
		ft := frame.FrameType(ftByte)
		if !ft.Valid() {
			t.Skip()
		}
		// Empty payload is legal for empty-tick frames but Assemble does not
		// require a specific frame type — just that the total size fits in
		// uint16. Bound the payload here so we do not blow the halfchannel
		// pending queue but do exercise the uint16 boundary condition.
		if len(payload) > halfchannel.MaxPayloadSize {
			t.Skip()
		}

		cf := halfchannel.ChannelFrame{
			ChanID:    chanID,
			ChanSeq:   chanSeq,
			FrameType: ft,
			Flags:     flags,
			Payload:   payload,
		}

		wire, err := outerassembler.Assemble(cf, [outerassembler.SACKBitmapSize]byte{}, env)
		if err != nil {
			// Only legal failure: payload size overflows uint16 minus channel header.
			if !errors.Is(err, outerassembler.ErrPayloadTooLarge) {
				t.Fatalf("unexpected Assemble error: %v", err)
			}
			return
		}

		// Round-trip through netingress.ReadFrame.
		hdr, gotPayload, rerr := netingress.ReadFrame(bytes.NewReader(wire))
		if rerr != nil {
			// The only way this should fail is io.EOF/ErrUnexpectedEOF at
			// stream boundary — which cannot happen when the reader has the
			// full wire slice. Anything else means Assemble emitted an
			// unparseable frame.
			if errors.Is(rerr, io.EOF) || errors.Is(rerr, io.ErrUnexpectedEOF) {
				t.Fatalf("Assemble produced truncated frame (len=%d): %v", len(wire), rerr)
			}
			t.Fatalf("Assemble→ReadFrame round-trip failed: %v", rerr)
		}

		// Header field invariants.
		if hdr.FrameType != ft {
			t.Fatalf("FrameType round-trip: got %v want %v", hdr.FrameType, ft)
		}
		if hdr.SVTNID != env.SVTNID {
			t.Fatalf("SVTNID round-trip mismatch")
		}
		if hdr.SrcAddr != env.SrcAddr {
			t.Fatalf("SrcAddr round-trip mismatch")
		}
		if hdr.DstAddr != env.DstAddr {
			t.Fatalf("DstAddr round-trip mismatch")
		}
		wantPayloadLen := outerassembler.ChannelHeaderSize(flags) + len(payload)
		if int(hdr.PayloadLen) != wantPayloadLen {
			t.Fatalf("PayloadLen: got %d want %d (channel_header_size=%d + payload=%d)",
				hdr.PayloadLen, wantPayloadLen,
				outerassembler.ChannelHeaderSize(flags), len(payload))
		}

		// The payload returned by ReadFrame is [channel_header || payload].
		// Decode the channel header from the start; the remainder must byte-equal
		// cf.Payload.
		chdr, chdrN, cerr := outerassembler.DecodeChannelHeaderN(gotPayload)
		if cerr != nil {
			t.Fatalf("channel-header decode after ReadFrame: %v", cerr)
		}
		if chdr.ChanID != chanID {
			t.Fatalf("channel header ChanID: got %d want %d", chdr.ChanID, chanID)
		}
		if chdr.ChanSeq != chanSeq {
			t.Fatalf("channel header ChanSeq: got %d want %d", chdr.ChanSeq, chanSeq)
		}
		if chdr.Flags != flags {
			t.Fatalf("channel header Flags: got %#x want %#x", chdr.Flags, flags)
		}
		if !bytes.Equal(gotPayload[chdrN:], payload) {
			t.Fatalf("application payload round-trip mismatch:\n  want=%x\n  got =%x", payload, gotPayload[chdrN:])
		}
	})
}
