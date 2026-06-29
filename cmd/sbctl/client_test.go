// Tests for cmd/sbctl: Authenticate(), loadEd25519Key(), tilde expansion,
// JSON envelope formatting, and connectAndRun error-return contract.
//
// Tests are named per BC-based convention (BC-2.07.002, BC-2.07.003, VP-067)
// for full traceability. All NEW/UPDATED tests MUST fail before implementation
// (Red Gate per BC-5.38.001).
//
// Package main (internal test file) so unexported names (loadEd25519Key,
// newSuccessEnvelope, newErrorEnvelope, Authenticate, connectAndRun) are
// directly accessible.
package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// encodeBase64URL encodes b using base64 URL encoding with no padding.
// Mirrors the encoding expected by Authenticate() per ADR-012.
func encodeBase64URL(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

// mockServerWithRead spawns a goroutine that sends the given JSON messages over
// the server half of a net.Pipe(), then reads exactly one reply from the client
// (the CHALLENGE_RESPONSE), and finally sends any remaining messages.
//
// The sync design (send challenge, read one reply, send auth result) matches the
// actual ADR-012 handshake flow and prevents the server goroutine from blocking
// when Authenticate() has not yet sent a reply (Red Gate).
//
// A deadline is applied to prevent permanent hangs in the Red Gate phase.
// The server goroutine is cleaned up via t.Cleanup (not defer-in-goroutine).
func mockServerWithRead(t *testing.T, challenge map[string]any, authResult map[string]any) net.Conn {
	t.Helper()
	server, client := net.Pipe()
	// Clean up the client connection; server goroutine closes itself.
	t.Cleanup(func() {
		_ = client.Close()
	})
	go func() {
		defer func() { _ = server.Close() }()
		// Deadline: unblock if client never reads/writes (Red Gate stub returns immediately).
		_ = server.SetDeadline(time.Now().Add(2 * time.Second))

		enc := json.NewEncoder(server)
		// Step 1: send CHALLENGE.
		if err := enc.Encode(challenge); err != nil {
			return
		}
		// Step 2: read CHALLENGE_RESPONSE (one newline-terminated JSON line).
		// This read will either succeed (real impl sent a response) or hit the
		// deadline (stub returned early without sending anything).
		buf := make([]byte, 8192)
		_, _ = server.Read(buf)
		// Step 3: send auth result (AUTH_OK or AUTH_FAIL).
		if authResult != nil {
			_ = enc.Encode(authResult)
		}
	}()
	return client
}

// freshNonce returns 32 crypto-random bytes and its base64url encoding.
func freshNonce(t *testing.T) ([]byte, string) {
	t.Helper()
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	return nonce, encodeBase64URL(nonce)
}

// wellFormedChallenge returns a challenge map with a valid 32-byte nonce
// signed by a freshly generated daemon key.
func wellFormedChallenge(t *testing.T) map[string]any {
	t.Helper()
	nonce, nonceB64 := freshNonce(t)
	_, daemonPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate daemon key: %v", err)
	}
	sig := ed25519.Sign(daemonPriv, nonce)
	return map[string]any{
		"type":       "challenge",
		"nonce":      nonceB64,
		"daemon_sig": encodeBase64URL(sig),
	}
}

// assertProtocolError checks that err is non-nil and is NOT the stub sentinel
// "not implemented". A "not implemented" error means the function returned
// before attempting any protocol exchange — that is a Red Gate FAIL.
//
// The real implementation must return a protocol-specific error (e.g.,
// "unexpected EOF", "auth_fail", "invalid nonce", "unexpected message type").
func assertProtocolError(t *testing.T, err error, subCase string) {
	t.Helper()
	if err == nil {
		t.Errorf("VP-067 violated: Authenticate returned nil on failure sub-case %q — expected non-nil error", subCase)
		return
	}
	if strings.Contains(err.Error(), "not implemented") {
		t.Errorf("Red Gate fail for sub-case %q: Authenticate returned stub error %q instead of a real protocol error — implementation is required", subCase, err.Error())
	}
}

