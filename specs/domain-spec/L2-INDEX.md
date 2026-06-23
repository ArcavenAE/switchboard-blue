---
artifact_id: L2-INDEX
document_type: domain-spec-index
level: L2
version: "1.0"
status: draft
producer: business-analyst
timestamp: 2026-06-23T00:00:00
modified: ["2026-06-23"]
phase: 1a
inputDocuments:
  - '_bmad-output/planning-artifacts/product-brief-switchboard-2026-03-31.md'
  - '_bmad-output/planning-artifacts/prd.md'
  - '_bmad-output/brainstorming/brainstorming-session-2026-03-07-001.md'
  - '_bmad-output/brainstorming/naming-node-type-parking-lot.md'
  - '_bmad-output/brainstorming/session-context-cache.md'
kos_anchors:
  - elem-asymmetric-half-channels
  - elem-dual-fastest-path-forwarding
  - elem-mvp-scope-single-lan
  - elem-node-router-architecture
  - elem-single-binary-three-modes
  - elem-ssh-end-to-end-encryption
  - elem-timeslice-framing
input-hash: "[md5-pending]"
traces_to: '_bmad-output/planning-artifacts/product-brief-switchboard-2026-03-31.md'
sections:
  - capabilities.md
  - entities.md
  - invariants.md
  - bounded-contexts.md
  - ubiquitous-language.md
  - edge-cases.md
  - assumptions.md
  - risks.md
  - failure-modes.md
  - differentiators.md
---

# L2 Domain Specification: Switchboard

> **Sharded artifact (DF-021).** This index provides navigation and summary.
> Detail lives in per-section files listed below. Each section targets
> 800–1,200 tokens for optimal LLM consumption.

## Domain Summary

Switchboard is a purpose-built session network — a switched virtual network
(SVTN) whose only cargo is terminal sessions. Three node types (access node,
console, control node) communicate through routers that provide latency-aware
multi-path forwarding with carrier-grade content separation. SSH provides
end-to-end trust; routers see identity, addressing, and traffic patterns but
never session content.

## Document Map

| Section | File | Primary Consumer | Purpose |
|---------|------|-----------------|---------|
| Domain Capabilities | capabilities.md | product-owner, architect, story-writer | CAP-NNN capability catalog — 9 subsystems, 29 capabilities |
| Domain Entities | entities.md | architect, product-owner | Entity model — nodes, routers, SVTNs, keys, frames, sessions |
| Domain Invariants | invariants.md | product-owner, architect | DI-NNN business rules — content separation, admission, framing |
| Bounded Contexts | bounded-contexts.md | architect | Context boundaries — 9 subsystems, scope-phase topology |
| Ubiquitous Language | ubiquitous-language.md | all | Glossary — node types, router modes, protocol terms |
| Edge Cases | edge-cases.md | story-writer, test-writer | DEC-NNN domain-level edge cases |
| Assumptions | assumptions.md | product-owner, test-writer | ASM-NNN with validation methods |
| Risks | risks.md | product-owner, architect | R-NNN risk register |
| Failure Modes | failure-modes.md | architect, test-writer | FM-NNN runtime failure catalog |
| Differentiators | differentiators.md | product-owner | Competitive differentiator → CAP-NNN mapping |

## Cross-References

| If you need... | Read these together |
|----------------|-------------------|
| BC creation input | capabilities.md + invariants.md + edge-cases.md + assumptions.md + risks.md + differentiators.md |
| Architecture design input | capabilities.md + entities.md + invariants.md + bounded-contexts.md + risks.md + failure-modes.md |
| Story decomposition input | capabilities.md + edge-cases.md |
| Holdout scenario generation | assumptions.md + risks.md + failure-modes.md |
| NFR derivation | risks.md + failure-modes.md |
| Full domain review | ALL sections |

## Subsystem Taxonomy

Nine subsystems derived from the PRD's named requirement groups. Each will
map to an `S` bucket in future BC-S.SS.NNN behavioral contracts.

| ID | Subsystem | Rationale |
|----|-----------|-----------|
| `sn` | session-networking | Core session primitives — channels, half-channels, SVTN establishment |
| `mf` | multipath-forwarding | Dual-path, duplicate-and-race, FEC, failover, loop prevention |
| `sd` | session-discovery | Multicast presence protocol, in-SVTN session advertisement |
| `sa` | session-access | Access node publishing, console attach/detach, access modes |
| `as` | admission-security | Key-based SVTN admission (Tier 1), session authorization (Tier 2) |
| `qo` | quality-observability | Latency budgets, degradation signaling, per-path metrics |
| `nm` | network-management | sbctl CLI, control node, key lifecycle, SVTN lifecycle |
| `co` | console-operations | Console control plane, remote control, attach UX |
| `do` | deployment-operations | Upgrade model, platform support, single-binary deployment |

Note: `bounded-contexts.md` replaces the template's `events.md` for this
pipeline-oriented domain. Processing stages are captured within capabilities.
`ubiquitous-language.md` replaces `event-flow.md` as the primary reference
artifact for domain terminology.

## ID Registry Summary

| ID Format | Count | Section |
|-----------|-------|---------|
| CAP-NNN | 29 | capabilities.md |
| DI-NNN | 12 | invariants.md |
| DEC-NNN | 14 | edge-cases.md |
| ASM-NNN | 8 | assumptions.md |
| R-NNN | 10 | risks.md |
| FM-NNN | 12 | failure-modes.md |

## Priority Distribution

| Priority | Count | Items |
|----------|-------|-------|
| P0 (must-have) | 16 | CAP-001–CAP-008, CAP-010, CAP-013, CAP-016–CAP-018, CAP-020, CAP-020a, CAP-020b |
| P1 (should-have) | 9 | CAP-009, CAP-011–CAP-012, CAP-014–CAP-015, CAP-019, CAP-021–CAP-022, CAP-025 |
| P2 (nice-to-have) | 4 | CAP-023–CAP-024, CAP-026–CAP-027 |
