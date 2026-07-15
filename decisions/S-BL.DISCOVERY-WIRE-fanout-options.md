---
artifact_id: S-BL.DISCOVERY-WIRE-fanout-options
document_type: decision
level: ops
version: "1.1"
status: decided
producer: architect
timestamp: 2026-07-14T00:00:00Z
updated: 2026-07-14T20:00:00Z
cycle: cycle-1
stories_in_scope: [S-BL.DISCOVERY-WIRE]
bc_traces:
  - BC-2.03.001
  - BC-2.01.008
relates_to: S-BL.DISCOVERY-WIRE-rulings
---

# Options: S-BL.DISCOVERY-WIRE hop-2 fan-out target resolution — Forward Obligation (a)

**This is not a ruling.** It presents an expanded option space for the human to choose from at
the story-ready gate. No option is adjudicated here; each is evaluated honestly, including
against itself where the evaluation surfaces a problem. All factual claims are grep/read-verified
against the tree at develop HEAD `1f25677` (post `S-BL.CLI-SURFACE-COMPLETION` merge) — symbol-only
citations per TD-031, no line numbers.

## Why this document exists

At the story-ready human gate for `S-BL.DISCOVERY-WIRE`, the human reviewed rulings v1.8's Human
Gate item 3 and **rejected both offered resolution paths** for hop-2 fan-out target resolution
(Forward Obligation (a) — binding node identity `NodeAddr` to a live TCP connection so the router
knows which connections to relay a `DISCOVERY_RELAY` frame to):

- **Option (i)** — sequencing dependency on an unnamed, unscheduled future
  node-identity-to-connection-binding story. Rejected.
- **Option (ii)** — a narrow story-local `Router.BindInterface(svtnID, nodeAddr, ifaceID)` seam.
  Rejected, and the architect's own ruling flagged its problem at the time: the seam still needs
  *some* connection-time identity signal to call it with — it relocates the gap rather than
  closing it.

The human asked for better options. This document supplies six, genuinely distinct in mechanism,
grounded directly in what the shipped code makes cheap or expensive — not variations on (i)/(ii).

## Grounding — what is actually true in the code today

These facts drove every evaluation below; get them wrong and every option's effort/risk estimate
is wrong too.

1. **The TCP data-plane accept path (`onAccept` in `runRouter`) carries zero node identity.**
   `sendMap` is keyed by `routing.InterfaceID`, allocated in accept order by `netingress.Serve`'s
   `ifaceCounter`. No `NodeAddr`, no `SVTNID`, nothing identity-shaped is captured at connect time.
   `netingress`'s own doc comment calls `OnAccept`-eligible connections "ADMITTED" — that word means
   "passed the accept-count semaphore" (CWE-770 backpressure), a *different* meaning of "admitted"
   than `admission.AdmittedKeySet`'s. Do not conflate the two when reading this codebase.
2. **The actual node-identity admission handshake (`admission.AdmitNode`) is fully implemented,
   fully unit-tested, and has zero production callers.** Confirmed by rulings v1.8's own
   grep-verified N1 fold-in: 13 call sites, all `_test.go`. `AdmitNode` takes a `Challenge`
   (32-byte nonce + router `Ed25519` signature) and a `ChallengeResponse` (node's `Ed25519`
   signature over the nonce) — both structs are explicitly documented as wire-format-ready,
   containing no private key material. `admission.GenerateChallenge`, `AdmittedKeySet.LookupByPubkey`,
   and `AdmittedKeySet.Lookup` all already exist and are exercised by tests. **What is missing is
   purely wire transport** — nothing calls `GenerateChallenge`/`AdmitNode` over any live connection
   anywhere in production. This is a materially smaller gap than "build an admission handshake from
   scratch," and it directly undercuts the premise behind rejecting option (ii): the "connection-time
   identity signal" the architect's caveat pointed at is not unscoped hand-waving — it is one
   specific, already-built, already-tested cryptographic primitive with no wire glue.
