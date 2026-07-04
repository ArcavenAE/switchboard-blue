// Package session tests for BC-2.04.001 PC-2 (session publication state).
// Traces: BC-2.04.001 PC-2, PC-3, PC-4; ADR-010; ARCH-08 §6.6 position 6.
// Red Gate: all tests below are designed to fail against the stub (todo() panic).
package session_test

import (
	"errors"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/session"
)

// newTestPublisher is a test helper that builds a Publisher backed by an empty
// AdmittedKeySet. Returns both so callers can register keys for admission-gated
// publish tests (BC-2.04.001 precondition 3; S-3.03 SessionAuth).
//
//nolint:unparam // AdmittedKeySet return used by S-3.03 admission-gated tests; signature intentional
func newTestPublisher(t *testing.T) (*session.Publisher, *admission.AdmittedKeySet) {
	t.Helper()
	keys := admission.NewAdmittedKeySet()
	p := session.NewPublisher(keys)
	return p, keys
}

// TestPublisher_Publish_AddsSessionToLiveSet verifies that Publish records
// the session name with a UTC timestamp (BC-2.04.001 PC-2; S-3.01a AC-002).
func TestPublisher_Publish_AddsSessionToLiveSet(t *testing.T) {
	t.Parallel()
	p, keys := newTestPublisher(t)
	_ = keys // available for S-3.03 admission-gated publish tests

	before := time.Now().UTC()
	if err := p.Publish("agent-01"); err != nil {
		t.Fatalf("Publish: unexpected error: %v", err)
	}
	after := time.Now().UTC()

	info, err := p.Get("agent-01")
	if err != nil {
		t.Fatalf("Get: unexpected error: %v", err)
	}
	if info.Name != "agent-01" {
		t.Errorf("Name = %q; want %q", info.Name, "agent-01")
	}
	if info.PublishedAt.Before(before) || info.PublishedAt.After(after) {
		t.Errorf("PublishedAt = %v; want between %v and %v", info.PublishedAt, before, after)
	}
}

// TestPublisher_Unpublish_RemovesFromLiveSet verifies that Unpublish removes a
// previously published session (BC-2.04.001 PC-4; S-3.01a AC-003).
func TestPublisher_Unpublish_RemovesFromLiveSet(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	if err := p.Publish("build"); err != nil {
		t.Fatalf("Publish: unexpected error: %v", err)
	}
	if err := p.Unpublish("build"); err != nil {
		t.Fatalf("Unpublish: unexpected error: %v", err)
	}

	_, err := p.Get("build")
	if !errors.Is(err, session.ErrSessionNotFound) {
		t.Errorf("Get after Unpublish: got %v; want ErrSessionNotFound", err)
	}
}

// TestPublisher_Unpublish_ErrSessionNotFound verifies that Unpublish returns
// ErrSessionNotFound for an unknown name (E-SES-001; BC-2.04.001).
func TestPublisher_Unpublish_ErrSessionNotFound(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	err := p.Unpublish("does-not-exist")
	if !errors.Is(err, session.ErrSessionNotFound) {
		t.Errorf("Unpublish missing: got %v; want ErrSessionNotFound", err)
	}
}

// TestPublisher_Publish_DuplicateReturnsAlreadyPublished verifies that
// publishing the same name twice returns ErrSessionAlreadyPublished
// (BC-2.04.001 invariant: canonical name uniqueness).
func TestPublisher_Publish_DuplicateReturnsAlreadyPublished(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	if err := p.Publish("agent-01"); err != nil {
		t.Fatalf("first Publish: unexpected error: %v", err)
	}
	err := p.Publish("agent-01")
	if !errors.Is(err, session.ErrSessionAlreadyPublished) {
		t.Errorf("second Publish: got %v; want ErrSessionAlreadyPublished", err)
	}
}

// TestPublisher_ListSessions_ReturnsSnapshot verifies that ListSessions returns
// all published sessions as a value copy (BC-2.04.001 PC-2; VP-031;
// ARCH-08 §6.6 rule 12: no internal pointer leak).
func TestPublisher_ListSessions_ReturnsSnapshot(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	for _, name := range []string{"agent-01", "agent-02", "build"} {
		if err := p.Publish(name); err != nil {
			t.Fatalf("Publish %q: %v", name, err)
		}
	}

	list := p.ListSessions()
	if len(list) != 3 {
		t.Fatalf("ListSessions: got %d sessions; want 3", len(list))
	}

	// Mutating the returned slice must not affect the publisher's internal state.
	list[0].Name = "mutated"
	list2 := p.ListSessions()
	if list2[0].Name == "mutated" {
		t.Error("ListSessions returned internal pointer; mutation leaked into publisher state")
	}
}

