// Package session — upstream.go defines the Authorizer hook that the access
// node consults before accepting a keystroke from an attached console, the
// KeystrokeSink interface for delegating keystroke forwarding to the effectful
// tmux layer, and the AccessNode type that wires together Publisher +
// ConsoleSet + keystroke serialization (BC-2.04.003; BC-2.04.004; BC-2.04.006).
//
// S-3.03 will replace NoOpAuthorizer with SessionAuth (Tier-2 per-session
// authorization); S-3.02 ships the hook so that AC-001..AC-008 pass with the
// default allow-all behaviour.
//
// Classification: boundary (ARCH-09). AccessNode is goroutine-free; the
// KeystrokeSink is implemented by the effectful tmux layer (internal/tmux),
// which writes to the real tmux subprocess. Tests inject a fake KeystrokeSink
// to assert forwarding without shelling out to tmux.
package session

import (
	"fmt"
	"sync"
	"time"

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

// KeystrokeSink is the interface implemented by the effectful tmux layer
// (internal/tmux.ControlMode, internal/tmux.PTYProxy, and
// internal/tmux.SessionConnector) for forwarding keystrokes into the running
// tmux session or PTY.
//
// AccessNode.SendKeystroke delegates to the injected KeystrokeSink after
// authorization and serialization. Tests inject a fake sink that records calls
// for assertion (AC-002; AC-007). Production callers inject
// internal/tmux.SessionConnector.
//
// SendInput must be safe for concurrent calls from multiple goroutines. It is
// called while AccessNode holds sinkMu, so the implementation must not call
// back into AccessNode under any lock.
type KeystrokeSink interface {
	// SendInput writes payload to the tmux session or PTY. The payload slice
	// must not be retained after SendInput returns.
	SendInput(payload []byte) error
}

// noOpSink is the default KeystrokeSink used when no sink is injected. It
// discards all payloads silently. Tests that do not assert forwarding use
// this via NewAccessNode without a WithKeystrokeSink option.
type noOpSink struct{}

func (noOpSink) SendInput(_ []byte) error { return nil }

// AccessNodeOption is a functional option for NewAccessNode.
type AccessNodeOption func(*AccessNode)

// WithKeystrokeSink sets the KeystrokeSink that AccessNode.SendKeystroke
// delegates to. Tests inject a fake sink; production callers inject
// tmux.SessionConnector (which implements KeystrokeSink via SendInput).
func WithKeystrokeSink(sink KeystrokeSink) AccessNodeOption {
	return func(a *AccessNode) {
		a.sink = sink
	}
}

// AccessNode is the in-process access node: it owns a Publisher (session
// lifecycle), a ConsoleSet (fan-out), a keystroke serialization mutex, an
// Authorizer hook for upstream keystroke gating, and a KeystrokeSink for
// forwarding keystrokes to the effectful tmux layer.
//
// AccessNode is goroutine-free (ARCH-09: boundary classification). It spawns
// no goroutines. The effectful tmux layer is responsible for I/O and goroutine
// management. Tests drive AccessNode synchronously without goroutine leaks.
//
// The zero value is not usable; construct with NewAccessNode.
//
// Concurrency: AccessNode is safe for concurrent use.
type AccessNode struct {
	pub        *Publisher
	consoles   *ConsoleSet
	authorizer Authorizer
	sink       KeystrokeSink

	// sinkMu serializes all keystroke writes through the sink before
	// forwarding to tmux (BC-2.04.006 Invariant 3: no keystroke race condition).
	// All consoles' keystrokes from any SendKeystroke call funnel through this
	// single mutex — the spec-mandated serialization point.
	sinkMu sync.Mutex

	// mu guards upstreams — the per-console bidirectional channel map used to
	// verify console attachment in SendKeystroke.
	mu        sync.Mutex
	upstreams map[ConsoleKey]chan []byte
}

// NewAccessNode constructs an AccessNode using the given Publisher and
// Authorizer. If auth is nil, NoOpAuthorizer is used. Functional options
// (e.g. WithKeystrokeSink) customize the node further. If no
// WithKeystrokeSink option is provided, a no-op sink is used (keystrokes are
// accepted but silently discarded — suitable for tests that do not assert
// forwarding).
func NewAccessNode(pub *Publisher, auth Authorizer, opts ...AccessNodeOption) *AccessNode {
	if auth == nil {
		auth = NoOpAuthorizer{}
	}
	a := &AccessNode{
		pub:        pub,
		consoles:   NewConsoleSet(),
		authorizer: auth,
		sink:       noOpSink{},
		upstreams:  make(map[ConsoleKey]chan []byte),
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Attach establishes a bidirectional channel for the console identified by key
// on the named session (BC-2.04.003 PC-1 through PC-3).
//
// AccessNode is goroutine-free: no per-console consumer goroutine is started.
// Keystrokes are forwarded synchronously via SendKeystroke → sink.SendInput
// (F-06 pass-2 arch rework). The upstream channel is still returned for
// callers that write directly to it (e.g. test helpers); however, AccessNode
// does NOT drain it — callers must use SendKeystroke for the authorizer +
// serialization guarantees.
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

	// Store bidirectional reference so Detach can close it and SendKeystroke
	// can verify console attachment.
	a.mu.Lock()
	a.upstreams[key] = us
	a.mu.Unlock()

	// Return send-only upstream to the caller (they must not close it).
	return ds, us, nil
}

// Detach closes the console's downstream channel and removes it from the
// ConsoleSet (BC-2.04.004 PC-1 through PC-3).
//
// The upstream channel is closed to free the ConsoleSet entry and signal any
// goroutine that may be draining it (F-03 pass-1 fix: console close-signal API).
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

	// Close upstream channel to signal any waiting reader.
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
// sinkMu is held during the sink.SendInput call to prevent keystroke
// interleaving under concurrent calls from multiple consoles (AC-007). All
// consoles' keystrokes funnel through this single mutex — the
// spec-mandated serialization point ("before forwarding to tmux").
//
// AccessNode is goroutine-free: SendKeystroke is synchronous. No channel send
// or goroutine is involved; the sink writes directly.
//
// Returns ErrConsoleNotFound if key is not currently attached. Returns the
// Authorizer's error if authorization is denied. Propagates sink.SendInput
// errors.
func (a *AccessNode) SendKeystroke(key ConsoleKey, sessionName string, payload []byte) error {
	if err := a.authorizer.Allow(key, sessionName, payload); err != nil {
		return err
	}

	// Verify the console is currently attached under a.mu.
	a.mu.Lock()
	_, ok := a.upstreams[key]
	a.mu.Unlock()

	if !ok {
		return fmt.Errorf("%w: %s", ErrConsoleNotFound, key)
	}

	// Serialize: only one goroutine forwards keystrokes at a time
	// (BC-2.04.006 Invariant 3; AC-007). All consoles share this mutex.
	a.sinkMu.Lock()
	defer a.sinkMu.Unlock()

	return a.sink.SendInput(payload)
}

// DeliverFrame fans out hdr to all currently-attached consoles, then calls
// Evict to remove any consoles whose channels have been closed (AC-008;
// BC-2.04.004 EC-002).
func (a *AccessNode) DeliverFrame(hdr frame.OuterHeader) {
	a.consoles.Deliver(hdr)
	a.consoles.Evict()
}

// Sweep evicts stale consoles whose keepalive heartbeat has not been updated
// within the given deadline (AC-008; BC-2.04.004 EC-002 keepalive crash path).
// Delegates to ConsoleSet.EvictStale. AccessNode is goroutine-free: the caller
// is responsible for invoking Sweep periodically (e.g. from a timer in
// cmd/switchboard). Tests call Sweep directly to advance the deadline.
//
// Returns the count of consoles evicted.
func (a *AccessNode) Sweep(deadline time.Duration) int {
	return a.consoles.EvictStale(deadline)
}
