---
artifact_id: S-BL.DATAPLANE-CONNECTOR-scoping-note
document_type: architect-design-note
story_id: S-BL.ACCESS-CONNECTOR, S-BL.CONSOLE-CONNECTOR   # proposed — not yet created; story-writer produces the actual story files
title: "Naming and bounding the endpoint-side data-plane connector prerequisite(s) — Layer 2 of S-BL.RESYNC-FRAME"
status: draft
producer: architect
timestamp: 2026-08-31T00:00:00Z
version: "1.0"
bc_traces:
  - BC-2.04.001   # access-node-to-tmux local half — closest existing BC; wire-connection establishment is NOT in its scope (see §3)
  - BC-2.04.003   # console attach/downstream-stream semantics — session-layer only, transport-agnostic (see §3)
  - BC-2.09.001   # E->PE graduation — router-to-router precedent; NOT reusable as-is for the access-node leg (see §2)
  - BC-2.09.003   # PE-CONNECTOR's PC-9 connect-half precedent
vp_traces: []   # no VPs minted by this note — see §5 for itemized needs; next free id is VP-081+
architecture_modules:
  - cmd/switchboard         # runAccess/runAccessWithConnector (access.go) gains the dial site; runConsole (mgmt_wire.go) is currently mgmt-plane-only
  - internal/upstreamdial    # PE-CONNECTOR's shipped package — pattern precedent, not directly reused code (see §2)
  - internal/session         # AccessNode/Publisher/ConsoleKey — the in-process session layer the connector must feed into
  - internal/arq              # SendBuffer the access-node connector must be constructed against
  - internal/arqsend         # Retransmitter — the send-path the connector's live connection ultimately dispatches through
  - internal/outerassembler  # session-bootstrap frame assembly (Q6 three-step pattern, reused)
related_documents:
  - .factory/decisions/S-BL.RESYNC-FRAME-placement-note.md
  - .factory/stories/S-7.04-FU-PE-CONNECTOR.md
  - .factory/decisions/S-7.04-FU-PE-CONNECTOR-placement-note.md
  - .factory/decisions/RULING-W6TB-C-console-transport.md
  - .factory/specs/architecture/ARCH-03-routing-engine.md
  - .factory/specs/behavioral-contracts/ss-04/BC-2.04.001.md
  - .factory/specs/behavioral-contracts/ss-04/BC-2.04.003.md
  - .factory/specs/behavioral-contracts/ss-09/BC-2.09.001.md
  - cmd/switchboard/access.go
  - cmd/switchboard/mgmt_wire.go
  - cmd/switchboard/node_identify_wire.go
  - internal/upstreamdial/connector.go
---

## Changelog

| Version | Change |
|---------|--------|
| 1.0 | Initial scoping note. Names S-BL.ACCESS-CONNECTOR (high-confidence, PE-CONNECTOR-shaped) and S-BL.CONSOLE-CONNECTOR (named but flagged lower-confidence — `runConsole` has zero data-plane scaffolding today, and RULING-W6TB-C already moved console's control surface off the SVTN wire model, so the console leg's shape is a genuinely open architectural question this note surfaces rather than presupposes). Written as the follow-on to `S-BL.RESYNC-FRAME-placement-note.md` §3/§8, at the human's request to name Layer 2 before S-BL.RESYNC-FRAME is decomposed. |

---

## 0. Context and purpose

`S-BL.RESYNC-FRAME-placement-note.md` (§3, §8) found that no production or test-harness code path in this repository performs real wire-level session-data relay between an access node and a router: `runAccess`/`runAccessWithConnector` (`cmd/switchboard/access.go`) construct `session.AccessNode`, a downstream `halfchannel.HalfChannel`, and a router instance that is explicitly "constructed-but-not-in-live-data-path," but never `net.Dial`, never call `outerassembler.Assemble`, and never read/write a wire frame. This note names and bounds that gap as proper backlog story(ies) per the human's request, so story-writer can create stub files and STORY-INDEX rows. It does not create story files, does not edit STORY-INDEX, and does not create or amend any BC/VP (itemized in §5, routed to product-owner).

---

## 1. Story name(s) — one vs. two, and why

**Two proposed stories, asymmetric in confidence:**