3. **`FrameTypePEConnect` is not that signal.** It is `internal/upstreamdial`'s router-to-router
   peering bootstrap discriminator (`scope_phase: PE`, a phase beyond the current single-router
   MVP per rulings v1.8's own Verified Premises) — its receive loop silently discards the bootstrap
   frame and passes everything else through unchanged. It identifies a peer *router*, not an
   admitted *node*, and carries no `NodeAddr`/`SVTNID` payload at all. Team-lead's candidate (a)
   ("piggyback on the existing admission/PE-connect handshake") is two candidates conflated into
   one; the "admission handshake" half doesn't exist on any wire path yet (point 2), and the
   "PE-connect" half is a different protocol serving a different topology layer. Neither is a free
   ride.
4. **`routing.Router.RegisterForwardingEntry` — the thing that populates `forwardingTable`, which
   `RouteFrame` depends on for its `(SVTNID, SrcAddr)` HMAC-key lookup — also has zero production
   callers.** `admin.key.register`'s handler calls `SVTNManager.RegisterKey` (→
   `AdmittedKeySet.RegisterKey`) only; it never calls `router.RegisterForwardingEntry`. Consequence:
   `RouteFrame` fail-closes (`ErrHMACVerificationFailed`, PATH-A, "auth key unavailable") on
   **every** inbound TCP data-plane frame in production today, before it would ever reach the
   `admission.IsAdmitted` check. Ordinary session-data relay is itself a still-dormant stub
   (`routing.SVTNRoute`'s own doc comment: `entry` is "available for future use", `payload`/`entry`
   discarded via `_ =`). Any option that assumes "piggyback on already-flowing authenticated
   data-plane traffic" is piggybacking on traffic that does not exist yet in production, for a
   reason unrelated to this story.
5. **`AdmittedKeySet.ListBySVTN(svtnID)` already gives the "who" half of fan-out for free** — the
   set of `NodeAddr`s admitted to an SVTN, zero new code. It is only the "which live connection
   reaches that NodeAddr" half that is missing. Worth naming because it means Forward Obligation
   (a) is smaller than "identity resolution from scratch" — half of it is already answered.
6. **`control_type` registry has room.** `BC-2.01.008` v1.2's unassigned range is `0x04–0xFF`
   (DRAIN=0x01, RESYNC=0x02 reserved, DISCOVERY_RELAY=0x03 just claimed by this story), append-only,
   sequential assignment. A new opcode for any of the wire-handshake options below is a clean,
   precedented registry addition, not a scarce resource.
7. **`routing.SplitHorizon`/`FrameArrivalHandler.OnFrameArrival` and DRAIN's observer were already
   evaluated by Ruling 3(d)/(f) and are not fresh ground** — see Options 5 and 6 below, which check
   team-lead's candidates (c) and a pull-based variant against this prior work rather than
   re-deriving it.

---

## Option 1 — Minimal connect-time identify handshake, built inline in this story (wire the existing `AdmitNode` primitive)

**Mechanism.** Add one new `control_type = 0x04` (`NODE_IDENTIFY`) opcode. Immediately after TCP
connect, before any session-data frame, the connecting node sends a `NODE_IDENTIFY` frame carrying
its Ed25519 public key (or `NodeAddr`, router re-derives via `frame.DeriveNodeAddress` and looks up
the pubkey via `AdmittedKeySet.LookupByPubkey`). The router responds with a `Challenge`
(`admission.GenerateChallenge`, already implemented) over the same connection; the node replies with
a `ChallengeResponse` (`Sign(node_priv, nonce)`, already the documented wire shape). The router calls
the **existing, already-tested** `admission.AdmitNode(challenge, resp, pubKey, svtnID, ks)`
unmodified. On success, a new `Router.BindInterface(svtnID, nodeAddr, ifaceID)`-shaped method (the
same shape rejected option (ii) proposed, but now driven by a verified event instead of an
unspecified caller) records `(SVTNID, NodeAddr) → IfaceID` in a small map alongside `nodeConn`.

**Code surfaces touched.** New `control_type=0x04` case in `route()`'s switch in
`cmd/switchboard/mgmt_wire.go` (same shape as the existing DRAIN `case 0x01`); a small
challenge/response wire codec (two new fixed-ish-size ctl payloads); one new `Router` method +
map; `onAccept` gains a call-out to send the `Challenge` once the connection is registered.
`admission.AdmitNode`, `GenerateChallenge`, `Challenge`, `ChallengeResponse`, `LookupByPubkey` are
reused verbatim — zero changes to `internal/admission`.

