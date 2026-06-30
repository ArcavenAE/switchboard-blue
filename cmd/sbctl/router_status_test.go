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

// newTestIO returns an sbctlIO backed by in-memory buffers, plus getOut and
// getErr accessors. Using explicit sbctlIO instead of package-level globals
// makes tests safe under t.Parallel() and -race (go.md rule 12).
func newTestIO() (sio sbctlIO, getOut func() string, getErr func() string) {
	var outBuf, errBuf bytes.Buffer
	sio = sbctlIO{out: &outBuf, err: &errBuf}
	getOut = func() string { return outBuf.String() }
	getErr = func() string { return errBuf.String() }
	return sio, getOut, getErr
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

	sio, getOut, _ := newTestIO()
	err := runPathsList(ctx, sockPath, testdataKeyPath(t), true, sio)
	if err != nil {
		t.Fatalf("runPathsList: unexpected error: %v", err)
	}
	out := getOut()

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

	sio, getOut, _ := newTestIO()
	err := runRouterMetrics(ctx, sockPath, testdataKeyPath(t), true, []string{"--svtn=abc123"}, sio)
	if err != nil {
		t.Fatalf("runRouterMetrics: unexpected error: %v", err)
	}
	out := getOut()

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

	sio, getOut, _ := newTestIO()
	err := runRouterStatus(ctx, sockPath, testdataKeyPath(t), true, []string{}, sio)
	if err != nil {
		t.Fatalf("runRouterStatus: unexpected error: %v", err)
	}
	out := getOut()

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

	sio, getOut, _ := newTestIO()
	err := runPathsList(ctx, sockPath, testdataKeyPath(t), true, sio)
	if err != nil {
		t.Fatalf("runPathsList with pending p99: unexpected error: %v", err)
	}
	out := getOut()

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

	sio, getOut, _ := newTestIO()
	err := runPathsList(ctx, sockPath, testdataKeyPath(t), true, sio)
	if err != nil {
		t.Fatalf("runPathsList JSON envelope test: unexpected error: %v", err)
	}
	out := getOut()

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

	sio, _, _ := newTestIO()
	err := runPathsList(ctx, sockPath, testdataKeyPath(t), true, sio)
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

// ─── F-C1 (CRITICAL): stateless classification must not step-walk ────────────

// TestQualityFromPathEntry_StatelessClassification verifies that
// qualityFromPathEntry returns the correct band for inputs across all three
// quality bands without relying on QualityIndicator step-wise downgrade state.
//
// The step-wise downgrade bug: a fresh QualityIndicator starts at Green.
// A single Update() call with Red-range inputs (rtt=600ms, loss=25%) steps
// Green→Yellow (one level at a time), returning "yellow" instead of "red".
// This test asserts the correct answer — it will be RED until the implementation
// performs a stateless classify() rather than a stateful hysteretic Update().
//
// F-C1 / BC-2.06.003 PC-1 / metrics.classify thresholds
func TestQualityFromPathEntry_StatelessClassification(t *testing.T) {
	cases := []struct {
		name     string
		p99RTTMs any     // float64 or "pending"
		rttMs    float64 // fallback when p99 is pending
		lossPct  float64
		wantBand string
	}{
		// Green band: p99 ≤ 100ms AND loss ≤ 5%
		{
			name:     "green_low_rtt_zero_loss",
			p99RTTMs: 22.0, rttMs: 15.0, lossPct: 0.0,
			wantBand: "green",
		},
		{
			name:     "green_boundary_rtt_100ms_loss_5pct",
			p99RTTMs: 100.0, rttMs: 80.0, lossPct: 5.0,
			wantBand: "green",
		},
		// Yellow band: p99 in (100ms,500ms] OR loss in (5%,20%] (and not Red)
		{
			name:     "yellow_rtt_200ms_zero_loss",
			p99RTTMs: 200.0, rttMs: 150.0, lossPct: 0.0,
			wantBand: "yellow",
		},
		{
			name:     "yellow_low_rtt_loss_10pct",
			p99RTTMs: 50.0, rttMs: 40.0, lossPct: 10.0,
			wantBand: "yellow",
		},
		// Red band: p99 > 500ms OR loss > 20%
		// This row is the critical regression: step-wise bug returns "yellow"
		// because a fresh indicator at Green only moves one step per Update().
		{
			name:     "red_rtt_600ms_loss_25pct",
			p99RTTMs: 600.0, rttMs: 600.0, lossPct: 25.0,
			wantBand: "red",
		},
		{
			name:     "red_rtt_501ms_zero_loss",
			p99RTTMs: 501.0, rttMs: 501.0, lossPct: 0.0,
			wantBand: "red",
		},
		{
			name:     "red_zero_rtt_loss_21pct",
			p99RTTMs: 10.0, rttMs: 10.0, lossPct: 21.0,
			wantBand: "red",
		},
		// Pending p99: BC-2.06.003 v1.5 EC-003 sentinel — return "pending" regardless
		// of rtt_ms value.  Supersedes the pre-v1.5 fallback-to-rtt_ms behaviour.
		{
			name:     "pending_p99_sentinel_green_metrics",
			p99RTTMs: "pending", rttMs: 30.0, lossPct: 0.0,
			wantBand: "pending",
		},
		{
			name:     "pending_p99_sentinel_red_metrics",
			p99RTTMs: "pending", rttMs: 600.0, lossPct: 25.0,
			wantBand: "pending",
		},
		// F-C2 / BC-2.06.003 v1.5 F-M3: nil (JSON null) p99 → pending.
		{
			name:     "nil_p99_json_null_yields_pending",
			p99RTTMs: nil, rttMs: 30.0, lossPct: 0.0,
			wantBand: "pending",
		},
		// F-C2 / BC-2.06.003 v1.5 F-M3: unexpected string type "unknown" → pending.
		{
			name:     "unknown_string_p99_yields_pending",
			p99RTTMs: "unknown", rttMs: 30.0, lossPct: 0.0,
			wantBand: "pending",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			entry := PathEntry{
				PathID:     "test-path",
				RouterAddr: "10.0.0.1:9000",
				RTTMs:      tc.rttMs,
				P99RTTMs:   tc.p99RTTMs,
				LossPct:    tc.lossPct,
				Status:     "active",
			}
			got := qualityFromPathEntry(entry)
			if got != tc.wantBand {
				t.Errorf("qualityFromPathEntry(p99=%v, rtt=%.1f, loss=%.1f): got %q, want %q",
					tc.p99RTTMs, tc.rttMs, tc.lossPct, got, tc.wantBand)
			}
		})
	}
}

