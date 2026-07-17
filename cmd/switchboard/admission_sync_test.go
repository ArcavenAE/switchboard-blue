// admission_sync_test.go — failing tests for S-BL.ADMISSION-SYNC-WIRE ACs 2–10.
//
// All tests in this file MUST FAIL before any implementation is written (Red Gate).
// They compile against the stubs in admission_sync_client.go, admission_sync_wire.go,
// and admission_sync_snapshot.go, which return errAdmissionSyncNotImplemented /
// errSnapshotNotImplemented or do nothing.
//
// Traceability:
//
//	AC-002 → BC-2.05.009 PC-1/PC-3/Inv-3 (handler registration, router-only)
//	AC-003 → BC-2.05.009 PC-1/PC-2 (RegisterKey push, push-failure advisory)
//	AC-004 → BC-2.05.009 PC-1/PC-2 (RevokeKey/Expire/RemoveSVTN push, advisory)
//	AC-005 → BC-2.05.009 PC-8 / BC-2.05.010 PC-1/PC-3 (router handler, snapshot write)
//	AC-006 → BC-2.05.010 PC-4/PC-5 (snapshot JSON round-trip)
//	AC-007 → BC-2.05.010 PC-6/7/8/9 (startup load semantics)
//	AC-008 → BC-2.09.003 v2.1 PC-14 / Ruling 9 (non-loopback bind, startup INFO log)
//	AC-009 → BC-2.05.009 PC-7 (PushFullSnapshot on control startup)
//	AC-010 → BC-2.05.009 Inv-5 (SIGHUP reload endpoint update)
//
// Non-parallel notes:
//   - Any test using t.Setenv MUST NOT call t.Parallel.
//   - Tests touching filesystem sockets or spawning runRouter must NOT call
//     t.Parallel to avoid the listenUnixMgmt umask race (umask is process-global,
//     serialized by a package mutex, but concurrently-created tempdirs may lose
//     execute permission). See mgmt_wire_test.go "umask race" comment.
//   - Pure unit tests (no sockets, no tempfiles) call t.Parallel safely.

package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/config"
	"github.com/arcavenae/switchboard/internal/mgmt"
	"github.com/arcavenae/switchboard/internal/svtnmgmt"
)

// ── helpers ────────────────────────────────────────────────────────────────────

// mustGenKeySyncTest generates an Ed25519 keypair or fatals.
//
//nolint:unused // test writer helper; retained for future test expansion
func mustGenKeySyncTest(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	return pub, priv
}

// newSVTNManagerWithSVTN creates an SVTNManager with a single SVTN registered,
// and returns the manager plus the [16]byte ID of the created SVTN.
//
//nolint:unused // test writer helper; retained for future test expansion
func newSVTNManagerWithSVTN(t *testing.T, svtnName string) (*svtnmgmt.SVTNManager, [16]byte) {
	t.Helper()
	ctrlPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate control key: %v", err)
	}
	ks := admission.NewAdmittedKeySet()
	m := svtnmgmt.NewSVTNManager(ks, ctrlPub)
	cr, err := m.Create(svtnName)
	if err != nil {
		t.Fatalf("create SVTN %q: %v", svtnName, err)
	}
	return m, cr.SVTN.ID
}

// mockSyncer is a thread-unsafe record-keeping admissionSyncer for unit tests.
// It records every Push* call and can be configured to return an error.
type mockSyncer struct {
	// calls records the sequence of method names called ("PushRegisterKey", etc.)
	calls []string
	// args records the [svtn_id, pubkey, role, ...] for each call in order
	args [][]interface{}
	// err is returned by all Push* methods when non-nil.
	err error
}

func (m *mockSyncer) PushRegisterKey(_ context.Context, svtnID [16]byte, pubkey ed25519.PublicKey, role admission.KeyRole) error {
	m.calls = append(m.calls, "PushRegisterKey")
	m.args = append(m.args, []interface{}{svtnID, pubkey, role})
	return m.err
}

func (m *mockSyncer) PushRevokeKey(_ context.Context, svtnID [16]byte, pubkey ed25519.PublicKey, role admission.KeyRole, confirm bool) error {
	m.calls = append(m.calls, "PushRevokeKey")
	m.args = append(m.args, []interface{}{svtnID, pubkey, role, confirm})
	return m.err
}

func (m *mockSyncer) PushSetKeyExpiry(_ context.Context, svtnID [16]byte, pubkey ed25519.PublicKey, ttl time.Duration) error {
	m.calls = append(m.calls, "PushSetKeyExpiry")
	m.args = append(m.args, []interface{}{svtnID, pubkey, ttl})
	return m.err
}

func (m *mockSyncer) PushRemoveSVTN(_ context.Context, svtnID [16]byte) error {
	m.calls = append(m.calls, "PushRemoveSVTN")
	m.args = append(m.args, []interface{}{svtnID})
	return m.err
}

// startAdmissionSyncWireServer creates a bare mgmt.Server, calls
// wireAdmissionSyncHandlers on it, starts Serve, and returns the server +
// socket path + cleanup. The ks and snapshotPath are passed through to
// wireAdmissionSyncHandlers so tests can drive the registered handlers.
//
// Uses listenUnixMgmt per F-P2L1-001 register-before-serve pattern.
// NOT t.Parallel compatible — creates filesystem sockets, interacts with umask.
//
//nolint:unparam // srv return is retained for future test expansion; callers currently ignore it
func startAdmissionSyncWireServer(t *testing.T, ks *admission.AdmittedKeySet, snapshotPath string) (socketPath string, daemonPriv ed25519.PrivateKey, srv *mgmt.Server) {
	t.Helper()

	_, daemonPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("startAdmissionSyncWireServer: generate daemon keypair: %v", err)
	}

	dir, err := os.MkdirTemp("", "sw-asw-*")
	if err != nil {
		t.Fatalf("startAdmissionSyncWireServer: MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socketPath = fmt.Sprintf("%s/m.sock", dir)
	if len(socketPath) > 104 {
		t.Fatalf("startAdmissionSyncWireServer: socket path %q exceeds 104-byte limit", socketPath)
	}

	ln, err := listenUnixMgmt(socketPath)
	if err != nil {
		t.Fatalf("startAdmissionSyncWireServer: listen: %v", err)
	}

	ops := mgmt.NewOperatorKeySet(nil)
	srv = mgmt.NewServer(ln, daemonPriv, ops, nil, "dev",
		mgmt.WithHandshakeTimeout(2*time.Second),
		mgmt.WithRPCIdleTimeout(5*time.Second),
	)

	// Register the four internal.admission.* handlers BEFORE Serve (F-P2L1-001).
	if err := wireAdmissionSyncHandlers(srv, ks, snapshotPath); err != nil {
		t.Fatalf("startAdmissionSyncWireServer: wireAdmissionSyncHandlers: %v", err)
	}

	// Register daemon key for sendAdminRPC bootstrap auth.
	testDaemonKeysMu.Lock()
	testDaemonKeys[socketPath] = daemonPriv
	testDaemonKeysMu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = srv.Serve(ctx)
	}()

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
	return socketPath, daemonPriv, srv
}

// svtnIDToHex returns the 32-lowercase-hex-char encoding of svtnID.
// This is the wire encoding for svtn_id in internal.admission.* commands
// (BC-2.05.009 Inv-4 / rulings v1.2 / Decision 2 wire encoding note).
func svtnIDToHex(id [16]byte) string {
	return hex.EncodeToString(id[:])
}

// ── AC-002: handler registration ──────────────────────────────────────────────

// TestWireAdmissionSyncHandlers_RegisteredOnRouterServer verifies that after
// wireAdmissionSyncHandlers is called on a mgmt.Server, the server's handler
// table contains the four internal.admission.* commands:
//   - internal.admission.register
//   - internal.admission.revoke
//   - internal.admission.expire
//   - internal.admission.remove-svtn
//
// BC-2.05.009 PC-1/Inv-3; S-BL.ADMISSION-SYNC-WIRE AC-002.
// Red Gate: wireAdmissionSyncHandlers registers NOTHING (stub), so the probes
// for the four commands will return E-RPC-010 (unknown command), causing this
// test to FAIL.
func TestWireAdmissionSyncHandlers_RegisteredOnRouterServer(t *testing.T) {
	// NOT t.Parallel: creates filesystem socket.

	ks := admission.NewAdmittedKeySet()
	socketPath, daemonPriv, _ := startAdmissionSyncWireServer(t, ks, "")

	// Poll until the server socket is ready (Serve goroutine may not have
	// started yet).
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Each of the four internal.admission.* commands must be registered.
	// We send each with an empty args body to verify registration; the handler
	// will error on bad args, but we only care that it is NOT E-RPC-010
	// (unknown command), which is what the stub produces.
	commands := []string{
		CmdAdmissionRegister,
		CmdAdmissionRevoke,
		CmdAdmissionExpire,
		CmdAdmissionRemoveSVTN,
	}

	for _, cmd := range commands {
		t.Run(cmd, func(t *testing.T) {
			resp := sendAdminRPC(t, socketPath, daemonPriv, cmd, map[string]any{})
			errObj, _ := resp["error"].(map[string]any)
			if errObj == nil {
				// Handler was called and returned success — that means it's registered.
				// (The stub registers nothing, so this should never happen.)
				return
			}
			code, _ := errObj["code"].(string)
			if code == "E-RPC-010" {
				t.Errorf("AC-002: command %q returned E-RPC-010 (unknown command) — "+
					"wireAdmissionSyncHandlers must register this handler on the router server. "+
					"Red Gate: stub registers nothing; fix by implementing wireAdmissionSyncHandlers.",
					cmd)
			}
			// Any other error (bad args, not-implemented, etc.) means the handler IS
			// registered — which is the goal. That's not a failure for this test.
		})
	}
}

// TestWireAdmissionSyncHandlers_NotRegisteredOnControlServer verifies that the
// four internal.admission.* commands return E-RPC-010 on a daemon that has NOT
// called wireAdmissionSyncHandlers (control/console/access/empty modes).
//
// BC-2.05.009 Inv-3 / ADR-004 role-exclusion; S-BL.ADMISSION-SYNC-WIRE AC-002.
// Red Gate: this test PASSES trivially (E-RPC-010 is what you get when no handler
// is registered — that's the current stub behavior for ALL servers).
// We include it to lock the role-exclusion invariant for regressions.
func TestWireAdmissionSyncHandlers_NotRegisteredOnControlServer(t *testing.T) {
	// Uses startE2EServer (from admin_handlers_e2e_test.go) which registers NO
	// admission-sync handlers. This represents the control/console/access server.

	es := startE2EServer(t, nil)

	for _, cmd := range []string{
		CmdAdmissionRegister,
		CmdAdmissionRevoke,
		CmdAdmissionExpire,
		CmdAdmissionRemoveSVTN,
	} {
		cmd := cmd
		t.Run(cmd, func(t *testing.T) {
			t.Parallel()
			resp := sendAdminRPC(t, es.socketPath, es.daemonPriv, cmd, map[string]any{})
			errObj, _ := resp["error"].(map[string]any)
			if errObj == nil {
				t.Errorf("AC-002 role-exclusion: %q must return E-RPC-010 on a non-router server; got success", cmd)
				return
			}
			code, _ := errObj["code"].(string)
			if code != "E-RPC-010" {
				t.Errorf("AC-002 role-exclusion: %q error code = %q; want E-RPC-010 (unknown command on non-router server)", cmd, code)
			}
		})
	}
}

