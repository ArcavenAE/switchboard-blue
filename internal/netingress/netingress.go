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
// Import constraints (ARCH-08 §6): this package MAY import internal/frame
// only. It receives a RouteFn from callers to avoid importing internal/routing
// (which would invert the ARCH-08 layering; ingress is upstream of routing).
package netingress

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/arcavenae/switchboard/internal/frame"
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

// Serve accepts connections on ln and spawns a per-connection goroutine
// running ServeConn against route. Returns when ln.Accept fails permanently
// or ctx is cancelled. All outstanding per-connection goroutines are joined
// before Serve returns (ARCH-01 goroutine lifecycle contract).
//
// CWE-770 mitigation: no more than MaxConcurrentConnections per-conn goroutines
// run at once. Excess connections are accepted (so the client sees a connect
// success rather than a refused connect at kernel level) then immediately closed
// to shed load.
func Serve(ctx context.Context, ln net.Listener, route RouteFn, logger Logger) error {
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
		// count bounded (CWE-770).
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
			defer c.Close()
			_ = ServeConn(ctx, c, route, logger)
		}(conn)
	}
}