// TestPublisher_EmptyOnStartup verifies that a fresh publisher reports no
// sessions (BC-2.04.001 EC-003: tmux server has no sessions on startup).
func TestPublisher_EmptyOnStartup(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	list := p.ListSessions()
	if len(list) != 0 {
		t.Errorf("fresh Publisher: ListSessions returned %d sessions; want 0", len(list))
	}
}

// =============================================================================
// S-3.02: AccessNode attach/detach/fan-out tests (AC-001..AC-008)
// Traces: BC-2.04.003, BC-2.04.004, BC-2.04.006
// Red Gate: all tests below fail against the stubs (panic from "not implemented").
// =============================================================================

// newTestAccessNode is a test helper that builds an AccessNode backed by a
// Publisher seeded with the given sessionNames and a NoOpAuthorizer.
// A NoOpSink is injected so that tests that do not assert forwarding do not
// fail with ErrNoKeystrokeSink (F-L-2: the default sink is now fail-loud;
// tests must explicitly opt into discard or supply a recording sink).
func newTestAccessNode(t *testing.T, sessionNames ...string) *session.AccessNode {
	t.Helper()
	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	for _, name := range sessionNames {
		if err := pub.Publish(name); err != nil {
			t.Fatalf("newTestAccessNode: Publish %q: %v", name, err)
		}
	}
	return session.NewAccessNode(pub, session.NoOpAuthorizer{}, session.WithKeystrokeSink(session.NoOpSink{}))
}

// newTestAccessNodeWithSink builds an AccessNode with a specific KeystrokeSink.
// Use this when the test needs to assert what the sink received (F-C-1/F-C-2).
func newTestAccessNodeWithSink(t *testing.T, sink session.KeystrokeSink, sessionNames ...string) *session.AccessNode {
	t.Helper()
	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	for _, name := range sessionNames {
		if err := pub.Publish(name); err != nil {
			t.Fatalf("newTestAccessNodeWithSink: Publish %q: %v", name, err)
		}
	}
	return session.NewAccessNode(pub, session.NoOpAuthorizer{}, session.WithKeystrokeSink(sink))
}

// recordingSink is a KeystrokeSink that records every payload passed to
// SendInput for later assertion. It is goroutine-safe.
// Tests inject this via WithKeystrokeSink to verify the real forwarding path
// (F-C-1/F-C-2 pass-3: AC-002/AC-007 were tautological; the sink was never
// consulted).
type recordingSink struct {
	mu       sync.Mutex
	received [][]byte
	sendErr  error
}

// SendInput records a copy of payload and returns r.sendErr.
func (r *recordingSink) SendInput(payload []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sendErr != nil {
		return r.sendErr
	}
	cp := make([]byte, len(payload))
	copy(cp, payload)
	r.received = append(r.received, cp)
	return nil
}

// Received returns a snapshot copy of all recorded payloads.
func (r *recordingSink) Received() [][]byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([][]byte, len(r.received))
	copy(out, r.received)
	return out
}

// Len returns the number of recorded payloads.
func (r *recordingSink) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.received)
}

// TestSession_Attach_EstablishesBidirectionalChannel verifies that Attach
// returns non-nil downstream and upstream channels on success
// (AC-001; BC-2.04.003 PC-1).
func TestSession_Attach_EstablishesBidirectionalChannel(t *testing.T) {
	t.Parallel()
	an := newTestAccessNode(t, "agent-01")

	downstream, upstream, err := an.Attach("console-X", "agent-01")
	if err != nil {
		t.Fatalf("Attach: unexpected error: %v", err)
	}
	if downstream == nil {
		t.Error("Attach: downstream channel is nil; want non-nil")
	}
	if upstream == nil {
		t.Error("Attach: upstream channel is nil; want non-nil")
	}
}

