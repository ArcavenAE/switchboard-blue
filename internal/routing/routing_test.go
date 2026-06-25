package routing_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/frame"
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
// The return value is consumed once stub methods are implemented; during the stub
// phase, stubs panic before return values are used.
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

// ── AC-004: TestRouteFrame_DropsUnadmitted ───────────────────────────────────

// TestRouteFrame_DropsUnadmitted verifies that RouteFrame returns
// admission.ErrNotAdmitted and drops the frame when the frame's src_addr
// is not in the admitted set for the frame's svtn_id.
//
// Traces to BC-2.05.002 postcondition 1 (frame from non-admitted source →
// dropped; E-ADM-003 logged).
func TestRouteFrame_DropsUnadmitted(t *testing.T) {
	t.Parallel()

	svtnID := mustSVTN(0x01)
	unadmittedPub, _ := mustGenEd25519(t)
	unadmittedAddr := nodeAddrForTest(svtnID, unadmittedPub) //nolint:staticcheck // consumed once stub is implemented

	admittedPub, _ := mustGenEd25519(t)
	admittedAddr := nodeAddrForTest(svtnID, admittedPub) //nolint:staticcheck // consumed once stub is implemented

	ks := admission.NewAdmittedKeySet()
	// Only admittedPub is registered — unadmittedPub is NOT.
	ks.RegisterKey(svtnID, admittedPub, admission.RoleAccess)

	r := routing.NewRouter(ks)
	r.RegisterForwardingEntry(svtnID, admittedAddr, [32]byte{})

	// Frame from unadmitted source.
	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtnID,
		SrcAddr:   unadmittedAddr,
		DstAddr:   admittedAddr,
	}
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
	nodeAAddr := nodeAddrForTest(svtnA, nodeAPub) //nolint:staticcheck // consumed once stub is implemented
	nodeBAddr := nodeAddrForTest(svtnB, nodeBPub) //nolint:staticcheck // consumed once stub is implemented

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
			srcAddr := nodeAddrForTest(tc.srcSVTN, srcPub) //nolint:staticcheck // consumed once stub is implemented
			dstAddr := nodeAddrForTest(tc.dstSVTN, dstPub) //nolint:staticcheck // consumed once stub is implemented

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
	srcAddr := nodeAddrForTest(svtnA, srcPub) //nolint:staticcheck // consumed once stub is implemented
	dstAddr := nodeAddrForTest(svtnA, dstPub) //nolint:staticcheck // consumed once stub is implemented

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
	unadmittedAddr := nodeAddrForTest(svtnID, unadmittedPub) //nolint:staticcheck // consumed once stub is implemented

	admittedPub, _ := mustGenEd25519(t)
	admittedAddr := nodeAddrForTest(svtnID, admittedPub) //nolint:staticcheck // consumed once stub is implemented

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, admittedPub, admission.RoleAccess)

	r := routing.NewRouter(ks)
	r.RegisterForwardingEntry(svtnID, admittedAddr, [32]byte{})
	r.RegisterForwardingEntry(svtnID, unadmittedAddr, [32]byte{}) // in table but not admitted

	// Source (unadmittedAddr) is NOT in admitted set, even though it has a
	// forwarding entry. RouteFrame must reject it before forwarding.
	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtnID,
		SrcAddr:   unadmittedAddr,
		DstAddr:   admittedAddr,
	}
	err := routing.RouteFrame(hdr, nil, r)
	if !errors.Is(err, admission.ErrNotAdmitted) {
		t.Errorf("admitted-set check before forwarding: want ErrNotAdmitted, got %v", err)
	}
}

// ── Fuzz harness: VP-057 — no frame from non-admitted source reaches any dest ─

// FuzzRouteFrame_NonAdmittedNeverForwarded is a fuzz target verifying that
// RouteFrame always returns a non-nil error when the frame's source address is
// not in the admitted set.
//
// Traces to VP-057 (no frame from non-admitted source reaches any destination)
// and BC-2.05.002 invariant 1.
func FuzzRouteFrame_NonAdmittedNeverForwarded(f *testing.F) {
	// Seed corpus.
	f.Add(byte(0x01), byte(0x02))

	f.Fuzz(func(t *testing.T, svtnByte, dstByte byte) {
		svtnID := mustSVTN(svtnByte)

		// Router key — not used in stub, skip if keygen fails.
		_, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Skip()
			return
		}
		unadmittedPub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Skip()
			return
		}
		admittedPub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Skip()
			return
		}
		_ = unadmittedPub //nolint:staticcheck // consumed once stub is implemented

		ks := admission.NewAdmittedKeySet()
		ks.RegisterKey(svtnID, admittedPub, admission.RoleAccess)
		// unadmittedPub deliberately NOT registered.

		r := routing.NewRouter(ks)
		admittedAddr := nodeAddrForTest(svtnID, admittedPub) //nolint:staticcheck // consumed once stub is implemented
		r.RegisterForwardingEntry(svtnID, admittedAddr, [32]byte{})

		unadmittedAddr := nodeAddrForTest(svtnID, unadmittedPub) //nolint:staticcheck // consumed once stub is implemented

		var dstAddr [8]byte
		dstAddr[0] = dstByte

		hdr := frame.OuterHeader{
			Version:   frame.VersionByte,
			FrameType: frame.FrameTypeData,
			SVTNID:    svtnID,
			SrcAddr:   unadmittedAddr,
			DstAddr:   dstAddr,
		}
		err = routing.RouteFrame(hdr, nil, r)
		if err == nil {
			t.Errorf("RouteFrame with unadmitted src on svtn %v: want error, got nil", svtnID)
		}
	})
}
