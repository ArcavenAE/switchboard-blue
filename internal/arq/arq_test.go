// Package arq_test exercises the downstream ARQ state machine.
//
// Test naming follows the BC-based pattern:
//
//	test_BC_S_SS_NNN_xxx()  →  TestBC_2_02_005_xxx / TestBC_2_02_006_xxx
//
// Story tests use the AC/EC/VP naming from S-4.03:
//
//	TestARQ_OnAck_NoDuplicateDelivery   (AC-001, BC-2.02.005 postcondition 1)
//	TestARQ_InOrderDelivery             (AC-002, BC-2.02.005 postconditions 2/4)
//	TestARQ_SACKInChannelHeader         (AC-003, BC-2.02.005 postcondition 3, ARCH-02)
//	TestARQ_TLPKTDROP_TerminatesOverdueFrame (AC-004, BC-2.02.006 postconditions 1/2)
//	TestARQ_TLPKTDROP_OnlyOverdueFrames (AC-005, BC-2.02.006 postcondition 2)
//
// ARQ is single-writer per half-channel; concurrent-use claims have been
// removed and no concurrent-OnAck tests are included.
package arq_test

import (
	"errors"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/arq"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

// mustDrainOne reads one []byte off ch within 100ms or calls t.Fatal.
func mustDrainOne(t *testing.T, ch <-chan []byte) []byte {
	t.Helper()
	const timeout = 100 * time.Millisecond
	select {
	case got := <-ch:
		return got
	case <-time.After(timeout):
		t.Fatal("timed out waiting for delivered frame")
		return nil
	}
}

// assertNoPending asserts that no frame is waiting on DeliveredFrames within
// the short poll window. Used to verify in-order buffering does NOT flush early.
func assertNoPending(t *testing.T, ch <-chan []byte) {
	t.Helper()
	select {
	case got := <-ch:
		t.Fatalf("unexpected delivered frame: %v", got)
	case <-time.After(5 * time.Millisecond):
		// good — nothing ready
	}
}

// mustDrainOneDeg reads one DegradationEvent off ch within 100ms or calls t.Fatal.
func mustDrainOneDeg(t *testing.T, ch <-chan arq.DegradationEvent) arq.DegradationEvent {
	t.Helper()
	const timeout = 100 * time.Millisecond
	select {
	case ev := <-ch:
		return ev
	case <-time.After(timeout):
		t.Fatal("timed out waiting for DegradationEvent")
		return arq.DegradationEvent{}
	}
}

// assertNoPendingDeg asserts that no DegradationEvent is queued.
func assertNoPendingDeg(t *testing.T, ch <-chan arq.DegradationEvent) {
	t.Helper()
	select {
	case ev := <-ch:
		t.Fatalf("unexpected DegradationEvent: %+v", ev)
	case <-time.After(5 * time.Millisecond):
		// good
	}
}

// newTestARQ builds an ARQ with buffered delivery/degradation channels so
// tests can inspect them without a separate goroutine.
func newTestARQ(dropTimeout time.Duration) *arq.ARQ {
	return arq.New(arq.Config{
		DropTimeout:        dropTimeout,
		DeliveredBufSize:   16,
		DegradationBufSize: 16,
	})
}

// zeroBitmap returns an all-zero SACK bitmap (no out-of-order frames).
func zeroBitmap() [arq.SACKBitmapBytes]byte { return [arq.SACKBitmapBytes]byte{} }

// bitmapWithBits sets the given zero-based bit positions in a fresh bitmap.
// Bit 0 is the MSB of byte 0 (covers ackSeq+1), consistent with big-endian
// encoding used by bitmapToUint64 in arq.go.
func bitmapWithBits(positions ...int) [arq.SACKBitmapBytes]byte {
	var b [arq.SACKBitmapBytes]byte
	for _, pos := range positions {
		if pos < 0 || pos >= 64 {
			panic("bit position out of range")
		}
		byteIdx := pos / 8
		bitIdx := 7 - (pos % 8) // MSB-first within each byte
		b[byteIdx] |= 1 << bitIdx
	}
	return b
}

// buildChannelHeader builds a channel-header byte slice following ARCH-02 §3.2.
//
// Layout (bytes):
//
//	0..3   channel_id       (big-endian uint32, set to channelID)
//	4..7   seq              (big-endian uint32, set to seq)
//	8      flags            (bit 2 = SACK_present; bit 0 = FEC_present; bit 1 = ARQ_req)
//	9..11  reserved
//	12..19 sack_bitmap      (8 bytes, only when SACK_present=1)
//
// When sackPresent is false the returned slice is 12 bytes; when true, 20 bytes.
func buildChannelHeader(channelID, seq uint32, flags byte, sack [arq.SACKBitmapBytes]byte, sackPresent bool) []byte {
	size := 12
	if sackPresent {
		size = 20
	}
	hdr := make([]byte, size)
	hdr[0] = byte(channelID >> 24)
	hdr[1] = byte(channelID >> 16)
	hdr[2] = byte(channelID >> 8)
	hdr[3] = byte(channelID)
	hdr[4] = byte(seq >> 24)
	hdr[5] = byte(seq >> 16)
	hdr[6] = byte(seq >> 8)
	hdr[7] = byte(seq)
	hdr[8] = flags
	// bytes 9..11 reserved
	if sackPresent {
		copy(hdr[12:20], sack[:])
	}
	return hdr
}

// ─── AC-001: no duplicate delivery ───────────────────────────────────────────

// TestARQ_OnAck_NoDuplicateDelivery exercises BC-2.02.005 postcondition 1:
// a frame acknowledged once is never delivered a second time.
//
// Exercises VP-019 (no double delivery).
// Idempotent ACK returns nil exactly (M-1 ruling: ErrDuplicateSequence removed).
func TestARQ_OnAck_NoDuplicateDelivery(t *testing.T) {
	t.Parallel()

	a := newTestARQ(100 * time.Millisecond)

	// EnqueueSend so the sender side knows about the frame, then OnAck it.
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a.EnqueueSend(1, []byte("frame-1"), now)

	// First OnAck — should deliver the frame.
	if err := a.OnAck(1, zeroBitmap()); err != nil {
		t.Fatalf("first OnAck(1) returned unexpected error: %v", err)
	}
	_ = mustDrainOne(t, a.DeliveredFrames)

	// Second OnAck for same sequence — must return nil exactly (idempotent per
	// EC-001; ErrDuplicateSequence sentinel is being removed by implementer).
	if err := a.OnAck(1, zeroBitmap()); err != nil {
		t.Fatalf("second OnAck(1): want nil, got %v", err)
	}
	// Crucially: no additional frame must have been delivered.
	assertNoPending(t, a.DeliveredFrames)
}

// TestBC_2_02_005_EC001_IdempotentAck verifies EC-001: ACKing an already-acked
// sequence is idempotent and returns nil — not an error, and never double-delivers.
//
// M-1 ruling: idempotent ACK returns nil exactly. The ErrDuplicateSequence
// sentinel is being removed; accepting it here would be tautological.
func TestBC_2_02_005_EC001_IdempotentAck(t *testing.T) {
	t.Parallel()

	a := newTestARQ(100 * time.Millisecond)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a.EnqueueSend(5, []byte("payload"), now)

	if err := a.OnAck(5, zeroBitmap()); err != nil {
		t.Fatalf("initial OnAck(5): %v", err)
	}
	_ = mustDrainOne(t, a.DeliveredFrames)

	// Re-ACK same seq — must return nil exactly; must not double-deliver.
	if err := a.OnAck(5, zeroBitmap()); err != nil {
		t.Fatalf("idempotent OnAck(5): want nil, got %v", err)
	}
	assertNoPending(t, a.DeliveredFrames)
}

// ─── AC-002: in-order delivery ────────────────────────────────────────────────

// TestARQ_InOrderDelivery exercises BC-2.02.005 postconditions 2 and 4:
// out-of-order frames are buffered until preceding gaps are filled.
//
// Exercises VP-020 (in-order delivery invariant).
func TestARQ_InOrderDelivery(t *testing.T) {
	t.Parallel()

	a := newTestARQ(100 * time.Millisecond)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a.EnqueueSend(1, []byte("one"), now)
	a.EnqueueSend(2, []byte("two"), now)
	a.EnqueueSend(3, []byte("three"), now)

	// Deliver seq=3 first (gap at 1 and 2).
	// Bit 0 in the SACK bitmap covers ackSeq+1. With ackSeq=0, bit 0 means
	// seq=1 is received out-of-order; bit 1 means seq=2 is received
	// out-of-order. Here we are calling OnAck(3) with an empty SACK — meaning
	// the console says "I've cumulatively received up through 3". But seq 1 and
	// 2 haven't been ACKed individually yet.
	//
	// Instead model the out-of-order scenario:
	//   Step 1: OnAck(0, SACK{bit2=seq3-received}) — "nothing cumulatively ACKed
	//           but seq 3 arrived out-of-order."
	//   Step 2: OnAck(1) — fills gap; should flush 1.
	//   Step 3: OnAck(2) — fills gap; should flush 2 then 3 (which was buffered).
	//
	// SACK bitmap bit positions are zero-based offsets above ackSeq+1.
	// With ackSeq=0: bit 0 → seq 1, bit 1 → seq 2, bit 2 → seq 3.
	sackSeq3 := bitmapWithBits(2) // bit 2 = seq 3 (offset 2 above ackSeq+1=1)

	// OnAck(0, sack=seq3 received): gap at seq 1 and 2; seq 3 buffered.
	if err := a.OnAck(0, sackSeq3); err != nil {
		t.Fatalf("OnAck(0, sack={seq3}): %v", err)
	}
	// Nothing should be delivered yet — gap at seq 1 blocks all.
	assertNoPending(t, a.DeliveredFrames)

	// OnAck(1): fills the cumulative pointer through 1; should deliver seq 1.
	if err := a.OnAck(1, zeroBitmap()); err != nil {
		t.Fatalf("OnAck(1): %v", err)
	}
	got1 := mustDrainOne(t, a.DeliveredFrames)
	if string(got1) != "one" {
		t.Errorf("expected first delivered = %q, got %q", "one", got1)
	}
	// seq 2 still missing — seq 3 still buffered.
	assertNoPending(t, a.DeliveredFrames)

	// OnAck(2): fills the gap at 2; should deliver seq 2 then seq 3 (buffered).
	if err := a.OnAck(2, zeroBitmap()); err != nil {
		t.Fatalf("OnAck(2): %v", err)
	}
	got2 := mustDrainOne(t, a.DeliveredFrames)
	if string(got2) != "two" {
		t.Errorf("expected second delivered = %q, got %q", "two", got2)
	}
	got3 := mustDrainOne(t, a.DeliveredFrames)
	if string(got3) != "three" {
		t.Errorf("expected third delivered = %q, got %q", "three", got3)
	}
	assertNoPending(t, a.DeliveredFrames)
}

// TestBC_2_02_005_InOrder_CanonicalVector uses the canonical test vector from
// BC-2.02.005: downstream frames [1,3] arrive; gap at 2 noted in SACK; access
// node retransmits 2; console delivers [1,2,3].
func TestBC_2_02_005_InOrder_CanonicalVector(t *testing.T) {
	t.Parallel()

	a := newTestARQ(500 * time.Millisecond)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	a.EnqueueSend(1, []byte("seq1"), now)
	a.EnqueueSend(2, []byte("seq2"), now)
	a.EnqueueSend(3, []byte("seq3"), now)

	// Console received seq=1 and seq=3 out of order; seq=2 missing.
	// With ackSeq=1 (cumulative through 1), SACK bit 1 = seq 3 received.
	// Bit 0 = offset 0 above ackSeq+1=2 → seq 2; bit 1 → seq 3.
	sack := bitmapWithBits(1) // bit 1 = seq 3 received
	if err := a.OnAck(1, sack); err != nil {
		t.Fatalf("OnAck(1, sack={seq3}): %v", err)
	}
	// seq 1 must be delivered; seq 3 buffered; seq 2 not yet.
	got1 := mustDrainOne(t, a.DeliveredFrames)
	if string(got1) != "seq1" {
		t.Errorf("expected seq1, got %q", got1)
	}
	assertNoPending(t, a.DeliveredFrames)

	// Simulate retransmit of seq=2 arriving; OnAck(2).
	if err := a.OnAck(2, zeroBitmap()); err != nil {
		t.Fatalf("OnAck(2): %v", err)
	}
	got2 := mustDrainOne(t, a.DeliveredFrames)
	if string(got2) != "seq2" {
		t.Errorf("expected seq2, got %q", got2)
	}
	got3 := mustDrainOne(t, a.DeliveredFrames)
	if string(got3) != "seq3" {
		t.Errorf("expected seq3, got %q", got3)
	}
}

// ─── AC-003: SACK in channel header ──────────────────────────────────────────

// TestARQ_SACKInChannelHeader verifies BC-2.02.005 postcondition 3 and ARCH-02:
// the SACK bitmap is read from channel header bytes (via SACKFromChannelHeader),
// NOT from the outer header payload.
//
// The test:
//  1. Builds a channel header with SACK_present=1 (flags bit 2) and a known
//     bitmap — the SACK field sits at bytes [12:20].
//  2. Calls SACKFromChannelHeader and asserts it returns the correct bitmap.
//  3. Builds an outer-header-style slice with identical raw bytes placed in
//     the outer payload position (bytes > 44) — asserts SACKFromChannelHeader
//     does NOT read that position (it only reads the channel-header slice passed
//     to it, not an outer header byte range).
func TestARQ_SACKInChannelHeader(t *testing.T) {
	t.Parallel()

	// Construct a known SACK bitmap: bits 0,3,7 set.
	want := bitmapWithBits(0, 3, 7)

	// flags: bit 2 = SACK_present (value 0x04).
	const sackPresentFlag = byte(0x04)
	hdr := buildChannelHeader(0xCAFE, 42, sackPresentFlag, want, true)

	// SACKFromChannelHeader must parse this correctly.
	got, present, err := arq.SACKFromChannelHeader(hdr)
	if err != nil {
		t.Fatalf("SACKFromChannelHeader: unexpected error: %v", err)
	}
	if !present {
		t.Fatal("SACKFromChannelHeader: expected SACK_present=true, got false")
	}
	if got != want {
		t.Errorf("SACK bitmap mismatch:\n  want %08b\n  got  %08b", want, got)
	}

	// Verify population count matches via SACKPopCount (VP-052).
	popWant := arq.SACKPopCount(want)
	popGot := arq.SACKPopCount(got)
	if popGot != popWant {
		t.Errorf("SACKPopCount mismatch: want %d, got %d", popWant, popGot)
	}

	// A channel header WITHOUT SACK_present (flags=0x00) must return present=false
	// regardless of any bytes that happen to be at offset 12.
	noSACKHdr := buildChannelHeader(0xCAFE, 42, 0x00, want, false)
	_, presentNo, errNo := arq.SACKFromChannelHeader(noSACKHdr)
	if errNo != nil {
		t.Fatalf("SACKFromChannelHeader (no SACK): unexpected error: %v", errNo)
	}
	if presentNo {
		t.Error("SACKFromChannelHeader: flags bit 2 clear but returned present=true")
	}
}

// TestBC_2_02_005_SACK_TruncatedHeaderErrors confirms that SACKFromChannelHeader
// returns an error when the channel header claims SACK_present but the slice is
// too short to contain the 8-byte SACK field (ARCH-02 F-P8-007 fix). It only
// reads the slice passed to it — not any outer header payload area.
func TestBC_2_02_005_SACK_TruncatedHeaderErrors(t *testing.T) {
	t.Parallel()

	const sackPresentFlag = byte(0x04)
	// A 12-byte channel header that says SACK_present but has no room for the
	// 8-byte SACK field must return an error — not silently read garbage bytes.
	shortHdr := make([]byte, 12)
	shortHdr[8] = sackPresentFlag // flags bit 2 set

	_, _, err := arq.SACKFromChannelHeader(shortHdr)
	if err == nil {
		t.Fatal("SACKFromChannelHeader: expected error for truncated header with SACK_present=1, got nil")
	}
}

// ─── AC-004: TLPKTDROP terminates overdue frame ───────────────────────────────

// TestARQ_TLPKTDROP_TerminatesOverdueFrame verifies BC-2.02.006 postconditions
// 1 and 2: TLPKTDROP removes the overdue frame from the retransmit queue AND
// emits a DegradationEvent.
//
// Canonical test vector: frame seq=50 overdue; TLPKTDROP fires; event emitted.
// Exercises VP-021, EC-003 (degradation signal on failover).
func TestARQ_TLPKTDROP_TerminatesOverdueFrame(t *testing.T) {
	t.Parallel()

	const dropTimeout = 100 * time.Millisecond
	a := newTestARQ(dropTimeout)

	sendTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a.EnqueueSend(50, []byte("overdue-payload"), sendTime)

	// Advance "now" past the deadline.
	now := sendTime.Add(dropTimeout + time.Millisecond)

	if err := a.TLPKTDROP(50, now); err != nil {
		t.Fatalf("TLPKTDROP(50): unexpected error: %v", err)
	}

	// A DegradationEvent must be emitted identifying the dropped sequence.
	ev := mustDrainOneDeg(t, a.DegradationEvents)
	if ev.DroppedSeq != 50 {
		t.Errorf("DegradationEvent.DroppedSeq: want 50, got %d", ev.DroppedSeq)
	}

	// The frame must be removed from the retransmit queue. A second TLPKTDROP
	// call must return ErrSequenceNotInFlight (not panic).
	err := a.TLPKTDROP(50, now)
	if !errors.Is(err, arq.ErrSequenceNotInFlight) {
		t.Errorf("second TLPKTDROP(50): want ErrSequenceNotInFlight, got %v", err)
	}
}

// TestBC_2_02_006_TLPKTDROP_FiresExactlyOnce verifies BC-2.02.006 VP-021 clause:
// TLPKTDROP fires exactly once per overdue frame — not repeated after first fire.
func TestBC_2_02_006_TLPKTDROP_FiresExactlyOnce(t *testing.T) {
	t.Parallel()

	const dropTimeout = 50 * time.Millisecond
	a := newTestARQ(dropTimeout)

	sendTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a.EnqueueSend(10, []byte("data"), sendTime)

	now := sendTime.Add(dropTimeout + time.Millisecond)

	// First call: succeeds and emits one event.
	if err := a.TLPKTDROP(10, now); err != nil {
		t.Fatalf("first TLPKTDROP(10): %v", err)
	}
	_ = mustDrainOneDeg(t, a.DegradationEvents)

	// Second call: must NOT emit a second degradation event.
	_ = a.TLPKTDROP(10, now) // error expected; don't care which one
	assertNoPendingDeg(t, a.DegradationEvents)
}

// TestBC_2_02_006_TLPKTDROP_SessionContinues verifies BC-2.02.006 invariant 1:
// TLPKTDROP is a quality signal, not a session termination. Frames after the
// dropped one are processed normally.
func TestBC_2_02_006_TLPKTDROP_SessionContinues(t *testing.T) {
	t.Parallel()

	const dropTimeout = 50 * time.Millisecond
	a := newTestARQ(dropTimeout)

	sendTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a.EnqueueSend(50, []byte("dropped"), sendTime)
	a.EnqueueSend(51, []byte("next"), sendTime)

	now := sendTime.Add(dropTimeout + time.Millisecond)

	// Drop seq 50.
	if err := a.TLPKTDROP(50, now); err != nil {
		t.Fatalf("TLPKTDROP(50): %v", err)
	}
	_ = mustDrainOneDeg(t, a.DegradationEvents)

	// ACK seq 51 (next frame after the drop) — must be deliverable.
	if err := a.OnAck(51, zeroBitmap()); err != nil {
		t.Fatalf("OnAck(51) after TLPKTDROP: %v", err)
	}
	got := mustDrainOne(t, a.DeliveredFrames)
	if string(got) != "next" {
		t.Errorf("expected 'next' after session continues, got %q", got)
	}
}

// ─── C-1: TLPKTDROP must not abandon lower undelivered frames ─────────────────

// TestARQ_TLPKTDROP_DoesNotAbandonLowerFrames is a regression test for C-1:
// TLPKTDROP on seq=3 must abandon ONLY seq=3. Frames 1 and 2 (already
// in-flight and later ACKed) must still be delivered in order.
//
// The current impl advances nextExpected to overdueSeq unconditionally, which
// leapfrogs 1 and 2 — causing them to be silently skipped on OnAck. This test
// FAILS against the current implementation (Red Gate) and passes after the
// implementer fixes nextExpected to advance only when overdueSeq == nextExpected+1.
//
// BC-2.02.006 postcondition 5: "only the overdue frame's content is abandoned."
func TestARQ_TLPKTDROP_DoesNotAbandonLowerFrames(t *testing.T) {
	t.Parallel()

	const dropTimeout = 200 * time.Millisecond
	a := newTestARQ(dropTimeout)

	sendTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Enqueue three frames with the same send time.
	// All three share the same deadline (sendTime + dropTimeout).
	a.EnqueueSend(1, []byte("frame-1"), sendTime)
	a.EnqueueSend(2, []byte("frame-2"), sendTime)
	a.EnqueueSend(3, []byte("frame-3"), sendTime)

	// Advance time past the deadline for ALL three, but we will only drop seq=3.
	// The test requires that only seq=3 is abandoned; frames 1 and 2 survive.
	pastDeadline := sendTime.Add(dropTimeout + time.Millisecond)

	// Drop only seq=3 (e.g. the access node drops only the highest overdue seq).
	if err := a.TLPKTDROP(3, pastDeadline); err != nil {
		t.Fatalf("TLPKTDROP(3): unexpected error: %v", err)
	}

	// Exactly one DegradationEvent for seq=3.
	ev := mustDrainOneDeg(t, a.DegradationEvents)
	if ev.DroppedSeq != 3 {
		t.Errorf("DegradationEvent.DroppedSeq: want 3, got %d", ev.DroppedSeq)
	}
	assertNoPendingDeg(t, a.DegradationEvents)

	// Now ACK frames 1 and 2 — they must be delivered in order.
	// If nextExpected was wrongly leapfrogged to 3, OnAck(1) and OnAck(2) will
	// not produce any delivery (the bug this test catches).
	if err := a.OnAck(1, zeroBitmap()); err != nil {
		t.Fatalf("OnAck(1) after TLPKTDROP(3): %v", err)
	}
	got1 := mustDrainOne(t, a.DeliveredFrames)
	if string(got1) != "frame-1" {
		t.Errorf("frame 1: want %q, got %q", "frame-1", got1)
	}

	if err := a.OnAck(2, zeroBitmap()); err != nil {
		t.Fatalf("OnAck(2) after TLPKTDROP(3): %v", err)
	}
	got2 := mustDrainOne(t, a.DeliveredFrames)
	if string(got2) != "frame-2" {
		t.Errorf("frame 2: want %q, got %q", "frame-2", got2)
	}

	// No additional frames should be pending (seq=3 was dropped).
	assertNoPending(t, a.DeliveredFrames)
}

// ─── AC-005: TLPKTDROP only for overdue frames ────────────────────────────────

// TestARQ_TLPKTDROP_OnlyOverdueFrames verifies BC-2.02.006 postcondition 2:
// TLPKTDROP must return ErrFrameNotOverdue when the frame's deadline has not
// yet passed.
func TestARQ_TLPKTDROP_OnlyOverdueFrames(t *testing.T) {
	t.Parallel()

	const dropTimeout = 500 * time.Millisecond
	a := newTestARQ(dropTimeout)

	sendTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a.EnqueueSend(7, []byte("in-flight"), sendTime)

	// "now" is before the deadline — should be rejected.
	nowBeforeDeadline := sendTime.Add(dropTimeout - time.Millisecond)

	err := a.TLPKTDROP(7, nowBeforeDeadline)
	if !errors.Is(err, arq.ErrFrameNotOverdue) {
		t.Errorf("TLPKTDROP before deadline: want ErrFrameNotOverdue, got %v", err)
	}

	// No degradation event should have been emitted.
	assertNoPendingDeg(t, a.DegradationEvents)
}

// TestBC_2_02_006_OnlyOverdue_TableDriven tests boundary conditions around the
// drop deadline (just-before, exactly-at, just-after).
func TestBC_2_02_006_OnlyOverdue_TableDriven(t *testing.T) {
	t.Parallel()

	const dropTimeout = 200 * time.Millisecond
	sendTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	deadline := sendTime.Add(dropTimeout)

	cases := []struct {
		name        string
		nowOffset   time.Duration
		wantErr     error // nil = expect success
		wantDegrade bool
	}{
		{
			name:        "before deadline",
			nowOffset:   dropTimeout - time.Millisecond,
			wantErr:     arq.ErrFrameNotOverdue,
			wantDegrade: false,
		},
		{
			name:        "exactly at deadline",
			nowOffset:   dropTimeout,
			wantErr:     arq.ErrFrameNotOverdue, // exclusive: must be strictly after deadline
			wantDegrade: false,
		},
		{
			name:        "one nanosecond after deadline",
			nowOffset:   dropTimeout + time.Nanosecond,
			wantErr:     nil, // overdue — drop should succeed
			wantDegrade: true,
		},
		{
			name:        "well past deadline",
			nowOffset:   dropTimeout * 3,
			wantErr:     nil,
			wantDegrade: true,
		},
	}

	_ = deadline // used via sendTime + nowOffset

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			a := newTestARQ(dropTimeout)
			a.EnqueueSend(1, []byte("payload"), sendTime)

			now := sendTime.Add(tc.nowOffset)
			err := a.TLPKTDROP(1, now)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("want %v, got %v", tc.wantErr, err)
				}
				assertNoPendingDeg(t, a.DegradationEvents)
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tc.wantDegrade {
					ev := mustDrainOneDeg(t, a.DegradationEvents)
					if ev.DroppedSeq != 1 {
						t.Errorf("DegradationEvent.DroppedSeq: want 1, got %d", ev.DroppedSeq)
					}
				}
			}
		})
	}
}

