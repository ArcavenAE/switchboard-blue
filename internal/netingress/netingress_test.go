package netingress

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/frame"
)

// makeFrameBytes returns the wire bytes for a single frame: encoded outer
// header followed by payload. HMACTag is left zero — routing logic tests
// verify HMAC; ingress tests only assert framing.
func makeFrameBytes(t *testing.T, ft frame.FrameType, svtn [16]byte, src, dst [8]byte, payload []byte) []byte {
	t.Helper()
	hdr := frame.OuterHeader{
		Version:    frame.VersionByte,
		FrameType:  ft,
		PayloadLen: uint16(len(payload)),
		SVTNID:     svtn,
		SrcAddr:    src,
		DstAddr:    dst,
	}
	encoded := frame.EncodeOuterHeader(hdr)
	buf := make([]byte, 0, len(encoded)+len(payload))
	buf = append(buf, encoded[:]...)
	buf = append(buf, payload...)
	return buf
}

func TestReadFrame_HappyPath(t *testing.T) {
	t.Parallel()
	payload := []byte("hello switchboard")
	svtn := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	src := [8]byte{0xa1, 0xa2, 0xa3, 0xa4, 0xa5, 0xa6, 0xa7, 0xa8}
	dst := [8]byte{0xb1, 0xb2, 0xb3, 0xb4, 0xb5, 0xb6, 0xb7, 0xb8}

	wire := makeFrameBytes(t, frame.FrameTypeData, svtn, src, dst, payload)
	r := bytes.NewReader(wire)

	hdr, gotPayload, err := ReadFrame(r)
	if err != nil {
		t.Fatalf("ReadFrame: unexpected error: %v", err)
	}
	if hdr.FrameType != frame.FrameTypeData {
		t.Errorf("FrameType: got %v want %v", hdr.FrameType, frame.FrameTypeData)
	}
	if hdr.PayloadLen != uint16(len(payload)) {
		t.Errorf("PayloadLen: got %d want %d", hdr.PayloadLen, len(payload))
	}
	if hdr.SVTNID != svtn {
		t.Errorf("SVTNID: got %x want %x", hdr.SVTNID, svtn)
	}
	if hdr.SrcAddr != src {
		t.Errorf("SrcAddr: got %x want %x", hdr.SrcAddr, src)
	}
	if hdr.DstAddr != dst {
		t.Errorf("DstAddr: got %x want %x", hdr.DstAddr, dst)
	}
	if !bytes.Equal(gotPayload, payload) {
		t.Errorf("payload: got %q want %q", gotPayload, payload)
	}
}

func TestReadFrame_ZeroLenPayload(t *testing.T) {
	t.Parallel()
	wire := makeFrameBytes(t, frame.FrameTypeEmptyTick, [16]byte{}, [8]byte{}, [8]byte{}, nil)
	r := bytes.NewReader(wire)

	hdr, payload, err := ReadFrame(r)
	if err != nil {
		t.Fatalf("ReadFrame: unexpected error: %v", err)
	}
	if hdr.FrameType != frame.FrameTypeEmptyTick {
		t.Errorf("FrameType: got %v want %v", hdr.FrameType, frame.FrameTypeEmptyTick)
	}
	if len(payload) != 0 {
		t.Errorf("payload length: got %d want 0", len(payload))
	}
}

func TestReadFrame_CleanEOFAtBoundary(t *testing.T) {
	t.Parallel()
	// Empty reader → clean EOF at header boundary.
	_, _, err := ReadFrame(bytes.NewReader(nil))
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF at clean stream end, got %v", err)
	}
}

func TestReadFrame_TruncatedHeader(t *testing.T) {
	t.Parallel()
	// One byte less than a full header → truncation, not clean end.
	buf := make([]byte, frame.OuterHeaderSize-1)
	buf[0] = frame.VersionByte
	buf[1] = byte(frame.FrameTypeData)

	_, _, err := ReadFrame(bytes.NewReader(buf))
	if err == nil {
		t.Fatalf("expected error on truncated header")
	}
	if errors.Is(err, io.EOF) {
		t.Fatalf("truncated header must not surface as clean io.EOF, got %v", err)
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("expected io.ErrUnexpectedEOF chain, got %v", err)
	}
}

func TestReadFrame_TruncatedPayload(t *testing.T) {
	t.Parallel()
	// Header claims 100-byte payload, but only 10 bytes follow.
	svtn := [16]byte{}
	src := [8]byte{}
	dst := [8]byte{}
	hdr := frame.OuterHeader{
		Version:    frame.VersionByte,
		FrameType:  frame.FrameTypeData,
		PayloadLen: 100,
		SVTNID:     svtn,
		SrcAddr:    src,
		DstAddr:    dst,
	}
	encoded := frame.EncodeOuterHeader(hdr)
	buf := append([]byte{}, encoded[:]...)
	buf = append(buf, bytes.Repeat([]byte{0xff}, 10)...) // short payload

	_, _, err := ReadFrame(bytes.NewReader(buf))
	if err == nil {
		t.Fatalf("expected error on truncated payload")
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("expected io.ErrUnexpectedEOF chain, got %v", err)
	}
}

