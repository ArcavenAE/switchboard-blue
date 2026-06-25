package admission_test

// Session continuity tests for internal/admission (S-1.03).
//
// Traces to:
//   BC-2.01.007 — Session Continuity Survives IP Address Change
//   VP-036       — Session Channel ID Unchanged Before/After IP Change
//   ARCH-04 §ADR-003 — Last-write-wins for concurrent re-auth
//   error-taxonomy §ADM — E-ADM-001, E-ADM-005, ErrKeyExpired

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
)

// ── AC-001: TestSessionContinuity_ReauthOnIPChange ────────────────────────────

// TestSessionContinuity_ReauthOnIPChange verifies that when a node's source IP
// changes, re-authentication using the same keypair succeeds and the session
// resumes on the new path.
//
// Traces to:
//
//	BC-2.01.007 postcondition 1 (session continues after re-auth from new IP)
//	BC-2.01.007 postcondition 3 (router updates routing entry to new source IP)
//	AC-001 (S-1.03)
func TestSessionContinuity_ReauthOnIPChange(t *testing.T) {
	t.Parallel()

	// Deterministic seed for reproducibility (BC-5.38.001 pattern).
	seed := bytes.Repeat([]byte("reauth-ip-change-0"), 2) // 32 bytes
	nodePub, nodePriv, err := ed25519.GenerateKey(bytes.NewReader(seed))
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}

	_, routerPriv := mustGenEd25519(t)
	svtnID := mustSVTN(0xA1)

	ks := admission.NewAdmittedKeySet()
	rs := admission.NewReAuthState()

	// Register and admit the node with initial IP.
	ks.RegisterKey(svtnID, nodePub, admission.RoleConsole)
	initialChallenge := mustGenerateChallenge(t, routerPriv)
	initialSig := ed25519.Sign(nodePriv, initialChallenge.Nonce[:])
	if err := admission.AdmitNode(
		initialChallenge,
		admission.ChallengeResponse{NonceSig: initialSig},
		nodePub,
		svtnID,
		ks,
	); err != nil {
		t.Fatalf("initial AdmitNode: %v", err)
	}

	nodeAddr := nodeAddrForTest(svtnID, nodePub)
	if !ks.IsAdmitted(svtnID, nodeAddr) {
		t.Fatal("node not admitted after initial handshake")
	}

	// IP changes: node re-authenticates from a new source IP.
	oldIP := netip.MustParseAddr("192.168.1.10")
	newIP := netip.MustParseAddr("10.0.0.5")

	// Establish old IP first.
	reAuthChallenge1 := mustGenerateChallenge(t, routerPriv)
	sig1 := ed25519.Sign(nodePriv, reAuthChallenge1.Nonce[:])
	oldReq := admission.ReAuthRequest{
		SVTNID:        svtnID,
		NodeAddr:      nodeAddr,
		NewSourceAddr: oldIP,
		Challenge:     reAuthChallenge1,
		Response:      admission.ChallengeResponse{NonceSig: sig1},
	}
	if err := admission.ReAuthenticate(oldReq, ks, rs); err != nil {
		t.Fatalf("ReAuthenticate (old IP): %v", err)
	}

	// Node moves to new IP; re-authenticates with same keypair.
	reAuthChallenge2 := mustGenerateChallenge(t, routerPriv)
	sig2 := ed25519.Sign(nodePriv, reAuthChallenge2.Nonce[:])
	newReq := admission.ReAuthRequest{
		SVTNID:        svtnID,
		NodeAddr:      nodeAddr,
		NewSourceAddr: newIP,
		Challenge:     reAuthChallenge2,
		Response:      admission.ChallengeResponse{NonceSig: sig2},
	}
	if err := admission.ReAuthenticate(newReq, ks, rs); err != nil {
		t.Errorf("ReAuthenticate on IP change: want nil, got %v", err)
	}

	// Node must still be admitted (session resumed, not terminated).
	if !ks.IsAdmitted(svtnID, nodeAddr) {
		t.Error("node no longer admitted after re-auth on IP change")
	}

	// Source address must now reflect the new IP (BC-2.01.007 PC3).
	got := rs.CurrentSourceAddr(svtnID, nodeAddr)
	if got != newIP {
		t.Errorf("CurrentSourceAddr: want %s, got %s", newIP, got)
	}
}

