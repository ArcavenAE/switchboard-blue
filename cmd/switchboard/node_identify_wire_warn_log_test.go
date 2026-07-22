// node_identify_wire_warn_log_test.go — daemon-level WARN-log tests for
// AC-002 PC-4 (malformed NodeIdentify → WARN) and AC-003 PC-2
// (zero SVTN ID → WARN: "node_identify: zero SVTN ID rejected").
//
// The existing direct-call tests in node_identify_wire_test.go verify the
// returned error sentinels but CANNOT observe the daemon-level WARN log
// because they call nodeIdentifyHandshake directly, bypassing the
// onAccept classification switch in mgmt_wire.go — specifically the
// default arm that handles unclassified errors such as malformed frames
// and zero-SVTN rejections.
//
// Strategy: identical to the existing AC-004..AC-009 daemon-level log
// companions in node_identify_wire_log_test.go — drive the full
// runRouter → onAccept → nodeIdentifyHandshake path via
// startRunRouterWithConfig, use scanForLine to assert the WARN log.
//
// NOT t.Parallel() on any test: daemon-level tests bind ephemeral TCP
// ports and may adjust package-level vars; serial execution is required
// (Q-AC002; F-DW-SP4-005).
//
// Traces to AC-002 PC-4 (malformed NodeIdentify), AC-003 PC-2
// (zero SVTN ID rejected WARN log).
package main

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/config"
)

// ── AC-003 PC-2: zero SVTN ID → WARN log ─────────────────────────────────────

// TestNodeIdentifyHandshake_ZeroSVTNID_LogsWarn is a daemon-level companion
// to TestNodeIdentifyHandshake_ZeroSVTNID_Rejected.
//
// It drives a NodeIdentify frame with an all-zero SVTN ID through the real
// runRouter → onAccept path and asserts that the WARN log contains the
// literal "node_identify: zero SVTN ID rejected" (AC-003 PC-2 postcondition).
//
// onAccept's classification switch does not have a dedicated case for the
// zero-SVTN error (it is not an admission sentinel), so the error falls to
// the default arm of onAccept's classification switch in mgmt_wire.go:
//
//	default:
//	    routerLogger.Log(fmt.Sprintf("runRouter: NODE_IDENTIFY handshake failed: %v", hsErr))
//
// The resulting log line contains the exact error string returned by
// nodeIdentifyHandshake: "node_identify: zero SVTN ID rejected".
//
// Discriminating property: removing the zero-SVTN guard from
// node_identify_wire.go prevents the "node_identify: zero SVTN ID rejected"
// substring from reaching the log — this test is the sole daemon-level
// discriminating guard for AC-003 PC-2's log postcondition.
//
// NOT t.Parallel(): daemon-level, binds ephemeral TCP.
//
// Traces to BC-2.01.009 Precondition 5; AC-003 PC-2.
func TestNodeIdentifyHandshake_ZeroSVTNID_LogsWarn(t *testing.T) {
	nodePub, _ := mustGenKeyHandshake(t)
	var zeroSVTNID [16]byte // all-zero

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
		// No AdmissionStateFile: keyset is empty; zero-SVTN rejection fires
		// before any admission lookup, so no snapshot is needed.
	}

	var buf syncBuffer
	errCh, cancel := startRunRouterWithConfig(t, cfg, &buf)
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(4 * time.Second):
		}
	})

	// Dial and send a NodeIdentify frame with a zero SVTN ID. The router
	// rejects immediately and closes the connection.
	conn, err := net.Dial("tcp", cfg.ListenAddr)
	if err != nil {
		t.Fatalf("dial %s: %v", cfg.ListenAddr, err)
	}
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	niFrame := buildNodeIdentifyFrame(zeroSVTNID, nodePub)
	_, _ = conn.Write(niFrame)
	_, _ = io.Copy(io.Discard, conn) // drain until daemon closes conn
	_ = conn.Close()

	// Primary assertion (AC-003 PC-2): the daemon log must contain the exact
	// literal "node_identify: zero SVTN ID rejected".
	//
	// Discriminating: renaming that literal in node_identify_wire.go fails this
	// assertion while TestNodeIdentifyHandshake_ZeroSVTNID_Rejected (direct-call)
	// remains green — so this companion is the sole daemon-level guard for
	// the WARN log postcondition.
	const wantLiteral = "node_identify: zero SVTN ID rejected"
	if !scanForLine(&buf, wantLiteral, 2*time.Second) {
		t.Errorf("AC-003 PC-2: want WARN log containing %q within 2s; "+
			"lines so far:\n%s", wantLiteral, buf.String())
	}
}

