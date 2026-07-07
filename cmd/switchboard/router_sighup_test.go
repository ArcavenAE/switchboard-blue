// Package main — router_sighup_test.go — S-7.04-FU-SIGHUP-RELOAD integration tests.
//
// These tests exercise the SIGHUP config-reload path in runRouter at the
// AC-001..AC-004 level.  They will all FAIL at the Red Gate because the
// sighupCh select-case in runRouter is currently a no-op stub.  Each test
// is named exactly as specified in the story.
//
// Traces: S-7.04-FU-SIGHUP-RELOAD AC-001 (TestRunRouter_SIGHUPReload_EtoPE),
//
//	AC-002 (TestRunRouter_SIGHUPReload_BadConfig_FailClosed),
//	AC-003 (TestRunRouter_SIGHUPReload_SessionsNotInterrupted),
//	AC-004 (TestRunRouter_VP038_EtoPEViaConfigOnly).
package main

import (
	"bufio"
	"bytes"
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/config"
	"github.com/arcavenae/switchboard/internal/testenv"
	"gopkg.in/yaml.v3"
)

// ── shared helpers ─────────────────────────────────────────────────────────────

// writeTempConfig marshals cfg as YAML into a new file under t.TempDir() and
// returns the path.  Fatals the test on marshal or write error.
func writeTempConfig(t *testing.T, cfg *config.Config) string {
	t.Helper()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("writeTempConfig: marshal: %v", err)
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("writeTempConfig: write %q: %v", path, err)
	}
	return path
}

// writeInvalidConfig writes a YAML file whose content will fail
// config.LoadFile or (*Config).Validate.  Returns the path.
// Using an empty listen_addr triggers E-CFG-001.
func writeInvalidConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	// listen_addr is required but empty → Validate returns E-CFG-001.
	content := "tick_interval: 10ms\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writeInvalidConfig: write %q: %v", path, err)
	}
	return path
}

// startRunRouterForReload launches runRouter with an injected sighupCh so that
// tests can drive the reload seam in-process.  Returns the buf that receives
// runRouter's writer output, the errCh, the cancel func, and the sighupCh.
//
// The caller MUST call cancel() and drain errCh (a t.Cleanup block works).
// waitForSocket is reused from router_drain_test.go (same package).
func startRunRouterForReload(t *testing.T, cfg *config.Config, cfgPath string) (
	buf *syncBuffer, errCh chan error, cancel context.CancelFunc, sighupCh chan os.Signal,
) {
	t.Helper()
	buf = &syncBuffer{}
	sighupCh = make(chan os.Signal, 1)
	ctx, cancelFn := context.WithCancel(context.Background())
	errCh = make(chan error, 1)
	go func() {
		errCh <- runRouter(ctx, buf, cfg, cfgPath, sighupCh)
	}()
	if !waitForSocket(cfg.ManagementSocket, 2*time.Second) {
		cancelFn()
		<-errCh
		t.Fatalf("startRunRouterForReload: mgmt socket %q not created within 2s",
			cfg.ManagementSocket)
	}
	return buf, errCh, cancelFn, sighupCh
}

// scanForLine polls buf until a line containing substr appears or the deadline
// elapses.  Returns true if found.
func scanForLine(buf *syncBuffer, substr string, budget time.Duration) bool { //nolint:unparam // budget is a caller-controlled knob; all current callers use 2s but the parameter is intentional
	deadline := time.Now().Add(budget)
	for time.Now().Before(deadline) {
		if strings.Contains(buf.String(), substr) {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return strings.Contains(buf.String(), substr)
}

// scanForReloadFailedLine polls buf until a single output line matches the
// EC-004 verbatim format:
//
//	config reload failed: <err>; continuing with previous config
//
// where <err> is non-empty and contains innerMarker (the error code that must
// appear inside the EC-004 wrapper, e.g. "E-CFG-001" for Validate errors,
// "E-CFG-004" for LoadFile not-found, "E-CFG-005" for parse errors).
// The check is line-scoped so that prefix and suffix must appear on the same
// output line (AC-002 format contract).
// Returns true if such a line is found within budget.
func scanForReloadFailedLine(buf *syncBuffer, innerMarker string, budget time.Duration) bool {
	const prefix = "config reload failed: "
	const suffix = "; continuing with previous config"
	check := func() bool {
		scanner := bufio.NewScanner(bytes.NewBufferString(buf.String()))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, prefix) &&
				strings.Contains(line, innerMarker) &&
				strings.HasSuffix(line, suffix) {
				return true
			}
		}
		return false
	}
	if check() {
		return true
	}
	deadline := time.Now().Add(budget)
	for time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
		if check() {
			return true
		}
	}
	return check()
}

