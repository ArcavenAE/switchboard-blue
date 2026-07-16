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
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/config"
	"github.com/arcavenae/switchboard/internal/drain"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/mgmt"
	"github.com/arcavenae/switchboard/internal/multipath"
	"github.com/arcavenae/switchboard/internal/netingress"
	"github.com/arcavenae/switchboard/internal/outerassembler"
	"github.com/arcavenae/switchboard/internal/routing"
	"github.com/arcavenae/switchboard/internal/session"
	"github.com/arcavenae/switchboard/internal/svtnmgmt"
	"github.com/arcavenae/switchboard/internal/upstreamdial"
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

// newMgmtServer constructs a management server for the given daemon mode but does
// NOT start the Serve goroutine. Callers MUST register all handlers via
// wireMetricsHandlers (or equivalent) before calling serveMgmtServer, to satisfy
// the register-before-serve invariant (F-P2L1-001 data-race fix).
//
// The daemonVersion is sourced from the package-level `version` variable injected
// by ldflags at build time (or "dev" for untagged builds). This satisfies ADR-012
// §Ruling 6 / BC-2.07.004 PC-7 / AC-007.
func newMgmtServer(
	cfg *config.Config,
	mode string,
	daemonPrivKey ed25519.PrivateKey,
	handlers []mgmt.Handler, //nolint:unparam // nil for access/console/router (AC-004); BuildAdminHandlers wired in control mode (S-6.06)
	// CR-002 resolution (S-6.06): control mode passes BuildAdminHandlers(svtnMgr)
	// here. Access, console, and router modes pass nil — those daemons correctly
	// return E-RPC-010 for any admin.key.* command (ADR-004 role-exclusion;
	// ARCH-04 disambiguation table; AC-004). The role field in admin.key.revoke args is parsed
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
		return nil, fmt.Errorf("newMgmtServer: %w", err)
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
				newServerErr = fmt.Errorf("newMgmtServer: NewServer: %v", r)
			}
		}()
		srv = mgmt.NewServer(ln, daemonPrivKey, operatorKeys, handlers, version)
	}()
	if newServerErr != nil {
		return nil, newServerErr
	}

	return srv, nil
}

// serveMgmtServer starts the management server's Serve goroutine under the given
// WaitGroup. All handlers MUST have been registered via Register (or
// wireMetricsHandlers) before this call — Serve reads s.handlers without a lock.
//
// Returns the *mgmt.Server so the caller can call Shutdown on graceful exit.
// This is a separate step from newMgmtServer to enforce the register-before-serve
// ordering (F-P2L1-001 data-race fix).
func serveMgmtServer(
	ctx context.Context,
	wg *sync.WaitGroup,
	srv *mgmt.Server,
) *mgmt.Server {
	// WaitGroup-tracked goroutine per ARCH-01 §Goroutine WaitGroup Contract.
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = srv.Serve(ctx)
	}()
	return srv
}

// startMgmtServer is a convenience wrapper combining newMgmtServer and
// serveMgmtServer for callers that do not need to register additional handlers
// between construction and serve. It is used by tests and by daemon modes that
// pass all handlers via the initial handlers slice.
//
// Production daemon modes (runAccess, runControl) use the explicit two-phase
// newMgmtServer → wireMetricsHandlers → serveMgmtServer sequence to satisfy
// the register-before-serve ordering (F-P2L1-001).
func startMgmtServer(
	ctx context.Context,
	wg *sync.WaitGroup,
	cfg *config.Config,
	mode string,
	daemonPrivKey ed25519.PrivateKey,
	handlers []mgmt.Handler, //nolint:unparam // nil in tests; non-nil when BuildAdminHandlers is passed
) (*mgmt.Server, error) {
	srv, err := newMgmtServer(cfg, mode, daemonPrivKey, handlers)
	if err != nil {
		return nil, err
	}
	return serveMgmtServer(ctx, wg, srv), nil
}

// Compile-time check: *upstreamdial.Connector satisfies Handle.
var _ upstreamdial.Handle = (*upstreamdial.Connector)(nil)

// nodeConn wraps the netingress-created Send/Done channels for one
// connected node in the per-node send map. Send is NEVER closed, by any
// goroutine, for the lifetime of the process (single-closer/
// no-send-after-close invariant — closing it would race the drain
// observer's concurrent sendMap.Range send). Done IS closed exactly once,
// but has two independent trigger call sites (this connection's own
// teardown, and the router-wide drain-shutdown flush pass) — both MUST
// close it via doneOnce, never via a direct close(nc.done).
type nodeConn struct {
	send         chan []byte
	done         chan struct{}
	doneOnce     *sync.Once
	writerExited chan struct{} // closed by the writer goroutine's own defer, on exit — single closer by construction, no sync.Once companion needed.
}

// nodeConnEvent identifies why nodeConnHook fired.
type nodeConnEvent int

const (
	nodeConnRegistered nodeConnEvent = iota // OnAccept stored the send channel
	nodeConnRemoved                         // cleanup deleted the map entry
)

// nodeConnHook, when non-nil, is called synchronously from the OnAccept
// registration closure and from the per-connection cleanup closure. Same
// shape and rationale as drainCoordHook: gives same-package tests an
// observable for per-node send-map lifecycle without exporting the map.
// nil in production (no-op). Any test that sets this MUST NOT call
// t.Parallel() — it is package-level mutable state.
var nodeConnHook func(event nodeConnEvent, ifaceID routing.InterfaceID)

// drainObserverFiredHook, when non-nil, is called synchronously at the top
// of the production drain observer's body — the single observer runRouter
// registers with drainCoord at drainCoord-construction time — before the
// send-map Range call, unconditionally on every invocation (whether or not
// any node is connected). nil in production (no-op). Any test that sets
// this MUST NOT call t.Parallel() — it is package-level mutable state.
var drainObserverFiredHook func()

