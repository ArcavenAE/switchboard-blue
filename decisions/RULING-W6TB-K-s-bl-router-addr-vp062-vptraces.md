---
artifact_id: RULING-W6TB-K-s-bl-router-addr-vp062-vptraces
document_type: decision
level: ops
version: "1.0"
status: final
producer: product-owner
timestamp: 2026-07-01T00:00:00
updated: 2026-07-01T00:00:00
cycle: v1.0.0-greenfield
stories_in_scope: [S-BL.ROUTER-ADDR]
closes_findings: [F-P4L3-001, F-P4L2-02, F-P4L2-03]
referenced_by:
  - .factory/stories/S-BL.ROUTER-ADDR.md
  - .factory/specs/verification-properties/VP-062.md
  - .factory/specs/verification-properties/VP-047.md
---

# Ruling W6TB-K — S-BL.ROUTER-ADDR VP-062 vp_traces, Concurrent-Oracle Intent, and Split-Red-Gate Design

**Adjudicator:** product-owner
**Date:** 2026-07-01
**Trigger:** S-BL.ROUTER-ADDR Pass-4 findings F-P4L3-001 (LOW-3), F-P4L2-02 (LOW-2), F-P4L2-03 (LOW-2)

This ruling adjudicates three related Pass-4 findings for S-BL.ROUTER-ADDR in a
single document because all three concern test oracle design for the same story and
none requires a BC or VP content change.

---

## Finding 1 of 3: VP-062 Unsupported vp_traces Claim (F-P4L3-001)

### Summary

S-BL.ROUTER-ADDR frontmatter declares `vp_traces: [VP-047, VP-062]`. VP-062
(JSON well-formedness fuzz, `implementing_story: S-W5.04`) has no AC, no
body-table row, no Task step, and no MODIFY row in S-BL.ROUTER-ADDR. The story
does not contribute to VP-062 implementation.

### Options

**Option A (adopted):** Remove VP-062 from `vp_traces` in S-BL.ROUTER-ADDR
frontmatter. Add a changelog note documenting the rationale: VP-062 fuzz
well-formedness is a compositional JSON property — adding a new string field
(`router_addr`) to `PathEntry` cannot break JSON marshaling; `encoding/json`
marshals any valid string field without error. VP-062 remains anchored to
S-W5.04 and is not perturbed by this story.

**Option B:** Add AC-006 exercising VP-062 fuzz corpus regeneration for the new
`router_addr` field. Rejected: VP-062 already includes `router_addr` fuzz coverage
via v1.4 Property 5b and fuzz corpus seeds 9 (added per Ruling-1). The S-W5.04
story owns VP-062 implementation. Duplicating that work in S-BL.ROUTER-ADDR
creates dual-ownership confusion.

### Decision: Option A

**Ruling: Remove VP-062 from `vp_traces`. No VP-062 content change required.**

### Rationale

JSON well-formedness is a compositional property of `encoding/json.Marshal`. A
story that adds a new `string` field to a struct does not need to re-verify that
`encoding/json` produces valid JSON — that invariant holds for all string fields
by construction and is already fuzz-tested in VP-062 via the `router_addr` seeds
added in v1.4 (Ruling-1). Claiming `vp_traces: VP-062` without any implementing
AC or Task step is a false traceability claim that would mislead the
consistency-validator.

### Changelog Note for S-BL.ROUTER-ADDR

```
VP-062 removed from vp_traces (RULING-W6TB-K F-P4L3-001): VP-062 (JSON
well-formedness fuzz) is anchored to S-W5.04, which already covers router_addr
fuzz seeds via VP-062 v1.4 Property 5b (Ruling-1). Adding a string field to
PathEntry does not perturb JSON well-formedness; the compositional property holds
by construction.
```

---

## Finding 2 of 3: Concurrent-Oracle Utility (F-P4L2-02)

### Summary

