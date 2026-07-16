---
artifact_id: S-BL.ADMISSION-SYNC-WIRE-rulings
document_type: rulings
version: "1.1"
status: draft
producer: architect
timestamp: 2026-07-15T00:00:00Z
modified: 2026-07-15T00:00:00Z
related_stories:
  - S-BL.ADMISSION-SYNC-WIRE
related_architecture:
  - decisions/identity-cluster-architecture.md (v1.2, authoritative input)
  - specs/architecture/ARCH-08-dependency-graph.md (v2.13, import-boundary authority)
  - specs/architecture/ARCH-01-process-connection-model.md (goroutine WaitGroup contract)
related_code:
  - cmd/switchboard/admin_handlers.go
  - cmd/switchboard/mgmt_wire.go
  - cmd/switchboard/router_control_wire.go
  - internal/admission/admission.go
  - internal/config/config.go
  - internal/mgmt/mgmt.go
---

# S-BL.ADMISSION-SYNC-WIRE: Elaboration Rulings 1.1

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.1 | 2026-07-15 | Ruling 9 added: router TCP mgmt listener security posture resolved by human ratification; summary table updated; story confirmed decomposition-ready (POL-001). |
| 1.0 | 2026-07-15 | Initial elaboration rulings — Rulings 1–8. |

## Preamble

This document translates the mechanism ratification in
`identity-cluster-architecture.md` v1.2 (Option A push RPC + router-side
VLR-local admitted-state snapshot, ODO-5 resolved) into the precise
implementation specification a story-writer needs to produce acceptance
criteria and implementation tasks. All decisions are grounded against
`develop@d249f88` code.

**All inputs verified:**
- Factory-artifacts worktree: `541737f` (confirmed at session start)
- `identity-cluster-architecture.md`: v1.2 at `541737f`
- Code topology: `cmd/switchboard/` and `internal/` at `develop@d249f88`

---

## Ruling 1 — The Push RPC: message shape and the four write paths

### Context

Every mutation to control's `AdmittedKeySet` is via `SVTNManager`, which
wraps `AdmittedKeySet`. The four mutation surfaces reachable from admin
handlers today are:

| Admin verb | SVTNManager method | AdmittedKeySet method |
|---|---|---|
| `admin.key.register` | `RegisterKey(svtnName, pubkey, role)` | `RegisterKey(svtnID, pubkey, role)` |
| `admin.key.revoke` | `RevokeKey(svtnName, pubkey, role, confirm)` | `RevokeKeyIfRoleMatches(...)` |
| `admin.key.expire` | `ExpireKey(svtnName, pubkey, ttl)` | `SetKeyExpiryIfRoleMatches(...)` (or similar) |
| `admin.svtn.destroy` | `Destroy(svtnName, ...)` | `RemoveSVTN(svtnID)` |

Each of these, once it succeeds in control's own `AdmittedKeySet`, must also
be applied to each configured router's `AdmittedKeySet`.

### Wire shape ruling

**RULING: Use the existing mgmt JSON-over-TCP protocol. Do NOT invent a
parallel protocol.**

The existing management protocol (`internal/mgmt`) uses:
1. Ed25519 challenge-response handshake on connect (ADR-012)
2. Newline-delimited JSON for all messages
3. Per-RPC `{"type":"request","id":"...","command":"...","args":{...}}` envelope

The push connection from control to each router management endpoint uses this
same protocol. Control authenticates to the router using its own daemon private
key (`daemonPriv`, already present in `runControl`). The router's management
server already handles incoming authenticated connections — the only new element
is: (a) a new set of handler commands registered on the router-mode server, and
(b) a client that dials in using control's own private key.

**Ruling on the new command names.** Three new internal-RPC command names for
the router-side push handlers:

| Event | New command name | Args |
|---|---|---|
| RegisterKey | `internal.admission.register` | `{svtn_id, pubkey_openssh, role}` |
| RevokeKey | `internal.admission.revoke` | `{svtn_id, pubkey_openssh, role, confirm}` |
| SetKeyExpiry | `internal.admission.expire` | `{svtn_id, pubkey_openssh, after}` |
| RemoveSVTN | `internal.admission.remove-svtn` | `{svtn_id}` |

The `internal.` prefix distinguishes these from operator-facing `admin.`
commands. The router-mode server registers these four handlers; the `admin.`
handlers are never registered on the router (ADR-004 / AC-004 / ARCH-04
role-exclusion remain fully intact — the push handlers are control→router
internal replication, not operator-facing RPCs).