// TestSession_Attach_UpstreamKeystrokesForwarded verifies that a keystroke sent
// via SendKeystroke reaches the injected KeystrokeSink (AC-002; BC-2.04.003
// PC-3; BC-2.04.006 Invariant 4).
//
// Pass-3 F-C-1 fix: the original test wrote to the upstream channel directly,
// which is buffered — it passed trivially without ever exercising the real
// forwarding path (AccessNode.SendKeystroke → sinkMu → sink.SendInput). This
// rewrite uses a recordingSink to assert that SendKeystroke calls through to
// the sink with the correct payload.
func TestSession_Attach_UpstreamKeystrokesForwarded(t *testing.T) {
	t.Parallel()
	rec := &recordingSink{}
	an := newTestAccessNodeWithSink(t, rec, "agent-02")

	if _, _, err := an.Attach("console-Y", "agent-02"); err != nil {
		t.Fatalf("Attach: %v", err)
	}

	payload := []byte("hello\r")
	if err := an.SendKeystroke("console-Y", "agent-02", payload); err != nil {
		t.Fatalf("SendKeystroke: unexpected error: %v", err)
	}

	got := rec.Received()
	if len(got) != 1 {
		t.Fatalf("recordingSink.Received(): got %d entries; want 1", len(got))
	}
	if string(got[0]) != string(payload) {
		t.Errorf("recordingSink.Received()[0] = %q; want %q", got[0], payload)
	}
}

// TestSession_Attach_NonexistentSession_ErrSesOne verifies that Attach returns
// ErrSessionNotFound when the named session does not exist
// (AC-003; BC-2.04.003 EC-002; E-SES-001).
func TestSession_Attach_NonexistentSession_ErrSesOne(t *testing.T) {
	t.Parallel()
	// AccessNode with no published sessions.
	an := newTestAccessNode(t)

	_, _, err := an.Attach("console-Z", "nonexistent")
	if !errors.Is(err, session.ErrSessionNotFound) {
		t.Errorf("Attach nonexistent: got %v; want ErrSessionNotFound", err)
	}
}

// TestSession_Detach_SessionContinues verifies that Detach closes the console's
// channel cleanly and that the session remains published (not torn down)
// (AC-004; BC-2.04.004 PC-1 + PC-2).
//
// "Session continues" is verified by confirming the session is still present in
// the publisher's live set after detach. No real tmux process is involved.
func TestSession_Detach_SessionContinues(t *testing.T) {
	t.Parallel()
	an := newTestAccessNode(t, "build")

	downstream, _, err := an.Attach("console-D", "build")
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}

	if err := an.Detach("console-D", "build"); err != nil {
		t.Fatalf("Detach: unexpected error: %v", err)
	}

	// The console's downstream channel must be closed after Detach.
	select {
	case _, ok := <-downstream:
		if ok {
			t.Error("Detach: downstream channel not closed; received value instead")
		}
		// ok == false: channel closed as expected
	default:
		t.Error("Detach: downstream channel not closed; default case reached (open and empty)")
	}

	// The session must still exist after Detach — Detach is non-destructive
	// (BC-2.04.004 invariant 1). Verified by re-attaching to the same session.
	// "build" was published in newTestAccessNode above.
	_, _, err2 := an.Attach("console-D2", "build")
	if err2 != nil {
		t.Errorf("post-Detach re-attach: got %v; want nil (session must still exist)", err2)
	}
}

// TestSession_SendKeystroke_AfterDetach_ReturnsErrConsoleNotFound verifies that
// SendKeystroke for a detached console returns ErrConsoleNotFound (F-H-2;
// BC-2.04.004 PC-3: no keystrokes forwarded after Detach).
func TestSession_SendKeystroke_AfterDetach_ReturnsErrConsoleNotFound(t *testing.T) {
	t.Parallel()
	rec := &recordingSink{}
	an := newTestAccessNodeWithSink(t, rec, "detach-session")

	if _, _, err := an.Attach("console-detach", "detach-session"); err != nil {
		t.Fatalf("Attach: %v", err)
	}

	// Verify SendKeystroke works before detach.
	if err := an.SendKeystroke("console-detach", "detach-session", []byte("pre")); err != nil {
		t.Fatalf("SendKeystroke pre-Detach: unexpected error: %v", err)
	}

	if err := an.Detach("console-detach", "detach-session"); err != nil {
		t.Fatalf("Detach: unexpected error: %v", err)
	}

	// Post-Detach: SendKeystroke must return ErrConsoleNotFound.
	err := an.SendKeystroke("console-detach", "detach-session", []byte("post"))
	if !errors.Is(err, session.ErrConsoleNotFound) {
		t.Errorf("SendKeystroke post-Detach: got %v; want ErrConsoleNotFound", err)
	}

	// The recording sink must not have received the post-detach keystroke.
	if rec.Len() != 1 {
		t.Errorf("recordingSink: got %d entries; want 1 (only pre-detach send)", rec.Len())
	}
}

