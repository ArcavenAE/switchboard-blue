package session_test

// Tests for the Publisher.HandleSessionsStatus RPC handler
// (S-BL.CONSOLE-OBS; BC-2.06.001 v1.7 PC-5 console-half; BC-2.06.002 v1.4 PC-3;
// DRIFT-001b + DRIFT-002).
//
// The handler answers two shapes:
//   sessions.status {}                  → all sessions, sorted by name
//   sessions.status {session_name:"X"}  → single session; E-SES-001 if unknown
//
// Snapshot values are Quality string ({green|yellow|red|pending}) and MissCount
// uint64. Wire shape is the SessionsStatusResponse type; these tests verify
// both the semantics and the transport-relevant fields (name, published_at,
// quality, miss_count) round-trip through JSON.

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/session"
)

// TestPublisher_HandleSessionsStatus_EmptyDaemon verifies that on a fresh
// daemon with no published sessions, sessions.status {} returns an empty
// list — not an error (BC-2.06.001 PC-5 console-half: legitimate empty state).
func TestPublisher_HandleSessionsStatus_EmptyDaemon(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	resp, err := p.HandleSessionsStatus(context.Background(), session.SessionsStatusRequest{})
	if err != nil {
		t.Fatalf("HandleSessionsStatus on empty daemon: err = %v; want nil", err)
	}
	if len(resp.Sessions) != 0 {
		t.Errorf("HandleSessionsStatus on empty daemon: Sessions len = %d; want 0",
			len(resp.Sessions))
	}
}

// TestPublisher_HandleSessionsStatus_AllSessions_SortedAndPending verifies
// that the "all" query returns every published session, sorted alphabetically
// by name, with quality "pending" and miss_count 0 for brand-new sessions
// (BC-2.06.001 PC-5; per-session pending semantics).
func TestPublisher_HandleSessionsStatus_AllSessions_SortedAndPending(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	for _, name := range []string{"charlie", "alpha", "bravo"} {
		if err := p.Publish(name); err != nil {
			t.Fatalf("Publish %q: %v", name, err)
		}
	}

	resp, err := p.HandleSessionsStatus(context.Background(), session.SessionsStatusRequest{})
	if err != nil {
		t.Fatalf("HandleSessionsStatus: %v", err)
	}
	if len(resp.Sessions) != 3 {
		t.Fatalf("Sessions len = %d; want 3", len(resp.Sessions))
	}
	wantNames := []string{"alpha", "bravo", "charlie"}
	for i, e := range resp.Sessions {
		if e.Name != wantNames[i] {
			t.Errorf("Sessions[%d].Name = %q; want %q", i, e.Name, wantNames[i])
		}
		if e.Quality != "pending" {
			t.Errorf("Sessions[%d].Quality = %q; want %q "+
				"(brand-new sessions must be pending)", i, e.Quality, "pending")
		}
		if e.MissCount != 0 {
			t.Errorf("Sessions[%d].MissCount = %d; want 0", i, e.MissCount)
		}
		if e.PublishedAt.IsZero() {
			t.Errorf("Sessions[%d].PublishedAt is zero; want non-zero UTC", i)
		}
		if e.PublishedAt.Location() != time.UTC {
			t.Errorf("Sessions[%d].PublishedAt location = %v; want UTC",
				i, e.PublishedAt.Location())
		}
	}
}

// TestPublisher_HandleSessionsStatus_SingleSession_Green verifies that the
// "by name" query returns exactly one entry for a session with a green
// measurement observed (Quality "green", MissCount 0).
func TestPublisher_HandleSessionsStatus_SingleSession_Green(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	if err := p.Publish("agent-01"); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	// Establish green baseline (moves quality out of pending).
	if err := p.OnSessionMeasurement("agent-01", 50, 1); err != nil {
		t.Fatalf("OnSessionMeasurement: %v", err)
	}

	resp, err := p.HandleSessionsStatus(
		context.Background(),
		session.SessionsStatusRequest{SessionName: "agent-01"},
	)
	if err != nil {
		t.Fatalf("HandleSessionsStatus: %v", err)
	}
	if len(resp.Sessions) != 1 {
		t.Fatalf("Sessions len = %d; want 1", len(resp.Sessions))
	}
	got := resp.Sessions[0]
	if got.Name != "agent-01" {
		t.Errorf("Name = %q; want %q", got.Name, "agent-01")
	}
	if got.Quality != "green" {
		t.Errorf("Quality = %q; want %q", got.Quality, "green")
	}
	if got.MissCount != 0 {
		t.Errorf("MissCount = %d; want 0", got.MissCount)
	}
}

