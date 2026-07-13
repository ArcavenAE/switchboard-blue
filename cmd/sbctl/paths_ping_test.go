// paths_ping_test.go — integration tests for `sbctl paths ping --router=<addr>`.
//
// BC/AC coverage map:
//
//	TestPathsPing_HappyPath_ReportsRTT               → AC-001, BC-2.06.004 PC-1
//	TestPathsPing_Unreachable_ENET001                → AC-002, BC-2.06.004 PC-2, EC-001
//	TestPathsPing_AuthFailure_EADM010                → AC-002, BC-2.06.004 PC-3, EC-002
//	TestPathsPing_SlowRoundTrip_NotAnError_NoQualityField → AC-003, BC-2.06.004 PC-4, EC-003, Invariant 2
//
// runPathsPing's Red Gate stub body is an unconditional panic (no branching)
// — calling it directly in-process would crash the whole cmd/sbctl test
// binary (an unrecovered panic terminates the process; testing's per-test
// recover only guards the goroutine running t.Run, not sibling tests) and
// take every unrelated test in this package down with it. All four tests
// therefore dispatch through the real compiled main() via the
// runProductionMain subprocess helper (production_exit_code_test.go),
// matching this repo's established pattern for exercising panic-risk dispatch
// paths (main_test.go's TestSubprocessMain_* hooks). The stub daemons below
// still run in-process in the PARENT test binary — only the client half
// (sbctl's own main()) is subprocess-isolated.
package main

import (
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"
)

// startSlowPingDaemon starts a stub daemon that performs the ADR-012
// handshake, then sleeps for delay before writing the RPC response carrying
// responseData. Used by TestPathsPing_SlowRoundTrip_NotAnError_NoQualityField
// to force a measurably large rtt_ms without depending on real network jitter.
func startSlowPingDaemon(t *testing.T, sockPath string, responseData json.RawMessage, delay time.Duration) net.Listener {
	t.Helper()

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("startSlowPingDaemon: listen on %s: %v", sockPath, err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer func() { _ = c.Close() }()
				_ = c.SetDeadline(time.Now().Add(10 * time.Second))

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

				var req map[string]interface{}
				if err := json.NewDecoder(c).Decode(&req); err != nil {
					return
				}
				reqID, _ := req["id"].(string)

				// Artificial delay BEFORE the RPC response — this is what
				// forces rtt_ms (measured dial-start to response-decode-
				// complete, client-side) to be measurably large.
				time.Sleep(delay)

				rpcResp := map[string]interface{}{
					"type": "response",
					"id":   reqID,
					"ok":   true,
					"data": responseData,
				}
				_ = json.NewEncoder(c).Encode(rpcResp)
			}(conn)
		}
	}()
	return ln
}

// startAuthFailDaemon starts a stub daemon that sends a well-formed CHALLENGE,
// reads the CHALLENGE_RESPONSE, then replies AUTH_FAIL with code E-ADM-010 —
// mirroring TestSbctl_AuthFailure_ExitsOneWithEADM010's hand-rolled server in
// main_test.go. No caller needs the listener handle — cleanup is registered
// internally via t.Cleanup.
func startAuthFailDaemon(t *testing.T, sockPath string) {
	t.Helper()

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("startAuthFailDaemon: listen on %s: %v", sockPath, err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

		challenge := `{"type":"challenge","nonce":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","daemon_sig":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}` + "\n"
		if _, err := conn.Write([]byte(challenge)); err != nil {
			return
		}
		buf := make([]byte, 4096)
		_, _ = conn.Read(buf)
		authFail := `{"type":"auth_fail","code":"E-ADM-010","message":"authentication failed"}` + "\n"
		_, _ = conn.Write([]byte(authFail))
	}()
}

