// metrics_wire.go — wires the metrics RPC handlers onto the daemon management
// server (BC-2.06.003 v1.9; S-W5.04 AC-001, AC-004, AC-005; F-P1L1-002).
//
// Purity classification (ARCH-09): boundary — connects the management server
// to metrics handler closures. No business logic lives here.
//
// Package DAG note (ARCH-12): cmd/switchboard imports internal/mgmt and
// internal/metrics via this file. internal/paths is NOT imported here;
// PathsListSource is satisfied by the emptyPathsSource stub until a
// PathTracker map is wired (S-BL.PATH-TRACKER-MAP).
package main

import (
	"github.com/arcavenae/switchboard/internal/metrics"
	"github.com/arcavenae/switchboard/internal/mgmt"
	"github.com/arcavenae/switchboard/internal/paths"
)

// emptyPathsSource is a PathsListSource that always returns an empty snapshot map.
// Used by the access daemon until a real PathTracker map is wired.
// Tracked in S-BL.PATH-TRACKER-MAP.
type emptyPathsSource struct{}

func (emptyPathsSource) AllSnapshots() map[string]paths.PathSnapshot {
	return map[string]paths.PathSnapshot{}
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

// wireMetricsHandlers registers the three metrics RPC handlers on srv using
// stub source implementations. Called from runAccess immediately after
// startMgmtServer returns (BC-2.06.003 v1.9; F-P1L1-002).
//
// The stub sources (emptyPathsSource, emptyRouterMetricsSource) return empty /
// not-found responses until real state stores are wired in follow-on stories.
func wireMetricsHandlers(srv *mgmt.Server) {
	mgmt.RegisterMetricsHandlers(srv, emptyPathsSource{}, emptyRouterMetricsSource{})
}
