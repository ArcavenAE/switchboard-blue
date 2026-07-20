// discovery_relay_wire.go — DISCOVERY_RELAY (control_type=0x03) hop-2 frame
// assembly and fan-out dispatch for cmd/switchboard (S-BL.DISCOVERY-WIRE
// Task 5: AC-014, AC-015, AC-016; Task 6b: AC-017).
//
// assembleDiscoveryRelayFrame is a pure function: no live connection or
// fan-out mechanism is needed to construct the frame bytes (AC-014
// postcondition 4).
//
// relayDispatch fans the assembled frame out to admitted nodes for the
// advertisement's SVTN, excluding the originating NodeAddr (AC-017).
// It is best-effort and non-blocking. Task 6d wires it into runRouter
// behind the decision.Relay gate — this helper itself does not check
// decision.Relay; it dispatches unconditionally (the caller gates).
//
// Purity classification (ARCH-09): assembleDiscoveryRelayFrame is pure-core
// (no I/O, deterministic serialization). relayDispatch is effectful (fan-out
// I/O to live connections).
package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"sync"

	"github.com/arcavenae/switchboard/internal/discovery"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/routing"
)

// discoveryRelayControlType identifies the DISCOVERY_RELAY control frame
// within FrameTypeCtl's payload discriminator byte (Decision 3(c)).
const discoveryRelayControlType = 0x03

// assembleDiscoveryRelayFrame builds the DISCOVERY_RELAY (control_type=0x03)
// outer frame bytes for one accepted advertisement (S-BL.DISCOVERY-WIRE
// Decision 3(c)).
//
// svtnID becomes the relay frame's own OuterHeader.SVTNID — SVTNID is
// deliberately NOT repeated inside the payload (Decision 3(c)). nodeAddr is
// the ORIGINATING access node's 8-byte address; sequence is the same uint64
// (epoch-qualified, F-DWSP4-001) hop-1 accepted; sessions is the
// per-session list to re-serialize (AC-016 — never a raw retransmission of
// hop-1's UDP bytes; hop-1's HMAC tag never appears in the relay frame).
//
// The returned frame's OuterHeader.HMACTag is the zero value (AC-015) —
// hop-2's trust boundary is the admitted TCP connection, not a per-frame
// HMAC, matching the S-7.04-FU-DRAIN-WIRE DRAIN precedent exactly.
//
// AC-014 / BC-2.01.008 Postcondition 2 (registry row, already landed v1.2),
// Postcondition 3 + Invariant 5/DI-007; BC-2.03.001 Postcondition 5.
//
// sessions must already satisfy discovery.EncodeSessionList's constraints
// (valid, non-empty UTF-8 names; count within the wire format's uint16
// bound) — by the time a session reaches this function it has already
// passed RouterIngest.Ingest's DecodeSessionList validation on the hop-1
// path that accepted it (AC-005..AC-013), so a violation here indicates a
// caller precondition bug, not a runtime/network condition. This function
// has no error return (its callers, including the Task 6 relay-dispatch
// closure this story gates, treat frame assembly as infallible for
// already-accepted sessions), so a violation panics rather than silently
// producing a malformed frame.
//
// The assembled payload's TOTAL serialized byte size (4-byte control header
// + 8-byte NodeAddr + 8-byte Sequence + session bytes) must also not exceed
// math.MaxUint16 (65535) — the wire size of OuterHeader.PayloadLen. This is
// currently unreachable in practice (sessions only ever arrive from a
// hop-1 datagram already bounded by discovery.MaxDiscoveryDatagramSize=
// 32768, and re-encoding here never expands that), but is checked
// explicitly rather than left as a silent uint16 truncation of PayloadLen —
// the same "shipped undetected" class of wire-field-truncation defect
// F-DWIP1-001 found on the hop-1 side.
//
// Wired live as of Task 6d: assembleDiscoveryRelayFrame is called by
// relayDispatch, which is called by the onRelay closure in runRouter behind
// the decision.Relay gate, reached via wireDiscoveryListener. Also exercised
// directly by discovery_relay_wire_test.go's AC-014/AC-015/AC-016/AC-017 tests.
func assembleDiscoveryRelayFrame(svtnID [16]byte, nodeAddr [8]byte, sequence uint64, sessions []discovery.SessionPresence) []byte {
	sessionBytes, err := discovery.EncodeSessionList(sessions)
	if err != nil {
		panic(fmt.Sprintf("assembleDiscoveryRelayFrame: EncodeSessionList: %v (caller precondition violated: sessions must already be valid by the time they reach relay assembly)", err))
	}

	payload := make([]byte, 0, 4+8+8+len(sessionBytes))
	payload = append(payload, discoveryRelayControlType, frame.VersionByte, 0x00, 0x00)
	payload = append(payload, nodeAddr[:]...)
	payload = binary.BigEndian.AppendUint64(payload, sequence)
	payload = append(payload, sessionBytes...) // EncodeSessionList already prefixes the BE uint16 count (payload[20:22])

	// Guard against silent PayloadLen truncation (F-DWIP4-N1): PayloadLen is
	// a uint16 wire field; without this check an oversized payload would
	// wrap silently in the uint16(len(payload)) conversion below rather than
	// failing loudly. See the doc comment above for why this is currently
	// unreachable but checked anyway.
	if len(payload) > math.MaxUint16 {
		panic(fmt.Sprintf("assembleDiscoveryRelayFrame: payload is %d bytes, exceeds the %d-byte maximum the uint16 PayloadLen wire field can represent (caller precondition violated: sessions must already be bounded by discovery.MaxDiscoveryDatagramSize by the time they reach relay assembly)", len(payload), math.MaxUint16))
	}

	// HMACTag is deliberately left as the zero value (AC-015): hop-2's trust
	// boundary is the admitted TCP connection this frame is sent over, not
	// a per-frame HMAC — matching the S-7.04-FU-DRAIN-WIRE DRAIN precedent
	// exactly (mgmt_wire.go's drainCoord.RegisterObserver closure).
	ehdr := frame.EncodeOuterHeader(frame.OuterHeader{
		Version:    frame.VersionByte,
		FrameType:  frame.FrameTypeCtl,
		SVTNID:     svtnID,
		PayloadLen: uint16(len(payload)),
	})

	raw := make([]byte, 0, len(ehdr)+len(payload))
	raw = append(raw, ehdr[:]...)
	raw = append(raw, payload...)
	return raw
}

