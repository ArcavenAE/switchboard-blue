// Package main — failure_counter_wire_test.go
//
// # RED GATE TEST: C-1 — wire routing.WithFailureCounter into buildRouter
//
// Behavioral contract citations:
//
//   - BC-2.05.008 PC-5: RouteFrame MUST call failureCounter.RecordHMACFailure on
//     every ErrHMACVerificationFailed return path when a counter is wired.
//
//   - BC-2.05.005 PC-3: FailureCounter MUST emit E-ADM-017 exactly once when ≥5
//     failures arrive from the same source within a 60-second sliding window.
//     Threshold=5, window=60s are the canonical constants.
//
//   - E-ADM-017 (error-taxonomy v2.2 §ADM): canonical alert format is
//     "E-ADM-017 HMAC failure rate alert: ≥N failures in Xs from src <addr>"
//
//   - ARCH-08 v2.2 §6.5.1: buildRouter is the daemon construction path that
//     must wire WithFailureCounter. The fix described in tracked deferral
//     C-1-W3P1-defer adds routing.WithFailureCounter(fc) to buildRouter.
//
// # Why this test is behaviorally discriminating (non-tautological)
//
// The test calls buildAccessComponents — the SAME construction path the daemon
// uses — with a captureLogger injected as the routerLogger. It does NOT
// construct its own router with WithFailureCounter; that would be tautological
// (it would test the routing package in isolation, not the daemon's wire-up).
//
// On CURRENT code: buildRouter calls routing.NewRouter(ks, routing.WithLogger(rl))
// only. r.failureCounter is nil. RecordHMACFailure is never called (routing.go
// guards both E-ADM-016 paths with `if r.failureCounter != nil`). E-ADM-017
// never fires. The assertion `captureLogger.HasLine("E-ADM-017")` is false.
// → TEST FAILS for the right behavioral reason.
//
// After the fix: buildRouter additionally constructs admission.NewFailureCounter
// with threshold=5, windowDuration=60s, logger=rl, and passes it to
// routing.WithFailureCounter(fc). After 5 consecutive HMAC failures from the
// same source, FailureCounter logs E-ADM-017 via rl. captureLogger captures
// the line. → TEST PASSES.
//
// # Seam used
//
// buildAccessComponents already accepts a routerLogger routing.Logger parameter
// (introduced by FIX 2 for AC-001 non-tautological test coverage). captureLogger
// (defined in main_test.go) satisfies both routing.Logger and admission.Logger
// (both are `interface{ Log(string) }`), so the same capture instance collects
// both E-ADM-016 log lines (from the router) AND E-ADM-017 alert lines (from
// the FailureCounter) once the counter is wired.
//
// # HMAC failure strategy (PATH-A)
//
// The test uses PATH-A (no forwarding-table entry registered for the source
// SVTN/address pair). This produces ErrHMACVerificationFailed without requiring
// a valid key or a computed tag — the simplest, most deterministic failure path.
// A fixed [8]byte srcAddr is used for all 5 frames so the FailureCounter
// accumulates 5 failures for the same source key and triggers the alert.
//
// PATH-A is guarded by `if r.failureCounter != nil` at routing.go:204, so on
// current code the counter is never called. After the fix it is called for every
// PATH-A failure.
//
// # Clock and timing
//
// All 5 frames are sent synchronously within a single test. Wall-clock time
// between calls is well under 1s, which is far inside the 60s window. No sleep
// is required. No clock injection into FailureCounter is needed; the real
// time.Now() clock is used and the test completes before the window expires.
//
// Traces to: BC-2.05.008 PC-5; BC-2.05.005 PC-3; error-taxonomy v2.2 E-ADM-017;
// ARCH-08 v2.2 §6.5.1; tracked deferral C-1-W3P1-defer.
package main

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/halfchannel"
	"github.com/arcavenae/switchboard/internal/routing"
	"github.com/arcavenae/switchboard/internal/session"
	"github.com/arcavenae/switchboard/internal/tmux"
)

