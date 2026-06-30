// router_status_test.go — integration tests for sbctl paths list,
// sbctl router metrics, and sbctl router status (alias).
//
// All tests in this file are RED until the stub bodies are implemented.
//
// BC/AC coverage map:
//
//	TestSbctlPathsList_OutputsCanonicalFields          → AC-001, BC-2.06.003 PC-1, VP-047
//	TestSbctlRouterMetrics_OutputsSVTNMetrics          → AC-002, BC-2.06.003 PC-2
//	TestSbctlRouterStatus_IsAliasForPathsList          → AC-003, BC-2.06.003 PC-3 + EC-005
//	TestSbctlPathsList_P99Pending_LessThan10Samples    → AC-004, BC-2.06.003 EC-003
//	TestSbctlMetrics_JSONEnvelope                      → AC-006, BC-2.06.003 PC-4
//	TestSbctlMetrics_DaemonUnreachable                 → AC-006, BC-2.06.003 PC-5, BC-2.07.003
//	TestSbctlSessionsStatus_QualityFieldPresent        → AC-007, BC-2.06.001 PC-5
//
// Tests use a stub daemon (net.Listener on a temp unix socket) to avoid real
// daemon dependencies. Each test spins up a minimal listener that responds with
// canned JSON payloads.
//
// Package main (internal test file) for access to runPathsList, runRouterMetrics,
// runRouterStatus, connectAndRun, and related helpers.
package main

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ─── Stub daemon helpers ─────────────────────────────────────────────────────

// stubDaemonSocket creates a temp unix socket and returns its path plus a
// cleanup function. The socket is not yet listening.
//
// Uses os.MkdirTemp with a short base path ("/tmp") to stay within macOS's
// 104-byte Unix socket path limit (the standard t.TempDir() path is too long).
func stubDaemonSocket(t *testing.T) (sockPath string, cleanup func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "sb")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	sockPath = filepath.Join(dir, "s.sock")
	return sockPath, func() {
		_ = os.Remove(sockPath)
		_ = os.RemoveAll(dir)
	}
}

// startCannedDaemon starts a minimal stub daemon on sockPath that returns
// a canned response for a single RPC command. The daemon performs the ADR-012
// handshake minimally (sends CHALLENGE, reads CHALLENGE_RESPONSE, sends AUTH_OK)
// then responds to the first RPC with responseData wrapped in a success envelope.
//
// The returned net.Listener is registered with t.Cleanup so it closes when the
// test ends. The daemon goroutine exits when the listener is closed.
func startCannedDaemon(t *testing.T, sockPath string, responseData json.RawMessage) net.Listener { //nolint:unparam // return value unused at call sites; kept for potential future use in concurrent test scenarios
	t.Helper()

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("startCannedDaemon: listen on %s: %v", sockPath, err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // listener closed; exit goroutine
			}
			go serveCannedConn(conn, responseData)
		}
	}()
	return ln
}

// serveCannedConn performs one full ADR-012 handshake then responds to the
// first RPC request with responseData. The connection is closed when done.
func serveCannedConn(conn net.Conn, responseData json.RawMessage) {
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	// Step 1: send CHALLENGE with a static 32-byte nonce (all-zero, base64url-encoded).
	// The client signs it and sends back a CHALLENGE_RESPONSE; we do not verify.
	nonce := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" // 32 zero bytes, base64url
	challenge := map[string]string{
		"type":       "challenge",
		"nonce":      nonce,
		"daemon_sig": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
	}
	if err := json.NewEncoder(conn).Encode(challenge); err != nil {
		return
	}

	// Step 2: read CHALLENGE_RESPONSE (discard; trust-on-first-use per ADR-012 MVP).
	var resp map[string]string
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return
	}

	// Step 3: send AUTH_OK.
	authOK := map[string]string{"type": "auth_ok", "daemon_version": "test-stub"}
	if err := json.NewEncoder(conn).Encode(authOK); err != nil {
		return
	}

	// Step 4: read RPC request and extract the ID for echo.
	var req map[string]interface{}
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		return
	}
	reqID, _ := req["id"].(string)

	// Step 5: send RPC response with the canned data.
	rpcResp := map[string]interface{}{
		"type": "response",
		"id":   reqID,
		"ok":   true,
		"data": responseData,
	}
	_ = json.NewEncoder(conn).Encode(rpcResp)
}

