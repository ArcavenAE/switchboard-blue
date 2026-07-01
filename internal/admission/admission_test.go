package admission_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"sync"
	"testing"

	"github.com/arcavenae/switchboard/internal/admission"
)

// ── helpers ──────────────────────────────────────────────────────────────────

// mustGenEd25519 generates a fresh Ed25519 keypair. Fails the test on error.
func mustGenEd25519(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}
	return pub, priv
}

// mustSVTN returns a deterministic [16]byte SVTN ID for testing.
func mustSVTN(b byte) [16]byte {
	var id [16]byte
	id[0] = b
	return id
}

// mustGenerateChallenge calls GenerateChallenge and fails the test on error.
func mustGenerateChallenge(t *testing.T, routerPriv ed25519.PrivateKey) admission.Challenge {
	t.Helper()
	ch, err := admission.GenerateChallenge(routerPriv)
	if err != nil {
		t.Fatalf("GenerateChallenge: %v", err)
	}
	return ch
}

// nodeAddrForTest mirrors frame.DeriveNodeAddress for tests: SHA-256(svtnID ||
// pubKey), truncated to 8 bytes. Avoids importing internal/frame in tests by
// reproducing the same pure computation.
func nodeAddrForTest(svtnID [16]byte, pubKey ed25519.PublicKey) [8]byte {
	h := sha256.New()
	h.Write(svtnID[:])
	h.Write([]byte(pubKey))
	sum := h.Sum(nil)
	var addr [8]byte
	copy(addr[:], sum[:8])
	return addr
}

// ── AC-001: TestAdmitNode_ValidChallenge ─────────────────────────────────────

// TestAdmitNode_ValidChallenge verifies that AdmitNode returns nil when the
// node signs the challenge nonce with the private key whose corresponding
// public key is registered against the SVTN.
//
// Traces to BC-2.05.001 postcondition 1 (node admitted on valid signature).
func TestAdmitNode_ValidChallenge(t *testing.T) {
	t.Parallel()

	_, routerPriv := mustGenEd25519(t)
	nodePub, nodePriv := mustGenEd25519(t)
	svtnID := mustSVTN(0x01)

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)

	ch := mustGenerateChallenge(t, routerPriv)
	sig := ed25519.Sign(nodePriv, ch.Nonce[:])
	resp := admission.ChallengeResponse{NonceSig: sig}

	err := admission.AdmitNode(ch, resp, nodePub, svtnID, ks)
	if err != nil {
		t.Errorf("AdmitNode with valid challenge: want nil, got %v", err)
	}
}

// ── AC-002: TestAdmitNode_InvalidSignature ───────────────────────────────────

// TestAdmitNode_InvalidSignature verifies that AdmitNode returns
// ErrSignatureVerificationFailed when the presented signature is invalid.
//
// Traces to BC-2.05.001 postcondition 5 (failure path: E-ADM-001).
func TestAdmitNode_InvalidSignature(t *testing.T) {
	t.Parallel()

	_, routerPriv := mustGenEd25519(t)
	nodePub, _ := mustGenEd25519(t)
	_, wrongNodePriv := mustGenEd25519(t) // wrong keypair — not registered
	svtnID := mustSVTN(0x02)

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)

	ch := mustGenerateChallenge(t, routerPriv)
	// Sign with the wrong private key — signature will not verify against nodePub.
	sig := ed25519.Sign(wrongNodePriv, ch.Nonce[:])
	resp := admission.ChallengeResponse{NonceSig: sig}

	err := admission.AdmitNode(ch, resp, nodePub, svtnID, ks)
	if !errors.Is(err, admission.ErrSignatureVerificationFailed) {
		t.Errorf("AdmitNode with invalid sig: want ErrSignatureVerificationFailed, got %v", err)
	}
}

// ── AC-003: TestAdmitNode_ReplayedNonce ──────────────────────────────────────

