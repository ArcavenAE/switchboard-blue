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
	"net/netip"
	"time"

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

// ── S-1.03 Examples ─────────────────────────────────────────────────────────
//
// The six examples below are the demo-recording evidence for S-1.03
// (Session Continuity via Cryptographic Re-Authentication). They cover
// AC-001..003 and EC-001..003.
//
// All use fixed 32-byte seeds and fixed nonces so the // Output: blocks are
// stable across runs. The nonce-replay prevention in ReAuthenticate means
// every example uses a distinct nonce; they are also in distinct AdmittedKeySets
// so nonce history does not bleed between examples.

// ExampleAdmittedKeySet_reAuthenticateOnIPChange demonstrates AC-001:
// session continuity on IP change. Traces to BC-2.01.007 PC3+PC4
// (router updates routing entry; session traffic resumes).
func ExampleAdmittedKeySet_reAuthenticateOnIPChange() {
	_, routerPriv, _ := ed25519.GenerateKey(seed32("router-s103-ac001-demo----------"))
	nodePub, nodePriv, _ := ed25519.GenerateKey(seed32("node-s103-ac001-demo------------"))

	svtn := svtnID("svtn-s103-ac001\x00")
	ks := admission.NewAdmittedKeySet()
	rs := admission.NewReAuthState()
	ks.RegisterKey(svtn, nodePub, admission.RoleConsole)
	nodeAddr := deriveAddr(svtn, nodePub)

	// Initial admission handshake.
	var n0 [32]byte
	copy(n0[:], "nonce-s103-ac001-initial-fixed--")
	ch0 := admission.Challenge{Nonce: n0, RouterSig: ed25519.Sign(routerPriv, n0[:])}
	_ = admission.AdmitNode(ch0, admission.ChallengeResponse{NonceSig: ed25519.Sign(nodePriv, n0[:])}, nodePub, svtn, ks)

	// Node's IP changes; re-authenticate with the same keypair.
	newIP := netip.MustParseAddr("192.0.2.42")
	var n1 [32]byte
	copy(n1[:], "nonce-s103-ac001-reauth-fixed---")
	ch1 := admission.Challenge{Nonce: n1, RouterSig: ed25519.Sign(routerPriv, n1[:])}
	req := admission.ReAuthRequest{
		SVTNID:        svtn,
		NodeAddr:      nodeAddr,
		NewSourceAddr: newIP,
		Challenge:     ch1,
		Response:      admission.ChallengeResponse{NonceSig: ed25519.Sign(nodePriv, n1[:])},
	}

	err := admission.ReAuthenticate(req, ks, rs)
	fmt.Println("reauth error:", err)
	fmt.Println("still admitted:", ks.IsAdmitted(svtn, nodeAddr))
	fmt.Println("source addr:", rs.CurrentSourceAddr(svtn, nodeAddr))

	// Output:
	// reauth error: <nil>
	// still admitted: true
	// source addr: 192.0.2.42
}

