// discovery_relay_wire.go — DISCOVERY_RELAY (control_type=0x03) hop-2 frame
// assembly for cmd/switchboard (S-BL.DISCOVERY-WIRE Task 5; AC-014, AC-015,
// AC-016).
//
// assembleDiscoveryRelayFrame is a pure function: no live connection or
// fan-out mechanism is needed to construct the frame bytes (AC-014
// postcondition 4). The relay-dispatch closure that fans the assembled
// frame out to admitted nodes (SVTN-scoped, exclude-originator,
// best-effort, SEC-DW-09 rate cap) is Task 6 — [GATED —
// depends_on S-BL.NODE-IDENTIFY-WIRE] — and is out of scope for this file
// until that companion story lands (see the story's Task 6 section).
//
// Purity classification (ARCH-09): pure-core — no I/O, deterministic
// serialization from decoded fields to bytes.
package main

import (
	"github.com/arcavenae/switchboard/internal/discovery"
)

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
// STUB — S-BL.DISCOVERY-WIRE (Red Gate, BC-5.38.001). Not yet implemented;
// body panics unconditionally so no test can accidentally pass before
// Task 5's Green step. No call site yet: this pure function is exercised
// directly by AC-014/AC-015/AC-016 unit tests (Task 5), not via any
// production call site — the relay-dispatch closure that would call it
// live is Task 6, GATED — depends_on S-BL.NODE-IDENTIFY-WIRE.
//
//nolint:unused // see doc comment above: exercised directly by tests, not wired yet
func assembleDiscoveryRelayFrame(svtnID [16]byte, nodeAddr [8]byte, sequence uint64, sessions []discovery.SessionPresence) []byte {
	panic("not implemented: S-BL.DISCOVERY-WIRE assembleDiscoveryRelayFrame")
}
