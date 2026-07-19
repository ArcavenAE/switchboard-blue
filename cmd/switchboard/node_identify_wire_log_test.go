// node_identify_wire_log_test.go — daemon-level companion tests for
// AC-004..AC-009 (S-BL.NODE-IDENTIFY-WIRE) that assert code-bearing WARN logs.
//
// The existing direct-call tests in node_identify_wire_test.go verify the
// returned error sentinels but cannot observe E-ADM WARN logs because those
// are emitted in the onAccept closure inside runRouter — one level above the
// nodeIdentifyHandshake call site. These daemon-level companions drive the
// full production path (runRouter → onAccept → nodeIdentifyHandshake →
// E-ADM log) and assert the log content.
//
// Strategy A per dispatch (each test drives real runRouter via
// startRunRouterWithConfig and scans buf via scanForLine).
//
// AC-007 (ErrNonceReplay / E-ADM-008): no daemon-level log test is included.
// The nonce pre-consume technique used by the direct-call test requires direct
// access to the daemon's internal ks, which is not exposed. Providing it at
// daemon level would require either a test seam (production change) or
// duplicating the onAccept emission switch in test code. The sentinel assertion
// in TestNodeIdentifyHandshake_ErrNonceReplay_ConnectionClosed (tightened in
// this file to require ErrNonceReplay) is the discriminating guard for that path.
//
// NOT t.Parallel() on any test: several tests override nodeIdentifyHandshakeTimeout
// (a package-level mutable var) and daemon-level tests bind ephemeral TCP ports.
// Serial execution is required for test isolation (Q-AC002; F-DW-SP4-005).
//
// Traces to BC-2.01.009 Error Codes E-ADM-001/003/005/015/022; AC-004..009.
package main

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/config"
)

// ── snapshot helpers for failure scenarios ────────────────────────────────────

// snapKeyLog mirrors the JSON fields from snapshotKey (admission_sync_snapshot.go)
// for constructing test snapshots inline. Test-only; not exported.
type snapKeyLog struct {
	PubKey  string `json:"pubkey"`
	Role    string `json:"role"`
	Revoked bool   `json:"revoked"`
	Expiry  string `json:"expiry,omitempty"`
}

// snapSVTNLog mirrors snapshotSVTN.
type snapSVTNLog struct {
	SVTNID string       `json:"svtn_id"`
	Keys   []snapKeyLog `json:"keys"`
}

// snapFileLog mirrors snapshotFile.
type snapFileLog struct {
	SchemaVersion int           `json:"schema_version"`
	Timestamp     string        `json:"timestamp"`
	SVTNs         []snapSVTNLog `json:"svtns"`
}

// writeAdmissionSnapshotLog writes a single-key admission snapshot file for
// (svtnID, pub) with the given key attributes and sets cfg.AdmissionStateFile.
// Must be called BEFORE startRunRouterWithConfig.
func writeAdmissionSnapshotLog(
	t *testing.T,
	cfg *config.Config,
	svtnID [16]byte,
	pub []byte, // ed25519.PublicKey is []byte
	revoked bool,
	expiry string, // RFC3339 or "" for no expiry
) {
	t.Helper()
	snap := snapFileLog{
		SchemaVersion: 1,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		SVTNs: []snapSVTNLog{{
			SVTNID: hex.EncodeToString(svtnID[:]),
			Keys: []snapKeyLog{{
				PubKey:  base64.RawURLEncoding.EncodeToString(pub),
				Role:    "access",
				Revoked: revoked,
				Expiry:  expiry,
			}},
		}},
	}
	data, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("writeAdmissionSnapshotLog: json.Marshal: %v", err)
	}
	f, err := os.CreateTemp(t.TempDir(), "admission-*.json")
	if err != nil {
		t.Fatalf("writeAdmissionSnapshotLog: CreateTemp: %v", err)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		t.Fatalf("writeAdmissionSnapshotLog: write: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("writeAdmissionSnapshotLog: close: %v", err)
	}
	cfg.AdmissionStateFile = f.Name()
}

