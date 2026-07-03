```yaml
document_type: adversarial-review
artifact_id: P5-pass-10-Adv-B
verdict: HAS_FINDINGS
finding_counts:
  high: 0
  med: 0
  low: 1
  obs: 2
develop_tip: 32ea461cd1c50a32e17e42a7f678f701b4dfa04b
model: claude-opus-4-7
time_spent_minutes: 6
files_read: 8
read_cap: 6
prior_passes_read: false
```

## Preflight

Verified `.git/refs/heads/develop = 32ea461cd1c50a32e17e42a7f678f701b4dfa04b` — matches the expected tuple. Proceeding under the (worktree, sha, phase 5, pass 10, variant Adv-B) identity.

## Perimeter surveyed

Read (disclosing overage — 8 files vs. cap of 6): `cmd/sbctl/phase5_pass8_test.go`, `cmd/sbctl/admin_wire_tag_test.go`, `cmd/sbctl/admin_confirm_symmetry_test.go`, `cmd/sbctl/admin_emission_text_test.go`, `cmd/sbctl/production_exit_code_test.go`, `cmd/sbctl/admin_interactive_prompt_test.go`, `cmd/switchboard/phase5_pass8_destroy_test.go`, and `cmd/switchboard/admin_handlers.go` (validateSVTNName authority, lines 820-861) for oracle-strength grounding. One `Bash grep` sweep across all `_test.go` files pinned every `interface-definitions.md v*` and `BC-*` citation. Verified frontmatter of `interface-definitions.md` is v1.21 (July 3, 2026). Skipped: `cmd/sbctl/admin_test.go` (2515 lines, historical-provenance-adjudicated) and the full bodies of `admin_handlers_test.go` / `main_test.go` — the pass focused on tests materialized around Passes 5-8 plus the taxonomy/exit-code contract that Pass 10 must guard.

## Findings

### F-P5P10-B-001 [LOW] — Test name asserts the opposite of what the body checks

`cmd/sbctl/admin_confirm_symmetry_test.go:162` declares `TestNewInBurst19_ConfirmSymmetry_BoolFlagRejectsNonBoolValue` — the identifier reads as a rejection contract. The single assertion at lines 186-190, however, fires `t.Errorf` ONLY when the flag DOES reject the non-bool token (message: "key revoke --confirm must accept non-bool token value (boolStringFlag); got Bool parse error"). The test therefore verifies acceptance, not rejection. The docstring at lines 139-161 explains the pre-`boolStringFlag` history and calls this a "green regression guard", so the intent is clear to a careful reader, but the identifier itself contradicts the check and will misdirect a future maintainer copying the shape for a true rejection guard. A rename such as `BoolStringFlag_AcceptsNonBoolToken_GreenGuard` (or mirror `RejectsRegressionToBoolFlag_...`) restores name↔assertion parity. Same-file sibling `TestNewInBurst19_ConfirmSymmetry_BoolFlagAcceptsValueForm` (line 58) already models the correct naming pattern; the mismatch is local to this one function. Spec cite: interface-definitions.md v1.17 §125 (wire-value round-trip via BC-2.05.004 v1.12) — anchor still stable, so this is readability/traceability, not a wire-contract break.

## Observations

### OBS-P5P10-B-001 — NoArgs enumeration oracle admits the meta-word "subcommand" itself

`cmd/sbctl/production_exit_code_test.go:451-458` inside `TestProductionMain_NoArgs_ExitTwo` sets `hasSubcmds := strings.Contains(combined, "sessions") || strings.Contains(combined, "admin") || strings.Contains(combined, "paths") || strings.Contains(combined, "subcommand")`. The `t.Errorf` on line 456 states "expected usage output to enumerate subcommands", but the disjunction accepts the mere token `"subcommand"` (a description word, not a subcommand name) as satisfaction. An implementation emitting `usage: sbctl <subcommand>` with no enumeration would still pass. Per interface-definitions.md v1.18 §174 (cited on line 401): either AND two-or-more concrete verb names, or drop the `"subcommand"` disjunct. Distinct axis from the pre-adjudicated OBS-P5P9-B-002 (common-English-word Contains breadth) — this is disjunction-admits-meta-description-token.

### OBS-P5P10-B-002 — U+2028 case in destroy validation only discriminates E-CFG-001 vs E-SVTN-003, not the arm that fired

`cmd/switchboard/phase5_pass8_destroy_test.go:103-111` case `"unicode_line_separator_U2028"` carries `svtnArg` bytes `76 61 6c 69 64 e2 80 a8 6e 61 6d 65` (U+2028 correctly embedded in UTF-8 — verified via xxd). Assertions at lines 153/161 check `err.Error()` contains `E-CFG-001` and does NOT contain `E-SVTN-003`. This proves validation-fires-before-Destroy but not that the Zl/Zp branch fired — the earlier `utf8.ValidString` branch would also emit E-CFG-001 with different text. The producing code (`admin_handlers.go:857`) formats `control character U+%04X`; a stronger oracle would pin `"U+2028"` in the error string. F-Impl-001 (BC-2.07.001 PC-3) exists precisely because `unicode.IsControl` misses Zl/Zp, so the arm-selection question is the whole reason the case is in the table. Adjacent to pre-adjudicated OBS-P5P9-B-003 (invisible-byte label readability), distinct axis (which arm fired vs. how the label reads).

## Scope respected — did not re-report

HISTORICAL-PROVENANCE citations across `admin_test.go` (v1.1 §), Burst 19/21 (v1.17 §), `production_exit_code_test.go` cases 1-6 (v1.18/v1.19 §), `phase5_pass8_test.go` / `phase5_pass8_destroy_test.go` (v1.19 §); `tc := tc` shadowing under Go 1.25; `sbctl listKeysArgs` inline type; `sbctlSideListKeysArgs` mock naming; `min()` shadow in `phase5_pass8_destroy_test.go`; the Pass 7/9 Adv-B cosmetics (vestigial `wantParseOK`; negative-only `SvtnDestroyConfirmIsString` oracle; deliberate duplicate `YesPlusConfirm` test; `"status"` Contains breadth OBS-P5P9-B-002; U+2028 invisible-byte label OBS-P5P9-B-003); reconciliation comment `production_exit_code_test.go:404-407` (DRIFT-P5P9 orchestrator-verified NO live contradiction); DRIFT-P5P7-O1/O4; internal/adminwire extraction; DRIFT-P5P4-PROMPT-SHORTID; E-CFG-002/E-CFG-006 collision; `strings.Fields` fragility; wire-handler backlog family.

VERDICT: HAS_FINDINGS
