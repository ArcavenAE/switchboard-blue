// Package routing_test — godoc examples exercising the public routing API end-to-end.
// This file is evidence for S-2.02 demo-recording: it demonstrates AC-004 and AC-005
// using deterministic inputs for pinned // Output: blocks.
package routing_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/hmac"
	"github.com/arcavenae/switchboard/internal/routing"
)

// seed32 returns a deterministic 32-byte io.Reader for ed25519 key generation.
func seed32(label string) *bytes.Reader {
	b := make([]byte, 32)
	copy(b, label)
	return bytes.NewReader(b)
}

// svtnID returns a deterministic [16]byte SVTN ID from a short ASCII label.
func svtnID(label string) [16]byte {
	var id [16]byte
	copy(id[:], label)
	return id
}

// deriveAddr reproduces frame.DeriveNodeAddress(svtnID, pubKey) so examples
// can compute node addresses without re-importing internal/frame explicitly.
func deriveAddr(svtn [16]byte, pub ed25519.PublicKey) [8]byte {
	h := sha256.New()
	h.Write(svtn[:])
	h.Write([]byte(pub))
	sum := h.Sum(nil)
	var addr [8]byte
	copy(addr[:], sum[:8])
	return addr
}

// admitNode fully admits a node: registers the key, builds a deterministic
// challenge, has the node sign the nonce, and calls AdmitNode.
func admitNode(ks *admission.AdmittedKeySet, svtn [16]byte, pub ed25519.PublicKey, priv ed25519.PrivateKey, nonceSeed string) {
	_, routerPriv, _ := ed25519.GenerateKey(seed32("router-key-for-routing-examples-"))
	ks.RegisterKey(svtn, pub, admission.RoleAccess)

	var nonce [32]byte
	copy(nonce[:], nonceSeed)
	ch := admission.Challenge{Nonce: nonce, RouterSig: ed25519.Sign(routerPriv, nonce[:])}
	resp := admission.ChallengeResponse{NonceSig: ed25519.Sign(priv, nonce[:])}
	if err := admission.AdmitNode(ch, resp, pub, svtn, ks); err != nil {
		panic(fmt.Sprintf("admitNode: %v", err))
	}
}

// exampleComputeTag computes the HMAC tag a sender would place in hdr.HMACTag.
// Protocol: zero HMACTag, encode header + payload, compute HMAC.
func exampleComputeTag(hdr frame.OuterHeader, payload []byte, authKey [hmac.KeySize]byte) [hmac.TagSize]byte {
	hdrForMAC := hdr
	hdrForMAC.HMACTag = [hmac.TagSize]byte{}
	encoded := frame.EncodeOuterHeader(hdrForMAC)
	msg := make([]byte, len(encoded)+len(payload))
	copy(msg, encoded[:])
	copy(msg[len(encoded):], payload)
	return hmac.ComputeHMAC(authKey[:], msg)
}

// ExampleRouter_dropsUnadmitted demonstrates AC-004 (post-S-3.04): RouteFrame
// drops the frame and returns admission.ErrNotAdmitted (E-ADM-003) when the
// frame's SrcAddr is not in the admitted set for the frame's SVTN.
//
// S-3.04: HMAC is enforced before admission. The forwarding entry and a valid
// HMAC tag are required so that the HMAC check passes and the admission
// check (ErrNotAdmitted) fires as the rejection reason.
//
// Traces to BC-2.05.002 postcondition 2.
func ExampleRouter_dropsUnadmitted() {
	nodePub, _, _ := ed25519.GenerateKey(seed32("node-key-for-routing-ac004------"))

	svtn := svtnID("svtn-demo-ac004\x00")
	ks := admission.NewAdmittedKeySet()

	// Register the key but do NOT complete the handshake — node is not admitted.
	ks.RegisterKey(svtn, nodePub, admission.RoleAccess)

	r := routing.NewRouter(ks)
	nodeAddr := deriveAddr(svtn, nodePub)

	// S-3.04: HMAC verified before admission; provide a forwarding entry and
	// compute a valid tag so the admission check (not HMAC) is the rejection.
	var authKey [hmac.KeySize]byte
	copy(authKey[:], "example-drops-unadmitted-key000")
	r.RegisterForwardingEntry(svtn, nodeAddr, authKey)

	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtn,
		SrcAddr:   nodeAddr,
		DstAddr:   nodeAddr, // self-addressed; only source admission matters here
	}
	hdr.HMACTag = exampleComputeTag(hdr, nil, authKey)

	err := routing.RouteFrame(hdr, nil, r)
	fmt.Println("is ErrNotAdmitted:", errors.Is(err, admission.ErrNotAdmitted))

	// Output:
	// is ErrNotAdmitted: true
}