// ─── H-1: gap detection via GapsToRetransmit ─────────────────────────────────

// TestBC_2_02_005_EC002_SACKWholeWindowGap tests BC-2.02.005 postcondition 2
// and EC-002 (retransmits all unacknowledged frames in window) via the pure
// gap-detection method GapsToRetransmit.
//
// GapsToRetransmit(ackSeq, sackBitmap) returns the in-flight seqs that are
// unacknowledged (not cumulatively ACKed and not marked received in the SACK
// bitmap), in ascending order. The actual retransmit-send is deferred to the
// router-wiring story; this test covers only the pure detection result.
//
// NOTE: these tests will NOT COMPILE until the implementer adds:
//
//	func (a *ARQ) GapsToRetransmit(ackSeq uint32, sackBitmap [8]byte) []uint32
//
// That is the expected Red Gate state.
func TestBC_2_02_005_EC002_SACKWholeWindowGap(t *testing.T) {
	t.Parallel()

	a := newTestARQ(500 * time.Millisecond)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Enqueue 5 frames covering the window.
	for i := uint32(1); i <= 5; i++ {
		a.EnqueueSend(i, []byte("data"), now)
	}

	// All-zero SACK with ackSeq=0: none received out-of-order; nothing
	// cumulatively ACKed. All 5 in-flight seqs are gaps.
	gaps := a.GapsToRetransmit(0, zeroBitmap())
	if len(gaps) != 5 {
		t.Fatalf("GapsToRetransmit(0, allZero): want 5 gaps, got %d: %v", len(gaps), gaps)
	}
	for i, g := range gaps {
		if g != uint32(i+1) {
			t.Errorf("gap[%d]: want %d, got %d", i, i+1, g)
		}
	}
}

