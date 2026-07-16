// admission_sync_client.go — control-side admission-state push client.
//
// admissionSyncer is the interface the four admin write handlers use to push
// admission-state changes to configured routers (S-BL.ADMISSION-SYNC-WIRE;
// BC-2.05.009 Rulings 1–2; decisions/S-BL.ADMISSION-SYNC-WIRE-rulings.md v1.2).
//
// A nil admissionSyncer is explicitly permitted — nil means "no routers configured"
// (single-router co-located deployment or router/console/access mode); methods
// are no-ops. Production: *admissionSyncClient. Tests: a mock/stub.
//
// ARCH-08 compliance: this file lives in cmd/switchboard (position 18, the top
// of the import DAG). It imports only internal/admission, internal/mgmt, and
// internal/config — all already imported by mgmt_wire.go.
//
// Purity classification (ARCH-09): boundary — effectful shell that dials TCP,
// sends JSON RPCs, and retries.

package main

import (
	"context"
	"crypto/ed25519"
	"errors"
	"sync"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/config"
)

// errAdmissionSyncNotImplemented is the stub sentinel returned by all
// admissionSyncClient methods. Tests that call these methods will receive this
// error, causing AC-003/004/009 tests to FAIL (Red Gate).
//
// Self-Check (BC-5.38.005 invariant 1): "If I include this real implementation,
// will the test for this function pass trivially without any implementer work?"
// Answer: Yes — therefore this body uses the sentinel error, not real logic.
var errAdmissionSyncNotImplemented = errors.New("admission sync: not implemented")

// Retry-with-backoff constants (BC-2.05.009 Ruling 2; documented per ruling):
//
//	initial delay: 100ms, multiplier: 2, max delay: 10s, max attempts: 5.
const (
	admissionSyncRetryInitial  = 100 * time.Millisecond
	admissionSyncRetryMaxDelay = 10 * time.Second
	admissionSyncRetryMax      = 5
)

// Push command name constants used by both the control-side client and the
// router-side handler registration. Defined here so both files can reference
// them without duplication (AC-002 tests also reference these constants).
const (
	// CmdAdmissionRegister is the internal RPC command for RegisterKey push.
	CmdAdmissionRegister = "internal.admission.register"
	// CmdAdmissionRevoke is the internal RPC command for RevokeKey push.
	CmdAdmissionRevoke = "internal.admission.revoke"
	// CmdAdmissionExpire is the internal RPC command for SetKeyExpiry push.
	CmdAdmissionExpire = "internal.admission.expire"
	// CmdAdmissionRemoveSVTN is the internal RPC command for RemoveSVTN push.
	CmdAdmissionRemoveSVTN = "internal.admission.remove-svtn"
)

// admissionSyncer is the interface the four admin write handlers use to push
// admission-state changes to configured routers.
//
// A nil value is explicitly permitted — nil means "no routers configured";
// methods are no-ops. Production: *admissionSyncClient. Tests: a mock/stub.
//
// svtnID is the resolved [16]byte UUID — NOT the human-readable SVTN name.
// The admin handler (which holds *svtnmgmt.SVTNManager) resolves name→[16]byte
// via m.SVTNByName before calling Push*. The router has no SVTNManager and
// therefore no name→ID map; it must receive the [16]byte directly.
//
// Traces to BC-2.05.009; decisions/S-BL.ADMISSION-SYNC-WIRE-rulings.md v1.2
// Decision 5 (corrected interface — svtnID [16]byte, not svtnName string).
type admissionSyncer interface {
	PushRegisterKey(ctx context.Context, svtnID [16]byte, pubkey ed25519.PublicKey, role admission.KeyRole) error
	PushRevokeKey(ctx context.Context, svtnID [16]byte, pubkey ed25519.PublicKey, role admission.KeyRole, confirm bool) error
	PushSetKeyExpiry(ctx context.Context, svtnID [16]byte, pubkey ed25519.PublicKey, ttl time.Duration) error
	PushRemoveSVTN(ctx context.Context, svtnID [16]byte) error
}

// admissionSyncClient is the control-side push client. It dials each configured
// router management endpoint on demand, completes the mgmt challenge-response
// handshake, sends the internal.admission.* RPC, and reads the response.
//
// Dial-on-demand: no persistent idle connection. Retry-with-backoff per
// admissionSyncRetry* constants above (BC-2.05.009 Ruling 2 Decision 4).
//
// Thread-safe: endpoints are protected by mu; Push* methods may be called
// concurrently from different admin handler goroutines.
type admissionSyncClient struct {
	mu        sync.RWMutex
	endpoints []config.RouterManagementEndpoint
	daemonPriv ed25519.PrivateKey
}

