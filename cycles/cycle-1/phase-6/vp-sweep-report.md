---
artifact_id: P6-VPSWEEP-BURST-2
document_type: phase-6-evidence
phase: phase-6-formal-hardening
lane: vp-corpus-adjudication
producer: formal-verifier (phase6-vpsweep)
coordinator: team-lead
timestamp: 2026-07-06T04:15:00Z
adjudicated_against: develop f09fe73
commits: [e047d21, 0ae16ce, 4155b80, d43511d]
status: complete
---

# Phase 6 — VP-Corpus Sweep (Burst 2)

NOTE (coordinator): this report is compiled FROM THE COMMITTED VP FILES and
batch commit bodies, which the coordinator verified directly. The sweep
agent's message-channel summary contained list-level inaccuracies
(reconstructed from memory); the files are authoritative.

## Totals (77 VP files + 2 VP-INDEX-only registrations = 79)

| Ruling | Count |
|--------|-------|
| PROVEN — verification_lock: true, proof_completed_date 2026-07-06, cited evidence at file:test-name vs f09fe73 | 55 |
| PARTIAL — evidence gap documented, lock stays false | 14 |
| UNPROVEN-BLOCKED — blocker documented, lock stays false | 8 |
| Index-only (VP-TBD-ACC, VP-VW6.NN — no files) | 2 |

## PARTIAL (burst-3 work queue) — from committed adjudications

| VP | Gap |
|----|-----|
| VP-025 | proptest deferred (deterministic sweep shipped) — drop-cache capacity |
| VP-026 | proptest deferred — path-score transitivity |
| VP-028 | proof-method drift + spec mismatch — config validation tick_interval |
| VP-029 | proof-method drift + spec mismatch — config required fields |
| VP-031 | real-tmux e2e deferred (hermetic fake-injected coverage present) |
| VP-032 | real-PTY e2e deferred (hermetic coverage present) |
| VP-040 | <2s e2e failover bound deferred (path-tracker inactivation proven) |
| VP-044 | multicast wire deferred → S-BL.DISCOVERY-WIRE (in-process registry proven) |
| VP-045 | real-socket PC-3 deferred → S-BL.DISCOVERY-WIRE |
| VP-046 | e2e ConnectWithKey harness absent (unit discharge present) |
| VP-051 | gopter proptest deferred (deterministic Phase-3 test shipped) |
| VP-053 | gopter proptest deferred (deterministic K=20 sweep shipped) |
| VP-056 | observer-continuity + re-attach + InjectDownstream helper absent |
| VP-062 | fuzz harness absent (FuzzSbctlMetricsJSON does not exist; integration coverage anchors VP-047) |

## UNPROVEN-BLOCKED — by blocker

- internal/testenv e2e infra absent: VP-033, VP-034, VP-037, VP-038
- testenv.ConnectWithSourceIP (multi-host): VP-036 (already lifecycle deferred)
- testenv.CreateSVTN + AttachProbe (multi-SVTN): VP-039 (already lifecycle deferred)
- S-BL.BENCH benchmark harness: VP-041, VP-042 (matches OBS-VP-BENCH drift row)

## Spec findings (routed to spec-steward queue)

1. VP-028/VP-029 property statements assume a decomposed config API
   (RouterConfig{TickIntervalUpstream,...} + ValidateRouter/Access/Console/
   Control) that does not match the shipped internal/config surface
   (monolithic config.Config with Config.Validate()). Statements need
   re-anchoring to the real API; underlying intent partially discharged.

## Process observation (coordinator)

Agent's message-channel final report diverged from its committed artifacts
(wrong VPs in partial/blocked lists, confabulated spec findings) while the
commits themselves were precise. Lesson: orchestrator decisions must be built
from committed artifacts, not agent self-reports — sibling of the
green-claim evidence-paste class (drbothen/vsdd-factory#513). Candidate
upstream note pending agent's confirmation reply.
