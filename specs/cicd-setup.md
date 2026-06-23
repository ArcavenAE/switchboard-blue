---
artifact_id: cicd-setup
status: draft
producer: devops-engineer
timestamp: 2026-06-23T00:00:00
phase: 1
inputDocuments:
  - '.github/workflows/ci.yml'
  - '.github/workflows/codeql.yml'
  - '.github/workflows/dependency-review.yml'
  - '.github/workflows/release.yml'
  - '.github/workflows/release-verify.yml'
  - '.github/workflows/scorecards.yml'
  - 'justfile'
  - 'lefthook.yml'
  - '.golangci.yml'
  - '.factory/specs/architecture/ARCH-10-tooling-selection.md'
---

# CI/CD Setup — Verification Report

Repository: `ArcavenAE/switchboard-blue`
Verified: 2026-06-23
Repo module: `github.com/arcavenae/switchboard`
Language: Go 1.25.4
Branching strategy: Gitflow (`develop` default, `main` stable)

---

## 1. Workflow Inventory

| File | Trigger | Purpose | Key Steps |
|------|---------|---------|-----------|
| `ci.yml` | push → `develop`; PR → `develop`, `main` | Main quality gate + alpha release | gofumpt check, go vet, golangci-lint, go test -race -coverprofile, coverage floor (75%), go build, cross-compile (4 targets), optional Apple sign/notarize, create pre-release, update homebrew tap |
| `codeql.yml` | push → `develop`/`main`; PR → `develop`/`main`; weekly Mon 00:00 UTC | SAST via CodeQL | checkout, setup-go, CodeQL init + autobuild + analyze (Go) |
| `dependency-review.yml` | PR (all) | Scan dependency manifest changes for known-vulnerable versions | harden-runner, checkout, dependency-review-action |
| `release.yml` | push tag `v*` | Stable release build | gofumpt, go vet, golangci-lint, go test -race, cross-compile (4 targets), optional Apple sign/notarize/pkg/dmg/notarytool, create GitHub Release, update homebrew tap |
| `release-verify.yml` | workflow_run: Release or CI completed | Verify homebrew-tap CI passed after formula push | check tap CI status via gh run list |
| `scorecards.yml` | branch_protection_rule event; weekly Tue 07:20 UTC; push → `develop` | OSSF Scorecard supply-chain analysis | scorecard-action, upload SARIF to code-scanning |

**Total workflows: 6**

---

## 2. Branch Protection Check

Branch protection was queried via:
```
gh api repos/ArcavenAE/switchboard-blue/branches/develop/protection
gh api repos/ArcavenAE/switchboard-blue/branches/main/protection
```

**Result: Both branches returned HTTP 404 — "Branch not protected".**

Neither `develop` nor `main` has any branch protection rules configured.

| Branch | Protected | Required status checks | Required reviewers | Dismiss stale reviews | Require signed commits | Restrict push | Allow force push | Allow deletions |
|--------|-----------|----------------------|-------------------|----------------------|----------------------|--------------|-----------------|----------------|
| `develop` | NO | — | — | — | — | — | — | — |
| `main` | NO | — | — | — | — | — | — | — |

This means:
- Anyone with write access can push directly to `develop` or `main` — bypassing CI entirely.
- No status checks are required for PRs to merge.
- Force-push to protected branches is not blocked.
- Commit signing is not enforced at the repo level (GitHub-side).

---

## 3. Branch Strategy Alignment

CLAUDE.md declares: "Gitflow. `develop` is the default branch. Branch from and PR into `develop`. Alpha releases are cut automatically from `develop`. Stable releases are cut from `main` via version tags (`v*`). Do not push directly to `main`."

The user's global personal rules (global CLAUDE.md) state: "NEVER push directly to `main` or `develop` branches. Always create a feature branch and open a PR."

The CI workflow files are correctly scoped:
- `ci.yml` triggers on push to `develop` (alpha release path) and PRs to `develop`/`main`.
- `release.yml` triggers on `v*` tags (stable release path).
- `codeql.yml` covers both `develop` and `main`.

However, because neither branch has protection rules, the gitflow discipline is enforced only by convention — not by GitHub. A push directly to `develop` or `main` would succeed and would trigger CI only in the `ci.yml` push path (for `develop`), not as a required gate.