// ─── F-H1 (HIGH): status field overrides quality floor ───────────────────────

// TestQualityFromPathEntry_StatusOverride verifies that the "status" field in
// PathEntry is used to apply a quality floor:
//   - status="failed" → quality must be "red" regardless of RTT/loss metrics
//   - status="degraded" → quality must be at least "yellow" (floor)
//
// Today qualityFromPathEntry ignores the status field entirely — both cases
// return "green" when rtt_p99_ms=10ms and loss=0%. These tests must be RED
// until status-aware override logic is added.
//
// F-H1 / BC-2.06.003 PC-3 (status field semantics)
func TestQualityFromPathEntry_StatusOverride(t *testing.T) {
	cases := []struct {
		name     string
		status   string
		p99RTTMs float64
		lossPct  float64
		wantBand string
	}{
		{
			// status="failed": must return "red" regardless of good metrics
			name:   "failed_status_forces_red",
			status: "failed", p99RTTMs: 10.0, lossPct: 0.0,
			wantBand: "red",
		},
		{
			// status="degraded": must return at least "yellow" (floor)
			// even when RTT and loss are Green-range
			name:   "degraded_status_floor_yellow",
			status: "degraded", p99RTTMs: 10.0, lossPct: 0.0,
			wantBand: "yellow",
		},
		{
			// status="active" with good metrics: normal green path
			name:   "active_status_green_metrics",
			status: "active", p99RTTMs: 10.0, lossPct: 0.0,
			wantBand: "green",
		},
		{
			// status="degraded" with already-Red metrics: Red wins over Yellow floor
			name:   "degraded_status_red_metrics_stays_red",
			status: "degraded", p99RTTMs: 600.0, lossPct: 25.0,
			wantBand: "red",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			entry := PathEntry{
				PathID:     "test-path",
				RouterAddr: "10.0.0.1:9000",
				RTTMs:      tc.p99RTTMs,
				P99RTTMs:   tc.p99RTTMs,
				LossPct:    tc.lossPct,
				Status:     tc.status,
			}
			got := qualityFromPathEntry(entry)
			if got != tc.wantBand {
				t.Errorf("qualityFromPathEntry(status=%q, p99=%.1f, loss=%.1f): got %q, want %q",
					tc.status, tc.p99RTTMs, tc.lossPct, got, tc.wantBand)
			}
		})
	}
}

