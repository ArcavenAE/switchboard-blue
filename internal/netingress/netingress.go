// Package netingress implements the router data-plane network ingress:
// a TCP listener that accepts node connections, reads self-delimiting
// framed messages (44-byte outer header carries PayloadLen), and dispatches
// each frame to a RouteFn.
//
// Story: S-BL.NI (network-ingress). This is the seam that replaces the
// placeholder accept-and-close loop in cmd/switchboard runRouter. It closes
// C-1-W3P1-defer (network-ingress listener) and lets BC-2.09.003 PC-9
// (cfg.ListenAddr application) become a live path.
//
// Classification (ARCH-09 v1.1): boundary — owns per-connection goroutines,
// bounded-reader state (CWE-400), and a concurrency semaphore (CWE-770).
// No routing decisions are made here; those live in internal/routing.
//
// Import constraints (ARCH-08 §6.5): this package MAY import internal/frame
// and internal/routing. internal/routing is imported for the InterfaceID
// type only (NodeHandle.IfaceID / ServeConfig.IfaceIDSeed) — no routing
// decisions are made here; RouteFn dispatch stays the caller's concern. The
// netingress→routing edge is a forward edge in ARCH-08's topological
// ordering (netingress at §6.5 pos 18, routing at pos 17; 18 > 17), and
// routing does not import netingress, so no cycle is introduced.
package netingress

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"

	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/routing"
)

// MaxFrameBytes is the largest single framed message the ingress will read
// from a connection: 44-byte outer header + max PayloadLen (uint16 max).
// This is the natural upper bound implied by the wire format (ARCH-02 §3.1);
// any oversize claim is a protocol error, not a slow-consume attack.
//
// Traces to VP-066 (CWE-400 bounded reads) applied to the data plane.
const MaxFrameBytes = frame.OuterHeaderSize + int(^uint16(0))

// MaxConcurrentConnections caps the number of in-flight per-connection
// goroutines Serve will run. Excess connections are accepted then closed
// immediately to shed load without unbounded goroutine growth.
//
// Traces to VP-070 (CWE-770 goroutine exhaustion) applied to the data plane.
// Matches the internal/mgmt limit (128) for symmetry across ingress paths.
const MaxConcurrentConnections = 128

// Logger is the minimal logging interface netingress accepts. nopLogger is
// used when the caller does not inject one.
type Logger interface {
	Log(msg string)
}

type nopLogger struct{}

func (nopLogger) Log(string) {}

// RouteFn is the frame-dispatch function ingress calls after successfully
// reading a frame. It is called from the per-connection goroutine. *routing.Router
// is wired at the call site via a closure: func(h, p) error { return routing.RouteFrame(h, p, r) }.
//
// RouteFn returning a non-nil error is NOT a signal to close the connection
// (frames are independent under LWW + fail-closed admission); the ingress
// keeps reading. The error is logged and dropped. See BC-2.05.008 PC-4:
// every drop is already logged inside RouteFrame; ingress-level logging
// would double-count.
type RouteFn func(hdr frame.OuterHeader, payload []byte) error

// ReadFrame reads exactly one framed message from r: OuterHeaderSize bytes
// followed by hdr.PayloadLen bytes of payload. Returns the parsed header
// and payload slice.
//
// Returns io.EOF only when the reader is at a clean stream-end at the header
// boundary (zero bytes read). A truncated header or truncated payload returns
// io.ErrUnexpectedEOF wrapped with context (CWE-400 fail-closed on truncation).
//
// The returned payload is a freshly allocated slice; the caller owns it.
func ReadFrame(r io.Reader) (frame.OuterHeader, []byte, error) {
	var hdrBuf [frame.OuterHeaderSize]byte
	n, err := io.ReadFull(r, hdrBuf[:])
	if err != nil {
		// Clean EOF at header boundary: no bytes read → stream ended between frames.
		if err == io.EOF && n == 0 {
			return frame.OuterHeader{}, nil, io.EOF
		}
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return frame.OuterHeader{}, nil, fmt.Errorf("read outer header: %w", io.ErrUnexpectedEOF)
		}
		return frame.OuterHeader{}, nil, fmt.Errorf("read outer header: %w", err)
	}
	hdr, perr := frame.ParseOuterHeader(hdrBuf[:])
	if perr != nil {
		return frame.OuterHeader{}, nil, fmt.Errorf("parse outer header: %w", perr)
	}
	payload := make([]byte, int(hdr.PayloadLen))
	if _, err := io.ReadFull(r, payload); err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return frame.OuterHeader{}, nil, fmt.Errorf("read payload of %d bytes: %w", hdr.PayloadLen, io.ErrUnexpectedEOF)
		}
		return frame.OuterHeader{}, nil, fmt.Errorf("read payload of %d bytes: %w", hdr.PayloadLen, err)
	}
	return hdr, payload, nil
}

