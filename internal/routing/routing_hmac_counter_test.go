// Package routing_test — AC-009 Red-Gate tests for RouteFrame + FailureCounter
// integration (S-W3.05; BC-2.05.008 PC-5 + invariant 5).
//
// # RED GATE
//
// These tests WILL NOT COMPILE against the current codebase because:
//  1. routing.WithFailureCounter does not yet exist.
//  2. The Router struct has no failureCounter field.
//  3. admission.FailureCounter does not yet exist.
//  4. admission.NewFailureCounter does not yet exist.
//  5. admission.WithNow does not yet exist.
//
// Compile failure is the Red Gate: it proves the wire-up is absent before
// implementation begins.
//
// # Implementer contract (BC-2.05.008 PC-5 + invariant 5)
//
// In internal/routing/routing.go:
//
//  1. Add `failureCounter *admission.FailureCounter` field to the Router struct.
//
//  2. Add functional option:
//
//     func WithFailureCounter(fc *admission.FailureCounter) RouterOption {
//     return func(r *Router) { r.failureCounter = fc }
//     }
//
//  3. In RouteFrame, on BOTH ErrHMACVerificationFailed return paths, call
//     r.failureCounter.RecordHMACFailure(string(hdr.SrcAddr[:])) (or use the
//     canonical string representation of hdr.SrcAddr — must match what the
//     FailureCounter and callers expect; typically fmt.Sprintf("%x", hdr.SrcAddr)
//     or just string(hdr.SrcAddr[:]); be consistent with BC-2.05.005 PC-3).
//
//     Guard with nil-check so existing tests without a counter still pass:
//
//     if r.failureCounter != nil {
//     r.failureCounter.RecordHMACFailure(srcKey)
//     }
//     return ErrHMACVerificationFailed
//
//  4. Do NOT call RecordHMACFailure on the success path (BC-2.05.008 PC-5 negative).
//
// In internal/admission/failure_counter.go (full contract in failure_counter_test.go):
//
//	type Logger interface { Log(msg string) }
//	type FailureCounterOption func(*FailureCounter)
//	func WithNow(fn func() time.Time) FailureCounterOption
//	func NewFailureCounter(threshold int, windowDuration time.Duration, logger Logger, opts ...FailureCounterOption) *FailureCounter
//	func (c *FailureCounter) RecordHMACFailure(srcAddr string)
//	func (c *FailureCounter) Timestamps(srcAddr string) []time.Time  // value copy
//
// # srcAddr key convention
//
// For FailureCounter, the srcAddr passed by RouteFrame is the string representation
// of hdr.SrcAddr. Use fmt.Sprintf("%x", hdr.SrcAddr) for a human-readable key
// that matches the E-ADM-017 log message format. Tests in this file use the same
// format for asserting call counts.
//
// Traces to: BC-2.05.008 PC-5; BC-2.05.008 invariant 5; S-W3.05 AC-009;
// BC-2.05.005 canonical test vector (5 failures → E-ADM-017).
package routing_test

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/hmac"
	"github.com/arcavenae/switchboard/internal/routing"
)

// ── fakeFailureCounter ────────────────────────────────────────────────────────

// fakeFailureCounter counts RecordHMACFailure calls per srcAddr.
// Used to verify routing-layer wire-up independently of the real FailureCounter's
// sliding-window logic (which is tested in failure_counter_test.go).
//
// Concurrency-safe.
type fakeFailureCounter struct {
	mu    sync.Mutex
	calls map[string]int
}

func newFakeFailureCounter() *fakeFailureCounter {
	return &fakeFailureCounter{calls: make(map[string]int)}
}

func (f *fakeFailureCounter) RecordHMACFailure(srcAddr string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls[srcAddr]++
}

func (f *fakeFailureCounter) CallCount(srcAddr string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[srcAddr]
}

func (f *fakeFailureCounter) TotalCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	total := 0
	for _, c := range f.calls {
		total += c
	}
	return total
}

// ── fakeAlertLog implements admission.Logger ──────────────────────────────────

// fakeAlertLog captures log lines for the real FailureCounter in integration tests.
// It satisfies admission.Logger (same single-method interface as routing.Logger).
type fakeAlertLog struct {
	mu    sync.Mutex
	lines []string
}

