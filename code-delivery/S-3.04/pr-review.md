# PR Review — #15 fix(routing): E-ADM-016 logging on HMAC failure (BC-2.05.008 PC-2)

**Verdict: APPROVE** — zero blocking findings.

This PR closes Wave 3 gate finding F-1 (HIGH): `RouteFrame` dropped HMAC-failed
frames correctly but never emitted the E-ADM-016 observability log required by
BC-2.05.008 PC-2. The fix is minimal, backward-compatible, and well-tested.

## What I verified (independently, from the diff + a local run)

- **Both HMAC-failure return paths now log before returning.** PATH-A
  (`entry == nil`, auth key unavailable) and PATH-B (`verifyFrameHMAC` returns
  false, tag mismatch) each emit a distinct, accurate E-ADM-016 line carrying
  `svtn_id` and `src_addr` in hex.
- **No spurious logging on the non-failure paths.** The admission-fail return
  (`ErrNotAdmitted`) and the success path do not emit E-ADM-016. The
  `Test_BC_2_05_008_no_log_on_hmac_success` test is a genuine mutation guard
  against an "always log" implementation.
- **Backward compatibility.** `NewRouter` becomes variadic with a `nopLogger`
  default, so every existing `NewRouter(ks)` caller is unaffected and log
  events are silently discarded unless a logger is injected.
- **Tests pass.** `go test -race -run BC_2_05_008 ./internal/routing/...` — 4/4
  PASS under the race detector. `go vet` clean, `go build ./...` clean,
  `go test -race ./internal/routing/... ./internal/session/...` PASS. This
  matches the evidence claimed in the PR description.
- **Diff coherence.** Both changed files are scoped to E-ADM-016 logging; no
  unrelated changes. ~476 lines, test-heavy — appropriate for the change.
- **Commits** follow conventional format with story/BC anchors.

## Findings

| # | Severity | Category | File:Line | Finding | Suggested fix |
|---|----------|----------|-----------|---------|---------------|
| 1 | NON_BLOCKING | spec | internal/routing/routing.go:175-180; routing_log_test.go:~195 | PATH-A (PC-4, no-forwarding-entry) logging is shipped against an explicitly-unresolved spec question. The test header itself hedges: "If the spec-steward rules PC-4 does not require a log, the companion test for PATH-A should be revised to assert NO log." Code currently asserts E-ADM-016 IS logged on PATH-A. | Defensible as the fail-safe choice (operators see every dropped frame) and it violates no postcondition, so it does not block merge. But resolve the PC-4 obligation with the spec owner and remove the conditional language from the test doc so the contract is unambiguous. |
| 2 | NON_BLOCKING | doc | docs/demo-evidence/S-3.04/evidence-report.md:36 | The S-3.04 evidence report credits `ExampleRouter_invalidHMACRejected` with "BC-2.05.008 PC-2 (E-ADM-016)", but that Example only asserts the returned error — it never verified the log emission. That overclaim is precisely the latent gap that produced gate finding F-1. This PR adds the real coverage in `routing_log_test.go` but does not update the evidence report to cite it. | Update the evidence report's BC traceability row for PC-2 / E-ADM-016 to reference the new `Test_BC_2_05_008_*` tests in `internal/routing/routing_log_test.go`, so the observability postcondition is traced to a test that actually asserts the log line. |
| 3 | NON_BLOCKING (nit) | test | internal/routing/routing_log_test.go:~120 | Doc comment reads "fakeLogCapture implements routing.Logger and captures log lines", but the type is named `routingFakeLog`. Stale name carried over from the tmux `fakeLogCapture` pattern it mirrors. | Rename the comment subject to `routingFakeLog` for accuracy. |

## Checklist

1. Diff coherence — PASS
2. Description accuracy — PASS (description matches the diff and the real cause of F-1)
3. Test coverage — PASS (both failure paths + success-path negative + EC-001 zero-tag)
4. Demo evidence — present for S-3.04; stale traceability noted (finding #2). This is a fix PR, not a new story; no new ACs requiring fresh recordings.
5. Commit quality — PASS
6. Diff size — PASS (~476 lines, test-heavy)
7. Missing changes — none material; evidence-report update (finding #2) recommended
8. Dependency status — S-3.04 (#9) already merged on develop; this fix builds on it cleanly

No blocking findings. Recommend addressing findings #1 and #2 in a follow-up
(or this PR if convenient) since both concern spec/doc accuracy around the very
postcondition this PR implements.
