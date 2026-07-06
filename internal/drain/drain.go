// Package drain implements the router graceful-drain coordinator (BC-2.09.002).
//
// The Drain coordinator broadcasts a shutdown signal to registered observers
// and waits (bounded by a configurable timeout) for each observer to
// acknowledge migration. It is the internal seam through which S-7.04
// applies BC-2.09.003 PC-7 (drain_timeout).
//
// This package is a leaf in ARCH-08 §6 DAG position 16 (internal/drain) and
// currently has no upward imports. When the DRAIN-over-SVTN wire protocol
// lands (follow-on story S-7.04-FU), an import of internal/routing may be
// added — that is the sole upward import ARCH-08 permits from position 16.
//
// This package is pure-core: it performs no I/O and opens no sockets. Timers
// are the only side-effect and are driven by the injected context.
package drain

import (
	"context"
	"errors"
	"sync"
	"time"
)

// DefaultTimeout is the default drain window when no cfg.DrainTimeout is set.
// ARCH-06 §Graceful Drain: "wait drain_timeout (default 10s) for nodes to
// migrate." Also BC-2.09.003 PC-7 zero-value semantics: zero → daemon default.
const DefaultTimeout = 10 * time.Second

// ErrTimeout is returned by Wait when the drain window elapsed before all
// registered observers acknowledged. Callers proceed with shutdown regardless
// (BC-2.09.002 EC-003: "timeout exceeded → disconnect anyway").
var ErrTimeout = errors.New("drain: timeout waiting for observers to acknowledge")

// Observer is a callback invoked once per Drain.Signal for each registered
// observer. Observers perform migration work (in Wave 7: broadcast DRAIN to
// their connected nodes over the SVTN channel). Returning from the callback
// counts as acknowledgment.
//
// Observers MUST honor ctx cancellation — when ctx is done, the drain window
// has elapsed and the observer is expected to unwind.
type Observer func(ctx context.Context)

// Drain is the router-side graceful-drain coordinator (BC-2.09.002).
//
// A Drain is single-use: Signal may be called at most once. Concurrent calls
// after the first return immediately; observers registered after the first
// Signal do NOT participate in that drain.
type Drain struct {
	timeout time.Duration

	mu        sync.Mutex
	observers []Observer
	signaled  bool
	done      chan struct{} // closed when all observers return, or timer fires
	timedOut  bool          // set true iff done was closed by the timeout branch
}

// New constructs a Drain coordinator with the supplied window.
//
// BC-2.09.003 PC-7 zero-value semantics: a zero timeout means "use daemon
// default" — DefaultTimeout (10s). Negative values are rejected by config
// validation (E-CFG-006); if a negative slips past, New defensively uses
// DefaultTimeout so a mis-wired caller never gets an unbounded drain.
func New(timeout time.Duration) *Drain {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &Drain{
		timeout:   timeout,
		observers: nil,
		done:      make(chan struct{}),
	}
}

// Timeout returns the resolved drain window (post-default substitution).
// Exposed for observability and tests.
func (d *Drain) Timeout() time.Duration {
	return d.timeout
}

// RegisterObserver adds fn to the set of callbacks invoked on Signal.
// Registrations made after Signal are ignored — Drain is single-use.
func (d *Drain) RegisterObserver(fn Observer) {
	if fn == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.signaled {
		return
	}
	d.observers = append(d.observers, fn)
}

// Signal initiates the drain. All registered observers are invoked
// concurrently. Signal returns immediately after fan-out; callers use Wait
// to block for acknowledgment.
//
// The supplied ctx is used only to derive the drain-window sub-context that
// observers receive; cancelling ctx does NOT abort observers already running.
// Signal is idempotent: subsequent calls are no-ops.
func (d *Drain) Signal(ctx context.Context) {
	d.mu.Lock()
	if d.signaled {
		d.mu.Unlock()
		return
	}
	d.signaled = true
	observers := d.observers
	d.mu.Unlock()

	drainCtx, cancel := context.WithTimeout(ctx, d.timeout)

	if len(observers) == 0 {
		// No observers → drain completes immediately. Cancel the derived
		// context to release resources and signal Wait.
		cancel()
		close(d.done)
		return
	}

	var obsWG sync.WaitGroup
	obsWG.Add(len(observers))
	for _, fn := range observers {
		fn := fn
		go func() {
			defer obsWG.Done()
			fn(drainCtx)
		}()
	}

	// Race: either all observers return, or the drain window elapses.
	go func() {
		defer cancel()
		obsAck := make(chan struct{})
		go func() {
			obsWG.Wait()
			close(obsAck)
		}()
		select {
		case <-obsAck:
			// Clean completion.
		case <-drainCtx.Done():
			// Window elapsed. Observers keep running but Wait unblocks
			// with ErrTimeout. BC-2.09.002 EC-003: caller proceeds with
			// disconnect regardless.
			d.mu.Lock()
			d.timedOut = true
			d.mu.Unlock()
		}
		close(d.done)
	}()
}

// Wait blocks until Signal has completed (either all observers ACKed or the
// drain window elapsed), or until ctx is cancelled — whichever comes first.
//
// Returns nil on clean drain, ErrTimeout when the drain window elapsed
// before observers ACKed, or ctx.Err() when the caller-supplied ctx was
// cancelled while Wait was blocked.
//
// If Signal has not been called, Wait blocks until either ctx cancels or
// Signal is called on another goroutine.
func (d *Drain) Wait(ctx context.Context) error {
	select {
	case <-d.done:
		d.mu.Lock()
		timedOut := d.timedOut
		d.mu.Unlock()
		if timedOut {
			return ErrTimeout
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
