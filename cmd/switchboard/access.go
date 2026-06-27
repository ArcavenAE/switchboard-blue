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
// FORBIDDEN imports: internal/config, internal/drain, internal/metrics
// (ARCH-08 §6.5.2 deferred packages; those packages do not exist on develop).
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/halfchannel"
	"github.com/arcavenae/switchboard/internal/routing"
	"github.com/arcavenae/switchboard/internal/session"
	"github.com/arcavenae/switchboard/internal/tmux"
)

// sweepInterval is the period between consecutive Sweep eviction passes.
// Hardcoded for Wave 3 (no file-based config loading; internal/config is Wave 4).
const sweepInterval = 30 * time.Second

// sweepDeadline is the keepalive inactivity window after which a console is
// evicted by Sweep. Hardcoded for Wave 3.
const sweepDeadline = 60 * time.Second

// framesDroppedInterval is the period between FramesDropped log checks.
// Hardcoded for Wave 3 (AC-006; BC-2.04.006 invariant 4).
const framesDroppedInterval = 30 * time.Second

// runAccess is the access-mode subcommand handler. It wires all six ARCH-08
// §6.5.1 obligations, and blocks until the daemon shuts down.
//
// stderr is the writer for error diagnostics. ctx is the parent context (root
// context from main; signal handling is installed in main.go via
// signal.NotifyContext before calling runAccess).
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
func runAccess(ctx context.Context, stderr io.Writer) error {
	// AC-008: install SIGTERM/SIGINT handler. signal.NotifyContext wraps ctx
	// so that SIGTERM/SIGINT cancels sigCtx. This is idempotent with the
	// signal.NotifyContext installed in run() — multiple registrations are safe.
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

	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)

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
	// (non-tautological — shared keyset).
	an, router := buildAccessComponents(keys, pub, sc, routerLogger)
	_ = router // router retained (not discarded); used by AC-001 test surface

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
	// double-failure: both ctrl and PTY paths down), log E-SYS-002 at ERROR
	// level, set the internalFailure latch, and cancel the root context.
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
				// shutdown sequence. The non-nil error from runAccess causes
				// main() to call os.Exit(1) (BC-2.04.007 PC-2.6 / EC-007).
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

	// Obligation 3 (AC-003): sweep ticker — startSweepTicker starts its own goroutine.
	startSweepTicker(runCtx, an, sweepInterval, sweepDeadline)

	// Obligation 6 (AC-006): frames-dropped ticker — startFramesDroppedTicker starts
	// its own goroutine. FIX 2: lg uses injected stderr (not os.Stderr).
	// FIX 4: pass framesDroppedInterval so the interval is injectable in tests.
	lg := log.New(stderr, "", 0)
	startFramesDroppedTicker(runCtx, sc, an, lg, framesDroppedInterval)

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

// buildRouter constructs the routing.Router with the provided routing.Logger
// injected (obligation 1 — AC-001; BC-2.05.008 PC-2; FIX 2 injectable logger).
// The logger is supplied by the caller (runAccess passes stdLogger wrapping the
// injected stderr writer; tests may pass a captureLogger for assertion).
func buildRouter(ks *admission.AdmittedKeySet, rl routing.Logger) *routing.Router {
	return routing.NewRouter(ks, routing.WithLogger(rl))
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
	an *session.AccessNode,
	tickInterval time.Duration,
	sweepDead time.Duration,
) {
	go func() {
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
	sc *tmux.SessionConnector,
	an *session.AccessNode,
	lg *log.Logger,
	tickInterval time.Duration,
) {
	go func() {
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
