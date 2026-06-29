// Tests for cmd/sbctl main() integration: connection-refused exit codes,
// auth-failure exit codes, no stdout on failure, and connection timeout.
//
// Tests are named per BC-based convention (BC-2.07.002, BC-2.07.003, VP-030)
// for full traceability.
//
// RED GATE STRATEGY (BC-5.38.001):
// Each test provides a valid testdata Ed25519 key so that loadEd25519Key
// succeeds. With the stubs returning "not implemented", the subprocess exits
// early at key loading (E-NET-001, not E-ADM-010) — causing the auth-failure
// test to fail. For the connection tests, tests also assert that the error
// message contains the specific target address, which the key-load error
// message will not contain.
//
// Package main (internal test file) for access to connectAndRun and dialTarget.
package main

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// subprocessGuard is the env variable used to gate subprocess test helpers.
const subprocessGuard = "SBCTL_TEST_SUBPROCESS_CASE"

// testdataKeyPath returns the absolute path to the testdata Ed25519 key fixture.
// Using an absolute path ensures the subprocess can locate it regardless of cwd.
func testdataKeyPath(t *testing.T) string {
	t.Helper()
	// Resolve relative to the current working directory of the test binary.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	return filepath.Join(wd, "testdata", "test_ed25519_key")
}

// subprocessEntrypoint is called at the TOP of TestSubprocessEntrypoint if the
// subprocess guard is set. It runs the specific test scenario and calls
// os.Exit — so it never returns.
func subprocessEntrypoint() {
	testCase := os.Getenv(subprocessGuard)
	if testCase == "" {
		return // not a subprocess — continue normally
	}

	target := os.Getenv("SBCTL_TEST_TARGET")
	keyPath := os.Getenv("SBCTL_TEST_KEY")
	timeoutStr := os.Getenv("SBCTL_TEST_TIMEOUT")
	to := 200 * time.Millisecond
	if timeoutStr != "" {
		if d, err := time.ParseDuration(timeoutStr); err == nil {
			to = d
		}
	}

	switch testCase {
	case "ConnectionRefused":
		connectAndRun(target, keyPath, false, to, "ping", nil)
	case "AuthFailure":
		connectAndRun(target, keyPath, false, to, "ping", nil)
	case "NoStdoutOnConnectionFailure":
		connectAndRun(target, keyPath, false, to, "ping", nil)
	case "ConnectionTimeout":
		connectAndRun(target, keyPath, false, to, "ping", nil)
	default:
		fmt.Fprintf(os.Stderr, "unknown subprocess test case: %s\n", testCase)
		os.Exit(3)
	}
}

// runSubprocess executes this test binary as a subprocess with the given env
// guard value and extra env vars. It returns the exit code, stdout, and
// stderr captured from the subprocess.
func runSubprocess(t *testing.T, testCase string, extraEnv ...string) (exitCode int, stdout, stderr string) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=TestSubprocessEntrypoint")
	cmd.Env = append(
		os.Environ(),
		subprocessGuard+"="+testCase,
	)
	cmd.Env = append(cmd.Env, extraEnv...)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()

	if err == nil {
		return 0, stdout, stderr
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("subprocess execution failed with non-exit error: %v", err)
	}
	return exitErr.ExitCode(), stdout, stderr
}

// TestSubprocessEntrypoint is the hook that re-exec'd subprocesses land in.
// It calls subprocessEntrypoint(), which either handles the case and exits, or
// returns immediately (parent process) and the test is skipped.
func TestSubprocessEntrypoint(t *testing.T) {
	subprocessEntrypoint()
	t.Skip("subprocess entrypoint — no-op in parent process")
}

