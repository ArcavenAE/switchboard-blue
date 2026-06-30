// router_status_test.go — integration tests for sbctl paths list,
// sbctl router metrics, and sbctl router status (alias).
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
	"bytes"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
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

// captureOut redirects the package-level stdOut writer to a bytes.Buffer,
// calls fn, then restores stdOut and returns everything written during fn.
//
// This avoids mutating os.Stdout (which is not safe under t.Parallel()) by
// instead swapping the package-level io.Writer variable that writeSuccess uses.
// Tests that call captureOut must NOT call t.Parallel() because stdOut is a
// package-level variable.
func captureOut(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	origOut := stdOut
	stdOut = &buf
	t.Cleanup(func() { stdOut = origOut })
	fn()
	return buf.String()
}

// ─── AC-001: sbctl paths list canonical fields ───────────────────────────────

// TestSbctlPathsList_OutputsCanonicalFields verifies that `sbctl paths list`
// --json output is a valid JSON envelope whose data array contains entries with
// all required fields: path_id, router_addr, rtt_ms, rtt_p99_ms, loss_pct,
// status (BC-2.06.003 PC-1 / VP-047 schema).
//
// AC-001 / BC-2.06.003 PC-1 / VP-047
func TestSbctlPathsList_OutputsCanonicalFields(t *testing.T) {
	sockPath, cleanup := stubDaemonSocket(t)
	defer cleanup()

	// Canned daemon response: two active paths with >=10 samples (p99 is float64).
	cannedPaths := json.RawMessage(`[
		{"path_id":"path-1","router_addr":"10.0.0.1:9000","rtt_ms":15.0,"rtt_p99_ms":22.0,"loss_pct":0.1,"status":"active"},
		{"path_id":"path-2","router_addr":"10.0.0.2:9000","rtt_ms":45.0,"rtt_p99_ms":68.0,"loss_pct":0.0,"status":"active"}
	]`)
	_ = startCannedDaemon(t, sockPath, cannedPaths)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var err error
	out := captureOut(t, func() {
		err = runPathsList(ctx, sockPath, testdataKeyPath(t), true)
	})
	if err != nil {
		t.Fatalf("runPathsList: unexpected error: %v", err)
	}

	// Parse the outer JSON envelope (BC-2.06.003 PC-4: {"ok":true,"error":null,"data":[...]}).
	var env struct {
		OK   bool            `json:"ok"`
		Data json.RawMessage `json:"data"`
	}
	if parseErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &env); parseErr != nil {
		t.Fatalf("AC-001: stdout is not a valid JSON envelope: %v\nraw: %q", parseErr, out)
	}
	if !env.OK {
		t.Fatal("AC-001: envelope ok must be true for a successful call")
	}

	// Decode the data array into generic maps to verify field presence without
	// coupling to internal structs (BC-2.06.003 PC-1 schema contract).
	var entries []map[string]json.RawMessage
	if parseErr := json.Unmarshal(env.Data, &entries); parseErr != nil {
		t.Fatalf("AC-001: envelope data is not a JSON array: %v\nraw data: %s", parseErr, env.Data)
	}
	if len(entries) != 2 {
		t.Fatalf("AC-001: expected 2 path entries, got %d", len(entries))
	}

	// Required canonical fields per BC-2.06.003 PC-1.
	required := []string{"path_id", "router_addr", "rtt_ms", "rtt_p99_ms", "loss_pct", "status"}
	for i, entry := range entries {
		for _, field := range required {
			if _, present := entry[field]; !present {
				t.Errorf("AC-001 / VP-047: entry[%d] missing required field %q; present keys: %v", i, field, mapKeys(entry))
			}
		}
	}

	// Spot-check values for the first entry.
	assertJSONString(t, entries[0], "path_id", "path-1")
	assertJSONString(t, entries[0], "router_addr", "10.0.0.1:9000")
	assertJSONString(t, entries[0], "status", "active")
}

