// node_identify_wire_test.go — Red Gate test suite for S-BL.NODE-IDENTIFY-WIRE.
//
// Covers AC-001 through AC-009, AC-011, AC-012.
// AC-010 (LWW rebind) and unit tests for BindInterface/UnbindInterface are in
// internal/routing/identity_test.go.
// AC-013 (AdmitNode expiry) is in internal/admission/admitnode_expiry_test.go.
//
// All tests in this file are RED GATE tests — they MUST FAIL against the
// unimplemented stubs in node_identify_wire.go and identity.go.
//
// Test structure: all I/O runs in goroutines (both the router side via
// nodeIdentifyHandshake and the node side via helper goroutines). This prevents
// deadlocks against stubs that return without reading from the connection —
// the router goroutine closes its end, unblocking the node goroutine.
//
// Traces to BC-2.01.009 (all error paths and success path).
// Traces to BC-2.01.010 PC-8 (cleanup removes binding) for AC-012.
package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/config"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/netingress"
	"github.com/arcavenae/switchboard/internal/routing"
)

// ── test helpers ──────────────────────────────────────────────────────────────

// mustGenKeyHandshake generates an Ed25519 keypair or fatals.
func mustGenKeyHandshake(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	return pub, priv
}

// mustSVTNHandshake returns a deterministic [16]byte SVTN ID.
func mustSVTNHandshake(b byte) [16]byte {
	var id [16]byte
	id[0] = b
	return id
}

// nodeAddrHandshake derives the node address from (svtnID, pubkey) using the
// same SHA-256 truncation as frame.DeriveNodeAddress.
func nodeAddrHandshake(svtnID [16]byte, pubKey ed25519.PublicKey) [8]byte {
	h := sha256.New()
	h.Write(svtnID[:])
	h.Write([]byte(pubKey))
	sum := h.Sum(nil)
	var addr [8]byte
	copy(addr[:], sum[:8])
	return addr
}

// encodeCtlHeaderRaw encodes a 44-byte outer header for a FrameTypeCtl (0x03) frame.
// Layout: version(1)+frameType(1)+payloadLen(2)+svtnID(16)+srcAddr(8)+dstAddr(8)+hmacTag(8).
// frameType is always 0x03 (FrameTypeCtl) for all NODE_IDENTIFY handshake test frames.
func encodeCtlHeaderRaw(payloadLen uint16, svtnID [16]byte) []byte {
	hdr := make([]byte, 44)
	hdr[0] = 0x01 // version = VersionByte
	hdr[1] = 0x03 // FrameTypeCtl
	binary.BigEndian.PutUint16(hdr[2:4], payloadLen)
	copy(hdr[4:20], svtnID[:])
	// src_addr[8], dst_addr[8], hmac_tag[8] remain zero
	return hdr
}

// buildNodeIdentifyFrame builds a valid 80-byte NodeIdentify frame.
// Uses encodeNodeIdentify (the production codec) so this helper doubles as
// a regression guard for that function.
func buildNodeIdentifyFrame(svtnID [16]byte, pubkey ed25519.PublicKey) []byte {
	return encodeNodeIdentify(svtnID, pubkey)
}

// buildChallengeResponseFrame builds a valid 112-byte ChallengeResponse frame.
// Uses encodeChallengeResponse (the production codec) so this helper doubles as
// a regression guard for that function.
func buildChallengeResponseFrame(svtnID [16]byte, nodePriv ed25519.PrivateKey, nonce [32]byte) []byte {
	sig := ed25519.Sign(nodePriv, nonce[:])
	resp := admission.ChallengeResponse{NonceSig: sig}
	return encodeChallengeResponse(svtnID, resp)
}

// buildChallengeResponseFrameWrongKey builds a ChallengeResponse signed with
// a DIFFERENT private key (bad signature).
func buildChallengeResponseFrameWrongKey(svtnID [16]byte, wrongPriv ed25519.PrivateKey, nonce [32]byte) []byte {
	sig := ed25519.Sign(wrongPriv, nonce[:])
	resp := admission.ChallengeResponse{NonceSig: sig}
	return encodeChallengeResponse(svtnID, resp)
}

// newTestHandle constructs a minimal netingress.NodeHandle for testing.
func newTestHandle(ifaceID routing.InterfaceID) netingress.NodeHandle {
	return netingress.NodeHandle{
		IfaceID: ifaceID,
		Send:    make(chan []byte, 16),
		Done:    make(chan struct{}),
	}
}

// handshakeResult captures the outputs of a nodeIdentifyHandshake call.
type handshakeResult struct {
	svtnID   [16]byte
	nodeAddr [8]byte
	err      error
}

// runRouterSide runs nodeIdentifyHandshake in a goroutine, sends the result
// on the returned channel, and closes routerConn when done to unblock the node
// side. Callers must drain the channel.
func runRouterSide(
	routerConn net.Conn,
	r *routing.Router,
	routerPriv ed25519.PrivateKey,
	ks *admission.AdmittedKeySet,
	h netingress.NodeHandle,
) <-chan handshakeResult {
	ch := make(chan handshakeResult, 1)
	go func() {
		svtnID, nodeAddr, err := nodeIdentifyHandshake(routerConn, r, routerPriv, ks, h)
		ch <- handshakeResult{svtnID: svtnID, nodeAddr: nodeAddr, err: err}
		// Close routerConn so that any blocked write/read on nodeConn unblocks.
		_ = routerConn.Close()
	}()
	return ch
}

// doFullNodeHandshake sends NodeIdentify, reads Challenge, sends ChallengeResponse.
// Runs in a goroutine; uses a 5s deadline. Errors are non-fatal (the router
// side may close the conn before the node side finishes).
func doFullNodeHandshake(nodeConn net.Conn, svtnID [16]byte, nodePub ed25519.PublicKey, nodePriv ed25519.PrivateKey) {
	_ = nodeConn.SetDeadline(time.Now().Add(5 * time.Second))

	// Send NodeIdentify.
	frame := buildNodeIdentifyFrame(svtnID, nodePub)
	if _, err := nodeConn.Write(frame); err != nil {
		return
	}

	// Read Challenge (144 bytes); extract nonce from bytes 48:80.
	buf := make([]byte, 144)
	if _, err := io.ReadFull(nodeConn, buf); err != nil {
		return
	}
	var nonce [32]byte
	copy(nonce[:], buf[48:80])

	// Send ChallengeResponse.
	crFrame := buildChallengeResponseFrame(svtnID, nodePriv, nonce)
	_, _ = nodeConn.Write(crFrame)
}

// ── AC-001: Successful handshake ──────────────────────────────────────────────

// TestNodeIdentifyHandshake_Success_BindingRecorded verifies the full success
// path: admitted, non-revoked, non-expired key + valid signature → handshake
// returns nil, BindInterface is called, LookupInterface returns correct IfaceID.
//
// Traces to BC-2.01.009 PC-1 through PC-7; BC-2.01.010 PC-1; AC-001.
func TestNodeIdentifyHandshake_Success_BindingRecorded(t *testing.T) {
	t.Parallel()

	_, routerPriv := mustGenKeyHandshake(t)
	nodePub, nodePriv := mustGenKeyHandshake(t)
	svtnID := mustSVTNHandshake(0x01)

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)
	r := routing.NewRouter(ks)

	routerConn, nodeConn := net.Pipe()
	t.Cleanup(func() { _ = routerConn.Close(); _ = nodeConn.Close() })

	const ifaceID routing.InterfaceID = 10
	h := newTestHandle(ifaceID)

	resCh := runRouterSide(routerConn, r, routerPriv, ks, h)

	var nodeWG sync.WaitGroup
	nodeWG.Add(1)
	go func() {
		defer nodeWG.Done()
		doFullNodeHandshake(nodeConn, svtnID, nodePub, nodePriv)
	}()

	res := <-resCh
	nodeWG.Wait()

	if res.err != nil {
		t.Fatalf("nodeIdentifyHandshake: want nil, got %v", res.err)
	}

	got, ok := r.LookupInterface(res.svtnID, res.nodeAddr)
	if !ok {
		t.Fatal("LookupInterface after success: want (ifaceID, true), got (_, false)")
	}
	if got != ifaceID {
		t.Errorf("LookupInterface after success: want %d, got %d", ifaceID, got)
	}
}