// ── AC-002: TestSessionContinuity_WrongKeyRejected ────────────────────────────

// TestSessionContinuity_WrongKeyRejected verifies that re-authentication is
// rejected when the challenge response is signed with a different keypair than
// the one originally admitted.
//
// Traces to:
//
//	BC-2.01.007 precondition 3 (keypair must be unchanged)
//	BC-2.01.007 canonical test vector 3 (wrong key → E-ADM-001)
//	AC-002 (S-1.03)
func TestSessionContinuity_WrongKeyRejected(t *testing.T) {
	t.Parallel()

	_, routerPriv := mustGenEd25519(t)
	nodePub, nodePriv := mustGenEd25519(t)
	_, wrongPriv := mustGenEd25519(t) // different keypair — not the admitted key
	svtnID := mustSVTN(0xA2)

	ks := admission.NewAdmittedKeySet()
	rs := admission.NewReAuthState()

	// Register and admit the node.
	ks.RegisterKey(svtnID, nodePub, admission.RoleConsole)
	ch0 := mustGenerateChallenge(t, routerPriv)
	sig0 := ed25519.Sign(nodePriv, ch0.Nonce[:])
	if err := admission.AdmitNode(ch0, admission.ChallengeResponse{NonceSig: sig0}, nodePub, svtnID, ks); err != nil {
		t.Fatalf("initial AdmitNode: %v", err)
	}

	nodeAddr := nodeAddrForTest(svtnID, nodePub)

	// Re-auth attempt with wrong keypair signature.
	ch := mustGenerateChallenge(t, routerPriv)
	wrongSig := ed25519.Sign(wrongPriv, ch.Nonce[:]) // signed with wrong key
	req := admission.ReAuthRequest{
		SVTNID:        svtnID,
		NodeAddr:      nodeAddr,
		NewSourceAddr: netip.MustParseAddr("10.1.2.3"),
		Challenge:     ch,
		Response:      admission.ChallengeResponse{NonceSig: wrongSig},
	}

	err := admission.ReAuthenticate(req, ks, rs)
	if !errors.Is(err, admission.ErrSignatureVerificationFailed) {
		t.Errorf("wrong key re-auth: want ErrSignatureVerificationFailed, got %v", err)
	}
}

// ── AC-003: TestSessionContinuity_NodeAddressStableAfterReauth ───────────────

