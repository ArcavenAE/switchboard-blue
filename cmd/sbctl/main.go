package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// version is overridable at build time via `-ldflags -X main.version=...`.
// Same pattern as cmd/switchboard/main.go so operators can disambiguate
// canonical vs alpha channel builds. Default "dev" identifies unset builds.
var version = "dev"

// defaultKeyPath is the default operator private key path (ARCH-05, interface-definitions.md).
const defaultKeyPath = "~/.ssh/id_ed25519"

// defaultTimeout is the connection timeout default per AC-007 and interface-definitions.md.
const defaultTimeout = 5 * time.Second

// defaultTarget is the default management socket address.
// EC-001: when --target is absent and the default socket is absent, E-NET-001 is returned.
const defaultTarget = "/run/switchboard-router.sock"

// usageError wraps an error that signals a CLI usage mistake (invalid subcommand,
// missing required flag, mutually-exclusive flags, non-interactive session without
// --confirm). main() maps usageError → os.Exit(2); all other errors → os.Exit(1).
// Spec authority: interface-definitions.md v1.18 §174.
type usageError struct {
	err error
}

func (e *usageError) Error() string { return e.err.Error() }
func (e *usageError) Unwrap() error { return e.err }

// usageErrf constructs a usageError with a formatted message.
func usageErrf(format string, args ...any) error {
	return &usageError{err: fmt.Errorf(format, args...)}
}

// reportedError wraps an error whose taxonomy envelope (or plain-text
// taxonomy line) has already been emitted to stderr by writeError. main()'s
// final handler skips its own fmt.Fprintf when it sees this wrapper so the
// stderr stream stays exactly one envelope (or one taxonomy line) —
// preserving whole-stream JSON parseability required by S-6.05
// json-envelope-integrity + BC-2.06.003 AC-006. The wrap preserves the
// errors.As unwrap chain so *usageError still discriminates exit 2 vs 1.
// Spec authority: BC-2.06.003 AC-006 (S-5.02), S-6.05. Issue: #89.
type reportedError struct {
	err error
}

func (e *reportedError) Error() string { return e.err.Error() }
func (e *reportedError) Unwrap() error { return e.err }

// reported wraps err as already-reported. Callers use this when they need to
// pair writeError with a return that has a specific structural type (e.g.
// wrapping usageErrf) so the emission and the wrap happen atomically.
func reported(err error) error {
	if err == nil {
		return nil
	}
	return &reportedError{err: err}
}

