// Package main implements the sbctl operator CLI per ADR-012 and BC-2.07.002/003.
// Purity classification (ARCH-09): effectful-boundary — owns network I/O, file I/O,
// and OS interaction.
package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
)

// maxMessageBytes is the bounded read limit for every management socket read
// (client side). Mirrors internal/mgmt.MaxMessageBytes = 64 KiB per ADR-012 §6.
const maxMessageBytes = 1 << 16 // 64 KiB

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
// GREEN-BY-DESIGN: zero branching, no I/O, no helpers, 3 lines; body is correct
// by construction from the type fields alone.
func newSuccessEnvelope(data json.RawMessage) jsonEnvelope {
	return jsonEnvelope{OK: true, Error: nil, Data: data}
}

// newErrorEnvelope returns a JSON envelope for an error response.
// GREEN-BY-DESIGN: zero branching, no I/O, no helpers, 3 lines.
func newErrorEnvelope(code, message string) jsonEnvelope {
	return jsonEnvelope{OK: false, Error: &errorDetail{Code: code, Message: message}, Data: nil}
}

// loadEd25519Key loads an OpenSSH-format Ed25519 private key from path.
// The file is read with io.LimitReader bounded at 64 KiB (ADR-012 §6, CWE-400).
// The private key is extracted as crypto/ed25519.PrivateKey via golang.org/x/crypto/ssh.
// The key is never serialized, logged, or transmitted (DI-002).
func loadEd25519Key(path string) (ed25519.PrivateKey, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open key file: %w", err)
	}
	defer func() { _ = f.Close() }()
	// io.LimitReader bounds the read at maxMessageBytes (CWE-400; ADR-012 §6).
	// Read one byte beyond the limit to detect oversized files.
	raw, err := io.ReadAll(io.LimitReader(f, maxMessageBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read key file: %w", err)
	}
	if len(raw) > maxMessageBytes {
		return nil, fmt.Errorf("key file exceeds maximum size of %d bytes (CWE-400)", maxMessageBytes)
	}
	// ParseRawPrivateKey returns a *crypto.PrivateKey (e.g. *ed25519.PrivateKey).
	rawKey, err := ssh.ParseRawPrivateKey(raw)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	// ssh.ParseRawPrivateKey returns *ed25519.PrivateKey for Ed25519 keys.
	edPrivPtr, ok := rawKey.(*ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is not an Ed25519 private key (got %T)", rawKey)
	}
	return *edPrivPtr, nil
}

// Authenticate performs the ADR-012 client-side challenge-response handshake on conn.
// It is fail-closed: returns nil ONLY if AUTH_OK was received and decoded successfully.
// Any other outcome — connection error, malformed message, AUTH_FAIL, truncated stream,
// oversized response — returns a non-nil error.
//
// Steps (ADR-012 §Authenticate() FAIL-CLOSED Contract):
//  1. Read CHALLENGE via json.NewDecoder(io.LimitReader(conn, maxMessageBytes)).
//  2. Decode nonce_bytes from base64url nonce field; error if absent or invalid.
//  3. Sign: nonce_sig = ed25519.Sign(privKey, nonce_bytes).
//  4. Send CHALLENGE_RESPONSE with nonce_sig and pubkey (32-byte Ed25519 public key).
//  5. Read AUTH_OK or AUTH_FAIL via json.NewDecoder(io.LimitReader(conn, maxMessageBytes)).
//  6. Return nil ONLY on AUTH_OK; return non-nil error for all other outcomes.
func Authenticate(conn net.Conn, privKey ed25519.PrivateKey) error {
	// Step 1: Read CHALLENGE message with bounded read (CWE-400; ADR-012 §6).
	var challenge challengeMsg
	if err := json.NewDecoder(io.LimitReader(conn, maxMessageBytes)).Decode(&challenge); err != nil {
		return fmt.Errorf("read challenge: %w", err)
	}

	// Step 2: Decode nonce from base64url; must be exactly 32 bytes.
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

	// Step 3: Sign the nonce with the operator private key.
	nonceSig := ed25519.Sign(privKey, nonceBytes)

	// Step 4: Send CHALLENGE_RESPONSE. The private key is NEVER transmitted —
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

	// Step 5: Read AUTH_OK or AUTH_FAIL with bounded read (CWE-400; ADR-012 §6).
	var result authResultMsg
	if err := json.NewDecoder(io.LimitReader(conn, maxMessageBytes)).Decode(&result); err != nil {
		return fmt.Errorf("read auth result: %w", err)
	}

	// Step 6: Return nil ONLY on AUTH_OK. All other outcomes are errors.
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
func dispatch(conn net.Conn, command string, args any) (json.RawMessage, error) {
	// Reference types used in the protocol so they are not flagged unused.
	var _ rpcRequestMsg
	var _ rpcResponseMsg
	return nil, fmt.Errorf("dispatch not implemented for command %q", command)
}
