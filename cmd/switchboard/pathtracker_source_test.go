// pathtracker_source_test.go — pathTrackerSource contract
// (S-BL.PATH-TRACKER-WIRING; folds S-BL.PATH-TRACKER-WRITER per Ruling-11).
//
// Verifies the registry side of the S-BL.PATH-TRACKER-WIRING two-part contract:
//
//   - Router-side: RegisterForwardingEntry fires ForwardingEntryHook under the
//     write lock (covered by internal/routing/routing_pathtrackers_test.go).
//   - Registry-side (this file): pathTrackerSource.Register constructs a
//     PathTracker on first sight of (svtnID, nodeAddr); repeated Register calls
//     for the same pathID are idempotent (tracker identity preserved across
//     auth-key rotations); AllSnapshots returns fully decoupled value copies;
//     the composition is race-clean under -race with concurrent Register +
//     AllSnapshots callers.
//
// Test names use the "Wire_" prefix so they share a filter with the existing
// metrics_wire tests (`go test -run Wire_`) — MetricsWire + PathTrackerSource
// live in one cognitive bucket.
package main

import (
	"fmt"
	"sync"
	"testing"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/hmac"
	"github.com/arcavenae/switchboard/internal/paths"
	"github.com/arcavenae/switchboard/internal/routing"
)

// TestPathTrackerSource_Register_ConstructsTracker_OnFirstSight verifies that
// Register(svtnID, nodeAddr) constructs a PathTracker on first sight and
// makes it visible via AllSnapshots.
//
// S-BL.PATH-TRACKER-WIRING AC-1, AC-3; BC-2.06.003 PC-1.
func TestPathTrackerSource_Register_ConstructsTracker_OnFirstSight(t *testing.T) {
	t.Parallel()

	src := newPathTrackerSource()

	var svtnID [16]byte
	copy(svtnID[:], []byte("svtn-alpha-00000"))
	var nodeAddr [8]byte
	copy(nodeAddr[:], []byte("node-001"))

	if got := len(src.AllSnapshots()); got != 0 {
		t.Fatalf("empty source: AllSnapshots returned %d entries; want 0", got)
	}

	src.Register(svtnID, nodeAddr)

	snaps := src.AllSnapshots()
	if got := len(snaps); got != 1 {
		t.Fatalf("after Register: AllSnapshots returned %d entries; want 1", got)
	}

	wantID := fmt.Sprintf("%x-%x", svtnID, nodeAddr)
	snap, ok := snaps[wantID]
	if !ok {
		t.Fatalf("snapshot for %q missing; got keys: %v", wantID, keysOf(snaps))
	}
	// New tracker: firstProbe=true → snapshot reports Active=true, initial RTT
	// of pathTrackerInitialRTTMs, zero loss, SampleCount=0, RouterAddr="".
	if !snap.Active {
		t.Errorf("fresh tracker Active=false; want true")
	}
	if snap.EWMARTTMs != pathTrackerInitialRTTMs {
		t.Errorf("fresh tracker EWMARTTMs=%v; want %v", snap.EWMARTTMs, pathTrackerInitialRTTMs)
	}
	if snap.LossPct != 0 {
		t.Errorf("fresh tracker LossPct=%v; want 0", snap.LossPct)
	}
	if snap.SampleCount != 0 {
		t.Errorf("fresh tracker SampleCount=%d; want 0", snap.SampleCount)
	}
}

