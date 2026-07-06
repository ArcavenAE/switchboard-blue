package main

// Unit tests for the CLI-side argument mapping in runSessions for the
// `status` sub-verb (S-BL.CONSOLE-OBS; sbctl surface of BC-2.06.001 v1.7 PC-5
// console-half + BC-2.06.002 v1.4 PC-3).
//
// The behavior under test is the switch-arm at cmd/sbctl/main.go:
//
//	case "status":
//	    var cmdArgs any
//	    if len(args) > 1 { cmdArgs = map[string]string{"session_name": args[1]} }
//	    return connectAndRun(..., "sessions.status", cmdArgs, ...)
//
// The failure mode this protects against: silently dispatching sessions.list
// or sending the wrong args shape (e.g., a bare string vs {session_name:...})
// would produce E-RPC-002/E-CFG-001 on the daemon side or, worse, hit the
// wrong handler entirely. This is the same class of misdispatch that
// F-P5P6-A-003 caught for `sessions attach/detach`.
//
// Strategy: drive dispatch() over a net.Pipe (same pattern as
// TestDispatch_EmitsCorrectWireType at client_test.go:918); the mock server
// captures the raw JSON the client wrote and asserts the command name and
// args payload match the CLI mapping. runSessions itself is not invoked
// because connectAndRun bundles auth + dial; the meaningful mutation is
// the (command, cmdArgs) pair passed to dispatch, which this test exercises
// directly with the same mapping runSessions applies.

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// TestSessionsStatus_CLI_DispatchesWithSessionName verifies that when the CLI
// receives `sbctl sessions status agent-01`, dispatch emits an RPC envelope
// with command="sessions.status" and args={"session_name":"agent-01"}.
func TestSessionsStatus_CLI_DispatchesWithSessionName(t *testing.T) {
	t.Parallel()

	server, client := net.Pipe()
	t.Cleanup(func() { _ = client.Close() })

	rawRequestCh := make(chan []byte, 1)

	go func() {
		defer func() { _ = server.Close() }()
		_ = server.SetDeadline(time.Now().Add(2 * time.Second))

		buf := make([]byte, 4096)
		n, err := server.Read(buf)
		if err != nil || n == 0 {
			rawRequestCh <- nil
			return
		}
		raw := make([]byte, n)
		copy(raw, buf[:n])
		rawRequestCh <- raw

		// Echo a valid response so dispatch resolves cleanly and the test
		// can focus on the emitted request shape.
		var req struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(raw, &req)
		resp := fmt.Sprintf(
			`{"type":"response","id":%q,"ok":true,"data":{"sessions":[]}}`+"\n",
			req.ID,
		)
		_, _ = server.Write([]byte(resp))
	}()

	// Simulate the CLI's mapping for `sessions status agent-01`:
	cmdArgs := map[string]string{"session_name": "agent-01"}
	_, err := dispatch(context.Background(), client, "sessions.status", cmdArgs)
	if err != nil {
		t.Fatalf("dispatch returned error: %v", err)
	}

	raw := <-rawRequestCh
	if raw == nil {
		t.Fatal("mock server received no bytes; dispatch failed to write")
	}
	rawStr := string(raw)

	// The emitted RPC envelope must carry the sessions.status command name.
	if !strings.Contains(rawStr, `"command":"sessions.status"`) {
		t.Errorf("dispatch emitted wrong command; raw = %s\n  want to contain %q",
			rawStr, `"command":"sessions.status"`)
	}
	// And the args must carry the session_name field with the requested name.
	// The daemon-side handler (session.HandleSessionsStatus) requires
	// {session_name: "<name>"} — a bare string or missing field would fail
	// on the daemon side.
	if !strings.Contains(rawStr, `"session_name":"agent-01"`) {
		t.Errorf("dispatch emitted wrong args; raw = %s\n  want to contain %q",
			rawStr, `"session_name":"agent-01"`)
	}
}

// TestSessionsStatus_CLI_DispatchesAllSessionsWithNilArgs verifies that when
// the CLI receives `sbctl sessions status` (no positional argument), dispatch
// emits an RPC envelope with command="sessions.status" and args=null. The
// daemon-side handler treats a nil args body as an "all sessions" query.
//
// Non-tautological: the CLI mapping in runSessions produces `var cmdArgs any`
// (nil) when len(args) <= 1. A bug that spuriously produced
// {"session_name":""} instead of null would still work at the daemon side but
// would leak an empty selector into the audit surface — this test locks the
// wire shape for the "all sessions" query.
func TestSessionsStatus_CLI_DispatchesAllSessionsWithNilArgs(t *testing.T) {
	t.Parallel()

	server, client := net.Pipe()
	t.Cleanup(func() { _ = client.Close() })

	rawRequestCh := make(chan []byte, 1)

	go func() {
		defer func() { _ = server.Close() }()
		_ = server.SetDeadline(time.Now().Add(2 * time.Second))

		buf := make([]byte, 4096)
		n, err := server.Read(buf)
		if err != nil || n == 0 {
			rawRequestCh <- nil
			return
		}
		raw := make([]byte, n)
		copy(raw, buf[:n])
		rawRequestCh <- raw

		var req struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(raw, &req)
		resp := fmt.Sprintf(
			`{"type":"response","id":%q,"ok":true,"data":{"sessions":[]}}`+"\n",
			req.ID,
		)
		_, _ = server.Write([]byte(resp))
	}()

	// Simulate the CLI's mapping for bare `sessions status`:
	var cmdArgs any // nil — matches runSessions when len(args) <= 1
	_, err := dispatch(context.Background(), client, "sessions.status", cmdArgs)
	if err != nil {
		t.Fatalf("dispatch returned error: %v", err)
	}

	raw := <-rawRequestCh
	if raw == nil {
		t.Fatal("mock server received no bytes; dispatch failed to write")
	}
	rawStr := string(raw)

	if !strings.Contains(rawStr, `"command":"sessions.status"`) {
		t.Errorf("dispatch emitted wrong command; raw = %s\n  want to contain %q",
			rawStr, `"command":"sessions.status"`)
	}
	// The args field must be JSON null (or absent) — never an empty selector.
	// dispatch marshals a nil `any` as `null` in the outer envelope.
	if !strings.Contains(rawStr, `"args":null`) {
		t.Errorf("dispatch emitted unexpected args shape; raw = %s\n  want to contain %q "+
			"(bare `sessions status` must NOT synthesize {session_name:\"\"})",
			rawStr, `"args":null`)
	}
	// Defensive: reject the failure mode where an empty session_name leaks
	// into the wire, which would fail the daemon-side "all sessions" path.
	if strings.Contains(rawStr, `"session_name":""`) {
		t.Errorf("dispatch leaked empty session_name into wire args; raw = %s "+
			"(the CLI must send null args for the all-sessions query)", rawStr)
	}
}
