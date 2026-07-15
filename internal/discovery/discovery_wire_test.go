// discovery_wire_test.go covers the router-side hop-1 ingest path
// (RouterIngest.Ingest — AC-005, AC-006, AC-008, AC-009, AC-010, AC-011,
// AC-012, AC-013) and the node-side hop-2 relay-ingest function
// (IngestRelayAdvertisement — AC-007). AC-005/AC-006 establish the
// admitted-node/SVTN DiscoveryAuthKeyFor-admitted test-setup pattern VP-080
// v1.7 cites as the surviving lineage after
// TestDiscovery_VP045_SVTNIsolation_MultipleScopes's retirement (see
// discovery_test.go's still-intact ten-test disposition — untouched by this
// file, per Task 4's own Green-step scope).
//
// redGateGuard, svtnA, svtnB, nodeA1, nodeA2 are shared from discovery_test.go
// (same package).
package discovery_test

import (
	"context"
	"crypto/ed25519"
	crypto_hmac "crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/discovery"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/hmac"
	"github.com/arcavenae/switchboard/internal/routing"
)

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// newAdmittedRouterForDiscoveryWire registers a fresh Ed25519 key for
// (svtnID, derived-NodeAddr) on a new AdmittedKeySet and returns a
// *routing.Router wrapping it — the AC-005/AC-006 test-setup pattern.
func newAdmittedRouterForDiscoveryWire(t testing.TB, svtnID [16]byte) (*routing.Router, ed25519.PublicKey, [8]byte) {
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

// testDeriveDiscoveryKey independently re-implements the HKDF-SHA256
// construction Decision 1 specifies (hmac.DeriveDiscoveryKey, once Task 1's
// Green step lands) so this test file can build cryptographically valid
// hop-1 datagrams without depending on the very stub under test as its own
// oracle. Same RFC 5869 Extract/Expand shape as internal/hmac's private
// hkdfSHA256: PRK = HMAC-SHA256(salt, IKM); T(1) = HMAC-SHA256(PRK,
// info || 0x01); output = T(1) (hmac.KeySize == sha256.Size, so a single
// block suffices).
func testDeriveDiscoveryKey(nodeAdmissionPubkey []byte, svtnID [16]byte) [hmac.KeySize]byte {
	extractMAC := crypto_hmac.New(sha256.New, svtnID[:])
	extractMAC.Write(nodeAdmissionPubkey)
	prk := extractMAC.Sum(nil)

	expandMAC := crypto_hmac.New(sha256.New, prk)
	expandMAC.Write([]byte(hmac.HKDFInfoDiscovery))
	expandMAC.Write([]byte{1})
	t1 := expandMAC.Sum(nil)

	var out [hmac.KeySize]byte
	copy(out[:], t1)
	return out
}

// buildHop1Body assembles the hop-1 body (everything after the 8-byte HMAC
// tag): SVTNID | NodeAddr | Sequence(BE uint64) | count(BE uint16) |
// sessions — the wire layout AC-005's note pins: body[0:16]=SVTNID,
// body[16:24]=NodeAddr, body[24:32]=Sequence, body[32:34]=count,
// body[34:]=sessions (per-session encoding matches discovery.go's existing
// encodeBody: uint16 name_len | name | uint8 status | uint8 quality).
func buildHop1Body(svtnID [16]byte, nodeAddr [8]byte, sequence uint64, sessions []discovery.SessionPresence) []byte {
	buf := make([]byte, 0, 16+8+8+2+64)
	buf = append(buf, svtnID[:]...)
	buf = append(buf, nodeAddr[:]...)
	buf = binary.BigEndian.AppendUint64(buf, sequence)
	buf = binary.BigEndian.AppendUint16(buf, uint16(len(sessions)))
	for _, s := range sessions {
		name := []byte(s.SessionName)
		buf = binary.BigEndian.AppendUint16(buf, uint16(len(name)))
		buf = append(buf, name...)
		buf = append(buf, byte(s.Status))
		buf = append(buf, byte(s.Quality))
	}
	return buf
}

// buildHop1Datagram assembles a full hop-1 raw multicast datagram: an
// 8-byte HMAC tag computed over the body under key, followed by the body
// itself (buildHop1Body). Pass a deliberately-wrong key to produce a
// datagram whose tag will not verify.
func buildHop1Datagram(key [hmac.KeySize]byte, svtnID [16]byte, nodeAddr [8]byte, sequence uint64, sessions []discovery.SessionPresence) []byte {
	body := buildHop1Body(svtnID, nodeAddr, sequence, sessions)
	tag := routing.ComputeAdvertisementHMAC(key[:], body)
	out := make([]byte, 0, len(tag)+len(body))
	out = append(out, tag[:]...)
	out = append(out, body...)
	return out
}

// buildHop2Payload assembles a DISCOVERY_RELAY payload in the shape
// IngestRelayAdvertisement expects (AC-007 postcondition 2; Decision 3(c)
// minus the 4-byte control header, which the caller strips before calling
// IngestRelayAdvertisement): NodeAddr | Sequence(BE) | count(BE) |
// sessions...
func buildHop2Payload(nodeAddr [8]byte, sequence uint64, sessions []discovery.SessionPresence) []byte {
	buf := make([]byte, 0, 8+8+2+64)
	buf = append(buf, nodeAddr[:]...)
	buf = binary.BigEndian.AppendUint64(buf, sequence)
	buf = binary.BigEndian.AppendUint16(buf, uint16(len(sessions)))
	for _, s := range sessions {
		name := []byte(s.SessionName)
		buf = binary.BigEndian.AppendUint16(buf, uint16(len(name)))
		buf = append(buf, name...)
		buf = append(buf, byte(s.Status))
		buf = append(buf, byte(s.Quality))
	}
	return buf
}

var oneSession = []discovery.SessionPresence{
	{SessionName: "agent-01", Status: discovery.Attached, Quality: discovery.QualityGreen},
}

// ---------------------------------------------------------------------------
// AC-005 — fixed-offset key-selector extraction precedes full body decode
// ---------------------------------------------------------------------------

// TestRouterIngest_KeySelectorExtraction_FixedOffset_NoFullDecodeBeforeAuth
// verifies AC-005 postconditions 1 and 2: SVTNID/NodeAddr are read via
// fixed-offset indexing to select the verification key, and decodeBody's
// variable-length, attacker-controlled session-entry walk never runs before
// HMAC succeeds. Oracle: a datagram with a well-formed 32-byte key-selector
// prefix, a deliberately wrong HMAC tag, and a session-list tail malformed
// in a way that would raise a DIFFERENT error if decoded before auth (a
// declared count the remaining bytes cannot satisfy) must still resolve to
// ErrInvalidHMACTag — no decode-error ever leaks pre-auth.
func TestRouterIngest_KeySelectorExtraction_FixedOffset_NoFullDecodeBeforeAuth(t *testing.T) {
	t.Parallel()
	defer redGateGuard(t)

	router, _, nodeAddr := newAdmittedRouterForDiscoveryWire(t, svtnA)
	ri := discovery.NewRouterIngest(discovery.RouterIngestConfig{Router: router})

	body := make([]byte, 0, 34)
	body = append(body, svtnA[:]...)
	body = append(body, nodeAddr[:]...)
	body = binary.BigEndian.AppendUint64(body, 1)
	body = binary.BigEndian.AppendUint16(body, 0xFFFF) // declares 65535 sessions, no session bytes follow
	raw := make([]byte, 0, 8+len(body))
	raw = append(raw, make([]byte, hmac.TagSize)...) // wrong/zero HMAC tag
	raw = append(raw, body...)

	_, err := ri.Ingest(raw)
	if !errors.Is(err, discovery.ErrInvalidHMACTag) {
		t.Fatalf("Ingest: got err %v, want ErrInvalidHMACTag (a decodeBody malformed-tail error must never leak pre-auth)", err)
	}
}

// TestRouterIngest_HMACCoversFullBody_TamperInSessionListDetected verifies
// AC-005 postcondition 3: the HMAC computation covers the complete raw body
// bytes, not merely the 24-byte key-selector prefix — a forger cannot leave
// SVTNID/NodeAddr untouched while corrupting the session list beneath an
// otherwise-valid tag.
func TestRouterIngest_HMACCoversFullBody_TamperInSessionListDetected(t *testing.T) {
	t.Parallel()
	defer redGateGuard(t)

	router, pub, nodeAddr := newAdmittedRouterForDiscoveryWire(t, svtnA)
	key := testDeriveDiscoveryKey([]byte(pub), svtnA)
	raw := buildHop1Datagram(key, svtnA, nodeAddr, 1, oneSession)

	// Tamper one byte inside the session-list tail, past the key-selector
	// prefix, without recomputing the tag.
	raw[len(raw)-1] ^= 0xFF

	ri := discovery.NewRouterIngest(discovery.RouterIngestConfig{Router: router})
	_, err := ri.Ingest(raw)
	if !errors.Is(err, discovery.ErrInvalidHMACTag) {
		t.Fatalf("Ingest: tampered session-list tail was not rejected: got err %v, want ErrInvalidHMACTag", err)
	}
}

// TestRouterIngest_ShortDatagram_RejectedBeforeLookup verifies AC-005
// postcondition 4: a raw datagram shorter than 32 bytes (8-byte HMAC tag +
// 24-byte SVTNID/NodeAddr key selector) is rejected before any key lookup
// is attempted.
func TestRouterIngest_ShortDatagram_RejectedBeforeLookup(t *testing.T) {
	t.Parallel()

	router, _, _ := newAdmittedRouterForDiscoveryWire(t, svtnA)
	ri := discovery.NewRouterIngest(discovery.RouterIngestConfig{Router: router})

	cases := []struct {
		name string
		len  int
	}{
		{"raw=0 bytes", 0},
		{"raw=31 bytes (one short of the 32-byte key-selector minimum)", 31},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			defer redGateGuard(t)

			decision, err := ri.Ingest(make([]byte, tc.len))
			if !errors.Is(err, discovery.ErrInvalidHMACTag) {
				t.Errorf("Ingest(%d bytes): got err %v, want ErrInvalidHMACTag (AC-005 postcondition 4)", tc.len, err)
			}
			if decision.Accept {
				t.Errorf("Ingest(%d bytes): Accept = true, want false", tc.len)
			}
		})
	}
}