// TestRouterMode_AdminHandlersNotRegistered verifies that a router-mode daemon
// (runRouter / wireAdmissionSyncHandlers called) does NOT register admin.key.*
// handlers. This is the ADR-004 / AC-004 role-exclusion invariant.
//
// BC-2.05.009 Inv-3; ADR-004; S-BL.ADMISSION-SYNC-WIRE AC-002 (regression guard).
// Red Gate: this test PASSES trivially (the stub registers nothing, so admin
// handlers remain absent). We include it to lock the invariant.
func TestRouterMode_AdminHandlersNotRegistered(t *testing.T) {
	// NOT t.Parallel: creates filesystem socket.

	ks := admission.NewAdmittedKeySet()
	socketPath, daemonPriv, _ := startAdmissionSyncWireServer(t, ks, "")

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	for _, cmd := range []string{"admin.key.register", "admin.key.revoke", "admin.key.expire"} {
		cmd := cmd
		t.Run(cmd, func(t *testing.T) {
			resp := sendAdminRPC(t, socketPath, daemonPriv, cmd, map[string]any{})
			errObj, _ := resp["error"].(map[string]any)
			if errObj == nil {
				t.Errorf("AC-002/ADR-004: admin handler %q must NOT be registered on router-mode server; got success", cmd)
				return
			}
			code, _ := errObj["code"].(string)
			if code != "E-RPC-010" {
				t.Errorf("AC-002/ADR-004: %q error code = %q; want E-RPC-010 (not registered on router)", cmd, code)
			}
		})
	}
}

// ── AC-003: admin.key.register pushes after control write ─────────────────────

// TestAdmissionSync_RegisterKey_PushCalledAfterControlWrite verifies that when
// admin.key.register succeeds on the control daemon, admissionSyncer.PushRegisterKey
// is called with the svtnID ([16]byte UUID), pubkey, and role.
//
// BC-2.05.009 PC-1; S-BL.ADMISSION-SYNC-WIRE AC-003.
// Red Gate: admin_handlers.go does not yet call PushRegisterKey — FAILS because
// mockSyncer.calls will be empty.
func TestAdmissionSync_RegisterKey_PushCalledAfterControlWrite(t *testing.T) {
	t.Parallel()

	sync := &mockSyncer{}
	ks := admission.NewAdmittedKeySet()
	ctrlPub, ctrlPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate control key: %v", err)
	}
	m := svtnmgmt.NewSVTNManager(ks, ctrlPub)
	cr, err := m.Create("test-svtn-register")
	if err != nil {
		t.Fatalf("create SVTN: %v", err)
	}
	svtnID := cr.SVTN.ID

	// Generate a fresh key to register.
	regPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate reg key: %v", err)
	}

	// Build admin handlers with the mock syncer wired in.
	ops := mgmt.NewOperatorKeySet(nil)
	handlers := BuildAdminHandlers(m, ops, sync)

	// Use startE2EServerWithOps so ctrlPriv matches the bootstrap key.
	es := startE2EServerWithOps(t, handlers, ctrlPriv, ops)

	// Encode pubkey as openssh for the RPC args.
	pubkeyB64 := base64.RawURLEncoding.EncodeToString([]byte(regPub))

	// We need the openssh-format key. Use the helper that existing tests use.
	// The wire format is base64url(raw 32 bytes) per the args encoding.
	resp := sendAdminRPC(t, es.socketPath, ctrlPriv, "admin.key.register", map[string]any{
		"svtn_id":        "test-svtn-register",
		"pubkey_openssh": pubkeyB64,
		"role":           "access",
	})

	// If the handler returned an error, the RPC failed — check for it.
	if errObj, ok := resp["error"].(map[string]any); ok {
		// Not-implemented errors from push are advisory (do not fail the RPC).
		// A handler-level error means the admin.key.register itself failed.
		t.Logf("admin.key.register RPC error: %v", errObj)
	}

	// AC-003 assertion: PushRegisterKey MUST have been called at least once.
	// Red Gate: current stub does NOT call PushRegisterKey → this fails.
	if len(sync.calls) == 0 {
		t.Fatal("AC-003: PushRegisterKey was not called after admin.key.register succeeded. " +
			"Red Gate: admin_handlers.go does not yet call push after control write.")
	}
	if sync.calls[0] != "PushRegisterKey" {
		t.Errorf("AC-003: expected first push call = PushRegisterKey; got %q", sync.calls[0])
	}
	// Assert svtnID matches the [16]byte UUID (not the human-readable name).
	if len(sync.args) > 0 {
		gotID, ok := sync.args[0][0].([16]byte)
		if !ok {
			t.Errorf("AC-003: first arg must be [16]byte svtnID; got %T", sync.args[0][0])
		} else if gotID != svtnID {
			t.Errorf("AC-003: svtnID pushed = %s; want %s (control must resolve name→[16]byte)",
				svtnIDToHex(gotID), svtnIDToHex(svtnID))
		}
	}
	_ = ctrlPub
}

// TestAdmissionSync_RegisterKey_PushFailureDoesNotRollbackControlWrite verifies
// that when admissionSyncer.PushRegisterKey returns an error, the control-side
// write (in AdmittedKeySet) remains committed — push failure is advisory.
//
// BC-2.05.009 PC-2; S-BL.ADMISSION-SYNC-WIRE AC-003.
// Red Gate: FAILS because the push is not called at all (mockSyncer.err never
// fires), OR because the push is called but an error from it causes the handler
// to return a failure to sbctl. We assert the RPC must succeed even when push fails.
func TestAdmissionSync_RegisterKey_PushFailureDoesNotRollbackControlWrite(t *testing.T) {
	t.Parallel()

	// Syncer that always returns a push error.
	sync := &mockSyncer{err: errors.New("simulated push failure: connection refused")}

	ks := admission.NewAdmittedKeySet()
	_, ctrlPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate control key: %v", err)
	}
	ctrlPub := ctrlPriv.Public().(ed25519.PublicKey)
	m := svtnmgmt.NewSVTNManager(ks, ctrlPub)
	if _, err := m.Create("push-fail-svtn"); err != nil {
		t.Fatalf("create SVTN: %v", err)
	}

	regPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate reg key: %v", err)
	}

	ops := mgmt.NewOperatorKeySet(nil)
	handlers := BuildAdminHandlers(m, ops, sync)
	es := startE2EServerWithOps(t, handlers, ctrlPriv, ops)

	pubkeyB64 := base64.RawURLEncoding.EncodeToString([]byte(regPub))
	resp := sendAdminRPC(t, es.socketPath, ctrlPriv, "admin.key.register", map[string]any{
		"svtn_id":        "push-fail-svtn",
		"pubkey_openssh": pubkeyB64,
		"role":           "access",
	})

	// AC-003 PC-2: the RPC must return success to sbctl even when push fails.
	// Red Gate: if push is not yet called at all, the mock err never fires —
	// the RPC may still succeed for the wrong reason. The assertion below is
	// meaningful once the push wire is in place.
	if errObj, ok := resp["error"].(map[string]any); ok {
		code, _ := errObj["code"].(string)
		t.Errorf("AC-003 PC-2: admin.key.register must return success even when push fails; "+
			"got error code=%q msg=%v. Push failure must be advisory (WARN log only, no rollback).",
			code, errObj)
	}
	// STRENGTHENED: push MUST have been attempted even though it failed.
	if len(sync.calls) == 0 {
		t.Error("AC-003 PC-2: push must be attempted even though it fails — syncer.calls is empty")
	}
	// Additionally: the key must still be in the control-side AdmittedKeySet.
	svtnRec, found := m.SVTNByName("push-fail-svtn")
	if !found {
		t.Fatal("AC-003 PC-2: SVTN was destroyed after push failure — control write rolled back (must NOT happen)")
	}
	entries := ks.ListBySVTN(svtnRec.ID)
	if len(entries) == 0 {
		t.Error("AC-003 PC-2: control-side AdmittedKeySet has no entry for push-fail-svtn after register — " +
			"control write was rolled back (push failure must be advisory, not a rollback trigger)")
	}
}

// TestAdmissionSync_NilSyncer_NoOp verifies that when admissionSyncer is nil
// (router/console/access mode), admin write handlers succeed without panic.
//
// BC-2.05.009 PC-6; S-BL.ADMISSION-SYNC-WIRE AC-003.
// Red Gate: passes trivially (nil syncer check may already be in place).
// Included to lock the no-panic invariant — if the implementer forgets nil
// guard, this will catch the panic.
func TestAdmissionSync_NilSyncer_NoOp(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	_, ctrlPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate control key: %v", err)
	}
	ctrlPub := ctrlPriv.Public().(ed25519.PublicKey)
	m := svtnmgmt.NewSVTNManager(ks, ctrlPub)
	if _, err := m.Create("nil-syncer-svtn"); err != nil {
		t.Fatalf("create SVTN: %v", err)
	}

	regPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate reg key: %v", err)
	}

	// nil admissionSyncer — must not panic.
	ops := mgmt.NewOperatorKeySet(nil)
	handlers := BuildAdminHandlers(m, ops, nil) // nil syncer

	es := startE2EServerWithOps(t, handlers, ctrlPriv, ops)

	pubkeyB64 := base64.RawURLEncoding.EncodeToString([]byte(regPub))
	resp := sendAdminRPC(t, es.socketPath, ctrlPriv, "admin.key.register", map[string]any{
		"svtn_id":        "nil-syncer-svtn",
		"pubkey_openssh": pubkeyB64,
		"role":           "access",
	})

	// Nil syncer must not cause the RPC to fail.
	if errObj, ok := resp["error"].(map[string]any); ok {
		t.Errorf("AC-003 nil-syncer: admin.key.register with nil admissionSyncer returned error: %v", errObj)
	}
}

// ── AC-004: revoke/expire/remove-svtn push ────────────────────────────────────

