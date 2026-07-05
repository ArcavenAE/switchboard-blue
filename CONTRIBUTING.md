# Contributing to Switchboard

Thank you for your interest in contributing! This guide will help you get started.

## Development Model

This project uses a **solo maintainer + AI agent team** development model. The human maintainer (arcaven) directs a team of AI agents that handle most implementation work. All PRs are reviewed by the maintainer before merge.

You don't need to use AI agents to contribute -- just follow this guide and submit a PR like any other open-source project.

## Prerequisites

- **Go 1.22+** -- [install](https://go.dev/doc/install)
- **just** -- [install](https://github.com/casey/just)
- **gofumpt** -- `go install mvdan.cc/gofumpt@latest`
- **golangci-lint** -- [install](https://golangci-lint.run/welcome/install/)

## Getting Started

```bash
git clone https://github.com/ArcavenAE/switchboard.git
cd switchboard
just build    # Build
just test     # Run tests
just lint     # Run linter (golangci-lint)
just fmt      # Format code (gofumpt)
```

## How to Contribute

### Reporting Bugs

Open a [Bug Report](https://github.com/ArcavenAE/switchboard/issues/new?template=bug-report.yml) and include:

- Version or commit hash
- Operating system
- Steps to reproduce
- Expected vs actual behavior

### Suggesting Features

Open a [Feature Request](https://github.com/ArcavenAE/switchboard/issues/new?template=feature-request.yml).

This project is part of the [Arcaven Agentic Engineering](https://github.com/ArcavenAE) platform. Features should align with the project's design values: user sovereignty, composability over frameworks, and gradual elaboration.

### Submitting Code

1. Fork the repo and create a feature branch from `develop`:
   ```bash
   git checkout -b feat/your-feature
   ```
2. Write tests for your changes
3. Run the full quality gate:
   ```bash
   just fmt
   just lint
   just test
   just smoke-quick
   ```
4. Create a PR using the PR template — fill in the **Blast Radius** block (see below).

### Blast radius

Every PR body carries a `## Blast Radius` section with three answers. A
CI check (`Blast Radius / Declaration present`) fails the PR if the
section is missing or every answer is empty. Copy the template from
[`.github/PULL_REQUEST_TEMPLATE.md`](.github/PULL_REQUEST_TEMPLATE.md).

**Why this exists.** On 2026-07-04 a tutorial-walk smoke caught four
operator-boundary regressions (`--help` printing a diagnostic to stderr,
`sbctl --version` flag missing, version banner hard-coded to a literal,
sbctl-a packaged with ldflags unwired) in a single afternoon. Each had
shipped through the quality gate; each was a "one-line change" that
looked mechanical in review. The sentinel smoke gate
([`test/smoke/invariants.sh`](test/smoke/invariants.sh)) now guards the
specific defect classes we know about. This block guards the class of
regressions we don't yet know about.

**The three prompts, and how to answer each:**

1. **Operator-visible surfaces touched.** Anything a user, operator, or
   downstream automation reads by name — CLI flags, subcommand output,
   `--help`/`--version` banners, config-file schemas, wire frame layouts,
   error taxonomy strings, log format, path metric emissions,
   `docs/getting-started.md` steps. Answer "none" only if this is a
   truly internal refactor with no reachable behaviour change.

2. **Silent-failure risk.** Could this ship a defect the current test
   suite does NOT catch? Cite the classes of regression that would slip
   through — e.g. *"banner reads 'dev' in packaged binary because
   ldflags not wired to the release recipe"*, *"help prints to stderr
   with exit 1 instead of stdout with exit 0"*, *"`sbctl <sub> --help`
   opens a socket before parsing"*. Answer "none" only if every
   reachable defect class is unit-covered.

3. **Smoke gate touched.** Does this PR add, change, or need a NEW
   sentinel in `test/smoke/invariants.sh`? If yes, cite the `INV-*` id
   and confirm the paired `docs/architecture.md §Smoke invariants` row
   is included in this diff. If no, say "no."

**What good answers look like.** "This changes `sbctl paths list` output
to add an `rtt_p95_ms` column. Silent-failure risk: existing tests
cover the column names but not the emission format for pending metrics
(BC-2.06.003 EC-003 sentinel `pending`). Adds INV-9 covering `sbctl
paths list --format=json` shape; paired docs row included." — three
sentences, three specific answers, no theatre.

**What bad answers look like.** "*TBD*." "*none*" across all three
prompts on a PR that changes a wire frame. "*just a small refactor*"
followed by 400 lines of diff touching admission code. The reviewer
will ask you to redo the block. The bot will ask you to redo the block.

**When "none" is the right answer.** Renaming an internal-only helper.
Rearranging test setup. Bumping a dev-only dependency. Editing a
comment. Adding a `_kos/` node or `_bmad-output/` artifact. If the code
you're touching cannot be reached by an operator command or a wire
peer, "none" is honest.

### Commit Message Format

[Conventional Commits](https://www.conventionalcommits.org/):

```
type(scope): description
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`

Examples:
- `feat: add session timeout configuration`
- `fix: prevent crash on empty input`
- `docs: update installation guide`

Rules:
- Imperative, present tense ("add feature" not "added feature")
- No capitalized first letter in description
- No period at end

## Code Standards

- **Formatting:** gofumpt (stricter than gofmt)
- **Linting:** golangci-lint with zero warnings
- **Testing:** Table-driven tests using stdlib `testing` (no testify)
- **Error handling:** Always check errors, wrap with `%w` for context
- **No `init()` functions** -- pass dependencies explicitly
- **Timestamps:** Always `time.Now().UTC()`

See [CLAUDE.md](CLAUDE.md) for the complete coding standards.

## What NOT to Contribute

- Heavy dependencies where the standard library suffices
- Telemetry, analytics, or phone-home features
- Features that create vendor lock-in or external service dependencies
- Code that stores or manages credentials (auth is always delegated)
- Testify or other test framework dependencies

## License

This project is [MIT licensed](LICENSE). By contributing, you agree that your contributions will be licensed under the same license.
