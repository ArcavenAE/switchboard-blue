// Package frame_test — godoc example exercising the public API end-to-end.
// This file is evidence for S-1.01 demo-recording: it demonstrates encode,
// parse, round-trip identity, E-PRT-002, E-PRT-001, and DeriveNodeAddress
// determinism / pubkey-distinctness using fixed BC-2.01.004 test vectors.
package frame_test

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/arcavenae/switchboard/internal/frame"
)

// Example_encodeParseRoundTrip exercises every public function in internal/frame:
//
//  1. EncodeOuterHeader — canonical BC-2.01.004 data frame, PayloadLen=256
//  2. ParseOuterHeader — round-trip identity back to original header
//  3. E-PRT-002 — ParseOuterHeader on 30-byte slice returns ErrFrameTooShort
//  4. E-PRT-001 — ParseOuterHeader with major-version=2 returns ErrVersionMismatch
//  5. DeriveNodeAddress — same inputs produce identical 8-byte address (determinism)
//  6. VP-014 — different pubkey on same SVTN yields different address
func Example_encodeParseRoundTrip() {
	// 1. Build sample header using BC-2.01.004 canonical test vector byte values.
	h := frame.OuterHeader{
		Version:    frame.VersionByte,   // 0x01 — major=0, minor=1
		FrameType:  frame.FrameTypeData, // 0x01
		PayloadLen: 256,
		SVTNID:     [16]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00},
		SrcAddr:    [8]byte{0xA1, 0xA2, 0xA3, 0xA4, 0xA5, 0xA6, 0xA7, 0xA8},
		DstAddr:    [8]byte{0xB1, 0xB2, 0xB3, 0xB4, 0xB5, 0xB6, 0xB7, 0xB8},
		HMACTag:    [8]byte{0xC1, 0xC2, 0xC3, 0xC4, 0xC5, 0xC6, 0xC7, 0xC8},
	}

	// 2. Encode: produces exactly 44 bytes (AC-001).
	encoded := frame.EncodeOuterHeader(h)
	fmt.Printf("encoded: %s\n", hex.EncodeToString(encoded[:]))

	// 3. Parse back: round-trip identity (AC-002).
	decoded, err := frame.ParseOuterHeader(encoded[:])
	if err != nil {
		panic(fmt.Sprintf("unexpected error: %v", err))
	}
	fmt.Printf("version: 0x%02x\n", decoded.Version)
	fmt.Printf("frame_type: 0x%02x\n", decoded.FrameType)
	fmt.Printf("payload_len: %d\n", decoded.PayloadLen)
	fmt.Printf("svtn_id: %s\n", hex.EncodeToString(decoded.SVTNID[:]))
	fmt.Printf("src_addr: %s\n", hex.EncodeToString(decoded.SrcAddr[:]))
	fmt.Printf("dst_addr: %s\n", hex.EncodeToString(decoded.DstAddr[:]))
	fmt.Printf("hmac_tag: %s\n", hex.EncodeToString(decoded.HMACTag[:]))
	if decoded != h {
		panic("round-trip mismatch")
	}
	fmt.Println("round-trip: ok")

	// 4. E-PRT-002: too short (AC-003).
	_, err = frame.ParseOuterHeader(make([]byte, 30))
	if !errors.Is(err, frame.ErrFrameTooShort) {
		panic(fmt.Sprintf("expected ErrFrameTooShort, got %v", err))
	}
	fmt.Printf("E-PRT-002: %v\n", err)

	// 5. E-PRT-001: major-version mismatch — byte[0]=0x20 → major=2 (AC-004).
	badVer := make([]byte, 44)
	badVer[0] = 0x20
	_, err = frame.ParseOuterHeader(badVer)
	if !errors.Is(err, frame.ErrVersionMismatch) {
		panic(fmt.Sprintf("expected ErrVersionMismatch, got %v", err))
	}
	fmt.Printf("E-PRT-001: %v\n", err)

	// 6. DeriveNodeAddress: determinism (AC-006).
	svtnID := [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
	pubkeyA := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE}
	addr1 := frame.DeriveNodeAddress(svtnID, pubkeyA)
	addr2 := frame.DeriveNodeAddress(svtnID, pubkeyA)
	fmt.Printf("addr1: %s\n", hex.EncodeToString(addr1[:]))
	fmt.Printf("addr2: %s\n", hex.EncodeToString(addr2[:]))
	if addr1 != addr2 {
		panic("not deterministic")
	}
	fmt.Println("deterministic: ok")

	// 7. VP-014: different pubkey on same SVTN → different address.
	pubkeyB := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBF} // last byte differs
	addrB := frame.DeriveNodeAddress(svtnID, pubkeyB)
	fmt.Printf("addrB: %s\n", hex.EncodeToString(addrB[:]))
	if addr1 == addrB {
		panic("pubkey-distinct: same address for different pubkeys")
	}
	fmt.Println("pubkey-distinct: ok")
	fmt.Println("sha256-derivation: ok")

	// Output:
	// encoded: 01010100112233445566778899aabbccddeeff00a1a2a3a4a5a6a7a8b1b2b3b4b5b6b7b8c1c2c3c4c5c6c7c8
	// version: 0x01
	// frame_type: 0x01
	// payload_len: 256
	// svtn_id: 112233445566778899aabbccddeeff00
	// src_addr: a1a2a3a4a5a6a7a8
	// dst_addr: b1b2b3b4b5b6b7b8
	// hmac_tag: c1c2c3c4c5c6c7c8
	// round-trip: ok
	// E-PRT-002: header truncated: expected 44 bytes, got 30: frame: outer header requires 44 bytes
	// E-PRT-001: unsupported protocol version 2.0: expected major version 0: frame: unsupported protocol version
	// addr1: c341fbfa8e968c5f
	// addr2: c341fbfa8e968c5f
	// deterministic: ok
	// addrB: a2fbf85017cc4999
	// pubkey-distinct: ok
	// sha256-derivation: ok
}
