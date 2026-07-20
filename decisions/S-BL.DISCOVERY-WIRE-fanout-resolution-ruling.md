---
artifact_id: S-BL.DISCOVERY-WIRE-fanout-resolution-ruling
document_type: decision
level: ops
version: "1.0"
status: final
producer: architect
timestamp: 2026-07-19T00:00:00Z
cycle: cycle-1
stories_in_scope: [S-BL.DISCOVERY-WIRE]
bc_traces:
  - BC-2.03.001
  - BC-2.01.008
  - BC-2.01.010
related_docs:
  - decisions/S-BL.DISCOVERY-WIRE-rulings.md
  - decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md
  - stories/S-BL.NODE-IDENTIFY-WIRE.md
---

# Ruling: S-BL.DISCOVERY-WIRE — SVTN-Scoped Connection-Enumeration Primitive for Task 6 Hop-2 Fan-Out

All factual claims below are grep/read-verified against the merged develop branch at commit
`7fcf0cf` (HEAD at dispatch). File:symbol anchors cited per TD-031 (symbol-only form, no
line numbers) except where a precise offset is load-bearing.

This ruling resolves the five open implementation-design questions that block Task 6 of
`S-BL.DISCOVERY-WIRE` (AC-017 / AC-018 hop-2 fan-out dispatch) from being fully specified
for implementation. It does NOT modify any story file, behavioral contract, STATE.md, or
STORY-INDEX — those downstream edits are flagged at the end and owned by story-writer/PO.

---

## Verified Premises

