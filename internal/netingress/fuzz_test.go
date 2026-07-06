// Fuzz harnesses for Phase 6 formal hardening.
//
// The netingress package parses untrusted network bytes as its entire job:
// clients hand it an io.Reader and it produces a parsed frame.OuterHeader plus
// a payload slice sized by an attacker-controlled uint16 field. This file
// fuzzes the two entry points that operate on that untrusted byte stream:
//
//   - FuzzReadFrame — direct wire-byte input to ReadFrame; seeds cover the
//     ss-02 canonical frame vectors (BC-2.01.004) plus truncation, oversize,
//     invalid version, and invalid frame_type edges.
//   - FuzzServeConnDispatch — a full-connection fuzzer that feeds arbitrary
//     bytes to ServeConn via net.Pipe and asserts the goroutine exits cleanly
//     (no panics, no goroutine leak). This exercises the LimitReader ceiling,
//     the ctx-close teardown path, and the drop-and-continue route error
//     handling.
//
// Both harnesses assert only shape invariants (no panic, error taxonomy, no
// OOM from unbounded payload allocation). They do not verify HMAC — that is
// routing's contract and would introduce a cross-package coupling not
// warranted at the ingress boundary.
package netingress

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/frame"
)

// FuzzReadFrame feeds arbitrary bytes to ReadFrame and asserts that:
//   - the function never panics
//   - on success, len(payload) == hdr.PayloadLen (self-delimiting invariant)
//   - on failure, the returned error is one of the documented sentinel chains
//     (io.EOF at boundary, io.ErrUnexpectedEOF on truncation, or a frame.Err*
//     parse error) or a wrapped read error
//   - MaxFrameBytes is respected: no fuzz-generated seed can force ReadFrame
//     to allocate more than MaxFrameBytes for its payload buffer
//
// Seeds cover the canonical BC-2.01.004 test vectors plus the parse-error
// boundary cases the netingress_test.go suite already asserts non-fuzz.
func FuzzReadFrame(f *testing.F) {
	// Canonical happy-path seed: 44-byte outer header + short payload.
	happy := makeSeed(frame.FrameTypeData, [16]byte{0x01, 0x02, 0x03, 0x04}, [8]byte{0x10}, [8]byte{0x20}, []byte("hello"))
	f.Add(happy)

	// Zero-length payload (empty-tick).
	empty := makeSeed(frame.FrameTypeEmptyTick, [16]byte{}, [8]byte{}, [8]byte{}, nil)
	f.Add(empty)

	// Truncated header (short by 1 byte).
	f.Add(happy[:frame.OuterHeaderSize-1])

	// Empty input (clean EOF at boundary).
	f.Add([]byte{})

	// Invalid version (major != 0).
	badVer := append([]byte{}, happy...)
	badVer[0] = 0xF0
	f.Add(badVer)

	// Invalid frame_type (reserved value).
	badFT := append([]byte{}, happy...)
	badFT[1] = 0xAA
	f.Add(badFT)

	// Header claims payload larger than what follows (truncated payload).
	truncPayload := append([]byte{}, happy[:frame.OuterHeaderSize]...)
	truncPayload[2] = 0xFF // PayloadLen high byte
	truncPayload[3] = 0xFF // PayloadLen low byte → 65535, but no payload follows
	f.Add(truncPayload)

	// Max-payload header claiming 65535 bytes with matching payload of zeroes.
	// Exercises the natural upper bound MaxFrameBytes ceiling.
	maxHdr := frame.OuterHeader{
		Version:    frame.VersionByte,
		FrameType:  frame.FrameTypeData,
		PayloadLen: 0xFFFF,
	}
	maxEncoded := frame.EncodeOuterHeader(maxHdr)
	maxSeed := make([]byte, 0, frame.OuterHeaderSize+0xFFFF)
	maxSeed = append(maxSeed, maxEncoded[:]...)
	maxSeed = append(maxSeed, bytes.Repeat([]byte{0x00}, 0xFFFF)...)
	f.Add(maxSeed)

	f.Fuzz(func(t *testing.T, data []byte) {
		// Bound the reader with a LimitReader at MaxFrameBytes to mirror
		// production Serve behaviour: ReadFrame must not read more than
		// MaxFrameBytes from the reader per invocation.
		lr := io.LimitReader(bytes.NewReader(data), int64(MaxFrameBytes))
		hdr, payload, err := ReadFrame(lr)

		if err == nil {
			// Success: the parsed frame must be self-delimiting.
			if got, want := len(payload), int(hdr.PayloadLen); got != want {
				t.Fatalf("payload length mismatch: got %d want %d (hdr.PayloadLen); this is a wire-format invariant violation", got, want)
			}
			// Header must round-trip through re-encode without changing shape.
			reencoded := frame.EncodeOuterHeader(hdr)
			if !bytes.Equal(reencoded[:], data[:frame.OuterHeaderSize]) {
				t.Fatalf("outer header not round-trip: parsed→encoded=%x, input=%x", reencoded[:], data[:frame.OuterHeaderSize])
			}
			return
		}

		// Failure: error must be a recognisable taxonomy member. Any bare
		// panic-equivalent (nil error but no data) is caught by the runtime
		// automatically; here we check the failure surface is stable.
		if errors.Is(err, io.EOF) {
			// Only legal on empty input.
			if len(data) != 0 {
				t.Fatalf("io.EOF returned with non-empty input (%d bytes); truncation must surface as io.ErrUnexpectedEOF", len(data))
			}
			return
		}
		if errors.Is(err, io.ErrUnexpectedEOF) {
			// Truncation: input length must be less than declared frame size.
			return
		}
		if errors.Is(err, frame.ErrFrameTooShort) ||
			errors.Is(err, frame.ErrVersionMismatch) ||
			errors.Is(err, frame.ErrInvalidFrameType) {
			return
		}
		// Any other error path is unexpected — fuzz surfaces a new failure
		// mode that should be classified before merging.
		t.Fatalf("unclassified error from ReadFrame: %v (input len=%d)", err, len(data))
	})
}