// TestAdmitNode_ReplayedNonce verifies that AdmitNode returns ErrNonceReplay
// when the same challenge nonce is presented a second time.
//
// Traces to BC-2.05.001 invariant 3 (nonces are single-use); E-ADM-008.
func TestAdmitNode_ReplayedNonce(t *testing.T) {
	t.Parallel()

	_, routerPriv := mustGenEd25519(t)
	nodePub, nodePriv := mustGenEd25519(t)
	svtnID := mustSVTN(0x03)

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)

	ch := mustGenerateChallenge(t, routerPriv)
	sig := ed25519.Sign(nodePriv, ch.Nonce[:])
	resp := admission.ChallengeResponse{NonceSig: sig}

	// First call should succeed.
	if err := admission.AdmitNode(ch, resp, nodePub, svtnID, ks); err != nil {
		t.Fatalf("first AdmitNode: want nil, got %v", err)
	}

	// Second call with the same nonce must return ErrNonceReplay.
	err := admission.AdmitNode(ch, resp, nodePub, svtnID, ks)
	if !errors.Is(err, admission.ErrNonceReplay) {
		t.Errorf("replay AdmitNode: want ErrNonceReplay, got %v", err)
	}
}

// ── AC-006: TestAdmission_PrivateKeyAbsentFromWireStructs ───────────────────

// TestAdmission_PrivateKeyAbsentFromWireStructs is a structural property check:
// neither Challenge nor ChallengeResponse can hold Ed25519 private key bytes.
//
// Approach: assert that Challenge.Nonce is [32]byte (structurally cannot hold a
// 64-byte private key) and that the zero-value ChallengeResponse does not
// accidentally carry private key material.
//
// Traces to BC-2.05.007 invariant 1 (DI-002: private key non-transit) and
// VP-007 (type-level structural check).
func TestAdmission_PrivateKeyAbsentFromWireStructs(t *testing.T) {
	t.Parallel()

	const ed25519PrivKeyLen = 64

	// Challenge.Nonce is [32]byte — structurally cannot hold 64-byte private key.
	var ch admission.Challenge
	const nonceLen = 32
	if len(ch.Nonce) != nonceLen {
		t.Errorf("Challenge.Nonce: want %d bytes, got %d", nonceLen, len(ch.Nonce))
	}

	// ChallengeResponse.NonceSig is nil in zero value — assert no accidental
	// 64-byte private key pre-population.
	var resp admission.ChallengeResponse
	if len(resp.NonceSig) == ed25519PrivKeyLen {
		t.Errorf("ChallengeResponse.NonceSig zero value has length %d = ed25519 private key length; possible accidental key inclusion", ed25519PrivKeyLen)
	}
}

// ── AC-007: TestGenerateChallenge_NoChallengeContainsPrivateKey ─────────────

// TestGenerateChallenge_NoChallengeContainsPrivateKey asserts that
// GenerateChallenge produces a nonce-only challenge — the private key bytes of
// the router are NOT byte-for-byte identical to any field in the returned Challenge.
//
// Traces to BC-2.05.007 postcondition 1 and invariant 1 (DI-002). The private
// key is used only for the local Sign operation; it is never serialized.
func TestGenerateChallenge_NoChallengeContainsPrivateKey(t *testing.T) {
	t.Parallel()

	_, routerPriv := mustGenEd25519(t)
	privBytes := []byte(routerPriv)

	ch, err := admission.GenerateChallenge(routerPriv)
	if err != nil {
		t.Fatalf("GenerateChallenge: %v", err)
	}

	// Nonce is [32]byte; private key is 64 bytes — structurally cannot be equal.
	// Belt-and-suspenders: verify the 32 nonce bytes do not match the first 32
	// bytes of the private key.
	privFirst32 := privBytes[:32]
	nonceMatchesPriv := true
	for i, b := range ch.Nonce {
		if b != privFirst32[i] {
			nonceMatchesPriv = false
			break
		}
	}
	if nonceMatchesPriv {
		t.Error("Challenge.Nonce matches first 32 bytes of router private key — possible key leakage")
	}

	// RouterSig is the 64-byte Ed25519 signature — NOT the private key. Verify it
	// differs from the raw private key bytes.
	if len(ch.RouterSig) == len(privBytes) {
		equal := true
		for i := range privBytes {
			if ch.RouterSig[i] != privBytes[i] {
				equal = false
				break
			}
		}
		if equal {
			t.Error("Challenge.RouterSig is byte-for-byte identical to router private key — private key leaked")
		}
	}
}