// TestSbctl_ConnectionRefused_ExitsOneWithENET001_VP030 verifies VP-030 and
// BC-2.07.003 PC-1 / PC-2:
// When sbctl cannot connect to the daemon, it exits with code 1 and stderr
// contains both "E-NET-001" and the target address.
//
// RED GATE: with the stub, loadEd25519Key returns "not implemented" and exits
// with "E-NET-001 key load failed..." — the address assertion FAILS because
// the stub's error message does not contain the target address.
//
// BC: BC-2.07.003 PC-1 (E-NET-001 on stderr), PC-2 (exit 1); VP-030.
func TestSbctl_ConnectionRefused_ExitsOneWithENET001_VP030(t *testing.T) {
	t.Parallel()

	// Use an address that nothing is listening on.
	target := "127.0.0.1:19998"
	keyPath := testdataKeyPath(t)

	exitCode, _, stderr := runSubprocess(t, "ConnectionRefused",
		"SBCTL_TEST_TARGET="+target,
		"SBCTL_TEST_KEY="+keyPath,
		"SBCTL_TEST_TIMEOUT=200ms",
	)

	if exitCode != 1 {
		t.Errorf("VP-030 violated: expected exit code 1, got %d\nstderr: %s", exitCode, stderr)
	}
	if !strings.Contains(stderr, "E-NET-001") {
		t.Errorf("VP-030 violated: expected stderr to contain 'E-NET-001'; got: %q", stderr)
	}
	// The error message MUST include the target address (BC-2.07.003 PC-1:
	// "daemon unreachable: <address>: <reason>").
	if !strings.Contains(stderr, target) {
		t.Errorf("VP-030 violated: expected stderr to contain target address %q; got: %q", target, stderr)
	}
}

// TestSbctl_AuthFailure_ExitsOneWithEADM010 verifies BC-2.07.002 PC-4 and AC-003:
// When the daemon sends AUTH_FAIL, sbctl exits with code 1 and stderr contains
// "E-ADM-010". No stdout output is produced.
//
// RED GATE: with the stub, loadEd25519Key returns "not implemented" so
// connectAndRun exits with E-NET-001 (key load failure) — NOT E-ADM-010.
// The assertion that stderr contains "E-ADM-010" therefore FAILS.
//
// BC: BC-2.07.002 PC-4; AC-003.
func TestSbctl_AuthFailure_ExitsOneWithEADM010(t *testing.T) {
	t.Parallel()

	// Start a mock server that accepts one connection, sends a valid CHALLENGE,
	// reads the CHALLENGE_RESPONSE, then sends AUTH_FAIL.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	target := ln.Addr().String()
	keyPath := testdataKeyPath(t)

	// Serve: send CHALLENGE, read CHALLENGE_RESPONSE, send AUTH_FAIL.
	// acceptErrCh allows detecting server setup problems.
	acceptErrCh := make(chan error, 1)
	go func() {
		// Short accept deadline: the Red Gate subprocess exits fast (key loading
		// fails), so if nothing connects within 1s, the server gives up.
		_ = ln.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second))
		conn, err := ln.Accept()
		if err != nil {
			acceptErrCh <- fmt.Errorf("accept: %w", err)
			return
		}
		acceptErrCh <- nil
		defer func() { _ = conn.Close() }()

		_ = conn.SetDeadline(time.Now().Add(3 * time.Second))

		// Send a well-formed CHALLENGE.
		challenge := `{"type":"challenge","nonce":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","daemon_sig":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}` + "\n"
		if _, err := conn.Write([]byte(challenge)); err != nil {
			return
		}
		// Read the CHALLENGE_RESPONSE.
		buf := make([]byte, 4096)
		_, _ = conn.Read(buf)
		// Send AUTH_FAIL.
		authFail := `{"type":"auth_fail","code":"E-ADM-010","message":"authentication failed"}` + "\n"
		_, _ = conn.Write([]byte(authFail))
	}()

	exitCode, stdout, stderr := runSubprocess(t, "AuthFailure",
		"SBCTL_TEST_TARGET="+target,
		"SBCTL_TEST_KEY="+keyPath,
		"SBCTL_TEST_TIMEOUT=3s",
	)

	// Wait for the server goroutine accept result.
	if serverErr := <-acceptErrCh; serverErr != nil {
		t.Logf("mock server accept error (subprocess may have failed before connecting): %v", serverErr)
	}

	if exitCode != 1 {
		t.Errorf("AC-003 violated: expected exit code 1, got %d\nstderr: %s", exitCode, stderr)
	}
	if !strings.Contains(stderr, "E-ADM-010") {
		t.Errorf("AC-003 violated: expected stderr to contain 'E-ADM-010'; got: %q", stderr)
	}
	if stdout != "" {
		t.Errorf("AC-003 violated: expected no stdout output on auth failure; got: %q", stdout)
	}
}

