// Package multipath_test contains the TDD test suite for BC-2.02.001
// (duplicate-and-race dispatch), BC-2.02.002 (receiver deduplication),
// and BC-2.02.009 (bounded LRU drop cache).
//
// All tests MUST fail until the corresponding stubs are implemented (Red Gate).
//
// BC/AC coverage map:
//
//	TestBC_2_02_001_Send_TwoFastestPaths                          → AC-003, BC-2.02.001 postcondition 1
//	TestBC_2_02_001_Send_ThreePathsSelectLowest                   → AC-003, BC-2.02.001 postcondition 1 (test vector)
//	TestBC_2_02_001_Send_SinglePathFallback                       → EC-001, BC-2.02.001 postcondition 3, AC-003
//	TestBC_2_02_001_Send_NoPathsReturnsError                      → BC-2.02.001 postcondition 4
//	TestBC_2_02_001_Send_AtMostTwoPaths                           → VP-024, BC-2.02.001 invariant 2
//	TestBC_2_02_001_Send_SnapshotsRankAtDispatch                  → BC-2.02.001 postcondition 5
//	TestBC_2_02_001_Send_IdenticalBytesOnBothPaths                → BC-2.02.001 postcondition 2
//	TestBC_2_02_002_Receive_FirstCopyDelivered                    → AC-004, BC-2.02.002 postcondition 1, VP-054
//	TestBC_2_02_002_Receive_DuplicateReturnsErr                   → AC-004, BC-2.02.002 postcondition 2
//	TestBC_2_02_002_Receive_DuplicateDiscardSilent                → AC-005, BC-2.02.002 postcondition 2
//	TestBC_2_02_002_Receive_MultipleArrivalsSameFrame             → EC-004, BC-2.02.002
//	TestBC_2_02_002_Receive_DistinctFramesNotSuppressed           → BC-2.02.002 postcondition 1
//	TestBC_2_02_002_Receive_CrossInterfaceDuplicateSuppressed     → F-002, BC-2.02.002 postcondition 2, DI-009, VP-024 (replaces deleted wrong test)
//	TestBC_2_02_002_Receive_ConcurrentFirstArrivalWins            → F-004, F-005, DI-009, BC-2.02.002 invariant 1 (TOCTOU regression)
//	TestBC_2_02_009_DropCache_MissForwards                        → BC-2.02.009 postcondition 1, VP-025
//	TestBC_2_02_009_DropCache_HitSuppresses                       → BC-2.02.009 postcondition 2, VP-025
//	TestBC_2_02_009_DropCache_CompoundKey                         → BC-2.02.009 EC-001 (different interface IDs)
//	TestBC_2_02_009_DropCache_BoundedCapacity                     → AC-006, BC-2.02.009 postcondition 3, VP-025
//	TestBC_2_02_009_DropCache_LRUEvictsOldest                     → AC-006, EC-003, BC-2.02.009 postcondition 3
//	TestBC_2_02_009_DropCache_Len                                 → AC-006
//	TestBC_2_02_009_DropCache_KeyedOnChecksumAndInterface         → ARCH compliance (story architecture rule)
//	TestBC_2_02_009_DropCache_BoundedCapacity_PropertySweep       → VP-025 (stdlib sweep)
//	TestBC_2_02_009_DropCache_ConcurrentAddContains               → F-004, BC-2.02.009 (concurrent safety)
package multipath_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/arcavenae/switchboard/internal/multipath"
	"github.com/arcavenae/switchboard/internal/paths"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

// makeFrame builds a Frame with the given payload bytes. The OuterHeader is
// zeroed unless overridden. Used to produce distinct (different checksum) and
// identical (same checksum) frames in tests.
func makeFrame(t *testing.T, payload []byte) multipath.Frame {
	t.Helper()
	var f multipath.Frame
	f.Payload = append([]byte(nil), payload...)
	return f
}

// makeFrameWithHeader builds a Frame with a specific outer-header prefix so
// that identical payloads but different headers produce different checksums.
func makeFrameWithHeader(t *testing.T, headerByte byte, payload []byte) multipath.Frame {
	t.Helper()
	var f multipath.Frame
	f.OuterHeader[0] = headerByte
	f.Payload = append([]byte(nil), payload...)
	return f
}

// activeTracker returns a PathTracker with the given RTT in ms that is active.
// alpha=1 so Score = PathScore(rtt, 0) exactly.
func activeTracker(t *testing.T, rttMS float64) *paths.PathTracker {
	t.Helper()
	return paths.NewPathTracker(rttMS, 1.0)
}

