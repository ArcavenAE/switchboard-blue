---
artifact_id: ARCH-INDEX
document_type: architecture-index
level: L3
version: "1.1"
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
input-hash: "TODO: compute-input-hash requires inputs: field; ARCH-INDEX uses inputDocuments: — reconcile field name or compute manually from inputDocuments list before freeze"
modified:
  - 2026-06-23T00:00:00
  - 2026-06-28T00:00:00 # v1.1 — ARCH-12 daemon management plane + ADR-012 added
---

# Architecture Index: Switchboard

> **Context Engineering:** Lightweight index (~300 tokens). Load only the section
> files needed for your task. All 44 BCs are covered; every module has a purity
> boundary classification.

## Document Map

| Section | File | Primary Consumer | Purpose |
|---------|------|-----------------|---------|
| System Overview | ARCH-00-overview.md | orchestrator, all agents | Architecture vision, E→PE→P topology, principles |
| Core Services | ARCH-01-core-services.md | implementer, story-writer | Daemon modes, lifecycle, supervision, mode dispatch |
| Protocol Stack | ARCH-02-protocol-stack.md | implementer, formal-verifier | Wire format, HMAC auth, Noise handshake (ADR-001, ADR-002) |
| Routing Engine | ARCH-03-routing-engine.md | implementer, formal-verifier | Path selection, duplicate-and-race, FEC, ARQ, failover |
| Admission & Security | ARCH-04-admission-security.md | security-reviewer, formal-verifier | Keypair admission, SVTN isolation, key model (ADR-003–005, ADR-009) |
| CLI & API | ARCH-05-cli-and-api.md | implementer, test-writer | sbctl, daemon RPC, Go package layout, module→BC mapping |
| Deployment & Ops | ARCH-06-deployment-and-ops.md | devops-engineer | Binary build, platform support, signing, upgrade model |
| Verification Architecture | ARCH-07-verification-architecture.md | formal-verifier | Purity boundaries, VP strategy, P0/P1 proof catalog |
| Dependency Graph | ARCH-08-dependency-graph.md | consistency-validator | Acyclic module DAG, topological order |
| Purity Boundary Map | ARCH-09-purity-boundary-map.md | implementer, formal-verifier | Per-package pure/boundary/infra/effectful classification |
| Tooling Selection | ARCH-10-tooling-selection.md | formal-verifier | Go-native verification toolchain selection and rationale |
| Verification Coverage Matrix | ARCH-11-verification-coverage-matrix.md | consistency-validator | VP-to-BC coverage table |
| Daemon Management Plane | ARCH-12-daemon-management-plane.md | implementer, security-reviewer, story-writer | ADR-012 wire protocol, internal/mgmt package, config additions, client auth, BC/VP/story recommendations |

## Cross-References

| If you need... | Read these together |
|----------------|-------------------|
| Implementation plan for a module | ARCH-05 + ARCH-08 + ARCH-09 |
| Verification plan for a module | ARCH-07 + ARCH-09 + ARCH-10 |
| Full module picture | ARCH-01 + ARCH-05 + ARCH-08 + ARCH-09 |
| Story decomposition input | ARCH-05 + ARCH-08 |
| Security review | ARCH-02 + ARCH-04 + ARCH-07 |
| Management plane implementation | ARCH-05 + ARCH-12 |

## Subsystem Registry

> **Source of truth** for subsystem names and IDs. BC frontmatter `subsystem:`,
> BC-INDEX subsystem column, story `subsystems:` fields, and PRD subsystem
> references MUST all use the exact Name from this table.

Modules tagged `(shared with SS-NN)` indicate a Go package that serves multiple subsystems. The primary owning subsystem is listed first; consumers are noted in parentheses.

