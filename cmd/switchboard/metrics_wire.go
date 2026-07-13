// metrics_wire.go — wires the metrics RPC handlers onto the daemon management
// server (BC-2.06.003 v1.15; S-W5.04 AC-001, AC-004, AC-005; F-P1L1-002;
// F-P2L1-003; S-BL.PATH-TRACKER-WIRING).
//
// Purity classification (ARCH-09): boundary — connects the management server
// to metrics handler closures. No business logic lives here.
//
// Package DAG note (ARCH-12): cmd/switchboard imports internal/mgmt,
// internal/metrics, internal/paths, and internal/routing via this file.
// pathTrackerSource adapts a live PathTracker registry into the PathsListSource
// interface consumed by the metrics handlers. cmd/switchboard is the sole
// package that may sit above both routing (DAG 5) and paths (DAG 8), so the
// registry lives here — routing exposes a typed hook to notify us when a
// forwarding entry lands (S-BL.PATH-TRACKER-WIRING; ARCH-08 §6).
package main

import (
	"errors"
	"fmt"
	"sync"

	"github.com/arcavenae/switchboard/internal/metrics"
	"github.com/arcavenae/switchboard/internal/mgmt"
	"github.com/arcavenae/switchboard/internal/paths"
	"github.com/arcavenae/switchboard/internal/routing"
)

// ErrRouterSVTNNotFound is the sentinel returned when router.status or
// router.metrics is invoked for an unknown SVTN. Wraps as E-RPC-011.
// Tests MUST use errors.Is(err, ErrRouterSVTNNotFound) — no string matching.
//
// Distinct from svtnmgmt.ErrSVTNNotFound; this sentinel is scoped to the
// router.metrics wire path and stamps E-RPC-011 with the E-SVTN-003 message
// prefix. (F-P9L1-02)
var ErrRouterSVTNNotFound = errors.New("E-RPC-011: SVTN not found")

// pathTrackerInitialRTTMs is the conservative initial RTT (milliseconds) seeded
// into every PathTracker constructed by the router forwarding-entry hook.
// Mirrors the TCP RFC 6298 first-probe convention: the initial value is
// replaced outright by paths.resetRTT on the first successful probe, so a
// conservative default does not poison the EWMA.
const pathTrackerInitialRTTMs = 250.0

// pathTrackerEWMAAlpha is the EWMA smoothing factor for PathTracker instances
// constructed by the router hook. 0.125 is the RFC 6298 recommended value
// (window of ~8 probes) and matches every other PathTracker constructor call
// site in the codebase.
const pathTrackerEWMAAlpha = 0.125

// pathTrackerSource is a PathsListSource backed by a map of
// pathID → *paths.PathTracker maintained by the router forwarding-entry hook
// (S-BL.PATH-TRACKER-WIRING). It is safe for concurrent access — the map is
// guarded by mu (S-BL.PATH-TRACKER-WRITER folded per Ruling-11), and each
// PathTracker in the map carries its own lock via paths.PathTracker.
//
// Lifecycle:
//   - Constructed empty via newPathTrackerSource().
//   - When newPathTrackerSourceFromRouter is used, the Router calls
//     Register(svtnID, nodeAddr) on every RegisterForwardingEntry — the
//     source lazily constructs a PathTracker on first sight of a pathID and
//     returns the existing one on subsequent calls (LWW at the tracker
//     identity level: replaces of the auth key do not lose accumulated
//     RTT/loss history).
//   - AllSnapshots() calls PathTracker.Snapshot() on every tracker under mu.RLock.
//
// The empty map (no tracked paths) satisfies BC-2.06.003 EC-001: paths.list
// returns {"paths":[],"message":"no active paths"} when AllSnapshots returns empty.
//
// F-P2L1-003 (Ruling-3 Option A): replaces emptyPathsSource stub. Ruling-6
// (production tracker enumeration): now fully wired via the router hook.
// Ruling-11 (concurrent-writer protection): mu guards Register + AllSnapshots.
type pathTrackerSource struct {
	mu       sync.RWMutex
	trackers map[string]*paths.PathTracker
}

