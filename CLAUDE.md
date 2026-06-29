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

## How to Work Here (kos Process)

### Re-introduction
Read charter.md before any substantive work. It contains:
- Current bedrock (what's committed)
- Current frontier (what's under exploration)
- Current graveyard (what's been ruled out)

### Session Protocol
1. Read charter.md (orient)
2. Identify the highest-value open question — or capture new ideas in _kos/ideas/
3. Write an Exploration Brief in _kos/probes/
4. Do the probe work
5. Write a finding in _kos/findings/
6. Harvest: update affected NODES (`_kos/nodes/{bedrock,frontier,graveyard}/*.yaml`),
   move files if confidence changed. Charter is renderer output (per orc F22,
   `kos charter render`); do NOT hand-edit charter prose outside
   `<!-- backdrop -->` blocks. Subrepo charter renderer extension tracked
   in aae-orc-gezz.

Cross-repo questions belong in the orchestrator's _kos/, not here.

### Ideas (pre-hypothesis brainstorming)
Ideas live in _kos/ideas/ as markdown files. Generative, possibly contradictory,
no commitment. When an idea crystallizes, extract into a frontier question + brief.

### Node Files
Nodes live in _kos/nodes/[confidence]/[id].yaml
Schema follows kos schema v0.3.
One node per file. Filename = node id.

### Confidence Changes
Moving a file between confidence directories IS the promotion.
Always accompany with a commit message explaining the evidence.

### Harvest Verification
Before starting the next cycle, verify:
- [ ] Finding written and committed
- [ ] Bedrock/frontier/graveyard NODES updated if state changed —
      edit `_kos/nodes/{bedrock,frontier,graveyard}/*.yaml`, NOT charter
      prose. Charter is renderer output (per orc F22,
      `brief-charter-as-projection-renderer.md`, `kos charter render`).
      Subrepo extension tracked in aae-orc-gezz; until it ships, treat
      charter sections outside `<!-- backdrop -->` blocks as read-only
      and edit the underlying nodes.
- [ ] Frontier questions updated (closed, opened, or revised)
- [ ] Exploration briefs marked complete or carried forward
