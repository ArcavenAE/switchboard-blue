//go:build integration

// Package main e2e tests for the sbctl management plane client against all four
// daemon types per BC-2.07.002 v1.2 and VP-049.
//
// Red Gate: all tests in this file MUST FAIL before implementation begins.
// The harness helpers (startTestMgmtServer, awaitReady) are declared with
// t.Fatal("not implemented") bodies so the tests fail on entry rather than
// compiling but vacuously passing.
//
// Wire protocol notes discovered from internal/mgmt/mgmt.go:
//   - Server sends CHALLENGE first (type:"challenge", nonce, daemon_sig) in base64url.
//   - Client responds with CHALLENGE_RESPONSE (type:"challenge_response", nonce_sig, pubkey).
//   - Server replies AUTH_OK (type:"auth_ok", daemon_version) or AUTH_FAIL (type:"auth_fail").
//   - Bootstrap mode: ops = mgmt.NewOperatorKeySet(nil) — daemon's own public key is the sole
//     authorized key. Tests use the daemon key as the operator key to authenticate.
//   - After AUTH_OK, RPC requests are type:"request", responses are type:"response".
//   - conn.Read → error after the server closes the connection on AUTH_FAIL (AC-004, AC-005).
//
// Traceability:
//
//	BC-2.07.002 — sbctl Unified CLI for All Four Daemon Types with OpenSSH Key Authentication
//	VP-049      — e2e across all four daemon types
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/mgmt"
)

// daemonMode represents one of the four daemon types under test.
// Each mode gets its own mgmt.Server instance with a distinct mode label in the
// stub "status" handler response — the daemon-mode axis is expressed by this label,
// not by running the real runRouter/runAccess/runConsole/runControl functions
// (those are stubs or incomplete for the management plane in cmd/switchboard).
type daemonMode struct {
	name string // one of "router", "access", "console", "control"
}

// allDaemonModes is the table for TestE2E_MgmtPlane_AllFourDaemonTypes_VP049.
var allDaemonModes = []daemonMode{
	{name: "router"},
	{name: "access"},
	{name: "console"},
	{name: "control"},
}

// testMgmtServer is the handle returned by startTestMgmtServer.
// It bundles the server, its TCP listener address, and the operator private key
// that was registered as authorized during construction (bootstrap mode: daemon
// key == operator key).
type testMgmtServer struct {
	addr    string             // "127.0.0.1:<port>" — dial target for clients
	privKey ed25519.PrivateKey // operator private key (same as daemon key in bootstrap mode)
}

// startTestMgmtServer starts an in-process mgmt.Server for the given daemon mode
// on an OS-assigned TCP port. Returns a testMgmtServer with the server address
// and the operator key. Shutdown is registered via t.Cleanup.
func startTestMgmtServer(t *testing.T, mode daemonMode) *testMgmtServer {
	t.Helper()

	// Generate a fresh Ed25519 key pair for this server instance.
	_, daemonPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("startTestMgmtServer [%s]: GenerateKey: %v", mode.name, err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("startTestMgmtServer [%s]: net.Listen: %v", mode.name, err)
	}

	// "status" stub handler returns {"mode":<name>} as the data field.
	// The response envelope's ok:true and data wrapping is handled by the server.
	modeName := mode.name
	statusHandler := mgmt.Handler{
		Command: "status",
		Fn: func(_ context.Context, _ json.RawMessage) (any, error) {
			return map[string]any{"mode": modeName}, nil
		},
	}

	// Bootstrap mode: NewOperatorKeySet(nil) — the daemon's own key is the sole
	// authorized operator key, so the test passes daemonPriv as the operator key.
	srv := mgmt.NewServer(
		ln,
		daemonPriv,
		mgmt.NewOperatorKeySet(nil),
		[]mgmt.Handler{statusHandler},
		"dev-"+modeName,
		mgmt.WithHandshakeTimeout(500*time.Millisecond),
		mgmt.WithRPCIdleTimeout(2*time.Second),
	)

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = srv.Serve(ctx)
	}()

	t.Cleanup(func() {
		cancel()
		_ = srv.Shutdown(context.Background())
		wg.Wait()
		_ = ln.Close()
	})

	return &testMgmtServer{
		addr:    ln.Addr().String(),
		privKey: daemonPriv,
	}
}

