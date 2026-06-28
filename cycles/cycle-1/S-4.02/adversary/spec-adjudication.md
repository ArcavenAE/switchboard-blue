---
document_type: spec-ruling
title: "S-4.02 Adversarial Review — Spec Adjudication"
ruling_id: RULING-002
date: 2026-06-28
author: product-owner
stories_affected: [S-4.02]
findings_adjudicated: [S-4.02-ADV-F1, S-4.02-ADV-F2, S-4.02-ADV-F3]
specs_modified:
  - .factory/specs/behavioral-contracts/ss-02/BC-2.02.004.md
handoff_required:
  - story-writer: apply AC-004 replacement text, vp_traces change, AC trace-label corrections
  - implementer/test-writer: cite BC-2.02.004 invariant 5 for bounded-state assertions
---

# RULING-002: S-4.02 Spec Adjudication — VP-042 Scoping, Bounded-State Invariant, AC Trace Corrections

## Context

Three spec-level findings emerged from a fresh-context adversarial review of story S-4.02
(internal/replay). All three are adjudicated here. This ruling amends BC-2.02.004 (v1.2 →
v1.3) and prescribes exact story body / frontmatter edits for story-writer to apply.
Code and test files are NOT touched by this ruling.

---

## FINDING 1 — False VP-042 Gate / Mis-traced Verification Property (CRITICAL)

### Finding Summary

AC-004 names `BenchmarkReplay_KeystrokeLatency` and `TestReplay_VP042_KeystrokeLatencyP99`
as "the VP-042 gate." The story `vp_traces` frontmatter lists `VP-042`. However, VP-042's
authoritative module is `internal/halfchannel` and measures the full keystroke-to-echo
**round-trip** latency across 500 trips on a loopback stack at 10ms/50ms tick intervals
(BC-2.01.001 / BC-2.02.001, NFR-001). A pure-core `internal/replay` unit benchmark that
measures the time for a single `OnUpstream` map-insert call cannot verify that property.
The 100ms gate has approximately 5 orders of magnitude headroom over any plausible sub-µs
map insert, making the gate structurally unfailable — a false-green gate that gives
unwarranted confidence in the 100ms SLO.

### Decision

**R1-F1-A: VP-042 stays exclusively in `internal/halfchannel`.**

VP-042 is an integration-level round-trip property that requires a full loopback stack
(testenv.NewLoopback, upstream + downstream half-channels, echo detection). It cannot be
meaningfully verified by a unit benchmark of an isolated replay state machine. VP-042 is
removed from S-4.02's scope.

**R1-F1-B: AC-004 is replaced with an honest micro-latency regression guard.**

`internal/replay.OnUpstream` is a pure-core state machine with no I/O. What it can
guarantee is that its per-call overhead is negligible relative to the tick interval
(10ms). The replacement AC uses a falsifiable, honest threshold tied to the tick budget,
not to the end-to-end NFR.

### Prescribed Changes for Story-Writer

#### 1. `vp_traces` frontmatter change

Remove `VP-042` from the `vp_traces` array. The array becomes:

```yaml
vp_traces: [VP-022, VP-023]
```

Add an explanatory comment in the story body (Implementation Notes or a new "VP
Scoping Note" subsection) stating:

> VP-042 (keystroke-to-echo p99 ≤ 100ms) is an integration property verified at the
> `internal/halfchannel` wave-gate benchmark, not here. S-4.02 verifies VP-022 and
> VP-023 only (no-duplicate-delivery and in-order-delivery property tests).

#### 2. AC-004 replacement text (EXACT)

Replace AC-004 in the story body with:

```markdown
### AC-004 (traces to BC-2.02.004 invariant 5 — bounded-state / micro-latency regression guard)
`Replay.OnUpstream` per-call overhead is ≤ 1µs (median) under no-contention conditions
with windowSize=64 and a pre-warmed window, as measured by a Go micro-benchmark over
10,000 iterations. This guards against inadvertent O(N²) or allocation-heavy regressions
in the replay state machine. This is NOT the VP-042 NFR gate (which is verified at the
internal/halfchannel integration level).
- **Test:** `BenchmarkReplay_OnUpstream_PerCall` (replaces `BenchmarkReplay_KeystrokeLatency`)
```

**Rationale for the 1µs threshold:** At a 10ms tick interval, the replay state machine
consumes at most 0.01% of the tick budget per call at 1µs. This is falsifiable (a
naive O(N) scan over the window would exceed it) while not being unreachably tight.
The threshold is a regression guard, not a production SLO.

**Rationale for removing `TestReplay_VP042_KeystrokeLatencyP99`:** That test name
embeds a false claim that it is verifying VP-042. The replacement benchmark name
`BenchmarkReplay_OnUpstream_PerCall` is honest about what it measures.

---

## FINDING 2 — Fabricated Bounded-State Traceability (HIGH)

### Finding Summary

The replay test suite asserts `pending ≤ windowSize-1` and `seen ≤ windowSize` and
anchors those assertions to "BC-2.02.004 invariant 3 / PC5 (bounded memory / DoS
resistance)." Neither anchor is correct. BC-2.02.004 invariant 3 states "The replay
window size N is fixed for the lifetime of a channel." PC5 states "Loss of N+1
consecutive frames results in a gap that is irrecoverable without retransmit." Neither
clause says anything about bounded receiver memory. The bounded-state assertions were
correct behavior being tested, but they had no spec authority.

