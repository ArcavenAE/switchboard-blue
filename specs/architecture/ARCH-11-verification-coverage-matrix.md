---
artifact_id: ARCH-11-verification-coverage-matrix
document_type: architecture-section
level: L3
version: "1.11"
status: draft
producer: architect
timestamp: 2026-06-29T00:00:00
phase: 1b
traces_to: ARCH-INDEX.md
inputDocuments:
  - '.factory/specs/behavioral-contracts/BC-INDEX.md'
  - '.factory/specs/architecture/ARCH-07-verification-architecture.md'
kos_anchors: []
modified:
  - 2026-06-23T00:00:00
  - 2026-06-26T12:00:00 # v1.1 — WG3-H-001: Update totals to 43 BCs / 58 VPs; add missing BC-2.05.008 → VP-058 trace row; update internal/routing VP count from 4 to 5 (add code-audit)
  - 2026-06-27T00:00:00 # v1.2 — Wave 3 gate F-2 adjudication: add VP-059 (FailureCounter threshold proptest, internal/admission); update totals to 59 VPs; update internal/admission VP count from 5 to 6
  - 2026-06-27T00:00:00 # v1.3 — BC-2.04.007 registration: add VP-060 (daemon lifecycle integration, cmd/switchboard); update totals to 60 VPs; add cmd/switchboard row (0→1 VP); update BC coverage 43→44 BCs
  - 2026-06-28T00:00:00 # v1.4 — VP-id assignment: add VP-061 (code-audit, internal/metrics) and VP-062 (fuzz, cmd/sbctl) for BC-2.06.003; update totals to 62 VPs
  - 2026-06-28T00:00:00 # v1.5 — F-002 + F-007: mint VP-063 (proptest, internal/paths) for BC-2.02.003 PC-5 degraded-flag boolean; fix stale "60 VPs total" prose to 63; update totals to 63 VPs
  - 2026-06-29T00:00:00 # v1.6 — BC-2.07.004 v1.3 Wave-5 Convergence Rulings A–E VP assignment: VP-068–VP-073 added; BC-2.07.004 row updated with full VP set; per-module counts updated; totals updated to 73 VPs
  - 2026-06-30T00:00:00 # v1.11 — F-P7L3-001: VP-075 module corrected from internal/mgmt to cmd/switchboard; BC-2.05.004 row modules cell updated; internal/mgmt count 9→8 (integration 7→6), cmd/switchboard count 2→3 (integration 2→3). Per-module sum unchanged at 75.
  - 2026-06-30T00:00:00 # v1.10 — F-T3-301: VP-074 (P1, BC-2.06.001 threshold classification) added; P1 VPs 17→18.
  - 2026-06-30T00:00:00 # v1.9 — PO Ruling 3 (S-5.02 Pass-4 scope ruling): VP-047 implementing_story transferred S-5.02 → S-W5.04 per vp_index_is_vp_catalog_source_of_truth policy. No BC→VP row changes; no count changes.
  - 2026-06-30T00:00:00 # v1.8 — Pass-2 lens-3 F-T3-003: VP-075 minted for BC-2.05.004 handler authority (integration, internal/mgmt); BC-2.05.004 row updated VP-046→VP-046+VP-075; internal/mgmt count 8→9; P0 VPs 52→53; totals updated to 75 VPs
  - 2026-06-29T00:00:00 # v1.7 — VP-074 added for BC-2.06.001 threshold classification (unit, internal/metrics); internal/metrics count 4→5; totals updated to 74 VPs
---

# ARCH-11: Verification Coverage Matrix

> Every BC must have at least one VP. This matrix is the coverage guarantee.
> VP-INDEX.md is the authoritative VP catalog; this section cross-references it.
> Total VP count: 75 (VP-001 through VP-075, per VP-INDEX v2.7).

## BC → VP Coverage Table

