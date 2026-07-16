// Package main — access.go contains the access-mode subcommand handler that
// wires all Wave-3 subsystems for the access node (S-W3.04; ARCH-08 §6.5.1).
//
// Six wiring obligations (ARCH-08 §6.5.1):
//  1. Inject real routing.Logger into NewRouter via WithLogger (AC-001).
//  2. Construct admission.AdmittedKeySet, session.Publisher, session.SessionAuth;
//     wire NewAccessNode(pub, auth, WithKeystrokeSink(sc)) (AC-002).
//  3. Sweep ticker → accessNode.Sweep(deadline) (AC-003).
//  4. sc.Frames() → accessNode.DeliverFrame bridge goroutine (AC-004/005).
//  5. Live *SessionAuth as Authorizer — fail-open default closed (AC-002).
//  6. FramesDropped structured log ticker (AC-006).
//
// Daemon lifecycle (BC-2.04.007):
//   - PC-1: sc.Connect failure → log + exit non-zero (AC-007).
//   - PC-2: SIGTERM/SIGINT → context cancel → all goroutines drain → exit 0
//     (AC-008).
//   - PC-2.6: sc.Err() non-nil after Connect → E-SYS-002 log → cancel → exit 1
//     (AC-007; BC-2.04.007 v1.1 PC-2.6/EC-007/Inv-5).
//
// Dependency boundary (ARCH-08 §6.5.2; ARCH-01 ADR-011 v1.5 §EC-005):
//   - internal/config:  PERMITTED — wired as of S-6.01/Wave 4 (tickIntervalFor,
//     runAccess config parameter). Import is present and intentional.
//   - internal/drain:   NOT imported — still deferred; do not add without a story.
//   - internal/metrics: wired via metrics_wire.go (S-W5.04 AC-001, AC-004, AC-005;
//     F-P1L1-002). Stub sources used until real PathTracker map is available.
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/config"
	"github.com/arcavenae/switchboard/internal/discovery"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/halfchannel"
	"github.com/arcavenae/switchboard/internal/routing"
	"github.com/arcavenae/switchboard/internal/session"
	"github.com/arcavenae/switchboard/internal/tmux"
)

// defaultTickInterval is the half-channel tick cadence used when no config file
// is supplied (Wave-3 hardcoded default, BC-2.09.003 PC-9 / Inv-5).
const defaultTickInterval = 10 * time.Millisecond

// defaultAdmissionKeyFile is the path used when admission_key_file is absent
// or empty in the config (Decision 2 / BC-2.09.004 PC-1; S-BL.NODE-ADMISSION-PROVISIONING).
// Applied at daemon startup by runAccess, NOT by Config.Validate().
// Declared as var (not const) so tests can redirect it to a tempdir via
// package-level assignment (mirrors framesDroppedInterval / newHalfChannel pattern;
// F-3 hermeticity fix — testability seam for S-BL.NODE-ADMISSION-PROVISIONING).
var defaultAdmissionKeyFile = "/var/lib/switchboard/access-admission-identity.pem"

// sweepInterval is the period between consecutive Sweep eviction passes.
// Hardcoded for Wave 3 (no file-based config loading; internal/config is Wave 4).
const sweepInterval = 30 * time.Second

// sweepDeadline is the keepalive inactivity window after which a console is
// evicted by Sweep. Hardcoded for Wave 3.
const sweepDeadline = 60 * time.Second

// hmacFailureThreshold and hmacFailureWindow are the FailureCounter constants
// mandated by BC-2.05.005 PC-3: emit E-ADM-017 after ≥5 failures from the
// same source within a 60-second sliding window.
const (
	hmacFailureThreshold = 5
	hmacFailureWindow    = 60 * time.Second
)

// framesDroppedInterval is the period between FramesDropped log checks.
// Hardcoded for Wave 3 (AC-006; BC-2.04.006 invariant 4).
// Declared as var (not const) so tests can inject a short interval via
// package-level assignment (framesDroppedInterval = 1ms before the call).
var framesDroppedInterval = 30 * time.Second

// newHalfChannel is the half-channel constructor seam.
// Declared as var (not called directly) so tests can inject a capturing stub
// to verify that tickIntervalFor(cfg) reaches halfchannel.New end-to-end
// (BC-2.09.003 PC-9 / Inv-5 / AC-009; mirrors the framesDroppedInterval pattern).
var newHalfChannel = halfchannel.New

