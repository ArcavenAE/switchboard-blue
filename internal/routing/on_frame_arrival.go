// Package routing — OnFrameArrival: DropCache wiring + collision-event logging.
//
// OnFrameArrival is the router-side frame-arrival handler that:
//  1. Consults the compound-key DropCache (checksum, arrival_interface_id)
//     from internal/multipath before forwarding (BC-2.02.009 postcondition 1;
//     AC-004).
//  2. Logs EC-005 collision events via an injected Logger when a drop-cache
//     hit occurs on a key that may represent a hash collision rather than a
//     true loop duplicate (BC-2.02.009 EC-005; AC-005).
//
// Logger injection follows the internal/routing.WithLogger pattern (S-2.02).
//
// Architecture constraints:
//   - internal/routing MAY import internal/multipath (one-way; multipath MUST
//     NOT import routing — ARCH-08 position 11; pass-2-bc009-scope ruling).
//   - Drop cache compound key is (checksum, arrival_interface_id), NEVER
//     checksum alone (ARCH-INDEX F-006; BC-2.02.009; AC-004).
//   - The router NEVER parses channel header payload (BC-2.01.005 / VP-015).
package routing

import (
	"container/list"
	"errors"
	"fmt"
	"hash/crc32"
	"sync"

	"github.com/arcavenae/switchboard/internal/multipath"
)

// ErrDropCacheHit is returned by OnFrameArrival when the compound key
// (checksum, arrival_interface_id) is already present in the DropCache,
// indicating a loop duplicate that must be silently discarded.
//
// BC-2.02.009 postcondition 1: "on cache miss: frame is forwarded normally;
// compound key (frame_checksum, arrival_interface_id) added to the drop cache."
// BC-2.02.009 postcondition 2 (implicit): cache hit → silent discard.
var ErrDropCacheHit = errors.New("routing: drop cache hit — frame suppressed as loop duplicate (BC-2.02.009)")

// collisionTrackCap is the maximum number of distinct (checksum, iface) keys
// held in the per-key hit-count LRU. Capped at multipath.DefaultDropCacheSize
// to satisfy AC-005 v1.4-a (CWE-401/400 — bounded memory under distinct-key
// flood, EC-006).
const collisionTrackCap = multipath.DefaultDropCacheSize

// aggregateLogSampleN is the global aggregate emission stride: one log line is
// emitted for every aggregateLogSampleN eligible candidates across all keys.
// With K=20000 distinct-key first hits each producing 1 candidate, this yields
// K/aggregateLogSampleN = 400 lines ≤ max(10, K/50) = 400 (AC-005 v1.4-b).
// For N=1000 same-key hits with per-key sampling producing 10 candidates, this
// yields at most ceil(10/aggregateLogSampleN) = 1 line ≥ 1 (observability OK)
// and ≤ max(2, N/100) = 10 (AC-005 v1.3).
const aggregateLogSampleN = 50

// dropCacheKey is the compound key used for per-key collision-event rate limiting.
type dropCacheKey struct {
	checksum uint32
	iface    InterfaceID
}

// hitCountEntry is stored in the per-key LRU list so that eviction can clean up
// the map entry in O(1).
type hitCountEntry struct {
	key   dropCacheKey
	count uint64
}

// FrameArrivalHandler is the router-level frame-arrival processing path.
// It wires:
//   - drop-cache loop-duplicate suppression (BC-2.02.009, AC-004)
//   - EC-005 collision-event logging via an injected Logger (AC-005)
//
// Zero value is not usable; construct via NewFrameArrivalHandler.
type FrameArrivalHandler struct {
	dropCache *multipath.DropCache
	logger    Logger // injected via WithFrameArrivalLogger; nopLogger if nil

	// hitCountMu guards the per-key hit-count LRU and the aggregate emission
	// counter for concurrent OnFrameArrival calls.
	// Per go.md rule 12: no internal pointer leaks from locked accessors.
	hitCountMu sync.Mutex

	// hitCountIndex maps a dropCacheKey to its list.Element in hitCountLRU.
	// Together with hitCountLRU, this forms a bounded LRU capped at
	// collisionTrackCap entries (AC-005 v1.4-a / EC-006 / CWE-401/400).
	hitCountIndex map[dropCacheKey]*list.Element
	// hitCountLRU is the ordered list (front = most-recently used) of
	// hitCountEntry values. Len() never exceeds collisionTrackCap.
	hitCountLRU *list.List

	// aggregateEmitCount is the global (cross-key) count of log-eligible
	// candidates produced by the per-key sampler. A log line is actually
	// emitted only when aggregateEmitCount%aggregateLogSampleN == 1, bounding
	// total aggregate output to K/aggregateLogSampleN for K distinct first-hits
	// (AC-005 v1.4-b / EC-006 / CWE-779).
	aggregateEmitCount uint64
}

