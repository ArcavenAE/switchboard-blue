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

---

## S-6.06 — Daemon-Side Admin RPC Handlers Adversarial Passes

### Finding Progression (Passes 1–19)

| Pass | Date | Lenses | CRIT | HIGH | MED | LOW | Verdict | Clean Count |
|------|------|--------|------|------|-----|-----|---------|-------------|
| 1–11 | 2026-06-29/30 | various | see burst-log | see burst-log | see burst-log | see burst-log | FINDINGS_REMAIN | 0/3 |
| 12 | 2026-06-30 | correctness/spec/traceability | 0 | 0 | 0 | 0 | CONVERGED (1/3 restart — reset by Pass-10 fix-burst) | reset |
| 13 | 2026-06-30 | correctness/spec/traceability | 0 | 0 | 0 | 0 | FINDINGS_REMAIN (regression check) | reset |
| 14 | 2026-06-30 | correctness/spec/traceability | 0 | 1 | 0 | 0 | BLOCK (F-P14L2-002 HIGH anchor gap) | 0/3 |
| 15 | 2026-06-30 | correctness/spec/traceability | 0 | 0 | 2 | 1 | BLOCK (MEDs after fix) | 0/3 |
| 16 | 2026-06-30 | correctness/spec/traceability | 0 | 0 | 0 | 0 | PASS — CONVERGED (1/3) | 1/3 |
| 17 | 2026-06-30 | correctness/spec/traceability | 0 | 0 | 1 | 1 | BLOCK (F-P17L2-001 MED) | 1/3 |
| 18 | 2026-06-30 | correctness/spec/traceability | 0 | 0 | 2 | 3 | BLOCK (F-P18L1-001/002 MED×2) | 1/3 |
| 19 | 2026-06-30 | correctness/spec/traceability | 0 | 0 | 3 | 2 | BLOCK (F-P19L*-001 dup×3 MED + 2 more MED) | 1/3 |
| 20 | 2026-06-30 | correctness/spec/traceability | 0 | 0 | 1 | 2 | BLOCK (F-P20L3-001 MED NOVEL — cross-layer ordering) | 1/3 |
| 21 | 2026-06-30 | correctness/spec/traceability | 0 | 1 | 4+2 | 5+1 | BLOCK (L3: F-P21L3-001 HIGH EC-008 stale; L1: 4 MED impl; L2: 2 MED VP-INDEX stale) | 1/3 |
| 22 | 2026-06-30 | correctness/spec/traceability | 0 | 2 | 2 | 0 | BLOCK (L3: F-P22L3-001/002 HIGH×2 "unconditionally" residuals; F-P22L3-003/004 MED×2 VP-076; L1+L2: PASS CLEAN) | 1/3 |
| 23 | 2026-06-30 | correctness/spec/traceability | 0 | 0 | 2 | 1 | BLOCK (L3: F-P23L3-001/002 MED×2 stale v1.10 cites in story lines 180+245; L1+L2: PASS CLEAN) | 1/3 |
| 24 | 2026-06-30 | correctness/spec/traceability | 0 | 0 | 1 | 0 | BLOCK (L3: F-P24L3-001 MED VP-076 v3.8 cite; L1+L2: PASS CLEAN) | 1/3 |
| 25 | 2026-06-30 | correctness/spec/traceability | 0 | 0 | 1 | 4 | BLOCK (L3: F-P25L3-001 MED story VP-076 v1.1 cite; L1: 4 LOW OBS; L2: PASS CLEAN) | 1/3 |
| 26 | 2026-06-30 | correctness/spec/traceability | 0 | 0 | 0 | 7+2 | PASS CLEAN (all 3 lenses; 7 LOW non-defect OBS L1 + 2 LOW out-of-scope OBS L3 → phase-5) | 2/3 |
| 27 | 2026-06-30 | correctness/spec/traceability | 0 | 0 | 0 | 7+0+0 | PASS CLEAN (all 3 lenses; L1: 7 LOW non-blocking OBS; L2: novelty LOW; L3: novelty ZERO) | 3/3-pending |
| 28 | 2026-06-30 | impl-internal/spec-impl/sibling-prop | 0 | 0 | 0 | 0 | PASS CLEAN (all 3 lenses; novelty NONE/ZERO/ZERO) — **CONVERGENCE-CLOSED** | 3/3 CLOSED |

### Trajectory Shorthand (Pass 16 onward, clean-pass tracking)

`16:PASS(1/3) → 17:BLOCK → 18:BLOCK → 19:BLOCK → 20:BLOCK → 21:BLOCK → 22:BLOCK → 23:BLOCK → 24:BLOCK → 25:BLOCK → 26:PASS(2/3) → 27:PASS(3/3-pending) → 28:PASS(3/3✓CLOSED)`

**BC-5.39.001 CONVERGENCE-CLOSED** after Pass-28. Clean-pass count: **3/3 CLOSED** (Pass-16 baseline + Pass-26 + Pass-27 + Pass-28). Third consecutive fully-clean pass. No fix-burst required. Spec tip at convergence: factory-artifacts HEAD (a6cdb88 lineage). Impl tip at convergence: d3f186c (unchanged since Pass-25).

### Pass-26 Details

**Date:** 2026-06-30
**Dispatch IDs:** lens-1 a05e401bf6bf753a1 / lens-2 a9efc33989be3c792 / lens-3 ae6b9da5fbadbaaba
**Spec tip dispatched against:** a6cdb88. **Impl tip:** d3f186c.

**Lens-1 (a05e401bf6bf753a1):** PASS CLEAN — novelty NONE. 7 LOW observations, all adjudicated as non-defects (mis-labels, intentional design, fail-closed behavior, dead code in test). No gating findings.

**Lens-2 (a9efc33989be3c792):** PASS CLEAN — novelty NONE. All wire-error strings byte-equivalent. ARCH-04 v1.13 + VP-076 v1.4 cites coherent. Sibling-sweep gap closed. No findings.

**Lens-3 (ae6b9da5fbadbaaba):** PASS CLEAN — novelty LOW. 2 LOW observations, explicitly out-of-scope (architectural / system-level), deferred to phase-5:
- O-P26L3-001 LOW: ARCH-04.md:30-40 modified-list non-monotonic + missing v1.7/v1.8/v1.11/v1.12 + v1.13 inserted before v1.9.
- O-P26L3-002 LOW: error-taxonomy.md:9-23 modified-list mixed ascending/descending ordering.

Both observations are out-of-perimeter for S-6.06 per-story scope per BC-5.39.002 PC2. Created as TaskList #117 (phase-5 routing).

**Verdict:** PASS CLEAN. Clean-pass count advances: 1/3 → **2/3**.