// connectorIface is the minimal subset of *tmux.SessionConnector used by
// runAccessWithConnector. *tmux.SessionConnector satisfies this interface by
// construction — no changes to internal/tmux are required for the interface
// itself. The seam enables tests to inject a fakeConnector for PC-2 and PC-2.6
// end-to-end coverage (ARCH-01 ADR-011 v1.5 §HIGH-B; ARCH-08 v2.1 §6.5.1
// obligation 4).
type connectorIface interface {
	Connect(ctx context.Context) error
	Frames() <-chan halfchannel.ChannelFrame
	Err() <-chan error
	Close() error
	RelayDropped() uint64
}

// tickIntervalFor returns the half-channel tick interval to use.
//
// When cfg is non-nil and cfg.TickInterval > 0, cfg.TickInterval is the single
// source of truth (BC-2.09.003 PC-9 / Inv-5 / AC-009). When cfg is nil (no
// --config supplied) or cfg.TickInterval is zero, defaultTickInterval (10ms) is
// returned.
//
// Note: drain_timeout, upstream_routers, and keepalive_interval application
// is explicitly deferred to S-7.04 (Wave 7). listen_addr is applied by
// S-BL.NI at the netingress listener bind (see runRouter in mgmt_wire.go —
// BC-2.09.003 PC-9). Those remaining fields are validated at startup
// (AC-005 through AC-008) but NOT applied here.
func tickIntervalFor(cfg *config.Config) time.Duration {
	if cfg != nil && cfg.TickInterval > 0 {
		return cfg.TickInterval
	}
	return defaultTickInterval
}

