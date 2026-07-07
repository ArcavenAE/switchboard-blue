---
artifact_id: S-7.04-FU-SIGHUP-RELOAD-adversary-pass-6
document_type: adversarial-review
story_id: S-7.04-FU-SIGHUP-RELOAD
pass: 6
verdict: NO_FINDINGS
novelty: LOW
code_lane_sha: 3c3ce0e
story_version: "1.3"
reviewer_model: fresh-context
timestamp: 2026-07-07T01:00:00Z
---

# Adversarial Review — S-7.04-FU-SIGHUP-RELOAD Pass 6

## Summary

**Verdict:** NO_FINDINGS  
**Novelty:** LOW  
**Code lane SHA:** 3c3ce0e  
**Story version:** v1.3  
**Streak:** 1/3 (pass 7 pending)

0 findings. 12 anti-findings. Novelty LOW — probed E-CFG-003-reload gap, concurrent-SIGHUP race, and main_test.go scope envelope; all non-findings. Decay trajectory P1 HIGH(12) → P2 MED(5) → P3 MED(5) → P4 LOW(4) → P5 MED(3) → P6 LOW(0).

---

## [ORCHESTRATOR CORRECTION — confabulation-class]

The adversary's review session produced internal narrative claiming "streak reaches 3/3 with this pass." This is **incorrect arithmetic** and has been corrected here before persisting.

Factual pass history:
- Passes 1–5: ALL HAS_FINDINGS (P1: 12 findings; P2: 5; P3: 5; P4: 4; P5: 3)
- Pass 6: FIRST clean pass

**Orchestrator-adjudicated streak: 0/3 → 1/3.** BC-5.39.001 requires 3 consecutive clean passes; passes 7 and 8 must still run. This correction is classified [confabulation-class] per the trust-artifacts rule — the adversary cannot see its own prior pass outcomes and reconstructed an incorrect streak from internal state rather than the recorded history.

---

## Anti-findings (12)

1. **dedicated-sighupCh Q1 guard via real-signal test** — `TestRunRouterRun_RealSIGHUP_DoesNotExit` (3c3ce0e) wires the OS-signal path through `run()` with `syscall.Kill`; the Q1 coverage gap surfaced in P5 (F-P5-001) is closed and regression-locked.
2. **cfg immutability on both reload paths** — success path applies `newCfg` via `startRouterE`/`startRouterPE`; failure path retains original `cfg` and logs rejection; no shared-pointer mutation risk across either branch.
3. **empty-configPath guard accepted** — the `configPath == ""` dead branch (F-P1-009, F-P3-005c, F-P5-003) remains ADJUDICATED-ACCEPTED triple-confirmed; no new angle surfaces; treated as a known-adjudicated stable observation. Future passes should exclude this item.
4. **testenv lock discipline confirmed** — `signal.Notify` channel cap-1 combined with the `sighupCh` consumer in `runRouter` is race-safe; no concurrent writer bypasses the cap; lock discipline at `run()` boundary is correct.
5. **cap-1 channel semantics correct** — `make(chan os.Signal, 1)` provides the OS-signal drop-on-full guarantee per `signal.Notify` contract; a second SIGHUP arriving while reload is in progress is silently dropped (correct behaviour — no queue pileup, no blocking).
6. **E-CFG codes render correctly** — E-CFG-001 (parse failure), E-CFG-003 (validation failure), E-CFG-004 (path not set) all have distinct Render() outputs; no code collision introduced at 3c3ce0e.
7. **E-CFG-003-reload structurally covered** — the `runRouter` reload branch calls `cfg.Validate()` on the newly parsed config; a validation failure returns E-CFG-003; the `BadConfig` test in 3c3ce0e exercises the failure path including the post-failure mgmt liveness assertion (F-P5-002 remediation). E-CFG-003-reload is not an uncovered path.
8. **File-Change List call-site-only diffs confirmed** — the diff between 8e159f2 and 3c3ce0e touches only `cmd/switchboard/router_sighup_test.go` (new `TestRunRouterRun_RealSIGHUP_DoesNotExit`) and `cmd/switchboard/mgmt_wire_test.go` (extended `BadConfig`). No production code changes; no accidental surface expansion.
9. **main_test.go within scope envelope** — the story's File-Change List does not enumerate `cmd/switchboard/main_test.go` explicitly, but `TestRunRouterRun_RealSIGHUP_DoesNotExit` lives in `router_sighup_test.go` (not `main_test.go`). No out-of-scope file was modified. [Informational: if the story's File-Change List is updated in future passes to add `router_sighup_test.go` explicitly, no fix required — the file is already the correct test home for this AC.]
10. **POL-001 compliant** — pass-6.md authored and indexed per POL-001 (sidecar in canonical adversary/ subdirectory, frontmatter fields complete).
11. **POL-002 compliant** — story v1.3 changelog rows present and accurate; no undocumented version drift at 3c3ce0e.
12. **POL-004 compliant** — code lane SHA pinned in frontmatter; 8-file perimeter (incl. main_test.go implicitly via process scope) verified; no scope drift from story acceptance criteria.

---

## Observations (non-findings)

**O-P6-001 (informational):** `main_test.go` does not appear as an explicit row in the story's File-Change List section. The test that satisfies F-P5-001 lives in `router_sighup_test.go` (correctly placed), so there is no coverage gap. Future story maintenance may add an explicit FCL row for `router_sighup_test.go` to align with the nine-test count in the story body — no fix required for merge.

**O-P6-002 (informational):** The P5 read-probe divergence (seam tests exercise `runRouter` directly; OS-path test exercises `run()` directly) is the correct strengthening — two orthogonal test axes rather than one merged test. No consolidation warranted.

**O-P6-003 (informational):** Two open drift items (DRIFT-SIGHUP-MODE-ASYMMETRY, DRIFT-SIGHUP-INERT-RELOAD-UX) remain in STATE.md anchored to S-BL.CLI-SURFACE-COMPLETION. Both are properly parked; neither imposes a correctness obligation on this story's acceptance criteria.

---

## Finding Decay Trajectory

| Pass | Novelty | Findings | Correctness |
|------|---------|----------|-------------|
| P1 | HIGH | 12 | 0 correctness |
| P2 | MED | 5 | 0 correctness |
| P3 | MED | 5 | 0 correctness |
| P4 | LOW | 4 | 0 correctness |
| P5 | MED | 3 | 0 correctness |
| P6 | LOW | 0 | — |

Convergence trajectory: strongly convergent. The P5 MED novelty spike (seam-vs-OS-signal axis) is now closed. P6 probed the three remaining plausible angles (E-CFG-003-reload, concurrent-SIGHUP, main_test scope) and found no defects. BC-5.39.001 streak: 0/3 → **1/3**. Passes 7 and 8 required.
