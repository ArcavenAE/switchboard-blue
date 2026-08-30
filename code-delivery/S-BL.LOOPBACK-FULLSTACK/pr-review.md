# PR Review — #135 S-BL.LOOPBACK-FULLSTACK

**Reviewer:** pr-reviewer (fresh-eyes, cognitive-diversity model)
**Branch:** feature/S-BL.LOOPBACK-FULLSTACK → develop
**HEAD:** f4731614070f1b11613dac1a95b0570aed26cdf7
**Base verified:** merge-base with develop @ 4bab30f (diff = 2509 insertions / 51 deletions across 22 files, matches PR metadata)
**Story:** S-BL.LOOPBACK-FULLSTACK v1.21, input-hash 7967a2f, 8pts, 16 in-scope ACs (AC-002..AC-017; AC-001 discharged pre-implementation)
**GitHub review posted:** COMMENTED (author arcavenai, 2026-08-30T14:03:54Z) — single-identity constraint per drbothen/vsdd-factory#626; reviewDecision is always empty on this repo. Verdict is stated in-body per project convention.

## CURRENT VERDICT — Cycle 2 (2026-08-30, HEAD faf4952): CONVERGED — APPROVE-equivalent

Cycle-2 re-review verified all three Cycle-1 findings are fixed; no blocking findings remain. Full Cycle-2 verification is appended at the end of this file ("## Cycle 2 — fix verification"). Cycle-2 GitHub review: COMMENTED (author arcavenai, 2026-08-30T14:22:23Z) — single-identity constraint per drbothen/vsdd-factory#626 (arcavenai authors AND reviews #135; GitHub structurally forbids a self-`--approve`/`--request-changes`, so `reviewDecision` is always empty and the verdict is carried in-body per project convention). The Cycle-1 verdict below is retained as history and is **SUPERSEDED**.

---

## Verdict — Cycle 1 (HEAD f473161): REQUEST_CHANGES-equivalent — narrowly, PR-body/docs-only [SUPERSEDED by Cycle 2 above]

The implementation is approve-worthy and would be a clean APPROVE on its own. Two items must be resolved before merge; **neither touches implementation code** — the frozen `235bb5a` implementation stands. Once F1 is fixed (ideally F2 too), this is an APPROVE-equivalent.

### Verification performed (no rubber-stamp)
- **Frozen-implementation claim holds.** `git diff --name-only 235bb5a..HEAD` = only the 16 tapes + evidence-report (one docs-only commit `f473161`); zero `internal/` changes. `235bb5a` is an ancestor of HEAD.
- **Tuple + story match.** HEAD `f4731614`, branch, base `develop`, and story v1.21 / input-hash `7967a2f` / 8pts all confirmed against `.factory/stories/S-BL.LOOPBACK-FULLSTACK.md` frontmatter.
- **Tests pass locally.** `go test -race ./internal/testenv/` → ok, exit 0. Bench compile-gate `go test -tags integration -run '^$' ./internal/bench/` → ok, exit 0. (Did not independently re-run the 25s VP-042 bench; reported p99=52.04ms is internally consistent and correctly framed as harness-delivered, not a verification_lock flip.)
- **Traceability complete & accurate.** 17 ACs total (AC-001 discharged; 16 in-scope). AC→BC→Q traces in the test-file section headers (`loopback_test.go`) match the story section headers verbatim. PR BC→AC table matches the story frontmatter's 7 BCs (BC-2.01.001/002/003, BC-2.02.001/002/003/005) and their harness-scope/PARTIAL qualifiers. 19 loopback test functions; each in-scope AC has a demonstrating test + `.tape` + passing run. AC-013 correctly carries no unit test (bench-file mod; compile-gate + bench run per the story's binding Verification Method).
- **POL-004 satisfied.** No rendered `.gif`/`.webm` committed; `.gitignore` covers them; only `.tape` sources + `evidence-report.md` tracked.
- **Sizing correct — NOT a split-story case.** One cohesive `internal/testenv` extension (new `loopback.go` + `loopback_test.go` + small edits to `testenv.go`/`testenv_test.go`/bench). ~1150 net code lines; the rest is demo evidence.
- **CI:** Quality Gate, CodeQL, dependency-review, Analyze(go), Harden-Runner all pass.