| SS ID | Name | Architecture Doc | Implementing Modules | Phase Introduced |
|-------|------|-----------------|---------------------|-----------------|
| SS-01 | session-networking | ARCH-02-protocol-stack.md | internal/frame, internal/halfchannel, internal/admission (shared with SS-05; used for re-auth on IP change per BC-2.01.007) | Phase 1 (E) |
| SS-02 | multipath-forwarding | ARCH-03-routing-engine.md | internal/multipath, internal/arq, internal/replay, internal/paths, internal/routing (shared with SS-05; used for forwarding decisions per BC-2.02.008) | Phase 1 (E) |
| SS-03 | session-discovery | ARCH-03-routing-engine.md | internal/discovery | Phase 2 (PE) |
| SS-04 | session-access | ARCH-01-core-services.md | internal/tmux, internal/session | Phase 1 (E) |
| SS-05 | admission-security | ARCH-04-admission-security.md | internal/hmac, internal/admission, internal/session | Phase 1 (E) |
| SS-06 | quality-observability | ARCH-03-routing-engine.md | internal/metrics, internal/paths | Phase 1 (E) |
| SS-07 | network-management | ARCH-05-cli-and-api.md, ARCH-12-daemon-management-plane.md | internal/svtnmgmt, internal/mgmt (Wave 5), cmd/sbctl | Phase 1 (E); internal/mgmt Wave 5 |
| SS-08 | console-operations | ARCH-01-core-services.md | internal/session, cmd/sbctl | Phase 2 (PE) |
| SS-09 | deployment-operations | ARCH-06-deployment-and-ops.md | internal/config, internal/drain | Phase 1 (E) |

## ADR Registry

| ADR | Decision | Section | Status |
|-----|----------|---------|--------|
| ADR-001 | HMAC algorithm: HMAC-SHA256 with per-node per-SVTN HKDF-derived key | ARCH-02, ARCH-04 | decided (amended 2026-06-23) |
| ADR-002 | FEC group size: N=4 default, tunable via config | ARCH-03 | decided |
| ADR-003 | Duplicate key registration: last-write-wins | ARCH-04 | decided |
| ADR-004 | Console key registration model: permission hierarchy (control > console > readonly); cross-role revocation rules | ARCH-04 | decided (amended 2026-06-23) |
| ADR-005 | Downstream ARQ continuity under router failover: resync on reconnect | ARCH-03 | decided |
| ADR-006 | Daemon RPC: JSON-over-Unix-socket with SSH signature auth | ARCH-05 | decided |
| ADR-007 | P router: separate build target, not included in MVP binary | ARCH-06 | decided |
| ADR-008 | Tick interval range: 5–50ms; validated as tuning parameter in Phase 3 | ARCH-02 | decided |
| ADR-009 | HMAC enforcement at RouteFrame boundary: fail-fast before admitted-set lookup (S-3.04) | ARCH-04 | decided |
| ADR-010 | Terminal session backend: tmux control mode primary, PTY proxy fallback (S-3.01) | ARCH-01, ARCH-04 | decided |
| ADR-011 | SessionConnector.Frames(): forwarding-channel design for failover-stable frame delivery (S-4.00) | ARCH-01 | decided |
| ADR-012 | Management-auth wire protocol: NDJSON framing, Ed25519 challenge-response, operator key set, bounded read (Wave 5) | ARCH-12 | decided |

## Tuning Parameters (to be validated in Phase 3)

The following are empirical parameters — architecture specifies the range and
mechanism; Phase 3 benchmarks validate the defaults:

- **Tick interval:** 5–50ms range. Default upstream 10ms, downstream 50ms. (ADR-008)
- **Presence heartbeat interval:** 30s default. To be validated against discovery latency tolerance in Phase 3.

## Changelog

