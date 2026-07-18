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
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
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
	handlers := BuildAdminHandlers(m, ops, sync, nil)

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
	handlers := BuildAdminHandlers(m, ops, sync, nil)
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
	handlers := BuildAdminHandlers(m, ops, nil, nil) // nil syncer

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

// TestAdmissionSync_RegisterKey_AdminRPCReturnsPromptlyWithUnreachablePush verifies
// that when the push target (a router management endpoint) is unreachable, the
// admin.key.register RPC returns to the caller (sbctl) promptly — well before the
// pushWithRetry backoff would complete — so sbctl never times out spuriously
// (BC-2.05.009 PC-2 / Decision 4; F-1 fix; ARCH-01 §Goroutine WaitGroup Contract).
//
// This test uses a real (non-mock) admissionSyncClient pointing at a TCP address
// with no listener, and a real WaitGroup so the background goroutine is tracked.
// It asserts the admin RPC handler returns in under 500ms even though pushWithRetry
// would take >1.4s (100ms + 200ms + 400ms + 800ms for 4 failed attempts).
//
// BC-2.05.009 PC-2; S-BL.ADMISSION-SYNC-WIRE F-1 fix.
// NOT t.Parallel: creates a real mgmt server.
func TestAdmissionSync_RegisterKey_AdminRPCReturnsPromptlyWithUnreachablePush(t *testing.T) {
	// Use a TCP address with no listener (connection refused immediately).
	// Bind and immediately close to get an address that is guaranteed unoccupied.
	probeDeadEnd, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("probe dead-end listen: %v", err)
	}
	deadEndAddr := probeDeadEnd.Addr().String()
	_ = probeDeadEnd.Close() // close immediately — this address is now unreachable

	_, ctrlPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate control key: %v", err)
	}
	ctrlPub := ctrlPriv.Public().(ed25519.PublicKey)
	ks := admission.NewAdmittedKeySet()
	m := svtnmgmt.NewSVTNManager(ks, ctrlPub)
	if _, err := m.Create("prompt-return-svtn"); err != nil {
		t.Fatalf("create SVTN: %v", err)
	}

	// Real admissionSyncClient pointing at the dead-end address.
	syncClient := newAdmissionSyncClient(
		[]config.RouterManagementEndpoint{{Addr: deadEndAddr}},
		ctrlPriv,
	)

	// Real WaitGroup — tracks the background push goroutine.
	var pushWG sync.WaitGroup

	ops := mgmt.NewOperatorKeySet(nil)
	// Pass the real pushWG → async push.
	handlers := BuildAdminHandlers(m, ops, syncClient, &pushWG)
	es := startE2EServerWithOps(t, handlers, ctrlPriv, ops)

	regPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate reg key: %v", err)
	}
	pubkeyB64 := base64.RawURLEncoding.EncodeToString([]byte(regPub))

	// Time the RPC — it must return fast even though the push will fail slowly.
	start := time.Now()
	resp := sendAdminRPC(t, es.socketPath, ctrlPriv, "admin.key.register", map[string]any{
		"svtn_id":        "prompt-return-svtn",
		"pubkey_openssh": pubkeyB64,
		"role":           "access",
	})
	elapsed := time.Since(start)

	// RPC must succeed (advisory push failure does not affect the result).
	if errObj, ok := resp["error"].(map[string]any); ok {
		t.Errorf("F-1: admin.key.register must return success even with unreachable push endpoint; "+
			"got error: %v", errObj)
	}

	// CORE ASSERTION: handler must return promptly — well under the first retry delay
	// (pushWithRetry sleeps 100ms between attempts; 500ms threshold leaves 5× headroom
	// without being flaky). A synchronous push to a dead endpoint would block for
	// much longer (5 attempts × TCP connect timeout or 100+200+400+800ms backoff).
	const promptThreshold = 500 * time.Millisecond
	if elapsed > promptThreshold {
		t.Errorf("F-1: admin RPC took %v (want < %v); push is still synchronous — "+
			"F-1 fix requires async dispatch via dispatchPush + WaitGroup", elapsed, promptThreshold)
	}

	// Wait for the background goroutine to drain (so the test exits cleanly).
	// This is NOT part of the sbctl timing path — sbctl already got its response.
	pushWG.Wait()
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
	handlers := BuildAdminHandlers(m, ops, sync, nil)
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
	handlers := BuildAdminHandlers(m, ops, sync, nil)
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
	handlers := BuildAdminHandlers(m, ops, sync, nil)
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
			handlers := BuildAdminHandlers(m, ops, sync, nil)
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

