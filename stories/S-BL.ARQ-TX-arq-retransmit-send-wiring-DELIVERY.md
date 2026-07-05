---
artifact_id: S-BL.ARQ-TX-DELIVERY
document_type: story-delivery
level: ops
story_id: S-BL.ARQ-TX
version: "1.0"
title: "wire ARQ retransmit-SEND path into router/multipath dispatch (BC-2.02.005 PC-3)"
status: delivered
producer: implementer
timestamp: 2026-07-05T22:15:00Z
modified: 2026-07-05T22:15:00Z
phase: 2
wave: 8
priority: P0
scope_phase: E
estimated_points: 3
delivered_points: 3
bc_traces:
  - BC-2.02.005   # ARQ retransmit-SEND (PC-3 gap→retransmit, PC-5 QUIC-model new seq)
  - BC-2.02.006   # TLPKTDROP (referenced; not modified — arq.TLPKTDROP untouched)
vp_traces:
  - VP-019        # ARQ retransmit property
  - VP-020        # ARQ retransmit property
subsystems: [transport-layer, arq]
architecture_modules:
  - internal/arqsend        # NEW
  - internal/arq
  - internal/outerassembler
  - internal/frame
  - internal/halfchannel
  - internal/routing
  - internal/netingress
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: [S-4.03, S-BL.OA]
blocks: []
head_sha: 3cd5fb7
branch: feat/s-bl-arq-tx-retransmit-send
base: origin/develop@5e625f1
worktree: .worktrees/arq-tx-story
drift_consumed:
  - id: S403-H1-DEFER
    description: "S-4.03 retransmit-SEND PC-3 deferred to router/multipath wiring story — Wave 4 audit."
adr_disposition: []
drift_dispositioned:
  - id: S404-OBS-F / S404-LOW-1
    verdict: not-yet-relevant
    rationale: >
      STATE.md line 90 anchors S404-OBS-F to "re-confirm on production
      wiring." The team-lead assignment flagged this story as a
      candidate for that re-confirmation. Analysis: S404-OBS-F is the
      E-FWD-001 rate-limit LATENT (see internal/routing/on_frame_arrival.go
      §"Two-tier rate-limiting strategy" + BC-2.02.008 PC-3 tests).
      That surface is the RECEIVE-side forwarding path where
      split-horizon-blocked frames get logged. This story delivers a
      SEND-side seam (retransmit composition → wire bytes → caller
      Dispatch). No new receive-side forwarding surface. The receive-
      side integration test (TestIntegration_GapWalkToRoutedRetransmit)
      DOES route retransmitted frames through routing.RouteFrame, but
      the frames route to a valid dst forwarding entry — no
      split-horizon path exhaustion, no E-FWD-001 log emission is
      exercised. S404-OBS-F remains LATENT until a live daemon
      send-path story wires a real egress + monitors rate-limit
      behavior under sustained retransmit load. This story is NOT
      that story (see Follow-ons §"Live send-path wiring"). Verdict:
      re-anchor S404-OBS-F to the live-egress follow-on, not this seam.
---

# S-BL.ARQ-TX — ARQ Retransmit-SEND Wiring (DELIVERY)

## What Landed

A new pure-core-composition package `internal/arqsend` that closes the
seam between the ARQ state machine (`internal/arq`) and authenticated
wire bytes. The pre-story surface had `arq.OnAck`, `arq.GapsToRetransmit`,
`arq.EnqueueSend`, and (as of this story) `arq.PayloadForInFlight` /
`arq.RemoveInFlight` — but nothing that turned a "retransmit this
payload" decision into an HMAC-authenticated wire frame verifiable by
the receive-side routing stack. This story delivers that seam.

Three concerns land together:

1. **Pure-core ARQ accessors** (`internal/arq/arq.go`):
   - `PayloadForInFlight(seq uint32) []byte` — defensive-copy read of an
     in-flight payload (go.md rule 12: never return internal pointers
     from locked state). Returns nil for absent seqs.
   - `RemoveInFlight(seq uint32)` — idempotent delete; releases the
     old queue entry after a retransmit has been committed under a new
     seq. No-op if the seq is not present.
   Both accessors are single-writer and hold the ARQ mutex per go.md
   rule 12. They are the minimal state-transition primitives arqsend
   needs to compose the QUIC-model release.