**Ruling on args encoding.** Use the same encoding conventions as existing
admin handlers:
- `svtn_id` — string SVTN name (the human-readable name that SVTNManager
  resolves to a `[16]byte` internally; control already has the name in the
  admin handler args; passing it through avoids a name↔ID translation step
  on the router side; the router's `SVTNManager` handles the same name-to-ID
  resolution). If the router has no SVTN by that name registered yet (a
  legitimately possible race on fresh install), the handler creates the entry
  rather than failing — `RegisterKey` must be idempotent from the router's
  perspective.
- `pubkey_openssh` — base64-encoded Ed25519 public key in OpenSSH wire format,
  identical to the existing `admin.key.register` / `admin.key.revoke` encoding.
- `role` — canonical role string ("control", "console", "access") identical to
  existing admin handlers.
- `after` — duration string (e.g., "8760h") identical to `admin.key.expire`.
- `confirm` — boolean, identical to `admin.key.revoke`.

This means control's push client can construct push-RPC args by reusing or
lightly adapting the args structs that `admin_handlers.go` already parses.

**Ruling on snapshot-vs-delta.** Per `identity-cluster-architecture.md`
Section 9 (reconnection and detachment-tolerance ruling, item 5): the
near-term story uses **full-snapshot push on every control startup** and
**per-write-delta push on each mutation**. The full snapshot on startup means
control, when it first dials a router after starting, pushes its entire current
`AdmittedKeySet` state. This is implemented as a sequence of
`internal.admission.register` calls for every `(svtn, pubkey, role)` triple in
the current snapshot, plus `internal.admission.expire` for any entries with an
active expiry. The per-write push on each mutation means each of the four write
paths issues the corresponding single-call delta push.

A `router.admission.full-snapshot` single-call bulk-push endpoint is an
optimization for later. Near-term: loop over the snapshot and issue individual
`internal.admission.register` calls. This is simpler to implement, test, and
reason about than a new bulk endpoint, and the startup push is not on any
latency-critical path.

---

## Ruling 2 — Control-side: config field, dial client, failure behavior

### Config field

**RULING: New `RouterManagementEndpoints []RouterManagementEndpoint` field in
`internal/config.Config`, structurally identical to the existing
`UpstreamRouters []UpstreamRouter` field.**

```go
// RouterManagementEndpoints lists the management endpoints of router daemons
// that control-mode should push admission-state updates to (S-BL.ADMISSION-SYNC-WIRE).
// Each entry carries a TCP host:port address of the router's management server.
// An empty slice means no push replication (single-router co-located deployment
// uses the Unix socket fallback; see S-BL.ADMISSION-SYNC-WIRE-rulings.md Ruling 2).
// Each entry's Addr is validated as host:port (E-CFG-003 / validateHostPort).
RouterManagementEndpoints []RouterManagementEndpoint `yaml:"router_management_endpoints"`

// RouterManagementEndpoint is a single entry in the router_management_endpoints list.
type RouterManagementEndpoint struct {
    Addr string `yaml:"addr"`
}
```

Validation: each `Addr` is validated by `validateHostPort` in
`config.Config.Validate()`, exactly as `UpstreamRouters[i].Addr` is validated
today. Failure cap: same `UpstreamRoutersFailureCap` pattern applies — collect
and return up to N validation errors rather than fail-fast (existing pattern at
`config.go:221-237`). SIGHUP-reload: `RouterManagementEndpoints` is read at
daemon startup from the loaded config; a SIGHUP reload re-reads the config and
updates the endpoint list held by the push client (see dial-client design below).

**Unix socket fallback.** When `RouterManagementEndpoints` is empty AND control
is started on the same machine as a router (`/run/switchboard-router.sock`
exists), control may fall back to dialing the router's local Unix socket. This
is a zero-config convenience path for single-machine deployments — not the
primary mechanism. The fallback is optional for the near-term implementation; if
not implemented in this story, document it as a follow-on. Do NOT implement
fallback as the default; explicit TCP config is the correct posture for
multi-router deployments.

**Config field is CONTROL-MODE ONLY.** The router's own `config.Config` does
not read this field. Routers never need to know what other routers exist.

### Dial client design

**RULING: A new `admissionSyncClient` type (or equivalent, name at implementer
discretion) in a new file `cmd/switchboard/admission_sync_client.go`.**

Structural properties:
- Holds the slice of `RouterManagementEndpoint` addresses (updated on SIGHUP).
- Holds control's `daemonPriv ed25519.PrivateKey` for authenticating to each
  router's management socket.
- Exposes four methods mirroring the four write paths:
  `PushRegisterKey`, `PushRevokeKey`, `PushSetKeyExpiry`, `PushRemoveSVTN`.