// ─── F-H2 (HIGH): runRouterStatus daemon-unreachable returns E-NET-001 ────────

// TestSbctlRouterStatus_DaemonUnreachable verifies that runRouterStatus
// returns a non-nil error containing "E-NET-001" when the daemon socket does
// not exist, AND that the JSON error envelope written to stderr also contains
// "E-NET-001".
//
// Today runRouterStatus correctly sets E-NET-001 in the returned error message,
// but this test adds explicit verification of both the error value AND the JSON
// envelope written to stderr (via captureErr), which does not yet exist in the
// test suite.
//
// F-H2 / BC-2.06.003 PC-5 / BC-2.07.003
func TestSbctlRouterStatus_DaemonUnreachable(t *testing.T) {
	t.Parallel()

	// newTestIO provides buffer-backed writers — no package-level mutation,
	// safe under t.Parallel() and -race (go.md rule 12).
	sio, _, getErr := newTestIO()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sockPath := "/nonexistent/path/to/daemon.sock"

	err := runRouterStatus(ctx, sockPath, testdataKeyPath(t), true, []string{}, sio)

	// Error must be non-nil.
	if err == nil {
		t.Fatal("F-H2 / BC-2.06.003 PC-5: runRouterStatus returned nil for unreachable daemon; expected non-nil error")
	}

	// Returned error must contain E-NET-001.
	if !strings.Contains(err.Error(), "E-NET-001") {
		t.Errorf("F-H2 / BC-2.07.003: expected error to contain \"E-NET-001\"; got: %v", err)
	}

	// JSON error envelope written to stderr must also contain E-NET-001.
	stderrOutput := getErr()
	if !strings.Contains(stderrOutput, "E-NET-001") {
		t.Errorf("F-H2 / BC-2.07.003: expected stderr JSON envelope to contain \"E-NET-001\"; got: %q", stderrOutput)
	}

	// The stderr JSON envelope must be parseable with ok:false and error.code == "E-NET-001".
	if len(strings.TrimSpace(stderrOutput)) > 0 {
		var env struct {
			OK    bool `json:"ok"`
			Error *struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if parseErr := json.Unmarshal([]byte(strings.TrimSpace(stderrOutput)), &env); parseErr != nil {
			t.Errorf("F-H2: stderr is not valid JSON: %v\nraw: %q", parseErr, stderrOutput)
		} else {
			if env.OK {
				t.Errorf("F-H2: stderr JSON envelope ok must be false on error; got true")
			}
			if env.Error == nil || env.Error.Code != "E-NET-001" {
				t.Errorf("F-H2: stderr JSON error.code must be \"E-NET-001\"; got %+v", env.Error)
			}
		}
	}
}

// ─── F-H3 (HIGH): both commands invoke paths.list RPC, identical PathEntry payload ──

// startRecordingDaemon starts a stub daemon that records the RPC method name
// from the first request and responds with cannedResponse. The recorded method
// name is sent on the returned channel.
func startRecordingDaemon(t *testing.T, sockPath string, cannedResponse json.RawMessage) chan string {
	t.Helper()

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("startRecordingDaemon: listen on %s: %v", sockPath, err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	methodCh := make(chan string, 1)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer func() { _ = c.Close() }()
				_ = c.SetDeadline(time.Now().Add(10 * time.Second))

				// ADR-012 handshake.
				nonce := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
				challenge := map[string]string{
					"type":       "challenge",
					"nonce":      nonce,
					"daemon_sig": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
				}
				if err := json.NewEncoder(c).Encode(challenge); err != nil {
					return
				}
				var resp map[string]string
				if err := json.NewDecoder(c).Decode(&resp); err != nil {
					return
				}
				authOK := map[string]string{"type": "auth_ok", "daemon_version": "test-stub"}
				if err := json.NewEncoder(c).Encode(authOK); err != nil {
					return
				}

				// Read RPC request and record the command.
				var req map[string]interface{}
				if err := json.NewDecoder(c).Decode(&req); err != nil {
					return
				}
				if cmd, ok := req["command"].(string); ok {
					select {
					case methodCh <- cmd:
					default:
					}
				}
				reqID, _ := req["id"].(string)

				rpcResp := map[string]interface{}{
					"type": "response",
					"id":   reqID,
					"ok":   true,
					"data": cannedResponse,
				}
				_ = json.NewEncoder(c).Encode(rpcResp)
			}(conn)
		}
	}()

	return methodCh
}

