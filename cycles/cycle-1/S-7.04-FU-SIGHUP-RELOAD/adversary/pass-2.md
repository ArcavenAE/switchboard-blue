---
pass: 2
story: S-7.04-FU-SIGHUP-RELOAD
code_lane_sha: 7345f21
verdict: HAS_FINDINGS
findings_count:
  medium: 2
  low: 3
  observation: 1
  process_gap: 1
anti_findings: 12
novelty: MED
concluded_at: 2026-07-06
---

# S-7.04-FU-SIGHUP-RELOAD Adversarial Pass 2

## Verdict

**HAS_FINDINGS** — 5 findings (2 MED, 3 LOW; 1 OBS/drift). All dispositioned same-day. Novelty MED (pass-1 HIGH findings resolved; new coverage gaps exposed in LoadFile arm and write-probe). Finding decay: 12 → 5.

## Findings

| ID | Severity | Title | Disposition |
|----|----------|-------|-------------|
| F-P2-001 | MED | LoadFile-arm untested (E-CFG-004 + E-CFG-005 error paths) | FIXED 256548b (E-CFG-004 + E-CFG-005 tests added) |
| F-P2-002 | MED | Write-probe near-tautological (read-deadline absent) | FIXED 256548b (read-deadline probe added; declared divergence from story outline) |
| F-P2-003 | LOW | Success-path cfg state unasserted after reload | FIXED 256548b |
| F-P2-004 | LOW | File-Change List incomplete [process-gap] | FIXED story v1.1 (router_config.go + mgmt_wire_test.go rows added) |
| F-P2-005 | LOW | Lock discipline: upstreamRouters assignment not atomic with emit | FIXED aa62242 |
| F-P2-OBS | OBS | Inert-reload UX: non-upstream field changes silently inert | DRIFT ROW filed (DRIFT-SIGHUP-INERT-RELOAD-UX) |

## Anti-Findings

12 confirmed correct behaviors: E-CFG-001 taxonomy wrapping preserved through reload error path, defer signal.Stop hygiene confirmed, signal.Notify buffered-capacity 1 rationale intact, ctx non-mutation in SIGHUP case confirmed, ingressCtx/dataWG/drainCoord non-touch confirmed, mgmtSrv non-touch confirmed, startup emission format match on reload path, pass-1 F-P1-001/002/004/005/006 remediations confirmed effective, equalStringSlices diff call correct, test file placement canonical (router_sighup_test.go), testenv SendReloadSignal seam non-invasive.

## Novelty

MED — new coverage layer (LoadFile error arm, write-probe strength, lock discipline). Structural correctness largely confirmed from pass 1.
