package outerassembler_test

import (
	"errors"
	"testing"

	"github.com/arcavenae/switchboard/internal/outerassembler"
)

// AC — Channel header 12-byte fixed layout when SACK_present=0
// (BC-2.01.005 postcondition 3; ARCH-02 §3.2). Wave-adv F-004.
func TestEncodeChannelHeader_TwelveBytesWhenSACKAbsent(t *testing.T) {
	t.Parallel()

	h := outerassembler.ChannelHeader{
		ChanID:  0x11223344,
		ChanSeq: 0x55667788,
		Flags:   0x00, // no SACK
	}

	encoded := outerassembler.EncodeChannelHeader(h)

	if len(encoded) != 12 {
		t.Fatalf("len(encoded)=%d, want 12 (BC-2.01.005 fixed layout, SACK_present=0)", len(encoded))
	}
	// chan_id bytes 0..3 big-endian
	if encoded[0] != 0x11 || encoded[1] != 0x22 || encoded[2] != 0x33 || encoded[3] != 0x44 {
		t.Errorf("bytes 0-3 (chan_id) = % x, want 11 22 33 44", encoded[0:4])
	}
	// chan_seq bytes 4..7 big-endian
	if encoded[4] != 0x55 || encoded[5] != 0x66 || encoded[6] != 0x77 || encoded[7] != 0x88 {
		t.Errorf("bytes 4-7 (chan_seq) = % x, want 55 66 77 88", encoded[4:8])
	}
	// flags byte 8
	if encoded[8] != 0x00 {
		t.Errorf("byte 8 (flags) = 0x%02x, want 0x00", encoded[8])
	}
	// reserved bytes 9..11 zero
	if encoded[9] != 0x00 || encoded[10] != 0x00 || encoded[11] != 0x00 {
		t.Errorf("bytes 9-11 (reserved) = % x, want 00 00 00", encoded[9:12])
	}
}

// AC — Channel header 20-byte layout when SACK_present=1 (bit 2 of flags).
func TestEncodeChannelHeader_TwentyBytesWhenSACKPresent(t *testing.T) {
	t.Parallel()

	h := outerassembler.ChannelHeader{
		ChanID:  1,
		ChanSeq: 2,
		Flags:   0x04, // bit 2 = SACK_present
		SACKBitmap: [8]byte{
			0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE,
		},
	}

	encoded := outerassembler.EncodeChannelHeader(h)

	if len(encoded) != 20 {
		t.Fatalf("len(encoded)=%d, want 20 (BC-2.01.005 SACK_present=1 layout)", len(encoded))
	}
	// SACK bitmap bytes 12..19
	for i, want := range h.SACKBitmap {
		if encoded[12+i] != want {
			t.Errorf("byte %d (sack_bitmap[%d]) = 0x%02x, want 0x%02x", 12+i, i, encoded[12+i], want)
		}
	}
}

// AC — Round-trip: decode(encode(h)) == h for every flag combination and
// representative payloads (BC-2.01.005 canonical test vectors row 2).
func TestChannelHeader_RoundTrip_AllFlagCombinations(t *testing.T) {
	t.Parallel()

	sackBitmap := [8]byte{0x01, 0x02, 0x04, 0x08, 0x10, 0x20, 0x40, 0x80}

	// Cover all 3 defined flag bits × two chan_seq boundary values.
	// (Reserved bits stay zero — verified separately by
	// TestDecodeChannelHeader_RejectsNonZeroReserved.)
	for flags := byte(0); flags < 8; flags++ {
		flags := flags
		for _, seq := range []uint32{1, 0xFFFFFFFF} {
			seq := seq
			name := ""
			switch flags {
			case 0:
				name = "no_flags"
			case 1:
				name = "FEC"
			case 2:
				name = "ARQ"
			case 3:
				name = "FEC+ARQ"
			case 4:
				name = "SACK"
			case 5:
				name = "FEC+SACK"
			case 6:
				name = "ARQ+SACK"
			case 7:
				name = "FEC+ARQ+SACK"
			}

			h := outerassembler.ChannelHeader{
				ChanID:  0xAABBCCDD,
				ChanSeq: seq,
				Flags:   flags,
			}
			if flags&outerassembler.FlagSACKPresent != 0 {
				h.SACKBitmap = sackBitmap
			}

			encoded := outerassembler.EncodeChannelHeader(h)

			decoded, err := outerassembler.DecodeChannelHeader(encoded)
			if err != nil {
				t.Fatalf("flags=%s seq=%d: DecodeChannelHeader: %v", name, seq, err)
			}

			if decoded.ChanID != h.ChanID ||
				decoded.ChanSeq != h.ChanSeq ||
				decoded.Flags != h.Flags {
				t.Errorf("flags=%s seq=%d: round-trip mismatch: got %+v, want %+v",
					name, seq, decoded, h)
			}
			// SACKBitmap must match when SACK_present=1; otherwise zero.
			if flags&outerassembler.FlagSACKPresent != 0 {
				if decoded.SACKBitmap != sackBitmap {
					t.Errorf("flags=%s: sack_bitmap = % x, want % x",
						name, decoded.SACKBitmap, sackBitmap)
				}
			} else if decoded.SACKBitmap != ([8]byte{}) {
				t.Errorf("flags=%s: sack_bitmap unexpectedly populated when SACK_present=0", name)
			}
		}
	}
}