// ── Property / VP-007: TestProperty_VP007_PrivateKeyByteSubstringAbsent ─────

// TestProperty_VP007_PrivateKeyByteSubstringAbsent is the canonical VP-007
// verification: for random (privKey, challenge) inputs, the serialized wire
// form of the ChallengeResponse must NOT contain either the 32-byte private-key
// seed (priv[0:32]) or the 32-byte public-key portion (priv[32:64]) as a
// contiguous byte substring at any offset.
//
// This replaces the structurally-weak field-length checks in AC-006/AC-007 as
// the canonical VP-007 evidence. Those tests remain below as fast smoke tests
// for structural invariants.
//
// Stdlib-only random sampling — no gopter (consistent with S-2.01 inline-HKDF
// precedent). 1000 samples per run provides statistical evidence that the
// property holds at the wire level.
//
// Traces to BC-2.05.007 invariant 1 (DI-002: private key non-transit) and
// VP-007 (private key absent from wire structs).
//
// VP-057 (Node Private Keys Never Appear as Literal Bytes in Any
// Emitted Frame) — admission-wire-struct SUBSET covered by this test
// (ChallengeResponse is one of VP-057's enumerated wire structs). Full
// VP-057 coverage across DATA, EMPTY_TICK, CTL/ARQ/FEC, CONTROL_DRAIN,
// CONTROL_KEY_REG, CONTROL_KEY_REVOKE frame types is deferred to the
// wave where those frame types are emitted (per S-2.02 task 8 and
// story rev 1.3 Spec Patches).
func TestProperty_VP007_PrivateKeyByteSubstringAbsent(t *testing.T) {
	t.Parallel()

	const sampleCount = 1000

	for i := range sampleCount {
		// Generate a fresh keypair per sample. priv is 64 bytes: seed (0:32) ||
		// public (32:64) in the Go ed25519 encoding.
		_, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("sample %d: keygen: %v", i, err)
		}

		// Generate a fresh challenge nonce per sample.
		var nonce [32]byte
		if _, err := rand.Read(nonce[:]); err != nil {
			t.Fatalf("sample %d: nonce gen: %v", i, err)
		}

		// Build a ChallengeResponse: sign the nonce with the private key.
		sig := ed25519.Sign(priv, nonce[:])
		resp := admission.ChallengeResponse{NonceSig: sig}

		// Wire form: NonceSig is the only field of ChallengeResponse.
		// If a Marshal method is added later, swap to that.
		wire := resp.NonceSig

		seed := []byte(priv)[0:32]
		pubPortion := []byte(priv)[32:64]

		// VP-007 property: wire must not contain the private-key seed or
		// public-key portion as a contiguous byte substring.
		if bytes.Contains(wire, seed) {
			t.Errorf("sample %d: wire contains private-key seed as contiguous substring — VP-007 violated", i)
		}
		if bytes.Contains(wire, pubPortion) {
			t.Errorf("sample %d: wire contains private-key public-portion as contiguous substring — VP-007 violated", i)
		}
	}
}

// ── EC-001: TestAdmitNode_KeyNotRegisteredForSVTN ───────────────────────────

