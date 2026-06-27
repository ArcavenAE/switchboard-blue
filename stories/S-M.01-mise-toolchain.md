---
artifact_id: S-M.01-mise-toolchain
document_type: story
level: ops
story_id: S-M.01
title: "migrate toolchain provisioning from Homebrew to mise"
status: draft
producer: story-writer
timestamp: 2026-06-26T00:00:00Z
phase: 2
epic: E-MAINT
wave: null
# wave: unscheduled — maintenance epic, NOT part of feature waves 1-7.
# Orchestrator should slot this into a maintenance/DX wave after Wave 7
# or as a standalone maintenance sweep, independent of feature delivery.
priority: P2
scope_phase: M
estimated_points: 5
tdd_mode: facade
# tdd_mode: facade — this story delivers config files and doc edits, not
# Go application logic. There is no meaningful "test first" loop for
# .mise.toml, workflow YAML edits, or CONTRIBUTING.md updates. Quality
# gate is: `mise install` reproduces the dev env; CI stays green.
behavioral_contracts: []
# BC status: pending PO authorship
# This is toolchain/DX infrastructure — no product behavioral contracts
# apply. Status remains draft until an owner decides whether to promote
# or treats the ACs below as sufficient governance.
verification_properties: []
depends_on: []
blocks: []
subsystems: []
architecture_modules: []
cycle: v1.0.0-greenfield
inputDocuments:
  - '.github/workflows/ci.yml'
  - 'lefthook.yml'
  - 'go.mod'
  - '.golangci.yml'
  - 'CONTRIBUTING.md'
  - 'CLAUDE.md'
---

# S-M.01: Migrate Toolchain Provisioning from Homebrew to mise

> **Note:** This is a maintenance/DX story with no product behavioral contract anchor.
> Execute outside of feature waves. Suggested trigger: `just maintenance` sweep or
> manual orchestrator dispatch.

## Narrative

- **As a** contributor or AI agent setting up the switchboard dev environment
- **I want to** run `mise install` at the repo root and have the full toolchain
  (Go, gofumpt, golangci-lint, just, lefthook) available at pinned versions
- **So that** environment setup is reproducible, version-controlled, and
  decoupled from the operator's global Homebrew state; CI and local dev use
  the same single source of truth for tool versions

## Acceptance Criteria

### AC-001 — mise.toml at repo root pins the full toolchain
A `mise.toml` (or `.mise.toml`) file exists at the repository root. It pins, at
minimum, the following tools at the versions currently in use:

| Tool | Version to pin | Source of truth |
|------|---------------|-----------------|
| Go | 1.25.4 | `go.mod` |
| gofumpt | 0.7.0 | `.github/workflows/ci.yml` (`go install mvdan.cc/gofumpt@v0.7.0`) |
| golangci-lint | latest v2.x compatible with `.golangci.yml` schema v2 | CI action `step-security/golangci-lint-action@v9.2.0` |
| just | current stable | `justfile` usage |
| lefthook | current stable | `lefthook.yml` usage |

Running `mise install` in a fresh clone reproduces a working dev environment.
Running `mise exec -- just fmt && mise exec -- just lint && mise exec -- just test`
passes without errors.

### AC-002 — single source of truth: CI tool versions match mise.toml
All GitHub Actions workflows that currently provision Go or dev tools
(`.github/workflows/ci.yml`, `.github/workflows/release.yml`, and any others
that call `actions/setup-go` or `go install`) are updated to either:

(a) use `jdx/mise-action` to install tools from `mise.toml`, so versions
    are read from one file, or
(b) retain the existing `actions/setup-go` + `go install` pattern but have
    a documented rationale explaining why CI cannot use mise (e.g., runner
    constraints), with a note that `mise.toml` remains the canonical version
    reference and CI values must be kept manually in sync.

Option (a) is preferred. In either case, CI must stay green on all platforms
(ubuntu-latest, macos-latest) and the tool versions in CI must match `mise.toml`.