// pingEnvelope decodes sbctl's stdout JSON envelope for a successful
// `paths ping` invocation and returns the "data" object's fields.
func pingEnvelope(t *testing.T, stdout string) map[string]json.RawMessage {
	t.Helper()
	var env struct {
		OK   bool            `json:"ok"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &env); err != nil {
		t.Fatalf("stdout is not a valid JSON envelope: %v\nraw: %q", err, stdout)
	}
	if !env.OK {
		t.Fatal("envelope ok must be true for a successful call")
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal(env.Data, &data); err != nil {
		t.Fatalf("envelope data is not a JSON object: %v\nraw data: %s", err, env.Data)
	}
	return data
}

// ─── AC-001: sbctl paths ping happy path ─────────────────────────────────────

// TestPathsPing_HappyPath_ReportsRTT verifies that `sbctl paths ping
// --router=<addr>` dials <addr> directly (overriding --target), issues
// paths.ping with empty args, and reports {"router": "<addr>", "rtt_ms":
// <float64>} with exit 0.
//
// AC-001 / BC-2.06.004 PC-1, Invariant 1.
func TestPathsPing_HappyPath_ReportsRTT(t *testing.T) {
	t.Parallel()

	sockPath, cleanup := stubDaemonSocket(t)
	defer cleanup()

	gotCmdCh := make(chan string, 1)
	_ = startCannedDaemonAssertCmd(t, sockPath, json.RawMessage(`{"pong":true}`), "paths.ping", gotCmdCh)

	// --target is deliberately a different (bogus) value than sockPath — the
	// --router=<addr> flag must override it (PC-1 "dials <addr> directly,
	// overriding --target").
	exitCode, stdout, stderr := runProductionMain(t,
		"--target", "/nonexistent/should-be-overridden.sock", "--key", testdataKeyPath(t),
		"paths", "ping", "--router="+sockPath,
	)
	if exitCode != 0 {
		t.Fatalf("AC-001: expected exit code 0, got %d\nstdout: %q\nstderr: %q", exitCode, stdout, stderr)
	}

	select {
	case gotCmd := <-gotCmdCh:
		if gotCmd != "paths.ping" {
			t.Errorf("AC-001: sbctl sent RPC command %q; want %q", gotCmd, "paths.ping")
		}
	default:
		t.Error("AC-001: no RPC command received by canned daemon — channel empty")
	}

	data := pingEnvelope(t, stdout)

	// "router" must equal the dialed address (BC-2.06.004 PC-1) — proves the
	// override took effect, not the bogus default target.
	assertJSONString(t, data, "router", sockPath)

	rawRTT, ok := data["rtt_ms"]
	if !ok {
		t.Fatal("AC-001: response missing rtt_ms field")
	}
	var rtt float64
	if err := json.Unmarshal(rawRTT, &rtt); err != nil {
		t.Fatalf("AC-001: rtt_ms is not numeric: %v (raw: %s)", err, rawRTT)
	}
	if rtt < 0 {
		t.Errorf("AC-001: rtt_ms = %v; want >= 0", rtt)
	}
}

// ─── AC-002: sbctl paths ping error paths ────────────────────────────────────

// TestPathsPing_Unreachable_ENET001 verifies that when the target daemon is
// unreachable before connection, sbctl reports E-NET-001 "daemon unreachable:
// <address>" and exits 1 (operational error).
//
// AC-002 / BC-2.06.004 PC-2, EC-001.
func TestPathsPing_Unreachable_ENET001(t *testing.T) {
	t.Parallel()

	sockPath := "/nonexistent/path/to/daemon-" + t.Name() + ".sock"

	exitCode, stdout, stderr := runProductionMain(t,
		"--target", "127.0.0.1:19986", "--key", testdataKeyPath(t),
		"paths", "ping", "--router="+sockPath,
	)
	if exitCode != 1 {
		t.Errorf("AC-002 / BC-2.06.004 PC-2: expected exit code 1, got %d\nstdout: %q\nstderr: %q", exitCode, stdout, stderr)
	}
	if !strings.Contains(stderr, "E-NET-001") {
		t.Errorf("AC-002 / BC-2.06.004 PC-2: expected stderr to contain \"E-NET-001\"; got: %q", stderr)
	}
	if !strings.Contains(stderr, sockPath) {
		t.Errorf("AC-002: expected stderr to name the unreachable address %q; got: %q", sockPath, stderr)
	}
}

// TestPathsPing_AuthFailure_EADM010 verifies that when the connection
// succeeds but Tier-1 authentication fails, sbctl reports E-ADM-010 and exits
// 1 — no paths.ping RPC is ever dispatched (auth failure occurs before
// command dispatch).
//
// AC-002 / BC-2.06.004 PC-3, EC-002.
func TestPathsPing_AuthFailure_EADM010(t *testing.T) {
	t.Parallel()

	sockPath, cleanup := stubDaemonSocket(t)
	defer cleanup()

	startAuthFailDaemon(t, sockPath)

	exitCode, stdout, stderr := runProductionMain(t,
		"--target", "127.0.0.1:19987", "--key", testdataKeyPath(t),
		"paths", "ping", "--router="+sockPath,
	)
	if exitCode != 1 {
		t.Errorf("AC-002 / BC-2.06.004 PC-3: expected exit code 1, got %d\nstdout: %q\nstderr: %q", exitCode, stdout, stderr)
	}
	if !strings.Contains(stderr, "E-ADM-010") {
		t.Errorf("AC-002 / BC-2.06.004 PC-3: expected stderr to contain \"E-ADM-010\"; got: %q", stderr)
	}
}

// ─── AC-003: slow round trip is not an error; no quality classification ─────

// TestPathsPing_SlowRoundTrip_NotAnError_NoQualityField verifies that a
// connection which succeeds but measures high latency is NOT an error —
// rtt_ms simply reports the larger measured value, exit 0 — and that neither
// paths.ping's response nor sbctl's synthesized output ever carries a
// quality/status classification field (that remains router.status's job).
//
// AC-003 / BC-2.06.004 PC-4, EC-003, Invariant 2.
func TestPathsPing_SlowRoundTrip_NotAnError_NoQualityField(t *testing.T) {
	t.Parallel()

	sockPath, cleanup := stubDaemonSocket(t)
	defer cleanup()

	const injectedDelay = 250 * time.Millisecond
	_ = startSlowPingDaemon(t, sockPath, json.RawMessage(`{"pong":true}`), injectedDelay)

	exitCode, stdout, stderr := runProductionMain(t,
		"--target", "127.0.0.1:19989", "--key", testdataKeyPath(t),
		"paths", "ping", "--router="+sockPath,
	)
	if exitCode != 0 {
		t.Fatalf("AC-003: slow round trip must not be an error; expected exit 0, got %d\nstdout: %q\nstderr: %q", exitCode, stdout, stderr)
	}

	data := pingEnvelope(t, stdout)

	rawRTT, ok := data["rtt_ms"]
	if !ok {
		t.Fatal("AC-003: response missing rtt_ms field")
	}
	var rtt float64
	if err := json.Unmarshal(rawRTT, &rtt); err != nil {
		t.Fatalf("AC-003: rtt_ms is not numeric: %v", err)
	}
	// rtt_ms must reflect the injected server-side delay — proves the
	// measurement spans dial-start to response-decode-complete, and that a
	// large value is reported as data, not converted into an error.
	if rtt < float64(injectedDelay.Milliseconds())/2 {
		t.Errorf("AC-003: rtt_ms = %v; want a value reflecting the injected %s delay", rtt, injectedDelay)
	}

	// Invariant 2 / PC-4: no quality/status classification field anywhere in
	// the response — paths.ping performs no quality classification.
	for _, forbidden := range []string{"quality", "status"} {
		if _, present := data[forbidden]; present {
			t.Errorf("AC-003 / Invariant 2: response must not carry a %q field (quality classification remains router.status's job); present keys: %v", forbidden, mapKeys(data))
		}
	}
}
