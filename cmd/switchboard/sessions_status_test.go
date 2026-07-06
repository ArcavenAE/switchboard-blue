package main

// Tests for sessionQualitySource.HandleSessionsStatus — the transport-agnostic
// handler behind the sessions.status RPC (S-BL.CONSOLE-OBS;
// BC-2.06.001 v1.7 PC-5 console-half; BC-2.06.002 v1.4 PC-3 operator export;
// DRIFT-001b + DRIFT-002 closures).
//
// The handler renders SessionSnapshot values as JSON-tagged
// SessionStatusEntry rows. Two dispatch shapes:
//   - req.SessionName == "" → all sessions, sorted alphabetically.
//   - req.SessionName != "" → exactly one session, or E-SES-001.
//
// The e2e tests in sessions_handlers_e2e_test.go cover Tier-2 admission and
// the JSON-RPC envelope; the tests here cover the pure handler contract:
// slice shape, ordering, quality/miss_count values, JSON wire fields.

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/session"
)

// newHandlerTestSource is the sessions_status_test-local factory: publishes
// the requested names on a fresh Publisher wired to a fresh source so the
// handler can be invoked against real (Publisher-seeded) registry state.
// Returns the wired Publisher for tests that need to Unpublish or observe.
func newHandlerTestSource(t *testing.T, names ...string) (*session.Publisher, *sessionQualitySource) {
	t.Helper()
	pub := session.NewPublisher(admission.NewAdmittedKeySet())
	src := newSessionQualitySourceFromPublisher(pub)
	for _, n := range names {
		if err := pub.Publish(n); err != nil {
			t.Fatalf("Publish %q: %v", n, err)
		}
	}
	return pub, src
}

// TestHandleSessionsStatus_Empty_NoSessions verifies that with no sessions
// published, the handler returns a zero-length Sessions slice and no error —
// distinguishing "no data" from "error" per the operator surface contract.
func TestHandleSessionsStatus_Empty_NoSessions(t *testing.T) {
	t.Parallel()
	_, src := newHandlerTestSource(t)

	resp, err := src.HandleSessionsStatus(context.Background(), SessionsStatusRequest{})
	if err != nil {
		t.Fatalf("HandleSessionsStatus: %v", err)
	}
	if len(resp.Sessions) != 0 {
		t.Errorf("Sessions: got %d entries; want 0", len(resp.Sessions))
	}
}

// TestHandleSessionsStatus_AllSessions_SortedByName verifies that the "all
// sessions" query returns entries alphabetically sorted by name. Deterministic
// output is required for operator display (VP-031).
func TestHandleSessionsStatus_AllSessions_SortedByName(t *testing.T) {
	t.Parallel()
	// Publish in non-alphabetical order to prove sorting is not accidental.
	_, src := newHandlerTestSource(t, "charlie", "alpha", "bravo")

	resp, err := src.HandleSessionsStatus(context.Background(), SessionsStatusRequest{})
	if err != nil {
		t.Fatalf("HandleSessionsStatus: %v", err)
	}
	if len(resp.Sessions) != 3 {
		t.Fatalf("Sessions length = %d; want 3", len(resp.Sessions))
	}
	want := []string{"alpha", "bravo", "charlie"}
	for i, w := range want {
		if resp.Sessions[i].Name != w {
			t.Errorf("Sessions[%d].Name = %q; want %q", i, resp.Sessions[i].Name, w)
		}
	}
}

// TestHandleSessionsStatus_AllSessions_QualityAndMissCount verifies that each
// entry carries the current quality band and lifetime miss count.
// Non-tautological: seeds two sessions with divergent observation histories
// (agent-01 gets a measurement + 3 misses → yellow+3; agent-02 gets no
// observations → pending+0). If the handler wired the wrong field or
// short-circuited to a constant, this test fails.
func TestHandleSessionsStatus_AllSessions_QualityAndMissCount(t *testing.T) {
	t.Parallel()
	_, src := newHandlerTestSource(t, "agent-01", "agent-02")

	// agent-01: green measurement, then 3 misses → yellow + missCount 3.
	if err := src.OnSessionMeasurement("agent-01", 50, 1); err != nil {
		t.Fatalf("OnSessionMeasurement: %v", err)
	}
	for i := 0; i < 3; i++ {
		if err := src.OnSessionMissingFrame("agent-01"); err != nil {
			t.Fatalf("OnSessionMissingFrame %d: %v", i, err)
		}
	}
	// agent-02: no observations → pending + missCount 0.

	resp, err := src.HandleSessionsStatus(context.Background(), SessionsStatusRequest{})
	if err != nil {
		t.Fatalf("HandleSessionsStatus: %v", err)
	}
	if len(resp.Sessions) != 2 {
		t.Fatalf("Sessions length = %d; want 2", len(resp.Sessions))
	}

	// Sort order is alphabetical, so agent-01 first.
	first := resp.Sessions[0]
	if first.Name != "agent-01" {
		t.Fatalf("Sessions[0].Name = %q; want agent-01", first.Name)
	}
	if first.Quality != "yellow" {
		t.Errorf("agent-01 Quality = %q; want yellow", first.Quality)
	}
	if first.MissCount != 3 {
		t.Errorf("agent-01 MissCount = %d; want 3", first.MissCount)
	}

	second := resp.Sessions[1]
	if second.Name != "agent-02" {
		t.Fatalf("Sessions[1].Name = %q; want agent-02", second.Name)
	}
	if second.Quality != "pending" {
		t.Errorf("agent-02 Quality = %q; want pending "+
			"(no observations recorded — brand-new session must not be reported as green)",
			second.Quality)
	}
	if second.MissCount != 0 {
		t.Errorf("agent-02 MissCount = %d; want 0", second.MissCount)
	}
}

