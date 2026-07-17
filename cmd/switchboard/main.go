package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/arcavenae/switchboard/internal/config"
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
	progName := filepath.Base(args[0])

	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(stdout)
	showVersion := fs.Bool("version", false, "print version and exit")

	if err := fs.Parse(args[1:]); err != nil {
		// --help / -h: usage was already printed to stdout by fs.Parse; treat as success.
		// BC-2.07.002 EC-003 Ruling A: --help exits 0 with no diagnostic accompaniment.
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if *showVersion || fs.NArg() == 0 {
		if _, err := fmt.Fprintf(stdout, "%s %s\n", progName, version); err != nil {
			return fmt.Errorf("write version: %w", err)
		}
		return nil
	}

	subcommand := fs.Arg(0)
	switch subcommand {
	case "version":
		if _, err := fmt.Fprintf(stdout, "%s %s\n", progName, version); err != nil {
			return fmt.Errorf("write version: %w", err)
		}
		return nil

	case "access":
		// Parse access-subcommand flags (--config) from the remaining args.
		accessFS := flag.NewFlagSet("access", flag.ContinueOnError)
		accessFS.SetOutput(stdout)
		configPath := accessFS.String("config", "", "path to YAML config file")
		if err := accessFS.Parse(fs.Args()[1:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}

		// ARCH-06 binding sequence: Config.Validate BEFORE any socket open.
		// If --config is provided, load and validate; abort with E-CFG-* on failure.
		// cfg is threaded into runAccess so that tick_interval is sourced from the
		// validated config (BC-2.09.003 PC-9 / Inv-5 / AC-009).
		var cfg *config.Config
		if *configPath != "" {
			loaded, err := config.LoadFile(*configPath)
			if err != nil {
				return err
			}
			if err := loaded.Validate(); err != nil {
				return err
			}
			cfg = loaded
		}

		// Daemon entry point: install signal handler, then delegate to runAccess.
		// runAccess blocks until shutdown (SIGTERM/SIGINT → exit 0; connect failure
		// or mid-session double-failure → non-nil error → main() calls os.Exit(1)).
		// Diagnostic output goes to os.Stderr; stdout is reserved for structured output.
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
		defer cancel()
		return runAccess(ctx, os.Stderr, cfg)

	case "router":
		routerFS := flag.NewFlagSet("router", flag.ContinueOnError)
		routerFS.SetOutput(stdout)
		configPath := routerFS.String("config", "", "path to YAML config file")
		if err := routerFS.Parse(fs.Args()[1:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}

		var cfg *config.Config
		if *configPath != "" {
			loaded, err := config.LoadFile(*configPath)
			if err != nil {
				return err
			}
			if err := loaded.Validate(); err != nil {
				return err
			}
			cfg = loaded
		}

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
		defer cancel()
		// S-7.04-FU-SIGHUP-RELOAD: dedicated SIGHUP channel, independent of the
		// SIGTERM/SIGINT NotifyContext so a reload signal does not cancel the daemon.
		sighupCh := make(chan os.Signal, 1)
		signal.Notify(sighupCh, syscall.SIGHUP)
		defer signal.Stop(sighupCh)
		// S-BL.CLI-SURFACE-COMPLETION Decision 4: dedicated drain-request channel,
		// signaled by the router.drain RPC handler (bridges into the same
		// shutdown sequence ctx.Done()/SIGTERM already trigger).
		drainRequestCh := make(chan struct{}, 1)
		return runRouter(ctx, os.Stderr, cfg, *configPath, sighupCh, drainRequestCh)

	case "console":
		consoleFS := flag.NewFlagSet("console", flag.ContinueOnError)
		consoleFS.SetOutput(stdout)
		configPath := consoleFS.String("config", "", "path to YAML config file")
		if err := consoleFS.Parse(fs.Args()[1:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}

		var cfg *config.Config
		if *configPath != "" {
			loaded, err := config.LoadFile(*configPath)
			if err != nil {
				return err
			}
			if err := loaded.Validate(); err != nil {
				return err
			}
			cfg = loaded
		}

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
		defer cancel()
		return runConsole(ctx, os.Stderr, cfg)

	case "control":
		controlFS := flag.NewFlagSet("control", flag.ContinueOnError)
		controlFS.SetOutput(stdout)
		configPath := controlFS.String("config", "", "path to YAML config file")
		if err := controlFS.Parse(fs.Args()[1:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}

		var cfg *config.Config
		if *configPath != "" {
			loaded, err := config.LoadFile(*configPath)
			if err != nil {
				return err
			}
			if err := loaded.Validate(); err != nil {
				return err
			}
			cfg = loaded
		}

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
		defer cancel()
		// S-BL.ADMISSION-SYNC-WIRE AC-010 / BC-2.05.009 Inv-5: dedicated SIGHUP
		// channel, independent of the SIGTERM/SIGINT NotifyContext so a reload
		// signal does NOT cancel the daemon (mirrors router-mode SIGHUP pattern).
		sighupCh := make(chan os.Signal, 1)
		signal.Notify(sighupCh, syscall.SIGHUP)
		defer signal.Stop(sighupCh)
		return runControl(ctx, os.Stderr, cfg, *configPath, sighupCh)

	default:
		return fmt.Errorf("unknown subcommand %q; try: access, router, console, control, version", subcommand)
	}
}

func main() {
	if err := run(os.Stdout, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "switchboard: %v\n", err)
		os.Exit(1)
	}
}