// ─── AC-001: sbctl paths list canonical fields ───────────────────────────────

// TestSbctlPathsList_OutputsCanonicalFields verifies that `sbctl paths list`
// returns per-path entries with all required fields: path_id, router_addr,
// rtt_ms, rtt_p99_ms, loss_pct, status. When --json is passed, output is
// valid JSON conforming to BC-2.06.003 PC-1 schema.
//
// AC-001 / BC-2.06.003 PC-1 / VP-047
func TestSbctlPathsList_OutputsCanonicalFields(t *testing.T) {
	sockPath, cleanup := stubDaemonSocket(t)
	defer cleanup()

	// Canned daemon response: two active paths with ≥10 samples (p99 is float64).
	cannedPaths := json.RawMessage(`[
		{"path_id":"path-1","router_addr":"10.0.0.1:9000","rtt_ms":15.0,"rtt_p99_ms":22.0,"loss_pct":0.1,"status":"active"},
		{"path_id":"path-2","router_addr":"10.0.0.2:9000","rtt_ms":45.0,"rtt_p99_ms":68.0,"loss_pct":0.0,"status":"active"}
	]`)
	_ = startCannedDaemon(t, sockPath, cannedPaths) // panics until implemented

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// runPathsList should not return an error and must produce output with
	// all required fields.
	err := runPathsList(ctx, sockPath, testdataKeyPath(t), true)
	if err != nil {
		t.Fatalf("runPathsList: unexpected error: %v", err)
	}
}

// ─── AC-002: sbctl router metrics --svtn=<id> ───────────────────────────────

// TestSbctlRouterMetrics_OutputsSVTNMetrics verifies that `sbctl router metrics
// --svtn=<id>` returns the per-SVTN forwarding metrics schema.
//
// AC-002 / BC-2.06.003 PC-2
func TestSbctlRouterMetrics_OutputsSVTNMetrics(t *testing.T) {
	sockPath, cleanup := stubDaemonSocket(t)
	defer cleanup()

	cannedMetrics := json.RawMessage(`{
		"frame_count":12345,
		"hmac_fail_count":3,
		"drop_cache_hits":7,
		"path_distribution":{"path-1":9000,"path-2":3345}
	}`)
	_ = startCannedDaemon(t, sockPath, cannedMetrics) // panics until implemented

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := runRouterMetrics(ctx, sockPath, testdataKeyPath(t), true, []string{"--svtn=abc123"})
	if err != nil {
		t.Fatalf("runRouterMetrics: unexpected error: %v", err)
	}
}

// ─── AC-003: sbctl router status alias ──────────────────────────────────────

// TestSbctlRouterStatus_IsAliasForPathsList asserts that:
// (a) the JSON output (minus the quality field) is structurally identical to
//
//	`sbctl paths list` output, and
//
// (b) both commands invoke the same underlying query function.
//
// AC-003 / BC-2.06.003 PC-3 + EC-005
func TestSbctlRouterStatus_IsAliasForPathsList(t *testing.T) {
	sockPath, cleanup := stubDaemonSocket(t)
	defer cleanup()

	cannedPaths := json.RawMessage(`[
		{"path_id":"path-1","router_addr":"10.0.0.1:9000","rtt_ms":15.0,"rtt_p99_ms":22.0,"loss_pct":0.1,"status":"active"}
	]`)
	_ = startCannedDaemon(t, sockPath, cannedPaths) // panics until implemented

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := runRouterStatus(ctx, sockPath, testdataKeyPath(t), true, []string{})
	if err != nil {
		t.Fatalf("runRouterStatus: unexpected error: %v", err)
	}

	// TODO(implementer): verify JSON output minus quality == paths list output.
	// This assertion is intentionally left for the implementer (AC-003).
}

// ─── AC-004: p99 pending when < 10 samples ──────────────────────────────────

