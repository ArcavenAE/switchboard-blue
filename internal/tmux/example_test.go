// Package tmux_test — godoc examples exercising the public tmux API end-to-end.
// This file is evidence for S-3.01a demo-recording (AC-001 through AC-004 +
// EC-001) and S-3.01b demo-recording (AC-001 through AC-003 + EC-001 + EC-003)
// using a hermetic fake control-mode stream and fake PTY allocator. No real
// tmux binary or PTY device is invoked.
package tmux_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/halfchannel"
	"github.com/arcavenae/switchboard/internal/session"
	"github.com/arcavenae/switchboard/internal/tmux"
)

// exampleNopCloser wraps an io.Reader with a no-op Close for use as a fake
// tmux subprocess stdout in examples (hermetic; no real tmux invoked).
type exampleNopCloser struct{ io.Reader }

func (exampleNopCloser) Close() error { return nil }

// exampleNopWriteCloser is a no-op io.WriteCloser used as the fake stdin.
// Connect writes list-sessions to stdin; the fake discards the write.
type exampleNopWriteCloser struct{}

func (exampleNopWriteCloser) Write(p []byte) (int, error) { return len(p), nil }
func (exampleNopWriteCloser) Close() error                { return nil }

// exampleStream builds a fake control-mode stdout from pre-scripted lines.
func exampleStream(lines ...string) io.ReadCloser {
	return exampleNopCloser{strings.NewReader(strings.Join(lines, "\n") + "\n")}
}

// exampleClosedNilChan returns a pre-closed nil-classification channel.
// Example fakes do not exercise the classification path (M-002/L-004).
func exampleClosedNilChan() <-chan error {
	ch := make(chan error, 1)
	close(ch)
	return ch
}

// exampleFakeExec returns a WithExecFunc option that yields the given stream.
// M-002/L-004: updated to match the new execFunc signature (stdin, stdout, classifyCh, err).
func exampleFakeExec(r io.ReadCloser) tmux.Option {
	return tmux.WithExecFunc(func(_ context.Context) (io.WriteCloser, io.ReadCloser, <-chan error, error) {
		return exampleNopWriteCloser{}, r, exampleClosedNilChan(), nil
	})
}

// newExampleControl is the canonical example constructor. It wires a
// Publisher and a downstream HalfChannel into a new ControlMode.
func newExampleControl(opts ...tmux.Option) *tmux.ControlMode {
	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)
	return tmux.New(pub, ds, opts...)
}

// ExampleControlMode_connect demonstrates AC-001: Connect establishes a tmux
// control mode connection against a hermetic fake stream. On success, a second
// Connect call returns ErrAlreadyConnected, and Close shuts down cleanly.
// Traces to BC-2.04.001 PC-1 + ADR-010 (tmux control mode primary).
func ExampleControlMode_connect() {
	stream := exampleStream(
		"%begin 0",
		"%end 0",
	)
	cm := newExampleControl(exampleFakeExec(stream))

	err := cm.Connect(context.Background())
	fmt.Println("connect error:", err)

	// Second call is rejected — single-use guard.
	err2 := cm.Connect(context.Background())
	fmt.Println("is ErrAlreadyConnected:", errors.Is(err2, tmux.ErrAlreadyConnected))

	_ = cm.Close()
	fmt.Println("closed cleanly")

	// Output:
	// connect error: <nil>
	// is ErrAlreadyConnected: true
	// closed cleanly
}

// ExampleControlMode_enumerateSessions demonstrates AC-002: after Connect, all
// sessions present in the %begin/%end list-sessions response block are published
// and visible via Sessions(). Traces to BC-2.04.001 PC-2.
func ExampleControlMode_enumerateSessions() {
	// The fake stream delivers the list-sessions response block — two sessions.
	stream := exampleStream(
		"%begin 0",
		"alpha",
		"beta",
		"%end 0",
	)
	cm := newExampleControl(exampleFakeExec(stream))

	_ = cm.Connect(context.Background())

	// Allow dispatchLoop time to consume the stream before querying sessions.
	time.Sleep(20 * time.Millisecond)

	sessions := cm.Sessions()
	fmt.Println("session count:", len(sessions))
	for _, s := range sessions {
		fmt.Println("session:", s.Name)
	}

	_ = cm.Close()

	// Output:
	// session count: 2
	// session: alpha
	// session: beta
}

