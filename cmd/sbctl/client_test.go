// Tests for cmd/sbctl: Authenticate(), loadEd25519Key(), and JSON envelope
// formatting.
//
// Tests are named per BC-based convention (BC-2.07.002, VP-067) for full
// traceability. All tests MUST fail before implementation (Red Gate per
// BC-5.38.001).
//
// Package main (internal test file) so unexported names (loadEd25519Key,
// newSuccessEnvelope, newErrorEnvelope, Authenticate) are directly accessible.
package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
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
func mockServerWithRead(t *testing.T, challenge map[string]any, authResult map[string]any) net.Conn {
	t.Helper()
	server, client := net.Pipe()
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
//
// RED GATE: sub-cases (a) through (h) must FAIL before implementation because
// the stub returns errors.New("not implemented") — which assertProtocolError
// explicitly rejects. The happy-path sub-case must FAIL because the stub
// returns non-nil on AUTH_OK.
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

		err := Authenticate(client, opPriv)
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

		err := Authenticate(client, opPriv)
		assertProtocolError(t, err, "VP067_b_malformed_challenge_json_decode_error")
	})

	t.Run("VP067_b_malformed_challenge_missing_nonce", func(t *testing.T) {
		t.Parallel()
		// Challenge has no nonce field — must be rejected per ARCH-12 step 2.
		conn := mockServerWithRead(t,
			map[string]any{"type": "challenge"}, // nonce absent
			nil,                                 // no auth result (client should error before this)
		)
		err := Authenticate(conn, opPriv)
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
		err := Authenticate(conn, opPriv)
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
		err := Authenticate(conn, opPriv)
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
		err := Authenticate(conn, opPriv)
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
		err := Authenticate(conn, opPriv)
		assertProtocolError(t, err, "VP067_f_wrong_response_type_returns_error")
	})

	t.Run("VP067_g_truncated_stream_after_challenge", func(t *testing.T) {
		t.Parallel()
		// Server closes connection after CHALLENGE — EOF before AUTH_OK/AUTH_FAIL.
		server, client := net.Pipe()
		t.Cleanup(func() { _ = client.Close() })
		go func() {
			defer func() { _ = server.Close() }()
			_ = server.SetDeadline(time.Now().Add(2 * time.Second))
			challenge := wellFormedChallenge(t)
			enc := json.NewEncoder(server)
			if err := enc.Encode(challenge); err != nil {
				return
			}
			// Read CHALLENGE_RESPONSE (or timeout) then close without sending auth result.
			buf := make([]byte, 8192)
			_, _ = server.Read(buf)
			// close immediately — truncated stream
		}()

		err := Authenticate(client, opPriv)
		assertProtocolError(t, err, "VP067_g_truncated_stream_after_challenge")
	})

	t.Run("VP067_h_oversized_auth_response_bounded_by_limit_reader", func(t *testing.T) {
		t.Parallel()
		// Server sends a > 64 KiB auth response. io.LimitReader must truncate it.
		// Authenticate must return a decode error, NOT succeed or hang.
		// This proves CWE-400 protection on the second json.Decoder (ADR-012 §6).
		server, client := net.Pipe()
		t.Cleanup(func() { _ = client.Close() })
		go func() {
			defer func() { _ = server.Close() }()
			_ = server.SetDeadline(time.Now().Add(5 * time.Second))
			challenge := wellFormedChallenge(t)
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

		err := Authenticate(client, opPriv)
		assertProtocolError(t, err, "VP067_h_oversized_auth_response_bounded_by_limit_reader")
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
		err := Authenticate(conn, opPriv)
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
// RED GATE: this test will PASS in the Red Gate phase because the stub never
// writes anything — an empty wire message cannot contain the private key. The
// test documents the security invariant; the implementer must keep it passing.
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

	// doneCh signals when the server goroutine has finished capturing bytes.
	doneCh := make(chan []byte, 1)

	go func() {
		// Apply a deadline so we don't block permanently if the stub never reads.
		_ = server.SetDeadline(time.Now().Add(1 * time.Second))
		defer func() { _ = server.Close() }()

		nonce, nonceB64 := freshNonce(t)
		_, daemonPriv, _ := ed25519.GenerateKey(rand.Reader)
		sig := ed25519.Sign(daemonPriv, nonce)

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

	_ = Authenticate(client, opPriv)
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
		privKey, err := loadEd25519Key(fixtureKey)
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
		_, err := loadEd25519Key(oversized)
		if err == nil {
			t.Fatal("loadEd25519Key accepted a file > 64 KiB — io.LimitReader boundary not enforced (CWE-400)")
		}
	})

	t.Run("nonexistent_file_returns_error", func(t *testing.T) {
		t.Parallel()
		_, err := loadEd25519Key("/nonexistent/path/to/key.pem")
		if err == nil {
			t.Fatal("loadEd25519Key returned nil error for a nonexistent path")
		}
	})
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
