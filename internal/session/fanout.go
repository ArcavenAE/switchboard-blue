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
	downstream chan frame.OuterHeader //nolint:unused // fields wired in Add implementation
	upstream   chan []byte            //nolint:unused // fields wired in Add implementation
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
}

// NewConsoleSet constructs an empty ConsoleSet ready for use.
func NewConsoleSet() *ConsoleSet {
	return &ConsoleSet{
		consoles: make(map[ConsoleKey]consoleEntry),
	}
}

// Add registers a new console into the attached set, returning its downstream
// and upstream channels (BC-2.04.003 PC-1; BC-2.04.003 PC-3).
//
// The returned downstream channel receives frame.OuterHeader values delivered
// by Deliver. The returned upstream channel delivers keystroke payloads sent
// by the console operator.
//
// downstream is buffered with capacity downstreamBufSize; upstream is unbuffered
// (callers must consume promptly).
//
// Returns ErrConsoleAlreadyAttached if key is already registered.
func (cs *ConsoleSet) Add(key ConsoleKey) (downstream <-chan frame.OuterHeader, upstream chan<- []byte, err error) {
	panic("not implemented") // todo: BC-2.04.003 PC-1 — establish bidirectional channel
}

// Remove deregisters the console identified by key, closing its downstream
// channel (BC-2.04.004 PC-1). The tmux session is unaffected — closing the
// downstream channel does not terminate the underlying session.
//
// Returns ErrConsoleNotFound if key is not registered.
func (cs *ConsoleSet) Remove(key ConsoleKey) error {
	panic("not implemented") // todo: BC-2.04.004 PC-1/PC-2 — close channel, evict from set
}

// Deliver sends a copy of hdr to every currently-attached console's downstream
// channel (BC-2.04.006 PC-1; invariant: no console is skipped).
//
// Deliver iterates over a value-copy snapshot of the attached set (CLAUDE.md
// Go rule 12: no internal pointer leak). If a console's channel is full, the
// frame is dropped for that console; the send is non-blocking to avoid
// head-of-line blocking on a slow consumer. Crash detection (eviction on closed
// channel) is handled by Evict.
func (cs *ConsoleSet) Deliver(hdr frame.OuterHeader) {
	panic("not implemented") // todo: BC-2.04.006 PC-1 — fan-out frame to all attached consoles
}

// Evict removes all consoles whose downstream channels are closed, returning
// the count of evicted consoles (BC-2.04.004 EC-002; BC-2.04.006 invariant).
//
// A console's downstream channel is considered closed if a non-blocking send
// panics (closed channel). Evict is called by the access node's delivery loop
// after each Deliver call to ensure crashed consoles are cleaned up.
func (cs *ConsoleSet) Evict() int {
	panic("not implemented") // todo: BC-2.04.004 EC-002 — detect + evict crashed consoles
}

// Len returns the number of currently-attached consoles.
func (cs *ConsoleSet) Len() int {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return len(cs.consoles)
}

// Snapshot returns a value-copy of the current console key set.
// The returned slice is decoupled from internal state (CLAUDE.md Go rule 12).
func (cs *ConsoleSet) Snapshot() []ConsoleKey {
	panic("not implemented") // todo: return value-copy snapshot of attached key set
}

// downstreamBufSize is the buffer depth for per-console downstream channels.
// A modest buffer prevents a slow console from blocking Deliver entirely while
// still bounding memory per console.
const downstreamBufSize = 64 //nolint:unused // consumed by Add implementation