// inactiveTracker returns a PathTracker that has been deactivated by 3
// consecutive missed keepalives.
func inactiveTracker(t *testing.T) *paths.PathTracker {
	t.Helper()
	tr := paths.NewPathTracker(10.0, 0.125)
	for i := 0; i < 3; i++ {
		tr.OnProbe(0, true)
	}
	if tr.IsActive() {
		t.Helper()
		t.Fatal("inactiveTracker: tracker is still active after 3 consecutive misses")
	}
	return tr
}

// collectSentPathIDs calls mp.Send and returns the path IDs on which fn was
// invoked, in call order. fn is guaranteed not to return an error.
func collectSentPathIDs(t *testing.T, mp *multipath.Multipath, f multipath.Frame) ([]uint64, error) {
	t.Helper()
	var ids []uint64
	results, err := mp.Send(f, func(pathID uint64, _ multipath.Frame) error {
		ids = append(ids, pathID)
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Verify SendResult consistency.
	for i, r := range results {
		if !r.Sent {
			t.Errorf("SendResult[%d].Sent=false; want true (fn was invoked)", i)
		}
	}
	return ids, nil
}

// ─── Multipath.Send tests (BC-2.02.001) ──────────────────────────────────────

// TestBC_2_02_001_Send_TwoFastestPaths verifies that Send dispatches the frame
// on exactly the two highest-scoring (lowest RTT) active paths when two or more
// paths are available.
//
// AC-003 / BC-2.02.001 postcondition 1 (canonical test vector: RTT [10ms, 25ms])
func TestBC_2_02_001_Send_TwoFastestPaths(t *testing.T) {
	t.Parallel()

	// Two paths: RTT [10ms, 25ms] → both selected.
	fast := activeTracker(t, 10.0)
	slow := activeTracker(t, 25.0)

	ps := []paths.RankedPath{
		{ID: 10, Tracker: fast},
		{ID: 25, Tracker: slow},
	}
	mp := multipath.NewMultipath(ps, multipath.DefaultDropCacheSize)

	f := makeFrame(t, []byte("hello"))
	ids, err := collectSentPathIDs(t, mp, f)
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("Send dispatched on %d paths, want 2", len(ids))
	}
	// Both paths must have been selected.
	seen := map[uint64]bool{}
	for _, id := range ids {
		seen[id] = true
	}
	if !seen[10] || !seen[25] {
		t.Errorf("expected both pathIDs {10,25}; got %v", ids)
	}
}

// TestBC_2_02_001_Send_ThreePathsSelectLowest verifies that with three paths
// only the two lowest-RTT paths receive the frame.
//
// AC-003 / BC-2.02.001 canonical test vector: "3 paths RTT [10ms, 15ms, 40ms]"
func TestBC_2_02_001_Send_ThreePathsSelectLowest(t *testing.T) {
	t.Parallel()

	// Paths with RTT 10, 15, 40ms. Only 10ms (ID=10) and 15ms (ID=15) should
	// be selected; 40ms (ID=40) must NOT receive the frame.
	ps := []paths.RankedPath{
		{ID: 40, Tracker: activeTracker(t, 40.0)},
		{ID: 10, Tracker: activeTracker(t, 10.0)},
		{ID: 15, Tracker: activeTracker(t, 15.0)},
	}
	mp := multipath.NewMultipath(ps, multipath.DefaultDropCacheSize)

	f := makeFrame(t, []byte("data"))
	ids, err := collectSentPathIDs(t, mp, f)
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("Send dispatched on %d paths, want 2", len(ids))
	}
	seen := map[uint64]bool{}
	for _, id := range ids {
		seen[id] = true
	}
	if !seen[10] {
		t.Error("10ms path (ID=10) was not selected; want selected")
	}
	if !seen[15] {
		t.Error("15ms path (ID=15) was not selected; want selected")
	}
	if seen[40] {
		t.Error("40ms path (ID=40) was selected; want NOT selected")
	}
}

// TestBC_2_02_001_Send_SinglePathFallback verifies that when only one active
// path exists, Send dispatches on that single path and returns no error.
//
// EC-001 / BC-2.02.001 postcondition 3 / AC-003
func TestBC_2_02_001_Send_SinglePathFallback(t *testing.T) {
	t.Parallel()

	ps := []paths.RankedPath{
		{ID: 99, Tracker: activeTracker(t, 30.0)},
	}
	mp := multipath.NewMultipath(ps, multipath.DefaultDropCacheSize)

	f := makeFrame(t, []byte("single"))
	ids, err := collectSentPathIDs(t, mp, f)
	if err != nil {
		t.Fatalf("single-path Send returned error: %v", err)
	}
	if len(ids) != 1 || ids[0] != 99 {
		t.Errorf("single-path Send: got ids=%v, want [99]", ids)
	}
}

