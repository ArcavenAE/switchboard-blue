# PR #24 Review — feat(S-4.01): per-path RTT/loss tracking and duplicate-and-race dispatch

**Verdict: APPROVE** (zero blocking findings)

Fresh-eyes review against the diff (`internal/paths/`, `internal/multipath/`, `docs/demo-evidence/S-4.01/`), the PR description, the story spec, and the test evidence. I independently ran `go test`, `go test -race`, and `golangci-lint` on both packages — all green.

## What I verified

- **All 7 ACs implemented and tested:**
  - AC-001 (PathScore transitive): `PathScore` formula `rtt*(1+(loss/100)*10)` is monotone in both inputs; transitivity proven by a 9×8-grid sweep (`TestBC_2_02_003_PathScore_PropertyTransitive_Manual`) plus crafted triples. Sound.
  - AC-002 (EWMA convergence): first-probe override (`resetRTT`) avoids the conservative-init poisoning the EWMA; `TestBC_2_02_003_PathTracker_EWMAConvergence` deliberately feeds varying RTTs to falsify a degenerate last-value (alpha=1) implementation. Good test design.
  - AC-003 (send two fastest): `Send` snapshots the path set under lock, ranks, truncates to ≤2, calls `fn` without holding the lock. Single-path fallback (EC-001) and at-most-two invariant covered.
  - AC-004 (first copy delivered) / AC-005 (silent discard): `Receive` uses checksum-only endpoint dedup via atomic `AddIfAbsent` (no Contains-then-Add TOCTOU). Concurrent first-arrival-wins asserted under `-race`.
  - AC-006 (bounded LRU): `container/list` + map, O(1) lookup, eviction at capacity; bounded-capacity property sweep present.
  - AC-007 (Hits counter): increments on `Add`/`AddIfAbsent` hit under the existing mutex; concurrent counter test under `-race`.
- **Tests pass, race-clean, lint-clean** — reproduced locally on both packages.
- **Commits** — conventional, scoped, story ID present, SSH-signed, no AI attribution. Red→Green TDD trail is visible and honest (including a commit that deletes a wrong-behavior-pinned test per spec ruling F-002).
- **Diff size** — ~3066 lines, but implementation is only ~560 lines (`paths.go` 234, `multipath.go` 326); the remainder is tests (~2029) and demo evidence. Proportionate.
- **Dependencies** — S-1.01 and S-2.02 are merged; this PR targets `develop` (Gitflow).
- **Deferrals** — BC-2.02.009 router wiring → S-4.04, EC-005 logging → S-4.04, VP-040/VP-054 e2e → integration harness, BC-2.02.001 EC-003 queue/E-NET-002 → future story. All ratified; none blocks an AC. Correctly NOT flagged as gaps.

## Findings

| # | Severity | Category | Finding |
|---|----------|----------|---------|
| 1 | suggestion | description | PR mermaid "Architecture Changes" diagram shows `internal/frame --> PathScore / Multipath.Send / Multipath.Receive`, implying these packages depend on `internal/frame`. They do not — `go list -deps` confirms `internal/paths` imports nothing internal and `internal/multipath` imports only `internal/paths`. The story's Architecture Compliance Rules table likewise states "internal/multipath imports frame + paths" / "internal/paths imports frame only", which the implementation does not follow. The code is self-consistent and pure-core, so this is a description/diagram inaccuracy, not a behavioral defect. |
| 2 | suggestion | coherence | `multipath.Frame` hardcodes `OuterHeader [44]byte` with a comment "the encoded 44-byte outer header", while `internal/frame` already exports `OuterHeaderSize = 44`, an `OuterHeader` struct, and `EncodeOuterHeader() [44]byte`. Reusing `frame.OuterHeaderSize` (or accepting the frame package's encoded header type) would remove the magic number and align with the architecture rule in finding #1. Not blocking — the package compiles and the checksum over `OuterHeader[:] || Payload` is correct regardless. |
| 3 | nit | coverage | Demo evidence is captured as `.txt` test transcripts rather than recordings. The generic PR-review checklist treats `.txt` demos as blocking, but that rule targets UI/binary products with a recordable surface. `internal/paths` and `internal/multipath` are pure-core libraries with no runnable binary; test-transcript evidence follows the established S-W3.04 precedent and an `evidence-report.md` with a full AC→test→transcript map is present. Recording this rationale inline in `evidence-report.md` (it is implied but not stated as a deviation) would pre-empt the question for future reviewers. |
| 4 | nit | description | PR body claims SEC-003 ("unchecked type assertion `oldest.Value.(dropEntry)`") is a LOW to "harden with comma-ok before S-4.04". The two assertion sites in `Add`/`AddIfAbsent` remain non-comma-ok in this PR. The invariant genuinely holds today (only `dropEntry` is ever pushed), so this is correctly a non-blocker, but the PR description should not imply hardening that was not done. Cheap to add `if e, ok := oldest.Value.(dropEntry); ok { delete(c.index, e.key) }`. |

## Why APPROVE

No correctness defect found. The concurrency story is solid (atomic check-and-insert eliminates the dedup TOCTOU, all shared state behind a mutex, race detector clean). The EWMA convergence and first-probe-override logic is correct and well-tested against the degenerate alpha=1 failure mode. The four findings are all observations or description/diagram inaccuracies — none affects behavior or any acceptance criterion, and none warrants blocking the merge. Recommend addressing finding #1 (correct the PR diagram/architecture claim) and #2 (reuse `frame.OuterHeaderSize`) in a follow-up or before S-4.04 wires this in.