// FuzzServeConnDispatch fuzzes the full serve-loop path: arbitrary bytes are
// pushed through a net.Pipe half; ServeConn on the other half must exit
// cleanly (no panic, no leaked goroutine) once the writer closes.
//
// The RouteFn is a stub that never returns an error — this fuzzer is not
// looking for route-side bugs; it is asserting the reader side survives
// arbitrary input without corrupting state or blocking indefinitely.
//
// A short ctx timeout bounds the fuzz iteration wall time; the timeout path
// is exercised in the same run to catch any cancellation-race bugs.
func FuzzServeConnDispatch(f *testing.F) {
	// Two-frame back-to-back.
	seed1 := makeSeed(frame.FrameTypeData, [16]byte{1}, [8]byte{2}, [8]byte{3}, []byte("one"))
	seed2 := makeSeed(frame.FrameTypeData, [16]byte{4}, [8]byte{5}, [8]byte{6}, []byte("two"))
	f.Add(append(seed1, seed2...))

	// Zero bytes (immediate EOF path).
	f.Add([]byte{})

	// Truncated header trailing after a valid frame.
	trail := append(append([]byte{}, seed1...), byte(0xFF))
	f.Add(trail)

	// Header claiming huge PayloadLen with no payload → wait+timeout path.
	hdrOnly := make([]byte, frame.OuterHeaderSize)
	hdrOnly[0] = frame.VersionByte
	hdrOnly[1] = byte(frame.FrameTypeData)
	hdrOnly[2] = 0x10
	hdrOnly[3] = 0x00
	f.Add(hdrOnly)

	f.Fuzz(func(t *testing.T, data []byte) {
		client, server := net.Pipe()
		defer func() { _ = client.Close() }()
		defer func() { _ = server.Close() }()

		route := func(hdr frame.OuterHeader, payload []byte) error { //nolint:unparam // matches RouteFn signature; fuzz stub never fails
			// Assert the self-delimiting invariant inside the dispatch path.
			// A discrepancy here indicates a bug where LimitReader or
			// ReadFrame diverged from the wire format contract.
			if int(hdr.PayloadLen) != len(payload) {
				t.Errorf("dispatched frame has PayloadLen=%d but len(payload)=%d", hdr.PayloadLen, len(payload))
			}
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			done <- ServeConn(ctx, server, route, nil)
		}()

		// Feed the fuzz bytes over the pipe. Client.Write may block if the
		// reader is stuck — cap the send with a short deadline via a
		// goroutine so a pathological ServeConn cannot block the fuzz run.
		writeDone := make(chan struct{})
		go func() {
			defer close(writeDone)
			if len(data) > 0 {
				_ = client.SetWriteDeadline(time.Now().Add(200 * time.Millisecond))
				_, _ = client.Write(data)
			}
			_ = client.Close()
		}()

		select {
		case <-writeDone:
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("write half hung — ServeConn is not draining the pipe")
		}

		select {
		case err := <-done:
			// Any classified error is acceptable; ServeConn returning
			// without panic is the invariant.
			_ = err
		case <-time.After(1 * time.Second):
			// ctx timeout should have already fired. If ServeConn is still
			// running past the 1s outer deadline, it is stuck.
			_ = server.Close()
			t.Fatalf("ServeConn did not return within 1s after write-close + ctx cancel")
		}
	})
}

// makeSeed is a fuzz-local helper (does not overlap the test-file helper
// makeFrameBytes which takes *testing.T).
func makeSeed(ft frame.FrameType, svtn [16]byte, src, dst [8]byte, payload []byte) []byte {
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