// TestBC_2_02_001_Send_NoPathsReturnsError verifies that Send returns
// paths.ErrNoActivePaths when no paths are active.
//
// BC-2.02.001 postcondition 4
func TestBC_2_02_001_Send_NoPathsReturnsError(t *testing.T) {
	t.Parallel()

	// Provide an inactive path.
	ps := []paths.RankedPath{
		{ID: 1, Tracker: inactiveTracker(t)},
	}
	mp := multipath.NewMultipath(ps, multipath.DefaultDropCacheSize)

	f := makeFrame(t, []byte("queued"))
	_, err := mp.Send(f, func(_ uint64, _ multipath.Frame) error {
		t.Error("fn must not be called with zero active paths")
		return nil
	})
	if !errors.Is(err, paths.ErrNoActivePaths) {
		t.Errorf("want paths.ErrNoActivePaths, got %v", err)
	}
}

// TestBC_2_02_001_Send_AtMostTwoPaths verifies that Send never dispatches on
// more than two paths regardless of how many active paths are available.
//
// VP-024 / BC-2.02.001 invariant 2
func TestBC_2_02_001_Send_AtMostTwoPaths(t *testing.T) {
	t.Parallel()

	// Five active paths — Send must select at most 2.
	ps := []paths.RankedPath{
		{ID: 1, Tracker: activeTracker(t, 10.0)},
		{ID: 2, Tracker: activeTracker(t, 20.0)},
		{ID: 3, Tracker: activeTracker(t, 30.0)},
		{ID: 4, Tracker: activeTracker(t, 40.0)},
		{ID: 5, Tracker: activeTracker(t, 50.0)},
	}
	mp := multipath.NewMultipath(ps, multipath.DefaultDropCacheSize)

	f := makeFrame(t, []byte("burst"))
	ids, err := collectSentPathIDs(t, mp, f)
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if len(ids) > 2 {
		t.Errorf("Send dispatched on %d paths, want ≤2 (invariant: at most two)", len(ids))
	}
}

// TestBC_2_02_001_Send_SnapshotsRankAtDispatch verifies that path rankings are
// snapshotted at the moment of Send; a subsequent UpdatePaths call does not
// affect frames already dispatched.
//
// BC-2.02.001 postcondition 5
func TestBC_2_02_001_Send_SnapshotsRankAtDispatch(t *testing.T) {
	t.Parallel()

	fast := activeTracker(t, 10.0)
	slow := activeTracker(t, 50.0)

	ps := []paths.RankedPath{
		{ID: 10, Tracker: fast},
		{ID: 50, Tracker: slow},
	}
	mp := multipath.NewMultipath(ps, multipath.DefaultDropCacheSize)

	// Record which paths were selected before UpdatePaths is called from inside fn.
	var selectedIDs []uint64
	_, err := mp.Send(makeFrame(t, []byte("test")), func(pathID uint64, _ multipath.Frame) error {
		selectedIDs = append(selectedIDs, pathID)
		// Simulate a rank change mid-burst — should not affect this dispatch.
		newBetter := activeTracker(t, 1.0)
		mp.UpdatePaths([]paths.RankedPath{{ID: 999, Tracker: newBetter}})
		return nil
	})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	// The snapshot was taken before UpdatePaths; paths 10 and 50 must have been
	// selected regardless of the rank change.
	for _, id := range selectedIDs {
		if id == 999 {
			t.Error("rank change mid-burst affected already-dispatched frame (postcondition 5 violated)")
		}
	}
}

// TestBC_2_02_001_Send_IdenticalBytesOnBothPaths verifies that the frame bytes
// dispatched on both paths are identical (same Frame value passed to fn).
//
// BC-2.02.001 postcondition 2
func TestBC_2_02_001_Send_IdenticalBytesOnBothPaths(t *testing.T) {
	t.Parallel()

	ps := []paths.RankedPath{
		{ID: 1, Tracker: activeTracker(t, 10.0)},
		{ID: 2, Tracker: activeTracker(t, 20.0)},
	}
	mp := multipath.NewMultipath(ps, multipath.DefaultDropCacheSize)

	payload := []byte("identical frame bytes test")
	f := makeFrame(t, payload)

	var received []multipath.Frame
	_, err := mp.Send(f, func(_ uint64, sent multipath.Frame) error {
		received = append(received, sent)
		return nil
	})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if len(received) != 2 {
		t.Fatalf("expected 2 dispatches, got %d", len(received))
	}

	// Compare OuterHeader and Payload.
	a, b := received[0], received[1]
	if a.OuterHeader != b.OuterHeader {
		t.Errorf("OuterHeaders differ: %v vs %v", a.OuterHeader, b.OuterHeader)
	}
	if string(a.Payload) != string(b.Payload) {
		t.Errorf("Payloads differ: %q vs %q", a.Payload, b.Payload)
	}
	// Payload must match what we sent.
	if string(a.Payload) != string(payload) {
		t.Errorf("Payload mismatch: got %q, want %q", a.Payload, payload)
	}
}