// ── AC-004: E-ADM-003 (ErrNotAdmitted) ───────────────────────────────────────

// TestNodeIdentifyHandshake_ErrNotAdmitted_LogsEADM003 is a daemon-level
// companion to TestNodeIdentifyHandshake_ErrNotAdmitted_ConnectionClosed.
// It asserts that the production onAccept closure emits an E-ADM-003 WARN
// log containing the SVTN ID when the node's key is not registered.
//
// Discriminating property: removing the `case errors.Is(hsErr, admission.ErrNotAdmitted)`
// arm from mgmt_wire.go:onAccept would prevent this log line, failing this test.
// The existing sentinel test would still pass (it checks the returned error,
// not the log) — this companion is the sole discriminating guard for the log.
//
// NOT t.Parallel(): daemon-level, binds ephemeral TCP.
//
// Traces to BC-2.01.009 Error Code E-ADM-003; AC-004.
func TestNodeIdentifyHandshake_ErrNotAdmitted_LogsEADM003(t *testing.T) {
	pub, priv := mustGenKeyHandshake(t)
	svtnID := mustSVTNHandshake(0x50)

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
		// AdmissionStateFile empty → daemon starts with empty keyset;
		// the connecting node key is not admitted → E-ADM-003.
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

	// Dial and complete the full handshake. AdmitNode will return ErrNotAdmitted
	// because the keyset is empty; onAccept logs E-ADM-003.
	conn, err := net.Dial("tcp", cfg.ListenAddr)
	if err != nil {
		t.Fatalf("dial %s: %v", cfg.ListenAddr, err)
	}
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	doFullNodeHandshake(conn, svtnID, pub, priv)
	_, _ = io.Copy(io.Discard, conn) // drain until daemon closes conn
	_ = conn.Close()

	// Primary assertion (AC-004 PC2): E-ADM-003 log with SVTN ID.
	if !scanForLine(&buf, "E-ADM-003", 2*time.Second) {
		t.Errorf("AC-004: want E-ADM-003 in daemon log; lines so far:\n%s", buf.String())
	}
	svtnHex := fmt.Sprintf("%x", svtnID)
	if !scanForLine(&buf, svtnHex, 2*time.Second) {
		t.Errorf("AC-004: want SVTN ID %s in daemon log (E-ADM-003); lines:\n%s", svtnHex, buf.String())
	}
}

// ── AC-005: E-ADM-005 (ErrKeyRevoked) ────────────────────────────────────────

// TestNodeIdentifyHandshake_ErrKeyRevoked_LogsEADM005 is a daemon-level
// companion to TestNodeIdentifyHandshake_ErrKeyRevoked_ConnectionClosed.
// It asserts the production onAccept closure emits an E-ADM-005 WARN log
// containing the SVTN ID when the node's key is revoked.
//
// Discriminating property: removing the `case errors.Is(hsErr, admission.ErrKeyRevoked)`
// arm from mgmt_wire.go:onAccept would prevent this log line, failing this test.
//
// NOT t.Parallel(): daemon-level, binds ephemeral TCP.
//
// Traces to BC-2.01.009 Error Code E-ADM-005; AC-005.
func TestNodeIdentifyHandshake_ErrKeyRevoked_LogsEADM005(t *testing.T) {
	pub, priv := mustGenKeyHandshake(t)
	svtnID := mustSVTNHandshake(0x51)

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}
	writeAdmissionSnapshotLog(t, cfg, svtnID, []byte(pub), true /* revoked */, "")

	var buf syncBuffer
	errCh, cancel := startRunRouterWithConfig(t, cfg, &buf)
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(4 * time.Second):
		}
	})

	conn, err := net.Dial("tcp", cfg.ListenAddr)
	if err != nil {
		t.Fatalf("dial %s: %v", cfg.ListenAddr, err)
	}
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	doFullNodeHandshake(conn, svtnID, pub, priv)
	_, _ = io.Copy(io.Discard, conn)
	_ = conn.Close()

	if !scanForLine(&buf, "E-ADM-005", 2*time.Second) {
		t.Errorf("AC-005: want E-ADM-005 in daemon log; lines so far:\n%s", buf.String())
	}
	svtnHex := fmt.Sprintf("%x", svtnID)
	if !scanForLine(&buf, svtnHex, 2*time.Second) {
		t.Errorf("AC-005: want SVTN ID %s in daemon log (E-ADM-005); lines:\n%s", svtnHex, buf.String())
	}
}

