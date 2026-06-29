---
document_type: convergence-trajectory
level: ops
version: "1.0"
status: in-progress
producer: state-manager
timestamp: 2026-06-25T00:00:00Z
cycle: cycle-1
inputs: [adversarial-reviews/]
input-hash: ""
traces_to: STATE.md
---

# Convergence Trajectory — cycle-1

## Extracted from STATE.md on 2026-06-25

---

## Phase 1 — Spec Crystallization Adversarial Passes

### Finding Progression

| Pass | Date | Total | CRIT | HIGH | MED | LOW | Verdict |
|------|------|-------|------|------|-----|-----|---------|
| 1 | 2026-06-23 | 27 | 5 | 11 | 9 | 2 | FINDINGS_REMAIN |
| 2 | 2026-06-23 | 18 | 3 | 8 | 6 | 1 | FINDINGS_REMAIN |
| 3 | 2026-06-23 | 17 | 4 | 9 | 3 | 1 | FINDINGS_REMAIN |
| 4 | 2026-06-23 | 21 | 4 | 9 | 6 | 2 | FINDINGS_REMAIN |
| 5 | 2026-06-23 | 17 | 0 | 8 | 7 | 2 | FINDINGS_REMAIN |
| 6 | 2026-06-23 | 14 | 0 | 7 | 6 | 1 | FINDINGS_REMAIN |
| 7 | 2026-06-24 | 7 | 0 | 2 | 4 | 1 | FINDINGS_REMAIN |
| 8 | 2026-06-24 | 9 | 0 | 3 | 5 | 1 | GATE_APPROVED_WITH_DRIFT |

### Trajectory Shorthand

`27 → 18 → 17 → 21 → 17 → 14 → 7 → 9`

Gate disposition: `approve-with-drift` (9 items carried into Phase 2).

### Per-Pass Details

#### Pass 1 (2026-06-23)

**Findings:** 27 (5 CRIT, 11 HIGH, 9 MED, 2 LOW; 3 process-gap tagged)

24 in-cycle addressed (5 critical + 11 high + 7 medium + 1 low); 2 process-gap deferred to upstream (F-025, F-027); 1 low deferred (covered by BA sweep).

Full findings: `.factory/cycles/cycle-1/adversarial-reviews/pass-01.md`

---

#### Pass 2 (2026-06-23)

**Findings:** 18 (3 CRIT, 8 HIGH, 6 MED, 1 LOW; 2 process-gap)

17 in-cycle (3 critical + 8 high + 6 medium addressed); F-019 (1 low) by-design at Phase 1d, deferred to Phase 2 backfill rule.

Full findings: `.factory/cycles/cycle-1/adversarial-reviews/pass-02.md`

---

#### Pass 3 (2026-06-23)

**Findings:** 17 (4 CRIT, 9 HIGH, 3 MED, 1 LOW; 1 process-gap)

All 17 in-cycle addressed (4 critical + 9 high + 3 medium + 1 low); F-P3-018 [process-gap] VP↔BC title-sync check filed upstream.

Full findings: `.factory/cycles/cycle-1/adversarial-reviews/pass-03.md`

---

#### Pass 4 (2026-06-23)

**Findings:** 21 (4 CRIT, 9 HIGH, 6 MED, 2 LOW; 1 process-gap)

Structural consistency audit (post-pass-4): 64 defects across 10 axes; 51 structural (closeable by 2 mechanical sweeps), 13 individual. 64 audit defects addressed mechanically + pass-4 findings F-P4-002, F-P4-008–013 covered by mechanical sweep. F-P4-001 (PRD §7 BC-2.09.003→CAP-028) resolved in round-5. All remaining pass-4 findings closed in round-5: F-P4-001, F-P4-004, F-P4-006, F-P4-014, F-P4-017, F-P4-018. Total pass-4 in-cycle resolution: 20 of 21 (F-P4-019 = stale CAP range in feasibility-report, closed by Sweep 2).

---

#### Pass 5 (2026-06-23)

**Findings:** 17 (0 CRIT, 8 HIGH, 7 MED, 2 LOW)

All 17 pass-5 findings closed across architect + PO refinement (split into 4 small bursts due to API connection drops).

---

#### Pass 6 (2026-06-24)

**Findings:** 14 (0 CRIT, 7 HIGH, 6 MED, 1 LOW)

All 14 pass-6 findings closed across 3 PO bursts + 1 architect burst. Priority drift (4 BCs P1→P0), BC contradiction fixes (BC-2.05.004, BC-2.06.001), error-taxonomy exit codes (E-ADM-011/012/013/014, E-CFG-006), interface-definitions --yes attribution + destructive sbctl svtn ops removal, module-criticality drop-cache placement, BC-2.09.003 DI-007 trace removal, 5 BCs missing VP rows added, BC-2.05.007 phantom sbctl debug removed, ARCH-11 module counts corrected.

---

#### Pass 7 (2026-06-24)

**Findings:** 7 (0 CRIT, 2 HIGH, 4 MED, 1 LOW)

All 7 pass-7 findings closed.

---

#### Pass 8 (2026-06-24)

