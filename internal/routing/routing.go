// Package routing implements SVTN-partitioned frame dispatch with fail-closed
// admission enforcement (BC-2.05.002, BC-2.05.006).
//
// Classification (ARCH-09 v1.1): boundary — holds forwarding table and admitted
// node map (mutable under mutex); routing decisions are pure but the forwarding
// table is mutable state.
//
// Import constraints (ARCH-08 §6): this package MAY import internal/frame,
// internal/hmac, and internal/admission only. No upward imports.
package routing

import (
	"errors"
	"fmt"
	"sync"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/hmac"
)

// Logger is a minimal logging interface injected into Router.
// BC-2.05.008 PC-2 requires E-ADM-016 to be logged at the router before
// RouteFrame returns on every HMAC-failure path. Callers supply a real
// logger; tests inject a fake that captures log lines for assertion.
type Logger interface {
	// Log records a single log line.
	Log(msg string)
}

// RouterOption is a functional option for NewRouter.
type RouterOption func(*Router)

// WithLogger sets the logger used by the Router. If not set, the Router
// uses nopLogger (log events are silently discarded). Tests inject a fake
// logger to assert mandatory E-ADM-016 emissions per BC-2.05.008.
func WithLogger(l Logger) RouterOption {
	return func(r *Router) {
		r.logger = l
	}
}

// nopLogger is the default logger. Log events are silently discarded.
// Production callers that want operator-visible log records should inject
// a real logger via WithLogger.
type nopLogger struct{}

func (nopLogger) Log(string) {}

// ErrNoForwardingEntry is returned by SVTNRoute when no forwarding-table
// entry exists for (svtnID, dstAddr). Maps to E-FWD-002 in the error
// taxonomy. Distinct from admission.ErrNotAdmitted (E-ADM-003, source
// admission failure) — callers use errors.Is to distinguish a
// forwarding-table miss from an admission rejection.
var ErrNoForwardingEntry = errors.New("routing: no forwarding entry for destination in this SVTN")

// ErrHMACVerificationFailed is returned by RouteFrame when the frame's
// wire HMAC tag does not match the expected MAC for the source node's
// FrameAuthKey, or when no forwarding-table entry exists for the source
// (auth key unavailable → frame is unverifiable → dropped fail-closed).
//
// Maps to E-ADM-016 in the error taxonomy (wire HMAC verification failed
// at RouteFrame: tag mismatch). Distinct from admission.ErrNotAdmitted
// (E-ADM-003, admitted-set rejection) — callers use errors.Is to
// distinguish forgery rejection from admission rejection.
//
// Traces to BC-2.05.008 postconditions 2 and 4; ADR-009.
var ErrHMACVerificationFailed = errors.New("routing: HMAC verification failed (E-ADM-016)")

// ForwardingEntry records a forwarding table entry for one destination node.
type ForwardingEntry struct {
	// NodeAddr is the 8-byte destination node address.
	NodeAddr [8]byte
	// SVTNID is the SVTN this node is admitted to.
	SVTNID [16]byte
	// FrameAuthKey is the per-(node, SVTN) HMAC key for this entry
	// (ARCH-04 §HMAC keying; ADR-001 amended).
	FrameAuthKey [hmac.KeySize]byte
}

// Router is the SVTN-partitioned forwarding engine.
//
// It holds:
//   - The admitted key set (borrowed from internal/admission) for fail-closed
//     admission enforcement (BC-2.05.002).
//   - A forwarding table partitioned by (svtnID, dstAddr) for SVTN isolation
//     (BC-2.05.006; ARCH-04 §SVTN Cryptographic Isolation).
//
// All exported methods are safe for concurrent use.
type Router struct {
	mu              sync.RWMutex
	admittedKeySet  *admission.AdmittedKeySet
	forwardingTable map[[16]byte]map[[8]byte]*ForwardingEntry
	logger          Logger
}

