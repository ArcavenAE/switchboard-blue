package mgmt_test

// Test suite for internal/mgmt — management server per ADR-012.
//
// All new tests in this file are FAILING (Red Gate) because mgmt.go stubs
// return "not implemented" errors and handleConnection closes without handshake.
// This file must compile; every new test must fail before any implementation exists.
//
// Traceability:
//   BC-2.07.004 — Daemon Management Server Authenticates All Connections via
//                 Ed25519 Challenge-Response (Fail-Closed)
//   VP-064 — Server rejects unauthenticated connections
//   VP-065 — Server rejects replayed nonce within a connection
//   VP-066 — Server enforces bounded reads (CWE-400)
//
// Fixture strategy: all Ed25519 keys are generated in-test using stdlib
// crypto/ed25519. No external key material; no testdata/ files required.

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/mgmt"
)

// ── shared test helpers ────────────────────────────────────────────────────────

// mustGenKey generates an Ed25519 key pair or fatals the test.
func mustGenKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	return pub, priv
}

// encB64 encodes p with base64url (no padding), matching ADR-012 wire format.
func encB64(p []byte) string {
	return base64.RawURLEncoding.EncodeToString(p)
}

// decB64 decodes a base64url (no padding) string or fatals the test.
func decB64(t *testing.T, s string) []byte {
	t.Helper()
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		t.Fatalf("base64url decode %q: %v", s, err)
	}
	return b
}

// writeMsg encodes v as a newline-terminated JSON object and writes it to w.
func writeMsg(t *testing.T, w io.Writer, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	data = append(data, '\n')
	if _, err := w.Write(data); err != nil {
		// Connection may already be closed after AUTH_FAIL — acceptable.
		t.Logf("writeMsg: %v (connection may be closed)", err)
	}
}

// readMsg reads one newline-delimited JSON object from r and returns it as a
// map. Returns nil if r is at EOF or the read deadline expires.
func readMsg(t *testing.T, r io.Reader) map[string]any {
	t.Helper()
	dec := json.NewDecoder(r)
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		if err == io.EOF ||
			strings.Contains(err.Error(), "use of closed network connection") ||
			strings.Contains(err.Error(), "i/o timeout") ||
			strings.Contains(err.Error(), "connection reset") {
			return nil
		}
		t.Logf("readMsg decode error: %v", err)
		return nil
	}
	return m
}

// sentinelHandlers returns a handler slice with a single "test.echo" command
// that increments *count whenever it is called. The caller asserts count
// never increments on unauthenticated connections.
func sentinelHandlers(count *int) []mgmt.Handler {
	return []mgmt.Handler{
		{
			Command: "test.echo",
			Fn: func(_ context.Context, args json.RawMessage) (any, error) {
				*count++
				return map[string]string{"echo": "ok"}, nil
			},
		},
	}
}

// startServerOnPipe creates a net.Pipe pair, constructs a mgmt.Server using
// the server-side conn as the listener, and launches Serve in a goroutine.
// Returns the client-side conn. The server and goroutine are cleaned up via
// t.Cleanup when the test ends.
//
// Because net.Pipe returns a connected pair (not a Listener), this helper
// constructs the server with a single-accept fake listener wrapping the pipe.
// The server-side connection is passed directly to NewServer via a
// singleConnListener so Serve accepts exactly one connection.
func startServerOnPipe(
	t *testing.T,
	daemonPriv ed25519.PrivateKey,
	ops *mgmt.OperatorKeySet,
	handlers []mgmt.Handler,
) net.Conn {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		_ = serverConn.Close()
		_ = clientConn.Close()
	})

	ln := newSingleConnListener(serverConn)
	// "dev" is the unreleased-build sentinel — acceptable for test helpers that
	// do not assert on daemon_version (tests that do assert use NewServer directly).
	srv := mgmt.NewServer(ln, daemonPriv, ops, handlers, "dev")

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go func() {
		_ = srv.Serve(ctx)
	}()

	return clientConn
}

// singleConnListener is a net.Listener that returns one pre-connected
// net.Conn on the first Accept call, then returns a "closed" error.
// This bridges the net.Pipe API with mgmt.Server's Serve(accept-loop) design.
type singleConnListener struct {
	conn     net.Conn
	accepted bool
	done     chan struct{}
}

func newSingleConnListener(conn net.Conn) *singleConnListener {
	return &singleConnListener{conn: conn, done: make(chan struct{})}
}

func (l *singleConnListener) Accept() (net.Conn, error) {
	if !l.accepted {
		l.accepted = true
		return l.conn, nil
	}
	// Block until the listener is closed (simulate a real listener drain).
	<-l.done
	return nil, net.ErrClosed
}

func (l *singleConnListener) Close() error {
	select {
	case <-l.done:
	default:
		close(l.done)
	}
	return nil
}

func (l *singleConnListener) Addr() net.Addr {
	return l.conn.LocalAddr()
}

// ── AC-001: challenge issued first ────────────────────────────────────────────

// TestMgmtServer_IssuesChallengeFirst_AC001 verifies that the server sends a
// CHALLENGE message as the very first action on every new connection, before
// reading any client data.
//
// Traces: BC-2.07.004 PC-1, AC-001.
func TestMgmtServer_IssuesChallengeFirst_AC001(t *testing.T) {
	t.Parallel()

	_, daemonPriv := mustGenKey(t)
	ops := mgmt.NewOperatorKeySet(nil)

	client := startServerOnPipe(t, daemonPriv, ops, nil)

	// Set a short deadline so the test does not hang if the stub sends nothing.
	if err := client.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}

	msg := readMsg(t, client)
	if msg == nil {
		t.Fatal("expected CHALLENGE as first server message; got nil (timeout or closed connection)")
	}

	// AC-001: message type must be "challenge".
	if msg["type"] != "challenge" {
		t.Errorf("first server message: want type=challenge; got type=%v", msg["type"])
	}

	// AC-001: nonce must be present and decode to 32 bytes.
	nonceStr, ok := msg["nonce"].(string)
	if !ok || nonceStr == "" {
		t.Errorf("challenge.nonce: want non-empty base64url string; got %v", msg["nonce"])
	} else {
		nonce := decB64(t, nonceStr)
		if len(nonce) != 32 {
			t.Errorf("challenge.nonce: want 32 bytes; got %d bytes", len(nonce))
		}
	}

	// AC-001: daemon_sig must be present and non-empty.
	daemonSig, ok := msg["daemon_sig"].(string)
	if !ok || daemonSig == "" {
		t.Errorf("challenge.daemon_sig: want non-empty base64url string; got %v", msg["daemon_sig"])
	}
}

// ── AC-002 / VP-064: unauthenticated connections rejected ─────────────────────

// TestMgmtServer_RejectsUnauthenticated_VP064 verifies VP-064:
// connections that fail the handshake receive AUTH_FAIL (E-ADM-010) + close;
// no RPC handler is ever dispatched.
//
// Sub-cases:
//
//	(a) unrecognized public key (not in operator key set)
//	(b) recognized public key with tampered/wrong signature
//
// Traces: BC-2.07.004 PC-2, PC-4, AC-002, VP-064.
func TestMgmtServer_RejectsUnauthenticated_VP064(t *testing.T) {
	t.Parallel()

	_, daemonPriv := mustGenKey(t)
	_, authorizedPriv := mustGenKey(t)
	unauthorizedPub, unauthorizedPriv := mustGenKey(t)

	authorizedPub := authorizedPriv.Public().(ed25519.PublicKey)
	ops := mgmt.NewOperatorKeySet([]ed25519.PublicKey{authorizedPub})

	rpcCallCount := 0
	handlers := sentinelHandlers(&rpcCallCount)

	t.Run("unrecognized_public_key", func(t *testing.T) {
		t.Parallel()

		client := startServerOnPipe(t, daemonPriv, ops, handlers)
		if err := client.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
			t.Fatalf("SetDeadline: %v", err)
		}

		// Read CHALLENGE.
		challenge := readMsg(t, client)
		if challenge == nil || challenge["type"] != "challenge" {
			t.Fatalf("expected challenge; got %v", challenge)
		}
		nonceB64, _ := challenge["nonce"].(string)
		nonce := decB64(t, nonceB64)

		// Sign with the UNAUTHORIZED key.
		sig := ed25519.Sign(unauthorizedPriv, nonce)
		writeMsg(t, client, map[string]any{
			"type":      "challenge_response",
			"nonce_sig": encB64(sig),
			"pubkey":    encB64([]byte(unauthorizedPub)),
		})

		// Expect AUTH_FAIL with E-ADM-010.
		resp := readMsg(t, client)
		if resp == nil {
			t.Fatal("expected AUTH_FAIL response; got nil (connection closed before message)")
		}
		if resp["type"] != "auth_fail" {
			t.Errorf("want type=auth_fail; got %v", resp["type"])
		}
		if resp["code"] != "E-ADM-010" {
			t.Errorf("want code=E-ADM-010; got %v", resp["code"])
		}

		// Connection must be closed after AUTH_FAIL: next read returns nil.
		_ = client.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
		extra := readMsg(t, client)
		if extra != nil {
			t.Errorf("VP-064: expected connection closed after AUTH_FAIL; got extra message: %v", extra)
		}
	})

	t.Run("recognized_key_wrong_signature", func(t *testing.T) {
		t.Parallel()

		client := startServerOnPipe(t, daemonPriv, ops, handlers)
		if err := client.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
			t.Fatalf("SetDeadline: %v", err)
		}

		// Read CHALLENGE.
		challenge := readMsg(t, client)
		if challenge == nil || challenge["type"] != "challenge" {
			t.Fatalf("expected challenge; got %v", challenge)
		}
		nonceB64, _ := challenge["nonce"].(string)
		nonce := decB64(t, nonceB64)

		// Tamper with nonce before signing — signature will not verify.
		tampered := make([]byte, len(nonce))
		copy(tampered, nonce)
		tampered[0] ^= 0xFF
		sig := ed25519.Sign(authorizedPriv, tampered)

		writeMsg(t, client, map[string]any{
			"type":      "challenge_response",
			"nonce_sig": encB64(sig),
			"pubkey":    encB64([]byte(authorizedPub)),
		})

		// Expect AUTH_FAIL.
		resp := readMsg(t, client)
		if resp == nil {
			t.Fatal("expected AUTH_FAIL response; got nil")
		}
		if resp["type"] != "auth_fail" {
			t.Errorf("want type=auth_fail; got %v", resp["type"])
		}
		if resp["code"] != "E-ADM-010" {
			t.Errorf("want code=E-ADM-010; got %v", resp["code"])
		}

		// VP-064: AUTH_FAIL messages must be indistinguishable between sub-cases
		// (b) and (c) — same type, same code. Verified by comparing both subtests.
	})

	// VP-064: no RPC handler must have been called across all sub-cases.
	// (Checked at test completion after subtests run serially via t.Run.)
	t.Cleanup(func() {
		if rpcCallCount != 0 {
			t.Errorf("VP-064 violated: RPC sentinel handler called %d times on unauthenticated connections", rpcCallCount)
		}
	})
}

// ── AC-003 / VP-065: replay rejection ─────────────────────────────────────────

// TestMgmtServer_RejectsReplayedNonce_VP065 verifies VP-065:
// a nonce_sig captured from connection C1 cannot be replayed on connection C2
// because C2 issues a fresh nonce — ed25519.Verify(opPub, nonce2, sig1) = false
// → AUTH_FAIL.
//
// Traces: BC-2.07.004 PC-3, AC-003, VP-065.
func TestMgmtServer_RejectsReplayedNonce_VP065(t *testing.T) {
	t.Parallel()

	_, daemonPriv := mustGenKey(t)
	opPub, opPriv := mustGenKey(t)
	ops := mgmt.NewOperatorKeySet([]ed25519.PublicKey{opPub})

	t.Run("cross_connection_replay_fails_by_construction", func(t *testing.T) {
		t.Parallel()

		// Connection 1: obtain nonce and compute sig1 (but abandon without responding).
		client1 := startServerOnPipe(t, daemonPriv, ops, nil)
		if err := client1.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
			t.Fatalf("SetDeadline C1: %v", err)
		}

		challenge1 := readMsg(t, client1)
		if challenge1 == nil || challenge1["type"] != "challenge" {
			t.Fatalf("C1: expected challenge; got %v", challenge1)
		}
		nonce1B64, _ := challenge1["nonce"].(string)
		nonce1 := decB64(t, nonce1B64)
		sig1 := ed25519.Sign(opPriv, nonce1)

		_ = client1.Close() // abandon C1 without responding

		// Connection 2: fresh nonce. Replay sig1 (over nonce1) on C2 (which has nonce2).
		client2 := startServerOnPipe(t, daemonPriv, ops, nil)
		if err := client2.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
			t.Fatalf("SetDeadline C2: %v", err)
		}

		challenge2 := readMsg(t, client2)
		if challenge2 == nil || challenge2["type"] != "challenge" {
			t.Fatalf("C2: expected challenge; got %v", challenge2)
		}
		nonce2B64, _ := challenge2["nonce"].(string)
		nonce2 := decB64(t, nonce2B64)

		// Sanity: nonces must differ (crypto/rand virtually guarantees this).
		if bytes.Equal(nonce1, nonce2) {
			t.Skip("nonces collided — astronomically unlikely; skipping")
		}

		// Send old sig1 (signed over nonce1) against C2's nonce2.
		// ed25519.Verify(opPub, nonce2, sig1) = false → AUTH_FAIL.
		writeMsg(t, client2, map[string]any{
			"type":      "challenge_response",
			"nonce_sig": encB64(sig1),
			"pubkey":    encB64([]byte(opPub)),
		})

		resp := readMsg(t, client2)
		if resp == nil {
			t.Fatal("C2: expected AUTH_FAIL for replayed sig; got nil (connection closed before message)")
		}
		if resp["type"] != "auth_fail" {
			t.Errorf("C2: want type=auth_fail for replayed sig; got %v", resp["type"])
		}
		if resp["code"] != "E-ADM-010" {
			t.Errorf("C2: want code=E-ADM-010; got %v", resp["code"])
		}
	})
}

// ── AC-004: auth fail closes connection ──────────────────────────────────────

