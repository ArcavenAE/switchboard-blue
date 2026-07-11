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

**Awaiting:** adversary pass 23 @ 4f2807c (streak 0/3)

---

## S-7.04-FU-PE-CONNECTOR — Adversarial Pass 22 (2026-07-08)

**Verdict:** HAS_FINDINGS — 1 LOW [doc-drift]

**Code HEAD:** 4f2807c (advanced from 7c6d841 — one remediation commit required)

### Finding F-P22-001 LOW [doc-drift]

**What:** An inline poll-tail comment in `RouterHandle.Restart` (`internal/testenv/testenv.go`) claimed "stub/unreachable addrs will exhaust the timeout and Mode() will fall back to r.mode" — doubly wrong: (1) the `r.mode` fallback is UNREACHABLE while a connector is wired (Restart sets `r.connector=conn` before the poll loop and never nils it on timeout — the poll exits with the connector still live, so Mode() delegates to the connector, never reaching the `r.mode` branch); (2) if the unreachable branch were taken, it would return `ModePE` (r.mode was set to ModePE above for non-empty upstreams), the inverse of the correct `ModeE`-via-connector-delegation result. Third defect inside Restart across the cycle (P16 header / P17 caller comment / P22 inline tail); eighth shape of the comment-vs-code-path family.

**Pattern — partial reconciliation relocates drift:** P16 fixed the Restart doc header; P17 fixed the caller-side red-gate annotation. Each fix reconciled only its flagged fragment, leaving the inline poll-tail comment unexamined. The full function-granularity sweep that should have closed the class was never applied until P22 forced it.

**Novel finding axis — unreachable-AND-inverted double error:** Prior shapes in the family were singly wrong (phantom symbol, orphaned fake, never-taken path). This shape is doubly wrong: the path is both unreachable (structural) and outcome-inverted (semantic) — two independent defects in one comment.

**Remediation (commit 4f2807c):**
- Comment rewritten with 6-row claim→code mapping.
- FULL-FUNCTION comment sweep of Restart + Mode(): every comment in both functions adjudicated. All others accurate — class is now closed for the whole function. Only remaining "fall back" occurrence in the file is the new accurate negation.
- Full CI gate cleared (golangci-lint 0 issues, go vet clean, race tests green, gofumpt no diffs).

**Story sync → v1.20:** Co-reference sweep adjudicated all hits — "fall back" 0 hits in story (false claim never propagated); 7 "r.mode" hits all accurate behavioral prose or historical records, preserved.

**P22 verification results:**
- All nine standing bars green: full CI gate, census re-derivation 24/24, absence-assertion keys verbatim, symbol resolution, claim→code blast-radius, double-liveness, citation-orthography both forms, classification-consistency 9/9, POL-002 sync.
- P19/P20 fixes verified holding.
- Code HEAD 4f2807c. Story HEAD v1.20.

**Trajectory shorthand (P1–P22):** 7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1/0/1/1/0/1

**Cycle ledger:** 22 passes, 32 findings (7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1/0/1/1/0/1), zero open. Streak 1/3 → 0/3. Awaiting: adversary pass 23 @ 4f2807c.

---

## S-7.04-FU-PE-CONNECTOR — Adversarial Pass 24 (2026-07-08)

**Verdict:** HAS_FINDINGS — 1 LOW [doc-drift]

**Code HEAD:** f2f0ba6 (advanced from c676134 — one remediation commit required)

### Finding F-P24-001 LOW [doc-drift]

**What:** A production comment in `connector.go` EC-004 guard used a stale relative numeric line-locator: "60 lines above" when the actual distance was 65 lines. The relative form is structurally unstable — it is never accurate because any intervening edit shifts the distance without updating the comment. This is the first relative-form locator found anywhere in the core files across 24 passes, and the first finding on a production-file comment surface previously unswept (all prior comment sweeps covered testenv, connector_test, mgmt_wire header, and citations surfaces, but connector.go itself had not been swept at file granularity until P24).

**Novel finding axis — relative numeric line-locator:** All prior doc-drift findings identified prose claims whose described code path was wrong (non-existent symbols, inverted semantics, unreachable paths, stale mechanism descriptions). This is the first instance of a comment whose semantic intent is correct but whose positional claim is structurally unstable by form — "60 lines above" could only be accurate at exactly one moment in time.

**Remediation (commit f2f0ba6):**
- Relative locator replaced with a prose anchor naming the cited construct directly (stable across line-number shifts).
- FULL-FILE comment sweep of `connector.go`: 18 comment scopes adjudicated — 17 accurate, 1 fixed. File now swept at file granularity.
- Story synced → v1.22 (on disk, verified): FCL row already used prose-anchor form per P19; story sweep clean — no propagation of the relative-locator form into story prose.
- Full CI gate cleared. Code behavior unchanged since P17.

**PERIMETER-COMPLETION milestone:** With P24's full-file sweep of `connector.go`, all comment surfaces in the story perimeter have now been swept at file granularity:

| File | Sweep pass | Scope | Findings |
|------|-----------|-------|---------|
| `internal/testenv/testenv.go` | P16 (Restart method) + P22 (Restart + Mode() full-function) | Production tree — method doc + inline | P16: 1 fixed (ReloadAddrs never-invoked); P22: 1 fixed (poll-tail inverted), all others accurate |
| `internal/upstreamdial/connector_test.go` | P23 | Test file — full file | P23: 1 fixed (reserved-port mechanism), 15 accurate |
| Citations (`router_sighup_test.go`, `mgmt_wire.go`) | P12 (line-number citation sweep) + P17 (red-gate provenance) + P19 (bare-form orthography) | Test + production header | Multiple passes: line-citation class closed, red-gate class retired, both orthography forms closed |
| `cmd/switchboard/mgmt_wire.go` (runRouter header) | P14 | Production header | P14: 1 fixed (deferred → shipped), all others accurate |
| `internal/upstreamdial/connector.go` | P24 | Production file — full file | P24: 1 fixed (relative locator → prose anchor), 17 accurate |

Every file in the core perimeter has been swept at file granularity. The story perimeter comment surface is COMPLETE.

**P24 verification results:**
- All nine standing bars green: full CI gate, census re-derivation 24/24, absence-assertion keys verbatim, symbol resolution, claim→code blast-radius, double-liveness, citation-orthography both forms, classification-consistency, POL-002 sync.
- P23 fix verified holding (spot-checked: storm test address-setup comment accurate, ephemeral pattern correctly described).
- Code behavior unchanged since P17. Code HEAD f2f0ba6. Story HEAD v1.22.

**Trajectory shorthand (P1–P24):** 7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1/0/1/1/0/1/1/1

**Cycle ledger:** 24 passes, 34 findings (7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1/0/1/1/0/1/1/1), zero open. Streak 0/3. Awaiting: adversary pass 25 @ f2f0ba6.

---

## S-7.04-FU-PE-CONNECTOR — Adversarial Pass 23 (2026-07-08)

**Verdict:** HAS_FINDINGS — 1 LOW [doc-drift]

**Code HEAD:** c676134 (advanced from 4f2807c — one remediation commit required)

### Finding F-P23-001 LOW [doc-drift]

**What:** A storm test in `connector_test.go` carried a hardcoded "127.0.0.1:1 reserved port" mechanism claim in an address-setup comment while the code uses the F-P1-005 probe-and-close ephemeral pattern. No `:1` literal exists anywhere in the repository; an adjacent comment in the same test was already accurate. This is the ninth family shape — the first address/port-mechanism claim in the cycle, and the first finding in a previously-unswept file (`connector_test.go`). A draft-stage artifact: the "reserved port" comment was written at the red-gate commit before the ephemeral port mechanism was finalized, and was never reconciled after the green-gate implementation pass.

**Sweep-granularity trajectory — line → function → file:** Prior sweeps progressed from line-level (P6, P12) through function-level (P16, P22) to file-level (P23). `connector_test.go` is the first file swept at full-file granularity in this cycle. 16 address/port-mechanism hits adjudicated: 15 accurate, 1 fixed.

**P22 fix verified holding:** Poll-tail comment in `RouterHandle.Restart` accurately reflects unconditional teardown-and-recreate (spot-checked via claim→code mapping); full-function sweep result holds. All nine bars otherwise green: full CI gate, census re-derivation 24/24, absence-assertion keys verbatim, symbol resolution, claim→code clean in blast radius, double-liveness, citation-orthography both forms, classification-consistency 9/9, POL-002 sync.

**Code behavior unchanged since P17.** Remediation is comment-only.

**Remediation (commit c676134):**
- Address-setup comment rewritten citing F-P1-005 explicitly with 5-row claim→code mapping (no `:1` pattern; `net.Listen("tcp", "127.0.0.1:0")` + `ln.Addr().String()` + `ln.Close()` + caller receives string).
- FULL-FILE comment sweep of `connector_test.go`: 16 address/port-mechanism hits adjudicated — 15 accurate (ephemeral-port pattern correctly described or historical-record rows), 1 fixed. File now swept at file granularity.
- Story synced → v1.21 (on disk, verified): "reserved port" 0 hits (never propagated to story); all "127.0.0.1" and "StormNoDeadlock" hits accurate/historical.

**Trajectory shorthand (P1–P23):** 7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1/0/1/1/0/1/1

**Cycle ledger:** 23 passes, 33 findings (7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1/0/1/1/0/1/1), zero open. Streak 0/3. Awaiting: adversary pass 24 @ c676134.

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

---

## S-7.04-FU-PE-CONNECTOR — Adversarial Pass 25 (2026-07-08)

**Verdict:** HAS_FINDINGS — 1 LOW [doc-drift]

**Code HEAD:** f2f0ba6 (unchanged — story-only fix)

### Finding Progression (P25)

| Pass | Code HEAD | Findings | Severity | Streak | Remediation |
|------|-----------|----------|----------|--------|-------------|
| 25 | f2f0ba6 | 1 | LOW [doc-drift] | reset 0/3 | story v1.23 (doc-only, code HEAD unchanged) |

**Trajectory shorthand (P1–P25):** 7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1/0/1/1/0/1/1/1/1

### Finding F-P25-001 LOW [doc-drift]

**What:** The story's "Estimated total test count: 15–25" roll-up summary (~L678) contradicted the enumerated test tables and code-verified counts. The placement note forecast of 15–25 was carried forward unchanged into the summary section while the actual delivered count (28 net-new + 1 migrated) was derivable from the enumerated tables and confirmed by grep:
- `internal/upstreamdial/connector_test.go`: `grep -cE "^func Test"` = **18**
- `cmd/switchboard/router_pe_connector_test.go`: `grep -cE "^func Test"` = **9**
- `internal/testenv/testenv_test.go`: **+1** (`TestRouterHandle_Restart_TwicePE`, F-P2-001 addition)
- Total: **28 net-new + 1 migrated** (vs forecast 15–25)

The overage of ~10 tests above the upper bound was driven by adversarial-remediation hardening not in the pre-implementation forecast: F-P1-004×4 (`TestNextBackoff_*`), F-P1-006 (`TestConnector_EC004_DropToZero_ModeEEmission`), F-P2-001×2 (`TestConnector_Stop_Idempotent` + `TestRouterHandle_Restart_TwicePE`), F-P4-001 (`TestConnector_NoEC004OnGracefulStop`), F-P5-001 (`TestConnector_ReloadAddrs_StormNoDeadlock`), F-P11-001 (`TestScanForLine_DetectsEFWD001ProductionEmission`).

**Remediation (story-only fix, code HEAD f2f0ba6 unchanged):**
- Summary rewritten: "Delivered total test count: 28 net-new + 1 migrated" with per-file code-verified breakdown.
- Explicit overage note added preserving the forecast comparison: placement forecast was accurate for the pre-implementation surface; post-adversarial additions are expected and appropriate.
- Story → v1.23. Full story sweep confirmed no stale "15–25" or "Estimated total" propagation into other story sections.

**P25 verification results:**
- All nine standing bars green: full CI gate (no code changes required), census re-derivation 24/24, absence-assertion keys verbatim, symbol resolution, claim→code blast-radius, double-liveness, citation-orthography both forms, classification-consistency, POL-002 sync.
- P24 fix verified holding (spot-checked: relative locator retired, connector.go perimeter-complete notation intact).
- Code behavior unchanged since P17. Code HEAD f2f0ba6. Story HEAD v1.23.

**Cycle ledger:** 25 passes, 35 findings (7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1/0/1/1/0/1/1/1/1), zero open. Streak reset 0/3.

**Awaiting:** adversary pass 26 @ f2f0ba6 (streak 0/3)

---

## S-7.04-FU-PE-CONNECTOR — Adversarial Pass 26 (2026-07-08)

**Verdict:** HAS_FINDINGS — 1 MED [doc-drift]

**Code HEAD:** f2f0ba6 (adversary pass @ this SHA; code fix 849e095 comment-only)

### Finding Progression (P26)

| Pass | Code HEAD | Findings | Severity | Streak | Remediation |
|------|-----------|----------|----------|--------|-------------|
| 26 | f2f0ba6 | 1 | MED [doc-drift] | reset 0/3 | story v1.24 + code comment 849e095 (comment-only) |

**Trajectory shorthand (P1–P26):** 7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1/0/1/1/0/1/1/1/1/1

### Finding F-P26-001 MED [doc-drift]

**What:** The bootstrap frame type used at `Connector.dialLoop` construction is `halfchannel.FrameTypeData` (a placeholder, per the shipped-deferral note added at v1.2), yet the story's normative specification cited the phantom symbol `FrameTypePEConnect` as the delivered value in two load-bearing locations:

1. **Q6 binding** (Placement Note §Q6) — stated that the bootstrap frame type was `frame.FrameTypePEConnect`, implying the distinct constant was already defined and shipped.
2. **AC-001 PC-2** — postcondition 2 cited `FrameTypePEConnect` as the concrete delivered postcondition.

`grep -rn FrameTypePEConnect` across the entire repository returns **0 hits** — the symbol does not exist anywhere in the codebase. The shipped-deferral note introduced at story v1.2 covered only the Envelope fields (`upstream_id`, `session_id`); the `ChannelFrame.FrameType` field was not enumerated as deferred, leaving the frame-type citation unreconciled.

**Adjudication rationale:** `internal/frame` is outside the story perimeter (perimeter covers `internal/upstreamdial`, `internal/testenv`, `cmd/switchboard`, and VP e2e test files). Defining a distinct `FrameTypePEConnect` constant in `internal/frame` without the receive loop that consumes it would be dead code. The not-core deferral class applies: the frame type's semantic meaning is only realizable when its consumer (the PE receive/forward loop, `S-BL.PE-RECEIVE-LOOP`) is delivered. Classifying as not-core-deferred is consistent with the Q6 original intent and the AC-001 design.

**Remediation:**

