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

	t.Run("cache_miss_first_arrival_returns_nil", func(t *testing.T) {
		t.Parallel()
		// AC-004 / BC-2.02.009 postcondition 1: first arrival on a fresh cache is a miss.
		// No custom logger needed — nopLogger default is sufficient.
		dc := multipath.NewDropCache(multipath.DefaultDropCacheSize)
		h := NewFrameArrivalHandler(dc)

		err := h.OnFrameArrival(frameBytes, ifaceA)
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
		if err := h.OnFrameArrival(frameBytes, ifaceA); err != nil {
			t.Fatalf("first arrival unexpected error: %v", err)
		}
		// Second arrival: must be suppressed.
		err := h.OnFrameArrival(frameBytes, ifaceA)
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
		if err := h.OnFrameArrival(frameBytes, ifaceA); err != nil {
			t.Fatalf("first arrival on ifaceA unexpected error: %v", err)
		}
		// Second: same bytes, different interface → must NOT be a hit.
		// Multipath duplicate-and-race requires both copies to survive (ARCH-03 F-006).
		err := h.OnFrameArrival(frameBytes, ifaceB)
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

		_ = h.OnFrameArrival(frameBytes, ifaceA) // first arrival (miss, adds key)

		// The key must now be in the cache: second arrival returns ErrDropCacheHit.
		err := h.OnFrameArrival(frameBytes, ifaceA)
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

	// First arrival: miss.
	if err := h.OnFrameArrival(frame, iface); err != nil {
		t.Fatalf("first arrival unexpected error: %v", err)
	}
	hitsBefore := dc.Hits()

	// Second arrival: hit — counter must increment.
	if err := h.OnFrameArrival(frame, iface); !errors.Is(err, ErrDropCacheHit) {
		t.Fatalf("second arrival: got err = %v; want ErrDropCacheHit (EC-003)", err)
	}

	hitsAfter := dc.Hits()
	if hitsAfter <= hitsBefore {
		t.Errorf("Hits() = %d after cache hit; want > %d (EC-003 / BC-2.02.009 PC-2)", hitsAfter, hitsBefore)
	}
}

// ---- AC-005 / BC-2.02.009 postcondition 2 / EC-004 + EC-005 ---------------

// TestBC_2_02_009_Router_CollisionEventLogged verifies that an injected Logger
// receives a log line when a drop-cache hit occurs (potential collision event).
//
// AC-005 / BC-2.02.009 postcondition 2 / EC-005:
// "EC-005 collision-event logging: the router's OnFrameArrival path injects
// a logger so that drop-cache hits are logged as potential collision events
// for investigation."
//
// EC-004: "Two different frames share a checksum on the same interface
// (hash collision) → legitimate frame incorrectly suppressed; event logged
// via injected logger as potential collision."
func TestBC_2_02_009_Router_CollisionEventLogged(t *testing.T) {
	t.Parallel()

	t.Run("logger_receives_line_on_cache_hit", func(t *testing.T) {
		t.Parallel()
		// AC-005 / BC-2.02.009 PC-2 / EC-005: injected logger must be called
		// when a drop-cache hit is observed.
		dc := multipath.NewDropCache(multipath.DefaultDropCacheSize)
		logger := &captureLogger{}
		// SA4006 false positive: WithFrameArrivalLogger panics in the Red Gate stub,
		// making h.OnFrameArrival appear unreachable to staticcheck. h IS used.
		h := NewFrameArrivalHandler(dc) //nolint:staticcheck // SA4006: Red Gate stub
		WithFrameArrivalLogger(logger)(h)

		frame := []byte("collision-test-frame")
		const iface InterfaceID = 42

		// First arrival: miss — logger should NOT be called.
		_ = h.OnFrameArrival(frame, iface)
		if len(logger.lines) != 0 {
			t.Errorf("logger called on cache miss; want no log on first arrival (AC-005)")
		}

		// Second arrival: hit — logger MUST be called.
		_ = h.OnFrameArrival(frame, iface)
		if len(logger.lines) == 0 {
			t.Errorf("logger not called on cache hit; want collision-event log line (AC-005 / BC-2.02.009 PC-2 / EC-005)")
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

		_ = h.OnFrameArrival(frame, iface)    // miss
		err := h.OnFrameArrival(frame, iface) // hit — must not panic
		if !errors.Is(err, ErrDropCacheHit) {
			t.Errorf("got err = %v; want ErrDropCacheHit (nopLogger path)", err)
		}
	})

	t.Run("multiple_hits_log_each_time", func(t *testing.T) {
		t.Parallel()
		// AC-005: every cache hit emits a log line — not just the first.
		dc := multipath.NewDropCache(multipath.DefaultDropCacheSize)
		logger := &captureLogger{}
		// SA4006 false positive: same as logger_receives_line_on_cache_hit subtest.
		h := NewFrameArrivalHandler(dc) //nolint:staticcheck // SA4006: Red Gate stub
		WithFrameArrivalLogger(logger)(h)

		frame := []byte("repeated-hit-frame")
		const iface InterfaceID = 99

		_ = h.OnFrameArrival(frame, iface) // first: miss
		_ = h.OnFrameArrival(frame, iface) // second: hit → log
		_ = h.OnFrameArrival(frame, iface) // third: hit → log

		if len(logger.lines) < 2 {
			t.Errorf("got %d log lines; want ≥ 2 (one per hit) (AC-005)", len(logger.lines))
		}
	})
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
	// SA4006 false positive: same pattern as AC-005 collision tests.
	h := NewFrameArrivalHandler(dc) //nolint:staticcheck // SA4006: Red Gate stub
	WithFrameArrivalLogger(logger)(h)

	frameFirst := []byte("ec004-first-frame")
	const iface InterfaceID = 11

	// Populate cache with first frame.
	if err := h.OnFrameArrival(frameFirst, iface); err != nil {
		t.Fatalf("first arrival unexpected error: %v", err)
	}

	// A "different frame" that happens to share the same cache key arrives.
	// From the router's perspective: cache hit → suppress + log.
	// We use the same bytes here because we cannot force a real CRC32 collision,
	// but the behavioral contract says ANY cache hit is logged (EC-005).
	err := h.OnFrameArrival(frameFirst, iface)
	if !errors.Is(err, ErrDropCacheHit) {
		t.Errorf("got err = %v; want ErrDropCacheHit (EC-004)", err)
	}
	if len(logger.lines) == 0 {
		t.Errorf("no collision-event log line; want logger called on drop-cache hit (EC-004 / EC-005 / AC-005)")
	}
}