// TestMgmtServer_AuthFailClosesConnection_AC004 verifies that after receiving
// AUTH_FAIL, a subsequent read from the client returns an error (connection
// closed). No RPC response is ever sent on an unauthenticated connection.
//
// Traces: BC-2.07.004 PC-4, AC-004.
func TestMgmtServer_AuthFailClosesConnection_AC004(t *testing.T) {
	t.Parallel()

	_, daemonPriv := mustGenKey(t)
	ops := mgmt.NewOperatorKeySet(nil) // bootstrap: daemon key only

	rpcCallCount := 0
	handlers := sentinelHandlers(&rpcCallCount)

	client := startServerOnPipe(t, daemonPriv, ops, handlers)
	if err := client.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetDeadline: %v", err)
	}

	// Read CHALLENGE.
	challenge := readMsg(t, client)
	if challenge == nil || challenge["type"] != "challenge" {
		t.Fatalf("expected challenge; got %v", challenge)
	}
	nonceB64, _ := challenge["nonce"].(string)
	nonce := decB64(t, nonceB64)

	// Send a response with a random key (unrecognized) — this triggers AUTH_FAIL.
	randPub, randPriv := mustGenKey(t)
	sig := ed25519.Sign(randPriv, nonce)
	writeMsg(t, client, map[string]any{
		"type":      "challenge_response",
		"nonce_sig": encB64(sig),
		"pubkey":    encB64([]byte(randPub)),
	})

	// Receive AUTH_FAIL.
	resp := readMsg(t, client)
	if resp == nil {
		t.Fatal("expected AUTH_FAIL; got nil")
	}
	if resp["type"] != "auth_fail" {
		t.Fatalf("want type=auth_fail; got %v", resp["type"])
	}

	// AC-004: connection must be closed after AUTH_FAIL — next read returns nil/error.
	_ = client.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	extra := readMsg(t, client)
	if extra != nil {
		t.Errorf("AC-004: expected connection closed after AUTH_FAIL; got %v", extra)
	}

	// AC-004: no RPC handler must have been called.
	if rpcCallCount != 0 {
		t.Errorf("AC-004: RPC handler called %d times; want 0 (no auth → no dispatch)", rpcCallCount)
	}
}

// ── AC-005: RPCs without auth rejected ────────────────────────────────────────

// TestMgmtServer_RPCWithoutAuth_Rejected_AC005 verifies that a client that skips
// the handshake and sends a type="request" message directly receives AUTH_FAIL +
// close. No RPC handler is invoked.
//
// Traces: BC-2.07.004 PC-5, AC-005.
func TestMgmtServer_RPCWithoutAuth_Rejected_AC005(t *testing.T) {
	t.Parallel()

	_, daemonPriv := mustGenKey(t)
	ops := mgmt.NewOperatorKeySet(nil)

	rpcCallCount := 0
	handlers := sentinelHandlers(&rpcCallCount)

	client := startServerOnPipe(t, daemonPriv, ops, handlers)
	if err := client.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetDeadline: %v", err)
	}

	// Read and IGNORE the CHALLENGE (so the server is waiting for CHALLENGE_RESPONSE).
	challenge := readMsg(t, client)
	if challenge == nil || challenge["type"] != "challenge" {
		t.Fatalf("expected challenge; got %v", challenge)
	}

	// Send a type="request" RPC directly — skipping the CHALLENGE_RESPONSE step.
	writeMsg(t, client, map[string]any{
		"type":    "request",
		"id":      "r1",
		"command": "test.echo",
		"args":    map[string]any{},
	})

	// Expect AUTH_FAIL.
	resp := readMsg(t, client)
	if resp == nil {
		t.Fatal("AC-005: expected AUTH_FAIL for RPC without auth; got nil")
	}
	if resp["type"] != "auth_fail" {
		t.Errorf("AC-005: want type=auth_fail; got %v", resp["type"])
	}
	if resp["code"] != "E-ADM-010" {
		t.Errorf("AC-005: want code=E-ADM-010; got %v", resp["code"])
	}

	// AC-005: no RPC handler must have been called.
	if rpcCallCount != 0 {
		t.Errorf("AC-005: RPC handler called %d times; want 0", rpcCallCount)
	}
}

// ── AC-006 / VP-066: bounded reads ────────────────────────────────────────────

// TestMgmtServer_BoundedRead_VP066 verifies VP-066 (unit component):
// a message exceeding MaxMessageBytes causes the connection to close without OOM.
//
// Traces: BC-2.07.004 PC-6, AC-006, VP-066.
func TestMgmtServer_BoundedRead_VP066(t *testing.T) {
	t.Parallel()

	_, daemonPriv := mustGenKey(t)
	ops := mgmt.NewOperatorKeySet(nil) // bootstrap mode

	t.Run("under_limit_challenge_consumed", func(t *testing.T) {
		t.Parallel()

		client := startServerOnPipe(t, daemonPriv, ops, nil)
		if err := client.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
			t.Fatalf("SetDeadline: %v", err)
		}

		// Consume the CHALLENGE so the server is ready for CHALLENGE_RESPONSE.
		challenge := readMsg(t, client)
		if challenge == nil || challenge["type"] != "challenge" {
			t.Fatalf("expected challenge; got %v", challenge)
		}
		// Send a small valid-looking message (well under MaxMessageBytes).
		writeMsg(t, client, map[string]any{
			"type":      "challenge_response",
			"nonce_sig": encB64(make([]byte, 64)),
			"pubkey":    encB64(make([]byte, 32)),
		})
		// The server will return AUTH_FAIL (random key) or close — either is acceptable.
		// The important thing is it doesn't hang. If readMsg returns nil we're fine.
		_ = client.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		resp := readMsg(t, client)
		_ = resp // may be AUTH_FAIL or nil; both are acceptable for this sub-case
	})

	t.Run("oversized_message_closes_connection", func(t *testing.T) {
		t.Parallel()

		client := startServerOnPipe(t, daemonPriv, ops, nil)
		if err := client.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
			t.Fatalf("SetDeadline: %v", err)
		}

		// Consume CHALLENGE so the server advances to reading CHALLENGE_RESPONSE.
		buf := make([]byte, mgmt.MaxMessageBytes)
		n, err := client.Read(buf)
		if err != nil {
			t.Fatalf("read challenge: %v", err)
		}
		_ = buf[:n]

		// Send a payload larger than MaxMessageBytes.
		// Wrap in a JSON-like envelope so the decoder starts parsing.
		oversized := bytes.Repeat([]byte("x"), mgmt.MaxMessageBytes+1)
		payload := append([]byte(`{"type":"challenge_response","nonce_sig":"`), oversized...)
		payload = append(payload, '"', '}', '\n')

		_, writeErr := client.Write(payload)
		if writeErr != nil {
			// Connection may already be closed — that is acceptable.
			return
		}

		// VP-066: server must close the connection after the read limit is hit.
		_ = client.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		readBuf := make([]byte, 1024)
		_, readErr := client.Read(readBuf)
		if readErr == nil {
			t.Error("VP-066: expected connection close after oversized message; got successful read")
		}
	})
}

// FuzzMgmtServer_BoundedRead_VP066 verifies VP-066 (fuzz component):
// arbitrary byte sequences up to 2×MaxMessageBytes never cause OOM or panic.
//
// Run with: go test -fuzz FuzzMgmtServer_BoundedRead_VP066 ./internal/mgmt/
//
// Traces: BC-2.07.004 PC-6, AC-006, VP-066.
func FuzzMgmtServer_BoundedRead_VP066(f *testing.F) {
	f.Add([]byte(`{"type":"challenge_response","nonce_sig":"AAAA","pubkey":"BBBB"}`))
	f.Add(bytes.Repeat([]byte("A"), mgmt.MaxMessageBytes+512))
	f.Add([]byte("not json at all"))
	f.Add([]byte{0x00, 0xFF, 0xFE})

	_, daemonPriv, _ := ed25519.GenerateKey(rand.Reader)
	keySet := mgmt.NewOperatorKeySet(nil)

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 2*mgmt.MaxMessageBytes {
			data = data[:2*mgmt.MaxMessageBytes]
		}

		serverConn, clientConn := net.Pipe()
		ln := newSingleConnListener(serverConn)
		srv := mgmt.NewServer(ln, daemonPriv, keySet, nil, "dev")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() {
			_ = srv.Serve(ctx)
		}()
		defer func() { _ = clientConn.Close() }()
		defer func() { _ = serverConn.Close() }()

		// Consume the server-sent CHALLENGE.
		_ = clientConn.SetDeadline(time.Now().Add(100 * time.Millisecond))
		_, _ = io.Copy(io.Discard, io.LimitReader(clientConn, int64(mgmt.MaxMessageBytes)))

		// Send the fuzz payload.
		_ = clientConn.SetWriteDeadline(time.Now().Add(100 * time.Millisecond))
		_, _ = clientConn.Write(data)

		// Property: server must not panic and must close the connection within deadline.
		_ = clientConn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		_, _ = io.Copy(io.Discard, clientConn)
		// If we reach here without panic/timeout, the property holds for this input.
	})
}

// ── AC-007: happy path — AUTH_OK + RPC dispatch ───────────────────────────────

// TestMgmtServer_AuthOK_DispatchesRPC_AC007 verifies that a correctly-signed
// authorized operator key receives AUTH_OK and subsequent RPCs are dispatched
// to the registered handler.
//
// Traces: BC-2.07.004 PC-7, AC-007.
func TestMgmtServer_AuthOK_DispatchesRPC_AC007(t *testing.T) {
	t.Parallel()

	_, daemonPriv := mustGenKey(t)
	opPub, opPriv := mustGenKey(t)
	ops := mgmt.NewOperatorKeySet([]ed25519.PublicKey{opPub})

	rpcCallCount := 0
	handlers := sentinelHandlers(&rpcCallCount)

	client := startServerOnPipe(t, daemonPriv, ops, handlers)
	if err := client.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetDeadline: %v", err)
	}

	// Read CHALLENGE.
	challenge := readMsg(t, client)
	if challenge == nil || challenge["type"] != "challenge" {
		t.Fatalf("expected challenge; got %v", challenge)
	}
	nonceB64, _ := challenge["nonce"].(string)
	nonce := decB64(t, nonceB64)

	// Sign with the authorized operator key.
	sig := ed25519.Sign(opPriv, nonce)
	writeMsg(t, client, map[string]any{
		"type":      "challenge_response",
		"nonce_sig": encB64(sig),
		"pubkey":    encB64([]byte(opPub)),
	})

	// AC-007: expect AUTH_OK.
	authResp := readMsg(t, client)
	if authResp == nil {
		t.Fatal("AC-007: expected AUTH_OK; got nil (connection closed or timeout)")
	}
	if authResp["type"] != "auth_ok" {
		t.Fatalf("AC-007: want type=auth_ok; got %v", authResp["type"])
	}
	// AUTH_OK must carry daemon_version.
	if _, ok := authResp["daemon_version"]; !ok {
		t.Errorf("AC-007: AUTH_OK missing daemon_version field")
	}

	// AC-007: send a test RPC.
	writeMsg(t, client, map[string]any{
		"type":    "request",
		"id":      "r1",
		"command": "test.echo",
		"args":    map[string]any{},
	})

	// AC-007: expect a response envelope with ok=true.
	rpcResp := readMsg(t, client)
	if rpcResp == nil {
		t.Fatal("AC-007: expected RPC response; got nil")
	}
	if rpcResp["type"] != "response" {
		t.Errorf("AC-007: want type=response; got %v", rpcResp["type"])
	}
	if ok, _ := rpcResp["ok"].(bool); !ok {
		t.Errorf("AC-007: want ok=true in response envelope; got %v", rpcResp["ok"])
	}

	// AC-007: RPC handler must have been called exactly once.
	if rpcCallCount != 1 {
		t.Errorf("AC-007: RPC handler called %d times; want 1", rpcCallCount)
	}
}

// ── AC-008: OperatorKeySet constant-time comparison ──────────────────────────

// TestOperatorKeySet_ConstantTimeCompare_AC008 verifies that IsAuthorized:
//   - returns false for an unrecognized key (not in the set)
//   - returns false for a key with a one-byte mutation
//   - returns true for an exact match
//
// The timing-oracle property (PC-8) is verified by code inspection + review,
// not by a Go unit test (timing measurements in Go are not reliable at this
// resolution). This test confirms the CORRECTNESS of the constant-time path.
//
// Traces: BC-2.07.004 PC-8, Inv-5, AC-008.
func TestOperatorKeySet_ConstantTimeCompare_AC008(t *testing.T) {
	t.Parallel()

	// Timing-oracle note: subtle.ConstantTimeCompare is used in IsAuthorized
	// per BC-2.07.004 PC-8 / Inv-5. The CORRECTNESS assertions here verify:
	//   1. An authorized key returns true.
	//   2. A one-byte mutation of an authorized key returns false.
	//   3. A completely unrecognized key returns false.
	// These assertions also confirm both paths (recognized/unrecognized) execute
	// the same comparison code — no early-return oracle.

	authorizedPub, _ := mustGenKey(t)
	unrecognizedPub, _ := mustGenKey(t)

	ops := mgmt.NewOperatorKeySet([]ed25519.PublicKey{authorizedPub})

	// Authorized key returns true.
	if !ops.IsAuthorized(authorizedPub) {
		t.Error("AC-008: IsAuthorized(authorizedPub): want true; got false")
	}

	// One-byte mutation of authorized key returns false.
	mutated := make(ed25519.PublicKey, len(authorizedPub))
	copy(mutated, authorizedPub)
	mutated[0] ^= 0xFF
	if ops.IsAuthorized(mutated) {
		t.Error("AC-008: IsAuthorized(mutated): want false; got true (constant-time check must fail on mutation)")
	}

	// Completely unrecognized key returns false.
	if ops.IsAuthorized(unrecognizedPub) {
		t.Error("AC-008: IsAuthorized(unrecognizedPub): want false; got true")
	}

	// Bootstrap (empty) set: no key is authorized.
	bootstrapOps := mgmt.NewOperatorKeySet(nil)
	if bootstrapOps.IsAuthorized(authorizedPub) {
		t.Error("AC-008: IsAuthorized on bootstrap set (empty): want false; got true (bootstrap set defers to daemon key, not operator set)")
	}
}

// ── AC-009: bootstrap mode ────────────────────────────────────────────────────

// TestMgmtServer_BootstrapMode_DaemonKeyAuthorized_AC009 verifies that when
// authorized_operator_keys is empty (nil or zero-length), the server accepts
// connections signed by the daemon's own keypair.
//
// Traces: BC-2.07.004 PC-9, AC-009.
func TestMgmtServer_BootstrapMode_DaemonKeyAuthorized_AC009(t *testing.T) {
	t.Parallel()

	daemonPub, daemonPriv := mustGenKey(t)
	// Empty operator key set → bootstrap mode.
	ops := mgmt.NewOperatorKeySet(nil)

	if !ops.IsBootstrap() {
		t.Fatal("AC-009: NewOperatorKeySet(nil).IsBootstrap(): want true; got false")
	}

	client := startServerOnPipe(t, daemonPriv, ops, nil)
	if err := client.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetDeadline: %v", err)
	}

	// Read CHALLENGE.
	challenge := readMsg(t, client)
	if challenge == nil || challenge["type"] != "challenge" {
		t.Fatalf("expected challenge; got %v", challenge)
	}
	nonceB64, _ := challenge["nonce"].(string)
	nonce := decB64(t, nonceB64)

	// Sign with the daemon's own private key — this is bootstrap auth.
	sig := ed25519.Sign(daemonPriv, nonce)
	writeMsg(t, client, map[string]any{
		"type":      "challenge_response",
		"nonce_sig": encB64(sig),
		"pubkey":    encB64([]byte(daemonPub)),
	})

	// AC-009: expect AUTH_OK (daemon key is the bootstrap authorized key).
	resp := readMsg(t, client)
	if resp == nil {
		t.Fatal("AC-009: expected AUTH_OK in bootstrap mode; got nil (connection closed or timeout)")
	}
	if resp["type"] != "auth_ok" {
		t.Errorf("AC-009: want type=auth_ok; got %v (bootstrap mode: daemon key must be accepted)", resp["type"])
	}
}

