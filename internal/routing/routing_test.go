package routing_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/hmac"
	"github.com/arcavenae/switchboard/internal/routing"
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

// nodeAddrForTest mirrors frame.DeriveNodeAddress: SHA-256(svtnID || pubKey)[:8].
func nodeAddrForTest(svtnID [16]byte, pubKey ed25519.PublicKey) [8]byte {
	h := sha256.New()
	h.Write(svtnID[:])
	h.Write([]byte(pubKey))
	sum := h.Sum(nil)
	var addr [8]byte
	copy(addr[:], sum[:8])
	return addr
}

// ── AC-004: TestRouteFrame_DropsUnadmitted ───────────────────────────────────

// TestRouteFrame_DropsUnadmitted verifies that RouteFrame returns
// admission.ErrNotAdmitted and drops the frame when the frame's src_addr
// is not in the admitted set for the frame's svtn_id.
//
// Traces to BC-2.05.002 postcondition 2 (frame from non-admitted source →
// dropped; E-ADM-003 logged).
func TestRouteFrame_DropsUnadmitted(t *testing.T) {
	t.Parallel()

	svtnID := mustSVTN(0x01)
	unadmittedPub, _ := mustGenEd25519(t)
	unadmittedAddr := nodeAddrForTest(svtnID, unadmittedPub)

	admittedPub, _ := mustGenEd25519(t)
	admittedAddr := nodeAddrForTest(svtnID, admittedPub)

	ks := admission.NewAdmittedKeySet()
	// Only admittedPub is registered — unadmittedPub is NOT.
	ks.RegisterKey(svtnID, admittedPub, admission.RoleAccess)

	r := routing.NewRouter(ks)
	r.RegisterForwardingEntry(svtnID, admittedAddr, [32]byte{})

	// S-3.04: HMAC is now enforced before admission. Register a forwarding entry
	// for unadmittedAddr so the auth key is present; compute a valid tag with
	// that key so HMAC passes and the admission check (ErrNotAdmitted) fires.
	var unadmittedAuthKey [hmac.KeySize]byte
	copy(unadmittedAuthKey[:], "drops-unadmitted-test-key-00000")
	r.RegisterForwardingEntry(svtnID, unadmittedAddr, unadmittedAuthKey)

	// Frame from unadmitted source with valid HMAC (so admission check fires).
	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtnID,
		SrcAddr:   unadmittedAddr,
		DstAddr:   admittedAddr,
	}
	hdr.HMACTag = computeValidTag(hdr, nil, unadmittedAuthKey)
	err := routing.RouteFrame(hdr, nil, r)
	if !errors.Is(err, admission.ErrNotAdmitted) {
		t.Errorf("RouteFrame unadmitted src: want ErrNotAdmitted, got %v", err)
	}
}

// ── AC-005: TestSVTNRoute_NoCrossContamination ───────────────────────────────

// TestSVTNRoute_NoCrossContamination verifies that SVTNRoute never delivers a
// frame to a node on a different SVTN: a frame with svtn_id=A is never
// forwarded to a node admitted only to svtn_id=B.
//
// Traces to BC-2.05.006 postcondition 1 (node receives only frames with SVTN ID
// matching its admitted SVTN) and postcondition 2 (forwarding engine partitions
// by SVTN ID).
func TestSVTNRoute_NoCrossContamination(t *testing.T) {
	t.Parallel()

	svtnA := mustSVTN(0x0A)
	svtnB := mustSVTN(0x0B)

	nodeAPub, _ := mustGenEd25519(t)
	nodeBPub, _ := mustGenEd25519(t)
	nodeAAddr := nodeAddrForTest(svtnA, nodeAPub)
	nodeBAddr := nodeAddrForTest(svtnB, nodeBPub)

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnA, nodeAPub, admission.RoleAccess)
	ks.RegisterKey(svtnB, nodeBPub, admission.RoleAccess)

	r := routing.NewRouter(ks)
	r.RegisterForwardingEntry(svtnA, nodeAAddr, [32]byte{0x01})
	r.RegisterForwardingEntry(svtnB, nodeBAddr, [32]byte{0x02})

	// Frame for SVTN-A addressed to nodeBAddr (which is in SVTN-B only).
	// SVTNRoute must not deliver it — nodeBAddr is not in SVTN-A's table.
	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtnA,
		SrcAddr:   nodeAAddr,
		DstAddr:   nodeBAddr,
	}
	err := routing.SVTNRoute(hdr, nil, r)
	if err == nil {
		t.Error("SVTNRoute cross-SVTN: want error (nodeBAddr not in svtnA forwarding table), got nil")
	}
}

// ── Property / VP harness: VP-010 — SVTN isolation boundary ─────────────────

