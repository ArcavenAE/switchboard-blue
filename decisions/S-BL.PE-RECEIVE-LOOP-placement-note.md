---
artifact_id: S-BL.PE-RECEIVE-LOOP-placement-note
document_type: architect-placement-note
story_id: S-BL.PE-RECEIVE-LOOP
title: "PE-connection receive/forward loop placement, frame-type design, arqsend wiring, and E-FWD-001 discharge for S-BL.PE-RECEIVE-LOOP"
status: final
producer: architect
timestamp: 2026-07-08T00:00:00Z
version: "1.2"
bc_traces:
  - BC-2.02.008   # PC-3/EC-003 E-FWD-001 exhaustion (postcondition 1 re-anchored from S-7.04-FU-PE-CONNECTOR AC-004)
  - BC-2.06.003   # PC-1 Failed-state observable via retransmit-driven path exhaustion
vp_traces: []    # no VP ownership in this story; VP-037 unblock path runs through S-7.04-FU-DRAIN-WIRE
forward_obligations_consumed:
  - FO-PE-LOOP-001   # Define frame.FrameTypePEConnect / adopt FrameTypeCtl; flip dialLoop bootstrap
subsystems: [deployment-operations, transport-layer]
architecture_modules:
  - cmd/switchboard
  - internal/upstreamdial
  - internal/routing
  - internal/arqsend
  - internal/netingress
  - internal/testenv
---

## Changelog

| Version | Change |
|---------|--------|
| 1.0 | Initial release. Full backtick-symbol sweep (Appendix A) performed prior to publication; all symbols verified against tree at `8eb54a5` (S-7.04-FU-PE-CONNECTOR merge SHA). |
| 1.1 | Remediate five spec-adversarial pass-1 findings: F-SP1-001 (HIGH [spec-defect]) — new Q8 ruling specifies FrameArrivalHandler-based wiring with full dependency construction; F-SP1-002 (HIGH [spec-gap]) — Q3 blast-radius enumeration for Valid() widening (test amendments + doc-comment updates); F-SP1-003 (HIGH [spec-gap]) — Q3 adds ARCH-02 frame_type table amendment obligation; F-SP1-005 (MED [spec-gap]) — Q6 strengthened with explicit per-reconnect-iteration join requirement; F-SP1-006 (MED [doc-drift]) — Q1 contradiction with Q2 annotated with explicit supersession. Appendix A updated with new symbols from Q8. |
| 1.2 | Remediate four spec-adversarial pass-2 findings: F-SP2-001 (CRITICAL [spec-defect]) — new Q9 ruling supersedes Q4/Q5 injection topology: arqsend `Dispatch` must NOT dial `ListenAddr`; the upstream fixture MUST write directly to the accepted PE connection; option (b) ruled (fixture assembles + writes frame directly; arqsend obligation audited and narrowed); F-SP2-002 (HIGH [spec-gap]) — Q9 specifies write-capable upstream fixture shape, placement (test-local, same file as other runRouter integration tests), and exact API (accepted-conn handle + `WriteFrame(wire []byte) error` method); F-SP2-003 (MED [spec-defect]) — Q9 mandates harness rule: every AC asserting OnFrameArrival must use the real runRouter goroutine pattern (not testenv.Restart which bypasses SetFrameCallback); F-SP2-004 (MED [doc-drift]) — Q3 blast-radius amended with two missed frame_test.go locations (`TestParseOuterHeader_AcceptsAllValidFrameTypes` "all five" comment and 5-element `valid` slice). Adjudicated-clean section added for five pass-2 non-findings (per F-SP2-001 report). Appendix A delta added for new fixture symbols. |

# Architect Placement Note: PE-Connection Receive/Forward Loop
## Story: S-BL.PE-RECEIVE-LOOP

This note answers seven design questions required to unblock story elaboration
and scheduling. All file anchors refer to the `develop` branch at HEAD `8eb54a5`
(S-7.04-FU-PE-CONNECTOR merge SHA). Rulings are binding for the story-writer
and implementer. Every API derivation block is grep-verified against the tree at
this SHA; see Appendix A for the symbol-sweep disposition table.

---

## Q1 — Receive goroutine ownership: inside `upstreamdial.Connector` vs wiring in `cmd/switchboard`

**Ruling: the receive goroutine lives inside `internal/upstreamdial.Connector`,
one goroutine per established connection, started after step-3 success in
`dialLoop`. `cmd/switchboard/mgmt_wire.go` is not modified for the receive-loop
itself — `runRouter` receives only a callback or channel seam added to the
`upstreamdial.Handle` interface. This keeps `internal/upstreamdial` routing-free
per the forbidden-edge constraint.**

**Derivation:**

ARCH-08 §6.6.2 at `8eb54a5` defines the forbidden edges for `internal/upstreamdial`:

> Forbidden: `drain`, `routing`, `testenv`, and any package at positions 20–23.

`internal/routing` is at position 17. A direct import of `routing` by
`upstreamdial` would be a NEW edge between positions 19 → 17 — numerically
acyclic but **functionally forbidden** because ARCH-08 explicitly lists
`routing` in the forbidden set. The placement note for S-7.04-FU-PE-CONNECTOR
(Q4, §"Forbidden edges") is unambiguous:

> `internal/upstreamdial` → `internal/routing` (routing is a boundary; connector
> does not participate in frame forwarding)

Two implementation shapes preserve this constraint:

**(a) Callback seam:** `Handle` gains a method
`SetFrameCallback(fn func([]byte) error)`. After step-3 success in `dialLoop`,
the receive goroutine calls `fn` for each raw frame. `runRouter` wires a closure
at construction, passing it to the connector. This is the same pattern
`netingress.ServeConn` already uses for `RouteFn`.

**(b) Channel seam:** the `Connector` exposes a `chan []byte` of received frames;
`runRouter` drains it in its own goroutine. This doubles the goroutine count and
buffers unconsumed frames, adding backpressure complexity for no benefit.

**Ruling: option (a) — callback seam.** It mirrors the existing `netingress.RouteFn`
pattern exactly and keeps the `Connector` as the goroutine owner without importing
`routing`.

**ARCH-08 obligation:** `internal/upstreamdial`'s allowed-import set DOES NOT
change — the callback receives `[]byte` raw frames at the connector boundary,
which requires only stdlib at the connector layer. The `Handle` interface gains a
setter; the `Connector` struct gains the callback field. Both changes are internal
to the existing registered package. No §6.4 registration is required by this
story.

> **[v1.1 supersession note — F-SP1-006]** The sentence in this Q1 derivation
> that read "No new import row is needed" and described the callback signature as
> `func([]byte) error` is superseded by Q2's ruling. Q2 rules that the connector
> callback signature is `type FrameFn func(hdr frame.OuterHeader, raw []byte) error`
> (not `func([]byte) error`) and that `upstreamdial` gains a direct `frame` import
> (ARCH-08 §6.5 amendment required). Q2 also rules that `upstreamdial.Handle`
> gains `SetFrameCallback(fn FrameFn)` (not `SetFrameCallback(fn func([]byte) error)`).
> Q1's routing-free constraint, goroutine placement decision, and option (a)
> callback-seam choice remain operative. The specific import and signature details
> are determined by Q2, which is authoritative. The v1.0 Q1 text is preserved here
> per factory history-preservation policy; read Q2 as the controlling specification
> for all type and import details.

**Cite:** ARCH-08 §6.6.2 forbidden-edges note at `8eb54a5`;
`internal/upstreamdial/connector.go` `dialLoop` structure (verified at `8eb54a5`);
`internal/netingress/netingress.go` `RouteFn` pattern (verified at `8eb54a5`).

---

## Q2 — Frame path: framing/deframing mechanism for incoming bytes on a PE connection

**Ruling: `netingress.ReadFrame` is reused for framing. After callback wiring, the
receive goroutine calls `netingress.ReadFrame(conn)` in a loop, then invokes the
caller-supplied callback with the raw frame bytes (outer header + payload as a
single slice). No new framing primitive is introduced.**

**Derivation from the `netingress` API (verified at `8eb54a5`):**

```go
// internal/netingress/netingress.go
// ReadFrame reads exactly one framed message from r: OuterHeaderSize bytes
// followed by hdr.PayloadLen bytes of payload. Returns the parsed header
// and payload slice.
func ReadFrame(r io.Reader) (frame.OuterHeader, []byte, error)
```

