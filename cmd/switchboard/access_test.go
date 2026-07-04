// Package main — access_test.go tests the runAccessWithConnector injection seam
// introduced by ARCH-01 ADR-011 v1.5 §HIGH-B (S-W3.04 adversarial convergence
// pass-2, task 6b).
//
// Both tests call runAccessWithConnector directly with a fakeConnector (unexported
// stub implementing connectorIface). This drives the PRODUCTION function rather than
// a test-local reconstruction of the drain logic — ensuring the PC-2.6 exit-code
// latch, E-SYS-002 log, and PC-2 clean-shutdown path are all tested end-to-end.
//
// AC-007 traces (BC-2.04.007 PC-2.6 + PC-2 + invariant 5):
//   - TestRunAccessWithConnectorPC26: fakeConnector.Err() delivers a non-nil error
//     → runAccessWithConnector returns non-nil (exit 1), stderr contains E-SYS-002.
//   - TestRunAccessWithConnectorPC2: context cancelled externally, no error on Err()
//     → runAccessWithConnector returns nil (exit 0), E-SYS-002 NOT written.
//
// Supersedes TestDaemonMidSessionDoubleFailureExitsNonZero (main_test.go), which
// reconstructed the drain logic in parallel — making it tautological per ARCH-01
// ADR-011 v1.5 §HIGH-B ruling.
package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/halfchannel"
	"github.com/arcavenae/switchboard/internal/routing"
	"github.com/arcavenae/switchboard/internal/session"
	"github.com/arcavenae/switchboard/internal/tmux"
)

// fakeConnector is an unexported stub implementing connectorIface for injection
// into runAccessWithConnector in PC-2 and PC-2.6 tests.
//
// Connect always returns nil (pre-condition for the mid-session/clean-shutdown
// paths). errCh and framesCh are controllable per test.
//
// Close() closes errCh and framesCh exactly once (sync.Once guards), unblocking
// the drain goroutine (range sc.Err()) and the bridge goroutine (range sc.Frames())
// inside runAccessWithConnector — mirroring the production *tmux.SessionConnector.Close()
// behaviour (closeErrCh.Do + closeForwardFrames.Do).
//
// Concurrency: safe; sync.Once guards channel closure.
type fakeConnector struct {
	// errCh is the channel returned by Err(). Pre-populate with a non-nil error
	// for PC-2.6; leave as a never-sending channel for PC-2.
	errCh chan error
	// framesCh is the channel returned by Frames(). Unbuffered; bridge goroutine
	// blocks on range until Close() closes it.
	framesCh chan halfchannel.ChannelFrame
	// relayDropped is returned by RelayDropped().
	relayDropped uint64

	// closeErrOnce and closeFramesOnce guard channel closure so Close() is
	// idempotent and race-free (mirrors production sync.Once pattern).
	closeErrOnce    sync.Once
	closeFramesOnce sync.Once
}

func (f *fakeConnector) Connect(_ context.Context) error { return nil }

func (f *fakeConnector) Frames() <-chan halfchannel.ChannelFrame {
	return f.framesCh
}

func (f *fakeConnector) Err() <-chan error {
	return f.errCh
}

// Close closes errCh and framesCh exactly once (idempotent), unblocking
// the drain goroutine and the bridge goroutine in runAccessWithConnector.
func (f *fakeConnector) Close() error {
	// Unblock the drain goroutine (range sc.Err() exits on channel close).
	f.closeErrOnce.Do(func() { close(f.errCh) })
	// Unblock the bridge goroutine (range sc.Frames() exits on channel close).
	f.closeFramesOnce.Do(func() { close(f.framesCh) })
	return nil
}

func (f *fakeConnector) RelayDropped() uint64 { return f.relayDropped }

