## PR Review — #121 `fix(VP-042): testenv-integrated keystroke-echo p99 benchmark`

**Verdict: REQUEST CHANGES**

> **Posting note:** The formal `gh pr review --request-changes` verdict was rejected by GitHub with
> `Review Can not request changes on your own pull request` — the authenticated account (`arcavenai`)
> is the PR author, and GitHub forbids self approve/request-changes. This review was therefore submitted
> via `gh pr review --comment` (a review thread, NOT `gh pr comment`) carrying the intended verdict.
> A different reviewer identity (or the pr-manager triage gate) must apply the formal REQUEST-CHANGES
> verdict before merge. Submitted review: author `arcavenai`, state `COMMENTED`, 2026-07-12T22:14:18Z.

Fresh-eyes review of the diff, description, and CI evidence only (no `.factory/` internals, no `internal/testenv` source). Fix-PR flow, 8-item checklist scoped to a small test/bench-only diff. The benchmark code itself is clean — **no correctness bugs found**. Two items drive the change request; the rest are nits.

---

### Checklist summary

| # | Item | Result |
|---|------|--------|
| 1 | Diff coherence | PASS — every hunk relates to the VP-042 bench migration |
| 2 | Description accuracy | **GAP** — body marks "All CI status checks passing" `[x]`; live CI shows `Declaration present` = **fail**. Commit claims it "retires the stale comments"; two stale references remain (F1) |
| 3 | Test coverage | PASS — diagnostic bench (ADR-007, not CI-gated); tag-free suite unaffected; evidence provided |
| 4 | Demo evidence | PASS — N/A, correctly declared (no user-observable behavior) |
| 5 | Commit quality | PASS (nit F5) — conventional, detailed body, `Refs: VP-042` |
| 6 | Diff size | PASS — +128 / −9, 2 files |
| 7 | Missing changes | **GAP** — the "retire stale comments" intent is only half-done (F1) |
| 8 | Dependency status | PASS — upstream S-BL.TESTENV (#110) DELIVERED; downstream S-BL.LOOPBACK-FULLSTACK correctly scoped out |

---

### Findings

**[MEDIUM · documentation/consistency] F1 — Comment migration in `internal/bench/keystroke_echo_bench_test.go` is incomplete; the file now contradicts itself.**

The diff updates the *package-level* doc block to the new, correct framing ("S-BL.TESTENV shipped but does NOT drive the full stack; VP-042 lock remains DEFERRED, still not measurable via testenv"). But two lower comments in the same file were left untouched and now say the opposite:

- The `BenchmarkKeystrokeEcho_P99` doc comment: *"enforces the VP-042 gate (≤ 100ms p99)"* and *"VP-042 on the full stack (with tick intervals) requires S-BL.TESTENV."*
- The inline comment above `if p99 > maxP99`: *"// VP-042 gate (S-BL.BENCH AC-002)… // This loopback is lower-bound only; the full-stack gate requires S-BL.TESTENV."*

Why it matters: this is a documentation-only PR — the comments **are** the deliverable. A reader who reaches the function doc lands on the exact overclaim this PR exists to prevent: "the full-stack gate requires S-BL.TESTENV" (already shipped ⇒ implies now available) and "enforces the VP-042 gate" both read as if the 100ms check *is* the VP-042 lock and S-BL.TESTENV alone unblocks it. The commit message states it "retires the stale 'S-BL.TESTENV not yet delivered' comments" — it retired one of three references. Fix: update the function doc + inline comment to match the package doc (lower-bound floor guard; real requirement is S-BL.LOOPBACK-FULLSTACK / halfchannel.Tick()+arq+multipath, not S-BL.TESTENV).

**[MEDIUM · ci/mergeability] F2 — Live CI shows a failing required check; PR body claims all checks pass.**

`gh pr checks 121` reports `Declaration present` = **fail** (11s), with CodeQL / Analyze (go) / Quality Gate still pending. The Pre-Merge Checklist asserts "All CI status checks passing… CI re-verifies on the PR" as `[x]`. I can't diagnose what `Declaration present` validates from diff-only context (reads like a repo/factory governance gate, not a code check), so I'm not attributing it to the code — but a red required check must be resolved or explained before merge, and the checklist item should not be `[x]` while it's red.

**[LOW · nit] F3 — "floor" wording for a 100 ms upper limit reads backwards.**

New file: `maxP99 = 100 * time.Millisecond // NFR-001 / VP-042 floor guard` and `"…p99 %v exceeds NFR-001 floor %v (lower-bound path)"`. 100 ms is NFR-001's *ceiling*; "floor" here means "the lower-bound path's guard," but "exceeds NFR-001 floor 100ms" may parse as a lower limit. Consider "limit"/"ceiling" while keeping "(lower-bound path)". Cosmetic.

**[LOW · nit, optional] F4 — Duplicated p99 block across the two bench files.**

The sort → `p99idx` → clamp → `ReportMetric` → threshold sequence is byte-for-byte identical in both benches. A tag-free helper file could host `computeP99([]time.Duration) time.Duration`. Optional only — `go.md` prefers "three similar lines over a premature helper," and this is 2 sites.

**[LOW · nit] F5 — Commit type `test(bench):` vs PR title `fix(VP-042):`.** Cosmetic; `test(bench)` is arguably the more accurate.

---

### Verified-good (no rubber-stamp)

- Build-tag gating correct: `//go:build integration` (line 1 + blank line) → new file compiles only under `-tags integration`; tag-free `go test ./...` never pulls in `testenv`.
- Timer discipline correct: `b.ResetTimer()` after setup, `b.StopTimer()` before sort; `b.Cleanup(env.Close)` registered.
- Percentile math correct for n=500: `int(float64(500)*0.99)` = 495 (valid index); `>= len` clamp is correct defensive handling. Matches the sibling.
- Imports minimal and all used; deliberately omits halfchannel/arq/multipath, matching the "does not exercise the full stack" claim.
- The new file's own comments are internally consistent and carefully non-overclaiming.
- Forward-safe: still passes the documented-dead `TickInterval*` into `LoopbackConfig`, so it auto-upgrades if testenv makes config live; and the error direction is **safe** — a wrong testenv finding would only *under*-claim (label real evidence a lower bound), never falsely flip the lock.

### Note on verifiability (acceptable for this PR)

The central finding about `internal/testenv` internals (LoopbackConfig discarded, synchronous `DeliverFrame`, no halfchannel/arq/multipath imports) is **not** in this diff, so it can't be confirmed from diff-only context. Acceptable here: independently checkable with a checkout, the benchmark's runtime behavior doesn't depend on it, and the error direction is conservative. Flagged only so the record shows it rests on the author's inspection + test evidence.
