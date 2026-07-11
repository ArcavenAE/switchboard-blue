package frame_test

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/arcavenae/switchboard/internal/frame"
)

// AC-001 / BC-2.01.004 — TestEncodeOuterHeader_WireFormatByteOffsets
// Verifies that EncodeOuterHeader places each field at the correct wire-format
// byte offset and that payload_len is encoded big-endian (256 → 0x01 0x00 at
// bytes 2-3). A little-endian regression would fail this test.
func TestEncodeOuterHeader_WireFormatByteOffsets(t *testing.T) {
	t.Parallel()

	h := frame.OuterHeader{
		Version:    frame.VersionByte,   // 0x01 (major=0, minor=1)
		FrameType:  frame.FrameTypeData, // 0x01
		PayloadLen: 256,                 // big-endian: bytes [01, 00]
		SVTNID:     [16]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00},
		SrcAddr:    [8]byte{0xA1, 0xA2, 0xA3, 0xA4, 0xA5, 0xA6, 0xA7, 0xA8},
		DstAddr:    [8]byte{0xB1, 0xB2, 0xB3, 0xB4, 0xB5, 0xB6, 0xB7, 0xB8},
		HMACTag:    [8]byte{0xC1, 0xC2, 0xC3, 0xC4, 0xC5, 0xC6, 0xC7, 0xC8},
	}

	encoded := frame.EncodeOuterHeader(h)

	// byte 0 = version
	if encoded[0] != 0x01 {
		t.Errorf("byte 0 (version) = 0x%02x, want 0x01", encoded[0])
	}
	// byte 1 = frame_type
	if encoded[1] != 0x01 {
		t.Errorf("byte 1 (frame_type) = 0x%02x, want 0x01 (FrameTypeData)", encoded[1])
	}
	// AC-001 / F-004: bytes 2-3 = payload_len BIG-ENDIAN (256 → 0x01, 0x00)
	if encoded[2] != 0x01 || encoded[3] != 0x00 {
		t.Errorf("bytes 2-3 (payload_len big-endian for 256) = 0x%02x 0x%02x, want 0x01 0x00", encoded[2], encoded[3])
	}
	// bytes 4-19 = svtn_id (16 bytes)
	for i := 0; i < 16; i++ {
		if encoded[4+i] != h.SVTNID[i] {
			t.Errorf("byte %d (svtn_id[%d]) = 0x%02x, want 0x%02x", 4+i, i, encoded[4+i], h.SVTNID[i])
		}
	}
	// bytes 20-27 = src_addr (8 bytes)
	for i := 0; i < 8; i++ {
		if encoded[20+i] != h.SrcAddr[i] {
			t.Errorf("byte %d (src_addr[%d]) = 0x%02x, want 0x%02x", 20+i, i, encoded[20+i], h.SrcAddr[i])
		}
	}
	// bytes 28-35 = dst_addr (8 bytes)
	for i := 0; i < 8; i++ {
		if encoded[28+i] != h.DstAddr[i] {
			t.Errorf("byte %d (dst_addr[%d]) = 0x%02x, want 0x%02x", 28+i, i, encoded[28+i], h.DstAddr[i])
		}
	}
	// bytes 36-43 = hmac_tag (8 bytes)
	for i := 0; i < 8; i++ {
		if encoded[36+i] != h.HMACTag[i] {
			t.Errorf("byte %d (hmac_tag[%d]) = 0x%02x, want 0x%02x", 36+i, i, encoded[36+i], h.HMACTag[i])
		}
	}
}

// TestEncodeOuterHeader_ExactlyFortyFourBytes is the AC-001-named test per
// S-1.01 story spec line 52. It asserts the size invariant in isolation
// (VP-003): if a future refactor changes the return type from [44]byte to
// a slice, this test must still observe the 44-byte requirement directly.
// Pairs with TestEncodeOuterHeader_WireFormatByteOffsets which verifies
// per-byte field offsets.
func TestEncodeOuterHeader_ExactlyFortyFourBytes(t *testing.T) {
	t.Parallel()

	h := frame.OuterHeader{
		Version:    frame.VersionByte,
		FrameType:  frame.FrameTypeData,
		PayloadLen: 0,
	}

	encoded := frame.EncodeOuterHeader(h)

	// VP-003 size invariant: encoded MUST be exactly 44 bytes.
	// Indexing into encoded[:] makes the value "used" per SA4006 and
	// exercises the slice length path (which differs from compile-time len
	// on a fixed array).
	if got := len(encoded[:]); got != 44 {
		t.Errorf("EncodeOuterHeader returned %d bytes, want 44", got)
	}
	// Pin OuterHeaderSize to the wire-format literal.
	if frame.OuterHeaderSize != 44 {
		t.Errorf("OuterHeaderSize = %d, want 44 per BC-2.01.004", frame.OuterHeaderSize)
	}
}

