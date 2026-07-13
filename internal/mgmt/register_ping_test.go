// register_ping_test.go — unit tests for the paths.ping RPC handler
// (AC-004 PC-3; BC-2.06.004 Postcondition 3, Invariant 1, Invariant 2).
//
// White-box (package mgmt, not mgmt_test): needs the unexported pingHandler()
// closure and the private Server.handlers table to (a) invoke the handler in
// isolation without a live socket and (b) prove zero-interaction when
// co-registered with the metrics trio on one shared server. mgmt_test.go's
// black-box style covers full-handshake RPC dispatch elsewhere in this
// package; this file is scoped to the handler contract itself.
package mgmt

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/arcavenae/switchboard/internal/metrics"
)

// TestPingHandler_EmptyArgsIn_PongOut_ZeroPathTrackerInteraction verifies
// AC-004 PC-3 / BC-2.06.004 Invariant 2: {} in, {"pong": true} out, no other
// side effect, zero PathTracker (or any tracker-backed source) interaction.
//
// Two independent proofs, both required — neither alone establishes the claim:
//
//  1. "shape" (direct closure invocation): calling pingHandler()'s returned Fn
//     with {} returns exactly {"pong":true} — no extra field, no omission, no
//     falsified value. Catches a wrong implementation that adds a field (e.g.
//     a leaked quality classification or rtt_ms — PC-4's explicit
//     prohibition, cross-referenced by this AC's own text), omits "pong", or
//     returns pong:false.
//  2. "zero_interaction_on_shared_server" (dynamic): registers paths.ping
//     alongside the full metrics trio on ONE server — exactly the shape
//     production wireMetricsHandlers produces — with a RouterMetricsSource
//     spy whose SVTNMetrics method fails the test if ever called, then looks
//     up paths.ping's Fn from that SAME server's handler table (not a
//     standalone construction) and dispatches it. Catches a regression where
//     paths.ping is accidentally aliased or delegated to shared
//     metrics/tracker logic once co-registered with the trio — a failure
//     mode proof 1 cannot see, because proof 1 never puts a live
//     tracker-backed source in scope to begin with. A nil PathsListSource is
//     passed for the unused paths/router.status parameter rather than a
//     built spy: internal/mgmt architecturally never imports internal/paths
//     (ARCH-12), so no PathSnapshot-shaped spy can be constructed from this
//     package, and none of this test's assertions touch that parameter.
//
// AC-004 PC-3; BC-2.06.004 Postcondition 3, Invariant 1, Invariant 2.
func TestPingHandler_EmptyArgsIn_PongOut_ZeroPathTrackerInteraction(t *testing.T) {
	t.Run("shape", func(t *testing.T) {
		fn := pingHandler()

		resp, err := fn(context.Background(), json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("pingHandler()(ctx, {}): unexpected error: %v", err)
		}

		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("marshal response: %v", err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(data, &fields); err != nil {
			t.Fatalf("unmarshal response into field map: %v", err)
		}
		if len(fields) != 1 {
			t.Errorf("response field count: got %d %v; want exactly 1 (\"pong\") — "+
				"BC-2.06.004 PC-4 forbids any additional field (e.g. a quality classification)",
				len(fields), fields)
		}
		pongRaw, ok := fields["pong"]
		if !ok {
			t.Fatalf("response missing \"pong\" field: %s", data)
		}
		var pong bool
		if err := json.Unmarshal(pongRaw, &pong); err != nil {
			t.Fatalf("unmarshal \"pong\" field: %v", err)
		}
		if !pong {
			t.Errorf("response \"pong\": got false; want true")
		}
	})

	t.Run("zero_interaction_on_shared_server", func(t *testing.T) {
		spy := routerMetricsSourceMustNotBeCalled{t: t}

		s := &Server{}
		if err := RegisterMetricsHandlers(s, nil, spy); err != nil {
			t.Fatalf("RegisterMetricsHandlers: %v", err)
		}
		if err := RegisterPingHandler(s); err != nil {
			t.Fatalf("RegisterPingHandler: %v", err)
		}

		var pingFn func(ctx context.Context, args json.RawMessage) (any, error)
		for _, h := range s.handlers {
			if h.Command == "paths.ping" {
				pingFn = h.Fn
			}
		}
		if pingFn == nil {
			t.Fatalf("paths.ping not found in handler table after RegisterPingHandler")
		}

		resp, err := pingFn(context.Background(), json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("paths.ping dispatch: unexpected error: %v", err)
		}
		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("marshal response: %v", err)
		}
		if string(data) != `{"pong":true}` {
			t.Errorf("paths.ping response: got %s; want {\"pong\":true}", data)
		}
		// spy.SVTNMetrics would have called t.Fatalf if paths.ping's dispatch
		// had reached into RouterMetricsSource; reaching this line proves it
		// did not, for this run.
	})
}

// routerMetricsSourceMustNotBeCalled is a metrics.RouterMetricsSource whose
// SVTNMetrics fails the test if ever invoked — a tripwire proving paths.ping,
// registered alongside the metrics trio on one shared server, never reaches
// into RouterMetricsSource (BC-2.06.004 Invariant 1 / AC-004 PC-3).
type routerMetricsSourceMustNotBeCalled struct {
	t *testing.T
}

func (r routerMetricsSourceMustNotBeCalled) SVTNMetrics(svtnID string) (metrics.RouterMetricsResponse, error) {
	r.t.Helper()
	r.t.Fatalf("BC-2.06.004 Invariant 1 / AC-004 PC-3: paths.ping dispatch invoked "+
		"RouterMetricsSource.SVTNMetrics(%q); paths.ping must perform zero "+
		"tracker-backed-source interaction", svtnID)
	return metrics.RouterMetricsResponse{}, nil
}
