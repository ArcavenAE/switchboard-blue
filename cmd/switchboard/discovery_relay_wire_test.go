// discovery_relay_wire_test.go covers AC-014, AC-015, AC-016 (DISCOVERY_RELAY
// hop-2 frame-assembly, Task 5) and AC-017 (hop-2 fan-out dispatch, Task 6b).
// AC-018 (relay rate cap, Task 6c) is in scope for a follow-on test function
// in this same file once Task 6c is ready.
package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/discovery"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/routing"
)

// TestAssembleDiscoveryRelayFrame_PayloadLayout verifies AC-014: the
// assembled DISCOVERY_RELAY frame is a FrameTypeCtl outer frame whose
// payload matches Decision 3(c)'s byte layout exactly — control_type=0x03,
// version=0x01, reserved=0x0000 at bytes 0-3; NodeAddr at bytes 4-11;
// Sequence (BE uint64) at bytes 12-19; session count (BE uint16) at bytes
// 20-21; sessions at bytes 22+ — and SVTNID is carried by the outer
// header, not repeated in the payload.
func TestAssembleDiscoveryRelayFrame_PayloadLayout(t *testing.T) {
	t.Parallel()
	defer redGateGuard(t)

	svtnID := [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
	nodeAddr := [8]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22}
	sequence := uint64(0x0102030405060708)
	sessions := []discovery.SessionPresence{
		{SessionName: "agent-01", Status: discovery.Attached, Quality: discovery.QualityGreen},
	}

	raw := assembleDiscoveryRelayFrame(svtnID, nodeAddr, sequence, sessions)

	if len(raw) < frame.OuterHeaderSize {
		t.Fatalf("assembleDiscoveryRelayFrame: got %d bytes, shorter than the %d-byte outer header alone", len(raw), frame.OuterHeaderSize)
	}
	hdr, err := frame.ParseOuterHeader(raw[:frame.OuterHeaderSize])
	if err != nil {
		t.Fatalf("ParseOuterHeader: %v", err)
	}
	if hdr.FrameType != frame.FrameTypeCtl {
		t.Errorf("hdr.FrameType = %v, want FrameTypeCtl (0x03)", hdr.FrameType)
	}
	if hdr.SVTNID != svtnID {
		t.Errorf("hdr.SVTNID = %x, want %x", hdr.SVTNID, svtnID)
	}

	payload := raw[frame.OuterHeaderSize:]
	if int(hdr.PayloadLen) != len(payload) {
		t.Errorf("hdr.PayloadLen = %d, want %d (len(payload))", hdr.PayloadLen, len(payload))
	}
	if len(payload) < 22 {
		t.Fatalf("payload = %d bytes, want at least 22 (4 control header + 8 NodeAddr + 8 Sequence + 2 count)", len(payload))
	}

	const discoveryRelayControlType = 0x03
	if payload[0] != discoveryRelayControlType {
		t.Errorf("payload[0] (control_type) = %#x, want %#x (DISCOVERY_RELAY)", payload[0], discoveryRelayControlType)
	}
	if payload[1] != 0x01 {
		t.Errorf("payload[1] (version) = %#x, want 0x01", payload[1])
	}
	if payload[2] != 0x00 || payload[3] != 0x00 {
		t.Errorf("payload[2:4] (reserved) = %x, want 0x0000", payload[2:4])
	}

	var gotNodeAddr [8]byte
	copy(gotNodeAddr[:], payload[4:12])
	if gotNodeAddr != nodeAddr {
		t.Errorf("payload[4:12] (NodeAddr) = %x, want %x", gotNodeAddr, nodeAddr)
	}

	gotSequence := binary.BigEndian.Uint64(payload[12:20])
	if gotSequence != sequence {
		t.Errorf("payload[12:20] (Sequence) = %#x, want %#x", gotSequence, sequence)
	}

	gotCount := binary.BigEndian.Uint16(payload[20:22])
	if int(gotCount) != len(sessions) {
		t.Errorf("payload[20:22] (session count) = %d, want %d", gotCount, len(sessions))
	}

	// Decode the session-list tail using discovery.go's own per-session
	// wire encoding (uint16 name_len | name | uint8 status | uint8
	// quality) and confirm it round-trips the session that was assembled.
	tail := payload[22:]
	if len(tail) < 2 {
		t.Fatalf("session-list tail = %d bytes, too short for one session entry", len(tail))
	}
	nameLen := int(binary.BigEndian.Uint16(tail[0:2]))
	if 2+nameLen+2 > len(tail) {
		t.Fatalf("session-list tail truncated: declares name_len=%d but tail is only %d bytes", nameLen, len(tail))
	}
	gotName := string(tail[2 : 2+nameLen])
	if gotName != sessions[0].SessionName {
		t.Errorf("decoded session name = %q, want %q", gotName, sessions[0].SessionName)
	}
	gotStatus := discovery.AttachmentStatus(tail[2+nameLen])
	gotQuality := discovery.QualityIndicator(tail[2+nameLen+1])
	if gotStatus != sessions[0].Status {
		t.Errorf("decoded status = %v, want %v", gotStatus, sessions[0].Status)
	}
	if gotQuality != sessions[0].Quality {
		t.Errorf("decoded quality = %v, want %v", gotQuality, sessions[0].Quality)
	}
}