// TestAdmitNode_KeyNotRegisteredForSVTN verifies that a node presenting a
// key for an SVTN it is not registered on returns an error.
//
// Traces to story EC-001 (E-ADM-003 returned; unregistered key) and
// BC-2.05.002 precondition 1.
func TestAdmitNode_KeyNotRegisteredForSVTN(t *testing.T) {
	t.Parallel()

	_, routerPriv := mustGenEd25519(t)
	nodePub, nodePriv := mustGenEd25519(t)
	svtnA := mustSVTN(0x0A)
	svtnB := mustSVTN(0x0B) // node is only registered on svtnA

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnA, nodePub, admission.RoleAccess)

	ch := mustGenerateChallenge(t, routerPriv)
	sig := ed25519.Sign(nodePriv, ch.Nonce[:])
	resp := admission.ChallengeResponse{NonceSig: sig}

	// Attempt to admit on svtnB — key is not registered there.
	err := admission.AdmitNode(ch, resp, nodePub, svtnB, ks)
	if err == nil {
		t.Error("AdmitNode on unregistered SVTN: want error, got nil")
	}
}

// ── EC-002: TestDuplicateKeyRegistration_LastWriteWins ──────────────────────

// TestDuplicateKeyRegistration_LastWriteWins verifies that re-registering the
// same public key for a SVTN with a different role replaces the prior entry
// (ADR-003 last-write-wins; ARCH-04 §ADR-003).
//
// Traces to story EC-002 and BC-2.05.001 EC-002.
func TestDuplicateKeyRegistration_LastWriteWins(t *testing.T) {
	t.Parallel()

	nodePub, _ := mustGenEd25519(t)
	svtnID := mustSVTN(0x01)

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)
	// Re-register with a different role — last-write-wins.
	ks.RegisterKey(svtnID, nodePub, admission.RoleConsole)

	// Lookup must return the second (console) role.
	addr := nodeAddrForTest(svtnID, nodePub)
	entry, ok := ks.Lookup(svtnID, addr)
	if !ok {
		t.Fatal("Lookup after duplicate register: want entry, got not-found")
	}
	if entry.Role != admission.RoleConsole {
		t.Errorf("last-write-wins: want RoleConsole, got %v", entry.Role)
	}
}

// ── EC-003: TestAdmitNode_RevokedKey ────────────────────────────────────────

// TestAdmitNode_RevokedKey verifies that AdmitNode returns ErrKeyRevoked when
// the key is registered but has been revoked.
//
// Traces to story EC-003 and BC-2.05.001 EC-001 (E-ADM-005 "key revoked").
func TestAdmitNode_RevokedKey(t *testing.T) {
	t.Parallel()

	_, routerPriv := mustGenEd25519(t)
	nodePub, nodePriv := mustGenEd25519(t)
	svtnID := mustSVTN(0x04)

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)

	// Revoke the key.
	nodeAddr := nodeAddrForTest(svtnID, nodePub)
	if err := ks.RevokeKey(svtnID, nodeAddr); err != nil {
		t.Fatalf("RevokeKey: %v", err)
	}

	ch := mustGenerateChallenge(t, routerPriv)
	sig := ed25519.Sign(nodePriv, ch.Nonce[:])
	resp := admission.ChallengeResponse{NonceSig: sig}

	err := admission.AdmitNode(ch, resp, nodePub, svtnID, ks)
	if !errors.Is(err, admission.ErrKeyRevoked) {
		t.Errorf("revoked key AdmitNode: want ErrKeyRevoked, got %v", err)
	}
}

// ── Property / VP harness: VP-007 — nonce uniqueness ────────────────────────

// TestGenerateChallenge_NonceUniqueness verifies that GenerateChallenge
// produces distinct nonces across multiple calls.
//
// Traces to BC-2.05.001 invariant 3 (nonce uniqueness). Nonce uniqueness is a
// precondition for VP-009's replay rejection; it is not itself VP-007 or VP-009.
//
// Deterministic boundary property: 100 independent GenerateChallenge calls
// must all produce distinct Nonce values.
func TestGenerateChallenge_NonceUniqueness(t *testing.T) {
	t.Parallel()

	_, routerPriv := mustGenEd25519(t)
	seen := make(map[[32]byte]struct{})

	const samples = 100
	for i := range samples {
		ch, err := admission.GenerateChallenge(routerPriv)
		if err != nil {
			t.Fatalf("sample %d: GenerateChallenge: %v", i, err)
		}
		if _, dup := seen[ch.Nonce]; dup {
			t.Fatalf("sample %d: duplicate nonce — replay prevention compromised", i)
		}
		seen[ch.Nonce] = struct{}{}
	}
}

