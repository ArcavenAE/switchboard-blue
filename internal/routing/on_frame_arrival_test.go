// Tests for FrameArrivalHandler: DropCache wiring and collision-event logging
// (BC-2.02.009 / S-4.04).
//
// Red Gate discipline: every test must FAIL until the implementation in
// on_frame_arrival.go replaces the panic("not implemented") stubs.
//
// VP traces: (none additional beyond BC-2.02.009 unit tests)
// AC traces: AC-004, AC-005 (BC-2.02.009 postconditions 1 and 2)
package routing

import (
	"errors"
	"sync"
	"testing"

	"github.com/arcavenae/switchboard/internal/multipath"
)

// ---- test logger -----------------------------------------------------------

// captureLogger records every Log call for assertion in tests.
type captureLogger struct {
	lines []string
}

func (c *captureLogger) Log(msg string) {
	c.lines = append(c.lines, msg)
}

// ---- AC-004 / BC-2.02.009 postcondition 1 ----------------------------------

// TestBC_2_02_009_Router_DropCacheWiring verifies the compound-key (checksum,
// arrival_interface_id) DropCache wiring in OnFrameArrival:
//   - Cache miss: nil returned, key added.
//   - Cache hit: ErrDropCacheHit returned, frame silently discarded.
//   - Compound key: same checksum on different interface IDs are distinct entries.
//   - Checksum alone is never used as the key (ARCH-INDEX F-006).
//
// AC-004 / BC-2.02.009 postcondition 1:
// "On cache miss: frame is forwarded normally; compound key
// (frame_checksum, arrival_interface_id) added to the drop cache."
func TestBC_2_02_009_Router_DropCacheWiring(t *testing.T) {
	t.Parallel()

	const (
		ifaceA InterfaceID = 1
		ifaceB InterfaceID = 2
	)

	frameBytes := []byte("test-frame-content-deterministic")

	// ifaces used by sub-tests: ifaceA and ifaceB are the arrival interfaces;
	// ifaceOther is a non-arrival interface so SplitHorizon.Forward succeeds on
	// cache-miss paths (required by the 4-param AC-006 signature).
	const ifaceOther InterfaceID = 99
	nopFn := ForwardFunc(func(_ InterfaceID, _ []byte) error { return nil })

	t.Run("cache_miss_first_arrival_returns_nil", func(t *testing.T) {
		t.Parallel()
		// AC-004 / BC-2.02.009 postcondition 1: first arrival on a fresh cache is a miss.
		// No custom logger needed — nopLogger default is sufficient.
		dc := multipath.NewDropCache(multipath.DefaultDropCacheSize)
		h := NewFrameArrivalHandler(dc)

		err := h.OnFrameArrival(frameBytes, ifaceA, []InterfaceID{ifaceA, ifaceOther}, nopFn)
		if err != nil {
			t.Errorf("got err = %v; want nil on cache miss (AC-004 / BC-2.02.009 PC-1)", err)
		}
	})

	t.Run("cache_hit_second_arrival_returns_ErrDropCacheHit", func(t *testing.T) {
		t.Parallel()
		// AC-004 / BC-2.02.009: second arrival of same frame on same interface → cache hit.
		dc := multipath.NewDropCache(multipath.DefaultDropCacheSize)
		h := NewFrameArrivalHandler(dc)

		// First arrival: populate the cache.
		if err := h.OnFrameArrival(frameBytes, ifaceA, []InterfaceID{ifaceA, ifaceOther}, nopFn); err != nil {
			t.Fatalf("first arrival unexpected error: %v", err)
		}
		// Second arrival: must be suppressed.
		err := h.OnFrameArrival(frameBytes, ifaceA, []InterfaceID{ifaceA, ifaceOther}, nopFn)
		if !errors.Is(err, ErrDropCacheHit) {
			t.Errorf("got err = %v; want ErrDropCacheHit on second arrival (AC-004 / BC-2.02.009 PC-1)", err)
		}
	})

	t.Run("compound_key_same_checksum_different_interface_is_miss", func(t *testing.T) {
		t.Parallel()
		// AC-004 / BC-2.02.009: same frame on a DIFFERENT interface is a cache miss.
		// Validates that the key is (checksum, arrival_interface_id), NEVER checksum
		// alone (ARCH-INDEX F-006; story architecture compliance rule).
		dc := multipath.NewDropCache(multipath.DefaultDropCacheSize)
		h := NewFrameArrivalHandler(dc)

		// First: arrive on ifaceA.
		if err := h.OnFrameArrival(frameBytes, ifaceA, []InterfaceID{ifaceA, ifaceOther}, nopFn); err != nil {
			t.Fatalf("first arrival on ifaceA unexpected error: %v", err)
		}
		// Second: same bytes, different interface → must NOT be a hit.
		// Multipath duplicate-and-race requires both copies to survive (ARCH-03 F-006).
		err := h.OnFrameArrival(frameBytes, ifaceB, []InterfaceID{ifaceB, ifaceOther}, nopFn)
		if err != nil {
			t.Errorf("got err = %v on ifaceB; want nil — compound key must not collapse to checksum alone (AC-004 / F-006)", err)
		}
	})

	t.Run("absent_key_gets_added_to_cache", func(t *testing.T) {
		t.Parallel()
		// AC-004: after a cache miss, the compound key is added to the drop cache
		// so the next arrival (same frame, same interface) is suppressed.
		dc := multipath.NewDropCache(multipath.DefaultDropCacheSize)
		h := NewFrameArrivalHandler(dc)

		_ = h.OnFrameArrival(frameBytes, ifaceA, []InterfaceID{ifaceA, ifaceOther}, nopFn) // first arrival (miss, adds key)

		// The key must now be in the cache: second arrival returns ErrDropCacheHit.
		err := h.OnFrameArrival(frameBytes, ifaceA, []InterfaceID{ifaceA, ifaceOther}, nopFn)
		if !errors.Is(err, ErrDropCacheHit) {
			t.Errorf("got err = %v after first arrival; want ErrDropCacheHit — key must be added on miss (AC-004)", err)
		}
	})
}