// newAdmissionSyncClient returns an *admissionSyncClient initialised with the
// given endpoints and daemonPriv key. The returned value satisfies admissionSyncer.
//
// endpoints may be empty — push methods become no-ops in that case.
func newAdmissionSyncClient(
	endpoints []config.RouterManagementEndpoint,
	daemonPriv ed25519.PrivateKey,
) *admissionSyncClient {
	return &admissionSyncClient{
		endpoints:  endpoints,
		daemonPriv: daemonPriv,
	}
}

// UpdateEndpoints replaces the client's endpoint list atomically.
// Called from runControl on SIGHUP reload (BC-2.05.009 Invariant 5 / AC-010).
// In-flight pushes are not interrupted; the new list takes effect for the next push.
func (c *admissionSyncClient) UpdateEndpoints(endpoints []config.RouterManagementEndpoint) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.endpoints = endpoints
}

// PushRegisterKey pushes an internal.admission.register RPC to all configured
// router endpoints after a successful admin.key.register on control.
//
// STUB: returns errAdmissionSyncNotImplemented so AC-003/AC-004 tests FAIL at
// the Red Gate. The implementer writes real dial/retry/send logic here.
//
// Self-Check (BC-5.38.005 invariant 1): "If I include this real implementation,
// will the test for this function pass trivially without any implementer work?"
// Answer: Yes — real push logic would satisfy AC-003; therefore this is a stub.
func (c *admissionSyncClient) PushRegisterKey(
	_ context.Context,
	_ [16]byte,
	_ ed25519.PublicKey,
	_ admission.KeyRole,
) error {
	return errAdmissionSyncNotImplemented
}

// PushRevokeKey pushes an internal.admission.revoke RPC to all configured
// router endpoints after a successful admin.key.revoke on control.
//
// STUB: returns errAdmissionSyncNotImplemented so AC-003/AC-004 tests FAIL.
//
// Self-Check (BC-5.38.005 invariant 1): "If I include this real implementation,
// will the test for this function pass trivially without any implementer work?"
// Answer: Yes — therefore stub.
func (c *admissionSyncClient) PushRevokeKey(
	_ context.Context,
	_ [16]byte,
	_ ed25519.PublicKey,
	_ admission.KeyRole,
	_ bool,
) error {
	return errAdmissionSyncNotImplemented
}

// PushSetKeyExpiry pushes an internal.admission.expire RPC to all configured
// router endpoints after a successful admin.key.expire on control.
//
// STUB: returns errAdmissionSyncNotImplemented so AC-003/AC-004 tests FAIL.
//
// Self-Check (BC-5.38.005 invariant 1): "If I include this real implementation,
// will the test for this function pass trivially without any implementer work?"
// Answer: Yes — therefore stub.
func (c *admissionSyncClient) PushSetKeyExpiry(
	_ context.Context,
	_ [16]byte,
	_ ed25519.PublicKey,
	_ time.Duration,
) error {
	return errAdmissionSyncNotImplemented
}

// PushRemoveSVTN pushes an internal.admission.remove-svtn RPC to all configured
// router endpoints after a successful admin.svtn.destroy on control.
//
// STUB: returns errAdmissionSyncNotImplemented so AC-003/AC-004 tests FAIL.
//
// Self-Check (BC-5.38.005 invariant 1): "If I include this real implementation,
// will the test for this function pass trivially without any implementer work?"
// Answer: Yes — therefore stub.
func (c *admissionSyncClient) PushRemoveSVTN(
	_ context.Context,
	_ [16]byte,
) error {
	return errAdmissionSyncNotImplemented
}

// PushFullSnapshot iterates all admitted key entries across all SVTNs in ks
// and issues internal.admission.register (and internal.admission.expire for
// entries with non-zero expiry) to each configured router endpoint.
//
// Called from runControl on startup, before the management server begins serving
// (BC-2.05.009 Postcondition 7 / AC-009 / Decision 10).
//
// STUB: returns errAdmissionSyncNotImplemented so AC-009 tests FAIL.
//
// Self-Check (BC-5.38.005 invariant 1): "If I include this real implementation,
// will the test for this function pass trivially without any implementer work?"
// Answer: Yes — therefore stub.
func (c *admissionSyncClient) PushFullSnapshot(_ context.Context, _ *admission.AdmittedKeySet) error {
	return errAdmissionSyncNotImplemented
}
