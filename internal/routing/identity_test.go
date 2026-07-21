// Package routing — identity_test.go: unit tests for BindInterface,
// LookupInterface, UnbindInterface, and InterfacesForSVTN (BC-2.01.010).
//
// These tests cover AC-010 (LWW rebind, stale cleanup guard),
// AC-012 (cleanup removes binding), and AC-017 (InterfacesForSVTN
// SVTN-scoped enumeration with originator exclusion). All tests are
// RED GATE tests — they MUST FAIL against unimplemented method stubs.
//
// Traces to BC-2.01.010 PC-1 (binding created), PC-2 (LWW on reconnect),
// PC-5 (lookup returns binding), PC-8 (binding removed on close),
// PC-9 (stale cleanup guard).
package routing_test

import (
	"sync"
	"sync/atomic"
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

// ── AC-017: InterfacesForSVTN enumeration with originator exclusion ────────────

// ifaceSet converts a slice of InterfaceID to a set for order-independent
// equality checks. Two slices with the same elements in any order produce
// equal sets.
func ifaceSet(ids []routing.InterfaceID) map[routing.InterfaceID]struct{} {
	s := make(map[routing.InterfaceID]struct{}, len(ids))
	for _, id := range ids {
		s[id] = struct{}{}
	}
	return s
}

// assertIfaceSetsEqual is a test helper that fails t if the slice result does
// not contain exactly the expected InterfaceIDs (as a set — order-independent).
func assertIfaceSetsEqual(t *testing.T, got []routing.InterfaceID, want ...routing.InterfaceID) {
	t.Helper()
	gotSet := ifaceSet(got)
	wantSet := ifaceSet(want)

	if len(gotSet) != len(wantSet) {
		t.Fatalf("InterfacesForSVTN: got %d elements %v, want %d elements %v",
			len(got), got, len(wantSet), want)
	}
	for id := range wantSet {
		if _, ok := gotSet[id]; !ok {
			t.Errorf("InterfacesForSVTN: result missing expected InterfaceID %d; got %v, want %v",
				id, got, want)
		}
	}
}

// TestInterfacesForSVTN_ExcludesOriginator_ReturnsRest verifies the core
// AC-017 behavior: bind 3 nodes under one svtnID; InterfacesForSVTN with
// the first nodeAddr excluded returns exactly the other two IfaceIDs.
//
// Result order is unspecified (map iteration); comparison is set-based.
//
// Traces to BC-2.01.010 (AC-017); S-BL.DISCOVERY-WIRE Task 6a;
// fanout-resolution-ruling.md Decision 1.
func TestInterfacesForSVTN_ExcludesOriginator_ReturnsRest(t *testing.T) {
	t.Parallel()

	r := mustNewRouter(t)
	svtnID := testSVTNID(0x10)

	nodeA := testNodeAddr(0xA0)
	nodeB := testNodeAddr(0xB0)
	nodeC := testNodeAddr(0xC0)

	const (
		ifaceA routing.InterfaceID = 10
		ifaceB routing.InterfaceID = 20
		ifaceC routing.InterfaceID = 30
	)

	r.BindInterface(svtnID, nodeA, ifaceA)
	r.BindInterface(svtnID, nodeB, ifaceB)
	r.BindInterface(svtnID, nodeC, ifaceC)

	result := r.InterfacesForSVTN(svtnID, nodeA)

	// Result must be {ifaceB, ifaceC} — set comparison, order not assumed.
	assertIfaceSetsEqual(t, result, ifaceB, ifaceC)

	// nodeA's iface must NOT appear.
	for _, id := range result {
		if id == ifaceA {
			t.Errorf("InterfacesForSVTN: excluded originator's ifaceID %d must not appear in result %v",
				ifaceA, result)
		}
	}
}

// TestInterfacesForSVTN_ExcludesOnlyOriginator verifies that when several
// nodes are bound, InterfacesForSVTN excludes ONLY the named originator and
// retains every other bound interface.
//
// Traces to BC-2.01.010 (AC-017); fanout-resolution-ruling.md Decision 1.
func TestInterfacesForSVTN_ExcludesOnlyOriginator(t *testing.T) {
	t.Parallel()

	r := mustNewRouter(t)
	svtnID := testSVTNID(0x11)

	type node struct {
		addr  [8]byte
		iface routing.InterfaceID
	}
	nodes := []node{
		{testNodeAddr(0x01), 101},
		{testNodeAddr(0x02), 102},
		{testNodeAddr(0x03), 103},
		{testNodeAddr(0x04), 104},
		{testNodeAddr(0x05), 105},
	}

	for _, n := range nodes {
		r.BindInterface(svtnID, n.addr, n.iface)
	}

	// Exclude node[2] (addr 0x03, iface 103).
	originator := nodes[2]
	result := r.InterfacesForSVTN(svtnID, originator.addr)

	// Must contain all ifaces except originator.iface.
	expected := make([]routing.InterfaceID, 0, len(nodes)-1)
	for _, n := range nodes {
		if n.addr != originator.addr {
			expected = append(expected, n.iface)
		}
	}
	assertIfaceSetsEqual(t, result, expected...)

	// Originator's iface must be absent.
	for _, id := range result {
		if id == originator.iface {
			t.Errorf("InterfacesForSVTN: excluded originator iface %d must not appear in result %v",
				originator.iface, result)
		}
	}
}

// TestInterfacesForSVTN_ExcludeUnboundNodeAddr_ReturnsAll verifies that
// passing an excludeNodeAddr that has no binding under the svtnID excludes
// nothing — all bound ifaces are returned.
//
// Traces to BC-2.01.010 (AC-017); fanout-resolution-ruling.md Decision 1
// ("excludeNodeAddr not in map → no exclusion").
func TestInterfacesForSVTN_ExcludeUnboundNodeAddr_ReturnsAll(t *testing.T) {
	t.Parallel()

	r := mustNewRouter(t)
	svtnID := testSVTNID(0x12)

	nodeA := testNodeAddr(0xA1)
	nodeB := testNodeAddr(0xB1)

	const (
		ifaceA routing.InterfaceID = 201
		ifaceB routing.InterfaceID = 202
	)

	r.BindInterface(svtnID, nodeA, ifaceA)
	r.BindInterface(svtnID, nodeB, ifaceB)

	// excludeNodeAddr is a fresh addr with no binding.
	unbound := testNodeAddr(0xFF)
	result := r.InterfacesForSVTN(svtnID, unbound)

	// Both ifaces must be present — nothing was excluded.
	assertIfaceSetsEqual(t, result, ifaceA, ifaceB)
}

// TestInterfacesForSVTN_UnknownSVTN_ReturnsNonNilEmptySlice verifies that
// calling InterfacesForSVTN for an svtnID with zero bindings returns a
// non-nil, length-0 slice. An implementation that returns a bare nil on
// a map-miss would fail the non-nil assertion.
//
// Traces to BC-2.01.010 (AC-017); fanout-resolution-ruling.md Decision 2
// (never-nil postcondition).
func TestInterfacesForSVTN_UnknownSVTN_ReturnsNonNilEmptySlice(t *testing.T) {
	t.Parallel()

	r := mustNewRouter(t)
	// svtnID with no bindings at all.
	svtnID := testSVTNID(0x20)
	exclude := testNodeAddr(0x00)

	result := r.InterfacesForSVTN(svtnID, exclude)

	if result == nil {
		t.Fatal("InterfacesForSVTN for unknown SVTN: want non-nil empty slice, got nil")
	}
	if len(result) != 0 {
		t.Fatalf("InterfacesForSVTN for unknown SVTN: want len=0, got len=%d: %v",
			len(result), result)
	}
}

// TestInterfacesForSVTN_AllExcluded_ReturnsNonNilEmptySlice verifies that
// when the only bound node is the one being excluded, InterfacesForSVTN
// returns a non-nil, length-0 slice.
//
// Traces to BC-2.01.010 (AC-017); fanout-resolution-ruling.md Decision 2
// (never-nil postcondition; "all-excluded" boundary).
func TestInterfacesForSVTN_AllExcluded_ReturnsNonNilEmptySlice(t *testing.T) {
	t.Parallel()

	r := mustNewRouter(t)
	svtnID := testSVTNID(0x21)
	nodeA := testNodeAddr(0xA2)
	const ifaceA routing.InterfaceID = 301

	r.BindInterface(svtnID, nodeA, ifaceA)

	// Exclude the only bound node.
	result := r.InterfacesForSVTN(svtnID, nodeA)

	if result == nil {
		t.Fatal("InterfacesForSVTN all-excluded: want non-nil empty slice, got nil")
	}
	if len(result) != 0 {
		t.Fatalf("InterfacesForSVTN all-excluded: want len=0, got len=%d: %v",
			len(result), result)
	}
}

// TestInterfacesForSVTN_SVTNIsolation verifies that bindings under svtnID-1
// and svtnID-2 are completely isolated: querying svtnID-1 never returns
// any iface that belongs to svtnID-2.
//
// Traces to BC-2.01.010 (AC-017); ARCH-04 §SVTN Cryptographic Isolation;
// fanout-resolution-ruling.md Decision 1.
func TestInterfacesForSVTN_SVTNIsolation(t *testing.T) {
	t.Parallel()

	r := mustNewRouter(t)
	svtn1 := testSVTNID(0x30)
	svtn2 := testSVTNID(0x31)

	// Bind two nodes to svtn1 and two different nodes to svtn2.
	node1A := testNodeAddr(0x31)
	node1B := testNodeAddr(0x32)
	node2A := testNodeAddr(0x33)
	node2B := testNodeAddr(0x34)

	const (
		iface1A routing.InterfaceID = 401
		iface1B routing.InterfaceID = 402
		iface2A routing.InterfaceID = 403
		iface2B routing.InterfaceID = 404
	)

	r.BindInterface(svtn1, node1A, iface1A)
	r.BindInterface(svtn1, node1B, iface1B)
	r.BindInterface(svtn2, node2A, iface2A)
	r.BindInterface(svtn2, node2B, iface2B)

	// Query svtn1, exclude node1A — should return only {iface1B}.
	result1 := r.InterfacesForSVTN(svtn1, node1A)
	assertIfaceSetsEqual(t, result1, iface1B)

	// svtn2 ifaces must not leak into svtn1 results.
	svtn2IfaceSet := map[routing.InterfaceID]struct{}{
		iface2A: {},
		iface2B: {},
	}
	for _, id := range result1 {
		if _, leaked := svtn2IfaceSet[id]; leaked {
			t.Errorf("InterfacesForSVTN SVTN isolation: svtn2 iface %d leaked into svtn1 result %v",
				id, result1)
		}
	}

	// Symmetric check: query svtn2, exclude node2B — should return only {iface2A}.
	result2 := r.InterfacesForSVTN(svtn2, node2B)
	assertIfaceSetsEqual(t, result2, iface2A)

	svtn1IfaceSet := map[routing.InterfaceID]struct{}{
		iface1A: {},
		iface1B: {},
	}
	for _, id := range result2 {
		if _, leaked := svtn1IfaceSet[id]; leaked {
			t.Errorf("InterfacesForSVTN SVTN isolation: svtn1 iface %d leaked into svtn2 result %v",
				id, result2)
		}
	}
}

// TestInterfacesForSVTN_ConcurrentBindUnbind_Race exercises InterfacesForSVTN
// under concurrent Bind/Unbind operations to verify the RLock snapshot
// discipline pins the ruling's "RLock snapshot then release" lock contract.
//
// The test asserts:
//   - No panic under concurrent Bind, Unbind, InterfacesForSVTN.
//   - Every InterfacesForSVTN result is a valid (possibly empty) subset of
//     the ever-bound interface IDs — no garbage values, no nil slice.
//   - Race detector reports no data races.
//
// Known flake note: an unrelated concurrent-race test in internal/admission
// (TestLookup_ConcurrentRegisterRace) is NOT referenced or affected here.
//
// Traces to BC-2.01.010 (AC-017); fanout-resolution-ruling.md Decision 1+2;
// go.md rule 12 (RLock snapshot; no internal pointer leak).
func TestInterfacesForSVTN_ConcurrentBindUnbind_Race(t *testing.T) {
	t.Parallel()

	r := mustNewRouter(t)
	svtnID := testSVTNID(0x40)

	// The universe of (nodeAddr, ifaceID) pairs that will be bound/unbound.
	type pair struct {
		addr  [8]byte
		iface routing.InterfaceID
	}
	pairs := []pair{
		{testNodeAddr(0x41), 501},
		{testNodeAddr(0x42), 502},
		{testNodeAddr(0x43), 503},
		{testNodeAddr(0x44), 504},
	}

	// Build the set of all ever-valid ifaces for postcondition check.
	validIfaces := make(map[routing.InterfaceID]struct{}, len(pairs))
	for _, p := range pairs {
		validIfaces[p.iface] = struct{}{}
	}

	const iterations = 500

	var (
		wg      sync.WaitGroup
		panics  atomic.Uint64
		invalid atomic.Uint64
		nils    atomic.Uint64
	)

	// Goroutines alternately Bind and Unbind each pair.
	for _, p := range pairs {
		p := p
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range iterations {
				if i%2 == 0 {
					r.BindInterface(svtnID, p.addr, p.iface)
				} else {
					r.UnbindInterface(svtnID, p.addr, p.iface)
				}
			}
		}()
	}

	// Goroutines call InterfacesForSVTN concurrently with the binders.
	for range len(pairs) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			excludeAddr := testNodeAddr(0x41) // exclude node[0]; arbitrary choice
			for range iterations {
				result := r.InterfacesForSVTN(svtnID, excludeAddr)
				// postcondition: result must never be nil
				if result == nil {
					nils.Add(1)
				}
				// postcondition: every returned ID must belong to the known universe
				for _, id := range result {
					if _, ok := validIfaces[id]; !ok {
						invalid.Add(1)
					}
				}
			}
		}()
	}

	wg.Wait()

	if panics.Load() > 0 {
		t.Errorf("InterfacesForSVTN_ConcurrentBindUnbind_Race: %d panics observed", panics.Load())
	}
	if nils.Load() > 0 {
		t.Errorf("InterfacesForSVTN_ConcurrentBindUnbind_Race: %d nil results (must be non-nil empty slice)",
			nils.Load())
	}
	if invalid.Load() > 0 {
		t.Errorf("InterfacesForSVTN_ConcurrentBindUnbind_Race: %d results contained invalid iface IDs",
			invalid.Load())
	}
}
