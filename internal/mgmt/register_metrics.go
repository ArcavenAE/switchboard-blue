// Registration of metrics RPC handlers on the mgmt.Server (ARCH-12; S-W5.04 AC-001, AC-004, AC-005).
//
// Register extends the handler table after construction — necessary because
// daemon components like the metrics handlers are wired after NewServer returns.
// The handler Fn closures close over injected dependencies (PathsListSource,
// RouterMetricsSource) so that internal/mgmt never imports data-plane packages
// (ARCH-12 §Package DAG Constraints).
//
// Purity classification (ARCH-09): boundary — wires effectful handlers into the
// management server; no business logic lives here.

package mgmt

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/arcavenae/switchboard/internal/metrics"
)

// Register appends h to the server's handler table. Returns an error if called
// after Server.Serve has started — callers MUST register all handlers before
// calling Serve (register-before-serve invariant; F-P2L1-001).
//
// S-W5.04: used to register paths.list, router.metrics, and router.status.
func (s *Server) Register(h Handler) error {
	if s.serving.Load() {
		return fmt.Errorf("mgmt.Register: cannot register handler %q after Serve has started", h.Command)
	}
	s.handlers = append(s.handlers, h)
	return nil
}

// RegisterMetricsHandlers registers the three metrics RPC handlers on s:
//   - "paths.list"     → metrics.PathsList   (BC-2.06.003 PC-1; AC-001)
//   - "router.metrics" → metrics.RouterMetrics (BC-2.06.003 PC-2; AC-004)
//   - "router.status"  → metrics.RouterStatus  (BC-2.06.003 PC-3; AC-005)
//
// pathsSrc provides AllSnapshots() for the paths.list and router.status handlers.
// routerSrc provides SVTNMetrics() for the router.metrics handler.
//
// Called once during daemon startup (cmd/switchboard), before Server.Serve.
// Returns an error if any handler registration fails (i.e. Serve has already started).
func RegisterMetricsHandlers(s *Server, pathsSrc metrics.PathsListSource, routerSrc metrics.RouterMetricsSource) error {
	if err := s.Register(Handler{Command: "paths.list", Fn: pathsListHandler(pathsSrc)}); err != nil {
		return err
	}
	if err := s.Register(Handler{Command: "router.metrics", Fn: routerMetricsHandler(routerSrc)}); err != nil {
		return err
	}
	return s.Register(Handler{Command: "router.status", Fn: routerStatusHandler(pathsSrc)})
}

// pathsListHandler returns a mgmt.Handler.Fn for the "paths.list" command.
// The returned function closes over src and delegates to metrics.PathsList.
func pathsListHandler(src metrics.PathsListSource) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		return metrics.PathsList(ctx, args, src)
	}
}

// routerMetricsHandler returns a mgmt.Handler.Fn for the "router.metrics" command.
// The returned function closes over src and delegates to metrics.RouterMetrics.
func routerMetricsHandler(src metrics.RouterMetricsSource) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		return metrics.RouterMetrics(ctx, args, src)
	}
}

// routerStatusHandler returns a mgmt.Handler.Fn for the "router.status" command.
// The returned function closes over src and delegates to metrics.RouterStatus.
func routerStatusHandler(src metrics.PathsListSource) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		return metrics.RouterStatus(ctx, args, src)
	}
}