// drainFlushTimeout bounds the router-wide shutdown-flush phase between
// drainCoord.Wait and ingressCancel(): a hard ceiling on ADDED shutdown
// latency, independent of and not deducted from cfg.DrainTimeout, since
// writers flush in parallel and the drain observer queues exactly one
// frame per node.
const drainFlushTimeout = 200 * time.Millisecond

// runRouter is the router-mode daemon entry point (S-BL.ROUTER-RUNTIME).
//
// It generates an ephemeral Ed25519 keypair, starts the management server
// with a nil admin-handler set (ADR-004 role-exclusion; ARCH-04 disambiguation
// table; AC-004 — router daemons MUST NOT register admin.key.* handlers),
// binds a data-plane TCP listener on cfg.ListenAddr, and blocks until ctx is
// cancelled (ARCH-01 §Goroutine WaitGroup Contract).
//
// The data-plane listener reads self-delimiting framed messages from each
// accepted connection and dispatches them to routing.RouteFrame via the
// internal/netingress package (S-BL.NI). RouteFrame enforces the fail-closed
// HMAC-then-admission ordering (ADR-009; BC-2.05.008) and records per-source
// HMAC failures against the FailureCounter for E-ADM-017 (BC-2.05.005 PC-3).
//
// Register-before-serve ordering (F-P2L1-001 / F-P2L1-002):
//  1. newMgmtServer — construct server (no goroutine)
//  2. wireMetricsHandlers — register metrics RPC handlers before Serve
//  3. buildRouter — construct routing.Router + FailureCounter
//  4. bind data-plane listener (cfg.ListenAddr — BC-2.09.003 PC-9 application)
//  5. serveMgmtServer — start Serve goroutine
//  6. netingress.Serve — start data-plane accept loop
//
// Ruling J: any mgmt-start failure aborts daemon startup — no degraded-management
// mode. A data-plane bind failure similarly aborts (the router is useless
// without a listen address).
//
// Logging: the injected writer receives the listen address and management
// socket path on successful startup. Callers (main.go) pass os.Stderr in
// production; tests may pass nil (in which case the messages are suppressed).
//
// S-7.04 application closures (BC-2.09.003 DEFERRED-APPLICATION table):
//   - drain_timeout      → drainTimeoutFor(cfg) drives the drain coordinator
//     (BC-2.09.003 PC-7; drain.New; shutdown-time Signal/Wait)
//   - keepalive_interval → keepaliveIntervalFor(cfg) resolves and is emitted
//     at the observability seam; the reconnect-side keepalive ticker ships in
//     this story inside upstreamdial.Connector (dialLoop's keepaliveTick /
//     maintainConn; BC-2.09.003 PC-8; FM-009).
//     MUST NOT be routed into sweepDeadline (console eviction — different
//     semantic; BC-2.09.003 PC-8 normative note).
//   - upstream_routers   → upstreamRoutersFor(cfg) resolves and is emitted
//     at the observability seam; a non-empty list signals PE-mode graduation
//     eligibility (BC-2.09.001 PC-1). Live upstream connection establishment
//     ships in this story via upstreamdial.New/Start (dial loop +
//     outerassembler.Envelope session bootstrap).
//
// #SHIPPED — SIGHUP config reload (BC-2.09.001 PC-1 / S-7.04-FU-SIGHUP-RELOAD)
// is implemented in the select loop below.
// #SHIPPED — PE-mode upstream connector (S-7.04-FU-PE-CONNECTOR): constructed
// via upstreamdial.New, started below, address list reloaded on SIGHUP via
// ReloadAddrs, stopped at graceful shutdown.
//
// #SHIPPED — DRAIN-over-SVTN wire protocol (BC-2.09.002 Inv-1;
// S-7.04-FU-DRAIN-WIRE): the single startup drain observer broadcasts a
// FrameTypeCtl DRAIN frame to every connected node's per-node send channel
// at drainCoord.Signal time; the netingress OnAccept seam registers and
// deregisters nodes in the per-node send map (Q-SEAM, Q-SINGLE-OBS).
func runRouter(ctx context.Context, w io.Writer, cfg *config.Config, configPath string, sighupCh chan os.Signal, drainRequestCh chan struct{}) error {
	// Router mode requires a loaded config to bind the data-plane listener.
	// main.go leaves cfg nil when --config is omitted; bare `switchboard router`
	// would then nil-deref on cfg.ListenAddr and panic — a violation of the
	// no-Go-panic operator taxonomy asserted by T2-4 / T3-4. Guard explicitly
	// so the failure surfaces as a taxonomy-shaped error instead.
	if cfg == nil {
		return fmt.Errorf("runRouter: E-CFG-004: --config is required for router mode (no config loaded)")
	}

	// Generate ephemeral daemon keypair (BC-2.07.004 Precondition 3 / AC-015).
	_, daemonPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("runRouter: generate daemon keypair: %w", err)
	}

	// Phase (a): construct mgmt server with NIL admin handlers.
	// AC-004 / ADR-004: router MUST NOT register admin.key.* handlers.
	mgmtSrv, mgmtErr := newMgmtServer(cfg, "router", daemonPriv, nil)
	if mgmtErr != nil {
		return fmt.Errorf("runRouter: construct management server: %w", mgmtErr)
	}

	// Phase (b): build the Router with a live FailureCounter.
	// buildRouter (defined in access.go) constructs a routing.Router with the
	// stdLogger wrapping w (nil-safe) and a FailureCounter at threshold=5,
	// window=60s per BC-2.05.005 PC-3. The FailureCounter now drives a live
	// E-ADM-017 path — no longer dormant.
	// Reordered above wireMetricsHandlers so the router is available to the
	// metrics wiring (S-BL.PATH-TRACKER-WIRING) as the source-of-PathTracker
	// registrations for paths.list.
	routerLogger := newStdLogger(w)
	routerKS := admission.NewAdmittedKeySet()

	// Phase (b1): load snapshot from admission_state_file before serving
	// (S-BL.ADMISSION-SYNC-WIRE AC-007; BC-2.05.010 PC-6/7). Fail-closed on
	// corrupt/unknown-schema file (Decision 7c / EC-011 / E-KEY-002).
	// Absent path → no-op; absent file → empty keyset + INFO log (Decision 7b).
	if err := loadSnapshotFromFile(cfg.AdmissionStateFile, routerKS); err != nil {
		return fmt.Errorf("runRouter: load admission snapshot: %w", err)
	}
	if cfg.AdmissionStateFile != "" {
		if _, statErr := os.Stat(cfg.AdmissionStateFile); os.IsNotExist(statErr) {
			if w != nil {
				_, _ = fmt.Fprintf(w, "switchboard router: admission_state_file not found — starting with empty keyset\n")
			}
		}
	}

	router := buildRouter(routerKS, routerLogger)

	// Phase (c): register metrics handlers before Serve starts. Passing the
	// live router installs a forwarding-entry hook that populates the paths.list
	// source on RegisterForwardingEntry (S-BL.PATH-TRACKER-WIRING).
	if err := wireMetricsHandlers(mgmtSrv, router); err != nil {
		return fmt.Errorf("runRouter: wire metrics handlers: %w", err)
	}

	// Phase (c2): register router-mode-exclusive control handlers (router.reload,
	// router.drain) before Serve starts — same register-before-serve ordering as
	// wireMetricsHandlers (S-BL.CLI-SURFACE-COMPLETION Decision 4 / AC-013).
	if err := wireRouterControlHandlers(mgmtSrv, configPath, sighupCh, drainRequestCh); err != nil {
		return fmt.Errorf("runRouter: wire router control handlers: %w", err)
	}

	// Phase (c3): register internal.admission.* push handlers (router-only;
	// ADR-004 role-exclusion / AC-004). Register before Serve (F-P2L1-001).
	// S-BL.ADMISSION-SYNC-WIRE AC-002/005/008.
	if err := wireAdmissionSyncHandlers(mgmtSrv, routerKS, cfg.AdmissionStateFile); err != nil {
		return fmt.Errorf("runRouter: wire admission sync handlers: %w", err)
	}

	// Phase (d): bind the data-plane TCP listener on cfg.ListenAddr
	// (BC-2.09.003 PC-9 application point — the deferred listen_addr binding
	// now applies at ingress-listener construction time).
	dataLn, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("runRouter: bind data-plane listener %q: %w", cfg.ListenAddr, err)
	}

	// Phase (e): start the mgmt Serve goroutine.
	var mgmtWG sync.WaitGroup
	serveMgmtServer(ctx, &mgmtWG, mgmtSrv)

	// Phase (f): start the netingress data-plane accept loop.
	// Each accepted connection runs ServeConn: read a framed message, dispatch
	// to routing.RouteFrame, repeat until EOF or ctx cancel. RouteFrame handles
	// fail-closed drop when no forwarding entry exists (auth key unavailable)
	// or when HMAC verification fails — both paths already log E-ADM-016 and
	// record E-ADM-017 counter increments per BC-2.05.008 PC-4 / PC-5.
	// ingressCtx is deliberately DETACHED from ctx's cancellation via
	// WithoutCancel — the explicit ingressCancel() call in the shutdown
	// block (and the defer safety net below) must remain the SOLE
	// teardown trigger, or the drain flush pass runs against already-dead
	// conns; F-DW-IMPL-001, placement note v1.10. Do NOT reparent to ctx.
	ingressCtx, ingressCancel := context.WithCancel(context.WithoutCancel(ctx))
	defer ingressCancel()

	// Per-node send map (Q-SEAM): value type *nodeConn, populated by the
	// onAccept closure below (netingress owns DATA creation; this closure
	// owns BEHAVIOR only — see nodeConn's doc comment). writerWG tracks the
	// writer goroutines onAccept starts; joined in the shutdown block below
	// (Shutdown ordering guarantee).
	var sendMap sync.Map // routing.InterfaceID -> *nodeConn
	var writerWG sync.WaitGroup

	route := func(hdr frame.OuterHeader, payload []byte) error {
		// Q-CTL-GUARD: ctl frame receive path — exclusively in this
		// closure, NOT the frozen PE FrameFn closure (Q2-AMENDED). Every
		// ctl frame arriving here is terminal-consumer by construction
		// under the current architecture (BC-2.01.008 v1.1 Invariant 2 —
		// no inter-router relay path exists anywhere in this codebase);
		// REVISIT this unconditional posture if a future story introduces
		// router-to-router forwarding.
		if hdr.FrameType == frame.FrameTypeCtl {
			if len(payload) < 4 {
				// E-PRT-002: control frame truncated. No conn is in scope
				// here — this closure's signature is func(hdr, payload)
				// error; there is no per-connection dispatch seam with
				// conn access (F-DW-SP3-003) — do not add one for this log
				// line.
				routerLogger.Log(fmt.Sprintf(
					"netingress: E-PRT-002: ctl frame payload_len=%d < 4; discarding",
					len(payload)))
				// silent-ignore: no connection close per BC-2.01.008 EC-002
				return nil
			}
			controlType := payload[0]
			switch controlType {
			case 0x01: // DRAIN — router-originated broadcast (Q2-AMENDED);
				// a router does not act on an inbound DRAIN byte from a node.
			default:
				// Unknown/forward-compat control_type (includes the
				// reserved-but-undispatched 0x02 RESYNC opcode until
				// S-BL.RESYNC-FRAME lands). BC-2.01.008 PC-4 (v1.1): silent-
				// ignore means NO logging of any kind on this path — not
				// even a diagnostic/informational line — and no connection
				// close. Do NOT add a log call to this arm: a well-formed-
				// but-unrecognized opcode is ordinary forward-compatible
				// protocol evolution, not an anomaly (asymmetric with the
				// E-PRT-002 branch above, which DOES log — truncation is
				// corruption, diagnostic-worthy).
			}
			return nil
		}
		return routing.RouteFrame(hdr, payload, router)
	}

	// onAccept is the BEHAVIOR half of the Q-SEAM ownership split:
	// netingress.Serve has already allocated h.IfaceID/h.Send/h.Done (DATA
	// half) before calling this closure. onAccept wraps them in a
	// *nodeConn, stores it in the per-node send map, starts the sole
	// writer goroutine for this connection, and returns a behavior-cleanup
	// func() that deregisters the node when its connection closes.
	onAccept := func(conn net.Conn, h netingress.NodeHandle) func() {
		nc := &nodeConn{
			send:         h.Send,
			done:         h.Done,
			doneOnce:     new(sync.Once),
			writerExited: make(chan struct{}),
		}
		sendMap.Store(h.IfaceID, nc)

		writerWG.Add(1)
		go func() {
			defer writerWG.Done()
			// Registered SECOND — after writerWG.Done() above — so Go's
			// LIFO defer order runs THIS one FIRST, i.e. close(writerExited)
			// happens-before writerWG.Done(). Load-bearing as of v1.7
			// (F-DW-SP7-001): the shutdown block's trailing
			// snapshotWG.Wait() join depends on every writerExited already
			// being closed by the time the final writerWG.Wait() returns.
			// Do NOT reorder these two defers.
			defer close(nc.writerExited)
			for {
				select {
				case msg := <-nc.send:
					if _, err := conn.Write(msg); err != nil {
						return // conn closed or write error; exit loop
					}
				case <-nc.done:
					// Flush any frame(s) already queued (e.g. a DRAIN frame
					// enqueued by the single observer's Range call) before
					// exiting — do NOT drop them silently (F-DW-SP3-005).
					for {
						select {
						case msg := <-nc.send:
							if _, err := conn.Write(msg); err != nil {
								return
							}
						default:
							return
						}
					}
				}
			}
		}()

		if nodeConnHook != nil {
			nodeConnHook(nodeConnRegistered, h.IfaceID)
		}

		return func() {
			sendMap.Delete(h.IfaceID)
			if nodeConnHook != nil {
				nodeConnHook(nodeConnRemoved, h.IfaceID)
			}
			// One of two possible done-close triggers (the other being the
			// router-wide shutdown-flush pass) — doneOnce makes the second
			// trigger a harmless no-op. send is NEVER closed (single-
			// closer/no-send-after-close invariant).
			nc.doneOnce.Do(func() { close(nc.done) })
		}
	}

	var dataWG sync.WaitGroup
	dataWG.Add(1)
	go func() {
		defer dataWG.Done()
		// Serve returns nil on ctx cancel or a wrapped Accept error on
		// terminal listener failure; we log and drop either way — the ctx
		// cancel path is the graceful shutdown.
		if serr := netingress.Serve(ingressCtx, dataLn, route, routerLogger, netingress.ServeConfig{OnAccept: onAccept, IfaceIDSeed: 2}); serr != nil && ingressCtx.Err() == nil {
			routerLogger.Log(fmt.Sprintf("runRouter: netingress.Serve exited: %v", serr))
		}
	}()

	// Resolve the three BC-2.09.003 DEFERRED-APPLICATION values. Each helper
	// applies the zero-value default per PC-7/PC-8/PC-9 semantics.
	drainWindow := drainTimeoutFor(cfg)
	keepaliveInterval := keepaliveIntervalFor(cfg)
	upstreamRouters := upstreamRoutersFor(cfg)

	// Construct the graceful-drain coordinator (BC-2.09.002). No observers
	// are registered in this story — the DRAIN-over-SVTN wire protocol that
	// broadcasts to connected nodes is a follow-on story. The coordinator
	// is nevertheless load-bearing: it holds the drain_timeout at the seam
	// where the follow-on wiring plugs in, and it emits the resolved window
	// at the observability seam below so operators can confirm the config
	// value flowed through.
	drainCoord := drain.New(drainWindow)

	// Single startup drain observer (Q-SINGLE-OBS): registered once here,
	// at drainCoord-construction time — guaranteed to precede
	// drainCoord.Signal (RegisterObserver no-ops after Signal), the only
	// ordering that matters. Captures a reference to the live per-node send
	// map; at Signal time it fires drainObserverFiredHook (test-only, nil
	// in production) as its FIRST statement, then iterates the map and
	// best-effort non-blocking sends a DRAIN frame to every registered node
	// (Q3.P1 — no wire ACK; the send cannot panic, since nc.send is never
	// closed). OnAccept does NOT register a per-connection observer — only
	// this single startup observer is ever registered.
	drainCoord.RegisterObserver(func(_ context.Context) {
		if drainObserverFiredHook != nil {
			drainObserverFiredHook()
		}
		payload := []byte{0x01, 0x01, 0x00, 0x00} // control_type=DRAIN, version=1, reserved
		ehdr := frame.EncodeOuterHeader(frame.OuterHeader{
			Version:    frame.VersionByte,
			FrameType:  frame.FrameTypeCtl,
			PayloadLen: uint16(len(payload)),
		})
		drainFrame := append(append([]byte{}, ehdr[:]...), payload...)
		sendMap.Range(func(_, value any) bool {
			nc := value.(*nodeConn)
			select {
			case nc.send <- drainFrame:
			default: // channel full — best-effort; nc.send is never closed.
			}
			return true
		})
	})

	// Test hook: when non-nil, tests can register a drain observer without
	// waiting on the DRAIN-over-SVTN wire protocol to land. Production code
	// leaves drainCoordHook nil — the hook is a no-op then. Kept behind a
	// package-level var (not exported) so it can only be set from same-package
	// tests. When a follow-on story wires the wire protocol, that story's
	// observers register here and the hook is retired.
	//
	// Evidence-only seam for DRIFT-HS006-DRAIN-TIMEOUT-FORCED-EXIT-UNEVIDENCED
	// (BC-2.09.002 EC-003: forced exit when observers do not ACK within
	// drain_timeout). See router_drain_test.go
	// TestRunRouter_ForcedExitPastDrainTimeout.
	if drainCoordHook != nil {
		drainCoordHook(drainCoord)
	}

	// Construct the FrameArrivalHandler for the PE receive path (AC-002 / Q8).
	// DropCache and FrameArrivalHandler are constructed here, after Phase (b),
	// so routerLogger is available for WithFrameArrivalLogger. The interfaceSet
	// for each PE connection is a single-element set containing peIfaceID; this
	// guarantees split-horizon always exhausts (E-FWD-001 fires deterministically)
	// because the arrival interface is the only forwarding candidate (Q8 ruling).
	// SEC follow-on: the PE receive path bypasses RouteFrame's HMAC admission
	// check — PE upstream connections are established outbound by the connector
	// itself (bootstrap handshake via dialLoop), not arbitrary ingress; admission
	// on PE receive is revisited in the DRAIN-WIRE/session-bootstrap era per Q8.
	dc := multipath.NewDropCache(multipath.DefaultDropCacheSize)
	arrivalHandler := routing.NewFrameArrivalHandler(dc)
	routing.WithFrameArrivalLogger(routerLogger)(arrivalHandler)
	// peIfaceID is the logical interface ID for the PE upstream receive path.
	// Production population of the full interface set is out of scope for this
	// story (F-SP11-003); a single-element set is deliberate here.
	peIfaceID := routing.InterfaceID(1)

	// Construct and start the PE outbound dial loop (BC-2.09.001 PC-2/PC-3,
	// S-7.04-FU-PE-CONNECTOR).  The Connector owns all dial goroutines,
	// per-address backoff timers, and keepalive ticker (Q2, Q3, Q7).
	// It is stopped in the shutdown path below.
	// Envelope carries zero node-identity fields: the full bootstrap (SrcAddr/DstAddr
	// derived from Ed25519 key material) is deferred to the session-bootstrap story.
	// The three-step "connection established" definition (Q6) is satisfied by
	// TCP dial + Assemble (zero env is valid) + Write returning nil.
	connector := upstreamdial.New(w, outerassembler.Envelope{}, keepaliveInterval, upstreamRouters)
	// SetFrameCallback MUST be called before Start() — set-once pre-launch
	// per the ordering contract (F-SP4-002). The FrameFn closure routes received
	// frames through arrivalHandler.OnFrameArrival with the single-interface set
	// that guarantees split-horizon exhaustion (Q8 ruling).
	connector.SetFrameCallback(func(hdr frame.OuterHeader, raw []byte) error {
		return arrivalHandler.OnFrameArrival(raw, peIfaceID, []routing.InterfaceID{peIfaceID}, nil)
	})
	connector.Start()

	// Log resolved listen address + mgmt socket path.
	// The writer is os.Stderr in production (main.go); the tutorial doc says
	// "stdout" — this is a known documentation drift called out in the PR body.
	if w != nil {
		_, _ = fmt.Fprintf(w, "switchboard router: data plane listening on %s\n", dataLn.Addr().String())
		_, _ = fmt.Fprintf(w, "switchboard router: management socket at %s\n", resolveManagementSocket(cfg, "router"))
		// AC-008 / BC-2.09.003 v2.1 PC-14 / Ruling 9: emit INFO log for the
		// router management listener bind address so operators can verify the
		// address and apply appropriate firewall policy. No loopback restriction
		// (Ruling 9 of S-BL.ADMISSION-SYNC-WIRE-rulings.md v1.2).
		if len(cfg.RouterManagementEndpoints) > 0 {
			for _, ep := range cfg.RouterManagementEndpoints {
				_, _ = fmt.Fprintf(w, "switchboard router: router management listener bound to %s (ensure firewall policy restricts access as appropriate)\n", ep.Addr)
			}
		} else {
			_, _ = fmt.Fprintf(w, "switchboard router: router management listener bound to %s (ensure firewall policy restricts access as appropriate)\n", resolveManagementSocket(cfg, "router"))
		}
		// BC-2.09.003 PC-7 application: drain_timeout emitted at startup so
		// operators can confirm the resolved value (config or default).
		_, _ = fmt.Fprintf(w, "switchboard router: drain_timeout=%s\n", drainCoord.Timeout())
		// BC-2.09.003 PC-8 application: keepalive_interval emitted at startup.
		// The reconnect-side keepalive ticker ships in this story inside
		// upstreamdial.Connector; the resolved value is captured here so the
		// config-to-application flow is auditable.
		_, _ = fmt.Fprintf(w, "switchboard router: keepalive_interval=%s\n", keepaliveInterval)
		// BC-2.09.003 PC-9 / BC-2.09.001 PC-1 application: upstream_routers
		// emitted at startup. Empty list = E mode; non-empty list = PE-mode
		// graduation eligibility.
		if len(upstreamRouters) == 0 {
			_, _ = fmt.Fprintf(w, "switchboard router: mode=E (no upstream_routers configured)\n")
		} else {
			_, _ = fmt.Fprintf(w, "switchboard router: mode=PE upstream_routers=%v\n", upstreamRouters)
		}
	}

	// Block until context is cancelled, a SIGHUP reload event fires, or an
	// RPC-triggered drain request arrives.
	// (ARCH-01 lifecycle contract; S-7.04-FU-SIGHUP-RELOAD two-case select,
	// widened to three cases by S-BL.CLI-SURFACE-COMPLETION Decision 4.)
	for {
		select {
		case <-ctx.Done():
			// Context cancelled — fall through to graceful shutdown below.
			goto shutdown
		case <-drainRequestCh:
			// RPC-triggered drain (router.drain, bridged via routerDrainRPCHandler)
			// reaches the same shutdown sequence as ctx.Done()/SIGTERM — same
			// drain-broadcast, per-node-flush, exit sequence (Decision 4 / AC-012
			// PC-2, AC-013).
			goto shutdown
		case <-sighupCh:
			// Fail-closed reload (BC-2.09.003 EC-004 / BC-2.09.001 PC-1).
			// Empty configPath means no config file was provided; skip silently.
			if configPath == "" {
				continue
			}
			loaded, loadErr := config.LoadFile(configPath)
			if loadErr != nil {
				if w != nil {
					_, _ = fmt.Fprintf(w, "config reload failed: %s; continuing with previous config\n", loadErr)
				}
				continue
			}
			if valErr := loaded.Validate(); valErr != nil {
				if w != nil {
					_, _ = fmt.Fprintf(w, "config reload failed: %s; continuing with previous config\n", valErr)
				}
				continue
			}
			newUpstreams := upstreamRoutersFor(loaded)
			// Pass updated address list to connector using set-equal semantics (Q1, AC-001).
			// The emission diff stays order-sensitive (equalStringSlices unchanged per Q1 ruling);
			// the connector reconciliation is set-equal internally (Q1 lawful difference).
			connector.ReloadAddrs(newUpstreams)
			if !equalStringSlices(upstreamRouters, newUpstreams) {
				upstreamRouters = newUpstreams
				if w != nil {
					if len(upstreamRouters) > 0 {
						_, _ = fmt.Fprintf(w, "switchboard router: mode=PE upstream_routers=%v\n", upstreamRouters)
					} else {
						_, _ = fmt.Fprintf(w, "switchboard router: mode=E (no upstream_routers configured)\n")
					}
				}
			}
		}
	}
