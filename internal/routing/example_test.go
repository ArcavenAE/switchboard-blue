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