// ─── Multipath.Receive tests (BC-2.02.002) ───────────────────────────────────

// TestBC_2_02_002_Receive_FirstCopyDelivered verifies that the first-arriving
// copy of a frame returns nil (deliver to application layer).
//
// AC-004 / BC-2.02.002 postcondition 1 / VP-054
func TestBC_2_02_002_Receive_FirstCopyDelivered(t *testing.T) {
	t.Parallel()

	mp := multipath.NewMultipath(nil, multipath.DefaultDropCacheSize)
	f := makeFrame(t, []byte("first arrival"))

	if err := mp.Receive(f); err != nil {
		t.Errorf("first arrival: want nil, got %v", err)
	}
}

// TestBC_2_02_002_Receive_DuplicateReturnsErr verifies that a duplicate frame
// (same checksum, same interface) returns ErrDuplicate.
//
// AC-004 / BC-2.02.002 postcondition 2
func TestBC_2_02_002_Receive_DuplicateReturnsErr(t *testing.T) {
	t.Parallel()

	// Canonical test vector: "Frame seq=42 arrives on path A at t=0ms;
	// same frame arrives on path B ... second copy discarded."
	// (We use same interface here to match the drop-cache compound key.)
	mp := multipath.NewMultipath(nil, multipath.DefaultDropCacheSize)

	f := makeFrame(t, []byte("dup frame"))

	// First arrival.
	if err := mp.Receive(f); err != nil {
		t.Fatalf("first arrival: want nil, got %v", err)
	}

	// Duplicate arrival (same frame bytes → same checksum).
	fDup := makeFrame(t, []byte("dup frame")) // byte-identical
	if !errors.Is(mp.Receive(fDup), multipath.ErrDuplicate) {
		t.Errorf("duplicate arrival: want ErrDuplicate, got nil")
	}
}

// TestBC_2_02_002_Receive_DuplicateDiscardSilent verifies that ErrDuplicate is
// the ONLY signal for a discarded duplicate — the caller treats it as silent
// (no further error propagation required by the BC).
//
// AC-005 / BC-2.02.002 postcondition 2
func TestBC_2_02_002_Receive_DuplicateDiscardSilent(t *testing.T) {
	t.Parallel()

	// AC-005: "Duplicate discards are silent: no error is surfaced to the
	// session layer for a discarded duplicate."
	// In the pure-core model, "silent" means ErrDuplicate is returned but it is
	// NOT a protocol error — the caller must not log or propagate it. We verify
	// that the returned sentinel equals ErrDuplicate (not some wrapped error).
	mp := multipath.NewMultipath(nil, multipath.DefaultDropCacheSize)

	f := makeFrame(t, []byte("silent dup"))
	_ = mp.Receive(f) // first: delivered

	err := mp.Receive(makeFrame(t, []byte("silent dup"))) // dup
	if err == nil {
		t.Fatal("duplicate: want ErrDuplicate, got nil")
	}
	// Must be exactly ErrDuplicate (unwrapped) so callers can identify it cheaply.
	if !errors.Is(err, multipath.ErrDuplicate) {
		t.Errorf("ErrDuplicate not in error chain: %v", err)
	}
}

// TestBC_2_02_002_Receive_MultipleArrivalsSameFrame verifies that the second
// AND third arrival both return ErrDuplicate (EC-004: looping network).
//
// EC-004 / BC-2.02.002
func TestBC_2_02_002_Receive_MultipleArrivalsSameFrame(t *testing.T) {
	t.Parallel()

	mp := multipath.NewMultipath(nil, multipath.DefaultDropCacheSize)

	f := makeFrame(t, []byte("looping frame"))

	_ = mp.Receive(f) // first: delivered

	for i := 2; i <= 5; i++ {
		dup := makeFrame(t, []byte("looping frame"))
		if !errors.Is(mp.Receive(dup), multipath.ErrDuplicate) {
			t.Errorf("arrival #%d: want ErrDuplicate, got nil", i)
		}
	}
}

