---
document_type: adversarial-review
artifact_id: P5-pass-7-Adv-B
verdict: CLEAN
finding_counts:
  high: 0
  med: 0
  low: 0
  obs: 5
develop_tip: 4d7d9e0a702228b6dca02970cb4c6290b32311be
model: claude-opus-4-7
time_spent_minutes: 5
lens: test-rigor-and-traceability
files_read:
  - cmd/sbctl/production_exit_code_test.go
  - cmd/sbctl/main.go
  - cmd/sbctl/admin_confirm_symmetry_test.go
  - cmd/sbctl/admin_emission_text_test.go
  - cmd/sbctl/admin_wire_tag_test.go
  - cmd/sbctl/admin.go
files_read_partial:
  - cmd/sbctl/main_test.go (lines 1-100, 260-340, 440-540)
  - cmd/sbctl/admin_test.go (lines 2320-2515)
  - cmd/sbctl/admin_interactive_prompt_test.go
read_cap: 6
prior_passes_read: false
adjudicated_deferrals_honored: true
---

# Phase 5 Pass 7 Adv-B — Test Rigor + Traceability Review

## Preflight

- Worktree develop tip: `4d7d9e0a702228b6dca02970cb4c6290b32311be` — matches dispatch tuple. No `ABORT_STALE_CHECKOUT`.
- Perimeter respected: Phase 5 test tier for `cmd/sbctl/` + `cmd/switchboard/` vs interface-definitions.md v1.19, error-taxonomy v4.6, BC-2.07.002 v1.9, S-6.03 v2.8.
- Adjudicated deferrals honored (wire handlers, HISTORICAL-PROVENANCE at named sites, `tc := tc` under Go 1.25, `listKeysArgs` inline type, `internal/adminwire` extraction, DRIFT-P5P4-PROMPT-SHORTID, E-CFG-002/E-CFG-006 collision, strings.Fields fragility) — none re-reported.

## Findings

None. No HIGH/MED/LOW findings identified within the review perimeter.

## Traceability Audit

Assertion-anchor constraint (assertion asserting a code minted in vX.Y must cite ≥ vX.Y) checked across in-perimeter test files:

- `production_exit_code_test.go` cites interface-definitions.md **v1.18** §133/§174/§71-73 — v1.18 is minting-time for the E-CFG-012/E-CFG-013 exit-2 discrimination and sessions sub-verb routing (F-P5P6-A-001/003/006). Tip has since advanced to v1.19 (superset). Admissible.
- `admin_confirm_symmetry_test.go`, `admin_emission_text_test.go`, `admin_wire_tag_test.go`, `admin_interactive_prompt_test.go` — Burst 19/21 files, HISTORICAL-PROVENANCE exempt per dispatch.
- No test asserts a wire-tag, error-code, or exit-code minted after its cited spec version.

Mock↔real-struct check: `admin_wire_tag_test.go` marshals the real `AdminSvtnCreateArgs` / `AdminSvtnDestroyArgs` / `AdminKeyRegisterArgs` from `admin.go` directly; no mock shapes introduced.

New exit-code discriminator contract (PR #65: `*usageError` → exit 2, others → exit 1) coverage: `production_exit_code_test.go` exercises all six main.go usage-error paths through subprocess re-exec (E-CFG-012, E-CFG-013, missing --name, unknown admin sub-verb, missing --key on `key register`, unknown top-level subcommand), plus bare no-args (exit 2 + enumerated subcommand list), plus sessions sub-verb routing (attach/detach/status/bogus → exit 2; list → exit 1 on E-NET-001; bare `sessions` defaults to list). All discriminator branches in `main.go:100-107` have covering cases. Adequate.

## Observations

**OBS-P5P7-B-001** — `cmd/sbctl/admin_confirm_symmetry_test.go:69-97`
`TestNewInBurst19_ConfirmSymmetry_BoolFlagAcceptsValueForm` declares a `wantParseOK bool` field but every case sets it to `true`; the `else` branch at line 130-134 handling the false-parse oracle is dead. The header comment scopes the failure-case oracle out, but the struct field is now vestigial. Cosmetic.

**OBS-P5P7-B-002** — `cmd/sbctl/admin_confirm_symmetry_test.go:243-263`
`TestNewInBurst19_ConfirmSymmetry_SvtnDestroyConfirmIsString` uses a negative-only oracle (`"invalid value"` MUST be absent from stderr). It does not round-trip the parsed token into the destroy args or verify end-to-end preservation. A caller could accept the flag, silently truncate it, and still pass. Acceptable under F-A-009 scope (UX symmetry, not shape validation — which lives in the wire-shape + E-CFG-012/E-CFG-013 tests), but the guard is narrower than the test name might suggest.

**OBS-P5P7-B-003** — `cmd/sbctl/main_test.go:293` (docstring)
Docstring opening paragraph still says `// TestSbctl_NoSubcommand_ExitsZero verifies the no-subcommand behavior...` while the function beneath is now `TestSbctl_NoSubcommand_ExitsTwoAfterP6`. Stale header after the Burst 23 rename. Grep-for-function-name may mislead.

**OBS-P5P7-B-004** — `cmd/sbctl/production_exit_code_test.go:246-250` (Case 6 comment)
Comment states: `main.go default arm already calls os.Exit(2)`. Post-PR #65, `main.go:97` returns `usageErrf(...)` and exit-2 is mapped by the `errors.As(&ue)` discriminator at `main.go:102-104`; the default arm no longer calls `os.Exit` directly. Assertion remains correct; rationale text describes the pre-refactor path. Comment drift, not test-body defect.

**OBS-P5P7-B-005** — `cmd/sbctl/admin_test.go:2349-2400` vs `cmd/sbctl/production_exit_code_test.go:185-195`
`TestSubprocessAdmin_YesPlusConfirmExitCode2` and the `destroy_yes_plus_confirm_E-CFG-012` case of `TestProductionMain_UsageErrors_ExitTwo` assert identical stimulus (`admin svtn destroy --name foo --yes --confirm SVTN-aabbccdd`), identical exit code (2), and identical stderr substring (`E-CFG-012`). Reconciliation comment at `admin_test.go:2358-2363` marks the overlap intentional (pre-PR-#65 reconciled owner), so redundancy is deliberate rather than accidental; costs one subprocess exec per run without asserting anything the newer table case does not.

## Rubric Application

- **POL-001 (changelog-completeness MED)** — no in-perimeter test artifact touched changelog surface. Not applicable.
- **POL-002 (story-index-row-sync MED)** — no in-perimeter test artifact touched story-index. Not applicable.

## Verdict Rationale

Five observations, zero findings at LOW or higher. Observations are cosmetic (stale docstring, stale comment) or narrow-oracle notes that document intent already captured in the test file headers; none rise to a rigor gap that would let a regression through. The new exit-code discriminator contract has adequate table-driven coverage across all six main.go usage-error branches plus top-level bare-invocation and sessions sub-verb routing paths. No mock↔real-struct divergence. All ASSERTION-ANCHOR citations satisfy the assertion-vs-cited-version constraint (either at minting-time or under HISTORICAL-PROVENANCE exemption). OBS-only ⇒ CLEAN per dispatch rubric.

VERDICT: CLEAN