// TestSVTNRoute_SVTNPartitionBoundary verifies that a router serving N SVTNs
// forwards frames only to nodes admitted to the correct SVTN partition.
//
// Table-driven: 3 SVTN pairs, each asserting no cross-contamination.
// Traces to BC-2.05.006 postcondition 2; VP-010 (SVTN isolation under
// all router configurations).
func TestSVTNRoute_SVTNPartitionBoundary(t *testing.T) {
	t.Parallel()

	type pair struct {
		srcSVTN [16]byte
		dstSVTN [16]byte
		name    string
	}
	cases := []pair{
		{mustSVTN(0x01), mustSVTN(0x02), "SVTN-1 to SVTN-2"},
		{mustSVTN(0x02), mustSVTN(0x03), "SVTN-2 to SVTN-3"},
		{mustSVTN(0xAA), mustSVTN(0xBB), "SVTN-AA to SVTN-BB"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srcPub, _ := mustGenEd25519(t)
			dstPub, _ := mustGenEd25519(t)
			srcAddr := nodeAddrForTest(tc.srcSVTN, srcPub)
			dstAddr := nodeAddrForTest(tc.dstSVTN, dstPub)

			ks := admission.NewAdmittedKeySet()
			ks.RegisterKey(tc.srcSVTN, srcPub, admission.RoleAccess)
			ks.RegisterKey(tc.dstSVTN, dstPub, admission.RoleAccess)

			r := routing.NewRouter(ks)
			r.RegisterForwardingEntry(tc.srcSVTN, srcAddr, [32]byte{0x01})
			r.RegisterForwardingEntry(tc.dstSVTN, dstAddr, [32]byte{0x02})

			// Frame for tc.srcSVTN trying to reach dstAddr (only in tc.dstSVTN).
			hdr := frame.OuterHeader{
				Version:   frame.VersionByte,
				FrameType: frame.FrameTypeData,
				SVTNID:    tc.srcSVTN,
				SrcAddr:   srcAddr,
				DstAddr:   dstAddr,
			}
			err := routing.SVTNRoute(hdr, nil, r)
			if err == nil {
				t.Errorf("%s: SVTNRoute cross-SVTN delivery: want error, got nil — SVTN isolation violated", tc.name)
			}
		})
	}
}

// ── Property / VP harness: VP-039 — SVTN isolation end-to-end ───────────────

// TestSVTNRoute_AdmittedFrameForwardedToCorrectSVTN verifies the happy path:
// a frame for SVTN-A is forwarded to a node admitted to SVTN-A when the
// destination is registered in SVTN-A's forwarding table.
//
// Traces to BC-2.05.006 postcondition 1; VP-039 (no cross-SVTN traffic under
// any router configuration).
func TestSVTNRoute_AdmittedFrameForwardedToCorrectSVTN(t *testing.T) {
	t.Parallel()

	svtnA := mustSVTN(0x0A)
	srcPub, _ := mustGenEd25519(t)
	dstPub, _ := mustGenEd25519(t)
	srcAddr := nodeAddrForTest(svtnA, srcPub)
	dstAddr := nodeAddrForTest(svtnA, dstPub)

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnA, srcPub, admission.RoleAccess)
	ks.RegisterKey(svtnA, dstPub, admission.RoleAccess)

	r := routing.NewRouter(ks)
	r.RegisterForwardingEntry(svtnA, srcAddr, [32]byte{0x01})
	r.RegisterForwardingEntry(svtnA, dstAddr, [32]byte{0x02})

	// Frame within SVTN-A: src → dst, both admitted.
	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtnA,
		SrcAddr:   srcAddr,
		DstAddr:   dstAddr,
	}
	err := routing.SVTNRoute(hdr, nil, r)
	if err != nil {
		t.Errorf("SVTNRoute same-SVTN: want nil, got %v", err)
	}
}

// ── Admitted-set check precedes forwarding (BC-2.05.002 postcondition 3) ─────

