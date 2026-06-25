---
artifact_id: VP-INDEX
document_type: verification-property-index
level: L4
version: "1.1"
status: draft
producer: architect
timestamp: 2026-06-23T00:00:00
phase: 1b
inputDocuments:
  - '.factory/specs/behavioral-contracts/BC-INDEX.md'
  - '.factory/specs/architecture/ARCH-07-verification-architecture.md'
  - '.factory/specs/architecture/ARCH-11-verification-coverage-matrix.md'
traces_to: '.factory/specs/architecture/ARCH-INDEX.md'
---

# VP-INDEX: Verification Properties

> Master index. One row per VP. Full contracts in individual VP-NNN files.
> Source of truth for VP catalog, IDs, modules, tools, phases, and counts.
> VP-INDEX total MUST equal: sum of per-tool counts = VP row count.

## Index

| VP ID | Title | BC(s) | Module | Method | Phase | Status | File |
|-------|-------|-------|--------|--------|-------|--------|------|
| VP-001 | ParseOuterHeader / EncodeOuterHeader round-trip | BC-2.01.004 | internal/frame | proptest | P0 | draft | VP-001.md |
| VP-002 | ParseOuterHeader rejects version mismatch | BC-2.01.004 | internal/frame | proptest | P0 | draft | VP-002.md |
| VP-003 | EncodeOuterHeader produces exactly 44 bytes | BC-2.01.004 | internal/frame | proptest | P0 | draft | VP-003.md |
| VP-004 | ComputeHMAC / VerifyHMAC consistency | BC-2.05.005 | internal/hmac | proptest | P0 | draft | VP-004.md |
| VP-005 | VerifyHMAC rejects single-bit flips | BC-2.05.005 | internal/hmac | fuzz | P0 | draft | VP-005.md |
| VP-006 | VerifyHMAC rejects wrong key | BC-2.05.005 | internal/hmac | proptest | P0 | draft | VP-006.md |
| VP-007 | Admission: private key never in wire structs | BC-2.05.001, BC-2.05.007 | internal/admission | proptest | P0 | implemented | VP-007.md |
| VP-008 | Admission fails for unregistered key | BC-2.05.001, BC-2.05.002 | internal/admission | proptest | P0 | implemented | VP-008.md |
| VP-009 | Admission rejects replayed nonce | BC-2.05.001 | internal/admission | proptest | P0 | implemented | VP-009.md |
| VP-010 | SVTNRoute never delivers to wrong SVTN | BC-2.05.006 | internal/routing | proptest | P0 | implemented | VP-010.md |
| VP-011 | Split-horizon: no forward toward arrival interface | BC-2.02.008 | internal/routing | proptest | P0 | draft | VP-011.md |
| VP-012 | SessionAuth rejects unauthorized console key | BC-2.05.003, BC-2.04.003 | internal/session | proptest | P0 | draft | VP-012.md |
| VP-013 | SessionAuth rejects upstream from read-only key | BC-2.04.005, BC-2.05.003 | internal/session | proptest | P0 | draft | VP-013.md |
| VP-014 | DeriveNodeAddress is deterministic | BC-2.01.006 | internal/frame | proptest | P0 | draft | VP-014.md |
| VP-015 | Router code never parses channel header payload | BC-2.01.005 | internal/routing | fuzz | P0 | draft | VP-015.md |
| VP-016 | HalfChannel.Tick emits exactly one frame per tick | BC-2.01.001, BC-2.01.003 | internal/halfchannel | proptest | P0 | draft | VP-016.md |
| VP-017 | HalfChannel sequence increments by exactly 1 | BC-2.01.003 | internal/halfchannel | proptest | P0 | draft | VP-017.md |
| VP-018 | HalfChannel emits empty frame when no payload | BC-2.01.001, BC-2.01.002 | internal/halfchannel | proptest | P0 | draft | VP-018.md |
| VP-019 | ARQ.OnAck never delivers a frame twice | BC-2.02.005 | internal/arq | proptest | P0 | draft | VP-019.md |
| VP-020 | ARQ delivers frames in-order | BC-2.02.005 | internal/arq | proptest | P0 | draft | VP-020.md |
| VP-021 | ARQ.TLPKTDROP triggers only when frame overdue | BC-2.02.006 | internal/arq | proptest | P0 | draft | VP-021.md |
| VP-022 | Replay.OnUpstream never delivers same seq twice | BC-2.02.004 | internal/replay | proptest | P0 | draft | VP-022.md |
| VP-023 | Replay delivers keystrokes in sequence order | BC-2.02.004 | internal/replay | proptest | P0 | draft | VP-023.md |
| VP-024 | Multipath delivers first copy, discards duplicates | BC-2.02.001, BC-2.02.002 | internal/multipath | proptest | P0 | draft | VP-024.md |
| VP-025 | DropCache never exceeds configured capacity | BC-2.02.009 | internal/multipath | proptest | P0 | draft | VP-025.md |
| VP-026 | PathScore ranking is transitive | BC-2.02.003 | internal/paths | proptest | P0 | draft | VP-026.md |
| VP-027 | QualityIndicator transitions: degradation only goes down | BC-2.06.001, BC-2.06.002 | internal/metrics | proptest | P1 | draft | VP-027.md |
| VP-028 | Config.Validate rejects out-of-range tick_interval | BC-2.09.003 | internal/config | proptest | P0 | draft | VP-028.md |
| VP-029 | Config.Validate rejects missing required fields | BC-2.09.003 | internal/config | proptest | P0 | draft | VP-029.md |
| VP-030 | sbctl exits 1 with E-NET-001 on connection refused | BC-2.07.003 | cmd/sbctl | integration | P0 | draft | VP-030.md |
| VP-031 | tmux control mode: 99% output event completeness | BC-2.04.001 | internal/tmux | integration | P1 | draft | VP-031.md |
| VP-032 | PTY fallback activates on control mode failure | BC-2.04.002 | internal/tmux | integration | P0 | draft | VP-032.md |
| VP-033 | Console attach/detach lifecycle | BC-2.04.003, BC-2.04.004 | internal/session | e2e | P1 | draft | VP-033.md |
| VP-034 | Multi-console fan-out: both consoles receive all frames | BC-2.04.006 | internal/session | e2e | P0 | draft | VP-034.md |
| VP-035 | Read-only console: upstream rejected by access node | BC-2.04.005 | internal/session | integration | P1 | draft | VP-035.md |
| VP-036 | Session continuity across IP address change | BC-2.01.007 | internal/admission | e2e | P0 | deferred | VP-036.md |
| VP-037 | Router drain: nodes migrate within 2s | BC-2.09.002 | internal/drain | e2e | P2 | draft | VP-037.md |
| VP-038 | E→PE graduation: config change only | BC-2.09.001 | internal/config | e2e | P2 | draft | VP-038.md |
| VP-039 | SVTN isolation: no cross-SVTN frame delivery | BC-2.05.006 | internal/routing | e2e | P0 | deferred | VP-039.md |
| VP-040 | Multipath failover: recovery < 2s | BC-2.02.003 | internal/multipath | e2e | P1 | draft | VP-040.md |
| VP-041 | Tick regularity: p99 jitter ≤ 2ms | BC-2.01.001 | internal/halfchannel | benchmark | P0 | draft | VP-041.md |
| VP-042 | Keystroke-to-echo: p99 ≤ 100ms | BC-2.01.001, BC-2.02.001 | internal/halfchannel | benchmark | P0 | draft | VP-042.md |
| VP-043 | XOR FEC: single loss in group recoverable | BC-2.02.007 | internal/arq | proptest | P1 | draft | VP-043.md |
| VP-044 | Presence advertisement includes required fields | BC-2.03.001, BC-2.03.003 | internal/discovery | integration | P1 | draft | VP-044.md |
| VP-045 | Console session enumeration without hostnames | BC-2.03.002 | internal/discovery | e2e | P1 | draft | VP-045.md |
| VP-046 | Key lifecycle: register/revoke/expire | BC-2.05.004 | internal/svtnmgmt | integration | P1 | draft | VP-046.md |
| VP-047 | Per-path metrics queryable via sbctl | BC-2.06.003 | internal/metrics | integration | P1 | draft | VP-047.md |
| VP-048 | Control node creates/destroys SVTNs | BC-2.07.001 | internal/svtnmgmt | integration | P2 | draft | VP-048.md |
| VP-049 | sbctl unified CLI with OpenSSH auth | BC-2.07.002 | cmd/sbctl | e2e | P2 | draft | VP-049.md |
| VP-050 | Console remotely controllable via sbctl | BC-2.08.001 | cmd/sbctl | e2e | P1 | draft | VP-050.md |
| VP-051 | HalfChannel independence: B unaffected by A's frame production | BC-2.01.003 | internal/halfchannel | proptest | P0 | draft | VP-051.md |
| VP-052 | Missing expected tick within deadline → indicator downgrade | BC-2.06.002 | internal/metrics | integration | P1 | draft | VP-052.md |
| VP-053 | Empty-tick frame sequence: K ticks → K frames with contiguous seq nums | BC-2.01.002 | internal/halfchannel | proptest | P0 | draft | VP-053.md |
| VP-054 | Receiver dedup: first-arriving copy delivered, duplicate discarded silently | BC-2.02.002 | internal/multipath | integration | P0 | draft | VP-054.md |
| VP-055 | Presence advertisement payload round-trip: required fields present and stable | BC-2.03.003 | internal/discovery | proptest | P1 | draft | VP-055.md |
| VP-056 | Console detach releases session without closing it; observers unaffected | BC-2.04.004 | internal/session | integration | P1 | draft | VP-056.md |
| VP-057 | Node private key bytes absent from all emitted frame types (sampling + HKDF sketch) | BC-2.05.007 | internal/admission | proptest | P0 | implemented | VP-057.md |
| VP-058 | RouteFrame calls verifyFrameHMAC before IsAdmitted and SVTNRoute | BC-2.05.008 | internal/routing | code-audit | P0 | draft | VP-058.md |

## Counts

| Total VPs | Proptest | Fuzz | Integration | E2E | Benchmark | Code-Audit |
|-----------|---------|------|-------------|-----|-----------|------------|
| 58 | 32 | 2 | 11 | 10 | 2 | 1 |

> Arithmetic check: 32 + 2 + 11 + 10 + 2 + 1 = 58. Consistent.

## Phase Distribution

| Phase | Count |
|-------|-------|
| P0 | 40 |
| P1 | 14 |
| P2 | 4 |
| **Total** | **58** |

> Phase recounted: VP-058 (P0) added. P0 = 39+1 = 40. P1 = 14. P2 = 4. Total = 58.

## BC Coverage Check

43 BCs total. 42 have at least 1 VP. BC-2.05.008 has VP-058 (draft, Wave 3). Zero coverage gaps.
