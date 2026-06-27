// Package routing_test — E-ADM-016 logging tests (BC-2.05.008 PC-2, PC-4).
//
// # RED GATE: these tests WILL NOT COMPILE against current code.
//
// The routing package does not yet define Logger, RouterOption, or
// WithLogger. These tests assert the MINIMUM API surface the implementer must
// add. They are intentionally written against that absent API so the compiler
// itself proves the Red Gate (build failure = all tests fail before
// implementation begins).
//
// # Implementer contract
//
// Add the following to internal/routing/routing.go (mirror the
// tmux.Logger / tmux.WithLogger pattern from internal/tmux/pty_fallback.go):
//
//	// Logger is a minimal logging interface injected into Router.
//	// BC-2.05.008 PC-2 requires E-ADM-016 to be logged at the router
//	// before RouteFrame returns on every HMAC-failure path. Tests inject
//	// a fake logger to capture and assert the required log content.
//	type Logger interface {
//	    Log(msg string)
//	}
//
//	// RouterOption is a functional option for NewRouter.
//	type RouterOption func(*Router)
//
//	// WithLogger sets the logger used by the Router. If not set, the Router
//	// uses a nop logger (log events are silently discarded). Tests inject a
//	// fakeLogCapture to assert mandatory E-ADM-016 emissions per BC-2.05.008.
//	func WithLogger(l Logger) RouterOption {
//	    return func(r *Router) {
//	        r.logger = l
//	    }
//	}
//
// NewRouter must accept variadic RouterOption:
//
//	func NewRouter(ks *admission.AdmittedKeySet, opts ...RouterOption) *Router
//
// # E-ADM-016 canonical log message (error-taxonomy.md §ADM)
//
// Every HMAC-failure return path in RouteFrame MUST emit a log record whose
// string contains ALL of the following:
//
//  1. The literal string "E-ADM-016"
//  2. The literal string "wire HMAC verification failed at RouteFrame"
//  3. The hex representation of hdr.SVTNID  (field name: svtn_id)
//  4. The hex representation of hdr.SrcAddr (field name: src_addr)
//
// Example conforming message (from error-taxonomy.md):
//
//	"wire HMAC verification failed at RouteFrame: tag mismatch for SVTN
//	 <svtn_id> from src <src_addr> (E-ADM-016)"
//
// # RouteFrame paths covered
//
// RouteFrame has two paths that return ErrHMACVerificationFailed:
//
//	PATH-A (~routing.go:138-140): entry == nil — no forwarding-table entry;
//	auth key unavailable; frame is unverifiable. BC-2.05.008 PC-4.
//	Logging obligation: these tests assert E-ADM-016 IS logged here too,
//	because operators must see every dropped frame regardless of path.
//	If the spec-steward rules PC-4 does not require a log, the companion
//	test for PATH-A should be revised to assert NO log.
//
//	PATH-B (~routing.go:144-146): verifyFrameHMAC returns false — tag mismatch.
//	BC-2.05.008 PC-2 explicitly mandates E-ADM-016 here.
//
// Traces to: BC-2.05.008 PC-2, PC-4; error-taxonomy.md E-ADM-016; VP-058.
package routing_test

import (
	"crypto/ed25519"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/hmac"
	"github.com/arcavenae/switchboard/internal/routing"
)

// -- fakeLogCapture -----------------------------------------------------------

// fakeLogCapture implements routing.Logger and captures log lines for
// assertion. Mirrors the fakeLogCapture pattern from
// internal/tmux/pty_fallback_test.go.
// Concurrency: safe (mu protects lines).
type routingFakeLog struct {
	mu    sync.Mutex
	lines []string
}

func (f *routingFakeLog) Log(msg string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lines = append(f.lines, msg)
}

// Count returns the number of captured log lines.
func (f *routingFakeLog) Count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.lines)
}

// Lines returns a snapshot of all captured log lines (value copy).
func (f *routingFakeLog) Lines() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.lines))
	copy(out, f.lines)
	return out
}

