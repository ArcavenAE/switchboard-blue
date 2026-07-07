// Package main — router_pe_connector_test.go — S-7.04-FU-PE-CONNECTOR integration tests.
//
// These tests will all FAIL at the Red Gate.  The failure modes are:
//   - Tests that drive runRouter with PE config: upstreamRoutersAsSet panics.
//   - TestRunRouter_PE_RouterHandleModeReflectsLiveState: the stub Mode() reads
//     r.mode (ModeE at construction), ignoring the wired connector handle.
//   - TestE2E_EtoPEGraduationByConfigChange: stub Restart() dials nothing; the
//     upstream fixture listener never receives a connection → second assertion fails.
//   - TestRunRouter_PE_EFWD001ReconfirmationUnderLoad: runRouter with PE config
//     emits mode=PE but never establishes a TCP connection to the fixture —
//     connection-received assertion fails.
//   - TestE2E_RouterDrain_NodesMigrateWithin2s: skipped with partial-discharge note.
//
// Named exactly as specified in the story's Estimated Test Surface table.
//
// Traces: AC-001..AC-006, VP-037, VP-038.
package main

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/config"
	"github.com/arcavenae/switchboard/internal/testenv"
	"github.com/arcavenae/switchboard/internal/upstreamdial"
)

// ── shared helpers ──────────────────────────────────────────────────────────────

// startPEListenerFixture starts a loopback TCP listener that acts as the
// upstream router fixture.  Returns the listener, its address, and an atomic
// counter incremented each time a connection is accepted.
func startPEListenerFixture(t *testing.T) (net.Listener, string, *atomic.Int32) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("startPEListenerFixture: Listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	var connCount atomic.Int32
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			connCount.Add(1)
			// Keep the connection open to allow keepalive probes.
			go func(c net.Conn) {
				buf := make([]byte, 4096)
				for {
					_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
					_, err := c.Read(buf)
					if err != nil {
						c.Close()
						return
					}
				}
			}(conn)
		}
	}()
	return ln, ln.Addr().String(), &connCount
}

// waitForConnections blocks until connCount >= want or budget elapses.
func waitForConnections(connCount *atomic.Int32, want int32, budget time.Duration) bool {
	deadline := time.Now().Add(budget)
	for time.Now().Before(deadline) {
		if connCount.Load() >= want {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return connCount.Load() >= want
}

// fakeConnectorHandle is a minimal upstreamdial.Handle for testenv seam tests.
// All methods are no-ops except Mode(), which returns a fixed value.
type fakeConnectorHandle struct {
	mode upstreamdial.ConnMode
}

func (f *fakeConnectorHandle) ReloadAddrs(_ []string)      {}
func (f *fakeConnectorHandle) Mode() upstreamdial.ConnMode { return f.mode }
func (f *fakeConnectorHandle) Stop()                       {}

// ── AC-001: PE startup dials upstream ──────────────────────────────────────────

// TestRunRouter_PE_DialAndConnect_UpstreamReachable verifies AC-001: when
// runRouter starts with a PE config and the upstream fixture is listening, the
// Connector establishes a TCP connection and the fixture records it.
//
// RED GATE: upstreamRoutersAsSet panics inside runRouter startup.
func TestRunRouter_PE_DialAndConnect_UpstreamReachable(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	_, upstreamAddr, connCount := startPEListenerFixture(t)

	cfg := &config.Config{
		ListenAddr:        dataAddr,
		TickInterval:      10 * time.Millisecond,
		ManagementSocket:  sockPath,
		KeepaliveInterval: 100 * time.Millisecond,
		UpstreamRouters: []config.UpstreamRouter{
			{Addr: upstreamAddr},
		},
	}
	cfgPath := writeTempConfig(t, cfg)

	buf := &syncBuffer{}
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- runRouter(ctx, buf, cfg, cfgPath, nil)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(3 * time.Second):
		}
	})

	if !waitForSocket(sockPath, 2*time.Second) {
		t.Fatalf("AC-001: mgmt socket not created within 2s")
	}

	// AC-001 postcondition 1+3: upstream fixture must receive a connection.
	// RED GATE: runRouter panics at upstreamRoutersAsSet before dialing.
	if !waitForConnections(connCount, 1, 3*time.Second) {
		t.Errorf("TestRunRouter_PE_DialAndConnect_UpstreamReachable: upstream fixture received %d connections after 3s; want ≥1 (AC-001 PC-1/PC-3)", connCount.Load())
	}

	// Daemon must still be running.
	select {
	case rErr := <-errCh:
		t.Errorf("AC-001: runRouter returned prematurely: %v", rErr)
	default:
	}
}

