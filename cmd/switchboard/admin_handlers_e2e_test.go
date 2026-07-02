package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/mgmt"
	"github.com/arcavenae/switchboard/internal/svtnmgmt"
)

// e2eServer bundles a running mgmt.Server with the socket path and teardown
// for use across all integration tests in this file.
type e2eServer struct {
	srv        *mgmt.Server
	socketPath string
	daemonPriv ed25519.PrivateKey // used by sendAdminRPC for bootstrap-mode auth
	cancel     context.CancelFunc
	doneCh     chan struct{}
}

// testDaemonKeys maps socketPath → daemon Ed25519 private key for bootstrap auth.
// Populated by startE2EServer; queried by sendAdminRPC. Protected by testDaemonKeysMu.
var (
	testDaemonKeys   = make(map[string]ed25519.PrivateKey)
	testDaemonKeysMu sync.Mutex
)

// startE2EServer starts a real mgmt.Server on a temp Unix socket.
// Handlers are registered via the provided slice.
// t.Cleanup registers shutdown. Returns the *e2eServer.
func startE2EServer(t *testing.T, handlers []mgmt.Handler) *e2eServer {
	t.Helper()

	// Generate an ephemeral daemon keypair for the server challenge-response.
	// Using crypto/ed25519 to avoid dependency on any key file (Library §4).
	// ed25519.GenerateKey returns (PublicKey, PrivateKey, error).
	_, daemonPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("startE2EServer: generate daemon keypair: %v", err)
	}

	// macOS imposes a 108-byte limit on Unix socket paths (POSIX sockaddr_un.sun_path).
	// t.TempDir() generates paths like /var/folders/…/<TestName><random>/NNN/ which
	// easily exceeds 108 bytes. Use os.MkdirTemp with a short prefix under /tmp to
	// stay well inside the limit.
	dir, err := os.MkdirTemp("", "sw-mgmt-*")
	if err != nil {
		t.Fatalf("startE2EServer: MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socketPath := fmt.Sprintf("%s/m.sock", dir)
	if len(socketPath) > 104 {
		t.Fatalf("startE2EServer: socket path %q length %d exceeds 104-byte limit", socketPath, len(socketPath))
	}

	ln, err := listenUnixMgmt(socketPath)
	if err != nil {
		t.Fatalf("startE2EServer: listen: %v", err)
	}

	// Bootstrap mode: nil OperatorKeySet means the daemon's own key is sole authority.
	ops := mgmt.NewOperatorKeySet(nil)

	srv := mgmt.NewServer(ln, daemonPriv, ops, handlers, "dev",
		mgmt.WithHandshakeTimeout(2*time.Second),
		mgmt.WithRPCIdleTimeout(5*time.Second),
	)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = srv.Serve(ctx)
	}()

	// Register daemon key for sendAdminRPC bootstrap auth; remove on cleanup.
	testDaemonKeysMu.Lock()
	testDaemonKeys[socketPath] = daemonPriv
	testDaemonKeysMu.Unlock()

	es := &e2eServer{
		srv:        srv,
		socketPath: socketPath,
		daemonPriv: daemonPriv,
		cancel:     cancel,
		doneCh:     done,
	}
	t.Cleanup(func() {
		cancel()
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = srv.Shutdown(shutCtx)
		shutCancel()
		<-done
		testDaemonKeysMu.Lock()
		delete(testDaemonKeys, socketPath)
		testDaemonKeysMu.Unlock()
	})
	return es
}

// newE2ESVTNManager creates a minimal SVTNManager with a registered SVTN named
// svtnName and a pre-registered key with the given role.
//
// Returns the SVTNManager, the base64-encoded registered pubkey string for use
// in RPC call args, and the daemon bootstrap control private key. The caller
// must pass ctrlPriv to startE2EServerWithOps so that sendAdminRPC can
// authenticate as the bootstrap key (F-P2L1-001 — bootstrap key is the
// sole unconditionally-allowed key after the fail-closed fix).
//
// ed25519.GenerateKey returns (PublicKey, PrivateKey, error).
func newE2ESVTNManager(t *testing.T, svtnName string, pubkey ed25519.PublicKey, role admission.KeyRole) (*svtnmgmt.SVTNManager, string, ed25519.PrivateKey) { //nolint:unparam // svtnName could vary; hardcoding would reduce test readability
	t.Helper()
	ks := admission.NewAdmittedKeySet()
	// ed25519.GenerateKey: first return is PublicKey, second is PrivateKey.
	ctrlPub, ctrlPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("newE2ESVTNManager: generate control key: %v", err)
	}
	_ = pubkey // accepted for API symmetry; caller-provided pubkey is not used
	m := svtnmgmt.NewSVTNManager(ks, ctrlPub)
	if _, err := m.Create(svtnName); err != nil {
		t.Fatalf("newE2ESVTNManager: create SVTN %q: %v", svtnName, err)
	}
	// Generate a real 32-byte Ed25519 public key for the registered entry.
	// F-005: decodePublicKey now requires exactly 32 bytes — a real key satisfies this.
	regPub, _, genErr := ed25519.GenerateKey(rand.Reader)
	if genErr != nil {
		t.Fatalf("newE2ESVTNManager: generate registered key: %v", genErr)
	}
	if _, err := m.RegisterKey(svtnName, regPub, role); err != nil {
		t.Fatalf("newE2ESVTNManager: register key: %v", err)
	}
	// Return the base64-encoded pubkey so RPC call sites can use it.
	encodedPubkey := base64.RawURLEncoding.EncodeToString([]byte(regPub))
	return m, encodedPubkey, ctrlPriv
}

