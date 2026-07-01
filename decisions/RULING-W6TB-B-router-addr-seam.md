---
artifact_id: RULING-W6TB-B-router-addr-seam
document_type: decision
level: ops
version: "1.0"
status: final
producer: architect
timestamp: 2026-07-01T00:00:00
updated: 2026-07-01T00:00:00
cycle: v1.0.0-greenfield
stories_in_scope: [S-BL.ROUTER-ADDR, S-BL.PATH-TRACKER-WIRING]
closes_findings: [DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER]
---

# Ruling W6TB-B — S-BL.ROUTER-ADDR Resolution Seam

**Question:** where and when does `router_addr` become known in the routing
subsystem, and can S-BL.ROUTER-ADDR ship an end-to-end observable without
depending on S-BL.PATH-TRACKER-WIRING (Wave-7 backlog)?

---

## Decision

**S-BL.ROUTER-ADDR is a unit-scope story (Option A). It adds `RouterAddr string`
to `PathSnapshot`, populates it at `PathTracker` construction time via a new
`NewPathTrackerWithAddr(addr string, ...)` constructor, and enriches the
`internal/metrics.PathsList` handler to pass the stored addr through
`PathEntryFromSnapshot`. End-to-end observability (non-empty `router_addr` from
`sbctl paths list` against a running daemon) is NOT achievable in this story and
is deferred to S-BL.PATH-TRACKER-WIRING.**

The `blocks: [S-BL.PATH-TRACKER-WIRING]` frontmatter on S-BL.ROUTER-ADDR is
correct: this story must land before PATH-TRACKER-WIRING (which wires production
`PathTracker` instances to the routing registry and supplies real `host:port`
values). The story does not depend on PATH-TRACKER-WIRING.

---

## Rationale

### 1. PathSnapshot has no RouterAddr field (current state)

Post-S-W5.04, `internal/paths.PathSnapshot` (paths.go) contains:
`EWMARTTMs`, `LossPct`, `Active`, `Degraded`, `P99RTTMs`, `SampleCount`.
There is no `RouterAddr` field. The `PathsList` handler in
`internal/metrics/handlers.go` line 65–67 hard-codes the empty string:

```go
// router_addr: "" — interim per BC-2.06.003 v1.9; PathSnapshot enrichment
// tracked in S-BL.ROUTER-ADDR (DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER).
entries = append(entries, PathEntryFromSnapshot(pathID, "", snap))
```

### 2. Where router_addr becomes known

`router_addr` is a property of the logical path, not a derived metric. It is
the remote router's `host:port` — the network coordinate the access node used
to establish the connection. This address is known at path construction time
(when the connection is dialed), not at probe time. The correct carrier is
`PathTracker`, not `PathSnapshot` directly, because `PathTracker` owns the
per-path state that generates snapshots.

**Architectural seam:** `PathTracker` is constructed in `internal/paths` (pure
core). The remote router address is established at the networking layer. The
bridge is the constructor: the caller (routing layer) passes `addr string` into
`NewPathTrackerWithAddr`. The tracker stores it; `Snapshot()` copies it into
`PathSnapshot.RouterAddr`. The pure-core boundary is preserved — the tracker
stores a string, does no I/O, makes no assertions about the address format.

### 3. Why end-to-end observability requires PATH-TRACKER-WIRING

In production, `PathTracker` instances are created by the routing subsystem when
paths to routers are established. That wiring point lives in Wave-7's
S-BL.PATH-TRACKER-WIRING. Until that story lands, the only `PathTracker`
construction sites in production code are either:

- Not yet wired to the mgmt-plane metrics source (the `PathsListSource` interface
  is backed by a no-op or test stub in development mode), or
- In tests only.

S-BL.ROUTER-ADDR cannot supply real `host:port` values through `sbctl paths list`
against a running daemon because no production `PathTracker` is registered with
the `pathTrackerSource` that `PathsList` queries. The interim empty-string
(`router_addr: ""`) remains correct for any production `sbctl paths list` call
until PATH-TRACKER-WIRING lands.

This means S-BL.ROUTER-ADDR is testable only at the unit level: construct a
`PathTracker` with a known addr, call `Snapshot()`, verify `RouterAddr` is
populated; construct a `PathsListSource` stub with that tracker; call `PathsList`;
verify the JSON output includes the non-empty `router_addr`. All tests pass
without PATH-TRACKER-WIRING.

### 4. Option B (reorder PATH-TRACKER-WIRING first/jointly) is rejected

Option B would require pulling forward S-BL.PATH-TRACKER-WIRING, a Wave-7 story.
That story requires the routing registry integration (routing subsystem wiring to
`internal/mgmt`), which is out of scope for Wave 6 and carries significant scope
risk. The Tranche A Ruling-1 and BC-2.06.003 v1.9 explicitly permit `router_addr:
""` as an interim sentinel until S-BL.ROUTER-ADDR ships. There is no Wave-6
convergence obligation to produce end-to-end observable `router_addr` values; the
obligation is only that `PathSnapshot` gains the field and the JSON shape is correct
when a `PathTracker` is constructed with an addr.