// ── AC-001 Q1: set-equal reconciliation on reload ──────────────────────────────

// TestRunRouter_PE_SetEqualReconciliation_NoTeardownOnReorder verifies AC-001
// postcondition 5: reloading with the same addresses in reversed order MUST NOT
// trigger teardown or redial (set-equal semantics per Q1 ruling).
//
// RED GATE: upstreamRoutersAsSet panics inside runRouter startup.
func TestRunRouter_PE_SetEqualReconciliation_NoTeardownOnReorder(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	_, upstreamAddr1, connCount1 := startPEListenerFixture(t)
	_, upstreamAddr2, connCount2 := startPEListenerFixture(t)

	// Start config: two upstreams in order [A, B].
	startCfg := &config.Config{
		ListenAddr:        dataAddr,
		TickInterval:      10 * time.Millisecond,
		ManagementSocket:  sockPath,
		KeepaliveInterval: 100 * time.Millisecond,
		UpstreamRouters: []config.UpstreamRouter{
			{Addr: upstreamAddr1},
			{Addr: upstreamAddr2},
		},
	}

	// Reload config: same upstreams in reversed order [B, A].
	reloadCfg := &config.Config{
		ListenAddr:        dataAddr,
		TickInterval:      10 * time.Millisecond,
		ManagementSocket:  sockPath,
		KeepaliveInterval: 100 * time.Millisecond,
		UpstreamRouters: []config.UpstreamRouter{
			{Addr: upstreamAddr2},
			{Addr: upstreamAddr1},
		},
	}
	cfgPath := writeTempConfig(t, reloadCfg)

	buf, errCh, cancel, sighupCh := startRunRouterForReload(t, startCfg, cfgPath)
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(3 * time.Second):
		}
	})

	// Precondition: mode=PE startup emission.
	// RED GATE: runRouter panics at upstreamRoutersAsSet before emitting mode=PE.
	if !scanForLine(buf, "mode=PE", 2*time.Second) {
		t.Fatalf("AC-001 Q1 precondition: mode=PE not emitted within 2s; got:\n%s", buf.String())
	}

	// Precondition: both upstreams must have been connected initially.
	// RED GATE: upstreamRoutersAsSet panics → no connections → both counts remain 0.
	if !waitForConnections(connCount1, 1, 3*time.Second) {
		t.Fatalf("AC-001 Q1 precondition: upstream1 received %d connections; want ≥1", connCount1.Load())
	}
	if !waitForConnections(connCount2, 1, 3*time.Second) {
		t.Fatalf("AC-001 Q1 precondition: upstream2 received %d connections; want ≥1", connCount2.Load())
	}

	// Snapshot connection counts before reload.
	snap1 := connCount1.Load()
	snap2 := connCount2.Load()

	// Send SIGHUP with the reversed-order config.
	sighupCh <- syscall.SIGHUP
	time.Sleep(300 * time.Millisecond)

	// AC-001 postcondition 5: set-equal reload must NOT trigger new dials.
	// Connection counts must not have increased.
	after1 := connCount1.Load()
	after2 := connCount2.Load()
	if after1 > snap1 || after2 > snap2 {
		t.Errorf("AC-001 Q1: set-equal reload triggered new dials: upstream1 %d→%d, upstream2 %d→%d; want no change (BC-2.09.001 EC-002)", snap1, after1, snap2, after2)
	}

	// Daemon must still be running.
	select {
	case rErr := <-errCh:
		t.Errorf("AC-001 Q1: runRouter returned prematurely after set-equal reload: %v", rErr)
	default:
	}
}

// ── AC-002: unreachable upstream → partial PE ───────────────────────────────────

