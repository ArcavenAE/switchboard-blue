---
artifact_id: S-4.01-pass1-spec-rulings
document_type: adjudication
level: ops
story_id: S-4.01
title: "Pass 1 Adversarial Review — Spec Rulings"
status: final
producer: product-owner
timestamp: 2026-06-27T00:00:00
phase: 3
cycle: v1.0.0-greenfield
---

# S-4.01 Pass 1 Spec Rulings

These rulings are authoritative. Implementer and test-writer must satisfy the
contracts as ruled below. No re-adjudication is needed unless a cited spec is
amended (version-bumped by the product-owner).

---

## RULING 1 — F-002: Endpoint Deduplication Key (checksum-only vs. compound)

### Question

Does BC-2.02.002 require the endpoint's `Multipath.Receive` deduplicator to key
on **checksum alone**, or on the compound `(checksum, arrival_interface_id)` that
the router's drop cache (BC-2.02.009) uses?

### Ruling

**Checksum alone at the endpoint.** The compound key
`(checksum, arrival_interface_id)` is correct ONLY for the router-level drop
cache (BC-2.02.009). The endpoint receiver must key on checksum (or equivalently,
sequence number within a channel) alone.

The test `TestBC_2_02_002_Receive_DifferentInterfaceSameChecksumNotSuppressed` pins
**wrong** behavior and must be deleted or rewritten to assert that the second copy
IS suppressed.

### Authoritative Citations

**BC-2.02.002 postcondition 2 (version 1.1):**
> "Any subsequent frame with the **same sequence number** from the same channel is
> discarded without error, without ACK side-effects."

Postcondition 2 keys on sequence number, not (sequence number, interface). There
is no interface qualifier at the endpoint.

**BC-2.02.002 canonical test vector 1 (version 1.1):**
> "Frame seq=42 arrives on path A at t=0ms; same frame arrives on path B at t=8ms
> → Frame delivered at t=0ms; second copy at t=8ms **discarded silently**."

The test vector explicitly exercises the cross-interface duplicate scenario and
specifies discard. This is definitive.

**BC-2.02.002 invariant 1 (DI-009, version 1.1):**
> "First arrival wins. This invariant is enforced at the **receiver**, not the
> router."

**VP-024 "Note on dedup scope" (version 1.0):**
> "This VP tests ENDPOINT-side dedup (BC-2.02.002 receiver-first-arrival
> semantics). Router-side dedup is orthogonal: routers use the compound key
> `(checksum, arrival_interface_id)` per ARCH-03 §Drop cache (F-006 resolution)
> to preserve multipath duplicate-and-race copies. By the time both copies reach
> the endpoint, they have traversed different routers via different interfaces —
> **endpoint-side dedup by checksum is the correct semantic per BC-2.02.002**
> (deliver first arrival, silently discard second)."

**BC-2.02.009 description (version 1.1):**
> "Using a compound key (not checksum alone) allows the same frame to arrive on
> two different interfaces — as in multipath duplicate-and-race — and be forwarded
> independently; only a true loop (same frame on the same interface) is suppressed."

BC-2.02.009 explicitly explains WHY the router uses compound key: so that
multipath copies on different interfaces are NOT dropped in transit. This
rationale does not apply at the endpoint, where the goal is exactly the opposite:
deliver only once.

**ARCH-03 §Duplicate-and-Race (version 1.0):**

The ARCH-03 `OnFrameArrival` pseudo-code uses `(checksum, arrival_interface_id)`
as the key and labels it "silently discard (BC-2.02.002, DI-009)". **This is a
documentation error in ARCH-03.** The pseudo-code describes a unified function
that is architecturally positioned at the router, not the endpoint. The citation
of BC-2.02.002 in that code comment is incorrect — BC-2.02.002 governs the
endpoint receiver; BC-2.02.009 governs the router drop cache. The paragraph
immediately following the pseudo-code clarifies:

