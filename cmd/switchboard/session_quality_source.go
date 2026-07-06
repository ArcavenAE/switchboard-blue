// session_quality_source.go — per-session QualityIndicator registry populated
// by session.Publisher hooks (S-BL.CONSOLE-OBS; BC-2.06.001 v1.7 PC-5
// console-half; BC-2.06.002 v1.4 PC-3 operator export; DRIFT-001b + DRIFT-002
// closures).
//
// Purity classification (ARCH-09): boundary — depends on internal/session
// (boundary) and internal/metrics (pure). Lives here because cmd/switchboard is
// the sole package permitted to sit above both internal/session (DAG 6) and
// internal/metrics (DAG 12) — internal/session MUST NOT import internal/metrics
// (topological inversion; would cascade through internal/tmux at DAG 7).
// Publisher exposes typed SessionHook callbacks so this file can wire the two
// packages without violating ARCH-08 §6.6. Mirrors pathTrackerSource
// (metrics_wire.go) which wires internal/paths onto internal/routing via
// routing.ForwardingEntryHook (S-BL.PATH-TRACKER-WIRING).
//
// Handler surface for the sessions.status RPC also lives here — the handler
// reads through this source, never through session internals.
package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/arcavenae/switchboard/internal/metrics"
	"github.com/arcavenae/switchboard/internal/session"
)

// SessionSnapshot is the operator-visible per-session status projection
// surfaced by `sbctl sessions status`. All fields are value copies; the
// source never leaks internal pointers (go.md rule 12).
//
// Quality is one of "green" | "yellow" | "red" | "pending". "pending" means
// no observation has been recorded on this session yet — the indicator was
// created by the Publisher hook and no OnSessionMeasurement /
// OnSessionMissingFrame call has followed.
//
// BC-2.06.001 v1.7 PC-5; BC-2.06.002 v1.4 PC-3.
type SessionSnapshot struct {
	// Name is the tmux session name (canonical identifier per BC-2.04.001).
	Name string
	// PublishedAt is the UTC timestamp when the session was published;
	// mirrors session.Info.PublishedAt.
	PublishedAt time.Time
	// Quality is the current three-band quality indicator, or "pending"
	// when no observation has been recorded yet on this session.
	Quality string
	// MissCount is the lifetime cumulative missing-frame count for this
	// session (BC-2.06.002 v1.4 PC-3; DRIFT-002).
	MissCount uint64
}

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

// ErrSessionNotFound is the sentinel returned by SessionSnapshot when the
// requested session is not in the source's registry. Distinct from
// session.ErrSessionNotFound — this one is scoped to the sessionQualitySource
// registry, not the Publisher's live-session map. In steady-state operation
// the two are in lockstep because the Publisher hook drives the registry;
// callers observing a divergence should treat both as "session not found".
var errQualitySessionNotFound = errors.New("E-SES-001: session not found in quality source")

// sessionQuality bundles the per-session QualityIndicator with an "observed"
// flag that distinguishes a brand-new session (quality "pending") from one
// that has received an actual green-band measurement (quality "green"). The
// underlying *metrics.QualityIndicator returns Green from Current() on
// construction — that value is not operator-truthful until at least one
// observation has landed.
//
// publishedAt is the timestamp Publisher captured on Publish; carried in the
// hook signature so the source does not need a second lock trip through
// Publisher.Get to render SessionSnapshot.
type sessionQuality struct {
	qi          *metrics.QualityIndicator
	observed    bool
	publishedAt time.Time
}

// sessionQualitySource is a boundary registry keyed by session name. It
// receives Publish / Unpublish notifications via SessionHook callbacks
// installed on the Publisher, and exposes the observation-driver +
// snapshot-reader surface consumed by the sessions.status RPC handler.
//
// Safe for concurrent access — the map is guarded by mu (RWMutex; fast-path
// RLock exists-check on Register, slow-path Lock re-check to avoid the
// concurrent-first-registration race, matching pathTrackerSource).
//
// Lifecycle:
//   - Constructed empty via newSessionQualitySource().
//   - When newSessionQualitySourceFromPublisher is used, the Publisher
//     calls OnPublished(name, publishedAt) on every Publish and
//     OnUnpublished(name, _) on every Unpublish — the source
//     idempotently constructs a per-session indicator on first sight and
//     drops it when Unpublish fires.
//   - OnSessionMeasurement / OnSessionMissingFrame drive observations onto
//     the named session's indicator (BC-2.06.001 PC-2; BC-2.06.002 PC-2).
//   - SessionSnapshots / SessionSnapshot render operator-visible tuples for
//     `sbctl sessions status` (BC-2.06.001 PC-5; BC-2.06.002 PC-3).
type sessionQualitySource struct {
	mu        sync.RWMutex
	qualities map[string]*sessionQuality
}

