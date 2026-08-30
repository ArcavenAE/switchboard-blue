# Improvement Proposals — loopback-v1.21 (2026-08-30)

Status: **PROPOSED**, awaiting human review — non-blocking 72h auto-defer (same convention as improvement-proposals-2026-08-28.md's IP-C2 set, which remains separately pending). 4 proposals, evidence-backed against artifacts read this sub-cycle review (see review-2026-08-30-loopback-v1.21.md).

**IP-C3-01** | category: agent/workflow | LOW | status: PROPOSED
**Title:** POL-005 dispatch tuples must copy source paths verbatim from story frontmatter — never paraphrase
**Evidence:** R37 Lens B tuple named source paths `internal/transport/{halfchannel,multipath,arq}.go`; real paths are `internal/{halfchannel,multipath,arq}/`. The story's `architecture_modules` frontmatter carries the correct paths. Lens B re-derived against real files; verdict unaffected (near-miss). First instance (rereview-R37-2026-08-29.md).
**Proposed change:** Add to the orchestrator POL-005 dispatch-construction step: source paths MUST be copied verbatim from the target story's `architecture_modules:` frontmatter, never reconstructed from memory.
**Routes to:** LOCAL (orchestrator dispatch-checklist). No engine filing — authoring habit, not a factory defect.

**IP-C3-02** | category: agent/template | MEDIUM | status: PROPOSED
**Title:** STORY-INDEX frontmatter version lagged its own changelog row (POL-001) — add a mechanical self-check
**Evidence:** Commit 06d5fe9 added a v4.175 changelog row + ref-column content but left frontmatter version at "4.174". Fixed by 8018ab5 before R38; R38 Lens A re-verified it held. New granularity of the propagation-gap family (prior: ref-column mega-cell vs external sources; this: intra-document frontmatter vs own changelog).
**Proposed change:** Whichever step bumps STORY-INDEX runs one assertion after any changelog-row-add: frontmatter version == newest changelog row version — before the burst completes. Generalizes the R22-LENSA-PROCESS-GAP recommendation to the frontmatter/changelog pair specifically, cheaply and immediately.
**Routes to:** LOCAL. STORY-INDEX maintenance convention.

**IP-C3-03** | category: convergence/pattern | LOW | status: PROPOSED
**Title:** Codify "prefer story-only anchoring over a note edit when architecturally sufficient" as a named guideline
**Evidence:** During the v1.21 anchor burst an architect note-edit to v1.17 was drafted then reverted, because the note is a declared input (a note change forces an input-hash recompute) and to keep v1.19 (BC-2.01.003) and v1.21 (BC-2.02.003) anchors procedurally symmetric. By contrast R33 legitimately bumped the note (v1.15→v1.16, a real TLPKTDROP fix) and accepted the recompute 4902d5d→7967a2f. Discipline: not "never touch the note," but "don't touch it when a story-only anchor is sufficient."
**Proposed change:** Make the discipline an explicit named non-mandatory guideline in the Step-4.5/§1.7 playbook so future sessions default to asking "is a note edit strictly necessary, or can it be anchored story-only per the GAP-4 precedent?" before touching a declared input.
**Routes to:** LOCAL — methodology guidance. Behavior already correct; this reduces drift risk.

**IP-C3-04** | category: pattern/quality | LOW (optional/research) | status: PROPOSED
**Title:** Record "safe-by-construction-but-textually-incomplete contract clause" as a named survivor class; consider a contract-completeness heuristic for future §1.7-class checks
**Evidence:** F-R40-LENSB-01 — loopbackSink.SendInput passes payload to downstreamHC.Enqueue (no copy, halfchannel.go:139-141), on its face contravening the KeystrokeSink "must not retain" clause (upstream.go:65-68). Adjudicated NONE because the upstream path hands a fresh per-tick copy (toMPFrame, story L1210); safety by ownership transfer. The story's Q4 safety prose (L413-416) addresses only the no-callback-under-lock clause, silent on the retain clause. Human reviewed and accepted as-is, declining the optional hardening.
**Proposed change (optional, not a mandate):** When a spec's safety argument invokes a multi-clause interface contract, a future §1.7-class check could enumerate every clause of the cited contract and confirm the safety prose addresses each. F-R40-LENSB-01 found the second clause by chance; a heuristic would make it systematic.
**Routes to:** LOCAL/ENGINE-adjacent (possible future §1.7 sub-check). Low priority per its research framing; human already exercised discretion on the motivating instance.

## Carried forward, not re-proposed here
- PAT-06 / sidecar-marker non-synthesis: 809 markers (up from 599, 165) — IP-C2-01 remains the open fix, still pending.
- IP-C2-01 through IP-C2-09 (2026-08-28): unaffected, still PENDING per STATE.md.
- cycles/cycle-1/convergence-trajectory.md template-compliance gap: pre-existing non-blocking drift (fails validate-template-compliance for sections it never had); no action proposed.