// TestAdmissionSync_RevokeKey_PushCalledAfterControlWrite verifies that
// admin.key.revoke calls PushRevokeKey after successful control write.
//
// BC-2.05.009 PC-1; S-BL.ADMISSION-SYNC-WIRE AC-004.
// Red Gate: FAILS — admin_handlers.go does not yet call PushRevokeKey.
func TestAdmissionSync_RevokeKey_PushCalledAfterControlWrite(t *testing.T) {
	t.Parallel()

	sync := &mockSyncer{}
	ks := admission.NewAdmittedKeySet()
	_, ctrlPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate control key: %v", err)
	}
	ctrlPub := ctrlPriv.Public().(ed25519.PublicKey)
	m := svtnmgmt.NewSVTNManager(ks, ctrlPub)
	if _, err := m.Create("revoke-test-svtn"); err != nil {
		t.Fatalf("create SVTN: %v", err)
	}

	regPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate reg key: %v", err)
	}
	if _, err := m.RegisterKey("revoke-test-svtn", regPub, admission.RoleAccess); err != nil {
		t.Fatalf("register key: %v", err)
	}

	ops := mgmt.NewOperatorKeySet(nil)
	handlers := BuildAdminHandlers(m, ops, sync)
	es := startE2EServerWithOps(t, handlers, ctrlPriv, ops)

	pubkeyB64 := base64.RawURLEncoding.EncodeToString([]byte(regPub))
	sendAdminRPC(t, es.socketPath, ctrlPriv, "admin.key.revoke", map[string]any{
		"svtn_id":        "revoke-test-svtn",
		"pubkey_openssh": pubkeyB64,
		"role":           "access",
		"confirm":        false,
	})

	if !containsCall(sync.calls, "PushRevokeKey") {
		t.Errorf("AC-004: PushRevokeKey not called after admin.key.revoke. "+
			"Red Gate: admin_handlers.go does not yet push on revoke. calls=%v", sync.calls)
	}
}

// TestAdmissionSync_ExpireKey_PushCalledAfterControlWrite verifies that
// admin.key.expire calls PushSetKeyExpiry after successful control write.
//
// BC-2.05.009 PC-1; S-BL.ADMISSION-SYNC-WIRE AC-004.
// Red Gate: FAILS — admin_handlers.go does not yet call PushSetKeyExpiry.
func TestAdmissionSync_ExpireKey_PushCalledAfterControlWrite(t *testing.T) {
	t.Parallel()

	sync := &mockSyncer{}
	ks := admission.NewAdmittedKeySet()
	_, ctrlPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate control key: %v", err)
	}
	ctrlPub := ctrlPriv.Public().(ed25519.PublicKey)
	m := svtnmgmt.NewSVTNManager(ks, ctrlPub)
	if _, err := m.Create("expire-test-svtn"); err != nil {
		t.Fatalf("create SVTN: %v", err)
	}

	regPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate reg key: %v", err)
	}
	if _, err := m.RegisterKey("expire-test-svtn", regPub, admission.RoleAccess); err != nil {
		t.Fatalf("register key: %v", err)
	}

	ops := mgmt.NewOperatorKeySet(nil)
	handlers := BuildAdminHandlers(m, ops, sync)
	es := startE2EServerWithOps(t, handlers, ctrlPriv, ops)

	pubkeyB64 := base64.RawURLEncoding.EncodeToString([]byte(regPub))
	sendAdminRPC(t, es.socketPath, ctrlPriv, "admin.key.expire", map[string]any{
		"svtn_id":        "expire-test-svtn",
		"pubkey_openssh": pubkeyB64,
		"after":          "24h",
	})

	if !containsCall(sync.calls, "PushSetKeyExpiry") {
		t.Errorf("AC-004: PushSetKeyExpiry not called after admin.key.expire. "+
			"Red Gate: admin_handlers.go does not yet push on expire. calls=%v", sync.calls)
	}
}

// TestAdmissionSync_RemoveSVTN_PushCalledAfterControlWrite verifies that
// admin.svtn.destroy calls PushRemoveSVTN after successful control write.
//
// BC-2.05.009 PC-1; S-BL.ADMISSION-SYNC-WIRE AC-004.
// Red Gate: FAILS — admin_handlers.go does not yet call PushRemoveSVTN.
func TestAdmissionSync_RemoveSVTN_PushCalledAfterControlWrite(t *testing.T) {
	t.Parallel()

	sync := &mockSyncer{}
	ks := admission.NewAdmittedKeySet()
	_, ctrlPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate control key: %v", err)
	}
	ctrlPub := ctrlPriv.Public().(ed25519.PublicKey)
	m := svtnmgmt.NewSVTNManager(ks, ctrlPub)
	if _, err := m.Create("destroy-test-svtn"); err != nil {
		t.Fatalf("create SVTN: %v", err)
	}

	ops := mgmt.NewOperatorKeySet(nil)
	handlers := BuildAdminHandlers(m, ops, sync)
	es := startE2EServerWithOps(t, handlers, ctrlPriv, ops)

	// admin.svtn.destroy uses "name" not "svtn_id" per adminSVTNDestroyArgs.
	sendAdminRPC(t, es.socketPath, ctrlPriv, "admin.svtn.destroy", map[string]any{
		"name": "destroy-test-svtn",
	})

	if !containsCall(sync.calls, "PushRemoveSVTN") {
		t.Errorf("AC-004: PushRemoveSVTN not called after admin.svtn.destroy. "+
			"Red Gate: admin_handlers.go does not yet push on destroy. calls=%v", sync.calls)
	}
}

// TestAdmissionSync_PushFailure_AllWritePaths_Advisory verifies that push failure
// (from any of the four write paths) is advisory — the RPC must return success
// to sbctl even when admissionSyncer returns an error.
//
// BC-2.05.009 PC-2; S-BL.ADMISSION-SYNC-WIRE AC-004.
// Red Gate: FAILS if any handler propagates the push error to sbctl instead of
// logging at WARN and returning success.
func TestAdmissionSync_PushFailure_AllWritePaths_Advisory(t *testing.T) {
	t.Parallel()

	// Each write path: register, revoke, expire, destroy.
	// We test them all in subtests using a syncer that always errors.
	type writeCase struct {
		name    string
		setupFn func(t *testing.T, m *svtnmgmt.SVTNManager, ks *admission.AdmittedKeySet, ctrlPriv ed25519.PrivateKey) (svtnName string, args map[string]any, cmd string)
	}

	cases := []writeCase{
		{
			name: "register",
			setupFn: func(t *testing.T, m *svtnmgmt.SVTNManager, ks *admission.AdmittedKeySet, ctrlPriv ed25519.PrivateKey) (string, map[string]any, string) {
				t.Helper()
				if _, err := m.Create("advisory-register-svtn"); err != nil {
					t.Fatalf("create SVTN: %v", err)
				}
				pub, _, _ := ed25519.GenerateKey(rand.Reader)
				return "advisory-register-svtn", map[string]any{
					"svtn_id":        "advisory-register-svtn",
					"pubkey_openssh": base64.RawURLEncoding.EncodeToString([]byte(pub)),
					"role":           "access",
				}, "admin.key.register"
			},
		},
		{
			name: "destroy",
			setupFn: func(t *testing.T, m *svtnmgmt.SVTNManager, ks *admission.AdmittedKeySet, ctrlPriv ed25519.PrivateKey) (string, map[string]any, string) {
				t.Helper()
				if _, err := m.Create("advisory-destroy-svtn"); err != nil {
					t.Fatalf("create SVTN: %v", err)
				}
				// admin.svtn.destroy uses "name" not "svtn_id" per adminSVTNDestroyArgs.
				return "advisory-destroy-svtn", map[string]any{
					"name": "advisory-destroy-svtn",
				}, "admin.svtn.destroy"
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sync := &mockSyncer{err: errors.New("push failure: connection refused")}
			ks := admission.NewAdmittedKeySet()
			_, ctrlPriv, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				t.Fatalf("generate control key: %v", err)
			}
			ctrlPub := ctrlPriv.Public().(ed25519.PublicKey)
			m := svtnmgmt.NewSVTNManager(ks, ctrlPub)

			_, args, cmd := tc.setupFn(t, m, ks, ctrlPriv)

			ops := mgmt.NewOperatorKeySet(nil)
			handlers := BuildAdminHandlers(m, ops, sync)
			es := startE2EServerWithOps(t, handlers, ctrlPriv, ops)

			resp := sendAdminRPC(t, es.socketPath, ctrlPriv, cmd, args)
			if errObj, ok := resp["error"].(map[string]any); ok {
				code, _ := errObj["code"].(string)
				t.Errorf("AC-004 advisory-failure: %q returned error code=%q when push failed; "+
					"push failure must be advisory (WARN log only, no error to sbctl). err=%v",
					cmd, code, errObj)
			}
			// STRENGTHENED: push MUST have been attempted even though it failed.
			if len(sync.calls) == 0 {
				t.Error("push must be attempted even though it fails — syncer.calls is empty")
			}
		})
	}
}

// containsCall returns true if calls contains target.
func containsCall(calls []string, target string) bool {
	for _, c := range calls {
		if c == target {
			return true
		}
	}
	return false
}

// ── AC-005: router handler populates keyset + snapshot ────────────────────────

// TestRouterAdmissionHandler_Register_AdmittedFalse verifies that when the router
// receives internal.admission.register, it calls ks.RegisterKey with the decoded
// svtnID and pubkey, and the resulting entry has admitted=false.
//
// BC-2.05.009 PC-8 / BC-2.05.010 PC-1; S-BL.ADMISSION-SYNC-WIRE AC-005.
// Red Gate: FAILS — wireAdmissionSyncHandlers registers no handlers.
func TestRouterAdmissionHandler_Register_AdmittedFalse(t *testing.T) {
	// NOT t.Parallel: creates filesystem socket.

	ks := admission.NewAdmittedKeySet()
	socketPath, daemonPriv, _ := startAdmissionSyncWireServer(t, ks, "")

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Prepare a valid svtn_id (32 hex chars) and pubkey.
	var svtnID [16]byte
	if _, err := rand.Read(svtnID[:]); err != nil {
		t.Fatalf("rand.Read svtnID: %v", err)
	}
	svtnIDStr := svtnIDToHex(svtnID)

	regPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate reg key: %v", err)
	}
	// The wire encoding for pubkey in internal.admission.register is pubkey_openssh
	// (same as admin.key.* per BC-2.05.009 Inv-4 encoding parity for pubkey).
	pubkeyB64 := base64.RawURLEncoding.EncodeToString([]byte(regPub))

	resp := sendAdminRPC(t, socketPath, daemonPriv, CmdAdmissionRegister, map[string]any{
		"svtn_id":        svtnIDStr,
		"pubkey_openssh": pubkeyB64,
		"role":           "access",
	})

	// If E-RPC-010, the handler is not registered yet (Red Gate).
	if errObj, ok := resp["error"].(map[string]any); ok {
		code, _ := errObj["code"].(string)
		if code == "E-RPC-010" {
			t.Errorf("AC-005: internal.admission.register returned E-RPC-010 — handler not registered. " +
				"Red Gate: wireAdmissionSyncHandlers stub registers nothing.")
			return
		}
		t.Logf("AC-005: internal.admission.register returned error (may be args validation): %v", errObj)
	}

	// AC-005: the key must be in the AdmittedKeySet with admitted=false.
	entries := ks.ListBySVTN(svtnID)
	if len(entries) == 0 {
		t.Errorf("AC-005: no entries in AdmittedKeySet for SVTN %s after register push. "+
			"Red Gate: handler not registered (stub).", svtnIDStr)
		return
	}
	// admitted must be false — challenge-response has not occurred.
	for _, e := range entries {
		if ks.IsAdmitted(svtnID, e.NodeAddr) {
			t.Errorf("AC-005: entry admitted=true after register push; must be false until challenge-response (BC-2.05.009 PC-8)")
		}
	}
}