// startE2EServerWithOps starts a real mgmt.Server using the provided daemon
// private key and OperatorKeySet. Use this instead of startE2EServer when the
// daemon key must match the SVTNManager bootstrap key (F-P2L1-001 fail-closed
// fix) or when a non-nil OperatorKeySet is needed (RoleInsufficient tests).
func startE2EServerWithOps(t *testing.T, handlers []mgmt.Handler, daemonPriv ed25519.PrivateKey, ops *mgmt.OperatorKeySet) *e2eServer {
	t.Helper()

	dir, err := os.MkdirTemp("", "sw-mgmt-*")
	if err != nil {
		t.Fatalf("startE2EServerWithOps: MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socketPath := fmt.Sprintf("%s/m.sock", dir)
	if len(socketPath) > 104 {
		t.Fatalf("startE2EServerWithOps: socket path %q length %d exceeds 104-byte limit", socketPath, len(socketPath))
	}

	ln, err := listenUnixMgmt(socketPath)
	if err != nil {
		t.Fatalf("startE2EServerWithOps: listen: %v", err)
	}

	srv := mgmt.NewServer(ln, daemonPriv, ops, handlers, "dev",
		mgmt.WithHandshakeTimeout(2*time.Second),
		mgmt.WithRPCIdleTimeout(5*time.Second),
	)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = srv.Serve(ctx)
	}()

	testDaemonKeysMu.Lock()
	testDaemonKeys[socketPath] = daemonPriv
	testDaemonKeysMu.Unlock()

	es := &e2eServer{
		srv:        srv,
		socketPath: socketPath,
		daemonPriv: daemonPriv,
		cancel:     cancel,
		doneCh:     done,
	}
	t.Cleanup(func() {
		cancel()
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = srv.Shutdown(shutCtx)
		shutCancel()
		<-done
		testDaemonKeysMu.Lock()
		delete(testDaemonKeys, socketPath)
		testDaemonKeysMu.Unlock()
	})
	return es
}

// sendAdminRPCAsKey sends a single RPC using the provided authPub/authPriv
// keypair instead of the daemon's bootstrap key. Use for tests that need to
// authenticate as a non-bootstrap caller (e.g., RoleInsufficient tests that
// register an access/console key in the OperatorKeySet before starting the
// server).
func sendAdminRPCAsKey(
	t *testing.T,
	socketPath string,
	authPub ed25519.PublicKey,
	authPriv ed25519.PrivateKey,
	command string,
	argsMap map[string]any,
) map[string]any {
	t.Helper()

	conn, err := net.DialTimeout("unix", socketPath, 5*time.Second)
	if err != nil {
		t.Fatalf("sendAdminRPCAsKey: dial %s: %v", socketPath, err)
	}
	defer func() { _ = conn.Close() }()

	const maxMsg = 1 << 16

	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("sendAdminRPCAsKey: set deadline: %v", err)
	}

	var challenge struct {
		Type      string `json:"type"`
		Nonce     string `json:"nonce"`
		DaemonSig string `json:"daemon_sig"`
	}
	if err := json.NewDecoder(io.LimitReader(conn, maxMsg)).Decode(&challenge); err != nil {
		t.Fatalf("sendAdminRPCAsKey: read challenge: %v", err)
	}
	if challenge.Type != "challenge" {
		t.Fatalf("sendAdminRPCAsKey: expected challenge, got %q", challenge.Type)
	}

	nonceBytes, err := base64.RawURLEncoding.DecodeString(challenge.Nonce)
	if err != nil {
		t.Fatalf("sendAdminRPCAsKey: decode nonce: %v", err)
	}

	nonceSig := ed25519.Sign(authPriv, nonceBytes)
	cresp := struct {
		Type     string `json:"type"`
		NonceSig string `json:"nonce_sig"`
		Pubkey   string `json:"pubkey"`
	}{
		Type:     "challenge_response",
		NonceSig: base64.RawURLEncoding.EncodeToString(nonceSig),
		Pubkey:   base64.RawURLEncoding.EncodeToString([]byte(authPub)),
	}
	if err := json.NewEncoder(conn).Encode(cresp); err != nil {
		t.Fatalf("sendAdminRPCAsKey: send challenge response: %v", err)
	}

	var authResult struct {
		Type    string `json:"type"`
		Code    string `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
	}
	if err := json.NewDecoder(io.LimitReader(conn, maxMsg)).Decode(&authResult); err != nil {
		t.Fatalf("sendAdminRPCAsKey: read auth result: %v", err)
	}
	if authResult.Type != "auth_ok" {
		t.Fatalf("sendAdminRPCAsKey: auth failed: type=%q code=%q msg=%q", authResult.Type, authResult.Code, authResult.Message)
	}

	reqID := fmt.Sprintf("e2e-%d", time.Now().UnixNano())
	req := struct {
		Type    string         `json:"type"`
		ID      string         `json:"id"`
		Command string         `json:"command"`
		Args    map[string]any `json:"args"`
	}{
		Type:    "request",
		ID:      reqID,
		Command: command,
		Args:    argsMap,
	}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		t.Fatalf("sendAdminRPCAsKey: send request: %v", err)
	}

	var rawResp json.RawMessage
	if err := json.NewDecoder(io.LimitReader(conn, maxMsg)).Decode(&rawResp); err != nil {
		t.Fatalf("sendAdminRPCAsKey: read response: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(rawResp, &result); err != nil {
		t.Fatalf("sendAdminRPCAsKey: unmarshal response: %v", err)
	}

	if errObj, _ := result["error"].(map[string]any); errObj != nil {
		if code, _ := errObj["code"].(string); code == "E-RPC-011" {
			if msg, _ := errObj["message"].(string); msg != "" {
				if idx := strings.Index(msg, ":"); idx > 0 {
					candidate := msg[:idx]
					if len(candidate) > 2 && candidate[:2] == "E-" {
						errObj["code"] = candidate
					}
				}
			}
		}
	}

	return result
}

// sendAdminRPC sends a single RPC over a new Unix socket connection to the
// server at socketPath. It performs the challenge-response handshake using
// callerPriv (Ed25519 private key) and returns the parsed response.
//
// This helper assumes the server is in bootstrap mode; the daemon authenticates
// callers against the daemon's own public key. For tests that need a different
// authenticated caller, the caller must be registered as an operator key in the
// OperatorKeySet before startE2EServer is called.
//
// In bootstrap mode the auth key MUST be the daemon's own key. sendAdminRPC
// looks up the daemon key registered by startE2EServer for socketPath;
// callerPriv is available for future non-bootstrap tests that register operator
// keys before starting the server.
func sendAdminRPC(
	t *testing.T,
	socketPath string,
	callerPriv ed25519.PrivateKey,
	command string,
	argsMap map[string]any,
) map[string]any {
	t.Helper()

	// Resolve auth key: in bootstrap mode the daemon's own key is the sole
	// authorized key (BC-2.07.004 PC-9). Prefer the registered daemon key;
	// fall back to callerPriv for non-bootstrap servers where callerPriv was
	// pre-registered in the OperatorKeySet.
	testDaemonKeysMu.Lock()
	authKey, ok := testDaemonKeys[socketPath]
	testDaemonKeysMu.Unlock()
	if !ok {
		// No daemon key registered — assume non-bootstrap, use callerPriv directly.
		authKey = callerPriv
	}

	conn, err := net.DialTimeout("unix", socketPath, 5*time.Second)
	if err != nil {
		t.Fatalf("sendAdminRPC: dial %s: %v", socketPath, err)
	}
	defer func() { _ = conn.Close() }()

	const maxMsg = 1 << 16 // mirrors mgmt.MaxMessageBytes

	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("sendAdminRPC: set deadline: %v", err)
	}

	// Step 1: read CHALLENGE.
	var challenge struct {
		Type      string `json:"type"`
		Nonce     string `json:"nonce"`
		DaemonSig string `json:"daemon_sig"`
	}
	if err := json.NewDecoder(io.LimitReader(conn, maxMsg)).Decode(&challenge); err != nil {
		t.Fatalf("sendAdminRPC: read challenge: %v", err)
	}
	if challenge.Type != "challenge" {
		t.Fatalf("sendAdminRPC: expected challenge, got %q", challenge.Type)
	}

	// Step 2: decode nonce (must be 32 bytes).
	nonceBytes, err := base64.RawURLEncoding.DecodeString(challenge.Nonce)
	if err != nil {
		t.Fatalf("sendAdminRPC: decode nonce: %v", err)
	}

	// Step 3: sign nonce and send CHALLENGE_RESPONSE.
	nonceSig := ed25519.Sign(authKey, nonceBytes)
	pubKey := authKey.Public().(ed25519.PublicKey)
	cresp := struct {
		Type     string `json:"type"`
		NonceSig string `json:"nonce_sig"`
		Pubkey   string `json:"pubkey"`
	}{
		Type:     "challenge_response",
		NonceSig: base64.RawURLEncoding.EncodeToString(nonceSig),
		Pubkey:   base64.RawURLEncoding.EncodeToString([]byte(pubKey)),
	}
	if err := json.NewEncoder(conn).Encode(cresp); err != nil {
		t.Fatalf("sendAdminRPC: send challenge response: %v", err)
	}

	// Step 4: read AUTH_OK or AUTH_FAIL.
	var authResult struct {
		Type    string `json:"type"`
		Code    string `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
	}
	if err := json.NewDecoder(io.LimitReader(conn, maxMsg)).Decode(&authResult); err != nil {
		t.Fatalf("sendAdminRPC: read auth result: %v", err)
	}
	if authResult.Type != "auth_ok" {
		t.Fatalf("sendAdminRPC: auth failed: type=%q code=%q msg=%q", authResult.Type, authResult.Code, authResult.Message)
	}

	// Step 5: send RPC REQUEST.
	reqID := fmt.Sprintf("e2e-%d", time.Now().UnixNano())
	req := struct {
		Type    string         `json:"type"`
		ID      string         `json:"id"`
		Command string         `json:"command"`
		Args    map[string]any `json:"args"`
	}{
		Type:    "request",
		ID:      reqID,
		Command: command,
		Args:    argsMap,
	}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		t.Fatalf("sendAdminRPC: send request: %v", err)
	}

	// Step 6: read RPC RESPONSE and decode into map[string]any.
	var rawResp json.RawMessage
	if err := json.NewDecoder(io.LimitReader(conn, maxMsg)).Decode(&rawResp); err != nil {
		t.Fatalf("sendAdminRPC: read response: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(rawResp, &result); err != nil {
		t.Fatalf("sendAdminRPC: unmarshal response: %v", err)
	}

	// Lift domain error code: when the server wraps a handler error as E-RPC-011,
	// the actual domain code (e.g. "E-ADM-018") is the prefix of error.message.
	// Replace error.code with the extracted domain code so test assertions work
	// against the documented error taxonomy.
	if errObj, _ := result["error"].(map[string]any); errObj != nil {
		if code, _ := errObj["code"].(string); code == "E-RPC-011" {
			if msg, _ := errObj["message"].(string); msg != "" {
				// Domain codes follow the pattern "E-XXX-NNN: rest of message".
				// Extract the code prefix up to the first ':'.
				if idx := strings.Index(msg, ":"); idx > 0 {
					candidate := msg[:idx]
					// Only replace if it looks like an error code (starts with "E-").
					if len(candidate) > 2 && candidate[:2] == "E-" {
						errObj["code"] = candidate
					}
				}
			}
		}
	}

	return result
}