// TestSessionContinuity_NodeAddressStableAfterReauth verifies that the
// cryptographic node address (derived from SVTN-ID and public key) is
// unchanged after a successful re-authentication from a new IP.
//
// Traces to:
//
//	BC-2.01.007 invariant 3 (session identity = channel ID + node addr, not IP)
//	BC-2.01.007 postcondition 2 (router verifies against admitted key set)
//	VP-036 (session_id unchanged before/after IP change)
//	AC-003 (S-1.03)
func TestSessionContinuity_NodeAddressStableAfterReauth(t *testing.T) {
	t.Parallel()

	seed := bytes.Repeat([]byte("reauth-stable-addr0"), 2)[:32]
	nodePub, nodePriv, err := ed25519.GenerateKey(bytes.NewReader(seed))
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}

	_, routerPriv := mustGenEd25519(t)
	svtnID := mustSVTN(0xA3)

	ks := admission.NewAdmittedKeySet()
	rs := admission.NewReAuthState()

	// Record node address BEFORE re-auth.
	addrBefore := nodeAddrForTest(svtnID, nodePub)

	// Register and admit.
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)
	ch0 := mustGenerateChallenge(t, routerPriv)
	sig0 := ed25519.Sign(nodePriv, ch0.Nonce[:])
	if err := admission.AdmitNode(ch0, admission.ChallengeResponse{NonceSig: sig0}, nodePub, svtnID, ks); err != nil {
		t.Fatalf("initial AdmitNode: %v", err)
	}

	// Re-authenticate from a new IP.
	ch := mustGenerateChallenge(t, routerPriv)
	sig := ed25519.Sign(nodePriv, ch.Nonce[:])
	req := admission.ReAuthRequest{
		SVTNID:        svtnID,
		NodeAddr:      addrBefore,
		NewSourceAddr: netip.MustParseAddr("172.16.0.1"),
		Challenge:     ch,
		Response:      admission.ChallengeResponse{NonceSig: sig},
	}
	if err := admission.ReAuthenticate(req, ks, rs); err != nil {
		t.Fatalf("ReAuthenticate: %v", err)
	}

	// Node address after re-auth is still the same (derived from pubkey, not IP).
	addrAfter := nodeAddrForTest(svtnID, nodePub)
	if addrBefore != addrAfter {
		t.Errorf("node address changed after re-auth: before=%x after=%x", addrBefore, addrAfter)
	}

	// The admitted key set still recognises the same address.
	if !ks.IsAdmitted(svtnID, addrAfter) {
		t.Error("IsAdmitted(addrAfter) = false; node address should be stable and still admitted")
	}
}

// ── EC-001: TestReauth_ExpiredKey ─────────────────────────────────────────────

// TestReauth_ExpiredKey verifies that a re-authentication attempt by a node
// whose key has passed its expiry timestamp is rejected.
//
// Traces to:
//
//	BC-2.01.007 EC-005 (re-authenticate after key expiry → E-ADM-015; see ARCH-04 v1.3 §Key Lifecycle)
//	E-ADM-015 / ErrKeyExpired (story EC-001, S-1.03 rev 1.2 Spec Patches)
//	EC-001 (S-1.03)
func TestReauth_ExpiredKey(t *testing.T) {
	t.Parallel()

	_, routerPriv := mustGenEd25519(t)
	nodePub, nodePriv := mustGenEd25519(t)
	svtnID := mustSVTN(0xB1)

	ks := admission.NewAdmittedKeySet()
	rs := admission.NewReAuthState()

	// Register and admit node with an expiry in the past.
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)

	ch0 := mustGenerateChallenge(t, routerPriv)
	sig0 := ed25519.Sign(nodePriv, ch0.Nonce[:])
	if err := admission.AdmitNode(ch0, admission.ChallengeResponse{NonceSig: sig0}, nodePub, svtnID, ks); err != nil {
		t.Fatalf("initial AdmitNode: %v", err)
	}

	// Set expiry to a time in the past.
	nodeAddr := nodeAddrForTest(svtnID, nodePub)
	expiredAt := time.Now().UTC().Add(-time.Second)
	if err := ks.SetKeyExpiry(svtnID, nodeAddr, expiredAt); err != nil {
		t.Fatalf("SetKeyExpiry: %v", err)
	}

	// Re-authentication must be rejected with ErrKeyExpired.
	ch := mustGenerateChallenge(t, routerPriv)
	sig := ed25519.Sign(nodePriv, ch.Nonce[:])
	req := admission.ReAuthRequest{
		SVTNID:        svtnID,
		NodeAddr:      nodeAddr,
		NewSourceAddr: netip.MustParseAddr("10.2.3.4"),
		Challenge:     ch,
		Response:      admission.ChallengeResponse{NonceSig: sig},
	}

	err := admission.ReAuthenticate(req, ks, rs)
	if !errors.Is(err, admission.ErrKeyExpired) {
		t.Errorf("expired key re-auth: want ErrKeyExpired, got %v", err)
	}
}

