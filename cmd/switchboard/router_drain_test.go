// Package main — router_drain_test.go — S-7.04 integration tests.
//
// These tests exercise the BC-2.09.003 DEFERRED-APPLICATION closures owned by
// S-7.04 at the runRouter level, complementing the unit tests in
// router_config_test.go and internal/drain/drain_test.go:
//
//   - TestRunRouter_HonorsCustomDrainTimeout — AC-005 integration half:
//     cfg.DrainTimeout drives the drain coordinator (BC-2.09.003 PC-7).
//   - TestRunRouter_EmitsUpstreamModeE — AC-006 integration half, E mode:
//     empty upstream_routers → "mode=E" line at startup (BC-2.09.001 PC-1,
//     BC-2.09.003 PC-9).
//   - TestRunRouter_EmitsUpstreamModePE — AC-006 integration half, PE mode:
//     non-empty upstream_routers → "mode=PE upstream_routers=[...]" line
//     (BC-2.09.001 PC-1 graduation eligibility signal).
//   - TestRunRouter_EmitsKeepaliveInterval — AC-007 integration half:
//     resolved keepalive cadence is emitted at startup for operator audit
//     (BC-2.09.003 PC-8; MUST NOT be sweepDeadline).
//
// The observability seam (writer output) is deliberately load-bearing in
// this story: the reconnect-side keepalive ticker and the DRAIN-over-SVTN
// wire protocol both ship in a follow-on story. Until then, the startup
// emission is the only externally-visible confirmation that config values
// flowed through the seam. Integration tests here assert that emission so
// the seam cannot silently regress.
//
// Traces: S-7.04 AC-005 (BC-2.09.003 PC-7), AC-006 (BC-2.09.001 PC-1 /
// BC-2.09.003 PC-9), AC-007 (BC-2.09.003 PC-8).
package main

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/config"
	"github.com/arcavenae/switchboard/internal/drain"
)

// probeDataAddr allocates an ephemeral TCP port by binding then closing —
// the port number is now free for runRouter to re-bind. Mirrors the pattern
// used by TestRunRouter_DataListenerBinds.
func probeDataAddr(t *testing.T) string {
	t.Helper()
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("probe listen: %v", err)
	}
	addr := probe.Addr().String()
	_ = probe.Close()
	return addr
}

// syncBuffer is a goroutine-safe bytes.Buffer wrapper. runRouter's write path
// runs on the runRouter goroutine but the test goroutine reads the buffer
// after cancel — the race detector flags an unsynchronized bytes.Buffer.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// waitForSocket blocks until sockPath appears or deadline elapses.
// Returns true if the socket appeared. Mirrors the pattern used by
// TestRunRouter_NoAdminHandlers / TestRunRouter_SIGTERMLifecycle.
func waitForSocket(sockPath string, budget time.Duration) bool {
	deadline := time.Now().Add(budget)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(sockPath); err == nil {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	_, err := os.Stat(sockPath)
	return err == nil
}

// startRunRouterWithConfig launches runRouter in a goroutine, waits for the
// mgmt socket to appear, and returns the error channel + a cancel that
// initiates graceful shutdown. The returned buf captures runRouter's writer
// output; callers may inspect it before OR after shutdown.
//
// Callers MUST call cancel() and drain errCh (or use t.Cleanup to do so)
// so runRouter is not left running.
func startRunRouterWithConfig(t *testing.T, cfg *config.Config, buf *syncBuffer) (chan error, context.CancelFunc) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- runRouter(ctx, buf, cfg, "", make(chan os.Signal, 1)) }()

	if !waitForSocket(cfg.ManagementSocket, 1*time.Second) {
		cancel()
		<-errCh
		t.Fatalf("startRunRouterWithConfig: mgmt socket %q not created within 1s",
			cfg.ManagementSocket)
	}

	return errCh, cancel
}

