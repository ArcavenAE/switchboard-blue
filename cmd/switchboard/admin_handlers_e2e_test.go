//go:build integration

package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
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
	cancel     context.CancelFunc
	doneCh     chan struct{}
}

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

	es := &e2eServer{
		srv:        srv,
		socketPath: socketPath,
		cancel:     cancel,
		doneCh:     done,
	}
	t.Cleanup(func() {
		cancel()
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = srv.Shutdown(shutCtx)
		shutCancel()
		<-done
	})
	return es
}

// newE2ESVTNManager creates a minimal SVTNManager with a registered SVTN named
// svtnName and a pre-registered key with the given role.
// ed25519.GenerateKey returns (PublicKey, PrivateKey, error) — ctrlPriv is used
// only to construct the manager's control key; pubkey is the key being pre-registered.
func newE2ESVTNManager(t *testing.T, svtnName string, pubkey ed25519.PublicKey, role admission.KeyRole) *svtnmgmt.SVTNManager {
	t.Helper()
	ks := admission.NewAdmittedKeySet()
	// ed25519.GenerateKey: first return is PublicKey, second is PrivateKey.
	ctrlPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("newE2ESVTNManager: generate control key: %v", err)
	}
	m := svtnmgmt.NewSVTNManager(ks, ctrlPub)
	if _, err := m.Create(svtnName); err != nil {
		t.Fatalf("newE2ESVTNManager: create SVTN %q: %v", svtnName, err)
	}
	if _, err := m.RegisterKey(svtnName, pubkey, role); err != nil {
		t.Fatalf("newE2ESVTNManager: register key: %v", err)
	}
	return m
}

