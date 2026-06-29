---
artifact_id: dependency-graph
document_type: dependency-graph
level: ops
version: "1.2"
status: draft
producer: story-writer
timestamp: 2026-06-27T00:00:00
phase: 2
inputDocuments:
  - '.factory/stories/STORY-INDEX.md'
  - '.factory/specs/architecture/ARCH-08-dependency-graph.md'
---

# Story Dependency Graph: Switchboard

## Dependency Table

| Story | Depends On | Blocks | Wave | Rationale |
|-------|-----------|--------|------|-----------|
| S-0.01 | (none) | S-1.01, S-1.02, S-2.01, S-4.02 | 0 | Root; BMAD scaffolding baseline |
| S-1.01 | S-0.01 | S-1.02, S-1.03, S-2.01, S-2.02, S-3.01, S-4.01, S-4.02, S-4.03, S-4.04 | 1 | internal/frame is DAG root; everything depends on wire codec |
| S-1.02 | S-1.01 | S-3.01, S-4.02, S-4.03 | 1 | halfchannel imports frame; session content flows through half-channel |
| S-2.01 | S-1.01 | S-2.02, S-4.04 | 2 | hmac imports frame; admission depends on hmac |
| S-2.02 | S-1.01, S-2.01 | S-1.03, S-3.03, S-4.04, S-6.02 | 2 | admission imports frame+hmac; SVTN isolation in routing depends on admission |
| S-3.01 | S-1.02, S-2.02 | S-3.02 | 3 | tmux imports halfchannel+session; admission must be ready for session auth |
| S-3.02 | S-3.01 | S-3.03 | 3 | session fan-out requires tmux stream to be established |
| S-3.03 | S-3.02, S-2.02 | S-6.02 | 3 | Tier-2 auth extends session; requires admission key set from S-2.02 |
| S-4.01 | S-1.01, S-2.02 | S-4.02, S-4.03, S-4.04, S-5.01 | 4 | paths+multipath import frame; routing decisions need admission (split-horizon) |
| S-4.02 | S-1.01, S-1.02 | (none) | 4 | replay imports frame+halfchannel |
| S-4.03 | S-1.01, S-1.02 | S-5.01 | 4 | arq imports frame+halfchannel; TLPKTDROP feeds metrics |
| S-4.04 | S-2.02, S-4.01 | (none) | 4 | split-horizon extends routing (S-2.02); needs paths from S-4.01 for ordering |
| S-5.01 | S-4.01, S-4.03, S-5.03 | S-5.02 | 5 | metrics imports paths (from S-4.01); TLPKTDROP signal from S-4.03; reads Snapshot().Degraded added by S-5.03 (F-005: edge now symmetric in both frontmatters) |
| S-5.02 | S-5.01, S-6.03 | (none) | 5 | sbctl paths list/router metrics/router status alias wires metrics + CLI structure; adds p99 histogram to internal/paths |
| S-5.03 | S-4.01 | S-5.01 | 5 | adds degraded flag + PathSnapshot + Snapshot() to internal/paths; S-5.01 consumes Snapshot().Degraded |
| S-6.01 | S-1.01 | S-6.02, S-6.03 | 4 | config imports nothing internal; needed before daemon management |
| S-6.02 | S-2.02, S-6.01, S-6.03 | S-5.02 | 5 | svtnmgmt imports admission+config; adds cmd/sbctl/admin.go into scaffold from S-6.03; key lifecycle before metrics CLI |
| S-6.03 | S-6.01 | S-5.02, S-6.02 | 5 | sbctl base structure + main.go scaffold; must exist before admin.go (S-6.02) and paths_list.go (S-5.02) are added |
| S-7.01 | S-4.03 | (none) | 6 | FEC extends arq (from S-4.03) |
| S-7.02 | S-2.02, S-3.02 | (none) | 6 | discovery imports routing (S-2.02) and session presence state (S-3.02) |
| S-7.03 | S-3.02, S-6.03 | (none) | 6 | console remote control uses session (S-3.02) and sbctl CLI (S-6.03) |
| S-7.04 | S-6.01, S-4.04 | (none) | 6 | PE graduation extends config (S-6.01); drain imports routing (S-4.04) |

