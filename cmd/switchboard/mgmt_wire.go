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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/arcavenae/switchboard/internal/config"
	"github.com/arcavenae/switchboard/internal/mgmt"
)

// umaskMu serialises the umask-critical bind(2) syscall in listenUnixMgmt.
// syscall.Umask is process-global: concurrent callers would race on the saved
// umask value and corrupt each other's socket permissions. The mutex keeps the
// umask change window to a single bind(2) call — the minimum possible exposure.
// This is safe in production (one socket per daemon) and eliminates the
// parallel-test race where os.MkdirTemp in another goroutine could receive a
// directory with 0600 permissions (no execute) if a wider critical section were used.
var umaskMu sync.Mutex

// listenUnixMgmt opens a Unix management socket at path with 0600 permissions
// atomically by setting the process umask to 0177 around only the bind(2) syscall.
//
// The implementation uses raw syscalls (Socket, Bind, Listen) instead of
// net.Listen so that the umask change is held only for the single bind(2)
// operation that creates the socket inode in the filesystem. This minimises
// the process-global umask exposure to one syscall, preventing interference
// with concurrent os.MkdirTemp calls in parallel tests while preserving the
// atomic-permission property required by AC-014 / CWE-276 / BC-2.07.004
// Invariant 7. A chmod-after-Listen approach MUST NOT be used (TOCTOU window).
//
// SetUnlinkOnClose(false) is applied so the socket file persists after
// listener Close; callers manage the socket file lifecycle explicitly.
func listenUnixMgmt(path string) (net.Listener, error) {
	// Guard: ensure the parent directory has execute permission before binding.
	// In a parallel test environment, another goroutine's listenUnixMgmt call may
	// have changed the process umask to 0177 while this directory was being created
	// via os.MkdirTemp, yielding a 0600 directory (no execute = no bind). The
	// permission should be 0700; restoring the execute bit here repairs the
	// process-global umask side-effect without affecting the socket file's own
	// permissions (those are still controlled by umask at bind-time below).
	// In production the parent directory is /run or /var/run (always 0755+), so this
	// branch is never taken and has no security or behavioral impact.
	parentDir := filepath.Dir(path)
	if fi, statErr := os.Lstat(parentDir); statErr == nil && fi.Mode().Perm()&0o100 == 0 {
		_ = os.Chmod(parentDir, fi.Mode().Perm()|0o111)
	}

	// Step 1: create the SOCK_STREAM Unix socket fd.
	fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, fmt.Errorf("socket: %w", err)
	}
	// Ensure fd is closed on exec and on error paths below.
	syscall.CloseOnExec(fd)

	// Step 2: bind with umask=0177 so the socket inode is created with 0600.
	// The critical section covers only the single bind(2) syscall — the minimum
	// possible exposure of the process-global umask.
	addr := &syscall.SockaddrUnix{Name: path}
	umaskMu.Lock()
	old := syscall.Umask(0o177) // 0777 &^ 0177 = 0600
	bindErr := syscall.Bind(fd, addr)
	syscall.Umask(old)
	umaskMu.Unlock()

	if bindErr != nil {
		_ = syscall.Close(fd)
		return nil, fmt.Errorf("bind %s: %w", path, bindErr)
	}

	// Step 3: mark the socket as a passive listener.
	if err := syscall.Listen(fd, syscall.SOMAXCONN); err != nil {
		_ = syscall.Close(fd)
		return nil, fmt.Errorf("listen: %w", err)
	}

	// Step 4: wrap fd in a net.Listener via os.File + net.FileListener.
	// net.FileListener duplicates the fd internally, so we close our copy.
	f := os.NewFile(uintptr(fd), path)
	ln, err := net.FileListener(f)
	_ = f.Close()
	if err != nil {
		return nil, fmt.Errorf("FileListener: %w", err)
	}

	// Do not auto-remove socket on Close — callers manage the socket file lifecycle.
	ln.(*net.UnixListener).SetUnlinkOnClose(false)
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
