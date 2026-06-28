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
	"errors"
	"fmt"
	"hash/crc32"

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

// FrameArrivalHandler is the router-level frame-arrival processing path.
// It wires:
//   - drop-cache loop-duplicate suppression (BC-2.02.009, AC-004)
//   - EC-005 collision-event logging via an injected Logger (AC-005)
//
// Zero value is not usable; construct via NewFrameArrivalHandler.
type FrameArrivalHandler struct {
	dropCache *multipath.DropCache
	logger    Logger // injected via WithFrameArrivalLogger; nopLogger if nil
}

// NewFrameArrivalHandler constructs a FrameArrivalHandler that consults dc
// for drop-cache loop-duplicate suppression (BC-2.02.009).
//
// opts are applied after construction; supply WithFrameArrivalLogger to enable
// EC-005 collision-event logging (AC-005).
//
// dc must not be nil.
func NewFrameArrivalHandler(dc *multipath.DropCache, opts ...RouterOption) *FrameArrivalHandler {
	h := &FrameArrivalHandler{
		dropCache: dc,
		logger:    nopLogger{},
	}
	_ = opts // RouterOption applies to *Router; use WithFrameArrivalLogger instead
	return h
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

// OnFrameArrival is the router-level frame-arrival handler.
//
// frameBytes is the raw frame (outer header + payload). It is treated as
// opaque — the channel header section is NEVER parsed (BC-2.01.005 / VP-015).
//
// arrivalIface is the logical interface ID the frame arrived on. Together with
// the CRC32 checksum of frameBytes it forms the compound drop-cache key
// (checksum, arrival_interface_id) per ARCH-03 F-006 and BC-2.02.009.
//
// On cache miss: the compound key is added to the DropCache and nil is
// returned — the caller should proceed to split-horizon forwarding
// (BC-2.02.009 postcondition 1; AC-004).
//
// On cache hit: ErrDropCacheHit is returned and the frame must be silently
// discarded (BC-2.02.009; EC-003). If the hit may represent a hash collision
// (EC-005 / EC-004 in the story), a collision-event log line is emitted via
// the injected logger (AC-005).
//
// The DropCache hit counter is incremented on every cache hit as required by
// BC-2.02.009 postcondition 2 (operator diagnostics). Increment is performed
// inside DropCache.AddIfAbsent (S-4.01).
func (h *FrameArrivalHandler) OnFrameArrival(frameBytes []byte, arrivalIface InterfaceID) error {
	// Compute compound drop-cache key: (crc32(frameBytes), arrival_interface_id).
	// frameBytes is treated as opaque — the channel header is NEVER parsed here
	// (BC-2.01.005 / VP-015 / AC-003).
	checksum := crc32.ChecksumIEEE(frameBytes)

	// AddIfAbsent atomically checks and inserts — eliminates TOCTOU window
	// (F-005 / BC-2.02.002 invariant 1). Returns false on hit (duplicate).
	firstArrival := h.dropCache.AddIfAbsent(checksum, uint64(arrivalIface))
	if !firstArrival {
		// Cache hit: loop duplicate (or hash collision) — log as potential
		// collision event for operator investigation (BC-2.02.009 EC-005; AC-005).
		h.logger.Log(fmt.Sprintf(
			"drop cache hit: potential loop duplicate or collision (checksum=0x%08x iface=%d) (BC-2.02.009 EC-005)",
			checksum, arrivalIface,
		))
		return ErrDropCacheHit
	}
	return nil
}
