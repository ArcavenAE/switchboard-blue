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
// OnAck returns delivered frames synchronously as [][]byte (pass-2 adjudication,
// Option a). No DeliveredFrames channel; no goroutines in the ARQ struct.
// TLPKTDROP returns a DegradationEvent value AND sends non-blocking to
// DegradationEvents chan for the metrics layer.
//
// ARQ is single-writer per half-channel; no concurrent-OnAck tests are included.
package arq_test

import (
	"errors"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/arq"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

// assertNoPendingDeg asserts that no DegradationEvent is queued on ch.
// Used to verify TLPKTDROP did NOT fire.
func assertNoPendingDeg(t *testing.T, ch <-chan arq.DegradationEvent) {
	t.Helper()
	select {
	case ev := <-ch:
		t.Fatalf("unexpected DegradationEvent: %+v", ev)
	default:
		// good — nothing queued
	}
}

// newTestARQ builds an ARQ with a buffered degradation channel so tests can
// inspect it without a separate goroutine. No DeliveredBufSize — delivery is
// now synchronous via OnAck return value (pass-2 adjudication).
func newTestARQ(dropTimeout time.Duration) *arq.ARQ {
	return arq.New(arq.Config{
		DropTimeout:        dropTimeout,
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
// Idempotent ACK returns (nil, nil) exactly (M-1 ruling: ErrDuplicateSequence removed).
func TestARQ_OnAck_NoDuplicateDelivery(t *testing.T) {
	t.Parallel()

	a := newTestARQ(100 * time.Millisecond)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a.EnqueueSend(1, []byte("frame-1"), now)

	// First OnAck — should deliver the frame synchronously.
	frames, err := a.OnAck(1, zeroBitmap())
	if err != nil {
		t.Fatalf("first OnAck(1) returned unexpected error: %v", err)
	}
	if len(frames) != 1 {
		t.Fatalf("first OnAck(1): want 1 frame delivered, got %d", len(frames))
	}
	if string(frames[0]) != "frame-1" {
		t.Errorf("first OnAck(1): want %q, got %q", "frame-1", frames[0])
	}

	// Second OnAck for same sequence — must return (nil, nil) exactly (idempotent
	// per EC-001; no double delivery).
	frames2, err2 := a.OnAck(1, zeroBitmap())
	if err2 != nil {
		t.Fatalf("second OnAck(1): want nil error, got %v", err2)
	}
	if len(frames2) != 0 {
		t.Errorf("second OnAck(1): want 0 frames (no double delivery), got %d", len(frames2))
	}
}

// TestBC_2_02_005_EC001_IdempotentAck verifies EC-001: ACKing an already-acked
// sequence is idempotent — returns nil error and zero frames, never double-delivers.
//
// M-1 ruling: idempotent ACK returns (nil, nil) exactly. The ErrDuplicateSequence
// sentinel is being removed; accepting it here would be tautological.
func TestBC_2_02_005_EC001_IdempotentAck(t *testing.T) {
	t.Parallel()

	a := newTestARQ(100 * time.Millisecond)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a.EnqueueSend(5, []byte("payload"), now)

	frames, err := a.OnAck(5, zeroBitmap())
	if err != nil {
		t.Fatalf("initial OnAck(5): %v", err)
	}
	if len(frames) != 1 {
		t.Fatalf("initial OnAck(5): want 1 frame, got %d", len(frames))
	}

	// Re-ACK same seq — must return nil error and zero frames; must not double-deliver.
	frames2, err2 := a.OnAck(5, zeroBitmap())
	if err2 != nil {
		t.Fatalf("idempotent OnAck(5): want nil error, got %v", err2)
	}
	if len(frames2) != 0 {
		t.Errorf("idempotent OnAck(5): want 0 frames, got %d", len(frames2))
	}
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

	// Step 1: OnAck(0, sack=seq3 received): gap at seq 1 and 2; seq 3 buffered.
	// Bit positions are zero-based offsets above ackSeq+1.
	// With ackSeq=0: bit 0 → seq 1, bit 1 → seq 2, bit 2 → seq 3.
	sackSeq3 := bitmapWithBits(2) // bit 2 = seq 3 (offset 2 above ackSeq+1=1)

	frames0, err := a.OnAck(0, sackSeq3)
	if err != nil {
		t.Fatalf("OnAck(0, sack={seq3}): %v", err)
	}
	// Nothing should be delivered yet — gap at seq 1 blocks all.
	if len(frames0) != 0 {
		t.Errorf("OnAck(0, sack={seq3}): want 0 frames (gap at 1 blocks), got %d", len(frames0))
	}

	// Step 2: OnAck(1): fills the cumulative pointer through 1; delivers seq 1.
	frames1, err := a.OnAck(1, zeroBitmap())
	if err != nil {
		t.Fatalf("OnAck(1): %v", err)
	}
	if len(frames1) != 1 {
		t.Fatalf("OnAck(1): want 1 frame, got %d", len(frames1))
	}
	if string(frames1[0]) != "one" {
		t.Errorf("OnAck(1): want %q, got %q", "one", frames1[0])
	}

	// Step 3: OnAck(2): fills the gap at 2; delivers seq 2 then seq 3 (buffered).
	frames2, err := a.OnAck(2, zeroBitmap())
	if err != nil {
		t.Fatalf("OnAck(2): %v", err)
	}
	if len(frames2) != 2 {
		t.Fatalf("OnAck(2): want 2 frames (seq 2 + buffered seq 3), got %d", len(frames2))
	}
	if string(frames2[0]) != "two" {
		t.Errorf("OnAck(2) frame[0]: want %q, got %q", "two", frames2[0])
	}
	if string(frames2[1]) != "three" {
		t.Errorf("OnAck(2) frame[1]: want %q, got %q", "three", frames2[1])
	}
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
	frames1, err := a.OnAck(1, sack)
	if err != nil {
		t.Fatalf("OnAck(1, sack={seq3}): %v", err)
	}
	// seq 1 must be delivered; seq 3 buffered; seq 2 not yet.
	if len(frames1) != 1 {
		t.Fatalf("OnAck(1, sack={seq3}): want 1 frame (seq1), got %d", len(frames1))
	}
	if string(frames1[0]) != "seq1" {
		t.Errorf("expected seq1, got %q", frames1[0])
	}

	// Simulate retransmit of seq=2 arriving; OnAck(2).
	frames2, err := a.OnAck(2, zeroBitmap())
	if err != nil {
		t.Fatalf("OnAck(2): %v", err)
	}
	if len(frames2) != 2 {
		t.Fatalf("OnAck(2): want 2 frames (seq2+seq3), got %d", len(frames2))
	}
	if string(frames2[0]) != "seq2" {
		t.Errorf("frame[0]: expected seq2, got %q", frames2[0])
	}
	if string(frames2[1]) != "seq3" {
		t.Errorf("frame[1]: expected seq3, got %q", frames2[1])
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
//  3. Builds a byte slice sized like an outer header (>44 bytes) where the
//     channel-header SACK region (bytes 8 flags, 12-19 bitmap) is zeroed but
//     the SACK-present flag and bitmap data are placed at an outer-header offset
//     (byte 45+). Asserts SACKFromChannelHeader returns present=false, proving
//     it only reads channel-header offsets and cannot pick up outer-header bytes
//     (F-P8-007 anti-regression, ARCH-02).
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

	// F-P8-007 anti-regression (ARCH-02): SACKFromChannelHeader must read ONLY
	// channel-header offsets, never outer-header payload bytes.
	//
	// Build a 60-byte slice that mimics a frame whose outer header occupies bytes
	// 0..43 and whose payload starts at byte 44.  Within this slice:
	//   - Bytes 8 (flags) and 12-19 (SACK bitmap) — the channel-header region —
	//     are all zero (SACK_present bit clear, bitmap zeroed).
	//   - Byte 45 is set to 0x04 (SACK_present flag value) and bytes 48-55 carry
	//     the known SACK bitmap — these are in the outer-payload region.
	//
	// Passing this slice as the channelHeader argument must return present=false
	// because the function reads flags from byte 8 (clear), not byte 45.
	outerStyleSlice := make([]byte, 60)
	const outerPayloadFlagsOff = 45
	const outerPayloadBitmapOff = 48
	outerStyleSlice[outerPayloadFlagsOff] = sackPresentFlag
	copy(outerStyleSlice[outerPayloadBitmapOff:outerPayloadBitmapOff+arq.SACKBitmapBytes], want[:])
	_, presentOuter, errOuter := arq.SACKFromChannelHeader(outerStyleSlice)
	if errOuter != nil {
		t.Fatalf("SACKFromChannelHeader (outer-style slice): unexpected error: %v", errOuter)
	}
	if presentOuter {
		t.Error("SACKFromChannelHeader: read SACK_present from outer-header offset instead of channel-header flags byte")
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
// emits a DegradationEvent both as return value and via the channel.
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

	ev, err := a.TLPKTDROP(50, now)
	if err != nil {
		t.Fatalf("TLPKTDROP(50): unexpected error: %v", err)
	}

	// The returned DegradationEvent must identify the dropped sequence.
	if ev.DroppedSeq != 50 {
		t.Errorf("returned DegradationEvent.DroppedSeq: want 50, got %d", ev.DroppedSeq)
	}

	// A DegradationEvent must also be sent on the channel (for the metrics layer).
	select {
	case chEv := <-a.DegradationEvents:
		if chEv.DroppedSeq != 50 {
			t.Errorf("channel DegradationEvent.DroppedSeq: want 50, got %d", chEv.DroppedSeq)
		}
	default:
		t.Fatal("DegradationEvents channel: expected event, got nothing")
	}

	// The frame must be removed from the retransmit queue. A second TLPKTDROP
	// call must return ErrSequenceNotInFlight (not panic).
	_, err2 := a.TLPKTDROP(50, now)
	if !errors.Is(err2, arq.ErrSequenceNotInFlight) {
		t.Errorf("second TLPKTDROP(50): want ErrSequenceNotInFlight, got %v", err2)
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

	// First call: succeeds and returns one event; channel has one event.
	ev, err := a.TLPKTDROP(10, now)
	if err != nil {
		t.Fatalf("first TLPKTDROP(10): %v", err)
	}
	if ev.DroppedSeq != 10 {
		t.Errorf("first TLPKTDROP(10) return: want DroppedSeq=10, got %d", ev.DroppedSeq)
	}
	// Drain the channel event.
	select {
	case <-a.DegradationEvents:
	default:
		t.Fatal("first TLPKTDROP(10): expected channel event, got nothing")
	}

	// Second call: must NOT emit a second degradation event on the channel.
	_, _ = a.TLPKTDROP(10, now) // error expected; don't care which one
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
	ev, err := a.TLPKTDROP(50, now)
	if err != nil {
		t.Fatalf("TLPKTDROP(50): %v", err)
	}
	if ev.DroppedSeq != 50 {
		t.Errorf("TLPKTDROP(50): want DroppedSeq=50, got %d", ev.DroppedSeq)
	}
	// Drain the channel event.
	select {
	case <-a.DegradationEvents:
	default:
		t.Fatal("TLPKTDROP(50): expected channel event, got nothing")
	}

	// ACK seq 51 (next frame after the drop) — must be deliverable.
	frames, err := a.OnAck(51, zeroBitmap())
	if err != nil {
		t.Fatalf("OnAck(51) after TLPKTDROP: %v", err)
	}
	if len(frames) != 1 {
		t.Fatalf("OnAck(51): want 1 frame, got %d", len(frames))
	}
	if string(frames[0]) != "next" {
		t.Errorf("expected 'next' after session continues, got %q", frames[0])
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
	a.EnqueueSend(1, []byte("frame-1"), sendTime)
	a.EnqueueSend(2, []byte("frame-2"), sendTime)
	a.EnqueueSend(3, []byte("frame-3"), sendTime)

	// Advance time past the deadline for ALL three, but we only drop seq=3.
	pastDeadline := sendTime.Add(dropTimeout + time.Millisecond)

	ev, err := a.TLPKTDROP(3, pastDeadline)
	if err != nil {
		t.Fatalf("TLPKTDROP(3): unexpected error: %v", err)
	}

	// Exactly one DegradationEvent for seq=3 via return value.
	if ev.DroppedSeq != 3 {
		t.Errorf("TLPKTDROP(3) return: want DroppedSeq=3, got %d", ev.DroppedSeq)
	}
	// Drain the channel event.
	select {
	case chEv := <-a.DegradationEvents:
		if chEv.DroppedSeq != 3 {
			t.Errorf("channel DegradationEvent.DroppedSeq: want 3, got %d", chEv.DroppedSeq)
		}
	default:
		t.Fatal("TLPKTDROP(3): expected channel event, got nothing")
	}
	assertNoPendingDeg(t, a.DegradationEvents)

	// Now ACK frames 1 and 2 — they must be delivered in order.
	// If nextExpected was wrongly leapfrogged to 3, OnAck(1) and OnAck(2) will
	// not produce any delivery (the bug this test catches).
	frames1, err := a.OnAck(1, zeroBitmap())
	if err != nil {
		t.Fatalf("OnAck(1) after TLPKTDROP(3): %v", err)
	}
	if len(frames1) != 1 {
		t.Fatalf("OnAck(1): want 1 frame, got %d", len(frames1))
	}
	if string(frames1[0]) != "frame-1" {
		t.Errorf("frame 1: want %q, got %q", "frame-1", frames1[0])
	}

	frames2, err := a.OnAck(2, zeroBitmap())
	if err != nil {
		t.Fatalf("OnAck(2) after TLPKTDROP(3): %v", err)
	}
	if len(frames2) != 1 {
		t.Fatalf("OnAck(2): want 1 frame, got %d", len(frames2))
	}
	if string(frames2[0]) != "frame-2" {
		t.Errorf("frame 2: want %q, got %q", "frame-2", frames2[0])
	}

	// No additional frames should be delivered (seq=3 was dropped).
	frames3, err := a.OnAck(3, zeroBitmap())
	if err != nil {
		t.Fatalf("OnAck(3) after TLPKTDROP(3): %v", err)
	}
	if len(frames3) != 0 {
		t.Errorf("OnAck(3) after drop: want 0 frames, got %d", len(frames3))
	}
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

	ev, err := a.TLPKTDROP(7, nowBeforeDeadline)
	if !errors.Is(err, arq.ErrFrameNotOverdue) {
		t.Errorf("TLPKTDROP before deadline: want ErrFrameNotOverdue, got %v", err)
	}
	// Returned event must be zero-value (DroppedSeq == 0 is the no-event sentinel).
	if ev.DroppedSeq != 0 {
		t.Errorf("TLPKTDROP before deadline: want zero DegradationEvent, got %+v", ev)
	}

	// No degradation event should have been emitted on the channel.
	assertNoPendingDeg(t, a.DegradationEvents)
}

// TestBC_2_02_006_OnlyOverdue_TableDriven tests boundary conditions around the
// drop deadline (just-before, exactly-at, just-after).
func TestBC_2_02_006_OnlyOverdue_TableDriven(t *testing.T) {
	t.Parallel()

	const dropTimeout = 200 * time.Millisecond
	sendTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

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

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			a := newTestARQ(dropTimeout)
			a.EnqueueSend(1, []byte("payload"), sendTime)

			now := sendTime.Add(tc.nowOffset)
			ev, err := a.TLPKTDROP(1, now)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("want %v, got %v", tc.wantErr, err)
				}
				// Zero-value DegradationEvent returned on error.
				if ev.DroppedSeq != 0 {
					t.Errorf("want zero DegradationEvent on error, got %+v", ev)
				}
				assertNoPendingDeg(t, a.DegradationEvents)
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tc.wantDegrade {
					if ev.DroppedSeq != 1 {
						t.Errorf("returned DegradationEvent.DroppedSeq: want 1, got %d", ev.DroppedSeq)
					}
					select {
					case chEv := <-a.DegradationEvents:
						if chEv.DroppedSeq != 1 {
							t.Errorf("channel DegradationEvent.DroppedSeq: want 1, got %d", chEv.DroppedSeq)
						}
					default:
						t.Fatal("expected channel DegradationEvent, got nothing")
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
	ev, err := a.TLPKTDROP(99, now)
	if err != nil {
		t.Fatalf("TLPKTDROP(99) during failover: %v", err)
	}
	if ev.DroppedSeq != 99 {
		t.Errorf("TLPKTDROP(99) return: want DroppedSeq=99, got %d", ev.DroppedSeq)
	}

	// Degradation signal must also be emitted on the channel.
	select {
	case chEv := <-a.DegradationEvents:
		if chEv.DroppedSeq != 99 {
			t.Errorf("channel DegradationEvent.DroppedSeq: want 99, got %d", chEv.DroppedSeq)
		}
	default:
		t.Fatal("TLPKTDROP(99): expected channel DegradationEvent, got nothing")
	}

	// ADR-005 resync: after failover reconnect, send new frame at seq=100.
	// The ARQ must accept and deliver it without error.
	a.EnqueueSend(100, []byte("post-failover"), now)
	frames, err := a.OnAck(100, zeroBitmap())
	if err != nil {
		t.Fatalf("OnAck(100) after failover resync: %v", err)
	}
	if len(frames) != 1 {
		t.Fatalf("OnAck(100): want 1 frame, got %d", len(frames))
	}
	if string(frames[0]) != "post-failover" {
		t.Errorf("expected 'post-failover' after resync, got %q", frames[0])
	}
}

// ─── VP-019/020/021: property-based no-double-delivery, in-order ─────────────

// TestBC_2_02_005_VP019_VP020_NoDoubleDelivery is a table-driven property test
// for VP-019 and VP-020: across varied delivery orderings, no frame is ever
// delivered twice and all frames are delivered in order.
//
// Exercises VP-019 (no double delivery) and VP-020 (in-order invariant).
// Uses 24 permutations of a 4-frame window (exhaustive for this window size).
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

			// Simulate frames arriving in the given order. For each step, build
			// a cumulative ACK = highest contiguous seq received, and SACK bitmap
			// for out-of-order frames. Collect all delivered frames from return values.
			received := make(map[uint32]bool)
			var delivered []uint32
			for _, seq := range perm {
				received[seq] = true
				cumACK := cumulativeACK(received)
				sack := buildSACKFromReceived(received, cumACK)
				frames, err := a.OnAck(cumACK, sack)
				if err != nil {
					t.Fatalf("OnAck(%d): %v", cumACK, err)
				}
				for _, f := range frames {
					if len(f) != 1 {
						t.Fatalf("unexpected payload length %d", len(f))
					}
					delivered = append(delivered, uint32(f[0]))
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
		ev, err := a.TLPKTDROP(i, now)
		if err != nil {
			t.Fatalf("TLPKTDROP(%d): %v", i, err)
		}
		if ev.DroppedSeq != i {
			t.Errorf("drop %d: returned DroppedSeq: want %d, got %d", i, i, ev.DroppedSeq)
		}
		// Drain the channel to keep the buffer clear for subsequent drops.
		select {
		case chEv := <-a.DegradationEvents:
			if chEv.DroppedSeq != i {
				t.Errorf("drop %d: channel DroppedSeq: want %d, got %d", i, i, chEv.DroppedSeq)
			}
		default:
			t.Fatalf("drop %d: expected channel DegradationEvent, got nothing", i)
		}
	}

	// After 10 drops, the session must still process the next frame normally.
	a.EnqueueSend(11, []byte("alive"), sendTime)
	frames, err := a.OnAck(11, zeroBitmap())
	if err != nil {
		t.Fatalf("OnAck(11) after 10 drops: %v", err)
	}
	if len(frames) != 1 {
		t.Fatalf("OnAck(11): want 1 frame, got %d", len(frames))
	}
	if string(frames[0]) != "alive" {
		t.Errorf("expected 'alive' after 10 drops, got %q", frames[0])
	}
}

// ─── large-scale property test (VP-019/020, 1000+ cases) ─────────────────────

// TestBC_2_02_005_VP019_VP020_LargeScale is a randomised property test that
// covers >1000 delivery-order scenarios using a linear congruential generator
// (no external dependencies, pure stdlib).
//
// For each trial: enqueue N frames, apply OnAck calls in a random permutation,
// verify all frames delivered exactly once in-order. Delivery is collected
// directly from OnAck return values — no channel drain, no time.After races.
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
		var delivered []uint32
		for _, seq := range seqs {
			received[seq] = true
			cumACK := cumulativeACK(received)
			sack := buildSACKFromReceived(received, cumACK)
			frames, err := a.OnAck(cumACK, sack)
			if err != nil {
				t.Fatalf("trial %d seq %d: OnAck(%d): %v", trial, seq, cumACK, err)
			}
			for _, f := range frames {
				delivered = append(delivered, uint32(f[0]))
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

// ─── EC-004 / F-H4: cumulative ACK past locally-absent sequence ───────────────

// TestARQ_OnAck_CumulativeAckPastLocallyAbsentSeq pins BC-2.02.005 invariant 4
// and the PC-4 scope note (F-H4 ruling, disposition A): when a single cumulative
// ACK scans across a sequence number for which payloadFor returns nil (frame not
// in inFlight or reorderBuf because it was never enqueued), the sender must
// advance nextExpected past it without error and without holding the gap.
//
// This is a characterisation test: it pins already-conformant behaviour in the
// current implementation. If it fails, the implementation diverges from
// invariant 4 of BC-2.02.005 v1.2.
//
// Canonical BC-2.02.005 v1.2 sender-semantics test vector:
//
//	"Sender sends frames 1,2,3; frame 2 cleaned from inFlight by prior partial
//	ACK; cumulative ACK for seq=3 arrives → nextExpected advances past 3;
//	inFlight emptied; no error or gap event emitted."
//
// Here we use the cleanest faithful encoding of that vector: seq=2 is NEVER
// enqueued (simulates "absent at ACK time" regardless of cause). A single
// OnAck(3, zeroBitmap) must scan seq=1 (deliver), seq=2 (nil → skip+advance),
// seq=3 (deliver). This directly exercises the payload==nil silent-skip branch
// in the Step-1 loop (arq.go ~line 208) — the exact subject of EC-004 /
// F-H4 / Task 10.
//
// Scenario:
//   - Fresh ARQ state (nextExpected=0).
//   - EnqueueSend for seq=1 ("frame-1") and seq=3 ("frame-3") only.
//   - seq=2 is NEVER enqueued → absent from both inFlight and reorderBuf.
//   - Single call: OnAck(3, zeroBitmap).
//
// Expected:
//   - Returns exactly [frame-1, frame-3] (frame-2 absent — advanced past, not held).
//   - nextExpected advanced to 3 (verified: idempotent OnAck(3) returns 0 frames).
//   - No error returned.
//   - No DegradationEvent emitted.
func TestARQ_OnAck_CumulativeAckPastLocallyAbsentSeq(t *testing.T) {
	t.Parallel()

	a := newTestARQ(500 * time.Millisecond)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Enqueue seq=1 and seq=3 only. seq=2 is deliberately absent — it was never
	// sent (or was already cleaned from inFlight by a prior partial ACK window).
	// This puts the ARQ in the exact state described by the BC v1.2 vector.
	a.EnqueueSend(1, []byte("frame-1"), now)
	a.EnqueueSend(3, []byte("frame-3"), now)

	// Single cumulative ACK spanning seq=1..3 with no SACK bits set.
	// Step-1 loop in OnAck iterates:
	//   seq=1: payloadFor(1) = "frame-1" → deliver, advance nextExpected=1
	//   seq=2: payloadFor(2) = nil        → skip (no delivery), advance nextExpected=2
	//   seq=3: payloadFor(3) = "frame-3" → deliver, advance nextExpected=3
	// This directly exercises the payload==nil silent-skip+advance path (EC-004).
	frames, err := a.OnAck(3, zeroBitmap())
	if err != nil {
		t.Fatalf("OnAck(3) across locally-absent seq=2: unexpected error: %v", err)
	}
	if len(frames) != 2 {
		t.Fatalf("OnAck(3): want 2 frames (frame-1, frame-3), got %d: %v", len(frames), frames)
	}
	if string(frames[0]) != "frame-1" {
		t.Errorf("OnAck(3): frames[0]: want %q, got %q", "frame-1", string(frames[0]))
	}
	if string(frames[1]) != "frame-3" {
		t.Errorf("OnAck(3): frames[1]: want %q, got %q", "frame-3", string(frames[1]))
	}

	// Idempotency: nextExpected is now 3; a repeat OnAck(3) must return nothing.
	// This verifies that nextExpected advanced all the way to ackSeq=3
	// (i.e., the absent seq=2 did not stall advancement at seq=1).
	idem, err := a.OnAck(3, zeroBitmap())
	if err != nil {
		t.Fatalf("idempotent OnAck(3): unexpected error: %v", err)
	}
	if len(idem) != 0 {
		t.Errorf("idempotent OnAck(3): want 0 frames, got %d", len(idem))
	}

	// No degradation events: absent seq=2 is not a loss event on the sender side
	// (invariant 4 — the cumulative ACK proves remote receipt).
	assertNoPendingDeg(t, a.DegradationEvents)
}

// TestARQ_OnAck_SACKWithoutCumulativeAdvance_RecoversOnNextCumulativeAck pins
// the Step-3 flush-guard caller-contract and eventual-recovery behaviour
// (arq.go ~line 238: `if ackSeq > prevNextExpected`).
//
// The flush guard requires the cumulative ACK to have advanced before it will
// deliver frames buffered by the SACK bitmap in Step 2. This documents the
// guard's assumption: conformant callers always advance the cumulative ACK
// when a gap fills. A non-conformant SACK-only call (cumulative not advanced)
// holds the buffered frame rather than delivering it immediately — eventual
// recovery occurs on the next cumulative advance.
//
// This is a characterisation test: it pins already-conformant behaviour.
// If it fails, the Step-3 guard has regressed (frames delivered prematurely
// before cumulative advance, or permanently lost on eventual recovery).
//
// Scenario:
//  1. EnqueueSend seq=1 and seq=2.
//  2. OnAck(0, sack={bit-0 set}) — cumulative=0 (no advance), SACK marks seq=1
//     as received out-of-order. Step 2 buffers seq=1 in reorderBuf. Step 3
//     guard fires: ackSeq(0) == prevNextExpected(0), so flush is skipped.
//     Assert: 0 frames delivered.
//  3. OnAck(1, zeroBitmap) — cumulative advances to 1. Step 1 delivers seq=1
//     from reorderBuf (payloadFor finds it there). Step 3 flush: tries seq=2,
//     not in reorderBuf → stops. Assert: [frame-1] delivered (in order).
//
// Note: step 2 uses sack bit-0 which covers ackSeq+1 = 0+1 = 1.
func TestARQ_OnAck_SACKWithoutCumulativeAdvance_RecoversOnNextCumulativeAck(t *testing.T) {
	t.Parallel()

	a := newTestARQ(500 * time.Millisecond)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	a.EnqueueSend(1, []byte("frame-1"), now)
	a.EnqueueSend(2, []byte("frame-2"), now)

	// Step 1: SACK-only call — cumulative ACK stays at 0, SACK bit-0 marks seq=1
	// as received. The flush guard (ackSeq == prevNextExpected == 0) prevents
	// delivery even though seq=1 is now in reorderBuf.
	sackSeq1 := bitmapWithBits(0) // bit 0 → ackSeq+1+0 = 0+1 = 1
	held, err := a.OnAck(0, sackSeq1)
	if err != nil {
		t.Fatalf("SACK-only OnAck(0): unexpected error: %v", err)
	}
	if len(held) != 0 {
		t.Errorf("SACK-only OnAck(0): want 0 frames (flush guard active), got %d: %v", len(held), held)
	}

	// Step 2: cumulative ACK advances to 1. Step 1 loop visits seq=1:
	// payloadFor(1) finds it in reorderBuf (buffered by the prior SACK), delivers
	// it, and advances nextExpected=1. Step 3 tries seq=2 (not in reorderBuf),
	// stops. Frame-1 is recovered in order — no permanent loss.
	recovered, err := a.OnAck(1, zeroBitmap())
	if err != nil {
		t.Fatalf("cumulative OnAck(1): unexpected error: %v", err)
	}
	if len(recovered) != 1 {
		t.Fatalf("cumulative OnAck(1): want 1 frame (frame-1), got %d: %v", len(recovered), recovered)
	}
	if string(recovered[0]) != "frame-1" {
		t.Errorf("cumulative OnAck(1): want %q, got %q", "frame-1", string(recovered[0]))
	}

	// No degradation events: the held-then-recovered frame is not a loss event.
	assertNoPendingDeg(t, a.DegradationEvents)
}

// ─── O-3: reorderBuf must not grow unbounded ─────────────────────────────────

// TestARQ_ReorderBuf_BoundedByWindowSize verifies that the reorder buffer does
// not grow unbounded when far-future out-of-order frames beyond the configured
// window are submitted.
//
// This test verifies only that reorderBuf is bounded by the SACK window (64
// positions above ackSeq). The inFlight map has no window bound in this package;
// window enforcement is deferred to S-5.01. See inFlight field comment.
//
// BC-2.02.005 invariant: frames outside the ARQ window are not retained.
// The SACK bitmap covers exactly 64 positions above ackSeq — frames at
// position >= 64 are outside the window.
func TestARQ_ReorderBuf_BoundedByWindowSize(t *testing.T) {
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
	frames0, err := a.OnAck(0, allBitsSet)
	if err != nil {
		t.Fatalf("OnAck(0, allBitsSet): %v", err)
	}
	// Nothing is delivered yet — nextExpected is still 0 and the cumulative ACK
	// is 0, so the reorderBuf flush cannot advance past nextExpected=0 without
	// a cumulative ACK for seq=1.
	if len(frames0) != 0 {
		t.Errorf("OnAck(0, allBitsSet): want 0 delivered (nextExpected=0), got %d", len(frames0))
	}

	// ACK seq=1 cumulatively to trigger flush. Seq 1 flushes, then the
	// consecutive reorderBuf entries flush through 64.
	frames1, err := a.OnAck(1, zeroBitmap())
	if err != nil {
		t.Fatalf("OnAck(1): %v", err)
	}
	// All windowSize (64) frames should be delivered in this single call:
	// seq=1 flushes via cumulative ACK, then seq 2..64 flush from reorderBuf.
	// If the implementation stored seqs 65..200 in reorderBuf they would also
	// flush here — that would be the unbounded-growth bug this test catches.
	if len(frames1) != windowSize {
		t.Errorf("OnAck(1) flush: expected %d delivered (window-bounded), got %d — possible unbounded reorderBuf growth",
			windowSize, len(frames1))
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