The `blocks:` relationship in the story frontmatter is a scheduling constraint, not
a functional dependency: PATH-TRACKER-WIRING is more valuable once RouterAddr is a
first-class field on PathSnapshot. It does not need to precede this story.

---

## Architectural Decisions

### RouterAddr lives on PathSnapshot (not on PathEntry)

The enrichment point is `PathSnapshot.RouterAddr string`. This keeps the pure-core
boundary clean: `PathTracker` stores the addr; `Snapshot()` copies it; the metrics
handler reads it. No new package dependencies are introduced.

Alternative rejected: storing `router_addr` only in `PathEntry` (the metrics layer)
would require the metrics handler to independently resolve the addr from an external
source (a routing table or connection registry), which would cross the pure-core
boundary of `internal/metrics`.

### Constructor variant: `NewPathTrackerWithAddr`

Add a new constructor alongside `NewPathTracker`:

```go
func NewPathTrackerWithAddr(addr string, initialRTTMS float64, alpha float64) *PathTracker
```

The existing `NewPathTracker` is preserved unchanged for callers that do not yet
have an addr (test-only sites, PATH-TRACKER-WIRING will migrate to the new
constructor). This avoids a breaking change in S-BL.ROUTER-ADDR's scope.

Alternative rejected: adding `addr string` as a parameter to the existing
`NewPathTracker` signature would be a breaking change requiring updates to all
existing `NewPathTracker` call sites (tests in S-W5.04 scope).

### When RouterAddr is set

`RouterAddr` is set at tracker construction time and is immutable thereafter
(a path's remote endpoint does not change during its lifetime). `Snapshot()`
copies it verbatim. No locking concern: the field is written once before the
tracker is shared and is thereafter read-only.

---

## Implications for S-BL.ROUTER-ADDR AC Set

When the story is fleshed out of backlog stub status, the AC set MUST include:

| AC | Description |
|----|-------------|
| AC-001 | `PathSnapshot.RouterAddr string` field added; `Snapshot()` copies it from the tracker |
| AC-002 | `NewPathTrackerWithAddr(addr, initialRTTMS, alpha)` constructor stores addr; `NewPathTracker` is unchanged |
| AC-003 | `PathsList` passes `snap.RouterAddr` (not `""`) to `PathEntryFromSnapshot` when the snap carries a non-empty addr |
| AC-004 | Unit test: `PathTracker` constructed with `addr="127.0.0.1:9000"` → `Snapshot().RouterAddr == "127.0.0.1:9000"` |
| AC-005 | Unit test: `PathsList` with a stub source returning a snapshot with `RouterAddr="127.0.0.1:9000"` → JSON output `"router_addr":"127.0.0.1:9000"` |
| AC-006 | DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER annotation in BC-2.06.003 PC-1 updated: replace "PathSnapshot enrichment tracked in follow-on story" with the permanent note that `router_addr` is populated from `PathSnapshot.RouterAddr`; remove the sentinel-permission clause for `""` (no longer permitted from a PathTracker with a set addr) |
| AC-007 | `NewPathTracker` (addr-less) continues to produce `PathSnapshot.RouterAddr == ""`; existing tests unaffected |

**Estimated points:** 2 (small — pure Go struct field, one constructor variant,
one handler line change, unit tests only).

**Wave scheduling:** Wave 7, before S-BL.PATH-TRACKER-WIRING. No Wave-6 hard
dependency (Ruling-1 permits `""` through Wave-6 convergence).

---

## Observable Coverage Matrix

| Test scope | Achievable in S-BL.ROUTER-ADDR | Requires PATH-TRACKER-WIRING |
|-----------|-------------------------------|------------------------------|
| `PathSnapshot.RouterAddr` field populated | Yes (unit) | No |
| `PathsList` JSON `router_addr` non-empty | Yes (unit with stub source) | No |
| `sbctl paths list` returns non-empty `router_addr` (live daemon) | No | Yes |
| VP-047 AC-006 `router_addr` field assertion updated | Yes (assert field present and equals stub addr) | No |
| BC-2.06.003 PC-1 `""` sentinel annotation removed | Yes | No |

---

## Decision Log

| Date | Actor | Entry |
|------|-------|-------|
| 2026-07-01 | architect | Initial ruling: unit-scope story (Option A). RouterAddr added to PathSnapshot; populated at construction via NewPathTrackerWithAddr; metrics handler passes it through. End-to-end observability deferred to PATH-TRACKER-WIRING. Estimated 2 points, Wave 7. BC-2.06.003 DRIFT annotation removed when story ships. |
