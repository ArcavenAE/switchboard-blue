// Package session tests for the per-session quality-indicator surface
// (S-BL.CONSOLE-OBS; BC-2.06.001 v1.7 PC-5 console-half; BC-2.06.002 v1.4 PC-3;
// DRIFT-001b + DRIFT-002 closures).
//
// The Publisher owns a parallel map of per-session *metrics.QualityIndicator
// values: created on Publish, dropped on Unpublish. OnSessionMeasurement and
// OnSessionMissingFrame route observations to the indicator for the named
// session. SessionSnapshots and SessionSnapshot expose the operator-visible
// {Name, PublishedAt, Quality, MissCount} tuple for `sbctl sessions status`.
//
// Quality enum returned in snapshots: "green" | "yellow" | "red" | "pending"
// per BC-2.06.003 v1.16 (locked by row_e_failed_and_pending; "failed" never
// appears as a quality value). "pending" means the indicator has received no
// observations yet — distinct from EC-008's empty-paths pending because it is
// per-session, not per-node.
package session_test

import (
	"errors"
	"sort"
	"sync"
	"testing"

	"github.com/arcavenae/switchboard/internal/session"
)

// TestPublisher_SessionSnapshots_EmptyOnStartup verifies that a fresh Publisher
// has no session snapshots (BC-2.06.001 PC-5 console-half — nothing to show).
func TestPublisher_SessionSnapshots_EmptyOnStartup(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	if got := p.SessionSnapshots(); len(got) != 0 {
		t.Errorf("fresh Publisher: SessionSnapshots length = %d, want 0", len(got))
	}
}

// TestPublisher_SessionSnapshots_PublishedSessionAppearsPending verifies that a
// newly-published session appears in SessionSnapshots with quality "pending"
// and miss_count 0 before any observation has been recorded.
//
// BC-2.06.001 v1.7 PC-5 — quality surfaced in the console session list view.
// Per-session "pending" semantics: no observations yet ⇒ indeterminate.
func TestPublisher_SessionSnapshots_PublishedSessionAppearsPending(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	if err := p.Publish("agent-01"); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	snaps := p.SessionSnapshots()
	if len(snaps) != 1 {
		t.Fatalf("SessionSnapshots: got %d entries, want 1", len(snaps))
	}
	if snaps[0].Name != "agent-01" {
		t.Errorf("snapshot Name = %q, want %q", snaps[0].Name, "agent-01")
	}
	if snaps[0].Quality != "pending" {
		t.Errorf("snapshot Quality = %q, want %q "+
			"(brand-new session must be pending until first observation; BC-2.06.001 PC-5)",
			snaps[0].Quality, "pending")
	}
	if snaps[0].MissCount != 0 {
		t.Errorf("snapshot MissCount = %d, want 0", snaps[0].MissCount)
	}
	if snaps[0].PublishedAt.IsZero() {
		t.Errorf("snapshot PublishedAt is zero; want non-zero UTC timestamp")
	}
}

// TestPublisher_OnSessionMeasurement_GoodMeasurementProducesGreen verifies that
// a green-range measurement on a published session transitions its quality to
// "green" (from "pending"), while leaving MissCount untouched.
//
// BC-2.06.001 PC-2 (Green: RTT ≤ 100 ms AND loss ≤ 5 %); PC-5 (surfaced).
func TestPublisher_OnSessionMeasurement_GoodMeasurementProducesGreen(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	if err := p.Publish("agent-01"); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if err := p.OnSessionMeasurement("agent-01", 50, 1); err != nil {
		t.Fatalf("OnSessionMeasurement: %v", err)
	}

	snap, ok := p.SessionSnapshot("agent-01")
	if !ok {
		t.Fatalf("SessionSnapshot(agent-01) = _, false; want ok")
	}
	if snap.Quality != "green" {
		t.Errorf("Quality after green measurement = %q, want %q",
			snap.Quality, "green")
	}
	if snap.MissCount != 0 {
		t.Errorf("MissCount after green measurement = %d, want 0", snap.MissCount)
	}
}

// TestPublisher_OnSessionMissingFrame_IncrementsMissCount verifies that
// OnSessionMissingFrame increments the per-session lifetime miss counter by
// exactly one, independent of any downgrade event.
//
// BC-2.06.002 v1.4 PC-3 — lifetime path-metric record of gap events.
func TestPublisher_OnSessionMissingFrame_IncrementsMissCount(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	if err := p.Publish("agent-01"); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// Cross the hysteresis threshold to prove the lifetime counter keeps
	// accumulating past the internal-reset boundary.
	const calls = 5
	for i := 0; i < calls; i++ {
		if err := p.OnSessionMissingFrame("agent-01"); err != nil {
			t.Fatalf("OnSessionMissingFrame call %d: %v", i, err)
		}
	}

	snap, ok := p.SessionSnapshot("agent-01")
	if !ok {
		t.Fatalf("SessionSnapshot(agent-01) = _, false; want ok")
	}
	if snap.MissCount != uint64(calls) {
		t.Errorf("MissCount after %d OnSessionMissingFrame calls = %d, want %d",
			calls, snap.MissCount, calls)
	}
}