// newMinimalAccessComponents constructs a minimal *session.AccessNode and
// *routing.Router suitable for injection into runAccessWithConnector in tests
// that do not need to inspect router or access-node behaviour — only the
// PC-2.6/PC-2 lifecycle path.
//
// Uses admission.NewAdmittedKeySet so the shared-keyset contract (ARCH-08 v2.0
// §6.5.1 obligation 1) is satisfied structurally, even though these tests do not
// exercise the router's RouteFrame path.
func newMinimalAccessComponents(t *testing.T) (*session.AccessNode, *routing.Router) {
	t.Helper()
	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)

	// Publish a session so AccessNode has something to attach to (not required
	// by PC-2/PC-2.6 paths, but avoids nil-panic in DeliverFrame if bridge fires).
	_ = pub.Publish("test-session")

	auth := session.NewSessionAuth()

	// newFakeSessionConnectorForSink gives us a real *tmux.SessionConnector as a
	// KeystrokeSink. We only need it for the WithKeystrokeSink parameter — it is
	// not the sc under test. Construct a minimal one that never connects.
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)
	ctrlForSink := tmux.New(pub, ds)
	ptyForSink := tmux.NewPTYProxy(pub, ds)
	scForSink := tmux.NewSessionConnector(ctrlForSink, ptyForSink)
	t.Cleanup(func() { _ = scForSink.Close() })

	an := session.NewAccessNode(pub, auth, session.WithKeystrokeSink(scForSink))

	cl := &captureLogger{}
	router := buildRouter(keys, cl)

	return an, router
}

// TestRunAccessWithConnectorPC26 — AC-007 / PC-2.6
// (BC-2.04.007 PC-2.6 + EC-007 + invariant 5; ARCH-01 ADR-011 v1.5 §HIGH-B)
//
// Verifies the mid-session double-failure path through the REAL runAccessWithConnector:
//
//  1. fakeConnector.Connect returns nil (pre-condition satisfied).
//  2. fakeConnector.Err() delivers a single non-nil error then closes.
//  3. runAccessWithConnector returns non-nil (exit 1).
//  4. The injected stderr contains the E-SYS-002 prefix:
//     "fatal: cannot connect to session backend: "
//  5. Returns within a bounded time (no hang).
//
// Discriminating property: if the production runAccessWithConnector's drain goroutine
// does NOT call cancel() on non-nil error from sc.Err(), runCtx.Done() never fires and
// the function hangs — the bounded deadline catches the regression. If the function
// returns nil instead of non-nil (i.e. internalFailure latch is not set), assertion 3
// fails.
//
// MUST exercise the production runAccessWithConnector — NOT a test-local drain loop.
// Per ARCH-01 ADR-011 v1.5 §HIGH-B: "Both PC-2 and PC-2.6 MUST be exercised through
// the real runAccessWithConnector call graph."
func TestRunAccessWithConnectorPC26(t *testing.T) {
	// AC-007/PC-2.6 — BC-2.04.007 PC-2.6 + EC-007 + invariant 5.
	// NOT t.Parallel(): context/cancel interaction.

	// E-SYS-002 canonical message prefix per error-taxonomy.md v2.1 §SYS.
	const esys002Prefix = "fatal: cannot connect to session backend: "
	// Inject a synthetic mid-session double-failure error (both ctrl and PTY down).
	midSessionErr := errors.New("both backends exhausted: synthetic mid-session double-failure")

	// Build fakeConnector: Err() has midSessionErr pre-loaded (buffered-1).
	// The drain goroutine reads the error, logs E-SYS-002, sets internalFailure,
	// and calls cancel(). Then runAccessWithConnector calls sc.Close(), which
	// closes errCh (via closeErrOnce.Do) so the drain goroutine's range exits.
	// Do NOT pre-close errCh — Close() closes it (mirroring production pattern).
	errCh := make(chan error, 1)
	errCh <- midSessionErr

	fc := &fakeConnector{
		errCh:    errCh,
		framesCh: make(chan halfchannel.ChannelFrame),
	}

	an, router := newMinimalAccessComponents(t)

	// Inject stderr writer so we can capture E-SYS-002 output.
	var stderr bytes.Buffer

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Drive the PRODUCTION runAccessWithConnector with the fake connector.
	// This MUST return within a bounded time; if it hangs the test fails.
	const returnDeadline = 500 * time.Millisecond
	resultCh := make(chan error, 1)
	go func() {
		resultCh <- runAccessWithConnector(ctx, &stderr, fc, an, router)
	}()

	var err error
	select {
	case err = <-resultCh:
	case <-time.After(returnDeadline):
		cancel() // clean up goroutine
		t.Fatalf("runAccessWithConnector did not return within %v after fakeConnector.Err() error; "+
			"drain goroutine must call cancel() on non-nil sc.Err() error (BC-2.04.007 PC-2.6 / invariant 5)",
			returnDeadline)
	}

	// Assertion 1 (PC-2.6 postcondition 3): non-nil return → exit 1.
	if err == nil {
		t.Fatal("runAccessWithConnector(fakeConnector with Err() error): got nil; " +
			"want non-nil error (exit 1 — BC-2.04.007 PC-2.6)")
	}

	// Assertion 2 (PC-2.6 postcondition 2 / E-SYS-002): injected stderr must contain
	// the canonical E-SYS-002 prefix. The error message also counts — check both.
	combined := err.Error() + " " + stderr.String()
	if !strings.Contains(combined, esys002Prefix) {
		t.Errorf("E-SYS-002 prefix %q not found in stderr or error; "+
			"got err=%q stderr=%q "+
			"(BC-2.04.007 PC-2.6; error-taxonomy.md v2.1 §SYS E-SYS-002)",
			esys002Prefix, err.Error(), stderr.String())
	}
}