---

## Findings (priority order)

### F1 — [BLOCKING] Blast Radius CI check is RED; PR body doesn't conform to the required template
`Blast Radius / Declaration present` (`.github/workflows/blast-radius-check.yml`) fails with `Missing "## Blast Radius" section in PR body`. Two causes, both in the PR description:
1. The section is `### Blast Radius` (H3, nested under "Risk Assessment & Deployment"). The gate's Rule 1 regex `^##\s+blast radius\b` requires a **top-level** `## Blast Radius`.
2. The body uses the generic vsdd risk template (`Systems affected` / `User impact` / `Data impact` / `Risk Level`), not the switchboard-specific three prompts the gate's Rule 2 requires: **1. Operator-visible surfaces touched / 2. Silent-failure risk / 3. Smoke gate touched** (CONTRIBUTING.md §Blast radius + `.github/PULL_REQUEST_TEMPLATE.md`). Even with the header fixed, Rule 2 still fails — none of the three prompts is present.

Severity precision: branch protection on `develop` only *requires* `Quality Gate` (which passes), so `mergeStateStatus` is `UNSTABLE`, not `BLOCKED` — the merge button is not hard-gated. But CONTRIBUTING.md states this check "fails the PR"; merging red defeats a deliberately-added control. Treat as must-fix-before-merge. Fix is a PR-description edit only. Good answers from the existing content: (1) surfaces: none — test-harness/library code in `internal/testenv` + one integration-tagged bench file; no CLI/wire/config/docs surface. (2) silent-failure risk: the exact class this story guards — a silent (nil,nil)-forever two-instance ARQ split — is regression-guarded by AC-014, and OnAck/upstream errors surface loud via AC-016/AC-017 rather than becoming silent timeouts. (3) smoke gate: no.

