// Package session — ConsoleSet manages the set of consoles attached to a
// session and fans out downstream frames to all of them (BC-2.04.006 PC-1).
//
// Classification: boundary (ARCH-09). Fan-out is pure in-process channel
// delivery; no I/O, no goroutines spawned here.
package session

import (
	"errors"
	"sync"

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

// consoleEntry holds the channels for one attached console.
// downstream delivers frames from the session to the console.
// upstream receives keystrokes from the console for forwarding to tmux.
type consoleEntry struct {
	downstream chan frame.OuterHeader
	upstream   chan []byte
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
	// evictQueue holds keys that have been gracefully detached via Remove but
	// were pending the next Evict sweep. In S-3.02 the keepalive-driven crash
	// path is deferred to S-3.03+; Remove enqueues here so Evict has a seam
	// for future crash detection without API change.
	evictQueue []ConsoleKey
}

// NewConsoleSet constructs an empty ConsoleSet ready for use.
func NewConsoleSet() *ConsoleSet {
	return &ConsoleSet{
		consoles: make(map[ConsoleKey]consoleEntry),
	}
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
// by the console operator.
//
// downstream is buffered with capacity downstreamBufSize; upstream is buffered
// with capacity upstreamBufSize so that a single keystroke does not block the
// sender when the effectful consumer is not yet draining.
//
// Returns ErrConsoleAlreadyAttached if key is already registered.
func (cs *ConsoleSet) Add(key ConsoleKey) (downstream <-chan frame.OuterHeader, upstream chan<- []byte, err error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if _, ok := cs.consoles[key]; ok {
		return nil, nil, ErrConsoleAlreadyAttached
	}

	entry := consoleEntry{
		downstream: make(chan frame.OuterHeader, downstreamBufSize),
		upstream:   make(chan []byte, upstreamBufSize),
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
	// Enqueue for Evict bookkeeping so callers can query eviction count.
	cs.evictQueue = append(cs.evictQueue, key)

	return nil
}

// Deliver sends a copy of hdr to every currently-attached console's downstream
// channel (BC-2.04.006 PC-1; invariant: no console is skipped).
//
// Deliver takes a value-copy snapshot of the channel set under RLock, releases
// the lock, then broadcasts to each channel. If a console's channel is full,
// the frame is dropped for that console to avoid head-of-line blocking on a
// slow consumer (BC-2.04.006 NFR-004 note).
func (cs *ConsoleSet) Deliver(hdr frame.OuterHeader) {
	cs.mu.RLock()
	snapshot := make([]chan frame.OuterHeader, 0, len(cs.consoles))
	for _, entry := range cs.consoles {
		snapshot = append(snapshot, entry.downstream)
	}
	cs.mu.RUnlock()

	for _, ch := range snapshot {
		select {
		case ch <- hdr:
		default:
			// Frame dropped: channel buffer full. Caller may call Evict() after
			// Deliver() to clean up any consoles that are no longer draining.
		}
	}
}

// Evict drains the internal evict queue, returning the count of evicted entries.
// (BC-2.04.004 EC-002; BC-2.04.006 invariant).
//
// In S-3.02, evictions are enqueued by Remove (graceful detach / crash-sim).
// The keepalive-driven crash path that calls Remove on timeout is deferred to
// S-3.03+; this method provides the seam without requiring an API change.
//
// Returns 0 when no evictions are pending.
func (cs *ConsoleSet) Evict() int {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	n := len(cs.evictQueue)
	cs.evictQueue = cs.evictQueue[:0]

	return n
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