- Each method dials each configured endpoint, completes the mgmt
  challenge-response handshake, sends the corresponding
  `internal.admission.*` RPC, reads the response, and closes the connection.
  **Dial-on-demand per push event** — no persistent idle connection in the
  near-term story.

**Retry-with-backoff.** Each dial attempt for a given endpoint uses bounded
exponential backoff on failure:
- Initial delay: 100ms
- Multiplier: 2
- Maximum delay: 10s
- Maximum attempts: 5 (subject to story implementer judgment; document the
  chosen values in code comments citing this ruling)

After N consecutive failures to a given endpoint, log a `WARN` and stop
retrying for that event. The next push event re-attempts from scratch.
**Push failure does NOT roll back the control-side write.** Control is the
authority; the admitted-key write to control's own `AdmittedKeySet` has already
committed. A failed push to a router means that router is temporarily stale;
the router's VLR-local snapshot (Ruling 3) will bridge the detachment period
until control re-pushes on its next startup.

**SIGHUP reload integration.** When `runControl` receives a SIGHUP and reloads
the config, it updates the `admissionSyncClient`'s endpoint list from the new
config. No in-flight push is interrupted; the new list is used for the next
push event.

**Full-snapshot push on control startup.** When `runControl` initializes, after
constructing the `SVTNManager` and loading any persisted state, it calls a new
method `PushFullSnapshot(ctx)` on the `admissionSyncClient`. This method
iterates over all entries in `SVTNManager`'s `AdmittedKeySet` snapshot (via
`ListBySVTN` across all SVTNs) and issues `internal.admission.register` (and
`internal.admission.expire` for any entries with non-zero expiry) to each
configured router endpoint.

**Goroutine / WaitGroup contract.** If the `admissionSyncClient` spawns any
background goroutines (e.g., for non-blocking push with retry), the caller
(`runControl`) MUST call `wg.Add(1)` synchronously before dispatching `go
client.runBackground(ctx, wg, ...)` — the canonical ARCH-01 §Goroutine
WaitGroup Contract (F-DWIP3-001). The goroutine itself calls `defer wg.Done()`
and must drain on `ctx` cancellation. If all push calls are synchronous
in-line (no background goroutine), no WaitGroup interaction is needed. The
near-term story MAY implement synchronous push (simpler); if so, call it out
explicitly in the implementation.

### Hooking push into the four write handlers

**RULING: The push is called from `admin_handlers.go` AFTER the successful
`SVTNManager.*` write, before returning the result to the caller.**

In each of the four handler functions (`makeRegisterHandler`,
`makeRevokeHandler`, `makeExpireHandler`, `makeAdminSVTNDestroyHandler`), after
the successful `m.RegisterKey / m.RevokeKey / m.ExpireKey / m.Destroy` call
and before constructing the response, call the corresponding
`admissionSyncClient.Push*` method.

The `admissionSyncClient` is injected into each handler via the `BuildAdminHandlers`
signature extension — a new parameter `syncClient admissionSyncer` (interface,
see below) alongside the existing `m *svtnmgmt.SVTNManager` and
`ops *mgmt.OperatorKeySet` parameters.

**Interface shape (for testability and nil-safety):**

```go
// admissionSyncer is the interface the four admin write handlers use to push
// admission-state changes to configured routers.
// A nil value is explicitly permitted — nil means "no routers configured"
// (single-router co-located deployment); methods are no-ops.
// Production: *admissionSyncClient. Tests: a mock/stub.
type admissionSyncer interface {
    PushRegisterKey(ctx context.Context, svtnName string, pubkey ed25519.PublicKey, role admission.KeyRole) error
    PushRevokeKey(ctx context.Context, svtnName string, pubkey ed25519.PublicKey, role admission.KeyRole, confirm bool) error
    PushSetKeyExpiry(ctx context.Context, svtnName string, pubkey ed25519.PublicKey, ttl time.Duration) error
    PushRemoveSVTN(ctx context.Context, svtnName string) error
}
```

Nil check: if `syncClient == nil`, the push is skipped. This preserves the
existing behavior for the router and console modes (which call `BuildAdminHandlers`
with `nil` today per ADR-004 / AC-004 — and will continue to pass nil;
they do NOT push to other routers). Only control mode passes a non-nil
`admissionSyncClient`.

**Push failure behavior in the handler.** Per the ratification in
`identity-cluster-architecture.md` Section 7 (detachment-tolerance ruling):
push failure DOES NOT roll back the control-side write and DOES NOT return an
error to the admin operator. It is logged at `WARN` level. The response
returned to the sbctl caller reflects the control-side write success, which is
the authoritative operation.

