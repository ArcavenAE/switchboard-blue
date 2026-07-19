// Package routing — identity_test.go: unit tests for BindInterface,
// LookupInterface, and UnbindInterface (BC-2.01.010).
//
// These tests cover AC-010 (LWW rebind, stale cleanup guard) and
// AC-012 (cleanup removes binding). All tests are RED GATE tests —
// they MUST FAIL against the unimplemented stubs in identity.go.
//
// Traces to BC-2.01.010 PC-1 (binding created), PC-2 (LWW on reconnect),
// PC-5 (lookup returns binding), PC-8 (binding removed on close),
// PC-9 (stale cleanup guard).
package routing_test

import (
	"testing"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/routing"
)

// mustNewRouter returns a fresh Router with an empty AdmittedKeySet.
func mustNewRouter(t *testing.T) *routing.Router {
	t.Helper()
	ks := admission.NewAdmittedKeySet()
	return routing.NewRouter(ks)
}

// testSVTNID returns a deterministic [16]byte SVTN ID for testing.
func testSVTNID(b byte) [16]byte {
	var id [16]byte
	id[0] = b
	return id
}

// testNodeAddr returns a deterministic [8]byte node address for testing.
func testNodeAddr(b byte) [8]byte {
	var addr [8]byte
	addr[0] = b
	return addr
}

// ── AC-010: LWW rebind ─────────────────────────────────────────────────────────

// TestBindInterface_LWW_Reconnect_OverwritesPriorBinding verifies that when a
// node reconnects (new IfaceID) for the same (svtnID, nodeAddr), BindInterface
// overwrites the prior binding. LookupInterface returns the new IfaceID.
//
// Traces to BC-2.01.010 PC-2 (LWW on reconnect); AC-010.
func TestBindInterface_LWW_Reconnect_OverwritesPriorBinding(t *testing.T) {
	t.Parallel()

	r := mustNewRouter(t)
	svtnID := testSVTNID(0x01)
	nodeAddr := testNodeAddr(0x01)

	const ifaceID1 routing.InterfaceID = 1
	const ifaceID2 routing.InterfaceID = 2

	r.BindInterface(svtnID, nodeAddr, ifaceID1)
	r.BindInterface(svtnID, nodeAddr, ifaceID2)

	got, ok := r.LookupInterface(svtnID, nodeAddr)
	if !ok {
		t.Fatal("LookupInterface: want (2, true) after LWW overwrite, got (_, false)")
	}
	if got != ifaceID2 {
		t.Errorf("LookupInterface: want ifaceID=2 after LWW overwrite, got %d", got)
	}
}

// TestBindInterface_StaleCleanupGuard_DoesNotRemoveNewBinding verifies that
// calling UnbindInterface with the OLD (stale) ifaceID after a LWW overwrite
// does NOT remove the new binding. LookupInterface still returns the new ifaceID.
//
// Traces to BC-2.01.010 PC-9 (stale cleanup guard); AC-010.
func TestBindInterface_StaleCleanupGuard_DoesNotRemoveNewBinding(t *testing.T) {
	t.Parallel()

	r := mustNewRouter(t)
	svtnID := testSVTNID(0x02)
	nodeAddr := testNodeAddr(0x02)

	const ifaceID1 routing.InterfaceID = 1
	const ifaceID2 routing.InterfaceID = 2

	// Bind ifaceID=1, then LWW overwrite with ifaceID=2.
	r.BindInterface(svtnID, nodeAddr, ifaceID1)
	r.BindInterface(svtnID, nodeAddr, ifaceID2)

	// Prior connection's cleanup fires with OLD ifaceID=1 — stale cleanup guard
	// must suppress this delete.
	r.UnbindInterface(svtnID, nodeAddr, ifaceID1)

	got, ok := r.LookupInterface(svtnID, nodeAddr)
	if !ok {
		t.Fatal("LookupInterface: want (2, true) after stale cleanup guard, got (_, false)")
	}
	if got != ifaceID2 {
		t.Errorf("LookupInterface: want ifaceID=2 after stale cleanup guard, got %d", got)
	}
}

