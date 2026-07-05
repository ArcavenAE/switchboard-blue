// Package arq_test — pure-core accessors for the retransmit-SEND path
// (S-BL.ARQ-TX). BC-2.02.005 PC-3 requires the sender to retransmit missing
// content when a SACK gap is detected. The boundary-layer sender lives in
// internal/arqsend; it needs read-and-remove access to the in-flight queue.
//
// Two accessors added here:
//
//	PayloadForInFlight(seq) []byte — returns a defensive copy of the
//	  in-flight payload, or nil when absent. The defensive copy honours
//	  go.md rule 12 (never return internal pointers from locked/managed
//	  state); ARQ retains no reference to the returned slice after return.
//	RemoveInFlight(seq)             — idempotent delete. Boundary layer
//	  calls this when a retransmit under a new seq has taken ownership
//	  of the payload (QUIC retransmit model, BC-2.02.005 PC-5: retransmit
//	  carries content under a NEW seq, so the OLD seq must be released).
//
// Both accessors preserve the pure-core classification: no I/O, no
// goroutines, no clocks. They exist to let a caller wire retransmit-SEND
// without importing effectful state into internal/arq.
package arq_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/arq"
)

// TestARQ_PayloadForInFlight_ReturnsCopy exercises the new PayloadForInFlight
// accessor. The returned slice must byte-equal the original payload and must
// not share the backing array with ARQ's internal state (go.md rule 12).
func TestARQ_PayloadForInFlight_ReturnsCopy(t *testing.T) {
	t.Parallel()

	a := arq.New(arq.Config{DropTimeout: time.Second})
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	orig := []byte("original-payload")
	a.EnqueueSend(5, orig, now)

	got := a.PayloadForInFlight(5)
	if !bytes.Equal(got, orig) {
		t.Fatalf("PayloadForInFlight(5): want %q, got %q", orig, got)
	}

	// Mutating the returned slice must not affect ARQ's internal copy.
	got[0] = 'X'
	got2 := a.PayloadForInFlight(5)
	if !bytes.Equal(got2, orig) {
		t.Fatalf("second PayloadForInFlight(5) leaked mutation: want %q, got %q", orig, got2)
	}
}

// TestARQ_PayloadForInFlight_AbsentReturnsNil documents the "not in flight"
// signal used by the boundary layer to short-circuit on unknown seqs.
func TestARQ_PayloadForInFlight_AbsentReturnsNil(t *testing.T) {
	t.Parallel()

	a := arq.New(arq.Config{DropTimeout: time.Second})

	if got := a.PayloadForInFlight(42); got != nil {
		t.Fatalf("PayloadForInFlight(42) on empty ARQ: want nil, got %v", got)
	}
}

// TestARQ_RemoveInFlight_Idempotent verifies RemoveInFlight deletes the
// in-flight entry for a known seq and is a no-op for an absent seq.
func TestARQ_RemoveInFlight_Idempotent(t *testing.T) {
	t.Parallel()

	a := arq.New(arq.Config{DropTimeout: time.Second})
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	a.EnqueueSend(7, []byte("data"), now)
	if !a.InFlightContains(7) {
		t.Fatalf("precondition failed: seq 7 not in flight after EnqueueSend")
	}

	a.RemoveInFlight(7)
	if a.InFlightContains(7) {
		t.Fatalf("RemoveInFlight(7): seq still present")
	}

	// Idempotent — removing again must be a no-op.
	a.RemoveInFlight(7)
	if a.InFlightContains(7) {
		t.Fatalf("RemoveInFlight(7) second call: seq resurrected")
	}

	// Absent seq — must not panic and must not error.
	a.RemoveInFlight(999)
}
