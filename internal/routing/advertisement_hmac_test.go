// advertisement_hmac_test.go covers internal/routing/advertisement_hmac.go's
// exported surface, including the S-BL.DISCOVERY-WIRE AC-004 additions
// (DiscoveryAuthKeyFor, DeriveDiscoveryKey). No test file previously existed
// for this production file — ComputeAdvertisementHMAC/VerifyAdvertisementHMAC
// were exercised only indirectly via internal/discovery's Encode/Decode
// tests; this file is created (not extended, despite the story's File-Change
// List wording) to house the new AC-004 tests.
package routing_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/hmac"
	"github.com/arcavenae/switchboard/internal/routing"
)

// redGateGuard recovers from a not-yet-implemented stub's panic and fails the
// test cleanly (Red Gate discipline, BC-5.38.001) instead of crashing the
// whole test binary. Once the relevant Task's Green step lands, the panic
// disappears and this guard becomes a silent no-op.
func redGateGuard(t *testing.T) {
	t.Helper()
	if r := recover(); r != nil {
		t.Fatalf("red gate: stub not yet implemented: %v", r)
	}
}

// newAdmittedRouter registers a fresh Ed25519 key for svtnID on a new
// AdmittedKeySet and returns a *routing.Router wrapping it, the raw pubkey,
// and the derived NodeAddr.
func newAdmittedRouter(t *testing.T, svtnID [16]byte) (*routing.Router, ed25519.PublicKey, [8]byte) {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate admission key: %v", err)
	}
	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, pub, admission.RoleAccess)
	nodeAddr := frame.DeriveNodeAddress(svtnID, []byte(pub))
	return routing.NewRouter(ks), pub, nodeAddr
}

// TestDiscoveryAuthKeyFor_LookupSuccessAndMiss verifies AC-004 postcondition
// 3: DiscoveryAuthKeyFor returns (key, true) when the (svtnID, nodeAddr) pair
// is admitted, and (zero, false) otherwise — a thin, read-only wrapper.
func TestDiscoveryAuthKeyFor_LookupSuccessAndMiss(t *testing.T) {
	t.Parallel()

	var svtnID [16]byte
	copy(svtnID[:], "routing-svtn-004")
	router, pub, nodeAddr := newAdmittedRouter(t, svtnID)

	t.Run("lookup success", func(t *testing.T) {
		defer redGateGuard(t)

		got, ok := router.DiscoveryAuthKeyFor(svtnID, nodeAddr)
		if !ok {
			t.Fatal("DiscoveryAuthKeyFor: ok = false for an admitted (svtnID, nodeAddr) pair, want true")
		}
		want := hmac.DeriveDiscoveryKey([]byte(pub), svtnID)
		if got != want {
			t.Errorf("DiscoveryAuthKeyFor returned %x, want %x (hmac.DeriveDiscoveryKey(pubkey, svtnID))", got, want)
		}
	})

	t.Run("lookup miss: unregistered nodeAddr", func(t *testing.T) {
		defer redGateGuard(t)

		unknownAddr := [8]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
		got, ok := router.DiscoveryAuthKeyFor(svtnID, unknownAddr)
		if ok {
			t.Error("DiscoveryAuthKeyFor: ok = true for an unregistered nodeAddr, want false")
		}
		var zero [hmac.KeySize]byte
		if got != zero {
			t.Errorf("DiscoveryAuthKeyFor: got non-zero key %x on lookup miss, want zero value", got)
		}
	})

	t.Run("lookup miss: unregistered svtnID", func(t *testing.T) {
		defer redGateGuard(t)

		var otherSVTN [16]byte
		copy(otherSVTN[:], "routing-svtn-oth")
		got, ok := router.DiscoveryAuthKeyFor(otherSVTN, nodeAddr)
		if ok {
			t.Error("DiscoveryAuthKeyFor: ok = true for an unregistered svtnID, want false")
		}
		var zero [hmac.KeySize]byte
		if got != zero {
			t.Errorf("DiscoveryAuthKeyFor: got non-zero key %x on lookup miss, want zero value", got)
		}
	})
}

// TestDeriveDiscoveryKey_SenderRouterAgree verifies AC-004 postcondition 4:
// the sender-side wrapper routing.DeriveDiscoveryKey(pubkey, svtnID) produces
// the identical output DiscoveryAuthKeyFor computes for that same node's own
// admitted key — a sending access node can derive its key locally without
// querying the router.
func TestDeriveDiscoveryKey_SenderRouterAgree(t *testing.T) {
	t.Parallel()
	defer redGateGuard(t)

	var svtnID [16]byte
	copy(svtnID[:], "routing-svtn-agr")
	router, pub, nodeAddr := newAdmittedRouter(t, svtnID)

	routerKey, ok := router.DiscoveryAuthKeyFor(svtnID, nodeAddr)
	if !ok {
		t.Fatal("DiscoveryAuthKeyFor: lookup failed for a just-registered node")
	}
	senderKey := routing.DeriveDiscoveryKey([]byte(pub), svtnID)

	if senderKey != routerKey {
		t.Errorf("routing.DeriveDiscoveryKey(pubkey, svtnID) = %x, router.DiscoveryAuthKeyFor(svtnID, nodeAddr) = %x — sender and router must derive the identical key", senderKey, routerKey)
	}
}