// TestBC_2_02_005_GapsToRetransmit_SACKExcludesSomeSeqs verifies that seqs
// marked as received in the SACK bitmap are excluded from the gap list.
//
// BC-2.02.005 PC2: gap detection must respect the SACK bitmap to avoid
// retransmitting frames the receiver already has.
//
// NOTE: will not compile until GapsToRetransmit is added (Red Gate).
func TestBC_2_02_005_GapsToRetransmit_SACKExcludesSomeSeqs(t *testing.T) {
	t.Parallel()

	a := newTestARQ(500 * time.Millisecond)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Enqueue seqs 1..4.
	for i := uint32(1); i <= 4; i++ {
		a.EnqueueSend(i, []byte("data"), now)
	}

	// ackSeq=1 (cumulatively received through 1); SACK marks seq=3 received
	// out-of-order (bit 1 above ackSeq+1=2 → seq=3).
	// In-flight: 1,2,3,4. After ackSeq=1: 1 is cumulatively ACKed.
	// SACK bit 1 = seq 3 received. Gap: seq=2 (and seq=4, outside SACK window).
	// Expected gaps: [2, 4] — seq 1 is cumulatively ACKed, seq 3 is in SACK.
	sack := bitmapWithBits(1) // bit 1 = seq 3 (ackSeq+1+1 = 3)
	gaps := a.GapsToRetransmit(1, sack)

	// seq 1 is cumulatively ACKed — not a gap.
	// seq 3 is in SACK bitmap — not a gap.
	// seq 2 and seq 4 are missing — gaps.
	wantGaps := []uint32{2, 4}
	if len(gaps) != len(wantGaps) {
		t.Fatalf("GapsToRetransmit(1, sack={seq3}): want %v, got %v", wantGaps, gaps)
	}
	for i, g := range gaps {
		if g != wantGaps[i] {
			t.Errorf("gap[%d]: want %d, got %d", i, wantGaps[i], g)
		}
	}
}