*Story (v1.24):*
- Shipped-deferral note extended to enumerate `ChannelFrame.FrameType` alongside the Envelope fields: the distinct PE-CONNECT frame type (Q6's `frame.FrameTypePEConnect`) is deferred to S-BL.PE-RECEIVE-LOOP, the consumer whose receive loop must distinguish bootstrap frames from session data.
- Q6 binding annotated with deferral marker preserving original intent: the distinct type is the goal; `FrameTypeData` is the placeholder pending S-BL.PE-RECEIVE-LOOP.
- AC-001 PC-2 annotated with deferral marker: delivered postcondition is satisfied with the `FrameTypeData` placeholder; the distinct type lands with S-BL.PE-RECEIVE-LOOP.
- Forward obligation FO-PE-LOOP-001 added to `S-BL.PE-RECEIVE-LOOP` Forward Obligations section: define the distinct PE-CONNECT bootstrap frame type and update `Connector.dialLoop` bootstrap construction to use it.

*Code (849e095, comment-only):*
- Comment amended at `dialLoop` bootstrap construction in `internal/upstreamdial/connector.go`: documents that `halfchannel.FrameTypeData` is a placeholder, the distinct type is defined in Q6 as `frame.FrameTypePEConnect`, and delivery is deferred to S-BL.PE-RECEIVE-LOOP. No behavior change.

**P26 verification results:**
- Story: deferral note covers all three deferred surfaces (Envelope fields from v1.2 + ChannelFrame.FrameType from v1.24); Q6 + AC-001 PC-2 cite placeholder explicitly with deferral markers; original intent preserved per F-P25 lesson (F-P20-001 classification-consistency).
- Code: comment-only commit; no test changes; no behavioral changes; CI gate green (no new logic).
- S-BL.PE-RECEIVE-LOOP: FO-PE-LOOP-001 row added to Forward Obligations; no version bump (stub version "0.1-backlog-stub", no changelog table — intentionally left).

**Cycle ledger:** 26 passes, 36 findings (7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1/0/1/1/0/1/1/1/1/1), zero open. Streak reset 0/3.

**Awaiting:** adversary pass 27 @ 849e095 (streak 0/3)

---

## S-7.04-FU-PE-CONNECTOR — Adversarial Pass 27 (2026-07-08)

**Verdict:** HAS_FINDINGS — 1 MED [doc-drift]

**Code HEAD:** 849e095 (unchanged — doc-only remediation)

### Finding Progression (P27)

| Pass | Code HEAD | Findings | Severity | Streak | Remediation |
|------|-----------|----------|----------|--------|-------------|
| 27 | 849e095 | 1 | MED [doc-drift] | 0/3 | story v1.25 + placement note v1.5 (doc-only, code HEAD unchanged) |
| 28 | 849e095 | 1 | LOW [doc-drift] | 0/3 | placement note v1.6 only (story + code unchanged) |

**Trajectory shorthand (P1–P28):** 7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1/0/1/1/0/1/1/1/1/1/1/1

### Finding F-P27-001 MED [doc-drift]

**What:** Normative Q6 binding (Placement Note §Q6) and AC-001 PC-2 both cited `outerassembler.Assemble(env, cf)` as the call made at `Connector.dialLoop` bootstrap construction. The real signature at `internal/outerassembler/assemble.go` is `Assemble(cf halfchannel.ChannelFrame, sackBitmap [SACKBitmapSize]byte, env Envelope) ([]byte, error)` — wrong on three counts:

1. **Reversed operand order** — `env` was cited first; real signature places `cf` first and `env` last.
2. **Missing required parameter** — `sackBitmap [SACKBitmapSize]byte` is the second argument; absent from both citations.
3. **Wrong parameter names** — implied by the reversal.

**Novel class:** An existing, resolved symbol (`outerassembler.Assemble`) cited with the WRONG SIGNATURE. This is distinct from prior symbol findings: F-P13-001 and F-P26-001 involved phantom symbols (0 grep hits) — those were phantom-symbol findings. F-P27-001 is the first wrong-signature finding in this cycle: the function exists and is used in code, but the normative citations describe a call form that is not legal Go (wrong arity and order).

**Provenance:** The false derivation originates in the Placement Note's "Derivation from the outerassembler API (assemble.go)" block, which presented:

```
// Assemble(env Envelope, cf halfchannel.ChannelFrame) ([]byte, error)
```

This signature never existed. The story inherited it for both Q6 binding and AC-001 PC-2 without independent verification against the actual source.

**Sibling relationship:** This is the same root-cause class as F-P1-001 (import-set drift inherited from placement note). The placement note is the specification authority; when the placement note contains a false derivation, both the note and any story sections citing it must be corrected at the source.

**Adjudication — MED upheld (not down-adjudicated to LOW):** Consumer story S-BL.PE-RECEIVE-LOOP will re-touch this call via forward obligation FO-PE-LOOP-001. The normative contract must accurately describe the code that satisfies it before S-BL.PE-RECEIVE-LOOP is dispatched. Down-adjudication to LOW would leave a wrong call signature in the spec that a future implementer would follow.

**Remediation:**

*Story (v1.25):*
- Q6 binding corrected: `outerassembler.Assemble(env, cf)` → `Assemble(cf, sackBitmap, env)` with parenthetical noting `sackBitmap` is a zero `[SACKBitmapSize]byte` (bootstrap frame carries no SACK), matching the delivered call in `dialLoop` in `internal/upstreamdial/connector.go`.
- AC-001 PC-2 corrected: same `Assemble(cf, sackBitmap, env)` form replacing the false two-argument reversal.
- Changelog row v1.25 added.

*Placement note (v1.5):*
- "Derivation from the outerassembler API (assemble.go)" block corrected: false signature replaced with the real signature `Assemble(cf halfchannel.ChannelFrame, sackBitmap [SACKBitmapSize]byte, env Envelope) ([]byte, error)`.
- Inline correction marker added noting the prior false form and this correction.
- "Connection established" definition list step 2 corrected to `Assemble(cf, sackBitmap, env)`.
- Changelog row v1.5 added.

**P27 verification results:**
- Co-reference sweep: two live `Assemble(` occurrences pre-fix (Q6 binding, AC-001 PC-2) both corrected. New changelog rows contain the corrected form as historical record. No other story sections carry the old form.
- Code unchanged: no code commits. `internal/upstreamdial/connector.go` delivered call is already `Assemble(cf, sackBitmap, env)` — always was.
- Placement note: false derivation corrected at source; old form preserved only in changelog as `~~...~~` historical record.

**Cycle ledger:** 27 passes, 37 findings (7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1/0/1/1/0/1/1/1/1/1/1), zero open. Streak 0/3.

**Awaiting:** adversary pass 28 @ 849e095 (streak 0/3)

---

## Pass 28 — HAS_FINDINGS @ 849e095 (streak 0/3)

**Date:** 2026-07-08

| Pass | Code HEAD | Findings | Severity | Streak | Remediation |
|------|-----------|----------|----------|--------|-------------|
| 28 | 849e095 | 1 | LOW [doc-drift] | 0/3 | placement note v1.6 only (story + code unchanged) |

### Finding F-P28-001 LOW [doc-drift]

**What:** Placement Note §Q6 "What the connect-half sends" narrative cited phantom symbol `` `FrameTypeControl` `` (0 repo hits). The real constant is `frame.FrameTypeCtl` (0x03). The `halfchannel` package aliases only `FrameTypeData` and `FrameTypeEmptyTick` — `FrameTypeControl` was never defined anywhere in the repo.

**Location:** Single occurrence in the Q6 narrative, two subsections below the F-P27-001 fix (within the same "What the connect-half sends" section). The F-P27-001 remediation fixed the Assemble call-signature in the "Derivation" block and the "Connection established" step but did not sweep the Q6 narrative prose, leaving this adjacent phantom.

**Root cause:** Partial-sweep propagation gap. F-P27-001's remediation swept the Assemble-signature occurrences but did not cover the entire Q6 section for all symbol citations. The F-P26-001 deferral framing (`FrameTypePEConnect`) was present and correctly marked deferred. `FrameTypeControl` was a separate phantom immediately adjacent, invisible to a targeted Assemble-signature sweep.

**Consumer context:** Forward obligation FO-PE-LOOP-001 in S-BL.PE-RECEIVE-LOOP references Q6 for the ChannelFrame construction. The cited constant must be correct before S-BL.PE-RECEIVE-LOOP is dispatched.

**Story/code clean:** Story artifact already uses correct `frame.FrameTypeCtl` in relevant sections and carries zero occurrences of `FrameTypeControl`. Code HEAD 849e095 unchanged throughout.

**Remediation (placement note v1.6):**
- Q6 narrative `FrameTypeControl` corrected to `frame.FrameTypeCtl` with F-P26-001 deferral framing aligned (the correction acknowledges that a distinct PE-specific frame type is deferred per F-P26-001; `frame.FrameTypeCtl` is the _current_ correct constant for the generic control-frame slot, distinct from the placeholder `FrameTypeData`).
- Inline correction marker added.
- **FULL backtick-symbol sweep of entire placement note performed** (~45 backtick-delimited symbols dispositioned):
  - All live: correct and present in repo.
  - `FrameTypePEConnect`: explicitly-deferred per F-P26-001 — KEEP.
  - `upstreamRoutersAsSet`: appears only in changelog historical row (F-P1-007 deletion) — KEEP as historical record.
  - `FrameTypeControl`: phantom — CORRECTED to `frame.FrameTypeCtl`.
- Changelog row v1.6 added.
- **Class closed:** This is the third placement-note-seeded defect in the cycle (F-P1-001 import-set, F-P27-001 signature, F-P28-001 phantom). After the full-file symbol sweep, no further phantom or wrong-signature symbols remain in the placement note. File-granularity sweep doctrine applied to this surface.

**Cycle ledger:** 28 passes, 38 findings (7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1/0/1/1/0/1/1/1/1/1/1/1), zero open. Streak 0/3.

**Awaiting:** adversary pass 29 @ 849e095 (streak 0/3)

---

## S-7.04-FU-PE-CONNECTOR — Adversarial Pass 29 (2026-07-08)

**Verdict:** HAS_FINDINGS — 1 LOW [impl-defect]

**Code HEAD:** 6b6f0cf (advanced from 849e095 — code fix required for impl-defect)

### Finding Progression (P29)

| Pass | Code HEAD @ review | Findings | Severity | Streak | Remediation |
|------|--------------------|----------|----------|--------|-------------|
| 29 | 849e095 | 1 | LOW [impl-defect] | 0/3 | code 6b6f0cf + story v1.26 |

**Trajectory shorthand (P1–P29):** 7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1/0/1/1/0/1/1/1/1/1/1/1/1

### Finding F-P29-001 LOW [impl-defect]

**What:** In `internal/upstreamdial/connector.go` `dialLoop`, the connected-count decrement (`c.connectedCount.Add(-1)`) and the drop-to-zero check (`c.connectedCount.Load() == 0`) were two separate atomic operations. With two or more upstream connections dropping near-simultaneously, both goroutines could decrement before either loaded the result — both would then observe a zero count and emit the EC-004 "mode=E (no upstream_routers configured)" line TWICE for a single logical ≥1→0 transition. This violates AC-002 PC5's single-fallback-event semantics (one emission per transition, not one per goroutine racing through the drop-to-zero moment).

**Structural blindspot:** All prior EC-004 tests were structurally incapable of catching this race:
- `TestConnector_EC004_DropToZero_ModeEEmission` (F-P1-006) used a single upstream — no concurrent interleaving possible.
- `TestConnector_NoEC004OnGracefulStop` (F-P4-001) tested graceful-Stop polarity — single live upstream, `Stop()` closes the ctx.
- `TestConnector_AllUpstreamsUnreachable_ModeE` tested never-connected case — connectedCount never goes ≥1→0.
None of these exercise ≥2 upstreams dropping simultaneously in a live-PE state.

**Fix (commit 6b6f0cf):** Transition ownership via `Add(-1)` return value: `newCount := c.connectedCount.Add(-1)`. The goroutine whose `Add(-1)` returns exactly 0 owns the ≥1→0 transition and emits EC-004. Guard is `if newCount == 0 && ctx.Err() == nil` — F-P4-001 ctx.Err() polarity rationale preserved unchanged.

**Regression test** `TestConnector_ConcurrentDropToZero_SingleEC004Emission` added to `internal/upstreamdial/connector_test.go`:
- Stress-loop with 2 upstream fixtures; closes both concurrently; asserts exactly one EC-004 emission per ≥1→0 cycle.
- RED gate: 40–50% catch rate across 180 unfixed cycles (timing-dependent race).
- Mutation-pin confirmed via stash/flip: reverting `newCount :=` to separate Load() restores RED behavior.
- Deterministic pass post-fix across all stress iterations.
- `go test -race` clean.

**FIRST code-behavior change since P17 (7c6d841).** All 29 prior passes from P18 onward either found no code defects or found doc/process defects with doc-only fixes.

**Non-finding adjudication:** Pass 29 also produced a placement note observation regarding VP-037/VP-038 `t-first` skeleton signature drift. After full perimeter analysis, this was adjudicated out-of-perimeter: the VP harness skeletons were authored at the `t testing.T` signature (pre-go.md rule 7 ctx-first enforcement); the P14 ctx-first sweep updated production call sites but not the VP skeleton comments in the story's AC-005 section. This is a VP anchor true-up item, not an adversarial finding — it touches spec artifacts outside the code perimeter and does not affect the running tests. Folded into the planned VP-037/VP-038 anchor true-up at PR time.

**Story sync → v1.26:**
- Test-surface table: `TestConnector_ConcurrentDropToZero_SingleEC004Emission` row added (NEW, F-P29-001).
- Roll-up updated: connector_test.go 18→19 tests; delivered total 28→29 net-new; adversarial-driven additions ~10→~11.
- Changelog row v1.26 added.
- Full co-reference sweep performed for consistency.

**P29 verification results:**
- Full CI gate: golangci-lint 0 issues, go vet clean, race tests green, gofumpt no diffs.
- Regression test RED-verified (40–50% catch rate over 180 unfixed cycles) and GREEN-verified post-fix.
- Mutation-pin confirmed via stash/flip.
- All prior fixes (P1–P28) verified holding.
- Streak 0/3 (HAS_FINDINGS resets).

**Cycle ledger:** 29 passes, 39 findings (7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1/0/1/1/0/1/1/1/1/1/1/1/1), zero open. Streak 0/3.

**Awaiting:** adversary pass 30 @ 6b6f0cf (streak 0/3)

---

## S-7.04-FU-PE-CONNECTOR — Adversarial Pass 30 (2026-07-08)

**Verdict:** NO_FINDINGS — streak 0/3 → 1/3

**Code HEAD:** 6b6f0cf (unchanged — zero code or story changes this pass)

### Finding Progression (P30)

| Pass | Code HEAD | Findings | Severity | Streak | Notes |
|------|-----------|----------|----------|--------|-------|
| 30 | 6b6f0cf | 0 | — | 1/3 | P29 fix deep-reviewed correct; concurrent-transition axis swept clean |

**Trajectory shorthand (P1–P30):** 7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1/0/1/1/0/1/1/1/1/1/1/1/1/0

### Summary

Adversary reviewed P29 surface (transition ownership atomic fix + regression test) from fresh context. All 11 standing bars green.

**P29 surface deep-reviewed correct:**

1. **Transition ownership under interleaved reconnect:** Distinct goroutines decrement via `Add(-1)` — each goroutine's return value is unique; exactly the goroutine whose `Add(-1)` returns 0 emits EC-004. Interleaved reconnect (concurrent dial completing while another drops) does not create a new ≥1→0 transition in the same cycle; the ownership model holds.
2. **Config-removal path (runRouter reload to zero upstreams):** `runRouter` sets the connector's address list via `ReloadAddrs`; when reloaded to empty, the connector's `dialLoop` goroutines receive ctx cancellation via the reload-path; `runRouter` itself emits exactly one mode=E via its own `runRouter` state machine (not via the connector). The connector's EC-004 path is guarded `ctx.Err() == nil` — skips cleanly during `runRouter`-driven teardown. Single-emission verified.
3. **Stress test (TestConnector_ConcurrentDropToZero_SingleEC004Emission) leak-free across all exit paths:** Both upstream fixture goroutines confirmed to exit cleanly (dial goroutines receive ctx cancel + fixture close, drain before test end); no goroutine leak under `-race` or `-count=5` repeated runs.
4. **Concurrent-transition axis swept — no sibling mutate-then-check patterns:**
   - Connect-side `Add(1)` (goroutine increments `connectedCount` on dial success): no spec'd single-event semantics for ≥0→1 transition; not an EC-004 emission site.
   - `Mode()` return value: pure `atomic.Load` — no ownership check, no emission, no sibling.
   - `reconcile` goroutine: single-goroutine by design (reconcileLoop); no concurrent mutation-then-check patterns.

**Anti-findings adjudicated (non-blocking):**

| Finding candidate | Adjudication |
|-------------------|--------------|
| `logWriter` blocking-send in `connector.go` (pre-existing send on unbuffered channel) | Pre-existing P1 pattern — not introduced by P29 fix; `dialLoop` goroutine blocks only on `logCh` send, not on EC-004 emission path; non-blocking on EC-004 emission itself. NOT a P29 regression. |
| Stale log line after reconnect race (log goroutine may emit a status line from before the reconnect completes) | Spec-correct — `logWriter` goroutine serializes log lines for the current connection lifetime; log lines from a dying connection are valid history, not duplicate EC-004 events. BC-2.09.003 does not require log-line suppression post-reconnect. |

**Also performed this burst:** Sprint-state fork reconciliation — burst rows v1.71–v1.97 (P2–P28) were appended to the frozen root sprint-state.yaml in error (ambiguous dispatch paths). That file has been annotated with a top-of-changelog ERRATUM block. Canonical live state is .factory/stories/sprint-state.yaml (v1.99). No further appends to root sprint-state.yaml.

### Outcome

- **No code changes** required.
- **No story changes** required.
- Code HEAD unchanged at 6b6f0cf. Story unchanged at v1.26.
- Cycle ledger: 30 passes, 39 findings (7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1/0/1/1/0/1/1/1/1/1/1/1/1/0), zero open.
- **Streak: 1/3.**

**Awaiting:** adversary pass 32 @ 6b6f0cf (streak 2/3)

---

## S-7.04-FU-PE-CONNECTOR — Adversarial Pass 31 (2026-07-08)

**Verdict:** NO_FINDINGS — streak 1/3 → 2/3

**Code HEAD:** 6b6f0cf (unchanged — zero code or story changes this pass)

### Finding Progression (P31)

| Pass | Code HEAD | Findings | Severity | Streak | Notes |
|------|-----------|----------|----------|--------|-------|
| 31 | 6b6f0cf | 0 | — | 2/3 | 12 bars green; concurrency interleavings independently re-derived; cross-artifact integrity verified |

**Trajectory shorthand (P1–P31):** 7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1/0/1/1/0/1/1/1/1/1/1/1/1/0/0

### Summary

All 12 standing bars green from fresh context.

**Concurrency axis independently re-derived:**

1. **2-upstream simultaneous drop:** Both goroutines call `Add(-1)`; the one whose return value is exactly 0 owns the ≥1→0 transition and emits EC-004 once. The other goroutine observes a negative return from `Add(-1)` and skips emission. Sound.
2. **Reconnect-in-window:** Distinct goroutines — one decrementing from a disconnect, another incrementing from a concurrent successful dial. Each goroutine's `Add(-1)` return is unique to that goroutine; no aliasing possible. Sound.
3. **Stop-races-genuine-loss:** If `Stop()` is called while a goroutine owns the ≥1→0 transition, `ctx.Err() != nil` and EC-004 is suppressed. The upstream count decrements legitimately — this is genuine loss (not a double-EC-004 scenario), spec-correct per BC-2.09.003 graceful-stop semantics.
4. **Exact-tie loss/teardown race:** If a drop goroutine owns the transition (returns 0) and concurrently `Stop()` fires, `ctx.Err()` is tested after `Add(-1)` returns; the ctx-cancel path correctly suppresses the emission. Benign.

**7 ctx-first perimeter files re-verified genuinely mechanical:** The P14 ctx-first sweep updated production call sites; adversary independently confirmed all 7 VP e2e test files contain only call-site signature updates — no behavioral assertions added or changed. Purely mechanical.

**Cross-artifact integrity:**
- VP-038 (concurrency convergence property) carries `lock: true` — matches story's claim that Add(-1)-return ownership is the convergence mechanism. Correct.
- VP-037 (emission correctness) carries `lock: false` (partial-discharge pending S-BL.PE-RECEIVE-LOOP) — matches story's deferral annotation for the receive-loop discharge axis. Correct.
- FO-PE-LOOP-001 in S-BL.PE-RECEIVE-LOOP Forward Obligations correctly scoped to the receive-loop story; not an open obligation on PE-CONNECTOR.

**Test-to-AC discharge mapping complete at 6b6f0cf:**
Full mapping of all AC acceptance criteria to test functions verified. No AC has an undischarged behavioral requirement.

### Anti-Finding Adjudicated

| Finding candidate | Adjudication |
|-------------------|-------------|
| S-7.04-FU-PE-CONNECTOR story priority P2 / 5 BC traces / 4 depends_on vs sprint-state entry priority P1 / 1 BC trace / 1 depends_on | **SYSTEMATIC-LAWFUL — do not re-raise.** This is authoring-set vs scheduling-set divergence: the story file captures the full implementation scope (all BCs touched, all story-level dependencies); the sprint-state entry records wave-planning priority and wave-scheduling dependencies only. Same pattern class as emission-diff vs connection-diff distinctions across all sibling FU stories. Not story drift. |

### Outcome

- **No code changes** required.
- **No story changes** required.
- Code HEAD unchanged at 6b6f0cf. Story unchanged at v1.26.
- Cycle ledger: 31 passes, 39 findings (7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1/0/1/1/0/1/1/1/1/1/1/1/1/0/0), zero open.
- **Streak: 2/3.**

**Awaiting:** adversary pass 32 @ 6b6f0cf (streak 2/3 — convergence pass)

---

## S-7.04-FU-PE-CONNECTOR — Adversarial Pass 32 (2026-07-08) — CONVERGENCE PASS

**Verdict:** NO_FINDINGS — streak 2/3 → 3/3 — **BC-5.39.001 Step 4.5 CONVERGED**

**Code HEAD:** 6b6f0cf (unchanged — zero code or story changes this pass)

### Finding Progression (P32)

| Pass | Code HEAD | Findings | Severity | Streak | Notes |
|------|-----------|----------|----------|--------|-------|
| 32 | 6b6f0cf | 0 | — | 3/3 CONVERGED | Convergence pass; prior remediations spot-checked; two false positives dismissed |

**Full trajectory shorthand (P1–P32):** 7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1/0/1/1/0/1/1/1/1/1/1/1/1/0/0/0

### Convergence-Pass Protocol

Prior remediation spot-check (three findings sampled):

| Finding | Sampled at P32 | Verdict |
|---------|----------------|---------|
| F-P24-001 (production comment relative locator → prose anchor, `connector.go`) | Prose anchor present; relative locator absent across file | HOLDING |
| F-P13-001 (phantom `buildAndWireConnector` citation in `router_sighup_test.go`) | Symbol `buildAndWireConnector` = 0 grep hits; replacement stable-anchor forms present | HOLDING |
| F-P15-001 (mutation-pin inverse-delegation assertion in `TestRunRouter_PE_RouterHandleModeReflectsLiveState`) | Mutation of `fakeConnE` return value still fails the assertion (pin-test still RED on flip); double-liveness bar holds | HOLDING |

**F-P29 fix independently re-derived (fresh context):** Adversary traced `dialLoop`'s `Add(-1)` return-value ownership from first principles. Goroutine whose `Add(-1)` returns exactly 0 uniquely owns the ≥1→0 transition; all other concurrent decrements return negative values; EC-004 emitted exactly once per ≥1→0 event. Verdict: structurally sound; no prior pass rationale needed to reach this conclusion.

**Two false-positive candidates investigated and dismissed:**

1. **Out-of-perimeter SIGHUP evidence-doc stale signature** — An annotation in the SIGHUP-reload evidence document appeared to cite a function signature that does not match the current codebase. Investigated: this annotation belongs to the sibling story S-7.04-FU-SIGHUP-RELOAD (not PE-CONNECTOR); the PE-CONNECTOR perimeter does not include SIGHUP evidence docs. Out-of-perimeter; not a finding for this story.

2. **Already-adjudicated docstring** — A docstring comment in `testenv.go` that describes the `Restart` method path was flagged as a candidate. Investigated: this was the subject of F-P16-001 (Pass 16 finding — doc comment accurately describes teardown-recreate path after P16 fix). The P16 fix is confirmed holding. Previously adjudicated; do-not-re-raise.

### Outcome

- **No code changes** required.
- **No story changes** required.
- Code HEAD unchanged at 6b6f0cf. Story unchanged at v1.26.
- **Streak: 3/3 — BC-5.39.001 Step 4.5 PER-STORY ADVERSARIAL CONVERGENCE ACHIEVED.**

---

## S-7.04-FU-PE-CONNECTOR — Convergence Summary

**Cycle closed:** 32 passes total (passes 1–32)
**Code behavior changes:** concentrated P1–P17 + P29 (13 code-touching passes); stable since 6b6f0cf
**Doc-fidelity tail:** P19–P28 (10 consecutive doc/process-gap passes, zero code changes)
**Final finding tally:** 39 findings total, all remediated, zero open

### Finding-Decay Shape

`7/3/3/1/1/2/2/1/1/1/1/1/1/1/1/1/1/0/1/1/0/1/1/1/1/1/1/1/1/0/0/0`

Initial burst P1 (7 findings, highest severity) → rapid decay P2–P4 (3/3/1) → liveness defect P5 (1) → doc straggler pair P6 (2) → process-gap stragglers P7–P10 → test-fidelity trio P11–P13/P15 → doc-drift tail P14/P16–P28 → impl-defect P29 (unexpected regression late in cycle) → clean finish P30/P31/P32.

### Findings by Class (39 total)

| Class | Count | Description |
|-------|-------|-------------|
| doc-drift | 14 | Comment prose accuracy, symbol citations, placement-note false derivations, semantic-accuracy claims; perimeter-completion achieved P24 |
| test-fidelity | 8 | Vacuous absence key P11, CI-gate errcheck P12, phantom symbol P13, orphaned fake P15, and pre-convergence test-quality findings P1/P3 |
| process-gap | 11 | Renumber stragglers (P7/P8/P9), co-reference coverage (P10), orthography gap (P19), classification mismatch (P20), import/import-prose gaps (P3), helper-deletion co-reference (P1/P3) |
| impl-defect | 6 | EC-004 graceful-stop polarity (P4), ReloadAddrs deadlock (P5), partial-discharge structural gap (P1), backoff contradiction (P2), Stop idempotency (P2), EC-004 concurrent-drop race (P29) |

### Notable Lessons Codified During Cycle

12 standing bars codified (bars 1–9 established by P18; bars 10–12 added by convergence):

| # | Bar | Codified at |
|---|-----|-------------|
| 1 | Full CI gate (lint/vet/race/fmt) | P12 |
| 2 | Census re-derivation (toolchain set-membership) | P9 |
| 3 | Absence-assertion key fidelity | P11 |
| 4 | Symbol resolution (every cited symbol grep-resolved) | P13 |
| 5 | Claim→code mapping in blast radius | P16 |
| 6 | Double-liveness (every test double wired must be observed, mutation-pinned) | P15 |
| 7 | Citation-orthography (both prefixed + bare forms) | P19 |
| 8 | Classification-consistency (all F-PNN-001 labels match ledger) | P20 |
| 9 | POL-002 sync (story version registered in STORY-INDEX each pass) | P13 |
| 10 | Convergence spot-check (3 prior findings sampled, re-derived, verified) | P32 |
| 11 | False-positive adjudication (out-of-perimeter + already-adjudicated dismissal) | P32 |
| 12 | Fresh-context re-derivation of P29 fix correctness | P32 |

### Lessons Filed Upstream (vsdd-factory issues)

| ID | Lesson | Filed at |
|----|--------|----------|
| #573 | Normative-AC symbol fidelity — wrong-signature finding (F-P27-001); existing symbol with wrong call signature; symbol-resolution bar necessary but not sufficient | P27 |
| #574 | Placement-note derivation blocks require independent verification against actual source before story citing | P28 |
| #575 | Concurrent-transition coverage — multi-upstream concurrent drop-to-zero is a structural blindspot for single-upstream-only EC tests | P29 |

---

## S-7.04-FU-PE-CONNECTOR — DELIVERED (2026-07-08)

**Merged:** PR #115 squash @ 8eb54a5 (2026-07-08T21:50:10Z). VP true-up PR #116 merged (VP-038 anchor:true, VP-037 anchor:false). Branch deleted. Cycle complete: 32 passes, 39 findings, converged P32, delivered same day. S-BL.PE-RECEIVE-LOOP unblocked.

---

## S-BL.PE-RECEIVE-LOOP — Spec-Adversarial Convergence Cycle

### Finding Progression

| Pass | Story version | Total | HIGH | MED | LOW | Streak | Remediation |
|------|---------------|-------|------|-----|-----|--------|-------------|
| 1 (spec) | v1.0 | 7 | 3 | 3 | 1 | 0/3 | note v1.1 (Q8) + story v1.1 — all remediated |
| 2 (spec) | v1.1 | 4 | 1C | 1 | 2 | 0/3 | note v1.2 (Q9 peWriteFixture) + story v1.2 — all remediated |
| 3 (spec) | v1.2 | 3 | 2 | 1 | 0 | 0/3 | note v1.3 (byte-contract fix: EncodeOuterHeader+append reconstruction + pin test) + story v1.3 — all remediated |
| 4 (spec) | v1.3 | 2 | 2 | 0 | 0 | 0/3 | note v1.4 (FrameFn return-value contract + SetFrameCallback ordering) + story v1.4 — all remediated; v1.3 remediations cleared under direct attack |
| 5 (spec) | v1.4 | 3 | 1 | 0 | 2 | 0/3 | note v1.5 (READ-error disposition) + story v1.5 + index v4.45 — remediated; streak resets 0/3 |
| 6 (spec) | v1.5 | 4 | 2 | 1 | 1 | 0/3 | note v1.6 + story v1.6 + index v4.46 — remediated; streak stays 0/3 |
| 7 (spec) | v1.6 | 5 | 1 | 2 | 2 | 0/3 | note v1.7 + story v1.7 + index v4.47 — remediated; streak stays 0/3 |
| 8 (spec) | v1.7 | 2 | 0 | 1 | 1 | 0/3 | story v1.8 + index v4.48 (note v1.7 unchanged) — remediated; streak stays 0/3 |
| 9 (spec) | v1.8 | 1 | 0 | 1 | 0 | 0/3 | story v1.9 + index v4.49 (note v1.7 unchanged) — remediated; streak stays 0/3 |
| 10 (spec) | v1.9 | 2 | 0 | 1 | 1 | 0/3 | note v1.8 + story v1.10 + index v4.50 (both note-side) — remediated; streak stays 0/3 |
| 11 (spec) | v1.10 | 3 | 1 | 0 | 2 | 0/3 | note v1.9 + story v1.11 + index v4.51 — remediated; streak stays 0/3 |
| 12 (spec) | v1.11 | 1 | 0 | 1 | 0 | 0/3 | note v1.10 + story v1.12 + index v4.52 — remediated; streak stays 0/3 |
| 13 (spec) | v1.12 | 1 | 0 | 1 | 0 | 0/3 | note v1.11 + story v1.13 + index v4.53 — remediated; streak stays 0/3 |
| 14 (spec) | v1.13 | 1 | 0 | 1 | 0 | 0/3 | note v1.12 + story v1.14 + index v4.54 — remediated; streak stays 0/3 |
| 15 (spec) | v1.14 | 1 | 0 | 0 | 1 | 0/3 | story v1.15 + index v4.55 (note v1.12 unchanged) — remediated; streak stays 0/3 |
| 16 (spec) | v1.15 | 0 | 0 | 0 | 0 | 1/3 | CLEAN — first clean pass of the cycle; no artifact changes; streak 0/3 → 1/3 |
| 17 (spec) | v1.15 | 1 | 0 | 1 | 0 | 0/3 | note v1.13 + story v1.16 + index v4.56 — remediated; STREAK RESET 1/3 → 0/3 |
| 18 (spec) | v1.16 | 1 | 0 | 1 | 0 | 0/3 | note v1.14 + story v1.17 + index v4.57 — remediated; streak stays 0/3 |
| 19 (spec) | v1.17 | 1 | 0 | 1 | 0 | 0/3 | note v1.15 + story v1.18 (metadata-only) + index v4.58 — remediated; streak stays 0/3 |
| 20 (spec) | v1.18 | 1 | 0 | 1 | 0 | 0/3 | note v1.16 + story v1.19 (metadata-only) + index v4.59 — remediated; streak stays 0/3 |
| 21 (spec) | v1.19 | 1 | 0 | 1 | 0 | 0/3 | note v1.17 + story v1.20 (metadata-only) + index v4.60 — remediated; streak stays 0/3 |

### Trajectory Shorthand

`7→4→3→2→3→4→5→2→1→2→3→1→1→1→1→0→1→1→1→1→1` — pass 1 HAS_FINDINGS → remediated; pass 2 HAS_FINDINGS → remediated; pass 3 HAS_FINDINGS → remediated; pass 4 HAS_FINDINGS → remediated; pass 5 HAS_FINDINGS → remediated; pass 6 HAS_FINDINGS → remediated; pass 7 HAS_FINDINGS → remediated; pass 8 HAS_FINDINGS → remediated (story-side only); pass 9 HAS_FINDINGS → remediated (story-side only); pass 10 HAS_FINDINGS → remediated (both note-side); pass 11 HAS_FINDINGS → remediated; pass 12 HAS_FINDINGS → remediated; pass 13 HAS_FINDINGS → remediated; pass 14 HAS_FINDINGS → remediated; pass 15 HAS_FINDINGS → remediated (story-side only); pass 16 CLEAN — first clean pass of the cycle; streak 1/3; pass 17 HAS_FINDINGS → remediated (hostile-implementer lens; streak reset 0/3); pass 18 HAS_FINDINGS → remediated (hostile-implementer round 2: discard-continuation; streak stays 0/3); pass 19 HAS_FINDINGS → remediated (doc-drift/incompletely-discharged prior remediation: line-break-spanning Option-B residual; F-SP7-003 sweep re-certified multi-line-tolerant; streak stays 0/3); pass 20 HAS_FINDINGS → remediated (doc-drift/incompletely-discharged prior remediation: v1.5 READ-error block unannotated after v1.6 superseded it; 17-block class-closure sweep; streak stays 0/3); pass 21 HAS_FINDINGS → remediated (doc-drift/incomplete sweep-completeness certification: v1.16 table missed four binding headers; table extended rows 18-21 + canonical grep pattern + meta-hit note; all four blocks current; streak stays 0/3); pass 22 next vs {v1.20, note v1.17}.

**Pass 20 detail section:** pass-19's multi-line retracted-mechanism sweep generalised into a full versioned-block-supersession sweep found the stale v1.5 READ-error block — three linked defects (unmarked header, false teardown prose, bare-return sketch); remediation includes the cycle's first WHOLE-CLASS closure — all 17 versioned binding blocks enumerated with dispositions, fencing the 'superseded-without-in-place-annotation' class wholesale; everything else clean (canonical pattern 7/7, metadata-only verified at diff level, first-principles AC audit all five testable, 10/10 temporal-coherence claims, hostile-implementer pool exhausted since round 3); observation: three consecutive passes (18/19/20) have found exactly one MED each in note historiography while story substance has been finding-free since pass 17 — the note's 2,600 lines of layered history are now the dominant defect surface, and the 17-block sweep just fenced its largest remaining class.

**Decay trajectory (finding counts per pass):** `7 → 4 → 3 → 2 → 3 → 4 → 5 → 2 → 1` — new READ-error surface discovered at pass 5; teardown wiring layer at pass 6; observable semantics layer (mode=PE ground-truthed as config-presence-only) at pass 7; THIRD consecutive remediation carrying a false ground-truth premise (v1.4 trap → v1.5 phantom mechanism → v1.6 false observable). F-SP7-003 incomplete sweep additionally recurred inside its own remediation (2 Q1-body residuals caught on orchestrator disk-audit with expanded grep patterns). Pass 8: THREE-PREMISE-STREAK BROKEN — all three v1.7 premises ground-truthed TRUE; both findings are pass-7 residual-text incoherence (Frankenstein enumeration + stale test name), not new ground-truth defects; first pass with zero HIGH. Remediated story-side only (note v1.7 unchanged). 4 API-stall recoveries at pass 8 (2 zero-work + 2 productive-partial), all recovered via disk-audit-first. Pass 9: single finding — pre-contract descriptor text in AC-001 integration-test entries (Test-names block + Estimated Test Surface row); ran a fresh top-to-bottom implementer-read sweep; all contracts mutually consistent elsewhere; second consecutive zero-HIGH pass. Decay 2→1.

### Pass 1 Details (2026-07-08)

**Story at review:** v1.0 | **Placement note at review:** v1.0

**Verdict:** HAS_FINDINGS — 7 findings (3 HIGH, 3 MED, 1 LOW). All remediated.

#### Findings

| ID | Severity | Class | Description | Remediation |
|----|----------|-------|-------------|-------------|
| F-SP1-001 | HIGH | spec-defect | AC wiring targeted `routing.RouteFrame` but `E-FWD-001` is emitted only from `FrameArrivalHandler.OnFrameArrival` which has zero production callers — the binding anchor was undischargeable as specified. | Q8 ruling: `runRouter` constructs `NewDropCache` + `NewFrameArrivalHandler` + `WithFrameArrivalLogger`; `SetFrameCallback` closure routes through `OnFrameArrival` (not `RouteFrame`). AC-001/002/004 + FCL row 6 rewritten. Note v1.1 §Q8. |
| F-SP1-002 | HIGH | spec-gap | `Valid()` widening to `0x06` breaks two existing `frame_test.go` pins (`just_above_max` case 0x06→false; `invalids` slice containing 0x06); doc comments in `frame.go` referencing "five values" not enumerated in story. | Q3 blast-radius enumeration added: `just_above_max` 0x06→0x07, `invalids` 0x06→0x07, five doc-comment occurrences named. FCL row 1+3 expanded. |
| F-SP1-003 | HIGH | spec-gap | ARCH-02 canonical `frame_type` table amendment obligation missing from story FCL/tasks — `FrameTypePEConnect = 0x06` goes on the wire, ARCH-02 is declared canonical source of truth for the outer-header wire format. | Q3 ARCH-02 amendment obligation added; FCL row 9 (`ARCH-02-protocol-stack.md` MODIFIED, `pe_connect=0x06`, same-commit-as-constant obligation); Task 3 added; Architecture Compliance Rules + File Structure Requirements updated. |
| F-SP1-004 | MED | doc-drift | `BC-2.09.001` cited in `AC-001` as contextual anchor but absent from frontmatter `bc_traces` and Anchors Consumed table. | BC-2.09.001 added to frontmatter `bc_traces`; Anchors Consumed table gains non-discharging contextual anchor row. |
| F-SP1-005 | MED | spec-gap | Q6 per-reconnect-iteration receive-goroutine join dropped from story — a "flapping" upstream can accumulate O(N) receive goroutines without it (goroutine-leak vector). | AC-005 PC-2 added (per-reconnect-iteration join, binding per Q6 v1.1); `TestRunRouter_PE_ReceiveLoop_LifecycleClean_OnStop` recast as flap-cycle test with rationale note; FCL row 7 flap-cycle description added. |
| F-SP1-006 | MED | doc-drift | Q1 stated "No new import row is needed" and described callback signature as `func([]byte) error`, directly contradicting Q2's ruling that `upstreamdial` gains a direct `frame` import and the signature is `type FrameFn func(hdr frame.OuterHeader, raw []byte) error`. | Q1 supersession annotation added to placement note v1.1; Q2 named as controlling spec for all import/signature details; Q1 routing-free constraint and callback-seam choice preserved. |
| F-SP1-007 | LOW | doc-drift | Orphaned test name: `TestRunRouter_PE_ReceiveLoop_LifecycleClean_OnStop` present in AC-005 test names block but absent from FCL row 7 integration tests list and Estimated Test Surface table. | Test name reconciled across AC-005 test-names block, FCL row 7, and Estimated Test Surface table. |

#### Remediation Summary

**Placement note v1.0 → v1.1:** New Q8 ruling (FrameArrivalHandler wiring — NewDropCache/NewFrameArrivalHandler/WithFrameArrivalLogger construction in runRouter; OnFrameArrival closure with single-interface set making E-FWD-001 deterministic; HMAC-bypass note; zero blast radius on RouteFrame callers); Q3 blast radius enumerated (frame_test pins + doc comments + ARCH-02); Q1 supersession annotation; Q6 flap-join rule strengthened; Appendix A +5 symbols (DropCache, NewDropCache, DefaultDropCacheSize, SVTNRoute, ErrDropCacheHit).

**Story v1.0 → v1.1:** AC-001 PC-2 rewritten to FrameArrivalHandler.OnFrameArrival wiring with full construction spec; AC-002 title + all PCs rewritten to Q8 wiring spec; AC-004 mechanism reframed (deterministic single-interface-set split-horizon block; HMAC bypass adjudicated acceptable + SEC follow-on flagged); FCL row 6 rewritten (RouteFrame closure → OnFrameArrival closure + multipath import); FCL row 9 added (ARCH-02 amendment); FCL row 3+1 expanded (frame_test/frame.go blast radius); BC-2.09.001 added to frontmatter + Anchors Consumed; AC-005 PC-2 + flap-cycle test; test names reconciled. FCL 8→9 rows.

#### Spec-Level Convergence ROI (F-SP1-001)

F-SP1-001 caught pre-code the same defect class that cost PE-CONNECTOR its AC-004 partial-discharge at impl-time (F-P1-002 in the PE-CONNECTOR cycle). The spec-adversarial pass detected that `routing.RouteFrame` cannot reach `E-FWD-001` — a structural wiring defect that would have required a full adversarial remediation cycle (potentially 5+ passes with code changes) to surface after implementation began. Q8's deterministic-exhaustion mechanism (single-interface set in the `FrameFn` closure guarantees split-horizon block on every non-bootstrap frame regardless of load) replaces the prior load-dependent framing entirely. HMAC-bypass adjudicated with SEC follow-on flag for PR — acceptable because PE upstream connections are established outbound by the connector itself, not arbitrary ingress.

Full findings: `.factory/cycles/cycle-1/adversarial-reviews/` (spec-pass-1 when authored).

### Pass 2 Details (2026-07-09)

**Story at review:** v1.1 | **Placement note at review:** v1.1

**Verdict:** HAS_FINDINGS — 4 findings (1 CRITICAL, 1 HIGH, 2 MED). All remediated.

#### Findings

| ID | Severity | Class | Description | Remediation |
|----|----------|-------|-------------|-------------|
| F-SP2-001 | CRITICAL | spec-defect | AC-004's test injection dispatched frames to the data-plane listener (`cfg.ListenAddr → netingress → RouteFrame`), physically disjoint from the dialed PE conn where Q8's `OnFrameArrival` wiring lives. The binding-anchor discharge was unreachable — arqsend `Dispatch` dials `ListenAddr` (the inbound netingress socket), not the outbound PE connection. | Q9 ruling: option (b) — `peWriteFixture` (test-local in `cmd/switchboard/router_pe_receive_test.go`) writes pre-assembled `FrameTypeData` frame directly to the accepted PE connection. arqsend obligation audited and narrowed (NOT silently retired — Q4 production-wiring ruling retained; arqsend's `Dispatch → net.Dial(ListenAddr)` injection shape superseded by Q9). S404-OBS-F + S404-LOW-1 discharged via `peWriteFixture` injection path. AC-004 precondition + both AC-004 `arqsend` occurrences rewritten. |
| F-SP2-002 | HIGH | spec-gap | No write-capable upstream fixture existed anywhere — both existing fixtures (`testenv.New` harness + dialer-only shapes) were drain-only. The Q9 injection path had no concrete fixture to anchor against. | Q9.2 specifies `peWriteFixture` struct (fields: `addr string`, `accepted chan net.Conn`, `ln net.Listener`), `startPEWriteFixture(t *testing.T) *peWriteFixture` (listener setup + goroutine to accept one conn), and `(*peWriteFixture).WriteFrame(t *testing.T, wire []byte) error` method (writes to accepted conn with 5s timeout). Placed test-local in `cmd/switchboard/router_pe_receive_test.go`. FCL row 7 expanded; Appendix A delta added. |
| F-SP2-003 | MED | spec-defect | AC-004 used `testenv.New` harness, which bypasses `runRouter` — `testenv.Restart` never calls `SetFrameCallback`. The entire `FrameArrivalHandler` wiring path (Q8's key fix) was invisible to the harness. Binding discharge for `OnFrameArrival` ACs was structurally impossible via `testenv`. | Q9.3 harness rule added to Design Constraints: every AC asserting `OnFrameArrival` (AC-001, AC-002, AC-004) MUST use the real `runRouter` goroutine pattern (not `testenv.Restart`). AC-001 and AC-002 preconditions rewritten to reference `runRouter` goroutine pattern + `peWriteFixture`. AC-005 harness adjudicated: lifecycle-only assertions; `runRouter` used for fidelity but not a harness-rule obligation; `testenv` may be used if test asserts only goroutine lifecycle. |
| F-SP2-004 | MED | doc-drift | Q3 blast-radius sweep missed two `frame_test.go` locations: (1) `TestParseOuterHeader_AcceptsAllValidFrameTypes` doc comment referencing "all five canonical" (needs "all six canonical"); (2) the 5-element `valid` slice in the same test (needs `frame.FrameTypePEConnect` as sixth element). | FCL row 3 expanded with items 6 and 7; total blast-radius locations corrected from 5 to 7; Task 7 updated. |

#### Non-Findings Adjudicated Clean (Pass 2)

| Item | Evidence | Verdict |
|------|----------|---------|
| `ForwardFunc` no-op consistent | `ForwardFunc` is a no-op at spec time; Q8 already documents that `OnFrameArrival` is the production path; no spec gap | CLEAN |
| Drop-cache duplicate semantics | `arqsend.Retransmit` creates a fresh `ChanSeq` per call; `DropCache` has no effect on first unique frame; with Q9 a single injected frame is sufficient | CLEAN |
| HMAC-bypass vs BC preconditions | PE upstream connections are outbound-established by the connector; inbound HMAC enforcement is admission-plane, not receive-loop plane; BC preconditions correctly scoped | CLEAN |
| `peIfaceID` no collision | `peIfaceID` derives from the listener address; test-local fixture uses ephemeral port; no collision with production routing entries | CLEAN |
| `routerLogger` satisfies `routing.Logger` | Interface satisfaction is structural; no behavioral gap | CLEAN |

#### Remediation Summary

**Placement note v1.1 → v1.2:** New Q9 ruling (injection topology — option (b) `peWriteFixture` test-local write to accepted PE conn; arqsend `Dispatch → net.Dial(ListenAddr)` injection shape superseded; Q4 production-wiring ruling retained; S404-OBS-F + S404-LOW-1 discharged via `peWriteFixture` path; harness rule: OnFrameArrival ACs use real `runRouter`). Q3 blast-radius completed to 7 locations (items 6–7: `TestParseOuterHeader_AcceptsAllValidFrameTypes` + `valid` slice). Summary of Rulings table updated (Q9 row added). Appendix A delta (3 new test-local symbols: `peWriteFixture`, `startPEWriteFixture`, `(*peWriteFixture).WriteFrame`).

**Story v1.1 → v1.2:** AC-004 injection rewritten (both occurrences — Q4 design constraint block and AC-004 precondition block): `net.Dial(routerListenAddr)` + `arqsend.Dispatch` closure removed; replaced with `peWriteFixture.WriteFrame(t, wire)` path; AC-004 PC-1 rewritten (`outerassembler.Assemble` call form cited); S404-OBS-F + S404-LOW-1 Anchors Consumed wording updated to Q9.4 discharge framing. FCL row 7 expanded (3 new test-local symbols). Q9.3 harness rule section added to Design Constraints. AC-001 + AC-002 preconditions rewritten to reference `runRouter` goroutine pattern. AC-005 harness adjudication documented. FCL row 3 expanded to 7 blast-radius locations. `internal/arqsend` removed from `architecture_modules` frontmatter and Library table. Token budget updated (~9k).

#### Pattern Note: Partial-Reconciliation-Relocates-Drift

Pass-1 fixed the production end of the wire (Q8 — `OnFrameArrival` wiring in `runRouter`). Pass-2 caught the test-injection end still plugged into the old socket (`arqsend.Dispatch → net.Dial(ListenAddr)` routing to netingress, not the PE conn). This is the same shape as PE-CONNECTOR P27→P28: a partial reconciliation that correctly fixes one layer exposes a previously-hidden defect in a sibling layer. The spec-adversarial cycle is working as designed — pass-1's fix revealed the injection topology defect only because it clarified what the production wiring actually was.

#### Process Note: Architect Stream Stall

The first architect dispatch for this remediation stalled at 600s with zero on-disk output (stream watchdog). This is the third stream-stall instance this cycle (evidence class: architect zero-output on non-trivial ruling work). Clean re-dispatch succeeded. No spec content was lost. The stall pattern is logged for upstream vsdd-factory tracking.

Full findings: `.factory/cycles/cycle-1/adversarial-reviews/` (spec-pass-2 when authored).

### Pass 3 Details (2026-07-09)

**Story at review:** v1.2 | **Placement note at review:** v1.2

**Verdict:** HAS_FINDINGS — 3 findings (2 HIGH, 1 MED). All remediated.

#### Findings

| ID | Severity | Class | Description | Remediation |
|----|----------|-------|-------------|-------------|
| F-SP3-001 | HIGH | spec-defect | Byte-contract contradiction: Q2 claimed `ReadOuterFrame` returns "full outer frame bytes" consistent with `netingress.ReadFrame` precedent, but `netingress.ReadFrame` returns PAYLOAD-ONLY per its own contract. Consequence: AC-004's production wiring would key the drop-cache on `crc32(payload-only)`, causing silent false-duplicate suppression on frames differing only in their outer header (e.g., different SrcAddr). AC-004 greens on a single test frame, masking the collision. | Q2 ruling: `frame.ReadOuterFrame` MUST return payload-only (consistent with `netingress.ReadFrame`); false "full-frame" claim retracted. Receive goroutine MUST reconstruct full frame: `ehdr := frame.EncodeOuterHeader(hdr)` (existing function at `8eb54a5`); `raw := append(ehdr[:], payload...)`. `FrameFn raw` is ALWAYS full outer-header+payload. Discrimination contract code block updated with reconstruction step. AC-001 PC-2 + AC-004 PC-1 rewritten. Pin test `TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader` specified: two frames differing only in `Envelope.SrcAddr` → assert ≥2 `E-FWD-001` emissions; with payload-only crc32 both would have same hash → only 1 emission (false-duplicate suppression). Added to AC-004 test-names block, Estimated Test Surface, FCL row 7, Task 10. `frame.EncodeOuterHeader` verified existing at `8eb54a5` (not new). |
| F-SP3-002 | HIGH | spec-gap | AC-005 flap-cycle test (`TestRunRouter_PE_ReceiveLoop_LifecycleClean_OnStop`) attributed to `router_pe_receive_test.go` and `peWriteFixture`, but `peWriteFixture` is a single-shot accept fixture with no disconnect seam — cannot simulate a flap cycle (connect→active→disconnect→reconnect). The test as specified was unimplementable against the attributed fixture and file. | AC-005 harness re-attributed: flap-cycle test re-homed to `connector_test.go` per the existing `heldConn + Close()` pattern already used there for connector lifecycle tests. New test name: `TestConnector_ReceiveLoop_FlapCycleJoin_NoLeak`. `peWriteFixture` explicitly de-attributed from AC-005. FCL row 7: "AC-005 is NOT in this file" note added. FCL row 5: flap-cycle test entry added to connector_test.go. AC-005 test-names block, test-level (unit-only), test-files, Estimated Test Surface, Tasks 12 and 14 all updated. |
| F-SP3-003 | MED | doc-drift | Q3 blast-radius sweep third consecutive incomplete pass: `OuterHeader.FrameType` field comment in `frame.go` enumerates type names as `"(data, ctl, arq, fec, empty-tick)"` — invisible to count/bound grep patterns; enumeration-text sweep was not explicitly specified. This is item-8; Q3 counted 7 locations. | FCL row 1 expanded with item-8: `OuterHeader.FrameType` field comment → append `, pe_connect`. FCL row 3 count "7 blast-radius" → "8 blast-radius". Task 5 updated with item-8 obligation. Task 7 updated with 8-location note. File Structure Requirements frame.go line updated. Extended sweep transcript and enumeration-aware sweep instruction recorded in Q3 to prevent future misses. |

#### Non-Findings Adjudicated Clean (Pass 3)

| Item | Evidence | Verdict |
|------|----------|---------|
| `peWriteFixture` drain-vs-WriteFrame concurrency | `peWriteFixture.WriteFrame` writes to `accepted` conn; drain reads are on a separate conn path; no concurrent access to the same socket | CLEAN |
| `Assemble → ReadOuterFrame` framing round-trip | `outerassembler.Assemble` produces wire bytes at correct offset; `frame.ReadOuterFrame` reads from same offset; end-to-end framing consistent | CLEAN |
| Zero-Envelope guard traversal | `FrameFn` invoked only after header parsed; nil-Envelope case not reachable before `ReadOuterFrame` returns successfully | CLEAN |
| Bootstrap discrimination direction | Q3 discrimination contract correctly routes `FrameTypePEConnect` to bootstrap path and `FrameTypeData` to forward path; no inversion | CLEAN |
| Keepalive interference | Empty-tick frames handled at netingress drain layer; PE receive goroutine never sees empty-tick frames; no interference with drop-cache or E-FWD-001 | CLEAN |

#### Remediation Summary

**Placement note v1.2 → v1.3:** Q2 framing-primitive section title updated and rewritten — `ReadOuterFrame` returns payload-only (consistent with `netingress.ReadFrame`); false v1.2 claim retracted; receive goroutine reconstruction obligation added (`frame.EncodeOuterHeader` + `append` — existing function, not new); `FrameFn raw` binding updated. Discrimination contract code block updated with reconstruction step. AC-005 flap-cycle test re-homed to `connector_test.go`; `peWriteFixture` de-attributed from AC-005 in FCL row 7. `OuterHeader.FrameType` field comment adjudicated item-8; Q3 blast-radius table updated to 8 locations; extended sweep transcript included; enumeration-aware sweep instruction added. Appendix A delta: `frame.EncodeOuterHeader` noted as reuse from v1.0 verification (existing at `8eb54a5`). Pass-3 adjudicated-clean section added.

**Story v1.2 → v1.3:** Q2 byte-contract pipeline propagated throughout (13 `EncodeOuterHeader` references; all byte-contract assertions updated). Pin test `TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader` added to AC-004 test-names block, Estimated Test Surface, FCL row 7, Task 10. AC-005 fully re-homed: 8 occurrences of new test name `TestConnector_ReceiveLoop_FlapCycleJoin_NoLeak` added; 3 surviving old-name occurrences are all historical/disambiguation (not replaced). FCL row 7 "AC-005 NOT in this file" note. FCL row 5 flap-cycle entry added. AC-005 test-level/test-files updated to unit-only. Item-8 `OuterHeader.FrameType` field comment in FCL row 1; blast-radius count 7→8 in FCL row 3; Task 5 and Task 7 updated. Test forecast ~8 → ~9 net-new (1 frame_test + 4 connector_test + 4 integration). Frontmatter: version 1.2→1.3; `placement_note` v1.2→v1.3.

#### Pattern Note: Injection Wire Layers (Pass 1→2→3)

Three consecutive passes each went one layer deeper on the same injection wire:
- **Pass 1 (production wiring layer):** Q8 fixed how `runRouter` wires `SetFrameCallback` → `OnFrameArrival`. The production path was wrong; fixed in the spec.
- **Pass 2 (test-injection socket layer):** F-SP2-001 caught that the test was injecting into the wrong socket (netingress/data-plane, not the PE conn). Fixed: `peWriteFixture` writes directly to the accepted PE conn.
- **Pass 3 (byte-content layer):** F-SP3-001 caught that the bytes injected would be keyed incorrectly in the drop-cache (payload-only hash → false-duplicate suppression). Fixed: reconstruction via `EncodeOuterHeader` + append before `FrameFn` invocation.

This is the diagnostic value of the pin-test discipline: F-SP3-001's AC-004 false-green mechanism (drop-cache keys on `crc32(payload-only)` while the spec claims full-frame keying) is precisely the defect class that the `TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader` pin test was designed to catch at implementation time. The spec now mandates the pin test so the false-green cannot survive the Red Gate.

Full findings: `.factory/cycles/cycle-1/adversarial-reviews/` (spec-pass-3 when authored).

### Pass 4 Details (2026-07-09)

**Story at review:** v1.3 | **Placement note at review:** v1.3

**Verdict:** HAS_FINDINGS — 2 findings (2 HIGH [spec-gap]). All remediated.

#### Findings

| ID | Severity | Class | Description | Remediation |
|----|----------|-------|-------------|-------------|
| F-SP4-001 | HIGH | spec-gap | FrameFn return-value handling unspecified. `errcheck` (mandated lint gate in `.golangci.yml` at `8eb54a5`) forces an explicit error-handling decision. The idiomatic exit-on-error form (`if err := frameFn(hdr, raw); err != nil { return }`) exits the receive loop on `ErrAllPathsSplitHorizon` — the first `frameFn` invocation returns this error and the loop terminates. This deterministically defeats the story's own byte-contract pin test `TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader`: frame B is never read, the second E-FWD-001 emission never fires, and the ≥2-emission assertion fails. The normative precedent (`netingress.ServeConn` drop-and-continue with double-count-avoidance rationale) was present in the note but not propagated to the story or made binding for the receive goroutine sketch. | Q2 binding return-value rule: non-nil `frameFn` return MUST NOT terminate the receive loop. Discard-and-continue (`_ = frameFn(hdr, raw)`) mandated, mirroring `netingress.ServeConn`'s `continue` idiom. `OnFrameArrival` already logs E-FWD-001 and EC-005 internally — double-count rationale applies; receive goroutine MUST NOT log the error. The `//nolint:errcheck` directive MUST NOT be used; blank-identifier discard satisfies `errcheck` without suppression. Exit-on-error form explicitly forbidden (FORBIDDEN code block added to Q2 and story Design Constraints). Pin test annotated as doubling as the loop-continuation pin: ≥2-emission requirement proves the loop continued after the first non-nil return. Receive goroutine sketch updated to `_ = frameFn(hdr, raw)`. |
| F-SP4-002 | HIGH | spec-gap | `SetFrameCallback` ordering relative to `Start()` unpinned. The natural insertion point (after the back-to-back `New`/`Start` in `runRouter` at `8eb54a5`) yields nil-deref panic or `-race` failure if the callback is not set before the dial goroutines are created. The placement note Q1 and Q8 described the production wiring but did not specify the ordering as a binding contract, did not name the goroutine-creation happens-before guarantee, and did not define the nil-guard posture. | Q1/Q8 SetFrameCallback ordering contract: MUST be called before `Start()`. `frameFn` field is set-once pre-launch. Goroutine-creation happens-before (Go memory model §"Goroutine creation") guarantees visibility to all goroutines launched by `Start()`. No additional field synchronization (mutex, atomic) required. Binding production wiring order in `runRouter`: `construct → SetFrameCallback → Start`. Concrete insertion point: between existing `upstreamdial.New(...)` and `connector.Start()` lines (verified at `8eb54a5` — currently adjacent with no call in between). Receive goroutine MAY assume non-nil under this ordering. Nil-guard posture: defense-in-depth silent discard (no log) as optional belt-and-suspenders. Post-Start mutation forbidden — implementer may panic or ignore but MUST NOT proceed with unsynchronized field write. |

#### Non-Findings Adjudicated Clean (Pass 4 — v1.3 remediations under direct attack)

| Item | Evidence | Verdict |
|------|----------|---------|
| Byte-contract round-trip exact (F-SP3-001) | `EncodeOuterHeader`/`ParseOuterHeader` lossless over all 44 bytes; reconstruction path produces same wire bytes that were written | CLEARED |
| Pin test valid (F-SP3-001) | `Envelope.SrcAddr` feeds into serialised outer-header bytes (verified at `8eb54a5` in `internal/outerassembler/assemble.go`); two frames differing in `SrcAddr` produce distinct crc32 checksums over full-frame bytes, yielding two distinct drop-cache misses and two independent E-FWD-001 emissions | CLEARED |
| E-FWD-001 blast-radius complete at 8 (F-SP3-003) | All 8 blast-radius locations enumerated and remediated; no ninth found; enumeration-aware sweep across `frame.go` + `frame_test.go` pattern `data, ctl` / `empty.tick` / `frame kind` returns only already-listed location | CLEARED |

#### Remediation Summary

**Placement note v1.3 → v1.4:** Q2 binding return-value contract added — discard-and-continue (`_ = frameFn(hdr, raw)`) mandated; non-nil return MUST NOT terminate loop; exit-on-error form FORBIDDEN with pin-test-defeat rationale; receive goroutine sketch updated; `netingress.ServeConn` normative precedent + double-count rationale cited. Q1/Q8 SetFrameCallback ordering contract added — MUST be called before `Start()`; set-once pre-launch; goroutine-creation happens-before covers visibility; construct→SetFrameCallback→Start production order binding; receive goroutine MAY assume non-nil; nil-guard defense-in-depth silent discard optional; post-Start mutation forbidden. Pass-4 adjudicated-clean section added. Appendix A delta: no new symbols (prior symbol table complete).

**Story v1.3 → v1.4:** Both contracts propagated throughout. Receive goroutine sketch in Q2 Design Constraints: bare `frameFn(hdr, raw)` → `_ = frameFn(hdr, raw)` with discard comment. Discrimination contract bare call → `_ = frameFn(hdr, raw)` with discard comment. New Design Constraints subsection: FrameFn return-value contract (F-SP4-001, binding). New Design Constraints subsection: SetFrameCallback Ordering Contract (F-SP4-002, binding). AC-002 PC-2 amended with insertion-point annotation (between `New(...)` and `Start()`). FCL row 4 updated (discard-and-continue + set-once pre-Start + post-Start prohibition). FCL row 5 updated (flap harness Phase 1: `SetFrameCallback` before `Start()` annotated). FCL row 6 updated (binding insertion point). AC-005 flap-cycle test name and Estimated Test Surface table updated with explicit before-`Start()` ordering. Pin test `TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader` annotated as loop-continuation pin (F-SP4-001) in AC-004 test names and Estimated Test Surface table. Frontmatter: version 1.3→1.4; `placement_note` v1.3→v1.4. Token budget ~10k → ~11k.

#### Pattern Note: Callback Contract Surface (New Surface)

Passes 1–3 addressed production wiring, test injection, and byte-content layers. Pass 4 addresses the callback CONTRACT surface — what happens at the invocation boundary itself (return value semantics, goroutine visibility ordering). This is orthogonal to all three prior layers and was untouched until the adversary specifically examined the `errcheck`-forced decision under the mandated lint gate. The mechanism is subtle: a compliant `errcheck` fix can silently defeat a spec-level pin test. The spec-adversarial cycle surfaced this class before implementation.

#### Decay Trajectory

`7 → 4 → 3 → 2`: finding count decay 7→4 (pass 1), 4→3 (pass 2), 3→2 (pass 3), 2→0 pending remediation (pass 4). New surface (callback contract) discovered at pass 4; previous three surfaces (production wiring, test injection, byte-content) confirmed clear under direct attack. Streak 0/3; pass 5 dispatches against story v1.4 + note v1.4.

Full findings: `.factory/cycles/cycle-1/adversarial-reviews/` (spec-pass-4 when authored).

---

### Pass 5 Details (2026-07-09)

**Story at review:** v1.4 | **Placement note at review:** v1.4

**Verdict:** HAS_FINDINGS — 1 HIGH, 2 LOW (both LOW adjudicated as accepted observations with rationale). Remediated.

#### Findings

| ID | Severity | Class | Description | Remediation |
|----|----------|-------|-------------|-------------|
| F-SP5-001 | HIGH | spec-gap | READ-error disposition unspecified. The v1.4 callback rule (non-nil `frameFn` return MUST NOT terminate the loop) created a wrong-direction trap at the read site: the idiomatic `if _, err := frame.ReadOuterFrame(...); err != nil { return }` exits the receive goroutine on EOF / disconnect — which is correct — but the blanket v1.4 "MUST NOT terminate" language contradicted it without qualification. An implementer applying the v1.4 rule to the read site would either introduce a blind `_ = err` discard (goroutine-leak on disconnect) or add an `//nolint:errcheck` suppression. | Note v1.4→v1.5 adds the READ-error contract: `io.EOF` / `io.ErrUnexpectedEOF` / `net.Error` (including timeout) returned by `frame.ReadOuterFrame` MUST terminate the receive goroutine (upstream disconnected or severed). The v1.4 "non-nil return MUST NOT terminate" rule is re-scoped explicitly to `frameFn` invocations only. Distinction codified: read-error = normal termination path; callback-error = drop-and-continue. New pin test `TestConnector_ReceiveLoop_ExitsOnReadError` specified in connector_test.go: inject a conn that returns `io.EOF` on read; assert goroutine exits within timeout. Story v1.5 propagates re-scoped rule to Design Constraints (FrameFn return-value contract re-scoped); receive goroutine sketch updated; AC-001 PC-3 added (loop exits on ReadOuterFrame error); Estimated Test Surface ~9→~10. |
| F-SP5-OBS-1 | LOW | spec-divergence | Bounded-read note in placement note Q2 (framing primitive describes a bounded-read precondition on `ReadOuterFrame`) diverges from the story's unbounded receive loop spec. Adversary flagged potential contract mismatch. | Accepted with rationale: the bounded-read precondition governs `ReadOuterFrame`'s internal framing discipline (reads exactly 44 header bytes then exactly `payload_length` bytes), not the receive goroutine's outer loop iteration policy. The receive goroutine exits on any read error (F-SP5-001 contract), which is the loop-exit gate — no separate bounded-read loop policy is needed. Q2 clarification annotation added. |
| F-SP5-OBS-2 | LOW | spec-completeness | `connector_test.go` fixture pattern for the new `TestConnector_ReceiveLoop_ExitsOnReadError` test not explicitly described in story (which existing fixture type to adapt — `heldConn` pattern or a new `errorConn` mock). | Clarification added to story AC-001 test-names block: `TestConnector_ReceiveLoop_ExitsOnReadError` should use a minimal `errorConn` implementing `net.Conn` with a `Read` method that returns `(0, io.EOF)` immediately, consistent with the `connector_test.go` pattern for injecting controlled errors. |

#### Non-Findings Adjudicated Clean (Pass 5 — v1.4 remediations under direct attack)

| Item | Evidence | Verdict |
|------|----------|---------|
| FrameFn return-value rule (F-SP4-001) | Discard-and-continue (`_ = frameFn(hdr, raw)`) still correct after READ-error re-scoping; the two rules are orthogonal (different invocation sites) | CLEARED |
| SetFrameCallback ordering contract (F-SP4-002) | Unchanged; construct→SetFrameCallback→Start remains binding | CLEARED |
| Byte-contract reconstruction (F-SP3-001) | EncodeOuterHeader+append path unaffected by READ-error contract | CLEARED |
| Pin test NoDuplicateSuppression (F-SP3-001) | Pin test spec valid; ≥2-emission requirement unchanged | CLEARED |
| Blast-radius 8 complete (F-SP3-003) | No new blast-radius sites; item-8 field comment obligation unchanged | CLEARED |

#### Remediation Summary

**Placement note v1.4 → v1.5:** Q2 framing-primitive section: bounded-read precondition clarification annotation added (governs internal `ReadOuterFrame` framing discipline, not outer loop policy). READ-error contract section added adjacent to v1.4 FrameFn return-value contract: `io.EOF`/`io.ErrUnexpectedEOF`/`net.Error` on `ReadOuterFrame` MUST terminate the receive goroutine; v1.4 "MUST NOT terminate" rule re-scoped to `frameFn` invocations only with FORBIDDEN-loop-exit on read-error form added. Pass-5 adjudicated-clean section added.

**Story v1.4 → v1.5:** Design Constraints FrameFn return-value contract re-scoped: "non-nil return MUST NOT terminate the receive loop" now explicitly qualified "applies to `frameFn` invocations only; read-site errors are governed by the READ-error contract below." READ-error contract subsection added (mirrors note v1.5). AC-001 PC-3 added: "receive goroutine exits when `frame.ReadOuterFrame` returns a non-nil error (upstream disconnect / EOF)." New test `TestConnector_ReceiveLoop_ExitsOnReadError` added to AC-001 test-names block with `errorConn` fixture note; Estimated Test Surface table updated ~9→~10. Frontmatter: version 1.4→1.5; `placement_note` v1.4→v1.5. STORY-INDEX v4.44→v4.45.

#### Pattern Note: Callback Contract Surface → READ-error Contract Surface (Pass 4 → Pass 5)

Each pass finds the next layer down on the same callback-seam axis:
- **Pass 1 (production wiring layer):** Q8 — how `runRouter` wires `SetFrameCallback` → `OnFrameArrival`.
- **Pass 2 (test-injection socket layer):** F-SP2-001 — test injecting into the wrong socket.
- **Pass 3 (byte-content layer):** F-SP3-001 — bytes keyed incorrectly in drop-cache.
- **Pass 4 (callback contract layer):** F-SP4-001/002 — FrameFn return-value semantics + SetFrameCallback ordering.
- **Pass 5 (read-site contract layer):** F-SP5-001 — the read-site error contract was left ambiguous by the v1.4 blanket rule; the rule that was written to protect the drop-and-continue path inadvertently created a wrong-direction trap at the read site. The spec-adversarial cycle surfaces each layer in sequence; pass 6 dispatches against a spec that is now fully specified at all five layers.

Full findings: `.factory/cycles/cycle-1/adversarial-reviews/` (spec-pass-5 when authored).

---

### Pass 6 Details (2026-07-09)

**Story at review:** v1.5 | **Placement note at review:** v1.5

**Verdict:** HAS_FINDINGS — 2 HIGH, 1 MED, 1 LOW. All remediated.

#### Findings

| ID | Severity | Class | Description | Remediation |
|----|----------|-------|-------------|-------------|
| F-SP6-001 | HIGH | spec-defect | v1.5 read-error contract specifies that `maintainConn` should close and re-establish the connection on read error — but `maintainConn` never reads from `conn`; it is a lifecycle supervisor that only calls `Start()`/`Stop()`. The read-error teardown mechanism assumed by the v1.5 contract did not exist. | Note v1.5→v1.6 adds the binding: on `ReadOuterFrame` error the receive goroutine MUST call `conn.Close()` before returning. `conn` is the `net.Conn` returned from the `upstreamdial` accept seam and is in-scope to the goroutine. `conn.Close()` races with `maintainConn`'s `Stop()`-triggered close — both are idempotent; first-close wins, second is a no-op per `net.Conn` contract. Story v1.6 propagates: AC-001 teardown obligation added; receive goroutine sketch updated. |
| F-SP6-002 | HIGH | spec-gap | `SetFrameCallback` ordering contract (F-SP4-002) is bound to `Connector` (concrete type), but `runRouter` holds a `Handle` interface value. The blast radius analysis was incomplete: does the interface shape require extending `Handle` with a `SetFrameCallback` method? If yes, `fakeConnectorHandle` (the test double implementing `Handle`) must also gain the method — potentially invalidating existing connector tests. | RULED Option A: `SetFrameCallback` remains concrete-only on `*Connector`. `runRouter` performs a type assertion to `*upstreamdial.Connector` at wiring time — justified because `runRouter` is a private function that constructs its own connector; the assertion is not a public contract obligation. `fakeConnectorHandle` (implements `Handle`) is unaffected; its `SetFrameCallback` method is not required. Story v1.6 adds the type-assertion obligation to AC-002 design constraints; FCL row 4 updated with assertion note. |
| F-SP6-003 | MED | spec-defect | AC-001 PC-3 (receive goroutine exits on `ReadOuterFrame` error) and AC-004's `Mode()` assertion are unverifiable under the `runRouter` goroutine harness — `runRouter` returns no handle to the receive goroutine, and `Mode()` is not directly observable from the test (it returns internal `r.mode` state). Both acceptance criteria require observable substitutes. | Story v1.6 codifies observable substitutes: for PC-3, close `peWriteFixture`'s connection and await goroutine exit via a channel signal or `peWriteFixture.accepted` drain; for `Mode()`, check the `mode=PE` log line emitted by `runRouter` on `SetFrameCallback` wiring (already in the test harness). AC-001 and AC-004 test patterns updated accordingly. |
| F-SP6-004 | LOW | doc-drift | Blast-radius count stated as 8 locations but two additional `frame_test.go` locations were identified: line `:501` (comment referencing the old five-value enumeration) and line `:540` (inline comment for the `invalids` table referencing stale frame-type list). Story FCL row 1 and Task 3 carried the old count; blast-radius table was incomplete. | Story v1.6 extends FCL row 1 and Task 3: blast-radius updated from 8 to 10 locations; items 9 (`:501` comment) and 10 (`:540` comment) added. |

#### Non-Findings Adjudicated Clean (Pass 6 — P1–3 transition-ownership trace)

| Item | Evidence | Verdict |
|------|----------|---------|
| P1 FrameArrivalHandler wiring (F-SP1-001) | `runRouter` construction binding unchanged; type-assertion route (F-SP6-002 Option A) does not affect OnFrameArrival wiring | CLEAN |
| P2 Q9 injection topology (F-SP2-001) | `peWriteFixture.WriteFrame` → accepted PE conn; unaffected by conn.Close() teardown binding (goroutine exits after Close; fixture operates on pre-exit window) | CLEAN |
| P3 byte-contract reconstruction (F-SP3-001) | EncodeOuterHeader+append path is inside the receive goroutine; teardown is after ReadOuterFrame error, which is before the reconstruction step — no conflict | CLEAN |

#### Remediation Summary

**Placement note v1.5 → v1.6:** F-SP6-001 teardown binding added to Q2 receive-goroutine spec: `conn.Close()` obligation on read-error exit; concurrent-close idempotency note (races with `maintainConn` Stop-triggered close — first wins, second no-op). F-SP6-002 Option A ruling added: `SetFrameCallback` concrete-only; `runRouter` type-assertion to `*Connector` justified; `fakeConnectorHandle` unaffected. Pass-6 adjudicated-clean section added.

**Story v1.5 → v1.6:** AC-001 teardown obligation propagated (PC-3 expanded with `conn.Close()` before goroutine return). AC-002 design constraints: type-assertion obligation for `SetFrameCallback` wiring added. FCL row 4 updated (assertion note). AC-001 + AC-004 test patterns updated with observable substitutes (peWriteFixture.accepted drain / mode=PE log line). FCL row 1 + Task 3: blast-radius 8→10 (items 9–10: frame_test.go :501/:540). Frontmatter: version 1.5→1.6; `placement_note` v1.5→v1.6. STORY-INDEX v4.45→v4.46.

#### Pattern Note: Descent — Teardown Wiring Layer (Pass 6)

Pass 6 attacked the layer beneath the v1.5 read-error contract remediation. Descent trajectory:
- **Pass 1 (routing target layer):** Q8 — which production symbol receives the callback.
- **Pass 2 (socket layer):** F-SP2-001 — which socket the test injects into.
- **Pass 3 (byte-content layer):** F-SP3-001 — what bytes the drop-cache keys on.
- **Pass 4 (callback contract layer):** F-SP4-001/002 — return value semantics + ordering.
- **Pass 5 (read-site contract layer):** F-SP5-001 — read-error terminates vs drop-and-continue ambiguity.
- **Pass 6 (teardown wiring layer):** F-SP6-001 — the mechanism v1.5 assumed (`maintainConn` closes) does not exist; the goroutine must close `conn` itself.

F-SP6-001 is the second consecutive instance of a remediation assuming an unbuilt mechanism (F-SP5-001 assumed `maintainConn` would handle reconnect after read-error; F-SP6-001 shows `maintainConn` never reads `conn`). Each pass finds the mechanism the previous pass relied upon does not exist one layer down.

Full findings: `.factory/cycles/cycle-1/adversarial-reviews/` (spec-pass-6 when authored).

---

### Pass 7 Details (2026-07-09)

**Story at review:** v1.6 | **Placement note at review:** v1.6

**Verdict:** HAS_FINDINGS — 1 HIGH, 2 MED, 2 LOW. All remediated.

**Key meta-finding:** Pass 7 killed the v1.6 remediation's own observability premise — mode=PE was ground-truthed as a config-presence signal (present whenever the config lists any upstream routers) rather than an establishment observable. This is the THIRD consecutive remediation carrying a false ground-truth premise: v1.4 assumed `maintainConn` handled read-error reconnect (it doesn't); v1.5 assumed `maintainConn` closes the conn on read-error (it doesn't); v1.6 assumed `mode=PE` was visible at the moment the receive goroutine's first frame arrives (it isn't — `mode=PE` is set at startup when upstreams are configured, not when the first TCP accept fires). The spec-adversarial cycle is surfacing each false premise in sequence.

**F-SP7-003 additional recurrence note:** The Option-A sweep (F-SP6-002 blast-radius) was incomplete in the note's quick-reference tables and left 2 Q1-body residuals. These residuals were caught on orchestrator disk-audit using expanded grep patterns (the architect's sweep transcript showed only table updates, not Q1-body prose). Sweep-pattern transcript discipline recorded: sweeps must include complete before/after grep evidence for prose sections, not just structured tables.

**4 API-stall recoveries this window:** architect zero-work (first dispatch) + architect partial (second dispatch, recovered); story-writer zero-work (first dispatch) + story-writer partial (second dispatch, recovered); third dispatch for each completed the remediation.

#### Findings

| ID | Severity | Class | Description | Remediation |
|----|----------|-------|-------------|-------------|
| F-SP7-001 | HIGH | spec-defect | v1.6 F-SP6-003 observable substitute for `Mode()` was specified as "check the `mode=PE` log line emitted by `runRouter` on `SetFrameCallback` wiring." Ground-truth audit: `mode=PE` is set when the config has any upstream routers (i.e., when the daemon starts in PE mode) — it is a config-presence signal, not an establishment observable. The log line fires at `runRouter` entry, before any TCP accept; it does not confirm that a PE connection has been established and the receive goroutine has started. The v1.6 observable would pass even if the receive goroutine never started. | Note v1.6→v1.7: three-observable semantics ruled BINDING for AC-001 and AC-004 test verification. Observable-1: `peWriteFixture.accepted` channel receives (proves TCP connection was accepted). Observable-2: frame delivered to `OnFrameArrival` / E-FWD-001 emitted (proves receive goroutine processed a frame). Observable-3: `peWriteFixture.accepted` drain (proves goroutine exited on read-error). `mode=PE` log line retracted as an observable for receive-goroutine establishment. Story v1.7 propagates: AC-001 and AC-004 test patterns updated with three-observable discipline; old `mode=PE` test-pattern annotation removed. |
| F-SP7-002 | MED | spec-divergence | AC-001 test-pattern section contained a parenthetical noting that `peWriteFixture.startPEWriteFixture` returns its `accepted` channel "before `Add(1)`" — a self-contradiction: `Add(1)` is the goroutine launch step inside the connector; the fixture cannot know goroutine-count state. The parenthetical was a vestigial implementation hint from an earlier spec version; it was never binding, but it created a false impression about the observable ordering. | Story v1.7: parenthetical removed; ordering rationale replaced with the three-observable semantics from F-SP7-001. |
| F-SP7-003 | MED | spec-divergence | F-SP6-002 Option-A ruling ("SetFrameCallback concrete-only; runRouter type-assertion to *Connector justified; fakeConnectorHandle unaffected") was applied to the story but the placement note quick-reference tables (Q1 summary row + Q8 summary row) still referenced the old Handle-interface shape. Orchestrator disk-audit with expanded grep patterns also caught 2 Q1-body prose residuals that the architect's sweep transcript did not show (transcript showed table updates only). | Note v1.7: Q1 summary row + Q8 summary row updated to reflect Option-A ruling (concrete-only, type-assertion route). Q1-body prose residuals corrected. Sweep-pattern transcript discipline recorded: sweeps must include complete before/after grep evidence for prose sections in addition to structured tables. |
| F-SP7-004 | LOW | doc-drift | Task 1 in story v1.6 cited placement note "v1.2" — stale by 5 versions. Correct version at pass-7 dispatch is v1.7. | Story v1.7: Task 1 citation corrected v1.2→v1.7. |
| F-SP7-005 | LOW | spec-completeness | After the receive goroutine calls `conn.Close()` on read-error (F-SP6-001 binding), there is a transient window where `mode=PE` is still set (config-presence signal) but no receive goroutine is active. The window is bounded by `keepaliveInterval` (the reconnect tick in `maintainConn`), which re-establishes the connection and restarts the receive goroutine. The spec was silent on this window. | Note v1.7: transient stale-ModePE window acknowledged and bounded. The window is `≤ keepaliveInterval` (reconnect tick); the window is intentional and spec-correct — the router stays in PE mode (config says so) and the reconnect loop will restart the receive goroutine. No behavioral change. |

#### Non-Findings Adjudicated Clean (Pass 7 — P1 traces)

| Item | Evidence | Verdict |
|------|----------|---------|
| Join structure (F-SP3-002 / AC-005) | `TestConnector_ReceiveLoop_FlapCycleJoin_NoLeak` flap-cycle harness pattern unchanged; connector.go `Stop()` join on receive goroutine via `doneCh` unaffected by observable-semantics change | CLEAN |
| Keepalive window (F-SP7-005) | Transient stale-ModePE window is bounded by `keepaliveInterval` and intentional; no goroutine leak, no unbounded state | MOSTLY-CLEAN (bounded + acknowledged) |

#### Remediation Summary

**Placement note v1.6 → v1.7:** Three-observable semantics ruling added (F-SP7-001 BINDING): Observable-1 `peWriteFixture.accepted` receive (TCP established), Observable-2 E-FWD-001 / `OnFrameArrival` delivery (receive goroutine active), Observable-3 `peWriteFixture.accepted` drain (goroutine exited). `mode=PE` log-line retracted as establishment observable (config-presence-only signal). Q1 + Q8 summary table rows updated to Option-A ruling (F-SP7-003 — concrete-only type-assertion; sweep-pattern transcript discipline recorded). Transient stale-ModePE window bounded by `keepaliveInterval` (F-SP7-005). Pass-7 adjudicated-clean section added.

**Story v1.6 → v1.7:** AC-001 and AC-004 test patterns updated with three-observable discipline; old `mode=PE` annotation removed. Task 1 citation corrected v1.2→v1.7 (F-SP7-004). AC-001 parenthetical self-contradiction removed (F-SP7-002). Frontmatter: version 1.6→1.7; `placement_note` v1.6→v1.7. STORY-INDEX v4.46→v4.47.

Full findings: `.factory/cycles/cycle-1/adversarial-reviews/` (spec-pass-7 when authored).

---

### Pass 8 Details (2026-07-09)

**Story at review:** v1.7 | **Placement note at review:** v1.7

**Verdict:** HAS_FINDINGS — 1 MED, 1 LOW. Remediated story-side only (note v1.7 unchanged).

**Key meta-finding:** Pass 8 broke the three-consecutive-false-premise streak. All three v1.7 premises were ground-truthed TRUE: (1) mode=PE has exactly two emission sites (config-presence, startup only; no third site found); (2) `peWriteFixture.accepted` receive precedes goroutine launch (TCP accept timing confirmed); (3) E-FWD-001 is assertable via the three-observable chain. Both findings are residual-text incoherence from pass-7's strike-and-annotate surgery, not new ground-truth defects. First pass with zero HIGH findings.

#### Findings

| ID | Severity | Class | Description | Remediation |
|----|----------|-------|-------------|-------------|
| F-SP8-001 | MED | spec-defect | AC-001 PC-3 'Use one of' enumeration opener survived pass-7's strike-and-annotate surgery: the justification text was struck but the enumeration structure was left live, still offering the retracted `mode=PE` log-line path alongside the three-observable discipline. The live text was Frankenstein-text — structurally an enumeration, semantically only one option was viable, yet the form implied choice. The coherence sweep caught this via grep for `mode=PE` in the live AC-001 PC-3 body. | Story v1.7→v1.8: AC-001 PC-3 restructured from enumeration to direct assertion; retracted alternatives removed from live text; accepted observable path stated as THE gate without alternatives enumeration. Note v1.7 unchanged (three-observable ruling itself is correct; only the story's AC propagation was incoherent). |
| F-SP8-002 | LOW | doc-drift | Stale flap-cycle test name `TestRunRouter_PE_ReceiveLoop_LifecycleClean_OnStop` appeared in the Estimated Test Surface roll-up table — the name used before pass-3 re-homed the test to `connector_test.go` with the new name `TestConnector_ReceiveLoop_FlapCycleJoin_NoLeak`. The story body AC-005 sections carried the correct new name; the stale name persisted only in the roll-up table. | Story v1.8: stale test name in roll-up table replaced with `TestConnector_ReceiveLoop_FlapCycleJoin_NoLeak`. |

#### Non-Findings Adjudicated Clean (Pass 8 — v1.7 premises under direct attack)

| Item | Evidence | Verdict |
|------|----------|---------|
| Three-observable table (F-SP7-001) | All three observables ground-truthed at `8eb54a5`: Observable-1 (`peWriteFixture.accepted` receive) is a channel send from the accept goroutine before goroutine launch; Observable-2 (E-FWD-001) is emitted inside `OnFrameArrival` and assertable via `TestScanForLine`; Observable-3 (drain) follows `conn.Close()` and goroutine exit. Table VERIFIED TRUE. | VERIFIED TRUE |
| accepted-timing premise (F-SP7-001) | `peWriteFixture.startPEWriteFixture` goroutine sends to `accepted` on `ln.Accept()` return; this is the TCP-accept syscall, which precedes any goroutine launch by the connector. No race. | VERIFIED TRUE |
| E-FWD-001 assertability (F-SP7-001) | E-FWD-001 is emitted by `FrameArrivalHandler.OnFrameArrival` at `8eb54a5`; assertable via existing `TestScanForLine` + `peWriteFixture.WriteFrame` injection chain per Q9. | VERIFIED TRUE |
| AC-004 precondition race-safety | AC-004 precondition requires `peWriteFixture.accepted` to be non-nil before `WriteFrame` — this is satisfied by the barrier from `startPEWriteFixture`'s goroutine completing `ln.Accept()` before the channel send; no TOCTOU. | CLEAN |
| mode=PE emission sites | grep at `8eb54a5` for mode=PE log emission: exactly two sites (runRouter startup log + SetFrameCallback wiring log); both are config-presence signals. No third site found. | VERIFIED TRUE |

#### Remediation Summary

**Placement note v1.7 → unchanged:** No changes to placement note. Three-observable ruling, Q1/Q8 Option-A, transient-window binding all remain as authored.

**Story v1.7 → v1.8:** AC-001 PC-3 restructured (F-SP8-001 — enumeration-form removed; accepted observable path stated as direct gate). Estimated Test Surface roll-up: stale flap-cycle test name replaced with `TestConnector_ReceiveLoop_FlapCycleJoin_NoLeak` (F-SP8-002). STORY-INDEX v4.47→v4.48.

#### Process Notes

**4 story-writer stalls this pass:** 2 zero-work dispatches + 2 productive-partial dispatches. All recovered via disk-audit-first pattern (read story from disk before dispatching; instruct agent to work from on-disk version). Zero lost work after recovery. Disk-audit-first now a standing dispatch protocol for story-writer remediation work after stall.

Full findings: `.factory/cycles/cycle-1/adversarial-reviews/` (spec-pass-8 when authored).

### Pass 9 Details (2026-07-10)

**Story at review:** v1.8 | **Placement note at review:** v1.7

**Verdict:** HAS_FINDINGS — 1 MED. Remediated story-side only (note v1.7 unchanged).

**Method:** Fresh top-to-bottom implementer-read sweep — every section read as if implementing for the first time, testing whether the described observable is achievable with the specified harness.

#### Findings

| ID | Severity | Class | Description | Remediation |
|----|----------|-------|-------------|-------------|
| F-SP9-001 | MED | doc-drift | AC-001 integration-test descriptors carried pre-contract text in two locations: (1) Test-names block descriptor for `TestRunRouter_PE_ReceiveLoop_ActiveAfterConnect` said "starts testenv PE router" — contradicts F-SP2-003 mandate that OnFrameArrival ACs MUST use real runRouter goroutine pattern (testenv.Restart never calls SetFrameCallback → nil FrameFn → vacuous assertion); (2) Estimated Test Surface row asserted `RouterHandle.Mode() == testenv.ModePE` as the establishment observable — Mode()-based establishment thrice-retracted across passes v1.6/v1.7/v1.8; RouterHandle has no analog under the mandated runRouter harness. Both descriptors are residual pre-v1.2/pre-v1.6 text that survived the option-A sweeps. | Story v1.8→v1.9: Test-names block descriptor replaced with runRouter-pattern description (startPEWriteFixture, peWriteFixture.WriteFrame, E-FWD-001 writer-output assertion); Estimated Test Surface row replaced with peWriteFixture.accepted establishment gate + E-FWD-001 liveness observable per binding three-observable table. STORY-INDEX v4.48→v4.49. |

#### Non-Findings Adjudicated Clean (Pass 9)

All other axes clean on the implementer-read sweep: Q8 OnFrameArrival wiring contracts mutually consistent; Q9 peWriteFixture injection path coherent across AC-001/AC-002/AC-004; READ-error disposition contract (F-SP5-001 binding) consistent across all three sketches and Design Constraints; SetFrameCallback ordering contract (F-SP4-002) unambiguous; FrameFn discard-and-continue contract (F-SP4-001) unambiguous; conn.Close() teardown wiring (F-SP6-001) consistent; three-observable table (F-SP7-001) consistent with finding: VERIFIED TRUE at pass 8; AC-004 precondition race-safety (F-SP7-002) unambiguous; all FCL rows consistent with story body; Estimated Test Surface totals consistent after row correction.

#### Remediation Summary

**Placement note v1.7 → unchanged:** No changes to placement note. Three-observable ruling, Q1/Q8 Option-A, transient-window binding, all Q9 injection topology rulings all remain as authored.

**Story v1.8 → v1.9:** AC-001 Test-names block descriptor updated (F-SP9-001 — pre-contract testenv text replaced with runRouter-pattern + peWriteFixture injection + E-FWD-001 writer-output assertion). AC-001 Estimated Test Surface row updated (F-SP9-001 — Mode()-based establishment gate replaced with peWriteFixture.accepted + E-FWD-001 liveness observable per binding three-observable table). STORY-INDEX v4.48→v4.49.

Full findings: `.factory/cycles/cycle-1/adversarial-reviews/` (spec-pass-9 when authored).

### Pass 10 Details (2026-07-10)

**Story at review:** v1.9 | **Placement note at review:** v1.7

**Verdict:** HAS_FINDINGS — 1 MED + 1 LOW. Both note-side historiography. STORY CLEAN this pass.

**Method:** Reverse-traversal (note Q-sections read bottom-up, Q9→Q1) + note-with-v1.9-eyes strategy (reading note as if implementing story v1.9 from scratch, checking every instruction is still current).

**Spec-Cycle Finding Progression (passes 1–10):**

| Pass | Date | Verdict | HIGH | MED | LOW | Story | Note | Index |
|------|------|---------|------|-----|-----|-------|------|-------|
| 1 | 2026-07-08 | HAS_FINDINGS | 3 | 3 | 1 | v1.0→v1.1 | v1.0→v1.1 | v4.40→v4.41 |
| 2 | 2026-07-08 | HAS_FINDINGS | 1 | 2 | 1 | v1.1→v1.2 | v1.1→v1.2 | v4.41→v4.42 |
| 3 | 2026-07-09 | HAS_FINDINGS | 2 | 1 | 0 | v1.2→v1.3 | v1.2→v1.3 | v4.42→v4.43 |
| 4 | 2026-07-09 | HAS_FINDINGS | 2 | 0 | 0 | v1.3→v1.4 | v1.3→v1.4 | v4.43→v4.44 |
| 5 | 2026-07-09 | HAS_FINDINGS | 1 | 0 | 2 | v1.4→v1.5 | v1.4→v1.5 | v4.44→v4.45 |
| 6 | 2026-07-09 | HAS_FINDINGS | 2 | 1 | 1 | v1.5→v1.6 | v1.5→v1.6 | v4.45→v4.46 |
| 7 | 2026-07-09 | HAS_FINDINGS | 1 | 2 | 2 | v1.6→v1.7 | v1.6→v1.7 | v4.46→v4.47 |
| 8 | 2026-07-09 | HAS_FINDINGS | 0 | 1 | 1 | v1.7→v1.8 | unchanged (v1.7) | v4.47→v4.48 |
| 9 | 2026-07-10 | HAS_FINDINGS | 0 | 1 | 0 | v1.8→v1.9 | unchanged (v1.7) | v4.48→v4.49 |
| 10 | 2026-07-10 | HAS_FINDINGS | 0 | 1 | 1 | v1.9→v1.10 (metadata) | v1.7→v1.8 | v4.49→v4.50 |
| 11 | 2026-07-10 | HAS_FINDINGS | 1 | 0 | 2 | v1.10→v1.11 | v1.8→v1.9 | v4.50→v4.51 |
| 12 | 2026-07-10 | HAS_FINDINGS | 0 | 1 | 0 | v1.11→v1.12 | v1.9→v1.10 | v4.51→v4.52 |
| 13 | 2026-07-10 | HAS_FINDINGS | 0 | 1 | 0 | v1.12→v1.13 | v1.10→v1.11 | v4.52→v4.53 |

**Trajectory shorthand:** `7→4→3→2→3→4→5→2→1→2→3→1→1`

#### Findings

| ID | Severity | Class | Description | Remediation |
|----|----------|-------|-------------|-------------|
| F-SP10-001 | MED | doc-drift | Note Q4/Q5 bodies preserved superseded test-injection instructions with NO supersession annotation. Q4 (arqsend injection path) and Q5 (testenv.New harness path) were both superseded at pass 2 by Q9's ruling that AC-004 must use peWriteFixture injection directly to accepted PE conn — but Q4/Q5 carried zero annotation flagging the supersession, unlike Q1/Q2 which received v1.2 `[SUPERSEDED BY Q9]` banners. Cold-eyes reader implementing from Q4/Q5 would follow retracted instructions. | Note v1.7→v1.8: Q4/Q5 bodies each prepended with `> **[SUPERSEDED BY Q9 at pass 2]** …` banner matching Q1/Q2 pattern. Annotation-only; no ruling content changed. |
| F-SP10-002 | LOW | doc-drift | Note frontmatter `architecture_modules` list diverged from note's own Q-rulings. `internal/arqsend` listed despite Q9.4 explicitly ruling arqsend out of scope (peWriteFixture path bypasses arqsend entirely). `internal/frame` and `internal/multipath` absent despite being in-scope per Q3 (frame.ReadOuterFrame) and implied by E-FWD-001 forward path. | Note v1.7→v1.8: frontmatter `architecture_modules` updated — `internal/arqsend` removed, `internal/frame` and `internal/multipath` added. |

#### Non-Findings Adjudicated Clean (Pass 10)

30+ citations re-verified via cold-eyes read. All story contract text clean — three-observable table, Q8 OnFrameArrival wiring, Q9 peWriteFixture injection topology, READ-error disposition (F-SP5-001 binding), SetFrameCallback ordering (F-SP4-002), FrameFn discard-and-continue (F-SP4-001), conn.Close() teardown (F-SP6-001), AC-004 E-FWD-001 precondition/postcondition, AC-005 lifecycle/doneCh, all FCL rows, Estimated Test Surface totals. Zero story contract defects.

**First story-clean pass of the cycle** (passes 1–9 all had story-side findings; pass 10 both findings are note historiography only).

**Third consecutive zero-HIGH pass** (passes 8/9/10 all zero HIGH).

**Defect surface contraction signal:** passes 1–7 found ground-truth contract defects; passes 8–9 found story residual text incoherence; pass 10 finds only note historiography (sections superseded at passes 2/7 but never annotated). Surface is contracting toward the note's historical record layer.

#### Remediation Summary

**Note v1.7 → v1.8:** Q4/Q5 supersession banners added (F-SP10-001, annotation-only). Frontmatter architecture_modules reconciled (F-SP10-002, metadata-only). No ruling content changed.

**Story v1.9 → v1.10:** Frontmatter `inputDocuments` entry for placement note updated from `v1.7` citation to `v1.8` citation (per F-SP7-004 version-pin policy — note-version citations in story frontmatter must be structural, not drift). Changelog row added. Metadata-only; no AC or contract text changed.

**STORY-INDEX v4.49 → v4.50:** S-BL.PE-RECEIVE-LOOP row updated to story v1.10 + note v1.8.

**Streak stays 0/3.** Both findings are note-side historiography (no story contract changes). Sprint-state v2.14→v2.15.

Full findings: `.factory/cycles/cycle-1/adversarial-reviews/` (spec-pass-10 when authored).

---

### Pass 11 Details (2026-07-10)

**Story at review:** v1.10 | **Placement note at review:** v1.8

**Verdict:** HAS_FINDINGS — 1 HIGH + 2 LOW. Remediated.

**New axis introduced:** Physical-realizability of prescribed test inputs. Discharge-simulation executed all 5 ACs: 4 clean, 1 unrealizable (AC-001 ExitsOnReadError injection recipe). First pass-11 agent was lost to 2 consecutive API stalls before output; fresh retry agent delivered.

**Ledger spot-check:** Zero ledger-vs-artifact drift found. All prior pass findings (F-SP1 through F-SP10) verified holding per artifact state at v1.10/v1.8.

#### Findings

| ID | Severity | Class | Description | Remediation |
|----|----------|-------|-------------|-------------|
| F-SP11-001 | HIGH | spec-defect | `TestConnector_ReceiveLoop_ExitsOnReadError` injection recipe physically unrealizable: the recipe specified writing a single byte (0xFF) to the connection, then asserting goroutine exit. Discharge-simulation showed this cannot work: `io.ReadFull` blocks until 44 bytes (the full outer-header) are available; a single-byte write leaves `ReadOuterFrame` blocked at the `io.ReadFull` call — the goroutine never reaches the read-error exit path. Additional defect: 0xFF at byte[0] routes to `ErrVersionMismatch` (byte[0] is the version field; only `0x01` is accepted) not `ErrInvalidFrameType` — the finding predated the ExitsOnVersionMismatch test by one full pass. This is a v1.5-era latent defect that survived 6 adversarial passes because prior passes validated contracts, wiring, and observables but never executed the literal injection bytes against `io.ReadFull`/`ParseOuterHeader` semantics. | Corrected recipe: write a complete 44-byte header with byte[0]=0x01 (version OK), byte[1]=0x07 (unknown FrameType, routes to `ErrInvalidFrameType`), PayloadLen=0 (no additional bytes needed). Companion pin `TestConnector_ReceiveLoop_ExitsOnVersionMismatch` adjudicated ADD: write header with byte[0]=0xFF (invalid version → `ErrVersionMismatch`); assert goroutine exits. Note v1.8→v1.9: ExitsOnReadError recipe corrected; ExitsOnVersionMismatch companion added. Story v1.10→v1.11: AC-001 test-names block + Estimated Test Surface updated; connector test count 5→6, total ~10→~11. |
| F-SP11-002 | LOW | doc-drift | Token budget comment in note was a stale v1.5-era pin (`~8k` from when arqsend was in scope and AC count was lower). Post-arqsend-removal and post-v1.8 annotation work the actual budget is ~11k. | Note v1.8→v1.9: token budget re-measured and updated. |
| F-SP11-003 | LOW | doc-drift | Note §8.2 contained a dangling pointer to a Q-section that was renumbered/removed in an earlier pass — the cross-reference pointed to a section that no longer exists under that identifier. | Note v1.8→v1.9: dangling pointer struck. |

#### Non-Findings Adjudicated Clean (Pass 11 — discharge-simulation sweep)

| AC | Discharge-simulation result |
|----|---------------------------|
| AC-001 (receive goroutine active / frames reach OnFrameArrival) | CLEAN — peWriteFixture write + E-FWD-001 three-observable chain physically realizable at 8eb54a5 |
| AC-002 (SetFrameCallback wiring) | CLEAN — runRouter construct→SetFrameCallback→Start ordering physically realizable |
| AC-003 (FO-PE-LOOP-001 discrimination) | CLEAN — FrameTypePEConnect/FrameTypeData discrimination physically realizable |
| AC-004 (E-FWD-001 exhaustion / S404 re-confirmation) | CLEAN — NoDuplicateSuppression pin test injection recipe physically realizable (EncodeOuterHeader+append produces distinct crc32 values per distinct SrcAddr) |
| AC-005 (lifecycle / doneCh) | CLEAN — FlapCycleJoin harness in connector_test.go with heldConn+Close() pattern physically realizable |

**ExitsOnReadError** (part of AC-001 test-names): UNREALIZABLE — corrected by F-SP11-001.

#### Remediation Summary

**Placement note v1.8 → v1.9:** ExitsOnReadError injection recipe corrected to complete-44-byte-header form (byte[0]=0x01, byte[1]=0x07, PayloadLen=0). ExitsOnVersionMismatch companion pin added (byte[0]=0xFF → ErrVersionMismatch, goroutine exits). Token budget re-measured (F-SP11-002). Dangling §8.2 pointer struck (F-SP11-003). Pass-11 adjudicated-clean section added.

**Story v1.10 → v1.11:** AC-001 test-names block updated: ExitsOnReadError recipe corrected; ExitsOnVersionMismatch added as companion pin. Estimated Test Surface: connector test row count 5→6; total ~10→~11. Frontmatter: version 1.10→1.11; `placement_note` v1.8→v1.9. STORY-INDEX v4.50→v4.51.

**Streak stays 0/3.** F-SP11-001 is HIGH [spec-defect] — latent-class (v1.5-era recipe defect not a remediation regression), but the decay trajectory moves 2→3 for the pass-11 count. Sprint-state v2.15→v2.16. Decay: 7→4→3→2→3→4→5→2→1→2→3.

**Process note:** First pass-11 agent lost to 2 consecutive API stalls; fresh retry agent delivered the full discharge-simulation and findings with no prior-agent output on disk to recover from.

Full findings: `.factory/cycles/cycle-1/adversarial-reviews/` (spec-pass-11 when authored).

---

### Pass 12 Details (2026-07-10)

**Story at review:** v1.11 | **Placement note at review:** v1.9

**Verdict:** HAS_FINDINGS — 1 MED. Remediated.

**Physical-realizability standing axis:** Both pass-11 corrected recipes verified REALIZABLE under direct attack — byte[0]=0x01 passes the nibble check, byte[1]=0x07 fails the amended `Valid()` upper-bound (0x06 is now the maximum valid value after FrameTypePEConnect addition), yielding `ErrInvalidFrameType` exit; `ExitsOnVersionMismatch` (byte[0]=0xFF) errors at version check before `frame_type` is inspected. All four recipe copies (AC-001 test-names block, AC-001 Estimated Test Surface, note ExitsOnReadError recipe, note ExitsOnVersionMismatch recipe) verified byte-identical. 10 frame blast-radius locations enumerated and verified byte-exact at ARCH-02 target. Pass-11 remediation held under direct attack.

**Single finding:** Defect surface now confined to cross-artifact doc prose.

#### Finding

| ID | Severity | Class | Description | Remediation |
|----|----------|-------|-------------|-------------|
| F-SP12-001 | MED | spec-completeness | ARCH-08 §6.5 amendment obligation (FCL row 1 / Task 3) specified only one edit to `internal/upstreamdial` import row — but the existing row carried a now-false parenthetical: "frame is NOT imported directly by upstreamdial — Corrected per F-P1-001". This parenthetical was accurate at pass-1 (F-P1-001 established that upstreamdial does NOT import frame directly). However, this story's implementation reverses that: upstreamdial MUST import `internal/frame` to call `frame.EncodeOuterHeader` and define `FrameTypePEConnect`. A spec that instructs "add the import row" while leaving the now-false "NOT imported" parenthetical in place produces a self-contradicting amendment. | Note v1.9→v1.10: second edit obligation added to the ARCH-08 §6.5 amendment block with binding replacement wording — the parenthetical "frame is NOT imported directly...Corrected per F-P1-001" is replaced with "frame IS imported directly by upstreamdial (added by S-BL.PE-RECEIVE-LOOP; reverses the F-P1-001 correction which was correct at PE-CONNECTOR time)". Blast-radius count 10→11 unified (item 11 is the ARCH-08 §6.5 parenthetical). Frame blast-radius stays 10 (the frame.go/frame_test.go sweep is unchanged). Story v1.11→v1.12: FCL row 1 blast-radius 10→11; Task 3 updated with second edit obligation. STORY-INDEX v4.51→v4.52. |

#### Non-Findings Adjudicated Clean (Pass 12 — pass-11 remediations under direct attack)

| Item | Evidence | Verdict |
|------|----------|---------|
| ExitsOnReadError recipe (F-SP11-001) | byte[0]=0x01 passes version nibble check; byte[1]=0x07 fails amended Valid() (max is now 0x06 after FrameTypePEConnect=0x06 addition); PayloadLen=0 → no additional read; goroutine exits ErrInvalidFrameType | REALIZABLE |
| ExitsOnVersionMismatch recipe (companion, F-SP11-001) | byte[0]=0xFF → ErrVersionMismatch returned before frame_type parsed; goroutine exits | REALIZABLE |
| Four recipe copies byte-identical | AC-001 test-names block + AC-001 Estimated Test Surface + note ExitsOnReadError + note ExitsOnVersionMismatch all carry same byte spec | IDENTICAL |
| 10 frame blast-radius locations (F-SP11-002 baseline, F-SP3-003 / F-SP1-002 history) | All 10 locations verified byte-exact at ARCH-02 target; no 11th frame.go/frame_test.go location found | EXACT |
| ARCH-02 amendment target | ARCH-02 amendment entry present and correctly targets FrameTypePEConnect=0x06 | EXACT |

#### Remediation Summary

**Placement note v1.9 → v1.10:** ARCH-08 §6.5 amendment block extended with second edit obligation — parenthetical "frame is NOT imported directly by upstreamdial — Corrected per F-P1-001" replaced with binding positive claim acknowledging the reversal. Blast-radius count 10→11 (item 11: ARCH-08 §6.5 parenthetical); frame blast-radius stays 10 (frame.go/frame_test.go sweep unchanged). Pass-12 adjudicated-clean section added.

**Story v1.11 → v1.12:** FCL row 1 blast-radius updated 10→11 (item 11 added: ARCH-08 §6.5 parenthetical). Task 3 updated with second edit obligation and binding replacement wording. Changelog row added. STORY-INDEX v4.51→v4.52.

**Streak stays 0/3.** Single MED finding; fourth single-finding pass in five (decay tail: 5→2→1→2→3→1). Sprint-state v2.16→v2.17. Decay: 7→4→3→2→3→4→5→2→1→2→3→1.

Full findings: `.factory/cycles/cycle-1/adversarial-reviews/` (spec-pass-12 when authored).

---

### Pass 13 Details (2026-07-10)

**Story at review:** v1.12 | **Placement note at review:** v1.10

**Verdict:** HAS_FINDINGS — 1 MED. Remediated.

**Method:** End-state coherence read — every spec claim traced to either a production code path or a test obligation, verifying the full package reads coherently from the perspective of an implementer arriving for the first time.

**Fix-one-instance-miss-the-siblings recurrence:** Pass 13 found the §6.6.2 sibling of F-SP12-001's §6.5 finding — both sections of ARCH-08 describe import-set constraints for `internal/upstreamdial`, and both carried the same stale "frame is NOT imported" claim. This is the third instance of the incomplete-sweep class in the spec cycle (after F-SP7-003 at pass 7 and F-SP10-001 at pass 10). Class-closure grep run after remediation: 4+4 hits across ARCH-08 §6.5 and §6.6.2 surfaces, all dispositioned; no further stale import-set claims remain. Orchestrator audit of the architect's transcript corrected an initial 3-hit undercount (line 316 `arqsend` was a benign substring match in a different context, not an ARCH-08 import-set claim).

#### Finding

| ID | Severity | Class | Description | Remediation |
|----|----------|-------|-------------|-------------|
| F-SP13-001 | MED | spec-completeness | ARCH-08 §6.6.2 permitted-importers section for `internal/upstreamdial` carried the same three stale import-set claims as §6.5: (1) parenthetical "frame is NOT imported directly by upstreamdial — Corrected per F-P1-001"; (2) absence of `internal/frame` from the permitted-importers list; (3) absence of the §6.5-style reversal annotation. F-SP12-001 added the second edit obligation to the §6.5 amendment block but did not sweep §6.6.2 — the fix-one-instance-miss-the-siblings pattern recurred. Blast-radius count advances 11→12 (item 12: ARCH-08 §6.6.2 import-set claim). | Note v1.10→v1.11: THIRD edit obligation added to the ARCH-08 amendment block — §6.6.2 permitted-importers section carries the same reversal annotation as §6.5; stale "NOT imported" parenthetical replaced with binding positive claim. Class-closure grep confirms 4+4 hits, all dispositioned; no further stale import-set targets. Orchestrator audit correction noted (+1 hit initially miscounted as 3). Story v1.12→v1.13: FCL row 1 blast-radius updated 11→12 (item 12 added: ARCH-08 §6.6.2 import-set claim). Task 3 updated with third edit obligation. STORY-INDEX v4.52→v4.53. |

#### Non-Findings Adjudicated Clean (Pass 13 — realizability sweep + end-state coherence)

| Item | Evidence | Verdict |
|------|----------|---------|
| ALL ~11 test recipes realizable | Full discharge-simulation sweep: ExitsOnReadError (complete-44-byte recipe), ExitsOnVersionMismatch, NoDuplicateSuppression, FlapCycleJoin — all physically realizable at 8eb54a5 | REALIZABLE |
| AC-002 exhaustion deterministic | SetFrameCallback→Start ordering deterministic under runRouter goroutine model; no race | CLEAN |
| AC-003 discard direction correct | FrameTypePEConnect discrimination routes to bootstrap path; FrameTypeData routes forward; no inversion | CLEAN |
| AC-004 byte-contract pin valid+distinguishing | EncodeOuterHeader+append produces distinct crc32 values for frames with distinct SrcAddr; drop-cache miss guaranteed | CLEAN |
| ARCH-02 single-row adequate | Single ARCH-02 amendment row for FrameTypePEConnect=0x06 covers all blast-radius frame.go+frame_test.go locations; no second row needed | CLEAN |
| End-state coherence otherwise clean | All spec contracts mutually consistent across Q8/Q9/three-observable/READ-error/SetFrameCallback-ordering/FrameFn-discard axes | CLEAN |

#### Remediation Summary

**Placement note v1.10 → v1.11:** ARCH-08 §6.6.2 amendment block extended with third edit obligation — same binding reversal annotation as §6.5. Class-closure grep transcript included with orchestrator audit correction (+1 hit, line 316 arqsend benign substring dispositioned). Pass-13 adjudicated-clean section added.

**Story v1.12 → v1.13:** FCL row 1 blast-radius updated 11→12 (item 12 added: ARCH-08 §6.6.2 import-set claim). Task 3 updated with third edit obligation. Changelog row added. STORY-INDEX v4.52→v4.53.

**Streak stays 0/3.** Single MED finding; fifth consecutive pass with single-finding-or-fewer and zero HIGH since pass 11's latent. Sprint-state v2.17→v2.18. Decay: 7→4→3→2→3→4→5→2→1→2→3→1→1.

Full findings: `.factory/cycles/cycle-1/adversarial-reviews/` (spec-pass-13 when authored).

---

### Pass 14 Details (2026-07-10)

**Story at review:** v1.13 | **Placement note at review:** v1.11

**Verdict:** HAS_FINDINGS — 1 MED. Remediated.

**P1a vector (cross-artifact citations outside ARCH-08):** Found BC-2.01.004:61 — the un-enumerated co-canonical sibling of the frame_type enum widening. BC-2.01.004 is the wire-format behavioral contract whose line 61 enumerates frame_type values; the story cited ARCH-02:74 as the sole amendment obligation, but F-P8-008 from the PE-CONNECTOR cycle (pass-8 precedent) had named BC-2.01.004 + ARCH-02 as a canonical pair. The frame_type enum widening (FrameTypePEConnect=0x06) must be reflected in BOTH documents in the SAME commit. This is the 4th incomplete-sweep-class instance (after F-SP7-003, F-SP10-001, F-SP13-001). Class-closure grep: two patterns ("arq=0x04, fec=0x05" and "empty_tick=0x02") × 2 hits each — BC-2.01.004:61 + ARCH-02:74, no third sibling anywhere in the spec tree.

**Blast-radius presentation settled:** "unified 12 + wire-format spec pair" — BC-2.01.004:61 paired with ARCH-02:74 as same-commit parallel obligations, not counted inside the 12. The 12 remains the frame.go + frame_test.go + ARCH-08 blast-radius; the wire-format spec pair is a separate orthogonal obligation.

**Remediation option (a) accepted:** BC-2.01.004:61 amended in SAME commit as FrameTypePEConnect; this is the story's concrete obligation, not an existing-code sweep (BC-2.01.004 governs wire-format semantics; the enum widening from 5 → 6 types must be atomic with the new constant definition).

**All other pass-14 axes HELD:** All 10+ ledger items re-confirmed. ExitsOnReadError + ExitsOnVersionMismatch recipes re-executed REALIZABLE. FCL↔Task bijection complete (9 rows × 9 tasks, no orphans). AC↔BC↔test traceability clean. POL-001 (version pin) and POL-002 (STORY-INDEX sync) both pass.

**Two orchestrator audit corrections during remediation:**
1. Architect's '8-row FCL table' → 9-row (third transcript-count catch of the spec cycle; prior catches at pass-8 [stall recovery] and pass-13 [3-hit→4-hit arqsend grep correction]).
2. Story-writer's row-141 Notes chain missing v1.14 entry — appended during story-writer dispatch.

**Sixth consecutive pass at single-finding-or-fewer. Zero HIGH since pass 11.**

#### Finding

| ID | Severity | Class | Description | Remediation |
|----|----------|-------|-------------|-------------|
| F-SP14-001 | MED | spec-completeness | BC-2.01.004:61 is the co-canonical wire-format sibling of ARCH-02:74 for the frame_type enum (F-P8-008 precedent from PE-CONNECTOR cycle: the pair is canonical for wire-format changes). The frame_type widening (FrameTypePEConnect=0x06) must be reflected in BC-2.01.004:61 in the SAME commit as FrameTypePEConnect and ARCH-02:74. The story cited ARCH-02:74 as the sole wire-format amendment obligation; BC-2.01.004:61 was uncited in story and note. Class-closure grep confirms exactly 2 hits per pattern (BC-2.01.004:61 + ARCH-02:74) — no third sibling. 4th incomplete-sweep-class instance (F-SP7-003, F-SP10-001, F-SP13-001, F-SP14-001). | Note v1.11→v1.12: BC-2.01.004:61 wire-format spec pair obligation added to the amendment block alongside ARCH-02:74; "unified 12 + wire-format spec pair" blast-radius framing documented; class-closure grep transcript included. Story v1.13→v1.14: FCL row 9 (ARCH-02 amendment) expanded to name BC-2.01.004:61 as same-commit co-obligation; Task 3 updated. Two orchestrator audit corrections applied: architect FCL row count 8→9; story Notes chain v1.14 entry appended. STORY-INDEX v4.53→v4.54. |

#### Non-Findings Adjudicated Clean (Pass 14 — all ledger items re-confirmed)

| Item | Evidence | Verdict |
|------|----------|---------|
| ExitsOnReadError recipe (F-SP11-001 corrected) | byte[0]=0x01 passes version nibble check; byte[1]=0x07 fails amended Valid() (max 0x06); PayloadLen=0 → goroutine exits ErrInvalidFrameType | REALIZABLE |
| ExitsOnVersionMismatch recipe | byte[0]=0xFF → ErrVersionMismatch before frame_type | REALIZABLE |
| FCL↔Task bijection | 9 FCL rows × 9 tasks; no orphan rows or tasks | HOLDS |
| ARCH-08 class-closure (pass-13 fence) | 4+4 hits already dispositioned; no new ARCH-08 import-set targets | CLOSED |
| POL-001 (version pins) | All artifact version citations current | PASS |
| POL-002 (STORY-INDEX sync) | story v1.14 → STORY-INDEX v4.54 | PASS |
| Blast-radius 12 unified | All 12 locations (10 frame.go/frame_test.go + ARCH-08 §6.5 + ARCH-08 §6.6.2) confirmed; BC-2.01.004:61 is outside the 12 (wire-format spec pair, separate axis) | CONFIRMED |
| AC↔BC↔test traceability | All AC discharge paths trace to BCs; all test names in FCL | CLEAN |

#### Remediation Summary

**Placement note v1.11 → v1.12:** BC-2.01.004:61 wire-format spec pair obligation added to ARCH-02 amendment block (same-commit obligation, separate from the unified-12 blast-radius). "Unified 12 + wire-format spec pair" framing documented: BC-2.01.004:61 is orthogonal to the frame.go/frame_test.go/ARCH-08 sweep, shares the same commit gate as ARCH-02:74. Class-closure grep transcript included with 2-pattern × 2-hit confirmation. Pass-14 adjudicated-clean section added.

**Story v1.13 → v1.14:** FCL row 9 (ARCH-02 amendment row) expanded with BC-2.01.004:61 as same-commit co-obligation and wire-format spec pair framing. Task 3 updated with BC-2.01.004:61 obligation note. Orchestrator audit corrections applied (FCL row count 8→9 in architect note; Notes chain v1.14 entry appended at row 141). Changelog row added. STORY-INDEX v4.53→v4.54.

**Streak stays 0/3.** Single MED finding; sixth consecutive pass at single-finding-or-fewer, zero HIGH since pass 11. Sprint-state v2.18→v2.19. Decay: 7→4→3→2→3→4→5→2→1→2→3→1→1→1.

Full findings: `.factory/cycles/cycle-1/adversarial-reviews/` (spec-pass-14 when authored).

---

### Pass 15 Details (2026-07-10)

**Story at review:** v1.14 | **Placement note at review:** v1.12

**Verdict:** HAS_FINDINGS — 1 LOW. Remediated.

**P1a attack-the-remediation vector:** The pass-14 remediation added BC-2.01.004.md as a co-obligation in FCL row 9, Task 3, and the story changelog — but omitted it from the File Structure Requirements Modified-files enumeration. This is the 5th incomplete-sweep-class instance in the spec cycle: F-SP7-003, F-SP10-001, F-SP13-001, F-SP14-001, F-SP15-001. The incoherence was introduced by the pass-14 remediation burst itself and was caught by the P1a attack-the-remediation-commit verification axis.

**Severity floor reached:** F-SP15-001 is the first finding in the spec cycle classified LOW. All prior single-finding passes (8, 9, 10, 12, 13, 14) were MED. Seventh consecutive single-finding-or-fewer pass; zero HIGH since pass 11's latent.

**ALL 14 ledger items re-verified and HOLD:**
- ExitsOnReadError + ExitsOnVersionMismatch recipes re-executed REALIZABLE (byte-contract pin traced through frame.go:92 SrcAddr bytes + on_frame_arrival.go:197 crc32-over-full-frame)
- Flap-cycle join recipe re-executed REALIZABLE (traced against connector_test.go harness template: heldConn + Close() pattern)
- Index row-141 Notes chain: CLEAN (v1.14 entry present; no gap)
- AC-001..005 cold-read: CLEAN (all contracts mutually consistent)
- Note Q3 region coherence: CLEAN (blast-radius enumeration consistent with story FCL)
- POL-001 (version pins): PASS
- POL-002 (STORY-INDEX sync): story v1.15 → STORY-INDEX v4.55

#### Finding

| ID | Severity | Class | Description | Remediation |
|----|----------|-------|-------------|-------------|
| F-SP15-001 | LOW | doc-drift | v1.14 added BC-2.01.004.md to FCL row 9 ("Modified files" column) and Task 3 ("co-obligation" note) and the story changelog — but the File Structure Requirements section's Modified-files enumeration was not updated to include `specs/behavioral-contracts/ss-02/BC-2.01.004.md`. The story specifies what files to modify; the File Structure Requirements table must enumerate every file the story will touch. Omitting BC-2.01.004.md from that enumeration makes the story incoherent: FCL says touch it, File Structure Requirements doesn't list it. 5th incomplete-sweep-class instance. | Story v1.14→v1.15: File Structure Requirements Modified-files list updated to include `specs/behavioral-contracts/ss-02/BC-2.01.004.md`. Changelog row added. STORY-INDEX v4.54→v4.55. Note v1.12 unchanged (finding is story-side only). |

#### Remediation Summary

**Story v1.14 → v1.15:** File Structure Requirements Modified-files enumeration updated with `specs/behavioral-contracts/ss-02/BC-2.01.004.md`. Changelog row added. STORY-INDEX v4.54→v4.55.

**Note v1.12 unchanged:** Finding is story-side only; placement note was not involved.

**Streak stays 0/3.** First LOW-only finding in the cycle. Seventh consecutive single-finding-or-fewer pass. Zero HIGH since pass 11. Sprint-state v2.19→v2.20. Decay: 7→4→3→2→3→4→5→2→1→2→3→1→1→1→1.

Full findings: `.factory/cycles/cycle-1/adversarial-reviews/` (spec-pass-15 when authored).

---

### Pass 16 Details (2026-07-10)

**Story at review:** v1.15 | **Placement note at review:** v1.12

**Verdict:** CLEAN — zero findings. Streak 0/3 → 1/3. First clean pass of the spec-adversarial cycle.

#### Summary

**Dispatch integrity:** Code baseline re-verified docs-only 8eb54a5..42baa8c (no behavioral changes in scope). All surfaces exercised from fresh context.

**P1a — v1.15 fix under direct attack:** File Structure Requirements Modified-files bullet for `specs/behavioral-contracts/ss-02/BC-2.01.004.md` (F-SP15-001 remediation) verified semantically consistent with FCL row 9 obligation and Task 3 co-obligation annotation. No incoherence introduced by the pass-15 burst itself.

**P1b — Negative-space audit (7 surfaces walked):**

| Surface | Resolution |
|---------|-----------|
| Goroutine spawn/join placement | Specified — Q6 / AC-005 binding; per-iteration join via doneCh |
| Stop() during in-flight ReadOuterFrame | Implementer's-choice — unblock chain is conn.Close() at :382; observable result (goroutine exits) identical under all orderings; AC-observables unaffected |
| frameFn lock discipline | Previously adjudicated — FrameFn is invoked on the receive goroutine only; no concurrent writers to the field; not a new gap |
| Reconnect/backoff timing | Implementer's-choice — keepaliveInterval governs; transient stale-ModePE window acknowledged and bounded per F-SP7-005 |
| testenv.Restart nil-frameFn path | Accept-and-drain fixture cannot deliver a frame to the receive loop; the nil-frameFn branch is structurally unreachable in the delivered test suite |
| Error-taxonomy funneling | ANY non-nil error from ReadOuterFrame takes the one exit branch; no discriminating AC assertion on error subtype — implementer's-choice on error wrapping |
| Logging discretion | No AC asserts on log content beyond E-FWD-001 via writer-output chain; logging detail is implementer's-choice |

**P1c — Token/estimate coherence:** 11 tests = 1 (router_pe_receive_test.go) + 6 (connector_test.go) + 4 (frame_test.go); blast-radius count 12 unified + wire-format spec pair (BC-2.01.004.md + ARCH-02:74); 3 spec-doc amendments (ARCH-02, ARCH-08 ×2 sections). All consistent with story Estimated Test Surface table and FCL.

**P1d — Four-way version consistency:** story v1.15 ↔ sprint-state v2.20 ↔ STATE awaiting "pass 16 (story v1.15 + note v1.12, streak 0/3)" ↔ STORY-INDEX v4.55. All four consistent at dispatch time.

**P2 — POL-001/002:** All artifact version citations current (POL-001 PASS). STORY-INDEX v4.55 row reflects story v1.15 (POL-002 PASS).

**P3 — Realizability spot-checks (2 additional recipes re-executed):**
- AC-004 exhaustion determinism: NoDuplicateSuppression pin — two frames with distinct SrcAddr produce distinct crc32 values over full outer-header+payload reconstruction path; drop-cache miss guaranteed for both → ≥2 E-FWD-001 emissions deterministic. REALIZABLE.
- PEConnectFrameDiscarded: FrameTypePEConnect discrimination routes to bootstrap path; FrameTypeData routes forward; AC-003 REALIZABLE.

**Ledger re-verification:** All 14 ledger items re-verified HOLD. Zero ledger-vs-artifact drift.

**Orchestrator spot-audit (3 claims confirmed on disk):**
1. Docs-only diff 8eb54a5..42baa8c — confirmed no behavioral source changes in story perimeter.
2. Parse-order :106-114 — frame.go ParseOuterHeader reads version byte at :106, frame_type nibble check at :114; byte[1]=0x07 fails Valid() upper-bound; confirmed at those lines.
3. Join-after-Close placement :365-383 — connector.go Stop() calls conn.Close() at :382 then waits on doneCh at :365 block; unblock chain confirmed structurally sound.

#### Outcome

- **No code changes** required.
- **No story changes** required. Story remains v1.15, note remains v1.12, index remains v4.55.
- **Streak: 0/3 → 1/3.**
- Sprint-state v2.20→v2.21. Decay: 7→4→3→2→3→4→5→2→1→2→3→1→1→1→1→0.

**Awaiting:** spec adversarial pass 17 @ {story v1.15, note v1.12} (streak 1/3)

---

### Pass 17 Details (2026-07-10)

**Story at review:** v1.15 | **Placement note at review:** v1.12

**Verdict:** HAS_FINDINGS — 1 MED. Remediated. STREAK RESET 1/3 → 0/3.

#### Finding F-SP17-001

| ID | Severity | Class | Description | Remediation |
|----|----------|-------|-------------|-------------|
| F-SP17-001 | MED | spec-gap/test-set underdetermination | Hostile-implementer lens: AC-003 discrimination contract's forward side was pinned only at FrameTypeData. A whitelist-data-only implementation (`if hdr.FrameType == FrameTypeData { forwardFrame(...) }`) passed ALL ~11 named tests while silently dropping FrameTypeCtl frames that the story's Non-Goals section explicitly promises to the S-BL.RESYNC-FRAME consumer. Under strict TDD the RED test set IS the contract; the prose sketch "forward all non-PEConnect frames" does not gate because no test exercises FrameTypeCtl forwarding. | BINDING pin test `TestConnector_ReceiveLoop_CtlFrameForwardedToCallback` added: constructs a FrameTypeCtl frame via `outerassembler.Assemble`, writes it to the PE fixture, asserts callback IS invoked (inverted assertion of `PEConnectFrameDiscarded` — the PEConnect discard path); else-branch comment updated to enumerate `empty_tick` + `type-agnostic-except-pe_connect`; connector test count 6→7; total ~11→~12. |

#### Pass 17 Confirmations

**P1b — Concurrency sweep (fresh-context):**
- `hitCountMu` and `DropCache` internal mutex: separate locks, no lock ordering hazard; receive goroutine holds `hitCountMu` during count-check only, releases before calling `frameFn`.
- `ReloadAddrs` set-diff isolation: new address set computed from config without holding any receive-path lock; stop/start cycle is the only mutation path.
- `Stop()` `stopOnce` idempotency: `sync.Once` guarantees single close of `stopCh`; concurrent callers block on the Once, only one proceeds.

**P1c — DRAIN-WIRE seam:** Non-Goals section's explicit DRAIN-WIRE forwarding promise was the source of the underdetermination gap; confirmed the seam exists and is exercised by the new pin test.

**P1d — VP traceability:** `vp_traces: []` correct — no VP pins a 5-type enum; this story's forward obligation extends Valid() to 6 types; enum independence verified.

**POL pass:** POL-001/002 confirmed for story v1.16 and index v4.56.

**2 recipes re-executed realizable:** AC-004 exhaustion determinism (NoDuplicateSuppression) and PEConnectFrameDiscarded — both hold against the updated spec.

**Architect count transcript:** Survived orchestrator audit with ZERO corrections. This is the first pass in the spec cycle where the architect's transcript count needed no correction (prior catches: pass-8 stall recovery, pass-13 3-hit→4-hit arqsend grep, pass-14 8-row→9-row FCL).

#### Remediation Summary

**Placement note v1.12 → v1.13 (architect):** Binding annotation added to the discrimination-contract section: the else-branch is type-agnostic-except-pe_connect, not data-only; pin test obligation documented; class closure noted (hostile-implementer test-set underdetermination, first instance in this spec cycle).

**Story v1.15 → v1.16 (story-writer):** `TestConnector_ReceiveLoop_CtlFrameForwardedToCallback` added to AC-003 test-names block; else-branch comment obligation in the receive-loop sketch updated; Estimated Test Surface connector count 6→7, total ~11→~12; changelog row added. **STORY-INDEX v4.55 → v4.56:** S-BL.PE-RECEIVE-LOOP row updated to story v1.16 + note v1.13.

#### Lesson: Fresh-Lens Rotation Is What Caught This

Pass 16 used the NEGATIVE-SPACE lens — walking surfaces the spec does NOT cover to verify the gap is intentional or already adjudicated. Pass 17 used the HOSTILE-IMPLEMENTER lens — constructing a malicious-but-compliant implementation to probe whether the test set is sufficient to reject it. These two lenses are complementary: CLEAN under NEGATIVE-SPACE does not certify HOSTILE-IMPLEMENTER. The underdetermination gap was invisible to negative-space reasoning (the prose said "forward all non-PEConnect frames" — no missing surface) but immediately visible to hostile-implementer reasoning (a whitelist-only impl satisfies all named tests). Fresh-lens rotation in each pass is the mechanism that found the cycle's most substantive finding since pass 11.

#### Outcome

- **Streak: 1/3 → 0/3 (RESET).** Pass-16 CLEAN does not carry; HAS_FINDINGS resets the counter.
- Sprint-state v2.21→v2.22. Decay: 7→4→3→2→3→4→5→2→1→2→3→1→1→1→1→0→1.

---

### Pass 18 Details (2026-07-10)

**Story at review:** v1.16 | **Placement note at review:** v1.13

**Verdict:** HAS_FINDINGS — 1 MED. Remediated. Streak stays 0/3.

#### Finding F-SP18-001

| ID | Severity | Class | Description | Remediation |
|----|----------|-------|-------------|-------------|
| F-SP18-001 | MED | spec-gap/test-set underdetermination | Hostile-implementer round 2: discard-side loop-continuation was unpinned. PEConnectFrameDiscarded asserted only 'FrameFn NOT invoked' — a discard-as-close implementation `{ conn.Close(); return }` passed every named test while converting each bootstrap frame into teardown+reconnect storm. This is the exact symmetric sibling of F-SP17-001: the forward-side continuation was pinned by NoDuplicateSuppression ≥2 (which requires loop continuation after a non-nil frameFn return), but the discard side had no analogue pin establishing that the loop must continue after discarding. Adversary disclosed fence-adjacency with the ledger-16 fence honestly; orchestrator verified the fence (ledger-16 fences discrimination and per-type pins, not action-continuation semantics) and confirmed F-SP18-001 is genuinely outside it. | Orchestrator-adjudicated shape: EXTEND PEConnectFrameDiscarded, not add a new test. Same conn writes PEConnect frame THEN a Data frame; assert (a) FrameFn NOT invoked for the bootstrap frame, (b) FrameFn IS invoked for the data frame. This two-frame single-connection assertion pins both discard-without-close and loop-continuation simultaneously. Counts UNCHANGED: 7 connector tests / ~12 total. AC-003 PC-4 gains explicit sentence: 'discard MUST NOT close the connection.' |

#### Pass 18 Kill Transcript

Four malicious archetypes raised and killed:
1. **Payload-only reconstruction** (strip 44-byte outer header, pass payload to frameFn): killed by NoDuplicateSuppression — DropCache keyed on full-frame crc32, payload-only reconstructed frames would not match original outer-header+payload checksums.
2. **Callback-before-check** (invoke frameFn before type discrimination): killed by PEConnectFrameDiscarded — if frameFn were invoked for PEConnect frames, the extended test would detect the bootstrap callback that should not fire.
3. **Reconnect-skip** (ignore conn.Close return, skip dial retry): killed by ExitsOnReadError PC(b) — the exit-on-error test directly pins that the goroutine exits on read error, implying prior conn.Close discipline.
4. **Ctl-pin circumvention** (make Assemble skip FrameTypeCtl or make Valid(0x03) false): killed by tracing the Ctl pin end-to-end — Assemble passthrough at :102 confirmed, Valid(0x03) evaluates to true (0x03 < 0x07 upper bound from v1.14's FrameTypePEConnect addition).

AC-002 and AC-004 count-tolerance both clean under extension (no test count change). POL-001/002 pass for story v1.17 and index v4.57.

#### Fence-Adjacency Adjudication

Adversary disclosed honestly that F-SP18-001 is adjacent to the Pass-17 Adjudicated section's forward-completeness ruling and the ledger-16 fences. Orchestrator audit confirmed:
- Ledger-16 fences: discrimination (AC-003 PC-1/2/3, FrameTypePEConnect discarded) and per-type pins (F-SP17-001 forward-continuation for FrameTypeCtl).
- F-SP18-001 scope: discard-action-continuation semantics (loop must not close conn on discard) — outside the ledger-16 fence scope. Genuinely novel gap, not a re-raise.

#### Remediation Summary

**Placement note v1.13 → v1.14 (architect, zero audit corrections — 2nd consecutive):** Discard-action-continuation class added to discrimination contract section; AC-003 PC-4 explicit connection-close prohibition documented; extend-not-add shape rationale noted; fence-adjacency adjudication recorded.

**Story v1.16 → v1.17 (story-writer, zero corrections):** PEConnectFrameDiscarded extended to two-frame assertion (PEConnect then Data, same conn); AC-003 PC-4 prohibition sentence added; Estimated Test Surface counts unchanged (7 connector / ~12 total); changelog row added. **STORY-INDEX v4.56 → v4.57:** S-BL.PE-RECEIVE-LOOP row updated to story v1.17 + note v1.14.

#### Observation: Discrimination Contract Now Symmetrically Complete

With F-SP18-001 remediated, the discrimination contract has BOTH action semantics pinned:
- Forward-and-continue (F-SP4-001 / F-SP17-001 pins): non-PEConnect frames reach frameFn AND the loop continues after non-nil return.
- Discard-and-continue (F-SP18-001 pin): PEConnect frames are discarded AND the loop continues without closing the connection.

The continuation axis is now symmetrically closed.

#### Outcome

- **Streak stays 0/3.** F-SP18-001 is MED — hostile-implementer round 2 finding.
- Sprint-state v2.22→v2.23. Decay: 7→4→3→2→3→4→5→2→1→2→3→1→1→1→1→0→1→1.

**Awaiting:** spec adversarial pass 20 @ {story v1.18, note v1.15} (streak 0/3)

---

### Pass 19 Details (2026-07-10)

**Story at review:** v1.17 | **Placement note at review:** v1.14

**Verdict:** HAS_FINDINGS — 1 MED. Remediated. Streak stays 0/3.

#### Finding F-SP19-001

| ID | Severity | Class | Description | Remediation |
|----|----------|-------|-------------|-------------|
| F-SP19-001 | MED | doc-drift/incompletely-discharged prior remediation | Note Q1 v1.1 supersession region carried a live unannotated Option-B claim ('Handle gains SetFrameCallback') SPANNING A LINE BREAK. The claim survived the F-SP7-003 sweep because all F-SP7-003 grep patterns were single-line; the two tokens 'Handle' and 'SetFrameCallback' appear on consecutive lines in the Q1 body, making them invisible to any grep that matches only within a single line. The claim directly contradicts the binding F-SP6-002 Option A ruling (SetFrameCallback is concrete-only on the Connector struct, NOT on the Handle interface) and falsely attributed the Handle placement decision to Q2. This is the 6th incomplete-sweep-class instance in the cycle (F-SP7-003, F-SP10-001, F-SP13-001, F-SP14-001, F-SP15-001, F-SP19-001) and the 2nd false sweep-completeness certification (the first being the F-SP7-003 original sweep in pass 7 which issued a class-closing claim that was later falsified at pass 10). The adversary found it by attacking the sweep methodology itself — using a joined-line (tr newline-to-space + grep) technique to expose tokens that straddle line boundaries. Orchestrator reproduced 2 hits independently using the same technique. | Note v1.14→v1.15 (architect): Option-B residual struck with strikethrough and annotated per the v1.7 sibling pattern ('~~Option B (discarded at F-SP6-002): Handle gains SetFrameCallback~~'); F-SP7-003 sweep re-certified using the canonical NEW multi-line-tolerant pattern (tr '\\n' ' ' | grep -o 'SetFrameCallback[^.]*'); post-fix transcript honestly recorded 7 hits (2 struck historical + 5 meta-references) all dispositioned; architect transcript matched orchestrator's independent grep exactly (3rd consecutive zero-correction delivery). Story v1.17→v1.18 (story-writer, metadata-only): note-version citation in inputDocuments updated v1.14→v1.15; story body was always Option-A-consistent (no story body change required). STORY-INDEX v4.57→v4.58: S-BL.PE-RECEIVE-LOOP row updated. |

#### Pass 19 Confirmations

**Realizability re-check:** Two-frame PEConnectFrameDiscarded extension (F-SP18-001 remediation) re-executed REALIZABLE — FrameTypePEConnect value 0x06 correctly handled; byte-consistency traced across all 4 story locations that reference the value (AC-003 test-names, Estimated Test Surface, FCL row sketch, task description).

**Hostile-implementer round 3 (3 archetypes exhausted):**
1. **Header mutation** (corrupt hdr.FrameType before discrimination): killed by call-site assertions — the outer frame is reconstructed via EncodeOuterHeader+append, producing a deterministic header; any corruption would require modifying the fixed-format header bytes which would invalidate the byte-contract pin.
2. **Double-invoke** (call frameFn for PEConnect frames before discarding): killed by PEConnectFrameDiscarded extended assertion — test asserts frameFn IS NOT invoked for the bootstrap frame; a double-invoke implementation would fire the callback for PEConnect and fail the assertion.
3. **Aliasing** (share a single FrameFn invocation across multiple frame types): non-observable under the current test surface — the pin tests exercise specific single-type paths; aliasing would not produce a different observable outcome without an additional test specifically targeting the aliasing scenario. Adjudicated: non-observable under current test surface; not a gap because the discrimination contract is type-keyed, not count-keyed.

**Cross-layer coherence:** All 17 remediation layers (passes 1-18) spot-checked clean. No incoherence introduced by the pass-19 note-side correction.

**POL pass:** POL-001 (version pins) and POL-002 (STORY-INDEX sync) both confirmed for note v1.15 and story v1.18.

**Ledger 1-18 hold:** No ledger item was affected by the metadata-only story update or the note annotation-only change.

#### Sweep Methodology Observation

The F-SP7-003 sweep (pass 7, first certification) used single-line grep patterns. The re-certification at pass 10 also used single-line patterns. The re-certification at pass 19 introduced the multi-line-tolerant technique and found the residual. This establishes a new canonical sweep pattern for any future SetFrameCallback / Handle interface scope sweeps:

```sh
tr '\n' ' ' < placement-note.md | grep -o 'SetFrameCallback[^.]*'
```

This pattern surfaces all SetFrameCallback occurrences regardless of whether the surrounding context spans a line boundary. The F-SP19-001 residual was invisible to 12 prior passes and 3 sweep certifications — the line-break was the hiding mechanism, not any doc structure or section boundary.

#### Process Observation: Sweep-Transcript Discipline Arc Complete

The sweep-transcript discipline arc for F-SP7-003 covers three phases:
1. **Passes 7/13/14:** Three correction rounds — architect transcript undercounted, orchestrator caught and corrected each time.
2. **Passes 17/18:** Two clean deliveries — architect transcript matched orchestrator's independent verification with zero corrections (2nd consecutive at pass 18).
3. **Pass 19:** Proactive honest over-counting — architect's transcript documented 7 hits with per-hit dispositions, making the count verifiable rather than self-certifying. Orchestrator's independent grep matched exactly. The discipline has matured from correction-dependent to proactively transparent.

#### Remediation Summary

**Placement note v1.14 → v1.15 (architect):** Option-B residual struck and annotated. F-SP7-003 sweep re-certified with multi-line-tolerant canonical pattern. 7-hit post-fix transcript with dispositions. 3rd consecutive zero-correction delivery.

**Story v1.17 → v1.18 (story-writer, metadata-only):** Note-version citation in `inputDocuments` frontmatter updated v1.14→v1.15. Story body unchanged (was always Option-A-consistent).

**STORY-INDEX v4.57 → v4.58:** S-BL.PE-RECEIVE-LOOP row updated to story v1.18 + note v1.15.

#### Outcome

- **Streak stays 0/3.** F-SP19-001 is MED — doc-drift/incompletely-discharged prior remediation. The residual was unannotated live text contradicting a binding ruling.
- Sprint-state v2.23→v2.24. Decay: 7→4→3→2→3→4→5→2→1→2→3→1→1→1→1→0→1→1→1.

**Awaiting:** spec adversarial pass 21 @ {story v1.19, note v1.16} (streak 0/3)

---

### Pass 20 Details (2026-07-10)

**Story at review:** v1.18 | **Placement note at review:** v1.15

**Verdict:** HAS_FINDINGS — 1 MED. Remediated. Streak stays 0/3.

**Method:** Applied pass-19's multi-line retracted-mechanism sweep to older note sections; swept all versioned binding blocks for unannotated supersessions.

**7th incomplete-sweep-class instance — generalizing F-SP19-001's shape:** F-SP19-001 established the pattern for line-break-spanning residuals. Pass 20 generalises it: any versioned binding block superseded by a later-version ruling without in-place annotation is a member of the same class. The v1.5 READ-error block is the instance — superseded at v1.6 by F-SP6-001 binding (conn.Close() obligation) without the three-part annotation the story's twin v1.5 header carries.

#### Finding F-SP20-001

| ID | Severity | Class | Description | Remediation |
|----|----------|-------|-------------|-------------|
| F-SP20-001 | MED | doc-drift/incompletely-discharged prior remediation | Note's v1.5 READ-error block (lines :365–421) was never annotated when v1.6 F-SP6-001 superseded it. Three linked defects: (1) header lacked 'amended v1.6' marker — the story's twin header carries 'amended v1.6' per the F-SP6-001 remediation convention; (2) live prose asserted the retracted 'dialLoop teardown closes the conn' mechanism — false, `maintainConn` is write-only at connector.go:399; (3) v1.5 sketch showed bare `return` without `_ = conn.Close()` — copy-pasteable wrong code 336 lines from the correct v1.6 sketch. | Note v1.15→v1.16 (architect): three-part annotation applied — (1) header marker 'amended v1.6 — see F-SP6-001 binding below' added; (2) retracted 'dialLoop teardown' prose struck with strikethrough and F-SP6-001 annotation; (3) v1.5 sketch banner added with pointer to correct v1.6 sketch (sketch body preserved as historical reference). CLASS-CLOSURE SWEEP of all 17 versioned binding blocks: 2 remediated, 2 previously annotated, 13 current — zero unannotated stale blocks remain. Orchestrator reconciled 17-block enumeration against independent 19-hit binding-marker grep: delta = nested sub-blocks + sweep-table meta-hits, all dispositioned. Story v1.18→v1.19 metadata-only: note pin v1.15→v1.16 in frontmatter (story:351 already carried 'amended v1.6'). STORY-INDEX v4.58→v4.59. |

#### Pass 20 Confirmations

- **v1.15 strikethrough (pass-19 remediation under direct attack):** Option-B residual struck and annotated correctly; multi-line-tolerant sweep returns zero live 'Handle gains SetFrameCallback' tokens. HOLDING.
- **Canonical pattern reconciled 7/7:** All 7 incomplete-sweep-class instances (F-SP7-003/F-SP10-001/F-SP13-001/F-SP14-001/F-SP15-001/F-SP19-001/F-SP20-001) share the fix-instance-miss-class root cause. The 17-block class-closure sweep fences the versioned-block-supersession class wholesale.
- **Story v1.18 metadata-only verified at diff level:** diff confirmed no AC or contract text changed from v1.17→v1.18; twin header in story already carried 'amended v1.6' correctly.
- **First-principles testability all five ACs:** AC-001 read-error exit (conn.Close() before return) TESTABLE; AC-002 SetFrameCallback ordering TESTABLE; AC-003 discrimination TESTABLE; AC-004 E-FWD-001 exhaustion TESTABLE; AC-005 lifecycle/doneCh TESTABLE.
- **10/10 note→story claims match:** Note v1.16 corrected READ-error binding obligations cross-checked against story v1.18 AC-001 PC-3 and Design Constraints READ-error contract — zero mismatches.
- **2 recipes re-traced realizable:** ExitsOnReadError complete-44-byte recipe and SetFrameCallback ordering both realizable at 8eb54a5.
- **Ledger 1-19 hold:** All prior pass findings verified holding at note v1.16 + story v1.18 state.
- **POL pass:** POL-001 version pins and POL-002 STORY-INDEX sync confirmed for note v1.16 and story v1.19.

#### Observation: Note Historiography Layer Now Primary Defect Surface

Three consecutive passes (18/19/20) have each found exactly one MED in note historiography while story substance has been finding-free since pass 17. The note's 2,600 lines of layered history are now the dominant defect surface. The 17-block class-closure sweep at pass 20 fences the largest remaining class (versioned-block-supersession-without-annotation) wholesale; the remaining historiography risk is isolated to within-block prose claims that predate a ruling but survived outside a versioned-block structure.

#### Outcome

- **Streak stays 0/3.** F-SP20-001 is MED — note historiography layer.
- Sprint-state v2.24→v2.25. Decay: 7→4→3→2→3→4→5→2→1→2→3→1→1→1→1→0→1→1→1→1.

**Awaiting:** spec adversarial pass 21 @ {story v1.19, note v1.16} (streak 0/3)

---

### Pass 21 Details (2026-07-10)

**Story at review:** v1.19 | **Placement note at review:** v1.16

**Verdict:** HAS_FINDINGS — 1 MED. Remediated. Streak stays 0/3.

**Method:** Sweep-of-the-sweep — pass-20's class-closure table itself certified falsely. Four binding headers unreachable by its grep patterns.

**8th incomplete-sweep-class instance — sweep certification defect:** The v1.16 class-closure sweep table declared "17 blocks, complete" but four binding-block headers could not be matched by the recorded grep patterns (`grep -nE "\*\*.*binding\b"` style). The four missed headers: `:262` FrameFn byte-contract (v1.3/F-SP3-001, structural peer of enumerated rows 3/4), `:511` Test shape, `:1812` Pin test shape, `:1928` Binding harness rule. All four verified CURRENT — no stale content was hiding behind the certification gap. The defect is purely in the sweep table's completeness claim. This is the 3rd false sweep-completeness certification in the cycle (prior: F-SP7-003 single-line-only certification, F-SP19-001 joined-line certification gap).

**Found via ledger-19 sanctioned shape:** Pass-19 established that "a block NOT in the table = valid finding even if that block's content is current." Pass-21 applied this directly: the adversary enumerated all binding blocks in the note and compared against the table, finding four absent entries.

#### Finding F-SP21-001

| ID | Severity | Class | Description | Remediation |
|----|----------|-------|-------------|-------------|
| F-SP21-001 | MED | doc-drift/incomplete sweep-completeness certification | Note v1.16 class-closure sweep table certified "17 blocks, complete" but missed four binding-block headers whose text doesn't match the recorded grep patterns: :262 FrameFn byte-contract (v1.3/F-SP3-001, structural peer of enumerated rows 3/4), :511 Test shape, :1812 Pin test shape, :1928 Binding harness rule. All four verified CURRENT — no stale content was hidden. Defect is purely in the certification. 8th incomplete-sweep-class instance, 3rd false completeness certification. | Note v1.16→v1.17 (architect): table extended with rows 18-21 covering the four missed blocks with disposition CURRENT for each; canonical grep pattern `grep -nE '\*\*[^*]*[Bb]inding'` recorded (produced 21 pre-edit hits); v1.17 addendum added preserving full v1.16 text; POST-EDIT META-HIT NOTE added per orchestrator audit: live post-edit count is 68 including documentation echoes (architect independently caught its own newly-added paragraph adding hit #68 — count discipline fully internalized); re-certified over 21 binding blocks, all current. Story v1.19→v1.20 metadata-only: note pin v1.16→v1.17 in frontmatter. STORY-INDEX v4.59→v4.60. |

#### Pass 21 Confirmations

- **v1.16 three-part annotation well-formed:** Pass-20's F-SP20-001 remediation (header marker + retracted-prose strike + sketch banner) verified intact and syntactically correct. HOLDING.
- **9 sweep-table dispositions audited TRUE:** All 9 non-extended rows in the v1.16 table independently verified — block exists at the stated line, disposition label correct (CURRENT/REMEDIATED/PREVIOUSLY-ANNOTATED as recorded). No false entries in the original 17-row set.
- **Story historiography CLEAN under class lens:** All story changelog entries checked for sweep-certification claims — none present (sweep table lives in note only). CLEAN.
- **Task 1-16 implementer dry-run — NO BLOCKING CONTRADICTIONS:** First full Task 1-16 implementer dry-run of this cycle. All 16 tasks traced realizable at 8eb54a5. Task-8 RED-gate-ordering observation (adversary noted that Task-8 mentions RED-gate before some preceding tasks are complete) adjudicated NOT a finding — task ordering is advisory sequencing for the implementer, not a spec contract; RED-gate discipline applies at PR submission, not task-by-task.
- **Notes-chain last-five audit:** Last 5 entries in STORY-INDEX Notes column for S-BL.PE-RECEIVE-LOOP verified accurate and sequentially coherent. CLEAN.
- **POL pass:** POL-001 version pins current; POL-002 STORY-INDEX sync confirmed for note v1.17 and story v1.20.
- **NoDuplicateSuppression + AC-005 lifecycle re-traced realizable:** Two binding contracts traced to 8eb54a5 — both realizable. No new unrealizability gaps.
- **Ledger 1-20 hold:** All 20 prior pass findings verified holding at note v1.17 + story v1.20 state.

#### Observation: Self-Referential Tail — Note Meta-Documentation is the Last Defect Surface

Four consecutive passes (18/19/20/21) have each found exactly one MED, all in note historiography — specifically in the sweep-certification and annotation metadata layers. Story substance has been finding-free since pass 17. The note's sweep table (the artifact that certified the note's correctness) is now the only surface still yielding findings, producing a self-referential tail: the adversary is auditing the certifications of the certifications. Pass-21's extension of the table to 21 blocks + canonical pattern + post-edit meta-hit note represents the most robust certification produced in this cycle.

#### Outcome

- **Streak stays 0/3.** F-SP21-001 is MED — certification-only defect in note historiography.
- Sprint-state v2.25→v2.26. Decay: 7→4→3→2→3→4→5→2→1→2→3→1→1→1→1→0→1→1→1→1→1.

**Awaiting:** spec adversarial pass 22 @ {story v1.20, note v1.17} (streak 0/3)