`ReadFrame` is self-delimiting via `OuterHeader.PayloadLen` (44-byte outer header
carries a `uint16` payload length). It is the canonical framing primitive used by
`netingress.ServeConn` for all incoming connections. The wire format on PE upstream
connections is identical — `outerassembler.Assemble` on the sending side produces a
wire frame consumed byte-for-byte by `ReadFrame` on the receiving side (documented
in `assemble.go`'s package comment at `8eb54a5`).

The callback signature at the boundary layer is therefore:

```go
// In internal/upstreamdial, passed to SetFrameCallback:
type FrameFn func(hdr frame.OuterHeader, raw []byte) error
```

This matches the shape of `netingress.RouteFn` and avoids a new type. The
`Connector`'s receive goroutine calls `netingress.ReadFrame(conn)` then
`frameFn(hdr, payload)`.

**Import note:** `netingress` is at position 18 (ARCH-08 §6.5). The
`Connector` at position 19 may NOT import `netingress` (that would be a
back-edge: 19 → 18). However, the `Connector`'s receive goroutine only needs to
call `ReadFrame` — a function that takes `io.Reader`. The legal options are:

1. **Inline the ReadFrame logic** in `upstreamdial` (re-implement the 44-byte
   outer-header read + payload read). This duplicates protocol knowledge.
2. **Extract `ReadFrame` to a position ≤ 19 package.** `netingress` is a
   boundary package (position 18) that also owns `Serve`/`ServeConn`/semaphore
   — too much to decompose for this. The reading primitive itself only imports
   `internal/frame` (position 2). A thin extract to a helper at
   `internal/netingress` is impossible given the position constraint.
3. **Lift `ReadFrame` to `internal/frame` or a new lower-layer package.**
   `internal/frame` (position 2) owns the outer-header codec
   (`ParseOuterHeader`, `EncodeOuterHeader`). Adding a `ReadFrame`-equivalent
   there (`ReadOuterFrame(io.Reader) (OuterHeader, []byte, error)`) would be
   at position 2 — importable by `upstreamdial` at position 19 without a
   back-edge. **This is the ruling.**

**Option 3 ruling:** A new function `frame.ReadOuterFrame(r io.Reader) (OuterHeader, []byte, error)` is added to `internal/frame/frame.go` (position 2). It implements the same read-header-then-payload logic as `netingress.ReadFrame`. `netingress.ReadFrame` may delegate to it (reducing duplication) or retain its own copy with a cross-reference comment — the implementer's choice.

The `upstreamdial` receive goroutine calls `frame.ReadOuterFrame(conn)` and
passes the result to the `FrameFn` callback. No import-graph change: `upstreamdial`
already has a transitive path to `frame` through `halfchannel` and `outerassembler`;
a direct `frame` import at position 19 is lawful (frame is position 2).

**ARCH-08 obligation:** `internal/upstreamdial` gains a direct import edge to
`internal/frame` (position 2). The allowed-import row must be updated in §6.5:
`{halfchannel, outerassembler}` → `{frame, halfchannel, outerassembler}`.
This is a §6.4 amendment (import-set extension of an existing package, not a
new package). The story implementer must update ARCH-08 §6.5 in the same commit
that introduces the `frame.ReadOuterFrame` import.

**Cite:** `internal/frame/frame.go` `ParseOuterHeader` (verified at `8eb54a5`);
`internal/netingress/netingress.go` `ReadFrame` (verified at `8eb54a5`);
ARCH-08 §6.5 position table.

---

## Q3 — FO-PE-LOOP-001 discharge: define `frame.FrameTypePEConnect` vs adopt `frame.FrameTypeCtl`

**Ruling: define a new constant `frame.FrameTypePEConnect` at the next available
value. The current five canonical values are `0x01`–`0x05` (verified at `8eb54a5`).
`FrameTypePEConnect` MUST be assigned value `0x06`. The receive loop discriminates
bootstrap frames from session data by checking `hdr.FrameType == frame.FrameTypePEConnect`
after `frame.ReadOuterFrame`. Bootstrap frames are consumed (dropped or ACK'd)
by the receiver; data frames are forwarded through the callback.**

**Why not `FrameTypeCtl` (0x03)?**

`FrameTypeCtl` (0x03) is defined in `internal/frame/frame.go` (verified at `8eb54a5`)
as a generic control-plane frame type. The placement note for S-7.04-FU-PE-CONNECTOR
(Q6, F-P28-001 correction) cites it as the "control-category constant" — but that
story deferred the specific PE-CONNECT constant to this story with rationale:

> "using `halfchannel.FrameTypeData` as placeholder makes bootstrap frames
> indistinguishable from session data at the receiver."

Adopting `FrameTypeCtl` would disambiguate bootstrap from data, but would conflate
PE-CONNECT with other future control-plane messages (keepalive ACKs, RESYNC,
DRAIN). A distinct `FrameTypePEConnect` is needed so the receive loop can apply
the right handler without a secondary discriminator field in the channel header.

**`frame.FrameType.Valid()` update obligation — full blast radius (F-SP1-002 + F-SP1-003):**

`internal/frame/frame.go` (verified at `8eb54a5`) currently defines:

```go
func (f FrameType) Valid() bool {
    return f >= FrameTypeData && f <= FrameTypeFec
}
```

With `FrameTypeFec = 0x05`, this accepts `0x01`–`0x05` and rejects `0x06`.
Adding `FrameTypePEConnect = 0x06` REQUIRES updating `Valid()` to
`return f >= FrameTypeData && f <= FrameTypePEConnect` (or the widened upper bound).
Failing to update `Valid()` will cause `frame.ParseOuterHeader` to return
`ErrInvalidFrameType` for every PE-CONNECT bootstrap frame received, silently
dropping all upstream bootstraps.

The `Valid()` widening has a full blast radius that the implementer MUST sweep and
remediate. Grep-verified against `8eb54a5`:

**Test amendments required (F-SP1-002):**

1. `internal/frame/frame_test.go` — `TestFrameType_Valid` table (verified at `8eb54a5`,
   lines containing the `just_above_max` case):
   - Current: `{"just_above_max", frame.FrameType(0x06), false}` — this case MUST be
     changed to `{"just_above_max", frame.FrameType(0x07), false}` because `0x06` will
     become `FrameTypePEConnect` (valid). The test name "just_above_max" remains accurate
     for `0x07` (one above the new max `0x06`). Verified: `frame_test.go` contains
     `{"just_above_max", frame.FrameType(0x06), false}` at `8eb54a5`.

2. `internal/frame/frame_test.go` — `TestParseOuterHeader_RejectsInvalidFrameType`
   (verified at `8eb54a5`):
   - Current: `invalids := []byte{0x00, 0x06, 0x77, 0xFF}` — the `0x06` entry MUST be
     changed to `0x07` (or any value `>= 0x07`) because `0x06` will no longer be invalid.
     Verified: `frame_test.go` contains `invalids := []byte{0x00, 0x06, 0x77, 0xFF}` at `8eb54a5`.

**Doc-comment updates required (F-SP1-002):**

3. `internal/frame/frame.go` `FrameType` type comment (verified at `8eb54a5`):
   - Current: `"Only five values are canonical; all others are reserved."` — MUST be
     updated to reflect six canonical values. Verified: `frame.go` line 27 contains
     `"Only five values are canonical; all others are reserved."` at `8eb54a5`.

4. `internal/frame/frame.go` `Valid()` doc comment (verified at `8eb54a5`):
   - Current: `"Valid reports whether the FrameType byte is one of the five canonical
     enum values defined in ARCH-02 §3.1. Returns false for 0x00 and 0x06..0xFF."` —
     MUST be updated: "six canonical enum values" and "Returns false for 0x00 and
     0x07..0xFF". Verified: `frame.go` lines 40–41 contain this text at `8eb54a5`.

5. `internal/frame/frame.go` `ErrInvalidFrameType` doc comment (verified at `8eb54a5`):
   - Current: `"not one of the five canonical FrameType values (per ARCH-02 §3.1)"` —
     MUST be updated to "six canonical" or "not in {0x01..0x06}". Verified: `frame.go`
     lines 47–48 contain this text at `8eb54a5`.

**Amended blast-radius (F-SP2-004 — two locations missed in v1.1):**

The v1.1 Q3 sweep was incomplete. Two additional `frame_test.go` locations require amendment
(grep-verified against `8eb54a5`):

6. `internal/frame/frame_test.go` — `TestParseOuterHeader_AcceptsAllValidFrameTypes` function
   doc comment (verified at `8eb54a5`, located at the comment block beginning "TestParseOuterHeader_AcceptsAllValidFrameTypes
   asserts that all five canonical FrameType values pass ParseOuterHeader's enum validation"):
   - Current: `"all five canonical FrameType values"` — MUST be updated to `"all six canonical
     FrameType values"`. Verified: this comment exists immediately above the function at `8eb54a5`.

7. `internal/frame/frame_test.go` — `TestParseOuterHeader_AcceptsAllValidFrameTypes` `valid`
   slice (verified at `8eb54a5`):
   - Current: 5-element slice `{FrameTypeData, FrameTypeEmptyTick, FrameTypeCtl, FrameTypeArq, FrameTypeFec}` — MUST have `frame.FrameTypePEConnect` appended as the sixth element. This is the regression guard that `Valid()` accepts the new constant; without it, a future narrowing of `Valid()` could silently break PE-CONNECT bootstrap parsing. Verified: the function body contains this exact 5-element slice at `8eb54a5`.

**Extended sweep transcript (F-SP2-004 re-sweep, broader patterns):**

The following grep patterns were run against `*.go` files at `8eb54a5` to satisfy the F-SP2-004
re-sweep requirement:

- `grep -rn "five" --include="*.go" internal/frame/` → hits at `frame_test.go` lines 501, 560 (both now enumerated above), and no other `frame/` files.
- `grep -rn "0x05" --include="*.go" internal/frame/` → hits at `frame.go` (FrameTypeFec constant definition) and `frame_test.go` (test data bytes — not FrameType assumptions; these are payload bytes in round-trip tests, not Valid() range bounds). No additional Valid()-range assumptions found.
- `grep -rn "FrameTypeFec" --include="*.go" .` (excluding `.factory/`) → hits: `internal/frame/frame.go` (constant definition), `internal/frame/frame_test.go` (test data), `internal/outerassembler/fuzz_test.go`. The `fuzz_test.go` hit is at the `ft.Valid()` gate pattern — it does NOT hard-code a range bound; `ft.Valid()` auto-adjusts when `Valid()` is widened. Verified: `fuzz_test.go` line 128 reads `if !ft.Valid() { return }` (adversary pass-2 confirmed self-adjusting; recorded as swept-clean per F-SP2-001 adjudication section below).
- `grep -rni "five" --include="*.md" .factory/specs/` → hits in ARCH-02 are the `fec=0x05` value in the `frame_type` table row, which is a value description not a count claim, and is already covered by the ARCH-02 amendment obligation in Q3. No additional count-five claims found in spec docs.

**All seven blast-radius locations now enumerated (complete list):**

| # | Location | Required change |
|---|----------|-----------------|
| 1 | `frame_test.go` `TestFrameType_Valid` `just_above_max` case | `FrameType(0x06) → FrameType(0x07)` |
| 2 | `frame_test.go` `TestParseOuterHeader_RejectsInvalidFrameType` `invalids` slice | `0x06 → 0x07` in slice |
| 3 | `frame.go` `FrameType` type doc comment | `"Only five values"` → `"Only six values"` |
| 4 | `frame.go` `Valid()` doc comment | `"five canonical…0x06..0xFF"` → `"six canonical…0x07..0xFF"` |
| 5 | `frame.go` `ErrInvalidFrameType` doc comment | `"five canonical"` → `"six canonical / not in {0x01..0x06}"` |
| 6 | `frame_test.go` `TestParseOuterHeader_AcceptsAllValidFrameTypes` doc comment | `"all five canonical"` → `"all six canonical"` (F-SP2-004) |
| 7 | `frame_test.go` `TestParseOuterHeader_AcceptsAllValidFrameTypes` `valid` slice | Append `frame.FrameTypePEConnect` as sixth element (F-SP2-004) |

No other files in the tree at `8eb54a5` contain "five values" or "five canonical"
assumptions anchored to the `0x05` upper bound per the extended sweep above. The
`outerassembler/fuzz_test.go` `ft.Valid()` gate is self-adjusting and requires no
change (swept-clean; see adjudicated-clean section below).

**ARCH-02 amendment obligation (F-SP1-003):**

ARCH-02 §"Outer Header Format" (at `.factory/specs/architecture/ARCH-02-protocol-stack.md`,
verified at `8eb54a5`) contains the canonical single source of truth for the wire
frame types. The `frame_type` row currently reads:

```
| 1 | 1 | frame_type | u8 enum: data=0x01, empty_tick=0x02, ctl=0x03, arq=0x04, fec=0x05 |
```

`FrameTypePEConnect = 0x06` goes on the wire — it is the bootstrap frame type
the PE upstream connection uses. ARCH-02 is declared the "canonical single source
of truth for the outer header wire format" (verified at `8eb54a5` in ARCH-02 preamble).
The implementer MUST amend the `frame_type` row in the same commit that defines
`FrameTypePEConnect`:

```
| 1 | 1 | frame_type | u8 enum: data=0x01, empty_tick=0x02, ctl=0x03, arq=0x04, fec=0x05, pe_connect=0x06 |
```

This is a parallel obligation to the ARCH-08 §6.5 amendment required by Q2 — both
spec documents require the same commit. Additionally, `internal/frame/frame.go`
line 31 contains the comment `"Frame type constants (ARCH-02 §3.1)"` — this comment
remains accurate and does not require amendment, but the new constant MUST appear
beneath it with an `(ARCH-02 §3.1)` inline citation.

**`dialLoop` bootstrap flip obligation (FO-PE-LOOP-001):** `internal/upstreamdial/connector.go`
`dialLoop` (verified at `8eb54a5`) currently sets:

```go
cf := halfchannel.ChannelFrame{
    FrameType: halfchannel.FrameTypeData,
}
```

This story flips it to:

```go
cf := halfchannel.ChannelFrame{
    FrameType: frame.FrameTypePEConnect,
}
```

`halfchannel` aliases only `FrameTypeData` and `FrameTypeEmptyTick` (verified at
`8eb54a5`). `frame.FrameTypePEConnect` must be imported directly from
`internal/frame`. This introduces a direct `frame` import in `upstreamdial` —
consistent with the Q2 ruling (same import-set extension, covered by the same
§6.4 amendment).

**Receive loop discrimination contract:** after `frame.ReadOuterFrame`, the
receive goroutine applies:

```
if hdr.FrameType == frame.FrameTypePEConnect {
    // bootstrap acknowledgment path (or silent discard if no reply needed)
} else {
    // data/ctl/arq/fec frame — pass to FrameFn callback
}
```

The exact bootstrap acknowledgment protocol is determined by the story-writer at
elaboration time. If no reply is defined in this story's scope, bootstrap frames
are silently discarded; the upstream router's PE-CONNECT is treated as a
registration event only.

**Cite:** `internal/frame/frame.go` (FrameType constants, Valid(), doc comments, verified at `8eb54a5`);
`internal/frame/frame_test.go` (`TestFrameType_Valid` just_above_max case, `TestParseOuterHeader_RejectsInvalidFrameType` invalids slice, verified at `8eb54a5`);
`internal/halfchannel/halfchannel.go` (FrameType aliases, verified at `8eb54a5`);
`internal/upstreamdial/connector.go` `dialLoop` bootstrap construction (verified at `8eb54a5`);
`.factory/specs/architecture/ARCH-02-protocol-stack.md` frame_type row (verified at `8eb54a5`).

---

## Q4 — `arqsend.Retransmitter` wiring into `runRouter`

**Ruling: `arqsend.New` is called inside `runRouter` after the connector is
constructed and started (Phase g, after Phase f). The `Retransmitter` is used
only in the integration test that exercises E-FWD-001 under sustained load; it
is NOT wired into the production `runRouter` datapath for this story. A
per-test construction inside the test body is the correct shape.**

**Derivation from the `arqsend` API (verified at `8eb54a5`):**

```go
// internal/arqsend/arqsend.go
func New(a *arq.ARQ, env outerassembler.Envelope, opts ...Option) *Retransmitter
func (r *Retransmitter) Retransmit(oldSeq, newSeq uint32, now time.Time, dispatch Dispatch) error
type Dispatch func(wire []byte) error
```

`arqsend.Retransmitter` is pure-core (no goroutines, no I/O). Its `Retransmit`
method requires an `*arq.ARQ` and an `outerassembler.Envelope`. In the integration
test context:

- A test-internal `*arq.ARQ` (constructed via `arq.New`) tracks retransmit state.
- The `Dispatch` callback sends wire bytes to the test router's data-plane listener
  address via `net.Dial` + `conn.Write` — the same loopback pattern
  `TestRunRouter_PE_EFWD001ReconfirmationUnderLoad` used in S-7.04-FU-PE-CONNECTOR
  (verified at `8eb54a5`).

The production `runRouter` does NOT need a persistent `Retransmitter` instance:
the production ARQ retransmit path is node-side (access nodes retransmit), not
router-side. Wiring a `Retransmitter` into `runRouter` would be production-dead
code outside this test's scope.

**Test construction point:** inside `TestRunRouter_PE_EFWD001ExhaustionUnderLoad`
(or equivalent name at elaboration), after the PE router is started and the
upstream fixture connection is established:

```go
a := arq.New(arq.Config{...})
rt := arqsend.New(a, outerassembler.Envelope{}, arqsend.WithChanID(1))
// dispatch: write wire bytes to the router's ListenAddr
dispatch := func(wire []byte) error {
    conn, err := net.Dial("tcp", routerListenAddr)
    if err != nil { return err }
    defer conn.Close()
    _, err = conn.Write(wire)
    return err
}
```

**Lifecycle:** the `Retransmitter` has no Stop/Close method (pure-core, no
goroutines). Its lifecycle is bounded to the test function.

**`cmd/switchboard` import impact:** `arqsend` is already imported transitively
through the test binary; no new production import is needed.

**Cite:** `internal/arqsend/arqsend.go` `New`, `Retransmit`, `Dispatch`
(verified at `8eb54a5`); `internal/arq/arq.go` `New` (verified at `8eb54a5`);
`cmd/switchboard/router_pe_connector_test.go` `TestRunRouter_PE_EFWD001ReconfirmationUnderLoad`
(loopback pattern, verified at `8eb54a5`).

---

## Q5 — E-FWD-001 exhaustion integration test shape

**Ruling: the test wires a real PE connection between a test router and an
upstream fixture, sends frames via `arqsend.Retransmitter.Retransmit` (dispatch
writes to the router's data-plane `ListenAddr`), and asserts `"E-FWD-001"` appears
in the router's writer output. Path exhaustion is achieved by setting the
`interfaceSet` to `[]routing.InterfaceID{arrivalIface}` only — i.e., the arrival
interface is the only forwarding candidate — which causes
`FrameArrivalHandler.OnFrameArrival` to call `SplitHorizon.Forward` with no
eligible output interface and emit E-FWD-001.**

**Exact E-FWD-001 emission string (verified at `8eb54a5`, from
`internal/routing/on_frame_arrival.go`):**

```
"all paths split-horizon-blocked: frame dropped (checksum=0x%08x iface=%d) (BC-2.02.008 E-FWD-001)"
```

**Assertion key:** `"E-FWD-001"` — the spec-anchored event code, NOT
`"split-horizon-blocked"` or `"all paths split-horizon"`. This is the lesson
from F-P11-001 (adversary pass-11 of S-7.04-FU-PE-CONNECTOR, committed at
`8eb54a5`): space vs hyphen mismatches make a vacuous negative assertion. Use the
event code tag that is stable across prose rewording of the emission text.
The mutation pin test `TestScanForLine_DetectsEFWD001ProductionEmission`
(verified at `8eb54a5` in `router_pe_connector_test.go`) validates this key.

**Test infrastructure required:**

The `testenv` package (position 23, verified at `8eb54a5`) provides
`testenv.NewWithRouters(ctx, t, n int)` which starts `n` in-process routers.
`testenv.New(ctx, t)` starts a single-router environment. For the E-FWD-001
exhaustion test, a single router with one PE upstream fixture is sufficient:

1. Start the test router via `testenv.New(ctx, t)`.
2. The test PE upstream is the existing `peLn` fixture listener already
   created inside `newEnv` (verified at `8eb54a5` in `testenv.go` — it is
   a `net.Listen("tcp", "127.0.0.1:0")` that accepts and drains connections,
   available via `e.PERouterAddr(t)`).
3. Restart the test router with `UpstreamRouters: []string{e.PERouterAddr(t)}`
   via `RouterHandle.Restart(t, cfg)` so the connector dials the fixture.
4. Wait for the receive loop to be active (poll `RouterHandle.Mode()` for `ModePE`).
5. Send frames via `arqsend.Retransmitter` dispatching to the router's
   `cfg.ListenAddr` (the data-plane TCP listener). Frames must be
   well-formed `outerassembler.Assemble` output to pass `ParseOuterHeader` and
   reach `FrameArrivalHandler.OnFrameArrival`.
6. Path exhaustion requires the forwarding table to have only the arrival
   interface as the eligible interface. Achieving this in `testenv` requires
   the story-writer to assess: does `testenv`'s routing table naturally have
   only one registered interface (the incoming node connection), making any
   frame from the upstream fixture arrive with the same `InterfaceID` as its
   only forwarding entry? If so, no special setup is needed. If not, a
   dedicated loopback fixture must pre-register a forwarding entry pointing
   back to the arrival interface. This is an elaboration decision.

**No new `testenv` API beyond what already exists is required** for Q5. The
existing seams (`NewWithRouters`, `PERouterAddr`, `RouterHandle.Restart`,
`RouterHandle.Mode`, `Env.StartRouter`) are sufficient.

**Cite:** `internal/testenv/testenv.go` `NewWithRouters`, `New`, `PERouterAddr`,
`StartRouter`, `RouterHandle.Restart`, `RouterHandle.Mode` (verified at `8eb54a5`);
`internal/routing/on_frame_arrival.go` E-FWD-001 emission line (verified at `8eb54a5`);
`cmd/switchboard/router_pe_connector_test.go` `TestScanForLine_DetectsEFWD001ProductionEmission`
(F-P11-001 mutation pin, verified at `8eb54a5`).

---

## Q6 — Concurrency contract: receive goroutine lifecycle vs `Connector.Stop`/`ReloadAddrs`

**Ruling: the receive goroutine is owned by `dialLoop` and exits when the
per-address context (`ctx context.Context` in `dialLoop`) is cancelled. It MUST
NOT hold any shared mutable state beyond the `net.Conn` passed to it by `dialLoop`.
`conn.Close()` (called by `dialLoop` after `maintainConn` returns) signals the
receive goroutine's `frame.ReadOuterFrame` loop to exit via `io.EOF` or
`net.Error`. No separate stop channel is needed for the receive goroutine; it
drains naturally when the connection closes.**

**Derivation:**

The shipped `dialLoop` in `internal/upstreamdial/connector.go` (verified at
`8eb54a5`) follows this pattern for each established connection:

1. `conn, err := dialer.DialContext(ctx, "tcp", addr)` — dial
2. `outerassembler.Assemble` + `conn.Write` — bootstrap
3. `c.connectedCount.Add(1)` — increment
4. `c.maintainConn(addr, conn, ctx.Done(), keepaliveTick.C)` — blocks until connection dead or stop
5. `newCount := c.connectedCount.Add(-1)` + `_ = conn.Close()` — teardown

The receive goroutine must be started between steps 3 and 4, and must use the
same per-address `ctx.Done()` channel as `maintainConn`. When `ctx` is cancelled
(via `addrCancel.cancel()` from `reconcile` or via `stopCh` close from `Stop()`),
the per-address context is cancelled, `DialContext` returns, and any ongoing
`frame.ReadOuterFrame(conn)` returns because the underlying `net.Conn` is closed.

**Exactly-once semantics:** the concurrent-transition lesson from F-P29-001
(EC-004 concurrent-drop race, S-7.04-FU-PE-CONNECTOR) applies symmetrically to
the receive loop. The receive goroutine MUST NOT access `c.connectedCount` or any
other shared state. The `connectedCount` lifecycle is owned by `dialLoop` alone
(increment after step 3, decrement via `Add(-1)` return value after step 5). The
receive goroutine's only output is calling the `FrameFn` callback with received
bytes — a stateless action from the concurrency perspective.

**Goroutine lifecycle contract:**

```
dialLoop goroutine:
    1. dial
    2. bootstrap
    3. connectedCount.Add(+1)
    4. START receive goroutine (owns conn, ctx.Done())
    5. maintainConn(addr, conn, ctx.Done(), tick)  ← blocks
    6. receive goroutine exits (conn closed or ctx done)
    7. connectedCount.Add(-1) — must occur AFTER receive goroutine exits
       OR be independent of receive goroutine state (no shared write)
    8. conn.Close()
    9. [if reconnecting] WAIT for receive goroutine from previous iteration to
       fully exit before beginning step 1 of the next dial iteration
```

The ordering between steps 6 and 7 is not constrained by shared state — the
receive goroutine does not modify `connectedCount`. But `dialLoop` MUST wait
for the receive goroutine to exit before looping to reconnect. This is a
**per-reconnect-iteration join requirement (F-SP1-005):**

> **Q6 per-reconnect join (binding):** Before `dialLoop` begins dialing a new
> connection for the same address (step 1 of a reconnect iteration), it MUST
> join — that is, block until completion of — the receive goroutine from the
> previous iteration. A `sync.WaitGroup` (Add(1) before step 4, Done() in the
> receive goroutine's deferred return) or a per-connection `done chan struct{}`
> (closed by the receive goroutine on exit) satisfies this requirement. The
> join MUST occur at the end of each dial iteration, before the reconnect
> backoff sleep and before the next dial attempt. Failure to join creates a
> goroutine-leak vector: a "flapping" upstream (rapid connect/disconnect) can
> accumulate O(N) receive goroutines reading from closed or recycled connections.

The AC-005 race test (covering `Connector.Stop()` teardown) MUST also cover a
**flap cycle** — that is, at least one complete connect-then-disconnect-then-reconnect
cycle — not only final teardown. A test that only exercises `Stop()` after one
successful connection does not exercise the per-iteration join path.

**`Stop()` teardown correctness:** `Connector.Stop()` calls `stopOnce.Do(close(c.stopCh))`
then `<-c.doneCh`. `c.doneCh` is closed by `reconcileLoop` after all
`addrCancel.done` channels are drained. For this to cover receive goroutines,
each `addrCancel.done` channel must not be closed until the receive goroutine for
that address has exited. The implementer must ensure the per-address `done chan struct{}`
is closed only after both `maintainConn` AND the receive goroutine have returned.
The per-iteration join (above) is a prerequisite for the teardown join to be
sound: without it, the `doneCh` close can race against a goroutine from a prior
iteration that was never joined.

**Cite:** `internal/upstreamdial/connector.go` `dialLoop`, `reconcile`,
`addrCancel` (verified at `8eb54a5`); F-P29-001 concurrent-drop race ruling
(S-7.04-FU-PE-CONNECTOR adversary pass-29, noted in DELIVERY at `8eb54a5`).

---

## Q7 — BC-2.06.003 PC-1 Failed-state observable: emission point and integration assertion key

**Ruling: BC-2.06.003 PC-1 "failed" status is emitted by
`internal/metrics.PathEntryFromSnapshot` in `internal/metrics/handlers.go` when
`PathSnapshot.Failed == true`. The `FrameArrivalHandler.OnFrameArrival` path that
emits E-FWD-001 does NOT directly emit BC-2.06.003 PC-1 — the two observables are
orthogonal. This story's BC-2.06.003 obligation is to demonstrate that the
send+forward path traversal (arqsend → PE receive loop → OnFrameArrival) is live
and exercised; the "failed" status emission is a downstream metrics concern gated
on path liveness failures, not on split-horizon drops.**

**Derivation:**

BC-2.06.003 PC-1 (verified at `8eb54a5`) defines `status: "failed"` as deriving
from `PathSnapshot.Failed == true`, which is set only when a previously-alive path
stops responding (`!firstProbe` liveness check in the `paths` package). The
`metrics.PathEntryFromSnapshot` function (verified at `8eb54a5` in
`internal/metrics/handlers.go`) implements:

```go
case snap.Failed:
    status = "failed"
```

This path is triggered by keepalive liveness failures — a path that was active
and went silent. It is NOT triggered by split-horizon drops (E-FWD-001). The two
are independent:

- E-FWD-001 fires because a frame's only forwarding option is the arrival interface.
  This is a topology condition, not a liveness failure.
- `status: "failed"` fires because keepalive probes stop receiving replies.
  This is a liveness condition.

**Consequence for this story's BC-2.06.003 discharge:**

The stub story's BC-2.06.003 PC-1 trace ("Failed-state via retransmit-driven path
exhaustion") conflates two independent mechanisms. The story-writer must clarify at
elaboration time which mechanism is being asserted:

- **Option A (E-FWD-001 path):** Assert that `"E-FWD-001"` fires under sustained
  retransmit load when the routing table has only the arrival interface. This is
  the AC-004 postcondition 1 discharge (the primary obligation re-anchored here).
  It does NOT require `status: "failed"` from `metrics`.

- **Option B (path-failed status path):** Assert that after a PE upstream
  connection drops, `sbctl paths list` returns `status: "failed"`. This is
  a follow-on behavioral property owned by `S-BL.PATH-FAILED-STATUS` infrastructure
  (already shipped at `8eb54a5`) and does not require the receive loop per se.

**Ruling: Option A is the operative discharge for this story.** The
BC-2.06.003 PC-1 trace in the stub is inherited from the original AC-004
framing and does not add a separate `status: "failed"` integration assertion
obligation in this story beyond what E-FWD-001 already covers.

**BC ambiguity flag (do not resolve unilaterally):** The stub story's
`bc_traces` section lists `BC-2.06.003` with the description "PC-1 Failed-state
via retransmit-driven path exhaustion." Reading BC-2.06.003 PC-1 literally, the
"failed" status field is about path liveness (`PathSnapshot.Failed`), not about
split-horizon drops. The PO should confirm whether (a) BC-2.06.003 PC-1 is
traced here to document that the full send+forward path enables future
`status: "failed"` path liveness testing, or (b) there is a spec-level linkage
between E-FWD-001 exhaustion and `status: "failed"` that I did not find. This
ambiguity does not block implementation of Q5 but must be resolved before AC
finalization.

**Integration assertion key for BC-2.06.003 (if Option A):** assert that the
writer output contains `"E-FWD-001"` (same key as Q5). No `"failed"` string
assertion is required by this story under Option A.

**Cite:** `internal/metrics/handlers.go` `PathEntryFromSnapshot` (verified at
`8eb54a5`); `internal/paths/paths.go` `PathSnapshot.Failed` field (verified at
`8eb54a5`); `internal/routing/on_frame_arrival.go` E-FWD-001 emission (verified
at `8eb54a5`); BC-2.06.003 v1.16 PC-1.

---

## Q8 — Production wiring: making E-FWD-001 reachable via the PE receive callback (F-SP1-001)

**Context (finding F-SP1-001):** The v1.0 note's Q1/Q2 rulings wired the PE
receive callback to `routing.RouteFrame`. The adversarial pass established that
this wiring cannot emit E-FWD-001: `RouteFrame` delegates to `SVTNRoute` (verified
at `8eb54a5` in `internal/routing/routing.go` — `RouteFrame` returns `SVTNRoute(hdr, payload, r)`);
`SVTNRoute` performs admission + forwarding-table lookup and returns `ErrNoForwardingEntry`
on miss, but NEVER calls `FrameArrivalHandler.OnFrameArrival` (verified: zero
production callers of `OnFrameArrival` or `NewFrameArrivalHandler` exist in `cmd/`
at `8eb54a5` — grep confirmed). `ErrAllPathsSplitHorizon` (the source of E-FWD-001)
is emitted exclusively by `SplitHorizon.Forward`, which is called only from
`OnFrameArrival` (verified at `8eb54a5` in `internal/routing/on_frame_arrival.go`).
Additionally, `runRouter` at `8eb54a5` constructs the `router` via
`buildRouter(admission.NewAdmittedKeySet(), routerLogger)` with an empty forwarding
table — AC-004's arqsend frames would die at admission (`ErrNotAdmitted`) before
reaching any forwarding decision.

**Ruling: the PE receive `FrameFn` callback in `runRouter` MUST route through a
properly-constructed `routing.FrameArrivalHandler` rather than calling
`routing.RouteFrame` directly. This is wiring option (a): `runRouter` constructs
a `*routing.FrameArrivalHandler` at startup (after Phase b), passes a closure
wrapping `handler.OnFrameArrival(...)` as the `FrameFn` to `connector.SetFrameCallback`,
and does NOT change the `netingress.Serve` path (which retains its existing
`routing.RouteFrame` closure).**

**Q8 wiring specification:**

### 8.1 — FrameArrivalHandler construction

`runRouter` constructs a `*routing.FrameArrivalHandler` immediately after the
router is built (after Phase b, before Phase c). Construction requires a
`*multipath.DropCache`:

```go
// After router := buildRouter(...):
dc := multipath.NewDropCache(multipath.DefaultDropCacheSize)
arrivalHandler := routing.NewFrameArrivalHandler(dc)
routing.WithFrameArrivalLogger(routerLogger)(arrivalHandler)
```

Verified at `8eb54a5`:
- `multipath.NewDropCache(capacity int) *DropCache` — `internal/multipath/multipath.go`
- `multipath.DefaultDropCacheSize = 10_000` — `internal/multipath/multipath.go`
- `routing.NewFrameArrivalHandler(dc *multipath.DropCache) *FrameArrivalHandler` — `internal/routing/on_frame_arrival.go`
- `routing.WithFrameArrivalLogger(l Logger) func(*FrameArrivalHandler)` — `internal/routing/on_frame_arrival.go`

### 8.2 — FrameFn closure wired to connector

The `FrameFn` callback set on the connector routes through `OnFrameArrival`. The
full signature of `OnFrameArrival` is (verified at `8eb54a5`):

```go
func (h *FrameArrivalHandler) OnFrameArrival(
    frameBytes []byte,
    arrivalIface InterfaceID,
    interfaceSet []InterfaceID,
    fn ForwardFunc,
) error
```

The `FrameFn` closure must supply:
- `arrivalIface` — the PE connection's logical `routing.InterfaceID`. A fixed
  constant (e.g. `routing.InterfaceID(1)` or a named PE-interface ID) is
  acceptable for this story; the value uniquely identifies the PE upstream path.
  The implementer assigns this at construction time; the exact value is an
  elaboration decision.
- `interfaceSet` — the set of forwarding candidates. For the E-FWD-001 exhaustion
  test, the interface set MUST be `[]routing.InterfaceID{peIfaceID}` only (the
  arrival interface is the sole candidate), which guarantees `SplitHorizon.Forward`
  returns `ErrAllPathsSplitHorizon`. In production, the interface set is populated
  from the router's forwarding table or a registry of connected data-plane nodes;
  see elaboration note below.
- `fn ForwardFunc` — the forward function that actually sends bytes to an interface.
  In production this dials the destination. In the integration test a no-op or
  capture function is acceptable (the E-FWD-001 path never calls `fn` because all
  paths are split-horizon blocked).

**Skeleton (illustrative):**

```go
peIfaceID := routing.InterfaceID(1) // PE upstream arrival interface
frameFn := upstreamdial.FrameFn(func(hdr frame.OuterHeader, raw []byte) error {
    // interfaceSet: for test = [peIfaceID] only (exhaustion); in production,
    // consult the forwarding table or registered interface registry.
    return arrivalHandler.OnFrameArrival(
        raw,
        peIfaceID,
        []routing.InterfaceID{peIfaceID}, // single-interface = guaranteed exhaustion in test
        func(iface routing.InterfaceID, frameBytes []byte) error {
            // production: forward to iface's connection; test: capture or discard
            return nil
        },
    )
})
connector.SetFrameCallback(frameFn)
```

### 8.3 — Import graph impact

`cmd/switchboard/mgmt_wire.go` already imports `internal/routing` (verified at
`8eb54a5`) and `internal/netingress`. Adding `internal/multipath` is a new import
at the `cmd/switchboard` layer. `cmd/switchboard` is at the top of the DAG (no
position constraint applies to the binary); this import is unconditionally legal.
No ARCH-08 §6.4 registration is required for `cmd/switchboard` imports.

Verified at `8eb54a5`: `cmd/switchboard/mgmt_wire.go` imports list includes
`internal/routing` and does NOT include `internal/multipath` — adding it is the
only import change in `cmd/switchboard`.

### 8.4 — `netingress.Serve` path unaffected

The `netingress.Serve` data-plane accept loop in `runRouter` retains its existing
wiring `routing.RouteFrame(hdr, payload, router)` unchanged (verified at
`8eb54a5`). That path is for frames arriving from connected access nodes on the
data-plane TCP listener. The `FrameArrivalHandler` path is strictly the PE
upstream receive goroutine. The two paths are disjoint and do not share router
state beyond the `*routing.Router` itself (which is safe for concurrent use via
its internal `sync.RWMutex`).

### 8.5 — Forwarding table + admission state for the integration test

AC-004's arqsend retransmit frames must survive admission and reach the forward
decision. The `runRouter` construction at `8eb54a5` uses
`admission.NewAdmittedKeySet()` (empty set — fail-closed). Frames from the test
would die at `ErrNotAdmitted` before `OnFrameArrival` is even invoked.

**The integration test MUST:**

1. Call `router.RegisterForwardingEntry(svtnID, nodeAddr, authKey)` with an entry
   that matches the test frame's `hdr.SVTNID` and `hdr.DstAddr` — so `SVTNRoute`
   does not return `ErrNoForwardingEntry` before the frame even reaches the
   handler. (Note: with the `FrameArrivalHandler` wiring, the frame goes directly
   to `arrivalHandler.OnFrameArrival`; `SVTNRoute` is NOT called on this path.
   `OnFrameArrival` does not consult the forwarding table — it operates on raw
   bytes and the drop cache only. The forwarding table constraint above applies to
   the `netingress` path, not the PE receive path.)
2. Ensure the outerassembler `Envelope` used to construct arqsend frames carries
   an `FrameAuthKey` matching a key the test supplies. Because the PE receive
   `FrameFn` goes directly to `OnFrameArrival` (bypassing `RouteFrame`'s HMAC
   check), HMAC admission is NOT checked on the PE receive path in this design.
   The test frames therefore do not need a valid HMAC. The story-writer must
   confirm this is acceptable for the E-FWD-001 exhaustion test or elect to add
   an explicit HMAC-verify step in the `FrameFn` closure.
3. Set `interfaceSet = []routing.InterfaceID{peIfaceID}` in the `FrameFn` closure
   to guarantee all paths are split-horizon blocked and E-FWD-001 fires on every
   non-bootstrap frame.

### 8.6 — Blast radius on existing RouteFrame callers

`routing.RouteFrame` callers at `8eb54a5` (grep-verified):
- `cmd/switchboard/mgmt_wire.go` `runRouter` Phase f ingress closure: UNAFFECTED — this path stays as-is.
- `internal/netingress/integration_test.go`: UNAFFECTED — test code, not modified by this story.
- `internal/arqsend/integration_test.go`: UNAFFECTED — test code using `RouteFrame` directly.
- `internal/admission/failure_counter_adversarial_test.go` `TestRouteFrame_EndToEnd_EADMAlertMessageFormat`: UNAFFECTED — test code.
- `internal/routing/example_test.go`: UNAFFECTED — example tests.
- `internal/routing/routing_internal_test.go`: UNAFFECTED — internal tests.

No production caller of `routing.RouteFrame` is modified. The `netingress` path
is explicitly preserved. `RouteFrame`'s signature and semantics are unchanged.

**Cite:** `internal/routing/routing.go` `RouteFrame`, `SVTNRoute` (verified at `8eb54a5` — `RouteFrame` returns `SVTNRoute(...)`, no `OnFrameArrival` call);
`internal/routing/on_frame_arrival.go` `OnFrameArrival`, `NewFrameArrivalHandler`, `WithFrameArrivalLogger` (verified at `8eb54a5`);
`internal/multipath/multipath.go` `NewDropCache`, `DefaultDropCacheSize` (verified at `8eb54a5`);
`internal/routing/split_horizon.go` `SplitHorizon.Forward`, `ErrAllPathsSplitHorizon` (verified at `8eb54a5`);
`cmd/switchboard/mgmt_wire.go` `runRouter` import list (verified at `8eb54a5` — `routing` present, `multipath` absent).

---

## Q9 — E-FWD-001 injection topology: upstream fixture write path and harness rule (F-SP2-001, F-SP2-002, F-SP2-003)

**Context:** Pass-2 adversarial review established three interlocking defects in the Q4/Q5
injection model:

- **F-SP2-001 (CRITICAL [spec-defect]):** Q8 wires production emission onto the PE receive
  path (`FrameFn → OnFrameArrival` on frames arriving over the DIALED upstream conn). But
  Q4/Q5's AC-004 test-injection vector has `arqsend.Dispatch` dial `cfg.ListenAddr` (the
  data-plane TCP listener) and write wire bytes there. Those bytes enter via
  `netingress.Serve → RouteFrame` — a physically disjoint socket from the dialed PE conn.
  `RouteFrame` does NOT call `OnFrameArrival` (verified at `8eb54a5`; zero production
  callers of `OnFrameArrival` from the netingress path). AC-004 as specified in v1.1 is
  undischargeable: the frame never reaches the FrameFn callback.

- **F-SP2-002 (HIGH [spec-gap]):** No write-capable upstream fixture exists. Both
  `startPEListenerFixture` (in `cmd/switchboard/router_pe_connector_test.go`, verified at
  `8eb54a5` — accept loop reads-and-drains, zero `Write` calls) and testenv's `peLn`
  (in `internal/testenv/testenv.go`, verified at `8eb54a5` — "Drain and close: we just
  need the connection to be accepted", zero `Write` calls) are read-only drains.

- **F-SP2-003 (MED [spec-defect]):** AC-004's precondition starts the router via
  `testenv.New`/`Restart`. `testenv.Restart` builds a bare `upstreamdial.New` and
  NEVER calls `SetFrameCallback` (verified at `8eb54a5` in `testenv.go` `Restart` —
  it calls `upstreamdial.New(...).Start()` with no callback wiring). `runRouter`
  (the real production function in `mgmt_wire.go`) is where `SetFrameCallback` will
  be called per the Q8 ruling; `testenv.Restart` bypasses this entirely. Any AC
  asserting that `OnFrameArrival` is reached therefore CANNOT use `testenv.Restart`;
  it must use the real `runRouter` goroutine pattern.

### 9.1 — Injection topology ruling (supersedes Q4 arqsend Dispatch and Q5 test shape)

**Ruling: option (b) — the upstream fixture assembles and writes one assembled outer frame
directly to the accepted PE connection. `arqsend.Retransmitter` is NOT used as the frame
producer in the AC-004 exhaustion integration test.**

The correct injection topology for AC-004:

```
PE upstream fixture (accepted conn)
    ──WRITES assembled outer frame──►  dialed PE conn in runRouter
                                           │
                                       PE receive goroutine in upstreamdial.Connector
                                           │ (via frame.ReadOuterFrame)
                                       FrameFn callback in runRouter
                                           │ (arrivalHandler.OnFrameArrival)
                                       SplitHorizon.Forward → ErrAllPathsSplitHorizon
                                           │
                                       E-FWD-001 logged in writer output  ◄── assert here
```

The frame the fixture writes is a valid `outerassembler.Assemble` output — the same
wire format that `frame.ReadOuterFrame` (new, defined by this story) expects. The fixture
uses `outerassembler.Assemble(cf, sackBitmap, env)` with a non-bootstrap `FrameType`
(e.g. `frame.FrameTypeData`) so it passes the `FrameTypePEConnect` discard check in the
receive goroutine and reaches `OnFrameArrival`. A zero `outerassembler.Envelope` is
sufficient; HMAC is not checked on the PE receive path (Q8 §8.5 ruling confirmed clean
per adjudicated-clean section below).

### 9.2 — Write-capable upstream fixture specification (F-SP2-002)

**Fixture placement: test-local, same file as other `runRouter` integration tests —
`cmd/switchboard/router_pe_receive_test.go` (NEW, per FCL row 7).**

A testenv seam is NOT required and would incur ARCH-08 position-23 implications
(testenv imports `outerassembler` at position 8; that edge is already present per
ARCH-08 §6.5 v2.8, so adding a `WriteFrame` helper would not add a new import edge —
but it would couple testenv's API surface to PE-frame-injection concerns that are local
to this story's test file). The lightest lawful option is a test-local fixture struct,
consistent with the pattern already used in `router_pe_connector_test.go`
(`startPEListenerFixture`).

**Fixture shape:**

```go
// peWriteFixture is a test-local upstream fixture that accepts one connection
// and exposes WriteFrame so the test can inject assembled outer frames into the
// PE receive goroutine.
type peWriteFixture struct {
    addr     string
    accepted chan net.Conn // buffered(1); receives the accepted conn
    ln       net.Listener
}

func startPEWriteFixture(t *testing.T) *peWriteFixture {
    t.Helper()
    ln, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil {
        t.Fatalf("startPEWriteFixture: Listen: %v", err)
    }
    t.Cleanup(func() { _ = ln.Close() })
    f := &peWriteFixture{addr: ln.Addr().String(), accepted: make(chan net.Conn, 1), ln: ln}
    go func() {
        conn, err := ln.Accept()
        if err != nil {
            return
        }
        // Drain incoming bytes (connector writes bootstrap + keepalives).
        go func(c net.Conn) {
            buf := make([]byte, 4096)
            for {
                if _, err := c.Read(buf); err != nil {
                    return
                }
            }
        }(conn)
        f.accepted <- conn
    }()
    return f
}

// WriteFrame writes a pre-assembled wire frame to the accepted conn.
// Blocks until the connection is accepted (or t fails).
func (f *peWriteFixture) WriteFrame(t *testing.T, wire []byte) {
    t.Helper()
    var conn net.Conn
    select {
    case conn = <-f.accepted:
        f.accepted <- conn // put back for subsequent calls
    case <-time.After(3 * time.Second):
        t.Fatal("peWriteFixture.WriteFrame: timed out waiting for accepted conn")
    }
    if _, err := conn.Write(wire); err != nil {
        t.Fatalf("peWriteFixture.WriteFrame: Write: %v", err)
    }
}
```

**Frame assembly in the test:**

```go
wire, err := outerassembler.Assemble(
    halfchannel.ChannelFrame{
        FrameType: frame.FrameTypeData,   // non-bootstrap → reaches OnFrameArrival
        ChanID:    1,
        ChanSeq:   1,
        Payload:   []byte{0x01},
    },
    [outerassembler.SACKBitmapSize]byte{},
    outerassembler.Envelope{},            // zero env — HMAC bypass per Q8 §8.5
)
// outerassembler.Assemble, outerassembler.SACKBitmapSize verified at 8eb54a5
// halfchannel.ChannelFrame, frame.FrameTypeData verified at 8eb54a5
if err != nil { t.Fatalf("Assemble: %v", err) }
fixture.WriteFrame(t, wire)
```

### 9.3 — Harness rule (F-SP2-003): runRouter goroutine pattern is mandatory for OnFrameArrival ACs

**Binding harness rule:** Every AC that asserts `OnFrameArrival` is reached — specifically
AC-001, AC-002, and AC-004 — MUST use the real `runRouter` goroutine pattern, not
`testenv.Restart`. The real pattern:

```go
buf := &syncBuffer{}
ctx, cancel := context.WithCancel(context.Background())
errCh := make(chan error, 1)
go func() {
    errCh <- runRouter(ctx, buf, cfg, cfgPath, nil)
}()
t.Cleanup(func() {
    cancel()
    select { case <-errCh: case <-time.After(3 * time.Second): }
})
```

`runRouter` is the code path that constructs the `FrameArrivalHandler` and calls
`connector.SetFrameCallback(frameFn)` per the Q8 ruling. `testenv.Restart` builds a bare
`upstreamdial.New` without calling `SetFrameCallback` — verified at `8eb54a5` in
`testenv.go` `Restart` implementation. This means any test using `testenv.Restart` will
have an unregistered `FrameFn` (nil); `OnFrameArrival` is never called; E-FWD-001 never
fires. Such a test would pass trivially for the wrong reason.

**Rationale for no testenv seam:** Adding a `SetFrameCallback` seam to `testenv` would
require testenv to import `routing` (or accept a `routing.FrameArrivalHandler`) — a
position-23 package importing position-17, which is lawful (23 > 17), but imports
`routing` into the test composition root unnecessarily. The real `runRouter` goroutine
pattern is already established in `router_pe_connector_test.go` (AC-001 through AC-004
in `TestRunRouter_PE_DialAndConnect_UpstreamReachable` et al., all verified at `8eb54a5`);
the new `router_pe_receive_test.go` file MUST follow the same pattern. No testenv API
change is required or permitted for this story.

### 9.4 — arqsend obligation audit and disposition (Q4 supersession accounting)

Option (b) rules `arqsend.Retransmitter` out of the E-FWD-001 integration test. The
Q4 ruling that arqsend is "test-internal only, not wired into production `runRouter`"
remains correct. What changes is the test's use of arqsend:

- **Q4's arqsend production-wiring ruling** (arqsend NOT in `runRouter`) — **RETAINED.**
  The production `runRouter` does not need a persistent `Retransmitter` instance; the
  production ARQ retransmit path is node-side. This ruling is unaffected.

- **Q4's test-internal arqsend construction** (the `Dispatch → net.Dial(ListenAddr)`
  shape) — **SUPERSEDED by Q9.** The `Dispatch` closure that dials `ListenAddr` is
  the injection path that F-SP2-001 identifies as physically disjoint from the PE
  receive goroutine. The entire arqsend frame-production role in AC-004 is replaced
  by the `peWriteFixture.WriteFrame` path.

- **S404-OBS-F "sustained send+forward" re-confirmation framing** — Q9 rules this
  is discharged through the `peWriteFixture` injection path. The "send" is
  `peWriteFixture.WriteFrame`; the "forward" attempt is `OnFrameArrival` routing
  through the split-horizon path. The S404-OBS-F obligation does NOT require
  `arqsend.Retransmitter` specifically; it requires a live frame traversing the full
  send+forward path. The `peWriteFixture` path satisfies this obligation.

- **S404-LOW-1 "live-egress re-confirmation"** — same disposition as S404-OBS-F.
  Both drift anchors are discharged by AC-004 using the `peWriteFixture` injection
  path.

**arqsend in the FCL:** `internal/arqsend` is removed from the `architecture_modules`
list of files touched by this story. The existing `arqsend` integration test
(`internal/arqsend/integration_test.go`) remains unmodified; it tests arqsend's own
`RouteFrame`-dispatch path and is unaffected by this story's injection topology change.

### 9.5 — Story propagation obligations (binding for story-writer)

The story-writer MUST propagate the following Q9 changes to `S-BL.PE-RECEIVE-LOOP.md`:

1. **Q4 dispatch closure** — remove the `net.Dial(cfg.ListenAddr)` dispatch shape from
   AC-004; replace with `peWriteFixture.WriteFrame` injection path.
2. **Q5 test infrastructure** — replace "dispatch writes to router's data-plane
   `ListenAddr`" with "upstream fixture writes assembled frame to the accepted PE
   connection via `peWriteFixture.WriteFrame`".
3. **AC-004 precondition** — the test uses the `runRouter` goroutine pattern with a
   `peWriteFixture` as the upstream. The `peWriteFixture` replaces both the precondition
   note about `arqsend.Retransmitter` construction and the `dispatch` closure body.
4. **FCL** — add `peWriteFixture` type definition to FCL row 7 (NEW file
   `cmd/switchboard/router_pe_receive_test.go`); remove `internal/arqsend` from the
   architecture_modules header.
5. **Q3 blast-radius** — add items 6 and 7 from the blast-radius table in Q3 above.

**Cite:** Pass-2 adversarial report (F-SP2-001 CRITICAL, F-SP2-002 HIGH, F-SP2-003 MED,
F-SP2-004 MED); `cmd/switchboard/router_pe_connector_test.go` `startPEListenerFixture`
(accept-and-drain, zero Write calls, verified at `8eb54a5`); `internal/testenv/testenv.go`
`peLn` goroutine (accept-and-drain, zero Write calls, verified at `8eb54a5`);
`internal/testenv/testenv.go` `Restart` (bare `upstreamdial.New` without
`SetFrameCallback`, verified at `8eb54a5`); `outerassembler.Assemble` (verified at
`8eb54a5`); `halfchannel.ChannelFrame` (verified at `8eb54a5`).

---

## Pass-2 Adjudicated-Clean (non-findings, per adversarial pass-2 report)

The following five items were raised by the pass-2 adversary but adjudicated clean.
They are recorded here per the "adjudicated-clean: cite pass-2 report, do not re-derive"
instruction.

| Item | Adversary concern | Ruling |
|------|-------------------|--------|
| `fn ForwardFunc` no-op consistent | `SplitHorizon` may not call `fn` if no eligible path — is this a vacuous test? | Clean. `SplitHorizon.Forward` returns `ErrAllPathsSplitHorizon` BEFORE calling `fn` on the empty-eligible path (verified at `8eb54a5` in `internal/routing/split_horizon.go`). E-FWD-001 fires on the return path regardless; the no-op `fn` is never invoked. The test is not vacuous. |
| Duplicate-frame drop-cache semantics | `DropCache` may suppress the second frame if two identical frames are injected, preventing E-FWD-001 from firing twice | Clean. `arqsend.Retransmit` creates a fresh `ChanSeq` per `Retransmit` call (verified at `8eb54a5`). With the Q9 ruling replacing arqsend with `peWriteFixture`, a single injected frame is sufficient — the test asserts `"E-FWD-001"` fires once. `DropCache` has no effect on the first unique frame (fresh checksum). |
| HMAC bypass vs BC-2.02.008 preconditions | PE receive path bypasses `RouteFrame` HMAC check — does BC-2.02.008 assume admission is enforced before `OnFrameArrival`? | Clean. BC-2.02.008 carries no admission assumption (verified at `8eb54a5` in `.factory/specs/`); it postconditions on the split-horizon event itself. `OnFrameArrival` treats `frameBytes` as opaque — no HMAC field in `on_frame_arrival.go`. The bypass is acceptable for this story; a SEC follow-on revisit is noted in Q8 §8.5. |
| `peIfaceID = InterfaceID(1)` collision | Could `InterfaceID(1)` collide with a data-plane interface ID already registered by `netingress`? | Clean. The data-plane listener in `runRouter` uses `netingress.Serve`, which does NOT register `InterfaceID` values with the router (verified at `8eb54a5`); `routing.InterfaceID` values are assigned by the `FrameFn` closure, not by `netingress`. No pre-existing `InterfaceID(1)` registration exists at construction time. The PE iface ID is assigned exclusively by the wiring in Q8. |
| `routerLogger` satisfies `routing.Logger` | Does `routerLogger` (constructed in `runRouter`) implement `routing.Logger` without a shim? | Clean. `routerLogger` is constructed via `newStdLogger(w)` (verified at `8eb54a5` in `mgmt_wire.go`); `routing.Logger` is the single-method `Log(string)` interface (verified at `8eb54a5` in `internal/routing/`); `newStdLogger` produces a concrete type that satisfies `Log(string)` (verified at `8eb54a5`). No shim is needed. |

---

## Scope Boundary vs S-7.04-FU-DRAIN-WIRE

| This story (PE-RECEIVE-LOOP) | S-7.04-FU-DRAIN-WIRE |
|---|---|
| Receive goroutine per PE connection; routes incoming frames to `FrameFn` callback | Broadcasts DRAIN signal to connected nodes via SVTN channel |
| Defines `frame.FrameTypePEConnect`; flips `dialLoop` bootstrap from placeholder | Registers observers on `drainCoord` to send DRAIN frames to nodes |
| E-FWD-001 exhaustion discharge (AC-004 postcondition 1) | VP-037 `verification_lock` flip (blocked on DRAIN broadcast) |

This story provides the receive loop that makes DRAIN broadcast meaningful — a DRAIN frame sent over a PE connection must be received and forwarded. `S-7.04-FU-DRAIN-WIRE` cannot be scheduled before this story merges.

## Scope Boundary vs S-BL.RESYNC-FRAME

`S-BL.RESYNC-FRAME` owns the RESYNC control-frame exchange initiated after a node migrates to a new upstream router. This story's receive loop handles the raw frame arrival and routing; it does not implement RESYNC semantics. A RESYNC frame arriving on a PE connection will be passed to the `FrameFn` callback as a normal `FrameTypeCtl` frame (if the RESYNC frame type is `FrameTypeCtl`); further dispatch is the RESYNC story's concern.

---

## Files-Changed Forecast (Candidate FCL)

| File | Change |
|------|--------|
| `internal/frame/frame.go` (MODIFIED) | Add `FrameTypePEConnect FrameType = 0x06` (ARCH-02 §3.1 citation); update `Valid()` upper bound to `<= FrameTypePEConnect`; update doc comments: FrameType type ("Only five" → "Only six"), Valid() ("five canonical…0x06..0xFF" → "six canonical…0x07..0xFF"), ErrInvalidFrameType ("five canonical" → "six canonical or not in {0x01..0x06}") |
| `internal/frame/frame_test.go` (MODIFIED) | Add `TestFrameType_Valid_PEConnect` asserting `FrameTypePEConnect.Valid() == true`; change `just_above_max` case from `FrameType(0x06)` to `FrameType(0x07)`; change `invalids` slice `0x06` entry to `0x07`; update `"five canonical enum values"` description comment and `"Bytes not in {0x01..0x05}"` comment |
| `internal/upstreamdial/connector.go` (MODIFIED) | Add `FrameFn` type + `SetFrameCallback(fn FrameFn)` to `Handle` interface; add `frameFn` field to `Connector`; add receive goroutine in `dialLoop` after step-3 success with per-connection `WaitGroup` join before reconnect; flip bootstrap `ChannelFrame.FrameType` from `halfchannel.FrameTypeData` to `frame.FrameTypePEConnect` (FO-PE-LOOP-001); add direct `internal/frame` import |
| `internal/upstreamdial/connector_test.go` (MODIFIED) | Unit tests: receive goroutine exits on conn close, `FrameTypePEConnect` bootstrap frame is discarded, data frames passed to `FrameFn`; flap-cycle test exercising per-reconnect join |
| `cmd/switchboard/mgmt_wire.go` (MODIFIED) | Construct `multipath.NewDropCache` + `routing.NewFrameArrivalHandler` after Phase b; wire `SetFrameCallback` on the connector with `FrameFn` closure routing through `arrivalHandler.OnFrameArrival`; add `internal/multipath` import |
| `cmd/switchboard/router_pe_receive_test.go` (NEW) | Integration tests: AC-001 (receive loop active after PE connection), AC-002 (E-FWD-001 fires under path exhaustion via `OnFrameArrival` with single-interface set), AC-003 (bootstrap frame discarded, not forwarded) |
| `.factory/specs/architecture/ARCH-02-protocol-stack.md` (MODIFIED) | §"Outer Header Format" `frame_type` table row: add `pe_connect=0x06` |
| `.factory/specs/architecture/ARCH-08-dependency-graph.md` (MODIFIED) | §6.5 update: `internal/upstreamdial` allowed imports `{halfchannel, outerassembler}` → `{frame, halfchannel, outerassembler}` |

**Estimated AC count:** 3–5 ACs. See §"Estimated AC count for story-writer" below.

---

## Summary of Rulings (Q1–Q9)

| Q | Ruling (one-line) |
|---|---|
| Q1 | Receive goroutine lives in `upstreamdial.Connector` (per-connection, spawned after step-3 success); `Handle` gains `SetFrameCallback(fn FrameFn)` seam; `upstreamdial` stays routing-free. (v1.0 import/signature details superseded by Q2 — see v1.1 supersession annotation.) |
| Q2 | Framing via new `frame.ReadOuterFrame(io.Reader) (OuterHeader, []byte, error)` at position 2; `upstreamdial` gains direct `frame` import (ARCH-08 §6.5 amendment required); callback signature `type FrameFn func(hdr frame.OuterHeader, raw []byte) error`. |
| Q3 | Define `frame.FrameTypePEConnect = 0x06`; update `Valid()` upper bound to `<= FrameTypePEConnect`; full blast radius: amend `just_above_max` test (0x06→0x07), invalids slice (0x06→0x07), five doc-comment occurrences in `frame.go`/`frame_test.go`, and ARCH-02 §"Outer Header Format" `frame_type` table row. |
| Q4 | `arqsend.New` is NOT wired into production `runRouter` (retained). Arqsend's test-internal `Dispatch → net.Dial(ListenAddr)` injection shape is **superseded by Q9** — that shape dispatches to the data-plane socket (netingress path), not the PE receive goroutine. arqsend is NOT the frame producer in AC-004. |
| Q5 | E-FWD-001 test uses the real `runRouter` goroutine pattern (not `testenv.Restart` — F-SP2-003 harness rule); upstream fixture is `peWriteFixture` which writes assembled outer frames to the accepted PE connection; asserts key `"E-FWD-001"` in writer output (F-P11-001 lesson retained). Injection topology fully specified by Q9. |
| Q6 | Receive goroutine exits naturally when `conn.Close()` called by `dialLoop` teardown; per-connection `WaitGroup`/`done chan struct{}` MUST be joined at end of each dial iteration before reconnect (F-SP1-005 per-reconnect join requirement); AC-005 race test MUST cover a flap cycle, not only `Stop()`. |
| Q7 | BC-2.06.003 PC-1 "failed" status is from `metrics.PathEntryFromSnapshot` (path liveness), NOT from E-FWD-001 (split-horizon); BC ambiguity flagged for PO confirmation; operative assertion is `"E-FWD-001"` key (Option A). |
| Q8 | PE receive `FrameFn` MUST route through `routing.NewFrameArrivalHandler`+`OnFrameArrival` (not `RouteFrame`) to make E-FWD-001 reachable; `runRouter` constructs `multipath.NewDropCache` + `routing.NewFrameArrivalHandler` after Phase b; `netingress.Serve` path unchanged; `cmd/switchboard` gains `internal/multipath` import. |
| Q9 | **Injection topology ruling** (supersedes Q4 dispatch + Q5 injection shape): option (b) — upstream PE fixture (`peWriteFixture`, test-local in `router_pe_receive_test.go`) writes assembled outer frame directly to the accepted PE connection; `arqsend.Retransmitter` is NOT used as frame producer in AC-004; harness rule: every AC asserting `OnFrameArrival` MUST use the real `runRouter` goroutine pattern (not `testenv.Restart`); S404-OBS-F and S404-LOW-1 discharged via `peWriteFixture` injection path; Q4 production-wiring ruling (arqsend not in `runRouter`) retained. |

---

## Estimated AC Count for Story-Writer

Based on the rulings above and the 5-point estimated scope in the stub story:

| AC | Trace | Description |
|----|-------|-------------|
| AC-001 | BC-2.09.001 PC-2/PC-3 | Receive loop is active after PE connection established; incoming frame from upstream is passed to `FrameArrivalHandler.OnFrameArrival` callback |
| AC-002 | BC-2.02.008 PC-3/EC-003, S404-OBS-F | E-FWD-001 fires under sustained path-exhaustion load via live PE connection + arqsend retransmit |
| AC-003 | FO-PE-LOOP-001 | Bootstrap frame with `FrameTypePEConnect` is discarded at receiver; NOT forwarded through routing callback |
| AC-004 | BC-2.06.003 PC-1 (Option A) | Live send+forward path traversal is exercised; BC-2.06.003 trace confirmed or clarified by PO per Q7 flag |
| AC-005 | S404-LOW-1 | Live egress re-confirmation: full send→forward path is demonstrated end-to-end |

Estimated: **5 ACs** (may collapse AC-004/AC-005 into AC-002 at elaboration).

---

## Appendix A: Backtick-Symbol Sweep

All `CamelCase/pkg.Symbol` tokens used in this note. Sweep performed against the
tree at `8eb54a5` using `grep` on the verified file paths.

| Symbol | File verified | Status |
|--------|--------------|--------|
| `frame.FrameTypeData` | `internal/frame/frame.go` | VERIFIED — `FrameTypeData FrameType = 0x01` |
| `frame.FrameTypeEmptyTick` | `internal/frame/frame.go` | VERIFIED — `FrameTypeEmptyTick FrameType = 0x02` |
| `frame.FrameTypeCtl` | `internal/frame/frame.go` | VERIFIED — `FrameTypeCtl FrameType = 0x03` |
| `frame.FrameTypeArq` | `internal/frame/frame.go` | VERIFIED — `FrameTypeArq FrameType = 0x04` |
| `frame.FrameTypeFec` | `internal/frame/frame.go` | VERIFIED — `FrameTypeFec FrameType = 0x05` |
| `frame.FrameTypePEConnect` | N/A | NEW CONSTANT — value `0x06`; to be defined by this story in `internal/frame/frame.go` |
| `frame.FrameType.Valid()` | `internal/frame/frame.go` | VERIFIED — `func (f FrameType) Valid() bool { return f >= FrameTypeData && f <= FrameTypeFec }` |
| `frame.ParseOuterHeader` | `internal/frame/frame.go` | VERIFIED — `func ParseOuterHeader(b []byte) (OuterHeader, error)` |
| `frame.OuterHeader` | `internal/frame/frame.go` | VERIFIED — `type OuterHeader struct { ... FrameType FrameType; PayloadLen uint16; ... }` |
| `frame.OuterHeaderSize` | `internal/frame/frame.go` | VERIFIED — `const OuterHeaderSize = 44` |
| `frame.ReadOuterFrame` | N/A | NEW FUNCTION — to be added to `internal/frame/frame.go` by this story |
| `netingress.ReadFrame` | `internal/netingress/netingress.go` | VERIFIED — `func ReadFrame(r io.Reader) (frame.OuterHeader, []byte, error)` |
| `netingress.RouteFn` | `internal/netingress/netingress.go` | VERIFIED — `type RouteFn func(hdr frame.OuterHeader, payload []byte) error` |
| `netingress.ServeConn` | `internal/netingress/netingress.go` | VERIFIED — `func ServeConn(ctx context.Context, conn net.Conn, route RouteFn, logger Logger) error` |
| `netingress.Serve` | `internal/netingress/netingress.go` | VERIFIED — `func Serve(ctx context.Context, ln net.Listener, route RouteFn, logger Logger) error` |
| `halfchannel.FrameTypeData` | `internal/halfchannel/halfchannel.go` | VERIFIED — alias of `frame.FrameTypeData` |
| `halfchannel.FrameTypeEmptyTick` | `internal/halfchannel/halfchannel.go` | VERIFIED — alias of `frame.FrameTypeEmptyTick` |
| `halfchannel.ChannelFrame` | `internal/halfchannel/halfchannel.go` | VERIFIED — `type ChannelFrame struct { ChanID uint32; ChanSeq uint32; FrameType frame.FrameType; Flags byte; Payload []byte }` |
| `outerassembler.Assemble` | `internal/outerassembler/assemble.go` | VERIFIED — `func Assemble(cf halfchannel.ChannelFrame, sackBitmap [SACKBitmapSize]byte, env Envelope) ([]byte, error)` |
| `outerassembler.Envelope` | `internal/outerassembler/assemble.go` | VERIFIED — `type Envelope struct { SVTNID [16]byte; SrcAddr [8]byte; DstAddr [8]byte; FrameAuthKey [hmac.KeySize]byte }` |
| `outerassembler.SACKBitmapSize` | `internal/outerassembler/channelheader.go` | VERIFIED — `const SACKBitmapSize = 8` |
| `routing.FrameArrivalHandler` | `internal/routing/on_frame_arrival.go` | VERIFIED — `type FrameArrivalHandler struct { ... }` |
| `routing.NewFrameArrivalHandler` | `internal/routing/on_frame_arrival.go` | VERIFIED — `func NewFrameArrivalHandler(dc *multipath.DropCache) *FrameArrivalHandler` |
| `routing.WithFrameArrivalLogger` | `internal/routing/on_frame_arrival.go` | VERIFIED — `func WithFrameArrivalLogger(l Logger) func(*FrameArrivalHandler)` |
| `routing.FrameArrivalHandler.OnFrameArrival` | `internal/routing/on_frame_arrival.go` | VERIFIED — `func (h *FrameArrivalHandler) OnFrameArrival(frameBytes []byte, arrivalIface InterfaceID, interfaceSet []InterfaceID, fn ForwardFunc) error` |
| `routing.ErrAllPathsSplitHorizon` | `internal/routing/split_horizon.go` | VERIFIED — `var ErrAllPathsSplitHorizon = errors.New("routing: split-horizon: no eligible output interface (E-FWD-001)")` |
| `routing.InterfaceID` | `internal/routing/split_horizon.go` | VERIFIED — `type InterfaceID uint64` |
| `routing.ForwardFunc` | `internal/routing/split_horizon.go` | VERIFIED — `type ForwardFunc func(iface InterfaceID, frameBytes []byte) error` |
| `routing.RouteFrame` | `internal/routing/routing.go` | VERIFIED — `func RouteFrame(hdr frame.OuterHeader, payload []byte, r *Router) error` |
| `arqsend.New` | `internal/arqsend/arqsend.go` | VERIFIED — `func New(a *arq.ARQ, env outerassembler.Envelope, opts ...Option) *Retransmitter` |
| `arqsend.Retransmitter` | `internal/arqsend/arqsend.go` | VERIFIED — `type Retransmitter struct { ... }` |
| `arqsend.Retransmitter.Retransmit` | `internal/arqsend/arqsend.go` | VERIFIED — `func (r *Retransmitter) Retransmit(oldSeq, newSeq uint32, now time.Time, dispatch Dispatch) error` |
| `arqsend.Dispatch` | `internal/arqsend/arqsend.go` | VERIFIED — `type Dispatch func(wire []byte) error` |
| `arqsend.WithChanID` | `internal/arqsend/arqsend.go` | VERIFIED — `func WithChanID(chanID uint32) Option` |
| `arqsend.ErrSequenceNotInFlight` | `internal/arqsend/arqsend.go` | VERIFIED — `var ErrSequenceNotInFlight = errors.New("arqsend: oldSeq not in flight")` |
| `arq.New` | `internal/arq/arq.go` | VERIFIED — `func New(cfg Config) *ARQ` |
| `arq.ARQ` | `internal/arq/arq.go` | VERIFIED — `type ARQ struct { ... }` |
| `upstreamdial.New` | `internal/upstreamdial/connector.go` | VERIFIED — `func New(w io.Writer, env outerassembler.Envelope, keepaliveInterval time.Duration, initialAddrs []string) *Connector` |
| `upstreamdial.Connector` | `internal/upstreamdial/connector.go` | VERIFIED — `type Connector struct { ... }` |
| `upstreamdial.Handle` | `internal/upstreamdial/connector.go` | VERIFIED — `type Handle interface { ReloadAddrs(addrs []string); Mode() ConnMode; Stop() }` |
| `upstreamdial.ConnMode` | `internal/upstreamdial/connector.go` | VERIFIED — `type ConnMode int32` |
| `upstreamdial.ModeE` | `internal/upstreamdial/connector.go` | VERIFIED — `ModeE ConnMode = 0` |
| `upstreamdial.ModePE` | `internal/upstreamdial/connector.go` | VERIFIED — `ModePE ConnMode = 1` |
| `upstreamdial.BackoffBase` | `internal/upstreamdial/connector.go` | VERIFIED — `const BackoffBase = 500 * time.Millisecond` |
| `testenv.New` | `internal/testenv/testenv.go` | VERIFIED — `func New(ctx context.Context, t testing.TB) *Env` |
| `testenv.NewWithRouters` | `internal/testenv/testenv.go` | VERIFIED — `func NewWithRouters(ctx context.Context, t testing.TB, n int) *Env` |
| `testenv.RouterHandle` | `internal/testenv/testenv.go` | VERIFIED — `type RouterHandle struct { ... }` |
| `testenv.RouterMode` | `internal/testenv/testenv.go` | VERIFIED — `type RouterMode int` |
| `testenv.ModeE` | `internal/testenv/testenv.go` | VERIFIED — `ModeE RouterMode = iota` |
| `testenv.ModePE` | `internal/testenv/testenv.go` | VERIFIED — `ModePE` (second iota value) |
| `testenv.RouterHandle.SetConnector` | `internal/testenv/testenv.go` | VERIFIED — `func (r *RouterHandle) SetConnector(h upstreamdial.Handle)` |
| `testenv.RouterHandle.Mode` | `internal/testenv/testenv.go` | VERIFIED — delegates to `connector.Mode()` when connector non-nil |
| `testenv.Env.StartRouter` | `internal/testenv/testenv.go` | VERIFIED — `func (e *Env) StartRouter(t testing.TB, cfg RouterConfig) *RouterHandle` |
| `testenv.Env.PERouterAddr` | `internal/testenv/testenv.go` | VERIFIED — `func (e *Env) PERouterAddr(t testing.TB) string` |
| `testenv.Env.SendDrainSignal` | `internal/testenv/testenv.go` | VERIFIED — `func (e *Env) SendDrainSignal(t testing.TB, idx int)` |
| `metrics.PathEntryFromSnapshot` | `internal/metrics/handlers.go` | VERIFIED — `func PathEntryFromSnapshot(pathID string, snap paths.PathSnapshot) PathEntry` (produces `status: "failed"` when `snap.Failed`) |
| `paths.PathSnapshot.Failed` | `internal/paths/paths.go` | VERIFIED — `type PathSnapshot struct { ... Failed bool ... }` |
| `multipath.NewDropCache` | `internal/multipath/multipath.go` | VERIFIED — `func NewDropCache(capacity int) *DropCache` (panics if capacity < 1) |
| `multipath.DropCache` | `internal/multipath/multipath.go` | VERIFIED — `type DropCache struct { ... }` (zero value not usable; construct via NewDropCache) |
| `multipath.DefaultDropCacheSize` | `internal/multipath/multipath.go` | VERIFIED — `const DefaultDropCacheSize = 10_000` |
| `routing.SVTNRoute` | `internal/routing/routing.go` | VERIFIED — `func SVTNRoute(hdr frame.OuterHeader, payload []byte, r *Router) error` — called by RouteFrame; does NOT call OnFrameArrival |
| `routing.ErrDropCacheHit` | `internal/routing/on_frame_arrival.go` | VERIFIED — `var ErrDropCacheHit = errors.New("routing: drop cache hit — frame suppressed as loop duplicate (BC-2.02.009)")` |

### Appendix A Delta (v1.2 additions — Q9 fixture symbols)

New symbols introduced by Q9 (test-local; NEW definitions in `cmd/switchboard/router_pe_receive_test.go`):

| Symbol | File | Status |
|--------|------|--------|
| `peWriteFixture` | `cmd/switchboard/router_pe_receive_test.go` | NEW TYPE — test-local upstream fixture struct with `addr string`, `accepted chan net.Conn`, `ln net.Listener` |
| `startPEWriteFixture` | `cmd/switchboard/router_pe_receive_test.go` | NEW FUNCTION — `func startPEWriteFixture(t *testing.T) *peWriteFixture`; starts loopback TCP listener, accepts one conn (draining read loop on connector's bootstrap/keepalive writes), exposes it via `accepted` channel |
| `peWriteFixture.WriteFrame` | `cmd/switchboard/router_pe_receive_test.go` | NEW METHOD — `func (f *peWriteFixture) WriteFrame(t *testing.T, wire []byte)` — writes pre-assembled outer frame to the accepted PE connection |

Previously verified symbols reused by Q9 (no re-verification required):

| Symbol | Prior verification | Q9 usage |
|--------|--------------------|----------|
| `outerassembler.Assemble` | Appendix A v1.0 | Used in test to assemble `FrameTypeData` frame for fixture injection |
| `outerassembler.SACKBitmapSize` | Appendix A v1.0 | Used in zero-value SACK bitmap for test frame |
| `outerassembler.Envelope` | Appendix A v1.0 | Zero envelope (HMAC bypass per Q8 §8.5) |
| `halfchannel.ChannelFrame` | Appendix A v1.0 | Test frame struct with `FrameType: frame.FrameTypeData` |
| `frame.FrameTypeData` | Appendix A v1.0 | Non-bootstrap type to pass PE-CONNECT discard check |