// ---- EC-003 / BC-2.02.009 postcondition 2 ---------------------------------

// TestBC_2_02_009_Router_DropCacheHitCounterIncremented verifies that the
// DropCache hit counter is incremented on each cache hit (loop duplicate
// suppression).
//
// EC-003: "Same frame arrives on the same interface twice → second silently
// discarded; DropCache hit counter incremented."
// BC-2.02.009 postcondition 2: "drop cache hit counter incremented (for
// operator diagnostics)."
func TestBC_2_02_009_Router_DropCacheHitCounterIncremented(t *testing.T) {
	t.Parallel()

	// EC-003 / BC-2.02.009 PC-2
	dc := multipath.NewDropCache(multipath.DefaultDropCacheSize)
	h := NewFrameArrivalHandler(dc)

	frame := []byte("ec003-frame")
	const iface InterfaceID = 7
	const ifaceOtherHit InterfaceID = 100
	nopFnHit := ForwardFunc(func(_ InterfaceID, _ []byte) error { return nil })

	// First arrival: miss.
	if err := h.OnFrameArrival(frame, iface, []InterfaceID{iface, ifaceOtherHit}, nopFnHit); err != nil {
		t.Fatalf("first arrival unexpected error: %v", err)
	}
	hitsBefore := dc.Hits()

	// Second arrival: hit — counter must increment.
	if err := h.OnFrameArrival(frame, iface, []InterfaceID{iface, ifaceOtherHit}, nopFnHit); !errors.Is(err, ErrDropCacheHit) {
		t.Fatalf("second arrival: got err = %v; want ErrDropCacheHit (EC-003)", err)
	}

	hitsAfter := dc.Hits()
	if hitsAfter <= hitsBefore {
		t.Errorf("Hits() = %d after cache hit; want > %d (EC-003 / BC-2.02.009 PC-2)", hitsAfter, hitsBefore)
	}
}

// ---- AC-005 / BC-2.02.009 postcondition 2 / EC-004 + EC-005 ---------------

