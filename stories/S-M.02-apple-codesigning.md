---
artifact_id: S-M.02-apple-codesigning
document_type: story
level: ops
story_id: S-M.02
title: "formalize Apple code-signing and notarization of release binaries (toggle-gated)"
status: draft
producer: story-writer
timestamp: 2026-06-26T00:00:00Z
phase: 2
epic: E-MAINT
wave: null
# wave: unscheduled — maintenance epic, NOT part of feature waves 1-7.
# ACTIVATION: milestone-gated. The signing toggle (vars.SIGNING_ENABLED) defaults
# to OFF/false. It must NOT be turned on until the product reaches the
# "functional-product" milestone (i.e., Waves 1-7 delivered, the binary is
# testable end-to-end). Enabling it before that milestone wastes Apple notarization
# quota and blocks releases on credentials the team may not yet have provisioned.
priority: P2
scope_phase: M
estimated_points: 5
tdd_mode: facade
# tdd_mode: facade — this story delivers justfile recipe additions, workflow YAML
# edits, scripts, and doc updates, not Go application logic. There is no meaningful
# Red-Gate TDD loop for justfile recipes or GitHub Actions YAML. Quality gate is a
# green signed-release dry-run (just sign && just verify succeed locally; CI
# sign-and-notarize job passes on a tag push with SIGNING_ENABLED=true).
behavioral_contracts: []
# BC status: pending PO authorship
# This is release-infrastructure / DX work — no product behavioral contracts apply.
# Status remains draft until an owner decides whether to promote or treats the ACs
# below as sufficient governance.
verification_properties: []
depends_on: []
blocks: []
subsystems: []
architecture_modules: []
cycle: v1.0.0-greenfield
implementation_strategy: gene-transfusion
transfusion_source:
  language: just + shell
  package: aae-orc/ThreeDoors
  module: ThreeDoors/justfile (signing/notarization section) + ThreeDoors/.github/workflows/release.yml
  version: "current HEAD"
  license: proprietary (same operator)
  notes: >
    ThreeDoors is the most mature aae-orc product with full codesign + notarize +
    staple + pkg + dmg recipes and the _require-var helper pattern. The aae-orc
    monorepo does not have a standalone "jira-cli" product; ThreeDoors is the correct
    reference. switchboard-blue's release.yml and justfile already incorporate earlier
    transfusions of these patterns (sign/verify recipes, sign-and-notarize CI job with
    SIGNING_ENABLED gate) — this story completes and reconciles that partial transfusion.
inputDocuments:
  - 'justfile'
  - '.github/workflows/release.yml'
  - '.github/workflows/release-verify.yml'
  - 'CONTRIBUTING.md'
  - 'CLAUDE.md'
---

# S-M.02: Formalize Apple Code-Signing and Notarization of Release Binaries (Toggle-Gated)

> **Note:** This is a maintenance/release-infrastructure story with no product
> behavioral contract anchor. Execute outside of feature waves.
>
> **ACTIVATION GATE:** The signing toggle (`vars.SIGNING_ENABLED`) defaults to
> `false`. Do NOT enable it until the product reaches the functional-product
> milestone (Waves 1-7 delivered, binary is end-to-end testable). Enabling early
> wastes Apple notarization quota and may block releases before credentials are
> provisioned.

## Narrative

- **As a** release engineer or maintainer publishing a stable switchboard release
- **I want** macOS binaries to be Apple code-signed (Developer ID Application),
  notarized, and stapled via a SIGNING_ENABLED toggle that defaults to OFF
- **So that** signed releases can be shipped to users without Gatekeeper warnings
  once the product is functional, without blocking current alpha/dev builds or
  requiring Apple credentials before the product is ready to ship

## Acceptance Criteria

### AC-001 — signing toggle gates the sign-and-notarize job (toggle default: OFF)
The `sign-and-notarize` job in `.github/workflows/release.yml` runs only when
`vars.SIGNING_ENABLED == 'true'` (a GitHub Actions repository variable, NOT a
secret). When the variable is absent, empty, or any value other than `'true'`,
the job is skipped and the release pipeline completes successfully with unsigned
binaries. No CI failure, no warning annotation for the skip.

