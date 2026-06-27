// Package session manages session publication state for the access node's
// downstream half-channel (ARCH-08 §6.6 position 6; BC-2.04.001; ADR-010).
//
// Classification: boundary (ARCH-09). This package owns session lifecycle
// state and coordinates downstream half-channel wiring. It has no I/O and
// does not spawn goroutines itself — those are the responsibility of the
// effectful layer (internal/tmux).
//
// Allowed internal imports: {frame, admission} per ARCH-08 §6.6.
// Current code imports only admission; frame is permitted but unused
// (FrameTypeData re-export was deleted when no consumer materialised).
// Forbidden: internal/routing, internal/tmux (circular).
package session

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
)

// ErrSessionNotFound is returned when an operation targets a named session
// that is not present in the publisher's live set (E-SES-001; BC-2.04.003).
var ErrSessionNotFound = errors.New("session: not found")

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

// Publisher manages the set of published tmux sessions (BC-2.04.001 PC-2;
// ARCH-08 §6.6 position 6). It holds a reference to the admission key set
// for use by future per-session admission gating (S-3.03 SessionAuth /
// Tier-2); the current Publish/Unpublish path does NOT consult the key set.
//
// The zero value is not usable; construct with NewPublisher.
//
// Concurrency: Publisher is safe for concurrent use.
type Publisher struct {
	mu       sync.RWMutex
	sessions map[string]Info
	keys     *admission.AdmittedKeySet
}

// NewPublisher constructs a Publisher seeded with an admission key set.
// The key set is reserved for S-3.03 SessionAuth (Tier-2 per-session
// admission gating); the current Publish/Unpublish path does not consult
// it. keys must not be nil.
func NewPublisher(keys *admission.AdmittedKeySet) *Publisher {
	return &Publisher{
		sessions: make(map[string]Info),
		keys:     keys,
	}
}

// Publish adds sessionName to the live set with the current UTC timestamp
// (BC-2.04.001 PC-2; PC-3).
//
// Returns ErrSessionAlreadyPublished if name is already present.
func (p *Publisher) Publish(sessionName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.sessions[sessionName]; ok {
		return ErrSessionAlreadyPublished
	}

	p.sessions[sessionName] = Info{
		Name:        sessionName,
		PublishedAt: time.Now().UTC(),
	}

	return nil
}

// Unpublish removes sessionName from the live set (BC-2.04.001 PC-4).
//
// Returns ErrSessionNotFound if the session is not in the live set.
func (p *Publisher) Unpublish(sessionName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.sessions[sessionName]; !ok {
		return ErrSessionNotFound
	}

	delete(p.sessions, sessionName)

	return nil
}

// ListSessions returns a snapshot of all currently published sessions, ordered
// alphabetically by name (BC-2.04.001 PC-2; VP-031).
//
// The returned slice is a value copy — mutations do not affect the Publisher's
// internal state (go.md rule 12: no internal pointer leak).
func (p *Publisher) ListSessions() []Info {
	p.mu.RLock()
	defer p.mu.RUnlock()

	out := make([]Info, 0, len(p.sessions))
	for _, info := range p.sessions {
		out = append(out, info)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})

	return out
}

// Get returns the Info for sessionName, or ErrSessionNotFound if absent
// (E-SES-001; BC-2.04.003).
func (p *Publisher) Get(sessionName string) (Info, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	info, ok := p.sessions[sessionName]
	if !ok {
		return Info{}, ErrSessionNotFound
	}

	return info, nil
}

// AdmittedKeySet exposes the underlying admission key set for read access by
// internal/tmux. Returns the same pointer passed to NewPublisher.
//
// The return type is a pointer to the concrete type, not an interface
// (Go rule 6: accept interfaces, return concrete types).
func (p *Publisher) AdmittedKeySet() *admission.AdmittedKeySet {
	return p.keys
}
