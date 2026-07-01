package discovery_test

import (
	"bytes"
	"context"
	"errors"
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
	// Corrupt a byte in the middle of the payload (not byte 0) so that
	// length-parsing is unaffected and actual HMAC tag verification is
	// exercised rather than the short-payload path (F-M-003).
	raw[len(raw)/2] ^= 0xFF

	err := d.ReceiveAdvertisement(ctx, raw)
	if !errors.Is(err, discovery.ErrInvalidHMACTag) {
		t.Fatalf("ReceiveAdvertisement: expected ErrInvalidHMACTag for tampered payload, got %v (AC-005 fail-closed)", err)
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
	if !errors.Is(err, discovery.ErrInvalidHMACTag) {
		t.Fatalf("ReceiveAdvertisement with empty payload: expected ErrInvalidHMACTag, got %v", err)
	}
}

// TestDiscovery_Advertise_HMACAuthenticated_TagCorruption verifies that
// corrupting the HMAC tag bytes at the end of the encoded payload is
// rejected with ErrInvalidHMACTag. This explicitly exercises the
// tag-comparison path (not the short-payload path) per F-M-003.
func TestDiscovery_Advertise_HMACAuthenticated_TagCorruption(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t, svtnA, nodeA1)
	d := discovery.New(cfg)
	ctx := context.Background()

	validPayload := discovery.AdvertisementPayload{
		NodeAddr: nodeA2,
		SVTNID:   svtnA,
		Sessions: []discovery.SessionPresence{
			{SessionName: "tag-corrupt-sess", Status: discovery.Detached, Quality: discovery.QualityGreen},
		},
	}
	raw := encodeOrFail(t, validPayload)
	if len(raw) < 4 {
		t.Fatalf("encoded payload too short (%d bytes) for tag corruption test", len(raw))
	}
	// Corrupt the last 4 bytes — the trailing end of the HMAC tag region.
	for i := len(raw) - 4; i < len(raw); i++ {
		raw[i] ^= 0xFF
	}

	err := d.ReceiveAdvertisement(ctx, raw)
	if !errors.Is(err, discovery.ErrInvalidHMACTag) {
		t.Fatalf("ReceiveAdvertisement with tag-corrupted payload: expected ErrInvalidHMACTag, got %v (AC-005 tag-compare path)", err)
	}

	// Strong oracle: session must not appear in Enumerate.
	entries, enumErr := d.Enumerate(ctx)
	if enumErr != nil {
		t.Fatalf("Enumerate after tag-corrupt rejection: %v", enumErr)
	}
	for _, e := range entries {
		if e.Presence.SessionName == "tag-corrupt-sess" {
			t.Error("tag-corrupted advertisement must not update the session list (BC-2.03.001 PC-5 fail-closed)")
		}
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
//
// RULING-W6TB-H: With HMAC-first ordering, a legitimate foreign-SVTN
// advertisement (signed with the foreign SVTN's key) passes HMAC verification
// but is rejected by the SVTN cross-scope check — returning ErrSVTNMismatch.
// This is the expected sentinel for admitted nodes on other SVTNs.
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

	// Inject a legitimate foreign-SVTN advertisement: node on SVTN-B encoded
	// with SVTN-B's HMAC key (encodeOrFail uses payload.SVTNID as the key).
	// RULING-W6TB-H: HMAC passes (foreign key matches foreign SVTN), then the
	// SVTN cross-scope check fires → ErrSVTNMismatch.
	nodeB := [8]byte{0xBB}
	advB := encodeOrFail(t, discovery.AdvertisementPayload{
		NodeAddr: nodeB,
		SVTNID:   svtnB,
		Sessions: []discovery.SessionPresence{
			{SessionName: "svtn-b-sess", Status: discovery.Detached, Quality: discovery.QualityGreen},
		},
	})
	// ReceiveAdvertisement must return ErrSVTNMismatch for a cross-scope
	// advertisement. AC-006: silent drop is not acceptable — the sentinel
	// contract must be enforced so callers can observe the rejection.
	svtnBErr := d.ReceiveAdvertisement(ctx, advB)
	if svtnBErr == nil || !errors.Is(svtnBErr, discovery.ErrSVTNMismatch) {
		t.Fatalf("ReceiveAdvertisement SVTN-B (legitimate foreign): got %v, want ErrSVTNMismatch (AC-006 sentinel required, RULING-W6TB-H)", svtnBErr)
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
// produce a valid HMAC must receive ErrInvalidHMACTag — NOT ErrSVTNMismatch.
//
// With HMAC-first ordering, the SVTN field is authenticated before the
// cross-scope check, so the attacker's distinguishing oracle is closed.
// The forged advertisement here uses a raw payload with foreign SVTN bytes
// whose HMAC was computed under the local SVTN key (i.e., the "wrong key"
// from the forger's perspective), causing HMAC verification to fail.
func TestDiscovery_Enumerate_SVTNIsolation_ForgedSVTN(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t, svtnA, nodeA1)
	d := discovery.New(cfg)
	ctx := context.Background()

	// Build an advertisement carrying svtnB bytes but signed with svtnA's
	// key — this simulates an attacker who writes a foreign SVTN ID but
	// does not know the foreign SVTN's key (so they sign with the local
	// key or a random key instead).
	//
	// We construct this by encoding with svtnA (so the HMAC is valid for
	// svtnA's key) and then flipping the SVTN ID bytes in the body to svtnB.
	// The HMAC now covers the wrong SVTN bytes, so verification fails.
	validLocalAdv, err := discovery.Encode(discovery.AdvertisementPayload{
		NodeAddr: nodeA2,
		SVTNID:   svtnA, // encode with local SVTN → correct HMAC for svtnA key
		Sessions: []discovery.SessionPresence{
			{SessionName: "forged-svtn-sess", Status: discovery.Detached, Quality: discovery.QualityGreen},
		},
	})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	// Flip the SVTN ID bytes in the body (bytes [8:24] after the 8-byte HMAC
	// tag prefix) to svtnB — the HMAC tag now covers body bytes that claim
	// svtnB but were signed as svtnA, so the HMAC is invalid.
	const hmacTagSize = 8 // routing.AdvertisementHMACTagSize
	if len(validLocalAdv) < hmacTagSize+16 {
		t.Fatalf("encoded advertisement too short to flip SVTN bytes")
	}
	forged := make([]byte, len(validLocalAdv))
	copy(forged, validLocalAdv)
	copy(forged[hmacTagSize:hmacTagSize+16], svtnB[:])

	// RULING-W6TB-H: HMAC-first ordering means the receiver derives the key
	// from the (now-forged) declared SVTN ID (svtnB). Since the body was
	// signed with svtnA's key but the declared SVTN is svtnB, HMAC
	// verification fails → ErrInvalidHMACTag before the SVTN check fires.
	forgedErr := d.ReceiveAdvertisement(ctx, forged)
	if !errors.Is(forgedErr, discovery.ErrInvalidHMACTag) {
		t.Fatalf("ReceiveAdvertisement forged-SVTN: got %v, want ErrInvalidHMACTag (RULING-W6TB-H: attacker must not get ErrSVTNMismatch before HMAC)", forgedErr)
	}
	// F-P5L2-MED-02: HMAC-first exclusivity — forged SVTN must never leak
	// ErrSVTNMismatch; if it does, the ordering invariant is violated.
	if errors.Is(forgedErr, discovery.ErrSVTNMismatch) {
		t.Fatal("HMAC-first exclusivity violated per RULING-W6TB-H")
	}
}

// TestDiscovery_Enumerate_SVTNIsolation_ErrSentinel verifies that when
// ReceiveAdvertisement receives a legitimate cross-scope advertisement
// (HMAC signed with the foreign SVTN's own key), it returns ErrSVTNMismatch —
// not nil and not ErrInvalidHMACTag (BC-2.03.002 Inv-1 sentinel).
//
// RULING-W6TB-H: encodeOrFail signs with payload.SVTNID (the foreign key),
// so HMAC verification passes under HMAC-first ordering; the SVTN cross-scope
// check then fires and returns ErrSVTNMismatch. This is the correct sentinel
// for legitimate admitted nodes on other SVTNs.
func TestDiscovery_Enumerate_SVTNIsolation_ErrSentinel(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t, svtnA, nodeA1)
	d := discovery.New(cfg)

	nodeB := [8]byte{0xCC}
	// encodeOrFail uses payload.SVTNID as the HMAC key — legitimate foreign
	// node signs with its own SVTN's key.
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
		t.Errorf("ReceiveAdvertisement cross-SVTN (legitimate foreign): err = %v, want errors.Is(err, ErrSVTNMismatch) == true (RULING-W6TB-H)", err)
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

			// Legitimate foreign-SVTN case (RULING-W6TB-H): advertisement
			// encoded with the foreign SVTN's own key (encodeOrFail uses
			// payload.SVTNID). HMAC-first: authentication passes (foreign
			// key is used to verify), then SVTN cross-scope check fires →
			// ErrSVTNMismatch. Silent drop (err==nil) is not acceptable.
			foreignAdv := encodeOrFail(t, discovery.AdvertisementPayload{
				NodeAddr: foreignNode,
				SVTNID:   pair.foreign,
				Sessions: []discovery.SessionPresence{
					{SessionName: "foreign-sess", Status: discovery.Detached, Quality: discovery.QualityGreen},
				},
			})
			if foreignErr := d.ReceiveAdvertisement(ctx, foreignAdv); foreignErr == nil || !errors.Is(foreignErr, discovery.ErrSVTNMismatch) {
				t.Fatalf("VP-045 foreign SVTN (legitimate): err=%v, expected ErrSVTNMismatch (RULING-W6TB-H)", foreignErr)
			}

			// Forged-SVTN case (RULING-W6TB-H): attacker writes foreign SVTN
			// bytes but cannot produce a valid HMAC for that foreign SVTN's
			// key. Simulate by encoding with the local key then flipping
			// the SVTN bytes in the wire body — HMAC now covers wrong bytes.
			// HMAC-first: receiver derives key from declared (forged) SVTN →
			// verification fails → must return ErrInvalidHMACTag.
			forgedAdv, encErr := discovery.Encode(discovery.AdvertisementPayload{
				NodeAddr: foreignNode,
				SVTNID:   pair.local, // sign with local key
				Sessions: []discovery.SessionPresence{
					{SessionName: "forged-svtn-sess", Status: discovery.Detached, Quality: discovery.QualityGreen},
				},
			})
			if encErr != nil {
				t.Fatalf("Encode forged adv: %v", encErr)
			}
			// Flip the SVTN ID bytes in the body to pair.foreign so that the
			// HMAC tag no longer matches the body content.
			const hmacTagSize = 8 // routing.AdvertisementHMACTagSize
			if len(forgedAdv) >= hmacTagSize+16 {
				copy(forgedAdv[hmacTagSize:hmacTagSize+16], pair.foreign[:])
			}
			forgedErr := d.ReceiveAdvertisement(ctx, forgedAdv)
			if !errors.Is(forgedErr, discovery.ErrInvalidHMACTag) {
				t.Fatalf("VP-045 forged SVTN: err=%v, want ErrInvalidHMACTag (RULING-W6TB-H: attacker must not receive ErrSVTNMismatch before HMAC fails)", forgedErr)
			}
			// F-P5L2-MED-02: HMAC-first exclusivity — forged SVTN must
			// never leak ErrSVTNMismatch before HMAC verification.
			if errors.Is(forgedErr, discovery.ErrSVTNMismatch) {
				t.Fatal("HMAC-first exclusivity violated per RULING-W6TB-H")
			}

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
				encoded, err := discovery.Encode(payload)
				if err != nil {
					return false
				}
				decoded, err := discovery.Decode(encoded)
				if err != nil {
					return false
				}
				reencoded, err := discovery.Encode(decoded)
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
			encoded, err := discovery.Encode(payload)
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
				encoded, err := discovery.Encode(payload)
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
				encoded, err := discovery.Encode(payload)
				if err != nil {
					return false // truncation must not error
				}
				decoded, decErr := discovery.Decode(encoded)
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
				// Round-trip stability: re-encoding the decoded payload must
				// produce the same bytes (truncated form is idempotent).
				reencoded, reErr := discovery.Encode(decoded)
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
		encoded, err := discovery.Encode(payload)
		if err != nil {
			return "", err
		}
		decoded, err := discovery.Decode(encoded)
		if err != nil {
			return "", err
		}
		if len(decoded.Sessions) != 1 {
			t.Fatalf("decoded session count = %d, want 1", len(decoded.Sessions))
		}
		// Verify round-trip stability of the encoded form.
		reencoded, err := discovery.Encode(decoded)
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