// ExampleRouter_validHMACForwarded demonstrates AC-001 (S-3.04): a frame with a
// valid HMAC tag proceeds past the HMAC check, past the admitted-set check, and
// into SVTNRoute. RouteFrame returns nil when src is admitted and dst has a
// forwarding entry.
//
// Traces to BC-2.05.008 PC-1 (valid HMAC → admission proceeds) + ADR-009 v1.6
// ordering: verifyFrameHMAC fires before IsAdmitted before SVTNRoute.
func ExampleRouter_validHMACForwarded() {
	nodePub, nodePriv, _ := ed25519.GenerateKey(seed32("node-key-for-routing-ac001------"))

	svtn := svtnID("svtn-demo-ac001\x00")
	ks := admission.NewAdmittedKeySet()
	admitNode(ks, svtn, nodePub, nodePriv, "nonce-for-routing-ac001-example-")

	r := routing.NewRouter(ks)
	nodeAddr := deriveAddr(svtn, nodePub)

	// Register both src and dst forwarding entries. In this example src == dst
	// (self-addressed) so a single entry covers both lookup paths (src for HMAC,
	// dst for SVTNRoute).
	var authKey [hmac.KeySize]byte
	copy(authKey[:], "ac001-valid-hmac-forwarded-key00")
	r.RegisterForwardingEntry(svtn, nodeAddr, authKey)

	// A small payload demonstrates that HMAC covers header+payload together.
	payload := []byte("hello-switchboard")

	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtn,
		SrcAddr:   nodeAddr,
		DstAddr:   nodeAddr,
	}
	hdr.HMACTag = exampleComputeTag(hdr, payload, authKey)

	err := routing.RouteFrame(hdr, payload, r)
	fmt.Println("routed without error:", err == nil)

	// Output:
	// routed without error: true
}

// ExampleRouter_invalidHMACRejected demonstrates AC-002 (S-3.04): a frame whose
// HMAC tag does not match the expected MAC is rejected immediately with
// ErrHMACVerificationFailed. Neither IsAdmitted nor SVTNRoute is reached.
//
// Traces to BC-2.05.008 PC-2 (tag mismatch → E-ADM-016 → ErrHMACVerificationFailed)
// + ADR-009 v1.6.
func ExampleRouter_invalidHMACRejected() {
	nodePub, nodePriv, _ := ed25519.GenerateKey(seed32("node-key-for-routing-ac002------"))

	svtn := svtnID("svtn-demo-ac002\x00")
	ks := admission.NewAdmittedKeySet()
	admitNode(ks, svtn, nodePub, nodePriv, "nonce-for-routing-ac002-example-")

	r := routing.NewRouter(ks)
	nodeAddr := deriveAddr(svtn, nodePub)

	var authKey [hmac.KeySize]byte
	copy(authKey[:], "ac002-valid-hmac-key-for-router0")
	r.RegisterForwardingEntry(svtn, nodeAddr, authKey)

	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtn,
		SrcAddr:   nodeAddr,
		DstAddr:   nodeAddr,
	}
	// Deliberately set a wrong (all-0xFF) tag — this will not match the computed MAC.
	hdr.HMACTag = [hmac.TagSize]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

	err := routing.RouteFrame(hdr, nil, r)
	fmt.Println("is ErrHMACVerificationFailed:", errors.Is(err, routing.ErrHMACVerificationFailed))

	// Output:
	// is ErrHMACVerificationFailed: true
}