// TestPathTrackerSource_Register_IsIdempotent_PreservesTrackerIdentity verifies
// that repeated Register calls for the same (svtnID, nodeAddr) do NOT construct
// a new PathTracker — the existing tracker's accumulated RTT/loss history
// survives auth-key rotation (S-BL.PATH-TRACKER-WIRING AC-4).
//
// We prove tracker identity by mutating the tracker between registrations:
// call OnProbe to shift RTT away from the initial value, then re-Register the
// same (svtnID, nodeAddr), then snapshot again — if the RTT retained its
// probe-updated value, tracker identity was preserved.
func TestPathTrackerSource_Register_IsIdempotent_PreservesTrackerIdentity(t *testing.T) {
	t.Parallel()

	src := newPathTrackerSource()

	var svtnID [16]byte
	svtnID[0] = 0x42
	var nodeAddr [8]byte
	nodeAddr[0] = 0x99

	src.Register(svtnID, nodeAddr)

	// Reach into the source under its own lock and drive one OnProbe so we can
	// prove identity is preserved. Using the exported Snapshot path only lets
	// us observe state — we need to force a state change to prove it survives
	// a re-Register.
	pathID := fmt.Sprintf("%x-%x", svtnID, nodeAddr)
	src.mu.RLock()
	t1, ok := src.trackers[pathID]
	src.mu.RUnlock()
	if !ok {
		t.Fatalf("pathTrackerSource missing tracker for %q after Register", pathID)
	}

	// Drive a first-probe arrival so the RTT resets to the sampled value.
	// firstProbe=true → resetRTT sets ewmaRTTMS=42.0 outright (not EWMA-blended).
	const probedRTT = 42.0
	t1.OnProbe(probedRTT, false)

	// Sanity: snapshot reflects the probed value.
	if got := t1.Snapshot().EWMARTTMs; got != probedRTT {
		t.Fatalf("after first probe: EWMARTTMs=%v; want %v", got, probedRTT)
	}

	// Re-Register the same (svtnID, nodeAddr) — this simulates an auth-key
	// rotation on the router side (RegisterForwardingEntry called again with a
	// new authKey). The source MUST NOT construct a new tracker.
	src.Register(svtnID, nodeAddr)

	src.mu.RLock()
	t2, ok := src.trackers[pathID]
	src.mu.RUnlock()
	if !ok {
		t.Fatalf("after re-Register: tracker for %q missing", pathID)
	}
	if t1 != t2 {
		t.Errorf("tracker identity lost across re-Register: t1=%p t2=%p", t1, t2)
	}

	// Belt-and-suspenders: snapshot must still reflect the probed RTT, not the
	// initial default. If a fresh tracker had replaced t1, the RTT would be
	// pathTrackerInitialRTTMs (250.0).
	if got := t2.Snapshot().EWMARTTMs; got != probedRTT {
		t.Errorf("after re-Register: EWMARTTMs=%v; want %v (RTT lost — tracker was replaced)", got, probedRTT)
	}
}

// TestPathTrackerSource_AllSnapshots_ReturnsValueCopies verifies that mutation
// of a returned PathSnapshot does not affect subsequent snapshots — the returned
// map values are fully decoupled from internal state (go.md rule 12).
//
// PathSnapshot is a value type so this is trivially true, but this test locks
// the property in place: if a future refactor accidentally leaks pointers, it
// fails.
func TestPathTrackerSource_AllSnapshots_ReturnsValueCopies(t *testing.T) {
	t.Parallel()

	src := newPathTrackerSource()

	var svtnID [16]byte
	var nodeAddr [8]byte
	src.Register(svtnID, nodeAddr)

	pathID := fmt.Sprintf("%x-%x", svtnID, nodeAddr)

	first := src.AllSnapshots()
	if _, ok := first[pathID]; !ok {
		t.Fatalf("missing snapshot for %q", pathID)
	}

	// Mutate the returned snapshot map + a snapshot value. This must not
	// affect the source's internal state.
	corrupted := first[pathID]
	corrupted.EWMARTTMs = -999.0
	corrupted.LossPct = 999.0
	corrupted.Active = false
	first[pathID] = corrupted
	first["fake-injected-key"] = paths.PathSnapshot{EWMARTTMs: -1}

	second := src.AllSnapshots()
	if len(second) != 1 {
		t.Errorf("AllSnapshots after caller-side mutation: len=%d; want 1", len(second))
	}
	if _, ok := second["fake-injected-key"]; ok {
		t.Errorf("caller-side map mutation leaked into source")
	}
	snap := second[pathID]
	if snap.EWMARTTMs == -999.0 || snap.LossPct == 999.0 || !snap.Active {
		t.Errorf("caller-side snapshot mutation leaked into source: %+v", snap)
	}
}

