---
artifact_id: S-7.04-FU-SIGHUP-RELOAD-adversary-pass-9
document_type: adversarial-review
story_id: S-7.04-FU-SIGHUP-RELOAD
pass: 9
verdict: HAS_FINDINGS
novelty: MED
code_lane_sha: fa97154
story_version: "1.4"
reviewer_model: fresh-context
timestamp: 2026-07-07T08:00:00Z
---

# Adversarial Review — S-7.04-FU-SIGHUP-RELOAD Pass 9

## Summary

**Verdict:** HAS_FINDINGS  
**Novelty:** MED  
**Code lane SHA:** fa97154 (reviewed at pass-9); **remediation SHA:** 48e3271  
**Story version reviewed:** v1.4; **story version post-remediation:** v1.5  
**Streak:** HOLDS 0/3

2 findings (both LOW, consistency-polish class). 6 observations. 14 anti-findings. Novelty MED — two consistency gaps between the story spec and the shipped implementation surfaced on a fresh-context read that examined the test file as a coherent set.

---

## Findings

### F-P9-001 [LOW] [consistency-polish] AC-003-only-test-missing-no-return-assert

**Class:** consistency-polish — test assertion present in nine sibling test functions absent from one.

**Evidence:** `TestRunRouter_SIGHUPReload_SessionsNotInterrupted` (AC-003 coverage) verified that an open TCP connection survives a SIGHUP reload by probing connection state after the `mode=PE` log line appeared. The test did not include the goroutine-still-running assertion present in every other reload test in the suite: `TestRunRouter_SIGHUPReload_EtoPE` (AC-001), `TestRunRouter_SIGHUPReload_BadConfig_FailClosed` (AC-002), `TestRunRouter_VP038_EtoPEViaConfigOnly` (AC-004), and the five adversarial-remediation additions (LoadFileNotFound, MalformedYAML, PEtoE, PEtoPE, IdempotentResend). All nine siblings assert, before `cancel ctx; wg.Wait()`, that `runRouter` is still active (no premature return). The AC-003 test relied on indirect evidence — the TCP connection remaining alive implying no goroutine exit — but the explicit assertion was absent.

**Why this was not caught earlier:** Passes 1–8 examined findings along correctness, test-coverage, and process-gap axes. No pass examined structural consistency of the assertion inventory across sibling test functions within the same file.

**→ FIXED 48e3271:** `TestRunRouter_SIGHUPReload_SessionsNotInterrupted` gains the identical select-block no-return assertion present in its nine siblings (goroutine-still-running check before context cancel + wg.Wait). Assertion message style matches the sibling pattern verbatim.

---

### F-P9-002 [LOW] [consistency-polish] testenv-seam-divergence

**Class:** consistency-polish — story spec described a construction-time seam; shipped code implements a post-hoc setter; test outline carried a stale 2-arg signature.

**Evidence:** The AC-004 testenv-extension description (story v1.4 at review time) stated that `RouterHandle` holds a `chan<- os.Signal` reference wired at construction time. The shipped code at fa97154 implements a `SetSighupCh` post-hoc setter: the handle is constructed first, then the caller invokes `SetSighupCh(ch)` to register the channel. This diverges from the construction-time description — the handle does not own the channel from the moment it is created.

Additionally, the story outline for `TestRunRouter_VP038_EtoPEViaConfigOnly` still showed a two-argument `SendReloadSignal(t, cfgPath)` signature. Adversary pass-1 F-011 ruled that `runRouter`'s `configPath` is fixed at goroutine start, making the external `cfgPath` parameter unnecessary; the one-argument `SendReloadSignal(t)` form superseded the two-argument form at that remediation. The outline was never reconciled: it continued showing `SendReloadSignal(t, cfgPath)` through story versions v1.1–v1.4.

Both sub-issues share a root cause: AC-004 testenv description and outline were updated piecemeal in response to finding verdicts without a holistic review of coherence with the shipped seam shape.

**→ FIXED split:**
- **Code half (48e3271):** `SetSighupCh` and `SendReloadSignal` in `internal/testenv/testenv.go` gain transitional-seam doc comments clarifying: (a) the post-hoc setter pattern is intentional for the transitional era; (b) construction-time wiring is deferred to PE-CONNECTOR-era testenv integration; (c) cross-reference to the PE-CONNECTOR forward obligation (construction-time wiring requirement, 5th anchored obligation). No behavioral change.
- **Story half (v1.5):** AC-004 testenv-extension description revised from construction-time wiring to post-hoc `SetSighupCh` setter with explicit provenance (transitional shape; construction-time wiring deferred to PE-CONNECTOR-era testenv integration, consistent with existing AC-004 PC-2 deferral note from v1.2). Test outline updated to one-argument `SendReloadSignal(t)` with inline comment citing adversary pass-1 F-011 as the source of the cfgPath drop.

---

## Observations (non-findings)

**O-P9-001 (accepted — dead-guard, 4th confirmation):** The `if configPath == ""` guard at the top of the `case <-sighupCh:` branch has been confirmed as unreachable dead code four times: F-P1-009, F-P3-005c, F-P5-003 (ADJUDICATED-ACCEPTED), and now this pass. No new angle surfaced. ACCEPTED — no action required.

**O-P9-002 (anchored — order-sensitive-diff, PE-CONNECTOR forward obligation):** The `equalStringSlices` diff in the reload select case compares `upstreamRouters` (old) against `newUpstreams` (new) in order. Two configs with the same upstream addresses in a different order will trigger a re-emission that is semantically unnecessary. Under the current implementation (no live dial-loop) this is cosmetically suboptimal but behaviorally harmless. Under PE-CONNECTOR, which dials each upstream address on transition, an unnecessary re-emission could trigger unnecessary reconnects. ANCHORED to PE-CONNECTOR elaboration as the 5th forward obligation.

