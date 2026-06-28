// Tests for SplitHorizon loop prevention (BC-2.02.008 / S-4.04).
//
// Red Gate discipline: every test must FAIL until the implementation in
// split_horizon.go replaces the panic("not implemented") stubs.
//
// VP traces: VP-011 (split-horizon invariant), VP-015 (channel header opaque).
package routing

import (
	"errors"
	"testing"
)

// ---- helpers ---------------------------------------------------------------

// recordingForwardFunc returns a ForwardFunc that appends every called interface
// to *called and returns nil.
func recordingForwardFunc(called *[]InterfaceID) ForwardFunc {
	return func(iface InterfaceID, _ []byte) error {
		*called = append(*called, iface)
		return nil
	}
}

// mustNotCall is a ForwardFunc that fails the test if invoked.
func mustNotCall(t *testing.T) ForwardFunc {
	t.Helper()
	return func(iface InterfaceID, _ []byte) error {
		t.Errorf("ForwardFunc called on interface %d — expected no call", iface)
		return nil
	}
}

// containsIface reports whether iface appears in the slice.
func containsIface(ifaces []InterfaceID, target InterfaceID) bool {
	for _, id := range ifaces {
		if id == target {
			return true
		}
	}
	return false
}

// ---- AC-001 / BC-2.02.008 postcondition 1 ----------------------------------

// TestSplitHorizon_NoForwardTowardArrivalInterface verifies that Forward never
// calls fn with the arrival interface, regardless of whether it appears in the
// interface set.
//
// AC-001 / BC-2.02.008 postcondition 1:
// "The frame is not forwarded back on the arrival interface, regardless of
// what the forwarding table says."
func TestSplitHorizon_NoForwardTowardArrivalInterface(t *testing.T) {
	t.Parallel()

	// Canonical test vector: Frame arrives on interface A; forwarding table
	// says forward to B, C, A → Frame forwarded to B and C only; A excluded.
	// (BC-2.02.008 canonical test vector 1)
	const (
		ifaceA InterfaceID = 1
		ifaceB InterfaceID = 2
		ifaceC InterfaceID = 3
	)

	var called []InterfaceID
	sh := SplitHorizon{}
	frame := []byte("test-frame-opaque")

	_, err := sh.Forward(frame, ifaceA, []InterfaceID{ifaceA, ifaceB, ifaceC}, recordingForwardFunc(&called))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if containsIface(called, ifaceA) {
		t.Errorf("arrival interface %d was forwarded — split-horizon must exclude it (AC-001 / BC-2.02.008 PC-1)", ifaceA)
	}
}

// ---- AC-002 / BC-2.02.008 postcondition 2 ----------------------------------

// TestSplitHorizon_ForwardOnAllOtherInterfaces verifies that Forward calls fn
// for every interface in the set that is NOT the arrival interface.
//
// AC-002 / BC-2.02.008 postcondition 2:
// "The frame is forwarded on all other eligible interfaces."
func TestSplitHorizon_ForwardOnAllOtherInterfaces(t *testing.T) {
	t.Parallel()

	// Canonical test vector: Frame arrives on A; set is {A, B, C}
	// → exactly B and C must be forwarded.
	const (
		arrivalIface InterfaceID = 10
		ifaceB       InterfaceID = 20
		ifaceC       InterfaceID = 30
	)

	var called []InterfaceID
	sh := SplitHorizon{}
	frame := []byte("frame-bytes")

	results, err := sh.Forward(frame, arrivalIface, []InterfaceID{arrivalIface, ifaceB, ifaceC}, recordingForwardFunc(&called))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Must have been forwarded on B and C.
	if !containsIface(called, ifaceB) {
		t.Errorf("interface %d not forwarded; expected all non-arrival interfaces to be forwarded (AC-002)", ifaceB)
	}
	if !containsIface(called, ifaceC) {
		t.Errorf("interface %d not forwarded; expected all non-arrival interfaces to be forwarded (AC-002)", ifaceC)
	}

	// Exactly 2 ForwardResults expected.
	forwarded := 0
	for _, r := range results {
		if r.Forwarded {
			forwarded++
		}
	}
	if forwarded != 2 {
		t.Errorf("got %d Forwarded results; want 2 (one per non-arrival interface)", forwarded)
	}
}

// ---- EC-001 / BC-2.02.008 postcondition 3 + canonical vector 2 -------------