**Closes or relocates the gap?** **Closes it.** This is not a relocated hand-wave — the identity
signal is a real, verified, cryptographically-authenticated event, using a primitive this codebase
already built and tested for exactly this purpose. It is what a future "session-bootstrap" story
would eventually deliver, scoped tightly to only the identity-binding slice (no key rotation UX, no
mid-connection re-admission, no revocation-at-handshake handling — flag those as explicit Non-Goals
if this option is chosen).

**Interaction with SEC-DW-08/09 + originator exclusion + BC-2.01.008.** Clean. AC-017's
exclude-originator postcondition becomes trivially correct (the router now knows exactly which
`IfaceID` is the originating `NodeAddr`'s connection). SEC-DW-09's rate cap is unaffected (it gates
relay dispatch, not this handshake). `BC-2.01.008` gains one registry row (`0x04`), same precedent
as `0x03`'s addition. No SVTN-isolation trade-off — this option, unlike Option 4 below, never sends
presence data to a non-member connection.

**Effort vs. this story's 8 pts.** Meaningfully larger than the rejected seam — new wire codec, new
router state, new tests for the handshake itself (success, wrong-SVTN, revoked-key, replayed-nonce
paths already covered by `AdmitNode`'s existing test suite, but the wire-transport wrapper needs its
own). Estimate: **+4 to +6 points** grafted onto this story, or scoped as its own story (see
Option 3). Growing *this* story's scope is a real cost — Task Breakdown, File-Change List, and
Token Budget would all need rework mid-elaboration.

**Failure modes.** A node that never completes the handshake (bad clock, revoked key, network
drop mid-handshake) simply never gets bound — same fail-closed posture `IsAdmitted` already has
elsewhere. A slow handshake delays first eligibility for relay receipt, not a correctness bug.

**Architect's read.** The strongest *closure* of the gap, and cheaper than it first looks because
the crypto is already built — but it is new scope grafted onto a story whose points were never
sized for it. Best paired with Option 3 (ship it as its own immediately-scheduled story) rather than
folded in here, unless the human explicitly wants `S-BL.DISCOVERY-WIRE`'s own points to absorb it.

---

## Option 2 — Lazy-bind on already-authenticated ordinary data-plane traffic (looks cheap, is not — included as a cautionary option)

**Mechanism.** Hook `RouteFrame`'s success path (mirroring the existing `forwardingEntryHook`
pattern) so that the first HMAC-verified, admitted frame from a connection records
`(SVTNID, SrcAddr) → IfaceID`. To make `RouteFrame` ever succeed in production, also wire
`router.RegisterForwardingEntry(svtnID, nodeAddr, authKey)` into `admin.key.register`'s handler
(reusing `AdmittedKey.FrameAuthKey`, already computed at `RegisterKey` time) — closing an adjacent,
pre-existing gap as a byproduct.

**Code surfaces touched.** `internal/routing` (new hook, mirroring `ForwardingEntryHook`);
`cmd/switchboard/admin_handlers.go` (`makeRegisterHandler` gains a `router.RegisterForwardingEntry`
call); `cmd/switchboard/mgmt_wire.go` (`route()` closure records the binding on success).

**Closes or relocates the gap?** **Neither, cleanly — it bottoms out on a third, deeper gap.**
Wiring `RegisterForwardingEntry` makes `RouteFrame` pass step 1 (auth-key lookup), but `RouteFrame`
step 3 (`admission.IsAdmitted`) still requires `entry.admitted == true`, which is only ever set by
`AdmitNode` — the same zero-production-caller function Option 1 wires up. **This option, pursued in
isolation, is inert**: it looks like a small, self-contained fix (two hook wirings) but actually
requires Option 1's handshake to be built anyway before any frame would ever pass `RouteFrame` in
production. Included specifically so this isn't independently rediscovered as "the cheap one" —
it is not.