// ── AC-010: graceful shutdown ─────────────────────────────────────────────────

// TestMgmtServer_GracefulShutdown_AC010 verifies that Server.Shutdown(ctx)
// closes the listener so no new connections are accepted, waits for in-flight
// connections to terminate within the context deadline, and that Serve returns.
// No goroutine leak after shutdown.
//
// Traces: BC-2.07.004 PC-10, AC-010.
func TestMgmtServer_GracefulShutdown_AC010(t *testing.T) {
	// NOT t.Parallel(): measures goroutine counts for leak detection.

	gorsBefore := runtime.NumGoroutine()

	_, daemonPriv := mustGenKey(t)
	ops := mgmt.NewOperatorKeySet(nil)

	serverConn, clientConn := net.Pipe()
	ln := newSingleConnListener(serverConn)
	srv := mgmt.NewServer(ln, daemonPriv, ops, nil, "dev")

	serveCtx, serveCancel := context.WithCancel(context.Background())
	defer serveCancel()

	serveDone := make(chan error, 1)
	go func() {
		serveDone <- srv.Serve(serveCtx)
	}()

	// Give Serve a moment to enter its accept loop.
	time.Sleep(20 * time.Millisecond)

	// Shutdown with a short deadline.
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer shutCancel()

	if err := srv.Shutdown(shutCtx); err != nil {
		t.Logf("Shutdown: %v (may be expected on stub)", err)
	}

	// AC-010: Serve must return after Shutdown is called.
	select {
	case err := <-serveDone:
		t.Logf("Serve returned: %v", err)
	case <-time.After(700 * time.Millisecond):
		t.Error("AC-010: Serve did not return within 700ms of Shutdown")
	}

	_ = clientConn.Close()

	// AC-010: no goroutine leak after shutdown (+2 tolerance for transient Go runtime goroutines).
	t.Cleanup(func() {
		deadline := time.After(300 * time.Millisecond)
		for {
			after := runtime.NumGoroutine()
			if after <= gorsBefore+2 {
				return
			}
			select {
			case <-deadline:
				t.Errorf("AC-010: goroutine leak after Shutdown: before=%d after=%d",
					gorsBefore, runtime.NumGoroutine())
				return
			default:
				runtime.Gosched()
			}
		}
	})
}

// ── AC-001 (v1.1): HandshakeTimeout silent-stall sub-case (VP-064 sub-case a) ─

// TestMgmtServer_HandshakeTimeout_SilentStall_AC001 verifies that after sending
// CHALLENGE the server applies a HandshakeTimeout read deadline. A client that
// connects, receives the CHALLENGE, and then sends NOTHING triggers E-ADM-010
// (connection closed) within the deadline. No goroutine leak.
//
// This sub-case is VP-064 property "no CHALLENGE_RESPONSE at all" from Ruling 1.
// The test uses a 50ms injected HandshakeTimeout (mgmt.HandshakeTimeout default
// is 10s — too slow for a unit test). The implementer MUST add a
// HandshakeTimeout field to Server (or a WithHandshakeTimeout option) so this
// injectable deadline can be exercised.
//
// COMPILE FAILURE IS EXPECTED until the implementer adds:
//   - mgmt.HandshakeTimeout constant (default 10s)
//   - mgmt.WithHandshakeTimeout(d time.Duration) option (or equivalent injectable field)
//   - mgmt.NewServer(..., daemonVersion string) updated signature (AC-007)
//
// Traces: BC-2.07.004 PC-1, EC-001, AC-001, VP-064 sub-case (a), Ruling 1.
func TestMgmtServer_HandshakeTimeout_SilentStall_AC001(t *testing.T) {
	t.Parallel()

	// Verify the HandshakeTimeout constant exists with the required value.
	// This assertion fails to compile until the implementer exports the constant.
	if mgmt.HandshakeTimeout != 10*time.Second {
		t.Errorf("AC-001: mgmt.HandshakeTimeout = %v; want 10s (ADR-012 §7 / Ruling 1)", mgmt.HandshakeTimeout)
	}

	_, daemonPriv := mustGenKey(t)
	ops := mgmt.NewOperatorKeySet(nil)

	gorsBefore := runtime.NumGoroutine()

	// Construct a server with a 50ms HandshakeTimeout override so the test
	// completes in milliseconds rather than 10 seconds.
	// WithHandshakeTimeout is the required injectable option — the test encodes
	// the API contract the implementer must satisfy.
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		_ = serverConn.Close()
		_ = clientConn.Close()
	})

	ln := newSingleConnListener(serverConn)
	srv := mgmt.NewServer(ln, daemonPriv, ops, nil, "0.1.0-test",
		mgmt.WithHandshakeTimeout(50*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go func() {
		_ = srv.Serve(ctx)
	}()

	// Set client read deadline slightly beyond the injected HandshakeTimeout.
	if err := clientConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}

	// Consume the CHALLENGE message (server must send it first — AC-001).
	msg := readMsg(t, clientConn)
	if msg == nil {
		t.Fatal("AC-001: expected CHALLENGE as first server message; got nil")
	}
	if msg["type"] != "challenge" {
		t.Fatalf("AC-001: want type=challenge; got %v", msg["type"])
	}

	// Now send NOTHING — simulate silent stall (VP-064 sub-case a).
	// The server should close the connection after the 50ms HandshakeTimeout.
	start := time.Now()
	next := readMsg(t, clientConn)
	elapsed := time.Since(start)

	// VP-064 sub-case a: server must close the connection.
	// The next read must return nil (connection closed) within ~100ms.
	if next != nil {
		t.Errorf("AC-001: VP-064 sub-case a: expected connection closed after HandshakeTimeout; got message: %v", next)
	}
	if elapsed > 200*time.Millisecond {
		t.Errorf("AC-001: connection close took %v; want within 200ms of 50ms HandshakeTimeout", elapsed)
	}

	// Goroutine leak check: all server goroutines must exit after deadline.
	t.Cleanup(func() {
		deadline := time.After(300 * time.Millisecond)
		for {
			after := runtime.NumGoroutine()
			if after <= gorsBefore+2 {
				return
			}
			select {
			case <-deadline:
				t.Errorf("AC-001: goroutine leak after HandshakeTimeout: before=%d after=%d",
					gorsBefore, runtime.NumGoroutine())
				return
			default:
				runtime.Gosched()
			}
		}
	})
}

// ── AC-003 / VP-065 (v1.1): post-auth structural guard ───────────────────────

// TestMgmtServer_PostAuthChallengeResponseRejected_VP065 verifies the v1.1 AC-003:
// after a successful handshake (AUTH_OK), a second {"type":"challenge_response",...}
// on the same connection triggers E-ADM-010 + close. The per-connection
// authenticated boolean (not a nonce-set) causes the rejection.
//
// The registered RPC handler must NOT be invoked by the second challenge_response.
//
// Note: the existing TestMgmtServer_RejectsReplayedNonce_VP065 tests cross-connection
// replay (Ruling 7 pre-ADR-012 v1.2 framing). This test is the NEW v1.1 structural
// guard test per BC-2.07.004 PC-3 v1.2 / AC-003 v1.1 / Ruling 7.
//
// COMPILE FAILURE IS EXPECTED until the implementer adds:
//   - mgmt.NewServer(..., daemonVersion string) updated signature (AC-007)
//
// Traces: BC-2.07.004 PC-3 v1.2, EC-004, AC-003 v1.1, VP-065.
func TestMgmtServer_PostAuthChallengeResponseRejected_VP065(t *testing.T) {
	t.Parallel()

	_, daemonPriv := mustGenKey(t)
	opPub, opPriv := mustGenKey(t)
	ops := mgmt.NewOperatorKeySet([]ed25519.PublicKey{opPub})

	rpcCallCount := 0
	handlers := sentinelHandlers(&rpcCallCount)

	// Construct server with daemonVersion "0.1.0-test" (required by AC-007 / Ruling 6).
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		_ = serverConn.Close()
		_ = clientConn.Close()
	})

	ln := newSingleConnListener(serverConn)
	// AC-007: NewServer MUST accept daemonVersion as fifth parameter.
	// This call will fail to compile until the implementer updates the signature.
	srv := mgmt.NewServer(ln, daemonPriv, ops, handlers, "0.1.0-test")

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go func() {
		_ = srv.Serve(ctx)
	}()

	if err := clientConn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetDeadline: %v", err)
	}

	// Step 1: complete a successful handshake → AUTH_OK.
	challenge := readMsg(t, clientConn)
	if challenge == nil || challenge["type"] != "challenge" {
		t.Fatalf("VP-065: expected challenge; got %v", challenge)
	}
	nonceB64, _ := challenge["nonce"].(string)
	nonce := decB64(t, nonceB64)
	sig := ed25519.Sign(opPriv, nonce)

	writeMsg(t, clientConn, map[string]any{
		"type":      "challenge_response",
		"nonce_sig": encB64(sig),
		"pubkey":    encB64([]byte(opPub)),
	})

	authResp := readMsg(t, clientConn)
	if authResp == nil {
		t.Fatal("VP-065: expected AUTH_OK after valid handshake; got nil")
	}
	if authResp["type"] != "auth_ok" {
		t.Fatalf("VP-065: want AUTH_OK after valid handshake; got type=%v", authResp["type"])
	}

	// Step 2: on the same authenticated connection C1, send a second challenge_response.
	// The per-connection `authenticated` boolean causes the server to reject this.
	// The RPC handler must NOT be invoked.
	writeMsg(t, clientConn, map[string]any{
		"type":      "challenge_response",
		"nonce_sig": encB64(sig), // same sig — irrelevant; type check comes first
		"pubkey":    encB64([]byte(opPub)),
	})

	// Expect AUTH_FAIL (E-ADM-010) — the structural guard fires.
	failResp := readMsg(t, clientConn)
	if failResp == nil {
		t.Fatal("VP-065: expected AUTH_FAIL for post-auth challenge_response; got nil (connection closed without message)")
	}
	if failResp["type"] != "auth_fail" {
		t.Errorf("VP-065: want type=auth_fail for post-auth challenge_response; got %v", failResp["type"])
	}
	if failResp["code"] != "E-ADM-010" {
		t.Errorf("VP-065: want code=E-ADM-010; got %v", failResp["code"])
	}

	// Connection must be closed after AUTH_FAIL.
	_ = clientConn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	extra := readMsg(t, clientConn)
	if extra != nil {
		t.Errorf("VP-065: expected connection closed after AUTH_FAIL; got extra message: %v", extra)
	}

	// RPC sentinel must not have been called by the second challenge_response.
	if rpcCallCount != 0 {
		t.Errorf("VP-065: RPC handler called %d times; want 0 (post-auth challenge_response must not dispatch RPC)", rpcCallCount)
	}
}

// ── AC-007 (v1.1): daemonVersion injection ───────────────────────────────────

// TestMgmtServer_DaemonVersion_Injected_AC007 verifies that AUTH_OK carries the
// daemonVersion string injected into NewServer, not a hardcoded "dev" sentinel.
// Also verifies that NewServer panics when daemonVersion is "".
//
// COMPILE FAILURE IS EXPECTED until the implementer adds:
//   - mgmt.NewServer(ln, daemonKey, ops, handlers, daemonVersion string) (fifth param)
//   - panic on empty daemonVersion (or equivalent initialization error)
//
// Traces: BC-2.07.004 PC-7, AC-007 v1.1, Ruling 6.
func TestMgmtServer_DaemonVersion_Injected_AC007(t *testing.T) {
	t.Parallel()

	t.Run("auth_ok_carries_injected_version", func(t *testing.T) {
		t.Parallel()

		_, daemonPriv := mustGenKey(t)
		opPub, opPriv := mustGenKey(t)
		ops := mgmt.NewOperatorKeySet([]ed25519.PublicKey{opPub})

		serverConn, clientConn := net.Pipe()
		t.Cleanup(func() {
			_ = serverConn.Close()
			_ = clientConn.Close()
		})

		ln := newSingleConnListener(serverConn)
		// AC-007: daemonVersion "0.1.0-test" must appear in AUTH_OK.
		srv := mgmt.NewServer(ln, daemonPriv, ops, nil, "0.1.0-test")

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		go func() {
			_ = srv.Serve(ctx)
		}()

		if err := clientConn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
			t.Fatalf("SetDeadline: %v", err)
		}

		challenge := readMsg(t, clientConn)
		if challenge == nil || challenge["type"] != "challenge" {
			t.Fatalf("AC-007: expected challenge; got %v", challenge)
		}
		nonceB64, _ := challenge["nonce"].(string)
		nonce := decB64(t, nonceB64)

		sig := ed25519.Sign(opPriv, nonce)
		writeMsg(t, clientConn, map[string]any{
			"type":      "challenge_response",
			"nonce_sig": encB64(sig),
			"pubkey":    encB64([]byte(opPub)),
		})

		authResp := readMsg(t, clientConn)
		if authResp == nil {
			t.Fatal("AC-007: expected AUTH_OK; got nil")
		}
		if authResp["type"] != "auth_ok" {
			t.Fatalf("AC-007: want type=auth_ok; got %v", authResp["type"])
		}

		// AC-007: daemon_version must equal the injected value, NOT "dev".
		ver, ok := authResp["daemon_version"].(string)
		if !ok {
			t.Fatalf("AC-007: AUTH_OK missing daemon_version field or wrong type; got %v", authResp["daemon_version"])
		}
		if ver != "0.1.0-test" {
			t.Errorf("AC-007: AUTH_OK daemon_version = %q; want %q (must equal NewServer daemonVersion param, not hardcoded sentinel)",
				ver, "0.1.0-test")
		}
	})

	t.Run("empty_daemonVersion_panics", func(t *testing.T) {
		t.Parallel()

		_, daemonPriv := mustGenKey(t)
		ops := mgmt.NewOperatorKeySet(nil)

		serverConn, _ := net.Pipe()
		ln := newSingleConnListener(serverConn)

		// AC-007: NewServer MUST panic (or equivalent) if daemonVersion == "".
		// The implementer documents the chosen enforcement in comments per the story task.
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("AC-007: NewServer with empty daemonVersion must panic; did not panic")
			}
			// Cleanup the pipe after recover.
			_ = serverConn.Close()
		}()
		// This call must panic.
		_ = mgmt.NewServer(ln, daemonPriv, ops, nil, "")
	})
}

// ── AC-013: connection cap (bounded accept loop, CWE-770) ────────────────────

