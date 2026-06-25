// Package routing — internal tests with access to unexported helpers.
//
// verifyFrameHMAC is //nolint:unused until wire-layer HMAC enforcement is wired
// into RouteFrame (next wave). This file exercises the function directly to pin
// the H-1 tautology fix: the function must read hdr.HMACTag from the wire before
// zeroing the field for MAC computation, not vice versa.
package routing

import (
	"testing"

	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/hmac"
)

// TestVerifyFrameHMAC_RejectsWrongTag pins the H-1 fix: verifyFrameHMAC must
// read the wire tag (hdr.HMACTag) and reject mismatches. Prior to the fix the
// function computed a fresh MAC and then verified that MAC against itself,
// always returning true regardless of the wire tag.
//
// Two sub-cases:
//  1. Wire tag is deliberately wrong — must return false.
//  2. Wire tag is the correct tag — must return true.
//
// Traces to BC-2.05.002 invariant 1 (wire-layer HMAC enforced); adversary
// pass-4 finding H-1.
func TestVerifyFrameHMAC_RejectsWrongTag(t *testing.T) {
	t.Parallel()

	var authKey [hmac.KeySize]byte
	copy(authKey[:], "test-auth-key-32-bytes-determin")

	payload := []byte("frame-payload")

	// Build a header. PayloadLen and SVTNID matter only for the encoding;
	// correctness of verifyFrameHMAC depends only on whether HMACTag matches.
	var hdr frame.OuterHeader
	hdr.Version = frame.VersionByte
	hdr.FrameType = frame.FrameTypeData
	hdr.PayloadLen = uint16(len(payload))
	copy(hdr.SVTNID[:], "test-svtn-16byt!")
	copy(hdr.SrcAddr[:], "srcaddr!")
	copy(hdr.DstAddr[:], "dstaddr!")

	// Sub-case 1: wrong wire tag — must be rejected.
	var wrongTag [hmac.TagSize]byte
	copy(wrongTag[:], "WRONGTAG")
	hdr.HMACTag = wrongTag

	if verifyFrameHMAC(hdr, payload, authKey) {
		t.Fatal("verifyFrameHMAC accepted a wrong wire tag — H-1 regression (tautological MAC verify)")
	}

	// Sub-case 2: compute what the correct tag is, then verify acceptance.
	//
	// Protocol: zero HMACTag, encode header + payload, compute HMAC, write back
	// into HMACTag. This mirrors the sender's tag-insertion step.
	hdrForMAC := hdr
	hdrForMAC.HMACTag = [hmac.TagSize]byte{}
	encoded := frame.EncodeOuterHeader(hdrForMAC)

	msg := make([]byte, len(encoded)+len(payload))
	copy(msg, encoded[:])
	copy(msg[len(encoded):], payload)

	correctTag := hmac.ComputeHMAC(authKey[:], msg)
	hdr.HMACTag = correctTag

	if !verifyFrameHMAC(hdr, payload, authKey) {
		t.Fatal("verifyFrameHMAC rejected the correct wire tag — implementation bug")
	}
}