// TestAssembleDiscoveryRelayFrame_ZeroHMACTag verifies AC-015: the relay
// frame's OuterHeader.HMACTag is the zero value — hop-2's trust boundary is
// the admitted TCP connection, not a per-frame HMAC, matching the
// S-7.04-FU-DRAIN-WIRE DRAIN precedent exactly.
func TestAssembleDiscoveryRelayFrame_ZeroHMACTag(t *testing.T) {
	t.Parallel()
	defer redGateGuard(t)

	svtnID := [16]byte{0x20}
	nodeAddr := [8]byte{0x30}
	sessions := []discovery.SessionPresence{
		{SessionName: "agent-02", Status: discovery.Detached, Quality: discovery.QualityYellow},
	}

	raw := assembleDiscoveryRelayFrame(svtnID, nodeAddr, 1, sessions)
	if len(raw) < frame.OuterHeaderSize {
		t.Fatalf("assembleDiscoveryRelayFrame: got %d bytes, shorter than the outer header alone", len(raw))
	}
	hdr, err := frame.ParseOuterHeader(raw[:frame.OuterHeaderSize])
	if err != nil {
		t.Fatalf("ParseOuterHeader: %v", err)
	}
	var zeroTag [8]byte
	if hdr.HMACTag != zeroTag {
		t.Errorf("hdr.HMACTag = %x, want zero value", hdr.HMACTag)
	}
}

// TestAssembleDiscoveryRelayFrame_NotRawHop1Bytes verifies AC-016: the
// relay frame's payload bytes are freshly constructed from the decoded
// fields — never a byte-for-byte copy of hop-1's raw UDP datagram — and
// hop-1's original HMAC tag never appears anywhere in the relay frame.
func TestAssembleDiscoveryRelayFrame_NotRawHop1Bytes(t *testing.T) {
	t.Parallel()
	defer redGateGuard(t)

	svtnID := [16]byte{0x40}
	nodeAddr := [8]byte{0x50}
	sequence := uint64(7)
	sessions := []discovery.SessionPresence{
		{SessionName: "agent-03", Status: discovery.Attached, Quality: discovery.QualityRed},
	}

	// A distinctive, non-zero hop-1 HMAC tag that would be trivially
	// detectable if it leaked verbatim into the relay frame.
	hop1Tag := [8]byte{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE}
	hop1Body := make([]byte, 0, 16+8+8+2)
	hop1Body = append(hop1Body, svtnID[:]...)
	hop1Body = append(hop1Body, nodeAddr[:]...)
	hop1Body = binary.BigEndian.AppendUint64(hop1Body, sequence)
	hop1Body = binary.BigEndian.AppendUint16(hop1Body, uint16(len(sessions)))
	hop1Raw := make([]byte, 0, len(hop1Tag)+len(hop1Body))
	hop1Raw = append(hop1Raw, hop1Tag[:]...)
	hop1Raw = append(hop1Raw, hop1Body...)

	relayRaw := assembleDiscoveryRelayFrame(svtnID, nodeAddr, sequence, sessions)

	if bytes.Equal(relayRaw, hop1Raw) {
		t.Error("assembleDiscoveryRelayFrame produced byte-identical output to a raw hop-1 datagram — must be re-serialized (AC-016 postcondition 1)")
	}
	if bytes.Contains(relayRaw, hop1Tag[:]) {
		t.Errorf("hop-1 HMAC tag %x appears inside the relay frame — hop-1's original HMAC tag must never appear anywhere in the relay frame (AC-016 postcondition 2)", hop1Tag)
	}
}

