---
story: S-7.04-FU-SIGHUP-RELOAD
convergence_date: 2026-07-07
bc_satisfied: BC-5.39.001
total_passes: 14
converged_at_pass: 14
streak_passes: [12, 13, 14]
final_code_sha: 48e3271
final_story_version: "1.7"
code_commits: 16
code_sha_range: "6823a83 → 48e3271"
total_findings: 35
open_findings: 0
forward_obligations: 5
drift_rows_parked: 2
---

# S-7.04-FU-SIGHUP-RELOAD — Adversarial Convergence Summary

## At a Glance

| Metric | Value |
|--------|-------|
| Total passes | 14 |
| HAS_FINDINGS passes | P1–P11 (11) |
| Clean passes | P12, P13, P14 (3 consecutive) |
| BC-5.39.001 satisfied | Pass 14 — streak 3/3 |
| Final code SHA | 48e3271 |
| Final story version | v1.7 |
| Code commits | 16 (6823a83 → 48e3271) |
| Total findings | 35 |
| Open findings | 0 |
| Adjudicated-accepted | 2 (dead guard, goto-shutdown) |

---

## Pass Timeline

| Pass | Verdict | Findings | Notes |
|------|---------|----------|-------|
| P1 | HAS_FINDINGS | 12 HIGH | First pass — broad initial survey |
| P2 | HAS_FINDINGS | 5 MED | Test-fidelity class opens |
| P3 | HAS_FINDINGS | 5 MED | Doc-sync / FCL-drift class opens |
| P4 | HAS_FINDINGS | 4 LOW | Phantom E-CFG-002; spec + test |
| P5 | HAS_FINDINGS | 3 MED | OS-signal coverage gap (novel axis) |
| P6 | NO_FINDINGS | 0 | Streak 0→1/3 (first clean) |
| P7 | HAS_FINDINGS | 1 LOW | FCL-drift recurrence (reset streak) |
| P8 | HAS_FINDINGS | 2 LOW | Test-strength: vacuous asserts |
| P9 | HAS_FINDINGS | 2 LOW | Consistency-polish class opens |
| P10 | HAS_FINDINGS | 1 LOW | FCL-drift 4th recurrence |
| P11 | HAS_FINDINGS | 1 LOW | FCL-drift 5th recurrence; class-closure escalation |
| P12 | NO_FINDINGS | 0 | Streak 0→1/3 |
| P13 | NO_FINDINGS | 0 | Streak 1→2/3 |
| **P14** | **NO_FINDINGS** | **0** | **Streak 2→3/3 — CONVERGED** |

---

## Finding Class Summary

| Class | Passes | Count | Resolution |
|-------|--------|-------|------------|
| Test-fidelity | P1, P2, P4, P5, P8, P9 | ~15 | Fixed same-burst |
| Doc-sync / FCL-drift | P2, P4, P7, P10, P11 | 5 | Class closed via full-surface sweep at v1.7 |
| Spec-vs-behavior | P4 (phantom E-CFG-002) | 1 | Fixed story v1.3 |
| Governance | P3 (POL-001/002 gap) | 2 | Fixed spec + story |
| Code-comment | P1 (banner) | 1 | Fixed P1 remediation |
| Adjudicated-accepted | P1 (dead guard), P8 (goto-shutdown) | 2 | 5 confirmations each; accepted |

**Severity mix:** 3 MED-initial (P1) → rest LOW/OBS. Zero correctness findings after pass 2. Code
lane unchanged since pass 5 (fa97154 P8 tests; 48e3271 P9 tests were test-only).

---

## Code Lane

```
6823a83  P1 remediation — initial signal wiring
...
8e159f2  P4 remediation — emission format helpers; AC-003 mgmt probe
3c3ce0e  P5 remediation — real-signal smoke; BadConfig fail-path liveness
fa97154  P8 remediation — non-vacuous deep-copy asserts; EC-003 input class
48e3271  P9 remediation — AC-003 no-return assert; transitional-seam doc comments
```

Total: 16 commits across 5 remediation bursts (P1/P2/P3, P4, P5, P8, P9).
Code frozen at 48e3271 after P9; passes P10/P11/P12/P13/P14 found zero code-correctness issues.

---

## Story Version History

| Version | Pass | Change |
|---------|------|--------|
| v1.0 | Elaboration | Initial story |
| v1.1 | P1–P3 | ACs + FCL initial |
| v1.2 | P3 | Wire-contract / drift |
| v1.3 | P4 | Phantom E-CFG-002; test count |
| v1.4 | P7 | FCL row + changelog |
| v1.5 | P9 | AC-004 testenv-extension corrected; 1-arg outline |
| v1.6 | P10 | Test count corrected (10 tests) |
| **v1.7** | **P11** | **Testenv FCL row corrected; full 8-row surface sweep** |

---

## Forward Obligations (anchored to S-7.04-FU-PE-CONNECTOR)

1. Order-sensitive diff semantics ruling — set-equal vs positional (O2, 6th confirm)
2. Mode()-seam construction-time wiring — SetSighupCh transitional seam to be superseded
3. EC-003 literal-code rendering — how error detail appears in PE-mode context (deferred at P3)
4. `upstreamRouters` concurrency design — mutex or channel-update needed when dial loop wires
5. VP-038 full activation — harness-ready at testenv but production activation lands with PE-CONNECTOR

---

## Drift Rows Parked (anchor: S-BL.CLI-SURFACE-COMPLETION)

- **DRIFT-SIGHUP-MODE-ASYMMETRY** — kill -HUP terminates access/console/control modes (default Go
  SIGHUP behavior); only router handles SIGHUP explicitly. UX gap in non-router daemon modes.
- **DRIFT-SIGHUP-INERT-RELOAD-UX** — valid config reload that changes only non-upstream fields
  processes silently; operator receives no confirmation. UX gap for operator workflows.

---

## Process Lessons Codified

1. **Same-burst paired story edits** — any remediation adding/renaming tests requires a paired
   story-writer edit in the same burst. The paired-edit rule prevents FCL-drift recurrence.
2. **Pre-pass count verification** — orchestrator upgraded from count-only to sweep-baseline + per-edit
   row re-verification before each adversary dispatch. Count-only was insufficient (P10 root cause).
3. **Full-surface FCL sweep** — partial reconciliations of a drifting artifact relocate drift; only
   a full-surface verification ends the class. Escalated to full sweep at P11.
4. **Adversary delivery clause** — adversary spawn idleness (API-stall) triggers abandon + resume
   recovery; the adversary's own streak count is verified by the orchestrator at convergence.
5. **Disk-audit-first silent-idle protocol** — if an adversary spawn appears idle, disk-audit the
   output directory before spawning a replacement to avoid double-counting a completed pass.

---

## Awaiting (per-story delivery steps 5–7)

- Demo evidence (record-demo)
- DELIVERY doc (S-7.04-FU-SIGHUP-RELOAD-DELIVERY.md)
- pr-manager dispatch
