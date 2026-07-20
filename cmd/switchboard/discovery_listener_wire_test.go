// discovery_listener_wire_test.go — tests for Task 6d:
// wire the hop-1 ingest → rate-cap → hop-2 fan-out chain into runRouter.
//
// DELIVERED (Task 6d GREEN): runRouter now constructs discovery.NewRouterIngest,
// iterates routerKS.AllSVTNEntries() at startup, and spawns a
// wireDiscoveryListener goroutine for each admitted SVTN with an onRelay
// closure wired through relayRateCap → relayDispatch. These tests verify
// that wiring.
//
// Design note: these tests drive behavior through runRouter, NOT through
// wireDiscoveryListener directly. The wireDiscoveryListener signature was
// updated to the 6-arg form (task6d-wiring-seam-ruling.md Decision 1/2)
// as part of the GREEN step; all call sites updated accordingly.
//
// Task 6d wiring-seam ruling: .factory/decisions/S-BL.DISCOVERY-WIRE-task6d-wiring-seam-ruling.md v1.0
// Fan-out resolution ruling:  .factory/decisions/S-BL.DISCOVERY-WIRE-fanout-resolution-ruling.md v1.0
// Story:                       .factory/stories/S-BL.DISCOVERY-WIRE.md
package main

import (
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/config"
	"github.com/arcavenae/switchboard/internal/discovery"
	"github.com/arcavenae/switchboard/internal/testenv"
)