Rationale: returning an error to the operator when the control write succeeded
would be misleading — the key IS registered in the authoritative store. The
operator would need to retry and the control side would receive a duplicate
write (which is fine — `RegisterKey` is last-write-wins idempotent). Logging
at WARN level is the correct signal: the operator can observe router staleness
and trigger a control restart (which issues the full-snapshot push) to
resynchronize.

---

## Ruling 3 — Router-side: new push handler + VLR-local snapshot

### New push handlers on the router-mode management server

**RULING: A new file `cmd/switchboard/admission_sync_wire.go` registers the
four `internal.admission.*` handlers on the router-mode management server.**

This file follows the same structural pattern as `router_control_wire.go`:
- A top-level `wireAdmissionSyncHandlers(srv *mgmt.Server, ks *admission.AdmittedKeySet, snapshotPath string) error` function
- Called from `runRouter` after `newMgmtServer` and before `serveMgmtServer`
  (register-before-serve invariant, F-P2L1-001)
- Four `mgmt.Handler` registrations, one per `internal.admission.*` command

Each handler receives the push args, decodes them, calls the corresponding
`AdmittedKeySet` method, and then — after a successful write — writes the
VLR-local snapshot to disk (see snapshot section below).

**Trust model for the push connection.** The router's management server uses
the same `OperatorKeySet`-based authentication as today. For the control-mode
push connection to authenticate, control's daemon private key must be in the
router's `AuthorizedOperatorKeys` list. This is an operational configuration
requirement:
- Each router's config must include control's Ed25519 public key in
  `authorized_operator_keys`.
- This is the same model as operator-to-daemon authentication (the operator's
  public key is in `authorized_operator_keys`; here, control's key is added
  alongside it).

This is a human configuration dependency — document it as an operator
requirement in the implementation, not as something the story can bypass.

**ARCH-08 compliance.** `cmd/switchboard` (ARCH-08 position 18, the top) may
import all lower-position packages. The new `admission_sync_wire.go` imports
only `internal/admission` (already imported by `mgmt_wire.go`) and
`internal/mgmt` (already imported). No new package import, no ARCH-08 position
change needed.

### VLR-local admitted-state snapshot

**RULING: JSON format. One file. Atomic write.**

**Format choice: JSON.** Rationale:
- Already used for all mgmt-protocol messages; no new serialization dependency.
- Human-readable, debuggable without tooling (important for operational
  troubleshooting when control is not available).
- Forward-compatible with the HLR/VLR replication model: the same JSON
  schema can be used as the carrier format for a future replication protocol.
  The near-term story defines the schema; the future story evolves it
  (schema versioning via a `schema_version` field, see below).
- Deterministic marshaling: use `encoding/json` with fields in a defined order
  (struct tags); no map iteration order dependency in the snapshot structure.
- Size: a snapshot with thousands of keys is still small (each entry is ~200
  bytes of JSON); JSON overhead is acceptable at any practical admitted-key-set
  size.

CBOR and binary were considered and rejected:
- CBOR: adds a dependency (`github.com/fxamacker/cbor/v2` or similar); no
  tooling in the existing codebase.
- Binary: not human-readable; requires a custom format spec; harder to evolve
  without a separate schema document.

**Snapshot JSON schema:**

```json
{
  "schema_version": 1,
  "timestamp": "2026-07-15T12:00:00Z",
  "svtns": [
    {
      "svtn_id": "<hex-encoded [16]byte>",
      "keys": [
        {
          "pubkey": "<base64url-encoded 32-byte Ed25519 public key>",
          "role": "access",
          "revoked": false,
          "expiry": "2027-07-15T12:00:00Z"
        }
      ]
    }
  ]
}
```

Fields:
- `schema_version` (int): currently 1. Forward-compat gate: if a future
  snapshot has a `schema_version` the router does not recognize, it fails
  closed on startup (treat as corrupt — see below).
- `timestamp` (RFC3339 UTC string): write time; informational only.
- `svtns[].svtn_id` (hex string, 32 hex chars = 16 bytes): the SVTN UUID.
- `svtns[].keys[].pubkey` (base64url, no padding): the raw 32-byte Ed25519
  public key.
- `svtns[].keys[].role` (string): "control", "console", or "access".
- `svtns[].keys[].revoked` (bool): true if this key has been revoked.
- `svtns[].keys[].expiry` (RFC3339 UTC string, omitempty): absent if no expiry.

**What the snapshot does NOT include:**
- The `admitted` boolean per `AdmittedKey`. A loaded entry is always
  `admitted=false` on load — the challenge-response handshake that sets
  `admitted=true` is a live connection event, not persisted state. Nodes must
  re-identify after a router restart regardless.
