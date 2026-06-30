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
	"flag"
	"fmt"
	"time"
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
	// Role is the authorization role of the key being revoked: "control",
	// "console", or "access". The daemon cross-checks this against the stored
	// role to prevent bypassing the confirm gate (HOLD-001 hybrid; E-ADM-019).
	Role string `json:"role"`
	// Confirm must be true for control-to-control revocation (ADR-004;
	// BC-2.05.004 precondition 1; AC-005).
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
func runAdmin(ctx context.Context, target, keyPath string, useJSON bool, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("admin: no subcommand specified; expected 'key' or 'list-keys'")
	}

	switch args[0] {
	case "key":
		return runAdminKey(ctx, target, keyPath, useJSON, args[1:])
	case "list-keys":
		return connectAndRun(ctx, target, keyPath, useJSON, "admin.list-keys", nil)
	default:
		return fmt.Errorf("admin: unknown subcommand %q; expected 'key' or 'list-keys'", args[0])
	}
}

// runAdminKey dispatches `sbctl admin key <subcommand>` commands.
func runAdminKey(ctx context.Context, target, keyPath string, useJSON bool, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("admin key: no subcommand specified; expected 'register', 'revoke', or 'expire'")
	}

	switch args[0] {
	case "register":
		return runAdminKeyRegister(ctx, target, keyPath, useJSON, args[1:])
	case "revoke":
		return runAdminKeyRevoke(ctx, target, keyPath, useJSON, args[1:])
	case "expire":
		return runAdminKeyExpire(ctx, target, keyPath, useJSON, args[1:])
	default:
		return fmt.Errorf("admin key: unknown subcommand %q; expected 'register', 'revoke', or 'expire'", args[0])
	}
}

// runAdminKeyRegister implements `sbctl admin key register`.
//
// Flags:
//
//	--key <pubkey>   OpenSSH-format Ed25519 public key (required)
//	--svtn <id>      SVTN identifier (required)
//	--role <role>    authorization role: control, console, access (default: console)
func runAdminKeyRegister(ctx context.Context, target, keyPath string, useJSON bool, args []string) error {
	fs := flag.NewFlagSet("admin key register", flag.ContinueOnError)
	keyFlag := fs.String("key", "", "Ed25519 public key in OpenSSH format (required)")
	svtnFlag := fs.String("svtn", "", "SVTN identifier (required)")
	roleFlag := fs.String("role", "console", "authorization role: control, console, access")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("admin key register: %w", err)
	}

	if *keyFlag == "" {
		return fmt.Errorf("admin key register: --key is required")
	}
	if *svtnFlag == "" {
		return fmt.Errorf("admin key register: --svtn is required")
	}
	// F-CS-005: validate --role enum before dispatching the RPC.
	// Mirrors the validation in runAdminKeyRevoke (lines ~178-183).
	switch *roleFlag {
	case "control", "console", "access":
		// valid
	default:
		return fmt.Errorf("admin key register: --role must be control, console, or access; got %q", *roleFlag)
	}

	rpcArgs := adminKeyRegisterArgs{
		SVTNID: *svtnFlag,
		Pubkey: *keyFlag,
		Role:   *roleFlag,
	}
	return connectAndRun(ctx, target, keyPath, useJSON, "admin.key.register", rpcArgs)
}

// runAdminKeyRevoke implements `sbctl admin key revoke`.
//
// Flags:
//
//	--key <pubkey>   OpenSSH-format Ed25519 public key (required)
//	--svtn <id>      SVTN identifier (required)
//	--role <role>    authorization role of the key: control, console, access (required)
//	--confirm        required for control-to-control revocation (ADR-004; AC-005)
func runAdminKeyRevoke(ctx context.Context, target, keyPath string, useJSON bool, args []string) error {
	fs := flag.NewFlagSet("admin key revoke", flag.ContinueOnError)
	keyFlag := fs.String("key", "", "Ed25519 public key in OpenSSH format (required)")
	svtnFlag := fs.String("svtn", "", "SVTN identifier (required)")
	roleFlag := fs.String("role", "", "authorization role of the key: control, console, access (required)")
	confirmFlag := fs.Bool("confirm", false, "confirm control-to-control revocation (ADR-004)")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("admin key revoke: %w", err)
	}

	if *keyFlag == "" {
		return fmt.Errorf("admin key revoke: --key is required")
	}
	if *svtnFlag == "" {
		return fmt.Errorf("admin key revoke: --svtn is required")
	}
	if *roleFlag == "" {
		return fmt.Errorf("admin key revoke: --role is required")
	}
	switch *roleFlag {
	case "control", "console", "access":
		// valid
	default:
		return fmt.Errorf("admin key revoke: --role must be control, console, or access; got %q", *roleFlag)
	}

	rpcArgs := adminKeyRevokeArgs{
		SVTNID:  *svtnFlag,
		Pubkey:  *keyFlag,
		Role:    *roleFlag,
		Confirm: *confirmFlag,
	}
	return connectAndRun(ctx, target, keyPath, useJSON, "admin.key.revoke", rpcArgs)
}

// runAdminKeyExpire implements `sbctl admin key expire`.
//
// Flags:
//
//	--key <pubkey>   OpenSSH-format Ed25519 public key (required)
//	--svtn <id>      SVTN identifier (required)
//	--after <dur>    TTL duration (required; e.g. "24h")
func runAdminKeyExpire(ctx context.Context, target, keyPath string, useJSON bool, args []string) error {
	fs := flag.NewFlagSet("admin key expire", flag.ContinueOnError)
	keyFlag := fs.String("key", "", "Ed25519 public key in OpenSSH format (required)")
	svtnFlag := fs.String("svtn", "", "SVTN identifier (required)")
	afterFlag := fs.String("after", "", "TTL duration (required; e.g. \"24h\")")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("admin key expire: %w", err)
	}

	if *keyFlag == "" {
		return fmt.Errorf("admin key expire: --key is required")
	}
	if *svtnFlag == "" {
		return fmt.Errorf("admin key expire: --svtn is required")
	}
	if *afterFlag == "" {
		return fmt.Errorf("admin key expire: --after is required")
	}

	// Client-side validation: parse duration to catch zero/negative early
	// (S-6.02 EC-003; BC-2.05.004 postcondition 3). Zero duration returns error
	// without dialing — avoids a round-trip for an invalid request.
	d, err := time.ParseDuration(*afterFlag)
	if err != nil {
		return fmt.Errorf("admin key expire: invalid --after duration %q: %w", *afterFlag, err)
	}
	if d <= 0 {
		return fmt.Errorf("admin key expire: --after duration must be positive, got %q", *afterFlag)
	}

	rpcArgs := adminKeyExpireArgs{
		SVTNID: *svtnFlag,
		Pubkey: *keyFlag,
		After:  *afterFlag,
	}
	return connectAndRun(ctx, target, keyPath, useJSON, "admin.key.expire", rpcArgs)
}