// ── AC-001 ─────────────────────────────────────────────────────────────────────

// TestRunRouter_SIGHUPReload_EtoPE verifies AC-001: sending syscall.SIGHUP
// on the injected channel after writing a PE config to disk causes runRouter
// to emit a "mode=PE" line and update its internal upstreamRouters state,
// without terminating the daemon.
//
// RED GATE: fails until the sighupCh case in runRouter implements reload dispatch.
func TestRunRouter_SIGHUPReload_EtoPE(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)

	// Start in E mode (no upstream_routers).
	startCfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}

	// Write a PE config to disk (one valid upstream entry).  This is the file
	// runRouter will reload when SIGHUP fires.  Use a separate probed port so
	// the addr is syntactically valid even though no server will be there.
	reloadAddr := probeDataAddr(t)
	reloadCfg := &config.Config{
		ListenAddr:   dataAddr, // same listen_addr — only upstream list changes
		TickInterval: 10 * time.Millisecond,
		UpstreamRouters: []config.UpstreamRouter{
			{Addr: reloadAddr},
		},
		ManagementSocket: sockPath,
	}
	cfgPath := writeTempConfig(t, reloadCfg)

	buf, errCh, cancel, sighupCh := startRunRouterForReload(t, startCfg, cfgPath)
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(3 * time.Second):
		}
	})

	// Precondition: confirm startup is in E mode.
	if !scanForLine(buf, "mode=E", 2*time.Second) {
		t.Fatalf("AC-001 precondition: startup did not emit mode=E within 2s; got:\n%s", buf.String())
	}

	// F-P2-003: deep-copy of cfg.UpstreamRouters before SIGHUP so we can
	// assert the original cfg pointer is not mutated on a SUCCESSFUL reload
	// (AC-001 postcondition 6 — cfg is not mutated; only the upstreamRouters
	// local variable inside runRouter changes).
	origUpstreams := make([]config.UpstreamRouter, len(startCfg.UpstreamRouters))
	copy(origUpstreams, startCfg.UpstreamRouters)

	// Send reload signal.
	sighupCh <- syscall.SIGHUP

	// Postcondition: mode=PE line must appear (AC-001 postcondition 4).
	if !scanForLine(buf, "mode=PE", 2*time.Second) {
		t.Errorf("AC-001: after SIGHUP, output missing mode=PE line within 2s; got:\n%s", buf.String())
	}
	if !scanForLine(buf, reloadAddr, 2*time.Second) {
		t.Errorf("AC-001: after SIGHUP, output missing upstream addr %q; got:\n%s", reloadAddr, buf.String())
	}

	// F-P2-003: cfg pointer must not be mutated by a successful reload.
	// The reload path operates on the fresh 'loaded' struct; cfg (the
	// startup-time parameter) must remain unchanged throughout (AC-001 PC-6).
	if len(startCfg.UpstreamRouters) != len(origUpstreams) {
		t.Errorf("AC-001/F-P2-003: cfg.UpstreamRouters length mutated on successful reload: before=%d after=%d",
			len(origUpstreams), len(startCfg.UpstreamRouters))
	} else {
		for i, want := range origUpstreams {
			if startCfg.UpstreamRouters[i] != want {
				t.Errorf("AC-001/F-P2-003: cfg.UpstreamRouters[%d] mutated on successful reload: before=%v after=%v",
					i, want, startCfg.UpstreamRouters[i])
			}
		}
	}

	// Postcondition 7: daemon has not returned.
	select {
	case rErr := <-errCh:
		t.Errorf("AC-001: runRouter returned prematurely after SIGHUP: %v", rErr)
	default:
		// expected — still running
	}
}

// ── AC-002 ─────────────────────────────────────────────────────────────────────