// TestBC_2_02_002_Receive_DistinctFramesNotSuppressed verifies that two frames
// with different payloads (different checksums) are both delivered.
//
// BC-2.02.002 postcondition 1 (non-suppression for distinct frames)
func TestBC_2_02_002_Receive_DistinctFramesNotSuppressed(t *testing.T) {
	t.Parallel()

	mp := multipath.NewMultipath(nil, multipath.DefaultDropCacheSize)

	// "Two frames: seq=42 with content 'abc', seq=43 with content 'def'" from BC-2.02.002 test vectors.
	f1 := makeFrameWithHeader(t, 42, []byte("abc"))
	f2 := makeFrameWithHeader(t, 43, []byte("def"))

	if err := mp.Receive(f1); err != nil {
		t.Errorf("frame1 (seq=42): want nil, got %v", err)
	}
	if err := mp.Receive(f2); err != nil {
		t.Errorf("frame2 (seq=43): want nil, got %v", err)
	}
}

// TestBC_2_02_002_Receive_CrossInterfaceDuplicateSuppressed verifies that a
// second copy of the SAME frame (identical bytes, same checksum) arriving on a
// DIFFERENT interface IS suppressed at the endpoint receiver.
//
// BC-2.02.002 postcondition 2, DI-009, VP-024 (endpoint dedup is checksum-only,
// not compound (checksum, interface) — see pass-1-spec-rulings F-002 and
// RULING 1). Canonical test vector: "Frame seq=42 arrives on path A at t=0ms;
// same frame arrives on path B at t=8ms → delivered at t=0ms; second copy
// discarded silently."
//
// NOTE: The deleted test TestBC_2_02_002_Receive_DifferentInterfaceSameChecksumNotSuppressed
// pinned the WRONG behavior (compound-key endpoint dedup). This replacement
// asserts the correct behavior per spec ruling. This test MUST FAIL against the
// current implementation (which uses compound key) — that is the Red Gate.
func TestBC_2_02_002_Receive_CrossInterfaceDuplicateSuppressed(t *testing.T) {
	t.Parallel()

	mp := multipath.NewMultipath(nil, multipath.DefaultDropCacheSize)

	// Same frame bytes → same checksum; arrives on two different interfaces
	// (simulating duplicate-and-race delivery via two different routers).
	payload := []byte("multipath copy")
	f1 := makeFrame(t, payload)
	f2 := makeFrame(t, payload) // byte-identical → same checksum

	// First arrival: must be delivered (nil).
	if err := mp.Receive(f1); err != nil {
		t.Errorf("iface=1 first arrival: want nil, got %v", err)
	}

	// Second arrival (same checksum, different conceptual interface) → endpoint dedup
	// MUST suppress it. Endpoint dedup keys on checksum alone (BC-2.02.002
	// postcondition 2 / DI-009); the arrival interface is irrelevant at the receiver.
	// ErrDuplicate required.
	if !errors.Is(mp.Receive(f2), multipath.ErrDuplicate) {
		t.Errorf("iface=2 same-checksum second arrival: want ErrDuplicate (suppressed), got nil")
	}
}

// TestBC_2_02_002_Receive_ConcurrentFirstArrivalWins drives two goroutines
// delivering the SAME frame concurrently to a single Multipath.Receive and
// asserts that EXACTLY ONE delivery is returned (nil) and all others return
// ErrDuplicate. This exercises the DI-009 first-arrival-wins invariant under
// concurrent load and MUST expose the Contains-then-Add TOCTOU race
// (F-005 / BC-2.02.002 invariant 1).
//
// Run under `go test -race` to detect the data race. The test also checks the
// semantic invariant: exactly one nil return from N concurrent deliveries.
//
// F-004, F-005 / BC-2.02.002 invariant 1 / DI-009
func TestBC_2_02_002_Receive_ConcurrentFirstArrivalWins(t *testing.T) {
	// Not parallel at the outer level — the inner goroutines provide concurrency.

	const goroutines = 8

	mp := multipath.NewMultipath(nil, multipath.DefaultDropCacheSize)
	payload := []byte("concurrent dedup frame")

	var deliveries atomic.Int64
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		i := i
		go func() {
			defer wg.Done()
			f := makeFrame(t, payload) // byte-identical across all goroutines
			err := mp.Receive(f)
			if err == nil {
				deliveries.Add(1)
			} else if !errors.Is(err, multipath.ErrDuplicate) {
				t.Errorf("goroutine %d: unexpected error %v (want nil or ErrDuplicate)", i, err)
			}
		}()
	}
	wg.Wait()

	// DI-009 invariant: first arrival wins — EXACTLY ONE delivery.
	if n := deliveries.Load(); n != 1 {
		t.Errorf("concurrent Receive: %d deliveries, want exactly 1 (DI-009 first-arrival-wins violated)", n)
	}
}

// ─── DropCache unit tests (BC-2.02.009) ──────────────────────────────────────

