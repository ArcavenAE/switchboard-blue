// metrics_wire.go — wires the metrics RPC handlers onto the daemon management
// server (BC-2.06.003 v1.9; S-W5.04 AC-001, AC-004, AC-005; F-P1L1-002; F-P2L1-003).
//
// Purity classification (ARCH-09): boundary — connects the management server
// to metrics handler closures. No business logic lives here.
//
// Package DAG note (ARCH-12): cmd/switchboard imports internal/mgmt, internal/metrics,
// and internal/paths via this file. pathTrackerSource adapts the PathTracker registry
// into the PathsListSource interface consumed by the metrics handlers.
package main

import (
	"sync"

	"github.com/arcavenae/switchboard/internal/metrics"
	"github.com/arcavenae/switchboard/internal/mgmt"
	"github.com/arcavenae/switchboard/internal/paths"
)

// pathTrackerSource is a PathsListSource backed by a map of
// pathID → *paths.PathTracker. It is safe for concurrent reads; the map
// is populated before the server starts and is read-only thereafter.
// Calling AllSnapshots() calls PathTracker.Snapshot() on each tracker,
// which takes the tracker's own mutex (go.md rule 12 — no internal pointer leak).
//
// The empty map (no tracked paths) satisfies EC-001: paths.list returns
// {"paths":[],"message":"no active paths"} when AllSnapshots returns empty.
//
// F-P2L1-003 (Ruling-3 Option A): replaces emptyPathsSource stub.
type pathTrackerSource struct {
	mu       sync.RWMutex
	trackers map[string]*paths.PathTracker
}

// newPathTrackerSource constructs a pathTrackerSource. The initial tracker map
// may be nil or empty — in that case AllSnapshots returns an empty map and
// paths.list returns EC-001 "no active paths". The map is populated at
// construction time; no dynamic registration after Serve starts.
func newPathTrackerSource() *pathTrackerSource {
	return &pathTrackerSource{
		trackers: make(map[string]*paths.PathTracker),
	}
}

// AllSnapshots implements metrics.PathsListSource. It calls PathTracker.Snapshot()
// on each registered tracker under the tracker's own mutex, returning a fully
// decoupled copy (go.md rule 12).
func (p *pathTrackerSource) AllSnapshots() map[string]paths.PathSnapshot {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make(map[string]paths.PathSnapshot, len(p.trackers))
	for id, t := range p.trackers {
		out[id] = t.Snapshot()
	}
	return out
}

// emptyRouterMetricsSource is a RouterMetricsSource that returns E-RPC-011 for
// every SVTN lookup. Used until a real forwarding-counter store is wired.
// Tracked in S-BL.ROUTER-METRICS-STORE.
type emptyRouterMetricsSource struct{}

func (emptyRouterMetricsSource) SVTNMetrics(svtnID string) (metrics.RouterMetricsResponse, error) {
	return metrics.RouterMetricsResponse{}, &metricsNotFoundError{svtnID: svtnID}
}

// metricsNotFoundError satisfies the E-RPC-011 contract for unknown SVTNs
// (BC-2.06.003 PC-2; AC-004). The mgmt layer wraps this in the E-RPC-011 envelope.
type metricsNotFoundError struct {
	svtnID string
}

func (e *metricsNotFoundError) Error() string {
	return "E-RPC-011: SVTN not found: " + e.svtnID
}

// wireMetricsHandlers registers the three metrics RPC handlers on srv.
// MUST be called before serveMgmtServer starts the Serve goroutine —
// Register returns an error if called after Serve has started (F-P2L1-001).
// Panics on error because a failure here indicates a programming invariant
// violation (register-before-serve not respected).
//
// BC-2.06.003 v1.9; S-W5.04 AC-001, AC-004, AC-005; F-P1L1-002; F-P2L1-001.
func wireMetricsHandlers(srv *mgmt.Server) {
	if err := mgmt.RegisterMetricsHandlers(srv, newPathTrackerSource(), emptyRouterMetricsSource{}); err != nil {
		panic("wireMetricsHandlers: register-before-serve invariant violated: " + err.Error())
	}
}
