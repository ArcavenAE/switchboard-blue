---
artifact_id: module-criticality
document_type: module-criticality
level: ops
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
inputs:
  - '.factory/specs/prd.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/domain-spec/capabilities.md'
input-hash: "[md5-pending]"
traces_to: '.factory/specs/prd.md'
---

# Module Criticality Classification: Switchboard

> Architecture subsystem names map to L2 domain subsystems.
> Exact Go package paths are indicative (architecture decision).
> Tier descriptions use the standard mutation kill-rate vocabulary.

## Tier Definitions

| Tier | Mutation Kill Rate Target | Description | Examples |
|------|--------------------------|-------------|----------|
| **CRITICAL** | >= 95% | Core security, protocol correctness, content separation | HMAC auth, frame encoding, admission security |
| **HIGH** | >= 90% | Important session behavior, quality, observable correctness | Half-channel logic, path selection, ARQ, TLPKTDROP |
| **MEDIUM** | >= 80% | Supporting functionality | Presence advertisement, key lifecycle, CLI output |
| **LOW** | >= 70% | Infrastructure, config parsing, logging, scaffolding | Config file parsing, log formatting |

## Module Inventory

- **frame** — Wire-format encoding/decoding of 44-byte outer header and channel header
- **hmac** — HMAC-SHA256 computation and verification for frame authentication
- **admission** — Tier 1 signed-challenge admission, admitted key set management
- **session-auth** — Tier 2 per-session authorization, read-only enforcement
- **halfchannel** — Timeslice clock, upstream/downstream state machines, tick regularity
- **multipath** — Duplicate-and-race dispatch, receiver deduplication
- **paths** — Per-path RTT/loss metrics, keep-alive, path ranking
- **arq** — Downstream ARQ with piggybacked ACK/SACK
- **replay** — Upstream idempotent replay window
- **routing** — Router split-horizon, drop cache, per-SVTN frame partitioning
- **discovery** — Presence advertisement, session enumeration
- **metrics** — Quality indicator computation, threshold logic
- **tmux-control** — tmux control mode integration, PTY fallback
- **config** — Config file parsing, validation, reload
- **sbctl** — CLI command dispatch, output formatting
- **svtn-mgmt** — SVTN create/destroy, key lifecycle management
- **drain** — Router drain signal, node migration coordination

## Module Classification

| Module | Path (indicative) | Tier | Rationale | Kill Rate Target | VP Count |
|--------|-------------------|------|-----------|-----------------|----------|
| frame | internal/frame | CRITICAL | Correct wire format is a security boundary (DI-001, DI-007); incorrect encoding breaks all downstream | >= 95% | TBD |
| hmac | internal/hmac | CRITICAL | Frame authentication is the SVTN trust boundary (DI-006); any bug enables forged frames | >= 95% | TBD |
| admission | internal/admission | CRITICAL | SVTN entry gate (DI-006, DI-005); bypass = attacker on network | >= 95% | TBD |
| session-auth | internal/session | CRITICAL | Tier 2 enforcement (DI-010); bypass = unauthorized session access | >= 95% | TBD |
| routing | internal/routing | CRITICAL | SVTN isolation (DI-005); loop prevention; security + correctness | >= 95% | TBD |
| halfchannel | internal/halfchannel | HIGH | Timeslice clock regularity (DI-008); incorrect ticks break liveness detection | >= 90% | TBD |
| arq | internal/arq | HIGH | Downstream delivery correctness; incorrect ARQ produces corrupt terminal state | >= 90% | TBD |
| replay | internal/replay | HIGH | Upstream keystroke delivery; incorrect deduplication causes phantom keystrokes | >= 90% | TBD |
| multipath | internal/multipath | HIGH | Duplicate-and-race correctness; incorrect deduplication causes duplicate data | >= 90% | TBD |
| paths | internal/paths | HIGH | Path ranking drives all forwarding decisions; incorrect ranking degrades latency | >= 90% | TBD |
| metrics | internal/metrics | HIGH | Quality indicator accuracy (NFR-014); incorrect threshold logic misleads operators | >= 90% | TBD |
| tmux-control | internal/tmux | HIGH | Session content flows through this module; PTY fallback correctness | >= 90% | TBD |
| discovery | internal/discovery | MEDIUM | Eventual consistency acceptable; bugs cause temporary stale session lists | >= 80% | TBD |
| svtn-mgmt | internal/svtnmgmt | MEDIUM | SVTN lifecycle; important but not a hot path; errors are recoverable | >= 80% | TBD |
| drain | internal/drain | MEDIUM | Graceful shutdown path; session loss on failure is bounded | >= 80% | TBD |
| config | internal/config | MEDIUM | Errors detected at startup before any sessions affected | >= 80% | TBD |
| sbctl | cmd/sbctl | LOW | Operator CLI; bugs affect UX, not security or data integrity | >= 70% | TBD |

