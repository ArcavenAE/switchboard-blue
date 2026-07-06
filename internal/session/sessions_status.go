package session

// File: sessions_status.go — RPC handler for the sessions.status mgmt-plane
// command (S-BL.CONSOLE-OBS; BC-2.06.001 v1.7 PC-5 console-half; BC-2.06.002
// v1.4 PC-3 operator export; DRIFT-001b + DRIFT-002 closures).
//
// The console-mode daemon registers a handler bound to Publisher.HandleSessionsStatus.
// The handler answers two shapes:
//
//	sessions.status {}                 → all sessions, sorted by name
//	sessions.status {session_name: "X"} → single session detail; E-SES-001 if unknown
//
// Snapshot values are value copies of SessionSnapshot (Name, PublishedAt UTC,
// Quality string {green|yellow|red|pending}, MissCount uint64). Publisher never
// leaks internal pointers (go.md rule 12).
//
// Error surface:
//   - E-SES-001: session_name provided but not in the live set
//     (wraps ErrSessionNotFound)
//
// Tier-2 admission (E-ADM-006 for RoleControl / RoleConsole only) is enforced
// at the cmd/switchboard handler shim, not here — this file is transport-agnostic.

import (
	"context"
	"fmt"
	"time"
)

// SessionsStatusRequest is the wire-format request payload for the
// sessions.status RPC command.
//
// If SessionName is empty, the handler returns snapshots for all sessions.
// If SessionName is non-empty, the handler returns exactly one snapshot for
// the named session, or E-SES-001 if the session is not published.
type SessionsStatusRequest struct {
	// SessionName selects a single session; empty selects all.
	SessionName string `json:"session_name,omitempty"`
}

// SessionsStatusResponse is the wire-format success response.
//
// Sessions is a value-copy slice sorted alphabetically by Name for the "all"
// query, or a single-element slice for the "by name" query. An empty response
// (Sessions == nil-or-empty) is legitimate — it means the daemon has no
// published sessions yet.
type SessionsStatusResponse struct {
	// Sessions is the operator-visible per-session status projection.
	Sessions []SessionStatusEntry `json:"sessions"`
}

// SessionStatusEntry is the JSON-tagged wire form of one SessionSnapshot.
//
// Quality values: "green" | "yellow" | "red" | "pending"
// (BC-2.06.003 v1.16 locked enum; "failed" NEVER appears here — it is a
// status value, not a quality value).
//
// MissCount is a lifetime cumulative counter of OnMissingFrame events for
// the session's underlying QualityIndicator (BC-2.06.002 v1.4 PC-3;
// DRIFT-002).
//
// PublishedAt is the RFC 3339 UTC timestamp when the session was published.
type SessionStatusEntry struct {
	// Name is the tmux session name.
	Name string `json:"name"`
	// PublishedAt is the RFC 3339 UTC timestamp of publication.
	PublishedAt time.Time `json:"published_at"`
	// Quality is one of "green" | "yellow" | "red" | "pending".
	Quality string `json:"quality"`
	// MissCount is the lifetime cumulative miss-frame event count.
	MissCount uint64 `json:"miss_count"`
}

// HandleSessionsStatus is the transport-agnostic handler for the
// sessions.status RPC. It returns per-session {name, published_at, quality,
// miss_count} tuples derived from the Publisher's live session set and
// per-session QualityIndicator wrappers.
//
// Semantics:
//   - req.SessionName == ""  → return all sessions, sorted alphabetically by name.
//   - req.SessionName != ""  → return exactly one entry; E-SES-001 if unknown.
//
// Empty result (no sessions published, req.SessionName == "") returns a
// zero-length Sessions slice and no error — the operator surface distinguishes
// "no sessions" from "error" naturally.
//
// The (context.Context) parameter is accepted for future cancel-plumbing but
// is currently unused; the handler is O(N) in the live set size under a
// read-lock and does not block on I/O.
//
// Return contract:
//   - (SessionsStatusResponse, nil) on success — sessions slice may be empty.
//   - (zero, E-SES-001 wrapped) when a named session is not found.
//
// Handler is safe for concurrent invocation — it holds the Publisher read-lock
// via SessionSnapshot/SessionSnapshots.
func (p *Publisher) HandleSessionsStatus(_ context.Context, req SessionsStatusRequest) (SessionsStatusResponse, error) {
	if req.SessionName != "" {
		snap, ok := p.SessionSnapshot(req.SessionName)
		if !ok {
			return SessionsStatusResponse{}, fmt.Errorf(
				"E-SES-001: session not found: %s: %w",
				req.SessionName, ErrSessionNotFound,
			)
		}
		return SessionsStatusResponse{
			Sessions: []SessionStatusEntry{snapshotToEntry(snap)},
		}, nil
	}

	snaps := p.SessionSnapshots()
	out := make([]SessionStatusEntry, 0, len(snaps))
	for _, s := range snaps {
		out = append(out, snapshotToEntry(s))
	}
	return SessionsStatusResponse{Sessions: out}, nil
}

// snapshotToEntry maps the internal SessionSnapshot value to the JSON-tagged
// wire entry. Kept private because the mapping is trivial and only serves
// the sessions.status handler; consumers should use SessionStatusEntry.
//
// The two structs are field-identical (Name, PublishedAt, Quality, MissCount)
// so a direct type conversion is sufficient — SessionSnapshot carries no JSON
// tags, SessionStatusEntry does. staticcheck S1016 recognizes this pattern.
func snapshotToEntry(s SessionSnapshot) SessionStatusEntry {
	return SessionStatusEntry(s)
}
