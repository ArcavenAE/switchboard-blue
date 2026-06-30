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

	var err error
	switch subcommand {
	case "svtn":
		err = connectAndRun(ctx, *target, *key, *jsonOut, "svtn.list", nil)
	case "sessions":
		err = connectAndRun(ctx, *target, *key, *jsonOut, "sessions.list", nil)
	case "paths":
		// `sbctl paths list` — canonical per-path metrics command (BC-2.06.003 PC-1).
		if len(args) < 2 || args[1] != "list" {
			fmt.Fprintf(os.Stderr, "usage: sbctl paths list\n")
			os.Exit(2)
		}
		err = runPathsList(ctx, *target, *key, *jsonOut)
	case "router":
		// `sbctl router metrics --svtn=<id>` or `sbctl router status --target <router>`.
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "usage: sbctl router <metrics|status> [flags]\n")
			os.Exit(2)
		}
		switch args[1] {
		case "metrics":
			err = runRouterMetrics(ctx, *target, *key, *jsonOut, args[2:])
		case "status":
			err = runRouterStatus(ctx, *target, *key, *jsonOut, args[2:])
		default:
			fmt.Fprintf(os.Stderr, "unknown router subcommand: %s\n", args[1])
			os.Exit(2)
		}
	case "console":
		err = connectAndRun(ctx, *target, *key, *jsonOut, "console.attach", nil)
	case "admin":
		err = runAdmin(ctx, *target, *key, *jsonOut, args[1:])
	case "version":
		err = connectAndRun(ctx, *target, *key, *jsonOut, "version", nil)
	case "ping":
		err = connectAndRun(ctx, *target, *key, *jsonOut, "ping", nil)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", subcommand)
		os.Exit(2)
	}

	if err != nil {
		os.Exit(1)
	}
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
