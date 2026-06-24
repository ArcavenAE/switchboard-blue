---
artifact_id: BC-2.01.004
document_type: behavioral-contract
level: L3
version: "1.2"
status: draft
producer: architect
timestamp: 2026-06-24T00:00:00
phase: 1a
bc_id: BC-2.01.004
subsystem: session-networking
architecture_module: internal/frame
capability: CAP-003
priority: P0
criticality: critical
scope_phase: E
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified:
  - version: "1.2"
    date: 2026-06-24
    author: architect
    change: "Invariant 3 corrected to match ARCH-02 normative payload_len definition (= channel header + application payload after the outer header). Resolves consistency F-006 (refs drbothen/vsdd-factory#260)."
deprecated: null
deprecated_by: null
replacement: null
retired: null
removed: null
removal_reason: null
inputDocuments:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '_bmad-output/planning-artifacts/prd.md'
traces_to: [CAP-003]
kos_anchors:
  - elem-ssh-end-to-end-encryption
---

# Behavioral Contract BC-2.01.004: Frame Outer-Header Encoding and Decoding at 44-Byte Fixed Layout

## Description

Every Switchboard frame carries a 44-byte outer header in a fixed binary layout. This header contains the router-visible metadata: protocol version, frame type, SVTN ID, source node address, destination node address, payload length, and an 8-byte HMAC tag (first 8 bytes of HMAC-SHA256). Routers parse this header to make forwarding decisions without inspecting the payload. The layout is fixed within a major protocol version; any change requires a major version increment.

## Preconditions

1. The sending node has initialized an outer header struct with all required fields.
2. The SVTN ID is a valid 16-byte identifier.
3. The destination and source node addresses are 8-byte values derived as `hash(SVTN-ID || public-key)`.
4. The HMAC tag (8 bytes, first 8 bytes of HMAC-SHA256 output) has been computed over the full frame using the node's frame_auth_key (per-node-per-SVTN HKDF derivation).

## Postconditions

1. The serialized outer header is exactly 44 bytes in big-endian byte order.
2. Outer header layout (44 bytes total):

   | Offset | Size | Field          | Notes                                                       |
   |--------|------|----------------|-------------------------------------------------------------|
   | 0      | 1    | version        | bits[7:4]=major, bits[3:0]=minor; v0.1 = 0x01               |
   | 1      | 1    | frame_type     | u8 enum: data=0x01, empty_tick=0x02, ctl=0x03, arq=0x04, fec=0x05 |
   | 2      | 2    | payload_len    | u16 big-endian                                              |
   | 4      | 16   | svtn_id        | 128-bit SVTN identifier                                     |
   | 20     | 8    | src_node_addr  | 64-bit                                                      |
   | 28     | 8    | dst_node_addr  | 64-bit                                                      |
   | 36     | 8    | hmac_tag       | first 8 bytes of HMAC-SHA256(frame_auth_key, full_frame)    |

3. The deserialized outer header matches the serialized values exactly (round-trip identity).
4. A router receiving the frame can parse the outer header without reading any byte beyond offset 43.

## Invariants

1. **DI-007**: The 44-byte outer header layout is fixed within major protocol version. Field positions and sizes are immutable within v1.x.
2. **DI-001**: The outer header contains no session content. All bytes beyond offset 43 are opaque to routers.
3. Length field reflects the number of bytes after the outer header (channel header + application payload), not counting the 44-byte outer header itself.

## Trigger

Frame assembly at the sending node before transmission.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Frame with version byte whose major nibble (bits 7–4) is greater than current major version (e.g., byte `0x20` = major=2, minor=0) | Router rejects frame with E-PRT-001; does not forward. See DEC-008. |
| EC-002 | Frame with length field set to 0 (empty-tick frame) | Valid. Router forwards normally; receiver identifies as EMPTY_TICK via frame_type field. |
| EC-003 | Frame with invalid SVTN ID (all-zeros) | Router rejects with E-ADM-003 (SVTN ID not found in admitted set). |
| EC-004 | Serialization of outer header on a little-endian machine | Fields are written in network byte order (big-endian) regardless of host endianness. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| version=0x01 (v0.1), frame_type=DATA (0x01), svtn_id=16 random bytes, src=8B, dst=8B, payload_len=256, hmac_tag=8B | 44-byte serialized header; byte 0 = 0x01 (version); byte 1 = 0x01 (frame_type); bytes 2-3 = 0x01,0x00 (payload_len=256 big-endian) | happy-path |
| Deserialize 44-byte header | All fields parse to expected values; no out-of-bounds read | happy-path |
| 43-byte buffer passed to deserializer | Returns E-PRT-002 "header truncated: expected 44 bytes, got 43" | error |
| version=0x20 (major=2, minor=0; unknown major) | Router returns E-PRT-001 "unsupported protocol version 2.0" | error |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-001, VP-002, VP-003 | serialize(deserialize(x)) == x for all valid headers | proptest/fuzz |
| VP-001, VP-002, VP-003 | Serialized outer header is always exactly 44 bytes | unit |
| VP-001, VP-002, VP-003 | No field in outer header overlaps with another field | unit |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-003 ("Frame envelope encoding and decoding") per capabilities.md §CAP-003 |
| L2 Domain Invariants | DI-007 (outer header format stability within major version), DI-001 (carrier-grade content separation) |
| Architecture Module | internal/frame |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-003 ("Frame envelope encoding and decoding") per capabilities.md §CAP-003 — this BC specifies the exact 44-byte layout that CAP-003 defines as the router-visible/endpoint-only boundary |

## Related BCs

- BC-2.01.005 — composes with: channel header follows outer header in frame layout
- BC-2.05.005 — depends on: HMAC in outer header is what BC-2.05.005 verifies