// runAccess is the thin constructor wrapper for the access-mode handler. It
// builds the real *tmux.SessionConnector (with defaultPTYAlloc), constructs
// access components via buildAccessComponents, and delegates to
// runAccessWithConnector (ARCH-01 ADR-011 v1.5 §HIGH-B).
//
// main.go calls runAccess(ctx, os.Stderr, cfg); a non-nil return triggers
// os.Exit(1). cfg is the validated config (nil when --config is not supplied).
// When cfg is non-nil, the half-channel tick interval is sourced from
// cfg.TickInterval (BC-2.09.003 PC-9 / Inv-5 / AC-009).
//
// S-W5.01: the management server is started before the data-plane connector
// per ARCH-12 §Daemon Mode Startup. The mgmt goroutine is WaitGroup-tracked
// per ARCH-01 §Goroutine WaitGroup Contract.
func runAccess(ctx context.Context, stderr io.Writer, cfg *config.Config) error {
	// Generate an ephemeral Ed25519 keypair for the daemon management identity
	// (BC-2.07.004 Precondition 3 / AC-015 / Ruling A.1). The key is ephemeral —
	// identity changes across restarts. Persistent key_file wiring is deferred to
	// S-6.02. The bootstrap OperatorKeySet (nil ops) means the daemon's own
	// ephemeral key is the sole authorized key until operator keys are configured.
	_, daemonPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		// Extremely unlikely (DRNG failure). Abort startup — a daemon without
		// a management identity cannot be securely administered.
		return fmt.Errorf("access: generate daemon keypair: %w", err)
	}

	// Construct the management server before opening data-plane connections
	// (ARCH-12 §Daemon Mode Startup — the access daemon starts its own
	// mgmt.Server before any data-plane I/O such as sc.Connect).
	//
	// Register-before-serve ordering (F-P2L1-001 data-race fix):
	//   Phase (a): newMgmtServer — construct server (no goroutine started).
	//   Phase (b): wireMetricsHandlers — register RPC handlers before Serve.
	//   Phase (c): serveMgmtServer — start the Serve goroutine.
	//
	// AC-004 (S-6.06): access-mode daemon passes nil admin handlers.
	// Only the control-mode daemon registers admin handlers via BuildAdminHandlers
	// (ADR-004 role-exclusion; ARCH-04 disambiguation table). Access daemons correctly return
	// E-RPC-010 for any admin.key.* command.
	mgmtSrv, mgmtErr := newMgmtServer(cfg, "access", daemonPriv, nil)
	if mgmtErr != nil {
		// Ruling J / BC-2.07.004 v1.4 EC-013: ANY mgmt-start failure aborts startup.
		// No degraded-management mode — a daemon that cannot be administered must not
		// serve the data plane (SOUL #4 silent-failure).
		return fmt.Errorf("access: construct management server: %w", mgmtErr)
	}

	// Construct the downstream half-channel (pure in-memory struct, no goroutines).
	// Moved above Phase (b) so the Router is available to wireMetricsHandlers as
	// the source-of-PathTracker registrations (S-BL.PATH-TRACKER-WIRING).
	// The tickIntervalFor seam (AC-009) still fires here.
	ds := newHalfChannel(1, halfchannel.Downstream, tickIntervalFor(cfg))

	// keys and pub are constructed once and shared with BOTH the AccessNode
	// (via Publisher) AND the Router, so AC-001's test can register a key and
	// observe E-ADM-016 emission on the daemon's own router instance without a
	// separate reconstruction (ARCH-08 v2.0 §6.5.1 obligation 1 non-tautology
	// requirement).
	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)

	ctrl := tmux.New(pub, ds)
	pty := tmux.NewPTYProxy(pub, ds)
	sc := tmux.NewSessionConnector(ctrl, pty)

	// Production router logger uses the injected stderr writer (FIX 2 — AC-001
	// injectable logger; FIX 1 — respects stderr redirection, not os.Stderr).
	routerLogger := stdLogger{log.New(stderr, "", 0)}

	// buildAccessComponents wires the full set of access-node components using
	// the shared keys and sc, and accepts the injectable routerLogger. The router
	// is also returned so tests can call RouteFrame on the daemon's own instance
	// (non-tautological — shared keyset). buildAccessComponents signature is
	// UNCHANGED per ARCH-01 ADR-011 v1.5 §HIGH-B.
	an, router := buildAccessComponents(keys, pub, sc, routerLogger)

	// Phase (b): register metrics RPC handlers BEFORE starting the Serve goroutine
	// (S-W5.04 AC-001, AC-004, AC-005; BC-2.06.003 PC-1/PC-2/PC-3; F-P2L1-001).
	// Pass the live router so wireMetricsHandlers installs a forwarding-entry hook
	// that populates the paths.list source on RegisterForwardingEntry
	// (S-BL.PATH-TRACKER-WIRING).
	if err := wireMetricsHandlers(mgmtSrv, router); err != nil {
		return fmt.Errorf("access: wire metrics handlers: %w", err)
	}

	// Phase (c): start the Serve goroutine.
	var mgmtWG sync.WaitGroup
	serveMgmtServer(ctx, &mgmtWG, mgmtSrv)

	// F-2 fix: install a deferred shutdown for mgmtSrv + mgmtWG so that ANY early
	// return after Phase (c) — including keypair-load failures (EC-006/EC-007) —
	// shuts down the mgmt goroutine and waits for it to exit. The explicit shutdown
	// block below (after runAccessWithConnector) also runs, but it is protected by
	// a once-Do guard so the two paths do not double-Shutdown or double-Wait.
	// AC-004 PC-3 ("no partial startup"): a keypair failure must not leave a running
	// mgmt goroutine (BC-2.09.004 S-BL.NODE-ADMISSION-PROVISIONING adversary F-2).
	var mgmtShutOnce sync.Once
	shutdownMgmt := func() {
		mgmtShutOnce.Do(func() {
			if mgmtSrv != nil {
				shutCtx, shutCancel := context.WithTimeout(context.Background(), 2*time.Second)
				_ = mgmtSrv.Shutdown(shutCtx)
				shutCancel()
			}
			mgmtWG.Wait()
		})
	}
	defer shutdownMgmt()

	// Phase (d): load or generate the admission keypair
	// (S-BL.NODE-ADMISSION-PROVISIONING, BC-2.09.004 PC-3 through PC-7).
	//
	// If the context is already cancelled, skip file I/O and proceed directly to
	// runAccessWithConnector, which will detect ctx.Err() != nil and emit E-SYS-002.
	// This preserves the pre-existing TestDaemonConnectFailureExitsNonZero behaviour
	// (pre-cancelled ctx → E-SYS-002, not a file-permission error).
	//
	// Effective key file path: cfg.AdmissionKeyFile when non-empty, else default.
	admissionKeyPath := defaultAdmissionKeyFile
	if cfg != nil && cfg.AdmissionKeyFile != "" {
		admissionKeyPath = cfg.AdmissionKeyFile
	}

	var disc *discovery.Discovery
	if ctx.Err() == nil {
		// Normal startup: load or generate the admission keypair.
		admissionPrivKey, loadErr := loadOrGenerateAdmissionKeypair(stderr, admissionKeyPath)
		if loadErr != nil {
			// F-4 fix: loadOrGenerateAdmissionKeypair already namespaces its errors
			// ("access: load admission keypair: <path>: <reason>", etc.) per E-KEY-001.
			// Re-wrapping with the same prefix produces a doubled/misleading message at
			// the daemon surface. Return loadErr directly.
			return loadErr
		}
		admissionPubKey := admissionPrivKey.Public().(ed25519.PublicKey)

		// Phase (e): construct discovery with LocalNodeAdmissionPubkey wired from
		// the admission keypair (Decision 5 / BC-2.09.004 PC-3e / BC-2.04.008 Pre-3).
		// disc.Run is started in runAccessWithConnector (Option Y, Decision 6).
		discoveryCfg := discovery.Config{
			LocalNodeAdmissionPubkey: []byte(admissionPubKey),
		}
		disc = discovery.New(discoveryCfg)
	}

	runErr := runAccessWithConnector(ctx, stderr, sc, an, router, disc)

	// Explicit shutdown of the management server now that the data-plane is draining.
	// shutdownMgmt is guarded by once-Do so calling it here and via defer is safe —
	// only one of the two actually executes the shutdown + Wait.
	shutdownMgmt()

	return runErr
}

