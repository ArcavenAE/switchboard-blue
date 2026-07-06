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
// Forbidden: internal/routing, internal/tmux (circular), internal/metrics
// (topological inversion — internal/metrics is DAG position 12, downstream
// of session; boundary composition happens in cmd/switchboard via typed
// SessionHook callbacks, mirroring routing.ForwardingEntryHook for
// pathTrackerSource in S-BL.PATH-TRACKER-WIRING).
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

// SessionHook is called once per Publish / Unpublish with the session name
// and its PublishedAt timestamp. It fires while the Publisher holds its
// write lock; hook implementations MUST NOT re-enter Publisher (any
// exported Publisher method that acquires p.mu would deadlock).
//
// The hook exists so that cmd/switchboard can maintain a per-session
// QualityIndicator registry (S-BL.CONSOLE-OBS) without violating
// ARCH-08 §6.6: internal/session (DAG position 6) MUST NOT import
// internal/metrics (DAG position 12; inverting the topological order
// would cascade through internal/tmux at position 7). Publisher is
// unaware of QualityIndicator; it only fires a notification. Mirrors
// routing.ForwardingEntryHook (S-BL.PATH-TRACKER-WIRING).
//
// For Unpublish, publishedAt is the original PublishedAt from Publish
// so observers can log an accurate session lifetime without re-fetching
// state that has already been removed.
type SessionHook func(sessionName string, publishedAt time.Time)

// Publisher manages the set of published tmux sessions (BC-2.04.001 PC-2;
// ARCH-08 §6.6 position 6). It holds a reference to the admission key set
// exposed to internal/tmux via AdmittedKeySet(). Tier-2 SessionAuth keys
// are registered independently via RegisterKey (BC-2.05.003 DI-011) and do
// NOT flow through this key set.
//
// The zero value is not usable; construct with NewPublisher.
//
// Concurrency: Publisher is safe for concurrent use.
type Publisher struct {
	mu       sync.RWMutex
	sessions map[string]Info
	keys     *admission.AdmittedKeySet
	// publishHook / unpublishHook are optional SessionHook callbacks fired
	// under p.mu.Lock after the sessions map has been mutated. Nil-safe:
	// both fields default to nil and Publish / Unpublish check before
	// firing. Attach via SetPublishHook / SetUnpublishHook (S-BL.CONSOLE-OBS).
	publishHook   SessionHook
	unpublishHook SessionHook
}

// NewPublisher constructs a Publisher seeded with an admission key set.
// The key set is consumed only by AdmittedKeySet() for internal/tmux;
// Tier-2 SessionAuth keys are registered independently (BC-2.05.003 DI-011).
// keys must not be nil.
//
// Both SessionHook fields default to nil. Consumers that need
// per-session observability wiring (S-BL.CONSOLE-OBS) attach hooks via
// SetPublishHook / SetUnpublishHook after construction — the wiring path
// where the source-of-hook is only constructible after the Publisher
// (see cmd/switchboard.newSessionQualitySourceFromPublisher). Mirrors
// routing.SetForwardingEntryHook.
func NewPublisher(keys *admission.AdmittedKeySet) *Publisher {
	return &Publisher{
		sessions: make(map[string]Info),
		keys:     keys,
	}
}

// SetPublishHook installs (or replaces) the SessionHook fired inside
// Publish after the sessions map is mutated. Takes p.mu.Lock briefly to
// swap the field so it composes safely with concurrent Publish callers —
// no torn read of the hook function pointer, no lost registration.
//
// Passing nil disables the hook. The hook MUST NOT re-enter Publisher.
//
// S-BL.CONSOLE-OBS.
func (p *Publisher) SetPublishHook(hook SessionHook) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.publishHook = hook
}

// SetUnpublishHook installs (or replaces) the SessionHook fired inside
// Unpublish after the sessions map is mutated. Same concurrency
// discipline as SetPublishHook. Passing nil disables the hook. The hook
// MUST NOT re-enter Publisher.
//
// S-BL.CONSOLE-OBS.
func (p *Publisher) SetUnpublishHook(hook SessionHook) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.unpublishHook = hook
}

// Publish adds sessionName to the live set with the current UTC timestamp
// (BC-2.04.001 PC-2; PC-3).
//
// Returns ErrSessionAlreadyPublished if name is already present.
//
// When a publishHook is installed, it fires exactly once under p.mu.Lock
// after the sessions-map write, carrying (sessionName, PublishedAt).
func (p *Publisher) Publish(sessionName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.sessions[sessionName]; ok {
		return ErrSessionAlreadyPublished
	}

	info := Info{
		Name:        sessionName,
		PublishedAt: time.Now().UTC(),
	}
	p.sessions[sessionName] = info
	if p.publishHook != nil {
		p.publishHook(sessionName, info.PublishedAt)
	}

	return nil
}

// Unpublish removes sessionName from the live set (BC-2.04.001 PC-4).
//
// Returns ErrSessionNotFound if the session is not in the live set.
//
// When an unpublishHook is installed, it fires exactly once under p.mu.Lock
// after the sessions-map delete, carrying (sessionName, PublishedAt) where
// PublishedAt is the timestamp from the original Publish (so observers can
// log accurate session lifetimes without needing a separate Get call).
func (p *Publisher) Unpublish(sessionName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	info, ok := p.sessions[sessionName]
	if !ok {
		return ErrSessionNotFound
	}

	delete(p.sessions, sessionName)
	if p.unpublishHook != nil {
		p.unpublishHook(sessionName, info.PublishedAt)
	}

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

// Exists reports whether sessionName is currently in the live published set.
// Satisfies the SessionRegistry interface used by ConsoleServer (S-7.03;
// BC-2.08.001 PC-1/PC-3 — production wiring of SessionRegistry to Publisher).
func (p *Publisher) Exists(sessionName string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, ok := p.sessions[sessionName]
	return ok
}
