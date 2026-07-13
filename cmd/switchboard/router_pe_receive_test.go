// Package main — router_pe_receive_test.go — S-BL.PE-RECEIVE-LOOP integration tests.
//
// Tests: AC-001, AC-002, AC-004 (per Q9.3 harness rule: every test that asserts
// OnFrameArrival is reached MUST use the real runRouter goroutine pattern — NOT
// testenv.Restart, which bypasses SetFrameCallback).
//
// Also defines test-local peWriteFixture (Q9.2, F-SP2-002) — struct + startPEWriteFixture
// + WriteFrame — used by these tests to inject frames from the upstream fixture side.
//
// Traces: AC-001 (BC-2.09.001 PC-2/PC-3), AC-002 (BC-2.02.008 PC-3), AC-004
//
//	(BC-2.02.008 PC-3/EC-003, S404-OBS-F, S404-LOW-1).
//
// NOTE: AC-005 (flap-cycle join, goroutine lifecycle) lives in
// internal/upstreamdial/connector_test.go per F-SP3-002 ruling.
//
// RED GATE: All four tests FAIL against the stub because:
//   - ReadOuterFrame panics (not implemented).
//   - The receive goroutine does not exist.
//   - SetFrameCallback stub exists but no goroutine reads frames.
//   - OnFrameArrival is never reached → "E-FWD-001" never emitted.
package main

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/config"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/halfchannel"
	"github.com/arcavenae/switchboard/internal/outerassembler"
)

// ── test-local upstream fixture (Q9.2, F-SP2-002) ──────────────────────────────

// peWriteFixture is a test-local upstream PE fixture that listens, accepts the
// connector's dialed connection, and provides a WriteFrame method.
//
// This type is test-local to cmd/switchboard — NOT exported, NOT shared with
// connector_test.go (which uses its own in-package fixture pattern).
type peWriteFixture struct {
	addr     string
	accepted chan net.Conn // receives the accepted net.Conn when connector dials
	ln       net.Listener
	mu       sync.Mutex
	conn     net.Conn // the accepted conn, set after accepted channel fires
}

// startPEWriteFixture starts a loopback TCP listener acting as the upstream PE
// router. It accepts the first connection in a background goroutine. The listener
// is closed via t.Cleanup.
func startPEWriteFixture(t *testing.T) *peWriteFixture {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("startPEWriteFixture: Listen: %v", err)
	}

	f := &peWriteFixture{
		addr:     ln.Addr().String(),
		accepted: make(chan net.Conn, 1),
		ln:       ln,
	}

	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		conn, aErr := ln.Accept()
		if aErr != nil {
			return
		}
		f.mu.Lock()
		f.conn = conn
		f.mu.Unlock()
		f.accepted <- conn
		// Keep draining reads so the conn stays alive for keepalive probes.
		buf := make([]byte, 4096)
		for {
			if _, err := conn.Read(buf); err != nil {
				return
			}
		}
	}()

	return f
}

// WriteFrame writes the pre-assembled wire frame bytes to the accepted PE
// connection. Must be called after the accepted channel has fired (i.e. after
// the connector has dialed and the fixture has accepted the connection).
func (f *peWriteFixture) WriteFrame(t *testing.T, wire []byte) {
	t.Helper()
	f.mu.Lock()
	conn := f.conn
	f.mu.Unlock()
	if conn == nil {
		t.Fatalf("peWriteFixture.WriteFrame: no connection accepted yet")
	}
	if _, err := conn.Write(wire); err != nil {
		t.Fatalf("peWriteFixture.WriteFrame: Write: %v", err)
	}
}

// ── helper: assemble a non-bootstrap frame for integration tests ──────────────

// assemblePEFrame assembles a wire frame via outerassembler.Assemble with the
// given FrameType and a zero Envelope (HMAC bypass — PE receive path does not
// pass through RouteFrame admission check, so zero envelope reaches OnFrameArrival).
// Fails the test if assembly fails.
func assemblePEFrame(t *testing.T, ft frame.FrameType) []byte {
	t.Helper()
	cf := halfchannel.ChannelFrame{
		FrameType: ft, // halfchannel.ChannelFrame.FrameType is frame.FrameType
		ChanID:    1,
		ChanSeq:   1,
		Payload:   []byte{0x01},
	}
	wire, err := outerassembler.Assemble(cf, [outerassembler.SACKBitmapSize]byte{}, outerassembler.Envelope{})
	if err != nil {
		t.Fatalf("assemblePEFrame(FrameType=%#x): Assemble: %v", byte(ft), err)
	}
	return wire
}