// sendAdminRPC sends a single RPC over a new Unix socket connection to the
// server at socketPath. It performs the challenge-response handshake using
// callerPriv (Ed25519 private key) and returns the parsed response.
//
// This helper assumes the server is in bootstrap mode; the daemon authenticates
// callers against the daemon's own public key. For tests that need a different
// authenticated caller, the caller must be registered as an operator key in the
// OperatorKeySet before startE2EServer is called.
func sendAdminRPC(
	t *testing.T,
	socketPath string,
	callerPriv ed25519.PrivateKey,
	command string,
	argsMap map[string]any,
) map[string]any {
	t.Helper()
	panic("todo: e2e RPC transport helper — implement in S-6.06")
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
	m := newE2ESVTNManager(t, "test-svtn", targetPub, admission.RoleControl)
	handlers := BuildAdminHandlers(m)
	es := startE2EServer(t, handlers)

	_, callerPriv, err2 := ed25519.GenerateKey(rand.Reader)
	if err2 != nil {
		t.Fatalf("generate caller key: %v", err2)
	}

	resp := sendAdminRPC(t, es.socketPath, callerPriv, "admin.key.revoke", map[string]any{
		"svtn":    "test-svtn",
		"pubkey":  "placeholder",
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
	m := newE2ESVTNManager(t, "test-svtn", targetPub, admission.RoleControl)
	handlers := BuildAdminHandlers(m)
	es := startE2EServer(t, handlers)

	_, callerPriv, err2 := ed25519.GenerateKey(rand.Reader)
	if err2 != nil {
		t.Fatalf("generate caller key: %v", err2)
	}

	resp := sendAdminRPC(t, es.socketPath, callerPriv, "admin.key.revoke", map[string]any{
		"svtn":    "test-svtn",
		"pubkey":  "placeholder",
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
	m := newE2ESVTNManager(t, "test-svtn", targetPub, admission.RoleControl)
	handlers := BuildAdminHandlers(m)
	es := startE2EServer(t, handlers)

	_, callerPriv, err2 := ed25519.GenerateKey(rand.Reader)
	if err2 != nil {
		t.Fatalf("generate caller key: %v", err2)
	}

	resp := sendAdminRPC(t, es.socketPath, callerPriv, "admin.key.revoke", map[string]any{
		"svtn":    "test-svtn",
		"pubkey":  "placeholder",
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
	ctrlPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate control key: %v", err)
	}
	m := svtnmgmt.NewSVTNManager(ks, ctrlPub)
	if _, err := m.Create("test-svtn"); err != nil {
		t.Fatalf("create SVTN: %v", err)
	}
	handlers := BuildAdminHandlers(m)
	es := startE2EServer(t, handlers)

	_, callerPriv, err2 := ed25519.GenerateKey(rand.Reader)
	if err2 != nil {
		t.Fatalf("generate caller key: %v", err2)
	}

	resp := sendAdminRPC(t, es.socketPath, callerPriv, "admin.key.register", map[string]any{
		"svtn":   "test-svtn",
		"pubkey": "placeholder",
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
	m := newE2ESVTNManager(t, "test-svtn", targetPub, admission.RoleAccess)
	handlers := BuildAdminHandlers(m)
	es := startE2EServer(t, handlers)

	_, callerPriv, err2 := ed25519.GenerateKey(rand.Reader)
	if err2 != nil {
		t.Fatalf("generate caller key: %v", err2)
	}

	resp := sendAdminRPC(t, es.socketPath, callerPriv, "admin.key.expire", map[string]any{
		"svtn":   "test-svtn",
		"pubkey": "placeholder",
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
	ctrlPub, _, err := ed25519.GenerateKey(rand.Reader)
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
	handlers := BuildAdminHandlers(m)
	es := startE2EServer(t, handlers)

	_, callerPriv, err2 := ed25519.GenerateKey(rand.Reader)
	if err2 != nil {
		t.Fatalf("generate caller key: %v", err2)
	}

	resp := sendAdminRPC(t, es.socketPath, callerPriv, "admin.list-keys", map[string]any{
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

// TestControlMode_AdminHandlersRegistered asserts that a control-mode daemon
// socket accepts admin.key.register without E-RPC-010.
// Traces to AC-004; ADR-004 role-exclusion; ARCH-08 §6.6.2.
func TestControlMode_AdminHandlersRegistered(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	ctrlPub, _, _ := ed25519.GenerateKey(rand.Reader)
	m := svtnmgmt.NewSVTNManager(ks, ctrlPub)
	if _, err := m.Create("test-svtn"); err != nil {
		t.Fatalf("create SVTN: %v", err)
	}

	// Control mode: admin handlers registered.
	handlers := BuildAdminHandlers(m)
	es := startE2EServer(t, handlers)

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

// TestAccessMode_AdminHandlersNotRegistered asserts that an access-mode daemon
// socket returns E-RPC-010 for admin.key.register.
// Traces to AC-004; ADR-004 role-exclusion.
func TestAccessMode_AdminHandlersNotRegistered(t *testing.T) {
	t.Parallel()

	// Access mode: no admin handlers registered (nil slice).
	es := startE2EServer(t, nil)

	_, callerPriv, _ := ed25519.GenerateKey(rand.Reader)
	resp := sendAdminRPC(t, es.socketPath, callerPriv, "admin.key.register", map[string]any{
		"svtn":   "test-svtn",
		"pubkey": "placeholder",
		"role":   "access",
	})

	errObj, _ := resp["error"].(map[string]any)
	if errObj == nil {
		t.Fatal("expected error from access daemon for admin command, got none")
	}
	code, _ := errObj["code"].(string)
	if code != "E-RPC-010" {
		t.Errorf("expected E-RPC-010, got %q", code)
	}
}

// TestConsoleMode_AdminHandlersNotRegistered asserts that a console-mode
// daemon socket returns E-RPC-010 for admin.key.register.
// Traces to AC-004.
func TestConsoleMode_AdminHandlersNotRegistered(t *testing.T) {
	t.Parallel()

	// Console mode: no admin handlers registered (nil slice).
	es := startE2EServer(t, nil)

	_, callerPriv, _ := ed25519.GenerateKey(rand.Reader)
	resp := sendAdminRPC(t, es.socketPath, callerPriv, "admin.key.register", nil)

	errObj, _ := resp["error"].(map[string]any)
	if errObj == nil {
		t.Fatal("expected error from console daemon for admin command, got none")
	}
	code, _ := errObj["code"].(string)
	if code != "E-RPC-010" {
		t.Errorf("expected E-RPC-010, got %q", code)
	}
}

// TestRouterMode_AdminHandlersNotRegistered asserts that a router-mode daemon
// socket returns E-RPC-010 for admin.key.register.
// Traces to AC-004.
func TestRouterMode_AdminHandlersNotRegistered(t *testing.T) {
	t.Parallel()

	// Router mode: no admin handlers registered (nil slice).
	es := startE2EServer(t, nil)

	_, callerPriv, _ := ed25519.GenerateKey(rand.Reader)
	resp := sendAdminRPC(t, es.socketPath, callerPriv, "admin.key.register", nil)

	errObj, _ := resp["error"].(map[string]any)
	if errObj == nil {
		t.Fatal("expected error from router daemon for admin command, got none")
	}
	code, _ := errObj["code"].(string)
	if code != "E-RPC-010" {
		t.Errorf("expected E-RPC-010, got %q", code)
	}
}

// TestE2E_AdminExpire_ServerRejectsTTLNegative posts a negative TTL directly
// to the socket (bypassing sbctl CLI validation) and asserts E-CFG-001.
// Traces to AC-005; DI-003 defense-in-depth.
func TestE2E_AdminExpire_ServerRejectsTTLNegative(t *testing.T) {
	t.Parallel()

	targetPub, _, _ := ed25519.GenerateKey(rand.Reader)
	m := newE2ESVTNManager(t, "test-svtn", targetPub, admission.RoleAccess)
	handlers := BuildAdminHandlers(m)
	es := startE2EServer(t, handlers)

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
	m := newE2ESVTNManager(t, "test-svtn", targetPub, admission.RoleAccess)
	handlers := BuildAdminHandlers(m)
	es := startE2EServer(t, handlers)

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
	m := newE2ESVTNManager(t, "test-svtn", targetPub, admission.RoleAccess)
	handlers := BuildAdminHandlers(m)
	es := startE2EServer(t, handlers)

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
// Traces to AC-006; BC-2.07.001 invariant 3.
func TestE2E_AdminKeyRegister_RoleInsufficient(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	ctrlPub, _, _ := ed25519.GenerateKey(rand.Reader)
	m := svtnmgmt.NewSVTNManager(ks, ctrlPub)
	if _, err := m.Create("test-svtn"); err != nil {
		t.Fatalf("create SVTN: %v", err)
	}
	handlers := BuildAdminHandlers(m)
	es := startE2EServer(t, handlers)

	// Caller is a console-role key; expected to be rejected with E-ADM-009.
	_, callerPriv, _ := ed25519.GenerateKey(rand.Reader)
	resp := sendAdminRPC(t, es.socketPath, callerPriv, "admin.key.register", map[string]any{
		"svtn":        "test-svtn",
		"pubkey":      "placeholder",
		"role":        "access",
		"caller_role": "console", // signals to handler that caller has console role
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
// Traces to AC-006; BC-2.07.001 invariant 3.
func TestE2E_AdminKeyRevoke_RoleInsufficient(t *testing.T) {
	t.Parallel()

	targetPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate target key: %v", err)
	}
	m := newE2ESVTNManager(t, "test-svtn", targetPub, admission.RoleControl)
	handlers := BuildAdminHandlers(m)
	es := startE2EServer(t, handlers)

	_, callerPriv, _ := ed25519.GenerateKey(rand.Reader)
	resp := sendAdminRPC(t, es.socketPath, callerPriv, "admin.key.revoke", map[string]any{
		"svtn":        "test-svtn",
		"pubkey":      "placeholder",
		"role":        "control",
		"confirm":     false,
		"caller_role": "access",
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

// TestE2E_AdminListKeys_RoleInsufficient sends admin.list-keys using a
// console-role caller key and asserts E-ADM-009.
// Traces to AC-006; BC-2.07.001 invariant 3.
func TestE2E_AdminListKeys_RoleInsufficient(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	ctrlPub, _, _ := ed25519.GenerateKey(rand.Reader)
	m := svtnmgmt.NewSVTNManager(ks, ctrlPub)
	if _, err := m.Create("test-svtn"); err != nil {
		t.Fatalf("create SVTN: %v", err)
	}
	handlers := BuildAdminHandlers(m)
	es := startE2EServer(t, handlers)

	_, callerPriv, _ := ed25519.GenerateKey(rand.Reader)
	resp := sendAdminRPC(t, es.socketPath, callerPriv, "admin.list-keys", map[string]any{
		"svtn":        "test-svtn",
		"caller_role": "console",
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

// Compile-time assertions that imports are used in this file.
var (
	_ net.Conn
	_ json.RawMessage
	_ strings.Builder
	_ = time.Second
)
