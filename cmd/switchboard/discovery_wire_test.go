// discovery_wire_test.go covers AC-001: only the router-mode daemon joins
// the SVTN-scoped multicast discovery group.
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/discovery"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/routing"
	"github.com/arcavenae/switchboard/internal/testenv"
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
func callWireDiscoveryListenerRecovered(ctx context.Context, wg *sync.WaitGroup, svtnID [16]byte, ri *discovery.RouterIngest, w io.Writer, onRelay func(discovery.RouterIngestDecision)) (panicked any, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicked = r
		}
	}()
	err = wireDiscoveryListener(ctx, wg, svtnID, ri, w, onRelay)
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
	testenv.RequireMulticastLoopback(t)

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
		p, err := callWireDiscoveryListenerRecovered(ctx, &wg, svtnID, ri, io.Discard, nil)
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

// TestWireDiscoveryListener_InvokesOnRelay_WhenRelayTrue is the ruling-mandated
// wiring-seam test (task6d-wiring-seam-ruling.md v1.0 RED checklist). It
// exercises the decision-threading branch in wireDiscoveryListener:
//
//	if decision.Relay && onRelay != nil { onRelay(decision) }
//
// (the `if decision.Relay && onRelay != nil { onRelay(decision) }` branch in wireDiscoveryListener) — the load-bearing seam that the orchestrator
// flagged as inspection-only coverage before this test existed.
//
// Test strategy: REAL datagram path (preferred over any fallback). We admit a
// real Ed25519 public key into an AdmittedKeySet, build a RouterIngest from
// the resulting router (exactly the AC-005/AC-006 pattern used by
// TestDiscovery_EncodeThenRouterIngest_AcceptsRealAdmittedNode in
// internal/discovery/discovery_wire_test.go), and use discovery.Encode()
// to produce a cryptographically valid hop-1 multicast datagram for Sequence=0
// (cold start → decision.Relay == true per AC-008). We then send it through a
// real UDP socket to the multicast group and observe the onRelay closure firing.
//
// Two assertions (discriminating by contract):
//  1. onRelay FIRES for a valid HMAC, cold-start Sequence datagram — proves the
//     `decision.Relay && onRelay != nil` branch is exercised and onRelay is
//     invoked with the decision carrying Relay==true.
//  2. onRelay does NOT fire for an HMAC-invalid datagram (wrong tag) — proves
//     that the HMAC/replay gate must pass for the branch to be entered; onRelay
//     is not invoked for reject-path datagrams.
//
// Discriminating evidence: if the `onRelay(decision)` call were removed from
// wireDiscoveryListener, assertion (1) would fail because `fired` stays false
// and no decision is captured. The test cannot pass vacuously without the
// branch being reached.
//
// NOT t.Parallel(): binds a real loopback multicast UDP socket on a fixed port
// (discovery.DiscoveryPort). Must not run concurrently with
// TestRunRouter_DiscoveryListener_JoinsGroup_RouterModeOnly.
//
// Traces to task6d-wiring-seam-ruling.md v1.0 RED checklist; AC-001/AC-008/
// AC-010; BC-2.03.001 Postcondition 1 delivery-mechanism note; Invariant 1
// (DI-004); ruling Decision 1/2 (onRelay nil semantics and threading).
func TestWireDiscoveryListener_InvokesOnRelay_WhenRelayTrue(t *testing.T) {
	testenv.RequireMulticastLoopback(t)

	// ----- Test setup -----
	// Admit a real Ed25519 public key into the AdmittedKeySet under our test
	// SVTN ID. This is the AC-005/AC-006 pattern: ks.RegisterKey(svtnID, pub,
	// admission.RoleAccess) + frame.DeriveNodeAddress(svtnID, pub) gives the
	// nodeAddr that discovery.Encode will embed in the datagram, and that
	// RouterIngest uses for key lookup.
	svtnID := [16]byte{0x77, 0x77, 0x77, 0x77}

	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate admission key: %v", err)
	}

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, pub, admission.RoleAccess)
	nodeAddr := frame.DeriveNodeAddress(svtnID, []byte(pub))

	router := routing.NewRouter(ks)
	ri := discovery.NewRouterIngest(discovery.RouterIngestConfig{Router: router})

	// ----- Relay capture oracle -----
	// The onRelay closure captures the decision in a mutex-protected struct
	// (ruling Decision 3 §4: multiple wireDiscoveryListener goroutines may call
	// onRelay concurrently; the closure must be safe for concurrent callers even
	// if this test only spawns one listener goroutine).
	var (
		mu          sync.Mutex
		fireCount   int
		gotDecision discovery.RouterIngestDecision
	)
	onRelay := func(d discovery.RouterIngestDecision) {
		mu.Lock()
		defer mu.Unlock()
		fireCount++
		gotDecision = d
	}

	// ----- Start the listener -----
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	var wg sync.WaitGroup

	type result struct {
		err      error
		panicked any
	}
	resultCh := make(chan result, 1)
	wg.Add(1)
	go func() {
		p, listenErr := callWireDiscoveryListenerRecovered(ctx, &wg, svtnID, ri, io.Discard, onRelay)
		resultCh <- result{err: listenErr, panicked: p}
	}()

	// Wait up to 500ms for the listener to bind and join the group.
	select {
	case r := <-resultCh:
		if r.panicked != nil {
			t.Fatalf("red gate: wireDiscoveryListener stub not yet implemented: %v", r.panicked)
		}
		// An immediate return (no panic, no error) before we even sent anything
		// is unexpected for a real listener; surface it as a test setup failure.
		t.Fatalf("wireDiscoveryListener returned before probe was sent: %v", r.err)
	case <-time.After(500 * time.Millisecond):
		// Listener is running.
	}

	// ----- Assertion 1: onRelay fires for a valid HMAC-authenticated datagram -----
	// discovery.Encode produces a cryptographically valid hop-1 datagram using
	// the admitted public key as the nodeAdmissionPubkey argument. Sequence=0
	// is a cold-start value: RouterIngest.Ingest accepts it unconditionally
	// (AC-008: no prior lastSeen entry → cold start → decision.Relay == true).
	validRaw, err := discovery.Encode(discovery.AdvertisementPayload{
		NodeAddr: nodeAddr,
		SVTNID:   svtnID,
		Sequence: 0,
		Sessions: []discovery.SessionPresence{
			{SessionName: "relay-test-session", Status: discovery.Attached, Quality: discovery.QualityGreen},
		},
	}, []byte(pub))
	if err != nil {
		t.Fatalf("Encode valid advertisement: %v", err)
	}

	groupAddr := discovery.MulticastAddrFor(svtnID)
	sendConn, err := net.DialUDP("udp4", nil, &net.UDPAddr{IP: groupAddr, Port: discovery.DiscoveryPort})
	if err != nil {
		t.Fatalf("DialUDP for valid probe: %v", err)
	}
	t.Cleanup(func() { _ = sendConn.Close() })

	if _, err := sendConn.Write(validRaw); err != nil {
		t.Fatalf("Write valid datagram: %v", err)
	}

	// Poll for the onRelay closure to fire (up to 2s).
	relayFired := false
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		count := fireCount
		mu.Unlock()
		if count > 0 {
			relayFired = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !relayFired {
		t.Errorf(
			"TestWireDiscoveryListener_InvokesOnRelay_WhenRelayTrue: " +
				"onRelay was NOT invoked after sending an HMAC-valid cold-start datagram — " +
				"the decision.Relay && onRelay != nil branch was not reached " +
				"(ruling RED checklist; the decision.Relay && onRelay != nil branch in wireDiscoveryListener)",
		)
	} else {
		mu.Lock()
		d := gotDecision
		mu.Unlock()

		if !d.Relay {
			t.Errorf("onRelay received decision with Relay == false, want true (cold-start AC-008)")
		}
		if d.SVTNID != svtnID {
			t.Errorf("onRelay decision.SVTNID = %x, want %x", d.SVTNID, svtnID)
		}
		if d.NodeAddr != nodeAddr {
			t.Errorf("onRelay decision.NodeAddr = %x, want %x", d.NodeAddr, nodeAddr)
		}
	}

	// ----- Assertion 2: onRelay does NOT fire for an HMAC-invalid datagram -----
	// Record current fireCount before sending the invalid datagram.
	mu.Lock()
	countBefore := fireCount
	mu.Unlock()

	// Build a datagram with a deliberately-wrong HMAC tag (all zeroes, which
	// will not verify against the admitted key). RouterIngest.Ingest will
	// return ErrInvalidHMACTag → decision.Relay = false → onRelay not called.
	// Wire layout: [8]byte HMAC tag | body (svtnID + nodeAddr + Sequence +
	// session list). Pad to 42 bytes (minimum valid-frame size) to pass the
	// keySelectorMinRaw guard and exercise the HMAC-fail branch, not the
	// short-datagram branch.
	invalidRaw := make([]byte, 42)       // 8 tag + 16 svtnID + 8 nodeAddr + 8 seq + 2 count
	copy(invalidRaw[8:24], svtnID[:])    // SVTNID at body[0:16]
	copy(invalidRaw[24:32], nodeAddr[:]) // NodeAddr at body[16:24]
	// HMAC tag stays all-zero → will not verify

	if _, err := sendConn.Write(invalidRaw); err != nil {
		t.Fatalf("Write invalid datagram: %v", err)
	}

	// Give the listener time to process the invalid datagram (if it were going
	// to call onRelay, it would within ~100ms of the send landing on the socket).
	// Also send several copies so the listener goroutine has time to process at
	// least one before the deadline elapses.
	for i := 0; i < 3; i++ {
		_, _ = sendConn.Write(invalidRaw)
	}
	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	countAfter := fireCount
	mu.Unlock()

	if countAfter != countBefore {
		t.Errorf(
			"TestWireDiscoveryListener_InvokesOnRelay_WhenRelayTrue: "+
				"onRelay was invoked %d additional time(s) after HMAC-invalid datagram(s) — "+
				"want 0 additional invocations; the HMAC/replay gate must block the callback",
			countAfter-countBefore,
		)
	}

	// ----- Teardown -----
	cancel()
	select {
	case r := <-resultCh:
		if r.panicked != nil {
			t.Fatalf("wireDiscoveryListener panicked on shutdown: %v", r.panicked)
		}
		if r.err != nil && ctx.Err() == nil {
			t.Errorf("wireDiscoveryListener: unexpected error on shutdown: %v", r.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("wireDiscoveryListener did not return after ctx cancellation")
	}
	wg.Wait()
}
