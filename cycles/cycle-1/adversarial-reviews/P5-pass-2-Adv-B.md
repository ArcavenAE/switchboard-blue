---
document_type: adversarial-review
artifact_id: P5-pass-2-Adv-B
version: "1.0"
phase: 5
pass: 2
lens: test-rigor-traceability-governance
adversary_variant: B
verdict: HAS_FINDINGS
finding_high: 0
finding_medium: 1
finding_low: 1
observation_count: 4
develop_tip: 7fe3e29e4358df16e4e2f1de65a4e0d972540b4a
model: opus
time_spent_minutes: 5
files_read: 5
read_cap: 6
prior_passes_read: false
producer: adversary
timestamp: 2026-07-02T00:00:00Z
---

# Phase 5 Pass 2 Adv-B Test-Rigor / Traceability Review

**Verdict:** HAS_FINDINGS
**Develop tip:** 7fe3e29e4358df16e4e2f1de65a4e0d972540b4a
**Model:** opus
**Time spent:** ~5 minutes
**Files read:** 5 / 6
**Scope lens:** test-rigor / traceability / governance

## Findings

### F-P5P2-B-001 [MEDIUM]: POL-003 version-pin conformance at 2/76 in VP-INDEX; substantive frontmatter pins likely lower

- **What:** Of 76 active VPs in `VP-INDEX.md`, only VP-048 (`BC-2.07.001 v1.12`) and VP-062 (`BC-2.06.003 v1.13`) carry an explicit `source_bc` version pin in the BC(s) column. The remaining 74 VPs cite the BC ID bare (e.g., VP-043 → `BC-2.02.007`, VP-044 → `BC-2.03.001, BC-2.03.003`). Sampled VP frontmatter (`VP-043.md` line 14) confirms bare pins in the substantive artifact, not just the index.
- **Where:**
  - `.factory/specs/verification-properties/VP-INDEX.md:27-102` (index rows; 2 with pins, 74 bare)
  - `.factory/specs/verification-properties/VP-043.md:14` (`source_bc: BC-2.02.007` — no version)
- **Why it's a finding:** POL-003 exists to prevent silent BC-drift undermining a VP's proof-basis. Without version pins, a BC postcondition can change and the VP's proof harness no longer verifies what it claims to. The changelog (v2.32, v2.20) shows POL-003 sync work is happening opportunistically (F-P4L3-MED-1 for VP-048) rather than systematically. At 2.6% conformance the invariant is not durably held; every VP without a pin is a latent drift bug.
- **Suggested remediation:** Batch-mint version pins across all 74 remaining VPs to match their current BC head versions, or narrow POL-003 scope to a defined subset (P0-only? BCs above some churn threshold?) and document the scoping decision.

### F-P5P2-B-002 [LOW]: BC-2.09.003 lists a "no current owner story" flagged deferral without a PENDING-S-* marker in DEFERRED-APPLICATION table

- **What:** `BC-2.09.003.md:184` DEFERRED-APPLICATION table lists `listen_addr` with "No current owner story — a network-listener introduction story is needed (flagged for STORY-INDEX)". Unlike the drain_timeout/upstream_routers/keepalive_interval rows (all owned by S-7.04), the listen_addr row has no owning story and no PENDING-S-* annotation.
- **Where:** `.factory/specs/behavioral-contracts/ss-09/BC-2.09.003.md:184`
- **Why it's a finding:** Contract says "flagged for STORY-INDEX" but there is no cross-check that STORY-INDEX actually carries the flag. Compared with the tracked-deferral pattern of the other three DEFERRED-APPLICATION rows, this row is one bookkeeping step short. If STORY-INDEX lookup surfaces no such flag entry, the gap is untracked in practice.
- **Suggested remediation:** Either (a) file a placeholder story (S-BL.LISTENER or equivalent) and record the ID in this row, or (b) add a PENDING-S-* / DRIFT-* marker recording where in STORY-INDEX the flag lives.

## Observations (non-blocking)

### O-P5P2-B-001: `internal/arq/fec_test.go` is strong — no tautologies detected

- Independent XOR oracle (`xorOracle`) computed separately from the SUT (`encodeGroup` returning encoder output) rules out mock-the-SUT tautologies.
- VP-043 property test claims a deterministic assertion count (35,000 recoveries + 7,000 parity checks) and *asserts on the count itself* in a `count_verify` subtest — a rare and effective belt-and-braces against silent test skipping (`fec_test.go:663-674`).
- Table-driven cases across group sizes 2/4/8 and payload widths 1/3/8/16/32; every loss position exhaustively enumerated. Byte-exact assertions throughout.

### O-P5P2-B-002: BC/VP/test trace for BC-2.02.007 (FEC) is fully live

BC-2.02.007 → VP-043 (implementing_story: S-7.01, added v1.1 2026-07-01) → `internal/arq/fec_test.go` `TestBC_2_02_007_VP043_SingleLossRecovery_Property` — every hop has an explicit cross-reference; test names encode BC ID directly (`TestBC_2_02_007_*`) with story-level aliases (`TestFEC_*`) preserving discoverability. No orphan trace edges.

### O-P5P2-B-003: [process-gap] E-CFG error-code collisions carried across 2 versions

BC-2.09.003 flags E-CFG-002 (private key export vs listen_addr host:port) and E-CFG-006 (sbctl admin flag conflict vs drain_timeout negative) as "pre-existing inconsistencies deferred to maintenance pass" in v1.6 and v1.7 changelogs. Two consecutive minor bumps acknowledged the collision without scheduling a maintenance pass. This is durable governance drift — the flag itself became the artifact. Neither is HIGH because both collisions are between assertion-side codes with distinct calling contexts, but a maintenance-pass story should be filed.

### O-P5P2-B-004: Changelog governance is exemplary

VP-INDEX changelog (lines 148-188) cites specific finding IDs (F-P4L3-MED-002, F-P5L3-LOW-1, F-P22L3-003, RULING-W6TB-J) for essentially every version bump. This is the opposite of governance drift — worth calling out as a positive.

## POL-003 conformance snapshot (required)

- VPs sampled: 76 (full VP-INDEX row scan + 1 frontmatter spot-check on VP-043)
- VPs with `source_bc:` version pin: 2 (VP-048 → `BC-2.07.001 v1.12`; VP-062 → `BC-2.06.003 v1.13`)
- Estimated total VP count: 76 active + 2 placeholder (VP-TBD-ACC, VP-VW6.NN)
- Ratio: 2 / 76 ≈ 2.6% in VP-INDEX; frontmatter spot-check (VP-043) suggests underlying VP files match the bare index representation. Baseline unknown; if the baseline was 0/N this is a marginal improvement, if the target is 76/76 this is a significant gap (see F-P5P2-B-001).