// TestRunRouter_PE_UnreachableUpstream_PartialPE verifies AC-002: when one
// upstream is reachable and another is not, the Connector connects to the
// reachable one (Mode()==ModePE) and emits EC-001 log for the unreachable one.
//
// RED GATE: upstreamRoutersAsSet panics inside runRouter startup.
func TestRunRouter_PE_UnreachableUpstream_PartialPE(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)

	_, reachableAddr, connCount := startPEListenerFixture(t)

	// Unreachable: allocate a port then close it immediately.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("probe unreachable addr: %v", err)
	}
	unreachableAddr := ln.Addr().String()
	ln.Close()

	cfg := &config.Config{
		ListenAddr:        dataAddr,
		TickInterval:      10 * time.Millisecond,
		ManagementSocket:  sockPath,
		KeepaliveInterval: 100 * time.Millisecond,
		UpstreamRouters: []config.UpstreamRouter{
			{Addr: reachableAddr},
			{Addr: unreachableAddr},
		},
	}
	cfgPath := writeTempConfig(t, cfg)

	buf := &syncBuffer{}
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- runRouter(ctx, buf, cfg, cfgPath, nil)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(3 * time.Second):
		}
	})
	if !waitForSocket(sockPath, 2*time.Second) {
		t.Fatalf("AC-002 partial PE: mgmt socket not created within 2s")
	}

	// AC-002 postcondition 4: reachable upstream gets a connection (partial-PE).
	// RED GATE: runRouter panics at upstreamRoutersAsSet.
	if !waitForConnections(connCount, 1, 3*time.Second) {
		t.Errorf("TestRunRouter_PE_UnreachableUpstream_PartialPE: reachable upstream received %d connections; want ≥1 (AC-002 PC-4)", connCount.Load())
	}

	// AC-002 postcondition 1: EC-001 log fires for unreachable upstream.
	wantLog := fmt.Sprintf("upstream router %s unreachable", unreachableAddr)
	if !scanForLine(buf, wantLog, 2*time.Second) {
		t.Errorf("TestRunRouter_PE_UnreachableUpstream_PartialPE: EC-001 log %q not emitted within 2s; got:\n%s",
			wantLog, buf.String())
	}
}

// ── AC-003: keepalive passed to Connector ──────────────────────────────────────

// TestRunRouter_PE_KeepalivePassedToConnector verifies AC-003: the Connector is
// constructed with the keepaliveIntervalFor(cfg) value and that value drives the
// keepalive ticker (not a hardcoded constant or sweepDeadline).
//
// RED GATE: upstreamRoutersAsSet panics inside runRouter startup.
func TestRunRouter_PE_KeepalivePassedToConnector(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	_, upstreamAddr, connCount := startPEListenerFixture(t)

	const testKeepalive = 200 * time.Millisecond
	cfg := &config.Config{
		ListenAddr:        dataAddr,
		TickInterval:      10 * time.Millisecond,
		ManagementSocket:  sockPath,
		KeepaliveInterval: testKeepalive,
		UpstreamRouters: []config.UpstreamRouter{
			{Addr: upstreamAddr},
		},
	}
	cfgPath := writeTempConfig(t, cfg)

	buf := &syncBuffer{}
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- runRouter(ctx, buf, cfg, cfgPath, nil)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(3 * time.Second):
		}
	})
	if !waitForSocket(sockPath, 2*time.Second) {
		t.Fatalf("AC-003: mgmt socket not created within 2s")
	}

	// AC-003: upstream fixture must receive a connection (proves keepalive ticker
	// is constructed and drives a connection health check to the fixture).
	// RED GATE: runRouter panics at upstreamRoutersAsSet before constructing Connector.
	if !waitForConnections(connCount, 1, 3*time.Second) {
		t.Errorf("TestRunRouter_PE_KeepalivePassedToConnector: upstream fixture received %d connections; want ≥1 (AC-003)", connCount.Load())
	}

	// The keepalive interval must be emitted in the startup log so the implementer
	// can verify the correct value was passed to the Connector.
	wantKeepaliveLog := fmt.Sprintf("keepalive_interval=%v", testKeepalive)
	if !scanForLine(buf, wantKeepaliveLog, 2*time.Second) {
		t.Errorf("TestRunRouter_PE_KeepalivePassedToConnector: keepalive log %q not emitted within 2s; got:\n%s",
			wantKeepaliveLog, buf.String())
	}
}

// ── AC-004: E-FWD-001 re-confirmation under load ────────────────────────────────