// ExampleControlMode_sessionLifecycle demonstrates AC-003: %session-created and
// %session-closed events are processed and update the published session set.
// Traces to BC-2.04.001 PC-3 + PC-4.
func ExampleControlMode_sessionLifecycle() {
	// The fake stream: list-sessions returns empty set, then lifecycle events fire.
	stream := exampleStream(
		"%begin 0",
		"%end 0",
		"%session-created gamma",
		"%session-created delta",
		"%session-closed gamma",
	)
	cm := newExampleControl(exampleFakeExec(stream))

	_ = cm.Connect(context.Background())

	// Allow dispatchLoop time to process all events.
	time.Sleep(20 * time.Millisecond)

	sessions := cm.Sessions()
	fmt.Println("session count:", len(sessions))
	for _, s := range sessions {
		fmt.Println("session:", s.Name)
	}

	_ = cm.Close()

	// Output:
	// session count: 1
	// session: delta
}

// ExampleControlMode_outputFramesDelivered demonstrates AC-004: %output events
// feed the downstream half-channel and frames are delivered via Frames().
// Traces to BC-2.04.001 PC-5.
func ExampleControlMode_outputFramesDelivered() {
	// The fake stream: one %output event carrying "hello" (space is \040 in
	// tmux octal encoding).
	stream := exampleStream(
		"%begin 0",
		"%end 0",
		`%output %1 hello\040world`,
	)
	cm := newExampleControl(exampleFakeExec(stream))

	_ = cm.Connect(context.Background())

	// Drain the Frames channel until we receive at least one frame or the
	// channel closes after dispatchLoop exits.
	var received int
	for f := range cm.Frames() {
		if len(f.Payload) > 0 {
			received++
		}
	}
	fmt.Println("frames received:", received)

	_ = cm.Close()

	// Output:
	// frames received: 1
}

// ExampleControlMode_tmuxUnavailable demonstrates EC-001: when tmux is not
// found in PATH, Connect returns ErrControlModeUnavailable and the caller
// should trigger PTY fallback (S-3.01b). Traces to BC-2.04.001 EC-001/EC-004
// and ADR-010 (PTY fallback trigger).
func ExampleControlMode_tmuxUnavailable() {
	// Inject an exec function that returns ErrControlModeUnavailable, simulating
	// a missing tmux binary (lookup failure; hermetic — no real PATH search).
	unavailableExec := tmux.WithExecFunc(func(_ context.Context) (io.WriteCloser, io.ReadCloser, <-chan error, error) {
		return nil, nil, nil, tmux.ErrControlModeUnavailable
	})
	cm := newExampleControl(unavailableExec)

	err := cm.Connect(context.Background())
	fmt.Println("is ErrControlModeUnavailable:", errors.Is(err, tmux.ErrControlModeUnavailable))

	// Output:
	// is ErrControlModeUnavailable: true
}

// ── S-3.01b examples ─────────────────────────────────────────────────────────

// exampleCapturingLogger captures log lines for hermetic assertion.
// Satisfies tmux.Logger without writing to stderr.
type exampleCapturingLogger struct {
	lines []string
}

func (l *exampleCapturingLogger) Log(msg string) {
	l.lines = append(l.lines, msg)
}

// exampleFakePTYAlloc returns a WithPTYAllocFunc option that yields a hermetic
// in-process pipe as the PTY master and the given pid. The slave side of the
// pipe is closed immediately — ioRelay reads EOF and exits cleanly.
func exampleFakePTYAlloc(pid int) tmux.PTYProxyOption {
	return tmux.WithPTYAllocFunc(func() (io.ReadWriteCloser, int, error) {
		// A pipe whose write end is closed immediately causes ioRelay to observe
		// EOF and exit, giving the example a clean lifecycle without goroutine leaks.
		pr, pw, err := os.Pipe()
		if err != nil {
			return nil, 0, err
		}
		_ = pw.Close() // slave side EOF — ioRelay will drain and exit
		return pr, pid, nil
	})
}

// newExamplePTYProxy constructs a PTYProxy wired to a fresh publisher,
// downstream, and capturing logger.  opts are appended after the required
// PTYAllocFunc so callers can override defaults.
func newExamplePTYProxy(log *exampleCapturingLogger, opts ...tmux.PTYProxyOption) *tmux.PTYProxy {
	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)
	allOpts := append([]tmux.PTYProxyOption{tmux.WithLogger(log)}, opts...)
	return tmux.NewPTYProxy(pub, ds, allOpts...)
}

