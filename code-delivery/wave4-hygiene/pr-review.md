# PR #29 Review — docs(wave4): comment hygiene

**Verdict: APPROVE**

Comment-only documentation hygiene patch. 3 files, 8 insertions / 8 deletions, all `//` comment lines. No code, logic, or test-assertion changes. Each correction was independently verified against the source.

## Checklist

| # | Item | Result |
|---|------|--------|
| 1 | Diff coherence | PASS — all changes are comment corrections tied to the stated findings |
| 2 | Description accuracy | PASS — PR body matches the actual diff exactly |
| 3 | Test coverage | N/A — no behavior changed; assertions untouched |
| 4 | Demo evidence | N/A — comment-only; no AC behavior to demo |
| 5 | Commit quality | PASS — `docs(wave4):` conventional format, detailed per-finding body, no AI attribution |
| 6 | Diff size | PASS — 8 lines |
| 7 | Missing changes | PASS — see note below |
| 8 | Dependency status | PASS — standalone follow-on; references S-6.01 which is merged (config is wired) |

## Verification performed

- **S403-COS1 (`arq.go`):** Confirmed `arq.go` imports only `fmt`, `math/bits`, `sort`, `time` — no `encoding/binary`. `SACKPopCount` body is `bits.OnesCount64(bitmapToUint64(bitmap))`. The removed "via encoding/binary" was a false claim; the new text is accurate.
- **L-1 (`access.go`):** Confirmed `cmd/switchboard/access.go` imports `internal/config` and uses `tickIntervalFor(cfg *config.Config)` / `runAccess`. The old "FORBIDDEN imports: internal/config" + "Build fails because internal/config ... do not yet exist" header was false post-S-6.01. New PERMITTED/deferred wording is correct.
- **drain/metrics:** Confirmed `internal/drain` and `internal/metrics` do not exist as packages and are not imported anywhere under `cmd/`. The "NOT imported — still deferred" claim is accurate.
- The new comment text introduces no new false claims or stale references.

## Findings

No blocking findings.

| Severity | Category | Finding | Suggestion |
|----------|----------|---------|------------|
| NIT | coherence | `access.go:52` (outside this diff) still reads `internal/config is Wave 4` in the `sweepInterval` doc comment. It is technically accurate (config landed in Wave 4 and is now wired) but reads as residual scaffolding language now that config is permitted. | Optional future tidy — not in scope for this PR. No action required here. |

## Conclusion

The corrections are accurate, narrowly scoped, and improve the truthfulness of the doc comments. The PR description faithfully describes the change. Approving.