// HasAll reports whether any captured log line contains ALL of the given
// substrings. Case-insensitive on the log line (lowercased); substrs are
// matched as-is (caller may lower them if needed).
func (f *routingFakeLog) HasAll(substrs ...string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, line := range f.lines {
		ll := strings.ToLower(line)
		allFound := true
		for _, s := range substrs {
			if !strings.Contains(ll, strings.ToLower(s)) {
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

// -- hex helpers --------------------------------------------------------------

func svtnHex(id [16]byte) string  { return fmt.Sprintf("%x", id) }
func addrHex(addr [8]byte) string { return fmt.Sprintf("%x", addr) }

// -- loggedRouterSetup --------------------------------------------------------

// loggedRouterSetup creates a Router injected with capLog for a fully-admitted
// node, mirrors admittedRouterSetup from routing_test.go but builds the Router
// via routing.NewRouter(ks, routing.WithLogger(capLog)) so the logger is wired.
//
// Returns: the logger-injected Router, srcAddr, and authKey for the admitted
// node. The caller can register additional forwarding entries as needed.
func loggedRouterSetup(
	t *testing.T,
	svtnID [16]byte,
	nodeSeedByte byte,
	routerSeedByte byte,
	capLog routing.Logger,
) (r *routing.Router, srcAddr [8]byte, authKey [hmac.KeySize]byte) {
	t.Helper()

	var nodeSeed [32]byte
	nodeSeed[0] = nodeSeedByte
	copy(nodeSeed[1:], "logtest-node-seed-filler-bytes00")
	nodePub, nodePriv := seedKeyDet(t, nodeSeed)

	var routerSeed [32]byte
	routerSeed[0] = routerSeedByte
	copy(routerSeed[1:], "logtest-rtr--seed-filler-bytes00")
	_, routerPriv := seedKeyDet(t, routerSeed)

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)

	// Derive node address: SHA-256(svtnID || pubkey)[:8].
	h := sha256.New()
	h.Write(svtnID[:])
	h.Write([]byte(nodePub))
	sum := h.Sum(nil)
	copy(srcAddr[:], sum[:8])

	// Complete challenge-response so IsAdmitted returns true.
	challenge, err := admission.GenerateChallenge(routerPriv)
	if err != nil {
		t.Fatalf("loggedRouterSetup GenerateChallenge: %v", err)
	}
	nonceSig := ed25519.Sign(nodePriv, challenge.Nonce[:])
	resp := admission.ChallengeResponse{NonceSig: nonceSig}
	if err := admission.AdmitNode(challenge, resp, nodePub, svtnID, ks); err != nil {
		t.Fatalf("loggedRouterSetup AdmitNode: %v", err)
	}

	authKey = hmac.DeriveKey([]byte(nodePub), svtnID)

	// Build the Router with the injected logger — the API under test.
	r = routing.NewRouter(ks, routing.WithLogger(capLog))
	r.RegisterForwardingEntry(svtnID, srcAddr, authKey)

	return r, srcAddr, authKey
}

// -- BC-2.05.008 PC-2: HMAC-verify-fail (PATH-B) logs E-ADM-016 ──────────────

// test_BC_2_05_008_hmac_verify_fail_logs_eadm016 verifies that RouteFrame
// emits exactly one log record carrying E-ADM-016 and the required fields when
// verifyFrameHMAC returns false (tag computed under wrong key → tag mismatch).
//
// Assertions:
//
//	(a) returned error is ErrHMACVerificationFailed  (regression guard)
//	(b) exactly 1 log record was emitted
//	(c) log record contains "E-ADM-016"
//	(d) log record contains "wire HMAC verification failed at RouteFrame"
//	(e) log record contains svtn_id (hex)
//	(f) log record contains src_addr (hex)
//
// Mutation resistance: fails if implementer (a) forgets to log, (b) logs the
// wrong code, or (c) omits svtn_id or src_addr.
//
// Traces to BC-2.05.008 PC-2; error-taxonomy.md E-ADM-016.
func Test_BC_2_05_008_hmac_verify_fail_logs_eadm016(t *testing.T) {
	t.Parallel()

	var svtnID [16]byte
	copy(svtnID[:], "log-test-svtn-pc2")

	capLog := &routingFakeLog{}
	r, srcAddr, authKey := loggedRouterSetup(t, svtnID, 0xA1, 0xB1, capLog)

	// Destination entry.
	var dstAddr [8]byte
	copy(dstAddr[:], "dstlogpc2")
	r.RegisterForwardingEntry(svtnID, dstAddr, [hmac.KeySize]byte{0xAB})

	// Tag computed under a WRONG key — verifyFrameHMAC returns false.
	var wrongKey [hmac.KeySize]byte
	copy(wrongKey[:], "wrong-key-forces-hmac-mismatch00")
	payload := []byte("pc2-log-test-payload")
	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtnID,
		SrcAddr:   srcAddr,
		DstAddr:   dstAddr,
	}
	hdr.HMACTag = computeValidTag(hdr, payload, wrongKey)
	_ = authKey // authKey is the real key; wrongKey causes the mismatch

	err := routing.RouteFrame(hdr, payload, r)

	// (a)
	if !errors.Is(err, routing.ErrHMACVerificationFailed) {
		t.Errorf("BC-2.05.008 PC-2: want ErrHMACVerificationFailed, got %v", err)
	}
	// (b)
	if capLog.Count() != 1 {
		t.Errorf("BC-2.05.008 PC-2: want exactly 1 log record, got %d; lines: %v",
			capLog.Count(), capLog.Lines())
	}
	// (c)
	if !capLog.HasAll("E-ADM-016") {
		t.Errorf("BC-2.05.008 PC-2: log record missing \"E-ADM-016\"; lines: %v", capLog.Lines())
	}
	// (d)
	if !capLog.HasAll("wire HMAC verification failed at RouteFrame") {
		t.Errorf("BC-2.05.008 PC-2: log record missing canonical message prefix; lines: %v", capLog.Lines())
	}
	// (e)
	if !capLog.HasAll(svtnHex(svtnID)) {
		t.Errorf("BC-2.05.008 PC-2: log record missing svtn_id %q; lines: %v", svtnHex(svtnID), capLog.Lines())
	}
	// (f)
	if !capLog.HasAll(addrHex(srcAddr)) {
		t.Errorf("BC-2.05.008 PC-2: log record missing src_addr %q; lines: %v", addrHex(srcAddr), capLog.Lines())
	}
}

// test_BC_2_05_008_zero_tag_logs_eadm016 verifies E-ADM-016 logging for the
// all-zero HMACTag edge case (EC-001 from BC-2.05.008 test-vector table).
// Same PATH-B assertions as test_BC_2_05_008_hmac_verify_fail_logs_eadm016.
//
// Traces to BC-2.05.008 EC-001; error-taxonomy.md E-ADM-016.
func Test_BC_2_05_008_zero_tag_logs_eadm016(t *testing.T) {
	t.Parallel()

	var svtnID [16]byte
	copy(svtnID[:], "log-test-svtn-ec1")

	capLog := &routingFakeLog{}
	r, srcAddr, _ := loggedRouterSetup(t, svtnID, 0xA2, 0xB2, capLog)

	var dstAddr [8]byte
	copy(dstAddr[:], "dstlogec10")
	r.RegisterForwardingEntry(svtnID, dstAddr, [hmac.KeySize]byte{0xCD})

	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtnID,
		SrcAddr:   srcAddr,
		DstAddr:   dstAddr,
		// HMACTag is zero-value ([8]byte{}) — EC-001.
	}

	err := routing.RouteFrame(hdr, []byte("ec1-log-payload"), r)

	if !errors.Is(err, routing.ErrHMACVerificationFailed) {
		t.Errorf("BC-2.05.008 EC-001: want ErrHMACVerificationFailed, got %v", err)
	}
	if capLog.Count() != 1 {
		t.Errorf("BC-2.05.008 EC-001: want 1 log record, got %d; lines: %v", capLog.Count(), capLog.Lines())
	}
	if !capLog.HasAll("E-ADM-016") {
		t.Errorf("BC-2.05.008 EC-001: missing \"E-ADM-016\"; lines: %v", capLog.Lines())
	}
	if !capLog.HasAll("wire HMAC verification failed at RouteFrame") {
		t.Errorf("BC-2.05.008 EC-001: missing canonical message; lines: %v", capLog.Lines())
	}
	if !capLog.HasAll(svtnHex(svtnID)) {
		t.Errorf("BC-2.05.008 EC-001: missing svtn_id %q; lines: %v", svtnHex(svtnID), capLog.Lines())
	}
	if !capLog.HasAll(addrHex(srcAddr)) {
		t.Errorf("BC-2.05.008 EC-001: missing src_addr %q; lines: %v", addrHex(srcAddr), capLog.Lines())
	}
}

// -- BC-2.05.008 PC-4: no-forwarding-entry path (PATH-A) logs E-ADM-016 ──────

// test_BC_2_05_008_no_entry_path_logs_eadm016 verifies that RouteFrame emits
// E-ADM-016 when there is no forwarding-table entry for (svtnID, srcAddr) —
// PATH-A, the auth-key-unavailable / unverifiable-frame path.
//
// If the spec-steward determines that PC-4 does NOT require logging, this test
// should be revised to assert capLog.Count() == 0. Until then, the conservative
// position is that every dropped frame must produce an operator-visible log record.
//
// Traces to BC-2.05.008 PC-4; error-taxonomy.md E-ADM-016.
func Test_BC_2_05_008_no_entry_path_logs_eadm016(t *testing.T) {
	t.Parallel()

	var svtnID [16]byte
	copy(svtnID[:], "log-test-svtn-pc4")

	// Empty admitted-key set — no nodes registered, no forwarding entries.
	ks := admission.NewAdmittedKeySet()

	capLog := &routingFakeLog{}
	// Build the Router with the injected logger — no forwarding entries registered.
	r := routing.NewRouter(ks, routing.WithLogger(capLog))

	var srcAddr [8]byte
	copy(srcAddr[:], "unknowns") // 8 bytes — fits [8]byte exactly

	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtnID,
		SrcAddr:   srcAddr,
	}

	err := routing.RouteFrame(hdr, nil, r)

	// (a) regression guard
	if !errors.Is(err, routing.ErrHMACVerificationFailed) {
		t.Errorf("BC-2.05.008 PC-4: want ErrHMACVerificationFailed, got %v", err)
	}
	// (b) exactly one log record
	if capLog.Count() != 1 {
		t.Errorf("BC-2.05.008 PC-4: want 1 log record, got %d; lines: %v", capLog.Count(), capLog.Lines())
	}
	// (c)
	if !capLog.HasAll("E-ADM-016") {
		t.Errorf("BC-2.05.008 PC-4: missing \"E-ADM-016\"; lines: %v", capLog.Lines())
	}
	// (d)
	if !capLog.HasAll("wire HMAC verification failed at RouteFrame") {
		t.Errorf("BC-2.05.008 PC-4: missing canonical message; lines: %v", capLog.Lines())
	}
	// (e)
	if !capLog.HasAll(svtnHex(svtnID)) {
		t.Errorf("BC-2.05.008 PC-4: missing svtn_id %q; lines: %v", svtnHex(svtnID), capLog.Lines())
	}
	// (f)
	if !capLog.HasAll(addrHex(srcAddr)) {
		t.Errorf("BC-2.05.008 PC-4: missing src_addr %q; lines: %v", addrHex(srcAddr), capLog.Lines())
	}
}

