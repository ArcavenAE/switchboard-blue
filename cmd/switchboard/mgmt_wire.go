// mgmt_wire.go — management server wiring for all four daemon modes.
//
// This file provides:
//  1. listenUnixMgmt — opens a Unix management socket with atomic 0600 permissions
//     via syscall.Umask(0177) (AC-014 / CWE-276 / Ruling 4).
//  2. buildMgmtListener — opens the management listener for the given mode.
//  3. startMgmtServer — the shared wiring helper called by each runXxx function
//     to start the management listener per ARCH-12 §Wiring into cmd/switchboard.
//  4. runRouter, runConsole, runControl — daemon mode stubs (not yet implemented;
//     return not-implemented errors until their owning stories ship).
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
	"syscall"

	"github.com/arcavenae/switchboard/internal/config"
	"github.com/arcavenae/switchboard/internal/mgmt"
)

// listenUnixMgmt opens a Unix management socket at path with 0600 permissions
// atomically by setting the process umask to 0177 before net.Listen and restoring
// the previous umask afterward.
//
// Using the umask-before-Listen approach ensures the socket file is created with
// the correct permissions atomically — there is no TOCTOU window between socket
// creation and permission assignment (CWE-276 / AC-014 / BC-2.07.004 Invariant 7
// / ARCH-12 §Unix Socket Permissions). A chmod-after-Listen approach MUST NOT be
// used as it introduces a TOCTOU window.
//
// SetUnlinkOnClose(false) is applied so the socket file is not automatically
// removed when the listener is closed (callers manage cleanup explicitly on
// daemon restart or via OS-level cleanup). This ensures tests and observability
// tools can stat the socket path after a Shutdown call.
func listenUnixMgmt(path string) (net.Listener, error) {
	old := syscall.Umask(0o177) // 0777 &^ 0177 = 0600
	ln, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	syscall.Umask(old)
	if err != nil {
		return nil, err
	}
	// Do not auto-remove socket on Close — callers manage the socket file lifecycle.
	ln.SetUnlinkOnClose(false)
	return ln, nil
}

// mgmtDefaultSocket returns the mode-specific default management socket address
// when ManagementSocket is empty in config per ARCH-05 §Daemon Management Socket.
func mgmtDefaultSocket(mode string) string {
	switch mode {
	case "router":
		return "/run/switchboard-router.sock"
	case "access":
		return "/run/switchboard-access.sock"
	case "console":
		// Console uses TCP on loopback only (AC-014 / BC-2.07.004 EC-013 / Invariant 7).
		return "127.0.0.1:9091"
	default:
		return "/run/switchboard-control.sock"
	}
}

// mgmtNetwork returns the network type for net.Listen for the given daemon mode.
// console uses TCP (bound to 127.0.0.1 only); all others use Unix sockets (ARCH-05).
func mgmtNetwork(mode string) string {
	if mode == "console" {
		return "tcp"
	}
	return "unix"
}

// resolveManagementSocket returns the effective socket address for the given mode:
// cfg.ManagementSocket (trimmed) if set, otherwise the mode-specific default.
func resolveManagementSocket(cfg *config.Config, mode string) string {
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
// For Unix socket modes it uses listenUnixMgmt to ensure 0600 permissions atomically
// (AC-014 / CWE-276). For TCP (console mode) it uses net.Listen directly.
// Returns a net.Listener that the caller passes to mgmt.NewServer.
func buildMgmtListener(cfg *config.Config, mode string) (net.Listener, error) {
	network, address := mgmtListenAddr(cfg, mode)
	if network == "unix" {
		ln, err := listenUnixMgmt(address)
		if err != nil {
			return nil, fmt.Errorf("buildMgmtListener: %w", err)
		}
		return ln, nil
	}
	// TCP (console mode — binds to 127.0.0.1 only per ARCH-05 / AC-014).
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
// The daemonVersion is sourced from the package-level `version` variable injected
// by ldflags at build time (or "dev" for untagged builds). This satisfies ADR-012
// §Ruling 6 / BC-2.07.004 PC-7 / AC-007.
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

	// Construct the management server. daemonVersion comes from the package-level
	// `version` variable (ldflags-injected; "dev" for unreleased builds).
	srv := mgmt.NewServer(ln, daemonPrivKey, operatorKeys, handlers, version)

	// WaitGroup-tracked goroutine per ARCH-01 §Goroutine WaitGroup Contract.
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = srv.Serve(ctx)
	}()

	return srv, nil
}

// runRouter is the router-mode daemon entry point stub.
func runRouter(ctx context.Context, stderr io.Writer, cfg *config.Config) error {
	_, err := startMgmtServer(ctx, &sync.WaitGroup{}, cfg, "router", nil, nil)
	if err != nil {
		_ = err
	}
	_ = stderr
	return errors.New("runRouter: not implemented")
}

// runConsole is the console-mode daemon entry point stub.
func runConsole(ctx context.Context, stderr io.Writer, cfg *config.Config) error {
	_, err := startMgmtServer(ctx, &sync.WaitGroup{}, cfg, "console", nil, nil)
	if err != nil {
		_ = err
	}
	_ = stderr
	return errors.New("runConsole: not implemented")
}

// runControl is the control-mode daemon entry point stub.
func runControl(ctx context.Context, stderr io.Writer, cfg *config.Config) error {
	_, err := startMgmtServer(ctx, &sync.WaitGroup{}, cfg, "control", nil, nil)
	if err != nil {
		_ = err
	}
	_ = stderr
	return errors.New("runControl: not implemented")
}
