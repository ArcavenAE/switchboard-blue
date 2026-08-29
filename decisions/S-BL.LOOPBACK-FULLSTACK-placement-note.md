---
artifact_id: S-BL.LOOPBACK-FULLSTACK-placement-note
document_type: architect-design-note
story_id: S-BL.LOOPBACK-FULLSTACK
title: "Full-stack loopback testenv extension: tick-driven halfchannel + arq + multipath wiring for VP-042"
status: draft
producer: architect
timestamp: 2026-07-12T00:00:00Z
version: "1.12"
bc_traces:
  - BC-2.01.001   # timeslice clock fires every tick regardless of data availability
  - BC-2.01.002   # empty-tick frame semantics
  - BC-2.02.001   # duplicate-and-race dispatch
  - BC-2.02.002   # endpoint checksum-only dedup
  - BC-2.02.005   # downstream ARQ (piggybacked ACK/SACK, TLPKTDROP)
vp_traces:
  - VP-042        # keystroke-to-echo p99 <= 100ms
architecture_modules:
  - internal/testenv
  - internal/halfchannel
  - internal/arq
  - internal/multipath
  - internal/paths
related_documents:
  - .factory/specs/verification-properties/VP-042.md
  - .factory/specs/architecture/ARCH-08-dependency-graph.md
  - .factory/specs/architecture/ARCH-03-routing-engine.md
---

## Changelog

| Version | Change |
|---------|--------|
| 1.12 | Consistency-audit remediation (2026-08-28, POL-005 dispatch, `.factory/cycles/cycle-1/S-BL.LOOPBACK-FULLSTACK/consistency-audit-2026-08-28.md`): MAJOR (Finding #1) — every live-text citation of the phantom branch `fix/vp-042-testenv-integrated-bench` (three occurrences: §Q2 ~L130, Package Impact Summary ~L613, Risks item 3 ~L642-643) corrected to reflect the actual merged state — `internal/bench/keystroke_echo_testenv_bench_test.go` is MERGED on `develop` (PR #121 @ `4c276d9`, `//go:build integration`-tagged, defines `BenchmarkKeystrokeToEcho_P99`); the branch was never real. MAJOR (Finding #2) — added a new "### Verification Method (AC-013)" subsection specifying the correct verification: `go build -tags integration ./internal/bench/...` (compile) + `go test -tags integration -run '^$' -bench=BenchmarkKeystrokeToEcho_P99 -benchtime=1x -count=1 ./internal/bench/` (run), replacing the prior incorrect framing (bare `go build` + `just bench`, neither of which can reach the integration-tagged function); explicit adjudication recorded: `just bench` stays the tag-free S-BL.BENCH lower-bound recipe, out of AC-013's scope (Option (a)). MINOR (Finding #4) — added Risks item 6, a forward obligation for story-writer to encode a File-List obligation updating VP-042.md's Proof Harness Skeleton to the binding two-call token shape. See "v1.12 Consistency-Audit Repair (2026-08-28)" section. |
| 1.11 | R8 re-review repair (2026-08-28): LOW (F-LENSA-01) / NITPICK (F-ORACLE-01) — the v1.10 erratum corrected the two SPACE-COLON `t.Helper()` citations `:460`→`:461`, but a THIRD live citation in SLASH notation (`testenv.go:384/460/475/528`, the `recordingTB` struct-field comment at §M2 ~L1463) survived uncorrected; now corrected to `384/461/475/528`. The v1.10 "two citations" accounting was complete for the space-colon form but the class also included the slash form; this completes the class sweep. Frozen v1.10 row/section left as-is per §2.9; ground truth `internal/testenv/testenv.go:461`. See "v1.11 R8 Re-Review Repair (2026-08-28)" section. |
| 1.10 | R7 re-review repair (2026-08-28): MED (F-LENSB-01) — §M2's ticker-start mitigation decision is corrected: the "upstream ticker MAY start at construction" / "start-time freedom PRESERVED" latitude ([v1.6 F-B-LENSB-01]) is WITHDRAWN. The upstream ticker now starts at `CreateSession` time, SYMMETRIC with the downstream ticker — race-free for the same single-goroutine reason, with no `EnqueueSend` dependency blocking it. The "Driver lifecycle pin" (§H3) and the "Steps 1–3" cross-reference sentence are updated so `CreateSession` is described as starting BOTH tickers, not just the downstream one. AC-017's "single-goroutine throughout" method text is sharpened: because AC-017 never calls `CreateSession` (the sole start-site for both tickers), no ticker goroutine of either kind can run during the test — this is now TRUE BY CONSTRUCTION, closing a latent `go test -race` hazard between a construction-started upstream ticker's locked `failLoud` write and AC-017's unlocked `len(stub.errorfCalls)` assertion read (outcome count was always 1; this was a race annotation, not a wrong count). LOW (F-LENSB-02) — the §H3 `sessionName` storage/timing pin is corrected: AC-017's `ErrConsoleNotFound` is caused PRIMARILY by the un-attached, zero-value `loopbackConsoleKey` (`AccessNode.SendKeystroke`, `internal/session/upstream.go:288-290`, gates on console attachment BEFORE comparing `sessionName` at line 292), not by the empty `sessionName`, which is demoted to a redundant second guarantee. Erratum — two live-text citations of `t.Helper()`'s line number corrected from `:460` to the disk-verified `:461` (`internal/testenv/testenv.go`); the unrelated `cmd/switchboard/access.go:460` citations, and the frozen v1.9 changelog row/section's `:460` citation, are untouched per §2.9. See "v1.10 R7 Re-Review Repair (2026-08-28)" section. |
| 1.9 | R6 re-review repair (2026-08-28): BLOCKER — §M2's `recordingTB` stub construction (`stub := &recordingTB{}`, embedding a nil `testing.TB`) is corrected to `stub := &recordingTB{TB: t}` (embedding the real enclosing `*testing.T`). Disk-verified against `internal/testenv/testenv.go`: `NewLoopback`→`newEnv` unconditionally calls `b.Helper()` (:384), `t.Helper()` (:460), and `t.Cleanup(...)` twice (:475, :528 — the ticker/env teardown AC-011/Q6 depend on); with a nil embed these promoted calls nil-panic at construction, before AC-016/AC-017's fault-injection procedure ever runs. The struct comment's false "unused methods panic if called" claim is corrected, and the prior "in place of the real `*testing.T`" / implicit "must NOT pass real t" framing is retracted and replaced with the correct semantics: `Errorf` is OVERRIDDEN on `*recordingTB`, so `stub.Errorf(...)` dispatches to the override and is captured — it never reaches the embedded real `t`, so the enclosing test is never marked failed; the embedded real `t` is required so `Helper`/`Cleanup`/`Fatalf` delegate to a live TB instead of nil-panicking. LOW (Lens B O-1) — §H3 gains a `sessionName` storage/timing pin: `sessionName` is stated as a `loopbackDriver` field, set in `CreateSession` alongside `loopbackConsoleKey` (before Steps 1–3 run), holding its zero value (`""`) before `CreateSession` has run — consistent with the "Driver lifecycle pin," and the reason AC-017's pre-`CreateSession` call observes `ErrConsoleNotFound` independent of `loopbackConsoleKey`'s own zero-value state. See "v1.9 R6 Re-Review Repair (2026-08-28)" section. |
| 1.8 | R5 re-review repair (2026-07-24): F-LENSB-B-01 — §H3 provisioning-timing/lifecycle pin: removed the "(or the `loopbackDriver` constructor)" console-provisioning latitude that contradicted AC-017's un-provisioned premise; added a "Driver lifecycle pin" specifying the `loopbackDriver` constructor builds the AccessNode triple, both multipath instances, and both half-channels in an un-provisioned state at construction (not lazily at `CreateSession`), so `SendKeystroke`/`onUpstreamTick()`/`onDownstreamTick()` are all safely pre-`CreateSession`-callable with no nil-deref; console provisioning (Publish/RegisterKey/Attach) confirmed deferred to `CreateSession` only; propagated a cross-reference at the "Steps 1–3" sentence. F-LENSB-B-02 — Q3 and the SendKeystroke doc comment (§Q5) now state explicitly that `SendKeystroke` performs no session-existence validation, by design, for AC-017. F-LENSB-B-03 — §M2 adds one recording-`testing.TB`-stub requirement covering both AC-016 and AC-017's fault-injection tests, so `failLoud`'s `t.Errorf` is captured/asserted rather than failing the enclosing real `*testing.T`. A-L967 (nitpick) — §H1 "Defect" prose's last live 1-value `decodeRTID(payload)==rt.id` token reworded to semantic English ("the delivered payload decodes to `rt.id`"); the frozen v1.4 §B-1 historical description at a separate line is untouched per §2.9. See "v1.8 R5 Re-Review Repair (2026-07-24)" section. |
| 1.7 | F-A-1 class propagation (2026-07-23): benchmark pseudocode L518 comment dropped the 1-value `decodeRTID(payload)==rt.id` code token — reworded to semantic English consistent with v1.6 L337 fix. Zero live 1-value `decodeRTID(payload)==` tokens remain outside dated history/defect-description sections. |
| 1.6 | R4 re-review repair (2026-07-22): F-B-LENSB-01 — resolved upstream-ticker/AC-017 timing tension using the preferred direct-seam approach: `onUpstreamTick()` named as a binding directly-callable package-private seam (symmetric to `onDownstreamTick()` / AC-016); AC-017 test specified to invoke `onUpstreamTick()` directly/synchronously (no ticker goroutine started), making AC-017 ticker-timing-independent; "MAY start at construction" preserved as valid but no longer load-bearing for AC-017 correctness; upstream seam guidance block added after §B-3/AC-016 section. L-C-1 — §H2 concrete-shape block comment lead-ins L1044/L1060 reworded from stale "onTick callback" form to reference `tickBody` wiring / seam method names (`startLoopbackTicker(env, ..., d.onUpstreamTick)` / `startLoopbackTicker(env, ..., d.onDownstreamTick)`). F-A-1 — RoundTrip doc-comment L337 purpose sentence reworded from `decodeRTID(payload)==rt.id` (1-value code token) to semantic English ("the delivered payload decodes to rt.id"). See "v1.6 R4 Re-Review Repair (2026-07-22)" section. |
| 1.5 | R3 re-review repair (2026-07-22): F-B-1 — `startLoopbackTicker` rewritten tick-free (no `hc.Tick()` in helper; parameter changed from `onTick func(halfchannel.ChannelFrame)` to `tickBody func()`; `hc` param dropped); both directions wired through their internal-tick seam methods (`onUpstreamTick` / `onDownstreamTick`) as the ticker `tickBody`; false-composition claim at §M2/§B-3 L1240 retracted and reworded. F-B-4 — `driver.errCh` "acceptable alternative" dropped; `t.Errorf`-based `failLoud` is now the sole specified error-surface mechanism. F-B-2 — first-SendKeystroke "chanSeq=1" invariant qualified as config-dependent (holds only when downstream interval > upstream round-trip, as in VP-042). F-B-2b — CreateSession-time window prose corrected from "N empty ticks" to "N+1" (data tick itself increments seq). See "v1.5 R3 Re-Review Repair (2026-07-22)" section. |
| 1.4 | R2 re-review repair (2026-07-22): B-1 — every `decodeRTID` call site in note corrected to 2-value form (`id, ok := decodeRTID(payload)`); `!ok` handling specified; rtSeq-starts-at-1 safety note added. B-2 — §M2 window-margin rationale corrected per-option (CreateSession-time: "few empty ticks before first send", not "chanSeq=1"; first-SendKeystroke: "chanSeq=1, nextExpected=0"); >3.2s-idle caveat documented; preferred option remains CreateSession-time for race-freedom. B-3 — AC-016 fault-injection method respecified: build driver, withhold downstream ticker, advance `downstreamHC` past 64 via empty `Tick()` calls, enqueue one payload, invoke `onDownstreamTick()` synchronously (no race); seam `onDownstreamTick()` named as REQUIRED package-private function. N-1 — §M2 "acceptable simpler alternative" paragraph removed; "no third option" invariant now holds without self-contradiction. See "v1.4 R2 Re-Review Repair (2026-07-22)" section. |
| 1.3 | R1 re-review repair (2026-07-22): B-F1 — §H3 provisioning sequence missing `pub.Publish(sessionName)` before `Attach` (added; Attach gates on pub.Get); B-F2 — `encodeRTID` Q3 call site corrected to 2-arg whole-payload form matching §M4 definition; B-F3 — §M2 lazy-start decision tightened: prefer `CreateSession`-time start (single-threaded, race-free), `sync.Once` required if first-`SendKeystroke` start is chosen; C-F1 — Q3 pseudocode L161 `map[uint64]chan frame.OuterHeader` corrected to `map[uint64]chan []byte`; C-F2 — v1.1 Q4 Addendum "Required revision" L680 phantom `mpFrame.ChanSeq()` corrected to captured `chanSeq`. See "v1.3 R1 Re-Review Repair (2026-07-22)" section. |
| 1.2 | Design repair addendum (2026-07-22): B1 ChanSeq threading — corrected `mpFrame.ChanSeq()` phantom call to use captured `f.ChanSeq` from the originating `ChannelFrame`; H1 harness API shape — `RoundTrip.done` changed to `chan []byte`, `WaitForEcho` returns `([]byte, bool)`; H2 halfchannel synchronization — per-direction mutex strategy specified; H3 console provisioning — `RegisterKey`+`Attach`+`Allow` sequence specified as construction-time step; M2 OnAck error handling + empty-tick window — loud failure + don't-tick-until-enqueue mitigation specified; M4 helper signatures — enumerated with explicit obligations. See "v1.2 Design Repair Addendum (2026-07-22)" section. |
| 1.1 | AC-001 sign-off (S-BL.LOOPBACK-FULLSTACK Risk 1 / Q4): reviewed the proposed `arq.OnAck` call-contract against `internal/arq/arq.go`, its full test suite, `internal/arqsend`, and ARCH-03 §Downstream ARQ. Verdict: REVISED. The `ackSeq`/SACK value convention is CONFIRMED correct; the `driver.arqServer`/`driver.arqClient` two-instance shape is a structural defect — `OnAck`'s payload recovery reads only from the calling instance's own `inFlight`/`reorderBuf`, populated exclusively by prior `EnqueueSend` calls on that SAME instance, so a never-`EnqueueSend`'d `arqClient` can never return a delivered payload and `WaitForEcho` would time out on every call. Required fix: collapse into one shared `*arq.ARQ` instance. See "Q4 Addendum — AC-001 Sign-off (2026-07-12)" below. Supersession banner added at the top of Q4. |
| 1.0 | Initial release. Full design note (Q1–Q8) for the tick-driven loopback stack, VP-042 benchmark shape, Non-Goals, package impact summary, story-sizing estimate, and Risks/Open Questions requiring story-writer ACs. |

# Architect Design Note: Full-Stack Loopback for VP-042
## Story: S-BL.LOOPBACK-FULLSTACK

This note answers the design questions needed to unblock story-writer for the
extension of `internal/testenv`'s `NewLoopback`/`LoopbackEnv` API from a
same-goroutine `DeliverFrame` shortcut into a tick-driven, protocol-accurate
loopback stack that can finally lock VP-042. All file:line anchors refer to
develop `f73676d`.

## Ground Truth (established by formal-verifier and this session's reading)

- `testenv.NewLoopback` (`internal/testenv/testenv.go:383`) discards its
  `LoopbackConfig` and calls `newEnv(ctx, b, 1)`. `LoopbackConfig.TickIntervalUpstream`
  / `TickIntervalDownstream` (`testenv.go:364`) are dead fields.
- `Env.SendKeystroke` (`testenv.go:744`) does **not** go through
  `session.AccessNode.SendKeystroke`/`KeystrokeSink` at all — it directly calls
  `sh.access.DeliverFrame(hdr)`, i.e. it synthesizes a *downstream* fan-out frame
  under the name "SendKeystroke". `AccessNode` is goroutine-free
  (`internal/session/upstream.go:128`); there is no tick scheduler anywhere in
  the path.
- ARCH-08 position 22 (test-only composition root, now 23) imports
  `{admission, drain, frame, outerassembler, session, upstreamdial}`. It does
  not import `halfchannel`/`arq`/`multipath`, so nothing in testenv drives
  `halfchannel.Tick()`.
- `Env.CollectFrames` (`testenv.go:758`) and `Conn`/`Console.CollectFrames`
  (`testenv.go:86`, `:161`) poll an **accumulating** slice — `WaitForEcho`
  (`testenv.go:1057`) returns as soon as the slice is non-empty, so a second
  concurrent or leftover round trip's frame satisfies a `WaitForEcho` call that
  isn't waiting for it. This is a distinct bug from the tick/protocol gap and
  must be fixed independently of it (Q5 below).
- ARCH-03 §Downstream ARQ / §Upstream Idempotent Replay / §F-023 (read-only
  console ACK) pin the real protocol asymmetry: **upstream keystroke delivery
  uses `internal/replay` (idempotent replay window), not ARQ** — "keystroke
  loss is self-healing without explicit ARQ" (ARCH-03 line 159). ARQ applies
  only to the **downstream** direction (access node = sender, console =
  receiver); the console's SACK bitmap acknowledging downstream frames rides
  on the console's own upstream channel header (F-023), not a separate ACK
  channel.
- No production code calls `arq.OnAck` today. `internal/arqsend` (the only
  production consumer of `*arq.ARQ`) only exercises the sender-side subset
  (`PayloadForInFlight`/`EnqueueSend`/`RemoveInFlight`). This design is
  therefore the **first proposed call site for `OnAck`** in the codebase — see
  Q4 for the specific call contract this note commits to, and the Risks
  section for why that commitment needs architect/adversarial sign-off before
  implementation, not just story-writer transcription.

---

## Q1 — Does this expand `internal/replay` scope too, per the team's request phrasing?

**Decision: No. Scope is exactly `{halfchannel, arq, multipath}` (+ the
transitively-required `internal/paths`), matching ARCH-08 v2.13. `internal/replay`
is explicitly out of scope.**

The dispatch request describes routing keystrokes "upstream through halfchannel
framing + arq + multipath duplicate-and-race." Read literally that could imply
ARQ on the upstream leg. ARCH-03 is unambiguous that it does not: upstream
keystroke reliability is `internal/replay`'s job (self-healing sliding replay
window), and ARQ is documented as downstream-only in both its package doc
(`internal/arq/arq.go:1`) and ARCH-03's "Downstream ARQ (internal/arq,
BC-2.02.005)" section. VP-042's own Source Contract cites BC-2.01.001 (tick)
and BC-2.02.001 (duplicate-and-race) — not BC-2.02.004 (replay) or BC-2.02.005
(ARQ) — as the two BCs it exists to verify; ARQ enters only because the
downstream leg of the round trip is architecturally required to carry it.