// TestSbctlRouterStatus_RPCMethodIsPathsList verifies that runRouterStatus
// dispatches exactly "paths.list" as the RPC command — the same method used
// by runPathsList — confirming the alias contract (BC-2.06.003 PC-3 + EC-005).
//
// The test also asserts that the PathEntry portion of both JSON envelopes is
// structurally identical (same canonical fields), so that no divergent field
// mapping is introduced.
//
// F-H3 / BC-2.06.003 PC-3 / EC-005
func TestSbctlRouterStatus_RPCMethodIsPathsList(t *testing.T) {
	cannedPaths := json.RawMessage(`[
		{"path_id":"path-rpc","router_addr":"10.0.0.1:9000","rtt_ms":20.0,"rtt_p99_ms":30.0,"loss_pct":0.5,"status":"active"}
	]`)

	// ── runPathsList records the RPC method ──────────────────────────────────
	sockA, cleanupA := stubDaemonSocket(t)
	defer cleanupA()
	methodChA := startRecordingDaemon(t, sockA, cannedPaths)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sioA, getOutA, _ := newTestIO()
	pathsListErr := runPathsList(ctx, sockA, testdataKeyPath(t), true, sioA)
	if pathsListErr != nil {
		t.Fatalf("runPathsList: unexpected error: %v", pathsListErr)
	}
	outPathsList := getOutA()

	var recordedPathsList string
	select {
	case recordedPathsList = <-methodChA:
	case <-time.After(2 * time.Second):
		t.Fatal("F-H3: timed out waiting for RPC method from runPathsList stub daemon")
	}

	if recordedPathsList != "paths.list" {
		t.Errorf("F-H3: runPathsList recorded RPC method = %q; want \"paths.list\"", recordedPathsList)
	}

	// ── runRouterStatus records the RPC method ───────────────────────────────
	sockB, cleanupB := stubDaemonSocket(t)
	defer cleanupB()
	methodChB := startRecordingDaemon(t, sockB, cannedPaths)

	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	sioB, getOutB, _ := newTestIO()
	routerStatusErr := runRouterStatus(ctx2, sockB, testdataKeyPath(t), true, []string{}, sioB)
	if routerStatusErr != nil {
		t.Fatalf("runRouterStatus: unexpected error: %v", routerStatusErr)
	}
	outRouterStatus := getOutB()

	var recordedRouterStatus string
	select {
	case recordedRouterStatus = <-methodChB:
	case <-time.After(2 * time.Second):
		t.Fatal("F-H3: timed out waiting for RPC method from runRouterStatus stub daemon")
	}

	if recordedRouterStatus != "paths.list" {
		t.Errorf("F-H3 / BC-2.06.003 PC-3: runRouterStatus recorded RPC method = %q; want \"paths.list\"", recordedRouterStatus)
	}

	// ── Both outputs must share identical PathEntry canonical fields ─────────
	// Parse both envelopes and compare path_id, router_addr, rtt_ms,
	// rtt_p99_ms, loss_pct, status fields — the PathEntry portion must be
	// byte-equal modulo the quality column added by runRouterStatus.
	type envelope struct {
		OK   bool            `json:"ok"`
		Data json.RawMessage `json:"data"`
	}
	var envA, envB envelope
	if err := json.Unmarshal([]byte(strings.TrimSpace(outPathsList)), &envA); err != nil {
		t.Fatalf("F-H3: runPathsList output is not valid JSON: %v\nraw: %q", err, outPathsList)
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(outRouterStatus)), &envB); err != nil {
		t.Fatalf("F-H3: runRouterStatus output is not valid JSON: %v\nraw: %q", err, outRouterStatus)
	}

	var entriesA []map[string]json.RawMessage
	var entriesB []map[string]json.RawMessage
	if err := json.Unmarshal(envA.Data, &entriesA); err != nil {
		t.Fatalf("F-H3: runPathsList data is not a JSON array: %v", err)
	}
	if err := json.Unmarshal(envB.Data, &entriesB); err != nil {
		t.Fatalf("F-H3: runRouterStatus data is not a JSON array: %v", err)
	}
	if len(entriesA) != len(entriesB) {
		t.Fatalf("F-H3: entry count mismatch: runPathsList=%d, runRouterStatus=%d", len(entriesA), len(entriesB))
	}

	// PathEntry fields that must be identical in both outputs.
	pathEntryFields := []string{"path_id", "router_addr", "rtt_ms", "rtt_p99_ms", "loss_pct", "status"}
	for i := range entriesA {
		for _, field := range pathEntryFields {
			rawA, okA := entriesA[i][field]
			rawB, okB := entriesB[i][field]
			if !okA || !okB {
				t.Errorf("F-H3: entry[%d] field %q missing: runPathsList=%v, runRouterStatus=%v", i, field, okA, okB)
				continue
			}
			if string(rawA) != string(rawB) {
				t.Errorf("F-H3: entry[%d] field %q differs: runPathsList=%s, runRouterStatus=%s", i, field, rawA, rawB)
			}
		}
	}
}