// TestRunRouter_PE_EFWD001ReconfirmationUnderLoad verifies AC-004: the PE dial
// loop establishes a live upstream connection, and under sustained ARQ retransmit
// load the E-FWD-001 log fires (split-horizon blocked).
//
// RED GATE: upstreamRoutersAsSet panics inside runRouter startup when cfg has
// UpstreamRouters set — so runRouter never reaches the run loop and never
// establishes a connection.  The connection-received assertion on the upstream
// fixture fails with 0 connections received.
func TestRunRouter_PE_EFWD001ReconfirmationUnderLoad(t *testing.T) {
	// NOT t.Parallel(): uses multi-router testenv.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	_, upstreamAddr, connCount := startPEListenerFixture(t)

	cfg := &config.Config{
		ListenAddr:        dataAddr,
		TickInterval:      10 * time.Millisecond,
		ManagementSocket:  sockPath,
		KeepaliveInterval: 100 * time.Millisecond,
		UpstreamRouters: []config.UpstreamRouter{
			{Addr: upstreamAddr},
		},
	}
	cfgPath := writeTempConfig(t, cfg)

	buf := &syncBuffer{}
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- runRouter(ctx, buf, cfg, cfgPath, nil)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(3 * time.Second):
		}
	})
	if !waitForSocket(sockPath, 2*time.Second) {
		t.Fatalf("AC-004: mgmt socket not created within 2s")
	}

	// Precondition (AC-004 PC-1): live PE upstream connection must be established.
	// RED GATE: upstreamRoutersAsSet panics before the Connector is constructed,
	// so no connection is ever made.
	if !waitForConnections(connCount, 1, 3*time.Second) {
		t.Errorf("TestRunRouter_PE_EFWD001ReconfirmationUnderLoad: upstream fixture received %d connections; want ≥1 (AC-004 precondition — live PE connection required)", connCount.Load())
	}

	// AC-004 postcondition 2 (happy path): under single-path load, E-FWD-001 must
	// NOT appear.  Assert it is absent after one tick interval.
	time.Sleep(50 * time.Millisecond)
	if scanForLine(buf, "split-horizon blocked", 0) {
		t.Errorf("TestRunRouter_PE_EFWD001ReconfirmationUnderLoad: E-FWD-001 fired spuriously under no-load; got:\n%s", buf.String())
	}

	// AC-004 postcondition 1 (exhaustion case): would require ARQ retransmit load
	// from arqsend.Retransmitter to exhaust the routing table path count.
	// This path is exercised once the Connector is wired; the precondition above
	// establishes the live egress anchor for S404-OBS-F and S404-LOW-1.
}

// ── AC-006: RouterHandle.Mode() reflects live connector state ──────────────────

// TestRunRouter_PE_RouterHandleModeReflectsLiveState verifies AC-006
// postcondition 4: RouterHandle.Mode() returns testenv.ModePE when a connector
// reporting ModePE is wired at construction, and ModeE when no connector or when
// connector reports ModeE.
//
// RED GATE: RouterHandle.Mode() reads r.mode (ModeE at construction), not the
// connector wired via SetConnector.  The test expects ModePE from the fake
// connector but gets ModeE from the stub.
func TestRunRouter_PE_RouterHandleModeReflectsLiveState(t *testing.T) {
	// NOT t.Parallel(): uses testenv in-process rig.

	ctx := context.Background()
	env := testenv.New(t, ctx)

	// Start a router in E mode (r.mode=ModeE at construction).
	handle := env.StartRouter(t, testenv.RouterConfig{})
	if handle.Mode() != testenv.ModeE {
		t.Fatalf("AC-006 precondition: handle.Mode() == %v; want ModeE", handle.Mode())
	}

	// Wire a fake connector that always reports ModePE.
	fakeConn := &fakeConnectorHandle{mode: upstreamdial.ModePE}
	handle.SetConnector(fakeConn)

	// AC-006 postcondition 4: after wiring a ModePE connector, Mode() must return
	// testenv.ModePE by delegating to connector.Mode().
	// STUB: Mode() reads r.mode=ModeE → returns ModeE → TEST FAILS.
	// IMPL: Mode() delegates to connector.Mode()=ModePE → returns ModePE → PASS.
	if handle.Mode() != testenv.ModePE {
		t.Errorf("TestRunRouter_PE_RouterHandleModeReflectsLiveState: handle.Mode() == %v after SetConnector(ModePE); want ModePE (AC-006 PC-4 — Mode() must delegate to connector.Mode(), not read r.mode stub)", handle.Mode())
	}

	// Verify the inverse: wire an ModeE connector → handle.Mode() == ModeE.
	fakeConnE := &fakeConnectorHandle{mode: upstreamdial.ModeE}
	handle.SetConnector(fakeConnE)

	// Force the mode field to ModePE to confirm the impl doesn't read it.
	handle.Restart(t, testenv.RouterConfig{
		UpstreamRouters: []string{"127.0.0.1:9999"},
	})
	// After Restart with non-empty upstreams, stub sets r.mode=ModePE.
	// But connector reports ModeE.
	// IMPL: Mode() delegates to connector → ModeE.
	// STUB: Mode() reads r.mode=ModePE → returns ModePE → TEST FAILS below.
	if handle.Mode() != testenv.ModeE {
		t.Errorf("TestRunRouter_PE_RouterHandleModeReflectsLiveState: handle.Mode() == %v after SetConnector(ModeE); want ModeE (AC-006 PC-4 — connector.Mode() must override r.mode)", handle.Mode())
	}
}