// newSessionQualitySource constructs an empty sessionQualitySource. Used by
// daemon modes that do not run a session Publisher (though currently only
// console mode registers session handlers, this shape mirrors
// newPathTrackerSource() for callers that want a stand-alone registry —
// tests use this constructor for direct seeding.).
func newSessionQualitySource() *sessionQualitySource {
	return &sessionQualitySource{
		qualities: make(map[string]*sessionQuality),
	}
}

// newSessionQualitySourceFromPublisher constructs a sessionQualitySource and
// installs both SessionHook callbacks on pub. After this call every
// pub.Publish / pub.Unpublish will drive the registry. pub MUST NOT be nil.
//
// The hooks fire inside pub.mu.Lock; OnPublished / OnUnpublished acquire the
// source's own separate lock, so there is no lock inversion (the source
// never re-enters Publisher — SessionHook contract).
//
// Mirrors newPathTrackerSourceFromRouter (S-BL.PATH-TRACKER-WIRING).
func newSessionQualitySourceFromPublisher(pub *session.Publisher) *sessionQualitySource {
	src := newSessionQualitySource()
	pub.SetPublishHook(src.OnPublished)
	pub.SetUnpublishHook(src.OnUnpublished)
	return src
}

// OnPublished is the SessionHook installed on Publisher.publishHook. It
// idempotently constructs a per-session indicator with observed=false so
// SessionSnapshots renders quality "pending" until the first observation.
//
// Called under Publisher.mu.Lock; hook contract requires no re-entry into
// Publisher (this implementation touches only its own state).
func (s *sessionQualitySource) OnPublished(sessionName string, publishedAt time.Time) {
	// Fast path: read lock to check for existing indicator.
	s.mu.RLock()
	_, exists := s.qualities[sessionName]
	s.mu.RUnlock()
	if exists {
		return
	}

	// Slow path: write lock to construct + insert. Re-check under write lock
	// to handle a concurrent-first-registration race — matches
	// pathTrackerSource.Register.
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.qualities[sessionName]; exists {
		return
	}
	s.qualities[sessionName] = &sessionQuality{
		qi:          metrics.NewQualityIndicator(),
		publishedAt: publishedAt,
	}
}

// OnUnpublished is the SessionHook installed on Publisher.unpublishHook. It
// drops the per-session indicator so a re-publish of the same name gets a
// fresh (pending, 0-miss) indicator. publishedAt is unused here but present
// on the SessionHook signature so observers can log accurate lifetimes.
//
// Called under Publisher.mu.Lock; hook contract requires no re-entry into
// Publisher.
func (s *sessionQualitySource) OnUnpublished(sessionName string, _ time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.qualities, sessionName)
}

// OnSessionMeasurement records a measurement (rttMs, lossPct) on the named
// session's QualityIndicator (BC-2.06.001 v1.7 PC-2/PC-3/PC-4).
//
// Returns errQualitySessionNotFound if sessionName is not in the registry.
// The observed flag transitions false → true on first call for this session,
// so subsequent SessionSnapshots renders the real quality band instead of
// "pending".
func (s *sessionQualitySource) OnSessionMeasurement(sessionName string, rttMs, lossPct float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sq, ok := s.qualities[sessionName]
	if !ok {
		return errQualitySessionNotFound
	}
	sq.qi.Update(rttMs, lossPct)
	sq.observed = true
	return nil
}

// OnSessionMissingFrame records one missing-frame event on the named
// session's QualityIndicator (BC-2.06.002 v1.4 PC-1..PC-3). The lifetime
// miss counter increments once per call, independent of hysteresis state.
//
// Returns errQualitySessionNotFound if sessionName is not in the registry.
// The observed flag transitions false → true on first call for this session.
func (s *sessionQualitySource) OnSessionMissingFrame(sessionName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sq, ok := s.qualities[sessionName]
	if !ok {
		return errQualitySessionNotFound
	}
	sq.qi.OnMissingFrame()
	sq.observed = true
	return nil
}