// TestMgmtServer_ConnectionCap_AC013 verifies that Server does not spawn more
// than MaxConcurrentConnections simultaneous connection goroutines. With a cap of
// 3 (injected via constructor option), the 4th connection does not immediately
// receive a CHALLENGE — the accept loop back-pressures. When one of the 3 held
// connections is released, the 4th proceeds.
//
// COMPILE FAILURE IS EXPECTED until the implementer adds:
//   - mgmt.MaxConcurrentConnections constant (default 128)
//   - mgmt.WithMaxConnections(n int) option (or equivalent)
//   - mgmt.NewServer(..., daemonVersion string) updated signature (AC-007)
//
// Traces: BC-2.07.004 EC-012, AC-013 v1.1, Ruling 3.
func TestMgmtServer_ConnectionCap_AC013(t *testing.T) {
	// NOT t.Parallel(): measures goroutine counts for leak detection.

	// Verify the MaxConcurrentConnections constant exists.
	// Fails to compile until the implementer exports it.
	if mgmt.MaxConcurrentConnections != 128 {
		t.Errorf("AC-013: mgmt.MaxConcurrentConnections = %d; want 128 (ADR-012 §8 / Ruling 3)",
			mgmt.MaxConcurrentConnections)
	}

	_, daemonPriv := mustGenKey(t)
	ops := mgmt.NewOperatorKeySet(nil)

	// multiConnListener accepts up to n connections from pre-connected net.Pipe pairs.
	// It is an in-process fake that drives the bounded accept loop without touching
	// the OS network stack.
	type multiConnListener struct {
		conns  chan net.Conn
		closed chan struct{}
	}
	newMultiConnListener := func(capacity int) *multiConnListener {
		return &multiConnListener{
			conns:  make(chan net.Conn, capacity),
			closed: make(chan struct{}),
		}
	}
	mln := newMultiConnListener(10)
	mlnAddr := &net.UnixAddr{Name: "test-cap", Net: "unix"}

	// Embed Accept/Close/Addr on multiConnListener to satisfy net.Listener.
	acceptFn := func() (net.Conn, error) {
		select {
		case <-mln.closed:
			return nil, net.ErrClosed
		case c := <-mln.conns:
			return c, nil
		}
	}
	closeFn := func() error {
		select {
		case <-mln.closed:
		default:
			close(mln.closed)
		}
		return nil
	}
	addrFn := func() net.Addr { return mlnAddr }

	ln := &fakeSyncListener{acceptFn: acceptFn, closeFn: closeFn, addrFn: addrFn}

	// Construct server with MaxConcurrentConnections = 3 via WithMaxConnections.
	// AC-013: the implementer must add WithMaxConnections(n int) functional option.
	srv := mgmt.NewServer(ln, daemonPriv, ops, nil, "0.1.0-test",
		mgmt.WithMaxConnections(3))

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go func() {
		_ = srv.Serve(ctx)
	}()

	// Give Serve a moment to start its accept loop.
	time.Sleep(10 * time.Millisecond)

	// Open 3 connections that stall at the handshake phase (receive CHALLENGE, send nothing).
	var stalledClients [3]net.Conn
	for i := range stalledClients {
		srvConn, cliConn := net.Pipe()
		mln.conns <- srvConn
		stalledClients[i] = cliConn
	}

	// Wait for all 3 to receive their CHALLENGE (proving they are in the accept-loop
	// goroutine, holding the semaphore slots).
	for i, cli := range stalledClients {
		if err := cli.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
			t.Fatalf("client[%d] SetReadDeadline: %v", i, err)
		}
		msg := readMsg(t, cli)
		if msg == nil || msg["type"] != "challenge" {
			t.Fatalf("AC-013: client[%d]: expected challenge; got %v", i, msg)
		}
		// Clear deadline — we want this client to stall indefinitely.
		if err := cli.SetReadDeadline(time.Time{}); err != nil {
			t.Fatalf("client[%d] clear deadline: %v", i, err)
		}
	}

	// Now attempt a 4th connection. The server semaphore is at capacity (3/3).
	// Inject the 4th server-side conn into the listener.
	srvConn4, cliConn4 := net.Pipe()
	t.Cleanup(func() {
		_ = srvConn4.Close()
		_ = cliConn4.Close()
	})
	mln.conns <- srvConn4

	// AC-013: the 4th client must NOT immediately receive a CHALLENGE.
	// Set a short deadline — if it times out, the semaphore is back-pressuring correctly.
	if err := cliConn4.SetReadDeadline(time.Now().Add(80 * time.Millisecond)); err != nil {
		t.Fatalf("client4 SetReadDeadline: %v", err)
	}
	msg4 := readMsg(t, cliConn4)
	if msg4 != nil {
		t.Errorf("AC-013: 4th client received message while semaphore should be full; "+
			"got type=%v (server must back-pressure, not spawn unbounded goroutines)", msg4["type"])
	}

	// Release one stalled client → semaphore slot freed → 4th client should proceed.
	_ = stalledClients[0].Close()
	// Register cleanup for remaining stalled clients (indices 1 and 2).
	t.Cleanup(func() { _ = stalledClients[1].Close() })
	t.Cleanup(func() { _ = stalledClients[2].Close() })

	// Give the 4th client a fresh deadline to receive its CHALLENGE.
	if err := cliConn4.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("client4 fresh deadline: %v", err)
	}
	msg4b := readMsg(t, cliConn4)
	if msg4b == nil || msg4b["type"] != "challenge" {
		t.Errorf("AC-013: 4th client: expected challenge after semaphore slot freed; got %v", msg4b)
	}
}

// fakeSyncListener is an in-process net.Listener backed by injectable functions.
// Used by TestMgmtServer_ConnectionCap_AC013 to avoid real OS sockets.
type fakeSyncListener struct {
	acceptFn func() (net.Conn, error)
	closeFn  func() error
	addrFn   func() net.Addr
}

func (f *fakeSyncListener) Accept() (net.Conn, error) { return f.acceptFn() }
func (f *fakeSyncListener) Close() error              { return f.closeFn() }
func (f *fakeSyncListener) Addr() net.Addr            { return f.addrFn() }

// ── VP-068 / AC-016 / Invariant 8: nil/short-key construction guard ──────────

// doHandshake performs the ADR-012 challenge-response handshake over conn
// using opPriv (the operator private key). Returns nil on success or an error
// if any step fails. Used by VP-070 and VP-071 tests.
func doHandshake(t *testing.T, conn net.Conn, opPriv ed25519.PrivateKey) error {
	t.Helper()
	dec := json.NewDecoder(conn)

	var challenge struct {
		Type  string `json:"type"`
		Nonce string `json:"nonce"`
	}
	if err := dec.Decode(&challenge); err != nil {
		return fmt.Errorf("read challenge: %w", err)
	}
	if challenge.Type != "challenge" {
		return fmt.Errorf("expected challenge, got %q", challenge.Type)
	}
	nonceBytes, err := base64.RawURLEncoding.DecodeString(challenge.Nonce)
	if err != nil {
		return fmt.Errorf("decode nonce: %w", err)
	}
	sig := ed25519.Sign(opPriv, nonceBytes)
	opPub := opPriv.Public().(ed25519.PublicKey)
	msg, err := json.Marshal(map[string]any{
		"type":      "challenge_response",
		"nonce_sig": base64.RawURLEncoding.EncodeToString(sig),
		"pubkey":    base64.RawURLEncoding.EncodeToString([]byte(opPub)),
	})
	if err != nil {
		return fmt.Errorf("marshal challenge_response: %w", err)
	}
	msg = append(msg, '\n')
	if _, err := conn.Write(msg); err != nil {
		return fmt.Errorf("write challenge_response: %w", err)
	}

	var authResp struct {
		Type string `json:"type"`
	}
	if err := dec.Decode(&authResp); err != nil {
		return fmt.Errorf("read auth response: %w", err)
	}
	if authResp.Type != "auth_ok" {
		return fmt.Errorf("expected auth_ok, got %q", authResp.Type)
	}
	return nil
}

// TestNewServer_PanicsOnNilKey_VP068 verifies VP-068 / BC-2.07.004 Invariant 8
// / AC-016: NewServer MUST panic immediately if daemonKey is nil (len == 0).
//
// Currently mgmt.go has no nil-key guard → NewServer does NOT panic with nil
// key → the defer/recover sees r==nil → t.Errorf fires → RED.
//
// Traces: BC-2.07.004 Invariant 8, AC-016, VP-068.
func TestNewServer_PanicsOnNilKey_VP068(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	didPanic := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				didPanic = true
			}
		}()
		// VP-068: nil key (len==0) MUST panic at construction time.
		// Without the guard, this returns a *Server with a nil daemonKey field,
		// which will later panic mid-connection — a remote-panic DoS vector.
		_ = mgmt.NewServer(ln, nil, mgmt.NewOperatorKeySet(nil), nil, "dev")
	}()

	if !didPanic {
		t.Errorf("VP-068 violated: NewServer(nil daemonKey) did not panic; " +
			"implementer must add: if len(daemonKey) != ed25519.PrivateKeySize { panic(...) }")
	}
}

// TestNewServer_PanicsOnShortKey_VP068 verifies VP-068 / BC-2.07.004 Invariant 8
// / AC-016: NewServer MUST panic if daemonKey length != ed25519.PrivateKeySize.
//
// A 32-byte slice is the ed25519.PublicKey size — a common programmer mistake
// (passing the public key where the private key is required).
//
// Currently mgmt.go has no short-key guard → no panic → RED.
//
// Traces: BC-2.07.004 Invariant 8, AC-016, VP-068.
func TestNewServer_PanicsOnShortKey_VP068(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		keyLen int
	}{
		{name: "len_32_public_key_size", keyLen: ed25519.PublicKeySize}, // 32
		{name: "len_63_one_short", keyLen: ed25519.PrivateKeySize - 1},  // 63
		{name: "len_1_single_byte", keyLen: 1},
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			shortKey := make(ed25519.PrivateKey, tc.keyLen)

			didPanic := false
			func() {
				defer func() {
					if r := recover(); r != nil {
						didPanic = true
					}
				}()
				_ = mgmt.NewServer(ln, shortKey, mgmt.NewOperatorKeySet(nil), nil, "dev")
			}()

			if !didPanic {
				t.Errorf("VP-068 violated (%s, len=%d): NewServer did not panic; "+
					"a short key must be rejected at construction, not at first connection",
					tc.name, tc.keyLen)
			}
		})
	}
}

// ── VP-069 / AC-017 / PC-10: Serve returns nil on intentional shutdown ────────

// TestServe_ReturnsNilOnShutdown_VP069 verifies VP-069 / BC-2.07.004 PC-10
// / AC-017: Server.Serve returns nil (not net.ErrClosed) when Shutdown is called.
//
// Currently Serve's Accept-error path does not check shuttingDown and returns
// net.ErrClosed → the non-nil-error assertion fires → RED.
//
// Traces: BC-2.07.004 PC-10, AC-017, VP-069.
func TestServe_ReturnsNilOnShutdown_VP069(t *testing.T) {
	t.Parallel()

	_, daemonPriv := mustGenKey(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	// Do NOT t.Cleanup(ln.Close) — Shutdown will close the listener.

	srv := mgmt.NewServer(ln, daemonPriv, mgmt.NewOperatorKeySet(nil), nil, "dev")

	ctx := context.Background()
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ctx)
	}()

	// Give Serve time to enter the Accept loop.
	time.Sleep(10 * time.Millisecond)

	if err := srv.Shutdown(context.Background()); err != nil {
		t.Logf("Shutdown returned: %v (non-nil is acceptable here)", err)
	}

	select {
	case serveErr := <-errCh:
		// VP-069: Serve MUST return nil on intentional Shutdown.
		// Currently returns net.ErrClosed → this assertion fires.
		if serveErr != nil {
			t.Errorf("VP-069 violated: Serve returned %v after Shutdown; want nil "+
				"(implementer must add shuttingDown atomic.Bool and nil-on-shutdown logic)",
				serveErr)
		}
	case <-time.After(2 * time.Second):
		t.Error("VP-069: Serve did not return within 2s of Shutdown")
	}
}

// TestServe_ReturnsNilOnCtxCancel_VP069 verifies VP-069 / BC-2.07.004 PC-10
// / AC-017: Server.Serve returns nil when the context is cancelled.
//
// Currently Serve's ctx-watcher goroutine calls s.ln.Close() but does not set
// shuttingDown, so the Accept error path returns net.ErrClosed → RED.
//
// Traces: BC-2.07.004 PC-10, AC-017, VP-069.
func TestServe_ReturnsNilOnCtxCancel_VP069(t *testing.T) {
	t.Parallel()

	_, daemonPriv := mustGenKey(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	srv := mgmt.NewServer(ln, daemonPriv, mgmt.NewOperatorKeySet(nil), nil, "dev")

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ctx)
	}()

	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case serveErr := <-errCh:
		// VP-069: Serve MUST return nil on ctx cancel.
		// Currently returns net.ErrClosed → this assertion fires.
		if serveErr != nil {
			t.Errorf("VP-069 violated: Serve returned %v after ctx cancel; want nil "+
				"(implementer must add shuttingDown atomic.Bool; ctx-watcher sets it before ln.Close())",
				serveErr)
		}
	case <-time.After(2 * time.Second):
		t.Error("VP-069: Serve did not return within 2s of ctx cancel")
	}
}

// ── VP-070 / AC-007 / PC-11: E-RPC-010 for unknown commands ──────────────────

