// discovery_relay_wire_test.go covers AC-014, AC-015, and AC-016 — the
// (not gated) DISCOVERY_RELAY hop-2 frame-assembly half of Task 5.
// AC-017/AC-018 (the fan-out/rate-cap half, GATED — depends_on
// S-BL.NODE-IDENTIFY-WIRE) are explicitly out of scope for this file.
package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"

	"github.com/arcavenae/switchboard/internal/discovery"
	"github.com/arcavenae/switchboard/internal/frame"
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
