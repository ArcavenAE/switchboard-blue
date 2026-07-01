---
artifact_id: RULING-W6TB-D-discovery-scope
document_type: decision
level: ops
version: "1.0"
status: final
producer: product-owner
timestamp: 2026-07-01T00:00:00
updated: 2026-07-01T00:00:00
cycle: v1.0.0-greenfield
stories_in_scope: [S-7.02, S-BL.DISCOVERY-WIRE]
closes_findings: [F-S7.02-P1L1-C1, F-S7.02-P1L1-C2, F-S7.02-P1L1-C3]
---

# Ruling W6TB-D — S-7.02 Session Discovery Scope

**Question:** do the three HIGH findings from S-7.02 Pass-1 LENS-1 — (C-1)
no-op heartbeat body, (C-2) no multicast I/O, (C-3) HMAC key equals SVTN ID
— require rebuilding S-7.02 to implement real multicast wire I/O, or is
in-memory registry scope correct for Wave 6 Tranche B?

---

## Decision

**Posture C — Hybrid: observable heartbeat in-story; multicast wire I/O
deferred to S-BL.DISCOVERY-WIRE.**

S-7.02 v1.1 is amended to v1.2:

1. The purity classification is tightened from `boundary — Multicast I/O`
   to `boundary — in-process registry seam (multicast I/O deferred)`.
2. AC-001b is revised: the heartbeat timer must fire an observable side
   effect (incrementing an exported or injected counter/channel) so the
   test oracle can assert N heartbeats fired in N ticks. A no-op body
   that a removed ticker would leave green is an unacceptable oracle.
3. AC-005 (HMAC key derivation) is annotated with an explicit DRIFT entry
   acknowledging `advertisementKey(svtnID) = svtnID` is a scoping
   placeholder pending the admitted-node HMAC key vocabulary from
   S-6.03/S-BL.DISCOVERY-WIRE. The AC is NOT dropped — the fail-closed
   rejection of a mismatched tag must still be verified.
4. Real multicast UDP socket, `net.ListenMulticastUDP`, and dispatch
   goroutine are deferred to the new backlog story S-BL.DISCOVERY-WIRE.
5. BC-2.03.001 is patched (v1.2) to reflect the registry/wire scope split.

---

## Rationale

### 1. Why not Posture B (build the wire boundary in S-7.02)

Wave-6 Tranche B already holds S-7.01 and S-BL.ROUTER-ADDR in the same
fix-burst window. Tranche B is scoped for delivery, not architectural
expansion.

The multicast wire boundary has two hard unresolved dependencies:

- **Admitted-node HMAC key vocabulary.** C-3 confirms the current key
  derivation is `advertisementKey(svtnID) = svtnID` — the SVTN ID is
  not admitted-node-scoped secret material. Real multicast authentication
  requires keying material established at Tier-1 admission (S-2.02/S-6.03
  surfaces). That vocabulary is not yet specified in any BC or story.
  Forcing it into S-7.02 would require specifying a new admission-key
  shape, which is a Phase 1a product-owner obligation, not a Wave-6
  implementer obligation.

- **No multicast address allocation spec.** BC-2.03.001 PC-1 states
  "SVTN-scoped multicast address is allocated for the SVTN's discovery
  channel." No BC or architecture doc specifies how that address is
  computed from the SVTN ID or where it is stored. Implementing
  `net.ListenMulticastUDP` without a specified multicast address
  derivation rule would produce unverifiable behavior.

Precedent: Ruling W6TB-B (S-BL.ROUTER-ADDR) and Ruling W6TB-C (S-7.03
scope reduction / S-BL.CONSOLE-OBS) both defer features with unresolved
design dependencies to backlog stories. Posture B is rejected on the same
grounds.

### 2. Why not Posture A (drop heartbeat and HMAC ACs entirely)

Posture A over-scopes the deferral. The heartbeat timer (AC-001b) and
HMAC fail-closed rejection (AC-005) are both unit-testable without
multicast I/O:

- Heartbeat: the timer goroutine already exists in `Run`. Adding an
  injected clock and an observable counter or notification channel is a
  < 30-line change that makes AC-001b testable and removes the dead-ticker
  risk identified in C-1.
- HMAC rejection: `ReceiveAdvertisement` already calls
  `routing.VerifyAdvertisementHMAC`. The fail-closed path is present in
  code; AC-005 only needs a test that supplies a wrong tag and asserts
  `ErrInvalidHMACTag` is returned. No UDP socket required.

