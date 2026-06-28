---
document_type: consistency-report
level: ops
version: "1.0"
producer: consistency-validator
timestamp: 2026-06-27T00:00:00Z
traces_to: STATE.md
audit_scope: Wave 3 pre-gate perimeter
develop_head: 849bd86
---

# Wave 3 Pre-Gate Consistency Audit

**Scope:** develop HEAD 849bd86. PRs #19 (T2 deterministic TOCTOU misclassification test)
and #20 (C-1 WithFailureCounter wiring). ARCH-08 v2.3. Stories S-W3.04, S-W3.05.

**Audit approach:** Every claim grounded by reading actual code files and running tests.
No findings from spec-only reasoning.

---

## Per-Dimension Results

| Dimension | Verdict | Severity of Findings |
|-----------|---------|---------------------|
| D-1: Spec↔code — C-1 RESOLVED claim matches code | PASS | — |
| D-2: Obligation T2 — deterministic test exists and exercises branch | PASS | LOW / OBSERVATION |
| D-3: Version/citation consistency — ARCH-08 v2.3 propagation | PASS with traceability gaps | LOW |
| D-4: BC citations — PC-5/PC-3 exist, E-ADM-016/017 are real | PASS | — |
| D-5: Deferral integrity — S-BL.NI tracking | PARTIAL PASS | MEDIUM |
| D-6: STATE.md ↔ filesystem coherence | PASS | — |

**Overall verdict: PASS — gate MAY proceed. Two non-blocking findings documented below.**

---

## Dimension 1: Spec↔Code — C-1 RESOLVED Claim

**Claim being checked:** ARCH-08 v2.3 §6.5.1 states "C-1 RESOLVED — `routing.WithFailureCounter(fc)`
(threshold=5, window=60s) wired in `buildRouter` alongside `routing.WithLogger`, PR #20 (commit 418de54)."

**Ground checks performed:**

1. `cmd/switchboard/access.go` lines 54–59 confirm:
   ```
   const (
       hmacFailureThreshold = 5
       hmacFailureWindow    = 60 * time.Second
   )
   ```
   Both constants exactly match the BC-2.05.005 PC-3 spec values.

2. `buildRouter` function (access.go lines 306–307):
   ```
   fc := admission.NewFailureCounter(hmacFailureThreshold, hmacFailureWindow, rl)
   return routing.NewRouter(ks, routing.WithLogger(rl), routing.WithFailureCounter(fc))
   ```
   `WithFailureCounter` is wired alongside `WithLogger`. Matches ARCH-08 v2.3 exactly.

3. Test `TestBuildRouter_WithFailureCounter_FiveFailures_TriggersEADM017` exists in
   `cmd/switchboard/failure_counter_wire_test.go`. The test:
   - Calls `buildAccessComponents` (the production daemon construction path, not a parallel reconstruction)
   - Drives 5 consecutive PATH-A HMAC failures from the same source
   - Asserts `captureLogger.HasLine("E-ADM-017")`

4. Test PASSES: confirmed by `go test ./cmd/switchboard/... -run TestBuildRouter_WithFailureCounter_FiveFailures_TriggersEADM017`
   Output: `--- PASS: TestBuildRouter_WithFailureCounter_FiveFailures_TriggersEADM017 (0.00s)`

**Verdict: PASS.** ARCH-08 v2.3's "C-1 RESOLVED" claim is accurate and grounded in code.

---

## Dimension 2: Obligation T2 — TOCTOU Misclassification Test

**Claim being checked:** PR #19 (849bd86) adds a deterministic test satisfying ADR-011 v1.6 Obligation T2.
S-W3.04 AC-010 asserts T2 is satisfied.

**Ground checks performed:**

1. File `internal/tmux/connector_toctou_misclass_test.go` EXISTS on filesystem.

2. Test name: `TestForwardFramesTOCTOUMisclassificationBranchDeterministic`. The test:
   - Reaches the `srcCh == prevSrcCh && inPTY == true` branch documented in `forwardFrames`
     (confirmed by reading test Phase 2 logic + `connector_frames.go` lines 169–170)
   - Uses two seam channels (`swapBarrier`, `swapBarrier2`) to control relay iteration timing
   - `ptyAllocFunc` blocks until `ptyAllocReady` is sent, guaranteeing `sc.active=ctrl`
     during the relay's second `activeSourceSnapshot` call
   - The test exercises the SECOND relay iteration (prevSrcCh=ctrl.frames path) — the gap
     NOT covered by `TestForwardFramesTOCTOURegressionDeterministic`

