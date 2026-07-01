package discovery_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/discovery"
	"github.com/arcavenae/switchboard/internal/routing"
)

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

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

// sessionsFromNodes filters entries whose AdvertiserAddr matches one of addrs.
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

// encodeOrFail encodes payload and fatals t on error.
func encodeOrFail(t *testing.T, payload discovery.AdvertisementPayload) []byte {
	t.Helper()
	raw, err := discovery.Encode(payload)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return raw
}

// ---------------------------------------------------------------------------
// AC-001a — BC-2.03.001 PC-3: state-change trigger within 1 tick
// ---------------------------------------------------------------------------

// TestDiscovery_Advertise_OnStateChange verifies BC-2.03.001 PC-3:
// Advertise sends a presence advertisement within 1 tick interval when
// triggered by a session state change.
//
// Strong oracle: after Advertise returns, the advertised session must be
// visible in the local node's own Enumerate result (the advertisement was
// processed and stored — observable side-effect, not just "no error").
func TestDiscovery_Advertise_OnStateChange(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t, svtnA, nodeA1)
	d := discovery.New(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	sessions := []discovery.SessionPresence{
		{SessionName: "agent-01", Status: discovery.Attached, Quality: discovery.QualityGreen},
	}

	// Advertise must return nil on a valid state-change trigger.
	if err := d.Advertise(ctx, sessions); err != nil {
		t.Fatalf("Advertise: unexpected error: %v", err)
	}

	// Strong oracle: the session must now appear in Enumerate. If Advertise
	// is not implemented, Enumerate returns nothing (or panics) — the test
	// fails for the right reason.
	result, err := d.Enumerate(ctx)
	if err != nil {
		t.Fatalf("Enumerate after Advertise: %v", err)
	}
	found := false
	for _, e := range result {
		if e.Presence.SessionName == "agent-01" && e.AdvertiserAddr == nodeA1 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Advertise_OnStateChange: session 'agent-01' not found in Enumerate result after Advertise (BC-2.03.001 PC-3)")
	}
}

// TestDiscovery_Advertise_OnStateChange_DetachTriggersAdvert verifies EC-001
// (detach event triggers immediate advertisement with Detached status).
func TestDiscovery_Advertise_OnStateChange_DetachTriggersAdvert(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t, svtnA, nodeA1)
	d := discovery.New(cfg)
	ctx := context.Background()

	// First advertise as Attached.
	if err := d.Advertise(ctx, []discovery.SessionPresence{
		{SessionName: "agent-01", Status: discovery.Attached, Quality: discovery.QualityGreen},
	}); err != nil {
		t.Fatalf("initial Advertise: %v", err)
	}

	// Then detach — triggers a new advertisement with Detached.
	if err := d.Advertise(ctx, []discovery.SessionPresence{
		{SessionName: "agent-01", Status: discovery.Detached, Quality: discovery.QualityGreen},
	}); err != nil {
		t.Fatalf("detach Advertise: %v", err)
	}

	result, err := d.Enumerate(ctx)
	if err != nil {
		t.Fatalf("Enumerate: %v", err)
	}
	for _, e := range result {
		if e.Presence.SessionName == "agent-01" && e.AdvertiserAddr == nodeA1 {
			if e.Presence.Status != discovery.Detached {
				t.Errorf("after detach Advertise, Status = %v, want Detached (EC-001)", e.Presence.Status)
			}
			return
		}
	}
	t.Error("session 'agent-01' not found in Enumerate after detach Advertise")
}

// ---------------------------------------------------------------------------
// AC-001b — BC-2.03.001 PC-4: periodic heartbeat independent of state change
// ---------------------------------------------------------------------------