// ── runRouter goroutine helper ────────────────────────────────────────────────

// startRunRouterPE starts runRouter in a goroutine with the given config.
// It returns the writer buffer (for scanForLine assertions), a cancel function,
// and an error channel. The caller's t.Cleanup cancels the context and drains
// the error channel.
func startRunRouterPE(t *testing.T, cfg *config.Config) *syncBuffer {
	t.Helper()
	cfgPath := writeTempConfig(t, cfg)
	buf := &syncBuffer{}
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- runRouter(ctx, buf, cfg, cfgPath, nil, nil)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(3 * time.Second):
		}
	})
	return buf
}

// ── AC-001: receive goroutine active; frames reach FrameArrivalHandler ────────

// TestRunRouter_PE_ReceiveLoop_ActiveAfterConnect verifies AC-001:
// after the connector dials the upstream fixture (peWriteFixture.accepted fires),
// a Data frame written to the fixture connection reaches OnFrameArrival, which
// emits "E-FWD-001" because the single-interface set always exhausts split-horizon.
//
// Establishment gate: peWriteFixture.accepted receive (TCP-accept-level, per the
// binding three-observable table in the story — F-SP7-001/F-SP7-002).
// Liveness observable: "E-FWD-001" in writer output.
//
// RED GATE: ReadOuterFrame panics (stub) → receive goroutine does not run →
// OnFrameArrival never reached → "E-FWD-001" never emitted → FAILS at RED.
func TestRunRouter_PE_ReceiveLoop_ActiveAfterConnect(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)

	fixture := startPEWriteFixture(t)

	cfg := &config.Config{
		ListenAddr:        dataAddr,
		TickInterval:      10 * time.Millisecond,
		ManagementSocket:  sockPath,
		KeepaliveInterval: 50 * time.Millisecond,
		UpstreamRouters: []config.UpstreamRouter{
			{Addr: fixture.addr},
		},
	}

	buf := startRunRouterPE(t, cfg)

	// Wait for the socket to be ready.
	if !waitForSocket(sockPath, 2*time.Second) {
		t.Fatalf("TestRunRouter_PE_ReceiveLoop_ActiveAfterConnect: management socket not ready within 2s")
	}

	// Establishment gate: peWriteFixture.accepted receive — TCP session open.
	select {
	case <-fixture.accepted:
	case <-time.After(3 * time.Second):
		t.Fatalf("TestRunRouter_PE_ReceiveLoop_ActiveAfterConnect: peWriteFixture not accepted within 3s")
	}

	// Write a Data frame from the upstream side.
	wire := assemblePEFrame(t, frame.FrameTypeData)
	fixture.WriteFrame(t, wire)

	// Liveness observable: "E-FWD-001" must appear in writer output.
	// With interfaceSet == []routing.InterfaceID{peIfaceID}, split-horizon always
	// exhausts → E-FWD-001 fires deterministically on every non-bootstrap frame.
	if !scanForLine(buf, "E-FWD-001", 3*time.Second) {
		t.Errorf("TestRunRouter_PE_ReceiveLoop_ActiveAfterConnect: \"E-FWD-001\" not found in writer output within 3s (receive goroutine not running or OnFrameArrival not wired)")
	}
}

// ── AC-002: FrameCallback wired to OnFrameArrival ─────────────────────────────

// TestRunRouter_PE_FrameCallback_WiredToOnFrameArrival verifies AC-002:
// runRouter constructs a FrameArrivalHandler and wires SetFrameCallback with a
// closure that calls arrivalHandler.OnFrameArrival. Confirmed by:
//   - A Data frame from the fixture reaching OnFrameArrival → "E-FWD-001" emitted.
//   - Import perimeter enforced separately by TestUpstreamdialImportPerimeter (internal/upstreamdial/connector_test.go, F-IP1-001).
//
// RED GATE: Same as AC-001 — receive goroutine not running → no "E-FWD-001".
func TestRunRouter_PE_FrameCallback_WiredToOnFrameArrival(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)

	fixture := startPEWriteFixture(t)

	cfg := &config.Config{
		ListenAddr:        dataAddr,
		TickInterval:      10 * time.Millisecond,
		ManagementSocket:  sockPath,
		KeepaliveInterval: 50 * time.Millisecond,
		UpstreamRouters: []config.UpstreamRouter{
			{Addr: fixture.addr},
		},
	}

	buf := startRunRouterPE(t, cfg)

	if !waitForSocket(sockPath, 2*time.Second) {
		t.Fatalf("TestRunRouter_PE_FrameCallback_WiredToOnFrameArrival: management socket not ready within 2s")
	}

	// Wait for PE connection.
	select {
	case <-fixture.accepted:
	case <-time.After(3 * time.Second):
		t.Fatalf("TestRunRouter_PE_FrameCallback_WiredToOnFrameArrival: peWriteFixture not accepted within 3s")
	}

	// Send a Data frame — must reach OnFrameArrival.
	wire := assemblePEFrame(t, frame.FrameTypeData)
	fixture.WriteFrame(t, wire)

	// "E-FWD-001" emission confirms the FrameCallback → OnFrameArrival path.
	if !scanForLine(buf, "E-FWD-001", 3*time.Second) {
		t.Errorf("TestRunRouter_PE_FrameCallback_WiredToOnFrameArrival: \"E-FWD-001\" not found within 3s (SetFrameCallback not wired to OnFrameArrival or receive goroutine not running)")
	}
}

