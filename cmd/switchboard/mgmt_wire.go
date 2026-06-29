// mgmt_wire.go — management server wiring stubs for all four daemon modes.
//
// This file provides:
//  1. startMgmtServer — the shared wiring helper called by each runXxx function
//     to start the management listener per ARCH-12 §Wiring into cmd/switchboard.
//  2. runRouter, runConsole, runControl — daemon mode stubs (no implementations
//     yet; return not-implemented errors until their owning stories ship).
//
// The wiring pattern for each daemon mode follows ARCH-12 §Daemon Mode Startup:
//
//	operatorKeys := mgmt.NewOperatorKeySet(cfg.AuthorizedOperatorKeys)
//	mgmtLn, err := net.Listen("unix", cfg.ManagementSocket)
//	mgmtSrv := mgmt.NewServer(mgmtLn, daemonPrivKey, operatorKeys, handlers)
//	wg.Add(1)
//	go func() { defer wg.Done(); _ = mgmtSrv.Serve(ctx) }()
//	// On shutdown: mgmtSrv.Shutdown(shutdownCtx)
//
// Default socket paths per ARCH-05 §Daemon Management Socket:
//   - router:  /run/switchboard-router.sock
//   - access:  /run/switchboard-access.sock
//   - console: 127.0.0.1:9091 (TCP)
//   - control: /run/switchboard-control.sock
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	"github.com/arcavenae/switchboard/internal/config"
	"github.com/arcavenae/switchboard/internal/mgmt"
)

// mgmtDefaultSocket returns the mode-specific default management socket address
// when ManagementSocket is empty in config per ARCH-05 §Daemon Management Socket.
func mgmtDefaultSocket(mode string) string {
	switch mode {
	case "router":
		return "/run/switchboard-router.sock"
	case "access":
		return "/run/switchboard-access.sock"
	case "console":
		return "127.0.0.1:9091"
	default:
		return "/run/switchboard-control.sock"
	}
}

// mgmtNetwork returns the network type for net.Listen for the given daemon mode.
// console uses TCP; all others use a Unix socket (ARCH-05).
func mgmtNetwork(mode string) string {
	if mode == "console" {
		return "tcp"
	}
	return "unix"
}

// resolveManagementSocket returns the effective socket address for the given mode:
// cfg.ManagementSocket (trimmed) if set, otherwise the mode-specific default.
//
// todo!() — implementer: uncomment the TrimSpace check and return logic below.
func resolveManagementSocket(cfg *config.Config, mode string) string {
	// todo!() — not implemented; stub returns mode default unconditionally.
	_ = strings.TrimSpace // keep import live
	if cfg != nil && strings.TrimSpace(cfg.ManagementSocket) != "" {
		return cfg.ManagementSocket
	}
	return mgmtDefaultSocket(mode)
}

// mgmtListenAddr returns the net.Listen network and address for the given mode.
func mgmtListenAddr(cfg *config.Config, mode string) (network, address string) {
	return mgmtNetwork(mode), resolveManagementSocket(cfg, mode)
}

// buildMgmtListener opens the management listener for the given mode and config.
// Returns a net.Listener that the caller passes to mgmt.NewServer.
func buildMgmtListener(cfg *config.Config, mode string) (net.Listener, error) {
	network, address := mgmtListenAddr(cfg, mode)
	ln, err := net.Listen(network, address)
	if err != nil {
		return nil, fmt.Errorf("buildMgmtListener: %w", err)
	}
	return ln, nil
}

// parsePEMOperatorKeys parses a slice of PEM-encoded Ed25519 public key strings
// into a slice of ed25519.PublicKey. Any entry that fails to parse is silently
// skipped — config.Validate() has already rejected malformed entries before this
// function is called (E-CFG-009).
func parsePEMOperatorKeys(pemKeys []string) []ed25519.PublicKey {
	out := make([]ed25519.PublicKey, 0, len(pemKeys))
	for _, entry := range pemKeys {
		block, _ := pem.Decode([]byte(entry))
		if block == nil || block.Type != "PUBLIC KEY" {
			continue
		}
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			continue
		}
		edPub, ok := pub.(ed25519.PublicKey)
		if !ok {
			continue
		}
		out = append(out, edPub)
	}
	return out
}