// TestRouteFrame_AdmittedSetCheckPrecedesForwarding verifies that RouteFrame
// checks the admitted set BEFORE attempting any forwarding decision.
//
// Scenario: source is not admitted; destination IS in forwarding table.
// If RouteFrame were to forward first, it would reach the destination. The
// test asserts the admitted-set check fires first (ErrNotAdmitted returned).
//
// Traces to BC-2.05.002 postcondition 3.
func TestRouteFrame_AdmittedSetCheckPrecedesForwarding(t *testing.T) {
	t.Parallel()

	svtnID := mustSVTN(0x05)
	unadmittedPub, _ := mustGenEd25519(t)
	unadmittedAddr := nodeAddrForTest(svtnID, unadmittedPub)

	admittedPub, _ := mustGenEd25519(t)
	admittedAddr := nodeAddrForTest(svtnID, admittedPub)

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, admittedPub, admission.RoleAccess)

	r := routing.NewRouter(ks)
	r.RegisterForwardingEntry(svtnID, admittedAddr, [32]byte{})

	// S-3.04: HMAC is now enforced before admission. Use a known key for
	// unadmittedAddr and compute a valid tag so HMAC passes; the admitted-set
	// check (ErrNotAdmitted) must still fire before SVTNRoute.
	var unadmittedKey [hmac.KeySize]byte
	copy(unadmittedKey[:], "admitted-set-precedes-test-key0")
	r.RegisterForwardingEntry(svtnID, unadmittedAddr, unadmittedKey) // in table but not admitted

	// Source (unadmittedAddr) is NOT in admitted set, even though it has a
	// forwarding entry. RouteFrame must reject it before forwarding.
	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtnID,
		SrcAddr:   unadmittedAddr,
		DstAddr:   admittedAddr,
	}
	// Compute a valid HMAC tag so HMAC passes; admission check must still fire.
	hdr.HMACTag = computeValidTag(hdr, nil, unadmittedKey)
	err := routing.RouteFrame(hdr, nil, r)

	// Precision pin (pass-4 M-1): assert the EXACT sentinel that proves admission
	// fires first. ErrNotAdmitted means the admission check caught the source
	// before any forwarding logic ran. ErrNoForwardingEntry would mean forwarding
	// logic ran first — that would violate BC-2.05.002 postcondition 3.
	if !errors.Is(err, admission.ErrNotAdmitted) {
		t.Errorf("admitted-set check before forwarding: want ErrNotAdmitted, got %v", err)
	}
	if errors.Is(err, routing.ErrNoForwardingEntry) {
		t.Errorf("admitted-set check before forwarding: got ErrNoForwardingEntry — forwarding ran before admission check, BC-2.05.002 postcondition 3 violated")
	}
}

// ── Fuzz harness: VP-008 — non-admitted source never forwarded ──────────────

// FuzzRouteFrame_NonAdmittedNeverForwarded is a fuzz target verifying that
// RouteFrame always returns a non-nil error when the frame's source address is
// not in the admitted set.
//
// Traces to VP-008 (Admission Fails for Unregistered Key, applied at routing layer)
// and BC-2.05.002 invariant 1.
func FuzzRouteFrame_NonAdmittedNeverForwarded(f *testing.F) {
	// Seed corpus: 80 bytes — 32 unadmitted seed + 32 admitted seed + 16 SVTN.
	// The prior 70-byte corpus caused t.Skip on every seeded run because the
	// length gate requires 80 bytes (pass-3 M-2).
	f.Add([]byte("unadmitted-seed-deterministic000admitted-seed-deterministic00000svtn-id-00000000"))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Need at least 80 bytes: 32 unadmitted seed + 32 admitted seed + 16 SVTN.
		if len(data) < 80 {
			t.Skip()
			return
		}

		// Derive unadmitted keypair from corpus bytes [0:32].
		unadmittedPub, _, err := ed25519.GenerateKey(bytes.NewReader(data[:32]))
		if err != nil {
			t.Skip()
			return
		}

		// Derive admitted keypair from corpus bytes [32:64].
		admittedPub, _, err := ed25519.GenerateKey(bytes.NewReader(data[32:64]))
		if err != nil {
			t.Skip()
			return
		}

		// Derive SVTN ID from corpus bytes [64:80].
		var svtnID [16]byte
		copy(svtnID[:], data[64:80])

		ks := admission.NewAdmittedKeySet()
		ks.RegisterKey(svtnID, admittedPub, admission.RoleAccess)
		// unadmittedPub deliberately NOT registered.

		r := routing.NewRouter(ks)
		admittedAddr := nodeAddrForTest(svtnID, admittedPub)
		r.RegisterForwardingEntry(svtnID, admittedAddr, [32]byte{})

		unadmittedAddr := nodeAddrForTest(svtnID, unadmittedPub)

		hdr := frame.OuterHeader{
			Version:   frame.VersionByte,
			FrameType: frame.FrameTypeData,
			SVTNID:    svtnID,
			SrcAddr:   unadmittedAddr,
			DstAddr:   admittedAddr,
		}
		err = routing.RouteFrame(hdr, nil, r)
		if err == nil {
			t.Errorf("RouteFrame with unadmitted src on svtn %v: want error, got nil", svtnID)
		}
	})
}

// ── S-3.04: HMAC wire-up tests (BC-2.05.008, ADR-009, VP-058) ────────────────
//
// These tests exercise RouteFrame's post-S-3.04 HMAC enforcement. All tests
// are GREEN against the post-S-3.04 implementation.

// seedKeyDet generates a deterministic Ed25519 keypair from a fixed 32-byte seed.
// Uses bytes.NewReader to satisfy ed25519.GenerateKey's io.Reader contract
// without invoking crypto/rand, making the test fully reproducible.
// (Pattern from VP-058.md proof harness skeleton.)
func seedKeyDet(t *testing.T, seed [32]byte) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(bytes.NewReader(seed[:]))
	if err != nil {
		t.Fatalf("seedKeyDet: %v", err)
	}
	return pub, priv
}

