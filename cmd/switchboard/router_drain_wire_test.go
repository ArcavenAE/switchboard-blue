// Package main — router_drain_wire_test.go — S-7.04-FU-DRAIN-WIRE integration
// and unit tests: DRAIN-over-SVTN wire propagation, the netingress OnAccept
// seam, and the ctl receive-path guard.
//
// Traces: AC-001 (BC-2.09.002 PC-1 / Q1 / Q-SEAM / Q-SINGLE-OBS), AC-002
// (Q-SEAM ownership split / Q-AC002), AC-003 (Q-SINGLE-OBS / Q-AC003 /
// Q-CTL-GUARD / BC-2.01.008 PC-4 + Invariant 2), AC-004 (VP-037 stage-1 /
// Q4-AMENDED).
//
// RED GATE: every test in this file that depends on runRouter's OnAccept
// closure, the per-node send map, the writer goroutine, or the single
// startup drain observer FAILS against the tree at this commit — none of
// that behavior is wired yet (S-7.04-FU-DRAIN-WIRE step (a) landed only the
// netingress-side ServeConfig/NodeHandle seam and compilable cmd/switchboard
// scaffolding; step (c) wires the behavior).
// TestRouter_CtlFrame_UnknownControlType_SilentIgnore is a known exception —
// see its own doc comment for why it is expected to pass pre-implementation.
package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/config"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/routing"
)

// ── shared harness helpers (Q-AC002) ────────────────────────────────────────

// nodeConnRecord captures one nodeConnHook invocation for test assertions.
type nodeConnRecord struct {
	event   nodeConnEvent
	ifaceID routing.InterfaceID
}

// setNodeConnHook installs a channel-backed nodeConnHook for the duration of
// the calling test and returns the channel. nodeConnHook is a package-level
// mutable var (Q-AC002) — callers MUST NOT call t.Parallel() (F-DW-SP4-005
// test-isolation requirement).
func setNodeConnHook(t *testing.T) chan nodeConnRecord {
	t.Helper()
	events := make(chan nodeConnRecord, 8)
	nodeConnHook = func(event nodeConnEvent, ifaceID routing.InterfaceID) {
		events <- nodeConnRecord{event: event, ifaceID: ifaceID}
	}
	t.Cleanup(func() { nodeConnHook = nil })
	return events
}

// awaitNodeConnEvent blocks until events yields a record for the given
// nodeConnEvent, or fails the test after budget elapses.
func awaitNodeConnEvent(t *testing.T, events chan nodeConnRecord, want nodeConnEvent, budget time.Duration) nodeConnRecord { //nolint:unparam // budget is a caller-controlled knob; all current callers use 2s but the parameter is intentional (mirrors scanForLine's budget in router_sighup_test.go)
	t.Helper()
	select {
	case rec := <-events:
		if rec.event != want {
			t.Fatalf("nodeConnHook event = %v, want %v", rec.event, want)
		}
		return rec
	case <-time.After(budget):
		t.Fatalf("nodeConnHook did not observe event %v within %v", want, budget)
		return nodeConnRecord{}
	}
}