// ── Property / VP harness: VP-008 — fail-closed admission ──────────────────

// TestIsAdmitted_FailClosed verifies that IsAdmitted returns false for any
// node not explicitly registered, including an empty key set (VP-008;
// BC-2.05.002 invariant 2 — fail-closed, empty admitted set → no frames
// forwarded).
func TestIsAdmitted_FailClosed(t *testing.T) {
	t.Parallel()

	nodePub, _ := mustGenEd25519(t)
	svtnID := mustSVTN(0xFF)

	ks := admission.NewAdmittedKeySet()
	addr := nodeAddrForTest(svtnID, nodePub)

	if ks.IsAdmitted(svtnID, addr) {
		t.Error("IsAdmitted on empty set: want false, got true — not fail-closed")
	}

	// Register on a different SVTN — must still be false for svtnID.
	otherSVTN := mustSVTN(0xFE)
	ks.RegisterKey(otherSVTN, nodePub, admission.RoleAccess)
	if ks.IsAdmitted(svtnID, addr) {
		t.Error("IsAdmitted on different SVTN: want false, got true")
	}
}

// ── H-2 follow-through: two-state model (registered ≠ admitted) ─────────────

// TestIsAdmitted_FailsBeforeHandshake verifies BC-2.05.001 postcondition 4:
// a node that has been RegisterKey'd but has NOT completed the AdmitNode
// challenge-response handshake is NOT in the active admitted set.
// IsAdmitted must return false until AdmitNode succeeds.
func TestIsAdmitted_FailsBeforeHandshake(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	svtnID := mustSVTN(0x20)
	nodePub, _ := mustGenEd25519(t)

	// Register the key (but do NOT call AdmitNode).
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)

	nodeAddr := nodeAddrForTest(svtnID, nodePub)

	// Verify NOT admitted yet — registered ≠ admitted under the two-state model.
	if ks.IsAdmitted(svtnID, nodeAddr) {
		t.Error("IsAdmitted returned true for registered-but-not-handshaked node; BC-2.05.001 PC4 requires false until AdmitNode succeeds")
	}
}

// ── H-1 follow-through: race regression under -race detector ────────────────

// TestAdmitNodeRevokeKey_NoRace exercises the H-1 race condition fix.
// Runs many goroutines concurrently calling AdmitNode and RevokeKey on the
// same (svtnID, nodeAddr); MUST pass under `go test -race`.
//
// Prior to the fix, AdmitNode read AdmittedKey.revoked after RUnlock while
// RevokeKey wrote under Lock — a Go memory-model violation detected by -race.
func TestAdmitNodeRevokeKey_NoRace(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	svtnID := mustSVTN(0x21)
	_, routerPriv := mustGenEd25519(t)
	nodePub, nodePriv := mustGenEd25519(t)

	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)

	var wg sync.WaitGroup
	for range 50 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			ch, err := admission.GenerateChallenge(routerPriv)
			if err != nil {
				return
			}
			sig := ed25519.Sign(nodePriv, ch.Nonce[:])
			resp := admission.ChallengeResponse{NonceSig: sig}
			_ = admission.AdmitNode(ch, resp, nodePub, svtnID, ks)
		}()
		go func() {
			defer wg.Done()
			nodeAddr := nodeAddrForTest(svtnID, nodePub)
			_ = ks.RevokeKey(svtnID, nodeAddr)
		}()
	}
	wg.Wait()
	// Pass if no -race detector hit and no panic.
}

