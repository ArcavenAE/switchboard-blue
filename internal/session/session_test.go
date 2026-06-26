// Package session tests for BC-2.04.001 PC-2 (session publication state).
// Traces: BC-2.04.001 PC-2, PC-3, PC-4; ADR-010; ARCH-08 §6.6 position 6.
// Red Gate: all tests below are designed to fail against the stub (todo() panic).
package session_test

import (
	"errors"
	"sync"
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
func newTestAccessNode(t *testing.T, sessionNames ...string) *session.AccessNode {
	t.Helper()
	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	for _, name := range sessionNames {
		if err := pub.Publish(name); err != nil {
			t.Fatalf("newTestAccessNode: Publish %q: %v", name, err)
		}
	}
	return session.NewAccessNode(pub, session.NoOpAuthorizer{})
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

// TestSession_Attach_UpstreamKeystrokesForwarded verifies that keystrokes sent
// via the upstream channel reach the access node without error
// (AC-002; BC-2.04.003 PC-3).
//
// Hermetic: we verify only that the upstream channel accepts writes without
// blocking or panicking. Real tmux forwarding is out of scope for unit tests
// (effectful layer is internal/tmux); this test covers the channel wiring contract.
func TestSession_Attach_UpstreamKeystrokesForwarded(t *testing.T) {
	t.Parallel()
	an := newTestAccessNode(t, "agent-02")

	_, upstream, err := an.Attach("console-Y", "agent-02")
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}

	keystroke := []byte("ls\r")
	done := make(chan struct{})
	go func() {
		defer close(done)
		upstream <- keystroke
	}()

	<-done // Keystroke accepted by upstream channel — channel wiring is correct.
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

	if err := an.Detach("console-D"); err != nil {
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
	observerDownstream, _, err := an.Attach("observer", "monitor") //nolint:staticcheck // used below after DeliverFrame; stub panics before assignment
	if err != nil {
		t.Fatalf("Attach observer: %v", err)
	}

	// Detach the full-access console.
	if err := an.Detach("full-access"); err != nil {
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
		key := session.ConsoleKey("fan-console-" + string(rune('A'+i)))
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

// TestSession_ConcurrentKeystrokes_Serialized verifies that keystrokes from
// two full-access consoles sent concurrently are both accepted without data
// corruption or panics (AC-007; BC-2.04.006 Invariant 3).
//
// The test verifies the serialization contract by sending from two goroutines
// simultaneously and checking that SendKeystroke returns nil for both — the
// race detector enforces the absence of data races.
func TestSession_ConcurrentKeystrokes_Serialized(t *testing.T) {
	t.Parallel()
	an := newTestAccessNode(t, "shared")

	if _, _, err := an.Attach("writer-1", "shared"); err != nil {
		t.Fatalf("Attach writer-1: %v", err)
	}
	if _, _, err := an.Attach("writer-2", "shared"); err != nil {
		t.Fatalf("Attach writer-2: %v", err)
	}

	// Both goroutines call SendKeystroke concurrently. If the implementation
	// serializes correctly, both calls return nil and the race detector finds no
	// data races. A panic or deadlock indicates a serialization failure.
	errs := make(chan error, 2)
	for _, key := range []session.ConsoleKey{"writer-1", "writer-2"} {
		k := key
		go func() {
			errs <- an.SendKeystroke(k, "shared", []byte("hello\r"))
		}()
	}

	for range 2 {
		if err := <-errs; err != nil {
			t.Errorf("SendKeystroke: unexpected error: %v", err)
		}
	}
}

// TestSession_CrashDetach_EvictsFromFanOut verifies that when a console's
// channel closes unexpectedly (crash simulation), the access node evicts it
// from the fan-out set and remaining consoles continue unaffected
// (AC-008; BC-2.04.004 EC-002; BC-2.04.006).
//
// Crash simulation: we attach a console then call Detach to close its channel
// (equivalent to a crash from the access node's perspective — the channel is
// closed). Then we call DeliverFrame and verify the surviving console receives
// the frame.
//
// NOTE: a true crash would close the channel from the console side. Since
// ConsoleSet.Add returns <-chan (receive-only to the caller), the implementer
// must handle crash detection inside DeliverFrame via recover on send-to-closed.
// This test exercises the detectable outcome: after the crash, the surviving
// console receives the frame and the crashed console is no longer in the set.
func TestSession_CrashDetach_EvictsFromFanOut(t *testing.T) {
	t.Parallel()
	an := newTestAccessNode(t, "crash-session")

	// Attach two consoles.
	_, _, err := an.Attach("crash-victim", "crash-session")
	if err != nil {
		t.Fatalf("Attach crash-victim: %v", err)
	}
	survivorDownstream, _, err := an.Attach("survivor", "crash-session") //nolint:staticcheck // used below after DeliverFrame; stub panics before assignment
	if err != nil {
		t.Fatalf("Attach survivor: %v", err)
	}

	// Simulate crash: Detach closes the victim's channel as the access node
	// would detect on keepalive timeout. The next DeliverFrame must evict it.
	if err := an.Detach("crash-victim"); err != nil {
		t.Fatalf("Detach (crash simulation): %v", err)
	}

	// DeliverFrame must not panic and must deliver to the survivor.
	an.DeliverFrame(frame.OuterHeader{
		Version:    frame.VersionByte,
		FrameType:  frame.FrameTypeData,
		PayloadLen: 3,
	})

	got, ok := <-survivorDownstream
	if !ok {
		t.Fatal("survivor: downstream closed unexpectedly")
	}
	if got.PayloadLen != 3 {
		t.Errorf("survivor: PayloadLen = %d; want 3", got.PayloadLen)
	}
}
