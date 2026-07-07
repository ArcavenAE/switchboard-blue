---
pass: 1
story: S-7.04-FU-SIGHUP-RELOAD
code_lane_sha: 74cd94f
verdict: HAS_FINDINGS
findings_count:
  medium: 2
  low: 3
  observation: 7
  process_gap: 1
anti_findings: 11
novelty: HIGH
concluded_at: 2026-07-06
---

# S-7.04-FU-SIGHUP-RELOAD Adversarial Pass 1

## Verdict

**HAS_FINDINGS** — 12 findings (2 MED, 3 LOW, 7 OBS/process-gap). All dispositioned same-day. Novelty HIGH (first pass on fresh story at 74cd94f).

## Findings

| ID | Severity | Title | Disposition |
|----|----------|-------|-------------|
| F-P1-001 | MED | EC-004 verbatim message not pinned in test oracle | FIXED c4c4a7b |
| F-P1-002 | MED | Tautological Mode() assertion in reload test | FIXED c4c4a7b |
| F-P1-003 | MED | Stale DEFERRED banner in story story outline [process-gap] | FIXED 7345f21 |
| F-P1-004 | LOW | E-CFG-001 wrapped inside reload-path wrapper text unclear | FIXED c4c4a7b |
| F-P1-005 | LOW | PE→E / PE→PE′ downgrade/stable transitions untested | FIXED c4c4a7b |
| F-P1-006 | LOW | cfg immutability unasserted on failure path | FIXED c4c4a7b |
| F-P1-007 | OBS | Order-sensitive diff in equalStringSlices | ANCHORED to S-7.04-FU-PE-CONNECTOR elaboration (BC-2.09.001 set-vs-order ruling required) |
| F-P1-008 | OBS | Closed-channel SIGHUP send silently no-ops | ADJUDICATED-ACCEPTED (unreachable; defensive dead code) |
| F-P1-009 | OBS | Silent no-op on empty configPath with SIGHUP | ADJUDICATED-ACCEPTED (unreachable; defensive dead code) |
| F-P1-010 | OBS | 100ms timing window in reload test | FIXED c4c4a7b (ordering-based assertion) |
| F-P1-011 | OBS | Unused sighupCh parameter in stub path | FIXED c9d4014 |
| F-P1-012 | OBS | SIGHUP kills other modes (access/console/control) via default Go SIGHUP | DRIFT ROW filed (DRIFT-SIGHUP-MODE-ASYMMETRY) |

## Anti-Findings

11 confirmed correct behaviors: signal.Notify buffered-channel best-practice, defer signal.Stop placement, ctx orthogonality (NotifyContext unchanged), fail-closed reload ordering (LoadFile → Validate → diff → emit), atomicity of upstreamRouters assignment, cfg pointer immutability as spec-stated, startup-emission format replication, runRouter goroutine non-return assertion, testenv sighupCh injection seam pattern, E-to-PE and PE-to-E round-trip structural correctness, E-CFG-001 taxonomy preserved inside EC-004 wrapper.

## Novelty

HIGH — first-pass coverage of new SIGHUP reload path. Finding decay expected at pass 2.