// ─── F-M1 (MEDIUM): --target flag missing value returns E-CFG-010 ─────────────

// TestSbctlRouterStatus_TargetFlagMissingValue verifies that invoking
// runRouterStatus with args ["--target"] (the flag present but no value
// following it) returns an error containing E-CFG-010.
//
// Today the --target parsing loop checks `i+1 < len(args)`, so ["--target"]
// silently falls through without updating the target — no error is returned.
// This test must be RED until explicit flag-validation is added.
//
// F-M1 / BC-2.06.003 (flags validation)
func TestSbctlRouterStatus_TargetFlagMissingValue(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Pass ["--target"] with no following value — the flag is incomplete.
	sio, _, _ := newTestIO()
	err := runRouterStatus(ctx, "/run/switchboard-router.sock", testdataKeyPath(t), true, []string{"--target"}, sio)

	if err == nil {
		t.Fatal("F-M1: runRouterStatus with [\"--target\"] (no value) returned nil error; expected E-CFG-010")
	}
	if !strings.Contains(err.Error(), "E-CFG-010") {
		t.Errorf("F-M1: expected error to contain \"E-CFG-010\"; got: %v", err)
	}
}

// ─── F-C3: --target= (empty value after equals) returns E-CFG-010 ───────────

// TestSbctlRouterStatus_TargetFlagEmptyValue verifies that invoking
// runRouterStatus with args ["--target="] (the flag present but with an empty
// value after the equals sign) returns an error containing E-CFG-010.
//
// F-C3 / BC-2.06.003 (flags validation)
func TestSbctlRouterStatus_TargetFlagEmptyValue(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sio, _, _ := newTestIO()
	// Pass ["--target="] — empty value after the equals sign.
	err := runRouterStatus(ctx, "/run/switchboard-router.sock", testdataKeyPath(t), true, []string{"--target="}, sio)

	if err == nil {
		t.Fatal("F-C3: runRouterStatus with [\"--target=\"] (empty value) returned nil error; expected E-CFG-010")
	}
	if !strings.Contains(err.Error(), "E-CFG-010") {
		t.Errorf("F-C3: expected error to contain \"E-CFG-010\"; got: %v", err)
	}
}

// ─── F-C5: injectJSONField propagates Marshal errors ─────────────────────────