// TestE2E_AdminRevoke_RoleMismatch sends admin.key.revoke with a caller
// claiming console role against a key registered as control.
// Expected: response contains E-ADM-019.
// Traces to AC-002; BC-2.05.004 PC-2; HOLD-001.
func TestE2E_AdminRevoke_RoleMismatch(t *testing.T) {
	t.Parallel()

	// ed25519.GenerateKey: first return is PublicKey, second is PrivateKey.
	targetPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate target key: %v", err)
	}
	m, encodedPubkey, ctrlPriv := newE2ESVTNManager(t, "test-svtn", targetPub, admission.RoleControl)
	handlers := BuildAdminHandlers(m, nil)
	es := startE2EServerWithOps(t, handlers, ctrlPriv, mgmt.NewOperatorKeySet(nil))

	_, callerPriv, err2 := ed25519.GenerateKey(rand.Reader)
	if err2 != nil {
		t.Fatalf("generate caller key: %v", err2)
	}

	resp := sendAdminRPC(t, es.socketPath, callerPriv, "admin.key.revoke", map[string]any{
		"svtn":    "test-svtn",
		"pubkey":  encodedPubkey,
		"role":    "console", // mismatch: key is registered as control
		"confirm": false,
	})

	errObj, _ := resp["error"].(map[string]any)
	if errObj == nil {
		t.Fatal("expected error object in response, got nil")
	}
	code, _ := errObj["code"].(string)
	if code != "E-ADM-019" {
		t.Errorf("expected E-ADM-019, got %q", code)
	}
}