1. **`S-BL.ACCESS-CONNECTOR`** — the access-node-side data-plane connector. High-confidence scope (§2): directly precedented by the shipped `S-7.04-FU-PE-CONNECTOR`, and the receiving-side admission primitives it needs (`NODE_IDENTIFY`) now exist in production, unlike when PE-CONNECTOR shipped.
2. **`S-BL.CONSOLE-CONNECTOR`** — the console-side data-plane connector. Named, but flagged as materially lower confidence and possibly premature to size as a normal story today (§4).

**Naming pattern justification:** mirrors `S-7.04-FU-PE-CONNECTOR`'s own shape — `{MODE-NAME}-CONNECTOR}` — where `PE` is the router's own CLI mode name once the connector is live. The parallel CLI mode names in `cmd/switchboard/main.go` are `"access"` and `"console"` (confirmed: `case "access":`, `case "console":`, alongside `"router"` and `"control"`), so `ACCESS-CONNECTOR`/`CONSOLE-CONNECTOR` is the direct analog. Both sit in the generic `S-BL.*` backlog namespace (matching `S-BL.RESYNC-FRAME`, `S-BL.OA`, `S-BL.NI`, `S-BL.DISCOVERY-WIRE`, `S-BL.NODE-IDENTIFY-WIRE`, etc.), not the epic-scoped `S-7.04-FU-*` namespace, since neither is a deferred obligation of a specific numbered epic story the way PE-CONNECTOR was of S-7.04.

**Why two, not one:** the two legs are not symmetric in what is currently known about them, and forcing them into a single story would either under-scope the console leg or over-block the access leg on an unresolved architectural question:

- The **access-node leg** has an unambiguous target shape: ARCH-03's "Downstream ARQ" section is explicit that `SendBuffer` lives at the access node, `S-7.04-FU-PE-CONNECTOR` already solved the identical *class* of problem (outbound dial + reconnect/backoff + session-bootstrap wire frame) for a structurally similar role, and the missing admission primitive (`NODE_IDENTIFY` client-side handshake) is a well-defined, boundable gap against code that already exists server-side. This can be scoped today with the same confidence PE-CONNECTOR itself was scoped with.
- The **console leg** does not have an unambiguous target shape yet. `runConsole` (`cmd/switchboard/mgmt_wire.go:1192+`) constructs only a management-plane RPC server (`newMgmtServer(cfg, "console", ...)` + `BuildConsoleHandlers`/`BuildSessionsHandlers`) and explicitly comments *"Console mode has no routing subsystem — pass nil router"* — there is no halfchannel, no ARQ, no frame read/write, not even a deferred stub for one. Separately, `RULING-W6TB-C-console-transport.md` (final, 2026-07-01) already made a deliberate, adjudicated decision to move console's *control* surface (attach/detach/switch) **off** the SVTN data-plane model and onto the mgmt-plane Unix socket, specifically to avoid building a second protocol stack in `sbctl`/console (its own rationale: *"sbctl has no ARQ stack... Implementing [the SVTN-channel interpretation] would require building a second control-message protocol inside the SVTN data plane — a major architectural undertaking with no precedent in the codebase"*). That ruling is scoped to control commands, not the downstream terminal-output data stream itself — but its existence, combined with `runConsole`'s complete absence of data-plane scaffolding, means it is a **live, unresolved architectural question** — not merely an unbuilt-but-settled one — whether the console's actual session-data reception should follow the same SVTN-wire dial-loop model this note proposes for the access node, or some other mechanism. Collapsing that open question into the well-bounded access-node story would either force a premature answer or block the access-node work on it unnecessarily.

Splitting lets `S-BL.ACCESS-CONNECTOR` proceed on solid footing immediately while `S-BL.CONSOLE-CONNECTOR` carries its own uncertainty explicitly, rather than diluting the access story's confidence.

---

## 2. `S-BL.ACCESS-CONNECTOR`

**Purpose:** give the access-mode daemon a live, reconnecting TCP data-plane connection to its router, so `internal/arq`'s downstream `SendBuffer` and `arqsend.Retransmitter` can dispatch real wire frames instead of the current in-process-only path.

### Scope boundary — exactly what this story adds