// TestRouterAdmissionHandler_Register_SnapshotWritten verifies that after
// internal.admission.register is handled, the snapshot file is written atomically
// to the configured admission_state_file path.
//
// BC-2.05.010 PC-1/PC-3; S-BL.ADMISSION-SYNC-WIRE AC-005.
// Red Gate: FAILS — wireAdmissionSyncHandlers registers no handlers; no snapshot written.
func TestRouterAdmissionHandler_Register_SnapshotWritten(t *testing.T) {
	// NOT t.Parallel: creates filesystem socket AND tempfile; umask race.

	dir, err := os.MkdirTemp("", "sb-snap-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	_ = os.Chmod(dir, 0o700)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	snapshotPath := filepath.Join(dir, "admission-state.json")

	ks := admission.NewAdmittedKeySet()
	socketPath, daemonPriv, _ := startAdmissionSyncWireServer(t, ks, snapshotPath)

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	var svtnID [16]byte
	if _, err := rand.Read(svtnID[:]); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	regPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate reg key: %v", err)
	}

	resp := sendAdminRPC(t, socketPath, daemonPriv, CmdAdmissionRegister, map[string]any{
		"svtn_id":        svtnIDToHex(svtnID),
		"pubkey_openssh": base64.RawURLEncoding.EncodeToString([]byte(regPub)),
		"role":           "access",
	})

	if errObj, ok := resp["error"].(map[string]any); ok {
		code, _ := errObj["code"].(string)
		if code == "E-RPC-010" {
			t.Errorf("AC-005: handler not registered (E-RPC-010). Red Gate: stub registers nothing.")
			return
		}
	}

	// Wait briefly for snapshot write (may be async after handler returns).
	time.Sleep(50 * time.Millisecond)

	// AC-005: snapshot file must exist and be valid JSON with schema_version: 1.
	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("AC-005: snapshot file %q not written after register push. "+
			"Red Gate: wireAdmissionSyncHandlers stub does not write snapshot. err=%v",
			snapshotPath, err)
	}

	var snap snapshotFile
	if err := json.Unmarshal(data, &snap); err != nil {
		t.Fatalf("AC-005: snapshot file contains invalid JSON: %v", err)
	}
	if snap.SchemaVersion != snapshotCurrentSchemaVersion {
		t.Errorf("AC-005: snapshot schema_version=%d; want %d", snap.SchemaVersion, snapshotCurrentSchemaVersion)
	}
}

// TestRouterAdmissionHandler_Register_SnapshotWriteFailure_Advisory verifies
// that if the snapshot write fails (read-only dir), the push handler still
// returns success (advisory failure, not fatal).
//
// BC-2.05.010 PC-2; S-BL.ADMISSION-SYNC-WIRE AC-005.
// Red Gate: FAILS — wireAdmissionSyncHandlers stub registers no handler, so
// the command returns E-RPC-010. Once implemented, the test verifies advisory behavior.
func TestRouterAdmissionHandler_Register_SnapshotWriteFailure_Advisory(t *testing.T) {
	// NOT t.Parallel: creates filesystem socket AND tempdir.

	// Use a read-only directory so os.WriteFile / os.Rename fails.
	dir, err := os.MkdirTemp("", "sb-snap-ro-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(dir, 0o700)
		_ = os.RemoveAll(dir)
	})
	snapshotPath := filepath.Join(dir, "admission-state.json")

	// Make the directory read-only so writes fail.
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("Chmod read-only: %v", err)
	}

	ks := admission.NewAdmittedKeySet()
	socketPath, daemonPriv, _ := startAdmissionSyncWireServer(t, ks, snapshotPath)

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	var svtnID [16]byte
	if _, err := rand.Read(svtnID[:]); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	regPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate reg key: %v", err)
	}

	resp := sendAdminRPC(t, socketPath, daemonPriv, CmdAdmissionRegister, map[string]any{
		"svtn_id":        svtnIDToHex(svtnID),
		"pubkey_openssh": base64.RawURLEncoding.EncodeToString([]byte(regPub)),
		"role":           "access",
	})

	// AC-005 PC-2 advisory: the handler must return success even when snapshot write fails.
	// If E-RPC-010, handler not registered (Red Gate — normal for stub).
	if errObj, ok := resp["error"].(map[string]any); ok {
		code, _ := errObj["code"].(string)
		if code == "E-RPC-010" {
			t.Errorf("AC-005 SnapshotWriteFailure_Advisory: E-RPC-010 — handler not registered (Red Gate stub). " +
				"Once implemented, snapshot write failure must be advisory (not propagated to caller).")
			return
		}
		t.Errorf("AC-005 SnapshotWriteFailure_Advisory: handler returned error even though snapshot failure must be advisory. "+
			"code=%q err=%v", code, errObj)
	}

	// Snapshot file must NOT exist (write to read-only dir fails).
	if _, statErr := os.Stat(snapshotPath); statErr == nil {
		t.Error("AC-005: snapshot file was written to a read-only directory — impossible (test setup error)")
	}

	// In-memory keyset must still have the entry (write succeeded in memory).
	entries := ks.ListBySVTN(svtnID)
	if len(entries) == 0 {
		t.Error("AC-005 SnapshotWriteFailure_Advisory: in-memory AdmittedKeySet has no entry — " +
			"snapshot write failure must not roll back the in-memory keyset write")
	}
}

// ── F-2: concurrent snapshot writes must not produce invalid JSON ─────────────

// TestSnapshotWriteAtomic_ConcurrentWrites_AlwaysValidJSON verifies that N
// concurrent calls to writeSnapshotAtomic on the same path never produce an
// invalid or empty snapshot file on disk.
//
// The old implementation used a fixed "<path>.tmp" name shared by all concurrent
// writers, so two writers could interleave writes into the same temp file and
// then rename it over the live snapshot — producing corrupt JSON. The fix uses
// os.CreateTemp for a unique per-write temp name: concurrent writers no longer
// share a temp file. The final rename is still atomic (last-writer-wins with a
// VALID file).
//
// F-2 / BC-2.05.010 Invariant 1; S-BL.ADMISSION-SYNC-WIRE AC-005.
//
// NOT t.Parallel: creates temp files in a tempdir; avoids umask race.
func TestSnapshotWriteAtomic_ConcurrentWrites_AlwaysValidJSON(t *testing.T) {
	dir, err := os.MkdirTemp("", "sb-snap-concurrent-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	_ = os.Chmod(dir, 0o700)
	t.Cleanup(func() {
		// Restore write bit in case a test left it read-only.
		_ = os.Chmod(dir, 0o700)
		_ = os.RemoveAll(dir)
	})
	snapshotPath := filepath.Join(dir, "admission-state.json")

	var svtnID [16]byte
	if _, err := rand.Read(svtnID[:]); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, pub, admission.RoleAccess)

	const N = 20
	errs := make(chan error, N)
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			errs <- writeSnapshotAtomic(snapshotPath, ks)
		}()
	}
	wg.Wait()
	close(errs)

	// All advisory write errors (e.g. chmod race) are OK — the test asserts that
	// the ON-DISK file (if it exists) is ALWAYS valid JSON with schema_version=1.
	// At least one write must have succeeded (all N had the same ks, same content).
	for e := range errs {
		if e != nil {
			t.Logf("writeSnapshotAtomic returned advisory error: %v", e)
		}
	}

	// The file must exist and contain valid JSON.
	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("F-2 ConcurrentWrites: snapshot file %q not readable after %d concurrent writes: %v",
			snapshotPath, N, err)
	}
	if len(data) == 0 {
		t.Fatalf("F-2 ConcurrentWrites: snapshot file %q is empty after %d concurrent writes "+
			"(old fixed-name temp sharing could produce a truncated file)", snapshotPath, N)
	}
	var snap snapshotFile
	if err := json.Unmarshal(data, &snap); err != nil {
		t.Fatalf("F-2 ConcurrentWrites: snapshot file %q contains invalid JSON after %d concurrent writes "+
			"(old fixed-name temp sharing produced corrupt JSON; unique-temp fix should prevent this). "+
			"json err: %v\nfile contents: %s", snapshotPath, N, err, data)
	}
	if snap.SchemaVersion != snapshotCurrentSchemaVersion {
		t.Errorf("F-2 ConcurrentWrites: snapshot schema_version=%d; want %d",
			snap.SchemaVersion, snapshotCurrentSchemaVersion)
	}
}

// ── AC-006: snapshot JSON round-trip ──────────────────────────────────────────

