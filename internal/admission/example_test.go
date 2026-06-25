// Package admission_test — godoc examples exercising the public API end-to-end.
// This file is evidence for S-2.02 demo-recording: it demonstrates AC-001 through
// AC-007 using fixed seeds for deterministic output.
package admission_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/arcavenae/switchboard/internal/admission"
)

// seed32 returns a deterministic 32-byte io.Reader for ed25519 key generation.
// label must be at most 32 bytes; shorter labels are right-padded with zeros.
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

// deriveAddr reproduces frame.DeriveNodeAddress(svtnID, pubKey) so that examples
// can compute the node address without importing internal/frame directly.
// Implements SHA-256(svtnID || pubKey) truncated to 8 bytes (AC per BC-2.01 §Address).
func deriveAddr(svtn [16]byte, pub ed25519.PublicKey) [8]byte {
	h := sha256.New()
	h.Write(svtn[:])
	h.Write([]byte(pub))
	sum := h.Sum(nil)
	var addr [8]byte
	copy(addr[:], sum[:8])
	return addr
}

// ExampleAdmittedKeySet_admitNode demonstrates AC-001: successful Ed25519
// challenge-response admission. Traces to BC-2.05.001 postcondition 4 (node
// added to active admitted set on valid signature).
func ExampleAdmittedKeySet_admitNode() {
	_, routerPriv, _ := ed25519.GenerateKey(seed32("router-key-for-s202-demo-ac001--"))
	nodePub, nodePriv, _ := ed25519.GenerateKey(seed32("node-key-for-s202-demo-ac001----"))

	svtn := svtnID("svtn-demo-ac001\x00")
	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtn, nodePub, admission.RoleAccess)

	// Deterministic challenge: fixed nonce, signed by router.
	var nonce [32]byte
	copy(nonce[:], "challenge-nonce-ac001-demo-fixed")
	ch := admission.Challenge{Nonce: nonce, RouterSig: ed25519.Sign(routerPriv, nonce[:])}

	// Node signs nonce with its private key.
	resp := admission.ChallengeResponse{NonceSig: ed25519.Sign(nodePriv, nonce[:])}

	err := admission.AdmitNode(ch, resp, nodePub, svtn, ks)
	fmt.Println("admit error:", err)
	fmt.Println("is admitted:", ks.IsAdmitted(svtn, deriveAddr(svtn, nodePub)))

	// Output:
	// admit error: <nil>
	// is admitted: true
}

// ExampleAdmittedKeySet_invalidSignature demonstrates AC-002: AdmitNode returns
// ErrSignatureVerificationFailed (E-ADM-001) when the node's signature does not
// verify against its registered public key. Traces to BC-2.05.001 postcondition 5.
func ExampleAdmittedKeySet_invalidSignature() {
	_, routerPriv, _ := ed25519.GenerateKey(seed32("router-key-for-s202-demo-ac002--"))
	nodePub, _, _ := ed25519.GenerateKey(seed32("node-key-for-s202-demo-ac002----"))
	// Use a different private key to produce a signature that won't verify against nodePub.
	_, wrongPriv, _ := ed25519.GenerateKey(seed32("wrong-key-for-s202-demo-ac002---"))

	svtn := svtnID("svtn-demo-ac002\x00")
	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtn, nodePub, admission.RoleAccess)

	var nonce [32]byte
	copy(nonce[:], "challenge-nonce-ac002-demo-fixed")
	ch := admission.Challenge{Nonce: nonce, RouterSig: ed25519.Sign(routerPriv, nonce[:])}
	resp := admission.ChallengeResponse{NonceSig: ed25519.Sign(wrongPriv, nonce[:])}

	err := admission.AdmitNode(ch, resp, nodePub, svtn, ks)
	fmt.Println("is ErrSignatureVerificationFailed:", errors.Is(err, admission.ErrSignatureVerificationFailed))

	// Output:
	// is ErrSignatureVerificationFailed: true
}

// ExampleAdmittedKeySet_replayDetection demonstrates AC-003: AdmitNode returns
// ErrNonceReplay (E-ADM-008) when the same challenge nonce is presented a second
// time. Traces to BC-2.05.001 invariant 3.
func ExampleAdmittedKeySet_replayDetection() {
	_, routerPriv, _ := ed25519.GenerateKey(seed32("router-key-for-s202-demo-ac003--"))
	nodePub, nodePriv, _ := ed25519.GenerateKey(seed32("node-key-for-s202-demo-ac003----"))

	svtn := svtnID("svtn-demo-ac003\x00")
	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtn, nodePub, admission.RoleAccess)

	var nonce [32]byte
	copy(nonce[:], "challenge-nonce-ac003-demo-fixed")
	ch := admission.Challenge{Nonce: nonce, RouterSig: ed25519.Sign(routerPriv, nonce[:])}
	resp := admission.ChallengeResponse{NonceSig: ed25519.Sign(nodePriv, nonce[:])}

	// First admission consumes the nonce.
	err1 := admission.AdmitNode(ch, resp, nodePub, svtn, ks)
	fmt.Println("first admission error:", err1)

	// Re-register resets admitted=false (EC-004 LWW semantic) so AdmitNode
	// can be called again — this isolates the replay check from the admitted gate.
	ks.RegisterKey(svtn, nodePub, admission.RoleAccess)

	// Second attempt with same nonce: replay detected.
	err2 := admission.AdmitNode(ch, resp, nodePub, svtn, ks)
	fmt.Println("is ErrNonceReplay:", errors.Is(err2, admission.ErrNonceReplay))

	// Output:
	// first admission error: <nil>
	// is ErrNonceReplay: true
}

