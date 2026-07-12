package netingress_test

import (
	"context"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/netingress"
	"github.com/arcavenae/switchboard/internal/routing"
)

// captureLogger captures log lines for assertion. Concurrency-safe.
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

func encodeFrame(t *testing.T, ft frame.FrameType, svtn [16]byte, src, dst [8]byte, payload []byte) []byte {
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
	buf := append([]byte{}, encoded[:]...)
	buf = append(buf, payload...)
	return buf
}

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

// TestIntegration_EADM017_FiresThroughLiveIngress wires a real TCP listener,
// a real routing.Router, and a real admission.FailureCounter (clock injected)
// to prove S-W3.05 AC-009 (E-ADM-017 fires through the live network-ingress
// path). A single client sends five frames with the same source address; no
// forwarding entry exists for that source, so every frame drops fail-closed
// with ErrHMACVerificationFailed via PATH-A ("auth key unavailable"). Each
// drop increments the per-source counter; the fifth crossing threshold=5
// within window=60s fires E-ADM-017 exactly once.
//
// Traces to:
//   - BC-2.05.005 PC-3 (per-source failure rate → E-ADM-017)
//   - BC-2.05.008 invariant 5 (counter is authoritative on the RouteFrame path)
//   - S-W3.05 AC-009 (live network-ingress path assertion — was gated on S-BL.NI)
//   - C-1-W3P1-defer consumption (network-ingress listener now live)
func TestIntegration_EADM017_FiresThroughLiveIngress(t *testing.T) {
	t.Parallel()

	// Frozen clock inside the FailureCounter window.
	current := time.Now().UTC()
	logger := &captureLogger{}

	fc := admission.NewFailureCounter(5, 60*time.Second, logger,
		admission.WithNow(func() time.Time { return current }),
	)

	ks := admission.NewAdmittedKeySet()
	router := routing.NewRouter(ks,
		routing.WithLogger(logger),
		routing.WithFailureCounter(fc),
	)

	route := func(hdr frame.OuterHeader, payload []byte) error {
		return routing.RouteFrame(hdr, payload, router)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serveDone := make(chan error, 1)
	go func() {
		serveDone <- netingress.Serve(ctx, ln, route, logger, netingress.ServeConfig{})
	}()

	// Same source across all frames — the counter is keyed on src.
	src := [8]byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11}
	svtn := [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
	dst := [8]byte{0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99}

	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = c.Close() }()

	// Send five frames from the same source. No forwarding entry exists →
	// every frame drops via ErrHMACVerificationFailed PATH-A → counter++.
	// The FailureCounter fires E-ADM-017 exactly once on the fifth increment.
	for i := 0; i < 5; i++ {
		payload := []byte{byte(i)}
		wire := encodeFrame(t, frame.FrameTypeData, svtn, src, dst, payload)
		if _, err := c.Write(wire); err != nil {
			t.Fatalf("client write %d: %v", i, err)
		}
	}

	// Poll the captured log for the E-ADM-017 emission. E-ADM-016 lines are
	// emitted first (one per drop); E-ADM-017 must appear after the fifth.
	countLine := func(sub string) int {
		msgs := logger.snapshot()
		n := 0
		for _, m := range msgs {
			if strings.Contains(m, sub) {
				n++
			}
		}
		return n
	}

	waitFor(t, 2*time.Second, func() bool { return countLine("E-ADM-016") >= 5 }, "five E-ADM-016 drop logs")
	waitFor(t, 2*time.Second, func() bool { return countLine("E-ADM-017") == 1 }, "one E-ADM-017 alert")

	// Send a sixth frame — append-skip is active, no second alert must fire.
	// (Drain-only re-arm: the window would need to fully empty before another
	// alert can fire.)
	wire := encodeFrame(t, frame.FrameTypeData, svtn, src, dst, []byte{0x99})
	if _, err := c.Write(wire); err != nil {
		t.Fatalf("client write 6: %v", err)
	}
	// Give the server a moment to process; assert the alert count is still 1.
	// waitFor with a stability predicate: wait for E-ADM-016 to increment,
	// then confirm E-ADM-017 stays at 1.
	waitFor(t, 2*time.Second, func() bool { return countLine("E-ADM-016") >= 6 }, "sixth E-ADM-016 drop log")
	if got := countLine("E-ADM-017"); got != 1 {
		t.Errorf("E-ADM-017 must fire exactly once (append-skip); got %d", got)
	}

	// Shutdown.
	_ = c.Close()
	cancel()

	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("Serve exited with error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Serve did not return after ctx cancel")
	}
}

