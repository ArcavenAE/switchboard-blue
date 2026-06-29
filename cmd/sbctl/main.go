package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"time"
)

// defaultKeyPath is the default operator private key path (ARCH-05, interface-definitions.md).
const defaultKeyPath = "~/.ssh/id_ed25519"

// defaultTimeout is the connection timeout default per AC-007 and interface-definitions.md.
const defaultTimeout = 5 * time.Second

// defaultTarget is the default management socket address.
// EC-001: when --target is absent and the default socket is absent, E-NET-001 is returned.
const defaultTarget = "/run/switchboard-router.sock"

func main() {
	// Global flags per interface-definitions.md §sbctl operator CLI and AC-006/AC-007.
	target := flag.String("target", defaultTarget, "daemon address (host:port or unix socket path)")
	key := flag.String("key", defaultKeyPath, "path to operator private key file")
	jsonOut := flag.Bool("json", false, "machine-readable JSON output")
	timeout := flag.Duration("timeout", defaultTimeout, "connection timeout")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: sbctl [--target=<addr>] [--key=<path>] [--json] [--timeout=<dur>] <subcommand> [args...]")
		os.Exit(2)
	}

	// Subcommand routing skeleton — empty handlers for now; filled by S-6.02 + S-5.02.
	// Each case eventually calls connectAndRun which wires loadEd25519Key, Authenticate,
	// dispatch, and the JSON envelope helpers.
	subcommand := args[0]

	switch subcommand {
	case "svtn":
		runSvtn(*target, *key, *jsonOut, *timeout, args[1:])
	case "sessions":
		runSessions(*target, *key, *jsonOut, *timeout, args[1:])
	case "paths":
		runPaths(*target, *key, *jsonOut, *timeout, args[1:])
	case "router":
		runRouter(*target, *key, *jsonOut, *timeout, args[1:])
	case "console":
		runConsole(*target, *key, *jsonOut, *timeout, args[1:])
	case "admin":
		runAdmin(*target, *key, *jsonOut, *timeout, args[1:])
	case "version":
		runVersion(*target, *key, *jsonOut, *timeout)
	case "ping":
		runPing(*target, *key, *jsonOut, *timeout)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", subcommand)
		os.Exit(2)
	}
}

// connectAndRun dials the daemon, authenticates, dispatches command with args,
// and writes the result to stdout (or stderr on failure). It is the common
// execution path for all subcommands once the stub is promoted to a real implementation.
//
// On connection failure: prints "E-NET-001 daemon unreachable: <target>: <reason>"
// to stderr, exits 1 (BC-2.07.003 PC-1/PC-2; AC-004).
// On auth failure:       prints "E-ADM-010 authentication failed" to stderr,
// exits 1 (BC-2.07.002 PC-4; AC-003).
//
//nolint:unparam // cmdArgs is always nil in stubs; callers will vary it after S-6.02/S-5.02 fill subcommand handlers
func connectAndRun(target, keyPath string, useJSON bool, timeout time.Duration, command string, cmdArgs any) {
	privKey, err := loadEd25519Key(keyPath)
	if err != nil {
		writeError(useJSON, "E-NET-001", fmt.Sprintf("key load failed: %s", err))
		os.Exit(1)
	}

	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialTarget(dialer, target)
	if err != nil {
		writeError(useJSON, "E-NET-001", fmt.Sprintf("daemon unreachable: %s: %s", target, err))
		os.Exit(1)
	}
	defer func() { _ = conn.Close() }()

	if err = Authenticate(conn, privKey); err != nil {
		writeError(useJSON, "E-ADM-010", "authentication failed")
		os.Exit(1)
	}

	data, err := dispatch(conn, command, cmdArgs)
	if err != nil {
		writeError(useJSON, "E-NET-001", fmt.Sprintf("dispatch failed: %s", err))
		os.Exit(1)
	}

	writeSuccess(useJSON, data)
}

// dialTarget dials the target address. If target starts with '/' it uses a Unix
// socket; otherwise TCP. EC-003: TCP fallback when --target=host:port specified.
func dialTarget(d net.Dialer, target string) (net.Conn, error) {
	if len(target) > 0 && target[0] == '/' {
		return d.Dial("unix", target)
	}
	return d.Dial("tcp", target)
}

// writeSuccess writes a success JSON envelope to stdout when --json is set,
// or the raw data bytes otherwise.
func writeSuccess(useJSON bool, data json.RawMessage) {
	if useJSON {
		env := newSuccessEnvelope(data)
		out, err := json.Marshal(env)
		if err != nil {
			fmt.Fprintf(os.Stderr, "marshal error: %s\n", err)
			os.Exit(3)
		}
		fmt.Println(string(out))
		return
	}
	fmt.Println(string(data))
}

// writeError writes a failure JSON envelope to stderr when --json is set,
// or a plain text error otherwise.
func writeError(useJSON bool, code, message string) {
	if useJSON {
		env := newErrorEnvelope(code, message)
		out, err := json.Marshal(env)
		if err != nil {
			fmt.Fprintf(os.Stderr, "marshal error: %s\n", err)
			return
		}
		fmt.Fprintln(os.Stderr, string(out))
		return
	}
	fmt.Fprintf(os.Stderr, "%s %s\n", code, message)
}

// runSvtn handles the "svtn" subcommand group (S-6.02 fills this in).
func runSvtn(target, key string, jsonOut bool, timeout time.Duration, _ []string) {
	connectAndRun(target, key, jsonOut, timeout, "svtn.list", nil)
}

// runSessions handles the "sessions" subcommand group (S-5.02 fills this in).
func runSessions(target, key string, jsonOut bool, timeout time.Duration, _ []string) {
	connectAndRun(target, key, jsonOut, timeout, "sessions.list", nil)
}

// runPaths handles the "paths" subcommand group (S-5.02 fills this in).
func runPaths(target, key string, jsonOut bool, timeout time.Duration, _ []string) {
	connectAndRun(target, key, jsonOut, timeout, "paths.list", nil)
}

// runRouter handles the "router" subcommand group (S-5.02 fills this in).
func runRouter(target, key string, jsonOut bool, timeout time.Duration, _ []string) {
	connectAndRun(target, key, jsonOut, timeout, "router.status", nil)
}

// runConsole handles the "console" subcommand group (S-7.03 fills this in).
func runConsole(target, key string, jsonOut bool, timeout time.Duration, _ []string) {
	connectAndRun(target, key, jsonOut, timeout, "console.attach", nil)
}

// runAdmin handles the "admin" subcommand group (S-6.02 fills this in).
func runAdmin(target, key string, jsonOut bool, timeout time.Duration, _ []string) {
	connectAndRun(target, key, jsonOut, timeout, "admin.list-keys", nil)
}

// runVersion prints the daemon version (stub; S-6.02 fills this in).
func runVersion(target, key string, jsonOut bool, timeout time.Duration) {
	connectAndRun(target, key, jsonOut, timeout, "version", nil)
}

// runPing checks connectivity to the daemon (stub; S-6.02 fills this in).
func runPing(target, key string, jsonOut bool, timeout time.Duration) {
	connectAndRun(target, key, jsonOut, timeout, "ping", nil)
}
