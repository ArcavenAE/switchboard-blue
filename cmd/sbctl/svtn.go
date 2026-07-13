// svtn.go implements the top-level `sbctl svtn` subcommand tree.
//
// Subcommands:
//
//	sbctl svtn status --name=<svtn-name>    (wire: admin.svtn.status; Decision 2 / AC-005..AC-008)
//	sbctl svtn destroy [...]                 (migration shim — usage-error redirect only; Decision 3 / AC-009)
//
// `svtn status` is a genuine standalone dispatch directly to admin.svtn.status
// — not routed through `sbctl admin` framing, matching the bare top-level read
// shape `paths list`/`router status` already use (Decision 2).
//
// `svtn destroy` (top-level) is a migration shim, not a parallel alias: it
// does not implement --id, does not dispatch admin.svtn.destroy, and does not
// duplicate the confirm gate. It always returns a usage error (exit 2)
// redirecting to the canonical `sbctl admin svtn destroy` form (Decision 3).
// The canonical destructive command remains exclusively `sbctl admin svtn
// destroy`, unaffected by this shim.
//
// Purity classification (ARCH-09): effectful-boundary (status) / pure CLI
// dispatch (destroy shim — no RPC, no I/O).
//
// STUB — S-BL.CLI-SURFACE-COMPLETION (Red Gate, BC-5.38.001). Not yet
// implemented; all three function bodies panic unconditionally so no test can
// accidentally pass before Task 2's Green step.
package main

import "context"

// runSvtn dispatches `sbctl svtn <sub-verb>` commands: status (AC-005..AC-008),
// destroy (AC-009), and unknown sub-verb (AC-010).
//
// STUB — S-BL.CLI-SURFACE-COMPLETION Task 2 (Green step) implements sub-verb
// routing (status → runSvtnStatus, destroy → runSvtnDestroyShim, unknown →
// usage error exit 2, same shape as the existing paths/router case arms'
// default arms). Red Gate: body panics unconditionally.
func runSvtn(ctx context.Context, target, keyPath string, useJSON bool, args []string, sio sbctlIO) error {
	panic("not implemented: S-BL.CLI-SURFACE-COMPLETION runSvtn")
}

// runSvtnStatus implements `sbctl svtn status --name=<svtn-name>`.
//
// AC-005..AC-008 / BC-2.07.001 v1.14 PC-4 — dispatches admin.svtn.status via
// the existing connectAndRun pattern. Missing --name is a client-side E-CFG-001
// usage error (exit 2) via usageErrf, per AC-008 PC-3.
//
// STUB — S-BL.CLI-SURFACE-COMPLETION Task 2 (Green step) implements flag
// parsing + dispatch. Red Gate: body panics unconditionally.
//
//nolint:unused // Red Gate stub — wired from runSvtn once Task 2's Green step lands.
func runSvtnStatus(ctx context.Context, target, keyPath string, useJSON bool, args []string, sio sbctlIO) error {
	panic("not implemented: S-BL.CLI-SURFACE-COMPLETION runSvtnStatus")
}

// runSvtnDestroyShim implements the top-level `sbctl svtn destroy` migration
// shim (Decision 3 / AC-009). Always returns a usage error (exit 2) with the
// exact redirect text naming the canonical `sbctl admin svtn destroy` form.
// No --id/--name flag parsing, no RPC dispatch, no confirm-gate invocation —
// the shim never touches args or sio beyond the returned error.
//
// STUB — S-BL.CLI-SURFACE-COMPLETION Task 2 (Green step) implements the
// literal redirect usageErrf return. Red Gate: body panics unconditionally.
//
//nolint:unused // Red Gate stub — wired from runSvtn once Task 2's Green step lands.
func runSvtnDestroyShim(sio sbctlIO) error {
	panic("not implemented: S-BL.CLI-SURFACE-COMPLETION runSvtnDestroyShim")
}