**No fix-burst required.** Pass-27 queued (clean-pass attempt #3 of 3). Spec tip: post-closeout SHA on factory-artifacts. Impl tip: d3f186c (unchanged).

---

### Pass-27 Details

**Date:** 2026-06-30
**Dispatch IDs:** lens-1 a68ef99c2850a5ae5 / lens-2 ad7f415313ffdd259 / lens-3 a73b40208a7fef653
**Spec tip dispatched against:** factory-artifacts HEAD (post-Pass-26 closeout). **Impl tip:** d3f186c (unchanged since Pass-25).

**Lens-1 (a68ef99c2850a5ae5):** PASS CLEAN — novelty LOW. 7 LOW non-blocking observations, all adjudicated non-blocking refinements:
- O-1 LOW: keyFingerprintAdmin(nil) latent footgun in mapAdminError list-keys path.
- O-2 LOW: decodePublicKey not validating Ed25519 point encoding.
- O-3 LOW: RoleMismatchError typed-detail path not covered by TestMapAdminError_ErrorWrapping.
- O-4 LOW: E-ADM-018 omits fingerprint — intentional per AC-005 (design decision).
- O-5 LOW: dead privHex variable in VP046 DI-002 test.
- O-6 LOW: goroutine accounting assertion in TestSVTNManager_ExpireKey_TOCTOU_RoleChangeRace.
- O-7 LOW: subtle.ConstantTimeCompare doc-comment accuracy.
All 7 routed to TaskList #115 (post-merge polish backlog). No gating findings.

**Lens-2 (ad7f415313ffdd259):** PASS CLEAN — novelty LOW. All wire-error strings byte-aligned; all version cites resolve coherently; layering claim corroborated against implementation. Adversary explicitly recommends Lens-2 streak counter advancement to 3/3.

**Lens-3 (a73b40208a7fef653):** PASS CLEAN — novelty ZERO. Pass-25 sibling-fix propagation has fully landed across all surfaces. Phase-5 deferred items (TaskList #118) correctly NOT re-flagged per BC-5.39.002 PC2.

**Verdict:** PASS CLEAN — second consecutive fully-clean pass. Clean-pass count advances: 2/3 → **3/3-pending**.

**No fix-burst required.** Pass-28 = convergence-close (clean-pass #3 of 3). Spec tip: factory-artifacts HEAD. Impl tip: d3f186c (unchanged).

---

### Pass-19 Details

**Lens-1:** PASS (6 LOW informational, non-gating) + dup-confirmed F-P19L*-001 MED (BC body VP table missing VP-076).
**Lens-2:** BLOCK — F-P19L*-001 MED (VP table) + F-P19L2-002 LOW (E-ADM-021 line cite 275-280→279-284).
**Lens-3:** BLOCK — F-P19L*-001 MED (VP table) + F-P19L3-002 MED (Traceability Stories missing EC-007/S-6.06) + F-P19L3-003 MED (modified-list non-monotonic).

All 5 gating findings (4 MED + 1 LOW) are spec-only, no impl changes needed. Root cause: Pass-18 fix-burst sibling-fix propagation gap — VP-076 minted in BC-2.05.004 v1.10 but three sibling locations within the same document not updated.

**Fix-burst commits:** 13164cb (BC-2.05.004 v1.10→v1.11 + BC-INDEX v1.6→v1.7) + 9843e9a (S-6.06 v1.16→v1.17 + STORY-INDEX v3.6→v3.7).
**Spec tip after fix:** 9843e9a. **Impl tip:** 6bd9e12 (unchanged).

---

### Pass-20 Details

**Date:** 2026-06-30
**Dispatch IDs:** lens-1 a0ce4060b99958c55 / lens-2 a8eaa3d24878b1fc8 / lens-3 a14728dee74678c40
**Spec tip dispatched against:** 9843e9a. **Impl tip:** 6bd9e12 (unchanged).

**Lens-1 (a0ce4060b99958c55):** PASS CLEAN — 2 MED + 1 LOW non-blocking polish observations only (non-gating).
**Lens-2 (a8eaa3d24878b1fc8):** PASS CLEAN — no gating findings.
**Lens-3 (a14728dee74678c40):** BLOCK — F-P20L3-001 MED NOVEL: cross-layer ordering ambiguity. Handler TTL validation at admin_handlers.go:279-284 fires BEFORE svtnmgmt bootstrap guard, so `{bootstrap_pubkey, after:"-1h"}` returns E-CFG-001 not E-ADM-021, contradicting BC EC-007 "unconditionally" language.

**Verdict:** BLOCK. Clean-pass count: 1/3 (unchanged — baseline Pass-16).

**Novelty note:** F-P20L3-001 is genuinely new — Passes 1–19 examined symmetry, guard position, and TTL bounds in isolation but never the cross-product of (bootstrap target × malformed input). Real convergence dividend.

**Product-owner ruling:** Option B (spec narrowing). Input validation precedes business-rule sentinels — current impl is correct, BC/VP wording was overstated. Mutation-prevention invariant preserved either way.

**Fix-burst commit:** 677140a — BC-2.05.004 v1.11→v1.12 (EC-007 narrowed to well-formed requests) + VP-076 v1.0→v1.1 (Property #3 scoped to well-formed) + BC-INDEX v1.7→v1.8 + error-taxonomy.md O-P20L3-001 fix (E-ADM-021 Tests citation cleanup, removed revoke test reference).

**Spec tip after fix:** 677140a. **Impl tip:** 6bd9e12 (unchanged).

---

### Pass-21 Details

**Date:** 2026-06-30
**Dispatch IDs:** lens-1 ada1125598286af4e / lens-2 a19f659c98fb7441a / lens-3 a27279f4b0c6808f3
**Spec tip dispatched against:** 677140a. **Impl tip:** 6bd9e12.

**Lens-1 (ada1125598286af4e):** BLOCK — 4 MED + 5 LOW.
- F-L1-A MED: mapAdminError default-arm untested
- F-L1-B MED: ErrInvalidDuration unreachable-claim has no DI-D arm
- F-L1-C MED: decodePublicKey silent swallow (go.md rule 3 violation)
- F-L1-D MED: TestResolveAndVerifyCallerRole expired_key_non_control_rejected mis-anchored; future-expiry-non-control branch uncovered
- 5 LOW informational

**Lens-2 (a19f659c98fb7441a):** BLOCK — 2 MED.
- F-P21L2-001 MED: dup-confirmed lens-3 EC-008 narrowing gap (same root cause as F-P21L3-001)
- F-P21L2-002 MED NEW: VP-INDEX VP-076 row + registry note still cite "unconditionally"/v1.10 (stale post Pass-20 Option-B)

**Lens-3 (a27279f4b0c6808f3):** BLOCK — 1 HIGH + 1 MED [process-gap] + 1 LOW.
- F-P21L3-001 HIGH: EC-008 stale "unconditionally" — sibling-fix propagation gap from Pass-20 Option-B narrowing (BC-2.05.004 v1.12 updated EC-007 but EC-008 not swept)
- F-P21L3-002 MED [process-gap]: BC EC narrowing not fanned out to story EC tables; recurring pattern (passes 19, 20, 21)
- O-P21L3-002 LOW: VP-076 stale v1.10 cite at line 68

**Verdict:** BLOCK. All 3 lenses blocked. Clean-pass count: 1/3 (unchanged).

**Substantive vs cosmetic assessment (convergence-reset question):**
Impl changes were defense-in-depth / test-quality only (mapAdminError signature refactored to eliminate double-decode + silent swallow; ErrInvalidDuration DI-D arm added; test renamed + TTL changed to cover previously-uncovered branch). No behavioral semantics changed. Orchestrator ruling: counter not reset. Pass-22 = clean-pass attempt #2 of 3.

**Fix-burst commits:**
- Spec (factory-artifacts): fc90ef2 (VP-INDEX v2.10→v2.11, VP-076 v1.1→v1.2) + 4229464 (S-6.06 v1.17→v1.18 EC-008 narrowed, STORY-INDEX v3.7→v3.8)
- Impl (feat/S-6.06): c519fc1 (F-L1-D test fix) + 0be8e97 (F-L1-A/B/C mapAdminError refactor + ErrInvalidDuration arm)

**Spec tip after fix:** 4229464. **Impl tip:** 0be8e97.

---

### Pass-22 Details

**Date:** 2026-06-30
**Dispatch IDs:** lens-1 aeaa638b208bc006a / lens-2 a72e3013057bcc11b / lens-3 a5eef7adde2c2635e
**Spec tip dispatched against:** 4229464. **Impl tip:** 0be8e97.

**Lens-1 (aeaa638b208bc006a):** PASS CLEAN — no gating findings.
**Lens-2 (a72e3013057bcc11b):** PASS CLEAN — no gating findings.
**Lens-3 (a5eef7adde2c2635e):** BLOCK — 2 HIGH + 2 MED + 1 [process-gap].
- F-P22L3-001 HIGH: story VP table row for VP-076 still cites EC-007/EC-008 "unconditionally" language (Pass-21 fix-burst narrowed BC and VP-076 body but story VP table row was not regenerated).
- F-P22L3-002 HIGH: error-taxonomy.md E-ADM-020/E-ADM-021 entries still carry "unconditionally...at any time" text and stale v1.10 BC-2.05.004 citation (Pass-20/21 bursts updated BC and VP-076 but not error-taxonomy entry text).
- F-P22L3-003 MED: VP-076 Property #1 and Property #2 prose still uses unnarrowed language (v1.2 updated Property #3 only).
- F-P22L3-004 MED: VP-076 proof-harness docstring inconsistent with narrowed scope.
- O-P22L3-002 [process-gap]: recurring 4-pass sweep miss pattern now fully documented; vsdd-factory issues #361–#364 filed.

**Verdict:** BLOCK. Clean-pass count: 1/3 (unchanged — baseline Pass-16).

**Convergence-reset ruling:** Fix-burst 4b42dd5 was spec-only (error-taxonomy + VP-076 + S-6.06 story + index updates). No behavioral semantics changed in impl. Orchestrator ruling: counter not reset per BC-5.39.001. Pass-23 = clean-pass attempt #2 of 3.

**Fix-burst commit:** 4b42dd5 — error-taxonomy.md v3.8→v3.9 (E-ADM-020/021 text updated, stale v1.10 cites removed) + VP-076 v1.2→v1.3 (Properties #1 & #2 narrowed + harness docstring) + S-6.06 v1.18→v1.19 (story VP table row regenerated) + VP-INDEX v2.11→v2.12 + STORY-INDEX v3.8→v3.9. Post-fix grep confirms zero current-state "unconditionally" residuals across specs/ + stories/.

**vsdd-factory upstream issues filed:**
- #361 — BC EC sibling-fix propagation gap (systematic fix-burst sweep discipline)
- #362 — VP-INDEX row description drift when VP body narrows
- #363 — Test-writer policy: negative tests for "unreachable in practice" default arms
- #364 — Adversary policy: detect test name/assertion semantic-anchoring drift

**Spec tip after fix:** 4b42dd5. **Impl tip:** 0be8e97 (unchanged).

---

### Pass-23 Details

**Date:** 2026-06-30
**Dispatch IDs:** lens-1 afd8f2e1b20cde42a / lens-2 aea17b5f734310b26 / lens-3 a1038b24343e5e306
**Spec tip dispatched against:** 4b42dd5. **Impl tip:** 0be8e97.

**Lens-1 (afd8f2e1b20cde42a):** PASS CLEAN — novelty LOW; impl tip 0be8e97 unchanged; no findings.

**Lens-2 (aea17b5f734310b26):** PASS CLEAN — 1 LOW non-blocking observation only.
- O-P23L2-001 LOW: VP-076 Source Contract section line 113 cites error-taxonomy.md v3.8; current is v3.9. Semantically coherent narrowing, paperwork drift only. Deferred to next VP-076 touch.

**Lens-3 (a1038b24343e5e306):** BLOCK — 2 MED + 1 [process-gap].
- F-P23L3-001 MED: S-6.06 v1.19 line 180 Error Code Map E-ADM-021 row cites `BC-2.05.004 EC-007 v1.10`; should be v1.12 (narrowed in Pass-20 Option-B fix-burst).
- F-P23L3-002 MED: S-6.06 v1.19 line 245 Task 12 Refs cites `BC-2.05.004 EC-007 v1.10`; should be v1.12.
- O-P23L3-001 LOW: VP-076 Property #1/#2 phrasing slightly tautological — noted, non-blocking.

**Verdict:** BLOCK. Clean-pass count: 1/3 (unchanged — baseline Pass-16).

**Process-gap codification — PROCESS-GAP-P23 (5th consecutive recurrence):**
Sibling-sweep gap has now recurred across Passes 19, 21, 22, 22-stragglers, and 23. Pattern: BC version-narrowing sweep updates BC body + VP body + index files + error-taxonomy, but misses story-body prose narrative (Error Code Map message annotations + Task references). Pass-22 grepped for "unconditionally" but did NOT grep for "v1.10" residuals. Refines and extends PROCESS-GAP-P21. vsdd-factory #361 comment appended (5th recurrence as additional evidence).

**Fix-burst commit:** 82721dc (product-owner) — S-6.06 v1.19→v1.20 + STORY-INDEX v3.9→v3.10. Both v1.10 cites at lines 180 and 245 bumped to v1.12. Exhaustive pre-edit + post-edit grep across BC/VP-076/VP-INDEX/error-taxonomy confirms zero current-state-narrative v1.10 residuals. ARCH-04 v1.10 cites at lines 263, 332 correctly left alone (different artifact, different version space). Changelog rows correctly left alone (historical-state descriptions).

**Convergence-reset ruling:** Spec-only fix (no impl change; 82721dc touches only S-6.06 story + STORY-INDEX). Per BC-5.39.001 spec-only-fix discipline, clean-pass counter does NOT reset. Pass-24 = clean-pass attempt #3 of 3 continues.

**Spec tip after fix:** 82721dc. **Impl tip:** 0be8e97 (unchanged).

---

### Pass-24 Details

**Date:** 2026-06-30
**Dispatch IDs:** lens-1 a6ead8d7956498972 / lens-2 a64e9dbb012bf369a / lens-3 a57d7569f4aaa7675
**Spec tip dispatched against:** 82721dc. **Impl tip:** 0be8e97.

**Lens-1 (a6ead8d7956498972):** PASS CLEAN — novelty LOW; no findings; impl tip 0be8e97 unchanged.

**Lens-2 (a64e9dbb012bf369a):** PASS CLEAN — 1 LOW advisory observation only.
- O-P24L2-001 LOW (out-of-scope): impl comment v1.10 cites at svtnmgmt.go:66,:332 + admin_handlers_test.go:821 — same axis as F-P24L3-001 but surfaced advisory; resolved by impl fix-burst 4b626cf.

**Lens-3 (a57d7569f4aaa7675):** BLOCK — 1 MED + 1 [process-gap] OBS.
- F-P24L3-001 MED: VP-076.md:113 Source Contract cited error-taxonomy.md v3.8; current version is v3.9. Root cause: Pass-22 fix-burst (4b42dd5) bumped error-taxonomy v3.8→v3.9 and VP-076 v1.2→v1.3 in the same commit but forgot to update VP-076's back-reference at line 113.
- O-P24L3-001 OBS [process-gap]: 6th-pass cite-drift recurrence — axis shifted to downstream-doc cite of upstream-doc version; new surface: impl source comments.

**Verdict:** BLOCK. Clean-pass count: 1/3 (unchanged — baseline Pass-16).

**Process-gap codification — PROCESS-GAP-P24 (6th consecutive recurrence):**
New axis: downstream-doc cite of upstream-doc version (VP-076 Source Contract cited error-taxonomy v3.8 after Pass-22 fix-burst bumped error-taxonomy to v3.9 and VP-076 to v1.3 in the same commit but missed VP-076's back-reference). New surface: impl source comments (svtnmgmt.go + admin_handlers_test.go v1.10 cite residuals). vsdd-factory #361 comment appended (6th recurrence).

**Convergence-reset ruling:** Doc-only + comment-only fix-bursts; no behavior changes. Per BC-5.39.001 doc-only-fix discipline, clean-pass counter NOT reset. Pass-25 = clean-pass attempt #3 of 3 continues.

**Fix-burst commits:**
- Spec (factory-artifacts): c5c948c — VP-076 v1.3→v1.4: line 113 v3.8→v3.9 cite fix; VP-INDEX v2.12→v2.13; pre/post-edit grep clean.
- Impl (feat/S-6.06-daemon-admin-handlers): 4b626cf — impl comment v1.10→v1.12 at 3 sites (svtnmgmt.go:66,:332 + admin_handlers_test.go:821); just fmt + just lint clean; just test-race 17/17 PASS, 0 races; comment-only, no behavior change. O-P24L2-001 also resolved.

**Spec tip after fix:** c5c948c. **Impl tip:** 4b626cf.

---

### Pass-25 Details

**Date:** 2026-06-30
**Dispatch IDs:** lens-1 ab521edc560a0b013 / lens-2 aae0edcaf3acf4640 / lens-3 a9a23dc563641c905
**Spec tip dispatched against:** c5c948c. **Impl tip:** 4b626cf.

**Lens-1 (ab521edc560a0b013):** PASS CLEAN — 4 LOW observations (non-gating).
- Obs-1 LOW: fallback-path coverage gap in resolveAndVerifyCallerRole — no-pubkey-in-ctx path untested; → TaskList #115.
- Obs-2 LOW: 3 stale ARCH-04 v1.10 cites in impl (admission.go:287, svtnmgmt.go:252, svtnmgmt.go:279) + 1 in story; adjudicated as intentional historical attribution (S-2.01:148 out-of-scope per PO).
- Obs-3 LOW: unreachable bogus fingerprint in list-keys default arm.
- Obs-4 LOW: dead code in VP046 test.

**Lens-2 (aae0edcaf3acf4640):** PASS CLEAN — novelty zero; no findings.

**Lens-3 (a9a23dc563641c905):** BLOCK — 1 MED finding + 1 [process-gap] OBS.
- F-P25L3-001 MED: S-6.06:204 cites "VP-076 v1.1"; current is v1.4. Stale version citation in story body.
- O-P25L3-001 OBS [process-gap]: 7th-recurrence sibling-sweep gap — new axis: downstream→upstream version cites (story body cites of upstream-artifact versions become stale after upstream version bumps). Mirror of PROCESS-GAP-P21/P23/P24 mechanism. Pass-24 fix-burst (c5c948c) updated VP-076 v1.3→v1.4 but did NOT sweep stories/ for "VP-076 v1.*" current-state cites.

**Verdict:** BLOCK. Clean-pass count: 1/3 (unchanged — baseline Pass-16). Pass-25 NOT counted.

**Process-gap codification — PROCESS-GAP-P25 (7th consecutive recurrence):**
Story body cites of upstream-artifact versions are stale after upstream version bumps. Pass-24 fix-burst (c5c948c) updated VP-076 v1.3→v1.4 but did NOT sweep stories/ for "VP-076 v1.*" current-state cites. Pattern mirrors PROCESS-GAP-P21/P23/P24 on yet another axis: downstream→upstream version cites. Upstream-rooted sweep rule: any document citing an artifact must be re-grepped when that artifact's version bumps. vsdd-factory #361 comment appended (7th recurrence + new axis).

**Convergence-reset ruling:** Both fix-bursts were doc-only / comment-only; no behavior changes. Per BC-5.39.001 doc-only-fix discipline, clean-pass counter NOT reset. Pass-26 = clean-pass attempt #3 of 3 continues.

**Note:** Obs-2 ARCH-04 v1.10 cites — S-2.01:148 cites ARCH-04 v1.1 — adjudicated as out-of-scope historical-attribution by PO; intentional, not part of this fix.

**Fix-burst commits:**
- Spec (factory-artifacts): a6cdb88 — S-6.06 v1.20→v1.21 + STORY-INDEX v3.10→v3.11; line 204 VP-076 v1.1→v1.4; line 263 ARCH-04 v1.10→v1.13; exhaustive pre/post-edit grep across stories+specs; zero (b)-class residuals remain.
- Impl (feat/S-6.06-daemon-admin-handlers): d3f186c — 4 impl/test ARCH-04 v1.10→v1.13 comment bumps at admission.go:287, svtnmgmt.go:252, svtnmgmt.go:279, admin_handlers.go:192; just fmt + just lint clean; just test-race 17/17 PASS, 0 races; comment-only, no behavior change.

**Spec tip after fix:** a6cdb88. **Impl tip:** d3f186c.

---

### Pass-28 Details

**Date:** 2026-06-30
**Dispatch IDs:** 3 fresh-context diverse-lens adversary passes (convergence-close)
**Spec tip dispatched against:** factory-artifacts HEAD (post-Pass-27 closeout). **Impl tip:** d3f186c (unchanged since Pass-25).

**Lens-1 (impl-internal):** PASS CLEAN — novelty NONE. All 7 sentinel arms covered, default arm covered, %w wrapping verified, UTC discipline verified, no locked-accessor leaks, no init()/panic violations outside main, no tautological tests, comprehensive negative-path coverage, no hidden allocations, no sentinel-vs-wire drift, race/TOCTOU regression tests intact.

**Lens-2 (spec↔impl drift):** PASS CLEAN — novelty ZERO. Wire-error verbatim consistency verified; layering claim (handler input-validation before bootstrap sentinel) verified at admin_handlers.go:279-284 + svtnmgmt.go:325/334/263/268; all version cites coherent (VP-076 v1.4, ARCH-04 v1.13, BC-2.05.004 v1.12, error-taxonomy v3.9); VP-INDEX arithmetic 76 total; bidirectional traceability confirmed.

**Lens-3 (within-doc/sibling-prop):** PASS CLEAN — novelty ZERO. All five mandatory sweeps clean; Pass-25 sibling-fix propagation fully landed; known phase-5-deferred items (TaskList #118) correctly not re-flagged per BC-5.39.002 PC2.

**Verdict:** PASS CLEAN — THIRD consecutive fully-clean pass. **BC-5.39.001 CONVERGENCE-CLOSED.**

**No fix-burst required.** Spec tip at convergence: factory-artifacts HEAD. Impl tip at convergence: d3f186c.

---

## Phase 5 — Adversarial Refinement Passes

### Finding Progression

| Pass | Date | Adv-A | Adv-B | Streak | develop_tip |
|------|------|-------|-------|--------|-------------|
| 1 | 2026-07-02 | HAS_FINDINGS (2H/1M/0L/2obs) | HAS_FINDINGS (1H/2M/1L/2obs) | 0/3 | 4659cb8 (annotated) |
| 2 | 2026-07-02 | HAS_FINDINGS (1M/2L) | HAS_FINDINGS (2M) | 0/3 | dc51b06 (annotated) |
| 3 | 2026-07-02 | HAS_FINDINGS (3H/4M/2L/3obs) | HAS_FINDINGS (0H/1M/2L/3obs) | 0/3 | c76a8d5 (rem) |
| 4 | 2026-07-03 | HAS_FINDINGS (3H/5M/2L) | HAS_FINDINGS (2H/2M) | 3/3 SATISFIED (passes 17/18/19) | cbd0272 |
| 5 | 2026-07-03 | HAS_FINDINGS (0H/2M/2L/1obs) | HAS_FINDINGS (0H/2M/1L/1obs) | 0/3 (streak reset) | cbd0272 |
| 6 | 2026-07-03 | HAS_FINDINGS (1H/4M/1L) | CLEAN (0/0/0+2obs) | 0/3 | d012dbf |
| 7 | 2026-07-03 | HAS_FINDINGS (0H/3M/0L+1obs) | CLEAN (0/0/0+5obs) | 0/3 | 4d7d9e0 |
| 8 | 2026-07-03 | HAS_FINDINGS (2H/4M/1L) | HAS_FINDINGS (0H/2M+1obs) | 0/3 | 4d7d9e0 |
| 9 | 2026-07-03 | HAS_FINDINGS (1H/2M/3L+3obs) | CLEAN (0/0/0+3obs) | 0/3 | 32ea461 |
| 10 | 2026-07-03 | HAS_FINDINGS (1H/1M) | HAS_FINDINGS (0H/0M/1L+2obs) | 0/3 | 32ea461 |
| 11 | 2026-07-03 | HAS_FINDINGS (1H/1M/3obs) | CLEAN (0/0/0+3obs) | 0/3 | 66e9ddc |
| 12 | 2026-07-03 | HAS_FINDINGS (0H/2M/2obs) | CLEAN (0/0/0+3obs) | 0/3 | 66e9ddc |

**Pass 5 notes:** Adv-B self-reported files_read 7 vs read_cap 6 (overage disclosed). BC-5.39.001 streak reset to 0/3. Pass 5 remediation pending (Burst 21).

**Integrity note (Pass 5 Adv-B):** files_read 7 vs read_cap 6 — overage self-disclosed by adversary for admin_interactive_prompt_test.go io.Pipe seam; rationalized in report.

**Pass 6 notes:** Burst 23 code+spec remediation (PR #65 4d7d9e0; interface-definitions v1.19; BC-2.07.002 v1.9; S-6.03 v2.8). Streak 0/3.

**Pass 7 notes:** Burst 25 code-only remediation (PR #66 b4ccd06; usageErrf sweep). Streak 0/3.

**Pass 8 notes:** Burst 27 code+spec remediation (PR #67 32ea461; interface-definitions v1.20). Streak 0/3.

**Pass 9 notes:** Burst 29 spec-only remediation (interface-definitions v1.21; all findings documentation-side; zero code changes). Streak 0/3.

**Pass 10 notes:** Burst 31 code+spec remediation (PR #68 66e9ddc; interface-definitions v1.22; phantom --at→--after corrected; E-CFG-001 exit-class split). Streak 0/3.

**Pass 11 notes:** Burst 33 spec-only remediation (interface-definitions v1.23; §131 revoke carve-out from runDestroyConfirmGate family; §137 scoping to svtn destroy + key register + admin recover; §109 --role REQUIRED syntax). Both adversaries disclosed read-cap overages (A: 7/6, B: 8/6). Both findings adjudicated spec-side (taxonomy v4.4 + E-ADM-018 already ruled the bool-confirm shape; §109 syntax row was simply missing the flag). Zero code changes. Streak 0/3; Pass 12 next.

**Pass 12 notes:** Burst 35 spec-only remediation (interface-definitions v1.24; §111 list-keys exit-code column extended with E-SVTN-003 + E-CFG-001; `--svtn <id>` → `--svtn <svtn-name>` placeholder sweep across §108/§109/§110/§130 recover; §108/§120 confirm-family flag consistency touch). Both findings adjudicated spec-side — list-keys was outside the register/revoke/expire audit umbrella; placeholder class error, not a code defect (orchestrator verified name-keying at svtnmgmt.go:254/300/370). Adv-B disclosed files_read 7 vs read_cap 6 (overage self-disclosed). Third consecutive zero-code-defect pass (P10/P11/P12). Streak 0/3 (Adv-A HAS_FINDINGS resets); Pass 13 next.

---

## S-7.04-FU-PE-CONNECTOR — Adversarial Pass 14 (2026-07-08)

**Verdict:** HAS_FINDINGS — 1 LOW [doc-drift]

**Code HEAD:** 670c64b (advanced from 0a350d6 — two remediation commits required)

### Finding F-P14-001 LOW [doc-drift]

**What:** runRouter's doc header in `mgmt_wire.go` still declared the PE connector "deferred to a follow-on story" while this story's body actually wires it. Three false ships-later claims were inherited from the develop base. This is the first finding in the cycle inherited from the base rather than introduced by a story or its remediations. The semantic-accuracy axis is orthogonal to all prior bars: no line citations were wrong, all symbols resolve, but the prose claims were factually false.

**Novel failure axis — semantic accuracy of prose claims:** Passes P1–P13 verified citation correctness (symbols resolve, line numbers accurate, version pins current) and absence-assertion fidelity. None examined whether the prose claims about shipped/deferred state were accurate given what this story actually delivers. A doc header authored correctly at base ("PE connector ships in a follow-on story") became false the moment this story wired the connector.

**Remediation:**
- 34e51d6: `#DEFERRED` comment block split — PE-CONNECTOR claim changed to `#SHIPPED`, DRAIN-WIRE deferral preserved. Symbol-resolution bar applied.
- 670c64b (opportunistic): go.md rule-7 violation fixed — `dialLoop` + 4 `testenv` exports reordered `ctx`-first; 33 call sites across 11 files updated. PERIMETER EXPANDS 7 → 14 files (7 VP e2e test files in other packages received mechanical call-site updates). Full repo suite green, full CI gate cleared after both commits.

**Story sync → v1.14:** FCL row 13 added for perimeter expansion; changelog row covering both commits; co-reference sweep fixed 2 live story occurrences of old `NewWithRouters` arg order.

**P14 verification results:**
- Full CI gate: golangci-lint 0 issues, go vet clean, race tests green, gofumpt no diffs.
- Census re-derivation: SET diff vs toolchain = ∅ (no new unregistered packages).
- Absence-assertion audit: CLEAN — `TestScanForLine_DetectsEFWD001ProductionEmission` (P11 fix) still passes.
- Symbol-resolution bar: all cited symbols verified.
- POL-002 sync: PASS — story v1.14 registered in STORY-INDEX v4.20→v4.21.
- All P1-P13 fixes verified holding.
- Streak 0/3 (HAS_FINDINGS resets).

**Trajectory shorthand (P1–P14):** 7/3/3/1/1/2/2/1/1/1/1/1/1/1

**Cycle ledger:** 14 passes, 26 findings (7/3/3/1/1/2/2/1/1/1/1/1/1/1), all fixed/adjudicated, zero open. Streak 0/3. Awaiting: adversary pass 15 @ 670c64b.

---

## S-7.04-FU-PE-CONNECTOR — Adversarial Pass 15 (2026-07-08)

**Verdict:** HAS_FINDINGS — 1 LOW [test-fidelity]

**Code HEAD:** 79c1284 (advanced from 670c64b — one remediation commit required)

### Finding F-P15-001 LOW [test-fidelity]

**What:** `TestRunRouter_PE_RouterHandleModeReflectsLiveState`'s ModeE-fake inverse-delegation setup was orphaned — `fakeConnE` was wired but never observed before `Restart` discarded it; the final assertion was satisfied by the live connector's failed dial; the comment misattributed the mechanism `"via fakeConnE"`. Proven by mutation: flipping the fake's return value still produced a passing test.

**Novel failure axis — orphaned double + misattributed mechanism:** This is the third distinct shape in the "comment claims a code path the test doesn't exercise" family: P11 surfaced a vacuous key (absence assertion never matched production string), P13 surfaced a phantom symbol (anchor text cited a non-existent function), and P15 surfaces an orphaned fake (double wired but not observed before teardown/restart discarded it, with the passing assertion actually satisfied by an independent mechanism). In all three cases the test's comment accurately described the intended verification, but the runtime path diverged — silently in every case.

**Adversary's empirical proof:** Mutation of `fakeConnE`'s return value produced no test failure; the passing assertion was driven by the live connector's failed dial (EC-001 ctx-cancel path), not the fake's inverse-delegation. The comment's claim `"via fakeConnE"` was therefore false.

**Remediation at code commit 79c1284:**
- New mutation-pinned inverse-delegation assertion: verifies that `fakeConnE` was actually called and that its return value shaped the result.
- Comment reattributed to the live failed-dial mechanism, no longer claiming `fakeConnE` as the driver.
- Full CI gate cleared (golangci-lint 0 issues, go vet clean, race tests green, gofumpt no diffs).

**Story sync → v1.15:** AC-006 test-names bullet updated; test-surface table row updated for the strengthened coverage; changelog row added; co-reference sweep adjudicated 8 hits (all correct historical records or correct live prose).

**P15 verification results:**
- Full CI gate: golangci-lint 0 issues, go vet clean, race tests green, gofumpt no diffs.
- Census re-derivation: SET diff vs toolchain = ∅.
- Absence-assertion audit: CLEAN — `TestScanForLine_DetectsEFWD001ProductionEmission` (P11 fix) still passes.
- Symbol-resolution bar: all cited symbols verified.
- Double-liveness bar: new bar codified — for every test double wired, verify an assertion consumes it before teardown/restart, and prove liveness by mutation (flip the double's value; the test must fail).
- POL-002 sync: PASS — story v1.15 registered in STORY-INDEX v4.21→v4.22.
- P3/P4 fixes mutation-probed: both hold.
- 7 VP e2e perimeter files confirmed purely mechanical (ctx-first call-site updates only; no behavioral assertions added).
- Core production code confirmed clean under fresh eyes and mutation probing.
- All P1-P14 fixes verified holding.
- Streak 0/3 (HAS_FINDINGS resets).

**Trajectory shorthand (P1–P15):** 7/3/3/1/1/2/2/1/1/1/1/1/1/1/1

**Cycle ledger:** 15 passes, 27 findings (7/3/3/1/1/2/2/1/1/1/1/1/1/1/1), all fixed/adjudicated, zero open. Streak 0/3. Awaiting: adversary pass 16 @ 79c1284.

---

## S-7.04-FU-PE-CONNECTOR — Adversarial Pass 16 (2026-07-08)

**Verdict:** HAS_FINDINGS — 1 LOW [doc-drift]

**Code HEAD:** 7daed41 (advanced from 79c1284 — doc-only remediation commit)

### Finding F-P16-001 LOW [doc-drift]

**What:** `testenv.go` `Restart` method's doc comment described a never-taken control path. The comment claimed "already wired → ReloadAddrs reuse" and stated "in both cases polls" — but the body unconditionally tears down and recreates the connector, with an empty-upstreams early-return that exits before any reuse path could run. The symbol cited (`ReloadAddrs`) is REAL and passes symbol-resolution, but it is never invoked on the path the comment describes.

**Novel failure axis — real-symbol-never-invoked defeats the symbol-resolution bar:** This is the fourth shape in the comment-vs-code-path family across this cycle:
- P11: vacuous absence key (string typo — key never matched production emission)
- P13: phantom symbol (cited function does not exist anywhere in the repo)
- P15: orphaned fake (double wired but discarded before observation; assertion satisfied by independent mechanism)
- P16: real-symbol-never-invoked (cited symbol exists and resolves grep, but the described control path — "already wired → ReloadAddrs reuse" — is never taken; the body unconditionally tears down regardless)

The symbol-resolution bar passes (grep finds `ReloadAddrs`). Only the claim→code mapping catches it: tracing every behavioral sentence in the doc comment to the specific code lines that implement it, verified at authoring time.

**Remediation at commit 7daed41 (doc-only):**
- Doc comment rewritten to accurately describe the unconditional teardown-and-recreate path.
- 7-row claim→code mapping in remediation report (each sentence in the original comment traced to the actual code line, confirming which claims were false).
- No production code changed. No test changes.

**Story sync → v1.16:** Changelog row added. Co-reference sweep adjudicated ~17 hits — all accurate/preserve (the false claim never propagated into the story prose — NO-EDIT on prose sections). Full CI gate green.

**P16 verification results:**
- Full CI gate: golangci-lint 0 issues, go vet clean, race tests green, gofumpt no diffs.
- P15 fix mutation-proven holding: mutation of `fakeConnE` return value still fails the new inverse-delegation assertion (double-liveness bar holds).
- All seven standing bars PASS.
- POL-002 sync: PASS — story v1.16 registered in STORY-INDEX v4.22→v4.23.
- Streak 0/3 (HAS_FINDINGS resets).

**Trajectory shorthand (P1–P16):** 7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1

**Cycle ledger:** 16 passes, 28 findings (7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1), all fixed/adjudicated, zero open. Streak 0/3. Awaiting: adversary pass 17 @ 7daed41.

---

## S-7.04-FU-PE-CONNECTOR — Adversarial Pass 18 (2026-07-08)

**Verdict:** NO_FINDINGS — streak 0/3 → 1/3

**Code HEAD:** 7c6d841 (unchanged — zero code changes this pass)

### Summary

All seven standing bars green from fresh context:
1. **Full CI gate** — golangci-lint 0 issues, go vet clean, race tests green, gofumpt no diffs, full-repo test suite pass.
2. **Census re-derivation** — 24/24 import sets exact-match toolchain; SET diff vs toolchain = ∅.
3. **Absence-assertion audit** — `TestScanForLine_DetectsEFWD001ProductionEmission` (P11 fix) still passes; key `"E-FWD-001"` confirmed accurate.
4. **Symbol resolution** — all symbols cited in authored comments and anchors grep-resolved.
5. **Claim→code clean in blast radius** — sweep of all 14 perimeter files confirmed; no false claim/code gaps introduced since P17.
6. **Double-liveness bar** — `TestRunRouter_PE_RouterHandleModeReflectsLiveState` mutation-pinned inverse-delegation assertion holds (P15 fix).
7. **POL-002 sync** — story v1.17 registered in STORY-INDEX v4.24 (no version change this pass; alignment confirmed).

**P17 fix verified holding:** zero `// Stub:` and `// After AC-` hits in all 7 core perimeter files — red-gate-provenance class confined and retired as recorded.

### Notable Adjudicated Anti-Findings

| Finding | Adjudication |
|---------|-------------|
| runRouter numbered-list step order in doc comment | Pre-existing develop drift; outside this story's diff; not a defect introduced by this story |
| FCL historical line citations | History rows accurately recording past state; not live claims about current code |
| dialLoop no-dial-timeout | Legitimate design within BC-2.09.001 EC-001 spec — ctx-cancellation bounds the loop duration; no missing timeout |
| upstreamdial pkg-doc forbidden-range "20–23" predates bench@24 | Harmless; internal/bench is a no-edge test leaf that never imports upstreamdial; DAG position 24 does not create a cycle |

### Outcome

- **No code changes** required.
- **No story changes** required.
- Code HEAD unchanged at 7c6d841. Story unchanged at v1.17.
- Cycle ledger: 18 passes, 29 findings (7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1/0), all fixed/adjudicated, zero open.
- **Streak: 1/3.**

**Awaiting:** adversary pass 19 @ 7c6d841 (streak 1/3)

---

## S-7.04-FU-PE-CONNECTOR — Adversarial Pass 17 (2026-07-08)

**Verdict:** HAS_FINDINGS — 1 LOW [semantic-accuracy]

**Code HEAD:** 7c6d841 (advanced from 7daed41 — doc-only remediation commit)

### Finding F-P17-001 LOW [semantic-accuracy]

**What:** A stale red-gate-era comment above `eRouter.Restart` in `TestE2E_EtoPEGraduationByConfigChange` contained two false claims: (1) "After AC-006: calls `connector.ReloadAddrs()`" — Restart unconditionally tears down any existing connector (Stop) and recreates via `upstreamdial.New`; it never calls `ReloadAddrs`. The Restart-teardown vs SIGHUP-seam-ReloadAddrs division was ratified in P16's claim→code mapping for `testenv.go Restart`. (2) "Stub: sets `r.mode=ModePE` unconditionally, dials nothing." — the stub was retired in this story's TDD implementation commits. The comment was authored at the red-suite commit d3bac4c (when the stub was live and the dial loop not yet wired) and was never reconciled after the green-gate implementation pass. Fifth instance of the comment-vs-code-path family; first instance in a test-CALLER location (all prior instances were in the method's OWN doc or in seam-wiring comments). Sub-shape: red-gate provenance (the "Stub:" prefix pattern).

**Novel failure axis — red-gate test suites annotate expected post-implementation behavior; green-gate reconciles assertions but not caller-side comments.** When the implementation commits replace stubs and wire live logic, per-method doc comments and test body comments directly on the method under test are swept — but the caller's side of the call site carries its own "after this story lands" annotation that can survive unremediated. The `TestE2E_EtoPEGraduationByConfigChange` comment was a forward-looking "after AC-006" annotation that documented what the test would prove; it was accurate at red-gate and became false at green-gate without producing any test failure.

**Remediation at commit 7c6d841 (doc-only):**
- Comment rewritten: "Tears down any existing connector and builds a fresh live-dialing `upstreamdial.Connector` against `peAddr`, polling up to 3s for ModePE (AC-006; teardown-recreate per the testenv Restart contract — ReloadAddrs reuse belongs to the production SIGHUP seam)."
- 4-row claim→code mapping verified against `testenv.go` `Restart` body.
- Full CI gate cleared.

**CLASS CLOSED — red-gate-provenance sub-shape confined and retired:** Orchestrator swept `// Stub:` and `// After AC-` across all 7 core perimeter files (`connector.go`, `connector_test.go`, `mgmt_wire.go`, `router_config.go`, `testenv.go`, `router_sighup_test.go`, `router_pe_connector_test.go`). These two lines in `TestE2E_EtoPEGraduationByConfigChange` were the only hits. The red-gate-provenance comment class is confined to a single test and is now retired.

**P17 verification results:**
- P16 fix holds: `Restart` doc comment accurately describes teardown-recreate (claim→code mapping spot-checked at 7c6d841; `ReloadAddrs` call absent from `Restart` body, confirmed).
- All seven standing bars PASS.
- Full CI gate green (golangci-lint 0 issues, go vet clean, race tests green, gofumpt no diffs).
- Story synced → v1.17: changelog row only; co-reference sweep confirmed story prose already clean (no "Stub:" or "After AC-006" residuals in story body).
- POL-002 sync: PASS — story v1.17 registered in STORY-INDEX v4.23→v4.24.
- Streak 0/3 (HAS_FINDINGS resets).

**Trajectory shorthand (P1–P17):** 7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1

**Cycle ledger:** 17 passes, 29 findings (7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1), all fixed/adjudicated, zero open. Streak 0/3. Awaiting: adversary pass 18 @ 7c6d841.

---

## S-7.04-FU-PE-CONNECTOR — Adversarial Pass 20 (2026-07-08)

**Verdict:** HAS_FINDINGS — 1 LOW [process-gap]

**Code HEAD:** 7c6d841 (unchanged — story-only fix)

### Finding F-P20-001 LOW [process-gap]

**What:** Story v1.18's changelog row self-classified F-P19-001 as "MED [doc-drift]" — disagreeing with 15 sibling statements that record "LOW [process-gap]" on both severity and class. This is the seventh shape of the record-consistency family: a remediation artifact misclassifying the finding it remediates. Fifteen sources establish the adjudicated classification: the pass-19 ledger entry, STATE.md frontmatter, STATE.md current-phase-steps row, sprint-state.yaml p19_remediation stanza, STORY-INDEX.md backlog row, and all 9 prior F-PNN-001 rows in the story's own changelog.

**Root cause — remediation dispatches must pin the adjudicated classification:** The P19 remediation dispatch said "close the bare-form citation gap" but did not quote the verbatim classification from the ledger. The author re-derived a plausible classification ("MED [doc-drift]" — citations drifting), but the actual adjudication at P19 was "LOW [process-gap]" (an orthography blind-spot in a class-closure claim). Without the pin, self-assessment diverged from the authoritative record.

**Remediation:** Story v1.19 (story-only fix, code HEAD unchanged 7c6d841):
- One-token correction in P19 changelog row: "MED [doc-drift]" → "LOW [process-gap]".
- FULL consistency sweep of all 9 F-PNN-001 classification strings in the story changelog — all 9 now match the trajectory ledger exactly.
- ORCHESTRATOR ADJUDICATION: F-P16-001 story label "LOW [doc-drift/semantic-accuracy]" vs ledger header "LOW [doc-drift]" — KEEP as-is. Severity matches (both LOW); the `/semantic-accuracy` qualifier is an elaboration the ledger body itself uses, not a contradiction. Future passes must not re-raise this as a discrepancy.

**P20 verification results:**
- All eight standing bars GREEN (full CI gate, census re-derivation, absence-assertion keys, symbol resolution, claim→code in blast radius, double-liveness, citation orthography, POL-002 sync).
- P19's fix verified holding: both orthography classes closed; zero live line citations outside SHA-pinned/historical rows.
- Code surface clean: all prior code-lane fixes holding.
- Code HEAD unchanged at 7c6d841. Story HEAD now v1.19.

**Cycle ledger:** 20 passes, 31 findings (7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1/0/1/1), all fixed/adjudicated, zero open. Streak 0/3.

**Awaiting:** adversary pass 21 @ 7c6d841 (streak 0/3)

---

## S-7.04-FU-PE-CONNECTOR — Adversarial Pass 21 (2026-07-08)

**Verdict:** NO_FINDINGS — streak 0/3 → 1/3

**Code HEAD:** 7c6d841 (unchanged — zero code or story changes this pass)

### Summary

All nine standing bars green from fresh context:
1. **Full CI gate** — golangci-lint 0 issues, go vet clean, race tests green, gofumpt no diffs, full-repo test suite pass.
2. **Census re-derivation** — 24/24 import sets identical to toolchain; SET diff vs toolchain = ∅.
3. **Absence-assertion keys verbatim** — `TestScanForLine_DetectsEFWD001ProductionEmission` key `"E-FWD-001"` confirmed accurate; vacuous-absence class remains closed.
4. **Symbol resolution** — all symbols cited in authored comments and anchors grep-resolved.
5. **Claim→code clean in blast radius** — sweep of all 14 perimeter files confirmed; no false claim/code gaps introduced since P20.
6. **Double-liveness bar** — `TestRunRouter_PE_RouterHandleModeReflectsLiveState` mutation-pinned inverse-delegation assertion holds (P15 fix).
7. **Citation-orthography both forms** — BOTH-orthography closure (prefixed `file.go:NNN` and bare `:NNN`) holds; only adjudicated SHA-pinned/historical keeps remain live.
8. **Classification-consistency** — all 9 F-PNN-001 classification strings in story changelog match the trajectory ledger exactly; P20 fix holds (v1.18 row reads LOW [process-gap]; residual MED string is inside the v1.19 remediation record — correct as history, not an error).
9. **POL-002 sync** — story v1.19 registered in STORY-INDEX v4.27 (no version change this pass; alignment confirmed).

### Notable Anti-Finding Adjudications (record, not defects)

| Finding | Adjudication |
|---------|-------------|
| testenv `Mode()` lock-release before external call pattern | Deliberate and correct — lock released before calling `upstreamdial.New` to avoid lock inversion; consistent with `Restart` teardown contract |
| `NewTicker` zero-interval unreachability | Re-confirmed dead code per prior sweep; not a defect (production callers always pass non-zero intervals) |
| Retired seam methods (`SetConnector`, `SetSighupCh`) have no dangling refs | Confirmed clean across all 14 perimeter files; retirement was complete |
| ctx-first parameter swap consistent across 30+ call sites | P14 go.md rule-7 sweep held; no reversion detected |
| `peRouterAddr` dynamic listener correct | Dynamic binding verifies against live accept loop; no spec contradiction |
| §6.6.2 forbidden-edges consistent | ARCH-08 §6.6.2 permitted-importers for upstreamdial and testenv both accurate |

### Outcome

- **No code changes** required.
- **No story changes** required.
- Code HEAD unchanged at 7c6d841. Story unchanged at v1.19.
- Cycle ledger: 21 passes, 31 findings (7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1/0/1/1/0), zero open.
- **Streak: 1/3.**

**Awaiting:** adversary pass 22 @ 7c6d841 (streak 1/3)

---

## S-7.04-FU-PE-CONNECTOR — Adversarial Pass 9 (2026-07-07)

**Verdict:** HAS_FINDINGS — streak RESET (P8 class-closing claim falsified)

**Code HEAD:** 49c9370 (unchanged — zero code changes this pass; five consecutive passes with zero code-correctness defects)

### Finding F-P9-001 LOW [process-gap]

**What:** ARCH-08 §6.5 authoritative census omitted `internal/bench` (PR #109 cd67394, present at anchor 62e38d3). P8 v2.9 stated "full-artifact arithmetic sweep verified no further discrepancies" — this claim was falsified by a one-liner toolchain re-derivation: `go list ./internal/... @ 62e38d3` returned 23 packages, not 22.

**Novel failure axis — set-membership vs arithmetic/per-row-content:** All eight prior passes (P1–P8) verified the census by examining rows already present in the table: checking arithmetic totals, confirming per-row content accuracy, and verifying cross-references. P8 applied a full-artifact sweep that confirmed all of this. But none of the nine passes ever re-ran the generating command to verify that no registered package was absent from the table. The set-membership axis is orthogonal to arithmetic and content correctness — a table can be internally consistent and arithmetically correct while still missing an entry.

**Remediation:** Option A — `internal/bench` appended at position 24, no renumber of existing rows, no code changes. ARCH-08 → v2.10 (on disk, verified). Architect ruling: position 24, no renumber.

**Toolchain re-derived census:** `go list ./internal/... @ 62e38d3` → 23 packages. Verified no other unregistered packages remain.

**Streak reset rationale:** P8 issued an explicit class-closing claim ("sweep verified no further discrepancies"). That claim was falsified by F-P9-001. A class-closing claim that is later falsified requires a streak reset regardless of finding severity; the streak cannot advance on a pass whose closure assertion did not hold.

**Cycle ledger:** 9 passes, 21 findings (7/3/3/1/1/2/2/1/1), all fixed/adjudicated, zero open. Code lane unchanged at 49c9370 (five consecutive passes with zero code-correctness defects).

**Awaiting:** adversary pass 10 @ 49c9370 (streak 0/3)

---

## S-7.04-FU-PE-CONNECTOR — Adversarial Pass 13 (2026-07-08)

**Verdict:** HAS_FINDINGS — 1 LOW [test-fidelity]

**Code HEAD:** 0a350d6 (advanced from 14ae327 — one remediation commit required)

### Finding F-P13-001 LOW [test-fidelity]

**What:** The P12 citation-stabilization commit 14ae327 introduced a PHANTOM symbol: two comments in `cmd/switchboard/router_sighup_test.go` cited `"runRouter/buildAndWireConnector"`. The function `buildAndWireConnector` does not exist anywhere in the repository — connector construction logic is inline in `runRouter` at `mgmt_wire.go:408`. This is the first finding introduced BY a remediation commit's anchor-stabilization pass rather than pre-existing in the production code.

**Novel failure axis — anchor-stabilization creates phantom references:** All twelve prior passes reviewed code and comments for correctness against the codebase as it stood. P12's remediation task was specifically to replace stale line-number citations with stable anchors — but that stabilization itself was not re-verified for symbol resolution. The replacement text cited a non-existent function by name, producing a phantom reference that is strictly worse than the stale line-number it replaced (a stale line number goes to the wrong line; a phantom symbol points to nothing). Standing bar codified: every symbol cited in an authored comment or anchor must be grep-resolved against the codebase before the commit is declared remediated.

**Remediation:** Code commit 0a350d6 on `story/s-7.04-fu-pe-connector`: both comments corrected to read `"both inline in runRouter in mgmt_wire.go"`. Symbol-resolution table in remediation report: `upstreamRoutersFor` → `router_config.go:77`, `keepaliveIntervalFor` → `router_config.go:57`, `runRouter` → `mgmt_wire.go:408`. `buildAndWireConnector` → 0 hits (confirms phantom). Full CI gate re-cleared (golangci-lint 0 issues, vet clean, race tests green, gofumpt no diffs).

**Story sync → v1.13:** Symbol-resolution verification bar added to changelog; `buildAndWireConnector` reference confirmed absent from story prose (appears only in v1.13 changelog row as historical record).

**P13 verification results:**
- Full CI gate: golangci-lint 0 issues, go vet clean, race tests green, gofumpt no diffs.
- Census re-derivation: SET diff vs toolchain = ∅ (no new unregistered packages).
- Absence-assertion audit: CLEAN — `TestScanForLine_DetectsEFWD001ProductionEmission` (P11 fix) still passes; pin-test key `"E-FWD-001"` confirmed accurate.
- POL-002 sync: PASS — story v1.13 registered in STORY-INDEX v4.19→v4.20.
- All P1-P12 code-lane fixes verified holding.
- Streak 0/3 (HAS_FINDINGS resets).

**Trajectory shorthand (P1–P13):** 7/3/3/1/1/2/2/1/1/1/1/1/1

**Cycle ledger:** 13 passes, 25 findings (7/3/3/1/1/2/2/1/1/1/1/1/1), all fixed/adjudicated, zero open. Streak 0/3. Awaiting: adversary pass 14 @ 0a350d6.

---

## S-7.04-FU-PE-CONNECTOR — Adversarial Pass 12 (2026-07-08)

**Verdict:** HAS_FINDINGS — 1 MED [test-hygiene/CI-gate]

**Code HEAD:** 14ae327 (advanced from 6e00332 — two remediation commits required)

### Finding F-P12-001 MED [test-hygiene/CI-gate]

**What:** The P11 pin test (`TestScanForLine_DetectsEFWD001ProductionEmission`) used three unchecked `buf.Write` calls. The golangci-lint errcheck linter (required CI step) flagged all three, making the branch unmergeable.

**Novel failure axis — remediation commit CI-gate regression:** Eleven prior passes had run the full local CI gate (golangci-lint, go vet, race tests, gofumpt) against feature/fix commits, but never against remediation commits specifically. The P11 fix was validated for test correctness and race cleanliness, but errcheck was not run against the new pin test before the state-manager burst closed P11. This is the first instance of a remediation commit itself introducing a CI gate violation in this cycle.

**Remediation:**
- d882686: errcheck discard (`_` assignment) added to the 3 `buf.Write` calls in `TestScanForLine_DetectsEFWD001ProductionEmission`; 2 stale `connector.go` line-number citations in test comments retired with stable mechanism-description anchors.
- 14ae327: 5 stale `mgmt_wire.go` line-number citations in `router_sighup_test.go` → stable function/guard anchors.
- golangci-lint 0 issues, go vet clean, race tests green post both commits.

**Line-number-citation class structurally closed:** Residual = 3× `on_frame_arrival.go:252` pin-test anchors — verified accurate at HEAD 14ae327 (KEEP). No further stale line-number citations remain across the codebase.

**Story sync → v1.12:** story-writer's sweep fixed 2 stale live-prose `mgmt_wire.go` citations in the story body (Concurrency Contract section + AC-003 precondition → stable anchors referencing the `addrsCh` guard and `SetSighupCh` seam).

**P12 verification results:**
- P11 fix (absence-assertion key `"E-FWD-001"`) holds at root — `TestScanForLine_DetectsEFWD001ProductionEmission` passes.
- Cycle-wide absence-assertion audit: CLEAN — no second instance of a vacuous absence assertion found.
- Census re-derivation: SET diff vs toolchain = ∅ (no new unregistered packages).
- POL-002 sync: PASS — story v1.12 registered in STORY-INDEX v4.18→v4.19.
- All P1-P11 code-lane fixes verified holding.
- Streak 0/3 (HAS_FINDINGS resets).

**Codified lesson:** Remediation commits must re-clear the FULL local CI gate (golangci-lint run ./..., go vet, race tests, gofumpt) before a pass is declared remediated — not just the test that motivated the fix. Stable mechanism anchors preferred over line-number citations in test comments.

**Cycle ledger:** 12 passes, 24 findings (7/3/3/1/1/2/2/1/1/1/1/1), all fixed/adjudicated, zero open. Streak 0/3. Awaiting: adversary pass 13 @ 14ae327.

---

## S-7.04-FU-PE-CONNECTOR — Adversarial Pass 11 (2026-07-07)

**Verdict:** HAS_FINDINGS — 1 LOW [test-fidelity]

**Code HEAD:** 6e00332 (advanced from 49c9370 — code fix required for test-fidelity defect)

### Finding F-P11-001 LOW [test-fidelity]

**Novel axis — negative-assertion string fidelity (absence assertions):** AC-004's negative assertion in `router_pe_connector_test.go` searched for `"split-horizon blocked"` (space form) to confirm E-FWD-001 does NOT fire. Production emission at `internal/routing/on_frame_arrival.go:252` uses hyphenated `"split-horizon-blocked: ... (BC-2.02.008 E-FWD-001)"`. The space form can never match the hyphenated production string — the assertion was vacuously true for the entire cycle. Ten prior passes (P1–P10) checked positive-emission polarity exclusively; none examined whether the key in an absence assertion could actually match the production string.

**How defect proven:** `strings.Contains(productionEmission, "split-horizon blocked")` == false; `strings.Contains(productionEmission, "split-horizon-blocked")` == true. The assertion passed unconditionally regardless of runtime behavior.

**Remediation (code commit 6e00332 on story/s-7.04-fu-pe-connector):**
- Search key corrected from `"split-horizon blocked"` to spec-anchored `"E-FWD-001"` (stable across prose rewording; appears in the production emission alongside the hyphenated form).
- Prose comment aligned to match the corrected key.
- Pin test `TestScanForLine_DetectsEFWD001ProductionEmission` added: embeds verbatim production emission from `on_frame_arrival.go:252`; proves (a) fixed key `"E-FWD-001"` detects the real emission (non-vacuousness proof), and (b) space form `"split-horizon blocked"` does NOT match (pins the defect shape).
- All suites green with `-race`.

**P11 verification results:**
- P10 fix (upstreamRoutersAsSet co-references → story v1.10) verified holding.
- Census re-derivation: SET diff vs toolchain = ∅ (no new unregistered packages).
- POL-002 sync: PASS — story v1.11 registered in STORY-INDEX v4.17→v4.18.
- All P1-P9 code-lane fixes verified holding.
- Streak 0/3 (HAS_FINDINGS resets).

**Story sync:** story-writer synced story → v1.11: pin-test `TestScanForLine_DetectsEFWD001ProductionEmission` registered in test-surface table, changelog row citing 6e00332 added, four-pattern co-reference sweep (`"split-horizon blocked"` / `"split-horizon-blocked"` / `"E-FWD-001"` / `scanForLine`) clean.

**Cycle ledger:** 11 passes, 23 findings (7/3/3/1/1/2/2/1/1/1/1), all fixed/adjudicated, zero open.

**Awaiting:** adversary pass 12 @ 6e00332 (streak 0/3)

---

## S-7.04-FU-PE-CONNECTOR — Adversarial Pass 10 (2026-07-07)

**Verdict:** HAS_FINDINGS — 1 LOW [process-gap]

**Code HEAD:** 49c9370 (unchanged — zero code changes this pass; six consecutive passes with zero code-correctness defects)

### Finding F-P10-001 LOW [process-gap]

**What:** Three live story references to the deleted helper `upstreamRoutersAsSet` (originally deleted by F-P1-007). All three cited the function in present tense as if it still existed: (1) AC-001 postcondition 5 cited it as the normative mechanism for `upstreamRoutersFor(cfg)` result; (2) the test-mapping row named it as the unit-under-test for the keepalive isolation test; (3) FO-1 resolution column cited it in present tense. This is the fourth straggler from the single F-P1-007 deletion — the same co-reference-staleness class as P7/P8/P9, now in the story artifact rather than the architecture doc.

**Remediation:** Story-writer fixed all three locations → story v1.10. Full-file co-reference sweep performed on the entire story file covering `upstreamRoutersAsSet`, `router_config`, `peConnectorHook`, and `SetSighupCh`. All remaining hits adjudicated: correct historical records (changelog, erratum notes) or correct live prose. No further stale references remain.

**Machine-verification of v2.10 census:** All 24 import sets in ARCH-08 §6.5 exact-match toolchain output; SET diff vs toolchain = ∅ unregistered packages; position sequence clean; bench↔testenv independence confirmed (Option A holds).

**Codified lesson:** Remediation dispatches for symbol deletion/rename must include a mandatory same-artifact co-reference grep with per-hit adjudication. The dispatch wording for F-P1-007 swept only the primary deletion site; this produced four straggler findings across passes 7-10 as the adversary independently discovered each co-reference surface. The sweep converts O(passes) straggler discovery into O(1). See lessons.md lesson #8 [codified].

**Cycle ledger:** 10 passes, 22 findings (7/3/3/1/1/2/2/1/1/1), all fixed/adjudicated, zero open. Code lane unchanged at 49c9370 (six consecutive passes with zero code-correctness defects).

**Awaiting:** adversary pass 11 @ 49c9370 (streak 0/3)

---

## S-7.04-FU-PE-CONNECTOR — Adversarial Pass 19 (2026-07-08)

**Verdict:** HAS_FINDINGS — 1 LOW [process-gap]

**Code HEAD:** 7c6d841 (unchanged — story-only fix)

### Finding F-P19-001 LOW [process-gap]

**What:** Four stale BARE-FORM line citations in story FCL row 1 — `:269`, `:337`, `:346-350`, `:284-287` — all shifted by the P14 ctx-first refactor (670c64b moved call sites in `router_config.go`). These survived passes 12 through 18 because P12's "line-citation class closed" sweep keyed on the prefixed orthography (`file.go:NNN`) only; the bare form (`:NNN`, filename implied by table context) was invisible to that sweep.

**Root cause — orthography gap:** P12 issued a class-closing claim ("line-citation class structurally closed") based on sweeping the `file.go:NNN` spelling. The bare `:NNN` form is a second spelling of the same class. A class-closure claim must enumerate BOTH spellings and sweep each. This is the sixth shape in the doc-vs-code defect family; the first found in a spec-artifact FCL row (as opposed to code comments or story prose).

**Remediation:** Story v1.18 (story-only fix, code HEAD unchanged at 7c6d841):
- Four bare-form citations in FCL row 1 converted to stable mechanism anchors (no fresh line numbers).
- BOTH-orthography sweep across the entire story: 6 additional live citations converted to symbol/mechanism anchors (`testenv.go:302`/`:326` retired-seam refs ×2 locations, `router_config.go:81`/`:76`).
- 2 legitimately KEPT: `testenv.go:956` SHA-pinned to 950285c (P12-adjudicated historical pin); `on_frame_arrival.go:252` P12-adjudicated pin anchor.
- Changelog rows preserved (historical state records).

**Closure verification:** Both orthography classes now closed. Residual verified: zero live line citations outside SHA-pinned/historical rows across the full story.

**P19 verification results:**
- All seven standing bars green (full CI gate, census re-derivation, absence-assertion keys, symbol resolution, claim→code in blast radius, double-liveness, POL-002 sync).
- Code surface clean: P19 adversary verified all code-lane fixes holding.
- Code HEAD unchanged at 7c6d841. Story HEAD now v1.18.

**Cycle ledger:** 19 passes, 30 findings (7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1/0/1), all fixed/adjudicated, zero open. Streak 1/3 → 0/3 (reset).

**Awaiting:** adversary pass 20 @ 7c6d841 (streak 0/3)