// runAccessWithConnector contains all orchestration logic for the access-mode
// handler: wiring obligations 3–6 (ARCH-08 §6.5.1), the PC-1 connect-failure
// path, the PC-2 clean-shutdown path, and the PC-2.6 sc.Err() drain path.
//
// sc is the connector interface — *tmux.SessionConnector in production; a
// fakeConnector in tests (enabling PC-2 and PC-2.6 end-to-end coverage without
// a real PTY environment). an and router are pre-constructed by runAccess or the
// test caller. (ARCH-01 ADR-011 v1.5 §HIGH-B; ARCH-08 v2.1 §6.5.1 obligation 4.)
//
// disc is optional (variadic, at most one value). When non-nil, Discovery.Run is
// started as a WG-tracked goroutine alongside the sweep and frames-dropped tickers
// (Decision 6 Option Y; BC-2.04.008 PC-1 through PC-4). Existing callers that
// omit disc continue to compile — nil disc is a no-op for the discovery goroutine.
//
// TODO(S-BL.NODE-ADMISSION-PROVISIONING): the discovery goroutine stub below
// is a no-op placeholder so that AC-008 tests FAIL at Red Gate. The implementer
// must replace the stub with the real WG-tracked goroutine per rulings §3.2.
//
// tick_interval is applied in runAccess (via tickIntervalFor) before the
// half-channel is constructed and before this function is called. Further
// deferred config fields (drain_timeout, upstream_routers, keepalive_interval)
// are owned by S-7.04 (Wave 7). listen_addr binding is owned by S-BL.NI and
// applied at netingress listener bind in runRouter.
//
// On ctx already done: returns E-SYS-002 error immediately without starting
// goroutines (BC-2.04.007 PC-1 / AC-007).
// On sc.Connect failure: writes E-SYS-002 diagnostic to stderr and returns a
// non-nil error (caller calls os.Exit(1) — BC-2.04.007 PC-1 / AC-007).
// On clean shutdown (SIGTERM/SIGINT): returns nil (exit 0 — BC-2.04.007 PC-2 /
// AC-008).
// On mid-session double-failure (sc.Err() non-nil): logs E-SYS-002, cancels
// context, returns non-nil error — caller calls os.Exit(1)
// (BC-2.04.007 PC-2.6 / EC-007 / invariant 5 / AC-007).
func runAccessWithConnector(
	ctx context.Context,
	stderr io.Writer,
	sc connectorIface,
	an *session.AccessNode,
	router *routing.Router,
	disc ...*discovery.Discovery,
) error {
	_ = router // router retained (not discarded); used by AC-001 test surface

	// AC-008: install SIGTERM/SIGINT handler. signal.NotifyContext wraps ctx
	// so that SIGTERM/SIGINT cancels sigCtx. This is idempotent with any
	// signal.NotifyContext installed upstream — multiple registrations are safe.
	sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// Wrap sigCtx with a cancel so the Err() drain goroutine can trigger
	// shutdown on mid-session double-failure (BC-2.04.007 PC-2.6 / invariant 5).
	runCtx, cancel := context.WithCancel(sigCtx)
	defer cancel()

	// AC-007: if context is already cancelled before we start, return E-SYS-002.
	if err := runCtx.Err(); err != nil {
		msg := fmt.Sprintf("fatal: cannot connect to session backend: %v", err)
		fmt.Fprintln(stderr, msg) //nolint:errcheck // best-effort stderr write
		return fmt.Errorf("%s", msg)
	}

	if err := sc.Connect(runCtx); err != nil {
		// AC-007 (BC-2.04.007 PC-1): E-SYS-002 diagnostic — "fatal: cannot
		// connect to session backend: <reason>".
		msg := fmt.Sprintf("fatal: cannot connect to session backend: %v", err)
		fmt.Fprintln(stderr, msg) //nolint:errcheck // best-effort stderr write
		return fmt.Errorf("%s", msg)
	}

	var wg sync.WaitGroup

	// FIX 3 (exit-code race): explicit latch set by the drain goroutine BEFORE
	// calling cancel(). After wg.Wait() we branch on this latch — never on
	// sigCtx.Err() — making the exit-code mapping race-free.
	// (BC-2.04.007 PC-2.6 / EC-007 / invariant 5).
	var internalFailure atomic.Bool

	// BC-2.04.007 v1.1 invariant 5 / PC-2.6 / EC-007 / AC-007:
	// Drain sc.Err() in a wg-tracked goroutine. On non-nil error (mid-session
	// double-failure: both ctrl and PTY paths down, OR PTY-source EOF via
	// ErrPTYSourceEOF), log E-SYS-002 at ERROR level, set the internalFailure
	// latch, and cancel the root context.
	// The goroutine also exits when sc.Err() is closed (normal sc.Close() path).
	//
	// This MUST be registered with wg before the frame bridge goroutine so
	// it is joined during shutdown (ARCH-01 v1.4 §Daemon sc.Err() drain
	// obligation).
	wg.Add(1)
	go func() {
		defer wg.Done()
		for err := range sc.Err() {
			if err != nil {
				// E-SYS-002: "fatal: cannot connect to session backend: <reason>"
				// FIX 1: write to injected stderr, not os.Stderr (BC-2.04.002 Inv-3
				// never-silent; respects stderr redirection).
				msg := fmt.Sprintf("fatal: cannot connect to session backend: %v", err)
				fmt.Fprintln(stderr, msg) //nolint:errcheck // best-effort stderr write
				// FIX 3: set latch BEFORE cancel() so the exit-code branch after
				// wg.Wait() sees a consistent value regardless of SIGTERM timing.
				internalFailure.Store(true)
				// Cancel runCtx — triggers <-runCtx.Done() below and starts PC-2
				// shutdown sequence. The non-nil error from runAccessWithConnector
				// causes main() to call os.Exit(1) (BC-2.04.007 PC-2.6 / EC-007).
				cancel()
				return
			}
		}
	}()

	// Obligation 4 (AC-005): sc.Frames() → an.DeliverFrame bridge goroutine.
	wg.Add(1)
	go func() {
		defer wg.Done()
		startFramesBridge(an, sc.Frames())
	}()

	// Obligation 3 (AC-003): sweep ticker — wg-tracked per ARCH-01 v1.7
	// §Goroutine WaitGroup Contract and ARCH-08 v2.2 obligation 3.
	wg.Add(1)
	startSweepTicker(runCtx, &wg, an, sweepInterval, sweepDeadline)

	// Obligation 6 (AC-006): frames-dropped ticker — wg-tracked per ARCH-01 v1.7
	// §Goroutine WaitGroup Contract and ARCH-08 v2.2 obligation 6.
	// FIX 2: lg uses injected stderr (not os.Stderr).
	// FIX 4: pass framesDroppedInterval so the interval is injectable in tests.
	lg := log.New(stderr, "", 0)
	wg.Add(1)
	startFramesDroppedTicker(runCtx, &wg, sc, an, lg, framesDroppedInterval)

	// Decision 6 Option Y: disc.Run is WG-tracked in this function alongside the
	// sweep and frames-dropped tickers (BC-2.04.008 PC-1 through PC-4).
	// ARCH-01 v1.7 §Goroutine WaitGroup Contract: wg.Add(1) in caller BEFORE go.
	// Decision 7: ctx.Canceled from disc.Run is clean shutdown — NOT internalFailure.
	if len(disc) > 0 && disc[0] != nil {
		d := disc[0]
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := d.Run(runCtx); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				// Unexpected disc.Run error — log but do NOT set internalFailure;
				// that is reserved for the sc.Err() mid-session double-failure path.
				lg.Printf("discovery: run error: %v", err)
			}
		}()
	}

	// Block until context cancellation (SIGTERM/SIGINT, mid-session double-failure,
	// or direct cancel — AC-008 / BC-2.04.007 PC-2 / PC-2.6).
	<-runCtx.Done()

	// AC-008: clean shutdown — close sc (closes sc.frames, stopping bridge goroutine),
	// then wait for all goroutines to drain.
	_ = sc.Close()
	wg.Wait()

	// Determine exit code: if the Err() drain goroutine set internalFailure before
	// calling cancel(), return a non-nil error so main() exits 1
	// (BC-2.04.007 PC-2.6 / EC-007). A clean SIGTERM/SIGINT cancellation yields nil
	// (exit 0 — BC-2.04.007 PC-2). Using an atomic latch makes this race-free: the
	// latch is set BEFORE cancel(), so the value is stable by the time wg.Wait()
	// returns (FIX 3).
	if internalFailure.Load() {
		return fmt.Errorf("fatal: mid-session backend failure")
	}

	return nil
}