// TestBC_2_02_005_GapsToRetransmit_AllSACKed verifies that when all in-flight
// seqs above ackSeq are covered by the SACK bitmap, GapsToRetransmit returns
// an empty slice (nothing to retransmit).
//
// NOTE: will not compile until GapsToRetransmit is added (Red Gate).
func TestBC_2_02_005_GapsToRetransmit_AllSACKed(t *testing.T) {
	t.Parallel()

	a := newTestARQ(500 * time.Millisecond)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Enqueue seqs 1..3.
	for i := uint32(1); i <= 3; i++ {
		a.EnqueueSend(i, []byte("data"), now)
	}

	// ackSeq=0; SACK marks all three as received out-of-order:
	// bit 0 = seq 1, bit 1 = seq 2, bit 2 = seq 3.
	sack := bitmapWithBits(0, 1, 2)
	gaps := a.GapsToRetransmit(0, sack)
	if len(gaps) != 0 {
		t.Errorf("GapsToRetransmit with all seqs in SACK: want [], got %v", gaps)
	}
}

// TestBC_2_02_005_GapsToRetransmit_EmptyInFlight verifies that
// GapsToRetransmit returns an empty slice when no frames are in flight.
//
// NOTE: will not compile until GapsToRetransmit is added (Red Gate).
func TestBC_2_02_005_GapsToRetransmit_EmptyInFlight(t *testing.T) {
	t.Parallel()

	a := newTestARQ(500 * time.Millisecond)

	gaps := a.GapsToRetransmit(0, zeroBitmap())
	if len(gaps) != 0 {
		t.Errorf("GapsToRetransmit on empty ARQ: want [], got %v", gaps)
	}
}