**Interaction with SEC-DW-08/09 + originator exclusion + BC-2.01.008.** N/A while inert. If Option
1 is also built, this option becomes a redundant *second* path to the same binding (no new value).

**Effort vs. this story's 8 pts.** Small in isolation (~1 point: two hook wirings) but that estimate
is misleading — it does not deliver Forward Obligation (a) on its own.

**Failure modes.** The obvious one: shipping this alone and believing the gap is closed, when in
fact zero connections will ever bind (no traffic ever reaches `RouteFrame` success without
`AdmitNode` also being wired).

**Architect's read.** Do not choose this in isolation. Worth keeping in the option space only
because it surfaces, honestly, how deep the missing-admission-handshake hole goes — two other
pieces of "already almost there" machinery (`forwardingTable` population, `RouteFrame`'s admitted
check) both independently confirm Option 1's gap is the real bottleneck, not a symptom of this
story's scope specifically.

---

## Option 3 — Name and schedule the companion story now, not "whatever future story" (reframes rejected option (i))

**Mechanism.** Identical payload to Option 1 (the `NODE_IDENTIFY` handshake), but delivered as its
own, immediately-following, already-named, already-scoped, already-pointed story — e.g.
`S-BL.NODE-IDENTIFY` — created and added to `S-BL.DISCOVERY-WIRE`'s `depends_on` *today*, not left
as an unnamed placeholder. `S-BL.DISCOVERY-WIRE` ships Tasks 1-5 (hop-1 ingest + hop-2 frame
construction) immediately, ungated, exactly as already planned; Task 6/AC-017/AC-018 gate on this
named, scheduled, scoped successor rather than an open-ended one.

**Why this is not the rejected option (i).** (i) was rejected — most plausibly — for its
open-endedness: "whatever future story delivers node-identity-to-connection binding," no name, no
schedule, no committed scope, referenced only generically as `FO-DRAIN-WIRE-002` with "no successor
story exists yet." This option removes exactly that property. The successor is named now, scoped
now (to Option 1's mechanism specifically — not a broader "session bootstrap" umbrella that could
scope-creep), and scheduled as the *next* story, not an indefinite future one.

**Code surfaces touched.** None in `S-BL.DISCOVERY-WIRE` itself beyond a `depends_on` edge and
Task 6's existing GATED framing (already present). All the Option-1-shaped code lands in the new
story.

**Closes or relocates the gap?** Closes it, on a committed timeline — but not inside this story.
This is a genuine third shape distinct from both rejected options: not "defer indefinitely" (i) and
not "build a stub seam that still needs the signal" (ii) — it is "build the real thing, just as the
very next unit of work, with a name."

**Interaction with SEC-DW-08/09 + originator exclusion + BC-2.01.008.** Identical to Option 1 once
the successor lands. Until then, AC-017/AC-018 remain correctly GATED — no security regression in
the interim; Tasks 1-5 ship real value (hop-1 ingest, replay protection, hop-2 frame construction)
with zero fan-out capability, which is the same interim state the story already plans for.

**Effort vs. this story's 8 pts.** **This story's 8 points are unaffected.** The successor story
needs its own point estimate (~4-6, mirroring Option 1's grafted-on delta, now sized as a whole
story rather than a graft).

**Failure modes.** If the named successor slips or is deprioritized, `S-BL.DISCOVERY-WIRE`'s
Task 6 stays gated indefinitely — same residual risk (i) had, but now visible against a named,
trackable story rather than an untraceable placeholder, so slippage is observable (`bd blocked`,
wave-scheduling visibility) rather than silent.

**Architect's read.** This is the recommended default if the human's objection to (i) was its
open-endedness rather than the idea of sequencing itself. It keeps this story's scope and points
honest while still committing to closing the gap on a concrete, visible timeline.

---

## Option 4 — Global (cross-SVTN) best-effort broadcast, defense relocated to the receiving node