### Decision

**R2: Add Invariant 5 to BC-2.02.004 (bounded-state / DoS-resistance).**

BC-2.02.004 is bumped from v1.2 to v1.3. Invariant 5 is added (see spec file for
exact text). The canonical form is:

> **5. Bounded receiver state (DoS-resistance)** (RULING-002): The replay receiver
> retains at most O(windowSize) entries across its pending (buffered, awaiting
> in-order delivery) and seen (dedup) sets combined. Formally:
> |pending| + |seen| ≤ 2 × windowSize at all times. No unbounded allocation is
> permitted regardless of the sequence of incoming chan_seq values. Implementations
> MUST enforce this cap by evicting the oldest seen entries once the seen set would
> exceed windowSize entries, and MUST silently discard any incoming frame whose
> distance from the current delivery frontier exceeds windowSize (treating it as
> irrecoverably old, consistent with PC5).

### Canonical Clause Reference for Implementer / Test-Writer

Tests asserting bounded-state MUST cite:

```
BC-2.02.004 invariant 5 (bounded receiver state / DoS-resistance) — v1.3
```

The assertion `|pending| + |seen| ≤ 2 × windowSize` is the normative bound.
The assertion `pending ≤ windowSize-1` (when the pending buffer holds at most
the N-1 frames ahead of the next expected seq) and `seen ≤ windowSize` are both
stricter and conformant sub-cases of invariant 5.

### Spec Files Modified

- `.factory/specs/behavioral-contracts/ss-02/BC-2.02.004.md` — v1.2 → v1.3,
  Invariant 5 added, `modified` changelog entry added.

---

## FINDING 3 — AC-003 Mis-Anchor (corroborated, HIGH)

### Finding Summary

Story AC-003 traces to "BC-2.02.004 invariant 1." Invariant 1 is:

> **DI-001**: The replay window contains keystroke content which is SSH-encrypted
> end-to-end; it is opaque to routers.

The AC-003 behavior — "The replay window carries the last N keystrokes; frames older
than the window are discarded without error" — has no relationship to SSH-encryption
opacity. The correct anchors are:

- **Invariant 3** — "The replay window size N is fixed for the lifetime of a channel"
  (establishes that N is the configurable bound)
- **Postcondition 1** — "Each upstream frame's payload includes the current keystroke(s)
  plus the last N-1 keystrokes (the replay window)" (establishes the window-carry
  semantics)

The adversary also flagged a broader AC↔BC-postcondition mis-map pattern for AC-001,
AC-002, and AC-004. Review findings below.

### Decision: Corrected AC Trace Labels

#### AC-001

**Current (wrong):** `traces to BC-2.02.004 postcondition 1`

**Assessment:** The behavior being tested is "`OnUpstream` never delivers the same
sequence number twice; second delivery returns `ErrAlreadyDelivered`."

**Correct anchor:** BC-2.02.004 postcondition 2 — "The access node deduplicates
keystrokes by sequence number: each keystroke is applied exactly once to the tmux
session." PC2 is the dedup-exactly-once guarantee. PC1 is about frame payload content
(carries current + last N-1), which is not what AC-001 tests.

**Prescribed replacement:**

```markdown
### AC-001 (traces to BC-2.02.004 postcondition 2)
```

#### AC-002

**Current:** `traces to BC-2.02.004 postcondition 2`

**Assessment:** The behavior being tested is "delivers keystrokes in sequence order;
out-of-order buffered."

