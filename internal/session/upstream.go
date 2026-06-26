// Package session — upstream.go defines the Authorizer hook that the access
// node consults before accepting a keystroke from an attached console, and the
// AccessNode type that wires together Publisher + ConsoleSet + keystroke
// serialization (BC-2.04.003; BC-2.04.004; BC-2.04.006).
//
// S-3.03 will replace NoOpAuthorizer with SessionAuth (Tier-2 per-session
// authorization); S-3.02 ships the hook so that AC-001..AC-008 pass with the
// default allow-all behaviour.
package session

import (
	"fmt"
	"sync"

	"github.com/arcavenae/switchboard/internal/frame"
)

// Authorizer is consulted by the access node's upstream-receive path before
// forwarding a console's keystroke frame to tmux. Returning a non-nil error
// causes the frame to be dropped and the error to be surfaced to the console.
//
// S-3.03 wires SessionAuth as the Authorizer. S-3.02 ships with NoOpAuthorizer
// (allow-all) so that attach/detach/fan-out tests pass without auth wired.
//
// Allow must be safe for concurrent calls from multiple goroutines.
type Authorizer interface {
	// Allow returns nil if the console identified by key is authorized to send
	// the given payload to sessionName, or a non-nil error otherwise.
	// The payload slice must not be retained after Allow returns.
	Allow(key ConsoleKey, sessionName string, payload []byte) error
}

// NoOpAuthorizer is the default allow-all implementation of Authorizer.
// Every Allow call returns nil unconditionally. It is safe for concurrent use.
//
// S-3.03 replaces this with SessionAuth (Tier-2 enforcement).
type NoOpAuthorizer struct{}

// Allow always returns nil (allow-all; no authorization logic).
func (NoOpAuthorizer) Allow(_ ConsoleKey, _ string, _ []byte) error {
	return nil
}

// upstreamSinkCap is the buffer depth for the AccessNode's upstream sink
// channel. Large enough to absorb bursts from concurrent consoles without
// blocking the consumer goroutines.
const upstreamSinkCap = 256

// AccessNode is the in-process access node: it owns a Publisher (session
// lifecycle), a ConsoleSet (fan-out), a keystroke serialization mutex, and an
// Authorizer hook for upstream keystroke gating.
//
// The zero value is not usable; construct with NewAccessNode.
//
// Concurrency: AccessNode is safe for concurrent use.
type AccessNode struct {
	pub        *Publisher
	consoles   *ConsoleSet
	authorizer Authorizer
	// upstreamMu serializes all keystroke writes to the tmux session before
	// forwarding (BC-2.04.006 Invariant 3: no keystroke race condition).
	upstreamMu sync.Mutex

	// mu guards upstreams — the per-console bidirectional channel map used to
	// close channels on Detach and to write in SendKeystroke.
	mu        sync.Mutex
	upstreams map[ConsoleKey]chan []byte

	// consumerWg tracks all per-console consumer goroutines. Used by tests and
	// graceful shutdown to wait for goroutines to drain.
	consumerWg sync.WaitGroup

	// UpstreamSink receives all keystroke payloads forwarded by the per-console
	// consumer goroutines in the order they arrive. Accessible to tests for
	// assertion (AC-002 PC-3; AC-007 serialization). S-3.03+ wires this into
	// the real tmux forwarding path.
	UpstreamSink chan []byte
}

// NewAccessNode constructs an AccessNode using the given Publisher and
// Authorizer. If auth is nil, NoOpAuthorizer is used.
func NewAccessNode(pub *Publisher, auth Authorizer) *AccessNode {
	if auth == nil {
		auth = NoOpAuthorizer{}
	}
	return &AccessNode{
		pub:          pub,
		consoles:     NewConsoleSet(),
		authorizer:   auth,
		upstreams:    make(map[ConsoleKey]chan []byte),
		UpstreamSink: make(chan []byte, upstreamSinkCap),
	}
}