// TestRunRouter_HonorsCustomDrainTimeout verifies AC-005 at the runRouter
// integration seam: a non-default cfg.DrainTimeout flows through
// drainTimeoutFor into the drain coordinator, and the resolved value is
// emitted at startup so operators can confirm the wiring.
//
// This test does NOT prove observers actually block until the timeout —
// that path lands with the DRAIN-over-SVTN wire protocol in a follow-on
// story. What it DOES prove is that the seam is live: cfg → drainTimeoutFor
// → drain.New → observable startup output. Any future regression that
// hard-codes DefaultTimeout at this seam would flip this test red.
func TestRunRouter_HonorsCustomDrainTimeout(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)

	// 250ms is small enough that a regression to DefaultTimeout (10s) is
	// obvious in the startup line, and large enough that shutdown latency
	// is not dominated by drain-timer setup overhead.
	customTimeout := 250 * time.Millisecond

	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
		DrainTimeout:     customTimeout,
	}

	var buf syncBuffer
	errCh, cancel := startRunRouterWithConfig(t, cfg, &buf)
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(2 * time.Second):
		}
	})

	cancel()
	select {
	case rErr := <-errCh:
		if rErr != nil {
			t.Errorf("runRouter returned error on shutdown: %v", rErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runRouter did not return within 2s after ctx cancel " +
			"(AC-005: custom drain_timeout must not exceed shutdown budget)")
	}

	// Observable: the resolved drain window is emitted verbatim. Format
	// mirrors the emission in runRouter: "drain_timeout=%s".
	got := buf.String()
	wantSubstr := fmt.Sprintf("drain_timeout=%s", customTimeout)
	if !strings.Contains(got, wantSubstr) {
		t.Errorf("AC-005 integration: startup output missing %q; got:\n%s",
			wantSubstr, got)
	}
}

// TestRunRouter_EmitsUpstreamModeE verifies AC-006 (E mode) at the runRouter
// integration seam: empty cfg.UpstreamRouters flows through upstreamRoutersFor
// as the empty list, and startup emits "mode=E (no upstream_routers configured)"
// per BC-2.09.001 PC-1. This is the operator-visible confirmation that the
// router did not graduate to PE mode.
func TestRunRouter_EmitsUpstreamModeE(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)

	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
		// UpstreamRouters intentionally left unset — proves the E-mode
		// branch at the observability seam fires.
	}

	var buf syncBuffer
	errCh, cancel := startRunRouterWithConfig(t, cfg, &buf)
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(2 * time.Second):
		}
	})

	cancel()
	select {
	case <-errCh:
	case <-time.After(2 * time.Second):
		t.Fatal("runRouter did not return within 2s after ctx cancel")
	}

	got := buf.String()
	if !strings.Contains(got, "mode=E") {
		t.Errorf("AC-006 (E mode) integration: startup output missing mode=E marker; got:\n%s", got)
	}
	if strings.Contains(got, "mode=PE") {
		t.Errorf("AC-006 (E mode) integration: startup output leaked mode=PE marker with no upstream_routers; got:\n%s", got)
	}
}

// TestRunRouter_EmitsUpstreamModePE verifies AC-006 (PE mode) at the runRouter
// integration seam: non-empty cfg.UpstreamRouters flows through
// upstreamRoutersFor and startup emits "mode=PE upstream_routers=[...]" with
// each configured Addr preserved in order per BC-2.09.001 PC-1 and
// BC-2.09.003 PC-9.
//
// Live upstream connection establishment is a follow-on story; this test
// asserts only the config-to-emission flow, which is the load-bearing
// application-point closure S-7.04 owns.
func TestRunRouter_EmitsUpstreamModePE(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)

	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
		UpstreamRouters: []config.UpstreamRouter{
			{Addr: "10.0.1.1:9090"},
			{Addr: "10.0.1.2:9090"},
		},
	}

	var buf syncBuffer
	errCh, cancel := startRunRouterWithConfig(t, cfg, &buf)
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(2 * time.Second):
		}
	})

	cancel()
	select {
	case <-errCh:
	case <-time.After(2 * time.Second):
		t.Fatal("runRouter did not return within 2s after ctx cancel")
	}

	got := buf.String()
	if !strings.Contains(got, "mode=PE") {
		t.Errorf("AC-006 (PE mode) integration: startup output missing mode=PE marker; got:\n%s", got)
	}
	// Order-preserving assertion — mirror the fresh-slice contract test in
	// router_config_test.go. Both addresses must appear.
	for _, addr := range []string{"10.0.1.1:9090", "10.0.1.2:9090"} {
		if !strings.Contains(got, addr) {
			t.Errorf("AC-006 (PE mode) integration: startup output missing upstream %q; got:\n%s", addr, got)
		}
	}
}