// admittedRouterSetup returns a Router with one fully-admitted node in svtnID.
//
// Returns:
//   - r: Router with forwarding entry for (svtnID, srcAddr) using authKey
//   - srcAddr: the node address derived from (svtnID, nodePub)
//   - authKey: the per-node HMAC key stored in the forwarding table
//
// The node has completed challenge-response via GenerateChallenge + AdmitNode,
// so admission.IsAdmitted(svtnID, srcAddr) returns true.
//
// Uses deterministic seeds so tests are reproducible. nodeSeedByte and
// routerSeedByte must be distinct across concurrent test cases.
func admittedRouterSetup(
	t *testing.T,
	svtnID [16]byte,
	nodeSeedByte byte,
	routerSeedByte byte,
) (r *routing.Router, srcAddr [8]byte, authKey [hmac.KeySize]byte) {
	t.Helper()

	var nodeSeed [32]byte
	nodeSeed[0] = nodeSeedByte
	copy(nodeSeed[1:], "s304-node-seed-filler-bytes-xxxx")
	nodePub, nodePriv := seedKeyDet(t, nodeSeed)

	var routerSeed [32]byte
	routerSeed[0] = routerSeedByte
	copy(routerSeed[1:], "s304-rtr--seed-filler-bytes-xxxx")
	_, routerPriv := seedKeyDet(t, routerSeed)

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)

	// Derive node address the same way DeriveNodeAddress does: SHA-256(svtnID || pubkey)[:8].
	h := sha256.New()
	h.Write(svtnID[:])
	h.Write([]byte(nodePub))
	sum := h.Sum(nil)
	copy(srcAddr[:], sum[:8])

	// Complete challenge-response so IsAdmitted returns true.
	challenge, err := admission.GenerateChallenge(routerPriv)
	if err != nil {
		t.Fatalf("admittedRouterSetup GenerateChallenge: %v", err)
	}
	nonceSig := ed25519.Sign(nodePriv, challenge.Nonce[:])
	resp := admission.ChallengeResponse{NonceSig: nonceSig}
	if err := admission.AdmitNode(challenge, resp, nodePub, svtnID, ks); err != nil {
		t.Fatalf("admittedRouterSetup AdmitNode: %v", err)
	}

	// Derive the auth key the same way admission.RegisterKey does: hmac.DeriveKey.
	// We re-derive it here so we can pass it to RegisterForwardingEntry and
	// also use it in tests to compute valid/invalid tags.
	authKey = hmac.DeriveKey([]byte(nodePub), svtnID)

	r = routing.NewRouter(ks)
	r.RegisterForwardingEntry(svtnID, srcAddr, authKey)

	return r, srcAddr, authKey
}

// computeValidTag computes the HMAC tag the sender would place in hdr.HMACTag.
//
// Protocol: zero HMACTag, encode header, concatenate payload, compute HMAC.
// This mirrors the sender's tag-insertion step (matching verifyFrameHMAC protocol).
func computeValidTag(hdr frame.OuterHeader, payload []byte, authKey [hmac.KeySize]byte) [hmac.TagSize]byte {
	hdrForMAC := hdr
	hdrForMAC.HMACTag = [hmac.TagSize]byte{}
	encoded := frame.EncodeOuterHeader(hdrForMAC)
	msg := make([]byte, len(encoded)+len(payload))
	copy(msg, encoded[:])
	copy(msg[len(encoded):], payload)
	return hmac.ComputeHMAC(authKey[:], msg)
}

// ── AC-001: TestRouteFrame_ValidHMAC_ProceedsToAdmission ─────────────────────

// TestRouteFrame_ValidHMAC_ProceedsToAdmission verifies that RouteFrame with a
// valid HMAC tag, an admitted node, and a forwarding entry proceeds through the
// admission check and SVTNRoute, returning nil.
//
// Traces to BC-2.05.008 PC-1; ADR-009 step 5 (proceed after HMAC passes).
//
// Regression guard: a mis-wired verifyFrameHMAC that rejects valid tags would
// make this test RED.
func TestRouteFrame_ValidHMAC_ProceedsToAdmission(t *testing.T) {
	t.Parallel()

	var svtnID [16]byte
	copy(svtnID[:], "s304-ac001-svtn0")

	r, srcAddr, authKey := admittedRouterSetup(t, svtnID, 0x01, 0x11)

	// Add a destination node so SVTNRoute can succeed.
	var dstAddr [8]byte
	copy(dstAddr[:], "dstaddr0")
	r.RegisterForwardingEntry(svtnID, dstAddr, [hmac.KeySize]byte{0xAA})

	payload := []byte("ac-001-payload")
	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtnID,
		SrcAddr:   srcAddr,
		DstAddr:   dstAddr,
	}
	// Compute and set the valid HMAC tag.
	hdr.HMACTag = computeValidTag(hdr, payload, authKey)

	err := routing.RouteFrame(hdr, payload, r)
	if err != nil {
		t.Errorf("AC-001: RouteFrame with valid HMAC + admitted src: want nil, got %v", err)
	}
}

