---
artifact_id: S-BL.PE-RECEIVE-LOOP-disposition-ruling
document_type: product-owner-ruling
version: "1.0"
status: final
producer: product-owner
timestamp: 2026-07-08T00:00:00Z
story_subject: S-BL.PE-RECEIVE-LOOP
ruling_questions: [Q-A, Q-B]
Q_A_decision: "(a) documentation-artifact reading — narrow anchor; BC-2.06.003 trace is prerequisite documentation only"
Q_B_decision: "single story — scope unchanged at 5 points"
---

# S-BL.PE-RECEIVE-LOOP Disposition Ruling v1.0

## Verified Premises

All factual claims in this ruling are grep-verified against the tree at
`8eb54a5` (S-7.04-FU-PE-CONNECTOR merge SHA). Sources are cited below.

| Premise | File | Evidence |
|---------|------|----------|
| `status: "failed"` is emitted by `metrics.PathEntryFromSnapshot` when `snap.Failed == true` | `internal/metrics/handlers.go:153-154` | `case snap.Failed: status = "failed"` |
| `PathSnapshot.Failed` is a field of `PathSnapshot` | `internal/paths/paths.go:341` | `Failed bool` |
| `PathSnapshot.Failed` is set `true` only inside the consecutive-miss threshold branch after `!t.firstProbe` (keepalive liveness path) | `internal/paths/paths.go:182-188` | `if !t.firstProbe { t.failed = true }` inside `t.consecutiveMisses >= consecutiveMissThreshold` |
| `PathSnapshot.Failed` is NOT written by split-horizon or `OnFrameArrival` | `internal/routing/on_frame_arrival.go` | grep for `Failed` — zero hits in that file |
| E-FWD-001 is emitted by `FrameArrivalHandler.OnFrameArrival` when `ErrAllPathsSplitHorizon` is returned by `SplitHorizon.Forward` | `internal/routing/on_frame_arrival.go:248-252` | `errors.Is(err, ErrAllPathsSplitHorizon)` → `"all paths split-horizon-blocked: frame dropped (... ) (BC-2.02.008 E-FWD-001)"` |
| `ErrAllPathsSplitHorizon` is a topology/forwarding condition, not a liveness condition | `internal/routing/on_frame_arrival.go` | Error indicates arrival interface is the only forwarding candidate — independent of path keepalive state |
| BC-2.06.003 PC-1 `status: "failed"` derivation is `PathSnapshot.Failed == true`, which is set only when `!firstProbe` and consecutive-miss threshold is reached | `BC-2.06.003 v1.16` PC-1 Status derivation block | "Failed > Degraded > Active ... `failed` iff `PathSnapshot.Failed == true` (liveness signal set only when `!firstProbe`)" |
| BC-2.02.008 PC-3/EC-003 is the E-FWD-001 split-horizon drop | `BC-2.02.008 v1.1` | PC-3: "If the only eligible interface is the arrival interface, the frame is dropped and an E-FWD-001 event is logged." EC-003: "Split-horizon drops the only available path → Frame dropped; E-FWD-001 logged." |

---

## Q-A: BC-2.06.003 PC-1 Anchor — What Obligation Does This Story Carry?

### Decision: (a) — Documentation-Artifact Reading

**The BC-2.06.003 PC-1 trace on S-BL.PE-RECEIVE-LOOP is a prerequisite-documentation
trace, not a discharge obligation. This story's binding anchor is BC-2.02.008 PC-3/EC-003
(E-FWD-001) only.**

### Rationale

The two mechanisms are orthogonal by code evidence:

**Mechanism A — E-FWD-001 (split-horizon, BC-2.02.008):**
`FrameArrivalHandler.OnFrameArrival` (`internal/routing/on_frame_arrival.go:248-252`)
emits E-FWD-001 when `ErrAllPathsSplitHorizon` is returned. This is a topology
condition: the arrival interface is the only forwarding candidate. It fires once per
dropped frame. It has no dependency on `PathSnapshot.Failed` or on any keepalive
state. The emission key `"E-FWD-001"` is stable and mutation-pinned by
`TestScanForLine_DetectsEFWD001ProductionEmission`.