// newExampleConnector constructs a SessionConnector from a ControlMode and
// PTYProxy sharing the same publisher/downstream.
func newExampleConnector(ctrlOpts []tmux.Option, ptyPID int, log *exampleCapturingLogger, scOpts ...tmux.SessionConnectorOption) *tmux.SessionConnector {
	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)

	ctrl := tmux.New(pub, ds, ctrlOpts...)
	pty := tmux.NewPTYProxy(pub, ds,
		tmux.WithLogger(log),
		exampleFakePTYAlloc(ptyPID),
	)
	return tmux.NewSessionConnector(ctrl, pty, scOpts...)
}

// ExampleSessionConnector_initialFallback demonstrates AC-001: when tmux
// control mode is unavailable on initial Connect, SessionConnector falls back
// to PTY proxy mode automatically. Traces to BC-2.04.002 PC-1 + ADR-010 v1.2.
func ExampleSessionConnector_initialFallback() {
	log := &exampleCapturingLogger{}

	// Inject a control mode exec that returns ErrControlModeUnavailable (no
	// tmux binary) and a PTY allocator that yields pid 42.
	unavailableExec := tmux.WithExecFunc(func(_ context.Context) (io.WriteCloser, io.ReadCloser, <-chan error, error) {
		return nil, nil, nil, tmux.ErrControlModeUnavailable
	})
	sc := newExampleConnector([]tmux.Option{unavailableExec}, 42, log)

	err := sc.Connect(context.Background())
	fmt.Println("connect error:", err)
	fmt.Println("in PTY mode:", sc.InPTYMode())

	sessions := sc.Sessions()
	fmt.Println("session count:", len(sessions))
	if len(sessions) > 0 {
		fmt.Println("session name:", sessions[0].Name)
	}

	_ = sc.Close()

	// Output:
	// connect error: <nil>
	// in PTY mode: true
	// session count: 1
	// session name: pty-42
}

// ExamplePTYProxy_publishSession demonstrates AC-002: in PTY proxy mode,
// the access node publishes the PTY session under the synthetic name
// "pty-<pid>" and writes a mandatory canonical log entry. No silent failure.
// Traces to BC-2.04.002 PC-2 + PC-3.
func ExamplePTYProxy_publishSession() {
	log := &exampleCapturingLogger{}
	pty := newExamplePTYProxy(log, exampleFakePTYAlloc(99))

	err := pty.Connect(context.Background())
	fmt.Println("connect error:", err)

	sessions := pty.Sessions()
	fmt.Println("session count:", len(sessions))
	if len(sessions) > 0 {
		fmt.Println("session name:", sessions[0].Name)
	}

	// Mandatory canonical log entry (BC-2.04.002 PC-3).
	var foundLog bool
	for _, line := range log.lines {
		if line == "tmux control mode unavailable; using PTY proxy mode. Functionality limited: no structured session metadata, no content-type detection." {
			foundLog = true
			break
		}
	}
	fmt.Println("mandatory log present:", foundLog)

	_ = pty.Close()

	// Output:
	// connect error: <nil>
	// session count: 1
	// session name: pty-99
	// mandatory log present: true
}

// ExamplePTYProxy_bothUnavailable demonstrates AC-003: when both tmux control
// mode and the PTY device are unavailable, Connect returns
// ErrPTYDeviceUnavailable (E-SYS-001) and the failure is never silent — the
// operator-facing guidance is emitted to the logger before the error is
// returned. Traces to BC-2.04.002 EC-004.
func ExamplePTYProxy_bothUnavailable() {
	log := &exampleCapturingLogger{}

	// Inject a PTY allocator that always fails (no PTY device on this host).
	noPTY := tmux.WithPTYAllocFunc(func() (io.ReadWriteCloser, int, error) {
		return nil, 0, tmux.ErrPTYDeviceUnavailable
	})
	pty := newExamplePTYProxy(log, noPTY)

	err := pty.Connect(context.Background())
	fmt.Println("is ErrPTYDeviceUnavailable:", errors.Is(err, tmux.ErrPTYDeviceUnavailable))

	// Operator-facing guidance MUST be logged before the error surfaces
	// (BC-2.04.002 invariant 3 — never silent).
	var foundGuidance bool
	for _, line := range log.lines {
		if strings.Contains(line, "Install 'openpty'") {
			foundGuidance = true
			break
		}
	}
	fmt.Println("operator guidance logged:", foundGuidance)

	// Output:
	// is ErrPTYDeviceUnavailable: true
	// operator guidance logged: true
}

