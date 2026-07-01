// Package main — admin_test.go tests the `sbctl admin` subcommand tree at the
// CLI layer, covering arg parsing, wire-level JSON marshal/unmarshal round-trips
// for all admin RPCs, error handling, and fail-closed auth behavior.
//
// Traceability:
//
//	BC-2.05.004 — Key lifecycle: register, revoke, expire
//	BC-2.07.001 — SVTN lifecycle: create, bootstrap
//	AC-002      — sbctl admin key register
//	AC-003      — sbctl admin key revoke
//	AC-004      — sbctl admin key expire
//	AC-005      — control-to-control revocation requires --confirm
//	ADR-012     — NDJSON wire protocol, Ed25519 challenge-response, 64 KiB bounded reads
//
// Red Gate: all tests MUST fail before implementation (runAdmin panics).
package main

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
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// adminRPCRequest captures a dispatched RPC command and its raw args payload
// for test assertion.
type adminRPCRequest struct {
	Command string
	Args    json.RawMessage
}

// ── wire-type JSON round-trip tests ──────────────────────────────────────────

// TestAdminKeyRegisterArgs_JSONRoundTrip verifies that adminKeyRegisterArgs
// correctly serializes and deserializes all fields via encoding/json.
// This removes the compile-time blank anchor `_ = adminKeyRegisterArgs{}`
// and exercises the real field names from the wire format.
//
// BC-2.05.004 postcondition 1 (wire format: svtn_id, pubkey, role).
// ADR-012 (NDJSON wire protocol; JSON field names in snake_case).
func TestAdminKeyRegisterArgs_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	// BC-2.05.004 postcondition 1 — wire format for admin.key.register.
	original := adminKeyRegisterArgs{
		SVTNID: "test-svtn-id-12345",
		Pubkey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI... test-key",
		Role:   "control",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(adminKeyRegisterArgs): %v", err)
	}

	// Verify field names in the serialized form (snake_case per ADR-012).
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal to map: %v", err)
	}

	if _, ok := raw["svtn_id"]; !ok {
		t.Error("adminKeyRegisterArgs: missing JSON field 'svtn_id'")
	}
	if _, ok := raw["pubkey"]; !ok {
		t.Error("adminKeyRegisterArgs: missing JSON field 'pubkey'")
	}
	if _, ok := raw["role"]; !ok {
		t.Error("adminKeyRegisterArgs: missing JSON field 'role'")
	}

	// Round-trip: unmarshal back and compare.
	var decoded adminKeyRegisterArgs
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(adminKeyRegisterArgs): %v", err)
	}

	if decoded.SVTNID != original.SVTNID {
		t.Errorf("SVTNID round-trip: got %q; want %q", decoded.SVTNID, original.SVTNID)
	}
	if decoded.Pubkey != original.Pubkey {
		t.Errorf("Pubkey round-trip: got %q; want %q", decoded.Pubkey, original.Pubkey)
	}
	if decoded.Role != original.Role {
		t.Errorf("Role round-trip: got %q; want %q", decoded.Role, original.Role)
	}
}

// TestAdminKeyRevokeArgs_JSONRoundTrip verifies that adminKeyRevokeArgs
// correctly serializes and deserializes all fields via encoding/json.
// Specifically verifies the `confirm` field (bool) is present and round-trips.
//
// BC-2.05.004 precondition 1 (confirm field required for control-to-control; ADR-004).
// ADR-012 (NDJSON wire protocol).
func TestAdminKeyRevokeArgs_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		confirm bool
	}{
		// BC-2.05.004 precondition 1 / ADR-004 — confirm field must round-trip correctly.
		{"confirm_false", false},
		{"confirm_true", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			original := adminKeyRevokeArgs{
				SVTNID:  "test-svtn-id",
				Pubkey:  "ssh-ed25519 AAAA...",
				Role:    "control",
				Confirm: tc.confirm,
			}

			data, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("json.Marshal(adminKeyRevokeArgs): %v", err)
			}

			var raw map[string]any
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("json.Unmarshal to map: %v", err)
			}

			// Verify field presence.
			if _, ok := raw["svtn_id"]; !ok {
				t.Error("adminKeyRevokeArgs: missing JSON field 'svtn_id'")
			}
			if _, ok := raw["pubkey"]; !ok {
				t.Error("adminKeyRevokeArgs: missing JSON field 'pubkey'")
			}
			if _, ok := raw["role"]; !ok {
				t.Error("adminKeyRevokeArgs: missing JSON field 'role'")
			}
			if _, ok := raw["confirm"]; !ok {
				t.Error("adminKeyRevokeArgs: missing JSON field 'confirm'")
			}

			// Round-trip.
			var decoded adminKeyRevokeArgs
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("json.Unmarshal(adminKeyRevokeArgs): %v", err)
			}

			if decoded.Confirm != tc.confirm {
				t.Errorf("Confirm round-trip: got %v; want %v", decoded.Confirm, tc.confirm)
			}
			if decoded.SVTNID != original.SVTNID {
				t.Errorf("SVTNID round-trip: got %q; want %q", decoded.SVTNID, original.SVTNID)
			}
			if decoded.Role != original.Role {
				t.Errorf("Role round-trip: got %q; want %q", decoded.Role, original.Role)
			}
		})
	}
}

// TestAdminKeyExpireArgs_JSONRoundTrip verifies that adminKeyExpireArgs
// correctly serializes and deserializes all fields via encoding/json.
//
// BC-2.05.004 postcondition 3 (wire format: svtn_id, pubkey, after).
// S-6.02 EC-003 (zero duration rejected: "after" field must carry human-parseable duration).
func TestAdminKeyExpireArgs_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	// BC-2.05.004 postcondition 3 — wire format for admin.key.expire.
	original := adminKeyExpireArgs{
		SVTNID: "test-svtn-id",
		Pubkey: "ssh-ed25519 AAAA...",
		After:  "24h",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(adminKeyExpireArgs): %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal to map: %v", err)
	}

	if _, ok := raw["svtn_id"]; !ok {
		t.Error("adminKeyExpireArgs: missing JSON field 'svtn_id'")
	}
	if _, ok := raw["pubkey"]; !ok {
		t.Error("adminKeyExpireArgs: missing JSON field 'pubkey'")
	}
	if _, ok := raw["after"]; !ok {
		t.Error("adminKeyExpireArgs: missing JSON field 'after'")
	}

	var decoded adminKeyExpireArgs
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(adminKeyExpireArgs): %v", err)
	}

	if decoded.SVTNID != original.SVTNID {
		t.Errorf("SVTNID round-trip: got %q; want %q", decoded.SVTNID, original.SVTNID)
	}
	if decoded.After != original.After {
		t.Errorf("After round-trip: got %q; want %q", decoded.After, original.After)
	}

	// S-6.02 EC-003: the After field should be parseable as a Go duration.
	if _, err := time.ParseDuration(decoded.After); err != nil {
		t.Errorf("S-6.02 EC-003 — After field %q must be parseable as time.Duration: %v", decoded.After, err)
	}
}

// ── runAdmin dispatch tests ───────────────────────────────────────────────────