// NewFrameArrivalHandler constructs a FrameArrivalHandler that consults dc
// for drop-cache loop-duplicate suppression (BC-2.02.009).
//
// To enable EC-005 collision-event logging (AC-005), apply
// WithFrameArrivalLogger(l)(h) after construction.
//
// dc must not be nil. Passing nil is a programmer error and panics at
// construction time (consistent with NewDropCache's fail-fast contract).
func NewFrameArrivalHandler(dc *multipath.DropCache) *FrameArrivalHandler {
	if dc == nil {
		panic("routing: NewFrameArrivalHandler dc must not be nil")
	}
	return &FrameArrivalHandler{
		dropCache:     dc,
		logger:        nopLogger{},
		hitCountIndex: make(map[dropCacheKey]*list.Element, collisionTrackCap),
		hitCountLRU:   list.New(),
	}
}

// WithFrameArrivalLogger returns a FrameArrivalHandlerOption that injects l
// as the logger for EC-005 collision-event log lines.
//
// Logger injection follows the internal/routing.WithLogger pattern established
// in S-2.02 (BC-2.02.009 postcondition 2; AC-005).
func WithFrameArrivalLogger(l Logger) func(*FrameArrivalHandler) {
	return func(h *FrameArrivalHandler) {
		h.logger = l
	}
}

// trackedKeyCount returns the number of distinct compound keys currently held
// in the per-key hit-count tracking structure.
//
// This accessor exists as a test seam only (AC-005-a / BC-5.38.001 Red Gate).
// It allows TestBC_2_02_009_Router_CollisionLog_DistinctKeyFlood_Bounded to
// assert that the tracking structure does not exceed its declared cap under a
// distinct-key flood (EC-006 / CWE-401/400).
//
// The bounded LRU ensures the returned value never exceeds collisionTrackCap.
func (h *FrameArrivalHandler) trackedKeyCount() int {
	h.hitCountMu.Lock()
	defer h.hitCountMu.Unlock()
	return h.hitCountLRU.Len()
}

// trackedIndexLen returns the number of entries in the hitCountIndex map.
//
// This accessor exists as a test seam only (CWE-401 parity check).
// TestBC_2_02_009_Router_OnFrameArrival_ConcurrentAccess asserts that
// len(hitCountIndex) == hitCountLRU.Len() after a concurrent flood, verifying
// that eviction removes from BOTH the LRU list and the backing map — a leak in
// either direction is the real CWE-401 risk that trackedKeyCount() alone cannot
// detect.
func (h *FrameArrivalHandler) trackedIndexLen() int {
	h.hitCountMu.Lock()
	defer h.hitCountMu.Unlock()
	return len(h.hitCountIndex)
}

