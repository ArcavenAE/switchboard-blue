```yaml
---
document_type: adversarial-review
artifact_id: P5-pass-8-Adv-B
verdict: HAS_FINDINGS
finding_counts:
  high: 0
  med: 2
  low: 0
  obs: 1
develop_tip: b4ccd061b03103e677837511233d3301de0e44f9
tuple_match: true
phase: 5
pass: 8
variant: Adv-B
lens: test-rigor+traceability
model: opus-4-7
time_spent_minutes: 7
files_read: 9
read_cap: 6
read_cap_overage_disclosed: true
prior_passes_read: false
---
```

Preflight: `.git/refs/heads/develop` = `b4ccd061b03103e677837511233d3301de0e44f9`. Matches dispatched tuple; no ABORT_STALE_CHECKOUT.

Read-cap disclosure (contract-required, not concealed): the perimeter of the CLI + admin test tier plus two spec-anchor confirmations required ~9 file touches (7 full-file Reads: `production_exit_code_test.go`, `main_test.go`, `console_test.go`, `admin_confirm_symmetry_test.go`, `admin_emission_text_test.go`, `admin_wire_tag_test.go`, `admin_handlers_emission_text_test.go`; 2 partial Reads: `router_status_test.go` 1-600, `admin_interactive_prompt_test.go`), plus lightweight greps against `interface-definitions.md v1.19`, `error-taxonomy.md v4.6`, and `admin.go` — over the ≤6 full-file cap. Overage acknowledged; no readings suppressed.

Adjudicated deferrals honored (not re-reported): vestigial `wantParseOK`; `SvtnDestroyConfirmIsString` negative-only oracle; Burst 19/21 v1.17 §125/§129/§130 historical citations; DRIFT-P5P4-PROMPT-SHORTID; `adminListKeysArgs` inline-struct wire coverage via daemon side.

---

## Findings

### F-P5P8-B-001 [MED] — Traceability misattribution: t.Errorf messages hardcode F-P5P6-A-001 for cases minted by F-P5P7 [process-gap]

**File:** `cmd/sbctl/production_exit_code_test.go:366-370` (inside `TestProductionMain_UsageErrors_ExitTwo`).

**Evidence.** The header comment blocks at lines 156-159 and 269-274 explicitly partition the 12 sub-cases: Cases 1-6 originate from finding **F-P5P6-A-001** (Pass 6 remediation — unknown-verb / missing-args / bare `sbctl` / `--help` / `-h` / unknown-global-flag), Cases 7-12 originate from Pass 7 findings — **F-P5P7-A-001** (bare `sbctl svtn`), **F-P5P7-A-002** (`admin svtn create` missing bootstrap-pubkey; `admin svtn destroy` missing `--confirm`), and **F-P5P7-A-003** (bare `admin`, bare `admin key`, unknown admin verb). However, the shared assertion arm that fires on any case mismatch (lines 366-370) reports every failure with the string `"F-P5P6-A-001"`, regardless of which case row failed. A failing Case 7 (bare `sbctl svtn`) therefore points a debugger at Pass 6's finding record instead of F-P5P7-A-001.

**Spec anchor.** `interface-definitions.md v1.19 §174` (exit-code discrimination; usageError → exit 2). Attribution is not spec-normative but is contract-normative for the Pass 7 remediation's provenance chain — the Pass 7 findings are cited in the file's own header, so the test emits the citation but drops it in the failure path.

**Why MED, not LOW.** In an adversarial verify-and-remediate loop, misattribution routes remediation cost to the wrong prior artifact. Cases 7-12 encode three distinct Pass-7 remediations; a green suite hides the defect, a red suite mislabels it. Same file, one-line-per-case fix (constant → indexed lookup or per-case tag).

---

### F-P5P8-B-002 [MED] — Vacuous alias / dispatch oracle: canned daemon never inspects `req["cmd"]` [process-gap]

**File:** `cmd/sbctl/router_status_test.go:61-127` (`startCannedDaemon` + `serveCannedConn` helpers); consumer tests at `:148` (`TestSbctlPathsList_OutputsCanonicalFields`), `:214` (`TestSbctlRouterMetrics_OutputsSVTNMetrics`), `:270` (`TestSbctlRouterStatus_IsAliasForPathsList`).

**Evidence.** `serveCannedConn` decodes an incoming request map but reads only `req["id"]` (for the JSON-RPC id echo). It never asserts on `req["cmd"]`. It then writes a canned response envelope regardless of whether the CLI dispatched `paths.list`, `router.metrics`, `router.status`, or `wallpaper.paint`. The docstring on `TestSbctlRouterStatus_IsAliasForPathsList` (line 270) explicitly claims the test verifies "that both commands invoke the same underlying paths.list RPC (single code path, no divergent implementation per F-P8-002 ruling)" — but the harness cannot observe cmd dispatch. The oracle for AC-003 / BC-2.06.003 PC-3 is therefore response-shape identity, not dispatch identity, and any future refactor that split `router status` onto its own RPC (say, `router.status` returning a shape-identical payload) would pass this test green while violating the aliasing contract the test names.

**Contrast (existence proof, sibling module):** `cmd/sbctl/console_test.go:46-49, 72-73, 92-93` — `startFakeServer` decodes the request and validates `req["cmd"]` against `expectedCmd`, letting `TestSbctlConsole_Attach/Detach/Switch` (BC-2.08.001) actually enforce cmd dispatch. The pattern exists and is applied one directory sibling away; the omission in `router_status_test.go` is not a platform limitation.

**Spec anchor.** `interface-definitions.md v1.19 §80-81` (router status --target aliasing to paths.list); BC-2.06.003 v1.13 PC-3 + EC-005.

**Why MED, not HIGH.** The suite still fails on the response-shape contract, so a regression that broke the CLI's field extraction would surface. But the specific single-code-path claim asserted in the docstring is unenforced — the test is name-vs-assertion mismatched. Same-file remediation (add `expectedCmd` field to canned config, assert in `serveCannedConn`). Also strengthens `TestSbctlPathsList` and `TestSbctlRouterMetrics` dispatch coverage in the same change.

---

## Observations

### OBS-P5P8-B-001 — `bare_sessions_defaults_to_list` uses exit-code-only oracle

**File:** `cmd/sbctl/production_exit_code_test.go:537` (sub-test inside `TestProductionMain_Sessions_SubVerbValidation`).

**Note.** The `bare_sessions_defaults_to_list` case asserts `exitCode != 1` (the sub-test's guard against usageError-vs-operational conflation). It carries no stderr-fingerprint assertion — for the paths.list dispatch, the natural sentinel is `E-NET-001` (daemon-unreachable) canonical prefix under BC-2.07.003, since the tests run without a daemon. An `exitCode != 1` assertion admits exit 2 (usageError) and exit 0 (which would indicate a stubbed success) as green. Weak but not incorrect: sibling cases in the same test do carry stderr fingerprints, so if the exit-2 partition regressed the sibling would catch it. Observation only — remediation is to add a `wantStderrContains: "E-NET-001"` check for this case.

---

## Rubric summary

POL-001 (changelog completeness) and POL-002 (story-index-row-sync) were out of scope for this test-tier lens and not evaluated. F-P5P8-B-001 is content-only within a test file (no changelog impact); F-P5P8-B-002 is content-only within a test file (no changelog impact). No spec version citation drift observed on files read this pass (v1.19 §174 / taxonomy v4.6 / BC-2.06.003 v1.13 / BC-2.07.003 all consistent with the tuple).

VERDICT: HAS_FINDINGS