// TestRouterIngest_FullValidFrameMinimum_42Bytes verifies AC-005's note: the
// full valid-frame minimum with SEC-DW-07's Sequence field is 42 bytes
// (8 tag + 16 SVTNID + 8 NodeAddr + 8 Sequence + 2 count) with zero
// sessions, and this minimum-size frame is accepted, not rejected by the
// (unaffected, upstream) raw>=32/body>=24 pre-lookup guard.
func TestRouterIngest_FullValidFrameMinimum_42Bytes(t *testing.T) {
	t.Parallel()
	defer redGateGuard(t)

	router, pub, nodeAddr := newAdmittedRouterForDiscoveryWire(t, svtnA)
	key := testDeriveDiscoveryKey([]byte(pub), svtnA)
	ri := discovery.NewRouterIngest(discovery.RouterIngestConfig{Router: router})

	raw := buildHop1Datagram(key, svtnA, nodeAddr, 1, nil)
	if len(raw) != 42 {
		t.Fatalf("test setup: buildHop1Datagram with zero sessions produced %d bytes, want 42", len(raw))
	}
	decision, err := ri.Ingest(raw)
	if err != nil {
		t.Fatalf("Ingest: unexpected error on the minimum-size valid frame: %v", err)
	}
	if !decision.Accept {
		t.Error("Ingest: minimum-size 42-byte valid frame rejected, want accepted")
	}
}