// TestAssembleDiscoveryRelayFrame_IngestRelayAdvertisement_RoundTrip is a
// cross-function bridging test: assembleDiscoveryRelayFrame (this package,
// AC-014) and discovery.IngestRelayAdvertisement (internal/discovery,
// AC-007) are complementary halves of the hop-2 relay path with no prior
// test exercising both together — every existing test on each side builds
// its own hand-rolled bytes instead of feeding one function's real output
// into the other. That is the same shape of gap that hid the F-DWIP1-001
// HIGH finding on the hop-1 side (Encode/RouterIngest.Ingest), so this is
// regression-guard coverage for a class of defect already proven to occur
// here, not a report of a currently-known bug — assemble and ingest were
// manually verified consistent (assemble's payload[4:] == ingest's
// expected input) before writing this test.
//
// Builds a real DISCOVERY_RELAY frame via assembleDiscoveryRelayFrame,
// strips the frame's 44-byte outer header and then the payload's 4-byte
// control header exactly as the router's hop-2 relay-dispatch path would
// (see IngestRelayAdvertisement's doc comment: it expects payload bytes
// starting after that 4-byte header), and feeds the result into a real
// discovery.Discovery's IngestRelayAdvertisement with the matching SVTNID.
// Asserts acceptance and that every session field round-trips into
// Enumerate's result.
//
// Placement: cmd/switchboard, not internal/discovery — this is the only
// package that can see both assembleDiscoveryRelayFrame (unexported, package
// main) and discovery.IngestRelayAdvertisement (exported, importable).
func TestAssembleDiscoveryRelayFrame_IngestRelayAdvertisement_RoundTrip(t *testing.T) {
	t.Parallel()
	defer redGateGuard(t)

	svtnID := [16]byte{0x60, 0x61, 0x62, 0x63}
	nodeAddr := [8]byte{0x70, 0x71, 0x72, 0x73}
	sequence := uint64(42)
	sessions := []discovery.SessionPresence{
		{SessionName: "agent-04", Status: discovery.Attached, Quality: discovery.QualityGreen},
		{SessionName: "agent-05", Status: discovery.Detached, Quality: discovery.QualityRed},
	}

	raw := assembleDiscoveryRelayFrame(svtnID, nodeAddr, sequence, sessions)

	hdr, err := frame.ParseOuterHeader(raw[:frame.OuterHeaderSize])
	if err != nil {
		t.Fatalf("ParseOuterHeader: %v", err)
	}
	fullPayload := raw[frame.OuterHeaderSize:]
	if len(fullPayload) < 4 {
		t.Fatalf("relay frame payload = %d bytes, too short to contain the 4-byte control header", len(fullPayload))
	}
	// Strip the 4-byte control header (control_type | version | reserved |
	// reserved) — the same offset IngestRelayAdvertisement's doc comment
	// specifies the caller must strip before calling it.
	ingestPayload := fullPayload[4:]

	d := discovery.New(discovery.Config{LocalSVTNID: svtnID})
	if err := d.IngestRelayAdvertisement(hdr.SVTNID, ingestPayload); err != nil {
		t.Fatalf("IngestRelayAdvertisement: unexpected error: %v (assembleDiscoveryRelayFrame's output was not accepted by its own complementary ingest function)", err)
	}

	entries, err := d.Enumerate(context.Background())
	if err != nil {
		t.Fatalf("Enumerate: unexpected error: %v", err)
	}
	got := make(map[string]discovery.SessionEntry, len(entries))
	for _, e := range entries {
		got[e.Presence.SessionName] = e
	}
	if len(got) != len(sessions) {
		t.Fatalf("Enumerate: got %d sessions, want %d", len(got), len(sessions))
	}
	for _, want := range sessions {
		entry, ok := got[want.SessionName]
		if !ok {
			t.Errorf("Enumerate: missing session %q", want.SessionName)
			continue
		}
		if entry.AdvertiserAddr != nodeAddr {
			t.Errorf("session %q: AdvertiserAddr = %x, want %x", want.SessionName, entry.AdvertiserAddr, nodeAddr)
		}
		if entry.Presence.Status != want.Status {
			t.Errorf("session %q: Status = %v, want %v", want.SessionName, entry.Presence.Status, want.Status)
		}
		if entry.Presence.Quality != want.Quality {
			t.Errorf("session %q: Quality = %v, want %v", want.SessionName, entry.Presence.Quality, want.Quality)
		}
	}
}

