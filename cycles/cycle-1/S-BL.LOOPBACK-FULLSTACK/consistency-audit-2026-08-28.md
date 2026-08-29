---
document_type: consistency-report
level: ops
version: "1.0"
status: "fail"
producer: consistency-validator
timestamp: 2026-08-28T00:00:00
phase: 2
inputs:
  - stories/S-BL.LOOPBACK-FULLSTACK.md
  - decisions/S-BL.LOOPBACK-FULLSTACK-placement-note.md
  - stories/STORY-INDEX.md
  - specs/verification-properties/VP-042.md
  - specs/architecture/ARCH-08-dependency-graph.md
  - specs/architecture/ARCH-03-routing-engine.md
  - specs/behavioral-contracts/ss-01/BC-2.01.001.md
  - specs/behavioral-contracts/ss-01/BC-2.01.002.md
  - specs/behavioral-contracts/ss-02/BC-2.02.001.md
  - specs/behavioral-contracts/ss-02/BC-2.02.002.md
  - specs/behavioral-contracts/ss-02/BC-2.02.005.md
  - stories/S-BL.TESTENV.md
  - stories/S-BL.PE-RECEIVE-LOOP.md
  - stories/S-BL.BENCH.md
  - stories/S-BL.BENCH-DELIVERY.md
input-hash: "55d6fb7"
traces_to: stories/S-BL.LOOPBACK-FULLSTACK.md
---

# Consistency Validation Report: S-BL.LOOPBACK-FULLSTACK (Fresh-Context Perimeter Audit, Step-4.5)

> **Scope note.** This is a *targeted, single-story, fresh-context perimeter audit*
> commissioned ahead of the human approval gate for `S-BL.LOOPBACK-FULLSTACK` v1.11 — it
> is complementary to, and deliberately different from, the R12/R13/R14 within-package
> spec-adversarial passes (all clean) that already checked internal consistency of the
> story package itself. Its purpose is to check whether the story's *premises about
> things outside the story package* — the real git history and source tree of
> `switchboard-blue`, and the BCs/VPs/ARCH docs/sibling stories it cites — still hold.
> It is not a full L1→L4 pipeline sweep (that scope does not apply: this story is a
> single unscheduled backlog item, not a wave/cycle boundary), so several sections of
> the canonical template below are marked out-of-scope rather than filled with
> synthetic content. The real findings live in the custom sections following §10.

## POL-005 dispatch-integrity tuple

`git -C .factory rev-parse HEAD` = `67e07df54c9b70c691a632a7c2bd2d4cc704d967` — **matches
expected**. Proceeding. Artifacts under review confirmed at expected versions: story
v1.11 (input-hash `1145d15`), placement-note v1.11, STORY-INDEX v4.149.

## Report Metadata

| Field | Value |
|-------|-------|
| **Product** | switchboard-blue |
| **Generated** | 2026-08-28T00:00:00 |
| **Generator** | consistency-validator (fresh-context perimeter audit) |
| **Artifacts Scanned** | 15 spec/story files + live `git` history/source tree of `switchboard-blue` (`develop` branch) |

## Summary

| # | Check | Result |
|---|-------|--------|
| 1 | L2 to L3 Requirement Coverage | out of scope (single-story audit, not a full spec-chain sweep) |
| 2 | L3 to L4 Verification Property Coverage | pass — see §Anchor Correctness |
| 3 | Dependency Acyclicity | pass — `depends_on: []`, not blocked; `blocks: []` |
| 4 | Architecture Alignment | pass — see §Cross-Document Reconciliation |
| 5 | Acceptance Criteria Quality | pass, with 2 MAJOR findings on AC-013 specifically — see §Findings |
| 6 | Story Sizing (all <= 13 points) | pass — 8 points |
| 7 | Priority Consistency | pass — P2, no unresolved higher-priority blockers |
| 8 | L1 to L2 to L3 to L4 Chain Completeness | out of scope (single-story audit) |
| 9 | AC Completeness Coverage | out of scope (covered by R12/R13/R14 within-package passes) |
| 10 | ASM/R Traceability | n/a — `assumption_validations: []`, `risk_mitigations: []` by design (placement-note risks addressed via AC-009/010/011, not ASM/R-registry IDs, per frontmatter comment) |

## 1. L2 to L3 Requirement Coverage