### F2 — [SUGGESTION] Committed evidence artifact mis-describes the AC-015 test
`docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/evidence-report.md` line 169 (and the PR's Test Evidence table row) describe `TestLoopbackDriver_NoRaceUnderConcurrentSendEcho` as "6 writer goroutines (alternating 250ms/50ms-equivalent load) and 6 reader goroutines" / "(6 writer + 6 reader goroutines)". The shipped test (`loopback_test.go:862`) is `const concurrency = 8` — 8 goroutines, each running one `SendKeystroke`/`WaitForEcho` pair with a 2s timeout; no writer/reader split, no 250ms/50ms load. The story spec says only "concurrent SendKeystroke/WaitForEcho pairs" (generic — correct); the "6+6 / 250ms/50ms" wording is carried-over boilerplate never reconciled with the final test. The AC-015 `.tape` is accurate — only the report prose + PR row are wrong. The report is a durable committed artifact certifying the test, so worth correcting; does not affect test correctness (passes clean under `-race`).

### F3 — [NIT] Pre-Merge Checklist names only "Quality Gate"
The checklist item "All CI status checks passing (`Quality Gate`)" is why the red Blast Radius check slipped past self-review. Consider naming all non-skipped policy-required checks (Quality Gate + Blast Radius/Declaration present).

---

## Disposition (Cycle 1)
REQUEST_CHANGES-equivalent, scoped to F1 (required) and F2 (recommended). Both fixes are PR-body / docs-only — no re-convergence of the frozen `235bb5a` implementation needed. F3 is optional. On F1 green + F2 corrected, this converts to APPROVE-equivalent.

---

## Cycle 2 — fix verification (2026-08-30, HEAD faf4952)

Fresh-eyes re-review at HEAD `faf4952e46deca9f59891d8d0f1e8d2a3f1ccbc4`, verifying the three Cycle-1 findings landed correctly and checking only for regressions the fix commit could have introduced. Does not re-litigate what Cycle 1 cleared (traceability, sizing, evidence coverage, frozen implementation).

### Cycle-1 findings — all three FIXED

- **F1 (BLOCKING → FIXED).** Top-level `## Blast Radius` now present (PR body line 30) with all three required prompts answered substantively (no TBD/empty): (1) Operator-visible surfaces touched — "None", `internal/testenv` + `docs/demo-evidence/**` only; (2) Silent-failure risk — "Low", each class tied to its covering test (AC-007 `SetSink` reflection guard, AC-015/016/017); (3) Smoke gate touched — "No", correctly scoped to `test/smoke/invariants.sh` (guard file confirmed to exist on disk, so the reasoning is grounded). The `Blast Radius / Declaration present` CI check is now **pass**.
- **F2 (SUGGESTION → FIXED, and correction verified accurate).** The stale "6 writer + 6 reader goroutines / 250ms/50ms load" text is gone from both the PR body Test Evidence row and `evidence-report.md` §AC-015; both now read "8 concurrent goroutines, each a `SendKeystroke`+`WaitForEcho` round trip, `-race`". Confirmed accurate against the test source: `TestLoopbackDriver_NoRaceUnderConcurrentSendEcho` declares `const concurrency = 8`, each goroutine one `SendKeystroke`→`WaitForEcho` round trip — the correction matches reality, not a swap of one wrong description for another.
- **F3 (NIT → FIXED).** Pre-Merge Checklist item now names both `Quality Gate` and `Blast Radius / Declaration present`.

### Git state — frozen-implementation claim holds
- HEAD `faf4952`, exactly one commit above the Cycle-1 HEAD `f473161`.
- `git diff --name-only f473161..HEAD` = **only** `docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/evidence-report.md`. Zero `internal/` files touched by the fix commit; the Step-4.5-CONVERGED implementation (`235bb5a`) is unchanged.

### CI — green against the current HEAD
Both key checks ran against `head_sha = faf4952` (verified via the Actions API, not the prior SHA) and concluded `success` — not pending:
- **Declaration present** — pass (the Blast Radius gate F1 was blocking on).
- **Quality Gate** — pass. Analyze(go), CodeQL, dependency-review, StepSecurity Harden-Runner — all pass. Release/notarize/homebrew/Nightly-Tier-2 — skipping (expected on a non-release feature PR).

### New observation (non-blocking NIT)
- **N1 (NIT, non-blocking).** The F1 fix leaves the PR body with two "Blast Radius" headings — the new top-level `## Blast Radius` (line 30, which satisfies the CI check) and a pre-existing `### Blast Radius` subsection under `## Risk Assessment & Deployment` (line 260). The two are internally consistent (both: `internal/testenv` only, no user/data impact, LOW risk) — harmless redundancy, optional future cleanup, does **not** block merge. No typos/formatting defects in the edited Blast Radius text or the corrected evidence report.

## Disposition (Cycle 2)
**CONVERGED — APPROVE-equivalent.** All three Cycle-1 findings correctly fixed, the corrected AC-015 description verified accurate against the test source, frozen-implementation guarantee intact (fix touched docs only), and all required CI checks pass against the current HEAD. The one new observation (N1) is a cosmetic non-blocking NIT. No blocking findings remain — ready to merge.

**Note on the formal-review gate:** the vsdd `validate-pr-review-posted` stop-hook expects a `gh pr review --approve`/`--request-changes`. On this repo that is structurally impossible: the gh identity (`arcavenai`) is also the PR author (`arcavenai`), and GitHub rejects a self-`--approve`/`--request-changes` (drbothen/vsdd-factory#626). The convergence record on this repo is, by documented convention, the COMMENTED review + in-body verdict + this `pr-review.md` + CI-green — matching every other `pr-review.md` under `.factory/code-delivery/`.
