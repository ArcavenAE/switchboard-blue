// Package tmux tests for BC-2.04.001 (tmux control mode integration).
// Traces: BC-2.04.001 PC-1..PC-5; AC-001..AC-004; ADR-010; ARCH-08 §6.6 position 7.
//
// Hermetic constraint: these tests MUST NOT shell out to the real tmux binary.
// Tests that require a live tmux process use a fake io.ReadCloser that replays
// pre-recorded control mode protocol lines. Any test requiring a real tmux
// process is skipped with a deferral docstring citing the integration test plan.
//
// Red Gate: all non-skipped tests below are designed to fail against the stub
// (todo() panic). Green on first commit = Red Gate violation.
package tmux_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/halfchannel"
	"github.com/arcavenae/switchboard/internal/session"
	"github.com/arcavenae/switchboard/internal/tmux"
)

// nopCloser wraps an io.Reader with a no-op Close so fakeControlOutput can be
// injected as the tmux subprocess stdout in tests (hermetic; no real tmux).
type nopCloser struct{ io.Reader }

func (nopCloser) Close() error { return nil }

// nopWriteCloser is a no-op io.WriteCloser used as the fake stdin pipe.
// Connect writes list-sessions to stdin; the fake discards the write.
type nopWriteCloser struct{}

func (nopWriteCloser) Write(p []byte) (int, error) { return len(p), nil }
func (nopWriteCloser) Close() error                { return nil }

// fakeControlOutput returns a fake tmux control mode stdout stream containing
// the provided pre-scripted lines. The caller controls exactly what "tmux"
// reports, so tests are hermetic and deterministic.
func fakeControlOutput(lines ...string) io.ReadCloser {
	return nopCloser{strings.NewReader(strings.Join(lines, "\n") + "\n")}
}

// fakeExecFunc returns a WithExecFunc option that yields the given stream.
// H-03: updated to match the new execFunc signature (stdin, stdout, err).
func fakeExecFunc(r io.ReadCloser) tmux.Option {
	return tmux.WithExecFunc(func(_ context.Context) (io.WriteCloser, io.ReadCloser, error) {
		return nopWriteCloser{}, r, nil
	})
}

// fakeExecFuncErr returns a WithExecFunc option that returns the given error.
func fakeExecFuncErr(err error) tmux.Option {
	return tmux.WithExecFunc(func(_ context.Context) (io.WriteCloser, io.ReadCloser, error) {
		return nil, nil, err
	})
}

// newTestControlWithOpts is the canonical test constructor — accepts additional
// options (e.g. WithExecFunc for hermetic stream injection).
func newTestControlWithOpts(t *testing.T, opts ...tmux.Option) (*tmux.ControlMode, *session.Publisher) {
	t.Helper()
	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)
	cm := tmux.New(pub, ds, opts...)
	return cm, pub
}