Current state: release.yml already has `if: vars.SIGNING_ENABLED == 'true'` on
the sign-and-notarize job. This AC requires verifying the condition is correct,
confirming the `release` job's `if: always() && ...` expression correctly handles
`sign-and-notarize.result == 'skipped'`, and documenting the toggle in CONTRIBUTING.md.

### AC-002 — when toggle is ON: macOS binaries are signed (Developer ID Application)
When `SIGNING_ENABLED=true`, the sign-and-notarize job:
1. Imports the Developer ID Application certificate from `APPLE_CERTIFICATE_P12`
   (base64-encoded .p12) + `APPLE_CERTIFICATE_PASSWORD` into a temporary build keychain.
2. Signs `switchboard-darwin-arm64` and `switchboard-darwin-amd64` with
   `codesign --force --options runtime --sign "$APPLE_SIGNING_IDENTITY" --timestamp`.
3. Verifies each signature with `codesign --verify --deep --strict`.
4. All signing identity and credential values come exclusively from GitHub Actions
   repository secrets (see AC-005 for the required secret names). No secret is
   committed to the repository.

### AC-003 — when toggle is ON: artifacts are notarized and stapled
After signing, the sign-and-notarize job notarizes and staples each macOS artifact
(at minimum: the signed binaries; optionally .pkg and .dmg if those scripts exist)
using `xcrun notarytool submit ... --wait --timeout 14400` followed by
`xcrun stapler staple`. Notarization credentials come from the three Apple secrets
documented in AC-005. After stapling, Gatekeeper assessment (`spctl --assess`)
passes on each artifact.

### AC-004 — just sign / just verify are reconciled with the CI signing path
The justfile `sign` and `verify` recipes are updated so that:
- `just sign` signs the appropriate binary for the current platform (used locally
  by the developer / operator for a dev-build signature check).
- A new `just sign-release` recipe (or equivalent naming) handles multi-arch:
  it signs both `bin/switchboard-darwin-arm64` and `bin/switchboard-darwin-amd64`
  using the same `codesign` flags as the CI job. CI's sign-and-notarize step
  invokes these justfile recipes (or sources the same shell fragment) so there is
  a single source of truth for the codesign flags.
- New supporting justfile recipes are added, adapted from aae-orc/ThreeDoors:
  `notarize <artifact>`, `notarize-status <id>`, `sign-check` (list Developer ID
  identities), `gatekeeper-check`. The `_require-var` helper already exists and
  is reused.

### AC-005 — required CI secrets and variables are documented
CONTRIBUTING.md (or a linked doc) lists every secret and variable that must be
configured in the GitHub repository before enabling signing:

| Name | Kind | Description |
|------|------|-------------|
| `APPLE_CERTIFICATE_P12` | Secret | Base64-encoded Developer ID Application .p12 certificate |
| `APPLE_CERTIFICATE_PASSWORD` | Secret | Passphrase for the .p12 certificate |
| `APPLE_INSTALLER_CERTIFICATE_P12` | Secret | Base64-encoded Developer ID Installer .p12 (for .pkg; may be deferred) |
| `APPLE_INSTALLER_CERTIFICATE_PASSWORD` | Secret | Passphrase for installer .p12 (may be deferred) |
| `APPLE_SIGNING_IDENTITY` | Secret | Codesign identity string, e.g. `Developer ID Application: Foo Inc (TEAMID)` |
| `APPLE_INSTALLER_IDENTITY` | Secret | Productsign identity string for .pkg (may be deferred) |
| `APPLE_NOTARIZATION_APPLE_ID` | Secret | Apple ID (email) used for notarytool |
| `APPLE_NOTARIZATION_PASSWORD` | Secret | App-specific password for notarytool |
| `APPLE_NOTARIZATION_TEAM_ID` | Secret | Apple Developer Team ID |
| `SIGNING_ENABLED` | Repository Variable | Set to `true` to enable signing; omit or set to anything else to disable |

The doc must also state: "All secrets are managed via the `release` GitHub Actions
environment. Do not commit any credential file. Use `.env.local` (gitignored) for
local development."

