---
generated: 2026-06-23
phase_entry: phase-1-spec-crystallization
anchor_strategy: reference-via-frontmatter
first_step: create-domain-spec
---

# Routing Decision — Switchboard

## Entry point

**VSDD Phase 1: Spec Crystallization**, starting with the L2 domain spec.

`/vsdd-factory:create-brief` is **skipped** — the BMAD product brief at
`_bmad-output/planning-artifacts/product-brief-switchboard-2026-03-31.md`
is substantive (334 lines), defines scope phases (E Router → Multi-Path/Multi-Hop
→ Global Topology), and was BMAD-validated 4/5 / "Pass".

## Anchor strategy: reference-via-frontmatter

Every artifact authored under `.factory/specs/` will carry an `inputDocuments:`
frontmatter array pointing back to the BMAD and KoS source documents. The BMAD
docs in `_bmad-output/` and KoS graph in `_kos/nodes/` remain authoritative;
`.factory/specs/` is a derived VSDD-format view that respects the kos process
as the project's source of architectural truth.

Required frontmatter template for every VSDD spec artifact:

```yaml
---
artifact_id: <e.g. L2-CAP-001 / BC-1.01.001 / VP-001 / ARCH-section-name>
status: draft | review | accepted
inputDocuments:
  - '_bmad-output/planning-artifacts/product-brief-switchboard-2026-03-31.md'
  - '_bmad-output/planning-artifacts/prd.md'
  - '_kos/nodes/bedrock/<node>.yaml'
kos_anchors:           # optional — direct kos node IDs
  - <node-id>
---
```

This frontmatter is the traceability fence. The spec-steward enforces it.
When BMAD docs or KoS nodes change, every artifact whose `inputDocuments` list
mentions them is flagged for input-hash drift review (via
`/vsdd-factory:check-input-drift`).

## Phase 1 sub-sequence

1. **`/vsdd-factory:create-domain-spec`** (L2) — produces
   `.factory/specs/domain-spec/L2-INDEX.md` + capability/entity/invariant
   shards. Inputs:
   - BMAD brief
   - BMAD PRD §"Domain-Specific Requirements", §"Network Infrastructure Requirements"
   - KoS bedrock nodes
   - Brainstorming session(s) where they ground domain concepts
2. **`/vsdd-factory:create-prd`** (L3) — produces
   `.factory/specs/prd.md` with BC-S.SS.NNN behavioral contracts decomposed
   from the BMAD PRD's named requirement groups (Session Networking,
   Multi-Path Forwarding, Session Discovery & Presence, Session Access &
   Sharing, Network Admission & Security, Network Quality & Observability,
   Network Management, Console Operations, Deployment & Operations).
3. **`/vsdd-factory:create-architecture`** — produces the ARCH-INDEX.md +
   7 section files (system-overview, module-decomposition, dependency-graph,
   api-surface, verification-architecture, purity-boundary-map,
   tooling-selection, verification-coverage-matrix). ADRs drawn from KoS
   bedrock nodes.
4. **Phase 1d adversarial spec review** — 3 clean passes minimum
   (`/vsdd-factory:phase-1d-adversarial-spec-review`).
5. **Human approval gate** with structured questions per FACTORY.md.

## Open questions deferred to Phase 1 execution

- **Subsystem taxonomy (S in BC-S.SS.NNN).** Suggested cut from the BMAD PRD
  + KoS bedrock: `session-networking`, `multipath-forwarding`,
  `session-discovery`, `session-access`, `admission-security`,
  `quality-observability`, `network-management`, `console-operations`,
  `deployment-operations`. Final shape is the business-analyst's call during
  L2 decomposition.
- **Story 0.1 reconciliation.** The BMAD `epic-0-project-scaffolding` + `story-0.1`
  describe work already done (the `cmd/switchboard/` stub). When `decompose-stories`
  runs, port these as a "scaffolding-complete" pre-cycle, not as a wave-1 story.
- **Charter / kos integration.** `charter.md` is currently template-empty.
  Decide during create-domain-spec whether to populate it as part of the
  domain spec output, or leave it as a separate kos-process artifact.

## Non-blocking debt logged

- `.factory/.gitignore` not bootstrapped (drbothen/vsdd-factory#230, comment
  added this session). Decide locally whether to hand-author the file or
  accept the `logs/` untracked noise until upstream fixes.