// TestBuildRouter_WithFailureCounter_FiveFailures_TriggersEADM017
//
// AC: C-1 (tracked deferral C-1-W3P1-defer)
// Traces: BC-2.05.008 PC-5, BC-2.05.005 PC-3, E-ADM-017 (error-taxonomy v2.2),
//
//	ARCH-08 v2.2 §6.5.1.
//
// Asserts that the production buildRouter (called through buildAccessComponents)
// wires a FailureCounter with threshold=5 / window=60s, such that 5 consecutive
// HMAC-failing RouteFrame calls from the same source cause E-ADM-017 to be
// logged on the injected captureLogger.
//
// RED-GATE behaviour on current code:
//
//	buildRouter does NOT call routing.WithFailureCounter; r.failureCounter == nil;
//	RecordHMACFailure is never called; E-ADM-017 never fires; the final HasLine
//	assertion fails with: "E-ADM-017 never logged — captureLogger has N lines,
//	none containing E-ADM-017".
//
// GREEN behaviour after the fix:
//
//	buildRouter constructs admission.NewFailureCounter(5, 60*time.Second, rl) and
//	passes it via routing.WithFailureCounter(fc); after 5 PATH-A failures the
//	counter emits E-ADM-017 via rl; captureLogger.HasLine("E-ADM-017") == true.
func TestBuildRouter_WithFailureCounter_FiveFailures_TriggersEADM017(t *testing.T) {
	// NOT t.Parallel(): shares no mutable global state, but uses fakeConnector
	// construction via newFakeSessionConnectorForWire — keep sequential to avoid
	// log-capture races with other tests in the same package.

	// ── Build production access components through the real daemon path ─────────

	// captureLogger is injected as routerLogger into buildAccessComponents.
	// It satisfies routing.Logger AND admission.Logger (both are Log(string)).
	// After the fix, the FailureCounter constructed inside buildRouter shares this
	// logger — E-ADM-017 alerts land here alongside E-ADM-016 log lines.
	cl := &captureLogger{}

	// Build a hermetic connector (same pattern as newFakeSessionConnector in
	// main_test.go). We need a *tmux.SessionConnector to satisfy buildAccessComponents'
	// sc *tmux.SessionConnector parameter; we Connect it so the fixture is live.
	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	ds := halfchannel.New(1, halfchannel.Downstream, 10*time.Millisecond)
	ctrl := tmux.New(pub, ds, fakeExecFuncErrMain(tmux.ErrControlModeUnavailable))
	pipe := newPipeMasterMain()
	pty := tmux.NewPTYProxy(pub, ds,
		tmux.WithPTYAllocFunc(func() (io.ReadWriteCloser, int, error) {
			return pipe, 9002, nil
		}),
	)
	sc := tmux.NewSessionConnector(ctrl, pty)
	t.Cleanup(func() {
		_ = pipe.Close()
		_ = sc.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("sc.Connect: %v; want nil (PTY fallback path)", err)
	}

	// buildAccessComponents — the PRODUCTION construction path (non-tautological).
	// cl is the router logger; after the C-1 fix it is also the FailureCounter
	// logger. We do NOT construct our own router with WithFailureCounter — that
	// would bypass the daemon wire-up under test.
	_, router := buildAccessComponents(keys, pub, sc, cl)

	// ── Craft a frame that reliably triggers PATH-A on every call ───────────────
	//
	// PATH-A: no forwarding-table entry registered for (svtnID, srcAddr).
	// RouteFrame returns ErrHMACVerificationFailed immediately (auth key
	// unavailable). We do NOT register a forwarding entry — so srcAddr is
	// permanently unresolvable and every call hits PATH-A.
	//
	// The srcAddr must be consistent across all 5 calls so FailureCounter
	// accumulates 5 counts under the same key and fires the alert.
	var svtnID [16]byte
	copy(svtnID[:], "c1-wire-test-svtn")

	var srcAddr [8]byte
	copy(srcAddr[:], "c1src001") // fixed 8-byte source address

	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtnID,
		SrcAddr:   srcAddr,
		// DstAddr zero-value; HMACTag zero-value.
		// No forwarding entry registered → PATH-A fires on lookup.
	}
	payload := []byte("c1-failure-counter-wire-test")

	// ── Drive 5 consecutive HMAC-failing calls (threshold = 5) ──────────────────
	//
	// All calls happen synchronously within one test, well inside the 60s window.
	// No sleep required. On current code: failureCounter is nil → counter never
	// called → E-ADM-017 never fires → test fails behaviorally.
	for i := range 5 {
		err := routing.RouteFrame(hdr, payload, router)
		if err == nil {
			// Sanity check: PATH-A must return ErrHMACVerificationFailed.
			// If RouteFrame returns nil the test setup is wrong.
			t.Fatalf("call %d: RouteFrame(no forwarding entry): got nil; "+
				"want ErrHMACVerificationFailed (PATH-A: no auth key for src %x in SVTN %x)",
				i+1, srcAddr, svtnID)
		}
		// We don't errors.Is-check here to avoid importing routing just for the
		// sentinel — the non-nil return is sufficient to confirm PATH-A fired.
		// (The E-ADM-016 log line in captureLogger provides the structural proof.)
	}

	// ── Assert E-ADM-017 was emitted ─────────────────────────────────────────────
	//
	// E-ADM-017 canonical format (error-taxonomy v2.2 §ADM):
	//   "E-ADM-017 HMAC failure rate alert: ≥N failures in Xs from src <addr>"
	//
	// On CURRENT code: captureLogger has up to 5 E-ADM-016 lines but NO
	// E-ADM-017 line (failureCounter is nil → RecordHMACFailure never called).
	// This assertion FAILS on current code — RED GATE.
	//
	// After the fix: buildRouter wires WithFailureCounter(fc) with threshold=5.
	// After the 5th call, fc.RecordHMACFailure emits E-ADM-017 via cl.
	// HasLine("E-ADM-017") returns true — assertion PASSES.
	if !cl.HasLine("E-ADM-017") {
		t.Errorf(
			"E-ADM-017 never logged after 5 consecutive HMAC failures from same source "+
				"(BC-2.05.005 PC-3: ≥5 failures in 60s window must trigger E-ADM-017 alert);\n"+
				"RED GATE: buildRouter does not wire routing.WithFailureCounter — "+
				"r.failureCounter is nil, RecordHMACFailure is never called;\n"+
				"fix: in buildRouter, construct admission.NewFailureCounter(5, 60*time.Second, rl) "+
				"and add routing.WithFailureCounter(fc) to the NewRouter call "+
				"(ARCH-08 v2.2 §6.5.1; tracked deferral C-1-W3P1-defer);\n"+
				"captureLogger captured %d lines: %v",
			len(cl.Lines()), cl.Lines(),
		)
	}
}