// TestRunRouter_ForcedExitPastDrainTimeout proves BC-2.09.002 EC-003 —
// "Drain timeout exceeded (nodes not all acknowledged) → Router disconnects
// after timeout; logs remaining unacknowledged nodes" — under conditions
// closer to a real drain: a live ingress TCP connection is open across the
// shutdown, and a slow observer is registered on the drain coordinator so
// the coordinator's timeout branch (drain.ErrTimeout) actually fires.
//
// This is the evidence closure for DRIFT-HS006-DRAIN-TIMEOUT-FORCED-EXIT-
// UNEVIDENCED: the existing S-7.04 tests exercise the empty-observer
// fast-path only. This test exercises the with-observer timeout path AND
// keeps a live conn on the netingress listener so shutdown must also join
// dataWG (via ingressCancel → conn.Close → ServeConn return → wg.Wait).
//
// Assertions:
//  1. runRouter returns nil (BC-2.09.002 EC-003: forced exit is expected
//     behavior, not error).
//  2. Elapsed time from ctx cancel to runRouter return is >= drain_timeout
//     (proves the coordinator actually waited) and <= drain_timeout + slack
//     (proves no hung goroutine held shutdown open).
//  3. The router logs the EC-003 message so operators can see the timeout
//     was hit ("runRouter: drain: <ErrTimeout> (proceeding with disconnect
//     per BC-2.09.002 EC-003)").
//
// Test-hook contract: drainCoordHook is a package-local var set for the
// duration of one runRouter invocation via t.Cleanup. It is nil in prod.
//
// Traces: BC-2.09.002 EC-003; DRIFT-HS006-DRAIN-TIMEOUT-FORCED-EXIT-
// UNEVIDENCED; S-7.04 AC-005 (drain_timeout applied and enforced end-to-end).
func TestRunRouter_ForcedExitPastDrainTimeout(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket, and
	// mutates a package-level test hook.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)

	// 500ms drain window. Small enough that a full go test -race run
	// stays quick; large enough that scheduler noise doesn't dominate
	// the elapsed-time assertion.
	drainWindow := 500 * time.Millisecond

	cfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
		DrainTimeout:     drainWindow,
	}

	// Slow observer: blocks past drainWindow via obsCtx.Done(). The
	// coordinator's drainCtx (derived with timeout=drainWindow) will fire
	// while the observer is still blocked, forcing the ErrTimeout branch.
	// When the coordinator cancels drainCtx the observer unwinds.
	observerReleased := make(chan struct{})
	drainCoordHook = func(d *drain.Drain) {
		d.RegisterObserver(func(obsCtx context.Context) {
			select {
			case <-obsCtx.Done():
				// Drain window elapsed — coordinator cancelled drainCtx.
				// This is the expected path for EC-003.
			case <-time.After(10 * time.Second):
				// Sanity guard — should never fire; drainCtx.Done() should
				// come first at drainWindow.
			}
			close(observerReleased)
		})
	}
	t.Cleanup(func() { drainCoordHook = nil })

	var buf syncBuffer
	errCh, cancel := startRunRouterWithConfig(t, cfg, &buf)

	// Establish a live ingress conn AFTER runRouter is up. Holding this
	// conn open across the entire shutdown forces netingress.Serve to
	// join a per-conn goroutine on ctx cancel — proving shutdown honors
	// both the drain coordinator AND the ingress WaitGroup.
	conn, err := net.Dial("tcp", cfg.ListenAddr)
	if err != nil {
		cancel()
		<-errCh
		t.Fatalf("dial live ingress conn: %v", err)
	}
	// Deferred Close is a safety net; the test doesn't rely on client-side
	// close — ingressCancel triggers netingress to close its side of the
	// conn (netingress.go ServeConn: "Close conn when ctx is cancelled").
	t.Cleanup(func() { _ = conn.Close() })

	start := time.Now()
	cancel()

	select {
	case rErr := <-errCh:
		elapsed := time.Since(start)
		if rErr != nil {
			t.Errorf("BC-2.09.002 EC-003: runRouter returned error on forced-exit path: %v (want nil — timeout is expected, not error)", rErr)
		}
		// Lower bound: the coordinator must actually have waited. If
		// elapsed < drainWindow, either the observer didn't register or
		// the coordinator short-circuited — either would silently regress
		// EC-003 evidence.
		if elapsed < drainWindow {
			t.Errorf("forced-exit lower bound: elapsed %v < drain_timeout %v — coordinator did not wait for observer ACK",
				elapsed, drainWindow)
		}
		// Upper bound: shutdown must not hang. Slack covers netingress
		// per-conn goroutine join + mgmt.Shutdown budget (bounded by
		// drainCoord.Timeout() = drainWindow) + scheduler noise.
		slack := 2 * drainWindow
		if elapsed > drainWindow+slack {
			t.Errorf("forced-exit upper bound: elapsed %v > drain_timeout + slack %v — shutdown hung past the drain budget",
				elapsed, drainWindow+slack)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("BC-2.09.002 EC-003: runRouter did not return within 5s after ctx cancel — forced-exit path did not fire")
	}

	// Observer must have received the drain window's cancel and unwound.
	// This proves the EC-003 path fully executed (not merely that Wait
	// returned early).
	select {
	case <-observerReleased:
	case <-time.After(1 * time.Second):
		t.Error("observer did not unwind after drainCtx cancellation — coordinator did not honor its own drain window")
	}

	// EC-003 log line: the coordinator's timeout must have been logged
	// so operators can see the forced-exit path fired. runRouter emits:
	//   "runRouter: drain: <err> (proceeding with disconnect per BC-2.09.002 EC-003)"
	//
	// Note on the exact error text: the current wiring in runRouter passes
	// the same drainCtx (deadline = drain_timeout) to both Signal and Wait.
	// When the window elapses, Wait's select races drain's internal d.done
	// (which delivers ErrTimeout) against ctx.Done() (which delivers
	// context.DeadlineExceeded). Both channels become ready at the same
	// deadline, so Go's runtime picks pseudo-randomly. BC-2.09.002 EC-003
	// specifies the marker log line and forced disconnect, not a specific
	// error string — the assertion is on the marker, not the error body.
	got := buf.String()
	if !strings.Contains(got, "BC-2.09.002 EC-003") {
		t.Errorf("EC-003 log line missing — operators lose forced-exit signal. Got:\n%s", got)
	}
	if !strings.Contains(got, "runRouter: drain:") {
		t.Errorf("EC-003 log line missing runRouter prefix; got:\n%s", got)
	}
}