// NewRouter returns an empty Router using ks as its admitted key set.
// ks must not be nil; the router does not own ks — the caller retains
// responsibility for key registration. Optional RouterOption values (e.g.
// WithLogger) are applied after construction.
func NewRouter(ks *admission.AdmittedKeySet, opts ...RouterOption) *Router {
	r := &Router{
		admittedKeySet:  ks,
		forwardingTable: make(map[[16]byte]map[[8]byte]*ForwardingEntry),
		logger:          nopLogger{},
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// RegisterForwardingEntry adds or replaces a forwarding table entry for
// (svtnID, nodeAddr). Last-write-wins semantics consistent with ADR-003.
//
// Traces to BC-2.05.006 postcondition 2: the forwarding engine is partitioned
// by SVTN ID — SVTN-A frames are forwarded only to SVTN-A admitted nodes.
func (r *Router) RegisterForwardingEntry(svtnID [16]byte, nodeAddr [8]byte, authKey [hmac.KeySize]byte) {
	entry := &ForwardingEntry{
		NodeAddr:     nodeAddr,
		SVTNID:       svtnID,
		FrameAuthKey: authKey,
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.forwardingTable[svtnID] == nil {
		r.forwardingTable[svtnID] = make(map[[8]byte]*ForwardingEntry)
	}
	r.forwardingTable[svtnID][nodeAddr] = entry
}

// RouteFrame checks the frame's wire HMAC tag first, then checks whether the
// frame's source address is in the admitted set, then dispatches via SVTNRoute.
//
// Ordering (ADR-009; BC-2.05.008 PC-3; VP-058):
//  1. Look up forwarding-table entry for (hdr.SVTNID, hdr.SrcAddr).
//     If absent → no auth key → log E-ADM-016 → return ErrHMACVerificationFailed (fail-closed).
//  2. Call verifyFrameHMAC with entry.FrameAuthKey.
//     If false → log E-ADM-016 → return ErrHMACVerificationFailed.
//  3. Check admitted set (admission.IsAdmitted).
//     If false → return admission.ErrNotAdmitted.
//  4. Dispatch via SVTNRoute.
//
// payload is the raw bytes following the outer header; it is never parsed here
// (R-001; ARCH-04 §Risk Mitigations).
func RouteFrame(hdr frame.OuterHeader, payload []byte, r *Router) error {
	// Step 1: forwarding-table lookup under RLock.
	//
	// Per ADR-009 v1.6: the RLock is held only for this forwarding-table lookup
	// and key copy (steps 1–3 below). FrameAuthKey is a [32]byte value type and
	// is copied into a local variable (authKey) before the lock is released —
	// defensive copy per ADR-009 v1.6 step 3. HMAC verification (step 2) runs
	// lock-free against that local copy. This keeps the critical section small
	// and avoids holding the forwarding-table RLock during CPU-bound HMAC
	// computation. Sequential HMAC-before-admitted ordering is preserved by
	// statement order in this function, not by lock holding (see ADR-009 v1.6
	// §"Ordering specification").
	r.mu.RLock()
	svtnTable, ok := r.forwardingTable[hdr.SVTNID]
	var (
		entry   *ForwardingEntry
		authKey [hmac.KeySize]byte
	)
	if ok {
		entry = svtnTable[hdr.SrcAddr]
		if entry != nil {
			authKey = entry.FrameAuthKey // copy before unlock (ADR-009 v1.6 step 3)
		}
	}
	r.mu.RUnlock()

	if entry == nil {
		// PATH-A: no forwarding-table entry → auth key unavailable → frame unverifiable.
		// Log E-ADM-016 so operators see every dropped frame (BC-2.05.008 PC-4).
		r.logger.Log(fmt.Sprintf(
			"wire HMAC verification failed at RouteFrame: auth key unavailable for SVTN %x from src %x (E-ADM-016)",
			hdr.SVTNID, hdr.SrcAddr,
		))
		return ErrHMACVerificationFailed
	}

	// Step 2: Verify the wire HMAC tag against the local copy of FrameAuthKey.
	// E-ADM-016: wire HMAC verification failed at RouteFrame (BC-2.05.008 PC-2).
	if !verifyFrameHMAC(hdr, payload, authKey) {
		// PATH-B: tag mismatch — log before return per BC-2.05.008 PC-2.
		r.logger.Log(fmt.Sprintf(
			"wire HMAC verification failed at RouteFrame: tag mismatch for SVTN %x from src %x (E-ADM-016)",
			hdr.SVTNID, hdr.SrcAddr,
		))
		return ErrHMACVerificationFailed
	}

	// Step 3: Fail-closed admitted-set check BEFORE any forwarding (BC-2.05.002 postcondition 3).
	if !r.admittedKeySet.IsAdmitted(hdr.SVTNID, hdr.SrcAddr) {
		return admission.ErrNotAdmitted
	}

	// Step 4: Dispatch.
	return SVTNRoute(hdr, payload, r)
}

// SVTNRoute performs SVTN-isolated forwarding for a pre-admitted frame.
//
// Per BC-2.05.006 postcondition 1: hdr.SVTNID scopes the forwarding lookup —
// only nodes admitted to hdr.SVTNID receive the frame. Nodes admitted to a
// different SVTN are never in the forwarding table for hdr.SVTNID.
//
// Per BC-2.05.006 postcondition 4: there is no administrative override that
// routes SVTN-B traffic to SVTN-A admitted nodes.
//
// Returns ErrNoForwardingEntry if hdr.DstAddr is not in the forwarding table
// for hdr.SVTNID. This is semantically distinct from admission.ErrNotAdmitted
// (E-ADM-003), which signals source admission failure; callers use errors.Is
// to distinguish the two conditions.
//
// payload is the raw bytes following the outer header; routers never parse
// the payload (R-001).
func SVTNRoute(hdr frame.OuterHeader, payload []byte, r *Router) error {
	r.mu.RLock()
	svtnTable, ok := r.forwardingTable[hdr.SVTNID]
	var entry *ForwardingEntry
	if ok {
		entry = svtnTable[hdr.DstAddr]
	}
	r.mu.RUnlock()

	if entry == nil {
		// DstAddr not in forwarding table for this SVTN — SVTN isolation enforced.
		return ErrNoForwardingEntry
	}

	_ = payload // payload is forwarded but not parsed here (R-001)
	_ = entry   // entry holds the authKey for wire-layer HMAC; available for future use

	return nil
}

// verifyFrameHMAC verifies a frame's wire HMAC tag against a freshly-computed
// expected MAC.
//
// The wire tag is read from hdr.HMACTag BEFORE the outer header is zero-tagged
// for MAC computation. The expected MAC is computed over the zeroed-tag header
// + payload bytes with authKey, then compared constant-time against the wire
// tag via hmac.VerifyHMAC (which wraps crypto/hmac.Equal).
//
// Returns true if and only if the wire tag exactly matches the expected MAC.
// Fail-closed: returns false on any mismatch or zero-length wire tag.
func verifyFrameHMAC(hdr frame.OuterHeader, payload []byte, authKey [hmac.KeySize]byte) bool {
	// Save the wire tag BEFORE clearing — defends against the tautological-verify
	// defect: clearing first and then "verifying" would check the computed tag
	// against itself and return true unconditionally.
	wireTag := hdr.HMACTag

	hdrForMAC := hdr
	hdrForMAC.HMACTag = [8]byte{}
	encoded := frame.EncodeOuterHeader(hdrForMAC)

	// Concatenate header bytes and payload as the message over which HMAC is computed.
	msg := make([]byte, len(encoded)+len(payload))
	copy(msg, encoded[:])
	copy(msg[len(encoded):], payload)

	// Verify the wire tag (the one the sender computed against the zeroed-tag
	// message and inserted back into the frame) against our recomputed expected MAC.
	return hmac.VerifyHMAC(authKey[:], msg, wireTag)
}