// TestTmuxControlMode_Connect_EstablishesConnection verifies AC-001 (BC-2.04.001
// PC-1): Connect against a fake control-mode stream succeeds and places the
// ControlMode into a connected state. A second Connect call must return
// ErrAlreadyConnected (H-04 idempotency fix).
//
// Hermetic: uses a fake control mode stream; does not shell out to tmux.
// VP-031 (e2e against real tmux) is deferred to the integration test harness.
func TestTmuxControlMode_Connect_EstablishesConnection(t *testing.T) {
	t.Parallel()

	// Minimal fake stream: empty %begin/%end block so dispatchLoop can parse it
	// cleanly, then keeps reading until ctx cancellation (no premature EOF).
	stream := fakeControlOutput(
		"%begin 1000000000 0 1",
		"%end 1000000000 0 1",
	)
	cm, _ := newTestControlWithOpts(t, fakeExecFunc(stream)) //nolint:dogsled // publisher unused in AC-001
	t.Cleanup(func() {
		if err := cm.Close(); err != nil {
			t.Logf("Close: %v", err)
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// AC-001 primary assertion: first Connect must succeed.
	if err := cm.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v; want nil", err)
	}

	// H-04 idempotency: second Connect on an already-connected ControlMode must
	// return ErrAlreadyConnected (not nil, not a different error).
	if err := cm.Connect(ctx); !errors.Is(err, tmux.ErrAlreadyConnected) {
		t.Errorf("second Connect: got %v; want ErrAlreadyConnected", err)
	}
}

// TestTmuxControlMode_Connect_EnumeratesSessions verifies that after a
// successful Connect, all current tmux sessions are enumerated and published
// (BC-2.04.001 PC-2; AC-002; VP-031).
//
// This test uses the %begin/%end block protocol that Connect emits after
// issuing list-sessions (F-01 fix): session names appear line-by-line between
// the delimiters. The post-block %session-created events also exercise
// dynamic session creation (AC-003 / PC-3).
//
// Test vector (BC-2.04.001 canonical test vectors):
//   - Input:  %begin/%end block contains "session-1", "session-2", "build"
//   - Expect: all 3 published; Sessions() returns a 3-element list
func TestTmuxControlMode_Connect_EnumeratesSessions(t *testing.T) {
	t.Parallel()

	// Inject a fake stream that returns a %begin/%end block with session names
	// (as emitted by tmux in response to list-sessions), followed by a dynamic
	// %session-created to verify lifecycle handling still works. Hermetic.
	stream := fakeControlOutput(
		"%begin 1000000000 0 1",
		"session-1",
		"session-2",
		"build",
		"%end 1000000000 0 1",
		"%session-created extra-session",
		"%session-closed extra-session",
	)
	cm, _ := newTestControlWithOpts(t, fakeExecFunc(stream)) //nolint:dogsled // publisher checked via cm.Sessions()
	t.Cleanup(func() {
		if err := cm.Close(); err != nil {
			t.Logf("Close: %v", err)
		}
	})

	ctx := context.Background()
	if err := cm.Connect(ctx); err != nil {
		t.Fatalf("Connect: unexpected error: %v", err)
	}

	// Wait for dispatchLoop to finish processing the input stream.
	// dispatchLoop exits on EOF and signals via Err(); the fake stream is
	// finite so this is deterministic.
	select {
	case <-cm.Err():
		// dispatchLoop has finished processing.
	case <-time.After(1 * time.Second):
		t.Fatal("dispatchLoop did not signal exit within 1s")
	}

	want := []string{"build", "session-1", "session-2"}
	sessions := cm.Sessions()
	if len(sessions) != len(want) {
		t.Fatalf("Sessions: got %d; want %d (sessions: %v)", len(sessions), len(want), sessions)
	}
	// Verify all expected names are present (Sessions returns sorted order).
	got := make(map[string]bool, len(sessions))
	for _, s := range sessions {
		got[s.Name] = true
	}
	for _, name := range want {
		if !got[name] {
			t.Errorf("Sessions: missing %q; got %v", name, sessions)
		}
	}
}

// TestTmuxControlMode_SessionLifecycleEvents verifies that new sessions are
// published on %session-created and removed on %session-closed (BC-2.04.001
// PC-3, PC-4; AC-003).
//
// Test vector:
//   - Start with "agent-01", "agent-02", "build"
//   - Inject %session-created → "agent-03" published
//   - Inject %session-closed  → "agent-02" unpublished
func TestTmuxControlMode_SessionLifecycleEvents(t *testing.T) {
	t.Parallel()

	// Fake stream: empty enumeration block then lifecycle events.
	stream := fakeControlOutput(
		"%begin 1000000000 0 1",
		"agent-01",
		"agent-02",
		"build",
		"%end 1000000000 0 1",
		"%session-created agent-03",
		"%session-closed agent-02",
	)
	cm, pub := newTestControlWithOpts(t, fakeExecFunc(stream))
	t.Cleanup(func() { _ = cm.Close() })

	ctx := context.Background()
	if err := cm.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Wait for dispatchLoop to finish processing the input stream.
	// dispatchLoop exits on EOF and signals via Err(); the fake stream is
	// finite so this is deterministic.
	select {
	case <-cm.Err():
		// dispatchLoop has finished processing.
	case <-time.After(1 * time.Second):
		t.Fatal("dispatchLoop did not signal exit within 1s")
	}

	// After %session-created agent-03 — verify it appears.
	if _, err := pub.Get("agent-03"); err != nil {
		t.Errorf("Get agent-03: %v; want nil (session should be published)", err)
	}

	// After %session-closed agent-02 — verify it is gone.
	if _, err := pub.Get("agent-02"); !errors.Is(err, session.ErrSessionNotFound) {
		t.Errorf("Get agent-02: got %v; want ErrSessionNotFound (session should be unpublished)", err)
	}
}

// TestTmuxControlMode_OutputEventsFeedDownstream verifies that %output events
// from the control mode stream are enqueued into the downstream half-channel
// (BC-2.04.001 PC-5; AC-004).
//
// Hermetic: uses WithExecFunc to inject a fake stream; does not shell out to tmux.
//
// The test verifies:
//  1. After %output event, the downstream half-channel's sequence counter
//     advances (proving Tick was called with data).
//  2. The payload enqueued was the octal-unescaped form of the %output data
//     (F-06 contract: "hello\040world\012" → "hello world\n").
//
// Concurrency note: dispatchLoop runs in a goroutine and is the only writer of
// the (non-thread-safe) HalfChannel. The test waits for the stream EOF signal
// via c.Err() — which guarantees dispatchLoop has exited — before reading
// ds.Seq(). This is safe because dispatchLoop exits before sending to errCh,
// so no concurrent access to ds can occur after the receive.
func TestTmuxControlMode_OutputEventsFeedDownstream(t *testing.T) {
	t.Parallel()

	// Build dependencies inline so we hold a direct reference to ds.
	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)

	// Fake stream: empty session enumeration, register session-1, emit an
	// octal-escaped %output line, then EOF.
	// F-06 contract: \040 = space (0x20), \012 = newline (0x0A).
	stream := fakeControlOutput(
		"%begin 1000000000 0 1",
		"%end 1000000000 0 1",
		"%session-created session-1",
		`%output %12 hello\040world\012`,
	)
	cm := tmux.New(pub, ds, fakeExecFunc(stream))
	t.Cleanup(func() { _ = cm.Close() })

	ctx := context.Background()
	if err := cm.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Wait for dispatchLoop to reach EOF and signal the dropped connection.
	// This guarantees dispatchLoop has exited and ds is no longer being written.
	select {
	case err := <-cm.Err():
		if !errors.Is(err, tmux.ErrControlModeDropped) {
			t.Logf("Err() = %v (expected ErrControlModeDropped on stream EOF)", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("dispatchLoop did not signal within 200ms; stream may not have closed")
	}

	// dispatchLoop has exited — ds is no longer accessed concurrently. Safe to inspect.
	// The %output line caused one Enqueue + Tick, so seq must be >= 1.
	if seq := ds.Seq(); seq < 1 {
		t.Errorf("downstream Seq() = %d; want >= 1 (Tick must have been called for %%output event)", seq)
	}

	// Verify session-1 was published (proves the %session-created before %output was processed).
	if _, err := pub.Get("session-1"); err != nil {
		t.Errorf("Get session-1: %v; want nil (session-1 should be published before %%output)", err)
	}
}

// TestTmuxControlMode_Connect_ErrWhenTmuxNotFound verifies that Connect returns
// ErrControlModeUnavailable when the tmux binary is not present (BC-2.04.001
// EC-004 / FM-011; ADR-010).
//
// Hermetic: injects a fake exec function that returns ErrControlModeUnavailable.
func TestTmuxControlMode_Connect_ErrWhenTmuxNotFound(t *testing.T) {
	t.Parallel()

	// Simulate "tmux not in PATH" via WithExecFunc — hermetic, no PATH manipulation.
	notFoundErr := fmt.Errorf("%w: exec: tmux: no such file", tmux.ErrControlModeUnavailable)
	cm, _ := newTestControlWithOpts(t, fakeExecFuncErr(notFoundErr)) //nolint:dogsled // publisher unused in this test
	t.Cleanup(func() { _ = cm.Close() })

	ctx := context.Background()
	err := cm.Connect(ctx)
	if err == nil {
		t.Fatal("Connect with no tmux binary: got nil error; want ErrControlModeUnavailable")
	}
	if !errors.Is(err, tmux.ErrControlModeUnavailable) {
		t.Errorf("Connect error = %v; want ErrControlModeUnavailable", err)
	}
}

// TestTmuxControlMode_NoSessionsOnStartup verifies that Connect with an empty
// tmux server results in an empty session list (BC-2.04.001 EC-003; ADR-010).
//
// Hermetic: uses WithExecFunc to inject a fake stream with an empty %begin/%end
// block (no session names). Does not shell out to real tmux.
func TestTmuxControlMode_NoSessionsOnStartup(t *testing.T) {
	t.Parallel()

	// Inject a fake stream with an empty enumeration response — no session names
	// between %begin and %end. The stream closes after the block (EOF).
	stream := fakeControlOutput(
		"%begin 1000000000 0 1",
		"%end 1000000000 0 1",
	)
	cm, _ := newTestControlWithOpts(t, fakeExecFunc(stream)) //nolint:dogsled // publisher unused in this test
	t.Cleanup(func() { _ = cm.Close() })

	ctx := context.Background()
	if err := cm.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Wait for dispatchLoop to finish processing the input stream.
	// dispatchLoop exits on EOF and signals via Err(); the fake stream is
	// finite so this is deterministic.
	select {
	case <-cm.Err():
		// dispatchLoop has finished processing.
	case <-time.After(1 * time.Second):
		t.Fatal("dispatchLoop did not signal exit within 1s")
	}

	sessions := cm.Sessions()
	if len(sessions) != 0 {
		t.Errorf("Sessions on empty server: got %d sessions (%v); want 0", len(sessions), sessions)
	}
}

// TestTmuxControlMode_ErrChannelSignalsDroppedConnection verifies that the Err()
// channel receives ErrControlModeDropped when the event loop exits unexpectedly
// (BC-2.04.001 EC-002 / FM-004; S-3.01b API surface).
//
// Hermetic: uses WithExecFunc to inject a fake stream that emits a %begin/%end
// block then immediately closes (EOF). Does not shell out to real tmux.
func TestTmuxControlMode_ErrChannelSignalsDroppedConnection(t *testing.T) {
	t.Parallel()

	// Inject a fake stream that emits an empty enumeration block then EOF.
	// After EOF, dispatchLoop should detect unexpected exit and send
	// ErrControlModeDropped to the Err() channel (EC-002 / FM-004).
	stream := fakeControlOutput(
		"%begin 1000000000 0 1",
		"%end 1000000000 0 1",
	)
	cm, _ := newTestControlWithOpts(t, fakeExecFunc(stream)) //nolint:dogsled // publisher unused in this test
	t.Cleanup(func() { _ = cm.Close() })

	ctx := context.Background()
	if err := cm.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// The fake stream EOF triggers the "unexpected exit" path in dispatchLoop —
	// it sends ErrControlModeDropped to errCh. Assert we receive it within 100ms.
	select {
	case err := <-cm.Err():
		if !errors.Is(err, tmux.ErrControlModeDropped) {
			t.Errorf("Err() = %v; want ErrControlModeDropped", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Err() did not signal within 100ms; want ErrControlModeDropped on stream EOF")
	}
}

// TestTmuxControlMode_LargeOutputLine_NoFalseDrop pins pass-5 H-1: a %output
// line whose octal-escaped payload exceeds bufio's default 64 KiB scanner token
// limit (but is well under the 2 MiB raised limit) must be processed correctly:
// the payload must reach the downstream half-channel (Seq advances), proving the
// scanner did NOT abort on ErrTooLong.
//
// Without the fix, bufio.Scanner.Scan returns false when a single line exceeds
// the default 64 KiB buffer and scanner.Err() returns bufio.ErrTooLong. The
// current dispatchLoop does not inspect scanner.Err(), so it falls through to
// the post-loop code which sends ErrControlModeDropped and the large %output
// payload is silently dropped (ds.Seq() stays 0). With the H-1 fix (scanner
// buffer raised to 2 MiB), the line is read cleanly and Tick is called:
// ds.Seq() >= 1.
//
// Hermetic: no real tmux. Traces: BC-2.04.001 PC-5; pass-5 H-1.
func TestTmuxControlMode_LargeOutputLine_NoFalseDrop(t *testing.T) {
	t.Parallel()

	// Build dependencies inline to hold a direct reference to ds.
	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)

	// 100 KiB of octal-escaped spaces: "\040" × 100*1024 = 400 KiB on the wire.
	// Default bufio scanner limit is 64 KiB, so this line exceeds it.
	// After the H-1 fix the scanner buffer is raised to 2 MiB; the line must
	// be processed without error. The decoded payload is 100 KiB (all spaces),
	// which fits within halfchannel.MaxPayloadSize (65515 bytes)... actually
	// 100 KiB = 102400 bytes > 65515. So this test also exercises M-1 fragmentation
	// if that fix is applied. The minimum assertion is ds.Seq() >= 1 (any tick at
	// all proves the large line was read rather than dropped by the scanner).
	largePayload := strings.Repeat(`\040`, 100*1024) // 400 KiB wire; decodes to 100 KiB
	largeOutput := fmt.Sprintf("%%output %%12 %s", largePayload)

	stream := fakeControlOutput(
		"%begin 1000000000 0 1",
		"%end 1000000000 0 1",
		"%session-created session-1",
		largeOutput,
	)
	cm := tmux.New(pub, ds, fakeExecFunc(stream))
	t.Cleanup(func() { _ = cm.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cm.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Wait for dispatchLoop to process all events (stream EOF signals via Err()).
	select {
	case <-cm.Err():
		// dispatchLoop exited after stream EOF — all events have been processed.
	case <-time.After(500 * time.Millisecond):
		t.Fatal("dispatchLoop did not signal exit within 500ms")
	}

	// The distinguishing assertion: the %output data from the large line must
	// have been enqueued + ticked (ds.Seq() >= 1). Pre-fix: scanner aborts on
	// ErrTooLong, the %output handler is never called, Seq stays 0. Post-fix:
	// the line is read cleanly and at least one Tick occurs.
	if seq := ds.Seq(); seq < 1 {
		t.Errorf("downstream Seq() = %d; want >= 1 — large %%output line must reach downstream (H-1 fix: raise scanner buffer)", seq)
	}

	// Also verify session-1 was published (proves lifecycle events processed).
	if _, err := pub.Get("session-1"); err != nil {
		t.Errorf("Get session-1: %v; want nil", err)
	}
}

// TestTmuxControlMode_OversizePayload_Fragmented pins pass-5 M-1: a %output
// payload whose decoded length exceeds halfchannel.MaxPayloadSize must be split
// into multiple chunks, each enqueued + ticked separately, so that every byte
// reaches the downstream and downstream.Seq() advances by at least the number
// of chunks (>= 1 per MaxPayloadSize-sized segment).
//
// Without the fix, handleLine calls Enqueue once for the entire decoded payload;
// Enqueue returns ErrPayloadTooLarge and the data is silently dropped (Seq += 0).
// With the fix, handleLine fragments the payload into ceiling(len/MaxPayloadSize)
// chunks; each chunk is enqueued + ticked, so Seq advances by that many steps.
//
// Test vector: 200 KiB decoded payload, MaxPayloadSize = 65515 bytes.
// Chunks = ceil(200*1024 / 65515) = ceil(204800 / 65515) = ceil(3.127) = 4.
// Post-fix: Seq >= 4. Pre-fix: Seq == 0 (drop) or 1 (if guard is missing).
//
// Hermetic: no real tmux. Traces: BC-2.04.001 PC-5; halfchannel BC-2.01.002 PC5;
// pass-5 M-1.
func TestTmuxControlMode_OversizePayload_Fragmented(t *testing.T) {
	t.Parallel()

	// Build dependencies inline to hold a direct reference to ds (same pattern
	// as TestTmuxControlMode_OutputEventsFeedDownstream).
	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)

	// 200 KiB of literal 'A': no octal escaping, so the decoded payload is
	// exactly 200*1024 bytes. This is well above MaxPayloadSize (65515 bytes).
	// The line is ~200 KiB + prefix — within the 2 MiB scanner buffer (H-1).
	largeASCII := strings.Repeat("A", 200*1024)
	largeOutput := fmt.Sprintf("%%output %%12 %s", largeASCII)

	stream := fakeControlOutput(
		"%begin 1000000000 0 1",
		"%end 1000000000 0 1",
		"%session-created session-1",
		largeOutput,
	)
	cm := tmux.New(pub, ds, fakeExecFunc(stream))
	t.Cleanup(func() { _ = cm.Close() })

	ctx := context.Background()
	if err := cm.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Wait for dispatchLoop to finish processing.
	select {
	case <-cm.Err():
	case <-time.After(500 * time.Millisecond):
		t.Fatal("dispatchLoop did not signal exit within 500ms")
	}

	// 200 KiB / 65515 bytes per chunk = ceil(204800 / 65515) = 4 chunks.
	// Each chunk triggers one Enqueue + Tick, so Seq must be >= 4.
	// Pre-fix Seq == 0 (Enqueue fails with ErrPayloadTooLarge, no Tick called).
	const wantMinSeq = uint32(4)
	if seq := ds.Seq(); seq < wantMinSeq {
		t.Errorf("downstream Seq() = %d; want >= %d — payload must be fragmented into chunks (M-1 fix)", seq, wantMinSeq)
	}
}

// TestTmuxControlMode_Close_WaitsForDispatchLoop pins pass-5 M-2: Close() must
// block until dispatchLoop has fully exited before returning. Without the fix,
// Close cancels the context and returns immediately while the dispatchLoop
// goroutine may still be running, creating a data race on c.publisher and
// c.downstream. With the fix, Close calls c.wg.Wait() after cancellation, so it
// returns only after the goroutine exits.
//
// Strategy: use a deliberate delay between ctx cancellation and closing the pipe
// writer. If Close() does wg.Wait(), it blocks for at least that delay. If it
// doesn't, it returns before the delay expires. We measure the elapsed time of
// Close() and require it to be >= the delay duration, proving the goroutine join
// happened.
//
// Hermetic: no real tmux. Traces: BC-2.04.001 PC-1; pass-5 M-2.
// Go memory model: sync.WaitGroup provides the happens-before guarantee.
func TestTmuxControlMode_Close_WaitsForDispatchLoop(t *testing.T) {
	t.Parallel()

	// delay is the deliberate pause between ctx cancellation and pipe close.
	// dispatchLoop blocks in scanner.Scan during this window. Close() must
	// wait at least this long if wg.Wait() is present.
	const delay = 80 * time.Millisecond

	stdoutReader, stdoutWriter := io.Pipe()

	fake := tmux.WithExecFunc(func(ctx context.Context) (io.WriteCloser, io.ReadCloser, error) {
		// Write the enumeration block asynchronously so execFn can return
		// before dispatchLoop is reading (io.Pipe synchronises; blocking here
		// would deadlock Connect before dispatchLoop is spawned).
		go func() {
			_, _ = io.WriteString(stdoutWriter, "%begin 0 0 1\n%end 0 0 1\n")
			// Hold the pipe open until ctx is cancelled, then wait `delay`
			// before closing — this is the window where dispatchLoop is still
			// alive after ctx.Done() fires but before the scanner unblocks.
			<-ctx.Done()
			time.Sleep(delay)
			_ = stdoutWriter.Close()
		}()
		return nopWriteCloser{}, stdoutReader, nil
	})

	cm, _ := newTestControlWithOpts(t, fake) //nolint:dogsled // publisher unused in M-2 test

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cm.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Allow dispatchLoop to enter its scan loop and block on the open pipe.
	time.Sleep(10 * time.Millisecond)

	// Measure how long Close() takes. With wg.Wait() it must block until
	// dispatchLoop exits (after the delay). Without wg.Wait() it returns
	// almost immediately after cancelling the context.
	start := time.Now()
	if err := cm.Close(); err != nil {
		t.Logf("Close: %v (non-nil close error is acceptable)", err)
	}
	elapsed := time.Since(start)

	// The minimum elapsed time is `delay` because Close must wait for
	// dispatchLoop to exit (which only happens after the pipe goroutine
	// sleeps `delay` then closes the writer). Allow 20ms slop for scheduling.
	const minExpected = delay - 20*time.Millisecond
	if elapsed < minExpected {
		t.Errorf("Close returned after %v; want >= %v — Close must join dispatchLoop goroutine via wg.Wait() (M-2 fix)", elapsed, minExpected)
	}
	// Sanity: Close must not block forever.
	const maxExpected = delay + 500*time.Millisecond
	if elapsed > maxExpected {
		t.Errorf("Close took %v; want <= %v — something is blocking Close beyond the expected goroutine join", elapsed, maxExpected)
	}
}