Consequence: this design puts `arq` on the **downstream** half-channel only.
Upstream keystroke delivery is halfchannel + multipath, with no reliability
layer beyond multipath's endpoint dedup — architecturally correct (loss would
be self-healing via replay in production; this benchmark has no simulated loss,
so replay's absence changes nothing observable). If the team wants full
BC-2.02.004 fidelity in the harness later, `internal/replay` (position 13,
also below 23) is a lawful, independent follow-on addition — it does not
change this design's shape.

---

## Q2 — Where does the tick-driving live: a new type, or methods bolted onto `Env`?

**Decision: a new unexported `loopbackDriver` type inside `internal/testenv`,
owned by `LoopbackEnv`, with `SendKeystroke`/`WaitForEcho`/`CreateSession` as
NEW methods on `*LoopbackEnv` — not on `*Env`.**

`LoopbackEnv` is currently `struct { Env *Env }` — a **named field**, not
Go anonymous embedding (confirmed: the existing merged bench test
(`internal/bench/keystroke_echo_testenv_bench_test.go`, `develop`, PR #121
@ `4c276d9`) does `env := lb.Env; env.CreateSession(b)`, never
`lb.CreateSession(b)`; if `Env` were embedded, both forms would resolve).
This means new methods on
`*LoopbackEnv` do not collide with or shadow `*Env`'s methods — they are
simply a separate method set reached via `lb.Foo(...)` instead of
`lb.Env.Foo(...)`.

**Why not extend `Env.SendKeystroke`/`Env.CollectFrames` in place:** those
methods back 10 other VPs (VP-033, 034, 036, 037, 038, 039, 040, 046 per the
package doc) via SVTN-shard fan-out semantics that are deliberately generic
("did a frame arrive on this session") — not round-trip-specific. Rewiring
them to be tick-driven and round-trip-tagged would be a blast-radius change
across every other testenv consumer for a semantics none of them asked for.
`LoopbackEnv` getting its own narrow, protocol-accurate method set is the
minimal-diff option: `NewLoopback` keeps calling `newEnv(ctx, b, 1)` (so
`lb.Env.Close()`/generic surface stay available, harmless if unused), and
`LoopbackEnv` additionally constructs and owns a `*loopbackDriver` with its
own dedicated session/shard.

**Why the loopback driver needs its own dedicated shard, not `env.defaultShard`:**
`newShard` (`testenv.go:534`) hardcodes
`session.WithKeystrokeSink(session.NoOpSink{})`. `session.AccessNode` has no
`SetSink` — the `KeystrokeSink` is fixed at construction via functional
option (`internal/session/upstream.go:104`), by design (production callers
inject a stable sink once; a mutable-sink escape hatch would weaken that
invariant for every other consumer of `AccessNode`, not just testenv). Rather
than add that escape hatch to production `session.AccessNode`, the loopback
driver builds its own `Publisher`/`SessionAuth`/`AccessNode` triple —
identical in shape to `newShard`, but with `WithKeystrokeSink(loopbackSink)`
from the start, where `loopbackSink` is the driver's own echo-generating sink
(Q4). This is a few lines of duplication against `newShard`, isolated to the
loopback path; it does not touch `newShard` or any other VP's shard.

---

## Q3 — Upstream flow: keystroke → server delivery

```
LoopbackEnv.SendKeystroke(t, sessionID, key)
    │  mints RoundTrip{id: driver.rtSeq.Add(1)}; registers a completion
    │  channel under that id in driver.pending (map[uint64]chan []byte,
    │  guarded by driver.mu)           // [v1.3 C-F1: was chan frame.OuterHeader; corrected to chan []byte, consistent with §H1]
    │  payload := encodeRTID(key, id)  // [v1.3 B-F2: 2-arg whole-payload form — see §M4 definition]
    ▼
driver.upstreamHC.Enqueue(payload)      // pure, non-blocking, halfchannel.go:143
    │  (returns to caller immediately — SendKeystroke does NOT block on
    │   delivery; this is deliberate: it models "the client queued a
    │   keystroke," not "the keystroke arrived." BC-2.01.001 requires the
    │   tick to fire on its own schedule regardless of enqueue timing.)
    ▼
[async] upstream ticker goroutine (Q6), every cfg.TickIntervalUpstream:
    f := driver.upstreamHC.Tick()                          // halfchannel.go:117
    if f.FrameType == frame.FrameTypeData {                // has payload
        driver.upstreamMP.Send(toMPFrame(f), driver.deliverUpstream)
    }
    // empty ticks are produced (BC-2.01.002) but not wire-dispatched —
    // see Non-Goals.
    ▼
driver.deliverUpstream(pathID, mpFrame) error   // called once per selected
    │  path (up to 2, duplicate-and-race, multipath.go:244) — the SAME
    │  callback for both, since both loopback paths terminate in this
    │  one process
    ▼
driver.upstreamMP.Receive(mpFrame)     // endpoint checksum dedup, multipath.go:318
    │  ErrDuplicate on the second-arriving copy → discard, return nil
    │  nil (first arrival) → continue
    ▼
driver.accessNode.SendKeystroke(loopbackConsoleKey, sessionName, mpFrame.Payload)
    │  internal/session/upstream.go:276 — authorizer check, sinkMu-serialized,
    │  synchronous call into the injected KeystrokeSink
    ▼
loopbackSink.SendInput(payload) error   // Q4
```

**`SendKeystroke` performs no session-existence validation [v1.8 F-LENSB-B-02]:**
The first step above — mint `RoundTrip`, register `driver.pending[id]`, encode
`payload`, `Enqueue` into `upstreamHC` — is UNCONDITIONAL. `SendKeystroke` does
NOT check that `sessionID` refers to an existing or provisioned session before
doing any of this. This is deliberate: AC-017 (§M2) calls `SendKeystroke` BEFORE
`CreateSession`, when the session's console is not yet provisioned (see the
"Driver lifecycle pin" in §H3), and depends on that pre-`CreateSession` call
succeeding at the mint/encode/enqueue level — the failure AC-017 exercises
surfaces later and downstream, at `accessNode.SendKeystroke` inside
`onUpstreamTick()` (`ErrConsoleNotFound`, via `failLoud`), not at `SendKeystroke`
itself. An implementer who adds a defensive session-existence guard to
`SendKeystroke` (e.g. `if !driver.sessionExists(sessionID) { t.Fatalf(...) }`)
would abort AC-017 at step 1, before `onUpstreamTick()` ever runs, and the test
would fail for the wrong reason instead of exercising the `failLoud` path it is
meant to test. `SendKeystroke` MUST remain unconditional in this respect; no
session-existence guard is permitted.

**Why `SendFunc` is called from inside the ticker goroutine, not spawned into
its own goroutine per path:** `multipath.Send`'s doc explicitly says `fn` is
called "without holding any internal lock" — it is safe to do real work
in `fn`. Both loopback paths have zero synthetic added latency (see
Non-Goals: no real network), so there's no concurrency benefit to spawning;
running both calls sequentially in the ticker goroutine is simpler and avoids
a class of races (out-of-order dedup-cache insertion) that a fully-faithful
network simulation would have to reckon with but this design deliberately does
not model.

---

## Q4 — Downstream flow: echo generation → client delivery → round-trip completion

> **[AC-001 sign-off annotation — 2026-07-12]** The `driver.arqServer` /
> `driver.arqClient` two-instance shape shown below is SUPERSEDED. Architect
> sign-off (Risk 1) found that `arq.OnAck`'s payload recovery (`payloadFor`,
> `arq.go:291`) reads only from the SAME instance's `inFlight`/`reorderBuf`
> maps, which are populated exclusively by prior `EnqueueSend` calls on that
> instance (`arq.go:339`). A separate `arqClient` that never receives
> `EnqueueSend` calls can never return a delivered payload from `OnAck` —
> `WaitForEcho` would time out on every call, not just in a subtle edge
> case. Collapse `arqServer`/`arqClient` into ONE shared `*arq.ARQ` instance.
> The `ackSeq`/SACK value convention chosen below (this frame's own ChanSeq,
> zero SACK) is CONFIRMED correct and unaffected by this fix. See
> "Q4 Addendum — AC-001 Sign-off (2026-07-12)" at the end of this note for
> the full ruling and reasoning trail. Do not implement the two-instance
> shape from the code blocks below — implement the shared-instance shape
> per the Addendum.