// TestNodeIdentifyHandshake_Success_ServeConnBegins_FrameRouted verifies that
// after a successful handshake ServeConn is actually running and processing
// frames — not just that nodeIdentifyHandshake returned nil. It drives the full
// daemon path (runRouter → onAccept → ServeConn) and sends a post-handshake
// FrameTypeData frame after the nodeConnRegistered hook fires. The frame must be
// consumed by the running ServeConn read loop; if ServeConn never started the
// connection would be closed or the frame unread.
//
// Discriminating property: if the daemon's onAccept did not hand the connection
// to ServeConn (i.e. the post-handshake hand-off were broken), the connection
// would be closed from the router side and assertConnAlive's write would fail
// or the subsequent read would return EOF/reset instead of a read timeout. The
// read-timeout outcome uniquely proves ServeConn is reading frames — the router
// never replies to FrameTypeData, so a timeout (not EOF, not data) is the only
// proof the connection is alive and being read.
//
// NOT t.Parallel(): binds ephemeral TCP + filesystem socket, overrides the
// package-level nodeConnHook test hook (Q-AC002 test-isolation requirement
// shared with router_drain_wire_test.go).
//
// Traces to BC-2.01.009 PC-6 (ServeConn begins), PC-7 (deadline cleared),
// PC-8 (fully-bound state); AC-001; story test-plan line 233-234.
func TestNodeIdentifyHandshake_Success_ServeConnBegins_FrameRouted(t *testing.T) {
	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}

	// Pre-load an admission snapshot so the daemon admits the test node key.
	info := makeAdmittedNode(t, cfg)

	// Install the channel-backed nodeConnHook so we can synchronise on
	// nodeConnRegistered (fires after sendMap.Store, i.e. after onAccept
	// returns and ServeConn has started reading the per-conn data-plane loop).
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

	// Dial and complete the NODE_IDENTIFY handshake. dialNodeAdmitted runs
	// completeNodeHandshake under the hood; after it returns the conn is
	// post-handshake (deadline cleared by the daemon's onAccept).
	nodeConn := dialNodeAdmitted(t, cfg, info)

	// Wait until onAccept has stored the node in sendMap and the
	// nodeConnRegistered hook has fired. After this event, ServeConn is
	// running and reading frames on nodeConn.
	awaitNodeConnEvent(t, events, nodeConnRegistered, 4*time.Second)

	// ASSERT ServeConn is running: send a FrameTypeData frame (router never
	// replies) and confirm the connection stays alive by observing a read
	// timeout rather than EOF/reset.
	//
	// Discriminating: if ServeConn were not started after onAccept (broken
	// hand-off), the daemon would not be reading from nodeConn. The TCP
	// socket would remain half-open from the daemon side; the write would
	// still succeed (TCP buffers it) but the socket would not be actively
	// drained. More crucially, any prior daemon-side Close would cause our
	// read to return EOF/reset immediately — not a timeout — so the
	// assertConnAlive timeout outcome is uniquely tied to ServeConn running.
	assertConnAlive(t, nodeConn)
}

// TestNodeIdentifyHandshake_Success_ServeConnBegins verifies that after a
// successful handshake the connection is left open (deadline cleared).
//
// Traces to BC-2.01.009 PC-7 (deadline cleared), PC-8 (fully bound); AC-001.
func TestNodeIdentifyHandshake_Success_ServeConnBegins(t *testing.T) {
	t.Parallel()

	_, routerPriv := mustGenKeyHandshake(t)
	nodePub, nodePriv := mustGenKeyHandshake(t)
	svtnID := mustSVTNHandshake(0x02)

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)
	r := routing.NewRouter(ks)

	// Use a loopback TCP connection (not net.Pipe) so we can test that the
	// connection stays open after the handshake. net.Pipe behaves differently
	// from real TCP for deadline semantics.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	var routerConn net.Conn
	var acceptErr error
	var acceptWG sync.WaitGroup
	acceptWG.Add(1)
	go func() {
		defer acceptWG.Done()
		routerConn, acceptErr = ln.Accept()
	}()

	nodeConn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { _ = nodeConn.Close() })

	acceptWG.Wait()
	if acceptErr != nil {
		t.Fatalf("Accept: %v", acceptErr)
	}
	t.Cleanup(func() { _ = routerConn.Close() })

	h := newTestHandle(11)

	resCh := runRouterSide(routerConn, r, routerPriv, ks, h)

	var nodeWG sync.WaitGroup
	nodeWG.Add(1)
	go func() {
		defer nodeWG.Done()
		doFullNodeHandshake(nodeConn, svtnID, nodePub, nodePriv)
	}()

	res := <-resCh
	nodeWG.Wait()

	if res.err != nil {
		t.Fatalf("nodeIdentifyHandshake: want nil, got %v", res.err)
	}
	// If we reach here, handshake returned nil — connection stayed open.
}

// ── AC-002: Malformed NodeIdentify ────────────────────────────────────────────

// TestNodeIdentifyHandshake_MalformedNodeIdentify_WrongPayloadLen verifies that
// a NodeIdentify frame with payload_len != 36 causes the connection to close.
//
// Traces to BC-2.01.009 Invariant 5 (exact payload lengths enforced); AC-002.
func TestNodeIdentifyHandshake_MalformedNodeIdentify_WrongPayloadLen(t *testing.T) {
	t.Parallel()

	_, routerPriv := mustGenKeyHandshake(t)
	svtnID := mustSVTNHandshake(0x10)

	ks := admission.NewAdmittedKeySet()
	r := routing.NewRouter(ks)

	routerConn, nodeConn := net.Pipe()
	t.Cleanup(func() { _ = routerConn.Close(); _ = nodeConn.Close() })

	h := newTestHandle(20)
	resCh := runRouterSide(routerConn, r, routerPriv, ks, h)

	// Send a frame with payload_len=20 (wrong — should be 36).
	go func() {
		_ = nodeConn.SetDeadline(time.Now().Add(3 * time.Second))
		hdr := encodeCtlHeaderRaw(20, svtnID)
		payload := make([]byte, 20)
		payload[0] = 0x04
		payload[1] = 0x01
		payload[2] = 0x01
		payload[3] = 0x00
		_, _ = nodeConn.Write(append(hdr, payload...))
		// Read response (will get EOF/error when router closes conn).
		_, _ = io.ReadFull(nodeConn, make([]byte, 1))
	}()

	res := <-resCh
	if res.err == nil {
		t.Fatal("nodeIdentifyHandshake: want error for wrong payload_len, got nil")
	}
	if strings.Contains(res.err.Error(), "unimplemented") {
		t.Errorf("nodeIdentifyHandshake: got stub error %q; want real malformed-frame error", res.err)
	}
	// AC-002 PC-4: the malformed-NodeIdentify path emits a WARN at the
	// nodeIdentifyHandshake level (before onAccept's classification switch,
	// which only runs for admission sentinels). The returned error must contain
	// "malformed NodeIdentify" — the literal shared by all wrong-payload-len
	// and wrong-field-value paths in node_identify_wire.go.
	// Discriminating: changing that literal prefix in production fails this
	// assertion while the nil-check above stays green.
	const wantMalformed = "malformed NodeIdentify"
	if !strings.Contains(res.err.Error(), wantMalformed) {
		t.Errorf("AC-002 PC-4: error does not indicate malformed NodeIdentify; want substring %q, got: %q",
			wantMalformed, res.err.Error())
	}
}

