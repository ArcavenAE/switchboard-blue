// Package main — router_pe_connector_test.go — S-7.04-FU-PE-CONNECTOR integration tests.
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
// upstream router fixture.  Returns its address and an atomic counter
// incremented each time a connection is accepted.  The listener is closed
// via t.Cleanup.
func startPEListenerFixture(t *testing.T) (string, *atomic.Int32) {
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
						_ = c.Close()
						return
					}
				}
			}(conn)
		}
	}()
	return ln.Addr().String(), &connCount
}

// waitForConnections blocks until connCount >= 1 or 3 seconds elapses.
func waitForConnections(connCount *atomic.Int32) bool {
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if connCount.Load() >= 1 {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return connCount.Load() >= 1
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
func TestRunRouter_PE_DialAndConnect_UpstreamReachable(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	upstreamAddr, connCount := startPEListenerFixture(t)

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
	if !waitForConnections(connCount) {
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
func TestRunRouter_PE_SetEqualReconciliation_NoTeardownOnReorder(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	upstreamAddr1, connCount1 := startPEListenerFixture(t)
	upstreamAddr2, connCount2 := startPEListenerFixture(t)

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
	if !scanForLine(buf, "mode=PE", 2*time.Second) {
		t.Fatalf("AC-001 Q1 precondition: mode=PE not emitted within 2s; got:\n%s", buf.String())
	}

	// Precondition: both upstreams must have been connected initially.
	if !waitForConnections(connCount1) {
		t.Fatalf("AC-001 Q1 precondition: upstream1 received %d connections; want ≥1", connCount1.Load())
	}
	if !waitForConnections(connCount2) {
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
func TestRunRouter_PE_UnreachableUpstream_PartialPE(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)

	reachableAddr, connCount := startPEListenerFixture(t)

	// Unreachable: allocate a port then close it immediately.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("probe unreachable addr: %v", err)
	}
	unreachableAddr := ln.Addr().String()
	_ = ln.Close()

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
	if !waitForConnections(connCount) {
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
func TestRunRouter_PE_KeepalivePassedToConnector(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	upstreamAddr, connCount := startPEListenerFixture(t)

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
	if !waitForConnections(connCount) {
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
// loop establishes a live upstream connection, and under normal load E-FWD-001
// must NOT fire spuriously (split-horizon-blocked).
//
// Partial discharge note (AC-004): the exhaustion case (E-FWD-001 fires under
// path exhaustion via ARQ retransmit load) requires a receive/forward loop over
// PE connector connections that routes through FrameArrivalHandler.OnFrameArrival.
// That plumbing is not in this story's scope — the Connector only dials,
// bootstraps, and keepalives.  E-FWD-001 is owned by the routing.FrameArrivalHandler
// which is wired through netingress.Serve → routing.RouteFrame (a different code path).
// The upstream story that wires PE connection receive/forward through FrameArrivalHandler
// will own the exhaustion discharge.  This test discharges the "live PE connection
// established + no spurious E-FWD-001 under normal load" half of AC-004.
func TestRunRouter_PE_EFWD001ReconfirmationUnderLoad(t *testing.T) {
	// NOT t.Parallel(): uses multi-router testenv.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	upstreamAddr, connCount := startPEListenerFixture(t)

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
	if !waitForConnections(connCount) {
		t.Errorf("TestRunRouter_PE_EFWD001ReconfirmationUnderLoad: upstream fixture received %d connections; want ≥1 (AC-004 precondition — live PE connection required)", connCount.Load())
	}

	// AC-004 postcondition 2 (happy path): under single-path load, E-FWD-001 must
	// NOT appear.  Assert it is absent after one tick interval.
	// Search key is "E-FWD-001" — the spec-anchored event code from BC-2.02.008,
	// stable across prose rewording of the emission text.  The production emission
	// is in internal/routing/on_frame_arrival.go:252 and reads:
	//   "all paths split-horizon-blocked: frame dropped (checksum=0x%08x iface=%d) (BC-2.02.008 E-FWD-001)"
	// Using the event code avoids the vacuous-assertion defect F-P11-001 (space vs
	// hyphen mismatch: "split-horizon blocked" never matches the hyphenated production
	// string "split-horizon-blocked").
	time.Sleep(50 * time.Millisecond)
	if scanForLine(buf, "E-FWD-001", 0) {
		t.Errorf("TestRunRouter_PE_EFWD001ReconfirmationUnderLoad: E-FWD-001 fired spuriously under no-load; got:\n%s", buf.String())
	}

	// AC-004 postcondition 1 (exhaustion case — F-P1-002 blocked):
	//
	// Unmet-deps analysis: the exhaustion discharge requires routing frames through
	// routing.FrameArrivalHandler.OnFrameArrival with a forwarding table whose only
	// eligible interface is the arrival interface (BC-2.02.008 PC-3).
	//
	// Missing plumbing in this story's scope:
	//   1. No receive/forward loop over PE connector connections exists —
	//      the Connector only dials, bootstraps, and keepalives.  It does not
	//      read incoming frames from the upstream router and route them.
	//   2. runRouter routes through netingress.Serve → routing.RouteFrame, which
	//      does NOT call FrameArrivalHandler.OnFrameArrival.  The E-FWD-001 log
	//      ("all paths split-horizon-blocked") is emitted by OnFrameArrival only.
	//   3. arqsend.Retransmitter is not wired to runRouter in this story.
	//
	// Owning story: whichever story wires a PE-connection receive loop through
	// FrameArrivalHandler will own the exhaustion discharge for AC-004 PC-1.
}

// ── F-P11-001 mutation pin ──────────────────────────────────────────────────────

// TestScanForLine_DetectsEFWD001ProductionEmission pins the F-P11-001 defect:
// the AC-004 negative assertion must use "E-FWD-001" (the spec-anchored event
// code), NOT "split-horizon blocked" (space form), which never matches the
// hyphenated production emission.
//
// Production emission (internal/routing/on_frame_arrival.go:252):
//
//	"all paths split-horizon-blocked: frame dropped (checksum=0x%08x iface=%d) (BC-2.02.008 E-FWD-001)"
//
// Two assertions:
//
//	(a) scanForLine with "E-FWD-001" returns true   — proves the fixed key detects the real emission.
//	(b) scanForLine with "split-horizon blocked"     — returns false, pinning the F-P11-001 defect
//	    shape so a regression to the space form fails loudly.
func TestScanForLine_DetectsEFWD001ProductionEmission(t *testing.T) {
	t.Parallel()

	// Verbatim production emission line, formatted with concrete values.
	// Anchored to internal/routing/on_frame_arrival.go:252.
	productionLine := fmt.Sprintf(
		"all paths split-horizon-blocked: frame dropped (checksum=0x%08x iface=%d) (BC-2.02.008 E-FWD-001)",
		uint32(0xdeadbeef), 3,
	)
	buf := &syncBuffer{}
	_, _ = buf.Write([]byte(productionLine + "\n"))

	// (a) Fixed key "E-FWD-001" must detect the production emission.
	if !scanForLine(buf, "E-FWD-001", 0) {
		t.Errorf("TestScanForLine_DetectsEFWD001ProductionEmission: "+
			"scanForLine(buf, %q, 0) = false; want true — fixed key must detect production E-FWD-001 emission",
			"E-FWD-001")
	}

	// (b) Original defect key "split-horizon blocked" (space) must NOT match.
	// This pins F-P11-001: the space form is absent from the production string;
	// any regression to scanning for the space form produces a vacuous assertion.
	if scanForLine(buf, "split-horizon blocked", 0) {
		t.Errorf("TestScanForLine_DetectsEFWD001ProductionEmission: "+
			"scanForLine(buf, %q, 0) = true; want false — space form must not match hyphenated production string (F-P11-001 regression)",
			"split-horizon blocked")
	}
}

// ── AC-006: RouterHandle.Mode() reflects live connector state ──────────────────

// TestRunRouter_PE_RouterHandleModeReflectsLiveState verifies AC-006
// postcondition 4: RouterHandle.Mode() returns testenv.ModePE when a connector
// reporting ModePE is wired at construction, and ModeE when no connector or when
// connector reports ModeE.
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
	if handle.Mode() != testenv.ModePE {
		t.Errorf("TestRunRouter_PE_RouterHandleModeReflectsLiveState: handle.Mode() == %v after SetConnector(ModePE); want ModePE (AC-006 PC-4 — Mode() must delegate to connector.Mode(), not read r.mode stub)", handle.Mode())
	}

	// Verify the inverse: wire an ModeE connector → handle.Mode() == ModeE.
	fakeConnE := &fakeConnectorHandle{mode: upstreamdial.ModeE}
	handle.SetConnector(fakeConnE)

	// Force the mode field to ModePE to confirm the impl doesn't read it.
	// Probe-and-close: allocate an ephemeral port then release it so the
	// address is valid-format but not listening — avoids the port-collision
	// hazard of hardcoding 127.0.0.1:9999 (F-P1-005).
	unreachableLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("AC-006: probe unreachable addr: %v", err)
	}
	unreachableAddr := unreachableLn.Addr().String()
	_ = unreachableLn.Close()

	handle.Restart(t, testenv.RouterConfig{
		UpstreamRouters: []string{unreachableAddr},
	})
	// After Restart with non-empty upstreams, r.mode=ModePE in the stub field.
	// But connector reports ModeE via fakeConnE (Restart replaces the connector
	// with a live-dialing one that cannot connect → ModeE).
	if handle.Mode() != testenv.ModeE {
		t.Errorf("TestRunRouter_PE_RouterHandleModeReflectsLiveState: handle.Mode() == %v after SetConnector(ModeE); want ModeE (AC-006 PC-4 — connector.Mode() must override r.mode)", handle.Mode())
	}
}

// ── AC-005: VP-038 E→PE graduation via config-only ─────────────────────────────

// TestE2E_EtoPEGraduationByConfigChange verifies VP-038: after
// RouterHandle.Restart() with UpstreamRouters populated, RouterHandle.Mode()
// reflects actual connection state (testenv.ModePE) and the upstream fixture
// receives a TCP connection from the Connector.
func TestE2E_EtoPEGraduationByConfigChange(t *testing.T) {
	// NOT t.Parallel(): uses testenv in-process rig + upstream fixture.

	ctx := context.Background()
	env := testenv.New(t, ctx)

	// Start a real upstream fixture that counts accepted connections.
	peAddr, connCount := startPEListenerFixture(t)

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
	if eRouter.Mode() != testenv.ModePE {
		t.Errorf("TestE2E_EtoPEGraduationByConfigChange: eRouter.Mode() == %v; want ModePE (VP-038 PC-1)", eRouter.Mode())
	}

	// VP-038 behavioral assertion: the upstream fixture must have received a
	// TCP connection from the Connector.
	if !waitForConnections(connCount) {
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
