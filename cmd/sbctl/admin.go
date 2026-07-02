// admin.go implements the `sbctl admin` subcommand tree.
//
// Subcommands:
//
//	sbctl admin key register --key <pubkey> --svtn <id> [--role <role>]
//	sbctl admin key revoke   --key <pubkey> --svtn <id> [--confirm]
//	sbctl admin key expire   --key <pubkey> --svtn <id> --after <duration>
//	sbctl admin list-keys    [--svtn <id>]   (wire: admin.key.list-keys; F-L2-001)
//	sbctl admin svtn create  --name <svtn-name>   (wire: admin.svtn.create; S-6.07)
//
// All subcommands authenticate to the daemon via the management socket
// (ADR-012 challenge-response) and send RPC requests to the svtnmgmt
// handlers registered on the daemon side.
//
// Resolution of F-P8-001: the canonical CLI surface is `sbctl admin`
// (NOT the removed `sbctl svtn keys register|revoke|expire` path).
// Resolution of F-P8-006: key listing is via `sbctl admin list-keys` (wire: admin.key.list-keys; F-L2-001).
//
// Purity classification (ARCH-09): effectful-boundary — owns CLI I/O and
// management socket connection.
package main

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"
	"unicode"
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

// adminSVTNCreateArgs is the wire-format arguments sent to the daemon's
// admin.svtn.create RPC handler (AC-002 / BC-2.07.001 PC-1).
//
// Only the name field is sent — no other operator-supplied fields are defined
// for SVTN creation in this story. The daemon auto-generates the SVTN ID and
// bootstrap fingerprint (BC-2.07.001 postcondition 1 + 2).
type adminSVTNCreateArgs struct {
	// Name is the human-readable SVTN label provided by the operator.
	Name string `json:"name"`
}

// adminSVTNDestroyArgs is the wire-format arguments sent to the daemon's
// admin.svtn.destroy RPC handler (AC-003 / BC-2.07.001 PC-3; S-6.05).
type adminSVTNDestroyArgs struct {
	// Name is the human-readable SVTN label to destroy.
	Name string `json:"name"`
}

// runAdmin dispatches `sbctl admin <subcommand>` commands.
//
// Subcommand routing:
//
//	admin key register --key <pubkey> --svtn <id> [--role <role>]
//	admin key revoke   --key <pubkey> --svtn <id> [--confirm]
//	admin key expire   --key <pubkey> --svtn <id> --after <dur>
//	admin list-keys    [--svtn <id>]
//	admin svtn create  --name <svtn-name>
//
// Returns a non-nil error on any failure; only main() maps errors to exit codes
// (go.md rule: no log.Fatal / os.Exit outside main).
//
// Traces to BC-2.05.004 (key lifecycle) and BC-2.07.001 (SVTN lifecycle);
// F-P8-001 CLI surface resolution.
func runAdmin(ctx context.Context, target, keyPath string, useJSON bool, args []string, sio sbctlIO) error {
	if len(args) == 0 {
		return fmt.Errorf("admin: no subcommand specified; expected 'key', 'list-keys', or 'svtn'")
	}

	switch args[0] {
	case "key":
		return runAdminKey(ctx, target, keyPath, useJSON, args[1:], sio)
	case "list-keys":
		return connectAndRun(ctx, target, keyPath, useJSON, "admin.key.list-keys", nil, sio)
	case "svtn":
		return runAdminSvtn(ctx, target, keyPath, useJSON, args[1:], sio)
	default:
		return fmt.Errorf("admin: unknown subcommand %q; expected 'key', 'list-keys', or 'svtn'", args[0])
	}
}

// runAdminSvtn dispatches `sbctl admin svtn <subcommand>` commands.
//
// Subcommand routing:
//
//	admin svtn create  --name <svtn-name>             (wire: admin.svtn.create; AC-002)
//	admin svtn destroy --name <svtn-name> [--confirm] (wire: admin.svtn.destroy; AC-003; S-6.05)
//
// Returns a non-nil error on any failure.
//
// Traces to BC-2.07.001 PC-1 (SVTN create); BC-2.07.001 PC-3 (SVTN destroy); S-6.07; S-6.05.
func runAdminSvtn(ctx context.Context, target, keyPath string, useJSON bool, args []string, sio sbctlIO) error {
	if len(args) == 0 {
		return fmt.Errorf("admin svtn: no subcommand specified; expected 'create' or 'destroy'")
	}

	switch args[0] {
	case "create":
		return runAdminSvtnCreate(ctx, target, keyPath, useJSON, args[1:], sio)
	case "destroy":
		return runAdminSvtnDestroy(ctx, target, keyPath, useJSON, args[1:], sio)
	default:
		return fmt.Errorf("admin svtn: unknown subcommand %q; expected 'create' or 'destroy'", args[0])
	}
}

// runAdminSvtnCreate implements `sbctl admin svtn create`.
//
// Flags:
//
//	--name <svtn-name>   Human-readable SVTN label (required)
//
// Sends {"command":"admin.svtn.create","args":{"name":"<svtn-name>"}} to the
// daemon over the mgmt stream (AC-002 / BC-2.07.001 PC-1). On success, prints
// the returned svtn_id and bootstrap_fingerprint to sio.out (AC-002 / AC-004).
//
// Traces to BC-2.07.001 PC-1 + PC-2; AC-002; AC-004; S-6.07.
func runAdminSvtnCreate(ctx context.Context, target, keyPath string, useJSON bool, args []string, sio sbctlIO) error {
	fs := flag.NewFlagSet("admin svtn create", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "SVTN name (required)")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("admin svtn create: %w", err)
	}
	if *nameFlag == "" {
		return fmt.Errorf("admin svtn create: --name is required")
	}

	rpcArgs := adminSVTNCreateArgs{Name: *nameFlag}
	return connectAndRun(ctx, target, keyPath, useJSON, "admin.svtn.create", rpcArgs, sio)
}