// TestNodeIdentifyHandshake_MalformedNodeIdentify_WrongMsgKind verifies that
// a NodeIdentify frame with msg_kind != 0x01 causes the connection to close
// with a decode error that names the offending field.
//
// Traces to BC-2.01.009 Invariant 5; AC-002.
//
// NOT t.Parallel(): overrides the package-level nodeIdentifyHandshakeTimeout
// var — parallel execution would race other tests relying on the 10s default.
func TestNodeIdentifyHandshake_MalformedNodeIdentify_WrongMsgKind(t *testing.T) {
	// Override timeout to 200ms: if the msg_kind guard is accidentally removed,
	// the frame decodes successfully and the driver blocks waiting for a
	// ChallengeResponse. The 200ms deadline fires fast, but the resulting
	// deadline error does NOT contain "msg_kind" — the new substring assertion
	// below then fails immediately, giving a clear red rather than a 10s hang.
	orig := nodeIdentifyHandshakeTimeout
	nodeIdentifyHandshakeTimeout = 200 * time.Millisecond
	t.Cleanup(func() { nodeIdentifyHandshakeTimeout = orig })

	_, routerPriv := mustGenKeyHandshake(t)
	nodePub, _ := mustGenKeyHandshake(t)
	svtnID := mustSVTNHandshake(0x11)

	ks := admission.NewAdmittedKeySet()
	r := routing.NewRouter(ks)

	routerConn, nodeConn := net.Pipe()
	t.Cleanup(func() { _ = routerConn.Close(); _ = nodeConn.Close() })

	h := newTestHandle(21)
	resCh := runRouterSide(routerConn, r, routerPriv, ks, h)

	go func() {
		_ = nodeConn.SetDeadline(time.Now().Add(3 * time.Second))
		payload := make([]byte, 36)
		payload[0] = 0x04
		payload[1] = 0x01
		payload[2] = 0x02 // wrong msg_kind — should be 0x01
		payload[3] = 0x00
		copy(payload[4:36], nodePub)
		hdr := encodeCtlHeaderRaw(36, svtnID)
		_, _ = nodeConn.Write(append(hdr, payload...))
		_, _ = io.ReadFull(nodeConn, make([]byte, 1))
	}()

	res := <-resCh
	if res.err == nil {
		t.Fatal("nodeIdentifyHandshake: want error for wrong msg_kind, got nil")
	}
	if strings.Contains(res.err.Error(), "unimplemented") {
		t.Errorf("nodeIdentifyHandshake: got stub error %q; want real malformed-frame error", res.err)
	}
	// Discriminating assertion: the decode guard at node_identify_wire.go:196
	// emits "msg_kind" in the error string. A deadline error (the fallback if
	// the guard is removed) does NOT contain this substring — so removing the
	// guard will fail this assertion quickly within the 200ms window.
	if !strings.Contains(res.err.Error(), "msg_kind") {
		t.Errorf("AC-002: error does not name the offending field; want substring %q, got: %q",
			"msg_kind", res.err.Error())
	}
}

// TestNodeIdentifyHandshake_MalformedNodeIdentify_NonZeroReservedByte verifies
// that a NodeIdentify frame with reserved byte (payload[3]) != 0x00 causes the
// connection to close with a decode error that names the offending field.
//
// Traces to BC-2.01.009 Invariant 5, EC-003; AC-002.
//
// NOT t.Parallel(): overrides the package-level nodeIdentifyHandshakeTimeout
// var — parallel execution would race other tests relying on the 10s default.
func TestNodeIdentifyHandshake_MalformedNodeIdentify_NonZeroReservedByte(t *testing.T) {
	// Override timeout to 200ms: if the reserved-byte guard is accidentally
	// removed, the frame decodes successfully and the driver blocks waiting for a
	// ChallengeResponse. The 200ms deadline fires fast, but the resulting
	// deadline error does NOT contain "reserved byte" — the new substring
	// assertion below then fails immediately, giving a clear red rather than a
	// 10s hang.
	orig := nodeIdentifyHandshakeTimeout
	nodeIdentifyHandshakeTimeout = 200 * time.Millisecond
	t.Cleanup(func() { nodeIdentifyHandshakeTimeout = orig })

	_, routerPriv := mustGenKeyHandshake(t)
	nodePub, _ := mustGenKeyHandshake(t)
	svtnID := mustSVTNHandshake(0x12)

	ks := admission.NewAdmittedKeySet()
	r := routing.NewRouter(ks)

	routerConn, nodeConn := net.Pipe()
	t.Cleanup(func() { _ = routerConn.Close(); _ = nodeConn.Close() })

	h := newTestHandle(22)
	resCh := runRouterSide(routerConn, r, routerPriv, ks, h)

	go func() {
		_ = nodeConn.SetDeadline(time.Now().Add(3 * time.Second))
		payload := make([]byte, 36)
		payload[0] = 0x04
		payload[1] = 0x01
		payload[2] = 0x01
		payload[3] = 0x01 // non-zero reserved — hard decoder error
		copy(payload[4:36], nodePub)
		hdr := encodeCtlHeaderRaw(36, svtnID)
		_, _ = nodeConn.Write(append(hdr, payload...))
		_, _ = io.ReadFull(nodeConn, make([]byte, 1))
	}()

	res := <-resCh
	if res.err == nil {
		t.Fatal("nodeIdentifyHandshake: want error for non-zero reserved byte, got nil")
	}
	if strings.Contains(res.err.Error(), "unimplemented") {
		t.Errorf("nodeIdentifyHandshake: got stub error %q; want real malformed-frame error", res.err)
	}
	// Discriminating assertion: the decode guard at node_identify_wire.go:199
	// emits "reserved byte" in the error string. A deadline error (the fallback
	// if the guard is removed) does NOT contain this substring — so removing the
	// guard will fail this assertion quickly within the 200ms window.
	if !strings.Contains(res.err.Error(), "reserved byte") {
		t.Errorf("AC-002: error does not name the offending field; want substring %q, got: %q",
			"reserved byte", res.err.Error())
	}
}

// TestNodeIdentifyHandshake_MalformedNodeIdentify_WrongControlType verifies that
// a NodeIdentify frame with control_type != 0x04 causes the connection to close
// with a decode error that names the offending field.
//
// Traces to BC-2.01.009 Invariant 5; AC-002 PC-2 (control_type guard).
//
// NOT t.Parallel(): overrides the package-level nodeIdentifyHandshakeTimeout
// var — parallel execution would race other tests relying on the 10s default.
func TestNodeIdentifyHandshake_MalformedNodeIdentify_WrongControlType(t *testing.T) {
	// Override timeout to 200ms: if the control_type guard (node_identify_wire.go:189)
	// is accidentally removed, the frame decodes successfully and the driver blocks
	// waiting for a ChallengeResponse. The 200ms deadline fires fast, but the
	// resulting deadline error does NOT contain "control_type" — the new substring
	// assertion below then fails immediately, giving a clear red rather than a
	// 10s hang.
	orig := nodeIdentifyHandshakeTimeout
	nodeIdentifyHandshakeTimeout = 200 * time.Millisecond
	t.Cleanup(func() { nodeIdentifyHandshakeTimeout = orig })

	_, routerPriv := mustGenKeyHandshake(t)
	nodePub, _ := mustGenKeyHandshake(t)
	svtnID := mustSVTNHandshake(0x13)

	ks := admission.NewAdmittedKeySet()
	r := routing.NewRouter(ks)

	routerConn, nodeConn := net.Pipe()
	t.Cleanup(func() { _ = routerConn.Close(); _ = nodeConn.Close() })

	h := newTestHandle(23)
	resCh := runRouterSide(routerConn, r, routerPriv, ks, h)

	go func() {
		_ = nodeConn.SetDeadline(time.Now().Add(3 * time.Second))
		payload := make([]byte, 36)
		payload[0] = 0x03 // wrong control_type — should be 0x04
		payload[1] = 0x01
		payload[2] = 0x01
		payload[3] = 0x00
		copy(payload[4:36], nodePub)
		hdr := encodeCtlHeaderRaw(36, svtnID)
		_, _ = nodeConn.Write(append(hdr, payload...))
		_, _ = io.ReadFull(nodeConn, make([]byte, 1))
	}()

	res := <-resCh
	if res.err == nil {
		t.Fatal("nodeIdentifyHandshake: want error for wrong control_type, got nil")
	}
	if strings.Contains(res.err.Error(), "unimplemented") {
		t.Errorf("nodeIdentifyHandshake: got stub error %q; want real malformed-frame error", res.err)
	}
	// Discriminating assertion: the decode guard at node_identify_wire.go:189
	// emits "control_type" in the error string. A deadline error (the fallback if
	// the guard is removed) does NOT contain this substring — so removing the
	// guard will fail this assertion quickly within the 200ms window.
	if !strings.Contains(res.err.Error(), "control_type") {
		t.Errorf("AC-002: error does not name the offending field; want substring %q, got: %q",
			"control_type", res.err.Error())
	}
}