func (l *fakeAlertLog) Log(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lines = append(l.lines, msg)
}

func (l *fakeAlertLog) Count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.lines)
}

func (l *fakeAlertLog) Lines() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]string, len(l.lines))
	copy(out, l.lines)
	return out
}

func (l *fakeAlertLog) HasAll(substrs ...string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, line := range l.lines {
		allFound := true
		for _, s := range substrs {
			if !strings.Contains(line, s) {
				allFound = false
				break
			}
		}
		if allFound {
			return true
		}
	}
	return false
}

// ── buildAdmittedKS ───────────────────────────────────────────────────────────

// buildAdmittedKS constructs an AdmittedKeySet with one fully admitted node,
// returning the key set, srcAddr, and authKey.
//
// This mirrors admittedRouterSetup (in routing_test.go) but returns the ks
// separately so callers can build a Router with additional options
// (e.g., WithFailureCounter).
func buildAdmittedKS(
	t *testing.T,
	svtnID [16]byte,
	nodeSeedByte byte,
	routerSeedByte byte,
) (ks *admission.AdmittedKeySet, srcAddr [8]byte, authKey [hmac.KeySize]byte) {
	t.Helper()

	var nodeSeed [32]byte
	nodeSeed[0] = nodeSeedByte
	copy(nodeSeed[1:], "ac009-node-seed-bytes-filler-xx")

	var routerSeed [32]byte
	routerSeed[0] = routerSeedByte
	copy(routerSeed[1:], "ac009-rtr--seed-bytes-filler-xx")

	nodePub, nodePriv := seedKeyDet(t, nodeSeed)
	_, routerPriv := seedKeyDet(t, routerSeed)

	ks = admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)

	// Derive node address using the same helper as other test files.
	srcAddr = nodeAddrForTest(svtnID, nodePub)

	// Complete challenge-response.
	challenge, err := admission.GenerateChallenge(routerPriv)
	if err != nil {
		t.Fatalf("buildAdmittedKS GenerateChallenge: %v", err)
	}
	nonceSig := ed25519.Sign(nodePriv, challenge.Nonce[:])
	resp := admission.ChallengeResponse{NonceSig: nonceSig}
	if err := admission.AdmitNode(challenge, resp, nodePub, svtnID, ks); err != nil {
		t.Fatalf("buildAdmittedKS AdmitNode: %v", err)
	}

	authKey = hmac.DeriveKey([]byte(nodePub), svtnID)
	return ks, srcAddr, authKey
}

// ── AC-009: TestRouteFrame_CallsRecordHMACFailureOnBothFailurePaths ───────────

