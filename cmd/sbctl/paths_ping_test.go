// paths_ping_test.go — integration tests for `sbctl paths ping --router=<addr>`.
//
// BC/AC coverage map:
//
//	TestPathsPing_HappyPath_ReportsRTT               → AC-001, BC-2.06.004 PC-1
//	TestPathsPing_Unreachable_ENET001                → AC-002, BC-2.06.004 PC-2, EC-001
//	TestPathsPing_AuthFailure_EADM010                → AC-002, BC-2.06.004 PC-3, EC-002
//	TestPathsPing_SlowRoundTrip_NotAnError_NoQualityField → AC-003, BC-2.06.004 PC-4, EC-003, Invariant 2
//
// runPathsPing is implemented (no longer a Red Gate stub). All four tests
// below still dispatch through the real compiled main() via the
// runProductionMain subprocess helper (production_exit_code_test.go) —
// retained as a regression defense: an unrecovered handler panic terminates
// the whole process (testing's per-test recover only guards the goroutine
// running t.Run, not sibling tests), so subprocess isolation contains any
// future regression to the child process's own exit code rather than taking
// every unrelated test in this package down with it. Matches this repo's
// established pattern for exercising panic-risk dispatch paths
// (main_test.go's TestSubprocessMain_* hooks). The stub daemons below still
// run in-process in the PARENT test binary — only the client half (sbctl's
// own main()) is subprocess-isolated.
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
func startSlowPingDaemon(t *testing.T, sockPath string, responseData json.RawMessage, delay time.Duration) net.Listener { //nolint:unparam // return value unused at call sites; kept for potential future use in concurrent test scenarios (matches startCannedDaemon's established pattern, router_status_test.go)
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

// pingBareData decodes sbctl's stdout as the AC-001 PC-4 bare data object
// directly — the default (no --json) output shape after F-CS-I4-001's
// remediation (runPathsPing previously hardcoded useJSON=true and always
// emitted the {"ok":...,"data":...} envelope regardless of --json). Fails
// if stdout carries an "ok" or "data" key, which would mean the envelope
// leaked into default-mode output.
func pingBareData(t *testing.T, stdout string) map[string]json.RawMessage {
	t.Helper()
	var data map[string]json.RawMessage
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &data); err != nil {
		t.Fatalf("F-CS-I4-001: default-mode stdout is not a JSON object: %v\nraw: %q", err, stdout)
	}
	for _, envelopeKey := range []string{"ok", "data"} {
		if _, present := data[envelopeKey]; present {
			t.Errorf("F-CS-I4-001: default-mode stdout must not carry envelope key %q "+
				"(AC-001 PC-4 bare shape); got: %s", envelopeKey, stdout)
		}
	}
	return data
}

// decodeRTTMs extracts and validates the numeric rtt_ms field from data.
// Shared by AC-001 (happy path) and AC-003 (slow round trip) subtests, each
// of which applies its own bound check against the returned value.
func decodeRTTMs(t *testing.T, data map[string]json.RawMessage, acLabel string) float64 {
	t.Helper()
	rawRTT, ok := data["rtt_ms"]
	if !ok {
		t.Fatalf("%s: response missing rtt_ms field", acLabel)
	}
	var rtt float64
	if err := json.Unmarshal(rawRTT, &rtt); err != nil {
		t.Fatalf("%s: rtt_ms is not numeric: %v (raw: %s)", acLabel, err, rawRTT)
	}
	return rtt
}

// assertPathsPingCommand asserts the canned daemon observed a "paths.ping"
// RPC command on gotCmdCh. Shared by the AC-001 default/--json subtests.
func assertPathsPingCommand(t *testing.T, gotCmdCh <-chan string) {
	t.Helper()
	select {
	case gotCmd := <-gotCmdCh:
		if gotCmd != "paths.ping" {
			t.Errorf("AC-001: sbctl sent RPC command %q; want %q", gotCmd, "paths.ping")
		}
	default:
		t.Error("AC-001: no RPC command received by canned daemon — channel empty")
	}
}

// ─── AC-001: sbctl paths ping happy path ─────────────────────────────────────

