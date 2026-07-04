---
pass_id: P5-pass-27-Adv-A
lane: A
phase: 5
cycle: cycle-1
timestamp: 2026-07-03T00:00:00Z
worktree_identity_tuple:
  factory_state_step: phase-5-pass-26-concluded-has-findings
  develop_tip_sha: 6deda15def9326f28e96f133e237aff5ecb74d7b
  factory_head_sha: 1f2f557
  go_module: github.com/arcavenae/switchboard
  readme_title: Switchboard
verdict: HAS_FINDINGS
findings_count: 4
critical: 0
high: 1
medium: 2
low: 1
observations: 0
findings: [F-P5P27-A-001, F-P5P27-A-002, F-P5P27-A-003, F-P5P27-A-004]
reconstructed_from_orchestrator_adjudication: true
---

# Phase 5 Pass 27 — Adversary A Review

**Lens:** Spec-completeness + traceability (POL-002 sibling propagation / systemic staleness)
**Perimeter:** `.factory/` artifacts only — no source code reads
**Anti-findings checked:** Pass-26 adjudicated deferrals (F-P5P26-A-001/002 SHIPPED c10d6ba Burst 68a;
F-P5P26-B-001/002 SHIPPED cf135b9 Burst 68b)

---

## F-P5P27-A-001 — HIGH — POL-002 — Wave-5 stories_merged + merge_shas Missing S-W5.02

**Finding class:** POL-002 sibling-sweep gap — wave completion record incomplete

**Description:** `.factory/stories/sprint-state.yaml` Wave-5 block `stories_merged` list contains
7 entries `[S-5.03, S-6.03, S-W5.01, S-5.01, S-6.02, S-6.06, S-5.02]` but Wave-5 consists of
8 stories per STORY-INDEX v3.79 wave summary "8/8 merged". S-W5.02 (e2e management plane harness,
PR #38, SHA d881f99) is absent from `stories_merged` and absent from `merge_shas`. The story
entry at sprint-state.yaml S-W5.02 block has `status: merged` confirming it is complete.

STORY-INDEX v3.79 L69 authoritative: `S-W5.02 | ... | merged (PR #38, d881f99)`.
STORY-INDEX v3.79 L91 wave summary: Wave 5 "8/8 merged".

**Blast radius:** Wave-5 completion record in sprint-state.yaml (2 fields: stories_merged list +
merge_shas entry). Severity HIGH per omission of a fully-merged story from the wave completion
record.

**Remediation:** Append `S-W5.02` to `stories_merged` list and add `S-W5.02: d881f99  # PR #38`
to `merge_shas`. Bump version comment 1.52 → 1.53.

**Status:** SHIPPED — Burst 71a (sprint-state v1.52 → v1.53)

---

## F-P5P27-A-002 — MEDIUM — POL-002 — STATE.md Current State Stale Pass-12 Narrative

**Finding class:** POL-002 sibling-sweep gap — STATE.md Current State prose stale by ~15 passes

**Description:** `STATE.md` §"Current State" prose paragraph reads:

> "Phase 5 Pass 12 split-adversary COMPLETE (Burst 34) + Pass 12 remediation COMPLETE..."
> "develop HEAD: 66e9ddc (unchanged — spec-only burst, no code merged). 45 BCs, 76 VPs,
>  53 stories (backlog +1 S-BL.CLI-SURFACE-COMPLETION), 18 internal packages."

This is Pass-12-era text. The authoritative frontmatter values are:
- `develop_head: 6deda15` (not 66e9ddc)
- `l4_vp_count: 77` (not 76 VPs)
- `total_stories: 54` (not 53)
- Current pass: 26 concluded (not 12)

The stale prose contradicts the correct frontmatter on the same file, creating a consistency
violation for any reader relying on the prose summary.

**Blast radius:** 1 artifact (STATE.md Current State section), cross-field contradiction.

**Remediation:** Replace the stale Pass-12 narrative with a fresh short paragraph reflecting
authoritative frontmatter values and current pipeline status.

**Status:** SHIPPED — Burst 71a (STATE.md Current State replaced)

---

## F-P5P27-A-003 — MEDIUM — POL-002 — Wave-5 wave_adversary_converged False Contradicts Siblings

**Finding class:** POL-002 cross-artifact consistency — within-document field contradiction

**Description:** `.factory/stories/sprint-state.yaml` Wave-5 block at L104 reads:

```yaml
wave_adversary_converged: false
```

This contradicts two sibling fields on the same Wave-5 block:
- `gate_disposition: CONVERGED` (L86)

And contradicts STATE.md frontmatter:
- `wave_5_gate: CONVERGED` (L26)

Wave 5 passed its adversarial gate with `gate_disposition: CONVERGED`. The `wave_adversary_converged`
boolean field must be `true` to be internally consistent with the gate verdict it belongs to.

Note: This is distinct from F-P5P26-A-002, which addressed `gate_disposition: pending` in the
sprint-state (that was fixed in Burst 68a). The `wave_adversary_converged: false` field survived
that remediation.

**Blast radius:** 1 field on 1 artifact. Severity MEDIUM for internal contradiction within the
same Wave-5 block.

**Remediation:** Flip `wave_adversary_converged: false` → `wave_adversary_converged: true`.

**Status:** SHIPPED — Burst 71a (sprint-state v1.53)

---

## F-P5P27-A-004 — LOW — Pending Intent — Wave-6 `tranche-c-closed` vs Wave-5 `closed` Terminology

**Finding class:** LOW — terminology asymmetry / pending-intent signal

**Description:** Wave-5 `status: closed` (standard lifecycle terminal state) while Wave-6
`status: tranche-c-closed` (tranche-lifecycle terminal state). An adversary reading only the
sprint-state could infer inconsistency: both waves are complete, but their terminal status
values differ.

**Adjudication:** BY-DESIGN — NO CHANGE.

Wave-6 was executed in three tranches (Tranche A: 2026-07-01, Tranche B: 2026-07-01,
Tranche C: 2026-07-02) with explicit gate checks at each tranche boundary per
wave-6-scope-decision.md and the CONVERGED_3_OF_3 gate disposition. The `tranche-c-closed`
status reflects this three-phase lifecycle and provides a richer audit trail than a bare
`closed`. Wave-5 was not tranched — it closed as a single gate event, so `closed` is the
correct and complete terminal state.

The terminology difference is load-bearing historical signal, not inconsistency. No change
warranted.

**Status:** ADJUDICATED BY-DESIGN — no remediation.