// buildAccessComponents constructs the session.AccessNode and routing.Router
// for the access mode handler, sharing the provided *admission.AdmittedKeySet
// so that BOTH the AccessNode and the Router operate on the same keyset.
//
// This satisfies the ARCH-08 v2.0 §6.5.1 obligation 1 non-tautology
// requirement: AC-001 can register a key into keys, then call
// router.RouteFrame(...) on the returned router instance, and observe
// E-ADM-016 emission — because the router and access node share ONE keyset.
//
// sc is wired as the KeystrokeSink (obligation 2 — AC-002; BC-2.04.005 PC-3).
//
// routerLogger is the routing.Logger injected into the router (FIX 2 — AC-001
// injectable logger; production passes stdLogger wrapping the injected stderr
// writer so observability respects stderr redirection and is capturable in tests).
//
// Returns: an (AccessNode with live SessionAuth), router (logger-wired Router).
// Neither return value is nil.
//
// Note: the router is constructed-but-not-in-live-data-path in Wave 3 (no
// network-ingress listener). It is retained so AC-001 can call RouteFrame
// on the daemon's own instance (ARCH-08 v2.0 §6.5.1 obligation 1).
func buildAccessComponents(
	keys *admission.AdmittedKeySet,
	pub *session.Publisher,
	sc *tmux.SessionConnector,
	routerLogger routing.Logger,
) (*session.AccessNode, *routing.Router) {
	auth := session.NewSessionAuth()
	an := session.NewAccessNode(pub, auth, session.WithKeystrokeSink(sc))
	router := buildRouter(keys, routerLogger)
	return an, router
}