// TestAssembleDiscoveryRelayFrame_PayloadOversize_Panics verifies the
// F-DWIP4-N1 guard: an assembled payload whose total byte size exceeds
// math.MaxUint16 (65535) — the wire size of OuterHeader.PayloadLen — panics
// rather than silently truncating PayloadLen via the uint16(len(payload))
// conversion. Currently unreachable via any real caller (sessions only ever
// arrive already bounded by discovery.MaxDiscoveryDatagramSize=32768, and
// re-encoding here never expands that; Task 6's relay-dispatch closure is
// GATED), but is the same class of silent wire-field-truncation defect
// F-DWIP1-001 found on the hop-1 side, so it is guarded and tested
// explicitly rather than left implicit.
//
// Does not use redGateGuard: that helper recovers ANY panic and fails the
// test via t.Fatalf, which would defeat this test's own deliberate panic
// assertion.
func TestAssembleDiscoveryRelayFrame_PayloadOversize_Panics(t *testing.T) {
	t.Parallel()

	svtnID := [16]byte{0x80}
	nodeAddr := [8]byte{0x90}

	// 300 sessions of 255-byte names comfortably exceeds the 65535-byte
	// PayloadLen limit (300 * (2+255+1+1) + 4 control + 8 NodeAddr +
	// 8 Sequence + 2 count ~= 77722 bytes) — well clear of the boundary so
	// this isn't a fragile off-by-one.
	longName := strings.Repeat("a", 255)
	sessions := make([]discovery.SessionPresence, 300)
	for i := range sessions {
		sessions[i] = discovery.SessionPresence{
			SessionName: longName,
			Status:      discovery.Attached,
			Quality:     discovery.QualityGreen,
		}
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("assembleDiscoveryRelayFrame: did not panic for an oversized payload, want a panic guarding against silent PayloadLen truncation")
			return
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "exceeds") || !strings.Contains(msg, "PayloadLen") {
			t.Errorf("assembleDiscoveryRelayFrame: panic value = %v, want a string message mentioning the PayloadLen bound being exceeded", r)
		}
	}()

	_ = assembleDiscoveryRelayFrame(svtnID, nodeAddr, 1, sessions)
}

// makeRelayTestNodeConn returns a *nodeConn whose send channel has the given
// buffer capacity. done and doneOnce are wired correctly (same invariants as
// the production nodeConn — done NEVER closed directly, only via doneOnce).
// The writerExited channel is set up but intentionally not closed in tests;
// none of the relay-dispatch paths write to it.
func makeRelayTestNodeConn(t *testing.T, sendBuf int) *nodeConn {
	t.Helper()
	return &nodeConn{
		send:         make(chan []byte, sendBuf),
		done:         make(chan struct{}),
		doneOnce:     &sync.Once{},
		writerExited: make(chan struct{}),
	}
}

// nonBlockingDrain drains exactly one frame from nc.send without blocking,
// returning (frame, true) if present and (nil, false) otherwise.
func nonBlockingDrain(nc *nodeConn) ([]byte, bool) {
	select {
	case b := <-nc.send:
		return b, true
	default:
		return nil, false
	}
}

// mustNotReceive asserts that nc.send has no frame buffered at the moment of
// the call. It is a discriminating assertion: if it fires, the channel was
// written when it should not have been. Call AFTER relayDispatch returns
// (relayDispatch is synchronous once the in-scope select-default fires).
func mustNotReceive(t *testing.T, nc *nodeConn, label string) {
	t.Helper()
	select {
	case b := <-nc.send:
		t.Errorf("%s: unexpected frame on send channel (%d bytes); originator must be excluded from fan-out", label, len(b))
	default:
		// correct — nothing was delivered
	}
}

