---
artifact_id: BC-2.03.001
document_type: behavioral-contract
level: L3
version: "1.5"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.03.001
subsystem: session-discovery
architecture_module: internal/discovery
capability: CAP-011
priority: P1
criticality: high
scope_phase: PE
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified:
  - date: 2026-07-13
    version: "1.5"
    change: >
      S-BL.DISCOVERY-WIRE-rulings.md v1.2 amendments executed: Precondition 3
      gains the concrete multicast-address derivation rule; Postcondition 1
      gains the router-mediated relay delivery-mechanism note; Postcondition 2
      gains a new monotonic `sequence` replay-resistance field + router-side
      discard rule (SEC-DW-07, cites VP-080); Postcondition 5's
      DRIFT-W6TBD-001 key-placeholder note is replaced with the concrete
      domain-separated `DiscoveryAuthKey` derivation rule (Ruling 1 v1.1) —
      resolves DRIFT-W6TBD-001's BC-side obligation. Invariant 1 (DI-004)
      reviewed, confirmed already-correct, no change.
deprecated: null
deprecated_by: null
replacement: null
retired: null
removed: null
removal_reason: null
inputDocuments:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/domain-spec/failure-modes.md'
  - '_bmad-output/planning-artifacts/prd.md'
traces_to: [CAP-011]
kos_anchors:
  - elem-node-router-architecture
---

# Behavioral Contract BC-2.03.001: Access Node Advertises Session Presence via SVTN-Scoped Multicast on State Change and Periodic Heartbeat

## Description

An access node broadcasts its presence and the state of its published sessions to all nodes on the SVTN via an SVTN-scoped multicast address. Advertisements are triggered by: (1) session state changes (new session, session closed, attachment status change), (2) periodic heartbeat (configurable interval, default 30 seconds), and (3) on-demand when a console sends a presence request. This enables consoles to discover sessions without hostnames or IP addresses.

**Implementation scope note (Ruling W6TB-D):** BC-2.03.001 covers the advertisement trigger model and payload semantics. The wire transport (UDP multicast socket, SVTN-scoped multicast address allocation, admitted-node HMAC key derivation) is split to S-BL.DISCOVERY-WIRE. The implementing story S-7.02 delivers the in-process registry model; PC-1 (multicast to all admitted nodes), PC-3 (1-tick network delivery), and PC-4 (network heartbeat broadcast) are fully verified only after S-BL.DISCOVERY-WIRE ships.

## Preconditions

1. The access node is admitted to an SVTN (Tier 1 admission complete).
2. The access node has at least one published session.
3. A SVTN-scoped multicast address is allocated for the SVTN's discovery channel.

   > **Derivation rule (Ruling S-BL.DISCOVERY-WIRE-2):** the multicast address
   > is `239.h0.h1.h2` where `h0..h2` are the first three bytes of
   > SHA-256(svtnID) — deterministic, static, requiring no allocation
   > bookkeeping. Only the router-mode daemon joins this multicast group.

## Postconditions

1. The advertisement is multicast to all admitted nodes on the SVTN.

   > **Delivery mechanism note (Ruling S-BL.DISCOVERY-WIRE-2):** "multicast"
   > here denotes SVTN-wide fan-out semantics, not direct peer-to-peer IP
   > multicast. Delivery is router-mediated: the access node sends one UDP
   > datagram to the SVTN-scoped multicast address (received only by the
   > router); the router authenticates it and relays it to each admitted node
   > over that node's own connection. This satisfies DI-004 (no direct
   > node-to-node communication) and DI-006 (HMAC verified at first router).
   > The `239.0.0.0/8` range is addressing hygiene, not a security boundary —
   > HMAC authentication remains the sole security boundary regardless of the
   > actual multicast-routing scope realized in a given deployment
   > (SEC-DW-08).
