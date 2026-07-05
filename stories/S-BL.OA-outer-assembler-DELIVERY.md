---
artifact_id: S-BL.OA-DELIVERY
document_type: story-delivery
level: ops
story_id: S-BL.OA
version: "1.0"
title: "outer assembler — compose ChannelFrame + OuterHeader into wire frames"
status: delivered
producer: implementer
timestamp: 2026-07-05T21:15:00Z
modified: 2026-07-05T21:15:00Z
phase: 2
wave: 8
priority: P1
scope_phase: E
estimated_points: 3
delivered_points: 3
bc_traces:
  - BC-2.01.002   # EMPTY_TICK / DATA discriminator; PC5 MaxPayloadSize
  - BC-2.01.004   # outer header layout + PayloadLen invariant
  - BC-2.01.005   # channel header layout + reserved-zero rule
  - BC-2.05.008   # wire HMAC verification (assembler emits, router verifies)
vp_traces:
  - VP-066        # CWE-400 bounded reads (adversarial-dual truncation)
subsystems: [transport-layer]
architecture_modules:
  - internal/outerassembler   # NEW
  - internal/frame
  - internal/halfchannel
  - internal/hmac
  - internal/routing
  - internal/netingress
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: [S-1.01, S-1.02, S-2.01, S-BL.NI]
blocks: [S-BL.ARQ-TX, S-BL.DISCOVERY-WIRE]
head_sha: 2732e59
branch: feat/s-bl-oa-outer-assembler
base: origin/develop@7a974f6
worktree: .worktrees/oa-story
drift_consumed:
  - id: wave-adv F-003
    description: "composed wire-format test (assembler → netingress → routing) — DEFERRED to S-BL.OA per closed-stories.md:104"
  - id: wave-adv F-004
    description: "channel-header serializer (12/20-byte layout, SACK conditional) — DEFERRED to S-BL.OA per closed-stories.md:105"
adr_disposition:
  - id: ADR-005
    scope: "resync wire-mechanics"
    verdict: separable-still-anchored
    rationale: >
      STATE.md line 89 re-anchored ADR-005 wire-mechanics to S-BL.OA, but
      the RESYNC frame is a control-frame type with its own state machine
      (RESYNC emission trigger, reconnect handler, replay from
      last_acked_seq+1) that is orthogonal to mechanical channel-header
      serialization. This story ships the wire-mechanics primitive (encode/
      decode a channel header, compose it into a wire frame, compute the
      HMAC that satisfies routing.RouteFrame) that a future RESYNC frame
      will use. The follow-on story S-BL.RESYNC-FRAME can now build on
      outerassembler without re-litigating the byte layout.
---

# S-BL.OA — Outer Assembler (DELIVERY)

## What Landed

A new pure-core package `internal/outerassembler` composing
`(halfchannel.ChannelFrame, sackBitmap, Envelope) → wire bytes`. Two
concerns:

1. **Channel-header wire-format codec** (BC-2.01.005; ARCH-02 §3.2) —
   `EncodeChannelHeader`, `DecodeChannelHeader`, `DecodeChannelHeaderN`,
   `ChannelHeaderSize`. Fixed 12-byte layout, +8 bytes when
   `SACK_present` flag bit 2 is set. Reserved bytes at offsets 9..11
   MUST be zero on decode (BC-2.01.005 PC-3 row 4).

2. **Composed assembler** — `Assemble(cf, sackBitmap, env) → ([]byte,
   error)`. Emits wire bytes byte-for-byte consumable by
   `netingress.ReadFrame` and verifiable by `routing.RouteFrame`. The
   HMAC message shape (zeroed-tag `outer_header || channel_header ||
   payload`) matches `routing.verifyFrameHMAC` exactly. Defensive
   `ErrPayloadTooLarge` re-check when `channel_header_size + payload`
   would truncate the `uint16` `PayloadLen`.

Both concerns are pure-core (ARCH-09): deterministic transformations,
no I/O, no globals, no goroutines. Import position 8 in ARCH-08 §6
(after tmux); imports only `frame`, `hmac`, `halfchannel`.

## Scope Delivered vs Deferred

**Delivered:**

- `internal/outerassembler/channelheader.go` — 12/20 byte codec, flag
  constants (`FlagFECPresent | FlagARQReq | FlagSACKPresent`), error
  sentinels (`ErrChannelHeaderTruncated`,
  `ErrChannelHeaderReservedNonZero`). Package doc comment names the
  ARCH-09 classification, ARCH-08 import position, and out-of-scope
  items with rationale.
- `internal/outerassembler/assemble.go` — `Envelope` struct (groups
  SVTNID + SrcAddr + DstAddr + FrameAuthKey to prevent src/dst swap in
  a positional signature), `Assemble` pure function, `ErrPayloadTooLarge`
  sentinel. Doc comment traces every branch to the BC and lays out the
  wire layout with byte offsets.
