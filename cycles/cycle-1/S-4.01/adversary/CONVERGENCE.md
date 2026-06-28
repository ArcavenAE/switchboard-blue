---
artifact_id: convergence-S-4.01
story: S-4.01
verdict: CONVERGED
streak: 3
streak_passes: [3, 4, 5]
converged_date: 2026-06-28
converged_tip: aaff609
impl_commit: aaff609
doc_commit: 327f5c6
total_passes: 5
not_converged_passes: [1, 2]
clean_passes: [3, 4, 5]
second_convergence: false
---

# S-4.01 Adversarial Convergence Record

## Result: CONVERGED (BC-5.39.001)

Three consecutive CONVERGED 0C/0H passes (3, 4, 5) at HEAD aaff609. Zero
CRITICAL. Zero HIGH across all three passes. BC-5.39.001 per-story
adversarial convergence obligation satisfied.

Post-convergence: comment-only header trim at 327f5c6 (zero behavioral
delta). Preserves convergence per S-W3.04 77c6229 precedent.

## Convergence Summary

| Pass | Lens | C | H | M | L | O | Verdict | Tip |
|------|------|---|---|---|---|---|---------|-----|
| 1 | initial review | 2 | 4 | — | — | — | NOT_CONVERGED | 571a31b |
| 2 | post-fix re-review | 0 | 2 | 4 | — | — | NOT_CONVERGED | cce35a3 |
| **3** | **spec-conformance / dedup / RTT** | **0** | **0** | **0** | **0** | **4** | **CONVERGED** | **982a0f3** |
| **4** | **concurrency / lock discipline** | **0** | **0** | **0** | **0** | **4** | **CONVERGED** | **aaff609** |
| **5** | **integration / edge-cases** | **0** | **0** | **0** | **0** | **4** | **CONVERGED** | **aaff609** |

## Pass History

### Pass 1 — NOT_CONVERGED 2C/4H (tip: 571a31b)

Two criticals and four highs.

**Criticals fixed:**

- F-001 (nil-safety / CWE-476): `Rank()` called `rp.Tracker.IsActive()`/`Score()`
  with no nil guard. `RankedPath` zero-value has `Tracker == nil`; nil tracker
  → production panic in routing hot path. Fixed: nil guard added.

- F-002 (spec-conformance, wrong-behavior-pinned test): `Receive` keyed drop
  cache on compound `(checksum, arrivalInterfaceID)`. VP-024 / BC-2.02.002 PC1-2 /
  DI-009 require endpoint dedup by checksum alone. With compound key, duplicates
  arriving on different interfaces both miss the cache — defeating DI-009
  first-arrival-wins. `TestBC_2_02_002_Receive_DifferentInterfaceSameChecksumNotSuppressed`
  pinned the wrong behavior. Fixed: dedup key changed to checksum-only; test
  corrected.

**Highs fixed (F-003..F-006):** silent failure (WriteAt no-error on bounds
violation), missing probe-to-active transition test, insufficient concurrent-Send
race coverage, EWMA first-probe boundary not locked to raw-RTT.

### Pass 2 — NOT_CONVERGED 0C/2H (tip: cce35a3)

Pass-1 criticals and highs all held — no regressions.

**Highs fixed:**

- F-H1 (dead-code / spec-conformance): `Multipath.dropCache` (BC-2.02.009 router
  loop-suppression) was constructed but NEVER read/written. No router
  `OnFrameArrival`/`Forward` method. BC-2.02.009 router behavior unreachable.
  Resolution: dead field removed; BC-2.02.009 router wiring explicitly deferred
  to S-4.04 with a comment + backlog entry.

- F-H2 (missing postcondition): BC-2.02.009 PC2 mandates a drop-cache hit counter
  for operator diagnostics. EC-005 mandates collision-event logging. Neither was
  present. Resolution: hit counter added to `DropCache`; collision hook wired;
  BC-2.02.009 router-wiring deferred annotation added.

**Mediums carried as non-blocking observations (M1..M4):** reactivation/first-probe
RTT duplication (maintenance cosmetic), EC-003/queue-with-timeout deferred (no
E-NET-002 queue — pure-core simplification noted in story), Send/UpdatePaths
race-test gap (lock validated), unused Receive interface-id param (YAGNI).

### Pass 3 — CONVERGED 0C/0H (tip: 982a0f3) — streak 1/3

Clean pass. Spec-conformance / dedup / RTT lens. Zero C/H. Four observations
carried forward (O-1..O-4).

### Pass 4 — CONVERGED 0C/0H (tip: aaff609) — streak 2/3

Clean pass. Concurrency / lock discipline lens. Zero C/H. Four observations
carried forward.

### Pass 5 — CONVERGED 0C/0H (tip: aaff609) — streak 3/3

Clean pass. Integration / edge-cases lens. Zero C/H. BC-5.39.001 satisfied.

## Non-Blocking Observations (carried post-convergence)

| ID | Pass | Description | Target |
|----|------|-------------|--------|
| O-1 | 3-5 | Weak/redundant EWMA-3-probe test — 3-probe path covered elsewhere; cosmetic | post-wave cosmetic |
| O-2 | 3 | Stale build-tag header comment | RESOLVED 327f5c6 |
| O-3 | 3-5 | BC-2.02.003 PC5: degraded-path flag (RTT >200ms) unimplemented in internal/paths — feeds quality-indicator subsystem (BC-2.06.001/ARCH-03); ranking already deprioritizes slow paths via score; flag wiring deferred to quality-indicator story | deferred → quality-indicator story |
| O-4 | 3-5 | Rank two-lock snapshot (paths mu + caller mu) — confirmed safe by design; no nested-lock inversion; reviewed by implementer | no action required |

## Fix Loop Summary (Passes 1-2)

Blocking items resolved before convergence streak:

- Nil-tracker panic (CWE-476) in `Rank()` — nil guard added
- Wrong-behavior-pinned dedup test (F-002) — checksum-only key; test corrected
- Probe-to-active transition test gap — test added
- Concurrent-Send race coverage — race test added
- Dead `dropCache` field removed; BC-2.02.009 router wiring deferred to S-4.04
- Hit counter added to `DropCache`; collision hook wired (BC-2.02.009 PC2)
- EWMA first-probe boundary: raw-RTT lock enforced

## Post-Convergence State

- Convergence tip: aaff609
- Post-convergence doc commit: 327f5c6 (comment-only header trim; zero behavioral delta)
- O-2 resolved at 327f5c6
- O-1 / O-3 / O-4 deferred (non-blocking, see table above)
- S-4.01 status: adversarial convergence COMPLETE; pending demo + PR + merge