// ── AC-006: E-ADM-015 (ErrKeyExpired) ────────────────────────────────────────

// TestNodeIdentifyHandshake_ErrKeyExpired_LogsEADM015 is a daemon-level
// companion to TestNodeIdentifyHandshake_ErrKeyExpired_ConnectionClosed.
// It asserts the production onAccept closure emits an E-ADM-015 WARN log
// containing the SVTN ID when the node's key has expired.
//
// Discriminating property: removing the `case errors.Is(hsErr, admission.ErrKeyExpired)`
// arm from mgmt_wire.go:onAccept would prevent this log line, failing this test.
//
// NOT t.Parallel(): daemon-level, binds ephemeral TCP.
//
// Traces to BC-2.01.009 Error Code E-ADM-015; AC-006.
func TestNodeIdentifyHandshake_ErrKeyExpired_LogsEADM015(t *testing.T) {
	pub, priv := mustGenKeyHandshake(t)
	svtnID := mustSVTNHandshake(0x52)

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}
	// Set expiry one second in the past.
	pastExpiry := time.Now().UTC().Add(-time.Second).Format(time.RFC3339)
	writeAdmissionSnapshotLog(t, cfg, svtnID, []byte(pub), false, pastExpiry)

	var buf syncBuffer
	errCh, cancel := startRunRouterWithConfig(t, cfg, &buf)
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(4 * time.Second):
		}
	})

	conn, err := net.Dial("tcp", cfg.ListenAddr)
	if err != nil {
		t.Fatalf("dial %s: %v", cfg.ListenAddr, err)
	}
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	doFullNodeHandshake(conn, svtnID, pub, priv)
	_, _ = io.Copy(io.Discard, conn)
	_ = conn.Close()

	if !scanForLine(&buf, "E-ADM-015", 2*time.Second) {
		t.Errorf("AC-006: want E-ADM-015 in daemon log; lines so far:\n%s", buf.String())
	}
	svtnHex := fmt.Sprintf("%x", svtnID)
	if !scanForLine(&buf, svtnHex, 2*time.Second) {
		t.Errorf("AC-006: want SVTN ID %s in daemon log (E-ADM-015); lines:\n%s", svtnHex, buf.String())
	}
}

// ── AC-008: E-ADM-001 (ErrSignatureVerificationFailed) ───────────────────────