// ── AC-002: TestRouteFrame_InvalidHMAC_ReturnsErrHMACVerificationFailed ──────

// TestRouteFrame_InvalidHMAC_ReturnsErrHMACVerificationFailed verifies that
// RouteFrame with an invalid HMAC tag returns ErrHMACVerificationFailed and
// does not proceed to the admitted-set check or SVTNRoute.
//
// Traces to BC-2.05.008 PC-2; ADR-009 step 4 (drop on HMAC mismatch);
// VP-058 property 3 (ErrHMACVerificationFailed before IsAdmitted/SVTNRoute).
func TestRouteFrame_InvalidHMAC_ReturnsErrHMACVerificationFailed(t *testing.T) {
	t.Parallel()

	var svtnID [16]byte
	copy(svtnID[:], "s304-ac002-svtn0")

	r, srcAddr, _ := admittedRouterSetup(t, svtnID, 0x02, 0x12)

	var dstAddr [8]byte
	copy(dstAddr[:], "dstaddr1")
	r.RegisterForwardingEntry(svtnID, dstAddr, [hmac.KeySize]byte{0xBB})

	payload := []byte("ac-002-payload")
	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtnID,
		SrcAddr:   srcAddr,
		DstAddr:   dstAddr,
		// HMACTag deliberately zero (invalid).
	}

	err := routing.RouteFrame(hdr, payload, r)
	if !errors.Is(err, routing.ErrHMACVerificationFailed) {
		t.Errorf("AC-002: invalid HMAC tag: want ErrHMACVerificationFailed, got %v", err)
	}
}

// ── AC-003: TestRouteFrame_HMACEnforcedBeforeAdmission ───────────────────────

// TestRouteFrame_HMACEnforcedBeforeAdmission verifies that HMAC verification
// occurs BEFORE the admitted-set check (ADR-009 ordering; BC-2.05.008 PC-3).
//
// Scenario: the node IS admitted, but sends an invalid HMAC tag. The frame
// must return ErrHMACVerificationFailed, not ErrNotAdmitted. If the admitted-set
// check ran first, it would pass (node is admitted), and then HMAC would check —
// but HMAC fires FIRST per ADR-009, so the wrong-tag frame is rejected before
// admission is even consulted.
//
// Traces to BC-2.05.008 PC-3; ADR-009 step 4 (ordering invariant);
// VP-058 property 1 (verifyFrameHMAC before IsAdmitted).
func TestRouteFrame_HMACEnforcedBeforeAdmission(t *testing.T) {
	t.Parallel()

	var svtnID [16]byte
	copy(svtnID[:], "s304-ac003-svtn0")

	r, srcAddr, _ := admittedRouterSetup(t, svtnID, 0x03, 0x13)

	var dstAddr [8]byte
	copy(dstAddr[:], "dstaddr2")
	r.RegisterForwardingEntry(svtnID, dstAddr, [hmac.KeySize]byte{0xCC})

	payload := []byte("ac-003-payload")
	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtnID,
		SrcAddr:   srcAddr,
		DstAddr:   dstAddr,
		// HMACTag zero → invalid tag. Node IS admitted (challenge-response done).
	}

	err := routing.RouteFrame(hdr, payload, r)

	if !errors.Is(err, routing.ErrHMACVerificationFailed) {
		t.Errorf("AC-003: admitted node with invalid HMAC: want ErrHMACVerificationFailed, got %v", err)
	}
	// Ordering invariant: if ErrNotAdmitted fires, it means admitted-set check ran
	// before HMAC verification — ADR-009 ordering violation.
	if errors.Is(err, admission.ErrNotAdmitted) {
		t.Error("AC-003: ordering violation — admitted-set check fired before HMAC verification (ADR-009)")
	}
}

// ── AC-004: TestRouteFrame_NoForwardingEntry_RejectsAsUnverifiable ────────────