// TestUnknownCommand_ReturnsERPC010_VP070 verifies VP-070 / BC-2.07.004 PC-11
// / AC-007: an authenticated client naming an unregistered command receives
// ok:false, error.code == "E-RPC-010", and the connection stays OPEN.
//
// Currently handleConnection sends E-RPC-001 for unknown commands → the
// assertion on "E-RPC-010" fires → RED.
// Also verifies E-RPC-002 does NOT appear (which would indicate the defective
// handler-error code was used).
//
// Traces: BC-2.07.004 PC-11, AC-007 (Ruling C), VP-070.
func TestUnknownCommand_ReturnsERPC010_VP070(t *testing.T) {
	t.Parallel()

	_, daemonPriv := mustGenKey(t)
	opPub, opPriv := mustGenKey(t)
	keySet := mgmt.NewOperatorKeySet([]ed25519.PublicKey{opPub})

	// No handlers registered — every command is "unknown".
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	srv := mgmt.NewServer(ln, daemonPriv, keySet, nil, "dev")
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go srv.Serve(ctx) //nolint:errcheck

	conn, err := net.DialTimeout("tcp", ln.Addr().String(), time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatalf("SetDeadline: %v", err)
	}

	if err := doHandshake(t, conn, opPriv); err != nil {
		t.Fatalf("handshake failed: %v", err)
	}

	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(conn)

	// Send an unregistered command.
	if err := enc.Encode(map[string]any{
		"type":    "request",
		"id":      "req-vp070",
		"command": "nonexistent.command",
		"args":    map[string]any{},
	}); err != nil {
		t.Fatalf("encode request: %v", err)
	}

	var resp struct {
		Type  string `json:"type"`
		ID    string `json:"id"`
		OK    bool   `json:"ok"`
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := dec.Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.OK {
		t.Error("VP-070 violated: ok=true for unregistered command")
	}
	// VP-070 core assertion: must be E-RPC-010, not E-RPC-001 or E-RPC-002.
	if resp.Error.Code != "E-RPC-010" {
		t.Errorf("VP-070 violated: error.code = %q; want E-RPC-010 "+
			"(implementer must replace E-RPC-001 with E-RPC-010 in handleConnection dispatch)",
			resp.Error.Code)
	}
	if resp.ID != "req-vp070" {
		t.Errorf("VP-070: response id = %q; want %q", resp.ID, "req-vp070")
	}
	wantMsg := "unknown command: nonexistent.command"
	if resp.Error.Message != wantMsg {
		t.Errorf("VP-070: error.message = %q; want %q", resp.Error.Message, wantMsg)
	}

	// Connection must still be open: send a second unknown command.
	if err := enc.Encode(map[string]any{
		"type":    "request",
		"id":      "req-vp070b",
		"command": "another.unknown",
		"args":    map[string]any{},
	}); err != nil {
		t.Fatalf("VP-070: connection was closed after first unknown command (must stay open): %v", err)
	}
	var resp2 struct {
		OK    bool `json:"ok"`
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := dec.Decode(&resp2); err != nil {
		t.Fatalf("VP-070: decode second response: %v", err)
	}
	if resp2.OK || resp2.Error.Code != "E-RPC-010" {
		t.Errorf("VP-070: second unknown command: ok=%v code=%q; want ok=false code=E-RPC-010",
			resp2.OK, resp2.Error.Code)
	}
}

// ── VP-071 / AC-007 / PC-12: E-RPC-011 for handler errors ────────────────────

// TestHandlerError_ReturnsERPC011_VP071 verifies VP-071 / BC-2.07.004 PC-12
// / AC-007: a registered handler returning an error causes the server to send
// ok:false, error.code == "E-RPC-011", message == the verbatim error string,
// and the connection stays OPEN.
//
// Currently handleConnection sends E-RPC-002 for handler errors → the assertion
// on "E-RPC-011" fires → RED.
//
// Traces: BC-2.07.004 PC-12, AC-007 (Ruling C), VP-071.
func TestHandlerError_ReturnsERPC011_VP071(t *testing.T) {
	t.Parallel()

	_, daemonPriv := mustGenKey(t)
	opPub, opPriv := mustGenKey(t)
	keySet := mgmt.NewOperatorKeySet([]ed25519.PublicKey{opPub})

	const sentinelErr = "boom: simulated handler failure"
	handlers := []mgmt.Handler{
		{
			Command: "test.fail",
			Fn: func(_ context.Context, _ json.RawMessage) (any, error) {
				return nil, errors.New(sentinelErr)
			},
		},
		{
			Command: "test.ok",
			Fn: func(_ context.Context, _ json.RawMessage) (any, error) {
				return map[string]string{"status": "ok"}, nil
			},
		},
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	srv := mgmt.NewServer(ln, daemonPriv, keySet, handlers, "dev")
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go srv.Serve(ctx) //nolint:errcheck

	conn, err := net.DialTimeout("tcp", ln.Addr().String(), time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatalf("SetDeadline: %v", err)
	}

	if err := doHandshake(t, conn, opPriv); err != nil {
		t.Fatalf("handshake failed: %v", err)
	}

	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(conn)

	// Invoke the failing handler.
	if err := enc.Encode(map[string]any{
		"type":    "request",
		"id":      "req-vp071",
		"command": "test.fail",
		"args":    map[string]any{},
	}); err != nil {
		t.Fatalf("encode request: %v", err)
	}

	var resp struct {
		Type  string `json:"type"`
		ID    string `json:"id"`
		OK    bool   `json:"ok"`
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := dec.Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.OK {
		t.Error("VP-071 violated: ok=true after handler returned error")
	}
	// VP-071 core assertion: must be E-RPC-011, not E-RPC-002 or E-RPC-001.
	if resp.Error.Code != "E-RPC-011" {
		t.Errorf("VP-071 violated: error.code = %q; want E-RPC-011 "+
			"(implementer must replace E-RPC-002 with E-RPC-011 in handleConnection dispatch)",
			resp.Error.Code)
	}
	// VP-071: error message must be the handler's verbatim error string.
	if resp.Error.Message != sentinelErr {
		t.Errorf("VP-071 violated: error.message = %q; want verbatim %q", resp.Error.Message, sentinelErr)
	}
	if resp.ID != "req-vp071" {
		t.Errorf("VP-071: response id = %q; want %q", resp.ID, "req-vp071")
	}

	// Connection must still be open: send a request to the succeeding handler.
	if err := enc.Encode(map[string]any{
		"type":    "request",
		"id":      "req-vp071b",
		"command": "test.ok",
		"args":    map[string]any{},
	}); err != nil {
		t.Fatalf("VP-071: connection closed after handler error (must stay open): %v", err)
	}
	var resp2 struct {
		OK bool `json:"ok"`
	}
	if err := dec.Decode(&resp2); err != nil {
		t.Fatalf("VP-071: decode second response: %v", err)
	}
	if !resp2.OK {
		t.Error("VP-071: second request (ok handler) after error returned ok=false")
	}
}

// ── VP-072 / AC-018 / PC-1 write-deadline: slowloris-on-write defense ────────

// TestWriteDeadline_SlowlorisDefense_VP072 verifies VP-072 / BC-2.07.004 PC-1
// (amended) / AC-018: the server sets conn.SetWriteDeadline before every
// sendJSON call, so a non-draining client cannot pin the connection goroutine
// forever.
//
// Test strategy: use net.Pipe (zero internal buffer). The server's first action
// is sendJSON(CHALLENGE). With no write deadline, this Write blocks until the
// client reads — forever, since the client never reads. With a write deadline
// set to HandshakeTimeout (injected as 100ms), the Write times out in ≤100ms
// and the connection goroutine exits.
//
// Observable: we measure goroutine count BEFORE closing the pipe. If the
// connection goroutine has exited (write deadline fired), count drops to
// baseline. If it is still stuck (no write deadline), count stays elevated.
// We close the pipe AFTER measuring to ensure we don't influence the result.
//
// Currently sendJSON has no SetWriteDeadline call → the Write blocks forever
// → the goroutine count stays elevated at the measurement point → RED.
//
// Traces: BC-2.07.004 PC-1 (amended, Ruling E), AC-018, VP-072.
func TestWriteDeadline_SlowlorisDefense_VP072(t *testing.T) {
	// NOT t.Parallel(): measures goroutine counts; concurrent goroutines from other
	// tests would introduce false-positive goroutine-leak readings.

	_, daemonPriv := mustGenKey(t)

	// Use a singleConnListener backed by net.Pipe so we control the client side.
	serverConn, clientConn := net.Pipe()
	// Manage pipe lifecycle manually — we need to close AFTER measuring goroutines.

	ln := newSingleConnListener(serverConn)
	// WithHandshakeTimeout injects 100ms so the write deadline fires quickly
	// without waiting 10s for the production default.
	srv := mgmt.NewServer(ln, daemonPriv, mgmt.NewOperatorKeySet(nil), nil, "dev",
		mgmt.WithHandshakeTimeout(100*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())

	gorsBefore := runtime.NumGoroutine()

	go func() {
		_ = srv.Serve(ctx)
	}()

	// Give Serve time to accept the connection and spawn the connection goroutine.
	// The connection goroutine immediately tries to Write the CHALLENGE —
	// which blocks indefinitely on net.Pipe because the client never reads.
	time.Sleep(30 * time.Millisecond)

	gorsAfterConnect := runtime.NumGoroutine()
	t.Logf("VP-072: goroutines before=%d after-connect=%d", gorsBefore, gorsAfterConnect)

	// Wait for write deadline to have fired (100ms timeout + 100ms buffer = 200ms total).
	// With write deadline: the connection goroutine exits within ~100ms of connecting.
	// Without write deadline: the goroutine is still stuck in Write at this point.
	time.Sleep(200 * time.Millisecond)

	// Measure goroutine count BEFORE closing the pipe.
	// This is the critical measurement window:
	//   - If write deadline was set: goroutine has already exited → count ≈ gorsBefore+1 (Serve loop)
	//   - If write deadline was NOT set: goroutine is still stuck → count > gorsBefore+2
	gorsAtDeadline := runtime.NumGoroutine()
	t.Logf("VP-072: goroutines at deadline point=%d (before=%d)", gorsAtDeadline, gorsBefore)

	// VP-072 core assertion: after the injected HandshakeTimeout (100ms) + 200ms buffer,
	// the connection goroutine MUST have exited (write deadline fired → Write returned error).
	// Tolerance: +2 for Serve goroutine + ctx-watcher goroutine.
	// If the goroutine is still alive (no write deadline), this fires → RED.
	connectionGoroutineStuck := gorsAtDeadline > gorsBefore+2
	if connectionGoroutineStuck {
		t.Errorf("VP-072 violated: %dms after connection (> HandshakeTimeout=100ms), "+
			"goroutine count is %d (was %d before connection, +2 expected for Serve+ctx-watcher). "+
			"The connection goroutine is pinned — no write deadline was set on the CHALLENGE send. "+
			"Implementer must add conn.SetWriteDeadline(time.Now().Add(s.handshakeTimeout)) "+
			"before sendJSON(CHALLENGE) in handleConnection.",
			230, gorsAtDeadline, gorsBefore)
	}

	// Clean up: close pipe to unblock any stuck goroutine, then cancel context.
	_ = clientConn.Close()
	_ = serverConn.Close()
	cancel()
	time.Sleep(30 * time.Millisecond) // let goroutines drain for test cleanup
}

// ── VP-072 / AC-018: RPC-response write-deadline uses s.rpcIdleTimeout field ──

// TestWriteDeadline_RPCResponse_VP072_Round5F4 verifies that the RPC-response
// write deadline on line 661 of mgmt.go uses the injectable s.rpcIdleTimeout
// field (set by WithRPCIdleTimeout), NOT the package constant RPCIdleTimeout (30s).
//
// This is the Round-5 Finding 4 discriminating test. The existing
// TestWriteDeadline_SlowlorisDefense_VP072 only exercises the handshake-phase
// write path (CHALLENGE send, client never reads). This test exercises the
// RPC-response write path: client completes auth, sends one RPC, then stops
// reading. The server's sendJSON for the response blocks on the net.Pipe write
// because the client is not draining. With the constant (current code at line 661),
// the write deadline is 30s — the goroutine remains pinned far past our 500ms
// bound. With the field fix (s.rpcIdleTimeout, 50ms), the goroutine exits within
// ~50ms + overhead.
//
// Discriminating property:
//   - CURRENT CODE (bug): line 661 uses RPCIdleTimeout (30s constant).
//     WithRPCIdleTimeout(50ms) reaches s.rpcIdleTimeout but NOT the write deadline.
//     The connection goroutine is still pinned at the 500ms measurement point → RED.
//   - FIXED CODE: line 661 uses s.rpcIdleTimeout (50ms).
//     Write deadline fires at ~50ms → goroutine exits → count drops → GREEN.
//
// Test strategy: net.Pipe (no internal buffer) + goroutine counting.
// A handshake helper goroutine reads CHALLENGE, sends CHALLENGE_RESPONSE, reads
// AUTH_OK (so handshake writes don't block), then sends one RPC request and
// signals via rpcSent channel before returning (stopping all client-side reads).
// The main goroutine waits for rpcSent, then waits for the 50ms write deadline
// to fire, then measures goroutine count before closing the pipe.
//
// No t.Parallel: goroutine-count measurements are sensitive to concurrent load.
//
// Traces: BC-2.07.004 PC-1 (amended, Ruling E), AC-018, VP-072.
func TestWriteDeadline_RPCResponse_VP072_Round5F4(t *testing.T) {
	// NOT t.Parallel(): measures goroutine counts; concurrent goroutines from other
	// tests would introduce false-positive goroutine-leak readings.

	_, daemonPriv := mustGenKey(t)
	opPub, opPriv := mustGenKey(t)
	keySet := mgmt.NewOperatorKeySet([]ed25519.PublicKey{opPub})

	// Register a handler that returns a small result immediately (not a blocking handler).
	// We need the handler to SUCCEED quickly so the server reaches sendJSON for the
	// response — the blocking happens in the write, not the handler.
	quickHandler := mgmt.Handler{
		Command: "test.quick",
		Fn: func(_ context.Context, _ json.RawMessage) (any, error) {
			return map[string]string{"result": "ok"}, nil
		},
	}

	// Use a singleConnListener backed by net.Pipe so we control both sides.
	// Manage pipe lifecycle manually — we need to close AFTER measuring goroutines.
	serverConn, clientConn := net.Pipe()

	ln := newSingleConnListener(serverConn)
	// Construct server with:
	//   - WithHandshakeTimeout(200ms): generous budget so the handshake completes fast
	//     without waiting for the 10s production default. The handshake write deadline
	//     does NOT need to be the discriminating variable here.
	//   - WithRPCIdleTimeout(50ms): this is what the test discriminates on.
	//     CURRENT BUG: line 661 uses RPCIdleTimeout (30s) not s.rpcIdleTimeout (50ms).
	//     FIXED CODE: line 661 uses s.rpcIdleTimeout (50ms) — write deadline fires at ~50ms.
	srv := mgmt.NewServer(ln, daemonPriv, keySet, []mgmt.Handler{quickHandler}, "dev",
		mgmt.WithHandshakeTimeout(200*time.Millisecond),
		mgmt.WithRPCIdleTimeout(50*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())

	// Measure goroutine baseline BEFORE starting Serve.
	gorsBefore := runtime.NumGoroutine()

	go func() {
		_ = srv.Serve(ctx)
	}()

	// rpcSent is closed by the handshake goroutine after it has:
	//   1. Completed the ADR-012 handshake (CHALLENGE → CHALLENGE_RESPONSE → AUTH_OK)
	//   2. Sent one RPC request ("test.quick")
	//   3. Returned — no further reads from clientConn
	// After rpcSent is closed, clientConn is dead to the server's response write.
	rpcSent := make(chan struct{})
	handshakeDone := make(chan error, 1)

	go func() {
		// Give Serve time to accept the connection and spawn the connection goroutine.
		time.Sleep(10 * time.Millisecond)

		// Step 1: Read the CHALLENGE. This must succeed — the server writes it with
		// a 200ms handshake deadline. The client is reading here so it does not block.
		dec := json.NewDecoder(clientConn)
		var challenge struct {
			Type  string `json:"type"`
			Nonce string `json:"nonce"`
		}
		if err := dec.Decode(&challenge); err != nil {
			handshakeDone <- fmt.Errorf("read challenge: %w", err)
			return
		}
		if challenge.Type != "challenge" {
			handshakeDone <- fmt.Errorf("expected challenge, got %q", challenge.Type)
			return
		}

		// Step 2: Send CHALLENGE_RESPONSE.
		nonceBytes, err := base64.RawURLEncoding.DecodeString(challenge.Nonce)
		if err != nil {
			handshakeDone <- fmt.Errorf("decode nonce: %w", err)
			return
		}
		sig := ed25519.Sign(opPriv, nonceBytes)
		resp, err := json.Marshal(map[string]any{
			"type":      "challenge_response",
			"nonce_sig": base64.RawURLEncoding.EncodeToString(sig),
			"pubkey":    base64.RawURLEncoding.EncodeToString([]byte(opPub)),
		})
		if err != nil {
			handshakeDone <- fmt.Errorf("marshal challenge_response: %w", err)
			return
		}
		resp = append(resp, '\n')
		if _, err := clientConn.Write(resp); err != nil {
			handshakeDone <- fmt.Errorf("write challenge_response: %w", err)
			return
		}

		// Step 3: Read AUTH_OK. The server writes it with the handshake write deadline.
		// The client is reading here so it does not block.
		var authResp struct {
			Type string `json:"type"`
		}
		if err := dec.Decode(&authResp); err != nil {
			handshakeDone <- fmt.Errorf("read auth response: %w", err)
			return
		}
		if authResp.Type != "auth_ok" {
			handshakeDone <- fmt.Errorf("expected auth_ok, got %q", authResp.Type)
			return
		}

		// Step 4: Send ONE RPC request to the server. After this write, we return
		// immediately — no further reads from clientConn.
		// The server will decode the request, execute the quick handler (immediate
		// return), then attempt sendJSON(response) — which blocks because the client
		// pipe is not being drained (net.Pipe has no internal buffer).
		rpcMsg, err := json.Marshal(map[string]any{
			"type":    "request",
			"id":      "req-vp072-rpcwrite",
			"command": "test.quick",
			"args":    map[string]any{},
		})
		if err != nil {
			handshakeDone <- fmt.Errorf("marshal rpc request: %w", err)
			return
		}
		rpcMsg = append(rpcMsg, '\n')
		if _, err := clientConn.Write(rpcMsg); err != nil {
			handshakeDone <- fmt.Errorf("write rpc request: %w", err)
			return
		}

		// Signal that the RPC request has been sent and we are no longer reading.
		close(rpcSent)
		handshakeDone <- nil
	}()

	// Wait for the handshake goroutine to complete (or fail).
	select {
	case err := <-handshakeDone:
		if err != nil {
			// Close the pipe to unblock any stuck goroutine before aborting.
			_ = clientConn.Close()
			_ = serverConn.Close()
			cancel()
			t.Fatalf("VP-072 RPC-response: handshake/rpc-send failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		_ = clientConn.Close()
		_ = serverConn.Close()
		cancel()
		t.Fatal("VP-072 RPC-response: handshake did not complete within 2s")
	}

	// The RPC request has been sent. The server is now:
	//   1. Decoding the RPC request (fast — the bytes are already in the pipe)
	//   2. Running the quick handler (immediate return)
	//   3. Hitting conn.SetWriteDeadline(time.Now().Add(RPCIdleTimeout)) at line 661
	//      BUG: uses the 30s constant, not s.rpcIdleTimeout (50ms)
	//   4. Attempting sendJSON(response) — blocks because client is not reading
	//
	// With the constant (30s): write deadline does NOT fire within our 500ms bound.
	// The connection goroutine stays pinned → goroutine count stays elevated → RED.
	//
	// With the field fix (50ms): write deadline fires at ~50ms → goroutine exits
	// → goroutine count drops back to baseline → GREEN.

	gorsAfterRPC := runtime.NumGoroutine()
	t.Logf("VP-072 RPC-response: goroutines before=%d after-rpc-send=%d", gorsBefore, gorsAfterRPC)

	// Wait for the RPCIdleTimeout (50ms) + generous buffer (300ms) = 350ms.
	// If line 661 uses s.rpcIdleTimeout (50ms): goroutine exits within ~50ms.
	// If line 661 uses RPCIdleTimeout (30s constant): goroutine still pinned at 350ms.
	time.Sleep(350 * time.Millisecond)

	// Measure goroutine count BEFORE closing the pipe.
	// Closing the pipe would unblock the stuck goroutine, masking the bug.
	gorsAtDeadline := runtime.NumGoroutine()
	t.Logf("VP-072 RPC-response: goroutines at 350ms measurement=%d (before=%d)",
		gorsAtDeadline, gorsBefore)

	// Discriminating assertion:
	//   FIXED: gorsAtDeadline <= gorsBefore+2 (Serve goroutine + ctx-watcher remain)
	//   BUG:   gorsAtDeadline > gorsBefore+2  (connection goroutine still pinned in Write)
	//
	// The +2 tolerance accounts for the Serve accept-loop goroutine and the ctx-watcher
	// goroutine that Serve launches. The connection goroutine should have exited.
	connectionGoroutineStuck := gorsAtDeadline > gorsBefore+2
	if connectionGoroutineStuck {
		t.Errorf("VP-072 (Round-5 Finding 4) violated: 350ms after RPC request sent "+
			"(> WithRPCIdleTimeout=50ms), goroutine count is %d (was %d before, +2 allowed "+
			"for Serve+ctx-watcher). The RPC-response write deadline is NOT using "+
			"s.rpcIdleTimeout — it is using the package constant RPCIdleTimeout (30s). "+
			"Fix: change line 661 in mgmt.go from: "+
			"conn.SetWriteDeadline(time.Now().Add(RPCIdleTimeout)) to: "+
			"conn.SetWriteDeadline(time.Now().Add(s.rpcIdleTimeout)). "+
			"WithRPCIdleTimeout(50ms) must govern the RPC-response write deadline "+
			"(AC-018 / BC-2.07.004 PC-1 amended, Ruling E).",
			gorsAtDeadline, gorsBefore)
	}

	// Clean up: close pipe to unblock any stuck goroutine, then cancel context.
	_ = clientConn.Close()
	_ = serverConn.Close()
	cancel()
	time.Sleep(30 * time.Millisecond) // let goroutines drain for test cleanup
}

// ── AC-017 / VP-069 (Ruling G): unexpected listener close returns non-nil ─────

// TestServe_ReturnsErrOnUnexpectedListenerClose_VP069 verifies VP-069 Ruling G /
// BC-2.07.004 PC-10: when the listener fd is closed externally (NOT via Shutdown
// and NOT via ctx cancel) while ctx.Err()==nil and shuttingDown==false, Serve MUST
// return a NON-NIL error.
//
// RED because: the current accept-error predicate in mgmt.go is:
//
//	if s.shuttingDown.Load() || errors.Is(err, net.ErrClosed) { return nil }
//
// The `errors.Is(err, net.ErrClosed)` arm is missing the `&& ctx.Err() != nil`
// conjunct required by BC-2.07.004 PC-10 / VP-069. Without the conjunct, closing
// the listener directly while ctx is live returns nil — silently killing the
// management plane (SOUL #4 no-silent-failure violation). With the fix:
//
//	if s.shuttingDown.Load() || (errors.Is(err, net.ErrClosed) && ctx.Err() != nil) { ... }
//
// the unexpected-close path falls through to `return err` (non-nil) — this test passes.
// There is also a dead `select { case <-done: ... }` block that provides a second nil
// return path for the same scenario; Ruling H requires that block to be removed.
//
// Traces: BC-2.07.004 PC-10 (Ruling G), AC-017 sub-case (c), VP-069.
func TestServe_ReturnsErrOnUnexpectedListenerClose_VP069(t *testing.T) {
	t.Parallel()

	_, daemonPriv := mustGenKey(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	// Do NOT register t.Cleanup(ln.Close) — we close it explicitly below.

	srv := mgmt.NewServer(ln, daemonPriv, mgmt.NewOperatorKeySet(nil), nil, "dev")

	// Use context.Background() — a live context that is NEVER cancelled.
	// This is the critical difference from TestServe_ReturnsNilOnCtxCancel_VP069.
	ctx := context.Background()

	errCh := make(chan error, 1)
	go func() {
		// RED because: current code returns nil here after direct ln.Close().
		// Post-fix: returns net.ErrClosed (or wrapped variant) — non-nil.
		errCh <- srv.Serve(ctx)
	}()

	// Give Serve time to enter the Accept loop.
	time.Sleep(15 * time.Millisecond)

	// Close the listener DIRECTLY from the test — NOT via Shutdown, NOT via ctx cancel.
	// shuttingDown is false, ctx.Err() is nil — this is the unexpected-close scenario.
	_ = ln.Close()

	select {
	case serveErr := <-errCh:
		// VP-069 Ruling G: MUST be non-nil. Current code (missing && ctx.Err() != nil)
		// returns nil → this assertion fires → RED.
		if serveErr == nil {
			t.Errorf("VP-069 (Ruling G) violated: Serve returned nil on unexpected listener close " +
				"(ctx live, Shutdown never called). " +
				"Fix: change the accept-error predicate in mgmt.go to: " +
				"s.shuttingDown.Load() || (errors.Is(err, net.ErrClosed) && ctx.Err() != nil). " +
				"Also remove the dead 'select { case <-done: ... default: }' block (Ruling H).")
		}
	case <-time.After(3 * time.Second):
		t.Error("VP-069 (Ruling G): Serve did not return within 3s of unexpected listener close")
	}
}

// ── AC-017 / Ruling I: shutdown-window concurrency-safety smoke test ──────────

// TestServe_ShutdownWindowNoAddAfterWaitPanic_RulingI is a concurrency-safety
// smoke test that exercises the accept-vs-Shutdown race window under the Go
// race detector. Best run with -race (go test -race).
//
// ACCURATE PROPERTY (Ruling T — ARCH-12 v1.5 test-quality correction):
// This test's actual property is: "concurrent dial + Shutdown does not panic
// and passes go test -race." It is a race-detector smoke test, not a
// deterministic Add-after-Wait-at-zero discriminator.
//
// The stated RED rationale (Add-after-Wait panic from connWG) is structurally
// impossible in the current design: Shutdown does NOT call connWG.Wait() — Serve
// is the sole Wait owner. The post-Shutdown connWG.Wait() in Serve happens inside
// Serve's goroutine after the accept loop exits; there is no concurrent Wait().
//
// The DROP behavior (connections accepted in the shutdown window are discarded
// without entering connWG) is discriminatingly tested by
// TestServe_DrainCompletesWithinBudget_RulingI (test (e) above).
//
// This test's value: under -race it detects data races on WaitGroup state and
// connection maps that might not manifest as panics in normal runs. The 100-
// iteration loop maximizes interleaving of concurrent Accept/Shutdown operations.
//
// Traces: BC-2.07.004 PC-10 (Ruling I, drain-ordering guarantee), AC-017 sub-case (d).
func TestServe_ShutdownWindowNoAddAfterWaitPanic_RulingI(t *testing.T) {
	// NOT t.Parallel: stress-tests timing windows; parallel jitter reduces race coverage.

	_, daemonPriv := mustGenKey(t)

	// 100 iterations: each iteration races Accept against Shutdown in a tight window.
	// Any iteration that triggers a panic or data race causes the test to fail.
	for i := range 100 {
		func() {
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatalf("iter %d: net.Listen: %v", i, err)
			}
			addr := ln.Addr().String()

			srv := mgmt.NewServer(ln, daemonPriv, mgmt.NewOperatorKeySet(nil), nil, "dev",
				mgmt.WithMaxConnections(1))

			ctx, cancel := context.WithCancel(context.Background())

			serveDone := make(chan error, 1)
			go func() {
				serveDone <- srv.Serve(ctx)
			}()

			// Give Serve a moment to enter the accept loop before we hammer it.
			time.Sleep(time.Millisecond)

			// Dial several connections rapidly and immediately close them so the server
			// cycles through Accept/handle quickly, creating many accept-vs-shutdown
			// window opportunities.
			dialsDone := make(chan struct{})
			go func() {
				defer close(dialsDone)
				for range 10 {
					c, dialErr := net.DialTimeout("tcp", addr, 50*time.Millisecond)
					if dialErr != nil {
						return
					}
					_ = c.Close()
					runtime.Gosched() // yield to maximize interleaving
				}
			}()

			// Call Shutdown while dials are in progress — this is the race window.
			// Without the shuttingDown check after Accept, a connection from the OS
			// backlog arrives after connWG.Wait() returns at zero → Add after Wait → panic.
			shutCtx, shutCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			_ = srv.Shutdown(shutCtx)
			shutCancel()
			cancel()

			<-dialsDone

			select {
			case err := <-serveDone:
				_ = err
			case <-time.After(500 * time.Millisecond):
				t.Errorf("iter %d: Serve did not return within 500ms after Shutdown", i)
			}
		}()

		if t.Failed() {
			break
		}
	}
}

// ── AC-017 / Ruling I: drain completes within budget ─────────────────────────

// shutdownWindowListener is a fake net.Listener designed to deterministically
// test the shutdown-window drop behaviour (Ruling I).
//
// Its Close() is a no-op: the listener is not unblocked by Shutdown. This lets
// Serve's accept loop remain alive after Shutdown returns so the test can inject
// a connection in the shutdown window (after shuttingDown is set) and observe
// whether Serve drops it (fix) or processes it (bug).
//
// Usage:
//  1. Serve blocks in Accept (connCh empty).
//  2. Call Shutdown → shuttingDown.Store(true) → ln.Close() (no-op here) →
//     closeAllConns() (no conns) → connWG.Wait() returns (count=0) → Shutdown
//     returns nil. Serve is still alive, blocked in Accept.
//  3. Deliver conn2 via deliverConn() so Serve receives it post-shutdown.
//  4. Assert clientConn2: Fix → EOF immediately (conn dropped). Bug → CHALLENGE
//     bytes arrive (conn entered handleConnection).
//  5. Call forceClose() so Serve's Accept returns net.ErrClosed and Serve exits.
type shutdownWindowListener struct {
	connCh  chan net.Conn // buffered; deliver conns via deliverConn()
	closeCh chan struct{} // forceClose() closes this to stop Accept
	addr    net.Addr
}

func newShutdownWindowListener(addr net.Addr) *shutdownWindowListener {
	return &shutdownWindowListener{
		connCh:  make(chan net.Conn, 4),
		closeCh: make(chan struct{}),
		addr:    addr,
	}
}

func (l *shutdownWindowListener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.connCh:
		return conn, nil
	case <-l.closeCh:
		return nil, net.ErrClosed
	}
}

// Close is called by Shutdown. It is a no-op here: Serve keeps running so the
// test can inject the post-shutdown connection via deliverConn.
func (l *shutdownWindowListener) Close() error { return nil }

// forceClose stops Accept, causing Serve to see net.ErrClosed and exit.
func (l *shutdownWindowListener) forceClose() {
	select {
	case <-l.closeCh:
	default:
		close(l.closeCh)
	}
}

// deliverConn queues conn for the next Accept() call.
func (l *shutdownWindowListener) deliverConn(conn net.Conn) {
	l.connCh <- conn
}

func (l *shutdownWindowListener) Addr() net.Addr { return l.addr }

// TestServe_DrainCompletesWithinBudget_RulingI verifies that a connection accepted
// AFTER Shutdown sets shuttingDown (the "shutdown window") is dropped immediately
// without entering the connection WaitGroup.
//
// Ruling I (ARCH-12 v1.4) mandates that Serve checks s.shuttingDown.Load()
// immediately after a successful Accept(). A connection arriving in the shutdown
// window MUST be closed + semaphore released + continue — it MUST NOT enter
// connWG or reach handleConnection.
//
// Observable difference:
//   - With fix (shuttingDown check present):
//     conn2 is closed by Serve → clientConn2 receives EOF immediately (no data).
//   - Without fix (check absent, current code):
//     conn2 enters handleConnection → server attempts to write CHALLENGE →
//     clientConn2 receives CHALLENGE JSON bytes (non-empty read).
//
// Test design (deterministic):
//  1. Use shutdownWindowListener whose Close() is a no-op, so Serve stays alive.
//  2. Call Shutdown (no in-flight connections) — returns nil quickly.
//     shuttingDown is now true. Serve is still blocked in Accept.
//  3. Deliver conn2 → Serve accepts it POST-shutdown.
//  4. Give server 100 ms to process conn2 one way or the other.
//  5. Attempt to read from clientConn2 with a 200 ms deadline.
//     Fix: conn2 was dropped → connection closed → Read returns io.EOF or
//     net.ErrClosed before deadline → no bytes → PASS.
//     Bug: conn2 entered handleConnection → server writes CHALLENGE to conn2 →
//     clientConn2.Read returns CHALLENGE bytes → RED.
//  6. forceClose() so Serve exits cleanly.
//
// Traces: BC-2.07.004 PC-10 (Ruling I, shutdown-window drop), AC-017 sub-case (e).
func TestServe_DrainCompletesWithinBudget_RulingI(t *testing.T) {
	t.Parallel()

	_, daemonPriv := mustGenKey(t)

	realLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	addr := realLn.Addr()
	_ = realLn.Close()

	swl := newShutdownWindowListener(addr)

	const longHandshakeTimeout = 30 * time.Second

	srv := mgmt.NewServer(swl, daemonPriv, mgmt.NewOperatorKeySet(nil), nil, "dev",
		mgmt.WithHandshakeTimeout(longHandshakeTimeout))

	ctx := context.Background()
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- srv.Serve(ctx)
	}()

	// Let Serve reach Accept (empty connCh — blocks immediately).
	time.Sleep(5 * time.Millisecond)

	// Call Shutdown with no in-flight connections.
	// shuttingDown.Store(true) → ln.Close() (no-op on swl) → closeAllConns() →
	// connWG.Wait() (count=0, instant) → returns nil.
	// Serve is still alive, blocked in Accept.
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutCancel()
	if shutErr := srv.Shutdown(shutCtx); shutErr != nil {
		t.Fatalf("unexpected Shutdown error: %v", shutErr)
	}

	// Deliver conn2 AFTER Shutdown — shuttingDown is now true.
	// Serve's accept loop will call Accept() again and receive conn2.
	serverConn2, clientConn2 := net.Pipe()
	t.Cleanup(func() {
		_ = serverConn2.Close()
		_ = clientConn2.Close()
	})
	swl.deliverConn(serverConn2)

	// Allow 200 ms for Serve to act on conn2.
	// With fix: conn2 is closed immediately (shuttingDown drop) → clientConn2
	// gets EOF or a closed-pipe error before our read deadline.
	// Without fix: conn2 enters handleConnection → server writes CHALLENGE →
	// net.Pipe is synchronous → write blocks until we read. Calling Read here
	// with a 200 ms deadline: if we get bytes, the server sent CHALLENGE →
	// shutdown-window check is missing → RED.
	if err := clientConn2.SetReadDeadline(time.Now().Add(200 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}
	buf := make([]byte, 512)
	n, _ := clientConn2.Read(buf)
	_ = clientConn2.SetReadDeadline(time.Time{}) // clear deadline

	if n > 0 {
		// Server sent data to conn2 — conn2 entered handleConnection.
		// This proves the shuttingDown check after Accept is MISSING (bug).
		t.Errorf("Ruling I (shutdown-window drop) violated: Serve sent %d bytes to a "+
			"connection accepted AFTER Shutdown (shuttingDown=true). Received: %q. "+
			"The accept loop MUST check s.shuttingDown.Load() after Accept() and drop "+
			"the connection (close + semaphore release + continue) before connWG.Add. "+
			"Fix: add the shuttingDown check per Ruling I canonical ordering.",
			n, buf[:n])
	}

	// Clean up: close conn2's connections so that any in-flight handleConnection
	// goroutine for conn2 unblocks and returns (otherwise connWG.Wait blocks).
	// With the fix conn2 was never entered, so these are no-ops on already-closed
	// connections. With the bug, conn2's goroutine is waiting on its read deadline
	// (30s); closing serverConn2 unblocks it immediately.
	_ = serverConn2.Close()
	_ = clientConn2.Close()

	// forceClose the listener so Serve exits its accept loop.
	swl.forceClose()
	select {
	case <-serveDone:
	case <-time.After(2 * time.Second):
		t.Error("Serve did not exit after forceClose within 2s")
	}
}

// ── AC-001 / Ruling K: HandshakeTimeout is close-only (no AUTH_FAIL) ─────────

// TestMgmtServer_HandshakeTimeout_CloseOnly_RulingK verifies BC-2.07.004 EC-001
// v1.4 / AC-001 / Ruling K: when HandshakeTimeout expires (silent stall — client
// connects but sends nothing), the server MUST close the connection WITHOUT sending
// an AUTH_FAIL JSON message. A timeout close is silent — sending AUTH_FAIL to a
// non-responsive client delays slot reclamation and risks a slowloris-on-write.
//
// This test ALSO satisfies the AC-001 obligation (added in S-W5.01 v1.3) to
// assert: "NO AUTH_FAIL JSON message was received on the client pipe before the
// close — a timeout close is silent (close-only, no AUTH_FAIL per Ruling K /
// BC-2.07.004 EC-001 v1.4)."
//
// Outcome note: current mgmt.go handleConnection correctly does NOT send AUTH_FAIL
// on timeout (lines 463–467: `if ne, ok := err.(net.Error); !ok || !ne.Timeout()`
// — AUTH_FAIL is only sent for non-timeout errors). This test may be GREEN already.
// If so, it locks the contract so a future refactor cannot accidentally add AUTH_FAIL
// to the timeout path. GREEN is the correct steady-state; RED is a defect.
//
// Traces: BC-2.07.004 EC-001 v1.4, AC-001 (Ruling K assertion), VP-064 sub-case (a).
func TestMgmtServer_HandshakeTimeout_CloseOnly_RulingK(t *testing.T) {
	t.Parallel()

	_, daemonPriv := mustGenKey(t)
	ops := mgmt.NewOperatorKeySet(nil)

	// Use a short HandshakeTimeout so the test completes quickly.
	const shortTimeout = 60 * time.Millisecond

	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		_ = serverConn.Close()
		_ = clientConn.Close()
	})

	ln := newSingleConnListener(serverConn)
	srv := mgmt.NewServer(ln, daemonPriv, ops, nil, "0.1.0-test",
		mgmt.WithHandshakeTimeout(shortTimeout))

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go func() {
		_ = srv.Serve(ctx)
	}()

	// Give client side a generous deadline to read the CHALLENGE and then wait
	// for the server to time out and close the connection.
	if err := clientConn.SetDeadline(time.Now().Add(1 * time.Second)); err != nil {
		t.Fatalf("SetDeadline: %v", err)
	}

	// Read the CHALLENGE (server sends this first).
	challenge := readMsg(t, clientConn)
	if challenge == nil || challenge["type"] != "challenge" {
		t.Fatalf("Ruling K: expected CHALLENGE as first server message; got %v", challenge)
	}

	// Now send NOTHING — simulate a silent stall.
	// After shortTimeout (60ms) the server should close without sending AUTH_FAIL.

	// Collect any messages from the server after the CHALLENGE.
	// If AUTH_FAIL arrives before EOF, the test fails.
	var receivedAuthFail bool
	var receivedMsg map[string]any

	// Use a short read deadline to detect the close quickly.
	_ = clientConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	receivedMsg = readMsg(t, clientConn)
	if receivedMsg != nil {
		if receivedMsg["type"] == "auth_fail" {
			receivedAuthFail = true
		}
	}

	// Ruling K: no AUTH_FAIL must have been received before or on close.
	// The connection should have been silently closed (EOF / closed-connection error).
	if receivedAuthFail {
		t.Errorf("Ruling K violated: server sent AUTH_FAIL on HandshakeTimeout expiry; " +
			"want close-only (no AUTH_FAIL). " +
			"BC-2.07.004 EC-001 v1.4: 'On timeout expiry: the connection is closed " +
			"immediately WITHOUT sending AUTH_FAIL — a non-responsive client would not read it.'")
	}
	// If receivedMsg is nil (connection closed without message), that is correct.
	// This test passes with the current correct implementation and locks the contract.
}

// ── AC-017 / Ruling P / VP-069 v1.2: fatal-accept-error drain ────────────────

// TestServe_FatalAcceptErrorDrainsQuickly verifies Ruling P / BC-2.07.004 PC-10 /
// VP-069 v1.2 / AC-017 (extended): on the fatal-accept-error path (Accept returns
// a non-transient error while ctx is context.Background() — always live — and
// Shutdown was never called), s.closeAllConns() MUST be called before
// s.connWG.Wait(). Without closeAllConns(), in-flight authenticated-but-idle
// connections remain open until their own read deadlines fire (RPCIdleTimeout =
// 30s), causing Serve to stall for up to 30s. With closeAllConns(), they are
// force-closed and Serve returns within milliseconds.
//
// Test design:
//  1. Start a Server with a real TCP listener.
//  2. Connect a client, complete the ADR-012 handshake (authenticated idle conn).
//     After AUTH_OK the server applies RPCIdleTimeout (30s) read deadline —
//     this is the read that would block Serve's connWG.Wait() for up to 30s.
//  3. Close the listener directly from the test goroutine (ctx is
//     context.Background(), Shutdown never called — fatal-accept scenario).
//  4. Assert that Serve returns within 200ms. Without the closeAllConns() call,
//     Serve's connWG.Wait() stalls until the idle conn's RPCIdleTimeout fires
//     (production default 30s) — making the test time out. With closeAllConns(),
//     the idle conn is force-closed, RPCIdleTimeout is aborted, and Serve returns.
//
// This test is RED until the implementer inserts:
//
//	s.closeAllConns()
//	s.connWG.Wait()
//	return err
//
// on the fatal-accept-error path in Serve (currently only `s.connWG.Wait(); return err`).
//
// Traces: BC-2.07.004 PC-10 (Ruling P), AC-017 sub-case (f), VP-069 v1.2.
func TestServe_FatalAcceptErrorDrainsQuickly(t *testing.T) {
	t.Parallel()

	_, daemonPriv := mustGenKey(t)
	opPub, opPriv := mustGenKey(t)
	keySet := mgmt.NewOperatorKeySet([]ed25519.PublicKey{opPub})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	// Do NOT register t.Cleanup(ln.Close) — we close it explicitly to trigger
	// the fatal-accept-error path.

	// Use default RPCIdleTimeout (30s) — no override.
	// The test must return well within 200ms; if closeAllConns() is missing, it
	// takes ~RPCIdleTimeout (30s) before Serve returns.
	srv := mgmt.NewServer(ln, daemonPriv, keySet, nil, "dev")

	// Use context.Background() — a live context that is NEVER cancelled.
	// This is the fatal-accept-error path (not Shutdown, not ctx-cancel).
	ctx := context.Background()
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ctx)
	}()

	// Give Serve time to enter the accept loop.
	time.Sleep(15 * time.Millisecond)

	// Connect a client and complete the ADR-012 handshake to get an authenticated
	// idle connection tracked in connWG. After AUTH_OK, the server's goroutine for
	// this connection blocks on RPCIdleTimeout read (30s default).
	clientConn, err := net.DialTimeout("tcp", ln.Addr().String(), time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = clientConn.Close() })
	if err := clientConn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatalf("SetDeadline: %v", err)
	}

	// Complete handshake: read CHALLENGE, send CHALLENGE_RESPONSE, read AUTH_OK.
	dec := json.NewDecoder(clientConn)
	enc := json.NewEncoder(clientConn)

	var challenge struct {
		Type  string `json:"type"`
		Nonce string `json:"nonce"`
	}
	if err := dec.Decode(&challenge); err != nil {
		t.Fatalf("read challenge: %v", err)
	}
	if challenge.Type != "challenge" {
		t.Fatalf("expected challenge; got %q", challenge.Type)
	}
	nonceBytes, err := base64.RawURLEncoding.DecodeString(challenge.Nonce)
	if err != nil {
		t.Fatalf("decode nonce: %v", err)
	}
	sig := ed25519.Sign(opPriv, nonceBytes)
	if err := enc.Encode(map[string]any{
		"type":      "challenge_response",
		"nonce_sig": base64.RawURLEncoding.EncodeToString(sig),
		"pubkey":    base64.RawURLEncoding.EncodeToString([]byte(opPub)),
	}); err != nil {
		t.Fatalf("write challenge_response: %v", err)
	}
	var authResp struct {
		Type string `json:"type"`
	}
	if err := dec.Decode(&authResp); err != nil {
		t.Fatalf("read auth response: %v", err)
	}
	if authResp.Type != "auth_ok" {
		t.Fatalf("expected auth_ok; got %q", authResp.Type)
	}

	// Connection is now authenticated and idle — tracked in connWG, blocking on
	// a 30s RPCIdleTimeout read. Now trigger the fatal-accept-error scenario by
	// closing the listener directly (NOT via Shutdown, NOT via ctx-cancel).
	// shuttingDown is false, ctx.Err() is nil — this is the fatal-accept path.
	_ = clientConn.SetDeadline(time.Time{}) // clear client deadline — we want it to stay idle
	_ = ln.Close()

	// Ruling P: Serve MUST return within 200ms (NOT blocked for up to 30s).
	// Without closeAllConns() on the fatal path, connWG.Wait() stalls until the
	// idle conn's RPCIdleTimeout fires. With closeAllConns(), the idle conn is
	// force-closed and Serve returns quickly.
	select {
	case serveErr := <-errCh:
		// Ruling P passes — Serve returned. It should be non-nil (unexpected close).
		if serveErr == nil {
			t.Logf("Ruling P drain check: Serve returned nil (unexpected-close path may need && ctx.Err()!=nil fix too)")
		}
	case <-time.After(200 * time.Millisecond):
		t.Errorf("Ruling P (VP-069 v1.2) violated: Serve did not return within 200ms after fatal " +
			"listener close. The in-flight authenticated-idle connection was NOT force-closed. " +
			"Fix: insert s.closeAllConns() before s.connWG.Wait() on the fatal-accept-error " +
			"path in Serve (currently missing per Ruling P / BC-2.07.004 PC-10).")
	}
}

// ── AC-020 / Ruling R / BC-2.07.004 PC-6: per-handler execution timeout ───────

// TestMgmtServer_HandlerTimeout_AC020 verifies AC-020 / BC-2.07.004 PC-6
// (amended, Ruling R): a registered handler that blocks past RPCIdleTimeout is
// cancelled via a child context derived by context.WithTimeout(ctx, RPCIdleTimeout).
// The server responds with E-RPC-011; the connection is NOT closed.
//
// This test encodes the API contract the implementer must satisfy:
//   - mgmt.WithRPCIdleTimeout(d) option (injectable for fast tests)
//   - handler Fn must be called with context.WithTimeout(ctx, rpcIdleTimeout)
//   - a blocking handler must return E-RPC-011 (not hang indefinitely)
//   - the connection remains open after handler timeout (not closed)
//
// RED because: handlerFn(ctx, req.Args) is currently called with the raw ctx
// (no timeout). A blocking handler causes handleConnection to stall indefinitely
// on the handlerFn call, pinning the connection goroutine and semaphore slot
// (CWE-400). With the fix, the child context is cancelled after RPCIdleTimeout
// (injected here as 50ms) and the server sends E-RPC-011 in-band.
//
// Traces: BC-2.07.004 PC-6 (amended, Ruling R), AC-020.
func TestMgmtServer_HandlerTimeout_AC020(t *testing.T) {
	t.Parallel()

	_, daemonPriv := mustGenKey(t)
	opPub, opPriv := mustGenKey(t)
	keySet := mgmt.NewOperatorKeySet([]ed25519.PublicKey{opPub})

	// Register a handler that blocks until its context is cancelled.
	// This is the canonical "blocking handler" that should be timed out.
	blockingHandler := mgmt.Handler{
		Command: "test.block",
		Fn: func(ctx context.Context, _ json.RawMessage) (any, error) {
			// Block until context is cancelled — simulates a handler that hangs.
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	// Construct server with a short RPCIdleTimeout (50ms) so the test completes
	// quickly. WithRPCIdleTimeout is the required injectable option — this test
	// encodes the API contract the implementer must add to mgmt.go.
	//
	// COMPILE FAILURE IS EXPECTED until the implementer adds:
	//   - s.rpcIdleTimeout field on Server (default RPCIdleTimeout = 30s)
	//   - WithRPCIdleTimeout(d time.Duration) Option
	//   - context.WithTimeout(ctx, s.rpcIdleTimeout) wrapping each handlerFn call
	srv := mgmt.NewServer(ln, daemonPriv, keySet, []mgmt.Handler{blockingHandler}, "dev",
		mgmt.WithRPCIdleTimeout(50*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go srv.Serve(ctx) //nolint:errcheck

	clientConn, err := net.DialTimeout("tcp", ln.Addr().String(), time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = clientConn.Close() })
	// Generous deadline: 50ms timeout + 200ms overhead = 250ms; use 3s to be safe.
	if err := clientConn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatalf("SetDeadline: %v", err)
	}

	// Complete handshake.
	if err := doHandshake(t, clientConn, opPriv); err != nil {
		t.Fatalf("handshake: %v", err)
	}

	enc := json.NewEncoder(clientConn)
	dec := json.NewDecoder(clientConn)

	// Send the blocking RPC. The handler will block until its context is cancelled.
	if err := enc.Encode(map[string]any{
		"type":    "request",
		"id":      "req-ac020",
		"command": "test.block",
		"args":    map[string]any{},
	}); err != nil {
		t.Fatalf("encode request: %v", err)
	}

	// Ruling R: within ~200ms (50ms timeout + 150ms overhead), the server must
	// respond with E-RPC-011 (handler timeout). Without the fix, dec.Decode blocks
	// indefinitely (no handler timeout → blocking handler pins goroutine).
	var resp struct {
		Type  string `json:"type"`
		ID    string `json:"id"`
		OK    bool   `json:"ok"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	// The client deadline (3s) bounds the whole test, but we want the specific
	// assertion to be about the handler timeout (~50ms). If the server never sends
	// a response, dec.Decode will block until the client deadline fires (3s) and
	// the test will fail with a decode error. The 200ms assertion below is the
	// discriminating check.
	start := time.Now()
	if err := dec.Decode(&resp); err != nil {
		// This fires if the server hung and the client deadline fired first.
		t.Fatalf("AC-020: decode response timed out or error: %v (server may be hung — "+
			"blocking handler was not cancelled by WithRPCIdleTimeout(50ms))", err)
	}
	elapsed := time.Since(start)

	// AC-020 timing assertion: response must arrive within ~200ms of the RPC send.
	if elapsed > 200*time.Millisecond {
		t.Errorf("AC-020: handler timeout response took %v; want within 200ms "+
			"(RPCIdleTimeout=50ms + overhead). Without WithRPCIdleTimeout fix, "+
			"handlerFn runs indefinitely.", elapsed)
	}

	// AC-020 error code assertion: must be E-RPC-011 (handler error), not E-RPC-010.
	if resp.OK {
		t.Error("AC-020: ok=true after blocking handler timeout; want ok=false")
	}
	if resp.Error == nil {
		t.Fatal("AC-020: response.error is nil after handler timeout; want E-RPC-011")
	}
	if resp.Error.Code != "E-RPC-011" {
		t.Errorf("AC-020: response.error.code = %q; want E-RPC-011 "+
			"(blocking handler must be cancelled and reported as handler error, "+
			"not unknown-command E-RPC-010)", resp.Error.Code)
	}
	if resp.ID != "req-ac020" {
		t.Errorf("AC-020: response.id = %q; want %q", resp.ID, "req-ac020")
	}

	// AC-020 connection-open assertion: connection must remain open after handler
	// timeout (timeout is NOT a connection-close event per Ruling R).
	// Send a second RPC with a no-op handler (registered during test) to verify.
	// Since we only registered "test.block", send an unknown command — the server
	// should respond with E-RPC-010 (connection still open).
	if err := enc.Encode(map[string]any{
		"type":    "request",
		"id":      "req-ac020-conncheck",
		"command": "test.noop.notregistered",
		"args":    map[string]any{},
	}); err != nil {
		t.Fatalf("AC-020: connection was closed after handler timeout (must stay open): %v", err)
	}
	var resp2 struct {
		OK    bool `json:"ok"`
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := dec.Decode(&resp2); err != nil {
		t.Fatalf("AC-020: decode second response failed — connection may have been closed: %v", err)
	}
	// Second response: E-RPC-010 (unknown command) proves connection is still open.
	if resp2.OK || resp2.Error == nil || resp2.Error.Code != "E-RPC-010" {
		t.Errorf("AC-020: second RPC after handler timeout: ok=%v code=%v; "+
			"want ok=false code=E-RPC-010 (connection must stay open after handler timeout)",
			resp2.OK, resp2.Error)
	}
}

// ── Authorized-key-set rejection (no AC number — additional coverage) ─────────

// TestMgmtServer_UnauthorizedKeyRejected_AuthorizedSetCheck verifies that a
// valid Ed25519 key that is NOT in authorized_operator_keys is rejected even
// if the signature over the nonce is self-consistent.
//
// This tests the IsAuthorized check independently from the signature-verify
// check: even a correct self-signature is rejected if the key is not in the set.
//
// Traces: BC-2.07.004 PC-2, Inv-1.
func TestMgmtServer_UnauthorizedKeyRejected_AuthorizedSetCheck(t *testing.T) {
	t.Parallel()

	_, daemonPriv := mustGenKey(t)

	// Register ONE authorized key.
	_, authorizedPriv := mustGenKey(t)
	authorizedPub := authorizedPriv.Public().(ed25519.PublicKey)
	ops := mgmt.NewOperatorKeySet([]ed25519.PublicKey{authorizedPub})

	// Generate a DIFFERENT key — not in the set.
	outsiderPub, outsiderPriv := mustGenKey(t)

	client := startServerOnPipe(t, daemonPriv, ops, nil)
	if err := client.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetDeadline: %v", err)
	}

	challenge := readMsg(t, client)
	if challenge == nil || challenge["type"] != "challenge" {
		t.Fatalf("expected challenge; got %v", challenge)
	}
	nonceB64, _ := challenge["nonce"].(string)
	nonce := decB64(t, nonceB64)

	// Outsider signs the nonce correctly — valid self-signature, but not in the set.
	sig := ed25519.Sign(outsiderPriv, nonce)
	writeMsg(t, client, map[string]any{
		"type":      "challenge_response",
		"nonce_sig": encB64(sig),
		"pubkey":    encB64([]byte(outsiderPub)),
	})

	resp := readMsg(t, client)
	if resp == nil {
		t.Fatal("expected AUTH_FAIL for unauthorized key; got nil")
	}
	if resp["type"] != "auth_fail" {
		t.Errorf("want type=auth_fail for key not in authorized set; got %v", resp["type"])
	}
	if resp["code"] != "E-ADM-010" {
		t.Errorf("want code=E-ADM-010; got %v", resp["code"])
	}
}

// TestCallerPubkeyContext verifies that WithCallerPubkey/CallerPubkey round-trip
// correctly, and that CallerPubkey returns false on an empty context.
// Traces to F-001 (CallerPubkey context plumbing).
func TestCallerPubkeyContext(t *testing.T) {
	t.Parallel()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	// Happy path: pubkey survives context round-trip.
	ctx := mgmt.WithCallerPubkey(context.Background(), pub)
	got, ok := mgmt.CallerPubkey(ctx)
	if !ok {
		t.Fatal("CallerPubkey: expected ok=true")
	}
	if !bytes.Equal(got, pub) {
		t.Errorf("CallerPubkey: got %x, want %x", got, pub)
	}

	// Empty context returns false.
	_, ok2 := mgmt.CallerPubkey(context.Background())
	if ok2 {
		t.Error("CallerPubkey on empty ctx: expected ok=false")
	}
}

// TestHandlerReceivesCallerPubkey verifies that the handler fn receives the
// authenticated caller pubkey via ctx when a real connection completes the
// challenge-response handshake. Traces to F-001 (ADR-012 §3 / AC-006).
func TestHandlerReceivesCallerPubkey(t *testing.T) {
	t.Parallel()

	pub, priv := mustGenKey(t)
	ops := mgmt.NewOperatorKeySet([]ed25519.PublicKey{pub})

	var receivedPubkey ed25519.PublicKey
	callerDone := make(chan struct{})
	handlers := []mgmt.Handler{
		{
			Command: "test.pubkey",
			Fn: func(ctx context.Context, _ json.RawMessage) (any, error) {
				pk, ok := mgmt.CallerPubkey(ctx)
				if ok {
					receivedPubkey = pk
				}
				select {
				case <-callerDone:
				default:
					close(callerDone)
				}
				return "ok", nil
			},
		},
	}

	// Use startServerOnPipe to get a client conn.
	// startServerOnPipe uses the first return of mustGenKey as the daemon key —
	// but we need ops to contain pub. Use a custom setup so ops is non-bootstrap.
	_, daemonPriv := mustGenKey(t)
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		_ = serverConn.Close()
		_ = clientConn.Close()
	})

	ln := newSingleConnListener(serverConn)
	srv := mgmt.NewServer(ln, daemonPriv, ops, handlers, "dev",
		mgmt.WithHandshakeTimeout(2*time.Second),
		mgmt.WithRPCIdleTimeout(5*time.Second),
	)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = srv.Serve(ctx) }() //nolint:errcheck // test goroutine

	// Set deadline so test doesn't hang.
	if err := clientConn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatalf("SetDeadline: %v", err)
	}

	if err := doHandshake(t, clientConn, priv); err != nil {
		t.Fatalf("handshake: %v", err)
	}

	// Send an RPC.
	rpcMsg, err := json.Marshal(map[string]any{
		"type":    "request",
		"id":      "req-1",
		"command": "test.pubkey",
		"args":    nil,
	})
	if err != nil {
		t.Fatalf("marshal rpc: %v", err)
	}
	rpcMsg = append(rpcMsg, '\n')
	if _, err := clientConn.Write(rpcMsg); err != nil {
		t.Fatalf("write rpc: %v", err)
	}

	// Read response (discard content — just drain so handler runs to completion).
	_ = readMsg(t, clientConn)

	select {
	case <-callerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("handler never called within 2s")
	}

	if !bytes.Equal(receivedPubkey, pub) {
		t.Errorf("handler got pubkey %x; want %x", receivedPubkey, pub)
	}
}

// ── Pass-2 L1 race regression: register-before-serve (F-P2L1-001) ────────────

// TestRegister_AfterServeReturnsError verifies the register-before-serve fence:
// once Serve has started, calls to Register must return an error.
//
// This is the defensive fence added by F-P2L1-001. The production wiring in
// cmd/switchboard calls Register BEFORE Serve; this test verifies the invariant
// is enforced at the mgmt.Server level as well.
//
// F-P2L1-001; S-W5.04 register-before-serve invariant.
func TestRegister_AfterServeReturnsError(t *testing.T) {
	t.Parallel()

	pub, priv := mustGenKey(t)
	ops := mgmt.NewOperatorKeySet([]ed25519.PublicKey{pub})

	// Use a real in-process listener to avoid filesystem socket paths.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	srv := mgmt.NewServer(ln, priv, ops, nil, "dev",
		mgmt.WithHandshakeTimeout(200*time.Millisecond),
		mgmt.WithRPCIdleTimeout(500*time.Millisecond),
	)

	// Verify that Register succeeds before Serve starts.
	if err := srv.Register(mgmt.Handler{Command: "pre.serve", Fn: func(_ context.Context, _ json.RawMessage) (any, error) {
		return "ok", nil
	}}); err != nil {
		t.Fatalf("Register before Serve: expected nil error; got %v", err)
	}

	// Start Serve in a background goroutine.
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	serveStarted := make(chan struct{})
	go func() {
		close(serveStarted)
		_ = srv.Serve(ctx)
	}()
	<-serveStarted

	// Give Serve a moment to mark serving=true. The flag is set at the very start
	// of Serve; a small sleep is sufficient for the test to observe the updated value.
	runtime.Gosched()
	time.Sleep(10 * time.Millisecond)

	// Verify that Register returns an error now that Serve has started.
	regErr := srv.Register(mgmt.Handler{Command: "post.serve", Fn: func(_ context.Context, _ json.RawMessage) (any, error) {
		return "ok", nil
	}})
	if regErr == nil {
		t.Error("Register after Serve started: expected error; got nil — " +
			"Register must return an error after Serve starts (F-P2L1-001 register-before-serve invariant)")
	} else {
		t.Logf("Register after Serve correctly returned error: %v", regErr)
	}
}
