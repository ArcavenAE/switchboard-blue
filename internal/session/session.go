// Package session manages session publication state for the access node's
// downstream half-channel (ARCH-08 §6.6 position 6; BC-2.04.001; ADR-010).
//
// Classification: boundary (ARCH-09). This package owns session lifecycle
// state and coordinates downstream half-channel wiring. It has no I/O and
// does not spawn goroutines itself — those are the responsibility of the
// effectful layer (internal/tmux).
//
// Allowed internal imports: {frame, admission} per ARCH-08 §6.6.
// Forbidden: internal/routing, internal/tmux (circular).
package session

import (
	"errors"
	"sync"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/frame"
)

// ErrSessionNotFound is returned when an operation targets a named session
// that is not present in the publisher's live set (E-SES-001; BC-2.04.001).
var ErrSessionNotFound = errors.New("session not found")

// ErrSessionAlreadyPublished is returned when Publish is called for a session
// name that is already in the live set. The caller should treat duplicate
// publishes as a no-op or a bug (BC-2.04.001 PC-2).
var ErrSessionAlreadyPublished = errors.New("session: name already published")

// Info holds the canonical metadata for one tmux session known to the
// publisher. The Name field is the tmux session name and is the canonical
// session identifier per BC-2.04.001 invariant 3.
type Info struct {
	// Name is the tmux session name (canonical session identifier).
	Name string
	// PublishedAt is the UTC timestamp when this session was published.
	PublishedAt time.Time
}

// Publisher manages the set of published tmux sessions and gates publication
// against the admission key set (BC-2.04.001 PC-2; ARCH-08 §6.6 position 6).
//
// The zero value is not usable; construct with NewPublisher.
//
// Concurrency: Publisher is safe for concurrent use.
type Publisher struct {
	mu       sync.RWMutex              //nolint:unused // Red Gate stub — used post-implementation
	sessions map[string]Info           //nolint:unused // Red Gate stub — used post-implementation
	keys     *admission.AdmittedKeySet //nolint:unused // Red Gate stub — used post-implementation
}

// NewPublisher constructs a Publisher that checks publication admission using
// keys (BC-2.04.001 precondition 3; S-3.01a task 5).
//
// keys must not be nil.
func NewPublisher(keys *admission.AdmittedKeySet) *Publisher {
	todo() // TODO(S-3.01a): implement per BC-2.04.001 PC-2; init sessions map
	return nil
}

// Publish adds sessionName to the live set with the current UTC timestamp
// (BC-2.04.001 PC-2; PC-3).
//
// Returns ErrSessionAlreadyPublished if name is already present.
func (p *Publisher) Publish(sessionName string) error {
	todo() // TODO(S-3.01a): implement per BC-2.04.001 PC-2
	return nil
}

// Unpublish removes sessionName from the live set (BC-2.04.001 PC-4).
//
// Returns ErrSessionNotFound if the session is not in the live set.
func (p *Publisher) Unpublish(sessionName string) error {
	todo() // TODO(S-3.01a): implement per BC-2.04.001 PC-4
	return nil
}

// ListSessions returns a snapshot of all currently published sessions, ordered
// alphabetically by name (BC-2.04.001 PC-2; VP-031).
//
// The returned slice is a value copy — mutations do not affect the Publisher's
// internal state (ARCH-08 §6.6 rule 12: no internal pointer leak).
func (p *Publisher) ListSessions() []Info {
	todo() // TODO(S-3.01a): implement per BC-2.04.001 PC-2; return sorted snapshot
	return nil
}

// Get returns the Info for sessionName, or ErrSessionNotFound if absent
// (E-SES-001; BC-2.04.001).
func (p *Publisher) Get(sessionName string) (Info, error) {
	todo() // TODO(S-3.01a): implement per BC-2.04.001; E-SES-001
	return Info{}, nil
}

// AdmittedKeySet exposes the underlying admission key set for read access by
// internal/tmux. Returns the same pointer passed to NewPublisher.
//
// The return type is a pointer to the concrete type, not an interface
// (Go rule 6: accept interfaces, return concrete types).
func (p *Publisher) AdmittedKeySet() *admission.AdmittedKeySet {
	todo() // TODO(S-3.01a): implement
	return nil
}

// FrameTypeData re-exports the canonical data frame type constant from
// internal/frame so that internal/tmux can reference it without importing
// internal/frame directly (ARCH-08 §6.6 tmux allowed imports: {halfchannel, session}).
//
// GREEN-BY-DESIGN: zero branching, no I/O, no helpers, 1 line.
const FrameTypeData = frame.FrameTypeData

// todo is a package-local helper that panics with an "not implemented" message.
// Its sole purpose is to satisfy the Red Gate discipline (BC-5.38.001): every
// non-trivial stub body calls todo() so that tests fail immediately rather than
// returning silent zero values.
//
// BC-5.38.005 self-check: "If I include this real implementation, will the test
// for this function pass trivially without any implementer work?" — yes for every
// function above; all use todo().
func todo() {
	panic("not implemented") //nolint:forbidigo // Red Gate stub — implementer replaces with real body
}