// TestRunRouter_SIGHUPReload_BadConfig_FailClosed verifies AC-002 / BC-2.09.003
// EC-004: a SIGHUP that points at an invalid config file leaves the daemon on
// the previous config and emits the EC-004 log line.
//
// RED GATE: fails until the sighupCh case implements fail-closed reload.
func TestRunRouter_SIGHUPReload_BadConfig_FailClosed(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)

	startCfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}

	// Write an invalid config (missing required listen_addr → fails Validate).
	badPath := writeInvalidConfig(t)

	buf, errCh, cancel, sighupCh := startRunRouterForReload(t, startCfg, badPath)
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(3 * time.Second):
		}
	})

	// Precondition: startup in E mode.
	if !scanForLine(buf, "mode=E", 2*time.Second) {
		t.Fatalf("AC-002 precondition: startup did not emit mode=E within 2s; got:\n%s", buf.String())
	}

	// F-006: capture cfg.UpstreamRouters (deep copy) before SIGHUP so we can
	// assert the original cfg pointer is not mutated by a fail-closed reload.
	origUpstreams := make([]config.UpstreamRouter, len(startCfg.UpstreamRouters))
	copy(origUpstreams, startCfg.UpstreamRouters)

	// Send reload signal with the bad config path already wired in.
	sighupCh <- syscall.SIGHUP

	// Postcondition 3: EC-004 log line must appear.
	// AC-002 specifies verbatim format: one output line matching
	//   "config reload failed: <err>; continuing with previous config"
	// where <err> is non-empty and contains "E-CFG-001" (the inner validation
	// error code from config.Validate on the bad config).
	if !scanForLine(buf, "config reload failed: ", 2*time.Second) {
		t.Fatalf("AC-002: after SIGHUP with bad config, output missing reload-failed prefix within 2s; got:\n%s",
			buf.String())
	}
	// Verify verbatim format: one line contains BOTH the prefix, E-CFG-001,
	// and the suffix — guaranteeing they are on the same output line.
	if !scanForReloadFailedLine(buf, "E-CFG-001", 100*time.Millisecond) {
		t.Errorf("AC-002: no single output line matches 'config reload failed: <E-CFG-001 err>; continuing with previous config'; got:\n%s",
			buf.String())
	}

	// F-006: cfg pointer must not be mutated by a fail-closed reload.
	// On failure paths the daemon must retain the previous config unchanged.
	if len(startCfg.UpstreamRouters) != len(origUpstreams) {
		t.Errorf("AC-002/F-006: cfg.UpstreamRouters length mutated: before=%d after=%d",
			len(origUpstreams), len(startCfg.UpstreamRouters))
	} else {
		for i, want := range origUpstreams {
			if startCfg.UpstreamRouters[i] != want {
				t.Errorf("AC-002/F-006: cfg.UpstreamRouters[%d] mutated: before=%v after=%v",
					i, want, startCfg.UpstreamRouters[i])
			}
		}
	}

	// Postcondition: no mode change occurred (AC-002 postcondition 4).
	// We already waited for the reload-failed line to appear above, so we
	// can proceed immediately — ordering-based rather than time-based.
	scanner := bufio.NewScanner(bytes.NewBufferString(buf.String()))
	modeLines := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "mode=") {
			modeLines++
		}
	}
	// Only one mode= line is expected: the initial mode=E startup emission.
	// A second mode= line would indicate a state change occurred (AC-002 violation).
	if modeLines > 1 {
		t.Errorf("AC-002: found %d mode= lines after bad-config SIGHUP; expected 1 (no state change); output:\n%s",
			modeLines, buf.String())
	}

	// Postcondition 5: daemon has not returned.
	select {
	case rErr := <-errCh:
		t.Errorf("AC-002: runRouter returned prematurely after bad-config SIGHUP: %v", rErr)
	default:
		// expected — still running
	}
}