// TestAuthenticate_FailClosed_VP067 verifies VP-067:
// Authenticate() returns nil ONLY on AUTH_OK; every other outcome returns
// a protocol-specific non-nil error. Each sub-case exercises exactly one
// failure mode from the VP-067 enumeration, plus the happy-path.
//
// Transport: net.Pipe (no real sockets).
// BC: BC-2.07.002 PC-2; VP-067; ARCH-12 §Authenticate() FAIL-CLOSED Contract.
// AC: AC-002 (ctx-first signature, deadline-expiry sub-case (i)).
//
// RED GATE: this test will fail to compile because the current Authenticate
// signature is Authenticate(net.Conn, ed25519.PrivateKey) error — missing the
// leading context.Context parameter. That compile error IS the Red Gate.
// After the signature is updated, sub-cases (a)–(h) must fail until the
// implementation is complete; sub-case (i) verifies ctx deadline expiry.
func TestAuthenticate_FailClosed_VP067(t *testing.T) {
	t.Parallel()

	_, opPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate operator key: %v", err)
	}

	t.Run("VP067_a_connection_error_on_challenge_read", func(t *testing.T) {
		t.Parallel()
		// Server closes immediately — read of CHALLENGE returns EOF.
		server, client := net.Pipe()
		t.Cleanup(func() { _ = client.Close() })
		go func() { _ = server.Close() }()

		err := Authenticate(context.Background(), client, opPriv)
		assertProtocolError(t, err, "VP067_a_connection_error_on_challenge_read")
	})

	t.Run("VP067_b_malformed_challenge_json_decode_error", func(t *testing.T) {
		t.Parallel()
		// Server sends garbage, not JSON.
		server, client := net.Pipe()
		t.Cleanup(func() { _ = client.Close() })
		go func() {
			defer func() { _ = server.Close() }()
			_, _ = server.Write([]byte("this is not json\n"))
			// drain any response the client might send
			buf := make([]byte, 1024)
			_ = server.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
			_, _ = server.Read(buf)
		}()

		err := Authenticate(context.Background(), client, opPriv)
		assertProtocolError(t, err, "VP067_b_malformed_challenge_json_decode_error")
	})

	t.Run("VP067_b_malformed_challenge_missing_nonce", func(t *testing.T) {
		t.Parallel()
		// Challenge has no nonce field — must be rejected per ARCH-12 step 2.
		conn := mockServerWithRead(t,
			map[string]any{"type": "challenge"}, // nonce absent
			nil,                                 // no auth result (client should error before this)
		)
		err := Authenticate(context.Background(), conn, opPriv)
		assertProtocolError(t, err, "VP067_b_malformed_challenge_missing_nonce")
	})

	t.Run("VP067_b_malformed_challenge_nonce_not_base64url", func(t *testing.T) {
		t.Parallel()
		conn := mockServerWithRead(t,
			map[string]any{
				"type":       "challenge",
				"nonce":      "!!!not-base64url!!!",
				"daemon_sig": encodeBase64URL(make([]byte, 64)),
			},
			nil,
		)
		err := Authenticate(context.Background(), conn, opPriv)
		assertProtocolError(t, err, "VP067_b_malformed_challenge_nonce_not_base64url")
	})

	t.Run("VP067_b_malformed_challenge_nonce_wrong_length", func(t *testing.T) {
		t.Parallel()
		shortNonce := make([]byte, 16) // 16 bytes, not 32 — must be rejected
		_, _ = rand.Read(shortNonce)
		conn := mockServerWithRead(t,
			map[string]any{
				"type":       "challenge",
				"nonce":      encodeBase64URL(shortNonce),
				"daemon_sig": encodeBase64URL(make([]byte, 64)),
			},
			nil,
		)
		err := Authenticate(context.Background(), conn, opPriv)
		assertProtocolError(t, err, "VP067_b_malformed_challenge_nonce_wrong_length")
	})

	t.Run("VP067_e_auth_fail_returns_error", func(t *testing.T) {
		t.Parallel()
		// Well-formed challenge, then AUTH_FAIL. Authenticate must return
		// a non-nil error that is NOT the stub sentinel.
		conn := mockServerWithRead(t,
			wellFormedChallenge(t),
			map[string]any{
				"type":    "auth_fail",
				"code":    "E-ADM-010",
				"message": "authentication failed",
			},
		)
		err := Authenticate(context.Background(), conn, opPriv)
		assertProtocolError(t, err, "VP067_e_auth_fail_returns_error")
		// The error must also surface E-ADM-010 so that connectAndRun can
		// distinguish auth failures from network failures.
		if err != nil && !strings.Contains(err.Error(), "E-ADM-010") && !strings.Contains(err.Error(), "auth") {
			t.Errorf("VP-067(e): expected error to reference auth failure (E-ADM-010 or 'auth'); got: %v", err)
		}
	})

	t.Run("VP067_f_wrong_response_type_returns_error", func(t *testing.T) {
		t.Parallel()
		// Server sends an unexpected message type after the challenge.
		conn := mockServerWithRead(t,
			wellFormedChallenge(t),
			map[string]any{"type": "unexpected_message_type"},
		)
		err := Authenticate(context.Background(), conn, opPriv)
		assertProtocolError(t, err, "VP067_f_wrong_response_type_returns_error")
	})

	t.Run("VP067_g_truncated_stream_after_challenge", func(t *testing.T) {
		t.Parallel()
		// Server closes connection after CHALLENGE — EOF before AUTH_OK/AUTH_FAIL.
		// Ruling W: hoist wellFormedChallenge(t) into test goroutine before go func()
		// so t.Fatalf (called inside wellFormedChallenge) fires in the test goroutine,
		// not a spawned goroutine where runtime.Goexit would leave the test indeterminate.
		challenge := wellFormedChallenge(t)
		server, client := net.Pipe()
		t.Cleanup(func() { _ = client.Close() })
		go func() {
			defer func() { _ = server.Close() }()
			_ = server.SetDeadline(time.Now().Add(2 * time.Second))
			enc := json.NewEncoder(server)
			if err := enc.Encode(challenge); err != nil {
				return
			}
			// Read CHALLENGE_RESPONSE (or timeout) then close without sending auth result.
			buf := make([]byte, 8192)
			_, _ = server.Read(buf)
			// close immediately — truncated stream
		}()

		err := Authenticate(context.Background(), client, opPriv)
		assertProtocolError(t, err, "VP067_g_truncated_stream_after_challenge")
	})

	t.Run("VP067_h_oversized_auth_response_bounded_by_limit_reader", func(t *testing.T) {
		t.Parallel()
		// Server sends a > 64 KiB auth response. io.LimitReader must truncate it.
		// Authenticate must return a decode error, NOT succeed or hang.
		// This proves CWE-400 protection on the second json.Decoder (ADR-012 §6).
		//
		// Ruling W: hoist wellFormedChallenge(t) into test goroutine before go func()
		// so t.Fatalf fires in the test goroutine, not the spawned goroutine.
		challenge := wellFormedChallenge(t)
		server, client := net.Pipe()
		t.Cleanup(func() { _ = client.Close() })
		go func() {
			defer func() { _ = server.Close() }()
			_ = server.SetDeadline(time.Now().Add(5 * time.Second))
			enc := json.NewEncoder(server)
			if err := enc.Encode(challenge); err != nil {
				return
			}
			// Read CHALLENGE_RESPONSE first so the client's send doesn't block.
			buf := make([]byte, 8192)
			_, _ = server.Read(buf)
			// Write an oversized message: a JSON object with a >64KiB "daemon_version" field.
			prefix := []byte(`{"type":"auth_ok","daemon_version":"`)
			padding := bytes.Repeat([]byte("x"), (1<<16)+1024) // > maxMessageBytes
			suffix := []byte(`"}` + "\n")
			_, _ = server.Write(prefix)
			_, _ = server.Write(padding)
			_, _ = server.Write(suffix)
		}()

		err := Authenticate(context.Background(), client, opPriv)
		assertProtocolError(t, err, "VP067_h_oversized_auth_response_bounded_by_limit_reader")
	})

	// VP067_i_deadline_expiry verifies AC-002 (Ruling 2, VP-067):
	// Authenticate() derives its read deadline from the context. When the context
	// deadline expires before the server sends anything, Authenticate must return
	// a non-nil error and MUST NOT hang.
	//
	// RED GATE: current Authenticate() has no ctx parameter at all — will not
	// compile. After the signature is updated but before deadline logic is
	// implemented, Authenticate will block indefinitely → test times out
	// (behaviorally failing with the right reason: missing deadline logic).
	//
	// BC: AC-002 §"derives the read deadline from ctx"; ARCH-12 §step 1.
	t.Run("VP067_i_deadline_expiry_server_silent", func(t *testing.T) {
		t.Parallel()
		// Server never sends anything — Authenticate must time out via ctx.
		server, client := net.Pipe()
		t.Cleanup(func() {
			_ = client.Close()
			_ = server.Close()
		})
		// Silent server: accept connection but never write.
		go func() {
			// Keep the connection alive for longer than the ctx deadline so
			// the timeout is the limiting factor, not a write error.
			_ = server.SetDeadline(time.Now().Add(2 * time.Second))
			buf := make([]byte, 64)
			_, _ = server.Read(buf) // drain anything the client might send
		}()

		// Context with a tight deadline — Authenticate must return within ~2x of this.
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		start := time.Now()
		err := Authenticate(ctx, client, opPriv)
		elapsed := time.Since(start)

		if err == nil {
			t.Error("VP067_i: Authenticate returned nil with context deadline expired — expected non-nil error")
		}
		// Must return within a reasonable bound (not hang indefinitely).
		// 500ms gives generous headroom while ruling out a hang.
		if elapsed > 500*time.Millisecond {
			t.Errorf("VP067_i: Authenticate hung for %v (> 500ms) — context deadline not applied to conn.SetReadDeadline", elapsed)
		}
	})

	t.Run("VP067_happy_path_auth_ok_returns_nil", func(t *testing.T) {
		t.Parallel()
		// Server sends AUTH_OK. Authenticate() MUST return nil.
		conn := mockServerWithRead(t,
			wellFormedChallenge(t),
			map[string]any{
				"type":           "auth_ok",
				"daemon_version": "0.1.0",
			},
		)
		err := Authenticate(context.Background(), conn, opPriv)
		if err != nil {
			t.Errorf("VP-067 violated: Authenticate returned non-nil error on AUTH_OK: %v", err)
		}
	})
}

// TestAuthenticate_PrivKeyNeverTransmitted verifies DI-002:
// the operator's private key seed bytes never appear in the CHALLENGE_RESPONSE
// message sent to the server. Only the 32-byte public key and the nonce
// signature go over the wire.
//
// BC: BC-2.07.002 Inv-2; ARCH-12 §ADR-012 step 3 ("private key NEVER leaves the client").
//
// RED GATE: this test will fail to compile because the current Authenticate
// signature lacks the leading context.Context parameter.
func TestAuthenticate_PrivKeyNeverTransmitted(t *testing.T) {
	t.Parallel()

	_, opPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate operator key: %v", err)
	}

	// Extract the seed (first 32 bytes) — this is the secret portion that must
	// never appear in any outbound message.
	privKeyBytes := []byte(opPriv)
	seed := privKeyBytes[:32]
	seedB64URL := base64.RawURLEncoding.EncodeToString(seed)
	seedB64Std := base64.StdEncoding.EncodeToString(seed)

	server, client := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})

	// Ruling W: hoist freshNonce(t) and daemonPriv generation into the test goroutine
	// before spawning go func(), so t.Fatalf fires in the test goroutine.
	nonce, nonceB64 := freshNonce(t)
	_, daemonPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate daemon key: %v", err)
	}
	sig := ed25519.Sign(daemonPriv, nonce)

	// doneCh signals when the server goroutine has finished capturing bytes.
	doneCh := make(chan []byte, 1)

	go func() {
		// Apply a deadline so we don't block permanently if the stub never reads.
		_ = server.SetDeadline(time.Now().Add(1 * time.Second))
		defer func() { _ = server.Close() }()

		enc := json.NewEncoder(server)
		_ = enc.Encode(map[string]any{
			"type":       "challenge",
			"nonce":      nonceB64,
			"daemon_sig": encodeBase64URL(sig),
		})

		// Read whatever the client sends (the CHALLENGE_RESPONSE, or nothing from the stub).
		buf := make([]byte, 4096)
		n, _ := server.Read(buf)
		captured := make([]byte, n)
		copy(captured, buf[:n])

		// Send AUTH_OK to let Authenticate() complete (if it's real).
		_ = enc.Encode(map[string]any{"type": "auth_ok", "daemon_version": "0.1.0"})

		doneCh <- captured
	}()

	_ = Authenticate(context.Background(), client, opPriv)
	captured := <-doneCh

	wire := string(captured)

	// The private key seed must NOT appear in any standard encoding on the wire.
	if strings.Contains(wire, seedB64URL) {
		t.Errorf("DI-002 violated: private key seed appears as base64url in CHALLENGE_RESPONSE: %s", wire)
	}
	if strings.Contains(wire, seedB64Std) {
		t.Errorf("DI-002 violated: private key seed appears as base64std in CHALLENGE_RESPONSE: %s", wire)
	}
}

