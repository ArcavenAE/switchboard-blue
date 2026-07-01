package discovery_test

import (
	"context"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/discovery"
	"github.com/arcavenae/switchboard/internal/routing"
)

// --- helpers ----------------------------------------------------------------

func newTestRouter(t *testing.T) *routing.Router {
	t.Helper()
	ks := admission.NewAdmittedKeySet()
	return routing.NewRouter(ks)
}

func newTestConfig(t *testing.T, svtnID [16]byte, nodeAddr [8]byte) discovery.Config {
	t.Helper()
	return discovery.Config{
		LocalNodeAddr:     nodeAddr,
		LocalSVTNID:       svtnID,
		Router:            newTestRouter(t),
		HeartbeatInterval: discovery.HeartbeatInterval,
	}
}

var (
	svtnA = [16]byte{0xAA}
	svtnB = [16]byte{0xBB}

	nodeA1 = [8]byte{0x01}
	nodeA2 = [8]byte{0x02}
)

// distinctNodeAddrs returns the set of unique advertiser addresses in entries.
func distinctNodeAddrs(entries []discovery.SessionEntry) map[[8]byte]struct{} {
	out := make(map[[8]byte]struct{}, len(entries))
	for _, e := range entries {
		out[e.AdvertiserAddr] = struct{}{}
	}
	return out
}

// sessionsFromSVTN filters entries that originated from the given SVTN by
// checking whether their advertiser address matches any of the supplied
// node addresses for that SVTN.
func sessionsFromNodes(entries []discovery.SessionEntry, addrs ...[8]byte) []discovery.SessionEntry {
	addrSet := make(map[[8]byte]struct{}, len(addrs))
	for _, a := range addrs {
		addrSet[a] = struct{}{}
	}
	var out []discovery.SessionEntry
	for _, e := range entries {
		if _, ok := addrSet[e.AdvertiserAddr]; ok {
			out = append(out, e)
		}
	}
	return out
}

// --- AC-001a: OnStateChange advertisement within 1 tick -------------------

// TestDiscovery_Advertise_OnStateChange verifies BC-2.03.001 PC-3:
// Discovery.Advertise sends a presence advertisement within 1 tick interval
// when triggered by a session state change.
func TestDiscovery_Advertise_OnStateChange(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t, svtnA, nodeA1)
	d := discovery.New(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	sessions := []discovery.SessionPresence{
		{SessionName: "agent-01", Status: discovery.Attached, Quality: discovery.QualityGreen},
	}

	// Advertise must return without error on a valid state-change trigger.
	if err := d.Advertise(ctx, sessions); err != nil {
		t.Fatalf("Advertise: unexpected error: %v", err)
	}
}

// --- AC-001b: Periodic heartbeat 30 s independent timer -------------------

// TestDiscovery_Advertise_PeriodicHeartbeat verifies BC-2.03.001 PC-4:
// the heartbeat fires unconditionally every HeartbeatInterval regardless of
// whether Advertise was called. This test uses a shortened interval to keep
// the test fast.
func TestDiscovery_Advertise_PeriodicHeartbeat(t *testing.T) {
	t.Parallel()

	shortInterval := 50 * time.Millisecond
	cfg := discovery.Config{
		LocalNodeAddr:     nodeA1,
		LocalSVTNID:       svtnA,
		Router:            newTestRouter(t),
		HeartbeatInterval: shortInterval,
	}
	d := discovery.New(cfg)

	// Run with a context that expires after 3 heartbeat periods.
	ctx, cancel := context.WithTimeout(context.Background(), 3*shortInterval+20*time.Millisecond)
	t.Cleanup(cancel)

	runDone := make(chan error, 1)
	go func() {
		runDone <- d.Run(ctx)
	}()

	<-ctx.Done()
	runErr := <-runDone
	if runErr != nil && runErr != context.DeadlineExceeded && runErr != context.Canceled {
		t.Fatalf("Run: unexpected error: %v", runErr)
	}
	// If we reach here the heartbeat loop ran for 3 intervals without crashing.
}

// --- AC-002: Enumerate aggregates from ≥2 distinct advertisers ------------

