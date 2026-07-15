---
artifact_id: S-M.03-sweep-methodology-hardening
document_type: story
level: ops
story_id: S-M.03
version: "1.0"
title: "harden spec-artifact citation-pin sweep methodology (canonical pattern library + semantic-claim verification)"
status: draft
producer: story-writer
timestamp: 2026-07-14T00:00:00Z
modified: 2026-07-14T00:00:00Z
phase: 2
epic: E-MAINT
wave: backlog
priority: P2
scope_phase: M
estimated_points: TBD
tdd_mode: facade
behavioral_contracts: []
verification_properties: []
depends_on: []
blocks: []
subsystems: []
architecture_modules: []
cycle: v1.0.0-greenfield
acceptance_criteria_count: 0
inputDocuments:
  - '.factory/cycles/cycle-1/S-BL.DISCOVERY-WIRE/adversary-convergence-state.json'
  - '.factory/stories/S-BL.DISCOVERY-WIRE.md'
provenance:
  origin: "process-gap PG-DWSP6-01 (S-BL.DISCOVERY-WIRE spec-adversarial convergence) — story-ready human gate disposition, 2026-07-14"
  disposition: "follow-up story targeting the self-improvement epic (S-7.02 cycle-closing checklist disposition (a); status draft satisfies checklist step 3's minimum bar)"
---

# S-M.03: Harden Spec-Artifact Citation-Pin Sweep Methodology

> **Note:** This is a maintenance/self-improvement story with no product behavioral
> contract anchor. Execute outside of feature waves — `E-MAINT` scope, per the
> `S-M.01`/`S-M.02` convention. **STATUS: DRAFT BACKLOG STUB.** Candidate deliverables
> below are sketched, not committed; full decomposition (ACs, tasks, file list) is
> deferred to scheduling time.

## Narrative

- **As an** orchestrator running spec-adversarial convergence cycles
- **I want** the mandatory citation-pin re-certification sweep to be reliably complete
  on the first pass, and sanctioned-exception classifications to require semantic
  verification, not just pattern presence
- **So that** the same sweep-completeness process-gap stops recurring across stories —
  it fired four distinct ways within a single convergence cycle
  (`S-BL.DISCOVERY-WIRE`) after already appearing once before that cycle (the DRAIN
  story's F-SP19-001)

## Context

`S-BL.DISCOVERY-WIRE`'s spec-adversarial convergence (16 passes, 2026-07-13..2026-07-14)
surfaced process-gap **PG-DWSP6-01** — a version-pin/citation sweep-completeness gap
that recurred in four distinct sub-classes within one convergence cycle:

1. **Single-line-regex baseline gap** — the original sweep pattern matched only
   same-line ID/version pairs (the `F-DWSP1-002`/`F-DWSP5-001` lineage; this sub-class
   predates F-DWSP6-001 and is the un-named baseline every later sub-class refined
   against).
2. **Line-wrap sub-class (F-DWSP6-001, MED)** — an ID/version pair split across a
   markdown line wrap (`VP-080` at the end of one line, `v1.3` at the start of the
   next) is invisible to single-line matching. Remediated by a multiline-tolerant Perl
   `-0777` whole-file-buffer sweep — a countermeasure the DRAIN story's F-SP19-001 had
   already ratified elsewhere in the cycle; the lesson did not propagate cross-story.
3. **Paren-form sub-class (F-DWSP11-001, LOW)** — `rulings (v1.3)` (paren-separated) is
   invisible to a suffix-only pattern requiring the version token immediately after the
   name. Remediated by extending the sweep with a paren-tolerant pattern.
4. **Sanctioned-exception semantic-verification gap (F-DWSP13-001, LOW)** — the
   sweep's "known historical citation, verified unchanged burst after burst" exception
   list protected a citation's *version number* without checking its *semantic claim*.
   Two spots were classified as "sanctioned point-in-time historical `rulings v1.6`
   citations" across v2.7-v2.10 and left unedited on every pass — but the claim they
   made (attributing the F-DWSP4-001 restart-liveness amendment to v1.6) was actually
   false; the canonical adoption version is v1.5. Pattern presence in an exception list
   is not truth verification.

The S-7.02 cycle-closing checklist's disposition options for a process-gap finding are
(a) follow-up story in STORY-INDEX, (b) justified deferral in the drift table, or (c)
upstream filing. This story is disposition (a): a follow-up story targeting the
self-improvement epic (`E-MAINT`), satisfying the checklist's step-3 minimum bar (draft
story exists) without committing to a decomposition yet.

Full narrative: `cycles/cycle-1/S-BL.DISCOVERY-WIRE/adversary-convergence-state.json`
(finding records for F-DWSP6-001, F-DWSP11-001, F-DWSP13-001, and the
`convergence_summary`/`next` fields naming PG-DWSP6-01); the v2.5-v2.10 changelog rows
in `S-BL.DISCOVERY-WIRE.md` (the sweep-certification lineage — each row documents the
sweep pattern set in force at that point and what it missed).

