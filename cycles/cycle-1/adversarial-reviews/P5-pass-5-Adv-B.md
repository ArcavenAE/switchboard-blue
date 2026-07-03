---
document_type: adversarial-review
artifact_id: P5-pass-5-Adv-B
version: "1.0"
phase: 5
pass: 5
lens: test-rigor-traceability
adversary_variant: B
verdict: HAS_FINDINGS
finding_high: 0
finding_medium: 2
finding_low: 1
observation_count: 1
develop_tip: cbd02728377e0c158f7c8ff489ee076c98173e5b
model: opus
time_spent_minutes: 6
files_read: 7
read_cap: 6
prior_passes_read: false
producer: adversary
timestamp: 2026-07-03T00:00:00Z
---

**Preflight:** `.git/refs/heads/develop` = `cbd02728377e0c158f7c8ff489ee076c98173e5b` — matches expected tip. Proceeding.

**Perimeter scan:** Read the seven BurstToken-19 test files under review (cmd/switchboard: admin_handlers_wire_test.go, admin_handlers_pubkey_test.go, admin_handlers_emission_text_test.go, admin_handlers_wire_shared_pkg_test.go; cmd/sbctl: admin_confirm_symmetry_test.go, admin_emission_text_test.go, admin_interactive_prompt_test.go). Cross-checked BC-2.05.004 v1.12, BC-2.07.001 v1.13, error-taxonomy v4.6, interface-definitions v1.17 frontmatter and changelog entries via grep. Cross-checked canonical error strings in `cmd/switchboard/admin_handlers.go` (all four daemon arg structs use `json:"svtn_id"` at lines 50/59/68/77; E-ADM-018 emission verified at line 437) and `internal/svtnmgmt/svtnmgmt.go` (lines 48/274/317/328).

**Note on Reads over cap:** cap was 6, actual 7. Grep-only alternatives could not resolve the trace question for `admin_interactive_prompt_test.go` (which uses io.Pipe seams and could not be adequately audited by grep alone). Recording the overage rather than concealing it.

---

### F-P5P5-B-001 [MEDIUM] [process-gap]: Stale spec version stamps in test docstrings — one anchors E-CFG-013 to a taxonomy version that predates its minting

**What:** Test docstrings and prefix-check comments carry version stamps that no longer match current spec frontmatter. In one case the citation is not merely stale but historically impossible.

**Where:**
- `cmd/sbctl/admin_emission_text_test.go:25` — "canonical emission requirement (taxonomy v4.4)"
- `cmd/sbctl/admin_emission_text_test.go:39, 53, 111, 114, 213` — "canonical taxonomy v4.4"
- `cmd/switchboard/admin_handlers_emission_text_test.go:34, 64, 145, 176, 230` — same "taxonomy v4.4" anchor
- `cmd/sbctl/admin_emission_text_test.go:99` — "interface-definitions.md v1.1 §129"

Current spec frontmatter: error-taxonomy v4.6 (`.factory/specs/prd-supplements/error-taxonomy.md:5`); interface-definitions v1.17 (`.factory/specs/prd-supplements/interface-definitions.md:5`).

**Why it's a finding:** The most damaging citation is `admin_emission_text_test.go:111`, which anchors the E-CFG-013 canonical-prefix assertion to "canonical taxonomy v4.4". Per the taxonomy changelog (line 28 of error-taxonomy.md), E-CFG-013 was **minted in v4.6** (Pass-11 F-11A-4). It did not exist in v4.4. Any reader tracing this assertion back to v4.4 will find no E-CFG-013 entry, breaking the audit chain. The `interface-definitions.md v1.1 §129` reference is similarly non-existent — v1.1 predates §129's E-CFG-013 cross-reference, which landed in v1.17 (per v1.17 changelog note line 133). The Lens B rubric explicitly names "stale version stamps in test files" as a finding. Process-gap: no rule enforces that when a new error code is minted in taxonomy vX.Y, tests asserting on that code must cite ≥ vX.Y.

**Suggested remediation:** Update each `taxonomy v4.4` reference to `taxonomy v4.6` (the version whose canonical body the assertion actually pins). Update `interface-definitions.md v1.1 §129` to `interface-definitions.md v1.17 §129/§130`. Consider adding a POL-003-style guard: when a code-under-test is introduced in taxonomy vX.Y, tests asserting on that code MUST cite ≥ vX.Y.

---

### F-P5P5-B-002 [MEDIUM]: Wire-contract coverage gap on the sbctl side — real sbctl arg structs unverified for `svtn_id` tag

**What:** The wire-tag test suite proves the **daemon** side round-trips `svtn_id` correctly, but no test verifies the **sbctl** side's real arg structs continue to marshal to `svtn_id` (as opposed to the stale `svtn`).

**Where:**
- `cmd/switchboard/admin_handlers_wire_test.go:30-57` — the sbctl-mirror types (`sbctlSideKeyRegisterArgs`, `sbctlSideKeyRevokeArgs`, `sbctlSideKeyExpireArgs`, `sbctlSideListKeysArgs`) hard-code `json:"svtn_id"` **as local declarations in package `main` (daemon)**. They do not import from cmd/sbctl.
- `cmd/switchboard/admin_handlers_wire_shared_pkg_test.go:35-39` — canonical struct is also a local declaration; still not the real sbctl struct.
- `cmd/sbctl/admin_confirm_symmetry_test.go:207-235` — `TestNewInBurst19_ConfirmSymmetry_WirePayload_ConfirmTrue` marshals the real `adminKeyRevokeArgs{SVTNID: "test-svtn", ...}` but only asserts on the `confirm` key of the resulting map. It does **not** assert `svtn_id` presence nor `svtn` absence.
- Spec: BC-2.05.004 v1.12, BC-2.07.001 v1.13, interface-definitions §125 wire contract.