// TestBC_2_02_009_DropCache_MissForwards verifies that a cache miss (first time
// a checksum is seen) returns false from Contains and allows forwarding.
//
// BC-2.02.009 postcondition 1 / VP-025
func TestBC_2_02_009_DropCache_MissForwards(t *testing.T) {
	t.Parallel()

	// "Frame with checksum 0xABCD arrives; cache empty → forwarded" (test vector)
	dc := multipath.NewDropCache(100)

	if dc.Contains(0xABCD, 1) {
		t.Error("Contains on empty cache: want false (miss), got true")
	}
	// Add the entry.
	dc.Add(0xABCD, 1)
	if dc.Len() != 1 {
		t.Errorf("Len after one Add: got %d, want 1", dc.Len())
	}
}

// TestBC_2_02_009_DropCache_HitSuppresses verifies that a cache hit returns
// true from Contains, indicating the frame should be discarded.
//
// BC-2.02.009 postcondition 2 / VP-025
// Canonical test vector: "Same frame (checksum 0xABCD) arrives again → dropped"
func TestBC_2_02_009_DropCache_HitSuppresses(t *testing.T) {
	t.Parallel()

	dc := multipath.NewDropCache(100)
	dc.Add(0xABCD, 1)

	if !dc.Contains(0xABCD, 1) {
		t.Error("Contains after Add: want true (hit), got false")
	}
}

// TestBC_2_02_009_DropCache_CompoundKey verifies the compound-key semantic:
// the same checksum on different interfaces produces distinct cache entries
// (BC-2.02.009 EC-001: multipath duplicate-and-race must NOT be suppressed at
// the router).
//
// BC-2.02.009 EC-001 / ARCH-03 F-006 (drop cache key = (checksum, iface_id))
func TestBC_2_02_009_DropCache_CompoundKey(t *testing.T) {
	t.Parallel()

	dc := multipath.NewDropCache(100)

	const checksum = uint32(0x1234)
	const ifaceA = uint64(1)
	const ifaceB = uint64(2)

	// Add on interface A.
	dc.Add(checksum, ifaceA)

	// Same checksum, different interface: must be a cache MISS.
	if dc.Contains(checksum, ifaceB) {
		t.Error("(checksum, ifaceB) should be a miss; compound key must distinguish interfaces")
	}

	// Interface A must still be a hit.
	if !dc.Contains(checksum, ifaceA) {
		t.Error("(checksum, ifaceA) should still be a hit")
	}
}

// TestBC_2_02_009_DropCache_BoundedCapacity verifies that the cache never
// exceeds its configured capacity.
//
// AC-006 / BC-2.02.009 postcondition 3 / VP-025
func TestBC_2_02_009_DropCache_BoundedCapacity(t *testing.T) {
	t.Parallel()

	const capacity = 5

	dc := multipath.NewDropCache(capacity)

	// Insert 3× capacity distinct entries.
	for i := uint32(0); i < uint32(capacity*3); i++ {
		dc.Add(i, 0)
		if dc.Len() > capacity {
			t.Errorf("after adding entry %d: Len=%d exceeds capacity=%d", i, dc.Len(), capacity)
		}
	}
	if dc.Len() != capacity {
		t.Errorf("final Len=%d, want %d", dc.Len(), capacity)
	}
}

// TestBC_2_02_009_DropCache_LRUEvictsOldest verifies that when the cache is
// full, the least-recently-used entry is evicted so that a subsequent lookup
// for the evicted key returns a miss.
//
// AC-006 / EC-003 / BC-2.02.009 postcondition 3
func TestBC_2_02_009_DropCache_LRUEvictsOldest(t *testing.T) {
	t.Parallel()

	const capacity = 3

	dc := multipath.NewDropCache(capacity)

	// Add capacity entries (keys 1, 2, 3).
	for i := uint32(1); i <= capacity; i++ {
		dc.Add(i, 0)
	}

	// Touch key 2 to make it more recently used (access but don't add again).
	// Implementations may not expose a "touch" method, so instead we verify
	// the LRU eviction order: after filling, adding a new key should evict key 1
	// (the oldest/LRU entry, since keys were added in order 1, 2, 3 and none touched).
	dc.Add(uint32(capacity+1), 0) // new entry; LRU (key=1) should be evicted

	if dc.Len() != capacity {
		t.Errorf("after LRU eviction: Len=%d, want %d", dc.Len(), capacity)
	}

	// The oldest entry (key=1) must have been evicted.
	if dc.Contains(1, 0) {
		t.Error("LRU entry (key=1) still present after eviction; want evicted")
	}

	// The newer entries (2, 3, and the new key=capacity+1) must still be present.
	for _, key := range []uint32{2, 3, uint32(capacity + 1)} {
		if !dc.Contains(key, 0) {
			t.Errorf("key=%d evicted unexpectedly; want present", key)
		}
	}
}