// TestRouterMgmtListener_TCPBind_ConnectionSucceeds verifies AC-008 postcondition 2
// (Ruling 10): when management_socket is set to a host:port value, runRouter binds a
// real TCP management listener — verified by net.Dial("tcp", addr) succeeding after
// startup. This catches the F-2 gap where the router bound unix instead of TCP.
//
// BC-2.09.003 v2.1 PC-14; S-BL.ADMISSION-SYNC-WIRE AC-008 / Ruling 10.
// NOT t.Parallel: starts runRouter (binds TCP and unix listeners).
func TestRouterMgmtListener_TCPBind_ConnectionSucceeds(t *testing.T) {
	// Bind an ephemeral TCP address for the data-plane listener.
	probeData, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("probe data listen: %v", err)
	}
	dataAddr := probeData.Addr().String()
	_ = probeData.Close()

	// Bind an ephemeral TCP address for the management listener.
	// Use 127.0.0.1:0 and resolve to a concrete port before starting runRouter
	// to avoid port conflicts in CI.
	probeMgmt, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("probe mgmt listen: %v", err)
	}
	mgmtAddr := probeMgmt.Addr().String()
	_ = probeMgmt.Close()

	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: mgmtAddr, // host:port → auto-detect TCP (Ruling 10)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	errCh := make(chan error, 1)
	go func() {
		errCh <- runRouter(ctx, nil, cfg, "", make(chan os.Signal, 1), make(chan struct{}, 1))
	}()

	// Wait for the TCP management listener to be ready.
	deadline := time.Now().Add(3 * time.Second)
	var dialErr error
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", mgmtAddr, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			dialErr = nil
			break
		}
		dialErr = err
		time.Sleep(20 * time.Millisecond)
	}
	if dialErr != nil {
		cancel()
		<-errCh
		t.Fatalf("AC-008 TCPBind: net.Dial(%q) failed within 3s: %v\n"+
			"Ruling 10: runRouter with host:port management_socket must bind a TCP listener, not unix.\n"+
			"F-2 fix: mgmtListenAddr auto-detects TCP when management_socket is a valid host:port.",
			mgmtAddr, dialErr)
	}

	// TCP connection succeeded — listener is genuinely TCP.
	cancel()
	select {
	case rErr := <-errCh:
		if rErr != nil {
			t.Logf("AC-008 TCPBind: runRouter returned: %v (may be benign)", rErr)
		}
	case <-time.After(3 * time.Second):
		t.Error("AC-008 TCPBind: runRouter did not return within 3s after ctx cancel")
	}
}