// TestRouteFrame_NoForwardingEntry_RejectsAsUnverifiable verifies that when no
// forwarding-table entry exists for (hdr.SVTNID, hdr.SrcAddr), RouteFrame returns
// ErrHMACVerificationFailed (auth key unavailable → frame is unverifiable → dropped
// fail-closed). The admitted-set check is never reached.
//
// Traces to BC-2.05.008 PC-4; ADR-009 step 2 (absent entry → drop);
// VP-058 property 4.
func TestRouteFrame_NoForwardingEntry_RejectsAsUnverifiable(t *testing.T) {
	t.Parallel()

	var svtnID [16]byte
	copy(svtnID[:], "s304-ac004-svtn0")

	// Build a fully admitted node but do NOT register a forwarding entry for its
	// source address (auth key unavailable).
	var nodeSeed [32]byte
	nodeSeed[0] = 0x04
	copy(nodeSeed[1:], "s304-node-seed-filler-bytes-xxxx")
	nodePub, nodePriv := seedKeyDet(t, nodeSeed)

	var routerSeed [32]byte
	routerSeed[0] = 0x14
	copy(routerSeed[1:], "s304-rtr--seed-filler-bytes-xxxx")
	_, routerPriv := seedKeyDet(t, routerSeed)

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)

	h := sha256.New()
	h.Write(svtnID[:])
	h.Write([]byte(nodePub))
	sum := h.Sum(nil)
	var srcAddr [8]byte
	copy(srcAddr[:], sum[:8])

	challenge, err := admission.GenerateChallenge(routerPriv)
	if err != nil {
		t.Fatalf("GenerateChallenge: %v", err)
	}
	nonceSig := ed25519.Sign(nodePriv, challenge.Nonce[:])
	resp := admission.ChallengeResponse{NonceSig: nonceSig}
	if err := admission.AdmitNode(challenge, resp, nodePub, svtnID, ks); err != nil {
		t.Fatalf("AdmitNode: %v", err)
	}

	// Deliberately NO forwarding entry for (svtnID, srcAddr).
	r := routing.NewRouter(ks)

	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtnID,
		SrcAddr:   srcAddr,
	}
	err = routing.RouteFrame(hdr, nil, r)
	if !errors.Is(err, routing.ErrHMACVerificationFailed) {
		t.Errorf("AC-004: no forwarding entry: want ErrHMACVerificationFailed, got %v", err)
	}
}

// ── AC-005: TestRouteFrame_ValidHMAC_Unadmitted_ReturnsErrNotAdmitted ─────────

// TestRouteFrame_ValidHMAC_Unadmitted_ReturnsErrNotAdmitted verifies that a frame
// with a valid HMAC tag from a node NOT in the admitted set returns
// admission.ErrNotAdmitted (not ErrHMACVerificationFailed). This confirms the two
// sentinels are distinct and that HMAC passes before admission fails.
//
// Traces to BC-2.05.008 EC-005; BC-2.05.008 invariant 2 (sentinel distinction).
func TestRouteFrame_ValidHMAC_Unadmitted_ReturnsErrNotAdmitted(t *testing.T) {
	t.Parallel()

	var svtnID [16]byte
	copy(svtnID[:], "s304-ac005-svtn0")

	// Create a node with a forwarding entry but NOT admitted (no AdmitNode call).
	nodePub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}

	h := sha256.New()
	h.Write(svtnID[:])
	h.Write([]byte(nodePub))
	sum := h.Sum(nil)
	var srcAddr [8]byte
	copy(srcAddr[:], sum[:8])

	authKey := hmac.DeriveKey([]byte(nodePub), svtnID)

	// RegisterKey so there is a forwarding entry (auth key present) but admitted=false.
	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)
	// No AdmitNode — node is NOT admitted.

	r := routing.NewRouter(ks)
	r.RegisterForwardingEntry(svtnID, srcAddr, authKey)

	var dstAddr [8]byte
	copy(dstAddr[:], "dstaddr3")
	r.RegisterForwardingEntry(svtnID, dstAddr, [hmac.KeySize]byte{0xDD})

	payload := []byte("ac-005-payload")
	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtnID,
		SrcAddr:   srcAddr,
		DstAddr:   dstAddr,
	}
	hdr.HMACTag = computeValidTag(hdr, payload, authKey)

	err = routing.RouteFrame(hdr, payload, r)
	if !errors.Is(err, admission.ErrNotAdmitted) {
		t.Errorf("AC-005: valid HMAC + unadmitted src: want ErrNotAdmitted, got %v", err)
	}
	// Sentinel distinction invariant (BC-2.05.008 invariant 2).
	if errors.Is(err, routing.ErrHMACVerificationFailed) {
		t.Error("AC-005: got ErrHMACVerificationFailed — sentinel collision (HMAC and admission errors must be distinct)")
	}
}

// ── EC-001: TestRouteFrame_ZeroHMACTag_Rejected ───────────────────────────────

