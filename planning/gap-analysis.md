---
generated: 2026-06-23
readiness_level: L2-equivalent-but-format-mismatched
target_level: L4 (full VSDD spec)
---

# Gap Analysis — Switchboard

## Readiness classification

Per the VSDD artifact-detection readiness table:

| Level | Required artifacts | Switchboard status |
|---|---|---|
| L0 — Nothing | — | ❌ has substantial planning corpus |
| L1 — Brief | Product brief | ✅ BMAD brief is substantive |
| L2 — PRD | PRD + L2 domain spec | ⚠️ BMAD PRD exists but uses BMAD-format requirement groups, not VSDD L2/L3 enumerated artifacts |
| L3 — PRD + Architecture | + ARCH-INDEX.md + 7 section files + L3 BC-S.SS.NNN behavioral contracts | ❌ no VSDD-format architecture or behavioral contracts |
| L4 — Full Spec | + L4 verification properties + stories | ❌ no L4 verification properties; no `.factory/stories/` content |

**Effective readiness:** between L1 and L2 — there is rich source material
(brief + BMAD PRD + brainstorms + KoS bedrock), but **none of it is in VSDD
4-level format** (L2 domain spec / L3 BC-S.SS.NNN / L4 VP-NNN), and none
of it lives under `.factory/specs/`.

## Artifact-by-artifact validation

### Brief — `_bmad-output/planning-artifacts/product-brief-switchboard-2026-03-31.md`

| Check | Result |
|---|---|
| Has all required sections | ✅ Executive Summary, Problem Statement, Why Now, Proposed Solution, Key Differentiators, Scope, User Journeys (referenced) |
| Substantive content | ✅ 334 lines, detailed |
| Scope is defined | ✅ Three scope phases (E Router Release, Multi-Path/Multi-Hop, Global Topology) |
| **VSDD location** | ❌ lives in `_bmad-output/`, not `.factory/specs/` |

**Status:** VALID content, location mismatch — needs to be either (a) copied/transformed into `.factory/specs/product-brief.md` or (b) referenced via `inputDocuments` frontmatter from the VSDD artifacts that consume it.

### PRD — `_bmad-output/planning-artifacts/prd.md`

| Check | Result |
|---|---|
| Has numbered FR-NNN requirements | ❌ uses named requirement groups (Session Networking, Multi-Path Forwarding, Session Discovery & Presence, …) under `## Functional Requirements`. Zero `### FR-NNN` entries. |
| Has numbered NFR-NNN requirements | ❌ same pattern — named subsections under `## Non-Functional Requirements` |
| Has measurable success criteria | ✅ User Success / Operational Success / Security Success / Death Conditions / Leading Indicators present |
| Edge case catalog | ⚠️ partial — Risk Mitigations + Open Questions sections, but no enumerated edge-case catalog |
| Bloat / token budget | ⚠️ 761 lines is on the upper edge — may exceed 30% of an implementer's context window if loaded whole |
| **VSDD 4-level mapping** | ❌ no separation into L2 domain spec vs L3 behavioral contracts; everything mixed at PRD level |

**Status:** RICH content, FORMAT mismatch. Cannot drop into `.factory/specs/prd.md` as-is — VSDD pipeline expects either FR-NNN (legacy) or, preferred, the 4-level decomposition with L2 capability sharding + L3 BC-S.SS.NNN per-file contracts.

### Architecture

**Status:** ❌ MISSING. No `.factory/specs/architecture/ARCH-INDEX.md`, no
section files (system-overview / module-decomposition / dependency-graph /
api-surface / verification-architecture / purity-boundary-map / tooling-selection
/ verification-coverage-matrix). The KoS bedrock nodes in `_kos/nodes/bedrock/`
are architectural building blocks but follow a different schema (KoS triples)
than VSDD ARCH-INDEX.

### Behavioral Contracts (L3 BC-S.SS.NNN)

**Status:** ❌ MISSING. No `.factory/specs/behavioral-contracts/BC-INDEX.md`,
no `BC-*.md` files.

### Verification Properties (L4 VP-NNN)

**Status:** ❌ MISSING. No `.factory/specs/verification-properties/VP-INDEX.md`,
no `VP-*.md` files.

### Stories

**Status:** ❌ MISSING from `.factory/stories/`. BMAD Story 0.1 exists in
`_bmad-output/implementation-artifacts/` and has already been delivered
as the stub code in `cmd/switchboard/`.

### Project context / brownfield signal