**Findings:** 9 (0 CRIT, 3 HIGH, 5 MED, 1 LOW)

Trajectory: 27 → 18 → 17 → 21 → 17 → 14 → 7 → 9 — GATE APPROVED with drift (approve-with-drift disposition; 9 items carried into Phase 2).

---

## S-1.02 — Half-Channel Clock Adversarial Passes

### Finding Progression

| Pass | Total | Verdict |
|------|-------|---------|
| 1 | 9 (2 CRIT, 3 HIGH, 2 MED, 2 LOW) | FINDINGS_REMAIN |
| 2 | 11 (0 CRIT, 4 HIGH, 4 MED, 3 LOW) | FINDINGS_REMAIN |
| 3 | 7 (0 CRIT, 1 HIGH, 3 MED, 2 LOW, 1 nitpick) | FINDINGS_REMAIN |
| 4 | 5 | FINDINGS_REMAIN |
| 5 | 4 (1 HIGH AC↔BC mis-anchor + 3 LOW test-quality nits) | FINDINGS_REMAIN |
| 6 | 3 (1 HIGH BC↔story drift, 1 MED AC trace, 1 LOW file structure) | FINDINGS_REMAIN |
| 7 | 0 | CONVERGED (1/3) |
| 8 | 0 | CONVERGED (2/3) |
| 9 | 0 | CONVERGED (3/3) — BC-5.39.001 satisfied |

### Trajectory Shorthand

`9 → 11 → 7 → 5 → 4 → 3 → 0 → 0 → 0`

- Total passes: 9
- Total findings resolved: 39 (5 critical, 8 high, 11 medium, 12 low, 3 nitpick across passes 1-6)
- Worktree HEAD at convergence: 1a6005e on feature/S-1.02-halfchannel-clock
- Clean streak: passes 7, 8, 9

---

## S-2.01 — HMAC Codec Adversarial Passes

### Finding Progression

| Pass | Total | Verdict |
|------|-------|---------|
| 1 | 9 | FINDINGS_REMAIN |
| 2 | 2 | FINDINGS_REMAIN |
| 3 | 4 | FINDINGS_REMAIN |
| 4 | 1 | FINDINGS_REMAIN |
| 5 | 0 | CONVERGED (1/3) |
| 6 | 0 | CONVERGED (2/3) |
| 7 | 1 | REGRESSION — doc citation fix dispatched |
| 8 | 0 | CONVERGED (1/3 restart) |
| 9 | 1 | REGRESSION — File Structure table fix dispatched |
| 10 | 0 | CONVERGED (1/3 restart) |
| 11 | 0 | CONVERGED (2/3) |
| 12 | 0 | CONVERGED (3/3) — BC-5.39.001 satisfied |

### Trajectory Shorthand

`9 → 2 → 4 → 1 → 0 → 0 → 1 → 0 → 1 → 0 → 0 → 0`

- Total passes: 12
- Total findings resolved: 17 (0 critical, 6 high, 5 medium, 6 low across all passes)
- Worktree HEAD at convergence: 9a1ef34 on feature/S-2.01-hmac-codec
- Clean streak: passes 10, 11, 12

Notable mid-flight events:
- drbothen/vsdd-factory#260 family — PO agent unilaterally introduced .factory as tracked gitlink (PR #4); closed without merge; filed as drbothen/vsdd-factory#263
- HKDF KAT requirement initially used self-circular anchor (pass-2); replaced with RFC 5869 §A.1 vector via unexported hkdfSHA256 helper (pass-3)
- AC↔BC trace systematically mis-anchored (pass-4); story rev 4 corrected

---

## S-2.02 — Admission + SVTN Isolation Adversarial Passes

### Finding Progression

| Pass | Total | Commit | Verdict |
|------|-------|--------|---------|
| 1–5 | (see cycle-1/S-2.02/adversary/) | various | FINDINGS_REMAIN |
| 6 | 0 | 276ac85 | CONVERGED (1/3) |
| 7 | 0 | 4f07b90 | CONVERGED (2/3) |
| 8 | 0 | 0313c6f | CONVERGED (3/3) — BC-5.39.001 satisfied |

### Trajectory Shorthand

`(passes 1-5 resolved) → 0 → 0 → 0`

Coverage verified: admission.go (lock discipline, nonce TTL/purge, LWW, deep-clone, distinct sentinels), routing.go (fail-closed admitted-set, SVTN-partitioned table, HMAC tag-snapshot guard), tests (AC-001..007 anchored to BCs; VP-007/008/010/039/057; H-1 race regression; LWW-after-revoke ADR-003).

Process-gap findings across passes 6/7/8: zero. No follow-up codifications required for this streak.

---

## Wave-1 Gate Adversary

| Pass | Total | Verdict |
|------|-------|---------|
| 1 | 4 (2 MED deferrable, 2 LOW) | CONVERGED (single-pass closure with deferrable mediums) |

---

## Refactor PR #3 (FrameType + MTU) Adversary