## Per-Module Risk Assessment

| Module | Tier | Blast Radius | Security Sensitivity | Implementation Complexity | Test Priority |
|--------|------|-------------|---------------------|--------------------------|--------------|
| frame | CRITICAL | high | high (DI-001, DI-007) | medium | P0 |
| hmac | CRITICAL | high | high (DI-006) | low | P0 |
| admission | CRITICAL | high | high (DI-006, DI-005) | medium | P0 |
| session-auth | CRITICAL | high | high (DI-010) | low | P0 |
| routing | CRITICAL | high | high (DI-005) | medium | P0 |
| halfchannel | HIGH | medium | medium (DI-008) | high | P0 |
| arq | HIGH | medium | low | high | P0 |
| replay | HIGH | medium | low | medium | P0 |
| multipath | HIGH | medium | low | medium | P0 |
| paths | HIGH | medium | low | medium | P1 |
| metrics | HIGH | low | low | low | P1 |
| tmux-control | HIGH | medium | medium (content path) | high | P1 |
| discovery | MEDIUM | low | low | medium | P1 |
| svtn-mgmt | MEDIUM | medium | medium | medium | P2 |
| drain | MEDIUM | medium | low | medium | P2 |
| config | MEDIUM | low | low | low | P0 |
| sbctl | LOW | low | low | low | P2 |

## Classification Summary

| Tier | Module Count | Percentage |
|------|-------------|------------|
| CRITICAL | 5 | 29% |
| HIGH | 8 | 47% |
| MEDIUM | 4 | 24% |
| LOW | 1 | 6% |
| **Total** | **18** | **~100%** (note: 5+8+4+1=18 modules) |

Note: The high percentage of CRITICAL and HIGH modules reflects the nature of the product — a security-sensitive network infrastructure component where content separation and admission correctness are primary requirements. This distribution is expected.

## Build Order

See `.factory/specs/architecture/ARCH-08-dependency-graph.md` for the canonical topological order.

## Implementation Priority Order

1. **frame** — All other modules depend on correct wire format
2. **hmac** — Security foundation; required for admission and routing
3. **admission** — Required before any sessions are possible
4. **routing** — Required for multi-node operation
5. **session-auth** — Required for Tier 2 access control
6. **halfchannel** — Core session protocol; P0 user experience
7. **arq** — Downstream reliability; P0 terminal correctness
8. **replay** — Upstream reliability; P0 keystroke delivery
9. **multipath** — Duplicate suppression and path dispatch
10. **tmux-control** — Session content integration
11. **paths** — Path quality metrics for path selection
12. **metrics** — Quality indicator for operator visibility
13. **config** — Config parsing for operator usability
14. **discovery** — Session enumeration (PE phase)
15. **svtn-mgmt** — SVTN lifecycle management
16. **drain** — Graceful router shutdown (PE phase)
17. **sbctl** — Operator CLI (builds on all above)
18. **others** — Log formatting, test harness, etc.

## Cross-Cutting Concerns by Tier

| Concern | CRITICAL modules | HIGH modules | MEDIUM/LOW modules |
|---------|-----------------|-------------|-------------------|
| Error handling | Return errors; no panics; no log.Fatal | Return errors; propagate with context | Return errors; log non-fatal |
| Logging | Structured (key/value); no private key bytes | Structured; no session content | Structured; INFO level default |
| Authentication | Every operation authenticated; fail-closed | Authentication checked at entry | Authentication at CLI level |
| Concurrency | Mutex-protected; no internal pointer leaks | EWMA updates under mutex | Single-threaded OK |
| Testing | Property tests + unit + integration | Unit + integration | Unit tests sufficient |