// ExampleRouter_hmacBeforeAdmission demonstrates AC-003 (S-3.04): HMAC
// verification fires BEFORE the admitted-set check. A frame from an admitted
// node that carries an invalid HMAC tag returns ErrHMACVerificationFailed — not
// admission.ErrNotAdmitted — proving the ordering invariant (ADR-009 v1.6;
// BC-2.05.008 PC-3; VP-058 property 1+2).
func ExampleRouter_hmacBeforeAdmission() {
	nodePub, nodePriv, _ := ed25519.GenerateKey(seed32("node-key-for-routing-ac003------"))

	svtn := svtnID("svtn-demo-ac003\x00")
	ks := admission.NewAdmittedKeySet()
	// Node IS admitted — if admission fired first it would pass and a different error
	// (or nil from SVTNRoute) would be returned.
	admitNode(ks, svtn, nodePub, nodePriv, "nonce-for-routing-ac003-example-")

	r := routing.NewRouter(ks)
	nodeAddr := deriveAddr(svtn, nodePub)

	var authKey [hmac.KeySize]byte
	copy(authKey[:], "ac003-hmac-before-admission-key0")
	r.RegisterForwardingEntry(svtn, nodeAddr, authKey)

	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtn,
		SrcAddr:   nodeAddr,
		DstAddr:   nodeAddr,
	}
	// Invalid tag — if the ordering were reversed (admission first), the error
	// would be nil (admitted node, self-dst present in table). The fact that we
	// get ErrHMACVerificationFailed proves HMAC fires first.
	hdr.HMACTag = [hmac.TagSize]byte{0x01}

	err := routing.RouteFrame(hdr, nil, r)
	fmt.Println("is ErrHMACVerificationFailed:", errors.Is(err, routing.ErrHMACVerificationFailed))
	fmt.Println("is ErrNotAdmitted:", errors.Is(err, admission.ErrNotAdmitted))

	// Output:
	// is ErrHMACVerificationFailed: true
	// is ErrNotAdmitted: false
}

// ExampleRouter_noForwardingEntry demonstrates AC-004 (S-3.04): when no
// forwarding-table entry exists for (hdr.SVTNID, hdr.SrcAddr), RouteFrame
// returns ErrHMACVerificationFailed because the auth key is unavailable — the
// frame is treated as unverifiable and dropped fail-closed. The admitted-set
// check is not reached.
//
// Traces to BC-2.05.008 PC-4 (missing entry → auth-key unavailable →
// ErrHMACVerificationFailed) + VP-058 property 4.
func ExampleRouter_noForwardingEntry() {
	nodePub, nodePriv, _ := ed25519.GenerateKey(seed32("node-key-for-routing-ac004------"))

	svtn := svtnID("svtn-demo-ac004\x00")
	ks := admission.NewAdmittedKeySet()
	admitNode(ks, svtn, nodePub, nodePriv, "nonce-for-routing-ac004-example-")

	r := routing.NewRouter(ks)
	nodeAddr := deriveAddr(svtn, nodePub)
	// Intentionally do NOT register a forwarding entry for (svtn, nodeAddr).

	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtn,
		SrcAddr:   nodeAddr,
		DstAddr:   nodeAddr,
	}
	// Tag value is irrelevant — the lookup fails before verifyFrameHMAC is called.
	hdr.HMACTag = [hmac.TagSize]byte{0xAB, 0xCD}

	err := routing.RouteFrame(hdr, nil, r)
	fmt.Println("is ErrHMACVerificationFailed:", errors.Is(err, routing.ErrHMACVerificationFailed))

	// Output:
	// is ErrHMACVerificationFailed: true
}

