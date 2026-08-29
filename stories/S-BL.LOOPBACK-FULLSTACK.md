---
artifact_id: S-BL.LOOPBACK-FULLSTACK
document_type: story
level: ops
story_id: S-BL.LOOPBACK-FULLSTACK
epic_id: E-1
title: "Full-stack loopback testenv extension: tick-driven halfchannel + arq + multipath wiring for VP-042"
status: draft
producer: story-writer
timestamp: 2026-07-12T00:00:00Z
version: "1.17"
phase: 2
epic: E-1
wave: backlog
priority: P2
scope_phase: E
points: 8
inputs:
  - .factory/decisions/S-BL.LOOPBACK-FULLSTACK-placement-note.md
  - .factory/specs/verification-properties/VP-042.md
  - .factory/specs/architecture/ARCH-08-dependency-graph.md
  - .factory/specs/architecture/ARCH-03-routing-engine.md
input-hash: "4902d5d"
traces_to: .factory/decisions/S-BL.LOOPBACK-FULLSTACK-placement-note.md
behavioral_contracts:
  - BC-2.01.001   # timeslice clock fires every tick regardless of data availability
  - BC-2.01.002   # empty-tick frame semantics
  - BC-2.02.001   # duplicate-and-race dispatch
  - BC-2.02.002   # endpoint checksum-only dedup
  - BC-2.02.005   # downstream ARQ (piggybacked ACK/SACK, TLPKTDROP)
verification_properties:
  - VP-042   # keystroke-to-echo p99 <= 100ms — harness delivery only; lock flip is a separate subsequent act, see Non-Goals / Forward Obligation
target_module: internal/testenv
estimated_days: null
assumption_validations: []
risk_mitigations: []   # placement note's 5 Risks are note-local (not ASM/R-registry IDs); addressed via AC-001/AC-009/AC-010/AC-011 instead of registry references
bc_traces:
  - BC-2.01.001   # timeslice clock fires every tick regardless of data availability
  - BC-2.01.002   # empty-tick frame semantics
  - BC-2.02.001   # duplicate-and-race dispatch
  - BC-2.02.002   # endpoint checksum-only dedup
  - BC-2.02.005   # downstream ARQ (piggybacked ACK/SACK, TLPKTDROP)
vp_traces:
  - VP-042   # keystroke-to-echo p99 <= 100ms — harness delivery only; lock flip is a separate subsequent act, see Non-Goals / Forward Obligation
subsystems: [transport-layer, quality-observability, session-networking]
architecture_modules:
  - internal/testenv
  - internal/halfchannel
  - internal/arq
  - internal/multipath
  - internal/paths
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: []   # S-BL.TESTENV already MERGED (PR #110, 62e38d3) — this story extends its NewLoopback/LoopbackEnv API; it is not blocked on that story, it builds on shipped code
blocks: []
inputDocuments:
  - '.factory/decisions/S-BL.LOOPBACK-FULLSTACK-placement-note.md'   # v1.15 — BINDING. Q1-Q8 + Non-Goals + Package Impact + 5 Risks, PLUS Q4 Addendum (AC-001 Sign-off) + v1.2 Design Repair Addendum (B1/H1/H2/H3/M2/M4) + v1.3 R1 Re-Review Repair (B-F1/B-F2/B-F3/C-F1/C-F2) + v1.4 R2 Re-Review Repair (B-1/B-2/B-3/N-1) + v1.5 R3 Re-Review Repair (F-B-1/F-B-4/F-B-2/F-B-2b) + v1.6 R4 Re-Review Repair (F-B-LENSB-01/L-C-1/F-A-1) + v1.7 F-A-1 propagation + v1.8 R5 Re-Review Repair (F-LENSB-B-01/F-LENSB-B-02/F-LENSB-B-03/A-L967) + v1.9 R6 Re-Review Repair (BLOCKER — recordingTB stub embed-real-t fix; LOW — sessionName storage/timing pin) + v1.10 R7 Re-Review Repair (MED F-LENSB-01 — both tickers CreateSession-started, AC-017 single-goroutine-by-construction; LOW F-LENSB-02 — AC-017 fault reattributed to un-attached console key; t.Helper() line-ref erratum :460→:461) + v1.11 R8 slash-form line-ref repair (F-LENSA-01/F-ORACLE-01 — remaining slash-notation `t.Helper()` citation corrected) + v1.12 consistency-audit remediation (MAJOR — phantom branch `fix/vp-042-testenv-integrated-bench` corrected to the actual merged file, `internal/bench/keystroke_echo_testenv_bench_test.go`, `develop`, PR #121 @ `4c276d9`, `//go:build integration`-tagged; MAJOR — new "Verification Method (AC-013)" subsection binding `go test -tags integration` as the AC-013 verifier, `just bench` adjudicated out of scope; MINOR — Risks item 6 forward obligation for VP-042.md's Proof Harness Skeleton) + v1.13 R15 adversarial reconvergence repair (MED F-R15-LENSB-01 — the v1.12 AC-013 compile-check bullet was itself vacuous: `go build` never compiles `_test.go` files WITH OR WITHOUT `-tags integration`, tag-independent, because `internal/bench/` is test-only; corrected to `go test -tags integration -run '^$' -count=1 ./internal/bench/` as the binding compile-gate, empirically confirmed to fail against an injected compile error where the untagged/build forms did not; LOW F-R15-LENSA-01 — the L62 blanket no-line-number-citations claim reworded to distinguish forbidden self-referential story-line-number citations from permitted disk-verified external-source anchors; LOW F-R15-LENSC-01 — the v1.12 changelog row's occurrence-count overcount corrected from 11 to 10, the internally-consistent enumerated figure) + v1.14 R18 adversarial reconvergence repair (MED F-R18-LENSC-01 — the naked `L256` live-prose citation of the §H2 per-direction-mutex binding rule de-anchored to a mechanism anchor; MED F-R18-LENSB-01 — Q3's upstream `failLoud` placement corrected to the IN-PLACE `SendKeystroke` call site inside `deliverUpstream`, never `upstreamMP.Send`'s return value in the tick body, per the `multipath.Send` sent≥1-returns-nil masking rationale; LOW O-2 — the `inputDocuments` L62 convention clause scoped to exclude changelog/provenance edit-location line refs from the self-referential-line-citation prohibition; LOW F-R18-LENSB-02 — §M2's window-safety invariant off-by-one corrected: `chanSeq` at the first data tick equals the number of empty ticks elapsed **plus one**, the data tick itself incrementing `seq`) + v1.15 R19 adversarial reconvergence repair (MED F-R19-LENSA-01 — this L57 clause and the L72 status-note v1.14 entry were each missing the F-R18-LENSB-02 item though the formal changelog's v1.14 row carried it; both backfilled with the missing item; LOW F-R19-LENSB-01 — the Q3 upstream-flow diagram gains the in-place `failLoud` step after `SendKeystroke`, propagating the v1.14 F-R18-LENSB-01 fix to this diagram, which the v1.14 burst had missed) + v1.16 R20 adversarial reconvergence repair (MED F-R20-LENSA-01/F-R20-LENSC-01 — this L57 clause's v1.14 item was itself still missing the O-2 item despite the v1.15 backfill pass, backfilled here so it now enumerates all four v1.14 items, matching the L72 status-note entry and the formal changelog's v1.14 row; MED F-R20-LENSA-02/F-R20-LENSC-02 — F-R19-LENSB-01 above relabeled LOW, was MED, matching the formal changelog's authoritative v1.15 row; also backfilled the v1.13 clause above with F-R15-LENSA-01/F-R15-LENSC-01, present in L72's v1.13 entry and the formal changelog's v1.13 row but missing here; declares the ledger-parity convention — this clause and the L72 status-note entry each mirror their formal-changelog row's finding-set 1:1, version-for-version, from v1.13 forward — see the Changelog section header note. Placement note unchanged at v1.15; this is a story-ledger-only repair, no note-version pin bump) + v1.17 R21 adversarial reconvergence repair (MED F-R21-LENSB-01 — downstream `OnAck` call-site re-anchored to the binding note: tick-body placement after `Send` returns, not chained off/gated-by `deliverDownstream`'s dedup, which put the tick-body-local `chanSeq` out of lexical scope and diverged from the note's tick-body-call binding; `multipath.Frame` has no `ChanSeq`, so the prior structure was not cleanly implementable; upstream already correct, untouched; repaired across 8 sites — the Q4 downstream diagram, AC-005, AC-006, Task 6, the BC-2.02.002/BC-2.02.005 Anchors Consumed rows, the Design Constraints `arq.OnAck` call-contract prose, and the Edge Cases duplicate-frame row; placement note unchanged at v1.15, input-hash unchanged at `4902d5d`, story-body-only repair). Where this story and the note diverge, the note governs.
  - '.factory/specs/verification-properties/VP-042.md'
  - '.factory/specs/architecture/ARCH-08-dependency-graph.md'   # v2.13 — this story's merge finalizes the PROSPECTIVE pos-23 import-set amendment
  - '.factory/specs/architecture/ARCH-03-routing-engine.md'
  - '.factory/stories/S-BL.TESTENV.md'
  - '.factory/stories/S-BL.PE-RECEIVE-LOOP.md'   # precedent for the Env.wg/closeCh ticker-goroutine idiom (Q6) and story-writer conventions (grep-resolved symbols; no self-referential story-line-number citations in live binding prose — changelog/provenance edit-location line refs excepted; disk-verified external-source anchors permitted) [v1.14 O-2 scoping]
acceptance_criteria_count: 17
backlog_origin:
  source: architect design note
  adjudication: "Human disposition, 2026-07-12: author now, deliver later — status draft, unscheduled. Not an adversarial-pass or PO-adjudication origin; commissioned directly to answer the open design questions VP-042.md v1.3's own history flagged (\"lock deferred to a testenv-integrated measurement post S-BL.TESTENV\") and to finalize ARCH-08 v2.13's PROSPECTIVE registration."
  drift_items_consumed: []
---

# S-BL.LOOPBACK-FULLSTACK: Full-Stack Loopback Testenv Extension for VP-042