// ExampleAdmittedKeySet_revokedKey demonstrates EC-003: AdmitNode returns
// ErrKeyRevoked (E-ADM-005) when the key has been revoked before the handshake.
// Traces to BC-2.05.001 EC-001.
func ExampleAdmittedKeySet_revokedKey() {
	_, routerPriv, _ := ed25519.GenerateKey(seed32("router-key-for-s202-demo-ac-rev-"))
	nodePub, nodePriv, _ := ed25519.GenerateKey(seed32("node-key-for-s202-demo-ac-rev---"))

	svtn := svtnID("svtn-demo-ac-rev\x00")
	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtn, nodePub, admission.RoleAccess)

	// Revoke the key before the handshake.
	nodeAddr := deriveAddr(svtn, nodePub)
	_ = ks.RevokeKey(svtn, nodeAddr)

	var nonce [32]byte
	copy(nonce[:], "challenge-nonce-acrev-demo-fixed")
	ch := admission.Challenge{Nonce: nonce, RouterSig: ed25519.Sign(routerPriv, nonce[:])}
	resp := admission.ChallengeResponse{NonceSig: ed25519.Sign(nodePriv, nonce[:])}

	err := admission.AdmitNode(ch, resp, nodePub, svtn, ks)
	fmt.Println("is ErrKeyRevoked:", errors.Is(err, admission.ErrKeyRevoked))

	// Output:
	// is ErrKeyRevoked: true
}

// ExampleAdmittedKeySet_isAdmitted demonstrates the two-state model: a registered
// key returns false from IsAdmitted until the challenge-response handshake completes.
// Traces to BC-2.05.001 postcondition 4 (admitted=false at RegisterKey; true only
// after AdmitNode succeeds).
func ExampleAdmittedKeySet_isAdmitted() {
	_, routerPriv, _ := ed25519.GenerateKey(seed32("router-key-for-s202-demo-isa----"))
	nodePub, nodePriv, _ := ed25519.GenerateKey(seed32("node-key-for-s202-demo-isa------"))

	svtn := svtnID("svtn-demo-isa\x00\x00\x00")
	ks := admission.NewAdmittedKeySet()
	nodeAddr := deriveAddr(svtn, nodePub)

	// Before registration: not admitted.
	fmt.Println("before register:", ks.IsAdmitted(svtn, nodeAddr))

	ks.RegisterKey(svtn, nodePub, admission.RoleAccess)

	// After registration but before handshake: still not admitted.
	fmt.Println("after register, before handshake:", ks.IsAdmitted(svtn, nodeAddr))

	// Complete the handshake.
	var nonce [32]byte
	copy(nonce[:], "challenge-nonce-isa-demo-fixed--")
	ch := admission.Challenge{Nonce: nonce, RouterSig: ed25519.Sign(routerPriv, nonce[:])}
	resp := admission.ChallengeResponse{NonceSig: ed25519.Sign(nodePriv, nonce[:])}
	_ = admission.AdmitNode(ch, resp, nodePub, svtn, ks)

	// After successful handshake: admitted.
	fmt.Println("after handshake:", ks.IsAdmitted(svtn, nodeAddr))

	// Output:
	// before register: false
	// after register, before handshake: false
	// after handshake: true
}

// ExampleGenerateChallenge_privateKeyAbsent demonstrates AC-006 and AC-007:
// GenerateChallenge produces a Challenge struct that contains no private key
// bytes. The router's private key is used only locally for the signing operation
// and is never serialized into the Challenge fields.
// Traces to BC-2.05.007 postcondition 1 and invariant 1 (DI-002).
func ExampleGenerateChallenge_privateKeyAbsent() {
	_, routerPriv, _ := ed25519.GenerateKey(seed32("router-key-for-s202-demo-gch----"))
	privBytes := []byte(routerPriv)

	ch, err := admission.GenerateChallenge(routerPriv)
	if err != nil {
		panic(fmt.Sprintf("GenerateChallenge: %v", err))
	}

	// Verify that neither Challenge field is a substring of the private key bytes.
	// Ed25519 private key = seed || public key (64 bytes); both halves checked.
	nonceInPriv := bytes.Contains(privBytes, ch.Nonce[:])
	sigInPriv := bytes.Contains(privBytes, ch.RouterSig)

	fmt.Println("nonce not in private key:", !nonceInPriv)
	fmt.Println("RouterSig not in private key:", !sigInPriv)
	fmt.Println("no private key material on wire:", !nonceInPriv && !sigInPriv)

	// Output:
	// nonce not in private key: true
	// RouterSig not in private key: true
	// no private key material on wire: true
}
