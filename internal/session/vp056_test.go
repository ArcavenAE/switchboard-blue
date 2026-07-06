// Package session_test: integration-style discharge for VP-056 (console detach
// releases session without closing it — session and observers unaffected).
//
// VP-056 has four postconditions, and the pre-burst-3 suite only exercised
// subsets:
//
//   - (a) postcondition 1: C.downstream transitions to closed — covered by
//     TestSession_Detach_SessionContinues.
//   - (b) postcondition 2: session S remains active on the access node — only
//     indirectly implied by "re-attach a DIFFERENT console succeeds"; not
//     asserted against the Publisher's authoritative live set.
//   - (c) postcondition 5: observer O keeps receiving DOWNSTREAM frames after C
//     detaches — TestSession_Detach_ReadOnlyObserversUnaffected covers a single
//     frame; the property is delivery continuity (i.e. multiple frames across
//     the detach event without loss to the observer).
//   - (d) postcondition 6: the SAME console C can re-attach and resume delivery
//     — no pre-burst-3 test re-attaches with the same ConsoleKey; existing
//     re-attach tests use a different key, which does not prove the ConsoleSet
//     entry was fully released.
//
// This file closes (b), (c), and (d) with three focused integration tests
// wired against the real internal/session API (Publisher + AccessNode +
// ConsoleSet); no `session.Manager` / `session.NewFakeTransport` primitives
// are introduced because they do not exist on develop f09fe73 (the VP-056.md
// skeleton is aspirational). The AccessNode already exposes DeliverFrame as
// the downstream-injection primitive; injectDownstream is a thin, file-local
// helper matching the skeleton's naming without introducing a new production
// surface.
//
// Traces:
//   - BC-2.04.004 PC-1 (a — channel closed)
//   - BC-2.04.004 PC-2 + Invariant 1 (b — session non-destructive)
//   - BC-2.04.004 PC-5 / EC-001 (c — read-only observers continue receiving)
//   - BC-2.04.004 PC-6 (d — session available for new console to attach; the
//     stricter same-key re-attach here is a superset)
//
// VP-056
package session_test

import (
	"testing"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/session"
)

// newVP056Rig constructs a Publisher + AccessNode pair with sessionName
// pre-published. Returning both handles is what lets the VP-056 tests assert
// against the Publisher's authoritative live-set — AccessNode does not export
// a Publisher accessor, so tests wire the pair themselves. Mirrors the pattern
// in newTestAccessNode (session_test.go) but preserves the Publisher handle.
func newVP056Rig(t *testing.T, sessionName string) (*session.Publisher, *session.AccessNode) {
	t.Helper()
	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	if err := pub.Publish(sessionName); err != nil {
		t.Fatalf("newVP056Rig: Publish %q: %v", sessionName, err)
	}
	an := session.NewAccessNode(pub, session.NoOpAuthorizer{}, session.WithKeystrokeSink(session.NoOpSink{}))
	return pub, an
}

// injectDownstream is a file-local test helper that names the injection point
// per the VP-056.md proof-harness skeleton. It is a thin wrapper over
// AccessNode.DeliverFrame — the same primitive production code uses to fan out
// downstream frames to attached consoles. Introducing this helper (rather than
// a new production method) keeps ARCH-09 boundary discipline intact: no new
// side-effectful surface leaks into internal/session.
func injectDownstream(an *session.AccessNode, hdr frame.OuterHeader) {
	an.DeliverFrame(hdr)
}

// TestVP056_Detach_PublisherRetainsSession discharges postcondition 2 of
// BC-2.04.004: Detach is non-destructive. After a console detaches, the
// session name MUST remain in the Publisher's live set.
//
// Prior tests inferred this by re-attaching a different console; this test
// asserts it directly against Publisher.Exists — the authoritative live-set
// probe used by ConsoleServer (session.go:229).
//
// Mutation-kill self-check: if AccessNode.Detach were to call a.pub.Unpublish
// (e.g. because a code path incorrectly conflated console detach with session
// teardown), pub.Exists would return false and the test would fail.
//
// BC-2.04.004 PC-2 / Invariant 1 / VP-056 postcondition (b)
func TestVP056_Detach_PublisherRetainsSession(t *testing.T) {
	t.Parallel()
	pub, an := newVP056Rig(t, "build")

	if _, _, err := an.Attach("console-C", "build"); err != nil {
		t.Fatalf("Attach: %v", err)
	}

	// Pre-condition: session is in the live set.
	if !pub.Exists("build") {
		t.Fatal("pre-Detach: Publisher.Exists(\"build\") = false; want true")
	}

	if err := an.Detach("console-C", "build"); err != nil {
		t.Fatalf("Detach: unexpected error: %v", err)
	}

	// Post-condition (b): session MUST still exist. Detach is non-destructive.
	if !pub.Exists("build") {
		t.Error("post-Detach: Publisher.Exists(\"build\") = false; want true (BC-2.04.004 invariant 1: detach never terminates the session)")
	}

	// Cross-check with the authoritative list — the session name MUST be in the
	// snapshot returned by ListSessions (used by sbctl-side observers).
	sessions := pub.ListSessions()
	found := false
	for _, info := range sessions {
		if info.Name == "build" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("post-Detach: ListSessions() = %v; want to contain \"build\"", sessions)
	}
}

