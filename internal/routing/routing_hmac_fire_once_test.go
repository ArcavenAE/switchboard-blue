// Package routing_test — SW305-M4 / W4-TEST-001 integration coverage for the
// RouteFrame + real admission.FailureCounter fire-once contract.
//
// # Coverage gap this file closes
//
// TestRouteFrame_FiveConsecutiveFailures_TriggersEADM017 (in
// routing_hmac_counter_test.go) proves the 5th failure emits E-ADM-017 exactly
// once, but stops there. That leaves three properties of the fire-once
// contract (BC-2.05.005 v1.6, error-taxonomy E-ADM-017) uncovered at the
// integration seam:
//
//  1. **Fire-once at threshold crossing** — the alert fires exactly ONCE on
//     the Nth failure, not on every Nth+K failure.
//  2. **Suppression within window** — continued failures at N+1, N+2, … while
//     firedAt[srcAddr] is set must NOT re-fire the alert.
//  3. **Drain-only re-arm** — after the sliding window fully drains
//     (all pre-fire timestamps aged out), a fresh threshold crossing MUST
//     produce a second alert.
//
// Unit coverage of the FailureCounter itself lives in
// internal/admission/failure_counter_test.go and
// failure_counter_adversarial_test.go. This file wires the REAL counter into
// a REAL Router via routing.WithFailureCounter and drives every code path
// through RouteFrame — the operator-visible entry point — so the integration
// contract is pinned end-to-end.
//
// Time is controlled deterministically via admission.WithNow(fn) — no sleeps.
//
// Run with `-race` and `-count=5` per SW305-M4 instructions.
//
// Traces to: BC-2.05.005 v1.6 (drain-only re-arm, append-skip); BC-2.05.008
// PC-5 + invariant 5 (RouteFrame → RecordHMACFailure wire-up); error-taxonomy
// v1.9 E-ADM-017 (fire-once).
package routing_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/hmac"
	"github.com/arcavenae/switchboard/internal/routing"
)

// fireOnceHarness bundles the pieces every fire-once test needs.
//
// Each test builds a distinct one so t.Parallel() sub-tests don't share
// mutable state.
type fireOnceHarness struct {
	router   *routing.Router
	alertLog *fakeAlertLog
	hdr      frame.OuterHeader
	payload  []byte
	srcKey   string
	current  *time.Time // clock cell — mutated via advance()
	base     time.Time
}

// newFireOnceHarness builds a Router wired to a real FailureCounter with
// deterministic clock. threshold and window are exposed so tests can dial
// them (e.g. tight window to keep test data compact).
//
// The frame it prepares has an HMAC tag computed under a WRONG key so every
// RouteFrame call in the test returns ErrHMACVerificationFailed — the failure
// path is the one that drives RecordHMACFailure (BC-2.05.008 PC-5).
func newFireOnceHarness(t *testing.T, svtnTag string, threshold int, window time.Duration) *fireOnceHarness {
	t.Helper()

	var svtnID [16]byte
	if len(svtnTag) > 16 {
		t.Fatalf("newFireOnceHarness: svtnTag %q > 16 bytes", svtnTag)
	}
	copy(svtnID[:], svtnTag)

	alertLog := &fakeAlertLog{}

	base := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	current := base

	fc := admission.NewFailureCounter(threshold, window, alertLog,
		admission.WithNow(func() time.Time { return current }),
	)

	// Distinct byte seeds per harness so parallel sub-tests do not share
	// admitted-set / key material.
	seedByte := byte(len(svtnTag) + 0x40)
	ks, srcAddr, authKey := buildAdmittedKS(t, svtnID, seedByte, seedByte^0x20)

	var dstAddr [8]byte
	copy(dstAddr[:], "sw305dst")

	r := routing.NewRouter(ks,
		routing.WithLogger(&routingFakeLog{}),
		routing.WithFailureCounter(fc),
	)
	r.RegisterForwardingEntry(svtnID, srcAddr, authKey)
	r.RegisterForwardingEntry(svtnID, dstAddr, [hmac.KeySize]byte{0x77})

	// Bad tag computed under a wrong key — every RouteFrame call fails
	// verifyFrameHMAC (BC-2.05.008 PATH-B).
	var wrongKey [hmac.KeySize]byte
	copy(wrongKey[:], "sw305-wrong-key-32-bytes-000000")

	payload := []byte("sw305-payload")
	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtnID,
		SrcAddr:   srcAddr,
		DstAddr:   dstAddr,
	}
	hdr.HMACTag = computeValidTag(hdr, payload, wrongKey)

	return &fireOnceHarness{
		router:   r,
		alertLog: alertLog,
		hdr:      hdr,
		payload:  payload,
		srcKey:   fmt.Sprintf("%x", srcAddr),
		current:  &current,
		base:     base,
	}
}