| BC ID | Title (abbreviated) | Module | VP(s) | Method | Phase |
|-------|---------------------|--------|-------|--------|-------|
| BC-2.01.001 | Timeslice clock fires on every tick | internal/halfchannel | VP-016, VP-018, VP-041, VP-042 | proptest | P0 |
| BC-2.01.002 | Empty-tick frame is a valid liveness signal | internal/halfchannel | VP-018, VP-053 | proptest | P0 |
| BC-2.01.003 | Independent upstream/downstream half-channels | internal/halfchannel | VP-017, VP-051 | proptest | P0 |
| BC-2.01.004 | 44-byte outer header encoding and decoding | internal/frame | VP-001, VP-002, VP-003 | proptest | P0 |
| BC-2.01.005 | Channel header opaque to routers | internal/routing | VP-015 | fuzz + audit | P0 |
| BC-2.01.006 | Session identity cryptographic derivation | internal/frame | VP-014 | proptest | P0 |
| BC-2.01.007 | Session continuity across IP change | internal/admission | VP-036 | e2e | P0 |
| BC-2.02.001 | Duplicate-and-race: same frame on two paths | internal/multipath | VP-024 | proptest | P0 |
| BC-2.02.002 | First-arriving copy delivered, duplicates discarded | internal/multipath | VP-024, VP-054 | proptest + integration | P0 |
| BC-2.02.003 | Per-path RTT/loss tracked, paths ranked | internal/paths | VP-026, VP-063 | proptest | P0 |
| BC-2.02.004 | Upstream idempotent replay window | internal/replay | VP-022, VP-023 | proptest | P0 |
| BC-2.02.005 | Downstream ARQ with piggybacked ACK/SACK | internal/arq | VP-019, VP-020 | proptest | P0 |
| BC-2.02.006 | TLPKTDROP terminates overdue downstream frames | internal/arq | VP-021 | proptest | P0 |
| BC-2.02.007 | XOR parity FEC, single loss recoverable | internal/arq | VP-043 | proptest | P1/PE |
| BC-2.02.008 | Split-horizon: no forward back toward arrival | internal/routing | VP-011 | proptest | P0 |
| BC-2.02.009 | Bounded drop cache suppresses looping duplicates | internal/multipath | VP-025 | proptest | P0 |
| BC-2.03.001 | Access node presence advertisement | internal/discovery | VP-044 | integration | P1/PE |
| BC-2.03.002 | Console session enumeration without hostnames | internal/discovery | VP-045 | e2e | P1/PE |
| BC-2.03.003 | Presence includes name, status, quality | internal/discovery | VP-044, VP-055 | integration + proptest | P1/PE |
| BC-2.04.001 | Access node connects to tmux control mode | internal/tmux | VP-031 | integration | P0 |
| BC-2.04.002 | PTY fallback when control mode unavailable | internal/tmux | VP-032 | integration | P0 |
| BC-2.04.003 | Console attach by name | internal/session | VP-033 | e2e | P0 |
| BC-2.04.004 | Console detach without closing session | internal/session | VP-033, VP-056 | e2e + integration | P0 |
| BC-2.04.005 | Read-only console rejects upstream keystrokes | internal/session | VP-013, VP-035 | proptest + integration | P0 |
| BC-2.04.006 | Multi-console fan-out | internal/session | VP-034 | e2e | P0 |
| BC-2.04.007 | Daemon startup exits non-zero on connect failure; SIGTERM/SIGINT triggers clean shutdown | cmd/switchboard | VP-060 | integration | P0 |
| BC-2.05.001 | Tier 1 SVTN admission via signed key challenge | internal/admission | VP-007, VP-009 | proptest | P0 |
| BC-2.05.002 | Router rejects non-admitted nodes — fail-closed | internal/admission | VP-008 | proptest | P0 |
| BC-2.05.003 | Tier 2 authorization enforced by access node | internal/session | VP-012, VP-013 | proptest | P0 |
| BC-2.05.004 | Key lifecycle: register, revoke, expire | internal/svtnmgmt, cmd/switchboard | VP-046, VP-075 | integration | P0 |
| BC-2.05.005 | HMAC frame authentication at first router | internal/hmac, internal/admission (PC-3) | VP-004, VP-005, VP-006, VP-059 | proptest + fuzz | P0 |
| BC-2.05.006 | SVTN cryptographic isolation | internal/routing | VP-010, VP-039 | proptest + e2e | P0 |
| BC-2.05.007 | Private keys never transit the network | internal/admission | VP-007, VP-057 | proptest + audit | P0 |
| BC-2.05.008 | RouteFrame HMAC enforcement | internal/routing | VP-058, VP-059 | code-audit + proptest | P0 |
| BC-2.06.001 | Quality indicator derived from latency/loss | internal/metrics | VP-027, VP-074 | proptest + unit | P1 |
| BC-2.06.002 | Missing frame triggers indicator downgrade | internal/metrics | VP-027, VP-052 | proptest + integration | P1 |
| BC-2.06.003 | Per-path RTT/loss queryable via sbctl | internal/metrics, cmd/sbctl | VP-047, VP-061, VP-062 | integration + code-audit + fuzz | P1 |
| BC-2.07.001 | Control node creates/destroys SVTNs | internal/svtnmgmt | VP-048 | integration | P2 |
| BC-2.07.002 | sbctl unified CLI with OpenSSH auth | cmd/sbctl | VP-049 | e2e | P2 |
| BC-2.07.003 | sbctl reports clear error when daemon unreachable | cmd/sbctl | VP-030 | integration | P0 |
| BC-2.07.004 | Daemon management server authenticates all connections via Ed25519 challenge-response (fail-closed) | internal/mgmt, cmd/switchboard | VP-064, VP-065, VP-066, VP-068, VP-069, VP-070, VP-071, VP-072, VP-073 | integration + unit + fuzz | P0 |
| BC-2.08.001 | Console remotely controllable via sbctl | cmd/sbctl | VP-050 | e2e | P1/PE |
| BC-2.09.001 | E→PE graduation by config change | internal/config | VP-038 | e2e | P2/PE |
| BC-2.09.002 | Router sends drain signal before shutdown | internal/drain | VP-037 | e2e | P2/PE |
| BC-2.09.003 | Router startup fails cleanly on malformed config | internal/config | VP-028, VP-029 | proptest | P0 |

