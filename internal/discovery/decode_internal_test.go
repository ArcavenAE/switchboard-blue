// decode_internal_test.go exercises decodeBody directly as an internal test
// (package discovery, not package discovery_test) because decodeBody is
// unexported. These tests are discriminating RED→GREEN tests for the
// Step-4.5 pass-4 F-3 UTF-8 partial-fix parity guard.
package discovery

import (
	"encoding/binary"
	"errors"
	"testing"
	"unicode/utf8"
)

// buildDecodeBodyRawWithRawName builds a raw body slice (without the HMAC tag
// prefix) with a single session whose name is given as raw bytes (possibly
// non-UTF-8). This bypasses the []byte(string) conversion, which is necessary
// to inject invalid UTF-8 sequences that cannot be constructed via the normal
// SessionPresence path. Layout mirrors decodeBody's expected format:
//
//	[16]SVTNID | [8]NodeAddr | [8]Sequence | uint16 count=1 |
//	uint16 name_len | name_bytes | uint8 status | uint8 quality
func buildDecodeBodyRawWithRawName(nameBytes []byte, status AttachmentStatus, quality QualityIndicator) []byte {
	svtnID := [16]byte{0xDE, 0xAD}
	nodeAddr := [8]byte{0x01}
	const sequence = uint64(42)

	body := make([]byte, 0, 34+2+len(nameBytes)+2)
	body = append(body, svtnID[:]...)
	body = append(body, nodeAddr[:]...)
	body = binary.BigEndian.AppendUint64(body, sequence)
	body = binary.BigEndian.AppendUint16(body, 1) // count = 1
	body = binary.BigEndian.AppendUint16(body, uint16(len(nameBytes)))
	body = append(body, nameBytes...)
	body = append(body, byte(status))
	body = append(body, byte(quality))
	return body
}

// TestDecodeBody_RejectsNonUTF8Name is the RED test for Step-4.5 pass-4 F-3:
// decodeBody must reject session names that are not valid UTF-8, matching the
// guard in DecodeSessionList (same file, Step-4.5 pass-1 HIGH).
//
// The identical round-trip argument applies to both paths: encodedSessionName
// requires valid UTF-8 (BC-2.03.003 Inv-1), so any name accepted here that
// fails re-encode breaks the Encode/Decode round-trip invariant.
//
// This is a discriminating test: removing the guard makes decodeBody accept
// the invalid-UTF-8 body and return a non-nil payload (nil error), which
// fails the err==nil assertion below.
func TestDecodeBody_RejectsNonUTF8Name(t *testing.T) {
	t.Parallel()

	invalidUTF8 := []byte{0xFF, 0xFE}
	// Sanity: confirm the bytes are in fact invalid UTF-8 so the test
	// is honest about what it exercises.
	if utf8.Valid(invalidUTF8) {
		t.Fatal("test setup: byte sequence must be invalid UTF-8")
	}

	body := buildDecodeBodyRawWithRawName(invalidUTF8, Detached, QualityGreen)

	payload, err := decodeBody(body)
	if err == nil {
		t.Fatalf("decodeBody(non-UTF-8 name): got nil error, payload=%+v; want ErrInvalidSessionName (BC-2.03.003 Inv-1, Step-4.5 pass-4 F-3)", payload)
	}
	if !errors.Is(err, ErrInvalidSessionName) {
		t.Errorf("decodeBody(non-UTF-8 name): got %v, want errors.Is(err, ErrInvalidSessionName)", err)
	}
}

// TestDecodeBody_AcceptsValidMultibyteUTF8Name is the regression guard:
// the UTF-8 fix must not reject valid multibyte UTF-8 session names.
// "café" is 5 bytes (4 ASCII + 1 two-byte rune) — a representative
// non-ASCII valid UTF-8 string.
func TestDecodeBody_AcceptsValidMultibyteUTF8Name(t *testing.T) {
	t.Parallel()

	name := "café"
	nameBytes := []byte(name)
	if !utf8.Valid(nameBytes) {
		t.Fatal("test setup: name must be valid UTF-8")
	}

	body := buildDecodeBodyRawWithRawName(nameBytes, Attached, QualityGreen)

	payload, err := decodeBody(body)
	if err != nil {
		t.Fatalf("decodeBody(valid multibyte UTF-8 name %q): unexpected error: %v", name, err)
	}
	if len(payload.Sessions) != 1 {
		t.Fatalf("decodeBody: got %d sessions, want 1", len(payload.Sessions))
	}
	if payload.Sessions[0].SessionName != name {
		t.Errorf("decodeBody: got SessionName=%q, want %q", payload.Sessions[0].SessionName, name)
	}
}