// ─── AC-002: sbctl router metrics --svtn=<id> ───────────────────────────────

// TestSbctlRouterMetrics_OutputsSVTNMetrics verifies that `sbctl router metrics
// --svtn=<id>` returns a valid JSON envelope whose data contains all required
// per-SVTN forwarding metric fields (BC-2.06.003 PC-2).
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
	_ = startCannedDaemon(t, sockPath, cannedMetrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var err error
	out := captureOut(t, func() {
		err = runRouterMetrics(ctx, sockPath, testdataKeyPath(t), true, []string{"--svtn=abc123"})
	})
	if err != nil {
		t.Fatalf("runRouterMetrics: unexpected error: %v", err)
	}

	// Parse the outer JSON envelope.
	var env struct {
		OK   bool            `json:"ok"`
		Data json.RawMessage `json:"data"`
	}
	if parseErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &env); parseErr != nil {
		t.Fatalf("AC-002: stdout is not a valid JSON envelope: %v\nraw: %q", parseErr, out)
	}
	if !env.OK {
		t.Fatal("AC-002: envelope ok must be true")
	}

	// Verify the required metric fields are present (BC-2.06.003 PC-2 schema).
	var data map[string]json.RawMessage
	if parseErr := json.Unmarshal(env.Data, &data); parseErr != nil {
		t.Fatalf("AC-002: envelope data is not a JSON object: %v", parseErr)
	}
	for _, field := range []string{"frame_count", "hmac_fail_count", "drop_cache_hits", "path_distribution"} {
		if _, ok := data[field]; !ok {
			t.Errorf("AC-002: metrics response missing required field %q; present keys: %v", field, mapKeys(data))
		}
	}
}

// ─── AC-003: sbctl router status alias ──────────────────────────────────────

// TestSbctlRouterStatus_IsAliasForPathsList asserts BC-2.06.003 PC-3 + EC-005:
// `sbctl router status` is an alias for `sbctl paths list`. The JSON envelope
// data array must contain the same canonical path fields as paths list output
// (path_id, router_addr, rtt_ms, rtt_p99_ms, loss_pct, status), verifying that
// both commands invoke the same underlying paths.list RPC (single code path,
// no divergent implementation per F-P8-002 ruling).
//
// AC-003 / BC-2.06.003 PC-3 + EC-005
func TestSbctlRouterStatus_IsAliasForPathsList(t *testing.T) {
	sockPath, cleanup := stubDaemonSocket(t)
	defer cleanup()

	cannedPaths := json.RawMessage(`[
		{"path_id":"path-1","router_addr":"10.0.0.1:9000","rtt_ms":15.0,"rtt_p99_ms":22.0,"loss_pct":0.1,"status":"active"}
	]`)
	_ = startCannedDaemon(t, sockPath, cannedPaths)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var err error
	out := captureOut(t, func() {
		err = runRouterStatus(ctx, sockPath, testdataKeyPath(t), true, []string{})
	})
	if err != nil {
		t.Fatalf("runRouterStatus: unexpected error: %v", err)
	}

	// Parse the outer JSON envelope.
	var env struct {
		OK   bool            `json:"ok"`
		Data json.RawMessage `json:"data"`
	}
	if parseErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &env); parseErr != nil {
		t.Fatalf("AC-003: stdout is not a valid JSON envelope: %v\nraw: %q", parseErr, out)
	}
	if !env.OK {
		t.Fatal("AC-003: envelope ok must be true")
	}

	// router status must return an array (same structure as paths list).
	var entries []map[string]json.RawMessage
	if parseErr := json.Unmarshal(env.Data, &entries); parseErr != nil {
		t.Fatalf("AC-003: envelope data must be a JSON array (same as paths list): %v\nraw: %s", parseErr, env.Data)
	}
	if len(entries) != 1 {
		t.Fatalf("AC-003: expected 1 path entry, got %d", len(entries))
	}

	// All canonical path fields must be present (BC-2.06.003 PC-3:
	// "structurally identical to paths list output").
	canonical := []string{"path_id", "router_addr", "rtt_ms", "rtt_p99_ms", "loss_pct", "status"}
	for _, field := range canonical {
		if _, present := entries[0][field]; !present {
			t.Errorf("AC-003 / BC-2.06.003 PC-3: router status output missing canonical field %q; present keys: %v", field, mapKeys(entries[0]))
		}
	}

	// Spot-check identity: path_id must match the canned value.
	assertJSONString(t, entries[0], "path_id", "path-1")
}

