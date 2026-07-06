// sessions_handlers.go — daemon-side session-observability RPC handlers.
//
// BuildSessionsHandlers returns the []mgmt.Handler slice for session
// observability RPCs registered by the console-mode daemon:
//
//	sessions.status  (S-BL.CONSOLE-OBS; BC-2.06.001 v1.7 PC-5 console-half;
//	                  BC-2.06.002 v1.4 PC-3 operator export;
//	                  DRIFT-001b + DRIFT-002 closures)
//
// The console-mode daemon calls BuildSessionsHandlers alongside
// BuildConsoleHandlers. Access / control / router daemons pass nil handlers
// (ADR-004 role-exclusion; ARCH-04 disambiguation table).
//
// Tier-2 admission (L1-C4): each handler checks that the authenticated caller
// holds RoleControl or RoleConsole via the mgmt-plane context (CallerPubkey +
// AdmittedKeySet lookup). Callers with any other role receive E-ADM-006.
// The admission helper is shared with BuildConsoleHandlers
// (verifyConsoleCallerRole in console_handlers.go) — same trust surface,
// same error envelope shape.
//
// Purity classification (ARCH-09): boundary — depends on session.Publisher
// (boundary) via handler-input contracts only, sessionQualitySource (same
// package) for state reads, and mgmt.Handler (struct). No data-plane imports
// permitted (ADR-004 + ARCH-12 data-plane/management-plane separation).
//
// Wiring shape (course-correct 2026-07-05, ARCH-08 §6.6 preservation): the
// per-session QualityIndicator map, observation entry points, and the
// sessions.status handler read-path all live in the boundary package
// cmd/switchboard — the same package that already imports both
// internal/session (DAG 6) and internal/metrics (DAG 12). session.Publisher
// fires typed SessionHook callbacks (mirrors routing.ForwardingEntryHook) so
// internal/session does not import internal/metrics. The handler here reads
// through sessionQualitySource, not through Publisher internals.
package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/mgmt"
)

// BuildSessionsHandlers returns a []mgmt.Handler for the sessions.status RPC.
// src must not be nil; ks must not be nil (both required for dispatch).
//
// src is the console daemon's sessionQualitySource — the observation registry
// populated by session.Publisher hooks and the read-path for sessions.status.
// Publisher itself is not passed here because the handler does not consult
// Publisher directly; state is projected through src to preserve the
// ARCH-08 §6.6 DAG (internal/session MUST NOT import internal/metrics).
//
// ks is the AdmittedKeySet used to enforce Tier-2 (session-plane)
// authorization for RoleControl / RoleConsole callers.
func BuildSessionsHandlers(src *sessionQualitySource, ks *admission.AdmittedKeySet) []mgmt.Handler {
	if src == nil {
		panic("BuildSessionsHandlers: sessionQualitySource must not be nil")
	}
	if ks == nil {
		panic("BuildSessionsHandlers: AdmittedKeySet must not be nil")
	}
	return []mgmt.Handler{
		{Command: "sessions.status", Fn: makeSessionsStatusHandler(src, ks)},
	}
}

// makeSessionsStatusHandler returns the sessions.status handler function.
//
// Wire contract:
//   - Request:  SessionsStatusRequest  {session_name?: string}
//   - Response: SessionsStatusResponse {sessions: [{name, published_at,
//     quality, miss_count}]}
//   - Errors:
//   - E-ADM-006 (Tier-2 authorization denied)
//   - E-CFG-001 (JSON unmarshal failure)
//   - E-SES-001 (session_name provided but not found)
//
// Empty request body ({} or {"session_name": ""}) returns all sessions. A
// non-empty session_name returns exactly one entry.
//
// Traces to BC-2.06.001 v1.7 PC-5 console-half; BC-2.06.002 v1.4 PC-3;
// DRIFT-001b + DRIFT-002; L1-C4.
func makeSessionsStatusHandler(
	src *sessionQualitySource,
	ks *admission.AdmittedKeySet,
) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		// L1-C4: Tier-2 admission check before any state read.
		// Shares the RoleControl/RoleConsole surface with console.attach/detach/switch
		// — sessions.status is an operator-visibility surface on the same daemon,
		// same trust boundary.
		if err := verifyConsoleCallerRole(ctx, ks, "sessions.status"); err != nil {
			return nil, err
		}

		// Empty args (JSON `null` or missing) is treated as an "all sessions"
		// query. The unmarshal call handles both {} and {"session_name": "..."}
		// shapes; only propagate the error when args is non-empty AND malformed.
		var req SessionsStatusRequest
		if len(args) > 0 && string(args) != "null" {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, fmt.Errorf("E-CFG-001: invalid request args: %w", err)
			}
		}

		return src.HandleSessionsStatus(ctx, req)
	}
}