// OnFrameArrival is the router-level end-to-end frame-arrival handler.
//
// It composes DropCache loop-duplicate suppression (BC-2.02.009, AC-004) and
// split-horizon forwarding (BC-2.02.008, AC-001/AC-002) into a single
// frame-arrival path (AC-006 / ARCH-03 §Duplicate-and-Race).
//
// frameBytes is the raw frame (outer header + payload). It is treated as
// opaque — the channel header section is NEVER parsed (BC-2.01.005 / VP-015).
//
// arrivalIface is the logical interface ID the frame arrived on. Together with
// the CRC32 checksum of frameBytes it forms the compound drop-cache key
// (checksum, arrival_interface_id) per ARCH-03 F-006 and BC-2.02.009.
//
// interfaceSet is the set of all interfaces to consider for forwarding.
// fn is the caller-supplied function that writes the frame to a specific output
// interface; it is called by SplitHorizon.Forward on each eligible interface.
//
// On cache miss (AC-006 a): the compound key is added to the DropCache, then
// SplitHorizon.Forward is called — forwarding the frame on all interfaces in
// interfaceSet except arrivalIface (BC-2.02.008 PC-1/PC-2; BC-2.02.009 PC-1).
// If all interfaces are the arrival interface, ErrAllPathsSplitHorizon is returned.
//
// On cache hit (AC-006 b): ErrDropCacheHit is returned, no forwarding occurs.
// A collision-event log line is emitted via the injected logger (AC-005;
// BC-2.02.009 EC-005).
//
// The DropCache hit counter is incremented on every cache hit as required by
// BC-2.02.009 postcondition 2 (operator diagnostics). Increment is performed
// inside DropCache.AddIfAbsent (S-4.01).
func (h *FrameArrivalHandler) OnFrameArrival(
	frameBytes []byte,
	arrivalIface InterfaceID,
	interfaceSet []InterfaceID,
	fn ForwardFunc,
) error {
	// Compute compound drop-cache key: (crc32(frameBytes), arrival_interface_id).
	// frameBytes is treated as opaque — the channel header is NEVER parsed here
	// (BC-2.01.005 / VP-015 / AC-003).
	checksum := crc32.ChecksumIEEE(frameBytes)

	// AddIfAbsent atomically checks and inserts — eliminates TOCTOU window
	// (F-005 / BC-2.02.002 invariant 1). Returns false on hit (duplicate).
	firstArrival := h.dropCache.AddIfAbsent(checksum, uint64(arrivalIface))
	if !firstArrival {
		// Cache hit: loop duplicate (or hash collision).
		//
		// Two-tier rate-limiting strategy (AC-005 v1.4 / EC-005):
		//
		// Tier 1 — per-key sampler: increment the per-key hit count in a
		// bounded LRU (cap: collisionTrackCap). Emit a candidate only when
		// count%100==1 (first hit + every 100th subsequent). This gives
		// ceil(N/100) candidates for N same-key hits (≤ max(2, N/100)).
		//
		// Tier 2 — aggregate limiter: only actually log when the global
		// candidate count is ≡1 (mod aggregateLogSampleN). This bounds
		// aggregate output to K/aggregateLogSampleN for K distinct-key
		// first-hits (AC-005 v1.4-b / CWE-779).
		//
		// Combination correctness:
		//   - Same-key N=1000: 10 candidates → at most 1 log line (≥1 ✓, ≤10 ✓).
		//   - Distinct-key K=20000: 20000 candidates → 400 log lines (≤400 ✓).
		//
		// Mutation goes through the handler under the lock (go.md rule 12).
		key := dropCacheKey{checksum: checksum, iface: arrivalIface}

		h.hitCountMu.Lock()
		count := h.incrementHitCountLocked(key)
		var shouldLog bool
		if count%100 == 1 {
			// Tier-1 candidate: increment the aggregate counter and check tier-2.
			h.aggregateEmitCount++
			shouldLog = h.aggregateEmitCount%aggregateLogSampleN == 1
		}
		h.hitCountMu.Unlock()

		if shouldLog {
			h.logger.Log(fmt.Sprintf(
				"drop cache hit: potential loop duplicate or collision (checksum=0x%08x iface=%d hit=%d) (BC-2.02.009 EC-005)",
				checksum, arrivalIface, count,
			))
		}
		return ErrDropCacheHit
	}

	// Cache miss: forward via split-horizon (AC-006 a).
	// SplitHorizon.Forward excludes arrivalIface from the output set (BC-2.02.008 PC-1)
	// and calls fn for every eligible interface (BC-2.02.008 PC-2).
	sh := SplitHorizon{}
	_, err := sh.Forward(frameBytes, arrivalIface, interfaceSet, fn)
	return err
}

// incrementHitCountLocked increments the hit count for key in the bounded LRU
// and returns the new count. It must be called with hitCountMu held.
//
// The LRU is capped at collisionTrackCap entries. When the cap is reached, the
// least-recently-used entry is evicted before inserting a new key. This bounds
// memory to O(collisionTrackCap) regardless of the number of distinct keys
// observed (AC-005 v1.4-a / EC-006 / CWE-401/400).
func (h *FrameArrivalHandler) incrementHitCountLocked(key dropCacheKey) uint64 {
	if elem, ok := h.hitCountIndex[key]; ok {
		// Key exists: update count in-place and move to front (MRU).
		entry := elem.Value.(hitCountEntry)
		entry.count++
		elem.Value = entry
		h.hitCountLRU.MoveToFront(elem)
		return entry.count
	}

	// New key: evict LRU entry if at capacity.
	if h.hitCountLRU.Len() >= collisionTrackCap {
		oldest := h.hitCountLRU.Back()
		if oldest != nil {
			h.hitCountLRU.Remove(oldest)
			delete(h.hitCountIndex, oldest.Value.(hitCountEntry).key)
		}
	}

	// Insert new entry at front with count=1.
	elem := h.hitCountLRU.PushFront(hitCountEntry{key: key, count: 1})
	h.hitCountIndex[key] = elem
	return 1
}