// TestRunRouter_JoinsDiscoveryGroups_ForAdmittedSVTNs verifies AC-001 at the
// runRouter daemon level (Task 6d startup loop): when runRouter starts with at
// least one admitted SVTN in its AdmissionStateFile, it must join that SVTN's
// multicast group and feed incoming datagrams into RouterIngest.Ingest — proved
// by the same probe-sender + captureLogger receive-oracle pattern used by
// TestRunRouter_DiscoveryListener_JoinsGroup_RouterModeOnly.
//
// GREEN (Task 6d delivered): runRouter constructs RouterIngest, iterates
// AllSVTNEntries() at startup, and calls wireDiscoveryListener for each
// admitted SVTN — the listener joins the multicast group and feeds datagrams
// into RouterIngest.Ingest, which emits the HMAC-failure log line this test
// observes. Was the RED gate before Task 6d; now a post-wiring regression guard.
//
// NOT t.Parallel(): binds a real loopback multicast UDP socket on a fixed port
// (discovery.DiscoveryPort). Must not run concurrently with
// TestRunRouter_DiscoveryListener_JoinsGroup_RouterModeOnly.
//
// Traces to AC-001 / BC-2.03.001 Postcondition 1, Invariant 1 (DI-004).
// Task 6d wiring-seam ruling v1.0 Decision 4 (startup loop phase placement).
func TestRunRouter_JoinsDiscoveryGroups_ForAdmittedSVTNs(t *testing.T) {
	testenv.RequireMulticastLoopback(t)

	// Seed the admission state file with one admitted SVTN so routerKS has a
	// non-empty AllSVTNEntries() result at runRouter startup. makeAdmittedNode
	// (router_drain_wire_test.go) writes a JSON snapshot and sets
	// cfg.AdmissionStateFile, matching exactly what loadSnapshotFromFile
	// consumes at runRouter Phase (b1).
	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}
	nodeInfo := makeAdmittedNode(t, cfg) // sets cfg.AdmissionStateFile
	svtnID := nodeInfo.svtnID            // [16]byte{0xAB, 0, ...}

	// captureLogger is the routing.Logger injected into runRouter; it records
	// all HMAC-failure log lines emitted by RouterIngest when it processes
	// the probe datagrams. An HMAC-miss fires a FailureCounter threshold
	// crossing log once >5 misses within the 60s window (SEC-DW-04/AC-013).
	// Using a captureLogger here works because runRouter passes routerLogger
	// (constructed from the *config.Config at startup) through to RouterIngest —
	// but only AFTER Task 6d wires the RouterIngest construction. Pre-wiring,
	// no RouterIngest is created so this logger is never consulted.
	//
	// NOTE: captureLogger is wired as the router's logger inside runRouter's
	// own construction path (buildRouter(routerKS, routerLogger)); it is NOT
	// the w io.Writer passed to startRunRouterWithConfig. The buf writer does
	// not receive RouterIngest log output. We use a separate captureLogger
	// injected via the package-level routerLoggerHook test seam if available,
	// or observe output in the syncBuffer — see Red Gate discussion below.
	//
	// Historical RED note: pre-Task-6d, runRouter had no RouterIngest, so no
	// log line was emitted and the test timed out ("no log observed"). Task 6d
	// GREEN wired the RouterIngest; this observer now receives the expected line.

	var buf syncBuffer
	errCh, cancel := startRunRouterWithConfig(t, cfg, &buf)
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(3 * time.Second):
			t.Error("runRouter did not return after cancel in cleanup")
		}
	})

	// Probe sender: send enough datagrams (>threshold=5) to svtnID's multicast
	// group to trigger a FailureCounter threshold-crossing log line — identical
	// to the existing TestRunRouter_DiscoveryListener_JoinsGroup_RouterModeOnly
	// oracle. After Task 6d, runRouter's wireDiscoveryListener goroutine for
	// svtnID will receive these and feed them into RouterIngest.Ingest, which
	// will accumulate HMAC failures and emit a log.
	groupAddr := discovery.MulticastAddrFor(svtnID)
	conn, err := net.DialUDP("udp4", nil, &net.UDPAddr{IP: groupAddr, Port: discovery.DiscoveryPort})
	if err != nil {
		t.Fatalf("DialUDP probe sender: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	// 40-byte probe — above keySelectorMinRaw (32) so RouterIngest processes it
	// rather than silently dropping it as an undersized frame.
	probe := make([]byte, 40)
	for i := range probe {
		probe[i] = 0xAB
	}
	const probeCount = 10 // > FailureCounter threshold (5/60s, SEC-DW-04/AC-013)
	for i := 0; i < probeCount; i++ {
		if _, err := conn.Write(probe); err != nil {
			t.Fatalf("probe Write %d/%d: %v", i+1, probeCount, err)
		}
	}

	// Observe the buf writer for an E-ADM-017 HMAC failure rate alert line.
	// After Task 6d GREEN step, routerLogger (= newStdLogger(w)) is passed as
	// RouterIngestConfig.Logger; once the FailureCounter threshold (5 misses
	// within 60s, SEC-DW-04 / AC-013) is crossed by our 10 probe datagrams, it
	// emits "E-ADM-017 HMAC failure rate alert" via logger.Log() → routerLogger
	// → w → buf. This substring is discriminating: it cannot appear from normal
	// startup messages (which contain "switchboard router:"), proving genuine
	// ingest processing by the per-SVTN listener goroutine.
	//
	// Post-wiring: the "E-ADM-017" line appears because Task 6d GREEN constructs
	// RouterIngest and spawns a per-SVTN listener goroutine inside runRouter.
	const hmacAlertSubstr = "E-ADM-017"
	received := false
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(buf.String(), hmacAlertSubstr) {
			received = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !received {
		t.Errorf(
			"TestRunRouter_JoinsDiscoveryGroups_ForAdmittedSVTNs: "+
				"no %q log line observed after %d probe datagrams to svtnID %x multicast group %s — "+
				"runRouter did not join the discovery group for the admitted SVTN "+
				"(regression: Task 6d startup loop that joins discovery groups for admitted SVTNs is not functioning — AC-001 daemon-level join oracle)",
			hmacAlertSubstr, probeCount, svtnID, groupAddr,
		)
	}
}

// TestRunRouter_WithAdmittedSVTN_ShutsDownCleanly is a discoveryWG-teardown
// guard that verifies runRouter with at least one admitted SVTN shuts down
// within the deadline after an immediate context cancellation (no sleep before
// cancel — contrast with TestRunRouter_DiscoveryListeners_CleanShutdown which
// sleeps 100ms to let listener goroutines start before cancelling).
//
// This is NOT an end-to-end relay test. The full runtime relay chain
// (runRouter → RouterIngest → onRelay closure → relayRateCap.allow →
// relayDispatch → admitted TCP connection) is NOT exercised here. That
// end-to-end path is:
//   - Covered at unit level by TestRelayDispatch_SVTNScoped_* and
//     TestRelayDispatch_RateCap_* (discovery_relay_wire_test.go)
//   - Deferred as Forward Obligation (h) in the story
//     (.factory/stories/S-BL.DISCOVERY-WIRE.md)
//
// Historical note: this test was authored as a GREEN-guard placeholder named
// "RelayFanOut_EndToEnd" during RED-phase authoring; renamed post-delivery to
// reflect what it actually asserts (immediate-cancel shutdown teardown with an
// admitted SVTN present). The rename is a documentation fix only — test logic
// and assertions are unchanged.
//
// NOT t.Parallel(): uses admission state file on disk and a real TCP listener.
//
// Traces to ruling Decision 4 (discoveryWG.Wait() teardown);
// AC-017 / AC-018 (unit coverage via TestRelayDispatch_*).
func TestRunRouter_WithAdmittedSVTN_ShutsDownCleanly(t *testing.T) {
	// Seed admission state so AllSVTNEntries() returns at least one SVTN after
	// Task 6d GREEN step constructs ri and starts the listener loop.
	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}
	_ = makeAdmittedNode(t, cfg) // sets cfg.AdmissionStateFile

	var buf syncBuffer
	errCh, cancel := startRunRouterWithConfig(t, cfg, &buf)

	// Shutdown guard: runRouter with an admitted SVTN must start and shut down
	// cleanly. Task 6d GREEN added discoveryWG.Wait() between dataWG.Wait() and
	// writerWG.Wait() — a graceful shutdown must drain all listener goroutines
	// before returning. A deadlock in discoveryWG.Wait() (e.g., a listener
	// goroutine stuck in ReadFromUDP despite ctx cancellation + conn.Close())
	// surfaces here as a 3s timeout.
	//
	// This test cancels immediately (no sleep before cancel). The sibling
	// TestRunRouter_DiscoveryListeners_CleanShutdown sleeps 100ms first, giving
	// listener goroutines time to start before teardown — a slightly different
	// fixture. Both guards are retained to catch different timing windows.
	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("runRouter returned error on clean shutdown: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("runRouter did not return within 3s after cancel — possible discoveryWG.Wait() deadlock")
	}
}

// TestRunRouter_DiscoveryListeners_CleanShutdown verifies ruling Decision 4
// (teardown): runRouter with at least one admitted SVTN cancels its context and
// returns within a bounded deadline, exercising the discoveryWG.Wait() join
// in the shutdown block.
//
// GREEN-guard: pre-Task-6d wiring, runRouter has no discoveryWG and returns
// cleanly regardless of admitted SVTNs — this test passes trivially today.
// It becomes discriminating after Task 6d GREEN: if discoveryWG.Wait() hangs
// (e.g., a listener goroutine does not observe ctx cancellation via conn.Close),
// this test surfaces the hang.
//
// To make this more discriminating for 6d even pre-wiring: we assert that with
// an admitted SVTN in the admission state, the shutdown still completes within
// the deadline. If the deadline is exceeded, the failure points at the
// discoveryWG.Wait() call added by Task 6d — the exact discriminating signal
// the ruling requires.
//
// NOT t.Parallel(): admission state file on disk.
//
// Traces to ruling Decision 4 (discoveryWG.Wait() between dataWG.Wait() and
// writerWG.Wait() in the shutdown block).
func TestRunRouter_DiscoveryListeners_CleanShutdown(t *testing.T) {
	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}
	_ = makeAdmittedNode(t, cfg) // sets cfg.AdmissionStateFile

	var buf syncBuffer
	errCh, cancel := startRunRouterWithConfig(t, cfg, &buf)

	// Give runRouter a moment to process the admission state and, after Task 6d
	// GREEN, start the per-SVTN listener goroutine(s).
	time.Sleep(100 * time.Millisecond)

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("runRouter clean shutdown: unexpected error: %v", err)
		}
		// Shutdown completed within the deadline — discoveryWG.Wait() did not
		// hang. After Task 6d this confirms the listener goroutine(s) exited
		// promptly on ingressCancel() + conn.Close() (ruling Decision 4).
	case <-time.After(3 * time.Second):
		// After Task 6d: this means discoveryWG.Wait() did not return because
		// a listener goroutine is stuck — the ingressCancel()+conn.Close()
		// mechanism did not unblock ReadFromUDP, or wg.Done() was missed.
		// Pre-Task-6d: this branch is unreachable (no discoveryWG).
		t.Fatal("TestRunRouter_DiscoveryListeners_CleanShutdown: runRouter did not return within 3s after ctx cancel — discoveryWG.Wait() appears to be hung (GREEN-guard)")
	}
}

