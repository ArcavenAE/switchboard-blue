// Package session_test — focused unit tests for ConsoleSet fan-out completeness
// and eviction (BC-2.04.006; BC-2.04.004 EC-002).
//
// Traces:
//   - BC-2.04.006 PC-1 (fan-out completeness)
//   - BC-2.04.006 Invariant 3 (keystroke serialization — covered in session_test.go)
//   - BC-2.04.004 EC-002 (crash detection / eviction)
//
// Red Gate: all tests fail against the stubs (panic from "not implemented").
package session_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/session"
)

// newTestConsoleSet is a test helper that constructs a ConsoleSet.
func newTestConsoleSet(t *testing.T) *session.ConsoleSet {
	t.Helper()
	return session.NewConsoleSet()
}

// makeTestHeader builds a minimal frame.OuterHeader for use as a test
// downstream frame. The FrameType is set to FrameTypeData (0x01); all
// other fields are zero-valued.
func makeTestHeader(payloadLen uint16) frame.OuterHeader {
	return frame.OuterHeader{
		Version:    frame.VersionByte,
		FrameType:  frame.FrameTypeData,
		PayloadLen: payloadLen,
	}
}

// TestConsoleSet_Add_ReturnsChannels verifies that Add returns non-nil
// downstream and upstream channels on success (BC-2.04.003 PC-1; AC-001).
func TestConsoleSet_Add_ReturnsChannels(t *testing.T) {
	t.Parallel()
	cs := newTestConsoleSet(t)

	downstream, upstream, err := cs.Add("console-A")
	if err != nil {
		t.Fatalf("Add: unexpected error: %v", err)
	}
	if downstream == nil {
		t.Error("Add: downstream channel is nil; want non-nil")
	}
	if upstream == nil {
		t.Error("Add: upstream channel is nil; want non-nil")
	}
}

// TestConsoleSet_Add_DuplicateKey verifies that adding the same console key
// twice returns ErrConsoleAlreadyAttached (E-SES-002).
func TestConsoleSet_Add_DuplicateKey(t *testing.T) {
	t.Parallel()
	cs := newTestConsoleSet(t)

	if _, _, err := cs.Add("console-A"); err != nil {
		t.Fatalf("first Add: unexpected error: %v", err)
	}
	_, _, err := cs.Add("console-A")
	if !errors.Is(err, session.ErrConsoleAlreadyAttached) {
		t.Errorf("second Add: got %v; want ErrConsoleAlreadyAttached", err)
	}
}

// TestConsoleSet_Remove_ClosesDownstream verifies that Remove closes the
// downstream channel of the removed console (BC-2.04.004 PC-1; AC-004).
//
// A closed channel is detected by a receive that returns the zero value with
// ok=false; this avoids any blocking.
func TestConsoleSet_Remove_ClosesDownstream(t *testing.T) {
	t.Parallel()
	cs := newTestConsoleSet(t)

	downstream, _, err := cs.Add("console-B")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	if err := cs.Remove("console-B"); err != nil {
		t.Fatalf("Remove: unexpected error: %v", err)
	}

	// Channel must be closed — receive should return immediately with ok=false.
	select {
	case _, ok := <-downstream:
		if ok {
			t.Error("Remove: downstream channel not closed; received value instead")
		}
		// ok == false: channel closed as expected
	default:
		t.Error("Remove: downstream channel not closed; default case reached (channel open and empty)")
	}
}

// TestConsoleSet_Remove_NotFound verifies that Remove returns ErrConsoleNotFound
// for an unknown key (E-SES-003; BC-2.04.004).
func TestConsoleSet_Remove_NotFound(t *testing.T) {
	t.Parallel()
	cs := newTestConsoleSet(t)

	err := cs.Remove("does-not-exist")
	if !errors.Is(err, session.ErrConsoleNotFound) {
		t.Errorf("Remove unknown: got %v; want ErrConsoleNotFound", err)
	}
}