// TestNodeIdentifyHandshake_MalformedNodeIdentify_WrongVersion verifies that
// a NodeIdentify frame with version != 0x01 causes the connection to close
// with a decode error that names the offending field.
//
// Traces to BC-2.01.009 Invariant 5; AC-002 PC-2 (version guard).
//
// NOT t.Parallel(): overrides the package-level nodeIdentifyHandshakeTimeout
// var — parallel execution would race other tests relying on the 10s default.
func TestNodeIdentifyHandshake_MalformedNodeIdentify_WrongVersion(t *testing.T) {
	// Override timeout to 200ms: if the version guard (node_identify_wire.go:192)
	// is accidentally removed, the frame decodes successfully and the driver blocks
	// waiting for a ChallengeResponse. The 200ms deadline fires fast, but the
	// resulting deadline error does NOT contain "version" — the new substring
	// assertion below then fails immediately, giving a clear red rather than a
	// 10s hang.
	orig := nodeIdentifyHandshakeTimeout
	nodeIdentifyHandshakeTimeout = 200 * time.Millisecond
	t.Cleanup(func() { nodeIdentifyHandshakeTimeout = orig })

	_, routerPriv := mustGenKeyHandshake(t)
	nodePub, _ := mustGenKeyHandshake(t)
	svtnID := mustSVTNHandshake(0x14)

	ks := admission.NewAdmittedKeySet()
	r := routing.NewRouter(ks)

	routerConn, nodeConn := net.Pipe()
	t.Cleanup(func() { _ = routerConn.Close(); _ = nodeConn.Close() })

	h := newTestHandle(24)
	resCh := runRouterSide(routerConn, r, routerPriv, ks, h)

	go func() {
		_ = nodeConn.SetDeadline(time.Now().Add(3 * time.Second))
		payload := make([]byte, 36)
		payload[0] = 0x04
		payload[1] = 0x02 // wrong version — should be 0x01
		payload[2] = 0x01
		payload[3] = 0x00
		copy(payload[4:36], nodePub)
		hdr := encodeCtlHeaderRaw(36, svtnID)
		_, _ = nodeConn.Write(append(hdr, payload...))
		_, _ = io.ReadFull(nodeConn, make([]byte, 1))
	}()

	res := <-resCh
	if res.err == nil {
		t.Fatal("nodeIdentifyHandshake: want error for wrong version, got nil")
	}
	if strings.Contains(res.err.Error(), "unimplemented") {
		t.Errorf("nodeIdentifyHandshake: got stub error %q; want real malformed-frame error", res.err)
	}
	// Discriminating assertion: the decode guard at node_identify_wire.go:192
	// emits "version" in the error string. A deadline error (the fallback if
	// the guard is removed) does NOT contain this substring — so removing the
	// guard will fail this assertion quickly within the 200ms window.
	if !strings.Contains(res.err.Error(), "version") {
		t.Errorf("AC-002: error does not name the offending field; want substring %q, got: %q",
			"version", res.err.Error())
	}
}

// ── BC-2.01.009: Malformed ChallengeResponse (Message 3) ─────────────────────

// TestNodeIdentifyHandshake_MalformedChallengeResponse_ConnectionClosed verifies
// that a ChallengeResponse frame with a wrong inner discriminator (msg_kind != 0x03)
// causes the connection to close with a decode error naming "ChallengeResponse".
//
// BC-2.01.009 Error-Codes table: "malformed ChallengeResponse frame
// (hdr.PayloadLen != 68, wrong discriminators at ChallengeResponse receipt) →
// Close immediately."
//
// The node side sends a valid NodeIdentify, reads the 144-byte Challenge to
// unblock the router's write on net.Pipe, then sends a ChallengeResponse frame
// whose inner payload carries msg_kind=0xFF instead of 0x03.
// decodeChallengeResponse (node_identify_wire.go:228) returns an error containing
// "ChallengeResponse" before AdmitNode is ever reached.
//
// Discriminating requirement: deleting the msg_kind guard at line 228 allows the
// decode to succeed; AdmitNode is then called with a zero NonceSig against an
// unregistered key, returning admission.ErrNotAdmitted whose message does NOT
// contain "ChallengeResponse" — so the substring assertion below fails immediately
// without any timeout hang (AdmitNode returns synchronously).
//
// NOT t.Parallel(): overrides the package-level nodeIdentifyHandshakeTimeout
// var — parallel execution would race other tests relying on the 10s default.
//
// Traces to BC-2.01.009 Error-Codes table (malformed ChallengeResponse); AC-002.
func TestNodeIdentifyHandshake_MalformedChallengeResponse_ConnectionClosed(t *testing.T) {
	orig := nodeIdentifyHandshakeTimeout
	nodeIdentifyHandshakeTimeout = 200 * time.Millisecond
	t.Cleanup(func() { nodeIdentifyHandshakeTimeout = orig })

	_, routerPriv := mustGenKeyHandshake(t)
	nodePub, _ := mustGenKeyHandshake(t)
	svtnID := mustSVTNHandshake(0x25)

	// Empty keyset: AdmitNode (only reached if the CR decode guard is deleted)
	// returns ErrNotAdmitted, whose message does not contain "ChallengeResponse".
	ks := admission.NewAdmittedKeySet()
	r := routing.NewRouter(ks)

	routerConn, nodeConn := net.Pipe()
	t.Cleanup(func() { _ = routerConn.Close(); _ = nodeConn.Close() })

	h := newTestHandle(47)
	resCh := runRouterSide(routerConn, r, routerPriv, ks, h)

	go func() {
		_ = nodeConn.SetDeadline(time.Now().Add(3 * time.Second))

		// Send valid NodeIdentify (Message 1) so the router proceeds to Message 2.
		niFrame := buildNodeIdentifyFrame(svtnID, nodePub)
		if _, err := nodeConn.Write(niFrame); err != nil {
			return
		}

		// Read the 144-byte Challenge (Message 2) to unblock the router's write
		// on net.Pipe; the router cannot proceed to reading Message 3 until we
		// drain its write.
		if _, err := io.ReadFull(nodeConn, make([]byte, 144)); err != nil {
			return
		}

		// Send malformed ChallengeResponse (Message 3): correct outer frame
		// (FrameTypeCtl, payloadLen=68) but payload[2]=0xFF instead of 0x03
		// (wrong msg_kind). decodeChallengeResponse checks payload[2] first.
		crPayload := make([]byte, challengeResponsePayloadSize)
		crPayload[0] = 0x04 // control_type correct
		crPayload[1] = 0x01 // version correct
		crPayload[2] = 0xFF // msg_kind WRONG — decodeChallengeResponse expects 0x03
		crPayload[3] = 0x00 // reserved correct
		// crPayload[4:68] = zero bytes (NonceSig; decode fails before reading them)
		hdr := encodeCtlHeaderRaw(uint16(challengeResponsePayloadSize), svtnID)
		_, _ = nodeConn.Write(append(hdr, crPayload...))
		_, _ = io.ReadFull(nodeConn, make([]byte, 1))
	}()

	res := <-resCh
	if res.err == nil {
		t.Fatal("nodeIdentifyHandshake: want error for malformed ChallengeResponse, got nil")
	}
	if strings.Contains(res.err.Error(), "unimplemented") {
		t.Errorf("nodeIdentifyHandshake: got stub error %q; want real malformed-frame error", res.err)
	}
	// Discriminating assertion: decodeChallengeResponse at node_identify_wire.go:228
	// emits "ChallengeResponse" in the error string. An admission error (the fallback
	// if the guard is removed) does NOT contain this substring — so removing the
	// guard fails this assertion immediately (no timeout required).
	if !strings.Contains(res.err.Error(), "ChallengeResponse") {
		t.Errorf("BC-2.01.009: error does not name the malformed message; want substring %q, got: %q",
			"ChallengeResponse", res.err.Error())
	}
}

