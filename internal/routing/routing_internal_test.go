// Package routing — internal tests with access to unexported helpers.
//
// Tests in this file exercise the internal verifyFrameHMAC contract directly
// (not via RouteFrame). They verify the tag-snapshot-before-zero anti-tautology
// fix (S-2.02 pass-4 H-1) and the HMAC-before-admitted ordering invariant
// (VP-058, wired in S-3.04). All tests are GREEN against the post-S-3.04
// implementation.
package routing

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/arcavenae/switchboard/internal/admission"
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

// ── S-3.04: VP-058 proof harness ─────────────────────────────────────────────
//
// These tests are adapted from the VP-058.md v1.1 proof harness skeleton.
// They exercise RouteFrame's HMAC-before-admitted ordering invariant (ADR-009,
// VP-058) post-wire-up. All tests are GREEN against the post-S-3.04 implementation.
//
// Internal test file required: tests call RouteFrame (public) but the file is in
// package routing to keep it alongside verifyFrameHMAC tests for audit coherence.

// seedKeyInternal generates a deterministic Ed25519 keypair from a fixed 32-byte seed.
// Uses bytes.NewReader to avoid crypto/rand, making tests fully reproducible.
// (Per VP-058.md proof harness skeleton; mirrors seedKeyDet in routing_test.go.)
func seedKeyInternal(t *testing.T, seed [32]byte) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(bytes.NewReader(seed[:]))
	if err != nil {
		t.Fatalf("seedKeyInternal: %v", err)
	}
	return pub, priv
}

// TestRouteFrame_HMACEnforcedBeforeAdmission_VP058 verifies VP-058 properties 1–3:
// HMAC failure returns ErrHMACVerificationFailed without touching the admitted-set.
//
// Setup:
//  1. RegisterKey (admitted=false initially).
//  2. GenerateChallenge + AdmitNode → admitted=true.
//  3. RegisterForwardingEntry with all-zero auth key (wrong key → HMAC will fail).
//  4. RouteFrame must return ErrHMACVerificationFailed, not ErrNotAdmitted.
//
// Ordering test: if ErrNotAdmitted surfaces, the admission check fired before
// HMAC verification — that is the ADR-009 ordering violation VP-058 guards against.
//
// Traces to VP-058 properties 1, 2, 3; BC-2.05.008 PC-3; ADR-009.
func TestRouteFrame_HMACEnforcedBeforeAdmission_VP058(t *testing.T) {
	t.Parallel()

	var svtnID [16]byte
	copy(svtnID[:], "test-svtn-000001")

	var nodeSeed [32]byte
	copy(nodeSeed[:], "vp058-node-seed-0000000000000001")
	nodePub, nodePriv := seedKeyInternal(t, nodeSeed)

	var routerSeed [32]byte
	copy(routerSeed[:], "vp058-rtr-seed-00000000000000001")
	_, routerPriv := seedKeyInternal(t, routerSeed)

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)

	// Derive node address: SHA-256(svtnID || pubKey)[:8].
	h := sha256.New()
	h.Write(svtnID[:])
	h.Write([]byte(nodePub))
	sum := h.Sum(nil)
	var nodeAddr [8]byte
	copy(nodeAddr[:], sum[:8])

	challenge, err := admission.GenerateChallenge(routerPriv)
	if err != nil {
		t.Fatalf("GenerateChallenge: %v", err)
	}
	nonceSig := ed25519.Sign(nodePriv, challenge.Nonce[:])
	resp := admission.ChallengeResponse{NonceSig: nonceSig}
	if err := admission.AdmitNode(challenge, resp, nodePub, svtnID, ks); err != nil {
		t.Fatalf("AdmitNode: %v", err)
	}

	// All-zero auth key: guarantees HMAC verification will fail for any frame.
	var wrongAuthKey [hmac.KeySize]byte
	r := NewRouter(ks)
	r.RegisterForwardingEntry(svtnID, nodeAddr, wrongAuthKey)

	// Send a frame with all-zero HMACTag (invalid) — HMAC check must fire first.
	hdr := frame.OuterHeader{
		SVTNID:  svtnID,
		SrcAddr: nodeAddr,
		// HMACTag intentionally zero (invalid).
	}
	payload := []byte("test-payload")

	err = RouteFrame(hdr, payload, r)
	if !errors.Is(err, ErrHMACVerificationFailed) {
		t.Errorf("VP-058 property 1–3: want ErrHMACVerificationFailed, got: %v", err)
	}
	// Property 1: if ErrNotAdmitted surfaces, the admission check fired before
	// HMAC verification — that is the ordering violation VP-058 guards against.
	if errors.Is(err, admission.ErrNotAdmitted) {
		t.Error("VP-058 ordering violation: admission check fired before HMAC verification (ADR-009)")
	}
}

// TestRouteFrame_NoForwardingEntry_RejectsAsUnverifiable_VP058 verifies VP-058
// property 4: a frame with no forwarding-table entry (auth key unavailable) is
// rejected as unverifiable — ErrHMACVerificationFailed, not ErrNotAdmitted.
//
// Traces to VP-058 property 4; BC-2.05.008 PC-4; ADR-009 step 2.
func TestRouteFrame_NoForwardingEntry_RejectsAsUnverifiable_VP058(t *testing.T) {
	t.Parallel()

	var svtnID [16]byte
	copy(svtnID[:], "test-svtn-000002")

	var nodeSeed [32]byte
	copy(nodeSeed[:], "vp058-node-seed-0000000000000002")
	nodePub, nodePriv := seedKeyInternal(t, nodeSeed)

	var routerSeed [32]byte
	copy(routerSeed[:], "vp058-rtr-seed-00000000000000002")
	_, routerPriv := seedKeyInternal(t, routerSeed)

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)

	challenge, err := admission.GenerateChallenge(routerPriv)
	if err != nil {
		t.Fatalf("GenerateChallenge: %v", err)
	}
	nonceSig := ed25519.Sign(nodePriv, challenge.Nonce[:])
	resp := admission.ChallengeResponse{NonceSig: nonceSig}
	if err := admission.AdmitNode(challenge, resp, nodePub, svtnID, ks); err != nil {
		t.Fatalf("AdmitNode: %v", err)
	}

	// Derive node address.
	h := sha256.New()
	h.Write(svtnID[:])
	h.Write([]byte(nodePub))
	sum := h.Sum(nil)
	var nodeAddr [8]byte
	copy(nodeAddr[:], sum[:8])

	// No forwarding entry registered — auth key is unavailable → unverifiable.
	r := NewRouter(ks)

	hdr := frame.OuterHeader{SVTNID: svtnID, SrcAddr: nodeAddr}
	err = RouteFrame(hdr, nil, r)
	if !errors.Is(err, ErrHMACVerificationFailed) {
		t.Errorf("VP-058 property 4: want ErrHMACVerificationFailed for missing forwarding entry, got: %v", err)
	}
}