// TestRunRouter_SIGHUPReload_LoadFileNotFound verifies the LoadFile error arm of
// the fail-closed reload path (F-P2-001 / BC-2.09.003 EC-004): when the config
// file is deleted between daemon startup and SIGHUP, LoadFile returns E-CFG-004,
// the daemon emits the EC-004 reload-failed log line with E-CFG-004 inside it,
// and continues running on the previous config.
func TestRunRouter_SIGHUPReload_LoadFileNotFound(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)

	startCfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}

	// Write a valid config, start the daemon, then DELETE the file to simulate
	// the operator removing it between startup and a SIGHUP.
	cfgPath := writeTempConfig(t, startCfg)

	buf, errCh, cancel, sighupCh := startRunRouterForReload(t, startCfg, cfgPath)
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(3 * time.Second):
		}
	})

	// Precondition: startup in E mode.
	if !scanForLine(buf, "mode=E", 2*time.Second) {
		t.Fatalf("LoadFileNotFound precondition: startup did not emit mode=E within 2s; got:\n%s", buf.String())
	}

	// Delete the config file so LoadFile will return E-CFG-004 on reload.
	if err := os.Remove(cfgPath); err != nil {
		t.Fatalf("LoadFileNotFound: remove config file: %v", err)
	}

	// Deep-copy cfg before SIGHUP to assert it is not mutated on this error path.
	origUpstreams := make([]config.UpstreamRouter, len(startCfg.UpstreamRouters))
	copy(origUpstreams, startCfg.UpstreamRouters)

	sighupCh <- syscall.SIGHUP

	// EC-004 log line must appear with E-CFG-004 inside it.
	if !scanForLine(buf, "config reload failed: ", 2*time.Second) {
		t.Fatalf("LoadFileNotFound: after SIGHUP, output missing reload-failed prefix within 2s; got:\n%s", buf.String())
	}
	if !scanForReloadFailedLine(buf, "E-CFG-004", 100*time.Millisecond) {
		t.Errorf("LoadFileNotFound: no single output line matches 'config reload failed: <E-CFG-004 err>; continuing with previous config'; got:\n%s",
			buf.String())
	}

	// cfg pointer must not be mutated on the LoadFile error path.
	if len(startCfg.UpstreamRouters) != len(origUpstreams) {
		t.Errorf("LoadFileNotFound: cfg.UpstreamRouters length mutated: before=%d after=%d",
			len(origUpstreams), len(startCfg.UpstreamRouters))
	}

	// No mode change: only the initial mode=E line should be present.
	scanner := bufio.NewScanner(bytes.NewBufferString(buf.String()))
	modeLines := 0
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "mode=") {
			modeLines++
		}
	}
	if modeLines > 1 {
		t.Errorf("LoadFileNotFound: found %d mode= lines; expected 1 (no state change); output:\n%s",
			modeLines, buf.String())
	}

	// Daemon has not returned.
	select {
	case rErr := <-errCh:
		t.Errorf("LoadFileNotFound: runRouter returned prematurely after not-found SIGHUP: %v", rErr)
	default:
		// expected — still running
	}
}