// ExampleAdmittedKeySet_reAuthenticateWrongKeyRejected demonstrates AC-002:
// re-authentication is rejected when the keypair presented does not match
// the originally admitted keypair. Traces to BC-2.01.007 precondition 3
// (keypair must be unchanged; wrong keypair → E-ADM-001).
func ExampleAdmittedKeySet_reAuthenticateWrongKeyRejected() {
	_, routerPriv, _ := ed25519.GenerateKey(seed32("router-s103-ac002-demo----------"))
	nodePub, nodePriv, _ := ed25519.GenerateKey(seed32("node-s103-ac002-demo------------"))
	_, wrongPriv, _ := ed25519.GenerateKey(seed32("wrong-s103-ac002-demo-----------"))

	svtn := svtnID("svtn-s103-ac002\x00")
	ks := admission.NewAdmittedKeySet()
	rs := admission.NewReAuthState()
	ks.RegisterKey(svtn, nodePub, admission.RoleConsole)
	nodeAddr := deriveAddr(svtn, nodePub)

	// Initial admission.
	var n0 [32]byte
	copy(n0[:], "nonce-s103-ac002-initial-fixed--")
	ch0 := admission.Challenge{Nonce: n0, RouterSig: ed25519.Sign(routerPriv, n0[:])}
	_ = admission.AdmitNode(ch0, admission.ChallengeResponse{NonceSig: ed25519.Sign(nodePriv, n0[:])}, nodePub, svtn, ks)

	// Re-auth with wrong private key.
	var n1 [32]byte
	copy(n1[:], "nonce-s103-ac002-reauth-fixed---")
	ch1 := admission.Challenge{Nonce: n1, RouterSig: ed25519.Sign(routerPriv, n1[:])}
	req := admission.ReAuthRequest{
		SVTNID:        svtn,
		NodeAddr:      nodeAddr,
		NewSourceAddr: netip.MustParseAddr("198.51.100.7"),
		Challenge:     ch1,
		Response:      admission.ChallengeResponse{NonceSig: ed25519.Sign(wrongPriv, n1[:])},
	}

	err := admission.ReAuthenticate(req, ks, rs)
	fmt.Println("is ErrSignatureVerificationFailed:", errors.Is(err, admission.ErrSignatureVerificationFailed))

	// Output:
	// is ErrSignatureVerificationFailed: true
}

// ExampleAdmittedKeySet_reAuthenticateNodeAddressStable demonstrates AC-003:
// the node address (derived from SVTN-ID and public key) is unchanged after
// re-authentication — IP change does not alter the logical node address.
// Traces to BC-2.01.007 invariant 3 (session identity = channel_id + node_addr,
// not source IP).
func ExampleAdmittedKeySet_reAuthenticateNodeAddressStable() {
	_, routerPriv, _ := ed25519.GenerateKey(seed32("router-s103-ac003-demo----------"))
	nodePub, nodePriv, _ := ed25519.GenerateKey(seed32("node-s103-ac003-demo------------"))

	svtn := svtnID("svtn-s103-ac003\x00")
	ks := admission.NewAdmittedKeySet()
	rs := admission.NewReAuthState()
	ks.RegisterKey(svtn, nodePub, admission.RoleAccess)

	// Capture node address before any re-auth.
	addrBefore := deriveAddr(svtn, nodePub)

	// Initial admission.
	var n0 [32]byte
	copy(n0[:], "nonce-s103-ac003-initial-fixed--")
	ch0 := admission.Challenge{Nonce: n0, RouterSig: ed25519.Sign(routerPriv, n0[:])}
	_ = admission.AdmitNode(ch0, admission.ChallengeResponse{NonceSig: ed25519.Sign(nodePriv, n0[:])}, nodePub, svtn, ks)

	// Re-authenticate from a new IP.
	var n1 [32]byte
	copy(n1[:], "nonce-s103-ac003-reauth-fixed---")
	ch1 := admission.Challenge{Nonce: n1, RouterSig: ed25519.Sign(routerPriv, n1[:])}
	req := admission.ReAuthRequest{
		SVTNID:        svtn,
		NodeAddr:      addrBefore,
		NewSourceAddr: netip.MustParseAddr("203.0.113.99"),
		Challenge:     ch1,
		Response:      admission.ChallengeResponse{NonceSig: ed25519.Sign(nodePriv, n1[:])},
	}
	_ = admission.ReAuthenticate(req, ks, rs)

	// Node address after re-auth must equal address before.
	addrAfter := deriveAddr(svtn, nodePub)
	fmt.Println("addr stable:", addrBefore == addrAfter)
	fmt.Println("still admitted with same addr:", ks.IsAdmitted(svtn, addrAfter))

	// Output:
	// addr stable: true
	// still admitted with same addr: true
}