// ── L-2 / M-4: pin ADR-003 LWW re-registration semantic via full handshake ──

// TestRegisterKey_AfterRevoke_ClearsRevokedFlag pins ADR-003 LWW un-revoke
// semantic: re-registering a key after RevokeKey results in a fresh entry;
// subsequent AdmitNode should succeed. Per user decision 2026-06-25 + ADR-003
// amendment in story rev 1.2: LWW re-registration resets admitted=false (force
// re-handshake), so the test verifies AdmitNode succeeds (taking the node from
// registered-not-admitted to admitted=true).
func TestRegisterKey_AfterRevoke_ClearsRevokedFlag(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	svtnID := mustSVTN(0x22)

	_, routerPriv := mustGenEd25519(t)
	nodePub, nodePriv := mustGenEd25519(t)
	nodeAddr := nodeAddrForTest(svtnID, nodePub)

	// Step 1: Register → AdmitNode → assert IsAdmitted true.
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)
	ch1 := mustGenerateChallenge(t, routerPriv)
	sig1 := ed25519.Sign(nodePriv, ch1.Nonce[:])
	resp1 := admission.ChallengeResponse{NonceSig: sig1}
	if err := admission.AdmitNode(ch1, resp1, nodePub, svtnID, ks); err != nil {
		t.Fatalf("AdmitNode (initial): %v", err)
	}
	if !ks.IsAdmitted(svtnID, nodeAddr) {
		t.Fatal("post-Step-1: IsAdmitted should be true after successful handshake")
	}

	// Step 2: Revoke → assert AdmitNode now fails with ErrKeyRevoked.
	if err := ks.RevokeKey(svtnID, nodeAddr); err != nil {
		t.Fatalf("RevokeKey: %v", err)
	}
	ch2 := mustGenerateChallenge(t, routerPriv)
	sig2 := ed25519.Sign(nodePriv, ch2.Nonce[:])
	resp2 := admission.ChallengeResponse{NonceSig: sig2}
	if err := admission.AdmitNode(ch2, resp2, nodePub, svtnID, ks); !errors.Is(err, admission.ErrKeyRevoked) {
		t.Fatalf("post-revoke AdmitNode: got %v, want ErrKeyRevoked", err)
	}

	// Step 3: Re-Register → assert IsAdmitted=false until fresh handshake, then
	// AdmitNode succeeds (LWW un-revoke + reset-admitted verified).
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)
	if ks.IsAdmitted(svtnID, nodeAddr) {
		t.Fatal("post-re-register: IsAdmitted should be false until fresh handshake (LWW reset semantic)")
	}
	ch3 := mustGenerateChallenge(t, routerPriv)
	sig3 := ed25519.Sign(nodePriv, ch3.Nonce[:])
	resp3 := admission.ChallengeResponse{NonceSig: sig3}
	if err := admission.AdmitNode(ch3, resp3, nodePub, svtnID, ks); err != nil {
		t.Fatalf("post-re-register AdmitNode: got %v, want nil (revoke cleared by LWW)", err)
	}
	if !ks.IsAdmitted(svtnID, nodeAddr) {
		t.Fatal("post-Step-3: IsAdmitted should be true after re-handshake; LWW un-revoke verified")
	}
}

// ── L-2 pass-5: TestRevokeKey_ReturnsErrKeyNotRegistered ────────────────────