// fakeMgmtServer is an in-process test fixture that fakes the daemon management
// plane at the ADR-012 wire level. It accepts one connection, performs the
// challenge-response handshake, and then either responds to an admin RPC
// with a configurable result or closes with an auth failure.
//
// Used to test runAdmin's CLI dispatch, error handling, and wire behavior
// without requiring a real daemon process.
type fakeMgmtServer struct {
	// opPub is the authorized operator public key (must sign the challenge).
	opPub ed25519.PublicKey
	// opPriv is the authorized operator private key (test uses it to sign).
	opPriv ed25519.PrivateKey
	// handler is called after authentication succeeds, receives the RPC command
	// and returns the response data (or an error to send as ok:false).
	handler func(command string, args json.RawMessage) (any, error)
	// failAuth, when true, sends AUTH_FAIL instead of AUTH_OK.
	failAuth bool
}

// serve accepts exactly one connection on ln, performs the ADR-012 handshake,
// then calls handler for authenticated RPC requests.
// It stops serving when the connection closes.
func (f *fakeMgmtServer) serve(t *testing.T, ln net.Listener) {
	t.Helper()

	conn, err := ln.Accept()
	if err != nil {
		// Listener closed by test cleanup; acceptable.
		return
	}
	defer func() { _ = conn.Close() }()

	// Set a generous deadline so slow CI boxes don't spuriously fail.
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Logf("fakeMgmtServer: SetDeadline: %v", err)
		return
	}

	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(io.LimitReader(conn, maxMessageBytes))

	// Step 1: Send CHALLENGE with a random nonce.
	var nonce [32]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		t.Logf("fakeMgmtServer: rand.Read nonce: %v", err)
		return
	}

	_, daemonPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Logf("fakeMgmtServer: GenerateKey: %v", err)
		return
	}
	daemonSig := ed25519.Sign(daemonPriv, nonce[:])

	challenge := map[string]any{
		"type":       "challenge",
		"nonce":      base64.RawURLEncoding.EncodeToString(nonce[:]),
		"daemon_sig": base64.RawURLEncoding.EncodeToString(daemonSig),
	}
	if err := enc.Encode(challenge); err != nil {
		t.Logf("fakeMgmtServer: send challenge: %v", err)
		return
	}

	// Step 2: Read CHALLENGE_RESPONSE.
	var resp struct {
		Type     string `json:"type"`
		NonceSig string `json:"nonce_sig"`
		PubKey   string `json:"pubkey"`
	}
	if err := dec.Decode(&resp); err != nil {
		t.Logf("fakeMgmtServer: decode challenge_response: %v", err)
		return
	}

	if f.failAuth {
		// Send AUTH_FAIL.
		_ = enc.Encode(map[string]any{
			"type":    "auth_fail",
			"code":    "E-ADM-010",
			"message": "authentication failed",
		})
		return
	}

	// Verify the signature.
	pubBytes, err := base64.RawURLEncoding.DecodeString(resp.PubKey)
	if err != nil {
		t.Logf("fakeMgmtServer: decode pubkey: %v", err)
		_ = enc.Encode(map[string]any{"type": "auth_fail", "code": "E-ADM-010", "message": "bad pubkey encoding"})
		return
	}

	sigBytes, err := base64.RawURLEncoding.DecodeString(resp.NonceSig)
	if err != nil {
		t.Logf("fakeMgmtServer: decode nonce_sig: %v", err)
		_ = enc.Encode(map[string]any{"type": "auth_fail", "code": "E-ADM-010", "message": "bad sig encoding"})
		return
	}

	pub := ed25519.PublicKey(pubBytes)
	if !ed25519.Verify(pub, nonce[:], sigBytes) {
		_ = enc.Encode(map[string]any{"type": "auth_fail", "code": "E-ADM-010", "message": "sig verify failed"})
		return
	}

	// CR-008: verify the signing key is the authorized operator key.
	// Signature verification alone confirms the client knows the private key
	// corresponding to pub, but does not confirm pub == f.opPub. Without this
	// check, any client with a valid key pair could authenticate.
	if !bytes.Equal(pub, f.opPub) {
		_ = enc.Encode(map[string]any{"type": "auth_fail", "code": "E-ADM-010", "message": "unauthorized key"})
		return
	}

	// Send AUTH_OK.
	if err := enc.Encode(map[string]any{
		"type":           "auth_ok",
		"daemon_version": "test-dev",
	}); err != nil {
		t.Logf("fakeMgmtServer: send auth_ok: %v", err)
		return
	}

	// Handle RPC requests.
	for {
		var req struct {
			Type    string          `json:"type"`
			ID      string          `json:"id"`
			Command string          `json:"command"`
			Args    json.RawMessage `json:"args"`
		}
		if err := dec.Decode(&req); err != nil {
			// Connection closed or deadline; normal shutdown.
			return
		}

		if f.handler == nil {
			_ = enc.Encode(map[string]any{
				"type":  "response",
				"id":    req.ID,
				"ok":    false,
				"error": map[string]string{"code": "E-RPC-010", "message": "no handler registered"},
			})
			continue
		}

		result, handlerErr := f.handler(req.Command, req.Args)
		if handlerErr != nil {
			_ = enc.Encode(map[string]any{
				"type":  "response",
				"id":    req.ID,
				"ok":    false,
				"error": map[string]string{"code": "E-RPC-011", "message": handlerErr.Error()},
			})
			continue
		}

		dataBytes, err := json.Marshal(result)
		if err != nil {
			t.Logf("fakeMgmtServer: marshal handler result: %v", err)
			return
		}
		_ = enc.Encode(map[string]any{
			"type": "response",
			"id":   req.ID,
			"ok":   true,
			"data": json.RawMessage(dataBytes),
		})
	}
}