| Pass | Total | Verdict |
|------|-------|---------|
| 1 | 0 | CONVERGED (1/3) |
| 2 | 0 | CONVERGED (2/3) |
| 3 | 0 | CONVERGED (3/3) — BC-5.39.001 satisfied |

---

## Wave 3 Integration Gate Adversary

Tree: develop @ b68e498 (all 5 Wave 3 stories merged: S-3.04, S-3.01a, S-3.01b, S-3.02, S-3.03)

| Pass | Date | Total | CRIT | HIGH | MED | LOW | OBS | Verdict |
|------|------|-------|------|------|-----|-----|-----|---------|
| 1 | 2026-06-27 | 8 | 0 | 0 | 3 | 2 | 3 | CONVERGED — wave-gate criterion met (0C/0H) |

### Trajectory Shorthand

`8 (0C/0H/3M/2L/3O)` — pass 1 CONVERGED

- MEDIUMs are LATENT (cmd/switchboard/main.go is a version stub; no live production caller)
- All 3 MEDIUMs carry forward to the cmd/switchboard wiring story (mandatory re-gate)
- Full report: `cycles/cycle-1/wave-3/adversary/pass-01.md`

### Finding Summary

| ID | Sev | File | Contract | Status |
|----|-----|------|----------|--------|
| W3-M-1 | MED | internal/routing/routing.go:144-146 | BC-2.05.008 PC-2 | carry-forward → wiring story |
| W3-M-2 | MED | internal/tmux/pty_fallback.go | BC-2.04.001 PC-5 / BC-2.04.002 | carry-forward → wiring story |
| W3-M-3 | MED | internal/session/upstream.go:156-159 | BC-2.05.003 PC-2 | carry-forward → wiring story |
| W3-L-1 | LOW | internal/session/upstream.go:213 | — (verified-inert) | recorded; no action |
| W3-L-2 | LOW | internal/tmux/pty_fallback.go:560-566 | — | carry-forward → wiring story |
| W3-O-1 | OBS | internal/routing/routing.go | BC-2.05.008 EC-006 | architect adjudication pending |
| W3-O-2 | OBS | cmd/switchboard/main.go | — | resolved when wiring story ships |
| W3-O-3 | OBS | internal/session/upstream.go:300 | — | informational |
| W3-PG-001 | [process-gap] | go.md/governance | constructor-default-polarity rule | codification follow-up at cycle-close |

---

## Wave 4 Wave-Level Adversary (6 diverse-lens passes, 2 rounds)

Tree: develop @ abeba27 (all 5 Wave 4 stories merged: S-4.01, S-4.02, S-4.03, S-4.04, S-6.01)

### Round 1 (3 passes)

| Pass | Date | Lenses | CRIT | HIGH | MED | LOW | OBS | Verdict |
|------|------|--------|------|------|-----|-----|-----|---------|
| W4-R1-1 | 2026-06-28 | spec/BC↔AC, security/CWE, concurrency/race | 0 | 0 | 0 | — | — | CONVERGED (1/3) |
| W4-R1-2 | 2026-06-28 | spec/BC↔AC, security/CWE, concurrency/race | 0 | 0 | 0 | — | — | CONVERGED (2/3) |
| W4-R1-3 | 2026-06-28 | spec/BC↔AC, security/CWE, concurrency/race | 0 | 0 | 0 | — | — | CONVERGED (3/3) — BC-5.39.001 satisfied |

### Round 2 (3 passes — fresh context confirmation)

| Pass | Date | Lenses | CRIT | HIGH | MED | LOW | OBS | Verdict |
|------|------|--------|------|------|-----|-----|-----|---------|
| W4-R2-1 | 2026-06-28 | spec/BC↔AC, security/CWE, concurrency/race | 0 | 0 | 0 | — | — | CONVERGED (1/3) |
| W4-R2-2 | 2026-06-28 | spec/BC↔AC, security/CWE, concurrency/race | 0 | 0 | 0 | — | — | CONVERGED (2/3) |
| W4-R2-3 | 2026-06-28 | spec/BC↔AC, security/CWE, concurrency/race | 0 | 0 | 0 | — | — | CONVERGED (3/3) — BC-5.39.001 satisfied |

### Trajectory Shorthand

`6/6 diverse-lens passes C=0/H=0/M=0` — Wave 4 wave-level adversary CONVERGED

---

## Wave 4 Gate Consistency Audit

Audit date: 2026-06-28. Auditor: consistency-validator.

| Finding | Severity | Status |
|---------|----------|--------|
| 14 total findings | CRIT:0 / HIGH:0 / MED:0 / LOW:+ / OBS:+ | All resolved in cycle-close burst |
| L-1: doc hygiene (stale ref + leftover stub docstring) | LOW | RESOLVED — PR #29 (7ef43b8) |
| S403-COS1: stale "encoding/binary" doc ref | OBS | RESOLVED — PR #29 (7ef43b8) |
| S403-COS2: leftover stub docstring | OBS | RESOLVED — PR #29 (7ef43b8) |

**Disposition:** CONDITIONAL PASS — 14 findings, all resolved in cycle-close burst; 0 CRITICAL. Wave gate APPROVED 2026-06-28.