`TestBC_2_06_003_RouterAddr_ConcurrentSnapshot` exercises concurrent `Snapshot()`
calls against a `PathTracker` that has `routerAddr` set. Since `routerAddr` is
written exactly once at construction (immutable after `NewPathTrackerWithAddr`
returns), the Go race detector would not flag an unlocked read of that field. The
finding questions whether the test adds value over
`TestBC_2_06_003_RouterAddr_ImmutableAfterConstruction`.

### Options

**Option A (adopted):** Retain as-is. Clarify via comment in the test file that
the concurrent test's value is not in verifying `routerAddr` (which is immutable)
but in exercising the full `Snapshot()` + `OnProbe()` interaction under concurrent
access: `Snapshot()` reads EWMA/histogram fields that `OnProbe()` writes under
mutex, and `routerAddr` is a stable oracle whose expected value is known without
locking. The test verifies that concurrent Snapshot/OnProbe does not produce a
`routerAddr` value inconsistency as a side-effect of mutex contention — a genuine
Go race test.

**Option B:** Extend the test to also assert EWMA/loss field consistency during
concurrent OnProbe. Out of scope for this story: EWMA/loss consistency under
concurrency is the concern of S-BL.PATH-QUALITY-EWMA, not S-BL.ROUTER-ADDR.
Adding that assertion here creates dual-ownership.

**Option C:** Delete the test as redundant. Rejected: the concurrent Snapshot +
OnProbe pairing is not exercised by `TestBC_2_06_003_RouterAddr_ImmutableAfterConstruction`
(which is single-threaded). The test has genuine value as a race-detector harness
for the mutex path. Deletion would create a coverage gap.

### Decision: Option A

**Ruling: Retain the concurrent oracle test. Add a clarifying comment. No code
change beyond the comment.**

### Required Comment (mechanical — implementer adds during task 9 clean-up)

At the top of `TestBC_2_06_003_RouterAddr_ConcurrentSnapshot`, add:

```go
// TestBC_2_06_003_RouterAddr_ConcurrentSnapshot exercises the concurrent
// Snapshot()+OnProbe() path under the Go race detector.
//
// routerAddr is immutable after construction, so it serves as a stable oracle
// whose value is known without locking. The test's actual target is the
// Snapshot()/OnProbe() mutex path: OnProbe writes EWMA/histogram fields that
// Snapshot reads under the same mutex. routerAddr consistency is the assertion
// vehicle, not the property under test.
//
// RULING-W6TB-K: this test is intentionally retained (not merged into
// TestBC_2_06_003_RouterAddr_ImmutableAfterConstruction, which is
// single-threaded and does not exercise concurrent Snapshot/OnProbe).
```

---

## Finding 3 of 3: Split-Red-Gate Design for TestVP047_RouterAddrNonEmpty (F-P4L2-03)

### Summary

`TestVP047_RouterAddrNonEmpty` has two parts:

- **Part A:** Bypasses the constructor via `fakePathsListSource`, verifying the
  handler seam integration (AC-005 handler-side oracle). This part is
  GREEN-BY-DESIGN at the Red Gate because `fakePathsListSource` is a stub —
  it does not depend on `NewPathTrackerWithAddr`.
- **Part B:** Exercises the full constructor → `Snapshot()` → `router_addr` path.
  This is the only assertion that fails at the Red Gate.

The finding notes that the split may appear inconsistent because Part A passes at
Red Gate while Part B fails, raising the question of whether Part A should be
folded into `TestPathsList_PassesRouterAddr`.

### Options

**Option A (adopted):** Accept the split. Add a comment documenting that both
parts are load-bearing for AC-005 coverage: Part A verifies the handler seam
integration using `pathTrackerListSource` (the real list-source wiring), which
`TestPathsList_PassesRouterAddr` does not replicate because that test uses a
stub `PathsListSource` with pre-built entries rather than wiring through
`pathTrackerListSource`. Part B is the failing Red Gate assertion that drives
the implementation.