**Finding:** Branch strategy is architecturally correct in workflow design but entirely unenforced at the repository level. See Gaps section.

---

## 4. Quality Gates Blocking Merge

Because branch protection is absent, **no workflow checks are currently required for PR merge**. The following checks _run_ on PRs but are not enforced:

| Check | Workflow | Runs on PR? | Currently required to merge? |
|-------|----------|-------------|------------------------------|
| gofumpt format check | `ci.yml` / quality-gate | yes | NO (no branch protection) |
| go vet | `ci.yml` / quality-gate | yes | NO |
| golangci-lint | `ci.yml` / quality-gate | yes | NO |
| go test -race + coverage ≥ 75% | `ci.yml` / quality-gate | yes | NO |
| go build | `ci.yml` / quality-gate | yes | NO |
| CodeQL | `codeql.yml` | yes | NO |
| dependency-review | `dependency-review.yml` | yes | NO |

Minimum required per VSDD Phase 3 readiness: `CI / Quality Gate` (test + lint + race detector + coverage floor). This job exists and is correct — it just needs to become a required status check on `develop`.

---

## 5. Signing Enforcement

### Git commit signing (SSH/GPG)

The global gitconfig enforces `commit.gpgsign = true` locally. This is a local developer constraint and cannot be verified from the workflow files alone.

