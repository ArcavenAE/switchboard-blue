// Package main implements the sbctl operator CLI per ADR-012 and BC-2.07.002/003.
// Purity classification (ARCH-09): effectful-boundary — owns network I/O, file I/O,
// and OS interaction.
package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// maxMessageBytes is the bounded read limit for every management socket read
// (client side). Mirrors internal/mgmt.MaxMessageBytes = 64 KiB per ADR-012 §6.
const maxMessageBytes = 1 << 16 // 64 KiB

// handshakeTimeout is the fallback deadline for Authenticate() when the context
// carries no deadline. Prevents indefinite hangs (CWE-400 slowloris, Ruling 2).
const handshakeTimeout = 10 * time.Second

// challengeMsg is the CHALLENGE message received from the daemon (ADR-012 step 2).
type challengeMsg struct {
	Type      string `json:"type"`
	Nonce     string `json:"nonce"`
	DaemonSig string `json:"daemon_sig"`
}

// challengeResponseMsg is the CHALLENGE_RESPONSE message sent to the daemon
// (ADR-012 step 3). The operator private key is NEVER transmitted — only the
// public key (32 bytes) and the nonce signature go over the wire (DI-002).
type challengeResponseMsg struct {
	Type     string `json:"type"`
	NonceSig string `json:"nonce_sig"`
	PubKey   string `json:"pubkey"`
}

// authResultMsg is the AUTH_OK or AUTH_FAIL message received from the daemon
// (ADR-012 steps 5a/5b).
type authResultMsg struct {
	Type          string `json:"type"`
	DaemonVersion string `json:"daemon_version,omitempty"`
	Code          string `json:"code,omitempty"`
	Message       string `json:"message,omitempty"`
}

// rpcRequestMsg is an authenticated RPC request envelope (ADR-012 step 6).
type rpcRequestMsg struct {
	Type    string `json:"type"`
	ID      string `json:"id"`
	Command string `json:"command"`
	Args    any    `json:"args"`
}

// rpcResponseMsg is an authenticated RPC response envelope (ADR-012 step 6).
type rpcResponseMsg struct {
	Type  string          `json:"type"`
	ID    string          `json:"id"`
	OK    bool            `json:"ok"`
	Error *errorDetail    `json:"error"`
	Data  json.RawMessage `json:"data"`
}

// errorDetail carries a structured error code and message in the JSON envelope
// (interface-definitions.md §JSON Output Schema).
type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   any    `json:"field"`
}

// jsonEnvelope is the outer JSON output schema for sbctl output on --json
// (interface-definitions.md §JSON Output Schema).
type jsonEnvelope struct {
	OK    bool            `json:"ok"`
	Error *errorDetail    `json:"error"`
	Data  json.RawMessage `json:"data"`
}

// newSuccessEnvelope returns a JSON envelope for a successful response.
func newSuccessEnvelope(data json.RawMessage) jsonEnvelope {
	return jsonEnvelope{OK: true, Error: nil, Data: data}
}

// newErrorEnvelope returns a JSON envelope for an error response.
func newErrorEnvelope(code, message string) jsonEnvelope {
	return jsonEnvelope{OK: false, Error: &errorDetail{Code: code, Message: message}, Data: nil}
}