// TestSession_SendKeystroke_WrongSession_ReturnsErrSessionMismatch verifies
// that SendKeystroke returns ErrSessionMismatch when the console is attached
// to a different session than the one named in the call (F-H-2).
func TestSession_SendKeystroke_WrongSession_ReturnsErrSessionMismatch(t *testing.T) {
	t.Parallel()
	rec := &recordingSink{}
	an := newTestAccessNodeWithSink(t, rec, "session-1", "session-2")

	if _, _, err := an.Attach("console-A", "session-1"); err != nil {
		t.Fatalf("Attach: %v", err)
	}

	err := an.SendKeystroke("console-A", "session-2", []byte("payload"))
	if !errors.Is(err, session.ErrSessionMismatch) {
		t.Errorf("SendKeystroke wrong session: got %v; want ErrSessionMismatch", err)
	}

	if rec.Len() != 0 {
		t.Errorf("recordingSink: got %d entries; want 0 (no send on mismatch)", rec.Len())
	}
}

// TestSession_Detach_ReadOnlyObserversUnaffected verifies that after a
// full-access console detaches, a second console (read-only observer) that is
// still attached continues receiving downstream frames unaffected
// (AC-005; BC-2.04.004 PC-5; BC-2.04.004 EC-001).
func TestSession_Detach_ReadOnlyObserversUnaffected(t *testing.T) {
	t.Parallel()
	an := newTestAccessNode(t, "monitor")

	// Attach console A (full-access) and console B (observer).
	_, _, err := an.Attach("full-access", "monitor")
	if err != nil {
		t.Fatalf("Attach full-access: %v", err)
	}
	observerDownstream, _, err := an.Attach("observer", "monitor")
	if err != nil {
		t.Fatalf("Attach observer: %v", err)
	}

	// Detach the full-access console.
	if err := an.Detach("full-access", "monitor"); err != nil {
		t.Fatalf("Detach full-access: %v", err)
	}

	// Deliver a frame — the observer must still receive it.
	an.DeliverFrame(frame.OuterHeader{
		Version:    frame.VersionByte,
		FrameType:  frame.FrameTypeData,
		PayloadLen: 7,
	})

	got, ok := <-observerDownstream
	if !ok {
		t.Fatal("observer: downstream closed unexpectedly after full-access detach")
	}
	if got.PayloadLen != 7 {
		t.Errorf("observer: PayloadLen = %d; want 7", got.PayloadLen)
	}
}

// TestSession_MultiConsoleFanOut_AllReceiveFrames verifies that two consoles
// attached simultaneously each receive all downstream frames independently
// (AC-006; BC-2.04.006 PC-1; invariant: fan-out completeness).
//
// Synchronization uses channels — no sleep.
func TestSession_MultiConsoleFanOut_AllReceiveFrames(t *testing.T) {
	t.Parallel()
	an := newTestAccessNode(t, "fleet")

	const numConsoles = 3
	downstreams := make([]<-chan frame.OuterHeader, numConsoles)
	for i := range numConsoles {
		key := session.ConsoleKey("fan-console-" + strconv.Itoa(i))
		downstream, _, err := an.Attach(key, "fleet")
		if err != nil {
			t.Fatalf("Attach %q: %v", key, err)
		}
		downstreams[i] = downstream
	}

	want := frame.OuterHeader{
		Version:    frame.VersionByte,
		FrameType:  frame.FrameTypeData,
		PayloadLen: 99,
	}
	an.DeliverFrame(want)

	var wg sync.WaitGroup
	wg.Add(numConsoles)
	for i, downstream := range downstreams {
		idx := i
		ch := downstream
		go func() {
			defer wg.Done()
			got, ok := <-ch
			if !ok {
				t.Errorf("console %d: downstream closed unexpectedly", idx)
				return
			}
			if got.PayloadLen != want.PayloadLen {
				t.Errorf("console %d: PayloadLen = %d; want %d", idx, got.PayloadLen, want.PayloadLen)
			}
		}()
	}
	wg.Wait()
}

