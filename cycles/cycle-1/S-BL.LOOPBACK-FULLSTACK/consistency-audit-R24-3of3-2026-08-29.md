---
document_type: consistency-report
level: ops
version: "1.0"
status: "fail"
producer: consistency-validator
timestamp: 2026-08-29T00:00:00
phase: 2
inputs:
  - stories/S-BL.LOOPBACK-FULLSTACK.md
  - decisions/S-BL.LOOPBACK-FULLSTACK-placement-note.md
  - stories/STORY-INDEX.md
  - specs/behavioral-contracts/ss-01/BC-2.01.001.md
  - specs/behavioral-contracts/ss-01/BC-2.01.002.md
  - specs/behavioral-contracts/ss-02/BC-2.02.001.md
  - specs/behavioral-contracts/ss-02/BC-2.02.002.md
  - specs/behavioral-contracts/ss-02/BC-2.02.005.md
  - specs/verification-properties/VP-042.md
  - specs/architecture/ARCH-08-dependency-graph.md
  - specs/architecture/ARCH-03-routing-engine.md
  - specs/architecture/ARCH-INDEX.md
input-hash: "50b49b5"
traces_to: stories/S-BL.LOOPBACK-FULLSTACK.md
---

# Consistency Validation Report: S-BL.LOOPBACK-FULLSTACK — §1.7 Fresh-Context Perimeter Audit

> **Scope note:** this is a §1.7 single-story fresh-context perimeter audit run
> after Step-4.5 adversarial reconvergence reached 3/3 (R22/R23/R24 all
> CLEAN), not a full-pipeline L1-to-L4 consistency validation. It checks
> whether the story's declared anchor set correctly and completely covers
> what the story must, and whether the spec package (story, placement note,
> STORY-INDEX, the 5 anchored BCs, VP-042, ARCH-08, ARCH-03) is mutually
> coherent — not whether the whole project's dependency graph, story sizing,
> or ASM/R registers are consistent. Sections of the canonical template that
> address whole-project structure (Dependency Acyclicity, Story Sizing across
> all stories, ASM/R Traceability, full L1-L4 chain) are marked N/A below with
> a pointer to why, and the actual audit content lives in the sections that do
> apply plus a dedicated "§1.7 Perimeter Audit" section.

## Report Metadata

| Field | Value |
|-------|-------|
| **Product** | switchboard |
| **Story under audit** | S-BL.LOOPBACK-FULLSTACK v1.17 (`input-hash 4902d5d`) |
| **Generated** | 2026-08-29T00:00:00 |
| **Generator** | consistency-validator |
| **Artifacts Scanned** | 12 spec artifacts (listed in `inputs:` above) + 18 code-level symbols/line-citations independently re-derived against `develop@2ce3a57` |

## POL-005 Dispatch-Integrity Tuple — VERIFIED

- `git -C .factory rev-parse HEAD` → `8712d52062b4639afc7965524aba81d3ef0b8cce` — matches expected. PROCEED.
- `stories/S-BL.LOOPBACK-FULLSTACK.md`: `version: "1.17"`, `input-hash: "4902d5d"` — confirmed on disk.
- `decisions/S-BL.LOOPBACK-FULLSTACK-placement-note.md`: `version: "1.15"` — confirmed on disk.
- `stories/STORY-INDEX.md`: `version: "4.158"` — confirmed on disk.
- STORY-INDEX's own row text confirms: "R24 CLEAN... convergence counter ADVANCES 2/3→3/3 — Step-4.5 adversarial convergence 3/3 RECONVERGED at v1.17; pending §1.7 fresh-context consistency audit + human approval gate" — confirms this audit runs at the correct pipeline point against the correct artifact versions.

All four tuple elements verified.

## Summary