Dropping these ACs would leave real behavioral gaps unverified. Posture A
is rejected.

### 3. Why Posture C is correct

The existing implementation correctly delivers a unit-scope session
registry: Advertise, Enumerate, ReceiveAdvertisement, Encode/Decode
round-trip, SVTN-scoped isolation, HMAC verification. These are all
testable in-process and cover AC-002, AC-003, AC-004, AC-005, AC-006.

The two gaps are narrow and fixable without multicast I/O:

- C-1: add an observable side effect to the heartbeat tick body.
- C-3: document the HMAC placeholder; it does not break any existing test
  (the HMAC is internally consistent within the in-process model).

This preserves the Wave-6 delivery window while creating a clean seam for
S-BL.DISCOVERY-WIRE to replace the registry I/O boundary with real
multicast sockets.

---

## Acceptance Criteria Delta (S-7.02 v1.1 → v1.2)

| AC | Current (v1.1) | Post-ruling (v1.2) | Status |
|----|---------------|-------------------|--------|
| AC-001a | `Advertise` fires within 1 tick on state change | Unchanged | KEEP |
| AC-001b | "heartbeat timer fires on schedule" — body is no-op | Revised: heartbeat tick must produce an observable side effect (injected counter or channel); test asserts N ticks → N heartbeat events | REVISE — C-1 fix |
| AC-002 | `Enumerate` aggregates ≥2 distinct advertisers | Unchanged | KEEP |
| AC-003 | Payload includes session_name, attachment_status, quality | Unchanged | KEEP |
| AC-004 | Round-trip stability | Unchanged | KEEP |
| AC-005 | HMAC tag fail-closed rejection | Unchanged behavior, but add explicit DRIFT annotation: "HMAC key = svtnID is scoping placeholder pending S-BL.DISCOVERY-WIRE admitted-node key vocabulary" | ANNOTATE — C-3 ack |
| AC-006 | SVTN cross-scope isolation (Enumerate returns 0 cross-SVTN sessions) | Unchanged | KEEP |

**`acceptance_criteria_count` stays 7 (no ACs dropped).**

New DRIFT entry to add to story body:

> **DRIFT-W6TBD-001 (scoping placeholder):** `advertisementKey(svtnID) =
> svtnID` uses the SVTN ID directly as the 16-byte HMAC key. This is
> internally consistent for the in-process registry model but is NOT
> admitted-node-scoped secret material. Real multicast authentication
> requires keying material from the Tier-1 admission layer. This is
> resolved in S-BL.DISCOVERY-WIRE, which must specify admitted-node HMAC
> key derivation in a new or amended BC before implementation.

---

## Story Frontmatter Delta (S-7.02)

| Field | Current (v1.1) | Post-ruling (v1.2) |
|-------|---------------|-------------------|
| `version` | `"1.1"` | `"1.2"` |
| `status` | `ready-for-red-gate` | `ready-for-red-gate` (unchanged) |
| `changed_by_rulings` | (absent) | `[RULING-W6TB-D]` |
| Purity Classification table — internal/discovery | `boundary — Multicast I/O; network presence advertisements` | `boundary — in-process registry seam; multicast I/O deferred to S-BL.DISCOVERY-WIRE` |

---

## BC-2.03.001 Amendments Required

Patch BC-2.03.001 from v1.1 to v1.2.

### Scope annotation added to Description section

After the first sentence add:

> **Implementation scope note (Ruling W6TB-D):** BC-2.03.001 covers the
> advertisement trigger model and payload semantics. The wire transport
> (UDP multicast socket, SVTN-scoped multicast address allocation, admitted-
> node HMAC key derivation) is split into BC-2.03.001-WIRE (implemented by
> S-BL.DISCOVERY-WIRE). The current implementing story S-7.02 delivers the
> in-process registry model; PC-1 (multicast to all admitted nodes), PC-3
> (1-tick network delivery), and PC-4 (network heartbeat broadcast) are
> fully verified only after S-BL.DISCOVERY-WIRE ships.

### Postcondition 4 annotation

Append to PC-4:

> **Observability gate (Ruling W6TB-D):** in the registry model (S-7.02),
> the periodic heartbeat timer fires an observable side effect verifiable by
> injecting a tick and asserting the heartbeat counter increments. Network
> dispatch to wire is deferred to S-BL.DISCOVERY-WIRE.

