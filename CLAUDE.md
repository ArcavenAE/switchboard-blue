# CLAUDE.md — Switchboard

## Project Overview

Switchboard is a Go project.

- **Language:** Go 1.25.4 (per `go.mod`)
- **Module:** `github.com/arcavenae/switchboard`

## Project Structure

```
cmd/switchboard/      # Entry point
internal/             # Internal packages
```

## Development Workflow

**Git workflow:** Gitflow. `develop` is the default branch. Branch from and
PR into `develop`. Alpha releases are cut automatically from `develop`.
Stable releases are cut from `main` via version tags (`v*`). Do not push
directly to `main`.

```bash
just fmt              # gofumpt formatting (run before every commit)
just lint             # golangci-lint — must pass with zero warnings
just test             # go test ./... -v
just test-race        # Race detector — run before pushing
just build            # Build the binary
just build-all        # Build for all release targets
just run              # Build and run
just sign             # Codesign the binary (macOS release artifact)
just verify           # Verify the codesign signature
just test-docker      # Run tests in Docker
just clean            # Remove build artifacts
```

CI workflows (`.github/workflows/`): `ci`, `codeql`, `dependency-review`,
`release`, `release-verify`, `scorecards`.

## Project References

| Path | Purpose |
|------|---------|
| `.factory/STATE.md` | VSDD pipeline state |
| `.factory/specs/` | Architecture, behavioral contracts, PRD supplements, verification properties |
| `.factory/stories/` | Story specs (sharded by spec type) |
| `_bmad-output/` | BMAD planning + brainstorming + implementation artifacts |
| `CONTRIBUTING.md` | Contributing guide |
| `.golangci.yml` | Linter configuration |

@.claude/rules/_index.md