2. **Boundary-layer retransmit seam** (`internal/arqsend`):
   `arqsend.New(a, env, opts...)` returns a `*Retransmitter`.
   `sender.Retransmit(oldSeq, newSeq, now, dispatch)` composes:
   - `arq.PayloadForInFlight(oldSeq)` → returns `ErrSequenceNotInFlight`
     if the seq is not in flight (dispatch is NOT called; no state
     mutation).
   - `outerassembler.Assemble(cf{ChanSeq: newSeq, Payload: payload}, {}, env)`
     to compose the wire bytes with the channel-header ChanSeq stamped
     to `newSeq` (BC-2.02.005 PC-5, QUIC retransmit model).
   - `dispatch(wire)` — if it returns an error, arqsend returns the
     wrapped error and leaves ARQ state UNCHANGED. The retransmit is
     re-tryable on the next `GapsToRetransmit` pass (no-orphan-state
     invariant).
   - On dispatch success: `arq.EnqueueSend(newSeq, payload, now)` then
     `arq.RemoveInFlight(oldSeq)`. The QUIC transition commits: the
     payload lives under the new seq; the old seq is released.

   Package doc names the ARCH-09 pure-core-composition classification
   (arqsend performs no I/O of its own — dispatch owns wire delivery)
   and the single-writer concurrency contract (one Retransmitter per
   ARQ handle per goroutine, matching arq's own single-writer contract).

3. **End-to-end composed integration** (`internal/arqsend/integration_test.go`):
   drives the full path — gap-detection → arqsend.Retransmit →
   netingress.ReadFrame → routing.RouteFrame — against an admitted
   node + registered forwarding entries. Verifies HMAC + admission +
   SVTNRoute all succeed at real payload bytes, and asserts ARQ state
   postconditions across a monotonic new-seq batch (BC-2.02.005 PC-5).

## Scope Delivered vs Deferred

**Delivered:**

- `internal/arq/arq.go` — `PayloadForInFlight` + `RemoveInFlight`
  accessors under the existing ARQ mutex.
- `internal/arq/retransmit_accessors_test.go` — 3 tests: returns
  defensive copy, returns nil for absent, RemoveInFlight is
  idempotent.
- `internal/arqsend/arqsend.go` — `Retransmitter`, `New`,
  `WithChanID`, `WithFlags`, `Dispatch`, `ErrSequenceNotInFlight`,
  and the `Retransmit(oldSeq, newSeq, now, dispatch)` method. Package
  doc traces the ARCH-09 classification, side-effect ordering, and
  concurrency contract.
- `internal/arqsend/retransmit_test.go` — 5 unit tests:
  happy-path emits wire for newSeq with the right ChanID/ChanSeq,
  unknown oldSeq returns `ErrSequenceNotInFlight` without side effect,
  dispatch error leaves ARQ state intact (no-orphan-state), explicit
  BC-2.02.005 PC-5 (newSeq ≠ oldSeq on wire), and HMAC verifiable
  against Envelope.FrameAuthKey using the exact shape
  `routing.verifyFrameHMAC` uses.
- `internal/arqsend/integration_test.go` — 2 end-to-end tests:
  gap-walk to routed retransmit (3 in-flight, SACK marks middle
  received, GapsToRetransmit returns 2 gaps, each retransmit routes
  end-to-end and ARQ state transitions cleanly) and monotonic
  new-seq batch (4 gaps, wire ChanSeq equals assigned new seq at
  every index, never equals corresponding old seq).
- `docs/demo-evidence/S-BL.ARQ-TX/AC-001-retransmit-wiring.tape` —
  VHS source only (POL-004; rendered artifacts gitignored).

**Deferred (documented here; each has a natural follow-on):**

- **Live send-path composition in `cmd/switchboard`** — the daemon
  does not yet call `arqsend.Retransmit`. The story's job is the
  seam; a follow-on wires a scheduler that polls `GapsToRetransmit`
  and invokes `Retransmitter.Retransmit` with a `Dispatch` that
  calls `multipath.Send` on the two fastest paths (BC-2.02.001).
- **SACK piggy-back on retransmit** — `WithFlags` accepts caller
  flags but the wire path currently always emits with an empty SACK
  bitmap (`FlagSACKPresent` unset). A follow-on can add a
  `WithSACKBitmap` option; the assembler already handles the flag
  correctly.
- **FEC group repair triggered by retransmit** — retransmits do not
  currently participate in FEC groups. Future story.
- **RESYNC frame emission** — see `S-BL.OA-DELIVERY.md` §"ADR-005
  Disposition". Retransmit-SEND is a data-path concern; RESYNC is a
  control-frame concern. They share the wire-mechanics primitive
  but not the state machine.

## Findings Consumed

| Finding | Description | Where closed |
|---------|-------------|--------------|
| S403-H1-DEFER | S-4.03 retransmit-SEND PC-3 deferred to router/multipath wiring story (Wave 4 audit) | `internal/arqsend/` seam + `TestIntegration_GapWalkToRoutedRetransmit` (composed gap-detection → wire → routing verification) |

The backlog stub at `.factory/stories/S-BL.ARQ-TX-arq-retransmit-send-wiring.md`
existed for the sole purpose of anchoring this drift item. This
DELIVERY file closes it.

## Package Placement Justification (ARCH-08 §6)

The retransmit-SEND seam lands in a NEW package `internal/arqsend`,
not `internal/arq/sender.go` and not `internal/multipath`. Options
considered and rejected:

- **`internal/arq/sender.go`** (backlog stub's candidate) — landing
  it here forces one of two bad shapes: (a) import
  `outerassembler.Envelope` into `internal/arq`, polluting the
  pure-core ARQ state machine with wire-format types; or (b) invert
  the dependency with an internal callback-style API that couples
  ARQ's public surface to the caller's send loop. Neither preserves
  ARCH-09 pure-core classification cleanly.
- **`internal/multipath`** — multipath is the fan-out primitive
  (choose two fastest paths, dispatch to both). Retransmit is a
  compose-then-fan-out step where the composition (payload lookup +
  Assemble) is the load-bearing part; multipath is a downstream
  consumer of the wire bytes retransmit produces. Landing retransmit
  IN multipath would require multipath to import `internal/arq` +
  `internal/outerassembler`, inverting the dependency graph.
- **`internal/routing`** — routing is the receive-side dispatch
  (parse → verify → route). Retransmit is send-side. Wrong direction.

The chosen shape mirrors two existing boundary-layer packages:
`internal/netingress` (composes `frame` + `hmac` + a caller-supplied
`RouteFn`) and `internal/multipath` (composes a caller-supplied
`SendFunc`). `internal/arqsend` composes `internal/arq` +
`internal/outerassembler` + a caller-supplied `Dispatch`. Same
shape, same concurrency contract, same ARCH-09 classification
(pure-core-composition; caller owns the effectful boundary).

**ARCH-08 §6 import position:** `internal/arqsend` imports
`internal/arq`, `internal/outerassembler`, `internal/frame`,
`internal/halfchannel`. Sits at position 9 (after `outerassembler`
at position 8). No cycle.

**Dispatch signature — deviation from assignment's `SendFn(dst,
wire)` shape:** assignment proposed
`type SendFn func(dst [8]byte, wire []byte) error`. Shipped shape
is `type Dispatch func(wire []byte) error`. Rationale: `dst` is
already encoded in the Envelope (`Envelope.DstAddr`) that
constructs the Retransmitter. Passing `dst` again in the callback
signature would either (a) redundantly duplicate what's already in
the outer header the caller can decode, or (b) invite drift between
the Envelope's dst and the callback's dst. The caller who wants
`dst`-aware dispatch can wrap: `dispatch := func(wire []byte)
error { return realSend(env.DstAddr, wire) }`. A future story that
adds per-retransmit dst selection (e.g. multipath fan-out over
multiple dst peers) can extend the callback shape without breaking
this seam. Semantic equivalence to the assignment's ask.

## Spec-vs-Code Contradictions Found

**None material. One design decision worth noting:**

2. **No-orphan-state on dispatch error.** BC-2.02.005 does not
   explicitly specify the sender-side transactionality of a failed
   send. The QUIC model implies it — a failed retransmit must be
   re-tryable — and this implementation makes it explicit: on
   dispatch error, oldSeq is NOT released and newSeq is NOT
   enqueued. The next `GapsToRetransmit` pass observes the same gap
   and re-emits. The unit test `TestRetransmit_DispatchErrorLeavesARQStateIntact`
   is the load-bearing assertion. No spec change proposed; the
   implementation is the conservative reading of the BC.

## Test Inventory

**New tests:**

- `internal/arq/retransmit_accessors_test.go` — 3 tests
  (defensive copy, absent returns nil, idempotent remove).
- `internal/arqsend/retransmit_test.go` — 5 tests (happy path,
  unknown oldSeq, dispatch error preserves state, BC-2.02.005 PC-5
  explicit, HMAC verify against envelope key).
- `internal/arqsend/integration_test.go` — 2 composed integration
  tests (gap-walk → routed retransmit, monotonic new-seqs across a
  batch of 4 gaps).

**Runs (all clean):**

- `go test ./... -count=1 -race` — all 22 packages green
  (`ok github.com/arcavenae/switchboard/internal/arqsend 2.364s`).
- `go test ./internal/arqsend/... -count=3 -race` — 7/7 tests pass
  across 3 iterations (concurrency clean).
- `golangci-lint run ./...` — 0 issues.
- `just fmt` — no changes (gofumpt clean).
- `just smoke-quick` — 14/14 sentinels pass (report
  `.smoke/20260705T213759Z/report.jsonl`).
- `bash test/smoke/spec-runner.sh` — 5/5 pass
  (`.smoke/20260705T213806Z-spec/report.jsonl`).
- `bash test/smoke/tier3-tutorial.sh` — 4/4 pass
  (`.smoke/20260705T213807Z-tier3/report.jsonl`).

## Blast Radius

**1. Operator-visible surfaces touched:** none. `internal/arq` gains
two accessor methods; no existing method changed signature.
`internal/arqsend` is a new pure-core-composition package with no CLI,
daemon flag, config field, log line, or error taxonomy addition.
`cmd/switchboard` and `cmd/sbctl` are byte-identical to
`origin/develop@5e625f1`. The seam is called only from tests today; a
future daemon wiring story owns the live send-path.

**2. Silent-failure risk:** low. Every code path is exercised by tests.
The load-bearing invariants (BC-2.02.005 PC-5 new-seq, no-orphan-state
on dispatch failure, HMAC verifiability against the envelope key) each
have a dedicated test. `arq.PayloadForInFlight` returns a defensive
copy so a mutation by arqsend's caller cannot corrupt the ARQ queue;
the accessor test asserts this. A regression in the seam surfaces
immediately as `TestIntegration_GapWalkToRoutedRetransmit` failing
against real routing + HMAC + admission — the load-bearing acceptance
signal for any future daemon-side caller.

**3. Smoke gate touched:** none. `just smoke-quick` (14/14),
`spec-runner.sh` (5/5), `tier3-tutorial.sh` (4/4) all pass without
change to any smoke asset. No new sentinel invariant added — no
operator-visible surface changed. The seam's contract is observable
only through composition with the ingress + router, and that
composition is asserted by the integration test.

**Ownership envelope respected:** touched only `internal/arq/`
(accessors + accessor test), `internal/arqsend/` (new package), and
`docs/demo-evidence/S-BL.ARQ-TX/` (VHS tape). Did NOT touch
`.beads/`, `.factory/` (this DELIVERY.md is written to the main
checkout by the implementer under the coordinator's rule that
factory-artifacts commits are handled elsewhere), `.run.yaml`,
`cmd/switchboard/`, `cmd/sbctl/`, `internal/routing/`,
`internal/netingress/`, `internal/frame/`, `internal/hmac/`,
`internal/halfchannel/`, `internal/multipath/`, `internal/admission/`,
or `internal/outerassembler/`.

## Follow-ons (Filed as Deferrals, not Regressions)

- **Live send-path wiring in `cmd/switchboard`** — a follow-on story
  polls `arq.GapsToRetransmit` on a scheduler tick and calls
  `Retransmitter.Retransmit` with a `Dispatch` that fans out via
  `multipath.Send` on the two fastest paths (BC-2.02.001). This is
  the natural composition target for the seam this story delivers.
- **SACK piggy-back on retransmit** — extend `arqsend` with a
  `WithSACKBitmap` option that stamps `outerassembler.FlagSACKPresent`
  + the bitmap into the emitted frame. Requires no changes to
  `internal/outerassembler` (the assembler already handles the
  conditional 8-byte SACK region).
- **RESYNC frame emission + reconnect state machine** —
  `S-BL.RESYNC-FRAME` (as recommended in `S-BL.OA-DELIVERY.md`).
- **FEC-group repair triggered by retransmit** — future story;
  BC-2.01.005 EC-004 territory.

## Commit Trail (on `feat/s-bl-arq-tx-retransmit-send`)

1. `ef502a9` — `feat(arq): PayloadForInFlight + RemoveInFlight accessors (S-BL.ARQ-TX)`
2. `230b588` — `feat(arqsend): retransmit composition seam (S-BL.ARQ-TX)`
3. `0ade41e` — `test(arqsend): end-to-end gap-walk → retransmit → route integration (S-BL.ARQ-TX)`
4. `3cd5fb7` — `docs(demo-evidence): S-BL.ARQ-TX retransmit-wiring VHS tape`

Base: `origin/develop@5e625f1`.
Head: `3cd5fb7`.

PR: opened as `feat(arqsend): S-BL.ARQ-TX — wire ARQ retransmit-SEND
path into router/multipath dispatch (BC-2.02.005 PC-3)` against
`develop`.