### AC-006 — release-verify.yml skips signature verification when signing is disabled
`release-verify.yml` is updated (or a new step added) so that signature and
notarization verification only runs when signing was enabled. When the
sign-and-notarize job was skipped, release-verify makes no attempt to verify
signatures and does not fail. If it does run codesign verification, it must check
the artifact actually exists and was signed before asserting.

### AC-007 — .env.local.example and gitignore are updated
A `.env.local.example` file (or an update to an existing one) documents the
local-signing environment variables:

```
# Apple code-signing — copy to .env.local and fill in your values
# Run `just sign-check` to list available identities on this machine.
APPLE_SIGNING_IDENTITY=""
APPLE_NOTARIZATION_APPLE_ID=""
APPLE_NOTARIZATION_PASSWORD=""
APPLE_NOTARIZATION_TEAM_ID=""
```

`.gitignore` must exclude `.env.local` and `*.env.*` (verify the existing pattern
covers this; add if missing). No credential file is ever committed.

### AC-008 — gene-transfer provenance recorded
A comment in the justfile signing section and/or a note in the Spec Patches table
states: "Signing/notarization recipes adapted from aae-orc/ThreeDoors justfile
(signing/notarization section) and aae-orc/ThreeDoors/.github/workflows/release.yml.
switchboard-blue's release.yml already incorporated an earlier partial transfusion
of these patterns; this story completes and reconciles it." The pattern is adapted,
not copied verbatim — switchboard is a CLI tool binary, not an .app bundle, so the
app bundle / .dmg steps are optional and scoped to a future story.

### AC-009 — CONTRIBUTING.md documents the activation milestone
CONTRIBUTING.md (release section) includes a note:

> **Signing activation:** Code-signing is implemented but disabled by default
> (`SIGNING_ENABLED` repository variable not set). Enable it only after the product
> reaches the functional-product milestone (end-to-end testable). Before enabling:
> provision the required secrets in the `release` GitHub Actions environment (see
> below), then set `SIGNING_ENABLED = true` in repository Variables.

## Non-Goals

- Do NOT enable `SIGNING_ENABLED` by default, or set the repository variable, as
  part of implementing this story.
- Do NOT build or sign Windows or Linux binaries (macOS Apple signing only).
- Do NOT implement a Homebrew formula update gated on signing (update-homebrew job
  already depends on sign-and-notarize.result == 'success' in release.yml — leave
  that logic unchanged).
- Do NOT change any Go application code (`cmd/`, `internal/`).
- Do NOT create .app bundle, .dmg, or .pkg scripts unless they already exist; if
  they do not exist, note them as out of scope / future story.
- Do NOT change CI quality-gate steps (formatting, linting, tests).

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Tag push with `SIGNING_ENABLED` not set (or `false`) | sign-and-notarize skipped; release job uses unsigned binaries artifact; release succeeds |
| EC-002 | Tag push with `SIGNING_ENABLED=true` but secrets not provisioned | CI fails at "Import certificates" step with a clear error; does NOT produce a release with unsigned binaries |
| EC-003 | `just sign` run without `APPLE_SIGNING_IDENTITY` set | `_require-var` helper prints actionable error message and exits non-zero before calling codesign |
| EC-004 | `just verify` run on an unsigned binary | `codesign --verify` exits non-zero; recipe fails with a clear error |
| EC-005 | Apple notarytool timeout (14400s) exceeded | CI step fails with timeout; staple step does not run; artifact is NOT uploaded as signed |
| EC-006 | DeveloperIDG2CA.cer curl fetch fails (egress blocked) | CI step fails with clear network error; release-verify should not then attempt signature check |
| EC-007 | update-homebrew job when sign-and-notarize is skipped | update-homebrew has `if: needs.sign-and-notarize.result == 'success'` — it is skipped, not failed; Homebrew tap is NOT updated for unsigned builds |
| EC-008 | .env.local missing locally, developer runs `just sign` | _require-var prints setup instructions (see justfile existing pattern); no silent failure |

## Architecture Mapping

| Component | Location | Pure/Effectful |
|-----------|----------|---------------|
| justfile signing recipes | `justfile` | effectful (shell exec, macOS toolchain) |
| sign-and-notarize job | `.github/workflows/release.yml` | effectful (CI runner, Apple APIs) |
| release-verify signing check | `.github/workflows/release-verify.yml` | effectful (CI runner) |
| `.env.local.example` | repo root | config (pure data) |
| CONTRIBUTING.md release section | `CONTRIBUTING.md` | pure data |