// dialNode dials an inbound TCP connection to cfg.ListenAddr, simulating a
// connected node. The connection is closed via t.Cleanup.
func dialNode(t *testing.T, cfg *config.Config) net.Conn {
	t.Helper()
	conn, err := net.Dial("tcp", cfg.ListenAddr)
	if err != nil {
		t.Fatalf("dialNode: dial %s: %v", cfg.ListenAddr, err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// dialNodeAndAwaitRegistration is the shared AC-001/AC-004 harness helper
// (Q-AC002/Q3-AMENDED discharge-trace step 3): starts runRouter, dials a
// simulated node, and blocks until nodeConnHook observes nodeConnRegistered
// for that connection — the mandatory accept/register barrier both ACs
// require before triggering drain (skipping it risks the observer's
// sendMap.Range seeing zero entries at Signal time).
//
// Returns the dialed conn, a non-blocking cancel (the caller calls this to
// trigger drainCoord.Signal via the shutdown block), and awaitReturn — an
// idempotent, Once-guarded blocking wait for runRouter's return value, safe
// to call both explicitly in the test body (for synchronization) and via
// the registered t.Cleanup safety net (leak protection if an earlier step
// fails a Fatal before the body reaches it). The registered IfaceID itself
// is AC-002's observable (see TestNetingress_OnAccept_RegistersNodeHandle)
// — AC-001/AC-004 callers don't need it, only the barrier it confirms.
func dialNodeAndAwaitRegistration(t *testing.T, cfg *config.Config, buf *syncBuffer) (
	conn net.Conn, cancel context.CancelFunc, awaitReturn func() error,
) {
	t.Helper()
	events := setNodeConnHook(t)
	errCh, cancelFn := startRunRouterWithConfig(t, cfg, buf)

	var once sync.Once
	var waitErr error
	awaitReturn = func() error {
		once.Do(func() {
			select {
			case waitErr = <-errCh:
			case <-time.After(2 * time.Second):
				waitErr = fmt.Errorf("runRouter did not return within 2s after ctx cancel")
			}
		})
		return waitErr
	}
	t.Cleanup(func() {
		cancelFn()
		_ = awaitReturn()
	})

	conn = dialNode(t, cfg)
	awaitNodeConnEvent(t, events, nodeConnRegistered, 2*time.Second)
	return conn, cancelFn, awaitReturn
}

// ── NODE_IDENTIFY handshake helpers (S-BL.NODE-IDENTIFY-WIRE Task 20) ────────
//
// After S-BL.NODE-IDENTIFY-WIRE lands, runRouter's onAccept closure runs the
// NODE_IDENTIFY handshake BEFORE calling sendMap.Store. Raw TCP connections
// that don't complete the handshake are rejected (conn closed within 10s).
// These helpers wire the node side of the handshake so the drain/ctl tests can
// establish admitted connections.

// admittedNodeInfo carries a test-generated Ed25519 keypair and the SVTN ID
// to use for all tests that need an admitted node.
type admittedNodeInfo struct {
	svtnID   [16]byte
	nodePub  ed25519.PublicKey
	nodePriv ed25519.PrivateKey
}

// makeAdmittedNode generates a node keypair and a deterministic SVTN ID, writes
// a JSON admission snapshot to a temp file, and sets cfg.AdmissionStateFile so
// that runRouter will load the key on startup. Must be called BEFORE
// startRunRouterWithConfig.
func makeAdmittedNode(t *testing.T, cfg *config.Config) admittedNodeInfo {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("makeAdmittedNode: GenerateKey: %v", err)
	}
	var svtnID [16]byte
	svtnID[0] = 0xAB // deterministic test SVTN

	// Build a minimal admission snapshot JSON directly (avoids importing the
	// private marshalSnapshot helper — uses the public snapshotFile fields
	// that writeSnapshotAtomic / loadSnapshotFromFile use).
	type snapKey struct {
		PubKey  string `json:"pubkey"`
		Role    string `json:"role"`
		Revoked bool   `json:"revoked"`
	}
	type snapSVTN struct {
		SVTNID string    `json:"svtn_id"`
		Keys   []snapKey `json:"keys"`
	}
	type snapFile struct {
		SchemaVersion int        `json:"schema_version"`
		Timestamp     string     `json:"timestamp"`
		SVTNs         []snapSVTN `json:"svtns"`
	}
	snap := snapFile{
		SchemaVersion: 1,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		SVTNs: []snapSVTN{{
			SVTNID: hex.EncodeToString(svtnID[:]),
			Keys: []snapKey{{
				PubKey:  base64.RawURLEncoding.EncodeToString([]byte(pub)),
				Role:    "access",
				Revoked: false,
			}},
		}},
	}
	data, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("makeAdmittedNode: json.Marshal: %v", err)
	}
	f, err := os.CreateTemp(t.TempDir(), "admission-*.json")
	if err != nil {
		t.Fatalf("makeAdmittedNode: CreateTemp: %v", err)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		t.Fatalf("makeAdmittedNode: write snapshot: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("makeAdmittedNode: close snapshot: %v", err)
	}
	cfg.AdmissionStateFile = f.Name()
	return admittedNodeInfo{svtnID: svtnID, nodePub: pub, nodePriv: priv}
}

// completeNodeHandshake performs the node side of the NODE_IDENTIFY handshake
// on conn using the provided admitted node info. The router must already be
// running and waiting for the handshake.
func completeNodeHandshake(t *testing.T, conn net.Conn, info admittedNodeInfo) {
	t.Helper()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer func() { _ = conn.SetDeadline(time.Time{}) }()

	// Send NodeIdentify (80 bytes).
	niFrame := buildNodeIdentifyFrame(info.svtnID, info.nodePub)
	if _, err := conn.Write(niFrame); err != nil {
		t.Fatalf("completeNodeHandshake: write NodeIdentify: %v", err)
	}

	// Read Challenge (144 bytes); extract nonce from bytes [48:80].
	buf := make([]byte, 144)
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatalf("completeNodeHandshake: read Challenge: %v", err)
	}
	var nonce [32]byte
	copy(nonce[:], buf[48:80])

	// Send ChallengeResponse (112 bytes).
	crFrame := buildChallengeResponseFrame(info.svtnID, info.nodePriv, nonce)
	if _, err := conn.Write(crFrame); err != nil {
		t.Fatalf("completeNodeHandshake: write ChallengeResponse: %v", err)
	}
}

// dialNodeAdmitted dials a node connection to cfg.ListenAddr and completes the
// NODE_IDENTIFY handshake. Returns the open (post-handshake) connection.
// The connection is closed via t.Cleanup.
func dialNodeAdmitted(t *testing.T, cfg *config.Config, info admittedNodeInfo) net.Conn {
	t.Helper()
	conn, err := net.Dial("tcp", cfg.ListenAddr)
	if err != nil {
		t.Fatalf("dialNodeAdmitted: dial %s: %v", cfg.ListenAddr, err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	completeNodeHandshake(t, conn, info)
	return conn
}

// dialNodeAndAwaitRegistrationAdmitted is the NODE_IDENTIFY-aware version of
// dialNodeAndAwaitRegistration. It calls makeAdmittedNode, starts runRouter,
// dials a connection, completes the handshake, and waits for nodeConnRegistered.
func dialNodeAndAwaitRegistrationAdmitted(t *testing.T, cfg *config.Config, buf *syncBuffer) (
	conn net.Conn, cancel context.CancelFunc, awaitReturn func() error,
) {
	t.Helper()
	info := makeAdmittedNode(t, cfg)
	events := setNodeConnHook(t)
	errCh, cancelFn := startRunRouterWithConfig(t, cfg, buf)

	var once sync.Once
	var waitErr error
	awaitReturn = func() error {
		once.Do(func() {
			select {
			case waitErr = <-errCh:
			case <-time.After(4 * time.Second):
				waitErr = fmt.Errorf("runRouter did not return within 4s after ctx cancel")
			}
		})
		return waitErr
	}
	t.Cleanup(func() {
		cancelFn()
		_ = awaitReturn()
	})

	conn = dialNodeAdmitted(t, cfg, info)
	awaitNodeConnEvent(t, events, nodeConnRegistered, 4*time.Second)
	return conn, cancelFn, awaitReturn
}

// Compile-time check: ensure admission is imported (used by makeAdmittedNode indirectly
// via the snapshot schema). The _ import ensures the package is linked even if
// Go's unused-import check is triggered before the function body is analyzed.
var _ = admission.RoleAccess

// ── shared raw-frame helpers (Q-CTL-GUARD pins) ─────────────────────────────

// writeRawFrame encodes hdr (with PayloadLen overwritten to len(payload))
// plus payload and writes the wire bytes to conn.
func writeRawFrame(t *testing.T, conn net.Conn, hdr frame.OuterHeader, payload []byte) {
	t.Helper()
	hdr.PayloadLen = uint16(len(payload))
	wire := frame.EncodeOuterHeader(hdr)
	out := append(append([]byte{}, wire[:]...), payload...)
	if _, err := conn.Write(out); err != nil {
		t.Fatalf("writeRawFrame: write: %v", err)
	}
}

// isTimeoutErr reports whether err is a net.Error with Timeout() true —
// distinguishes "nothing to read yet" from "peer closed the connection".
func isTimeoutErr(err error) bool {
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

// assertConnAlive proves the router did NOT close conn in response to
// whatever was written immediately before this call: it writes a
// subsequent well-formed FrameTypeData frame and confirms (a) the write
// succeeds and (b) a short read attempt times out rather than returning
// EOF/connection-reset. The router never replies to Data frames, so a read
// timeout — not a reply — is the expected, implementation-agnostic proof
// the connection stayed open (Q-CTL-GUARD pin test liveness step).
func assertConnAlive(t *testing.T, conn net.Conn) {
	t.Helper()
	writeRawFrame(t, conn, frame.OuterHeader{Version: frame.VersionByte, FrameType: frame.FrameTypeData}, []byte{0x00})
	_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	rbuf := make([]byte, 1)
	_, err := conn.Read(rbuf)
	_ = conn.SetReadDeadline(time.Time{})
	if err == nil {
		t.Fatalf("assertConnAlive: unexpected data from server (the router never replies to Data frames)")
	}
	if !isTimeoutErr(err) {
		t.Fatalf("assertConnAlive: conn appears closed by the router: %v (want a read timeout, proving the conn stayed open)", err)
	}
}

// ── AC-001 ───────────────────────────────────────────────────────────────

// TestDrainObserver_AssemblesAndSendsDRAINFrame proves BC-2.09.002 PC-1 /
// Q1 / Q-SEAM / Q-SINGLE-OBS: the single startup drain observer, at
// drainCoord.Signal time, assembles a FrameTypeCtl frame with
// control_type=0x01 (DRAIN) and non-blocking-sends it to every node
// registered in the live per-node send map, and drainCoord.Wait returns
// nil.
//
// Updated by S-BL.NODE-IDENTIFY-WIRE Task 20: uses dialNodeAndAwaitRegistrationAdmitted
// because onAccept now requires the NODE_IDENTIFY handshake before sendMap.Store.
func TestDrainObserver_AssemblesAndSendsDRAINFrame(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket, mutates
	// the package-level nodeConnHook test hook (Q-AC002 test-isolation).

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}

	var buf syncBuffer
	conn, cancel, awaitReturn := dialNodeAndAwaitRegistrationAdmitted(t, cfg, &buf)

	cancel() // production path 1 (Q3-AMENDED): shutdown block → drainCoord.Signal

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	hdr, payload, err := frame.ReadOuterFrame(conn)
	if err != nil {
		t.Fatalf("AC-001: reading DRAIN frame from simulated node: %v "+
			"(single startup observer never sent a frame)", err)
	}
	if hdr.FrameType != frame.FrameTypeCtl {
		t.Errorf("AC-001: FrameType = %#x, want FrameTypeCtl (%#x)", byte(hdr.FrameType), byte(frame.FrameTypeCtl))
	}
	if len(payload) < 1 || payload[0] != 0x01 {
		t.Errorf("AC-001: payload[0] = %v, want 0x01 (DRAIN opcode); payload=%v", payload, payload)
	}

	if rErr := awaitReturn(); rErr != nil {
		t.Errorf("runRouter returned error on shutdown: %v", rErr)
	}

	if strings.Contains(buf.String(), "BC-2.09.002 EC-003") {
		t.Errorf("AC-001: drainCoord.Wait did not return nil — EC-003 forced-exit marker present; got:\n%s", buf.String())
	}
}

// ── AC-002 ───────────────────────────────────────────────────────────────

// TestNetingress_OnAccept_RegistersNodeHandle proves AC-002 postcondition
// 1: netingress.Serve allocates NodeHandle.IfaceID from the
// ServeConfig.IfaceIDSeed-seeded counter (>= 2; peIfaceID=1 stays
// reserved) and calls OnAccept for an admitted connection; runRouter's
// OnAccept closure fires nodeConnHook(nodeConnRegistered, ...) — the
// send-map-registration observable (Q-AC002).
//
// Updated by S-BL.NODE-IDENTIFY-WIRE Task 20: uses makeAdmittedNode +
// dialNodeAdmitted because onAccept now requires the NODE_IDENTIFY handshake
// before sendMap.Store fires nodeConnHook(nodeConnRegistered).
func TestNetingress_OnAccept_RegistersNodeHandle(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket, mutates
	// the package-level nodeConnHook test hook (Q-AC002 test-isolation).

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}

	info := makeAdmittedNode(t, cfg)
	events := setNodeConnHook(t)
	var buf syncBuffer
	errCh, cancel := startRunRouterWithConfig(t, cfg, &buf)
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(4 * time.Second):
		}
	})

	dialNodeAdmitted(t, cfg, info)

	rec := awaitNodeConnEvent(t, events, nodeConnRegistered, 4*time.Second)
	if rec.ifaceID < 2 {
		t.Errorf("AC-002: NodeHandle.IfaceID = %d, want >= 2 (peIfaceID=1 reserved)", rec.ifaceID)
	}
}