**Mechanism B — `status: "failed"` (path liveness, BC-2.06.003 PC-1):**
`metrics.PathEntryFromSnapshot` (`internal/metrics/handlers.go:153-154`) emits
`status: "failed"` when `snap.Failed == true`. `PathSnapshot.Failed` is written in
`internal/paths/paths.go:182-188` only inside the consecutive-miss threshold branch
when `!t.firstProbe` — i.e., a previously-alive path that stopped responding to
keepalive probes. This mechanism is entirely within `internal/paths` and
`internal/metrics`. `FrameArrivalHandler.OnFrameArrival` does not write to any
`PathTracker`, does not touch `PathSnapshot.Failed`, and does not interact with
`internal/paths` at all (zero `paths` package references in `on_frame_arrival.go`).

**Why the stub's BC-2.06.003 PC-1 description is misleading but harmless:**
The stub describes the anchor as "Failed-state via retransmit-driven path exhaustion."
Reading this literally: the `status: "failed"` field in BC-2.06.003 PC-1 is NOT reached
via retransmit-driven E-FWD-001. Sustained split-horizon drops cause E-FWD-001 to fire
repeatedly, but they do not set `PathSnapshot.Failed`. The only path to
`PathSnapshot.Failed == true` is keepalive probe misses accumulating beyond threshold —
which is outside this story's scope entirely.

