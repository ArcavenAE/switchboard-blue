// Package multipath implements duplicate-and-race dispatch (BC-2.02.001) and
// receiver-side frame deduplication via a bounded LRU drop cache (BC-2.02.002,
// BC-2.02.009) for the Switchboard routing engine.
//
// Frames are sent simultaneously on the two highest-scoring paths. The receiver
// delivers the first-arriving copy and silently discards subsequent copies that
// share the same (checksum, arrival_interface_id) compound key (ARCH-03 F-006,
// BC-2.02.009).
//
// This package is pure-core: it performs no network I/O. Path I/O is owned by
// the caller; this package only makes dispatch decisions and manages the drop
// cache.
package multipath

import (
	"container/list"
	"errors"
	"fmt"
	"hash/crc32"
	"sync"

	"github.com/arcavenae/switchboard/internal/paths"
)

// DefaultDropCacheSize is the maximum number of (checksum, interface_id) entries
// in the DropCache before LRU eviction begins (ARCH-03 default: 10,000).
const DefaultDropCacheSize = 10_000

// ErrDuplicate is returned by Multipath.Receive when the arriving frame's
// compound key (checksum, arrival_interface_id) matches an entry in the drop
// cache, indicating a duplicate that must be silently discarded (BC-2.02.002
// postcondition 2, AC-004, AC-005).
var ErrDuplicate = errors.New("multipath: duplicate frame")

// Frame is the minimal frame representation consumed by this package.
// The caller is responsible for computing the outer header bytes used for
// checksum calculation (ARCH-03: crc32(outer_header || payload)).
type Frame struct {
	// OuterHeader is the encoded 44-byte outer header of the frame.
	OuterHeader [44]byte
	// Payload is the variable-length payload that follows the outer header.
	Payload []byte
}

// dropKey is the compound cache key for the LRU drop cache (ARCH-03 F-006).
type dropKey struct {
	checksum           uint32
	arrivalInterfaceID uint64
}

// dropEntry is stored in the LRU list alongside its key so eviction can
// clean up the map entry in O(1).
type dropEntry struct {
	key dropKey
}

// DropCache is a bounded LRU cache of (frame_checksum, arrival_interface_id)
// compound keys used to detect and suppress loop-duplicate frames
// (BC-2.02.009). Checksum lookup is O(1); capacity is enforced by LRU eviction.
//
// Zero value is not usable; construct via NewDropCache.
type DropCache struct {
	mu       sync.Mutex
	capacity int
	index    map[dropKey]*list.Element
	lru      *list.List // front = most-recently used
}

// NewDropCache constructs a DropCache with the given maximum capacity.
// capacity must be ≥ 1; a typical value is DefaultDropCacheSize (10,000).
func NewDropCache(capacity int) *DropCache {
	return &DropCache{
		capacity: capacity,
		index:    make(map[dropKey]*list.Element, capacity),
		lru:      list.New(),
	}
}

// Contains reports whether the compound key (checksum, arrivalInterfaceID) is
// present in the cache. A hit means the frame is a loop duplicate and should
// be silently discarded (BC-2.02.009 postcondition 2).
func (c *DropCache) Contains(checksum uint32, arrivalInterfaceID uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.index[dropKey{checksum, arrivalInterfaceID}]
	return ok
}

// Add inserts the compound key (checksum, arrivalInterfaceID) into the cache.
// If the cache is already at capacity, the least-recently-used entry is evicted
// before insertion (BC-2.02.009 postcondition 3, AC-006).
func (c *DropCache) Add(checksum uint32, arrivalInterfaceID uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := dropKey{checksum, arrivalInterfaceID}

	// If already present, move to front (most-recently used) and return.
	if elem, ok := c.index[key]; ok {
		c.lru.MoveToFront(elem)
		return
	}

	// Evict LRU entry if at capacity.
	if c.lru.Len() >= c.capacity {
		oldest := c.lru.Back()
		if oldest != nil {
			c.lru.Remove(oldest)
			delete(c.index, oldest.Value.(dropEntry).key)
		}
	}

	// Insert new entry at front.
	elem := c.lru.PushFront(dropEntry{key: key})
	c.index[key] = elem
}

// Len returns the current number of entries in the cache.
func (c *DropCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lru.Len()
}

// AddIfAbsent atomically inserts the compound key (checksum, arrivalInterfaceID)
// into the cache if it is not already present, and reports whether the key was
// newly added (i.e., this is the first arrival). The check and insert are
// performed under a single lock acquisition, eliminating the Contains-then-Add
// TOCTOU window (F-005 / BC-2.02.002 invariant 1).
//
// Returns true when the frame is a first-arrival (caller should deliver).
// Returns false when the frame is a duplicate (caller should discard).
func (c *DropCache) AddIfAbsent(checksum uint32, arrivalInterfaceID uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := dropKey{checksum, arrivalInterfaceID}

	if elem, ok := c.index[key]; ok {
		// Already present: move to front (LRU refresh) and report duplicate.
		c.lru.MoveToFront(elem)
		return false
	}

	// Evict LRU entry if at capacity.
	if c.lru.Len() >= c.capacity {
		oldest := c.lru.Back()
		if oldest != nil {
			c.lru.Remove(oldest)
			delete(c.index, oldest.Value.(dropEntry).key)
		}
	}

	// Insert new entry at front.
	elem := c.lru.PushFront(dropEntry{key: key})
	c.index[key] = elem
	return true
}

// SendResult describes the dispatch outcome for a single path when Multipath.Send
// is called.
type SendResult struct {
	// PathID is the ID of the path on which the frame was dispatched.
	PathID uint64
	// Sent is true if the frame was handed to the caller's send function for
	// this path.
	Sent bool
}