// ── AC-003: Zero SVTN ID ──────────────────────────────────────────────────────

// TestNodeIdentifyHandshake_ZeroSVTNID_Rejected verifies that a NodeIdentify
// frame with an all-zero SVTN ID causes the connection to close immediately.
//
// Traces to BC-2.01.009 Precondition 5, Invariant 3; AC-003.
func TestNodeIdentifyHandshake_ZeroSVTNID_Rejected(t *testing.T) {
	t.Parallel()

	_, routerPriv := mustGenKeyHandshake(t)
	nodePub, _ := mustGenKeyHandshake(t)
	var zeroSVTNID [16]byte // all-zero

	ks := admission.NewAdmittedKeySet()
	r := routing.NewRouter(ks)

	routerConn, nodeConn := net.Pipe()
	t.Cleanup(func() { _ = routerConn.Close(); _ = nodeConn.Close() })

	h := newTestHandle(30)
	resCh := runRouterSide(routerConn, r, routerPriv, ks, h)

	go func() {
		_ = nodeConn.SetDeadline(time.Now().Add(3 * time.Second))
		frame := buildNodeIdentifyFrame(zeroSVTNID, nodePub)
		_, _ = nodeConn.Write(frame)
		_, _ = io.ReadFull(nodeConn, make([]byte, 1))
	}()

	res := <-resCh
	if res.err == nil {
		t.Fatal("nodeIdentifyHandshake: want error for zero SVTN ID, got nil")
	}
	if strings.Contains(res.err.Error(), "unimplemented") {
		t.Errorf("nodeIdentifyHandshake: got stub error %q; want real zero-SVTN-ID rejection error", res.err)
	}
	// AC-003 PC-2: the error message MUST contain the exact literal
	// "node_identify: zero SVTN ID rejected" (node_identify_wire.go).
	// Discriminating: renaming that literal in production fails this assertion
	// while the nil-check above stays green — so this is the sole guard for
	// the literal string postcondition.
	const wantLiteral = "node_identify: zero SVTN ID rejected"
	if !strings.Contains(res.err.Error(), wantLiteral) {
		t.Errorf("AC-003 PC-2: error message does not contain literal %q; got: %q", wantLiteral, res.err.Error())
	}
}

// ── AC-004: ErrNotAdmitted ────────────────────────────────────────────────────

// TestNodeIdentifyHandshake_ErrNotAdmitted_ConnectionClosed verifies that when
// AdmitNode returns ErrNotAdmitted, the connection is closed. BindInterface is
// NOT called.
//
// Traces to BC-2.01.009 Error Code E-ADM-003; AC-004.
func TestNodeIdentifyHandshake_ErrNotAdmitted_ConnectionClosed(t *testing.T) {
	t.Parallel()

	_, routerPriv := mustGenKeyHandshake(t)
	nodePub, nodePriv := mustGenKeyHandshake(t)
	svtnID := mustSVTNHandshake(0x20)

	// Empty keyset — key is NOT registered.
	ks := admission.NewAdmittedKeySet()
	r := routing.NewRouter(ks)

	routerConn, nodeConn := net.Pipe()
	t.Cleanup(func() { _ = routerConn.Close(); _ = nodeConn.Close() })

	const ifaceID routing.InterfaceID = 40
	h := newTestHandle(ifaceID)
	resCh := runRouterSide(routerConn, r, routerPriv, ks, h)

	var nodeWG sync.WaitGroup
	nodeWG.Add(1)
	go func() {
		defer nodeWG.Done()
		doFullNodeHandshake(nodeConn, svtnID, nodePub, nodePriv)
	}()

	res := <-resCh
	nodeWG.Wait()

	if res.err == nil {
		t.Fatal("nodeIdentifyHandshake: want error (ErrNotAdmitted), got nil")
	}
	// The error must be ErrNotAdmitted, not the stub "unimplemented" error.
	if !errors.Is(res.err, admission.ErrNotAdmitted) {
		t.Errorf("handshakeErr: want ErrNotAdmitted (E-ADM-003), got %v", res.err)
	}

	// Verify no binding was recorded.
	nodeAddr := nodeAddrHandshake(svtnID, nodePub)
	_, ok := r.LookupInterface(svtnID, nodeAddr)
	if ok {
		t.Error("LookupInterface after ErrNotAdmitted: want (0, false), got (_, true)")
	}
}

// ── AC-005: ErrKeyRevoked ─────────────────────────────────────────────────────

// TestNodeIdentifyHandshake_ErrKeyRevoked_ConnectionClosed verifies that when
// the node's key is revoked, the connection is closed with E-ADM-005.
//
// Traces to BC-2.01.009 Error Code E-ADM-005; AC-005.
func TestNodeIdentifyHandshake_ErrKeyRevoked_ConnectionClosed(t *testing.T) {
	t.Parallel()

	_, routerPriv := mustGenKeyHandshake(t)
	nodePub, nodePriv := mustGenKeyHandshake(t)
	svtnID := mustSVTNHandshake(0x21)

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)
	nodeAddr := nodeAddrHandshake(svtnID, nodePub)
	if err := ks.RevokeKey(svtnID, nodeAddr); err != nil {
		t.Fatalf("RevokeKey: %v", err)
	}

	r := routing.NewRouter(ks)

	routerConn, nodeConn := net.Pipe()
	t.Cleanup(func() { _ = routerConn.Close(); _ = nodeConn.Close() })

	h := newTestHandle(41)
	resCh := runRouterSide(routerConn, r, routerPriv, ks, h)

	var nodeWG sync.WaitGroup
	nodeWG.Add(1)
	go func() {
		defer nodeWG.Done()
		doFullNodeHandshake(nodeConn, svtnID, nodePub, nodePriv)
	}()

	res := <-resCh
	nodeWG.Wait()

	if res.err == nil {
		t.Fatal("nodeIdentifyHandshake: want error (ErrKeyRevoked), got nil")
	}
	if !errors.Is(res.err, admission.ErrKeyRevoked) {
		t.Errorf("handshakeErr: want ErrKeyRevoked (E-ADM-005), got %v", res.err)
	}

	_, ok := r.LookupInterface(svtnID, nodeAddr)
	if ok {
		t.Error("LookupInterface after ErrKeyRevoked: want (0, false), got (_, true)")
	}
}

// ── AC-006: ErrKeyExpired ─────────────────────────────────────────────────────

// TestNodeIdentifyHandshake_ErrKeyExpired_ConnectionClosed verifies that when
// the node's key has a past expiry, the connection is closed with E-ADM-015.
//
// REQUIRES Task 16: AdmitNode must gain expiry check. RED GATE until then.
//
// Traces to BC-2.01.009 Error Code E-ADM-015; BC-2.05.001 PC-6; AC-006.
func TestNodeIdentifyHandshake_ErrKeyExpired_ConnectionClosed(t *testing.T) {
	t.Parallel()

	_, routerPriv := mustGenKeyHandshake(t)
	nodePub, nodePriv := mustGenKeyHandshake(t)
	svtnID := mustSVTNHandshake(0x22)

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)
	nodeAddr := nodeAddrHandshake(svtnID, nodePub)
	pastExpiry := time.Now().UTC().Add(-time.Second)
	if err := ks.SetKeyExpiry(svtnID, nodeAddr, pastExpiry); err != nil {
		t.Fatalf("SetKeyExpiry: %v", err)
	}

	r := routing.NewRouter(ks)

	routerConn, nodeConn := net.Pipe()
	t.Cleanup(func() { _ = routerConn.Close(); _ = nodeConn.Close() })

	h := newTestHandle(42)
	resCh := runRouterSide(routerConn, r, routerPriv, ks, h)

	var nodeWG sync.WaitGroup
	nodeWG.Add(1)
	go func() {
		defer nodeWG.Done()
		doFullNodeHandshake(nodeConn, svtnID, nodePub, nodePriv)
	}()

	res := <-resCh
	nodeWG.Wait()

	if res.err == nil {
		t.Fatal("nodeIdentifyHandshake: want error (ErrKeyExpired), got nil")
	}
	if !errors.Is(res.err, admission.ErrKeyExpired) {
		t.Errorf("handshakeErr: want ErrKeyExpired (E-ADM-015), got %v", res.err)
	}

	_, ok := r.LookupInterface(svtnID, nodeAddr)
	if ok {
		t.Error("LookupInterface after ErrKeyExpired: want (0, false), got (_, true)")
	}
}