// ExampleSessionConnector_oldTmuxFallback demonstrates EC-001: when tmux
// exists but rejects the -CC flag (old version), the classification sentinel
// ErrControlModeUnsupportedFlag arrives on the Err() channel after subprocess
// exit. SessionConnector's watchAndFallback goroutine handles it asynchronously
// and activates PTY proxy mode. Traces to BC-2.04.002 EC-001 + ADR-010 v1.2.
func ExampleSessionConnector_oldTmuxFallback() {
	log := &exampleCapturingLogger{}

	// classifyCh delivers ErrControlModeUnsupportedFlag after the fake
	// subprocess's stdout stream closes, simulating an old tmux that rejects -CC.
	classifyCh := make(chan error, 1)
	classifyCh <- tmux.ErrControlModeUnsupportedFlag
	close(classifyCh)

	// The exec func returns a valid (but immediately-empty) stream so Connect
	// succeeds.  The classifyCh fires after EOF, which watchAndFallback picks up.
	oldTmuxExec := tmux.WithExecFunc(func(_ context.Context) (io.WriteCloser, io.ReadCloser, <-chan error, error) {
		// Empty stream — dispatchLoop observes EOF immediately after the %begin/%end.
		stream := exampleStream("%begin 0", "%end 0")
		return exampleNopWriteCloser{}, stream, classifyCh, nil
	})

	sc := newExampleConnector([]tmux.Option{oldTmuxExec}, 77, log)

	err := sc.Connect(context.Background())
	fmt.Println("connect error:", err)

	// Allow watchAndFallback goroutine time to receive the classification signal
	// and activate PTY proxy mode.
	time.Sleep(20 * time.Millisecond)

	fmt.Println("in PTY mode:", sc.InPTYMode())
	_ = sc.Close()

	// Output:
	// connect error: <nil>
	// in PTY mode: true
}

// ExampleSessionConnector_midSessionFallback demonstrates EC-003: when tmux
// control mode drops mid-session and all three reconnect attempts via
// ControlModeFactory also fail, SessionConnector falls back to PTY proxy mode
// and logs the mandatory "tmux control mode lost; falling back to PTY proxy"
// message. Traces to BC-2.04.002 EC-003 + ADR-010 v1.2.
func ExampleSessionConnector_midSessionFallback() {
	log := &exampleCapturingLogger{}

	// Initial control mode connects successfully with an empty stream (EOF
	// arrives immediately after the response block, triggering a drop signal).
	initialExec := tmux.WithExecFunc(func(_ context.Context) (io.WriteCloser, io.ReadCloser, <-chan error, error) {
		stream := exampleStream("%begin 0", "%end 0")
		return exampleNopWriteCloser{}, stream, exampleClosedNilChan(), nil
	})

	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)

	ctrl := tmux.New(pub, ds, initialExec)
	pty := tmux.NewPTYProxy(pub, ds,
		tmux.WithLogger(log),
		exampleFakePTYAlloc(55),
	)

	// Factory always fails — all three reconnect attempts are exhausted before
	// the fallback to PTY proxy mode is triggered (BC-2.04.002 EC-003).
	failFactory := tmux.WithControlModeFactory(func(_ context.Context) (*tmux.ControlMode, error) {
		return nil, tmux.ErrControlModeUnavailable
	})
	sc := tmux.NewSessionConnector(ctrl, pty, failFactory)

	err := sc.Connect(context.Background())
	fmt.Println("connect error:", err)

	// Allow watchAndFallback time to exhaust all 3 reconnect attempts and
	// activate PTY proxy mode.
	time.Sleep(50 * time.Millisecond)

	fmt.Println("in PTY mode:", sc.InPTYMode())

	// Mandatory log entry written before fallback activation.
	var foundFallbackLog bool
	for _, line := range log.lines {
		if line == "tmux control mode lost; falling back to PTY proxy" {
			foundFallbackLog = true
			break
		}
	}
	fmt.Println("fallback log present:", foundFallbackLog)

	_ = sc.Close()

	// Output:
	// connect error: <nil>
	// in PTY mode: true
	// fallback log present: true
}
