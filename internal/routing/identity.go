// Package routing — identity binding: (SVTNID, NodeAddr) → InterfaceID.
//
// After a successful NODE_IDENTIFY handshake (BC-2.01.009), the router
// records the mapping (svtnID, nodeAddr) → ifaceID in identityIfaceMap so
// that the DISCOVERY_RELAY fan-out path (S-BL.DISCOVERY-WIRE Task 6,
// AC-017/AC-018) can resolve a node's cryptographic address to the sendMap
// key for its live connection.
//
// All three methods are protected by r.mu (write lock for Bind/Unbind,
// read lock for Lookup) — identical to RegisterForwardingEntry /
// LookupForwardingEntry discipline (go.md rule 12).
//
// Purity classification (ARCH-09): pure-core — mutex-protected map
// operations; no I/O; deterministic given lock discipline.
//
// Architecture note: no new imports are needed. internal/routing is ARCH-08
// position 5 and already satisfies all import constraints for this file.
//
// Traces to BC-2.01.010; S-BL.NODE-IDENTIFY-WIRE-rulings.md §8, §12.
package routing

// BindInterface records (svtnID, nodeAddr) → ifaceID after a successful
// NODE_IDENTIFY handshake. Called from onAccept in runRouter after AdmitNode
// returns nil. Last-write-wins (ADR-003): a node reconnect with a new TCP
// connection overwrites the prior binding — the prior connection's cleanup
// func removes it via UnbindInterface when it eventually closes.
//
// BindInterface acquires r.mu write lock.
//
// Traces to BC-2.01.010 PC-1 (binding created), PC-2 (LWW on reconnect),
// PC-4 (write lock held); rulings §8 (method signature), §12 (LWW semantics).
func (r *Router) BindInterface(svtnID [16]byte, nodeAddr [8]byte, ifaceID InterfaceID) {
	// todo: unimplemented
}

// LookupInterface returns the InterfaceID for (svtnID, nodeAddr), or 0 and
// false if no binding exists. Used by the DISCOVERY_RELAY fan-out closure
// (S-BL.DISCOVERY-WIRE Task 6) to resolve a NodeAddr to a send-map key.
//
// LookupInterface acquires r.mu read lock. Return type is (InterfaceID, bool)
// — a value type, not a pointer into internal state (go.md rule 12).
//
// Traces to BC-2.01.010 PC-5 (lookup returns binding if present), PC-6
// (read lock held), PC-7 (return type is value); rulings §8.
func (r *Router) LookupInterface(svtnID [16]byte, nodeAddr [8]byte) (InterfaceID, bool) {
	return 0, false
}

// UnbindInterface removes the (svtnID, nodeAddr) binding. Called from the
// per-connection cleanup func (the func() returned by onAccept to
// netingress.Serve) when the connection closes, so identityIfaceMap stays
// consistent with sendMap.
//
// Stale cleanup guard (BC-2.01.010 PC-9): if the stored binding for
// (svtnID, nodeAddr) maps to a DIFFERENT ifaceID than the caller's own
// (a LWW overwrite occurred and the prior connection's cleanup fires after
// the new binding was installed), UnbindInterface MUST NOT remove the new
// binding. The caller passes its own ifaceID so the guard can detect the
// stale case.
//
// UnbindInterface acquires r.mu write lock.
//
// Signature note (discrepancy with rulings §8 signature): rulings §8 pins
// UnbindInterface(svtnID [16]byte, nodeAddr [8]byte) with no ifaceID
// parameter, but BC-2.01.010 PC-9 and the stale-cleanup guard semantics
// (AC-010 test "call UnbindInterface with old ifaceID=1") require the
// caller's ifaceID to implement the guard correctly. This stub uses the
// three-argument form; the implementer must reconcile with rulings §8 or
// the PO must confirm PC-9's guard requires the third parameter.
//
// Traces to BC-2.01.010 PC-8 (binding removed on close), PC-9 (stale
// cleanup guard), PC-10 (write lock held); rulings §8, §12.
func (r *Router) UnbindInterface(svtnID [16]byte, nodeAddr [8]byte, callerIfaceID InterfaceID) {
	// todo: unimplemented
}