**Mechanism.** Router-side fan-out does zero target resolution: reuse `sendMap.Range` verbatim,
the exact DRAIN dispatch shape, broadcasting the `DISCOVERY_RELAY` frame to *every* connected
`nodeConn` regardless of SVTN or originator. SVTN-scoping is relocated to the receiver: the
node-local ingest function already adjudicated by rulings v1.8's F-DWSP8-001 fix compares the
frame's `OuterHeader.SVTNID` against `d.cfg.LocalSVTNID` and discards on mismatch — this exists
today, no new code. Add one new receiver-side check: discard if the frame's `NodeAddr` and
`Sequence` match this node's own last-sent values (the node already tracks its own outbound
`Sequence` state in memory — cheap comparison, no new state class).

**Code surfaces touched.** Router side: none beyond reusing the existing `sendMap.Range` shape —
literally the cheapest possible code path, zero new binding infrastructure. Node side: one new
comparison in the relay-ingest function (`internal/discovery`), plus AC-017 postconditions 1 and 3
need rewording (delivery guarantee changes from "never sent to non-members / never receives an
echo" to "received-then-discarded" for both cases).

**Closes or relocates the gap?** Neither — it **eliminates the need for it** by changing what the
router promises. No `NodeAddr`↔connection binding is required at all.

**Interaction with SEC-DW-08/09 + originator exclusion + BC-2.01.008.** This is where the honest
cost lives. `BC-2.03.001` Postcondition 1 currently reads "the advertisement is multicast to all
admitted nodes **on the SVTN**" — broadcasting to non-member connections is a literal violation of
that postcondition as worded today, not just an isolation-hygiene concern; it would need a BC
amendment, not just an AC reword. More substantively: session-name/attach-status presence data for
SVTN-A would transit the wire to SVTN-B's (and every other tenant's) open connections before being
discarded — a new cross-SVTN information exposure that SEC-DW-08's "HMAC is the sole boundary"
framing does not already cover (SEC-DW-08 was scoped to hop-1's multicast-segment leak, where the
leak is to anyone on the LAN who was never going to be authenticated anyway; this is a leak to
*other admitted-but-wrong-SVTN* nodes on hop-2, a materially different trust boundary). This needs
its own dedicated security sign-off, not an inherited one. AC-017 postcondition 3's "never receives
an echo" also becomes false as literally worded (the node *does* receive it, then discards it) —
needs rewording to "never acts on an echo of its own advertisement," a real but honest weakening.

**Effort vs. this story's 8 pts.** **Near zero new router-side code (~0-1 points).** Node-side
self-echo check is small (~0.5 points). Cheapest option in the set by a wide margin.

**Failure modes.** Presence-data exposure across SVTN tenants scales with the number of SVTNs a
single router multiplexes — a router serving many SVTNs turns every discovery heartbeat into an
all-tenants broadcast. Rate cap (SEC-DW-09) still applies per-originator, but does not bound the
*number of non-member recipients* a single relay reaches, which is a different amplification axis
than SEC-DW-09 was designed against.