Branch protection `required_signatures` (GitHub's commit signature requirement) is **not configured** because branch protection itself is absent. GitHub's signing enforcement requires branch protection to be enabled first, then `required_signatures: true` to be set on the protected branch.

**Finding:** Commit signing is enforced locally via global gitconfig but is not enforced at the GitHub repository level. Any contributor without the local gitconfig rule can push unsigned commits.

### Binary signing (Apple Developer ID)

`ci.yml` and `release.yml` both have a `sign-and-notarize` job that runs when `vars.SIGNING_ENABLED == 'true'` is set on the `release` environment. The job uses hardened runtime, codesign, Apple notarytool, and produces both `.dmg` and `.pkg` artifacts. Signing secrets are referenced via `${{ secrets.APPLE_* }}` — no credentials are hardcoded.

---

## 6. ARCH-10 Verification Toolchain Coverage

ARCH-10 specifies the following verification toolchain. This section maps each tool against current CI coverage.

| Tool | ARCH-10 requirement | ci.yml covers it? | Trigger | Gap? |
|------|--------------------|--------------------|---------|------|
| `go test ./...` | every commit, blocks merge | YES — `go test ./... -v -count=1 -race -coverprofile=coverage.out` | push + PR | none |
| `go test -race ./...` | every commit, blocks merge | YES — `-race` flag present in quality-gate job | push + PR | none |
| `golangci-lint` | every commit, blocks merge | YES — step-security/golangci-lint-action | push + PR | none |
| `go vet` | every commit, blocks merge | YES — explicit `go vet ./...` step | push + PR | none |
| `gofumpt` | every commit, blocks merge | YES — `gofumpt -l .` check | push + PR | none |
| `staticcheck` | via golangci-lint | YES — `.golangci.yml` enables `staticcheck` | push + PR | none |
| `go test -fuzz` | nightly CI job, blocks nightly gate | NO — no fuzz CI job exists | — | P1 gap |
| `gopter` (property tests) | every commit (pure-core packages) | NOT PRESENT — no gopter dependency or test yet | — | P2 gap (pre-Phase 3) |
| `go-mutesting` | Phase 5 gate | NO nightly/phase-5 CI job — Phase 5 scope | — | P2 gap (Phase 5, not Phase 3) |
| `benchstat` regression check | PR against main, warning | NO — no bench CI job | — | P2 gap |

**Coverage floor**: `ci.yml` enforces `COVERAGE_THRESHOLD=75`. This is not mentioned in ARCH-10 but is a stronger gate than ARCH-10 requires.

**golangci-lint linter gap**: `.golangci.yml` enables `errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`, `misspell`, `unconvert`, `unparam`. ARCH-10 specifically calls for `gosec`, `revive`, `bodyclose`. These three are **not enabled** in the current `.golangci.yml`. This means security anti-patterns (weak crypto, blocked imports per `gosec`) are not caught by CI despite ARCH-10 requiring them.

---

## 7. Gaps and Recommendations

### P0 — Blocks Phase 3 (TDD Implementation)

| ID | Gap | Detail |
|----|-----|--------|
| P0-001 | **No branch protection on `develop`** | `ci.yml` quality-gate runs but nothing requires it to pass before merge. A PR can be merged with a failing test suite. Phase 3 requires enforced red-green discipline: merging broken code is not acceptable. Fix: configure branch protection on `develop` with `CI / Quality Gate` as a required status check. |
| P0-002 | **No branch protection on `main`** | Stable releases cut from `main` have no protection. A force-push to `main` would corrupt the release history. Fix: configure branch protection on `main` with the same required checks plus `Restrict who can push`. |
| P0-003 | **Commit signature enforcement missing at repo level** | Global gitconfig enforces signing locally, but GitHub does not reject unsigned commits. Any CI bot, dependabot merge, or contributor without the local rule can push unsigned commits to protected branches once protection is configured. Fix: after enabling branch protection, set `required_signatures: true` on both `develop` and `main`. |

### P1 — Should fix before Phase 5 (Formal Hardening)

| ID | Gap | Detail |
|----|-----|--------|
| P1-001 | **No nightly fuzz CI job** | ARCH-10 specifies `go test -fuzz=... -fuzztime=300s` as a nightly gate that blocks nightly. No workflow implements this. Add a scheduled workflow running fuzz targets for `frame`, `hmac`, `admission`, `config` packages. |
| P1-002 | **`gosec`, `revive`, `bodyclose` linters missing** | ARCH-10 names these as required. `.golangci.yml` does not enable them. `gosec` in particular is required to catch `math/rand` vs `crypto/rand` misuse in security packages. Fix: add these three linters to `.golangci.yml`. |

### P2 — Nice to have / Phase 5+ scope

| ID | Gap | Detail |
|----|-----|--------|
| P2-001 | **No benchstat regression CI job** | ARCH-10 defines benchmarks for `BenchmarkHalfChannelTick`, `BenchmarkFrameEncode`, etc. with explicit regression thresholds. No CI job runs `just bench` + `benchstat`. Add a benchmark job on PRs to `main` (warning, not hard block per ARCH-10). |
| P2-002 | **No go-mutesting CI job** | ARCH-10 places mutation testing at Phase 5 gate, so this is not blocking Phase 3. When Phase 5 begins, add a `just muttest` job (or equivalent) as a manual/phase gate trigger. |
| P2-003 | **gopter property tests not yet present** | No test files use `gopter` yet. This is expected pre-Phase 3. Test-writer agents will add them. No CI change needed until the tests exist. |
| P2-004 | **`release-verify.yml` is advisory only** | The homebrew-tap check uses `::warning::` not `exit 1` for tap CI failure. If the tap formula is broken, no pipeline is blocked. Consider upgrading to a hard failure for stable releases. |
| P2-005 | **`dependency-review.yml` has no `fail-on-severity` config** | The workflow uses defaults. Explicitly setting `fail-on-severity: moderate` would prevent merging PRs that introduce medium+ CVEs. Low effort, high value. |

---

## 8. Summary

| Category | Count | Notes |
|----------|-------|-------|
| Workflows | 6 | All pinned to SHA, all use step-security/harden-runner |
| Jobs with required-check status | 0 | Branch protection absent — none are enforced |
| P0 gaps (block Phase 3) | 3 | All relate to branch protection and signing enforcement |
| P1 gaps | 2 | Fuzz CI job missing; gosec/revive/bodyclose linters missing |
| P2 gaps | 5 | Bench CI, mutation CI (Phase 5), gopter (Phase 3 TDD), advisory hardening |

**Branch protection on `develop`:** absent — does not match the user's global push restriction rules.
**Branch protection on `main`:** absent — does not match the user's global push restriction rules.

The existing workflow files are well-structured, correctly pinned, and exercising the right gates (`go test -race`, `golangci-lint`, `gofumpt`, coverage floor). The sole blocker for Phase 3 is that none of these gates are _required_ — they are advisory. Configuring branch protection resolves P0-001 and P0-002; adding `required_signatures` resolves P0-003.

No workflow files were modified during this verification pass. Gaps are documented here for resolution via dedicated PRs.