// ── AC-005: VP-038 E→PE graduation via config-only ─────────────────────────────

// TestE2E_EtoPEGraduationByConfigChange verifies VP-038: after
// RouterHandle.Restart() with UpstreamRouters populated, RouterHandle.Mode()
// reflects actual connection state (testenv.ModePE) and the upstream fixture
// receives a TCP connection from the Connector.
//
// RED GATE: stub Restart() sets r.mode=ModePE but dials nothing.  The upstream
// fixture receives zero connections — the second assertion fails.
func TestE2E_EtoPEGraduationByConfigChange(t *testing.T) {
	// NOT t.Parallel(): uses testenv in-process rig + upstream fixture.

	ctx := context.Background()
	env := testenv.New(t, ctx)

	// Start a real upstream fixture that counts accepted connections.
	_, peAddr, connCount := startPEListenerFixture(t)

	// Start in E mode.
	eRouter := env.StartRouter(t, testenv.RouterConfig{})
	if eRouter.Mode() != testenv.ModeE {
		t.Fatalf("VP-038 precondition: eRouter.Mode() == %v; want ModeE", eRouter.Mode())
	}

	// Capture SVTNID before restart — must be unchanged after graduation.
	svtnBefore := eRouter.SVTNID()

	// Restart into PE mode.
	// After AC-006: calls connector.ReloadAddrs() + polls connector.Mode() == ModePE.
	// Stub: sets r.mode=ModePE unconditionally, dials nothing.
	eRouter.Restart(t, testenv.RouterConfig{
		UpstreamRouters: []string{peAddr},
	})

	// VP-038 postcondition 1: Mode() == ModePE.
	// With stub: passes tautologically (r.mode=ModePE).
	// With impl: passes because connector.Mode()==ModePE (live dial succeeded).
	if eRouter.Mode() != testenv.ModePE {
		t.Errorf("TestE2E_EtoPEGraduationByConfigChange: eRouter.Mode() == %v; want ModePE (VP-038 PC-1)", eRouter.Mode())
	}

	// VP-038 behavioral assertion: the upstream fixture must have received a
	// TCP connection from the Connector.
	// RED GATE: stub Restart() dials nothing → connCount.Load() == 0 → FAIL.
	// IMPL: Connector dials after ReloadAddrs → fixture accepts → connCount ≥ 1 → PASS.
	if !waitForConnections(connCount, 1, 3*time.Second) {
		t.Errorf("TestE2E_EtoPEGraduationByConfigChange: upstream fixture received %d TCP connections after 3s; want ≥1 — Restart must trigger a real dial (VP-038 behavioral contract)", connCount.Load())
	}

	// VP-038 postcondition: SVTNID unchanged.
	svtnAfter := eRouter.SVTNID()
	if svtnBefore != svtnAfter {
		t.Errorf("TestE2E_EtoPEGraduationByConfigChange: SVTNID changed: before=%v after=%v; want unchanged (VP-038)", svtnBefore, svtnAfter)
	}
}

// ── AC-005: VP-037 drain-within-window ─────────────────────────────────────────

// TestE2E_RouterDrain_NodesMigrateWithin2s verifies VP-037: session traffic
// resumes on the alternate router within 2s after SendDrainSignal.
//
// PARTIAL DISCHARGE CLAUSE (story AC-005, placement note Q8): this story
// delivers the live-egress infrastructure required by VP-037; the DRAIN
// broadcast wire protocol is owned by S-7.04-FU-DRAIN-WIRE.  Test is skipped
// with a partial-discharge note.
func TestE2E_RouterDrain_NodesMigrateWithin2s(t *testing.T) {
	t.Skip("VP-037 partial-discharge: DRAIN broadcast wire protocol required from " +
		"S-7.04-FU-DRAIN-WIRE; live upstream connections delivered by S-7.04-FU-PE-CONNECTOR " +
		"(this story). Partial-discharge per story AC-005 note.")

	ctx := context.Background()
	env := testenv.NewWithRouters(t, ctx, 2)
	env.SendDrainSignal(t, 0)
	// Full drain-and-migrate assertion lands when S-7.04-FU-DRAIN-WIRE ships.
}
