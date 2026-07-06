---
artifact_id: P6-GAPCLOSE-BURST-3
document_type: phase-6-evidence
phase: phase-6-formal-hardening
lane: vp-gap-close
producer: formal-verifier (phase6-gapclose) + spec-steward (phase6-specfix)
coordinator: team-lead
timestamp: 2026-07-06T06:15:00Z
merged_as: 0516f3a (PR #106, squash)
lock_commits: [2f50fcd, 8ccca91, bdde10e]
status: complete
---

# Phase 6 — Burst 3: VP Gap-Close (Lanes A + B)

## Lane A (tests, PR #106 → 0516f3a)

Five test-only files, 1140 lines. Six VPs discharged:

| VP | Discharging tests | Mutation-kill check |
|----|-------------------|---------------------|
| VP-025 | multipath_prop_test.go: TestProp_VP025_DropCache_NeverExceedsCapacity + AddIfAbsent sibling | eviction fence >=→> fails at cap=1 |
| VP-026 | paths_vp026_prop_test.go: TestProp_VP026_PathScore_Transitive + TotalOrderConsistency | non-monotonic score factor fails triples |
| VP-051 | halfchannel_prop_test.go: TestProp_VP051_HalfChannelIndependence | shared seq counter fails B.Seq()==0 |
| VP-053 | halfchannel_prop_test.go: TestProp_VP053_EmptyTickSequence | missing seq increment fails contiguity |
| VP-056 | vp056_test.go: Detach_PublisherRetainsSession + ObserverContinues + SameKeyReAttach | 3 mutants (Unpublish-on-detach, skipped map delete, retained entry) each fail |
| VP-062 | vp062_fuzz_test.go: FuzzSbctlMetricsJSON (90s, 1,815,095 execs, 0 crashes) + SeedsAlone CI companion | 4 mutants (PendingKind branch, router_addr tag, control-byte emit, alias field) each fail |

Skeleton-vs-shipped-API adaptations recorded in changelogs (VP-056 real
AccessNode/Publisher/ConsoleSet API, file-local injectDownstream — no new
production surface; VP-062 QualityFromEntry + RTTKind discrimination).
Nothing unprovable-as-stated.

## Lane B (spec re-anchor, 2f50fcd + coordinator lock flip 8ccca91)

VP-028/VP-029 re-anchored from phantom decomposed config API to shipped
config.Config/Validate(); proof_method proptest → table-driven (finite
domains, justified); discharged by existing tests (coordinator-verified
present + passing); citation name-drift in v1.2 changelog corrected at v1.3.

## Corpus census after burst 3 (coordinator-verified from files)

| State | Count | Files |
|-------|-------|-------|
| PROVEN (verification_lock: true) | **63** | — |
| PARTIAL — infra-deferred, justified | 6 | VP-031/032 (real-tmux/PTY e2e), VP-040 (<2s e2e bound), VP-044/045 (S-BL.DISCOVERY-WIRE), VP-046 (e2e key harness) |
| UNPROVEN-BLOCKED — justified | 8 | VP-033/034/037/038 (internal/testenv), VP-036/039 (multi-host/SVTN testenv), VP-041/042 (S-BL.BENCH; OBS-VP-BENCH row) |
| Total files | 77 | + 2 VP-INDEX-only registrations |
