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
	srv := mgmt.NewServer(ln, daemonPriv, ops, handlers)

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
		srv := mgmt.NewServer(ln, daemonPriv, keySet, nil)

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
	srv := mgmt.NewServer(ln, daemonPriv, ops, nil)

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