## Topological Sort (Root → Leaf)

```
Wave 0: S-0.01
Wave 1: S-1.01, S-1.02                   (both depend only on S-0.01 or each other)
Wave 2: S-2.01, S-2.02                   (depend on Wave 1)
Wave 3: S-3.01, S-3.02, S-3.03           (depend on Wave 1+2)
Wave 4: S-4.01, S-4.02, S-4.03, S-4.04, S-6.01  (depend on Wave 1+2; S-6.01 depends only on S-1.01)
Wave 5: S-5.01, S-5.02, S-5.03, S-6.02, S-6.03  (depend on Wave 4)
Wave 6: S-7.01, S-7.02, S-7.03, S-7.04  (depend on Wave 3+4+5)
```

## Cycle-Freeness Verification

Manual topological sort confirms no back-edges:
- No story in Wave N depends on a story in Wave M where M > N.
- S-5.02 depends on S-6.03 and S-5.01 — both Wave 5 stories. S-5.02 is placed last in Wave 5 (after S-6.03).
- S-6.02 depends on S-6.03 (sbctl base structure); S-5.02 depends on S-6.03 and S-5.01 — consistent within Wave 5.
- S-5.03 depends only on S-4.01 (Wave 4, complete). S-5.03 blocks S-5.01 (compositional: S-5.01 reads Snapshot().Degraded). Wave ordering: S-5.03 is a Wave-5 story (parallel to S-6.03, S-6.02 chain); S-5.01 must follow S-5.03.
- **S-6.02 and S-5.02 must serialize** (both edit cmd/sbctl/main.go). Recommended order: S-6.03 → {S-6.02, S-5.01+S-5.03} → S-5.02. S-6.02 and S-5.01/S-5.03 may run in parallel since they touch different modules.

**DAG is acyclic. Verified.**

## Traceability Matrices

### BC to Stories Matrix

| BC ID | Covering Stories | All BCs Covered? |
|-------|-----------------|-----------------|
| BC-2.01.001 | S-1.02 | yes |
| BC-2.01.002 | S-1.02 | yes |
| BC-2.01.003 | S-1.02 | yes |
| BC-2.01.004 | S-1.01 | yes |
| BC-2.01.005 | S-1.01 | yes |
| BC-2.01.006 | S-1.01 | yes |
| BC-2.01.007 | S-1.03 | yes |
| BC-2.02.001 | S-4.01 | yes |
| BC-2.02.002 | S-4.01 | yes |
| BC-2.02.003 | S-4.01 (PC-1 through PC-4, PC-6), S-5.03 (PC-5: degraded flag) | yes |
| BC-2.02.004 | S-4.02 | yes |
| BC-2.02.005 | S-4.03 | yes |
| BC-2.02.006 | S-4.03 | yes |
| BC-2.02.007 | S-7.01 | yes |
| BC-2.02.008 | S-4.04 | yes |
| BC-2.02.009 | S-4.01 (postconditions 1+2: DropCache primitive + Hits() accessor), S-4.04 (router OnFrameArrival wiring + EC-005 logging) | yes |
| BC-2.03.001 | S-7.02 | yes |
| BC-2.03.002 | S-7.02 | yes |
| BC-2.03.003 | S-7.02 | yes |
| BC-2.04.001 | S-3.01 | yes |
| BC-2.04.002 | S-3.01 | yes |
| BC-2.04.003 | S-3.02 | yes |
| BC-2.04.004 | S-3.02 | yes |
| BC-2.04.005 | S-3.03 | yes |
| BC-2.04.006 | S-3.02 | yes |
| BC-2.04.007 | S-W3.04 | yes |
| BC-2.05.001 | S-2.02 | yes |
| BC-2.05.002 | S-2.02 | yes |
| BC-2.05.003 | S-3.03 | yes |
| BC-2.05.004 | S-6.02 | yes |
| BC-2.05.005 | S-2.01 | yes |
| BC-2.05.006 | S-2.02 | yes |
| BC-2.05.007 | S-2.02 | yes |
| BC-2.05.008 | S-3.04, S-W3.04, S-W3.05 | yes |
| BC-2.06.001 | S-5.01 | yes |
| BC-2.06.002 | S-5.01 | yes |
| BC-2.06.003 | S-5.02 (canonical sbctl paths list + sbctl router metrics + alias sbctl router status + p99) | yes |
| BC-2.07.001 | S-6.02 | yes |
| BC-2.07.002 | S-6.03 | yes |
| BC-2.07.003 | S-6.03 | yes |
| BC-2.08.001 | S-7.03 | yes |
| BC-2.09.001 | S-7.04 | yes |
| BC-2.09.002 | S-7.04 | yes |
| BC-2.09.003 | S-6.01 | yes |