// TestOnRelayClosureConcurrentAccess verifies that the relayRateCap.allow()
// method is safe for concurrent callers — a mandatory requirement for Task 6d
// (ruling Decision 3 §4: "Multiple wireDiscoveryListener goroutines call the
// SAME onRelay closure concurrently. The closure captures relayRateCap — MUST
// be mutex-guarded inside its own type").
//
// This test simulates N simultaneous calls to relayRateCap.allow() from
// separate goroutines, matching the concurrency shape of the onRelay closure
// after Task 6d: one per-SVTN listener goroutine calls onRelay on every
// relay-worthy decision, and multiple SVTNs may produce decisions at the same
// time (all funnel through the same relayRateCap instance). Must pass
// go test -race.
//
// RED state: relayRateCap.allow() is already implemented with a sync.Mutex
// guard (relay_rate_cap.go), so this test PASSES today — it is a pre-condition
// verification test, not a behavioral RED gate test. It ensures the rate cap's
// concurrency contract is locked in before the wiring is added, so the full
// onRelay closure will inherit the safe allow() behaviour without needing a
// separate post-6d race test.
//
// Note: this test references relayRateCap and newRelayRateCap directly — both
// are already defined in relay_rate_cap.go and do NOT require the Task 6d
// GREEN step. The test DOES pass today. It is included in this file because
// it is mandated by the task6d-wiring-seam-ruling.md v1.0 RED checklist
// (§"Write a concurrent-onRelay test") and must be present before GREEN begins.
//
// Traces to ruling Decision 3 §4 (concurrent access obligation, binding);
// relay_rate_cap.go (sync.Mutex on allow); go test -race.
func TestOnRelayClosureConcurrentAccess(t *testing.T) {
	t.Parallel()

	const goroutines = 8
	const callsPerGoroutine = 50

	svtnID := [16]byte{0xCC, 0xCC, 0xCC, 0xCC}
	nodeAddrA := [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	nodeAddrB := [8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18}

	cap := newRelayRateCap()
	// Do NOT inject a fake clock — use real time.Now so the -race detector sees
	// the real concurrent access pattern (no artificial serialization via a
	// shared mutex on the clock).

	var wg sync.WaitGroup
	wg.Add(goroutines)
	start := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			<-start // all goroutines start simultaneously
			nodeAddr := nodeAddrA
			if i%2 == 0 {
				nodeAddr = nodeAddrB
			}
			for j := 0; j < callsPerGoroutine; j++ {
				// Simulate the onRelay closure's rate-cap check path
				// (ruling Decision 1 / Decision 3):
				//   if relayRateCap.allow(decision.SVTNID, decision.NodeAddr) {
				//       relayDispatch(router, &sendMap, decision)
				//   }
				// The test drives only allow() — relayDispatch is exercised by
				// TestRelayDispatch_* tests; this test's concern is race-free access.
				_ = cap.allow(svtnID, nodeAddr)
			}
		}()
	}

	close(start)
	wg.Wait()

	// Suppressed count is observable (ruling Decision 3 §2, SEC-DW-09
	// philosophy); assert it is reachable without a race.
	_ = cap.suppressed()

	// No assertions on the exact count — allow() is rate-limited (~1/sec per
	// key) and real-clock timing makes the allowed/dropped split non-deterministic
	// in a parallel test. The assertion is purely "no data race" (enforced by
	// go test -race) and "no panic".
}