// ServeConn reads frames from conn in a loop until conn errors (including
// clean EOF) or ctx is cancelled, dispatching each frame to route.
//
// Bounded reads: each Read is preceded by an io.LimitReader capped at
// MaxFrameBytes; a client that opens a connection and sends nothing cannot
// consume more than that per attempted frame. Truncated frames drop the
// connection fail-closed.
//
// route errors are logged and dropped (see RouteFn doc for rationale);
// the connection stays open.
func ServeConn(ctx context.Context, conn net.Conn, route RouteFn, logger Logger) error {
	if logger == nil {
		logger = nopLogger{}
	}
	// Close conn when ctx is cancelled so a blocked Read returns.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()

	// LimitReader is refreshed per-frame; a single frame cannot exceed
	// MaxFrameBytes on the wire.
	for {
		lr := io.LimitReader(conn, int64(MaxFrameBytes))
		hdr, payload, err := ReadFrame(lr)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			// Cancelled ctx surfaces as a closed-conn read error; treat as clean.
			if ctx.Err() != nil {
				return nil
			}
			logger.Log(fmt.Sprintf("netingress: read frame from %s: %v", conn.RemoteAddr(), err))
			return err
		}
		if err := route(hdr, payload); err != nil {
			// Drop-and-continue: routing already logs E-ADM-016/017 per BC-2.05.008.
			continue
		}
	}
}

// NodeHandle is the write handle for an accepted node connection, fully
// populated by netingress BEFORE OnAccept is called (netingress owns DATA
// creation — see ServeConfig.OnAccept's Ownership split note). OnAccept
// receives it as a finished value; it does not construct any of its fields.
type NodeHandle struct {
	IfaceID routing.InterfaceID
	// Send is netingress-created. The caller stores it and selects/sends on
	// it; it is NEVER closed, by any goroutine, for the lifetime of the
	// process — enforced by convention (every call site honors this), not
	// by the type system.
	Send chan []byte
	// Done is netingress-created. It is closed EXACTLY ONCE by the caller,
	// through a caller-owned sync.Once — this package never calls
	// close(h.Done) directly.
	Done chan struct{}
}

// ServeConfig carries optional hooks and allocation parameters for Serve.
type ServeConfig struct {
	// OnAccept is called for each ADMITTED connection — i.e. one that has
	// cleared the CWE-770 concurrency-cap semaphore — NEVER for a
	// connection this Serve loop sheds. It fires as the FIRST ACT of the
	// newly spawned per-conn goroutine — NOT from the Serve accept loop
	// itself (Goroutine pin) — after netingress has allocated the
	// connection's InterfaceID and created its Send/Done channels in that
	// same goroutine (NodeHandle is fully populated before the call), and
	// strictly before ServeConn starts reading. The returned func()
	// (behavior cleanup) is deferred in that SAME per-conn goroutine,
	// registered AFTER the goroutine's defer wg.Done() so Go's LIFO defer
	// ordering guarantees cleanup completes before wg.Done() fires. Every
	// OnAccept call is paired 1:1, same-goroutine, with exactly one
	// guaranteed cleanup invocation. If nil, no hook fires and netingress
	// allocates nothing for that connection (zero-cost when unused); a
	// shed connection likewise allocates nothing and calls neither
	// OnAccept nor cleanup, regardless of whether OnAccept is nil.
	OnAccept func(conn net.Conn, h NodeHandle) func()

	// PerConnRoute, if non-nil, is called after OnAccept to obtain the
	// RouteFn for this specific connection (receiving the same conn and
	// NodeHandle as OnAccept). Serve passes the returned RouteFn to ServeConn
	// instead of the shared route. If nil, the shared route is used. This is
	// the mechanism for E-ADM-023 teardown (S-BL.NODE-IDENTIFY-WIRE rulings
	// §17): the per-conn route captures conn and calls conn.Close() on a
	// duplicate NodeIdentify.
	PerConnRoute func(conn net.Conn, h NodeHandle) RouteFn

	// IfaceIDSeed is the first InterfaceID netingress allocates for an
	// admitted connection when OnAccept is non-nil or PerConnRoute is
	// non-nil; subsequent admitted connections get IfaceIDSeed+1,
	// IfaceIDSeed+2, ... via an internal atomic counter. Shed connections
	// do not consume a value from this counter. Callers reserve low IDs
	// (e.g. a PE-upstream interface ID) by setting IfaceIDSeed above them.
	// Zero defaults to 2.
	IfaceIDSeed routing.InterfaceID
}