// TestBC_2_02_009_Router_CollisionLogRateLimited verifies the rate-limited /
// sampled collision-event logging contract of OnFrameArrival (AC-005 v1.3 /
// BC-2.02.009 postcondition 2 / EC-005).
//
// AC-005 v1.3 requires:
//  1. The FIRST drop-cache hit on a given compound key (checksum, iface) MUST
//     produce at least one log line (observability preserved; operator alerted).
//  2. N rapid identical-key hits MUST produce far fewer than N log lines
//     (bounded; CWE-779 log-spam DoS mitigation; BC-2.02.009 EC-002).
//     Threshold: log_lines <= max(2, N/100).  For N=1000 → at most 10 lines.
//
// EC-004: "Two different frames share a checksum on the same interface →
// legitimate frame incorrectly suppressed; event logged via injected logger
// as potential collision."
func TestBC_2_02_009_Router_CollisionLogRateLimited(t *testing.T) {
	t.Parallel()

	t.Run("first_hit_produces_at_least_one_log_line", func(t *testing.T) {
		t.Parallel()
		// AC-005 v1.3 / BC-2.02.009 PC-2 / EC-005: the first drop-cache hit on
		// a given compound key MUST produce at least one collision-event log line.
		// Operators must be alerted on the first occurrence.
		dc := multipath.NewDropCache(multipath.DefaultDropCacheSize)
		logger := &captureLogger{}
		h := NewFrameArrivalHandler(dc)
		WithFrameArrivalLogger(logger)(h)

		frame := []byte("collision-test-frame")
		const iface InterfaceID = 42
		const ifaceOtherColl InterfaceID = 200
		nopFnColl := ForwardFunc(func(_ InterfaceID, _ []byte) error { return nil })

		// First arrival: miss — logger should NOT be called.
		_ = h.OnFrameArrival(frame, iface, []InterfaceID{iface, ifaceOtherColl}, nopFnColl)
		if len(logger.lines) != 0 {
			t.Errorf("logger called on cache miss; want no log on first arrival (AC-005 v1.3)")
		}

		// Second arrival: first HIT — logger MUST produce at least one line.
		_ = h.OnFrameArrival(frame, iface, []InterfaceID{iface, ifaceOtherColl}, nopFnColl)
		if len(logger.lines) == 0 {
			t.Errorf("logger not called on first cache hit; want at least one collision-event log line (AC-005 v1.3 / BC-2.02.009 PC-2 / EC-005)")
		}
	})

	t.Run("no_logger_injected_does_not_panic", func(t *testing.T) {
		t.Parallel()
		// Defensive: if no logger is injected, OnFrameArrival must not panic.
		// NewFrameArrivalHandler installs a nopLogger by default.
		dc := multipath.NewDropCache(multipath.DefaultDropCacheSize)
		h := NewFrameArrivalHandler(dc) // no WithFrameArrivalLogger

		frame := []byte("nop-logger-frame")
		const iface InterfaceID = 5
		const ifaceOtherNop InterfaceID = 201
		nopFnNop := ForwardFunc(func(_ InterfaceID, _ []byte) error { return nil })

		_ = h.OnFrameArrival(frame, iface, []InterfaceID{iface, ifaceOtherNop}, nopFnNop)    // miss
		err := h.OnFrameArrival(frame, iface, []InterfaceID{iface, ifaceOtherNop}, nopFnNop) // hit — must not panic
		if !errors.Is(err, ErrDropCacheHit) {
			t.Errorf("got err = %v; want ErrDropCacheHit (nopLogger path)", err)
		}
	})

	t.Run("flood_of_hits_produces_bounded_log_lines", func(t *testing.T) {
		t.Parallel()
		// AC-005 v1.3 / EC-005 (flood scenario):
		//
		// N rapid identical-key drop-cache hits (N=1000) MUST produce at most
		// max(2, N/100) = max(2, 10) = 10 log lines.
		//
		// Current implementation logs on every hit (unbounded), so this test
		// MUST FAIL until rate-limiting is implemented (Red Gate / BC-5.38.001).
		//
		// Rationale: unbounded per-hit logging under a routing loop or replayed-
		// frame flood violates CWE-779 (log injection / unbounded log output)
		// and is inconsistent with BC-2.02.009 EC-002's bounded-defense model.
		const N = 1000
		maxLines := func(n int) int {
			// threshold: max(2, N/100) — from AC-005 v1.3
			if n/100 > 2 {
				return n / 100
			}
			return 2
		}(N)

		dc := multipath.NewDropCache(multipath.DefaultDropCacheSize)
		logger := &captureLogger{}
		h := NewFrameArrivalHandler(dc)
		WithFrameArrivalLogger(logger)(h)

		frame := []byte("flood-hit-frame-ec005")
		const iface InterfaceID = 77
		const ifaceOtherFlood InterfaceID = 204
		nopFnFlood := ForwardFunc(func(_ InterfaceID, _ []byte) error { return nil })

		// First arrival: cache miss — populates the drop-cache key.
		if err := h.OnFrameArrival(frame, iface, []InterfaceID{iface, ifaceOtherFlood}, nopFnFlood); err != nil {
			t.Fatalf("first arrival (cache miss) unexpected error: %v", err)
		}

		// N identical-key hits in rapid succession (simulating a routing loop or
		// replayed-frame flood — BC-2.02.009 EC-002 / EC-005).
		for i := 0; i < N; i++ {
			_ = h.OnFrameArrival(frame, iface, []InterfaceID{iface, ifaceOtherFlood}, nopFnFlood)
		}

		got := len(logger.lines)
		if got > maxLines {
			t.Errorf(
				"flood of %d identical-key hits produced %d log lines; want <= %d (max(2, N/100)) — unbounded logging violates AC-005 v1.3 / CWE-779 (EC-005)",
				N, got, maxLines,
			)
		}
		// Also assert that at least one line was emitted (first-hit observability).
		if got == 0 {
			t.Errorf("no log lines produced for %d hits; want at least one (first-hit observability — AC-005 v1.3)", N)
		}
	})
}

// ---- AC-006 / BC-2.02.009 PC-1 + BC-2.02.008 PC-2 — end-to-end composition --