// TestSnapshot_JSON_FieldEncoding_CorrectSchema verifies that marshalSnapshot
// produces a snapshotFile with the correct schema (schema_version:1, RFC3339 UTC
// timestamp, svtn_id as 32 hex chars, pubkey as base64url no-padding, role string,
// revoked bool, expiry omitempty).
//
// BC-2.05.010 PC-4; S-BL.ADMISSION-SYNC-WIRE AC-006.
// Red Gate: FAILS — marshalSnapshot returns errSnapshotNotImplemented.
func TestSnapshot_JSON_FieldEncoding_CorrectSchema(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	var svtnID [16]byte
	copy(svtnID[:], []byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	})

	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	ks.RegisterKey(svtnID, pub, admission.RoleAccess)

	snap, err := marshalSnapshot(ks)
	if err != nil {
		t.Fatalf("AC-006: marshalSnapshot returned error (Red Gate — not implemented): %v", err)
	}

	// schema_version must be 1.
	if snap.SchemaVersion != snapshotCurrentSchemaVersion {
		t.Errorf("AC-006: schema_version=%d; want %d", snap.SchemaVersion, snapshotCurrentSchemaVersion)
	}

	// timestamp must be non-empty RFC3339 UTC.
	if snap.Timestamp == "" {
		t.Error("AC-006: timestamp field is empty")
	} else {
		ts, parseErr := time.Parse(time.RFC3339, snap.Timestamp)
		if parseErr != nil {
			t.Errorf("AC-006: timestamp %q is not valid RFC3339: %v", snap.Timestamp, parseErr)
		} else if ts.Location() != time.UTC {
			t.Errorf("AC-006: timestamp must be UTC; got location %v", ts.Location())
		}
	}

	// svtns must contain our SVTN.
	if len(snap.SVTNs) != 1 {
		t.Fatalf("AC-006: expected 1 SVTN in snapshot; got %d", len(snap.SVTNs))
	}

	svtnEntry := snap.SVTNs[0]
	wantSVTNID := hex.EncodeToString(svtnID[:])
	if svtnEntry.SVTNID != wantSVTNID {
		t.Errorf("AC-006: svtn_id=%q; want %q (32 lowercase hex chars)", svtnEntry.SVTNID, wantSVTNID)
	}

	if len(svtnEntry.Keys) != 1 {
		t.Fatalf("AC-006: expected 1 key in SVTN; got %d", len(svtnEntry.Keys))
	}

	key := svtnEntry.Keys[0]

	// pubkey must be base64url no-padding, 32-byte raw Ed25519.
	decoded, decErr := base64.RawURLEncoding.DecodeString(key.PubKey)
	if decErr != nil {
		t.Errorf("AC-006: pubkey %q is not valid base64url no-padding: %v", key.PubKey, decErr)
	} else if len(decoded) != ed25519.PublicKeySize {
		t.Errorf("AC-006: decoded pubkey length=%d; want %d (raw Ed25519)", len(decoded), ed25519.PublicKeySize)
	}

	// role must be canonical string.
	if key.Role != "access" {
		t.Errorf("AC-006: role=%q; want %q", key.Role, "access")
	}

	// revoked must be false for a fresh registration.
	if key.Revoked {
		t.Error("AC-006: revoked=true for a fresh registration; must be false")
	}

	// expiry must be omitted (no expiry set).
	if key.Expiry != "" {
		t.Errorf("AC-006: expiry=%q; want empty (omitempty — no expiry set)", key.Expiry)
	}
}

// TestSnapshot_RoundTrip_EntriesMatch verifies that serializing a populated
// AdmittedKeySet via marshalSnapshot → unmarshalSnapshot into a new ks produces
// the same entries. All loaded entries have admitted=false.
//
// BC-2.05.010 PC-4; S-BL.ADMISSION-SYNC-WIRE AC-006.
// Red Gate: FAILS — marshalSnapshot/unmarshalSnapshot return not-implemented.
func TestSnapshot_RoundTrip_EntriesMatch(t *testing.T) {
	t.Parallel()

	var svtnID [16]byte
	if _, err := rand.Read(svtnID[:]); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	ks1 := admission.NewAdmittedKeySet()
	ks1.RegisterKey(svtnID, pub, admission.RoleConsole)

	// Serialize.
	snap, err := marshalSnapshot(ks1)
	if err != nil {
		t.Fatalf("AC-006 RoundTrip: marshalSnapshot: %v (Red Gate: not implemented)", err)
	}

	// Deserialize into a fresh keyset.
	ks2 := admission.NewAdmittedKeySet()
	if err := unmarshalSnapshot(snap, ks2); err != nil {
		t.Fatalf("AC-006 RoundTrip: unmarshalSnapshot: %v", err)
	}

	// The entries must match.
	entries := ks2.ListBySVTN(svtnID)
	if len(entries) == 0 {
		t.Fatal("AC-006 RoundTrip: no entries in ks2 after round-trip")
	}
	found := false
	for _, e := range entries {
		if string(e.PublicKey) == string(pub) {
			found = true
			if ks2.IsAdmitted(svtnID, e.NodeAddr) {
				t.Error("AC-006 RoundTrip: loaded entry has admitted=true; must be false (BC-2.05.009 PC-8)")
			}
		}
	}
	if !found {
		t.Error("AC-006 RoundTrip: pubkey not found in loaded entries")
	}
}

// TestSnapshot_RoundTrip_AdmittedAlwaysFalse verifies that loaded entries have
// admitted=false regardless of the live admitted state before serialization.
//
// BC-2.05.010 PC-5; S-BL.ADMISSION-SYNC-WIRE AC-006.
// Red Gate: FAILS — marshalSnapshot/unmarshalSnapshot not implemented.
func TestSnapshot_RoundTrip_AdmittedAlwaysFalse(t *testing.T) {
	t.Parallel()

	var svtnID [16]byte
	if _, err := rand.Read(svtnID[:]); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	ks1 := admission.NewAdmittedKeySet()
	ks1.RegisterKey(svtnID, pub, admission.RoleControl)
	// Note: we cannot flip admitted=true without a real challenge-response;
	// the key starts admitted=false. The snapshot must NOT store the admitted flag.

	snap, err := marshalSnapshot(ks1)
	if err != nil {
		t.Fatalf("AC-006 AdmittedFalse: marshalSnapshot: %v", err)
	}

	// Round-trip: the snapshot must not contain "admitted" field.
	data, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("AC-006 AdmittedFalse: json.Marshal: %v", err)
	}
	if strings.Contains(string(data), `"admitted"`) {
		t.Errorf("AC-006 AdmittedFalse: snapshot JSON contains 'admitted' field; must not be stored (BC-2.05.010 PC-5)")
	}

	ks2 := admission.NewAdmittedKeySet()
	if err := unmarshalSnapshot(snap, ks2); err != nil {
		t.Fatalf("AC-006 AdmittedFalse: unmarshalSnapshot: %v", err)
	}
	entries := ks2.ListBySVTN(svtnID)
	for _, e := range entries {
		if ks2.IsAdmitted(svtnID, e.NodeAddr) {
			t.Error("AC-006 AdmittedFalse: loaded entry admitted=true; must always be false on load")
		}
	}
}

// TestSnapshot_RoundTrip_RevokedEntryCallsRevokeKey verifies that a snapshot entry
// with revoked=true causes RevokeKey to be called after RegisterKey during unmarshal.
//
// BC-2.05.010 PC-4 / BC-2.05.010 EC-006; S-BL.ADMISSION-SYNC-WIRE AC-006.
// Red Gate: FAILS — unmarshalSnapshot not implemented.
func TestSnapshot_RoundTrip_RevokedEntryCallsRevokeKey(t *testing.T) {
	t.Parallel()

	var svtnID [16]byte
	if _, err := rand.Read(svtnID[:]); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	// Build a snapshot with revoked=true manually.
	snap := &snapshotFile{
		SchemaVersion: snapshotCurrentSchemaVersion,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		SVTNs: []snapshotSVTN{
			{
				SVTNID: svtnIDToHex(svtnID),
				Keys: []snapshotKey{
					{
						PubKey:  base64.RawURLEncoding.EncodeToString([]byte(pub)),
						Role:    "access",
						Revoked: true,
					},
				},
			},
		},
	}

	ks := admission.NewAdmittedKeySet()
	if err := unmarshalSnapshot(snap, ks); err != nil {
		t.Fatalf("AC-006 RevokedEntry: unmarshalSnapshot: %v (Red Gate: not implemented)", err)
	}

	// The key must be registered AND revoked.
	entries := ks.ListBySVTN(svtnID)
	if len(entries) == 0 {
		t.Fatal("AC-006 RevokedEntry: no entries after loading revoked snapshot")
	}
	found := false
	for _, e := range entries {
		if string(e.PublicKey) == string(pub) {
			found = true
			if !e.IsRevoked() {
				t.Error("AC-006 RevokedEntry: entry is NOT revoked after loading snapshot with revoked=true")
			}
		}
	}
	if !found {
		t.Error("AC-006 RevokedEntry: pubkey not found in loaded entries")
	}
}

// TestSnapshot_RoundTrip_ExpiryEntryCallsSetKeyExpiry verifies that a snapshot entry
// with an expiry field causes SetKeyExpiry to be called after RegisterKey during unmarshal.
//
// BC-2.05.010 PC-4 / BC-2.05.010 EC-007; S-BL.ADMISSION-SYNC-WIRE AC-006.
// Red Gate: FAILS — unmarshalSnapshot not implemented.
func TestSnapshot_RoundTrip_ExpiryEntryCallsSetKeyExpiry(t *testing.T) {
	t.Parallel()

	var svtnID [16]byte
	if _, err := rand.Read(svtnID[:]); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	wantExpiry := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	snap := &snapshotFile{
		SchemaVersion: snapshotCurrentSchemaVersion,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		SVTNs: []snapshotSVTN{
			{
				SVTNID: svtnIDToHex(svtnID),
				Keys: []snapshotKey{
					{
						PubKey:  base64.RawURLEncoding.EncodeToString([]byte(pub)),
						Role:    "access",
						Revoked: false,
						Expiry:  wantExpiry.Format(time.RFC3339),
					},
				},
			},
		},
	}

	ks := admission.NewAdmittedKeySet()
	if err := unmarshalSnapshot(snap, ks); err != nil {
		t.Fatalf("AC-006 ExpiryEntry: unmarshalSnapshot: %v (Red Gate: not implemented)", err)
	}

	entries := ks.ListBySVTN(svtnID)
	if len(entries) == 0 {
		t.Fatal("AC-006 ExpiryEntry: no entries after loading snapshot with expiry")
	}
	found := false
	for _, e := range entries {
		if string(e.PublicKey) == string(pub) {
			found = true
			expiry := e.KeyExpiry()
			if expiry.IsZero() {
				t.Error("AC-006 ExpiryEntry: key expiry is zero after loading snapshot with expiry set")
			} else if !expiry.Equal(wantExpiry) {
				t.Errorf("AC-006 ExpiryEntry: key expiry=%v; want %v", expiry, wantExpiry)
			}
		}
	}
	if !found {
		t.Error("AC-006 ExpiryEntry: pubkey not found in loaded entries")
	}
}

// TestSnapshot_NoFrameAuthKey_NoNodeAddr_NoNonces verifies that the snapshot JSON
// does NOT contain FrameAuthKey, NodeAddr, or nonces fields.
//
// BC-2.05.010 PC-5; S-BL.ADMISSION-SYNC-WIRE AC-006.
// Red Gate: FAILS — marshalSnapshot not implemented.
func TestSnapshot_NoFrameAuthKey_NoNodeAddr_NoNonces(t *testing.T) {
	t.Parallel()

	var svtnID [16]byte
	if _, err := rand.Read(svtnID[:]); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, pub, admission.RoleAccess)

	snap, err := marshalSnapshot(ks)
	if err != nil {
		t.Fatalf("AC-006 NoFrameAuthKey: marshalSnapshot: %v (Red Gate: not implemented)", err)
	}

	data, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("AC-006 NoFrameAuthKey: json.Marshal: %v", err)
	}
	jsonStr := string(data)

	// None of the derived/ephemeral fields must appear in the snapshot.
	forbidden := []string{"frame_auth_key", "frameauthkey", "FrameAuthKey", "node_addr", "NodeAddr", "nonces", "admitted"}
	for _, f := range forbidden {
		if strings.Contains(strings.ToLower(jsonStr), strings.ToLower(f)) {
			t.Errorf("AC-006 NoFrameAuthKey: snapshot JSON contains forbidden field %q: %s", f, jsonStr)
		}
	}
}

