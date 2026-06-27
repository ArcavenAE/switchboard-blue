package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
)

var version = "dev"

// run is the testable entry point. It parses args, dispatches to the
// appropriate subcommand handler, and returns any error.
//
// Subcommands:
//   - "access"   → runAccess (AC-001 through AC-008; S-W3.04)
//   - "version" (or --version flag, or no subcommand) → print version
//
// The run(stdout, args) signature is established by the wave-0 stub and MUST be
// preserved (ARCH-01 §cmd/switchboard Package Layout).
//
// STUB: the "access" dispatch path is compilable but calls runAccess which
// panics. Existing "version" path is preserved and GREEN-BY-DESIGN (zero
// branching in that path once the subcommand check is passed).
func run(stdout io.Writer, args []string) error {
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(stdout)
	showVersion := fs.Bool("version", false, "print version and exit")

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	if *showVersion || fs.NArg() == 0 {
		if _, err := fmt.Fprintf(stdout, "switchboard %s\n", version); err != nil {
			return fmt.Errorf("write version: %w", err)
		}
		return nil
	}

	subcommand := fs.Arg(0)
	switch subcommand {
	case "version":
		if _, err := fmt.Fprintf(stdout, "switchboard %s\n", version); err != nil {
			return fmt.Errorf("write version: %w", err)
		}
		return nil

	case "access":
		// STUB: runAccess panics — all AC-001..AC-008 daemon integration tests
		// are red (Red Gate, BC-5.38.001). Subcommand dispatch compiles cleanly.
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
		defer cancel()
		return runAccess(ctx, stdout)

	default:
		return fmt.Errorf("unknown subcommand %q; try: access, version", subcommand)
	}
}

func main() {
	if err := run(os.Stdout, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "switchboard: %v\n", err)
		os.Exit(1)
	}
}