// ---------------------------------------------------------------------------
// AC-006 — HMAC-first fail-closed verification with unified reject sentinel
// ---------------------------------------------------------------------------

// TestRouterIngest_LookupMissAndTagMismatch_IndistinguishableRejection
// verifies AC-006: a lookup-miss (unknown NodeAddr) and an HMAC-tag mismatch
// (known NodeAddr, wrong key) both resolve to the identical ErrInvalidHMACTag
// sentinel, no datagram is relayed, and no registry state is mutated on
// either rejection path.
func TestRouterIngest_LookupMissAndTagMismatch_IndistinguishableRejection(t *testing.T) {
	t.Parallel()

	router, pub, nodeAddr := newAdmittedRouterForDiscoveryWire(t, svtnA)
	key := testDeriveDiscoveryKey([]byte(pub), svtnA)
	ri := discovery.NewRouterIngest(discovery.RouterIngestConfig{Router: router})

	cases := []struct {
		name string
		raw  []byte
	}{
		{
			name: "lookup miss: unknown NodeAddr",
			raw:  buildHop1Datagram(key, svtnA, [8]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, 1, oneSession),
		},
		{
			name: "HMAC tag mismatch: known NodeAddr, wrong key",
			raw:  buildHop1Datagram([hmac.KeySize]byte{0xAB}, svtnA, nodeAddr, 1, oneSession),
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			defer redGateGuard(t)

			decision, err := ri.Ingest(tc.raw)
			if !errors.Is(err, discovery.ErrInvalidHMACTag) {
				t.Fatalf("Ingest(%s): got err %v, want ErrInvalidHMACTag", tc.name, err)
			}
			if decision.Accept {
				t.Errorf("Ingest(%s): decision.Accept = true, want false on rejection", tc.name)
			}
			if decision.Relay {
				t.Errorf("Ingest(%s): decision.Relay = true, want false on rejection", tc.name)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AC-007 — node-local relay-ingest: IngestRelayAdvertisement
// ---------------------------------------------------------------------------

// TestDiscovery_IngestRelayAdvertisement_SVTNMismatch_ErrSVTNMismatch
// verifies AC-007 postcondition 3: the relay frame's own OuterHeader.SVTNID
// is compared against d.cfg.LocalSVTNID, and ErrSVTNMismatch is returned on
// mismatch — defense-in-depth against a relay/routing bug, not a crypto
// check.
func TestDiscovery_IngestRelayAdvertisement_SVTNMismatch_ErrSVTNMismatch(t *testing.T) {
	t.Parallel()
	defer redGateGuard(t)

	cfg := newTestConfig(t, svtnA, nodeA1)
	d := discovery.New(cfg)

	payload := buildHop2Payload(nodeA2, 1, oneSession)
	err := d.IngestRelayAdvertisement(svtnB, payload) // svtnB != d's LocalSVTNID (svtnA)
	if !errors.Is(err, discovery.ErrSVTNMismatch) {
		t.Fatalf("IngestRelayAdvertisement: got err %v, want ErrSVTNMismatch", err)
	}
}

// TestDiscovery_IngestRelayAdvertisement_NoHMACRequired verifies AC-007
// postcondition 2: no per-frame HMAC is verified — trust derives from the
// already-admitted TCP connection the relay frame arrived on (AC-015). A
// payload built with zero cryptographic material must still be accepted
// when the SVTN matches.
func TestDiscovery_IngestRelayAdvertisement_NoHMACRequired(t *testing.T) {
	t.Parallel()
	defer redGateGuard(t)

	cfg := newTestConfig(t, svtnA, nodeA1)
	d := discovery.New(cfg)

	payload := buildHop2Payload(nodeA2, 1, oneSession)
	if err := d.IngestRelayAdvertisement(svtnA, payload); err != nil {
		t.Fatalf("IngestRelayAdvertisement: unexpected error: %v", err)
	}
}

// TestDiscovery_IngestRelayAdvertisement_Success_RegistryReplaceOnWrite
// verifies AC-007 postcondition 4: on success, the same registry
// replace-on-write update ReceiveAdvertisement previously performed for a
// given advertiser is applied.
func TestDiscovery_IngestRelayAdvertisement_Success_RegistryReplaceOnWrite(t *testing.T) {
	t.Parallel()
	defer redGateGuard(t)

	cfg := newTestConfig(t, svtnA, nodeA1)
	d := discovery.New(cfg)
	ctx := context.Background()

	first := buildHop2Payload(nodeA2, 1, []discovery.SessionPresence{
		{SessionName: "agent-02-old", Status: discovery.Attached, Quality: discovery.QualityGreen},
	})
	if err := d.IngestRelayAdvertisement(svtnA, first); err != nil {
		t.Fatalf("IngestRelayAdvertisement (first): unexpected error: %v", err)
	}

	second := buildHop2Payload(nodeA2, 2, []discovery.SessionPresence{
		{SessionName: "agent-02-new", Status: discovery.Attached, Quality: discovery.QualityGreen},
	})
	if err := d.IngestRelayAdvertisement(svtnA, second); err != nil {
		t.Fatalf("IngestRelayAdvertisement (second): unexpected error: %v", err)
	}

	entries, err := d.Enumerate(ctx)
	if err != nil {
		t.Fatalf("Enumerate: %v", err)
	}
	got := sessionsFromNodes(entries, nodeA2)
	if len(got) != 1 || got[0].Presence.SessionName != "agent-02-new" {
		t.Errorf("Enumerate after two IngestRelayAdvertisement calls = %+v, want exactly the replace-on-write result [agent-02-new]", got)
	}
}

// ---------------------------------------------------------------------------
// AC-008 — cold-start acceptance
// ---------------------------------------------------------------------------

// TestVP080_DiscoveryIngest_ColdStartAcceptance verifies AC-008: an
// HMAC-verified datagram with any declared Sequence (including 0) is
// accepted for a (SVTNID, NodeAddr) pair with no prior recorded Sequence.
func TestVP080_DiscoveryIngest_ColdStartAcceptance(t *testing.T) {
	t.Parallel()
	defer redGateGuard(t)

	router, pub, nodeAddr := newAdmittedRouterForDiscoveryWire(t, svtnA)
	key := testDeriveDiscoveryKey([]byte(pub), svtnA)
	ri := discovery.NewRouterIngest(discovery.RouterIngestConfig{Router: router})

	raw := buildHop1Datagram(key, svtnA, nodeAddr, 0, oneSession)
	decision, err := ri.Ingest(raw)
	if err != nil {
		t.Fatalf("Ingest: unexpected error on cold-start datagram: %v", err)
	}
	if !decision.Accept || !decision.Relay {
		t.Errorf("Ingest: decision = %+v, want Accept=true Relay=true (AC-008 cold start)", decision)
	}
	if decision.Sequence != 0 {
		t.Errorf("Ingest: decision.Sequence = %d, want 0", decision.Sequence)
	}
}

// ---------------------------------------------------------------------------
// AC-009 — replay/stale discard
// ---------------------------------------------------------------------------

// TestVP080_DiscoveryIngest_ReplayDiscard_ExactSequence verifies AC-009
// postconditions 1-3: a second HMAC-verified datagram declaring the exact
// same Sequence as the current lastSeen watermark is discarded — even
// though its HMAC passes.
func TestVP080_DiscoveryIngest_ReplayDiscard_ExactSequence(t *testing.T) {
	t.Parallel()
	defer redGateGuard(t)

	router, pub, nodeAddr := newAdmittedRouterForDiscoveryWire(t, svtnA)
	key := testDeriveDiscoveryKey([]byte(pub), svtnA)
	ri := discovery.NewRouterIngest(discovery.RouterIngestConfig{Router: router})

	if _, err := ri.Ingest(buildHop1Datagram(key, svtnA, nodeAddr, 5, oneSession)); err != nil {
		t.Fatalf("Ingest (establish lastSeen=5): %v", err)
	}

	decision, err := ri.Ingest(buildHop1Datagram(key, svtnA, nodeAddr, 5, oneSession))
	if err != nil {
		t.Fatalf("Ingest (exact replay): unexpected error: %v", err)
	}
	if !decision.Accept {
		t.Error("Ingest (exact replay): Accept = false, want true (HMAC still passes)")
	}
	if decision.Relay {
		t.Error("Ingest (exact replay): Relay = true, want false (non-increasing Sequence discarded)")
	}
}

// TestVP080_DiscoveryIngest_ReplayDiscard_LowerSequence verifies the same
// discard rule for a Sequence strictly lower than the current watermark.
func TestVP080_DiscoveryIngest_ReplayDiscard_LowerSequence(t *testing.T) {
	t.Parallel()
	defer redGateGuard(t)

	router, pub, nodeAddr := newAdmittedRouterForDiscoveryWire(t, svtnA)
	key := testDeriveDiscoveryKey([]byte(pub), svtnA)
	ri := discovery.NewRouterIngest(discovery.RouterIngestConfig{Router: router})

	if _, err := ri.Ingest(buildHop1Datagram(key, svtnA, nodeAddr, 10, oneSession)); err != nil {
		t.Fatalf("Ingest (establish lastSeen=10): %v", err)
	}
	decision, err := ri.Ingest(buildHop1Datagram(key, svtnA, nodeAddr, 3, oneSession))
	if err != nil {
		t.Fatalf("Ingest (lower sequence): unexpected error: %v", err)
	}
	if !decision.Accept {
		t.Error("Ingest (lower sequence): Accept = false, want true (HMAC passes)")
	}
	if decision.Relay {
		t.Error("Ingest (lower sequence): Relay = true, want false")
	}
}

// TestVP080_DiscoveryIngest_ReplayDiscard_NoRelaySideEffect verifies AC-009
// postcondition 4: lastSeen is unchanged by a discard — proven by a forward
// datagram declaring one more than the ORIGINAL lastSeen (not the discarded
// value) still being accepted and relayed after an intervening discard.
func TestVP080_DiscoveryIngest_ReplayDiscard_NoRelaySideEffect(t *testing.T) {
	t.Parallel()
	defer redGateGuard(t)

	router, pub, nodeAddr := newAdmittedRouterForDiscoveryWire(t, svtnA)
	key := testDeriveDiscoveryKey([]byte(pub), svtnA)
	ri := discovery.NewRouterIngest(discovery.RouterIngestConfig{Router: router})

	if _, err := ri.Ingest(buildHop1Datagram(key, svtnA, nodeAddr, 10, oneSession)); err != nil {
		t.Fatalf("Ingest (establish lastSeen=10): %v", err)
	}
	if _, err := ri.Ingest(buildHop1Datagram(key, svtnA, nodeAddr, 10, oneSession)); err != nil {
		t.Fatalf("Ingest (discard, exact replay): unexpected error: %v", err)
	}

	decision, err := ri.Ingest(buildHop1Datagram(key, svtnA, nodeAddr, 11, oneSession))
	if err != nil {
		t.Fatalf("Ingest (forward after discard): unexpected error: %v", err)
	}
	if !decision.Accept || !decision.Relay {
		t.Errorf("Ingest (forward after discard): decision = %+v, want Accept=true Relay=true — lastSeen must be unaffected by the intervening discard", decision)
	}
}

// ---------------------------------------------------------------------------
// AC-010 — forward acceptance
// ---------------------------------------------------------------------------

// TestVP080_DiscoveryIngest_ForwardAcceptance_AdvancesState verifies AC-010
// postconditions 1-4: a strictly-increasing Sequence is accepted, the
// accept+relay decision is emitted, and lastSeen advances.
func TestVP080_DiscoveryIngest_ForwardAcceptance_AdvancesState(t *testing.T) {
	t.Parallel()
	defer redGateGuard(t)

	router, pub, nodeAddr := newAdmittedRouterForDiscoveryWire(t, svtnA)
	key := testDeriveDiscoveryKey([]byte(pub), svtnA)
	ri := discovery.NewRouterIngest(discovery.RouterIngestConfig{Router: router})

	if _, err := ri.Ingest(buildHop1Datagram(key, svtnA, nodeAddr, 5, oneSession)); err != nil {
		t.Fatalf("Ingest (establish lastSeen=5): %v", err)
	}
	decision, err := ri.Ingest(buildHop1Datagram(key, svtnA, nodeAddr, 6, oneSession))
	if err != nil {
		t.Fatalf("Ingest (forward): unexpected error: %v", err)
	}
	if !decision.Accept || !decision.Relay {
		t.Errorf("Ingest (forward): decision = %+v, want Accept=true Relay=true", decision)
	}
	if decision.Sequence != 6 {
		t.Errorf("Ingest (forward): decision.Sequence = %d, want 6", decision.Sequence)
	}
}

// TestVP080_DiscoveryIngest_RestartForwardProgress verifies AC-010
// postcondition 6 (F-DWSP4-001): a restarted access node's first
// post-restart datagram — declaring a freshly-sampled epoch and a low
// counter — is accepted via forward acceptance (not AC-008's cold-start
// path, since a lastSeen entry already exists), because its composite
// Sequence exceeds the router's prior lastSeen watermark.
func TestVP080_DiscoveryIngest_RestartForwardProgress(t *testing.T) {
	t.Parallel()
	defer redGateGuard(t)

	router, pub, nodeAddr := newAdmittedRouterForDiscoveryWire(t, svtnA)
	key := testDeriveDiscoveryKey([]byte(pub), svtnA)
	ri := discovery.NewRouterIngest(discovery.RouterIngestConfig{Router: router})

	// Sequence composite: high 32 bits = UTC unix epoch seconds sampled at
	// Discovery-instance start, low 32 bits = the original counter.
	firstEpoch := uint64(1_700_000_000) // arbitrary but realistic UTC unix seconds
	firstSeq := firstEpoch<<32 | 42     // counter=42 before restart
	if _, err := ri.Ingest(buildHop1Datagram(key, svtnA, nodeAddr, firstSeq, oneSession)); err != nil {
		t.Fatalf("Ingest (pre-restart): %v", err)
	}

	restartEpoch := firstEpoch + 10 // process restarted 10s later
	restartSeq := restartEpoch<<32 | 1
	decision, err := ri.Ingest(buildHop1Datagram(key, svtnA, nodeAddr, restartSeq, oneSession))
	if err != nil {
		t.Fatalf("Ingest (post-restart): unexpected error: %v", err)
	}
	if !decision.Accept || !decision.Relay {
		t.Errorf("Ingest (post-restart): decision = %+v, want Accept=true Relay=true (restart forward progress)", decision)
	}
	if decision.Sequence != restartSeq {
		t.Errorf("Ingest (post-restart): decision.Sequence = %d, want %d", decision.Sequence, restartSeq)
	}
}

// ---------------------------------------------------------------------------
// AC-011 — bounded, fixed-size UDP read buffer
// ---------------------------------------------------------------------------

// TestRouterIngest_OversizedDatagram_RejectedNoPartialParse verifies AC-011:
// a datagram exceeding the sized buffer (a realistic worst-case legitimate
// advertisement, not the 65,507-byte UDP/IP theoretical maximum) is
// rejected without partial-parse.
func TestRouterIngest_OversizedDatagram_RejectedNoPartialParse(t *testing.T) {
	t.Parallel()
	defer redGateGuard(t)

	router, pub, nodeAddr := newAdmittedRouterForDiscoveryWire(t, svtnA)
	key := testDeriveDiscoveryKey([]byte(pub), svtnA)
	ri := discovery.NewRouterIngest(discovery.RouterIngestConfig{Router: router})

	giantName := strings.Repeat("x", 60000)
	sessions := []discovery.SessionPresence{{SessionName: giantName, Status: discovery.Attached, Quality: discovery.QualityGreen}}
	raw := buildHop1Datagram(key, svtnA, nodeAddr, 1, sessions)

	decision, err := ri.Ingest(raw)
	if err == nil {
		t.Fatalf("Ingest: oversized datagram (%d bytes) accepted, want rejection", len(raw))
	}
	if decision.Accept {
		t.Error("Ingest: oversized datagram Accept = true, want false (no partial parse)")
	}
}

// ---------------------------------------------------------------------------
// AC-012 — aggregate rate cap; FailureCounter visibility-only
// ---------------------------------------------------------------------------

// TestRouterIngest_AggregateRateCap_NotPerSource verifies AC-012
// postconditions 1 and 3: an aggregate (not per-source) token-bucket cap
// rejects datagrams once the aggregate rate is exceeded, and a source
// rotating its declared NodeAddr does not evade the cap. Two admitted
// identities alternate, each individually well within any sane per-source
// rate, but the combined burst volume must eventually trip the aggregate
// cap.
func TestRouterIngest_AggregateRateCap_NotPerSource(t *testing.T) {
	t.Parallel()
	defer redGateGuard(t)

	ks := admission.NewAdmittedKeySet()
	pubA, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key A: %v", err)
	}
	pubB, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key B: %v", err)
	}
	ks.RegisterKey(svtnA, pubA, admission.RoleAccess)
	ks.RegisterKey(svtnA, pubB, admission.RoleAccess)
	router := routing.NewRouter(ks)
	nodeA := frame.DeriveNodeAddress(svtnA, []byte(pubA))
	nodeB := frame.DeriveNodeAddress(svtnA, []byte(pubB))
	keyA := testDeriveDiscoveryKey([]byte(pubA), svtnA)
	keyB := testDeriveDiscoveryKey([]byte(pubB), svtnA)

	ri := discovery.NewRouterIngest(discovery.RouterIngestConfig{Router: router})

	const burst = 5000
	accepted := 0
	for i := 0; i < burst; i++ {
		var raw []byte
		if i%2 == 0 {
			raw = buildHop1Datagram(keyA, svtnA, nodeA, uint64(i/2+1), oneSession)
		} else {
			raw = buildHop1Datagram(keyB, svtnA, nodeB, uint64(i/2+1), oneSession)
		}
		decision, ingestErr := ri.Ingest(raw)
		if ingestErr == nil && decision.Accept {
			accepted++
		}
	}
	if accepted >= burst {
		t.Errorf("Ingest: all %d datagrams accepted across 2 rotated identities — aggregate rate cap did not engage (SEC-DW-03 postcondition 3)", accepted)
	}
}

// TestRouterIngest_FailureCounter_VisibilityOnly_NeverGates verifies AC-012
// postcondition 2: FailureCounter is invoked on HMAC-rejection events for
// operator visibility only — it never gates admission based on the
// attacker-controlled declared NodeAddr.
func TestRouterIngest_FailureCounter_VisibilityOnly_NeverGates(t *testing.T) {
	t.Parallel()
	defer redGateGuard(t)

	router, pub, nodeAddr := newAdmittedRouterForDiscoveryWire(t, svtnA)
	key := testDeriveDiscoveryKey([]byte(pub), svtnA)
	ri := discovery.NewRouterIngest(discovery.RouterIngestConfig{Router: router})

	// FailureCounter's shipped threshold is 5/60s (BC-2.05.005 PC-3) — send
	// more than that many HMAC failures from the SAME declared NodeAddr,
	// then confirm a genuinely valid datagram from that NodeAddr is still
	// accepted.
	var wrongKey [hmac.KeySize]byte
	wrongKey[0] = 0xFF
	for i := 0; i < 10; i++ {
		_, _ = ri.Ingest(buildHop1Datagram(wrongKey, svtnA, nodeAddr, uint64(i+1), oneSession))
	}

	decision, err := ri.Ingest(buildHop1Datagram(key, svtnA, nodeAddr, 100, oneSession))
	if err != nil {
		t.Fatalf("Ingest (valid, after 10 prior HMAC failures from the same NodeAddr): unexpected error: %v", err)
	}
	if !decision.Accept {
		t.Error("Ingest: valid datagram rejected after prior HMAC failures from the same declared NodeAddr — FailureCounter must never gate on this field (AC-012 postcondition 2)")
	}
}

// ---------------------------------------------------------------------------
// AC-013 — rate-limited, counter-based failure logging
// ---------------------------------------------------------------------------

// captureLogger is a routing.Logger test double that records every Log call
// for assertion.
type captureLogger struct {
	mu    sync.Mutex
	lines []string
}

func (c *captureLogger) Log(msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lines = append(c.lines, msg)
}

func (c *captureLogger) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.lines)
}