// TestOnFrameArrival_ForwardsViaSplitHorizon_AfterDropCacheMiss verifies that
// OnFrameArrival composes DropCache suppression and split-horizon forwarding
// into a single end-to-end frame-arrival handler (AC-006 / BC-2.02.009 PC-1 +
// BC-2.02.008 PC-2 / ARCH-03 §Duplicate-and-Race).
//
// AC-006 (a) — DropCache MISS: OnFrameArrival adds the compound key
// (checksum, arrival_interface_id) to the DropCache, then calls
// SplitHorizon.Forward(frame, arrival_interface_id, interface_set), and the
// frame is forwarded on all interfaces in the set EXCEPT arrival_interface_id.
//
// AC-006 (b) — DropCache HIT: OnFrameArrival silently discards the frame and
// does NOT invoke SplitHorizon.Forward; no forwarding occurs.
//
// Constraint: SplitHorizon.Forward must have at least one non-test caller
// (OnFrameArrival) — this test enforces that observable requirement.
//
// The test calls OnFrameArrival with the full end-to-end signature:
//
//	OnFrameArrival(frameBytes []byte, arrivalIface InterfaceID,
//	               interfaceSet []InterfaceID, fn ForwardFunc) error
//
// This new 4-parameter signature is what the implementer must add to
// FrameArrivalHandler so that OnFrameArrival can compose the two surfaces
// directly. The existing 2-parameter signature is the pre-AC-006 stub.
func TestOnFrameArrival_ForwardsViaSplitHorizon_AfterDropCacheMiss(t *testing.T) {
	t.Parallel()

	const (
		arrival InterfaceID = 1
		ifaceB  InterfaceID = 2
		ifaceC  InterfaceID = 3
	)

	interfaceSet := []InterfaceID{arrival, ifaceB, ifaceC}
	frameBytes := []byte("ac006-end-to-end-frame-composition")

	// ---- AC-006 (a): DropCache MISS → forwarded on ifaceB and ifaceC only ----
	t.Run("miss_forwards_on_non_arrival_interfaces", func(t *testing.T) {
		t.Parallel()

		dc := multipath.NewDropCache(multipath.DefaultDropCacheSize)
		h := NewFrameArrivalHandler(dc)

		var forwarded []InterfaceID
		fn := func(iface InterfaceID, _ []byte) error {
			forwarded = append(forwarded, iface)
			return nil
		}

		// AC-006 (a): first arrival is a DropCache miss. OnFrameArrival must:
		//   1. Add compound key to DropCache.
		//   2. Call SplitHorizon.Forward → fn called for ifaceB and ifaceC.
		err := h.OnFrameArrival(frameBytes, arrival, interfaceSet, ForwardFunc(fn))
		if err != nil {
			t.Fatalf("OnFrameArrival on miss: got err = %v; want nil (AC-006)", err)
		}

		// Frame must have been forwarded on exactly the 2 non-arrival interfaces.
		if containsIface(forwarded, arrival) {
			t.Errorf("arrival interface %d was forwarded — split-horizon must exclude it (AC-006 / BC-2.02.008 PC-1)", arrival)
		}
		if !containsIface(forwarded, ifaceB) {
			t.Errorf("interface %d was NOT forwarded; want forwarding on all non-arrival ifaces (AC-006 / BC-2.02.008 PC-2)", ifaceB)
		}
		if !containsIface(forwarded, ifaceC) {
			t.Errorf("interface %d was NOT forwarded; want forwarding on all non-arrival ifaces (AC-006 / BC-2.02.008 PC-2)", ifaceC)
		}
		if len(forwarded) != 2 {
			t.Errorf("got %d forwarded interfaces; want exactly 2 (AC-006)", len(forwarded))
		}
	})

	// ---- AC-006 (b): DropCache HIT → no forwarding -------------------------
	t.Run("hit_discards_without_forwarding", func(t *testing.T) {
		t.Parallel()

		dc := multipath.NewDropCache(multipath.DefaultDropCacheSize)
		h := NewFrameArrivalHandler(dc)

		mustNotForward := func(iface InterfaceID, _ []byte) error {
			t.Errorf("ForwardFunc called on iface %d on DropCache HIT — must not forward (AC-006 b)", iface)
			return nil
		}

		// First arrival: miss — populates DropCache. Forwarding is expected here.
		var firstForwarded []InterfaceID
		firstFn := func(iface InterfaceID, _ []byte) error {
			firstForwarded = append(firstForwarded, iface)
			return nil
		}
		if err := h.OnFrameArrival(frameBytes, arrival, interfaceSet, ForwardFunc(firstFn)); err != nil {
			t.Fatalf("first arrival unexpected error: %v", err)
		}

		// Second arrival: HIT — OnFrameArrival must NOT call ForwardFunc.
		err := h.OnFrameArrival(frameBytes, arrival, interfaceSet, ForwardFunc(mustNotForward))
		if !errors.Is(err, ErrDropCacheHit) {
			t.Errorf("got err = %v; want ErrDropCacheHit on second arrival (AC-006 b)", err)
		}
	})
}

// ---- AC-005-a/b / EC-006 — distinct-key flood: bounded memory + bounded log --

