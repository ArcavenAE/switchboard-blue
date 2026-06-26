// Package session tests for BC-2.04.001 PC-2 (session publication state).
// Traces: BC-2.04.001 PC-2, PC-3, PC-4; ADR-010; ARCH-08 §6.6 position 6.
// Red Gate: all tests below are designed to fail against the stub (todo() panic).
package session_test

import (
	"errors"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/session"
)

// newTestPublisher is a test helper that builds a Publisher backed by an empty
// AdmittedKeySet. Returns both so callers can register keys for admission-gated
// publish tests (BC-2.04.001 precondition 3; S-3.03 SessionAuth).
//
//nolint:unparam // AdmittedKeySet return used by S-3.03 admission-gated tests; signature intentional
func newTestPublisher(t *testing.T) (*session.Publisher, *admission.AdmittedKeySet) {
	t.Helper()
	keys := admission.NewAdmittedKeySet()
	p := session.NewPublisher(keys)
	return p, keys
}

// TestPublisher_Publish_AddsSessionToLiveSet verifies that Publish records
// the session name with a UTC timestamp (BC-2.04.001 PC-2; S-3.01a AC-002).
func TestPublisher_Publish_AddsSessionToLiveSet(t *testing.T) {
	t.Parallel()
	p, keys := newTestPublisher(t)
	_ = keys // available for S-3.03 admission-gated publish tests

	before := time.Now().UTC()
	if err := p.Publish("agent-01"); err != nil {
		t.Fatalf("Publish: unexpected error: %v", err)
	}
	after := time.Now().UTC()

	info, err := p.Get("agent-01")
	if err != nil {
		t.Fatalf("Get: unexpected error: %v", err)
	}
	if info.Name != "agent-01" {
		t.Errorf("Name = %q; want %q", info.Name, "agent-01")
	}
	if info.PublishedAt.Before(before) || info.PublishedAt.After(after) {
		t.Errorf("PublishedAt = %v; want between %v and %v", info.PublishedAt, before, after)
	}
}

// TestPublisher_Unpublish_RemovesFromLiveSet verifies that Unpublish removes a
// previously published session (BC-2.04.001 PC-4; S-3.01a AC-003).
func TestPublisher_Unpublish_RemovesFromLiveSet(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	if err := p.Publish("build"); err != nil {
		t.Fatalf("Publish: unexpected error: %v", err)
	}
	if err := p.Unpublish("build"); err != nil {
		t.Fatalf("Unpublish: unexpected error: %v", err)
	}

	_, err := p.Get("build")
	if !errors.Is(err, session.ErrSessionNotFound) {
		t.Errorf("Get after Unpublish: got %v; want ErrSessionNotFound", err)
	}
}

// TestPublisher_Unpublish_ErrSessionNotFound verifies that Unpublish returns
// ErrSessionNotFound for an unknown name (E-SES-001; BC-2.04.001).
func TestPublisher_Unpublish_ErrSessionNotFound(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	err := p.Unpublish("does-not-exist")
	if !errors.Is(err, session.ErrSessionNotFound) {
		t.Errorf("Unpublish missing: got %v; want ErrSessionNotFound", err)
	}
}

// TestPublisher_Publish_DuplicateReturnsAlreadyPublished verifies that
// publishing the same name twice returns ErrSessionAlreadyPublished
// (BC-2.04.001 invariant: canonical name uniqueness).
func TestPublisher_Publish_DuplicateReturnsAlreadyPublished(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	if err := p.Publish("agent-01"); err != nil {
		t.Fatalf("first Publish: unexpected error: %v", err)
	}
	err := p.Publish("agent-01")
	if !errors.Is(err, session.ErrSessionAlreadyPublished) {
		t.Errorf("second Publish: got %v; want ErrSessionAlreadyPublished", err)
	}
}

// TestPublisher_ListSessions_ReturnsSnapshot verifies that ListSessions returns
// all published sessions as a value copy (BC-2.04.001 PC-2; VP-031;
// ARCH-08 §6.6 rule 12: no internal pointer leak).
func TestPublisher_ListSessions_ReturnsSnapshot(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	for _, name := range []string{"agent-01", "agent-02", "build"} {
		if err := p.Publish(name); err != nil {
			t.Fatalf("Publish %q: %v", name, err)
		}
	}

	list := p.ListSessions()
	if len(list) != 3 {
		t.Fatalf("ListSessions: got %d sessions; want 3", len(list))
	}

	// Mutating the returned slice must not affect the publisher's internal state.
	list[0].Name = "mutated"
	list2 := p.ListSessions()
	if list2[0].Name == "mutated" {
		t.Error("ListSessions returned internal pointer; mutation leaked into publisher state")
	}
}

// TestPublisher_EmptyOnStartup verifies that a fresh publisher reports no
// sessions (BC-2.04.001 EC-003: tmux server has no sessions on startup).
func TestPublisher_EmptyOnStartup(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	list := p.ListSessions()
	if len(list) != 0 {
		t.Errorf("fresh Publisher: ListSessions returned %d sessions; want 0", len(list))
	}
}