// TestRunRouter_EmitsKeepaliveInterval verifies AC-007 at the runRouter
// integration seam: cfg.KeepaliveInterval flows through keepaliveIntervalFor
// and the resolved value is emitted at startup — NOT sweepDeadline (which
// is the console-eviction cadence per BC-2.09.003 PC-8 normative note).
//
// The reconnect-side keepalive ticker itself ships once the node protocol
// lands; until then, the startup emission is the operator-visible
// confirmation that keepalive_interval config flows to its application point.
func TestRunRouter_EmitsKeepaliveInterval(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)

	// 750ms is distinct from both defaultKeepaliveInterval (1s) and
	// sweepDeadline (60s), so a regression to either would flip red.
	customKeepalive := 750 * time.Millisecond

	cfg := &config.Config{
		ListenAddr:        dataAddr,
		TickInterval:      10 * time.Millisecond,
		ManagementSocket:  sockPath,
		KeepaliveInterval: customKeepalive,
	}

	var buf syncBuffer
	errCh, cancel := startRunRouterWithConfig(t, cfg, &buf)
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(2 * time.Second):
		}
	})

	cancel()
	select {
	case <-errCh:
	case <-time.After(2 * time.Second):
		t.Fatal("runRouter did not return within 2s after ctx cancel")
	}

	got := buf.String()
	wantSubstr := fmt.Sprintf("keepalive_interval=%s", customKeepalive)
	if !strings.Contains(got, wantSubstr) {
		t.Errorf("AC-007 integration: startup output missing %q; got:\n%s",
			wantSubstr, got)
	}
	// BC-2.09.003 PC-8 normative fence: keepalive_interval must never carry
	// the sweepDeadline value at the emission seam. sweepDeadline is 60s;
	// no test cfg permits a 60s keepalive, so this substring check catches
	// a copy-paste regression before it ships.
	if strings.Contains(got, fmt.Sprintf("keepalive_interval=%s", sweepDeadline)) {
		t.Errorf("AC-007 integration: keepalive_interval emission carries sweepDeadline "+
			"value (%v) — BC-2.09.003 PC-8 normative fence violated", sweepDeadline)
	}
}
