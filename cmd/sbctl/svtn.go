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
package main

import (
	"context"
	"strings"
)

// runSvtn dispatches `sbctl svtn <sub-verb>` commands: status (AC-005..AC-008),
// destroy (AC-009), and unknown sub-verb (AC-010).
func runSvtn(ctx context.Context, target, keyPath string, useJSON bool, args []string, sio sbctlIO) error {
	if len(args) == 0 {
		return usageErrf("svtn: unknown sub-verb; expected 'status' or 'destroy'")
	}
	switch args[0] {
	case "status":
		return runSvtnStatus(ctx, target, keyPath, useJSON, args[1:], sio)
	case "destroy":
		return runSvtnDestroyShim(sio)
	default:
		return usageErrf("svtn: unknown sub-verb %q; expected 'status' or 'destroy'", args[0])
	}
}

// runSvtnStatus implements `sbctl svtn status --name=<svtn-name>`.
//
// AC-005..AC-008 / BC-2.07.001 v1.14 PC-4 — dispatches admin.svtn.status via
// the existing connectAndRun pattern. Missing --name is a client-side E-CFG-001
// usage error (exit 2) via usageErrf, per AC-008 PC-3.
//
// Output is always the JSON envelope, matching paths ping's design — svtn
// status is a single structured query result with no table representation.
//
//nolint:unparam // useJSON is part of the run* dispatch signature contract (main.go); svtn status always emits the JSON envelope (see above)
func runSvtnStatus(ctx context.Context, target, keyPath string, useJSON bool, args []string, sio sbctlIO) error {
	var name string
	for i, arg := range args {
		if arg == "--name" {
			if i+1 < len(args) {
				name = args[i+1]
			}
		} else if strings.HasPrefix(arg, "--name=") {
			name = strings.TrimPrefix(arg, "--name=")
		}
	}
	if name == "" {
		_ = writeError(true, "E-CFG-001", "svtn status: --name is required", sio)
		return reported(usageErrf("E-CFG-001: svtn status: --name is required"))
	}
	return connectAndRun(ctx, target, keyPath, true, "admin.svtn.status", map[string]string{"name": name}, sio)
}

// runSvtnDestroyShim implements the top-level `sbctl svtn destroy` migration
// shim (Decision 3 / AC-009). Always returns a usage error (exit 2) with the
// exact redirect text naming the canonical `sbctl admin svtn destroy` form.
// No --id/--name flag parsing, no RPC dispatch, no confirm-gate invocation —
// the shim never touches args or sio beyond the returned error.
//
//nolint:unparam // sio is part of the run* dispatch signature contract; the shim deliberately never writes to it (Decision 3 PC-2/PC-3/PC-4)
func runSvtnDestroyShim(sio sbctlIO) error {
	return usageErrf("svtn destroy: use 'sbctl admin svtn destroy --name=<svtn-name> [--confirm=<svtn-short-id>|--yes]'")
}