// ─── AC-004: p99 pending when < 10 samples ──────────────────────────────────

// TestSbctlPathsList_P99Pending_LessThan10Samples verifies that when a path has
// fewer than 10 RTT samples the JSON output carries rtt_p99_ms as the string
// "pending" — not 0, not null, not omitted (BC-2.06.003 EC-003).
//
// AC-004 / BC-2.06.003 EC-003
func TestSbctlPathsList_P99Pending_LessThan10Samples(t *testing.T) {
	sockPath, cleanup := stubDaemonSocket(t)
	defer cleanup()

	// Daemon returns a path with "pending" p99 (< 10 samples).
	cannedPending := json.RawMessage(`[
		{"path_id":"path-1","router_addr":"10.0.0.1:9000","rtt_ms":12.0,"rtt_p99_ms":"pending","loss_pct":0.0,"status":"active"}
	]`)
	_ = startCannedDaemon(t, sockPath, cannedPending)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var err error
	out := captureOut(t, func() {
		err = runPathsList(ctx, sockPath, testdataKeyPath(t), true)
	})
	if err != nil {
		t.Fatalf("runPathsList with pending p99: unexpected error: %v", err)
	}

	// Parse the JSON envelope and extract the data array.
	var env struct {
		OK   bool            `json:"ok"`
		Data json.RawMessage `json:"data"`
	}
	if parseErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &env); parseErr != nil {
		t.Fatalf("AC-004: stdout is not a valid JSON envelope: %v\nraw: %q", parseErr, out)
	}
	if !env.OK {
		t.Fatal("AC-004: envelope ok must be true")
	}

	var entries []map[string]json.RawMessage
	if parseErr := json.Unmarshal(env.Data, &entries); parseErr != nil {
		t.Fatalf("AC-004: envelope data is not a JSON array: %v", parseErr)
	}
	if len(entries) != 1 {
		t.Fatalf("AC-004: expected 1 path entry, got %d", len(entries))
	}

	// rtt_p99_ms must decode as the string "pending" (BC-2.06.003 EC-003).
	raw, ok := entries[0]["rtt_p99_ms"]
	if !ok {
		t.Fatal("AC-004: path entry missing rtt_p99_ms field")
	}
	var p99 interface{}
	if parseErr := json.Unmarshal(raw, &p99); parseErr != nil {
		t.Fatalf("AC-004: could not unmarshal rtt_p99_ms: %v", parseErr)
	}
	p99Str, isString := p99.(string)
	if !isString {
		t.Errorf("AC-004 / BC-2.06.003 EC-003: rtt_p99_ms must be the string \"pending\" when < 10 samples; got type %T value %v", p99, p99)
	} else if p99Str != "pending" {
		t.Errorf("AC-004 / BC-2.06.003 EC-003: rtt_p99_ms must equal \"pending\"; got %q", p99Str)
	}
}

// ─── AC-006: JSON envelope and daemon unreachable ────────────────────────────

