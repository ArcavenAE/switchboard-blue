package frame_test

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"reflect"
	"testing"

	"github.com/arcavenae/switchboard/internal/frame"
)

// AC-001 — TestEncodeOuterHeader_ExactlyFortyFourBytes
// Traces to BC-2.01.004 postcondition 1.
func TestEncodeOuterHeader_ExactlyFortyFourBytes(t *testing.T) {
	t.Parallel()

	h := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		PayloadLen: 256,
		SVTNID: [16]byte{
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
		},
		SrcAddr: [8]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88},
		DstAddr: [8]byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11},
		HMACTag: [8]byte{0xde, 0xad, 0xbe, 0xef, 0x01, 0x02, 0x03, 0x04},
	}

	encoded := frame.EncodeOuterHeader(h)

	// The return type is [OuterHeaderSize]byte — a fixed-size array.
	// Its length is always OuterHeaderSize (44) by construction.
	if len(encoded) != frame.OuterHeaderSize {
		t.Errorf("EncodeOuterHeader returned %d bytes, want %d", len(encoded), frame.OuterHeaderSize)
	}
	if frame.OuterHeaderSize != 44 {
		t.Errorf("OuterHeaderSize is %d, want 44", frame.OuterHeaderSize)
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
			name: "fec frame all-ones hmac_tag (EC-004)",
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
// Traces to BC-2.01.004 precondition 1 (ErrFrameTooShort / E-FRM-001).
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
		})
	}
}

// AC-004 — TestParseOuterHeader_VersionMismatch
// Traces to BC-2.01.004 precondition 2 (ErrVersionMismatch / E-FRM-002).
// Major version nibble (bits [7:4]) is non-zero → ErrVersionMismatch.
// Minor version differences (major nibble == 0) must NOT return ErrVersionMismatch.
func TestParseOuterHeader_VersionMismatch(t *testing.T) {
	t.Parallel()

	makeFrame := func(versionByte byte) []byte {
		b := make([]byte, frame.OuterHeaderSize)
		b[0] = versionByte
		b[1] = frame.FrameTypeData
		binary.BigEndian.PutUint16(b[2:4], 0)
		return b
	}

	t.Run("major=1 (0x10) returns ErrVersionMismatch", func(t *testing.T) {
		t.Parallel()
		_, err := frame.ParseOuterHeader(makeFrame(0x10))
		if !errors.Is(err, frame.ErrVersionMismatch) {
			t.Errorf("ParseOuterHeader(version=0x10) = %v, want ErrVersionMismatch", err)
		}
	})

	t.Run("major=15 (0xF0) returns ErrVersionMismatch", func(t *testing.T) {
		t.Parallel()
		_, err := frame.ParseOuterHeader(makeFrame(0xF0))
		if !errors.Is(err, frame.ErrVersionMismatch) {
			t.Errorf("ParseOuterHeader(version=0xF0) = %v, want ErrVersionMismatch", err)
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

	// Add seeds after stubs are implemented (they panic now, but the fuzz
	// corpus structure is required for the harness to compile).
	_ = seed1
	_ = seed2
	_ = seed3
	_ = seed4
	_ = seed5

	f.Add(make([]byte, frame.OuterHeaderSize))

	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) != frame.OuterHeaderSize {
			return
		}
		h, err := frame.ParseOuterHeader(b)
		if err != nil {
			return
		}
		encoded := frame.EncodeOuterHeader(h)
		// Every encoded byte must match the original input byte-for-byte
		// (round-trip identity; VP-001/002/003).
		for i := range frame.OuterHeaderSize {
			if encoded[i] != b[i] {
				t.Errorf("round-trip mismatch at byte %d: got 0x%02x, want 0x%02x", i, encoded[i], b[i])
			}
		}
	})
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