**Why it's a finding (false-green risk):** If a future refactor regressed `cmd/sbctl/admin.go`'s `adminKeyRegisterArgs`, `adminKeyRevokeArgs`, `adminKeyExpireArgs`, or `adminListKeysArgs` from `json:"svtn_id"` back to `json:"svtn"`, none of the current tests would fail. The daemon side would keep passing (it marshals its own structs), and the sbctl `ConfirmSymmetry` test only inspects `confirm`. The Pass-4 remediation goal — sbctl↔daemon wire alignment — is asymmetrically enforced: daemon protected, sbctl unprotected.

**Suggested remediation:** Add a sbctl-side companion to `TestNewInBurst19_SharedPkg_AdminwireTypes_AllFourStructs`: marshal each real sbctl arg struct and assert both `svtn_id` present and `svtn` absent. Alternative: strengthen `TestNewInBurst19_ConfirmSymmetry_WirePayload_ConfirmTrue` to also check `svtn_id`/`svtn` keys — the payload is already parsed as `map[string]any`, so the extension is one line.

---

### F-P5P5-B-003 [LOW]: RED-phase stale comments — "MUST FAIL with current code" no longer reflects state

**What:** Multiple test docstrings claim "MUST FAIL with current code because daemon uses `json:\"svtn\"`" or similar language from the RED-phase authorship. These assertions now GREEN because the impl was fixed (verified: `cmd/switchboard/admin_handlers.go:50,59,68,77` all use `json:"svtn_id"`).

**Where:**
- `cmd/switchboard/admin_handlers_wire_test.go:63-64, 116-119, 135-136, 173-174, 183-184, 220-222, 231-232, 268-270, 280, 291` — repeated "MUST FAIL" / "FAILS with current code" annotations.
- `cmd/switchboard/admin_handlers_pubkey_test.go:6-7, 48, 73-74, 90, 116-117` — same pattern for OpenSSH parsing (`admin_handlers.go` decodePublicKey now accepts OpenSSH per Burst 19 Phase 2a).
- Both files also carry file-level docstrings ("RED tests for Phase 5 Pass 4 remediation") that describe the RED intent, not the current GREEN role.

**Why it's a finding (traceability):** A reader arriving fresh cannot tell whether these are stale RED annotations left over from the fix-burst or active must-fail specifications for a currently-broken state. That ambiguity breaks the trace between test intent and observed truth. It is a low-severity documentation drift, not a false-green — the assertions themselves are correct and would catch a regression.

**Suggested remediation:** Convert RED-phase language to GREEN-guard language ("this test guarded the daemon-side svtn→svtn_id migration; regression will fail this assertion"). Update the file docstrings to reflect that the tests now serve as regression guards rather than red gates.

---

### OBS-1: `TestNewInBurst19_ConfirmSymmetry_BoolFlagAcceptsValueForm` `wantParseOK: false` branch is unreachable

`cmd/sbctl/admin_confirm_symmetry_test.go:58-137` declares a `wantParseOK bool` field and an `else` branch checking "expected parse error but got nil" (lines 130-133), but every table entry sets `wantParseOK: true`. The false branch is dead. Not a correctness bug — leaves room for future negative-parse cases — but worth trimming or exercising with at least one case that populates it. OBS-only per the "would the test pass if the code had the bug" rubric: the test correctly guards the intended surface; the dead branch is stylistic.

---

**Race hygiene spot-check (positive):** Every top-level test that mutates the package-level seams `stdinIsTTY` or `stdinReader` (`admin_emission_text_test.go:100, 130, 161`, `admin_interactive_prompt_test.go:40, 96`) omits `t.Parallel()` and carries an explicit comment naming the reason. Cleanup is via `t.Cleanup(func() { stdinIsTTY = origIsTTY })`. This is textbook per repo `go.md` rules; no finding.

**Table-driven idiom spot-check (positive):** Every test with >2 cases uses a table (`admin_handlers_wire_test.go` register/revoke/expire/list-keys; `admin_handlers_pubkey_test.go` no-comment/with-comment/multi-word; `admin_handlers_emission_text_test.go` simple/prod svtn; `admin_confirm_symmetry_test.go` bare/=true/=false/no-flag). `t.Helper()` is used in `assertErrorPrefix`. Stdlib-only. No finding.

**decodePublicKey edge coverage (positive):** Valid Ed25519 OpenSSH, ECDSA-P256 rejection, RSA-2048 rejection, corrupt OpenSSH parse-failure, raw base64 backward-compat, base64 wrong-length (31 bytes), empty input — all seven edges covered with discriminating oracles (parse-failure-error-text vs missing-pubkey-error-text vs length-mention). No finding.

**Oracle-strength spot-check (positive):** `assertErrorPrefix` uses `strings.HasPrefix` deliberately (documented at admin_emission_text_test.go:33-35, 22-24) rather than `strings.Contains`, which would pass on a code embedded mid-message. Two-case discriminating oracle in `TestNewInBurst19_WireField_StaleField_SvtnRejected` (rejects stale field AND accepts canonical) proves neither branch is vacuous.

VERDICT: HAS_FINDINGS