`loopbackSink.SendInput` (the `KeystrokeSink` injected into the loopback
driver's dedicated `AccessNode`, per Q2) is the echo generator:

```go
func (s *loopbackSink) SendInput(payload []byte) error {
    return s.driver.downstreamHC.Enqueue(payload)   // echoes the FULL
}                                                     // payload verbatim,
                                                       // including the
                                                       // embedded RT-ID —
                                                       // the sink does not
                                                       // need to understand
                                                       // the correlation
                                                       // scheme; it just
                                                       // echoes bytes, like
                                                       // real tmux would.
```

`SendInput` is called while `AccessNode` holds `sinkMu`
(`internal/session/upstream.go:63`: "must not call back into AccessNode under
any lock"). `Enqueue` only touches the downstream `HalfChannel`'s own pending
queue — it never calls back into `AccessNode` — so this is safe by construction,
and it is also the *correct* modeling of BC-2.01.001: the echo is queued, not
delivered synchronously; the downstream ticker decides when it actually goes
out.

```
[async] downstream ticker goroutine, every cfg.TickIntervalDownstream:
    f := driver.downstreamHC.Tick()
    if f.FrameType == frame.FrameTypeData {
        driver.downstreamARQ.EnqueueSend(f.ChanSeq, f.Payload, time.Now())  // arq.go:339
        // [v1.2 correction] f.ChanSeq is captured here from the originating
        // halfchannel.ChannelFrame.  It is NOT recoverable from mpFrame later:
        // multipath.Frame has only OuterHeader+Payload — no ChanSeq field or method.
        chanSeq := f.ChanSeq
        driver.downstreamMP.Send(toMPFrame(f), driver.deliverDownstream)
        // deliverDownstream runs synchronously within this goroutine (Q3 rationale);
        // chanSeq remains valid at the OnAck call site below.
        delivered, err := driver.downstreamARQ.OnAck(chanSeq, zeroSACK)  // arq.go:201
        // [v1.2 correction] arqServer/arqClient renamed to single downstreamARQ instance (Addendum).
        // [v1.2 correction] mpFrame.ChanSeq() was phantom — multipath.Frame has no ChanSeq;
        //   use the captured f.ChanSeq from the ChannelFrame above.
        // err MUST be checked and fail loud (M2 — SOUL.md #4):
        if err != nil {
            // surface via t.Errorf / log.Printf — do NOT swallow
            driver.failLoud(err)
            return
        }
        //  ackSeq = this frame's own ChanSeq (cumulative watermark == +1 in
        //  no-loss harness); SACK bitmap all-zero (no reordering — Non-Goals).
        for _, payload := range delivered:
            id, ok := decodeRTID(payload)    // [v1.4 B-1] 2-value; on !ok, no pending entry matches
            if !ok { continue }              // decode failure: payload too short; skip (can't be a real
                                              // pending key — rtSeq.Add(1) ensures ids start at 1,
                                              // so id=0 never collides with any pending key)
            driver.mu.Lock(); ch := driver.pending[id]; delete(driver.pending, id); driver.mu.Unlock()
            if ch != nil { ch <- payload }   // [v1.2 correction] sends []byte payload, not frameFor(payload)
                                              // — WaitForEcho now returns payload for AC-014 assertion
    }
```

> **[v1.2 correction annotation — 2026-07-22]** Three corrections applied to
> this pseudocode block vs. v1.1:
> (1) `mpFrame.ChanSeq()` replaced with captured `f.ChanSeq` from
> `halfchannel.ChannelFrame` — `multipath.Frame` has no `ChanSeq` method or
> field (B1 fix; see v1.2 Addendum §B1).
> (2) `arqServer`/`arqClient` renamed to `downstreamARQ` (single shared instance,
> carried forward from the v1.1 Addendum).
> (3) `ch <- frameFor(payload)` replaced with `ch <- payload` — `RoundTrip.done`
> is now `chan []byte`; the raw payload is what `WaitForEcho` must return for
> AC-014's `decodeRTID` assertion (H1 fix; see v1.2 Addendum §H1).

**`arqClient.OnAck` call-contract decision (flagged for architect sign-off,
see Risks):** no production code calls `OnAck` yet, so there is no existing
call-site convention to match. This design treats `OnAck`'s `ackSeq` argument
as "the highest downstream `ChanSeq` this receiver has now observed in order"
— i.e. **locally-derived from arrival**, not a peer-supplied value — and
calls it once per received (post-dedup) downstream frame with that frame's
own `ChanSeq`. Because the loopback has a single downstream producer emitting
strictly increasing `ChanSeq` values one tick at a time, and no synthetic
loss/reordering (Non-Goals), this call is equivalent to "advance cumulative
delivery by exactly one" on every call — it never needs `OnAck`'s SACK-buffer
or gap-handling paths in the happy path, but it does exercise `OnAck`'s real
window-validation (`RULING-003`/`ErrAckOutOfWindow`, arq.go:220) and
delivery-pointer bookkeeping on every sample.

**Why not call `GapsToRetransmit`/`TLPKTDROP` at all:** see Non-Goals — there
is no simulated loss, so `arqServer.inFlight` never accumulates a real gap.
Wiring an active poll for a condition that structurally cannot occur in this
harness would be dead code exercised by nothing. `EnqueueSend` alone still
gives an honest measurement of the sender-side bookkeeping cost (map insert +
deadline computation) that production incurs on every downstream tick.

---

## Q5 — Fixing the `CollectFrames` accumulation short-circuit

**Decision: a new `RoundTrip` token type, opaque outside the package, carrying
a private completion channel. `LoopbackEnv.SendKeystroke` returns one;
`LoopbackEnv.WaitForEcho` consumes one. Neither reads the shared/accumulating
frame buffer that `Env.CollectFrames` uses.**

```go
// RoundTrip identifies one SendKeystroke → echo round trip in a loopback
// environment. Returned by LoopbackEnv.SendKeystroke; consumed exactly once
// by LoopbackEnv.WaitForEcho.
//
// [v1.2 correction — H1] done is chan []byte (was chan frame.OuterHeader).
// frame.OuterHeader carries no payload; the round-trip id rides in the
// payload bytes (encodeRTID/decodeRTID). WaitForEcho must return the
// delivered payload so callers can assert the delivered payload decodes to rt.id (AC-014 load-bearing part).
type RoundTrip struct {
    id   uint64
    done chan []byte // buffered 1; written by the downstream ticker goroutine
                     // on delivery; carries the full echo payload (including
                     // the 8-byte RT-ID suffix) — NOT an OuterHeader.
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
// elapses (fails t via t.Fatalf/b.Errorf on timeout — mirrors the existing
// Env.WaitForEcho failure convention). Unlike Env.WaitForEcho, which returns
// as soon as ANY frame is buffered on the session (correct for VP-033/034/
//036/039's "did anything arrive" semantics but wrong for VP-042's per-sample
// semantics), this reads only rt's own completion channel — a concurrent or
// stale round trip's frame cannot satisfy it.
//
// [v1.2 correction — H1] Returns ([]byte, bool): the delivered echo payload
// (or nil) and a present-flag (false on timeout). Callers assert:
//   payload, ok := lb.WaitForEcho(t, rt, timeout)
//   if !ok { t.Fatalf(...) }
//   id, ok2 := decodeRTID(payload); if !ok2 || id != rt.id { t.Errorf(...) }  // AC-014(b) [v1.4 B-1]
func (lb *LoopbackEnv) WaitForEcho(t testing.TB, rt RoundTrip, timeout time.Duration) (payload []byte, ok bool)
```

This sidesteps the accumulation bug entirely rather than patching
`CollectFrames`'s polling loop — no shared growing slice is in this path at
all. `Env.CollectFrames`/`Conn`/`Console.CollectFrames` are unchanged; their
accumulation semantics are correct for the VPs that use them (probes and
consoles legitimately want "everything received so far").

The correlation ID rides in the payload bytes (8-byte big-endian suffix,
`encodeRTID`/`decodeRTID` — trivial, package-private), not in
`frame.OuterHeader` — the outer header is a fixed 44-byte wire layout
(`internal/frame/frame.go:66`) with no spare field, so payload-embedding is
the only option that doesn't touch the wire format. This also means the
`loopbackSink` (Q4) doesn't need to know about correlation at all — it just
echoes bytes, matching how a real KeystrokeSink (tmux) works.

---

## Q6 — Goroutine / lifecycle plan

**Decision: two ticker goroutines (upstream, downstream), registered on the
*existing* `Env.wg`/`Env.closeCh` — no new WaitGroup or close-channel.**

`Env` already has `wg sync.WaitGroup`, `closeCh chan struct{}`, `closeOnce
sync.Once` (`testenv.go:434-436`), and `Env.Close()` already does
`closeOnce.Do(func() { close(closeCh); wg.Wait() })` (`testenv.go:561`),
registered via `t.Cleanup(e.Close)` in `newEnv` (`testenv.go:528`). Both
`AttachConsole` and `AttachProbe` already start goroutines this exact way
(`wg.Add(1)` before `go func() { defer wg.Done(); select { case <-closeCh:
return; ...} }()`, `testenv.go:664-680`). The loopback ticker goroutines
should use the identical pattern — same file, same package, same idiom
already used twice in this exact struct — rather than invent a second
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

**Wiring (v1.5 F-B-1):** the upstream ticker's `tickBody` is `d.onUpstreamTick`
and the downstream ticker's `tickBody` is `d.onDownstreamTick`. Each seam method
owns its half-channel's `Tick()` call under the corresponding per-direction mutex
(per §H2 and L256 binding rule), so every `Tick()` is mutex-guarded regardless
of how the goroutine invokes the body.

This is the same lifecycle shape as `cmd/switchboard/access.go:460`
(`startSweepTicker`) and `:500` (`startFramesDroppedTicker`) — the
production idiom for "ticker + WaitGroup + cancellation-channel" per
go.md rule 12's spirit and the S-4.00 wg-join clarification (ARCH-08
§6.5.1 obligations 3/6). The `hc` parameter is absent because the body
closure carries any half-channel reference; the lifecycle contract is
otherwise identical. No new Close() method is needed on `LoopbackEnv`;
`b.Cleanup(env.Close)` (already registered by `newEnv`) tears everything
down, and `wg.Wait()` blocks until both ticker goroutines have observed
`closeCh` and returned — deterministic, no leaked goroutines, matching the
existing `AttachConsole`/`AttachProbe` guarantee.

`NewLoopback` must validate `cfg.TickIntervalUpstream`/`TickIntervalDownstream`
against `halfchannel.MinTickInterval`/`MaxTickInterval` (5ms–50ms,
`halfchannel.go:44`) and `b.Fatalf` on an out-of-bounds value — matching the
existing fail-loud convention (`t.Fatalf` on illegal construction throughout
this file, e.g. `NewWithRouters` at `testenv.go:454`). Note VP-042's own
`downstreamInterval` (50ms) sits exactly at `MaxTickInterval` — legal, but
worth a comment at the validation site since it's the boundary case.

---

## Q7 — Synthetic path construction for `multipath.NewMultipath`

**Decision: two `paths.RankedPath`s per direction (4 total), each backed by
a `paths.NewPathTracker(initialRTTMS, alpha)` — no `OnProbe` calls needed.**

`paths.NewPathTracker` (`internal/paths/paths.go:115`) sets `active: true`
at construction — a fresh tracker is immediately eligible for `Rank`
(`paths.go:407`) without any probe history. `multipath.NewMultipath` requires
`[]paths.RankedPath` at construction (`multipath.go:215`), and
`multipath.Send` internally calls `paths.Rank` on every call (`multipath.go:252`)
— so testenv must import `internal/paths` **directly** to reference
`paths.RankedPath`/`paths.NewPathTracker`, even though the team's dispatch
only named `{halfchannel, arq, multipath}`. This is a Go-imposed transitive
requirement (referencing an exported type from an indirectly-imported package
requires a direct import), not a scope expansion I'm choosing — ARCH-08 v2.13
(already amended) includes `paths` at position 11 for exactly this reason.

```go
func newLoopbackPaths() []paths.RankedPath {
    return []paths.RankedPath{
        {ID: 1, Tracker: paths.NewPathTracker(1.0, 0.125)},
        {ID: 2, Tracker: paths.NewPathTracker(1.0, 0.125)},
    }
}
```

Two `*multipath.Multipath` instances are constructed — one per direction
(`upstreamMP`, `downstreamMP`) — each combining the pathSet used by
whichever side is the sender for that direction, and the `recvDedup` cache
used by whichever side is the receiver for that direction. This is the
minimal shape: one process, one loopback, no cross-process boundary means
there's no reason to split sender-state and receiver-state into separate
instances per endpoint.

---

## Q8 — New `internal/` package required?

**No new package.** All of this lands inside `internal/testenv` (existing
position 23, test-only composition root). ARCH-08 §6.4's new-package
protocol does not apply — this is an import-set expansion of an existing
package, the same class of change as v2.6 (`upstreamdial` pre-code
registration) and v2.8/v2.11 (testenv import-set corrections), already
amended into ARCH-08 v2.13 (this session).

---

## What VP-042's Benchmark Looks Like Against This

```go
func BenchmarkKeystrokeToEcho_P99(b *testing.B) {
    ctx := context.Background()
    lb := testenv.NewLoopback(ctx, b, testenv.LoopbackConfig{
        TickIntervalUpstream:   10 * time.Millisecond,
        TickIntervalDownstream: 50 * time.Millisecond,
    })
    sessionID := lb.CreateSession(b)

    latencies := make([]time.Duration, 0, 500)
    b.ResetTimer()
    for i := 0; i < 500; i++ {
        start := time.Now()
        rt := lb.SendKeystroke(b, sessionID, "x")
        payload, ok := lb.WaitForEcho(b, rt, 500*time.Millisecond) // [v1.2: returns ([]byte,bool)]
        if !ok { b.Fatalf("WaitForEcho timed out on sample %d", i) }
        _ = payload // AC-014: full test asserts the delivered payload decodes to rt.id (2-value decodeRTID)
        latencies = append(latencies, time.Since(start))
    }
    b.StopTimer()
    // ... sort, p99, b.ReportMetric, b.Errorf on breach — unchanged from
    // the existing VP-042 skeleton / keystroke_echo_bench_test.go pattern.
}
```

This is a small, deliberate divergence from the VP-042.md proof-harness
skeleton's exact call shape (`env.SendKeystroke` / `env.WaitForEcho` two-call
form with no token) — the skeleton predates the discovery that a token is
required to fix the accumulation bug (Q5). `test-writer`/`story-writer`
should treat the skeleton as directionally correct and this note's shape as
the binding API. Expected latency distribution: dominated by tick-cadence
wait (~half the upstream interval + half the downstream interval on average,
≈30ms; worst free-running case approaching the sum, ≈60ms, still comfortably
inside VP-042's 100ms ceiling — consistent with VP-042.md's own "~30s for 500
samples" estimate at these intervals, i.e. ~60ms/sample).

### Verification Method (AC-013)

**Decision: AC-013 verifies via the direct `go test -tags integration`
invocation the merged benchmark file's own doc comment already prescribes
— not a bare `go build`, and not `just bench`. `just bench` is explicitly
adjudicated OUT of AC-013's scope and stays unchanged.**

The benchmark lands inside
`internal/bench/keystroke_echo_testenv_bench_test.go` (merged on
`develop`, PR #121 @ `4c276d9`), which carries `//go:build integration` at
its top (the same convention `internal/testenv`'s own integration tests
already use, per that file's package comment). Two consequences for
AC-013's verification method:

- A **bare** `go build ./internal/bench/...` (no `-tags integration`)
  EXCLUDES this file from the build entirely — it verifies nothing about
  `BenchmarkKeystrokeToEcho_P99`. The compile check MUST carry the tag:
  `go build -tags integration ./internal/bench/...` (and/or
  `go vet -tags integration ./internal/bench/...`).
- `just bench` (`justfile:53-66`) runs `go test
  -bench=BenchmarkKeystrokeEcho_P99 -benchtime=1x -count=1
  ./internal/bench/` — no `-tags integration`, and it targets
  `BenchmarkKeystrokeEcho_P99` (no "To"), the untagged S-BL.BENCH
  lower-bound function defined in the SEPARATE file
  `keystroke_echo_bench_test.go`. It structurally cannot run
  `BenchmarkKeystrokeToEcho_P99` (WITH "To") — different build-tag gate,
  different function name. It runs today, unaffected by this story, and
  verifies nothing this story changes.

**AC-013's binding verification method is the direct invocation the
merged file's own doc comment already specifies** (its "Run with:"
block):

```sh
go test -tags integration -run '^$' -bench=BenchmarkKeystrokeToEcho_P99 \
    -benchtime=1x -count=1 ./internal/bench/
```

producing the p99 metric via `b.ReportMetric`, unchanged from the pattern
above.

**`just bench` adjudication: Option (a)** — leave `just bench` as the
tag-free lower-bound recipe; AC-013 verifies via the direct command
above, not a justfile recipe. Rationale, grounded in this note's own
design intent: the Story-Sizing Estimate already commits to "no new
CI/deployment surface... everything is additive inside
`internal/testenv`" for this story — a new or changed justfile recipe
would be exactly that kind of surface change, unmotivated by anything
else in this design. `just bench`'s purpose (S-BL.BENCH) is specifically
the tag-free lower-bound measurement; conflating it with the full-stack
integration benchmark would blur a distinction the merged file's own
package comment goes out of its way to preserve ("statistically
equivalent to `BenchmarkKeystrokeEcho_P99`... does NOT constitute
full-stack VP-042 evidence"). The merged file already documents its own
invocation convention independent of any justfile recipe (its "Run
with:" comment, verbatim above) — AC-013 citing that same command is the
minimal-diff, already-established path, not a new one this story would
have to invent. If a future story wants a convenience recipe for the
integration-tagged bench suite generally (not scoped to this one
function), that is a separable, additive change outside AC-013's scope.

---

## Non-Goals (Explicit)

This story does NOT implement:

- **Real network I/O or cross-process operation.** Both "paths" are
  synthetic, zero-added-latency, in-process function calls. No sockets, no
  serialization to wire bytes (no `outerassembler.Assemble`/`DecodeChannelHeader`
  round trip) — `multipath.Frame`/`halfchannel.ChannelFrame` are passed as Go
  structs, not encoded bytes. If a future VP wants byte-level wire-format
  coverage in a loopback harness, that is a separate, additive story (see
  Risks).
- **Simulated packet loss, retransmission, or TLPKTDROP.** `GapsToRetransmit`
  and `TLPKTDROP` are not called on any schedule. `internal/arqsend` and
  `internal/outerassembler`-based real retransmit dispatch are **not** added
  to testenv's import set by this story — they would only be needed for a
  loss-injection follow-on. `internal/arq`'s own pure-core unit tests already
  cover the reorder/gap/TLPKTDROP state machine; this benchmark's job is
  realistic tick-driven happy-path latency, not re-proving ARQ correctness.
- **`internal/replay` / upstream idempotent-window fidelity.** See Q1 — out
  of scope per ARCH-03, and not part of the requested import set.
- **Empty-tick wire dispatch.** Empty ticks are produced by `Tick()` (BC-2.01.002
  compliance) but not dispatched over multipath in this harness — they carry
  no round-trip token and would not change the measured property. A
  full-fidelity extension could add this later at zero cost to the measured
  p99.
- **Changing `Env.SendKeystroke`/`Env.CollectFrames`/`Env.WaitForEcho`.**
  These remain exactly as they are for the 10 other VPs that use them.
- **A VP-042 verification_lock flip inside this story.** This story delivers
  the harness; locking VP-042 is a separate, subsequent act (run the
  benchmark, record evidence, update VP-042.md's frontmatter) once the
  harness lands and the architect/adversary have signed off on the `OnAck`
  call-contract decision in Q4 (see Risks).

---

## Package Impact Summary

| Package | Change | ARCH-08 §6.4 required? |
|---------|--------|------------------------|
| `internal/testenv` | New `loopbackDriver` type; `LoopbackEnv.SendKeystroke`/`WaitForEcho`/`CreateSession`/`RoundTrip`; `NewLoopback` wires halfchannel/arq/multipath/paths instead of discarding `LoopbackConfig` | No (existing package) — but import-set expansion requires the §6.4-equivalent pre-code registration already done in ARCH-08 v2.13 (this session) |
| `internal/halfchannel` | None — read-only consumer (`New`, `Tick`, `Enqueue`) | No |
| `internal/arq` | None — read-only consumer (`New`, `EnqueueSend`, `OnAck`); first production(-adjacent) call site for `OnAck` (see Risks) | No |
| `internal/multipath` | None — read-only consumer (`NewMultipath`, `Send`, `Receive`) | No |
| `internal/paths` | None — read-only consumer (`NewPathTracker`, `RankedPath`) | No |
| `internal/bench` | `keystroke_echo_testenv_bench_test.go` (merged on `develop`, PR #121 @ `4c276d9`, `//go:build integration`-tagged) updated to the token-based two-call shape; package comment's "lower bound only" framing retired once the full stack lands | No |

**No new `internal/` package. ARCH-08 registration is the import-set
amendment already applied (v2.13, DRAFT/PROSPECTIVE, this session) — it
becomes final at this story's merge per the same machine-verification
protocol used for every prior testenv import-set change (v2.5, v2.8, v2.11).**

---

## Story-Sizing Estimate

**Estimate: 5–8 points (medium-large single story, or split into two:
"tick-driven halfchannel + multipath wiring" then "arq + round-trip-token
API").**

Rationale:
- The tick-driving mechanism (Q6) is low-risk and small — it's a direct copy
  of an idiom already used twice in the same file and twice more in
  `cmd/switchboard/access.go`.
- The multipath wiring (Q3, Q7) is low-risk — `Send`/`Receive`/`NewMultipath`
  are small, well-documented, already-tested pure APIs; the synthetic path
  construction is a few lines.
- The ARQ wiring (Q4) is the size and risk driver: it commits to a call
  contract (`OnAck`'s `ackSeq` semantics) that has no existing production
  precedent to copy, and that commitment should be reviewed (architect
  sign-off or an adversarial pass) before implementer treats it as settled —
  see Risks. If that review surfaces a different call contract, the
  downstream half of this design changes, not the upstream half or the
  tick-driving mechanism.
- The round-trip-token API (Q5) touches the merged
  `internal/bench/keystroke_echo_testenv_bench_test.go` (`develop`, PR #121
  @ `4c276d9`) and VP-042.md's harness skeleton, both of which need
  updating to the new two-call-with-token shape — small but real fan-out.
- No new package, no CI/deployment surface, no cross-cutting production code
  change — everything is additive inside `internal/testenv` plus the ARCH-08
  spec amendment already applied.

---

## Risks / Open Questions for story-writer to Encode as ACs

1. **`arq.OnAck` call-contract (Q4) needs explicit sign-off before
   implementation, not just transcription.** This design proposes a specific,
   internally-consistent convention (`ackSeq` = locally-observed frame's own
   `ChanSeq`, zero SACK in the no-loss happy path) because no production call
   site exists to copy. Story-writer should add an AC requiring either (a) an
   architect placement-note addendum confirming this contract before
   `dev-story` begins, or (b) a fast adversarial pass on this note
   specifically targeting Q4 before implementation starts. Getting this wrong
   doesn't break VP-042's measured number (the happy path is forgiving) but
   would misinform whatever *next* story tries to reuse `OnAck` for a real
   ACK/SACK path (e.g. a future loss-injection VP).
2. **`PathTracker.IsActive()` initial-state dependency.** This design relies
   on `NewPathTracker` defaulting `active: true` (confirmed by reading
   `paths.go:115-124` in this session) so no `OnProbe` warm-up is needed.
   Implementer should add a one-line assertion/test confirming this rather
   than re-deriving it from source at implementation time — cheap insurance
   against a future `paths` package change silently breaking the loopback's
   path activation.
3. **`RoundTrip.done` channel buffering and double-delivery.** If a
   `WaitForEcho` call times out and the corresponding entry is never read
   from `driver.pending`, the downstream ticker's completion-signal send
   (`ch <- frameFor(payload)`) would block forever unless `done` is buffered
   (proposed: buffer 1). Story-writer should add an explicit AC for the
   timeout-then-late-arrival case: the driver must still `delete` the pending
   entry and not leak it, and the buffered send must not block the ticker
   goroutine even if nobody ever reads it.
4. **Bounded `pending` map growth under a hung round trip.** If
   `WaitForEcho` is never called for a `RoundTrip` (test bug), `driver.pending`
   accumulates permanently until `Close`. This is a `t.Fatalf`-shaped
   programmer-error case, not a production concern (testenv is test-only) — a
   docstring warning is likely sufficient, but story-writer should decide
   whether it warrants an active safeguard (e.g. `t.Cleanup` asserting the map
   is empty) or is out of scope.
5. **This story does not itself flip VP-042's `verification_lock`.** See
   Non-Goals. Story-writer should scope the story to "harness lands, and is
   run once manually to produce evidence for the VP-042.md changelog" —
   the *lock* decision (editing `verification_lock: true`) is a separate
   PO/architect act per existing VP lifecycle convention (compare how VP-042's
   own history table already distinguishes "audited"/"partial evidence" from
   a lock flip).
6. **VP-042.md's Proof Harness Skeleton is superseded by this story's
   binding API shape (Q5; see "What VP-042's Benchmark Looks Like Against
   This," which already notes the divergence).** Story-writer should add a
   File-List obligation updating VP-042.md's Proof Harness Skeleton section
   to the binding two-call token shape this note specifies
   (`SendKeystroke` → `RoundTrip`; `WaitForEcho(t, rt, timeout) ([]byte,
   bool)`), so a future reader of VP-042.md isn't misled by a skeleton the
   shipped code no longer matches.

---

## Q4 Addendum — AC-001 Sign-off (2026-07-12)

**Scope:** discharges AC-001 (S-BL.LOOPBACK-FULLSTACK Risk 1) — the
pre-implementation review of the `arq.OnAck` call-contract proposed in Q4,
required before `dev-story` may treat that contract as settled (Risk 1,
option (a): "an architect placement-note addendum confirming this contract
before `dev-story` begins").

### Verdict: REVISED

The `ackSeq`/SACK **value convention** the note proposes — `ackSeq` = the
just-arrived frame's own `ChanSeq`, SACK bitmap all-zero in the no-loss
happy path — is **CONFIRMED correct**. But the **instance topology** the
note's Q4 code blocks assume (`driver.arqServer` for `EnqueueSend`,
`driver.arqClient` for `OnAck`, as two separate `*arq.ARQ` values) is a
structural defect that would make the harness non-functional: `OnAck`
would return zero delivered payloads on every call, `driver.pending`
entries would never resolve, and `WaitForEcho` would time out on every
round trip — not a subtle correctness gap, a hard benchmark failure. This
contradicts the note's own Risk 1 framing ("getting this wrong doesn't
break VP-042's measured number") for this specific failure mode; it does
break the measured number, because no measurement would ever complete.

**Required revision:** collapse `driver.arqServer` and `driver.arqClient`
into a single shared `*arq.ARQ` field (e.g. `driver.downstreamARQ`). The
downstream ticker's existing call sequence is otherwise unchanged: on each
tick that produces a data frame, call `EnqueueSend(f.ChanSeq, f.Payload,
now)` on that instance, then — after `Send` → `deliverDownstream` →
`Receive` complete synchronously within the same tick and goroutine (Q3's
own established rationale for not spawning per-path goroutines already
guarantees this ordering) — call `OnAck(chanSeq, zeroSACK)` on
the SAME instance, where `chanSeq` is captured as `chanSeq := f.ChanSeq`
from the originating `ChannelFrame` before `toMPFrame` is called
[v1.3 C-F2: corrected from `OnAck(mpFrame.ChanSeq(), zeroSACK)` —
`multipath.Frame` has no `ChanSeq`; superseded in full by v1.2 §B1].
With that fix, the proposed `ackSeq`/SACK convention is
binding as originally written.

### Reasoning trail

**1. `OnAck`'s payload-delivery mechanism is instance-local and
EnqueueSend-dependent, not a generic "process an incoming frame" call.**

`OnAck` (`internal/arq/arq.go:201`) delivers payloads via two paths, both
scoped to the receiver's own instance state:

- Step 1 (`arq.go:235-244`) walks `nextExpected+1..ackSeq` and calls
  `a.payloadFor(seq)` (`arq.go:291-301`) for each. `payloadFor` checks
  `a.reorderBuf` then `a.inFlight` — **and nothing else**. There is no
  third source.
- Step 2 (`arq.go:250-265`) buffers SACK-flagged out-of-order sequences
  into `a.reorderBuf`, but only by cloning from `a.inFlight[seq]`
  (`arq.go:255-261`) — if `inFlight` doesn't have the entry, nothing is
  buffered, silently.

`a.inFlight` is populated in exactly one place in the entire package:
`EnqueueSend` (`arq.go:339-348`, `a.inFlight[seq] = &inFlightFrame{...}`).
`a.reorderBuf` is populated in exactly one place: `OnAck`'s own Step 2,
itself sourced from `inFlight`. There is no method that lets a caller
inject a payload for `OnAck` to return other than a prior `EnqueueSend` on
that same `*ARQ` value. A fresh instance that has never seen `EnqueueSend`
has both maps permanently empty; `OnAck` on it will return `(nil, nil)`
for every call, forever, regardless of what `ackSeq` value is supplied —
`nextExpected` still advances (masking the problem: no error is returned),
but `toDeliver` never gets anything appended.

Traced through the design's own pseudocode (Q4): `driver.arqClient` is
never the target of any `EnqueueSend` call anywhere in Q3 or Q4 — only
`driver.arqServer` receives `EnqueueSend`. Under the two-instance shape as
written, `driver.arqClient.OnAck(...)` returns an empty slice on every
downstream tick; the `for each payload in delivered` loop
(placement note, downstream-flow pseudocode) never executes;
`driver.pending[id]` is never resolved; every `WaitForEcho` call times out.

**2. All existing evidence in this codebase uses ONE shared instance for
both "record what I sent" and "process what came back," never two.**

- Every test in `internal/arq/arq_test.go` that exercises `OnAck` calls
  `EnqueueSend` on the identical `*ARQ` receiver variable first — e.g.
  `TestARQ_OnAck_NoDuplicateDelivery` (`arq_test.go:114-141`:
  `a.EnqueueSend(1, ...)` then `a.OnAck(1, ...)` on the same `a`), the
  SACK-buffering test (`arq_test.go:186-228`: three `EnqueueSend` calls
  then three `OnAck` calls, same `a`), the failover-resync test
  (`arq_test.go:835-868`), the property-based fuzz test
  (`arq_test.go:900-914`: `EnqueueSend` then `OnAck` inside the same loop
  iteration, same `a`). No test anywhere constructs two `*ARQ` values
  where one receives `EnqueueSend` and a different one receives `OnAck`
  for the same sequence space.
- `internal/arqsend` — the only current production consumer of `*arq.ARQ`,
  and the closest thing to an existing calling convention — is explicit
  that a `Retransmitter` "holds an `*arq.ARQ` handle" (singular,
  `arqsend.go:9`, `:66`, `:100`) and that "Two Retransmitters over
  independent ARQ handles are independent and may run in parallel"
  (`arqsend.go:26-29`) — i.e. the unit of sharing is one handle per flow,
  not one handle per role. `arqsend` only exercises the
  `EnqueueSend`/`PayloadForInFlight`/`RemoveInFlight` subset of that SAME
  handle; the natural (and only structurally workable) place for a future
  production `OnAck` call, once a real console's piggybacked ACK/SACK
  arrives over the wire (F-023), is on that SAME handle — not a second one.
- The `ARQ` struct itself (`arq.go:118-146`) carries both `inFlight`
  (sender bookkeeping) and `nextExpected`/`reorderBuf` (delivery-pointer
  bookkeeping) as fields of ONE struct. This is a unified per-flow state
  machine, not two role-scoped state machines that happen to share a Go
  type.

The package doc's "Receiver role (console): OnAck..." / "Sender role
(access node): TLPKTDROP..." framing (`arq.go:111-115`) describes what
each *method* accomplishes conceptually within the protocol (advancing the
receiver's confirmed-delivery state vs. terminating an overdue send) — it
is not evidence that two separate instances are intended. Given (1) above,
it cannot be: `OnAck` structurally cannot do its job without the same
instance's `EnqueueSend` history.

**3. ARCH-03 is consistent with this reading once "the caller's tick loop
forwards them to the terminal" is read as this harness's stand-in, not a
literal physical-terminal requirement.**

ARCH-03 §Downstream ARQ (`ARCH-03-routing-engine.md:163-176`) describes
the *protocol* in terms of two conceptual roles (access-node SendBuffer,
console RecvBuffer) but cites a single package, `internal/arq`, for the
whole section — consistent with one Go state machine modeling the
access-node's local view of both roles (what it sent, what's now
confirmed via the console's real piggybacked ACK/SACK). The "delivery
contract" note (`ARCH-03-routing-engine.md:195-201`, "OnAck returns
deliverable frames synchronously... the caller's tick loop forwards them
to the terminal") is written from production's perspective, where the
real console is a separate physical endpoint that does its own
(un-modeled-here) receive-side buffering; in THIS harness, `driver.pending`
+ `WaitForEcho` is the stand-in for "the terminal," and it is fed
correctly once `OnAck` is called on the same instance that sent the frame.

### Answers to the three specific checks requested

**(a) Is `ackSeq` cumulative-highest-in-order vs per-frame semantically
distinguishable in `arq.go`, and does the proposed convention match?**

Distinguishable, and the implementation is unambiguously cumulative:
`OnAck`'s Step 1 loop is `for seq := a.nextExpected + 1; seq <= ackSeq;
seq++` (`arq.go:235`) — `ackSeq` is a watermark, not a bare per-frame
tag, and the loop explicitly handles multi-frame gaps in one call
(BC-2.02.005 invariant 4, `arq.go:224-229`). The design's "this frame's
own `ChanSeq`" framing is a *value choice* that is mathematically
equivalent to "advance the cumulative watermark by exactly one" — but only
under this harness's own guarantees (single downstream producer, strictly
increasing `ChanSeq`, one frame per tick, no simulated loss/reordering —
Non-Goals). That equivalence claim in the note is correct. It requires the
single-shared-instance fix above to matter at all — with two instances,
`nextExpected` on `arqClient` advances in a vacuum with no payload ever
recoverable.

**(b) Does zero-SACK ever mis-train the window state?**

No, given the fix. Because `ackSeq == nextExpected+1` on every call in
this topology, Step 3's reorder-buffer flush (`arq.go:271-283`) is always
a no-op — nothing was ever buffered by Step 2, since there is never a
genuine out-of-order arrival to represent with SACK bits — and Step 2
itself is a correct no-op against an all-zero bitmap. This matches ground
truth for a genuinely loss-free, strictly in-order stream. Caveat already
correctly named in the note's Risk 1: this zero-SACK convention is valid
*because* Non-Goals excludes loss/reordering, not because it is a
general-purpose convention — a future loss-injection VP reusing this call
site must replace it with a bitmap reflecting true reorder state.

**(c) Any interaction with RULING-003 window validation at the boundary
(first frame, wraparound)?**

None under the PREFERRED `CreateSession`-time start, subject to the
timing caveat in §M2: `ErrAckOutOfWindow`'s guard (`ackSeq - a.nextExpected >
sackWindowSize`, `arq.go:220`; RULING-003, `ARCH-03-routing-engine.md:
203-209`) is safe as long as fewer than 64 empty ticks precede the first
data frame (64 × 50ms ≈ 3.2s at the standard interval). For the VP-042
benchmark (CreateSession immediately followed by the send loop) this is
trivially satisfied. [v1.4 B-2 correction: the first downstream ChanSeq is
NOT guaranteed to be 1 under CreateSession-time start — it equals the number
of empty ticks elapsed, not a constant. "chanSeq = 1" holds only for the
first-SendKeystroke option where the ticker starts immediately before the
first data tick.] 32-bit `ChanSeq` wraparound is explicitly out of MVP scope
(`arq.go:231-234`, RULING-001 §R2, ~49–497-day wrap interval) and structurally
unreachable within a 500-sample benchmark at millisecond-scale tick intervals.

### Constraints the implementer must observe

1. **One shared `*arq.ARQ` instance for the downstream direction.**
   Rename `driver.arqServer`/`driver.arqClient` to a single field (e.g.
   `driver.downstreamARQ`). `EnqueueSend` and `OnAck` for a given `ChanSeq`
   MUST be called on that same instance, in that order, within the same
   downstream-ticker goroutine tick.
2. **Do not reuse the always-zero-SACK convention outside this harness's
   Non-Goals envelope.** It is correct here because loss/reordering are
   out of scope; a future loss-injection story reusing `OnAck` must
   compute a real bitmap.
3. **Add a regression guard against reintroducing the two-instance
   shape.** A short test asserting the downstream driver has exactly one
   `*arq.ARQ` field (or that `EnqueueSend`/`OnAck` observably operate on
   shared state — e.g. a round trip actually completing at all) is
   sufficient; this is cheap insurance given the failure mode is silent
   (no error, just permanently empty delivery).
4. **Story-writer scope:** amend Risk 1 / AC-001 wording to record this
   addendum's verdict (REVISED, not simple CONFIRMED) and to bind the
   implementer to the shared-instance shape — the current story-writer
   input (Q4 as originally written) still shows the two-instance code and
   would mislead an implementer working from the code blocks alone without
   this addendum in hand.

**Files consulted for this sign-off:** `internal/arq/arq.go` (full file),
`internal/arq/arq_test.go` (OnAck/EnqueueSend call sites),
`internal/arqsend/arqsend.go` (package doc + `Retransmitter` composition),
`.factory/specs/architecture/ARCH-03-routing-engine.md:155-220`
(§Upstream Idempotent Replay tail, §Downstream ARQ, ADR-005 lead-in). No
production or test code anywhere in the repo constructs two `*arq.ARQ`
instances in a sender/receiver split — confirmed via
`grep -rn "arqServer\|arqClient\|arq\.New("` across the tree.

---

## v1.2 Design Repair Addendum (2026-07-22)

**Scope:** Discharges the ARCHITECT-owned findings from the adversarial
spec-review (`spec-review-2026-07-22.md`): B1 (BLOCKER), H1 (HIGH), H2 (HIGH),
H3 (HIGH), M2 (MED), M4 (MED). Verdict on each finding follows. In-place
pseudocode corrections are annotated in the sections above.

**Source verification pass (PAT-04):** all ground-truth claims in the review
were re-verified against shipped source before designing. No claim was wrong.
Files read: `internal/multipath/multipath.go` (Frame struct, L38-43),
`internal/halfchannel/halfchannel.go` (ChannelFrame L64-69, concurrency doc
L89-90, Tick L117-137, Enqueue L143-154), `internal/frame/frame.go`
(OuterHeader L66-84), `internal/session/upstream.go` (SendKeystroke L276-300),
`internal/testenv/testenv.go` (attach idiom L646-648, NewLoopback L383-386),
`internal/arq/arq.go` (OnAck L201, EnqueueSend L339, ErrAckOutOfWindow L66-71).

---

### §B1 — ChanSeq threading (BLOCKER)

**Defect:** The Q4 downstream pseudocode called
`driver.downstreamARQ.OnAck(mpFrame.ChanSeq(), zeroSACK)`. `multipath.Frame`
has exactly two fields: `OuterHeader [44]byte` and `Payload []byte` — no
`ChanSeq` method or field. Won't compile. `ChanSeq` exists only on the
originating `halfchannel.ChannelFrame` (field `ChanSeq uint32`), which is in
scope at the top of the downstream ticker body (`f := driver.downstreamHC.Tick()`)
but is not transmitted through `toMPFrame`/`multipath.Frame` (the multipath
encode/decode round trip is a declared Non-Goal; story declares multipath a
read-only consumer with no new fields added).

**Decision:** Capture `f.ChanSeq` into a local variable immediately after the
`Tick()` call, before `toMPFrame(f)` is called, and pass that local to `OnAck`.
This is valid because:

1. The downstream ticker goroutine is single-goroutine for this half-channel
   (H2 fix, below), so `f` is not modified after `Tick()` returns.
2. `deliverDownstream` runs synchronously within the same goroutine tick (Q3
   rationale — no per-path goroutine is spawned), so `chanSeq` is still valid
   when `OnAck` is called at the end of the same tick.
3. No field is added to `multipath.Frame` — the Non-Goal and read-only consumer
   contract are preserved.

**Corrected tick body (canonical form):**

```go
f := driver.downstreamHC.Tick()
if f.FrameType == frame.FrameTypeData {
    chanSeq := f.ChanSeq          // capture BEFORE toMPFrame — Frame has no ChanSeq
    driver.downstreamARQ.EnqueueSend(chanSeq, f.Payload, time.Now())
    driver.downstreamMP.Send(toMPFrame(f), driver.deliverDownstream)
    // deliverDownstream runs synchronously (Q3); chanSeq still valid here
    delivered, err := driver.downstreamARQ.OnAck(chanSeq, zeroSACK)
    if err != nil {
        driver.failLoud(fmt.Errorf("downstreamARQ.OnAck seq=%d: %w", chanSeq, err))
        return
    }
    for _, payload := range delivered {
        id, ok := decodeRTID(payload)     // [v1.4 B-1] 2-value; id=0 on !ok never matches a pending key
        if !ok { continue }               // payload too short; skip
        driver.mu.Lock()
        ch := driver.pending[id]
        delete(driver.pending, id)
        driver.mu.Unlock()
        if ch != nil {
            ch <- payload   // chan []byte — delivers echo payload to WaitForEcho
        }
    }
}
```

**`toMPFrame` obligation (ties to M4):** `toMPFrame(f halfchannel.ChannelFrame) multipath.Frame`
copies `f.Payload` into `multipath.Frame.Payload` and fills `OuterHeader` from
channel metadata. It does NOT carry `ChanSeq` into the returned struct — this is
intentional and correct; `ChanSeq` is captured separately before this call.

---

### §H1 — Harness API shape for AC-014

**Defect:** `WaitForEcho` was void-returning; `RoundTrip.done` was
`chan frame.OuterHeader`. `frame.OuterHeader` carries no payload (its fields are
Version/FrameType/PayloadLen/SVTNID/SrcAddr/DstAddr/HMACTag — see
`frame.go:66-84`). The round-trip id rides in payload bytes via `encodeRTID`/
`decodeRTID`. AC-014(b) requires the caller to assert the delivered payload
decodes to `rt.id`. This is impossible with a void-returning function and a
payload-less carrier.

**Decision:** The following type signatures are binding for this story:

```go
// RoundTrip identifies one SendKeystroke → echo round trip.
// The done channel carries the delivered echo payload ([]byte), which includes
// the 8-byte RT-ID suffix that WaitForEcho's caller uses for AC-014 assertion.
type RoundTrip struct {
    id   uint64
    done chan []byte // buffered 1
}

// WaitForEcho blocks until the echo for rt arrives or timeout elapses.
// Returns (payload, true) on delivery; (nil, false) on timeout.
// On timeout the test should call t.Fatalf — WaitForEcho does NOT call
// t.Fatalf itself so that benchmark callers can record the miss and continue
// rather than abort the run (matches the existing Env.WaitForEcho pattern).
// The returned payload is the verbatim echo payload; AC-014 callers assert:
//   id, ok2 := decodeRTID(payload); if !ok2 || id != rt.id { t.Errorf(...) }  // [v1.4 B-1]
func (lb *LoopbackEnv) WaitForEcho(t testing.TB, rt RoundTrip, timeout time.Duration) (payload []byte, ok bool)
```

`RoundTrip.done` is buffered 1 so that a late delivery after a timeout does not
block the downstream ticker goroutine (existing Risk 3 in the note, unchanged).
The `frameFor` helper from the v1.1 Q4 pseudocode is eliminated — it was only
needed to bridge payload to `frame.OuterHeader`; with `chan []byte` the payload
passes directly, no bridge needed. `frameFor` is removed from M4's helper
enumeration.

---

### §H2 — HalfChannel synchronization

**Defect:** `halfchannel.HalfChannel` is explicitly not safe for concurrent use
(doc: "Tick() and Enqueue() [must be] called from a single goroutine or under
external synchronisation", `halfchannel.go:89-90`). The v1.1 design accessed
each half-channel from two goroutines: `upstreamHC` was Enqueued from the
test goroutine (`SendKeystroke`) while the upstream ticker goroutine called
`Tick()`; `downstreamHC` was Enqueued from the upstream ticker goroutine
(`loopbackSink.SendInput`) while the downstream ticker goroutine called `Tick()`.
`driver.mu` guards only `driver.pending`, not the half-channels. `go test -race`
will fail.

**Decision:** Per-direction mutex serializing `Enqueue` and `Tick` on the same
half-channel.

Rationale for mutex over channel-funnel: the two half-channels are accessed from
at most two goroutines each (test + upstream ticker for upstream; upstream ticker
+ downstream ticker for downstream). A `sync.Mutex` per half-channel wrapper is
three lines at the call site and zero structural change to `loopbackDriver`. A
buffered-channel funnel (redirecting enqueues into the ticker goroutine) is a
more significant structural change: it requires a dedicated enqueue channel, a
select loop in the ticker, and careful sizing; the benefit (true single-goroutine
access) is not needed here because the lock window is tiny (a slice append for
`Enqueue`; no I/O). The mutex is the simpler, less error-prone choice.

**Concrete shape:**

```go
type loopbackDriver struct {
    upstreamHCMu   sync.Mutex           // serializes upstreamHC.Enqueue + Tick
    upstreamHC     *halfchannel.HalfChannel
    downstreamHCMu sync.Mutex           // serializes downstreamHC.Enqueue + Tick
    downstreamHC   *halfchannel.HalfChannel
    // ... other fields unchanged
}

// In SendKeystroke (test goroutine):
func (lb *LoopbackEnv) SendKeystroke(...) RoundTrip {
    // ...
    lb.driver.upstreamHCMu.Lock()
    err := lb.driver.upstreamHC.Enqueue(payload)
    lb.driver.upstreamHCMu.Unlock()
    // ...
}

// Upstream ticker tickBody (wired as startLoopbackTicker(env, upstreamInterval, d.onUpstreamTick)):
func (d *loopbackDriver) onUpstreamTick() {
    d.upstreamHCMu.Lock()
    f := d.upstreamHC.Tick()
    d.upstreamHCMu.Unlock()
    // ...
}

// In loopbackSink.SendInput (called from upstream ticker goroutine):
func (s *loopbackSink) SendInput(payload []byte) error {
    s.driver.downstreamHCMu.Lock()
    err := s.driver.downstreamHC.Enqueue(payload)
    s.driver.downstreamHCMu.Unlock()
    return err
}

// Downstream ticker tickBody (wired as startLoopbackTicker(env, downstreamInterval, d.onDownstreamTick)):
func (d *loopbackDriver) onDownstreamTick() {
    d.downstreamHCMu.Lock()
    f := d.downstreamHC.Tick()
    d.downstreamHCMu.Unlock()
    // ...
}
```

**AC obligation:** The story-writer must add an acceptance criterion asserting
`just test-race` passes (i.e. the CI race detector finds no data races in the
loopback driver). This AC has no existing number in v1.1; the story-writer
assigns it.

---

### §H3 — Console provisioning sequence

**Defect:** `AccessNode.SendKeystroke` returns `ErrConsoleNotFound` unless the
console key is registered and attached. The v1.1 design references the
`AttachConsole` pattern but never specifies `RegisterKey`+`Attach`+authorizer
`Allow` for the loopback console specifically. Without these, every upstream
keystroke silently fails (`ErrConsoleNotFound` returned by `SendKeystroke`,
never surfaced), AC-004/005/006/014 happy paths all time out.

**Decision:** Console provisioning happens ONLY in `CreateSession` — never in
the `loopbackDriver` constructor. [v1.8 F-LENSB-B-01] This is a hard boundary,
not a style preference: AC-017 (§M2) requires building the driver and calling
`SendKeystroke` + `onUpstreamTick()` synchronously BEFORE `CreateSession` runs,
and observing `ErrConsoleNotFound` (via `failLoud`) as the result. If the
`loopbackDriver` constructor provisioned the console eagerly, that pre-
`CreateSession` `SendKeystroke` call would already have an attached console
available by the time `onUpstreamTick()` processes it — `accessNode.SendKeystroke`
would SUCCEED, `failLoud` would never fire, and AC-017's step-4 assertion would
fail. Permitting construction-time provisioning as an implementation choice
(as prior versions of this note did) is therefore incompatible with AC-017 and
is withdrawn.

**Driver lifecycle pin [v1.8 F-LENSB-B-01]:** the `loopbackDriver` constructor
(invoked once, from `NewLoopback`, before `CreateSession` is ever called)
builds ALL of the following, fully initialized and immediately usable, but
with the console UN-PROVISIONED (no `Publish`/`RegisterKey`/`Attach` has run):

- The `Publisher`/`SessionAuth`/`AccessNode` triple (Q2).
- BOTH `*multipath.Multipath` instances, `upstreamMP` and `downstreamMP` (Q7).
- BOTH `*halfchannel.HalfChannel` instances, `upstreamHC` and `downstreamHC` (§H2).

Concretely: `SendKeystroke` (enqueues into `upstreamHC`), `onUpstreamTick()`
(dequeues from `upstreamHC`, drives `upstreamMP.Send`/`Receive`, then calls
`accessNode.SendKeystroke`), and `onDownstreamTick()` are ALL safely callable
on a freshly-constructed, pre-`CreateSession` driver — none of them nil-deref,
because `upstreamMP`/`downstreamMP`/`upstreamHC`/`downstreamHC` are built AT
CONSTRUCTION, not lazily at `CreateSession`. Only the console's session-level
authorization state (Publish/RegisterKey/Attach, below) is deferred to
`CreateSession`. `CreateSession` therefore does exactly two things: (1) the
console-provisioning sequence below, and (2) starting BOTH ticker goroutines
— downstream and upstream (§M2's preferred `CreateSession`-time start,
[v1.10 F-LENSB-01] now symmetric for both tickers) — it does NOT construct
the driver, the multipath instances, or the half-channels; those already exist
from `NewLoopback`.

This matches the shipped `testenv.AttachConsole` idiom (`testenv.go:646-648`):

```go
// Session provisioning — called ONLY from CreateSession, never from the
// loopbackDriver constructor (the driver, its AccessNode, and both
// multipath/half-channel pairs already exist and are usable — see
// "Driver lifecycle pin" above; only console attachment is deferred here):
loopbackConsoleKey := driver.env.newConsoleKey()  // opaque ConsoleKey

// [v1.3 B-F1] Publish into the driver's OWN dedicated Publisher BEFORE Attach.
// AccessNode.Attach calls pub.Get(sessionName) as its first gate
// (internal/session/upstream.go:203-208); if the session is not published,
// Attach returns ErrSessionNotFound and the driver's t.Fatalf fires at
// construction — the happy path is unreachable.
// The loopback driver builds its own Publisher/SessionAuth/AccessNode triple
// (Q2), so this publishes into driver.pub, NOT env.defaultShard's publisher.
// Publisher.Publish signature: func (p *Publisher) Publish(sessionName string) error
if err := sh.pub.Publish(sessionName); err != nil {
    t.Fatalf("loopbackDriver: Publish session %q: %v", sessionName, err)
}
sh.auth.RegisterKey(sessionName, loopbackConsoleKey, session.RoleFull)
downstream, _, err := sh.access.Attach(loopbackConsoleKey, sessionName)
if err != nil {
    t.Fatalf("loopbackDriver: Attach loopback console: %v", err)
}
// The authorizer (sh.auth) already covers loopbackConsoleKey via RegisterKey;
// no additional Allow configuration is needed — RegisterKey with RoleFull
// is what makes sh.auth.Allow(loopbackConsoleKey, sessionName, _) return nil.
_ = downstream  // downstream channel not used — echo delivery is via loopbackSink
```

**Complete corrected provisioning sequence (v1.3):**

1. `sh.pub.Publish(sessionName)` — publish into the driver's own dedicated Publisher so `Attach`'s `pub.Get` gate passes.
2. `sh.auth.RegisterKey(sessionName, loopbackConsoleKey, session.RoleFull)` — register the console key so `SendKeystroke`'s authorizer check passes.
3. `sh.access.Attach(loopbackConsoleKey, sessionName)` — attach the console to the session; succeeds now that the Publisher has the session.

Steps 1–3 MUST appear in this order. `RegisterKey` before `Attach` was already required (from v1.2); `Publish` before `Attach` is the new v1.3 requirement. [v1.8] Steps 1–3 are exactly the console-provisioning work `CreateSession` performs; `CreateSession` additionally starts both ticker goroutines — downstream and upstream ([v1.10 F-LENSB-01] §M2's preferred `CreateSession`-time start, now symmetric for both) — as a separate concern unordered relative to 1–3 — see "Driver lifecycle pin" above.

`loopbackConsoleKey` is stored on `loopbackDriver` and passed to
`driver.accessNode.SendKeystroke(loopbackConsoleKey, sessionName, payload)` in
the upstream delivery callback (Q3 `accessNode.SendKeystroke` line).

**`sessionName` storage/timing pin [v1.9 R6 LOW fix, Lens B O-1; v1.10
F-LENSB-02 fault-attribution correction]:**
`sessionName` is likewise stored as a `loopbackDriver` field (e.g.
`driver.sessionName`), set once in `CreateSession` alongside
`loopbackConsoleKey`, at the same point — before Steps 1–3 above run (step 1,
`sh.pub.Publish(sessionName)`, is `sessionName`'s first use, so the field is
populated no later than immediately before that call). Consistent with the
"Driver lifecycle pin" above (construction leaves the driver un-provisioned;
`CreateSession` provisions it), `driver.sessionName` holds its zero value
(`""`) at any point before `CreateSession` has run. [v1.10 F-LENSB-02] The
PRIMARY cause of AC-017's `ErrConsoleNotFound` is the UN-ATTACHED, zero-value
`loopbackConsoleKey` — not the empty `sessionName`. `AccessNode.SendKeystroke`
(`internal/session/upstream.go:276-301`) takes `sinkMu`, then at line 288
looks up `attached, ok := a.consoles.Session(key)`; because `loopbackConsoleKey`
has never been `Attach`ed (that only happens inside `CreateSession`'s Steps
1–3, which have not run before AC-017's call), this lookup returns
`ok == false`, and the console-attachment gate at lines 289-290 returns
`ErrConsoleNotFound` IMMEDIATELY — before `sessionName` is ever read. The
`attached != sessionName` comparison at line 292 (which would return
`ErrSessionMismatch`) is unreached on this path. The empty `sessionName` is
therefore a REDUNDANT second guarantee, not the operative cause: had some
OTHER session already been published while `loopbackConsoleKey` remained
unattached, the same `ErrConsoleNotFound` would fire at lines 289-290 for the
same reason; had `loopbackConsoleKey` instead been attached to a real session
while `sessionName` stayed `""`, the attachment gate would pass but the
`attached != ""` comparison at line 292 would then return `ErrSessionMismatch`
— either way a loud error, never a silent success. For the happy-path ACs
(AC-004/005/006 and others), `driver.sessionName` is populated and stable by
the time any upstream tick can reach the delivery callback, because
`CreateSession` sets it before returning and no `SendKeystroke` call that
matters to those ACs is made before `CreateSession` completes.

Note: the `downstream` channel returned by `Attach` is discarded because the
loopback driver does not need a separate downstream-frame collector — echo
delivery flows through `loopbackSink` → `downstreamHC.Enqueue` → downstream
ticker → `driver.pending`, not through the `AccessNode.DeliverFrame` fan-out
path. Discarding it is intentional and must be documented with a comment.

---

### §M2 — OnAck error handling and empty-tick window

**Defect (error swallowing):** The v1.1 Q4 pseudocode captured `err` from
`OnAck` but never checked it. SOUL.md §4 (no silent failure). If `OnAck`
returns `ErrAckOutOfWindow`, `delivered` is nil, all pending round trips time
out with no diagnostic, and the test reports "latency >100ms" when the real
problem is a window violation.

**Decision:** `OnAck`'s error MUST be checked and surfaced as a loud failure:

```go
delivered, err := driver.downstreamARQ.OnAck(chanSeq, zeroSACK)
if err != nil {
    // ErrAckOutOfWindow is the only expected error in this harness.
    // Any error here is a harness construction bug, not a latency measurement.
    driver.failLoud(fmt.Errorf("downstreamARQ.OnAck seq=%d: %w", chanSeq, err))
    return
}
```

`driver.failLoud` calls `t.Errorf` (not `t.Fatalf`) to allow the ticker
goroutine to return cleanly; the test is already doomed at this point.
[v1.5 F-B-4] The `driver.errCh chan error` (buffered 1) alternative previously
mentioned here is DROPPED. With AC-017 adding an upstream failLoud path symmetric
to AC-016's downstream, both ticker goroutines can call `failLoud` concurrently;
a buffered-1 channel blocks the second sender if the first fills the buffer and
nothing drains → `wg.Wait()` deadlock at teardown. The `t.Errorf`-based `failLoud`
is the sole specified error-surface mechanism — it is goroutine-safe, non-blocking,
and does not require a drain step.

**Defect (empty-tick window coupling):** `halfchannel.Tick()` increments `seq`
on EVERY tick, including empty ticks when there is no payload. `ErrAckOutOfWindow`
fires when `ackSeq - nextExpected > 64` (`arq.go:66-71`). The downstream ticker
starts at `NewLoopback` construction but `EnqueueSend` is only called on data
ticks. If the test harness idles more than 64 downstream ticks (64 × 50ms = 3.2s
at the standard interval) before the first `SendKeystroke`, the downstream
`HalfChannel.seq` has advanced 64 positions, but `downstreamARQ.nextExpected`
is still 0 — the first real data tick produces `chanSeq = 65` (or higher),
making `OnAck(65, zeroSACK)` return `ErrAckOutOfWindow` (65 - 0 = 65 > 64),
`delivered = nil`, and the first benchmark sample times out.

**Mitigation decision [v1.3 B-F3 tightened; v1.4 B-2 rationale corrected;
v1.10 F-LENSB-01 tightened]:**
Do not start the downstream ticker goroutine at `NewLoopback` time.
**PREFERRED: start the downstream ticker at `CreateSession` time.**
`CreateSession` is called once, from a single goroutine, before any
`SendKeystroke` calls — there is no concurrency at that point, so starting
the ticker there is race-free and requires no additional synchronization.
[v1.10 F-LENSB-01] The upstream ticker likewise starts at `CreateSession`
time, SYMMETRIC with the downstream ticker — NOT at construction. The prior
"upstream ticker MAY start at construction" / "start-time freedom PRESERVED"
latitude ([v1.6 F-B-LENSB-01]) is WITHDRAWN: the upstream ticker has no
`EnqueueSend` dependency, so nothing blocks a `CreateSession`-time start
either, and the same single-goroutine, race-free argument above applies to
it unchanged. This also makes AC-017's independence from ticker-start timing
TRUE BY CONSTRUCTION rather than a discipline the test must separately
uphold — see **AC-017 fault-injection method and required upstream test
seam** below, which no longer needs to guard against a construction-time
start because that option no longer exists.

**Per-option window-safety invariant [v1.4 B-2]:**

- **CreateSession-time start (PREFERRED):** `chanSeq` at the first data tick
  equals N+1, where N is the number of empty ticks elapsed between
  `CreateSession` and the first `SendKeystroke` (the data tick itself
  increments `seq` via the unconditional `h.seq++` in `halfchannel.Tick()`).
  [v1.5 F-B-2b] This is NOT guaranteed to be 1 — empty ticks accumulate
  freely from the moment the ticker starts. Window safety holds as long as
  fewer than 64 empty ticks precede the first data frame, i.e. N < 64
  (64 × 50ms = 3.2s at the standard interval). The ≤64 safety bound is
  unaffected by the N+1 correction. For the VP-042 benchmark, `CreateSession`
  is immediately followed by the send loop, so this trivially holds. For
  production-pattern tests with a delay between `CreateSession` and first
  `SendKeystroke`, a >3.2s idle gap will trigger `ErrAckOutOfWindow` — see
  the Edge Cases table row for that scenario. **The preferred option is chosen
  for its RACE-FREEDOM (B-F3), not for a stronger window margin.**

- **First-`SendKeystroke` start with `sync.Once` (alternative):** the first
  data tick fires immediately after the first enqueue, so `chanSeq` = 1 and
  `nextExpected` = 0, giving `1 - 0 = 1 ≤ 64` — the window margin is
  maximal. [v1.5 F-B-2] This `chanSeq=1` invariant is config-dependent: it
  holds only when the downstream interval exceeds the upstream round-trip
  latency (so no empty downstream ticks precede the echo), which is true for
  VP-042's config (downstream 50ms ≫ upstream round-trip ≈ 10ms). With a
  downstream interval shorter than the upstream round-trip, empty ticks would
  precede the echo and `chanSeq` > 1 at the first data tick. The `sync.Once`
  guard is MANDATORY (see below).

If the implementer chooses first-`SendKeystroke` start instead (e.g. for
a future multi-session shape), a `sync.Once` guard on ticker launch is
MANDATORY. Without it, concurrent first `SendKeystroke` calls (AC-008)
each observe not-started and launch duplicate tickers, producing
double-consumption of `ChanSeq` values and corrupting the ARQ window
(`deliverDownstream` called twice for seq=1; `EnqueueSend` / `OnAck`
sequence corrupted). The `sync.Once` idiom is already house convention
in this design (`closeOnce sync.Once` in the note / story L442). There
is no third option: `CreateSession`-time start (preferred) or
`sync.Once`-guarded first-`SendKeystroke` start.

A Reset-based approach is NOT available: `arq.ARQ` exposes no `Reset`
method, so the two options above are exhaustive.

The story-writer should add an AC for: "if `OnAck` returns an error, the harness
surfaces it as a loud failure (not a silent timeout)."

**AC-016 fault-injection method and required test seam [v1.4 B-3]:**

The previously considered approaches for forcing `OnAck` → `ErrAckOutOfWindow`
are BOTH unsound and must not be used:

- **(A) Start ticker at `NewLoopback` time:** contradicts B-F3 (race with AC-008);
  forbidden by the §M2 preferred design.
- **(B) Test-goroutine `OnAck` call:** races the downstream ticker goroutine's
  own `OnAck` call on the shared `downstreamARQ` — violates the `arq.ARQ`
  single-writer contract ("must be called from a single goroutine … Concurrent
  calls are NOT safe", `arq.go` package doc) and will be caught by
  `go test -race` (AC-015). Both paths are off the table.

**Sound method:** test exercises the downstream tick body synchronously with no
ticker goroutine running:

1. Build the `loopbackDriver` but do **not** start the downstream ticker goroutine.
2. Advance `driver.downstreamHC` past the 64-frame window by calling
   `driver.onDownstreamTick()` 65+ times synchronously (each empty tick — no
   payload enqueued — increments `downstreamHC.seq` without calling
   `EnqueueSend`, so `downstreamARQ.nextExpected` stays at 0).
3. Enqueue ONE data payload into `downstreamHC` (via `downstreamHC.Enqueue`).
4. Call `driver.onDownstreamTick()` one more time synchronously. This fires
   `downstreamARQ.EnqueueSend(chanSeq)` then `downstreamARQ.OnAck(chanSeq, zeroSACK)`
   with `chanSeq > 64` and `nextExpected = 0` — `OnAck` returns
   `ErrAckOutOfWindow`.
5. Assert that `driver.failLoud` was called (surfaced via `t.Errorf`, loud,
   not a silent `WaitForEcho` timeout).

This is single-goroutine throughout — no race, no `sync.Once` interaction,
no ticker goroutine competing for `downstreamARQ`.

**Required seam:** `onDownstreamTick()` must be a directly-callable
**package-private method** on `loopbackDriver` (or equivalent named function)
that contains the downstream tick body logic. [v1.5 F-B-1] `startLoopbackTicker`
is now a generic no-arg-callback ticker driver; the downstream tick body lives
entirely in `onDownstreamTick()`. The ticker goroutine invokes this body via
its `tickBody` argument (`startLoopbackTicker(env, interval, d.onDownstreamTick)`);
the AC-016 test invokes `onDownstreamTick()` directly, synchronously, without
starting the ticker goroutine — no race. This seam MUST be exposed so that
the AC-016 test can invoke it synchronously without launching the ticker
goroutine. The story-writer binds AC-016 to this seam; the implementer must
expose it. (The method is already named `onDownstreamTick()` in the note's
§H2 concrete shape — see above; this requirement locks that name as binding.)

**AC-017 fault-injection method and required upstream test seam [v1.6
F-B-LENSB-01; v1.10 F-LENSB-01 tightened to true-by-construction]:**

AC-017 tests that `SendKeystroke` called BEFORE `CreateSession` surfaces
`ErrConsoleNotFound` via `failLoud`. [v1.10 F-LENSB-01] Because both the
downstream AND upstream tickers now start ONLY at `CreateSession` time (see
the §M2 "Mitigation decision" above and the "Driver lifecycle pin" in §H3),
and AC-017 deliberately never calls `CreateSession`, NO ticker goroutine of
either kind can run at any point during the AC-017 test — this is now TRUE
BY CONSTRUCTION, not merely a discipline the test must separately uphold.
The test's direct, synchronous `onUpstreamTick()` call is therefore the ONLY
driver of the upstream tick body for the whole test; there is no concurrent
ticker goroutine that could also touch `upstreamHC`, `upstreamMP`, or (via
§M2's `recordingTB` stub) `stub.errorfCalls` while the test's own assertion
reads it. AC-017's fault-injection test invokes `onUpstreamTick()` DIRECTLY
and SYNCHRONOUSLY, exactly as AC-016 invokes `onDownstreamTick()` directly:

1. Do NOT start the upstream ticker goroutine. [v1.10] This step is now
   automatically satisfied rather than a separate discipline to uphold:
   `CreateSession` is the sole start-site for both tickers (§M2), and
   AC-017's procedure never calls it, so no upstream ticker goroutine can
   exist at any point in this test.
2. Call `SendKeystroke` before `CreateSession` — this enqueues a payload into
   `upstreamHC` (no console registered yet).
3. Call `driver.onUpstreamTick()` synchronously. This fires the upstream tick body:
   `upstreamHC.Tick()` dequeues the payload, `accessNode.SendKeystroke(...)` is
   called, returns `ErrConsoleNotFound` (no console registered), `failLoud` fires.
4. Assert that `driver.failLoud` was called (surfaced via `t.Errorf`). [v1.10]
   Because no ticker goroutine of either kind ever runs (step 1, now true by
   construction), this assertion's unlocked read of `stub.errorfCalls` — after
   the synchronous `onUpstreamTick()` call in step 3 has already returned —
   observes no concurrent writer. `go test -race` (AC-015) stays clean on
   this path by construction, closing the latent race the prior
   construction-start latitude permitted (the outcome count was always 1;
   the hazard was a race annotation on the read, not a wrong assertion
   value).

This is single-goroutine throughout — no race, no ticker-timing dependency,
ticker-start order is irrelevant to AC-017's correctness, and — per [v1.10]
— no ticker of either kind ever launches during this test at all.

**Required seam:** `onUpstreamTick()` must be a directly-callable **package-private
method** on `loopbackDriver` (symmetric to `onDownstreamTick()`). [v1.6] This is
already the method name in the §H2 concrete shape above; this requirement locks that
name as binding for the AC-017 test, exactly as `onDownstreamTick()` is bound for
AC-016. The story-writer binds AC-017 to this seam; the implementer must expose it.

**Recording `testing.TB` requirement for AC-016/AC-017 fault-injection tests
[v1.8 F-LENSB-B-03; v1.9 R6 BLOCKER fix]:** Both AC-016 (above) and AC-017
(above) assert that `driver.failLoud` FIRED as the PASSING outcome. But
`driver.failLoud` calls `t.Errorf` on the driver's OWN stored `testing.TB`
(the one supplied to `NewLoopback` at construction) — if that stored
`testing.TB` were the enclosing REAL `*testing.T` running the AC-016/AC-017
test itself with no override in front of it, that `t.Errorf` call would mark
the ENCLOSING test FAILED the instant `failLoud` fires. An AC-016/AC-017 test
written directly against the bare real `t`, with no interposed type, would
therefore be marked failed by Go's testing framework at the exact moment it
is supposed to observe a pass.

**Fix:** AC-016 and AC-017's fault-injection tests construct their driver
(via `NewLoopback`) with a RECORDING `testing.TB` stub/spy that EMBEDS the
real enclosing `*testing.T` and OVERRIDES only `Errorf` — feasible because
these are white-box, in-package tests (`package testenv`), and
`NewLoopback`/`SendKeystroke`/`WaitForEcho`/the driver already accept
`testing.TB` rather than a concrete `*testing.T`/`*testing.B`:

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

**Why embedding the real `t` is safe, not merely tolerated:** `Errorf` is
OVERRIDDEN on `*recordingTB` — Go method dispatch resolves `stub.Errorf(...)`
(and therefore `failLoud`'s call, which only ever sees `stub` through the
`testing.TB` interface) to `recordingTB.Errorf`, which appends into
`errorfCalls` and returns; an explicit method on the outer type always wins
over a promoted method from an embedded field, so this call never falls
through to the embedded real `t`'s `Errorf`. The enclosing test is therefore
NOT marked failed by `failLoud`'s `t.Errorf` call — this holds regardless of
whether the real `t` is embedded or the field is left nil. What DOES require
the real `t` to be embedded is the UNOVERRIDDEN methods: `NewLoopback` →
`newEnv` calls `b.Helper()` (`testenv.go:384`), `t.Helper()` (`:461`),
`t.Cleanup(func(){...})` (`:475`), and `t.Cleanup(e.Close)` (`:528`, the
ticker/env teardown AC-011/Q6 depend on) on every construction,
unconditionally — none of these are overridden on `recordingTB`, so they
promote straight through to the embedded field. `stub := &recordingTB{}`
(the prior v1.8 shape, no `TB:` set) leaves that embedded field nil; the
very first `b.Helper()` call inside `NewLoopback` panics on a nil-interface
method call, before AC-016's or AC-017's fault-injection procedure — or even
construction — completes. `&recordingTB{TB: t}` is therefore both correct
(the enclosing test cannot be failed by `failLoud`, by (1) above) and
required (construction cannot otherwise complete, by this paragraph). The
prior "in place of the real `*testing.T`" framing, and any reading of it as
"the real enclosing `t` must NOT be passed to `NewLoopback`," is retracted:
it inverted the actual constraint.

The `mu sync.Mutex` guard is required because `failLoud` may be invoked from a
ticker goroutine (the general case) even though AC-016/AC-017's own fault-injection
procedures call `onDownstreamTick()`/`onUpstreamTick()` synchronously from the test
goroutine — the stub must be safe regardless of which caller pattern exercises it.
The REAL enclosing `*testing.T` (`t`) plays two roles here: embedded inside
`stub` so `Helper`/`Cleanup`/`Fatalf` delegate to a live TB, and used directly,
after the fault-injection procedure completes, to report the assertion against
`stub.errorfCalls`. `failLoud`'s `t.Errorf` call is captured by the `Errorf`
override and never reaches the embedded real value in either role. This
recording-stub requirement is scoped to AC-016 and AC-017 only; every other
(happy-path) AC constructs its driver with the real `*testing.T`/`*testing.B`
directly (no `recordingTB` wrapper) exactly as before.

---

### §M4 — Helper signatures

The following conversion helpers must be implemented as package-private functions
in `internal/testenv`. Their obligations are enumerated here for the story-writer
to transcribe into tasks.

```go
// toMPFrame converts a halfchannel.ChannelFrame to a multipath.Frame.
// OBLIGATION: copies f.Payload into Frame.Payload. Does NOT carry f.ChanSeq
// into the returned struct (multipath.Frame has no ChanSeq field — B1 fix).
// OuterHeader bytes are synthesized from loopback channel metadata (no wire
// serialization; this is a Non-Goal).
func toMPFrame(f halfchannel.ChannelFrame) multipath.Frame

// encodeRTID encodes a round-trip id as an 8-byte big-endian suffix appended
// to key. Returns the combined payload: append([]byte(key), id_bytes...).
// Package-private. Pure function (no I/O).
func encodeRTID(key string, id uint64) []byte

// decodeRTID extracts the round-trip id from the last 8 bytes of payload.
// Returns (id, true) if len(payload) >= 8; (0, false) otherwise.
// Package-private. Pure function (no I/O).
// Because rtSeq.Add(1) starts ids at 1, id=0 (the !ok sentinel value) never
// collides with a real pending key — a decode failure is safely diagnosable,
// not a false hit against driver.pending.
func decodeRTID(payload []byte) (id uint64, ok bool)

// zeroSACK is the all-zero SACK bitmap passed to OnAck in the no-loss harness.
// Correct because Non-Goals excludes packet loss and reordering; a future
// loss-injection story must replace this with a real bitmap.
var zeroSACK [arq.SACKBitmapBytes]byte  // zero value, never written

// frameFor has been ELIMINATED in v1.2. It was only needed to bridge payload
// to frame.OuterHeader for the chan frame.OuterHeader completion carrier.
// RoundTrip.done is now chan []byte; payload passes directly, no bridge needed.
```

**`toMPFrame` ChanSeq-preservation obligation (B1 tie-in):** the function
signature takes a `halfchannel.ChannelFrame`, which carries `ChanSeq`. The
function does NOT embed ChanSeq into the returned `multipath.Frame`. This is
the correct contract — the caller captures `chanSeq := f.ChanSeq` BEFORE calling
`toMPFrame`, and uses that captured value for `OnAck`. Any future modification
to `toMPFrame` that attempts to thread `ChanSeq` through `multipath.Frame`
would require adding a field to that struct, which is prohibited (Non-Goal).

**`WaitForEcho` payload/header consistency obligation (H1 tie-in):** the
completion channel carries `[]byte` (the echo payload). `WaitForEcho` returns
this slice directly to the caller. The caller asserts `id, ok2 := decodeRTID(payload); ok2 && id == rt.id`
(AC-014(b); [v1.4 B-1] — 2-value call required). The chain: `loopbackSink.SendInput` echoes the full payload (including
RT-ID suffix) → `downstreamHC.Enqueue(payload)` → downstream ticker →
`delivered` slice from `OnAck` → `ch <- payload` → `WaitForEcho` returns it.
No transform is applied to payload bytes between sink and return. `decodeRTID`
reads the last 8 bytes; `encodeRTID` appends them. These must be inverse
functions: `decodeRTID(encodeRTID(key, id)) == id` for all `key`, `id`.

---

**Files consulted for this v1.2 addendum:** `internal/multipath/multipath.go`
(Frame struct, confirmed no ChanSeq), `internal/halfchannel/halfchannel.go`
(ChannelFrame, concurrency doc, Tick, Enqueue), `internal/frame/frame.go`
(OuterHeader fields), `internal/session/upstream.go` (SendKeystroke),
`internal/testenv/testenv.go` (AttachConsole idiom, NewLoopback),
`internal/arq/arq.go` (OnAck, EnqueueSend, ErrAckOutOfWindow window guard).

---

## v1.3 R1 Re-Review Repair (2026-07-22)

**Scope:** Applies orchestrator-verified (PAT-04) findings from the R1 re-review
pass: B-F1 (HIGH — missing `Publish` precondition in §H3), B-F2 (MED —
`encodeRTID` arity mismatch Q3 vs §M4), B-F3 (MED — lazy-start races with AC-008),
C-F1 (MED — stale `chan frame.OuterHeader` in Q3 pseudocode), C-F2 (MED — live
phantom `mpFrame.ChanSeq()` in v1.1 Addendum "Required revision"). All five were
confirmed real against source before this repair. Core design (B1/H1/H2/H3-core/M2-core)
confirmed SOUND — these are propagation-edge and precondition defects only.

**Source consulted for B-F1:** `internal/session/upstream.go:203-208`
(`Attach` calls `a.pub.Get(sessionName)` before the auth gate — confirmed);
`internal/session/session.go:137` (`func (p *Publisher) Publish(sessionName string) error`
— canonical signature).

---

### §B-F1 — Missing `Publish` precondition in §H3 provisioning (HIGH)

**Defect:** The §H3 provisioning sequence listed `newConsoleKey()` →
`RegisterKey(sessionName, key, RoleFull)` → `Attach(key, sessionName)` as the
complete sequence. But `AccessNode.Attach` (`upstream.go:203-208`) calls
`a.pub.Get(sessionName)` as its FIRST gate — before the auth check. The loopback
driver builds its own dedicated `Publisher`/`SessionAuth`/`AccessNode` triple (Q2),
so that Publisher is empty at construction. Without a prior `Publish` call,
`Attach` returns `ErrSessionNotFound` and `t.Fatalf` fires at driver construction —
the H3 happy path is unreachable as written.

**Fix:** `sh.pub.Publish(sessionName)` is inserted as step 1 of the provisioning
sequence, before `RegisterKey` and `Attach`. See the corrected code block in §H3
above.

**`Publisher.Publish` signature (from source):** `func (p *Publisher) Publish(sessionName string) error`

This call publishes into the DRIVER'S OWN Publisher (`sh.pub`), not into
`env.defaultShard`'s publisher — consistent with Q2's explicit statement that the
loopback driver builds its own independent triple.

**Corrected complete provisioning sequence:**
1. `sh.pub.Publish(sessionName)` — makes `pub.Get(sessionName)` succeed in `Attach`.
2. `sh.auth.RegisterKey(sessionName, loopbackConsoleKey, session.RoleFull)` — makes the authorizer allow the key.
3. `sh.access.Attach(loopbackConsoleKey, sessionName)` — attaches the console.

---

### §B-F2 — `encodeRTID` arity mismatch Q3 vs §M4 (MED)

**Defect:** Q3 pseudocode called `encodeRTID(id)` (1-arg, implied suffix-only return);
§M4 defined `func encodeRTID(key string, id uint64) []byte` (2-arg, whole-payload
return). Contradictory: different arity, and Q3 also double-appended
(`append([]byte(key), encodeRTID(id)...)`).

**Fix:** The **2-arg whole-payload form** from §M4 is canonical:
`func encodeRTID(key string, id uint64) []byte`. The Q3 call site is corrected
in place (see above) to `payload := encodeRTID(key, id)` — single call, no
manual append. §M4 definition is unchanged. Story-writer transcribes the 2-arg
form for both definition and every call site.

---

### §B-F3 — Lazy-start option races with AC-008 (MED)

**Defect:** §M2 offered first-`SendKeystroke` OR `CreateSession`-time lazy start
with no synchronization specified. Under AC-008 (two concurrent `SendKeystroke`
calls), first-`SendKeystroke` start lets multiple goroutines each observe
not-started and launch duplicate tickers, double-consuming `ChanSeq` values
and corrupting the ARQ window. `go test -race` (AC-015) would also catch the
unguarded `started` flag race.

**Fix (tightened in §M2 above):** PREFERRED: start the downstream ticker at
`CreateSession` time (single-threaded, no concurrency yet, zero synchronization
overhead). If first-`SendKeystroke` start is used, a `sync.Once` guard on ticker
launch is MANDATORY — the `sync.Once` idiom is already house convention in this
design (`closeOnce sync.Once`). No third option.

---

### §C-F1 — Stale `chan frame.OuterHeader` in Q3 pseudocode (MED)

**Defect:** Q3 pseudocode narrated `driver.pending (map[uint64]chan frame.OuterHeader, ...)` — a live design statement, stale relative to the §H1 fix (`chan []byte`).

**Fix:** Corrected in Q3 pseudocode (above) to `map[uint64]chan []byte`. The
v1.1 Q4 pseudocode (`ch <- frameFor(payload)` with superseded annotation) and
the §M4 `frameFor` elimination note are already correctly annotated as
old/superseded — those were NOT changed.

---

### §C-F2 — Phantom `mpFrame.ChanSeq()` in v1.1 Q4 Addendum prose (MED)

**Defect:** The v1.1 Q4 Addendum "Required revision" paragraph contained a live
binding instruction ending with `... call \`OnAck(mpFrame.ChanSeq(), zeroSACK)\`
on the SAME instance.` This used the B1 phantom (`multipath.Frame` has no
`ChanSeq` method) as if it were settled API — a live phantom instruction in a
"Required revision" section that the story-writer would transcribe.

**Fix:** Corrected in place to `OnAck(chanSeq, zeroSACK)` with an inline
`[v1.3 C-F2]` annotation explaining `chanSeq` is captured from the originating
`ChannelFrame` before `toMPFrame`. The correction supersedes the B1 fix already
applied to the Q4 code block — no live phantom instruction remains anywhere in
the note.

---

**Files consulted for this v1.3 addendum:** `internal/session/upstream.go:203-208`
(Attach pub.Get gate), `internal/session/session.go:137` (Publisher.Publish signature).
All other ground-truth claims verified by orchestrator PAT-04 against source before
this repair; no additional reads required.

---

## v1.4 R2 Re-Review Repair (2026-07-22)

**Scope:** Applies orchestrator-verified (PAT-04) findings from the R2 re-review
pass: B-1 (MED — `decodeRTID` arity mismatch: 1-value call sites throughout note),
B-2 (MED — §M2 window rationale wrong for CreateSession-time preferred option),
B-3 (MED — AC-016 fault-injection guidance unsound; both prior options race or
contradict design invariants), N-1 (MED — §M2 self-contradiction "no third option"
vs "acceptable simpler alternative"). All four confirmed real via PAT-04 ground
truth before this repair. Core design (B-F1/B-F3/A-M1) reconfirmed SOUND — these
are propagation-edge and rationale defects only.

---

### §B-1 — `decodeRTID` arity mismatch (MED)

**Defect:** §M4 defines `decodeRTID(payload []byte) (id uint64, ok bool)` (2-value),
but call sites in the note used 1-value form: `id := decodeRTID(payload)` in the
Q4/downstream pseudocode blocks (two locations), and `decodeRTID(payload) == rt.id` /
`!= rt.id` in the AC-014 assertion prose in §Q5 and §H1.

**Fix:** All call sites corrected to 2-value form throughout the note:
- Pseudocode blocks: `id, ok := decodeRTID(payload); if !ok { continue }`
- AC-014 assertion prose: `id, ok2 := decodeRTID(payload); if !ok2 || id != rt.id { t.Errorf(...) }`
- §M4 `decodeRTID` definition annotated with the `rtSeq.Add(1)` safety invariant:
  because ids start at 1, `id=0` (the `!ok` sentinel) never collides with a real
  pending key — decode failure is safely diagnosable, not a false hit.

---

### §B-2 — Window-margin rationale wrong for CreateSession-time option (MED)

**Defect:** §M2 L1156-1158 claimed "by the time the downstream ticker produces
its first tick, `chanSeq` = 1 and `nextExpected` = 0, giving `1 - 0 = 1 ≤ 64`"
as justification for CreateSession-time start. This is incorrect: `Tick()`
increments `seq` on EVERY tick (including empty ticks, confirmed `halfchannel.go:118`
unconditional `h.seq++`). Under CreateSession-time start, empty ticks accumulate
freely from the moment the ticker starts — `chanSeq` at the first data tick equals
the number of empty ticks elapsed, NOT 1. The "chanSeq = 1" invariant holds only
for the first-`SendKeystroke` option (ticker starts immediately before first data
tick). The same error appeared in the RULING-003 window analysis paragraph.

**Fix:** Both locations corrected. Per-option window invariants now stated explicitly:
- **CreateSession-time (PREFERRED):** window safe while fewer than 64 empty ticks
  precede first data frame (≈ 3.2s at 50ms interval); trivially satisfied in VP-042
  benchmark but caveat documented. Preferred for RACE-FREEDOM, not window margin.
- **First-SendKeystroke (alternative):** `chanSeq = 1`, `nextExpected = 0` → margin
  maximal. `sync.Once` MANDATORY.

---

### §B-3 — AC-016 fault-injection unsound; seam specified (MED)

**Defect:** The note's §M2 story-writer guidance suggested forcing
`OnAck` → `ErrAckOutOfWindow` via either (A) start ticker at `NewLoopback` (contradicts
B-F3, forbidden) or (B) a test-goroutine `OnAck` call (races the ticker goroutine's
`OnAck` on `downstreamARQ` — violates `arq.ARQ` single-writer contract, caught by
`go test -race` / AC-015). Both options are unsound.

**Fix:** Sound fault-injection method specified in §M2:
1. Build driver without starting the downstream ticker goroutine.
2. Advance `downstreamHC` past 64 via 65+ synchronous empty `onDownstreamTick()` calls.
3. Enqueue one payload; call `onDownstreamTick()` once more — `OnAck(chanSeq, zeroSACK)`
   fires with `chanSeq > 64`, `nextExpected = 0` → returns `ErrAckOutOfWindow`.
4. Assert `driver.failLoud` was called (loud, not silent timeout).

**Required seam:** `onDownstreamTick()` is named as a BINDING package-private method
on `loopbackDriver` that exposes the downstream tick body for synchronous invocation.
This seam must be implemented; the story-writer binds AC-016 to it.

---

### §N-1 — §M2 "no third option" self-contradiction (MED)

**Defect:** §M2 stated "There is no third option" then immediately offered "An
acceptable simpler alternative: reset `downstreamARQ.nextExpected` to
`downstreamHC.Seq()` …" — a live contradiction from a v1.2-era leftover.

**Fix:** The "acceptable simpler alternative" paragraph removed; replaced with:
"A Reset-based approach is NOT available: `arq.ARQ` exposes no `Reset` method, so
the two options above are exhaustive." The "no third option" invariant now holds
without self-contradiction.

---

**Ground truth source:** all PAT-04 claims transcribed from orchestrator-verified
source reads before this repair. No additional file reads required for these fixes.

---

## v1.5 R3 Re-Review Repair (2026-07-22)

Round 3 re-review found 1 HIGH + 1 MED (story-writer scope) + 2 LOW + 2 nitpicks.
The architect-owned findings are fixed inline above; this section records the
design decisions made and the rationale.

### §F-B-1 (HIGH) — `startLoopbackTicker` made tick-free; directions wired via seam methods

**Defect (PAT-04 verified):** The v1.4 Q6 `startLoopbackTicker` signature was
`func startLoopbackTicker(env *Env, hc *halfchannel.HalfChannel, interval time.Duration, onTick func(halfchannel.ChannelFrame))`
with body `onTick(hc.Tick())` — the HELPER called `hc.Tick()` inside the ticker
goroutine, with no mutex. But §H2 (`onDownstreamTick`) and L256 require every
`Tick()` call under the per-direction mutex. The downstream half-channel is also
`Enqueue`'d from `loopbackSink.SendInput` under `downstreamHCMu` (from the
upstream ticker goroutine). Two goroutines touching `downstreamHC` with mutex
on only one → DATA RACE → fails AC-015 (`just test-race`).

Additionally, v1.4 §M2/§B-3 (~L1240) claimed `onDownstreamTick()` is "the same
body that the ticker goroutine calls from `startLoopbackTicker`" — which does not
type-check (`func()` ≠ `func(ChannelFrame)`), and if adapted by wrapping would
double-tick (seq 2×, ARQ window corruption) or run `hc.Tick()` outside the mutex.

**Design decision:**
1. `startLoopbackTicker` is rewritten to be TICK-FREE. New signature:
   `func startLoopbackTicker(env *Env, interval time.Duration, tickBody func())`.
   The `hc` parameter is removed. The body is `case <-ticker.C: tickBody()`.
   The helper owns only goroutine lifecycle (wg, closeCh, ticker), not any
   half-channel operation.
2. Both directions route through the internal-tick seam methods:
   - Upstream ticker: `startLoopbackTicker(env, upstreamInterval, d.onUpstreamTick)`
   - Downstream ticker: `startLoopbackTicker(env, downstreamInterval, d.onDownstreamTick)`
   Each seam method calls `<dir>HCMu.Lock(); f := d.<dir>HC.Tick(); <dir>HCMu.Unlock()`
   then the direction's flow logic. Every `Tick()` is therefore mutex-guarded
   (satisfies §H2 + L256). `onDownstreamTick()` is the single directly-callable
   seam AC-016 requires — the AC-016 test invokes it synchronously without starting
   any goroutine → no race.
3. The false-composition sentence retracted and replaced (see §M2/§B-3 above).

The "same shape as `cmd/switchboard/access.go:460`" comparison is updated to note
the `hc` parameter is absent; the lifecycle contract (wg/closeCh/ticker pattern)
remains the same.

### §F-B-4 (LOW) — `errCh` alternative dropped

**Defect:** The `driver.errCh chan error` (buffered 1) offered as an acceptable
alternative to `failLoud` can deadlock with AC-017's symmetric upstream error
path: both ticker goroutines call `failLoud` concurrently; the first fills the
buffer-1 channel; the second blocks forever; `wg.Wait()` hangs.

**Decision:** The errCh alternative is dropped entirely. The `t.Errorf`-based
`failLoud` is the sole specified error-surface mechanism. It is goroutine-safe,
non-blocking by specification, and requires no drain step. If a future need arises
for cross-goroutine error aggregation, buffered-≥2 + guaranteed drain would be
required, but that need does not arise from the current AC set.

### §F-B-2 (LOW→nitpick) — "chanSeq=1" qualified as config-dependent

**Defect:** The first-`SendKeystroke`-start option stated `chanSeq=1` as if it
were a general invariant. It holds only when downstream interval > upstream
round-trip latency (VP-042: 50ms ≫ ~10ms). A shorter downstream interval would
permit empty ticks to precede the echo, yielding `chanSeq > 1`.

**Fix:** Prose qualified: "holds for VP-042's interval config (downstream 50ms ≫
upstream round-trip); a downstream interval shorter than the upstream round-trip
would yield chanSeq > 1."

### §F-B-2b (NITPICK) — CreateSession-time chanSeq off-by-one corrected

**Defect:** "chanSeq at the first data tick equals the number of empty ticks
elapsed" — incorrect because the data tick itself increments `seq`
(`halfchannel.Tick()`: unconditional `h.seq++` before the payload branch).
First data `chanSeq` = N+1, not N.

**Fix:** Prose corrected to N+1. The ≤64 safety bound (N < 64) is unaffected;
the correction is cosmetic but necessary for accuracy when callers reason about
the first `chanSeq` value.

---

**Ground truth source (v1.5):** PAT-04 verified against source reads documented
in rereview-R3-2026-07-22.md. `startLoopbackTicker` body (`onTick(hc.Tick())`),
`onDownstreamTick`/`onUpstreamTick` signatures (no-arg, mutex-internal), and
L256 binding rule all confirmed from source before this repair.

---

## v1.6 R4 Re-Review Repair (2026-07-22)

Round 4 re-review found 2 LOW + 2 NITPICK (no HIGH, no MED). Architect-owned
findings (note-side) are fixed inline above; this section records the design
decisions made and rationale.

### §F-B-LENSB-01 (LOW) — upstream ticker / AC-017 timing tension resolved

**Defect (PAT-04 verified):** The §M2 prose at ~L1182 said "The upstream ticker
MAY start at construction (it has no `EnqueueSend` dependency)." AC-017's
fault-injection test (story L868-875, transcribed from this design) triggers
`ErrConsoleNotFound` by calling `SendKeystroke` BEFORE `CreateSession`. This
test only surfaces via `failLoud` if the upstream ticker is running to process
the enqueued keystroke through the upstream tick body. An implementer who starts
the upstream ticker at `CreateSession` time (symmetric with the downstream M2
lazy-start preferred option) would make AC-017 enqueue-then-never-process →
hang to timeout rather than firing `failLoud`. The design's own "MAY" permitted
a choice that breaks one of its own ACs.

**Design decision (preferred approach chosen):** Resolve the tension by making
AC-017's test **ticker-timing-independent**, mirroring the AC-016/`onDownstreamTick()`
pattern exactly. AC-017 invokes `onUpstreamTick()` **directly and synchronously**
— no upstream ticker goroutine is started — so the upstream tick body runs
`deliverUpstream` → `accessNode.SendKeystroke` → `ErrConsoleNotFound` →
`failLoud`, deterministically, regardless of whether the upstream ticker would
normally start at construction or at `CreateSession`.

**Implementation:** `onUpstreamTick()` is declared a binding directly-callable
package-private seam (symmetric to `onDownstreamTick()`). The "MAY start at
construction" language at §M2 is preserved (the timing freedom is still valid for
the normal/happy path); it is no longer load-bearing for AC-017 correctness. The
upstream seam guidance block immediately following the §B-3/AC-016 section (added
in this v1.6) specifies the 4-step AC-017 test procedure. The story-writer
transcribes the seam binding and test procedure into AC-017.

**Approach NOT taken:** changing "MAY" → "MUST start at construction" — rejected
because (a) it would overconstrain the upstream ticker lifecycle without clear
benefit (the upstream direction has no empty-tick window problem), and (b) the
direct-seam approach removes the timing dependency entirely, which is strictly
cleaner and matches the already-established AC-016 idiom in this codebase.

### §L-C-1 (LOW) — stale `onTick callback` comment lead-ins in §H2 reworded

**Defect:** §H2 concrete-shape block comment lead-ins used the retired "onTick"
descriptor (a relic of v1.4's `onTick func(ChannelFrame)` shape that F-B-1 renamed
to `tickBody func()`). The methods below them (`onUpstreamTick()`/`onDownstreamTick()`)
were correctly named; only the comment lead-in word "onTick" was stale.

**Fix:** L1044 reworded to:
`// Upstream ticker tickBody (wired as startLoopbackTicker(env, upstreamInterval, d.onUpstreamTick)):`
L1060 reworded to:
`// Downstream ticker tickBody (wired as startLoopbackTicker(env, downstreamInterval, d.onDownstreamTick)):`

These lead-ins now reference the `tickBody` parameter and the exact
`startLoopbackTicker` call-site wiring, consistent with the v1.5 tick-free
rewrite and §F-B-1's `tickBody func()` signature.

### §F-A-1 (NITPICK) — RoundTrip doc-comment purpose sentence reworded

**Defect:** RoundTrip doc-comment ~L337 purpose sentence contained the 1-value
code token `decodeRTID(payload)==rt.id`. The binding 2-value assertion was
correct immediately adjacent, but the 1-value form in the purpose sentence was
a copyable wrong-arity token.

**Fix:** Reworded to semantic English:
`"WaitForEcho must return the delivered payload so callers can assert the delivered payload decodes to rt.id (AC-014 load-bearing part)."`
No code expression in the purpose sentence; the 2-value binding assertion adjacent
(L361) is unchanged.

---

**Ground truth source (v1.6):** Edits derived from rereview-R4-2026-07-22.md
findings (disk-verified per PAT-04 protocol). No source re-reads required for
these note-only doc/comment repairs; F-B-LENSB-01 design decision based on
analysis of AC-016 seam pattern already in this note.

---

## v1.8 R5 Re-Review Repair (2026-07-24)

Round 5 re-review found 1 MED (F-LENSB-B-01) + 2 LOW (F-LENSB-B-02,
F-LENSB-B-03) + 1 optional nitpick (A-L967, Lens A). All three headline
findings trace to one design area — the `loopbackDriver` lifecycle /
provisioning-timing / goroutine-ownership boundary around AC-017 — and are
resolved coherently in this single pass per defect-lifecycle §1.3 ("fix the
whole class, not just the flagged instance"), rather than as three isolated
patches.

### §F-LENSB-B-01 (MED) — §H3 provisioning-timing contradicted AC-017's un-provisioned premise

**Defect (PAT-04 verified):** §H3's "**Decision:**" permitted console
provisioning "(in `CreateSession` or in the `loopbackDriver` constructor)"
while the adjacent code comment said "Construction-time provisioning — called
once per CreateSession" — internally contradictory, and load-bearing: if an
implementer took the permitted constructor-provisioning path, the driver's
console would already be attached by the time AC-017 (§M2) calls
`SendKeystroke` pre-`CreateSession`, so `SendKeystroke` would SUCCEED instead
of surfacing `ErrConsoleNotFound` via `failLoud` — AC-017's step-4 assertion
would fail. The note permitted a construction choice that broke one of its
own ACs.

**Fix:** §H3's "Decision" now states console provisioning happens ONLY in
`CreateSession`; the constructor-provisioning latitude is withdrawn. A new
"Driver lifecycle pin" subsection (§H3) makes the full state machine
explicit: the `loopbackDriver` constructor (via `NewLoopback`) builds the
AccessNode triple (Q2), both `*multipath.Multipath` instances (Q7), and both
`*halfchannel.HalfChannel` instances (§H2) — all fully usable — with the
console left UN-PROVISIONED; only `CreateSession` performs Publish/
RegisterKey/Attach plus starts the downstream ticker. This resolves the R5
sub-issue directly: `onUpstreamTick()` cannot nil-deref `upstreamMP` when run
pre-`CreateSession` (as AC-017 requires), because `upstreamMP` — like
`downstreamMP` and both half-channels — is built at construction, not
lazily. The code comment is corrected to "Session provisioning — called ONLY
from CreateSession, never from the loopbackDriver constructor." A
cross-reference was propagated to the "Steps 1–3 MUST appear in this order"
sentence, tying console provisioning and the separately-timed downstream
ticker start back to the lifecycle pin. Grepped for every other "constructor"
reference in the note — none found (this was the sole occurrence).

### §F-LENSB-B-02 (LOW) — `SendKeystroke`'s no-validation dependency made explicit

**Defect:** AC-017 depends on `SendKeystroke` performing no session-existence
check (it must succeed at mint/encode/enqueue even when called before
`CreateSession`, with the real failure surfacing later inside
`onUpstreamTick()`), but the note never stated this as a constraint — an
implementer adding a defensive session-existence guard would abort AC-017 at
step 1, before `failLoud` could ever fire.

**Fix:** Q3 gains a "`SendKeystroke` performs no session-existence validation"
paragraph immediately after the upstream-flow pseudocode, stating the
constraint and its AC-017 rationale explicitly, with an explicit prohibition
on adding a defensive guard. The `SendKeystroke` doc comment at its
definition (§Q5) gains a matching one-paragraph note cross-referencing Q3, so
the constraint is visible both at the flow narrative and at the function
signature itself.

### §F-LENSB-B-03 (LOW, residual from AC-016) — recording `testing.TB` stub specified for fault-injection tests

**Defect:** AC-016 and AC-017's fault-injection procedures (§M2) both assert
that `driver.failLoud` fired as their PASSING outcome, but `failLoud` calls
`t.Errorf` on the driver's own stored `testing.TB` — if that's the real
`*testing.T` running the AC-016/AC-017 test, `t.Errorf` marks the ENCLOSING
test FAILED at the moment `failLoud` fires, not just recorded as an
assertable fact.

**Fix:** A single new subsection in §M2, positioned after both the AC-016 and
AC-017 procedures so it covers both without duplication, specifies that these
two tests construct their driver (via `NewLoopback`) with a recording
`testing.TB` stub/spy — a white-box, in-package type embedding `testing.TB`
and overriding `Errorf` to append into a mutex-guarded slice — rather than
the real `*testing.T`. The real enclosing `*testing.T` then asserts against
the stub's recorded calls instead of receiving `failLoud`'s `t.Errorf`
directly. Scoped explicitly to AC-016/AC-017 only; all happy-path ACs are
unaffected and continue to construct their driver with the real
`*testing.T`/`*testing.B`.

### §A-L967 (NITPICK, Lens A) — disposition: cleaned

**Assessment:** §H1's "**Defect:**" paragraph is live design-rationale prose
(not a dated changelog row or repair-addendum section), so it was safe to
reword. The 1-value code token `decodeRTID(payload) == rt.id` was the last
live occurrence of that class in the note (confirmed by grep — the one
remaining match, at the v1.4 §B-1 historical defect description, is a frozen
repair-addendum row and was left untouched per §2.9).

**Fix:** Reworded to "AC-014(b) requires the caller to assert the delivered
payload decodes to `rt.id`" — semantic English, no code expression, matching
the pattern already applied at v1.6 §F-A-1 and v1.7. Zero live 1-value
`decodeRTID(payload)==` tokens remain anywhere in the note.

---

**Ground truth source (v1.8):** All four fixes are note-internal
consistency/completeness repairs to sections already grounded in prior PAT-04
source verification (§H3 provisioning against `internal/session/upstream.go`
and `session.go` per the v1.3 addendum; Q7's multipath construction and §H2's
half-channel construction per their own sections). No additional source reads
were required for this repair.

---

## v1.9 R6 Re-Review Repair (2026-08-28)

Round 6 re-review (3 lenses + oracle) of the v1.8-note/v1.7-story R5-repairs
found the oracle GREEN and F-LENSB-B-01/02 + concurrency SOUND, but two
independent lenses (A and B) converged on the v1.8 F-LENSB-B-03
recording-`testing.TB`-stub fix as a BLOCKER, corroborated by
orchestrator disk-verification against `internal/testenv/testenv.go`. One
LOW (Lens B O-1) was also raised. Both are resolved in this pass.

### BLOCKER — `recordingTB` stub embeds a nil `testing.TB`, nil-panics at construction

**Defect (disk-verified against `internal/testenv/testenv.go`):**
`NewLoopback(ctx, b testing.TB, cfg)` calls `b.Helper()` (`testenv.go:384`)
then `newEnv(ctx, b, 1)`; `newEnv` calls `t.Helper()` (`:460`),
`t.Cleanup(func(){...})` (`:475`), and `t.Cleanup(e.Close)` (`:528` — the
ticker/env teardown AC-011/Q6 depend on). The v1.8 §M2 fix specified
`type recordingTB struct { testing.TB; mu sync.Mutex; errorfCalls []string }`
overriding ONLY `Errorf`, constructed as `stub := &recordingTB{}` — the
embedded `testing.TB` field was left at its zero value, i.e. nil. `Helper`
and `Cleanup` are NOT overridden, so calling them on `stub` promotes to the
nil embedded interface and panics — at the very first `b.Helper()` inside
`NewLoopback`, before AC-016's or AC-017's fault-injection procedure, or even
construction, completes. Both ACs would crash unconditionally, not just in a
subtle edge case. The v1.8 struct comment ("embed to satisfy the interface;
unused methods panic if called") was itself the error: `Helper`/`Cleanup`
are not unused — they are called on every `NewLoopback`.

**Compounding defect — inverted rationale:** the v1.8 fix's framing
("construct their driver ... with a RECORDING `testing.TB` stub/spy in
place of the real `*testing.T`", `stub := &recordingTB{}` with no real `t`
passed in at all) reads as, and was applied as, "the real enclosing `t`
must not be passed to `NewLoopback`, or `failLoud`'s `t.Errorf` marks this
passing test FAILED." That premise is incorrect Go semantics: `recordingTB`
OVERRIDES `Errorf`, so `stub.Errorf(...)` — the only path `failLoud` can
reach, since it only ever sees `stub` through the `testing.TB` interface —
dispatches to the override and is captured; it never falls through to an
embedded value's `Errorf` regardless of whether that embedded value is real
or nil. Avoiding the real `t` was never what protected the enclosing test;
the `Errorf` override was. Withholding the real `t` bought nothing and cost
everything (`Helper`/`Cleanup` correctness) that the override doesn't cover.

**Fix (adjudicated: embed the real enclosing `t`):** §M2's `recordingTB`
construction changes from `stub := &recordingTB{}` to
`stub := &recordingTB{TB: t}`. The struct's embedded-field comment is
corrected to state that `Helper`/`Cleanup`/`Fatalf` promote to and are
serviced by the embedded real `t` (citing `testenv.go:384/460/475/528`),
and that only `Errorf` is overridden, to capture rather than fail. The
"in place of the real `*testing.T`" framing and its "must NOT pass real t"
implication are retracted and replaced with the correct rule: the override,
not the absence of a real `t`, is what keeps the enclosing test green; the
real `t` is additionally REQUIRED so `Helper`/`Cleanup`(including the
`Cleanup(e.Close)` ticker teardown)/`Fatalf` delegate to a live TB instead of
nil-panicking. Full before/after text is in §M2 above (the "Recording
`testing.TB` requirement" subsection, tagged `[v1.8 F-LENSB-B-03; v1.9 R6
BLOCKER fix]`). Grepped the note for every other `recordingTB`/stub
construction and every "unused methods panic"/"must NOT be passed" framing
— the §M2 occurrence fixed here was the only live instance of either; no
sibling propagation was required beyond it. This live-prose section sits
inside the "v1.2 Design Repair Addendum" heading, itself amended by every
subsequent re-review round (v1.3 through v1.8) — it is current binding spec,
not frozen history, and was in-scope to edit; the dated v1.8 §F-LENSB-B-03
subsection below (in the "v1.8 R5 Re-Review Repair" section) records the
now-superseded fix and is left untouched per §2.9.

### LOW (Lens B O-1) — `driver.sessionName` storage/timing unpinned

**Defect:** the upstream delivery callback (Q3) calls
`driver.accessNode.SendKeystroke(loopbackConsoleKey, sessionName, payload)`,
but `sessionName` was only ever shown as a local in §H3's `CreateSession`
provisioning code block — the note never stated that `sessionName` is
stored as a `loopbackDriver` field, nor when that field is populated.
Harmless for AC-017 (a zero-value `sessionName` still yields
`ErrConsoleNotFound`, since no session named `""` is ever published), but
the happy-path ACs (AC-004/005/006 and others) need the field and its
timing pinned to be implementable without guessing.

**Fix:** §H3 gains a "`sessionName` storage/timing pin" paragraph,
immediately after the sentence establishing `loopbackConsoleKey` is stored
on `loopbackDriver`. It states `sessionName` is likewise a `loopbackDriver`
field, set once in `CreateSession` alongside `loopbackConsoleKey` before
Steps 1–3 run, and that — consistent with the "Driver lifecycle pin"
(construction leaves the driver un-provisioned; `CreateSession` provisions
it) — `driver.sessionName` holds its zero value (`""`) before `CreateSession`
has run. This is what makes AC-017's pre-`CreateSession` call observe
`ErrConsoleNotFound` (no session named `""` exists), independent of
`loopbackConsoleKey`'s own zero-value state, and pins the happy-path timing
(`driver.sessionName` is populated and stable by the time any upstream tick
can reach the delivery callback) for AC-004/005/006.

### Governance

Both fixes are edits to live spec prose (§M2, §H3) inside sections already
amended across v1.2 through v1.8; no frozen dated changelog row or prior
repair-addendum section was touched. Story and index files are untouched —
per dispatch, this placement-note edit is transcribed into the story by
story-writer separately.

**Ground truth source (v1.9):** the BLOCKER fix's `Helper`/`Cleanup`
call-site line numbers (`testenv.go:384/460/475/528`) were disk-verified
against `internal/testenv/testenv.go` by the orchestrator before dispatch
and re-confirmed against the same file in this session. No additional
source reads were required for the LOW fix (an internal note-consistency
pin, not a ground-truth claim against source).

## v1.10 R7 Re-Review Repair (2026-08-28)

Round 7 re-review found one MED (F-LENSB-01) and one LOW (F-LENSB-02), both
tracing to the `loopbackDriver` upstream-ticker start-timing / AC-017
fault-attribution area — the same design region v1.6 and v1.9 previously
touched. Both are resolved in this pass, plus a line-reference erratum
folded in from the same dispatch.

### F-LENSB-01 (MED) — upstream-ticker start-site under-pinned; contradicted AC-017's single-goroutine claim; latent `-race` hazard

**Defect:** §M2's ticker-start mitigation decision said the downstream
ticker starts at `CreateSession` (PREFERRED) while the upstream ticker "MAY
start at construction ... [v1.6 F-B-LENSB-01] This start-time freedom is
PRESERVED." Meanwhile AC-017's method text claimed "single-goroutine
throughout" while step 1 only asked the test to refrain from starting the
upstream ticker itself — it did not, and structurally could not, prevent a
construction-time-started upstream ticker from running in the background
during the AC-017 test, because "MAY start at construction" made that
ticker's existence an implementer's free choice, not something AC-017's own
procedure controlled. If an implementer took the construction-start option,
`NewLoopback` would start a live background upstream ticker goroutine.
That ticker's `onUpstreamTick()` calls `failLoud` under `recordingTB.mu`
(§M2's `Errorf` override) on any tick that hits an error path, while
AC-017's own assertion (`if len(stub.errorfCalls) != 1 { ... }`) reads
`stub.errorfCalls` UNLOCKED, directly, no `mu.Lock()` around the read. A
background ticker goroutine writing under lock concurrently with the test
goroutine's unlocked read is a data race by Go's memory model regardless of
whether the two ever observe inconsistent values in a given run — `go test
-race` (AC-015, `just test-race`) flags it, intermittently reproducing
depending on scheduler timing. The outcome COUNT stays 1 in every run (the
construction-started upstream ticker, with no keystroke enqueued for it to
process beyond AC-017's own single enqueue, never itself calls `failLoud`
in practice) — this is why prior rounds' review, which checks assertion
values, did not catch it: it is a race annotation on the read, not a wrong
count.

**Fix:** §M2's "Mitigation decision" is corrected so the upstream ticker
starts ONLY at `CreateSession` time, SYMMETRIC with the downstream ticker —
not at construction. The "MAY start at construction" / "start-time freedom
PRESERVED" latitude from v1.6 is WITHDRAWN. The single-goroutine,
race-free argument already given for the downstream `CreateSession`-time
start (`CreateSession` is called once, from a single goroutine, before any
`SendKeystroke` — no concurrency, so no synchronization is needed) applies
identically to the upstream ticker; the upstream ticker's earlier freedom
was never load-bearing for anything but AC-017 in the first place, and
AC-017 already required NOT starting the upstream ticker goroutine at all
during its own procedure. The "Driver lifecycle pin" (§H3) and the "Steps
1–3 MUST appear in this order" cross-reference sentence are both updated:
`CreateSession` now starts BOTH tickers, not just the downstream one.
AC-017's method text (§M2) is sharpened from "the test must refrain from
starting the upstream ticker" to a TRUE-BY-CONSTRUCTION guarantee: because
`CreateSession` is now the SOLE start-site for either ticker, and AC-017's
procedure never calls `CreateSession`, no ticker goroutine of any kind can
exist during the AC-017 test — full stop, not contingent on the test's own
discipline. Step 1 ("Do NOT start the upstream ticker goroutine") is kept
in the enumerated procedure but reworded to state it is now automatically
satisfied rather than a separate obligation. Step 4's assertion prose gains
a sentence stating explicitly that the unlocked `len(stub.errorfCalls)`
read is race-free because no concurrent writer can exist on this path,
closing the `go test -race` hazard by construction. The `mu sync.Mutex`
guard on `recordingTB.Errorf` is UNCHANGED and still required for the
general/happy-path case (AC-004/005/006/014 etc.), where `failLoud` may
legitimately fire from a live ticker goroutine; only AC-017's own safety
argument for its specific unlocked assertion read is sharpened, not the
stub's general contract.

**Verification (ground truth, this session):** `internal/session/upstream.go`
`AccessNode.SendKeystroke` (lines 276-301) confirmed: `sinkMu` is taken
first; line 288 `attached, ok := a.consoles.Session(key)`; lines 289-290
`if !ok { return ...ErrConsoleNotFound }`; line 292
`if attached != sessionName { return ...ErrSessionMismatch }` — the
console-attachment gate fires before the `sessionName` comparison,
confirming (independent of F-LENSB-02 below) that AC-017's fault path does
not depend on which ticker is running, only on whether one runs at all.

### F-LENSB-02 (LOW) — `sessionName` pin misattributed AC-017's fault cause

**Defect:** §H3's `sessionName` storage/timing pin credited the empty
`sessionName` as independently sufficient to produce AC-017's
`ErrConsoleNotFound`, "independent of `loopbackConsoleKey`'s own zero-value
state." Disk-verified against `internal/session/upstream.go:276-301`: the
console-attachment gate (`a.consoles.Session(key)` → `ErrConsoleNotFound`
at lines 288-290) fires FIRST, before `sessionName` is compared at line
292. Since `loopbackConsoleKey` is never `Attach`ed before AC-017's call
(that only happens inside `CreateSession`, which AC-017 never calls), the
attachment-gate failure is what actually produces `ErrConsoleNotFound` in
this scenario — the empty `sessionName` value is never read on this path,
so it cannot be the operative cause the prior text described.

**Fix:** §H3's `sessionName` storage/timing pin is reworded to attribute
AC-017's fault to the un-attached, zero-value `loopbackConsoleKey` (the
first-triggered `a.consoles.Session(key)` → `ErrConsoleNotFound` gate,
lines 288-290), with the empty `sessionName` demoted to a REDUNDANT second
guarantee: had a session been published under some other name while
`loopbackConsoleKey` stayed unattached, the same `ErrConsoleNotFound` would
still fire at the attachment gate; had `loopbackConsoleKey` instead been
attached to a real session while `sessionName` stayed `""`, the attachment
gate would pass but the line-292 comparison would then return
`ErrSessionMismatch` — either way a loud, `failLoud`-surfaced error, never
a silent success. AC-017 step 3's existing text ("`ErrConsoleNotFound` (no
console registered)") already matched this and required no change.

### Erratum — `t.Helper()` line-reference correction (`:460` → `:461`)

**Defect:** Two live-text citations of `internal/testenv/testenv.go`'s
`t.Helper()` call site inside `newEnv` read `:460`. Disk-verification this
session confirms the call is actually at line 461 (`b.Helper()` at line
384, `func newEnv` at line 460, `t.Helper()` on the line immediately
after it, `:461`).

**Fix:** The two LIVE-text occurrences — the §M2 `recordingTB` struct
comment's inline citation, and the "Why embedding the real `t` is safe"
prose paragraph's citation — are corrected from `:460` to `:461`. Left
untouched, per §2.9 (frozen dated content) and because they cite a
DIFFERENT file: the two `cmd/switchboard/access.go:460` comparisons (an
unrelated file, correct as-is); the frozen v1.9 changelog row (line ~34 in
the pre-v1.10 numbering) and the frozen "v1.9 R6 Re-Review Repair" section
below, both of which cite `:460` as part of their own dated historical
description of what v1.9 disk-verified at the time — rewriting frozen
history to match a later correction would falsify the record of what that
round actually found and fixed.

### Governance

All three fixes are edits to live spec prose (§M2's "Mitigation decision"
and "AC-017 fault-injection method" subsections, §H3's "Driver lifecycle
pin" and "Steps 1–3" cross-reference and `sessionName` pin) inside sections
already amended across v1.2 through v1.9; no frozen dated changelog row or
prior repair-addendum section was touched. Story and index files are
untouched — per dispatch, this placement-note edit is transcribed into the
story by story-writer separately.

**Ground truth sources (v1.10):** `internal/session/upstream.go:276-301`
(`AccessNode.SendKeystroke` gate ordering — console-attachment check at
lines 288-290 precedes the `sessionName` comparison at line 292) and
`internal/testenv/testenv.go` (`b.Helper()` at line 384, `func newEnv` at
line 460, `t.Helper()` at line 461, `t.Cleanup(func(){...})` at line 475,
`t.Cleanup(e.Close)` at line 528) were both supplied as pre-verified ground
truth for this dispatch and were not independently re-read from disk in
this pass beyond confirming the note's citations against the supplied line
numbers.

## v1.11 R8 Re-Review Repair (2026-08-28)

### Erratum-of-the-erratum — a third `t.Helper()` citation, in slash notation, survived the v1.10 sweep

**Defect (LOW, F-LENSA-01 / NITPICK, F-ORACLE-01):** The v1.10 erratum
("`t.Helper()` line-reference correction (`:460` → `:461`)", above) states
it corrected "two live-text citations" of the `t.Helper()` call site inside
`newEnv` — both in the SPACE-COLON form (`testenv.go:384`, `:461`, ...).
That accounting was complete for the space-colon citation form, but a
THIRD live citation existed in a different notation: the SLASH form
`testenv.go:384/460/475/528`, at the §M2 `recordingTB` struct-field
comment (~L1463, the `NewLoopback call (testenv.go:384/460/475/528)`
inline comment). This slash-form citation packs all four call-site line
numbers into one compact positional reference and was not swept by the
v1.10 erratum's two space-colon fixes, so it still read stale `460` for
the `t.Helper()` position after v1.10 shipped.

**Fix:** The single live slash-form occurrence is corrected from
`384/460/475/528` to `384/461/475/528`. No other slash-form or space-colon
citation exists in live text — the v1.10 erratum's own two fixes (§M2
struct comment prose paragraph, and the "Why embedding the real `t` is
safe" paragraph, both now reading `:461`) are unaffected and untouched;
the two `cmd/switchboard/access.go:460` citations (a different file,
correct as-is) are untouched; the frozen v1.9 changelog row and "v1.9 R6
Re-Review Repair" section's `:460` citations are untouched per §2.9; the
frozen v1.10 changelog row and "v1.10 R7 Re-Review Repair" section
(including its "two citations" erratum text above) are also untouched per
§2.9 — this section corrects the completeness of that claim by addendum,
not by editing the frozen v1.10 text in place.

### Governance

This is a single-line live-text edit inside §M2 (the `recordingTB`
struct-field comment introduced across v1.3–v1.9 and touched again by
v1.10's erratum). No frozen dated changelog row or prior repair-addendum
section was touched, including v1.10's own erratum subsection, which
remains the historical record of what that round found and fixed. Story
and index files are untouched — per dispatch, this placement-note edit is
transcribed into the story by story-writer separately.

**Ground truth source (v1.11):** `internal/testenv/testenv.go` — `b.Helper()`
at line 384, `func newEnv` at line 460, `t.Helper()` at line 461,
`t.Cleanup(...)` at lines 475 and 528 — supplied as orchestrator-verified
ground truth for this dispatch, consistent with the v1.10 ground truth
already on record above.

## v1.12 Consistency-Audit Repair (2026-08-28)

A fresh-context consistency audit (VSDD §1.7) at the Step-4.5 gate found
two MAJOR and one MINOR perimeter defect in this note. Audit record:
`.factory/cycles/cycle-1/S-BL.LOOPBACK-FULLSTACK/consistency-audit-2026-08-28.md`.
All three are resolved in this pass.

### Finding #1 (MAJOR) — phantom-branch citation swept across three occurrences

**Defect:** The note characterized the merged benchmark file
`internal/bench/keystroke_echo_testenv_bench_test.go` as WIP on a branch,
`fix/vp-042-testenv-integrated-bench`, at three live-text locations (§Q2
~L130 "the existing WIP bench test", Package Impact Summary ~L613, Risks
item 3 ~L642-643). That branch does not exist — verified absent from both
`git branch --list` and `git ls-remote` — and the file is in fact already
merged to `develop` via PR #121 (`4c276d9`, "fix(VP-042): testenv-integrated
keystroke-echo p99 benchmark (#121)", 2026-07-12).

**Fix:** All three occurrences corrected to state the file is merged on
`develop` (PR #121 @ `4c276d9`), `//go:build integration`-tagged, defining
`BenchmarkKeystrokeToEcho_P99`. §Q2's citation — which independently
confirms `LoopbackEnv`'s named-field-not-embedding shape via the merged
file's `env := lb.Env; env.CreateSession(b)` call pattern — is otherwise
unchanged; that code-shape claim was re-verified against the merged file
this session and remains accurate.

### Finding #2 (MAJOR) — AC-013 verified nothing

**Defect:** AC-013's verification intent, as it stood prior to this
repair, cited a bare `go build ./internal/bench/...` and `just bench` as
proof the updated benchmark works. Both are incapable of exercising
`BenchmarkKeystrokeToEcho_P99`: the merged file carries `//go:build
integration`, which a tag-less `go build` excludes from compilation
entirely, and `just bench` (`justfile:53-66`) invokes `go test
-bench=BenchmarkKeystrokeEcho_P99 ...` — no `-tags integration`, and a
DIFFERENT function name (no "To") belonging to the separate, untagged
`keystroke_echo_bench_test.go` (S-BL.BENCH lower-bound).

**Fix:** Added a new "### Verification Method (AC-013)" subsection (below
"## What VP-042's Benchmark Looks Like Against This") specifying the
correct compile + run commands — `go build -tags integration
./internal/bench/...` (and/or `go vet -tags integration
./internal/bench/...`) for compile, and `go test -tags integration -run
'^$' -bench=BenchmarkKeystrokeToEcho_P99 -benchtime=1x -count=1
./internal/bench/` for run — matching the "Run with:" block the merged
file's own doc comment already prescribes. Explicit adjudication
recorded: **Option (a)** — `just bench` remains the tag-free S-BL.BENCH
lower-bound recipe, unmodified and out of AC-013's scope; AC-013 verifies
via the direct `go test -tags integration ...` command above. Rationale:
this story's own Story-Sizing Estimate already commits to "no new
CI/deployment surface... everything is additive inside
`internal/testenv`" — extending or duplicating the justfile would be
exactly that kind of surface change, unmotivated by anything else in this
design; the merged file already documents its own invocation convention
independent of any justfile recipe, so citing that same command is the
minimal-diff, already-established path.

### Finding #4 (MINOR) — forward obligation for VP-042.md not recorded

**Defect:** The note's Risks section (five items) never stated that
VP-042.md's Proof Harness Skeleton — explicitly acknowledged elsewhere in
this note as diverging from the binding API this design specifies (see
"What VP-042's Benchmark Looks Like Against This": "a small, deliberate
divergence from the VP-042.md proof-harness skeleton's exact call shape")
— needs updating to match. No "Forward Obligation" section existed to
confirm or deny coverage.

**Fix:** Added Risks item 6, a one-line forward obligation directing
story-writer to encode a File-List obligation updating VP-042.md's Proof
Harness Skeleton section to the binding two-call token shape
(`SendKeystroke` → `RoundTrip`; `WaitForEcho(t, rt, timeout) ([]byte,
bool)`) this note specifies.

### Governance

Findings #1 and #4 are edits/additions to live spec prose (§Q2, Package
Impact Summary, Risks) inside sections already amended across v1.2
through v1.11; no frozen dated changelog row or prior repair-addendum
section was touched. Finding #2 is a net-new subsection — no prior
AC-013 verification content existed in the note to correct in place; the
incorrect framing the audit found lived at the story level, downstream of
this note. Story and index files are untouched — per dispatch, this
placement-note edit is transcribed into the story by story-writer
separately.

**Ground truth sources (v1.12):** `git branch --list` / `git ls-remote`
(phantom branch absent — zero results for either); `git log develop` /
`git show develop:internal/bench/keystroke_echo_testenv_bench_test.go`
(PR #121, `4c276d9`, `//go:build integration` tag, `BenchmarkKeystrokeToEcho_P99`
definition, "Run with:" doc comment, `env := lb.Env; env.CreateSession(b)`
call pattern); `git show develop:internal/bench/keystroke_echo_bench_test.go`
(sibling untagged `BenchmarkKeystrokeEcho_P99`, no "To"); the repo
`justfile` §`bench` recipe (`justfile:53-66`, `BenchmarkKeystrokeEcho_P99`,
no `-tags integration`) — all supplied by the orchestrator's dispatch fact
base and independently re-confirmed against disk in this session.