// TestRunRouter_SIGHUPReload_MalformedYAML verifies the LoadFile parse-error arm
// of the fail-closed reload path (F-P2-001 / BC-2.09.003 EC-004): when the config
// file is replaced with malformed YAML, LoadFile returns E-CFG-005, the daemon
// emits the EC-004 reload-failed log line with E-CFG-005 inside it, and continues
// running on the previous config.
func TestRunRouter_SIGHUPReload_MalformedYAML(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)

	startCfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}

	// Write a valid config first so the daemon starts cleanly, then overwrite
	// it with malformed YAML before the reload fires.
	cfgPath := writeTempConfig(t, startCfg)

	buf, errCh, cancel, sighupCh := startRunRouterForReload(t, startCfg, cfgPath)
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(3 * time.Second):
		}
	})

	// Precondition: startup in E mode.
	if !scanForLine(buf, "mode=E", 2*time.Second) {
		t.Fatalf("MalformedYAML precondition: startup did not emit mode=E within 2s; got:\n%s", buf.String())
	}

	// Replace the config file with unparseable YAML content.
	malformed := []byte(":\t:\t:\n  this is: [not valid yaml\n")
	if err := os.WriteFile(cfgPath, malformed, 0o600); err != nil {
		t.Fatalf("MalformedYAML: write malformed config: %v", err)
	}

	// Deep-copy cfg before SIGHUP to assert it is not mutated on this error path.
	origUpstreams := make([]config.UpstreamRouter, len(startCfg.UpstreamRouters))
	copy(origUpstreams, startCfg.UpstreamRouters)

	sighupCh <- syscall.SIGHUP

	// EC-004 log line must appear with E-CFG-005 inside it.
	if !scanForLine(buf, "config reload failed: ", 2*time.Second) {
		t.Fatalf("MalformedYAML: after SIGHUP, output missing reload-failed prefix within 2s; got:\n%s", buf.String())
	}
	if !scanForReloadFailedLine(buf, "E-CFG-005", 100*time.Millisecond) {
		t.Errorf("MalformedYAML: no single output line matches 'config reload failed: <E-CFG-005 err>; continuing with previous config'; got:\n%s",
			buf.String())
	}

	// cfg pointer must not be mutated on the parse-error path.
	if len(startCfg.UpstreamRouters) != len(origUpstreams) {
		t.Errorf("MalformedYAML: cfg.UpstreamRouters length mutated: before=%d after=%d",
			len(origUpstreams), len(startCfg.UpstreamRouters))
	}

	// No mode change: only the initial mode=E line should be present.
	scanner := bufio.NewScanner(bytes.NewBufferString(buf.String()))
	modeLines := 0
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "mode=") {
			modeLines++
		}
	}
	if modeLines > 1 {
		t.Errorf("MalformedYAML: found %d mode= lines; expected 1 (no state change); output:\n%s",
			modeLines, buf.String())
	}

	// Daemon has not returned.
	select {
	case rErr := <-errCh:
		t.Errorf("MalformedYAML: runRouter returned prematurely after malformed-YAML SIGHUP: %v", rErr)
	default:
		// expected — still running
	}
}

// ── AC-003 ─────────────────────────────────────────────────────────────────────