**Correct anchor:** This is not directly postcondition 2 (dedup). The closest existing
postcondition is PC2 (exactly-once), but the out-of-order buffering and in-sequence
delivery is more precisely the behavior VP-023 formalizes: monotonically non-decreasing
delivery order. Within BC-2.02.004, there is no explicit postcondition for delivery
ordering — this is captured via VP-023 (which traces from the invariant "Keystroke
sequence numbers are monotonically increasing within a channel; duplicates are
discarded" — invariant 2). AC-002 traces to **invariant 2** more accurately.

**Prescribed replacement:**

```markdown
### AC-002 (traces to BC-2.02.004 invariant 2)
```

#### AC-003

**Current (wrong):** `traces to BC-2.02.004 invariant 1`

**Correct anchor:** BC-2.02.004 invariant 3 + postcondition 1 (dual anchor).

**Prescribed replacement:**

```markdown
### AC-003 (traces to BC-2.02.004 invariant 3 + postcondition 1)
```

#### AC-004

Superseded by Finding 1. See Finding 1 prescribed replacement above.

**Prescribed replacement:**

```markdown
### AC-004 (traces to BC-2.02.004 invariant 5 — bounded-state / micro-latency regression guard)
```

---

## Summary Table: Prescribed Story-Writer Edits

| Item | Current | Prescribed Replacement | Finding |
|------|---------|----------------------|---------|
| `vp_traces` frontmatter | `[VP-022, VP-023, VP-042]` | `[VP-022, VP-023]` | F1 |
| AC-004 text | keystroke-to-echo p99 ≤ 100ms, VP-042 gate | per-call OnUpstream ≤ 1µs median (regression guard, not VP-042) | F1 |
| AC-004 test name | `BenchmarkReplay_KeystrokeLatency` | `BenchmarkReplay_OnUpstream_PerCall` | F1 |
| AC-001 trace label | `postcondition 1` | `postcondition 2` | F3 |
| AC-002 trace label | `postcondition 2` | `invariant 2` | F3 |
| AC-003 trace label | `invariant 1` | `invariant 3 + postcondition 1` | F3 |
| AC-004 trace label | `postcondition 3` | `invariant 5 — bounded-state / micro-latency regression guard` | F1+F2 |

## Summary Table: Spec Changes

| File | Change | Finding |
|------|--------|---------|
| BC-2.02.004.md | v1.2 → v1.3; Invariant 5 added (bounded receiver state / DoS-resistance) | F2 |

## Handoff Notes

**For story-writer (bc_array_changes_propagate_to_body_and_acs policy):**
Apply all prescribed changes in the "Summary Table: Prescribed Story-Writer Edits" table
to story S-4.02's body content and frontmatter. Do NOT change BC files (those are owned
by product-owner and already applied). The `vp_traces` frontmatter change (`VP-042`
removal) is a frontmatter edit; the AC text and trace-label changes are body edits.

**For implementer/test-writer:**
- Bounded-state assertions cite: `BC-2.02.004 invariant 5 (RULING-002, v1.3)`
- The VP-042 benchmark is NOT implemented in `internal/replay`. Do not write
  `BenchmarkReplay_KeystrokeLatency` or any test claiming to verify VP-042 in this
  package. VP-042 is verified at the `internal/halfchannel` integration level.
- Implement `BenchmarkReplay_OnUpstream_PerCall` instead: 10,000 iterations,
  pre-warmed window, median per-call latency assertion ≤ 1µs.

**VP citation changes:**
VP-042's `inputDocuments` and `module` fields already correctly point to
`internal/halfchannel`. No VP file edits are required by this ruling.

---

## RULING-002 Amendment 1 — AC-003 Anchor Correction (PC1 is Sender-Side; Correct Receiver Anchor is Invariant 5)

**Date:** 2026-06-28
**Author:** product-owner
**Supersedes:** RULING-002 Finding 3, AC-003 prescribed replacement

### Problem

RULING-002 Finding 3 prescribed AC-003 trace to "BC-2.02.004 invariant 3 +
postcondition 1." This introduced a new mis-anchor. Postcondition 1 states:

> "Each upstream frame's **payload includes** the current keystroke(s) plus the
> last N-1 keystrokes (the replay window)."

PC1 is a **sender-side payload-assembly postcondition**. It is implemented by
`internal/halfchannel`'s `dequeueUpstream(replay_window)` call (ARCH-03
§Upstream Idempotent Replay, Half-Channel Architecture pseudocode). The
`internal/replay` module is a **pure receiver**: it processes `chan_seq` values
for deduplication and in-order delivery. It never assembles, inspects, or
validates frame payload content. Anchoring a receiver AC to a sender
postcondition is a category error.

The named test `TestReplay_WindowBoundary` cannot and does not verify PC1.
`TestReplay_WindowBoundary` tests the receiver-side discard behavior when a
frame's `chan_seq` lies outside the delivery frontier window — a behavior that
PC1 says nothing about.

### Analysis of AC-003's Two Testable Clauses

AC-003 wording (post-RULING-002, current story v1.1):

> "The replay window carries the last N keystrokes (where N is configured).
> Frames older than the window are discarded without error."

| Clause | Receiver-Verifiable? | Correct Anchor |
|--------|---------------------|----------------|
| "replay window carries the last N keystrokes (where N is configured)" | Yes — N is a fixed, configurable window parameter. The receiver uses N as its window bound. | BC-2.02.004 **invariant 3** ("The replay window size N is fixed for the lifetime of a channel") |
| "frames older than the window are discarded without error" | Yes — this is the receiver-side discard behavior when a frame's distance from the delivery frontier exceeds windowSize. | BC-2.02.004 **invariant 5** ("MUST silently discard any incoming frame whose distance from the current delivery frontier exceeds windowSize", RULING-002) |