// TestSbctl_NoStdoutOnConnectionFailure verifies BC-2.07.003 PC-3 and AC-005:
// When the daemon is unreachable, sbctl produces zero bytes on stdout.
//
// BC: BC-2.07.003 PC-3; AC-005.
func TestSbctl_NoStdoutOnConnectionFailure(t *testing.T) {
	t.Parallel()

	target := "127.0.0.1:19997"
	keyPath := testdataKeyPath(t)

	_, stdout, _ := runSubprocess(t, "NoStdoutOnConnectionFailure",
		"SBCTL_TEST_TARGET="+target,
		"SBCTL_TEST_KEY="+keyPath,
		"SBCTL_TEST_TIMEOUT=200ms",
	)

	if stdout != "" {
		t.Errorf("BC-2.07.003 PC-3 violated: expected empty stdout on connection failure; got %d bytes: %q", len(stdout), stdout)
	}
}

// TestSbctl_ConnectionTimeout verifies BC-2.07.003 Inv-2 and AC-007:
// sbctl does not hang indefinitely. After --timeout expires, it exits with
// E-NET-001. The elapsed wall time must be >= the configured timeout, which
// proves sbctl actually waited on the network (not just failed at key loading).
//
// RED GATE: with the stub, loadEd25519Key returns "not implemented" immediately
// (< 50ms). The elapsed time assertion (>= 80ms) therefore FAILS.
//
// BC: BC-2.07.003 Inv-2 (timeout); AC-007.
func TestSbctl_ConnectionTimeout(t *testing.T) {
	t.Parallel()

	// Start a listener that accepts connections but never sends data.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	target := ln.Addr().String()
	keyPath := testdataKeyPath(t)

	// Accept and hold connections indefinitely.
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			// Block forever — never read or write.
			t.Cleanup(func() { _ = conn.Close() })
		}
	}()

	const timeoutDur = 100 * time.Millisecond
	start := time.Now()
	exitCode, _, stderr := runSubprocess(t, "ConnectionTimeout",
		"SBCTL_TEST_TARGET="+target,
		"SBCTL_TEST_KEY="+keyPath,
		"SBCTL_TEST_TIMEOUT="+timeoutDur.String(),
	)
	elapsed := time.Since(start)

	// The subprocess must have waited at least timeoutDur - 20ms before exiting.
	// If it exited in < 80ms, it failed at key loading (not at the network level).
	const minElapsed = 80 * time.Millisecond
	if elapsed < minElapsed {
		t.Errorf("AC-007 violated: sbctl exited too fast (%v < %v) — key loading likely failed rather than hitting the network timeout; real dial+timeout required", elapsed, minElapsed)
	}

	// Must exit within a reasonable bound after the timeout.
	const maxElapsed = 5 * time.Second
	if elapsed > maxElapsed {
		t.Errorf("AC-007 violated: sbctl did not exit within %v of the %v timeout (elapsed %v)", maxElapsed, timeoutDur, elapsed)
	}

	if exitCode != 1 {
		t.Errorf("AC-007 violated: expected exit code 1, got %d\nstderr: %s", exitCode, stderr)
	}
	if !strings.Contains(stderr, "E-NET-001") {
		t.Errorf("AC-007 violated: expected stderr to contain 'E-NET-001'; got: %q", stderr)
	}
}
