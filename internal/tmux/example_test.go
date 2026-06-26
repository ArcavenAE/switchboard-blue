// Package tmux_test — godoc examples exercising the public tmux API end-to-end.
// This file is evidence for S-3.01a demo-recording: it demonstrates AC-001
// through AC-004 and EC-001 using a hermetic fake control-mode stream. No
// real tmux binary is invoked.
package tmux_test

import (
	"context"
	"errors"
	"fmt"
	"io"
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

// exampleFakeExec returns a WithExecFunc option that yields the given stream.
// H-03: signature is (stdin WriteCloser, stdout ReadCloser, err).
func exampleFakeExec(r io.ReadCloser) tmux.Option {
	return tmux.WithExecFunc(func(_ context.Context) (io.WriteCloser, io.ReadCloser, error) {
		return exampleNopWriteCloser{}, r, nil
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
	unavailableExec := tmux.WithExecFunc(func(_ context.Context) (io.WriteCloser, io.ReadCloser, error) {
		return nil, nil, tmux.ErrControlModeUnavailable
	})
	cm := newExampleControl(unavailableExec)

	err := cm.Connect(context.Background())
	fmt.Println("is ErrControlModeUnavailable:", errors.Is(err, tmux.ErrControlModeUnavailable))

	// Output:
	// is ErrControlModeUnavailable: true
}