// TestBC_2_02_009_Router_CollisionLog_DistinctKeyFlood_Bounded verifies the
// two security-critical bounds introduced by AC-005 v1.4 to close CWE-401/400
// (unbounded tracking structure) and CWE-779 (aggregate log-spam DoS) under an
// attacker-controlled distinct-key flood (EC-006).
//
// Sub-requirement (a): BOUNDED MEMORY — the per-key collision-tracking structure
// MUST be capped to at most the DropCache capacity (DefaultDropCacheSize = 10,000)
// or a fixed implementation-chosen maximum. With K=20,000 distinct keys the
// current unbounded hitCounts map grows to K entries (>10,000), violating the
// cap. A correct bounded implementation stays at or below its declared cap.
//
// Sub-requirement (b): BOUNDED AGGREGATE LOG — N distinct-key collisions MUST
// produce far fewer than N aggregate log lines. Threshold: max(10, K/50). With
// K=20,000 the threshold is max(10, 400) = 400. The current implementation logs
// on every first hit (count%100==1 is true when count==1), so it emits K≈20,000
// lines — far above 400. This is the primary RED.
//
// AC-005 v1.4 / BC-2.02.009 EC-002 / EC-006 / CWE-401 / CWE-779.
func TestBC_2_02_009_Router_CollisionLog_DistinctKeyFlood_Bounded(t *testing.T) {
	// NOT t.Parallel() — this test mutates a shared handler in a tight loop;
	// parallelism with other tests that share the package-level state would
	// produce false results. Running serially keeps the assertion deterministic.

	const K = 20_000

	maxAggregateLogLines := func(k int) int {
		// AC-005 v1.4 threshold: max(10, K/50).
		if k/50 > 10 {
			return k / 50
		}
		return 10
	}(K) // = max(10, 400) = 400 for K=20_000

	dc := multipath.NewDropCache(multipath.DefaultDropCacheSize)
	logger := &captureLogger{}
	h := NewFrameArrivalHandler(dc)
	WithFrameArrivalLogger(logger)(h)

	const baseIface InterfaceID = 1
	const egressIface InterfaceID = 2
	nopFn := ForwardFunc(func(_ InterfaceID, _ []byte) error { return nil })

	// Generate K distinct compound keys: vary the frame bytes so each key gets a
	// unique CRC32 checksum. Each distinct frame produces a unique
	// (checksum, baseIface) compound key.
	//
	// Sequence per key:
	//   1. First arrival (cache miss): populates the DropCache entry.
	//   2. Second arrival (cache hit): triggers the collision-logging path.
	//
	// This exercises the distinct-key flood scenario from EC-006.
	for i := range K {
		// Construct frame bytes that produce a distinct (checksum, iface) key.
		// We embed i directly in the frame payload; CRC32 of distinct payloads
		// will differ with overwhelming probability for all i in [0, K).
		frame := make([]byte, 8)
		frame[0] = byte(i)
		frame[1] = byte(i >> 8)
		frame[2] = byte(i >> 16)
		frame[3] = byte(i >> 24)
		frame[4] = 0xAB
		frame[5] = 0xCD
		frame[6] = 0xEF
		frame[7] = 0x01

		// First arrival on baseIface: cache miss — populates the DropCache key.
		// Ignore error: it may be nil (miss) or ErrDropCacheHit (if checksum
		// happened to collide with a prior key — vanishingly unlikely but harmless
		// for this test's purpose since we are measuring aggregate log output).
		_ = h.OnFrameArrival(frame, baseIface, []InterfaceID{baseIface, egressIface}, nopFn)

		// Second arrival on baseIface: cache hit — triggers collision logging.
		_ = h.OnFrameArrival(frame, baseIface, []InterfaceID{baseIface, egressIface}, nopFn)
	}

	// --- Assert (b): aggregate log bound (primary RED) ---
	//
	// The current implementation logs on count%100==1, which is true when
	// count==1 (first hit per key). With K distinct keys each generating one
	// first-hit log line, the current code emits ~K lines >> maxAggregateLogLines.
	// A correct implementation applies a global rate-limit/token-bucket/sampling
	// budget so that aggregate output stays <= max(10, K/50).
	gotLines := len(logger.lines)
	if gotLines > maxAggregateLogLines {
		t.Errorf(
			"distinct-key flood (K=%d): got %d aggregate collision-log lines; want <= %d (max(10, K/50)) — unbounded aggregate logging violates AC-005 v1.4-b / CWE-779 (EC-006)",
			K, gotLines, maxAggregateLogLines,
		)
	}

	// --- Assert (a): tracking-structure memory bound ---
	//
	// trackedKeyCount() returns len(hitCounts) — currently unbounded. With K
	// distinct keys, the map grows to ~K entries. A correct bounded implementation
	// caps the structure at <= DefaultDropCacheSize (10,000) with LRU or
	// equivalent eviction (AC-005 v1.4-a / EC-006).
	//
	// K=20,000 > DefaultDropCacheSize=10,000 so the current unbounded map
	// exceeds the cap, producing a RED on this assertion once (b) is fixed but
	// (a) is not.
	trackedKeys := h.trackedKeyCount()
	if trackedKeys > multipath.DefaultDropCacheSize {
		t.Errorf(
			"distinct-key flood (K=%d): tracking structure holds %d keys; want <= %d (DropCache cap) — unbounded map violates AC-005 v1.4-a / CWE-401/400 (EC-006)",
			K, trackedKeys, multipath.DefaultDropCacheSize,
		)
	}
}

// ---- F-CONC-001 / BC-2.02.009 — concurrent OnFrameArrival race guard -------

// concurrentCaptureLogger is a race-safe logger for concurrent tests.
// captureLogger must NOT be shared across goroutines (its slice has no lock);
// this variant guards every append with a mutex so the test harness itself is
// race-free (a data race in the harness would be a false positive for the
// implementation under review).
type concurrentCaptureLogger struct {
	mu    sync.Mutex
	count int
}

func (c *concurrentCaptureLogger) Log(_ string) {
	c.mu.Lock()
	c.count++
	c.mu.Unlock()
}

func (c *concurrentCaptureLogger) logCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.count
}