- `internal/outerassembler/channelheader_test.go` — 6 unit tests
  covering fixed layout, SACK-present layout, round-trip across every
  flag combination × chan_seq boundary values, size-by-flag
  determinism, truncated-buffer rejection (4 sub-cases including
  SACK-flag-but-only-12-bytes), and reserved-non-zero rejection.
- `internal/outerassembler/assemble_test.go` — 6 unit tests: wire-shape
  matches `PayloadLen`, `FrameType` passthrough for all 5 enum values
  (BC-2.01.002 EMPTY_TICK / DATA discriminator), Envelope fields at
  correct wire offsets, channel-header decodes at offset 44, defensive
  MaxPayloadSize boundary check (exact-max valid + one-over rejected
  for both SACK cases), and local HMAC recompute matches wire tag.
- `internal/outerassembler/integration_test.go` — 3 composed integration
  tests: `TestIntegration_Composed_RoutingRouteFrameVerifies` (3
  sub-cases: DATA / EMPTY_TICK / DATA-with-SACK), adversarial dual 1
  `TestIntegration_FlippedBit_FailsHMAC` (7 bit-flip regions: SVTN,
  src, dst, chan_id, flags, payload head, payload mid), adversarial
  dual 2 `TestIntegration_Truncation_FailsParse` (4 truncation
  shapes: header short, payload zero, channel header short, payload
  mid-cut).
- `docs/demo-evidence/S-BL.OA/AC-001-compose-roundtrip.tape` — VHS
  source only (POL-004; rendered artifacts gitignored).

**Deferred (documented in the package doc comment):**

- Egress transport (TCP/UDP dial, framing on the wire) — the
  assembler emits bytes, but nothing dispatches them. Live daemon
  send-path wiring stays in a follow-on.
- ARQ retransmit TX-side buffering — S-4.03 / S-BL.ARQ-TX.
- FEC group assembly (BC-2.01.005 EC-004) — future story.
- Discovery wire format — `internal/discovery`.
- RESYNC frame emission + reconnect state machine — see "ADR-005
  Disposition" below.
- Live daemon send-path wiring in `cmd/switchboard`.

## Findings Consumed

| Finding | Description | Where closed |
|---------|-------------|--------------|
| wave-adv F-003 | Composed wire-format test (assembler → netingress → routing) | `internal/outerassembler/integration_test.go::TestIntegration_Composed_RoutingRouteFrameVerifies` + two adversarial-dual tests |
| wave-adv F-004 | Channel-header serializer with 12/20-byte layout | `internal/outerassembler/channelheader.go` + `channelheader_test.go` (6 tests) |

Both findings were carried in `.factory/cycles/cycle-1/closed-stories.md`
lines 104–105 as DEFERRED to S-BL.OA. Both close cleanly.

## ADR-005 Resync Disposition

**Verdict: separable / still-anchored to a follow-on story.**