3. Test PASSES: `--- PASS: TestForwardFramesTOCTOUMisclassificationBranchDeterministic (0.00s)`

4. The `activeSourceSnapshot` function in `connector_frames.go` uses a single `sc.mu` hold
   for the full `{src, srcCh, inPTY}` snapshot (verified lines 54–85 of `pty_fallback.go`),
   confirming the fix is intact.

**Finding T2-1 — LOW / OBSERVATION: S-W3.04 AC-010 does not mention the new test**

- File: `.factory/stories/S-W3.04-daemon-assembly.md` (version 1.4, lines 239–263)
- AC-010 states T2 is "fully satisfied" citing `TestForwardFramesTOCTOUCount50` and
  `TestForwardFramesTOCTOURegressionDeterministic`, both "verified passing on develop at e9421d8"
- The test header for `connector_toctou_misclass_test.go` explicitly states the existing
  deterministic test does NOT cover the second-relay-iteration misclassification branch
- S-W3.04 was last updated at version 1.4 before PR #19 landed; AC-010 was never updated
  to mention `TestForwardFramesTOCTOUMisclassificationBranchDeterministic`

**Classification:** LOW (traceability-only gap, not a code defect). T2 is satisfied in code;
the story's AC-010 simply doesn't mention the newest test that most directly exercises the
cited branch. No behavioral regression.

**Remediation (non-blocking, defer to Wave 4 story hygiene):** Update S-W3.04 AC-010 to add
`TestForwardFramesTOCTOUMisclassificationBranchDeterministic` in `internal/tmux/connector_toctou_misclass_test.go`
to the list of tests that satisfy Obligation T2. The story version should bump to 1.5.
This is cosmetic traceability; it does not block the gate.

**Verdict: PASS.** T2 obligation is satisfied in code. AC-010 has a traceability gap (LOW),
non-blocking.

---

## Dimension 3: Version/Citation Consistency

**Claim being checked:** ARCH-08 v2.3 is referenced consistently in ARCH-INDEX and STATE.md.
Any story pinning ARCH-08 v2.2 that should reference v2.3?

**Ground checks:**

1. ARCH-INDEX changelog (last row): "2026-06-27 | architect | ARCH-08 v2.3: C-1 RESOLVED..."
   — ARCH-INDEX is current.

2. STATE.md last entry: "ARCH-08 bumped to v2.3; ARCH-INDEX changelog updated." — STATE.md
   correctly reflects v2.3.

3. `w3_c1_disposition`: "RESOLVED — WithFailureCounter wired buildRouter (threshold=5/window=60s);
   OBS-3 closed; network-ingress listener deferred S-BL.NI" — accurate.

4. S-W3.04 story token-budget table (line 310): `ARCH-08-dependency-graph.md (§6.5–§6.6 v2.1)`
   — pins v2.1. Also in-comment on line 59: "ARCH-08 v2.1".

5. S-W3.05 story: ARCH-08 citations are `§6.5 (positions 4 + 5)` without a version pin — acceptable.

**Finding V-1 — LOW: S-W3.04 story pins ARCH-08 at v2.1; current version is v2.3**

- File: `.factory/stories/S-W3.04-daemon-assembly.md` lines 59, 310, 343, 365, 407, 412, 435, 462
- The v2.2→v2.3 delta introduced the WithFailureCounter wiring (C-1). This is delivered by PR #20,
  which is NOT part of the S-W3.04 story implementation. The v2.2 delta addressed I-1 wg-join, which
  IS part of S-W3.04. The story's v2.1 pin predates both.
- Semantic relevance: The v2.2→v2.3 changes are about obligations that were added as separate
  fix-PRs after S-W3.04 merged. The story's core obligations all match ARCH-08 §6.5.1 which is
  semantically unchanged for its six listed obligations. The version pin is stale but not misleading
  for story implementers reviewing completed work.

**Classification:** LOW (citation drift on a completed story). The v2.2/v2.3 changes are captured
in separate PRs #18 and #20. No implementer would be misled at this stage.

**Remediation (non-blocking):** S-W3.04 story changelog entry at version 1.5 should update
ARCH-08 version pin from v2.1 to v2.3 as housekeeping. Not gate-blocking.

**Verdict: PASS.** No v2.2→v2.3 delta introduces a semantically relevant inconsistency in any
currently active story.

---

## Dimension 4: BC Citations