// AC-002 — TestParseEncodeRoundTrip
// Traces to BC-2.01.004 postcondition 2 (round-trip identity).
func TestParseEncodeRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		h    frame.OuterHeader
	}{
		{
			name: "data frame v0.1",
			h: frame.OuterHeader{
				Version:    frame.VersionByte,
				FrameType:  frame.FrameTypeData,
				PayloadLen: 0,
				SVTNID:     [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
				SrcAddr:    [8]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11},
				DstAddr:    [8]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88},
				HMACTag:    [8]byte{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE},
			},
		},
		{
			name: "empty tick frame with non-zero payload_len",
			h: frame.OuterHeader{
				Version:    frame.VersionByte,
				FrameType:  frame.FrameTypeEmptyTick,
				PayloadLen: 65535,
				SVTNID:     [16]byte{0xFF, 0xFE, 0xFD, 0xFC, 0xFB, 0xFA, 0xF9, 0xF8, 0xF7, 0xF6, 0xF5, 0xF4, 0xF3, 0xF2, 0xF1, 0xF0},
				SrcAddr:    [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
				DstAddr:    [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
				HMACTag:    [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			},
		},
		{
			name: "ctl frame all-zero svtn_id (EC-002)",
			h: frame.OuterHeader{
				Version:    frame.VersionByte,
				FrameType:  frame.FrameTypeCtl,
				PayloadLen: 100,
				SVTNID:     [16]byte{},
				SrcAddr:    [8]byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0},
				DstAddr:    [8]byte{0xF0, 0xDE, 0xBC, 0x9A, 0x78, 0x56, 0x34, 0x12},
				HMACTag:    [8]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88},
			},
		},
		{
			name: "arq frame src equals dst (EC-003)",
			h: frame.OuterHeader{
				Version:    frame.VersionByte,
				FrameType:  frame.FrameTypeArq,
				PayloadLen: 512,
				SVTNID:     [16]byte{0xCA, 0xFE, 0xBA, 0xBE, 0xDE, 0xAD, 0xBE, 0xEF, 0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF},
				SrcAddr:    [8]byte{0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55},
				DstAddr:    [8]byte{0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55},
				HMACTag:    [8]byte{0xFF, 0xEE, 0xDD, 0xCC, 0xBB, 0xAA, 0x99, 0x88},
			},
		},
		{
			name: "fec frame all-ones hmac_tag (max-value boundary)",
			h: frame.OuterHeader{
				Version:    frame.VersionByte,
				FrameType:  frame.FrameTypeFec,
				PayloadLen: 1024,
				SVTNID:     [16]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
				SrcAddr:    [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
				DstAddr:    [8]byte{0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01},
				HMACTag:    [8]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			},
		},
		{
			name: "data frame all-zero hmac_tag (EC-004)",
			h: frame.OuterHeader{
				Version:    frame.VersionByte,
				FrameType:  frame.FrameTypeData,
				PayloadLen: 128,
				SVTNID:     [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
				SrcAddr:    [8]byte{0xAB, 0xCD, 0xEF, 0x01, 0x23, 0x45, 0x67, 0x89},
				DstAddr:    [8]byte{0x98, 0x76, 0x54, 0x32, 0x10, 0xFE, 0xDC, 0xBA},
				HMACTag:    [8]byte{},
			},
		},
		{
			name: "data frame payload_len=1",
			h: frame.OuterHeader{
				Version:    frame.VersionByte,
				FrameType:  frame.FrameTypeData,
				PayloadLen: 1,
				SVTNID:     [16]byte{0x10, 0x20, 0x30, 0x40, 0x50, 0x60, 0x70, 0x80, 0x90, 0xA0, 0xB0, 0xC0, 0xD0, 0xE0, 0xF0, 0x00},
				SrcAddr:    [8]byte{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
				DstAddr:    [8]byte{0xBB, 0xBB, 0xBB, 0xBB, 0xBB, 0xBB, 0xBB, 0xBB},
				HMACTag:    [8]byte{0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC},
			},
		},
		{
			name: "ctl frame minor version boundary",
			h: frame.OuterHeader{
				Version:    0x0F, // major=0, minor=15 — still valid (minor differences non-blocking)
				FrameType:  frame.FrameTypeCtl,
				PayloadLen: 200,
				SVTNID:     [16]byte{0xAB, 0xCD, 0xEF, 0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF, 0x01, 0x23, 0x45, 0x67, 0x89},
				SrcAddr:    [8]byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01},
				DstAddr:    [8]byte{0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02},
				HMACTag:    [8]byte{0x03, 0x03, 0x03, 0x03, 0x03, 0x03, 0x03, 0x03},
			},
		},
		{
			name: "fec frame max payload_len boundary",
			h: frame.OuterHeader{
				Version:    frame.VersionByte,
				FrameType:  frame.FrameTypeFec,
				PayloadLen: 32768,
				SVTNID:     [16]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F},
				SrcAddr:    [8]byte{0xFF, 0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF, 0x00},
				DstAddr:    [8]byte{0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF},
				HMACTag:    [8]byte{0x55, 0xAA, 0x55, 0xAA, 0x55, 0xAA, 0x55, 0xAA},
			},
		},
		{
			name: "arq frame distinctive bit patterns",
			h: frame.OuterHeader{
				Version:    frame.VersionByte,
				FrameType:  frame.FrameTypeArq,
				PayloadLen: 9999,
				SVTNID:     [16]byte{0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
				SrcAddr:    [8]byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0},
				DstAddr:    [8]byte{0xF1, 0xE2, 0xD3, 0xC4, 0xB5, 0xA6, 0x97, 0x88},
				HMACTag:    [8]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77},
			},
		},
		{
			name: "data frame all fields distinguished",
			h: frame.OuterHeader{
				Version:    frame.VersionByte,
				FrameType:  frame.FrameTypeData,
				PayloadLen: 60000,
				SVTNID:     [16]byte{0xDE, 0xAD, 0xC0, 0xDE, 0xBE, 0xEF, 0xCA, 0xFE, 0xDE, 0xAD, 0xC0, 0xDE, 0xBE, 0xEF, 0xCA, 0xFE},
				SrcAddr:    [8]byte{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11},
				DstAddr:    [8]byte{0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22},
				HMACTag:    [8]byte{0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			encoded := frame.EncodeOuterHeader(tc.h)
			decoded, err := frame.ParseOuterHeader(encoded[:])
			if err != nil {
				t.Fatalf("ParseOuterHeader returned unexpected error: %v", err)
			}
			if decoded != tc.h {
				t.Errorf("round-trip mismatch\ngot:  %+v\nwant: %+v", decoded, tc.h)
			}
		})
	}
}

