// admin.go implements the `sbctl admin` subcommand tree.
//
// Subcommands:
//
//	sbctl admin key register --key <pubkey> --svtn <id> [--role <role>]
//	sbctl admin key revoke   --key <pubkey> --svtn <id> [--confirm]
//	sbctl admin key expire   --key <pubkey> --svtn <id> --after <duration>
//	sbctl admin list-keys    [--svtn <id>]
//
// All subcommands authenticate to the daemon via the management socket
// (ADR-012 challenge-response) and send RPC requests to the svtnmgmt
// handlers registered on the daemon side.
//
// Resolution of F-P8-001: the canonical CLI surface is `sbctl admin`
// (NOT the removed `sbctl svtn keys register|revoke|expire` path).
// Resolution of F-P8-006: key listing is via `sbctl admin list-keys`.
//
// Purity classification (ARCH-09): effectful-boundary — owns CLI I/O and
// management socket connection.
package main

import (
	"context"
)

// adminKeyRegisterArgs is the wire-format arguments sent to the daemon's
// admin.key.register RPC handler (interface-definitions.md §JSON Output Schema).
//
// Private key material is NEVER transmitted (DI-002; BC-2.05.004 invariant 2).
type adminKeyRegisterArgs struct {
	// SVTNID is the SVTN identifier to register the key for.
	SVTNID string `json:"svtn_id"`
	// Pubkey is the OpenSSH-format Ed25519 public key (authorized_keys format).
	Pubkey string `json:"pubkey"`
	// Role is the authorization role: "control", "console", or "access".
	Role string `json:"role"`
}

// adminKeyRevokeArgs is the wire-format arguments sent to the daemon's
// admin.key.revoke RPC handler.
type adminKeyRevokeArgs struct {
	// SVTNID is the SVTN identifier to revoke the key from.
	SVTNID string `json:"svtn_id"`
	// Pubkey is the OpenSSH-format Ed25519 public key to revoke.
	Pubkey string `json:"pubkey"`
	// Confirm must be true for control-to-control revocation (ADR-004;
	// BC-2.05.004 invariant 1; AC-005).
	Confirm bool `json:"confirm"`
}

// adminKeyExpireArgs is the wire-format arguments sent to the daemon's
// admin.key.expire RPC handler.
type adminKeyExpireArgs struct {
	// SVTNID is the SVTN identifier that owns the key.
	SVTNID string `json:"svtn_id"`
	// Pubkey is the OpenSSH-format Ed25519 public key to expire.
	Pubkey string `json:"pubkey"`
	// After is the human-parseable duration string (e.g. "24h") representing
	// the TTL. Zero or negative durations are rejected with E-CFG-001 by the
	// daemon (BC-2.05.004 postcondition 3; S-6.02 EC-003).
	After string `json:"after"`
}

// runAdmin dispatches `sbctl admin <subcommand>` commands.
//
// Subcommand routing:
//
//	admin key register --key <pubkey> --svtn <id> [--role <role>]
//	admin key revoke   --key <pubkey> --svtn <id> [--confirm]
//	admin key expire   --key <pubkey> --svtn <id> --after <dur>
//	admin list-keys    [--svtn <id>]
//
// Returns a non-nil error on any failure; only main() maps errors to exit codes
// (go.md rule: no log.Fatal / os.Exit outside main).
//
// Traces to BC-2.05.004 (key lifecycle) and BC-2.07.001 (SVTN lifecycle);
// F-P8-001 CLI surface resolution.
func runAdmin(_ context.Context, _, _ string, _ bool, _ []string) error { //nolint:unparam // ctx and params used by implementation; stub panics
	// Ensure wire-type variables are referenced so the compiler does not
	// flag them unused while stubs are in place. These are used by the
	// real sub-dispatch once implemented.
	_ = adminKeyRegisterArgs{}
	_ = adminKeyRevokeArgs{}
	_ = adminKeyExpireArgs{}

	panic("not implemented: runAdmin (BC-2.05.004, BC-2.07.001, F-P8-001)")
}