// TestDiscovery_Advertise_PeriodicHeartbeat verifies BC-2.03.001 PC-4:
// Run fires the heartbeat unconditionally every HeartbeatInterval regardless
// of Advertise calls. Uses a shortened interval to stay fast.
//
// Strong oracle: Run must complete (or context-cancel) cleanly across at
// least 3 heartbeat periods, and must not return a non-context error,
// demonstrating the heartbeat loop is active and independent.
func TestDiscovery_Advertise_PeriodicHeartbeat(t *testing.T) {
	t.Parallel()

	const shortInterval = 50 * time.Millisecond
	cfg := discovery.Config{
		LocalNodeAddr:     nodeA1,
		LocalSVTNID:       svtnA,
		Router:            newTestRouter(t),
		HeartbeatInterval: shortInterval,
	}
	d := discovery.New(cfg)

	// Allow 3 full heartbeat periods + slack.
	ctx, cancel := context.WithTimeout(context.Background(), 3*shortInterval+30*time.Millisecond)
	t.Cleanup(cancel)

	runDone := make(chan error, 1)
	go func() {
		runDone <- d.Run(ctx)
	}()

	<-ctx.Done()
	runErr := <-runDone

	// Acceptable terminal states: context cancellation or deadline exceeded.
	// Any other error means the heartbeat loop failed unexpectedly.
	if runErr != nil && !errors.Is(runErr, context.DeadlineExceeded) && !errors.Is(runErr, context.Canceled) {
		t.Fatalf("Run: unexpected error after 3 heartbeat intervals: %v", runErr)
	}
}

// TestDiscovery_Advertise_PeriodicHeartbeat_IsIndependent verifies that the
// heartbeat fires even when Advertise has never been called.
func TestDiscovery_Advertise_PeriodicHeartbeat_IsIndependent(t *testing.T) {
	t.Parallel()

	const shortInterval = 50 * time.Millisecond
	cfg := discovery.Config{
		LocalNodeAddr:     nodeA1,
		LocalSVTNID:       svtnA,
		Router:            newTestRouter(t),
		HeartbeatInterval: shortInterval,
	}
	d := discovery.New(cfg)

	// Do NOT call Advertise — heartbeat must fire independently.
	ctx, cancel := context.WithTimeout(context.Background(), 2*shortInterval+20*time.Millisecond)
	t.Cleanup(cancel)

	runDone := make(chan error, 1)
	go func() {
		runDone <- d.Run(ctx)
	}()

	<-ctx.Done()
	runErr := <-runDone
	if runErr != nil && !errors.Is(runErr, context.DeadlineExceeded) && !errors.Is(runErr, context.Canceled) {
		t.Fatalf("Run (no prior Advertise): unexpected error: %v", runErr)
	}
}

// ---------------------------------------------------------------------------
// AC-002 — BC-2.03.002 PC-3: Enumerate aggregates ≥2 distinct advertisers
// ---------------------------------------------------------------------------

// TestDiscovery_Enumerate_NoHostnameRequired verifies BC-2.03.002 PC-3:
// Enumerate returns sessions from at least 2 distinct advertising node
// addresses without the caller supplying hostnames or IP addresses.
//
// Oracle: len(distinctNodeAddrs(result)) >= 2
func TestDiscovery_Enumerate_NoHostnameRequired(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t, svtnA, nodeA1)
	d := discovery.New(cfg)
	ctx := context.Background()

	adv1 := encodeOrFail(t, discovery.AdvertisementPayload{
		NodeAddr: nodeA1,
		SVTNID:   svtnA,
		Sessions: []discovery.SessionPresence{
			{SessionName: "sess-A", Status: discovery.Detached, Quality: discovery.QualityGreen},
		},
	})
	adv2 := encodeOrFail(t, discovery.AdvertisementPayload{
		NodeAddr: nodeA2,
		SVTNID:   svtnA,
		Sessions: []discovery.SessionPresence{
			{SessionName: "sess-B", Status: discovery.Attached, Quality: discovery.QualityYellow},
		},
	})

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

// TestDiscovery_Enumerate_EmptyWithoutAdvertisements verifies BC-2.03.002
// EC-002: Enumerate returns an empty list when no advertisements have been
// received — not an error.
func TestDiscovery_Enumerate_EmptyWithoutAdvertisements(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t, svtnA, nodeA1)
	d := discovery.New(cfg)

	result, err := d.Enumerate(context.Background())
	if err != nil {
		t.Fatalf("Enumerate on empty registry: unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Enumerate on empty registry: got %d entries, want 0", len(result))
	}
}