> **Status note:** This story is authored to full spec but is deliberately **draft / unscheduled** per human disposition (2026-07-12) — "author now, deliver later." AC-001 (the `arq.OnAck` sign-off gate) is **DISCHARGED** (2026-07-12, verdict REVISED — see AC-001 below): the value convention is confirmed, and implementation is bound to the single-shared-instance topology from the Q4 Addendum. **v1.2 (2026-07-22):** adversarial spec-review repairs applied (B1/H1/H2/H3/M1/M2/M3/M4 + LOW) — consistent with placement-note v1.2. **v1.3 (2026-07-22):** note v1.3 transcription + R1 story fixes applied (B-F1/B-F2/B-F3/A-M1/A-M2/A-L1/A-L2) — consistent with placement-note v1.3. **v1.4 (2026-07-22):** note v1.4 transcription + R2 story fixes applied (B-1 decodeRTID 2-value, B-3 AC-016 sound injection via onDownstreamTick seam, B-4 new AC-017 upstream-error-loud, A-N2/C-LOW1) — consistent with placement-note v1.4. **v1.5 (2026-07-22):** note v1.5 transcription + R3 story fixes applied (F-B-1 tick-free startLoopbackTicker + seam wiring, F-B-4 errCh dropped, F-1 decodeRTID/changelog, F-2/C task-ref, F-2/A AC-017 cross-ref) — consistent with placement-note v1.5. **v1.6 (2026-07-23):** note v1.7 transcription + R4 story repairs applied (F-B-LENSB-01 AC-017 direct-seam respec + onUpstreamTick() binding, F-A-1 RoundTrip doc-comment 1-value token) — consistent with placement-note v1.7. **v1.7 (2026-08-28):** note v1.8 transcription + R5 story repairs applied (F-LENSB-B-01 driver-lifecycle pin, F-LENSB-B-02 SendKeystroke no-validation, F-LENSB-B-03 recording testing.TB) — consistent with placement-note v1.8. **v1.8 (2026-08-28):** note v1.9 transcription + R6 story repairs applied (BLOCKER — the F-LENSB-B-03 `recordingTB` stub is corrected from `&recordingTB{}` (nil embed, nil-panics at construction) to `&recordingTB{TB: t}` (embeds the real enclosing `*testing.T`, required so `Helper`/`Cleanup`/`Fatalf` don't nil-panic; `Errorf` alone is overridden, so the enclosing test is never marked failed); LOW — `sessionName` storage/timing pin added to Design Constraints) — consistent with placement-note v1.9. **v1.9 (2026-08-28):** note v1.10 transcription + R7 story repairs applied (MED F-LENSB-01 — the upstream ticker's "MAY start at construction" latitude is WITHDRAWN; both tickers now start at `CreateSession` time, symmetric; AC-017's single-goroutine independence from ticker-start timing is TRUE BY CONSTRUCTION, closing a latent `go test -race` hazard on the unlocked `stub.errorfCalls` read; LOW F-LENSB-02 — AC-017's `ErrConsoleNotFound` is reattributed primarily to the un-attached `loopbackConsoleKey` per `internal/session/upstream.go:288-290`, with the empty `sessionName` demoted to a redundant second guarantee; erratum — two live-text `t.Helper()` citations corrected `:460`→`:461`) — consistent with placement-note v1.10. **v1.10 (2026-08-28):** note v1.11 transcription + R8 story repair applied (the third live `t.Helper()` citation corrected in SLASH notation, `testenv.go:384/460/475/528`→`384/461/475/528`, in the §M2 `recordingTB` struct-field comment — completing the `:460`→`:461` citation-class sweep the v1.9 space-colon erratum missed) — consistent with placement-note v1.11. **v1.11 (2026-08-28):** R11 spec-review repair (F-LENSA-R11-01 LOW / F-LENSC-R11-01) — this status-note ledger backfilled with the missing v1.10 entry (above) and extended with this v1.11 entry, restoring the per-version status-note convention; no design-content change, placement-note unchanged (v1.11) — consistent with placement-note v1.11. **v1.12 (2026-08-28):** consistency-audit remediation (POL-005 dispatch, `.factory/cycles/cycle-1/S-BL.LOOPBACK-FULLSTACK/consistency-audit-2026-08-28.md`) — consistent with placement-note v1.12. MAJOR (Finding #1) — every live-prose citation of the phantom branch `fix/vp-042-testenv-integrated-bench` (Story-Sizing Rationale, Q2 Design Constraints, AC-013, Package Impact Summary, Token Budget, Task 10, Previous Story Intelligence, File Structure Requirements) corrected to the actual merged state: `internal/bench/keystroke_echo_testenv_bench_test.go` is MERGED on `develop` (PR #121 @ `4c276d9`), `//go:build integration`-tagged, defines `BenchmarkKeystrokeToEcho_P99`; the branch was never real. MAJOR (Finding #2) — AC-013's verification method corrected from bare `go build ./internal/bench/...` + `just bench` (neither reaches the integration-tagged benchmark) to the binding `go test -tags integration -run '^$' -bench=BenchmarkKeystrokeToEcho_P99 -benchtime=1x -count=1 ./internal/bench/` invocation; `just bench` explicitly adjudicated OUT of AC-013's scope (it runs the separate, untagged `BenchmarkKeystrokeEcho_P99` lower-bound function — S-BL.BENCH's recipe, unaffected by this story). MEDIUM (Finding #3) — the Previous Story Intelligence row for S-BL.PE-RECEIVE-LOOP's line-number-citation claim reconciled to as-practiced reality: self-referential story-symbol/story-line-number citations are forbidden, but disk-verified external-source anchors (`internal/session/upstream.go:288-290`, `internal/testenv/testenv.go:461`) are permitted and practiced throughout this story. MINOR (Finding #4) — new File Structure Requirements row encoding the forward obligation that this story's binding API shape supersedes VP-042.md's Proof Harness Skeleton (placement note Risks item 6). **v1.13 (2026-08-29):** R15 adversarial reconvergence repair (POL-005 dispatch) — consistent with placement-note v1.13. MED (F-R15-LENSB-01) — AC-013's Verification Method compile-check corrected: the v1.12 `go build -tags integration ./internal/bench/...` bullet was itself vacuous (`go build` never compiles `_test.go` files, tag-independent — `internal/bench/` is test-only); replaced with the binding compile-gate `go test -tags integration -run '^$' -count=1 ./internal/bench/`; the binding RUN command is unchanged. LOW (F-R15-LENSA-01) — the `inputDocuments` L62 blanket "no line-number citations" claim reworded to distinguish forbidden self-referential story-line-number citations from permitted disk-verified external-source anchors, matching the Previous Story Intelligence reconciliation already made in v1.12. LOW (F-R15-LENSC-01) — the v1.12 changelog row's "11 occurrences" overcount corrected to 10 (the internally-consistent enumerated figure) via this v1.13 row, per §2.9 (frozen rows are not edited in place). **v1.14 (2026-08-29):** R18 adversarial reconvergence repair (POL-005 dispatch) — consistent with placement-note v1.14. MED (F-R18-LENSC-01) — the one live-prose citation of a stale naked `L256` line number as the per-direction-mutex binding rule (§Q6 wiring paragraph) de-anchored: `L256` on disk is `loopbackSink.SendInput`'s doc comment, unrelated to the mutex rule; replaced with the note's mechanism anchor ("the §H2 per-direction-mutex binding rule — every `Enqueue`/`Tick` call site acquires the corresponding mutex"), carrying no naked line number; whole-file grep confirmed no other live naked `L<digits>` binding-rule citations exist. MED (F-R18-LENSB-01) — AC-017's upstream `failLoud` placement corrected to match the note's Q3/AC-017 ruling: the check MUST be pinned IN-PLACE at the `SendKeystroke` call site inside `deliverUpstream`, never on `upstreamMP.Send`'s return value in the upstream tick body — `multipath.Send` returns `nil` whenever at least one selected path's `Sent` is true, and the dedup sibling path always returns `nil` without ever calling `SendKeystroke`, masking a failing delivering path's error from any post-`Send` check; `deliverUpstream` still returns the error afterward (additive to the return, not a replacement). AC-017's "Required behavior" paragraph, Task 5, and Task 13 corrected to the in-place placement; downstream (`OnAck`) wording untouched — already correct. LOW (O-2) — the `inputDocuments` L62 convention clause scoped to exclude changelog/provenance edit-location line refs from the self-referential-line-citation prohibition. LOW (F-R18-LENSB-02) — §M2's window-safety invariant off-by-one corrected: `chanSeq` at the first data tick equals the number of empty ticks elapsed between `CreateSession` and the first `SendKeystroke` **plus one** (the data tick itself increments `seq`), matching AC-016's own boundary and the placement note's F-B-2b fix; omitted from this entry until now, backfilled here to match the formal changelog's v1.14 row. **v1.15 (2026-08-29):** R19 adversarial reconvergence repair (POL-005 dispatch) — consistent with placement-note v1.15. MED (F-R19-LENSA-01) — this status-note's v1.14 entry (above) and the `inputDocuments` L57 v1.14 clause were each missing the F-R18-LENSB-02 item though the formal changelog's v1.14 row carried it; both backfilled with the missing item. LOW (F-R19-LENSB-01) — the Q3 upstream-flow diagram (§Q3, ~L349) gains the in-place `failLoud` step immediately after `driver.accessNode.SendKeystroke`, mirroring AC-017's already-correct in-place placement and the Q4 downstream diagram's symmetric `if err != nil { driver.failLoud(err); return }` pattern; the v1.14 R18 fix propagated to AC-017/Task 5/Task 13/this status note but had missed the Q3 diagram itself. No design-content change beyond these two propagation repairs. **v1.16 (2026-08-29):** R20 adversarial reconvergence repair (POL-005 dispatch), story-ledger-only — placement note unchanged at v1.15. MED (F-R20-LENSA-01/F-R20-LENSC-01) — the `inputDocuments` L57 v1.14 clause was still missing the O-2 item, present here and in the formal changelog's v1.14 row; backfilled there to match. MED (F-R20-LENSA-02/F-R20-LENSC-02) — F-R19-LENSB-01 above is relabeled LOW (was MED), matching the formal changelog's authoritative v1.15 row. Declares the ledger-parity convention (see the Changelog section header note) and, applying it retroactively, backfills L57's v1.13 clause with F-R15-LENSA-01/F-R15-LENSC-01, already present here and in the formal changelog's v1.13 row — consistent with placement-note v1.15 (unchanged). **v1.17 (2026-08-29):** R21 adversarial reconvergence repair (POL-005 dispatch, finding F-R21-LENSB-01, MEDIUM), story-body-only — placement note unchanged at v1.15. Downstream `OnAck` call-site re-anchored to the binding note: `OnAck` runs once per tick IN THE TICK BODY after `Send` returns, structurally decoupled from `deliverDownstream`'s dedup — not chained off/gated-by `deliverDownstream` as previously depicted, which put the tick-body-local `chanSeq` out of lexical scope and was not cleanly implementable (`multipath.Frame` has no `ChanSeq`). Upstream (`SendKeystroke` inside `deliverUpstream`) was already correct and is untouched. Repaired across 8 sites: the Q4 downstream diagram, AC-005, AC-006, Task 6, the BC-2.02.002/BC-2.02.005 Anchors Consumed rows, the Design Constraints `arq.OnAck` call-contract prose, and the Edge Cases duplicate-frame row. AC count 17 unchanged; input-hash unchanged at `4902d5d`. Do not implement from Q4's original code blocks alone — the Addendum and v1.2/v1.3/v1.4/v1.5/v1.6/v1.7/v1.8/v1.9/v1.10/v1.11/v1.12/v1.13/v1.14/v1.15/v1.16/**v1.17** corrections govern.

## Narrative

- **As a** verification engineer trying to lock VP-042 (keystroke-to-echo p99 ≤ 100ms)
- **I want** `internal/testenv`'s `NewLoopback`/`LoopbackEnv` extended from a same-goroutine
  `DeliverFrame` shortcut into a tick-driven, protocol-accurate loopback stack spanning
  `internal/halfchannel` + `internal/arq` + `internal/multipath` + `internal/paths`
- **So that** VP-042's benchmark measures the real round-trip path (tick cadence, duplicate-and-race
  dispatch, endpoint dedup, downstream ARQ bookkeeping) instead of an in-process echo shortcut that
  bypasses all of it, and the harness can be run once to produce honest evidence for a future
  `verification_lock` decision

## Context

`S-BL.TESTENV` (merged PR #110, `62e38d3`) shipped `internal/testenv` including `NewLoopback` and
`LoopbackEnv`, but `NewLoopback` discards its `LoopbackConfig` and calls `newEnv(ctx, b, 1)` —
`LoopbackConfig.TickIntervalUpstream`/`TickIntervalDownstream` are dead fields. `Env.SendKeystroke` does
not go through `session.AccessNode.SendKeystroke`/`KeystrokeSink` at all; it directly calls
`sh.access.DeliverFrame(hdr)`, synthesizing a downstream fan-out frame under the name "SendKeystroke."
There is no tick scheduler anywhere in the path. `S-BL.BENCH` (merged PR #109, `cd67394`) recorded
VP-042 as **adopted-partial**: an honest lower-bound-only measurement (in-process loopback echo p99
~0.002ms vs the 100ms limit) with a declared divergence — the inline echo path bypasses
`arq`/`multipath`/tick-scheduling entirely. VP-042 v1.3's own changelog states the lock is "deferred to
a testenv-integrated measurement post S-BL.TESTENV."

This story is that testenv-integrated measurement. It is scoped and designed entirely by the architect
design note listed as this story's binding input
(`.factory/decisions/S-BL.LOOPBACK-FULLSTACK-placement-note.md` v1.15) — **story-writer's job here is
transcription, not re-derivation.** Where this story and the placement note appear to diverge, the note
governs; where this story and VP-042.md's older proof-harness skeleton diverge (the skeleton's two-call
`env.SendKeystroke`/`env.WaitForEcho` shape vs. this story's token-based `RoundTrip` API), the placement
note's shape is binding — the skeleton predates the discovery (Q5) that a token is required to fix a
distinct accumulation bug in `Env.CollectFrames`.

**AC-001 sign-off (2026-07-12, verdict REVISED):** the note's Q4 was reviewed against `arq.go`'s full
test suite, `internal/arqsend`, and ARCH-03 §Downstream ARQ before this story could be scheduled. The
`ackSeq`/SACK value convention is CONFIRMED correct as originally written. The `driver.arqServer`/
`driver.arqClient` two-instance topology Q4's code blocks show is a structural defect — `OnAck`'s
payload recovery (`payloadFor`) reads only the calling instance's own `inFlight`/`reorderBuf`, populated
exclusively by that SAME instance's prior `EnqueueSend` calls; a `arqClient` that never receives
`EnqueueSend` returns `(nil, nil)` from `OnAck` on every call, forever, so every `WaitForEcho` would time
out — not a subtle correctness gap, a hard benchmark failure. The required fix, binding on this story per
AC-001 below, is a single shared `*arq.ARQ` instance for the downstream direction (e.g.
`driver.downstreamARQ`). See the placement note's "Q4 Addendum — AC-001 Sign-off (2026-07-12)" for the
full reasoning trail.

**Also discharged by this story:** ARCH-08 v2.13's PROSPECTIVE amendment to `internal/testenv`'s §6.5
pos-23 import set — `{admission, drain, frame, outerassembler, session, upstreamdial}` →
`{admission, arq, drain, frame, halfchannel, multipath, outerassembler, paths, session, upstreamdial}`
— becomes final, machine-verified (`go list`), at this story's merge, per the same protocol used for
every prior testenv import-set change (v2.5, v2.8, v2.11).

## Story-Sizing Rationale (points: 8, architect range 5–8)

The placement note's own estimate is 5–8 points, broken down as: tick-driving (Q6) is low-risk and
small — a direct copy of an idiom already used twice in `testenv.go` (`AttachConsole`/`AttachProbe`)
and twice more in `cmd/switchboard/access.go`; multipath wiring (Q3, Q7) is low-risk — small,
well-tested pure APIs, a few lines of synthetic path construction; the round-trip-token API (Q5)
touches the merged bench test and VP-042.md's skeleton, small but real fan-out. **The ARQ wiring (Q4) is
the size and risk driver** — it commits to a call contract (`arq.OnAck`'s `ackSeq` semantics) that has
no existing production precedent to copy, and the note itself flags that commitment as needing
architect/adversarial sign-off before an implementer treats it as settled (Risk 1 below).

Story-writer selects the **upper end of the range (8)**, not the midpoint, for three reasons beyond the
placement note's own text: (1) AC-001 is a hard pre-implementation gate, not just a risk note — it adds
real process latency before `dev-story` can properly start, which the note's code-size-only estimate
doesn't price in; (2) four of the five Risks (not just Risk 1) resolve into their own gating or
decision-bearing ACs (AC-009, AC-010, AC-011) rather than being absorbed silently into the main
implementation tasks; (3) the merged-bench cross-reference (Package Impact, `internal/bench` row) is real
fan-out into an already-merged, `//go:build integration`-tagged file
(`internal/bench/keystroke_echo_testenv_bench_test.go`, `develop`, PR #121 @ `4c276d9`) that must be
updated in place to the new token-based API shape, which is coordination overhead the tick/multipath/token
estimate doesn't include.

**AC-001 gate resolved pre-scheduling (2026-07-12):** the gate priced into reason (1) has now been
discharged — verdict REVISED (single shared `*arq.ARQ` instance required; see AC-001) — before this
story left draft/unscheduled status. Resolution surfaced a structural topology defect, not a value-
convention question, but the fix is scoped entirely inside Task 6's existing wiring work; it does not
add a new task, package, or test file beyond the new regression-guard AC (AC-014). No scope growth —
the estimate stays 8 points.

## Anchors Consumed

| Anchor | Verbatim ID | Source | Disposition |
|--------|-------------|--------|-------------|
| Timeslice clock fires on every tick regardless of data availability | BC-2.01.001 | VP-042 Source Contract; placement note Q3, Q6 | TO DISCHARGE (harness-scope) — upstream/downstream ticker goroutines call `HalfChannel.Tick()` on a fixed schedule per `cfg.TickIntervalUpstream`/`TickIntervalDownstream`, independent of `Enqueue` timing; `NewLoopback` validates both intervals against `halfchannel.MinTickInterval`/`MaxTickInterval` |
| Empty-tick frame semantics | BC-2.01.002 | placement note Q1, Non-Goals | TO DISCHARGE (partial, harness-scope) — `Tick()` produces an empty-tick frame on schedule when nothing is enqueued; this story does NOT wire-dispatch empty ticks over multipath (Non-Goals) — a harness-scope boundary, not a production behavior change |
| Duplicate-and-race: same frame sent on two fastest paths simultaneously | BC-2.02.001 | VP-042 Source Contract; placement note Q3, Q7 | TO DISCHARGE — `multipath.Send` dispatches every payload over both synthetic `paths.RankedPath`s per direction; `deliverUpstream`/`deliverDownstream` is called once per selected path |
| Endpoint checksum-only dedup | BC-2.02.002 | placement note frontmatter; Q3 | TO DISCHARGE — `multipath.Receive` returns `ErrDuplicate` on the second-arriving copy of a duplicate-and-raced frame, discarded in `deliverUpstream`/`deliverDownstream`; upstream this gates `accessNode.SendKeystroke` (called only on the first-arriving path); downstream it gates only `deliverDownstream` itself — `downstreamARQ.OnAck` is called once per tick, unconditionally, and is not reached from `deliverDownstream` (AC-005, AC-006) |
| Downstream ARQ (piggybacked ACK/SACK, TLPKTDROP) | BC-2.02.005 | placement note Q1, Q4 + Q4 Addendum | TO DISCHARGE (downstream leg only — upstream ARQ is explicitly out of scope per Q1/ARCH-03) — every downstream tick's data frame passes through `driver.downstreamARQ.EnqueueSend`; the SAME tick body then calls the SAME `driver.downstreamARQ.OnAck` exactly once, in the tick body after `Send`/`deliverDownstream` return, per the Q4 call-contract (single shared instance — AC-001 **DISCHARGED**, verdict REVISED) |
| Keystroke-to-echo p99 ≤ 100ms | VP-042 | VP-042.md | HARNESS DELIVERED, NOT LOCKED — this story ships the measurement harness and runs it once for evidence; the `verification_lock` flip is a separate subsequent act (see Forward Obligation) |

---

## Design Constraints

The following subsections transcribe the placement note's binding decisions (Q2–Q8). They are not
re-derived here; where a code sketch is reproduced, it is the note's sketch, not a new one.

### Loopback Driver Ownership and Dedicated Shard (Q2)

**Binding (per placement note Q2).**

A new unexported `loopbackDriver` type lives inside `internal/testenv`, owned by `LoopbackEnv`.
`SendKeystroke`/`WaitForEcho`/`CreateSession` are **new methods on `*LoopbackEnv`**, not on `*Env`.
`LoopbackEnv` is `struct { Env *Env }` — a named field, not Go embedding (confirmed: the existing merged
bench test (`internal/bench/keystroke_echo_testenv_bench_test.go`, `develop`, PR #121 @ `4c276d9`) does
`env := lb.Env; env.CreateSession(b)`, never `lb.CreateSession(b)`) — so new
`*LoopbackEnv` methods do not collide with or shadow `*Env`'s method set.

**[H3 — v1.3] Console provisioning (required for upstream happy path).** `AccessNode.SendKeystroke`
returns `ErrConsoleNotFound` unless the console key is registered and attached. Console provisioning
happens ONLY in `LoopbackEnv.CreateSession` — never in the `loopbackDriver` constructor. [v1.8
F-LENSB-B-01] This is a hard boundary, not a style preference: AC-017 requires building the driver and
calling `SendKeystroke` + `onUpstreamTick()` synchronously BEFORE `CreateSession` runs, and observing
`ErrConsoleNotFound` (via `failLoud`) as the result. If the `loopbackDriver` constructor provisioned the
console eagerly, that pre-`CreateSession` `SendKeystroke` call would already have an attached console
available by the time `onUpstreamTick()` processes it — `accessNode.SendKeystroke` would SUCCEED,
`failLoud` would never fire, and AC-017's step-4 assertion would fail. Construction-time provisioning is
therefore withdrawn as an implementation option.

**Driver lifecycle pin [v1.8 F-LENSB-B-01]:** the `loopbackDriver` constructor (invoked once, from
`NewLoopback`, before `CreateSession` is ever called) builds ALL of the following, fully initialized and
immediately usable, but with the console UN-PROVISIONED (no `Publish`/`RegisterKey`/`Attach` has run):

- The `Publisher`/`SessionAuth`/`AccessNode` triple (Q2).
- BOTH `*multipath.Multipath` instances, `upstreamMP` and `downstreamMP` (Q7).
- BOTH `*halfchannel.HalfChannel` instances, `upstreamHC` and `downstreamHC` (H2 above).

Concretely: `SendKeystroke` (enqueues into `upstreamHC`), `onUpstreamTick()` (dequeues from `upstreamHC`,
drives `upstreamMP.Send`/`Receive`, then calls `accessNode.SendKeystroke`), and `onDownstreamTick()` are
ALL safely callable on a freshly-constructed, pre-`CreateSession` driver — none of them nil-deref, because
`upstreamMP`/`downstreamMP`/`upstreamHC`/`downstreamHC` are built AT CONSTRUCTION, not lazily at
`CreateSession`. Only the console's session-level authorization state (Publish/RegisterKey/Attach, below)
is deferred to `CreateSession`. `CreateSession` therefore does exactly two things: (1) the
console-provisioning sequence below, and (2) starting BOTH ticker goroutines — downstream and upstream
(M2's preferred `CreateSession`-time start, [v1.10 F-LENSB-01] now symmetric for both tickers) — it
does NOT construct the driver, the multipath instances, or the half-channels; those already exist from
`NewLoopback`.

```go
// Session provisioning — called ONLY from CreateSession, never from the
// loopbackDriver constructor (the driver, its AccessNode, and both
// multipath/half-channel pairs already exist and are usable — see
// "Driver lifecycle pin" above; only console attachment is deferred here):
// sh is the driver's own dedicated Publisher/SessionAuth/AccessNode triple (Q2):
//   sh.pub   = driver.pub   (*session.Publisher)
//   sh.auth  = driver.auth  (*session.SessionAuth)
//   sh.access = driver.access (*session.AccessNode, constructed with WithKeystrokeSink(loopbackSink))
loopbackConsoleKey := driver.env.newConsoleKey()   // opaque ConsoleKey

// [v1.3 B-F1] Publish into the driver's OWN dedicated Publisher BEFORE Attach.
// AccessNode.Attach calls pub.Get(sessionName) as its first gate; if the session
// is not published, Attach returns ErrSessionNotFound and t.Fatalf fires at
// construction — the happy path is unreachable without this step.
// Publisher.Publish signature: func (p *Publisher) Publish(sessionName string) error
if err := sh.pub.Publish(sessionName); err != nil {
    t.Fatalf("loopbackDriver: Publish session %q: %v", sessionName, err)
}
sh.auth.RegisterKey(sessionName, loopbackConsoleKey, session.RoleFull)
downstream, _, err := sh.access.Attach(loopbackConsoleKey, sessionName)
if err != nil {
    t.Fatalf("loopbackDriver: Attach loopback console: %v", err)
}
_ = downstream  // downstream channel not used — echo delivery flows through
                 // loopbackSink → downstreamHC, not AccessNode.DeliverFrame fan-out
```

**Complete provisioning sequence (v1.3 — steps MUST appear in this order):**
1. `sh.pub.Publish(sessionName)` — publish into the driver's own dedicated Publisher so `Attach`'s `pub.Get` gate passes.
2. `sh.auth.RegisterKey(sessionName, loopbackConsoleKey, session.RoleFull)` — register the console key so `SendKeystroke`'s authorizer check passes.
3. `sh.access.Attach(loopbackConsoleKey, sessionName)` — attach the console to the session.

[v1.8 F-LENSB-B-01] Steps 1–3 are exactly the console-provisioning work `CreateSession` performs;
`CreateSession` additionally starts both ticker goroutines — downstream and upstream ([v1.10 F-LENSB-01]
M2's preferred `CreateSession`-time start, now symmetric for both) — as a separate concern unordered
relative to 1–3 — see "Driver lifecycle pin" above.

`loopbackConsoleKey` is stored on the driver and passed to
`driver.accessNode.SendKeystroke(loopbackConsoleKey, sessionName, payload)` in the upstream delivery
callback (Q3). Without this step every upstream keystroke returns `ErrConsoleNotFound` — AC-004/005/006/014
happy paths all time out.

**`sessionName` storage/timing pin [v1.9 R6 LOW fix, Lens B O-1; v1.10 F-LENSB-02 fault-attribution
correction]:** `sessionName` is likewise stored as a
`loopbackDriver` field (e.g. `driver.sessionName`), set once in `CreateSession` alongside
`loopbackConsoleKey`, at the same point — before Steps 1–3 above run (step 1, `sh.pub.Publish(sessionName)`,
is `sessionName`'s first use, so the field is populated no later than immediately before that call).
Consistent with the "Driver lifecycle pin" above (construction leaves the driver un-provisioned;
`CreateSession` provisions it), `driver.sessionName` holds its zero value (`""`) at any point before
`CreateSession` has run. [v1.10 F-LENSB-02] The PRIMARY cause of AC-017's `ErrConsoleNotFound` is the
UN-ATTACHED, zero-value `loopbackConsoleKey` — not the empty `sessionName`. `AccessNode.SendKeystroke`
(`internal/session/upstream.go:276-301`) takes `sinkMu`, then at line 288 looks up
`attached, ok := a.consoles.Session(key)`; because `loopbackConsoleKey` has never been `Attach`ed (that
only happens inside `CreateSession`'s Steps 1–3, which have not run before AC-017's call), this lookup
returns `ok == false`, and the console-attachment gate at lines 289-290 returns `ErrConsoleNotFound`
IMMEDIATELY — before `sessionName` is ever read. The `attached != sessionName` comparison at line 292
(which would return `ErrSessionMismatch`) is unreached on this path. The empty `sessionName` is therefore
a REDUNDANT second guarantee, not the operative cause: had some OTHER session already been published
while `loopbackConsoleKey` remained unattached, the same `ErrConsoleNotFound` would fire at lines 289-290
for the same reason; had `loopbackConsoleKey` instead been attached to a real session while `sessionName`
stayed `""`, the attachment gate would pass but the `attached != ""` comparison at line 292 would then
return `ErrSessionMismatch` — either way a loud error, never a silent success.
For the happy-path ACs (AC-004/005/006 and others), `driver.sessionName` is populated and stable by the
time any upstream tick can reach the delivery callback, because `CreateSession` sets it before returning
and no `SendKeystroke` call that matters to those ACs is made before `CreateSession` completes.

`Env.SendKeystroke`/`Env.CollectFrames` are **not** extended in place: those methods back 10 other VPs
via generic SVTN-shard fan-out semantics that none of them asked to become tick-driven or
round-trip-tagged. `NewLoopback` keeps calling `newEnv(ctx, b, 1)` (so `lb.Env.Close()`/generic surface
stay available, harmless if unused); `LoopbackEnv` additionally constructs and owns a `*loopbackDriver`
with its own dedicated session/shard.

The driver needs a **dedicated shard**, not `env.defaultShard`: `newShard` hardcodes
`session.WithKeystrokeSink(session.NoOpSink{})`, and `session.AccessNode` has no `SetSink` — the
`KeystrokeSink` is fixed at construction via functional option, by design (a mutable-sink escape hatch
would weaken that invariant for every other `AccessNode` consumer, not just testenv). The loopback
driver instead builds its own `Publisher`/`SessionAuth`/`AccessNode` triple — identical in shape to
`newShard`, but with `WithKeystrokeSink(loopbackSink)` from the start, where `loopbackSink` is the
driver's own echo-generating sink (Q4). This duplication is isolated to the loopback path; it does not
touch `newShard` or any other VP's shard, and it does not add a `SetSink` escape hatch to production
`session.AccessNode`.

**[H2 — v1.2] Per-direction HalfChannel mutex.** `halfchannel.HalfChannel` is not safe for concurrent
use (`Tick` and `Enqueue` must be called from a single goroutine or under external synchronisation per
the halfchannel concurrency contract). Each half-channel is accessed from two goroutines: `upstreamHC`
from the test goroutine (`SendKeystroke` calls `Enqueue`) and the upstream ticker goroutine (calls
`Tick`); `downstreamHC` from the upstream ticker goroutine (`loopbackSink.SendInput` calls `Enqueue`)
and the downstream ticker goroutine (calls `Tick`). `driver.mu` guards only `driver.pending`. The
`loopbackDriver` struct MUST carry per-direction mutexes:

```go
type loopbackDriver struct {
    upstreamHCMu   sync.Mutex  // serializes upstreamHC.Enqueue (test goroutine)
                                 //   + upstreamHC.Tick (upstream ticker goroutine)
    upstreamHC     *halfchannel.HalfChannel
    downstreamHCMu sync.Mutex  // serializes downstreamHC.Enqueue (upstream ticker,
                                 //   via loopbackSink.SendInput)
                                 //   + downstreamHC.Tick (downstream ticker goroutine)
    downstreamHC   *halfchannel.HalfChannel
    // ... other fields unchanged
}
```

Every `Enqueue` and `Tick` call site on each half-channel acquires the corresponding mutex. See AC-015.

### Upstream Flow: Keystroke → Server Delivery (Q3)

**Binding (per placement note Q3).**

```
LoopbackEnv.SendKeystroke(t, sessionID, key)
    mints RoundTrip{id: driver.rtSeq.Add(1)}; registers a completion channel
    under that id in driver.pending, guarded by driver.mu
    payload := encodeRTID(key, id)   // [v1.3 B-F2] 2-arg whole-payload form (key + 8-byte BE id suffix)
    ↓
driver.upstreamHC.Enqueue(payload)   // pure, non-blocking — returns to caller
                                      // immediately; SendKeystroke does NOT
                                      // block on delivery (BC-2.01.001 requires
                                      // the tick to fire on its own schedule
                                      // regardless of enqueue timing)
    ↓
[async] upstream ticker, every cfg.TickIntervalUpstream:
    f := driver.upstreamHC.Tick()
    if f.FrameType == frame.FrameTypeData {
        driver.upstreamMP.Send(toMPFrame(f), driver.deliverUpstream)
    }
    // empty ticks are produced (BC-2.01.002) but not wire-dispatched (Non-Goals)
    ↓
driver.deliverUpstream(pathID, mpFrame) error   // called once per selected
    path (up to 2, duplicate-and-race) — the SAME callback for both, since
    both loopback paths terminate in this one process
    ↓
driver.upstreamMP.Receive(mpFrame)   // endpoint checksum dedup
    ErrDuplicate on second-arriving copy → discard, return nil
    ↓
err := driver.accessNode.SendKeystroke(loopbackConsoleKey, sessionName, mpFrame.Payload)
    // [v1.15 F-R19-LENSB-01] err checked IN-PLACE here, inside deliverUpstream,
    // symmetric with §M2's downstream OnAck pattern (AC-017) — NOT via a check
    // on upstreamMP.Send's return value in the tick body: multipath.Send returns
    // nil whenever sent>=1, and the dedup sibling path's deliverUpstream call
    // always returns nil without ever calling SendKeystroke, so a post-Send
    // check can never observe a failing delivering path's error
    if err != nil {
        driver.failLoud(err)
        return err   // still propagates to multipath.Send — see AC-017
    }
    ↓
loopbackSink.SendInput(payload) error   // Q4
```

**`SendKeystroke` performs no session-existence validation [v1.8 F-LENSB-B-02]:** the first step above —
mint `RoundTrip`, register `driver.pending[id]`, encode `payload`, `Enqueue` into `upstreamHC` — is
UNCONDITIONAL. `SendKeystroke` does NOT check that `sessionID` refers to an existing or provisioned
session before doing any of this. This is deliberate: AC-017 calls `SendKeystroke` BEFORE `CreateSession`,
when the session's console is not yet provisioned (see the "Driver lifecycle pin" above), and depends on
that pre-`CreateSession` call succeeding at the mint/encode/enqueue level — the failure AC-017 exercises
surfaces later and downstream, at `accessNode.SendKeystroke` inside `onUpstreamTick()`
(`ErrConsoleNotFound`, via `failLoud`), not at `SendKeystroke` itself. An implementer who adds a
defensive session-existence guard to `SendKeystroke` would abort AC-017 at step 1, before
`onUpstreamTick()` ever runs, and the test would fail for the wrong reason instead of exercising the
`failLoud` path it is meant to test. `SendKeystroke` MUST remain unconditional in this respect; no
session-existence guard is permitted.

`SendFunc` is called from inside the ticker goroutine, not spawned into its own goroutine per path —
`multipath.Send`'s doc states `fn` is called without holding any internal lock, so real work in `fn` is
safe; with zero synthetic added latency (Non-Goals: no real network) there is no concurrency benefit to
spawning, and running both calls sequentially avoids a class of out-of-order dedup-cache-insertion races
that a fully-faithful network simulation would have to reckon with but this design deliberately does not
model.

### Downstream Flow: Echo Generation → Round-Trip Completion (Q4, as REVISED by the Q4 Addendum) — AC-001 DISCHARGED

**Binding, per placement note Q4 AS AMENDED by the Q4 Addendum — AC-001 Sign-off (2026-07-12,
verdict REVISED).** The `driver.arqServer`/`driver.arqClient` two-instance shape Q4's original code
blocks show below is SUPERSEDED — do not implement it. `EnqueueSend` and `OnAck` for a given `ChanSeq`
MUST be called on ONE shared `*arq.ARQ` instance (`driver.downstreamARQ`), in that order, within the
same downstream-ticker tick. The `ackSeq`/SACK value convention is unaffected and remains binding as
written.

`loopbackSink.SendInput` — the `KeystrokeSink` injected into the driver's dedicated `AccessNode` — is
the echo generator:

```go
func (s *loopbackSink) SendInput(payload []byte) error {
    return s.driver.downstreamHC.Enqueue(payload)   // echoes the FULL payload
}                                                     // verbatim, including the
                                                       // embedded RT-ID — the sink
                                                       // does not need to understand
                                                       // the correlation scheme; it
                                                       // just echoes bytes, like real
                                                       // tmux would
```

`SendInput` is called while `AccessNode` holds `sinkMu` ("must not call back into `AccessNode` under any
lock"); `Enqueue` only touches the downstream `HalfChannel`'s own pending queue, never calling back into
`AccessNode`, so this is safe by construction — and it is the correct modeling of BC-2.01.001: the echo
is queued, not delivered synchronously; the downstream ticker decides when it actually goes out.

```
[async] downstream ticker, every cfg.TickIntervalDownstream
  (started lazily at CreateSession — see M2 mitigation):
    f := driver.downstreamHC.Tick()
    if f.FrameType == frame.FrameTypeData {
        chanSeq := f.ChanSeq   // [B1] capture from ChannelFrame BEFORE toMPFrame;
                                // multipath.Frame has no ChanSeq field/method
        driver.downstreamARQ.EnqueueSend(chanSeq, f.Payload, time.Now())
        driver.downstreamMP.Send(toMPFrame(f), driver.deliverDownstream)
        // driver.deliverDownstream(pathID, mpFrame) error is Send's per-path
        // callback (called once per selected path, up to 2, duplicate-and-race):
        // it calls driver.downstreamMP.Receive(mpFrame) — endpoint dedup; first
        // arrival returns nil, the second-arriving copy returns ErrDuplicate and
        // is discarded (BC-2.02.002/AC-005) — and does nothing else. Send runs
        // both calls synchronously within this goroutine before returning (Q3
        // rationale); chanSeq remains valid, in scope, at the OnAck call site
        // below regardless of how deliverDownstream's dedup resolved.
        delivered, err := driver.downstreamARQ.OnAck(chanSeq, zeroSACK)
        // [B1] uses captured chanSeq, not mpFrame.ChanSeq() (phantom — Frame has no ChanSeq)
        // [M2] err MUST be checked and fail loud — not swallowed
        // SAME instance that received EnqueueSend above, called once per tick
        // (NOT gated by, and not invoked from inside, deliverDownstream's dedup) —
        // required per the Q4 Addendum (AC-001). ackSeq = this frame's own
        // ChanSeq (locally-derived); SACK bitmap all-zero (no loss simulated)
        if err != nil { driver.failLoud(err); return }
        for each payload in delivered:
            id, ok := decodeRTID(payload)   // [v1.4 B-1] 2-value; rtSeq.Add(1) starts ids at 1 so id=0 (the !ok sentinel) never collides with a real pending key
            if !ok { continue }             // payload too short; skip — decode failure cannot be a real pending key
            driver.mu.Lock(); ch := driver.pending[id]; delete(driver.pending, id); driver.mu.Unlock()
            if ch != nil { ch <- payload }   // [H1] chan []byte — sends raw echo payload; unblocks WaitForEcho
    }
```

**`arq.OnAck` call-contract — sign-off DISCHARGED (AC-001, 2026-07-12, verdict REVISED).** No production
code calls `OnAck` today; `internal/arqsend` (the only production consumer of `*arq.ARQ`) only exercises
the sender-side subset (`PayloadForInFlight`/`EnqueueSend`/`RemoveInFlight`). This design is the **first
proposed call site for `OnAck`** in the codebase. The `ackSeq` convention — the highest downstream
`ChanSeq` this receiver has now observed in order (locally-derived from arrival, not a peer-supplied
value), called once per downstream TICK that produces a data frame, from the tick body itself (`OnAck`'s
own doc comment: "the caller's tick loop forwards them to the terminal," `arq.go:195`), with that
frame's own `ChanSeq` — is CONFIRMED correct given a single downstream producer emitting strictly
increasing `ChanSeq` values with no synthetic loss/reordering, and exercises `OnAck`'s real
window-validation (`RULING-003`/`ErrAckOutOfWindow`) and delivery-pointer bookkeeping on every sample.

`GapsToRetransmit`/`TLPKTDROP` are deliberately **not** called — there is no simulated loss, so
`downstreamARQ.inFlight` never accumulates a real gap; wiring an active poll for a condition that
structurally cannot occur in this harness would be dead code (Non-Goals).

**[M2 — v1.2] OnAck error handling (SOUL.md §4 — no silent failure).** The downstream ticker MUST
check `OnAck`'s error return and fail loud — not swallow it. `ErrAckOutOfWindow` is the only expected
error; if it fires, all pending round trips will silently time out, masking a harness construction bug
as "high latency." Required shape:

```go
delivered, err := driver.downstreamARQ.OnAck(chanSeq, zeroSACK)
if err != nil {
    driver.failLoud(fmt.Errorf("downstreamARQ.OnAck seq=%d: %w", chanSeq, err))
    return
}
```

`driver.failLoud` calls `t.Errorf` (not `t.Fatalf`) so the ticker goroutine can return cleanly; the
test is already doomed at this point. [v1.5 F-B-4] The `driver.errCh chan error` alternative is dropped:
a buffered-1 channel deadlocks when both ticker goroutines call `failLoud` concurrently (first fills the
buffer; second blocks forever; `wg.Wait()` hangs). `t.Errorf`-based `failLoud` is the sole specified
error-surface mechanism — goroutine-safe, non-blocking by specification, requires no drain step.

**[M2 — v1.3 B-F3] Empty-tick window mitigation (downstream ticker start timing).** `halfchannel.HalfChannel`
increments its sequence counter on EVERY `Tick()` call, including empty ticks when no payload is
queued. `ErrAckOutOfWindow` fires when `chanSeq - downstreamARQ.nextExpected > 64`. If the downstream
ticker starts at `NewLoopback` construction time but `CreateSession`/`SendKeystroke` are called more
than 64 downstream tick intervals later (64 × 50ms = 3.2s at the standard interval), the first data
tick produces a `chanSeq` exceeding `nextExpected` by more than 64, and the first `OnAck` returns
`ErrAckOutOfWindow`. **Mitigation (PREFERRED): start the downstream ticker goroutine at `CreateSession`
time — called once, from a single goroutine, before any `SendKeystroke` calls, so no additional
synchronization is needed.** [v1.10 F-LENSB-01] The upstream ticker likewise starts at `CreateSession`
time, SYMMETRIC with the downstream ticker — NOT at construction. The prior "upstream ticker MAY start
at construction" latitude is WITHDRAWN: the upstream ticker has no `EnqueueSend` dependency, so nothing
blocks a `CreateSession`-time start either, and the same single-goroutine, race-free argument above
applies to it unchanged. This also makes AC-017's independence from ticker-start timing TRUE BY
CONSTRUCTION rather than a discipline the test must separately uphold — see AC-017 below.
**[v1.4 B-2, corrected v1.14 F-R18-LENSB-02] Window-safety invariant for CreateSession-time start:**
`chanSeq` at the first data tick equals the number of empty ticks elapsed between `CreateSession` and the
first `SendKeystroke` **plus one** (the data tick itself increments `seq` via the unconditional `h.seq++`
in `halfchannel.Tick()`) — NOT guaranteed to be 1 (empty ticks accumulate freely from the moment the
ticker starts). Window safety holds as long as fewer than 64 empty ticks precede the first data frame
(64 × 50ms = 3.2s at the
standard interval). For the VP-042 benchmark, `CreateSession` is immediately followed by the send
loop, so this trivially holds. For tests with a >3.2s idle gap between `CreateSession` and first
`SendKeystroke`, `ErrAckOutOfWindow` will fire — see the Edge Cases table. The preferred option is
chosen for its RACE-FREEDOM (B-F3), not for a stronger window margin.

**If the implementer uses first-`SendKeystroke` start instead, a `sync.Once` guard on ticker launch is
MANDATORY.** Without it, concurrent first `SendKeystroke` calls (AC-008) each observe not-started and
launch duplicate tickers, double-consuming `ChanSeq` values and corrupting the ARQ window. The
`sync.Once` idiom is already house convention in this design (`closeOnce sync.Once`). There is no third
option: `CreateSession`-time start (preferred) or `sync.Once`-guarded first-`SendKeystroke` start.

**[v1.4 B-3, updated v1.5 F-B-1] Required seam:** the downstream tick body MUST be factored into a
package-private method `onDownstreamTick()` on `loopbackDriver` that can be called synchronously without
the ticker goroutine running. [v1.5] `startLoopbackTicker` is now a generic no-arg-callback driver; the
downstream tick body lives entirely in `onDownstreamTick()`. The downstream ticker goroutine invokes this
body via `startLoopbackTicker(env, downstreamInterval, d.onDownstreamTick)` (the `tickBody` argument);
the AC-016 test invokes `onDownstreamTick()` directly, synchronously, without starting the ticker
goroutine — no race. The method name `onDownstreamTick()` is BINDING per note §M2 §B-3.

**[v1.6 F-B-LENSB-01] Required upstream seam:** `onUpstreamTick()` must be a directly-callable
package-private method on `loopbackDriver`, symmetric to `onDownstreamTick()`. The upstream tick body
lives entirely in `onUpstreamTick()`; the upstream ticker goroutine invokes it via
`startLoopbackTicker(env, upstreamInterval, d.onUpstreamTick)` (the `tickBody` argument). [v1.10
F-LENSB-01] Because `CreateSession` is now the SOLE start-site for both tickers (see the Mitigation
decision above), and AC-017 deliberately never calls `CreateSession`, no upstream ticker goroutine can
exist at any point during the AC-017 test — this is TRUE BY CONSTRUCTION, not a discipline the test must
separately uphold. The AC-017 fault-injection test invokes `onUpstreamTick()` directly, synchronously,
as the sole driver of the upstream tick body for the whole test. The method name `onUpstreamTick()` is
BINDING per note §M2 §F-B-LENSB-01 (symmetric with `onDownstreamTick()`).

**Recording `testing.TB` requirement for AC-016/AC-017 fault-injection tests [v1.8 F-LENSB-B-03; v1.9
R6 BLOCKER fix]:** Both AC-016 and AC-017 assert that `driver.failLoud` FIRED as the PASSING outcome. But
`driver.failLoud` calls `t.Errorf` on the driver's OWN stored `testing.TB` (the one supplied to
`NewLoopback` at construction) — if that stored `testing.TB` were the enclosing REAL `*testing.T` running
the AC-016/AC-017 test itself with no override in front of it, that `t.Errorf` call would mark the
ENCLOSING test FAILED the instant `failLoud` fires. An AC-016/AC-017 test written directly against the
bare real `t`, with no interposed type, would therefore be marked failed by Go's testing framework at the
exact moment it is supposed to observe a pass.

**Fix:** AC-016 and AC-017's fault-injection tests construct their driver (via `NewLoopback`) with a
RECORDING `testing.TB` stub/spy that EMBEDS the real enclosing `*testing.T` and OVERRIDES only `Errorf` —
feasible because these are white-box, in-package tests (`package testenv`), and
`NewLoopback`/`SendKeystroke`/`WaitForEcho`/the driver already accept `testing.TB` rather than a concrete
`*testing.T`/`*testing.B`:

```go
// recordingTB is a minimal testing.TB stub used ONLY by AC-016/AC-017's
// fault-injection tests, so failLoud's t.Errorf is CAPTURED and asserted
// instead of failing the enclosing real *testing.T. It EMBEDS the real
// enclosing t (constructed as &recordingTB{TB: t} — never the zero value)
// so that Helper/Cleanup/Fatalf, which NewLoopback/newEnv call
// unconditionally (testenv.go:384 b.Helper(), :461 t.Helper(), :475
// t.Cleanup(func(){...}), :528 t.Cleanup(e.Close) for the ticker/env
// teardown AC-011/Q6 depend on), promote through to a live TB instead of
// nil-panicking. Only Errorf is overridden, to capture rather than fail.
type recordingTB struct {
    testing.TB          // MUST be the real enclosing t (&recordingTB{TB: t}),
                         // never left nil — Helper/Cleanup/Fatalf promote to
                         // this embedded value and are exercised by every
                         // NewLoopback call (testenv.go:384/461/475/528); a
                         // nil embed nil-panics at construction, before
                         // AC-016/AC-017's fault-injection procedure — or
                         // even NewLoopback itself — completes.
    mu          sync.Mutex
    errorfCalls []string
}

func (r *recordingTB) Errorf(format string, args ...any) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.errorfCalls = append(r.errorfCalls, fmt.Sprintf(format, args...))
}

// AC-016/AC-017 construct the driver against the stub, embedding the real t:
stub := &recordingTB{TB: t}
lb := testenv.NewLoopback(ctx, stub, testenv.LoopbackConfig{ /* ... */ })
// ... exercise the fault-injection procedure (steps above) ...
if len(stub.errorfCalls) != 1 {
    t.Errorf("expected exactly one failLoud call, got %d: %v", len(stub.errorfCalls), stub.errorfCalls)
}
```

**Why embedding the real `t` is required, not merely permitted [v1.9 R6 BLOCKER fix]:** `Errorf` is
OVERRIDDEN on `*recordingTB` — Go method dispatch resolves `stub.Errorf(...)` (and therefore `failLoud`'s
call, which only ever sees `stub` through the `testing.TB` interface) to `recordingTB.Errorf`, which
appends into `errorfCalls` and returns; this call never falls through to the embedded field's `Errorf`
regardless of whether that embedded field is real or nil — so the enclosing test is never marked failed by
`failLoud`, and withholding the real `t` buys nothing on that front. What DOES require the real `t` to be
embedded is the UNOVERRIDDEN methods: `NewLoopback` → `newEnv` calls `b.Helper()` (`testenv.go:384`),
`t.Helper()` (`:461`), `t.Cleanup(func(){...})` (`:475`), and `t.Cleanup(e.Close)` (`:528`, the ticker/env
teardown AC-011/Q6 depend on) on every construction, unconditionally — none of these are overridden on
`recordingTB`, so they promote straight through to the embedded field. The prior v1.8 shape,
`stub := &recordingTB{}` (no `TB:` set), leaves that embedded field nil; the very first `b.Helper()` call
inside `NewLoopback` panics on a nil-interface method call, before AC-016's or AC-017's fault-injection
procedure — or even construction — completes, so both ACs would crash unconditionally, not just in a
subtle edge case. `&recordingTB{TB: t}` is therefore both correct (the enclosing test cannot be failed by
`failLoud`) and required (construction cannot otherwise complete). **The prior "in place of the real
`*testing.T`" framing, and any reading of it as "the real enclosing `t` must NOT be passed to
`NewLoopback`," is RETRACTED: it inverted the actual constraint.** That retired, broken shape —
`stub := &recordingTB{}` with no real `t` passed in at all — must not be implemented; it is documented
here only as the superseded form.

The `mu sync.Mutex` guard is required because `failLoud` may be invoked from a ticker goroutine (the
general case) even though AC-016/AC-017's own fault-injection procedures call
`onDownstreamTick()`/`onUpstreamTick()` synchronously from the test goroutine — the stub must be safe
regardless of which caller pattern exercises it. The REAL enclosing `*testing.T` (`t`) plays two roles
here: embedded inside `stub` so `Helper`/`Cleanup`/`Fatalf` delegate to a live TB, and used directly, after
the fault-injection procedure completes, to report the assertion against `stub.errorfCalls`. `failLoud`'s
`t.Errorf` call is captured by the `Errorf` override and never reaches the embedded real value in either
role. This recording-stub requirement is scoped to AC-016 and AC-017 only; every other (happy-path) AC
constructs its driver with the real `*testing.T`/`*testing.B` directly (no `recordingTB` wrapper) exactly
as before.

### RoundTrip Token API — Fixing the CollectFrames Accumulation Short-Circuit (Q5)

**Binding (per placement note Q5).**

`Env.CollectFrames` and `Conn`/`Console.CollectFrames` poll an **accumulating**
slice — `Env.WaitForEcho` returns as soon as the slice is non-empty, so a second concurrent or leftover
round trip's frame satisfies a `WaitForEcho` call that isn't waiting for it. This is a distinct bug from
the tick/protocol gap and is fixed independently of it, by sidestepping it entirely rather than patching
`CollectFrames`'s polling loop:

```go
// RoundTrip identifies one SendKeystroke → echo round trip in a loopback
// environment. Returned by LoopbackEnv.SendKeystroke; consumed exactly once
// by LoopbackEnv.WaitForEcho.
//
// [H1 — v1.2] done is chan []byte (was chan frame.OuterHeader).
// frame.OuterHeader carries no payload; the RT-ID rides in the payload bytes
// (encodeRTID/decodeRTID). WaitForEcho must return the delivered payload so
// callers can assert the delivered payload decodes to rt.id (AC-014 load-bearing part). [v1.6 F-A-1]
type RoundTrip struct {
    id   uint64
    done chan []byte // buffered 1; written by the downstream ticker goroutine
                     // on delivery; carries the full echo payload (including
                     // the 8-byte RT-ID suffix) — NOT a frame.OuterHeader
}

// SendKeystroke drives a keystroke through the full loopback protocol stack
// (Q3) and returns a token identifying this specific round trip.
//
// [v1.8 F-LENSB-B-02] SendKeystroke performs NO session-existence validation —
// it unconditionally mints the RoundTrip, encodes the payload, and enqueues
// into upstreamHC regardless of whether sessionID has been provisioned via
// CreateSession. This is deliberate and load-bearing for AC-017; see Q3,
// "SendKeystroke performs no session-existence validation," for the full
// rationale.
func (lb *LoopbackEnv) SendKeystroke(t testing.TB, sessionID SessionID, key string) RoundTrip

// WaitForEcho blocks until the echo tagged with rt arrives, or timeout
// elapses. Returns (payload, true) on delivery; (nil, false) on timeout —
// callers should t.Fatalf on timeout. Unlike Env.WaitForEcho, which returns
// as soon as ANY frame is buffered on the session, this reads only rt's own
// completion channel — a concurrent or stale round trip's frame cannot satisfy it.
//
// [H1 — v1.2] Returns ([]byte, bool). AC-014 callers assert:
//   payload, ok := lb.WaitForEcho(t, rt, timeout)
//   if !ok { t.Fatalf(...) }
//   id, ok2 := decodeRTID(payload); if !ok2 || id != rt.id { t.Errorf(...) }  // [v1.4 B-1] 2-value required
func (lb *LoopbackEnv) WaitForEcho(t testing.TB, rt RoundTrip, timeout time.Duration) (payload []byte, ok bool)
```

No shared growing slice is in this path at all. `Env.CollectFrames`/`Conn`/`Console.CollectFrames` are
unchanged — their accumulation semantics remain correct for the VPs that use them. The correlation ID
rides in the payload bytes (8-byte big-endian suffix, `encodeRTID`/`decodeRTID`, package-private), not in
`frame.OuterHeader` (which is a fixed 44-byte wire layout with no spare field) — this also means
`loopbackSink` doesn't need to know about correlation at all; it just echoes bytes, matching how a real
`KeystrokeSink` (tmux) works.

### Goroutine / Lifecycle Plan (Q6)

**Binding (per placement note Q6).**

Two ticker goroutines (upstream, downstream), registered on the **existing** `Env.wg`/`Env.closeCh` — no
new `WaitGroup` or close channel. `Env` already has `wg sync.WaitGroup`, `closeCh chan struct{}`,
`closeOnce sync.Once`; `Env.Close()` already does `closeOnce.Do(func() { close(closeCh); wg.Wait() })`,
registered via `t.Cleanup(e.Close)` in `newEnv`. `AttachConsole` and `AttachProbe` already start
goroutines this exact way (`wg.Add(1)` before `go func() { defer wg.Done(); select { case <-closeCh:
return; ... } }()`) — the loopback tickers use the identical pattern rather than inventing a second
lifecycle mechanism:

```go
// [v1.5 F-B-1] startLoopbackTicker is TICK-FREE: it does NOT call hc.Tick()
// itself. The caller supplies a tickBody that owns the Tick() call under
// the appropriate per-direction mutex (see §H2). The hc parameter is removed;
// the caller closes over any half-channel reference via the tickBody closure.
func startLoopbackTicker(
    env *Env,
    interval time.Duration,
    tickBody func(),
) {
    env.wg.Add(1)
    go func() {
        defer env.wg.Done()
        ticker := time.NewTicker(interval)
        defer ticker.Stop()
        for {
            select {
            case <-env.closeCh:
                return
            case <-ticker.C:
                tickBody()
            }
        }
    }()
}
```

**Wiring (v1.5 F-B-1):** the upstream ticker's `tickBody` is `d.onUpstreamTick` and the downstream
ticker's `tickBody` is `d.onDownstreamTick`. Each seam method owns its half-channel's `Tick()` call under
the corresponding per-direction mutex (per the §H2 per-direction-mutex binding rule — every
`Enqueue`/`Tick` call site acquires the corresponding mutex), so every `Tick()` is
mutex-guarded regardless of how the goroutine invokes the body:

- `startLoopbackTicker(env, upstreamInterval, d.onUpstreamTick)` — upstream ticker
- `startLoopbackTicker(env, downstreamInterval, d.onDownstreamTick)` — downstream ticker

This is the same lifecycle shape as `cmd/switchboard/access.go`'s `startSweepTicker`/
`startFramesDroppedTicker` — the production idiom for "ticker + WaitGroup + cancellation-channel."
The `hc` parameter is absent because the body closure carries any half-channel reference; the lifecycle
contract is otherwise identical. No new `Close()` method is needed on `LoopbackEnv`; `t.Cleanup(env.Close)`
(already registered by `newEnv`) tears everything down, and `wg.Wait()` blocks until both ticker goroutines
have observed `closeCh` and returned — deterministic, no leaked goroutines, matching the existing
`AttachConsole`/`AttachProbe` guarantee.

`NewLoopback` must validate `cfg.TickIntervalUpstream`/`TickIntervalDownstream` against
`halfchannel.MinTickInterval`/`MaxTickInterval` (5ms–50ms) and `b.Fatalf` on an out-of-bounds value,
matching the existing fail-loud convention (`t.Fatalf` on illegal construction throughout this file, e.g.
`NewWithRouters`). **VP-042's own `downstreamInterval` (50ms) sits exactly at `MaxTickInterval`** — legal,
but the validation site needs a comment noting this, since it's the boundary case (AC-002).

### Synthetic Path Construction (Q7)

**Binding (per placement note Q7).**

Two `paths.RankedPath`s per direction (4 total), each backed by `paths.NewPathTracker(1.0, 0.125)` — no
`OnProbe` calls needed. `paths.NewPathTracker` sets `active: true` at construction, so a fresh tracker is
immediately eligible for `Rank` without any probe history.

```go
func newLoopbackPaths() []paths.RankedPath {
    return []paths.RankedPath{
        {ID: 1, Tracker: paths.NewPathTracker(1.0, 0.125)},
        {ID: 2, Tracker: paths.NewPathTracker(1.0, 0.125)},
    }
}
```

`multipath.NewMultipath` requires `[]paths.RankedPath` at construction, and `multipath.Send` internally
calls `paths.Rank` on every call — so testenv must import `internal/paths` **directly**, a Go-imposed
transitive requirement (referencing an exported type from an indirectly-imported package requires a
direct import), not a scope expansion story-writer is choosing. ARCH-08 v2.13 already includes `paths` at
position 11 for exactly this reason. Two `*multipath.Multipath` instances are constructed — one per
direction (`upstreamMP`, `downstreamMP`) — each combining the pathSet used by whichever side is the
sender for that direction and the `recvDedup` cache used by whichever side is the receiver. Both
instances use `dropCacheCapacity` = `multipath.DefaultDropCacheSize` (the standard sentinel; no custom
sizing needed for the two-path loopback case).

### No New Package (Q8)

**Binding (per placement note Q8).** All of this lands inside `internal/testenv` (existing position 23,
test-only composition root). ARCH-08 §6.4's new-package protocol does not apply — this is an import-set
expansion of an existing package, the same class of change as v2.6/v2.8/v2.11.

---

## Acceptance Criteria

**AC-001 was a pre-implementation gate. It is DISCHARGED — see below — but its binding constraints
carry forward into AC-006 and the Design Constraints Q4 section; do not implement from Q4's original
code blocks without applying the Addendum.**

### AC-001 (DISCHARGED 2026-07-12, verdict REVISED; traces to Q4 / Risk 1)

The `arq.OnAck` call-contract proposed in Q4 (`ackSeq` = the locally-observed frame's own `ChanSeq`,
zero SACK in the no-loss happy path) had no existing production call site to copy — this design is the
first proposed caller of `OnAck` in the codebase. Per Risk 1 option (a), an architect placement-note
addendum reviewed the contract before this story could be scheduled: **"Q4 Addendum — AC-001 Sign-off
(2026-07-12)"** (introduced note v1.1, carried in v1.3) in `S-BL.LOOPBACK-FULLSTACK-placement-note.md`,
discharging this gate.

**Verdict: REVISED.** The `ackSeq`/SACK value convention is CONFIRMED correct as proposed. The
`driver.arqServer`/`driver.arqClient` **two-instance topology** Q4's original code blocks show is a
structural defect: `OnAck`'s payload recovery (`payloadFor` / instance-local inFlight lookup) reads only
the calling instance's own `inFlight`/`reorderBuf` maps, populated exclusively by that SAME instance's
prior `EnqueueSend` calls (`EnqueueSend` / instance-local inFlight write). A separate `arqClient` that
never receives `EnqueueSend` returns
`(nil, nil)` from `OnAck` on every call, silently — no error, `nextExpected` still advances — so every
`WaitForEcho` in the harness would time out on every round trip. This is not a subtle edge case; it is
a hard, silent benchmark failure the note's original Risk 1 framing ("getting this wrong doesn't break
VP-042's measured number") did not anticipate for this specific failure mode.

**Binding on the implementer, carried forward from the Q4 Addendum:**

1. Use **one shared `*arq.ARQ` instance** for the downstream direction — a single field on the driver
   (e.g. `driver.downstreamARQ`), not a `arqServer`/`arqClient` pair.
2. `EnqueueSend` and `OnAck` for a given `ChanSeq` MUST be called on that **same instance**, **in that
   order**, **within the same downstream-ticker goroutine tick**.
3. Do not reuse the always-zero-SACK convention outside this harness's Non-Goals envelope (no
   loss/reordering) — a future loss-injection story reusing `OnAck` must compute a real bitmap.
4. A regression guard against reintroducing the two-instance shape is required — see **AC-014**.

**Test:** none for the gate itself — this was a process gate, not a code test, and it is now discharged.
The behavioral consequence of getting the topology wrong is covered by AC-006 (per-call wiring) and
AC-014 (end-to-end round-trip-completes regression guard). `dev-story` MUST implement Q4's downstream
flow per the single-shared-instance shape (Design Constraints, Q4 section, as amended) — not from the
original two-instance code blocks in isolation.

### AC-002 (traces to BC-2.01.001; Q6)

**(a) Interval validation:** `NewLoopback` validates `cfg.TickIntervalUpstream` and
`cfg.TickIntervalDownstream` against `halfchannel.MinTickInterval`/`MaxTickInterval` and `b.Fatalf`s on
an out-of-bounds value. The validation site carries a comment noting that VP-042's own
`downstreamInterval` (50ms) sits exactly at `MaxTickInterval` — legal, boundary case.

**(b) Tick independence:** Both ticker goroutines fire `HalfChannel.Tick()` on their configured schedule
independent of `Enqueue` timing — a keystroke enqueued between ticks waits for the next tick and never
triggers an out-of-band delivery. (BC-2.01.001 compliance.)

**Test (a):** `TestNewLoopback_RejectsOutOfBoundsTickInterval` (table-driven: below `MinTickInterval`,
above `MaxTickInterval`, exactly at `MaxTickInterval` = legal).
**Test (b):** `TestLoopbackDriver_TicksFireOnSchedule` (enqueue between ticks, assert delivery does not
precede the next tick boundary).

### AC-003 (traces to BC-2.01.002; Non-Goals)

The upstream and downstream tickers call `Tick()` every interval regardless of whether data is enqueued
(empty ticks are produced, satisfying BC-2.01.002), but an empty-tick `ChannelFrame` (`FrameType !=
frame.FrameTypeData`) is never passed to `multipath.Send` — only data frames are wire-dispatched. This
is a harness-scope boundary (Non-Goals), not a production behavior change.

**Test:** `TestLoopbackDriver_EmptyTicksNotDispatched` — assert `Tick()` is called on every interval
(instrument via a tick-count hook) while `multipath.Send` call count only increments on data-bearing
ticks.

### AC-004 (traces to BC-2.02.001; Q3, Q7)

`upstreamMP`/`downstreamMP` are each constructed via `multipath.NewMultipath` with the two synthetic
`paths.RankedPath`s from `newLoopbackPaths()`. A single `Enqueue`d payload, once ticked, is dispatched by
`multipath.Send` to both paths (duplicate-and-race); `deliverUpstream`/`deliverDownstream` is invoked
once per selected path.

**Test:** `TestLoopbackDriver_DuplicateAndRaceDispatch` — instrument `deliverUpstream` (or
`deliverDownstream`) with a call-count hook, assert it fires exactly twice per ticked data frame (once
per synthetic path).

### AC-005 (traces to BC-2.02.002; Q3, Q4)

The second-arriving copy of a duplicate-and-raced frame is discarded by `multipath.Receive`'s endpoint
checksum dedup (`ErrDuplicate`) at each direction's `deliverUpstream`/`deliverDownstream` call (AC-004)
— but the two directions differ in what the dedup gates, because of where each direction's
forward-progress call sits relative to the per-path callback:

- **Upstream:** `accessNode.SendKeystroke` is called INSIDE `deliverUpstream`, after its own
  `upstreamMP.Receive` dedup check — so the dedup directly gates `SendKeystroke`: of the two
  `deliverUpstream` calls per AC-004, exactly one (the first-arriving path) reaches `SendKeystroke`;
  the duplicate's `deliverUpstream` call returns via `ErrDuplicate` before `SendKeystroke` is ever
  called.
- **Downstream:** `downstreamARQ.OnAck` is called once per downstream TICK, in the tick body, AFTER
  `downstreamMP.Send` (and both its internal `deliverDownstream` calls) return — not from inside
  `deliverDownstream`. The dedup inside `deliverDownstream`'s `downstreamMP.Receive` call therefore
  does NOT gate `OnAck`: `OnAck` fires exactly once per data-bearing tick regardless of how the two
  `deliverDownstream` calls resolved. What the downstream dedup actually protects is `deliverDownstream`
  itself staying a no-op on the duplicate path (`Receive` returns `ErrDuplicate` and `deliverDownstream`
  does nothing else) — i.e. of the two `deliverDownstream` calls per AC-004, exactly one `Receive` call
  returns `nil` (first arrival, dedup state updated) and the other returns `ErrDuplicate` (discarded);
  neither outcome is wired to call `OnAck` directly.

**Test:** `TestLoopbackDriver_EndpointDedupDiscardsSecondArrival` — upstream: assert
`accessNode.SendKeystroke` is called exactly once per ticked data frame despite two `deliverUpstream`
invocations. Downstream: assert `downstreamMP.Receive` (or an instrumented `deliverDownstream` hook)
is called exactly twice per ticked data frame (once per path) with exactly one `nil` return and one
`ErrDuplicate` return; separately assert `downstreamARQ.OnAck` is called exactly once per ticked data
frame (its cadence is tick-driven, not gated by this dedup — see AC-006).

### AC-006 (traces to BC-2.02.005; Q4 as REVISED by the Q4 Addendum — AC-001 DISCHARGED)

Every downstream tick that produces a data frame is passed to `driver.downstreamARQ.EnqueueSend(f.ChanSeq,
f.Payload, time.Now())` before dispatch (`downstreamMP.Send`). After `Send` returns — `Send`'s two
per-path `deliverDownstream` calls (dedup via `downstreamMP.Receive`) having already completed
synchronously within the same tick/goroutine — the SAME tick body calls the SAME `driver.downstreamARQ`
instance's `OnAck` exactly once, using the tick-body-local `chanSeq` captured before dispatch, with an
all-zero SACK bitmap, per the AC-001-discharged call contract. `OnAck`'s cadence is ONE PER DOWNSTREAM
DATA TICK — it is not invoked from inside `deliverDownstream` and is not conditioned on which of the two
`deliverDownstream` calls' `Receive` result won the dedup. `EnqueueSend` and `OnAck` MUST be the same
`*arq.ARQ` value (a separate `arqServer`/`arqClient` split is a structural defect per the Q4 Addendum:
`OnAck` would return zero delivered payloads on every call, and every `WaitForEcho` would silently time
out). `GapsToRetransmit`/`TLPKTDROP` are not called on any schedule (Non-Goals).

**Test:** `TestLoopbackDriver_DownstreamARQWiring` — assert `EnqueueSend` is called once per downstream
data tick and `OnAck` is called once per downstream data tick (in the tick body, after `Send` returns),
with the frame's own `ChanSeq`, on the same `*arq.ARQ` instance; assert the call is NOT made from within
`deliverDownstream` (e.g. by instrumenting `OnAck` and `deliverDownstream` separately and asserting
`OnAck`'s single call happens after both `deliverDownstream` calls for that tick have returned). Assert
`GapsToRetransmit`/`TLPKTDROP` are never invoked in this harness. See AC-014 for the mandatory
end-to-end regression guard against reintroducing the two-instance shape.

### AC-007 (traces to Q2 — dedicated shard)

`loopbackDriver` constructs its own `Publisher`/`SessionAuth`/`AccessNode` triple at construction time,
with `session.WithKeystrokeSink(loopbackSink)` set from the start. `env.defaultShard` is untouched — the
loopback driver never mutates it, and no `SetSink` method is added to production `session.AccessNode`.

**Test:** `TestLoopbackDriver_DedicatedShard_NoDefaultShardMutation` — assert `env.defaultShard`'s
`KeystrokeSink` remains `session.NoOpSink{}` after a `LoopbackEnv` is constructed and exercised;
`TestSessionAccessNode_NoSetSinkMethod` — a compile-time/reflection guard confirming
`session.AccessNode` gained no new sink-mutation method.

### AC-008 (traces to Q5 — RoundTrip token API)

`LoopbackEnv.SendKeystroke` returns a `RoundTrip` token. `LoopbackEnv.WaitForEcho(t, rt, timeout)`
returns `(payload []byte, ok bool)` — it consumes exactly one token, reading only that token's own
`done chan []byte` completion channel. It never reads `Env.CollectFrames`' accumulating buffer. A
concurrent or stale round trip's frame cannot satisfy a `WaitForEcho` call for a different token.

**Test:** `TestLoopbackEnv_WaitForEcho_DoesNotConsumeOtherRoundTrips` — issue two concurrent
`SendKeystroke` calls, `WaitForEcho` on the second token first, assert it does not return early on the
first token's frame; `TestLoopbackEnv_WaitForEcho_IgnoresStaleCollectFramesBuffer` — pre-populate
`Env.CollectFrames`' buffer with an unrelated frame before issuing a round trip, assert `WaitForEcho`
still waits for its own token.

### AC-009 (traces to Risk 3 — `RoundTrip.done` buffering and no-leak/no-block)

`RoundTrip.done` is `chan []byte`, buffered 1. The downstream ticker's completion path unconditionally
deletes the `driver.pending` entry and sends the echo payload into `done` — it does so whether or not
`WaitForEcho` has been called. On a `WaitForEcho` timeout, the buffered send into `done` does not block
the ticker goroutine even if nobody ever reads from `done` again (buffer capacity 1 absorbs the send).
The `driver.pending` entry is always deleted at delivery, independent of any waiter.

**Test:** `TestLoopbackEnv_WaitForEcho_TimeoutThenLateArrival_NoLeak` — issue `SendKeystroke`, call
`WaitForEcho` with a timeout shorter than the configured tick cadence so it times out, then allow the
echo to arrive; assert (a) the ticker goroutine's send into `done` does not block/deadlock, (b)
`driver.pending` no longer holds the entry after the late arrival is processed, (c) no goroutine leak is
detected (`t.Cleanup` + goroutine-count check, mirroring the `Env.Close()`/`wg.Wait()` leak-check
convention used elsewhere in this package).

### AC-010 (traces to Risk 2 — `PathTracker.IsActive()` initial-state assertion)

An explicit, cheap assertion/test confirms `paths.NewPathTracker(1.0, 0.125).IsActive()` returns `true`
immediately at construction, with no `OnProbe` call — insurance against a future `internal/paths` change
silently breaking the loopback's path activation and producing a confusing downstream failure (e.g.
`multipath.Send` silently excluding a path from `Rank`) instead of a clear, localized one.

**Test:** `TestNewLoopbackPaths_TrackersActiveWithoutProbe` — construct `newLoopbackPaths()`, assert
`IsActive()` is `true` on every returned `paths.RankedPath.Tracker` with zero `OnProbe` calls made.

### AC-011 / DECISION (traces to Risk 4 — pending-map diagnostic; v1.2 reframe)

**Context (v1.2 reframe per M1):** The downstream ticker's completion path unconditionally deletes each
`driver.pending` entry at delivery, independent of whether `WaitForEcho` was called (AC-009). Therefore
the original premise — "pending accumulates if `WaitForEcho` isn't called" — is false in the no-loss
harness: every delivered round-trip entry is drained whether or not a waiter is present. AC-011 is
reframed as a **non-blocking diagnostic**, not a leak-detector.

**Decision:** `LoopbackEnv` construction registers a `t.Cleanup` that, at teardown, logs/reports any
entries still in `driver.pending`. In the no-loss harness, the map should always be empty at teardown
(all deliveries drain it). A non-empty map at teardown indicates either a `WaitForEcho` that returned
`(nil, false)` on timeout (entry was already drained by delivery before the waiter checked it, but that
path is AC-009) or a logic error in the RT-ID decode path (entry not drained because `decodeRTID`
failed). To make this deterministic rather than timing-dependent, the test verifies the diagnostic
against an **injected decode-mismatch scenario**: a round trip is issued, a synthetic entry with a
mismatched ID is injected into `driver.pending` directly (bypassing normal send), and the teardown
assertion confirms it is reported. This makes the diagnostic a testable contract.

**Test:** `TestLoopbackEnv_Cleanup_DiagnosticOnPendingLeak` — inject a synthetic stale entry into
`driver.pending` before teardown; assert the `t.Cleanup`-registered diagnostic fires and logs it. Does
NOT use `t.Fatalf` (diagnostic only — does not abort the test). Companion
`TestLoopbackEnv_Cleanup_SilentWhenDrained` — normal round trip + `WaitForEcho` leaves the map empty
at teardown with no diagnostic firing.

### AC-012 (traces to Q6 — goroutine lifecycle)

Both ticker goroutines (upstream, downstream) register on the existing `Env.wg`/`Env.closeCh` — no new
`WaitGroup` or close channel is introduced. `t.Cleanup(env.Close)` (already registered by `newEnv`) tears
both goroutines down deterministically; `wg.Wait()` blocks until both have observed `closeCh` and
returned. No `Close()` method is added to `LoopbackEnv`.

**Test:** `TestLoopbackEnv_TickerGoroutines_JoinOnClose` — construct a `LoopbackEnv`, call
`lb.Env.Close()` (or trigger the registered cleanup), assert both ticker goroutines have exited via a
`sync.WaitGroup`-based join-confirmation, with a bounded timeout guarding against a hang (matching the
existing `AttachConsole`/`AttachProbe` leak-check pattern in this package).

### AC-013 (traces to Package Impact — merged bench cross-reference)

`internal/bench/keystroke_echo_testenv_bench_test.go` (merged on `develop`, PR #121 @ `4c276d9`,
`//go:build integration`-tagged) is updated from its current two-call `env.SendKeystroke`/
`env.WaitForEcho` shape (the VP-042.md skeleton shape, now superseded — see Context) to the token-based
shape: `rt := lb.SendKeystroke(b, sessionID, "x"); payload, ok := lb.WaitForEcho(b, rt,
500*time.Millisecond)`. The call matches the shipped `NewLoopback` signature order (ctx, b,
LoopbackConfig) and the H1 two-value `WaitForEcho` return. The package comment's "lower bound only"
framing (inherited from S-BL.BENCH's honest-partial-evidence disclosure) is retired once this full stack
lands, since the divergence it disclosed (bypassing arq/multipath/tick-scheduling) no longer exists.

**Test:** no new test — this AC is a modification of an existing benchmark file. **Verification Method
(binding per placement note v1.15):** AC-013 verifies via the direct invocation the merged file's own doc
comment already prescribes — not `go build` (tagged or untagged; both are vacuous against this test-only
package), and not `just bench`. `internal/bench/` contains only two files (`keystroke_echo_bench_test.go`
and `keystroke_echo_testenv_bench_test.go`), both `_test.go` — `go build` never compiles `_test.go` files,
WITH OR WITHOUT `-tags integration` (the tag is irrelevant to the exclusion; the file is excluded because
it is a test file, not because a build tag is absent), so `go build -tags integration ./internal/bench/...`
verifies nothing about `BenchmarkKeystrokeToEcho_P99` regardless of the tag. The compile check for a
test-only package must compile the TEST BINARY instead:

```sh
go test -tags integration -run '^$' -count=1 ./internal/bench/
```

(`go vet -tags integration ./internal/bench/...` is a sound, faster alternative — it type-checks the
tagged test files without linking, and may be used in its place). Then run:

```sh
go test -tags integration -run '^$' -bench=BenchmarkKeystrokeToEcho_P99 \
    -benchtime=1x -count=1 ./internal/bench/
```

producing the `p99_rtt_ms` metric via `b.ReportMetric`. **`just bench` is explicitly OUT of AC-013's
scope** (`justfile:53-66`) — it runs the separate, untagged `BenchmarkKeystrokeEcho_P99` (no "To") defined
in `keystroke_echo_bench_test.go`, the S-BL.BENCH tag-free lower-bound recipe; it structurally cannot run
`BenchmarkKeystrokeToEcho_P99` (different build-tag gate, different function name), runs today unaffected
by this story, and verifies nothing this story changes.

### AC-014 (regression guard, added 2026-07-12; traces to AC-001 Q4 Addendum)

A mandatory regression guard against reintroducing the `arqServer`/`arqClient` two-instance shape the
Q4 Addendum ruled out (AC-001, constraint 4). The failure mode is silent — no error, no panic, `OnAck`
just returns `(nil, nil)` forever — so a structural assertion alone is not sufficient; the guard MUST
include a behavioral assertion that a full round trip actually completes.

**Mandatory:** a test drives a complete `SendKeystroke` → `WaitForEcho` round trip through a real
`LoopbackEnv` (not a mock/stub of `downstreamARQ`) and asserts the round trip **completes** — i.e., the
delivered frame/payload is non-empty and `WaitForEcho` returns before its timeout, not merely that it
returns. A test that only checks `WaitForEcho` returns (without inspecting what it returned) would not
catch the two-instance failure mode, since a hang manifests as a *timeout* while a subtler variant could
return a zero-value frame without an explicit assertion catching it — the non-empty-delivery assertion
is the load-bearing part of this AC.

**Acceptable supplementary coverage:** the placement note's Addendum also proposes a structural
assertion — that the downstream driver has exactly one `*arq.ARQ`-typed field (e.g. via reflection over
`loopbackDriver`'s field set, or a compile-time check that only one field of that type exists). This is
acceptable as ADDITIONAL coverage but does not substitute for the behavioral round-trip-completes
assertion above, which is mandatory.

**Test:** `TestLoopbackEnv_RoundTripCompletes_SingleSharedARQInstance` — construct a `LoopbackEnv`,
`SendKeystroke`, `WaitForEcho` with a generous timeout, assert (a) `ok == true` (no timeout), (b)
`id, ok2 := decodeRTID(payload); ok2 && id == rt.id` — [v1.4 B-1] 2-value call required; the returned
`[]byte` payload must decode successfully and match the expected RT-ID, proving delivery actually happened
on the correct round trip (not merely that the call returned).
Optionally paired with `TestLoopbackDriver_SingleARQInstanceField` (structural — reflects over the
driver's fields, asserts exactly one `*arq.ARQ`-typed field) as supplementary, non-substituting
coverage.

### AC-015 (traces to BC-2.01.001 / H2 — halfchannel synchronization; added v1.2)

The loopback driver runs clean under `go test -race` / `just test-race`. No data race is detected on
either `upstreamHC` or `downstreamHC` accesses. The per-direction mutexes (`upstreamHCMu`,
`downstreamHCMu` — see Design Constraints Q2 §H2) ensure that `Enqueue` and `Tick` on each
`HalfChannel` are never called concurrently from different goroutines without synchronisation, satisfying
the halfchannel concurrency contract.

**Test:** `TestLoopbackDriver_NoRaceUnderConcurrentSendEcho` — run a loopback benchmark with concurrent
`SendKeystroke`/`WaitForEcho` pairs under `-race`; assert no `DATA RACE` annotation appears (this test
is expected to appear in the CI `just test-race` output with a PASS result). Verification: the CI
pipeline's `just test-race` step is the authoritative gate; this test is the per-story hook into that
gate for this specific race class.

### AC-016 (traces to BC-2.02.005 / §M2 note — OnAck error loud; added v1.3; sound method specified v1.4 B-3)

If `driver.downstreamARQ.OnAck` returns an error, the harness MUST surface it as a loud failure — not
silently convert it to a `WaitForEcho` timeout. This is the note §M2's explicit obligation ("the
story-writer should add an AC for: if OnAck returns an error, the harness surfaces it as a loud failure
(not a silent timeout)"). The mechanism is `driver.failLoud` (see §M2 Design Constraints): `t.Errorf`
from the ticker goroutine. [v1.5 F-B-4] The `driver.errCh chan error` alternative is dropped — see §M2
for rationale (deadlock with symmetric AC-017 upstream path). A silent timeout, while observable, masks a
harness construction bug as high latency — the loud failure is the load-bearing diagnostic contract.

**Fault-injection test (sound method — v1.4 B-3):** `TestLoopbackDriver_OnAckError_SurfacesLoud` —
uses the `onDownstreamTick()` package-private seam (the directly-callable method on `loopbackDriver`
containing the downstream tick body — see note §M2 §B-3 required seam; BINDING on the implementer):

1. Build the `loopbackDriver` via `NewLoopback` against `&recordingTB{TB: t}` — see "Recording
   `testing.TB` requirement for AC-016/AC-017 fault-injection tests [v1.8 F-LENSB-B-03; v1.9 R6
   BLOCKER fix]" in Design Constraints above: the stub EMBEDS the real enclosing `t` (required so
   `NewLoopback`'s `Helper`/`Cleanup` calls don't nil-panic) and OVERRIDES only `Errorf`, so
   `failLoud`'s `t.Errorf` is captured into `stub.errorfCalls` rather than marking this passing test
   FAILED — but do **not** start the downstream ticker goroutine.
2. Call `driver.onDownstreamTick()` 65 times synchronously (each empty tick — no payload enqueued —
   increments `downstreamHC.seq` without calling `EnqueueSend`, so `downstreamARQ.nextExpected` stays 0).
3. Enqueue one data payload into `downstreamHC` (via `downstreamHC.Enqueue`).
4. Call `driver.onDownstreamTick()` one more time synchronously. This fires
   `downstreamARQ.EnqueueSend(chanSeq)` then `downstreamARQ.OnAck(chanSeq, zeroSACK)` with
   `chanSeq > 64` and `nextExpected = 0` — `OnAck` returns `ErrAckOutOfWindow`.
5. Assert, against the real enclosing `*testing.T`, that the stub recorded exactly one `Errorf` call
   (`stub.errorfCalls`) — i.e. `driver.failLoud` was called (loud, not a silent `WaitForEcho` timeout).

This is single-goroutine throughout — no race, no `sync.Once` interaction, no ticker goroutine competing
for `downstreamARQ`. Both previously-considered approaches are UNSOUND and must NOT be used:
- Starting the ticker at `NewLoopback` time contradicts B-F3 (races with AC-008).
- A test-goroutine `OnAck` call races the ticker goroutine's own `OnAck` on `downstreamARQ`, violating
  the `arq.ARQ` single-writer contract and failing AC-015.

**Implements:** Task 6 (downstream flow + `driver.failLoud` / `onDownstreamTick()` seam).

### AC-017 (traces to BC-2.01.001 — upstream-error loud failure; added v1.4 B-4)

`deliverUpstream` MUST check the error returned by `driver.accessNode.SendKeystroke` IN-PLACE, at the call
site, and surface any failure via `driver.failLoud` — symmetric with AC-016's requirement on the
downstream path. `accessNode.SendKeystroke` can return `ErrConsoleNotFound`, `ErrSessionMismatch`, or an
authorizer error (as confirmed by PAT-04 source verification); `deliverUpstream` still returns that error
to its caller afterward. Without explicit error checking, an upstream delivery failure is swallowed and
surfaces only as a `WaitForEcho` timeout — a silent failure violating SOUL.md §4.

**Required behavior [v1.14 F-R18-LENSB-01]:** the check MUST be pinned IN-PLACE at the `SendKeystroke`
call site inside `deliverUpstream` — call `driver.failLoud(err)` immediately when
`driver.accessNode.SendKeystroke` returns non-nil, THEN still return the error from `deliverUpstream`
(the check is additive to the return, not a replacement for it) — symmetric with §M2's already-correct
in-place downstream `OnAck` check. A check placed on `upstreamMP.Send`'s return value in the upstream
tick body (`onUpstreamTick()`) is UNSOUND and MUST NOT be used: `multipath.Send` (`multipath.go:244-286`)
returns `nil` to its caller whenever at least one selected path's `SendResult.Sent` is true; in the
upstream duplicate-and-race dispatch (two paths, Q7) the dedup sibling's `Receive` call returns
`ErrDuplicate` and its `deliverUpstream` call returns `nil` WITHOUT ever calling `SendKeystroke`, so
`sent` is always at least 1 from the dedup path alone and `multipath.Send` always returns `nil` to
`onUpstreamTick()` regardless of whether the delivering path's `SendKeystroke` call failed — a tick-body
check on `Send`'s return value can NEVER observe the masked per-path error. Because only the delivering
(first-arriving) path ever reaches `SendKeystroke` — the dedup sibling returns before `SendKeystroke` is
called — the in-place check fires `failLoud` exactly once per failing tick, consistent with this AC's
"exactly one `Errorf` call" assertion below.

**Fault-injection test (direct-seam method — [v1.6 F-B-LENSB-01; v1.10 F-LENSB-01 tightened to
true-by-construction]):** `TestLoopbackDriver_UpstreamDeliveryError_SurfacesLoud` — single-goroutine,
ticker-timing-independent, symmetric with the AC-016/`onDownstreamTick()` pattern:

1. Build the `loopbackDriver` via `NewLoopback` against `&recordingTB{TB: t}` — see "Recording
   `testing.TB` requirement for AC-016/AC-017 fault-injection tests [v1.8 F-LENSB-B-03; v1.9 R6
   BLOCKER fix]" in Design Constraints above: the stub EMBEDS the real enclosing `t` (required so
   `NewLoopback`'s `Helper`/`Cleanup` calls don't nil-panic) and OVERRIDES only `Errorf`, so
   `failLoud`'s `t.Errorf` is captured into `stub.errorfCalls` rather than marking this passing test
   FAILED. [v1.10] Do NOT start the upstream ticker goroutine — this step is now automatically
   satisfied rather than a separate discipline to uphold: `CreateSession` is the sole start-site for
   both tickers (see Design Constraints §M2 above), and this test never calls it, so no upstream
   ticker goroutine can exist at any point in this test.
2. Call `SendKeystroke` before `CreateSession` — this enqueues a payload into `upstreamHC` (no
   console registered yet, so `accessNode.SendKeystroke` will return `ErrConsoleNotFound`; per
   [v1.8 F-LENSB-B-02], `SendKeystroke` performs no session-existence validation, so this pre-`CreateSession`
   call succeeds at the mint/encode/enqueue level).
3. Call `driver.onUpstreamTick()` synchronously. This fires the upstream tick body:
   `upstreamHC.Tick()` dequeues the payload, `accessNode.SendKeystroke(...)` is called, returns
   `ErrConsoleNotFound` (no console registered), `failLoud` fires.
4. Assert, against the real enclosing `*testing.T`, that the stub recorded exactly one `Errorf` call
   (`stub.errorfCalls`) — i.e. `driver.failLoud` was called. [v1.10] Because no ticker goroutine of
   either kind ever runs (step 1, now true by construction), this assertion's unlocked read of
   `len(stub.errorfCalls)` — after the synchronous `onUpstreamTick()` call in step 3 has already
   returned — observes no concurrent writer. `go test -race` (AC-015) stays clean on this path by
   construction, closing the latent race the prior construction-start latitude permitted (the outcome
   count was always 1; the hazard was a race annotation on the read, not a wrong assertion value).

This is single-goroutine throughout — no race, no ticker-timing dependency, ticker-start order is
irrelevant to AC-017's correctness, and — per [v1.10] — no ticker of either kind ever launches during
this test at all. The seam binding `onUpstreamTick()` is REQUIRED per note §M2 §F-B-LENSB-01.

**Implements:** Task 5 (upstream flow including `deliverUpstream` error handling), Task 13 (test `TestLoopbackDriver_UpstreamDeliveryError_SurfacesLoud`).

Transcribed from the placement note. This story does NOT implement:

- **Real network I/O or cross-process operation.** Both synthetic paths are zero-added-latency,
  in-process function calls — no sockets, no serialization to wire bytes (no
  `outerassembler.Assemble`/`DecodeChannelHeader` round trip); `multipath.Frame`/`halfchannel.ChannelFrame`
  are passed as Go structs, not encoded bytes. Byte-level wire-format coverage in a loopback harness would
  be a separate, additive future story.
- **Simulated packet loss, retransmission, or TLPKTDROP.** `GapsToRetransmit` and `TLPKTDROP` are not
  called on any schedule. `internal/arqsend` and `internal/outerassembler`-based real retransmit dispatch
  are not added to testenv's import set — they would only be needed for a loss-injection follow-on.
  `internal/arq`'s own pure-core unit tests already cover the reorder/gap/TLPKTDROP state machine; this
  benchmark's job is realistic tick-driven happy-path latency, not re-proving ARQ correctness.
- **`internal/replay` / upstream idempotent-window fidelity.** Out of scope per Q1 and ARCH-03 — ARQ is
  documented as downstream-only; upstream keystroke reliability is `internal/replay`'s job in production,
  and this benchmark has no simulated loss, so replay's absence changes nothing observable.
- **Empty-tick wire dispatch.** Empty ticks are produced by `Tick()` (BC-2.01.002 compliance) but not
  dispatched over multipath in this harness — they carry no round-trip token and would not change the
  measured property.
- **Changing `Env.SendKeystroke`/`Env.CollectFrames`/`Env.WaitForEcho`.** These remain exactly as they are
  for the 10 other VPs that use them.
- **A VP-042 `verification_lock` flip.** See Forward Obligation below — this story delivers and runs the
  harness once for evidence; locking VP-042 is a separate, subsequent PO/architect act.

---

## Architecture Mapping

| Component | Package | New / Modified | Notes |
|-----------|---------|-----------------|-------|
| `loopbackDriver` (type) | `internal/testenv` | New | Owns dedicated `Publisher`/`SessionAuth`/`AccessNode`, both `Multipath` instances, both `HalfChannel`s, ONE shared `*arq.ARQ` instance for the downstream direction (`downstreamARQ` — AC-001 Addendum), `pending` map, `upstreamHCMu`/`downstreamHCMu sync.Mutex` (H2 — serialize Enqueue+Tick per direction), `loopbackConsoleKey` (H3); exposes `onDownstreamTick()` package-private method as required seam for AC-016 fault injection (v1.4 B-3); exposes `onUpstreamTick()` package-private method as required seam for AC-017 fault injection (v1.6 F-B-LENSB-01 — symmetric with `onDownstreamTick()`) |
| `RoundTrip` (type) | `internal/testenv` | New | Opaque outside the package; carries `id uint64` + `done chan []byte` (buffered 1; H1 fix — was `chan frame.OuterHeader`) |
| `loopbackSink` (type) | `internal/testenv` | New | Implements `session.KeystrokeSink`; echoes payload verbatim into `downstreamHC.Enqueue` |
| `LoopbackEnv.SendKeystroke`/`WaitForEcho`/`CreateSession` | `internal/testenv` | New (methods on `*LoopbackEnv`) | Do not collide with `*Env`'s method set (named field, not embedding); `WaitForEcho` returns `(payload []byte, ok bool)` (H1 fix); `CreateSession` provisions loopback console `RegisterKey`+`Attach` (H3 fix) |
| `startLoopbackTicker(env *Env, interval time.Duration, tickBody func())` (helper) | `internal/testenv` | New | [v1.5 F-B-1] Tick-free: registers on `Env.wg`/`Env.closeCh`; fires `tickBody()` on each tick; no `hc` param. Wired: upstream → `d.onUpstreamTick`, downstream → `d.onDownstreamTick`. Identical lifecycle shape to `AttachConsole`/`AttachProbe`. |
| `newLoopbackPaths` (helper) | `internal/testenv` | New | Two `paths.RankedPath`s per direction; each backed by `paths.NewPathTracker(1.0, 0.125)`; `dropCacheCapacity` = `DefaultDropCacheSize` |
| `toMPFrame(f halfchannel.ChannelFrame) multipath.Frame` (helper) | `internal/testenv` | New | Copies `f.Payload` into `multipath.Frame.Payload`; does NOT carry `f.ChanSeq` — caller captures `chanSeq := f.ChanSeq` BEFORE calling this (B1 fix). No `ChanSeq` field added to `multipath.Frame` |
| `encodeRTID(key string, id uint64) []byte` (helper) | `internal/testenv` | New | Appends 8-byte big-endian `id` suffix to `[]byte(key)`; pure, package-private |
| `decodeRTID(payload []byte) (id uint64, ok bool)` (helper) | `internal/testenv` | New | Reads last 8 bytes of `payload` as big-endian uint64; returns `(0, false)` if `len(payload) < 8`; pure, package-private |
| `var zeroSACK [arq.SACKBitmapBytes]byte` | `internal/testenv` | New | All-zero SACK bitmap for `OnAck` no-loss calls; zero value, never written. Only valid inside this harness's Non-Goals envelope (no loss/reordering) |
| `frameFor` | N/A | REMOVED (v1.2) | Eliminated — was only needed to bridge payload to `chan frame.OuterHeader`; `RoundTrip.done` is now `chan []byte` so payload passes directly (H1 fix) |
| `NewLoopback` | `internal/testenv` | Modified | Wires halfchannel/arq/multipath/paths instead of discarding `LoopbackConfig`; adds Min/MaxTickInterval validation |
| `halfchannel.HalfChannel` | `internal/halfchannel` | Read-only consumer | `New`, `Tick`, `Enqueue` |
| `arq.ARQ` | `internal/arq` | Read-only consumer | `New`, `EnqueueSend`, `OnAck` — first production-adjacent `OnAck` call site; call contract DISCHARGED 2026-07-12, verdict REVISED (AC-001) |
| `multipath.Multipath` | `internal/multipath` | Read-only consumer | `NewMultipath`, `Send`, `Receive` |
| `paths.PathTracker`/`RankedPath` | `internal/paths` | Read-only consumer | `NewPathTracker`, `RankedPath` |
| `keystroke_echo_testenv_bench_test.go` | `internal/bench` | Modified | Token-based two-call shape (AC-013) |

## Edge Cases

| Edge Case | Handling |
|-----------|----------|
| `WaitForEcho` times out, echo arrives later | `RoundTrip.done` buffered 1; downstream ticker's send never blocks even if nobody reads it; `driver.pending` entry is still deleted (AC-009) |
| `WaitForEcho` never called for a `RoundTrip` (test bug) | The downstream ticker's completion path unconditionally drains `driver.pending` at delivery (AC-009) — there is no leak in the no-loss harness. An entry lingers only if the echo never arrives (e.g. `decodeRTID` mismatch); the non-blocking `t.Cleanup` diagnostic (AC-011) reports any such undrained entry without aborting the test |
| Duplicate frame arrival (same payload, two synthetic paths) | `multipath.Receive` returns `ErrDuplicate` on the second arrival inside `deliverUpstream`/`deliverDownstream` — upstream this discards before `accessNode.SendKeystroke` is ever called; downstream it discards inside `deliverDownstream` only — `downstreamARQ.OnAck` is unaffected, since it is called once per tick from the tick body regardless (AC-005, AC-006) |
| Tick interval exactly at `MaxTickInterval` (50ms) | Legal — VP-042's own `downstreamInterval` sits exactly here; validation site carries a boundary comment (AC-002) |
| Fresh `paths.RankedPath` with no probe history | `NewPathTracker` defaults `active: true`; `Rank()` considers it eligible with zero `OnProbe` calls (AC-010) |
| `OnAck` window-validation / `ErrAckOutOfWindow` path | Normally unreachable in the no-loss happy path. Reachable if downstream ticker starts too early (before first `EnqueueSend`) and idles > 64 ticks — mitigated by lazy ticker start at `CreateSession` (M2). If `OnAck` returns this error it MUST be surfaced loud via `driver.failLoud`, not swallowed (M2, AC-006) |
| `decodeRTID` failure (payload < 8 bytes or malformed) | Returns `(0, false)`; `driver.pending[0]` lookup returns nil channel; no send — round trip times out. The teardown diagnostic (AC-011) reports the pending entry as undrained |
| Two concurrent `SendKeystroke`/`WaitForEcho` round trips | Each has its own `RoundTrip.id` and `done` channel; AC-008 guarantees no cross-talk |
| `EnqueueSend`/`OnAck` called on separate `*arq.ARQ` instances (`arqServer`/`arqClient` split) | RULED OUT by AC-001 (Q4 Addendum, 2026-07-12) — `OnAck` on a never-`EnqueueSend`'d instance returns `(nil, nil)` on every call, silently; every round trip would time out. One shared `downstreamARQ` instance is required; AC-014 is the regression guard |

## Purity Classification

| Component | Classification | Rationale |
|-----------|-----------------|-----------|
| `loopbackDriver`, ticker goroutines, `RoundTrip` | Effectful (test infrastructure) | Goroutines, tickers, channel synchronization — same class as existing `AttachConsole`/`AttachProbe` |
| `halfchannel`, `arq`, `multipath`, `paths` (as consumed) | Pure-core, UNCHANGED | testenv becomes an effectful DRIVER of their `Tick()`/`OnAck()`/`Send()` entry points; their own purity boundary is unchanged by this edge (ARCH-08 v2.13 rationale) |

## Package Impact Summary

(Transcribed from the placement note.)

| Package | Change | ARCH-08 §6.4 required? |
|---------|--------|------------------------|
| `internal/testenv` | New `loopbackDriver` type; `LoopbackEnv.SendKeystroke`/`WaitForEcho`/`CreateSession`/`RoundTrip`; `NewLoopback` wires halfchannel/arq/multipath/paths instead of discarding `LoopbackConfig` | No (existing package) — import-set expansion requires the §6.4-equivalent pre-code registration already done in ARCH-08 v2.13 |
| `internal/halfchannel` | None — read-only consumer (`New`, `Tick`, `Enqueue`) | No |
| `internal/arq` | None — read-only consumer (`New`, `EnqueueSend`, `OnAck`); first production-adjacent call site for `OnAck` (AC-001) | No |
| `internal/multipath` | None — read-only consumer (`NewMultipath`, `Send`, `Receive`) | No |
| `internal/paths` | None — read-only consumer (`NewPathTracker`, `RankedPath`) | No |
| `internal/bench` | `keystroke_echo_testenv_bench_test.go` (merged on `develop`, PR #121 @ `4c276d9`, `//go:build integration`-tagged) updated to the token-based two-call shape; "lower bound only" framing retired (AC-013) | No |

**No new `internal/` package.** ARCH-08 registration is the import-set amendment already applied
(v2.13, DRAFT/PROSPECTIVE) — it becomes final at this story's merge per the same machine-verification
protocol used for every prior testenv import-set change (v2.5, v2.8, v2.11).

---

## Token Budget Estimate (forecast)

| Component | Est. tokens |
|-----------|-------------|
| This story spec | ~9k |
| Placement note (binding input, full read required) | ~6k |
| Referenced production code (`testenv.go`, `halfchannel.go`, `arq.go`, `multipath.go`, `paths.go` — read-only consumer surfaces) | ~7k |
| Test infrastructure context (existing `testenv` patterns, merged bench test) | ~3k |
| **Total implementing-agent context** | **~25k — well within 20–30% of a 200k context window. No story split required.** |

## Tasks (MANDATORY)

1. [x] **GATE (DISCHARGED 2026-07-12):** AC-001 resolved via architect placement-note addendum ("Q4
   Addendum — AC-001 Sign-off," introduced note v1.1, carried in v1.3) — verdict REVISED. Read the Addendum before Task 6: it supersedes
   Q4's original `arqServer`/`arqClient` code blocks with a single shared `*arq.ARQ` instance
   (`driver.downstreamARQ`).
2. [ ] Implement `loopbackDriver` inside `internal/testenv` with its own `Publisher`/`SessionAuth`/
   `AccessNode` triple constructed via `session.WithKeystrokeSink(loopbackSink)` (Q2, AC-007). Include
   `upstreamHCMu`/`downstreamHCMu sync.Mutex` fields for per-direction halfchannel serialization (H2,
   AC-015).
3. [ ] Implement `RoundTrip` + `driver.pending map[uint64]chan []byte` (buffered-1 channels; H1) +
   `rtSeq atomic.Uint64` (Q5, AC-008, AC-009).
4. [ ] Implement `LoopbackEnv.SendKeystroke`/`WaitForEcho`/`CreateSession` on `*LoopbackEnv` (Q2, Q5).
   `WaitForEcho` returns `(payload []byte, ok bool)` (H1). `CreateSession` provisions the loopback
   console via `pub.Publish(sessionName)` → `RegisterKey`+`Attach` (H3/B-F1, AC-007). Downstream
   ticker starts at `CreateSession` time, PREFERRED per B-F3 (single-threaded, race-free); if
   first-`SendKeystroke` start is used instead, `sync.Once` is MANDATORY (M2 — not at `NewLoopback` construction).
5. [ ] Implement upstream flow: `Enqueue` (under `upstreamHCMu`) → upstream ticker `Tick()` (under
   `upstreamHCMu`) → `upstreamMP.Send` → `deliverUpstream` → `upstreamMP.Receive` dedup →
   `accessNode.SendKeystroke` → `loopbackSink.SendInput` (Q3, AC-004, AC-005, H2). Check the
   `SendKeystroke` error IN-PLACE at its call site inside `deliverUpstream` and call `driver.failLoud(err)`
   if non-nil, then still return the error from `deliverUpstream` — do NOT check `upstreamMP.Send`'s
   return value instead; it cannot observe the masked per-path error (AC-017) [v1.14 F-R18-LENSB-01].
6. [ ] Implement downstream flow: `loopbackSink.SendInput` → `downstreamHC.Enqueue` (under
   `downstreamHCMu`) → downstream ticker `Tick()` (under `downstreamHCMu`) → capture `chanSeq :=
   f.ChanSeq` (B1) → `driver.downstreamARQ.EnqueueSend(chanSeq, ...)` → `downstreamMP.Send(toMPFrame(f),
   deliverDownstream)` (internally invokes `deliverDownstream` → `downstreamMP.Receive` dedup once per
   selected path, synchronously, before `Send` returns) → THEN, still in the same tick body, using the
   SAME tick-body-local `chanSeq`, the SAME `driver.downstreamARQ.OnAck(chanSeq, zeroSACK)` (called
   once per tick, NOT from inside `deliverDownstream`, NOT gated by its dedup outcome) → check err
   loudly via `driver.failLoud` (M2, AC-016) → `driver.pending` lookup → `ch <- payload` (H1,
   `chan []byte`). Expose `onDownstreamTick()` as a package-private method containing the entire
   downstream tick body — including the `OnAck` call and delivered-payload loop — as one function
   (required seam for AC-016 fault-injection test — note §M2 §B-3).
   (Q4 as amended by the Q4 Addendum and v1.2/v1.4 corrections, AC-006) — **one shared `*arq.ARQ`
   field only; do not split into `arqServer`/`arqClient`** (AC-001).
7. [ ] Implement `NewLoopback` config validation against `halfchannel.MinTickInterval`/
   `MaxTickInterval`, `b.Fatalf` on violation, with the 50ms-boundary comment (Q6, AC-002).
8. [ ] Register both ticker goroutines on the existing `Env.wg`/`Env.closeCh` via `startLoopbackTicker`
   (Q6, AC-012) using the tick-free signature `(env, interval, tickBody func())` [v1.5 F-B-1]: wire
   upstream as `startLoopbackTicker(env, upstreamInterval, d.onUpstreamTick)` and downstream as
   `startLoopbackTicker(env, downstreamInterval, d.onDownstreamTick)` — no new `WaitGroup`/close channel.
9. [ ] Implement synthetic path construction — two `paths.RankedPath`s per direction backed by
   `paths.NewPathTracker(1.0, 0.125)`, plus the `PathTracker.IsActive()` initial-state assertion (Q7,
   AC-010).
10. [ ] Wire the `driver.pending`-empty `t.Cleanup` safeguard (AC-011); update the merged
    `keystroke_echo_testenv_bench_test.go` (`develop`, PR #121 @ `4c276d9`) to the token-based
    shape (AC-013).
11. [ ] Implement the regression guard against reintroducing the two-instance `arqServer`/`arqClient`
    shape (AC-014): a behavioral test that a full `SendKeystroke`/`WaitForEcho` round trip actually
    completes — assert `ok == true` AND `id, ok2 := decodeRTID(payload); ok2 && id == rt.id`
    [v1.5 F-1] 2-value form required (v1.4 changelog claimed this was fixed here but was not; corrected
    now). A structural exactly-one-`*arq.ARQ`-field assertion may be added as supplementary coverage
    but does not substitute for the behavioral assertion.
12. [ ] Confirm `go test -race` / `just test-race` passes for the loopback driver (AC-015 — the
    per-direction mutexes added in Task 2 are the mechanism; this task verifies them under the race
    detector).
13. [ ] Implement and test the upstream-error loud-failure path (AC-017): check
    `accessNode.SendKeystroke`'s returned error IN-PLACE at its call site inside `deliverUpstream` (NOT
    on `upstreamMP.Send`'s return value in the upstream tick body — that check is unsound, see AC-017)
    and call `driver.failLoud(err)` if non-nil, then still return the error from `deliverUpstream`
    [v1.14 F-R18-LENSB-01]. Expose `onUpstreamTick()` as a package-private method
    containing the upstream tick body (required seam for AC-017 fault-injection test — note §M2
    §F-B-LENSB-01). Write `TestLoopbackDriver_UpstreamDeliveryError_SurfacesLoud` using the
    direct-seam 4-step method: (1) do NOT start upstream ticker; (2) call `SendKeystroke` before
    `CreateSession`; (3) call `driver.onUpstreamTick()` synchronously; (4) assert `driver.failLoud`
    fired (via `t.Errorf`). Single-goroutine, ticker-timing-independent. [v1.6 F-B-LENSB-01]
14. [ ] Run the harness once manually to produce VP-042 evidence; hand off to PO/architect for the
    `verification_lock` decision — **this is explicitly NOT this story's Definition of Done; see Forward
    Obligation.**

## Previous Story Intelligence (MANDATORY)

| Predecessor | Lesson carried forward |
|-------------|--------------------------|
| S-BL.TESTENV (merged PR #110, `62e38d3`) | Ships the `NewLoopback`/`LoopbackConfig`/`LoopbackEnv` skeleton this story extends. `LoopbackEnv` is a named field (`struct { Env *Env }`), not embedding — confirmed via the existing merged bench call shape (`internal/bench/keystroke_echo_testenv_bench_test.go`, `develop`, PR #121 @ `4c276d9`) `env := lb.Env; env.CreateSession(b)`. |
| S-BL.BENCH (merged PR #109, `cd67394`) | VP-042 partial evidence already recorded (in-process loopback echo p99 ~0.002ms) is an honest LOWER-BOUND-ONLY measurement — declared divergence: the inline echo path bypasses arq/multipath/tick-scheduling. This story removes that divergence. |
| S-BL.PE-RECEIVE-LOOP (merged PR #118, `e940fc2`) | Established the `env.wg`/`env.closeCh`-registered ticker-goroutine idiom as house convention for test goroutines needing deterministic teardown — `startLoopbackTicker` (Q6) reuses the identical shape. Also: every new symbol claim must be grep-resolved or marked "(new — defined by this story)"; self-referential story-symbol/story-line-number citations are forbidden in story prose — use mechanism-anchor descriptions for anything this story itself defines. Disk-verified external-source line-number anchors remain permitted and are practiced throughout this story's Design Constraints (e.g. `internal/session/upstream.go:288-290`, `internal/testenv/testenv.go:461`) — both conventions followed in this story. |
| VP-042.md v1.3 | The VP's own proof-harness skeleton (`env.SendKeystroke`/`env.WaitForEcho`, no token) is directionally correct but superseded by this story's `RoundTrip`-token two-call shape (Q5) — the skeleton predates the discovery that a token is required to fix `CollectFrames`'s accumulation short-circuit. |

## Architecture Compliance Rules (MANDATORY)

| Rule | Compliance |
|------|------------|
| ARCH-08 §6.5 pos-23 import set | This story's merge FINALIZES the PROSPECTIVE v2.13 amendment; implementer runs the §6.4-equivalent machine-verification (`go list`) at merge per the testenv v2.5/v2.8/v2.11 precedent, flipping the ARCH-08 entry from PROSPECTIVE to verified. This story does not itself edit ARCH-08 prose (owned by architect). |
| §6.2 forbidden-edge check | No forbidden edge — `halfchannel`/`arq`/`multipath`/`paths` gain no new import; `testenv` remains a leaf (imported by nothing outside `_test` files). |
| `session.AccessNode` fixed-sink invariant | Preserved — `KeystrokeSink` is injected once at construction via `WithKeystrokeSink(loopbackSink)` on the driver's own `AccessNode`; no `SetSink` escape hatch is added to production `session.AccessNode` (Q2, AC-007). |
| `Env.SendKeystroke`/`Env.CollectFrames`/`Env.WaitForEcho` | Unchanged — the 10 other VPs depending on their generic SVTN-shard fan-out semantics are unaffected (Non-Goals). |

## Library & Framework Requirements (MANDATORY)

Stdlib only: `testing`, `time` (ticker), `sync`/`sync/atomic`. Internal packages: `internal/halfchannel`,
`internal/arq`, `internal/multipath`, `internal/paths` (all already vendored in-module, read-only
consumption). No new external dependency.

## File Structure Requirements (MANDATORY)

| File | Change |
|------|--------|
| `internal/testenv/loopback.go` (new — implementer's choice of filename, or inline in `testenv.go`) | `loopbackDriver` (with `upstreamHCMu`/`downstreamHCMu` — H2; `onDownstreamTick()` seam — v1.4 B-3/AC-016; `onUpstreamTick()` seam — v1.6 F-B-LENSB-01/AC-017), `RoundTrip` (`done chan []byte` — H1), `loopbackSink`, `LoopbackEnv.SendKeystroke`/`WaitForEcho`/`CreateSession` (returns `([]byte,bool)` — H1; provisions console — H3), `startLoopbackTicker`, `newLoopbackPaths` (`DefaultDropCacheSize` — LOW), `toMPFrame`, `encodeRTID`, `decodeRTID`, `zeroSACK` (M4); `frameFor` REMOVED (H1/M4) |
| `internal/testenv/testenv.go` | `NewLoopback` modified to wire halfchannel/arq/multipath/paths instead of discarding `LoopbackConfig` |
| `internal/bench/keystroke_echo_testenv_bench_test.go` (merged on `develop`, PR #121 @ `4c276d9`, `//go:build integration`-tagged) | Modified — token-based two-call shape (AC-013); "lower bound only" comment retired |
| `.factory/specs/architecture/ARCH-08-dependency-graph.md` | §6.5 pos-23 row: PROSPECTIVE → machine-verified at merge (architect/implementer act at merge time, not a story-writer edit) |
| `.factory/specs/verification-properties/VP-042.md` (Proof Harness Skeleton section) — **Forward Obligation, not this story's own edit** | This story's binding `RoundTrip`-token two-call API shape (Q5) supersedes VP-042.md's existing Proof Harness Skeleton (its `env.SendKeystroke`/`env.WaitForEcho` two-call form predates the discovery that a token is required to fix `Env.CollectFrames`'s accumulation short-circuit — see Context, Non-Goals). Per placement note v1.15 Risks item 6: a downstream act (PO/architect, alongside or following the `verification_lock` decision — see Forward Obligation below) MUST update VP-042.md's skeleton to the binding two-call token shape (`SendKeystroke` → `RoundTrip`; `WaitForEcho(t, rt, timeout) ([]byte, bool)`), so a future reader of VP-042.md is not misled by a skeleton the shipped code no longer matches. This is a forward obligation on VP-042.md, not a new AC of this story. |

---

## Delivery Plan Note — POL-005

Any adversarial or evaluation dispatch for this story (per-story pass, wave-gate Perimeter-2, or any
other evaluation dispatch) **MUST embed the POL-005 (`adversary-dispatch-integrity`, HIGH) verification
tuple** in the dispatch prompt — `{repo path, branch, expected HEAD SHA at dispatch time, artifact IDs +
versions under review}` — per `.factory/policies.yaml` POL-005 (registered 2026-07-12). The dispatched
agent's first action must verify its observed `git rev-parse HEAD` and artifact versions against the
tuple before proceeding; on mismatch, it must ABORT the pass and report the divergence as the pass
result rather than reviewing stale state.

## Forward Obligation — VP-042 `verification_lock` (explicitly NOT part of this story)

This story delivers the harness and, per AC-013/Task 14 (the run-harness-once-manually task), is run once manually to produce evidence for
VP-042.md's changelog. **Flipping `verification_lock: false → true` in VP-042.md's frontmatter is a
separate, subsequent PO/architect act** — it requires explicit sign-off distinct from "the harness
compiles and its own tests pass." Do not treat this story's merge, by itself, as a VP-042 lock event.
This mirrors how VP-042's own history table already distinguishes "audited"/"partial evidence" entries
from a lock flip.

**Second forward obligation (placement note v1.15 Risks item 6):** this story's binding `RoundTrip`-token
API shape (Q5) supersedes VP-042.md's existing Proof Harness Skeleton — see the File Structure
Requirements row above. Updating that skeleton section is likewise a separate, subsequent PO/architect
act, not part of this story's own Definition of Done; it may be bundled with the `verification_lock`
decision above or done independently.

---

## Changelog

**Ledger parity convention (declared v1.16, R20 root-cause fix):** the `inputDocuments` L57 provenance clause and the top-of-file status-note blockquote's entry (L72) each mirror this table's row for their shared version 1:1 — same finding IDs, same severities — from v1.16 forward, applied retroactively to v1.13/v1.14/v1.15 in this pass. Pre-v1.13 L57/L72 entries are grandfathered historical headline summaries and are not held to this convention.

| Version | Date | Change |
|---------|------|--------|
| 1.17 | 2026-08-29 | R21 adversarial reconvergence repair (POL-005 dispatch), fixing one architect-confirmed finding (F-R21-LENSB-01, MEDIUM) — downstream `OnAck` call-site re-anchored to the binding note (tick-body placement, not `deliverDownstream`-gated). The story's downstream `OnAck` call-site was depicted as chained off `deliverDownstream` (per-arrival, dedup-gated), which put the tick-body-local `chanSeq` out of lexical scope and diverged from the binding note, where `OnAck` runs once per tick IN THE TICK BODY after `Send` returns, structurally decoupled from `deliverDownstream`'s dedup; `multipath.Frame` has no `ChanSeq`, so the prior chained-off-`deliverDownstream` structure was not cleanly implementable. Upstream (`accessNode.SendKeystroke` inside `deliverUpstream`) was already correct and is untouched — only the downstream leg was wrong. Repaired across 8 sites: the Q4 downstream diagram (fenced pseudocode re-nested so `OnAck` and the delivered-payload loop sit inside the tick's data-frame `if` block, no longer chained off a standalone `deliverDownstream`/`Receive` arrow-chain — that standalone chain folded into an explanatory comment on `Send`); AC-005 (body + test rewritten to state the upstream/downstream dedup asymmetry explicitly — upstream's dedup gates `SendKeystroke` directly, downstream's dedup gates only `deliverDownstream` itself, never `OnAck`); AC-006 (body + test rewritten to state `OnAck`'s cadence as exactly one per downstream data tick, called from the tick body after `Send` returns, not gated by or invoked from inside `deliverDownstream`); Task 6 (rewritten to sequence `Send` returning before the same-tick-body `OnAck` call using the tick-body-local `chanSeq`, with `onDownstreamTick()` now explicitly binding the entire downstream tick body — the `OnAck` call and delivered-payload loop included — as one function); the two Anchors Consumed table rows for BC-2.02.002 and BC-2.02.005 (reworded to the corrected call-site relationship); the Design Constraints `arq.OnAck` call-contract prose (reworded to anchor the tick-body call convention to `OnAck`'s own doc comment — "the caller's tick loop forwards them to the terminal," `arq.go:195` — dropping the now-superseded "called once per received (post-dedup) downstream frame" framing); and the Edge Cases "Duplicate frame arrival" row (reworded to state `OnAck` is unaffected by the dedup outcome, since it fires once per tick regardless). `inputDocuments`/status-note pins NOT bumped — L57/L72 each gain a v1.17 entry documenting this repair, but the declared input `S-BL.LOOPBACK-FULLSTACK-placement-note.md` is unchanged at v1.15; this is a story-body-only repair. AC count 17 unchanged (no new AC; existing AC-005/AC-006 bodies and tests corrected in place). Points 8 unchanged. Input-hash UNCHANGED at `4902d5d` (declared input content did not change; confirmed via read-only rc.24 `compute-input-hash`, no `--update` run). |
| 1.16 | 2026-08-29 | R20 adversarial reconvergence repair (POL-005 dispatch), fixing two orchestrator-verified findings independently corroborated by Lens A and Lens C, both confined to this story's own summary-ledger surfaces — no placement-note-side change; note stays v1.15. MED (Finding F-R20-LENSA-01 / F-R20-LENSC-01) — the `inputDocuments` L57 v1.14 clause listed only F-R18-LENSC-01/F-R18-LENSB-01/F-R18-LENSB-02, omitting the fourth v1.14 item, O-2, that both this table's frozen v1.14 row and the status-note blockquote's v1.14 entry (L72) already carried; backfilled so L57's v1.14 clause now enumerates all four items, matching L72 and this table's v1.14 row. MED (Finding F-R20-LENSA-02 / F-R20-LENSC-02) — F-R19-LENSB-01 (the Q3 upstream-flow diagram in-place `failLoud` fix) was labeled MED in both the L57 v1.15 clause and the L72 v1.15 entry, but is LOW in this table's authoritative v1.15 row (below); corrected to LOW in both ledgers. Root-cause (whole-class): declared the ledger-parity convention above this table — L57 and L72 each mirror their formal-changelog row's finding-set 1:1, version-for-version, from v1.13 forward (pre-v1.13 entries grandfathered). Applied retroactively: L57's v1.13 clause was also missing F-R15-LENSA-01 and F-R15-LENSC-01 — present in L72's v1.13 entry and this table's v1.13 row — backfilled. Forward-correction — the v1.15 row's (below) closing sentence "both backfilled to match this table's authoritative v1.14 row" was inaccurate for the L57 side: R19 backfilled L57 with F-R18-LENSB-02 but L57 still lacked O-2 at that time, so it did not in fact fully match; that frozen v1.15 row is left as originally written per §2.9, and the L57/L72 v1.15 entries' own "both backfilled to match" wording is softened in this pass to "both backfilled with the missing item" so no ledger surface overclaims a parity that did not hold until this v1.16 pass. `inputDocuments`/status-note pins NOT bumped — L57/L72 each gain a v1.16 entry documenting this repair, but the declared input `S-BL.LOOPBACK-FULLSTACK-placement-note.md` is unchanged at v1.15; this is a story-side-only ledger reconciliation. AC count 17 unchanged; points 8 unchanged. Input-hash UNCHANGED at `4902d5d` (declared input content did not change; confirmed via read-only rc.24 `compute-input-hash`, no `--update` run). |
| 1.15 | 2026-08-29 | R19 adversarial reconvergence repair (POL-005 dispatch), fixing two propagation gaps left by the v1.14 R18 burst — the story's own follow-on findings, distinct from the placement note's separate v1.15 R19 fix (F-R19-LENSC-01, a note-internal citation repair not requiring story content changes). MED (Finding F-R19-LENSA-01) — the v1.14 §M2 window-safety off-by-one fix (F-R18-LENSB-02, "plus one") was already present in this table's frozen v1.14 row but had been omitted from two summary ledgers: the `inputDocuments` L57 v1.14 clause and the top-of-file status-note blockquote's v1.14 entry (L72), both of which listed only F-R18-LENSC-01/F-R18-LENSB-01/O-2. Both ledgers backfilled with the missing F-R18-LENSB-02 item, matching this table's authoritative v1.14 row (§2.9 backfill of a same-burst-cycle omission — not an edit to the frozen v1.14 row itself). LOW (Finding F-R19-LENSB-01) — the v1.14 F-R18-LENSB-01 fix (pinning `failLoud` IN-PLACE at the `SendKeystroke` call site inside `deliverUpstream`, never on `upstreamMP.Send`'s return value) had propagated to AC-017, Task 5, and Task 13, but not to this story's own Q3 upstream-flow pseudocode diagram (~L349), which still showed `SendKeystroke` flowing straight to `loopbackSink.SendInput` with no error check — an intra-document asymmetry against the Q4 downstream diagram's already-correct `if err != nil { driver.failLoud(err); return }` step. Added the mirroring in-place `failLoud` step to the Q3 diagram, consistent with AC-017 and the note's Q3. `inputDocuments`/Context section note-version pins updated v1.14→v1.15 (L57, L100, AC-013 Verification Method header, File Structure Requirements table row, Forward Obligation section) to track the placement note's own v1.15 bump, even though that note-side fix (F-R19-LENSC-01) required no further story transcription. AC count 17 unchanged (ledger/diagram consistency repairs, not scope growth). Points 8 unchanged. Input-hash recomputed (`06d5209` → `4902d5d`; declared input `S-BL.LOOPBACK-FULLSTACK-placement-note.md` content changed v1.14→v1.15, computed via rc.24 `compute-input-hash`). Consistent with placement-note v1.15. |
| 1.14 | 2026-08-29 | R18 adversarial reconvergence repair (POL-005 dispatch), propagating architect placement-note v1.14's repair into this story body. MED (Finding F-R18-LENSC-01) — the one live-prose citation of a stale naked `L256` line number as "the ... binding rule" (§Q6 wiring paragraph, ~L700) de-anchored: `L256` on disk is `loopbackSink.SendInput`'s doc comment, unrelated to the per-direction-mutex rule. Replaced with the note's mechanism anchor — "the §H2 per-direction-mutex binding rule (every `Enqueue`/`Tick` call site acquires the corresponding mutex)" — carrying no naked line number. Whole-file grep confirmed this was the only live naked `L<digits>` binding-rule citation; no other instances found. MED (Finding F-R18-LENSB-01) — AC-017's upstream `failLoud` placement design corrected to match the note's Q3/AC-017 ruling: the check MUST be pinned IN-PLACE at the `driver.accessNode.SendKeystroke` call site inside `deliverUpstream`, never at a check on `upstreamMP.Send`'s return value in the upstream tick body — `multipath.Send` (`multipath.go:244-286`) returns `nil` whenever at least one selected path's `Sent` is true, and in the upstream duplicate-and-race dispatch the dedup sibling path's `deliverUpstream` call always returns `nil` without ever calling `SendKeystroke`, so a tick-body check on `Send`'s return value can never observe a masked per-path error from the delivering path; `deliverUpstream` still returns the error afterward (additive to the return, not a replacement). AC-017's opening paragraph, "Required behavior" paragraph, Task 5, and Task 13 all corrected to the in-place placement; downstream (`OnAck`) wording untouched — already correct. LOW (O-2, Lens C non-gating observation) — the `inputDocuments` L62 convention clause ("no self-referential story-line-number citations") scoped to "...in live binding prose (changelog/provenance edit-location line refs excepted); disk-verified external-source anchors permitted," resolving the mild tension with the changelog rows' own edit-location line refs; L62's inputs list itself left unrestructured. Also corrected: the §M2 window-safety invariant's off-by-one (F-R18-LENSB-02, LOW) — "`chanSeq` at the first data tick equals the number of empty ticks elapsed" reworded to "...equals the number of empty ticks elapsed **plus one** (the data tick itself increments `seq`)," matching AC-016's own `E=65 → chanSeq=66` boundary and the placement note's F-B-2b fix; the correct `<64` boundary sentence left unchanged. `inputDocuments`/Context section note-version pins updated v1.13→v1.14 (L57, L100, AC-013 Verification Method header, File Structure Requirements table row, Forward Obligation section); status note blockquote gains a v1.14 entry. AC count 17 unchanged (spec-correctness repairs, not scope growth). Points 8 unchanged. Input-hash recomputed (declared input `S-BL.LOOPBACK-FULLSTACK-placement-note.md` content changed v1.13→v1.14). Consistent with placement-note v1.14. |
| 1.13 | 2026-08-29 | R15 adversarial reconvergence repair (POL-005 dispatch), propagating architect placement-note v1.13's repair into this story body. MED (Finding F-R15-LENSB-01) — AC-013's "Verification Method" compile-check corrected: the v1.12 row's `go build -tags integration ./internal/bench/...` compile check was itself vacuous and its causal claim wrong — `go build` never compiles `_test.go` files, WITH OR WITHOUT `-tags integration` (tag-independent exclusion; `internal/bench/` is test-only, containing only `keystroke_echo_bench_test.go` and `keystroke_echo_testenv_bench_test.go`, both `_test.go`), so the prescribed compile check verified nothing (empirically confirmed: exits 0 even against a deliberately injected compile error). Replaced with the binding compile-gate `go test -tags integration -run '^$' -count=1 ./internal/bench/` (empirically confirmed to fail against the same injected error), with `go vet -tags integration ./internal/bench/...` noted as a sound faster alternative; the binding RUN command (`go test -tags integration -run '^$' -bench=BenchmarkKeystrokeToEcho_P99 -benchtime=1x -count=1 ./internal/bench/`) and the `just bench` Option (a) out-of-scope adjudication are unchanged. LOW (Finding F-R15-LENSA-01) — the `inputDocuments` L62 blanket claim "story-writer conventions (grep-resolved symbols, no line-number citations)" reworded to "grep-resolved symbols; no self-referential story-line-number citations — disk-verified external-source anchors permitted," matching the Previous Story Intelligence reconciliation v1.12's Finding #3 already made at that row (this story cites external-source anchors like `internal/session/upstream.go:288-290` and `internal/testenv/testenv.go:461` in live prose) — the only live residual of that class. LOW (Finding F-R15-LENSC-01) — the v1.12 row's "11 occurrences" overcount of the phantom-branch citation is corrected here to 10, the internally-consistent figure per that row's own enumeration (Story-Sizing Rationale ×2, Q2 Design Constraints, AC-013 header + body, Package Impact Summary, Token Budget Estimate, Task 10, Previous Story Intelligence S-BL.TESTENV row, File Structure Requirements = 10); the frozen v1.12 row's text is left as originally written per §2.9 and this correction is recorded here instead of by editing that row. `inputDocuments`/Context section note-version pins updated v1.12→v1.13 (L57, L100, AC-013 Verification Method header, File Structure Requirements table row, Forward Obligation section); status note blockquote gains a v1.13 entry. AC count 17 unchanged (spec-correctness repairs, not scope growth). Points 8 unchanged. Input-hash recomputed (declared input `S-BL.LOOPBACK-FULLSTACK-placement-note.md` content changed v1.12→v1.13). Consistent with placement-note v1.13. |
| 1.12 | 2026-08-28 | Consistency-audit remediation (POL-005 dispatch, `.factory/cycles/cycle-1/S-BL.LOOPBACK-FULLSTACK/consistency-audit-2026-08-28.md`), propagating architect placement-note v1.12's repair into this story body. MAJOR (Finding #1) — every live-prose citation of the phantom branch `fix/vp-042-testenv-integrated-bench` (11 occurrences: Story-Sizing Rationale ×2, Q2 Design Constraints, AC-013 header + body, Package Impact Summary, Token Budget Estimate, Task 10, Previous Story Intelligence S-BL.TESTENV row, File Structure Requirements) corrected to the actual merged state — `internal/bench/keystroke_echo_testenv_bench_test.go` is MERGED on `develop` (PR #121 @ `4c276d9`), `//go:build integration`-tagged, defines `BenchmarkKeystrokeToEcho_P99`; the branch was never real. MAJOR (Finding #2) — AC-013's "Verification Method" rewritten: the prior bare `go build ./internal/bench/...` + `just bench` framing is wrong on both counts (neither reaches the integration-tagged function); replaced with the binding `go build -tags integration ./internal/bench/...` compile check + `go test -tags integration -run '^$' -bench=BenchmarkKeystrokeToEcho_P99 -benchtime=1x -count=1 ./internal/bench/` run, producing `p99_rtt_ms`; explicit adjudication recorded that `just bench` stays the tag-free S-BL.BENCH lower-bound recipe, out of AC-013's scope. MEDIUM (Finding #3) — the Previous Story Intelligence row for S-BL.PE-RECEIVE-LOOP's "line-number citations are forbidden in story prose" claim was inaccurate (this story contains disk-verified external-source citations like `internal/session/upstream.go:288-290` and `internal/testenv/testenv.go:461`, deliberately maintained across R7/R8); reconciled to distinguish forbidden self-referential story-symbol/story-line-number citations from permitted, practiced disk-verified external-source anchors. MINOR (Finding #4) — new File Structure Requirements row + Forward Obligation section paragraph encoding the forward obligation (placement note Risks item 6) that this story's binding two-call token API shape (`SendKeystroke` → `RoundTrip`; `WaitForEcho(t, rt, timeout) ([]byte, bool)`) supersedes VP-042.md's Proof Harness Skeleton, to be updated by a separate downstream PO/architect act — not a new AC. `inputDocuments`/Context section note-version pin updated v1.11→v1.12; status note blockquote gains a v1.12 entry. AC count 17 unchanged (spec-correctness repairs, not scope growth). Points 8 unchanged. Input-hash recomputed (declared input `S-BL.LOOPBACK-FULLSTACK-placement-note.md` content changed v1.11→v1.12). Consistent with placement-note v1.12. |
| 1.11 | 2026-08-28 | R11 spec-review repair (F-LENSA-R11-01 LOW / F-LENSC-R11-01, corroborated by two fresh lenses). The top-of-story status-note blockquote (L72) version-ledger was one revision behind the document's own version — it carried entries through v1.9 but not the v1.10 revision, and its closing "corrections govern" enumeration ended at v1.9, contradicting frontmatter version 1.10. Restored the per-version status-note convention: backfilled the missing v1.10 entry (mirroring this table's v1.10 row) and added a v1.11 entry for this repair; extended the enumeration to …/v1.9/v1.10/v1.11. No design-content change; placement-note untouched (stays v1.11); inputDocuments/Context pins unchanged. Status note blockquote updated with v1.10 (backfill) + v1.11 entries. AC count 17 unchanged. Points 8 unchanged. Input-hash 1145d15 (unchanged — declared inputs byte-identical). Consistent with placement-note v1.11. |
| 1.10 | 2026-08-28 | Transcribe note v1.11 R8 repair (F-LENSA-01/F-ORACLE-01). Corrected the third live `t.Helper()` citation in SLASH notation (`testenv.go:384/460/475/528` → `384/461/475/528`) in the `recordingTB` struct-field comment (~L545) — missed by the v1.9 `:460`→`:461` erratum, which only caught the space-colon form. Completes the citation-class sweep. inputDocuments/Context pins → note v1.11. Input-hash recomputed. AC count 17 unchanged. Consistent with placement-note v1.11. |
| 1.9 | 2026-08-28 | Transcribe note v1.10 + R7 story repairs (F-LENSB-01/F-LENSB-02). F-LENSB-01 (MED) — the §M2 "Mitigation decision" and "Driver lifecycle pin" (§H3) are corrected: the "upstream ticker MAY start at construction" / "start-time freedom PRESERVED" latitude ([v1.6 F-B-LENSB-01]) is WITHDRAWN — the upstream ticker now starts at `CreateSession` time, SYMMETRIC with the downstream ticker (both tickers, race-free for the same single-goroutine reason, no `EnqueueSend` dependency blocking either). The "Driver lifecycle pin" sentence and the "Steps 1–3" cross-reference sentence are updated so `CreateSession` is described as starting BOTH tickers, not just the downstream one. AC-017's "single-goroutine throughout" method text (Design Constraints "Required upstream seam" paragraph, and AC-017 body steps 1/4/closing sentence) is sharpened to TRUE BY CONSTRUCTION: because AC-017 never calls `CreateSession` (the sole start-site for both tickers), no ticker goroutine of either kind can run during the test; the test's direct synchronous `onUpstreamTick()` call is therefore the sole driver, so the assertion's unlocked read of `len(stub.errorfCalls)` observes no concurrent writer — closing a latent `go test -race` hazard the prior construction-start latitude permitted (the outcome count was always 1; the hazard was a race annotation on the read, not a wrong assertion value). F-LENSB-02 (LOW) — the §H3 `sessionName` storage/timing pin is corrected: AC-017's `ErrConsoleNotFound` is caused PRIMARILY by the un-attached, zero-value `loopbackConsoleKey` (`AccessNode.SendKeystroke`, `internal/session/upstream.go:288-290`, gates on console attachment BEFORE comparing `sessionName` at line 292), not by the empty `sessionName`, which is demoted to a redundant second guarantee. Erratum — two live-text citations of `t.Helper()`'s line number corrected from `:460` to the disk-verified `:461` (`internal/testenv/testenv.go`) at the `recordingTB` struct comment and the "Why embedding the real `t` is required" prose paragraph; the frozen v1.8 changelog row's `:460` citation is untouched per §2.9. `inputDocuments` pin and Context section note-version updated v1.9→v1.10. AC count: 17 (unchanged — both fixes are corrections to existing AC-017 body and Design Constraints prose, no new ACs). Points: 8 (unchanged — spec-correctness repairs, not scope growth). Status note blockquote updated with a v1.9 entry. Input-hash recomputed (`497607b` → `89d5e3c`; note content changed v1.9→v1.10, computed via rc.24 `compute-input-hash`). Consistent with placement-note v1.10. |
| 1.8 | 2026-08-28 | Transcribe note v1.9 + R6 story repairs. BLOCKER — the F-LENSB-B-03 `recordingTB` stub (Design Constraints "Recording `testing.TB` requirement" subsection) is corrected from `stub := &recordingTB{}` (embedding a nil `testing.TB`) to `stub := &recordingTB{TB: t}` (embedding the real enclosing `*testing.T`); disk-verified against `internal/testenv/testenv.go`: `NewLoopback`→`newEnv` unconditionally calls `b.Helper()` (`:384`), `t.Helper()` (`:460`), and `t.Cleanup(...)` twice (`:475`, `:528` — the ticker/env teardown AC-011/Q6 depend on), so a nil embed nil-panics at construction, before AC-016's or AC-017's fault-injection procedure — or even construction — completes; the struct comment's false "unused methods panic if called" claim is corrected to state that `Helper`/`Cleanup`/`Fatalf` promote to and are serviced by the embedded real `t`, and that only `Errorf` is overridden, to capture rather than fail; the prior "in place of the real `*testing.T`" framing and its "must NOT pass real t" implication are RETRACTED (it inverted the actual constraint — `Errorf`'s override, not the absence of a real `t`, is what keeps the enclosing test green). AC-016 step 1 and AC-017 step 1 updated to build the driver via `NewLoopback` against `&recordingTB{TB: t}` and to drop the retracted "must NOT be passed" rationale. LOW (Lens B O-1) — `sessionName` storage/timing pin added to Design Constraints (§H3 area): `sessionName` is a `loopbackDriver` field, set in `CreateSession` alongside `loopbackConsoleKey` (before Steps 1–3 run), holding its zero value (`""`) before `CreateSession` has run — which is why AC-017's pre-`CreateSession` `SendKeystroke` call reaches `accessNode.SendKeystroke(loopbackConsoleKey, "", payload)` and observes `ErrConsoleNotFound`; happy-path ACs (AC-004/005/006) have it populated by the time any upstream tick reaches the delivery callback. F-LC-R6-001 (erratum) — the v1.7 row below misquotes the removed §H3 text: it renders the removed text as a "…never in the `loopbackDriver` constructor (or the loopbackDriver constructor)" double-prohibition and labels it "redundant-latitude," but the removed text was actually the PERMISSIVE latitude "(in `CreateSession` or in the `loopbackDriver` constructor)" — an option that was withdrawn, not a redundant prohibition; the v1.7 row is left as originally written (frozen per §2.9) and this erratum is recorded here rather than by editing that row. `inputDocuments` pin and Context section note-version updated v1.8→v1.9. AC count: 17 (unchanged — both fixes are corrections to existing AC-016/AC-017 bodies and Design Constraints prose, no new ACs). Points: 8 (unchanged — spec-correctness repairs, not scope growth). Status note blockquote updated with a v1.8 entry. Input-hash recomputed (`61e8091` → `497607b`; note content changed v1.8→v1.9). Consistent with placement-note v1.9. |
| 1.7 | 2026-08-28 | Transcribe note v1.8 + R5 story repairs (F-LENSB-B-01/F-LENSB-B-02/F-LENSB-B-03). F-LENSB-B-01 (driver lifecycle pin) — the `loopbackDriver` constructor is now pinned as the SOLE builder of the `Publisher`/`SessionAuth`/`AccessNode` triple, BOTH `*multipath.Multipath` instances (`upstreamMP`/`downstreamMP`), and BOTH `*halfchannel.HalfChannel` instances (`upstreamHC`/`downstreamHC`), fully initialized and immediately usable at construction time, with only the console left UN-PROVISIONED (no `Publish`/`RegisterKey`/`Attach`); the prior "console provisioning happens ONLY in `CreateSession` — never in the `loopbackDriver` constructor (or the loopbackDriver constructor)" redundant-latitude phrasing is removed, since AC-017 requires calling `SendKeystroke` + `onUpstreamTick()` synchronously BEFORE `CreateSession` runs without any nil-deref on `upstreamMP`/`upstreamHC` — construction-time provisioning of the multipath/half-channel pairs (as opposed to the console) is therefore mandatory, not optional latitude. F-LENSB-B-02 (`SendKeystroke` no session-existence validation) — `SendKeystroke`'s mint/register/encode/`Enqueue` sequence is now explicit as UNCONDITIONAL; no session-existence guard is permitted, since AC-017 depends on a pre-`CreateSession` `SendKeystroke` call succeeding at the mint/encode/enqueue level and failing later, downstream, at `accessNode.SendKeystroke` inside `onUpstreamTick()` (`ErrConsoleNotFound` via `failLoud`) — a defensive guard would abort AC-017 at step 1 for the wrong reason. F-LENSB-B-03 (recording `testing.TB` stub) — new `recordingTB` type specified (embeds `testing.TB`, captures `Errorf` calls under a `sync.Mutex`-guarded `errorfCalls` slice) so AC-016/AC-017's fault-injection tests construct their driver via `NewLoopback` against the STUB rather than the real enclosing `*testing.T` — otherwise `driver.failLoud`'s `t.Errorf` would mark the enclosing test FAILED at the exact moment it is meant to observe a pass; AC-016/AC-017 bodies updated to reference the stub construction step and assert against `stub.errorfCalls` instead of the raw `failLoud` invocation, with cross-references to the recording-stub requirement threaded through both fault-injection test bodies (AC-016 step 1, AC-017 step 1) and the Design Constraints section. F-C-1 (erratum) — the v1.6 row below opens "Transcribe note v1.6"; that should read "Transcribe note v1.7" — v1.6's own body already states the `inputDocuments` pin moved v1.5→v1.7, so v1.6 transcribed note v1.7's content, not note v1.6's; the v1.6 row is left as originally written (frozen per §2.9) and this erratum is recorded here rather than by editing that row. `inputDocuments` pin and Context section note-version updated v1.7→v1.8. AC count: 17 (unchanged — all three findings are precision/consistency repairs to existing AC-016/AC-017 bodies and Design Constraints prose, no new ACs). Points: 8 (unchanged — spec-correctness repairs, not scope growth). Status note blockquote updated with a v1.7 entry. Input-hash recomputed (`65ffc11` → `61e8091`; note content changed v1.7→v1.8). Consistent with placement-note v1.8. |
| 1.6 | 2026-07-23 | Transcribe note v1.6 + R4 story repairs. F-B-LENSB-01 (LOW) — AC-017 fault-injection test respecified to direct-seam method: (1) do NOT start upstream ticker; (2) call `SendKeystroke` before `CreateSession`; (3) call `driver.onUpstreamTick()` synchronously → tick body runs `deliverUpstream`→`accessNode.SendKeystroke`→`ErrConsoleNotFound`→`failLoud`; (4) assert `driver.failLoud` fired. `onUpstreamTick()` named as binding directly-callable package-private seam (symmetric to `onDownstreamTick()`); upstream seam paragraph added after downstream seam paragraph in §M2; `onUpstreamTick()` seam added to Architecture Mapping `loopbackDriver` row, File Structure `loopback.go` row, and Task 13 body. F-A-1 (NITPICK) — RoundTrip doc-comment L447 purpose sentence reworded from 1-value `decodeRTID(payload) == rt.id` code token to semantic English ("the delivered payload decodes to rt.id"); whole-story sweep confirms zero remaining live 1-value `decodeRTID(payload) ==` tokens outside dated history rows. inputDocuments pin updated v1.5→v1.7 (note content changed). Status note blockquote updated with v1.6 entry. Points: 8 (unchanged). Input-hash recomputed. |
| 1.5 | 2026-07-22 | Transcribe note v1.5 + R3 story fixes. F-B-1 (HIGH) — `startLoopbackTicker` code block replaced with tick-free form (`tickBody func()` param, no `hc` param, body `case <-ticker.C: tickBody()`); wiring prose updated: upstream ticker → `d.onUpstreamTick`, downstream ticker → `d.onDownstreamTick` as `tickBody`; "same shape as" comparison updated to note `hc` param absent; Required-seam paragraph (§M2 §B-3) reworded to reflect `startLoopbackTicker` as generic no-arg-callback driver and `tickBody` wiring; Task 8 wording updated; Architecture Mapping `startLoopbackTicker` row updated. F-B-4 (LOW) — `driver.errCh chan error` "acceptable alternative" removed from AC-016 body and §M2 Design Constraints; `t.Errorf`-based `failLoud` is the sole specified error-surface mechanism. F-1 (MED) — Task 11 body assertion corrected from 1-value `decodeRTID(payload) == rt.id` to 2-value form with `[v1.5 F-1]` tag; v1.4 changelog false-claim note added. F-2/C (LOW) — Forward Obligation "Task 12" corrected to "Task 14 (the run-harness-once-manually task)". F-2/A (NITPICK) — AC-017 "Implements: Task 5" corrected to "Implements: Task 5, Task 13". Status note blockquote updated with v1.5 entry. inputDocuments pin updated v1.4→v1.5. Points: 8 (unchanged — spec-correctness repairs). Input-hash recomputed. |
| 1.4 | 2026-07-22 | Transcribe note v1.4 + R2 story fixes. B-1 — all `decodeRTID` call sites corrected to 2-value form (`id, ok := decodeRTID(payload)`); on `!ok` handling specified (skip — `rtSeq.Add(1)` ensures `id=0` never matches a real pending key); downstream-flow pseudocode, AC-014 test assertion, and Task-11 body updated to 2-value form. B-3 — AC-016 fault-injection respecified to sound method: build driver without downstream ticker goroutine; advance `downstreamHC` past 64 via 65+ synchronous empty `onDownstreamTick()` calls; enqueue one payload; call `onDownstreamTick()` once more → `ErrAckOutOfWindow`; assert `driver.failLoud` fires (loud, not silent timeout); removed both unsound prior approaches (ticker-at-NewLoopback contradicts B-F3; test-goroutine OnAck races ticker, violates arq single-writer / AC-015); `onDownstreamTick()` seam named as BINDING package-private method. B-4 — new AC-017 added: upstream-error loud failure symmetric with AC-016; `deliverUpstream`/`accessNode.SendKeystroke` error must be checked in the upstream tick path and surfaced via `driver.failLoud`; fault-injection test specified; traces to BC-2.01.001; AC count 16→17. A-N2 — v1.4 changelog row explicitly notes input-hash recomputed (note content changed v1.3→v1.4). C-LOW1 — AC-016 body gains explicit Task-6 cross-reference; AC-017 body gains explicit Task-5 cross-reference. NITPICK — `sh` receiver in H3 provisioning block annotated with one-line definition comment. Status note blockquote updated with v1.4 entry. inputDocuments pin and Context section note-version updated v1.3→v1.4. Points: 8 (unchanged — spec-correctness repairs, not scope growth). Input-hash recomputed. |
| 1.3 | 2026-07-22 | Transcribe note v1.3 + R1 story fixes. B-F1 — §H3 provisioning: `sh.pub.Publish(sessionName)` inserted as step 1 before `RegisterKey`+`Attach` (Attach gates on pub.Get; driver builds its own Publisher); sequence now pub.Publish → RegisterKey → Attach. B-F2 — Q3 `encodeRTID` call site corrected from `append([]byte(key), encodeRTID(id)...)` to `payload := encodeRTID(key, id)` (2-arg whole-payload form matching §M4 canonical definition). B-F3 — M2 lazy-start tightened: `CreateSession`-time downstream ticker start is PREFERRED (single-threaded, race-free); if first-`SendKeystroke` start is used instead, `sync.Once` is MANDATORY; language "or at first SendKeystroke" without guard removed. A-M1 — stale Edge Cases row ("WaitForEcho never called for a RoundTrip ... t.Cleanup asserts map is empty") rewritten to reflect unconditional-drain reality (AC-009): pending is drained on delivery regardless of waiter; lingering entries only when decodeRTID fails; consistent with reframed AC-011. A-M2 — note version pin in Context (L100) updated from v1.1 to v1.3; Q4 Addendum citations at two body locations disambiguated to "introduced note v1.1, carried in v1.3." A-L1 — new AC-016 added: fault-injection test for OnAck error loud (mirrors AC-011 pattern; traces to BC-2.02.005; note §M2 explicit obligation). A-L2 — AC-001 body references to `arq.go:291` and `arq.go:339` converted from numeric line-refs to mechanism-anchors (`payloadFor` / instance-local inFlight lookup and `EnqueueSend`). AC count: 15 → 16. Points: 8 (unchanged). |
| 1.2 | 2026-07-22 | Spec-review repairs (B1/H1/H2/H3/M1/M2/M3/M4 + LOW polish): B1 — downstream OnAck call now uses captured `chanSeq := f.ChanSeq` from `halfchannel.ChannelFrame`, not phantom `mpFrame.ChanSeq()`. H1 — `RoundTrip.done` changed to `chan []byte`; `WaitForEcho` returns `(payload []byte, ok bool)`; completion path sends raw payload; `frameFor` helper eliminated. H2 — `loopbackDriver` gains `upstreamHCMu`/`downstreamHCMu sync.Mutex` serializing Enqueue+Tick per direction; new AC-015 requires `just test-race` passes. H3 — `CreateSession`/driver construction must `RegisterKey(loopbackConsoleKey, RoleFull)` + `Attach(loopbackConsoleKey, sessionName)` before upstream `SendKeystroke` can succeed. M1 — AC-011 reframed: pending-map diagnostic is non-blocking `t.Cleanup` assertion against deterministic decode-mismatch scenario (consistent with unconditional drain in AC-009). M2 — `OnAck` error MUST be checked and fail loud (`driver.failLoud`); downstream ticker starts lazily (at `CreateSession`/first `SendKeystroke`) to prevent `ErrAckOutOfWindow` from idle-tick window drift. M3 — `ARCH-03-routing-engine.md` added to hash-bearing `inputs:` list; input-hash recomputed. M4 — helper signatures enumerated in Architecture Mapping and File Structure: `toMPFrame`, `encodeRTID`, `decodeRTID`, `zeroSACK` with explicit obligations; `frameFor` REMOVED. LOW — AC-002 split into (a)/(b); numeric line-refs converted to mechanism-anchors; `dropCacheCapacity`/`DefaultDropCacheSize` noted; AC-013 notes call-order alignment. AC count: 14 → 15. Points: 8 (unchanged — spec-correctness repairs, not scope growth). Consistent with placement-note v1.2. |
| 1.1 | 2026-07-12 | AC-001 amendment consuming the placement note's Q4 Addendum — AC-001 Sign-off (v1.0 → v1.1), the architect review required by Risk 1 option (a) before this story could leave draft/unscheduled status. **Verdict: REVISED, not simple CONFIRMED.** The `ackSeq`/SACK value convention is CONFIRMED correct as originally proposed. The `driver.arqServer`/`driver.arqClient` two-instance topology Q4's original code blocks showed is a structural defect: `OnAck`'s payload recovery (`payloadFor`) reads only the calling instance's own `inFlight`/`reorderBuf`, populated exclusively by that SAME instance's prior `EnqueueSend` calls — a never-`EnqueueSend`'d `arqClient` returns `(nil, nil)` from `OnAck` on every call, silently, so every `WaitForEcho` would time out on every round trip (a hard, silent benchmark failure, not the forgiving happy-path miss Risk 1's original framing assumed for this failure mode). **AC-001 status: DISCHARGED 2026-07-12** — reworded from a pre-implementation gate to a discharged record of the verdict, binding the implementer to one shared `*arq.ARQ` instance (`driver.downstreamARQ`); `EnqueueSend` and `OnAck` for a given `ChanSeq` MUST run on that same instance, in that order, within the same downstream-ticker tick. **New AC-014 added** (regression guard, not present in v1.0): a mandatory behavioral test that a full `SendKeystroke`→`WaitForEcho` round trip actually completes with non-empty delivery — guards specifically against the silent `(nil, nil)`-forever failure mode a bare "did it return" assertion would miss. The architect's alternative structural phrasing (assert the driver has exactly one `*arq.ARQ`-typed field) is accepted as supplementary coverage only; the behavioral round-trip-completes assertion is mandatory. **Mirrored throughout:** the Q4 Design Constraints subsection (heading, binding statement, downstream-ticker code block, and call-contract prose rewritten to the single-instance shape and cross-referenced to the Addendum); AC-005/AC-006 test bodies (`arqClient`/`arqServer` naming replaced with `driver.downstreamARQ`, AC-006 now cites AC-014); the Anchors Consumed table (BC-2.02.002/BC-2.02.005 rows); the Architecture Mapping and Edge Cases tables (new edge-case row for the ruled-out two-instance shape); Tasks (Task 1 marked discharged with a pointer to the Addendum, Task 6 rewritten to the single-instance wiring, new Task 11 for the AC-014 regression guard, former Task 11 renumbered to Task 12, Forward Obligation's cross-reference updated to match); Story-Sizing Rationale (new paragraph confirming the gate resolved pre-scheduling inside Task 6's existing scope — no scope growth, estimate stays 8 points); Context section (new paragraph summarizing the sign-off); the status-note blockquote (gate status updated from blocking-pending to discharged, with an explicit warning not to implement from Q4's original code blocks alone); frontmatter (`inputDocuments` placement-note pin `v1.0` → `v1.1` with the Addendum summarized inline, `acceptance_criteria_count` 13 → 14, `input-hash` recomputed to `d621ea4` per `compute-input-hash --update` — the placement note's content changed independent of this story's own edits). Package Impact Summary's "(Transcribed from the placement note)" table is left as-is by design — it mirrors the note's own Package Impact table, which the Addendum does not itself amend. |
| 1.0 | 2026-07-12 | Initial story authored to full spec, draft/unscheduled per human disposition ("author now, deliver later"). Transcribes architect placement note v1.0 (Q1–Q8 binding design decisions, 5 Risks) faithfully — no design re-derivation. 8 points (architect range 5–8; upper bound selected for AC-001's pre-implementation sign-off gate plus three additional risk-derived ACs/decisions — AC-009/AC-010/AC-011). 13 ACs, AC-001 a hard pre-implementation gate on the `arq.OnAck` call-contract (no existing production precedent). 1 Forward Obligation (VP-042 `verification_lock` flip explicitly out of scope). `depends_on: []` — S-BL.TESTENV already merged (PR #110, `62e38d3`); this story extends its `NewLoopback`/`LoopbackEnv` surface rather than blocking on it. |