// TestPublisher_OnSessionMissingFrame_DowngradesQuality verifies that
// three consecutive missing frames downgrade the per-session quality from
// green (established via a prior measurement) to yellow. The lifetime miss
// count reflects all three events.
//
// BC-2.06.002 PC-2 — degrades one level per HysteresisCount consecutive misses.
func TestPublisher_OnSessionMissingFrame_DowngradesQuality(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	if err := p.Publish("agent-01"); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	// Establish green baseline (moves quality out of pending).
	if err := p.OnSessionMeasurement("agent-01", 50, 1); err != nil {
		t.Fatalf("OnSessionMeasurement: %v", err)
	}

	// HysteresisCount consecutive misses ⇒ green → yellow.
	for i := 0; i < 3; i++ {
		if err := p.OnSessionMissingFrame("agent-01"); err != nil {
			t.Fatalf("OnSessionMissingFrame %d: %v", i, err)
		}
	}

	snap, ok := p.SessionSnapshot("agent-01")
	if !ok {
		t.Fatalf("SessionSnapshot(agent-01) = _, false; want ok")
	}
	if snap.Quality != "yellow" {
		t.Errorf("Quality after 3 misses on green baseline = %q, want %q",
			snap.Quality, "yellow")
	}
	if snap.MissCount != 3 {
		t.Errorf("MissCount = %d, want 3", snap.MissCount)
	}
}

// TestPublisher_OnSessionMeasurement_UnknownSession_ErrSessionNotFound verifies
// that an observation on an unknown session name returns ErrSessionNotFound
// so the caller can surface E-SES-001 to the operator.
func TestPublisher_OnSessionMeasurement_UnknownSession_ErrSessionNotFound(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	err := p.OnSessionMeasurement("does-not-exist", 50, 1)
	if !errors.Is(err, session.ErrSessionNotFound) {
		t.Errorf("OnSessionMeasurement on unknown session: got %v, want ErrSessionNotFound", err)
	}
}

// TestPublisher_OnSessionMissingFrame_UnknownSession_ErrSessionNotFound mirrors
// the measurement-path check for the missing-frame path.
func TestPublisher_OnSessionMissingFrame_UnknownSession_ErrSessionNotFound(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	err := p.OnSessionMissingFrame("does-not-exist")
	if !errors.Is(err, session.ErrSessionNotFound) {
		t.Errorf("OnSessionMissingFrame on unknown session: got %v, want ErrSessionNotFound", err)
	}
}

// TestPublisher_SessionSnapshot_UnknownSession verifies that SessionSnapshot
// returns (zero, false) for an unknown name — Go-idiomatic (T, bool)
// signature per go.md rule 12.
func TestPublisher_SessionSnapshot_UnknownSession(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	snap, ok := p.SessionSnapshot("does-not-exist")
	if ok {
		t.Errorf("SessionSnapshot on unknown session: got ok=true, want false")
	}
	if snap != (session.SessionSnapshot{}) {
		t.Errorf("SessionSnapshot on unknown session: got %+v, want zero value", snap)
	}
}

// TestPublisher_Unpublish_DropsQualityIndicator verifies that Unpublish removes
// the session's QualityIndicator from the internal map so it is not surfaced
// in subsequent SessionSnapshots. A follow-up Publish of the same name starts
// a fresh indicator (quality "pending", miss_count 0).
//
// Non-tautological: publishes name, records a measurement + a miss, unpublishes,
// then re-publishes and asserts the new snapshot is clean.
func TestPublisher_Unpublish_DropsQualityIndicator(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	if err := p.Publish("agent-01"); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if err := p.OnSessionMeasurement("agent-01", 50, 1); err != nil {
		t.Fatalf("OnSessionMeasurement: %v", err)
	}
	if err := p.OnSessionMissingFrame("agent-01"); err != nil {
		t.Fatalf("OnSessionMissingFrame: %v", err)
	}
	if err := p.Unpublish("agent-01"); err != nil {
		t.Fatalf("Unpublish: %v", err)
	}

	// After Unpublish the snapshot must be gone entirely.
	if _, ok := p.SessionSnapshot("agent-01"); ok {
		t.Errorf("SessionSnapshot after Unpublish: got ok=true, want false")
	}

	// Re-publish: fresh indicator with pending quality and zero miss count.
	if err := p.Publish("agent-01"); err != nil {
		t.Fatalf("re-Publish: %v", err)
	}
	snap, ok := p.SessionSnapshot("agent-01")
	if !ok {
		t.Fatalf("SessionSnapshot after re-Publish: got ok=false, want true")
	}
	if snap.Quality != "pending" {
		t.Errorf("Quality after re-Publish = %q, want %q "+
			"(indicator must be reset — no state carried across publish cycles)",
			snap.Quality, "pending")
	}
	if snap.MissCount != 0 {
		t.Errorf("MissCount after re-Publish = %d, want 0", snap.MissCount)
	}
}