| # | Premise | File:Symbol | Evidence |
|---|---|---|---|
| V1 | `identityIfaceMap map[[16]byte]map[[8]byte]InterfaceID` lives on `Router` in `internal/routing/identity.go`, protected by `r.mu`, populated by `BindInterface` on a successful NODE_IDENTIFY handshake | `internal/routing/identity.go`, `BindInterface` | Shipped PR #127 @ `7fcf0cf`; direct code read |
| V2 | `LookupInterface(svtnID [16]byte, nodeAddr [8]byte) (InterfaceID, bool)` exists as a point read under `r.mu.RLock`; returns `(InterfaceID, bool)` value type per go.md rule 12 | `internal/routing/identity.go`, `LookupInterface` | Same PR |
| V3 | `UnbindInterface(svtnID [16]byte, nodeAddr [8]byte, callerIfaceID InterfaceID)` has a stale-cleanup guard: suppresses the delete when stored IfaceID != callerIfaceID (prevents a LWW-overwritten binding from being removed by the prior connection's cleanup) | `internal/routing/identity.go`, `UnbindInterface` | Same PR; BC-2.01.010 PC-9 |
| V4 | `sendMap sync.Map // routing.InterfaceID -> *nodeConn` is LOCAL to `runRouter` in `cmd/switchboard/mgmt_wire.go` — declared inside the function body, not a field on `Router` or any exported type | `cmd/switchboard/mgmt_wire.go`, `runRouter` (local `var sendMap sync.Map`) | Direct read; `mgmt_wire.go:592` comment "Per-node send map (Q-SEAM)" |
| V5 | `nodeConn` is a `cmd/switchboard`-package-private type; `internal/routing` importing `cmd/switchboard` is a cycle — ARCH-08 position 18 vs. position 5 | `cmd/switchboard/mgmt_wire.go`, `nodeConn`; ARCH-08 §Import DAG | Import direction verified: position 5 cannot import position 18 |
| V6 | `nc.send` is NEVER closed (per `nodeConn` doc comment: "Done IS closed exactly once, by the writer goroutine... send is NEVER closed — prevents panic in the drain observer's concurrent sendMap.Range send") | `cmd/switchboard/mgmt_wire.go`, `nodeConn` struct doc | Direct read; same comment in the DRAIN-observer context |
| V7 | `AdmittedKeySet.ListBySVTN(svtnID [16]byte) []AdmittedKey` exists; returns a value-copy snapshot under `s.mu` read lock | `internal/admission/admission.go`, `ListBySVTN` | Direct read |
| V8 | `AdmittedKeySet.AllSVTNEntries() map[[16]byte][]AdmittedKey` exists; returns a snapshot of all admitted key entries grouped by SVTN ID | `internal/admission/admission.go`, `AllSVTNEntries` | Direct read |
| V9 | `wireAdmissionSyncHandlers` IS called from `runRouter` at Phase (c3), before `serveMgmtServer` | `cmd/switchboard/mgmt_wire.go`, `runRouter` Phase c3 | Direct read confirmed at line ~557 |
| V10 | `wireDiscoveryListener` is NOT called from `runRouter` at `7fcf0cf`; `discovery_wire.go`'s own package doc still contains the original deferral comment: "Wiring `wireDiscoveryListener` into `runRouter`'s daemon lifecycle is therefore left to a follow-on story once an SVTN-admission-event source exists" | `cmd/switchboard/discovery_wire.go` package doc | Grep of `cmd/switchboard/mgmt_wire.go` for `wireDiscovery` — no hit; comment text read directly |
| V11 | S-BL.NODE-ADMISSION-PROVISIONING (PR #125 @ `ce06f6a`) delivered `loadOrGenerateAdmissionKeypair` + `runAccessWithConnector` calling `d.Run(runCtx)` in a goroutine | `.factory/stories/S-BL.NODE-IDENTIFY-WIRE.md`, "Previous Story Intelligence" S-BL.NODE-ADMISSION-PROVISIONING row | Story v1.13 direct read; git log confirms merge |
| V12 | S-BL.NODE-IDENTIFY-WIRE story is `status: ready`, `depends_on: [S-BL.ADMISSION-SYNC-WIRE, S-BL.NODE-ADMISSION-PROVISIONING]`, with "Both prerequisite stories delivered" stated in narrative | `.factory/stories/S-BL.NODE-IDENTIFY-WIRE.md` frontmatter + narrative | Direct read |
| V13 | `InterfaceID` is a value type used as a sync.Map key and return type in the existing `LookupInterface` | `internal/routing/routing.go`, `InterfaceID` type | Confirmed: `LookupInterface` already returns it by value; it is a scalar type |

---

## Decision 1 — Primitive Signature and Home

**RULING:** The routing layer provides SVTN-scoped interface-ID enumeration. The
`cmd/switchboard` layer maps those IDs through `sendMap` to live `*nodeConn` pointers and
drives the best-effort non-blocking send. Neither layer imports the other's private types.

### Routing-layer method (new)

**Home:** `internal/routing/identity.go` — alongside the already-shipped `BindInterface`,
`LookupInterface`, and `UnbindInterface` methods. No new file. No new imports needed
(same map, same lock, same type vocabulary already in scope).

**Signature:**
```go
// InterfacesForSVTN returns a snapshot of all InterfaceIDs currently bound
// to nodes admitted to svtnID, EXCLUDING the node identified by excludeNodeAddr
// (the originating advertisement sender — satisfies AC-017's exclude-originator
// requirement without requiring the caller to do a post-filter round-trip).
//
// The snapshot is taken under r.mu read lock; the returned []InterfaceID is
// freshly allocated and caller-owned, with no aliasing into internal state
// (go.md rule 12). Callers map each returned ID through sendMap to obtain the
// live *nodeConn; a missing sendMap entry (connection closed between snapshot
// and send) is a silent skip, consistent with AC-017's best-effort delivery
// semantics.
//
// Returns a non-nil, empty slice when svtnID has no bindings or all bindings
// belong to excludeNodeAddr. Never returns nil.
//
// Traces to S-BL.DISCOVERY-WIRE AC-017; BC-2.03.001 PC-1 delivery-mechanism
// note; BC-2.01.010 (identityIfaceMap lifecycle).
func (r *Router) InterfacesForSVTN(svtnID [16]byte, excludeNodeAddr [8]byte) []InterfaceID {
    r.mu.RLock()
    defer r.mu.RUnlock()

    inner, ok := r.identityIfaceMap[svtnID]
    if !ok {
        return []InterfaceID{}
    }
    out := make([]InterfaceID, 0, len(inner))
    for nodeAddr, ifaceID := range inner {
        if nodeAddr == excludeNodeAddr {
            continue
        }
        out = append(out, ifaceID)
    }
    return out
}
```

**Return type rationale:** `[]InterfaceID` is a freshly-allocated value slice under
the read lock. `InterfaceID` is a value type (V13). No pointer into internal state; no
aliasing. Satisfies go.md rule 12 ("Never return internal pointers from a locked accessor")
by the same construction `LookupInterface` already uses.

### cmd/switchboard-layer resolver (inline closure — see Decision 3)

The `cmd/switchboard` layer's fan-out step, written inline in `runRouter`:
```go
// Call after router.InterfacesForSVTN; r.mu is NOT held here.
ifaceIDs := router.InterfacesForSVTN(svtnID, originNodeAddr)
for _, ifaceID := range ifaceIDs {
    val, ok := sendMap.Load(ifaceID)
    if !ok {
        continue // connection gone between snapshot and send — silent skip
    }
    nc := val.(*nodeConn)
    select {
    case nc.send <- relayFrame:
    default:
    }
}
```

**Import-direction constraint (ARCH-08):** `internal/routing` (position 5) returns
`[]InterfaceID` — a type it already owns. No `cmd/switchboard` type crosses into routing.
The cmd layer owns sendMap lookup and the `*nodeConn` send; nothing in `internal/routing`
sees `nodeConn` or `sendMap`. The DAG is unmodified by this addition. Verified: no new
imports required in either layer.

---

## Decision 2 — Locking, TOCTOU Window, and Missing-Entry Semantics

**RULING:** The snapshot lock scope is confined to `InterfacesForSVTN`'s own body.
The TOCTOU window between snapshot and `sendMap.Load` is accepted. Missing sendMap
entries are silent skips.

### Lock scope

`InterfacesForSVTN` holds `r.mu.RLock()` for the duration of the map scan and slice
construction, releasing it (via `defer`) before returning. `r.mu` is NOT held during the
caller's `sendMap.Load` loop. Reason: `sendMap` is a `sync.Map` — independently
concurrent-safe; no cross-lock pairing needed. Taking `r.mu` into the sendMap lookup
would create a hold-two-locks sequence with no established lock-order elsewhere in the
codebase, and provides no correctness benefit since the snapshot is complete and
self-consistent at return time.

### TOCTOU window — accepted

Between `InterfacesForSVTN` returning and the caller's `sendMap.Load` for each IfaceID,
a connection may:

1. **Close (`sendMap.Delete` + `UnbindInterface`).** `sendMap.Load(ifaceID)` returns
   `(nil, false)`. The resolver checks `ok` and silently skips. No send attempted.
   Correct behavior: best-effort means a closing connection simply misses this relay.

2. **LWW overwrite (reconnect — same NodeAddr, new TCP, new IfaceID).** The snapshot
   holds the OLD IfaceID. `sendMap.Load(oldIfaceID)` returns either the old `*nodeConn`
   (cleanup not yet fired — best-effort send or silent drop) or `(nil, false)` (cleanup
   fired — silent skip). The NEW IfaceID is not in the snapshot; the reconnected node
   receives the next advertisement. Both outcomes are within AC-017's best-effort envelope.

3. **New connection added after snapshot.** Omitted from snapshot; receives future relays.
   Consistent with best-effort.

The stale-cleanup guard on `UnbindInterface` (V3 — only deletes if stored IfaceID ==
callerIfaceID) ensures `identityIfaceMap` at all times reflects the latest live binding
or is absent (never stale from a prior connection). The snapshot therefore reads a
consistent state; the TOCTOU window is a bounded race only against the send step,
not against the identity map's correctness.

### nc.send is never closed — no panic risk

Per V6, `nc.send` is never closed by design. The `select { case nc.send <- frame: default: }`
pattern cannot panic regardless of whether the writer goroutine is still running. The
`default` branch fires silently when the channel is full or the writer has exited. This is
the identical shape the DRAIN observer already uses in production.

### Missing-entry ruling

A `sendMap.Load` returning `(nil, false)` for an IfaceID from the snapshot is a **silent
skip** — the relay is not sent to that connection. No log, no counter, no error return.
This matches AC-017's best-effort delivery requirement and the DRAIN observer's own
precedent for absent entries.

---

## Decision 3 — Resolver Seam Wiring

**RULING:** Direct closure in `runRouter`, NOT a package-level var.

The DRAIN observer (the structural precedent for hop-2 fan-out dispatch) is a direct closure
in `runRouter` that captures `sendMap` and `drainCoord` by reference. It has no package-var
injection seam. AC-017/AC-018 integration tests exercise the same production path through
`net.Pipe` pairs or `internal/testenv`-wired in-process connections.

`nodeIdentifyHandshakeFn` is a package var for a specific reason that does NOT apply here:
the handshake fires synchronously in `onAccept` on a bare `net.Conn`, and unit tests need
a mock that avoids a full three-message wire roundtrip. The fan-out dispatch is triggered
asynchronously (from the relay-ingest path, not blocking `onAccept`) and is exercisable
through real in-process connections — no mock substitution is needed or desirable.

**Concrete placement:** The relay-dispatch closure is declared inline in `runRouter`,
alongside the DRAIN observer and the `onAccept` closure. It captures:
- `router` (for `router.InterfacesForSVTN`) — already in scope in `runRouter`
- `sendMap` (for `sendMap.Load`) — already in scope in `runRouter`

It is NOT assigned to a package-level `var`. Test code for `discovery_relay_wire_test.go`
drives the full production path using `net.Pipe` connection pairs wired through
`runRouter`'s accept loop, precisely as `cmd/switchboard/node_identify_wire_test.go`
already does for the NODE_IDENTIFY handshake.

---

## Decision 4 — Scope Placement

**RULING (recommendation — orchestrator/user confirms):** Amend Task 6 within
`S-BL.DISCOVERY-WIRE` to include `InterfacesForSVTN` as an explicit sub-step. Do NOT
create a separate story.

### Rationale

`InterfacesForSVTN` is one new method on `Router` in `internal/routing/identity.go`
alongside three already-shipped methods on the same map. Estimated effort: ~25 lines of
implementation + ~35 lines of unit tests. This is a task decomposition item, not a story.

The carve-out precedent (`S-BL.NODE-IDENTIFY-WIRE`) was justified by 10 points of new
wire protocol: a multi-message codec, a handshake driver with 10-second deadline, a
challenge-response exchange, and a new keyed map with three methods and full test coverage
for 13 ACs. `InterfacesForSVTN` is a single range-based read on an existing map guarded
by an existing lock, with no new imports and no new wire behavior.

### Recommended Task 6 decomposition

Story-writer should replace the current monolithic Task 6 with:

- **Task 6a:** Add `Router.InterfacesForSVTN(svtnID [16]byte, excludeNodeAddr [8]byte) []InterfaceID`
  to `internal/routing/identity.go`. Unit tests in `internal/routing/identity_test.go`
  alongside the existing Bind/Lookup/Unbind tests: returns correct IDs; excludes originator;
  empty SVTN returns non-nil empty slice; `go test -race` with concurrent Bind/Unbind passes.
- **Task 6b:** Implement the hop-2 relay-dispatch closure inline in `runRouter` (not a
  package var). Calls `InterfacesForSVTN` + `sendMap.Load` + best-effort non-blocking send.
  Integration tests in `cmd/switchboard/discovery_relay_wire_test.go`: router with two-plus
  admitted nodes; trigger relay ingest; assert relay frame arrives at non-originator channels;
  assert originator channel does NOT receive its own advertisement.
- **Task 6c (AC-018):** Implement the `~1/sec` per-`(SVTNID, NodeAddr)` relay-rate-cap map
  at the dispatch-decision point (Ruling 3(e), SEC-DW-09): silent drop on excess-rate
  arrivals, visibility counter only, never a gate. Separate acceptance criterion with its
  own test.

This is a recommendation to story-writer. It is NOT a blocking gate — story-writer may
decompose differently as long as all three behaviors are covered by ACs and tests.

---

## Decision 5 — Forward Obligation (e)/(f) Reconciliation

### Forward Obligation (e)

**RULING: INFRASTRUCTURE-RESOLVED. Wiring residual remains, unblocked.**

**Infrastructure gap (named in Ruling 4):** "The router process has no source of 'which
SVTN(s) am I serving' — `admission.AdmittedKeySet` has no SVTN-enumeration method; the
only production `RegisterKey` caller runs in control-mode, a separate, disconnected OS
process from router-mode."

**Status at `7fcf0cf`:** RESOLVED.

- `wireAdmissionSyncHandlers` is called from `runRouter` Phase (c3) (V9). The router's
  `AdmittedKeySet` is now populated via control→router push (four `internal.admission.*`
  RPCs: register, revoke, expire, remove-svtn).
- `AdmittedKeySet.AllSVTNEntries()` (V8) and `ListBySVTN()` (V7) now exist. The router
  can enumerate its admitted SVTNs at startup.
- The `discovery_wire.go` comment's stated condition — "left to a follow-on story once an
  SVTN-admission-event source exists" — is NOW MET. The admission-event source (push via
  `wireAdmissionSyncHandlers`) has shipped.

**Residual:** `wireDiscoveryListener` is still not called from `runRouter` (V10). The
production code has not yet added the startup loop that iterates admitted SVTNs and calls
`wireDiscoveryListener` for each. This residual is no longer blocked by any missing
infrastructure component. It is deliverable today.

**Recommended placement:** Add the `wireDiscoveryListener` wiring call as a sub-task of
Task 6's delivery (e.g., Task 6d), or as a named follow-on immediately after Task 6's
merge. The loop shape:
```go
for svtnID := range routerKS.AllSVTNEntries() { // snapshot at startup
    svtnID := svtnID // capture for goroutine
    wg.Add(1)
    go wireDiscoveryListener(ingressCtx, &wg, svtnID, ri, w)
}
```
where `ri` is the `RouterIngest` instance the relay-dispatch closure feeds. This is
Task 3's original stated scope. The `discovery_wire.go` doc comment should be updated
when this call lands.

**Story-writer action (not executed here):** Update Forward Obligations table row (e)
status to "INFRASTRUCTURE RESOLVED — `wireDiscoveryListener` wiring call pending;
deliverable in Task 6d or immediate follow-on; no longer blocked."

### Forward Obligation (f)

**RULING: CLOSED.**

S-BL.NODE-ADMISSION-PROVISIONING (PR #125 @ `ce06f6a`) delivered both facets that
Forward Obligation (f) named (Ruling 5, both independently discovered compounding gaps):

1. **Node-side admission keypair provisioning:** `loadOrGenerateAdmissionKeypair`
   generates/loads the access node's stable Ed25519 private key; the public key populates
   `discovery.Config.LocalNodeAdmissionPubkey`. `internal/config.Config` now has an
   admission-keypair field. Closes: "no production code path anywhere supplies a running
   access-node process with its own admission keypair."

2. **`Discovery.Run()` daemon-lifecycle wiring into `runAccess`:** `runAccessWithConnector`
   calls `d.Run(runCtx)` in a goroutine. Closes: "`internal/discovery.New`/`Discovery.Run`
   have zero production callers anywhere in the repository."

Both are confirmed from the S-BL.NODE-IDENTIFY-WIRE story's "Previous Story Intelligence"
section (V11), which cites PR #125 explicitly for both deliveries.

**Story-writer action (not executed here):** Update Forward Obligations table row (f)
status to "CLOSED — S-BL.NODE-ADMISSION-PROVISIONING (PR #125 @ ce06f6a)."

---

## Implementation Checklist for Task 6 Delivery

For the implementer and test-writer executing S-BL.DISCOVERY-WIRE Task 6 / AC-017 / AC-018:

### Phase 1 — Routing-layer primitive (Task 6a)

- [ ] Add `Router.InterfacesForSVTN(svtnID [16]byte, excludeNodeAddr [8]byte) []InterfaceID`
      to `internal/routing/identity.go`. No new imports.
- [ ] Lock discipline: `r.mu.RLock()` / `defer r.mu.RUnlock()`. Snapshot ONLY under the
      lock. Lock is released before returning the slice — do NOT hold r.mu across the
      caller's sendMap access.
- [ ] Return type: `[]InterfaceID` (value slice, freshly allocated, never nil). Empty SVTN:
      `return []InterfaceID{}`.
- [ ] Unit tests in `internal/routing/identity_test.go`:
  - [ ] Returns all IfaceIDs for svtnID, correctly excluding excludeNodeAddr.
  - [ ] Returns non-nil empty slice (len=0) when svtnID has no bindings.
  - [ ] Returns non-nil empty slice when only the excluded NodeAddr is bound.
  - [ ] Returns non-nil empty slice when svtnID is absent from identityIfaceMap.
  - [ ] `go test -race`: concurrent Bind/Unbind/InterfacesForSVTN on same svtnID passes.
- [ ] Confirm `go list -deps ./internal/routing` shows no new cmd/ imports.

### Phase 2 — Discovery listener wiring (Task 3 residual, Forward Obligation (e))

- [ ] Add `wireDiscoveryListener` call loop to `runRouter` startup: iterate
      `routerKS.AllSVTNEntries()` snapshot; for each svtnID, `wg.Add(1); go wireDiscoveryListener(...)`.
      Wire in the same register-before-serve / `wg`-tracked pattern as `wireMetricsHandlers`.
- [ ] Update `discovery_wire.go` package-doc deferral comment: remove "left to a follow-on
      story once an SVTN-admission-event source exists" qualifier (condition now met).
- [ ] Integration test: verify that `runRouter` with a populated `AdmittedKeySet` (at least
      one registered SVTN) joins the corresponding multicast group — extend or reference the
      existing `TestRunRouter_DiscoveryListener_JoinsGroup_RouterModeOnly` test.

### Phase 3 — Fan-out dispatch closure (Task 6b)

- [ ] Implement relay-dispatch closure inline in `runRouter`. NOT a package var.
- [ ] Captures `router` (for `InterfacesForSVTN`) and `sendMap` (for `sendMap.Load`).
- [ ] Call chain: `router.InterfacesForSVTN(svtnID, originNodeAddr)` → range over
      `[]InterfaceID` → `sendMap.Load(ifaceID)` → if `!ok` silent skip → assert
      `val.(*nodeConn)` → `select { case nc.send <- relayFrame: default: }`.
- [ ] Relay frame: `FrameTypeCtl` + `control_type=0x03` (DISCOVERY_RELAY), payload per
      Ruling 3(c) v1.5 layout (22-byte fixed header: control_type=0x03 | version=0x01 |
      reserved=0x0000 | NodeAddr[4:12] | Sequence uint64 BE[12:20] | count uint16[20:22] |
      sessions[22:]).
- [ ] Integration tests in `cmd/switchboard/discovery_relay_wire_test.go`:
  - [ ] Two admitted nodes (A, B) on same SVTN; A sends advertisement; router relays to B;
        B's send channel receives the relay frame.
  - [ ] Originator exclusion: A's advertisement is NOT relayed back to A's send channel.
  - [ ] Three-node test: A sends; B and C receive; A does not.
  - [ ] Missing sendMap entry: IfaceID in identityIfaceMap but absent from sendMap (simulate
        closed connection) → no panic; relay continues to remaining targets.
  - [ ] `go test -race` passes.

### Phase 4 — Relay rate cap (Task 6c, AC-018)

- [ ] Implement `~1/sec` per-`(SVTNID, NodeAddr)` rate-cap map at the relay-dispatch
      decision point (Ruling 3(e), SEC-DW-09). Silent drop on excess-rate arrivals.
- [ ] Rate cap is the relay-amplification cap — distinct from the hop-1 ingest SEC-DW-03
      token-bucket (different enforcement point, different purpose).
- [ ] Visibility counter follows the SEC-DW-03 philosophy: non-gating, `FailureCounter`-style,
      threshold-crossing emission only. Never promotes to a rate-gate itself.
- [ ] Unit test: two arrivals within 1 second from the same `(svtnID, nodeAddr)` — first
      relayed, second silently dropped; no panic, no error return.

### ARCH-08 compliance

- [ ] `internal/routing` (position 5): no new imports. Verified via `go list -deps ./internal/routing`.
- [ ] `cmd/switchboard` (position 18): already imports `internal/routing`; `discovery_relay_wire.go`
      already imports `internal/discovery` (position 14). No new imports needed.
- [ ] `internal/routing/identity.go` adds no new `internal/admission`, `internal/hmac`, or
      `cmd/*` imports — none are needed. `InterfacesForSVTN` works only on `identityIfaceMap`
      (already in scope as a `Router` field).

---

## Downstream Touch-List (story-writer / PO — not executed here)

| Artifact | Change | Owner |
|---|---|---|
| `.factory/stories/S-BL.DISCOVERY-WIRE.md` | Decompose Task 6 into 6a (InterfacesForSVTN), 6b (fan-out closure), 6c (rate cap), 6d (wireDiscoveryListener-from-runRouter); update Forward Obligations table row (e) to "INFRASTRUCTURE RESOLVED — wiring call pending, deliverable in Task 6d"; update row (f) to "CLOSED — S-BL.NODE-ADMISSION-PROVISIONING (PR #125 @ ce06f6a)" | story-writer |
| `cmd/switchboard/discovery_wire.go` | Update package-doc deferral comment when the `wireDiscoveryListener` call is added to `runRouter` (Phase 2 of the checklist above) | implementer (at delivery time) |

---

## Decision Log

| Date | Actor | Entry |
|------|-------|-------|
| 2026-07-19 (v1.0) | architect | **Initial ruling, five decisions on the SVTN-scoped connection-enumeration primitive for S-BL.DISCOVERY-WIRE Task 6 / AC-017 / AC-018.** Verified against develop @ `7fcf0cf`. **(1) Primitive and home:** `Router.InterfacesForSVTN(svtnID [16]byte, excludeNodeAddr [8]byte) []InterfaceID` in `internal/routing/identity.go`; returns freshly-allocated `[]InterfaceID` value-snapshot under `r.mu.RLock`; excludes originatorNodeAddr at routing layer. cmd/switchboard layer maps returned IfaceIDs through local `sendMap` to `*nodeConn` and applies best-effort non-blocking send. ARCH-08 DAG unmodified — no new imports in either layer. **(2) Locking:** r.mu.RLock held ONLY for the snapshot; released before any sendMap access. TOCTOU window accepted: missing sendMap entry = silent skip (connection closed between snapshot and send); LWW-overwritten IfaceID = best-effort old-send or silent skip; both within AC-017's best-effort semantics. nc.send is never closed (V6) — `select { case nc.send <- frame: default: }` cannot panic. **(3) Resolver seam:** direct closure in `runRouter` (DRAIN-observer pattern), NOT a package var. nodeIdentifyHandshakeFn package-var pattern is inapplicable here — fan-out is asynchronous and exercisable through real in-process connections. **(4) Scope placement:** amend Task 6 in S-BL.DISCOVERY-WIRE into sub-steps 6a/6b/6c/6d; NOT a separate story. InterfacesForSVTN is one method on an existing map — task, not story. **(5) (e)/(f) reconciliation:** (e) INFRASTRUCTURE-RESOLVED by S-BL.ADMISSION-SYNC-WIRE (PR #126): wireAdmissionSyncHandlers now in runRouter (V9), AllSVTNEntries/ListBySVTN now on AdmittedKeySet (V7/V8); wireDiscoveryListener-from-runRouter call is the remaining deliverable (V10), unblocked, recommended as Task 6d. (f) CLOSED by S-BL.NODE-ADMISSION-PROVISIONING (PR #125 @ ce06f6a): both facets delivered — node keypair provisioning AND Discovery.Run() lifecycle wiring into runAccess (V11). |