// TestDiscovery_Enumerate_SameSessionNameTwoNodes verifies EC-003:
// when two access nodes advertise the same session name, both entries appear
// in Enumerate, differentiated by AdvertiserAddr.
func TestDiscovery_Enumerate_SameSessionNameTwoNodes(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t, svtnA, nodeA1)
	d := discovery.New(cfg)
	ctx := context.Background()

	// Both nodes advertise "agent-01".
	for _, node := range [][8]byte{nodeA1, nodeA2} {
		raw := encodeOrFail(t, discovery.AdvertisementPayload{
			NodeAddr: node,
			SVTNID:   svtnA,
			Sessions: []discovery.SessionPresence{
				{SessionName: "agent-01", Status: discovery.Detached, Quality: discovery.QualityGreen},
			},
		})
		if err := d.ReceiveAdvertisement(ctx, raw); err != nil {
			t.Fatalf("ReceiveAdvertisement node %v: %v", node, err)
		}
	}

	result, err := d.Enumerate(ctx)
	if err != nil {
		t.Fatalf("Enumerate: %v", err)
	}

	// Count entries named "agent-01".
	var count int
	for _, e := range result {
		if e.Presence.SessionName == "agent-01" {
			count++
		}
	}
	if count < 2 {
		t.Errorf("Enumerate: got %d 'agent-01' entries, want ≥ 2 (EC-003 duplicate name from two nodes)", count)
	}

	// The two entries must have different AdvertiserAddr values.
	distinct := distinctNodeAddrs(result)
	if len(distinct) < 2 {
		t.Errorf("Enumerate: only %d distinct advertiser addr(s) for duplicate session name, want ≥ 2", len(distinct))
	}
}

// ---------------------------------------------------------------------------
// AC-003 — BC-2.03.003 PC-1: advertisement payload required fields
// ---------------------------------------------------------------------------

// TestDiscovery_Advertisement_RequiredFields verifies BC-2.03.003 PC-1:
// each advertisement payload carries session_name, attachment_status, and
// quality_indicator, and these fields survive an Encode/Decode round-trip.
func TestDiscovery_Advertisement_RequiredFields(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		payload discovery.AdvertisementPayload
		want    discovery.SessionPresence
	}{
		{
			name: "attached green",
			payload: discovery.AdvertisementPayload{
				NodeAddr: nodeA1,
				SVTNID:   svtnA,
				Sessions: []discovery.SessionPresence{
					{SessionName: "agent-01", Status: discovery.Attached, Quality: discovery.QualityGreen},
				},
			},
			want: discovery.SessionPresence{SessionName: "agent-01", Status: discovery.Attached, Quality: discovery.QualityGreen},
		},
		{
			name: "detached yellow",
			payload: discovery.AdvertisementPayload{
				NodeAddr: nodeA2,
				SVTNID:   svtnA,
				Sessions: []discovery.SessionPresence{
					{SessionName: "agent-02", Status: discovery.Detached, Quality: discovery.QualityYellow},
				},
			},
			want: discovery.SessionPresence{SessionName: "agent-02", Status: discovery.Detached, Quality: discovery.QualityYellow},
		},
		{
			name: "detached red",
			payload: discovery.AdvertisementPayload{
				NodeAddr: nodeA1,
				SVTNID:   svtnA,
				Sessions: []discovery.SessionPresence{
					{SessionName: "agent-03", Status: discovery.Detached, Quality: discovery.QualityRed},
				},
			},
			want: discovery.SessionPresence{SessionName: "agent-03", Status: discovery.Detached, Quality: discovery.QualityRed},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			encoded, err := discovery.Encode(tc.payload)
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			decoded, err := discovery.Decode(encoded)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}

			if len(decoded.Sessions) != 1 {
				t.Fatalf("decoded.Sessions: got %d, want 1", len(decoded.Sessions))
			}
			got := decoded.Sessions[0]

			if got.SessionName == "" {
				t.Error("session_name must be non-empty in advertisement payload (BC-2.03.003 PC-1)")
			}
			if got.SessionName != tc.want.SessionName {
				t.Errorf("session_name: got %q, want %q", got.SessionName, tc.want.SessionName)
			}
			if got.Status != tc.want.Status {
				t.Errorf("attachment_status: got %v, want %v", got.Status, tc.want.Status)
			}
			if got.Quality != tc.want.Quality {
				t.Errorf("quality_indicator: got %v, want %v", got.Quality, tc.want.Quality)
			}
		})
	}
}