shutdown:

	// Graceful shutdown (BC-2.09.002):
	//   1. Signal drain — the single startup observer broadcasts DRAIN to
	//      every connected node's per-node send channel and returns after
	//      queuing (Q3.P1 best-effort — no wire ACK).
	//   2. Wait, bounded by drain_timeout. On timeout we proceed with
	//      disconnect anyway (BC-2.09.002 EC-003).
	//   3. Shutdown ordering guarantee (Q-SEAM): a bounded, snapshot-scoped
	//      flush phase forces every still-live writer goroutine to drain
	//      its queued frame(s) to the wire BEFORE connections are torn
	//      down — otherwise Wait returning nil only proves queuing, not
	//      writing, and a node could read EOF instead of DRAIN.
	//   4. Cancel ingress ctx so netingress.Serve closes the listener and
	//      joins its per-conn goroutines.
	//   5. Shut down mgmt with a budget derived from the same drain window
	//      (previously hardcoded 5s — now driven by cfg.DrainTimeout so
	//      operators have a single lever for shutdown budget tuning).
	// Stop the PE connector first so dial goroutines exit before we
	// shut down the ingress listener (AC-001 Q2 lifecycle contract).
	connector.Stop()

	drainCtx, drainCtxCancel := context.WithTimeout(context.Background(), drainCoord.Timeout())
	drainCoord.Signal(drainCtx)
	if derr := drainCoord.Wait(drainCtx); derr != nil {
		routerLogger.Log(fmt.Sprintf("runRouter: drain: %v (proceeding with disconnect per BC-2.09.002 EC-003)", derr))
	}
	drainCtxCancel()

	// Router-wide flush pass (Shutdown ordering guarantee, Q-SEAM): tell
	// every still-live writer goroutine to drain its queued frame(s) and
	// exit, BEFORE the connections are torn down. In the SAME Range call,
	// snapshot the live nodeConn set — the bounded wait below joins
	// EXACTLY this snapshot, never the shared writerWG (a late admission
	// concurrently calling writerWG.Add(1) must never race a writerWG
	// Wait — see the Shutdown concurrency ledger cited in Design
	// Constraints, F-DW-SP6-001).
	var snapshot []*nodeConn
	sendMap.Range(func(_, value any) bool {
		nc := value.(*nodeConn)
		nc.doneOnce.Do(func() { close(nc.done) })
		snapshot = append(snapshot, nc)
		return true
	})

	// Bounded wait for ONLY the snapshotted writers to flush and exit.
	// snapshotWG is LOCAL to this phase — no other goroutine, including
	// any OnAccept invocation, ever touches it, so Add-concurrent-with-
	// Wait cannot occur on it structurally. Every Add below runs
	// synchronously before any of the per-entry goroutines — or the
	// Wait-calling goroutine — is spawned.
	var snapshotWG sync.WaitGroup
	snapshotWG.Add(len(snapshot))
	for _, nc := range snapshot {
		nc := nc
		go func() {
			defer snapshotWG.Done()
			<-nc.writerExited
		}()
	}
	// closerWG (F-DW-SP7-001) tracks the flushDone-closer goroutine itself,
	// joined explicitly below regardless of which select branch fires.
	var closerWG sync.WaitGroup
	closerWG.Add(1)
	flushDone := make(chan struct{})
	go func() {
		defer closerWG.Done()
		snapshotWG.Wait()
		close(flushDone)
	}()
	select {
	case <-flushDone:
	case <-time.After(drainFlushTimeout):
		routerLogger.Log("runRouter: drain flush deadline exceeded; proceeding with shutdown")
	}

	ingressCancel() // SIGNALS shutdown only — netingress.Serve's own watcher
	// goroutine closes the listener ASYNCHRONOUSLY off this same
	// cancellation; a connection already racing ln.Accept() can still be
	// admitted after this line returns. A late admission here is
	// unconditionally BENIGN — nothing is parked on writerWG or
	// snapshotWG at this point in the sequence.

	// dataWG joins the netingress.Serve goroutine. Serve's own doc contract
	// ("All outstanding per-connection goroutines are joined before Serve
	// returns") plus its internal per-conn-goroutine join together
	// guarantee that by the time this call returns: (a) the accept loop
	// has permanently exited — no further OnAccept call and therefore no
	// further writerWG.Add(1) is possible (Goroutine pin); AND (b) every
	// per-conn goroutine's deferred behavior-cleanup has already run — so
	// every nc.done this router ever created is closed by the time this
	// line returns (doneOnce makes a second close from the flush pass
	// above, if it already fired, a harmless no-op).
	dataWG.Wait()

	// Final UNBOUNDED join — the SOLE writerWG.Wait() call in the entire
	// sequence. Safe on the grounds that no writerWG.Add(1) can occur
	// after this point (grounded on dataWG.Wait having joined Serve's
	// accept loop) and every writer still blocked in its send/done select
	// has already observed nc.done closed (grounded on dataWG.Wait having
	// joined every per-conn cleanup). Restores the ARCH-01 goroutine-join
	// contract for writerWG in full — the ONLY writerWG.Wait() anywhere in
	// the sequence, at the ONLY point where no concurrent Add is possible.
	writerWG.Wait()

	// Join the bounded phase's own goroutines (F-DW-SP7-001) before this
	// shutdown block returns — closes an ARCH-01 goroutine-join-contract
	// gap on the drainFlushTimeout-exceeded path. Both calls are PROVEN
	// PROMPT: within each writer goroutine, close(nc.writerExited) is
	// deferred AFTER writerWG.Done() in source order, so Go's LIFO defer
	// order runs the writerExited close BEFORE writerWG.Done() — meaning
	// every writer's writerExited has ALREADY closed by the time the
	// writerWG.Wait() above returns.
	snapshotWG.Wait()
	closerWG.Wait()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), drainCoord.Timeout())
	defer shutCancel()
	_ = mgmtSrv.Shutdown(shutCtx)
	mgmtWG.Wait()
	return nil
}

