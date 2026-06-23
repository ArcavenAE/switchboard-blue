---
artifact_id: BC-2.01.004
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
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
modified: []
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

Every Switchboard frame carries a 44-byte outer header in a fixed binary layout. This header contains the router-visible metadata: protocol version, frame type, SVTN ID, destination node address, source node address, payload length, and HMAC. Routers parse this header to make forwarding decisions without inspecting the payload. The layout is fixed within a major protocol version; any change requires a major version increment.

## Preconditions

1. The sending node has initialized an outer header struct with all required fields.
2. The SVTN ID is a valid 16-byte identifier.
3. The destination and source node addresses are 8-byte values derived as `hash(SVTN-ID || public-key)`.
4. The HMAC has been computed over the full frame (outer header fields + payload) using the node's admission key.

## Postconditions

1. The serialized outer header is exactly 44 bytes in big-endian byte order.
2. Field layout: [version: 2B][frame_type: 2B][svtn_id: 16B][dst_addr: 8B][src_addr: 8B][length: 4B][hmac: 16B] — total 44 bytes (note: HMAC is truncated to 16 bytes).
3. The deserialized outer header matches the serialized values exactly (round-trip identity).
4. A router receiving the frame can parse the outer header without reading any byte beyond offset 43.

## Invariants

1. **DI-007**: The 44-byte outer header layout is fixed within major protocol version. Field positions and sizes are immutable within v1.x.
2. **DI-001**: The outer header contains no session content. All bytes beyond offset 43 are opaque to routers.
3. Length field reflects the number of bytes in the payload (after the channel header), not the total frame size.

## Trigger

Frame assembly at the sending node before transmission.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Frame with version field indicating major version > current | Router rejects frame with E-PRT-001; does not forward. See DEC-008. |
| EC-002 | Frame with length field set to 0 (empty-tick frame) | Valid. Router forwards normally; receiver identifies as EMPTY_TICK via frame_type field. |
| EC-003 | Frame with invalid SVTN ID (all-zeros) | Router rejects with E-ADM-003 (SVTN ID not found in admitted set). |
| EC-004 | Serialization of outer header on a little-endian machine | Fields are written in network byte order (big-endian) regardless of host endianness. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| version=1, frame_type=DATA, svtn_id=16 random bytes, dst=8B, src=8B, length=256, hmac=16B | 44-byte serialized header; bytes 0-1 = 0x00,0x01; bytes 2-3 = 0x00,0x01 (DATA type) | happy-path |
| Deserialize 44-byte header | All fields parse to expected values; no out-of-bounds read | happy-path |
| 43-byte buffer passed to deserializer | Returns E-PRT-002 "header truncated: expected 44 bytes, got 43" | error |
| version=2 (unknown major) | Router returns E-PRT-001 "unsupported protocol version 2" | error |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-TBD | serialize(deserialize(x)) == x for all valid headers | proptest/fuzz |
| VP-TBD | Serialized outer header is always exactly 44 bytes | unit |
| VP-TBD | No field in outer header overlaps with another field | unit |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-003 ("Frame envelope encoding and decoding") per capabilities.md §CAP-003 |
| L2 Domain Invariants | DI-007 (outer header format stability within major version), DI-001 (carrier-grade content separation) |
| Architecture Module | [filled by architect] |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-003 ("Frame envelope encoding and decoding") per capabilities.md §CAP-003 — this BC specifies the exact 44-byte layout that CAP-003 defines as the router-visible/endpoint-only boundary |

## Related BCs

- BC-2.01.005 — composes with: channel header follows outer header in frame layout
- BC-2.05.005 — depends on: HMAC in outer header is what BC-2.05.005 verifies