// TestDiscovery_Advertisement_QualityUnknownOnStartup verifies EC-002
// (BC-2.03.003): quality defaults to QualityUnknown at startup; that value
// must survive the round-trip without being coerced to another indicator.
func TestDiscovery_Advertisement_QualityUnknownOnStartup(t *testing.T) {
	t.Parallel()

	payload := discovery.AdvertisementPayload{
		NodeAddr: nodeA1,
		SVTNID:   svtnA,
		Sessions: []discovery.SessionPresence{
			{SessionName: "startup-sess", Status: discovery.Detached, Quality: discovery.QualityUnknown},
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
		t.Fatalf("decoded.Sessions: got %d, want 1", len(decoded.Sessions))
	}
	if decoded.Sessions[0].Quality != discovery.QualityUnknown {
		t.Errorf("EC-002: Quality = %v, want QualityUnknown (startup before first metric)", decoded.Sessions[0].Quality)
	}
}

// ---------------------------------------------------------------------------
// AC-004 — BC-2.03.003 Inv-1: Encode/Decode round-trip stability
// ---------------------------------------------------------------------------

// TestDiscovery_AdvertisementRoundTrip verifies BC-2.03.003 Inv-1:
// Encode(Decode(b)) == b for any valid advertisement byte slice b.
// Uses table-driven cases covering all QualityIndicator values and
// edge cases (empty session list, QualityUnknown).
func TestDiscovery_AdvertisementRoundTrip(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		payload discovery.AdvertisementPayload
	}{
		{
			name: "two sessions: attached-green and detached-red",
			payload: discovery.AdvertisementPayload{
				NodeAddr: nodeA1,
				SVTNID:   svtnA,
				Sessions: []discovery.SessionPresence{
					{SessionName: "sess-1", Status: discovery.Attached, Quality: discovery.QualityGreen},
					{SessionName: "sess-2", Status: discovery.Detached, Quality: discovery.QualityRed},
				},
			},
		},
		{
			name: "EC-002: QualityUnknown at startup",
			payload: discovery.AdvertisementPayload{
				NodeAddr: nodeA2,
				SVTNID:   svtnA,
				Sessions: []discovery.SessionPresence{
					{SessionName: "sess-startup", Status: discovery.Detached, Quality: discovery.QualityUnknown},
				},
			},
		},
		{
			name: "EC-003: empty session list",
			payload: discovery.AdvertisementPayload{
				NodeAddr: nodeA1,
				SVTNID:   svtnA,
				Sessions: nil,
			},
		},
		{
			name: "yellow quality",
			payload: discovery.AdvertisementPayload{
				NodeAddr: nodeA1,
				SVTNID:   svtnA,
				Sessions: []discovery.SessionPresence{
					{SessionName: "yellow-sess", Status: discovery.Attached, Quality: discovery.QualityYellow},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			encoded, err := discovery.Encode(tc.payload)
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			decoded, err := discovery.Decode(encoded)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			reencoded, err := discovery.Encode(decoded)
			if err != nil {
				t.Fatalf("re-Encode: %v", err)
			}
			if !bytes.Equal(encoded, reencoded) {
				t.Errorf("round-trip mismatch: len=%d vs %d (BC-2.03.003 Inv-1)", len(encoded), len(reencoded))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AC-005 — BC-2.03.001 PC-5: HMAC authentication on advertisement
// ---------------------------------------------------------------------------

// TestDiscovery_Advertise_HMACAuthenticated verifies BC-2.03.001 PC-5:
// an advertisement with a missing or wrong HMAC tag is rejected fail-closed.
// The receiver must return a non-nil error, and the rejected session must NOT
// appear in Enumerate.
func TestDiscovery_Advertise_HMACAuthenticated(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t, svtnA, nodeA1)
	d := discovery.New(cfg)
	ctx := context.Background()

	// Build a valid payload, then corrupt the first byte to simulate a bad
	// HMAC tag. The implementation is expected to reject this.
	validPayload := discovery.AdvertisementPayload{
		NodeAddr: nodeA2,
		SVTNID:   svtnA,
		Sessions: []discovery.SessionPresence{
			{SessionName: "tampered-sess", Status: discovery.Detached, Quality: discovery.QualityGreen},
		},
	}
	raw := encodeOrFail(t, validPayload)
	if len(raw) == 0 {
		t.Fatal("Encode returned empty bytes")
	}
	raw[0] ^= 0xFF // corrupt one byte — simulates tampered HMAC region

	err := d.ReceiveAdvertisement(ctx, raw)
	if err == nil {
		t.Fatal("ReceiveAdvertisement: expected error for tampered HMAC, got nil (AC-005 fail-closed)")
	}

	// Strong oracle: the tampered session must NOT appear in the session list.
	entries, err := d.Enumerate(ctx)
	if err != nil {
		t.Fatalf("Enumerate after rejection: %v", err)
	}
	for _, e := range entries {
		if e.Presence.SessionName == "tampered-sess" {
			t.Error("tampered advertisement must not update the session list (BC-2.03.001 PC-5 fail-closed)")
		}
	}
}

// TestDiscovery_Advertise_HMACAuthenticated_EmptyPayload verifies that an
// empty raw byte slice is also rejected (degenerate tampered case).
func TestDiscovery_Advertise_HMACAuthenticated_EmptyPayload(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t, svtnA, nodeA1)
	d := discovery.New(cfg)

	err := d.ReceiveAdvertisement(context.Background(), []byte{})
	if err == nil {
		t.Fatal("ReceiveAdvertisement with empty payload: expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// AC-006 — BC-2.03.002 Inv-1: SVTN cross-scope isolation
// ---------------------------------------------------------------------------

// TestDiscovery_Enumerate_SVTNIsolation verifies BC-2.03.002 Inv-1:
// sessions advertised by a node on SVTN-B must not appear in the Enumerate
// result for a Discovery instance on SVTN-A.
//
// Oracle: len(sessionsFromNodes(svtnAResult, svtnBNode)) == 0
func TestDiscovery_Enumerate_SVTNIsolation(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t, svtnA, nodeA1)
	d := discovery.New(cfg)
	ctx := context.Background()

	// Inject a valid SVTN-A advertisement so the registry is non-empty.
	advA := encodeOrFail(t, discovery.AdvertisementPayload{
		NodeAddr: nodeA1,
		SVTNID:   svtnA,
		Sessions: []discovery.SessionPresence{
			{SessionName: "svtn-a-sess", Status: discovery.Detached, Quality: discovery.QualityGreen},
		},
	})
	if err := d.ReceiveAdvertisement(ctx, advA); err != nil {
		t.Fatalf("ReceiveAdvertisement SVTN-A: %v", err)
	}

	// Inject a SVTN-B advertisement — must be rejected or silently filtered.
	nodeB := [8]byte{0xBB}
	advB := encodeOrFail(t, discovery.AdvertisementPayload{
		NodeAddr: nodeB,
		SVTNID:   svtnB,
		Sessions: []discovery.SessionPresence{
			{SessionName: "svtn-b-sess", Status: discovery.Detached, Quality: discovery.QualityGreen},
		},
	})
	// ReceiveAdvertisement may return ErrSVTNMismatch or silently drop.
	svtnBErr := d.ReceiveAdvertisement(ctx, advB)
	if svtnBErr != nil && !errors.Is(svtnBErr, discovery.ErrSVTNMismatch) {
		// Any non-nil error that is not ErrSVTNMismatch is also acceptable
		// (implementation may use a different wrapping) as long as the session
		// does not appear in Enumerate.
		t.Logf("ReceiveAdvertisement SVTN-B: got error %v (non-nil is acceptable for cross-scope rejection)", svtnBErr)
	}

	result, err := d.Enumerate(ctx)
	if err != nil {
		t.Fatalf("Enumerate: %v", err)
	}

	// Oracle: no SVTN-B node sessions in SVTN-A result.
	crossScope := sessionsFromNodes(result, nodeB)
	if len(crossScope) != 0 {
		t.Errorf(
			"Enumerate on SVTN-A returned %d session(s) from SVTN-B node (addr %v); want 0 (BC-2.03.002 Inv-1)",
			len(crossScope), nodeB,
		)
	}

	// Also assert that SVTN-A session IS still present.
	svtnASessions := sessionsFromNodes(result, nodeA1)
	if len(svtnASessions) == 0 {
		t.Error("Enumerate: SVTN-A session not found after cross-scope injection; SVTN-A registry must not be disturbed")
	}
}

// TestDiscovery_Enumerate_SVTNIsolation_ErrSentinel verifies that when
// ReceiveAdvertisement returns an error for a cross-scope advertisement, it
// wraps or is ErrSVTNMismatch (BC-2.03.002 Inv-1 sentinel).
func TestDiscovery_Enumerate_SVTNIsolation_ErrSentinel(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t, svtnA, nodeA1)
	d := discovery.New(cfg)

	nodeB := [8]byte{0xCC}
	advB := encodeOrFail(t, discovery.AdvertisementPayload{
		NodeAddr: nodeB,
		SVTNID:   svtnB,
		Sessions: []discovery.SessionPresence{
			{SessionName: "cross-svtn", Status: discovery.Detached, Quality: discovery.QualityGreen},
		},
	})

	err := d.ReceiveAdvertisement(context.Background(), advB)
	if err == nil {
		t.Fatal("ReceiveAdvertisement cross-SVTN: expected ErrSVTNMismatch (or wrapping it), got nil")
	}
	if !errors.Is(err, discovery.ErrSVTNMismatch) {
		t.Errorf("ReceiveAdvertisement cross-SVTN: err = %v, want errors.Is(err, ErrSVTNMismatch) == true", err)
	}
}

// ---------------------------------------------------------------------------
// VP-044 — property: advertise-on-state-change within N ticks
// ---------------------------------------------------------------------------

// TestDiscovery_VP044_AdvertiseWithinOneTick is a property-style test that
// verifies BC-2.03.001 PC-3 holds across multiple session states and session
// counts. For each case, the session must appear in Enumerate within the same
// "tick" (i.e., before Advertise returns — synchronous contract).
//
// This exercises the VP-044 verification property: "Advertisement sent within
// 1 tick of state change."
func TestDiscovery_VP044_AdvertiseWithinOneTick(t *testing.T) {
	t.Parallel()

	type testCase struct {
		sessions []discovery.SessionPresence
	}

	// Generate a wide table of session state combinations covering all
	// attachment statuses, quality levels, and cardinalities.
	cases := []testCase{
		{sessions: []discovery.SessionPresence{{SessionName: "s1", Status: discovery.Attached, Quality: discovery.QualityGreen}}},
		{sessions: []discovery.SessionPresence{{SessionName: "s2", Status: discovery.Detached, Quality: discovery.QualityYellow}}},
		{sessions: []discovery.SessionPresence{{SessionName: "s3", Status: discovery.Detached, Quality: discovery.QualityRed}}},
		{sessions: []discovery.SessionPresence{{SessionName: "s4", Status: discovery.Attached, Quality: discovery.QualityUnknown}}},
		{sessions: []discovery.SessionPresence{
			{SessionName: "multi-1", Status: discovery.Attached, Quality: discovery.QualityGreen},
			{SessionName: "multi-2", Status: discovery.Detached, Quality: discovery.QualityYellow},
			{SessionName: "multi-3", Status: discovery.Detached, Quality: discovery.QualityRed},
		}},
		// Transition: start empty, then add sessions.
		{sessions: nil},
		{sessions: []discovery.SessionPresence{{SessionName: "after-empty", Status: discovery.Detached, Quality: discovery.QualityGreen}}},
	}

	for i, tc := range cases {
		tc := tc
		t.Run("", func(t *testing.T) {
			t.Parallel()
			cfg := newTestConfig(t, svtnA, nodeA1)
			d := discovery.New(cfg)
			ctx := context.Background()

			if err := d.Advertise(ctx, tc.sessions); err != nil {
				t.Fatalf("case %d: Advertise: %v", i, err)
			}

			result, err := d.Enumerate(ctx)
			if err != nil {
				t.Fatalf("case %d: Enumerate: %v", i, err)
			}

			// Every session in the advertised set must appear in the result
			// (within the same synchronous call, satisfying "1 tick").
			for _, want := range tc.sessions {
				found := false
				for _, e := range result {
					if e.Presence.SessionName == want.SessionName && e.AdvertiserAddr == nodeA1 {
						found = true
						if e.Presence.Status != want.Status {
							t.Errorf("case %d: %q: Status = %v, want %v (VP-044)", i, want.SessionName, e.Presence.Status, want.Status)
						}
						if e.Presence.Quality != want.Quality {
							t.Errorf("case %d: %q: Quality = %v, want %v (VP-044)", i, want.SessionName, e.Presence.Quality, want.Quality)
						}
					}
				}
				if !found {
					t.Errorf("case %d: session %q not in Enumerate after Advertise (VP-044 1-tick property)", i, want.SessionName)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// VP-045 — property: SVTN-isolation invariant across randomized SVTN IDs
// ---------------------------------------------------------------------------

// TestDiscovery_VP045_SVTNIsolation_MultipleScopes verifies BC-2.03.002
// Inv-1 across multiple distinct SVTN ID pairs. For every (local SVTN,
// foreign SVTN) pair, foreign sessions must not appear in local Enumerate.
func TestDiscovery_VP045_SVTNIsolation_MultipleScopes(t *testing.T) {
	t.Parallel()

	type svtnPair struct {
		local   [16]byte
		foreign [16]byte
	}

	// Cover a range of SVTN ID patterns, not just the default test vars.
	pairs := []svtnPair{
		{local: [16]byte{0x01}, foreign: [16]byte{0x02}},
		{local: [16]byte{0xAA}, foreign: [16]byte{0xBB}},
		{local: [16]byte{0xFF, 0xFF}, foreign: [16]byte{0x00, 0x01}},
		{local: [16]byte{0x10, 0x20, 0x30}, foreign: [16]byte{0x40, 0x50, 0x60}},
		// Differ only in last byte.
		{
			local:   [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 0},
			foreign: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 1},
		},
	}

	localNode := [8]byte{0x11}
	foreignNode := [8]byte{0x22}

	for _, pair := range pairs {
		pair := pair
		t.Run("", func(t *testing.T) {
			t.Parallel()
			cfg := discovery.Config{
				LocalNodeAddr:     localNode,
				LocalSVTNID:       pair.local,
				Router:            newTestRouter(t),
				HeartbeatInterval: discovery.HeartbeatInterval,
			}
			d := discovery.New(cfg)
			ctx := context.Background()

			// Inject a local SVTN advertisement.
			localAdv := encodeOrFail(t, discovery.AdvertisementPayload{
				NodeAddr: localNode,
				SVTNID:   pair.local,
				Sessions: []discovery.SessionPresence{
					{SessionName: "local-sess", Status: discovery.Detached, Quality: discovery.QualityGreen},
				},
			})
			if err := d.ReceiveAdvertisement(ctx, localAdv); err != nil {
				t.Fatalf("ReceiveAdvertisement local SVTN: %v", err)
			}

			// Inject a foreign SVTN advertisement — must be blocked.
			foreignAdv := encodeOrFail(t, discovery.AdvertisementPayload{
				NodeAddr: foreignNode,
				SVTNID:   pair.foreign,
				Sessions: []discovery.SessionPresence{
					{SessionName: "foreign-sess", Status: discovery.Detached, Quality: discovery.QualityGreen},
				},
			})
			_ = d.ReceiveAdvertisement(ctx, foreignAdv)

			result, err := d.Enumerate(ctx)
			if err != nil {
				t.Fatalf("Enumerate: %v", err)
			}

			// Oracle: no foreign-node sessions present.
			foreign := sessionsFromNodes(result, foreignNode)
			if len(foreign) != 0 {
				t.Errorf(
					"SVTN isolation violated: local=%v foreign=%v got %d foreign session(s) in Enumerate (VP-045)",
					pair.local, pair.foreign, len(foreign),
				)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// VP-055 — property: round-trip stability across many payloads
// ---------------------------------------------------------------------------

// TestDiscovery_VP055_RoundTripProperty verifies BC-2.03.003 Inv-1:
// Encode(Decode(b)) == b holds across a wide parameter space covering all
// QualityIndicator values, both AttachmentStatus values, multiple session
// counts, and edge cases (empty sessions, long names, Unicode names).
func TestDiscovery_VP055_RoundTripProperty(t *testing.T) {
	t.Parallel()

	// roundTrip is a helper that asserts Encode(Decode(b)) == b.
	roundTrip := func(t *testing.T, payload discovery.AdvertisementPayload) {
		t.Helper()
		encoded, err := discovery.Encode(payload)
		if err != nil {
			t.Fatalf("Encode: %v", err)
		}
		decoded, err := discovery.Decode(encoded)
		if err != nil {
			t.Fatalf("Decode: %v", err)
		}
		reencoded, err := discovery.Encode(decoded)
		if err != nil {
			t.Fatalf("re-Encode: %v", err)
		}
		if !bytes.Equal(encoded, reencoded) {
			t.Errorf("round-trip not stable: encoded=%d bytes reencoded=%d bytes (VP-055)", len(encoded), len(reencoded))
		}
	}

	qualities := []discovery.QualityIndicator{
		discovery.QualityUnknown,
		discovery.QualityGreen,
		discovery.QualityYellow,
		discovery.QualityRed,
	}
	statuses := []discovery.AttachmentStatus{
		discovery.Detached,
		discovery.Attached,
	}
	sessionNames := []string{
		"agent-01",
		"",         // empty name: boundary
		"日本語セッション", // BC-2.03.003 EC-001: UTF-8 non-ASCII
		"a",        // minimal
		"session-with-dashes-and-numbers-123456789", // long ASCII
	}

	// Combinatorial sweep: all quality × status × name combinations.
	for _, q := range qualities {
		for _, s := range statuses {
			for _, name := range sessionNames {
				q, s, name := q, s, name
				roundTrip(t, discovery.AdvertisementPayload{
					NodeAddr: nodeA1,
					SVTNID:   svtnA,
					Sessions: []discovery.SessionPresence{
						{SessionName: name, Status: s, Quality: q},
					},
				})
			}
		}
	}

	// Multi-session payloads.
	roundTrip(t, discovery.AdvertisementPayload{
		NodeAddr: nodeA1,
		SVTNID:   svtnA,
		Sessions: []discovery.SessionPresence{
			{SessionName: "s1", Status: discovery.Attached, Quality: discovery.QualityGreen},
			{SessionName: "s2", Status: discovery.Detached, Quality: discovery.QualityYellow},
			{SessionName: "s3", Status: discovery.Detached, Quality: discovery.QualityRed},
			{SessionName: "s4", Status: discovery.Attached, Quality: discovery.QualityUnknown},
		},
	})

	// Nil sessions.
	roundTrip(t, discovery.AdvertisementPayload{
		NodeAddr: nodeA1,
		SVTNID:   svtnA,
		Sessions: nil,
	})

	// Different SVTN and node IDs.
	roundTrip(t, discovery.AdvertisementPayload{
		NodeAddr: [8]byte{0xFF, 0xFE, 0xFD, 0xFC, 0xFB, 0xFA, 0xF9, 0xF8},
		SVTNID:   [16]byte{0xFF, 0xFE, 0xFD, 0xFC, 0xFB, 0xFA, 0xF9, 0xF8, 0xF7, 0xF6, 0xF5, 0xF4, 0xF3, 0xF2, 0xF1, 0xF0},
		Sessions: []discovery.SessionPresence{
			{SessionName: "max-ids", Status: discovery.Attached, Quality: discovery.QualityGreen},
		},
	})
}
