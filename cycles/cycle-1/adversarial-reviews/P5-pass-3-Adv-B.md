---
document_type: adversarial-review
artifact_id: P5-pass-3-Adv-B
version: "1.0"
phase: 5
pass: 3
lens: test-rigor-traceability-governance
adversary_variant: B
verdict: HAS_FINDINGS
finding_high: 0
finding_medium: 1
finding_low: 2
observation_count: 3
develop_tip: 7fe3e29e4358df16e4e2f1de65a4e0d972540b4a
model: opus
time_spent_minutes: 5
files_read: 6
read_cap: 6
prior_passes_read: false
producer: adversary
timestamp: 2026-07-02T00:00:00Z
---

# Phase 5 Pass 3 Adv-B Test-Rigor / Traceability Review

**Verdict:** HAS_FINDINGS (0H / 1M / 2L / 3O)
**Develop tip:** `7fe3e29e4358df16e4e2f1de65a4e0d972540b4a`
**Model:** opus | **Time spent:** ~5 min | **Files read:** 6/6 | **Lens:** test-rigor / traceability / governance

Files read (against cap):
1. `.factory/specs/verification-properties/VP-INDEX.md`
2. `.factory/specs/verification-properties/VP-043.md`
3. `.factory/specs/behavioral-contracts/ss-02/BC-2.02.007.md`
4. `.factory/specs/verification-properties/VP-062.md` (first 80 lines)
5. `internal/arq/fec_test.go`
6. `.factory/stories/S-7.03-console-remote-control.md` (first 100 lines)

## Findings

### F-P5P3-B-001 [MEDIUM]: VP-043 declares proof_method proptest+gopter, but the implementing test uses a hand-rolled LCG loop — spec/implementation drift

**What.** VP-043 formally declares its proof mechanism as gopter-driven proptest, but the shipped test does not import gopter and does not use any property-based-testing framework; it uses a deterministic MMIX LCG loop that mimics one.

**Where.**
- VP-043 frontmatter `proof_method: proptest` — `.factory/specs/verification-properties/VP-043.md:16`
- VP-043 Proof Method table row: "`| proptest | gopter v0.2.9+ | no | random groups of 4 byte-slice payloads`" — `VP-043.md:54`
- VP-043 harness skeleton imports `github.com/leanovate/gopter` and calls `gopter.NewProperties`, `prop.ForAll`, `gen.SliceOfN(16, gen.UInt8())` — `VP-043.md:63-131`
- Actual implementing test `TestBC_2_02_007_VP043_SingleLossRecovery_Property` — `internal/arq/fec_test.go:573-680`. Imports at `fec_test.go:18-26` are `errors`, `fmt`, `testing`, `time`, `internal/arq`, `internal/frame` only. No `gopter` import anywhere in the file.
- The property loop at `fec_test.go:593-609` implements a manual LCG: `seed = seed*6364136223846793005 + 1442695040888963407` with `randByte := func() byte { return byte(lcgNext() >> 56) }`.

**Why this matters.** Governance drift on the VP catalog's declared verification mechanism.
1. VP-INDEX (row 69) counts VP-043 in the Proptest bucket (34 total). If VP-043 is not a proptest, the count is stale and the "Arithmetic check: 34 + 4 + 22 + 10 + 2 + 2 + 2 = 76" is bucket-mislabelled.
2. Someone auditing "which VPs are gopter-backed" via `grep -r gopter internal/` gets zero hits for the internal/arq module and would conclude VP-043 is unimplemented — a false alarm loop.
3. gopter provides shrinking and reproducible seeds on failure; the hand-rolled LCG covers 35 000 cases but produces no minimal failing example on regression. This is a genuine coverage-quality drop vs the declared mechanism, not just paperwork.
4. VP-043 changelog last touched v1.0→v1.1 for "S-7.01 LENS-3 traceability backfill" (VP-INDEX v2.24). The traceability backfill re-verified the story linkage but did not re-verify the proof mechanism, which had already drifted.

**Suggested remediation.** Pick one:
- (a) Update VP-043 Proof Method to `strong-oracle` or `deterministic-table+lcg`, remove gopter from the harness skeleton, and reclassify VP-INDEX's row 69 bucket (Proptest 34→33 + a new "Table/Deterministic" bucket, or fold into Unit 2→3). Update VP-INDEX changelog to v2.35 recording the reclassification. Fix the arithmetic footer accordingly.
- (b) Migrate `TestBC_2_02_007_VP043_SingleLossRecovery_Property` to use `github.com/leanovate/gopter` per the harness skeleton, preserving the count invariant if desired.

### F-P5P3-B-002 [LOW]: VP-043 frontmatter `source_bc` missing v1.3 version pin — POL-003 gap

**What.** VP-043 frontmatter carries `source_bc: BC-2.02.007` with no version qualifier, but BC-2.02.007 is currently at v1.3 (with substantive PC-5 wire-vocabulary change at v1.2 and traceability change at v1.3).

**Where.**
- VP-043 frontmatter — `.factory/specs/verification-properties/VP-043.md:14`: `source_bc: BC-2.02.007`
- BC-2.02.007 version — `.factory/specs/behavioral-contracts/ss-02/BC-2.02.007.md:5`: `version: "1.3"`
- BC-2.02.007 PC-5 changed frame-type wire encoding v1.1→v1.2 (`FRAME_TYPE=PARITY` retired → `frame_type=fec=0x05`). VP-043 test at `fec_test.go:117-119` asserts `frame.FrameTypeFec != 0x05` — so it is tracking v1.2 semantics but the VP frontmatter doesn't record this pin.