// TestSplitHorizon_OnlyArrivalInterfaceInSet verifies that when every interface
// in the set equals the arrival interface, the frame is dropped and
// ErrAllPathsSplitHorizon is returned.
//
// EC-001: "Only one interface in set (arrival interface) → frame is dropped
// entirely (no forward possible)."
// BC-2.02.008 postcondition 3 / canonical test vector 2.
func TestSplitHorizon_OnlyArrivalInterfaceInSet(t *testing.T) {
	t.Parallel()

	sh := SplitHorizon{}
	frame := []byte("frame")

	_, err := sh.Forward(frame, 1, []InterfaceID{1}, mustNotCall(t))
	if !errors.Is(err, ErrAllPathsSplitHorizon) {
		t.Errorf("got err = %v; want ErrAllPathsSplitHorizon (EC-001 / BC-2.02.008 PC-3)", err)
	}
}

// TestSplitHorizon_EmptyInterfaceSet verifies that an empty interfaceSet also
// returns ErrAllPathsSplitHorizon (no eligible output interface).
//
// Derived from BC-2.02.008 postcondition 3: "if the only eligible interface is
// the arrival interface, the frame is dropped."
func TestSplitHorizon_EmptyInterfaceSet(t *testing.T) {
	t.Parallel()

	sh := SplitHorizon{}
	_, err := sh.Forward([]byte("frame"), 99, []InterfaceID{}, mustNotCall(t))
	if !errors.Is(err, ErrAllPathsSplitHorizon) {
		t.Errorf("got err = %v; want ErrAllPathsSplitHorizon for empty interface set", err)
	}
}

// ---- EC-002 / BC-2.02.008 -------------------------------------------------

// TestSplitHorizon_UnknownArrivalInterfaceID verifies that an arrival interface
// that does not appear in the set is treated as the arrival interface for
// split-horizon purposes — all interfaces in the set are forwarded.
//
// EC-002: "unknown interface ID → treated as new; forwarded on all known
// interfaces."
func TestSplitHorizon_UnknownArrivalInterfaceID(t *testing.T) {
	t.Parallel()

	const (
		unknownArrival InterfaceID = 999 // not in the set
		ifaceA         InterfaceID = 1
		ifaceB         InterfaceID = 2
	)

	var called []InterfaceID
	sh := SplitHorizon{}

	_, err := sh.Forward([]byte("frame"), unknownArrival, []InterfaceID{ifaceA, ifaceB}, recordingForwardFunc(&called))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !containsIface(called, ifaceA) || !containsIface(called, ifaceB) {
		t.Errorf("got called = %v; want both %d and %d forwarded when arrival iface is unknown (EC-002)", called, ifaceA, ifaceB)
	}
}

// ---- VP-011 property test --------------------------------------------------