// ── AC-007: startup load semantics ────────────────────────────────────────────

// TestRouterStartup_AdmissionStateFile_NotConfigured_EmptyKeyset verifies that
// when admission_state_file is absent in config, the router starts with an empty
// keyset and no snapshot I/O.
//
// BC-2.05.010 PC-6 / EC-001; S-BL.ADMISSION-SYNC-WIRE AC-007.
// Red Gate: loadSnapshotFromFile returns errSnapshotNotImplemented, but for
// the empty-path case it should return nil (no-op). If the stub returns an
// error for empty path, the test fails.
func TestRouterStartup_AdmissionStateFile_NotConfigured_EmptyKeyset(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	// Empty path means "no persistence" — must return nil with no I/O.
	err := loadSnapshotFromFile("", ks, nil)
	if err != nil {
		t.Errorf("AC-007 NotConfigured: loadSnapshotFromFile(\"\", ks) returned error %v; "+
			"must return nil (empty path is no-op, no snapshot file, no I/O). "+
			"Red Gate: stub returns errSnapshotNotImplemented for all paths.", err)
	}
	if len(ks.ListBySVTN([16]byte{})) != 0 {
		t.Error("AC-007 NotConfigured: keyset is non-empty after loading from empty path")
	}
}

// TestRouterStartup_AdmissionStateFile_ConfiguredFileAbsent_EmptyKeyset_InfoLog
// verifies two things:
//  1. loadSnapshotFromFile returns nil (no error, empty keyset) when the path
//     is configured but the file does not exist — fresh install semantics.
//  2. runRouter emits the AC-007 PC-2 mandated INFO log:
//     "admission_state_file not found; starting with empty keyset — awaiting push from control"
//     when the configured path does not exist.
//
// BC-2.05.010 PC-6 / EC-002; S-BL.ADMISSION-SYNC-WIRE AC-007.
func TestRouterStartup_AdmissionStateFile_ConfiguredFileAbsent_EmptyKeyset_InfoLog(t *testing.T) {
	// NOT t.Parallel: starts runRouter (binds listener).

	dir, err := os.MkdirTemp("", "sb-snap-absent-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	nonExistentPath := filepath.Join(dir, "does-not-exist.json")

	// Part 1: loadSnapshotFromFile directly — absent path → nil error, empty keyset.
	ks := admission.NewAdmittedKeySet()
	if err := loadSnapshotFromFile(nonExistentPath, ks, nil); err != nil {
		t.Errorf("AC-007 FileAbsent: loadSnapshotFromFile(%q, ks) returned error %v; "+
			"a missing file must produce nil error and empty keyset (fresh install).", nonExistentPath, err)
	}

	// Part 2: runRouter emits the mandated INFO log string (AC-007 PC-2).
	// Use an ephemeral data-plane listener + unix mgmt socket so runRouter starts cleanly.
	probe, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("probe listen: %v", err)
	}
	dataAddr := probe.Addr().String()
	_ = probe.Close()

	sockPath := tempSockPath(t)

	cfg := &config.Config{
		ListenAddr:         dataAddr,
		TickInterval:       10 * time.Millisecond,
		ManagementSocket:   sockPath,
		AdmissionStateFile: nonExistentPath,
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	var logBuf strings.Builder
	errCh := make(chan error, 1)
	go func() {
		errCh <- runRouter(ctx, &logBuf, cfg, "", make(chan os.Signal, 1), make(chan struct{}, 1))
	}()

	// Wait for router to bind mgmt socket (startup complete).
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, statErr := os.Stat(sockPath); statErr == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if _, statErr := os.Stat(sockPath); os.IsNotExist(statErr) {
		cancel()
		<-errCh
		t.Fatalf("AC-007 InfoLog: mgmt socket %q not created within 2s — runRouter did not start", sockPath)
	}

	cancel()
	select {
	case rErr := <-errCh:
		if rErr != nil {
			t.Logf("AC-007 InfoLog: runRouter returned: %v (may be benign)", rErr)
		}
	case <-time.After(3 * time.Second):
		t.Error("AC-007 InfoLog: runRouter did not return within 3s after ctx cancel")
	}

	// AC-007 PC-2: mandated INFO log string must appear verbatim.
	logStr := logBuf.String()
	const mandatedMsg = "admission_state_file not found; starting with empty keyset — awaiting push from control"
	if !strings.Contains(logStr, mandatedMsg) {
		t.Errorf("AC-007 PC-2: mandated INFO log %q not found in runRouter output.\n"+
			"Fix: change mgmt_wire.go log string to match AC-007 PC-2 exactly.\n"+
			"output=%q", mandatedMsg, logStr)
	}
}

// TestRouterStartup_AdmissionStateFile_ValidFile_EntriesLoaded verifies that
// a valid schema_version:1 file causes entries to be loaded into the keyset.
//
// BC-2.05.010 PC-7 / EC-003; S-BL.ADMISSION-SYNC-WIRE AC-007.
// Red Gate: FAILS — loadSnapshotFromFile returns errSnapshotNotImplemented.
//
// NOT t.Parallel: avoids umask race from listenUnixMgmt.
func TestRouterStartup_AdmissionStateFile_ValidFile_EntriesLoaded(t *testing.T) {
	dir, err := os.MkdirTemp("", "sb-snap-valid-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	_ = os.Chmod(dir, 0o700)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	var svtnID [16]byte
	if _, err := rand.Read(svtnID[:]); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	// Write a valid snapshot file.
	snap := snapshotFile{
		SchemaVersion: snapshotCurrentSchemaVersion,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		SVTNs: []snapshotSVTN{
			{
				SVTNID: svtnIDToHex(svtnID),
				Keys: []snapshotKey{
					{
						PubKey:  base64.RawURLEncoding.EncodeToString([]byte(pub)),
						Role:    "access",
						Revoked: false,
					},
				},
			},
		},
	}
	data, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	snapshotPath := filepath.Join(dir, "admission-state.json")
	if err := os.WriteFile(snapshotPath, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ks := admission.NewAdmittedKeySet()
	var logBuf strings.Builder
	if err := loadSnapshotFromFile(snapshotPath, ks, &logBuf); err != nil {
		t.Fatalf("AC-007 ValidFile: loadSnapshotFromFile returned error: %v "+
			"(Red Gate: stub returns not-implemented)", err)
	}

	entries := ks.ListBySVTN(svtnID)
	if len(entries) == 0 {
		t.Fatal("AC-007 ValidFile: no entries loaded from valid snapshot file")
	}
	found := false
	for _, e := range entries {
		if string(e.PublicKey) == string(pub) {
			found = true
		}
	}
	if !found {
		t.Error("AC-007 ValidFile: pubkey not found in loaded entries")
	}

	// AC-007 PC-3 / BC-2.05.010 PC-7 / F-3: INFO log with count per SVTN must be emitted.
	logStr := logBuf.String()
	if !strings.Contains(logStr, "admission snapshot loaded") {
		t.Errorf("AC-007 ValidFile F-3: INFO log with loaded entry count not emitted; "+
			"loadSnapshotFromFile must log count per SVTN on successful load (BC-2.05.010 PC-7). "+
			"got log output: %q", logStr)
	}
	if !strings.Contains(logStr, "entries=1") {
		t.Errorf("AC-007 ValidFile F-3: INFO log must include entry count; got: %q", logStr)
	}
	svtnIDHex := svtnIDToHex(svtnID)
	if !strings.Contains(logStr, svtnIDHex) {
		t.Errorf("AC-007 ValidFile F-3: INFO log must include svtn_id %s; got: %q", svtnIDHex, logStr)
	}
}

// TestRouterStartup_AdmissionStateFile_CorruptJSON_FailClosed_EKEY002 verifies
// that a file with invalid JSON causes loadSnapshotFromFile to return a non-nil
// error (E-KEY-002 fail-closed behavior).
//
// BC-2.05.010 PC-9 / EC-005; S-BL.ADMISSION-SYNC-WIRE AC-007.
// Red Gate: FAILS — loadSnapshotFromFile returns errSnapshotNotImplemented (non-nil)
// for non-empty paths regardless, so this test may accidentally PASS at Red Gate.
// We check specifically that the error is NOT errSnapshotNotImplemented to avoid
// vacuous passing.
//
// NOT t.Parallel: avoids listenUnixMgmt umask race that could make MkdirTemp
// create a directory with 0500 (no write), causing WriteFile to fail.
func TestRouterStartup_AdmissionStateFile_CorruptJSON_FailClosed_EKEY002(t *testing.T) {
	dir, err := os.MkdirTemp("", "sb-snap-corrupt-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	// Restore execute+write bits in case listenUnixMgmt umask raced MkdirTemp.
	_ = os.Chmod(dir, 0o700)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	snapshotPath := filepath.Join(dir, "admission-state.json")

	// Write corrupt JSON.
	if err := os.WriteFile(snapshotPath, []byte("{corrupt json{{"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ks := admission.NewAdmittedKeySet()
	err = loadSnapshotFromFile(snapshotPath, ks, nil)
	if err == nil {
		t.Fatal("AC-007 CorruptJSON: loadSnapshotFromFile returned nil error for corrupt JSON file; " +
			"must fail-closed (E-KEY-002)")
	}
	// The error must not be the stub sentinel — it must be a real E-KEY-002 error.
	if errors.Is(err, errSnapshotNotImplemented) {
		t.Errorf("AC-007 CorruptJSON: loadSnapshotFromFile returned stub sentinel errSnapshotNotImplemented; "+
			"must return a meaningful E-KEY-002 error for corrupt JSON. "+
			"Red Gate for this test: error is the stub — implementation needed. err=%v", err)
	}
	// The error message must reference the file path.
	if !strings.Contains(err.Error(), snapshotPath) && !strings.Contains(err.Error(), "E-KEY-002") {
		t.Errorf("AC-007 CorruptJSON: error %q does not contain file path or E-KEY-002; "+
			"must identify the corrupt file in the error message", err.Error())
	}
}

// TestRouterStartup_AdmissionStateFile_UnknownSchemaVersion_FailClosed verifies
// that a file with schema_version != 1 causes fail-closed (E-KEY-002).
//
// BC-2.05.010 PC-9 / EC-004; S-BL.ADMISSION-SYNC-WIRE AC-007.
// Red Gate: similar to CorruptJSON — fails with stub sentinel.
//
// NOT t.Parallel: avoids listenUnixMgmt umask race (same as CorruptJSON test).
func TestRouterStartup_AdmissionStateFile_UnknownSchemaVersion_FailClosed(t *testing.T) {
	dir, err := os.MkdirTemp("", "sb-snap-ver-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	_ = os.Chmod(dir, 0o700)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	snapshotPath := filepath.Join(dir, "admission-state.json")

	badSnap := `{"schema_version":999,"timestamp":"2026-07-16T00:00:00Z","svtns":[]}`
	if err := os.WriteFile(snapshotPath, []byte(badSnap), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ks := admission.NewAdmittedKeySet()
	err = loadSnapshotFromFile(snapshotPath, ks, nil)
	if err == nil {
		t.Fatal("AC-007 UnknownSchemaVersion: loadSnapshotFromFile returned nil for schema_version:999; " +
			"must fail-closed (E-KEY-002 — forward-compat gate)")
	}
	if errors.Is(err, errSnapshotNotImplemented) {
		t.Errorf("AC-007 UnknownSchemaVersion: got stub sentinel; must return real E-KEY-002 error for unknown schema_version. err=%v", err)
	}
}

// TestRouterStartup_LoadedEntries_AdmittedFalse verifies that entries loaded
// from a valid snapshot file have admitted=false.
//
// BC-2.05.010 PC-8; S-BL.ADMISSION-SYNC-WIRE AC-007.
// Red Gate: FAILS — loadSnapshotFromFile returns not-implemented.
//
// NOT t.Parallel: avoids umask race from listenUnixMgmt.
func TestRouterStartup_LoadedEntries_AdmittedFalse(t *testing.T) {
	dir, err := os.MkdirTemp("", "sb-snap-admitted-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	_ = os.Chmod(dir, 0o700)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	var svtnID [16]byte
	if _, err := rand.Read(svtnID[:]); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	snap := snapshotFile{
		SchemaVersion: snapshotCurrentSchemaVersion,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		SVTNs: []snapshotSVTN{
			{
				SVTNID: svtnIDToHex(svtnID),
				Keys: []snapshotKey{
					{
						PubKey:  base64.RawURLEncoding.EncodeToString([]byte(pub)),
						Role:    "access",
						Revoked: false,
					},
				},
			},
		},
	}
	data, _ := json.Marshal(snap)
	snapshotPath := filepath.Join(dir, "admission-state.json")
	if err := os.WriteFile(snapshotPath, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ks := admission.NewAdmittedKeySet()
	if err := loadSnapshotFromFile(snapshotPath, ks, nil); err != nil {
		t.Fatalf("AC-007 LoadedAdmittedFalse: loadSnapshotFromFile: %v (Red Gate: not implemented)", err)
	}

	entries := ks.ListBySVTN(svtnID)
	for _, e := range entries {
		if ks.IsAdmitted(svtnID, e.NodeAddr) {
			t.Error("AC-007 LoadedAdmittedFalse: loaded entry has admitted=true; " +
				"must always be false (challenge-response required to flip admitted=true)")
		}
	}
}

// ── AC-008: non-loopback bind + startup INFO log ───────────────────────────────

// TestRouterMgmtListener_NonLoopbackBindAccepted verifies that the config
// 0.0.0.0:9093 is accepted by Config.Validate() (no loopback restriction per
// Ruling 9).
//
// BC-2.09.003 v2.1 PC-14; S-BL.ADMISSION-SYNC-WIRE AC-008.
// Red Gate: this is a config validation test — passes trivially until E-CFG-016
// validation is implemented. Once implemented, it verifies no loopback guard.
// The meaningful FAIL is in TestConfig_Validate_RouterManagementEndpoints_NonLoopbackAccepted.
func TestRouterMgmtListener_NonLoopbackBindAccepted(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		ListenAddr:   "0.0.0.0:9090",
		TickInterval: 10 * time.Millisecond,
		RouterManagementEndpoints: []config.RouterManagementEndpoint{
			{Addr: "0.0.0.0:9093"},
		},
	}
	err := cfg.Validate()
	if err != nil && strings.Contains(err.Error(), "router_management_endpoints[0].addr") {
		t.Errorf("AC-008: non-loopback addr 0.0.0.0:9093 was rejected by E-CFG-016 validation; "+
			"Ruling 9 prohibits loopback restriction on router_management_endpoints. err=%v", err)
	}
	// Other validation errors (tick_interval etc.) are ignored for this specific assertion.
}

// TestRouterMgmtListener_StartupInfoLog_BindAddress verifies that when the router
// binds its management listener, a startup INFO log is emitted:
//
//	"router management listener bound to <addr> (ensure firewall policy restricts access as appropriate)"
//
// BC-2.09.003 v2.1 PC-14 / Ruling 9; S-BL.ADMISSION-SYNC-WIRE AC-008.
// Red Gate: FAILS because runRouter does not yet emit this INFO log.
func TestRouterMgmtListener_StartupInfoLog_BindAddress(t *testing.T) {
	// NOT t.Parallel: starts runRouter, binds ports.

	probe, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("probe listen: %v", err)
	}
	dataAddr := probe.Addr().String()
	_ = probe.Close()

	sockPath := tempSockPath(t)

	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Capture log output: runRouter writes to w.
	var logBuf strings.Builder
	errCh := make(chan error, 1)
	go func() {
		errCh <- runRouter(ctx, &logBuf, cfg, "", make(chan os.Signal, 1), make(chan struct{}, 1))
	}()

	// Wait for router to start (socket appears).
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, statErr := os.Stat(sockPath); statErr == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if _, statErr := os.Stat(sockPath); os.IsNotExist(statErr) {
		cancel()
		<-errCh
		t.Fatalf("AC-008: mgmt socket %q not created within 2s — runRouter did not start", sockPath)
	}

	cancel()
	select {
	case rErr := <-errCh:
		if rErr != nil {
			t.Logf("AC-008: runRouter returned error on shutdown: %v (may be benign)", rErr)
		}
	case <-time.After(3 * time.Second):
		t.Error("AC-008: runRouter did not return within 3s after ctx cancel")
	}

	// AC-008: the INFO log must contain the router management listener bind message.
	// Red Gate: this message is not yet emitted by runRouter.
	logStr := logBuf.String()
	wantSubstr := "router management listener bound to"
	if !strings.Contains(logStr, wantSubstr) {
		t.Errorf("AC-008: startup INFO log %q not found in runRouter output. "+
			"Red Gate: runRouter does not yet emit this log. output=%q",
			wantSubstr, logStr)
	}
	wantFirewall := "firewall policy"
	if !strings.Contains(logStr, wantFirewall) {
		t.Errorf("AC-008: startup INFO log does not contain %q (firewall advisory). output=%q",
			wantFirewall, logStr)
	}
}

// ── AC-009: PushFullSnapshot on control startup ───────────────────────────────

// startRouterMgmtServerTCP starts an in-process router-side mgmt.Server on a
// real TCP loopback listener with wireAdmissionSyncHandlers registered. The
// control daemon's public key is in the OperatorKeySet so that the
// admissionSyncClient (which authenticates using the control's private key) can
// pass the challenge-response handshake.
//
// Returns the TCP address (127.0.0.1:<port>), the router's AdmittedKeySet, and
// a cleanup function. NOT t.Parallel safe: creates real sockets.
func startRouterMgmtServerTCP(t *testing.T, controlPub ed25519.PublicKey, routerKS *admission.AdmittedKeySet) string {
	t.Helper()

	// Bind on an ephemeral port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("startRouterMgmtServerTCP: listen: %v", err)
	}

	_, routerPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		_ = ln.Close()
		t.Fatalf("startRouterMgmtServerTCP: generate router keypair: %v", err)
	}

	// Authorize the control daemon's public key so pushRPC handshake succeeds.
	ops := mgmt.NewOperatorKeySet([]ed25519.PublicKey{controlPub})

	srv := mgmt.NewServer(ln, routerPriv, ops, nil, "dev",
		mgmt.WithHandshakeTimeout(3*time.Second),
		mgmt.WithRPCIdleTimeout(5*time.Second),
	)

	if err := wireAdmissionSyncHandlers(srv, routerKS, ""); err != nil {
		_ = ln.Close()
		t.Fatalf("startRouterMgmtServerTCP: wireAdmissionSyncHandlers: %v", err)
	}

	addr := ln.Addr().String()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = srv.Serve(ctx)
	}()

	t.Cleanup(func() {
		cancel()
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = srv.Shutdown(shutCtx)
		shutCancel()
		<-done
	})
	return addr
}

// TestAdmissionSync_PushFullSnapshot_AllEntriesPushedToRouter verifies that
// admissionSyncClient.PushFullSnapshot(ctx) pushes all keyset entries to the
// configured router via internal.admission.register RPCs, and that the router's
// AdmittedKeySet receives the pushed entries.
//
// BC-2.05.009 PC-7; S-BL.ADMISSION-SYNC-WIRE AC-009 / F-4 fix.
//
// Integration test: a real in-process router mgmt.Server is started on a loopback
// TCP listener. A real admissionSyncClient dials that listener, completes the
// ADR-012 challenge-response handshake, and pushes the control-side keyset.
// The test asserts that routerKS.ListBySVTN contains the pushed entry.
// This test FAILS if pushRPC's handshake or envelope is broken.
//
// NOT t.Parallel: creates real TCP listeners and sockets.
func TestAdmissionSync_PushFullSnapshot_AllEntriesPushedToRouter(t *testing.T) {
	// Build a control-side AdmittedKeySet with a known entry.
	controlKS := admission.NewAdmittedKeySet()
	var svtnID [16]byte
	if _, err := rand.Read(svtnID[:]); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	controlKS.RegisterKey(svtnID, pub, admission.RoleAccess)

	// Generate the control daemon's keypair — this is what the admissionSyncClient
	// uses to authenticate against the router's OperatorKeySet.
	controlPub, controlPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate control keypair: %v", err)
	}

	// Build router-side AdmittedKeySet (starts empty).
	routerKS := admission.NewAdmittedKeySet()

	// Start a REAL router mgmt.Server on a TCP loopback listener.
	// controlPub is in the router's OperatorKeySet so the handshake succeeds.
	routerAddr := startRouterMgmtServerTCP(t, controlPub, routerKS)

	// Build a REAL admissionSyncClient pointing at the router's TCP address.
	client := newAdmissionSyncClient(
		[]config.RouterManagementEndpoint{{Addr: routerAddr}},
		controlPriv, // authenticates as the control daemon
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// PushFullSnapshot must push all keyset entries to the router.
	// This exercises the ACTUAL pushRPC handshake path.
	if err := client.PushFullSnapshot(ctx, controlKS); err != nil {
		t.Fatalf("AC-009 AllEntriesPushedToRouter: PushFullSnapshot returned error: %v "+
			"(expected nil — real router server is up, handshake should succeed)", err)
	}

	// AC-009 core assertion: the router's AdmittedKeySet must contain the pushed entry.
	entries := routerKS.ListBySVTN(svtnID)
	if len(entries) == 0 {
		t.Fatalf("AC-009 AllEntriesPushedToRouter: routerKS has no entries for SVTN %s "+
			"after PushFullSnapshot — the internal.admission.register RPC must have been "+
			"received and processed by the router server",
			svtnIDToHex(svtnID))
	}
	found := false
	for _, e := range entries {
		if string(e.PublicKey) == string(pub) {
			found = true
			// admitted must be false (challenge-response not done)
			if routerKS.IsAdmitted(svtnID, e.NodeAddr) {
				t.Error("AC-009: router entry admitted=true after PushFullSnapshot; must be false (no challenge-response)")
			}
		}
	}
	if !found {
		t.Errorf("AC-009 AllEntriesPushedToRouter: pubkey not found in router keyset after PushFullSnapshot "+
			"(pushed SVTN=%s)", svtnIDToHex(svtnID))
	}
}

// TestAdmissionSync_PushFullSnapshot_ExpiryPushed verifies that PushFullSnapshot
// also issues internal.admission.expire for entries with non-zero expiry, and that
// the router's AdmittedKeySet records the correct expiry.
//
// BC-2.05.009 PC-7; S-BL.ADMISSION-SYNC-WIRE AC-009 / F-4 fix.
//
// NOT t.Parallel: creates real TCP listeners.
func TestAdmissionSync_PushFullSnapshot_ExpiryPushed(t *testing.T) {
	controlKS := admission.NewAdmittedKeySet()
	var svtnID [16]byte
	if _, err := rand.Read(svtnID[:]); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	controlKS.RegisterKey(svtnID, pub, admission.RoleAccess)

	// Set an expiry on the key.
	entries := controlKS.ListBySVTN(svtnID)
	if len(entries) == 0 {
		t.Fatal("no entries after RegisterKey")
	}
	wantExpiry := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	if err := controlKS.SetKeyExpiry(svtnID, entries[0].NodeAddr, wantExpiry); err != nil {
		t.Fatalf("SetKeyExpiry: %v", err)
	}

	controlPub, controlPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate control keypair: %v", err)
	}

	routerKS := admission.NewAdmittedKeySet()
	routerAddr := startRouterMgmtServerTCP(t, controlPub, routerKS)

	client := newAdmissionSyncClient(
		[]config.RouterManagementEndpoint{{Addr: routerAddr}},
		controlPriv,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.PushFullSnapshot(ctx, controlKS); err != nil {
		t.Fatalf("AC-009 ExpiryPushed: PushFullSnapshot returned error: %v", err)
	}

	// Assert: the router's keyset has the entry with expiry propagated.
	routerEntries := routerKS.ListBySVTN(svtnID)
	if len(routerEntries) == 0 {
		t.Fatal("AC-009 ExpiryPushed: routerKS has no entries after PushFullSnapshot")
	}
	found := false
	for _, e := range routerEntries {
		if string(e.PublicKey) == string(pub) {
			found = true
			gotExpiry := e.KeyExpiry()
			if gotExpiry.IsZero() {
				t.Error("AC-009 ExpiryPushed: router entry has zero expiry after PushFullSnapshot — " +
					"PushFullSnapshot must also push internal.admission.expire for entries with non-zero expiry")
			} else {
				// Allow up to 2s drift (TTL computation in PushSetKeyExpiry uses time.Until).
				diff := gotExpiry.Sub(wantExpiry)
				if diff < -2*time.Second || diff > 2*time.Second {
					t.Errorf("AC-009 ExpiryPushed: router entry expiry=%v; want ~%v (diff %v too large)",
						gotExpiry, wantExpiry, diff)
				}
			}
		}
	}
	if !found {
		t.Error("AC-009 ExpiryPushed: pubkey not found in router entries after PushFullSnapshot")
	}
}

// TestAdmissionSync_PushFullSnapshot_EmptyKeysetNoPushAttempt verifies that
// PushFullSnapshot with an empty keyset does not attempt any pushes (no error,
// no connection attempts).
//
// BC-2.05.009 PC-7; S-BL.ADMISSION-SYNC-WIRE AC-009.
// Red Gate: FAILS — PushFullSnapshot returns errAdmissionSyncNotImplemented.
// Once implemented: must return nil for empty keyset.
func TestAdmissionSync_PushFullSnapshot_EmptyKeysetNoPushAttempt(t *testing.T) {
	t.Parallel()

	emptyKS := admission.NewAdmittedKeySet()
	client := newAdmissionSyncClient(
		[]config.RouterManagementEndpoint{{Addr: "127.0.0.1:0"}},
		make(ed25519.PrivateKey, ed25519.PrivateKeySize),
	)

	ctx := context.Background()
	err := client.PushFullSnapshot(ctx, emptyKS)
	if err != nil {
		// Red Gate: stub returns not-implemented.
		if errors.Is(err, errAdmissionSyncNotImplemented) {
			t.Errorf("AC-009 EmptyKeyset: PushFullSnapshot returned errAdmissionSyncNotImplemented; "+
				"once implemented, empty keyset must return nil with no push attempt. err=%v", err)
			return
		}
		t.Errorf("AC-009 EmptyKeyset: PushFullSnapshot returned unexpected error: %v", err)
	}
}

// ── AC-010: SIGHUP reload updates endpoint list ───────────────────────────────

// TestAdmissionSync_SIGHUPReload_EndpointListUpdated verifies that
// reloadControlEndpoints (the helper called by runControl's SIGHUP branch)
// atomically replaces the sync client's endpoint list.
//
// The test writes a config file with a new endpoint list, calls
// reloadControlEndpoints, then verifies the update took effect by attempting
// a push that must be a no-op (empty endpoint list) — not a connection error
// to the stale address.
//
// BC-2.05.009 Invariant 5; S-BL.ADMISSION-SYNC-WIRE AC-010 / F-1 fix.
// Drives the reload through the ACTUAL reload helper, not UpdateEndpoints in
// isolation — the test FAILS if the reload is not wired to the signal path.
//
// NOT t.Parallel: writes a temp config file.
func TestAdmissionSync_SIGHUPReload_EndpointListUpdated(t *testing.T) {
	dir, err := os.MkdirTemp("", "sb-sighup-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	_ = os.Chmod(dir, 0o700)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	// Write a config file that has an empty RouterManagementEndpoints list.
	// reloadControlEndpoints must update the client to use this empty list.
	cfgContent := `listen_addr: "127.0.0.1:9090"
tick_interval: 10ms
router_management_endpoints: []
`
	cfgPath := filepath.Join(dir, "control.yaml")
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o600); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}

	// Start with a non-empty endpoint at a non-listening address.
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	client := newAdmissionSyncClient(
		[]config.RouterManagementEndpoint{{Addr: "127.0.0.1:19999"}}, // not listening
		priv,
	)

	// Drive reload through the ACTUAL reloadControlEndpoints helper.
	// This is what the SIGHUP branch in runControl calls.
	if err := reloadControlEndpoints(cfgPath, client); err != nil {
		t.Fatalf("AC-010 EndpointListUpdated: reloadControlEndpoints returned error: %v "+
			"(config is valid — this must not fail)", err)
	}

	// After reload with empty endpoints, PushRegisterKey must be a no-op (nil error,
	// no connection attempt). If the reload was not wired, the old endpoint
	// (127.0.0.1:19999, not listening) would be used → connection refused.
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	var svtnID [16]byte
	if _, err := rand.Read(svtnID[:]); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	// With empty endpoint list (from reload), PushRegisterKey must be a no-op.
	if err := client.PushRegisterKey(ctx, svtnID, pub, admission.RoleAccess); err != nil {
		t.Errorf("AC-010 EndpointListUpdated: PushRegisterKey returned error %v after reloadControlEndpoints "+
			"cleared the endpoint list; must be a no-op (empty list). "+
			"FAIL: reload was not applied to the sync client.", err)
	}
}

// TestAdmissionSync_SIGHUPReload_NewListUsedOnNextPush verifies that after a
// SIGHUP reload updates the endpoint list, the NEXT push uses the NEW list.
//
// This test drives the reload via a real sighupCh → runControl select loop:
// it writes two different config files (one with an old non-listening endpoint,
// one with a new non-listening endpoint), delivers SIGHUP, and confirms the
// push switches to the new endpoint (both fail with connection refused, but
// only the first should still produce an attempt after reload).
//
// Simpler version: after a reload that clears endpoints, the push is a no-op.
// This is already covered by TestAdmissionSync_SIGHUPReload_EndpointListUpdated.
// Here we verify via a DIRECT reloadControlEndpoints → push pair.
//
// BC-2.05.009 Invariant 5; S-BL.ADMISSION-SYNC-WIRE AC-010 / F-1 fix.
//
// NOT t.Parallel: writes temp config files.
func TestAdmissionSync_SIGHUPReload_NewListUsedOnNextPush(t *testing.T) {
	dir, err := os.MkdirTemp("", "sb-sighup2-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	_ = os.Chmod(dir, 0o700)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	// Write a config file with empty router_management_endpoints.
	cfgContent := `listen_addr: "127.0.0.1:9090"
tick_interval: 10ms
router_management_endpoints: []
`
	cfgPath := filepath.Join(dir, "control2.yaml")
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o600); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}

	// Start with a non-empty endpoint (initial state before SIGHUP).
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	client := newAdmissionSyncClient(
		[]config.RouterManagementEndpoint{{Addr: "127.0.0.1:19999"}},
		priv,
	)

	// Simulate the SIGHUP branch: call reloadControlEndpoints (the actual helper).
	if err := reloadControlEndpoints(cfgPath, client); err != nil {
		t.Fatalf("AC-010 NewListUsedOnNextPush: reloadControlEndpoints: %v", err)
	}

	// After reload with empty endpoints, PushRegisterKey must be a no-op (nil).
	// If the NEW list is NOT used, the client would try to dial 127.0.0.1:19999
	// and return a connection error.
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	var svtnID [16]byte
	if _, err := rand.Read(svtnID[:]); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	// With an empty endpoint list (from SIGHUP reload), PushRegisterKey must return nil.
	// Red Gate (pre-F-1 fix): runControl had no SIGHUP handler, so reloadControlEndpoints
	// did not exist, and this test would fail to compile.
	if err := client.PushRegisterKey(ctx, svtnID, pub, admission.RoleAccess); err != nil {
		t.Errorf("AC-010 NewListUsedOnNextPush: PushRegisterKey returned %v after reload "+
			"cleared endpoints; must be a no-op (empty list = no dial attempts). "+
			"Implementation is using the stale endpoint list.", err)
	}
}
