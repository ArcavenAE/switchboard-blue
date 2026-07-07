---
pass: 3
story: S-7.04-FU-SIGHUP-RELOAD
code_lane_sha: 256548b
verdict: HAS_FINDINGS
findings_count:
  medium: 3
  low: 2
  observation: 0
  process_gap: 2
anti_findings: 12
novelty: MED
streak_after: 0/3
concluded_at: 2026-07-06
---

# S-7.04-FU-SIGHUP-RELOAD Adversarial Pass 3

## Verdict

**HAS_FINDINGS** — 5 findings (3 MED, 2 LOW; 2 are process-gap). All dispositioned same-day. Zero code-correctness findings — all findings are process-gap or spec-clarity gaps. Novelty MED. Streak: 0/3 (pass 4 dispatches next).

## Findings

| ID | Severity | Title | Disposition |
|----|----------|-------|-------------|
| F-P3-001 | MED | AC-004 PC-2 unverifiable via stub seam (RouterHandle.Mode() tautological) | FIXED story v1.2 (emission-based observable; Mode()-seam deferred to testenv/PE-CONNECTOR era) |
| F-P3-002 | MED | Changelog row 1.1 missing from story table [process-gap — POL-001] | FIXED story v1.2 |
| F-P3-003 | MED | STORY-INDEX stale at v1.0 [process-gap — POL-002; orchestrator sequencing error: index sync deferred past pass dispatch] | FIXED THIS BURST (STORY-INDEX sync to v1.2 in state-manager burst) |
| F-P3-004 | LOW | Dangling DELIVERY pointer in story frontmatter | FIXED 8a40a0a |
| F-P3-005a | LOW | Diff-guard IdempotentResend untested | FIXED 8a40a0a (IdempotentResend test added) |
| F-P3-005b | OBS | SIGHUP during shutdown race (bounded-correct) | ADJUDICATED-ACCEPTED (bounded-correct; context cancellation races are by-design) |
| F-P3-005c | OBS | Dead configPath branch (defensive dead code) | ADJUDICATED-ACCEPTED (defensive dead code; unreachable in production flow) |

## Anti-Findings

12 confirmed correct behaviors: pass-2 remediations all effective (LoadFile-arm coverage, write-probe read-deadline, cfg immutability, lock discipline, File-Change List entries), emission-based observable (mode=PE line on writer) structurally sound, goroutine non-return assertion correct, testenv SendReloadSignal injection seam correct, AC-001/AC-002/AC-003 test structures sound, E-CFG-004/E-CFG-005 error path coverage confirmed, signal.Stop defer placement correct, SIGHUP does not touch ingressCtx/dataWG/drainCoord/mgmtSrv confirmed.

## Trajectory Note

P1 HIGH (12 findings) → P2 MED (5 findings) → P3 MED (5 findings, zero code-correctness findings). Finding decay underway. P3 findings are exclusively process-gap or spec-clarity — no implementation defects remain. Pass 4 dispatches against story v1.2 / code lane 8a40a0a.