// TestRouteFrame_ZeroHMACTag_Rejected verifies that a frame with an all-zero
// HMACTag is rejected with ErrHMACVerificationFailed (EC-001; E-ADM-016 logged).
//
// Traces to BC-2.05.008 EC-001.
func TestRouteFrame_ZeroHMACTag_Rejected(t *testing.T) {
	t.Parallel()

	var svtnID [16]byte
	copy(svtnID[:], "s304-ec001-svtn0")

	r, srcAddr, _ := admittedRouterSetup(t, svtnID, 0x05, 0x15)

	var dstAddr [8]byte
	copy(dstAddr[:], "dstaddr4")
	r.RegisterForwardingEntry(svtnID, dstAddr, [hmac.KeySize]byte{0xEE})

	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtnID,
		SrcAddr:   srcAddr,
		DstAddr:   dstAddr,
		// HMACTag all-zero — the zero value of [8]byte.
	}

	err := routing.RouteFrame(hdr, []byte("ec-001-payload"), r)
	if !errors.Is(err, routing.ErrHMACVerificationFailed) {
		t.Errorf("EC-001: all-zero HMAC tag: want ErrHMACVerificationFailed, got %v", err)
	}
}

// ── EC-002: TestRouteFrame_WrongKeyHMAC_Rejected ──────────────────────────────

// TestRouteFrame_WrongKeyHMAC_Rejected verifies that a frame with an HMAC tag
// computed under a different node's key (cross-node forgery) is rejected with
// ErrHMACVerificationFailed (EC-002; E-ADM-016 logged).
//
// Traces to BC-2.05.008 EC-002; ADR-009 (per-node keying prevents cross-node forgery).
func TestRouteFrame_WrongKeyHMAC_Rejected(t *testing.T) {
	t.Parallel()

	var svtnID [16]byte
	copy(svtnID[:], "s304-ec002-svtn0")

	r, srcAddr, _ := admittedRouterSetup(t, svtnID, 0x06, 0x16)

	var dstAddr [8]byte
	copy(dstAddr[:], "dstaddr5")
	r.RegisterForwardingEntry(svtnID, dstAddr, [hmac.KeySize]byte{0xFF})

	// Compute HMAC tag using a completely different (wrong) key.
	var wrongKey [hmac.KeySize]byte
	copy(wrongKey[:], "wrong-key-for-cross-node-forgery")

	payload := []byte("ec-002-payload")
	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtnID,
		SrcAddr:   srcAddr,
		DstAddr:   dstAddr,
	}
	hdr.HMACTag = computeValidTag(hdr, payload, wrongKey)

	err := routing.RouteFrame(hdr, payload, r)
	if !errors.Is(err, routing.ErrHMACVerificationFailed) {
		t.Errorf("EC-002: cross-node forged HMAC: want ErrHMACVerificationFailed, got %v", err)
	}
}

// ── EC-003: TestRouteFrame_AdmittedNodeForwardingEntryPurged ──────────────────

// TestRouteFrame_AdmittedNodeForwardingEntryPurged verifies that a node which IS
// in the admitted set but whose forwarding-table entry has been purged (auth key
// unavailable) is rejected with ErrHMACVerificationFailed. The admitted-set check
// is never reached.
//
// Traces to BC-2.05.008 EC-003; ADR-009 step 2.
func TestRouteFrame_AdmittedNodeForwardingEntryPurged(t *testing.T) {
	t.Parallel()

	var svtnID [16]byte
	copy(svtnID[:], "s304-ec003-svtn0")

	// Build admitted setup, then do NOT register a forwarding entry for srcAddr.
	var nodeSeed [32]byte
	nodeSeed[0] = 0x07
	copy(nodeSeed[1:], "s304-node-seed-filler-bytes-xxxx")
	nodePub, nodePriv := seedKeyDet(t, nodeSeed)

	var routerSeed [32]byte
	routerSeed[0] = 0x17
	copy(routerSeed[1:], "s304-rtr--seed-filler-bytes-xxxx")
	_, routerPriv := seedKeyDet(t, routerSeed)

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)

	h := sha256.New()
	h.Write(svtnID[:])
	h.Write([]byte(nodePub))
	sum := h.Sum(nil)
	var srcAddr [8]byte
	copy(srcAddr[:], sum[:8])

	challenge, err := admission.GenerateChallenge(routerPriv)
	if err != nil {
		t.Fatalf("GenerateChallenge: %v", err)
	}
	nonceSig := ed25519.Sign(nodePriv, challenge.Nonce[:])
	resp := admission.ChallengeResponse{NonceSig: nonceSig}
	if err := admission.AdmitNode(challenge, resp, nodePub, svtnID, ks); err != nil {
		t.Fatalf("AdmitNode: %v", err)
	}

	// Router with NO forwarding entry for srcAddr (simulates purged entry).
	r := routing.NewRouter(ks)

	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtnID,
		SrcAddr:   srcAddr,
	}
	err = routing.RouteFrame(hdr, nil, r)
	if !errors.Is(err, routing.ErrHMACVerificationFailed) {
		t.Errorf("EC-003: admitted node + purged forwarding entry: want ErrHMACVerificationFailed, got %v", err)
	}
}