2. Each advertisement includes: access node address, a monotonic sequence value, list of session names, per-session attachment status, per-session quality indicator.

   > **Replay-resistance field (SEC-DW-07, Ruling S-BL.DISCOVERY-WIRE-1):**
   > each advertisement additionally includes a monotonic `sequence` value,
   > unique-and-increasing per (access node, SVTN), incremented on every
   > outbound advertisement (state-change or heartbeat-triggered). The router
   > discards any HMAC-verified advertisement whose `sequence` is not strictly
   > greater than the last-accepted value for that (SVTN, node) pair, even
   > though HMAC passed — closing the replay window that would otherwise let
   > a captured, still-valid frame be re-injected indefinitely and defeat
   > BC-2.03.002 Postcondition 5's staleness-expiry guarantee. Cold-start
   > (router restart, or first frame from a newly-admitted node) accepts
   > unconditionally, bounding the residual replay window to at most one
   > heartbeat interval — the same bounded-not-perfect posture this project
   > already accepts for admission-layer nonce replay (`nonceTTL=60s`).
   > Verified by VP-080.
3. On state change (session added/removed/attached/detached): advertisement sent within 1 tick interval.
4. On periodic heartbeat: advertisement sent every 30 seconds regardless of state change. **Observability gate (Ruling W6TB-D):** in the registry model (S-7.02), the periodic heartbeat timer fires an observable side effect verifiable by injecting a tick and asserting a heartbeat counter increments. Network dispatch to wire is deferred to S-BL.DISCOVERY-WIRE.
5. Advertisement is authenticated (HMAC in outer header) so receivers can verify it is from an admitted node.

   > **Key derivation (Ruling S-BL.DISCOVERY-WIRE-1, v1.1):** The HMAC key
   > authenticating an advertisement frame is `DiscoveryAuthKey :=
   > hmac.DeriveDiscoveryKey(nodeAdmissionPubkey, svtnID)` — HKDF-SHA256 over
   > the same `(nodeAdmissionPubkey, svtnID)` inputs `internal/admission`
   > already uses for the session-data `frame_auth_key` (ADR-001), but with a
   > distinct info label (`HKDFInfoDiscovery = "switchboard-discovery-auth"`,
   > vs. the existing `HKDFInfo = "switchboard-frame-auth"`) so the two keys
   > are cryptographically independent — a compromise of one does not imply
   > the other. No new KDF primitive is introduced; `hkdfSHA256` (the
   > underlying HKDF implementation) is unchanged and shared by both call
   > sites. The key is verified exclusively by the router that receives the
   > advertisement off the discovery multicast socket (the "first router" per
   > DI-006), using fixed-offset extraction of only the key-selector fields
   > (`SVTNID`, `NodeAddr`) before any variable-length body content is parsed
   > (SEC-DW-01) — access and console nodes never independently look up or
   > re-verify another node's `DiscoveryAuthKey`; they receive
   > already-authenticated advertisements via the router's relay over their
   > own admitted connection. The router additionally enforces a
   > per-(SVTN,NodeAddr) monotonic sequence check to reject replayed frames
   > (SEC-DW-07 — see Postcondition 2's replay-resistance field note above).

## Invariants

1. **DI-004**: Advertisements flow node-to-router-to-node via the SVTN; no direct node-to-node multicast.
2. **DI-005**: Advertisements are SVTN-scoped — nodes on SVTN-B do not receive advertisements from SVTN-A.
3. The access node does not advertise session content, only metadata (name, status, quality indicator).

## Trigger

Session state change; periodic heartbeat timer fires; console sends on-demand presence request.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 (FM-005) | Heartbeat advertisement lost in transit | Next heartbeat (30s later) or next state change will refresh. Consoles may show stale data for one heartbeat interval — acceptable per FM-005. |
| EC-002 (DEC-014) | tmux session closes while advertisement is in flight | Access node sends a session-removed advertisement on the next state change event. |
| EC-003 | Access node loses SVTN connection briefly | Reconnects and sends a full-state advertisement on reconnection to resync all consoles. |
| EC-004 | Access node has 100 sessions | Single advertisement frame may need to be fragmented or multiple frames sent per heartbeat (implementation decision: max sessions per frame is architecture concern). |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| New tmux session "agent-01" created | Advertisement multicast within 1 tick: {node_addr, sessions: [{name:"agent-01", attached:false, quality:green}]} | happy-path |
| Session "agent-01" attached by console | Advertisement multicast: {sessions: [{name:"agent-01", attached:true, quality:green}]} | happy-path |
| 30 seconds pass with no state change | Periodic heartbeat advertisement multicast | happy-path |
| Console sends presence request | Access node responds with current full-state advertisement | happy-path |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-044 | Advertisement sent within 1 tick of state change | integration |
| VP-044 | Advertisement contains no session content — metadata only | code-audit |
| VP-044 | Advertisement is SVTN-scoped: received only by admitted nodes | integration |
| VP-080 | Router-side discovery ingest discards an HMAC-valid advertisement whose sequence is not strictly increasing (replay rejection) | integration |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-011 ("Multicast presence advertisement") per capabilities.md §CAP-011 |
| L2 Domain Invariants | DI-004 (no direct node-to-node), DI-005 (SVTN cryptographic isolation) |
| Architecture Module | internal/discovery |
| Stories | S-7.02, S-BL.DISCOVERY-WIRE (deferred: PC-1/PC-3/PC-4 wire delivery) |
| Capability Anchor Justification | CAP-011 ("Multicast presence advertisement") per capabilities.md §CAP-011 — this BC specifies the advertisement trigger conditions and payload that CAP-011 defines as "state change, periodic heartbeat, and on-demand request" |

## Related BCs

- BC-2.03.002 — composes with: console discovery depends on these advertisements
- BC-2.03.003 — composes with: advertisement payload structure defined in BC-2.03.003

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.5 | 2026-07-13 | `S-BL.DISCOVERY-WIRE-rulings.md` v1.2 amendments executed (Ruling 1 + Ruling 2, resolves DRIFT-W6TBD-001's BC-side obligation): Precondition 3 gains the concrete multicast-address derivation rule (`239.h0.h1.h2` = first 3 bytes of SHA-256(svtnID); router-only group membership). Postcondition 1 gains the router-mediated relay delivery-mechanism note (multicast denotes SVTN-wide fan-out, not direct peer-to-peer IP multicast; HMAC is the sole security boundary per SEC-DW-08). Postcondition 2 gains a NEW monotonic `sequence` replay-resistance field in the field list plus the router-side non-increasing discard rule (SEC-DW-07); cites VP-080 (minted this session, `status: draft`). Postcondition 5's DRIFT-W6TBD-001 `svtnID`-as-key placeholder note is replaced with the concrete domain-separated `DiscoveryAuthKey := hmac.DeriveDiscoveryKey(nodeAdmissionPubkey, svtnID)` derivation rule (Ruling 1 v1.1, SEC-DW-06). Invariant 1 (DI-004) reviewed against Ruling 2's finding and confirmed already-correct — no change. Verification Properties table gains a VP-080 row (replay-rejection, integration) alongside the existing VP-044 rows. |
| v1.4 | 2026-07-01 | Pass-2 L3 fix-burst (RULING-W6TB-D bidirectional-trace closure): Stories row updated to add S-BL.DISCOVERY-WIRE with deferred PC-1/PC-3/PC-4 wire delivery annotation. |
| v1.3 | 2026-07-01 | S-7.02 LENS-3 traceability backfill (RULING-W6TB-D): Traceability.Stories row filled with S-7.02. |
| v1.2 | 2026-07-01 | Ruling W6TB-D: scope split annotation added. PC-1 wire transport, PC-4 network dispatch, and admitted-node HMAC key vocabulary (DRIFT-W6TBD-001) deferred to S-BL.DISCOVERY-WIRE. Observability gate added to PC-4: heartbeat timer observable via injected counter. PC-5 key placeholder note added. |
| v1.1 | 2026-06-23 | Initial behavioral contract creation. |