// ── AC-007: ErrNonceReplay ────────────────────────────────────────────────────

// TestNodeIdentifyHandshake_ErrNonceReplay_ConnectionClosed verifies that when
// the challenge nonce is already consumed, the connection is closed.
//
// The nonce is pre-consumed by calling AdmitNode directly before the handshake
// starts. We then capture the actual nonce from the Challenge frame and send
// a ChallengeResponse for that nonce, which is now already consumed.
//
// Traces to BC-2.01.009 Error Code E-ADM-008; AC-007.
func TestNodeIdentifyHandshake_ErrNonceReplay_ConnectionClosed(t *testing.T) {
	t.Parallel()

	_, routerPriv := mustGenKeyHandshake(t)
	nodePub, nodePriv := mustGenKeyHandshake(t)
	svtnID := mustSVTNHandshake(0x23)

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)
	r := routing.NewRouter(ks)

	routerConn, nodeConn := net.Pipe()
	t.Cleanup(func() { _ = routerConn.Close(); _ = nodeConn.Close() })

	h := newTestHandle(43)
	resCh := runRouterSide(routerConn, r, routerPriv, ks, h)

	var nodeWG sync.WaitGroup
	nodeWG.Add(1)
	go func() {
		defer nodeWG.Done()
		_ = nodeConn.SetDeadline(time.Now().Add(5 * time.Second))

		// Send NodeIdentify.
		niFrame := buildNodeIdentifyFrame(svtnID, nodePub)
		if _, err := nodeConn.Write(niFrame); err != nil {
			return
		}

		// Read Challenge — get the actual nonce the router generated.
		buf := make([]byte, 144)
		if _, err := io.ReadFull(nodeConn, buf); err != nil {
			return
		}
		var routerNonce [32]byte
		copy(routerNonce[:], buf[48:80])
		routerSig := buf[80:144]

		// Pre-consume this nonce by calling AdmitNode directly BEFORE sending
		// the ChallengeResponse back. After this call, routerNonce is in ks.nonces.
		validSig := ed25519.Sign(nodePriv, routerNonce[:])
		preChallenge := admission.Challenge{
			Nonce:     routerNonce,
			RouterSig: routerSig,
		}
		preResp := admission.ChallengeResponse{NonceSig: validSig}
		_ = admission.AdmitNode(preChallenge, preResp, nodePub, svtnID, ks)
		// routerNonce is now consumed in ks.nonces.

		// Now send ChallengeResponse with the same valid nonce_sig.
		// AdmitNode will find the nonce already consumed → ErrNonceReplay.
		crFrame := buildChallengeResponseFrame(svtnID, nodePriv, routerNonce)
		_, _ = nodeConn.Write(crFrame)
	}()

	res := <-resCh
	nodeWG.Wait()

	if res.err == nil {
		t.Fatal("nodeIdentifyHandshake: want error (ErrNonceReplay), got nil")
	}
	// The pre-consume records routerNonce in ks.nonces before the handshake's own
	// AdmitNode call. AdmitNode calls recordNonceUnlocked(routerNonce) which finds
	// the nonce already consumed and returns ErrNonceReplay — before reaching the
	// signature check. ErrNotAdmitted and ErrSignatureVerificationFailed cannot
	// occur on this path; the original disjunction was overly broad.
	if !errors.Is(res.err, admission.ErrNonceReplay) {
		t.Errorf("handshakeErr: want ErrNonceReplay (E-ADM-008), got %v", res.err)
	}
}

// ── AC-008: ErrSignatureVerificationFailed ────────────────────────────────────

// TestNodeIdentifyHandshake_ErrSignatureVerificationFailed_ConnectionClosed
// verifies that when NonceSig does not verify against the registered public key,
// the connection is closed with E-ADM-001.
//
// Traces to BC-2.01.009 Error Code E-ADM-001; AC-008.
func TestNodeIdentifyHandshake_ErrSignatureVerificationFailed_ConnectionClosed(t *testing.T) {
	t.Parallel()

	_, routerPriv := mustGenKeyHandshake(t)
	nodePub, _ := mustGenKeyHandshake(t)
	_, wrongNodePriv := mustGenKeyHandshake(t)
	svtnID := mustSVTNHandshake(0x24)

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)
	r := routing.NewRouter(ks)

	routerConn, nodeConn := net.Pipe()
	t.Cleanup(func() { _ = routerConn.Close(); _ = nodeConn.Close() })

	h := newTestHandle(44)
	resCh := runRouterSide(routerConn, r, routerPriv, ks, h)

	var nodeWG sync.WaitGroup
	nodeWG.Add(1)
	go func() {
		defer nodeWG.Done()
		_ = nodeConn.SetDeadline(time.Now().Add(5 * time.Second))

		// Send NodeIdentify (with nodePub — this is the registered key).
		niFrame := buildNodeIdentifyFrame(svtnID, nodePub)
		if _, err := nodeConn.Write(niFrame); err != nil {
			return
		}

		// Read Challenge.
		buf := make([]byte, 144)
		if _, err := io.ReadFull(nodeConn, buf); err != nil {
			return
		}
		var nonce [32]byte
		copy(nonce[:], buf[48:80])

		// Sign with WRONG key — not the registered nodePub.
		crFrame := buildChallengeResponseFrameWrongKey(svtnID, wrongNodePriv, nonce)
		_, _ = nodeConn.Write(crFrame)
	}()

	res := <-resCh
	nodeWG.Wait()

	if res.err == nil {
		t.Fatal("nodeIdentifyHandshake: want error (ErrSignatureVerificationFailed), got nil")
	}
	if !errors.Is(res.err, admission.ErrSignatureVerificationFailed) {
		t.Errorf("handshakeErr: want ErrSignatureVerificationFailed (E-ADM-001), got %v", res.err)
	}

	nodeAddr := nodeAddrHandshake(svtnID, nodePub)
	_, ok := r.LookupInterface(svtnID, nodeAddr)
	if ok {
		t.Error("LookupInterface after bad sig: want (0, false), got (_, true)")
	}
}

// ── AC-009: Handshake timeout ─────────────────────────────────────────────────

// TestNodeIdentifyHandshake_Timeout_E_ADM_022 verifies that when the handshake
// deadline fires, the connection is closed (E-ADM-022).
//
// nodeIdentifyHandshakeTimeout is overridden to 50ms so the test completes in
// well under a second. The node side sends nothing; the handshake's own
// conn.SetDeadline(50ms) fires deterministically.
//
// NOT t.Parallel(): overrides the package-level nodeIdentifyHandshakeTimeout
// var; must run serially to avoid data races with other tests.
//
// Traces to BC-2.01.009 Precondition 4, EC-002, E-ADM-022; AC-009.
func TestNodeIdentifyHandshake_Timeout_E_ADM_022(t *testing.T) {
	// Override the production 10s deadline to 50ms so the test runs fast.
	// Restore the original value on exit via t.Cleanup.
	orig := nodeIdentifyHandshakeTimeout
	nodeIdentifyHandshakeTimeout = 50 * time.Millisecond
	t.Cleanup(func() { nodeIdentifyHandshakeTimeout = orig })

	_, routerPriv := mustGenKeyHandshake(t)

	ks := admission.NewAdmittedKeySet()
	r := routing.NewRouter(ks)

	routerConn, nodeConn := net.Pipe()
	t.Cleanup(func() { _ = routerConn.Close(); _ = nodeConn.Close() })

	h := newTestHandle(45)
	resCh := runRouterSide(routerConn, r, routerPriv, ks, h)

	// Node side: do nothing (no frames sent). The handshake's own 50ms deadline
	// fires; the router closes its end; the node-side read returns EOF.
	go func() {
		_ = nodeConn.SetDeadline(time.Now().Add(3 * time.Second))
		_, _ = io.ReadFull(nodeConn, make([]byte, 1)) // unblocks when router closes conn
	}()

	res := <-resCh
	if res.err == nil {
		t.Fatal("nodeIdentifyHandshake: want timeout error (E-ADM-022), got nil")
	}
	// The error must be a real deadline/timeout error, NOT the stub "unimplemented" sentinel.
	// A deadline-exceeded error contains "deadline" or "timeout" in its message;
	// the stub returns "unimplemented: nodeIdentifyHandshake".
	if strings.Contains(res.err.Error(), "unimplemented") {
		t.Errorf("nodeIdentifyHandshake: got stub error %q; want real deadline-exceeded error (E-ADM-022)", res.err)
	}
}