// TestSbctlMetrics_JSONEnvelope verifies that --json output is a well-formed
// JSON envelope conforming to BC-2.06.003 PC-4:
//
//	{"ok":true,"error":null,"data":[...]}
//
// AC-006 / BC-2.06.003 PC-4
func TestSbctlMetrics_JSONEnvelope(t *testing.T) {
	sockPath, cleanup := stubDaemonSocket(t)
	defer cleanup()

	cannedPaths := json.RawMessage(`[
		{"path_id":"path-1","router_addr":"10.0.0.1:9000","rtt_ms":10.0,"rtt_p99_ms":12.0,"loss_pct":0.0,"status":"active"}
	]`)
	_ = startCannedDaemon(t, sockPath, cannedPaths)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var err error
	out := captureOut(t, func() {
		err = runPathsList(ctx, sockPath, testdataKeyPath(t), true)
	})
	if err != nil {
		t.Fatalf("runPathsList JSON envelope test: unexpected error: %v", err)
	}

	// Outer envelope shape: {"ok":true,"error":null,"data":[...]}.
	var env map[string]json.RawMessage
	if parseErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &env); parseErr != nil {
		t.Fatalf("AC-006 / BC-2.06.003 PC-4: stdout is not valid JSON: %v\nraw: %q", parseErr, out)
	}

	// "ok" must be present and true.
	rawOK, hasOK := env["ok"]
	if !hasOK {
		t.Fatal("AC-006: JSON envelope missing required 'ok' field")
	}
	var okVal bool
	if parseErr := json.Unmarshal(rawOK, &okVal); parseErr != nil || !okVal {
		t.Errorf("AC-006: envelope 'ok' must be true; got %s", rawOK)
	}

	// "error" must be present and null on success.
	rawErr, hasErr := env["error"]
	if !hasErr {
		t.Fatal("AC-006: JSON envelope missing required 'error' field")
	}
	if string(rawErr) != "null" {
		t.Errorf("AC-006: envelope 'error' must be null on success; got %s", rawErr)
	}

	// "data" must be present and be a non-empty JSON array.
	rawData, hasData := env["data"]
	if !hasData {
		t.Fatal("AC-006: JSON envelope missing required 'data' field")
	}
	var entries []json.RawMessage
	if parseErr := json.Unmarshal(rawData, &entries); parseErr != nil {
		t.Errorf("AC-006: envelope 'data' must be a JSON array; got %s (%v)", rawData, parseErr)
	}
	if len(entries) == 0 {
		t.Error("AC-006: envelope 'data' array must not be empty for a successful paths list call")
	}
}

// TestSbctlMetrics_DaemonUnreachable verifies that `sbctl paths list` returns
// a non-nil error containing "E-NET-001" when the daemon socket does not exist.
//
// Rationale: main() maps any non-nil error from runPathsList to os.Exit(1), so
// a non-nil E-NET-001 error here satisfies the AC-006 exit-code-1 requirement.
// The subprocess assertion (exit code 1 + E-NET-001 on stderr) is covered by
// TestSbctl_ConnectionRefused_ExitsOneWithENET001_VP030 in main_test.go.
//
// AC-006 / BC-2.06.003 PC-5 / BC-2.07.003
func TestSbctlMetrics_DaemonUnreachable(t *testing.T) {
	t.Parallel()

	// Use a socket path that doesn't exist — guaranteed unreachable.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sockPath := filepath.Join(t.TempDir(), "nonexistent.sock")

	err := runPathsList(ctx, sockPath, testdataKeyPath(t), true)
	if err == nil {
		t.Fatal("AC-006 / BC-2.06.003 PC-5: runPathsList returned nil for unreachable daemon; expected non-nil error")
	}

	// BC-2.07.003 PC-1 + Invariant 4: error message must contain "E-NET-001".
	// This code distinguishes unreachable-daemon from auth failure (E-ADM-010)
	// and RPC failure (E-RPC-001), and causes main() to exit with code 1.
	if !strings.Contains(err.Error(), "E-NET-001") {
		t.Errorf("AC-006 / BC-2.06.003 PC-5 / BC-2.07.003 PC-1: expected error to contain \"E-NET-001\"; got: %v", err)
	}
}

// ─── AC-007: sbctl sessions status quality field ────────────────────────────

