package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
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
	// Redirect flag usage output to stdout so --help/-h text goes to stdout
	// and the process exits 0, per AC-012 / BC-2.07.002 EC-003 (Ruling A).
	flag.CommandLine.SetOutput(os.Stdout)
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("usage: sbctl [--target=<addr>] [--key=<path>] [--json] [--timeout=<dur>] <subcommand> [args...]")
		os.Exit(0)
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
		err = connectAndRun(ctx, *target, *key, *jsonOut, "sessions.list", nil, sio)
	case "paths":
		// `sbctl paths list` — canonical per-path metrics command (BC-2.06.003 PC-1).
		if len(args) < 2 || args[1] != "list" {
			fmt.Fprintf(os.Stderr, "usage: sbctl paths list\n")
			os.Exit(2)
		}
		err = runPathsList(ctx, *target, *key, *jsonOut, sio)
	case "router":
		// `sbctl router metrics --svtn=<id>` or `sbctl router status --target <router>`.
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "usage: sbctl router <metrics|status> [flags]\n")
			os.Exit(2)
		}
		switch args[1] {
		case "metrics":
			err = runRouterMetrics(ctx, *target, *key, *jsonOut, args[2:], sio)
		case "status":
			err = runRouterStatus(ctx, *target, *key, *jsonOut, args[2:], sio)
		default:
			fmt.Fprintf(os.Stderr, "unknown router subcommand: %s\n", args[1])
			os.Exit(2)
		}
	case "console":
		err = runConsole(ctx, *target, *key, *jsonOut, args[1:], sio)
	case "admin":
		err = runAdmin(ctx, *target, *key, *jsonOut, args[1:], sio)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\nrun 'sbctl' with no args for usage\n", subcommand)
		os.Exit(2)
	}

	if err != nil {
		os.Exit(1)
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
// or a plain text error otherwise.
func writeError(useJSON bool, code, message string, sio sbctlIO) {
	if useJSON {
		env := newErrorEnvelope(code, message)
		out, err := json.Marshal(env)
		if err != nil {
			_, _ = fmt.Fprintf(sio.err, "marshal error: %s\n", err)
			return
		}
		_, _ = fmt.Fprintln(sio.err, string(out))
		return
	}
	_, _ = fmt.Fprintf(sio.err, "%s %s\n", code, message)
}