// TestRunRouter_NodeConnClose_CleansUpSendMap proves AC-002 postcondition
// 4: when the node's connection closes, OnAccept's behavior-cleanup func()
// removes the per-node send-map entry and fires
// nodeConnHook(nodeConnRemoved, ...) for the same IfaceID that was
// registered.
//
// Updated by S-BL.NODE-IDENTIFY-WIRE Task 20: polls dial + completes the
// NODE_IDENTIFY handshake before awaiting nodeConnRegistered.
func TestRunRouter_NodeConnClose_CleansUpSendMap(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket, mutates
	// the package-level nodeConnHook test hook (Q-AC002 test-isolation).

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}

	info := makeAdmittedNode(t, cfg)
	events := setNodeConnHook(t)
	var buf syncBuffer
	errCh, cancel := startRunRouterWithConfig(t, cfg, &buf)
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(4 * time.Second):
		}
	})

	// Poll-dial the data-plane listener then complete the handshake.
	var conn net.Conn
	var err error
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn, err = net.DialTimeout("tcp", cfg.ListenAddr, 50*time.Millisecond)
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("dial %s: %v", cfg.ListenAddr, err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	completeNodeHandshake(t, conn, info)

	registered := awaitNodeConnEvent(t, events, nodeConnRegistered, 4*time.Second)

	// Close from the CLIENT side — this is the trigger for the server's
	// per-conn goroutine to observe a read error, return from ServeConn,
	// and invoke OnAccept's behavior-cleanup func() (Q-SEAM ownership
	// split; AC-002 postcondition 4).
	_ = conn.Close()

	removed := awaitNodeConnEvent(t, events, nodeConnRemoved, 2*time.Second)
	if removed.ifaceID != registered.ifaceID {
		t.Errorf("AC-002 cleanup: nodeConnRemoved ifaceID = %d, want %d (same connection)",
			removed.ifaceID, registered.ifaceID)
	}
}