// TestVP056_Detach_ObserverContinuesReceivingMultipleFrames discharges
// postcondition 5 of BC-2.04.004 with delivery-continuity semantics. After a
// full-access console C detaches, the read-only observer O MUST continue to
// receive EVERY subsequent downstream frame — the fan-out set must be updated
// atomically with respect to the detach without dropping frames on the
// remaining consoles.
//
// Prior test TestSession_Detach_ReadOnlyObserversUnaffected asserts a single
// frame arrives at O after C detaches; this test asserts CONTINUITY — a burst
// of N frames all reach O. If the fan-out iterated over a stale snapshot that
// still included C's (now-closed) downstream channel, the send to C would
// consume the frame and drop it for O, or worse panic on send-to-closed
// channel; either failure surfaces as a hang or a lost frame here.
//
// Mutation-kill self-check: replace ConsoleSet.Remove's `delete(cs.consoles,
// key)` with a no-op (leaving C's entry in the map while its channel is
// closed) and Deliver would attempt to send on a closed channel — a panic
// under the RLock. Test fails.
//
// N=DownstreamBufSize is chosen to fully load the observer's buffered channel
// so a single dropped frame is detectable without racing the buffer.
//
// BC-2.04.004 PC-5 / EC-001 / VP-056 postcondition (c)
func TestVP056_Detach_ObserverContinuesReceivingMultipleFrames(t *testing.T) {
	t.Parallel()
	_, an := newVP056Rig(t, "monitor")

	if _, _, err := an.Attach("full-access-C", "monitor"); err != nil {
		t.Fatalf("Attach C: %v", err)
	}
	observerDownstream, _, err := an.Attach("observer-O", "monitor")
	if err != nil {
		t.Fatalf("Attach O: %v", err)
	}

	// Detach the full-access console; the observer stays.
	if err := an.Detach("full-access-C", "monitor"); err != nil {
		t.Fatalf("Detach C: %v", err)
	}

	// Post-detach: deliver a burst of frames. Every one must reach the
	// observer. Cap the burst at DownstreamBufSize so the observer's buffered
	// channel does not overflow (which would legitimately drop frames per
	// BC-2.04.006 NFR-004; that is not the property under test here).
	const burst = session.DownstreamBufSize
	for i := 0; i < burst; i++ {
		injectDownstream(an, frame.OuterHeader{
			Version:    frame.VersionByte,
			FrameType:  frame.FrameTypeData,
			PayloadLen: uint16(i + 1),
		})
	}

	// The observer's channel is buffered; drain synchronously up to burst.
	// A lost frame surfaces as either a shorter drain (channel empty before
	// burst is reached) or an out-of-order PayloadLen.
	for i := 0; i < burst; i++ {
		select {
		case got, ok := <-observerDownstream:
			if !ok {
				t.Fatalf("frame %d: observer downstream closed unexpectedly (session should be active per BC-2.04.004 PC-2)", i)
			}
			if got.PayloadLen != uint16(i+1) {
				t.Errorf("frame %d: PayloadLen = %d; want %d (delivery order lost)", i, got.PayloadLen, i+1)
			}
		default:
			t.Fatalf("frame %d: observer downstream empty; expected frame with PayloadLen %d (delivery continuity broken)", i, i+1)
		}
	}
}

// TestVP056_Detach_SameKeyReAttach_ResumesDelivery discharges postcondition 6
// of BC-2.04.004 with the stricter same-key re-attach property. After a
// console with key K detaches, an Attach call using the SAME key K MUST
// succeed and the new downstream channel MUST receive subsequent frames.
//
// This is a superset of PC-6 as literally worded ("session becomes available
// for a NEW console to attach"). The stricter test rules out a residual-entry
// bug in ConsoleSet where a closed downstream channel or a stale map entry
// prevents same-key re-attach — a bug the pre-burst-3 "re-attach with a
// different key" tests cannot catch because they never exercise the same key.
//
// Mutation-kill self-check: if ConsoleSet.Remove closed the downstream channel
// but omitted `delete(cs.consoles, key)`, the second Add would return
// ErrConsoleAlreadyAttached and the test would fail on the re-attach step.
//
// BC-2.04.004 PC-6 / VP-056 postcondition (d)
func TestVP056_Detach_SameKeyReAttach_ResumesDelivery(t *testing.T) {
	t.Parallel()
	_, an := newVP056Rig(t, "deploy")

	// First attach + detach.
	firstDownstream, _, err := an.Attach("console-K", "deploy")
	if err != nil {
		t.Fatalf("first Attach: %v", err)
	}
	if err := an.Detach("console-K", "deploy"); err != nil {
		t.Fatalf("Detach: unexpected error: %v", err)
	}

	// The first downstream channel MUST be closed.
	select {
	case _, ok := <-firstDownstream:
		if ok {
			t.Error("post-Detach: first downstream not closed; received value instead")
		}
	default:
		t.Error("post-Detach: first downstream not closed; default case reached (open and empty)")
	}

	// Re-attach with the SAME console key. If ConsoleSet retained the entry,
	// this fails with ErrConsoleAlreadyAttached (mutation-kill target).
	secondDownstream, _, err := an.Attach("console-K", "deploy")
	if err != nil {
		t.Fatalf("re-Attach same key: got %v; want nil (PC-6: session available for new console to attach)", err)
	}

	// Deliver a frame; the new downstream MUST receive it.
	want := frame.OuterHeader{
		Version:    frame.VersionByte,
		FrameType:  frame.FrameTypeData,
		PayloadLen: 42,
	}
	injectDownstream(an, want)

	select {
	case got, ok := <-secondDownstream:
		if !ok {
			t.Fatal("re-attach: second downstream closed unexpectedly before receiving frame")
		}
		if got.PayloadLen != want.PayloadLen {
			t.Errorf("re-attach: PayloadLen = %d; want %d", got.PayloadLen, want.PayloadLen)
		}
	default:
		t.Error("re-attach: second downstream empty; expected delivered frame (delivery did not resume)")
	}
}