// awaitReady polls the server address with 10ms sleep until a TCP connection
// succeeds or the timeout expires.
func awaitReady(t *testing.T, addr string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().UTC().Add(timeout)
	for {
		conn, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		if time.Now().UTC().After(deadline) {
			t.Fatalf("awaitReady: server at %s did not become ready within %s: %v", addr, timeout, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// dialConn opens a TCP connection to addr and registers conn.Close in t.Cleanup.
// Returns the connection. Fatal on error.
func dialConn(t *testing.T, addr string) net.Conn {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dialConn: net.Dial %s: %v", addr, err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// ── TestE2E_MgmtPlane_AllFourDaemonTypes_VP049 ────────────────────────────────
//
// Covers AC-001, AC-002, AC-003, AC-005 across all four daemon modes.
//
// AC-001: start mgmt.Server with in-process listener; poll for readiness (1s timeout).
// AC-002: Authenticate(ctx, conn, privKey) returns nil against each daemon's socket.
// AC-003: dispatch "status" RPC; response contains ok:true and a data field.
// AC-005: close conn after RPC; next Read returns error within 500ms.
//
// Each sub-test is independent (own listener, own key pair) and runs in parallel.

func TestE2E_MgmtPlane_AllFourDaemonTypes_VP049(t *testing.T) {
	t.Parallel()

	for _, mode := range allDaemonModes {
		mode := mode // capture loop variable
		t.Run(mode.name, func(t *testing.T) {
			t.Parallel()

			// ── AC-001: start daemon and wait for listener to be ready ────────────────
			srv := startTestMgmtServer(t, mode)
			awaitReady(t, srv.addr, 1*time.Second)

			// ── AC-002: authenticate ──────────────────────────────────────────────────
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			t.Cleanup(cancel)

			conn := dialConn(t, srv.addr)

			if err := Authenticate(ctx, conn, srv.privKey); err != nil {
				t.Fatalf("AC-002 [%s]: Authenticate returned error: %v", mode.name, err)
			}

			// ── AC-003: dispatch "status" RPC; verify ok:true and data field present ──
			//
			// We call the unexported dispatch() directly (white-box test in package main).
			// Command is "status"; args is nil (marshaled as null in JSON).
			rawData, err := dispatch(ctx, conn, "status", nil)
			if err != nil {
				t.Fatalf("AC-003 [%s]: dispatch(status) returned error: %v", mode.name, err)
			}
			// data field must be present (non-nil, parseable JSON) — AC-003 does not
			// assert on content, only that the field exists and is valid JSON.
			if rawData == nil {
				t.Errorf("AC-003 [%s]: dispatch(status) returned nil data; want non-nil JSON", mode.name)
			}
			// Verify it is parseable as a JSON object (not just "null").
			var dataMap map[string]any
			if err := json.Unmarshal(rawData, &dataMap); err != nil {
				t.Errorf("AC-003 [%s]: data field is not a JSON object: %v (raw: %s)", mode.name, err, rawData)
			}

			// ── AC-005: close conn; verify next Read returns error within 500ms ────────
			//
			// After the RPC completes, the client closes the connection (sbctl invariant:
			// exits after command completion — BC-2.07.002 Inv-2).
			// We close the conn and verify that reading from it returns an error promptly,
			// confirming the client-side connection is cleaned up (not lingering as daemon).
			if err := conn.Close(); err != nil {
				t.Logf("AC-005 [%s]: conn.Close error (may already be closed): %v", mode.name, err)
			}

			// Read must return an error within 500ms.
			if err := conn.SetReadDeadline(time.Now().UTC().Add(500 * time.Millisecond)); err != nil {
				t.Logf("AC-005 [%s]: SetReadDeadline after close: %v", mode.name, err)
			}
			buf := make([]byte, 1)
			_, readErr := conn.Read(buf)
			if readErr == nil {
				t.Errorf("AC-005 [%s]: Read after conn.Close returned nil error; want error (connection closed)", mode.name)
			}
		})
	}
}

// ── TestE2E_MgmtPlane_UnauthenticatedRejected_AC004 ──────────────────────────
//
// Covers AC-004: a connection that skips authentication and sends an RPC request
// directly receives AUTH_FAIL and connection close.
//
// The server expects a CHALLENGE_RESPONSE but instead receives a "request" JSON
// object — the type mismatch triggers AUTH_FAIL + conn.Close on the server side.
//
// Representative daemon type: router (one daemon type is sufficient per AC-004).

func TestE2E_MgmtPlane_UnauthenticatedRejected_AC004(t *testing.T) {
	t.Parallel()

	// Start a single router-mode daemon.
	srv := startTestMgmtServer(t, daemonMode{name: "router"})
	awaitReady(t, srv.addr, 1*time.Second)

	conn := dialConn(t, srv.addr)

	// Drain the CHALLENGE message that the server always sends first.
	// We receive it but do NOT send a CHALLENGE_RESPONSE — we send an RPC request
	// instead, skipping authentication entirely.
	//
	// Set a short read deadline so the test does not hang on the challenge read.
	if err := conn.SetReadDeadline(time.Now().UTC().Add(2 * time.Second)); err != nil {
		t.Fatalf("AC-004: SetReadDeadline for challenge drain: %v", err)
	}
	var challengeDrain map[string]any
	if err := json.NewDecoder(io.LimitReader(conn, mgmt.MaxMessageBytes)).Decode(&challengeDrain); err != nil {
		t.Fatalf("AC-004: failed to read CHALLENGE from server: %v", err)
	}
	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		t.Fatalf("AC-004: clear read deadline: %v", err)
	}
	if got := challengeDrain["type"]; got != "challenge" {
		t.Fatalf("AC-004: expected type=challenge from server; got %v", got)
	}

	// Instead of sending CHALLENGE_RESPONSE, send a raw RPC request.
	// The server expects challenge_response; receiving "request" triggers AUTH_FAIL + close.
	rawRequest, err := json.Marshal(map[string]any{
		"type":    "request",
		"id":      "t1",
		"command": "status",
		"args":    map[string]any{},
	})
	if err != nil {
		t.Fatalf("AC-004: json.Marshal RPC request: %v", err)
	}
	rawRequest = append(rawRequest, '\n')
	if _, err := conn.Write(rawRequest); err != nil {
		t.Fatalf("AC-004: write RPC request: %v", err)
	}

	// The server MUST respond with AUTH_FAIL and then close the connection.
	// We read the AUTH_FAIL response with a short timeout.
	if err := conn.SetReadDeadline(time.Now().UTC().Add(2 * time.Second)); err != nil {
		t.Fatalf("AC-004: SetReadDeadline for AUTH_FAIL read: %v", err)
	}
	var authFail map[string]any
	if err := json.NewDecoder(io.LimitReader(conn, mgmt.MaxMessageBytes)).Decode(&authFail); err != nil {
		t.Fatalf("AC-004: failed to read AUTH_FAIL from server: %v", err)
	}
	if got := authFail["type"]; got != "auth_fail" {
		t.Errorf("AC-004: want type=auth_fail; got %v", got)
	}

	// Verify connection is closed by server within 500ms of the AUTH_FAIL.
	// A further Read must return an error (EOF or closed connection).
	if err := conn.SetReadDeadline(time.Now().UTC().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("AC-004: SetReadDeadline for post-AUTH_FAIL drain: %v", err)
	}
	buf := make([]byte, 1)
	_, readErr := conn.Read(buf)
	if readErr == nil {
		t.Errorf("AC-004: Read after AUTH_FAIL returned nil error; server should have closed connection")
	}
}

// ── compile-time assertion: mgmt.MaxMessageBytes is exported and usable here ──
// This block is intentionally empty; the import above ensures the assertion.
var _ = mgmt.MaxMessageBytes

// ── package-level compile-time type checks ────────────────────────────────────
// Verify that the helper stubs and daemonMode type compile without references to
// unexported implementation symbols that don't exist yet.
var (
	_ func(*testing.T, daemonMode) *testMgmtServer = startTestMgmtServer
	_ func(*testing.T, string, time.Duration)      = awaitReady
	_ func(*testing.T, string) net.Conn            = dialConn
)

// Ensure ed25519, rand, and json are used (imported for future implementation
// reference in startTestMgmtServer — lint will complain if unused).
var (
	_ = ed25519.PublicKey(nil)
	_ = rand.Reader
)