// TestPublisher_SessionSnapshots_SortedByName verifies that SessionSnapshots
// returns entries alphabetically sorted by name, matching ListSessions'
// ordering convention (VP-031; deterministic output for operator display).
func TestPublisher_SessionSnapshots_SortedByName(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	// Publish in non-alphabetical order to prove sorting is not accidental.
	for _, name := range []string{"charlie", "alpha", "bravo"} {
		if err := p.Publish(name); err != nil {
			t.Fatalf("Publish %q: %v", name, err)
		}
	}

	snaps := p.SessionSnapshots()
	got := make([]string, len(snaps))
	for i, s := range snaps {
		got[i] = s.Name
	}

	want := []string{"alpha", "bravo", "charlie"}
	if !sort.StringsAreSorted(got) {
		t.Errorf("SessionSnapshots names not sorted: got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("SessionSnapshots[%d].Name = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestPublisher_SessionSnapshots_ValueCopy verifies that mutating the returned
// slice does not affect the Publisher's internal state (go.md rule 12: no
// internal pointer leak). Matches ListSessions' snapshot contract.
func TestPublisher_SessionSnapshots_ValueCopy(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	if err := p.Publish("agent-01"); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	snaps := p.SessionSnapshots()
	if len(snaps) != 1 {
		t.Fatalf("SessionSnapshots: got %d, want 1", len(snaps))
	}
	snaps[0].Name = "mutated"
	snaps[0].Quality = "red"
	snaps[0].MissCount = 999

	// Second call must return fresh values.
	snaps2 := p.SessionSnapshots()
	if snaps2[0].Name != "agent-01" {
		t.Errorf("mutation leaked to Publisher: Name = %q; want %q", snaps2[0].Name, "agent-01")
	}
	if snaps2[0].Quality != "pending" {
		t.Errorf("mutation leaked to Publisher: Quality = %q; want %q", snaps2[0].Quality, "pending")
	}
	if snaps2[0].MissCount != 0 {
		t.Errorf("mutation leaked to Publisher: MissCount = %d; want 0", snaps2[0].MissCount)
	}
}

// TestPublisher_ConcurrentObservations exercises OnSessionMeasurement,
// OnSessionMissingFrame, and SessionSnapshots concurrently across multiple
// sessions to expose data races under -race. The functional oracle is the
// exact miss-count total across the workload — the counter is serialisable.
func TestPublisher_ConcurrentObservations(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	const sessions = 4
	const missesPerSession = 250

	for i := 0; i < sessions; i++ {
		name := "agent-" + string(rune('a'+i))
		if err := p.Publish(name); err != nil {
			t.Fatalf("Publish %q: %v", name, err)
		}
	}

	var wg sync.WaitGroup
	// One miss-writer + one measurement-writer + one reader per session.
	wg.Add(sessions * 3)

	for i := 0; i < sessions; i++ {
		name := "agent-" + string(rune('a'+i))
		go func(n string) {
			defer wg.Done()
			for j := 0; j < missesPerSession; j++ {
				if err := p.OnSessionMissingFrame(n); err != nil {
					t.Errorf("OnSessionMissingFrame(%q): %v", n, err)
					return
				}
			}
		}(name)
		go func(n string) {
			defer wg.Done()
			for j := 0; j < missesPerSession; j++ {
				// Green-range measurements — these do NOT change the lifetime
				// miss counter (validated by the exact-count oracle below).
				if err := p.OnSessionMeasurement(n, 50, 1); err != nil {
					t.Errorf("OnSessionMeasurement(%q): %v", n, err)
					return
				}
			}
		}(name)
		go func(_ string) {
			defer wg.Done()
			for j := 0; j < missesPerSession; j++ {
				_ = p.SessionSnapshots()
			}
		}(name)
	}

	wg.Wait()

	// Exact-count oracle: each session must have received exactly
	// missesPerSession OnSessionMissingFrame calls, regardless of scheduling.
	for i := 0; i < sessions; i++ {
		name := "agent-" + string(rune('a'+i))
		snap, ok := p.SessionSnapshot(name)
		if !ok {
			t.Errorf("SessionSnapshot(%q): ok=false, want true", name)
			continue
		}
		if snap.MissCount != uint64(missesPerSession) {
			t.Errorf("MissCount(%q) = %d, want %d "+
				"(counter must be exact under concurrent workload)",
				name, snap.MissCount, missesPerSession)
		}
	}
}