// loadEd25519Key loads an OpenSSH-format Ed25519 private key from path.
// The file is read with io.LimitReader bounded at 64 KiB (ADR-012 §6, CWE-400).
// The private key is extracted as crypto/ed25519.PrivateKey via golang.org/x/crypto/ssh.
// The key is never serialized, logged, or transmitted (DI-002).
//
// homeDir is the home-directory lookup injected by the caller for tilde expansion
// (BC-2.07.003 EC-007 + Precondition 3). Production callers pass os.UserHomeDir;
// tests pass a per-call closure — no shared package-global is mutated, so parallel
// tests are safe under -race.
//
// Tilde expansion rules:
//   - "~/" or exactly "~" prefix: expanded via homeDir() before file-open.
//   - homeDir() error → E-CFG-010 with the ORIGINAL path in the message.
//   - expansion ok but file unreadable → E-CFG-010 with the EXPANDED path.
//   - "~username" (other-user) is out of scope; treated as a literal path.
func loadEd25519Key(path string, homeDir func() (string, error)) (ed25519.PrivateKey, error) {
	originalPath := path

	// Expand "~" or "~/" prefix only (not "~username"); BC-2.07.003 EC-007.
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := homeDir()
		if err != nil {
			// sub-case (a): homeDir error → original path in message.
			return nil, fmt.Errorf("key load failed: %s: home directory unavailable: %w", originalPath, err)
		}
		if path == "~" {
			path = home
		} else {
			// path starts with "~/" — replace leading "~" with home.
			path = home + path[1:]
		}
	}

	f, err := os.Open(path)
	if err != nil {
		// sub-case (b): expansion ok but file unreadable → expanded path in message.
		return nil, fmt.Errorf("key load failed: %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	// io.LimitReader bounds the read at maxMessageBytes (CWE-400; ADR-012 §6).
	// Read one byte beyond the limit to detect oversized files.
	raw, err := io.ReadAll(io.LimitReader(f, maxMessageBytes+1))
	if err != nil {
		return nil, fmt.Errorf("key load failed: %s: %w", path, err)
	}
	if len(raw) > maxMessageBytes {
		return nil, fmt.Errorf("key load failed: %s: file exceeds 64 KiB limit", path)
	}

	// ParseRawPrivateKey returns a *crypto.PrivateKey (e.g. *ed25519.PrivateKey).
	rawKey, err := ssh.ParseRawPrivateKey(raw)
	if err != nil {
		return nil, fmt.Errorf("key load failed: %s: %w", path, err)
	}

	// ssh.ParseRawPrivateKey returns *ed25519.PrivateKey for Ed25519 keys.
	edPrivPtr, ok := rawKey.(*ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key load failed: %s: not an Ed25519 private key (got %T)", path, rawKey)
	}
	return *edPrivPtr, nil
}

// Authenticate performs the ADR-012 client-side challenge-response handshake on conn.
// It is fail-closed: returns nil ONLY if AUTH_OK was received and decoded successfully.
// Any other outcome — connection error, malformed message, AUTH_FAIL, truncated stream,
// oversized response — returns a non-nil error.
//
// context.Context is always the first parameter (go.md rule 7; Ruling 2).
// The read deadline is derived from the context:
//  1. If ctx has a deadline, use it; else fall back to handshakeTimeout (10s).
//     conn.SetReadDeadline is called so the call self-bounds regardless of caller.
//
// Steps (ADR-012 §Authenticate() FAIL-CLOSED Contract):
//  1. Set read deadline from ctx (CWE-400 slowloris; Ruling 2).
//  2. Read CHALLENGE via json.NewDecoder(io.LimitReader(conn, maxMessageBytes)).
//  3. Decode nonce_bytes from base64url nonce field; error if absent or invalid.
//  4. Sign: nonce_sig = ed25519.Sign(privKey, nonce_bytes).
//     NOTE: daemon_sig is decoded above but NOT verified in MVP.
//     Trust-on-first-use deferral per S-6.03 AC-002 and ARCH-12 §Authenticate()
//     FAIL-CLOSED Contract step 4 note. Verification deferred to post-MVP hardening.
//  5. Send CHALLENGE_RESPONSE with nonce_sig and pubkey (32-byte Ed25519 public key).
//  6. Read AUTH_OK or AUTH_FAIL via json.NewDecoder(io.LimitReader(conn, maxMessageBytes)).
//  7. Return nil ONLY on AUTH_OK; return non-nil error for all other outcomes.
func Authenticate(ctx context.Context, conn net.Conn, privKey ed25519.PrivateKey) error {
	// Step 1: derive read deadline from context (Ruling 2, CWE-400 slowloris).
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().UTC().Add(handshakeTimeout)
	}
	if err := conn.SetReadDeadline(deadline); err != nil {
		return fmt.Errorf("set read deadline: %w", err)
	}

	// Step 2: Read CHALLENGE message with bounded read (CWE-400; ADR-012 §6).
	var challenge challengeMsg
	if err := json.NewDecoder(io.LimitReader(conn, maxMessageBytes)).Decode(&challenge); err != nil {
		return fmt.Errorf("read challenge: %w", err)
	}

	// Step 3: Decode nonce from base64url; must be exactly 32 bytes.
	if challenge.Nonce == "" {
		return fmt.Errorf("challenge missing nonce field")
	}
	nonceBytes, err := base64.RawURLEncoding.DecodeString(challenge.Nonce)
	if err != nil {
		return fmt.Errorf("decode challenge nonce: %w", err)
	}
	if len(nonceBytes) != 32 {
		return fmt.Errorf("challenge nonce must be 32 bytes, got %d", len(nonceBytes))
	}

	// Step 4: Sign the nonce with the operator private key.
	// daemon_sig is decoded (field present in challengeMsg) but NOT verified in MVP.
	// Trust-on-first-use deferral: S-6.03 AC-002; ARCH-12 §Authenticate() step 4 note.
	// Post-MVP: verify daemon_sig against the daemon's public key from a trust store.
	nonceSig := ed25519.Sign(privKey, nonceBytes)

	// Step 5: Send CHALLENGE_RESPONSE. The private key is NEVER transmitted —
	// only the 32-byte public key goes over the wire (DI-002).
	pubKey := privKey.Public().(ed25519.PublicKey)
	resp := challengeResponseMsg{
		Type:     "challenge_response",
		NonceSig: base64.RawURLEncoding.EncodeToString(nonceSig),
		PubKey:   base64.RawURLEncoding.EncodeToString(pubKey),
	}
	if err := json.NewEncoder(conn).Encode(resp); err != nil {
		return fmt.Errorf("send challenge response: %w", err)
	}

	// Step 6: Read AUTH_OK or AUTH_FAIL with bounded read (CWE-400; ADR-012 §6).
	var result authResultMsg
	if err := json.NewDecoder(io.LimitReader(conn, maxMessageBytes)).Decode(&result); err != nil {
		return fmt.Errorf("read auth result: %w", err)
	}

	// Step 7: Return nil ONLY on AUTH_OK. All other outcomes are errors.
	switch result.Type {
	case "auth_ok":
		return nil
	case "auth_fail":
		return fmt.Errorf("E-ADM-010: authentication failed")
	default:
		return fmt.Errorf("unexpected auth response type: %q", result.Type)
	}
}