Out of scope for this audit. This story does not introduce new L2 capabilities or BCs —
it anchors 5 pre-existing, already-published BCs (BC-2.01.001, BC-2.01.002, BC-2.02.001,
BC-2.02.002, BC-2.02.005), all of which pre-date this story and were validated at their
own L2→L3 authoring time. See §Anchor Correctness below for the semantic re-verification
performed instead.

## 2. L3 to L4 Verification Property Coverage

| BC-S.SS.NNN | Description | VP-NNN? | Justification if no VP |
|-------------|-------------|---------|----------------------|
| BC-2.01.001 | Timeslice clock fires every tick | VP-042 (+ VP-016/017/018/041/053) | covered |
| BC-2.01.002 | Empty-tick liveness signal | (no direct VP cited by this story; VP-052/053 in BC file) | this story discharges BC-2.01.002 at harness-scope only (produce, not wire-dispatch, empty ticks) — see Non-Goals |
| BC-2.02.001 | Duplicate-and-race dispatch | VP-042 (+ VP-024) | covered |
| BC-2.02.002 | Endpoint dedup | VP-042 anchor only (VP-024/054 in BC file) | covered |
| BC-2.02.005 | Downstream ARQ | VP-042 anchor only (VP-019/020 in BC file) | covered |

Result: **PASS**. Full semantic re-verification (not just ID-presence) performed in
§Anchor Correctness below.

## 3. Dependency Acyclicity

`depends_on: []` (S-BL.TESTENV already merged — PR #110, `62e38d3` — not a blocking
dependency edge); `blocks: []`. No cycle possible with zero edges. **PASS.**

## 4. Architecture Alignment

### 4.1 Module Coverage