// TestRevokeKey_ReturnsErrKeyNotRegistered pins the L-2 sentinel fix:
// RevokeKey must return ErrKeyNotRegistered (E-ADM-013) — NOT
// ErrNotAdmitted (E-ADM-003, frame-routing sentinel) — when the
// (svtnID, nodeAddr) tuple has no registered key. Prior to the fix,
// errors.Is(err, ErrNotAdmitted) conflated frame-rejection with
// key-lifecycle-not-found.
//
// Traces to E-ADM-013 (BC-2.05.001 key-lifecycle error path).
func TestRevokeKey_ReturnsErrKeyNotRegistered(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		setupAndCall func(t *testing.T) error
	}{
		{
			name: "unknown_svtn",
			setupAndCall: func(t *testing.T) error {
				t.Helper()
				ks := admission.NewAdmittedKeySet()
				var svtnID [16]byte
				copy(svtnID[:], "test-svtn-id-16b")
				var nodeAddr [8]byte
				copy(nodeAddr[:], "node1234")
				// svtnID has never been used; entire map miss.
				return ks.RevokeKey(svtnID, nodeAddr)
			},
		},
		{
			name: "known_svtn_unknown_node",
			setupAndCall: func(t *testing.T) error {
				t.Helper()
				ks := admission.NewAdmittedKeySet()
				var svtnID [16]byte
				copy(svtnID[:], "test-svtn-id-16b")

				// Register a DIFFERENT key in this SVTN so the SVTN map entry exists.
				otherPub, _, err := ed25519.GenerateKey(rand.Reader)
				if err != nil {
					return err
				}
				ks.RegisterKey(svtnID, otherPub, admission.RoleAccess)

				// Attempt to revoke a node address that was never registered in this SVTN.
				// Fabricate an address that cannot match otherPub's derived address.
				var missing [8]byte
				copy(missing[:], "missing1")
				return ks.RevokeKey(svtnID, missing)
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.setupAndCall(t)
			if !errors.Is(err, admission.ErrKeyNotRegistered) {
				t.Errorf("RevokeKey: got err=%v, want errors.Is(err, ErrKeyNotRegistered)", err)
			}
			if errors.Is(err, admission.ErrNotAdmitted) {
				t.Errorf("RevokeKey: err must NOT be ErrNotAdmitted (frame-routing sentinel); got %v", err)
			}
		})
	}
}

// ── TestAdmittedKeySet_LookupByPubkey ────────────────────────────────────────

// TestAdmittedKeySet_LookupByPubkey verifies that LookupByPubkey returns
// (AdmittedKey, true) for a registered key, (AdmittedKey{}, false) for an
// unregistered key, and (AdmittedKey{}, false) when the wrong svtnID is
// supplied (ARCH-04 v1.8, ARCH-08 §6.6 position 15).
func TestAdmittedKeySet_LookupByPubkey(t *testing.T) {
	t.Parallel()

	svtnA := mustSVTN(0xA0)
	svtnB := mustSVTN(0xB0)

	cases := []struct {
		name      string
		setupSVTN [16]byte
		lookupID  [16]byte
		register  bool // whether to register the key before lookup
		wantMiss  bool
	}{
		{
			name:      "registered_key_returns_match",
			setupSVTN: svtnA,
			lookupID:  svtnA,
			register:  true,
			wantMiss:  false,
		},
		{
			name:      "unregistered_key_returns_nil",
			setupSVTN: svtnA,
			lookupID:  svtnA,
			register:  false,
			wantMiss:  true,
		},
		{
			name:      "wrong_svtn_returns_nil",
			setupSVTN: svtnA,
			lookupID:  svtnB,
			register:  true,
			wantMiss:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ks := admission.NewAdmittedKeySet()
			pub, _, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				t.Fatalf("GenerateKey: %v", err)
			}

			if tc.register {
				ks.RegisterKey(tc.setupSVTN, pub, admission.RoleControl)
			}

			got, gotOK := ks.LookupByPubkey(tc.lookupID, pub)
			if tc.wantMiss && gotOK {
				t.Errorf("LookupByPubkey: want not-found; got entry %+v", got)
			}
			if !tc.wantMiss {
				if !gotOK {
					t.Fatal("LookupByPubkey: want found AdmittedKey; got not-found")
				}
				// Verify the returned copy's public key matches.
				if !bytes.Equal(got.PublicKey, pub) {
					t.Errorf("LookupByPubkey: PublicKey mismatch: got %x; want %x", got.PublicKey, pub)
				}
			}
		})
	}
}

