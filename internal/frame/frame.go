// Package frame implements the Switchboard wire-format outer header codec.
// All nodes share this 44-byte outer header layout (ARCH-02, BC-2.01.004).
// Channel header bytes that follow the outer header are opaque to routers
// and are never parsed here (BC-2.01.005).
package frame

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// OuterHeaderSize is the fixed wire size of the outer header in bytes.
// Layout: version(1) + frame_type(1) + payload_len(2) + svtn_id(16) +
// src_addr(8) + dst_addr(8) + hmac_tag(8) = 44.
const OuterHeaderSize = 44

// Protocol version constants. VersionByte encodes major in bits[7:4] and
// minor in bits[3:0]. v0.1 = 0x01.
const (
	VersionMajor = 0
	VersionMinor = 1
	VersionByte  = 0x01
)

// FrameType is the wire-format enum byte for the frame_type field in the outer
// header (ARCH-02 §3.1). Only six canonical values are canonical; all others are
// reserved. ParseOuterHeader rejects reserved values with ErrInvalidFrameType.
type FrameType byte

// Frame type constants (ARCH-02 §3.1).
const (
	FrameTypeData      FrameType = 0x01
	FrameTypeEmptyTick FrameType = 0x02
	FrameTypeCtl       FrameType = 0x03 // payload carries a control_type discriminator byte (BC-2.01.008 schema home; ARCH-02 § "Outer Header Format")
	FrameTypeArq       FrameType = 0x04
	FrameTypeFec       FrameType = 0x05
	FrameTypePEConnect FrameType = 0x06 // (ARCH-02 §3.1)
)

// Valid reports whether the FrameType byte is one of the six canonical enum
// values defined in ARCH-02 §3.1. Returns false for 0x00 and 0x07..0xFF.
func (f FrameType) Valid() bool {
	return f >= FrameTypeData && f <= FrameTypePEConnect
}

// ErrInvalidFrameType is returned by ParseOuterHeader when the parsed
// frame_type byte is not one of the six canonical FrameType values
// (not in {0x01..0x06} per ARCH-02 §3.1).
var ErrInvalidFrameType = errors.New("frame: invalid frame_type")

// ErrFrameTooShort is returned by ParseOuterHeader when the input slice
// is shorter than OuterHeaderSize (44) bytes. Traces to E-PRT-002 /
// BC-2.01.004 precondition 1.
var ErrFrameTooShort = errors.New("frame: outer header requires 44 bytes")

// ErrVersionMismatch is returned by ParseOuterHeader when the version field's
// major nibble (bits[7:4]) is non-zero. Traces to E-PRT-001 / BC-2.01.004
// precondition 2.
var ErrVersionMismatch = errors.New("frame: unsupported protocol version")

// OuterHeader is the 44-byte outer header of every Switchboard frame.
// Fields match the ARCH-02 canonical wire layout exactly; no channel-header
// fields are included here (BC-2.01.005 invariant 1).
type OuterHeader struct {
	// Version encodes major version in bits[7:4] and minor in bits[3:0].
	// v0.1 = 0x01.
	Version byte
	// FrameType identifies the frame kind (data, ctl, arq, fec, empty-tick, pe_connect).
	FrameType FrameType
	// PayloadLen is the byte count of everything following the outer header
	// (channel header + application payload). Stored big-endian on the wire
	// per ARCH-02.
	PayloadLen uint16
	// SVTNID is the 16-byte session virtual transport network identifier.
	SVTNID [16]byte
	// SrcAddr is the 8-byte source node address.
	SrcAddr [8]byte
	// DstAddr is the 8-byte destination node address.
	DstAddr [8]byte
	// HMACTag is the 8-byte HMAC authentication tag.
	HMACTag [8]byte
}

// ReadOuterFrame reads exactly one framed message from r: OuterHeaderSize bytes
// followed by hdr.PayloadLen bytes of payload. Returns the parsed header and
// payload slice. The []byte return is payload-only; it does NOT include the outer
// header bytes. (new — S-BL.PE-RECEIVE-LOOP)
func ReadOuterFrame(r io.Reader) (OuterHeader, []byte, error) {
	var hdrBuf [OuterHeaderSize]byte
	if _, err := io.ReadFull(r, hdrBuf[:]); err != nil {
		return OuterHeader{}, nil, fmt.Errorf("frame.ReadOuterFrame: reading header: %w", err)
	}
	hdr, err := ParseOuterHeader(hdrBuf[:])
	if err != nil {
		return OuterHeader{}, nil, fmt.Errorf("frame.ReadOuterFrame: parsing header: %w", err)
	}
	payload := make([]byte, int(hdr.PayloadLen))
	if hdr.PayloadLen > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return OuterHeader{}, nil, fmt.Errorf("frame.ReadOuterFrame: reading payload: %w", err)
		}
	}
	return hdr, payload, nil
}

// EncodeOuterHeader serialises h into exactly OuterHeaderSize (44) bytes
// using the ARCH-02 big-endian wire layout.
func EncodeOuterHeader(h OuterHeader) [OuterHeaderSize]byte {
	var b [OuterHeaderSize]byte
	b[0] = h.Version
	b[1] = byte(h.FrameType)
	binary.BigEndian.PutUint16(b[2:4], h.PayloadLen)
	copy(b[4:20], h.SVTNID[:])
	copy(b[20:28], h.SrcAddr[:])
	copy(b[28:36], h.DstAddr[:])
	copy(b[36:44], h.HMACTag[:])
	return b
}

// ParseOuterHeader deserialises the first 44 bytes of b into an OuterHeader.
// Returns ErrFrameTooShort if len(b) < 44, or ErrVersionMismatch if the version
// major nibble (bits[7:4]) is non-zero. Minor-version differences are tolerated.
func ParseOuterHeader(b []byte) (OuterHeader, error) {
	if len(b) < OuterHeaderSize {
		return OuterHeader{}, fmt.Errorf("header truncated: expected %d bytes, got %d: %w", OuterHeaderSize, len(b), ErrFrameTooShort)
	}
	// Check major version nibble only — minor differences are forward-compatible.
	major := (b[0] >> 4) & 0x0F
	minor := b[0] & 0x0F
	if major != VersionMajor {
		return OuterHeader{}, fmt.Errorf("unsupported protocol version %d.%d: expected major version %d: %w", major, minor, VersionMajor, ErrVersionMismatch)
	}
	ft := FrameType(b[1])
	if !ft.Valid() {
		return OuterHeader{}, fmt.Errorf("frame_type %#x: %w", b[1], ErrInvalidFrameType)
	}
	var h OuterHeader
	h.Version = b[0]
	h.FrameType = ft
	h.PayloadLen = binary.BigEndian.Uint16(b[2:4])
	copy(h.SVTNID[:], b[4:20])
	copy(h.SrcAddr[:], b[20:28])
	copy(h.DstAddr[:], b[28:36])
	copy(h.HMACTag[:], b[36:44])
	return h, nil
}