**Architect's read.** Genuinely the cheapest option and worth surfacing honestly — but it requires
a `BC-2.03.001` PC-1 amendment (not just this story's own AC wording) and very likely a security-
reviewer sign-off given the new cross-SVTN exposure. Recommend only if the human is comfortable
formally relaxing the BC's SVTN-scoping promise; do not present this as free.

---

## Option 5 — Pull instead of push (node-initiated poll) — evaluated and discarded, documented for completeness

**Mechanism considered.** Instead of the router resolving targets and pushing, admitted nodes poll
the router for pending SVTN-scoped relayed advertisements over their own already-open connection,
via a new request/response ctl-frame pair.

**Why discarded.** Two independent dead ends, not one: (1) it doesn't avoid the binding problem, it
reshapes it — the router still needs to know which SVTN a *bare* TCP connection belongs to in order
to filter what it hands back on a pull, which is the same missing signal Options 1/3 build; a pull
adds new wire-protocol surface (a request/response pair, plus queue-per-SVTN state) for no net
reduction in the actual gap. (2) The one place in this codebase where an authenticated-caller-identity
mechanism *already* exists — the mgmt RPC plane's `resolveAndVerifyCallerRole`, used by
`admin.key.*`/`router.reload`/`router.drain` — cannot be reused for this at all: router-mode's mgmt
listener binds a local Unix socket (`/run/switchboard-router.sock`), reachable only by same-host
operator tooling (`sbctl`), not by remote access nodes over the network the data-plane TCP listener
serves. There is no free identity signal hiding in the mgmt plane for this use case.

**Architect's read.** Not a real sixth option — logged so the human can see it was checked, not
skipped.

---

## Option 6 — Reuse the DRAIN-over-SVTN observer-registration machinery as the identity↔connection map — evaluated and discarded, already consumed

**Mechanism considered.** Team-lead's candidate (c): reuse `S-7.04-FU-DRAIN-WIRE`'s per-node
observer/dispatch registration as the source of the identity binding hop-2 needs.

**Why discarded.** Checked directly against the shipped code, not assumed. DRAIN's dispatch is a
**single** startup observer (`Q-SINGLE-OBS`) registered once at `drainCoord` construction; at
`Signal` time it does `sendMap.Range` over literally every connected node with **zero** filtering —
no SVTN check, no `NodeAddr` check, nothing to exclude an originator, because DRAIN has no
originator concept. "Per-node" in that story's title refers to *each node's own send channel being
targeted individually inside the Range loop*, not to any per-node identity registration — `sendMap`
there is the *exact same* `IfaceID`-only, zero-identity map this document's Grounding section
describes in point 1. Ruling 3(d) already reused DRAIN's *dispatch shape* (the
`select{case nc.send<-frame: default:}` best-effort pattern) for hop-2's relay closure — that reuse
is already landed in the story's Decision 3(d). There is no additional identity information left to
extract from DRAIN; asking for it a second time would find the same empty map.

**Architect's read.** Not a real sixth option either, for a different reason than Option 5 — the
substance was already adopted (dispatch shape) and the part that sounds promising (identity
binding) isn't actually there. Logged so the human can see it was checked against the real code, not
dismissed on the name alone.

---

## Comparison table

| # | Option | Closes / relocates / eliminates the gap | Router-side new code | Node-side new code | BC/AC changes needed | Effort vs. 8 pts | Primary risk |
|---|---|---|---|---|---|---|---|
| 1 | Minimal identify handshake, inline | Closes | New opcode + codec + `BindInterface` + map | None (reuses `AdmitNode` as-is) | None (AC-017/018 land as already written) | +4 to +6 pts grafted on | Grows this story's scope mid-elaboration |
| 2 | Lazy-bind on data-plane traffic | Neither (inert alone) | Two hook wirings | None | None | ~1 pt but delivers nothing alone | False sense of "cheap" — silently depends on Option 1 anyway |
| 3 | Name + schedule the companion story now | Closes, on a committed timeline, out-of-story | None in this story | None in this story | `depends_on` edge added now | 0 pts here; ~4-6 pts in the new story | Successor could still slip — but visibly, not silently |
| 4 | Global broadcast + receiver-side defense | Eliminates the need for binding | ~0-1 pts (reuse DRAIN shape) | ~0.5 pts (self-echo check) | **BC-2.03.001 PC-1 amendment** + AC-017 reword | Cheapest by far | New cross-SVTN presence-data exposure; needs security sign-off |
| 5 | Node-initiated pull | N/A — discarded | — | — | — | — | Reshapes, doesn't close, the gap; no reusable identity signal exists on the mgmt plane for remote nodes |
| 6 | Reuse DRAIN observer registration | N/A — discarded | — | — | — | — | Already consumed for dispatch shape (Ruling 3(d)); carries zero identity information |

## Ranked recommendation

This section states the architect's read, not a ruling — the human decides.

1. **Option 3** (name + schedule the companion story now) is the closest fit to what the rejection
   of (i) most plausibly objected to (open-endedness, not sequencing itself), while keeping this
   story's own scope and points honest.
2. **Option 1** (build the same handshake, but inline in this story) is the same mechanism as
   Option 3, cheaper to reason about in one place, more expensive to this story's own points and
   elaboration state — pick this over Option 3 only if the human wants the gap fully closed in one
   PR rather than two sequenced ones.