- Nonces (the replay-prevention map). Nonces are ephemeral per-connection
  state; persisting them would be wrong (a router restart should not honor
  60-second-old nonces that may no longer be valid). The nonce map starts
  empty on every startup.
- `FrameAuthKey` and `NodeAddr`: these are derived deterministically from
  `(svtnID, pubkey)` via `hmac.DeriveKey` and `frame.DeriveNodeAddress`
  respectively — they are NOT stored. `RegisterKey` recomputes them on load,
  same as on a fresh `RegisterKey` call from a push. This avoids storing
  derived material and keeps the snapshot schema clean.

**Snapshot write path.** After each successful push-handler invocation:
1. Call `ks.ListBySVTN(svtnID)` for the affected SVTN (or all SVTNs for
   `RemoveSVTN`) to capture current state.
2. Serialize to the snapshot JSON schema.
3. Atomic write via `os.WriteFile` to a temp file in the same directory,
   then `os.Rename` to `admission_state_file` path. This prevents a
   partial-write leaving a corrupt snapshot (standard Go atomic-write
   idiom).
4. On write failure: log WARN, do not fail the push handler (the push
   succeeded and the in-memory keyset is up to date; the snapshot file
   remains from the previous write — stale but not corrupt).

For a full-snapshot push at control startup, the router writes the snapshot
once after all `internal.admission.register` calls are processed. Each
individual call still triggers a snapshot write (step 3 above) — this is
acceptable; the per-call write cost is negligible.

### Config field: `admission_state_file`

**RULING: New `AdmissionStateFile string` field in `config.Config`.**

```go
// AdmissionStateFile is the path where the router-mode daemon writes and reads
// its VLR-local admitted-state snapshot (S-BL.ADMISSION-SYNC-WIRE).
// Optional — when absent or empty, the router starts with an empty keyset and
// does not persist admission state across restarts.
// When present, must be a non-empty, writable file path (E-CFG-XXX).
AdmissionStateFile string `yaml:"admission_state_file"`
```

This field is read by `runRouter` only; it is ignored by all other daemon
modes. Validation: if present, verify the parent directory exists and is
writable (`os.Stat` the parent dir). Do not require the file itself to
exist — a missing file is the "fresh install" state (empty keyset, await push).

### Router startup load behavior

In `runRouter`, after constructing the `AdmittedKeySet` and before starting the
management server (or the data-plane listener):