// advance sets the harness clock to base + d.
func (h *fireOnceHarness) advance(d time.Duration) {
	*h.current = h.base.Add(d)
}

// routeExpectHMACFail drives one RouteFrame call and asserts the path
// returns ErrHMACVerificationFailed (the fire-once contract only holds when
// the failure path is actually taken).
func (h *fireOnceHarness) routeExpectHMACFail(t *testing.T, callIdx int) {
	t.Helper()
	err := routing.RouteFrame(h.hdr, h.payload, h.router)
	if !errors.Is(err, routing.ErrHMACVerificationFailed) {
		t.Fatalf("call %d: want ErrHMACVerificationFailed (bad tag path), got %v", callIdx, err)
	}
}

// assertAlertCount asserts alertLog.Count() equals want, printing the
// captured lines on failure for easier diagnosis.
func (h *fireOnceHarness) assertAlertCount(t *testing.T, want int, at string) {
	t.Helper()
	if got := h.alertLog.Count(); got != want {
		t.Errorf("%s: want alertLog.Count()==%d, got %d; lines: %v", at, want, got, h.alertLog.Lines())
	}
}

// ── SW305-M4 (a): fire-once at threshold crossing ─────────────────────────────

// TestRouteFrame_EADM017_FiresOnceAtThresholdCrossing pins the primary
// fire-once property: with threshold=5, RouteFrame call 4 (four failures)
// produces zero alerts, RouteFrame call 5 (fifth failure at threshold)
// produces exactly one alert, and NO extra alert appears from the exact
// crossing itself.
//
// This is a stronger assertion than the existing 5→1 test — that test only
// checks the terminal state. Here we assert the pre-threshold state is 0
// and the crossing state is exactly 1, pinning the boundary.
//
// Traces to: BC-2.05.005 v1.6 PC-3 (fire-once on threshold crossing);
// BC-2.05.008 PC-5.
func TestRouteFrame_EADM017_FiresOnceAtThresholdCrossing(t *testing.T) {
	t.Parallel()

	h := newFireOnceHarness(t, "sw305-fire1-svtn", 5, 60*time.Second)

	// Calls 1..4: failures accumulate but alert must NOT fire.
	for i := range 4 {
		h.advance(time.Duration(i) * time.Second)
		h.routeExpectHMACFail(t, i+1)
	}
	h.assertAlertCount(t, 0, "SW305-M4 (a) pre-threshold")

	// Call 5: threshold crossing → exactly one alert.
	h.advance(4 * time.Second)
	h.routeExpectHMACFail(t, 5)
	h.assertAlertCount(t, 1, "SW305-M4 (a) at threshold")

	// Alert must carry the canonical E-ADM-017 substring.
	if !h.alertLog.HasAll("E-ADM-017", "HMAC failure rate alert") {
		t.Errorf("SW305-M4 (a): alert missing canonical E-ADM-017 substring; lines: %v",
			h.alertLog.Lines())
	}
	// Alert must reference the src key format used by RouteFrame
	// (fmt.Sprintf("%x", hdr.SrcAddr) per BC-2.05.008 invariant 5).
	found := false
	for _, line := range h.alertLog.Lines() {
		if strings.Contains(line, h.srcKey) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("SW305-M4 (a): alert does not reference src key %q; lines: %v",
			h.srcKey, h.alertLog.Lines())
	}
}

// ── SW305-M4 (b): no re-fire during window ───────────────────────────────────