// ── AC-011: Duplicate NodeIdentify (E-ADM-023) ────────────────────────────────

// TestNodeIdentifyHandshake_DuplicateNodeIdentify_E_ADM_023 verifies that a
// second NODE_IDENTIFY control frame (control_type=0x04) arriving on an
// already-admitted connection closes that connection (BC-2.01.009 Invariant 7;
// E-ADM-023; rulings §17 Option B).
//
// Discriminating-property: the test drives the production route closure inside
// runRouter (not a test-local copy). The §17 fix makes case 0x04 call
// conn.Close() directly; without it ServeConn drops the error and the
// connection stays open. Only that one change toggles this test.
//
// Test structure (mirrors TestRouter_CtlFrame_ShortPayload_NoConnClose):
//  1. Start runRouter with an admitted node key loaded from a snapshot file.
//  2. Dial a TCP node connection, complete the NODE_IDENTIFY handshake.
//  3. Wait for nodeConnRegistered to confirm onAccept completed and ServeConn
//     is reading the per-conn data-plane loop.
//  4. Send a well-formed duplicate NODE_IDENTIFY ctl frame on the same conn.
//  5. ASSERT: the router closes the connection — a read on the node side
//     returns a non-timeout net error or EOF within a bounded deadline.
//
// Red Gate: currently FAILS because mgmt_wire.go case 0x04 returns an error
// but netingress.ServeConn drops it (continue); the connection stays open;
// the deadline read returns a timeout error → the test fatal-logs
// "duplicate NODE_IDENTIFY did not close the connection".
//
// Traces to BC-2.01.009 Invariant 7, EC-004, E-ADM-023; AC-011.
//
// NOT t.Parallel(): binds ephemeral TCP + filesystem socket, mutates the
// package-level nodeConnHook test hook (Q-AC002 test-isolation requirement
// shared with router_drain_wire_test.go).
func TestNodeIdentifyHandshake_DuplicateNodeIdentify_E_ADM_023(t *testing.T) {
	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}

	// Pre-load an admission snapshot so runRouter admits the test node key.
	info := makeAdmittedNode(t, cfg)

	// Install a channel-backed nodeConnHook so we know when onAccept finishes
	// and ServeConn has started reading the per-conn data-plane loop.
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

	// Dial and complete the NODE_IDENTIFY handshake; this advances the connection
	// through onAccept (handshake → sendMap.Store → nodeConnRegistered hook).
	nodeConn := dialNodeAdmitted(t, cfg, info)

	// Wait until onAccept has stored the node in sendMap (nodeConnRegistered event)
	// so ServeConn is reading the per-conn loop before we send the duplicate frame.
	awaitNodeConnEvent(t, events, nodeConnRegistered, 4*time.Second)

	// Build the duplicate NODE_IDENTIFY payload: the same wire layout as a
	// NodeIdentify message (control_type=0x04, version=0x01, msg_kind=0x01,
	// reserved=0x00, pubkey[32]=zeros — exact field values are irrelevant; the
	// router closes on control_type=0x04 regardless of the payload contents).
	dupPayload := make([]byte, nodeIdentifyPayloadSize)
	dupPayload[0] = nodeIdentifyControlType // 0x04
	dupPayload[1] = frame.VersionByte       // 0x01
	dupPayload[2] = msgKindNodeIdentify     // 0x01
	// dupPayload[3] = 0x00 already — reserved byte
	// dupPayload[4:36] = zero bytes — pubkey (irrelevant; router does not decode past control_type)

	writeRawFrame(t, nodeConn, frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeCtl,
		SVTNID:    info.svtnID,
	}, dupPayload)

	// ASSERT: the router closes the connection — a read returns a non-timeout
	// error (EOF or connection reset) within the deadline.
	//
	// Correct (§17 Option B): case 0x04 calls conn.Close() directly;
	// ServeConn's next ReadFrame returns a net error; nodeConn read returns
	// EOF/reset within the deadline → test PASSES.
	//
	// Broken (current): case 0x04 returns an error but ServeConn drops it
	// (continue); the connection stays open; the deadline read returns a
	// timeout error → test FAILS.
	const closedDeadline = 2 * time.Second
	_ = nodeConn.SetReadDeadline(time.Now().Add(closedDeadline))
	rbuf := make([]byte, 1)
	_, readErr := nodeConn.Read(rbuf)
	_ = nodeConn.SetReadDeadline(time.Time{})

	if readErr == nil {
		t.Fatal("AC-011: duplicate NODE_IDENTIFY did not close the connection: " +
			"read returned nil error (unexpected data from router)")
	}
	var netErr net.Error
	if errors.As(readErr, &netErr) && netErr.Timeout() {
		// The read timed out — meaning the connection stayed open.
		// This is the current (broken) behavior: ServeConn drops the route error.
		t.Fatalf("AC-011: duplicate NODE_IDENTIFY did not close the connection within %v "+
			"(ServeConn must not drop the E-ADM-023 error; case 0x04 must call conn.Close() "+
			"directly per rulings §17 Option B)", closedDeadline)
	}
	// Any non-timeout error (EOF, connection reset, closed pipe) means the
	// router closed the connection — which is the required behavior.

	// AC-011 PC-2: the router MUST emit a WARN log containing "E-ADM-023"
	// (mgmt_wire.go:636: `routerLogger.Log("node_identify: duplicate NodeIdentify
	// on established connection (E-ADM-023)")`).
	// Discriminating: deleting that Log call (while keeping conn.Close()) leaves
	// the connection-close assertion above passing but fails this one — so this
	// is the sole discriminating guard for the WARN log postcondition.
	if !scanForLine(&buf, "E-ADM-023", 2*time.Second) {
		t.Errorf("AC-011 PC-2: daemon log does not contain \"E-ADM-023\" within 2s "+
			"(mgmt_wire.go:636 routerLogger.Log must emit the code); log:\n%s", buf.String())
	}
}

// ── AC-012: Cleanup func calls UnbindInterface ────────────────────────────────