// ── AC-003 ───────────────────────────────────────────────────────────────

// TestDrainObserver_RegisteredAtStartup_FiresOnSignal proves AC-003
// postcondition 1 (Q-SINGLE-OBS / Q-AC003): the single startup drain
// observer is registered at drainCoord-construction time and fires
// drainObserverFiredHook as the first statement of its body, on every
// Signal — independent of any live node connection.
//
// RED GATE: runRouter registers zero production observers today; nothing
// ever sets drainObserverFiredHook, so it never fires.
func TestDrainObserver_RegisteredAtStartup_FiresOnSignal(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket, mutates
	// the package-level drainObserverFiredHook test hook (F-DW-SP4-005
	// test-isolation requirement).

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}

	fired := make(chan struct{})
	var once sync.Once
	drainObserverFiredHook = func() { once.Do(func() { close(fired) }) }
	t.Cleanup(func() { drainObserverFiredHook = nil })

	var buf syncBuffer
	errCh, cancel := startRunRouterWithConfig(t, cfg, &buf)

	cancel()

	select {
	case rErr := <-errCh:
		if rErr != nil {
			t.Errorf("runRouter returned error on shutdown: %v", rErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runRouter did not return within 2s after ctx cancel")
	}

	select {
	case <-fired:
	default:
		t.Errorf("AC-003: drainObserverFiredHook did not fire (single startup observer not registered/invoked)")
	}

	if strings.Contains(buf.String(), "BC-2.09.002 EC-003") {
		t.Errorf("AC-003: drainCoord.Wait did not return nil — EC-003 forced-exit marker present; got:\n%s", buf.String())
	}
}

// TestRouter_CtlFrame_ShortPayload_NoConnClose proves the Q-CTL-GUARD /
// BC-2.01.008 EC-002 length guard: a FrameTypeCtl frame with
// payload_len < 4 is silently discarded (E-PRT-002 logged, no
// conn.RemoteAddr() clause — no conn is in scope in the route closure) and
// the connection is NOT closed.
//
// Updated by S-BL.NODE-IDENTIFY-WIRE Task 20: completes the NODE_IDENTIFY
// handshake (required by onAccept) before sending the ctl test frame.
func TestRouter_CtlFrame_ShortPayload_NoConnClose(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}

	info := makeAdmittedNode(t, cfg)
	var buf syncBuffer
	errCh, cancel := startRunRouterWithConfig(t, cfg, &buf)
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(4 * time.Second):
		}
	})

	conn := dialNodeAdmitted(t, cfg, info)

	// PayloadLen=1 — shorter than the 4-byte control message BC-2.01.008
	// requires before payload[0] may be read.
	writeRawFrame(t, conn, frame.OuterHeader{Version: frame.VersionByte, FrameType: frame.FrameTypeCtl}, []byte{0x00})

	assertConnAlive(t, conn)

	if !scanForLine(&buf, "E-PRT-002", 2*time.Second) {
		t.Errorf("Q-CTL-GUARD: E-PRT-002 marker not found in log for short ctl payload; got:\n%s", buf.String())
	}
}

