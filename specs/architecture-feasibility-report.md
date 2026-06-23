---
artifact_id: architecture-feasibility-report
document_type: architecture-feasibility-report
level: ops
version: "1.0"
status: approved
producer: architect
timestamp: 2026-06-23T00:00:00
phase: 1b
inputDocuments:
  - '.factory/specs/prd.md'
  - '.factory/specs/behavioral-contracts/BC-INDEX.md'
  - '.factory/specs/domain-spec/L2-INDEX.md'
  - '.factory/specs/prd-supplements/nfr-catalog.md'
traces_to: '.factory/specs/prd.md'
prd_version: "1.0"
input-hash: "[md5-pending]"
---

# Architecture Feasibility Report: Switchboard

## Executive Summary

The PRD v1.0 (42 BCs across 9 subsystems) is architecturally feasible with the
chosen Go-native toolchain. All 42 BCs have verification strategies using available
tools. No BCs are infeasible. The subsystem grouping is coherent with clean
purity-boundary separation. One structural correction is made (consolidating
admission + HMAC into a unified security module dependency layer), and five deferred
decisions from the PO are resolved as explicit ADRs. The PRD is approved for Phase 2
story decomposition.

## Constraint Mapping

| BC ID | Requirement | Architecture Constraint | Feasibility | Resolution |
|-------|-------------|------------------------|-------------|-----------|
| BC-2.01.001 | Timeslice clock fires every tick | `time.NewTicker` with compensation; hot path must not allocate | feasible | Pre-allocated frame pool; compensation logic in halfchannel |
| BC-2.01.004 | 44-byte fixed outer header | Fixed-size struct encoding in pure-core frame package | feasible | `[44]byte` array; no allocations |
| BC-2.01.005 | Channel header opaque to routers | Router code must not import or parse channel header type | feasible | CI scan enforces no `channel_header` import in routing |
| BC-2.02.001 | Duplicate-and-race on two paths | Concurrent send on two connections; goroutine-safe | feasible | Two goroutines write to two connections; no shared state |
| BC-2.02.004 | Upstream replay window | Ring buffer in pure replay module | feasible | Fixed-size ring buffer; O(1) operations |
| BC-2.02.005 | Downstream ARQ + SACK bitmap | 64-bit bitmap for 64-frame window | feasible | `uint64` SACK; standard sliding window |
| BC-2.02.007 | XOR FEC (PE phase) | FEC group coordinator gated by `upstream_routers` config | feasible | Build-tag-free gating via config; no dead code |
| BC-2.04.001 | tmux control mode | Unix socket IPC to tmux; `%output` event parsing | feasible | Standard `bufio.Scanner` over Unix socket; PTY fallback |
| BC-2.05.001 | Signed key challenge | `golang.org/x/crypto/ssh` Sign/Verify; nonce store | feasible | Standard Go SSH library; nonce TTL map |
| BC-2.05.005 | HMAC frame auth | HMAC-SHA256 (stdlib `crypto/hmac`); ADR-001 | feasible | Stdlib only; no external crypto dependency |
| BC-2.05.006 | SVTN cryptographic isolation | Per-SVTN routing table; key derivation | feasible | Map keyed by SVTN ID; no shared state between SVTNs |
| BC-2.09.001 | E→PE by config change | `upstream_routers` config field gates PE code paths | feasible | Empty slice = E; non-empty = PE; no rebuild needed |
| BC-2.09.003 | Config validation before bind | Validation returns `[]error`; bind only after empty error slice | feasible | Standard validation pattern; NFR-011 gate |

## Subsystem Grouping Assessment

| L2 Subsystem | PRD Grouping Valid? | NFR Profile Coherent? | Notes |
|--------------|--------------------|-----------------------|-------|
| session-networking (CAP-001–004) | Yes | Yes | Timeslice framing + session identity form a coherent security/performance unit |
| multipath-forwarding (CAP-005–010) | Yes | Yes | All reliability NFRs (NFR-002, NFR-003, NFR-007) are coherently in this subsystem |
| session-discovery (CAP-011–012) | Yes | Yes | Correctly scoped to PE phase; no latency NFRs bleed in from E phase |
| session-access (CAP-013–016) | Yes | Yes | tmux + PTY + attach/detach form a coherent session access unit |
| admission-security (CAP-017–020) | Yes | Yes | Two-tier key model is coherent; HMAC + admission are tightly coupled (both use same key material) |
| quality-observability (CAP-021–022) | Yes | Yes | Metrics + quality indicator are correctly grouped; NFR-014 maps cleanly here |
| network-management (CAP-023–024) | Yes | Yes | SVTN lifecycle + CLI are correctly grouped; no performance NFRs |
| console-operations (CAP-025) | Yes | Yes | Single-BC subsystem; correctly isolated to PE phase |
| deployment-operations (CAP-026–027) | Yes | Yes | E→PE graduation + drain form a coherent ops unit; BC-2.09.003 (P0) correctly included |