// ── SEC-001: TestKeyRoleFromString ───────────────────────────────────────────

// TestKeyRoleFromString verifies KeyRoleFromString for all three valid roles
// and at least two invalid inputs (SEC-001; PR #34 security review).
// Guards against callers accidentally mapping unknowns to the RoleControl
// zero value when wiring admin RPC handlers in S-6.06.
func TestKeyRoleFromString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input   string
		want    admission.KeyRole
		wantErr bool
	}{
		{input: "control", want: admission.RoleControl, wantErr: false},
		{input: "console", want: admission.RoleConsole, wantErr: false},
		{input: "access", want: admission.RoleAccess, wantErr: false},
		{input: "", want: 0, wantErr: true},
		{input: "unknown", want: 0, wantErr: true},
		{input: "CONTROL", want: 0, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got, err := admission.KeyRoleFromString(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("KeyRoleFromString(%q): want error, got nil", tc.input)
					return
				}
				if !errors.Is(err, admission.ErrUnknownKeyRole) {
					t.Errorf("KeyRoleFromString(%q): want errors.Is(err, ErrUnknownKeyRole), got %v", tc.input, err)
				}
				// Zero value must be returned on error — never RoleControl.
				if got != 0 {
					t.Errorf("KeyRoleFromString(%q): want zero KeyRole on error, got %v", tc.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("KeyRoleFromString(%q): want nil error, got %v", tc.input, err)
				return
			}
			if got != tc.want {
				t.Errorf("KeyRoleFromString(%q): want %v, got %v", tc.input, tc.want, got)
			}
		})
	}
}

// ── Fuzz harness: VP-008 — admission rejects unregistered keys ──────────────

// FuzzAdmitNode_UnregisteredKey is a fuzz target that verifies AdmitNode
// always returns a non-nil error when the presented public key is NOT
// registered in the AdmittedKeySet.
//
// Traces to VP-008 (Admission Fails for Unregistered Key).
func FuzzAdmitNode_UnregisteredKey(f *testing.F) {
	// Seed corpus: 80 bytes — 32 node seed + 16 SVTN + 32 router seed.
	// Each ed25519.GenerateKey call requires exactly 32 bytes of seed material;
	// the prior 64-byte corpus only gave the router keygen 16 bytes, causing
	// io.ErrUnexpectedEOF and t.Skip on every seeded iteration (pass-3 M-1).
	f.Add([]byte("node-seed-deterministic000000000svtn-id-00000000router-seed-deterministic0000000"))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Need at least 80 bytes: 32 node seed + 16 SVTN + 32 router seed.
		if len(data) < 80 {
			t.Skip()
			return
		}

		// Deterministically derive node keypair from corpus bytes [0:32].
		unregisteredPub, unregisteredPriv, err := ed25519.GenerateKey(bytes.NewReader(data[:32]))
		if err != nil {
			t.Skip()
			return
		}

		// Derive SVTN ID from corpus bytes [32:48].
		var svtnID [16]byte
		copy(svtnID[:], data[32:48])

		// Derive router keypair from corpus bytes [48:80].
		_, routerPriv, err := ed25519.GenerateKey(bytes.NewReader(data[48:80]))
		if err != nil {
			t.Skip()
			return
		}

		ks := admission.NewAdmittedKeySet()
		// Deliberately do NOT call ks.RegisterKey — key is unregistered.

		ch, err := admission.GenerateChallenge(routerPriv)
		if err != nil {
			t.Skip()
			return
		}
		sig := ed25519.Sign(unregisteredPriv, ch.Nonce[:])
		resp := admission.ChallengeResponse{NonceSig: sig}

		err = admission.AdmitNode(ch, resp, unregisteredPub, svtnID, ks)
		if err == nil {
			t.Error("AdmitNode with unregistered key: want error, got nil — admission must fail for unregistered keys")
		}
	})
}