// buildRelayRouter constructs a Router whose AdmittedKeySet contains one
// registered key per (svtnID, nodeAddr) pair, then calls BindInterface to
// populate identityIfaceMap. The returned router is ready for
// Router.InterfacesForSVTN calls. We call BindInterface only — not the full
// NODE_IDENTIFY handshake — because relayDispatch tests exercise the dispatch
// helper directly, not the wire path.
func buildRelayRouter(t *testing.T, bindings []struct {
	svtnID   [16]byte
	nodeAddr [8]byte
	ifaceID  routing.InterfaceID
},
) *routing.Router {
	t.Helper()
	ks := admission.NewAdmittedKeySet()
	// Register a synthetic key for each distinct (svtnID, nodeAddr) so the
	// AdmittedKeySet is in a consistent state. The key material itself does not
	// matter for relay-dispatch tests — only identityIfaceMap is queried.
	for _, b := range bindings {
		syntheticPub := make([]byte, 32)
		syntheticPub[0] = byte(b.ifaceID & 0xFF) // distinct per entry, non-zero
		ks.RegisterKey(b.svtnID, syntheticPub, admission.RoleAccess)
	}
	r := routing.NewRouter(ks)
	for _, b := range bindings {
		r.BindInterface(b.svtnID, b.nodeAddr, b.ifaceID)
	}
	return r
}

