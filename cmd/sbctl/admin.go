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
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode"

	"golang.org/x/term"
)

// stdinIsTTY reports whether os.Stdin is connected to a terminal.
// Package-level var so tests can swap it out without a real TTY.
// Production value uses golang.org/x/term.IsTerminal.
var stdinIsTTY = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// stdinReader is the reader used for interactive confirm prompts.
// Package-level var so tests can inject a pipe reader without a real TTY.
var stdinReader io.Reader = os.Stdin

// adminKeyRegisterArgs is the wire-format arguments sent to the daemon's
// admin.key.register RPC handler (interface-definitions.md §JSON Output Schema).
//
// Private key material is NEVER transmitted (DI-002; BC-2.05.004 invariant 2).
type adminKeyRegisterArgs struct {
	// SVTNID is the SVTN identifier to register the key for.
	SVTNID string `json:"svtn_id"`
	// Pubkey is the OpenSSH-format Ed25519 public key (authorized_keys format).
	Pubkey string `json:"pubkey_openssh"`
	// Role is the authorization role: "control", "console", or "access".
	Role string `json:"role"`
}

// adminKeyRevokeArgs is the wire-format arguments sent to the daemon's
// admin.key.revoke RPC handler.
type adminKeyRevokeArgs struct {
	// SVTNID is the SVTN identifier to revoke the key from.
	SVTNID string `json:"svtn_id"`
	// Pubkey is the OpenSSH-format Ed25519 public key to revoke.
	Pubkey string `json:"pubkey_openssh"`
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
	Pubkey string `json:"pubkey_openssh"`
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

// boolStringFlag is a flag.Value implementation for a flag that can be passed
// as a bare boolean flag (--confirm) or with an explicit value (--confirm=true,
// --confirm=false, --confirm=some-token). Implementing IsBoolFlag() makes the
// flag package accept the bare form without requiring an argument.
//
// Used for `admin key revoke --confirm` to achieve symmetry with
// `admin svtn destroy --confirm=<token>` (String flag; F-A-009).
type boolStringFlag struct {
	val string
	set bool
}

func (f *boolStringFlag) String() string { return f.val }

func (f *boolStringFlag) Set(v string) error {
	f.val = v
	f.set = true
	return nil
}

// IsBoolFlag tells the flag package that bare --confirm (no =value) is valid
// and equivalent to --confirm=true.
func (f *boolStringFlag) IsBoolFlag() bool { return true }

// isTrue returns true if the flag was set and the value is not "false" or "0".
func (f *boolStringFlag) isTrue() bool {
	return f.set && f.val != "false" && f.val != "0"
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
		fs := flag.NewFlagSet("admin list-keys", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		svtnID := fs.String("svtn", "", "SVTN ID")
		if err := fs.Parse(args[1:]); err != nil {
			return fmt.Errorf("admin list-keys: %w", err)
		}
		if *svtnID == "" {
			return fmt.Errorf("admin list-keys: --svtn <id> is required")
		}
		type listKeysArgs struct {
			SVTNID string `json:"svtn_id"`
		}
		return connectAndRun(ctx, target, keyPath, useJSON, "admin.key.list-keys", listKeysArgs{SVTNID: *svtnID}, sio)
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
//	--name <svtn-name>              Human-readable SVTN label to destroy (required)
//	--confirm <svtn-short-id>       Non-interactive confirmation: SVTN-<first-8-hex-chars>
//	--yes                           Bypass the confirm gate for scripted use (stderr warning emitted)
//
// Confirm-gate behaviour per interface-definitions.md v1.1 §125/§127/§129 and ADR-004:
//
//   - Path 1 (flag supplied): --confirm=SVTN-<8-hex> satisfies the check
//     non-interactively.  The CLI validates the shape; a mismatch aborts before RPC.
//   - Path 2 (flag omitted, stdin is a TTY): interactive prompt on stderr:
//     "Type SVTN-<short-id> to confirm: ".  A matching response dispatches the RPC;
//     any mismatch aborts.
//   - Path 3 (flag omitted, stdin is NOT a TTY): non-interactive scripting signal;
//     aborts with a clear error pointing to --confirm or --yes.
//   - Path 4 (--yes only): bypasses the confirm gate; emits a warning to stderr.
//   - Path 5 (--yes + --confirm): E-CFG-012 usage error, exit 2.
//
// The confirm gate is a human-in-loop typing ceremony (§125), NOT a server-side
// identity match.  Server-side identity-match enforcement is deferred to
// DRIFT-S605-CONFIRM-IDENTITY.
//
// Sends {"command":"admin.svtn.destroy","args":{"name":"<svtn-name>"}} to the
// daemon over the mgmt stream (AC-003 / BC-2.07.001 PC-3).  On success, prints
// confirmation to sio.out.  Exits with non-zero on E-SVTN-003 (SVTN not found).
//
// Traces to BC-2.07.001 PC-3; AC-003; interface-definitions.md v1.1 §117/§125/§127/§129;
// ADR-004; S-6.05.

// confirmSVTNShortIDValid returns true if s matches the "SVTN-<8hexchars>"
// pattern required by the destroy confirmation gate (ADR-004;
// interface-definitions.md v1.1 §125).
//
// Normalizes s to lowercase before validation so that "SVTN-AABBCCDD" is
// accepted alongside "SVTN-aabbccdd" (Fix F-11A-5).
func confirmSVTNShortIDValid(s string) bool {
	// Normalize to lowercase so that "SVTN-AABBCCDD" is accepted alongside
	// "SVTN-aabbccdd" without weakening the shape assertion (Fix F-11A-5).
	s = strings.ToLower(s)
	// After ToLower the prefix is "svtn-"; "SVTN-" would never match.
	const prefix = "svtn-"
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
	confirmFlag := fs.String("confirm", "", "Confirmation short-ID: SVTN-<first-8-hex-chars>")
	yesFlag := fs.Bool("yes", false, "bypass the confirm gate for scripted use (stderr warning emitted)")

	// F-STORY-001: argument parsing MUST precede dispatch.
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("admin svtn destroy: %w", err)
	}
	if *nameFlag == "" {
		return fmt.Errorf("admin svtn destroy: --name is required")
	}

	// Confirm gate (ADR-004; interface-definitions.md v1.1 §125/§127/§129).
	if err := runDestroyConfirmGate(*confirmFlag, *yesFlag, sio); err != nil {
		return err
	}

	rpcArgs := adminSVTNDestroyArgs{Name: *nameFlag}
	if err := connectAndRun(ctx, target, keyPath, useJSON, "admin.svtn.destroy", rpcArgs, sio); err != nil {
		return err
	}

	// Print SVTN name so the operator can confirm which SVTN was destroyed
	// (test: outBuf must contain svtnName — client-side print, not from server response).
	// Gated on !useJSON: in --json mode, writeSuccess (main.go) emits the canonical
	// envelope to sio.out; a trailing plain-text line would corrupt the envelope and
	// violate interface-definitions.md:164 (universal --json envelope contract).
	// F-P7L1-MED-1 fix; peer admin commands (svtn create, key register, key revoke,
	// key expire) all return connectAndRun directly with no post-print.
	if !useJSON {
		_, _ = fmt.Fprintf(sio.out, "destroyed SVTN: %s\n", *nameFlag)
	}
	return nil
}

// runDestroyConfirmGate implements the five-path confirm gate for destructive
// admin operations (interface-definitions.md v1.1 §125/§127/§129; ADR-004).
//
// confirmVal: value of --confirm flag (empty string when flag is absent or --confirm=)
// yes:        value of --yes flag
// sio:        output sinks for the warning and the interactive prompt
//
// Path 1 — --confirm=SVTN-<8-hex> supplied: static shape-check only.
//   - Valid shape → return nil (proceed). Identity-match deferred to DRIFT-S605-CONFIRM-IDENTITY.
//   - Empty value (--confirm= with no value) → treat as absent → fall through to path 2/3.
//   - Invalid shape → error, no RPC.
//
// Path 2 — --confirm absent AND stdin is a TTY: interactive prompt on stderr.
//
//	Prompts "Type SVTN-<short-id> to confirm: " and validates the response shape.
//	Identity-match (verifying the token names this specific SVTN) is deferred.
//
// Path 3 — --confirm absent AND stdin is NOT a TTY: error pointing to --confirm or --yes.
//
// Path 4 — --yes alone: emit stderr warning, return nil (bypass).
//
// Path 5 — --yes + --confirm: E-CFG-012 usage error, return non-nil.
//
// stdinIsTTY and stdinReader are package-level vars so tests can inject fakes.
func runDestroyConfirmGate(confirmVal string, yes bool, sio sbctlIO) error {
	// Path 5: --yes combined with --confirm is a usage error (E-CFG-012; §127).
	if yes && confirmVal != "" {
		return fmt.Errorf("E-CFG-012: --yes cannot be combined with --confirm; pick one")
	}

	// Path 4: --yes alone bypasses the check (§127).
	if yes {
		_, _ = fmt.Fprintln(sio.err, "WARNING: --yes bypasses confirmation; ensure correct --name target before scripting")
		return nil
	}

	// Path 1: --confirm supplied with a non-empty value — static shape check.
	if confirmVal != "" {
		if !confirmSVTNShortIDValid(confirmVal) {
			return fmt.Errorf("admin svtn destroy: invalid --confirm %q; "+
				"expected SVTN-<8 lowercase hex characters>", confirmVal)
		}
		// Shape is valid; proceed. Identity-match deferred to DRIFT-S605-CONFIRM-IDENTITY.
		return nil
	}

	// --confirm absent (empty string).  Check whether stdin is a TTY.
	if !stdinIsTTY() {
		// Path 3: non-interactive session (E-CFG-013).
		return fmt.Errorf("E-CFG-013: non-interactive session: --confirm is required for scripted use; use --confirm=<svtn-short-id> or --yes")
	}

	// Path 2: interactive mode — prompt on stderr and read from stdinReader.
	// The operator must type the SVTN short-ID exactly as printed at create time.
	// Shape-only validation; identity-match deferred to DRIFT-S605-CONFIRM-IDENTITY.
	_, _ = fmt.Fprint(sio.err, "Type the SVTN short-ID (e.g. SVTN-abcd1234) to confirm: ")
	line, err := bufio.NewReader(stdinReader).ReadString('\n')
	if err != nil {
		return fmt.Errorf("interactive confirmation: read error: %w", err)
	}
	line = strings.TrimRight(line, "\r\n ")
	if !confirmSVTNShortIDValid(line) {
		return fmt.Errorf("interactive confirmation failed: expected SVTN-<8 lowercase hex characters>, got: %q", line)
	}
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
//	--key <pubkey>              OpenSSH-format Ed25519 public key (required)
//	--svtn <id>                 SVTN identifier (required)
//	--role <role>               authorization role: control, console, access (default: console)
//	--confirm <svtn-short-id>   Non-interactive confirmation: SVTN-<first-8-hex-chars>
//	--yes                       Bypass the confirm gate for scripted use (stderr warning emitted)
//
// Confirm-gate behaviour mirrors `admin svtn destroy` (interface-definitions.md v1.1 §105/§125;
// ADR-004; Fix F-11A-1).
func runAdminKeyRegister(ctx context.Context, target, keyPath string, useJSON bool, args []string, sio sbctlIO) error {
	fs := flag.NewFlagSet("admin key register", flag.ContinueOnError)
	keyFlag := fs.String("key", "", "Ed25519 public key in OpenSSH format (required)")
	svtnFlag := fs.String("svtn", "", "SVTN identifier (required)")
	roleFlag := fs.String("role", "console", "authorization role: control, console, access")
	confirmFlag := fs.String("confirm", "", "Confirmation short-ID: SVTN-<first-8-hex-chars>")
	yesFlag := fs.Bool("yes", false, "bypass the confirm gate for scripted use (stderr warning emitted)")

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
	// Mirrors the validation in runAdminKeyRevoke.
	switch *roleFlag {
	case "control", "console", "access":
		// valid
	default:
		return fmt.Errorf("admin key register: --role must be control, console, or access; got %q", *roleFlag)
	}

	// Confirm gate (ADR-004; interface-definitions.md v1.1 §105/§125; Fix F-11A-1).
	if err := runDestroyConfirmGate(*confirmFlag, *yesFlag, sio); err != nil {
		return err
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
	confirmFlag := &boolStringFlag{}
	fs.Var(confirmFlag, "confirm", "confirm control-to-control revocation; bare --confirm or --confirm=true (ADR-004)")

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
		Confirm: confirmFlag.isTrue(),
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