// TestBC_2_02_009_Router_OnFrameArrival_ConcurrentAccess is a combined
// race-guard and security-property test (F-CONC-001 / AC-005-a/b / BC-2.02.009).
//
// It resolves the M-1 vs Pass-2 dispute empirically: M-1 (pass-3 security lens)
// claims that under a combined same-key + >cap distinct-key flood, per-key
// hit-count eviction from hitCountLRU resets count=1 and re-arms the
// count%100==1 per-key candidate, potentially unbounding aggregate log output.
// Pass-2 (security lens) ruled this BENIGN because the global tier-2 limiter
// (aggregateEmitCount%aggregateLogSampleN==1) still bounds TOTAL output
// sublinearly (AC-005-b / CWE-779 holds).
//
// This test exercises BOTH flood paths simultaneously under concurrency:
//   - Distinct-key flood: 25,600 distinct compound keys (16 goroutines × 1,600
//     distinct seeds) — well above collisionTrackCap (=DefaultDropCacheSize=10,000),
//     so hitCountLRU eviction fires heavily.
//   - Same-key hammering: 4 pre-seeded keys, each hit thousands of times across
//     goroutines — exercises MoveToFront under lock contention and the
//     count%100==1 per-key re-arm path.
//
// Combined: ~51,200 total cache-hit calls (16 goroutines × 3,200 calls each,
// half same-key, half distinct).
//
// DropCache sizing: the DropCache is constructed with capacity dropCacheCapacity
// (= 30,000), which is large enough to hold all 25,604 seeded distinct keys
// (25,600 goroutine-distinct + 4 same-key). This ensures that every key seeded
// in the pre-seed phase stays RESIDENT in the DropCache throughout the storm, so
// every storm call is a genuine DropCache HIT — driving the hit-count LRU on
// every call. Without this, seeded keys fall off the DropCache's own LRU (which
// has DefaultDropCacheSize=10,000 << 25,604), so later storm calls become MISSES
// that bypass the hit-count LRU entirely and the eviction path is never reached.
// collisionTrackCap (the hit-count LRU cap) is hardwired to
// multipath.DefaultDropCacheSize in production; only the DropCache capacity in
// the test is changed here so all hits land.
//
// Invariants asserted (all must hold — do NOT weaken bounds to force green):
//
//	(a) EVICTION FIRED: after the storm, trackedKeyCount() == collisionTrackCap
//	    exactly. With 25,600 distinct keys all resident in the DropCache, every
//	    storm call drives the hit-count LRU; 25,600 > cap=10,000 guarantees
//	    eviction fired and pinned the LRU at its cap. A value < cap means eviction
//	    never triggered (the path we are here to fence). Do NOT relax to <=.
//
//	(b) MAP/LRU PARITY: after the storm, len(hitCountIndex) == hitCountLRU.Len()
//	    and both values <= collisionTrackCap. Catches a leak where eviction removes
//	    from one structure but not the other (the real CWE-401 risk).
//
//	(c) BOUNDED AGGREGATE LOG: total log lines emitted across all goroutines MUST
//	    be far below total cache-hit calls. Bound: logLines <= totalHits/aggregateLogSampleN + safetyMargin.
//	    Arithmetic: totalHits ≈ 51,200; aggregateLogSampleN=50;
//	    totalHits/aggregateLogSampleN = 1,024; safetyMargin = 500 (generous, accounts
//	    for nondeterministic scheduling). Bound = 1,524. This is strongly sublinear
//	    relative to totalHits (1,524/51,200 < 3%). If this assertion FAILS, M-1 is
//	    a confirmed defect — report numbers, do NOT relax the bound.
//
//	(d) RACE-FREE: no data race reported by the race detector (-race flag).
//
// Exact log-line counts are NOT asserted — they are nondeterministic under
// concurrent scheduling. Only the bound is checked.
func TestBC_2_02_009_Router_OnFrameArrival_ConcurrentAccess(t *testing.T) {
	// NOT t.Parallel() — this test is CPU-intensive and uses many goroutines;
	// running concurrently with the distinct-key flood test would produce
	// misleading timing. Serial execution keeps assertions deterministic.

	const (
		numGoroutines = 16

		// distinctPerGoroutine: each goroutine generates this many unique keys.
		// 16 × 1,600 = 25,600 distinct keys — 2.56× collisionTrackCap (=10,000),
		// so hitCountLRU and hitCountIndex eviction fire heavily under contention.
		distinctPerGoroutine = 1600

		// sameKeyHitsPerGoroutine: each goroutine hits each pre-seeded same-key
		// frame this many times (exercises MoveToFront + count%100==1 re-arm).
		sameKeyHitsPerGoroutine = 400

		// Number of shared same-key frames (each hit sameKeyHitsPerGoroutine × numGoroutines times).
		numSameKeyFrames = 4

		// dropCacheCapacity: large enough to keep ALL seeded keys resident so every
		// storm call is a DropCache HIT. Total distinct keys seeded:
		//   numGoroutines × distinctPerGoroutine + numSameKeyFrames = 25,604.
		// 30,000 provides headroom above that without being excessively large.
		// NOTE: collisionTrackCap (the hit-count LRU cap) stays at
		// multipath.DefaultDropCacheSize = 10,000 per production wiring; only this
		// test-local DropCache capacity is enlarged.
		dropCacheCapacity = 30_000
	)

	// totalCacheHitCalls: all calls after the seed that land as drop-cache hits.
	// Each goroutine: distinctPerGoroutine distinct-key hits (all seeded and kept
	// resident by the oversized DropCache) + numSameKeyFrames*sameKeyHitsPerGoroutine
	// same-key hits.
	// Across all goroutines: 16 × (1,600 + 4×400) = 16 × 3,200 = 51,200.
	const totalCacheHitCalls = numGoroutines * (distinctPerGoroutine + numSameKeyFrames*sameKeyHitsPerGoroutine)

	// aggregateBound: AC-005-b security property.
	// logLines <= totalCacheHitCalls/aggregateLogSampleN + safetyMargin.
	// = 51,200/50 + 500 = 1,024 + 500 = 1,524.
	// This is strongly sublinear: 1,524 / 51,200 < 3%.
	// If this bound fails, M-1 is a confirmed defect.
	const aggregateBound = totalCacheHitCalls/aggregateLogSampleN + 500

	dc := multipath.NewDropCache(dropCacheCapacity)
	logger := &concurrentCaptureLogger{}
	h := NewFrameArrivalHandler(dc)
	WithFrameArrivalLogger(logger)(h)

	const arrivalIface InterfaceID = 1
	const egressIface InterfaceID = 2
	interfaceSet := []InterfaceID{arrivalIface, egressIface}
	nopFn := ForwardFunc(func(_ InterfaceID, _ []byte) error { return nil })

	buildFrame := func(seed int) []byte {
		f := make([]byte, 8)
		f[0] = byte(seed)
		f[1] = byte(seed >> 8)
		f[2] = byte(seed >> 16)
		f[3] = byte(seed >> 24)
		f[4] = 0xCA
		f[5] = 0xFE
		f[6] = 0xBA
		f[7] = 0xBE
		return f
	}

	// sameKeyFrames: 4 frames pre-seeded into the drop cache so ALL goroutine
	// hits on these frames are cache hits (exercises MoveToFront + per-key count
	// re-arm under high contention across 16 goroutines).
	sameKeyFrames := make([][]byte, numSameKeyFrames)
	for i := range sameKeyFrames {
		sameKeyFrames[i] = buildFrame(0xDEAD_0000 + i)
		_ = h.OnFrameArrival(sameKeyFrames[i], arrivalIface, interfaceSet, nopFn) // seed: cache miss, adds key
	}

	// Pre-seed the drop cache for ALL distinct keys each goroutine will use.
	// The DropCache capacity (dropCacheCapacity=30,000) exceeds total distinct keys
	// (25,604), so every seeded key stays resident. During the storm every call
	// is therefore a DropCache HIT — routing execution into the hit-count LRU
	// path (on_frame_arrival.go incrementHitCountLocked). Without the oversized
	// DropCache, the DropCache's own LRU would evict earlier seeds (its default
	// cap is only 10,000), turning later storm calls into MISSES that bypass the
	// hit-count LRU entirely and leave eviction untested.
	for g := range numGoroutines {
		for i := range distinctPerGoroutine {
			seed := (g+1)*0x10000 + i
			frame := buildFrame(seed)
			_ = h.OnFrameArrival(frame, arrivalIface, interfaceSet, nopFn) // seed: cache miss
		}
	}

	// Storm phase: all goroutines hammer simultaneously.
	// Each goroutine interleaves:
	//   - sameKeyHitsPerGoroutine hits per same-key frame (×numSameKeyFrames)
	//   - distinctPerGoroutine distinct-key cache hits
	// Because all keys are resident in the DropCache, all calls are HITs that
	// drive the hit-count LRU. 25,600 distinct keys > collisionTrackCap=10,000,
	// so the hit-count LRU evicts 15,600 entries and pins at exactly cap.
	// This exercises the evict-from-both-structures path (map + list) under
	// heavy lock contention — the real CWE-401 risk being fenced.
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := range numGoroutines {
		go func(goroutineID int) {
			defer wg.Done()

			// Distinct-key hits for this goroutine (all resident in oversized DropCache).
			for i := range distinctPerGoroutine {
				seed := (goroutineID+1)*0x10000 + i
				frame := buildFrame(seed)
				_ = h.OnFrameArrival(frame, arrivalIface, interfaceSet, nopFn)
			}

			// Same-key hammering: each of the numSameKeyFrames frames hit
			// sameKeyHitsPerGoroutine times, interleaved to maximise contention.
			for i := range sameKeyHitsPerGoroutine {
				frame := sameKeyFrames[i%numSameKeyFrames]
				_ = h.OnFrameArrival(frame, arrivalIface, interfaceSet, nopFn)
			}
		}(g)
	}

	wg.Wait()

	lruLen := h.trackedKeyCount()
	indexLen := h.trackedIndexLen()

	// --- Assert (a): EVICTION FIRED (F-CONC-M1 / CWE-401 evict-from-both-structures) ---
	//
	// 25,600 distinct keys were seeded and all stayed resident in the DropCache
	// (capacity=30,000). Every storm call was therefore a DropCache HIT driving
	// the hit-count LRU. Since 25,600 > collisionTrackCap=10,000, the LRU MUST
	// have evicted 15,600 entries and be pinned at exactly collisionTrackCap.
	// A value < collisionTrackCap means eviction never triggered — the
	// evict-from-both-structures path (the real CWE-401 risk) was not exercised.
	// Do NOT change == to <= here; a strict equality check is the whole point.
	if lruLen != collisionTrackCap {
		t.Errorf(
			"EVICTION NOT FIRED: trackedKeyCount()=%d != collisionTrackCap=%d — "+
				"hit-count LRU eviction path was not reached (F-CONC-M1 / CWE-401); "+
				"DropCache capacity=%d, distinct keys seeded=%d; "+
				"if lruLen < cap, eviction never triggered (check DropCache resident count); "+
				"if lruLen > cap, the cap invariant is broken",
			lruLen, collisionTrackCap,
			dropCacheCapacity, numGoroutines*distinctPerGoroutine+numSameKeyFrames,
		)
	}

	// --- Assert (b): MAP/LRU PARITY (CWE-401 leak detection) ---
	//
	// Both structures must agree on the count of tracked keys, and neither may
	// exceed collisionTrackCap. A divergence means eviction removed from the
	// LRU list without cleaning the map (memory leak) or vice versa (dangling
	// pointer into freed list element).
	if lruLen != indexLen {
		t.Errorf(
			"MAP/LRU PARITY VIOLATED: hitCountLRU.Len()=%d != len(hitCountIndex)=%d — eviction leaked one structure (CWE-401 / F-CONC-001)",
			lruLen, indexLen,
		)
	}
	if lruLen > collisionTrackCap {
		t.Errorf(
			"LRU bound violated: hitCountLRU.Len()=%d > collisionTrackCap=%d (AC-005 v1.4-a / EC-006 / CWE-401/400)",
			lruLen, collisionTrackCap,
		)
	}
	if indexLen > collisionTrackCap {
		t.Errorf(
			"index bound violated: len(hitCountIndex)=%d > collisionTrackCap=%d (AC-005 v1.4-a / EC-006 / CWE-401/400)",
			indexLen, collisionTrackCap,
		)
	}

	// --- Assert (c): BOUNDED AGGREGATE LOG (AC-005-b / CWE-779 / M-1 empirical test) ---
	//
	// Arithmetic (see const declarations above):
	//   totalCacheHitCalls = 51,200
	//   aggregateLogSampleN = 50 (from on_frame_arrival.go)
	//   upper bound = totalCacheHitCalls/aggregateLogSampleN + safetyMargin
	//               = 1,024 + 500 = 1,524
	//
	// If the two-tier rate limiter is correct (Pass-2 / M-1-benign), logLines
	// will be on the order of 1,000–1,024 (the aggregate counter fires once per
	// aggregateLogSampleN candidates). If M-1 is a real defect (per-key eviction
	// resets count=1, re-arms tier-1 candidate unconditionally, bypassing tier-2
	// throttling), logLines will approach totalCacheHitCalls/100 ≈ 512 per
	// re-arm cycle — still below this bound in the single-pass case, but the
	// aggregate counter's monotone growth means tier-2 will NOT fire on every
	// re-arm. The bound is generous enough that a benign implementation passes
	// with margin, while a broken tier-2 (e.g. reset aggregateEmitCount on
	// eviction) would blow through it.
	//
	// Do NOT relax this bound to force green. If it fails, report:
	//   totalCacheHitCalls, logLines, aggregateBound — and route to implementer.
	logLines := logger.logCount()
	if logLines > aggregateBound {
		t.Errorf(
			"AGGREGATE LOG BOUND VIOLATED (AC-005-b / CWE-779 / M-1 confirmed defect): "+
				"totalCacheHitCalls=%d logLines=%d aggregateBound=%d "+
				"(bound = totalHits/%d + 500 = %d/%d + 500) — "+
				"aggregate log output is NOT sublinear; M-1 is a real bug",
			totalCacheHitCalls, logLines, aggregateBound,
			aggregateLogSampleN, totalCacheHitCalls, aggregateLogSampleN,
		)
	}

	t.Logf(
		"concurrent flood complete: lruLen=%d indexLen=%d logLines=%d totalCacheHitCalls=%d aggregateBound=%d collisionTrackCap=%d dropCacheCapacity=%d evictionFired=%v",
		lruLen, indexLen, logLines, totalCacheHitCalls, aggregateBound, collisionTrackCap, dropCacheCapacity, lruLen == collisionTrackCap,
	)
}