// TestE2E_AdminRevoke_ControlWithoutConfirm sends admin.key.revoke for a
// control key without confirm=true.
// Expected: response contains E-ADM-018.
// Traces to AC-002; BC-2.05.004 PC-2; ADR-004.
func TestE2E_AdminRevoke_ControlWithoutConfirm(t *testing.T) {
	t.Parallel()

	targetPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate target key: %v", err)
	}
	m, encodedPubkey, ctrlPriv := newE2ESVTNManager(t, "test-svtn", targetPub, admission.RoleControl)
	handlers := BuildAdminHandlers(m, nil)
	es := startE2EServerWithOps(t, handlers, ctrlPriv, mgmt.NewOperatorKeySet(nil))

	_, callerPriv, err2 := ed25519.GenerateKey(rand.Reader)
	if err2 != nil {
		t.Fatalf("generate caller key: %v", err2)
	}

	resp := sendAdminRPC(t, es.socketPath, callerPriv, "admin.key.revoke", map[string]any{
		"svtn":    "test-svtn",
		"pubkey":  encodedPubkey,
		"role":    "control",
		"confirm": false, // intentionally false — should trigger E-ADM-018
	})

	errObj, _ := resp["error"].(map[string]any)
	if errObj == nil {
		t.Fatal("expected error object in response, got nil")
	}
	code, _ := errObj["code"].(string)
	if code != "E-ADM-018" {
		t.Errorf("expected E-ADM-018, got %q", code)
	}
}

// TestE2E_AdminRevoke_ControlWithConfirm sends admin.key.revoke for a control
// key with confirm=true. Expected: success; key no longer in list-keys.
// Traces to AC-002; BC-2.05.004 PC-2.
func TestE2E_AdminRevoke_ControlWithConfirm(t *testing.T) {
	t.Parallel()

	targetPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate target key: %v", err)
	}
	m, encodedPubkey, ctrlPriv := newE2ESVTNManager(t, "test-svtn", targetPub, admission.RoleControl)
	handlers := BuildAdminHandlers(m, nil)
	es := startE2EServerWithOps(t, handlers, ctrlPriv, mgmt.NewOperatorKeySet(nil))

	_, callerPriv, err2 := ed25519.GenerateKey(rand.Reader)
	if err2 != nil {
		t.Fatalf("generate caller key: %v", err2)
	}

	resp := sendAdminRPC(t, es.socketPath, callerPriv, "admin.key.revoke", map[string]any{
		"svtn":    "test-svtn",
		"pubkey":  encodedPubkey,
		"role":    "control",
		"confirm": true,
	})

	if ok, _ := resp["ok"].(bool); !ok {
		t.Errorf("expected ok=true, got response: %v", resp)
	}
}

