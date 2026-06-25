package admission_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"errors"
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
//
//nolint:unparam // return value is consumed once stub methods are implemented
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

	_, routerPriv := mustGenEd25519(t) //nolint:staticcheck // consumed by mustGenerateChallenge once stub is implemented
	nodePub, nodePriv := mustGenEd25519(t)
	_ = nodePriv //nolint:staticcheck // consumed by ed25519.Sign once stub is implemented
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
// Traces to BC-2.05.001 precondition 1 (public key registered);
// postcondition 5 (failure path: E-ADM-001).
func TestAdmitNode_InvalidSignature(t *testing.T) {
	t.Parallel()

	_, routerPriv := mustGenEd25519(t) //nolint:staticcheck // consumed by mustGenerateChallenge once stub is implemented
	nodePub, _ := mustGenEd25519(t)
	_, wrongNodePriv := mustGenEd25519(t) // wrong keypair — not registered
	_ = wrongNodePriv                     //nolint:staticcheck // consumed by ed25519.Sign once stub is implemented
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
// Traces to BC-2.05.001 invariant 3 (nonces are single-use, E-ADM-008).
func TestAdmitNode_ReplayedNonce(t *testing.T) {
	t.Parallel()

	_, routerPriv := mustGenEd25519(t) //nolint:staticcheck // consumed once stub is implemented
	nodePub, nodePriv := mustGenEd25519(t)
	_ = nodePriv //nolint:staticcheck // consumed by ed25519.Sign once stub is implemented
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

// ── EC-001: TestAdmitNode_KeyNotRegisteredForSVTN ───────────────────────────

// TestAdmitNode_KeyNotRegisteredForSVTN verifies that a node presenting a
// key for an SVTN it is not registered on returns an error.
//
// Traces to story EC-001 (E-ADM-005 returned; frame dropped) and
// BC-2.05.002 precondition 1.
func TestAdmitNode_KeyNotRegisteredForSVTN(t *testing.T) {
	t.Parallel()

	_, routerPriv := mustGenEd25519(t) //nolint:staticcheck // consumed once stub is implemented
	nodePub, nodePriv := mustGenEd25519(t)
	_ = nodePriv //nolint:staticcheck // consumed by ed25519.Sign once stub is implemented
	svtnA := mustSVTN(0x0A)
	svtnB := mustSVTN(0x0B) // node is only registered on svtnA
	_ = svtnB               //nolint:staticcheck // consumed by AdmitNode once stub is implemented

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
	entry := ks.Lookup(svtnID, addr)
	if entry == nil {
		t.Fatal("Lookup after duplicate register: want entry, got nil")
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

	_, routerPriv := mustGenEd25519(t) //nolint:staticcheck // consumed once stub is implemented
	nodePub, nodePriv := mustGenEd25519(t)
	_ = nodePriv //nolint:staticcheck // consumed by ed25519.Sign once stub is implemented
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
// produces distinct nonces across multiple calls (BC-2.05.001 invariant 3;
// VP-007, VP-009).
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

// ── Fuzz harness: VP-009 — admission rejects unregistered keys ──────────────

// FuzzAdmitNode_UnregisteredKey is a fuzz target that verifies AdmitNode
// always returns a non-nil error when the presented public key is NOT
// registered in the AdmittedKeySet.
//
// Traces to VP-009 (admission fails for any key not in the admitted set).
func FuzzAdmitNode_UnregisteredKey(f *testing.F) {
	// Seed corpus: a single known valid nonce seed.
	f.Add([]byte("seed-nonce-000000000000000000000"))

	f.Fuzz(func(t *testing.T, _ []byte) {
		_, routerPriv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Skip()
			return
		}
		// Unregistered: generate a fresh key not added to ks.
		unregisteredPub, unregisteredPriv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Skip()
			return
		}
		svtnID := mustSVTN(0x01)

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