// relayDispatch fans out a DISCOVERY_RELAY frame to every node admitted to
// decision.SVTNID except the originating node (decision.NodeAddr).
//
// It resolves target interfaces via Router.InterfacesForSVTN (SVTN-scoped,
// exclude-originator, fresh snapshot under RLock), assembles the relay frame
// once, then attempts a best-effort non-blocking send on each target's
// nodeConn.send channel. If a sendMap entry is missing (connection closed
// between snapshot and send), the target is silently skipped — no log, no
// counter, no error (Decision 2 / TOCTOU window). If a send channel is full,
// the frame is dropped silently (Decision 3 / best-effort).
//
// relayDispatch does NOT check decision.Relay — it dispatches
// unconditionally. The caller is responsible for gating on decision.Relay
// before calling this function (Task 6d).
//
// Matches the DRAIN observer fan-out shape at mgmt_wire.go:842–849.
// AC-017 / BC-2.03.001 Postcondition 1 delivery-mechanism note;
// fanout-resolution-ruling.md v1.0 Decisions 1/2/3.
func relayDispatch(router *routing.Router, sendMap *sync.Map, decision discovery.RouterIngestDecision) {
	ifaceIDs := router.InterfacesForSVTN(decision.SVTNID, decision.NodeAddr)
	relayFrame := assembleDiscoveryRelayFrame(decision.SVTNID, decision.NodeAddr, decision.Sequence, decision.Sessions)
	for _, ifaceID := range ifaceIDs {
		val, ok := sendMap.Load(ifaceID)
		if !ok {
			continue
		}
		nc := val.(*nodeConn)
		select {
		case nc.send <- relayFrame:
		default:
		}
	}
}
