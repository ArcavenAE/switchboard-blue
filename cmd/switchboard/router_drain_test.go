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
	go func() { errCh <- runRouter(ctx, buf, cfg) }()

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
