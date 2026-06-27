// Package session — ConsoleSet manages the set of consoles attached to a
// session and fans out downstream frames to all of them (BC-2.04.006 PC-1).
//
// Classification: boundary (ARCH-09). Fan-out is pure in-process channel
// delivery; no I/O, no goroutines spawned here.
package session

import (
	"errors"
	"sync"
	"time"

	"github.com/arcavenae/switchboard/internal/frame"
)

// ErrConsoleAlreadyAttached is returned by ConsoleSet.Add when a console key
// that is already in the attached set is added again (E-SES-002).
var ErrConsoleAlreadyAttached = errors.New("session: console already attached")

// ErrConsoleNotFound is returned by ConsoleSet.Remove when the given console
// key is not in the attached set (E-SES-003).
var ErrConsoleNotFound = errors.New("session: console not found")

// ConsoleKey is the unique string identifier for an attached console.
// It is opaque to this package; the caller is responsible for uniqueness.
type ConsoleKey string

// consoleEntry holds the channels and keepalive state for one attached console.
// downstream delivers frames from the session to the console.
// upstream receives keystrokes from the console for forwarding to tmux.
// lastHeartbeat records the last time Heartbeat was called for this console;
// used by EvictStale to detect crash/disconnect.
type consoleEntry struct {
	downstream    chan frame.OuterHeader
	upstream      chan []byte
	lastHeartbeat time.Time
}

// ConsoleSet manages the set of consoles attached to a single session and fans
// out downstream frames to all of them (BC-2.04.006 PC-1; BC-2.04.004 PC-5).
//
// The zero value is not usable; construct with NewConsoleSet.
//
// Concurrency: ConsoleSet is safe for concurrent use.
type ConsoleSet struct {
	mu       sync.RWMutex
	consoles map[ConsoleKey]consoleEntry
	nowFn    func() time.Time
}

// ConsoleSetOption is a functional option for NewConsoleSet.
type ConsoleSetOption func(*ConsoleSet)

// ConsoleSetWithClock replaces the wall-clock source used by ConsoleSet.
// Tests inject a fake clock to deterministically control time; production
// code uses the default time.Now().UTC().
func ConsoleSetWithClock(fn func() time.Time) ConsoleSetOption {
	return func(cs *ConsoleSet) {
		cs.nowFn = fn
	}
}

// NewConsoleSet constructs an empty ConsoleSet ready for use.
func NewConsoleSet(opts ...ConsoleSetOption) *ConsoleSet {
	cs := &ConsoleSet{
		consoles: make(map[ConsoleKey]consoleEntry),
		nowFn:    func() time.Time { return time.Now().UTC() },
	}
	for _, opt := range opts {
		opt(cs)
	}
	return cs
}

// downstreamBufSize is the buffer depth for per-console downstream channels.
// A modest buffer prevents a slow console from blocking Deliver entirely while
// still bounding memory per console.
const downstreamBufSize = 64

// upstreamBufSize is the buffer depth for per-console upstream channels.
// A modest buffer lets the test helper (and real callers) enqueue a keystroke
// without an immediate reader. The effectful layer (S-3.03+) drains this channel.
const upstreamBufSize = 16

// Add registers a new console into the attached set, returning its downstream
// and upstream channels (BC-2.04.003 PC-1; BC-2.04.003 PC-3).
//
// The returned downstream channel receives frame.OuterHeader values delivered
// by Deliver. The returned upstream channel delivers keystroke payloads sent
// by the console operator. The caller receives a bidirectional upstream chan so
// that AccessNode can close it (signalling the consumer goroutine) on Detach.
//
// CLOSE-RACE CONTRACT (F-H-5 pass-3): ConsoleSet owns the upstream channel.
// Remove and EvictStale close the upstream channel outside the write lock.
// Callers MUST NOT send to the upstream channel concurrently with Remove or
// EvictStale — closing a channel while a concurrent goroutine is sending to it
// panics. Production callers route all sends through AccessNode.SendKeystroke,
// which does not write to the upstream channel; direct sends are only for test
// harnesses that fully control the lifecycle.
//
// downstream is buffered with capacity downstreamBufSize; upstream is buffered
// with capacity upstreamBufSize so that a single keystroke does not block the
// sender when the effectful consumer is not yet draining.
//
// Returns ErrConsoleAlreadyAttached if key is already registered.
func (cs *ConsoleSet) Add(key ConsoleKey) (downstream <-chan frame.OuterHeader, upstream chan []byte, err error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if _, ok := cs.consoles[key]; ok {
		return nil, nil, ErrConsoleAlreadyAttached
	}

	entry := consoleEntry{
		downstream:    make(chan frame.OuterHeader, downstreamBufSize),
		upstream:      make(chan []byte, upstreamBufSize),
		lastHeartbeat: cs.nowFn(),
	}
	cs.consoles[key] = entry

	return entry.downstream, entry.upstream, nil
}

