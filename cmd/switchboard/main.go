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
		// Daemon entry point: install signal handler, then delegate to runAccess.
		// runAccess blocks until shutdown (SIGTERM/SIGINT → exit 0; connect failure
		// or mid-session double-failure → non-nil error → main() calls os.Exit(1)).
		// Diagnostic output goes to os.Stderr; stdout is reserved for structured output.
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
		defer cancel()
		return runAccess(ctx, os.Stderr)

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