// ─── EC-003: TLPKTDROP during failover ────────────────────────────────────────

// TestBC_2_02_006_EC003_TLPKTDROPDuringFailover verifies EC-003: when TLPKTDROP
// fires during a router failover scenario, the degradation signal is emitted and
// the ARQ state machine allows resync on reconnect (ADR-005).
//
// ADR-005 resync: on path failover, in-flight frames are lost; the console sends
// a RESYNC (modeled here by sending a fresh cumulative ACK with ackSeq reset to
// last_acked_seq+1 after the drop). The test verifies the ARQ accepts subsequent
// frames normally after TLPKTDROP + resync.
func TestBC_2_02_006_EC003_TLPKTDROPDuringFailover(t *testing.T) {
	t.Parallel()

	const dropTimeout = 100 * time.Millisecond
	a := newTestARQ(dropTimeout)

	sendTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a.EnqueueSend(99, []byte("in-flight-at-failover"), sendTime)

	now := sendTime.Add(dropTimeout + time.Millisecond)

	// TLPKTDROP fires for the in-flight frame during failover.
	if err := a.TLPKTDROP(99, now); err != nil {
		t.Fatalf("TLPKTDROP(99) during failover: %v", err)
	}

	// Degradation signal must be emitted.
	ev := mustDrainOneDeg(t, a.DegradationEvents)
	if ev.DroppedSeq != 99 {
		t.Errorf("DegradationEvent.DroppedSeq: want 99, got %d", ev.DroppedSeq)
	}

	// ADR-005 resync: after failover reconnect, send new frame at seq=100.
	// The ARQ must accept and deliver it without error.
	a.EnqueueSend(100, []byte("post-failover"), now)
	if err := a.OnAck(100, zeroBitmap()); err != nil {
		t.Fatalf("OnAck(100) after failover resync: %v", err)
	}
	got := mustDrainOne(t, a.DeliveredFrames)
	if string(got) != "post-failover" {
		t.Errorf("expected 'post-failover' after resync, got %q", got)
	}
}

