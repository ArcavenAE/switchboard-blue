---
document_type: adversarial_review_report
artifact_id: P5P6-Adv-B
verdict: CLEAN
finding_counts:
  high: 0
  medium: 0
  low: 0
  observations: 2
develop_tip: d012dbfc92d15cc5f5113f63c79052f00f274861
model: opus-4-7
lens: TEST_RIGOR_TRACEABILITY
perimeter: phase-5-test-tier-admin-surface
time_spent_minutes: 5
files_read:
  - .factory/specs/prd-supplements/interface-definitions.md
  - .factory/specs/prd-supplements/error-taxonomy.md
  - cmd/sbctl/admin_wire_tag_test.go
  - cmd/sbctl/admin_emission_text_test.go
  - cmd/switchboard/admin_handlers_wire_test.go
  - cmd/switchboard/admin_handlers_emission_text_test.go
read_cap: 6
prior_passes_read: false
---

## Summary

Phase 5 Pass 6 fresh-context adversarial review, variant Adv-B (TEST RIGOR + TRACEABILITY lens), of the Phase 5 test tier against interface-definitions.md v1.18 and error-taxonomy.md v4.6.

The Burst 19/21 test suite in cmd/switchboard and cmd/sbctl demonstrates disciplined wire-contract regression coverage:

- **Wire-tag guards** (`admin_handlers_wire_test.go`, `admin_wire_tag_test.go`) cover all four daemon arg structs plus the three top-level sbctl arg structs with both (a) round-trip assertions and (b) marshaled-map presence/absence checks — a stale `json:"svtn"` regression would fail both directions.
- **`StaleField_SvtnRejected`** (admin_handlers_wire_test.go:326-394) implements the recommended two-case discriminating oracle pattern (case A: stale field rejected with E-CFG-001; case B: canonical field accepted), correctly distinguishing "handler always errors" from "stale field silently dropped, correct field accepted."
- **Emission-text guards** use `assertErrorPrefix` (HasPrefix, not Contains) for E-CFG-012 (both call sites), E-CFG-013, E-ADM-018, and E-ADM-013 — enforcing byte-identical canonical emission at the message start.
- **Target-flag symmetry** for Path 4 `--yes` warnings is guarded twice (Destroy=--name, KeyRegister=--svtn), each side asserting both presence of the correct flag and absence of the wrong flag.
- **Confirm-gate coverage** (admin_confirm_symmetry_test.go, admin_interactive_prompt_test.go) matches interface-definitions §125/§129/§130, with self-acknowledged deferrals to DRIFT-P5P4-PROMPT-SHORTID.

No tautologies, name-vs-assertion mismatches, missing regression guards on wire/emission contracts, or historically-impossible spec citations were found within the perimeter and outside the enumerated adjudicated deferrals. The v1.17 spec citations in Burst 19/21 test docstrings receive the same historical-provenance treatment already adjudicated for admin_test.go v1.1 citations (provenance at test-authoring time, not assertion anchors).

POL-001 (changelog-completeness) — satisfied: interface-definitions.md v1.18 changelog note at line 136; error-taxonomy.md v4.6 changelog row at line 230.
POL-002 (story-index-row-sync) — satisfied for S-BL.ADMIN-RECOVER-WIRE (STORY-INDEX row 139, changelog row 183, draft v1.0).

## Findings

None.

## Observations

### OBS-P5P6-B-001 [low] `sbctlSideListKeysArgs` mock carries a field the real sbctl side does not have

**File:** cmd/switchboard/admin_handlers_wire_test.go:49-55
**Spec citation:** interface-definitions.md §116, admin.key.list-keys wire row

The mock struct is defined as:

```go
type sbctlSideListKeysArgs struct {
    SVTNID     string `json:"svtn_id"`
    CallerRole string `json:"caller_role"`
}
```

But the actual sbctl-side type (cmd/sbctl/admin.go:170-172) is a local inline struct inside `runAdmin`:

```go
type listKeysArgs struct {
    SVTNID string `json:"svtn_id"`
}
```

The sbctl inline struct has no `CallerRole` field — sbctl relies on the daemon to resolve caller role from the pubkey in ctx. The mock name `sbctlSideListKeysArgs` implies fidelity to sbctl's wire shape, but instead it models what the daemon `adminListKeysArgs` accepts (which does include `caller_role`). The docstring at lines 49-51 partially acknowledges the mismatch ("sbctl does not currently have a separate list-keys args struct but may pass svtn_id inline"), but the name remains misleading for a future reader.

Not a rediscoverable defect — the adjudicated deferral (sbctl `adminListKeysArgs` local inline type not accessible from test package; daemon-side coverage sufficient) explicitly enumerated this in the dispatch. Flagged as an observation rather than a finding because the misleading naming is a readability concern, not a correctness gap. Suggested phrasing on future refactor: `daemonListKeysArgsPayload` (what the wire looks like arriving at the daemon).

### OBS-P5P6-B-002 [low] [process-gap] Historical-provenance v1.17 spec citations in Burst 19/21 tests parallel the adjudicated admin_test.go pattern

**Files:**
- cmd/switchboard/admin_handlers_wire_test.go:11-12
- cmd/sbctl/admin_wire_tag_test.go:13-14
- cmd/sbctl/admin_wire_tag_test.go:42, 120
- cmd/sbctl/admin_emission_text_test.go:99

Interface-definitions.md is now v1.18 (line 5), incorporating Pass 5 Burst 21 remediation (v1.18 changelog note line 136). The Burst 19/21 test files cite "interface-definitions v1.17 §125" or "v1.17 §129/§130" in their docstring anchors.

Under the adjudication logic already applied to admin_test.go v1.1 provenance citations at lines ~1642/1834/1855/2433/2477/2522 (explicitly deferred by the dispatch), these v1.17 citations are provenance comments recording the spec version current at test-authoring time (Burst 19 authored against v1.17; Burst 21 also referenced v1.17 §125 as the wire-contract anchor before v1.18 was minted). The assertions themselves target the canonical emissions in taxonomy v4.6 (which is current) and BC-2.05.004 v1.12 / BC-2.07.001 v1.13 (both current in the behavioral-contracts index).

Flagged as an observation for parity — the same adjudication should extend consistently to these newer test files. If the sprint later decides to normalize provenance citations to "the earliest spec version at which the anchored clause reached its current form," these files would be included in that sweep alongside admin_test.go.

## Adjudicated deferrals respected

The following were verified as-still-deferred and were NOT re-reported as findings:

- svtn.list / svtn.version / svtn.ping / sessions.list wire handlers (backlog; S-BL.DISCOVERY-WIRE pending per task #79)
- sbctl admin recover PENDING (annotated in interface-definitions §119-123 per F-P5P5-A-002)
- internal/adminwire package extraction (acknowledged in admin_handlers_wire_shared_pkg_test.go docstring)
- historical-provenance v1.1 citations at admin_test.go lines ~1642/1834/1855/2433/2477/2522
- `tc := tc` capture lines obsoleted under Go 1.25 semantics
- sbctl `listKeysArgs` inline local type — no direct wire-tag guard on sbctl side; daemon-side round-trip is the guard (per admin_wire_tag_test.go:16-23)

VERDICT: CLEAN