// TestHandleSessionsStatus_SingleSession_ByName verifies that a non-empty
// SessionName returns exactly one entry — the named session.
func TestHandleSessionsStatus_SingleSession_ByName(t *testing.T) {
	t.Parallel()
	_, src := newHandlerTestSource(t, "agent-01", "agent-02", "agent-03")

	if err := src.OnSessionMeasurement("agent-02", 50, 1); err != nil {
		t.Fatalf("OnSessionMeasurement: %v", err)
	}

	resp, err := src.HandleSessionsStatus(
		context.Background(),
		SessionsStatusRequest{SessionName: "agent-02"},
	)
	if err != nil {
		t.Fatalf("HandleSessionsStatus: %v", err)
	}
	if len(resp.Sessions) != 1 {
		t.Fatalf("Sessions length = %d; want 1", len(resp.Sessions))
	}
	if resp.Sessions[0].Name != "agent-02" {
		t.Errorf("Sessions[0].Name = %q; want agent-02", resp.Sessions[0].Name)
	}
	if resp.Sessions[0].Quality != "green" {
		t.Errorf("Sessions[0].Quality = %q; want green", resp.Sessions[0].Quality)
	}
}

// TestHandleSessionsStatus_SingleSession_Unknown_ESES001 verifies that a
// SessionName that is not in the registry returns E-SES-001 wrapped
// around errQualitySessionNotFound. The error message must name the missing
// session so operators can copy-paste it into a session-lookup command.
func TestHandleSessionsStatus_SingleSession_Unknown_ESES001(t *testing.T) {
	t.Parallel()
	_, src := newHandlerTestSource(t)

	resp, err := src.HandleSessionsStatus(
		context.Background(),
		SessionsStatusRequest{SessionName: "does-not-exist"},
	)
	if err == nil {
		t.Fatalf("HandleSessionsStatus(unknown): err = nil; want E-SES-001")
	}
	if !errors.Is(err, errQualitySessionNotFound) {
		t.Errorf("err = %v; want errQualitySessionNotFound wrapped", err)
	}
	if !strings.Contains(err.Error(), "E-SES-001") {
		t.Errorf("err message = %q; want it to contain %q", err.Error(), "E-SES-001")
	}
	if !strings.Contains(err.Error(), "does-not-exist") {
		t.Errorf("err message = %q; want it to name the missing session %q",
			err.Error(), "does-not-exist")
	}
	// Response value on error path is the zero response — Sessions must be
	// nil (no partial data leaked).
	if resp.Sessions != nil {
		t.Errorf("Sessions on error path = %+v; want nil", resp.Sessions)
	}
}

// TestHandleSessionsStatus_JSONRoundTrip verifies that a successful response
// serialises with the operator-facing JSON field names locked by
// BC-2.06.001 v1.7 PC-5 and BC-2.06.002 v1.4 PC-3: name, published_at,
// quality, miss_count, sessions.
//
// Non-tautological: seeds one miss (below HysteresisCount) so quality stays
// green while miss_count == 1 — proves the handler is emitting the *live*
// state and not a canned response.
func TestHandleSessionsStatus_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	_, src := newHandlerTestSource(t, "agent-01")
	if err := src.OnSessionMeasurement("agent-01", 50, 1); err != nil {
		t.Fatalf("OnSessionMeasurement: %v", err)
	}
	if err := src.OnSessionMissingFrame("agent-01"); err != nil {
		t.Fatalf("OnSessionMissingFrame: %v", err)
	}

	resp, err := src.HandleSessionsStatus(context.Background(), SessionsStatusRequest{})
	if err != nil {
		t.Fatalf("HandleSessionsStatus: %v", err)
	}
	buf, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	got := string(buf)

	// Operator-facing wire field names (BC-2.06.003 v1.16-locked).
	for _, need := range []string{
		`"sessions"`,
		`"name":"agent-01"`,
		`"published_at"`,
		`"quality":"green"`, // 1 miss < HysteresisCount ⇒ still green.
		`"miss_count":1`,    // lifetime counter — reflects the one miss.
	} {
		if !strings.Contains(got, need) {
			t.Errorf("JSON = %s\nmust contain %s", got, need)
		}
	}
}

// TestHandleSessionsStatus_AfterUnpublish_DropsFromAllQuery verifies that
// Unpublish removes a session from the "all sessions" projection. This
// exercises the OnUnpublished hook wiring end-to-end through the handler.
func TestHandleSessionsStatus_AfterUnpublish_DropsFromAllQuery(t *testing.T) {
	t.Parallel()
	pub, src := newHandlerTestSource(t, "agent-01", "agent-02")

	if err := pub.Unpublish("agent-01"); err != nil {
		t.Fatalf("Unpublish: %v", err)
	}

	resp, err := src.HandleSessionsStatus(context.Background(), SessionsStatusRequest{})
	if err != nil {
		t.Fatalf("HandleSessionsStatus: %v", err)
	}
	if len(resp.Sessions) != 1 {
		t.Fatalf("Sessions length = %d; want 1 (agent-01 unpublished)", len(resp.Sessions))
	}
	if resp.Sessions[0].Name != "agent-02" {
		t.Errorf("Sessions[0].Name = %q; want agent-02", resp.Sessions[0].Name)
	}
}