// TestSbctl_KeyLoading_Ed25519 verifies AC-001:
// A well-formed OpenSSH Ed25519 private key file loads successfully and
// produces an ed25519.PrivateKey with length 64. A file larger than 64 KiB
// is rejected with a non-nil error (ADR-012 §6 CWE-400 protection).
//
// BC: BC-2.07.002 PC-2 (key loading); AC-001; ARCH-12 §OpenSSH Key Loading.
func TestSbctl_KeyLoading_Ed25519(t *testing.T) {
	t.Parallel()

	// Locate the pre-generated testdata fixture.
	fixtureKey := filepath.Join("testdata", "test_ed25519_key")

	t.Run("well_formed_ed25519_key_loads_to_64_bytes", func(t *testing.T) {
		t.Parallel()
		privKey, err := loadEd25519Key(fixtureKey, os.UserHomeDir)
		if err != nil {
			t.Fatalf("loadEd25519Key(%q) returned unexpected error: %v", fixtureKey, err)
		}
		if len(privKey) != ed25519.PrivateKeySize {
			t.Errorf("expected private key length %d (ed25519.PrivateKeySize), got %d", ed25519.PrivateKeySize, len(privKey))
		}
	})

	t.Run("file_larger_than_64KiB_is_rejected", func(t *testing.T) {
		t.Parallel()
		// Write a file larger than maxMessageBytes (64 KiB).
		tmp := t.TempDir()
		oversized := filepath.Join(tmp, "big.key")
		// 65 KiB + 1 byte — exceeds the io.LimitReader bound.
		garbage := bytes.Repeat([]byte{0}, (1<<16)+1)
		if err := os.WriteFile(oversized, garbage, 0o600); err != nil {
			t.Fatalf("write oversized file: %v", err)
		}
		_, err := loadEd25519Key(oversized, os.UserHomeDir)
		if err == nil {
			t.Fatal("loadEd25519Key accepted a file > 64 KiB — io.LimitReader boundary not enforced (CWE-400)")
		}
	})

	t.Run("nonexistent_file_returns_error", func(t *testing.T) {
		t.Parallel()
		_, err := loadEd25519Key("/nonexistent/path/to/key.pem", os.UserHomeDir)
		if err == nil {
			t.Fatal("loadEd25519Key returned nil error for a nonexistent path")
		}
	})
}

// TestSbctl_KeyLoadFailure_ExitsOneWithECFG010 verifies AC-003 (BC-2.07.003 EC-005,
// Ruling 5): when the --key file is missing, oversized, malformed, or wrong key
// type, sbctl emits E-CFG-010 "key load failed: <path>: <reason>" to stderr,
// exits 1, and makes NO connection attempt.
//
// The "no connection attempt" assertion is enforced by targeting a non-listening
// address: if the key-load error fires before dial, E-CFG-010 appears without
// E-NET-001; if the implementation incorrectly dials first, the subprocess will
// also emit E-NET-001 (or fail with the wrong code), failing the assertion.
//
// BC: BC-2.07.003 EC-005; AC-003 §key load failure.
//
// RED GATE: current connectAndRun emits E-NET-001 for key load failures (bug).
// Tests fail because stderr contains "E-NET-001" not "E-CFG-010".
func TestSbctl_KeyLoadFailure_ExitsOneWithECFG010(t *testing.T) {
	t.Parallel()

	// Non-listening address — if a dial is attempted, it will fail immediately
	// with E-NET-001, which the test uses to detect the ordering bug.
	const nonListeningTarget = "127.0.0.1:19996"

	type tc struct {
		name       string
		setupKey   func(t *testing.T) string // returns path to supply as --key
		wantInPath string                    // sub-string expected in error path portion
	}

	cases := []tc{
		{
			name: "missing_key_file",
			setupKey: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "nonexistent_key")
			},
			wantInPath: "nonexistent_key",
		},
		{
			name: "oversized_key_file",
			setupKey: func(t *testing.T) string {
				t.Helper()
				tmp := t.TempDir()
				p := filepath.Join(tmp, "oversized.key")
				garbage := bytes.Repeat([]byte{0x41}, (1<<16)+1)
				if err := os.WriteFile(p, garbage, 0o600); err != nil {
					t.Fatalf("write oversized: %v", err)
				}
				return p
			},
			wantInPath: "oversized.key",
		},
		{
			name: "malformed_pem_key_file",
			setupKey: func(t *testing.T) string {
				t.Helper()
				tmp := t.TempDir()
				p := filepath.Join(tmp, "malformed.key")
				if err := os.WriteFile(p, []byte("not a pem file\n"), 0o600); err != nil {
					t.Fatalf("write malformed: %v", err)
				}
				return p
			},
			wantInPath: "malformed.key",
		},
		{
			name: "wrong_key_type_rsa",
			setupKey: func(t *testing.T) string {
				t.Helper()
				// Write a PEM block with wrong header type to simulate a non-Ed25519 key.
				// Using OPENSSH PRIVATE KEY header with garbage body to trigger parse error.
				tmp := t.TempDir()
				p := filepath.Join(tmp, "rsa_style.key")
				// This PEM has the right type marker but wrong body — ParseRawPrivateKey will
				// reject it as malformed (effectively "not Ed25519" from sbctl's perspective).
				content := "-----BEGIN OPENSSH PRIVATE KEY-----\nbm90YW55dGhpbmc=\n-----END OPENSSH PRIVATE KEY-----\n"
				if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
					t.Fatalf("write wrong-type key: %v", err)
				}
				return p
			},
			wantInPath: "rsa_style.key",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			keyPath := tc.setupKey(t)

			exitCode, stdout, stderr := runSubprocess(t, "KeyLoadFailure",
				"SBCTL_TEST_TARGET="+nonListeningTarget,
				"SBCTL_TEST_KEY="+keyPath,
				"SBCTL_TEST_TIMEOUT=200ms",
			)

			if exitCode != 1 {
				t.Errorf("AC-003 violated: expected exit code 1, got %d\nstderr: %s", exitCode, stderr)
			}
			// Must emit E-CFG-010, not E-NET-001 (ordering: key load before dial).
			if !strings.Contains(stderr, "E-CFG-010") {
				t.Errorf("AC-003 violated: expected 'E-CFG-010' in stderr; got: %q", stderr)
			}
			// Path portion of the error message must reference the key file.
			if !strings.Contains(stderr, tc.wantInPath) {
				t.Errorf("AC-003 violated: expected path %q in stderr; got: %q", tc.wantInPath, stderr)
			}
			// Must NOT emit E-NET-001 (key load fails before any dial attempt).
			if strings.Contains(stderr, "E-NET-001") {
				t.Errorf("AC-003 violated (ordering): 'E-NET-001' must not appear when key load fails; got: %q", stderr)
			}
			// No stdout on failure (BC-2.07.003 PC-3).
			if stdout != "" {
				t.Errorf("AC-003 violated: expected empty stdout; got: %q", stdout)
			}
		})
	}
}