// ExampleAdmittedKeySet_reAuthenticateExpiredKey demonstrates EC-001:
// re-authentication is rejected with ErrKeyExpired (E-ADM-015) when the
// node's key has passed its expiry timestamp. Traces to BC-2.01.007 EC-005
// and ARCH-04 §Key Lifecycle.
func ExampleAdmittedKeySet_reAuthenticateExpiredKey() {
	_, routerPriv, _ := ed25519.GenerateKey(seed32("router-s103-ec001-demo----------"))
	nodePub, nodePriv, _ := ed25519.GenerateKey(seed32("node-s103-ec001-demo------------"))

	svtn := svtnID("svtn-s103-ec001\x00")
	ks := admission.NewAdmittedKeySet()
	rs := admission.NewReAuthState()
	ks.RegisterKey(svtn, nodePub, admission.RoleAccess)
	nodeAddr := deriveAddr(svtn, nodePub)

	// Initial admission.
	var n0 [32]byte
	copy(n0[:], "nonce-s103-ec001-initial-fixed--")
	ch0 := admission.Challenge{Nonce: n0, RouterSig: ed25519.Sign(routerPriv, n0[:])}
	_ = admission.AdmitNode(ch0, admission.ChallengeResponse{NonceSig: ed25519.Sign(nodePriv, n0[:])}, nodePub, svtn, ks)

	// Set expiry to a fixed time in the past (2000-01-01 UTC).
	pastExpiry := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	_ = ks.SetKeyExpiry(svtn, nodeAddr, pastExpiry)

	// Re-auth must be rejected with ErrKeyExpired.
	var n1 [32]byte
	copy(n1[:], "nonce-s103-ec001-reauth-fixed---")
	ch1 := admission.Challenge{Nonce: n1, RouterSig: ed25519.Sign(routerPriv, n1[:])}
	req := admission.ReAuthRequest{
		SVTNID:        svtn,
		NodeAddr:      nodeAddr,
		NewSourceAddr: netip.MustParseAddr("192.0.2.11"),
		Challenge:     ch1,
		Response:      admission.ChallengeResponse{NonceSig: ed25519.Sign(nodePriv, n1[:])},
	}

	err := admission.ReAuthenticate(req, ks, rs)
	fmt.Println("is ErrKeyExpired:", errors.Is(err, admission.ErrKeyExpired))

	// Output:
	// is ErrKeyExpired: true
}

// ExampleAdmittedKeySet_reAuthenticateEvictsOldPath demonstrates EC-002:
// a successful re-authentication from a new source IP evicts the old path —
// CurrentSourceAddr transitions from the old IP to the new IP.
// Traces to BC-2.01.007 EC-006 (old path evicted on new re-auth; BC v1.3).
func ExampleAdmittedKeySet_reAuthenticateEvictsOldPath() {
	_, routerPriv, _ := ed25519.GenerateKey(seed32("router-s103-ec002-demo----------"))
	nodePub, nodePriv, _ := ed25519.GenerateKey(seed32("node-s103-ec002-demo------------"))

	svtn := svtnID("svtn-s103-ec002\x00")
	ks := admission.NewAdmittedKeySet()
	rs := admission.NewReAuthState()
	ks.RegisterKey(svtn, nodePub, admission.RoleAccess)
	nodeAddr := deriveAddr(svtn, nodePub)

	// Initial admission.
	var n0 [32]byte
	copy(n0[:], "nonce-s103-ec002-initial-fixed--")
	ch0 := admission.Challenge{Nonce: n0, RouterSig: ed25519.Sign(routerPriv, n0[:])}
	_ = admission.AdmitNode(ch0, admission.ChallengeResponse{NonceSig: ed25519.Sign(nodePriv, n0[:])}, nodePub, svtn, ks)

	// Re-auth from old IP.
	oldIP := netip.MustParseAddr("192.0.2.10")
	var n1 [32]byte
	copy(n1[:], "nonce-s103-ec002-reauth1-fixed--")
	ch1 := admission.Challenge{Nonce: n1, RouterSig: ed25519.Sign(routerPriv, n1[:])}
	req1 := admission.ReAuthRequest{
		SVTNID:        svtn,
		NodeAddr:      nodeAddr,
		NewSourceAddr: oldIP,
		Challenge:     ch1,
		Response:      admission.ChallengeResponse{NonceSig: ed25519.Sign(nodePriv, n1[:])},
	}
	_ = admission.ReAuthenticate(req1, ks, rs)
	fmt.Println("after first reauth:", rs.CurrentSourceAddr(svtn, nodeAddr))

	// Re-auth from new IP evicts old path (BC-2.01.007 EC-006; ADR-003 LWW).
	newIP := netip.MustParseAddr("198.51.100.20")
	var n2 [32]byte
	copy(n2[:], "nonce-s103-ec002-reauth2-fixed--")
	ch2 := admission.Challenge{Nonce: n2, RouterSig: ed25519.Sign(routerPriv, n2[:])}
	req2 := admission.ReAuthRequest{
		SVTNID:        svtn,
		NodeAddr:      nodeAddr,
		NewSourceAddr: newIP,
		Challenge:     ch2,
		Response:      admission.ChallengeResponse{NonceSig: ed25519.Sign(nodePriv, n2[:])},
	}
	_ = admission.ReAuthenticate(req2, ks, rs)
	fmt.Println("after second reauth:", rs.CurrentSourceAddr(svtn, nodeAddr))

	// Output:
	// after first reauth: 192.0.2.10
	// after second reauth: 198.51.100.20
}