**Why this matters.** POL-003 candidate-pin gap. If BC-2.02.007 changes semantics again (e.g., adjusts group-size default or parity-frame header format), an implementer looking at VP-043 frontmatter has no anchor to know which BC version this VP was verified against. The traceability edge points to the BC file, not the BC version.

**Suggested remediation.** Update VP-043 frontmatter to `source_bc: BC-2.02.007 v1.3`; add VP-INDEX row 69 BC-column annotation "BC-2.02.007 v1.3" mirroring the pattern used for VP-048/VP-062. Include in the same changelog bump.

### F-P5P3-B-003 [LOW]: VP-062 frontmatter `source_bc` missing v1.13 pin that its own VP-INDEX row carries

**What.** VP-INDEX row 88 for VP-062 declares `BC-2.06.003 v1.13` — one of only two VP-INDEX rows with an explicit BC version pin. But VP-062's own frontmatter carries the un-pinned form.

**Where.**
- VP-INDEX row 88: `BC-2.06.003 v1.13` — `.factory/specs/verification-properties/VP-INDEX.md:88`
- VP-062 frontmatter — `.factory/specs/verification-properties/VP-062.md:14`: `source_bc: BC-2.06.003`

**Why this matters.** POL-003 is the one axis on this project where consistent version pinning is being tracked. VP-062 is one of only two VPs the catalog claims are version-pinned. Having the frontmatter fall back to un-pinned undermines the invariant "VP-INDEX BC column is the source of truth" — either both should carry the pin, or the VP-INDEX row should not claim a pin the frontmatter doesn't assert.

**Suggested remediation.** Sync VP-062 frontmatter `source_bc: BC-2.06.003 v1.13`; log as POL-003 catalog↔frontmatter sync in VP-INDEX changelog v2.35.

## Observations (non-blocking)

### O-P5P3-B-001 [positive callout]: FEC test file demonstrates strong-oracle discipline (no tautology)

`internal/arq/fec_test.go` implements `xorOracle()` (lines 50-67) as an INDEPENDENT XOR reference that does not call the SUT. Tests assert `encoder output == oracle output` and `recovered payload == original bytes`. The `TestBC_2_02_007_Encode_ParityXORCorrect` test verifies `enc.AddFrame` output against `xorOracle(payloads)` computed from the raw byte inputs. This is the correct pattern — a mock-the-SUT/self-referential tautology (e.g. `enc.Recover(enc.Encode(p)) == p` with no independent oracle) would be a critical rigor gap; the shipped test avoids it. Worth calling out as a positive baseline for future test-writer prompts.

### O-P5P3-B-002 [governance]: S-7.03 body cites BC-2.07.004 EC-013 in AC-001/002/003 but frontmatter `bc_traces` contains only BC-2.08.001

Story `bc_traces: [BC-2.08.001]` (line 17); ACs 1/2/3 all say "per BC-2.07.004 EC-013". The AC "traces to" markers only name BC-2.08.001, so this is not a strict frontmatter-body coherence violation (BC-2.07.004 is cited as transport context, not as a formal trace). However: if BC-2.07.004 EC-013 changes wire-format semantics (mgmt-plane Unix socket vs TCP loopback), S-7.03's ACs move materially. The story's `changed_by_rulings: [RULING-W6TB-C]` captures the last such change. Recommend a lightweight secondary-trace convention (e.g., `bc_context: [BC-2.07.004]`) so downstream ripple checks fire when EC-013 mutates. Not blocking convergence; noting for process-gap follow-up. `[process-gap]`

### O-P5P3-B-003 [process-gap]: POL-003 conformance is stalled at 2/76 across multiple passes

VP-INDEX changelog v2.32 explicitly records "VP-048 bumped v1.7→v1.8 (F-P4L3-MED-1, POL-003): source_bc pin sync BC-2.07.001 v1.11→v1.12". That's one POL-003-motivated bump. There is no evidence of a systematic sweep. Findings F-P5P3-B-002 and -B-003 above show at least two more candidate VPs (VP-043, VP-062) that would gain from POL-003 pinning. If POL-003 is expected to converge, the review pipeline needs a rule that flags every VP whose source_bc points to a BC modified in the last N passes, not just the ones organically touched. `[process-gap]`

## POL-003 conformance snapshot (required)

**Count:** 2 VP-INDEX rows carry explicit `BC-X.YZ.NNN vN.M` version pins:
- VP-048 (row 74): `BC-2.07.001 v1.12`
- VP-062 (row 88): `BC-2.06.003 v1.13`

**Total active VPs in VP-INDEX:** 76 (per Counts table line 114; two placeholders VP-TBD-ACC and VP-VW6.NN are not counted as active).

**Ratio:** 2 / 76 = **2.6%**.

**Trend note:** Unchanged from the stated baseline (~2/76 prior passes). One POL-003-labelled sync (VP-INDEX v2.32) occurred this cycle, but the two-VP total did not increase — that sync was a version-pin refresh on an already-pinned row (VP-048 v1.11→v1.12), not net new pinning. POL-003 conformance is stalled; per O-P5P3-B-003 a systematic sweep is needed for real progress.
