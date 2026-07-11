---
artifact_id: BC-2.01.008
document_type: behavioral-contract
level: L3
version: "1.0"
status: draft
producer: product-owner
timestamp: 2026-07-11T00:00:00
phase: 1a
bc_id: BC-2.01.008
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
  - '.factory/specs/behavioral-contracts/ss-01/BC-2.01.004.md'
  - '.factory/decisions/S-7.04-FU-DRAIN-WIRE-placement-note.md'
traces_to: [CAP-003]
kos_anchors:
  - elem-ssh-end-to-end-encryption
---

# Behavioral Contract BC-2.01.008: Router-Terminated Control Frame Payload Schema (control_type Discriminator)

## Description

When a router receives a `ctl (frame_type = 0x03)` frame for which it is the terminal consumer (not a transit forwarder), it parses the first byte of the frame payload as a `control_type` discriminator to identify the control operation. This BC is the authoritative schema home for the `control_type` byte enumeration and the fixed-length control message layout. It was created as a consequence of the F-DW-SP1-005 adjudication that added a router-terminated-ctl carve-out to BC-2.01.004 Inv-2 and BC-2.01.005 PC-2.

## Preconditions

1. A frame has been received whose outer-header `frame_type` byte equals `0x03` (`ctl`).
2. The router is the addressed terminal consumer of this frame (i.e., the frame is directed at the router itself, not being forwarded to a downstream node).
3. The frame's `payload_len` field (outer header offset 2–3) is at least 4 bytes.

## Postconditions

1. The router reads `payload[0]` as the `control_type` discriminator byte.
2. Defined `control_type` opcodes:

   | control_type | Value | Defined by | Description |
   |-------------|-------|------------|-------------|
   | DRAIN       | 0x01  | S-7.04-FU-DRAIN-WIRE | Router is draining; connected node should migrate to alternate router |
   | RESYNC      | 0x02  | S-BL.RESYNC-FRAME (reserved, not yet dispatched) | Session resynchronization signal |
   | (unassigned) | 0x03–0xFF | future stories | MUST be silently ignored by all current receivers |

3. The full control message layout for all currently-defined opcodes is a fixed 4 bytes:

   | Offset | Size | Field         | Notes                                             |
   |--------|------|---------------|---------------------------------------------------|
   | 0      | 1    | control_type  | Opcode; see table above                           |
   | 1      | 1    | version       | Control message protocol version; `0x01` for v1  |
   | 2      | 2    | reserved      | Zero-filled; receiver MUST ignore                 |

4. A `control_type` value not listed in the defined table above MUST be silently ignored by the receiver — no error, no logging, no connection close (forward-compatibility rule FO-DRAIN-WIRE-001).

## Invariants

1. **DI-001 carve-out**: This schema only applies to frames where the router is the terminal consumer. Frames of `frame_type = ctl` that are being forwarded by a transit router remain unconditionally opaque — the transit router MUST NOT parse `payload[0]` for routing purposes.
2. **Schema growth is append-only**: New `control_type` opcodes are assigned sequentially (0x03, 0x04, …). Existing opcode values are never reassigned or reused.
3. **control_type=0x02 (RESYNC) is reserved but not dispatched** until S-BL.RESYNC-FRAME lands. Any receiver encountering 0x02 before that story MUST apply the silent-ignore rule (Postcondition 4).
4. **DI-007**: The 4-byte fixed layout at offset 0–3 is stable within major protocol version. Future control messages MAY extend beyond byte 3, but bytes 0–3 retain their meaning.

## Trigger

Receipt of a `ctl (0x03)` frame by its terminal router consumer.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | `control_type = 0xFF` (unrecognized) | Silently ignored per Postcondition 4; no error, no connection close |
| EC-002 | `payload_len` < 4 (truncated control message) | Router discards frame with E-PRT-002 "control frame truncated: expected ≥4 bytes payload, got N" |
| EC-003 | `control_type = 0x02 (RESYNC)` received before S-BL.RESYNC-FRAME is implemented | Silently ignored; treated as unrecognized opcode |
| EC-004 | `control_type = 0x01 (DRAIN)` received by a non-draining router (e.g., a node acting as PE upstream that receives a DRAIN signal meant for it) | Processed per BC-2.09.002: node migrates to alternate router |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| `ctl` frame, `control_type=0x01`, version=0x01, reserved=0x0000 | Router dispatches DRAIN operation | happy-path |
| `ctl` frame, `control_type=0xFF` (unrecognized) | Frame silently ignored; no error returned | edge-case |
| `ctl` frame, `payload_len=2` (truncated) | Returns E-PRT-002 "control frame truncated" | error |
| `ctl` frame, `control_type=0x02` (RESYNC, reserved) | Silently ignored until S-BL.RESYNC-FRAME dispatches it | edge-case |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-015 | This BC's carve-out does not affect VP-015: SVTNRoute in internal/routing remains payload-independent; control payload parsing occurs in cmd/switchboard post-routing | code-audit note |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-003 ("Frame envelope encoding and decoding") per capabilities.md §CAP-003 |
| L2 Domain Invariants | DI-001 (carrier-grade content separation — terminal-consumer carve-out), DI-007 (layout stability within major version) |
| Architecture Module | internal/frame (schema definition); cmd/switchboard (dispatch site) |
| Stories | S-7.04-FU-DRAIN-WIRE (first control_type opcode: DRAIN=0x01) |
| Capability Anchor Justification | CAP-003 ("Frame envelope encoding and decoding") per capabilities.md §CAP-003 — this BC defines the router-addressed control payload schema that is part of the wire format CAP-003 specifies; the control_type discriminator is a sub-field of the ctl frame's payload within the CAP-003 frame envelope |

## Related BCs

- BC-2.01.004 — authority for: outer header frame_type field; Inv-2 router-terminated-ctl carve-out references this BC as schema home
- BC-2.01.005 — carve-out context: PC-2 router-opacity rule; carve-out note references this BC
- BC-2.09.002 — first consumer: DRAIN opcode (control_type=0x01) defined and used by the drain signal mechanism

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-07-11 | Created. Schema home for ctl (0x03) control payload control_type discriminator. Consequence of F-DW-SP1-005 adjudication (router-terminated-ctl carve-out to DI-001 opacity invariant). Defines control_type=0x01 (DRAIN), reserves 0x02 (RESYNC). Forward-compat silent-ignore rule FO-DRAIN-WIRE-001 encoded as Postcondition 4 and Invariant 1. |