// TestRouteFrame_CallsRecordHMACFailureOnBothFailurePaths verifies that
// RouteFrame calls router.failureCounter.RecordHMACFailure(srcAddr) on BOTH
// ErrHMACVerificationFailed paths (PATH-A and PATH-B), and NOT on success.
//
// The test injects a fakeFailureCounter (not the real counter) via
// routing.WithFailureCounter to verify the routing-layer wire-up independently.
//
// Traces to BC-2.05.008 PC-5; BC-2.05.008 invariant 5; S-W3.05 AC-009.
func TestRouteFrame_CallsRecordHMACFailureOnBothFailurePaths(t *testing.T) {
	t.Parallel()

	// ── PATH-A: no forwarding entry for src ──────────────────────────────────
	t.Run("PATH-A no forwarding entry calls RecordHMACFailure", func(t *testing.T) {
		t.Parallel()

		var svtnID [16]byte
		copy(svtnID[:], "ac009-patha-svtn")

		ks := admission.NewAdmittedKeySet()
		fc := newFakeFailureCounter()
		capLog := &routingFakeLog{} // from routing_log_test.go (same package)

		// Router with no forwarding entries.
		r := routing.NewRouter(ks,
			routing.WithLogger(capLog),
			routing.WithFailureCounter(fc), // IMPLEMENTER MUST ADD THIS OPTION
		)

		var srcAddr [8]byte
		copy(srcAddr[:], "pathAsrc")
		srcKey := fmt.Sprintf("%x", srcAddr) // canonical key format

		hdr := frame.OuterHeader{
			Version:   frame.VersionByte,
			FrameType: frame.FrameTypeData,
			SVTNID:    svtnID,
			SrcAddr:   srcAddr,
		}

		err := routing.RouteFrame(hdr, nil, r)

		if !errors.Is(err, routing.ErrHMACVerificationFailed) {
			t.Errorf("PATH-A: want ErrHMACVerificationFailed, got %v", err)
		}
		// RecordHMACFailure must be called exactly once.
		if got := fc.CallCount(srcKey); got != 1 {
			t.Errorf("PATH-A: want RecordHMACFailure called 1 time for src %q, got %d "+
				"(BC-2.05.008 invariant 5: must be called on ALL ErrHMACVerificationFailed paths)",
				srcKey, got)
		}
	})

	// ── PATH-B: forwarding entry present, tag mismatch ───────────────────────
	t.Run("PATH-B tag mismatch calls RecordHMACFailure", func(t *testing.T) {
		t.Parallel()

		var svtnID [16]byte
		copy(svtnID[:], "ac009-pathb-svtn")

		fc := newFakeFailureCounter()
		capLog := &routingFakeLog{}

		ks, srcAddr, authKey := buildAdmittedKS(t, svtnID, 0xC2, 0xD2)
		srcKey := fmt.Sprintf("%x", srcAddr)

		var dstAddr [8]byte
		copy(dstAddr[:], "pathBdst0")

		r := routing.NewRouter(ks,
			routing.WithLogger(capLog),
			routing.WithFailureCounter(fc),
		)
		r.RegisterForwardingEntry(svtnID, srcAddr, authKey)
		r.RegisterForwardingEntry(svtnID, dstAddr, [hmac.KeySize]byte{0xAB})

		// Tag computed under wrong key — verifyFrameHMAC returns false.
		var wrongKey [hmac.KeySize]byte
		copy(wrongKey[:], "wrong-key-for-path-b-test-00000")
		payload := []byte("path-b-payload")
		hdr := frame.OuterHeader{
			Version:   frame.VersionByte,
			FrameType: frame.FrameTypeData,
			SVTNID:    svtnID,
			SrcAddr:   srcAddr,
			DstAddr:   dstAddr,
		}
		hdr.HMACTag = computeValidTag(hdr, payload, wrongKey)

		err := routing.RouteFrame(hdr, payload, r)

		if !errors.Is(err, routing.ErrHMACVerificationFailed) {
			t.Errorf("PATH-B: want ErrHMACVerificationFailed, got %v", err)
		}
		if got := fc.CallCount(srcKey); got != 1 {
			t.Errorf("PATH-B: want RecordHMACFailure called 1 time for src %q, got %d "+
				"(BC-2.05.008 PC-5: counter must be called before return on tag-mismatch path)",
				srcKey, got)
		}
	})

	// ── SUCCESS: valid HMAC — RecordHMACFailure must NOT be called ────────────
	t.Run("SUCCESS valid HMAC does NOT call RecordHMACFailure", func(t *testing.T) {
		t.Parallel()

		var svtnID [16]byte
		copy(svtnID[:], "ac009-succ-svtn0")

		fc := newFakeFailureCounter()
		capLog := &routingFakeLog{}

		ks, srcAddr, authKey := buildAdmittedKS(t, svtnID, 0xC3, 0xD3)

		var dstAddr [8]byte
		copy(dstAddr[:], "successdst")

		r := routing.NewRouter(ks,
			routing.WithLogger(capLog),
			routing.WithFailureCounter(fc),
		)
		r.RegisterForwardingEntry(svtnID, srcAddr, authKey)
		r.RegisterForwardingEntry(svtnID, dstAddr, [hmac.KeySize]byte{0xCD})

		payload := []byte("success-payload")
		hdr := frame.OuterHeader{
			Version:   frame.VersionByte,
			FrameType: frame.FrameTypeData,
			SVTNID:    svtnID,
			SrcAddr:   srcAddr,
			DstAddr:   dstAddr,
		}
		hdr.HMACTag = computeValidTag(hdr, payload, authKey) // valid tag

		// HMAC passes. The node IS admitted (buildAdmittedKS runs challenge-response).
		err := routing.RouteFrame(hdr, payload, r)
		if err != nil {
			t.Logf("SUCCESS: RouteFrame returned %v (non-nil is OK here if HMAC passed)", err)
		}
		// What matters: HMAC verification succeeded, so RecordHMACFailure must NOT fire.
		if errors.Is(err, routing.ErrHMACVerificationFailed) {
			t.Errorf("SUCCESS: valid HMAC should not return ErrHMACVerificationFailed")
		}
		if fc.TotalCalls() != 0 {
			t.Errorf("SUCCESS: RecordHMACFailure must NOT be called on successful HMAC verification "+
				"(BC-2.05.008 PC-5 negative assertion); got %d calls", fc.TotalCalls())
		}
	})
}