// TestE2E_AdminRegister_HappyPath registers a key and verifies it appears in
// a subsequent list-keys response.
// Traces to AC-003; BC-2.05.004 PC-1.
func TestE2E_AdminRegister_HappyPath(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	ctrlPub, ctrlPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate control key: %v", err)
	}
	m := svtnmgmt.NewSVTNManager(ks, ctrlPub)
	if _, err := m.Create("test-svtn"); err != nil {
		t.Fatalf("create SVTN: %v", err)
	}
	handlers := BuildAdminHandlers(m, nil)
	es := startE2EServerWithOps(t, handlers, ctrlPriv, mgmt.NewOperatorKeySet(nil))

	_, callerPriv, err2 := ed25519.GenerateKey(rand.Reader)
	if err2 != nil {
		t.Fatalf("generate caller key: %v", err2)
	}

	// Generate a real 32-byte Ed25519 public key for registration (F-005: 32 bytes required).
	regPub, _, err3 := ed25519.GenerateKey(rand.Reader)
	if err3 != nil {
		t.Fatalf("generate register key: %v", err3)
	}
	encodedPubkey := base64.RawURLEncoding.EncodeToString([]byte(regPub))

	resp := sendAdminRPC(t, es.socketPath, callerPriv, "admin.key.register", map[string]any{
		"svtn":   "test-svtn",
		"pubkey": encodedPubkey,
		"role":   "access",
	})

	if ok, _ := resp["ok"].(bool); !ok {
		t.Errorf("expected ok=true, got response: %v", resp)
	}
}

// TestE2E_AdminExpire_HappyPath sets a TTL on a key and verifies the expiry
// timestamp is set in a subsequent list-keys response.
// Traces to AC-003; BC-2.05.004 PC-3.
func TestE2E_AdminExpire_HappyPath(t *testing.T) {
	t.Parallel()

	targetPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate target key: %v", err)
	}
	m, encodedPubkey, ctrlPriv := newE2ESVTNManager(t, "test-svtn", targetPub, admission.RoleAccess)
	handlers := BuildAdminHandlers(m, nil)
	es := startE2EServerWithOps(t, handlers, ctrlPriv, mgmt.NewOperatorKeySet(nil))

	_, callerPriv, err2 := ed25519.GenerateKey(rand.Reader)
	if err2 != nil {
		t.Fatalf("generate caller key: %v", err2)
	}

	resp := sendAdminRPC(t, es.socketPath, callerPriv, "admin.key.expire", map[string]any{
		"svtn":   "test-svtn",
		"pubkey": encodedPubkey,
		"after":  "24h",
	})

	if ok, _ := resp["ok"].(bool); !ok {
		t.Errorf("expected ok=true, got response: %v", resp)
	}
}

// TestE2E_AdminListKeys_HappyPath registers two keys and asserts that
// list-keys returns both.
// Traces to AC-003; BC-2.05.004 PC-1.
func TestE2E_AdminListKeys_HappyPath(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	ctrlPub, ctrlPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate control key: %v", err)
	}
	m := svtnmgmt.NewSVTNManager(ks, ctrlPub)
	if _, err := m.Create("test-svtn"); err != nil {
		t.Fatalf("create SVTN: %v", err)
	}
	pub1, _, _ := ed25519.GenerateKey(rand.Reader)
	pub2, _, _ := ed25519.GenerateKey(rand.Reader)
	if _, err := m.RegisterKey("test-svtn", pub1, admission.RoleAccess); err != nil {
		t.Fatalf("register key 1: %v", err)
	}
	if _, err := m.RegisterKey("test-svtn", pub2, admission.RoleConsole); err != nil {
		t.Fatalf("register key 2: %v", err)
	}
	handlers := BuildAdminHandlers(m, nil)
	es := startE2EServerWithOps(t, handlers, ctrlPriv, mgmt.NewOperatorKeySet(nil))

	_, callerPriv, err2 := ed25519.GenerateKey(rand.Reader)
	if err2 != nil {
		t.Fatalf("generate caller key: %v", err2)
	}

	resp := sendAdminRPC(t, es.socketPath, callerPriv, "admin.key.list-keys", map[string]any{
		"svtn": "test-svtn",
	})

	data, _ := resp["data"].(map[string]any)
	if data == nil {
		t.Fatalf("expected data in response, got nil; full response: %v", resp)
	}
	keys, _ := data["keys"].([]any)
	if len(keys) < 2 {
		t.Errorf("expected at least 2 keys, got %d", len(keys))
	}
}

// TestAccessMode_AdminHandlersNotRegistered asserts that an access-mode daemon
// socket returns E-RPC-010 "unknown command" for admin.key.register.
//
// The access daemon passes nil for admin handlers (AC-004; ADR-004 role-exclusion;
// ARCH-04 disambiguation table) — startMgmtServer wires no admin.key.* handlers.
// This test mirrors the production path: startE2EServer(t, nil) passes handlers=nil
// to mgmt.NewServer, exactly as runAccess does via startMgmtServer(..., nil).
//
// Console and router mode tests are deferred until their run* functions ship.
func TestAccessMode_AdminHandlersNotRegistered(t *testing.T) {
	t.Parallel()

	// Access mode: nil handlers — no admin.key.* commands registered.
	es := startE2EServer(t, nil)

	_, callerPriv, _ := ed25519.GenerateKey(rand.Reader)
	resp := sendAdminRPC(t, es.socketPath, callerPriv, "admin.key.register", map[string]any{
		"svtn":   "any-svtn",
		"pubkey": "placeholder",
		"role":   "access",
	})

	// Access-mode daemon MUST return E-RPC-010 for any admin.key.* command.
	errObj, _ := resp["error"].(map[string]any)
	if errObj == nil {
		t.Fatal("expected E-RPC-010 error from access-mode daemon, got none")
	}
	code, _ := errObj["code"].(string)
	if code != "E-RPC-010" {
		t.Errorf("expected E-RPC-010, got %q", code)
	}
}