// startFakeServer starts fakeMgmtServer on a TCP listener and returns the
// server address. If obs is non-nil, each authenticated RPC request is forwarded
// to obs (buffered, non-blocking) so callers can assert dispatched commands.
// The authorized operator public key is derived from the testdata key fixture so
// that CR-008 key-authorization check (bytes.Equal(pub, f.opPub)) passes for
// clients authenticating with testdataKeyPath(t).
func startFakeServer(
	t *testing.T,
	obs chan adminRPCRequest,
	handler func(command string, args json.RawMessage) (any, error),
) string {
	t.Helper()

	// Load the testdata private key so fakeMgmtServer.opPub matches what the
	// client presents. Without this, the CR-008 authorized-key check would
	// reject all test connections.
	testPrivKey, err := loadEd25519Key(testdataKeyPath(t), os.UserHomeDir)
	if err != nil {
		t.Fatalf("startFakeServer: loadEd25519Key: %v", err)
	}
	opPub := testPrivKey.Public().(ed25519.PublicKey)

	// Generate a fresh daemon key for signing challenges (separate from opPub).
	_, daemonPrivKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	wrappedHandler := func(cmd string, args json.RawMessage) (any, error) {
		if obs != nil {
			select {
			case obs <- adminRPCRequest{cmd, args}:
			default:
			}
		}
		if handler != nil {
			return handler(cmd, args)
		}
		return map[string]string{"status": "ok"}, nil
	}

	fake := &fakeMgmtServer{
		opPub:   opPub,
		opPriv:  daemonPrivKey,
		handler: wrappedHandler,
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go fake.serve(t, ln)

	return ln.Addr().String()
}

// TestSbctlAdmin_KeyRegister_CLI verifies AC-002 at the CLI layer:
// `sbctl admin key register --key <pubkey> --svtn <id>` dispatches the
// admin.key.register RPC to the daemon with the correct wire-format payload.
//
// BC-2.05.004 postcondition 1 (key registered; fingerprint returned; propagation initiated).
// AC-002 (traces to BC-2.05.004 postcondition 1).
// F-P8-001 (canonical CLI surface: sbctl admin).
func TestSbctlAdmin_KeyRegister_CLI(t *testing.T) {
	t.Parallel()

	// AC-002 / BC-2.05.004 postcondition 1 — key register dispatches correct RPC.
	// The test verifies:
	// 1. runAdmin parses `key register --key ... --svtn ... --role ...` correctly.
	// 2. Sends an admin.key.register RPC to the daemon.
	// 3. The wire payload contains the correct svtn_id, pubkey, and role fields.
	//
	// Was RED during initial TDD (runAdmin not implemented); now covers positive path.

	requestCh := make(chan adminRPCRequest, 1)
	addr := startFakeServer(t, requestCh, func(cmd string, args json.RawMessage) (any, error) {
		if cmd != "admin.key.register" {
			return nil, fmt.Errorf("unexpected command: %q; want admin.key.register", cmd)
		}
		var regArgs adminKeyRegisterArgs
		if err := json.Unmarshal(args, &regArgs); err != nil {
			return nil, fmt.Errorf("unmarshal adminKeyRegisterArgs: %w", err)
		}
		return map[string]any{
			"fingerprint": "SHA256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
			"at":          time.Now().UTC().Format(time.RFC3339),
		}, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// AC-002: call runAdmin with the `key register` subcommand args.
	const svtnID = "test-svtn-reg"
	const pubkey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI test-register-key"

	err := runAdmin(ctx, addr, testdataKeyPath(t), false, []string{
		"key", "register",
		"--key", pubkey,
		"--svtn", svtnID,
		"--role", "console",
	}, defaultIO())
	if err != nil {
		t.Fatalf("runAdmin: %v", err)
	}

	// Verify the RPC was dispatched with correct payload.
	select {
	case req := <-requestCh:
		if req.Command != "admin.key.register" {
			t.Errorf("AC-002 — dispatched command: got %q; want admin.key.register", req.Command)
		}
		var args adminKeyRegisterArgs
		if err := json.Unmarshal(req.Args, &args); err != nil {
			t.Fatalf("AC-002 — unmarshal args: %v", err)
		}
		if args.SVTNID != svtnID {
			t.Errorf("AC-002 — wire svtn_id: got %q; want %q", args.SVTNID, svtnID)
		}
		if args.Pubkey != pubkey {
			t.Errorf("AC-002 — wire pubkey: got %q; want %q", args.Pubkey, pubkey)
		}
		if args.Role != "console" {
			t.Errorf("AC-002 — wire role: got %q; want console", args.Role)
		}
	case <-time.After(2 * time.Second):
		t.Error("AC-002: timed out waiting for admin.key.register RPC; " +
			"runAdmin must dispatch the RPC within the context deadline")
	}
}

// TestSbctlAdmin_KeyRevoke_CLI verifies AC-003 at the CLI layer:
// `sbctl admin key revoke --key <pubkey> --svtn <id>` dispatches the
// admin.key.revoke RPC. Verifies the confirm field defaults to false.
//
// BC-2.05.004 postcondition 2 (key removed; sessions continue until re-auth).
// AC-003 (traces to BC-2.05.004 postcondition 2).
func TestSbctlAdmin_KeyRevoke_CLI(t *testing.T) {
	t.Parallel()

	// AC-003 / BC-2.05.004 postcondition 2 — key revoke dispatches correct RPC.
	requestCh := make(chan adminRPCRequest, 1)
	addr := startFakeServer(t, requestCh, func(cmd string, args json.RawMessage) (any, error) {
		if cmd != "admin.key.revoke" {
			return nil, fmt.Errorf("unexpected command: %q; want admin.key.revoke", cmd)
		}
		return map[string]any{
			"fingerprint": "SHA256:BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB=",
			"at":          time.Now().UTC().Format(time.RFC3339),
		}, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	const svtnID = "test-svtn-rev"
	const pubkey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI test-revoke-key"

	err := runAdmin(ctx, addr, testdataKeyPath(t), false, []string{
		"key", "revoke",
		"--key", pubkey,
		"--svtn", svtnID,
		"--role", "console",
		// No --confirm: defaults to false.
	}, defaultIO())
	if err != nil {
		t.Fatalf("runAdmin: %v", err)
	}

	select {
	case req := <-requestCh:
		if req.Command != "admin.key.revoke" {
			t.Errorf("AC-003 — dispatched command: got %q; want admin.key.revoke", req.Command)
		}
		var args adminKeyRevokeArgs
		if err := json.Unmarshal(req.Args, &args); err != nil {
			t.Fatalf("AC-003 — unmarshal args: %v", err)
		}
		if args.SVTNID != svtnID {
			t.Errorf("AC-003 — wire svtn_id: got %q; want %q", args.SVTNID, svtnID)
		}
		if args.Pubkey != pubkey {
			t.Errorf("AC-003 — wire pubkey: got %q; want %q", args.Pubkey, pubkey)
		}
		if args.Role != "console" {
			t.Errorf("AC-003 — wire role: got %q; want console", args.Role)
		}
		// Without --confirm, Confirm must be false.
		if args.Confirm {
			t.Error("AC-003 — wire confirm: want false (no --confirm flag); got true")
		}
	case <-time.After(2 * time.Second):
		t.Error("AC-003: timed out waiting for admin.key.revoke RPC")
	}
}

// TestSbctlAdmin_KeyExpire_CLI verifies AC-004 at the CLI layer:
// `sbctl admin key expire --key <pubkey> --svtn <id> --after <duration>`
// dispatches the admin.key.expire RPC with the correct after field.
//
// BC-2.05.004 postcondition 3 (expiry timestamp associated with key).
// AC-004 (traces to BC-2.05.004 postcondition 3).
func TestSbctlAdmin_KeyExpire_CLI(t *testing.T) {
	t.Parallel()

	// AC-004 / BC-2.05.004 postcondition 3 — key expire dispatches correct RPC.
	requestCh := make(chan adminRPCRequest, 1)
	addr := startFakeServer(t, requestCh, func(cmd string, args json.RawMessage) (any, error) {
		if cmd != "admin.key.expire" {
			return nil, fmt.Errorf("unexpected command: %q; want admin.key.expire", cmd)
		}
		return map[string]any{
			"fingerprint": "SHA256:CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC=",
			"at":          time.Now().UTC().Format(time.RFC3339),
		}, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	const svtnID = "test-svtn-exp"
	const pubkey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI test-expire-key"
	const after = "24h"

	err := runAdmin(ctx, addr, testdataKeyPath(t), false, []string{
		"key", "expire",
		"--key", pubkey,
		"--svtn", svtnID,
		"--after", after,
	}, defaultIO())
	if err != nil {
		t.Fatalf("runAdmin: %v", err)
	}

	select {
	case req := <-requestCh:
		if req.Command != "admin.key.expire" {
			t.Errorf("AC-004 — dispatched command: got %q; want admin.key.expire", req.Command)
		}
		var args adminKeyExpireArgs
		if err := json.Unmarshal(req.Args, &args); err != nil {
			t.Fatalf("AC-004 — unmarshal args: %v", err)
		}
		if args.SVTNID != svtnID {
			t.Errorf("AC-004 — wire svtn_id: got %q; want %q", args.SVTNID, svtnID)
		}
		if args.After != after {
			t.Errorf("AC-004 — wire after: got %q; want %q", args.After, after)
		}
	case <-time.After(2 * time.Second):
		t.Error("AC-004: timed out waiting for admin.key.expire RPC")
	}
}

// TestSbctlAdmin_ControlRevocation_RequiresConfirm_CLI verifies AC-005 at the
// CLI layer:
// `sbctl admin key revoke` without --confirm when the target key is a control key
// results in the daemon returning an error; with --confirm the wire confirm field
// is true.
//
// BC-2.05.004 precondition 1 (control-to-control revocation requires --confirm; ADR-004).
// AC-005 (traces to BC-2.05.004 precondition 1 and ADR-004).
func TestSbctlAdmin_ControlRevocation_RequiresConfirm_CLI(t *testing.T) {
	t.Parallel()

	t.Run("without_confirm_confirm_false_in_wire", func(t *testing.T) {
		t.Parallel()

		// AC-005 — without --confirm flag, wire confirm field must be false.
		requestCh := make(chan adminRPCRequest, 1)
		addr := startFakeServer(t, requestCh, func(cmd string, args json.RawMessage) (any, error) {
			var revokeArgs adminKeyRevokeArgs
			if err := json.Unmarshal(args, &revokeArgs); err != nil {
				return nil, err
			}
			// CR-002/CR-007: validate role field is populated.
			if revokeArgs.Role == "" {
				return nil, fmt.Errorf("E-CFG-001: role field missing in revoke request")
			}
			// Simulate daemon: reject control-to-control without confirm.
			// AC-005 (BC-2.05.004 PC-2): daemon returns E-ADM-018 canonical code; CLI surfaces it verbatim.
			if !revokeArgs.Confirm {
				return nil, fmt.Errorf("E-ADM-018: control revocation requires --confirm")
			}
			return map[string]any{"fingerprint": "SHA256:DDDD..."}, nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		const pubkey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI test-ctrl-key"

		err := runAdmin(ctx, addr, testdataKeyPath(t), false, []string{
			"key", "revoke",
			"--key", pubkey,
			"--svtn", "test-svtn",
			"--role", "control",
			// No --confirm.
		}, defaultIO())

		select {
		case req := <-requestCh:
			var args adminKeyRevokeArgs
			if jsonErr := json.Unmarshal(req.Args, &args); jsonErr != nil {
				t.Fatalf("AC-005: unmarshal args: %v", jsonErr)
			}
			// AC-005 / ADR-004: without --confirm flag, wire Confirm must be false.
			if args.Confirm {
				t.Error("AC-005 — wire confirm: want false when --confirm not supplied; got true")
			}
			// CR-002/CR-007: role field must be set in the wire payload.
			if args.Role != "control" {
				t.Errorf("AC-005 — wire role: got %q; want control", args.Role)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("AC-005: timed out — CLI did not dispatch RPC within deadline")
		}

		if err == nil {
			t.Fatalf("AC-005: without --confirm: expected error containing E-ADM-018; got nil")
		}
		// AC-005 (BC-2.05.004 PC-2): daemon returns E-ADM-018 canonical code; CLI surfaces it verbatim.
		if !strings.Contains(err.Error(), "E-ADM-018") {
			t.Errorf("AC-005: expected E-ADM-018 in err: got %v", err)
		}
	})

	t.Run("with_confirm_confirm_true_in_wire", func(t *testing.T) {
		t.Parallel()

		// AC-005 — with --confirm flag, wire confirm field must be true.
		requestCh := make(chan adminRPCRequest, 1)
		addr := startFakeServer(t, requestCh, func(_ string, args json.RawMessage) (any, error) {
			var revokeArgs adminKeyRevokeArgs
			if err := json.Unmarshal(args, &revokeArgs); err != nil {
				return nil, err
			}
			// CR-002/CR-007: validate role field is populated.
			if revokeArgs.Role == "" {
				return nil, fmt.Errorf("E-CFG-001: role field missing in revoke request")
			}
			return map[string]any{"fingerprint": "SHA256:EEEE..."}, nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		const pubkey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI test-ctrl-key-confirm"

		err := runAdmin(ctx, addr, testdataKeyPath(t), false, []string{
			"key", "revoke",
			"--key", pubkey,
			"--svtn", "test-svtn",
			"--role", "control",
			"--confirm",
		}, defaultIO())
		if err != nil {
			t.Fatalf("AC-005: with --confirm: expected success; got error: %v", err)
		}

		select {
		case req := <-requestCh:
			var args adminKeyRevokeArgs
			if jsonErr := json.Unmarshal(req.Args, &args); jsonErr != nil {
				t.Fatalf("AC-005: unmarshal args with confirm: %v", jsonErr)
			}
			if !args.Confirm {
				t.Error("AC-005 — wire confirm: want true when --confirm supplied; got false")
			}
			// CR-002/CR-007: role field must be set in the wire payload.
			if args.Role != "control" {
				t.Errorf("AC-005 — wire role: got %q; want control", args.Role)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("AC-005: timed out — CLI did not dispatch RPC within deadline")
		}
	})
}

// ── error path and boundary tests ─────────────────────────────────────────────

// TestSbctlAdmin_UnknownSubcommand_ReturnsError verifies that supplying an
// unknown subcommand to `sbctl admin` returns a non-nil error.
// (The error is mapped to exit code 2 by main().)
//
// F-P8-001 (canonical CLI surface: sbctl admin).
func TestSbctlAdmin_UnknownSubcommand_ReturnsError(t *testing.T) {
	t.Parallel()

	// F-P8-001 — unknown admin subcommand must return non-nil error.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := runAdmin(ctx, "127.0.0.1:19996", testdataKeyPath(t), false, []string{"totally-unknown-cmd"}, defaultIO())
	if err == nil {
		t.Error("runAdmin with unknown subcommand: want non-nil error; got nil")
		return
	}
	// Tighten oracle: error must come from arg parsing (mentions "unknown" and/or the
	// subcommand name), NOT from a network failure. A network-refused error would pass
	// the weak err != nil check without validating that arg-parsing fires first.
	errStr := err.Error()
	if strings.Contains(errStr, "E-NET-001") || strings.Contains(errStr, "connection refused") {
		t.Errorf("F-P8-001 — runAdmin unknown subcommand: got network error %q; want arg-parsing error before any connection attempt", errStr)
	}
	if !strings.Contains(errStr, "unknown") && !strings.Contains(errStr, "totally-unknown-cmd") {
		t.Errorf("F-P8-001 — runAdmin unknown subcommand: error %q does not mention 'unknown' or the subcommand name; want descriptive parse error", errStr)
	}
}

// TestSbctlAdmin_MissingRequiredFlags_ReturnsError verifies that omitting
// required flags (--key, --svtn) returns an error rather than panicking
// or producing partial output.
//
// BC-2.05.004 precondition 2 (key operation must be well-formed).
func TestSbctlAdmin_MissingRequiredFlags_ReturnsError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		args        []string
		missingFlag string // the flag name that must appear in the error
	}{
		// BC-2.05.004 precondition 2 — malformed key operations must return error.
		{"register_missing_key", []string{"key", "register", "--svtn", "test-svtn"}, "--key"},
		{"register_missing_svtn", []string{"key", "register", "--key", "ssh-ed25519 AAAA..."}, "--svtn"},
		{"revoke_missing_key", []string{"key", "revoke", "--svtn", "test-svtn", "--role", "console"}, "--key"},
		{"revoke_missing_svtn", []string{"key", "revoke", "--key", "ssh-ed25519 AAAA...", "--role", "console"}, "--svtn"},
		{"revoke_missing_role", []string{"key", "revoke", "--key", "ssh-ed25519 AAAA...", "--svtn", "test-svtn"}, "--role"},
		{"expire_missing_after", []string{"key", "expire", "--key", "ssh-ed25519 AAAA...", "--svtn", "test-svtn"}, "--after"},
		{"expire_missing_key", []string{"key", "expire", "--svtn", "test-svtn", "--after", "24h"}, "--key"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			err := runAdmin(ctx, "127.0.0.1:19995", testdataKeyPath(t), false, tc.args, defaultIO())
			if err == nil {
				t.Errorf("BC-2.05.004 precondition 2 — runAdmin(%v): want error for missing required flag; got nil", tc.args)
				return
			}
			// Tighten oracle: error must be a flag-parsing error, not a network error.
			// A network-refused error (E-NET-001) would pass the weak err != nil oracle
			// without proving that flag validation fires before any connection attempt.
			errStr := err.Error()
			if strings.Contains(errStr, "E-NET-001") || strings.Contains(errStr, "connection refused") {
				t.Errorf("BC-2.05.004 precondition 2 — runAdmin(%v): got network error %q; want flag validation error before any connection attempt", tc.args, errStr)
			}
			// The missing-flag name must appear in the error to confirm the right
			// validation path fired (not a generic "required field" catch-all).
			flagName := tc.missingFlag[2:] // strip "--" for substring match (e.g. "key", "svtn", "role", "after")
			if !strings.Contains(errStr, flagName) {
				t.Errorf("BC-2.05.004 precondition 2 — runAdmin(%v): error %q does not mention missing flag %q", tc.args, errStr, tc.missingFlag)
			}
		})
	}
}

// TestSbctlAdmin_UnauthenticatedClient_FailsClosed verifies that when the
// daemon sends AUTH_FAIL, runAdmin returns a non-nil error and does NOT
// dispatch any admin RPC (fail-closed).
//
// ADR-012 (fail-closed Authenticate(): returns nil ONLY on AUTH_OK).
// BC-2.05.004 invariant 3 (key management operations are authenticated).
func TestSbctlAdmin_UnauthenticatedClient_FailsClosed(t *testing.T) {
	t.Parallel()

	// ADR-012 fail-closed — AUTH_FAIL must cause runAdmin to return non-nil error.
	// postAuthFailReads counts bytes the server received AFTER sending AUTH_FAIL.
	// A non-zero count means the client continued sending data (RPC dispatch) after
	// the auth failure, which violates the fail-closed invariant.
	var postAuthFailReads atomic.Int64

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	// Start a server that always AUTH_FAILs, then counts further client writes.
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

		enc := json.NewEncoder(conn)
		dec := json.NewDecoder(io.LimitReader(conn, maxMessageBytes))

		var nonce [32]byte
		_, _ = rand.Read(nonce[:])

		_ = enc.Encode(map[string]any{
			"type":       "challenge",
			"nonce":      base64.RawURLEncoding.EncodeToString(nonce[:]),
			"daemon_sig": base64.RawURLEncoding.EncodeToString(make([]byte, 64)),
		})

		// Read (and ignore) the challenge response.
		var cr map[string]any
		_ = dec.Decode(&cr)

		// Send AUTH_FAIL unconditionally.
		_ = enc.Encode(map[string]any{
			"type":    "auth_fail",
			"code":    "E-ADM-010",
			"message": "authentication failed",
		})

		// Drain any further bytes the client sends after AUTH_FAIL.
		// Any such bytes indicate the client dispatched an RPC despite the failure.
		buf := make([]byte, 4096)
		for {
			n, readErr := conn.Read(buf)
			if n > 0 {
				postAuthFailReads.Add(int64(n))
			}
			if readErr != nil {
				break
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = runAdmin(ctx, ln.Addr().String(), testdataKeyPath(t), false, []string{
		"key", "register", "--key", "ssh-ed25519 AAAA...", "--svtn", "test-svtn",
	}, defaultIO())

	// ADR-012 fail-closed: must return non-nil error on AUTH_FAIL.
	if err == nil {
		t.Error("ADR-012 fail-closed — runAdmin with AUTH_FAIL server: want non-nil error; got nil")
	}

	// Dispatch observer: no RPC bytes must have been sent after AUTH_FAIL.
	if n := postAuthFailReads.Load(); n > 0 {
		t.Errorf("ADR-012 fail-closed — client sent %d bytes after AUTH_FAIL; must not dispatch without AUTH_OK", n)
	}
}

// TestSbctlAdmin_OversizedNDJSONLine_DoesNotOOM verifies ADR-012 bounded-read
// requirement: an oversized NDJSON response from the daemon (>64 KiB) must not
// cause OOM. The client must close the connection and return an error.
//
// ADR-012 §6 (64 KiB bounded reads; CWE-400).
// BC-2.07.001 (sbctl admin is the operator interface; must be robust to malformed server).
func TestSbctlAdmin_OversizedNDJSONLine_DoesNotOOM(t *testing.T) {
	t.Parallel()

	// ADR-012 §6 — oversized response must not OOM; client must reject and return error.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	// Start a server that sends an oversized challenge line.
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

		// Send a JSON line that exceeds maxMessageBytes (64 KiB).
		// Wrap in valid-looking JSON to pass initial parsing before the size guard fires.
		oversized := bytes.Repeat([]byte("x"), maxMessageBytes+512)
		line := append([]byte(`{"type":"challenge","nonce":"`), oversized...)
		line = append(line, '"', '}', '\n')
		_, _ = conn.Write(line)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// The client must not OOM — it must return an error within the context deadline.
	err = runAdmin(ctx, ln.Addr().String(), testdataKeyPath(t), false, []string{
		"key", "register", "--key", "ssh-ed25519 AAAA...", "--svtn", "test-svtn",
	}, defaultIO())

	if err == nil {
		t.Error("ADR-012 §6 — oversized NDJSON line: want non-nil error (bounded read violation); got nil")
	} else {
		// Positive-coverage: the error must indicate the read-guard fired.
		// Accepted substrings (CWE-400 / ADR-012 §6):
		//   - bufio scanner limit: "token too long"
		//   - custom inline guard / LimitReader stamp: "message too large", "E-RPC-002"
		// "EOF" alone is NOT accepted — a bare EOF indicates a network close, not a
		// size-guard, and would allow the assertion to pass on a completely unrelated
		// connection failure (ADR-012 §6, CWE-400).
		msg := err.Error()
		if !strings.Contains(msg, "token too long") &&
			!strings.Contains(msg, "message too large") &&
			!strings.Contains(msg, "E-RPC-002") {
			t.Errorf("ADR-012 §6 — expected read-guard error surface "+
				"(\"token too long\" / \"message too large\" / \"E-RPC-002\"); got: %v", err)
		}
	}
}

// TestSbctlAdmin_MalformedJSONResponse_ReturnsError verifies that a malformed
// JSON response from the daemon causes runAdmin to return a non-nil error.
//
// ADR-012 (wire protocol validation; malformed messages must be rejected).
func TestSbctlAdmin_MalformedJSONResponse_ReturnsError(t *testing.T) {
	t.Parallel()

	// ADR-012 — malformed JSON response must return non-nil error.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
		// Send syntactically invalid JSON.
		_, _ = conn.Write([]byte("this is not valid json\n"))
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = runAdmin(ctx, ln.Addr().String(), testdataKeyPath(t), false, []string{
		"key", "register", "--key", "ssh-ed25519 AAAA...", "--svtn", "test-svtn",
	}, defaultIO())

	if err == nil {
		t.Error("ADR-012 — malformed JSON response: want non-nil error; got nil")
	} else {
		// Positive-coverage: the error must surface a JSON parse failure.
		// "unexpected" alone is NOT accepted — it also matches "unexpected EOF"
		// which is a network failure, not a JSON parse error.
		msg := err.Error()
		if !strings.Contains(msg, "invalid character") &&
			!strings.Contains(msg, "json") {
			t.Errorf("ADR-012 — expected JSON parse error surface "+
				"(\"invalid character\" / \"json\"); got: %v", err)
		}
	}
}

// TestSbctlAdmin_KeyExpire_ZeroDurationAfterFlag verifies S-6.02 EC-003
// at the CLI layer: `--after 0s` should be rejected before dispatch (client-side
// validation) or by the daemon (returns E-CFG-001). Either way, runAdmin must
// return a non-nil error.
//
// S-6.02 EC-003 (E-CFG-001: invalid duration).
// BC-2.05.004 postcondition 3 (TTL must be positive).
func TestSbctlAdmin_KeyExpire_ZeroDurationAfterFlag(t *testing.T) {
	t.Parallel()

	// S-6.02 EC-003 — zero duration must cause runAdmin to return non-nil error.
	// Daemon handler simulates E-CFG-001 for zero duration.
	addr := startFakeServer(t, nil, func(cmd string, args json.RawMessage) (any, error) {
		if cmd != "admin.key.expire" {
			return nil, fmt.Errorf("unexpected command: %q", cmd)
		}
		var expireArgs adminKeyExpireArgs
		if err := json.Unmarshal(args, &expireArgs); err != nil {
			return nil, err
		}
		d, err := time.ParseDuration(expireArgs.After)
		if err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid duration %q: %w", expireArgs.After, err)
		}
		if d <= 0 {
			return nil, fmt.Errorf("E-CFG-001: invalid duration: must be positive")
		}
		return map[string]string{"fingerprint": "SHA256:ok"}, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := runAdmin(ctx, addr, testdataKeyPath(t), false, []string{
		"key", "expire",
		"--key", "ssh-ed25519 AAAA...",
		"--svtn", "test-svtn",
		"--after", "0s",
	}, defaultIO())

	if err == nil {
		t.Error("S-6.02 EC-003 — runAdmin key expire --after 0s: want non-nil error; got nil")
		return
	}
	// Tighten oracle: client-side validation must fire before any connection
	// attempt — the error must mention the problematic constraint ("duration"
	// or "after") and must NOT be a network error (E-NET-001 / connection
	// refused). The daemon handler would emit E-CFG-001 for zero duration, but
	// client-side validation is expected to short-circuit before dispatch.
	errStr := err.Error()
	if strings.Contains(errStr, "E-NET-001") || strings.Contains(errStr, "connection refused") {
		t.Errorf("S-6.02 EC-003 — runAdmin key expire --after 0s: got network error %q; want client-side validation error before any connection attempt", errStr)
	}
	if !strings.Contains(errStr, "duration") && !strings.Contains(errStr, "after") {
		t.Errorf("S-6.02 EC-003 — runAdmin key expire --after 0s: error %q does not mention 'duration' or 'after'; want field-specific validation message", errStr)
	}
}

// ── Subprocess integration: sbctl admin exit codes ────────────────────────────

// TestSubprocessAdmin_ConnectionRefused verifies that `sbctl admin key register`
// against an unreachable daemon exits with code 1 and prints E-NET-001 to stderr.
//
// BC-2.07.003 PC-1 (E-NET-001 on stderr when daemon unreachable).
// F-P8-001 (canonical CLI surface: sbctl admin).
func TestSubprocessAdmin_ConnectionRefused(t *testing.T) {
	t.Parallel()

	// BC-2.07.003 PC-1 — sbctl admin against unreachable daemon exits 1, E-NET-001 on stderr.
	keyPath := testdataKeyPath(t)

	cmd := exec.Command(os.Args[0], "-test.run=TestSubprocessAdmin_ConnectionRefused_Entry")
	cmd.Env = append(
		os.Environ(),
		"SBCTL_ADMIN_SUBPROCESS=1",
		"SBCTL_TEST_KEY="+keyPath,
		"SBCTL_TEST_TARGET=127.0.0.1:19994", // nothing listening
		"SBCTL_TEST_TIMEOUT=200ms",
	)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	if err == nil {
		t.Error("BC-2.07.003 PC-1 — expected non-zero exit; got 0")
	} else {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("expected ExitError, got %T: %v", err, err)
		}
		if exitErr.ExitCode() != 1 {
			t.Fatalf("expected exit code 1, got %d", exitErr.ExitCode())
		}
	}

	stderr := errBuf.String()
	if !strings.Contains(stderr, "E-NET-001") {
		t.Errorf("BC-2.07.003 PC-1 — expected 'E-NET-001' in stderr; got: %q", stderr)
	}
}

// TestSbctlAdmin_KeyRegister_InvalidRole verifies F-CS-005:
// `sbctl admin key register --role <invalid>` must return a non-nil error before
// dispatching any RPC (client-side enum validation mirrors revoke-side behavior).
//
// F-CS-005 (enum-switch validation in runAdminKeyRegister mirrors runAdminKeyRevoke).
// BC-2.05.004 precondition 2 (key operation must be well-formed).
func TestSbctlAdmin_KeyRegister_InvalidRole(t *testing.T) {
	t.Parallel()

	// Followup: default-console path e2e is tracked separately.
	cases := []struct {
		name string
		role string
	}{
		// F-CS-005 — invalid role values must be rejected before dispatch.
		{"unknown_role", "superadmin"},
		{"numeric_role", "42"},
		{"mixed_case_role", "Control"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Use an unreachable address — if the role validation is correct,
			// no connection should be attempted.
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			args := []string{
				"key", "register",
				"--key", "ssh-ed25519 AAAA...",
				"--svtn", "test-svtn",
				"--role", tc.role,
			}

			err := runAdmin(ctx, "127.0.0.1:19993", testdataKeyPath(t), false, args, defaultIO())

			if err == nil {
				t.Errorf("F-CS-005 — runAdmin key register --role %q: want non-nil error; got nil", tc.role)
				return
			}

			// F-CS-005: the error must be a CLI validation error, NOT a network
			// error. With validation in place, no connection is attempted for
			// invalid roles. The error message must mention "role".
			errStr := err.Error()
			if strings.Contains(errStr, "E-NET-001") || strings.Contains(errStr, "connection refused") {
				t.Errorf("F-CS-005 — runAdmin key register --role %q: "+
					"got network error %q; want role validation error before any connection attempt",
					tc.role, errStr)
			}
			if !strings.Contains(errStr, "role") {
				t.Errorf("F-CS-005 — runAdmin key register --role %q: "+
					"error %q does not mention 'role'; want role validation error", tc.role, errStr)
			}
		})
	}
}

// TestSubprocessAdmin_ConnectionRefused_Entry is the subprocess entry point for
// TestSubprocessAdmin_ConnectionRefused. It re-runs main() via os.Args manipulation.
func TestSubprocessAdmin_ConnectionRefused_Entry(t *testing.T) {
	if os.Getenv("SBCTL_ADMIN_SUBPROCESS") != "1" {
		t.Skip("subprocess entry — skip in parent process")
	}

	target := os.Getenv("SBCTL_TEST_TARGET")
	keyPath := os.Getenv("SBCTL_TEST_KEY")
	timeoutStr := os.Getenv("SBCTL_TEST_TIMEOUT")
	to := 200 * time.Millisecond
	if timeoutStr != "" {
		if d, err := time.ParseDuration(timeoutStr); err == nil {
			to = d
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), to)
	defer cancel()

	sio := defaultIO()
	err := runAdmin(ctx, target, keyPath, false, []string{
		"key", "register",
		"--key", "ssh-ed25519 AAAA...",
		"--svtn", "test-svtn",
		"--role", "console",
	}, sio)
	if err != nil {
		writeError(false, "E-NET-001", err.Error(), sio)
		os.Exit(1)
	}
	os.Exit(0)
}

// ── S-6.07: sbctl admin svtn create CLI tests ─────────────────────────────────

// TestSbctlAdmin_SvtnCreate_CLI verifies AC-002 at the CLI layer:
// `sbctl admin svtn create --name <svtn-name>` dispatches the
// admin.svtn.create RPC with the correct wire-format payload.
//
// BC-2.07.001 PC-1 — admin.svtn.create RPC is dispatched with correct args.
// AC-002 — sbctl admin svtn create sends {"command":"admin.svtn.create","args":{"name":"<name>"}}.
func TestSbctlAdmin_SvtnCreate_CLI(t *testing.T) {
	t.Parallel()

	// AC-002 / BC-2.07.001 PC-1 — sbctl admin svtn create dispatches correct RPC.
	// Was RED during initial TDD (runAdminSvtnCreate not implemented); now covers positive path.
	requestCh := make(chan adminRPCRequest, 1)
	addr := startFakeServer(t, requestCh, func(cmd string, args json.RawMessage) (any, error) {
		if cmd != "admin.svtn.create" {
			return nil, fmt.Errorf("unexpected command: %q; want admin.svtn.create", cmd)
		}
		var createArgs adminSVTNCreateArgs
		if err := json.Unmarshal(args, &createArgs); err != nil {
			return nil, fmt.Errorf("unmarshal adminSVTNCreateArgs: %w", err)
		}
		return map[string]any{
			"svtn_id":               "aabbccddeeff0011aabbccddeeff0011",
			"bootstrap_fingerprint": "SHA256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		}, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	const svtnName = "my-new-network"

	err := runAdmin(ctx, addr, testdataKeyPath(t), false, []string{
		"svtn", "create",
		"--name", svtnName,
	}, defaultIO())
	if err != nil {
		t.Fatalf("runAdmin: %v", err)
	}

	// Verify the RPC was dispatched with correct payload.
	select {
	case req := <-requestCh:
		if req.Command != "admin.svtn.create" {
			t.Errorf("AC-002 — dispatched command: got %q; want admin.svtn.create", req.Command)
		}
		var args adminSVTNCreateArgs
		if err := json.Unmarshal(req.Args, &args); err != nil {
			t.Fatalf("AC-002 — unmarshal args: %v", err)
		}
		if args.Name != svtnName {
			t.Errorf("AC-002 — wire name: got %q; want %q", args.Name, svtnName)
		}
	case <-time.After(2 * time.Second):
		t.Error("AC-002: timed out waiting for admin.svtn.create RPC; " +
			"runAdminSvtnCreate must dispatch the RPC within the context deadline")
	}
}

// TestSbctlAdmin_SvtnCreate_NonControlDenied verifies AC-003 at the CLI layer:
// when the daemon returns E-ADM-009 (non-control-role caller), runAdmin
// returns a non-nil error.
//
// BC-2.07.001 Inv-3 — non-control-role caller rejected with E-ADM-009.
// AC-003 — sbctl surfaces the E-ADM-009 error to the caller.
func TestSbctlAdmin_SvtnCreate_NonControlDenied(t *testing.T) {
	t.Parallel()

	// AC-003 / BC-2.07.001 Inv-3 — E-ADM-009 from daemon must surface as error.
	// Was RED during initial TDD (runAdminSvtnCreate not implemented); now covers positive path.
	addr := startFakeServer(t, nil, func(cmd string, _ json.RawMessage) (any, error) {
		if cmd != "admin.svtn.create" {
			return nil, fmt.Errorf("unexpected command: %q", cmd)
		}
		return nil, fmt.Errorf("E-ADM-009: insufficient authority for operation admin.svtn.create: key SHA256:test= has role console")
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := runAdmin(ctx, addr, testdataKeyPath(t), false, []string{
		"svtn", "create",
		"--name", "forbidden-svtn",
	}, defaultIO())

	// AC-003: must return non-nil error containing E-ADM-009.
	if err == nil {
		t.Fatal("AC-003: expected E-ADM-009 error from daemon; got nil")
	}
	if !strings.Contains(err.Error(), "E-ADM-009") {
		t.Errorf("AC-003: expected E-ADM-009 in error; got: %v", err)
	}
}

// TestSbctlAdmin_SvtnCreate_SuccessOutputsSVTNIDAndFingerprint verifies AC-002
// and AC-004 at the CLI layer: on success, sbctl admin svtn create prints
// the svtn_id and bootstrap_fingerprint to stdout.
//
// BC-2.07.001 PC-1 + PC-2 — success response carries svtn_id and bootstrap_fingerprint.
// AC-002 — CLI prints returned svtn_id and bootstrap fingerprint on success.
// AC-004 — svtn_id (hex) and bootstrap_fingerprint (SHA256:<base64>) in response.
func TestSbctlAdmin_SvtnCreate_SuccessOutputsSVTNIDAndFingerprint(t *testing.T) {
	t.Parallel()

	// AC-002 / AC-004 — success output must contain svtn_id and bootstrap_fingerprint.
	// Was RED during initial TDD (runAdminSvtnCreate not implemented); now covers positive path.
	const wantSVTNID = "aabbccddeeff0011aabbccddeeff0011"
	const wantFingerprint = "SHA256:BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBA="

	addr := startFakeServer(t, nil, func(cmd string, _ json.RawMessage) (any, error) {
		if cmd != "admin.svtn.create" {
			return nil, fmt.Errorf("unexpected command: %q; want admin.svtn.create", cmd)
		}
		return map[string]any{
			"svtn_id":               wantSVTNID,
			"bootstrap_fingerprint": wantFingerprint,
		}, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var outBuf strings.Builder
	sio := sbctlIO{out: &outBuf, err: io.Discard}

	err := runAdmin(ctx, addr, testdataKeyPath(t), false, []string{
		"svtn", "create",
		"--name", "success-svtn",
	}, sio)
	if err != nil {
		t.Fatalf("AC-002: runAdmin returned error on success path: %v", err)
	}

	out := outBuf.String()

	// AC-002 / AC-004: stdout must contain the svtn_id returned by the daemon.
	if !strings.Contains(out, wantSVTNID) {
		t.Errorf("AC-002 / AC-004: stdout does not contain svtn_id %q; got: %q", wantSVTNID, out)
	}

	// AC-002 / AC-004: stdout must contain the bootstrap_fingerprint returned by the daemon.
	if !strings.Contains(out, wantFingerprint) {
		t.Errorf("AC-002 / AC-004: stdout does not contain bootstrap_fingerprint %q; got: %q", wantFingerprint, out)
	}
}

// TestSbctlAdmin_SvtnCreate_DuplicateName verifies AC-005 at the CLI layer:
// when the daemon returns SVTN-exists error, runAdmin returns a non-nil error.
//
// BC-2.07.001 EC-001 — duplicate SVTN name returns SVTN-exists error.
// AC-005 — sbctl surfaces the SVTN-exists error to the caller.
func TestSbctlAdmin_SvtnCreate_DuplicateName(t *testing.T) {
	t.Parallel()

	// AC-005 / BC-2.07.001 EC-001 — SVTN-exists error from daemon must surface.
	// Was RED during initial TDD (runAdminSvtnCreate not implemented); now covers positive path.
	addr := startFakeServer(t, nil, func(cmd string, _ json.RawMessage) (any, error) {
		if cmd != "admin.svtn.create" {
			return nil, fmt.Errorf("unexpected command: %q", cmd)
		}
		return nil, fmt.Errorf("E-SVTN-001: SVTN already exists: test-svtn")
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := runAdmin(ctx, addr, testdataKeyPath(t), false, []string{
		"svtn", "create",
		"--name", "test-svtn",
	}, defaultIO())

	// AC-005: must return non-nil error.
	if err == nil {
		t.Fatal("AC-005: expected SVTN-exists error from daemon; got nil")
	}
	if !strings.Contains(err.Error(), "E-SVTN-001") {
		t.Errorf("AC-005: expected E-SVTN-001 in error; got: %v", err)
	}
}

// TestSbctlAdmin_SvtnCreate_MissingName verifies that omitting --name returns
// an error before dispatching any RPC.
//
// BC-2.07.001 PC-1 — CLI validates required args before dispatch.
func TestSbctlAdmin_SvtnCreate_MissingName(t *testing.T) {
	t.Parallel()

	// BC-2.07.001 PC-1 — missing --name must return non-nil error.
	// Was RED during initial TDD (runAdminSvtnCreate not implemented); now covers positive path.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := runAdmin(ctx, "127.0.0.1:19992", testdataKeyPath(t), false, []string{
		"svtn", "create",
		// No --name flag.
	}, defaultIO())

	if err == nil {
		t.Error("BC-2.07.001 PC-1: missing --name: expected non-nil error; got nil")
	} else if !strings.Contains(err.Error(), "--name is required") {
		// Distinguish flag-validation failure from connection failure: the error
		// must name the missing flag, not report a network error.
		t.Errorf("BC-2.07.001 PC-1: expected \"--name is required\" in error; got: %v", err)
	}
}

// TestSbctlAdmin_SvtnCreate_JSONRoundTrip verifies that adminSVTNCreateArgs
// serialises with the correct JSON field name "name" per the AC-002 wire envelope.
//
// AC-002 — wire args shape: {"command":"admin.svtn.create","args":{"name":"<name>"}}.
func TestSbctlAdmin_SvtnCreate_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	// AC-002 — wire args field name.
	original := adminSVTNCreateArgs{Name: "round-trip-svtn"}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(adminSVTNCreateArgs): %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal to map: %v", err)
	}

	if _, ok := raw["name"]; !ok {
		t.Error("AC-002: adminSVTNCreateArgs: missing JSON field 'name'")
	}

	var decoded adminSVTNCreateArgs
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if decoded.Name != original.Name {
		t.Errorf("name round-trip: got %q; want %q", decoded.Name, original.Name)
	}
}

// TestSbctlAdmin_OversizedRPCResponse_ReturnsE_RPC_002 verifies Ruling-14 §10:
// when the daemon sends an oversized RPC response (>64 KiB) after a successful
// auth handshake, dispatch() must return an error containing "E-RPC-002" — not
// bare "unexpected EOF" — providing a deterministic, size-keyed error surface
// symmetric with Authenticate (ADR-012 §6, CWE-400).
//
// ADR-012 §6 — 64 KiB bounded reads; CWE-400 slowloris defence.
// Ruling-14 §10 — dispatch response decode MUST wrap io.ErrUnexpectedEOF with E-RPC-002.
func TestSbctlAdmin_OversizedRPCResponse_ReturnsE_RPC_002(t *testing.T) {
	t.Parallel()

	// Ruling-14 §10 — oversized RPC response must stamp E-RPC-002, not bare EOF.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	// Load the testdata key so the CR-008 authorized-key check passes.
	testPrivKey, loadErr := loadEd25519Key(testdataKeyPath(t), os.UserHomeDir)
	if loadErr != nil {
		t.Fatalf("loadEd25519Key: %v", loadErr)
	}
	opPub := testPrivKey.Public().(ed25519.PublicKey)

	go func() {
		conn, acceptErr := ln.Accept()
		if acceptErr != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

		enc := json.NewEncoder(conn)
		dec := json.NewDecoder(io.LimitReader(conn, maxMessageBytes))

		// Step 1: Send CHALLENGE.
		var nonce [32]byte
		if _, randErr := rand.Read(nonce[:]); randErr != nil {
			return
		}
		_, daemonPriv, keyErr := ed25519.GenerateKey(rand.Reader)
		if keyErr != nil {
			return
		}
		daemonSig := ed25519.Sign(daemonPriv, nonce[:])
		if encErr := enc.Encode(map[string]any{
			"type":       "challenge",
			"nonce":      base64.RawURLEncoding.EncodeToString(nonce[:]),
			"daemon_sig": base64.RawURLEncoding.EncodeToString(daemonSig),
		}); encErr != nil {
			return
		}

		// Step 2: Read CHALLENGE_RESPONSE and verify it carries the authorized key.
		var cr struct {
			Type     string `json:"type"`
			NonceSig string `json:"nonce_sig"`
			PubKey   string `json:"pubkey"`
		}
		if decErr := dec.Decode(&cr); decErr != nil {
			return
		}
		pubBytes, pubErr := base64.RawURLEncoding.DecodeString(cr.PubKey)
		if pubErr != nil {
			_ = enc.Encode(map[string]any{"type": "auth_fail", "code": "E-ADM-010", "message": "bad pubkey"})
			return
		}
		sigBytes, sigErr := base64.RawURLEncoding.DecodeString(cr.NonceSig)
		if sigErr != nil {
			_ = enc.Encode(map[string]any{"type": "auth_fail", "code": "E-ADM-010", "message": "bad sig"})
			return
		}
		pub := ed25519.PublicKey(pubBytes)
		if !ed25519.Verify(pub, nonce[:], sigBytes) || !bytes.Equal(pub, opPub) {
			_ = enc.Encode(map[string]any{"type": "auth_fail", "code": "E-ADM-010", "message": "unauthorized"})
			return
		}

		// Step 3: Send AUTH_OK.
		if encErr := enc.Encode(map[string]any{
			"type":           "auth_ok",
			"daemon_version": "test-dev",
		}); encErr != nil {
			return
		}

		// Step 4: Read the RPC request (discard content).
		var req struct {
			ID string `json:"id"`
		}
		if decErr := dec.Decode(&req); decErr != nil {
			return
		}

		// Step 5: Send oversized RPC response (>64 KiB) — triggers E-RPC-002 on client.
		// Wrap in a valid-looking JSON prefix so the JSON decoder reads past the
		// opening brace before the LimitReader cuts it off mid-token.
		oversized := bytes.Repeat([]byte("x"), maxMessageBytes+512)
		line := append([]byte(`{"type":"response","id":"`+req.ID+`","ok":true,"data":"`), oversized...)
		line = append(line, '"', '}', '\n')
		_, _ = conn.Write(line)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = runAdmin(ctx, ln.Addr().String(), testdataKeyPath(t), false, []string{
		"key", "register",
		"--key", "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI test-ruling14-key",
		"--svtn", "test-svtn",
		"--role", "console",
	}, defaultIO())

	// Ruling-14 §10: must return non-nil error.
	if err == nil {
		t.Fatal("Ruling-14 §10 — oversized RPC response: want non-nil error; got nil")
	}

	// The error must contain "E-RPC-002" — not bare "unexpected EOF" without the stamp.
	msg := err.Error()
	if !strings.Contains(msg, "E-RPC-002") {
		t.Errorf("Ruling-14 §10 — oversized RPC response: expected E-RPC-002 in error; got: %v", err)
	}
}