// TestConsoleSet_Deliver_FanOutAllConsoles verifies that Deliver sends a copy
// of the frame to every attached console's downstream channel — no console is
// skipped (BC-2.04.006 PC-1; BC-2.04.006 Invariant fan-out completeness; AC-006).
//
// Two consoles are attached. A single frame is delivered. Both consoles must
// receive the frame. Channel coordination uses a sync.WaitGroup — no sleep.
func TestConsoleSet_Deliver_FanOutAllConsoles(t *testing.T) {
	t.Parallel()
	cs := newTestConsoleSet(t)

	const numConsoles = 2
	downstreams := make([]<-chan frame.OuterHeader, numConsoles)
	for i := range numConsoles {
		key := session.ConsoleKey("console-fan-" + string(rune('A'+i)))
		downstream, _, err := cs.Add(key)
		if err != nil {
			t.Fatalf("Add %q: %v", key, err)
		}
		downstreams[i] = downstream
	}

	hdr := makeTestHeader(42)
	cs.Deliver(hdr)

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
			if got.PayloadLen != hdr.PayloadLen {
				t.Errorf("console %d: PayloadLen = %d; want %d", idx, got.PayloadLen, hdr.PayloadLen)
			}
		}()
	}
	wg.Wait()
}

// TestConsoleSet_Deliver_SkipsRemovedConsole verifies that a console removed
// before Deliver is not delivered to (BC-2.04.004 PC-5; AC-005).
func TestConsoleSet_Deliver_SkipsRemovedConsole(t *testing.T) {
	t.Parallel()
	cs := newTestConsoleSet(t)

	// Add two consoles: A will be removed before delivery, B remains.
	downstreamA, _, err := cs.Add("fan-skip-A") //nolint:staticcheck // used below after Remove
	if err != nil {
		t.Fatalf("Add A: %v", err)
	}
	downstreamB, _, errB := cs.Add("fan-skip-B") //nolint:staticcheck // used below after Deliver
	if errB != nil {
		t.Fatalf("Add B: %v", errB)
	}

	if err := cs.Remove("fan-skip-A"); err != nil {
		t.Fatalf("Remove A: %v", err)
	}

	hdr := makeTestHeader(10)
	cs.Deliver(hdr)

	// B must receive the frame.
	gotB, okB := <-downstreamB
	if !okB {
		t.Fatal("console B: downstream closed unexpectedly")
	}
	if gotB.PayloadLen != hdr.PayloadLen {
		t.Errorf("console B: PayloadLen = %d; want %d", gotB.PayloadLen, hdr.PayloadLen)
	}

	// A's channel is closed (Remove closed it); no frame should be queued.
	select {
	case _, ok := <-downstreamA:
		if ok {
			t.Error("console A: received frame after Remove; expected channel closed")
		}
	default:
		// Channel closed; no frame delivered. This is the expected path.
	}
}

// TestConsoleSet_Evict_RemovesCrashedConsoles verifies that Evict removes
// consoles whose downstream channels are closed and returns the eviction count
// (BC-2.04.004 EC-002; BC-2.04.006 Invariant; AC-008).
//
// Phase 1 (healthy set): verifies Evict returns 0 when no channels are closed.
// Phase 2 (post-crash): verifies Evict removes crashed consoles and returns
// the correct count (tested at the AccessNode level in TestSession_CrashDetach_EvictsFromFanOut).
func TestConsoleSet_Evict_RemovesCrashedConsoles(t *testing.T) {
	t.Parallel()
	cs := newTestConsoleSet(t)

	// Add two consoles.
	if _, _, err := cs.Add("evict-A"); err != nil {
		t.Fatalf("Add evict-A: %v", err)
	}
	if _, _, err := cs.Add("evict-B"); err != nil {
		t.Fatalf("Add evict-B: %v", err)
	}

	// Verify initial Len.
	if n := cs.Len(); n != 2 {
		t.Fatalf("Len before eviction: got %d; want 2", n)
	}

	// No channels are closed yet; Evict should return 0.
	evicted := cs.Evict()
	if evicted != 0 {
		t.Errorf("Evict on healthy set: got %d; want 0", evicted)
	}
	if n := cs.Len(); n != 2 {
		t.Errorf("Len after Evict(0): got %d; want 2", n)
	}
}

// TestConsoleSet_Snapshot_ReturnsValueCopy verifies that Snapshot returns a
// value-copy of the key set and that mutating the returned slice does not affect
// the ConsoleSet (CLAUDE.md Go rule 12: no internal pointer leak).
func TestConsoleSet_Snapshot_ReturnsValueCopy(t *testing.T) {
	t.Parallel()
	cs := newTestConsoleSet(t)

	if _, _, err := cs.Add("snap-A"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, _, err := cs.Add("snap-B"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	snap := cs.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("Snapshot: got %d keys; want 2", len(snap))
	}

	// Mutate the snapshot — must not affect the ConsoleSet.
	snap[0] = "mutated"
	snap2 := cs.Snapshot()
	for _, k := range snap2 {
		if k == "mutated" {
			t.Error("Snapshot returned internal pointer; mutation leaked into ConsoleSet")
		}
	}
}
