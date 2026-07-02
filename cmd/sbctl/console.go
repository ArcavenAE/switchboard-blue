// console.go implements the `sbctl console` subcommand tree.
//
// Subcommands:
//
//	sbctl console attach --target <console_addr> --session <name>
//	sbctl console detach --target <console_addr>
//	sbctl console switch --target <console_addr> --session <name>
//
// Transport: JSON-over-Unix-socket (mgmt-plane, ADR-006/ADR-012; RULING-W6TB-C).
//
// Architecture compliance: cmd/sbctl MUST NOT import internal/routing,
// internal/arq, internal/multipath, or internal/halfchannel
// (ARCH-08 §6.6; RULING-W6TB-C). All three subcommands use the
// management-plane Unix socket, same pattern as `sbctl admin`.
//
// Traces to BC-2.08.001 (Console Remotely Controllable via sbctl).
//
// Purity classification (ARCH-09): effectful-boundary — owns CLI I/O and
// management socket connection.
package main

import (
	"context"
	"flag"
	"fmt"
)

// consoleAttachArgs is the wire-format arguments sent to the console daemon's
// console.attach RPC handler (AC-001; BC-2.08.001 PC-1).
//
// SessionName is the tmux session name to attach to.
type consoleAttachArgs struct {
	// SessionName is the tmux session name to attach to.
	SessionName string `json:"session_name"`
}

// consoleSwitchArgs is the wire-format arguments sent to the console daemon's
// console.switch RPC handler (AC-003; BC-2.08.001 PC-3).
//
// SessionName is the tmux session name to switch to (detach current, attach new).
type consoleSwitchArgs struct {
	// SessionName is the tmux session name to switch to.
	SessionName string `json:"session_name"`
}

// runConsole dispatches `sbctl console <subcommand>` commands.
//
// Subcommand routing:
//
//	console attach --session <name>   (wire: console.attach; AC-001)
//	console detach                    (wire: console.detach; AC-002)
//	console switch --session <name>   (wire: console.switch; AC-003)
//
// Returns a non-nil error on any failure; only main() maps errors to exit codes
// (go.md rule: no log.Fatal / os.Exit outside main).
//
// Traces to BC-2.08.001 PC-1 (attach), PC-2 (detach), PC-3 (switch).
// Transport: mgmt-plane Unix socket (ADR-006/ADR-012; RULING-W6TB-C).
func runConsole(ctx context.Context, target, keyPath string, useJSON bool, args []string, sio sbctlIO) error {
	if len(args) == 0 {
		return fmt.Errorf("console: no subcommand specified; expected 'attach', 'detach', or 'switch'")
	}

	switch args[0] {
	case "attach":
		return runConsoleAttach(ctx, target, keyPath, useJSON, args[1:], sio)
	case "detach":
		return runConsoleDetach(ctx, target, keyPath, useJSON, args[1:], sio)
	case "switch":
		return runConsoleSwitch(ctx, target, keyPath, useJSON, args[1:], sio)
	default:
		return fmt.Errorf("console: unknown subcommand %q; expected 'attach', 'detach', or 'switch'", args[0])
	}
}

// runConsoleAttach implements `sbctl console attach`.
//
// Flags:
//
//	--session <name>   tmux session name to attach to (required)
//
// Sends {"command":"console.attach","args":{"session_name":"<name>"}} to the
// console daemon over the mgmt Unix socket (BC-2.08.001 PC-1; AC-001).
//
// Error codes:
//   - E-SES-001 (E-RPC-011 envelope): unknown session name
//   - E-ADM-006 (E-RPC-011 envelope): auth denied
//
// Traces to BC-2.08.001 PC-1; AC-001.
func runConsoleAttach(ctx context.Context, target, keyPath string, useJSON bool, args []string, sio sbctlIO) error {
	fs := flag.NewFlagSet("console attach", flag.ContinueOnError)
	sessionFlag := fs.String("session", "", "tmux session name to attach to (required)")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("console attach: %w", err)
	}
	if *sessionFlag == "" {
		return fmt.Errorf("console attach: --session is required")
	}

	rpcArgs := consoleAttachArgs{SessionName: *sessionFlag}
	return connectAndRun(ctx, target, keyPath, useJSON, "console.attach", rpcArgs, sio)
}

// runConsoleDetach implements `sbctl console detach`.
//
// Sends {"command":"console.detach"} to the console daemon over the mgmt
// Unix socket (BC-2.08.001 PC-2; AC-002). Detach does not close the session.
//
// Error codes:
//   - E-SES-004: not attached for command
//
// Traces to BC-2.08.001 PC-2; AC-002.
func runConsoleDetach(ctx context.Context, target, keyPath string, useJSON bool, args []string, sio sbctlIO) error {
	fs := flag.NewFlagSet("console detach", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("console detach: %w", err)
	}

	return connectAndRun(ctx, target, keyPath, useJSON, "console.detach", nil, sio)
}

// runConsoleSwitch implements `sbctl console switch`.
//
// Flags:
//
//	--session <name>   tmux session name to switch to (required)
//
// Sends {"command":"console.switch","args":{"session_name":"<name>"}} to the
// console daemon over the mgmt Unix socket (BC-2.08.001 PC-3; AC-003).
// Atomically detaches from the current session and attaches to the named session.
//
// Error codes:
//   - E-SES-001 (E-RPC-011 envelope): unknown session name
//   - E-SES-004: not attached for command (detach leg failed)
//   - E-ADM-006 (E-RPC-011 envelope): auth denied
//
// Traces to BC-2.08.001 PC-3; AC-003.
func runConsoleSwitch(ctx context.Context, target, keyPath string, useJSON bool, args []string, sio sbctlIO) error {
	fs := flag.NewFlagSet("console switch", flag.ContinueOnError)
	sessionFlag := fs.String("session", "", "tmux session name to switch to (required)")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("console switch: %w", err)
	}
	if *sessionFlag == "" {
		return fmt.Errorf("console switch: --session is required")
	}

	rpcArgs := consoleSwitchArgs{SessionName: *sessionFlag}
	return connectAndRun(ctx, target, keyPath, useJSON, "console.switch", rpcArgs, sio)
}