// newPathTrackerSource constructs an empty pathTrackerSource. Used for daemon
// modes that do not run a routing subsystem (console, control) — the source
// returns an empty tracker map and paths.list returns EC-001 "no active paths".
func newPathTrackerSource() *pathTrackerSource {
	return &pathTrackerSource{
		trackers: make(map[string]*paths.PathTracker),
	}
}

// newPathTrackerSourceFromRouter constructs a pathTrackerSource that is
// populated on demand by the router's forwarding-entry hook. Installs
// WithForwardingEntryHook on r before returning — the hook fires every time
// RegisterForwardingEntry is called, and Register (below) constructs a
// PathTracker on first sight of (svtnID, nodeAddr).
//
// r MUST NOT be nil — callers with no live router use newPathTrackerSource()
// directly (see runConsole, runControl).
//
// The hook is installed on r via a functional option-style mutator: routing
// exposes a receiver method Router.SetForwardingEntryHook (below) so the hook
// can be attached after construction. This is necessary because the router
// and the source are cyclic — the source needs a reference to the router (no,
// actually it doesn't; only the router's hook needs a reference to the source).
// Callers pass the router purely to attach the hook.
//
// S-BL.PATH-TRACKER-WIRING; BC-2.06.003 PC-1.
func newPathTrackerSourceFromRouter(r *routing.Router) *pathTrackerSource {
	src := newPathTrackerSource()
	r.SetForwardingEntryHook(src.Register)
	return src
}

// Register is the callback the router fires on every RegisterForwardingEntry.
// It is idempotent per pathID: the first call constructs a PathTracker; every
// subsequent call for the same (svtnID, nodeAddr) is a no-op (tracker identity
// preserved across auth-key rotations per S-BL.PATH-TRACKER-WIRING AC-4).
//
// The router calls this while holding its own write lock; Register acquires
// its own separate lock (pathTrackerSource.mu) so there is no lock inversion.
// Router's contract: hook implementations MUST NOT re-enter Router — Register
// touches only its own state, so this is satisfied.
//
// pathID format mirrors the routing.Router forwardingTable partitioning
// ("%x-%x", svtnID, nodeAddr) so tests and operators can correlate identities.
func (p *pathTrackerSource) Register(svtnID [16]byte, nodeAddr [8]byte) {
	pathID := fmt.Sprintf("%x-%x", svtnID, nodeAddr)

	// Fast path: read lock to check for existing tracker.
	p.mu.RLock()
	_, exists := p.trackers[pathID]
	p.mu.RUnlock()
	if exists {
		return
	}

	// Slow path: write lock to construct + insert. Re-check under write lock
	// to handle the concurrent-first-registration race.
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.trackers[pathID]; exists {
		return
	}
	p.trackers[pathID] = paths.NewPathTracker(pathTrackerInitialRTTMs, pathTrackerEWMAAlpha)
}

// AllSnapshots implements metrics.PathsListSource. It takes a read snapshot
// of the tracker map under mu.RLock, then calls PathTracker.Snapshot() on
// each — Snapshot returns a value copy (go.md rule 12) so the returned map
// is fully decoupled from internal state.
func (p *pathTrackerSource) AllSnapshots() map[string]paths.PathSnapshot {
	p.mu.RLock()
	// Copy pointers under the RLock so we can drop the source lock before
	// calling each PathTracker.Snapshot(). Each tracker has its own mutex
	// and is safe for concurrent snapshot; holding the source RLock across
	// N snapshot calls would unnecessarily block writers.
	pinned := make(map[string]*paths.PathTracker, len(p.trackers))
	for id, t := range p.trackers {
		pinned[id] = t
	}
	p.mu.RUnlock()

	out := make(map[string]paths.PathSnapshot, len(pinned))
	for id, t := range pinned {
		out[id] = t.Snapshot()
	}
	return out
}