// -- Mutation resistance: no spurious log on success path ─────────────────────

// test_BC_2_05_008_no_log_on_hmac_success verifies that E-ADM-016 is NOT
// logged when HMAC verification passes. A naive implementation that always
// logs would fail this test. Traces to BC-2.05.008 PC-1.
func Test_BC_2_05_008_no_log_on_hmac_success(t *testing.T) {
	t.Parallel()

	var svtnID [16]byte
	copy(svtnID[:], "log-test-svtn-ok0")

	capLog := &routingFakeLog{}
	r, srcAddr, authKey := loggedRouterSetup(t, svtnID, 0xA3, 0xB3, capLog)

	var dstAddr [8]byte
	copy(dstAddr[:], "dstlogok00")
	r.RegisterForwardingEntry(svtnID, dstAddr, [hmac.KeySize]byte{0xEF})

	payload := []byte("success-path-payload")
	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtnID,
		SrcAddr:   srcAddr,
		DstAddr:   dstAddr,
	}
	hdr.HMACTag = computeValidTag(hdr, payload, authKey) // valid tag — HMAC passes

	_ = routing.RouteFrame(hdr, payload, r)

	if capLog.HasAll("E-ADM-016") {
		t.Errorf("BC-2.05.008 PC-1: E-ADM-016 spuriously logged on success path; lines: %v",
			capLog.Lines())
	}
}