// stdLogger wraps *log.Logger to satisfy routing.Logger's Log(string) method.
type stdLogger struct{ l *log.Logger }

func (s stdLogger) Log(msg string) { s.l.Print(msg) }

// newStdLogger returns a stdLogger writing to w. If w is nil, output is
// discarded — matches the nil-writer suppression contract used by runRouter
// for test injection.
func newStdLogger(w io.Writer) stdLogger {
	if w == nil {
		w = io.Discard
	}
	return stdLogger{log.New(w, "", 0)}
}

// buildRouter constructs the routing.Router with the provided routing.Logger
// and a FailureCounter injected (obligation 1 — AC-001; BC-2.05.008 PC-2/PC-5;
// FIX 2 injectable logger; C-1 wire-up).
//
// The FailureCounter (threshold=5, window=60s per BC-2.05.005 PC-3) is wired
// and will emit E-ADM-017 after ≥5 HMAC failures from the same source within the
// window. As of S-BL.NI the network-ingress listener that feeds RouteFrame is
// live (runRouter → netingress.Serve → routing.RouteFrame); the counter is no
// longer dormant when routers run — the live path is exercised by
// TestIntegration_EADM017_FiresThroughLiveIngress in internal/netingress.
//
// The logger is supplied by the caller (runAccess passes stdLogger wrapping the
// injected stderr writer; tests may pass a captureLogger for assertion).
// stdLogger satisfies both routing.Logger and admission.Logger (both are
// interface{ Log(string) }) so the same instance serves both the router and
// the counter.
func buildRouter(ks *admission.AdmittedKeySet, rl routing.Logger) *routing.Router {
	fc := admission.NewFailureCounter(hmacFailureThreshold, hmacFailureWindow, rl)
	return routing.NewRouter(ks, routing.WithLogger(rl), routing.WithFailureCounter(fc))
}

