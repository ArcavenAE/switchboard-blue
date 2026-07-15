// discovery_wire_test.go covers AC-001: only the router-mode daemon joins
// the SVTN-scoped multicast discovery group.
package main

import (
	"context"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/discovery"
	"github.com/arcavenae/switchboard/internal/routing"
)

// redGateGuard recovers from a not-yet-implemented stub's panic and fails
// the test cleanly (Red Gate discipline, BC-5.38.001) instead of crashing
// the whole test binary. Once the relevant Task's Green step lands, the
// panic disappears and this guard becomes a silent no-op. Shared by every
// _test.go file in package main covering S-BL.DISCOVERY-WIRE.
func redGateGuard(t *testing.T) {
	t.Helper()
	if r := recover(); r != nil {
		t.Fatalf("red gate: stub not yet implemented: %v", r)
	}
}

// callWireDiscoveryListenerRecovered calls wireDiscoveryListener, recovering
// any panic into a returned value instead of letting it cross the goroutine
// boundary — recover() only catches a panic within the SAME goroutine's own
// deferred function, so a `defer redGateGuard(t)` in the TEST's goroutine
// cannot protect a panic raised inside a separately-spawned goroutine. This
// helper is that goroutine-local recovery point; the caller inspects
// `panicked` and fails the test explicitly from the main test goroutine.
func callWireDiscoveryListenerRecovered(ctx context.Context, wg *sync.WaitGroup, svtnID [16]byte, ri *discovery.RouterIngest, w io.Writer) (panicked any, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicked = r
		}
	}()
	err = wireDiscoveryListener(ctx, wg, svtnID, ri, w)
	return panicked, err
}

// TestRunRouter_DiscoveryListener_JoinsGroup_RouterModeOnly verifies AC-001:
// only the router-mode daemon joins the SVTN-scoped multicast discovery
// group (BC-2.03.001 Postcondition 1 delivery-mechanism note, Invariant 1 /
// DI-004).
//
// Postcondition 1 (router joins) is tested directly against
// wireDiscoveryListener — the Task 3 unit this story's File-Change List
// assigns to this test file — rather than through the full runRouter daemon
// lifecycle. runRouter does not call wireDiscoveryListener yet: wiring it in
// is Task 3's own Green-step action (see discovery_wire.go's stub doc
// comment, and the S-BL.DISCOVERY-WIRE Red Gate stub commit's note on why
// eager wiring at this step would panic every existing router-mode test). A
// probe sender on the loopback interface is the receive-side oracle: once
// wireDiscoveryListener really binds and joins the group, the probe
// datagram becomes observable to it (Ingest is ALSO a stub — Task 2 — but
// that is exercised end-to-end by internal/discovery/discovery_wire_test.go,
// not re-tested here).
//
// Postcondition 2 (runAccess/runConsole/runControl never join) is not
// independently probed at runtime here: this story's File-Change List does
// not touch those three functions (only cmd/switchboard/mgmt_wire.go's
// runRouter is listed as modified), so this story introduces zero
// multicast-join code path for them — verified by inspection of the
// File-Change List rather than by spinning up three additional full daemon
// lifecycles to prove the absence of an effect neither function can produce
// (neither imports internal/discovery). Flagged explicitly as a partial-scope
// decision rather than silently narrowed.
func TestRunRouter_DiscoveryListener_JoinsGroup_RouterModeOnly(t *testing.T) {
	// NOT t.Parallel(): binds a real loopback multicast UDP socket on a
	// fixed port (discovery.DiscoveryPort).

	svtnID := [16]byte{0x51, 0x51, 0x51, 0x51}
	ks := admission.NewAdmittedKeySet()
	router := routing.NewRouter(ks)
	logger := &captureLogger{} // shared test double, main_test.go
	ri := discovery.NewRouterIngest(discovery.RouterIngestConfig{Router: router, Logger: logger})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	var wg sync.WaitGroup

	type result struct {
		err      error
		panicked any
	}
	resultCh := make(chan result, 1)
	// wg.Add(1) MUST happen here, synchronously before `go`, per ARCH-01
	// §Goroutine WaitGroup Contract (F-DWIP3-001) — wireDiscoveryListener
	// itself only calls wg.Done(), matching every other wg-tracked call
	// site in this package.
	wg.Add(1)
	go func() {
		p, err := callWireDiscoveryListenerRecovered(ctx, &wg, svtnID, ri, io.Discard)
		resultCh <- result{err: err, panicked: p}
	}()

	select {
	case r := <-resultCh:
		if r.panicked != nil {
			t.Fatalf("red gate: wireDiscoveryListener stub not yet implemented: %v", r.panicked)
		}
		if r.err != nil {
			t.Fatalf("wireDiscoveryListener returned before the probe was even sent: %v", r.err)
		}
		// A real listener implementation blocks on ctx, so an immediate,
		// panic-free, error-free return here would itself be surprising —
		// but is not this test's concern to police.
	case <-time.After(500 * time.Millisecond):
		// Listener appears to be running (didn't immediately panic or
		// return) — proceed to probe it.
	}

	groupAddr := discovery.MulticastAddrFor(svtnID)
	conn, err := net.DialUDP("udp4", nil, &net.UDPAddr{IP: groupAddr, Port: discovery.DiscoveryPort})
	if err != nil {
		t.Fatalf("DialUDP probe sender: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	// Receive-side oracle (F-DWIP1-003): a bare conn.Write returning nil only
	// proves the client-side syscall succeeded, not that wireDiscoveryListener
	// ever read the datagram off the multicast socket. No key is admitted on
	// ks, so every probe here is a guaranteed HMAC lookup-miss; sending more
	// than FailureCounter's threshold (5/60s, SEC-DW-04/AC-013) of
	// >=32-byte (keySelectorMinRaw) datagrams and observing a captured log
	// line is proof the listener actually joined the group, read the bytes,
	// and fed them into RouterIngest.Ingest — genuine reception, not merely
	// an unerrored send.
	probe := make([]byte, 40)
	for i := range probe {
		probe[i] = 0xAB
	}
	const probeCount = 10
	for i := 0; i < probeCount; i++ {
		if _, err := conn.Write(probe); err != nil {
			t.Fatalf("probe Write %d/%d: %v", i+1, probeCount, err)
		}
	}

	received := false
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(logger.Lines()) > 0 {
			received = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !received {
		t.Errorf("wireDiscoveryListener: no threshold-crossing HMAC-failure log line observed after %d probe datagrams — listener did not appear to receive/ingest them (AC-001 receive-side oracle)", probeCount)
	}

	cancel()
	select {
	case r := <-resultCh:
		if r.panicked != nil {
			t.Fatalf("red gate: wireDiscoveryListener stub not yet implemented: %v", r.panicked)
		}
		if r.err != nil && ctx.Err() == nil {
			t.Errorf("wireDiscoveryListener: unexpected error: %v", r.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("wireDiscoveryListener did not return after ctx cancellation")
	}
	wg.Wait()
}