// TestRunRouter_SIGHUPReload_SessionsNotInterrupted verifies AC-003 /
// BC-2.09.001 PC-4: an open TCP connection to the ingress listener is NOT
// closed by a valid reload.
//
// RED GATE: fails because the reload path is not yet implemented; the mode=PE
// assertion will timeout first.
func TestRunRouter_SIGHUPReload_SessionsNotInterrupted(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)

	startCfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}

	// PE reload config.
	reloadAddr := probeDataAddr(t)
	reloadCfg := &config.Config{
		ListenAddr:   dataAddr,
		TickInterval: 10 * time.Millisecond,
		UpstreamRouters: []config.UpstreamRouter{
			{Addr: reloadAddr},
		},
		ManagementSocket: sockPath,
	}
	cfgPath := writeTempConfig(t, reloadCfg)

	buf, errCh, cancel, sighupCh := startRunRouterForReload(t, startCfg, cfgPath)
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(3 * time.Second):
		}
	})

	// Precondition: startup in E mode.
	if !scanForLine(buf, "mode=E", 2*time.Second) {
		t.Fatalf("AC-003 precondition: startup did not emit mode=E within 2s; got:\n%s", buf.String())
	}

	// Establish a live TCP connection to the ingress listener before sending SIGHUP.
	conn, err := net.Dial("tcp", dataAddr)
	if err != nil {
		t.Fatalf("AC-003: dial ingress listener: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	// Send reload signal.
	sighupCh <- syscall.SIGHUP

	// Wait for PE mode emission — confirms reload completed.
	if !scanForLine(buf, "mode=PE", 2*time.Second) {
		t.Errorf("AC-003: after SIGHUP, output missing mode=PE line within 2s; got:\n%s", buf.String())
	}

	// Postcondition 1: the open TCP connection must not have been closed by
	// the reload path.
	//
	// F-P2-002 divergence from story outline: the story's AC-003 outline
	// suggests "write a byte" as the liveness probe, but a write to a FIN'd
	// socket typically succeeds into the send buffer — the kernel reports the
	// RST only on the NEXT write or on a read.  A read-with-deadline is
	// strictly stronger: EOF or connection-reset on a read means the daemon
	// closed its side (FAIL); a read deadline timeout means the socket is
	// still open (PASS).  See ## Ruling Divergence in DELIVERY doc.
	_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	oneByte := make([]byte, 1)
	_, readErr := conn.Read(oneByte)
	if readErr != nil {
		var netErr net.Error
		if ok := isNetError(readErr, &netErr); ok && netErr.Timeout() {
			// Deadline timeout — connection is still open; daemon did not close it.
		} else {
			// EOF or connection-reset — daemon closed its side during reload.
			t.Errorf("AC-003: TCP connection closed by daemon during reload — read error: %v", readErr)
		}
	}
}

// isNetError attempts a type assertion to net.Error and writes the result into
// dst.  Returns true on success.
func isNetError(err error, dst *net.Error) bool {
	if ne, ok := err.(net.Error); ok { //nolint:errorlint // asserting concrete interface for timeout check
		*dst = ne
		return true
	}
	return false
}

// ── F-005: PE→E and PE→PE′ transition tests ────────────────────────────────

// TestRunRouter_SIGHUPReload_PEtoE verifies that a router starting in PE mode
// (upstream_routers=[A]) downgrades to E mode when reloaded with a config
// that has no upstream_routers.
//
// Pins mgmt_wire.go:562-564 behavior (adversary pass-1 F-005).
func TestRunRouter_SIGHUPReload_PEtoE(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	upstreamA := probeDataAddr(t)

	// Start in PE mode (upstream_routers=[A]).
	startCfg := &config.Config{
		ListenAddr:   dataAddr,
		TickInterval: 10 * time.Millisecond,
		UpstreamRouters: []config.UpstreamRouter{
			{Addr: upstreamA},
		},
		ManagementSocket: sockPath,
	}

	// Reload config: no upstream_routers → should drop to E mode.
	reloadCfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}
	cfgPath := writeTempConfig(t, reloadCfg)

	buf, errCh, cancel, sighupCh := startRunRouterForReload(t, startCfg, cfgPath)
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(3 * time.Second):
		}
	})

	// Precondition: startup in PE mode.
	if !scanForLine(buf, "mode=PE", 2*time.Second) {
		t.Fatalf("F-005/PE→E precondition: startup did not emit mode=PE within 2s; got:\n%s", buf.String())
	}

	// Send reload with the E-mode config.
	sighupCh <- syscall.SIGHUP

	// Postcondition 1: mode=E line must appear after reload.
	if !scanForLine(buf, "mode=E", 2*time.Second) {
		t.Errorf("F-005/PE→E: after SIGHUP with no-upstream config, output missing mode=E within 2s; got:\n%s",
			buf.String())
	}

	// Postcondition 2: no spurious "mode=PE upstream_routers=[]" line —
	// the daemon must not emit a PE line with an empty upstream list.
	snapshot := buf.String()
	scanner := bufio.NewScanner(bytes.NewBufferString(snapshot))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "mode=PE") && strings.Contains(line, "upstream_routers=[]") {
			t.Errorf("F-005/PE→E: daemon emitted 'mode=PE upstream_routers=[]' — should have emitted mode=E; line: %q", line)
		}
	}

	// Postcondition 3: daemon still running.
	select {
	case rErr := <-errCh:
		t.Errorf("F-005/PE→E: runRouter returned prematurely after reload: %v", rErr)
	default:
		// expected — still running
	}
}