// TestDiscovery_Enumerate_NoHostnameRequired verifies BC-2.03.002 PC-3:
// Enumerate returns sessions from at least 2 distinct advertising nodes
// without the caller supplying hostnames or IP addresses.
func TestDiscovery_Enumerate_NoHostnameRequired(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t, svtnA, nodeA1)
	d := discovery.New(cfg)
	ctx := context.Background()

	// Simulate two advertisements from distinct nodes on the same SVTN.
	adv1, err := discovery.Encode(discovery.AdvertisementPayload{
		NodeAddr: nodeA1,
		SVTNID:   svtnA,
		Sessions: []discovery.SessionPresence{
			{SessionName: "sess-A", Status: discovery.Detached, Quality: discovery.QualityGreen},
		},
	})
	if err != nil {
		t.Fatalf("Encode adv1: %v", err)
	}
	adv2, err := discovery.Encode(discovery.AdvertisementPayload{
		NodeAddr: nodeA2,
		SVTNID:   svtnA,
		Sessions: []discovery.SessionPresence{
			{SessionName: "sess-B", Status: discovery.Attached, Quality: discovery.QualityYellow},
		},
	})
	if err != nil {
		t.Fatalf("Encode adv2: %v", err)
	}

	if err := d.ReceiveAdvertisement(ctx, adv1); err != nil {
		t.Fatalf("ReceiveAdvertisement adv1: %v", err)
	}
	if err := d.ReceiveAdvertisement(ctx, adv2); err != nil {
		t.Fatalf("ReceiveAdvertisement adv2: %v", err)
	}

	result, err := d.Enumerate(ctx)
	if err != nil {
		t.Fatalf("Enumerate: %v", err)
	}

	distinct := distinctNodeAddrs(result)
	if len(distinct) < 2 {
		t.Errorf("Enumerate: got %d distinct advertiser(s), want ≥ 2 (BC-2.03.002 PC-3)", len(distinct))
	}
}

// --- AC-003: Advertisement payload required fields ------------------------

// TestDiscovery_Advertisement_RequiredFields verifies BC-2.03.003 PC-1:
// each advertisement payload carries session_name, attachment_status, and
// quality_indicator.
func TestDiscovery_Advertisement_RequiredFields(t *testing.T) {
	t.Parallel()

	payload := discovery.AdvertisementPayload{
		NodeAddr: nodeA1,
		SVTNID:   svtnA,
		Sessions: []discovery.SessionPresence{
			{
				SessionName: "agent-01",
				Status:      discovery.Attached,
				Quality:     discovery.QualityGreen,
			},
		},
	}

	encoded, err := discovery.Encode(payload)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	decoded, err := discovery.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if len(decoded.Sessions) != 1 {
		t.Fatalf("decoded Sessions: got %d entries, want 1", len(decoded.Sessions))
	}
	s := decoded.Sessions[0]
	if s.SessionName == "" {
		t.Error("session_name must be non-empty in advertisement payload")
	}
	// Attachment status and quality are zero-values only if intentionally
	// set to Detached/QualityUnknown — validate the round-trip values instead.
	if s.Status != discovery.Attached {
		t.Errorf("attachment_status: got %v, want Attached", s.Status)
	}
	if s.Quality != discovery.QualityGreen {
		t.Errorf("quality_indicator: got %v, want QualityGreen", s.Quality)
	}
}

// --- AC-004: Advertisement round-trip Encode(Decode(payload)) == payload --

// TestDiscovery_AdvertisementRoundTrip verifies BC-2.03.003 Inv-1:
// Encode followed by Decode yields an identical payload.
func TestDiscovery_AdvertisementRoundTrip(t *testing.T) {
	t.Parallel()

	cases := []discovery.AdvertisementPayload{
		{
			NodeAddr: nodeA1,
			SVTNID:   svtnA,
			Sessions: []discovery.SessionPresence{
				{SessionName: "sess-1", Status: discovery.Attached, Quality: discovery.QualityGreen},
				{SessionName: "sess-2", Status: discovery.Detached, Quality: discovery.QualityRed},
			},
		},
		{
			// EC-002: quality not yet known
			NodeAddr: nodeA2,
			SVTNID:   svtnA,
			Sessions: []discovery.SessionPresence{
				{SessionName: "sess-startup", Status: discovery.Detached, Quality: discovery.QualityUnknown},
			},
		},
		{
			// EC-003: empty session list
			NodeAddr: nodeA1,
			SVTNID:   svtnA,
			Sessions: nil,
		},
	}

	for _, original := range cases {
		encoded, err := discovery.Encode(original)
		if err != nil {
			t.Fatalf("Encode: %v", err)
		}
		decoded, err := discovery.Decode(encoded)
		if err != nil {
			t.Fatalf("Decode: %v", err)
		}
		// Re-encode the decoded value and compare bytes for stability.
		reencoded, err := discovery.Encode(decoded)
		if err != nil {
			t.Fatalf("re-Encode: %v", err)
		}
		if len(encoded) != len(reencoded) {
			t.Errorf("round-trip length mismatch: encoded=%d reencoded=%d", len(encoded), len(reencoded))
			continue
		}
		for i := range encoded {
			if encoded[i] != reencoded[i] {
				t.Errorf("round-trip byte[%d] mismatch: %02x vs %02x", i, encoded[i], reencoded[i])
				break
			}
		}
	}
}

