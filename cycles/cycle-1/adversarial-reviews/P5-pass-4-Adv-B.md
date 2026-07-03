---
document_type: adversarial-review
artifact_id: P5-pass-4-Adv-B
version: "1.0"
phase: 5
pass: 4
lens: test-rigor-traceability
adversary_variant: B
verdict: HAS_FINDINGS
finding_high: 2
finding_medium: 2
finding_low: 0
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

# Phase 5 Pass 4 Adv-B Test-Rigor / Traceability Review (BACKFILLED)

> **BACKFILL NOTE:** This report was reconstructed on 2026-07-03 from
> `cycles/cycle-1/session-checkpoints.md`, `STATE.md` (Burst 19 deltas), and
> `sprint-state.yaml` (pass_4_remediation.findings_resolved). The original
> standalone report was never written — findings were tracked directly in the
> remediation burst (Burst 19). Finding titles and IDs are verbatim from the
> checkpoint text; full finding bodies were not recorded in the source and are
> absent here. All 4 B-lens findings were resolved by PR #63 (cbd0272).

**Verdict:** HAS_FINDINGS
**Develop tip at dispatch:** `c76a8d5` (Pass 3 remediation merged — develop head at Pass 4 dispatch)
**Model:** opus
**Time spent:** unknown
**Files read:** unknown / cap 6
**Lens:** test-rigor / traceability

## Finding Catalog (verbatim from checkpoint source)

### F-B-001 [HIGH]: svtn_id wire field (cross-lens from test perspective)

Source: `sprint-state.yaml` pass_4_remediation.findings_resolved: "F-A-001/F-B-001 (svtn_id wire field)"; `STATE.md` Burst 19 deltas "svtn_id wire field".

Wire-contract traceability finding: from the test-rigor lens, no test verified the sbctl↔daemon wire contract for `svtn_id`. Tests would have passed even with the stale `json:"svtn"` field because the contract was untested at the wire level. Cross-lens finding with F-A-001. Resolved by Burst 19 Phase 2a (implementer wire migration) + Phase 2b (spec-steward interface-definitions + taxonomy). Full finding body not recorded in source checkpoint.

### F-B-002 [HIGH]: OpenSSH pubkey parsing (cross-lens from test perspective)

Source: `sprint-state.yaml` pass_4_remediation.findings_resolved: "F-A-002/F-B-002 (OpenSSH pubkey parsing)"; `STATE.md` Burst 19 deltas "OpenSSH pubkey".

Test-coverage gap: no test exercised the OpenSSH public key format through `decodePublicKey`. The daemon-side implementation only accepted raw base64, and this regression surface was undetected by the test suite. Cross-lens finding with F-A-002. Resolved by Burst 19 Phase 2a. Full finding body not recorded in source checkpoint.

### F-B-003 [MEDIUM]: --confirm symmetry + E-CFG-012 (cross-lens from test perspective)

Source: `sprint-state.yaml` pass_4_remediation.findings_resolved: "F-A-008/F-B-003 (--confirm symmetry boolStringFlag)" and "F-A-010/F-B-003 (E-CFG-012 'pick one')"; `STATE.md` Burst 19 deltas "--confirm symmetry"; `cmd/sbctl/admin.go (--confirm symmetry boolStringFlag)`.

Test-coverage and traceability gap: the confirm-gate behavior and E-CFG-012 canonical text were unverified by tests. The `--confirm` flag symmetry contract between sbctl and daemon was not exercised, and the E-CFG-012 emission text was not validated against the taxonomy canonical form. Cross-lens finding with F-A-008 (confirm symmetry) and F-A-010 (E-CFG-012). Resolved by Burst 19 Phase 2b (taxonomy v4.5) + Phase 2c (sbctl confirm gate). Full finding body not recorded in source checkpoint.

### F-B-004 [MEDIUM]: E-ADM-013 "no key with" prefix (cross-lens from test perspective)

Source: `sprint-state.yaml` pass_4_remediation.findings_resolved: "F-A-004/F-B-004 (E-ADM-013 'no key with' prefix)"; `STATE.md` Burst 19 deltas "E-ADM-013 prefix"; `error-taxonomy v4.4 → v4.5: E-ADM-013 "no key with" prefix added`.

Traceability gap: the E-ADM-013 emission was not verified against the taxonomy canonical text, and the "no key with" prefix requirement was undetected by the test suite. Cross-lens finding with F-A-004. Resolved by Burst 19 Phase 2b (taxonomy v4.5). Full finding body not recorded in source checkpoint.

## Resolution

All 4 B-lens findings resolved by Burst 19 — PR #63 `cbd0272` merged 2026-07-03.

Artifacts changed:
- `cmd/switchboard/admin_handlers.go` — svtn_id wire field, pubkey parsing, E-ADM-013 canonical text
- `cmd/sbctl/admin.go` — E-CFG-012 canonical text, --confirm symmetry boolStringFlag
- `.factory/specs/error-taxonomy.md` v4.4 → v4.5

BC-5.39.001 streak: passes 17/18/19 SATISFIED (3/3 clean).

VERDICT: HAS_FINDINGS
