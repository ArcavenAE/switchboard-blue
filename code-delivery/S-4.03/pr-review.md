# PR Review — #26: feat(arq): downstream ARQ with piggybacked ACK/SACK and TLPKTDROP (S-4.03)

**Verdict: APPROVE** — zero BLOCKING findings.

Fresh-eyes review of `internal/arq` (new pure-core package). Reviewed the full
diff against `origin/develop`: `internal/arq/arq.go`, `internal/arq/arq_test.go`,
and `.factory/.../red-gate-log.md`. Verified locally: `go vet`, `go test`,
`go test -race`, `gofumpt -l`, and `golangci-lint` all clean.

## What I verified (no rubber-stamp)

- **Diff coherence:** Against the actual PR base (`origin/develop`, which already
  contains S-4.01 PR #24), the diff is exactly 3 files scoped to this story. No
  unrelated changes. (A stale local `develop` initially showed multipath/paths
  files; those are already merged upstream and do NOT appear in the PR diff.)
- **AC-001 (no duplicate delivery):** `OnAck` advances `nextExpected`
  monotonically and deletes delivered seqs from both maps; re-ACK returns
  `(nil, nil)`. Confirmed by the 24-permutation and 1024-trial property tests —
  exactly-once delivery across all orderings.
- **AC-002 (in-order delivery):** Step-1 cumulative scan + Step-2 SACK buffering +
  Step-3 consecutive flush. Traced the canonical [1,3]→retransmit-2 vector by
  hand and it delivers [1],[2,3] in order. The Step-3 flush guard
  (`ackSeq > prevNextExpected`) is correct and the recovery path is pinned by
  `...RecoversOnNextCumulativeAck`.
- **AC-003 (SACK in channel header):** `SACKFromChannelHeader` reads flags at
  byte 8 and bitmap at bytes 12–19 only; the outer-header anti-regression test
  (F-P8-007) confirms it never reads outer-payload offsets. Truncated headers
  return errors rather than reading garbage.
- **AC-004 / AC-005 (TLPKTDROP):** Removes only the overdue frame; advances
  `nextExpected` only when `overdueSeq == nextExpected+1` (C-1 fix, preserves
  lower undelivered frames); exclusive deadline (`now.After`) verified by the
  before/at/1ns-after/well-past table.
- **Security (wire input):** `OnAck` rejects out-of-window `ackSeq` in O(1)
  before iterating (RULING-003 DoS guard); unsigned-subtraction guard also
  rejects stale ACKs. `SACKFromChannelHeader` validates length before slicing.
- **Pure-core / architecture:** No goroutines, no timers, no I/O, no OS calls,
  no `init()`. Single-writer contract documented. `DegradationEvents` send is
  non-blocking with a guaranteed-buffered channel.
- **Pointer ownership (go.md rule 12):** Returned payloads are either copies
  (from `inFlight`) or ownership-transferred (deleted from `reorderBuf` before/at
  return). `EnqueueSend` copies the input payload. No aliasing into retained state.
- **Test quality:** stdlib `testing` only (no testify), `t.Helper()` on helpers,
  `t.Parallel()` throughout, table-driven where >2 cases.

## Findings

| # | Severity | Category | Finding | Suggestion |
|---|----------|----------|---------|------------|
| 1 | NON-BLOCKING | idiom | Stale doc comment on `SACKPopCount` (arq.go:454) claims "via encoding/binary"; `encoding/binary` is neither imported nor used — the body calls `bitmapToUint64`. (Already noted as Pass A LOW.) | Drop the "via encoding/binary" phrase. |
| 2 | INFO | idiom | Sentinel errors (arq.go:71/76/81) use `fmt.Errorf` with constant strings; `errors.New` is the more idiomatic form for non-formatted sentinels. Functionally identical; lint-clean. | Optional: switch to `errors.New`. |
| 3 | INFO | idiom | `SackWindowSize` (exported) + `sackWindowSize` (unexported alias) is a redundant pair (arq.go:58–64). Harmless; the exported const is consumed by tests. | Optional: collapse to one. |
| 4 | INFO | robustness | `New` with both `DropTimeout==0` and `TickInterval==0` yields `dropTimeout==0`, making every frame instantly droppable. Config validation is the caller's (S-5.01) responsibility per the purity boundary, so this is acceptable for a pure-core lib. | Optional: document or guard a zero/zero config. |
| 5 | INFO | docs | Leftover "GREEN-BY-DESIGN" docstring in a test comment (arq_test.go:949). (Already noted as Pass A LOW.) | Optional cleanup. |

## Known accepted deferrals (not flagged)

- `inFlight` unbounded growth → S403-O4, deferred to S-5.01 (documented in code).
- uint32 sequence wraparound → RULING-001 §R2 (documented in code).
- ADR-005 resync mechanics → DRIFT-S4.03-001, deferred to S-5.01.

## Notes for the merge step

- The PR description's API signatures for `OnAck`/`SACKFromChannelHeader` are
  slightly imprecise versus the code (`SACKFromChannelHeader` returns
  `(bitmap, bool, error)`), but the implemented API is sound and well-documented.
  Not a blocker.

All 5 acceptance criteria are met, tests are comprehensive and green (incl.
`-race`), lint/fmt/vet clean. Recommend merge after the listed NON-BLOCKING
cosmetic comment-cleanup is addressed at the team's convenience.