// TestNodeIdentifyHandshake_CleanupFunc_UnbindInterface_Called is a real
// integration test that drives the production runRouter daemon, establishes an
// admitted connection through the real onAccept path, closes the connection
// from the client side, and asserts that the identity binding is removed once
// the cleanup closure fires.
//
// Discriminating property: this test is the ONLY coverage for
// mgmt_wire.go:778 (router.UnbindInterface in the cleanup closure). Disabling
// that line leaves the binding in identityIfaceMap indefinitely, and
// LookupInterface after nodeConnRemoved would still return (ifaceID, true) —
// causing the post-cleanup assertion below to fail. The test is therefore
// discriminating: removing mgmt_wire.go:778 flips it from PASS to FAIL.
//
// The router reference is captured through the nodeIdentifyHandshakeFn wrapper:
// the wrapper calls the real handshake unchanged but stores the *routing.Router
// pointer the daemon passes to it, making LookupInterface available without
// any production-code change. The wrapper is restored via t.Cleanup.
//
// NOT t.Parallel(): overrides the package-level nodeIdentifyHandshakeFn and
// nodeConnHook vars; must run serially to avoid data races with other tests
// that rely on the real handshake (same isolation rationale as the timeout and
// E-ADM-008 tests).
//
// Traces to BC-2.01.010 PC-8; AC-012.
func TestNodeIdentifyHandshake_CleanupFunc_UnbindInterface_Called(t *testing.T) {
	// Capture the *routing.Router the daemon passes to the handshake function.
	// The wrapper calls the real handshake unchanged; the only side effect is
	// storing the router pointer for post-cleanup LookupInterface assertions.
	// Synchronization: the wrapper runs on the onAccept goroutine; the test
	// goroutine reads routerCapture after awaitNodeConnEvent(nodeConnRegistered),
	// which happens-after the wrapper has stored the pointer (the hook fires
	// after sendMap.Store, which follows the handshake return).
	var routerCapture atomic.Pointer[routing.Router]
	var svtnCapture atomic.Value     // stores [16]byte
	var nodeAddrCapture atomic.Value // stores [8]byte

	origFn := nodeIdentifyHandshakeFn
	nodeIdentifyHandshakeFn = func(
		conn net.Conn,
		r *routing.Router,
		priv ed25519.PrivateKey,
		ks *admission.AdmittedKeySet,
		h netingress.NodeHandle,
	) ([16]byte, [8]byte, error) {
		svtnID, nodeAddr, err := origFn(conn, r, priv, ks, h)
		if err == nil {
			routerCapture.Store(r)
			svtnCapture.Store(svtnID)
			nodeAddrCapture.Store(nodeAddr)
		}
		return svtnID, nodeAddr, err
	}
	t.Cleanup(func() { nodeIdentifyHandshakeFn = origFn })

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}

	// Pre-load an admission snapshot so runRouter admits the test node key.
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

	// Dial and complete the NODE_IDENTIFY handshake. After this returns,
	// the wrapper has stored the router pointer in routerCapture.
	nodeConn := dialNodeAdmitted(t, cfg, info)

	// Wait for onAccept to complete (handshake → sendMap.Store →
	// nodeConnRegistered hook). After this point routerCapture is populated
	// and the binding exists in identityIfaceMap.
	registered := awaitNodeConnEvent(t, events, nodeConnRegistered, 4*time.Second)

	r := routerCapture.Load()
	if r == nil {
		t.Fatal("AC-012: routerCapture is nil after nodeConnRegistered — wrapper did not fire")
	}
	svtnID, _ := svtnCapture.Load().([16]byte)
	nodeAddr, _ := nodeAddrCapture.Load().([8]byte)

	// ASSERT: binding is present while the connection is live.
	got, ok := r.LookupInterface(svtnID, nodeAddr)
	if !ok {
		t.Fatalf("AC-012: LookupInterface before conn close: want (ifaceID, true), got (0, false); "+
			"registered ifaceID=%d svtnID=%x nodeAddr=%x", registered.ifaceID, svtnID, nodeAddr)
	}
	if got != registered.ifaceID {
		t.Errorf("AC-012: LookupInterface before conn close: want ifaceID=%d, got %d",
			registered.ifaceID, got)
	}

	// Close from the client side — triggers the daemon's per-conn ServeConn to
	// return, which invokes the onAccept cleanup func:
	//   sendMap.Delete(h.IfaceID)
	//   router.UnbindInterface(svtnID, nodeAddr, h.IfaceID)  ← line 778 under test
	//   nodeConnHook(nodeConnRemoved, h.IfaceID)
	_ = nodeConn.Close()

	// Wait for the cleanup closure to complete (bounded deadline).
	// nodeConnRemoved fires synchronously at line 780, AFTER UnbindInterface
	// at line 778 — so when this returns, the binding is already gone.
	awaitNodeConnEvent(t, events, nodeConnRemoved, 2*time.Second)

	// ASSERT: binding is removed after cleanup.
	// Discriminating property: if mgmt_wire.go:778 is disabled, this assertion
	// FAILS because identityIfaceMap still holds the entry.
	got, ok = r.LookupInterface(svtnID, nodeAddr)
	if ok {
		t.Errorf("AC-012: LookupInterface after conn close: want (0, false), got (%d, true); "+
			"router.UnbindInterface was NOT called by the cleanup func (mgmt_wire.go:778 missing)",
			got)
	}
	if got != 0 {
		t.Errorf("AC-012: LookupInterface after conn close: want InterfaceID=0, got %d", got)
	}
}

// ── AC-007 PC2: E-ADM-008 nonce-replay WARN log ───────────────────────────────

// TestNodeIdentifyHandshake_NonceReplay_E_ADM_008_Logged verifies that when
// the handshake returns admission.ErrNonceReplay, onAccept's classification
// switch emits a WARN log containing "E-ADM-008" and the SVTN ID in hex.
//
// The nonce-replay error cannot be induced over black-box TCP (the router
// never reuses a nonce), so this test injects a deterministic stub via the
// nodeIdentifyHandshakeFn package var added in commit 55c24af. The stub
// returns a known non-zero SVTN ID alongside ErrNonceReplay, so the test can
// assert both the error code and the SVTN ID appear in the daemon log.
//
// Discriminating property: changing the E-ADM-008 arm's code string, removing
// the "svtn=%x" field, or deleting the arm (falling to default) all flip this
// test from PASS to FAIL.
//
// NOT t.Parallel(): overrides the package-level nodeIdentifyHandshakeFn var;
// must run serially to avoid data races with tests relying on the real
// handshake (same isolation as TestNodeIdentifyHandshake_Timeout_E_ADM_022).
//
// Traces to BC-2.01.009 error-path classification; AC-007 PC2.
func TestNodeIdentifyHandshake_NonceReplay_E_ADM_008_Logged(t *testing.T) {
	// knownSvtnID is a non-zero SVTN ID whose hex representation we assert
	// appears in the log line so the test is discriminating on the svtn= field.
	var knownSvtnID [16]byte
	knownSvtnID[0] = 0xDE
	knownSvtnID[1] = 0xAD
	knownSvtnID[2] = 0xBE
	knownSvtnID[3] = 0xEF
	knownSvtnIDHex := fmt.Sprintf("%x", knownSvtnID)

	orig := nodeIdentifyHandshakeFn
	nodeIdentifyHandshakeFn = func(
		conn net.Conn,
		_ *routing.Router,
		_ ed25519.PrivateKey,
		_ *admission.AdmittedKeySet,
		_ netingress.NodeHandle,
	) ([16]byte, [8]byte, error) {
		// Close the connection so the daemon goroutine doesn't leak waiting
		// for I/O — mirrors what the real handshake does on error (it closes
		// the conn before returning the non-nil error).
		_ = conn.Close()
		return knownSvtnID, [8]byte{}, admission.ErrNonceReplay
	}
	t.Cleanup(func() { nodeIdentifyHandshakeFn = orig })

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}

	// No admitted-node snapshot needed: the stub returns an error before any
	// admission check, so runRouter starts with an empty key set.
	var buf syncBuffer
	errCh, cancel := startRunRouterWithConfig(t, cfg, &buf)
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(4 * time.Second):
		}
	})

	// Dial a connection — this triggers onAccept, which calls the stub, which
	// returns ErrNonceReplay, which causes the E-ADM-008 arm to log.
	conn, err := net.DialTimeout("tcp", cfg.ListenAddr, 2*time.Second)
	if err != nil {
		t.Fatalf("AC-007: dial %s: %v", cfg.ListenAddr, err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	// ASSERT: the daemon log contains "E-ADM-008" within a bounded deadline.
	// Discriminating property: removing/changing the code in mgmt_wire.go:712-713
	// causes this assertion to fail.
	if !scanForLine(&buf, "E-ADM-008", 2*time.Second) {
		t.Errorf("AC-007 PC2: daemon log does not contain \"E-ADM-008\" within 2s; log:\n%s",
			buf.String())
	}

	// ASSERT: the SVTN ID hex also appears in the same log buffer.
	// Discriminating property: removing "svtn=%x" from mgmt_wire.go:713 causes
	// this assertion to fail.
	if !scanForLine(&buf, knownSvtnIDHex, 2*time.Second) {
		t.Errorf("AC-007 PC2: daemon log does not contain SVTN ID %q within 2s; log:\n%s",
			knownSvtnIDHex, buf.String())
	}
}