func TestReadFrame_InvalidVersion(t *testing.T) {
	t.Parallel()
	buf := make([]byte, frame.OuterHeaderSize)
	buf[0] = 0xF0 // major=15
	buf[1] = byte(frame.FrameTypeData)

	_, _, err := ReadFrame(bytes.NewReader(buf))
	if err == nil {
		t.Fatalf("expected version error")
	}
	if !errors.Is(err, frame.ErrVersionMismatch) {
		t.Fatalf("expected ErrVersionMismatch chain, got %v", err)
	}
}

func TestReadFrame_InvalidFrameType(t *testing.T) {
	t.Parallel()
	buf := make([]byte, frame.OuterHeaderSize)
	buf[0] = frame.VersionByte
	buf[1] = 0xAA // reserved

	_, _, err := ReadFrame(bytes.NewReader(buf))
	if err == nil {
		t.Fatalf("expected frame-type error")
	}
	if !errors.Is(err, frame.ErrInvalidFrameType) {
		t.Fatalf("expected ErrInvalidFrameType chain, got %v", err)
	}
}

func TestReadFrame_TwoFramesBackToBack(t *testing.T) {
	t.Parallel()
	// Framing is self-delimiting via PayloadLen: reading the first frame
	// leaves the stream positioned at the start of the second.
	a := makeFrameBytes(t, frame.FrameTypeData, [16]byte{}, [8]byte{}, [8]byte{}, []byte("A"))
	b := makeFrameBytes(t, frame.FrameTypeData, [16]byte{}, [8]byte{}, [8]byte{}, []byte("BB"))
	r := bytes.NewReader(append(a, b...))

	_, p1, err := ReadFrame(r)
	if err != nil {
		t.Fatalf("frame 1: %v", err)
	}
	if string(p1) != "A" {
		t.Errorf("frame 1 payload: got %q want %q", p1, "A")
	}
	_, p2, err := ReadFrame(r)
	if err != nil {
		t.Fatalf("frame 2: %v", err)
	}
	if string(p2) != "BB" {
		t.Errorf("frame 2 payload: got %q want %q", p2, "BB")
	}
	// Third read should hit clean EOF.
	_, _, err = ReadFrame(r)
	if !errors.Is(err, io.EOF) {
		t.Errorf("expected io.EOF after two frames, got %v", err)
	}
}

// recorder is a test double for RouteFn that captures dispatched frames.
type recorder struct {
	mu     sync.Mutex
	frames []frameRecord
	errOn  int // return an error on the N-th call (1-indexed); 0 = never
}

type frameRecord struct {
	hdr     frame.OuterHeader
	payload []byte
}

func (r *recorder) route(hdr frame.OuterHeader, payload []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.frames = append(r.frames, frameRecord{hdr: hdr, payload: append([]byte(nil), payload...)})
	if r.errOn > 0 && len(r.frames) == r.errOn {
		return errors.New("route error injected")
	}
	return nil
}

func (r *recorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.frames)
}

// captureLogger is a Logger that captures messages for assertion.
type captureLogger struct {
	mu   sync.Mutex
	msgs []string
}

func (l *captureLogger) Log(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.msgs = append(l.msgs, msg)
}

func (l *captureLogger) snapshot() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]string, len(l.msgs))
	copy(out, l.msgs)
	return out
}

// waitFor polls fn every 5ms until it returns true or deadline expires.
// Fail-closed on timeout with fmt lines identifying the caller. Matches
// TestRegister_AfterServeReturnsError discipline (no time.Sleep as barrier).
func waitFor(t *testing.T, deadline time.Duration, fn func() bool, label string) {
	t.Helper()
	end := time.Now().Add(deadline)
	for time.Now().Before(end) {
		if fn() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("waitFor %q: deadline %v exceeded", label, deadline)
}

func TestServeConn_DispatchesFramesUntilEOF(t *testing.T) {
	t.Parallel()
	// Simulate a conn by using net.Pipe.
	client, server := net.Pipe()
	defer func() { _ = client.Close() }()

	rec := &recorder{}
	logger := &captureLogger{}

	done := make(chan error, 1)
	go func() {
		done <- ServeConn(context.Background(), server, rec.route, logger)
	}()

	// Write three frames.
	frames := [][]byte{
		makeFrameBytes(t, frame.FrameTypeData, [16]byte{1}, [8]byte{2}, [8]byte{3}, []byte("one")),
		makeFrameBytes(t, frame.FrameTypeData, [16]byte{4}, [8]byte{5}, [8]byte{6}, []byte("two")),
		makeFrameBytes(t, frame.FrameTypeEmptyTick, [16]byte{}, [8]byte{}, [8]byte{}, nil),
	}
	for _, f := range frames {
		if _, err := client.Write(f); err != nil {
			t.Fatalf("client write: %v", err)
		}
	}

	waitFor(t, 2*time.Second, func() bool { return rec.count() == 3 }, "three frames dispatched")

	// Close client → server side sees EOF → ServeConn returns nil.
	_ = client.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("ServeConn: got %v want nil on clean close", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("ServeConn did not return after client close")
	}
}

func TestServeConn_CtxCancelReturns(t *testing.T) {
	t.Parallel()
	_, server := net.Pipe()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- ServeConn(ctx, server, func(frame.OuterHeader, []byte) error { return nil }, nil)
	}()

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("ServeConn on ctx cancel: got %v want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("ServeConn did not return after ctx cancel")
	}
}