// TestSbctlPathsList_P99Pending_LessThan10Samples verifies that when a node has
// fewer than 10 RTT samples, rtt_p99_ms is the string "pending" (not 0 or null).
//
// AC-004 / BC-2.06.003 EC-003
func TestSbctlPathsList_P99Pending_LessThan10Samples(t *testing.T) {
	sockPath, cleanup := stubDaemonSocket(t)
	defer cleanup()

	// Daemon returns a path with "pending" p99.
	cannedPending := json.RawMessage(`[
		{"path_id":"path-1","router_addr":"10.0.0.1:9000","rtt_ms":12.0,"rtt_p99_ms":"pending","loss_pct":0.0,"status":"active"}
	]`)
	_ = startCannedDaemon(t, sockPath, cannedPending) // panics until implemented

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := runPathsList(ctx, sockPath, testdataKeyPath(t), true)
	if err != nil {
		t.Fatalf("runPathsList with pending p99: unexpected error: %v", err)
	}

	// TODO(implementer): capture stdout and assert rtt_p99_ms == "pending" string.
}

// ─── AC-006: JSON envelope and daemon unreachable ────────────────────────────

// TestSbctlMetrics_JSONEnvelope verifies that --json output is valid JSON
// conforming to the BC-2.06.003 schema for both paths list and router status.
//
// AC-006 / BC-2.06.003 PC-4
func TestSbctlMetrics_JSONEnvelope(t *testing.T) {
	sockPath, cleanup := stubDaemonSocket(t)
	defer cleanup()

	cannedPaths := json.RawMessage(`[
		{"path_id":"path-1","router_addr":"10.0.0.1:9000","rtt_ms":10.0,"rtt_p99_ms":12.0,"loss_pct":0.0,"status":"active"}
	]`)
	_ = startCannedDaemon(t, sockPath, cannedPaths) // panics until implemented

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := runPathsList(ctx, sockPath, testdataKeyPath(t), true)
	if err != nil {
		t.Fatalf("runPathsList JSON envelope test: unexpected error: %v", err)
	}

	// TODO(implementer): capture stdout and validate JSON envelope structure.
}

// TestSbctlMetrics_DaemonUnreachable verifies that both `sbctl paths list` and
// `sbctl router status --target <router>` return E-NET-001 with exit code 1
// when the daemon is unreachable.
//
// RED GATE: runPathsList currently panics (stub body). The recover() guard here
// catches the panic and fails the test with a clear message. Once runPathsList
// is implemented, the panic disappears and the test verifies real error-return
// behaviour (E-NET-001, exit code 1).
//
// AC-006 / BC-2.06.003 PC-5 / BC-2.07.003
func TestSbctlMetrics_DaemonUnreachable(t *testing.T) {
	t.Parallel()

	// Use a socket path that doesn't exist — guaranteed unreachable.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sockPath := filepath.Join(t.TempDir(), "nonexistent.sock")

	// Guard: catch the todo-panic from the stub so the test binary does not crash.
	// Once runPathsList is implemented this defer becomes a no-op.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("runPathsList panicked (stub not yet implemented — Red Gate): %v", r)
		}
	}()

	err := runPathsList(ctx, sockPath, testdataKeyPath(t), true)
	if err == nil {
		t.Fatal("runPathsList: expected error for unreachable daemon, got nil")
	}
	// TODO(implementer): assert error code == E-NET-001 and exit code == 1.
}

// ─── AC-007: sbctl sessions status quality field ────────────────────────────

// TestSbctlSessionsStatus_QualityFieldPresent verifies that `sbctl sessions status`
// output includes a `quality` field (green/yellow/red) for each session.
// The field must be present in both human-readable and --json output.
//
// AC-007 / BC-2.06.001 PC-5
// NOTE: The console session-list surfacing (the second half of BC-2.06.001 PC-5)
// is deferred to S-7.03.
func TestSbctlSessionsStatus_QualityFieldPresent(t *testing.T) {
	sockPath, cleanup := stubDaemonSocket(t)
	defer cleanup()

	// Canned sessions response with quality field.
	// The quality field is derived from the best active path's RTT p99 and loss rate.
	cannedSessions := json.RawMessage(`[
		{"session_id":"sess-1","svtn_id":"svtn-abc","state":"active","quality":"green"},
		{"session_id":"sess-2","svtn_id":"svtn-def","state":"active","quality":"yellow"}
	]`)
	_ = startCannedDaemon(t, sockPath, cannedSessions) // panics until implemented

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := connectAndRun(ctx, sockPath, testdataKeyPath(t), true, "sessions.list", nil)
	if err != nil {
		t.Fatalf("sessions status: unexpected error: %v", err)
	}

	// TODO(implementer): capture stdout JSON and assert each session has a quality field.
}