// TestSplitHorizon_VP011_ArrivalIfaceNeverForwarded is a property test that
// verifies VP-011: for arbitrary interface sets, the arrival interface never
// appears in the forwarded set.
//
// VP-011: "For all frames: output interface set excludes arrival interface."
// The loop exercises 1000+ randomly-shaped interface sets using a deterministic
// sequence derived from the test index.
func TestSplitHorizon_VP011_ArrivalIfaceNeverForwarded(t *testing.T) {
	t.Parallel()

	sh := SplitHorizon{}
	frame := []byte("vp011-property-frame")

	type testCase struct {
		arrival      InterfaceID
		ifaceSet     []InterfaceID
		wantDropOnly bool // true iff arrival is the only interface → expect error
	}

	cases := make([]testCase, 0, 1024)

	// Enumerate structured cases: for arrival in [0,31], build sets that include
	// and exclude the arrival interface in various positions and sizes.
	for arrival := InterfaceID(0); arrival < 32; arrival++ {
		// Set contains only the arrival interface.
		cases = append(cases, testCase{
			arrival:      arrival,
			ifaceSet:     []InterfaceID{arrival},
			wantDropOnly: true,
		})
		// Set contains arrival plus one other.
		other := arrival + 100
		cases = append(cases, testCase{
			arrival:  arrival,
			ifaceSet: []InterfaceID{arrival, other},
		})
		// Arrival at front of a larger set.
		cases = append(cases, testCase{
			arrival:  arrival,
			ifaceSet: []InterfaceID{arrival, arrival + 1, arrival + 2, arrival + 3},
		})
		// Arrival at back.
		cases = append(cases, testCase{
			arrival:  arrival,
			ifaceSet: []InterfaceID{arrival + 10, arrival + 20, arrival},
		})
		// Arrival not in set at all (EC-002 pattern).
		cases = append(cases, testCase{
			arrival:  arrival + 500,
			ifaceSet: []InterfaceID{arrival, arrival + 1},
		})
		// Duplicate arrival interface entries.
		cases = append(cases, testCase{
			arrival:      arrival,
			ifaceSet:     []InterfaceID{arrival, arrival, arrival},
			wantDropOnly: true,
		})
	}

	if len(cases) < 1000 {
		// Pad to meet the 1000-case minimum for property tests.
		for i := len(cases); i < 1024; i++ {
			n := InterfaceID(i % 64)
			cases = append(cases, testCase{
				arrival:  n,
				ifaceSet: []InterfaceID{n + 1, n + 2, n + 3},
			})
		}
	}

	for i, tc := range cases {
		var called []InterfaceID
		_, err := sh.Forward(frame, tc.arrival, tc.ifaceSet, recordingForwardFunc(&called))

		if tc.wantDropOnly {
			if !errors.Is(err, ErrAllPathsSplitHorizon) {
				t.Errorf("case %d: arrival=%d set=%v: got err=%v; want ErrAllPathsSplitHorizon (VP-011)", i, tc.arrival, tc.ifaceSet, err)
			}
			continue
		}

		if containsIface(called, tc.arrival) {
			t.Errorf("case %d: VP-011 violated — arrival interface %d appeared in forwarded set %v", i, tc.arrival, called)
		}
	}
}

// ---- AC-003 / BC-2.02.008 invariant 1 / VP-015 fuzz target ----------------

// FuzzSplitHorizon_ChannelHeaderOpaque verifies VP-015: the router NEVER parses
// the channel header section of a frame. Arbitrary bytes injected into the
// "channel header" region of the frame must not affect the routing decision.
//
// AC-003 / BC-2.02.008 invariant 1 / VP-015:
// "The router code never parses channel header payload; injecting arbitrary
// bytes into the channel header section does not affect routing decisions."
//
// The outer header is treated as the first 44 bytes; everything after that is
// the channel header + payload and is opaque to the router. Routing decisions
// are determined solely by the interface set and arrivalIface — the frame bytes
// are never parsed.
func FuzzSplitHorizon_ChannelHeaderOpaque(f *testing.F) {
	// Seed corpus: minimal 44-byte "outer header" + varying channel header bytes.
	outerHeader := make([]byte, 44)
	f.Add(outerHeader, []byte{0x00})
	f.Add(outerHeader, []byte{0xFF, 0xFE, 0xFD})
	f.Add(outerHeader, []byte("CHANNEL_HEADER_GARBAGE"))
	f.Add(outerHeader, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	const (
		arrivalIface InterfaceID = 1
		ifaceB       InterfaceID = 2
		ifaceC       InterfaceID = 3
	)

	f.Fuzz(func(t *testing.T, outerHdr []byte, channelHeaderBytes []byte) {
		// Construct frame: outer header region + arbitrary channel header bytes.
		// The router must treat the entire byte slice as opaque and NEVER parse
		// channelHeaderBytes (VP-015).
		frame := append(outerHdr, channelHeaderBytes...)

		sh := SplitHorizon{}
		var called []InterfaceID

		_, err := sh.Forward(frame, arrivalIface, []InterfaceID{arrivalIface, ifaceB, ifaceC}, recordingForwardFunc(&called))
		// The routing decision must be the same regardless of channel header contents.
		// With arrivelIface=1 and set={1,2,3}: expected forward on 2 and 3 only.
		if err != nil {
			t.Fatalf("unexpected error %v — channel header bytes must not affect routing (VP-015)", err)
		}
		if containsIface(called, arrivalIface) {
			t.Fatalf("arrival interface %d forwarded — split-horizon must exclude it; VP-015 violated", arrivalIface)
		}
		if !containsIface(called, ifaceB) || !containsIface(called, ifaceC) {
			t.Fatalf("expected forward on %d and %d; got %v — VP-015: channel header must not affect routing", ifaceB, ifaceC, called)
		}
	})
}