// Remove deregisters the console identified by key, closing its downstream
// channel (BC-2.04.004 PC-1). The tmux session is unaffected — closing the
// downstream channel does not terminate the underlying session.
//
// Returns ErrConsoleNotFound if key is not registered.
func (cs *ConsoleSet) Remove(key ConsoleKey) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	entry, ok := cs.consoles[key]
	if !ok {
		return ErrConsoleNotFound
	}

	close(entry.downstream)
	delete(cs.consoles, key)

	return nil
}

// Deliver sends a copy of hdr to every currently-attached console's downstream
// channel (BC-2.04.006 PC-1; invariant: no console is skipped).
//
// The RLock is held for the entire loop — snapshot and sends happen under the
// same lock acquisition. This prevents a concurrent Remove (which takes WLock)
// from closing a channel while a send is in-flight, eliminating the
// close-during-send race that would cause a panic (F-01 pass-1 fix).
//
// If a console's channel is full the frame is dropped for that console to avoid
// head-of-line blocking on a slow consumer (BC-2.04.006 NFR-004).
func (cs *ConsoleSet) Deliver(hdr frame.OuterHeader) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	for _, entry := range cs.consoles {
		select {
		case entry.downstream <- hdr:
		default:
			// Frame dropped: channel buffer full. Caller may call Evict() after
			// Deliver() to clean up any consoles that are no longer draining.
		}
	}
}

// Heartbeat records a keepalive timestamp for the console identified by key
// (AC-008; BC-2.04.004 EC-002). The timestamp is used by EvictStale to
// distinguish live consoles from stale/crashed ones.
//
// Returns ErrConsoleNotFound if key is not registered. The timestamp is
// recorded as time.Now().UTC() at the moment Heartbeat is called.
func (cs *ConsoleSet) Heartbeat(key ConsoleKey) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	entry, ok := cs.consoles[key]
	if !ok {
		return ErrConsoleNotFound
	}

	entry.lastHeartbeat = cs.nowFn()
	cs.consoles[key] = entry

	return nil
}

// EvictStale removes consoles whose lastHeartbeat is older than deadline ago
// from now (AC-008; BC-2.04.004 EC-002 keepalive crash path). Each evicted
// console's downstream channel is closed under WLock (same as Remove), and its
// upstream channel is closed to release any blocked sender. The upstream
// channel close is deferred to outside the lock (same rationale as Remove).
//
// Returns the count of consoles evicted.
//
// Caller is responsible for invoking EvictStale periodically (e.g. via a timer
// in cmd/switchboard or AccessNode.Sweep). AccessNode is goroutine-free, so
// no background sweeper is started here; tests drive Sweep directly.
func (cs *ConsoleSet) EvictStale(deadline time.Duration) int {
	cutoff := cs.nowFn().Add(-deadline)

	cs.mu.Lock()

	var stale []ConsoleKey
	var upstreams []chan []byte

	for key, entry := range cs.consoles {
		if entry.lastHeartbeat.Before(cutoff) {
			stale = append(stale, key)
			upstreams = append(upstreams, entry.upstream)
			close(entry.downstream)
			delete(cs.consoles, key)
		}
	}

	cs.mu.Unlock()

	// Close upstream channels outside the lock (writes may block; avoid
	// holding WLock during a potential send from a concurrent caller).
	for _, us := range upstreams {
		close(us)
	}

	return len(stale)
}

// IsAttached reports whether the console identified by key is currently in the
// attached set. It is the single authoritative check for attachment state
// (F-C-4: ConsoleSet is the single source of truth; callers must not
// maintain a parallel map that can drift after EvictStale).
func (cs *ConsoleSet) IsAttached(key ConsoleKey) bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	_, ok := cs.consoles[key]
	return ok
}

// Len returns the number of currently-attached consoles.
func (cs *ConsoleSet) Len() int {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return len(cs.consoles)
}

// Snapshot returns a value-copy of the current console key set.
// The returned slice is decoupled from internal state (go.md rule 12).
func (cs *ConsoleSet) Snapshot() []ConsoleKey {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	out := make([]ConsoleKey, 0, len(cs.consoles))
	for key := range cs.consoles {
		out = append(out, key)
	}

	return out
}
