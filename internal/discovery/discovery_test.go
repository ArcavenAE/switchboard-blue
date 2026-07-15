package discovery_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"net"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/discovery"
	"github.com/arcavenae/switchboard/internal/routing"
	"github.com/arcavenae/switchboard/internal/testenv"
)

// redGateGuard recovers from a not-yet-implemented stub's panic and fails the
// test cleanly (Red Gate discipline, BC-5.38.001) instead of crashing the
// whole test binary. Once the relevant Task's Green step lands, the panic
// disappears and this guard becomes a silent no-op — the assertions after
// `defer redGateGuard(t)` then run for real, with no test-file change
// required. Shared by discovery_test.go and discovery_wire_test.go (same
// package).
func redGateGuard(t *testing.T) {
	t.Helper()
	if r := recover(); r != nil {
		t.Fatalf("red gate: stub not yet implemented: %v", r)
	}
}

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
		LocalNodeAddr:            nodeAddr,
		LocalSVTNID:              svtnID,
		Router:                   newTestRouter(t),
		HeartbeatInterval:        discovery.HeartbeatInterval,
		LocalNodeAdmissionPubkey: testNodeAdmissionPubkey,
	}
}

var (
	svtnA = [16]byte{0xAA}
	svtnB = [16]byte{0xBB}

	nodeA1 = [8]byte{0x01}
	nodeA2 = [8]byte{0x02}

	// testNodeAdmissionPubkey is a fixed placeholder pubkey for tests that
	// exercise Encode/Decode/Advertise wire-format behavior and do not care
	// about cryptographic correctness against a real admitted router (that
	// coverage lives in discovery_wire_test.go's
	// TestDiscovery_EncodeThenRouterIngest_AcceptsRealAdmittedNode, which
	// uses a real generated Ed25519 pubkey admitted via
	// newAdmittedRouterForDiscoveryWire). Any non-empty value satisfies
	// Config.LocalNodeAdmissionPubkey and Encode/Decode's parameter of the
	// same name — HKDF has no notion of a "valid-shaped" key, only IKM
	// bytes (F-DWIP1-001).
	testNodeAdmissionPubkey = []byte("test-node-admission-pubkey-placeholder")
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

// TestDiscovery_Advertise_PeriodicHeartbeat is an integration sanity test for
// the real time.Ticker code path — verifies observability under real scheduler
// jitter, NOT for oracle exact-N discrimination (see
// TestDiscovery_Advertise_PeriodicHeartbeat_ExactN for that).
//
// Wide tolerance [expectedTicks/2, expectedTicks*2] is intentional:
// wall-clock jitter prevents exact counting. This test detects catastrophic
// failures only (heartbeat never fires, always fires 0 times). The
// exact-N oracle that can catch a removed ticker body is in _ExactN
// (RULING-W6TB-G).
//
// BC-2.03.001 PC-4; AC-001b (S-7.02 v1.3).
func TestDiscovery_Advertise_PeriodicHeartbeat(t *testing.T) {
	t.Parallel()

	const shortInterval = 5 * time.Millisecond
	const windowDuration = 30 * time.Millisecond
	const expectedTicks = int(windowDuration / shortInterval) // ~6

	var count int
	var mu sync.Mutex

	cfg := discovery.Config{
		LocalNodeAddr:     nodeA1,
		LocalSVTNID:       svtnA,
		Router:            newTestRouter(t),
		HeartbeatInterval: shortInterval,
		HeartbeatObserver: func() {
			mu.Lock()
			count++
			mu.Unlock()
		},
	}
	d := discovery.New(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), windowDuration)
	t.Cleanup(cancel)

	runDone := make(chan error, 1)
	go func() {
		runDone <- d.Run(ctx)
	}()

	<-ctx.Done()
	runErr := <-runDone

	// Acceptable terminal states: context cancellation or deadline exceeded.
	if runErr != nil && !errors.Is(runErr, context.DeadlineExceeded) && !errors.Is(runErr, context.Canceled) {
		t.Fatalf("Run: unexpected error after heartbeat window: %v", runErr)
	}

	mu.Lock()
	got := count
	mu.Unlock()

	// Wide tolerance [expectedTicks/2, expectedTicks*2] to survive CI scheduler
	// jitter; the rate oracle is delegated to _IsIndependent.
	lo := expectedTicks / 2
	hi := expectedTicks * 2
	if got < lo || got > hi {
		t.Errorf("HeartbeatObserver called %d times, want [%d, %d] over %v window (BC-2.03.001 PC-4 observer-fires oracle)",
			got, lo, hi, windowDuration)
	}
}

// TestDiscovery_Advertise_PeriodicHeartbeat_IsIndependent verifies that the
// heartbeat fires even when Advertise has never been called.
//
// Strong oracle: HeartbeatObserver count must be ≥ 1 after 2 full intervals.
func TestDiscovery_Advertise_PeriodicHeartbeat_IsIndependent(t *testing.T) {
	t.Parallel()

	const shortInterval = 5 * time.Millisecond
	const windowDuration = 20 * time.Millisecond

	var count int
	var mu sync.Mutex

	cfg := discovery.Config{
		LocalNodeAddr:     nodeA1,
		LocalSVTNID:       svtnA,
		Router:            newTestRouter(t),
		HeartbeatInterval: shortInterval,
		HeartbeatObserver: func() {
			mu.Lock()
			count++
			mu.Unlock()
		},
	}
	d := discovery.New(cfg)

	// Do NOT call Advertise — heartbeat must fire independently.
	ctx, cancel := context.WithTimeout(context.Background(), windowDuration)
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

	mu.Lock()
	got := count
	mu.Unlock()

	if got < 1 {
		t.Errorf("HeartbeatObserver not called after %v with no prior Advertise; want ≥ 1 (BC-2.03.001 PC-4 independent of state change)",
			windowDuration)
	}
}

// TestDiscovery_Advertise_PeriodicHeartbeat_ExactN verifies AC-001b exactly:
// N injected ticks produce exactly N HeartbeatObserver calls AND
// HeartbeatCount() == N.
//
// Uses Config.TickSource for deterministic tick delivery — no wall-clock
// sensitivity. A removed ticker body causes count == 0 and test failure
// (RULING-W6TB-G: the no-op-removal oracle).
//
// BC-2.03.001 PC-4; AC-001b (S-7.02 v1.3).
func TestDiscovery_Advertise_PeriodicHeartbeat_ExactN(t *testing.T) {
	t.Parallel()

	const N = 5
	var count int
	var mu sync.Mutex

	tickCh := make(chan time.Time, N)
	cfg := discovery.Config{
		LocalNodeAddr:     nodeA1,
		LocalSVTNID:       svtnA,
		Router:            newTestRouter(t),
		HeartbeatInterval: time.Second, // irrelevant; TickSource overrides
		HeartbeatObserver: func() {
			mu.Lock()
			count++
			mu.Unlock()
		},
		TickSource: tickCh,
	}
	d := discovery.New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	runDone := make(chan error, 1)
	go func() {
		runDone <- d.Run(ctx)
	}()

	// Send exactly N ticks and verify each one fires the observer.
	now := time.Now().UTC()
	for i := range N {
		tickCh <- now
		// Poll with a short deadline to detect a stuck observer.
		deadline := time.Now().Add(100 * time.Millisecond)
		for {
			mu.Lock()
			got := count
			mu.Unlock()
			if got == i+1 {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("tick %d: HeartbeatObserver not called within 100ms (got %d, want %d)", i+1, got, i+1)
			}
			runtime.Gosched()
		}
	}

	cancel()
	if err := <-runDone; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Run: unexpected error: %v", err)
	}

	mu.Lock()
	got := count
	mu.Unlock()
	if got != N {
		t.Errorf("HeartbeatObserver called %d times after %d ticks, want exactly %d (BC-2.03.001 PC-4 exact-N observer oracle)", got, N, N)
	}

	// HeartbeatCount() must also equal N — atomic counter is independent of
	// the optional observer (M-1).
	if hc := d.HeartbeatCount(); hc != N {
		t.Errorf("HeartbeatCount() = %d after %d ticks, want exactly %d (BC-2.03.001 PC-4 exact-N count oracle)", hc, N, N)
	}
}

