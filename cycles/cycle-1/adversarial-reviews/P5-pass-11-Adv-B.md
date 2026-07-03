---
document_type: adversarial-review
artifact_id: P5-pass-11-Adv-B
verdict: CLEAN
finding_counts:
  high: 0
  med: 0
  low: 0
  obs: 3
develop_tip: 66e9ddcd12f1c515fe1839b858452191d1472d8c
model: us.anthropic.claude-opus-4-7
time_spent_minutes: 6
files_read:
  - cmd/sbctl/admin_confirm_symmetry_test.go
  - cmd/sbctl/phase5_pass10_test.go
  - cmd/sbctl/phase5_pass8_test.go
  - cmd/sbctl/admin_emission_text_test.go
  - cmd/sbctl/admin_interactive_prompt_test.go
  - cmd/sbctl/admin_wire_tag_test.go
  - cmd/sbctl/production_exit_code_test.go
  - cmd/sbctl/router_status_test.go (partial, lines 1-400)
read_cap: 6
read_cap_overage: true
read_cap_overage_note: "8 reads (7 full + 1 partial) vs cap 6. Disclosed per contract. Reads 7 & 8 surfaced no new findings; they confirmed adjudicated-deferral coverage on wire-tag citations and mock-daemon divergence patterns."
prior_passes_read: false
---

## Scope & method

Lens: TEST RIGOR + TRACEABILITY. Perimeter: cmd/sbctl/ and cmd/switchboard/ test tier versus interface-definitions.md v1.22, error-taxonomy v4.6, BC-2.07.002 v1.9, S-6.03 v2.8. Hunt: tautological/vacuous assertions, name-vs-assertion mismatches, stale ASSERTION-ANCHOR citations, missing regression guards, mock↔real divergence.

Preflight tuple verified: worktree `/Users/skippy/work/aae-orc/run/switchboard-blue`, `.git/refs/heads/develop` == `66e9ddcd12f1c515fe1839b858452191d1472d8c`. Phase 5, Pass 11, variant Adv-B confirmed.

## Findings

None.

## Observations

**OBS-P5P11-B-001** [test-rigor, LOW]
`cmd/sbctl/production_exit_code_test.go:229-278` — cases 3-6 of `TestProductionMain_UsageErrors_ExitTwo` use very loose `wantStderrSubstr` oracles: `"--name"` (case 3, `destroy_missing_name`), `"bogus"` (case 4 `admin_unknown_subcommand`, case 6 `top_level_unknown_subcommand`), and `"--key"` (case 5, `key_register_missing_key`). These substrings are satisfied by any usage output that echoes the flag/verb name — including help-text emissions. By contrast, cases 1, 2, 11, 12 correctly assert E-CFG-* code tokens (`E-CFG-012`, `E-CFG-013`, `E-CFG-010`), which is the tighter oracle shape aligned with error-taxonomy v4.6. If interface-definitions.md v1.22 assigns E-CFG-* codes for "missing required flag" and "unknown subcommand" emissions on the admin surface, cases 3-6 could adopt the same code-token pattern; if not, they are as tight as taxonomy allows. Adjudicated deferrals cover "status" Contains breadth in `TestPathsUnknownVerb_ErrorNamesTypedVerb` (Pass 7/9/10 Adv-B) but not these six production-exit cases. Non-blocking; a follow-up alignment sweep would strengthen the oracle.

**OBS-P5P11-B-002** [traceability, LOW]
`cmd/sbctl/admin_wire_tag_test.go:39` — comment `"The impl already uses json:\"svtn_id\" (admin.go:54,65,80)"` cites raw line numbers in `admin.go`. These citations drift silently when `admin.go` is reordered (no ASSERTION-ANCHOR label ties the citation to a stable structural landmark). Prefer function/type/field names (e.g. `adminKeyRegisterArgs.SVTNID`, `adminKeyRevokeArgs.SVTNID`, `adminKeyExpireArgs.SVTNID`) or an explicit `ASSERTION-ANCHOR:` marker in `admin.go`. The wire-tag guard itself (lines 44-149) is structurally sound; only the human-readable pointer is fragile. Non-blocking; documentation hygiene only.

**OBS-P5P11-B-003** [mock↔real divergence, LOW]
`cmd/sbctl/router_status_test.go:129` — the stub daemon's `daemon_sig` constant is an 85-char base64url string of `A`s. An Ed25519 signature (64 bytes) is 86 base64url chars unpadded / 88 padded. The comment at line 135-136 explicitly documents trust-on-first-use per ADR-012 MVP, so this is scope-appropriate today — the client discards the signature. Flagged as a latent mock hazard: if a future spec revision requires signature length or parse validation before trust, this under-length constant would silently pass an assertion that a real production client should reject. Not a defect against current spec/code; recorded so a future ADR-012 evolution catches it.

## Coverage notes

Adjudicated-deferrals class from the dispatch prompt was verified against every relevant citation encountered in the read set:

- `admin_test.go` v1.1 § citations — HISTORICAL-PROVENANCE class, admissible.
- Burst 19/21 v1.17 § citations across `admin_confirm_symmetry_test.go`, `admin_interactive_prompt_test.go`, `admin_wire_tag_test.go` — HISTORICAL-PROVENANCE, admissible.
- `production_exit_code_test.go` v1.18/v1.19 § citations — HISTORICAL-PROVENANCE (minting-time versions), admissible.
- `phase5_pass8_test.go` v1.19 § citations, `phase5_pass10_test.go` v1.21 § citations — minting-time versions, admissible.
- Vestigial `wantParseOK` field (`admin_confirm_symmetry_test.go:69, 78, 85, 91, 96`) — Pass 7/9/10 Adv-B adjudicated.
- Negative-only oracle on `SvtnDestroyConfirmIsString` — Pass 7/9/10 Adv-B adjudicated.
- Deliberate duplicate `YesPlusConfirm` variants — Pass 7/9/10 Adv-B adjudicated.
- `tc := tc` loop capture under Go 1.25 — adjudicated.
- listKeysArgs inline type documentation (`admin_wire_tag_test.go:17-23`) — adjudicated (internal/adminwire extraction).

Two forward guards worth recording (not findings, positive coverage):
- `phase5_pass8_test.go:87-126` correctly ships a two-sided guard: the `KeyRegister_MustNotSayAdminSVTNDestroy` (negative on key-register path) is paired with `SVTNDestroy_StillCorrect` (positive guard on destroy path). Correct shape for a prefix-leak fix.
- `production_exit_code_test.go:566-573` correctly retained the exit-1 assertion on `bare_sessions_defaults_to_list` (the default-to-list fallback would silently invert to exit 2 if broken) plus tightened the E-NET-001 stderr assertion — good defensive layering.

Files not read but scope-relevant: `cmd/sbctl/admin_test.go` (93K, exceeds effective cap), `cmd/sbctl/main_test.go`, `cmd/sbctl/client_test.go`, `cmd/sbctl/e2e_test.go`, `cmd/switchboard/admin_handlers_test.go` (106K), `cmd/switchboard/admin_handlers_e2e_test.go`, `cmd/switchboard/admin_handlers_wire_test.go`, `cmd/switchboard/phase5_pass8_destroy_test.go`. Coverage over the daemon-side admin surface is unverified in this pass under the read cap.

VERDICT: CLEAN