// startMgmtServer starts the management server goroutine for the given daemon mode.
// It applies the mode-specific socket default if ManagementSocket is empty in cfg,
// constructs the mgmt.Server, and adds a wg-tracked goroutine per ARCH-01
// §Goroutine WaitGroup Contract.
//
// Returns the *mgmt.Server so the caller can call Shutdown on graceful exit.
func startMgmtServer(
	ctx context.Context,
	wg *sync.WaitGroup,
	cfg *config.Config,
	mode string,
	daemonPrivKey ed25519.PrivateKey,
	handlers []mgmt.Handler, //nolint:unparam // nil in mode stubs; real runRouter/runConsole/runControl callers pass mode-specific handler slices
) (*mgmt.Server, error) {
	// Parse authorized operator keys from config (PEM → ed25519.PublicKey).
	// Empty list → bootstrap mode (daemon key is the sole authorized key).
	var pemKeys []string
	if cfg != nil {
		pemKeys = cfg.AuthorizedOperatorKeys
	}
	operatorKeys := mgmt.NewOperatorKeySet(parsePEMOperatorKeys(pemKeys))

	// Open the management listener.
	ln, err := buildMgmtListener(cfg, mode)
	if err != nil {
		return nil, fmt.Errorf("startMgmtServer: %w", err)
	}

	// Construct and start the management server.
	srv := mgmt.NewServer(ln, daemonPrivKey, operatorKeys, handlers)

	// WaitGroup-tracked goroutine per ARCH-01 §Goroutine WaitGroup Contract.
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = srv.Serve(ctx)
	}()

	return srv, nil
}

// runRouter is the router-mode daemon entry point stub.
//
// todo!() — implementer: load daemon keypair, build router-mode handler slice
// (router.status, router.metrics, router.reload, svtn.list), call startMgmtServer,
// start router data-plane, wire shutdown sequence per ARCH-12.
func runRouter(ctx context.Context, stderr io.Writer, cfg *config.Config) error {
	// todo!() — not implemented.
	// Stub calls startMgmtServer so the function tree is live from main.go down.
	_, err := startMgmtServer(ctx, &sync.WaitGroup{}, cfg, "router", nil, nil)
	if err != nil {
		_ = err // expected; not propagated — this is a stub
	}
	_ = stderr
	return errors.New("runRouter: not implemented")
}

// runConsole is the console-mode daemon entry point stub.
//
// todo!() — implementer: load daemon keypair, build console-mode handler slice,
// call startMgmtServer (TCP on 127.0.0.1:9091), start console data-plane,
// wire shutdown sequence per ARCH-12.
func runConsole(ctx context.Context, stderr io.Writer, cfg *config.Config) error {
	// todo!() — not implemented.
	_, err := startMgmtServer(ctx, &sync.WaitGroup{}, cfg, "console", nil, nil)
	if err != nil {
		_ = err
	}
	_ = stderr
	return errors.New("runConsole: not implemented")
}

// runControl is the control-mode daemon entry point stub.
//
// todo!() — implementer: load daemon keypair, build control-mode handler slice,
// call startMgmtServer, start control data-plane, wire shutdown sequence per ARCH-12.
func runControl(ctx context.Context, stderr io.Writer, cfg *config.Config) error {
	// todo!() — not implemented.
	_, err := startMgmtServer(ctx, &sync.WaitGroup{}, cfg, "control", nil, nil)
	if err != nil {
		_ = err
	}
	_ = stderr
	return errors.New("runControl: not implemented")
}