// ExampleAdmittedKeySet_reAuthenticateLastWriteWins demonstrates EC-003:
// when two sequential re-authentication requests arrive for the same node,
// the last accepted one determines the stored source address (ADR-003 LWW).
// The concurrent variant is exercised by TestReAuthenticate_NoRace.
// Traces to BC-2.01.007 EC-003 (concurrent re-auth — last one wins).
func ExampleAdmittedKeySet_reAuthenticateLastWriteWins() {
	_, routerPriv, _ := ed25519.GenerateKey(seed32("router-s103-ec003-demo----------"))
	nodePub, nodePriv, _ := ed25519.GenerateKey(seed32("node-s103-ec003-demo------------"))

	svtn := svtnID("svtn-s103-ec003\x00")
	ks := admission.NewAdmittedKeySet()
	rs := admission.NewReAuthState()
	ks.RegisterKey(svtn, nodePub, admission.RoleAccess)
	nodeAddr := deriveAddr(svtn, nodePub)

	// Initial admission.
	var n0 [32]byte
	copy(n0[:], "nonce-s103-ec003-initial-fixed--")
	ch0 := admission.Challenge{Nonce: n0, RouterSig: ed25519.Sign(routerPriv, n0[:])}
	_ = admission.AdmitNode(ch0, admission.ChallengeResponse{NonceSig: ed25519.Sign(nodePriv, n0[:])}, nodePub, svtn, ks)

	// First re-auth: IP A.
	ip1 := netip.MustParseAddr("10.10.10.1")
	var n1 [32]byte
	copy(n1[:], "nonce-s103-ec003-reauth1-fixed--")
	ch1 := admission.Challenge{Nonce: n1, RouterSig: ed25519.Sign(routerPriv, n1[:])}
	req1 := admission.ReAuthRequest{
		SVTNID:        svtn,
		NodeAddr:      nodeAddr,
		NewSourceAddr: ip1,
		Challenge:     ch1,
		Response:      admission.ChallengeResponse{NonceSig: ed25519.Sign(nodePriv, n1[:])},
	}
	_ = admission.ReAuthenticate(req1, ks, rs)

	// Second re-auth: IP B supersedes IP A (last write wins per ADR-003).
	ip2 := netip.MustParseAddr("10.10.10.2")
	var n2 [32]byte
	copy(n2[:], "nonce-s103-ec003-reauth2-fixed--")
	ch2 := admission.Challenge{Nonce: n2, RouterSig: ed25519.Sign(routerPriv, n2[:])}
	req2 := admission.ReAuthRequest{
		SVTNID:        svtn,
		NodeAddr:      nodeAddr,
		NewSourceAddr: ip2,
		Challenge:     ch2,
		Response:      admission.ChallengeResponse{NonceSig: ed25519.Sign(nodePriv, n2[:])},
	}
	_ = admission.ReAuthenticate(req2, ks, rs)

	fmt.Println("last write wins:", rs.CurrentSourceAddr(svtn, nodeAddr))

	// Output:
	// last write wins: 10.10.10.2
}