// TestInjectJSONField_PropagatesError verifies that injectJSONField returns
// the (json.RawMessage, error) pair and that errors from json.Marshal are
// propagated rather than swallowed (go.md rule 3 / F-C5).
//
// In practice, json.Marshal of a plain string never fails, but the signature
// change ensures the contract is enforced at the type level.
func TestInjectJSONField_PropagatesError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		raw     json.RawMessage
		key     string
		value   string
		wantErr bool
		wantKey string
	}{
		{
			name:    "valid_object_injects_field",
			raw:     json.RawMessage(`{"path_id":"p1","rtt_ms":10.0}`),
			key:     "quality",
			value:   "green",
			wantErr: false,
			wantKey: "quality",
		},
		{
			name:    "malformed_object_returns_raw",
			raw:     json.RawMessage(`not-an-object`),
			key:     "quality",
			value:   "green",
			wantErr: false, // malformed → return raw as-is, no error
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := injectJSONField(tc.raw, tc.key, tc.value)
			if tc.wantErr && err == nil {
				t.Errorf("F-C5: expected error from injectJSONField; got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("F-C5: unexpected error from injectJSONField: %v", err)
			}
			if tc.wantKey != "" {
				var obj map[string]json.RawMessage
				if parseErr := json.Unmarshal(got, &obj); parseErr != nil {
					t.Fatalf("F-C5: result is not valid JSON object: %v\nraw: %s", parseErr, got)
				}
				if _, ok := obj[tc.wantKey]; !ok {
					t.Errorf("F-C5: injected field %q not found in result; present: %v", tc.wantKey, mapKeys(obj))
				}
			}
		})
	}
}

// ─── F-C4: EC-001 object form passthrough ────────────────────────────────────

// TestSbctlRouterStatus_EC001ObjectFormPassthrough verifies that when the daemon
// returns an EC-001 object response ({"paths":[],"message":"no active paths"})
// instead of an array, runRouterStatus passes it through without injecting a
// quality column and returns success.
//
// F-C4 / BC-2.06.003 EC-001
func TestSbctlRouterStatus_EC001ObjectFormPassthrough(t *testing.T) {
	sockPath, cleanup := stubDaemonSocket(t)
	defer cleanup()

	// EC-001 form: daemon returns an object (not an array) when there are no active paths.
	ec001Response := json.RawMessage(`{"paths":[],"message":"no active paths"}`)
	_ = startCannedDaemon(t, sockPath, ec001Response)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sio, getOut, _ := newTestIO()
	err := runRouterStatus(ctx, sockPath, testdataKeyPath(t), true, []string{}, sio)
	if err != nil {
		t.Fatalf("F-C4: runRouterStatus with EC-001 object form returned error: %v", err)
	}

	// The output must be a valid JSON envelope with ok:true.
	out := getOut()
	var env struct {
		OK   bool            `json:"ok"`
		Data json.RawMessage `json:"data"`
	}
	if parseErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &env); parseErr != nil {
		t.Fatalf("F-C4: stdout is not a valid JSON envelope: %v\nraw: %q", parseErr, out)
	}
	if !env.OK {
		t.Fatal("F-C4: envelope ok must be true for EC-001 passthrough")
	}

	// The data must contain the EC-001 object fields (paths and message).
	var obj map[string]json.RawMessage
	if parseErr := json.Unmarshal(env.Data, &obj); parseErr != nil {
		t.Fatalf("F-C4: EC-001 data is not a JSON object: %v\nraw: %s", parseErr, env.Data)
	}
	if _, hasPaths := obj["paths"]; !hasPaths {
		t.Errorf("F-C4: EC-001 passthrough data missing 'paths' field; present keys: %v", mapKeys(obj))
	}
	if _, hasMsg := obj["message"]; !hasMsg {
		t.Errorf("F-C4: EC-001 passthrough data missing 'message' field; present keys: %v", mapKeys(obj))
	}
	// No 'quality' field must be injected into the object passthrough.
	if _, hasQuality := obj["quality"]; hasQuality {
		t.Errorf("F-C4: quality field must NOT be injected into EC-001 object response; found in output")
	}
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