// runConsole is the console-mode daemon entry point (S-7.03).
//
// It generates an ephemeral Ed25519 keypair, constructs a session.Publisher and
// session.ConsoleState, starts the management server with BuildConsoleHandlers
// registered, and blocks until ctx is cancelled (ARCH-01 §Goroutine WaitGroup
// Contract).
//
// AC-004 (S-6.06): console mode does NOT register admin.key.* handlers —
// console daemons must NOT register admin handlers (ADR-004 role-exclusion;
// ARCH-04 disambiguation table).
//
// Register-before-serve ordering (F-P2L1-001 / F-P2L1-002):
//  1. newMgmtServer — construct server (no goroutine)
//  2. BuildConsoleHandlers passed via NewServer initial handlers (already registered)
//  3. wireMetricsHandlers — register metrics RPC handlers before Serve
//  4. serveMgmtServer — start Serve goroutine
func runConsole(ctx context.Context, _ io.Writer, cfg *config.Config) error {
	// Generate ephemeral daemon keypair (BC-2.07.004 Precondition 3 / AC-015).
	_, daemonPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("runConsole: generate daemon keypair: %w", err)
	}

	// Build console session infrastructure.
	ks := admission.NewAdmittedKeySet()

	// Register configured operator keys into ks with RoleConsole so that
	// verifyConsoleCallerRole (in BuildConsoleHandlers) admits them for console RPCs
	// (BC-2.08.001 Inv-1; L1-C4; F-P2L1-001).
	//
	// Two-layer authorization for console RPCs:
	//   Layer 1 (Tier-1, mgmt-plane): mgmt.Server authenticates the caller via
	//           Ed25519 challenge-response against the OperatorKeySet. Only keys in
	//           OperatorKeySet reach the handler (BC-2.07.004).
	//   Layer 2 (Tier-2, session-plane): verifyConsoleCallerRole checks the caller's
	//           key against ks (AdmittedKeySet). Keys absent from ks receive E-ADM-006
	//           even if they passed Layer 1.
	//
	// Both layers use cfg.AuthorizedOperatorKeys as the source of trusted keys.
	// The zero svtnID ([16]byte{}) is the console-daemon's global partition — console
	// keys are not SVTN-scoped (ARCH-04 §Console Key Scope; ADR-006).
	var zeroSVTN [16]byte
	for _, pub := range parsePEMOperatorKeys(func() []string {
		if cfg != nil {
			return cfg.AuthorizedOperatorKeys
		}
		return nil
	}()) {
		ks.RegisterKey(zeroSVTN, pub, admission.RoleConsole)
	}

	pub := session.NewPublisher(ks)

	// Construct the sessionQualitySource and install the SessionHook callbacks
	// on the Publisher. From this point every pub.Publish / pub.Unpublish
	// drives the source's per-session QualityIndicator registry — the
	// boundary construct that owns the internal/metrics dependency so that
	// internal/session does not (ARCH-08 §6.6 topological order:
	// internal/session at DAG 6, internal/metrics at DAG 12). Mirrors
	// newPathTrackerSourceFromRouter (S-BL.PATH-TRACKER-WIRING).
	src := newSessionQualitySourceFromPublisher(pub)

	consoleState := session.NewConsoleState()
	consoleSrv := session.NewConsoleServer(pub, consoleState)

	// Phase (a): construct server with console + sessions handlers pre-registered
	// (no goroutine). newMgmtServer also parses cfg.AuthorizedOperatorKeys for the
	// OperatorKeySet (Layer 1); ks above handles Layer 2 (Tier-2 admission).
	//
	// BuildSessionsHandlers registers sessions.status alongside console.attach /
	// detach / switch — the same daemon, the same Tier-2 admission surface.
	// Together they satisfy BC-2.06.001 v1.7 PC-5 console-half (quality) and
	// BC-2.06.002 v1.4 PC-3 (miss_count) via the mgmt-plane (DRIFT-001b +
	// DRIFT-002 closures; S-BL.CONSOLE-OBS). The handler reads through the
	// sessionQualitySource, not through Publisher internals.
	initialHandlers := append(
		BuildConsoleHandlers(consoleSrv, ks),
		BuildSessionsHandlers(src, ks)...,
	)
	mgmtSrv, mgmtErr := newMgmtServer(cfg, "console", daemonPriv, initialHandlers)
	if mgmtErr != nil {
		return fmt.Errorf("runConsole: construct management server: %w", mgmtErr)
	}

	// Phase (b): register metrics handlers before Serve starts (F-P2L1-001, F-P2L1-002).
	// Console mode has no routing subsystem — pass nil router; the paths.list
	// source is an empty registry, and BC-2.06.003 EC-001 "no active paths"
	// applies (S-BL.PATH-TRACKER-WIRING).
	if err := wireMetricsHandlers(mgmtSrv, nil); err != nil {
		return fmt.Errorf("runConsole: wire metrics handlers: %w", err)
	}

	// Phase (c): start the Serve goroutine.
	var mgmtWG sync.WaitGroup
	serveMgmtServer(ctx, &mgmtWG, mgmtSrv)

	// Block until context is cancelled (ARCH-01 lifecycle contract).
	<-ctx.Done()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	_ = mgmtSrv.Shutdown(shutCtx)
	mgmtWG.Wait()
	return nil
}