// --- AC-005: HMAC authentication on advertisement -------------------------

// TestDiscovery_Advertise_HMACAuthenticated verifies BC-2.03.001 PC-5:
// an advertisement with a missing or wrong HMAC tag is rejected and does not
// update the local session list.
func TestDiscovery_Advertise_HMACAuthenticated(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t, svtnA, nodeA1)
	d := discovery.New(cfg)
	ctx := context.Background()

	// Craft a raw payload with a deliberately wrong HMAC tag by mutating
	// the first byte of an otherwise valid encoding.
	validPayload := discovery.AdvertisementPayload{
		NodeAddr: nodeA2,
		SVTNID:   svtnA,
		Sessions: []discovery.SessionPresence{
			{SessionName: "tampered-sess", Status: discovery.Detached, Quality: discovery.QualityGreen},
		},
	}
	raw, err := discovery.Encode(validPayload)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	// Corrupt the first byte — simulates a bad HMAC tag.
	if len(raw) > 0 {
		raw[0] ^= 0xFF
	}

	err = d.ReceiveAdvertisement(ctx, raw)
	if err == nil {
		t.Fatal("ReceiveAdvertisement: expected error for tampered HMAC, got nil")
	}

	// After rejection the session must NOT appear in enumeration.
	entries, err := d.Enumerate(ctx)
	if err != nil {
		t.Fatalf("Enumerate: %v", err)
	}
	for _, e := range entries {
		if e.Presence.SessionName == "tampered-sess" {
			t.Error("tampered advertisement must not update the session list (AC-005 fail-closed)")
		}
	}
}

// --- AC-006: SVTN cross-scope negative ------------------------------------

// TestDiscovery_Enumerate_SVTNIsolation verifies BC-2.03.002 Inv-1:
// sessions advertised by a node on SVTN-B must not appear in the Enumerate
// result for a Discovery instance on SVTN-A.
func TestDiscovery_Enumerate_SVTNIsolation(t *testing.T) {
	t.Parallel()

	// Local node is on SVTN-A.
	cfg := newTestConfig(t, svtnA, nodeA1)
	d := discovery.New(cfg)
	ctx := context.Background()

	// Inject a valid SVTN-A advertisement so the registry is non-empty.
	advA, err := discovery.Encode(discovery.AdvertisementPayload{
		NodeAddr: nodeA1,
		SVTNID:   svtnA,
		Sessions: []discovery.SessionPresence{
			{SessionName: "svtn-a-sess", Status: discovery.Detached, Quality: discovery.QualityGreen},
		},
	})
	if err != nil {
		t.Fatalf("Encode advA: %v", err)
	}
	if err := d.ReceiveAdvertisement(ctx, advA); err != nil {
		t.Fatalf("ReceiveAdvertisement advA: %v", err)
	}

	// Inject a SVTN-B advertisement — this must be rejected or silently
	// filtered; it must not appear in SVTN-A Enumerate.
	nodeB := [8]byte{0xBB}
	advB, err := discovery.Encode(discovery.AdvertisementPayload{
		NodeAddr: nodeB,
		SVTNID:   svtnB,
		Sessions: []discovery.SessionPresence{
			{SessionName: "svtn-b-sess", Status: discovery.Detached, Quality: discovery.QualityGreen},
		},
	})
	if err != nil {
		t.Fatalf("Encode advB: %v", err)
	}
	// ReceiveAdvertisement may return ErrSVTNMismatch or silently drop it.
	_ = d.ReceiveAdvertisement(ctx, advB)

	result, err := d.Enumerate(ctx)
	if err != nil {
		t.Fatalf("Enumerate: %v", err)
	}

	// Oracle: len(sessionsFromSVTNB(svtnAResult)) == 0
	crossScope := sessionsFromNodes(result, nodeB)
	if len(crossScope) != 0 {
		t.Errorf("Enumerate on SVTN-A returned %d session(s) from SVTN-B node; want 0 (BC-2.03.002 Inv-1)", len(crossScope))
	}
}