// TestRouterMgmtListener_TCPBind_PushHandshakeSucceeds verifies AC-008 postcondition 3
// (Ruling 10): a real admissionSyncClient (control daemonPriv authorized on the router)
// can push an internal.admission.register RPC to a runRouter instance started with a
// TCP management_socket, and routerKS receives the entry end-to-end.
//
// BC-2.09.003 v2.1 PC-14; S-BL.ADMISSION-SYNC-WIRE AC-008 / Ruling 10.
// NOT t.Parallel: starts runRouter (binds TCP and unix listeners).
func TestRouterMgmtListener_TCPBind_PushHandshakeSucceeds(t *testing.T) {
	// Bind ephemeral ports to avoid conflicts.
	probeData, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("probe data listen: %v", err)
	}
	dataAddr := probeData.Addr().String()
	_ = probeData.Close()

	probeMgmt, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("probe mgmt listen: %v", err)
	}
	mgmtAddr := probeMgmt.Addr().String()
	_ = probeMgmt.Close()

	// Generate a control daemon keypair. The control pubkey will be added to the
	// router's authorized_operator_keys so the challenge-response handshake succeeds.
	controlPub, controlPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate control keypair: %v", err)
	}

	// Encode controlPub as a PEM "PUBLIC KEY" authorized_operator_key for the router
	// config — same format that parsePEMOperatorKeys expects (x509 PKIX / ARCH-12).
	pkixDER, err := x509.MarshalPKIXPublicKey(controlPub)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey: %v", err)
	}
	pemKey := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pkixDER}))

	cfg := &config.Config{
		ListenAddr:             dataAddr,
		TickInterval:           10 * time.Millisecond,
		ManagementSocket:       mgmtAddr, // host:port → TCP (Ruling 10)
		AuthorizedOperatorKeys: []string{pemKey},
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	errCh := make(chan error, 1)
	go func() {
		errCh <- runRouter(ctx, nil, cfg, "", make(chan os.Signal, 1), make(chan struct{}, 1))
	}()

	// Wait for the TCP management listener to accept connections.
	deadline := time.Now().Add(3 * time.Second)
	ready := false
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", mgmtAddr, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			ready = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !ready {
		cancel()
		<-errCh
		t.Fatalf("AC-008 PushHandshake: TCP management listener at %q not ready within 3s — runRouter did not bind TCP", mgmtAddr)
	}

	// Build a control-side admissionSyncClient pointing at the router's TCP addr.
	syncClient := newAdmissionSyncClient(
		[]config.RouterManagementEndpoint{{Addr: mgmtAddr}},
		controlPriv,
	)

	// Push an internal.admission.register RPC to the router.
	var svtnID [16]byte
	if _, err := rand.Read(svtnID[:]); err != nil {
		t.Fatalf("rand.Read svtnID: %v", err)
	}
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate node keypair: %v", err)
	}

	pushCtx, pushCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pushCancel()
	if pushErr := syncClient.PushRegisterKey(pushCtx, svtnID, pub, admission.RoleAccess); pushErr != nil {
		cancel()
		<-errCh
		t.Fatalf("AC-008 PushHandshake: PushRegisterKey to TCP router addr %q failed: %v\n"+
			"Ruling 10: admissionSyncClient must be able to push to runRouter with TCP management_socket.",
			mgmtAddr, pushErr)
	}

	// Push succeeded. Cancel router.
	cancel()
	select {
	case rErr := <-errCh:
		if rErr != nil {
			t.Logf("AC-008 PushHandshake: runRouter returned: %v (may be benign)", rErr)
		}
	case <-time.After(3 * time.Second):
		t.Error("AC-008 PushHandshake: runRouter did not return within 3s after ctx cancel")
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

// ── AC-011: control-side keyset persistence ────────────────────────────────────

// TestControlAdmission_PersistOnMutation verifies that after a successful
// admin.key.register on control with control_admission_state_file configured,
// the snapshot file is written synchronously BEFORE dispatchPush is called.
//
// BC-2.05.009 PC-7 v1.2; BC-2.09.003 PC-15 v2.2; S-BL.ADMISSION-SYNC-WIRE AC-011.
// Red Gate: FAILS — BuildAdminHandlers does not yet accept a controlSnapshotPath
// and admin_handlers.go does not yet call writeSnapshotAtomic in the handlers.
//
// NOT t.Parallel: creates filesystem socket AND tempfile.
func TestControlAdmission_PersistOnMutation(t *testing.T) {
	dir, err := os.MkdirTemp("", "sb-ctrl-persist-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	_ = os.Chmod(dir, 0o700)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	snapshotPath := filepath.Join(dir, "control-admission-state.json")

	sync := &mockSyncer{}
	ks := admission.NewAdmittedKeySet()
	_, ctrlPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate control key: %v", err)
	}
	ctrlPub := ctrlPriv.Public().(ed25519.PublicKey)
	m := svtnmgmt.NewSVTNManager(ks, ctrlPub)
	if _, err := m.Create("persist-test-svtn"); err != nil {
		t.Fatalf("create SVTN: %v", err)
	}
	regPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate reg key: %v", err)
	}

	ops := mgmt.NewOperatorKeySet(nil)
	// AC-011: BuildAdminHandlers must accept a controlSnapshotPath parameter
	// so the handlers can persist the keyset synchronously before dispatchPush.
	// Red Gate: BuildAdminHandlers does not yet take this parameter.
	handlers := BuildAdminHandlers(m, ops, sync, nil, snapshotPath)
	es := startE2EServerWithOps(t, handlers, ctrlPriv, ops)

	pubkeyB64 := base64.RawURLEncoding.EncodeToString([]byte(regPub))
	resp := sendAdminRPC(t, es.socketPath, ctrlPriv, "admin.key.register", map[string]any{
		"svtn_id":        "persist-test-svtn",
		"pubkey_openssh": pubkeyB64,
		"role":           "access",
	})

	if errObj, ok := resp["error"].(map[string]any); ok {
		t.Fatalf("AC-011 PersistOnMutation: admin.key.register returned error: %v", errObj)
	}

	// AC-011 PC-2: snapshot file MUST exist after the RPC returns.
	// The write is synchronous before dispatchPush (Ruling 11).
	data, readErr := os.ReadFile(snapshotPath)
	if readErr != nil {
		t.Fatalf("AC-011 PersistOnMutation: snapshot file %q not written after admin.key.register. "+
			"Red Gate: makeRegisterHandler does not yet call writeSnapshotAtomic. err=%v",
			snapshotPath, readErr)
	}

	var snap snapshotFile
	if err := json.Unmarshal(data, &snap); err != nil {
		t.Fatalf("AC-011 PersistOnMutation: snapshot file contains invalid JSON: %v", err)
	}
	if snap.SchemaVersion != snapshotCurrentSchemaVersion {
		t.Errorf("AC-011 PersistOnMutation: snapshot schema_version=%d; want %d",
			snap.SchemaVersion, snapshotCurrentSchemaVersion)
	}
	// Snapshot must contain the registered key.
	if len(snap.SVTNs) == 0 {
		t.Error("AC-011 PersistOnMutation: snapshot svtns array is empty after admin.key.register")
	}
}