// runControl is the control-mode daemon entry point (F-002 / S-6.06 AC-004).
// It generates an ephemeral Ed25519 keypair, constructs a SVTNManager with an
// empty AdmittedKeySet, starts the management server with BuildAdminHandlers,
// and blocks until ctx is cancelled (ARCH-01 §Goroutine WaitGroup Contract).
//
// Only the control-mode daemon registers admin handlers (ADR-004 role-exclusion;
// ARCH-04 disambiguation table; AC-004). Access, console, and router daemons pass nil.
//
// Register-before-serve ordering (F-P2L1-001 / F-P2L1-002):
//  1. newMgmtServer — construct server (no goroutine)
//  2. BuildAdminHandlers passed via NewServer initial handlers (already registered)
//  3. wireMetricsHandlers — register metrics RPC handlers before Serve
//  4. serveMgmtServer — start Serve goroutine
//  5. PushFullSnapshot — push all keyset entries to configured routers before serving
//     (S-BL.ADMISSION-SYNC-WIRE AC-009 / BC-2.05.009 Postcondition 7 / Decision 10)
func runControl(ctx context.Context, _ io.Writer, cfg *config.Config) error {
	// Generate ephemeral daemon keypair (BC-2.07.004 Precondition 3 / AC-015).
	// The daemon key is used by mgmt.Server for the Ed25519 challenge-response.
	_, daemonPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("runControl: generate daemon keypair: %w", err)
	}

	daemonPub := daemonPriv.Public().(ed25519.PublicKey)
	ks := admission.NewAdmittedKeySet()
	m := svtnmgmt.NewSVTNManager(ks, daemonPub)

	// Bootstrap mode: nil OperatorKeySet (no pre-configured operator keys).
	// The daemon's own key is the sole bootstrap authority (SVTNManager.IsBootstrapKey).
	ops := mgmt.NewOperatorKeySet(nil)

	// Construct the admission sync client with the configured router management
	// endpoints (S-BL.ADMISSION-SYNC-WIRE AC-009/010 / BC-2.05.009 Ruling 2).
	// An empty endpoint list → push methods are no-ops (single-router deployment).
	var endpoints []config.RouterManagementEndpoint
	if cfg != nil {
		endpoints = cfg.RouterManagementEndpoints
	}
	syncClient := newAdmissionSyncClient(endpoints, daemonPriv)

	// Phase (a): construct server with admin handlers pre-registered (no goroutine).
	// Pass the real syncClient so push calls are wired into make*Handler
	// (S-BL.ADMISSION-SYNC-WIRE AC-003/004).
	mgmtSrv, mgmtErr := newMgmtServer(cfg, "control", daemonPriv, BuildAdminHandlers(m, ops, syncClient))
	if mgmtErr != nil {
		return fmt.Errorf("runControl: construct management server: %w", mgmtErr)
	}

	// Phase (b): register metrics handlers before Serve starts (F-P2L1-001, F-P2L1-002).
	// Control mode has no routing subsystem — pass nil router; the paths.list
	// source is an empty registry, and BC-2.06.003 EC-001 "no active paths"
	// applies (S-BL.PATH-TRACKER-WIRING).
	if err := wireMetricsHandlers(mgmtSrv, nil); err != nil {
		return fmt.Errorf("runControl: wire metrics handlers: %w", err)
	}

	// Phase (c): start the Serve goroutine.
	var mgmtWG sync.WaitGroup
	serveMgmtServer(ctx, &mgmtWG, mgmtSrv)

	// Phase (d): push full snapshot to all configured routers BEFORE serving
	// (BC-2.05.009 Postcondition 7 / AC-009 / Decision 10). Push error is
	// advisory — log and continue.
	if pushErr := syncClient.PushFullSnapshot(ctx, ks); pushErr != nil {
		// Advisory: WARN only; do not fail startup (BC-2.05.009 PC-2).
		_ = pushErr
	}

	// Block until context is cancelled (ARCH-01 lifecycle contract).
	<-ctx.Done()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	_ = mgmtSrv.Shutdown(shutCtx)
	mgmtWG.Wait()
	return nil
}