// AC-003 — TestParseOuterHeader_TooShort
// Traces to BC-2.01.004 precondition 1 (ErrFrameTooShort / E-PRT-002).
func TestParseOuterHeader_TooShort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		size int
	}{
		{"zero bytes", 0},
		{"one byte", 1},
		{"half size", 22},
		{"one short", 43},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := frame.ParseOuterHeader(make([]byte, tc.size))
			if !errors.Is(err, frame.ErrFrameTooShort) {
				t.Errorf("ParseOuterHeader(%d bytes) = %v, want ErrFrameTooShort", tc.size, err)
			}
			// F-005 contract: error must be WRAPPED (not the bare sentinel)
			// so callers can attach context via %w without losing the sentinel match.
			if err == frame.ErrFrameTooShort {
				t.Errorf("ParseOuterHeader(%d bytes) returned bare sentinel, want wrapped error", tc.size)
			}
		})
	}
}

// AC-004 — TestParseOuterHeader_VersionMismatch
// Traces to BC-2.01.004 precondition 2 (ErrVersionMismatch / E-PRT-001).
// Major version nibble (bits [7:4]) is non-zero → ErrVersionMismatch.
// Minor version differences (major nibble == 0) must NOT return ErrVersionMismatch.
func TestParseOuterHeader_VersionMismatch(t *testing.T) {
	t.Parallel()

	makeFrame := func(versionByte byte) []byte {
		b := make([]byte, frame.OuterHeaderSize)
		b[0] = versionByte
		b[1] = byte(frame.FrameTypeData)
		binary.BigEndian.PutUint16(b[2:4], 0)
		return b
	}

	t.Run("major=1 (0x10) returns ErrVersionMismatch", func(t *testing.T) {
		t.Parallel()
		_, err := frame.ParseOuterHeader(makeFrame(0x10))
		if !errors.Is(err, frame.ErrVersionMismatch) {
			t.Errorf("ParseOuterHeader(version=0x10) = %v, want ErrVersionMismatch", err)
		}
		// F-005 contract: error must be WRAPPED (not the bare sentinel)
		// so callers can attach context via %w without losing the sentinel match.
		if err == frame.ErrVersionMismatch {
			t.Errorf("ParseOuterHeader(version=0x10) returned bare sentinel, want wrapped error")
		}
	})

	t.Run("major=15 (0xF0) returns ErrVersionMismatch", func(t *testing.T) {
		t.Parallel()
		_, err := frame.ParseOuterHeader(makeFrame(0xF0))
		if !errors.Is(err, frame.ErrVersionMismatch) {
			t.Errorf("ParseOuterHeader(version=0xF0) = %v, want ErrVersionMismatch", err)
		}
		// F-005 contract: error must be WRAPPED (not the bare sentinel)
		// so callers can attach context via %w without losing the sentinel match.
		if err == frame.ErrVersionMismatch {
			t.Errorf("ParseOuterHeader(version=0xF0) returned bare sentinel, want wrapped error")
		}
	})

	t.Run("major=0 minor=5 (0x05) does NOT return ErrVersionMismatch", func(t *testing.T) {
		t.Parallel()
		_, err := frame.ParseOuterHeader(makeFrame(0x05))
		if errors.Is(err, frame.ErrVersionMismatch) {
			t.Errorf("ParseOuterHeader(version=0x05) returned ErrVersionMismatch, but minor-only differences must not trigger it")
		}
	})
}