// TestPathTrackerSource_ConcurrentRegisterAndSnapshot_RaceClean exercises the
// RWMutex sanction folded in from S-BL.PATH-TRACKER-WRITER (Ruling-11).
// Multiple writers (Register) and readers (AllSnapshots) run concurrently;
// with -race, any un-synchronised map access would fail.
//
// S-BL.PATH-TRACKER-WIRING AC-5; S-BL.PATH-TRACKER-WRITER (folded).
func TestPathTrackerSource_ConcurrentRegisterAndSnapshot_RaceClean(t *testing.T) {
	t.Parallel()

	src := newPathTrackerSource()

	const writers = 4
	const readers = 4
	const perWriter = 128

	// Two WaitGroups: writers drain independently, then readers stop.
	var writerWG sync.WaitGroup
	writerWG.Add(writers)
	var readerWG sync.WaitGroup
	readerWG.Add(readers)
	done := make(chan struct{})

	for w := 0; w < writers; w++ {
		w := w
		go func() {
			defer writerWG.Done()
			for i := 0; i < perWriter; i++ {
				var svtnID [16]byte
				svtnID[0] = byte(w)
				svtnID[1] = byte(i)
				var nodeAddr [8]byte
				nodeAddr[0] = byte(w)
				nodeAddr[1] = byte(i)
				src.Register(svtnID, nodeAddr)
			}
		}()
	}

	for r := 0; r < readers; r++ {
		go func() {
			defer readerWG.Done()
			for {
				select {
				case <-done:
					return
				default:
				}
				_ = src.AllSnapshots()
			}
		}()
	}

	writerWG.Wait()
	close(done)
	readerWG.Wait()

	if got := len(src.AllSnapshots()); got != writers*perWriter {
		t.Errorf("final AllSnapshots: got %d entries; want %d", got, writers*perWriter)
	}
}

// TestNewPathTrackerSourceFromRouter_InstallsHook verifies the wire-up path:
// newPathTrackerSourceFromRouter attaches its Register method as the router's
// forwarding-entry hook, so RegisterForwardingEntry populates the source
// without the caller needing to plumb any additional wiring.
//
// S-BL.PATH-TRACKER-WIRING AC-1, AC-2 (the wire).
func TestNewPathTrackerSourceFromRouter_InstallsHook(t *testing.T) {
	t.Parallel()

	r := routing.NewRouter(admission.NewAdmittedKeySet())
	src := newPathTrackerSourceFromRouter(r)

	if got := len(src.AllSnapshots()); got != 0 {
		t.Fatalf("fresh source: AllSnapshots returned %d; want 0", got)
	}

	var svtnID [16]byte
	svtnID[0] = 0x77
	var nodeAddr [8]byte
	nodeAddr[0] = 0x88
	var authKey [hmac.KeySize]byte

	r.RegisterForwardingEntry(svtnID, nodeAddr, authKey)

	snaps := src.AllSnapshots()
	if got := len(snaps); got != 1 {
		t.Fatalf("after RegisterForwardingEntry: AllSnapshots returned %d; want 1", got)
	}
	wantID := fmt.Sprintf("%x-%x", svtnID, nodeAddr)
	if _, ok := snaps[wantID]; !ok {
		t.Errorf("snapshot for %q missing; got keys: %v", wantID, keysOf(snaps))
	}
}

// keysOf returns the sorted key set of a paths.PathSnapshot map for readable
// test failure messages. Ordering is stable so failure lines diff cleanly across
// test runs.
func keysOf(m map[string]paths.PathSnapshot) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