// TestRunAccessWithConnectorPC2 — AC-007 / PC-2
// (BC-2.04.007 PC-2; ARCH-01 ADR-011 v1.5 §HIGH-B)
//
// Verifies the clean-shutdown path through the REAL runAccessWithConnector:
//
//  1. fakeConnector.Connect returns nil (pre-condition satisfied).
//  2. fakeConnector.Err() is a channel that never sends (no mid-session failure).
//  3. Context is cancelled externally (simulating SIGTERM/SIGINT → root-context cancel).
//  4. runAccessWithConnector returns nil (exit 0 — BC-2.04.007 PC-2).
//  5. Injected stderr does NOT contain the E-SYS-002 prefix
//     (clean shutdown writes no error message).
//  6. Returns within a bounded time.
//
// Discriminating property: if the production runAccessWithConnector does NOT select
// on <-runCtx.Done(), the function never returns after ctx cancel — the bounded
// deadline catches the regression. If it returns non-nil on clean SIGTERM, assertion 4
// fails (distinguishes PC-2 from PC-2.6).
//
// MUST exercise the production runAccessWithConnector — NOT a test-local drain loop.
func TestRunAccessWithConnectorPC2(t *testing.T) {
	// AC-007/PC-2 — BC-2.04.007 PC-2.
	// NOT t.Parallel(): context/cancel interaction.

	// E-SYS-002 canonical message prefix (must NOT appear in clean-shutdown path).
	const esys002Prefix = "fatal: cannot connect to session backend: "

	// Build fakeConnector: Err() never sends (clean-shutdown path; no mid-session error).
	// Use a non-buffered channel that is never written to and not closed until sc.Close().
	errCh := make(chan error) // unbuffered, never written to

	// framesCh is never closed by Close() until sc.Close() is called.
	// We make it with capacity 0 (unbuffered) so range in the bridge goroutine blocks.
	fc := &fakeConnector{
		errCh:    errCh,
		framesCh: make(chan halfchannel.ChannelFrame),
	}

	an, router := newMinimalAccessComponents(t)

	var stderr bytes.Buffer

	ctx, cancel := context.WithCancel(context.Background())

	const returnDeadline = 500 * time.Millisecond
	resultCh := make(chan error, 1)
	go func() {
		resultCh <- runAccessWithConnector(ctx, &stderr, fc, an, router)
	}()

	// Give the production goroutines a brief window to start (sweep ticker,
	// frames-dropped ticker, drain goroutine, bridge goroutine).
	time.Sleep(20 * time.Millisecond)

	// Simulate SIGTERM: cancel the context. runAccessWithConnector must observe
	// <-runCtx.Done() and return nil (exit 0 — PC-2 clean-shutdown path).
	cancel()

	var err error
	select {
	case err = <-resultCh:
	case <-time.After(returnDeadline):
		t.Fatalf("runAccessWithConnector did not return within %v after ctx cancel; "+
			"must observe <-runCtx.Done() promptly (BC-2.04.007 PC-2 — clean shutdown)",
			returnDeadline)
	}

	// Assertion 1 (PC-2 postcondition 5): nil return → exit 0.
	if err != nil {
		t.Errorf("runAccessWithConnector(ctx cancelled, no Err() error): got %v; "+
			"want nil (exit 0 — BC-2.04.007 PC-2 clean-shutdown path)", err)
	}

	// Assertion 2: E-SYS-002 must NOT appear on clean shutdown.
	// If the drain goroutine incorrectly fires E-SYS-002 on a nil/zero-value
	// channel close, this assertion catches it.
	stderrOut := stderr.String()
	if strings.Contains(stderrOut, esys002Prefix) {
		t.Errorf("clean-shutdown path wrote E-SYS-002 prefix %q to stderr; "+
			"must NOT write E-SYS-002 on SIGTERM/ctx-cancel (BC-2.04.007 PC-2); "+
			"got stderr: %q", esys002Prefix, stderrOut)
	}
}