**BC Coverage: 44/44 (100%)**

### VP to Stories Matrix

| VP ID | Story | BC Source |
|-------|-------|-----------|
| VP-001 | S-1.01 | BC-2.01.004 |
| VP-002 | S-1.01 | BC-2.01.004 |
| VP-003 | S-1.01 | BC-2.01.004 |
| VP-004 | S-2.01 | BC-2.05.005 |
| VP-005 | S-2.01 | BC-2.05.005 |
| VP-006 | S-2.01 | BC-2.05.005 |
| VP-007 | S-2.02 | BC-2.05.001, BC-2.05.007 |
| VP-008 | S-2.02 | BC-2.05.001, BC-2.05.002 |
| VP-009 | S-2.02 | BC-2.05.001 |
| VP-010 | S-2.02 | BC-2.05.006 |
| VP-011 | S-4.04 | BC-2.02.008 |
| VP-012 | S-3.03 | BC-2.05.003, BC-2.04.003 |
| VP-013 | S-3.03 | BC-2.04.005, BC-2.05.003 |
| VP-014 | S-1.01 | BC-2.01.006 |
| VP-015 | S-4.04 | BC-2.01.005 |
| VP-016 | S-1.02 | BC-2.01.001, BC-2.01.003 |
| VP-017 | S-1.02 | BC-2.01.003 |
| VP-018 | S-1.02 | BC-2.01.001, BC-2.01.002 |
| VP-019 | S-4.03 | BC-2.02.005 |
| VP-020 | S-4.03 | BC-2.02.005 |
| VP-021 | S-4.03 | BC-2.02.006 |
| VP-022 | S-4.02 | BC-2.02.004 |
| VP-023 | S-4.02 | BC-2.02.004 |
| VP-024 | S-4.01 | BC-2.02.001, BC-2.02.002 |
| VP-025 | S-4.01 | BC-2.02.009 |
| VP-026 | S-4.01, S-5.03 | BC-2.02.003 |
| VP-027 | S-5.01 | BC-2.06.001, BC-2.06.002 |
| VP-028 | S-6.01 | BC-2.09.003 |
| VP-029 | S-6.01 | BC-2.09.003 |
| VP-030 | S-6.03 | BC-2.07.003 |
| VP-031 | S-3.01 | BC-2.04.001 |
| VP-032 | S-3.01 | BC-2.04.002 |
| VP-033 | S-3.02 | BC-2.04.003, BC-2.04.004 |
| VP-034 | S-3.02 | BC-2.04.006 |
| VP-035 | S-3.03 | BC-2.04.005 |
| VP-036 | S-1.03 | BC-2.01.007 |
| VP-037 | S-7.04 | BC-2.09.002 |
| VP-038 | S-7.04 | BC-2.09.001 |
| VP-039 | S-2.02 | BC-2.05.006 |
| VP-040 | S-4.01, S-5.03 | BC-2.02.003 |
| VP-041 | S-1.02 | BC-2.01.001 |
| VP-042 | S-4.02 | BC-2.01.001, BC-2.02.001 |
| VP-043 | S-7.01 | BC-2.02.007 |
| VP-044 | S-7.02 | BC-2.03.001, BC-2.03.003 |
| VP-045 | S-7.02 | BC-2.03.002 |
| VP-046 | S-6.02 | BC-2.05.004 |
| VP-047 | S-5.02 | BC-2.06.003 |
| VP-048 | S-6.02 | BC-2.07.001 |
| VP-049 | S-6.03 | BC-2.07.002 |
| VP-050 | S-7.03 | BC-2.08.001 |
| VP-051 | S-1.02 | BC-2.01.003 |
| VP-052 | S-5.01 | BC-2.06.002 |
| VP-053 | S-1.02 | BC-2.01.002 |
| VP-054 | S-4.01 | BC-2.02.002 |
| VP-055 | S-7.02 | BC-2.03.003 |
| VP-056 | S-3.02 | BC-2.04.004 |
| VP-057 | S-2.02 | BC-2.05.007 |

