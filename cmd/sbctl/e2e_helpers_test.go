//go:build integration

package main

// e2e_helpers_test.go — test-only infrastructure for the management plane e2e harness.
//
// Contains:
//   - closingListenerWrapper + closingConn: server-side FIN observation for AC-005
//   - per-mode handler constructors: routerHandlers, accessHandlers, consoleHandlers, controlHandlers
//
// Traceability:
//
//	AC-005 — Q6 ruling (Option A): server-side listener wrapper observing client FIN
//	AC-001 — Q1 ruling (Option A): per-mode distinct handler tables

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/arcavenae/switchboard/internal/mgmt"
)

// ── closingListenerWrapper ─────────────────────────────────────────────────────
//
// Wraps a net.Listener. Every Conn returned by Accept() is wrapped in a
// closingConn that fires a channel when the server-side Read() returns io.EOF
// or net.ErrClosed (i.e., when the client has closed its side of the connection).
// This instruments the actual production `defer conn.Close()` in connectAndRun —
// not a tautological local close check (Q6 ruling — Option A, AC-005, BC-2.07.002 Inv-2).
//
// Thread safety: the closed map is protected by mu. waitForCloseAfter snapshots
// the channel under mu; callers must not hold mu themselves.
//
// closeEvents is a monotonically-increasing counter incremented each time any
// tracked conn's Read fires the closed channel. Use closeCounter() to snapshot
// before a client dials, then waitForCloseAfter(baseline, d) to wait for the
// counter to exceed baseline — this provably ignores earlier closes (e.g. the
// awaitReady probe conn) and only fires on closes that happen after the snapshot.

type closingListenerWrapper struct {
	net.Listener
	mu          sync.Mutex
	closed      map[net.Conn]chan struct{} // fired when server-side Read returns io.EOF/ErrClosed
	closeEvents atomic.Uint64              // incremented each time a tracked conn fires closed
}

// newClosingListenerWrapper wraps ln with FIN-observation capability.
func newClosingListenerWrapper(ln net.Listener) *closingListenerWrapper {
	return &closingListenerWrapper{
		Listener: ln,
		closed:   make(map[net.Conn]chan struct{}),
	}
}

// Accept wraps the underlying Listener.Accept() result in a closingConn.
// The closingConn fires closed[conn] when the remote side closes.
func (w *closingListenerWrapper) Accept() (net.Conn, error) {
	conn, err := w.Listener.Accept()
	if err != nil {
		return nil, err
	}
	ch := make(chan struct{})
	cc := &closingConn{Conn: conn, closed: ch}

	w.mu.Lock()
	w.closed[cc] = ch
	w.mu.Unlock()

	// Increment closeEvents when this conn closes.
	go func() {
		<-ch
		w.closeEvents.Add(1)
	}()

	return cc, nil
}

// closeCounter returns a snapshot of the current close-event counter.
// Call this BEFORE the real client dials; then pass the result to waitForCloseAfter.
func (w *closingListenerWrapper) closeCounter() uint64 {
	return w.closeEvents.Load()
}

// waitForCloseAfter waits until the close-event counter exceeds baseline within d.
// This ignores any closes that occurred before the baseline snapshot (e.g. the
// awaitReady probe conn) and fires only when a new close is observed.
func (w *closingListenerWrapper) waitForCloseAfter(baseline uint64, d time.Duration) error {
	deadline := time.Now().UTC().Add(d)
	for {
		if w.closeEvents.Load() > baseline {
			return nil
		}
		if time.Now().UTC().After(deadline) {
			return fmt.Errorf("no tracked conn observed close within %s", d)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// closingConn wraps a net.Conn, firing the closed channel on the first Read
// that returns io.EOF or net.ErrClosed. This signals that the remote client
// has sent FIN (graceful close) or the connection was forcibly closed.
type closingConn struct {
	net.Conn
	closeOnce sync.Once
	closed    chan struct{}
}

// Read delegates to the underlying Conn and fires closed on io.EOF or ErrClosed.
// AC-005: fire ONLY on client-side FIN (EOF) or server-initiated close (ErrClosed).
// Idle-timeout errors are NOT client closes.
func (c *closingConn) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	if err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
			c.closeOnce.Do(func() { close(c.closed) })
		}
	}
	return n, err
}

// ── per-mode handler constructors ─────────────────────────────────────────────
//
// Each constructor returns the handler set that corresponds to that daemon mode's
// actual registered subcommands per ARCH-12. The distinct tables exercise per-mode
// handler-registration differences even though runXxx entrypoints are not the
// test vehicle (Q1 ruling — Option A, AC-001).

func routerHandlers() []mgmt.Handler {
	return []mgmt.Handler{
		{
			Command: "paths.list",
			Fn: func(_ context.Context, _ json.RawMessage) (any, error) {
				return map[string]any{"paths": []any{}}, nil
			},
		},
		{
			Command: "status",
			Fn: func(_ context.Context, _ json.RawMessage) (any, error) {
				return map[string]any{"mode": "router"}, nil
			},
		},
	}
}

func accessHandlers() []mgmt.Handler {
	return []mgmt.Handler{
		{
			Command: "session.list",
			Fn: func(_ context.Context, _ json.RawMessage) (any, error) {
				return map[string]any{"sessions": []any{}}, nil
			},
		},
		{
			Command: "status",
			Fn: func(_ context.Context, _ json.RawMessage) (any, error) {
				return map[string]any{"mode": "access"}, nil
			},
		},
	}
}

func consoleHandlers() []mgmt.Handler {
	return []mgmt.Handler{
		{
			Command: "console.status",
			Fn: func(_ context.Context, _ json.RawMessage) (any, error) {
				// Distinct response from base status so cross-wiring bugs surface as data-mismatch.
				return map[string]any{"mode": "console", "status": "ok"}, nil
			},
		},
		{
			Command: "status",
			Fn: func(_ context.Context, _ json.RawMessage) (any, error) {
				return map[string]any{"mode": "console"}, nil
			},
		},
	}
}

func controlHandlers() []mgmt.Handler {
	return []mgmt.Handler{
		{
			Command: "admin.key.list",
			Fn: func(_ context.Context, _ json.RawMessage) (any, error) {
				return map[string]any{"keys": []any{}}, nil
			},
		},
		{
			Command: "status",
			Fn: func(_ context.Context, _ json.RawMessage) (any, error) {
				return map[string]any{"mode": "control"}, nil
			},
		},
	}
}