// SessionSnapshot returns the operator-visible snapshot for a single named
// session. Returns (zero, false) when sessionName is not in the registry
// (Go-idiomatic map-lookup shape per go.md rule 12).
//
// BC-2.06.001 v1.7 PC-5 — quality surfaced via `sbctl sessions status`.
func (s *sessionQualitySource) SessionSnapshot(sessionName string) (SessionSnapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sq, ok := s.qualities[sessionName]
	if !ok {
		return SessionSnapshot{}, false
	}
	return SessionSnapshot{
		Name:        sessionName,
		PublishedAt: sq.publishedAt,
		Quality:     qualityString(sq.qi.Current(), sq.observed),
		MissCount:   sq.qi.MissCount(),
	}, true
}

// SessionSnapshots returns an operator-visible snapshot of all currently
// registered sessions, sorted alphabetically by name (VP-031). The returned
// slice is a value copy — mutations do not affect the source's internal
// state (go.md rule 12).
//
// BC-2.06.001 v1.7 PC-5 console-half; DRIFT-001b closure.
func (s *sessionQualitySource) SessionSnapshots() []SessionSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]SessionSnapshot, 0, len(s.qualities))
	for name, sq := range s.qualities {
		out = append(out, SessionSnapshot{
			Name:        name,
			PublishedAt: sq.publishedAt,
			Quality:     qualityString(sq.qi.Current(), sq.observed),
			MissCount:   sq.qi.MissCount(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

// qualityString maps the internal (metrics.Quality, observed) pair to the
// operator-visible enum surface per BC-2.06.003 v1.16.
//
// Rules:
//   - !observed  ⇒ "pending"  (per-session equivalent of EC-008; no data yet)
//   - observed && Green  ⇒ "green"
//   - observed && Yellow ⇒ "yellow"
//   - observed && Red    ⇒ "red"
//   - anything else (defensive) ⇒ "pending"
//
// "failed" is deliberately absent — BC-2.06.003 v1.16 locks the quality enum
// to {green, yellow, red, pending}; "failed" is a *status* value, not a
// *quality* value.
func qualityString(current metrics.Quality, observed bool) string {
	if !observed {
		return "pending"
	}
	switch current {
	case metrics.Green:
		return "green"
	case metrics.Yellow:
		return "yellow"
	case metrics.Red:
		return "red"
	default:
		// Defensive: unexpected Quality value ⇒ pending (indeterminate),
		// mirroring metrics.Quality.String()'s "unknown" fallback intent
		// without leaking a non-enum value to the operator.
		return "pending"
	}
}

// HandleSessionsStatus is the transport-agnostic handler for the
// sessions.status RPC. It returns per-session {name, published_at, quality,
// miss_count} tuples derived from the source's registry.
//
// Semantics:
//   - req.SessionName == ""  → return all sessions, sorted alphabetically by name.
//   - req.SessionName != ""  → return exactly one entry; E-SES-001 if unknown.
//
// Empty result (no sessions in the registry, req.SessionName == "") returns a
// zero-length Sessions slice and no error — the operator surface distinguishes
// "no sessions" from "error" naturally.
//
// The (context.Context) parameter is accepted for future cancel-plumbing but
// is currently unused; the handler is O(N) in the registry size under a
// read-lock and does not block on I/O.
//
// Return contract:
//   - (SessionsStatusResponse, nil) on success — sessions slice may be empty.
//   - (zero, E-SES-001 wrapped) when a named session is not found.
//
// Handler is safe for concurrent invocation.
func (s *sessionQualitySource) HandleSessionsStatus(_ context.Context, req SessionsStatusRequest) (SessionsStatusResponse, error) {
	if req.SessionName != "" {
		snap, ok := s.SessionSnapshot(req.SessionName)
		if !ok {
			return SessionsStatusResponse{}, fmt.Errorf(
				"E-SES-001: session not found: %s: %w",
				req.SessionName, errQualitySessionNotFound,
			)
		}
		return SessionsStatusResponse{
			Sessions: []SessionStatusEntry{snapshotToEntry(snap)},
		}, nil
	}

	snaps := s.SessionSnapshots()
	out := make([]SessionStatusEntry, 0, len(snaps))
	for _, sn := range snaps {
		out = append(out, snapshotToEntry(sn))
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