// ── EC-004: TestRouteFrame_EmptyPayload_ValidHMAC_Forwarded ───────────────────

// TestRouteFrame_EmptyPayload_ValidHMAC_Forwarded verifies that an empty-payload
// (zero-length payload) frame with a correct HMAC tag is forwarded normally.
//
// Traces to BC-2.05.008 EC-004; BC-2.05.005 EC-001 (HMAC over empty payload valid).
//
// Regression guard: verifyFrameHMAC must handle nil/empty payload (HMAC over
// header-only message) without error.
func TestRouteFrame_EmptyPayload_ValidHMAC_Forwarded(t *testing.T) {
	t.Parallel()

	var svtnID [16]byte
	copy(svtnID[:], "s304-ec004-svtn0")

	r, srcAddr, authKey := admittedRouterSetup(t, svtnID, 0x08, 0x18)

	var dstAddr [8]byte
	copy(dstAddr[:], "dstaddr6")
	r.RegisterForwardingEntry(svtnID, dstAddr, [hmac.KeySize]byte{0x11})

	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtnID,
		SrcAddr:   srcAddr,
		DstAddr:   dstAddr,
		// PayloadLen stays 0.
	}
	// Empty payload: computeValidTag with nil payload.
	hdr.HMACTag = computeValidTag(hdr, nil, authKey)

	err := routing.RouteFrame(hdr, nil, r)
	if err != nil {
		t.Errorf("EC-004: empty payload + valid HMAC: want nil, got %v", err)
	}
}

// ── EC-005: TestRouteFrame_ValidHMAC_RevokedNode_ReturnsErrNotAdmitted ────────

// TestRouteFrame_ValidHMAC_RevokedNode_ReturnsErrNotAdmitted verifies that a frame
// with valid HMAC from a node that is in the admitted set but subsequently revoked
// returns admission.ErrNotAdmitted (not ErrHMACVerificationFailed).
//
// This is an edge case distinct from EC-003 (purged forwarding entry): here the
// forwarding entry still exists (auth key available), HMAC passes, but the
// admitted-set check fails because the key is revoked.
//
// Traces to BC-2.05.008 EC-005; BC-2.05.004 (key revocation).
func TestRouteFrame_ValidHMAC_RevokedNode_ReturnsErrNotAdmitted(t *testing.T) {
	t.Parallel()

	var svtnID [16]byte
	copy(svtnID[:], "s304-ec005-svtn0")

	var nodeSeed [32]byte
	nodeSeed[0] = 0x09
	copy(nodeSeed[1:], "s304-node-seed-filler-bytes-xxxx")
	nodePub, nodePriv := seedKeyDet(t, nodeSeed)

	var routerSeed [32]byte
	routerSeed[0] = 0x19
	copy(routerSeed[1:], "s304-rtr--seed-filler-bytes-xxxx")
	_, routerPriv := seedKeyDet(t, routerSeed)

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)

	h := sha256.New()
	h.Write(svtnID[:])
	h.Write([]byte(nodePub))
	sum := h.Sum(nil)
	var srcAddr [8]byte
	copy(srcAddr[:], sum[:8])

	challenge, err := admission.GenerateChallenge(routerPriv)
	if err != nil {
		t.Fatalf("GenerateChallenge: %v", err)
	}
	nonceSig := ed25519.Sign(nodePriv, challenge.Nonce[:])
	resp := admission.ChallengeResponse{NonceSig: nonceSig}
	if err := admission.AdmitNode(challenge, resp, nodePub, svtnID, ks); err != nil {
		t.Fatalf("AdmitNode: %v", err)
	}

	authKey := hmac.DeriveKey([]byte(nodePub), svtnID)

	r := routing.NewRouter(ks)
	r.RegisterForwardingEntry(svtnID, srcAddr, authKey)

	var dstAddr [8]byte
	copy(dstAddr[:], "dstaddr7")
	r.RegisterForwardingEntry(svtnID, dstAddr, [hmac.KeySize]byte{0x22})

	// Revoke the node. The forwarding entry and auth key still exist.
	// RevokeKey operates on ks directly — Router.admittedKeySet is unexported,
	// but ks is the same pointer passed to NewRouter.
	if err := ks.RevokeKey(svtnID, srcAddr); err != nil {
		t.Fatalf("RevokeKey: %v", err)
	}

	payload := []byte("ec-005-payload")
	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtnID,
		SrcAddr:   srcAddr,
		DstAddr:   dstAddr,
	}
	hdr.HMACTag = computeValidTag(hdr, payload, authKey)

	err = routing.RouteFrame(hdr, payload, r)
	if !errors.Is(err, admission.ErrNotAdmitted) {
		t.Errorf("EC-005: valid HMAC + revoked node: want ErrNotAdmitted, got %v", err)
	}
}
