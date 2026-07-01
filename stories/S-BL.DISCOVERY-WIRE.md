---
artifact_id: S-BL.DISCOVERY-WIRE
document_type: story
level: ops
story_id: S-BL.DISCOVERY-WIRE
version: "1.0"
title: "Discovery wire boundary: UDP multicast I/O, admitted-node HMAC keys, multicast address allocation"
status: backlog
producer: product-owner
timestamp: 2026-07-01T00:00:00
modified: 2026-07-01T00:00:00
phase: 2
epic: E-7
wave: backlog
priority: P1
scope_phase: PE
estimated_points: 8
bc_traces:
  - BC-2.03.001
  - BC-2.03.002
vp_traces: [VP-044, VP-045]
subsystems: [session-discovery]
architecture_modules: [internal/discovery]
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: [S-7.02, S-2.02]
blocks: []
changed_by_rulings: [RULING-W6TB-D]
acceptance_criteria_count: 0
---

# S-BL.DISCOVERY-WIRE: Discovery Wire Boundary — UDP Multicast I/O

> **Status:** Backlog stub. Full story decomposition required at scheduling
> time. A product-owner pass is needed to specify admitted-node HMAC key
> derivation (DRIFT-W6TBD-001) before implementer pickup.

## Context

Ruling W6TB-D (`.factory/decisions/RULING-W6TB-D-discovery-scope.md`)
established that S-7.02 delivers an in-process registry model for session
discovery. Real multicast wire I/O and admitted-node HMAC key derivation
are deferred to this story.

## Open Design Obligations (must be resolved before scheduling)

### 1. Admitted-node HMAC key vocabulary (DRIFT-W6TBD-001)

`advertisementKey(svtnID) = svtnID` in the S-7.02 implementation is a
scoping placeholder. The SVTN ID is not admitted-node-scoped secret
material — any observer of one advertisement can compute the key.

Product-owner must specify:
- How admitted-node HMAC key material is derived at Tier-1 admission (S-2.02 layer).
- Where the key is stored (in memory on the node, derived from the admission keypair?).
- BC-2.03.001 PC-5 must be amended with the concrete key derivation rule.

### 2. SVTN-scoped multicast address derivation

BC-2.03.001 PC-1 requires "a SVTN-scoped multicast address is allocated
for the SVTN's discovery channel." No architecture doc specifies:
- How the multicast group address is derived from the SVTN ID.
- The IPv6/IPv4 multicast range in use.
- Whether the address is static per SVTN or allocated dynamically.

An ADR or ARCH-03 amendment is required before implementation.

## Scope (at scheduling time)

1. Specify and implement admitted-node HMAC key derivation for advertisement
   authentication; amend BC-2.03.001 PC-5.
2. Specify SVTN-scoped multicast address derivation from SVTN ID; amend
   ARCH-03 and BC-2.03.001 PC-1.
3. Replace in-process `Advertise`/`ReceiveAdvertisement` registry paths
   in `internal/discovery` with `net.ListenMulticastUDP` dispatch goroutine.
4. Wire `Run()` heartbeat body to call `Advertise` (actual network dispatch)
   every 30 s — fully satisfying BC-2.03.001 PC-4 at the network layer.
5. Integration tests: VP-044 verified over real UDP multicast (loopback);
   VP-045 verified with at least two real UDP sockets on loopback.
6. Verify BC-2.03.001 PC-1 (advertisement reaches all admitted nodes on
   SVTN) and PC-3 (1-tick network delivery) with integration harness.

## Estimated Points

5–8 (TBD pending admitted-node key vocabulary complexity).

## Changelog

| Version | Date | Change |
|---------|------|--------|
| v1.0 | 2026-07-01 | Backlog stub created per Ruling W6TB-D. Full decomposition deferred to Wave-7 planning. |