// contentionSink is a KeystrokeSink that detects concurrent entry into
// SendInput. It has NO internal lock — the serialization guarantee must come
// entirely from the caller (AccessNode.sinkMu). The in-flight counter is
// accessed via atomic ops so the test logic itself is race-free, but if two
// goroutines enter simultaneously the counter exceeds 1 and violated is set.
// runtime.Gosched() widens the concurrency window: without an outer lock,
// goroutines preempted between AddInt64(+1) and AddInt64(-1) will overlap.
//
// With AccessNode.sinkMu in place: calls are serialized; the counter is always
// exactly 1 during a call; `go test -race` stays clean.
// With AccessNode.sinkMu removed: goroutines overlap; violated is set to 1
// and `go test -race` detects unsynchronized access.
type contentionSink struct {
	inFlight int64 // number of goroutines currently inside SendInput
	maxSeen  int64 // high-water mark of concurrent in-flight calls
	calls    int64 // total completed calls
	violated int32 // 1 if inFlight ever exceeded 1
}

// SendInput implements KeystrokeSink using an unsynchronized in-flight counter.
func (s *contentionSink) SendInput(_ []byte) error {
	cur := atomic.AddInt64(&s.inFlight, 1)
	for {
		prev := atomic.LoadInt64(&s.maxSeen)
		if cur <= prev || atomic.CompareAndSwapInt64(&s.maxSeen, prev, cur) {
			break
		}
	}
	if cur > 1 {
		atomic.StoreInt32(&s.violated, 1)
	}
	// Yield so that any other goroutine waiting outside the (absent) lock can
	// enter and overlap with this call.
	runtime.Gosched()
	atomic.AddInt64(&s.inFlight, -1)
	atomic.AddInt64(&s.calls, 1)
	return nil
}

// TestSession_ConcurrentKeystrokes_Serialized verifies that keystrokes from N
// consoles sent concurrently are serialized at the sinkMu boundary in
// AccessNode.SendKeystroke (AC-007; BC-2.04.006 Invariant 3).
//
// Pass-5 F-H-1 fix: earlier passes used recordingSink whose internal r.mu lock
// made the test tautological — it passed whether or not AccessNode.sinkMu
// existed because the sink's own lock masked the missing outer lock.
//
// This rewrite injects a contentionSink (see type above) whose SendInput has
// NO lock. It detects concurrent entry via an atomic in-flight counter. With
// AccessNode.sinkMu present, calls are serialized and the counter never
// exceeds 1. With sinkMu absent, goroutines overlap: the counter exceeds 1
// (violated is set) and `go test -race` fires on the unsynchronized counter
// access.
func TestSession_ConcurrentKeystrokes_Serialized(t *testing.T) {
	t.Parallel()

	const numConsoles = 4
	const sendsPerConsole = 100

	sink := &contentionSink{}
	an := newTestAccessNodeWithSink(t, sink, "shared")

	consoleKeys := make([]session.ConsoleKey, numConsoles)
	for i := range numConsoles {
		key := session.ConsoleKey("writer-" + strconv.Itoa(i))
		consoleKeys[i] = key
		if _, _, err := an.Attach(key, "shared"); err != nil {
			t.Fatalf("Attach %q: %v", key, err)
		}
	}

	var wg sync.WaitGroup
	wg.Add(numConsoles)
	payload := []byte("k")
	for i := range numConsoles {
		idx := i
		go func() {
			defer wg.Done()
			for range sendsPerConsole {
				if err := an.SendKeystroke(consoleKeys[idx], "shared", payload); err != nil {
					t.Errorf("SendKeystroke %q: %v", consoleKeys[idx], err)
				}
			}
		}()
	}
	wg.Wait()

	total := int64(numConsoles * sendsPerConsole)
	if got := atomic.LoadInt64(&sink.calls); got != total {
		t.Fatalf("contentionSink: got %d calls; want %d", got, total)
	}
	if atomic.LoadInt32(&sink.violated) != 0 {
		t.Fatalf("contentionSink: concurrent entry detected (maxSeen=%d); sinkMu serialization broken (AC-007 / BC-2.04.006 Inv-3)", atomic.LoadInt64(&sink.maxSeen))
	}
}