// TestBindInterface_CleanDisconnect_ThenReconnect verifies the clean-disconnect
// path: bind ifaceID=1, unbind with matching ifaceID=1 (lookup returns (0,false)),
// then bind ifaceID=2 (lookup returns (2,true)).
//
// Traces to BC-2.01.010 PC-8 (binding removed on close), EC-005 (clean reconnect); AC-010.
func TestBindInterface_CleanDisconnect_ThenReconnect(t *testing.T) {
	t.Parallel()

	r := mustNewRouter(t)
	svtnID := testSVTNID(0x03)
	nodeAddr := testNodeAddr(0x03)

	const ifaceID1 routing.InterfaceID = 1
	const ifaceID2 routing.InterfaceID = 2

	r.BindInterface(svtnID, nodeAddr, ifaceID1)

	// Clean disconnect: UnbindInterface with matching ifaceID=1.
	r.UnbindInterface(svtnID, nodeAddr, ifaceID1)

	got, ok := r.LookupInterface(svtnID, nodeAddr)
	if ok {
		t.Errorf("LookupInterface after clean unbind: want (0, false), got (%d, true)", got)
	}
	if got != 0 {
		t.Errorf("LookupInterface after clean unbind: want InterfaceID=0, got %d", got)
	}

	// Reconnect: BindInterface re-inserts with ifaceID=2.
	r.BindInterface(svtnID, nodeAddr, ifaceID2)

	got, ok = r.LookupInterface(svtnID, nodeAddr)
	if !ok {
		t.Fatal("LookupInterface after reconnect bind: want (2, true), got (_, false)")
	}
	if got != ifaceID2 {
		t.Errorf("LookupInterface after reconnect bind: want ifaceID=2, got %d", got)
	}
}

// ── AC-012: UnbindInterface removes binding ────────────────────────────────────

// TestUnbindInterface_RemovesBinding verifies that UnbindInterface with the
// MATCHING ifaceID removes the binding. LookupInterface returns (0, false).
//
// Traces to BC-2.01.010 PC-8 (binding removed on close); AC-012.
func TestUnbindInterface_RemovesBinding(t *testing.T) {
	t.Parallel()

	r := mustNewRouter(t)
	svtnID := testSVTNID(0x04)
	nodeAddr := testNodeAddr(0x04)

	const ifaceID routing.InterfaceID = 5

	r.BindInterface(svtnID, nodeAddr, ifaceID)

	// Sanity: binding is present before unbind.
	got, ok := r.LookupInterface(svtnID, nodeAddr)
	if !ok || got != ifaceID {
		t.Fatalf("pre-unbind LookupInterface: want (%d, true), got (%d, %v)", ifaceID, got, ok)
	}

	// UnbindInterface with matching ifaceID must remove the entry.
	r.UnbindInterface(svtnID, nodeAddr, ifaceID)

	got, ok = r.LookupInterface(svtnID, nodeAddr)
	if ok {
		t.Errorf("LookupInterface after UnbindInterface: want (0, false), got (%d, true)", got)
	}
	if got != 0 {
		t.Errorf("LookupInterface after UnbindInterface: want InterfaceID=0, got %d", got)
	}
}

// TestLookupInterface_Unbound_ReturnsFalse verifies that LookupInterface for an
// (svtnID, nodeAddr) with no binding returns (0, false).
//
// Traces to BC-2.01.010 PC-5, EC-003; AC-012.
func TestLookupInterface_Unbound_ReturnsFalse(t *testing.T) {
	t.Parallel()

	r := mustNewRouter(t)
	svtnID := testSVTNID(0x05)
	nodeAddr := testNodeAddr(0x05)

	got, ok := r.LookupInterface(svtnID, nodeAddr)
	if ok {
		t.Errorf("LookupInterface for unbound: want (0, false), got (%d, true)", got)
	}
	if got != 0 {
		t.Errorf("LookupInterface for unbound: want InterfaceID=0, got %d", got)
	}
}

// TestBindInterface_NilNestedMap_AllocatesEntry verifies that BindInterface
// correctly allocates the inner map when svtnID is seen for the first time
// (EC-010: nested map for svtnID absent on first BindInterface call).
//
// Traces to BC-2.01.010 PC-1 (binding created), EC-010.
func TestBindInterface_NilNestedMap_AllocatesEntry(t *testing.T) {
	t.Parallel()

	r := mustNewRouter(t)
	// Use an svtnID not previously seen.
	svtnID := testSVTNID(0xFF)
	nodeAddr := testNodeAddr(0xFF)

	const ifaceID routing.InterfaceID = 42

	r.BindInterface(svtnID, nodeAddr, ifaceID)

	got, ok := r.LookupInterface(svtnID, nodeAddr)
	if !ok {
		t.Fatal("LookupInterface after first-ever bind: want (42, true), got (_, false)")
	}
	if got != ifaceID {
		t.Errorf("LookupInterface after first-ever bind: want %d, got %d", ifaceID, got)
	}
}