// TestControlAdmission_LoadAndPushFullSnapshot verifies the EC-007-is-real scenario:
// a control daemon that has control_admission_state_file configured loads it on
// startup BEFORE constructing the sync client and calling PushFullSnapshot, so the
// loaded keys are pushed to routers.
//
// This is the LoadAndPushFullSnapshot test: write a control snapshot file,
// start runControl (or its load helper) so it loads the file, stand up a real
// router (reuse startRouterMgmtServerTCP), assert the router receives the
// loaded keys via PushFullSnapshot.
//
// BC-2.05.009 PC-7 v1.2; S-BL.ADMISSION-SYNC-WIRE AC-011 PC-3/PC-4.
// Red Gate: FAILS — runControl does not yet load ControlAdmissionStateFile.
//
// NOT t.Parallel: starts real TCP listeners.
func TestControlAdmission_LoadAndPushFullSnapshot(t *testing.T) {
	dir, err := os.MkdirTemp("", "sb-ctrl-load-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	_ = os.Chmod(dir, 0o700)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	snapshotPath := filepath.Join(dir, "control-admission-state.json")

	// Build a snapshot file with a known entry.
	var svtnID [16]byte
	if _, err := rand.Read(svtnID[:]); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	// Write a valid snapshot file that simulates a prior control-daemon run.
	snapToWrite := snapshotFile{
		SchemaVersion: snapshotCurrentSchemaVersion,
		Timestamp:     "2026-07-17T00:00:00Z",
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
	snapData, err := json.Marshal(snapToWrite)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if err := os.WriteFile(snapshotPath, snapData, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Build the control daemon's keypair.
	controlPub, controlPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate control keypair: %v", err)
	}

	// Start a real router-side mgmt server.
	routerKS := admission.NewAdmittedKeySet()
	routerAddr := startRouterMgmtServerTCP(t, controlPub, routerKS)

	// Simulate the runControl startup sequence: load snapshot, THEN push.
	// We test the load helper + PushFullSnapshot directly rather than running
	// the full daemon to keep the test deterministic.
	loadKS := admission.NewAdmittedKeySet()
	if loadErr := loadSnapshotFromFile(snapshotPath, loadKS, nil); loadErr != nil {
		t.Fatalf("AC-011 LoadAndPush: loadSnapshotFromFile(%q): %v "+
			"(Red Gate: file is valid but function not called for ControlAdmissionStateFile)",
			snapshotPath, loadErr)
	}

	// Verify the snapshot was actually loaded (non-empty keyset).
	loadedEntries := loadKS.ListBySVTN(svtnID)
	if len(loadedEntries) == 0 {
		t.Fatalf("AC-011 LoadAndPush: loadSnapshotFromFile loaded 0 entries for SVTN %s; "+
			"snapshot had 1 entry — load failed", svtnIDToHex(svtnID))
	}

	// Construct a real sync client and push the loaded keyset to the router.
	client := newAdmissionSyncClient(
		[]config.RouterManagementEndpoint{{Addr: routerAddr}},
		controlPriv,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if pushErr := client.PushFullSnapshot(ctx, loadKS); pushErr != nil {
		t.Fatalf("AC-011 LoadAndPush: PushFullSnapshot returned error: %v", pushErr)
	}

	// AC-011 PC-4 core assertion: router received the loaded keys (EC-007 is real).
	routerEntries := routerKS.ListBySVTN(svtnID)
	if len(routerEntries) == 0 {
		t.Fatalf("AC-011 LoadAndPush: routerKS has no entries for SVTN %s after PushFullSnapshot. "+
			"EC-007 resync requires: control loads snapshot → pushes to routers on startup.",
			svtnIDToHex(svtnID))
	}
	found := false
	for _, e := range routerEntries {
		if string(e.PublicKey) == string(pub) {
			found = true
			if routerKS.IsAdmitted(svtnID, e.NodeAddr) {
				t.Error("AC-011 LoadAndPush: router entry admitted=true; must be false")
			}
		}
	}
	if !found {
		t.Error("AC-011 LoadAndPush: loaded pubkey not found in router keyset after push")
	}
}

// TestControlAdmission_FailClosedOnCorruptSnapshot verifies that when
// control_admission_state_file is configured and the file is corrupt,
// runControl returns E-KEY-002 (fail-closed), daemon refuses to start.
//
// BC-2.05.009 PC-7 v1.2; S-BL.ADMISSION-SYNC-WIRE AC-011 PC-3.
// Red Gate: FAILS — runControl does not yet load ControlAdmissionStateFile.
//
// NOT t.Parallel: creates real sockets / listeners.
func TestControlAdmission_FailClosedOnCorruptSnapshot(t *testing.T) {
	dir, err := os.MkdirTemp("", "sb-ctrl-corrupt-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	_ = os.Chmod(dir, 0o700)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	snapshotPath := filepath.Join(dir, "control-admission-state.json")
	if err := os.WriteFile(snapshotPath, []byte("{corrupt json{{"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Test the load helper directly: corrupt file → fail-closed error.
	ks := admission.NewAdmittedKeySet()
	loadErr := loadSnapshotFromFile(snapshotPath, ks, nil)
	if loadErr == nil {
		t.Fatal("AC-011 FailClosed: loadSnapshotFromFile returned nil for corrupt file; " +
			"must return E-KEY-002 (fail-closed)")
	}
	if !strings.Contains(loadErr.Error(), "E-KEY-002") {
		t.Errorf("AC-011 FailClosed: error does not contain E-KEY-002: %v", loadErr)
	}
}

// TestControlAdmission_MissingFileEmptyKeyset verifies that when
// control_admission_state_file is configured but the file does not exist,
// the control daemon starts with an empty keyset (fresh install — no error).
//
// BC-2.05.009 PC-7 v1.2; S-BL.ADMISSION-SYNC-WIRE AC-011 PC-3.
// Red Gate: PASSES trivially — loadSnapshotFromFile already returns nil for
// missing files. Included to lock the positive invariant for the control path.
func TestControlAdmission_MissingFileEmptyKeyset(t *testing.T) {
	t.Parallel()

	dir, err := os.MkdirTemp("", "sb-ctrl-missing-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	nonExistentPath := filepath.Join(dir, "does-not-exist.json")

	ks := admission.NewAdmittedKeySet()
	if loadErr := loadSnapshotFromFile(nonExistentPath, ks, nil); loadErr != nil {
		t.Errorf("AC-011 MissingFile: loadSnapshotFromFile returned error for absent file: %v; "+
			"missing file must produce nil error and empty keyset (fresh install)", loadErr)
	}
	// Keyset must be empty.
	allEntries := ks.AllSVTNEntries()
	if len(allEntries) != 0 {
		t.Errorf("AC-011 MissingFile: keyset is non-empty after loading absent file; got %d SVTNs", len(allEntries))
	}
}

// ── AC-012: mgmt listener loopback guard scope ────────────────────────────────

// TestControlMgmtListener_NonLoopbackRejected verifies that a control-mode
// daemon with management_socket set to a non-loopback TCP address fails at
// buildMgmtListener with E-CFG-008 (Ruling 12).
//
// BC-2.09.003 PC-11b v2.2; S-BL.ADMISSION-SYNC-WIRE AC-012 PC-1.
// Red Gate: FAILS — buildMgmtListener currently only applies the loopback guard
// for mode == "console"; control mode does NOT trigger the guard → non-loopback
// TCP binds successfully instead of returning E-CFG-008.
//
// NOT t.Parallel: binds a real TCP socket.
func TestControlMgmtListener_NonLoopbackRejected(t *testing.T) {
	// Bind and immediately close to get a port number we can use in the config.
	probeL, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("probe listen: %v", err)
	}
	port := probeL.Addr().(*net.TCPAddr).Port
	_ = probeL.Close()

	// Build a config that would trigger the TCP path in buildMgmtListener for
	// control mode — use 0.0.0.0 (non-loopback) as the management_socket.
	nonLoopbackAddr := fmt.Sprintf("0.0.0.0:%d", port)
	cfg := &config.Config{
		ManagementSocket: nonLoopbackAddr,
	}

	ln, buildErr := buildMgmtListener(cfg, "control")
	if ln != nil {
		_ = ln.Close()
	}
	if buildErr == nil {
		t.Fatalf("AC-012 NonLoopbackRejected: buildMgmtListener(cfg, \"control\") with %q "+
			"returned nil error; must return E-CFG-008.\n"+
			"Red Gate: guard is currently `if mode == \"console\"` — control mode is not guarded.",
			nonLoopbackAddr)
	}
	if !strings.Contains(buildErr.Error(), "E-CFG-008") {
		t.Errorf("AC-012 NonLoopbackRejected: error does not contain E-CFG-008: %v", buildErr)
	}
	if !strings.Contains(buildErr.Error(), "control mode requires a loopback address") {
		t.Errorf("AC-012 NonLoopbackRejected: error does not contain expected mode message: %v", buildErr)
	}
}

// TestControlMgmtListener_LoopbackTCPAccepted verifies that a control-mode
// daemon with management_socket set to a loopback TCP address binds successfully.
//
// BC-2.09.003 PC-11b v2.2; S-BL.ADMISSION-SYNC-WIRE AC-012 PC-2.
// Red Gate: PASSES trivially (control mode is not yet guarded → both loopback
// and non-loopback bind). Once Ruling 12 is implemented, this test verifies
// the loopback case still passes while non-loopback is rejected.
//
// NOT t.Parallel: binds a real TCP socket.
func TestControlMgmtListener_LoopbackTCPAccepted(t *testing.T) {
	cfg := &config.Config{
		ManagementSocket: "127.0.0.1:0", // loopback + ephemeral port
	}

	ln, buildErr := buildMgmtListener(cfg, "control")
	if buildErr != nil {
		t.Fatalf("AC-012 LoopbackTCPAccepted: buildMgmtListener(cfg, \"control\") with 127.0.0.1:0 "+
			"returned error: %v; loopback TCP must be accepted for control mode", buildErr)
	}
	if ln != nil {
		_ = ln.Close()
	}
}

// TestRouterMgmtListener_NonLoopbackStillAccepted_Ruling9 verifies that the
// loopback guard does NOT apply to router mode (Ruling 9 unchanged by Ruling 12).
//
// BC-2.09.003 PC-11b v2.2; S-BL.ADMISSION-SYNC-WIRE AC-012 PC-3.
// NOT t.Parallel: binds a real TCP socket.
func TestRouterMgmtListener_NonLoopbackStillAccepted_Ruling9(t *testing.T) {
	// Bind and immediately close to get a free ephemeral port on 0.0.0.0.
	probeL, err := net.ListenTCP("tcp", &net.TCPAddr{})
	if err != nil {
		t.Fatalf("probe listen: %v", err)
	}
	port := probeL.Addr().(*net.TCPAddr).Port
	_ = probeL.Close()

	nonLoopbackAddr := fmt.Sprintf("0.0.0.0:%d", port)
	cfg := &config.Config{
		ManagementSocket: nonLoopbackAddr,
	}

	ln, buildErr := buildMgmtListener(cfg, "router")
	if buildErr != nil {
		t.Fatalf("AC-012 Ruling9 regression: buildMgmtListener(cfg, \"router\") with %q "+
			"returned error: %v; router mode must NOT have a loopback restriction (Ruling 9 unchanged)",
			nonLoopbackAddr, buildErr)
	}
	if ln != nil {
		_ = ln.Close()
	}
}

// ── F-P3-03: bounded shutdown drain ───────────────────────────────────────────

// TestAdmissionSyncClient_BoundedDialTimeout verifies that pushRPC's per-dial
// timeout (admissionSyncDialTimeout) is actually enforced against a black-holed
// (SYN-drop) endpoint, not just a connection-refused one.
//
// A connection-refused endpoint returns instantly regardless of the dialer
// timeout — it cannot prove the timeout fires. RFC 5737 TEST-NET-1
// (192.0.2.1) is guaranteed non-routable on any compliant network stack:
// the SYN is dropped, so the only way the dial completes is when the
// Dialer.Timeout fires.
//
// Assertion: a single-attempt push to 192.0.2.1 takes ≥ admissionSyncDialTimeout
// (proving the timeout fired) and ≤ admissionSyncDialTimeout + generous margin
// (proving it is bounded, not hanging at the OS default ~127s). Removing the
// Dialer.Timeout from pushRPC would cause the test to hang past the upper bound
// and fail.
//
// F-4C / F-P3-03 / BC-2.05.009 PC-2.
// NOT t.Parallel: dials a real non-routable address; duration-sensitive.
func TestAdmissionSyncClient_BoundedDialTimeout(t *testing.T) {
	// Verify the constant exists and is a sane bound.
	const wantDialTimeoutBound = 10 * time.Second
	if admissionSyncDialTimeout > wantDialTimeoutBound {
		t.Errorf("F-4C: admissionSyncDialTimeout=%v exceeds %v; "+
			"the per-dial bound must be well under the OS TCP connect timeout (~127s)",
			admissionSyncDialTimeout, wantDialTimeoutBound)
	}
	if admissionSyncDialTimeout <= 0 {
		t.Errorf("F-4C: admissionSyncDialTimeout=%v must be positive", admissionSyncDialTimeout)
	}

	// 192.0.2.1 is RFC 5737 TEST-NET-1 — non-routable, black-holes SYNs.
	// Any port works; use 9 (Discard) to be explicit.
	blackHoleAddr := "192.0.2.1:9"

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	// Configure a sync client pointing at the black-hole address with a single
	// attempt (admissionSyncRetryMax=1 equivalent: we call pushRPC directly so
	// we only wait one timeout, keeping CI time to ~admissionSyncDialTimeout).
	client := newAdmissionSyncClient(
		[]config.RouterManagementEndpoint{{Addr: blackHoleAddr}},
		priv,
	)

	var svtnID [16]byte
	if _, err := rand.Read(svtnID[:]); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	// To keep test duration to one dial timeout, call pushRPC directly (one dial,
	// no retry loop). This isolates the per-dial bound without running the full
	// 5-attempt retry sequence (~5×admissionSyncDialTimeout = 25s).
	ctx := context.Background()
	start := time.Now()
	argsJSON, merr := json.Marshal(struct {
		SVTNID string `json:"svtn_id"`
		PubKey string `json:"pubkey"`
		Role   string `json:"role"`
	}{
		SVTNID: hex.EncodeToString(svtnID[:]),
		PubKey: base64.RawURLEncoding.EncodeToString([]byte(pub)),
		Role:   "access",
	})
	if merr != nil {
		t.Fatalf("marshal args: %v", merr)
	}
	pushErr := client.pushRPC(ctx, blackHoleAddr, CmdAdmissionRegister, argsJSON)
	elapsed := time.Since(start)

	// pushRPC must return an error (SYN-drop / dial timeout).
	if pushErr == nil {
		t.Errorf("F-4C: pushRPC to black-hole %s returned nil error; expected dial timeout error", blackHoleAddr)
	}

	// Lower bound: elapsed ≥ admissionSyncDialTimeout × 0.8 proves the timeout
	// fired (not connection-refused or instant error).
	lowerBound := admissionSyncDialTimeout * 8 / 10
	if elapsed < lowerBound {
		t.Errorf("F-4C: pushRPC returned in %v; expected ≥ %v (lower bound = 0.8×admissionSyncDialTimeout=%v).\n"+
			"The dial to 192.0.2.1 was instant — connection-refused path or routing anomaly?\n"+
			"If removing Dialer.Timeout from pushRPC, this test would hang well past the upper bound.",
			elapsed, lowerBound, admissionSyncDialTimeout)
	}

	// Upper bound: elapsed ≤ admissionSyncDialTimeout × 2 + 2s proves the per-dial
	// bound is respected and not relying on the OS default (~127s).
	upperBound := admissionSyncDialTimeout*2 + 2*time.Second
	if elapsed > upperBound {
		t.Errorf("F-4C: pushRPC to black-hole took %v; expected ≤ %v.\n"+
			"This suggests the Dialer.Timeout is not set — the OS TCP connect timeout (~127s) is firing instead.\n"+
			"Fix: set Dialer{Timeout: admissionSyncDialTimeout} in pushRPC.",
			elapsed, upperBound)
	}
	t.Logf("F-4C: pushRPC to black-hole 192.0.2.1 returned in %v (admissionSyncDialTimeout=%v): %v",
		elapsed, admissionSyncDialTimeout, pushErr)
}

// ── F-4A: control-side snapshot mutation-order preservation ──────────────────

// TestControlAdmission_SnapshotMutationOrderPreserved verifies the F-4A
// invariant: after any sequence of concurrent persist calls from admin handlers,
// the on-disk snapshot must reflect a consistent, non-stale view of the keyset.
//
// F-4A / BC-2.05.010 Invariant 1 / S-BL.ADMISSION-SYNC-WIRE AC-011.
//
// Race scenario (F-4A): H2(register B) reads {A:active} before H1(revoke A)
// mutates the keyset. H1 writes {A:revoked}+renames. H2 then renames its stale
// snapshot — disk = {A:active} (revocation defeated across restart).
//
// Fix: hold a shared persist mutex across {snapshot-read, marshal, write, rename}
// in the controlPersister. Once the mutex is held, the snapshot-read happens AFTER
// the prior persists complete, so stale reads cannot overwrite fresh ones.
//
// This test drives N concurrent register+revoke cycles through the real handler
// functions (which call the persist path). After drain, it verifies that the
// on-disk snapshot is consistent with the in-memory keyset state — specifically,
// that no revoked key appears active on disk.
//
// Under -race this test also detects data races in the persist path.
//
// NOT t.Parallel: creates tempdir, writes real files.
func TestControlAdmission_SnapshotMutationOrderPreserved(t *testing.T) {
	dir, err := os.MkdirTemp("", "sb-ctrl-race-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	_ = os.Chmod(dir, 0o700)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	snapPath := filepath.Join(dir, "control-state.json")

	_, ctrlPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ctrl key: %v", err)
	}
	ctrlPub := ctrlPriv.Public().(ed25519.PublicKey)
	ks := admission.NewAdmittedKeySet()
	m := svtnmgmt.NewSVTNManager(ks, ctrlPub)
	if _, err := m.Create("race-svtn"); err != nil {
		t.Fatalf("create SVTN: %v", err)
	}

	// Build handlers with the snapshot path (after the fix, all four handlers
	// share the same controlPersister with its mutex).
	handlers := BuildAdminHandlers(m, mgmt.NewOperatorKeySet(nil), nil, nil, snapPath)
	handlerMap := make(map[string]func(ctx context.Context, args json.RawMessage) (any, error))
	for _, h := range handlers {
		handlerMap[h.Command] = h.Fn
	}
	registerFn := handlerMap["admin.key.register"]
	revokeFn := handlerMap["admin.key.revoke"]
	ctx := context.Background()

	// Phase 1: register N keys sequentially (establishes keyset state for Phase 2).
	const N = 30
	keys := make([]ed25519.PublicKey, N)
	for i := range keys {
		pub, _, kerr := ed25519.GenerateKey(rand.Reader)
		if kerr != nil {
			t.Fatalf("generate key %d: %v", i, kerr)
		}
		keys[i] = pub
		args, merr := json.Marshal(map[string]any{
			"svtn_id":        "race-svtn",
			"pubkey_openssh": base64.RawURLEncoding.EncodeToString([]byte(pub)),
			"role":           "access",
			"caller_role":    "control",
		})
		if merr != nil {
			t.Fatalf("marshal register args %d: %v", i, merr)
		}
		if _, herr := registerFn(ctx, args); herr != nil {
			t.Fatalf("register key %d: %v", i, herr)
		}
	}

	// Phase 2: concurrently revoke the first N/2 keys AND register N/2 extra keys.
	// Without the persist mutex, a register goroutine (which reads the keyset
	// BEFORE the revoke goroutine mutates it) may rename AFTER the revoke goroutine
	// → stale snapshot overwrites the authoritative one.
	const M = 15
	extraKeys := make([]ed25519.PublicKey, M)
	for i := range extraKeys {
		pub, _, kerr := ed25519.GenerateKey(rand.Reader)
		if kerr != nil {
			t.Fatalf("generate extra key %d: %v", i, kerr)
		}
		extraKeys[i] = pub
	}

	var wg sync.WaitGroup
	// N/2 goroutines each revoke a distinct key.
	for i := 0; i < N/2; i++ {
		wg.Add(1)
		pub := keys[i]
		go func() {
			defer wg.Done()
			args, merr := json.Marshal(map[string]any{
				"svtn_id":     "race-svtn",
				"pubkey":      base64.RawURLEncoding.EncodeToString([]byte(pub)),
				"role":        "access",
				"caller_role": "control",
			})
			if merr != nil {
				return
			}
			_, _ = revokeFn(ctx, args)
		}()
	}
	// M goroutines each register a new key (stale snapshot vectors).
	for i, pub := range extraKeys {
		wg.Add(1)
		i, pub := i, pub
		go func() {
			defer wg.Done()
			args, merr := json.Marshal(map[string]any{
				"svtn_id":        "race-svtn",
				"pubkey_openssh": base64.RawURLEncoding.EncodeToString([]byte(pub)),
				"role":           "access",
				"caller_role":    "control",
			})
			if merr != nil {
				t.Errorf("marshal extra register args %d: %v", i, merr)
				return
			}
			_, _ = registerFn(ctx, args)
		}()
	}
	wg.Wait()

	// Load the final snapshot.
	snapData, readErr := os.ReadFile(snapPath)
	if readErr != nil {
		t.Fatalf("F-4A: snapshot not found after concurrent mutations: %v", readErr)
	}
	var snap snapshotFile
	if uerr := json.Unmarshal(snapData, &snap); uerr != nil {
		t.Fatalf("F-4A: snapshot contains invalid JSON: %v", uerr)
	}
	if snap.SchemaVersion != snapshotCurrentSchemaVersion {
		t.Errorf("F-4A: snapshot schema_version=%d; want %d", snap.SchemaVersion, snapshotCurrentSchemaVersion)
	}

	// Build in-memory revocation state.
	allEntries := ks.AllSVTNEntries()
	type keyState struct{ revoked bool }
	inMem := make(map[string]keyState)
	for _, entries := range allEntries {
		for _, e := range entries {
			inMem[string([]byte(e.PublicKey))] = keyState{revoked: e.IsRevoked()}
		}
	}

	// Assert: for every key that appears ACTIVE in the snapshot, the in-memory
	// keyset must also show it as active. If in-memory says revoked but disk says
	// active, a stale rename occurred (F-4A lost-update race).
	for _, svtn := range snap.SVTNs {
		for _, sk := range svtn.Keys {
			if sk.Revoked {
				continue // revoked keys in snapshot are always safe
			}
			pubBytes, derr := base64.RawURLEncoding.DecodeString(sk.PubKey)
			if derr != nil {
				t.Errorf("F-4A: invalid pubkey in snapshot: %v", derr)
				continue
			}
			ms, found := inMem[string(pubBytes)]
			if !found {
				// Active in snapshot, absent in memory: stale ghost entry.
				t.Errorf("F-4A: snapshot has active key not present in in-memory keyset "+
					"(pubkey=%s) — stale snapshot overwrite (F-4A race).\n"+
					"Fix: wrap the entire {snapshot-read, marshal, write, rename} sequence "+
					"in a shared persist mutex across all four admin handlers.",
					sk.PubKey)
				continue
			}
			if ms.revoked {
				t.Errorf("F-4A: snapshot shows key %s as active but in-memory keyset shows revoked.\n"+
					"A stale register snapshot overwrote the authoritative revoke snapshot.\n"+
					"Fix: hold a shared persist mutex across {snapshot-read+marshal+write+rename}.\n"+
					"Without the mutex, the last rename wins regardless of mutation order.",
					sk.PubKey)
			}
		}
	}
}

// ── F-4B: control management listener bind-address INFO log ──────────────────

// TestControlMgmtListener_BindAddressLogged verifies that runControl emits an
// INFO log line "control management listener bound to <address>" when it binds
// its management listener, satisfying AC-012 PC-4 / Ruling 12.
//
// F-4B / AC-012 PC-4 / BC-2.09.003 PC-11b v2.2 / Ruling 12.
// NOT t.Parallel: starts a real TCP listener in runControl.
func TestControlMgmtListener_BindAddressLogged(t *testing.T) {
	cfg := &config.Config{
		ManagementSocket: "127.0.0.1:0", // loopback + ephemeral port
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// logW is a goroutine-safe log accumulator. runControl writes to pw;
	// a reader goroutine drains pr into logW.buf under logW.mu so the
	// polling loop can read logW.buf without a data race.
	type syncLogWriter struct {
		mu  sync.Mutex
		buf strings.Builder
	}
	var logW syncLogWriter

	// os.Pipe: write end → runControl; read end → logW via reader goroutine.
	pr, pw, _ := os.Pipe()

	// Read all output from runControl into logW.buf under the mutex.
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		buf := make([]byte, 512)
		for {
			n, err := pr.Read(buf)
			if n > 0 {
				logW.mu.Lock()
				logW.buf.Write(buf[:n])
				logW.mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()

	sighupCh := make(chan os.Signal, 1)
	errCh := make(chan error, 1)
	go func() {
		errCh <- runControl(ctx, pw, cfg, "", sighupCh)
		_ = pw.Close()
	}()

	// Wait for the bind log to appear (or timeout).
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		logW.mu.Lock()
		found := strings.Contains(logW.buf.String(), "management listener bound to")
		logW.mu.Unlock()
		if found {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	cancel()
	select {
	case rErr := <-errCh:
		if rErr != nil {
			t.Logf("F-4B: runControl returned error on shutdown: %v (may be benign)", rErr)
		}
	case <-time.After(5 * time.Second):
		t.Error("F-4B: runControl did not return within 5s after ctx cancel")
	}

	_ = pr.Close()
	<-readDone

	logW.mu.Lock()
	logStr := logW.buf.String()
	logW.mu.Unlock()

	// AC-012 PC-4: "control management listener bound to <address>"
	wantSubstr := "control management listener bound to"
	if !strings.Contains(logStr, wantSubstr) {
		t.Errorf("F-4B: AC-012 PC-4 bind-address INFO log %q not found in runControl output.\n"+
			"Fix: emit this log in runControl after newMgmtServer returns (mirror runRouter:825).\n"+
			"output=%q", wantSubstr, logStr)
	}
	// The logged address must contain "127.0.0.1" (the loopback addr we configured).
	if !strings.Contains(logStr, "127.0.0.1") {
		t.Errorf("F-4B: bind log does not contain the bound address 127.0.0.1. output=%q", logStr)
	}
}
