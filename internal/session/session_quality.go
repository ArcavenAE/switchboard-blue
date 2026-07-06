package session

// File: session_quality.go — per-session QualityIndicator wiring on Publisher
// (S-BL.CONSOLE-OBS; BC-2.06.001 v1.7 PC-5 console-half; BC-2.06.002 v1.4 PC-3;
// DRIFT-001b + DRIFT-002 closures).
//
// The Publisher owns a parallel map of *metrics.QualityIndicator values keyed
// by session name, created on Publish and dropped on Unpublish. Observations
// (measurements + missing-frame events) route through OnSessionMeasurement /
// OnSessionMissingFrame to the named session's indicator. SessionSnapshots
// and SessionSnapshot expose the operator-visible {Name, PublishedAt, Quality,
// MissCount} tuple for `sbctl sessions status`.
//
// Quality enum surfaced in snapshots: "green" | "yellow" | "red" | "pending"
// per BC-2.06.003 v1.16 (locked by row_e_failed_and_pending; "failed" NEVER
// appears as a quality value). "pending" here means "no observation received
// yet on this session" — distinct from EC-008's empty-paths pending because
// this is per-session, not per-node aggregate.
//
// ARCH-08 §6.6: internal/session now imports internal/metrics in addition to
// its existing {frame, admission} imports; internal/metrics is at position 12
// with only {paths} allowed, so the import direction is downstream — no cycle.
// The story's architecture_modules field ([internal/session, internal/metrics])
// declares the added dependency.

import (
	"sort"
	"time"

	"github.com/arcavenae/switchboard/internal/metrics"
)

// SessionSnapshot is the operator-visible per-session status projection
// surfaced by `sbctl sessions status`. All fields are value copies; the
// Publisher never leaks internal pointers (go.md rule 12).
//
// Quality is one of "green" | "yellow" | "red" | "pending". "pending" means
// no observation has been recorded on this session yet — the indicator was
// created by Publish and no OnSessionMeasurement / OnSessionMissingFrame call
// has followed.
//
// BC-2.06.001 v1.7 PC-5; BC-2.06.002 v1.4 PC-3.
type SessionSnapshot struct {
	// Name is the tmux session name (canonical identifier per BC-2.04.001).
	Name string
	// PublishedAt is the UTC timestamp when the session was published;
	// mirrors Info.PublishedAt.
	PublishedAt time.Time
	// Quality is the current three-band quality indicator, or "pending" when
	// no observation has been recorded yet on this session.
	Quality string
	// MissCount is the lifetime cumulative missing-frame count for this
	// session (BC-2.06.002 v1.4 PC-3; DRIFT-002).
	MissCount uint64
}

// sessionQuality bundles the per-session QualityIndicator with an "observed"
// flag that distinguishes a brand-new session (quality "pending") from one
// that has received an actual green-band measurement (quality "green"). The
// underlying *metrics.QualityIndicator returns Green from Current() on
// construction — that value is not operator-truthful until at least one
// observation has landed.
type sessionQuality struct {
	qi       *metrics.QualityIndicator
	observed bool
}

// newSessionQuality constructs a fresh per-session quality record. The
// underlying indicator starts at metrics.Green; the observed flag is false,
// so SessionSnapshots renders the session as "pending" until the first
// observation.
func newSessionQuality() *sessionQuality {
	return &sessionQuality{qi: metrics.NewQualityIndicator()}
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

// OnSessionMeasurement records a measurement (rttMs, lossPct) on the named
// session's QualityIndicator (BC-2.06.001 v1.7 PC-2/PC-3/PC-4).
//
// Returns ErrSessionNotFound if sessionName is not in the live set (E-SES-001
// equivalent). The observed flag transitions false → true on first call for
// this session, so subsequent SessionSnapshots renders the real quality band
// instead of "pending".
func (p *Publisher) OnSessionMeasurement(sessionName string, rttMs, lossPct float64) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	sq, ok := p.qualities[sessionName]
	if !ok {
		return ErrSessionNotFound
	}
	sq.qi.Update(rttMs, lossPct)
	sq.observed = true
	return nil
}

// OnSessionMissingFrame records one missing-frame event on the named session's
// QualityIndicator (BC-2.06.002 v1.4 PC-1..PC-3). The lifetime miss counter
// increments once per call, independent of hysteresis state.
//
// Returns ErrSessionNotFound if sessionName is not in the live set.
// The observed flag transitions false → true on first call for this session.
func (p *Publisher) OnSessionMissingFrame(sessionName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	sq, ok := p.qualities[sessionName]
	if !ok {
		return ErrSessionNotFound
	}
	sq.qi.OnMissingFrame()
	sq.observed = true
	return nil
}

// SessionSnapshot returns the operator-visible snapshot for a single named
// session. Returns (zero, false) when sessionName is not in the live set
// (Go-idiomatic map-lookup shape per go.md rule 12).
//
// BC-2.06.001 v1.7 PC-5 — quality surfaced via `sbctl sessions status`.
func (p *Publisher) SessionSnapshot(sessionName string) (SessionSnapshot, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	info, ok := p.sessions[sessionName]
	if !ok {
		return SessionSnapshot{}, false
	}
	sq, ok := p.qualities[sessionName]
	if !ok {
		// Invariant violation: sessions and qualities maps must stay in sync
		// (Publish adds to both; Unpublish removes from both). Treat as
		// not-found for the caller — the daemon-side handler will emit
		// E-SES-001 either way.
		return SessionSnapshot{}, false
	}
	return SessionSnapshot{
		Name:        info.Name,
		PublishedAt: info.PublishedAt,
		Quality:     qualityString(sq.qi.Current(), sq.observed),
		MissCount:   sq.qi.MissCount(),
	}, true
}

// SessionSnapshots returns an operator-visible snapshot of all currently
// published sessions, sorted alphabetically by name (VP-031). The returned
// slice is a value copy — mutations do not affect the Publisher's internal
// state (go.md rule 12).
//
// BC-2.06.001 v1.7 PC-5 console-half; DRIFT-001b closure.
func (p *Publisher) SessionSnapshots() []SessionSnapshot {
	p.mu.RLock()
	defer p.mu.RUnlock()

	out := make([]SessionSnapshot, 0, len(p.sessions))
	for name, info := range p.sessions {
		sq, ok := p.qualities[name]
		if !ok {
			// Skip on invariant violation rather than emit an incomplete
			// snapshot; ListSessions still shows the raw session presence.
			continue
		}
		out = append(out, SessionSnapshot{
			Name:        info.Name,
			PublishedAt: info.PublishedAt,
			Quality:     qualityString(sq.qi.Current(), sq.observed),
			MissCount:   sq.qi.MissCount(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}
