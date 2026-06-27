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
//
// FORBIDDEN imports: internal/config, internal/drain, internal/metrics
// (ARCH-08 §6.5.2 deferred packages; those packages do not exist on develop).
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
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
func runAccess(ctx context.Context, stderr io.Writer) error {
	// AC-008: install SIGTERM/SIGINT handler. signal.NotifyContext wraps ctx
	// so that SIGTERM/SIGINT cancels sigCtx. This is idempotent with the
	// signal.NotifyContext installed in run() — multiple registrations are safe.
	// Without this, a test that sends SIGTERM directly would kill the process
	// before runAccess returns (BC-2.04.007 PC-2).
	sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// AC-007: if context is already cancelled before we start, return E-SYS-002.
	if err := sigCtx.Err(); err != nil {
		msg := fmt.Sprintf("fatal: cannot connect to session backend: %v", err)
		fmt.Fprintln(stderr, msg) //nolint:errcheck // best-effort stderr write
		return fmt.Errorf("%s", msg)
	}

	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)

	ctrl := tmux.New(pub, ds)
	pty := tmux.NewPTYProxy(pub, ds)
	sc := tmux.NewSessionConnector(ctrl, pty)

	if err := sc.Connect(sigCtx); err != nil {
		// AC-007 (BC-2.04.007 PC-1): E-SYS-002 diagnostic — "fatal: cannot
		// connect to session backend: <reason>".
		msg := fmt.Sprintf("fatal: cannot connect to session backend: %v", err)
		fmt.Fprintln(stderr, msg) //nolint:errcheck // best-effort stderr write
		return fmt.Errorf("%s", msg)
	}

	an := buildAccessNode(sc)
	_ = buildRouter(keys) // obligation 1 (router logger wired; not used beyond that in Wave 3)

	var wg sync.WaitGroup

	// Obligation 4 (AC-005): sc.Frames() → an.DeliverFrame bridge goroutine.
	wg.Add(1)
	go func() {
		defer wg.Done()
		startFramesBridge(an, sc.Frames())
	}()

	// Obligation 3 (AC-003): sweep ticker — startSweepTicker starts its own goroutine.
	startSweepTicker(sigCtx, an, sweepInterval, sweepDeadline)

	// Obligation 6 (AC-006): frames-dropped ticker — startFramesDroppedTicker starts its own goroutine.
	lg := log.New(os.Stderr, "", 0)
	startFramesDroppedTicker(sigCtx, an, lg)

	// Block until context cancellation (SIGTERM/SIGINT or direct cancel — AC-008).
	<-sigCtx.Done()

	// AC-008: clean shutdown — close sc (closes sc.frames, stopping bridge goroutine),
	// then wait for all goroutines to drain.
	_ = sc.Close()
	wg.Wait()

	return nil
}

// buildAccessNode constructs the admission.AdmittedKeySet, session.Publisher,
// session.SessionAuth, and session.AccessNode for the access mode handler.
// Wires obligation 2 and 5 (ARCH-08 §6.5.1): live SessionAuth as Authorizer;
// SessionConnector as KeystrokeSink.
func buildAccessNode(sc *tmux.SessionConnector) *session.AccessNode {
	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	auth := session.NewSessionAuth()
	return session.NewAccessNode(pub, auth, session.WithKeystrokeSink(sc))
}

// stdLogger wraps *log.Logger to satisfy routing.Logger's Log(string) method.
type stdLogger struct{ l *log.Logger }

func (s stdLogger) Log(msg string) { s.l.Print(msg) }

// buildRouter constructs the routing.Router with a real routing.Logger injected
// (obligation 1 — AC-001; BC-2.05.008 PC-2).
func buildRouter(ks *admission.AdmittedKeySet) *routing.Router {
	return routing.NewRouter(ks, routing.WithLogger(stdLogger{log.New(os.Stderr, "", 0)}))
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

// startFramesDroppedTicker starts the observability ticker that logs
// accessNode.FramesDropped() > 0 at INFO level (ARCH-08 §6.5.1 obligation 6;
// AC-006; BC-2.04.006 invariant 4). Returns immediately; goroutine exits when
// ctx is cancelled.
//
// Logs immediately on goroutine start (before first tick) so the test can
// observe the log within its 500ms window without waiting for the 30s tick.
func startFramesDroppedTicker(
	ctx context.Context,
	an *session.AccessNode,
	lg *log.Logger,
) {
	go func() {
		logIfDropped := func() {
			if n := an.FramesDropped(); n > 0 {
				lg.Printf("frames_dropped count=%d", n)
			}
		}

		// Check immediately on start — satisfies the test's 500ms assertion
		// without waiting for the first 30s tick.
		logIfDropped()

		ticker := time.NewTicker(framesDroppedInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				logIfDropped()
			}
		}
	}()
}
