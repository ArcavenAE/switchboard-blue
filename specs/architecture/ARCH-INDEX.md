---
artifact_id: ARCH-INDEX
document_type: architecture-index
level: L3
version: "1.0"
status: draft
producer: architect
timestamp: 2026-06-23T00:00:00
phase: 1b
inputDocuments:
  - '.factory/specs/prd.md'
  - '.factory/specs/behavioral-contracts/BC-INDEX.md'
  - '.factory/specs/domain-spec/L2-INDEX.md'
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/domain-spec/risks.md'
  - '.factory/specs/domain-spec/failure-modes.md'
  - '.factory/specs/prd-supplements/nfr-catalog.md'
  - '.factory/specs/prd-supplements/interface-definitions.md'
  - '.factory/specs/module-criticality.md'
  - '_bmad-output/planning-artifacts/prd.md'
kos_anchors:
  - elem-single-binary-three-modes
  - elem-node-router-architecture
  - elem-timeslice-framing
  - elem-asymmetric-half-channels
  - elem-ssh-end-to-end-encryption
  - elem-dual-fastest-path-forwarding
  - elem-mvp-scope-single-lan
traces_to: '.factory/specs/prd.md'
deployment_topology: single-service
input-hash: "[md5-pending]"
---

# Architecture Index: Switchboard

> **Context Engineering:** Lightweight index (~300 tokens). Load only the section
> files needed for your task. All 42 BCs are covered; every module has a purity
> boundary classification.

## Document Map

| Section | File | Primary Consumer | Purpose |
|---------|------|-----------------|---------|
| System Overview | ARCH-00-overview.md | orchestrator, all agents | Architecture vision, E→PE→P topology, principles |
| Core Services | ARCH-01-core-services.md | implementer, story-writer | Daemon modes, lifecycle, supervision, mode dispatch |
| Protocol Stack | ARCH-02-protocol-stack.md | implementer, formal-verifier | Wire format, HMAC auth, Noise handshake (ADR-001, ADR-002) |
| Routing Engine | ARCH-03-routing-engine.md | implementer, formal-verifier | Path selection, duplicate-and-race, FEC, ARQ, failover |
| Admission & Security | ARCH-04-admission-security.md | security-reviewer, formal-verifier | Keypair admission, SVTN isolation, key model (ADR-003–005) |
| CLI & API | ARCH-05-cli-and-api.md | implementer, test-writer | sbctl, daemon RPC, Go package layout, module→BC mapping |
| Deployment & Ops | ARCH-06-deployment-and-ops.md | devops-engineer | Binary build, platform support, signing, upgrade model |
| Verification Architecture | ARCH-07-verification-architecture.md | formal-verifier | Purity boundaries, VP strategy, P0/P1 proof catalog |
| Dependency Graph | ARCH-08-dependency-graph.md | consistency-validator | Acyclic module DAG, topological order |
| Purity Boundary Map | ARCH-09-purity-boundary-map.md | implementer, formal-verifier | Per-package pure/boundary/infra/effectful classification |
| Tooling Selection | ARCH-10-tooling-selection.md | formal-verifier | Go-native verification toolchain selection and rationale |
| Verification Coverage Matrix | ARCH-11-verification-coverage-matrix.md | consistency-validator | VP-to-BC coverage table |

## Cross-References

| If you need... | Read these together |
|----------------|-------------------|
| Implementation plan for a module | ARCH-05 + ARCH-08 + ARCH-09 |
| Verification plan for a module | ARCH-07 + ARCH-09 + ARCH-10 |
| Full module picture | ARCH-01 + ARCH-05 + ARCH-08 + ARCH-09 |
| Story decomposition input | ARCH-05 + ARCH-08 |
| Security review | ARCH-02 + ARCH-04 + ARCH-07 |

## Subsystem Registry

> **Source of truth** for subsystem names and IDs. BC frontmatter `subsystem:`,
> BC-INDEX subsystem column, story `subsystems:` fields, and PRD subsystem
> references MUST all use the exact Name from this table.

| SS ID | Name | Architecture Doc | Implementing Modules | Phase Introduced |
|-------|------|-----------------|---------------------|-----------------|
| SS-01 | session-networking | ARCH-02-protocol-stack.md | internal/frame, internal/halfchannel | Phase 1 (E) |
| SS-02 | multipath-forwarding | ARCH-03-routing-engine.md | internal/multipath, internal/arq, internal/replay, internal/paths | Phase 1 (E) |
| SS-03 | session-discovery | ARCH-03-routing-engine.md | internal/discovery | Phase 2 (PE) |
| SS-04 | session-access | ARCH-01-core-services.md | internal/tmux, internal/session | Phase 1 (E) |
| SS-05 | admission-security | ARCH-04-admission-security.md | internal/hmac, internal/admission, internal/session | Phase 1 (E) |
| SS-06 | quality-observability | ARCH-03-routing-engine.md | internal/metrics, internal/paths | Phase 1 (E) |
| SS-07 | network-management | ARCH-05-cli-and-api.md | internal/svtnmgmt, cmd/sbctl | Phase 1 (E) |
| SS-08 | console-operations | ARCH-01-core-services.md | internal/session, cmd/sbctl | Phase 2 (PE) |
| SS-09 | deployment-operations | ARCH-06-deployment-and-ops.md | internal/config, internal/drain | Phase 1 (E) |

## ADR Registry

| ADR | Decision | Section | Status |
|-----|----------|---------|--------|
| ADR-001 | HMAC algorithm: HMAC-SHA256 with per-SVTN derived key | ARCH-02, ARCH-04 | decided |
| ADR-002 | FEC group size: N=4 default, tunable via config | ARCH-03 | decided |
| ADR-003 | Duplicate key registration: last-write-wins | ARCH-04 | decided |
| ADR-004 | Console key registration model: console keys managed by control node only | ARCH-04 | decided |
| ADR-005 | Downstream ARQ continuity under router failover: resync on reconnect | ARCH-03 | decided |
| ADR-006 | Daemon RPC: JSON-over-Unix-socket with SSH signature auth | ARCH-05 | decided |
| ADR-007 | P router: separate build target, not included in MVP binary | ARCH-06 | decided |
| ADR-008 | Tick interval range: 5–50ms; validated as tuning parameter in Phase 3 | ARCH-02 | decided |

## Tuning Parameters (to be validated in Phase 3)

The following are empirical parameters — architecture specifies the range and
mechanism; Phase 3 benchmarks validate the defaults:

- **Tick interval:** 5–50ms range. Default upstream 10ms, downstream 50ms. (ADR-008)
- **Presence heartbeat interval:** 30s default. To be validated against discovery latency tolerance in Phase 3.

## Open Frontier Questions (for KoS process)

1. Is HMAC-SHA256 (no Noise integration in MVP) sufficient, or does the control plane need Noise XX for router-to-router in PE phase?
2. Does the ARQ SACK bitmap size need to be configurable, or is a fixed 64-bit bitmap sufficient for all session types?
3. What is the correct goroutine model for 1,000 concurrent sessions — per-session goroutines vs. event-loop? (NFR-004 open question)
4. Does the drop cache need a TTL-based eviction in addition to LRU, to avoid retransmit suppression in long-running sessions?
5. At PE phase, does router-to-router Noise handshake reuse the same keypair as node admission, or require a separate router identity keypair?