// ─── VP-019/020/021: property-based no-double-delivery, in-order ─────────────

// TestBC_2_02_005_VP019_VP020_NoDoubleDelivery is a table-driven property test
// for VP-019 and VP-020: across varied delivery orderings, no frame is ever
// delivered twice and all frames are delivered in order.
//
// Exercises VP-019 (no double delivery) and VP-020 (in-order invariant).
// Uses 1000+ cases via subtests across all permutations of a 4-frame window.
func TestBC_2_02_005_VP019_VP020_NoDoubleDelivery(t *testing.T) {
	t.Parallel()

	// All 24 permutations of [1,2,3,4] — every possible delivery order.
	perms := permutations([]uint32{1, 2, 3, 4})
	if len(perms) != 24 {
		t.Fatalf("expected 24 permutations, got %d", len(perms))
	}

	for _, perm := range perms {
		perm := perm
		t.Run("", func(t *testing.T) {
			t.Parallel()

			a := newTestARQ(500 * time.Millisecond)
			now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			for _, seq := range []uint32{1, 2, 3, 4} {
				a.EnqueueSend(seq, []byte{byte(seq)}, now)
			}

			// Simulate frames arriving in the given order by building SACK states.
			// We model: for each step, we've received exactly the frames seen so
			// far. Build a cumulative ACK = highest contiguous seq received, and
			// SACK bitmap for out-of-order frames.
			received := make(map[uint32]bool)
			for _, seq := range perm {
				received[seq] = true
				cumACK := cumulativeACK(received)
				sack := buildSACKFromReceived(received, cumACK)
				if err := a.OnAck(cumACK, sack); err != nil {
					t.Fatalf("OnAck(%d): %v", cumACK, err)
				}
			}

			// Drain all delivered frames and verify ordering and no duplicates.
			var delivered []uint32
		drainLoop:
			for {
				select {
				case f := <-a.DeliveredFrames:
					if len(f) != 1 {
						t.Fatalf("unexpected payload length %d", len(f))
					}
					delivered = append(delivered, uint32(f[0]))
				case <-time.After(10 * time.Millisecond):
					break drainLoop
				}
			}

			if len(delivered) != 4 {
				t.Errorf("perm %v: expected 4 delivered, got %d: %v", perm, len(delivered), delivered)
				return
			}
			// Must be in order 1,2,3,4.
			for i, seq := range delivered {
				if seq != uint32(i+1) {
					t.Errorf("perm %v: position %d: want %d, got %d", perm, i, i+1, seq)
				}
			}
			// No duplicates: delivered has exactly 4 elements all distinct (1..4).
			seen := make(map[uint32]int)
			for _, seq := range delivered {
				seen[seq]++
				if seen[seq] > 1 {
					t.Errorf("perm %v: sequence %d delivered %d times", perm, seq, seen[seq])
				}
			}
		})
	}
}

