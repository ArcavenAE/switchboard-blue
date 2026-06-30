// mgmt_wire.go — management server wiring for cmd/switchboard.
//
// This file provides:
//  1. listenUnixMgmt — opens a Unix management socket with atomic 0600 permissions
//     via syscall.Umask(0177) (AC-014 / CWE-276 / Ruling 4).
//  2. buildMgmtListener — opens the management listener for the given mode.
//  3. startMgmtServer — the shared wiring helper called by runAccess (and future
//     fully-implemented daemon modes) per ARCH-12 §Wiring into cmd/switchboard.
//  4. runRouter, runConsole, runControl — daemon mode stubs (not yet implemented;
//     return not-implemented errors until their owning stories ship). These stubs
//     deliberately do NOT open a management listener — starting one for a daemon
//     that immediately exits would leak a bound socket and an untracked goroutine.
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

	// Pre-bind cleanup: if a stale socket inode exists at path (e.g. from a
	// previous daemon crash or SIGKILL where SetUnlinkOnClose(false) left the
	// inode on disk), remove it before Bind to prevent EADDRINUSE on restart.
	// The os.ModeSocket guard ensures only socket-mode inodes are removed —
	// regular files, directories, and symlinks are left untouched (AC-019 /
	// BC-2.07.004 EC-013 / Ruling O).
	if fi, lstatErr := os.Lstat(path); lstatErr == nil && fi.Mode()&os.ModeSocket != 0 {
		_ = os.Remove(path) // ignore error — Bind will fail if inode still present
	}

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
// (AC-014 / CWE-276). For TCP (console mode) it validates the host is loopback
// before calling net.Listen (BC-2.07.004 EC-013 / Ruling D / VP-073 / AC-014).
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
	// TCP (console mode). Enforce loopback-only binding before calling net.Listen
	// (BC-2.07.004 EC-013 / AC-014 Ruling D / VP-073). Validation must happen here
	// because config.Validate has no mode parameter.
	host, _, splitErr := net.SplitHostPort(address)
	if splitErr != nil {
		return nil, fmt.Errorf("E-CFG-008: management_socket: cannot parse address %q: %w", address, splitErr)
	}
	if !isMgmtLoopbackHost(host) {
		return nil, fmt.Errorf("E-CFG-008: management_socket: console mode requires a loopback address (127.0.0.1, [::1], or localhost); got: %s", address)
	}
	ln, err := net.Listen(network, address)
	if err != nil {
		return nil, fmt.Errorf("buildMgmtListener: %w", err)
	}
	return ln, nil
}

// isMgmtLoopbackHost reports whether host (as returned by net.SplitHostPort) is
// an acceptable loopback address for the console-mode management TCP listener.
// Accepts: "127.0.0.1", "::1", "localhost". Rejects empty string, "0.0.0.0", "::", etc.
func isMgmtLoopbackHost(host string) bool {
	switch strings.ToLower(host) {
	case "localhost", "127.0.0.1", "::1":
		return true
	case "":
		return false
	}
	// Also accept any address in the 127.x.x.x loopback range via net.IP check.
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
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
	handlers []mgmt.Handler, //nolint:unparam // nil for access/console/router (AC-004); BuildAdminHandlers wired in control mode (S-6.06)
	// CR-002 resolution (S-6.06): control mode passes BuildAdminHandlers(svtnMgr)
	// here. Access, console, and router modes pass nil — those daemons correctly
	// return E-RPC-010 for any admin.key.* command (ADR-004 role-exclusion;
	// ARCH-08 §6.6.2; AC-004). The role field in admin.key.revoke args is parsed
	// as admission.KeyRole and passed as currentRole to SVTNManager.RevokeKey
	// (HOLD-001 hybrid; F-002 ruling; S-6.06 AC-002).
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
	// mgmt.NewServer panics on construction invariant violations (nil/short key,
	// empty daemonVersion). Recover here so callers receive an error instead of a
	// binary crash — the caller can log and decide whether to abort or continue.
	var srv *mgmt.Server
	var newServerErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				_ = ln.Close()
				newServerErr = fmt.Errorf("startMgmtServer: NewServer: %v", r)
			}
		}()
		srv = mgmt.NewServer(ln, daemonPrivKey, operatorKeys, handlers, version)
	}()
	if newServerErr != nil {
		return nil, newServerErr
	}

	// WaitGroup-tracked goroutine per ARCH-01 §Goroutine WaitGroup Contract.
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = srv.Serve(ctx)
	}()

	return srv, nil
}

// runRouter is the router-mode daemon entry point.
// The router daemon body (and its management server wiring) ships in its
// owning story; until then this mode is not implemented. It deliberately
// does NOT open a management listener — starting one for a daemon that
// does not run would leak a bound socket and an untracked goroutine.
//
// AC-004 (S-6.06): when router mode is implemented, startMgmtServer MUST pass
// nil (or an empty slice) for admin handlers — router daemons must NOT register
// admin.key.* handlers (ADR-004 role-exclusion; ARCH-08 §6.6.2).
func runRouter(_ context.Context, _ io.Writer, _ *config.Config) error {
	return errors.New("runRouter: not implemented")
}

// runConsole is the console-mode daemon entry point.
// The console daemon body (and its management server wiring) ships in its
// owning story; until then this mode is not implemented. It deliberately
// does NOT open a management listener — starting one for a daemon that
// does not run would leak a bound socket and an untracked goroutine.
//
// AC-004 (S-6.06): when console mode is implemented, startMgmtServer MUST pass
// nil (or an empty slice) for admin handlers — console daemons must NOT register
// admin.key.* handlers (ADR-004 role-exclusion; ARCH-08 §6.6.2).
func runConsole(_ context.Context, _ io.Writer, _ *config.Config) error {
	return errors.New("runConsole: not implemented")
}

// runControl is the control-mode daemon entry point.
// The control daemon body (and its management server wiring) ships in its
// owning story; until then this mode is not implemented. It deliberately
// does NOT open a management listener — starting one for a daemon that
// does not run would leak a bound socket and an untracked goroutine.
//
// TODO(S-6.06): replace stub body with a real implementation that calls
// startMgmtServer with BuildAdminHandlers(svtnMgr) instead of nil, resolving
// CR-002 per story S-6.06 AC-001 / AC-004. Only the control-mode daemon
// registers admin handlers (ADR-004 role-exclusion; ARCH-08 §6.6.2).
func runControl(_ context.Context, _ io.Writer, _ *config.Config) error {
	return errors.New("runControl: not implemented")
}