// TestRouterIngest_FailureLogging_ThresholdCrossingOnly_NotPerPacket
// verifies AC-013: discovery HMAC-rejection logging fires only on
// FailureCounter's own threshold-crossing emission (BC-2.05.005 PC-3,
// threshold=5/60s), not unconditionally per rejected packet — distinct from
// BC-2.05.008's per-packet TCP HMAC-failure logging policy.
func TestRouterIngest_FailureLogging_ThresholdCrossingOnly_NotPerPacket(t *testing.T) {
	t.Parallel()
	defer redGateGuard(t)

	router, _, nodeAddr := newAdmittedRouterForDiscoveryWire(t, svtnA)
	logger := &captureLogger{}
	ri := discovery.NewRouterIngest(discovery.RouterIngestConfig{Router: router, Logger: logger})

	var wrongKey [hmac.KeySize]byte
	wrongKey[0] = 0xFF

	// Below FailureCounter's threshold (5): must produce zero threshold-
	// crossing log lines — a per-packet policy would already have logged 3.
	const belowThreshold = 3
	for i := 0; i < belowThreshold; i++ {
		_, _ = ri.Ingest(buildHop1Datagram(wrongKey, svtnA, nodeAddr, uint64(i+1), oneSession))
	}
	if got := logger.count(); got != 0 {
		t.Errorf("after %d HMAC failures (below threshold=5): logger recorded %d lines, want 0 (not per-packet)", belowThreshold, got)
	}

	// Cross the threshold: at least one log line must now exist, but far
	// fewer than the total packet count sent so far (proving the emission
	// is threshold-crossing, not per-packet).
	const additionalToTrip = 5
	for i := 0; i < additionalToTrip; i++ {
		_, _ = ri.Ingest(buildHop1Datagram(wrongKey, svtnA, nodeAddr, uint64(belowThreshold+i+1), oneSession))
	}
	totalSent := belowThreshold + additionalToTrip
	if got := logger.count(); got == 0 {
		t.Errorf("after %d total HMAC failures (threshold=5 crossed): logger recorded 0 lines, want at least 1", totalSent)
	} else if got >= totalSent {
		t.Errorf("after %d total HMAC failures: logger recorded %d lines — not threshold-crossing, looks like per-packet logging (BC-2.05.008 policy, not adopted here)", totalSent, got)
	}
}