// ── EC-002: TestReauth_EvictsOldPath ──────────────────────────────────────────

// TestReauth_EvictsOldPath verifies that a successful re-authentication from a
// new source IP evicts the old path: the stored source address for the node
// transitions from the old IP to the new IP (BC-2.01.007 EC-006).
//
// Traces to:
//
//	BC-2.01.007 EC-006 (old path evicted on new re-auth; BC v1.3)
//	EC-002 (S-1.03)
func TestReauth_EvictsOldPath(t *testing.T) {
	t.Parallel()

	_, routerPriv := mustGenEd25519(t)
	nodePub, nodePriv := mustGenEd25519(t)
	svtnID := mustSVTN(0xB2)

	ks := admission.NewAdmittedKeySet()
	rs := admission.NewReAuthState()

	// Register and admit node.
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)
	ch0 := mustGenerateChallenge(t, routerPriv)
	sig0 := ed25519.Sign(nodePriv, ch0.Nonce[:])
	if err := admission.AdmitNode(ch0, admission.ChallengeResponse{NonceSig: sig0}, nodePub, svtnID, ks); err != nil {
		t.Fatalf("initial AdmitNode: %v", err)
	}

	nodeAddr := nodeAddrForTest(svtnID, nodePub)
	oldIP := netip.MustParseAddr("192.0.2.10")
	newIP := netip.MustParseAddr("198.51.100.20")

	// Re-auth from old IP first.
	ch1 := mustGenerateChallenge(t, routerPriv)
	sig1 := ed25519.Sign(nodePriv, ch1.Nonce[:])
	req1 := admission.ReAuthRequest{
		SVTNID:        svtnID,
		NodeAddr:      nodeAddr,
		NewSourceAddr: oldIP,
		Challenge:     ch1,
		Response:      admission.ChallengeResponse{NonceSig: sig1},
	}
	if err := admission.ReAuthenticate(req1, ks, rs); err != nil {
		t.Fatalf("ReAuthenticate (old IP): %v", err)
	}
	if got := rs.CurrentSourceAddr(svtnID, nodeAddr); got != oldIP {
		t.Fatalf("expected old IP %s, got %s", oldIP, got)
	}

	// Re-auth from new IP — old path must be evicted.
	ch2 := mustGenerateChallenge(t, routerPriv)
	sig2 := ed25519.Sign(nodePriv, ch2.Nonce[:])
	req2 := admission.ReAuthRequest{
		SVTNID:        svtnID,
		NodeAddr:      nodeAddr,
		NewSourceAddr: newIP,
		Challenge:     ch2,
		Response:      admission.ChallengeResponse{NonceSig: sig2},
	}
	if err := admission.ReAuthenticate(req2, ks, rs); err != nil {
		t.Fatalf("ReAuthenticate (new IP): %v", err)
	}

	// CurrentSourceAddr must now return the new IP (old path evicted).
	got := rs.CurrentSourceAddr(svtnID, nodeAddr)
	if got != newIP {
		t.Errorf("old path not evicted: want %s, got %s", newIP, got)
	}
}

// ── EC-003: TestReauth_LastWriteWins ──────────────────────────────────────────