No restructuring proposed. All L2 subsystem boundaries translate cleanly to
architecture module boundaries.

## Proposed Restructuring

None. The L2 domain grouping aligns with architecture module boundaries. The only
architectural correction is the explicit dependency: `internal/routing` imports
`internal/admission` (routing decisions require the admitted key set for SVTN
isolation). This is a correct coupling, not a violation.

## Subsystem-to-Module Mapping

| L2 Subsystem | Modules | Pure/Effectful |
|-------------|---------|---------------|
| session-networking | internal/frame, internal/halfchannel | pure-core |
| multipath-forwarding | internal/multipath, internal/arq, internal/replay, internal/paths | pure-core |
| session-discovery | internal/discovery | boundary |
| session-access | internal/tmux, internal/session | boundary + effectful |
| admission-security | internal/hmac, internal/admission, internal/session | pure-core + boundary |
| quality-observability | internal/metrics, internal/paths | pure-core |
| network-management | internal/svtnmgmt, cmd/sbctl | boundary + effectful |
| console-operations | internal/session, cmd/sbctl | boundary + effectful |
| deployment-operations | internal/config, internal/drain | pure-core + effectful |

## Infeasible BCs

**None.** All 42 BCs have a verification strategy that is feasible with the Go-native
toolchain (gopter, go test -fuzz, go test -race, golangci-lint, go-mutesting).

The highest complexity verification is VP-015 (channel header opacity at router),
which uses fuzz + code audit. This is feasible and documented in ARCH-07.

### Phase 1c-Refinement Re-Affirmation (VP-053 through VP-057)

Five additional VPs were created in Phase 1c to close coverage gaps identified by
the PO sweep. Each is explicitly feasible:

| BC | VP | Feasibility Rationale |
|----|----|-----------------------|
| BC-2.01.002 | VP-053 | proptest over K∈[1,100] empty ticks; pure halfchannel state machine; fake clock; no I/O |
| BC-2.02.002 | VP-054 | integration test with in-process receiver stack + fake transport; no real network required |
| BC-2.03.003 | VP-055 | proptest over UTF-8 session names 1–255 bytes; pure Encode/Decode codec; no I/O |
| BC-2.04.004 | VP-056 | integration test with in-process session manager + fake transport; no tmux process required |
| BC-2.05.007 | VP-057 | proptest sampling (100 keypairs × 7 frame types) + inline HKDF non-exposure proof sketch; Go stdlib crypto only |

No new infeasibility findings. The HKDF one-way argument for VP-057 (Part ii) is a
reasoning sketch, not a machine-checked proof; this is explicitly noted in VP-057's
feasibility assessment as the standard approach for this class of security property.

## Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| NFR-002 jitter (5ms p99 at 5ms tick) | Medium | High | Phase 3 benchmark validates; architecture allows tick interval as low as 5ms but does not mandate it |
| NFR-004 (1,000 concurrent sessions) | Low | Medium | Per-connection goroutine model initially; profiling gates event-loop refactor if needed |
| tmux control mode reliability (R-004) | Medium | Medium | PTY fallback always available (BC-2.04.002); integration test (VP-031) validates 99% completeness |
| FEC group size N=4 default (ADR-002) | Low | Low | Tunable via config; Phase 3 benchmarks validate against PE topology loss rates |

## Decision Log

| Decision | Alternatives Considered | Chosen | Rationale |
|----------|------------------------|--------|-----------|
| HMAC algorithm (ADR-001) | BLAKE2, SHA-512, Poly1305 | HMAC-SHA256 | Aligns with HKDF-SHA256 in Noise; stdlib only; truncated to 16 bytes |
| FEC group size (ADR-002) | N=2 (50% overhead), N=8 (delayed recovery) | N=4 (20% overhead) | MOSH precedent; Phase 3 validates |
| Duplicate key registration (ADR-003) | Reject duplicate | Last-write-wins | Operational flexibility; authenticated registrant |
| Console key registration (ADR-004) | Allow console to register | Control node only | Principle of least privilege; access nodes have no management capability |
| ARQ failover continuity (ADR-005) | Stateful transfer | Resync from last ACK | Simpler; correct; stateful transfer deferred to PE phase |
| Daemon RPC (ADR-006) | gRPC, custom binary | JSON-over-Unix-socket | Zero dependency; debuggable; JSON schema already specified |
| P router build (ADR-007) | Include in main binary | Separate build target | NFR-012 (binary size); prevents accidental use |

## Approval

| Role | Decision | Date |
|------|----------|------|
| Architect | approve | 2026-06-23 |
| Product Owner | acknowledged | [pending PO review] |