> "**Drop cache key (F-006):** The drop cache key is `(checksum,
> arrival_interface_id)`, not `(checksum)` alone. This ensures two copies of a
> frame arriving on different interfaces are **both kept** — multipath delivery
> requires both copies to survive intermediate hops so the fastest arrives first."

This paragraph describes router-hop behavior ("survive intermediate hops"), not
endpoint behavior. The sentence "The destination node receives at most two copies
and delivers the first, discarding the duplicate via the same `(checksum,
arrival_interface_id)` keying" is the ARCH-03 error: it incorrectly extends the
compound key to the endpoint. The BCs and VP-024 are the source of truth for
endpoint behavior and they are unambiguous.

### Contract the Implementer and Test-Writer Must Satisfy

- `Multipath.Receive` (endpoint) deduplications key on **checksum alone** (or
  equivalently sequence number within a channel per BC-2.02.002 postcondition 2;
  both are correct and consistent — checksum uniquely identifies frame content,
  sequence number uniquely identifies a frame's slot in the channel; for identical
  duplicate-and-race copies these are equivalent identifiers).
- When frame F arrives on interface A, then the same frame F (identical bytes,
  same checksum, same seq) arrives on interface B, the second arrival MUST be
  discarded. `ErrDuplicate` (or nil delivery, per the interface contract) is
  returned for the second arrival regardless of which interface it came from.
- `TestBC_2_02_002_Receive_DifferentInterfaceSameChecksumNotSuppressed` asserts
  the wrong behavior. It must be **replaced** with a test that asserts:
  same-checksum frame arriving on a different interface IS suppressed (second copy
  discarded, first delivered).
- AC-004 in the story ("delivers first-arriving copy and returns `ErrDuplicate`
  for subsequent copies with the same checksum") is correct and consistent with
  this ruling.

### Required ARCH-03 Correction (Architect Owns the Edit)

ARCH-03 §Duplicate-and-Race last sentence of the "Drop cache key (F-006)" block:

> CURRENT (incorrect): "The destination node receives at most two copies and
> delivers the first, discarding the duplicate via the same `(checksum,
> arrival_interface_id)` keying."

> CORRECTED: "The destination node receives at most two copies. The endpoint
> receiver deduplicates by checksum alone (BC-2.02.002): the first-arriving copy
> is delivered; the second is silently discarded regardless of arrival interface.
> Router-side compound keying ensures both copies reach the destination; endpoint-
> side checksum-only keying ensures only one is delivered to the application."

The ARCH-03 code comment `silently discard (BC-2.02.002, DI-009)` should be
changed to `silently discard (BC-2.02.009, DI-009)` since that pseudo-code
describes the router drop cache, not the endpoint receiver.

Version bump: ARCH-03 version 1.0 → 1.1. Architect owns this edit.

---

## RULING 2 — F-006: Path Reactivation After Recovery

### Question

Does BC-2.02.003 require a path deactivated after 3 consecutive missed probes to
return to the active set when successful probes resume? If yes, what is the exact
reactivation condition?

### Ruling

**Yes, reactivation is required.** A deactivated path MUST return to the active
path set when successful probes resume. The reactivation condition is: **first
successful probe response received** on the previously-failed path.

### Authoritative Citations

**BC-2.02.003 postcondition 1 (version 1.1):**
> "After each keep-alive round-trip, the path RTT is **updated** using an EWMA."

"Each" round-trip — there is no carve-out for paths that were previously failed.
If a probe round-trip completes, the RTT is updated. This implies the path must
be eligible to receive probes and report responses even after deactivation.

**BC-2.02.003 postcondition 6 (version 1.1):**
> "A path with > N consecutive missed keep-alives (implementation: N=3) is marked
> as **failed and removed from the active path set**."

Postcondition 6 specifies what triggers removal. It does not say the path is
permanently removed. The word "consecutive" is key: the condition is N
*consecutive* misses. A successful probe breaks the consecutive streak.

**BC-2.02.003 canonical test vector 4 (version 1.1):**
> "Path RTT spikes to 300ms for 2 probes then recovers → EWMA smooths spike;
> path briefly degrades in ranking; **recovers on good probes**."

"Recovers on good probes" is a direct statement that recovery occurs. While this
test vector addresses degradation (not full deactivation), the principle is the
same: good probes → recovery.

**BC-2.02.003 edge case EC-001 (DEC-003, version 1.1):**
> "Path RTT degrades from 10ms to 250ms → EWMA smoothly transitions ranking;
> path moves to lower priority. After sustained degradation, path removed from
> active set. **Failover within 2 seconds.**"

EC-001 says "removed from active set" but frames this in the context of sustained
degradation. It does not say paths are permanently removed. The VP-040 failover
test (VP-040) tests recovery of the SESSION, not permanent path loss — it
exercises the path-fail → session-recovers-on-remaining-path cycle. The intent is
detection and failover, not permanent exclusion.

**VP-040 property statement (version 1.0):**
> "When one of two active paths fails (simulated by closing one router
> connection): session traffic recovers on the remaining path within 2 seconds."

VP-040 tests failure → recovery of the session. The story's E2E test
`TestE2E_Multipath_FailoverRecovery` uses `env.CloseRouterConnection` which
simulates a hard path failure. This confirms the path-fail path is implemented
as a detection+failover mechanism, not as a one-way door.

### Exact Reactivation Condition

> A path transitions from FAILED back to the ACTIVE path set upon receiving the
> **first successful probe response** (a keep-alive round-trip completes). On
> reactivation, the path's RTT is initialized from the measured round-trip time
> of the reactivating probe (not carried over from before deactivation). Loss rate
> EWMA resets to 0 (conservative reactivation assumption: the path is initially
> treated as loss-free and ranked by RTT alone until further probes accumulate
> loss statistics).

### Rationale for "First Success"

BC-2.02.003 does not specify N consecutive successes for reactivation. Requiring
N successes before reactivation would create an asymmetry (N misses to deactivate,
N successes to reactivate) that is not stated in the spec. The canonical test
vector ("recovers on good probes") uses the plural loosely — what it requires is
that recovery happens, not that a specific count is needed. First-success
reactivation is consistent with the failover intent (minimize downtime) and with
the EWMA postcondition (RTT is updated after each round-trip). The implementer MAY
choose to require 2 consecutive successes as a hysteresis guard if oscillation is
observed in testing, but 1 success is the minimum the spec mandates.

### Contract the Implementer Must Satisfy

- `PathTracker` must track whether a path is in ACTIVE or FAILED state.
- Transition ACTIVE → FAILED: 3 consecutive missed probes (postcondition 6).
  "Consecutive" means the count resets on any successful probe while still ACTIVE.
- Transition FAILED → ACTIVE: first successful probe response received.
  RTT initialized from the reactivating probe's round-trip time.
  Loss EWMA resets to 0.
- FAILED paths are still probed (probe sends must continue). Only delivery of
  frames is suspended for FAILED paths.
- The test for this behavior must cover: ACTIVE → (3 misses) → FAILED → (1
  success) → ACTIVE, with assertions that the path is absent from the active set
  between the two transitions and present after the third.

### BC-2.02.003 Clarifying Amendment

BC-2.02.003 postcondition 6 is amended to add the reactivation condition. This
is a clarification (no behavior change for the deactivation trigger), not a
new requirement.

**BC-2.02.003 postcondition 6 — CURRENT:**
> "A path with > N consecutive missed keep-alives (implementation: N=3) is marked
> as failed and removed from the active path set."

**BC-2.02.003 postcondition 6 — AMENDED (version 1.2):**
> "A path with > N consecutive missed keep-alives (implementation: N=3) is marked
> as failed and removed from the active path set. A failed path is re-added to the
> active path set upon the first successful keep-alive round-trip; its RTT is
> initialized from the reactivating probe's measured RTT and its loss EWMA resets
> to 0. Probes continue to be sent to failed paths so that recovery is detected."

Version bump: BC-2.02.003 version 1.1 → 1.2.

---

## RULING 3 — F-010: Single-Path Dispatch — One Copy or Two?

### Question

BC-2.02.001 postcondition 3 says "one send" on single-path fallback. ARCH-03
lines 45-46 say "both copies go to the same path (degenerate case)." Which is
authoritative for S-4.01?

### Ruling

**One send. BC-2.02.001 postcondition 3 is authoritative.** ARCH-03 lines 45-46
are incorrect and must be corrected.

### Authoritative Citations

**BC-2.02.001 postcondition 3 (version 1.1):**
> "If only one path is available, the frame is sent on that single path (**no
> error; single-path fallback**)."

"Sent on that single path" — one send. No duplication.

**BC-2.02.001 edge case EC-001 (version 1.1):**
> "Node has exactly one connected router → Frame sent on that router only; **no
> duplicate**. Quality indicator notes single-path mode."

"No duplicate" is unambiguous. One send to one router.

**BC-2.02.001 canonical test vector 3 (version 1.1):**
> "1 path available → Frame dispatched on single path; no error"

One send, no error.

**S-4.01 edge case EC-001 (story, version current):**
> "Only one path available → Send dispatches on single path; no error"

The story's own edge case EC-001 is consistent with the BC: one send.

**ARCH-03 §Path Selection and Quality Tracking (version 1.0), lines 45-46:**
> "If only one path exists (E router MVP), both copies go to the same path
> (degenerate case)."

This contradicts BC-2.02.001 postcondition 3 and EC-001. It is a spec error in
ARCH-03. The BCs are the behavioral source of truth per VSDD methodology; ARCH-03
is the structural mapping. Where ARCH-03 specifies behavior that contradicts a
BC, the BC governs.

### Why Sending Two to the Same Path Is Wrong

Sending two identical frames to the same path provides no resilience benefit (both
will fail if the path fails), doubles bandwidth on the single path, and creates
spurious duplicates that must be deduplicated at the receiver. BC-2.02.001 is
correct in specifying single send for single-path mode.

### Contract the Implementer Must Satisfy

- `Multipath.Send` in single-path mode (exactly one path in the path set) MUST
  dispatch exactly **one copy** of the frame.
- No error is returned for single-path mode.
- The test `TestMultipath_SendOnSinglePath` (or equivalent) must assert that
  exactly one send is observed on the underlying path mock/spy.

### Required ARCH-03 Correction (Architect Owns the Edit)

ARCH-03 §Path Selection and Quality Tracking:

> CURRENT (incorrect): "If only one path exists (E router MVP), both copies go
> to the same path (degenerate case)."

> CORRECTED: "If only one path exists, the frame is sent on that single path
> with no duplication (BC-2.02.001 postcondition 3, EC-001). Single-path mode
> is noted in the quality indicator."

Version bump: ARCH-03 version 1.0 → 1.1 (combined with the Q1 correction above;
both corrections are in the same version bump).

---

## Summary of Changes Required

| Artifact | Change | Owner | Version Bump |
|----------|--------|-------|-------------|
| BC-2.02.003 | Add reactivation condition to postcondition 6 | product-owner | 1.1 → 1.2 |
| ARCH-03 | Fix endpoint dedup key citation in §Duplicate-and-Race | architect | 1.0 → 1.1 |
| ARCH-03 | Fix single-path dispatch description in §Path Selection | architect | (same 1.0 → 1.1) |
| `TestBC_2_02_002_Receive_DifferentInterfaceSameChecksumNotSuppressed` | Delete and replace — pins wrong behavior | test-writer | n/a |

BC-2.02.001, BC-2.02.002, BC-2.02.009, VP-024, VP-040, VP-054 require **no
behavioral changes**. They are correctly specified. The adversary's concern on
Q1 is validated by the specs; the implementation and the test that pins compound-
key endpoint behavior are both wrong.

---

## Cross-Reference

| Finding | Ruling Summary |
|---------|---------------|
| F-002 | Endpoint dedup: checksum-only. ARCH-03 has a doc error. Impl + test both wrong. |
| F-006 | Path reactivation: required on first successful probe. BC-2.02.003 amended (v1.2). |
| F-010 | Single-path: one send. ARCH-03 has a doc error. BC-2.02.001 governs. |