// ExampleRouter_validHMACUnadmittedRejected demonstrates AC-005 (S-3.04):
// ErrHMACVerificationFailed and admission.ErrNotAdmitted are distinct sentinels.
// A frame with a valid HMAC from a node that is NOT in the admitted set returns
// admission.ErrNotAdmitted — not ErrHMACVerificationFailed. Callers use
// errors.Is to distinguish the two.
//
// Traces to BC-2.05.008 EC-005 / invariant 2.
func ExampleRouter_validHMACUnadmittedRejected() {
	nodePub, _, _ := ed25519.GenerateKey(seed32("node-key-for-routing-ac005------"))

	svtn := svtnID("svtn-demo-ac005\x00")
	ks := admission.NewAdmittedKeySet()
	// Register key but do NOT complete the handshake — node is not admitted.
	ks.RegisterKey(svtn, nodePub, admission.RoleAccess)

	r := routing.NewRouter(ks)
	nodeAddr := deriveAddr(svtn, nodePub)

	var authKey [hmac.KeySize]byte
	copy(authKey[:], "ac005-unadmitted-valid-hmac-key0")
	r.RegisterForwardingEntry(svtn, nodeAddr, authKey)

	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtn,
		SrcAddr:   nodeAddr,
		DstAddr:   nodeAddr,
	}
	// Valid tag so HMAC passes; rejection comes from the admitted-set check.
	hdr.HMACTag = exampleComputeTag(hdr, nil, authKey)

	err := routing.RouteFrame(hdr, nil, r)
	fmt.Println("is ErrNotAdmitted:", errors.Is(err, admission.ErrNotAdmitted))
	fmt.Println("is ErrHMACVerificationFailed:", errors.Is(err, routing.ErrHMACVerificationFailed))

	// Output:
	// is ErrNotAdmitted: true
	// is ErrHMACVerificationFailed: false
}

// ExampleRouter_zeroHMACTagRejected demonstrates EC-001 (S-3.04): a frame whose
// HMACTag field is all-zeros is rejected with ErrHMACVerificationFailed. An
// all-zero tag cannot be a valid MAC except by astronomical chance.
//
// Traces to BC-2.05.008 EC-001.
func ExampleRouter_zeroHMACTagRejected() {
	nodePub, nodePriv, _ := ed25519.GenerateKey(seed32("node-key-for-routing-ec001------"))

	svtn := svtnID("svtn-demo-ec001\x00")
	ks := admission.NewAdmittedKeySet()
	admitNode(ks, svtn, nodePub, nodePriv, "nonce-for-routing-ec001-example-")

	r := routing.NewRouter(ks)
	nodeAddr := deriveAddr(svtn, nodePub)

	var authKey [hmac.KeySize]byte
	copy(authKey[:], "ec001-zero-tag-rejected-key00000")
	r.RegisterForwardingEntry(svtn, nodeAddr, authKey)

	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtn,
		SrcAddr:   nodeAddr,
		DstAddr:   nodeAddr,
	}
	// HMACTag is zero-value (all zeros) — must be rejected.
	// hdr.HMACTag = [hmac.TagSize]byte{} is the zero value; no assignment needed.

	err := routing.RouteFrame(hdr, nil, r)
	fmt.Println("is ErrHMACVerificationFailed:", errors.Is(err, routing.ErrHMACVerificationFailed))

	// Output:
	// is ErrHMACVerificationFailed: true
}