1. A new effectful package, `internal/accessdial` (name chosen to mirror `internal/upstreamdial`'s naming; final name is story-writer's/architect's call at scoping time), providing a `Connector` type: `net.Dial("tcp", routerAddr)` to the configured router address, with reconnect/backoff on failure or connection loss.
2. **Client-side NODE_IDENTIFY handshake** — the genuine new wire-protocol work this story adds beyond PE-CONNECTOR's precedent (§2.1 below): send `NodeIdentify` (reusing `encodeNodeIdentify`, already shipped in `cmd/switchboard/node_identify_wire.go:105`), receive and decode the router's `Challenge` response (a **new** `decodeChallenge` function — only the router-side `encodeChallenge` exists today, at `node_identify_wire.go:130`; no client-side decoder exists), sign the challenge nonce and send `ChallengeResponse` (reusing `encodeChallengeResponse`, `node_identify_wire.go:160`). This is real admission, not the placeholder/deferred bootstrap PE-CONNECTOR shipped with (see §2.1).
3. Integration with `internal/outerassembler.Assemble`/`internal/arqsend.Retransmitter`: once the connection is admitted, the `Envelope{SVTNID, SrcAddr, DstAddr, FrameAuthKey}` derived from the completed handshake is handed to the `arqsend.Retransmitter` construction site (or an equivalent live-dispatch wiring in `runAccessWithConnector`), and the connector's `net.Conn.Write` becomes the `Dispatch` callback `arqsend.Retransmitter.Retransmit` already expects (`internal/arqsend/arqsend.go:64`, `Dispatch func(wire []byte) error`).
4. A frame **read** loop on the live connection: `netingress.ReadFrame`-equivalent (`internal/netingress` itself is accept-loop-owned and architecturally upstream of `internal/routing`, per its own import-constraint doc comment — this story either reuses `netingress.ReadFrame`'s exported function directly, since it is a free function not bound to `Serve`'s accept-loop machinery, or duplicates the same bounded-read discipline; story-writer's call) to receive frames the router relays down to this access node — including, eventually, RESYNC frames per `S-BL.RESYNC-FRAME`'s router-side relay design (`S-BL.RESYNC-FRAME-placement-note.md` §4.3).
5. Per-connection lifecycle: connection-established → admitted → live-data states; teardown on read/write error triggers reconnect with backoff (parameters TBD at scoping time — `internal/upstreamdial`'s Q5-ruling constants, `BackoffBase=500ms`/`BackoffCap=30s`/25% jitter, are a directly reusable precedent, not a binding requirement).
6. `runAccess`/`runAccessWithConnector` (`access.go`) wiring: construct the `Connector` at startup (mirroring `runRouter`'s `connector := upstreamdial.New(...)` call site), pass the router address from config (a new config field, e.g. `router_addr`, analogous to `upstream_routers`), start it, wire its received frames into `session.AccessNode`'s existing delivery path.

### Non-goals

- Does **not** implement RESYNC's own emitter/receiver/replay-trigger logic — that is `S-BL.RESYNC-FRAME`'s own scope (Layer 1, independent of this story; see §6).
- Does **not** implement multi-router failover selection (which router to dial when more than one is configured, or ADR-005's "reconnects to another router" scenario at the policy level) — single configured router address is the MVP scope, matching "E router has a single path" (ARCH-03 §ADR-005).
- Does **not** touch `internal/routing`, `internal/drain`, or `internal/testenv` from the new package (forbidden-edge precedent directly inherited from `S-7.04-FU-PE-CONNECTOR`'s own placement note Q4 ruling — `internal/upstreamdial → internal/routing` and `→ internal/testenv` are both named-forbidden there for the same layering reasons that apply here).
- Does **not** resolve the console-side leg — see `S-BL.CONSOLE-CONNECTOR` (§4).
- Does **not** implement key/credential persistence beyond whatever `internal/admission`'s existing client-side signing primitives require (ed25519 keypair loading is already solved elsewhere in `access.go`'s `loadOrGenerateAdmissionKeypair` — reused, not rebuilt).

### BC/VP anchors

- **Traces to** (existing, no change needed): BC-2.04.001 Precondition 3 ("The access node is admitted to an SVTN (or admission in progress)") — this story is what actually discharges that precondition; BC-2.04.001 itself stays `architecture_module: internal/tmux` and out of scope for editing.
- **Needs new BC content** — see §5, item 1.
- **VP needs** — see §5, item 3.

### Points estimate

**8–13 points.** Directly comparable in shape and size to `S-7.04-FU-PE-CONNECTOR` (delivered at 8 points, per its own frontmatter `estimated_points: 8`), plus real material this story does that PE-CONNECTOR explicitly deferred: PE-CONNECTOR's own Q6 design constraint states admission was *"DEFERRED (class: not-core)... The Connector is constructed with zero-valued `outerassembler.Envelope` fields... Full node-identity derivation... is deferred as not-core"* — i.e., PE-CONNECTOR shipped an **unauthenticated** bootstrap. This story cannot take that shortcut: RESYNC's Layer 2 needs a genuinely admitted, identity-bound connection (so `routing.LookupInterface`/`BindInterface`, which key on `(svtnID, nodeAddr)`, resolve correctly), and the client-side `NODE_IDENTIFY` handshake did not exist as a reusable primitive when PE-CONNECTOR was scoped (it shipped a week later — see §2.1). Estimate weighted above PE-CONNECTOR's own 8 points for this reason.

### 2.1 Relationship to the merged `S-7.04-FU-PE-CONNECTOR` — precise

**What PE-CONNECTOR delivered** (verified against `.factory/stories/S-7.04-FU-PE-CONNECTOR.md`, merged PR #115, `internal/upstreamdial/connector.go`): an outbound `net.Dial` loop from a **router in PE mode** to its configured `upstream_routers` — i.e., a **router-to-router** hierarchy leg, gated by `BC-2.09.001` ("E Router Graduates to PE Mode by Adding Upstream Router Connections in Config"). Its `Connector` type lives in `internal/upstreamdial` (DAG position 19/20, imports `{halfchannel, outerassembler}` only), is constructed and driven entirely by `runRouter` (`cmd/switchboard/mgmt_wire.go:893-909`), and its "connection established" contract is a three-step definition (binding, per its placement note Q6): `net.Dial` succeeds, `outerassembler.Assemble` succeeds, `conn.Write` succeeds — **with admission explicitly deferred** ("not-core"): the `Connector` is constructed with a **zero-valued** `outerassembler.Envelope` (no real `SrcAddr`/`DstAddr`/`SVTNID`/`FrameAuthKey`) and the bootstrap frame uses `halfchannel.FrameTypeData` as a placeholder rather than a distinct connect frame type, per the story's own "Shipped deferral (implementation-era note)" section.

**Does the access-node leg differ from, parallel, or reuse PE-CONNECTOR?** All three, precisely:

- **Parallels (pattern, not code):** the dial-loop shape (encapsulated `Connector` type owning reconnect/backoff state, driven by the daemon's `run*` entry point, exposing a narrow `Handle` interface, communicating address/config changes via a buffered channel rather than shared mutable state) is a directly reusable architectural pattern. The Q6 three-step "connection established" definition is the right shape to reuse for the access node too (dial → bootstrap-assemble → write), just with a genuinely populated `Envelope` instead of a zero one (see below).
- **Reuses (literal code, cross-package):** `internal/outerassembler.Assemble` itself (already package-general, not PE-specific); `cmd/switchboard/node_identify_wire.go`'s message-codec functions (`encodeNodeIdentify`, `encodeChallengeResponse`, and — once written — the new `decodeChallenge`) are directly reusable, since they encode/decode wire bytes, not router-only logic.
- **Does not reuse (separate package, separate DAG node):** `internal/upstreamdial.Connector` itself is not imported or extended — it is constructed exclusively by `runRouter` for the router-to-router role, and its own placement note forbids exactly the kind of routing/testenv coupling the access-node connector would also need to avoid, independently. A new package (`internal/accessdial` or equivalent) is the right unit, mirroring `internal/upstreamdial`'s existence as its own DAG node rather than folding into it — matching the codebase's existing convention of one connector package per connecting *role* (there is no single generic "dialer" package shared across router-PE and access-node; `internal/upstreamdial`'s own name is role-specific).
- **The genuine, previously-unnamed gap PE-CONNECTOR did NOT cover:** the access-node → router leg entirely. PE-CONNECTOR's own story text is explicit that it is the *router*'s outbound leg (*"Outbound TCP Dial Loop on PE Graduation"*, narrative: *"As an operator who has graduated a router from E to PE mode... I want the router to actually dial..."*). Nothing in its scope, its `depends_on`, or its `blocks` list names the access-node side at all. `S-BL.RESYNC-FRAME-placement-note.md` §3.2's repository-wide grep (zero `net.Dial`/`net.Listen` for access-mode data-plane, confirmed again in this note's own re-verification of `access.go`) is the direct evidence this leg remains unbuilt.
- **A material improvement available now that was not available to PE-CONNECTOR:** PE-CONNECTOR shipped 2026-07-08 (merge SHA `8eb54a5`); `BC-2.01.008` v1.3 (NODE_IDENTIFY, the client-admittable handshake) landed 2026-07-15 — **after** PE-CONNECTOR merged. PE-CONNECTOR's admission deferral was a real constraint of its own timing, not a considered rejection of doing real admission. `S-BL.ACCESS-CONNECTOR`, scoped now, has no equivalent excuse — the server-side handshake primitives it needs already exist in production (`node_identify_wire.go`), and it should do genuine admission rather than repeat PE-CONNECTOR's zero-envelope shortcut. This is the single largest scope difference from the PE-CONNECTOR precedent, and the main driver of this story's points estimate sitting above PE-CONNECTOR's own 8.

---

## 3. `S-BL.CONSOLE-CONNECTOR`

**Purpose (proposed, subject to the caveat in §1 and below):** give the console-mode daemon a live data-plane connection to receive downstream session output and carry RESYNC emission, symmetric in intent to `S-BL.ACCESS-CONNECTOR`.

### Scope boundary — proposed, default-assumption shape (NOT a confident scope — see caveat)

If the console leg follows the same model as the access-node leg (the default architectural assumption, absent a decision otherwise): `net.Dial` to a router, client-side `NODE_IDENTIFY` handshake (identical primitives to §2), a downstream frame-read loop feeding `internal/arq`'s receive-side reorder/SACK state, and (once `S-BL.RESYNC-FRAME` Layer 1 lands) the RESYNC gap-detection/emission logic on top of that connection.

### Why this scope is NOT safely assumed today — the caveat, stated plainly

Unlike the access-node leg, there is no existing architecture text or shipped code establishing that this is the right shape for console:

1. **`runConsole` has zero data-plane scaffolding of any kind** (`cmd/switchboard/mgmt_wire.go:1192+`) — no halfchannel, no ARQ, no frame codec use, not even a deferred/stubbed field. This is a stronger absence than the access-node case (which at least constructs a `halfchannel.HalfChannel` and a `session.AccessNode`, just never wires them to the network) — there is nothing in `runConsole` to extend.
2. **`RULING-W6TB-C-console-transport.md` already made a considered, final (`status: final`) decision to move console's own control surface OFF the SVTN data-plane model** and onto the mgmt-plane Unix socket, precisely to avoid building a second protocol stack for console (*"Implementing Inv-3 as written would require building a second control-message protocol inside the SVTN data plane — a major architectural undertaking with no precedent in the codebase and no CAP/BC grounding for the required protocol design"*). That ruling is scoped to attach/detach/switch (control), not the downstream terminal-output stream — but its reasoning (avoid a second protocol surface in `sbctl`/console) is directly in tension with proposing an SVTN-wire dial-loop for console's *data*, and this note is not the right venue to adjudicate that tension.
3. **`BC-2.04.003`** ("Console Attaches to Session by Name; Receives Downstream Stream and Sends Upstream Keystrokes") is deliberately transport-agnostic — `architecture_module: internal/session`, describing only that "the console establishes a channel with the access node," never specifying the physical transport. It does not settle the question either way.

**Recommendation:** story-writer/product-owner should treat `S-BL.CONSOLE-CONNECTOR` as named but **not yet ready for normal points-estimation** — either (a) route a short architecture pre-pass (a focused question: does console session-data delivery ride the same SVTN dial-loop model as the access node, or does `RULING-W6TB-C`'s mgmt-plane precedent extend to data too, via some streaming/long-poll RPC mechanism instead?) before story-writer sizes it, or (b) accept the default assumption in this section as a working scope and size it symmetrically with `S-BL.ACCESS-CONNECTOR`, explicitly flagging the assumption as unconfirmed in the story's own frontmatter/notes. This note does not make that call.

### Non-goals (if scoped per the default assumption)

Same non-goals as `S-BL.ACCESS-CONNECTOR` §2, plus: does not reopen or re-litigate `RULING-W6TB-C`'s control-surface decision (attach/detach/switch stay mgmt-plane).

### BC/VP anchors

Would need the same class of new BC content as `S-BL.ACCESS-CONNECTOR` (§5), console-specific, plus an explicit citation resolving the tension with `RULING-W6TB-C` noted above — flagged for product-owner/architect, not drafted here pending the §3 caveat's resolution.

### Points estimate

**Not estimated** pending the §3 caveat. If the default-assumption scope is accepted as-is, this note's rough order-of-magnitude expectation is comparable to `S-BL.ACCESS-CONNECTOR` (8–13 points), since the wire-level work (dial, handshake, read loop) is structurally the same shape — only the direction of data flow and the receive-side ARQ semantics (reorder/SACK, vs. send-side `SendBuffer`) differ.

---

## 4. Dependency graph

```
S-BL.ACCESS-CONNECTOR  ──┐
                          ├──> S-BL.RESYNC-FRAME "Layer 2" (rescoped AC-005: real
S-BL.CONSOLE-CONNECTOR ──┘      two-daemon, wire-level, "no content loss" round trip)

S-BL.RESYNC-FRAME "Layer 1" (wire assemble/parse, router dispatch+relay via
routing.LookupInterface, emitter/replay pure logic against fakes)
    — no edge to either connector story —
```

**Explicit edges:**

- `S-BL.RESYNC-FRAME` Layer 2 (the true end-to-end round trip; the stub's original AC-005, rescoped) **depends_on** both `S-BL.ACCESS-CONNECTOR` (RESYNC's replay-actor, per ARCH-03's "Sender (access node): SendBuffer") and `S-BL.CONSOLE-CONNECTOR` (RESYNC's emitter, per ARCH-03's "Receiver (console)" — the console is the party that detects the reconnect gap and sends the RESYNC frame in the first place, so it also needs a live wire connection to exist and reconnect before Layer 2 can be exercised at all).
- `S-BL.RESYNC-FRAME` Layer 1 has **no dependency** on either connector story. Per `S-BL.RESYNC-FRAME-placement-note.md` §8, Layer 1's router-side dispatch/relay test injects a hand-built RESYNC frame directly into `buildRoute` and asserts relay onto a fake `sendMap` entry (mirroring `discovery_relay_wire_test.go`'s `buildRelayRouter` pattern — no live connection needed), and its emitter/replay-loop logic is tested against fake `Dispatch` callbacks (mirroring `arqsend`'s own existing unit-test seam). **Confirmed: Layer 1 is genuinely independent and can proceed in parallel with, or entirely before, either connector story**, with zero rework risk — the wire-format functions and the `routing.LookupInterface`-based relay do not change shape based on which connector eventually drives real traffic through them.

### Recommended sequencing

1. **`S-BL.ACCESS-CONNECTOR`** first — highest confidence, most directly unblocks real end-to-end verification of both `S-BL.RESYNC-FRAME` and the broader "does session data actually flow over the wire yet" question this whole investigation surfaced. Can start immediately; no blocking dependency on anything not already merged.
2. **`S-BL.RESYNC-FRAME` Layer 1** in parallel with (1) — no shared files, no dependency edge, can be staffed concurrently. (If sequenced instead of parallelized for staffing reasons, order does not matter — Layer 1 is equally buildable before or after either connector story.)
3. **Resolve the `S-BL.CONSOLE-CONNECTOR` caveat** (§3) — either the short architecture pre-pass or an explicit product-owner acceptance of the default-assumption scope — before story-writer sizes it. This can happen in parallel with (1) and (2); it only needs to land before `S-BL.CONSOLE-CONNECTOR` itself is scheduled.
4. **`S-BL.CONSOLE-CONNECTOR`** once (3) resolves.
5. **`S-BL.RESYNC-FRAME` Layer 2** (rescoped AC-005) once both connector stories merge.

---

## 5. BC/VP additions the connector story/stories would require — itemized for product-owner (not made here)

1. **New BC, access-node data-plane connection establishment** — no existing BC covers this. `BC-2.04.001`'s Precondition 3 assumes admission as a given ("The access node is admitted to an SVTN (or admission in progress)") without defining the mechanism; `BC-2.09.001` is router-graduation-specific and not reusable (title: "E Router Graduates to PE Mode..."). Candidate ID: **`BC-2.04.009`** (next free in `ss-04`, the session-access subsystem `S-BL.ACCESS-CONNECTOR` most naturally belongs to). Should define: the "connection established" postcondition (dial + client-NODE_IDENTIFY-handshake + first-frame-write, mirroring PE-CONNECTOR's Q6 three-step shape but with real admission substituted for the zero-envelope placeholder), reconnect/backoff observable behavior (EC rows mirroring `BC-2.09.001` EC-001/EC-004's shape), and how completion of this BC's postconditions discharges `BC-2.04.001` PC-3's admission precondition (a Related-BCs citation update to `BC-2.04.001`, mechanical, not a content change to it).
2. **New client-side `decodeChallenge` function is a code-level gap, not a BC gap** — noted here only because it is the concrete evidence backing the "real admission is now buildable" claim in §2.1; no BC change needed for this specific function, it is pure wire-codec implementation work within `S-BL.ACCESS-CONNECTOR`'s own scope.
3. **New VPs**, itemized for whichever story ships them:
   - Access-connector "dial + admit + first-frame-write" round trip (mirrors `VP-038`'s "E→PE via config-only... live PE mode requires dial loop" shape, but for access-node admission).
   - Reconnect preserves session/ARQ state across a torn-down connection (directly needed by `S-BL.RESYNC-FRAME`'s own open question §7.3/§9 item 6 — "does closing a connection today tear down associated ARQ/session state, or does it already survive independently" — this VP is the natural place to pin that answer down empirically once the connector exists).
   - (Console leg, once §3's caveat resolves) — the console-side analog of both VPs above.
4. **If `S-BL.CONSOLE-CONNECTOR`'s default-assumption scope is accepted**, `BC-2.04.003`'s "the console establishes a channel with the access node" language should gain a citation to whatever new BC defines that channel's actual transport (mirroring item 1's `BC-2.04.001` citation update) — mechanical, once the underlying BC exists.

None of items 1–4 are created or edited by this note. All are routed to product-owner for adjudication and drafting, per this note's scope constraint.

---

## 6. Traceability summary

| Question | Finding | Confidence | Primary citations |
|---|---|---|---|
| One story or two? | Two — access leg is well-bounded, console leg is not | High for the split itself; asymmetric confidence in the two scopes | `access.go` (has partial data-plane scaffolding) vs. `mgmt_wire.go:1192+` `runConsole` (has none); `RULING-W6TB-C-console-transport.md` |
| Access-connector scope | PE-CONNECTOR-shaped, with real admission substituted for PE-CONNECTOR's deferred zero-envelope bootstrap | High | `S-7.04-FU-PE-CONNECTOR.md` Q6, "Shipped deferral" note; `node_identify_wire.go` function inventory |
| PE-CONNECTOR relationship | Parallels the dial-loop pattern; reuses wire-codec functions; does not reuse `internal/upstreamdial` itself; genuinely did not cover the access-node leg | Definitive | `S-7.04-FU-PE-CONNECTOR.md` narrative + Design Constraints; merge date 2026-07-08 vs. BC-2.01.008 v1.3 date 2026-07-15 |
| Console-connector scope | Named but not confidently scoped; recommend an architecture pre-step before sizing | Medium (the uncertainty itself is high-confidence; the resolution is not this note's call) | `mgmt_wire.go` `runConsole`; `RULING-W6TB-C-console-transport.md`; `BC-2.04.003` (transport-agnostic) |
| RESYNC Layer 1 independence | Genuinely parallel-safe, zero dependency on either connector story | High | `S-BL.RESYNC-FRAME-placement-note.md` §8; `discovery_relay_wire_test.go` fake-router pattern precedent |
