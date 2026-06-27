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
//
// STUB: all non-trivial function bodies panic("not implemented: … (S-W3.04
// AC-NNN)"). go build ./... passes; all daemon integration tests are red
// (BC-5.38.001 Red Gate).
package main

import (
	"context"
	"io"
	"log"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
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

// runAccess is the access-mode subcommand handler. It wires all six ARCH-08
// §6.5.1 obligations, installs signal handling, and blocks until the daemon
// shuts down.
//
// stdout is the writer for human-readable messages (version, errors). ctx is the
// parent context (root context from main; signal handling is installed in main.go
// via signal.NotifyContext before calling runAccess).
//
// On sc.Connect failure: writes E-SYS-002 diagnostic to stderr and returns a
// non-nil error (caller calls os.Exit(1) — BC-2.04.007 PC-1 / AC-007).
// On clean shutdown (SIGTERM/SIGINT): returns nil (exit 0 — BC-2.04.007 PC-2 /
// AC-008).
//
// STUB: body panics — all six integration tests (AC-001 through AC-008) are red
// (BC-5.38.001). go build ./... passes because the signature is correct and all
// imports compile.
func runAccess(_ context.Context, _ io.Writer) error {
	panic("not implemented: runAccess daemon wiring (S-W3.04 AC-001–AC-008)")
}

// buildAccessNode constructs the admission.AdmittedKeySet, session.Publisher,
// session.SessionAuth, and session.AccessNode for the access mode handler.
// Wires obligation 2 and 5 (ARCH-08 §6.5.1): live SessionAuth as Authorizer;
// SessionConnector as KeystrokeSink.
//
// STUB: panics (S-W3.04 AC-002).
func buildAccessNode(sc *tmux.SessionConnector) *session.AccessNode {
	panic("not implemented: buildAccessNode (S-W3.04 AC-002)")
}

// buildRouter constructs the routing.Router with a real routing.Logger injected
// (obligation 1 — AC-001; BC-2.05.008 PC-2).
//
// STUB: panics (S-W3.04 AC-001).
func buildRouter(ks *admission.AdmittedKeySet) *routing.Router {
	panic("not implemented: buildRouter (S-W3.04 AC-001)")
}

// startFramesBridge starts the sc.Frames() → accessNode.DeliverFrame goroutine
// (ARCH-08 §6.5.1 obligation 4; AC-005; ADR-011). The goroutine exits when
// sc.Frames() is closed (on sc.Close()).
//
// STUB: panics (S-W3.04 AC-004/AC-005).
func startFramesBridge(
	_ *session.AccessNode,
	_ <-chan halfchannel.ChannelFrame,
) {
	panic("not implemented: startFramesBridge Frames()→DeliverFrame goroutine (S-W3.04 AC-005)")
}

// startSweepTicker starts the periodic sweep goroutine that calls
// accessNode.Sweep(sweepDeadline) on each tick (ARCH-08 §6.5.1 obligation 3;
// AC-003; BC-2.04.004 PC-1+PC-3). Exits when ctx is cancelled.
//
// STUB: panics (S-W3.04 AC-003).
func startSweepTicker(
	_ context.Context,
	_ *session.AccessNode,
	_ time.Duration,
	_ time.Duration,
) {
	panic("not implemented: startSweepTicker goroutine (S-W3.04 AC-003)")
}

// startFramesDroppedTicker starts the 30-second observability ticker that logs
// accessNode.FramesDropped() > 0 at INFO level (ARCH-08 §6.5.1 obligation 6;
// AC-006; BC-2.04.006 invariant 4). Exits when ctx is cancelled.
//
// STUB: panics (S-W3.04 AC-006).
func startFramesDroppedTicker(
	_ context.Context,
	_ *session.AccessNode,
	_ *log.Logger,
) {
	panic("not implemented: startFramesDroppedTicker goroutine (S-W3.04 AC-006)")
}