### AC-003 — lefthook hooks resolve tools via mise
`lefthook.yml` is updated so that pre-commit and pre-push hooks invoke
`gofumpt` and `golangci-lint` through mise shims (e.g., via `mise exec --`
prefix or by ensuring mise's shim directory is on PATH when lefthook runs).
Running `lefthook install && git commit` in a repo set up via `mise install`
invokes the pinned tool versions, not any globally-installed Homebrew versions.

### AC-004 — CONTRIBUTING.md updated: mise-first setup instructions
`CONTRIBUTING.md` Prerequisites section is updated to replace the Homebrew-
centric instructions with mise-first instructions:

```
Prerequisites:
- mise — https://mise.jdx.dev/getting-started.html
  Run `mise install` in the repo root to get Go, gofumpt, golangci-lint,
  just, and lefthook at the pinned versions.
```

The existing `go install mvdan.cc/gofumpt@latest` and ad-hoc
`golangci-lint` install instructions are removed or replaced. The `just`
workflow commands remain unchanged.

### AC-005 — CLAUDE.md operator preference updated
`CLAUDE.md` (project-level) is updated to replace the "prefer brew" standing
directive with "prefer mise" for tool/runtime management, consistent with the
operator's global standing preference shift. The update is scoped to the
project-level `CLAUDE.md`; the global `~/.claude/CLAUDE.md` is out of scope.

### AC-006 — migration/back-compat note for Homebrew contributors
A brief migration note is added to `CONTRIBUTING.md` (or a `docs/mise-migration.md`
if the maintainer prefers to keep CONTRIBUTING.md short). The note must cover:

1. mise and Homebrew can coexist; mise manages per-repo tool versions via
   `.tool-versions` / `mise.toml` without touching globally-installed Homebrew tools.
2. Contributors who installed Go/gofumpt/golangci-lint via Homebrew can keep
   their global installs; `mise install` installs a parallel set under `~/.local/share/mise/`.
3. How to activate mise shims in the shell (`eval "$(mise activate zsh)"` or
   equivalent for bash/fish) so `just fmt` / `just lint` pick up the repo-pinned
   versions automatically.

### AC-007 — devcontainer (optional, best-effort)
No devcontainer currently exists. If one is added as part of this story or a
companion story, it should install tools via mise rather than apt/brew. This AC
is **optional** and explicitly out of scope for the minimum-viable delivery of
this story; the maintainer should decide whether to bundle devcontainer creation
here or create a separate story (`S-MAINT-02-devcontainer`). Document the decision
in the Spec Patches table.

## Non-Goals

- No changes to application code (`cmd/`, `internal/`), `go.mod` module path,
  or build output semantics.
- No changes to release codesign/notarize workflows beyond tool-provisioning
  steps (the Apple toolchain — `codesign`, `xcrun`, `security` — is macOS-native
  and is NOT managed by mise).
- No removal of the Homebrew tap or `Formula/switchboard.rb` — those are for
  distributing the switchboard binary to users, not for developer toolchain setup.
- No changes to `just` recipe semantics; all existing `just <target>` commands
  must continue to work unchanged.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Contributor has a globally-installed Go version different from 1.25.4 | mise activates the pinned 1.25.4 in repo context; `go version` inside the repo returns 1.25.4 |
| EC-002 | CI runner cache invalidated after mise.toml update | `mise install` re-downloads pinned versions; no stale-tool breakage |
| EC-003 | golangci-lint major version boundary (v1 vs v2) | `.golangci.yml` uses schema `version: "2"` which requires golangci-lint v2.x; mise pin must target v2.x, not v1.x |
| EC-004 | mise not installed on contributor machine | Error message from `mise install` is clear; CONTRIBUTING.md install link is present |
| EC-005 | lefthook runs before mise shims are on PATH | Document the `mise activate` shell-hook requirement; consider adding a `mise exec --` wrapper in lefthook.yml |
| EC-006 | release.yml or scorecards.yml also installs Go | Audit all workflows; ensure all Go version references point to `mise.toml` or remain in sync |

## Architecture Mapping

| Component | Location | Pure/Effectful |
|-----------|----------|---------------|
| mise.toml | repo root | config (pure data) |
| .github/workflows/ci.yml | CI provisioning step | effectful (CI runner) |
| lefthook.yml | pre-commit/pre-push hooks | effectful (shell exec) |
| CONTRIBUTING.md | docs | pure data |
| CLAUDE.md (project) | operator directives | pure data |

## Token Budget Estimate

| Context Source | Estimated Tokens |
|----------------|-----------------|
| This story spec | ~1,200 |
| .github/workflows/ci.yml | ~800 |
| .github/workflows/release.yml | ~800 |
| lefthook.yml | ~200 |
| CONTRIBUTING.md | ~300 |
| CLAUDE.md (project) | ~200 |
| mise docs (jdx/mise-action README) | ~400 |
| Tool outputs overhead | ~200 |
| **Total** | **~4,100** |
| Agent context window | 200K |
| **Budget usage** | **~2.1%** |

## Tasks

1. [ ] Read current `.github/workflows/ci.yml`, `release.yml`, `codeql.yml`,
       `dependency-review.yml`, `scorecards.yml` — identify all places that
       install Go or dev tools
2. [ ] Determine golangci-lint v2.x version compatible with `step-security/golangci-lint-action@v9.2.0`
       (the action pins an internal version; cross-reference mise-installable release)
3. [ ] Create `mise.toml` at repo root with [tools] section pinning Go 1.25.4,
       gofumpt 0.7.0, golangci-lint v2.x, just, lefthook
4. [ ] Update `.github/workflows/ci.yml` quality-gate job: replace `actions/setup-go`
       + `go install mvdan.cc/gofumpt@v0.7.0` with `jdx/mise-action` (or document
       rationale for Option (b) per AC-002)
5. [ ] Audit remaining workflows for Go/tool provisioning and apply consistent approach
6. [ ] Update `lefthook.yml` to invoke tools via mise exec or rely on mise shims on PATH
7. [ ] Update `CONTRIBUTING.md` Prerequisites section (AC-004)
8. [ ] Update project `CLAUDE.md` to replace "prefer brew" with "prefer mise" (AC-005)
9. [ ] Add migration/back-compat note to `CONTRIBUTING.md` (AC-006)
10. [ ] Decide devcontainer scope: add `S-MAINT-02-devcontainer` story or bundle here (AC-007)
11. [ ] Run `mise install && just fmt && just lint && just test` locally — confirm green
12. [ ] Open PR; verify CI passes on ubuntu-latest and (if applicable) macos-latest

## Previous Story Intelligence

N/A — first story in the maintenance epic. No predecessor intelligence to carry forward.

## Architecture Compliance Rules

| Rule | Source | Enforcement |
|------|--------|-------------|
| No application code changes | Non-Goals section | PR reviewer: diff must touch only config/doc files and .github/workflows |
| Go version in mise.toml must match go.mod | go.mod `go 1.25.4` | CI `go version` assertion or `mise exec -- go version` check |
| golangci-lint version must use schema v2 | `.golangci.yml` `version: "2"` | `just lint` must pass after change |
| codesign/notarize steps in release.yml are out of scope | Non-Goals | Do not modify Apple-toolchain steps |

## Library & Framework Requirements

| Tool | Version | Purpose |
|------|---------|---------|
| mise | latest stable (>=2024.x) | Toolchain version manager |
| jdx/mise-action | latest | GitHub Actions integration for mise |
| Go | 1.25.4 | Must match `go.mod` exactly |
| gofumpt | 0.7.0 | Must match `go install mvdan.cc/gofumpt@v0.7.0` in current CI |
| golangci-lint | v2.x (exact patch TBD by implementer) | Must be compatible with `.golangci.yml` schema `version: "2"` |
| just | current stable | No version constraint; stable release sufficient |
| lefthook | current stable | No version constraint; stable release sufficient |

## File Structure Requirements

| File | Action | Purpose |
|------|--------|---------|
| `mise.toml` | create | Canonical tool version pins for the repo |
| `.github/workflows/ci.yml` | modify | Replace Go/tool provisioning with mise-action |
| `.github/workflows/release.yml` | modify (if applicable) | Align Go version reference with mise.toml |
| `.github/workflows/codeql.yml` | inspect/modify if needed | Ensure Go version sourced from mise.toml or go.mod |
| `lefthook.yml` | modify | Invoke tools through mise shims |
| `CONTRIBUTING.md` | modify | mise-first prerequisites, migration note |
| `CLAUDE.md` (project root) | modify | Update "prefer brew" → "prefer mise" |

## Open Questions for Orchestrator

1. **Story ID scheme:** The existing stories follow `S-N.NN` (feature) or `S-BL.XX`
   (backlog) patterns. No maintenance story ID scheme is established. This story uses
   `S-MAINT-01` as a placeholder. The orchestrator should reconcile this with the
   STORY-INDEX and epic registry, or establish a canonical maintenance prefix
   (e.g., `S-M.01`, `S-DX.01`).

2. **golangci-lint exact version:** The CI uses `step-security/golangci-lint-action@v9.2.0`
   which bundles an internal golangci-lint binary. The implementer must verify the
   exact golangci-lint version that action bundles (or the latest v2.x release) and
   pin it in `mise.toml`. The story cannot pre-fill this without running the action.

3. **devcontainer scope:** AC-007 is explicitly optional. Should devcontainer creation
   be bundled into this story or tracked as `S-MAINT-02`? Recommendation: separate story,
   since devcontainer work is independently valuable and has different reviewers/testing.

4. **Wave placement:** This story has no feature-wave dependencies. It should be
   placed in a maintenance wave or run as a standalone sweep, explicitly NOT in
   Waves 1-7 of the current greenfield cycle.

5. **`jdx/mise-action` in CI:** The action requires internet access to download
   tool binaries. The current CI uses `step-security/harden-runner` with
   `egress-policy: audit`. Verify that mise's download URLs are not blocked and
   add them to the egress allowlist if required.

## Spec Patches

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-06-26 | Initial draft |