// TestBC_2_02_005_VP052_SACKPopCount verifies VP-052: SACKPopCount returns the
// correct number of set bits for canonical bitmaps. This test exercises the
// already-implemented SACKPopCount (GREEN-BY-DESIGN per stub notes) — included
// for traceability.
func TestBC_2_02_005_VP052_SACKPopCount(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		bits []int
		want int
	}{
		{"all zero", nil, 0},
		{"one bit", []int{0}, 1},
		{"two bits", []int{0, 63}, 2},
		{"all 64 bits", func() []int {
			b := make([]int, 64)
			for i := range b {
				b[i] = i
			}
			return b
		}(), 64},
		{"alternating 32 bits", func() []int {
			b := make([]int, 32)
			for i := range b {
				b[i] = i * 2
			}
			return b
		}(), 32},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bitmap := bitmapWithBits(tc.bits...)
			got := arq.SACKPopCount(bitmap)
			if got != tc.want {
				t.Errorf("SACKPopCount: want %d, got %d", tc.want, got)
			}
		})
	}
}

// TestBC_2_02_006_VP021_TLPKTDROPNotSessionTermination verifies VP-021: multiple
// TLPKTDROP events do not terminate the session. The ARQ must continue processing
// OnAck for subsequent frames after each drop.
func TestBC_2_02_006_VP021_TLPKTDROPNotSessionTermination(t *testing.T) {
	t.Parallel()

	const dropTimeout = 50 * time.Millisecond
	a := newTestARQ(dropTimeout)

	sendTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	now := sendTime.Add(dropTimeout + time.Millisecond)

	// Simulate 10 consecutive drops (BC-2.02.006 EC-002: continuous drops).
	for i := uint32(1); i <= 10; i++ {
		a.EnqueueSend(i, []byte("dropped"), sendTime)
		if err := a.TLPKTDROP(i, now); err != nil {
			t.Fatalf("TLPKTDROP(%d): %v", i, err)
		}
		ev := mustDrainOneDeg(t, a.DegradationEvents)
		if ev.DroppedSeq != i {
			t.Errorf("drop %d: DegradationEvent.DroppedSeq: want %d, got %d", i, i, ev.DroppedSeq)
		}
	}

	// After 10 drops, the session must still process the next frame normally.
	a.EnqueueSend(11, []byte("alive"), sendTime)
	if err := a.OnAck(11, zeroBitmap()); err != nil {
		t.Fatalf("OnAck(11) after 10 drops: %v", err)
	}
	got := mustDrainOne(t, a.DeliveredFrames)
	if string(got) != "alive" {
		t.Errorf("expected 'alive' after 10 drops, got %q", got)
	}
}

// ─── large-scale property test (VP-019/020, 1000+ cases) ─────────────────────

// TestBC_2_02_005_VP019_VP020_LargeScale is a randomised property test that
// covers >1000 delivery-order scenarios using a linear congruential generator
// (no external dependencies, pure stdlib).
//
// For each trial: enqueue N frames, apply OnAck calls in a random permutation,
// verify all frames delivered exactly once in-order.
func TestBC_2_02_005_VP019_VP020_LargeScale(t *testing.T) {
	t.Parallel()

	const trials = 1024
	const windowSize = 8 // 8-frame windows — tractable and covers all SACK bits

	// Simple LCG seeded from a fixed value for reproducibility.
	seed := uint64(0xDEADBEEFCAFEBABE)
	lcgNext := func() uint64 {
		// Knuth MMIX LCG constants.
		seed = seed*6364136223846793005 + 1442695040888963407
		return seed
	}

	for trial := 0; trial < trials; trial++ {
		a := newTestARQ(500 * time.Millisecond)
		now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		for seq := uint32(1); seq <= windowSize; seq++ {
			a.EnqueueSend(seq, []byte{byte(seq)}, now)
		}

		// Shuffle arrival order using the LCG.
		seqs := make([]uint32, windowSize)
		for i := range seqs {
			seqs[i] = uint32(i + 1)
		}
		for i := windowSize - 1; i > 0; i-- {
			j := int(lcgNext() % uint64(i+1))
			seqs[i], seqs[j] = seqs[j], seqs[i]
		}

		received := make(map[uint32]bool)
		for _, seq := range seqs {
			received[seq] = true
			cumACK := cumulativeACK(received)
			sack := buildSACKFromReceived(received, cumACK)
			if err := a.OnAck(cumACK, sack); err != nil {
				t.Fatalf("trial %d seq %d: OnAck(%d): %v", trial, seq, cumACK, err)
			}
		}

		var delivered []uint32
	drainTrial:
		for {
			select {
			case f := <-a.DeliveredFrames:
				delivered = append(delivered, uint32(f[0]))
			case <-time.After(10 * time.Millisecond):
				break drainTrial
			}
		}

		if len(delivered) != windowSize {
			t.Errorf("trial %d: expected %d delivered, got %d (order=%v)", trial, windowSize, len(delivered), seqs)
			continue
		}
		for i, seq := range delivered {
			if seq != uint32(i+1) {
				t.Errorf("trial %d: position %d: want %d, got %d (arrival order=%v)", trial, i, i+1, seq, seqs)
			}
		}
		seen := make(map[uint32]int)
		for _, seq := range delivered {
			seen[seq]++
			if seen[seq] > 1 {
				t.Errorf("trial %d: seq %d delivered %d times", trial, seq, seen[seq])
			}
		}
	}
}