// ── BC-2.05.008 EC-006: 5 consecutive RouteFrame failures → E-ADM-017 ────────

// TestRouteFrame_FiveConsecutiveFailures_TriggersEADM017 verifies the full
// end-to-end integration: 5 RouteFrame calls that each return
// ErrHMACVerificationFailed result in the REAL admission.FailureCounter
// emitting E-ADM-017 via the injected logger.
//
// Canonical BC-2.05.008 EC-006 test vector:
// "5 consecutive HMAC failures from same src_addr within 60s →
//
//	after 5th: E-ADM-017 emitted by FailureCounter; RouteFrame called
//	RecordHMACFailure 5 times."
//
// Uses the REAL admission.FailureCounter with admission.WithNow for clock control.
// Requires both admission.NewFailureCounter and routing.WithFailureCounter.
//
// Traces to BC-2.05.008 EC-006; S-W3.05 AC-009 integration path; VP-059.
func TestRouteFrame_FiveConsecutiveFailures_TriggersEADM017(t *testing.T) {
	t.Parallel()

	var svtnID [16]byte
	copy(svtnID[:], "ec006-int-svtn00")

	alertLog := &fakeAlertLog{}

	// Real FailureCounter, threshold=5, window=60s, clock seam injected.
	base := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	current := base

	fc := admission.NewFailureCounter(5, 60*time.Second, alertLog,
		admission.WithNow(func() time.Time { return current }), // IMPLEMENTER MUST ADD THIS OPTION
	)

	ks, srcAddr, authKey := buildAdmittedKS(t, svtnID, 0xE1, 0xF1)

	var dstAddr [8]byte
	copy(dstAddr[:], "ec006dst0")

	routeLog := &routingFakeLog{}
	r := routing.NewRouter(ks,
		routing.WithLogger(routeLog),
		routing.WithFailureCounter(fc),
	)
	r.RegisterForwardingEntry(svtnID, srcAddr, authKey)
	r.RegisterForwardingEntry(svtnID, dstAddr, [hmac.KeySize]byte{0x99})

	// Frame with bad HMAC — every call returns ErrHMACVerificationFailed.
	var wrongKey [hmac.KeySize]byte
	copy(wrongKey[:], "ec006-wrong-key-for-5-failures0")
	payload := []byte("ec006-payload")
	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtnID,
		SrcAddr:   srcAddr,
		DstAddr:   dstAddr,
	}
	hdr.HMACTag = computeValidTag(hdr, payload, wrongKey)

	// 5 consecutive failures, each within the 60s window.
	for i := range 5 {
		current = base.Add(time.Duration(i) * time.Second)
		err := routing.RouteFrame(hdr, payload, r)
		if !errors.Is(err, routing.ErrHMACVerificationFailed) {
			t.Fatalf("EC-006: call %d: want ErrHMACVerificationFailed, got %v", i+1, err)
		}
	}

	// After 5 failures: E-ADM-017 must have fired exactly once.
	if alertLog.Count() != 1 {
		t.Errorf("EC-006: want exactly 1 E-ADM-017 alert after 5 consecutive RouteFrame failures, "+
			"got %d; lines: %v", alertLog.Count(), alertLog.Lines())
	}
	if !alertLog.HasAll("E-ADM-017") {
		t.Errorf("EC-006: alert log missing \"E-ADM-017\"; lines: %v", alertLog.Lines())
	}
}
