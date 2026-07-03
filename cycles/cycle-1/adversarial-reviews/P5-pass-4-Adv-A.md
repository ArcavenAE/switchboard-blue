---
document_type: adversarial-review
artifact_id: P5-pass-4-Adv-A
version: "1.0"
phase: 5
pass: 4
lens: public-surface-operator-ux
adversary_variant: A
verdict: HAS_FINDINGS
finding_high: 3
finding_medium: 5
finding_low: 2
observation_count: unknown
develop_tip: c76a8d5
model: opus
time_spent_minutes: unknown
files_read: unknown
read_cap: 6
prior_passes_read: false
producer: adversary
timestamp: 2026-07-03T00:00:00Z
backfilled: true
backfill_source: cycles/cycle-1/session-checkpoints.md
backfill_date: 2026-07-03
---

# Phase 5 Pass 4 Adv-A Public-Surface Review (BACKFILLED)

> **BACKFILL NOTE:** This report was reconstructed on 2026-07-03 from
> `cycles/cycle-1/session-checkpoints.md`, `STATE.md` (Burst 19 deltas), and
> `sprint-state.yaml` (pass_4_remediation.findings_resolved). The original
> standalone report was never written — findings were tracked directly in the
> remediation burst (Burst 19). Finding titles and IDs are verbatim from the
> checkpoint text; full finding bodies were not recorded in the source and are
> absent here. All 10 A-lens findings were resolved by PR #63 (cbd0272).

**Verdict:** HAS_FINDINGS
**Develop tip at dispatch:** `c76a8d5` (Pass 3 remediation merged — develop head at Pass 4 dispatch)
**Model:** opus
**Time spent:** unknown
**Files read:** unknown / cap 6
**Lens:** public-surface / operator-UX

## Finding Catalog (verbatim from checkpoint source)

### F-A-001 [HIGH]: svtn_id wire field

Source: `sprint-state.yaml` pass_4_remediation.findings_resolved; `STATE.md` Burst 19 deltas "svtn_id wire field".

Wire-contract mismatch: sbctl side was marshaling `json:"svtn"` (stale) while daemon expected `json:"svtn_id"`. Cross-lens finding with F-B-001. Resolved by Burst 19 Phase 2a (implementer daemon-side wire migration) + Phase 2b (spec-steward interface-definitions + taxonomy). Full finding body not recorded in source checkpoint.

### F-A-002 [HIGH]: OpenSSH pubkey parsing

Source: `sprint-state.yaml` pass_4_remediation.findings_resolved; `STATE.md` Burst 19 deltas "OpenSSH pubkey".

Pubkey parsing deficiency: daemon-side `decodePublicKey` did not accept OpenSSH-format public keys, only raw base64. Cross-lens finding with F-B-002. Resolved by Burst 19 Phase 2a. Full finding body not recorded in source checkpoint.

### F-A-003 [MEDIUM]: E-ADM-018 parenthetical

Source: `sprint-state.yaml` pass_4_remediation.findings_resolved; `STATE.md` Burst 19 deltas "E-ADM-018/013/CFG-012 taxonomy drift"; `error-taxonomy v4.4 → v4.5: E-ADM-018 parenthetical removed`.

Taxonomy canonical text drift: E-ADM-018 emission site carried an unauthorized parenthetical not in the canonical taxonomy text. Adv-A only (public-surface observation from operator UX angle). Resolved by Burst 19 Phase 2b (spec-steward taxonomy v4.5). Full finding body not recorded in source checkpoint.

### F-A-004 [MEDIUM]: E-ADM-013 "no key with" prefix

Source: `sprint-state.yaml` pass_4_remediation.findings_resolved; `STATE.md` Burst 19 deltas "E-ADM-013 prefix"; `error-taxonomy v4.4 → v4.5: E-ADM-013 "no key with" prefix added`.

Taxonomy canonical text drift: E-ADM-013 emission lacked the "no key with" prefix required by taxonomy canonical text. Cross-lens finding with F-B-004. Resolved by Burst 19 Phase 2b (taxonomy v4.5). Full finding body not recorded in source checkpoint.