// TestSbctl_RPCDispatchFailure_ExitsOneWithERPC001 verifies AC-004 (BC-2.07.003
// EC-006, Ruling 5): when authentication succeeds (AUTH_OK) but the subsequent
// RPC dispatch fails (server returns "ok":false), sbctl emits E-RPC-001
// "rpc failed: <command>: <reason>" to stderr, exits 1, and produces no stdout.
//
// BC: BC-2.07.003 EC-006; BC-2.07.003 Invariant 4 (E-RPC-001 distinct from
// E-NET-001 and E-CFG-010); AC-004.
//
// RED GATE: current dispatch() returns "not implemented" and connectAndRun
// maps that to E-NET-001 (bug). Tests fail because stderr contains "E-NET-001"
// not "E-RPC-001", and the server never completes the handshake before the
// subprocess fails.
func TestSbctl_RPCDispatchFailure_ExitsOneWithERPC001(t *testing.T) {
	t.Parallel()

	// Start a mock server that completes AUTH_OK then returns RPC failure.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	target := ln.Addr().String()
	keyPath := testdataKeyPath(t)

	// serverDoneCh carries a nil or error from the server goroutine.
	serverDoneCh := make(chan error, 1)
	go func() {
		// Accept deadline: subprocess may fail before connecting in Red Gate phase.
		if tcpLn, ok := ln.(*net.TCPListener); ok {
			_ = tcpLn.SetDeadline(time.Now().Add(3 * time.Second))
		}
		conn, err := ln.Accept()
		if err != nil {
			serverDoneCh <- fmt.Errorf("accept: %w", err)
			return
		}
		defer func() { _ = conn.Close() }()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

		// Step 1: Send a valid CHALLENGE with a 32-byte nonce.
		nonce := make([]byte, 32)
		_, _ = rand.Read(nonce)
		challenge := fmt.Sprintf(
			`{"type":"challenge","nonce":"%s","daemon_sig":"%s"}`+"\n",
			base64.RawURLEncoding.EncodeToString(nonce),
			base64.RawURLEncoding.EncodeToString(make([]byte, 64)),
		)
		if _, err := fmt.Fprint(conn, challenge); err != nil {
			serverDoneCh <- fmt.Errorf("write challenge: %w", err)
			return
		}

		// Step 2: Read CHALLENGE_RESPONSE.
		buf := make([]byte, 8192)
		if _, err := conn.Read(buf); err != nil {
			serverDoneCh <- fmt.Errorf("read challenge_response: %w", err)
			return
		}

		// Step 3: Send AUTH_OK.
		const authOK = `{"type":"auth_ok","daemon_version":"0.1.0"}` + "\n"
		if _, err := fmt.Fprint(conn, authOK); err != nil {
			serverDoneCh <- fmt.Errorf("write auth_ok: %w", err)
			return
		}

		// Step 4: Read RPC request (if any).
		rpcBuf := make([]byte, 8192)
		_, _ = conn.Read(rpcBuf)

		// Step 5: Send RPC failure response (ok:false).
		// Ruling U: type MUST be "response" (not "rpc_response") so dispatch() reaches
		// the ok:false path rather than being rejected for the wrong reason (type mismatch).
		// With Ruling U implemented, "rpc_response" would produce E-RPC-001 for the wrong
		// reason (type mismatch), masking the intent of this test (ok:false path).
		const rpcFail = `{"type":"response","id":"1","ok":false,"error":{"code":"E-RPC-001","message":"rpc failed: router.status: unknown command"},"data":null}` + "\n"
		if _, err := fmt.Fprint(conn, rpcFail); err != nil {
			serverDoneCh <- fmt.Errorf("write rpc_fail: %w", err)
			return
		}

		serverDoneCh <- nil
	}()

	exitCode, stdout, stderr := runSubprocess(t, "RPCDispatchFailure",
		"SBCTL_TEST_TARGET="+target,
		"SBCTL_TEST_KEY="+keyPath,
		"SBCTL_TEST_TIMEOUT=3s",
	)

	// Log server result for debugging; don't fail the test on server errors alone.
	if serverErr := <-serverDoneCh; serverErr != nil {
		t.Logf("mock server error (subprocess may have failed before auth in Red Gate): %v", serverErr)
	}

	if exitCode != 1 {
		t.Errorf("AC-004 violated: expected exit code 1, got %d\nstderr: %s", exitCode, stderr)
	}
	if !strings.Contains(stderr, "E-RPC-001") {
		t.Errorf("AC-004 violated: expected 'E-RPC-001' in stderr; got: %q", stderr)
	}
	// E-NET-001 must NOT appear — dispatch failure is distinct from unreachable.
	if strings.Contains(stderr, "E-NET-001") {
		t.Errorf("AC-004 violated (Invariant 4): 'E-NET-001' must not appear for RPC dispatch failure; got: %q", stderr)
	}
	// No stdout on failure (BC-2.07.003 PC-3).
	if stdout != "" {
		t.Errorf("AC-004 violated: expected empty stdout; got: %q", stdout)
	}
}