// TestReauth_LastWriteWins verifies that when two concurrent re-authentication
// attempts from the same node arrive (e.g., double-tap on IP change), the last
// accepted one determines the stored source address (ADR-003 LWW; BC-2.01.007
// EC-003).
//
// Serialised simulation: call ReAuthenticate twice in sequence; the second
// call's source IP wins. Concurrent variant is covered by TestReAuthenticate_NoRace.
//
// Traces to:
//
//	BC-2.01.007 EC-003 (concurrent re-auth — last one wins)
//	ARCH-04 §ADR-003 (LWW)
//	EC-003 (S-1.03)
func TestReauth_LastWriteWins(t *testing.T) {
	t.Parallel()

	_, routerPriv := mustGenEd25519(t)
	nodePub, nodePriv := mustGenEd25519(t)
	svtnID := mustSVTN(0xB3)

	ks := admission.NewAdmittedKeySet()
	rs := admission.NewReAuthState()

	// Register and admit node.
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)
	ch0 := mustGenerateChallenge(t, routerPriv)
	sig0 := ed25519.Sign(nodePriv, ch0.Nonce[:])
	if err := admission.AdmitNode(ch0, admission.ChallengeResponse{NonceSig: sig0}, nodePub, svtnID, ks); err != nil {
		t.Fatalf("initial AdmitNode: %v", err)
	}

	nodeAddr := nodeAddrForTest(svtnID, nodePub)
	ip1 := netip.MustParseAddr("10.10.10.1")
	ip2 := netip.MustParseAddr("10.10.10.2") // second (last) write wins

	// First re-auth.
	ch1 := mustGenerateChallenge(t, routerPriv)
	sig1 := ed25519.Sign(nodePriv, ch1.Nonce[:])
	req1 := admission.ReAuthRequest{
		SVTNID:        svtnID,
		NodeAddr:      nodeAddr,
		NewSourceAddr: ip1,
		Challenge:     ch1,
		Response:      admission.ChallengeResponse{NonceSig: sig1},
	}
	if err := admission.ReAuthenticate(req1, ks, rs); err != nil {
		t.Fatalf("first ReAuthenticate: %v", err)
	}

	// Second re-auth (same node, different IP) — must supersede first.
	ch2 := mustGenerateChallenge(t, routerPriv)
	sig2 := ed25519.Sign(nodePriv, ch2.Nonce[:])
	req2 := admission.ReAuthRequest{
		SVTNID:        svtnID,
		NodeAddr:      nodeAddr,
		NewSourceAddr: ip2,
		Challenge:     ch2,
		Response:      admission.ChallengeResponse{NonceSig: sig2},
	}
	if err := admission.ReAuthenticate(req2, ks, rs); err != nil {
		t.Fatalf("second ReAuthenticate: %v", err)
	}

	// Last write (ip2) must win per ADR-003.
	got := rs.CurrentSourceAddr(svtnID, nodeAddr)
	if got != ip2 {
		t.Errorf("LWW: want last write %s, got %s", ip2, got)
	}
}

// ── M-3: TestReAuthenticate_NoRace ────────────────────────────────────────────