| Architecture Component | Stories Covering It | Coverage |
|-----------------------|--------------------|---------:|
| `internal/testenv` (ARCH-08 pos-23) | S-BL.TESTENV (merged), S-BL.LOOPBACK-FULLSTACK (this story, extends) | full — import-set expansion, no new package |
| `internal/halfchannel`, `internal/arq`, `internal/multipath`, `internal/paths` | read-only consumers per this story | full — verified against real signatures, see §Sibling-Story Consistency |
| `internal/bench` | S-BL.BENCH (merged PR #109) + this story's AC-013 | **partial — see MAJOR findings below; the file this story targets is already merged via a different, unlinked PR (#121) than the story's frontmatter assumes** |

### 4.2 Component Consistency

ARCH-08 §6.5 pos-23's v2.13 PROSPECTIVE import-set amendment
(`{admission, drain, frame, outerassembler, session, upstreamdial}` →
`{admission, arq, drain, frame, halfchannel, multipath, outerassembler, paths, session,
upstreamdial}`) matches this story's Architecture Compliance Rules exactly, verified
against ARCH-08's own changelog table row `2.13`. **PASS** — see §Cross-Document
Reconciliation for full detail.

## 5. Acceptance Criteria Quality

### 5.1 Concreteness

| Story | AC Count | Vague ACs | Untestable ACs |
|-------|----------|-----------|----------------|
| S-BL.LOOPBACK-FULLSTACK | 17 | none | none — every AC names a concrete test function; AC-013's *test itself* is concrete, but its stated *verification method* is unsound against the current repo — see MAJOR findings below |

### 5.2 Testability

All 17 ACs are testable by test-writer/implementer. AC-013 is the one exception worth
flagging structurally: its prose claims `just bench` will exercise the updated benchmark,
which is verifiably false against the current `justfile` and build-tag configuration (see
§Findings, Finding 2). This does not make AC-013 *untestable* — the underlying test
(`go test -tags integration -bench=BenchmarkKeystrokeToEcho_P99 ./internal/bench/`) is
perfectly testable — but the story's stated verification method for it is wrong.

## 6. Story Sizing

| Story | Points | Status |
|-------|-------:|--------|
| S-BL.LOOPBACK-FULLSTACK | 8 | ok (architect range 5–8, upper bound selected and justified — see note below) |

Note: one of the three reasons given for selecting the upper bound (Story-Sizing
Rationale reason 3, "coordination overhead" from a cross-branch dependency on
`fix/vp-042-testenv-integrated-bench`) is undermined by Finding 1 below — that branch no
longer exists, its content is already on `develop`. The other two reasons (AC-001's
pre-implementation gate latency; four of five Risks resolving into dedicated ACs) stand
independently and still justify the 8-point selection, so this does not change the
sizing verdict, only one leg of its justification.

## 7. Priority Consistency

P2, draft/unscheduled by explicit human disposition ("author now, deliver later"). No
dependency on any P0/P1 story (`depends_on: []`). No story `blocks:` this one. **PASS.**

## 8. L1 to L2 to L3 to L4 Chain Completeness

Out of scope — this story sits entirely within an already-established L1→L4 chain (all
5 anchored BCs and VP-042 pre-date this story). No new CAP/BC/VP is introduced. Full
chain integrity for this product is the concern of the periodic full-pipeline consistency
sweep, not a single backlog-story pre-approval audit.

## 9. AC Completeness Coverage

Out of scope for this pass — BC clause-level coverage, edge-case coverage, and
cross-cutting NFR coverage for this story were the explicit subject of the R12/R13/R14
within-package adversarial lenses (reported clean). This audit's mandate was the
complementary *perimeter* check (§Anchor Correctness through §Dangling References below),
not a re-run of that AC-completeness sweep.

## 10. ASM/R Traceability

`assumption_validations: []`, `risk_mitigations: []` in frontmatter — by explicit design,
per the frontmatter's own comment: *"placement note's 5 Risks are note-local (not ASM/R-
registry IDs); addressed via AC-001/AC-009/AC-010/AC-011 instead of registry references."*
Verified against the placement note's own "Risks / Open Questions" section (5 risks): Risk
1 → AC-001, Risk 2 → AC-010, Risk 3 → AC-009, Risk 4 → AC-011, Risk 5 → Non-Goals/Forward
Obligation (deliberately not an AC — the VP-042 lock flip is scoped as a separate PO/
architect act). All 5 risks accounted for. **PASS**, n/a for the standard ASM/R-registry
table (none apply to this story).

---

## Anchor Correctness (semantic, not just ID-presence)

**Verdict: PASS**, with one non-gating observation.

Read BC-2.01.001, BC-2.01.002, BC-2.02.001, BC-2.02.002, BC-2.02.005, and VP-042 in full.
Each BC's Postconditions/Invariants/Edge-Cases genuinely support the AC that cites it:

- BC-2.01.001 (timeslice clock fires every tick) ↔ AC-002/AC-003: sound — Postcondition 1
  ("exactly one frame departs on each tick boundary regardless of whether application
  data is queued") is exactly what AC-002(b) and the two-ticker architecture implement.
- BC-2.01.002 (empty-tick liveness) ↔ AC-003: sound — the BC distinguishes "empty ticks
  are produced" from "empty ticks are wire-dispatched"; the story's Non-Goals boundary
  (produce but don't dispatch) is a harness-scope carve-out the BC doesn't forbid.
- BC-2.02.001 (duplicate-and-race) ↔ AC-004: sound, matches `multipath.Send` semantics
  exactly (verified against real code, see §Sibling-Story Consistency).
- BC-2.02.002 (dedup) ↔ AC-005: sound in outcome, see observation below.
- BC-2.02.005 (downstream ARQ) ↔ AC-006/AC-014/AC-016: sound. Its EC-005
  (`ackSeq - nextExpected > 64` → `ErrAckOutOfWindow`) is the exact mechanism AC-016's
  fault-injection test exercises, further confirmed against ARCH-03's "Input validation
  (RULING-003)" paragraph (same 64-position window rule, near-verbatim).
- VP-042's Source Contract cites only BC-2.01.001 + BC-2.02.001 (`source_bc:
  BC-2.01.001`); the story anchors 5 BCs. Not a contradiction — VP-042's Source Contract
  names the two BCs closest to the property statement itself; the story's broader anchor
  set is the harness-construction BCs needed to make the measurement honest.

**Observation (non-gating, pre-existing, not introduced by this story):** the story
labels BC-2.02.002 "endpoint checksum-only dedup." BC-2.02.002's own body (Postcondition
1, Invariant 1) actually says "identified by sequence number" — never "checksum." The
"checksum" characterization is not invented by this story: it is authoritative per
ARCH-03 line 130 ("The endpoint receiver deduplicates by checksum alone (BC-2.02.002)"),
itself a recorded correction (ARCH-03's own changelog, "Correction 1... fixed endpoint
dedup key description"). So the story's usage is correct against the authoritative
architecture doc, but BC-2.02.002.md's own prose was never back-patched to match — a
pre-existing BC/ARCH drift outside this story's remit. Recommend a separate ticket
against BC-2.02.002, not a finding against this story.

## Cross-Document Reconciliation

**Verdict: 3 of 4 targeted spot-checks PASS cleanly; 1 surfaces MAJOR new findings.**

**(a) VP-042 lock-deferral changelog — PASS.** VP-042.md v1.3 changelog reads verbatim:
*"Lock deferred to a testenv-integrated measurement post S-BL.TESTENV."* Matches the
story's Context section exactly.

**(b) ARCH-08 §6.5 pos-23 v2.13 PROSPECTIVE — PASS.** ARCH-08's own changelog (both the
`modified:` frontmatter entry and the §Changelog table row `2.13 | 2026-07-12`) carries
the exact import-set delta the story claims, `STATUS: DRAFT/PROSPECTIVE — pending
S-BL.LOOPBACK-FULLSTACK story delivery`, with the precedent citations (v2.5, v2.8, v2.11)
all real, verified rows in ARCH-08's own changelog.

**(c) S-BL.BENCH declared lower-bound-only divergence — PASS**, via the correct artifact.
`S-BL.BENCH.md` itself is only the pre-delivery backlog stub (status: backlog, version
`0.1-backlog-stub`) — the delivered evidence lives in `S-BL.BENCH-DELIVERY.md` (PR #109),
which states verbatim: *"the loopback is a lower bound (no arq, no multipath, no tick
scheduling)"* and *"Full-stack VP-042 verification requires S-BL.TESTENV."* Matches the
story's characterization exactly.

**(d) ARCH-03 consistency — PASS.** ARCH-03's "Downstream ARQ" and "Duplicate-and-Race"
sections align with the story's Design Constraints Q3/Q4: ARCH-03's "Delivery contract
(S-4.03 pass-2 adjudication): `OnAck` returns deliverable frames synchronously as
`[][]byte`... The caller's tick loop forwards them to the terminal" is exactly the
contract the story's downstream-tick pseudocode implements.

### FINDING 1 — MAJOR: stale cross-branch reference to already-merged code

The story (frontmatter, Context, AC-013, Package Impact Summary, File Structure
Requirements, Task 10, and Story-Sizing Rationale reason 3) repeatedly describes
`internal/bench/keystroke_echo_testenv_bench_test.go` as a **"WIP bench test"** living
**"on branch `fix/vp-042-testenv-integrated-bench`"**.

Verified directly against the `switchboard-blue` git history and current `develop` HEAD:

- The branch `fix/vp-042-testenv-integrated-bench` **does not exist**, locally or on
  `origin` (`git branch -a`, `git ls-remote --heads origin` — no match).
- The file was **already merged into `develop`** via **PR #121**, commit `4c276d935b089`,
  **2026-07-12 17:32:22 -0500** — `fix(VP-042): testenv-integrated keystroke-echo p99
  benchmark (#121)`. Present in current `develop` HEAD (blob `ce69ae62...`).
- That merge date is the **same calendar day** this story was first authored (v1.0/v1.1,
  2026-07-12). The story has since gone through 11 revisions across six weeks
  (→ 2026-08-28) and eight placement-note revisions without correcting this premise.
- The merged file's header comment independently documents the same divergence analysis
  the story's Context section gives — strong evidence this is the intended target file,
  just already landed.
- The merged file's body confirms the two-call, non-token shape the story describes as
  needing replacement: `env.SendKeystroke(b, sessionID, "x"); env.WaitForEcho(b,
  sessionID, "x", echoTimeout)` via `env := lb.Env`. Only the "still on an unmerged
  branch" premise is stale — the technical characterization of the *current* shape is
  accurate.

**Impact:** an implementer following AC-013/Task 10 literally would look for a
nonexistent branch. Correct instruction: edit this file directly on `develop` (or the
story's own feature branch cut from `develop`) — no cross-branch coordination is needed.
This also undercuts (without invalidating — two of three reasons still stand) Story-Sizing
Rationale reason (3), which cites "coordination overhead... from a file on a different
branch" as part of the justification for the 8-point (upper) sizing.

### FINDING 2 — MAJOR: AC-013's stated verification method does not exercise the file it claims to

Compounding Finding 1: even correctly editing the already-merged file, AC-013's
verification text — *"`go build ./internal/bench/...` succeeds... and `just bench` runs
the updated benchmark to completion, producing a `p99_rtt_ms` metric"* — does not
actually exercise it:

1. **Build tag exclusion.** The merged file is guarded by `//go:build integration`. A
   bare `go build ./internal/bench/...` (no `-tags integration`) silently excludes it
   from compilation — no error, the file is simply invisible to the build.
2. **Name and recipe mismatch.** `justfile`'s `bench` recipe runs `go test
   -bench=BenchmarkKeystrokeEcho_P99 -benchtime=1x -count=1 ./internal/bench/` — no
   `-tags integration`, targeting `BenchmarkKeystrokeEcho_P99` (**no "To"**), a *different,
   untagged* function in the separate file `keystroke_echo_bench_test.go` (S-BL.BENCH's
   original lower-bound benchmark, confirmed present on `develop`). The testenv-integrated
   benchmark this story updates is `BenchmarkKeystrokeToEcho_P99` (**with "To"**), in the
   `integration`-tagged file. `just bench` **never invokes it**.

No task/AC updates the `justfile` `bench` recipe or otherwise establishes a real
invocation path for the benchmark this story updates. As written, `just bench` will run
and pass using the *old, untouched* lower-bound benchmark, giving a false impression that
AC-013 is satisfied.

**Recommend:** either (a) add a task to update the `bench` recipe (or add a
`bench-integration` recipe) invoking `go test -tags integration
-bench=BenchmarkKeystrokeToEcho_P99 ./internal/bench/`, or (b) reword AC-013's
verification method to name that direct invocation instead of `just bench`.

## Coverage Gaps

**Verdict: no blocking gaps in the harness's coverage of VP-042's stated protocol
elements.** Two non-blocking observations:

- **MINOR:** VP-042.md's "Proof Harness Skeleton" (a two-call, non-token,
  pre-`LoopbackEnv`-wrapper sample) is not scheduled for update by any task in this
  story. The story's own Context acknowledges the skeleton is "superseded," and the
  Forward Obligation section defers the `verification_lock` flip to a separate act — but
  never states that act must also refresh the embedded code sample. Recommend the
  Forward Obligation section explicitly name this.
- **OBSERVATION (not a gap):** the harness's downstream-ARQ modeling calls `EnqueueSend`
  and `OnAck` on one shared instance within the same tick, using a locally-derived
  `ackSeq` rather than a peer-supplied ACK after a real round trip — so VP-042's measured
  latency never includes any piggyback-ACK-round-trip contribution. This is extensively
  disclosed in the Q4 Addendum/AC-006, and is consistent with ARCH-03's stated `OnAck`
  contract and BC-2.02.005 Postcondition 4 (terminal delivery gates on in-order arrival,
  not ACK receipt) — architecturally sound, not a hidden gap. Recorded for explicit
  reader sign-off only.

## Sibling-Story Consistency

**Verdict: technical claims about S-BL.TESTENV/S-BL.PE-RECEIVE-LOOP's shipped code are
extraordinarily precise — every spot-checked line-number citation, API signature, and
idiom claim matches real `develop` source exactly.** One documentation-integrity finding.

Verified directly against `internal/testenv/testenv.go`, `internal/session/upstream.go`,
`internal/arq/arq.go`, `internal/halfchannel/halfchannel.go`,
`internal/multipath/multipath.go`, `internal/paths/paths.go`, `cmd/switchboard/access.go`
on `develop`:

- `NewLoopback` (`testenv.go:383-387`) discards `cfg`, calls `newEnv(ctx, b, 1)` — exact.
- `LoopbackConfig.TickIntervalUpstream`/`TickIntervalDownstream` — confirmed dead fields.
- `LoopbackEnv struct { Env *Env }` — confirmed named field, not embedding.
- `Env.SendKeystroke` calls `sh.access.DeliverFrame(hdr)` directly, bypassing
  `session.AccessNode.SendKeystroke`/`KeystrokeSink` — exact.
- `b.Helper()` at `testenv.go:384`, `t.Helper()` at `:461`, `t.Cleanup(...)` at `:475`
  and `:528` — all four line numbers match exactly, including the corrected `:461` (the
  subject of the story's own v1.9/v1.10/v1.11 citation-repair chain).
- `AccessNode.SendKeystroke` (`internal/session/upstream.go:276-301`) — `sinkMu.Lock()`
  at 285, `attached, ok := a.consoles.Session(key)` at **line 288**, `ErrConsoleNotFound`
  gate at **lines 289-290**, `attached != sessionName` → `ErrSessionMismatch` at **line
  292** — byte-for-byte match to the story's F-LENSB-02 fault-attribution paragraph.
- `AttachConsole`'s `e.wg.Add(1)` / `go func() { defer e.wg.Done(); select { case
  <-e.closeCh: return ... } }()` idiom — exact match to the Q6 precedent claim.
- `cmd/switchboard/access.go` has both `startSweepTicker` and `startFramesDroppedTicker`.
- `arq.ARQ.OnAck(ackSeq uint32, sackBitmap [SACKBitmapBytes]byte) ([][]byte, error)`,
  `EnqueueSend(seq uint32, payload []byte, now time.Time)`, `halfchannel.MinTickInterval
  = 5ms` / `MaxTickInterval = 50ms`, `multipath.NewMultipath(pathSet []paths.RankedPath,
  dropCacheCapacity int)`, `Send`/`Receive`/`ErrDuplicate`/`DefaultDropCacheSize =
  10_000`, `paths.NewPathTracker(initialRTTMS, alpha float64) *PathTracker` /
  `IsActive() bool` — all signatures confirmed exact matches.

### FINDING 3 — MEDIUM: false self-consistency claim re: line-number-citation convention

The story's Previous Story Intelligence table states: *"S-BL.PE-RECEIVE-LOOP...
line-number citations are forbidden in story prose — use mechanism-anchor descriptions
(both followed in this story)."* Confirmed the convention text is real and accurately
quoted (`S-BL.PE-RECEIVE-LOOP.md:86-88`).

However, **"(both followed in this story)" is false** for the line-number-citation
clause. This story's Design Constraints and AC-016/AC-017 sections are saturated with
literal numeric line-number citations — `testenv.go:384`, `:461` (formerly `:460`),
`:475`, `:528`; `internal/session/upstream.go:276-301`, `:288`, `289-290`, `:292` — and
**three entire adversarial repair rounds of this story (v1.9, v1.10, v1.11) were
dedicated to correcting drift in exactly these citations** (the `:460`→`:461` erratum and
its slash-form duplicate). The convention was honored only in AC-001's body (v1.3's A-L2
fix converted `arq.go:291`/`arq.go:339` to mechanism-anchors).

This did not surface in R12-R14's within-package passes because they treated citation
*precision* as the correctness question rather than asking whether citations belong in
the prose at all per the story's own claimed convention. **Not a design defect** — every
citation independently re-verified accurate against real source. Recommend correcting or
scoping down the "(both followed)" claim.

## Dangling/Broken References

**Verdict: 1 dangling reference (Finding 1 above); all other references resolve
cleanly.**

Confirmed **NOT dangling**: BC-2.01.001, BC-2.01.002, BC-2.02.001, BC-2.02.002,
BC-2.02.005, VP-042, ARCH-08, ARCH-03, S-BL.TESTENV.md (+ delivered state on `develop`),
S-BL.PE-RECEIVE-LOOP.md, S-BL.BENCH.md/S-BL.BENCH-DELIVERY.md, STORY-INDEX.md row (v4.149,
matches dispatch tuple), and every cited Go API surface (§Sibling-Story Consistency).

Confirmed **dangling**: `fix/vp-042-testenv-integrated-bench` (Finding 1).

---

## Cross-Reference Validation

### ID Consistency

| Check | Status | Issues |
|-------|--------|--------|
| BC IDs unique | pass | none |
| VP IDs unique | pass | none |
| Story ACs trace to valid BC | pass | all 5 BC anchors resolve; AC→BC trace annotations verified sound (see §Anchor Correctness) |

### Canonical Frontmatter Validation

| Artifact | document_type | level | version | producer | traces_to | Status |
|----------|--------------|-------|---------|----------|-----------|--------|
| S-BL.LOOPBACK-FULLSTACK.md | present | ops | present ("1.11") | present | present | pass |

## Spec vs Implementation Drift

| Artifact | Spec Version | Implementation State | Drift Detected | Notes |
|----------|-------------|---------------------|---------------|-------|
| `internal/bench/keystroke_echo_testenv_bench_test.go` | story describes as WIP/unmerged | already merged (PR #121, `4c276d9`, 2026-07-12) | **yes** | Finding 1 |
| `justfile` `bench` recipe | story's AC-013 assumes it exercises the new benchmark | recipe targets a different, untagged benchmark function | **yes** | Finding 2 |
| `internal/testenv`, `internal/session`, `internal/arq`, `internal/halfchannel`, `internal/multipath`, `internal/paths` APIs | story's binding design cites specific signatures/line numbers | all verified exact matches | no | §Sibling-Story Consistency |

## Findings

### Critical
None.

### Major
1. **Stale cross-branch reference** to `fix/vp-042-testenv-integrated-bench` (frontmatter,
   Context, AC-013, Package Impact Summary, File Structure Requirements, Task 10,
   Story-Sizing Rationale reason 3) — the branch no longer exists; its target file merged
   to `develop` via PR #121 the same day this story was first authored. Should be
   corrected before/during `dev-story` dispatch.
2. **AC-013's verification method is unsound** against the current repo — `just bench`
   neither compiles (build-tag exclusion) nor targets (name mismatch) the benchmark this
   story updates. Should be corrected before/during `dev-story` dispatch.

### Minor
3. **MEDIUM-severity documentation-integrity finding** (see §Sibling-Story Consistency,
   Finding 3): false "(both followed in this story)" claim re: the line-number-citation
   convention inherited from S-BL.PE-RECEIVE-LOOP.
4. VP-042.md's Proof Harness Skeleton not scheduled for update after this story ships.
5. Pre-existing (not this story's) BC-2.02.002 body-text vs ARCH-03 "checksum"
   characterization drift — recommend a separate ticket.

## Validation Gate Result

**FAIL** — blocking findings: Major #1 and #2 above (both concrete, narrow, and cheaply
fixable — neither undermines the core technical design, which was independently
re-verified line-by-line against real source and found exceptionally accurate). Findings
#3-#5 are non-blocking and may be deferred.

## Overall Metrics

| Metric | Value |
|--------|-------|
| **Total Checks** | 5 perimeter dimensions (anchor correctness, cross-doc reconciliation, coverage gaps, sibling-story consistency, dangling references) + 10 template-mapped checks |
| **Passed** | 8 of 10 template checks fully; 3 of 5 perimeter dimensions clean, 2 with findings |
| **Failed** | 0 structural/critical; 2 Major (both narrow, corrective, non-structural) |
| **Warnings** | 3 (1 Medium, 2 Minor) |
| **Overall Status** | inconsistencies-found (perimeter drift only — core design sound) |

The core technical design (Q2–Q8, AC-001/AC-014's shared-`*arq.ARQ`-instance resolution,
goroutine lifecycle, halfchannel mutex discipline) is exceptionally well-grounded — every
spot-checked claim against real source code, including precise line-number citations,
verified byte-for-byte accurate. The blocking findings are narrow and mechanical: a stale
cross-branch reference to code merged the same day the story was first authored (never
re-verified across 11 subsequent revisions), and a knock-on AC-013 verification method
that, as worded, will not actually exercise the file/benchmark it claims to. Recommend
both are corrected in the story text before/during `dev-story` dispatch so the
implementer is not sent chasing a phantom branch or given a false-positive verification
step. The Medium/Minor items are non-blocking opportunistic cleanup.

## Appendix: Validation Methodology

Fresh-context, single-story perimeter audit performed by directly reading (Read/Grep) the
full text of the story, its placement note, its 5 anchored BCs, VP-042, ARCH-08, ARCH-03,
STORY-INDEX, and all named sibling stories (S-BL.TESTENV, S-BL.PE-RECEIVE-LOOP,
S-BL.BENCH/-DELIVERY) inside `.factory/`; then cross-checking every factual claim the
story makes about code *outside* `.factory/` (branch existence, merge history, exact API
signatures, exact source line numbers) directly against the live `switchboard-blue` git
repository (`git branch -a`, `git ls-remote`, `git log`, `git show <sha>:<path>`, `Read`
on `internal/testenv/testenv.go`, `internal/session/upstream.go`,
`internal/arq/arq.go`, `internal/halfchannel/halfchannel.go`,
`internal/multipath/multipath.go`, `internal/paths/paths.go`,
`cmd/switchboard/access.go`, and `justfile`). No modification made to any source or spec
artifact; this report is the only file written. Consistent with the consistency-validator
AGENTS.md constraint set (report-only, never modifies source artifacts).