func main() {
	// Global flags per interface-definitions.md §sbctl operator CLI and AC-006/AC-007.
	target := flag.String("target", defaultTarget, "daemon address (host:port or unix socket path)")
	key := flag.String("key", defaultKeyPath, "path to operator private key file")
	jsonOut := flag.Bool("json", false, "machine-readable JSON output")
	timeout := flag.Duration("timeout", defaultTimeout, "connection timeout")
	// --version: first-touch operator ergonomics. Prints "<basename> <version>"
	// to stdout and exits 0 (BC-2.07.002 EC-003 Ruling A analog for --version).
	// Basename lets alpha channel builds identify themselves (e.g. "sbctl-a").
	showVersion := flag.Bool("version", false, "print version and exit")
	// Redirect flag usage output to stdout so --help/-h text goes to stdout
	// and the process exits 0, per AC-012 / BC-2.07.002 EC-003 (Ruling A).
	flag.CommandLine.SetOutput(os.Stdout)
	flag.Parse()

	if *showVersion {
		_, _ = fmt.Fprintf(os.Stdout, "%s %s\n", filepath.Base(os.Args[0]), version)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) == 0 {
		// No subcommand supplied — usage error per interface-definitions.md v1.18 §174.
		// Print enumerated subcommand list to stderr so the operator knows what to type.
		fmt.Fprintf(os.Stderr, "usage: sbctl [--target=<addr>] [--key=<path>] [--json] [--timeout=<dur>] <subcommand> [args...]\n")
		fmt.Fprintf(os.Stderr, "available subcommands: sessions, paths, router, console, admin\n")
		os.Exit(2)
	}

	// Single timeout budget threaded through dial + Authenticate + dispatch
	// so total wall-clock honors --timeout once (defect E).
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	subcommand := args[0]
	sio := defaultIO()

	var err error
	switch subcommand {
	case "sessions":
		err = runSessions(ctx, *target, *key, *jsonOut, args[1:], sio)
	case "paths":
		// `sbctl paths list` — canonical per-path metrics command (BC-2.06.003 PC-1).
		// F-P5P8-A-006: distinguish no sub-verb (generic usage hint) from an unknown
		// sub-verb (router-style error naming the typed verb, exit 2).
		if len(args) < 2 {
			err = usageErrf("usage: sbctl paths list")
		} else if args[1] != "list" {
			err = usageErrf("paths: unknown sub-verb %q; expected 'list'", args[1])
		} else {
			err = runPathsList(ctx, *target, *key, *jsonOut, sio)
		}
	case "router":
		// `sbctl router metrics --svtn=<id>` or `sbctl router status --target <router>`.
		if len(args) < 2 {
			err = usageErrf("usage: sbctl router <metrics|status> [flags]")
		} else {
			switch args[1] {
			case "metrics":
				err = runRouterMetrics(ctx, *target, *key, *jsonOut, args[2:], sio)
			case "status":
				err = runRouterStatus(ctx, *target, *key, *jsonOut, args[2:], sio)
			default:
				err = usageErrf("router: unknown subcommand %q; expected 'metrics' or 'status'", args[1])
			}
		}
	case "console":
		err = runConsole(ctx, *target, *key, *jsonOut, args[1:], sio)
	case "admin":
		err = runAdmin(ctx, *target, *key, *jsonOut, args[1:], sio)
	default:
		err = usageErrf("unknown subcommand: %s\nrun 'sbctl' with no args for usage", subcommand)
	}

	if err != nil {
		// #89 / S-6.05 json-envelope-integrity: skip the re-print if the
		// error was already reported to stderr at the call site. This is
		// the single-print contract — stderr on the error path is
		// exactly one envelope (--json) or one taxonomy line (plain).
		var re *reportedError
		if !errors.As(err, &re) {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		var ue *usageError
		if errors.As(err, &ue) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

// runSessions dispatches `sbctl sessions <sub-verb>` commands.
//
// Sub-verb routing per interface-definitions.md v1.18 §71-73 (F-P5P6-A-003):
//
//	list              → sessions.list RPC (may exit 1 on E-NET-001)
//	attach|detach|status → exit 2, not-implemented (deferred to S-BL.DISCOVERY-WIRE family)
//	<anything else>   → exit 2, unknown sub-verb
func runSessions(ctx context.Context, target, keyPath string, useJSON bool, args []string, sio sbctlIO) error {
	subVerb := "list" // bare `sbctl sessions` defaults to list
	if len(args) > 0 {
		subVerb = args[0]
	}

	switch subVerb {
	case "list":
		return connectAndRun(ctx, target, keyPath, useJSON, "sessions.list", nil, sio)
	case "attach", "detach", "status":
		return usageErrf("sessions %s: not implemented; deferred to backlog (S-BL.DISCOVERY-WIRE family)", subVerb)
	default:
		return usageErrf("sessions: unknown sub-verb %q", subVerb)
	}
}

// writeSuccess writes a success JSON envelope to sio.out when --json is set,
// or the raw data bytes otherwise.
func writeSuccess(useJSON bool, data json.RawMessage, sio sbctlIO) {
	if useJSON {
		env := newSuccessEnvelope(data)
		out, err := json.Marshal(env)
		if err != nil {
			_, _ = fmt.Fprintf(sio.err, "marshal error: %s\n", err)
			os.Exit(3)
		}
		_, _ = fmt.Fprintln(sio.out, string(out))
		return
	}
	_, _ = fmt.Fprintln(sio.out, string(data))
}

// writeError writes a failure JSON envelope to sio.err when --json is set,
// or a plain text error otherwise, and returns a reportedError wrapping the
// same code+message so callers can `return writeError(...)` in one line.
// main() sees the reportedError wrapper and skips re-printing, keeping
// stderr to exactly one envelope / taxonomy line (#89 / S-6.05
// json-envelope-integrity).
func writeError(useJSON bool, code, message string, sio sbctlIO) error {
	if useJSON {
		env := newErrorEnvelope(code, message)
		out, err := json.Marshal(env)
		if err != nil {
			_, _ = fmt.Fprintf(sio.err, "marshal error: %s\n", err)
			return reported(fmt.Errorf("%s: marshal envelope: %w", code, err))
		}
		_, _ = fmt.Fprintln(sio.err, string(out))
	} else {
		_, _ = fmt.Fprintf(sio.err, "%s %s\n", code, message)
	}
	return reported(fmt.Errorf("%s: %s", code, message))
}