**Why option (a) is correct and option (b) is not:**
Option (b) would require this story to assert a `status: "failed"` observable via
`sbctl paths list`. But producing that observable requires: (i) an established path
registered with `PathTracker`, (ii) keepalive probes being sent and missed ≥
`consecutiveMissThreshold` times, and (iii) `metrics.PathEntryFromSnapshot` serving
the result. None of this is exercised by the receive goroutine, arqsend wiring, or
split-horizon path-exhaustion integration. Asserting it here would require substantial
scope expansion and would be testing `internal/paths` keepalive liveness — a fully
orthogonal mechanism already shipped by S-BL.PATH-FAILED-STATUS (PR #99, `c098827`).

**Why option (c) is not warranted:**
BC-2.06.003 itself is not ambiguous — v1.16 is precise: `status: "failed"` is emitted
iff `PathSnapshot.Failed == true` (liveness signal, keepalive path). The ambiguity is
in the stub's AC anchor description, not in the BC text. The BC does not need a version
bump.

**What BC-2.06.003 PC-1 legitimately documents on this story:**
The receive goroutine this story ships is a structural prerequisite for future
path-liveness observable testing. Once the receive loop is live, incoming frames from
a PE upstream connection participate in the forwarding table. If a keepalive probe
sequence then fails, `PathSnapshot.Failed` will be set and BC-2.06.003 PC-1 becomes
testable end-to-end. Recording this trace in the story's `bc_traces` is therefore
correct as a prerequisite-documentation trace — it explains why this story matters to
BC-2.06.003 observability infrastructure. It does not create a discharge obligation in
this story.

### Ruling on BC-2.06.003 Trace Disposition

The BC-2.06.003 trace in `bc_traces` frontmatter remains. It is correct as a
prerequisite-documentation record. The anchor table row in the story body description
("To discharge: Failed-state emission observable via full send+forward path") is
inaccurate and must be corrected at elaboration time by the story-writer. The corrected
disposition for that row is:

> **Non-discharging prerequisite trace.** This story ships the receive goroutine that
> makes the full send+forward path live. BC-2.06.003 PC-1 `status: "failed"` (path
> liveness) is NOT discharged here — it requires the keepalive missed-probe mechanism
> (internal/paths), which is orthogonal to E-FWD-001 (split-horizon,
> internal/routing). Future path-liveness observability testing depends on the
> infrastructure this story ships.

### Follow-up Obligations from Q-A

**No new story needs to be minted.** The BC-2.06.003 PC-1 discharge path (keepalive
liveness → `status: "failed"` observable end-to-end via a live PE upstream connection)
is already covered: S-BL.PATH-FAILED-STATUS shipped the `status: "failed"` mechanism
(PR #99, `c098827`). The remaining gap — an integration test asserting `status:
"failed"` specifically after a PE upstream connection drops its keepalives — is a
Wave-7+ observability hardening item. If this discharge obligation needs explicit
scheduling, it should be a new `S-BL.PE-PATH-LIVENESS-OBSERVABLE` stub created at the
time the team confirms it is in scope. It is NOT required for S-BL.PE-RECEIVE-LOOP and
should not be created from this ruling alone.

**Story-writer obligation at elaboration:** correct the Anchors Consumed table row for
BC-2.06.003 PC-1 from "To discharge" to "Non-discharging prerequisite trace" per the
wording above.

---

## Q-B: Delivery Scope — Single Story or Split?

### Decision: Single story, 5 points — no split

**The Q-A ruling does not change the story's delivery scope. S-BL.PE-RECEIVE-LOOP
remains a single 5-point story as sketched.**

### Rationale

Q-A's ruling removes a discharge obligation (BC-2.06.003 PC-1 `status: "failed"`
integration assertion), which would only reduce scope, not increase it. The core
delivery scope is unchanged:

| Scope item | Status after Q-A |
|------------|-----------------|
| Receive goroutine per PE connection (`upstreamdial.Connector`, callback seam) | Required — primary delivery |
| `frame.FrameTypePEConnect = 0x06` definition + `Valid()` update | Required — FO-PE-LOOP-001 discharge |
| `frame.ReadOuterFrame` extraction to `internal/frame` | Required — framing primitive |
| ARCH-08 §6.5 import-set amendment | Required — import-graph governance |
| `arqsend.Retransmitter` test-internal wiring | Required — E-FWD-001 exhaustion integration |
| E-FWD-001 exhaustion integration test (BC-2.02.008 PC-3/EC-003) | Required — binding anchor |
| S404-OBS-F + S404-LOW-1 re-confirmation | Required — drift anchor discharge |
| BC-2.06.003 PC-1 `status: "failed"` integration assertion | REMOVED — non-discharging trace |

The removal of the `status: "failed"` assertion reduces the AC count from the
placement note's upper estimate of 5 (where AC-004 partially covered BC-2.06.003) to a
tighter 4–5 ACs, but this is within the 5-point estimate's variance. No split is
warranted.

---

## Obligation Trace Table

| Obligation | Type | Source | This Story Discharges? | Notes |
|------------|------|--------|----------------------|-------|
| BC-2.02.008 PC-3/EC-003 — E-FWD-001 fires when only eligible interface is arrival interface | BC postcondition / edge case | Re-anchored from S-7.04-FU-PE-CONNECTOR AC-004 v1.3 (F-P1-002) | YES | Integration assertion key: `"E-FWD-001"` in writer output |
| BC-2.06.003 PC-1 — `status: "failed"` via path liveness failure | BC postcondition | Re-anchored from S-7.04-FU-PE-CONNECTOR AC-004 v1.3 (F-P1-002) | NO — prerequisite documentation only | Mechanism is keepalive miss threshold (internal/paths), orthogonal to E-FWD-001 (internal/routing); shipped by S-BL.PATH-FAILED-STATUS |
| FO-PE-LOOP-001 — define `frame.FrameTypePEConnect`; flip `dialLoop` bootstrap | Forward Obligation | S-7.04-FU-PE-CONNECTOR F-P26-001 | YES | `FrameTypePEConnect = 0x06`; `Valid()` upper bound updated |
| S404-OBS-F — E-FWD-001 rate-limit LATENT re-confirmation | Drift anchor | STATE.md; re-anchored from PE-CONNECTOR AC-004 | YES — via E-FWD-001 exhaustion integration test | |
| S404-LOW-1 — live-egress re-confirmation (3 LOW + SEC-001) | Drift anchor | STATE.md; re-anchored from PE-CONNECTOR AC-004 | YES — via full send+forward path traversal | |

---

## Follow-up Obligations Minted

None. No new story stubs are minted by this ruling. If a future wave explicitly schedules
a PE-path liveness end-to-end integration test (asserting `status: "failed"` after
keepalive probes drop on a live PE upstream), that story should be created at that time.

---

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-07-08 | Initial ruling — Q-A: option (a), documentation-artifact reading; BC-2.06.003 PC-1 is non-discharging prerequisite trace; BC-2.02.008 PC-3/EC-003 is the binding anchor. Q-B: single story, 5 points unchanged. All premises grep-verified at `8eb54a5`. |