// TestSbctl_TildeExpansion_DefaultKey verifies AC-008 (BC-2.07.003 EC-007 +
// Precondition 3): loadEd25519Key expands leading ~ via the homeDir parameter
// before opening the file. Four sub-cases are required:
//
//  1. Happy path: ~ expands to a real dir with a valid key.
//  2. Sub-case (a): homeDir() returns error → E-CFG-010 with ORIGINAL path.
//  3. Sub-case (b): expansion ok but file missing → E-CFG-010 with EXPANDED path.
//  4. ~username literal: treated as literal (homeDir not called).
//
// BC: BC-2.07.003 EC-007 + Precondition 3; AC-008.
//
// RACE-SAFE DESIGN: homeDir is injected as a per-call parameter to
// loadEd25519Key — no package-global is mutated. All subtests can run in
// parallel without data races under -race. (Option b from the race-fix
// analysis: parameter injection instead of global mutation.)
func TestSbctl_TildeExpansion_DefaultKey(t *testing.T) {
	t.Parallel()

	// setupKeyFile writes an OpenSSH Ed25519 key fixture to dir/.ssh/id_ed25519
	// and returns the full path.
	setupKeyFile := func(t *testing.T, dir string) string {
		t.Helper()
		sshDir := filepath.Join(dir, ".ssh")
		if err := os.MkdirAll(sshDir, 0o700); err != nil {
			t.Fatalf("mkdir .ssh: %v", err)
		}
		fixtureBytes, err := os.ReadFile(filepath.Join("testdata", "test_ed25519_key"))
		if err != nil {
			t.Fatalf("read testdata key: %v", err)
		}
		keyPath := filepath.Join(sshDir, "id_ed25519")
		if err := os.WriteFile(keyPath, fixtureBytes, 0o600); err != nil {
			t.Fatalf("write key: %v", err)
		}
		return keyPath
	}

	t.Run("happy_path_tilde_slash_expands_to_home", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		setupKeyFile(t, tmp)

		// Inject homeDir returning our temp dir — no global mutation.
		homeDir := func() (string, error) { return tmp, nil }

		privKey, err := loadEd25519Key("~/.ssh/id_ed25519", homeDir)
		if err != nil {
			t.Fatalf("AC-008 happy path: loadEd25519Key returned error: %v", err)
		}
		if len(privKey) != ed25519.PrivateKeySize {
			t.Errorf("AC-008 happy path: key length = %d, want %d", len(privKey), ed25519.PrivateKeySize)
		}
	})

	t.Run("sub_case_a_homedir_error_uses_original_path", func(t *testing.T) {
		// AC-008 sub-case (a): homeDir() returns error → E-CFG-010 with the
		// ORIGINAL unexpanded path in the message.
		t.Parallel()

		homeDir := func() (string, error) {
			return "", fmt.Errorf("home directory unavailable: no HOME environment")
		}

		const inputPath = "~/.ssh/id_ed25519"
		_, err := loadEd25519Key(inputPath, homeDir)
		if err == nil {
			t.Fatal("AC-008(a): expected error when homeDir fails, got nil")
		}
		// Error message must contain the ORIGINAL path (not an expanded form).
		if !strings.Contains(err.Error(), inputPath) {
			t.Errorf("AC-008(a): error message must contain original path %q; got: %v", inputPath, err)
		}
		// Must not contain an expanded path like "/home/..." or "/Users/...".
		if strings.Contains(err.Error(), "/home/") || strings.Contains(err.Error(), "/Users/") {
			t.Errorf("AC-008(a): error message must use original path, not expanded; got: %v", err)
		}
	})

	t.Run("sub_case_b_expansion_ok_but_file_missing_uses_expanded_path", func(t *testing.T) {
		// AC-008 sub-case (b): homeDir() succeeds but file doesn't exist →
		// E-CFG-010 with the EXPANDED path in the message.
		t.Parallel()
		tmp := t.TempDir()
		// Do NOT create the key file — the file is missing.

		homeDir := func() (string, error) { return tmp, nil }

		const inputPath = "~/.ssh/id_ed25519"
		expandedPath := filepath.Join(tmp, ".ssh", "id_ed25519")

		_, err := loadEd25519Key(inputPath, homeDir)
		if err == nil {
			t.Fatal("AC-008(b): expected error when expanded file is missing, got nil")
		}
		// Error message must contain the EXPANDED path (not the original ~-prefixed form).
		if !strings.Contains(err.Error(), expandedPath) {
			t.Errorf("AC-008(b): error message must contain expanded path %q; got: %v", expandedPath, err)
		}
	})

	t.Run("tilde_username_treated_as_literal", func(t *testing.T) {
		// AC-008 sub-case (~username): "~root/.ssh/id_ed25519" is treated as a
		// literal path (not expanded). homeDir must NOT be called.
		t.Parallel()

		called := false
		homeDir := func() (string, error) {
			called = true
			return "", fmt.Errorf("should not have been called for ~username path")
		}

		const literalPath = "~root/.ssh/id_ed25519"
		_, err := loadEd25519Key(literalPath, homeDir)
		if err == nil {
			t.Fatal("AC-008(~username): expected error for literal ~username path, got nil")
		}
		// homeDir must NOT have been called.
		if called {
			t.Error("AC-008(~username): homeDir was called for a ~username path — only ~/ should trigger expansion")
		}
		// Error message must reference the literal path.
		if !strings.Contains(err.Error(), literalPath) {
			t.Errorf("AC-008(~username): error message must contain literal path %q; got: %v", literalPath, err)
		}
	})
}

// TestSbctl_ConnectAndRun_ReturnsError verifies AC-009 (go.md "no os.Exit outside
// main()"): connectAndRun (or the equivalent dispatch entrypoint in client.go)
// must return an error — it must NOT call os.Exit. If os.Exit were called, the
// test process would immediately exit and the test suite would crash with a
// non-zero status and no FAIL output, making the bug self-evidently visible.
//
// The test calls connectAndRun in-process with a mock that returns an auth failure
// and asserts it returns a non-nil error (not nil, and not void).
//
// BC: AC-009; go.md rule "No log.Fatal / os.Exit outside main()"; Ruling 5.
//
// RED GATE: current connectAndRun has signature:
//
//	func connectAndRun(target, keyPath string, useJSON bool, timeout time.Duration, command string, cmdArgs any)
//
// This test calls a function expected to have signature:
//
//	func connectAndRun(ctx context.Context, target, keyPath string, useJSON bool, command string, cmdArgs any) error
//
// The call will fail to compile until the signature is updated (the ctx-first,
// error-return version). That compile failure IS the Red Gate.
func TestSbctl_ConnectAndRun_ReturnsError(t *testing.T) {
	t.Parallel()

	// Use a non-listening address so the connection attempt fails immediately.
	const target = "127.0.0.1:19995"
	keyPath := testdataKeyPath(t)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// connectAndRun must return an error. If it calls os.Exit instead, this
	// test process terminates immediately — the t.Errorf line is never reached
	// but the whole test binary exits non-zero with no FAIL output (observable
	// from the go test harness).
	err := connectAndRun(ctx, target, keyPath, false, "ping", nil)
	if err == nil {
		t.Error("AC-009 violated: connectAndRun returned nil error on connection failure — expected non-nil error")
	}
}

// TestDispatch_EmitsCorrectWireType verifies AC-010 (BC-2.07.002 PC-3 / ADR-012
// §3 step 6 / Ruling M):
// dispatch() MUST emit `"type":"request"` in the outbound RPC envelope.
// The server (internal/mgmt) expects exactly `"type":"request"` and closes the
// connection silently on any other value after authentication — a wire-type
// mismatch causes every RPC to fail with a connection-closed E-RPC-001 decode
// error on the client side.
//
// RED because: current client.go sets Type: "rpc_request" (line ~254); the
// spec requires Type: "request" per ADR-012 §3 step 6. This test asserts the
// spec wire-type, so it MUST fail against the current buggy literal until the
// one-line fix is applied.
//
// Test strategy: drive dispatch() over a net.Pipe; the mock server side reads
// the raw bytes the client writes and checks the type field, then returns a
// canonical `"type":"response"` envelope; dispatch() must decode that as a
// successful call (nil error, non-nil data).
//
// BC: BC-2.07.002 PC-3; AC-010; ADR-012 §3 step 6; Ruling M.
func TestDispatch_EmitsCorrectWireType(t *testing.T) {
	t.Parallel()

	server, client := net.Pipe()
	t.Cleanup(func() { _ = client.Close() })

	// rawRequestCh receives the raw JSON bytes the client sends for the RPC request.
	rawRequestCh := make(chan []byte, 1)

	go func() {
		defer func() { _ = server.Close() }()
		_ = server.SetDeadline(time.Now().Add(2 * time.Second))

		// Read the entire RPC request line written by dispatch().
		buf := make([]byte, 4096)
		n, err := server.Read(buf)
		if err != nil || n == 0 {
			rawRequestCh <- nil
			return
		}
		raw := make([]byte, n)
		copy(raw, buf[:n])
		rawRequestCh <- raw

		// Respond with the canonical server wire-type per ADR-012 §3 step 6.
		// "type":"response" is the ONLY accepted server wire-type (Ruling M).
		const response = `{"type":"response","id":"1","ok":true,"data":{}}` + "\n"
		_, _ = server.Write([]byte(response))
	}()

	data, err := dispatch(context.Background(), client, "ping", nil)

	// Retrieve what the mock server captured before asserting.
	raw := <-rawRequestCh

	// AC-010 / Ruling M primary assertion: the type field MUST be "request".
	// RED because: current code emits "rpc_request", spec requires "request".
	if raw == nil {
		t.Fatal("AC-010: mock server received no bytes from dispatch() — dispatch may have failed before writing")
	}
	rawStr := string(raw)
	if !strings.Contains(rawStr, `"type":"request"`) {
		t.Errorf("AC-010 violated (ADR-012 §3 step 6 / Ruling M): dispatch() emitted wrong wire-type.\n  raw bytes: %s\n  want:       contains %q\n  RED because: client emits \"rpc_request\", spec requires \"request\"",
			rawStr, `"type":"request"`)
	}
	if strings.Contains(rawStr, `"type":"rpc_request"`) {
		t.Errorf("AC-010 violated: dispatch() emitted forbidden wire-type \"rpc_request\" (must be \"request\")")
	}

	// Secondary assertion: dispatch() must decode the canonical "type":"response"
	// server response as a successful call — nil error and non-nil data.
	if err != nil {
		t.Errorf("AC-010: dispatch() returned unexpected error with canonical \"type\":\"response\" server response: %v", err)
	}
	if data == nil {
		t.Error("AC-010: dispatch() returned nil data with a successful server response")
	}
}