// SendFunc is the caller-supplied function that actually writes a frame to a
// specific path. Multipath.Send calls it (possibly twice, for the two fastest
// paths) without holding any internal lock.
type SendFunc func(pathID uint64, f Frame) error

// Multipath orchestrates duplicate-and-race dispatch and receiver-side
// deduplication. It holds the active set of ranked paths and two drop caches:
//
//   - dropCache: router-level loop-duplicate suppression, keyed on
//     (checksum, arrival_interface_id) (BC-2.02.009).
//   - recvDedup: endpoint-level first-arrival dedup, keyed on checksum alone
//     (BC-2.02.002 / DI-009). Implemented as a DropCache with a fixed zero
//     interface ID so the compound key collapses to checksum-only.
//
// Zero value is not usable; construct via NewMultipath.
type Multipath struct {
	mu        sync.Mutex
	pathSet   []paths.RankedPath
	dropCache *DropCache
	recvDedup *DropCache
}

// NewMultipath constructs a Multipath dispatcher with the provided initial
// ranked path set and a drop cache of the given capacity.
func NewMultipath(pathSet []paths.RankedPath, dropCacheCapacity int) *Multipath {
	// Clone the slice so the caller cannot mutate our internal state.
	cloned := make([]paths.RankedPath, len(pathSet))
	copy(cloned, pathSet)
	return &Multipath{
		pathSet:   cloned,
		dropCache: NewDropCache(dropCacheCapacity),
		recvDedup: NewDropCache(dropCacheCapacity),
	}
}

// UpdatePaths atomically replaces the ranked path set used for dispatch
// decisions (BC-2.02.001 postcondition 5: rankings are snapshotted at dispatch
// time; this call updates the snapshot for future dispatches only).
func (m *Multipath) UpdatePaths(pathSet []paths.RankedPath) {
	cloned := make([]paths.RankedPath, len(pathSet))
	copy(cloned, pathSet)
	m.mu.Lock()
	m.pathSet = cloned
	m.mu.Unlock()
}

// Send dispatches f on the two highest-scoring paths in the current path set
// (duplicate-and-race, BC-2.02.001 postcondition 1). fn is called once per
// selected path without holding any internal lock.
//
// If only one path is active, f is sent on that single path (EC-001).
// If no paths are active, Send returns ErrNoActivePaths without calling fn.
//
// The returned []SendResult has one entry per path on which fn was called.
func (m *Multipath) Send(f Frame, fn SendFunc) ([]SendResult, error) {
	// Snapshot the path set under lock so rank changes mid-dispatch do not
	// affect this frame (BC-2.02.001 postcondition 5).
	m.mu.Lock()
	snapshot := make([]paths.RankedPath, len(m.pathSet))
	copy(snapshot, m.pathSet)
	m.mu.Unlock()

	ranked, err := paths.Rank(snapshot)
	if err != nil {
		return nil, err
	}

	// Select at most two fastest paths (BC-2.02.001 invariant 2).
	selected := ranked
	if len(selected) > 2 {
		selected = selected[:2]
	}

	results := make([]SendResult, 0, len(selected))
	var lastErr error
	for _, rp := range selected {
		// fn is called without holding any internal lock.
		if fnErr := fn(rp.ID, f); fnErr == nil {
			results = append(results, SendResult{PathID: rp.ID, Sent: true})
		} else {
			// Record the failed attempt so callers can observe it (F-003).
			results = append(results, SendResult{PathID: rp.ID, Sent: false})
			lastErr = fnErr
		}
	}
	// If every selected path failed, surface a wrapped error rather than
	// returning ([], nil) which would silently hide the total failure (F-003).
	sent := 0
	for _, r := range results {
		if r.Sent {
			sent++
		}
	}
	if sent == 0 && lastErr != nil {
		return results, fmt.Errorf("multipath: all %d path(s) failed: %w", len(selected), lastErr)
	}
	return results, nil
}

// frameChecksum computes the CRC32 IEEE checksum over the outer header
// concatenated with the payload (ARCH-03: crc32(outer_header || payload)).
func frameChecksum(f Frame) uint32 {
	h := crc32.NewIEEE()
	_, _ = h.Write(f.OuterHeader[:])
	_, _ = h.Write(f.Payload)
	return h.Sum32()
}

// Receive deduplicates an arriving frame at the endpoint layer. It computes
// the CRC32 checksum of (outerHeader || payload) and performs an atomic
// check-and-insert on the endpoint dedup cache keyed by checksum alone
// (BC-2.02.002 / DI-009 / F-002).
//
// Endpoint dedup is checksum-only: a duplicate copy of the same frame arriving
// on a different interface is still suppressed (DI-009 first-arrival-wins).
// The arrival interface is NOT part of the endpoint dedup key; it is only used
// at the router level (BC-2.02.009 / dropCache).
//
// The check and insert are performed under a single lock acquisition to
// eliminate the Contains-then-Add TOCTOU race (F-005 / BC-2.02.002 invariant 1).
//
// On first arrival: nil is returned — the caller should deliver f to the
// application layer (BC-2.02.002 postcondition 1).
//
// On duplicate: ErrDuplicate is returned and the frame must be silently
// discarded without ACK side-effects (BC-2.02.002 postcondition 2, AC-004,
// AC-005).
func (m *Multipath) Receive(f Frame, _ uint64) error {
	checksum := frameChecksum(f)
	// recvDedup uses a fixed zero interface ID so the compound key collapses to
	// checksum-only semantics (F-002). AddIfAbsent is atomic (F-005).
	if !m.recvDedup.AddIfAbsent(checksum, 0) {
		return ErrDuplicate
	}
	return nil
}