### Postcondition 5 annotation

Append to PC-5:

> **Key placeholder (Ruling W6TB-D / DRIFT-W6TBD-001):** in S-7.02, the
> HMAC key is `svtnID` (the SVTN identifier itself). This is a scoping
> placeholder. S-BL.DISCOVERY-WIRE must specify and implement admitted-node-
> scoped HMAC key material before multicast deployment. The fail-closed
> rejection behavior (ErrInvalidHMACTag on wrong tag) is fully verified in
> S-7.02.

### Changelog entry (BC-2.03.001 v1.2)

```
v1.2 | 2026-07-01 | product-owner | Ruling W6TB-D: scope split annotation
added. PC-1 wire transport, PC-4 network dispatch, and admitted-node HMAC
key vocabulary (DRIFT-W6TBD-001) deferred to S-BL.DISCOVERY-WIRE.
Observability gate added to PC-4: heartbeat timer observable via injected
counter. PC-5 key placeholder note added.
```

---

## Backlog Story Stub Required: S-BL.DISCOVERY-WIRE

Create `.factory/stories/S-BL.DISCOVERY-WIRE.md` with the following stub
frontmatter and scope. Full story decomposition deferred to Wave-7 planning.

```yaml
story_id: S-BL.DISCOVERY-WIRE
title: "Discovery wire boundary: UDP multicast I/O, admitted-node HMAC keys, multicast address allocation"
status: backlog
wave: backlog
bc_traces: [BC-2.03.001, BC-2.03.002]
depends_on: [S-7.02, S-2.02]
blocks: []
changed_by_rulings: [RULING-W6TB-D]
```

Scope (to be designed at scheduling time, requires product-owner pass):

1. Specify admitted-node HMAC key derivation for advertisement
   authentication (resolves DRIFT-W6TBD-001; amends BC-2.03.001 PC-5).
2. Specify SVTN-scoped multicast address derivation from SVTN ID.
3. Replace in-process `Advertise`/`ReceiveAdvertisement` paths with
   `net.ListenMulticastUDP` dispatch goroutine.
4. Wire Run() heartbeat body to call `Advertise` (actual network dispatch)
   every 30s (BC-2.03.001 PC-4 full network verification).
5. Integration test: VP-044 verified over real UDP multicast (loopback).
6. Verify BC-2.03.001 PC-1 (advertisement reaches all admitted nodes on
   SVTN) and PC-3 (1-tick network delivery) with integration harness.

**Estimated points:** 5–8 (TBD; admitted-node key vocabulary adds scope).

---

## File Structure MODIFY Table Delta (S-7.02 v1.2)

The file structure table in the story body is unchanged — the same two
files deliver the in-process model. Add a note column entry:

| File | Action | Purpose | Ruling Note |
|------|--------|---------|-------------|
| internal/discovery/discovery.go | modify | Add HeartbeatCount field or injected tick channel to Run(); no multicast socket | C-1 fix; observable heartbeat |
| internal/discovery/discovery_test.go | modify | Add `TestDiscovery_Advertise_PeriodicHeartbeat` asserting N ticks → N events; extend AC-005 test for HMAC DRIFT annotation | C-1 + C-3 |

No new files are required for S-7.02 v1.2. Multicast socket files
(`discovery_wire.go`, multicast address derivation) are S-BL.DISCOVERY-WIRE
scope.

---

## VP Impact

No VP retirements. VP-044 integration scope note:

> VP-044 is partially verified in S-7.02 (state-change trigger, heartbeat
> timer observability). Network delivery latency ("within 1 tick") requires
> S-BL.DISCOVERY-WIRE. VP-044 status remains `partial` until
> S-BL.DISCOVERY-WIRE ships.

---

## Decision Log

| Date | Actor | Entry |
|------|-------|-------|
| 2026-07-01 | product-owner | Initial ruling: Posture C. In-process registry model is correct scope for S-7.02. C-1 fixed by making heartbeat observable (injected counter/channel). C-2 accepted as design intent — multicast wire deferred to S-BL.DISCOVERY-WIRE. C-3 acknowledged as scoping placeholder via DRIFT-W6TBD-001; fail-closed HMAC rejection stays testable in-process. BC-2.03.001 patched v1.1→v1.2 with scope-split annotation. S-BL.DISCOVERY-WIRE stub created. Precedent: Ruling W6TB-B (router-addr seam), Ruling W6TB-C (console-transport scope reduction). |
