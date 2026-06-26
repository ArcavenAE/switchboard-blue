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
	"errors"
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

// ErrNoKeystrokeSink is returned by the default sink when no KeystrokeSink has
// been injected at construction. Callers that need a real sink MUST use
// WithKeystrokeSink; callers that explicitly want to discard keystrokes should
// use WithKeystrokeSink(NoOpSink{}).
var ErrNoKeystrokeSink = errors.New("session: no keystroke sink installed; construct AccessNode with WithKeystrokeSink")

// noSink is the default fail-loud sink. It returns ErrNoKeystrokeSink on every
// call so that production callers that forget to inject a sink fail visibly
// rather than silently discarding keystrokes (F-L-2 pass-3: anti-silent-failure).
type noSink struct{}

func (noSink) SendInput(_ []byte) error { return ErrNoKeystrokeSink }

// NoOpSink is an exported KeystrokeSink that discards all payloads silently.
// Tests that do not assert forwarding use this explicitly to make the
// "no-op" intent clear at the call site.
//
// Use: session.NewAccessNode(pub, auth, session.WithKeystrokeSink(session.NoOpSink{}))
type NoOpSink struct{}

// SendInput discards payload and returns nil.
func (NoOpSink) SendInput(_ []byte) error { return nil }

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
}

// NewAccessNode constructs an AccessNode using the given Publisher and
// Authorizer. If auth is nil, NoOpAuthorizer is used. Functional options
// (e.g. WithKeystrokeSink) customize the node further.
//
// If no WithKeystrokeSink option is provided, SendKeystroke returns
// ErrNoKeystrokeSink on every call. This is a deliberate fail-loud default
// (F-L-2): production callers MUST inject a real sink; tests that do not
// assert forwarding should use WithKeystrokeSink(NoOpSink{}).
func NewAccessNode(pub *Publisher, auth Authorizer, opts ...AccessNodeOption) *AccessNode {
	if auth == nil {
		auth = NoOpAuthorizer{}
	}
	a := &AccessNode{
		pub:        pub,
		consoles:   NewConsoleSet(),
		authorizer: auth,
		sink:       noSink{},
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
// OWNERSHIP CONTRACT (F-H-5 pass-3): The upstream channel is owned by the
// ConsoleSet entry. The returned chan<- []byte is a send-only narrow view.
// Callers MUST NOT write to the upstream channel concurrently with Detach or
// Sweep — closing a channel while a concurrent goroutine is sending to it
// panics. The safe path for production callers is SendKeystroke, which does
// not write to the upstream channel at all; direct channel writes are for test
// harnesses only that fully control the attach/detach lifecycle and guarantee
// no sends occur after detach.
//
// Returns:
//   - downstream: a receive-only channel of frame.OuterHeader values delivered
//     from the session to the console (BC-2.04.003 PC-2).
//   - upstream: a send-only channel for keystroke payloads from the console to
//     the access node (BC-2.04.003 PC-3). Not drained by AccessNode.
//   - err: ErrSessionNotFound if sessionName is not in the publisher's live set
//     (E-SES-001; BC-2.04.003 EC-002); ErrConsoleAlreadyAttached if key is
//     already attached (E-SES-002), including console_id and session_name in
//     the message.
//
// A successful Attach adds key to the ConsoleSet. Subsequent DeliverFrame calls
// will fan out to the new console.
func (a *AccessNode) Attach(key ConsoleKey, sessionName string) (downstream <-chan frame.OuterHeader, upstream chan<- []byte, err error) {
	if _, err := a.pub.Get(sessionName); err != nil {
		return nil, nil, fmt.Errorf("%w: %s", ErrSessionNotFound, sessionName)
	}

	ds, us, err := a.consoles.Add(key)
	if err != nil {
		if errors.Is(err, ErrConsoleAlreadyAttached) {
			return nil, nil, fmt.Errorf("session: console %s already attached to session %s: %w", key, sessionName, ErrConsoleAlreadyAttached)
		}
		return nil, nil, err
	}

	// Return send-only upstream to the caller (they must not close it;
	// ownership stays with ConsoleSet).
	return ds, us, nil
}

// Detach closes the console's downstream channel and removes it from the
// ConsoleSet (BC-2.04.004 PC-1 through PC-3).
//
// ConsoleSet.Remove closes the downstream channel under its write-lock (same
// as Add), and EvictStale also closes the upstream channel outside the lock.
// After Detach the console's entry is removed from ConsoleSet — future
// SendKeystroke calls for this key return ErrConsoleNotFound (BC-2.04.004 PC-3).
//
// The tmux session on the access node continues running (BC-2.04.004 invariant 1:
// detach is non-destructive).
//
// Returns ErrConsoleNotFound if key is not currently attached (E-SES-003),
// including console_id in the message.
func (a *AccessNode) Detach(key ConsoleKey) error {
	if err := a.consoles.Remove(key); err != nil {
		if errors.Is(err, ErrConsoleNotFound) {
			return fmt.Errorf("session: console %s not found: %w", key, ErrConsoleNotFound)
		}
		return err
	}
	return nil
}

// SendKeystroke forwards payload to the tmux session on behalf of console key,
// after consulting the Authorizer (BC-2.04.006 Invariant 3: serialization).
//
// ConsoleSet is the single source of truth for attachment state (F-C-4
// pass-3 fix). SendKeystroke consults cs.IsAttached directly — there is no
// parallel map that could drift after EvictStale evicts a stale console. This
// eliminates the class of bug where an evicted console's keystrokes leak
// through because a stale a.upstreams entry was not cleaned up.
//
// sinkMu is held during the sink.SendInput call to prevent keystroke
// interleaving under concurrent calls from multiple consoles (AC-007). All
// consoles' keystrokes funnel through this single mutex — the
// spec-mandated serialization point ("before forwarding to tmux").
//
// AccessNode is goroutine-free: SendKeystroke is synchronous. No channel send
// or goroutine is involved; the sink writes directly.
//
// Returns ErrConsoleNotFound if key is not currently attached (E-SES-003),
// including console_id and session_name in the message. Returns the
// Authorizer's error if authorization is denied. Propagates sink.SendInput
// errors.
func (a *AccessNode) SendKeystroke(key ConsoleKey, sessionName string, payload []byte) error {
	if err := a.authorizer.Allow(key, sessionName, payload); err != nil {
		return err
	}

	// ConsoleSet is the single source of truth: check attachment here, not in
	// a parallel map. EvictStale removes from cs.consoles; after that, any
	// SendKeystroke call returns ErrConsoleNotFound without touching the sink.
	if !a.consoles.IsAttached(key) {
		return fmt.Errorf("session: console %s not found in session %s: %w", key, sessionName, ErrConsoleNotFound)
	}

	// Serialize: only one goroutine forwards keystrokes at a time
	// (BC-2.04.006 Invariant 3; AC-007). All consoles share this mutex.
	a.sinkMu.Lock()
	defer a.sinkMu.Unlock()

	return a.sink.SendInput(payload)
}

// DeliverFrame fans out hdr to all currently-attached consoles.
func (a *AccessNode) DeliverFrame(hdr frame.OuterHeader) {
	a.consoles.Deliver(hdr)
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

// Heartbeat records a keepalive timestamp for the console identified by key.
// Delegates to ConsoleSet.Heartbeat. Returns ErrConsoleNotFound if key is not
// currently attached.
func (a *AccessNode) Heartbeat(key ConsoleKey) error {
	return a.consoles.Heartbeat(key)
}