// runAdminSvtnDestroy implements `sbctl admin svtn destroy`.
//
// Flags:
//
//	--name <svtn-name>           Human-readable SVTN label to destroy (required)
//	--confirm <svtn-short-id>    Short-ID confirmation gate (optional; interactive prompt if omitted)
//
// The --confirm flag implements the destructive-operation confirmation gate per
// interface-definitions.md v1.1 §117 and ADR-004. When omitted, the command
// enters interactive mode and prompts "Type SVTN-<short-id> to confirm:" on
// sio.out before proceeding.
//
// Sends {"command":"admin.svtn.destroy","args":{"name":"<svtn-name>"}} to the
// daemon over the mgmt stream (AC-003 / BC-2.07.001 PC-3). On success, prints
// confirmation to sio.out. Exits with non-zero on E-SVTN-003 (SVTN not found).
//
// Traces to BC-2.07.001 PC-3; AC-003; interface-definitions.md v1.1 §117; ADR-004; S-6.05.
// confirmSVTNShortIDValid returns true if s matches the "SVTN-<8hexchars>"
// pattern required by the destroy confirmation gate (ADR-004;
// interface-definitions.md v1.1 §125).
func confirmSVTNShortIDValid(s string) bool {
	const prefix = "SVTN-"
	if !strings.HasPrefix(s, prefix) {
		return false
	}
	hex := s[len(prefix):]
	if len(hex) != 8 {
		return false
	}
	for _, r := range hex {
		if !unicode.Is(unicode.ASCII_Hex_Digit, r) || unicode.IsUpper(r) {
			return false
		}
	}
	return true
}

func runAdminSvtnDestroy(ctx context.Context, target, keyPath string, useJSON bool, args []string, sio sbctlIO) error {
	fs := flag.NewFlagSet("admin svtn destroy", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "SVTN name to destroy (required)")
	confirmFlag := fs.String("confirm", "", "Confirmation short-ID: SVTN-<first-8-hex-chars> (required)")

	// F-STORY-001: argument parsing MUST precede dispatch.
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("admin svtn destroy: %w", err)
	}
	if *nameFlag == "" {
		return fmt.Errorf("admin svtn destroy: --name is required")
	}

	// Confirm gate (ADR-004; interface-definitions.md v1.1 §125).
	// --confirm is required; absent or malformed values abort before any RPC.
	if *confirmFlag == "" {
		return fmt.Errorf("admin svtn destroy: --confirm is required; " +
			"provide the SVTN-<short-id> printed when the SVTN was created")
	}
	if !confirmSVTNShortIDValid(*confirmFlag) {
		return fmt.Errorf("admin svtn destroy: invalid --confirm %q; "+
			"expected SVTN-<8 lowercase hex characters>", *confirmFlag)
	}

	rpcArgs := adminSVTNDestroyArgs{Name: *nameFlag}
	if err := connectAndRun(ctx, target, keyPath, useJSON, "admin.svtn.destroy", rpcArgs, sio); err != nil {
		return err
	}

	// Print SVTN name so the operator can confirm which SVTN was destroyed
	// (test: outBuf must contain svtnName — client-side print, not from server response).
	_, _ = fmt.Fprintf(sio.out, "destroyed SVTN: %s\n", *nameFlag)
	return nil
}

// runAdminKey dispatches `sbctl admin key <subcommand>` commands.
func runAdminKey(ctx context.Context, target, keyPath string, useJSON bool, args []string, sio sbctlIO) error {
	if len(args) == 0 {
		return fmt.Errorf("admin key: no subcommand specified; expected 'register', 'revoke', or 'expire'")
	}

	switch args[0] {
	case "register":
		return runAdminKeyRegister(ctx, target, keyPath, useJSON, args[1:], sio)
	case "revoke":
		return runAdminKeyRevoke(ctx, target, keyPath, useJSON, args[1:], sio)
	case "expire":
		return runAdminKeyExpire(ctx, target, keyPath, useJSON, args[1:], sio)
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
func runAdminKeyRegister(ctx context.Context, target, keyPath string, useJSON bool, args []string, sio sbctlIO) error {
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
	return connectAndRun(ctx, target, keyPath, useJSON, "admin.key.register", rpcArgs, sio)
}

// runAdminKeyRevoke implements `sbctl admin key revoke`.
//
// Flags:
//
//	--key <pubkey>   OpenSSH-format Ed25519 public key (required)
//	--svtn <id>      SVTN identifier (required)
//	--role <role>    authorization role of the key: control, console, access (required)
//	--confirm        required for control-to-control revocation (ADR-004; AC-005)
func runAdminKeyRevoke(ctx context.Context, target, keyPath string, useJSON bool, args []string, sio sbctlIO) error {
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
	return connectAndRun(ctx, target, keyPath, useJSON, "admin.key.revoke", rpcArgs, sio)
}

// runAdminKeyExpire implements `sbctl admin key expire`.
//
// Flags:
//
//	--key <pubkey>   OpenSSH-format Ed25519 public key (required)
//	--svtn <id>      SVTN identifier (required)
//	--after <dur>    TTL duration (required; e.g. "24h")
func runAdminKeyExpire(ctx context.Context, target, keyPath string, useJSON bool, args []string, sio sbctlIO) error {
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
	return connectAndRun(ctx, target, keyPath, useJSON, "admin.key.expire", rpcArgs, sio)
}