### F-A-005 [LOW]: taxonomy-only adjudication (spec-steward) — item 1

Source: `sprint-state.yaml` pass_4_remediation.findings_resolved: "F-A-005/F-A-006 (taxonomy-only adjudication; spec-steward)".

Taxonomy-scope finding adjudicated as spec-steward responsibility only; no code change required. Specific taxonomy item not recorded in source checkpoint beyond the adjudication label. Resolved by Burst 19 Phase 2b. Full finding body not recorded in source checkpoint.

### F-A-006 [LOW]: taxonomy-only adjudication (spec-steward) — item 2

Source: `sprint-state.yaml` pass_4_remediation.findings_resolved: "F-A-005/F-A-006 (taxonomy-only adjudication; spec-steward)".

Taxonomy-scope finding adjudicated as spec-steward responsibility only; no code change required. Specific taxonomy item not recorded in source checkpoint beyond the adjudication label. Resolved by Burst 19 Phase 2b. Full finding body not recorded in source checkpoint.

### F-A-007 [MEDIUM]: E-INT-999 canonical text

Source: `sprint-state.yaml` pass_4_remediation.findings_resolved; `STATE.md` Burst 19 deltas "E-INT-999".

Taxonomy canonical text drift: E-INT-999 emission site did not match the canonical "unmapped internal condition, programmer error, please report" text required by the taxonomy. Adv-A only. Resolved by Burst 19. Full finding body not recorded in source checkpoint.

### F-A-008 [MEDIUM]: --confirm symmetry boolStringFlag

Source: `sprint-state.yaml` pass_4_remediation.findings_resolved; `STATE.md` Burst 19 deltas "--confirm symmetry"; `cmd/sbctl/admin.go (--confirm symmetry boolStringFlag)`.

Interactive confirm-gate operator UX gap: `--confirm` flag symmetry between sbctl and daemon not enforced via a boolStringFlag type; confirm gate could accept unexpected values. Cross-lens finding with F-B-003. Resolved by Burst 19 Phase 2c (sbctl-side operator-UX fixes). Full finding body not recorded in source checkpoint.

### F-A-009 [HIGH]: interactive prompt short-id substitution

Source: `sprint-state.yaml` pass_4_remediation.findings_resolved; `STATE.md` Burst 19 deltas "interactive prompt short-id substitution"; `DRIFT-P5P4-PROMPT-SHORTID RESOLVED: interactive prompt substitution shipped`.

Operator UX defect: interactive `sbctl admin svtn destroy` confirmation prompt displayed the literal placeholder `<short-id>` rather than the actual SVTN short-id from the operation context. Adv-A only (public-surface / operator UX lens). Resolved by Burst 19 Phase 2c. Full finding body not recorded in source checkpoint.

### F-A-010 [MEDIUM]: E-CFG-012 "pick one"

Source: `sprint-state.yaml` pass_4_remediation.findings_resolved; `STATE.md` Burst 19 deltas "E-CFG-012 canonical"; `error-taxonomy v4.4 → v4.5: E-CFG-012 "pick one" canonical text`.

Taxonomy canonical text drift: E-CFG-012 emission did not match the "pick one" canonical phrasing required by the taxonomy. Cross-lens finding with F-B-003. Resolved by Burst 19 Phase 2b (taxonomy v4.5) + Phase 2c (sbctl). Full finding body not recorded in source checkpoint.

## Resolution

All 10 A-lens findings resolved by Burst 19 — PR #63 `cbd0272` merged 2026-07-03.

Artifacts changed:
- `cmd/switchboard/admin_handlers.go` — svtn_id wire field, pubkey parsing, E-ADM-018/013/INT-999 canonical text
- `cmd/sbctl/admin.go` — E-CFG-012/013 canonical text, interactive prompt short-id, --confirm symmetry boolStringFlag, yes-warning targetFlag parameterization
- `.factory/specs/error-taxonomy.md` v4.4 → v4.5

BC-5.39.001 streak: passes 17/18/19 SATISFIED (3/3 clean).

VERDICT: HAS_FINDINGS