// ---- constructor nil-guard -------------------------------------------------

// TestNewFrameArrivalHandler_NilDropCachePanics verifies that constructing a
// FrameArrivalHandler with a nil DropCache panics at construction time, not
// deferred to the first OnFrameArrival call (CWE-476 / go.md "no panics in
// library code" adjacency — programmer-precondition panic is appropriate at
// wiring time, mirroring NewDropCache's fail-fast contract).
func TestNewFrameArrivalHandler_NilDropCachePanics(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("NewFrameArrivalHandler(nil) did not panic; want panic at construction time")
		}
	}()

	NewFrameArrivalHandler(nil)
}

// ---- EC-004 / BC-2.02.009 EC-005 (hash collision scenario) ----------------

// TestBC_2_02_009_Router_HashCollisionLogged verifies that when a hash collision
// occurs (two different frames produce the same checksum on the same interface),
// the legitimate second frame is suppressed and a collision event is logged.
//
// EC-004: "Two different frames share a checksum on the same interface →
// legitimate frame incorrectly suppressed; event logged via injected logger
// as potential collision."
//
// Note: we cannot guarantee a real CRC32 collision in a unit test, so this
// test exercises the same observable behavior: any drop-cache hit triggers
// a collision-event log line, because the router cannot distinguish a true
// loop duplicate from a collision — it just logs and suppresses (BC-2.02.009
// EC-005).
func TestBC_2_02_009_Router_HashCollisionLogged(t *testing.T) {
	t.Parallel()

	// EC-004: simulate the observable: once key (checksum, iface) is in cache,
	// any subsequent arrival on the same (checksum, iface) is logged as a
	// potential collision regardless of actual content.
	dc := multipath.NewDropCache(multipath.DefaultDropCacheSize)
	logger := &captureLogger{}
	h := NewFrameArrivalHandler(dc)
	WithFrameArrivalLogger(logger)(h)

	frameFirst := []byte("ec004-first-frame")
	const iface InterfaceID = 11
	const ifaceOtherEC InterfaceID = 203
	nopFnEC := ForwardFunc(func(_ InterfaceID, _ []byte) error { return nil })

	// Populate cache with first frame.
	if err := h.OnFrameArrival(frameFirst, iface, []InterfaceID{iface, ifaceOtherEC}, nopFnEC); err != nil {
		t.Fatalf("first arrival unexpected error: %v", err)
	}

	// A "different frame" that happens to share the same cache key arrives.
	// From the router's perspective: cache hit → suppress + log.
	// We use the same bytes here because we cannot force a real CRC32 collision,
	// but the behavioral contract says ANY cache hit is logged (EC-005).
	err := h.OnFrameArrival(frameFirst, iface, []InterfaceID{iface, ifaceOtherEC}, nopFnEC)
	if !errors.Is(err, ErrDropCacheHit) {
		t.Errorf("got err = %v; want ErrDropCacheHit (EC-004)", err)
	}
	if len(logger.lines) == 0 {
		t.Errorf("no collision-event log line; want logger called on drop-cache hit (EC-004 / EC-005 / AC-005)")
	}
}