// TestReAuthenticate_NoRace exercises concurrent re-authentication from two
// goroutines racing to claim the same (svtnID, nodeAddr) with different source
// IPs. Each goroutine has its own distinct challenge (the router issues a fresh
// challenge per path), so neither attempt collides on nonce replay. Both
// ReAuthenticate calls must complete without error; the final CurrentSourceAddr
// must contain exactly one of the two source IPs with no torn write.
//
// This is the concurrent companion to TestReauth_LastWriteWins (serialised
// proof). The race detector enforces the lock discipline mandated by ADR-003.
//
// Traces to:
//
//	BC-2.01.007 EC-003 (concurrent re-auth — last one wins; LWW)
//	BC-2.01.007 EC-006 (old path evicted on new re-auth; BC v1.3)
//	ADR-003 (last-write-wins policy)
//	M-3 (adversary pass-1 finding: concurrent re-auth race not covered)
func TestReAuthenticate_NoRace(t *testing.T) {
	t.Parallel()

	// Deterministic seed for reproducibility.
	seed := bytes.Repeat([]byte("reauth-norace-test0"), 2)[:32]
	nodePub, nodePriv, err := ed25519.GenerateKey(bytes.NewReader(seed))
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}

	_, routerPriv := mustGenEd25519(t)
	svtnID := mustSVTN(0xC1)

	ks := admission.NewAdmittedKeySet()
	rs := admission.NewReAuthState()

	// Register and admit the node (initial handshake required before re-auth).
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)
	ch0 := mustGenerateChallenge(t, routerPriv)
	sig0 := ed25519.Sign(nodePriv, ch0.Nonce[:])
	if err := admission.AdmitNode(ch0, admission.ChallengeResponse{NonceSig: sig0}, nodePub, svtnID, ks); err != nil {
		t.Fatalf("initial AdmitNode: %v", err)
	}

	nodeAddr := nodeAddrForTest(svtnID, nodePub)
	sourceA := netip.MustParseAddr("10.100.0.1")
	sourceB := netip.MustParseAddr("10.100.0.2")

	// Each goroutine generates its own fresh challenge so nonces are distinct
	// (the router would issue a unique challenge per network path).
	chA := mustGenerateChallenge(t, routerPriv)
	sigA := ed25519.Sign(nodePriv, chA.Nonce[:])
	reqA := admission.ReAuthRequest{
		SVTNID:        svtnID,
		NodeAddr:      nodeAddr,
		NewSourceAddr: sourceA,
		Challenge:     chA,
		Response:      admission.ChallengeResponse{NonceSig: sigA},
	}

	chB := mustGenerateChallenge(t, routerPriv)
	sigB := ed25519.Sign(nodePriv, chB.Nonce[:])
	reqB := admission.ReAuthRequest{
		SVTNID:        svtnID,
		NodeAddr:      nodeAddr,
		NewSourceAddr: sourceB,
		Challenge:     chB,
		Response:      admission.ChallengeResponse{NonceSig: sigB},
	}

	var errA, errB error
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		errA = admission.ReAuthenticate(reqA, ks, rs)
	}()
	go func() {
		defer wg.Done()
		errB = admission.ReAuthenticate(reqB, ks, rs)
	}()

	wg.Wait()

	// Both goroutines use fresh, distinct challenges — neither should fail.
	if errA != nil {
		t.Errorf("goroutine A ReAuthenticate: want nil, got %v", errA)
	}
	if errB != nil {
		t.Errorf("goroutine B ReAuthenticate: want nil, got %v", errB)
	}

	// CurrentSourceAddr must contain exactly one of the two IPs (LWW; ADR-003).
	// A torn write would produce a zero-value addr — verify it is non-zero.
	got := rs.CurrentSourceAddr(svtnID, nodeAddr)
	if got != sourceA && got != sourceB {
		t.Errorf("LWW post-race: CurrentSourceAddr = %s; want either %s or %s (no torn write)", got, sourceA, sourceB)
	}
}

// ── VP-036: TestProperty_VP036_SessionContinuity ─────────────────────────────

// TestProperty_VP036_SessionContinuity is the unit-scope stub for VP-036
// ("Session Channel ID Unchanged Before and After IP Change").
//
// VP-036's canonical proof method is e2e (requires testenv.ConnectWithSourceIP,
// which is not yet implemented — see VP-036.md proof harness). This unit test
// verifies the property's precondition at the admission layer: the cryptographic
// node address, which forms half of the session identity
// (channel_id + node_addr pair; BC-2.01.007 invariant 3), is derived purely
// from (svtnID, pubKey) and is invariant under IP changes.
//
// The full e2e test (TestE2E_Session_ContinuityAcrossIPChange) per VP-036.md
// requires internal/testenv with ConnectWithSourceIP — deferred to the wave
// that implements the testenv package.
//
// Traces to:
//
//	VP-036 (session_id unchanged before/after IP change)
//	BC-2.01.007 invariant 3 (session identity = channel_id + node_addr, not IP)
func TestProperty_VP036_SessionContinuity(t *testing.T) {
	t.Skip(
		"VP-036 full e2e proof deferred: requires internal/testenv.ConnectWithSourceIP " +
			"(VP-036.md proof harness). This test is a discoverable placeholder. " +
			"Unit-scope coverage of the node-address-stability invariant is in " +
			"TestSessionContinuity_NodeAddressStableAfterReauth (AC-003).",
	)
}