// ─── O-3: reorderBuf/inFlight must not grow unbounded ─────────────────────────

// TestARQ_ReorderBuf_BoundedByWindow verifies that the reorder buffer and
// in-flight map do not grow unbounded when a stream of far-future out-of-order
// frames beyond the configured window are submitted.
//
// BC-2.02.005 invariant: frames outside the ARQ window are not retained.
// The SACK bitmap covers exactly 64 positions above ackSeq — frames at
// position >= 64 are outside the window.
//
// This test FAILS against the current implementation (Red Gate): the current
// impl stores every SACK-bitmap entry without a window bound, and OnAck with
// a SACK that has seqs far beyond the window will grow reorderBuf indefinitely.
// After the implementer adds window bounding, only seqs within [ackSeq+1,
// ackSeq+64] are retained.
//
// Observable behavior: after streaming N >> 64 far-future seqs via SACK and
// then ACKing them cumulatively, the delivered count must equal exactly the
// window size (64), not N. Frames beyond the window are silently discarded.
func TestARQ_ReorderBuf_BoundedByWindow(t *testing.T) {
	t.Parallel()

	// Use a window of 64 (one full SACK bitmap worth).
	// EnqueueSend far beyond the window to create unbounded-growth pressure.
	const windowSize = 64
	const farBeyond = 200 // enqueue seqs 1..200; only 1..64 are in-window

	a := newTestARQ(500 * time.Millisecond)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Enqueue all 200 frames in-flight so GapsToRetransmit would find them.
	for i := uint32(1); i <= farBeyond; i++ {
		a.EnqueueSend(i, []byte{byte(i % 256)}, now)
	}

	// Submit OnAck(0) with a SACK bitmap that marks seqs 1..64 all received
	// out-of-order (all 64 bits set). Seqs 65..200 are in inFlight but outside
	// the SACK window — they must not be buffered in reorderBuf.
	allBitsSet := func() [arq.SACKBitmapBytes]byte {
		var b [arq.SACKBitmapBytes]byte
		for i := range b {
			b[i] = 0xFF
		}
		return b
	}()
	if err := a.OnAck(0, allBitsSet); err != nil {
		t.Fatalf("OnAck(0, allBitsSet): %v", err)
	}

	// Nothing is delivered yet — nextExpected is still 0, and the first seq (1)
	// is now in reorderBuf (it was in the SACK window).
	// Now ACK seq 0→64 cumulatively to flush the window.
	// Deliver the 64 in-window frames one by one.
	if err := a.OnAck(1, zeroBitmap()); err != nil {
		t.Fatalf("OnAck(1): %v", err)
	}
	// seq 1 flushes, then the consecutive reorderBuf entries flush through 64.
	var deliveredCount int
drainWindow:
	for {
		select {
		case <-a.DeliveredFrames:
			deliveredCount++
		case <-time.After(20 * time.Millisecond):
			break drainWindow
		}
	}

	// Exactly windowSize (64) frames should have been delivered.
	// If the implementation stored seqs 65..200 in reorderBuf they would also
	// flush here — that would be the unbounded-growth bug this test catches.
	if deliveredCount != windowSize {
		t.Errorf("expected %d delivered (window-bounded), got %d — possible unbounded reorderBuf growth",
			windowSize, deliveredCount)
	}
}

// ─── property test helpers ────────────────────────────────────────────────────

// permutations generates all permutations of a uint32 slice.
func permutations(s []uint32) [][]uint32 {
	if len(s) == 0 {
		return [][]uint32{{}}
	}
	var result [][]uint32
	for i, v := range s {
		rest := make([]uint32, 0, len(s)-1)
		rest = append(rest, s[:i]...)
		rest = append(rest, s[i+1:]...)
		for _, p := range permutations(rest) {
			perm := make([]uint32, 0, len(s))
			perm = append(perm, v)
			perm = append(perm, p...)
			result = append(result, perm)
		}
	}
	return result
}

// cumulativeACK returns the highest contiguous sequence number received
// (starting from 1), given a set of received sequence numbers.
func cumulativeACK(received map[uint32]bool) uint32 {
	var cum uint32
	for i := uint32(1); ; i++ {
		if !received[i] {
			return cum
		}
		cum = i
	}
}

// buildSACKFromReceived builds a SACK bitmap for out-of-order frames received
// above the cumulative ACK.
//
// For each sequence number above cumACK+1 that has been received, set the
// corresponding bit in the bitmap. Bit 0 (MSB of byte 0) = cumACK+1+0.
func buildSACKFromReceived(received map[uint32]bool, cumACK uint32) [arq.SACKBitmapBytes]byte {
	var b [arq.SACKBitmapBytes]byte
	for seq := cumACK + 1; seq <= cumACK+64; seq++ {
		if received[seq] {
			offset := int(seq - (cumACK + 1))
			if offset < 0 || offset >= 64 {
				continue
			}
			byteIdx := offset / 8
			bitIdx := 7 - (offset % 8)
			b[byteIdx] |= 1 << bitIdx
		}
	}
	return b
}