// startFramesBridge starts the sc.Frames() → accessNode.DeliverFrame goroutine
// (ARCH-08 §6.5.1 obligation 4; AC-005; ADR-011). The goroutine exits when
// framesCh is closed (on sc.Close()).
//
// Each ChannelFrame is converted to a frame.OuterHeader carrying the frame type
// and payload length. The routing fields (SVTNID, SrcAddr, DstAddr, HMACTag)
// are zero-valued at this layer — the inner channel carries terminal output
// rather than routed network frames.
//
// Note: this function is synchronous — the caller is responsible for running it
// in a goroutine.
func startFramesBridge(
	an *session.AccessNode,
	framesCh <-chan halfchannel.ChannelFrame,
) {
	for f := range framesCh {
		an.DeliverFrame(frame.OuterHeader{
			FrameType:  f.FrameType,
			PayloadLen: uint16(len(f.Payload)),
		})
	}
}

// startSweepTicker starts the periodic sweep goroutine that calls
// accessNode.Sweep(sweepDeadline) on each tick (ARCH-08 §6.5.1 obligation 3;
// AC-003; BC-2.04.004 PC-1+PC-3). Returns immediately; goroutine exits when
// ctx is cancelled.
//
// tickInterval controls how often Sweep is called (30s in production; 1ms in
// tests for fast eviction). sweepDead is the keepalive inactivity window passed
// to Sweep.
func startSweepTicker(
	ctx context.Context,
	wg *sync.WaitGroup,
	an *session.AccessNode,
	tickInterval time.Duration,
	sweepDead time.Duration,
) {
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(tickInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				an.Sweep(sweepDead)
			}
		}
	}()
}

// startFramesDroppedTicker starts the observability ticker that logs both
// sc.RelayDropped() (relay-layer drops) and an.FramesDropped() (ConsoleSet-
// layer drops) on each tick (ARCH-08 §6.5.1 obligation 6; AC-006;
// BC-2.04.006 v1.4 invariant 4; ARCH-01 v1.4 §Relay-drop counter contract).
//
// sc is connectorIface so that runAccessWithConnector can pass either a real
// *tmux.SessionConnector (production) or a fakeConnector (tests). Only
// RelayDropped() is called on sc here.
//
// tickInterval controls how often the log line is emitted. Production passes
// framesDroppedInterval (30s); tests may pass a shorter interval for fast
// tick-driven assertions (FIX 4 — mirrors startSweepTicker parameterisation).
//
// Log format: "frames_dropped relay=<N> consoles=<M>" (both counters cumulative,
// no reset). Emitted unconditionally on each tick — operators can distinguish
// relay overload (relay=N non-zero) from stalled console (consoles=M non-zero).
//
// Returns immediately; goroutine exits when ctx is cancelled.
func startFramesDroppedTicker(
	ctx context.Context,
	wg *sync.WaitGroup,
	sc connectorIface,
	an *session.AccessNode,
	lg *log.Logger,
	tickInterval time.Duration,
) {
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(tickInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				lg.Printf("frames_dropped relay=%d consoles=%d",
					sc.RelayDropped(), an.FramesDropped())
			}
		}
	}()
}