// TestPublisher_HandleSessionsStatus_SingleSession_YellowWithMisses verifies
// that when a session has taken enough misses to downgrade from green to
// yellow, the response reflects Quality "yellow" and MissCount 3.
//
// Trace: BC-2.06.001 PC-5 (quality surface); BC-2.06.002 PC-3 (miss export);
// BC-2.06.002 PC-2 (three consecutive misses degrade one level).
func TestPublisher_HandleSessionsStatus_SingleSession_YellowWithMisses(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	if err := p.Publish("agent-01"); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if err := p.OnSessionMeasurement("agent-01", 50, 1); err != nil {
		t.Fatalf("OnSessionMeasurement: %v", err)
	}
	for i := 0; i < 3; i++ {
		if err := p.OnSessionMissingFrame("agent-01"); err != nil {
			t.Fatalf("OnSessionMissingFrame %d: %v", i, err)
		}
	}

	resp, err := p.HandleSessionsStatus(
		context.Background(),
		session.SessionsStatusRequest{SessionName: "agent-01"},
	)
	if err != nil {
		t.Fatalf("HandleSessionsStatus: %v", err)
	}
	if len(resp.Sessions) != 1 {
		t.Fatalf("Sessions len = %d; want 1", len(resp.Sessions))
	}
	got := resp.Sessions[0]
	if got.Quality != "yellow" {
		t.Errorf("Quality = %q; want %q "+
			"(3 misses on green baseline ⇒ yellow per BC-2.06.002 PC-2)",
			got.Quality, "yellow")
	}
	if got.MissCount != 3 {
		t.Errorf("MissCount = %d; want 3", got.MissCount)
	}
}

// TestPublisher_HandleSessionsStatus_UnknownSession_ESES001 verifies that
// querying a session name not in the live set returns E-SES-001 wrapping
// ErrSessionNotFound.
func TestPublisher_HandleSessionsStatus_UnknownSession_ESES001(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	_, err := p.HandleSessionsStatus(
		context.Background(),
		session.SessionsStatusRequest{SessionName: "does-not-exist"},
	)
	if err == nil {
		t.Fatal("HandleSessionsStatus on unknown session: err = nil; want error")
	}
	if !errors.Is(err, session.ErrSessionNotFound) {
		t.Errorf("err chain does not wrap ErrSessionNotFound: %v", err)
	}
	if got := err.Error(); !contains(got, "E-SES-001") {
		t.Errorf("err message = %q; want to include %q", got, "E-SES-001")
	}
	if got := err.Error(); !contains(got, "does-not-exist") {
		t.Errorf("err message = %q; want to include session name %q",
			got, "does-not-exist")
	}
}

// TestPublisher_HandleSessionsStatus_JSONRoundTrip verifies that the response
// marshals to JSON with the exact field names sbctl consumes (name,
// published_at, quality, miss_count) and unmarshals back to an equivalent
// struct. This is the wire-contract test.
func TestPublisher_HandleSessionsStatus_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	if err := p.Publish("agent-01"); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if err := p.OnSessionMissingFrame("agent-01"); err != nil {
		t.Fatalf("OnSessionMissingFrame: %v", err)
	}

	resp, err := p.HandleSessionsStatus(context.Background(), session.SessionsStatusRequest{})
	if err != nil {
		t.Fatalf("HandleSessionsStatus: %v", err)
	}

	// Marshal to JSON — sbctl consumes exactly these field names.
	buf, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	got := string(buf)
	// Note: a miss counts as an observation (observed=true), so the underlying
	// QualityIndicator's classification is surfaced instead of "pending". One
	// miss on a fresh (Green) indicator is below HysteresisCount, so the
	// classification stays at Green while MissCount records the lifetime event.
	for _, want := range []string{
		`"sessions"`,
		`"name":"agent-01"`,
		`"published_at":`,
		`"quality":"green"`,
		`"miss_count":1`,
	} {
		if !contains(got, want) {
			t.Errorf("JSON output missing %q; full output: %s", want, got)
		}
	}

	// Round-trip: unmarshal back.
	var back session.SessionsStatusResponse
	if err := json.Unmarshal(buf, &back); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if len(back.Sessions) != 1 {
		t.Fatalf("round-trip Sessions len = %d; want 1", len(back.Sessions))
	}
	if back.Sessions[0].Name != "agent-01" {
		t.Errorf("round-trip Name = %q; want %q", back.Sessions[0].Name, "agent-01")
	}
	if back.Sessions[0].Quality != "green" {
		t.Errorf("round-trip Quality = %q; want %q",
			back.Sessions[0].Quality, "green")
	}
	if back.Sessions[0].MissCount != 1 {
		t.Errorf("round-trip MissCount = %d; want 1", back.Sessions[0].MissCount)
	}
}

// TestPublisher_HandleSessionsStatus_EmptySessionNameSelectsAll verifies that
// an explicit empty SessionName ("") is treated as the "all" query, not as
// a zero-value key lookup that would spuriously match nothing.
func TestPublisher_HandleSessionsStatus_EmptySessionNameSelectsAll(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	for _, name := range []string{"alpha", "bravo"} {
		if err := p.Publish(name); err != nil {
			t.Fatalf("Publish %q: %v", name, err)
		}
	}

	resp, err := p.HandleSessionsStatus(
		context.Background(),
		session.SessionsStatusRequest{SessionName: ""},
	)
	if err != nil {
		t.Fatalf("HandleSessionsStatus with empty SessionName: %v", err)
	}
	if len(resp.Sessions) != 2 {
		t.Errorf("Sessions len = %d; want 2", len(resp.Sessions))
	}
}

// contains is a tiny helper because testing/internal/testenv is not exported.
// Prefer strings.Contains would work equally — this shape keeps the imports
// minimal at the test-file level.
func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