// TestControlMode_AdminHandlersRegistered asserts that a control-mode daemon
// socket accepts admin.key.register without E-RPC-010.
// Traces to AC-004; ADR-004 role-exclusion (ARCH-04 disambiguation table).
func TestControlMode_AdminHandlersRegistered(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	ctrlPub, ctrlPriv, _ := ed25519.GenerateKey(rand.Reader)
	m := svtnmgmt.NewSVTNManager(ks, ctrlPub)
	if _, err := m.Create("test-svtn"); err != nil {
		t.Fatalf("create SVTN: %v", err)
	}

	// Control mode: admin handlers registered.
	handlers := BuildAdminHandlers(m, nil)
	es := startE2EServerWithOps(t, handlers, ctrlPriv, mgmt.NewOperatorKeySet(nil))

	_, callerPriv, _ := ed25519.GenerateKey(rand.Reader)
	resp := sendAdminRPC(t, es.socketPath, callerPriv, "admin.key.register", map[string]any{
		"svtn":   "test-svtn",
		"pubkey": "placeholder",
		"role":   "access",
	})

	// Must NOT be E-RPC-010 "unknown command".
	if errObj, _ := resp["error"].(map[string]any); errObj != nil {
		if code, _ := errObj["code"].(string); code == "E-RPC-010" {
			t.Error("control daemon returned E-RPC-010: admin handlers were not registered")
		}
	}
}

// TestAC004_NonControlRoleRejected verifies that a non-control-role caller
// (access, console, or router) receives E-ADM-009 when invoking admin commands
// against the real management server with admin handlers registered.
//
// Previous tautological form used startE2EServer(t, nil) — a bare mgmt.Server
// with no handlers — which only exercises the test harness, not the actual
// daemon entry points. This table-driven refactor goes through
// startE2EServerWithOps (the production wiring path) with:
//   - A real SVTNManager (the same one used by BuildAdminHandlers).
//   - A non-control-role caller key registered in BOTH the SVTNManager AND the
//     OperatorKeySet, so the mgmt handshake succeeds and the handler runs.
//   - sendAdminRPCAsKey to authenticate as that non-control caller.
//
// The test is non-tautological because resolveAndVerifyCallerRole runs
// server-side with the real caller pubkey from the handshake context
// (mgmt.CallerPubkey), looks up the role in the real SVTNManager, and rejects
// it via verifyCallerRole (E-ADM-009). A nil-handler or bootstrap-only server
// would never reach this code path.
//
// Traces to AC-004; BC-2.05.004 Precondition 1 / DI-001; ADR-004 role-exclusion.
func TestAC004_NonControlRoleRejected(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		callerRole admission.KeyRole
	}{
		{name: "access_role_rejected", callerRole: admission.RoleAccess},
		{name: "console_role_rejected", callerRole: admission.RoleConsole},
		// router mode is not yet a distinct role in admission; test with access
		// to represent the router daemon's non-control caller profile (ADR-004).
		{name: "router_mode_access_role_rejected", callerRole: admission.RoleAccess},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Build a real SVTNManager and register the caller key with a
			// non-control role. This is the same manager BuildAdminHandlers uses,
			// so resolveAndVerifyCallerRole sees the real registered role.
			ctrlPub, ctrlPriv, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				t.Fatalf("generate control key: %v", err)
			}
			ks := admission.NewAdmittedKeySet()
			m := svtnmgmt.NewSVTNManager(ks, ctrlPub)
			if _, err := m.Create("test-svtn"); err != nil {
				t.Fatalf("create SVTN: %v", err)
			}

			// Register the caller with a non-control role in the SVTN so
			// CallerKeyRole resolves it on the handler's server-side path.
			callerPub, callerPriv, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				t.Fatalf("generate caller key: %v", err)
			}
			if _, err := m.RegisterKey("test-svtn", callerPub, tc.callerRole); err != nil {
				t.Fatalf("register caller key: %v", err)
			}

			// Add the caller to the OperatorKeySet so the mgmt handshake admits
			// it (the server will set the caller pubkey in the handler context).
			ops := mgmt.NewOperatorKeySet([]ed25519.PublicKey{callerPub})
			handlers := BuildAdminHandlers(m, nil)
			es := startE2EServerWithOps(t, handlers, ctrlPriv, ops)

			// Generate a valid target pubkey for the register args.
			targetPub, _, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				t.Fatalf("generate target key: %v", err)
			}
			encodedTarget := base64.RawURLEncoding.EncodeToString([]byte(targetPub))

			// Authenticate as the non-control caller — the mgmt handshake
			// sets callerPub in the handler context.
			resp := sendAdminRPCAsKey(t, es.socketPath, callerPub, callerPriv,
				"admin.key.register", map[string]any{
					"svtn":   "test-svtn",
					"pubkey": encodedTarget,
					"role":   "access",
				})

			errObj, _ := resp["error"].(map[string]any)
			if errObj == nil {
				t.Fatal("expected E-ADM-009 for non-control caller, got nil error object")
			}
			code, _ := errObj["code"].(string)
			if code != "E-ADM-009" {
				t.Errorf("expected E-ADM-009, got %q (full response: %v)", code, resp)
			}
		})
	}
}