func TestServeConn_RouteErrorDoesNotDropConnection(t *testing.T) {
	t.Parallel()
	client, server := net.Pipe()
	defer func() { _ = client.Close() }()

	rec := &recorder{errOn: 1} // first frame returns error
	done := make(chan error, 1)
	go func() {
		done <- ServeConn(context.Background(), server, rec.route, nil)
	}()

	f1 := makeFrameBytes(t, frame.FrameTypeData, [16]byte{1}, [8]byte{}, [8]byte{}, []byte("bad"))
	f2 := makeFrameBytes(t, frame.FrameTypeData, [16]byte{2}, [8]byte{}, [8]byte{}, []byte("good"))
	if _, err := client.Write(f1); err != nil {
		t.Fatalf("write f1: %v", err)
	}
	if _, err := client.Write(f2); err != nil {
		t.Fatalf("write f2: %v", err)
	}

	waitFor(t, 2*time.Second, func() bool { return rec.count() == 2 }, "both frames dispatched despite route error")

	_ = client.Close()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("ServeConn did not return")
	}
}

func TestServeConn_MalformedFrameDropsConnection(t *testing.T) {
	t.Parallel()
	client, server := net.Pipe()
	defer func() { _ = client.Close() }()

	rec := &recorder{}
	logger := &captureLogger{}

	done := make(chan error, 1)
	go func() {
		done <- ServeConn(context.Background(), server, rec.route, logger)
	}()

	// Send a header with an invalid frame type.
	bad := make([]byte, frame.OuterHeaderSize)
	bad[0] = frame.VersionByte
	bad[1] = 0xEE // reserved
	if _, err := client.Write(bad); err != nil {
		t.Fatalf("write bad frame: %v", err)
	}

	select {
	case err := <-done:
		if err == nil {
			t.Fatalf("ServeConn on malformed frame: got nil, want non-nil")
		}
		if !errors.Is(err, frame.ErrInvalidFrameType) {
			t.Fatalf("expected ErrInvalidFrameType chain, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("ServeConn did not return on malformed frame")
	}

	// Route should never have been invoked.
	if rec.count() != 0 {
		t.Errorf("route calls on malformed frame: got %d want 0", rec.count())
	}
	// Logger should have captured the read error.
	msgs := logger.snapshot()
	found := false
	for _, m := range msgs {
		if strings.Contains(m, "netingress: read frame") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected read-error log; got %v", msgs)
	}
}

func TestServe_AcceptsMultipleConnectionsAndJoinsOnCtxCancel(t *testing.T) {
	t.Parallel()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()

	var routed atomic.Int64
	route := func(frame.OuterHeader, []byte) error { //nolint:unparam // matches RouteFn signature; test route never fails
		routed.Add(1)
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serveDone := make(chan error, 1)
	go func() {
		serveDone <- Serve(ctx, ln, route, nil)
	}()

	// Open two connections, send one frame each.
	dial := func() net.Conn {
		c, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			t.Fatalf("dial: %v", err)
		}
		return c
	}
	c1 := dial()
	c2 := dial()
	f := makeFrameBytes(t, frame.FrameTypeData, [16]byte{1}, [8]byte{}, [8]byte{}, []byte("x"))
	if _, err := c1.Write(f); err != nil {
		t.Fatalf("c1 write: %v", err)
	}
	if _, err := c2.Write(f); err != nil {
		t.Fatalf("c2 write: %v", err)
	}

	waitFor(t, 2*time.Second, func() bool { return routed.Load() == 2 }, "two frames routed")

	_ = c1.Close()
	_ = c2.Close()

	cancel()

	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("Serve: got %v want nil on ctx cancel", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Serve did not return after ctx cancel")
	}
}

func TestServe_ClosesListenerOnCtxCancel(t *testing.T) {
	t.Parallel()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- Serve(ctx, ln, func(frame.OuterHeader, []byte) error { return nil }, nil)
	}()

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve: got %v want nil on ctx cancel", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Serve did not return after ctx cancel")
	}

	// Dialing after cancel should fail (listener closed).
	if _, err := net.DialTimeout("tcp", ln.Addr().String(), 200*time.Millisecond); err == nil {
		t.Errorf("expected dial to fail after Serve returned; listener should be closed")
	}
}