// TestDispatch_AcceptsResponseType verifies AC-010 (BC-2.07.002 PC-3 / Ruling M):
// dispatch() MUST accept a server response envelope carrying `"type":"response"`
// as a successful RPC reply. The canonical server wire-type is `"response"` per
// ADR-012 §3 step 6; the client must not reject or hang on this type.
//
// This test locks in the acceptance contract so that a future refactor cannot
// accidentally break the response-type handling.
//
// BC: BC-2.07.002 PC-3; AC-010; ADR-012 §3 step 6; Ruling M.
func TestDispatch_AcceptsResponseType(t *testing.T) {
	t.Parallel()

	server, client := net.Pipe()
	t.Cleanup(func() { _ = client.Close() })

	go func() {
		defer func() { _ = server.Close() }()
		_ = server.SetDeadline(time.Now().Add(2 * time.Second))

		// Drain the RPC request so dispatch() does not block on its Encode call.
		buf := make([]byte, 4096)
		_, _ = server.Read(buf)

		// Reply with the canonical "type":"response" wire-type (ADR-012 §3 step 6).
		const response = `{"type":"response","id":"1","ok":true,"data":{"status":"ok"}}` + "\n"
		_, _ = server.Write([]byte(response))
	}()

	data, err := dispatch(context.Background(), client, "router.status", nil)
	// dispatch() must decode "type":"response" as success (nil error, non-nil data).
	if err != nil {
		t.Errorf("AC-010 violated: dispatch() rejected canonical \"type\":\"response\" server response: %v", err)
	}
	if data == nil {
		t.Error("AC-010: dispatch() returned nil data for a successful \"type\":\"response\" server reply")
	}
}

// TestSbctl_JSONEnvelopeFormat verifies AC-006:
// newSuccessEnvelope produces {"ok":true,"error":null,"data":{...}} and
// newErrorEnvelope produces {"ok":false,"error":{"code":"...","message":"..."},"data":null}.
//
// Both shapes must decode correctly per interface-definitions.md §JSON Output Schema.
// BC: BC-2.07.002 PC-3; AC-006.
func TestSbctl_JSONEnvelopeFormat(t *testing.T) {
	t.Parallel()

	t.Run("success_envelope_ok_true_error_null_data_present", func(t *testing.T) {
		t.Parallel()
		rawData := json.RawMessage(`{"key":"value"}`)
		env := newSuccessEnvelope(rawData)

		out, err := json.Marshal(env)
		if err != nil {
			t.Fatalf("marshal success envelope: %v", err)
		}

		var decoded map[string]json.RawMessage
		if err := json.Unmarshal(out, &decoded); err != nil {
			t.Fatalf("unmarshal success envelope: %v", err)
		}

		// "ok" must be present and be true.
		if _, ok := decoded["ok"]; !ok {
			t.Error("success envelope missing 'ok' field")
		}
		var okVal bool
		if err := json.Unmarshal(decoded["ok"], &okVal); err != nil || !okVal {
			t.Errorf("success envelope 'ok' must be true; got raw: %s", decoded["ok"])
		}

		// "error" must be present and null.
		if _, ok := decoded["error"]; !ok {
			t.Error("success envelope missing 'error' field")
		}
		if string(decoded["error"]) != "null" {
			t.Errorf("success envelope 'error' must be null; got: %s", decoded["error"])
		}

		// "data" must be present and match the input.
		if _, ok := decoded["data"]; !ok {
			t.Error("success envelope missing 'data' field")
		}
		if string(decoded["data"]) != `{"key":"value"}` {
			t.Errorf("success envelope 'data' mismatch; got: %s", decoded["data"])
		}
	})

	t.Run("error_envelope_ok_false_error_present_data_null", func(t *testing.T) {
		t.Parallel()
		env := newErrorEnvelope("E-NET-001", "daemon unreachable: localhost:9090: connection refused")

		out, err := json.Marshal(env)
		if err != nil {
			t.Fatalf("marshal error envelope: %v", err)
		}

		var decoded map[string]json.RawMessage
		if err := json.Unmarshal(out, &decoded); err != nil {
			t.Fatalf("unmarshal error envelope: %v", err)
		}

		// "ok" must be false.
		var okVal bool
		if err := json.Unmarshal(decoded["ok"], &okVal); err != nil || okVal {
			t.Errorf("error envelope 'ok' must be false; got raw: %s", decoded["ok"])
		}

		// "error" must be a non-null object with "code" and "message".
		if string(decoded["error"]) == "null" {
			t.Fatal("error envelope 'error' must not be null")
		}
		var errObj map[string]json.RawMessage
		if err := json.Unmarshal(decoded["error"], &errObj); err != nil {
			t.Fatalf("unmarshal error.error object: %v", err)
		}
		var code string
		if err := json.Unmarshal(errObj["code"], &code); err != nil || code != "E-NET-001" {
			t.Errorf("error.code mismatch: want E-NET-001, got %s (err: %v)", code, err)
		}

		// "data" must be null.
		if string(decoded["data"]) != "null" {
			t.Errorf("error envelope 'data' must be null; got: %s", decoded["data"])
		}
	})
}

// TestDispatch_RejectsWrongResponseType verifies AC-010 (BC-2.07.002 PC-3, Ruling U /
// ARCH-12 v1.5): dispatch() MUST validate resp.Type == "response" after decoding. A
// server reply carrying "type":"rpc_response" (even with "ok":true) MUST be rejected
// with a non-nil E-RPC-001 error containing "unexpected response type". A wrong-type
// response MUST NOT be silently accepted based on the ok flag alone.
//
// RED because: current dispatch() never checks resp.Type — it accepts any response
// where resp.OK is true, regardless of the type field. With Ruling U implemented,
// the type check takes precedence over the ok check.
//
// BC: BC-2.07.002 PC-3; AC-010; Ruling U (ARCH-12 v1.5).
func TestDispatch_RejectsWrongResponseType(t *testing.T) {
	t.Parallel()

	t.Run("rpc_response_type_with_ok_true_is_rejected", func(t *testing.T) {
		t.Parallel()
		// Mock server replies with forbidden "type":"rpc_response" but "ok":true.
		// dispatch() MUST return a non-nil error (type mismatch, E-RPC-001).
		// RED: current dispatch() accepts this (ignores resp.Type, checks resp.OK only).
		server, client := net.Pipe()
		t.Cleanup(func() { _ = client.Close() })

		go func() {
			defer func() { _ = server.Close() }()
			_ = server.SetDeadline(time.Now().Add(2 * time.Second))
			// Drain the request so the client's Encode does not block.
			buf := make([]byte, 4096)
			_, _ = server.Read(buf)
			// Respond with the FORBIDDEN type "rpc_response" + ok:true.
			const forbidden = `{"type":"rpc_response","id":"1","ok":true,"data":{}}` + "\n"
			_, _ = server.Write([]byte(forbidden))
		}()

		_, err := dispatch(context.Background(), client, "router.status", nil)
		if err == nil {
			t.Errorf("Ruling U violated: dispatch() returned nil error on 'rpc_response' type reply — expected non-nil E-RPC-001 (type mismatch); current dispatch() ignores resp.Type and silently accepts on ok:true (RED Gate)")
		}
		if err != nil && !strings.Contains(err.Error(), "unexpected response type") {
			t.Errorf("Ruling U: error must mention 'unexpected response type'; got: %v", err)
		}
	})

	t.Run("ok_false_with_correct_type_is_erpc001_not_type_mismatch", func(t *testing.T) {
		t.Parallel()
		// Verify that the type-check path is distinct from the ok:false path.
		// A correct-type response with ok:false must produce E-RPC-001 from the ok-false
		// path (not "unexpected response type"). This locks in the guard ordering:
		// type check first, then ok check.
		server, client := net.Pipe()
		t.Cleanup(func() { _ = client.Close() })

		go func() {
			defer func() { _ = server.Close() }()
			_ = server.SetDeadline(time.Now().Add(2 * time.Second))
			buf := make([]byte, 4096)
			_, _ = server.Read(buf)
			// Correct type, ok:false — should reach the ok-false error path.
			const rpcFail = `{"type":"response","id":"1","ok":false,"error":{"code":"E-RPC-001","message":"rpc failed: router.status: unknown command"},"data":null}` + "\n"
			_, _ = server.Write([]byte(rpcFail))
		}()

		_, err := dispatch(context.Background(), client, "router.status", nil)
		if err == nil {
			t.Error("dispatch() must return non-nil error on ok:false response")
		}
		// The error must NOT say "unexpected response type" — the type was correct.
		if err != nil && strings.Contains(err.Error(), "unexpected response type") {
			t.Errorf("Ruling U guard-ordering violated: ok:false with correct type emitted 'unexpected response type' instead of reaching the ok-false error path; got: %v", err)
		}
	})
}