| VP-058 | S-3.04 | BC-2.05.008 |
| VP-059 | S-W3.05 | BC-2.05.005 |
| VP-060 | S-W3.04 | BC-2.04.007 |
| VP-061 | S-5.02 | BC-2.06.003 |
| VP-062 | S-5.02 | BC-2.06.003 |
| VP-063 | S-5.03 | BC-2.02.003 |

**VP Coverage: 63/63 (100%)**

## Gap Register

No gaps identified. All 44 BCs are covered by at least one story. All 63 VPs are assigned to stories. BC-2.02.003 PC-5 coverage gap (drift S401-O3) closed by S-5.03. VP-060 (BC-2.04.007 daemon lifecycle) is assigned to S-W3.04 (implementing_story: null in VP file — assigned here by traceability from BC-2.04.007 coverage; architect should update VP-060.md implementing_story field).

## Phase 1 Drift Items Addressed in Stories

| Drift Item | Story | Resolution |
|-----------|-------|-----------|
| F-P8-001 (BC-2.05.004 CLI surface `sbctl admin`) | S-6.02 | Story uses canonical `sbctl admin` subcommand |
| F-P8-002 (VP-030/VP-049 `sbctl router status`) | S-6.03 | Story notes canonical `sbctl router status` command |
| F-P8-003 (BC-2.08.001 architecture_module=internal/session) | S-7.03 | Story targets internal/session correctly |
| F-P8-006 (BC-2.05.007 `sbctl svtn keys list` / `sbctl admin list-keys`) | S-6.02 | Story notes canonical key list command |
| F-P8-007 (BC-2.02.005 SACK in channel header) | S-4.03 | Story AC-003 explicitly requires SACK in channel header |
| F-P8-008 (BC-2.02.007 `frame_type=fec=0x05`) | S-7.01 | Story AC-001 uses canonical enum value fec=0x05 |
| F-P8-009 (feasibility-report deployment-operations range) | (noted) | Low priority; does not affect story content; route to architect |

Items F-P8-004 and F-P8-005 (VP-026/VP-027 invariant references) are architect/test-writer Phase 3 items; not addressable in story decomposition.

## Changelog

| Version | Date | Author | Change |
|---------|------|--------|--------|
| 1.2 | 2026-06-28 | story-writer | Consistency audit fixes: F-005 — S-5.01 depends_on now includes S-5.03 (symmetric with S-5.03 blocks:[S-5.01]); F-003 — BC coverage figure corrected 42/42→44/44; add missing BC matrix rows BC-2.04.007 (S-W3.04) and BC-2.05.008 (S-3.04, S-W3.04, S-W3.05); F-004 — VP coverage figure corrected 57/57→63/63; add missing VP matrix rows VP-058 (S-3.04), VP-059 (S-W3.05), VP-060 (S-W3.04), VP-061 (S-5.02), VP-062 (S-5.02), VP-063 (S-5.03); update gap register note |
| 1.1 | 2026-06-28 | story-writer | Wave-5 planning: add S-5.03 node (depends S-4.01, blocks S-5.01); correct S-6.02 edges (remove S-6.02→S-6.03 blocking edge; add S-6.03→S-6.02 depends edge); update S-5.02 deps note (confirm S-5.01 + S-6.03); update BC-2.02.003 matrix row to add S-5.03 (PC-5); update BC-2.06.003 matrix row to note canonical+alias+p99 scope; add S-5.03 to VP-026 and VP-040 rows; update gap register (44 BCs covered; drift S401-O3 closed); add serialization note to cycle-freeness section. |
| 1.0 | 2026-06-27 | story-writer | Initial creation |