// TestSession_CrashDetach_EvictsFromFanOut verifies the keepalive crash
// detection path (AC-008; BC-2.04.004 EC-002): a console that misses its
// heartbeat deadline is evicted by Sweep, and subsequent SendKeystroke calls
// for that console return ErrConsoleNotFound. The surviving console is unaffected.
//
// Uses a fake clock (injected via WithClock) for deterministic selective eviction
// without sleeps (F-C-1/F-C-2/F-H-3).
func TestSession_CrashDetach_EvictsFromFanOut(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	fakeNow := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return fakeNow
	}
	advance := func(d time.Duration) {
		mu.Lock()
		defer mu.Unlock()
		fakeNow = fakeNow.Add(d)
	}

	rec := &recordingSink{}
	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	if err := pub.Publish("crash-session"); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	an := session.NewAccessNode(pub, session.NoOpAuthorizer{},
		session.WithKeystrokeSink(rec),
		session.WithClock(clock),
	)

	// T=0: attach both consoles.
	victimDownstream, _, err := an.Attach("crash-victim", "crash-session")
	if err != nil {
		t.Fatalf("Attach crash-victim: %v", err)
	}
	survivorDownstream, _, err := an.Attach("survivor", "crash-session")
	if err != nil {
		t.Fatalf("Attach survivor: %v", err)
	}

	// Advance to T=2s, heartbeat only survivor.
	advance(2 * time.Second)
	if err := an.Heartbeat("survivor"); err != nil {
		t.Fatalf("Heartbeat survivor: %v", err)
	}

	// Advance to T=4s. Sweep with 3s deadline:
	// cutoff = T=4s - 3s = T=1s.
	// victim.lastHeartbeat = T=0 < T=1s → evicted.
	// survivor.lastHeartbeat = T=2s >= T=1s → survives.
	advance(2 * time.Second)
	evicted := an.Sweep(3 * time.Second)
	if evicted != 1 {
		t.Errorf("Sweep(3s): evicted %d; want 1 (only victim evicted)", evicted)
	}

	// victim's downstream channel must be closed.
	select {
	case _, ok := <-victimDownstream:
		if ok {
			t.Error("victim: downstream channel open after eviction; want closed")
		}
	default:
		t.Error("victim: downstream channel not closed after eviction")
	}

	// survivor's downstream must still be open: deliver a frame and receive it.
	an.DeliverFrame(frame.OuterHeader{
		Version:    frame.VersionByte,
		FrameType:  frame.FrameTypeData,
		PayloadLen: 5,
	})
	select {
	case got, ok := <-survivorDownstream:
		if !ok {
			t.Fatal("survivor: downstream closed unexpectedly")
		}
		if got.PayloadLen != 5 {
			t.Errorf("survivor: PayloadLen = %d; want 5", got.PayloadLen)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("survivor: no frame received within 100ms")
	}

	// SendKeystroke to victim returns ErrConsoleNotFound.
	err = an.SendKeystroke("crash-victim", "crash-session", []byte("x"))
	if !errors.Is(err, session.ErrConsoleNotFound) {
		t.Errorf("SendKeystroke post-eviction victim: got %v; want ErrConsoleNotFound", err)
	}

	// SendKeystroke to survivor succeeds.
	if err := an.SendKeystroke("survivor", "crash-session", []byte("y")); err != nil {
		t.Errorf("SendKeystroke survivor: got %v; want nil", err)
	}
}

// TestAccessNode_SendKeystroke_NoSink_ReturnsError verifies that an AccessNode
// constructed WITHOUT a WithKeystrokeSink option returns ErrNoKeystrokeSink on
// every SendKeystroke call (F-L-3 pass-7; anti-silent-failure guard).
//
// noSink{} is the default sink wired in NewAccessNode. This test proves the
// fail-loud path fires so that production callers that forget to inject a real
// sink fail visibly rather than silently discarding keystrokes.
//
// newTestAccessNode injects NoOpSink to avoid this error in all other tests;
// here we bypass that helper and construct the AccessNode without any sink
// option to exercise the default path.
func TestAccessNode_SendKeystroke_NoSink_ReturnsError(t *testing.T) {
	t.Parallel()

	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	if err := pub.Publish("nosink-session"); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	// Deliberately omit WithKeystrokeSink — noSink{} default must fire.
	an := session.NewAccessNode(pub, session.NoOpAuthorizer{})

	if _, _, err := an.Attach("nosink-console", "nosink-session"); err != nil {
		t.Fatalf("Attach: %v", err)
	}

	err := an.SendKeystroke("nosink-console", "nosink-session", []byte("x"))
	if !errors.Is(err, session.ErrNoKeystrokeSink) {
		t.Errorf("SendKeystroke with no sink: got %v; want ErrNoKeystrokeSink", err)
	}
}