| # | Check | Result |
|---|-------|--------|
| 1 | L2 to L3 Requirement Coverage (BC↔AC semantic mapping, this story's 5 anchored BCs) | pass |
| 2 | L3 to L4 Verification Property Coverage (BC↔VP-042 linkage) | pass (with Finding 2, non-blocking) |
| 3 | Dependency Acyclicity | N/A — single-story scope; see §3 |
| 4 | Architecture Alignment (ARCH-08 position numbers, ARCH-03 delivery contract) | pass |
| 5 | Acceptance Criteria Quality (17 ACs) | pass |
| 6 | Story Sizing | pass (8 pts, within architect range) |
| 7 | Priority Consistency | pass |
| 8 | L1 to L2 to L3 to L4 Chain Completeness | N/A — single-story scope; see §8 |
| 9 | AC Completeness Coverage | pass |
| 10 | ASM/R Traceability | N/A — this story carries no ASM/R registry entries (`assumption_validations: []`, `risk_mitigations: []` are explicit in frontmatter, with justification); see §10 |
| — | **§1.7 Perimeter Audit — subsystems anchor** | **FAIL (Finding 1, HIGH)** |

## 1. L2 to L3 Requirement Coverage

### 1.1 Anchored BCs to this story's ACs

This story does not derive from L2 capabilities directly (it is a
test-infrastructure story authored from an architect design note, not a PRD
capability); the relevant coverage check is BC → AC.

| BC-S.SS.NNN | Description | Covered by AC? | Gap? |
|---|---|---|---|
| BC-2.01.001 | Timeslice clock fires every tick regardless of data | AC-002 | no |
| BC-2.01.002 | Empty-tick frame is a valid liveness signal | AC-003 (partial, harness-scope — honestly disclosed) | no |
| BC-2.02.001 | Duplicate-and-race: same frame on two fastest paths | AC-004 | no |
| BC-2.02.002 | Receiver delivers first-arriving copy, discards duplicates | AC-005 | no |
| BC-2.02.005 | Downstream ARQ with piggybacked ACK/SACK | AC-001/AC-006/AC-014/AC-016 | no |

All five BC→AC mappings independently re-verified against the target BC's
actual postconditions/invariants/edge cases (not just ID resolution) — see
§1.7 Perimeter Audit, Dimension 1a, for the full semantic cross-check and the
code-level verification of every load-bearing claim underneath them (e.g. the
`multipath.Frame` has no `ChanSeq` field claim, the `OnAck` single-shared-
instance requirement, the `multipath.Send` masking-error mechanics).

## 2. L3 to L4 Verification Property Coverage

### 2.1 Behavioral Contracts to Verification Properties

| BC-S.SS.NNN | Description | VP-NNN? | Justification if no VP |
|---|---|---|---|
| BC-2.01.001 | Timeslice clock | VP-042 (also VP-016/017/018/041/053) | — |
| BC-2.01.002 | Empty-tick liveness | — (no VP-042 back-reference in this BC's own file) | see Finding 2 — asymmetry, not a missing-VP defect |
| BC-2.02.001 | Duplicate-and-race | VP-042 (explicit row, proof method "benchmark") | — |
| BC-2.02.002 | Endpoint dedup | — (no VP-042 back-reference) | see Finding 2 |
| BC-2.02.005 | Downstream ARQ | — (no VP-042 back-reference) | see Finding 2 |

VP-042.md's own "Source Contract" section names only BC-2.01.001 and
BC-2.02.001 — exactly the two BCs above that carry an explicit VP-042
back-reference in their own files. The story anchors to 3 additional BCs
(BC-2.01.002, BC-2.02.002, BC-2.02.005) as harness-construction machinery,
per the placement note's Q1 reasoning. This is **Finding 2 (MEDIUM,
non-blocking)** — see §1.7 Perimeter Audit, Dimension 3, for full detail and
disposition.

## 3. Dependency Acyclicity

N/A for this audit's scope. `S-BL.LOOPBACK-FULLSTACK` carries `depends_on: []`
and `blocks: []` in its own frontmatter (confirmed on disk) — it is not
blocked on `S-BL.TESTENV` despite extending its API, because `S-BL.TESTENV`
is already merged (PR #110, `62e38d3`). No cycle is possible from a single
zero-dependency node. Full fleet-wide topological-sort validation is outside
a single-story perimeter audit's scope.

## 4. Architecture Alignment

### 4.1 Module Coverage

| Architecture Component | Position (ARCH-08) | Story's Relationship | Coverage |
|---|---|---|---|
| `internal/testenv` | 23 | modified (New `loopbackDriver`, `RoundTrip`, etc.) | full |
| `internal/halfchannel` | 5 | read-only consumer | full |
| `internal/arq` | 9 | read-only consumer; first production-adjacent `OnAck` call site | full |
| `internal/multipath` | 14 | read-only consumer | full |
| `internal/paths` | 11 | read-only consumer (transitive, Go-imposed) | full |
| `internal/bench` | 24 | modified (`keystroke_echo_testenv_bench_test.go`, AC-013) | full |

All four new import-set positions (5, 9, 11, 14) independently re-verified
against the current ARCH-08 table — all strictly below `internal/testenv`'s
position 23, confirming the "lawful forward edges, no cycle" claim. ARCH-08's
own frontmatter version is `"2.13"` — the current, latest version — and its
top changelog row is this exact story's PROSPECTIVE amendment, correctly
still marked pending story delivery.

### 4.2 Component Consistency

Story `architecture_modules:` frontmatter (`internal/testenv`,
`internal/halfchannel`, `internal/arq`, `internal/multipath`,
`internal/paths`) matches the Architecture Mapping table in the story body
exactly. ARCH-03's binding delivery contract for `OnAck` ("returns
deliverable frames synchronously as `[][]byte`... no internal
`DeliveredFrames` channel," ARCH-03 lines 212-218) was independently
re-derived against `internal/arq/arq.go:201`'s actual signature
(`func (a *ARQ) OnAck(ackSeq uint32, sackBitmap [SACKBitmapBytes]byte)
([][]byte, error)`) — confirmed to match exactly.

**No undeclared-component references found.** See Finding 1 in §1.7 Perimeter
Audit for the one architecture-alignment gap found — not a module/component
issue, but the story's `subsystems:` anchor (a level above individual Go
packages) is misaligned against ARCH-INDEX's Subsystem Registry.

## 5. Acceptance Criteria Quality

### 5.1 Concreteness

| Story | AC Count | Vague ACs | Untestable ACs |
|---|---|---|---|
| S-BL.LOOPBACK-FULLSTACK | 17 (AC-001–AC-017; frontmatter `acceptance_criteria_count: 17` matches body count) | none found | none found |

### 5.2 Testability

Every AC (AC-002 through AC-017; AC-001 is a discharged pre-implementation
process gate, not a code AC) names a specific `Test:` — a concrete test
function name and, for the fault-injection ACs (AC-016/AC-017), an exact
numbered procedure using a directly-callable package-private seam
(`onDownstreamTick()`/`onUpstreamTick()`) rather than timing-dependent ticker
races. No AC requires subjective human judgment. AC-014's regression guard
explicitly specifies both a mandatory behavioral assertion (round trip
actually completes with non-empty, correctly-decoded payload) and an
optional supplementary structural assertion, correctly distinguishing the two
so the mandatory assertion cannot be satisfied by a weaker check alone.

## 6. Story Sizing

| Story | Points | Status |
|---|---:|---|
| S-BL.LOOPBACK-FULLSTACK | 8 | ok (architect range 5-8; story-writer selected the upper bound with an explicit, disk-checkable rationale — AC-001's pre-implementation gate latency, 3 of 5 Risks resolving into gating ACs, and real merged-file fan-out into `internal/bench`) |

## 7. Priority Consistency

Story priority: P2, status: draft/unscheduled (`wave: backlog`). `depends_on:
[]`, `blocks: []` — no priority-inversion is possible with zero dependency
edges in either direction. STORY-INDEX confirms this story is not blocking
or blocked by any other backlog item.

## 8. L1 to L2 to L3 to L4 Chain Completeness

N/A in the full-project sense (this is not a whole-pipeline audit). The
L3-to-L4 slice relevant to this story (5 BCs → VP-042) is covered in §2
above and in the §1.7 Perimeter Audit Dimension 3 finding. No L1/L2
artifacts were in scope for this dispatch (this story does not itself trace
to a PRD/L2 capability — it derives from an architect design note answering
an open question VP-042.md's own history had flagged).

### Broken Chains

None found within this story's own chain (note → story → BCs → VP-042 →
ARCH-08/ARCH-03 → STORY-INDEX). See §1.7 Perimeter Audit for the two
real gaps found, both of which are asymmetries/omissions rather than broken
references.

### Orphaned Artifacts

None found among the artifacts scanned.

## 9. AC Completeness Coverage

### 9.1 BC Clause Coverage (Level 1)

| BC-S.SS.NNN | Total Clauses (PC+INV) | Covered | Uncovered | Gap Entries | Coverage % |
|---|---|---|---|---|---|
| BC-2.01.001 | 5 PC + 3 INV = 8 | 8 | 0 | 0 | 100% |
| BC-2.01.002 | 5 PC + 3 INV = 8 | 3 PC (PC1-3) + 3 INV | 2 PC (PC4-5, router/receiver-level) | 1 (explicit Non-Goals disclosure: "Empty-tick wire dispatch") | 75% (honestly declared partial — see §1.7 Dim. 2) |
| BC-2.02.001 | 5 PC + 3 INV = 8 | 8 | 0 | 0 | 100% |
| BC-2.02.002 | 4 PC + 2 INV = 6 | 6 | 0 | 0 | 100% |
| BC-2.02.005 | 5 PC + 4 INV = 9 | EnqueueSend/OnAck happy path (PC1-3,5 + INV1,3) | GapsToRetransmit/TLPKTDROP (PC not applicable — no loss simulated, explicit Non-Goal) | 1 (explicit Non-Goals disclosure: "Simulated packet loss, retransmission, or TLPKTDROP") | ~78% (partial by explicit, disclosed design — loss/retransmit is out of this harness's Non-Goals, not an oversight) |

**L1 Score:** ~91% (weighted by clause count; the two partial rows are both
explicitly and honestly declared in the story's own Non-Goals section, not
silent gaps — see §1.7 Perimeter Audit Dimension 4 observation on the one
place this disclosure asymmetry in framing style could be tightened).

### 9.2 Edge Case & Error Coverage (Level 2)

| Source | Total IDs | Covered | Orphaned | Coverage % |
|---|---|---|---|---|
| BC Edge Cases (EC-NNN) relevant to this harness | 4 (BC-2.02.002 EC-004/EC-005 window/reserved-seq; BC-2.02.005 EC-005 out-of-window ACK; BC-2.01.002 EC-004 double-delivery) | 4 (EC-005 out-of-window ACK is the direct subject of AC-016; the others are exercised implicitly by construction — no loss/reorder is simulated, so the wrap/reservation edge cases are structurally inapplicable, not skipped) | 0 | 100% |
| Error Taxonomy (E-xxx-NNN) | N/A — this story does not introduce new production error paths; `ErrConsoleNotFound`/`ErrSessionMismatch`/`ErrAckOutOfWindow` are pre-existing sentinels this story's ACs (AC-016/AC-017) exercise, not define | — | — | N/A |

**L2 Score:** 100% (of the in-scope edge cases; error-taxonomy is N/A for
this story's own change footprint)

### 9.3 Cross-Cutting Coverage (Level 3)

| Category | Total | Covered | Uncovered | Coverage % |
|---|---|---|---|---|
| NFR-NNN (P0/P1) | 1 (NFR-001, keystroke-to-echo p99, cited by VP-042) | 1 | 0 | 100% |
| Holdout-BC Alignment | N/A — this story is draft/unscheduled; no holdout scenarios exist for it yet | — | — | N/A |
| UI Component States | N/A — not a UI story | — | — | N/A |

**L3 Score:** 100% (of applicable categories)

### 9.4 AC Completeness Summary

| Level | Weight | Score | Weighted |
|---|---|---|---|
| L1 — BC Clause Coverage | 50% | 91% | 45.5% |
| L2 — Edge Case & Error Coverage | 30% | 100% | 30.0% |
| L3 — Cross-Cutting Coverage | 20% | 100% | 20.0% |
| **Overall** | **100%** | | **95.5%** |

**Gate Result:** PASS (threshold: >= 90% weighted overall) — the partial-
coverage rows (BC-2.01.002, BC-2.02.005) are both deliberate, disclosed
harness-scope boundaries per the story's own Non-Goals section, cross-checked
against VP-042.md's requirements and found not to conflict with anything
VP-042 actually needs (§1.7 Perimeter Audit Dimension 2).

## 10. ASM/R Traceability

N/A — story frontmatter explicitly carries `assumption_validations: []` and
`risk_mitigations: []`, with the justification recorded inline: "placement
note's 5 Risks are note-local (not ASM/R-registry IDs); addressed via
AC-001/AC-009/AC-010/AC-011 instead of registry references." Independently
confirmed: the placement note's Risks section items 1-5 map one-to-one onto
AC-001 (Risk 1), AC-010 (Risk 2), AC-009 (Risk 3), the AC-011 diagnostic
(Risk 4), and the Forward Obligation (Risk 5) — all four risk-bearing ACs
exist in the story body exactly where claimed. No registry-level ASM/R gap.

## Cross-Reference Validation

### ID Consistency

| Check | Status | Issues |
|---|---|---|
| BC IDs unique | pass | — |
| VP IDs unique | pass | — |
| Story AC IDs unique | pass | AC-001 through AC-017, no duplicates or gaps |
| BC traces to valid CAP | pass | BC-2.01.001/002 → CAP-001; BC-2.02.001/002 → CAP-005; BC-2.02.005 → CAP-008 (all confirmed in each BC's own frontmatter `traces_to`) |
| VP traces to valid BC | pass | VP-042 `source_bc: BC-2.01.001` confirmed present and correct |
| Story ACs trace to valid BC | pass | every AC's "(traces to BC-...)" annotation resolves to a real anchored BC |

### Naming Convention Compliance

| Convention | Expected Pattern | Violations |
|---|---|---|
| BC naming | BC-S.SS.NNN | none |
| VP naming | VP-NNN | none |
| AC naming | AC-NNN | none |
| Subsystem naming (story `subsystems:` frontmatter) | exact Name from ARCH-INDEX Subsystem Registry (MUST rule) | **`transport-layer` — not a valid registry name (Finding 1, HIGH)** |

### Canonical Frontmatter Validation

| Artifact | document_type | level | version | producer | traces_to | Status |
|---|---|---|---|---|---|---|
| stories/S-BL.LOOPBACK-FULLSTACK.md | present | present | present | present | present | pass |
| decisions/S-BL.LOOPBACK-FULLSTACK-placement-note.md | present | N/A (architect-design-note doesn't carry `level`) | present | present | N/A (`related_documents:` used instead of `traces_to:`) | pass (note-type frontmatter is internally consistent with its own type's convention, used identically by sibling placement notes in this project) |
| BC-2.01.001.md / BC-2.01.002.md / BC-2.02.001.md / BC-2.02.002.md / BC-2.02.005.md | present | present (L3) | present | present | present | pass (all 5) |
| specs/verification-properties/VP-042.md | present | present (L4) | present | present | present | pass |

## Spec vs Implementation Drift

| Artifact | Spec Version | Implementation State | Drift Detected | Notes |
|---|---|---|---|---|
| `internal/testenv/testenv.go` (`NewLoopback`, `LoopbackConfig`, `LoopbackEnv`) | story v1.17 describes current pre-implementation state | matches exactly — `NewLoopback` still discards `cfg` and calls `newEnv(ctx, b, 1)` at `develop@2ce3a57`, confirmed byte-for-byte against the story's Context section claim | no | This story has not yet been implemented (draft/unscheduled); the "drift" check here confirms the story's *baseline description* of current code is accurate, which it is |
| `internal/bench/keystroke_echo_testenv_bench_test.go` | story cites PR #121 @ `4c276d9`, `//go:build integration` tag, `BenchmarkKeystrokeToEcho_P99` | confirmed present, tag confirmed, function confirmed at line 65, `env := lb.Env` pattern confirmed at line 79 | no | — |
| `cmd/switchboard/access.go` (`startSweepTicker`/`startFramesDroppedTicker` line citations) | story/note cite `:460`/`:500` | actual: call sites at 446/454, definitions at 595/635 | **yes (Finding 3, LOW)** | Non-blocking — illustrative analogy citation only, not load-bearing for any AC or task |
| `internal/session/upstream.go` (`SendKeystroke`, console-attachment gate, `sessionName` comparison) | story cites `:276-301`, `:288-290`, `:292` | all three confirmed exact | no | — |
| `internal/arq/arq.go` (`OnAck`, `EnqueueSend`, `payloadFor`, `SACKBitmapBytes`) | story cites `:201`, `:339`, `:291`, `SACKBitmapBytes=8` | all confirmed exact | no | — |
| `internal/multipath/multipath.go` (`Frame` struct, `Send`'s masking-error mechanics, `DefaultDropCacheSize`) | story's `Frame` has-no-`ChanSeq` claim and `sent>=1→nil` claim | both confirmed true on disk | no | Load-bearing for the entire v1.17/R21 repair — independently verified true |
| `internal/halfchannel/halfchannel.go` (`MinTickInterval`/`MaxTickInterval`, `ChannelFrame.ChanSeq`) | story cites 5ms/50ms bounds, `ChannelFrame` has `ChanSeq` | both confirmed exact | no | — |
| `internal/paths/paths.go` (`NewPathTracker` active-by-default) | story cites `active: true` at construction | confirmed exact | no | — |
| ARCH-08 §6.5 pos-23 import set | story cites v2.13, DRAFT/PROSPECTIVE | confirmed current/latest version, status matches | no | — |
| ARCH-03 downstream-ARQ delivery contract | story/note cite `OnAck` returns `[][]byte` synchronously, no internal channel | confirmed matches `arq.go:201`'s actual signature | no | — |

## §1.7 Perimeter Audit — Full Dimensional Findings

The checks above map the §1.7 audit onto the canonical template's sections.
This section carries the full detail for the two genuine perimeter defects
and one non-blocking drift item found, organized by the five audit
dimensions specified in the dispatch.

### Dimension 1 — Anchor Completeness & Correctness

All 5 BC↔AC semantic mappings verified correct (§1 above). No AC claims a
BC anchor the BC does not support. No missing BC/VP anchor found. **The one
anchor gap is one level up: the story's `subsystems:` frontmatter — see
Finding 1.**

Independently re-verified, load-bearing technical claims underneath the BC
anchors (all confirmed TRUE against `develop@2ce3a57`): `multipath.Frame` has
no `ChanSeq` field (the entire basis for the v1.17/R21 repair);
`multipath.Send` returns `nil` whenever `sent>=1` (the entire basis for the
R18/AC-017 in-place-`failLoud` requirement); no production code calls
`arq.OnAck` today, and `internal/arqsend` exercises only the sender-side
subset (the entire basis for AC-001's risk framing); `ARQ.OnAck`'s
`payloadFor`/`inFlight` are strictly instance-local (the entire basis for
the single-shared-`*arq.ARQ`-instance requirement).

**Finding 1 (HIGH) — `subsystems:` frontmatter anchor is semantically wrong
against ARCH-INDEX's authoritative Subsystem Registry.**

Story frontmatter: `subsystems: [transport-layer, quality-observability,
session-networking]`.

ARCH-INDEX's Subsystem Registry carries an explicit MUST rule: "BC
frontmatter `subsystem:`, BC-INDEX subsystem column, story `subsystems:`
fields, and PRD subsystem references MUST all use the exact Name from this
table." The registry (SS-01 through SS-09) contains exactly: session-
networking, multipath-forwarding, session-discovery, session-access,
admission-security, quality-observability, network-management, console-
operations, deployment-operations.

`transport-layer` is not in this list — a direct MUST-rule violation,
disk-verified.

More significantly, the story is **missing `multipath-forwarding` (SS-02)**
— the subsystem that owns most of what it builds: SS-02's registry row lists
`internal/multipath, internal/arq, internal/replay, internal/paths,
internal/routing`, and 3 of this story's 5 `architecture_modules:`
(`internal/arq`, `internal/multipath`, `internal/paths`) are SS-02-owned; 3
of the story's 5 anchored BCs (BC-2.02.001, BC-2.02.002, BC-2.02.005)
independently carry `subsystem: multipath-forwarding` in their own
frontmatter (confirmed by direct grep of all 5 BC files). `quality-
observability` (SS-06, owning `internal/metrics`/`internal/mgmt`) is present
in the story's list but has no `architecture_modules:` support — its only
basis is the narrative line in BC-2.01.002's Description ("The quality
observability subsystem depends on this invariant..."), a materially weaker
basis than a direct `subsystem:` frontmatter match.

**Not unique to this story — systemic across the S-BL.* test-infrastructure
backlog:**

| Story | `subsystems:` frontmatter |
|---|---|
| S-BL.BENCH | `[transport-layer]` |
| S-BL.LOOPBACK-FULLSTACK | `[transport-layer, quality-observability, session-networking]` |
| S-BL.TESTENV | `[transport-layer, network-management, deployment-operations, session-management]` |
| S-BL.PE-RECEIVE-LOOP | `[deployment-operations, transport-layer]` |

`S-BL.TESTENV` additionally uses `session-management`, which also does not
appear in the registry. Root cause: `internal/testenv` and `internal/bench`
— the actual modules all four stories build — appear in **no** subsystem's
"Implementing Modules" column anywhere in the registry, so four independent
authoring passes each reached for the same informal, unregistered
`transport-layer` label rather than either citing the real subsystems whose
production code the harness drives, or having the registry formally
acknowledge a home for cross-cutting test infrastructure.

**Disposition:** genuine perimeter defect, not correct-as-designed. Per this
audit's own charter and the project's semantic-anchoring discipline (ARCH-
INDEX's MUST rule), this should be corrected before the story is treated as
fully converged. Because the pattern is systemic, the durable fix is
architectural (register a subsystem/row for cross-cutting test
infrastructure, or standardize on registry-real names everywhere). For this
story specifically, the minimal correct fix is: `subsystems: [session-
networking, multipath-forwarding]` — drop `transport-layer`; add
`multipath-forwarding`; drop `quality-observability` (no
`architecture_modules:` support).

### Dimension 2 — Scope Boundary Correctness

No gaps found. Non-Goals (harness-only VP-042 delivery/lock deferred,
upstream-ARQ out of scope, empty-tick non-wire-dispatch, loss/retransmit
out of scope) all independently cross-checked against VP-042.md's own
Property Statement/Source Contract/Feasibility Assessment and against
ARCH-03's architecture (upstream self-healing via `internal/replay`,
downstream-only ARQ) — no conflict found in either direction. `internal/arq`
and `internal/multipath`'s own correctness properties are correctly treated
as already covered by their own pure-core unit test suites
(`internal/arq/arq_test.go`, `internal/multipath/multipath_test.go`, both
confirmed to exist and be substantial), not this harness's job to re-prove.

### Dimension 3 — Cross-Document Coherence

Story ↔ note parity confirmed complete through v1.15 (note frozen there since
v1.15; story's two subsequent versions, v1.16/v1.17, are story-body-only
ledger/citation repairs requiring no further note change, per the story's own
changelog cross-references "Consistent with placement-note vX.Y" at every
step). ARCH-08 v2.13 confirmed current/latest and correctly still
DRAFT/PROSPECTIVE pending this story's delivery. ARCH-08 position numbers
(halfchannel=5, arq=9, paths=11, multipath=14, all below testenv=23)
independently re-derived and confirmed. ARCH-03's `OnAck` delivery contract
independently re-derived against `arq.go`'s actual signature and confirmed to
match. STORY-INDEX's row correctly reflects R22/R23/R24 CLEAN, convergence
3/3, and the "pending §1.7 audit" state this dispatch is running inside of.
No broken artifact references found anywhere in the chain.

**Finding 2 (MEDIUM, non-blocking) — VP-042.md's own Source Contract (2 BCs)
is narrower than the story's actual 5-BC anchor set; never closed on
VP-042.md's side.** Full detail in §2 above. The extra 3 BCs
(BC-2.01.002, BC-2.02.002, BC-2.02.005) are legitimate harness-construction
machinery per the placement note's Q1 reasoning (you cannot build a
protocol-accurate downstream leg without exercising ARQ/dedup), not a
mistake — but VP-042.md is never updated to disclose this broader
dependency, and the asymmetry is corroborated by the fact that only the same
2 BCs VP-042.md names carry an explicit VP-042 back-reference in their own
files; the other 3 mention VP-042 nowhere. **Recommendation:** bundle a
VP-042.md Source Contract update with the already-scheduled Forward
Obligation (updating VP-042.md's Proof Harness Skeleton) — both are
downstream PO/architect acts on the same document at the same future
lifecycle point. Not a blocker for this story's own convergence.

### Dimension 4 — Convention Consistency

BC-S.SS.NNN / VP-NNN / AC-NNN / S-BL.XXX naming used consistently throughout,
no drift found. Canonical frontmatter fields present and well-formed across
the story, note, and all 5 BC files. The `subsystems:` convention violation
is Finding 1 (above) — a convention-consistency defect as much as an
anchor-correctness one, since it is systemic across 4 sibling stories, not a
one-off typo.

Observation (not a finding against this story): BC-2.01.001.md carries both
a "Verification Properties" table and separate "VP Anchors"/"Architecture
Anchors" sections; its sibling BC-2.01.002.md (same subsystem, same
originating pass) has neither of the latter two, ending at "Related BCs." A
pre-existing template-completeness gap in BC-2.01.002.md itself, out of this
story's remediation scope — noted only because it is part of why Finding 2's
asymmetry is not visible from reading BC-2.01.002.md alone.

**Finding 3 (LOW, non-blocking) — stale `cmd/switchboard/access.go:460`/
`:500` analogy citations.** Full detail in the Spec vs Implementation Drift
table above. Illustrative citation only; does not gate any AC or task; the
underlying claim (this design's ticker idiom mirrors an existing production
pattern) remains true, only the specific line numbers have drifted since
`access.go` last changed shape. Correct in a future patch cycle.

### Dimension 5 — Coverage-Gap Surfacing

18 distinct symbols/line-citations the story's design depends on were
independently re-derived against `develop@2ce3a57` (full table in "Spec vs
Implementation Drift" above) — every one confirmed present, correctly typed/
signed, and (where a specific line number was cited) exact, with the single
exception of Finding 3. No assumed-but-absent dependency was found.

## Findings

### Critical

None. Finding 1 is rated HIGH rather than CRITICAL/BLOCKER because it is a
metadata/traceability defect — it does not affect any AC's testability, any
code-level correctness claim, or the technical soundness of the design (all
independently re-verified true) — but per this audit's charter it is still
treated as a genuine blocking finding for full convergence, not a deferrable
observation.

### Major

- **Finding 1 (HIGH)** — `subsystems:` frontmatter anchor (`transport-layer`
  invalid per ARCH-INDEX registry; `multipath-forwarding` missing despite
  owning 3 of 5 anchored BCs and 3 of 5 `architecture_modules:`). Systemic
  across 4 sibling S-BL.* stories. **Recommend fixing before this story is
  treated as fully converged** — minimal fix: `subsystems: [session-
  networking, multipath-forwarding]`.
- **Finding 2 (MEDIUM)** — VP-042.md's Source Contract (2 BCs) narrower than
  the story's actual 5-BC anchor set; asymmetry never closed on VP-042.md's
  side. Not a blocker for this story; bundle with the already-scheduled
  VP-042.md Forward Obligation.

### Minor

- **Finding 3 (LOW)** — stale `cmd/switchboard/access.go:460`/`:500` analogy
  citations (actual: 446/454 call sites, 595/635 definitions). Non-blocking,
  illustrative only.
- **Observation** — BC-2.01.002.md lacks the "VP Anchors"/"Architecture
  Anchors" sections its sibling BC-2.01.001.md has. Pre-existing, out of this
  story's scope.

## Validation Gate Result

**FAIL** — blocking finding: Finding 1 (`subsystems:` frontmatter anchor
invalid against ARCH-INDEX's Subsystem Registry MUST rule; missing
`multipath-forwarding`). This is a metadata-only fix (one frontmatter line),
not a design or implementation defect — everything else audited across all
five dimensions, including every load-bearing code-level claim underneath
the story's design (18 symbols/citations independently re-derived against
`develop@2ce3a57`), checked out clean. Findings 2 and 3 are explicitly
non-blocking and may be carried forward/bundled with already-scheduled
downstream work.

## Overall Metrics

| Metric | Value |
|--------|-------|
| **Total Checks** | 5 audit dimensions × (BC↔AC semantic mapping ×5, code-symbol re-derivation ×18, cross-document coherence ×6 documents, scope-boundary cross-check ×5 Non-Goals items, subsystem-registry cross-check ×3 declared subsystems) = 37 discrete checks |
| **Passed** | 35 |
| **Failed** | 1 (subsystems anchor — Finding 1) |
| **Warnings** | 2 (Finding 2, Finding 3 — both explicitly non-blocking) |
| **Overall Status** | inconsistencies-found |

This story's spec package is exceptionally thorough and internally
consistent — 24 rounds of adversarial reconvergence produced a design whose
every load-bearing technical claim (checked exhaustively here, symbol by
symbol, line number by line number, against the actual `develop@2ce3a57`
code tree) is accurate. The gaps this audit found are, appropriately, at the
perimeter the internal adversarial process was never scoped to check: a
subsystem-registry anchor one level above the BC/VP chain, and a VP↔BC
trace-set asymmetry only visible by reading VP-042.md side-by-side with the
story rather than reading the story alone. **Recommended action:** fix
Finding 1 (single frontmatter line) before advancing past the human approval
gate; carry Findings 2 and 3 forward as already-scheduled/low-priority
follow-up.

## Appendix: Validation Methodology

This audit was dispatched as a §1.7 fresh-context consistency audit, run
after Step-4.5 adversarial reconvergence reached 3/3 (R22/R23/R24 CLEAN),
per POL-005 (`adversary-dispatch-integrity`). Its charter, distinct from the
adversarial lenses that precede it, is to check the PERIMETER — whether the
story's declared anchor set correctly and completely covers what the story
must, and whether the whole spec package is mutually coherent — not to
re-check defects within the perimeter the adversarial lenses were already
shown.

Method: (1) verified the POL-005 dispatch-integrity tuple (`.factory` HEAD
sha, story/note/STORY-INDEX versions) before any content review; (2) read the
full story (1416 lines), the full placement note's design sections (Q1-Q8,
~900 lines directly relevant to design content, plus the full formal
changelog through v1.15 confirming story/note parity), and all 5 anchored BC
files, VP-042.md, and the relevant ARCH-08/ARCH-03/ARCH-INDEX sections in
full; (3) independently re-derived every load-bearing technical claim and
every cited file:line anchor against the actual `internal/*` and
`cmd/switchboard/*` source at `develop@2ce3a57` via targeted `grep`/`sed`,
rather than trusting the spec's own citations; (4) cross-checked the story's
`subsystems:` frontmatter against ARCH-INDEX's Subsystem Registry (the
project's own declared "source of truth" for that field) and against the
`subsystem:` field of each of the 5 anchored BC files directly; (5)
cross-checked VP-042.md's own Source Contract against the story's broader
anchor set and against each anchored BC's own VP back-reference (or absence
thereof); (6) cross-checked the story's Non-Goals against VP-042.md's
Property Statement/Feasibility Assessment and against ARCH-03's architecture
to confirm no scope-boundary conflict. No source artifact was modified;
this report is the sole output.