PC1 is irrelevant to both clauses as a receiver-side anchor. The correct
dual-anchor is **invariant 3 + invariant 5**.

### Decision

**R1-A1: AC-003 trace label corrected from "invariant 3 + postcondition 1" to
"invariant 3 + invariant 5".**

No AC-003 body wording change is required. The existing wording ("carries the
last N keystrokes… frames older than the window are discarded without error")
accurately describes receiver-side behavior. Only the trace label is wrong.

No BC-2.02.004 spec changes are required. Invariant 3 and invariant 5 (added
by RULING-002/Finding 2) already exist at v1.3 with the correct semantics.

### Prescribed Change for Story-Writer

Replace the AC-003 trace label in S-4.02's story body:

**Current (wrong, per RULING-002 Finding 3):**
```markdown
### AC-003 (traces to BC-2.02.004 invariant 3 + postcondition 1)
```

**Correct (this amendment):**
```markdown
### AC-003 (traces to BC-2.02.004 invariant 3 + invariant 5)
```

No other AC-003 content changes.

### Canonical Test-Writer Docstring Citations (All Four ACs)

These are the authoritative citation strings test-writer MUST embed in test
function doc comments. They supersede any citation strings implied by RULING-002
Finding 3 for AC-003.

#### AC-001 → `TestReplay_NoDuplicateDelivery`
```
// Verifies: BC-2.02.004 postcondition 2 (dedup exactly-once: each keystroke
// applied exactly once to the tmux session; second delivery of same chan_seq
// returns ErrAlreadyDelivered). RULING-002 Finding 3.
```

#### AC-002 → `TestReplay_InOrderDelivery`
```
// Verifies: BC-2.02.004 invariant 2 (chan_seq monotonically increasing within
// a channel; out-of-order frames buffered and delivered in sequence order).
// RULING-002 Finding 3.
```

#### AC-003 → `TestReplay_WindowBoundary`
```
// Verifies: BC-2.02.004 invariant 3 (replay window size N is fixed for the
// lifetime of a channel; N is the configurable bound) + invariant 5 (receiver
// silently discards any frame whose distance from delivery frontier exceeds
// windowSize; no error returned to caller). RULING-002 Amendment 1.
// NOTE: postcondition 1 (payload carries last N-1 keystrokes) is sender-side;
// it is verified at internal/halfchannel, NOT here.
```

#### AC-004 → `BenchmarkReplay_OnUpstream_PerCall`
```
// Verifies: BC-2.02.004 invariant 5 (bounded receiver state / micro-latency
// regression guard). Per-call OnUpstream overhead ≤ 1µs median, 10,000
// iterations, windowSize=64, pre-warmed window. NOT the VP-042 NFR gate.
// RULING-002 Finding 1+2.
```

### Updated Summary Table (AC Trace Labels — Authoritative Post-Amendment)

| AC | Test | Trace Label | Finding |
|----|------|-------------|---------|
| AC-001 | `TestReplay_NoDuplicateDelivery` | BC-2.02.004 **postcondition 2** | RULING-002 F3 |
| AC-002 | `TestReplay_InOrderDelivery` | BC-2.02.004 **invariant 2** | RULING-002 F3 |
| AC-003 | `TestReplay_WindowBoundary` | BC-2.02.004 **invariant 3 + invariant 5** | **This amendment** |
| AC-004 | `BenchmarkReplay_OnUpstream_PerCall` | BC-2.02.004 **invariant 5** (bounded-state / micro-latency guard) | RULING-002 F1+F2 |

### Sanity-Check: AC-001, AC-002, AC-004

The fresh-context adversarial pass independently corroborated these anchors from
RULING-002. This amendment confirms they are correct receiver-side anchors:

- **AC-001 → PC2**: "The access node deduplicates keystrokes by sequence number:
  each keystroke is applied exactly once." PC2 is unambiguously receiver-side
  dedup semantics. `TestReplay_NoDuplicateDelivery` directly verifies the
  "second delivery returns ErrAlreadyDelivered" behavior. Correct.

- **AC-002 → invariant 2**: "Keystroke sequence numbers are monotonically
  increasing within a channel; duplicates are discarded." Invariant 2 is the
  monotonic-ordering invariant that AC-002's out-of-order buffering test
  enforces. (PC2 only covers dedup; ordering is invariant 2 / VP-023.) Correct.

- **AC-004 → invariant 5**: Invariant 5 (bounded receiver state) was added by
  RULING-002/F2 precisely to give the bounded-state and micro-latency regression
  guard a spec authority. The 1µs median threshold is a regression guard for the
  state-machine's O(windowSize) operations. Correct.