// TestRouter_CtlFrame_UnknownControlType_SilentIgnore proves BC-2.01.008
// v1.1 PC-4 / Invariant 2 / FO-DRAIN-WIRE-001: a well-formed (>=4-byte)
// FrameTypeCtl frame carrying a control_type the router does not dispatch
// (0xFF here; also covers the reserved-but-undispatched 0x02 RESYNC) is
// silently ignored with NO logging of any kind and no connection close.
//
// Updated by S-BL.NODE-IDENTIFY-WIRE Task 20: completes the NODE_IDENTIFY
// handshake (required by onAccept) before sending the unknown-ctl test frame.
func TestRouter_CtlFrame_UnknownControlType_SilentIgnore(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}

	info := makeAdmittedNode(t, cfg)
	var buf syncBuffer
	errCh, cancel := startRunRouterWithConfig(t, cfg, &buf)
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(4 * time.Second):
		}
	})

	conn := dialNodeAdmitted(t, cfg, info)

	// 4-byte payload with an unrecognized control_type (0xFF) — neither
	// 0x01 (DRAIN) nor the reserved-but-undispatched 0x02 (RESYNC).
	writeRawFrame(t, conn, frame.OuterHeader{Version: frame.VersionByte, FrameType: frame.FrameTypeCtl}, []byte{0xFF, 0x01, 0x00, 0x00})

	assertConnAlive(t, conn)

	got := buf.String()
	if strings.Contains(got, "unrecognized ctl control_type") {
		t.Errorf("BC-2.01.008 PC-4: unknown control_type must NOT be logged at all; got:\n%s", got)
	}
	if strings.Contains(got, "E-PRT-002") {
		t.Errorf("Q-CTL-GUARD: E-PRT-002 is reserved for the short-payload case, not a well-formed-but-unrecognized control_type; got:\n%s", got)
	}
}