3. **Option 4** (global broadcast + receiver-side defense) is legitimate and by far the cheapest,
   but is not a free lunch — it requires a real `BC-2.03.001` amendment and a security-reviewer
   look at the new cross-SVTN exposure. Good choice only if the human is comfortable formally
   relaxing the SVTN-scoping promise.
4. **Option 2** should not be chosen in isolation — it is documented so it is not independently
   proposed later as "the cheap fix" without realizing it depends on Option 1's handshake anyway.
5. **Options 5 and 6** are ruled out, not merely deprioritized — logged for audit completeness.

---

## Disposition (v1.1, 2026-07-14)

**Human selected Option 1** at the `S-BL.DISCOVERY-WIRE` story-ready gate, 2026-07-14 — the same day
this document was produced. This document is now a **decided record**: the option space it presents
remains accurate as an evaluation, but the choice among options is closed. Do not re-open this
comparison as if it were still live; a new architect ruling, not an edit to this document, is the
correct vehicle if the decision is ever revisited.

**What "Option 1" means as executed.** Option 1 as originally scoped in this document was two
variants sharing one mechanism (the `NODE_IDENTIFY` handshake) differing only in *where* it lands —
grafted inline into `S-BL.DISCOVERY-WIRE` (Option 1 literally) or delivered as a named, scheduled
companion story (Option 3). The human's selection is Option 1's *mechanism* delivered via Option 3's
*shape*: a new, immediately-named companion story, **`S-BL.NODE-IDENTIFY-WIRE`**, carries the
handshake; `S-BL.DISCOVERY-WIRE`'s own points (8) and Tasks 1-5 are unaffected, and Task 6/AC-017/
AC-018 gate on the new story by name. Full disposition text, rationale, and the story ID are recorded
in `S-BL.DISCOVERY-WIRE-rulings.md` v1.9, Ruling 3 subsection "Ruling 3(f) Forward Obligation,
SEC-DW-07, and the discovery port — human gate disposition," item (j) — that document is now the
authoritative record of the disposition; this document remains the authoritative record of the
*evaluation* that produced the six options.

**Options 2 and 3 — not selected, closed.** Option 2 (lazy-bind on data-plane traffic) was never a
real contender in isolation, per its own "Architect's read" above (inert without Option 1). Option 3
(name-and-schedule with Option 1's mechanism deferred to the successor) is the shape the human's
selection adopted structurally, but the human chose it *combined with* Option 1's substance rather
than Option 3 read narrowly against Option 1 as two separate, mutually exclusive choices — see "What
Option 1 means as executed" above. Both are closed as independently-selectable options.

**Options 4, 5, 6 — discarded, closed.** Option 4 (global broadcast + receiver-side defense) was not
selected; its `BC-2.03.001` PC-1 amendment and security-reviewer-sign-off cost were not incurred.
Options 5 and 6 were already discarded at v1.0 (not real contenders — see their own sections above)
and remain discarded; nothing in this disposition reopens them.

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.1 | 2026-07-14 | Human disposition recorded: Option 1 selected at the `S-BL.DISCOVERY-WIRE` story-ready human gate (companion story `S-BL.NODE-IDENTIFY-WIRE` named and scheduled, Option 1's `NODE_IDENTIFY` handshake mechanism delivered via Option 3's name-and-schedule-now shape). New "Disposition" section added above; options 2-6 formally closed (2/3 not independently selected, 4-6 discarded per the v1.0 evaluation, unchanged). `status: options-for-human-review` → `decided`. No change to the six options' evaluations, the Grounding section, the Comparison table, or the Ranked recommendation — this is a disposition record layered on top, not a re-evaluation. |
| 1.0 | 2026-07-14 | Initial publication. Six options for `S-BL.DISCOVERY-WIRE` Ruling 3(f)'s fan-out target-resolution Forward Obligation, produced after the human rejected both paths `S-BL.DISCOVERY-WIRE-rulings.md` v1.8's Human Gate item 3 originally offered. Options 1-4 genuinely distinct and evaluated on their merits; Options 5-6 evaluated and discarded. Ranked recommendation: Option 3, then Option 1, then Option 4 (conditional), Option 2 not in isolation, Options 5/6 ruled out. |