// ── AC-004: E-FWD-001 exhaustion + S404-OBS-F / S404-LOW-1 re-confirmation ────

// TestRunRouter_PE_EFWD001ExhaustionUnderLoad verifies AC-004:
// peWriteFixture.WriteFrame writes a pre-assembled outer frame directly to the
// accepted PE connection. E-FWD-001 fires because the single-interface-set
// split-horizon topology guarantees exhaustion (interfaceSet == []routing.InterfaceID{peIfaceID}).
// Re-confirms S404-OBS-F and S404-LOW-1 via peWriteFixture injection path (Q9.4 disposition).
//
// RED GATE: ReadOuterFrame panics → FAILS at RED.
func TestRunRouter_PE_EFWD001ExhaustionUnderLoad(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)

	fixture := startPEWriteFixture(t)

	cfg := &config.Config{
		ListenAddr:        dataAddr,
		TickInterval:      10 * time.Millisecond,
		ManagementSocket:  sockPath,
		KeepaliveInterval: 50 * time.Millisecond,
		UpstreamRouters: []config.UpstreamRouter{
			{Addr: fixture.addr},
		},
	}

	buf := startRunRouterPE(t, cfg)

	if !waitForSocket(sockPath, 2*time.Second) {
		t.Fatalf("TestRunRouter_PE_EFWD001ExhaustionUnderLoad: management socket not ready within 2s")
	}

	// Precondition gate: peWriteFixture.accepted receive.
	select {
	case <-fixture.accepted:
	case <-time.After(3 * time.Second):
		t.Fatalf("TestRunRouter_PE_EFWD001ExhaustionUnderLoad: peWriteFixture not accepted within 3s")
	}

	// Assemble a Data frame via outerassembler.Assemble (Q9 frame assembly form).
	wire, err := outerassembler.Assemble(
		halfchannel.ChannelFrame{
			FrameType: frame.FrameTypeData, // halfchannel.ChannelFrame.FrameType is frame.FrameType
			ChanID:    1,
			ChanSeq:   1,
			Payload:   []byte{0x01},
		},
		[outerassembler.SACKBitmapSize]byte{},
		outerassembler.Envelope{},
	)
	if err != nil {
		t.Fatalf("TestRunRouter_PE_EFWD001ExhaustionUnderLoad: Assemble: %v", err)
	}
	fixture.WriteFrame(t, wire)

	// Postcondition 1: "E-FWD-001" must appear in writer output (BC-2.02.008 PC-3/EC-003).
	// Single-interface set → split-horizon always exhausts → deterministic emission.
	if !scanForLine(buf, "E-FWD-001", 3*time.Second) {
		t.Errorf("TestRunRouter_PE_EFWD001ExhaustionUnderLoad: \"E-FWD-001\" not found in writer output within 3s (S404-OBS-F + S404-LOW-1 re-confirmation via peWriteFixture injection path)")
	}
}

// TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader is the F-SP3-001
// byte-contract pin test + F-SP4-001 loop-continuation pin (Q9 §9.1a):
//
// Two frames with IDENTICAL payload but DIFFERING OuterHeader.SrcAddr ([8]byte
// 0x01... vs 0x02...) both produce "E-FWD-001" (≥2 emissions). This proves:
//
//	(a) Full-frame reconstruction is wired correctly: crc32.ChecksumIEEE is computed
//	    over the full frame (outer header + payload), not payload-only. Payload-only
//	    would collide on identical payloads → false-duplicate suppression → only 1
//	    emission.
//	(b) The receive loop CONTINUES after the first non-nil frameFn return
//	    (ErrAllPathsSplitHorizon): the ≥2-emission requirement pins that loop does
//	    not exit on non-nil frameFn return (F-SP4-001).
//
// RED GATE: ReadOuterFrame panics → FAILS at RED. After implementation, a
// payload-only reconstruction bug would produce only 1 emission (false-dup
// suppression), also failing the test.
func TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)

	fixture := startPEWriteFixture(t)

	cfg := &config.Config{
		ListenAddr:        dataAddr,
		TickInterval:      10 * time.Millisecond,
		ManagementSocket:  sockPath,
		KeepaliveInterval: 50 * time.Millisecond,
		UpstreamRouters: []config.UpstreamRouter{
			{Addr: fixture.addr},
		},
	}

	buf := startRunRouterPE(t, cfg)

	if !waitForSocket(sockPath, 2*time.Second) {
		t.Fatalf("TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader: management socket not ready within 2s")
	}

	// Precondition gate: peWriteFixture.accepted receive.
	select {
	case <-fixture.accepted:
	case <-time.After(3 * time.Second):
		t.Fatalf("TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader: peWriteFixture not accepted within 3s")
	}

	// Frame A: SrcAddr = [8]byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}
	// Frame B: SrcAddr = [8]byte{0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02}
	// Both have identical payload []byte{0x01}.
	// Full-frame crc32 differs (SrcAddr in outer header differs) → no false-dup suppression.
	// Payload-only crc32 would collide → second frame suppressed → only 1 E-FWD-001.

	// Build frame A with custom SrcAddr via outerassembler + direct header manipulation.
	// We assemble normally, then replace the SrcAddr bytes in the encoded wire frame.
	// Outer header layout (ARCH-02): byte[0]=version, byte[1]=frame_type,
	// bytes[2:4]=payload_len, bytes[4:20]=svtn_id, bytes[20:28]=src_addr,
	// bytes[28:36]=dst_addr, bytes[36:44]=hmac_tag.

	// Assemble a base frame to get the correct outer header + payload layout.
	baseCF := halfchannel.ChannelFrame{
		FrameType: frame.FrameTypeData, // halfchannel.ChannelFrame.FrameType is frame.FrameType
		ChanID:    1,
		ChanSeq:   1,
		Payload:   []byte{0x01},
	}
	baseWire, err := outerassembler.Assemble(baseCF, [outerassembler.SACKBitmapSize]byte{}, outerassembler.Envelope{})
	if err != nil {
		t.Fatalf("TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader: Assemble base: %v", err)
	}

	// Clone the wire and patch SrcAddr (bytes[20:28]) for frames A and B.
	wireA := make([]byte, len(baseWire))
	copy(wireA, baseWire)
	for i := 20; i < 28; i++ {
		wireA[i] = 0x01 // SrcAddr frame A
	}

	wireB := make([]byte, len(baseWire))
	copy(wireB, baseWire)
	for i := 20; i < 28; i++ {
		wireB[i] = 0x02 // SrcAddr frame B (different)
	}

	// Write both frames to the upstream fixture side.
	fixture.WriteFrame(t, wireA)
	fixture.WriteFrame(t, wireB)

	// Assert ≥2 "E-FWD-001" emissions within the budget.
	// We poll scanForLine for the first emission, then wait briefly for the second.
	if !scanForLine(buf, "E-FWD-001", 3*time.Second) {
		t.Fatalf("TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader: first \"E-FWD-001\" not found within 3s (receive loop not running or full-frame reconstruction not implemented)")
	}

	// Count emissions. The drop cache is keyed by crc32(full_frame); both frames have
	// different full-frame bytes, so neither is cached as a dup. Both produce E-FWD-001.
	// Allow up to 3s for both emissions.
	const key = "E-FWD-001"
	countEmissions := func() int {
		output := buf.String()
		count := 0
		for j := 0; j+len(key) <= len(output); j++ {
			if output[j:j+len(key)] == key {
				count++
				j += len(key) - 1
			}
		}
		return count
	}

	deadline := time.Now().Add(3 * time.Second)
	var emissions int
	for time.Now().Before(deadline) {
		emissions = countEmissions()
		if emissions >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if emissions < 2 {
		t.Errorf("TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader: got %d \"E-FWD-001\" emissions, want ≥2 (payload-only crc32 reconstruction would collide → false dup suppression → only 1 emission; full-frame reconstruction required per F-SP3-001; OR loop exits on first non-nil frameFn return, violating F-SP4-001)",
			emissions)
	}
}
