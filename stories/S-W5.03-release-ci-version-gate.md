---
document_type: story
story_id: S-W5.03
epic_id: E-9
version: "1.0"
status: draft
producer: story-writer
phase: 2
points: 2
depends_on: [S-W5.01]
blocks: []
behavioral_contracts: [BC-2.07.004]
verification_properties: []
tdd_mode: strict
priority: P1
# BC status: pending PO authorship — Ruling S deferral tracking story
# Note: this story targets a CI/devops gap (Ruling S, ARCH-12 v1.5). It touches
# .github/workflows/ and/or Justfile/Makefile CI targets — not internal/mgmt or
# cmd/sbctl. E-9 is the CI/devops epic; if E-9 does not yet exist, create it
# ("release verification and CI hardening"). This story is non-blocking for the
# current wave; it MUST ship before the first tagged release of the management plane.
---

# S-W5.03: Release CI Version Gate — Assert Binary Is Semver Not "dev"

> **Execute:** `/vsdd-factory:deliver-story S-W5.03`

## Scope Note

This story covers a CI/devops gap identified in ARCH-12 v1.5 Ruling S. It is a
**follow-up tracking story** — it does NOT touch `internal/mgmt`, `cmd/sbctl`, or
`cmd/switchboard`. All changes are confined to CI workflow files and/or `Justfile`.

The management plane wires `daemonVersion` (from `cmd/switchboard.version` via
ldflags at build time) into every AUTH_OK response. BC-2.07.004 PC-7 states:
"Hardcoding `"dev"` in production is a defect." There is currently no CI assertion
that a tagged/release build actually injects a semver string. A broken ldflags
injection silently produces a release binary with `daemon_version: "dev"` in every
AUTH_OK — invisible until an operator notices the wrong version string.

**Scope in:** `.github/workflows/` (release or release-verify workflow); `Justfile`
(dev build vs. release build targets).

**Scope out:** `internal/mgmt`, `cmd/sbctl`, `cmd/switchboard`.

## Behavioral Contracts

| BC | Title | PCs covered |
|----|-------|------------|
| BC-2.07.004 | Daemon Management Server Authenticates All Connections via Ed25519 Challenge-Response (Fail-Closed) | PC-7 (AUTH_OK `daemon_version` field equals ldflags-injected semver; `"dev"` is unreleased-build sentinel only; hardcoding `"dev"` in production is a defect) |

## Narrative

- **As a** release engineer
- **I want** CI to reject any tagged release whose binary reports version `"dev"`
- **So that** shipping an unversioned binary is structurally impossible — ldflags
  injection failure is caught before any artifact leaves the build pipeline

## Acceptance Criteria

### AC-001 (traces to BC-2.07.004 PC-7 — Ruling S)
A build produced with a semver tag (e.g., `v1.2.3`) injects the version via ldflags so
that `switchboard --version` (or equivalent version-inspection mechanism) outputs a
semver string, NOT `"dev"`. The ldflags injection target is `cmd/switchboard.version`
(the package-level `var version = "dev"` sentinel). Canonical build command:
```sh
go build -ldflags="-X cmd/switchboard.version=v1.2.3" ./cmd/switchboard/
```
The resulting binary must output a semver string (matching `v\d+\.\d+\.\d+.*`) when
inspected.
- **Test:** `TestReleaseBuild_VersionIsSemver` or equivalent Justfile/CI smoke test —
  build with a test semver tag injected; invoke the binary's version flag or parse
  the build info; assert the version string is NOT `"dev"` and matches the semver
  pattern `v\d+\.\d+\.\d+.*`.

### AC-002 (traces to BC-2.07.004 PC-7 — Ruling S)
The CI pipeline includes a `release-verify` step (in `.github/workflows/release-verify.yml`
or equivalent) that:
1. Builds the binary with release ldflags (using the git tag as the version string).
2. Runs `switchboard --version` (or parses the binary's embedded version via `strings`
   or debug/buildinfo) to extract the version field.
3. Asserts the version string matches `v\d+\.\d+\.\d+.*` and is NOT `"dev"` or empty.
4. Fails the CI run with a clear, human-readable error message if the assertion does not hold.

The `just build` (dev build, no tag) MUST correctly produce `"dev"`. Only the release
build path (`just build-all` or CI release) injects the semver.
- **Test:** CI workflow diff showing the `release-verify` step with the version assertion
  present. Also add a `TestVersionSentinel` in `cmd/switchboard/` that asserts the
  `version` variable is non-empty (guards against removing the default sentinel).

## Architecture Mapping