// ── AC-004 ───────────────────────────────────────────────────────────────

// TestE2E_RouterDrain_WireRoundTrip is the VP-037 stage-1 discharge
// (Q4-AMENDED): an untagged (non-`go:build integration`) real-runRouter
// test proving the wire round-trip — a connected node receives a
// FrameTypeCtl frame with payload[0]=0x01 within 2s of drainCoord.Signal,
// and drainCoord.Wait returns nil within the default drain window.
//
// Updated by S-BL.NODE-IDENTIFY-WIRE Task 20: uses
// dialNodeAndAwaitRegistrationAdmitted because onAccept now requires the
// NODE_IDENTIFY handshake before sendMap.Store.
func TestE2E_RouterDrain_WireRoundTrip(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket, mutates
	// the package-level nodeConnHook test hook (Q-AC002 test-isolation).

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}

	var buf syncBuffer
	conn, cancel, awaitReturn := dialNodeAndAwaitRegistrationAdmitted(t, cfg, &buf)

	cancel()

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	hdr, payload, err := frame.ReadOuterFrame(conn)
	if err != nil {
		t.Fatalf("VP-037 stage-1: reading DRAIN frame from simulated node: %v", err)
	}
	if hdr.FrameType != frame.FrameTypeCtl {
		t.Errorf("VP-037 stage-1: FrameType = %#x, want FrameTypeCtl (%#x)", byte(hdr.FrameType), byte(frame.FrameTypeCtl))
	}
	if len(payload) < 1 || payload[0] != 0x01 {
		t.Errorf("VP-037 stage-1: payload[0] = %v, want 0x01 (DRAIN opcode per BC-2.01.008 PC-2); payload=%v", payload, payload)
	}

	if rErr := awaitReturn(); rErr != nil {
		t.Errorf("runRouter returned error on shutdown: %v", rErr)
	}

	if strings.Contains(buf.String(), "BC-2.09.002 EC-003") {
		t.Errorf("VP-037 stage-1: drainCoord.Wait did not return nil — EC-003 forced-exit marker present; got:\n%s", buf.String())
	}
}