// AC-005 — TestChannelHeaderOpaque_NotInOuterHeader
// Traces to BC-2.01.005 invariant 1: OuterHeader must contain exactly the 7
// fields specified in ARCH-02 §Outer Header Format — no channel-header fields.
func TestChannelHeaderOpaque_NotInOuterHeader(t *testing.T) {
	t.Parallel()

	want := map[string]bool{
		"Version":    true,
		"FrameType":  true,
		"PayloadLen": true,
		"SVTNID":     true,
		"SrcAddr":    true,
		"DstAddr":    true,
		"HMACTag":    true,
	}

	typ := reflect.TypeOf(frame.OuterHeader{})
	got := make(map[string]bool, typ.NumField())
	for i := range typ.NumField() {
		got[typ.Field(i).Name] = true
	}

	// Check for unexpected fields (channel-header leakage).
	for name := range got {
		if !want[name] {
			t.Errorf("OuterHeader has unexpected field %q — channel-header fields must not appear in OuterHeader (BC-2.01.005 invariant 1)", name)
		}
	}

	// Check that all required fields are present.
	for name := range want {
		if !got[name] {
			t.Errorf("OuterHeader missing required field %q", name)
		}
	}

	// Assert exact count.
	if len(got) != len(want) {
		t.Errorf("OuterHeader has %d fields, want exactly %d", len(got), len(want))
	}
}