## Coverage Summary

| Metric | Value |
|--------|-------|
| Total BCs | 45 |
| BCs with ≥1 VP | 45 |
| BCs with 0 VPs | 0 |
| Total unique VPs | 75 |
| P0 VPs | 53 |
| P1 VPs | 18 |
| P2+ VPs | 4 |

## Per-Module VP Count

VP counts recounted from VP-INDEX (canonical source of truth, 75 VPs total).

| Module | VP Count | Methods |
|--------|---------|---------|
| internal/frame | 4 | proptest (4) |
| internal/hmac | 3 | proptest (2), fuzz (1) |
| internal/halfchannel | 7 | proptest (5), benchmark (2) |
| internal/arq | 4 | proptest (4) |
| internal/replay | 2 | proptest (2) |
| internal/multipath | 4 | proptest (2), e2e (1), integration (1) |
| internal/paths | 2 | proptest (2) |
| internal/metrics | 5 | proptest (1), integration (2), code-audit (1), unit (1) |
| internal/admission | 6 | proptest (5), e2e (1) |
| internal/routing | 5 | proptest (2), fuzz (1), e2e (1), code-audit (1) |
| internal/session | 6 | proptest (2), e2e (2), integration (2) |
| internal/tmux | 2 | integration (2) |
| internal/config | 3 | proptest (2), e2e (1) |
| internal/discovery | 3 | integration (1), e2e (1), proptest (1) |
| internal/svtnmgmt | 2 | integration (2) |
| internal/drain | 1 | e2e (1) |
| internal/mgmt | 8 | unit (1), fuzz (1), integration (6) |
| cmd/sbctl | 5 | integration (2), e2e (2), fuzz (1) |
| cmd/switchboard | 3 | integration (3) |
| **Total** | **75** | |

Per-module sum = 75 (no off-table VPs).
VP-059 (proptest, internal/admission) added 2026-06-27. VP-060 (integration, cmd/switchboard) added 2026-06-27.
VP-061 (code-audit, internal/metrics) and VP-062 (fuzz, cmd/sbctl) added 2026-06-28 for BC-2.06.003.
VP-063 (proptest, internal/paths) added 2026-06-28 for BC-2.02.003 PC-5 degraded-flag boolean.
VP-064 (integration, internal/mgmt), VP-065 (integration, internal/mgmt), VP-066 (fuzz, internal/mgmt) added 2026-06-28 for BC-2.07.004 Wave-5.
VP-067 (integration, cmd/sbctl) added 2026-06-28 for BC-2.07.002 Wave-5.
VP-068–VP-073 added 2026-06-29 for BC-2.07.004 v1.3 Wave-5 Convergence Rulings A–E:
  VP-068 (unit, internal/mgmt), VP-069 (integration, internal/mgmt), VP-070 (integration, internal/mgmt),
  VP-071 (integration, internal/mgmt), VP-072 (integration, internal/mgmt), VP-073 (integration, cmd/switchboard).
VP-074 (unit, internal/metrics) added 2026-06-29 for BC-2.06.001 threshold classification (L-001 disambiguation; unit test covering all 6 nominal regions + 8 boundary values).
VP-075 (integration, cmd/switchboard) added 2026-06-30 for BC-2.05.004 handler-layer caller-role enforcement (Pass-2 lens-3 F-T3-003). VP-046 anchored internal/svtnmgmt (key store propagation); VP-075 anchored cmd/switchboard (BuildAdminHandlers authority gate). F-P7L3-001 (2026-06-30): module corrected from internal/mgmt to cmd/switchboard; internal/mgmt 9→8 (integration 7→6), cmd/switchboard 2→3 (integration 2→3).

## Zero-VP BCs Check

Per the coverage table above, all 45 BCs have at least one VP. No gaps.

VP-053 through VP-057 were added in Phase 1c-refinement to close coverage gaps
identified by the PO sweep (BC-2.01.002, BC-2.02.002, BC-2.03.003, BC-2.04.004,
BC-2.05.007 previously lacked a VP with `source_bc:` pointing at them).

VP-059 was added 2026-06-27 for BC-2.05.005 PC-3 (per-source HMAC failure rate alert).
BC-2.05.005 PC-3 previously had no dedicated VP — VP-004/005/006 cover the HMAC
primitive but not the FailureCounter threshold behavior. VP-059 closes this gap.

VP-060 was added 2026-06-27 for BC-2.04.007 (daemon lifecycle: connect-failure exit
and clean SIGTERM/SIGINT shutdown). BC-2.04.007 is a new BC authored by the PO for
Wave 3 scope. VP-060 covers both lifecycle postcondition paths via integration testing
with subprocess launch and OS signal delivery.

## Infeasible Properties

No BCs are flagged as verification-infeasible with the chosen Go-native toolchain.
The fuzz + audit approach for VP-015 (channel header opacity at router) is the
most complex verification: it requires a CI scan that the router code path has no
`channel_header` type assertion or field access. This is feasible via `grep` + CI
gate, supplemented by fuzz testing that sends malformed channel headers and verifies
the router does not crash or behave differently.