**Claims being checked:**
- BC-2.05.008 PC-5 exists and states RouteFrame calls RecordHMACFailure
- BC-2.05.005 PC-3 exists and specifies threshold=5/window=60s
- E-ADM-016 and E-ADM-017 are real and used consistently

**Ground checks:**

1. BC-2.05.008 PC-5 (behavioral-contracts/ss-05/BC-2.05.008.md line 63):
   "On every `ErrHMACVerificationFailed` return path... `RouteFrame` calls
   `router.failureCounter.RecordHMACFailure(srcAddrHex)` BEFORE returning"
   — Exists and matches code behavior. PASS.

2. BC-2.05.005 PC-3 (behavioral-contracts/ss-05/BC-2.05.005.md line 61):
   "a 60-second window. When the count for a `src_addr` reaches or exceeds **5**
   within any trailing 60-second window, the `FailureCounter` emits... E-ADM-017"
   — threshold=5, window=60s explicitly stated. Matches code constants. PASS.

3. Error taxonomy `error-taxonomy.md`:
   - E-ADM-016 (line 52): exists, documents PATH-A and PATH-B variants, references routing.ErrHMACVerificationFailed
   - E-ADM-017 (line 53): exists, documents format "E-ADM-017 HMAC failure rate alert: ≥<threshold>
     failures in <window_seconds>s from src <src_addr>", severity=degraded, references BC-2.05.005 PC-3
   Both entries are real, consistent with code and BC specs.

4. E-ADM-017 message format consistency: BC-2.05.005 PC-3 says "E-ADM-017 HMAC failure rate alert:
   ≥`<threshold>` failures in `<window_seconds>`s from src `<src_addr>`". Error taxonomy says same
   parameterized format. Both consistent.

**Verdict: PASS.** All BC clause citations are real, accurate, and consistent across specs and code.

---

## Dimension 5: Deferral Integrity — S-BL.NI

**Claim being checked:** S-BL.NI (network-ingress listener) is tracked consistently in STORY-INDEX,
STATE, and ARCH-08 with no dangling references or orphaned deferrals.

**Ground checks:**

1. STORY-INDEX line 97: S-BL.NI row exists, status=draft, Wave 4+. Present.

2. STATE.md: Multiple references to S-BL.NI in the session checkpoint and resume section.
   `w3_c1_disposition`: "network-ingress listener deferred S-BL.NI" — correct.

3. ARCH-08 v2.3 §6.5.1 C-1 RESOLVED block: "Only remaining deferral at this boundary: The
   network-ingress LISTENER... tracked as story S-BL.NI." — consistent with STATE.

4. No `S-BL.NI.md` story file exists in `.factory/stories/` — only `S-BL.OA-outer-assembler.md`
   exists. S-BL.NI is represented only as a row in STORY-INDEX, which is expected for draft/backlog stories.

**Finding D5-1 — MEDIUM: S-BL.NI obligation description is stale after C-1 resolution**

- File: `.factory/stories/STORY-INDEX.md` line 97
- Current description: "MUST wire routing.WithFailureCounter(fc) alongside routing.WithLogger(rl)
  in buildRouter; MUST include daemon-level integration test asserting E-ADM-017 fires through
  the daemon's own router; partial wiring (logger only) is FORBIDDEN per ARCH-08 v2.2 §6.5.1"
- Reality after PR #20: `WithFailureCounter(fc)` IS ALREADY WIRED in `buildRouter`. The "MUST
  wire" obligation is already complete. Partial wiring is no longer the risk.
- ARCH-08 v2.3 correctly states: "No partial-wiring obligation remains outstanding for the
  failure counter itself." But STORY-INDEX still describes the pre-C1-resolution state.
- The S-BL.NI description also cites ARCH-08 v2.2 (superseded by v2.3).
- Additionally: the description says S-BL.NI should include an integration test for E-ADM-017
  "through the daemon's own router." Since E-ADM-017 is now wired and tested by
  `TestBuildRouter_WithFailureCounter_FiveFailures_TriggersEADM017`, the S-BL.NI integration
  test obligation may need to focus on the live-data-path validation (frames arriving from network
  triggering RouteFrame), not the counter wiring itself.

**Classification:** MEDIUM (misleading spec for a future story — wrong obligation description,
stale ARCH-08 version pin, and incorrect implication that failure counter is not yet wired).
Not gate-blocking for Wave 3 (S-BL.NI is Wave 4+), but a Wave 4 story-writer reading this
would start from wrong premises.