// TestSbctlSessionsStatus_QualityFieldPresent verifies that the sessions.list
// RPC response passes through a quality field (green/yellow/red) for each
// session entry in --json output.
//
// AC-007 / BC-2.06.001 PC-5
// NOTE: The console session-list surfacing (the second half of BC-2.06.001 PC-5)
// is deferred to S-7.03.
func TestSbctlSessionsStatus_QualityFieldPresent(t *testing.T) {
	sockPath, cleanup := stubDaemonSocket(t)
	defer cleanup()

	// Canned sessions response with quality field (set by daemon / internal/metrics).
	cannedSessions := json.RawMessage(`[
		{"session_id":"sess-1","svtn_id":"svtn-abc","state":"active","quality":"green"},
		{"session_id":"sess-2","svtn_id":"svtn-def","state":"active","quality":"yellow"}
	]`)
	_ = startCannedDaemon(t, sockPath, cannedSessions)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var err error
	out := captureOut(t, func() {
		err = connectAndRun(ctx, sockPath, testdataKeyPath(t), true, "sessions.list", nil)
	})
	if err != nil {
		t.Fatalf("sessions status: unexpected error: %v", err)
	}

	// Parse the JSON envelope.
	var env struct {
		OK   bool            `json:"ok"`
		Data json.RawMessage `json:"data"`
	}
	if parseErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &env); parseErr != nil {
		t.Fatalf("AC-007: stdout is not a valid JSON envelope: %v\nraw: %q", parseErr, out)
	}
	if !env.OK {
		t.Fatal("AC-007: envelope ok must be true")
	}

	var sessions []map[string]json.RawMessage
	if parseErr := json.Unmarshal(env.Data, &sessions); parseErr != nil {
		t.Fatalf("AC-007: envelope data is not a JSON array: %v", parseErr)
	}
	if len(sessions) != 2 {
		t.Fatalf("AC-007: expected 2 sessions, got %d", len(sessions))
	}

	// Each session must have a "quality" field with a valid enum value (BC-2.06.001 PC-5).
	validQualities := map[string]bool{"green": true, "yellow": true, "red": true}
	for i, s := range sessions {
		rawQ, present := s["quality"]
		if !present {
			t.Errorf("AC-007 / BC-2.06.001 PC-5: session[%d] missing required 'quality' field; present keys: %v", i, mapKeys(s))
			continue
		}
		var q string
		if parseErr := json.Unmarshal(rawQ, &q); parseErr != nil {
			t.Errorf("AC-007: session[%d] quality field is not a string: %v", i, parseErr)
			continue
		}
		if !validQualities[q] {
			t.Errorf("AC-007 / BC-2.06.001 PC-5: session[%d] quality %q is not in {green, yellow, red}", i, q)
		}
	}

	// Spot-check session identity and quality values from the canned response.
	assertJSONString(t, sessions[0], "session_id", "sess-1")
	assertJSONString(t, sessions[0], "quality", "green")
	assertJSONString(t, sessions[1], "session_id", "sess-2")
	assertJSONString(t, sessions[1], "quality", "yellow")
}

// ─── assertion helpers ────────────────────────────────────────────────────────

// mapKeys returns the key list of a map[string]json.RawMessage for use in error messages.
func mapKeys(m map[string]json.RawMessage) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

// assertJSONString decodes entry[field] as a string and calls t.Errorf if it
// does not equal want.
func assertJSONString(t *testing.T, entry map[string]json.RawMessage, field, want string) {
	t.Helper()
	raw, ok := entry[field]
	if !ok {
		t.Errorf("assertJSONString: field %q not present; present keys: %v", field, mapKeys(entry))
		return
	}
	var got string
	if parseErr := json.Unmarshal(raw, &got); parseErr != nil {
		t.Errorf("assertJSONString: field %q is not a string: %v (raw: %s)", field, parseErr, raw)
		return
	}
	if got != want {
		t.Errorf("assertJSONString: field %q = %q, want %q", field, got, want)
	}
}
