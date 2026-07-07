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
	"encoding/json"
	"fmt"
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
func scanForReloadFailedLine(buf *syncBuffer, innerMarker string, budget time.Duration) bool { //nolint:unparam // budget is a caller-controlled settle window; all current callers use 100ms but the parameter is intentional
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

// modeELine returns the exact mode=E output line emitted by runRouter for
// zero upstream_routers — derived from mgmt_wire.go:527/566 format strings
// (527 startup emission, 566 reload emission).
// Pinning the full string (prefix + token) guards against partial-match
// false-positives and format drift (F-SIGHUP-P4-001).
func modeELine() string {
	return "switchboard router: mode=E (no upstream_routers configured)"
}

// modePELine returns the exact mode=PE output line emitted by runRouter for
// a given upstreamRouters slice — derived from mgmt_wire.go:529/564 format
// string ("switchboard router: mode=PE upstream_routers=%v\n" where %v is
// fmt.Sprintf("%v", addrs)).
// Pinning the full prefix + upstream_routers= token guards against partial-
// match false-positives and format drift (F-SIGHUP-P4-001).
func modePELine(addrs []string) string {
	return fmt.Sprintf("switchboard router: mode=PE upstream_routers=%v", addrs)
}

// scanForExactModeLine polls buf until a line that contains the full mode-line
// string (as returned by modeELine or modePELine) appears, or the budget
// elapses.  The match is line-scoped via a bufio.Scanner to ensure prefix and
// suffix appear on the same output line.
//
// Using contains-on-a-line (rather than full-line equality) tolerates a
// trailing newline that the scanner strips, while still requiring the complete
// prefix "switchboard router: " and the complete token on one line.
//
// F-SIGHUP-P4-001: replaces loose "mode=PE" / "mode=E" substring checks in
// AC-001 so that the full production format is pinned to the test.
func scanForExactModeLine(buf *syncBuffer, fullLine string, budget time.Duration) bool { //nolint:unparam // budget is a caller-controlled knob; all current callers use 2s but the parameter is intentional
	check := func() bool {
		scanner := bufio.NewScanner(bytes.NewBufferString(buf.String()))
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), fullLine) {
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

// dialMgmtAndReadChallenge dials the management Unix socket at sockPath and
// reads the first JSON message from the server.  Returns the decoded map and
// nil error if a well-formed JSON object is received within the read deadline.
// Used by F-SIGHUP-P4-004 (AC-003 post-reload mgmt-socket probe).
func dialMgmtAndReadChallenge(t *testing.T, sockPath string) (map[string]any, error) {
	t.Helper()
	conn, err := net.DialTimeout("unix", sockPath, time.Second)
	if err != nil {
		return nil, fmt.Errorf("dial mgmt socket %q: %w", sockPath, err)
	}
	defer func() { _ = conn.Close() }()
	if err := conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		return nil, fmt.Errorf("SetReadDeadline: %w", err)
	}
	var msg map[string]any
	if err := json.NewDecoder(conn).Decode(&msg); err != nil {
		return nil, fmt.Errorf("decode JSON from mgmt socket: %w", err)
	}
	return msg, nil
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
	// F-SIGHUP-P4-001: use full-line format (prefix + token), not loose "mode=E".
	if !scanForExactModeLine(buf, modeELine(), 2*time.Second) {
		t.Fatalf("AC-001 precondition: startup did not emit exact mode=E line %q within 2s; got:\n%s",
			modeELine(), buf.String())
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
	// F-SIGHUP-P4-001: pin the full production format including "switchboard router: "
	// prefix and "upstream_routers=" token — not a loose "mode=PE" substring.
	wantPELine := modePELine([]string{reloadAddr})
	if !scanForExactModeLine(buf, wantPELine, 2*time.Second) {
		t.Errorf("AC-001: after SIGHUP, output missing exact mode=PE line %q within 2s; got:\n%s",
			wantPELine, buf.String())
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

	// F-P5-002: AC-002 PC-6 — establish a live TCP connection to the ingress
	// listener BEFORE the bad reload fires.  After the fail-closed reload, the
	// connection must still be open (symmetric with AC-003 which probes the
	// successful reload path).
	ingressConn, dialErr := net.Dial("tcp", dataAddr)
	if dialErr != nil {
		t.Fatalf("AC-002/F-P5-002: dial ingress listener before SIGHUP: %v", dialErr)
	}
	t.Cleanup(func() { _ = ingressConn.Close() })

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

	// F-P5-002: AC-002 PC-6 — ingress listener and mgmt server remain serving
	// after a failed reload.  These probes are symmetric with AC-003's
	// postcondition 1 + F-P4-004 assertions on the successful reload path.

	// PC-6 probe 1: the accepted TCP connection must still be open after the
	// fail-closed reload.  A read-with-deadline timeout (no data) means the
	// connection is alive; EOF / connection-reset means the daemon closed it.
	_ = ingressConn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	oneByte := make([]byte, 1)
	_, readErr := ingressConn.Read(oneByte)
	if readErr != nil {
		var netErr net.Error
		if ok := isNetError(readErr, &netErr); ok && netErr.Timeout() {
			// Deadline timeout — connection still open; daemon did not close it.
		} else {
			// EOF or connection-reset — daemon incorrectly closed the connection
			// during a fail-closed reload (AC-002 PC-6 violation).
			t.Errorf("AC-002/F-P5-002: TCP ingress connection closed by daemon during fail-closed reload — read error: %v", readErr)
		}
	}

	// PC-6 probe 2: management server must still be reachable after the failed
	// reload.  Dial the mgmt socket and assert a well-formed challenge arrives.
	challenge, mgmtErr := dialMgmtAndReadChallenge(t, sockPath)
	if mgmtErr != nil {
		t.Errorf("AC-002/F-P5-002: mgmt socket not serving after fail-closed reload — %v", mgmtErr)
	} else {
		if challenge["type"] != "challenge" {
			t.Errorf("AC-002/F-P5-002: post-fail-reload mgmt response type = %v; want %q",
				challenge["type"], "challenge")
		}
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
	// F-SIGHUP-P4-001: pin exact line format.
	if !scanForExactModeLine(buf, modeELine(), 2*time.Second) {
		t.Fatalf("AC-003 precondition: startup did not emit exact mode=E line %q within 2s; got:\n%s",
			modeELine(), buf.String())
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
	// F-SIGHUP-P4-001: pin exact line format including upstream_routers= token.
	wantPELine := modePELine([]string{reloadAddr})
	if !scanForExactModeLine(buf, wantPELine, 2*time.Second) {
		t.Errorf("AC-003: after SIGHUP, output missing exact mode=PE line %q within 2s; got:\n%s",
			wantPELine, buf.String())
	}

	// Postcondition 1: the open TCP connection must not have been closed by
	// the reload path.
	//
	// Divergence from story AC-003 outline (deliberate: read-probe is strictly
	// stronger): the story's AC-003 outline suggests "write a byte" as the
	// liveness probe, but a write to a FIN'd socket typically succeeds into
	// the send buffer — the kernel reports the RST only on the NEXT write or
	// on a read.  A read-with-deadline is strictly stronger: EOF or
	// connection-reset on a read means the daemon closed its side (FAIL); a
	// read deadline timeout means the socket is still open (PASS).
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

	// F-SIGHUP-P4-004: AC-003 PC-2 names "mgmtSrv/mgmtWG — management server
	// continues serving" among untouched surfaces.  Probe the management socket
	// post-reload: dial it and assert a well-formed challenge is received,
	// confirming mgmt.Server is still alive after the reload cycle.
	challenge, mgmtErr := dialMgmtAndReadChallenge(t, sockPath)
	if mgmtErr != nil {
		t.Errorf("AC-003/F-P4-004: mgmt socket not serving after reload — %v", mgmtErr)
	} else {
		if challenge["type"] != "challenge" {
			t.Errorf("AC-003/F-P4-004: post-reload mgmt response type = %v; want %q",
				challenge["type"], "challenge")
		}
	}

	// Postcondition: daemon has not returned.
	select {
	case rErr := <-errCh:
		t.Errorf("AC-003: runRouter returned prematurely after SIGHUP: %v", rErr)
	default:
		// expected — still running
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
	// F-SIGHUP-P4-001: pin exact line format.
	wantStartPELine := modePELine([]string{upstreamA})
	if !scanForExactModeLine(buf, wantStartPELine, 2*time.Second) {
		t.Fatalf("F-005/PE→E precondition: startup did not emit exact mode=PE line %q within 2s; got:\n%s",
			wantStartPELine, buf.String())
	}

	// F-P8-001: deep-copy cfg.UpstreamRouters (contains upstream A) before the
	// SIGHUP that reloads to empty; assert cfg is not mutated on a successful reload
	// (AC-001 PC-6 — the cfg parameter is immutable throughout runRouter's lifetime).
	origUpstreamsPEtoE := make([]config.UpstreamRouter, len(startCfg.UpstreamRouters))
	copy(origUpstreamsPEtoE, startCfg.UpstreamRouters)

	// Send reload with the E-mode config.
	sighupCh <- syscall.SIGHUP

	// Postcondition 1: mode=E line must appear after reload.
	// F-SIGHUP-P4-001: pin exact line format.
	if !scanForExactModeLine(buf, modeELine(), 2*time.Second) {
		t.Errorf("F-005/PE→E: after SIGHUP with no-upstream config, output missing exact mode=E line %q within 2s; got:\n%s",
			modeELine(), buf.String())
	}

	// F-P8-001: cfg pointer must not be mutated by the successful PE→E reload.
	// cfg.UpstreamRouters must still contain upstream A (not empty).
	if len(startCfg.UpstreamRouters) != len(origUpstreamsPEtoE) {
		t.Errorf("F-P8-001/PE→E: cfg.UpstreamRouters length mutated on successful reload: before=%d after=%d",
			len(origUpstreamsPEtoE), len(startCfg.UpstreamRouters))
	} else {
		for i, want := range origUpstreamsPEtoE {
			if startCfg.UpstreamRouters[i] != want {
				t.Errorf("F-P8-001/PE→E: cfg.UpstreamRouters[%d] mutated on successful reload: before=%v after=%v",
					i, want, startCfg.UpstreamRouters[i])
			}
		}
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
	// F-SIGHUP-P4-001: pin exact line format including upstream_routers= token.
	wantStartPELineA := modePELine([]string{upstreamA})
	if !scanForExactModeLine(buf, wantStartPELineA, 2*time.Second) {
		t.Fatalf("F-005/PE→PE′ precondition: startup did not emit exact mode=PE line %q within 2s; got:\n%s",
			wantStartPELineA, buf.String())
	}

	// F-P8-001: deep-copy cfg.UpstreamRouters (contains upstream A) before the
	// SIGHUP that reloads to B; assert cfg is not mutated on a successful reload
	// (AC-001 PC-6 — the cfg parameter is immutable; runRouter operates on the
	// freshly loaded struct, not on the startup-time cfg pointer).
	origUpstreamsPE := make([]config.UpstreamRouter, len(startCfg.UpstreamRouters))
	copy(origUpstreamsPE, startCfg.UpstreamRouters)

	// Send reload with the PE-[B] config.
	sighupCh <- syscall.SIGHUP

	// Postcondition 1: new mode=PE line with [B] must appear.
	// F-SIGHUP-P4-001: pin exact line format including new upstream_routers= token.
	wantReloadPELineB := modePELine([]string{upstreamB})
	if !scanForExactModeLine(buf, wantReloadPELineB, 2*time.Second) {
		t.Errorf("F-005/PE→PE′: after SIGHUP, output missing exact mode=PE line %q within 2s; got:\n%s",
			wantReloadPELineB, buf.String())
	}

	// F-P8-001: cfg pointer must not be mutated by the successful PE→PE′ reload.
	// cfg.UpstreamRouters must still contain upstream A (not B, not empty).
	if len(startCfg.UpstreamRouters) != len(origUpstreamsPE) {
		t.Errorf("F-P8-001/PE→PE′: cfg.UpstreamRouters length mutated on successful reload: before=%d after=%d",
			len(origUpstreamsPE), len(startCfg.UpstreamRouters))
	} else {
		for i, want := range origUpstreamsPE {
			if startCfg.UpstreamRouters[i] != want {
				t.Errorf("F-P8-001/PE→PE′: cfg.UpstreamRouters[%d] mutated on successful reload: before=%v after=%v",
					i, want, startCfg.UpstreamRouters[i])
			}
		}
	}

	// Postcondition 2: daemon still running.
	select {
	case rErr := <-errCh:
		t.Errorf("F-005/PE→PE′: runRouter returned prematurely after reload: %v", rErr)
	default:
		// expected — still running
	}
}

// TestRunRouter_SIGHUPReload_IdempotentResend verifies that the reload
// diff-guard (equalStringSlices in mgmt_wire.go) suppresses duplicate mode=
// emission when a second SIGHUP is sent with an unchanged config.
//
// Pins mgmt_wire.go:560 behavior (adversary pass-3 F-P3-005a).
func TestRunRouter_SIGHUPReload_IdempotentResend(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)
	upstreamA := probeDataAddr(t)

	// Start in E mode.
	startCfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}

	// Reload config: PE mode with upstream_routers=[A].
	reloadCfg := &config.Config{
		ListenAddr:   dataAddr,
		TickInterval: 10 * time.Millisecond,
		UpstreamRouters: []config.UpstreamRouter{
			{Addr: upstreamA},
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
		t.Fatalf("IdempotentResend precondition: startup did not emit mode=E within 2s; got:\n%s", buf.String())
	}

	// First reload: E→PE; a mode=PE line must appear.
	sighupCh <- syscall.SIGHUP
	if !scanForLine(buf, "mode=PE", 2*time.Second) {
		t.Fatalf("IdempotentResend: first SIGHUP did not produce mode=PE within 2s; got:\n%s", buf.String())
	}

	// Snapshot the mode= count after the first reload settles.
	countModeLines := func(s string) int {
		scanner := bufio.NewScanner(bytes.NewBufferString(s))
		n := 0
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "mode=") {
				n++
			}
		}
		return n
	}
	countAfterFirst := countModeLines(buf.String())

	// Second reload: same config file on disk — diff-guard must suppress emission.
	sighupCh <- syscall.SIGHUP

	// The fail-closed path emits nothing on an identical config, so we wait a
	// short bounded interval for any spurious emission to materialise before
	// asserting absence.  Mirror the same 200 ms budget used by AC-003's
	// read-deadline settle.
	time.Sleep(200 * time.Millisecond)

	countAfterSecond := countModeLines(buf.String())
	if countAfterSecond != countAfterFirst {
		t.Errorf("IdempotentResend: second SIGHUP with unchanged config emitted %d additional mode= line(s); expected 0; output:\n%s",
			countAfterSecond-countAfterFirst, buf.String())
	}

	// Daemon must still be running.
	select {
	case rErr := <-errCh:
		t.Errorf("IdempotentResend: runRouter returned prematurely after second reload: %v", rErr)
	default:
		// expected — still running
	}
}

// ── AC-004 ─────────────────────────────────────────────────────────────────────

// TestRunRouter_SIGHUPReload_InvalidUpstreamAddr_FailClosed verifies the
// BC-2.09.001 EC-003 input class: a config whose upstream_routers[N].addr is
// malformed (not a valid host:port) fails Validate with E-CFG-001 (Validate
// wraps all field failures under E-CFG-001, including upstream addr errors —
// NOT E-CFG-003, which is the per-field code; see internal/config/config.go).
// The daemon must emit the EC-004 reload-failed line and continue on the previous
// config unchanged (fail-closed per BC-2.09.001 PC-1).
//
// Cross-BC Note: EC-003 vs E-CFG-001/E-CFG-003 Rendering Nuance
// E-CFG-003 is the per-field upstream_routers addr error code documented in
// BC-2.09.003; however, (*Config).Validate collects all field failures into a
// single *ConfigError{Code: "E-CFG-001", Detail: ...} that concatenates every
// ValidationError string. The reload-failed log line therefore always carries
// "E-CFG-001" as the outer code, with the upstream_routers[0].addr detail nested
// inside. This test pins the CURRENT production rendering; the literal question
// of whether to surface E-CFG-003 at the outer level is parked at
// S-7.04-FU-PE-CONNECTOR.
//
// F-P8-002: addresses adversary pass-8 finding F-SIGHUP-P8-002 (no reload test
// driving Validate failure via invalid upstream_routers[N].addr).
func TestRunRouter_SIGHUPReload_InvalidUpstreamAddr_FailClosed(t *testing.T) {
	// NOT t.Parallel(): binds ephemeral TCP + filesystem socket.

	dataAddr := probeDataAddr(t)
	sockPath := tempSockPath(t)

	// Start in E mode (no upstream_routers).
	startCfg := &config.Config{
		ListenAddr:       dataAddr,
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}

	// Build a config whose upstream_routers[0].addr is malformed — "notaport"
	// has no colon, so net.SplitHostPort returns an error → Validate returns
	// E-CFG-001 with a detail containing "upstream_routers[0].addr".
	invalidUpstreamCfg := &config.Config{
		ListenAddr:   dataAddr,
		TickInterval: 10 * time.Millisecond,
		UpstreamRouters: []config.UpstreamRouter{
			{Addr: "notaport"},
		},
		ManagementSocket: sockPath,
	}
	cfgPath := writeTempConfig(t, invalidUpstreamCfg)

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
		t.Fatalf("F-P8-002 precondition: startup did not emit mode=E within 2s; got:\n%s", buf.String())
	}

	// Deep-copy cfg before SIGHUP to assert it is not mutated by the fail-closed
	// reload (the E-mode startCfg has an empty UpstreamRouters slice — that's fine;
	// the load-bearing assertion is the fail-closed log line below).
	origUpstreams := make([]config.UpstreamRouter, len(startCfg.UpstreamRouters))
	copy(origUpstreams, startCfg.UpstreamRouters)

	// Send reload with the invalid-upstream config.
	sighupCh <- syscall.SIGHUP

	// EC-004 log line must appear with E-CFG-001 as the outer error code.
	// The format is one line: "config reload failed: E-CFG-001: <detail>; continuing with previous config".
	if !scanForLine(buf, "config reload failed: ", 2*time.Second) {
		t.Fatalf("F-P8-002: after SIGHUP with invalid upstream addr, output missing reload-failed prefix within 2s; got:\n%s",
			buf.String())
	}
	// Verify E-CFG-001 appears on the reload-failed line (outer code from Validate).
	if !scanForReloadFailedLine(buf, "E-CFG-001", 100*time.Millisecond) {
		t.Errorf("F-P8-002: no single output line matches 'config reload failed: <E-CFG-001 err>; continuing with previous config'; got:\n%s",
			buf.String())
	}
	// Verify the detail names upstream_routers[0].addr — pins that EC-003 input
	// class is validated by Validate (not silently swallowed) even though the outer
	// code is E-CFG-001.
	if !scanForReloadFailedLine(buf, "upstream_routers[0].addr", 100*time.Millisecond) {
		t.Errorf("F-P8-002: reload-failed line does not name upstream_routers[0].addr in the E-CFG-001 detail; got:\n%s",
			buf.String())
	}

	// cfg pointer must not be mutated on the Validate-error path.
	if len(startCfg.UpstreamRouters) != len(origUpstreams) {
		t.Errorf("F-P8-002: cfg.UpstreamRouters length mutated by fail-closed reload: before=%d after=%d",
			len(origUpstreams), len(startCfg.UpstreamRouters))
	}

	// No mode change: only the initial mode=E line should be present.
	snapshot := buf.String()
	scanner := bufio.NewScanner(bytes.NewBufferString(snapshot))
	modeLines := 0
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "mode=") {
			modeLines++
		}
	}
	if modeLines > 1 {
		t.Errorf("F-P8-002: found %d mode= lines after invalid-upstream-addr SIGHUP; expected 1 (no state change); output:\n%s",
			modeLines, snapshot)
	}

	// Daemon must still be running.
	select {
	case rErr := <-errCh:
		t.Errorf("F-P8-002: runRouter returned prematurely after invalid-upstream-addr SIGHUP: %v", rErr)
	default:
		// expected — still running
	}
}

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
