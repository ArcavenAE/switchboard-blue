---
artifact_id: BC-2.02.007
document_type: behavioral-contract
level: L3
version: "1.2"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.02.007
subsystem: multipath-forwarding
architecture_module: internal/arq
capability: CAP-009
priority: P1
criticality: high
scope_phase: PE
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified:
  - 2026-07-01T00:00:00 # v1.2 — PC-5 wire vocabulary corrected: FRAME_TYPE=PARITY → frame_type=fec=0x05 (canonical enum value per F-P8-008 Phase 1 drift fix; aligns with S-7.01 AC-001 and story Architecture Compliance Rules)
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
traces_to: [CAP-009]
kos_anchors:
  - elem-dual-fastest-path-forwarding
---

# Behavioral Contract BC-2.02.007: XOR Parity FEC Covers Frame Groups; Single Loss in Group Recoverable Without Retransmit

## Description

In multi-path topologies (PE router phase), the downstream half-channel optionally uses XOR parity forward error correction. A parity frame is computed over a group of N data frames using bitwise XOR. If exactly one data frame in the group is lost, the receiver can reconstruct it from the parity frame and the remaining N-1 data frames, without issuing a retransmit request. This reduces latency on lossy paths by eliminating one ARQ round-trip for single-frame losses.

## Preconditions

1. The node is in PE router phase (multi-hop topology active).
2. FEC is enabled via configuration (disabled by default in E router phase).
3. The FEC group size is configured (implementation: N=4 data frames per parity frame).

## Postconditions

1. After every N data frames, a parity frame is emitted carrying XOR of those N frames' payloads.
2. If 0 data frames are lost: receiver delivers all N frames normally; parity frame provides redundant verification.
3. If exactly 1 data frame is lost: receiver reconstructs the missing frame from the parity frame and the other N-1 frames; delivers all N frames in order.
4. If 2+ data frames are lost in one group: FEC recovery fails; ARQ retransmit is triggered for the missing frames (falls back to BC-2.02.005).
5. The parity frame carries `frame_type=fec=0x05` in the channel header so routers and receivers distinguish it from data frames (canonical enum value; `FRAME_TYPE=PARITY` is a retired alias — do not use).

## Invariants

1. XOR parity is payload-only — the outer and channel headers of the parity frame are not XORed with data headers.
2. FEC group size N is fixed per channel for the channel lifetime; changes require a new channel.
3. FEC does not replace ARQ — it supplements it. ARQ remains the fallback for multi-frame loss.

## Trigger

Nth data frame dispatched on downstream half-channel; parity frame computed and emitted.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Last group is incomplete (session ends mid-group) | Parity frame is not emitted for incomplete groups. ARQ handles any loss in the final partial group. |
| EC-002 | Two data frames in the same FEC group are lost | FEC recovery fails; ARQ retransmits both; session continues with added latency. |
| EC-003 | Parity frame is lost | No recovery impact — parity frame is only needed if a data frame is also lost. If both parity and one data frame lost, ARQ recovers the data frame. |
| EC-004 | FEC enabled in E router phase (single-path) | Configuration valid but FEC provides no benefit on single-path. Parity frames are generated and transmitted; the receiver holds them in case of loss recovery (unlikely but harmless). |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| 4 data frames [D1,D2,D3,D4]; all arrive | Parity frame P verified against D1⊕D2⊕D3⊕D4; all frames delivered | happy-path |
| 4 data frames; D2 lost; P arrives | D2 = P⊕D1⊕D3⊕D4 reconstructed; all 4 delivered in order | edge-case |
| 4 data frames; D2 and D3 lost; P arrives | FEC fails; ARQ triggered for D2 and D3 | edge-case |
| Parity frame lost; all data frames arrive | All 4 delivered normally; no recovery path needed | happy-path |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-043 | Single lost frame in FEC group reconstructed correctly | proptest |
| VP-043 | XOR parity: P = D1⊕D2⊕...⊕DN | unit |
| VP-043 | Double loss falls back to ARQ with correct SACK reporting | integration |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-009 ("XOR parity FEC for burst-loss recovery") per capabilities.md §CAP-009 |
| L2 Domain Invariants | DI-001 (carrier-grade content separation — FEC operates on encrypted payload) |
| Architecture Module | internal/arq |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-009 ("XOR parity FEC for burst-loss recovery") per capabilities.md §CAP-009 — this BC is the complete behavioral specification of the XOR FEC strategy for single-loss recovery |

## Related BCs

- BC-2.02.005 — depends on: ARQ is the fallback when FEC recovery fails
