---
generated: 2026-06-23
mode_detected: greenfield-with-scaffolding
spec_format_detected: BMAD
target_spec_format: VSDD (4-level hierarchy — Brief / L2 / L3 BC-S.SS.NNN / L4 VP)
---

# Artifact Inventory — Switchboard

Scan of the `switchboard-blue` workspace for planning artifacts that exist
prior to VSDD pipeline entry.

## Outside `.factory/` — BMAD planning corpus

| Artifact | Path | Lines | Status |
|---|---|---|---|
| Product Brief | `_bmad-output/planning-artifacts/product-brief-switchboard-2026-03-31.md` | 334 | Substantive — exec summary, problem statement, solution, differentiators, scope phases |
| PRD | `_bmad-output/planning-artifacts/prd.md` | 761 | BMAD-format — named requirement groups under "Functional Requirements" / "Non-Functional Requirements", not enumerated FR-XXX/NFR-XXX |
| PRD Validation Report | `_bmad-output/planning-artifacts/prd-validation-report.md` | 466 | BMAD self-rated 4/5 holistic, overallStatus: Pass (2026-04-04) |
| Epic 0 | `_bmad-output/planning-artifacts/epic-0-project-scaffolding.md` | 61 | Scaffolding epic; contains Story 0.1 |
| Story 0.1 | `_bmad-output/implementation-artifacts/story-0.1.md` | 105 | "Stub Switchboard Binary and Tests" — the current `cmd/switchboard/main.go` is this story's output |
| Brainstorm 001 | `_bmad-output/brainstorming/brainstorming-session-2026-03-07-001.md` | — | Architecture brainstorm session |
| Naming parking lot | `_bmad-output/brainstorming/naming-node-type-parking-lot.md` | — | Open naming questions |
| Session context cache | `_bmad-output/brainstorming/session-context-cache.md` | — | Cross-session continuity notes |

## Outside `.factory/` — KoS knowledge graph

| Artifact | Path | Notes |
|---|---|---|
| KoS bedrock | `_kos/nodes/bedrock/` | 7 architectural bedrock nodes (per commit `c5b4c22`). Established facts with evidence. |
| KoS frontier | `_kos/nodes/frontier/` | Open exploration questions |
| Charter | `charter.md` | Empty template — Bedrock / Frontier / Graveyard sections present but unpopulated. Follows kos Orient → Question → Probe → Harvest → Promote process |

## Inside `.factory/`

| Artifact | Status |
|---|---|
| `STATE.md` | INITIALIZED, phase: pre-1, mode: greenfield |
| `.factory/specs/` | Empty (only `.gitkeep` files in `behavioral-contracts/`, `verification-properties/`, `architecture/`, `prd-supplements/`) |
| `.factory/stories/` | Empty |
| `.factory/cycles/`, `holdout-scenarios/`, `dtu-clones/`, `semport/`, `code-delivery/`, `demo-evidence/` | All empty |

## Source code (potential brownfield signal)

| Path | Lines | Role |
|---|---|---|
| `cmd/switchboard/main.go` | 34 | Stub entry point — output of BMAD Story 0.1 |
| `cmd/switchboard/main_test.go` | 83 | Stub test scaffold |
| `go.mod` | — | Go 1.25.4, module `github.com/arcavenae/switchboard`, no external deps yet (go.sum empty) |
| `internal/` | — | Directory does not yet contain code |

## Tooling / DevOps already provisioned

- `justfile` with `fmt`/`lint`/`test`/`test-race`/`build`/`build-all`/`run`/`sign`/`verify`/`test-docker`/`clean`
- `lefthook.yml` (pre-commit hooks)
- `.golangci.yml` (lint config)
- `Dockerfile.test` + `docker-compose.test.yml`
- CI workflows: `ci`, `codeql`, `dependency-review`, `release`, `release-verify`, `scorecards`
- `.editorconfig`, `.gitattributes`, `.dockerignore`, `.env.local.example`
- `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`, `LICENSE`, `README.md`
- `.claude/rules/` (commit + bash + go style guides)
- `.tool-versions` (asdf/mise version pins)