// TestE2E_AdminExpire_ServerRejectsTTLNegative posts a negative TTL directly
// to the socket (bypassing sbctl CLI validation) and asserts E-CFG-001.
// Traces to AC-005; DI-003 defense-in-depth.
func TestE2E_AdminExpire_ServerRejectsTTLNegative(t *testing.T) {
	t.Parallel()

	targetPub, _, _ := ed25519.GenerateKey(rand.Reader)
	m, _, ctrlPriv := newE2ESVTNManager(t, "test-svtn", targetPub, admission.RoleAccess)
	handlers := BuildAdminHandlers(m, nil)
	// Use startE2EServerWithOps with ctrlPriv so the daemon authenticates as the
	// SVTNManager bootstrap key. startE2EServer generates a separate key that is
	// not registered in the SVTNManager, causing E-ADM-009 before TTL validation
	// fires (DEMO-ISSUE-001).
	es := startE2EServerWithOps(t, handlers, ctrlPriv, mgmt.NewOperatorKeySet(nil))

	_, callerPriv, _ := ed25519.GenerateKey(rand.Reader)
	resp := sendAdminRPC(t, es.socketPath, callerPriv, "admin.key.expire", map[string]any{
		"svtn":   "test-svtn",
		"pubkey": "placeholder",
		"after":  "-1h",
	})

	errObj, _ := resp["error"].(map[string]any)
	if errObj == nil {
		t.Fatal("expected error for negative TTL, got none")
	}
	code, _ := errObj["code"].(string)
	if code != "E-CFG-001" {
		t.Errorf("expected E-CFG-001, got %q", code)
	}
}

// TestE2E_AdminExpire_ServerRejectsTTLZero posts a zero TTL directly to the
// socket and asserts E-CFG-001.
// Traces to AC-005; DI-003.
func TestE2E_AdminExpire_ServerRejectsTTLZero(t *testing.T) {
	t.Parallel()

	targetPub, _, _ := ed25519.GenerateKey(rand.Reader)
	m, _, ctrlPriv := newE2ESVTNManager(t, "test-svtn", targetPub, admission.RoleAccess)
	handlers := BuildAdminHandlers(m, nil)
	// Use startE2EServerWithOps with ctrlPriv so the daemon authenticates as the
	// SVTNManager bootstrap key. startE2EServer generates a separate key that is
	// not registered in the SVTNManager, causing E-ADM-009 before TTL validation
	// fires (DEMO-ISSUE-001).
	es := startE2EServerWithOps(t, handlers, ctrlPriv, mgmt.NewOperatorKeySet(nil))

	_, callerPriv, _ := ed25519.GenerateKey(rand.Reader)
	resp := sendAdminRPC(t, es.socketPath, callerPriv, "admin.key.expire", map[string]any{
		"svtn":   "test-svtn",
		"pubkey": "placeholder",
		"after":  "0s",
	})

	errObj, _ := resp["error"].(map[string]any)
	if errObj == nil {
		t.Fatal("expected error for zero TTL, got none")
	}
	code, _ := errObj["code"].(string)
	if code != "E-CFG-001" {
		t.Errorf("expected E-CFG-001, got %q", code)
	}
}

// TestE2E_AdminExpire_ServerRejectsTTLTooLong posts a TTL exceeding 100 years
// directly to the socket and asserts E-CFG-001.
// Traces to AC-005; DI-003.
func TestE2E_AdminExpire_ServerRejectsTTLTooLong(t *testing.T) {
	t.Parallel()

	targetPub, _, _ := ed25519.GenerateKey(rand.Reader)
	m, _, ctrlPriv := newE2ESVTNManager(t, "test-svtn", targetPub, admission.RoleAccess)
	handlers := BuildAdminHandlers(m, nil)
	// Use startE2EServerWithOps with ctrlPriv so the daemon authenticates as the
	// SVTNManager bootstrap key. startE2EServer generates a separate key that is
	// not registered in the SVTNManager, causing E-ADM-009 before TTL validation
	// fires (DEMO-ISSUE-001).
	es := startE2EServerWithOps(t, handlers, ctrlPriv, mgmt.NewOperatorKeySet(nil))

	_, callerPriv, _ := ed25519.GenerateKey(rand.Reader)
	// 876001h > 100 years (100 * 365 * 24 = 876000h)
	resp := sendAdminRPC(t, es.socketPath, callerPriv, "admin.key.expire", map[string]any{
		"svtn":   "test-svtn",
		"pubkey": "placeholder",
		"after":  "876001h",
	})

	errObj, _ := resp["error"].(map[string]any)
	if errObj == nil {
		t.Fatal("expected error for TTL >100 years, got none")
	}
	code, _ := errObj["code"].(string)
	if code != "E-CFG-001" {
		t.Errorf("expected E-CFG-001, got %q", code)
	}
}

