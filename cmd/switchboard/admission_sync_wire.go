// admission_sync_wire.go — router-side handler registration for the four
// internal.admission.* push commands (S-BL.ADMISSION-SYNC-WIRE; BC-2.05.009
// Postcondition 1 / BC-2.05.010).
//
// wireAdmissionSyncHandlers is called from runRouter AFTER newMgmtServer and
// BEFORE serveMgmtServer (register-before-serve invariant F-P2L1-001, same
// pattern as wireRouterControlHandlers and wireMetricsHandlers).
//
// The four internal.admission.* commands are router-only — control/console/access
// modes never call wireAdmissionSyncHandlers (ADR-004 / AC-004 role-exclusion).
//
// ARCH-08 compliance: cmd/switchboard (position 18, the top). Imports only
// internal/admission and internal/mgmt, both already imported by mgmt_wire.go.
//
// Purity classification (ARCH-09): boundary — effectful shell that registers
// handlers which mutate the keyset and write snapshot to disk.

package main

import (
	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/mgmt"
)

// wireAdmissionSyncHandlers registers the four internal.admission.* push
// command handlers on srv. Must be called after newMgmtServer and before
// serveMgmtServer (register-before-serve invariant F-P2L1-001).
//
// ks is the router's AdmittedKeySet — the same instance passed to buildRouter.
// snapshotPath is cfg.AdmissionStateFile; an empty string disables snapshot
// persistence (writes are silently skipped).
//
// STUB: registers NOTHING and returns nil so AC-002/005 tests FAIL at the Red
// Gate. The implementer registers the four handlers here.
//
// Self-Check (BC-5.38.005 invariant 1): "If I include this real implementation,
// will the test for this function pass trivially without any implementer work?"
// Answer: Yes — registering the four handlers would satisfy AC-002; therefore
// this is a stub that returns nil without registering anything.
func wireAdmissionSyncHandlers(
	_ *mgmt.Server,
	_ *admission.AdmittedKeySet,
	_ string,
) error {
	// STUB — the implementer registers:
	//   srv.Register(mgmt.Handler{Command: CmdAdmissionRegister, Fn: makeAdmissionRegisterHandler(ks, snapshotPath)})
	//   srv.Register(mgmt.Handler{Command: CmdAdmissionRevoke,   Fn: makeAdmissionRevokeHandler(ks, snapshotPath)})
	//   srv.Register(mgmt.Handler{Command: CmdAdmissionExpire,   Fn: makeAdmissionExpireHandler(ks, snapshotPath)})
	//   srv.Register(mgmt.Handler{Command: CmdAdmissionRemoveSVTN, Fn: makeAdmissionRemoveSVTNHandler(ks, snapshotPath)})
	return nil
}