// AC — Decode returns ChannelHeaderSize (12 or 20) depending on flag.
// Determinism via wire length is BC-2.01.005 PC-3.
func TestChannelHeaderSize_ReflectsSACKFlag(t *testing.T) {
	t.Parallel()

	if got := outerassembler.ChannelHeaderSize(0); got != 12 {
		t.Errorf("ChannelHeaderSize(flags=0)=%d, want 12", got)
	}
	if got := outerassembler.ChannelHeaderSize(outerassembler.FlagSACKPresent); got != 20 {
		t.Errorf("ChannelHeaderSize(flags=SACK)=%d, want 20", got)
	}
	// Setting other flags without SACK still yields 12.
	if got := outerassembler.ChannelHeaderSize(outerassembler.FlagFECPresent | outerassembler.FlagARQReq); got != 12 {
		t.Errorf("ChannelHeaderSize(FEC|ARQ, no SACK)=%d, want 12", got)
	}
}

// AC — Decode rejects a truncated buffer (BC-2.01.005 EC-002; endpoint
// returns E-PRT-003).
func TestDecodeChannelHeader_RejectsTruncated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		buf  []byte
	}{
		{"empty", nil},
		{"one_byte_short_of_fixed", make([]byte, 11)},
		{"eleven_bytes", make([]byte, 11)},
		{"sack_flag_but_only_twelve_bytes", func() []byte {
			// A 12-byte header whose flags claim SACK_present but the
			// buffer is not extended to 20 bytes.
			h := outerassembler.EncodeChannelHeader(outerassembler.ChannelHeader{
				ChanID:  1,
				ChanSeq: 1,
				Flags:   outerassembler.FlagSACKPresent,
			})
			return h[:12] // truncate the SACK bytes
		}()},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := outerassembler.DecodeChannelHeader(tc.buf)
			if err == nil {
				t.Fatalf("DecodeChannelHeader(%d bytes) returned nil error, want ErrChannelHeaderTruncated", len(tc.buf))
			}
			if !errors.Is(err, outerassembler.ErrChannelHeaderTruncated) {
				t.Errorf("err = %v, want errors.Is(err, ErrChannelHeaderTruncated)", err)
			}
		})
	}
}

// AC — Decode rejects a non-zero reserved field (BC-2.01.005 PC-3 row 3
// requires reserved bytes to be zero).
func TestDecodeChannelHeader_RejectsNonZeroReserved(t *testing.T) {
	t.Parallel()

	h := outerassembler.EncodeChannelHeader(outerassembler.ChannelHeader{
		ChanID:  1,
		ChanSeq: 1,
		Flags:   0,
	})
	// Corrupt reserved bytes.
	h[9] = 0x01

	_, err := outerassembler.DecodeChannelHeader(h[:])
	if err == nil {
		t.Fatalf("expected ErrChannelHeaderReservedNonZero for non-zero reserved byte")
	}
	if !errors.Is(err, outerassembler.ErrChannelHeaderReservedNonZero) {
		t.Errorf("err = %v, want errors.Is(err, ErrChannelHeaderReservedNonZero)", err)
	}
}