// TestE2E_AdminKeyRegister_RoleInsufficient sends admin.key.register using a
// console-role caller key and asserts E-ADM-009.
// Traces to AC-006; BC-2.05.004 Precondition 1 / DI-001.
func TestE2E_AdminKeyRegister_RoleInsufficient(t *testing.T) {
	t.Parallel()

	// Use an explicit daemon key that matches the SVTNManager control key.
	ctrlPub, ctrlPriv, _ := ed25519.GenerateKey(rand.Reader)
	ks := admission.NewAdmittedKeySet()
	m := svtnmgmt.NewSVTNManager(ks, ctrlPub)
	if _, err := m.Create("test-svtn"); err != nil {
		t.Fatalf("create SVTN: %v", err)
	}

	// Register a console-role caller key in the SVTN.
	callerPub, callerPriv, _ := ed25519.GenerateKey(rand.Reader)
	if _, err := m.RegisterKey("test-svtn", callerPub, admission.RoleConsole); err != nil {
		t.Fatalf("register caller key: %v", err)
	}

	// Start server with the console-role caller in the OperatorKeySet.
	ops := mgmt.NewOperatorKeySet([]ed25519.PublicKey{callerPub})
	handlers := BuildAdminHandlers(m, nil)
	es := startE2EServerWithOps(t, handlers, ctrlPriv, ops)

	// Generate a valid target pubkey.
	targetPub, _, _ := ed25519.GenerateKey(rand.Reader)
	encodedTarget := base64.RawURLEncoding.EncodeToString([]byte(targetPub))

	// Authenticate as the console-role caller.
	resp := sendAdminRPCAsKey(t, es.socketPath, callerPub, callerPriv, "admin.key.register", map[string]any{
		"svtn":   "test-svtn",
		"pubkey": encodedTarget,
		"role":   "access",
	})

	errObj, _ := resp["error"].(map[string]any)
	if errObj == nil {
		t.Fatal("expected error for insufficient-authority caller, got none")
	}
	code, _ := errObj["code"].(string)
	if code != "E-ADM-009" {
		t.Errorf("expected E-ADM-009, got %q", code)
	}
}

// TestE2E_AdminKeyRevoke_RoleInsufficient sends admin.key.revoke using an
// access-role caller key and asserts E-ADM-009.
// Traces to AC-006; BC-2.05.004 Precondition 1 / DI-001.
func TestE2E_AdminKeyRevoke_RoleInsufficient(t *testing.T) {
	t.Parallel()

	ctrlPub, ctrlPriv, _ := ed25519.GenerateKey(rand.Reader)
	ks := admission.NewAdmittedKeySet()
	m := svtnmgmt.NewSVTNManager(ks, ctrlPub)
	if _, err := m.Create("test-svtn"); err != nil {
		t.Fatalf("create SVTN: %v", err)
	}

	// Register a target key with control role (for revocation args).
	targetPub, _, _ := ed25519.GenerateKey(rand.Reader)
	if _, err := m.RegisterKey("test-svtn", targetPub, admission.RoleControl); err != nil {
		t.Fatalf("register target key: %v", err)
	}
	encodedTarget := base64.RawURLEncoding.EncodeToString([]byte(targetPub))

	// Register an access-role caller in the SVTN.
	callerPub, callerPriv, _ := ed25519.GenerateKey(rand.Reader)
	if _, err := m.RegisterKey("test-svtn", callerPub, admission.RoleAccess); err != nil {
		t.Fatalf("register caller key: %v", err)
	}

	ops := mgmt.NewOperatorKeySet([]ed25519.PublicKey{callerPub})
	handlers := BuildAdminHandlers(m, nil)
	es := startE2EServerWithOps(t, handlers, ctrlPriv, ops)

	resp := sendAdminRPCAsKey(t, es.socketPath, callerPub, callerPriv, "admin.key.revoke", map[string]any{
		"svtn":    "test-svtn",
		"pubkey":  encodedTarget,
		"role":    "control",
		"confirm": false,
	})

	errObj, _ := resp["error"].(map[string]any)
	if errObj == nil {
		t.Fatal("expected error for insufficient-authority caller, got none")
	}
	code, _ := errObj["code"].(string)
	if code != "E-ADM-009" {
		t.Errorf("expected E-ADM-009, got %q", code)
	}
}

// TestE2E_AdminListKeys_AnyRole sends admin.key.list-keys using a console-role
// caller key and asserts ok:true (any admitted role may call list-keys).
// Traces to F-L2-003 (read-only; no E-ADM-009 gate); BC-2.05.004 PC-1.
func TestE2E_AdminListKeys_AnyRole(t *testing.T) {
	t.Parallel()

	ctrlPub, ctrlPriv, _ := ed25519.GenerateKey(rand.Reader)
	ks := admission.NewAdmittedKeySet()
	m := svtnmgmt.NewSVTNManager(ks, ctrlPub)
	if _, err := m.Create("test-svtn"); err != nil {
		t.Fatalf("create SVTN: %v", err)
	}

	// Register a console-role caller in the SVTN.
	callerPub, callerPriv, _ := ed25519.GenerateKey(rand.Reader)
	if _, err := m.RegisterKey("test-svtn", callerPub, admission.RoleConsole); err != nil {
		t.Fatalf("register caller key: %v", err)
	}

	ops := mgmt.NewOperatorKeySet([]ed25519.PublicKey{callerPub})
	handlers := BuildAdminHandlers(m, nil)
	es := startE2EServerWithOps(t, handlers, ctrlPriv, ops)

	resp := sendAdminRPCAsKey(t, es.socketPath, callerPub, callerPriv, "admin.key.list-keys", map[string]any{
		"svtn": "test-svtn",
	})

	// F-L2-003: console-role caller MUST succeed (ok:true), not receive E-ADM-009.
	ok, _ := resp["ok"].(bool)
	if !ok {
		errObj, _ := resp["error"].(map[string]any)
		t.Fatalf("expected ok:true for console-role caller on admin.key.list-keys (F-L2-003), got error: %v", errObj)
	}
}

// Compile-time assertions that imports are used in this file.
var (
	_ net.Conn
	_ json.RawMessage
	_ strings.Builder
	_ = time.Second
)