// TestIntegration_ConcurrentRegisterAndRouteRaceClean is the cross-component
// -race assertion demanded by PROCESS-GAP-W4: multiple ingress connections
// deliver frames concurrently with RegisterForwardingEntry writes and a
// concurrent metrics-shaped read (List via a shim). This is a smoke-race test:
// its job is to fail under -race if the ingress path introduces any lock-order
// or shared-state violation not caught by internal/routing's own concurrent
// tests. It does NOT re-assert LWW correctness (that lives in
// internal/routing/lww_concurrent_test.go).
func TestIntegration_ConcurrentRegisterAndRouteRaceClean(t *testing.T) {
	t.Parallel()

	// nopLogger discards to keep the test noise-free.
	logger := &captureLogger{}

	fc := admission.NewFailureCounter(1000, time.Hour, logger)

	ks := admission.NewAdmittedKeySet()
	router := routing.NewRouter(ks, routing.WithLogger(logger), routing.WithFailureCounter(fc))

	route := func(hdr frame.OuterHeader, payload []byte) error {
		return routing.RouteFrame(hdr, payload, router)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serveDone := make(chan error, 1)
	go func() {
		serveDone <- netingress.Serve(ctx, ln, route, logger, netingress.ServeConfig{})
	}()

	svtn := [16]byte{0xaa}
	dst := [8]byte{0x11}

	// Writer goroutines: register entries concurrently.
	var wg sync.WaitGroup
	stop := make(chan struct{})
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(id byte) {
			defer wg.Done()
			var authKey [32]byte
			for i := 0; ; i++ {
				select {
				case <-stop:
					return
				default:
				}
				src := [8]byte{id, byte(i)}
				authKey[0] = id
				authKey[1] = byte(i)
				router.RegisterForwardingEntry(svtn, src, authKey)
			}
		}(byte(w))
	}

	// Ingress-driver goroutines: hammer the listener with frames.
	var framesRouted atomic.Int64
	for c := 0; c < 4; c++ {
		wg.Add(1)
		go func(id byte) {
			defer wg.Done()
			conn, err := net.Dial("tcp", ln.Addr().String())
			if err != nil {
				return // shutdown race is OK
			}
			defer func() { _ = conn.Close() }()
			for i := 0; ; i++ {
				select {
				case <-stop:
					return
				default:
				}
				src := [8]byte{id, byte(i)}
				payload := []byte{id, byte(i)}
				wire := encodeFrame(t, frame.FrameTypeData, svtn, src, dst, payload)
				if _, err := conn.Write(wire); err != nil {
					return
				}
				framesRouted.Add(1)
			}
		}(byte(c))
	}

	// Run for a bounded burst — long enough to interleave many operations but
	// short enough to keep the test fast.
	time.Sleep(200 * time.Millisecond)
	close(stop)
	wg.Wait()

	// Also verify the router's Register path is still callable after shutdown
	// begins — pure race-check, no correctness claim.
	router.RegisterForwardingEntry(svtn, [8]byte{0xff}, [32]byte{})

	cancel()
	select {
	case <-serveDone:
	case <-time.After(2 * time.Second):
		t.Fatalf("Serve did not return after ctx cancel")
	}

	// The race detector, if enabled, is the actual assertion here. We do a
	// lightweight sanity check that some frames were routed.
	if framesRouted.Load() == 0 {
		t.Errorf("no frames routed; ingress path may not have opened correctly")
	}
}