1. If `cfg.AdmissionStateFile == ""`: start with empty keyset (existing behavior).
2. If `cfg.AdmissionStateFile != ""`:
   a. Attempt to read and parse the file.
   b. If the file does not exist: start with empty keyset and log INFO
      ("admission_state_file not found; starting with empty keyset — awaiting
      push from control").
   c. If the file exists but fails to parse (invalid JSON, unknown
      `schema_version`, missing required fields): **fail closed** — return an
      error from `runRouter` and do not start. Log a FATAL-level message with
      the file path and parse error. The operator must investigate and either
      delete the corrupt file (triggering fresh-start semantics) or restore
      from backup.
   d. If the file parses successfully: iterate over `svtns[].keys[]` and call
      `ks.RegisterKey(svtnID, pubkey, role)` for each entry. If `revoked==true`,
      call `ks.RevokeKey(svtnID, nodeAddr)` after registration. If `expiry` is
      present and non-zero, call `ks.SetKeyExpiry(svtnID, nodeAddr, expiry)`.
      Log INFO with the count of loaded entries per SVTN.

The `admitted=false` invariant from step (d) is important: loaded entries are
in the keyset but NOT admitted. Connecting nodes must still complete the
challenge-response handshake (`NODE_IDENTIFY` per `S-BL.NODE-IDENTIFY-WIRE`)
before `IsAdmitted` returns true. The snapshot persists the authorization
eligibility (the key IS registered), not the live admission state.

---

## Ruling 4 — Detachment/restart resilience: scenario enumeration

The three binding constraints from `identity-cluster-architecture.md` Section 8
are:
- (a) MANY routers
- (b) control-detachment resilience (HARD)
- (c) HLR/VLR target end-state

Below is the complete scenario map showing how this design satisfies constraint (b):

| Scenario | What happens | Constraint (b) satisfied? |
|---|---|---|
| **Control absent at router start (fresh install)** | `admission_state_file` missing → router starts with empty keyset. Management socket is ready. When control first attaches and dials in, it issues the full-snapshot push. Router's keyset is populated from that push. Until then, the router has no admitted keys — `AdmitNode` returns `ErrNotAdmitted` for all requests. | **Yes** — router starts without control. Not a degraded state for a fresh install (no keys to serve yet anyway). |
| **Control absent at router start (restart after prior sync)** | `admission_state_file` exists from prior push → router loads snapshot into keyset. Router serves from loaded state immediately. Control's absence does not delay or block the router's start. When control eventually reattaches, it pushes the current snapshot (which may include changes made while router was down), overwriting the router's loaded state. | **Yes** — router starts independently from snapshot; serves existing admitted keys without control. HARD REQUIREMENT satisfied. |
| **Router restart during control detachment** | Same as "control absent at router start (restart after prior sync)" — the snapshot was last written at the most recent successful push. Any admission-state changes control made while detached are not yet reflected in the snapshot, and will be pushed when control reattaches. | **Yes** — router recovers from its own snapshot; no dependency on control for startup. The gap between "control's last push" and "router restart" is bounded by the control detachment period; the VLR-local snapshot bridges it. |
| **Control reconnect after detachment** | Control starts (or the process restarts), calls `PushFullSnapshot(ctx)`, which iterates control's current `AdmittedKeySet` and pushes `internal.admission.register` for every entry to each configured router endpoint. Router handler receives each push, applies to its keyset, and writes the updated snapshot. Router is now current. | **Yes** — no manual operator intervention needed for resync; re-dial is automatic on control start. |
| **Push failure during a live write** | `admin.key.register` succeeds on control, push to one router fails (timeout, connection refused, router restarting). The key is in control's keyset. The push is logged at WARN. The router is temporarily stale. When control next starts OR when the next push event occurs, the router will be pushed the full current snapshot (or at minimum the next delta). | **Yes** — the router's prior snapshot remains intact. It may be missing the latest key until the next push. This is a temporary stale window, not a HARD REQUIREMENT violation (the router is functional with the keys it has). |
| **Control receives rapid writes (burst of `admin.key.register`)** | Each write triggers a per-write push. If N writes arrive in rapid succession, N pushes are dispatched to each router. Dial-on-demand means N TCP connections are opened. This is fine for the near-term story at realistic admitted-key-set sizes. A batch-push optimization (debounce + single snapshot) is a follow-on. | **Yes** — all writes are propagated; no loss. |

---

## Ruling 5 — Full-vs-split scoping decision

**RULING: This story delivers FULL admitted-key-material sync (pubkey + role +
revoked status + expiry), not a narrower SVTN-presence-only slice.**

Reasons:
1. `S-BL.NODE-IDENTIFY-WIRE` needs full material. Its `AdmitNode` call requires
   the actual `ed25519.PublicKey` to verify the `ChallengeResponse.NonceSig`.
   An SVTN-presence-only sync (which delivers only "SVTN X has some members"
   without the actual pubkeys) cannot satisfy `AdmitNode`.
2. Splitting would create a temporary state where the router has SVTN presence
   but no key material — `wireDiscoveryListener` could join multicast groups
   that the router cannot yet verify frames for. This is not a useful
   intermediate state operationally.
3. The implementation cost of full-material sync is not materially higher than
   SVTN-presence-only: the same push RPC shape is used; the args just include
   the full `(pubkey, role)` tuple alongside the `svtn_id`.
4. The VLR-local snapshot (Ruling 3) already encodes full material — there is
   no point in designing a partial-material snapshot.

**Discovery-wire side note.** `S-BL.DISCOVERY-WIRE` Task 3 needed only the
SVTN-presence signal (multicast group join address derived from `svtnID`). That
need is satisfied as a natural side-effect of full-material sync: after
`internal.admission.register` for any key in SVTN X, the router's keyset has
at least one entry for SVTN X, and `discovery.MulticastAddrFor(svtnID)` can
be called to join the group. No separate SVTN-presence mechanism is needed.

---

## Ruling 6 — Test strategy sketch

### Unit-testable (no live mgmt server required)

| Test | What it exercises |
|---|---|
| Snapshot write: call `RegisterKey(svtn, pubkey, role)` on an `AdmittedKeySet`, serialize to snapshot format, verify JSON field values | Serialization correctness |
| Snapshot round-trip: serialize → write to temp file → read → deserialize → call `RegisterKey` for each entry → verify `ListBySVTN` returns expected entries | Snapshot write/load correctness |
| Snapshot load with `revoked=true`: verify `RevokeKey` is called post-load; `IsAdmitted` returns false | Revocation correctness after load |
| Snapshot load with non-zero expiry: verify `SetKeyExpiry` is called post-load | Expiry correctness after load |
| Fail-closed on corrupt file: write invalid JSON to snapshot path, call `runRouter` startup load logic, expect error return | Fail-closed on corrupt |
| Fail-closed on unknown `schema_version`: write `{"schema_version": 999, ...}` to snapshot path, expect error | Forward-compat gate |
| Missing file → empty keyset: path configured but file absent → `ListBySVTN` returns empty | Missing file semantics |
| Config parsing: `router_management_endpoints` with valid and invalid `host:port` entries | Config validation |
| `admissionSyncClient.PushRegisterKey` with nil sync client: verify no-op, no panic | Nil syncer safety |

### Integration-testable (requires two mgmt servers: control and router)

These require a real management socket connection between two in-process
`mgmt.Server` instances (the existing `mgmt_wire_test.go` pattern shows how to
wire a `mgmt.Server` with a Unix socket in tests).

| Test | What it exercises |
|---|---|
| `admin.key.register` on control → push to router → router's `AdmittedKeySet` has the entry | End-to-end push RPC |
| `admin.key.revoke` on control → push to router → router's `AdmittedKeySet` marks key revoked | Revoke propagation |
| `admin.key.expire` on control → push to router → router's keyset has non-zero expiry | Expiry propagation |
| `admin.svtn.destroy` on control → push router → router's keyset has no entries for that SVTN | RemoveSVTN propagation |
| Push failure (router management socket not listening) → control write still succeeds → error logged | Failure isolation |
| Control startup with populated snapshot → full-snapshot push → router keyset matches control | Startup snapshot push |
| Router restart → loads snapshot → entries present without control → control reattaches → push overwrites | Restart resilience |

The test infrastructure for two-mgmt-server integration tests can be built
on top of the existing `startMgmtServer` helper and `net.Pipe()` or a
loopback TCP listener.

---

## Ruling 7 — BC groundwork for PO (list only, no BCs written here)

The product owner needs to author or amend the following behavioral contracts
before the story-writer can write acceptance criteria:

| BC action | Description |
|---|---|
| New BC: `admission-state-sync` (ss-XX number TBD) | Behavioral contract for the push RPC: preconditions (control has admitted keys; router endpoint configured), postconditions (router's `AdmittedKeySet` matches control's on successful push), invariants (push failure does not roll back control write; admitted=false on load), error cases (router unreachable: WARN logged, no rollback). |
| New BC: `admission-state-snapshot` (ss-XX number TBD) | Behavioral contract for the VLR-local snapshot: write-on-receive, load-on-startup-if-present-and-valid, fail-closed-on-corrupt, missing-file→empty-keyset, schema-version gate. |
| New BC or amend BC-2.09.003: `router_management_endpoints` config field | Config-validation postconditions: each entry validated as host:port (E-CFG-003), SIGHUP-reload semantics, empty list is valid (no push replication), config-mode constraint (control-mode only). |
| New BC or amend BC-2.09.003: `admission_state_file` config field | Config-validation postconditions: parent directory must exist if path is set (E-CFG-XXX), empty string means no persistence, router-mode only. |
| Amend BC-2.05.004 (admin.key.register, admin.key.revoke, admin.key.expire) | Add postcondition: "If `RouterManagementEndpoints` is non-empty and the push to a router fails, the write is not rolled back; WARN is logged." This makes the push-failure behavior a first-class postcondition, not an implementation detail. |
| Amend BC-2.07.001 PC-3 (admin.svtn.destroy) | Add postcondition: "push `internal.admission.remove-svtn` to all configured router endpoints after `RemoveSVTN`." |

---

## Ruling 8 — ARCH-08 / ARCH-01 compliance

### ARCH-08 compliance

All new code lives in `cmd/switchboard` (position 18 — the top of the import
DAG). No new `internal/` package is introduced. The new files:
- `cmd/switchboard/admission_sync_client.go` — imports `internal/admission`,
  `internal/mgmt`, `internal/config` (all already imported by `mgmt_wire.go`)
- `cmd/switchboard/admission_sync_wire.go` — same imports

No new ARCH-08 position registration is needed. The import graph is unchanged.

The only package that gains a new import consideration is `internal/admission`:
the snapshot serialization requires reading `AdmittedKey` fields
(`PublicKey`, `Role`, `IsRevoked()`, `KeyExpiry()`). All of these are already
exported — no changes to `internal/admission` are needed for the snapshot
serialization (all serialization code lives in `cmd/switchboard`).

**ARCH-09 purity:** both new files are boundary-effectful (they perform I/O:
TCP dials and file writes). This is consistent with the existing
`mgmt_wire.go` and `router_control_wire.go` purity classification.

### ARCH-01 goroutine / WaitGroup contract

The `admissionSyncClient` MAY spawn a background goroutine for non-blocking
push (e.g., to avoid blocking the admin handler's response while a push is
in flight). If it does:

- `runControl` calls `wg.Add(1)` synchronously before `go client.runBackground(ctx, wg)`.
- The goroutine calls `defer wg.Done()` and exits cleanly on `ctx.Done()`.
- This is the F-DWIP3-001 / ARCH-01 §Goroutine WaitGroup Contract pattern,
  established by `serveMgmtServer` (mgmt_wire.go:322) and
  `wireDiscoveryListener` (discovery_wire.go:77).

If push is synchronous in-line (no background goroutine), no WaitGroup
interaction is needed.

---

## Summary table

| Item | Ruling |
|---|---|
| Push RPC protocol | Existing `internal/mgmt` JSON-over-TCP protocol; new `internal.admission.*` command names; same encoding as `admin.*` handlers |
| Push commands | `internal.admission.register`, `internal.admission.revoke`, `internal.admission.expire`, `internal.admission.remove-svtn` |
| Push-vs-snapshot | Per-write delta push + full-snapshot push on control startup |
| Control config field | `RouterManagementEndpoints []RouterManagementEndpoint` in `config.Config`, twin of `UpstreamRouters` |
| Control dial direction | Control dials routers (TCP); dial-on-demand per push event; retry-with-backoff; no persistent idle connection |
| Push-failure behavior | Log WARN; never roll back control write; next startup push will resync |
| Router TCP mgmt listener | Router needs a TCP mgmt endpoint; configured by pointing `management_socket` at a TCP `host:port` in the router's config; control's `RouterManagementEndpoints` references this address |
| Router config field | `AdmissionStateFile string` in `config.Config` |
| Snapshot format | JSON; `schema_version: 1`; fields: svtn_id (hex), pubkey (base64url), role, revoked, expiry; no admitted, no FrameAuthKey, no NodeAddr (all derived) |
| Snapshot write | Atomic: write temp + rename; on each successful push-handler invocation |
| Snapshot load | At startup if file present and valid; fail-closed on corrupt; missing file → empty keyset |
| Full-vs-split scoping | FULL material sync; no SVTN-presence-only slice |
| Auth: control→router | Control's daemon pubkey in router's `authorized_operator_keys` (operator config requirement) |
| ARCH-08 impact | None; all new code in cmd/switchboard (position 18) |
| ARCH-01 WaitGroup | Standard F-DWIP3-001 pattern if background goroutine used; otherwise no WaitGroup needed |
| Router TCP mgmt listener security posture | RESOLVED (human ratified 2026-07-15): permit any bind address; challenge-response is the auth boundary; no loopback restriction; startup log of bind address required (advisory); see Ruling 9 |
| Open human flags | None — story is decomposition-ready |

---

## Ruling 9 — Router TCP management listener security posture

**RESOLVED (human ratified 2026-07-15):** Permit any bind address. The mgmt
challenge-response handshake (ADR-012) is the authentication boundary;
non-loopback network exposure is the operator's firewall-policy responsibility.
This is deliberately distinct from VP-073 console-mode (loopback-only) because
control→router push is inherently cross-host: the network is multi-machine by
design (control attaches remotely, many routers), so loopback-only binding
would defeat the story's purpose. Consistent with how `ListenAddr` and
`UpstreamRouters` are already handled (validated as host:port, no loopback
restriction).

**NOT loopback-restricted** — do not apply the `isMgmtLoopbackHost` guard from
`buildMgmtListener` to router-mode TCP management endpoints.

### Implementation requirements for `buildMgmtListener`

- `validateHostPort`-style validation applies to every entry in
  `RouterManagementEndpoints[].Addr` (E-CFG-003), same as `UpstreamRouters`.
- No loopback restriction is applied; any valid `host:port` is accepted,
  including `0.0.0.0:PORT` and named interfaces.
- A startup INFO log naming the bind address is required: e.g.
  `"router management listener bound to %s (ensure firewall policy restricts
  access as appropriate)"`. This is advisory, not a gate — it surfaces the
  bound address for operators who care.

### AC implication for the story

The router TCP management listener MUST NOT enforce loopback-only binding.
The story's acceptance criteria must include:
- A test or explicit case confirming a non-loopback `host:port` in
  `RouterManagementEndpoints` is accepted without error.
- The startup log line emitted when the management listener is bound
  (inspectable in integration tests).

### Open-flags status

This was the last open human-decision flag in this rulings document.
**All AC-prerequisites are now satisfied. The story S-BL.ADMISSION-SYNC-WIRE
is decomposition-ready — zero open flags remain.**