**Option B:** Fold Part A into `TestPathsList_PassesRouterAddr`; leave
`TestVP047_RouterAddrNonEmpty` as constructor-through-Snapshot exclusively.
Rejected: Part A and Part B together constitute the AC-005 oracle split required
by the story's acceptance criterion ("integration test verifying VP-047 AC-006
router_addr field assertion with non-empty stub addr"). Folding Part A into
`TestPathsList_PassesRouterAddr` would obscure the VP-047 traceability for Part A
and create a test that is not labeled as a VP-047 trace.

### Decision: Option A

**Ruling: Retain the split. Both parts must remain in `TestVP047_RouterAddrNonEmpty`.
Add a comment documenting that both are load-bearing.**

### Required Comment (mechanical — test-writer adds during test file creation)

At the top of `TestVP047_RouterAddrNonEmpty`, add:

```go
// TestVP047_RouterAddrNonEmpty is structured as two parts (RULING-W6TB-K F-P4L2-03):
//
// Part A (GREEN-BY-DESIGN at Red Gate): exercises the handler seam integration
// using pathTrackerListSource (the real list-source wiring). Part A uses a
// fakePathsListSource stub so it passes before the constructor is implemented.
// Its value is verifying that the handler correctly passes router_addr from the
// PathEntry to the JSON response — a seam that TestPathsList_PassesRouterAddr
// does not cover because that test uses pre-built stub PathEntry values rather
// than wiring through pathTrackerListSource.
//
// Part B (FAILS at Red Gate): exercises NewPathTrackerWithAddr → Snapshot() →
// router_addr. This is the Red Gate assertion that drives the implementation.
//
// Both parts MUST remain in this test. Together they constitute the AC-005 oracle
// for VP-047 router_addr traceability.
```

---

## Files to Modify

### `.factory/stories/S-BL.ROUTER-ADDR.md`

1. In frontmatter, change `vp_traces: [VP-047, VP-062]` to `vp_traces: [VP-047]`.
2. Append to changelog:

```
| <next-ver> | 2026-07-01 | product-owner | RULING-W6TB-K: (F-P4L3-001) VP-062 removed from vp_traces — compositional JSON property holds by construction; S-W5.04 already owns VP-062 fuzz coverage including router_addr seeds. (F-P4L2-02) Concurrent-oracle comment requirement noted for implementer. (F-P4L2-03) Split-red-gate comment requirement noted for test-writer. No AC or Task changes. |
```

Note: no VP files require content changes for this ruling. The comments required
for Findings 2 and 3 are mechanical additions to test file source code, dispatched
to implementer and test-writer respectively during the S-BL.ROUTER-ADDR
implementation burst.

---

## Downstream Dispatch Table

| Artifact | Change | Agent | When |
|----------|--------|-------|------|
| `.factory/stories/S-BL.ROUTER-ADDR.md` | vp_traces update + changelog (frontmatter only per bc_array_changes_propagate_to_body_and_acs) | product-owner (this ruling) | This burst |
| `internal/metrics/path_tracker_test.go` (worktree) | Add RULING-W6TB-K comment to `TestBC_2_06_003_RouterAddr_ConcurrentSnapshot` | implementer | S-BL.ROUTER-ADDR implementation burst |
| Integration test file (worktree) | Add RULING-W6TB-K comment to `TestVP047_RouterAddrNonEmpty` | test-writer | S-BL.ROUTER-ADDR test-write burst |

---

## Decision Log

| Date | Actor | Entry |
|------|-------|-------|
| 2026-07-01 | product-owner | F-P4L3-001: VP-062 removed from vp_traces. JSON well-formedness is compositional; adding a string field cannot break marshaling; S-W5.04 already fuzz-covers router_addr. F-P4L2-02: concurrent-oracle test retained with clarifying comment. Its value is the Snapshot/OnProbe mutex pairing under the race detector, not the routerAddr field itself. F-P4L2-03: split-red-gate design accepted. Part A (handler seam) and Part B (constructor path) are independently load-bearing for AC-005 coverage. |