**O-P9-003 (informational — AC-001-vacuous-assert-backstopped-by-PEtoE/PEtoPE):** AC-001's primary test asserts that the mode=PE line is emitted after SIGHUP but does not verify that re-emitting the same config produces no second emission. The no-change case (IdempotentResend) is covered by the sibling test added at pass-3 F-005a. PEtoE and PEtoPE tests (pass-2 F-001, pass-3 F-005a) provide full round-trip coverage. No residual gap.

**O-P9-004 (accepted — goto-shutdown):** The `goto shutdown` at function tail is consistent with prior adjudication (pass-8 O-P8-004). Go-idiomatic single-target cleanup label. No new angle. ACCEPTED, no action required.

**O-P9-005 (informational — banner-integrity):** The startup `mode=E` / `mode=PE` emission banner format is consistent between startup path and reload path. The `"mode=PE upstream_routers=%v\n"` format string matches S-7.04's startup emission. No drift.

**O-P9-006 (informational — symlink/rename-reload):** If the operator replaces the config file atomically via `mv newcfg.yaml config.yaml` (rename) or via symlink swap, `config.LoadFile(configPath)` re-opens by name on each SIGHUP, picking up the new inode automatically. The reload path is robust to the most common operator file-management patterns. No action required.

---

## Anti-findings (14)

1. **F-P9-001 + F-P9-002 FIXED (48e3271 + story v1.5)** — AC-003 no-return assert parity achieved (identical select-block assertion added); testenv-seam-divergence resolved in code (transitional-seam doc comments) and story (description corrected; outline updated to 1-arg signature). Both LOW consistency-polish findings fully remediated.
2. **All pass-1 through pass-8 remediations held** — no regression across any of the 30 prior findings (12 P1 + 5 P2 + 5 P3 + 4 P4 + 3 P5 accepted/fixed + 1 P6 clean + 1 P7 fixed + 2 P8 fixed; all confirmed stable at 48e3271).
3. **AC-001 through AC-004 behavioral correctness intact** — 2 findings both LOW consistency-polish; zero behavioral-correctness findings this pass.
4. **nil-config guard (go.md rule 13)** — fail-closed constructor default intact; no security-perimeter parameter change at 48e3271.
5. **cap-1 channel semantics correct** — `make(chan os.Signal, 1)` drop-on-full guarantee confirmed; no concurrent-SIGHUP race angle surfaced.
6. **E-CFG-003-reload coverage extended** — P8 non-empty deep-copy asserts (fa97154) and invalid-upstream-addr test confirmed stable at 48e3271; EC-003 input-class {empty, invalid-addr} coverage maintained.
7. **Code lane perimeter clean** — 48e3271 remediation touches only test files and testenv doc comments (no-return assert addition in one test + transitional-seam doc comments); no production surface behavioral change; scope within story perimeter.
8. **POL-001 compliant** — pass-9.md authored in canonical `adversary/` subdirectory with complete frontmatter; artifact_id schema-compliant; matching changelog row in STORY-INDEX v3.98.
9. **POL-002 compliant** — STORY-INDEX backlog row updated to 9 passes / pass 10 pending; story version cell v1.4 → v1.5; changelog row v3.98 present; no undocumented version drift.
10. **POL-004 compliant** — code lane SHA pinned in frontmatter (fa97154 reviewed; 48e3271 remediation); perimeter verified; no scope drift from ACs.
11. **Correctness stable across 7 consecutive passes** — last behavioral-correctness finding was pass-2 (F-P2-001, PEtoE/PEtoPE test gap). Passes 3–9 contain only LOW test-strength, doc-class, and consistency-polish findings.
12. **Non-empty deep-copy asserts (P8 F-P8-001 fix) stable** — PEtoPE and PEtoE branches retain non-empty deep-copy assertions at 48e3271; no regression.
13. **InvalidUpstreamAddr test (P8 F-P8-002 fix) stable** — `TestRunRouter_SIGHUPReload_InvalidUpstreamAddr_FailClosed` present and passing at 48e3271; EC-003 input-class coverage maintained.
14. **PE-CONNECTOR forward obligation inventory complete at 5** — (1) dial-loop integration, (2) Failed-state trigger, (3) retransmit-send boundary (O-P8-001), (4) order-sensitive-diff (O-P9-002), (5) construction-time wiring for RouterHandle ownership (F-P9-002). Complete inventory documented for PE-CONNECTOR elaboration.

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
| P7 | LOW | 1 (doc/process-gap) | 0 correctness |
| P8 | MED | 2 (test-strength) | 0 correctness |
| P9 | MED | 2 (consistency-polish) | 0 correctness |

Streak: HOLDS **0/3**. P10 required.

Novelty note: MED is warranted because the consistency-polish class — structural assertion-set uniformity across sibling test functions, and story-spec vs implementation seam divergence — had not been explicitly examined in prior passes. Both findings are low-severity and targeted; neither reveals a behavioral gap. The fresh-context read surfaced them by examining the test file as a coherent set rather than by tracing individual ACs to their assertions.

PE-CONNECTOR note: F-P9-002 sharpens the PE-CONNECTOR Mode()-seam obligation. The transitional `SetSighupCh` post-hoc setter approach is an intentional staging decision for this story. When PE-CONNECTOR ships, `RouterHandle` must be upgraded to construction-time wiring so the handle owns the router it represents — matching the PE-CONNECTOR semantics where the handle tracks real routing state from instantiation, not via post-hoc-injected signal channels. This is the 5th forward obligation now anchored to PE-CONNECTOR elaboration.