## Candidate Deliverables (for elaboration, not commitment)

Sketched only — architect/story-writer elaboration at scheduling time decides shape,
sequencing, and which subset ships:

1. **Canonical pin-pattern library** — a single, maintained set of citation-pin sweep
   patterns (multiline-tolerant, paren-form-tolerant, comment-form-tolerant) that every
   story's mandatory re-certification sweep uses, rather than each story accumulating
   its own ad-hoc pattern set burst-by-burst (as `S-BL.DISCOVERY-WIRE`'s v2.5→v2.9
   changelog rows show happening in place).
2. **Semantic-claim verification step for sanctioned-exception lists** — before a
   citation is added to a "verified unchanged, historical, do-not-touch" exception
   list, require a one-line statement of the semantic claim being protected and
   confirmation that claim is still true — not just that the version number matches
   its point-in-time-correct form. F-DWSP13-001 demonstrates a pattern-correct
   exception can still protect a false claim.
3. **Possible validate-hook** — a hook or lint step that runs the canonical pattern
   library automatically at spec-adversarial pass time, rather than relying on each
   pass's ad-hoc invocation.

## Open Design Obligations (must be resolved before scheduling)

### 1. Where does the canonical pattern library live?

Options include a shared script/module under `.factory/` tooling, a documented
procedure in a rules file, or an upstream `drbothen/vsdd-factory` engine feature (this
project has precedent for filing engine-methodology gaps upstream — see
`F-DW-IMPL-001` → `drbothen/vsdd-factory#620`, `F-DW-DV-001` → `#622`). Not adjudicated
here; needs an architect/orchestrator decision on local-remediation vs. upstream-filing
vs. both.

### 2. Semantic-claim verification — how much ceremony?

Candidate deliverable 2 trades sweep speed for correctness. A verification step that
requires re-reading and re-confirming every sanctioned exception on every pass could
itself become ceremony that gets skipped under time pressure — the same failure mode
this fleet's `tooling-friction.md`/`charter-light-touch.md`-style rules guard against
elsewhere. Needs elaboration on scope: every exception on every pass, or only at
exception-creation time plus periodic audit.

### 3. Validate-hook scope and false-positive risk

A validate-hook needs to distinguish live-prose citations from history-layer citations
(changelog rows, frontmatter `modified:` entries, Provenance blockquotes) — the exact
classification judgment human/adversary passes currently make manually. An automated
hook risks either false positives (flagging correct historical citations) or false
negatives (missing the next sub-class). Not adjudicated here.

## Provenance

- **Origin:** process-gap **PG-DWSP6-01**, recorded in
  `cycles/cycle-1/S-BL.DISCOVERY-WIRE/adversary-convergence-state.json` across four
  sub-instances (baseline single-line gap, F-DWSP6-001 line-wrap, F-DWSP11-001
  paren-form, F-DWSP13-001 sanctioned-exception semantic-verification gap) and the
  v2.5-v2.10 sweep-certification lineage in `S-BL.DISCOVERY-WIRE.md`'s Changelog.
- **Disposition:** story-ready human gate for `S-BL.DISCOVERY-WIRE`, 2026-07-14 —
  follow-up story targeting the self-improvement epic (`E-MAINT`), per the S-7.02
  cycle-closing checklist's disposition (a) (follow-up story in STORY-INDEX). Status
  `draft` satisfies the checklist's step-3 minimum bar (draft story exists targeting
  the self-improvement epic); full decomposition is explicitly deferred to scheduling
  time.
- **Epic/family convention:** follows `S-M.01`/`S-M.02`'s established
  maintenance/self-improvement story family (`epic: E-MAINT`, `S-M.NN` ID scheme,
  introduced 2026-06-27 per `STORY-INDEX.md`'s Maintenance Stories section) — unlike
  `S-M.01`/`S-M.02`, this stub is deliberately thin (0 ACs) rather than fully
  elaborated, matching the backlog-stub convention (`S-BL.DISCOVERY-WIRE` v1.0/v1.1
  shape) rather than a scheduled-and-ready maintenance story.
- **Status:** stays `draft`, `wave: backlog` — no elaboration has occurred yet; this
  stub exists to satisfy the S-7.02 disposition-(a) minimum bar and to give
  PG-DWSP6-01 a concrete anchor other than an open drift row.

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-07-14 | Backlog stub created per S-BL.DISCOVERY-WIRE's story-ready human gate disposition for process-gap PG-DWSP6-01 (S-7.02 cycle-closing checklist disposition (a) — follow-up story targeting the self-improvement epic). Three candidate deliverables sketched (canonical pin-pattern library, semantic-claim verification step, possible validate-hook) per the four PG-DWSP6-01 sub-instances (baseline single-line gap, F-DWSP6-001 line-wrap, F-DWSP11-001 paren-form, F-DWSP13-001 exception-list semantic-verification gap). No ACs decomposed; full elaboration deferred to scheduling time. |
