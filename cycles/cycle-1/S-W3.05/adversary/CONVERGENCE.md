---
artifact_id: convergence-S-W3.05
story: S-W3.05
verdict: CONVERGED
streak: 3
streak_passes: [10, 11, 12]
converged_date: 2026-06-27
converged_tip: f6038d2
impl_commit: b945aab
sec001_fix_commit: f6038d2
total_passes: 12
not_converged_passes: [01, 02, 03, 04, 05, 06]
superseded_clean_passes: [07, 08, 09]
clean_passes: [10, 11, 12]
second_convergence: true
---

# S-W3.05 Adversarial Convergence Record

## Result: CONVERGED (second convergence at f6038d2)

Three consecutive clean passes (10, 11, 12) at HEAD f6038d2. Zero
CRITICAL. Zero HIGH across all three passes.

## Convergence History

### First convergence (superseded)

Passes 07-09 achieved CONVERGED at 5c3d7ea. Streak reset by SEC-001
(HIGH, CWE-476, nil-logger deref panic in `NewFailureCounter`) found by
the security-reviewer during PR #16 review — AFTER the 07-09 convergence.
SEC-001 was fixed in commit f6038d2 (eager nil-logger guard +
discriminating panic test). Spec updated: BC-2.05.005 v1.7→v1.8
(sanctions the constructor precondition), VP-059 v1.1→v1.2 (TrackedSourceCount
name fix). No behavioral change to alerting logic.

### Second convergence (current)

Three fresh-context passes (10/11/12) re-ran against the ACTUAL tip
f6038d2. All returned CONVERGED.

## Convergence Summary

| Pass | Lens | C | H | M | L | Verdict | Tip |
|------|------|---|---|---|---|---------|-----|
| 01 | initial spec-review | — | — | — | — | NOT_CONVERGED | — |
| 02 | re-review post-fix-1 | — | — | — | — | NOT_CONVERGED | — |
| 03 | re-review post-fix-2 | — | — | — | — | NOT_CONVERGED | — |
| 04 | restart: spec+wiring | 0 | 1 | 2 | 1 | NOT_CONVERGED | — |
| 05 | restart: anti-taut+proptest | 2 | 1 | 0 | 3 | NOT_CONVERGED | — |
| 06 | restart: integration/wiring | 0 | 1 | 0 | 0 | NOT_CONVERGED | — |
| 07 | spec-conformance + anti-tautology | 0 | 0 | 0 | 3 | CONVERGED (superseded) | 5c3d7ea |
| 08 | concurrency + memory/resource-bounds | 0 | 0 | 0 | 2 | CONVERGED (superseded) | 5c3d7ea |
| 09 | integration + RouteFrame wiring | 0 | 0 | 0 | 3 | CONVERGED (superseded) | 5c3d7ea |
| **10** | **security / nil-safety (CWE-476/400/117/770)** | **0** | **0** | **0** | **2** | **CONVERGED** | **f6038d2** |
| **11** | **spec-conformance / anti-tautology** | **0** | **0** | **0** | **4** | **CONVERGED** | **f6038d2** |
| **12** | **concurrency / integration** | **0** | **0** | **0** | **3** | **CONVERGED** | **f6038d2** |

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

## Streak Reset (passes 07-09 superseded)

SEC-001 (CWE-476, HIGH) was found post-5c3d7ea by the security-reviewer
during PR #16 review. The nil-logger deref panic was reachable in
production `NewFailureCounter` calls when called without a logger. Fixed
in f6038d2 (eager guard, exact panic message, discriminating test).

The 07-09 streak was CONVERGED but at a tip that had an undetected HIGH.
Three fresh-context passes at f6038d2 (passes 10/11/12) confirm that
the fix is correct, no regression was introduced, and the implementation
is now clean.

**Process note:** Passes 07-09 missed SEC-001 because none of the lenses
explicitly targeted constructor-injected dependency nil-safety / panic-path
sweeps. Added as process-gap lesson: adversarial lenses should include
an explicit nil-safety / panic-path sweep for constructor-injected
dependencies.

## Deferred LOW Findings (from passes 10-12)

| ID | Source | Description | Target |
|----|--------|-------------|--------|
| OBS-1/p10 | pass-10 | evictLRU empty-slice dead branch (unreachable guard) | post-wave cosmetic |
| OBS-2/p10 | pass-10 | appendSkip lastFire-local trace not covered by focused test | post-wave cosmetic |
| OBS-1/p11 | pass-11 | comment/func-name drift (stale Red-Gate, TrackedSourceCount) | post-wave cosmetic |
| OBS-2/p11 | pass-11 | AC-016 10k vs 1M iteration placeholder | post-wave |
| OBS-3/p11 | pass-11 | stale Red-Gate header comment (pre-v1.6 fired map[string]bool) | post-wave cosmetic |
| OBS-4/p11 | pass-11 | BC-2.05.005 v1.7 citation in test file header (should be v1.8) | post-wave cosmetic |
| OBS-1/p12 | pass-12 | evictLRU dead branch (carried from p10) | post-wave cosmetic |
| OBS-2/p12 | pass-12 | VP-059 TrackedSourceCount drift — FIXED in v1.2 | FIXED |
| OBS-3/p12 | pass-12 | BC-2.05.005 stale re-arm test-vector rows — FIXED in v1.8 | FIXED |

Also carried from passes 07-09:

| ID | Source | Description | Target |
|----|--------|-------------|--------|
| obs-1/p09 | pass-09 | Routing e2e full-canonical-phrase assertion through RouteFrame | FOLD INTO S-W3.04 |
| obs-3/p09 | pass-09 | Stale Red-Gate test comment | post-wave cosmetic |

## Spec Versions at Second Convergence (f6038d2)

- BC-2.05.005: v1.8 (sanction of SEC-001 nil-logger precondition; no behavioral change)
- BC-2.05.008: v1.3
- VP-059: v1.2 (TrackedSourceCount→SourceCount name fix; no behavioral change)
- story S-W3.05: v1.3
- error-taxonomy.md: v2.0