// TestRunRouter_SIGHUPReload_PEtoPE verifies that a router in PE mode with
// upstream_routers=[A] transitions to PE mode with [B] when reloaded with a
// config that replaces [A] with [B].
//
// Pins mgmt_wire.go:560-561 behavior (adversary pass-1 F-005).
func TestRunRouter_SIGHUPReload_PEtoPE(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	upstreamA := probeDataAddr(t)
	upstreamB := probeDataAddr(t)

	// Start in PE mode with upstream_routers=[A].
	startCfg := &config.Config{
		ListenAddr:   dataAddr,
		TickInterval: 10 * time.Millisecond,
		UpstreamRouters: []config.UpstreamRouter{
			{Addr: upstreamA},
		},
		ManagementSocket: sockPath,
	}

	// Reload config: replace [A] with [B].
	reloadCfg := &config.Config{
		ListenAddr:   dataAddr,
		TickInterval: 10 * time.Millisecond,
		UpstreamRouters: []config.UpstreamRouter{
			{Addr: upstreamB},
		},
		ManagementSocket: sockPath,
	}
	cfgPath := writeTempConfig(t, reloadCfg)

	buf, errCh, cancel, sighupCh := startRunRouterForReload(t, startCfg, cfgPath)
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(3 * time.Second):
		}
	})

	// Precondition: startup in PE mode with [A].
	if !scanForLine(buf, "mode=PE", 2*time.Second) {
		t.Fatalf("F-005/PE→PE′ precondition: startup did not emit mode=PE within 2s; got:\n%s", buf.String())
	}
	if !scanForLine(buf, upstreamA, 2*time.Second) {
		t.Fatalf("F-005/PE→PE′ precondition: startup did not emit upstream addr %q within 2s; got:\n%s",
			upstreamA, buf.String())
	}

	// Send reload with the PE-[B] config.
	sighupCh <- syscall.SIGHUP

	// Postcondition 1: new mode=PE line with [B] must appear.
	if !scanForLine(buf, "mode=PE", 2*time.Second) {
		t.Errorf("F-005/PE→PE′: after SIGHUP, output missing mode=PE within 2s; got:\n%s", buf.String())
	}
	if !scanForLine(buf, upstreamB, 2*time.Second) {
		t.Errorf("F-005/PE→PE′: after SIGHUP, output missing new upstream addr %q within 2s; got:\n%s",
			upstreamB, buf.String())
	}

	// Postcondition 2: daemon still running.
	select {
	case rErr := <-errCh:
		t.Errorf("F-005/PE→PE′: runRouter returned prematurely after reload: %v", rErr)
	default:
		// expected — still running
	}
}

// ── AC-004 ─────────────────────────────────────────────────────────────────────

// TestRunRouter_VP038_EtoPEViaConfigOnly verifies VP-038: the router graduates
// from E to PE mode via in-process sighupCh injection using a
// testenv.RouterHandle, without process restart.
//
// RED GATE: fails because (a) the reload logic is a no-op stub so Mode() never
// transitions to ModePE, and (b) the mode=PE line is never emitted.
func TestRunRouter_VP038_EtoPEViaConfigOnly(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)

	// Start in E mode.
	startCfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}

	// Write a PE config to disk.
	reloadAddr := probeDataAddr(t)
	reloadCfg := &config.Config{
		ListenAddr:   dataAddr,
		TickInterval: 10 * time.Millisecond,
		UpstreamRouters: []config.UpstreamRouter{
			{Addr: reloadAddr},
		},
		ManagementSocket: sockPath,
	}
	cfgPath := writeTempConfig(t, reloadCfg)

	// Start the real runRouter goroutine and obtain the sighupCh write end.
	buf, errCh, cancel, rawSighupCh := startRunRouterForReload(t, startCfg, cfgPath)
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(3 * time.Second):
		}
	})

	// Construct a RouterHandle wired to the real sighupCh so that
	// SendReloadSignal drives the same channel runRouter selects on.
	ctx := context.Background()
	env := testenv.New(t, ctx)
	handle := env.StartRouter(t, testenv.RouterConfig{})
	// Wire the sighupCh onto the handle for the signal-path assertion.
	handle.SetSighupCh(rawSighupCh)

	// Precondition: E mode startup.
	if !scanForLine(buf, "mode=E", 2*time.Second) {
		t.Fatalf("VP-038 precondition: startup did not emit mode=E within 2s; got:\n%s", buf.String())
	}

	// Drive reload via the testenv seam.
	handle.SendReloadSignal(t)

	// AC-004 postcondition 1+2: mode=PE emitted without process restart.
	// scanForLine carries the postcondition — the observed emission on the
	// real runRouter output buffer IS the authoritative mode assertion.
	// handle.Mode() is NOT asserted here: Restart() unconditionally sets
	// r.mode=ModePE regardless of the real runRouter state (the stub handle
	// is disconnected from the goroutine), making it a tautological check
	// that cannot fail (adversary pass-1 F-002).
	if !scanForLine(buf, "mode=PE", 2*time.Second) {
		t.Errorf("VP-038/AC-004: after SendReloadSignal, output missing mode=PE line within 2s; got:\n%s", buf.String())
	}

	// AC-004 postcondition 3: goroutine has not returned.
	select {
	case rErr := <-errCh:
		t.Errorf("VP-038/AC-004: runRouter returned prematurely after reload: %v", rErr)
	default:
		// expected — still running
	}
}