| Date | Author | Change |
|------|--------|--------|
| 2026-06-23 | architect | Round-1 architectural refinement (pass-01 adversarial review): wire format canonicalized to bit-precise 44-byte outer header (F-001, F-002, F-004, F-011); HMAC keying updated to per-node HKDF-SHA256 (F-003); drop cache key extended to (checksum, arrival_interface_id) (F-006); quality thresholds aligned to NFR-001/BC-2.06.001 (F-008); hysteresis canonical value set to 3 measurements (F-021); read-only console ACK resolved via degenerate upstream half-channel (F-023); permission hierarchy (control > console > readonly) and cross-role revocation rules documented in ADR-004 (F-010); SHA-256 adopted for address derivation replacing Blake3 (F-007); VP-051 and VP-052 added; VP total now 52; VP-040 module corrected to internal/multipath (F-014). |
| 2026-06-23 | architect | Phase 1c-refinement: VP-053 through VP-057 added for BC coverage closure (BC-2.01.002, BC-2.02.002, BC-2.03.003, BC-2.04.004, BC-2.05.007). VP total now 57. |
| 2026-06-25 | architect | Wave 3 planning refresh: ADR-009 (HMAC enforcement at RouteFrame boundary, S-3.04) and ADR-010 (tmux control mode + PTY fallback, S-3.01) added to ADR Registry. ARCH-04 bumped to v1.4 (ADR-009 section). ARCH-08 bumped to v1.2 (§6.5: full Wave 1–3 package table declaring `internal/session` at position 6, `internal/tmux` at position 13). No new packages — Wave 3 reuses existing DAG positions from the full topological order already documented in §§1–4. |
| 2026-06-27 | architect | ADR-011 added (SessionConnector.Frames() forwarding-channel design, S-4.00). ARCH-01 bumped to v1.3. ARCH-08 bumped to v1.9 (§6.5.1 wiring obligations, §6.5.2 import set, §6.6.1 feasibility register, §6.6.2 prospective positions). cmd/switchboard position 18 registered as ACTIVE SCOPE for story S-4.00. |
| 2026-06-27 | architect | ARCH-01 v1.4: ADR-011 §Concurrency amended — relay-drop counter contract (`sc.relayDropped` atomic, `RelayDropped()` method), relay busy-spin guard (ctx param + `runtime.Gosched()`), daemon `sc.Err()` drain obligation in wg-tracked goroutine (E-SYS-002, BC-2.04.002 invariant 3). Per S-W3.04 adversarial convergence adjudication. |
| 2026-06-27 | architect | ARCH-01 v1.5: ADR-011 Amendment — PTY-source EOF is session-fatal (ErrPTYSourceEOF on `sc.errCh`; discrimination: PTY mode → fatal, control mode → yield-and-retry). New E-SYS-003 taxonomy entry. `runAccess` split into thin wrapper + `runAccessWithConnector(connectorIface)` injection seam for PC-2/PC-2.6 coverage. Per S-W3.04 adversarial convergence pass-2. |
| 2026-06-27 | architect | ARCH-01 v1.6: ADR-011 HIGH-A TOCTOU fix — `activeSourceSnapshot()` helper reads `{src, srcCh, inPTYMode}` under single `sc.mu` hold to eliminate race between separate activeFrSource + InPTYMode acquisitions (~20% EC-002 false-EOF). Two new test obligations (T1 bounded-exit, T2 EC-002 stress). Per S-W3.04 adversarial convergence pass-3. |
| 2026-06-27 | architect | ARCH-01 v1.7: Wave-3 wave-level adversarial pass-1 I-1 adjudication — added §Goroutine WaitGroup Contract under Daemon Lifecycle. All four post-connect goroutines in `runAccessWithConnector` MUST be wg-tracked; `startSweepTicker` and `startFramesDroppedTicker` accept `*sync.WaitGroup`. Cross-refs: BC-2.04.007 PC-2 postcon-6, ARCH-08 §6.5.1 v2.2, S-W3.04 AC-008. |
| 2026-06-27 | architect | ARCH-08 v2.0: §6.5.1 obligation 1 clarified — router is constructed-but-not-yet-in-data-path in Wave 3; `buildRouter` return value MUST be assigned (not discarded); shared `*admission.AdmittedKeySet` instance required. Per S-W3.04 adversarial convergence adjudication. |
| 2026-06-27 | architect | ARCH-08 v2.1: §6.5.2 import set adds `internal/frame` (OuterHeader carrier, DAG pos 2 leaf). §6.5.1 obligation 4 note: `runAccess` injection seam split. EC-005 "CI enforces structurally" wording accepted as Wave-4 follow-up. Per S-W3.04 adversarial convergence pass-2. |
| 2026-06-27 | architect | ARCH-08 v2.2: Wave-3 wave-level adversarial pass-1 C-1/I-1 adjudication. C-1 TRACKED-DEFER: `routing.WithFailureCounter` wiring deferred to future network-ingress story (E-ADM-016/017 must wire together when RouteFrame enters live data path). I-1 wg-join: obligations 3 and 6 updated — `startSweepTicker` and `startFramesDroppedTicker` accept `*sync.WaitGroup` for deterministic BC-2.04.007 PC-2 postcon-6 verification. |
| 2026-06-27 | architect | ARCH-08 v2.3: C-1 RESOLVED — `routing.WithFailureCounter(fc)` (threshold=5, window=60s) wired in `buildRouter` alongside `routing.WithLogger`, PR #20 (commit 418de54). Partial-wiring concern closed; BC-2.05.008 PC-5 and BC-2.05.005 PC-3 satisfied. OBS-3 resolved. Only remaining deferral is the network-ingress listener (S-BL.NI). |
| 2026-06-28 | architect | ARCH-03 v1.5–1.6: Wave 5 design notes. (1) Degraded-path flag (S-5.03, BC-2.02.003 PC-5): `IsDegraded()` accessor, `degraded bool` field under existing `mu`, `DegradedRTTThresholdMS=200`, `PathSnapshot` value type for consistent multi-field reads (go.md rule 12), `internal/metrics` consumes via `Snapshot().Degraded`. (2) p99 RTT accumulator (S-5.02, BC-2.06.003 rtt_p99_ms): 16-bucket fixed-histogram owned by PathTracker, 128 bytes per path, O(1) update and query, `P99RTTMs` carried in `PathSnapshot`, accuracy VP deferred to S-BL.BENCH. BC-2.06.003 added to ARCH-03 inputDocuments. |
| 2026-06-29 | architect | ARCH-04 v1.8: CR-001 ruling — `internal/svtnmgmt` must not import `internal/frame` (ARCH-08 §6.5 position 15 violation). Resolution: add `AdmittedKeySet.LookupByPubkey(svtnID, pubkey)` to `internal/admission`; svtnmgmt replaces direct `frame.DeriveNodeAddress` calls with this method. HOLD-001 implementation invariant step 2 updated to reference `LookupByPubkey`. |
| 2026-06-28 | architect | ARCH-12 v1.0 added: Daemon Management Plane — closes the Wave 5 management-plane gap. ADR-012 specifies NDJSON framing, Ed25519 challenge-response handshake (6-step sequence), operator key set (config-sourced, distinct from SVTN AdmittedKeySet), bounded-read contract (MaxMessageBytes=64KiB, CWE-400). Specifies internal/mgmt package (Server, NewServer, OperatorKeySet, Handler registry), config additions (ManagementSocket, AuthorizedOperatorKeys), cmd/sbctl Authenticate() fail-closed contract, golang.org/x/crypto dependency. Recommends BC-2.07.004 (server-side auth), VP-058 through VP-062, stories S-6.03/S-W5.01/S-W5.02. Flags S-6.03 EC-002 bug (E-ADM-001 should be E-ADM-010). ARCH-05 bumped to v1.1. ARCH-INDEX bumped to v1.1. |

## Open Frontier Questions (for KoS process)

1. Is HMAC-SHA256 (no Noise integration in MVP) sufficient, or does the control plane need Noise XX for router-to-router in PE phase?
2. Does the ARQ SACK bitmap size need to be configurable, or is a fixed 64-bit bitmap sufficient for all session types?
3. What is the correct goroutine model for 1,000 concurrent sessions — per-session goroutines vs. event-loop? (NFR-004 open question)
4. Does the drop cache need a TTL-based eviction in addition to LRU, to avoid retransmit suppression in long-running sessions?
5. At PE phase, does router-to-router Noise handshake reuse the same keypair as node admission, or require a separate router identity keypair?
