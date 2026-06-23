---
artifact_id: BC-2.01.005
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.01.005
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

# Behavioral Contract BC-2.01.005: Channel Header Is Opaque to Routers — Parseable Only by Endpoints

## Description

The channel header immediately follows the 44-byte outer header in every frame. It carries endpoint-visible state: channel ID, sequence number, sender timestamp, FEC metadata, and flags. Routers have no code path that reads or parses the channel header. Endpoints (access nodes and consoles) parse it for session management. This boundary enforces carrier-grade content separation at the protocol level.

## Preconditions

1. A frame has a valid 44-byte outer header (per BC-2.01.004).
2. The channel header follows the outer header in the byte stream.
3. The router's frame-processing code path terminates at byte offset 43 (end of outer header).

## Postconditions

1. The router forwards the frame based solely on outer header fields (SVTN ID, destination address, frame type).
2. The router does not read, log, modify, or cache any bytes at or after offset 44.
3. The receiving endpoint parses the channel header to extract: channel_id, sequence, timestamp, fec_meta, flags.
4. The channel header format may evolve via TLV extensions without requiring router upgrades.

## Invariants

1. **DI-001**: Router cannot read session content, which begins at the channel header level. Channel header is opaque to routers by design.
2. **DI-007**: Channel header TLV extensions do not require router upgrades (only outer header changes do).
3. There is no router API, diagnostic mode, or error path that exposes channel header contents.

## Trigger

Frame arrival at a router; frame arrival at an endpoint.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Channel header contains unknown TLV extension type | Endpoint skips the unknown TLV per TLV skip-unknown rule. Router is unaffected — it never reads the channel header. |
| EC-002 | Frame is smaller than outer_header + minimum channel header size | Endpoint returns E-PRT-003 (frame truncated). Router: if the outer header is complete, the router has already forwarded the frame — truncation detection is an endpoint responsibility. |
| EC-003 | Router diagnostic mode requested via sbctl | Router returns outer-header-level metadata only (source, destination, SVTN ID, frame type, frame count). Channel header contents are never returned. |
| EC-004 | Frame with channel header FEC field indicating parity frame | Endpoint identifies as parity frame via flags field; processes for FEC reconstruction. Router forwards identically. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Router receives frame; inspects forwarding | Router reads bytes 0–43 only; bytes 44+ untouched | happy-path |
| Endpoint receives frame; parses channel header at offset 44 | channel_id, sequence, timestamp, fec_meta, flags correctly extracted | happy-path |
| Channel header with unknown TLV type 0xFF | Endpoint skips TLV cleanly; no error; frame processed normally | edge-case |
| Router diagnostic query via `sbctl router frames --svtn=X` | Returns: src_addr, dst_addr, frame_type, frame_count, timestamp. No channel header fields. | happy-path |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-TBD | Router code contains no read access to frame bytes at offset ≥ 44 | code-audit / formal |
| VP-TBD | Channel header parses correctly for all valid field combinations | proptest |
| VP-TBD | Unknown TLV types in channel header are skipped without error | fuzz |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-003 ("Frame envelope encoding and decoding") per capabilities.md §CAP-003 |
| L2 Domain Invariants | DI-001 (carrier-grade content separation), DI-007 (outer header format stability) |
| Architecture Module | [filled by architect] |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-003 ("Frame envelope encoding and decoding") per capabilities.md §CAP-003 — this BC specifies the router/endpoint parsing boundary that CAP-003 defines as "Router parses outer; endpoints parse channel header" |

## Related BCs

- BC-2.01.004 — depends on: outer header must be valid before channel header is reached
- BC-2.05.005 — related to: HMAC in outer header authenticates the entire frame including the opaque channel header