**Remediation (before Wave 4 story is activated):** Update S-BL.NI row in STORY-INDEX to:
1. Remove "MUST wire routing.WithFailureCounter(fc) alongside routing.WithLogger(rl) in
   buildRouter" — this is already done (C-1 resolved)
2. Update the obligation to focus on: wiring a network-ingress listener (bind/accept network
   frames → feed to RouteFrame); no counter wiring obligation remains
3. Retain the E-ADM-017 integration test obligation but scope it to the live data path
4. Update ARCH-08 citation from v2.2 to v2.3

**Verdict: PARTIAL PASS.** The deferral is correctly tracked and does not create orphaned
references. However, the S-BL.NI obligation description is stale and would mislead a Wave 4
implementer. MEDIUM finding, non-blocking for Wave 3 gate.

---

## Dimension 6: STATE.md ↔ Filesystem Coherence

**Claims being checked:** STATE.md says develop HEAD = 849bd86, 3/3 convergence, both pre-gate
items merged. Does this match git log and story statuses?

**Ground checks:**

1. `git rev-parse HEAD` = `849bd86ee03e1d6724a39dccadc1343d2880d24c` — matches.

2. Git log confirms:
   - 849bd86: T2 test (PR #19) — MERGED
   - 418de54: C-1 WithFailureCounter (PR #20) — MERGED
   - e9421d8: I-1 wg-join fix (PR #18) — MERGED
   All three pre-gate items present in git history.

3. STORY-INDEX counts: "Total: 28 (26 wave stories + S-M.01 + S-M.02)", "Complete: 13".
   Filesystem has 29 story `.md` files (excluding STORY-INDEX and dependency-graph.md).
   The discrepancy: S-BL.OA has a file but is counted in backlog (not in the 28). Count is
   consistent with the stated methodology (backlog stories not counted in the 28 total).

4. STATE.md `wave_3_gate_human_gate: PENDING` — correctly reflects no human approval yet.

5. STATE.md `wave_3_convergence_summary`: "3/3 CLEAN passes" — consistent with adversary
   streak=3 in `wave_3_gate_adversary_streak: 3`.

**Verdict: PASS.** STATE.md accurately reflects the git and filesystem state.

---

## Summary of All Findings

| ID | Dimension | Severity | Real defect or traceability gap? | Description |
|----|-----------|----------|----------------------------------|-------------|
| T2-1 | D-2 (Obligation T2) | LOW | Traceability gap | S-W3.04 AC-010 (v1.4) does not mention `TestForwardFramesTOCTOUMisclassificationBranchDeterministic` (added PR #19). The AC says T2 is "fully satisfied" citing only the pre-PR-19 tests. Not a code defect — test passes and exercises the correct branch. |
| V-1 | D-3 (Version citations) | LOW | Traceability gap | S-W3.04 story multiple locations pin ARCH-08 at v2.1; current version is v2.3. The v2.2/v2.3 delta is semantically handled by separate fix PRs. |
| D5-1 | D-5 (Deferral integrity) | MEDIUM | Misleading spec (stale, wrong obligation) | S-BL.NI row in STORY-INDEX describes WithFailureCounter wiring as a future obligation, but it is already done. S-BL.NI also cites ARCH-08 v2.2 (superseded). A Wave 4 story-writer would start from wrong premises. |

**Blocking findings:** NONE.

**Overall gate verdict: PASS.** The Wave 3 perimeter is internally consistent and grounded in
code. Both pre-gate deliveries (C-1 and T2) are correctly implemented, tested, and passing.
ARCH-08 v2.3's "C-1 RESOLVED" claim is accurate. BC contract clauses and error taxonomy entries
are real and consistent. S-BL.NI is tracked. The two LOW findings are traceability housekeeping
on completed work; the MEDIUM finding is a stale spec entry for a future Wave 4 story.

---

## Remediation Tracking

| Finding | Action | Priority | Suggested vehicle |
|---------|--------|----------|-------------------|
| T2-1 | Add `TestForwardFramesTOCTOUMisclassificationBranchDeterministic` to S-W3.04 AC-010; bump story to v1.5 | post-gate cosmetic | Wave 4 story hygiene pass |
| V-1 | Update S-W3.04 ARCH-08 version pin from v2.1 to v2.3 | post-gate cosmetic | Wave 4 story hygiene pass |
| D5-1 | Rewrite S-BL.NI row in STORY-INDEX to remove already-completed FailureCounter wiring obligation; update ARCH-08 citation to v2.3; scope to live-ingress-listener obligation | before Wave 4 S-BL.NI story activation | Story-writer pass at Wave 4 kick-off |