// loadOrGenerateAdmissionKeypair loads or generates the access node's persistent
// Ed25519 admission keypair from keyPath (PKCS#8 PEM format, "PRIVATE KEY" block).
//
// First run (file absent): generates a new keypair and writes it atomically to
// keyPath with mode 0600. Parent directory is created (0700) if absent.
// Logs INFO to stderr: "admission identity: generated new keypair at <path>".
//
// Subsequent start (file present): parses the PKCS#8 PEM and returns the key.
// Any parse failure is fail-closed — returns a non-nil error wrapping
// "access: load admission keypair: <path>: <reason>".
//
// File permissions broader than 0600: advisory WARNING logged, not fatal.
//
// On every successful return (first-run or subsequent load), emits an INFO log
// with the admission public key as base64url (no padding) of the raw 32-byte key.
//
// stderr is used for all log output. It must not be nil.
// Implements BC-2.09.004 PC-3 through PC-7; rulings §1.3; Decisions 3, 4.
func loadOrGenerateAdmissionKeypair(stderr io.Writer, keyPath string) (ed25519.PrivateKey, error) {
	lg := log.New(stderr, "", 0)

	f, openErr := os.Open(keyPath)
	if openErr != nil {
		if !errors.Is(openErr, os.ErrNotExist) {
			return nil, fmt.Errorf("access: load admission keypair: %s: %w", keyPath, openErr)
		}

		// First run: file absent — generate a new keypair.
		_, priv, genErr := ed25519.GenerateKey(rand.Reader)
		if genErr != nil {
			return nil, fmt.Errorf("access: generate admission keypair: %w", genErr)
		}

		// Marshal to PKCS#8 DER, then PEM-encode.
		der, marshalErr := x509.MarshalPKCS8PrivateKey(priv)
		if marshalErr != nil {
			return nil, fmt.Errorf("access: marshal admission keypair: %w", marshalErr)
		}
		pemData := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

		// Create parent directory if absent (MkdirAll is a no-op if it exists).
		if mkErr := os.MkdirAll(filepath.Dir(keyPath), 0o700); mkErr != nil {
			return nil, fmt.Errorf("access: create admission keypair dir: %w", mkErr)
		}

		// Atomic write: write to <path>.tmp, then rename.
		// F-7 fix: use O_CREATE|O_EXCL|O_WRONLY so the open fails if .tmp already
		// exists (e.g. from a previous interrupted run or attacker pre-creation).
		// This eliminates the TOCTOU window where WriteFile would inherit the
		// pre-existing mode until the subsequent Chmod ran. The mode 0o600 is applied
		// atomically at create time; no separate Chmod is needed.
		// (Security: mitigates stale-0644-.tmp-reuse; adversary pass-1 F-7.)
		tmpPath := keyPath + ".tmp"
		f, createErr := os.OpenFile(tmpPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if createErr != nil {
			return nil, fmt.Errorf("access: write admission keypair tmp: %w", createErr)
		}
		_, writeErr := f.Write(pemData)
		closeErr := f.Close()
		if writeErr != nil {
			_ = os.Remove(tmpPath)
			return nil, fmt.Errorf("access: write admission keypair tmp: %w", writeErr)
		}
		if closeErr != nil {
			_ = os.Remove(tmpPath)
			return nil, fmt.Errorf("access: write admission keypair tmp: %w", closeErr)
		}
		if renameErr := os.Rename(tmpPath, keyPath); renameErr != nil {
			_ = os.Remove(tmpPath)
			return nil, fmt.Errorf("access: rename admission keypair: %w", renameErr)
		}

		lg.Printf("admission identity: generated new keypair at %s", keyPath)

		logAdmissionPubkey(lg, priv)
		return priv, nil
	}
	defer func() { _ = f.Close() }()

	// Subsequent start: file present — check permissions, then load.
	fi, statErr := f.Stat()
	if statErr != nil {
		return nil, fmt.Errorf("access: load admission keypair: %s: %w", keyPath, statErr)
	}
	// F-1 fix: use OpenSSH semantics — warn if any group or other bit is set.
	// The naive numeric `perm > 0o600` predicate misses modes like 0o444 (292),
	// which is less than 0o600 (384) yet is world-readable. Testing for any group
	// or other permission bit (`perm&0o077 != 0`) catches all such cases correctly.
	// (BC-2.09.004 PC-4; rulings §1.4; adversary pass-1 F-1.)
	if perm := fi.Mode().Perm(); perm&0o077 != 0 {
		lg.Printf("admission identity key file %s has permissions %04o: expected 0600; private key may be exposed",
			keyPath, perm)
	}

	// Read and parse.
	data, readErr := io.ReadAll(f)
	if readErr != nil {
		return nil, fmt.Errorf("access: load admission keypair: %s: %w", keyPath, readErr)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("access: load admission keypair: %s: PEM decode failed", keyPath)
	}

	key, parseErr := x509.ParsePKCS8PrivateKey(block.Bytes)
	if parseErr != nil {
		return nil, fmt.Errorf("access: load admission keypair: %s: %w", keyPath, parseErr)
	}

	priv, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("access: load admission keypair: %s: not an Ed25519 key", keyPath)
	}

	logAdmissionPubkey(lg, priv)
	return priv, nil
}

// logAdmissionPubkey emits the INFO log with the base64url (no padding) encoding
// of the raw 32-byte Ed25519 admission public key (BC-2.09.004 PC-7; Decision 4).
func logAdmissionPubkey(lg *log.Logger, priv ed25519.PrivateKey) {
	pub := priv.Public().(ed25519.PublicKey)
	b64 := base64.RawURLEncoding.EncodeToString([]byte(pub))
	lg.Printf("access: admission identity pubkey (register with admin.key.register): %s", b64)
}
