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
	"io"
	"net"
	"sync"
	"time"

	"github.com/arcavenae/switchboard/internal/mgmt"
)

// ── closingListenerWrapper ─────────────────────────────────────────────────────
//
// Wraps a net.Listener. Every Conn returned by Accept() is wrapped in a
// closingConn that fires a channel when the server-side Read() returns io.EOF
// (i.e., when the client has closed its side of the connection). This instruments
// the actual production `defer conn.Close()` in connectAndRun — not a tautological
// local close check (Q6 ruling — Option A, AC-005, BC-2.07.002 Inv-2).
//
// Thread safety: the closed map is protected by mu. callerClosedWithin snapshots
// the channel under mu; callers must not hold mu themselves.

type closingListenerWrapper struct {
	net.Listener
	mu     sync.Mutex
	closed map[net.Conn]chan struct{} // fired when server-side Read returns io.EOF
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

	return cc, nil
}

// clientClosedWithin returns nil if the most-recently-accepted connection's
// closed channel fires within d, or an error on timeout.
// It searches for the first entry whose channel has fired (or will fire within d).
// If no connections have been tracked, it returns an error immediately.
func (w *closingListenerWrapper) clientClosedWithin(conn net.Conn, d time.Duration) error {
	w.mu.Lock()
	ch, ok := w.closed[conn]
	w.mu.Unlock()

	if !ok {
		return errors.New("closingListenerWrapper: conn not found in closed map")
	}

	select {
	case <-ch:
		return nil
	case <-time.After(d):
		return errors.New("closingListenerWrapper: client did not close connection within deadline")
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
func (c *closingConn) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	if err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
			c.closeOnce.Do(func() { close(c.closed) })
		}
		// Also fire on any read-of-closed-conn scenario (OS-level).
		var ne net.Error
		if errors.As(err, &ne) {
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
				return map[string]any{"mode": "console"}, nil
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
