---
artifact_id: nfr-catalog
document_type: prd-supplement-nfr-catalog
level: L3
version: "1.0"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
inputs:
  - '.factory/specs/prd.md'
  - '.factory/specs/domain-spec/L2-INDEX.md'
  - '.factory/specs/domain-spec/risks.md'
  - '.factory/specs/domain-spec/assumptions.md'
  - '_bmad-output/planning-artifacts/prd.md'
input-hash: "[md5-pending]"
traces_to: '.factory/specs/prd.md'
---

# Non-Functional Requirements Catalog: Switchboard

> PRD supplement — extracted from PRD Section 4.
> Referenced by: architect, performance-engineer, formal-verifier.

## NFR Registry

| ID | Category | Requirement | Target | Validation Method | Priority | Risk Source |
|----|----------|-------------|--------|------------------|----------|-------------|
| NFR-001 | Performance | Keystroke-to-echo latency (p99) over single-hop LAN | ≤ 100ms p99 at tick intervals 5–50ms | Benchmark: measure keystroke timestamp to terminal echo at console over LAN; compare to raw SSH baseline at same tick rates; target: at least one tick rate meets goal | P0 | R-002, ASM-001 |
| NFR-002 | Performance | Frame processing loop jitter on developer laptop | p99 timer jitter ≤ 5ms per tick cycle under normal OS workload | Benchmark: measure tick timer accuracy over 1000 ticks on macOS and Linux under typical dev workload; measure GC pause duration via runtime/metrics | P0 | R-003, ASM-002 |
| NFR-003 | Reliability | Multi-path failover time when one path fails | < 2 seconds from path failure detection to session traffic on alternate path | Chaos test: kill one router in multi-path setup; measure time to session recovery; must be < 2s | P0 | DEC-003 |
| NFR-004 | Scalability | Concurrent sessions per E router instance | ≥ 1,000 concurrent active sessions with < 10% overhead per additional 100 sessions | Load test: ramp to 1,000 concurrent sessions; measure CPU and memory; verify no session degradation | P1 | N/A |
| NFR-005 | Security | Session content opacity at router boundary | 0 bytes of session payload visible at router in any code path | Code audit: verify router code has no parser for channel header or payload; integration test: capture all data at router under all conditions including error paths | P0 | R-001, DI-001 |
| NFR-006 | Reliability | Protocol version compatibility: clean rejection on mismatch | Major version mismatch results in E-PRT-001 (not silent corruption or crash) | Interoperability test matrix: test all version combinations; verify clean rejection on incompatible major version | P0 | R-005, DI-007 |
| NFR-007 | Performance | Timeslice framing overhead vs. raw SSH baseline | Timeslice framing adds ≤ 20ms p99 latency compared to raw SSH on same LAN | Benchmark: compare framed vs. raw SSH keystroke-to-echo; measure overhead; target: ≤ 20ms p99 overhead | P0 | R-002, ASM-001 |
| NFR-008 | Security | Private key non-transit | Private key material (bytes) never present in any network frame, log output, or CLI output | Code audit + fuzz: scan all serialization paths for private key leakage; property test: private key bytes do not appear in any output | P0 | DI-002 |
| NFR-009 | Reliability | Empty-tick frame regularity | Empty-tick frames emitted with ≤ 2ms timing deviation from configured tick interval under normal OS conditions | Unit test: measure actual tick intervals over 1000 ticks; p99 deviation ≤ 2ms | P0 | DI-008, ASM-002 |
| NFR-010 | Reliability | Access node tmux control mode reliability under load | ≥ 99% %output event completeness under sustained high-output sessions (10KB/s terminal output) | Stress test: run high-output session (fast ls recursion, make output); measure %output events vs. actual bytes; target: 99% completeness | P1 | R-004, ASM-003 |
| NFR-011 | Reliability | Config error detection at startup | 100% of config validation errors reported before first connection accepted | Unit test: systematically inject each config error type; verify error reported and daemon exits before listening | P0 | FM-010 |
| NFR-012 | Performance | Binary size (static) | ≤ 20MB (uncompressed) for the combined switchboard binary | Build check: measure binary size on each target platform (amd64, arm64); alert if > 20MB | P2 | ASM-004 |
| NFR-013 | Reliability | SVTN cryptographic isolation | Node admitted only to SVTN-A cannot receive any frame from SVTN-B on the same router | Integration test: two SVTNs on same router; verify no cross-SVTN frame delivery under all conditions | P0 | DI-005 |
| NFR-014 | Performance | Quality indicator update latency | Quality indicator updates within 2 tick cycles of path quality change | Integration test: degrade path quality; measure time to indicator update; must be ≤ 2 × tick_interval | P1 | CAP-021 |
| NFR-015 | Reliability | E router single-node E2E: five-minute setup | Operator can complete E router setup (install → first session attached) in ≤ 5 minutes with ≤ 3 CLI commands per machine | Timed onboarding walkthrough with target persona; measure clock time and command count | P1 | ASM-004 |

## NFR Categories

| Category | Description | Validation Agent |
|----------|-------------|-----------------|
| Performance | Throughput, latency, memory, binary size | performance-engineer |
| Security | Auth, encryption, content opacity, private key protection | security-reviewer, formal-verifier |
| Reliability | Uptime, recovery, data integrity, completeness | formal-verifier, e2e-tester |
| Scalability | Concurrent sessions, growth, resource consumption | performance-engineer |

## NFR-to-Module Mapping

| NFR ID | Affected Modules | Architectural Impact |
|--------|-----------------|---------------------|
| NFR-001 | internal/halfchannel, internal/frame | Tick interval must be tunable; benchmark harness required |
| NFR-002 | internal/halfchannel, main event loop | Go runtime GC tuning; avoid heap allocations on hot path |
| NFR-003 | internal/paths, internal/multipath | Keepalive interval must be short enough to detect and fail over in < 2s |
| NFR-004 | router/forwarding, internal/session | Connection tracking must scale; avoid per-session goroutine if possible |
| NFR-005 | router/forwarding | Code review gate: no channel header parser in router; CI scan |
| NFR-006 | internal/frame | Version check must be first operation in frame processing |
| NFR-007 | internal/halfchannel | Measure overhead at build-time; add latency budget to integration tests |
| NFR-008 | internal/admission, key file handling | Private key must be held in zeroing memory; never passed to serializer |
| NFR-009 | internal/halfchannel | Timer implementation: use time.NewTicker with compensation; avoid drift |
| NFR-010 | access/tmux_control | Stress test in CI; measure event completeness under synthetic load |
| NFR-011 | internal/config | Validation must be complete before bind/listen call |
| NFR-012 | build system | ldflags strip debug; UPX optional for distribution |
| NFR-013 | router/svtn_routing | Per-SVTN frame table; SVTN ID check before any forwarding decision |
| NFR-014 | internal/metrics, internal/paths | Metrics pipeline must be low-latency; avoid batching that delays indicator |
| NFR-015 | all daemons, documentation | Onboarding script; minimal required config; sensible defaults |

## Open NFR Questions (Deferred to Architecture)

- **NFR-004 target**: 1,000 concurrent sessions may need revisiting based on Go goroutine cost profiling. Architecture should model session-per-goroutine vs. event-loop approaches.
- **NFR-002 jitter target**: 5ms p99 may be too tight on heavily loaded macOS systems. The 5–50ms tick interval range means 5ms jitter is 10–100% of the tick period. Architecture should validate against ASM-002 probes before hardening.