## Token Budget Estimate

| Context Source | Estimated Tokens |
|----------------|-----------------|
| This story spec | ~2,000 |
| `justfile` (current) | ~400 |
| `.github/workflows/release.yml` (current) | ~1,200 |
| `.github/workflows/release-verify.yml` | ~400 |
| `aae-orc/ThreeDoors/justfile` signing section (reference) | ~800 |
| `aae-orc/ThreeDoors/.github/workflows/release.yml` (reference) | ~1,200 |
| `CONTRIBUTING.md` | ~300 |
| Tool outputs overhead | ~300 |
| **Total** | **~6,600** |
| Agent context window | 200K |
| **Budget usage** | **~3.3%** |

## Tasks

1. [ ] Read `justfile`, `.github/workflows/release.yml`, `.github/workflows/release-verify.yml`,
       `CONTRIBUTING.md`, and `.gitignore` in the switchboard-blue repo.
2. [ ] Read `aae-orc/ThreeDoors/justfile` (signing/notarization section) and
       `aae-orc/ThreeDoors/.github/workflows/release.yml` as the gene-transfer reference.
3. [ ] Verify `release.yml` sign-and-notarize `if:` condition and the `release` job's
       `if: always() && ...` expression handle the skipped case correctly (no false failure).
       Patch if needed.
4. [ ] Add to justfile: `sign-release` (multi-arch codesign), `notarize <artifact>`,
       `notarize-status <id>`, `notarize-log <id>`, `notarize-history`,
       `sign-check`, `gatekeeper-check`. Reuse existing `_require-var`.
       Add provenance comment to the signing section.
5. [ ] Update the existing `verify` recipe to accept a binary path argument (or add
       `verify-release` for multi-arch), so CI and local use the same recipe.
6. [ ] Create or update `.env.local.example` with the signing env vars (AC-007).
7. [ ] Verify `.gitignore` excludes `.env.local` and `*.env.*`; add if missing.
8. [ ] Update `release-verify.yml` to skip signature verification when signing was
       not enabled (AC-006).
9. [ ] Update `CONTRIBUTING.md`: add secrets/variables table (AC-005) and
       activation-milestone note (AC-009).
10. [ ] Verify that `update-homebrew` job correctly depends on signed artifacts
        and is skipped (not failed) for unsigned builds — no change expected, confirm only.
11. [ ] Run `just sign-check` locally — confirm identities listed or a clear error
        if no Developer ID is provisioned.
12. [ ] Run `just sign && just verify` locally with a dev build — confirm recipes work.
13. [ ] Open PR; verify CI is green on a non-tag push (signing skipped, build succeeds).
14. [ ] Record dry-run evidence in PR description: CI log showing sign-and-notarize
        skipped cleanly, release created with unsigned binaries.

## Previous Story Intelligence

S-M.01 (mise toolchain): No direct dependency, but if S-M.01 is delivered first,
the CI workflows may use `jdx/mise-action` for Go provisioning. The sign-and-notarize
job runs on `macos-latest` and invokes Apple system tools (`codesign`, `xcrun`,
`security`) — these are macOS-native and are NOT managed by mise. No interaction
expected; verify that a mise-based CI still works on the macOS runner after S-M.01
is merged.

## Architecture Compliance Rules

| Rule | Source | Enforcement |
|------|--------|-------------|
| No Go application code changes | Non-Goals | PR diff must not touch `cmd/` or `internal/` |
| signing toggle must default to OFF | AC-001, operator intent | PR reviewer: repository variable `SIGNING_ENABLED` must NOT be set as part of this PR |
| credentials never committed | AC-005, AC-007 | PR reviewer: diff must not contain any credential value; .gitignore must cover .env.local |
| single source of truth for codesign flags | AC-004 | CI sign step must call the justfile recipe or source the same shell fragment, not duplicate flags |
| update-homebrew must remain gated on signing success | Non-Goals | `update-homebrew.if` must still require `sign-and-notarize.result == 'success'` |

## Library & Framework Requirements