// TestRelayDispatch_SVTNScoped_ExcludeOriginator_BestEffortNonBlocking is the
// mandated AC-017 test (S-BL.DISCOVERY-WIRE story, Task 6b). Postconditions:
//
//  1. The router iterates live connections for the advertisement's SVTN,
//     excluding the originating NodeAddr.
//  2. Dispatch is best-effort non-blocking: select { case nc.send<-frame: default: }
//  3. The originating node does not receive an echo via hop-2.
//  4. No queueing, no retry, no wire ACK.
//
// All subtests call relayDispatch — an undefined symbol at RED time. The
// expected RED state is a compile-fail: "undefined: relayDispatch".
//
// Traces to BC-2.03.001 Postcondition 1 delivery-mechanism note; fanout-
// resolution-ruling.md v1.0 Decisions 1/2/3; S-BL.DISCOVERY-WIRE AC-017.
func TestRelayDispatch_SVTNScoped_ExcludeOriginator_BestEffortNonBlocking(t *testing.T) {
	// Fixed test vectors — not invented; derived from AC-017 postconditions.
	svtnID := [16]byte{0x10, 0x20, 0x30, 0x40, 0x50, 0x60, 0x70, 0x80, 0x90, 0xA0, 0xB0, 0xC0, 0xD0, 0xE0, 0xF0, 0x00}
	nodeAddrA := [8]byte{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA}
	nodeAddrB := [8]byte{0xBB, 0xBB, 0xBB, 0xBB, 0xBB, 0xBB, 0xBB, 0xBB}
	nodeAddrC := [8]byte{0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC}
	ifaceIDA := routing.InterfaceID(101)
	ifaceIDB := routing.InterfaceID(102)
	ifaceIDC := routing.InterfaceID(103)

	sessions := []discovery.SessionPresence{
		{SessionName: "test-session", Status: discovery.Attached, Quality: discovery.QualityGreen},
	}
	const testSeq = uint64(42)

	// expectedFrame builds the canonical DISCOVERY_RELAY frame that should be
	// delivered to each non-originating node (AC-017 PC-4, frame-identity
	// check). Callers compare received bytes against this value.
	expectedFrame := func(t *testing.T) []byte {
		t.Helper()
		return assembleDiscoveryRelayFrame(svtnID, nodeAddrA, testSeq, sessions)
	}

	// --- subtest 1: two nodes, A originates, B receives, A does not ---
	t.Run("two_nodes_originator_excluded", func(t *testing.T) {
		// PC-1: router iterates live connections for advertisement's SVTN,
		//       excluding the originating NodeAddr.
		// PC-3: originating node does not receive echo.
		router := buildRelayRouter(t, []struct {
			svtnID   [16]byte
			nodeAddr [8]byte
			ifaceID  routing.InterfaceID
		}{
			{svtnID, nodeAddrA, ifaceIDA},
			{svtnID, nodeAddrB, ifaceIDB},
		})

		ncA := makeRelayTestNodeConn(t, 4)
		ncB := makeRelayTestNodeConn(t, 4)
		var sendMap sync.Map
		sendMap.Store(ifaceIDA, ncA)
		sendMap.Store(ifaceIDB, ncB)

		decision := discovery.RouterIngestDecision{
			Accept:   true,
			Relay:    true,
			SVTNID:   svtnID,
			NodeAddr: nodeAddrA, // A is the originator
			Sequence: testSeq,
			Sessions: sessions,
		}

		relayDispatch(router, &sendMap, decision)

		// B must receive exactly one frame.
		got, ok := nonBlockingDrain(ncB)
		if !ok {
			t.Error("node B: expected to receive relay frame, got nothing")
		} else if want := expectedFrame(t); !bytes.Equal(got, want) {
			t.Errorf("node B: frame mismatch: got %d bytes, want %d bytes", len(got), len(want))
		}

		// A must NOT receive its own advertisement echoed back.
		mustNotReceive(t, ncA, "node A (originator)")
	})

	// --- subtest 2: three nodes, A originates, B and C receive, A does not ---
	t.Run("three_nodes_fanout_width_two", func(t *testing.T) {
		// PC-1 + PC-3 at fan-out width >1.
		router := buildRelayRouter(t, []struct {
			svtnID   [16]byte
			nodeAddr [8]byte
			ifaceID  routing.InterfaceID
		}{
			{svtnID, nodeAddrA, ifaceIDA},
			{svtnID, nodeAddrB, ifaceIDB},
			{svtnID, nodeAddrC, ifaceIDC},
		})

		ncA := makeRelayTestNodeConn(t, 4)
		ncB := makeRelayTestNodeConn(t, 4)
		ncC := makeRelayTestNodeConn(t, 4)
		var sendMap sync.Map
		sendMap.Store(ifaceIDA, ncA)
		sendMap.Store(ifaceIDB, ncB)
		sendMap.Store(ifaceIDC, ncC)

		decision := discovery.RouterIngestDecision{
			Accept:   true,
			Relay:    true,
			SVTNID:   svtnID,
			NodeAddr: nodeAddrA,
			Sequence: testSeq,
			Sessions: sessions,
		}

		relayDispatch(router, &sendMap, decision)

		// B and C must each receive exactly one frame.
		for _, tc := range []struct {
			nc    *nodeConn
			label string
		}{
			{ncB, "node B"},
			{ncC, "node C"},
		} {
			got, ok := nonBlockingDrain(tc.nc)
			if !ok {
				t.Errorf("%s: expected to receive relay frame, got nothing", tc.label)
			} else if want := expectedFrame(t); !bytes.Equal(got, want) {
				t.Errorf("%s: frame mismatch: got %d bytes, want %d bytes", tc.label, len(got), len(want))
			}
		}

		// A (originator) must not receive anything.
		mustNotReceive(t, ncA, "node A (originator)")
	})

	// --- subtest 3: best-effort non-blocking — full send channel does not stall ---
	t.Run("best_effort_nonblocking_full_channel", func(t *testing.T) {
		// PC-2: dispatch is best-effort non-blocking.
		// A has a full (unbuffered) send channel — relayDispatch must not block.
		// B has a normally-buffered send channel — still receives.
		router := buildRelayRouter(t, []struct {
			svtnID   [16]byte
			nodeAddr [8]byte
			ifaceID  routing.InterfaceID
		}{
			{svtnID, nodeAddrA, ifaceIDA}, // originator — excluded; channel state irrelevant
			{svtnID, nodeAddrB, ifaceIDB}, // fast target
			{svtnID, nodeAddrC, ifaceIDC}, // slow/full target — must NOT stall dispatch
		})

		ncA := makeRelayTestNodeConn(t, 4)
		ncB := makeRelayTestNodeConn(t, 4)
		// ncC: unbuffered channel with no reader — any blocking send would hang forever.
		ncC := makeRelayTestNodeConn(t, 0) // unbuffered

		var sendMap sync.Map
		sendMap.Store(ifaceIDA, ncA)
		sendMap.Store(ifaceIDB, ncB)
		sendMap.Store(ifaceIDC, ncC)

		decision := discovery.RouterIngestDecision{
			Accept:   true,
			Relay:    true,
			SVTNID:   svtnID,
			NodeAddr: nodeAddrA,
			Sequence: testSeq,
			Sessions: sessions,
		}

		// If relayDispatch blocks on ncC, the test goroutine hangs. We guard
		// with a timeout: if relayDispatch returns within the deadline, it is
		// non-blocking; if it does not return within the deadline, the test
		// fails via t.Fatal from the watchdog goroutine.
		done := make(chan struct{})
		go func() {
			relayDispatch(router, &sendMap, decision)
			close(done)
		}()
		select {
		case <-done:
			// returned promptly — non-blocking confirmed
		case <-time.After(2 * time.Second):
			t.Fatal("relayDispatch did not return within 2s — blocking send detected (PC-2 violation)")
		}

		// B must still receive despite C being full.
		got, ok := nonBlockingDrain(ncB)
		if !ok {
			t.Error("node B: expected frame despite node C's full channel, got nothing")
		} else if want := expectedFrame(t); !bytes.Equal(got, want) {
			t.Errorf("node B: frame mismatch: got %d bytes, want %d bytes", len(got), len(want))
		}

		// C: channel was full / unbuffered; drop is expected (no assertion — the
		// point is that dispatch continued, not that C got the frame).
		// A (originator): still excluded.
		mustNotReceive(t, ncA, "node A (originator)")
	})

	// --- subtest 4: missing sendMap entry — silent skip, no panic ---
	t.Run("missing_sendmap_entry_silent_skip", func(t *testing.T) {
		// Decision 2: TOCTOU window — missing sendMap entry (connection closed
		// between snapshot and send) is a silent skip. No panic. Remaining
		// targets still receive.
		//
		// Three IfaceIDs in identityIfaceMap; only two in sendMap.
		router := buildRelayRouter(t, []struct {
			svtnID   [16]byte
			nodeAddr [8]byte
			ifaceID  routing.InterfaceID
		}{
			{svtnID, nodeAddrA, ifaceIDA},
			{svtnID, nodeAddrB, ifaceIDB},
			{svtnID, nodeAddrC, ifaceIDC}, // bound in identity map but absent from sendMap
		})

		ncA := makeRelayTestNodeConn(t, 4)
		ncB := makeRelayTestNodeConn(t, 4)
		// ncC intentionally NOT stored in sendMap — simulates closed connection.
		var sendMap sync.Map
		sendMap.Store(ifaceIDA, ncA)
		sendMap.Store(ifaceIDB, ncB)
		// ifaceIDC absent — silent skip expected

		decision := discovery.RouterIngestDecision{
			Accept:   true,
			Relay:    true,
			SVTNID:   svtnID,
			NodeAddr: nodeAddrA,
			Sequence: testSeq,
			Sessions: sessions,
		}

		// Must not panic (no defer-recover needed; a panic propagates to t.Fail).
		relayDispatch(router, &sendMap, decision)

		// B must receive (remaining target).
		got, ok := nonBlockingDrain(ncB)
		if !ok {
			t.Error("node B: expected to receive relay frame despite missing sendMap entry for node C, got nothing")
		} else if want := expectedFrame(t); !bytes.Equal(got, want) {
			t.Errorf("node B: frame mismatch: got %d bytes, want %d bytes", len(got), len(want))
		}

		// A (originator) excluded.
		mustNotReceive(t, ncA, "node A (originator)")
	})

	// --- subtest 5: SVTN isolation — different SVTN does not receive ---
	t.Run("svtn_isolation_different_svtn_excluded", func(t *testing.T) {
		// PC-1: router iterates connections for the advertisement's SVTN only.
		// A node admitted to svtnID2 must NOT receive an advertisement on svtnID.
		svtnID2 := [16]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
		nodeAddrD := [8]byte{0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD}
		ifaceIDD := routing.InterfaceID(104)

		router := buildRelayRouter(t, []struct {
			svtnID   [16]byte
			nodeAddr [8]byte
			ifaceID  routing.InterfaceID
		}{
			{svtnID, nodeAddrA, ifaceIDA},
			{svtnID, nodeAddrB, ifaceIDB},
			{svtnID2, nodeAddrD, ifaceIDD}, // different SVTN
		})

		ncA := makeRelayTestNodeConn(t, 4)
		ncB := makeRelayTestNodeConn(t, 4)
		ncD := makeRelayTestNodeConn(t, 4)
		var sendMap sync.Map
		sendMap.Store(ifaceIDA, ncA)
		sendMap.Store(ifaceIDB, ncB)
		sendMap.Store(ifaceIDD, ncD)

		decision := discovery.RouterIngestDecision{
			Accept:   true,
			Relay:    true,
			SVTNID:   svtnID, // advertisement is on svtnID, not svtnID2
			NodeAddr: nodeAddrA,
			Sequence: testSeq,
			Sessions: sessions,
		}

		relayDispatch(router, &sendMap, decision)

		// B (same SVTN as originator, non-originator) must receive.
		got, ok := nonBlockingDrain(ncB)
		if !ok {
			t.Error("node B: expected to receive relay frame (same SVTN), got nothing")
		} else if want := expectedFrame(t); !bytes.Equal(got, want) {
			t.Errorf("node B: frame mismatch: got %d bytes, want %d bytes", len(got), len(want))
		}

		// A (originator) excluded.
		mustNotReceive(t, ncA, "node A (originator)")

		// D (different SVTN) must NOT receive.
		mustNotReceive(t, ncD, "node D (different SVTN — svtnID2, not svtnID)")
	})

	// --- subtest 6: frame identity — delivered bytes equal assembleDiscoveryRelayFrame output ---
	t.Run("frame_identity_equals_assembleDiscoveryRelayFrame", func(t *testing.T) {
		// AC-016 boundary / PC-4: the relay frame delivered to targets is NOT a
		// raw hop-1 retransmission — it is the DISCOVERY_RELAY re-serialized
		// form produced by assembleDiscoveryRelayFrame. Assert byte-equality.
		router := buildRelayRouter(t, []struct {
			svtnID   [16]byte
			nodeAddr [8]byte
			ifaceID  routing.InterfaceID
		}{
			{svtnID, nodeAddrA, ifaceIDA},
			{svtnID, nodeAddrB, ifaceIDB},
		})

		ncA := makeRelayTestNodeConn(t, 4)
		ncB := makeRelayTestNodeConn(t, 4)
		var sendMap sync.Map
		sendMap.Store(ifaceIDA, ncA)
		sendMap.Store(ifaceIDB, ncB)

		decision := discovery.RouterIngestDecision{
			Accept:   true,
			Relay:    true,
			SVTNID:   svtnID,
			NodeAddr: nodeAddrA,
			Sequence: testSeq,
			Sessions: sessions,
		}

		relayDispatch(router, &sendMap, decision)

		got, ok := nonBlockingDrain(ncB)
		if !ok {
			t.Fatal("node B: expected relay frame, got nothing")
		}
		// The expected frame is assembled independently using the same parameters
		// carried in the RouterIngestDecision — not a copy of the input bytes.
		want := assembleDiscoveryRelayFrame(decision.SVTNID, decision.NodeAddr, decision.Sequence, decision.Sessions)
		if !bytes.Equal(got, want) {
			t.Errorf("frame identity check failed: delivered %d bytes, assembleDiscoveryRelayFrame produced %d bytes", len(got), len(want))
		}
	})
}

// New imports added for AC-017 tests:
//   admission — used by buildRelayRouter (NewAdmittedKeySet, RegisterKey, RoleAccess)
//   routing   — used by buildRelayRouter (NewRouter, BindInterface, InterfaceID type)
//   sync      — used by makeRelayTestNodeConn (sync.Once) and sendMap (sync.Map)
//   time      — used by the best-effort non-blocking subtest watchdog (time.After)
// context and encoding/binary were already imported for AC-014-016 tests above.