// TestRouteFrame_EADM017_DoesNotRefireDuringWindow pins the append-skip
// suppression property: after threshold crossing at call 5, calls 6..10 that
// each also fail HMAC verification within the same 60s window MUST NOT
// produce additional alerts.
//
// This is BC-2.05.005 v1.6 EC-011: while firedAt[srcAddr] is set (fired at
// call 5 clock time), new failure timestamps are NOT appended, threshold is
// NOT re-crossed, and the alert stays at exactly 1.
//
// Traces to: BC-2.05.005 v1.6 (append-skip EC-011); error-taxonomy E-ADM-017
// fire-once semantics.
func TestRouteFrame_EADM017_DoesNotRefireDuringWindow(t *testing.T) {
	t.Parallel()

	const threshold = 5
	const window = 60 * time.Second
	h := newFireOnceHarness(t, "sw305-fire2-svtn", threshold, window)

	// Cross the threshold: 5 failures at T=0..4.
	for i := range threshold {
		h.advance(time.Duration(i) * time.Second)
		h.routeExpectHMACFail(t, i+1)
	}
	h.assertAlertCount(t, 1, "SW305-M4 (b) at threshold")

	// Continued failures within the window: calls 6..15 at T=10s, 20s, 30s,
	// 40s, 50s, and then five more at T=51s..55s (still inside 60s window
	// because cutoff at T=55 is T=55-60 = -5s → all pre-fire T=0..4 entries
	// remain in the window → no drain → no re-arm).
	//
	// Under append-skip, firedAt is set from call 5, so calls 6..N do NOT
	// append. Alert must stay at 1.
	continuationTimes := []time.Duration{
		10 * time.Second, 20 * time.Second, 30 * time.Second, 40 * time.Second, 50 * time.Second,
		51 * time.Second, 52 * time.Second, 53 * time.Second, 54 * time.Second, 55 * time.Second,
	}
	for i, dt := range continuationTimes {
		h.advance(dt)
		h.routeExpectHMACFail(t, threshold+i+1)
	}
	h.assertAlertCount(t, 1, "SW305-M4 (b) continued failures within window")
}

// ── SW305-M4 (c): re-arm after window drain ──────────────────────────────────

// TestRouteFrame_EADM017_RearmsAfterWindowDrain pins the drain-only re-arm
// property (BC-2.05.005 v1.6 PC-3): after the sliding window fully drains
// (all pre-fire timestamps aged out), a fresh threshold crossing MUST fire
// a SECOND E-ADM-017 alert.
//
// Scenario (threshold=5, window=60s):
//
//	T=0:      5 failures at same instant → alert #1. firedAt[src]=T=0.
//	          append-skip in force; slice=[T=0 ×5].
//	T=61s:    call 6 — cutoff=T=1; all T=0 entries < T=1 → drained.
//	          len(keep)==0 → re-arm triggers → firedAt cleared → T=61
//	          appended (count=1, no alert).
//	T=61..65: 4 more failures → threshold re-crossed → alert #2.
//
// Timing invariant (drain-only re-arm + append-skip):
// Batch-2's FIRST call must occur strictly MORE than windowDuration after
// the LAST batch-1 entry so the window fully drains on that call. Batch-1
// all at T=0 → last entry T=0 → batch-2 starts at T=61 > T=0+60s ✓.
//
// (This mirrors the pattern in failure_counter_adversarial_test.go's
// TestFailureCounter_SustainedAttackReFires but drives it through RouteFrame
// so the integration path is the one under test.)
//
// Traces to: BC-2.05.005 v1.6 PC-3 (drain-only re-arm); BC-2.05.005 EC-009
// (periodic re-fire under sustained attack); error-taxonomy E-ADM-017.
func TestRouteFrame_EADM017_RearmsAfterWindowDrain(t *testing.T) {
	t.Parallel()

	const threshold = 5
	const window = 60 * time.Second
	h := newFireOnceHarness(t, "sw305-fire3-svtn", threshold, window)

	// Batch 1: 5 failures at T=0 (same instant → last entry timestamp = T=0).
	for i := range threshold {
		h.advance(0) // all at T=0 so the window fully drains at T=61
		h.routeExpectHMACFail(t, i+1)
	}
	h.assertAlertCount(t, 1, "SW305-M4 (c) batch 1 at threshold")

	// Advance to T=61s — cutoff = T=1 → all batch-1 entries (T=0) < T=1
	// → drained → re-arm on this call → T=61 appended (count=1, no alert).
	h.advance(61 * time.Second)
	h.routeExpectHMACFail(t, threshold+1)
	h.assertAlertCount(t, 1, "SW305-M4 (c) first call after drain (re-arm, no re-fire)")

	// Batch 2: 4 more failures at T=62s..65s → count=5 at call T=65s
	// → threshold re-crossed → alert #2.
	for i := range 4 {
		h.advance((62 + time.Duration(i)) * time.Second)
		h.routeExpectHMACFail(t, threshold+1+i+1)
	}
	h.assertAlertCount(t, 2, "SW305-M4 (c) batch 2 re-crosses threshold")

	// Both alerts should carry the canonical E-ADM-017 substring.
	if !h.alertLog.HasAll("E-ADM-017", "HMAC failure rate alert") {
		t.Errorf("SW305-M4 (c): alert log missing canonical E-ADM-017 substring; lines: %v",
			h.alertLog.Lines())
	}
}
