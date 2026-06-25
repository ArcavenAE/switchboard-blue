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
	"sync"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/hmac"
)

// ErrNoForwardingEntry is returned by SVTNRoute when the destination
// address has no forwarding-table entry for the SVTN. Distinct from
// admission.ErrNotAdmitted (which signals source admission failure)
// to enable callers to distinguish admission rejection from forwarding-
// table miss via errors.Is.
var ErrNoForwardingEntry = errors.New("routing: no forwarding entry for destination in this SVTN")

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
}

// NewRouter returns an empty Router using ks as its admitted key set.
// ks must not be nil; the router does not own ks — the caller retains
// responsibility for key registration.
func NewRouter(ks *admission.AdmittedKeySet) *Router {
	return &Router{
		admittedKeySet:  ks,
		forwardingTable: make(map[[16]byte]map[[8]byte]*ForwardingEntry),
	}
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

// RouteFrame checks whether the frame's source address is in the admitted set
// for the frame's SVTN, then dispatches it via SVTNRoute.
//
// Fail-closed (BC-2.05.002 invariant 2): if hdr.SrcAddr is NOT in the admitted
// set for hdr.SVTNID, the frame is dropped and admission.ErrNotAdmitted is
// returned (E-ADM-003). No frame is ever forwarded before this check.
//
// Per BC-2.05.002 postcondition 3: the admitted-set check happens BEFORE any
// forwarding logic executes.
//
// payload is the raw bytes following the outer header; it is never parsed here
// (R-001; ARCH-04 §Risk Mitigations).
func RouteFrame(hdr frame.OuterHeader, payload []byte, r *Router) error {
	// Fail-closed: admitted-set check BEFORE any forwarding (BC-2.05.002 postcondition 3).
	if !r.admittedKeySet.IsAdmitted(hdr.SVTNID, hdr.SrcAddr) {
		return admission.ErrNotAdmitted
	}

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
//
//nolint:unused // wired into RouteFrame in the next wave when wire-layer HMAC enforcement is added (BC-2.05.002 invariant 1)
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