// Attach establishes a bidirectional channel for the console identified by key
// on the named session (BC-2.04.003 PC-1 through PC-3).
//
// A per-console consumer goroutine is started to drain the upstream channel and
// forward keystrokes to UpstreamSink (F-06 pass-1 fix; AC-002 PC-3). The
// goroutine exits when the upstream channel is closed (i.e. on Detach).
//
// Returns:
//   - downstream: a receive-only channel of frame.OuterHeader values delivered
//     from the session to the console (BC-2.04.003 PC-2).
//   - upstream: a send-only channel for keystroke payloads from the console to
//     the access node (BC-2.04.003 PC-3).
//   - err: ErrSessionNotFound if sessionName is not in the publisher's live set
//     (E-SES-001; BC-2.04.003 EC-002); ErrConsoleAlreadyAttached if key is
//     already attached.
//
// A successful Attach adds key to the ConsoleSet. Subsequent DeliverFrame calls
// will fan out to the new console.
func (a *AccessNode) Attach(key ConsoleKey, sessionName string) (downstream <-chan frame.OuterHeader, upstream chan<- []byte, err error) {
	if _, err := a.pub.Get(sessionName); err != nil {
		return nil, nil, fmt.Errorf("%w: %s", ErrSessionNotFound, sessionName)
	}

	ds, us, err := a.consoles.Add(key)
	if err != nil {
		return nil, nil, err
	}

	// Store bidirectional reference so Detach can close it (stopping consumer)
	// and SendKeystroke can write to it.
	a.mu.Lock()
	a.upstreams[key] = us
	a.mu.Unlock()

	// Start a per-console consumer goroutine that drains the upstream channel
	// and forwards each payload to UpstreamSink. The goroutine exits when the
	// upstream channel is closed in Detach (F-06 pass-1 fix).
	a.consumerWg.Add(1)
	go func() {
		defer a.consumerWg.Done()
		for payload := range us {
			a.UpstreamSink <- payload
		}
	}()

	// Return send-only upstream to the caller (they must not close it).
	return ds, us, nil
}

// Detach closes the console's downstream channel and removes it from the
// ConsoleSet (BC-2.04.004 PC-1 through PC-3).
//
// Detach also closes the upstream channel to stop the per-console consumer
// goroutine started in Attach (F-03 pass-1 fix: console close-signal API).
//
// The tmux session on the access node continues running (BC-2.04.004 invariant 1:
// detach is non-destructive).
//
// Returns ErrConsoleNotFound if key is not currently attached.
func (a *AccessNode) Detach(key ConsoleKey) error {
	// Remove from ConsoleSet first (closes downstream, removes from map).
	if err := a.consoles.Remove(key); err != nil {
		return err
	}

	// Close upstream channel to stop the consumer goroutine.
	a.mu.Lock()
	us, ok := a.upstreams[key]
	if ok {
		delete(a.upstreams, key)
	}
	a.mu.Unlock()

	if ok {
		close(us)
	}

	return nil
}

// SendKeystroke forwards payload to the tmux session on behalf of console key,
// after consulting the Authorizer (BC-2.04.006 Invariant 3: serialization).
//
// The upstreamMu mutex is held during the send to prevent keystroke
// interleaving under concurrent calls (AC-007). The payload is written to the
// console's upstream channel, which the per-console consumer goroutine drains
// into UpstreamSink (F-02 pass-1 fix: payload no longer discarded).
//
// Returns ErrConsoleNotFound if key is not currently attached. Returns the
// Authorizer's error if authorization is denied.
func (a *AccessNode) SendKeystroke(key ConsoleKey, sessionName string, payload []byte) error {
	if err := a.authorizer.Allow(key, sessionName, payload); err != nil {
		return err
	}

	// Look up the upstream channel under the AccessNode's own lock.
	a.mu.Lock()
	us, ok := a.upstreams[key]
	a.mu.Unlock()

	if !ok {
		return fmt.Errorf("%w: %s", ErrConsoleNotFound, key)
	}

	// Serialize: only one goroutine forwards keystrokes at a time
	// (BC-2.04.006 Invariant 3; AC-007).
	a.upstreamMu.Lock()
	defer a.upstreamMu.Unlock()

	// Write payload to the upstream channel. The per-console consumer goroutine
	// drains it into UpstreamSink. Non-blocking send: if the channel is full,
	// the keystroke is dropped to avoid blocking the caller (same backpressure
	// semantics as Deliver; S-3.03+ will tune this with flow-control).
	select {
	case us <- payload:
	default:
		// Upstream channel full: keystroke dropped under backpressure.
		// The upstreamMu ensures no interleaving even on drop.
	}

	return nil
}

// DeliverFrame fans out hdr to all currently-attached consoles, then calls
// Evict to remove any consoles whose channels have been closed (AC-008;
// BC-2.04.004 EC-002).
func (a *AccessNode) DeliverFrame(hdr frame.OuterHeader) {
	a.consoles.Deliver(hdr)
	a.consoles.Evict()
}