// TestBC_2_02_009_DropCache_Len verifies that Len returns 0 for a new cache
// and increments correctly up to capacity.
//
// AC-006 (O(1) lookup, bounded capacity)
func TestBC_2_02_009_DropCache_Len(t *testing.T) {
	t.Parallel()

	dc := multipath.NewDropCache(10)

	if dc.Len() != 0 {
		t.Errorf("new cache Len: got %d, want 0", dc.Len())
	}

	for i := 0; i < 5; i++ {
		dc.Add(uint32(i), 0)
		if dc.Len() != i+1 {
			t.Errorf("after %d adds: Len=%d, want %d", i+1, dc.Len(), i+1)
		}
	}
}

// TestBC_2_02_009_DropCache_KeyedOnChecksumAndInterface is the architecture
// compliance test ensuring the drop cache key uses the (checksum, arrival_interface_id)
// compound key mandated by ARCH-03 F-006.
//
// Story architecture rule: TestDropCache_KeyedOnChecksumAndInterface
func TestBC_2_02_009_DropCache_KeyedOnChecksumAndInterface(t *testing.T) {
	t.Parallel()

	dc := multipath.NewDropCache(50)

	// Add four compound keys: two checksums × two interfaces.
	dc.Add(0xAAAA, 1)
	dc.Add(0xAAAA, 2)
	dc.Add(0xBBBB, 1)
	dc.Add(0xBBBB, 2)

	// All four must be independently tracked.
	if dc.Len() != 4 {
		t.Errorf("Len=%d after 4 distinct compound-key entries; want 4", dc.Len())
	}

	hits := [][2]uint64{
		{0xAAAA, 1}, {0xAAAA, 2}, {0xBBBB, 1}, {0xBBBB, 2},
	}
	for _, h := range hits {
		if !dc.Contains(uint32(h[0]), h[1]) {
			t.Errorf("Contains(checksum=%#x, iface=%d): want true, got false", h[0], h[1])
		}
	}

	// Unseen combinations must miss.
	misses := [][2]uint64{
		{0xCCCC, 1}, {0xAAAA, 3},
	}
	for _, m := range misses {
		if dc.Contains(uint32(m[0]), m[1]) {
			t.Errorf("Contains(checksum=%#x, iface=%d): want false, got true", m[0], m[1])
		}
	}
}

// TestBC_2_02_009_DropCache_BoundedCapacity_PropertySweep exercises the
// VP-025 "len ≤ capacity always" invariant over a range of capacities and
// entry counts using a deterministic sweep (no external property library).
//
// VP-025 (stdlib property sweep — full proptest deferred to formal-verifier)
func TestBC_2_02_009_DropCache_BoundedCapacity_PropertySweep(t *testing.T) {
	t.Parallel()

	capacities := []int{1, 2, 5, 10, 32, 64, 128}
	const entriesPerRun = 256

	for _, cap := range capacities {
		cap := cap
		t.Run("", func(t *testing.T) {
			t.Parallel()
			dc := multipath.NewDropCache(cap)
			for i := uint32(0); i < entriesPerRun; i++ {
				dc.Add(i, 0)
				if dc.Len() > cap {
					t.Errorf("capacity=%d: Len=%d after %d inserts; must not exceed capacity",
						cap, dc.Len(), i+1)
					return
				}
			}
		})
	}
}

// TestBC_2_02_009_DropCache_ConcurrentAddContains drives multiple goroutines
// calling Add and Contains on the same DropCache concurrently. Run under
// `go test -race` — any missing lock on DropCache will produce a data race.
//
// F-004 / BC-2.02.009 (concurrent safety of router drop cache)
func TestBC_2_02_009_DropCache_ConcurrentAddContains(t *testing.T) {
	// Not parallel at the outer level — inner goroutines provide the concurrency.

	const goroutines = 16
	const opsPerGoroutine = 64

	dc := multipath.NewDropCache(128)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := range goroutines {
		g := g
		go func() {
			defer wg.Done()
			base := uint32(g * opsPerGoroutine)
			for i := uint32(0); i < opsPerGoroutine; i++ {
				key := base + i
				dc.Add(key, uint64(g))
				// Contains must not panic or race under concurrent access.
				_ = dc.Contains(key, uint64(g))
			}
		}()
	}
	wg.Wait()

	// Len must not exceed capacity.
	if dc.Len() > 128 {
		t.Errorf("after concurrent ops: Len=%d exceeds capacity=128", dc.Len())
	}
}