// ExampleRouter_wrongKeyHMACRejected demonstrates EC-002 (S-3.04): a frame whose
// HMAC tag was computed under a different node's key (cross-node forgery) is
// rejected with ErrHMACVerificationFailed. The forged tag does not match the
// expected MAC computed from the source node's registered FrameAuthKey.
//
// Traces to BC-2.05.008 EC-002.
func ExampleRouter_wrongKeyHMACRejected() {
	nodePub, nodePriv, _ := ed25519.GenerateKey(seed32("node-key-for-routing-ec002------"))

	svtn := svtnID("svtn-demo-ec002\x00")
	ks := admission.NewAdmittedKeySet()
	admitNode(ks, svtn, nodePub, nodePriv, "nonce-for-routing-ec002-example-")

	r := routing.NewRouter(ks)
	nodeAddr := deriveAddr(svtn, nodePub)

	// Legitimate key registered in the forwarding table for this node.
	var legitimateKey [hmac.KeySize]byte
	copy(legitimateKey[:], "ec002-legitimate-node-key0000000")
	r.RegisterForwardingEntry(svtn, nodeAddr, legitimateKey)

	// Attacker uses a different key to forge the HMAC tag.
	var attackerKey [hmac.KeySize]byte
	copy(attackerKey[:], "ec002-attacker-forged-key0000000")

	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtn,
		SrcAddr:   nodeAddr,
		DstAddr:   nodeAddr,
	}
	// Tag is computed under the attacker's key — mismatch with the registered key.
	hdr.HMACTag = exampleComputeTag(hdr, nil, attackerKey)

	err := routing.RouteFrame(hdr, nil, r)
	fmt.Println("is ErrHMACVerificationFailed:", errors.Is(err, routing.ErrHMACVerificationFailed))

	// Output:
	// is ErrHMACVerificationFailed: true
}

// ExampleRouter_svtnIsolation demonstrates AC-005 (post-S-3.04): SVTNRoute never
// delivers a frame to a node on a different SVTN. A frame with svtn_id=A is
// never forwarded to a node admitted only to svtn_id=B.
//
// S-3.04: HMAC is enforced before admission and forwarding. A forwarding entry
// for nodeA in svtnA (with a valid HMAC tag) is required so that HMAC and
// admission pass; SVTNRoute then rejects addrB (only in svtnB) with
// ErrNoForwardingEntry — demonstrating SVTN isolation.
//
// Traces to BC-2.05.006 postcondition 1.
func ExampleRouter_svtnIsolation() {
	pubA, privA, _ := ed25519.GenerateKey(seed32("node-key-for-routing-ac005-svtnA"))
	pubB, _, _ := ed25519.GenerateKey(seed32("node-key-for-routing-ac005-svtnB"))

	svtnA := svtnID("svtn-demo-ac005A")
	svtnB := svtnID("svtn-demo-ac005B")

	ks := admission.NewAdmittedKeySet()
	// Admit nodeA to svtn-A.
	admitNode(ks, svtnA, pubA, privA, "nonce-for-svtn-isolation-demo-A-")
	// Register nodeB to svtn-B only (not admitted to svtn-A).
	ks.RegisterKey(svtnB, pubB, admission.RoleAccess)

	r := routing.NewRouter(ks)

	addrA := deriveAddr(svtnA, pubA)
	addrB := deriveAddr(svtnB, pubB)

	// S-3.04: register nodeA's forwarding entry in svtnA with a known auth key.
	// HMAC is verified before admission and forwarding; nodeA needs an entry.
	var authKeyA [hmac.KeySize]byte
	copy(authKeyA[:], "svtn-isolation-example-keyA0000")
	r.RegisterForwardingEntry(svtnA, addrA, authKeyA)

	// Register nodeB's forwarding entry under svtn-B only.
	var authKeyB [hmac.KeySize]byte
	copy(authKeyB[:], "svtn-isolation-example-keyB0000")
	r.RegisterForwardingEntry(svtnB, addrB, authKeyB)

	// Build a frame: src=nodeA (on svtn-A), dst=nodeB (on svtn-B).
	// HMAC passes (nodeA has a valid entry in svtnA). Admission passes (nodeA
	// is admitted to svtnA). SVTNRoute looks up addrB in svtnA's forwarding
	// table — which has no entry for addrB. This enforces SVTN isolation:
	// nodeB is only reachable via svtnB frames.
	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtnA,
		SrcAddr:   addrA,
		DstAddr:   addrB, // destination belongs to svtn-B, not svtn-A
	}
	hdr.HMACTag = exampleComputeTag(hdr, nil, authKeyA)

	err := routing.RouteFrame(hdr, nil, r)
	fmt.Println("is ErrNoForwardingEntry:", errors.Is(err, routing.ErrNoForwardingEntry))

	// Output:
	// is ErrNoForwardingEntry: true
}