`STATE.md` line 89 says "ADR-005 resync wire-mechanics re-anchored
S-BL.OA (S-BL.NI delivered PR #94 ingress-only)." `ARCH-03.md`
~line 244 was written earlier and says the wire-mechanics belong to
S-BL.NI. The tension is real; the resolution below reconciles them
honestly rather than papering over either.

The **wire-mechanics primitive** (encode/decode a channel header,
compose the header into a MAC'd frame that `routing.RouteFrame`
verifies) is what this story delivers. Any RESYNC frame emitted by a
future story will encode its state through this codec and this
assembler.

The **RESYNC frame protocol** (a control-frame type with its own
state machine: emission trigger on connection-loss detection,
receiver-side reconnect handler, `last_acked_seq+1` replay pump) is
NOT wire-format primitive — it is a control-flow concern layered
above the primitive. That layer:

- Requires a new `FrameType` enum value dedicated to RESYNC (or a
  ctl-frame subtype). The current enum has DATA/EMPTY_TICK/CTL/ARQ/FEC;
  RESYNC would be a new dispatch target with dedicated payload
  semantics.
- Requires an emitter co-located with the send-path buffer + a
  receiver co-located with the ARQ replay buffer. Neither of those
  buffers is in the scope of this pure-core assembler.
- Requires a state machine with observable states (waiting-for-ack,
  resyncing, streaming) that survives connection churn. Testing
  demands live sockets + timers, which is boundary-layer territory
  (netingress or a new resync package), not pure-core.

**Recommended follow-on:** file `S-BL.RESYNC-FRAME` for the control-
frame emission + reconnect state machine. It will consume the
primitives this story delivers without re-litigating the byte layout.

**STATE.md text amendment recommended (not applied — orchestrator
owns STATE.md updates):** line 89 should read "ADR-005 wire-format
primitive shipped in S-BL.OA; ADR-005 RESYNC control-frame protocol
+ reconnect state machine still anchored to a follow-on
(S-BL.RESYNC-FRAME)." This makes the two halves explicit.

## Spec-vs-Code Contradictions Found

**None material. Two minor tensions worth noting:**

1. `ARCH-03.md` ~line 244 vs `STATE.md` line 89 disagree about which
   story owns ADR-005 resync work. Addressed under "ADR-005
   Disposition" above; recommended STATE.md text amendment carried
   there. No code implication — resync is deferred either way.

2. `ARCH-02.md` §3.2 flag bit table has bits 3–7 as reserved. The
   codec stores them into `Flags` as read, and re-encodes them if
   the caller populates them. Reserved-bit-must-be-zero on flag byte
   is NOT enforced (only the 3 explicit reserved bytes at offsets
   9–11 are). The BC-2.01.005 spec is silent on flag reserved bits.
   This is the conservative interpretation: allow future spec
   extension without a codec change. If the spec later adds "bits
   3–7 MUST be zero on decode," a one-line check lands in
   `DecodeChannelHeaderN`. No follow-on filed unless the spec
   tightens.

## Test Inventory

**New tests (all in `internal/outerassembler/`):**

- `channelheader_test.go` — 6 tests, ~200 lines
- `assemble_test.go` — 6 tests, ~340 lines
- `integration_test.go` — 3 tests × 3/7/4 sub-cases = 14 assertions,
  ~330 lines

**Runs (all clean):**

- `just fmt` — no changes.
- `go vet ./internal/outerassembler/...` — no output.
- `go test -race -count=3 ./internal/outerassembler/... ./internal/routing/... ./internal/netingress/... ./internal/halfchannel/...` — all green.
- `go test ./... -count=1` — all 21 packages green (`ok internal/outerassembler 1.560s`).
- `golangci-lint run ./...` — 0 issues.
- `just smoke-quick` — 14/14 sentinels pass (report artifact `.smoke/20260705T210957Z/report.jsonl`).
- `bash test/smoke/spec-runner.sh` — 5/5 pass (`.smoke/20260705T211005Z-spec/report.jsonl`).
- `bash test/smoke/tier3-tutorial.sh` — 4/4 pass (`.smoke/20260705T211006Z-tier3/report.jsonl`).

## Blast Radius

**Operator-visible surfaces touched:** none. This is a pure-core
library package. No CLI, no daemon flag, no config field, no log
line, no error taxonomy addition. `cmd/switchboard` and `cmd/sbctl`
are byte-identical to `origin/develop@7a974f6`.

**Silent-failure risk:** low. The assembler is called by nothing in
the current tree — every code path is exercised only by tests. A
regression in `Assemble` or the channel-header codec fails
`TestIntegration_Composed_RoutingRouteFrameVerifies` immediately;
that test is the load-bearing acceptance signal for any future caller
in the send-path.

**Smoke gate touched:** none. `just smoke-quick` (14/14),
`spec-runner.sh` (5/5), `tier3-tutorial.sh` (4/4) all pass without
change to any smoke asset. No new sentinel invariant added because
no operator-visible surface changed; the assembler's contract is
observable only through composition with the ingress + router.

**Ownership envelope respected:** touched only
`internal/outerassembler/` and `docs/demo-evidence/S-BL.OA/`. Did
NOT touch `.beads/`, `.factory/` (this DELIVERY.md is written to
the main checkout by the implementer under the coordinator's rule
that factory-artifacts commits are handled elsewhere), `.run.yaml`,
`cmd/sbctl/`, `cmd/switchboard/`, `internal/routing/`,
`internal/netingress/`, `internal/frame/`, `internal/hmac/`, or
`internal/halfchannel/`.

## Follow-ons (Filed as Deferrals, not Regressions)

- **S-BL.RESYNC-FRAME** — RESYNC control-frame emission + reconnect
  state machine (see "ADR-005 Disposition" above). Owns the state
  machine that this story's primitive underpins.
- **S-BL.ARQ-TX** — retransmit TX-side buffering. Consumes
  `Assemble` for the wire encoding but adds the retransmit buffer +
  timeout logic.
- **Send-path wiring in `cmd/switchboard`** — a future story
  connects `Assemble` to a real egress path. Today the assembler is
  only reachable from tests; a live daemon send-path is a distinct
  story with its own boundary-layer risks (retry policy, TCP writer
  buffering, back-pressure).
- **STATE.md line 89 text refinement** — separate the wire-mechanics
  primitive (shipped) from the RESYNC control-frame protocol
  (deferred). Orchestrator-owned edit.

## Commit Trail (on `feat/s-bl-oa-outer-assembler`)

1. `6ec0326` — `feat(outerassembler): channel-header codec — 12/20-byte layout with SACK conditional (S-BL.OA)`
2. `9c05abf` — `feat(outerassembler): Assemble composes ChannelFrame + Envelope into wire bytes (S-BL.OA)`
3. `7d7fa25` — `test(outerassembler): composed roundtrip + adversarial duals (F-003)`
4. `6d5dc63` — `docs(demo-evidence): S-BL.OA AC-001 compose-roundtrip VHS tape`
5. `2732e59` — `style(outerassembler): drop trailing blank line per gofumpt`

Base: `origin/develop@7a974f6` (includes PR #94 + PR #95).
Head: `2732e59`.

PR: opened as `feat(outerassembler): S-BL.OA — compose ChannelFrame
+ OuterHeader into wire frames` against `develop`.