// TestNodeIdentifyHandshake_ErrSigVerifyFailed_LogsEADM001 is a daemon-level
// companion to TestNodeIdentifyHandshake_ErrSignatureVerificationFailed_ConnectionClosed.
// It asserts the production onAccept closure emits an E-ADM-001 WARN log
// containing the SVTN ID when the node signs the challenge with the wrong key.
//
// Discriminating property: removing the `case errors.Is(hsErr, admission.ErrSignatureVerificationFailed)`
// arm from mgmt_wire.go:onAccept would prevent this log line, failing this test.
//
// NOT t.Parallel(): daemon-level, binds ephemeral TCP.
//
// Traces to BC-2.01.009 Error Code E-ADM-001; AC-008.
func TestNodeIdentifyHandshake_ErrSigVerifyFailed_LogsEADM001(t *testing.T) {
	pub, _ := mustGenKeyHandshake(t)
	_, wrongPriv := mustGenKeyHandshake(t)
	svtnID := mustSVTNHandshake(0x54)

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}
	writeAdmissionSnapshotLog(t, cfg, svtnID, []byte(pub), false, "")

	var buf syncBuffer
	errCh, cancel := startRunRouterWithConfig(t, cfg, &buf)
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(4 * time.Second):
		}
	})

	// Node presents the registered pub key in NodeIdentify but signs the
	// challenge nonce with a DIFFERENT private key → ErrSignatureVerificationFailed.
	conn, err := net.Dial("tcp", cfg.ListenAddr)
	if err != nil {
		t.Fatalf("dial %s: %v", cfg.ListenAddr, err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	niFrame := buildNodeIdentifyFrame(svtnID, pub)
	if _, writeErr := conn.Write(niFrame); writeErr != nil {
		t.Fatalf("write NodeIdentify: %v", writeErr)
	}
	challengeBuf := make([]byte, 144)
	if _, readErr := io.ReadFull(conn, challengeBuf); readErr != nil {
		t.Fatalf("read Challenge: %v", readErr)
	}
	var nonce [32]byte
	copy(nonce[:], challengeBuf[48:80])
	crFrame := buildChallengeResponseFrameWrongKey(svtnID, wrongPriv, nonce)
	_, _ = conn.Write(crFrame)
	_, _ = io.Copy(io.Discard, conn) // drain until daemon closes conn

	if !scanForLine(&buf, "E-ADM-001", 2*time.Second) {
		t.Errorf("AC-008: want E-ADM-001 in daemon log; lines so far:\n%s", buf.String())
	}
	svtnHex := fmt.Sprintf("%x", svtnID)
	if !scanForLine(&buf, svtnHex, 2*time.Second) {
		t.Errorf("AC-008: want SVTN ID %s in daemon log (E-ADM-001); lines:\n%s", svtnHex, buf.String())
	}
}

// ── AC-009: E-ADM-022 (handshake timeout) ─────────────────────────────────────

// TestNodeIdentifyHandshake_Timeout_LogsEADM022 is the daemon-level companion
// to TestNodeIdentifyHandshake_Timeout_E_ADM_022. It:
//  1. Overrides nodeIdentifyHandshakeTimeout to 50ms so the daemon's own
//     deadline fires quickly (Task 2 deterministic-timeout fix).
//  2. Starts runRouter.
//  3. Dials a connection and sends nothing — the daemon's internal
//     conn.SetDeadline(50ms) fires, and onAccept logs E-ADM-022.
//  4. Asserts the daemon log contains "E-ADM-022" and "handshake timeout".
//
// Discriminating property:
//   - Removing the `case errors.Is(hsErr, os.ErrDeadlineExceeded)` arm from
//     mgmt_wire.go:onAccept prevents the E-ADM-022 log; first scanForLine fails.
//   - Removing "handshake timeout" from the Sprintf format string prevents the
//     second scanForLine from matching.
//
// NOT t.Parallel(): overrides the package-level nodeIdentifyHandshakeTimeout
// var; serial execution prevents data races with tests relying on the 10s value.
//
// Traces to BC-2.01.009 Precondition 4, EC-002, E-ADM-022; AC-009.
func TestNodeIdentifyHandshake_Timeout_LogsEADM022(t *testing.T) {
	// Override the handshake deadline to 50ms so the test runs in well under 1s.
	// Restore the original production value on exit.
	orig := nodeIdentifyHandshakeTimeout
	nodeIdentifyHandshakeTimeout = 50 * time.Millisecond
	t.Cleanup(func() { nodeIdentifyHandshakeTimeout = orig })

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
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

	// Dial a connection and send nothing. The daemon's internal 50ms deadline
	// fires, closes conn, and emits the E-ADM-022 log.
	conn, err := net.Dial("tcp", cfg.ListenAddr)
	if err != nil {
		t.Fatalf("dial %s: %v", cfg.ListenAddr, err)
	}
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	_, _ = io.Copy(io.Discard, conn) // wait for daemon to close
	_ = conn.Close()

	if !scanForLine(&buf, "E-ADM-022", 2*time.Second) {
		t.Errorf("AC-009: want E-ADM-022 in daemon log; lines so far:\n%s", buf.String())
	}
	if !scanForLine(&buf, "handshake timeout", 2*time.Second) {
		t.Errorf("AC-009: want 'handshake timeout' in daemon log (AC-009 PC3); lines:\n%s", buf.String())
	}
}
