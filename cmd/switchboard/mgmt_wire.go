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
	"github.com/arcavenae/switchboard/internal/netingress"
	"github.com/arcavenae/switchboard/internal/routing"
	"github.com/arcavenae/switchboard/internal/session"
	"github.com/arcavenae/switchboard/internal/svtnmgmt"
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
//     at the observability seam; the reconnect-side keepalive ticker itself
//     ships when node-connection plumbing lands (BC-2.09.003 PC-8; FM-009).
//     MUST NOT be routed into sweepDeadline (console eviction — different
//     semantic; BC-2.09.003 PC-8 normative note).
//   - upstream_routers   → upstreamRoutersFor(cfg) resolves and is emitted
//     at the observability seam; a non-empty list signals PE-mode graduation
//     eligibility (BC-2.09.001 PC-1). Live upstream connection establishment
//     ships once the outer-header session-bootstrap protocol lands.
//
// #DEFERRED — SIGHUP config reload (BC-2.09.001 PC-1 Signal-of-graduation),
// live DRAIN-over-SVTN wire protocol (BC-2.09.002 Inv-1), and the actual
// PE-mode upstream connector still ship in a follow-on story once admission
// plumbing and node-facing SVTN channels are wired. In this story the drain
// coordinator seam and the three DEFERRED-APPLICATION closures are in place;
// the wire protocol connects to them without further daemon-level refactor.
func runRouter(ctx context.Context, w io.Writer, cfg *config.Config) error {
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
	router := buildRouter(admission.NewAdmittedKeySet(), routerLogger)

	// Phase (c): register metrics handlers before Serve starts. Passing the
	// live router installs a forwarding-entry hook that populates the paths.list
	// source on RegisterForwardingEntry (S-BL.PATH-TRACKER-WIRING).
	if err := wireMetricsHandlers(mgmtSrv, router); err != nil {
		return fmt.Errorf("runRouter: wire metrics handlers: %w", err)
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
	ingressCtx, ingressCancel := context.WithCancel(ctx)
	defer ingressCancel()
	route := func(hdr frame.OuterHeader, payload []byte) error {
		return routing.RouteFrame(hdr, payload, router)
	}
	var dataWG sync.WaitGroup
	dataWG.Add(1)
	go func() {
		defer dataWG.Done()
		// Serve returns nil on ctx cancel or a wrapped Accept error on
		// terminal listener failure; we log and drop either way — the ctx
		// cancel path is the graceful shutdown.
		if serr := netingress.Serve(ingressCtx, dataLn, route, routerLogger); serr != nil && ingressCtx.Err() == nil {
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

	// Log resolved listen address + mgmt socket path.
	// The writer is os.Stderr in production (main.go); the tutorial doc says
	// "stdout" — this is a known documentation drift called out in the PR body.
	if w != nil {
		_, _ = fmt.Fprintf(w, "switchboard router: data plane listening on %s\n", dataLn.Addr().String())
		_, _ = fmt.Fprintf(w, "switchboard router: management socket at %s\n", resolveManagementSocket(cfg, "router"))
		// BC-2.09.003 PC-7 application: drain_timeout emitted at startup so
		// operators can confirm the resolved value (config or default).
		_, _ = fmt.Fprintf(w, "switchboard router: drain_timeout=%s\n", drainCoord.Timeout())
		// BC-2.09.003 PC-8 application: keepalive_interval emitted at startup.
		// The reconnect-side keepalive ticker itself ships with the node
		// protocol; the resolved value is captured here so the config-to-
		// application flow is auditable.
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

	// Block until context is cancelled (ARCH-01 lifecycle contract).
	<-ctx.Done()

	// Graceful shutdown (BC-2.09.002):
	//   1. Signal drain — observers (when the wire protocol lands) broadcast
	//      DRAIN to their connected nodes and wait for ACKs.
	//   2. Wait, bounded by drain_timeout. On timeout we proceed with
	//      disconnect anyway (BC-2.09.002 EC-003).
	//   3. Cancel ingress ctx so netingress.Serve closes the listener and
	//      joins its per-conn goroutines.
	//   4. Shut down mgmt with a budget derived from the same drain window
	//      (previously hardcoded 5s — now driven by cfg.DrainTimeout so
	//      operators have a single lever for shutdown budget tuning).
	drainCtx, drainCtxCancel := context.WithTimeout(context.Background(), drainCoord.Timeout())
	drainCoord.Signal(drainCtx)
	if derr := drainCoord.Wait(drainCtx); derr != nil {
		routerLogger.Log(fmt.Sprintf("runRouter: drain: %v (proceeding with disconnect per BC-2.09.002 EC-003)", derr))
	}
	drainCtxCancel()

	ingressCancel()
	dataWG.Wait()

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
	consoleState := session.NewConsoleState()
	consoleSrv := session.NewConsoleServer(pub, consoleState)

	// Phase (a): construct server with console handlers pre-registered (no goroutine).
	// newMgmtServer also parses cfg.AuthorizedOperatorKeys for the OperatorKeySet
	// (Layer 1); ks above handles Layer 2 (Tier-2 admission).
	mgmtSrv, mgmtErr := newMgmtServer(cfg, "console", daemonPriv, BuildConsoleHandlers(consoleSrv, ks))
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

	// Phase (a): construct server with admin handlers pre-registered (no goroutine).
	mgmtSrv, mgmtErr := newMgmtServer(cfg, "control", daemonPriv, BuildAdminHandlers(m, ops))
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

	// Block until context is cancelled (ARCH-01 lifecycle contract).
	<-ctx.Done()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	_ = mgmtSrv.Shutdown(shutCtx)
	mgmtWG.Wait()
	return nil
}
