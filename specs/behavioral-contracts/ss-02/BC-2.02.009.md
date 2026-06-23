---
artifact_id: BC-2.02.009
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.02.009
subsystem: multipath-forwarding
architecture_module: internal/multipath
capability: CAP-010
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
  - '.factory/specs/domain-spec/edge-cases.md'
  - '.factory/specs/domain-spec/failure-modes.md'
  - '_bmad-output/planning-artifacts/prd.md'
traces_to: [CAP-010]
kos_anchors:
  - elem-node-router-architecture
---

# Behavioral Contract BC-2.02.009: Bounded Drop Cache Suppresses Looping Duplicate Frames by Checksum

## Description

Each router maintains a bounded LRU cache of recently-forwarded frame checksums. When a frame arrives whose checksum matches an entry in the cache, the frame is silently discarded as a loop duplicate. Retransmits generate new frames with different content and thus different checksums, so they pass through. This is the second-line loop prevention mechanism (after split-horizon, BC-2.02.008).

## Preconditions

1. The router has a drop cache initialized (bounded size, implementation: configurable, default 10,000 entries).
2. The frame has been verified (HMAC check passed) before checksum lookup.
3. A checksum function has been applied to the frame bytes (implementation: CRC32 or faster; not a security checksum — it is a duplicate-detection checksum only).

## Postconditions

1. On cache miss: frame is forwarded normally; checksum added to the drop cache.
2. On cache hit: frame is silently discarded; drop cache hit counter incremented (for operator diagnostics).
3. Cache entries age out via LRU eviction when the cache is full.
4. Retransmit frames (different content, same sequence) produce different checksums and are NOT suppressed.

## Invariants

1. **DI-009**: This mechanism implements the "prevent routing loops" aspect of DI-009's "retransmits produce different checksums and pass through" guarantee.
2. The drop cache does not persist across router restarts (in-memory only).
3. Drop cache operations do not block frame forwarding — cache lookup is O(1).

## Trigger

Frame received at router after HMAC verification.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 (DEC-009) | Drop cache is full; new checksum evicts old entry | Evicted checksum may no longer suppress a re-arriving old frame (acceptable — old frames arriving after eviction are harmless and will be deduplicated at the receiver). |
| EC-002 (FM-003) | Routing loop floods router faster than cache eviction | CPU load increases. Cache is bounded; excess duplicates are processed and added to cache, evicting older entries. Router operator alerted via drop cache hit rate metric. |
| EC-003 | Two different frames hash to the same checksum (collision) | Legitimate frame incorrectly suppressed. Probability negligible with 32-bit checksum at typical traffic rates. Logged as a potential collision event for investigation. |
| EC-004 | Router restart clears drop cache | Previously seen frames may briefly pass through if they re-arrive after restart. Receiver deduplication (BC-2.02.002) handles this. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Frame with checksum 0xABCD arrives; cache empty | Frame forwarded; checksum 0xABCD added to cache | happy-path |
| Same frame (checksum 0xABCD) arrives again | Frame dropped silently; hit counter incremented | happy-path |
| Retransmit: new content, same sequence → different checksum 0xEF01 | Frame forwarded; checksum 0xEF01 added to cache | happy-path |
| Cache full (10,000 entries); new frame arrives | LRU entry evicted; new checksum added; new frame forwarded | edge-case |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-025 | Cache hit suppresses frame; miss forwards frame | unit |
| VP-025 | Cache never grows beyond configured maximum | proptest |
| VP-025 | Retransmit (different content) always produces cache miss | unit |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-010 ("Router split-horizon and duplicate suppression") per capabilities.md §CAP-010 |
| L2 Domain Invariants | DI-009 (receiver deduplication: first arrival wins; retransmits produce different checksums) |
| Architecture Module | internal/multipath |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-010 ("Router split-horizon and duplicate suppression") per capabilities.md §CAP-010 — this BC specifies the checksum-based drop cache that CAP-010 defines as the "bounded drop cache of frame checksums" |

## Related BCs

- BC-2.02.008 — composes with: split-horizon is first-line; drop cache is second-line loop prevention
- BC-2.02.002 — related to: receiver deduplication handles cases drop cache misses