// emptyRouterMetricsSource is a RouterMetricsSource that returns E-RPC-011 for
// every SVTN lookup. Used until a real forwarding-counter store is wired.
//
// #DEFERRED: S-BL.ROUTER-METRICS-STORE — production forwarding-counter store
// deferred to a later wave. Handler surface, response types, and adapter
// interface land in S-W5.04; production population (wiring this to a real
// counter store) lands in S-BL.ROUTER-METRICS-STORE. Mirrors the historic
// pathTrackerSource deferred pattern (now closed by S-BL.PATH-TRACKER-WIRING).
type emptyRouterMetricsSource struct{}

func (emptyRouterMetricsSource) SVTNMetrics(svtnID string) (metrics.RouterMetricsResponse, error) {
	return metrics.RouterMetricsResponse{}, &metricsNotFoundError{svtnID: svtnID}
}

// metricsNotFoundError satisfies the E-RPC-011 contract for unknown SVTNs
// (BC-2.06.003 PC-2; AC-004). The mgmt dispatch layer re-wraps this error in the
// JSON-RPC error envelope (see internal/mgmt mgmt.go dispatch convention).
//
// Is implements errors.Is support so callers can do errors.Is(err, ErrRouterSVTNNotFound)
// without string matching (go.md error-handling rule 3; H-1 Pass-8).
type metricsNotFoundError struct {
	svtnID string
}

func (e *metricsNotFoundError) Error() string {
	return "E-RPC-011: SVTN not found: " + e.svtnID
}

// Is reports whether target equals ErrRouterSVTNNotFound, enabling errors.Is matching.
func (e *metricsNotFoundError) Is(target error) bool {
	return target == ErrRouterSVTNNotFound
}

// wireMetricsHandlers registers the metrics RPC handlers (paths.list,
// router.metrics, router.status) plus paths.ping on srv.
// MUST be called before serveMgmtServer starts the Serve goroutine —
// Register returns an error if called after Serve has started (F-P2L1-001).
// Returns an error on registration failure so the main-package caller can
// log.Fatalf — only main is the allowed exit site (go.md; F-P10L1-04).
//
// router is the live routing.Router for the daemon. When non-nil, the metrics
// pathTrackerSource is populated by the router's forwarding-entry hook
// (S-BL.PATH-TRACKER-WIRING). When nil (console, control modes with no router
// in scope), the source is an empty registry and paths.list returns
// EC-001 "no active paths".
//
// paths.ping (S-BL.CLI-SURFACE-COMPLETION Decision 1 / AC-004) is registered
// here — not inside mgmt.RegisterMetricsHandlers — because it targets an
// arbitrary daemon and is not scoped to the metrics-handler trio; it is
// available on every daemon mode that calls this function (runRouter,
// runAccess, runConsole, runControl), matching the metrics handlers' reach.
//
// BC-2.06.003 v1.15; S-W5.04 AC-001, AC-004, AC-005; F-P1L1-002; F-P2L1-001;
// S-BL.PATH-TRACKER-WIRING; BC-2.06.004 (S-BL.CLI-SURFACE-COMPLETION).
func wireMetricsHandlers(srv *mgmt.Server, router *routing.Router) error {
	var pathsSrc *pathTrackerSource
	if router != nil {
		pathsSrc = newPathTrackerSourceFromRouter(router)
	} else {
		pathsSrc = newPathTrackerSource()
	}
	if err := mgmt.RegisterMetricsHandlers(srv, pathsSrc, emptyRouterMetricsSource{}); err != nil {
		return fmt.Errorf("wireMetricsHandlers: register-before-serve invariant violated: %w", err)
	}
	if err := mgmt.RegisterPingHandler(srv); err != nil {
		return fmt.Errorf("wireMetricsHandlers: register-before-serve invariant violated: %w", err)
	}
	return nil
}