| Signal | Status |
|---|---|
| `cmd/switchboard/main.go` exists | ✅ 34 lines — stub from BMAD Story 0.1 |
| `cmd/switchboard/main_test.go` exists | ✅ 83 lines — stub tests |
| `internal/` populated | ❌ empty |
| `go.sum` populated | ❌ empty — no external deps |
| Real product surface | ❌ none — it is a "hello world"-class scaffold |

**Implication:** Although there is source code, it does **not** justify
brownfield-mode pipeline. The code is the BMAD Story 0.1 output (project
scaffolding only); there is no business logic to ingest. **Greenfield with
a head-start scaffolding** is the right characterization.

## Mode reconciliation

| Source | Says | Reality |
|---|---|---|
| `.factory/STATE.md` | `mode: greenfield` | ✅ correct |
| Orchestrator mode detection rule | "has src/ → BROWNFIELD" | ❌ false-positive — the src/ is just BMAD's scaffolding output |
| BMAD trail | brief → PRD → epic-0 → story-0.1 (DONE) | Aligned with greenfield-with-scaffolding |

Recommend keeping `mode: greenfield` and treating the existing `cmd/switchboard/`
code as the equivalent of a scaffolding wave already delivered.

## Concrete gap list

1. **No VSDD-format brief copy in `.factory/specs/`** — either copy the BMAD brief in or reference it via frontmatter `inputDocuments`.
2. **No L2 domain spec** (`.factory/specs/domain-spec/L2-INDEX.md` + capability shards). Needs creation from brief + BMAD PRD's "Domain-Specific Requirements" + KoS bedrock.
3. **No L3 PRD with BC-S.SS.NNN behavioral contracts.** The BMAD PRD's named requirement groups need to be decomposed into enumerated behavioral contracts (subsystem-scoped).
4. **No L4 verification properties.** Each L3 BC needs at least one VP-NNN with success criteria + verification method.
5. **No VSDD architecture docs.** ARCH-INDEX.md plus 7 section files; ADRs to be authored for major decisions (multi-path forwarding model, carrier-grade content separation, OpenSSH trust layer, session-native primitives, …). KoS bedrock nodes are inputs but not the artifact itself.
6. **No verification architecture.** Pure-core / effectful-shell purity boundary map needed before TDD wave.
7. **No stories in `.factory/stories/`.** The BMAD epic-0 / story-0.1 can be ported as the first cycle; subsequent epics need to be decomposed from L3 BCs.
8. **`.factory/.gitignore` not bootstrapped** — already filed upstream (drbothen/vsdd-factory#230). Local workaround: hand-create or accept the untracked `logs/` noise.
9. **STATE.md still says `product: corverax` in some path discussions** — already filed upstream (drbothen/vsdd-factory#229); local STATE.md is correct (`product: switchboard`) so no action.
10. **No `inputDocuments` frontmatter convention chosen yet** for how `.factory/specs/*` will trace back to `_bmad-output/` and `_kos/nodes/`. This matters for governance — VSDD wants traceability and the KoS graph is the project's source of architectural truth.

## Recommended next actions

The right entry point is **VSDD Phase 1: Spec Crystallization** — *not*
`/vsdd-factory:create-brief` (the brief already exists and is solid).

Suggested sequence:

1. **Anchor inputs.** Decide how `_bmad-output/` and `_kos/nodes/` are
   referenced from `.factory/specs/`. Two options:
   - **(A) Port** — copy the BMAD brief into `.factory/specs/product-brief.md` with
     a `source:` frontmatter pointer.
   - **(B) Reference** — leave the BMAD docs in place and have every `.factory/specs/`
     artifact carry `inputDocuments:` frontmatter pointing back to the BMAD originals.
   (B) preserves the BMAD history and avoids divergence. Recommended.
2. **Run `/vsdd-factory:create-domain-spec`** to produce the L2 domain
   spec from the brief + BMAD PRD's "Domain-Specific Requirements" + KoS
   bedrock nodes.
3. **Run `/vsdd-factory:create-prd`** to produce the VSDD-format L3 PRD
   with enumerated BC-S.SS.NNN behavioral contracts, decomposing the BMAD
   PRD's named requirement groups into per-subsystem contracts.
4. **Run `/vsdd-factory:create-architecture`** to produce ARCH-INDEX.md
   plus the 7 section files, drawing ADRs from the KoS bedrock nodes.
5. **Phase 1d adversarial spec review** — 3 clean passes minimum.
6. **`/vsdd-factory:decompose-stories`** — produces `.factory/stories/`
   with the BMAD epic-0 ported in as the first cycle (already delivered).
7. From there the normal Phase 2 → 3 → 4 → 5 → 6 → 7 pipeline applies.