// ── AC-002 PC-4: malformed NodeIdentify → WARN log ───────────────────────────

// TestNodeIdentifyHandshake_MalformedNodeIdentify_LogsWarn is a daemon-level
// companion to TestNodeIdentifyHandshake_MalformedNodeIdentify_WrongPayloadLen.
//
// It drives a NodeIdentify frame with the wrong payload_len (20 instead of
// 36) through the real runRouter → onAccept path and asserts that the WARN
// log is emitted containing the error text (AC-002 PC-4 postcondition).
//
// Malformed-frame errors are not admission sentinels, so they fall to the
// default arm in onAccept's switch (mgmt_wire.go):
//
//	default:
//	    routerLogger.Log(fmt.Sprintf("runRouter: NODE_IDENTIFY handshake failed: %v", hsErr))
//
// The log line therefore contains "malformed NodeIdentify" — the literal
// prefix that all NodeIdentify decode-error paths in node_identify_wire.go
// share (AC-002 PC-4).
//
// Discriminating property: removing the malformed-payload-len guard from
// node_identify_wire.go suppresses the "malformed NodeIdentify" substring in
// the log. This test is the sole daemon-level discriminating guard for the
// WARN log postcondition — the existing direct-call test in
// node_identify_wire_test.go asserts the returned error but cannot observe
// the daemon log.
//
// NOT t.Parallel(): daemon-level, binds ephemeral TCP.
//
// Traces to BC-2.01.009 Invariant 5; AC-002 PC-4.
func TestNodeIdentifyHandshake_MalformedNodeIdentify_LogsWarn(t *testing.T) {
	var svtnID [16]byte
	svtnID[0] = 0x58 // non-zero, distinct byte

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
		// No AdmissionStateFile: the malformed-frame rejection fires before any
		// admission lookup, so no key snapshot is needed.
	}

	var buf syncBuffer
	errCh, cancel := startRunRouterWithConfig(t, cfg, &buf)
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(4 * time.Second):
		}
	})

	// Dial and send a NodeIdentify outer frame with payload_len=20 (wrong;
	// the correct value is 36). encodeCtlHeaderRaw writes FrameTypeCtl (0x03)
	// so the outer frame-type check passes; the payload-length guard
	// (node_identify_wire.go) fires first and returns
	// "malformed NodeIdentify: payload_len…".
	conn, err := net.Dial("tcp", cfg.ListenAddr)
	if err != nil {
		t.Fatalf("dial %s: %v", cfg.ListenAddr, err)
	}
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	const wrongPayloadLen = 20
	hdr := encodeCtlHeaderRaw(wrongPayloadLen, svtnID)
	payload := make([]byte, wrongPayloadLen)
	payload[0] = 0x04 // control_type (would be correct if payload_len were right)
	payload[1] = 0x01 // version
	payload[2] = 0x01 // msg_kind
	payload[3] = 0x00 // reserved
	_, _ = conn.Write(append(hdr, payload...))
	_, _ = io.Copy(io.Discard, conn) // drain until daemon closes conn
	_ = conn.Close()

	// Primary assertion (AC-002 PC-4): the daemon log must contain the
	// "malformed NodeIdentify" literal emitted by node_identify_wire.go's
	// decode guards and propagated to the default arm's Sprintf.
	//
	// Discriminating: removing the payload-len decode guard suppresses this
	// substring from the log line; the direct-call test's error assertion
	// would also fail, but this companion is the sole daemon-level guard for
	// the WARN emission path (the onAccept default arm).
	const wantSubstr = "malformed NodeIdentify"
	if !scanForLine(&buf, wantSubstr, 2*time.Second) {
		t.Errorf("AC-002 PC-4: want WARN log containing %q within 2s; "+
			"lines so far:\n%s", wantSubstr, buf.String())
	}
}
