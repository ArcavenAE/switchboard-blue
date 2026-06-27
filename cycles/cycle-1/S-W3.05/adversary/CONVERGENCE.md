---
artifact_id: convergence-S-W3.05
story: S-W3.05
verdict: CONVERGED
streak: 3
streak_passes: [07, 08, 09]
converged_date: 2026-06-27
impl_commit: b945aab
test_commit: 5c3d7ea
total_passes: 9
not_converged_passes: [01, 02, 03, 04, 05, 06]
clean_passes: [07, 08, 09]
---

# S-W3.05 Adversarial Convergence Record

## Result: CONVERGED

Three consecutive clean passes (07, 08, 09). Zero CRITICAL. Zero HIGH across
all three passes.

## Convergence Summary

| Pass | Lens | C | H | M | L | Verdict |
|------|------|---|---|---|---|---------|
| 01 | initial spec-review | — | — | — | — | NOT_CONVERGED |
| 02 | re-review post-fix-1 | — | — | — | — | NOT_CONVERGED |
| 03 | re-review post-fix-2 | — | — | — | — | NOT_CONVERGED |
| 04 | restart: spec+wiring | 0 | 1 | 2 | 1 | NOT_CONVERGED |
| 05 | restart: anti-taut+proptest | 2 | 1 | 0 | 3 | NOT_CONVERGED |
| 06 | restart: integration/wiring | 0 | 1 | 0 | 0 | NOT_CONVERGED |
| **07** | **spec-conformance + anti-tautology** | **0** | **0** | **0** | **3** | **CONVERGED** |
| **08** | **concurrency + memory/resource-bounds** | **0** | **0** | **0** | **2** | **CONVERGED** |
| **09** | **integration + RouteFrame wiring** | **0** | **0** | **0** | **3** | **CONVERGED** |

## Prior Fix Loop

Passes 01–06 were NOT_CONVERGED. Blocking items resolved in fix loop
(commits b945aab impl, 5c3d7ea tests):

- E-ADM-017 canonical message phrase restored: "HMAC failure rate alert:"
- Per-source append-skip bound (CWE-770 / EC-011 / BC-2.05.005 v1.6)
- Drain-only re-arm: dead keep[0].After(lastFire) branch removed
- VP-059 proptest: stateful model, 3 configs, seed 1337+idx, non-tautological
- Dead-key discriminating test (AC-012)
- AC-003/AC-004/AC-015 corrected; AC-016/AC-017 added (story v1.2)
- error-taxonomy.md v2.0: prose/annotation aligned to drain-only re-arm
  (message-format string UNCHANGED)

## Deferred LOW Findings

| ID | Source | Description | Target |
|----|--------|-------------|--------|
| O-1/p07 | pass-07 obs-1 | AC-016 iteration count comment 10k vs 1M placeholder | post-wave cosmetic |
| O-2/p07 | pass-07 obs-2 | %.0fs precision for sub-second windows (outside configured range) | post-wave if sub-sec windows introduced |
| O-3/p07 | pass-07 obs-3 | Step2/Step3 re-arm partial redundancy — harmless | post-wave cosmetic |
| O-2/p08 | pass-08 obs-2 | Stale TrackedSourceCount() name in comment | post-wave cosmetic |
| obs-1/p09 | pass-09 obs-1 | Routing e2e full-canonical-phrase assertion through RouteFrame | FOLD INTO S-W3.04 |
| obs-3/p09 | pass-09 obs-3 | Stale Red-Gate test comment (pre-v1.6 fired map[string]bool) | post-wave cosmetic |

## Spec Versions at Convergence

- BC-2.05.005: v1.6
- BC-2.05.008: v1.3
- VP-059: v1.1
- story S-W3.05: v1.2 (af05c04)
- error-taxonomy.md: v2.0