// dispatch sends a single authenticated RPC request over conn and returns the raw
// response data. conn MUST already be authenticated (Authenticate called and returned nil).
// command is the RPC command name (e.g. "router.status"). args is marshaled into the
// request envelope. Returns the raw data field from the response envelope.
// On server error (ok:false) or decode failure: returns E-RPC-001 error
// (BC-2.07.003 EC-006; Ruling 5).
//
// ctx is always the first parameter (go.md rule 7; AC-011 Ruling V).
// TODO (Ruling V GREEN): derive conn.SetReadDeadline from ctx before response decode.
// TODO (Ruling U GREEN): validate resp.Type == "response" before checking resp.OK.
// TODO (Ruling X GREEN): generate non-constant per-call req.ID; verify resp.ID == req.ID.
func dispatch(ctx context.Context, conn net.Conn, command string, args any) (json.RawMessage, error) {
	req := rpcRequestMsg{
		Type:    "request",
		ID:      "1",
		Command: command,
		Args:    args,
	}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return nil, fmt.Errorf("rpc failed: %s: send request: %w", command, err)
	}

	var resp rpcResponseMsg
	if err := json.NewDecoder(io.LimitReader(conn, maxMessageBytes)).Decode(&resp); err != nil {
		return nil, fmt.Errorf("rpc failed: %s: decode response: %w", command, err)
	}

	if !resp.OK {
		reason := "server returned ok:false"
		if resp.Error != nil && resp.Error.Message != "" {
			reason = resp.Error.Message
		}
		return nil, fmt.Errorf("rpc failed: %s: %s", command, reason)
	}

	return resp.Data, nil
}

// connectAndRun dials the daemon, authenticates, dispatches command with args,
// and writes the result to stdout (or stderr on failure). It is the common
// execution path for all subcommands.
//
// Returns error — never calls os.Exit (go.md rule; AC-009). Only main() maps
// errors to exit codes.
//
// Error taxonomy (BC-2.07.003 Invariant 4; Ruling 5):
//   - Key load failure (before dial): E-CFG-010 "key load failed: <path>: <reason>"
//   - Dial failure (daemon unreachable): E-NET-001 "daemon unreachable: <target>: <reason>"
//   - Auth failure (AUTH_FAIL): E-ADM-010 "authentication failed"
//   - RPC dispatch failure (post-AUTH_OK): E-RPC-001 "rpc failed: <command>: <reason>"
//
//nolint:unparam // cmdArgs is always nil in current stubs; callers will vary after S-6.02/S-5.02
func connectAndRun(ctx context.Context, target, keyPath string, useJSON bool, command string, cmdArgs any) error {
	// Key load and validation BEFORE any dial (BC-2.07.003 EC-005; Ruling 5).
	// os.UserHomeDir is the real home-directory lookup; tests inject per-call.
	privKey, err := loadEd25519Key(keyPath, os.UserHomeDir)
	if err != nil {
		writeError(useJSON, "E-CFG-010", err.Error())
		return err
	}

	// Single timeout budget: context carries the deadline, threading through
	// dial + Authenticate + dispatch so total wall-clock honors --timeout once
	// (defect E: avoid double-counting).
	var conn net.Conn
	if len(target) > 0 && target[0] == '/' {
		conn, err = (&net.Dialer{}).DialContext(ctx, "unix", target)
	} else {
		conn, err = (&net.Dialer{}).DialContext(ctx, "tcp", target)
	}
	if err != nil {
		msg := fmt.Sprintf("daemon unreachable: %s: %s", target, err)
		writeError(useJSON, "E-NET-001", msg)
		return fmt.Errorf("E-NET-001: %s", msg)
	}
	defer func() { _ = conn.Close() }()

	if err = Authenticate(ctx, conn, privKey); err != nil {
		// BC-2.07.003 Inv-2: a timeout during auth is treated as "unreachable" —
		// the daemon failed to respond within the budget. Report E-NET-001.
		// AUTH_FAIL from the daemon reports E-ADM-010 (BC-2.07.002 PC-4).
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			msg := fmt.Sprintf("daemon unreachable: %s: connection timed out", target)
			writeError(useJSON, "E-NET-001", msg)
			return fmt.Errorf("E-NET-001: %s", msg)
		}
		writeError(useJSON, "E-ADM-010", "authentication failed")
		return err
	}

	data, err := dispatch(ctx, conn, command, cmdArgs)
	if err != nil {
		writeError(useJSON, "E-RPC-001", err.Error())
		return err
	}

	writeSuccess(useJSON, data)
	return nil
}