| Component | Module | Pure/Effectful |
|-----------|--------|---------------|
| CI release workflow | .github/workflows/ | effectful (CI I/O) |
| Justfile release target | Justfile | effectful (shell) |
| `cmd/switchboard.version` sentinel | cmd/switchboard | pure declaration (ldflags injection point) |

## Token Budget Estimate (MANDATORY)

| Context Source | Estimated Tokens |
|---------------|-----------------|
| This story spec | ~800 |
| BC-2.07.004.md (PC-7 section) | ~300 |
| ARCH-12 §Ruling S + §Wiring into cmd/switchboard (version param) | ~400 |
| Existing `.github/workflows/` files | ~500 |
| Existing `Justfile` | ~400 |
| **Total** | **~2,400** |
| Agent context window | 200K |
| **Budget usage** | **~1.2%** |

## Tasks (MANDATORY)

1. [ ] Read BC-2.07.004 PC-7 and ARCH-12 §Ruling S
2. [ ] Read `cmd/switchboard/main.go` (or `version.go`) to locate the `var version = "dev"` sentinel and confirm ldflags injection target
3. [ ] Read `Justfile` to identify existing build targets and the pattern for ldflags injection
4. [ ] Read `.github/workflows/` to identify the release workflow (or absence thereof)
5. [ ] Add or extend `just build-release` (or equivalent) to inject version via ldflags: `-X cmd/switchboard.version=$(git describe --tags --exact-match 2>/dev/null || echo dev)`
6. [ ] Add `release-verify` CI step in `.github/workflows/release-verify.yml` (or extend existing release workflow):
   - Build binary with release ldflags
   - Assert version is NOT `"dev"` and matches semver pattern
   - Fail CI with clear error message on assertion failure
7. [ ] Add `TestVersionSentinel` in `cmd/switchboard/` asserting `version != ""` (guards against removing the sentinel)
8. [ ] `just fmt && just lint` pass

## Previous Story Intelligence (MANDATORY)

| Story | Key Decisions | Patterns Established | Gotchas Discovered |
|-------|--------------|---------------------|-------------------|
| S-W5.01 | `mgmt.NewServer` takes `daemonVersion string`; panics if empty; AUTH_OK includes `daemon_version` field | ldflags injection via `cmd/switchboard.version` | "dev" sentinel is intentional for dev builds — CI must distinguish release vs. dev build explicitly |
| S-6.01 | CI workflow uses `just` for all build/test/lint steps | All CI build steps go through Justfile targets | CI job ordering: lint/test run before release-verify |

## Architecture Compliance Rules (MANDATORY)

| Rule | Source | Enforcement |
|------|--------|-------------|
| `cmd/switchboard.version` MUST be the sole ldflags injection point for daemon version | ARCH-12 §Wiring; BC-2.07.004 PC-7 | `TestVersionSentinel` |
| Release CI step MUST fail if binary version is `"dev"` or empty | BC-2.07.004 PC-7 / Ruling S | `release-verify` CI step |
| Dev build (`just build`) MUST produce `"dev"` sentinel — ldflags injection is release-path only | ARCH-12 §Ruling S note | `TestReleaseBuild_VersionIsSemver` (build with injected semver) |

## Library & Framework Requirements (MANDATORY)

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.25.4 | Per go.mod (mise-pinned) |
| `go build -ldflags` | stdlib build tooling | Inject version at compile time via `-X cmd/switchboard.version=<ver>` |
| `git describe --tags` | git | Derive version string from tag for release builds |
| GitHub Actions | current | CI workflow execution |

## File Structure Requirements (MANDATORY)

| File | Action | Purpose |
|------|--------|---------|
| `Justfile` | modify | Add `build-release` target (or extend existing) with ldflags version injection |
| `.github/workflows/release-verify.yml` | create or modify | Add `release-verify` step: build with ldflags, assert version is semver, fail CI if `"dev"` |
| `cmd/switchboard/version.go` (or `main.go`) | verify/no-change | Confirm `var version = "dev"` sentinel exists as the ldflags injection target |
| `cmd/switchboard/version_test.go` | create or modify | `TestVersionSentinel` — assert `version != ""` |

## Changelog

| Version | Date | Author | Change |
|---------|------|--------|--------|
| 1.0 | 2026-06-29 | story-writer | Initial creation — follow-up tracking story for ARCH-12 v1.5 Ruling S (CI assertion that release binary version is semver, not "dev"; BC-2.07.004 PC-7 enforcement). Targets E-9 CI/devops epic. Non-blocking for current wave; must ship before first tagged management-plane release. |