| Tool / API | Version / Constraint | Purpose |
|------------|---------------------|---------|
| `codesign` | macOS-native (Xcode Command Line Tools) | Binary signing |
| `xcrun notarytool` | macOS-native (Xcode 13+) | Notarization submission and stapling |
| `xcrun stapler` | macOS-native | Notarization ticket stapling |
| `security` | macOS-native | Keychain management in CI |
| `spctl` | macOS-native | Gatekeeper assessment |
| `just` | current stable (already pinned in justfile) | Recipe runner |
| GitHub Actions `macos-latest` runner | current | Required for Apple toolchain |
| `actions/upload-artifact` | v7 (already pinned) | Upload signed artifacts |
| `actions/download-artifact` | v8 (already pinned) | Download artifacts for signing job |

No new external library or action dependency is added by this story.

## File Structure Requirements

| File | Action | Purpose |
|------|--------|---------|
| `justfile` | modify | Add sign-release, notarize, notarize-status, notarize-log, notarize-history, sign-check, gatekeeper-check recipes; add provenance comment |
| `.github/workflows/release.yml` | verify + patch if needed | Confirm sign-and-notarize condition and release job if-expression; no structural change expected |
| `.github/workflows/release-verify.yml` | modify | Skip signature verification when signing was not enabled |
| `.env.local.example` | create (if not exists) or modify | Document local signing env vars |
| `.gitignore` | verify + patch if missing | Ensure `.env.local` and `*.env.*` are excluded |
| `CONTRIBUTING.md` | modify | Add secrets table, activation-milestone note, signing section |

## Open Questions for Orchestrator

1. **Installer artifacts (.pkg, .dmg):** `release.yml` already has `create-app.sh`,
   `create-dmg.sh`, `create-pkg.sh` script invocations in the sign-and-notarize job.
   These scripts do not yet exist in the repo. Should creating them be in scope for
   this story (gene-transfer from aae-orc/ThreeDoors/scripts/) or deferred to a
   follow-on story (e.g., S-M.03-release-packaging)? Recommendation: defer — this
   story's minimum viable scope is signed raw binaries. If the scripts are missing,
   the sign-and-notarize job will fail when SIGNING_ENABLED=true; document this as
   a known gap and note which script paths are expected.

2. **`release.yml` scripts path:** The current release.yml references
   `scripts/create-app.sh`, `scripts/create-dmg.sh`, `scripts/create-pkg.sh` but
   these do not appear to exist in switchboard-blue. Confirm before implementing;
   if they are missing, decide whether to (a) stub them out in this story,
   (b) remove those steps and scope signing to raw binaries only for now, or
   (c) create a separate packaging story first.

3. **Notarization of raw binaries vs. bundles:** Apple's notarytool requires a .zip,
   .dmg, or .pkg — it does not notarize a bare Mach-O binary directly. For a
   command-line tool distributed via Homebrew, the typical approach is to zip the
   binary before submitting. Confirm: should notarization target a zip of each
   darwin binary, or wait for .pkg/.dmg packaging? The operator should decide before
   the implementer begins. ThreeDoors notarizes .pkg and .dmg; for Homebrew-distributed
   CLI tools, a zip is more common.

4. **Devcontainer interaction:** Signing requires macOS (Apple toolchain). Any
   devcontainer would be Linux-based and cannot run signing locally. Confirm the
   expectation: devcontainer is for Go development only; signing is a macOS-only
   operation (local or CI). Document this boundary clearly in CONTRIBUTING.md.

5. **`harden-runner` egress for Apple APIs:** release.yml uses
   `step-security/harden-runner` with `egress-policy: audit`. Notarytool contacts
   Apple's notarization API; the DeveloperIDG2CA.cer download is from apple.com.
   Verify these URLs are not blocked and add them to the egress allowlist if needed
   (this was flagged in S-M.01 open questions for mise download URLs; same concern
   applies here for Apple's notarization endpoints).

6. **`APPLE_INSTALLER_*` secrets:** The current release.yml imports both an
   Application and an Installer certificate. If .pkg creation is deferred (Q1 above),
   the installer certificate import steps should be made conditional or removed for
   now to avoid CI failure when those secrets are not provisioned. Confirm scope.

## Spec Patches

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-06-26 | Initial draft |