// TestDispatch_RespReadDeadlineEnforced verifies AC-011 (BC-2.07.003 Invariant 2,
// ADR-012 §7, Ruling V / ARCH-12 v1.5): dispatch() MUST apply a read deadline
// derived from ctx before decoding the RPC response. Two sub-cases:
//
//  1. ctx WITH deadline: the ctx deadline fires when the server goes silent.
//  2. ctx WITHOUT deadline (context.Background()): the 30s fallback
//     (rpcResponseFallbackTimeout) must arm the read deadline. To make this
//     observable without a 30s wait, the test overrides rpcResponseFallbackTimeout
//     to 100ms via t.Cleanup-restored assignment.
//
// Also asserts the deadline is CLEARED after dispatch returns: a subsequent
// blocking read on the same conn (post-dispatch) must not inherit the old deadline.
// This is verified by capturing SetReadDeadline calls through a recording wrapper.
//
// RED because: current dispatch() only sets the deadline when ctx has one
// (if dl, ok := ctx.Deadline(); ok { _ = conn.SetReadDeadline(dl) }) — it has NO
// 30s fallback for a deadline-less ctx, swallows the SetReadDeadline error, and
// never clears the deadline via defer. The no-deadline sub-case hangs or fires too
// late; the deadline-clear sub-case never zeroes the deadline.
//
// BC: BC-2.07.003 Invariant 2; AC-011; Ruling V (ARCH-12 v1.5); go.md rule 7.
func TestDispatch_RespReadDeadlineEnforced(t *testing.T) {
	t.Parallel()

	// sub-case 1: ctx WITH deadline — existing coverage preserved.
	t.Run("ctx_with_deadline_fires_before_silent_server", func(t *testing.T) {
		t.Parallel()

		server, client := net.Pipe()
		t.Cleanup(func() {
			_ = client.Close()
			_ = server.Close()
		})

		// Mock server: drain the RPC request then go permanently silent.
		go func() {
			defer func() { _ = server.Close() }()
			// Server deadline longer than ctx (200ms) so ctx fires first (GREEN).
			// In RED state (no deadline), dispatch blocks until server deadline expires.
			_ = server.SetDeadline(time.Now().Add(2 * time.Second))
			buf := make([]byte, 4096)
			_, _ = server.Read(buf) // drain the RPC request line
			// Go silent — never write the response.
			buf2 := make([]byte, 1)
			_, _ = server.Read(buf2)
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		start := time.Now()
		_, err := dispatch(ctx, client, "router.status", nil)
		elapsed := time.Since(start)

		if err == nil {
			t.Errorf("Ruling V (ctx deadline) violated: dispatch() returned nil with a silent server — expected a non-nil timeout error; dispatch() must call conn.SetReadDeadline from ctx before the response Decode (RED Gate)")
		}
		const maxAllowed = 1 * time.Second
		if elapsed > maxAllowed {
			t.Errorf("Ruling V (ctx deadline) violated: dispatch() hung for %v (> %v) — read deadline from ctx was not applied (RED Gate)", elapsed, maxAllowed)
		}
	})

	// sub-case 2: ctx WITHOUT deadline — the 30s fallback (rpcResponseFallbackTimeout)
	// must arm a deadline so dispatch does not hang forever.
	//
	// Strategy (option a from spec): override rpcResponseFallbackTimeout to 100ms
	// and restore via t.Cleanup. The test asserts dispatch returns a non-nil error
	// within a bounded window, proving the fallback was armed.
	//
	// RED because: current dispatch() checks `if dl, ok := ctx.Deadline(); ok { ... }`
	// — with context.Background() (no deadline), ok==false and SetReadDeadline is
	// never called. dispatch() hangs until the server goroutine closes the pipe.
	t.Run("no_deadline_ctx_fallback_arms_deadline", func(t *testing.T) {
		t.Parallel()

		// Override the fallback timeout to 100ms so the test completes quickly.
		// Restore via t.Cleanup so parallel tests are unaffected.
		orig := rpcResponseFallbackTimeout
		rpcResponseFallbackTimeout = 100 * time.Millisecond
		t.Cleanup(func() { rpcResponseFallbackTimeout = orig })

		server, client := net.Pipe()
		t.Cleanup(func() {
			_ = client.Close()
			_ = server.Close()
		})

		// Mock server: drain the RPC request then hang (never writes response).
		// Server deadline is 2s — much longer than the 100ms fallback so the
		// fallback fires first (GREEN). In RED state, dispatch() blocks until the
		// server deadline expires.
		go func() {
			defer func() { _ = server.Close() }()
			_ = server.SetDeadline(time.Now().Add(2 * time.Second))
			buf := make([]byte, 4096)
			_, _ = server.Read(buf) // drain RPC request
			// Hang — never send response.
			buf2 := make([]byte, 1)
			_, _ = server.Read(buf2)
		}()

		// No deadline on context — fallback path must engage.
		start := time.Now()
		_, err := dispatch(context.Background(), client, "router.status", nil)
		elapsed := time.Since(start)

		// dispatch() MUST return a non-nil error (fallback deadline fires).
		// RED: current dispatch() skips SetReadDeadline with no-deadline ctx —
		// it blocks until the server goroutine's 2s deadline closes the pipe.
		if err == nil {
			t.Errorf("Ruling V (no-deadline fallback) violated: dispatch() returned nil with no-deadline ctx and silent server — rpcResponseFallbackTimeout must arm conn.SetReadDeadline (RED Gate)")
		}
		// Must return within a generous bound (fallback=100ms; allow 1s headroom).
		// RED: without fallback, dispatch blocks ~2s (server deadline), failing this.
		const maxAllowed = 1 * time.Second
		if elapsed > maxAllowed {
			t.Errorf("Ruling V (no-deadline fallback) violated: dispatch() hung for %v (> %v) — rpcResponseFallbackTimeout fallback was not applied to conn.SetReadDeadline (RED Gate)", elapsed, maxAllowed)
		}
	})

	// sub-case 3: deadline is CLEARED after dispatch returns (defer-clear).
	// After a successful dispatch call on a conn, a subsequent read on that
	// conn must not immediately time out due to a stale deadline.
	//
	// Strategy: use a recordingConn wrapper that tracks all SetReadDeadline calls.
	// After dispatch returns, assert the final SetReadDeadline call used time.Time{}
	// (zero time = cleared). This proves defer func() { _ = conn.SetReadDeadline(time.Time{}) }()
	// was executed.
	//
	// RED because: current dispatch() never calls SetReadDeadline at all, so no
	// clearing call is made. The recordingConn assertion for a zero-time final
	// call will fail (no calls recorded).
	t.Run("deadline_cleared_after_dispatch_returns", func(t *testing.T) {
		t.Parallel()

		server, client := net.Pipe()
		t.Cleanup(func() {
			_ = client.Close()
			_ = server.Close()
		})

		// Mock server: drain request, reply with a successful response.
		go func() {
			defer func() { _ = server.Close() }()
			_ = server.SetDeadline(time.Now().Add(2 * time.Second))
			buf := make([]byte, 4096)
			_, _ = server.Read(buf)
			const resp = `{"type":"response","id":"1","ok":true,"data":{}}` + "\n"
			_, _ = server.Write([]byte(resp))
		}()

		// Wrap client in a recordingConn to capture SetReadDeadline calls.
		rec := &recordingConn{Conn: client}

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		_, err := dispatch(ctx, rec, "router.status", nil)
		if err != nil {
			t.Fatalf("dispatch() returned unexpected error: %v", err)
		}

		// After dispatch returns, the last SetReadDeadline call MUST be time.Time{} (clear).
		// RED: current dispatch() makes no SetReadDeadline calls at all.
		if len(rec.deadlines) == 0 {
			t.Errorf("Ruling V (deadline-clear) violated: dispatch() made no SetReadDeadline calls — expected at least one call to arm and one to clear (RED Gate)")
		} else {
			last := rec.deadlines[len(rec.deadlines)-1]
			if !last.IsZero() {
				t.Errorf("Ruling V (deadline-clear) violated: last SetReadDeadline call was %v, want time.Time{} (zero) — deadline must be cleared via defer after dispatch returns (RED Gate)", last)
			}
		}
	})
}

// recordingConn wraps a net.Conn and records all SetReadDeadline calls.
// Used by TestDispatch_RespReadDeadlineEnforced sub-case 3.
type recordingConn struct {
	net.Conn
	deadlines []time.Time
}

func (r *recordingConn) SetReadDeadline(t time.Time) error {
	r.deadlines = append(r.deadlines, t)
	return r.Conn.SetReadDeadline(t)
}

// TestDispatch_IDEchoEnforced verifies AC-010 (BC-2.07.002 PC-3, Ruling X /
// ARCH-12 v1.5 / ADR-012 §3 step 6): dispatch() MUST generate a non-constant
// per-call request id, and MUST verify resp.ID == req.ID after decoding. A response
// carrying a different ID than the request MUST cause dispatch() to return a
// non-nil E-RPC-001 error.
//
// Two sub-cases:
//  1. ID mismatch: server replies with a wrong id — dispatch() must return non-nil error.
//  2. Non-constant id: two sequential dispatch() calls (each on its own conn) emit
//     DIFFERENT ids, and neither equals the constant "1". The mock server echoes
//     back whatever id it received, so the echo-check still passes once the impl
//     uses a non-constant id.
//
// RED because: current dispatch() hardcodes req.ID = "1". The non-constant
// sub-case directly fails the "not equal to '1'" assertion. The mismatch sub-case
// is also RED because dispatch() never checks resp.ID.
//
// BC: BC-2.07.002 PC-3; AC-010; Ruling X (ARCH-12 v1.5); ADR-012 §3 step 6.
func TestDispatch_IDEchoEnforced(t *testing.T) {
	t.Parallel()

	// sub-case 1: ID mismatch — server replies with a wrong id.
	// Preserved from the prior TestDispatch_RejectsIDMismatch.
	t.Run("id_mismatch_returns_error", func(t *testing.T) {
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

			// Reply with a DELIBERATELY WRONG id.
			const wrongIDResponse = `{"type":"response","id":"WRONG-ID","ok":true,"data":{}}` + "\n"
			_, _ = server.Write([]byte(wrongIDResponse))
		}()

		_, err := dispatch(context.Background(), client, "router.status", nil)

		raw := <-rawRequestCh

		// dispatch() MUST return non-nil error on ID mismatch.
		// RED: current dispatch() hardcodes id "1" and never verifies resp.ID.
		if err == nil {
			t.Errorf("Ruling X violated: dispatch() returned nil on ID mismatch (server sent 'WRONG-ID') — expected non-nil E-RPC-001; dispatch() must check resp.ID == req.ID (RED Gate)")
		}

		if raw == nil {
			t.Fatal("Ruling X: mock server received no bytes from dispatch()")
		}
		rawStr := string(raw)
		if !strings.Contains(rawStr, `"id":`) {
			t.Errorf("Ruling X: dispatch() request missing 'id' field; got: %s", rawStr)
		}
		if strings.Contains(rawStr, `"id":""`) {
			t.Errorf("Ruling X: dispatch() sent empty 'id' field; got: %s", rawStr)
		}
	})

	// sub-case 2: non-constant id — two sequential dispatch() calls emit different
	// ids, and neither equals the hardcoded constant "1".
	//
	// Each call gets its own net.Pipe (single-RPC-per-connection). The mock server
	// captures the raw request JSON and echoes back the received id so the echo
	// check passes once the impl uses a non-constant id.
	//
	// RED because: current dispatch() hardcodes ID: "1". Both captured ids will be
	// "1", directly triggering the "must not equal '1'" assertion.
	t.Run("id_is_non_constant_across_calls", func(t *testing.T) {
		t.Parallel()

		// makeEchoConn sets up a net.Pipe mock server that:
		//   1. Reads one request line, captures it.
		//   2. Decodes the id from the raw JSON.
		//   3. Echoes back {"type":"response","id":"<captured_id>","ok":true,"data":{}}.
		//
		// Returns (client conn, channel that delivers the captured id string).
		// All t.Fatalf calls are hoisted before go func() per Ruling W discipline.
		makeEchoConn := func(t *testing.T) (net.Conn, <-chan string) {
			t.Helper()
			server, client := net.Pipe()
			t.Cleanup(func() { _ = client.Close() })

			idCh := make(chan string, 1)
			go func() {
				defer func() { _ = server.Close() }()
				_ = server.SetDeadline(time.Now().Add(2 * time.Second))

				buf := make([]byte, 4096)
				n, err := server.Read(buf)
				if err != nil || n == 0 {
					idCh <- ""
					return
				}
				raw := buf[:n]

				// Decode the id field from the raw JSON request.
				var req struct {
					ID string `json:"id"`
				}
				capturedID := ""
				if jsonErr := json.Unmarshal(raw, &req); jsonErr == nil {
					capturedID = req.ID
				}
				idCh <- capturedID

				// Echo the id back in the response so dispatch()'s echo-check passes.
				resp := fmt.Sprintf(`{"type":"response","id":%q,"ok":true,"data":{}}`, capturedID) + "\n"
				_, _ = server.Write([]byte(resp))
			}()
			return client, idCh
		}

		// Call 1.
		conn1, idCh1 := makeEchoConn(t)
		_, err1 := dispatch(context.Background(), conn1, "router.status", nil)
		id1 := <-idCh1

		// Call 2.
		conn2, idCh2 := makeEchoConn(t)
		_, err2 := dispatch(context.Background(), conn2, "router.status", nil)
		id2 := <-idCh2

		// Both calls should succeed (echo-check passes when ids match).
		if err1 != nil {
			t.Errorf("Ruling X (non-constant id): dispatch() call 1 returned unexpected error: %v", err1)
		}
		if err2 != nil {
			t.Errorf("Ruling X (non-constant id): dispatch() call 2 returned unexpected error: %v", err2)
		}

		// IDs must not be the constant "1".
		// RED: current impl hardcodes ID: "1"; this assertion fires immediately.
		if id1 == "1" {
			t.Errorf("Ruling X (non-constant id) violated: dispatch() call 1 used constant id %q — id must be non-constant per AC-010 (RED Gate)", id1)
		}
		if id2 == "1" {
			t.Errorf("Ruling X (non-constant id) violated: dispatch() call 2 used constant id %q — id must be non-constant per AC-010 (RED Gate)", id2)
		}

		// IDs must differ across calls (probabilistically guaranteed if non-constant).
		// With time.Now().UnixNano() this is deterministic for sequential calls.
		if id1 != "" && id2 != "" && id1 == id2 {
			t.Errorf("Ruling X (non-constant id) violated: dispatch() produced the same id %q for two different calls — id must vary per call per AC-010 (RED Gate)", id1)
		}

		// IDs must be non-empty.
		if id1 == "" {
			t.Error("Ruling X: dispatch() call 1 emitted empty id")
		}
		if id2 == "" {
			t.Error("Ruling X: dispatch() call 2 emitted empty id")
		}
	})
}
