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
	"sync"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/hmac"
)

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
	mu              sync.RWMutex                              //nolint:unused // implementer wires in all methods
	admittedKeySet  *admission.AdmittedKeySet                 //nolint:unused // implementer wires in RouteFrame, SVTNRoute
	forwardingTable map[[16]byte]map[[8]byte]*ForwardingEntry //nolint:unused // implementer wires in RouteFrame, SVTNRoute
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
	panic("not implemented: S-2.02 Router.RegisterForwardingEntry")
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
	panic("not implemented: S-2.02 RouteFrame")
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
// Returns admission.ErrNotAdmitted (E-ADM-003) if hdr.DstAddr is not in the
// forwarding table for hdr.SVTNID (split-horizon defense-in-depth).
//
// payload is the raw bytes following the outer header; routers never parse
// the payload (R-001).
func SVTNRoute(hdr frame.OuterHeader, payload []byte, r *Router) error {
	panic("not implemented: S-2.02 SVTNRoute")
}

// splitHorizon reports whether hdr.DstAddr should be excluded from forwarding
// on the arrival interface arrivalNodeAddr (BC-2.02.008 / E-FWD-001 split-
// horizon stub). Unexported — wired into SVTNRoute by the implementer.
func splitHorizon(hdr frame.OuterHeader, arrivalNodeAddr [8]byte) bool { //nolint:unused // implementer wires in SVTNRoute
	panic("not implemented: S-2.02 splitHorizon")
}

// verifyFrameHMAC checks the HMAC tag on hdr+payload against the per-(node,
// SVTN) frame_auth_key stored in the forwarding table.
//
// Fail-closed: returns false on any verification failure including missing
// forwarding entry (BC-2.05.002 invariant 1).
func verifyFrameHMAC(hdr frame.OuterHeader, payload []byte, authKey [hmac.KeySize]byte) bool { //nolint:unused // implementer wires in RouteFrame
	panic("not implemented: S-2.02 verifyFrameHMAC")
}