// Serve accepts connections on ln and spawns a per-connection goroutine
// running ServeConn against route. Returns when ln.Accept fails permanently
// or ctx is cancelled. All outstanding per-connection goroutines are joined
// before Serve returns (ARCH-01 goroutine lifecycle contract).
//
// CWE-770 mitigation: no more than MaxConcurrentConnections per-conn goroutines
// run at once. Excess connections are accepted (so the client sees a connect
// success rather than a refused connect at kernel level) then immediately closed
// to shed load.
//
// Runtime compat: callers passing ServeConfig{} (OnAccept == nil and
// PerConnRoute == nil) see unchanged runtime behavior — Serve allocates no
// IfaceID/Send/Done and calls no hook.
func Serve(ctx context.Context, ln net.Listener, route RouteFn, logger Logger, cfg ServeConfig) error {
	if logger == nil {
		logger = nopLogger{}
	}
	// Close listener when ctx is cancelled so a blocked Accept returns.
	acceptDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = ln.Close()
		case <-acceptDone:
		}
	}()
	defer close(acceptDone)

	sem := make(chan struct{}, MaxConcurrentConnections)
	var wg sync.WaitGroup

	// ifaceCounter allocates NodeHandle.IfaceID for admitted connections,
	// seeded by cfg.IfaceIDSeed (default 2). Started at seed-1 so the first
	// ifaceCounter.Add(1) below yields the seed itself.
	seed := cfg.IfaceIDSeed
	if seed == 0 {
		seed = 2
	}
	var ifaceCounter atomic.Uint64
	ifaceCounter.Store(uint64(seed) - 1)

	for {
		conn, err := ln.Accept()
		if err != nil {
			// Wait for outstanding per-conn goroutines before returning.
			wg.Wait()
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("netingress: accept: %w", err)
		}
		// Try to reserve a slot; if full, shed this connection to keep goroutine
		// count bounded (CWE-770). A shed connection never reaches cfg.OnAccept
		// (admission-gating) — it allocates no InterfaceID/Send/Done.
		select {
		case sem <- struct{}{}:
		default:
			logger.Log(fmt.Sprintf("netingress: connection cap reached, closing %s", conn.RemoteAddr()))
			_ = conn.Close()
			continue
		}
		wg.Add(1)
		go func(c net.Conn) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() { _ = c.Close() }()

			// OnAccept fires as the FIRST ACT of this goroutine (Goroutine
			// pin), never from the accept loop, once this connection has
			// cleared admission (Ownership split: netingress owns DATA
			// creation; the caller's closure owns BEHAVIOR only).
			// PerConnRoute, if set, is called with the same NodeHandle and
			// its returned RouteFn replaces the shared route for this conn
			// (E-ADM-023 teardown seam, rulings §17).
			var cleanup func()
			connRoute := route
			if cfg.OnAccept != nil || cfg.PerConnRoute != nil {
				h := NodeHandle{
					IfaceID: routing.InterfaceID(ifaceCounter.Add(1)),
					Send:    make(chan []byte, 32),
					Done:    make(chan struct{}),
				}
				if cfg.OnAccept != nil {
					cleanup = cfg.OnAccept(c, h)
				}
				if cfg.PerConnRoute != nil {
					connRoute = cfg.PerConnRoute(c, h)
				}
			}
			// Registered AFTER defer wg.Done() above so LIFO defer ordering
			// runs cleanup before wg.Done() fires.
			if cleanup != nil {
				defer cleanup()
			}
			_ = ServeConn(ctx, c, connRoute, logger)
		}(conn)
	}
}