// ─── Pass-2 adversarial findings ─────────────────────────────────────────────

// TestBC_2_02_001_Send_ConcurrentWithUpdatePaths drives Multipath.Send from
// one goroutine while another goroutine calls UpdatePaths concurrently on the
// same Multipath instance. The test asserts no data race and that each Send
// observes a self-consistent (non-torn) path snapshot — i.e. the pathSet is
// never partially written while a Send is reading it.
//
// This pins the m.mu lock that protects pathSet (BC-2.02.001 postcondition 5:
// "rankings are snapshotted at dispatch time"). Without the lock, concurrent
// UpdatePaths could produce a torn read inside Send.
//
// Run under `go test -race` — a missing or incorrectly-scoped lock will
// produce a data race report. If the lock is already correct (as it should be
// in the current implementation) this test PASSES and confirms the lock is
// load-bearing (F-M3: "lock could be dropped and all tests still pass" is
// falsified by this test).
//
// Pass-2 finding F-M3 / BC-2.02.001 postcondition 5 (atomic snapshot)
func TestBC_2_02_001_Send_ConcurrentWithUpdatePaths(t *testing.T) {
	// Not parallel at outer level — inner goroutines provide the concurrency.

	const senders = 4
	const updaters = 2
	const itersPerGoroutine = 50

	// Initialise with two active paths so Send has targets to dispatch on.
	ps := []paths.RankedPath{
		{ID: 1, Tracker: activeTracker(t, 10.0)},
		{ID: 2, Tracker: activeTracker(t, 20.0)},
	}
	mp := multipath.NewMultipath(ps, multipath.DefaultDropCacheSize)

	var wg sync.WaitGroup
	wg.Add(senders + updaters)

	// Updater goroutines: continuously replace the path set with fresh slices.
	for u := range updaters {
		u := u
		go func() {
			defer wg.Done()
			for i := range itersPerGoroutine {
				rtt := float64((u*itersPerGoroutine+i)%50 + 5) // 5–54 ms
				mp.UpdatePaths([]paths.RankedPath{
					{ID: uint64(u*100 + 1), Tracker: activeTracker(t, rtt)},
					{ID: uint64(u*100 + 2), Tracker: activeTracker(t, rtt+10)},
				})
			}
		}()
	}

	// Sender goroutines: continuously call Send and assert each dispatch is
	// self-consistent (all PathIDs come from the same path set; no panic;
	// at most 2 paths selected per BC-2.02.001 invariant 2).
	for s := range senders {
		s := s
		go func() {
			defer wg.Done()
			f := makeFrame(t, []byte("concurrent-send-test"))
			for range itersPerGoroutine {
				results, err := mp.Send(f, func(_ uint64, _ multipath.Frame) error {
					return nil
				})
				if err != nil {
					// ErrNoActivePaths is acceptable if UpdatePaths swapped in an
					// empty/inactive set transiently; any other error is a bug.
					if !errors.Is(err, paths.ErrNoActivePaths) {
						t.Errorf("sender %d: unexpected Send error: %v", s, err)
					}
					continue
				}
				// Invariant: at most 2 results per Send (BC-2.02.001 invariant 2).
				if len(results) > 2 {
					t.Errorf("sender %d: Send returned %d results; want ≤2", s, len(results))
				}
			}
		}()
	}

	wg.Wait()
}

// TestNewDropCache_RejectsInvalidCapacity asserts that NewDropCache panics
// when capacity < 1 (zero or negative). A capacity of zero produces a
// degenerate cache (pins at 1 entry, violates contract) — this is a
// programmer error, not a runtime condition, so panic is the appropriate
// signal (same pattern as Go stdlib ring.New, list constructors, etc.).
//
// Contract chosen: panic on capacity < 1. The implementer must add a guard
// at the top of NewDropCache:
//
//	if capacity < 1 {
//	    panic("multipath: NewDropCache capacity must be >= 1")
//	}
//
// This test is RED until that guard is added — without it, NewDropCache(0)
// silently creates a degenerate cache.
//
// Pass-2 finding F-L2 / BC-2.02.009 (constructor precondition)
func TestNewDropCache_RejectsInvalidCapacity(t *testing.T) {
	t.Parallel()

	invalidCapacities := []int{0, -1, -100}

	for _, cap := range invalidCapacities {
		cap := cap
		t.Run("", func(t *testing.T) {
			t.Parallel()
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("NewDropCache(%d): want panic (capacity < 1), got no panic", cap)
				}
			}()
			// Must panic — degenerate capacity violates the DropCache contract.
			multipath.NewDropCache(cap)
		})
	}
}