// TestPathsPing_HappyPath_ReportsRTT verifies that `sbctl paths ping
// --router=<addr>` dials <addr> directly (overriding --target), issues
// paths.ping with empty args, and reports {"router": "<addr>", "rtt_ms":
// <float64>} with exit 0.
//
// Two subtests cover both output modes (F-CS-I4-001: runPathsPing previously
// hardcoded useJSON=true, always emitting the {"ok":...,"data":...} envelope
// regardless of --json — interface-definitions.md v1.31 §214 requires JSON
// output only when --json is present):
//
//   - default_bare_data: no --json → stdout is the bare AC-001 PC-4 object
//     at top level, no envelope wrapper.
//   - json_flag_envelope: --json → stdout is the {"ok":true,"data":{...}}
//     envelope, and only then.
//
// AC-001 / BC-2.06.004 PC-1, Invariant 1.
func TestPathsPing_HappyPath_ReportsRTT(t *testing.T) {
	t.Run("default_bare_data", func(t *testing.T) {
		t.Parallel()

		sockPath, cleanup := stubDaemonSocket(t)
		defer cleanup()

		gotCmdCh := make(chan string, 1)
		_ = startCannedDaemonAssertCmd(t, sockPath, json.RawMessage(`{"pong":true}`), "paths.ping", gotCmdCh)

		// --target is deliberately a different (bogus) value than sockPath —
		// the --router=<addr> flag must override it (PC-1 "dials <addr>
		// directly, overriding --target").
		exitCode, stdout, stderr := runProductionMain(t,
			"--target", "/nonexistent/should-be-overridden.sock", "--key", testdataKeyPath(t),
			"paths", "ping", "--router="+sockPath,
		)
		if exitCode != 0 {
			t.Fatalf("AC-001: expected exit code 0, got %d\nstdout: %q\nstderr: %q", exitCode, stdout, stderr)
		}

		assertPathsPingCommand(t, gotCmdCh)

		data := pingBareData(t, stdout)

		// "router" must equal the dialed address (BC-2.06.004 PC-1) — proves
		// the override took effect, not the bogus default target.
		assertJSONString(t, data, "router", sockPath)

		if rtt := decodeRTTMs(t, data, "AC-001"); rtt < 0 {
			t.Errorf("AC-001: rtt_ms = %v; want >= 0", rtt)
		}
	})

	t.Run("json_flag_envelope", func(t *testing.T) {
		t.Parallel()

		sockPath, cleanup := stubDaemonSocket(t)
		defer cleanup()

		gotCmdCh := make(chan string, 1)
		_ = startCannedDaemonAssertCmd(t, sockPath, json.RawMessage(`{"pong":true}`), "paths.ping", gotCmdCh)

		exitCode, stdout, stderr := runProductionMain(t,
			"--target", "/nonexistent/should-be-overridden.sock", "--key", testdataKeyPath(t), "--json",
			"paths", "ping", "--router="+sockPath,
		)
		if exitCode != 0 {
			t.Fatalf("AC-001: expected exit code 0, got %d\nstdout: %q\nstderr: %q", exitCode, stdout, stderr)
		}

		assertPathsPingCommand(t, gotCmdCh)

		data := pingEnvelope(t, stdout)

		assertJSONString(t, data, "router", sockPath)

		if rtt := decodeRTTMs(t, data, "AC-001"); rtt < 0 {
			t.Errorf("AC-001: rtt_ms = %v; want >= 0", rtt)
		}
	})
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
// Two subtests cover both output modes (F-CS-I4-001, see
// TestPathsPing_HappyPath_ReportsRTT's doc comment for the full rationale):
// default_bare_data (no --json, bare object) and json_flag_envelope
// (--json, envelope).
//
// AC-003 / BC-2.06.004 PC-4, EC-003, Invariant 2.
func TestPathsPing_SlowRoundTrip_NotAnError_NoQualityField(t *testing.T) {
	const injectedDelay = 250 * time.Millisecond

	assertSlowPingData := func(t *testing.T, data map[string]json.RawMessage) {
		t.Helper()
		// rtt_ms must reflect the injected server-side delay — proves the
		// measurement spans dial-start to response-decode-complete, and that
		// a large value is reported as data, not converted into an error.
		if rtt := decodeRTTMs(t, data, "AC-003"); rtt < float64(injectedDelay.Milliseconds())/2 {
			t.Errorf("AC-003: rtt_ms = %v; want a value reflecting the injected %s delay", rtt, injectedDelay)
		}

		// Invariant 2 / PC-4: no quality/status classification field
		// anywhere in the response — paths.ping performs no quality
		// classification.
		for _, forbidden := range []string{"quality", "status"} {
			if _, present := data[forbidden]; present {
				t.Errorf("AC-003 / Invariant 2: response must not carry a %q field "+
					"(quality classification remains router.status's job); present keys: %v",
					forbidden, mapKeys(data))
			}
		}
	}

	t.Run("default_bare_data", func(t *testing.T) {
		t.Parallel()

		sockPath, cleanup := stubDaemonSocket(t)
		defer cleanup()

		_ = startSlowPingDaemon(t, sockPath, json.RawMessage(`{"pong":true}`), injectedDelay)

		exitCode, stdout, stderr := runProductionMain(t,
			"--target", "127.0.0.1:19989", "--key", testdataKeyPath(t),
			"paths", "ping", "--router="+sockPath,
		)
		if exitCode != 0 {
			t.Fatalf("AC-003: slow round trip must not be an error; expected exit 0, got %d\nstdout: %q\nstderr: %q", exitCode, stdout, stderr)
		}

		assertSlowPingData(t, pingBareData(t, stdout))
	})

	t.Run("json_flag_envelope", func(t *testing.T) {
		t.Parallel()

		sockPath, cleanup := stubDaemonSocket(t)
		defer cleanup()

		_ = startSlowPingDaemon(t, sockPath, json.RawMessage(`{"pong":true}`), injectedDelay)

		exitCode, stdout, stderr := runProductionMain(t,
			"--target", "127.0.0.1:19989", "--key", testdataKeyPath(t), "--json",
			"paths", "ping", "--router="+sockPath,
		)
		if exitCode != 0 {
			t.Fatalf("AC-003: slow round trip must not be an error; expected exit 0, got %d\nstdout: %q\nstderr: %q", exitCode, stdout, stderr)
		}

		assertSlowPingData(t, pingEnvelope(t, stdout))
	})
}