// TestDiscovery_HeartbeatCount_MonotonicallyIncreases verifies that
// HeartbeatCount() increases by exactly 1 for each injected tick and never
// decrements. Companion to ExactN test (M-1; BC-2.03.001 PC-4).
func TestDiscovery_HeartbeatCount_MonotonicallyIncreases(t *testing.T) {
	t.Parallel()

	const N = 4
	tickCh := make(chan time.Time, N)
	cfg := discovery.Config{
		LocalNodeAddr:     nodeA1,
		LocalSVTNID:       svtnA,
		Router:            newTestRouter(t),
		HeartbeatInterval: time.Second,
		TickSource:        tickCh,
	}
	d := discovery.New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	runDone := make(chan error, 1)
	go func() {
		runDone <- d.Run(ctx)
	}()

	now := time.Now().UTC()
	var prev uint64
	for i := range N {
		tickCh <- now
		// Poll until HeartbeatCount advances.
		deadline := time.Now().Add(100 * time.Millisecond)
		for {
			cur := d.HeartbeatCount()
			if cur > prev {
				// Strictly increasing: each tick adds exactly 1.
				if cur != uint64(i+1) {
					t.Errorf("tick %d: HeartbeatCount() = %d, want %d (monotonic increase by 1)", i+1, cur, i+1)
				}
				prev = cur
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("tick %d: HeartbeatCount() did not advance within 100ms (stuck at %d)", i+1, prev)
			}
			runtime.Gosched()
		}
	}

	cancel()
	if err := <-runDone; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Run: unexpected error: %v", err)
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

	adv1 := buildHop2Payload(nodeA1, 1, []discovery.SessionPresence{
		{SessionName: "sess-A", Status: discovery.Detached, Quality: discovery.QualityGreen},
	})
	adv2 := buildHop2Payload(nodeA2, 1, []discovery.SessionPresence{
		{SessionName: "sess-B", Status: discovery.Attached, Quality: discovery.QualityYellow},
	})

	// AC-007: registry population now flows through the node-side
	// relay-ingest path (hop-2 DISCOVERY_RELAY payload, no per-frame HMAC —
	// trust derives from the admitted TCP connection), not the retired
	// ReceiveAdvertisement (which received hop-1 UDP directly — Ruling 2:
	// no node receives hop-1 UDP directly in the shipped topology).
	if err := d.IngestRelayAdvertisement(svtnA, adv1); err != nil {
		t.Fatalf("IngestRelayAdvertisement adv1: %v", err)
	}
	if err := d.IngestRelayAdvertisement(svtnA, adv2); err != nil {
		t.Fatalf("IngestRelayAdvertisement adv2: %v", err)
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

	// Both nodes advertise "agent-01" via the hop-2 relay-ingest path
	// (AC-007 — ReceiveAdvertisement is retired; no node receives hop-1 UDP
	// directly, Ruling 2).
	for _, node := range [][8]byte{nodeA1, nodeA2} {
		raw := buildHop2Payload(node, 1, []discovery.SessionPresence{
			{SessionName: "agent-01", Status: discovery.Detached, Quality: discovery.QualityGreen},
		})
		if err := d.IngestRelayAdvertisement(svtnA, raw); err != nil {
			t.Fatalf("IngestRelayAdvertisement node %v: %v", node, err)
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
			encoded, err := discovery.Encode(tc.payload, testNodeAdmissionPubkey)
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			decoded, err := discovery.Decode(encoded, testNodeAdmissionPubkey)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}

			if len(decoded.Sessions) != 1 {
				t.Fatalf("decoded.Sessions: got %d, want 1", len(decoded.Sessions))
			}
			got := decoded.Sessions[0]

			// F-P5L2-LOW-01: empty-name check removed — Encode now rejects empty
			// session names at encoding time (ErrInvalidSessionName); the decoded
			// value cannot be empty if Encode/Decode succeeded. The equality
			// assertion below is the correct oracle.
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

	encoded, err := discovery.Encode(payload, testNodeAdmissionPubkey)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	decoded, err := discovery.Decode(encoded, testNodeAdmissionPubkey)
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
			encoded, err := discovery.Encode(tc.payload, testNodeAdmissionPubkey)
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			decoded, err := discovery.Decode(encoded, testNodeAdmissionPubkey)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			reencoded, err := discovery.Encode(decoded, testNodeAdmissionPubkey)
			if err != nil {
				t.Fatalf("re-Encode: %v", err)
			}
			if !bytes.Equal(encoded, reencoded) {
				t.Errorf("round-trip mismatch: len=%d vs %d (BC-2.03.003 Inv-1)", len(encoded), len(reencoded))
			}

			// Field-equality oracle (F-M-002): byte equality above is
			// necessary but not sufficient — verify decoded fields match
			// the original payload to catch silent coercions.
			if decoded.NodeAddr != tc.payload.NodeAddr {
				t.Errorf("round-trip NodeAddr: got %v, want %v", decoded.NodeAddr, tc.payload.NodeAddr)
			}
			if decoded.SVTNID != tc.payload.SVTNID {
				t.Errorf("round-trip SVTNID: got %v, want %v", decoded.SVTNID, tc.payload.SVTNID)
			}
			if len(decoded.Sessions) != len(tc.payload.Sessions) {
				t.Fatalf("round-trip Sessions len: got %d, want %d", len(decoded.Sessions), len(tc.payload.Sessions))
			}
			for i, want := range tc.payload.Sessions {
				got := decoded.Sessions[i]
				if got.SessionName != want.SessionName {
					t.Errorf("round-trip Sessions[%d].SessionName: got %q, want %q", i, got.SessionName, want.SessionName)
				}
				if got.Status != want.Status {
					t.Errorf("round-trip Sessions[%d].Status: got %v, want %v", i, got.Status, want.Status)
				}
				if got.Quality != want.Quality {
					t.Errorf("round-trip Sessions[%d].Quality: got %v, want %v", i, got.Quality, want.Quality)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AC-005 — BC-2.03.001 PC-5: HMAC authentication on advertisement
// ---------------------------------------------------------------------------

// TestDiscovery_Advertise_HMACAuthenticated verifies BC-2.03.001 PC-5:
// an advertisement with a missing or wrong HMAC tag is rejected fail-closed.
//
// AC-007 rewrite: HMAC authentication now lives exclusively at the
// router-side hop-1 ingest path (RouterIngest.Ingest, AC-005/AC-006) — no
// node receives hop-1 UDP directly (Ruling 2), so the retired
// ReceiveAdvertisement's authentication property is re-verified here
// against RouterIngest.Ingest using the DiscoveryAuthKeyFor-admitted
// test-setup pattern (newAdmittedRouterForDiscoveryWire, shared from
// discovery_wire_test.go).
func TestDiscovery_Advertise_HMACAuthenticated(t *testing.T) {
	t.Parallel()

	router, pub, nodeAddr := newAdmittedRouterForDiscoveryWire(t, svtnA)
	key := testDeriveDiscoveryKey([]byte(pub), svtnA)
	ri := discovery.NewRouterIngest(discovery.RouterIngestConfig{Router: router})

	// Build a valid datagram, then corrupt a byte in the middle so that
	// length-parsing is unaffected and actual HMAC tag verification is
	// exercised rather than the short-payload path (F-M-003).
	raw := buildHop1Datagram(key, svtnA, nodeAddr, 1, []discovery.SessionPresence{
		{SessionName: "tampered-sess", Status: discovery.Detached, Quality: discovery.QualityGreen},
	})
	raw[len(raw)/2] ^= 0xFF

	decision, err := ri.Ingest(raw)
	if !errors.Is(err, discovery.ErrInvalidHMACTag) {
		t.Fatalf("Ingest: expected ErrInvalidHMACTag for tampered payload, got %v (AC-005 fail-closed)", err)
	}
	if decision.Accept || decision.Relay {
		t.Errorf("Ingest: tampered datagram must not be accepted or relayed, got Accept=%v Relay=%v (BC-2.03.001 PC-5 fail-closed)", decision.Accept, decision.Relay)
	}
}

// TestDiscovery_Advertise_HMACAuthenticated_EmptyPayload verifies that an
// empty raw byte slice is also rejected (degenerate tampered case).
//
// AC-007 rewrite: see TestDiscovery_Advertise_HMACAuthenticated's doc
// comment — this property now belongs to RouterIngest.Ingest.
func TestDiscovery_Advertise_HMACAuthenticated_EmptyPayload(t *testing.T) {
	t.Parallel()

	router, _, _ := newAdmittedRouterForDiscoveryWire(t, svtnA)
	ri := discovery.NewRouterIngest(discovery.RouterIngestConfig{Router: router})

	_, err := ri.Ingest([]byte{})
	if !errors.Is(err, discovery.ErrInvalidHMACTag) {
		t.Fatalf("Ingest with empty payload: expected ErrInvalidHMACTag, got %v", err)
	}
}

// TestDiscovery_Advertise_HMACAuthenticated_TagCorruption verifies that
// corrupting the HMAC tag bytes at the end of the encoded payload is
// rejected with ErrInvalidHMACTag. This explicitly exercises the
// tag-comparison path (not the short-payload path) per F-M-003.
//
// AC-007 rewrite: see TestDiscovery_Advertise_HMACAuthenticated's doc
// comment — this property now belongs to RouterIngest.Ingest.
func TestDiscovery_Advertise_HMACAuthenticated_TagCorruption(t *testing.T) {
	t.Parallel()

	router, pub, nodeAddr := newAdmittedRouterForDiscoveryWire(t, svtnA)
	key := testDeriveDiscoveryKey([]byte(pub), svtnA)
	ri := discovery.NewRouterIngest(discovery.RouterIngestConfig{Router: router})

	raw := buildHop1Datagram(key, svtnA, nodeAddr, 1, []discovery.SessionPresence{
		{SessionName: "tag-corrupt-sess", Status: discovery.Detached, Quality: discovery.QualityGreen},
	})
	if len(raw) < 4 {
		t.Fatalf("encoded payload too short (%d bytes) for tag corruption test", len(raw))
	}
	// Corrupt the last 4 bytes — trailing bytes of the encoded body (payload
	// region). The HMAC tag itself is at raw[0:8]; the body follows. Flipping
	// bytes at the tail alters body content that was signed by the sender, so
	// the receiver's HMAC verification over the (now-tampered) body fails to
	// match the untouched tag at the head and returns ErrInvalidHMACTag.
	for i := len(raw) - 4; i < len(raw); i++ {
		raw[i] ^= 0xFF
	}

	decision, err := ri.Ingest(raw)
	if !errors.Is(err, discovery.ErrInvalidHMACTag) {
		t.Fatalf("Ingest with tag-corrupted payload: expected ErrInvalidHMACTag, got %v (AC-005 tag-compare path)", err)
	}
	if decision.Accept || decision.Relay {
		t.Errorf("Ingest: tag-corrupted datagram must not be accepted or relayed, got Accept=%v Relay=%v (BC-2.03.001 PC-5 fail-closed)", decision.Accept, decision.Relay)
	}
}

// ---------------------------------------------------------------------------
// AC-006 — BC-2.03.002 Inv-1: SVTN cross-scope isolation
// ---------------------------------------------------------------------------

// TestDiscovery_Enumerate_SVTNIsolation verifies BC-2.03.002 Inv-1:
// sessions relayed for SVTN-B must not appear in the Enumerate result for a
// Discovery instance on SVTN-A.
//
// Oracle: len(sessionsFromNodes(svtnAResult, svtnBNode)) == 0
//
// AC-007 rewrite: the node-side relay-ingest path (IngestRelayAdvertisement)
// has no per-frame HMAC (trust derives from the admitted TCP connection,
// AC-015) — SVTN isolation at this layer is enforced by comparing the relay
// frame's own OuterHeader.SVTNID against d.cfg.LocalSVTNID
// (AC-007 postcondition 3), not by the HMAC-first-then-SVTN-check ordering
// that governed the retired ReceiveAdvertisement (that property now lives
// at the router-side hop-1 ingest path — see TestRouterIngest_* in
// discovery_wire_test.go and this file's rewritten *_ForgedSVTN test).
func TestDiscovery_Enumerate_SVTNIsolation(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t, svtnA, nodeA1)
	d := discovery.New(cfg)
	ctx := context.Background()

	// Inject a valid SVTN-A relay advertisement so the registry is non-empty.
	advA := buildHop2Payload(nodeA1, 1, []discovery.SessionPresence{
		{SessionName: "svtn-a-sess", Status: discovery.Detached, Quality: discovery.QualityGreen},
	})
	if err := d.IngestRelayAdvertisement(svtnA, advA); err != nil {
		t.Fatalf("IngestRelayAdvertisement SVTN-A: %v", err)
	}

	// A relay frame whose own OuterHeader.SVTNID names a different SVTN must
	// be rejected with ErrSVTNMismatch (AC-007 postcondition 3) —
	// defense-in-depth against a relay/routing bug delivering the wrong
	// SVTN's frame to this node. Silent drop (err==nil) is not acceptable.
	nodeB := [8]byte{0xBB}
	advB := buildHop2Payload(nodeB, 1, []discovery.SessionPresence{
		{SessionName: "svtn-b-sess", Status: discovery.Detached, Quality: discovery.QualityGreen},
	})
	svtnBErr := d.IngestRelayAdvertisement(svtnB, advB)
	if svtnBErr == nil || !errors.Is(svtnBErr, discovery.ErrSVTNMismatch) {
		t.Fatalf("IngestRelayAdvertisement SVTN-B: got %v, want ErrSVTNMismatch (AC-006/AC-007 sentinel required)", svtnBErr)
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

// TestDiscovery_Enumerate_SVTNIsolation_ForgedSVTN verifies RULING-W6TB-H:
// an attacker who forges the SVTN field (sets foreign SVTN bytes) but cannot
// produce a valid HMAC must receive ErrInvalidHMACTag — NOT a distinguishing
// SVTN-specific error.
//
// AC-007 rewrite: RULING-W6TB-H's HMAC-first-vs-SVTN-check ordering property
// now lives exclusively at the router-side hop-1 ingest path
// (RouterIngest.Ingest, AC-005/AC-006) — no node receives hop-1 UDP directly
// (Ruling 2). Router-side key derivation is a (svtnID, nodeAddr) admitted-key
// LOOKUP (DiscoveryAuthKeyFor), not the old SVTNID-only key scheme, so a
// forged/unadmitted SVTN simply produces a lookup miss — indistinguishable
// from an HMAC tag mismatch (F-P7L2-MED-01, already covered structurally by
// TestRouterIngest_LookupMissAndTagMismatch_IndistinguishableRejection in
// discovery_wire_test.go). This test re-verifies the same forged-SVTN shape
// against that entry point.
func TestDiscovery_Enumerate_SVTNIsolation_ForgedSVTN(t *testing.T) {
	t.Parallel()

	router, pub, nodeAddr := newAdmittedRouterForDiscoveryWire(t, svtnA)
	key := testDeriveDiscoveryKey([]byte(pub), svtnA)
	ri := discovery.NewRouterIngest(discovery.RouterIngestConfig{Router: router})

	// Build a datagram signed under svtnA's admitted key, then flip the
	// declared SVTNID bytes in the body to svtnB — svtnB has no admitted key
	// for this nodeAddr, so the router's key lookup misses. This simulates
	// an attacker who writes a foreign SVTN ID but does not know (and
	// cannot derive) that foreign SVTN's admitted key.
	validLocal := buildHop1Datagram(key, svtnA, nodeAddr, 1, []discovery.SessionPresence{
		{SessionName: "forged-svtn-sess", Status: discovery.Detached, Quality: discovery.QualityGreen},
	})

	const hmacTagSize = 8 // routing.AdvertisementHMACTagSize
	if len(validLocal) < hmacTagSize+16 {
		t.Fatalf("encoded advertisement too short to flip SVTN bytes")
	}
	forged := make([]byte, len(validLocal))
	copy(forged, validLocal)
	copy(forged[hmacTagSize:hmacTagSize+16], svtnB[:])

	decision, forgedErr := ri.Ingest(forged)
	if !errors.Is(forgedErr, discovery.ErrInvalidHMACTag) {
		t.Fatalf("Ingest forged-SVTN: got %v, want ErrInvalidHMACTag (RULING-W6TB-H: attacker must not get a distinguishing SVTN-specific error)", forgedErr)
	}
	// F-P7L2-MED-01: HMAC-first exclusivity — strict identity check catches
	// regressions where impl wraps a distinguishing error into the returned
	// value (leaking SVTN validity to attacker).
	if forgedErr != discovery.ErrInvalidHMACTag {
		t.Fatalf("HMAC-first exclusivity: expected err == ErrInvalidHMACTag (identity), got %v", forgedErr)
	}
	if decision.Accept || decision.Relay {
		t.Errorf("Ingest: forged-SVTN datagram must not be accepted or relayed, got Accept=%v Relay=%v", decision.Accept, decision.Relay)
	}
}

// TestDiscovery_Enumerate_SVTNIsolation_ErrSentinel verifies that when
// IngestRelayAdvertisement receives a cross-scope relay frame (its own
// OuterHeader.SVTNID differs from d.cfg.LocalSVTNID), it returns
// ErrSVTNMismatch — not nil and not ErrInvalidHMACTag (BC-2.03.002 Inv-1
// sentinel).
//
// AC-007 rewrite: see TestDiscovery_Enumerate_SVTNIsolation's doc comment —
// this property now belongs to IngestRelayAdvertisement's postcondition 3
// direct SVTNID equality check, not HMAC-first ordering (hop-2 carries no
// per-frame HMAC at all).
func TestDiscovery_Enumerate_SVTNIsolation_ErrSentinel(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t, svtnA, nodeA1)
	d := discovery.New(cfg)

	nodeB := [8]byte{0xCC}
	advB := buildHop2Payload(nodeB, 1, []discovery.SessionPresence{
		{SessionName: "cross-svtn", Status: discovery.Detached, Quality: discovery.QualityGreen},
	})

	err := d.IngestRelayAdvertisement(svtnB, advB)
	if err == nil {
		t.Fatal("IngestRelayAdvertisement cross-SVTN: expected ErrSVTNMismatch (or wrapping it), got nil")
	}
	if !errors.Is(err, discovery.ErrSVTNMismatch) {
		t.Errorf("IngestRelayAdvertisement cross-SVTN: err = %v, want errors.Is(err, ErrSVTNMismatch) == true (AC-007 postcondition 3)", err)
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
// VP-045 — RETIRED (AC-007, F-DWSP8-001)
// ---------------------------------------------------------------------------
//
// TestDiscovery_VP045_SVTNIsolation_MultipleScopes exercised
// ReceiveAdvertisement's HMAC-first-then-SVTN-check property across multiple
// randomized SVTN ID pairs. ReceiveAdvertisement is retired (AC-007
// postcondition 1 — no caller exists in the shipped topology; no node
// receives hop-1 UDP directly, Ruling 2). Per the story's explicit
// disposition this test is retired outright, not extended: the property it
// covered now splits across two entry points with different trust models
// (router-side hop-1 HMAC-first ordering — TestRouterIngest_* in
// discovery_wire_test.go and this file's rewritten *_ForgedSVTN test; and
// node-side hop-2 direct-SVTNID-equality — this file's rewritten
// SVTNIsolation/_ErrSentinel tests), neither of which is "the same function
// under randomized SVTN pairs" any more.
//
// ---------------------------------------------------------------------------
// VP-055 — property: round-trip stability across many payloads
// ---------------------------------------------------------------------------

// TestPropPresenceAdvertisement_RoundTrip verifies BC-2.03.003 Inv-1:
// Encode(Decode(b)) == b holds across a wide parameter space covering all
// QualityIndicator values, both AttachmentStatus values, multiple session
// counts, and edge cases (long names, Unicode names). Session names are
// restricted to 1..255 valid UTF-8 bytes (F-P5L2-CRIT-01; VP-055 v1.2).
//
// Traces: VP-055, BC-2.03.003 Inv-1, RULING-W6TB-J.
func TestPropPresenceAdvertisement_RoundTrip(t *testing.T) {
	t.Parallel()

	// roundTrip asserts Encode(Decode(b)) == b (byte stability) and that
	// decoded fields match the original payload (field-equality oracle,
	// F-M-002 — byte equality alone does not catch silent coercions).
	roundTrip := func(t *testing.T, payload discovery.AdvertisementPayload) {
		t.Helper()
		encoded, err := discovery.Encode(payload, testNodeAdmissionPubkey)
		if err != nil {
			t.Fatalf("Encode: %v", err)
		}
		decoded, err := discovery.Decode(encoded, testNodeAdmissionPubkey)
		if err != nil {
			t.Fatalf("Decode: %v", err)
		}
		reencoded, err := discovery.Encode(decoded, testNodeAdmissionPubkey)
		if err != nil {
			t.Fatalf("re-Encode: %v", err)
		}
		if !bytes.Equal(encoded, reencoded) {
			t.Errorf("round-trip not stable: encoded=%d bytes reencoded=%d bytes (VP-055)", len(encoded), len(reencoded))
		}

		// Field-equality checks.
		if decoded.NodeAddr != payload.NodeAddr {
			t.Errorf("round-trip NodeAddr: got %v, want %v", decoded.NodeAddr, payload.NodeAddr)
		}
		if decoded.SVTNID != payload.SVTNID {
			t.Errorf("round-trip SVTNID: got %v, want %v", decoded.SVTNID, payload.SVTNID)
		}
		if len(decoded.Sessions) != len(payload.Sessions) {
			t.Fatalf("round-trip Sessions len: got %d, want %d", len(decoded.Sessions), len(payload.Sessions))
		}
		for i, want := range payload.Sessions {
			got := decoded.Sessions[i]
			if got.SessionName != want.SessionName {
				t.Errorf("round-trip Sessions[%d].SessionName: got %q, want %q", i, got.SessionName, want.SessionName)
			}
			if got.Status != want.Status {
				t.Errorf("round-trip Sessions[%d].Status: got %v, want %v", i, got.Status, want.Status)
			}
			if got.Quality != want.Quality {
				t.Errorf("round-trip Sessions[%d].Quality: got %v, want %v", i, got.Quality, want.Quality)
			}
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
	// F-P5L2-CRIT-01: empty name removed — Encode now rejects empty session
	// names (ErrInvalidSessionName). Empty-name rejection is exercised by
	// TestPropPresenceAdvertisement_RejectsEmptyOrInvalidUTF8.
	sessionNames := []string{
		"agent-01",
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

	// ---------------------------------------------------------------------------
	// Gopter property: round-trip identity holds for arbitrary valid UTF-8
	// names in [1, 255] bytes (VP-055 v1.2; F-P5L2-CRIT-01 — empty removed).
	// ---------------------------------------------------------------------------

	properties := gopter.NewProperties(gopter.DefaultTestParameters())

	genName := gen.AnyString().SuchThat(func(s string) bool {
		b := []byte(s)
		return len(b) >= 1 && len(b) <= 255 && utf8.Valid(b)
	})
	genStatus := gen.OneConstOf(discovery.Detached, discovery.Attached)
	genQuality := gen.OneConstOf(
		discovery.QualityUnknown,
		discovery.QualityGreen,
		discovery.QualityYellow,
		discovery.QualityRed,
	)

	properties.Property(
		"round-trip: Encode(Decode(Encode(p))) == Encode(p) for valid UTF-8 names 1–255 bytes",
		prop.ForAll(
			func(name string, status discovery.AttachmentStatus, quality discovery.QualityIndicator) bool {
				payload := discovery.AdvertisementPayload{
					NodeAddr: nodeA1,
					SVTNID:   svtnA,
					Sessions: []discovery.SessionPresence{
						{SessionName: name, Status: status, Quality: quality},
					},
				}
				encoded, err := discovery.Encode(payload, testNodeAdmissionPubkey)
				if err != nil {
					return false
				}
				decoded, err := discovery.Decode(encoded, testNodeAdmissionPubkey)
				if err != nil {
					return false
				}
				reencoded, err := discovery.Encode(decoded, testNodeAdmissionPubkey)
				if err != nil {
					return false
				}
				return bytes.Equal(encoded, reencoded)
			},
			genName,
			genStatus,
			genQuality,
		),
	)

	properties.TestingRun(t)
}

// ---------------------------------------------------------------------------
// VP-055 v1.2 property tests (RULING-W6TB-J)
// ---------------------------------------------------------------------------

// TestPropPresenceAdvertisement_RejectsEmptyOrInvalidUTF8 verifies that
// Encode returns ErrInvalidSessionName for session names that are empty or
// contain invalid UTF-8 byte sequences. Oversize (>255 byte) names are
// truncated, not rejected — only empty and non-UTF-8 inputs are errors.
//
// Traces: VP-055 v1.2, BC-2.03.003 PC-2, RULING-W6TB-J, F-P5L2-HIGH-01.
func TestPropPresenceAdvertisement_RejectsEmptyOrInvalidUTF8(t *testing.T) {
	t.Parallel()

	// Manual cases: empty, and specific known-invalid UTF-8 byte sequences.
	manualCases := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"bare 0xFF 0xFE", "\xff\xfe"},
		{"mid-string invalid byte 0x80", "a\x80b"},
	}
	for _, tc := range manualCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			payload := discovery.AdvertisementPayload{
				NodeAddr: nodeA1,
				SVTNID:   svtnA,
				Sessions: []discovery.SessionPresence{
					{SessionName: tc.input, Status: discovery.Attached, Quality: discovery.QualityGreen},
				},
			}
			encoded, err := discovery.Encode(payload, testNodeAdmissionPubkey)
			if err == nil {
				t.Errorf("Encode(%q): got nil error, want ErrInvalidSessionName (VP-055 v1.2)", tc.input)
			}
			if err != nil && !errors.Is(err, discovery.ErrInvalidSessionName) {
				t.Errorf("Encode(%q): err = %v, want errors.Is(err, ErrInvalidSessionName) == true", tc.input, err)
			}
			// Output must be zero-value on error.
			if encoded != nil {
				t.Errorf("Encode(%q): returned non-nil bytes on error (want zero-value)", tc.input)
			}
		})
	}

	// Gopter-driven: inject a 0xFF byte (never valid UTF-8) into arbitrary
	// byte sequences. These must all be rejected with ErrInvalidSessionName.
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 200
	properties := gopter.NewProperties(params)

	// Generate names containing an invalid UTF-8 byte (0xFF suffix).
	genInvalidUTF8 := gen.SliceOf(gen.UInt8Range(0x00, 0xFE)).Map(func(bs []uint8) string {
		// Append 0xFF — never valid UTF-8, forcing an invalid sequence.
		return string(append(bs, 0xFF))
	})

	properties.Property(
		"encode rejects names containing invalid UTF-8 with ErrInvalidSessionName",
		prop.ForAll(
			func(name string) bool {
				payload := discovery.AdvertisementPayload{
					NodeAddr: nodeA1,
					SVTNID:   svtnA,
					Sessions: []discovery.SessionPresence{
						{SessionName: name, Status: discovery.Attached, Quality: discovery.QualityGreen},
					},
				}
				encoded, err := discovery.Encode(payload, testNodeAdmissionPubkey)
				if err == nil {
					return false // must return an error
				}
				if encoded != nil {
					return false // output must be zero-value
				}
				return errors.Is(err, discovery.ErrInvalidSessionName)
			},
			genInvalidUTF8,
		),
	)

	properties.TestingRun(t)
}

// TestPropPresenceAdvertisement_TruncatesOversize verifies that Encode
// truncates session names exceeding 255 bytes to a ≤255-byte valid UTF-8
// string ending with "…" (U+2026, 3 bytes), returns err == nil, and that
// the pre-ellipsis prefix is a byte-prefix of the input at a valid rune
// boundary (utf8.RuneStart holds at the prefix boundary).
//
// Traces: VP-055 v1.2, BC-2.03.003 PC-2 + EC-001, RULING-W6TB-J,
// F-P5L2-HIGH-02, F-P5L2-HIGH-03.
func TestPropPresenceAdvertisement_TruncatesOversize(t *testing.T) {
	t.Parallel()

	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 200
	properties := gopter.NewProperties(params)

	// Generate valid UTF-8 strings whose byte length is in [256, 2048].
	// Strategy: start with a valid UTF-8 string and pad with ASCII 'x' until
	// it exceeds 255 bytes, then cap at 2048.
	genOversizeName := gen.AnyString().Map(func(s string) string {
		for !utf8.ValidString(s) || len([]byte(s)) == 0 {
			s += "x"
		}
		for len([]byte(s)) <= 255 {
			s += "x"
		}
		b := []byte(s)
		if len(b) > 2048 {
			// Truncate to 2048 at a rune boundary.
			cut := 2048
			for cut > 0 && !utf8.RuneStart(b[cut]) {
				cut--
			}
			s = string(b[:cut])
		}
		return s
	}).SuchThat(func(s string) bool {
		b := []byte(s)
		return utf8.ValidString(s) && len(b) >= 256 && len(b) <= 2048
	})

	ellipsis := "…" // U+2026, 3 UTF-8 bytes

	properties.Property(
		"encode truncates oversize valid UTF-8 names: err==nil, ≤255 bytes, ends with '…', prefix at rune boundary",
		prop.ForAll(
			func(name string) bool {
				payload := discovery.AdvertisementPayload{
					NodeAddr: nodeA1,
					SVTNID:   svtnA,
					Sessions: []discovery.SessionPresence{
						{SessionName: name, Status: discovery.Attached, Quality: discovery.QualityGreen},
					},
				}
				encoded, err := discovery.Encode(payload, testNodeAdmissionPubkey)
				if err != nil {
					return false // truncation must not error
				}
				decoded, decErr := discovery.Decode(encoded, testNodeAdmissionPubkey)
				if decErr != nil {
					return false
				}
				result := decoded.Sessions[0].SessionName

				// Must be within the 255-byte cap.
				if len([]byte(result)) > 255 {
					return false
				}
				// Must be valid UTF-8.
				if !utf8.ValidString(result) {
					return false
				}
				// Must end with the ellipsis marker.
				if !strings.HasSuffix(result, ellipsis) {
					return false
				}
				// Pre-ellipsis prefix must be a byte-prefix of the original
				// input and must end at a valid rune boundary.
				prefix := result[:len(result)-len(ellipsis)]
				if !utf8.ValidString(prefix) {
					return false
				}
				if !strings.HasPrefix(name, prefix) {
					return false
				}
				// Boundary check: the byte immediately after the prefix must
				// start a new rune (or equal len(input)), confirming no
				// mid-rune cut (F-P5L2-HIGH-03).
				prefixLen := len([]byte(prefix))
				inputBytes := []byte(name)
				if prefixLen < len(inputBytes) && !utf8.RuneStart(inputBytes[prefixLen]) {
					return false
				}
				// F-P7L2-MED-02: maximality — the impl cuts at byte 252 then
				// walks back at most 3 bytes to a rune boundary (max UTF-8
				// rune width is 4 bytes; walkback consumes at most 3
				// continuation bytes). So the prefix must be at least 249
				// bytes long for any oversize input; shorter means the impl
				// truncated more aggressively than the rune-boundary walkback
				// justifies and is silently losing content.
				ellipsisBytes := []byte(ellipsis)
				resultBytes := []byte(result)
				prefixBytes := resultBytes[:len(resultBytes)-len(ellipsisBytes)]
				if len(prefixBytes) < 249 {
					return false
				}
				// Round-trip stability: re-encoding the decoded payload must
				// produce the same bytes (truncated form is idempotent).
				reencoded, reErr := discovery.Encode(decoded, testNodeAdmissionPubkey)
				if reErr != nil {
					return false
				}
				return bytes.Equal(encoded, reencoded)
			},
			genOversizeName,
		),
	)

	properties.TestingRun(t)
}

// ---------------------------------------------------------------------------
// AC-004b boundary test (F-P5L2-HIGH-03 content-preservation oracle)
// ---------------------------------------------------------------------------

// TestDiscovery_Encode_SessionName255ByteCap verifies exact boundary behaviour
// for the 255-byte session-name cap (BC-2.03.003 PC-2 + EC-001; M-2):
//
//   - 255-byte ASCII name: accepted verbatim, Encode err == nil, round-trip
//     byte-exact (F-P5L2-HIGH-03 at-boundary case).
//   - 256-byte ASCII name: truncated to exactly "b"×252 + "…" = 255 bytes
//     (content-preservation oracle, F-P5L2-HIGH-03).
//   - 512-byte "日"×170 name: truncated to "日"×84 + "…" = 84×3+3 = 255 bytes
//     (multi-byte rune boundary, F-P5L2-HIGH-03).
//   - Mid-rune boundary: 250×"a" + 10×"日" (280 bytes) — pre-ellipsis prefix
//     ends at a valid rune boundary, result is valid UTF-8 (F-L2-001).
//
// Traces: VP-055 v1.2, BC-2.03.003 PC-2 + EC-001, AC-004b, RULING-W6TB-J,
// F-P5L2-HIGH-01, F-P5L2-HIGH-02, F-P5L2-HIGH-03.
func TestDiscovery_Encode_SessionName255ByteCap(t *testing.T) {
	t.Parallel()

	encodeDecodeSession := func(t *testing.T, name string) (string, error) {
		t.Helper()
		payload := discovery.AdvertisementPayload{
			NodeAddr: nodeA1,
			SVTNID:   svtnA,
			Sessions: []discovery.SessionPresence{
				{SessionName: name, Status: discovery.Attached, Quality: discovery.QualityGreen},
			},
		}
		encoded, err := discovery.Encode(payload, testNodeAdmissionPubkey)
		if err != nil {
			return "", err
		}
		decoded, err := discovery.Decode(encoded, testNodeAdmissionPubkey)
		if err != nil {
			return "", err
		}
		if len(decoded.Sessions) != 1 {
			t.Fatalf("decoded session count = %d, want 1", len(decoded.Sessions))
		}
		// Verify round-trip stability of the encoded form.
		reencoded, err := discovery.Encode(decoded, testNodeAdmissionPubkey)
		if err != nil {
			return "", err
		}
		if !bytes.Equal(encoded, reencoded) {
			t.Errorf("round-trip not stable: encoded=%d bytes reencoded=%d bytes (VP-055)", len(encoded), len(reencoded))
		}
		return decoded.Sessions[0].SessionName, nil
	}

	t.Run("255-byte ASCII at-boundary accept verbatim", func(t *testing.T) {
		t.Parallel()
		name255 := strings.Repeat("a", 255)
		got, err := encodeDecodeSession(t, name255)
		if err != nil {
			t.Fatalf("Encode: %v (255-byte name must not be rejected)", err)
		}
		if got != name255 {
			t.Errorf("255-byte name: got len=%d %q, want verbatim %d bytes (M-2 at-boundary)", len(got), got[:min(len(got), 20)], len(name255))
		}
	})

	t.Run("256-byte ASCII content-preservation oracle", func(t *testing.T) {
		t.Parallel()
		// F-P5L2-HIGH-03: the pre-ellipsis content must be the byte-exact
		// prefix of the input at the cut point (252 bytes for ASCII).
		want := strings.Repeat("b", 252) + "…"
		got, err := encodeDecodeSession(t, strings.Repeat("b", 256))
		if err != nil {
			t.Fatalf("Encode: %v (256-byte name must truncate, not error)", err)
		}
		if got != want {
			t.Errorf("256-byte name: got %q (len=%d), want %q (len=%d) (F-P5L2-HIGH-03 content-preservation oracle)", got, len(got), want, len(want))
		}
	})

	t.Run("512-byte UTF-8 日×170 content-preservation oracle", func(t *testing.T) {
		t.Parallel()
		// "日" = 3 UTF-8 bytes; 84×3 = 252 content bytes + 3-byte ellipsis = 255.
		// F-P5L2-HIGH-03: assert exact truncated content.
		want := strings.Repeat("日", 84) + "…"
		got, err := encodeDecodeSession(t, strings.Repeat("日", 170))
		if err != nil {
			t.Fatalf("Encode: %v (日×170 name must truncate, not error)", err)
		}
		if got != want {
			t.Errorf("日×170 name: got %q (len=%d), want %q (len=%d) (F-P5L2-HIGH-03 content-preservation oracle)", got, len(got), want, len(want))
		}
	})

	t.Run("mid-rune boundary: 250×a + 10×日 (280 bytes)", func(t *testing.T) {
		t.Parallel()
		// "日" = 0xE6 0x97 0xA5 (3 bytes). The 252-byte content window ends at
		// byte 252, which lands at byte offset 2 of the second "日" rune
		// (250 ASCII bytes + 2 bytes of the first "日"). Correct truncation
		// must walk back to byte 250 (last ASCII byte boundary), so the
		// result is 250×"a" + "…" = 253 bytes total (F-L2-001).
		mixedName := strings.Repeat("a", 250) + strings.Repeat("日", 10)
		if len(mixedName) != 280 {
			t.Fatalf("test input len=%d, want 280", len(mixedName))
		}
		got, err := encodeDecodeSession(t, mixedName)
		if err != nil {
			t.Fatalf("Encode: %v (mid-rune name must truncate, not error)", err)
		}
		// F-P7L2-MED-03: exact-content oracle — the comment documents the
		// expected output (250×"a"+"…" = 253 bytes) but prior code never
		// asserted it, weakening F-P5L2-HIGH-03 exactly where the walkback
		// branch matters most. Assert exact content here, matching the
		// ASCII sibling subtests' pattern.
		wantMidRune := strings.Repeat("a", 250) + "…"
		if got != wantMidRune {
			t.Errorf("mid-rune boundary: got %q (len=%d), want %q (len=%d) — walkback must land at byte 250 (last rune start before cut=252, closing F-P5L2-HIGH-03)", got, len(got), wantMidRune, len(wantMidRune))
		}
		if len(got) > 255 {
			t.Errorf("mid-rune: decoded name len=%d, want ≤255 bytes (F-L2-001)", len(got))
		}
		if !strings.HasSuffix(got, "…") {
			t.Errorf("mid-rune: truncated name %q does not end with '…' (F-L2-001 truncation contract)", got)
		}
		if !utf8.ValidString(got) {
			t.Errorf("mid-rune: truncated result is not valid UTF-8 — truncation landed mid-rune (F-L2-001)")
		}
		// The pre-ellipsis prefix must be a byte-prefix of the input at a
		// valid rune boundary (utf8.RuneStart check).
		ellipsis := "…"
		prefix := got[:len(got)-len(ellipsis)]
		if !strings.HasPrefix(mixedName, prefix) {
			t.Errorf("mid-rune: pre-ellipsis prefix %q is not a byte-prefix of input (F-P5L2-HIGH-03)", prefix)
		}
		prefixLen := len([]byte(prefix))
		inputBytes := []byte(mixedName)
		if prefixLen < len(inputBytes) && !utf8.RuneStart(inputBytes[prefixLen]) {
			t.Errorf("mid-rune: prefix boundary byte %#x at offset %d is not a rune start (F-P5L2-HIGH-03)", inputBytes[prefixLen], prefixLen)
		}
	})
}

// ---------------------------------------------------------------------------
// Post-merge deferrals — issues #49 (Advertise validation), #50 (nameLen==0
// decode rejection), #51 (uint16 session-count guard).
// ---------------------------------------------------------------------------

// TestDiscovery_Advertise_RejectsInvalidSessionName is the regression test
// for issue #49: Advertise must validate each session name against the same
// rules Encode uses (BC-2.03.003 PC-2), returning ErrInvalidSessionName and
// leaving the local registry untouched. This guards against a future refactor
// that would let invalid entries leak into the wire path once Advertise is
// wired to real transmit.
func TestDiscovery_Advertise_RejectsInvalidSessionName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
	}{
		{"empty session name", ""},
		{"invalid UTF-8 (0xFF suffix)", "agent\xff"},
		{"mid-string invalid byte 0x80", "a\x80b"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := newTestConfig(t, svtnA, nodeA1)
			d := discovery.New(cfg)
			ctx := context.Background()

			err := d.Advertise(ctx, []discovery.SessionPresence{
				{SessionName: tc.input, Status: discovery.Attached, Quality: discovery.QualityGreen},
			})
			if !errors.Is(err, discovery.ErrInvalidSessionName) {
				t.Fatalf("Advertise(%q): got %v, want ErrInvalidSessionName (issue #49; BC-2.03.003 PC-2)", tc.input, err)
			}

			// Strong oracle: the invalid entry must NOT appear in the local
			// registry. Advertise validates up-front and never stores partial
			// state, so Enumerate stays empty.
			entries, enumErr := d.Enumerate(ctx)
			if enumErr != nil {
				t.Fatalf("Enumerate after rejected Advertise: %v", enumErr)
			}
			if len(entries) != 0 {
				t.Errorf("Advertise(%q): registry contains %d entries after rejection, want 0 (fail-closed)", tc.input, len(entries))
			}
		})
	}
}

// TestDiscovery_Advertise_RejectsInvalidSessionName_DoesNotPurgePriorEntries
// verifies that a rejected Advertise call leaves any previously-stored valid
// entries intact (the pre-purge in Advertise runs only after validation).
func TestDiscovery_Advertise_RejectsInvalidSessionName_DoesNotPurgePriorEntries(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t, svtnA, nodeA1)
	d := discovery.New(cfg)
	ctx := context.Background()

	if err := d.Advertise(ctx, []discovery.SessionPresence{
		{SessionName: "valid", Status: discovery.Attached, Quality: discovery.QualityGreen},
	}); err != nil {
		t.Fatalf("initial Advertise: %v", err)
	}

	// A second Advertise with an invalid entry must return the sentinel and
	// leave the earlier valid entry in place.
	err := d.Advertise(ctx, []discovery.SessionPresence{
		{SessionName: "", Status: discovery.Attached, Quality: discovery.QualityGreen},
	})
	if !errors.Is(err, discovery.ErrInvalidSessionName) {
		t.Fatalf("second Advertise (empty name): got %v, want ErrInvalidSessionName", err)
	}

	entries, err := d.Enumerate(ctx)
	if err != nil {
		t.Fatalf("Enumerate: %v", err)
	}
	found := false
	for _, e := range entries {
		if e.Presence.SessionName == "valid" && e.AdvertiserAddr == nodeA1 {
			found = true
			break
		}
	}
	if !found {
		t.Error("prior valid entry 'valid' missing after a rejected Advertise (must not purge on validation failure)")
	}
}

// buildAdvertisementWithZeroLengthName constructs an authenticated wire frame
// carrying a single session whose name-length field is zero. The frame is
// signed with the correct SVTN key so HMAC verification passes and the
// zero-length check in decodeBody is exercised (rather than short-circuited
// by the HMAC path). Used by TestDiscovery_Decode_RejectsZeroLengthName's
// Decode() half.
func buildAdvertisementWithZeroLengthName(t *testing.T, svtnID [16]byte, nodeAddr [8]byte) []byte {
	t.Helper()

	// Body layout: [16]SVTNID | [8]NodeAddr | [8]Sequence | uint16 count |
	// per-session: uint16 name_len | name_bytes | uint8 status | uint8
	// quality (SEC-DW-07 widened Sequence field).
	body := make([]byte, 0, 16+8+8+2+2+0+1+1)
	body = append(body, svtnID[:]...)
	body = append(body, nodeAddr[:]...)
	body = binary.BigEndian.AppendUint64(body, 1) // Sequence = 1 (value irrelevant to this test)
	body = binary.BigEndian.AppendUint16(body, 1) // count = 1
	body = binary.BigEndian.AppendUint16(body, 0) // name_len = 0 (asymmetric)
	body = append(body, byte(discovery.Detached), byte(discovery.QualityGreen))

	// The HMAC key derivation mirrors discovery.advertisementKey (SVTN ID
	// used verbatim as the key material). Signing with the same key the
	// receiver will derive means HMAC verifies and the decoder proceeds to
	// the nameLen==0 branch. Irrelevant in practice: decodeBody rejects
	// nameLen==0 before Decode ever reaches HMAC verification, so this key
	// choice does not affect the test's oracle either way.
	tag := routing.ComputeAdvertisementHMAC(svtnID[:], body)

	raw := make([]byte, 0, len(tag)+len(body))
	raw = append(raw, tag[:]...)
	raw = append(raw, body...)
	return raw
}

// TestDiscovery_Decode_RejectsZeroLengthName is the regression test for
// issue #50: decodeBody must reject frames carrying nameLen==0. Encode never
// produces such frames (encodedSessionName rejects empty input), so any peer
// emitting one is malformed or adversarial. Rejecting closes the round-trip
// asymmetry (decoded struct with SessionName="" cannot be re-Encoded).
//
// AC-007 rewrite: the original second half of this test exercised
// ReceiveAdvertisement (retired). RouterIngest.Ingest is the receiver that
// AC-007 pointed callers at — hop-1 is router-terminated only (Ruling 2) —
// and its zero-length-name rejection runs through a materially different
// pipeline (DecodeSessionList, invoked only AFTER HMAC verification
// succeeds, not decodeBody's pre-HMAC parse), so it is re-verified here
// against a genuinely admitted key rather than assumed to behave
// identically to the retired function.
func TestDiscovery_Decode_RejectsZeroLengthName(t *testing.T) {
	t.Parallel()

	raw := buildAdvertisementWithZeroLengthName(t, svtnA, nodeA1)

	// Public Decode surface: must return an error. Decode wraps decodeBody
	// failures as ErrInvalidHMACTag (parser detail is not leaked to
	// unauthenticated peers). The strong oracle is that the frame is not
	// accepted; the sentinel identity confirms the wrapping contract.
	if _, err := discovery.Decode(raw, testNodeAdmissionPubkey); !errors.Is(err, discovery.ErrInvalidHMACTag) {
		t.Fatalf("Decode(zero-length name frame): got %v, want ErrInvalidHMACTag (issue #50; pre-HMAC decode failures are wrapped)", err)
	}

	// RouterIngest.Ingest path — sessions are decoded via DecodeSessionList
	// only after HMAC verification succeeds, so this half signs with a
	// genuinely admitted key and verifies the post-HMAC zero-length-name
	// rejection fires.
	router, pub, admittedNodeAddr := newAdmittedRouterForDiscoveryWire(t, svtnA)
	key := testDeriveDiscoveryKey([]byte(pub), svtnA)
	ri := discovery.NewRouterIngest(discovery.RouterIngestConfig{Router: router})

	admittedRaw := buildHop1Datagram(key, svtnA, admittedNodeAddr, 1, []discovery.SessionPresence{
		{SessionName: "", Status: discovery.Detached, Quality: discovery.QualityGreen},
	})
	decision, err := ri.Ingest(admittedRaw)
	if !errors.Is(err, discovery.ErrInvalidHMACTag) {
		t.Fatalf("Ingest(zero-length name frame): got %v, want ErrInvalidHMACTag (post-HMAC decode failures are wrapped)", err)
	}
	if decision.Accept || decision.Relay {
		t.Errorf("Ingest: zero-length-name frame must not be accepted or relayed, got Accept=%v Relay=%v", decision.Accept, decision.Relay)
	}
}

// TestDiscovery_Encode_RejectsMoreThan65535Sessions is the regression test
// for issue #51: encodeBody must reject payloads whose session count would
// overflow the uint16 wire count field, rather than silently truncating.
// The failure mode without a guard is a wire frame whose count undercounts
// the sessions actually appended, producing a decode error on the receiver.
func TestDiscovery_Encode_RejectsMoreThan65535Sessions(t *testing.T) {
	t.Parallel()

	// 65536 = uint16 overflow boundary. Names are single-byte to keep the
	// slice-allocation cost bounded; the guard fires before any encoding
	// work, so this test completes in well under a second.
	const overflowCount = 65536
	sessions := make([]discovery.SessionPresence, overflowCount)
	for i := range sessions {
		sessions[i] = discovery.SessionPresence{
			SessionName: "s",
			Status:      discovery.Detached,
			Quality:     discovery.QualityGreen,
		}
	}

	encoded, err := discovery.Encode(discovery.AdvertisementPayload{
		NodeAddr: nodeA1,
		SVTNID:   svtnA,
		Sessions: sessions,
	}, testNodeAdmissionPubkey)
	if !errors.Is(err, discovery.ErrTooManySessions) {
		t.Fatalf("Encode(%d sessions): got %v, want ErrTooManySessions (issue #51)", overflowCount, err)
	}
	if encoded != nil {
		t.Errorf("Encode(%d sessions): returned non-nil bytes on error (want zero-value)", overflowCount)
	}
}

// TestDiscovery_Encode_Accepts65535Sessions verifies the boundary case: the
// maximum encodable session count (uint16 max) is accepted, distinguishing
// the overflow guard from an off-by-one that would reject 65535 too. We
// inspect the wire count field directly rather than round-tripping through
// Decode, which enforces a stricter runtime cap for pre-HMAC parse-work
// bounding (see decodeBody's maxSessionsPerAdvertisement).
func TestDiscovery_Encode_Accepts65535Sessions(t *testing.T) {
	t.Parallel()

	const maxCount = 65535
	sessions := make([]discovery.SessionPresence, maxCount)
	for i := range sessions {
		sessions[i] = discovery.SessionPresence{
			SessionName: "s",
			Status:      discovery.Detached,
			Quality:     discovery.QualityGreen,
		}
	}

	encoded, err := discovery.Encode(discovery.AdvertisementPayload{
		NodeAddr: nodeA1,
		SVTNID:   svtnA,
		Sessions: sessions,
	}, testNodeAdmissionPubkey)
	if err != nil {
		t.Fatalf("Encode(%d sessions): got %v, want nil (65535 is the max, not overflow)", maxCount, err)
	}
	// Wire layout: 8-byte HMAC tag | 16-byte SVTNID | 8-byte NodeAddr |
	// 8-byte Sequence | uint16 count. The count lives at offset 40
	// (SEC-DW-07 widened Sequence uint32→uint64, F-DWSP4-001).
	const countOffset = routing.AdvertisementHMACTagSize + 16 + 8 + 8
	if len(encoded) < countOffset+2 {
		t.Fatalf("Encode returned %d bytes, too short to inspect count field", len(encoded))
	}
	gotCount := binary.BigEndian.Uint16(encoded[countOffset : countOffset+2])
	if gotCount != maxCount {
		t.Errorf("wire count field = %d, want %d (guard must not fire at exactly the uint16 max)", gotCount, maxCount)
	}
}

// ---------------------------------------------------------------------------
// S-BL.DISCOVERY-WIRE AC-002 — MulticastAddrFor determinism
// ---------------------------------------------------------------------------

// TestMulticastAddrFor_Deterministic_SHA256Derived verifies AC-002: the
// SVTN-scoped multicast address is 239.h0.h1.h2, where h0..h2 are the first
// three bytes of SHA-256(svtnID) — deterministic, computable independently,
// static (no allocation/release step), and stable across repeated calls
// (Decision 2(b)).
func TestMulticastAddrFor_Deterministic_SHA256Derived(t *testing.T) {
	t.Parallel()
	defer redGateGuard(t)

	svtnID := [16]byte{0xAA, 0xBB, 0xCC, 0xDD, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
	h := sha256.Sum256(svtnID[:])
	want := net.IPv4(239, h[0], h[1], h[2])

	got1 := discovery.MulticastAddrFor(svtnID)
	if !got1.Equal(want) {
		t.Errorf("MulticastAddrFor(%x) = %v, want %v (239.h0.h1.h2 from SHA-256(svtnID))", svtnID, got1, want)
	}

	// Postcondition 3: the same SVTN ID always produces the same address
	// across repeated calls.
	got2 := discovery.MulticastAddrFor(svtnID)
	if !got1.Equal(got2) {
		t.Errorf("MulticastAddrFor: non-deterministic — first call %v, second call %v", got1, got2)
	}

	// Distinct SVTN IDs should (in the overwhelmingly common case) derive
	// distinct addresses — a constant-return implementation would still
	// pass the two assertions above, so this guards against that class of
	// trivial-but-wrong implementation, same rationale as
	// TestDeriveKey_DistinctSVTNsProduceDistinctKeys in internal/hmac.
	otherSVTN := [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
	gotOther := discovery.MulticastAddrFor(otherSVTN)
	if gotOther.Equal(got1) {
		t.Errorf("MulticastAddrFor: distinct SVTN IDs %x and %x produced the same address %v (constant-return regression)", svtnID, otherSVTN, gotOther)
	}
}

// ---------------------------------------------------------------------------
// S-BL.DISCOVERY-WIRE AC-003 — sender-side dispatch: real UDP, TTL=1, no join
// ---------------------------------------------------------------------------

// TestDiscovery_Advertise_WriteToMulticast_TTL1_NoGroupJoin verifies AC-003
// postcondition 1: the access-node dispatch path performs a plain UDP send
// to the SVTN-derived multicast address — no net.ListenMulticastUDP, no
// group join on the sender side (Decision 2(a); SEC-DW-08). A real loopback
// multicast listener is the test's receive-side oracle: if Advertise still
// only mutates the in-process registry (S-7.02 behavior) rather than
// sending real UDP, the listener goroutine below times out with nothing
// received.
//
// Postcondition 2 (multicast TTL explicitly set to 1) is NOT independently
// wire-verified here: reading a received datagram's IP TTL from the
// receive side requires control-message support (golang.org/x/net/ipv4's
// PacketConn, or raw syscall-level CMSG parsing) that this story's Library &
// Framework Requirements section commits to NOT introducing (zero new
// third-party dependencies). How the sender sets TTL=1 (syscall-level
// setsockopt vs. some other stdlib-only mechanism) is itself Task 3
// Green-step design, not fixed by this story's rulings. Deferred to code
// review at Green rather than fabricating a weak assertion.
func TestDiscovery_Advertise_WriteToMulticast_TTL1_NoGroupJoin(t *testing.T) {
	// NOT t.Parallel(): binds a real loopback multicast UDP socket.
	defer redGateGuard(t)

	iface := testenv.MulticastLoopbackInterface(t)
	svtnID := [16]byte{0xDE, 0xAD, 0xBE, 0xEF}
	groupAddr := discovery.MulticastAddrFor(svtnID)

	listenAddr := &net.UDPAddr{IP: groupAddr, Port: discovery.DiscoveryPort}
	conn, err := net.ListenMulticastUDP("udp4", iface, listenAddr)
	if err != nil {
		t.Fatalf("ListenMulticastUDP (test-side receive oracle): %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	cfg := discovery.Config{
		LocalNodeAddr:            [8]byte{0x01},
		LocalSVTNID:              svtnID,
		Router:                   newTestRouter(t),
		LocalNodeAdmissionPubkey: testNodeAdmissionPubkey,
	}
	d := discovery.New(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	recvCh := make(chan int, 1)
	go func() {
		buf := make([]byte, 2048)
		_ = conn.SetReadDeadline(time.Now().Add(4 * time.Second))
		n, _, readErr := conn.ReadFromUDP(buf)
		if readErr != nil {
			recvCh <- 0
			return
		}
		recvCh <- n
	}()

	sessions := []discovery.SessionPresence{
		{SessionName: "agent-01", Status: discovery.Attached, Quality: discovery.QualityGreen},
	}
	if err := d.Advertise(ctx, sessions); err != nil {
		t.Fatalf("Advertise: unexpected error: %v", err)
	}

	select {
	case n := <-recvCh:
		if n == 0 {
			t.Fatal("AC-003 postcondition 1: no UDP datagram received on the SVTN-derived multicast group — Advertise did not send real UDP")
		}
	case <-time.After(4500 * time.Millisecond):
		t.Fatal("AC-003 postcondition 1: timed out waiting for a multicast datagram from Advertise")
	}
}