// FuzzEncodeParseRoundTrip is a structural fuzz harness covering VP-001/002/003.
// Seeds are valid 44-byte outer header frames.
// The fuzz function checks parse(encode(parse(b))) == parse(b) for any 44-byte
// input that parses without error.
func FuzzEncodeParseRoundTrip(f *testing.F) {
	// Seed 1: canonical v0.1 data frame
	seed1 := func() []byte {
		h := frame.OuterHeader{
			Version:    frame.VersionByte,
			FrameType:  frame.FrameTypeData,
			PayloadLen: 256,
			SVTNID:     [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			SrcAddr:    [8]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11},
			DstAddr:    [8]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88},
			HMACTag:    [8]byte{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE},
		}
		enc := frame.EncodeOuterHeader(h)
		return enc[:]
	}

	// Seed 2: empty-tick frame payload_len=0
	seed2 := func() []byte {
		h := frame.OuterHeader{
			Version:    frame.VersionByte,
			FrameType:  frame.FrameTypeEmptyTick,
			PayloadLen: 0,
			SVTNID:     [16]byte{0xFF, 0xFE, 0xFD, 0xFC, 0xFB, 0xFA, 0xF9, 0xF8, 0xF7, 0xF6, 0xF5, 0xF4, 0xF3, 0xF2, 0xF1, 0xF0},
			SrcAddr:    [8]byte{},
			DstAddr:    [8]byte{},
			HMACTag:    [8]byte{},
		}
		enc := frame.EncodeOuterHeader(h)
		return enc[:]
	}

	// Seed 3: ctl frame all-max fields
	seed3 := func() []byte {
		h := frame.OuterHeader{
			Version:    frame.VersionByte,
			FrameType:  frame.FrameTypeCtl,
			PayloadLen: 65535,
			SVTNID:     [16]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			SrcAddr:    [8]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			DstAddr:    [8]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			HMACTag:    [8]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		}
		enc := frame.EncodeOuterHeader(h)
		return enc[:]
	}

	// Seed 4: arq frame with minor version 5 (major=0 must not fail version check)
	seed4 := func() []byte {
		h := frame.OuterHeader{
			Version:    0x05, // major=0, minor=5
			FrameType:  frame.FrameTypeArq,
			PayloadLen: 1234,
			SVTNID:     [16]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
			SrcAddr:    [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			DstAddr:    [8]byte{0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01},
			HMACTag:    [8]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11},
		}
		enc := frame.EncodeOuterHeader(h)
		return enc[:]
	}

	// Seed 5: fec frame
	seed5 := func() []byte {
		h := frame.OuterHeader{
			Version:    frame.VersionByte,
			FrameType:  frame.FrameTypeFec,
			PayloadLen: 8192,
			SVTNID:     [16]byte{0xCA, 0xFE, 0xBA, 0xBE, 0x00, 0x00, 0x00, 0x00, 0xDE, 0xAD, 0xBE, 0xEF, 0x00, 0x00, 0x00, 0x00},
			SrcAddr:    [8]byte{0x55, 0xAA, 0x55, 0xAA, 0x55, 0xAA, 0x55, 0xAA},
			DstAddr:    [8]byte{0xAA, 0x55, 0xAA, 0x55, 0xAA, 0x55, 0xAA, 0x55},
			HMACTag:    [8]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF},
		}
		enc := frame.EncodeOuterHeader(h)
		return enc[:]
	}

	f.Add(seed1())
	f.Add(seed2())
	f.Add(seed3())
	f.Add(seed4())
	f.Add(seed5())
	f.Add(make([]byte, frame.OuterHeaderSize))

	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) != frame.OuterHeaderSize {
			return
		}
		h, err := frame.ParseOuterHeader(b)

		// VP-002: any non-zero major nibble must trigger ErrVersionMismatch.
		if (b[0]>>4)&0x0F != 0 {
			if !errors.Is(err, frame.ErrVersionMismatch) {
				t.Fatalf("VP-002 violation: major nibble=%d non-zero but err=%v, want errors.Is(ErrVersionMismatch)", (b[0]>>4)&0x0F, err)
			}
			return
		}

		// VP-001 round-trip identity for accepted frames.
		if err != nil {
			return
		}
		encoded := frame.EncodeOuterHeader(h)
		// Every encoded byte must match the original input byte-for-byte
		// (round-trip identity; VP-001/003).
		for i := range frame.OuterHeaderSize {
			if encoded[i] != b[i] {
				t.Errorf("round-trip mismatch at byte %d: got 0x%02x, want 0x%02x", i, encoded[i], b[i])
			}
		}
	})
}

// TestFrameType_Valid asserts FrameType.Valid() returns true exactly for the
// six canonical enum values (ARCH-02 §3.1) and false otherwise.
//
// Red Gate: references frame.FrameType (named type) and (FrameType).Valid()
// which do not exist until the implementer's commit lands.
func TestFrameType_Valid(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		ft   frame.FrameType
		want bool
	}{
		{"data", frame.FrameTypeData, true},
		{"empty_tick", frame.FrameTypeEmptyTick, true},
		{"ctl", frame.FrameTypeCtl, true},
		{"arq", frame.FrameTypeArq, true},
		{"fec", frame.FrameTypeFec, true},
		{"pe_connect", frame.FrameTypePEConnect, true},
		{"zero", frame.FrameType(0x00), false},
		{"just_above_max", frame.FrameType(0x07), false},
		{"max_byte", frame.FrameType(0xFF), false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.ft.Valid(); got != tc.want {
				t.Errorf("FrameType(%#x).Valid() = %v, want %v", byte(tc.ft), got, tc.want)
			}
		})
	}
}

// TestParseOuterHeader_RejectsInvalidFrameType asserts that a 44-byte buffer
// with an out-of-range frame_type byte returns an error wrapping
// frame.ErrInvalidFrameType per errors.Is (ARCH-02 §3.1, F-002).
//
// Red Gate: references frame.ErrInvalidFrameType which does not exist until
// the implementer's commit lands.
func TestParseOuterHeader_RejectsInvalidFrameType(t *testing.T) {
	t.Parallel()
	// Bytes not in {0x01..0x06}: the canonical six enum values.
	invalids := []byte{0x00, 0x07, 0x77, 0xFF}
	for _, b := range invalids {
		b := b
		t.Run(fmt.Sprintf("frame_type=%#x", b), func(t *testing.T) {
			t.Parallel()
			buf := make([]byte, frame.OuterHeaderSize)
			buf[0] = frame.VersionByte // valid version
			buf[1] = b                 // invalid frame_type
			_, err := frame.ParseOuterHeader(buf)
			if err == nil {
				t.Fatalf("ParseOuterHeader with frame_type=%#x returned nil error, want ErrInvalidFrameType", b)
			}
			if !errors.Is(err, frame.ErrInvalidFrameType) {
				t.Errorf("err = %v, want errors.Is(err, frame.ErrInvalidFrameType)", err)
			}
		})
	}
}

// TestParseOuterHeader_AcceptsAllValidFrameTypes asserts that all six
// canonical FrameType values pass ParseOuterHeader's enum validation.
// This is a regression guard against an over-strict validator.
//
// Red Gate: references frame.FrameType (named type) and frame.ErrInvalidFrameType
// which do not exist until the implementer's commit lands.
func TestParseOuterHeader_AcceptsAllValidFrameTypes(t *testing.T) {
	t.Parallel()
	valid := []frame.FrameType{
		frame.FrameTypeData,
		frame.FrameTypeEmptyTick,
		frame.FrameTypeCtl,
		frame.FrameTypeArq,
		frame.FrameTypeFec,
		frame.FrameTypePEConnect,
	}
	for _, ft := range valid {
		ft := ft
		t.Run(fmt.Sprintf("frame_type=%#x", byte(ft)), func(t *testing.T) {
			t.Parallel()
			buf := make([]byte, frame.OuterHeaderSize)
			buf[0] = frame.VersionByte
			buf[1] = byte(ft)
			hdr, err := frame.ParseOuterHeader(buf)
			if err != nil {
				t.Fatalf("ParseOuterHeader with valid frame_type=%#x: unexpected err: %v", byte(ft), err)
			}
			if hdr.FrameType != ft {
				t.Errorf("hdr.FrameType = %#x, want %#x", byte(hdr.FrameType), byte(ft))
			}
		})
	}
}

// TestFrameType_Valid_PEConnect asserts that FrameTypePEConnect.Valid() returns
// true (0x06 is now a canonical enum value) and that FrameType(0x07).Valid()
// returns false (the upper bound is not over-widened). (S-BL.PE-RECEIVE-LOOP AC-003)
//
// This test will PASS immediately against the stub (FrameTypePEConnect and the
// widened Valid() are already defined in the stub commit) — that is expected and
// correct per the RED gate rule: valid-constant tests pass at RED; behavioral
// tests that require the receive goroutine fail at RED.
func TestFrameType_Valid_PEConnect(t *testing.T) {
	t.Parallel()

	if !frame.FrameTypePEConnect.Valid() {
		t.Errorf("FrameTypePEConnect.Valid() = false, want true (0x06 must be canonical after S-BL.PE-RECEIVE-LOOP)")
	}
	if frame.FrameType(0x07).Valid() {
		t.Errorf("FrameType(0x07).Valid() = true, want false (upper bound must not be over-widened beyond 0x06)")
	}
}

// assertSHA256Address is a test helper that derives the expected 8-byte node
// address using SHA-256(svtnID || publicKey)[:8] and compares to got.
func assertSHA256Address(t *testing.T, svtnID [16]byte, publicKey []byte, got [8]byte) {
	t.Helper()
	h := sha256.New()
	h.Write(svtnID[:])
	h.Write(publicKey)
	sum := h.Sum(nil)
	var want [8]byte
	copy(want[:], sum[:8])
	if got != want {
		t.Errorf("DeriveNodeAddress result mismatch\ngot:  %x\nwant: %x (SHA-256(%x || %x)[:8])",
			got, want, svtnID, publicKey)
	}
}
