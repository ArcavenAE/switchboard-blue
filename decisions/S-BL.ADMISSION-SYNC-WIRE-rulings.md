---
artifact_id: S-BL.ADMISSION-SYNC-WIRE-rulings
document_type: rulings
version: "1.5"
status: draft
producer: architect
timestamp: 2026-07-15T00:00:00Z
modified: 2026-07-17T00:00:00Z
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

# S-BL.ADMISSION-SYNC-WIRE: Elaboration Rulings 1.4

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.5 | 2026-07-17 | F-P6-02 adjudicated (Ruling 13): partial-failure gap in PushFullSnapshot for revoked entries resolved via approach (c) — skip `internal.admission.register` for revoked entries entirely; issue revoke-only RPC for entries already present on router. No wire format change. BC-2.05.009 Invariant 6 / PC-5 tension resolved: the "less-restrictive" prohibition is absolute; "stale/missing" (permitted by PC-5) and "actively made less-restrictive" (forbidden by Invariant 6) are distinct states. F-P6-01 confirmed as impl-only fix (no spec change). |
| 1.4 | 2026-07-17 | F-P3-01 + F-P3-02 adversary pass-3 findings adjudicated (human decision: both in-scope). Code-verified: (1) `runControl` (`mgmt_wire.go:1176`) constructs `ks := admission.NewAdmittedKeySet()` fresh; no load path; `PushFullSnapshot` at line 1222 hits the empty-keyset early return at `admission_sync_client.go:378–381` — EC-007 resync guarantee is inert today; (2) no persist-write hook in any of the four admin handlers (`admin_handlers.go`); (3) `writeSnapshotAtomic` / `loadSnapshotFromFile` / `marshalSnapshot` / `unmarshalSnapshot` in `admission_sync_snapshot.go` are fully implemented and reusable as-is; (4) `mgmtListenAddr` (`mgmt_wire.go:187–199`) auto-detect fires for ALL non-console modes — control and access modes with a `host:port` `management_socket` silently bind TCP on all interfaces; (5) `buildMgmtListener` loopback guard at line 221 checks `if mode == "console"` — control/access TCP has no guard and no bind log. Ruling 11 added: control-side admission keyset persistence via new `control_admission_state_file` config field (control-mode-only, PC-15 in BC-2.09.003; write on each committed admin.key.*/admin.svtn.* mutation, before dispatchPush, advisory; load before PushFullSnapshot on startup, fail-closed on corrupt, missing→empty). Ruling 12 added: scope-correct the TCP auto-detect + loopback guard — router mode keeps no loopback guard (Ruling 9 unchanged); control/access modes with a host:port management_socket apply the loopback guard and MUST emit a bind-address INFO log; only router mode may bind non-loopback TCP. Summary table rows added for Rulings 11 and 12. BC/story propagation list specified. |
| 1.3 | 2026-07-17 | F-2 adversary defect (HIGH, feature-blocking): router TCP mgmt listener wiring gap. Code-verified facts: (1) `mgmtNetwork(mode)` (`mgmt_wire.go:159–164`) returns `"tcp"` ONLY for `"console"`; router/control/access all return `"unix"` — no existing TCP-bind path for router mode; (2) `admission_sync_client.go:142` always dials `"tcp"` — confirmed; (3) `buildMgmtListener` TCP branch (`mgmt_wire.go:194–208`) enforces `isMgmtLoopbackHost` and is only reached for console because `mgmtNetwork("router")=="unix"`; (4) `validateHostPort` (`internal/config/config.go:401`) uses `net.SplitHostPort` + numeric range check — reusable for auto-detect; (5) blast-radius: all existing router tests supply `tempSockPath(t)` → filesystem paths (e.g. `/tmp/sb-XXXXX/m.sock`); `net.SplitHostPort` fails on those paths (no colon), so auto-detect TCP-vs-unix on `management_socket` has zero impact on existing tests. Ruling 10 added: auto-detect mechanism — if `management_socket` parses as a valid `host:port` via `validateHostPort`, `mgmtNetwork` returns `"tcp"` for router mode and `buildMgmtListener` binds a TCP listener WITHOUT the `isMgmtLoopbackHost` guard (Ruling 9 already ratified: no loopback restriction for router); otherwise unix default preserved. Default when `management_socket` absent or a filesystem path: unix (existing behavior unchanged). AC-008 correction: current postconditions only assert config acceptance + INFO log; must also assert a real TCP bind and a real TCP connection succeed. Corrected postconditions and new test names specified. Summary table `Router TCP mgmt listener` row updated. F-1/F-2 interaction note added (async push: Decision 4 permit applies; the dial-on-demand connect in `pushRPC` is unaffected by the network-selection change — both changes are independent and compose cleanly). |
| 1.2 | 2026-07-16 | Contradiction fix: Ruling 1 / Decision 2 wire encoding of `svtn_id` corrected from "SVTN name string" to "32 lowercase hex characters encoding the 16-byte `[16]byte` UUID". Code-verified facts: (1) `SVTN.ID` is `crypto/rand`-generated at `svtnmgmt.go:197–199`, not name-derived; (2) `hmac.DeriveKey` (`hmac.go:130`) and `frame.DeriveNodeAddress` (`address.go:11`) both consume the exact `[16]byte` as salt/input — any mismatch produces wrong FrameAuthKey/NodeAddr, defeating `AdmitNode`; (3) `runRouter` (`mgmt_wire.go:491`) constructs no `SVTNManager`, so the router has no name→ID map; (4) `makeRegisterHandler` (`admin_handlers.go:185`) holds `m *svtnmgmt.SVTNManager` with `SVTNByName(name) (SVTN, bool)` at `svtnmgmt.go:608` — control resolves name→ID before putting anything on the wire. `admissionSyncer` interface updated to take `svtnID [16]byte` instead of `svtnName string` (Decision 5 / Ruling 2); admin handlers resolve name→ID at the call site via `m.SVTNByName`. Summary table `svtn_id` row updated. All other rulings unchanged. |
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
admin handlers, with the following correction to wire `svtn_id`:
- `svtn_id` — **32 lowercase hex characters encoding the 16-byte `[16]byte`
  SVTN UUID** (e.g. `"a3f2b1c9..."`). This is the VERIFIED-CORRECT encoding.
  Rationale: (a) `SVTN.ID` is `crypto/rand`-generated (`svtnmgmt.go:197–199`),
  not name-derived; (b) `hmac.DeriveKey` (`hmac.go:130`) and
  `frame.DeriveNodeAddress` (`address.go:11`) both consume the exact `[16]byte`
  value — a name-based encoding would produce a different HKDF salt and SHA-256
  input, causing FrameAuthKey and NodeAddr to diverge between control and router,
  defeating `AdmitNode`; (c) the router (`runRouter`, `mgmt_wire.go:491`) has no
  `SVTNManager` and therefore has no name→ID resolution path; (d) the admin
  handler on control (`makeRegisterHandler`, `admin_handlers.go:185`) already
  holds `m *svtnmgmt.SVTNManager`, which exposes `SVTNByName(name) (SVTN, bool)`
  (`svtnmgmt.go:608`), allowing name→`[16]byte` resolution at the call site
  before anything goes on the wire. The admin handlers resolve name→`[16]byte`
  BEFORE calling `admissionSyncer.Push*`, and the syncer methods accept `[16]byte`
  directly (see corrected interface, Ruling 2). If the router has no SVTN record
  for the given `svtnID` yet (legitimately possible on fresh install),
  `RegisterKey` creates the entry idempotently.
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

**Interface shape (for testability and nil-safety) — CORRECTED in v1.2:**

```go
// admissionSyncer is the interface the four admin write handlers use to push
// admission-state changes to configured routers.
// A nil value is explicitly permitted — nil means "no routers configured"
// (single-router co-located deployment); methods are no-ops.
// Production: *admissionSyncClient. Tests: a mock/stub.
//
// svtnID is the resolved [16]byte UUID — NOT the human-readable SVTN name.
// The admin handler (which holds *svtnmgmt.SVTNManager) resolves name→[16]byte
// via m.SVTNByName before calling Push*. The router has no SVTNManager and
// therefore no name→ID map; it must receive the [16]byte directly.
type admissionSyncer interface {
    PushRegisterKey(ctx context.Context, svtnID [16]byte, pubkey ed25519.PublicKey, role admission.KeyRole) error
    PushRevokeKey(ctx context.Context, svtnID [16]byte, pubkey ed25519.PublicKey, role admission.KeyRole, confirm bool) error
    PushSetKeyExpiry(ctx context.Context, svtnID [16]byte, pubkey ed25519.PublicKey, ttl time.Duration) error
    PushRemoveSVTN(ctx context.Context, svtnID [16]byte) error
}
```

**Call-site pattern in admin handlers** (e.g., `makeRegisterHandler`):
```go
// After m.RegisterKey(a.SVTNName, pubkey, role) succeeds:
svtn, ok := m.SVTNByName(a.SVTNName)
if ok && syncClient != nil {
    _ = syncClient.PushRegisterKey(ctx, svtn.ID, pubkey, role) // failure advisory
}
```
This keeps the admin handler's existing `a.SVTNName` arg structure intact and
resolves name→ID via the SVTNManager that is already present in the handler closure.
The resolved `[16]byte` is then passed through the wire as a 32-char hex string.

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

## Summary table (updated through Ruling 12)

| Item | Ruling |
|---|---|
| Push RPC protocol | Existing `internal/mgmt` JSON-over-TCP protocol; new `internal.admission.*` command names; same encoding as `admin.*` handlers |
| `svtn_id` wire encoding | **32 lowercase hex chars = `[16]byte` UUID** (NOT the human-readable SVTN name). Control resolves name→`[16]byte` via `m.SVTNByName` before calling `admissionSyncer.Push*`. Router has no `SVTNManager`; receives hex directly and calls `ks.RegisterKey(svtnID [16]byte, ...)`. |
| Push commands | `internal.admission.register`, `internal.admission.revoke`, `internal.admission.expire`, `internal.admission.remove-svtn` |
| Push-vs-snapshot | Per-write delta push + full-snapshot push on control startup |
| Control config field (push endpoints) | `RouterManagementEndpoints []RouterManagementEndpoint` in `config.Config`, twin of `UpstreamRouters` |
| Control config field (persistence) | **NEW:** `ControlAdmissionStateFile string` (`control_admission_state_file`); control-mode only; BC-2.09.003 PC-15 (to be added). See Ruling 11. |
| Control dial direction | Control dials routers (TCP); dial-on-demand per push event; retry-with-backoff; no persistent idle connection |
| Push-failure behavior | Log WARN; never roll back control write; next startup push will resync (EC-007 guarantee requires `control_admission_state_file` to be configured) |
| Control persistence write path | After each successful `m.*` mutation: `writeSnapshotAtomic(controlSnapshotPath, ks)` synchronously before `dispatchPush`. Advisory (WARN on failure). No-op when field absent. |
| Control persistence load path | `loadSnapshotFromFile(controlSnapshotPath, ks, w)` in `runControl` BEFORE `syncClient` construction and `PushFullSnapshot`. Fail-closed on corrupt; missing→empty keyset. |
| EC-007 resync guarantee | Requires `control_admission_state_file` configured. Without it, `PushFullSnapshot` pushes an empty keyset — EC-007 is inert. |
| Router TCP mgmt listener | Auto-detect on `management_socket`: `net.SplitHostPort` passes → TCP; otherwise → unix. Router: NO loopback guard (Ruling 9). Default absent/filesystem-path → unix. See Rulings 9 + 10. |
| Router TCP mgmt listener security posture | No loopback restriction (Ruling 9, human ratified 2026-07-15): control→router push is inherently cross-host; ADR-012 challenge-response is auth boundary. INFO log of bound address required. |
| Control/access TCP mgmt listener security posture | **LOOPBACK-ONLY** (Ruling 12): admin planes are operator-local; E-CFG-008 on non-loopback. Loopback TCP accepted. INFO log emitted on successful bind. |
| Mgmt TCP guard (`buildMgmtListener`) | `if mode != "router"` applies loopback guard (replaces `if mode == "console"`). Router exempt (Ruling 9). Console/control/access: loopback-only TCP. |
| Bind-address INFO log | Emitted after `net.Listen` succeeds for ALL TCP binds. Router: includes firewall advisory. Console/control/access: mode name + address, no advisory. |
| Router config field (snapshot) | `AdmissionStateFile string` (`admission_state_file`); router-mode only (BC-2.09.003 PC-13) |
| Snapshot format | JSON; `schema_version: 1`; svtn_id (32 hex chars), pubkey (base64url), role, revoked, expiry; no admitted/FrameAuthKey/NodeAddr (derived). Shared by router (BC-2.05.010) and control (Ruling 11). |
| Router snapshot write | Atomic temp+rename; on each successful push-handler invocation |
| Router snapshot load | At startup if present and valid; fail-closed on corrupt; missing → empty keyset |
| Full-vs-split scoping | FULL material sync; no SVTN-presence-only slice |
| Auth: control→router | Control's daemon pubkey in router's `authorized_operator_keys` (operator config requirement) |
| ARCH-08 impact | None; all new code in cmd/switchboard (position 18) |
| ARCH-01 WaitGroup | Standard F-DWIP3-001 pattern if background goroutine used; otherwise no WaitGroup needed |
| AC-008 correction | Real TCP bind + push handshake assertions; two new test names. See Ruling 10 §AC-008 Correction. |
| Story points estimate | Revised to 10–11 pts (was 8): +2–3 pts for Ruling 11 control persistence. |
| PushFullSnapshot revoked-entry handling | **Skip `internal.admission.register` for revoked entries.** Issue `internal.admission.revoke` only (router treats "key not found" as success — absent = correct non-admissible terminal state). The register+revoke two-RPC pattern is PROHIBITED for revoked entries; a partial-failure window where register succeeds but revoke fails actively violates Invariant 6. Approach (c). See Ruling 13. |
| Open human flags | None — all findings adjudicated. |

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

---

## Ruling 10 — Router TCP management listener: `mgmtNetwork` / `buildMgmtListener` binding mechanism (F-2 fix)

### Defect confirmed (code-verified)

Fresh-context adversary pass (2026-07-17) filed F-2 (HIGH, feature-blocking). The
following code facts were independently verified by the architect before issuing
this ruling:

| Fact | Location | Verified |
|------|----------|---------|
| `mgmtNetwork(mode)` returns `"tcp"` only for `"console"`; all other modes (including `"router"`) return `"unix"` | `mgmt_wire.go:159–164` | Confirmed |
| `buildMgmtListener` TCP branch (loopback guard) is only reachable when `mgmtNetwork` returns `"tcp"` — never for router mode | `mgmt_wire.go:185–209` | Confirmed |
| `admission_sync_client.go` always dials `"tcp"` (`dialer.DialContext(ctx, "tcp", addr)`) | `admission_sync_client.go:142` | Confirmed |
| `validateHostPort` uses `net.SplitHostPort` + numeric port in [0,65535]; available in `internal/config` | `config.go:401–418` | Confirmed |
| All existing router tests supply filesystem socket paths via `tempSockPath(t)` (e.g. `/tmp/sb-XXXXX/m.sock`); `net.SplitHostPort` fails on filesystem paths → auto-detect does not change existing test behavior | `router_sighup_test.go`, `router_control_wire_test.go`, `router_drain_test.go`, `admission_sync_test.go` | Confirmed — zero blast radius |

**Net effect of gap:** operator sets router `management_socket: "0.0.0.0:9093"` →
`resolveManagementSocket` returns `"0.0.0.0:9093"` → `mgmtNetwork("router")` returns
`"unix"` → `listenUnixMgmt("0.0.0.0:9093")` tries to bind a unix socket at a
nonsense path → bind fails or creates a garbage inode. Control dials TCP port 9093
→ connection refused. Routers never receive pushes. `admission.AdmitNode` returns
`ErrNotAdmitted` for all nodes. The story's entire purpose is defeated.

### Chosen mechanism: auto-detect via `validateHostPort` (Candidate A, extended)

**RULING: `mgmtNetwork` and `buildMgmtListener` are made config-driven for router
mode by testing whether the effective `management_socket` value is a valid
`host:port`.**

**Mechanism (precise):**

1. `mgmtListenAddr(cfg, mode)` already calls `resolveManagementSocket` then
   `mgmtNetwork`. Change `mgmtNetwork` to accept the resolved address as an
   additional input, OR (preferred, avoids signature churn) inline the detection
   in `mgmtListenAddr`:

   ```go
   func mgmtListenAddr(cfg *config.Config, mode string) (network, address string) {
       address = resolveManagementSocket(cfg, mode)
       if mode == "console" {
           return "tcp", address
       }
       // Router (and other unix-default modes): if the resolved address is a
       // valid host:port, bind TCP. Otherwise bind unix.
       // This is the Ruling 10 auto-detect for router-mode TCP management
       // (S-BL.ADMISSION-SYNC-WIRE F-2 fix). Uses the existing validateHostPort
       // helper from internal/config — no new parsing logic.
       if validateHostPort(address) == nil {
           return "tcp", address
       }
       return "unix", address
   }
   ```

   `validateHostPort` is already imported via `internal/config` from this package
   (`cmd/switchboard` imports `internal/config`). The function is package-private
   in `internal/config`; it must be promoted to exported, or the detection must
   be inlined using `net.SplitHostPort` + numeric-port check directly in
   `mgmt_wire.go`. Either is acceptable; the implementer chooses the cleaner path.
   **Preference: inline the equivalent check in `mgmt_wire.go` to avoid exporting
   an internal-config helper solely for this call site.** Inline form:
   ```go
   _, portStr, err := net.SplitHostPort(address)
   isTCP := err == nil && portStr != "" // non-empty port = valid host:port
   ```

2. `buildMgmtListener` already branches on `network == "unix"` vs TCP. The TCP
   branch currently enforces `isMgmtLoopbackHost` — **this guard MUST NOT be
   applied to router mode** (Ruling 9). Add a `mode` parameter to
   `buildMgmtListener`, or thread the information through `mgmtListenAddr`. The
   simplest implementation: check `mode != "console"` in the TCP branch to skip
   the loopback guard for router-mode TCP listeners:

   ```go
   func buildMgmtListener(cfg *config.Config, mode string) (net.Listener, error) {
       network, address := mgmtListenAddr(cfg, mode)
       if network == "unix" {
           ln, err := listenUnixMgmt(address)
           if err != nil {
               return nil, fmt.Errorf("buildMgmtListener: %w", err)
           }
           return ln, nil
       }
       // TCP. Loopback restriction applies ONLY to console mode (VP-073 /
       // BC-2.07.004 EC-013 / Ruling D). Router-mode TCP listeners are
       // explicitly NOT loopback-restricted (Ruling 9 / S-BL.ADMISSION-SYNC-WIRE).
       if mode == "console" {
           host, _, splitErr := net.SplitHostPort(address)
           if splitErr != nil {
               return nil, fmt.Errorf("E-CFG-008: management_socket: cannot parse address %q: %w", address, splitErr)
           }
           if !isMgmtLoopbackHost(host) {
               return nil, fmt.Errorf("E-CFG-008: management_socket: console mode requires a loopback address (127.0.0.1, [::1], or localhost); got: %s", address)
           }
       }
       ln, err := net.Listen(network, address)
       if err != nil {
           return nil, fmt.Errorf("buildMgmtListener: %w", err)
       }
       return ln, nil
   }
   ```

3. **Default when `management_socket` is absent or is a filesystem path:**
   `resolveManagementSocket` returns the mode-specific default
   (`mgmtDefaultSocket("router")` = `"/run/switchboard-router.sock"`). A
   filesystem path fails `net.SplitHostPort` (missing colon or non-numeric port)
   → `isTCP = false` → `mgmtListenAddr` returns `"unix"`. **Existing unix-socket
   behavior is fully preserved.** All existing tests pass unchanged.

4. **`mgmtNetwork(mode string) string` is currently exported by name to tests.**
   The `TestMgmtDefaultSocket_PerMode` test in `mgmt_wire_test.go` does NOT call
   `mgmtNetwork` directly (it only calls `mgmtDefaultSocket`). A grep confirms
   no test calls `mgmtNetwork` by name. Safe to refactor its signature or collapse
   it into `mgmtListenAddr` without breaking existing tests.

### Blast-radius mitigation

- All existing router-mode tests use `tempSockPath(t)` filesystem paths.
  `net.SplitHostPort("/tmp/sb-*/m.sock")` fails → auto-detect returns `"unix"`.
  No existing test behavior changes.
- Console mode is unaffected: the `mode == "console"` branch in `mgmtNetwork` and
  in `buildMgmtListener` is preserved exactly as today.
- Control/access modes: default to unix (no change). Neither dials their own
  mgmt socket; they use it only as a listener.

### Loopback-guard applicability

Router-mode TCP management listeners: **NO loopback guard** (Ruling 9, already
ratified 2026-07-15). The `isMgmtLoopbackHost` check is console-only.

### F-1 / F-2 interaction note

F-1 (push is synchronous in admin handler goroutine, can block past sbctl 5s
deadline when a router is unreachable): Ruling 2 / Decision 4 already permit
async background push via the ARCH-01 WaitGroup contract. The implementer will
move `pushWithRetry` calls to a WaitGroup-tracked background goroutine.

F-2 (router binds unix instead of TCP): the fix changes which network
`buildMgmtListener` uses to open the router's management listener. This is
entirely server-side (router). The client (`pushRPC`) always dials TCP; this
does not change.

**The two fixes compose cleanly and are independent.** F-2 must be merged first
(or together with F-1) because without a TCP listener the async push goroutines
would connect-refuse on every attempt. With F-2 in place the listener is TCP; the
F-1 async goroutine dials it successfully. No ordering constraint beyond: "F-2
unblocks F-1's retry path."

### AC-008 correction

The current AC-008 postconditions are:

1. A `router_management_endpoints` entry with `addr: "0.0.0.0:9093"` is accepted
   by `Config.Validate()` without error.
2. Startup INFO log emitted: `"router management listener bound to <addr> ..."`.
3. The INFO log is inspectable in integration tests.

**These postconditions are INSUFFICIENT.** They do not assert that the router
actually binds a TCP listener or that a TCP connection to it succeeds. A router
can emit the INFO log string unconditionally (and currently does, at
`mgmt_wire.go:789–799`) while the listener is still a unix socket. Both existing
AC-008 tests (`TestRouterMgmtListener_NonLoopbackBindAccepted` and
`TestRouterMgmtListener_StartupInfoLog_BindAddress`) can pass while the feature
remains non-functional.

**Corrected AC-008 postconditions (replace the existing three):**

1. **Config acceptance (unchanged):** A `router_management_endpoints` entry with
   `addr: "0.0.0.0:9093"` (non-loopback) is accepted by `Config.Validate()`
   without error (no `isMgmtLoopbackHost` guard — Ruling 9).

2. **TCP bind assertion:** When `management_socket` is set to a `host:port` value
   (e.g. `"127.0.0.1:<ephemeral>"`), `runRouter` opens a TCP management listener
   on that address. Verified by: after `runRouter` starts, `net.DialTimeout("tcp",
   addr, ...)` succeeds (connection accepted or immediately closed — a TCP
   connection is established, not "connection refused").

3. **TCP dial end-to-end (mgmt handshake):** A real `admissionSyncClient` pointing
   at the router's TCP management address can complete the ADR-012
   challenge-response handshake and send an `internal.admission.register` RPC
   that the router's `AdmittedKeySet` receives. This is the functional test that
   the push path works end-to-end.

4. **Startup INFO log (unchanged):** The startup INFO log
   `"router management listener bound to <addr> (ensure firewall policy restricts
   access as appropriate)"` is emitted with the resolved bind address.

5. **Unix default preserved:** When `management_socket` is absent or set to a
   filesystem path, `runRouter` binds a unix socket (existing behavior
   unaffected).

**Corrected test names (replace existing two; add two new):**

| Test name | What it asserts |
|-----------|----------------|
| `TestRouterMgmtListener_NonLoopbackBindAccepted` | Postcondition 1 — config accepts non-loopback addr (unchanged) |
| `TestRouterMgmtListener_StartupInfoLog_BindAddress` | Postcondition 4 — INFO log emitted (unchanged) |
| `TestRouterMgmtListener_TCPBind_ConnectionSucceeds` | Postcondition 2 — `net.Dial("tcp", addr)` succeeds after runRouter starts with a TCP management_socket; verifies listener is genuinely TCP, not unix |
| `TestRouterMgmtListener_TCPBind_PushHandshakeSucceeds` | Postcondition 3 — real admissionSyncClient pushes an internal.admission.register RPC to a runRouter instance started with a TCP management_socket; routerKS receives the entry |

The two new tests (`TCPBind_ConnectionSucceeds`, `TCPBind_PushHandshakeSucceeds`)
are the minimal assertions that close the functional gap. Both use a loopback
`host:port` (e.g. `"127.0.0.1:0"` with `net.Listen("tcp", "127.0.0.1:0")` to
obtain an ephemeral port) to avoid port-conflict flakiness in CI.

`TestAdmissionSync_PushFullSnapshot_AllEntriesPushedToRouter` (AC-009) already
exercises the push handshake against a TCP listener (`startRouterMgmtServerTCP`)
but uses a manually constructed mgmt.Server — it does not exercise the
`mgmtListenAddr` auto-detect path. The new `TCPBind_PushHandshakeSucceeds` test
closes this by going through `runRouter` directly.

### Downstream propagation list (Ruling 10)

The story-writer and PO need the following edits (architect does NOT touch these;
routing is the orchestrator's responsibility):

| Artifact | Edit needed |
|----------|-------------|
| `.factory/stories/S-BL.ADMISSION-SYNC-WIRE.md` | (a) Update `decisions/S-BL.ADMISSION-SYNC-WIRE-rulings.md` binding comment in frontmatter to `v1.3`; (b) replace AC-008 postconditions 1–3 with the corrected five postconditions above; (c) add test names `TestRouterMgmtListener_TCPBind_ConnectionSucceeds` and `TestRouterMgmtListener_TCPBind_PushHandshakeSucceeds` to AC-008; (d) add implementation task: update `mgmtListenAddr` (or `mgmtNetwork`) to auto-detect TCP vs unix on `management_socket`; update `buildMgmtListener` to skip loopback guard for non-console TCP. |
| `BC-2.09.003` (or story Decision 8 section) | Confirm PC-14 postcondition language covers the TCP-bind behavior (not just config acceptance). Story decision section may need a note that "router binds TCP when management_socket is a host:port" is now explicit, not merely implied. |

---

## Ruling 11 — Control-side admission keyset persistence (F-P3-01, now in-scope by human decision)

### Defect confirmed (code-verified)

Adversary pass 3 filed F-P3-01 (MEDIUM, correctness gap). The following facts were
independently verified before issuing this ruling:

| Fact | Location | Verified |
|------|----------|---------|
| `runControl` constructs `ks := admission.NewAdmittedKeySet()` fresh on every startup — no load path | `mgmt_wire.go:1176` | Confirmed |
| `PushFullSnapshot(ctx, ks)` at control startup iterates `ks.AllSVTNEntries()` and immediately hits the empty-keyset early-return | `mgmt_wire.go:1222`; `admission_sync_client.go:376–381` | Confirmed |
| None of the four admin mutation handlers (`makeRegisterHandler`, `makeRevokeHandler`, `makeExpireHandler`, `makeAdminSVTNDestroyHandler`) contains any persist-write call after `m.RegisterKey` / `m.RevokeKey` / `m.ExpireKey` / `m.Destroy` | `admin_handlers.go:277–293`, `:332–343`, `:427–436`, `:973–984` | Confirmed |
| `writeSnapshotAtomic(path string, ks *admission.AdmittedKeySet) error` is implemented, takes path + keyset, atomic write, advisory return — fully reusable | `admission_sync_snapshot.go:214–262` | Confirmed |
| `loadSnapshotFromFile(path string, ks *admission.AdmittedKeySet, w io.Writer) error` is implemented, handles missing→nil, corrupt→E-KEY-002, present+valid→populate | `admission_sync_snapshot.go:276–334` | Confirmed |
| `marshalSnapshot` / `unmarshalSnapshot` use `schema_version:1`, `snapshotFile` struct, base64url pubkeys, RFC3339 expiry — format is self-contained and shareable | `admission_sync_snapshot.go:95–198` | Confirmed |
| BC-2.09.003 PC-13 is explicitly scoped "router-mode only" | `BC-2.09.003.md:247` | Confirmed |

**Net effect of gap:** Every control restart discards all registered keys. `PushFullSnapshot`
on startup pushes nothing. BC-2.05.009 EC-007 ("Control restarts → full-snapshot push corrects
router staleness") cannot hold. Ruling 4 scenario "Control reconnect after detachment" is
inoperable: control forgets its authoritative state on every process exit.

### Config field decision: NEW field, not reuse of `admission_state_file`

**RULING: A new `ControlAdmissionStateFile string` field in `internal/config.Config` with yaml
key `control_admission_state_file`.**

Rationale for NOT reusing `admission_state_file`:
- The two files serve opposite roles: `admission_state_file` is the router's VLR-local **cache**
  of pushed state (written by the router's push handlers on receive, read by the router on
  startup). `control_admission_state_file` is the control daemon's **authoritative source**
  (written by control after each successful mutation, read by control on startup). Merging them
  under one field name would imply they are the same thing to operators.
- BC-2.09.003 PC-13 explicitly scopes `admission_state_file` as "router-mode only." Reusing
  it for control would require amending PC-13's mode scope AND would create an operational
  confusion risk: an operator who misreads config YAML and points control and router at the
  same path would cause the router to corrupt control's authoritative state on the next push.
  Two fields with distinct names make this mistake impossible.
- The mode-separation principle established by PC-13/PC-14 (router-only vs control-only fields)
  is clean and should be preserved. Control-side persistence is a new behavioral surface.

```go
// ControlAdmissionStateFile is the path where the control-mode daemon persists its
// authoritative AdmittedKeySet (S-BL.ADMISSION-SYNC-WIRE Ruling 11).
// Optional — when absent or empty, control does not persist admission state and
// PushFullSnapshot on startup will push an empty keyset (no EC-007 resync guarantee).
// When present, must be a non-empty, non-whitespace path (E-CFG-017).
// Control-mode only — ignored by router/console/access modes.
ControlAdmissionStateFile string `yaml:"control_admission_state_file"`
```

Config validation: if present, the value must be non-empty and not whitespace-only (E-CFG-017),
using the same pattern as E-CFG-015 for `admission_state_file`. No file I/O in `Validate()`
(ARCH-06 §Config purity contract). This is PC-15 in BC-2.09.003.

### Write path (persist-on-mutation)

**RULING: After each successful `SVTNManager.*` mutation, call `writeSnapshotAtomic` with
`cfg.ControlAdmissionStateFile` and control's `AdmittedKeySet`, BEFORE `dispatchPush`.**

Placement in each handler:
1. `m.RegisterKey(...)` / `m.RevokeKey(...)` / `m.ExpireKey(...)` / `m.Destroy(...)` succeeds.
2. **`writeSnapshotAtomic(controlSnapshotPath, ks)` — persist first.**
3. Then `dispatchPush(ctx, wg, func(...) { syncClient.Push*(...)})` — push second (async).

The persist-write is synchronous on the handler goroutine (same as the `m.*` write). It does
NOT move into the `dispatchPush` goroutine — the goal is durability of the authoritative state
independent of whether the push succeeds or the goroutine runs. If `controlSnapshotPath == ""`
(not configured), `writeSnapshotAtomic` is a no-op (it already checks for empty path at
`admission_sync_snapshot.go:216–218`).

**Write failure behavior:** Advisory — log WARN, do not return error to the RPC caller, do not
roll back the `m.*` write. Same policy as the router push-handler snapshot write (BC-2.05.010
PC-2 / Ruling 3). The control-side write has already committed; the snapshot write is a
durability aid, not a transaction participant.

The `ks *admission.AdmittedKeySet` reference is currently threaded into the admin handlers via
`SVTNManager`. To call `writeSnapshotAtomic(path, ks)` in the handlers, the path and the
AdmittedKeySet accessor must be accessible. Two clean approaches:

**Option A (preferred):** inject `controlSnapshotPath string` as a new parameter into
`BuildAdminHandlers`, alongside the existing `syncClient` and `pushWG`. The handlers close
over it. The path is empty string when control is not configured for persistence (no-op path).

**Option B:** inject a `persistFn func()` callback into `BuildAdminHandlers`. The callback
captures path + ks in the closure at the `runControl` call site.

**Recommendation: Option A.** It is simpler, avoids an extra closure layer, and keeps the
handler signature directly readable. The path is just a string.

`BuildAdminHandlers` new signature (conceptual):
```go
func BuildAdminHandlers(
    m *svtnmgmt.SVTNManager,
    ops *mgmt.OperatorKeySet,
    syncClient admissionSyncer,
    controlSnapshotPath string,   // NEW — empty string = no persistence
    pushWG ...*sync.WaitGroup,
) []mgmt.Handler
```

The four write-handler makers (`makeRegisterHandler`, `makeRevokeHandler`, `makeExpireHandler`,
`makeAdminSVTNDestroyHandler`) each gain `controlSnapshotPath string` as a parameter and call
`writeSnapshotAtomic(controlSnapshotPath, m.AdmittedKeySet())` after the successful `m.*` write
and before `dispatchPush`.

`m.AdmittedKeySet()` must expose the underlying `*admission.AdmittedKeySet`. Verify that
`SVTNManager` already has an accessor; if not, one must be added. Note: the snapshot already
calls `ks.AllSVTNEntries()` via `marshalSnapshot` — the accessor need only return the pointer,
no new AdmittedKeySet methods are required.

### Load path (startup before PushFullSnapshot)

**RULING: In `runControl`, after `ks := admission.NewAdmittedKeySet()` and BEFORE constructing
`syncClient` or calling `PushFullSnapshot`, load the control snapshot.**

```go
ks := admission.NewAdmittedKeySet()

// Ruling 11: load persisted control-side admission state before constructing the sync
// client and before PushFullSnapshot. This ensures EC-007 resync is real.
var controlSnapshotPath string
if cfg != nil {
    controlSnapshotPath = cfg.ControlAdmissionStateFile
}
if loadErr := loadSnapshotFromFile(controlSnapshotPath, ks, w); loadErr != nil {
    // Fail-closed on corrupt snapshot (E-KEY-002). Missing file → nil return → continue.
    return fmt.Errorf("runControl: load control admission snapshot: %w", loadErr)
}
// ... then: m := svtnmgmt.NewSVTNManager(ks, daemonPub)
// ... then: syncClient := newAdmissionSyncClient(...)
// ... then: syncClient.PushFullSnapshot(ctx, ks)
```

**Semantics on load:**
- `controlSnapshotPath == ""` (not configured): `loadSnapshotFromFile` is a no-op (returns nil),
  `ks` remains empty, EC-007 guarantee does NOT apply (operator must configure the field to get
  the resync guarantee). This should be documented in the operator guide.
- File missing: `loadSnapshotFromFile` returns nil → empty keyset → fresh install semantics.
  `PushFullSnapshot` is a no-op (correct for fresh install).
- File present + valid: `unmarshalSnapshot` populates `ks` via `RegisterKey` calls. All entries
  are `admitted=false` — this is correct for control (control is the write authority, not the
  challenge-response validator; it never calls `AdmitNode`). The `admitted` field is irrelevant
  on control's keyset. The loaded entries are the registered-key registry only.
- File present + corrupt / unknown schema_version: `loadSnapshotFromFile` returns `E-KEY-002`
  error → `runControl` returns error → daemon exits 1 (fail-closed). This is the correct behavior:
  a corrupt authoritative state file must not be silently ignored; operator must investigate.

The log writer (`w`) is already available in `runControl` — pass it to `loadSnapshotFromFile`
for the INFO log on successful load. The INFO log emitted by `loadSnapshotFromFile` uses the
router-targeted message "switchboard router: admission snapshot loaded: svtn_id=... entries=...".
The implementer SHOULD either reuse this log (acceptable for initial implementation) or update
the log prefix to "switchboard control:" for clarity.

### `admitted` flag on loaded entries

Control is the key-registration authority. It never performs the challenge-response handshake
for peer connections (that is the router's role). `admitted=false` for all loaded entries is
therefore not just acceptable but correct: control's keyset tracks which keys ARE registered
(eligibility), not which connections ARE live (admission state). No special handling needed.

### Serialization format shared with BC-2.05.010

The `snapshotFile` struct (`schema_version:1`, `snapshotSVTN`, `snapshotKey`) is used
unchanged for both the router-side VLR-local snapshot and the control-side authoritative
snapshot. The format is already implemented and tested. No format changes are needed.

The two files have different operational semantics (router's is a cache; control's is
authoritative), but the on-disk representation is identical. This simplifies the implementation
and means a corrupt control snapshot file can be diagnosed with the same tooling as a corrupt
router snapshot file.

### New BC work required

| BC action | Description |
|-----------|-------------|
| Amend BC-2.09.003 | Add PC-15: `control_admission_state_file` optional config field; non-empty when present (E-CFG-017); control-mode only. Same pattern as PC-13. |
| Amend BC-2.05.009 PC-7 | Add qualifier: "The EC-007 resync guarantee holds only when `control_admission_state_file` is configured and the snapshot was successfully written on each prior mutation. When the field is absent, `PushFullSnapshot` pushes an empty keyset (no resync). Operators who require EC-007 MUST configure `control_admission_state_file`." |
| No change to BC-2.05.010 | Remains router-VLR-local snapshot only. The control-side persistence is a new behavior under the new config field and does not alter the router-side BC. |

### Story points impact

Adding control-side persistence adds approximately 2–3 story-points of scope:
- New config field + validation test: ~0.5 pt
- `BuildAdminHandlers` signature extension + per-handler persist call: ~0.5 pt
- Load path in `runControl`: ~0.5 pt
- New AC (control persistence) + test: ~1 pt
- BC amendments (BC-2.09.003 PC-15, BC-2.05.009 PC-7 qualifier): ~0.5 pt

**Recommended revised estimate: 10–11 story points** (was 8). The PO/story-writer should
confirm this estimate when amending the story.

---

## Ruling 12 — Mgmt listener TCP/loopback scope correction (F-P3-02)

### Defect confirmed (code-verified)

Adversary pass 3 filed F-P3-02 (MEDIUM, security scope overreach). The following facts were
independently verified before issuing this ruling:

| Fact | Location | Verified |
|------|----------|---------|
| `mgmtListenAddr` (`mgmt_wire.go:187–199`): after the `if mode == "console"` TCP fast-path, the `net.SplitHostPort` auto-detect fires for ALL remaining modes — `"router"`, `"control"`, `"access"`, and any unknown mode | `mgmt_wire.go:188–198` | Confirmed |
| `buildMgmtListener` loopback guard check: `if mode == "console"` at line 221 — control and access modes with a host:port management_socket reach TCP `net.Listen` with NO loopback guard | `mgmt_wire.go:218–229` | Confirmed |
| `buildMgmtListener` bind-address INFO log: the comment in Ruling 9 requires an INFO log for router-mode TCP binds; no log is emitted for control/access TCP binds in the current implementation | `mgmt_wire.go:201–235` | Confirmed — no INFO log for non-router non-console TCP |
| `mgmtDefaultSocket("control")` returns `"/run/switchboard-control.sock"` — a filesystem path that fails `net.SplitHostPort` — so the default case is safe today | `mgmt_wire.go:154` | Confirmed |
| `mgmtDefaultSocket("access")` returns `"/run/switchboard-access.sock"` — same, safe default | `mgmt_wire.go:147` | Confirmed |

**Attack surface created by over-generalization:** An operator setting
`management_socket: "0.0.0.0:9091"` in a control-mode config silently binds the
`admin.key.*` / `admin.svtn.*` management plane (key registration, revocation, expiry, SVTN
destroy) on all network interfaces. There is no loopback guard, no bind log, and no
documentation of the behavior. `sbctl` has no legitimate need to connect to a control daemon
from a remote host — it is a local administrative tool. The control admin plane MUST fail
closed (loopback-only) unless the operator makes an explicit, documented opt-in.

### Decision: option (b) — keep auto-detect for all modes, apply loopback guard to control/access

**RULING: The TCP auto-detect (`net.SplitHostPort` check in `mgmtListenAddr`) remains
applicable to ALL non-console modes. But `buildMgmtListener` is amended to apply the loopback
guard to control and access modes (in addition to console). Only router-mode TCP listeners
skip the loopback guard (Ruling 9 unchanged). All modes that bind non-loopback TCP MUST emit
the bind-address INFO log.**

Rationale for option (b) over option (a):

- **Option (a) (restrict auto-detect to router only)** would force control and access operators
  who want a loopback TCP management socket (legitimate for container environments where Unix
  sockets are awkward) to set a `host:port` address, which currently gets rejected. Restricting
  auto-detect entirely from control/access forecloses this legitimate use case.
- **Option (b) (keep auto-detect, add loopback guard)** allows control and access to use TCP
  management sockets (e.g., `management_socket: "127.0.0.1:9091"`) while enforcing that the
  address is loopback-only. This matches the console-mode policy (which is also local-only)
  and is the security-correct default: control's admin plane is operator-local.

**Non-loopback TCP for control/access:** Any future need to expose control's admin plane
non-locally must be an explicit architectural decision with its own ruling — not a silent
consequence of setting `management_socket: "0.0.0.0:9091"`. Until such a ruling exists,
non-loopback TCP for control/access is rejected with the same `E-CFG-008` error as console.

### Precise implementation specification

```go
func buildMgmtListener(cfg *config.Config, mode string) (net.Listener, error) {
    network, address := mgmtListenAddr(cfg, mode)
    if network == "unix" {
        ln, err := listenUnixMgmt(address)
        if err != nil {
            return nil, fmt.Errorf("buildMgmtListener: %w", err)
        }
        return ln, nil
    }
    // TCP path.
    // Loopback guard: applies to console (VP-073 / BC-2.07.004 EC-013 / Ruling D),
    // control (Ruling 12: admin plane is operator-local; sbctl connects locally),
    // and access (Ruling 12: same rationale).
    // Router-mode TCP management listeners are NOT loopback-restricted (Ruling 9:
    // control→router push is inherently cross-host; ADR-012 challenge-response is
    // the auth boundary).
    if mode != "router" {
        host, _, splitErr := net.SplitHostPort(address)
        if splitErr != nil {
            return nil, fmt.Errorf("E-CFG-008: management_socket: cannot parse address %q: %w", address, splitErr)
        }
        if !isMgmtLoopbackHost(host) {
            return nil, fmt.Errorf(
                "E-CFG-008: management_socket: %s mode requires a loopback address "+
                    "(127.0.0.1, [::1], or localhost); got: %s",
                mode, address,
            )
        }
    }
    ln, err := net.Listen(network, address)
    if err != nil {
        return nil, fmt.Errorf("buildMgmtListener: %w", err)
    }
    // INFO log for any TCP bind — operator visibility into bound address.
    // For router: log matches Ruling 9 advisory; for console/control/access: loopback
    // is guaranteed above, so the log is informational only.
    return ln, nil
}
```

The `if mode != "router"` check cleanly separates router (no guard) from all other modes
(loopback guard). Future modes default to the stricter policy unless explicitly opted out with
a new ruling — the same fail-closed default the go.md rule 13 requires for security-perimeter
constructors.

**Bind-address INFO log:** `buildMgmtListener` should emit an INFO log after a successful
`net.Listen`, for ALL modes that bind TCP (router, console, control, access). The log string
SHOULD include the mode name and resolved address so operators can confirm what is bound:
```
"<mode> management listener bound to <address>"
```
For router mode this matches Ruling 9's existing advisory ("ensure firewall policy restricts
access as appropriate"). For console/control/access (loopback-only) the advisory suffix is
omitted. The INFO log is written to the daemon's log writer `w`, which is already available
in `runControl`/`runRouter`/`runConsole` — thread it through `buildMgmtListener` as an
`io.Writer` parameter (or use `slog`/`log` if a standard logger is already wired).

**Blast-radius:** Zero impact on existing tests. All existing tests for control, access, and
console modes use filesystem paths (default sockets). Filesystem paths fail `net.SplitHostPort`
→ `mgmtListenAddr` returns `"unix"` → `buildMgmtListener` takes the unix branch — the TCP
loopback guard is never reached. No test behavior changes.

### BC/story propagation from Ruling 12

| Artifact | Edit needed |
|----------|-------------|
| Story `S-BL.ADMISSION-SYNC-WIRE.md` — Non-Goals | Remove or update any Non-Goal bullet that deferred control-side loopback enforcement. The `buildMgmtListener` scope correction is now in-scope per human decision. |
| BC-2.09.003 | Add or amend a postcondition noting that TCP management listeners on control and access modes MUST be loopback-only (E-CFG-008 for non-loopback). Scoped note: router mode is exempt (Ruling 9). |
| Story ACs | Add a new AC (e.g., AC-011) or extend an existing AC: "control-mode or access-mode daemon with `management_socket: "0.0.0.0:<port>"` returns E-CFG-008 at listener bind time; a loopback address (e.g., `127.0.0.1:<port>`) succeeds." This is a new testable postcondition not covered by the existing AC-008 (which tests router mode). |
| Implementation task | Amend `buildMgmtListener`: replace `if mode == "console"` guard with `if mode != "router"`. Add INFO log for all TCP bind paths. |

---

## Ruling 13 — PushFullSnapshot partial-failure gap for revoked entries (F-P6-02)

### The tension: PC-5 (advisory push) vs Invariant 6 (no less-restrictive state)

Adversary pass 6 filed F-P6-02 (MEDIUM). The conflict is real, not illusory.

**PC-5** holds that push failures are advisory — a router that does not receive a push is
"stale," meaning it may be MISSING updates relative to control. Stale = still not more
permissive than before the push attempt.

**Invariant 6** holds an ABSOLUTE prohibition: `PushFullSnapshot` MUST NOT produce a router
state that is LESS RESTRICTIVE than control's authoritative state. Revoked = not active.

**The current spec (BC-2.05.009 v1.3 PC-7c, EC-009) mandates a two-RPC sequence** for each
revoked entry in `PushFullSnapshot`:
1. `internal.admission.register` — creates the entry as active on the router
2. `internal.admission.revoke` — marks it revoked on the router

Both RPCs are issued on independent dials, each advisory-continue on failure. In the
partial-failure scenario where (1) succeeds but (2) fails, the router is left with the key
in a REGISTERED-ACTIVE state. For a fresh router (empty keyset — the EC-008 scenario), the
entry was ABSENT before the push; after the partial push it is PRESENT-AND-ACTIVE. The push
has ACTIVELY made the router less restrictive for a revoked key.

**PC-5's carve-out does not cover this.** PC-5 tolerates "router is missing an update" — the
pre-push state on the router was NOT having that key at all (not admissible); after a partial
push the state is having it as active (admissible). That is the wrong direction. PC-5 says
"stale is ok"; it does not say "creating new incorrect admission is ok."

The adversary is correct: "stale/missing" (PC-5 tolerates) and "actively less-restrictive"
(Invariant 6 forbids) are distinct, and the current two-RPC sequence conflates them under the
partial-failure case.

### Ruling: approach (c) — skip register for revoked entries; send revoke-only RPC

**RULING: For entries that are REVOKED in control's `AdmittedKeySet`, `PushFullSnapshot` MUST
NOT issue `internal.admission.register` at all.** The correct push sequence for a revoked
entry is:

- If the router already has the entry (prior push was partial or it was registered before
  revocation): issue `internal.admission.revoke` to mark it revoked. This is idempotent —
  revoking an already-revoked entry is safe.
- If the router does NOT have the entry (fresh router, empty keyset): issue NOTHING. A missing
  key is not admissible, which is the correct terminal state for a revoked key. The router is
  not less restrictive than control.

**Why approach (c) is correct and sufficient:**

The terminal state that satisfies Invariant 6 for a revoked key is "absent OR revoked" on the
router. Both states are non-admissible. The problem with the two-RPC sequence was that
"register" creates a transiently-admissible state on the router — and if the following "revoke"
fails, that transient state persists. Approach (c) eliminates the transient admissible state
entirely by never issuing the register.

A fresh router with no entry for a revoked key is already in the correct terminal state
(non-admissible). Issuing a register to immediately follow with a revoke provides no
operational benefit — the router never needs to have a revoked key in ACTIVE state. The
register+revoke pattern was chosen in BC-2.05.009 v1.3 to ensure the router's snapshot
captures the revoked key's metadata for auditability; that motivation does not justify
creating a partial-failure window that violates Invariant 6.

**Why approach (a) (amend Invariant 6) is REJECTED:**

Invariant 6 is a security-correctness invariant. Weakening it to accommodate an implementation
pattern that creates a less-restrictive window would undermine the guarantee that
`admin.key.revoke` is durable and effective. The invariant should be STRENGTHENED by choosing
a push strategy that makes violation structurally impossible.

**Why approach (b) (extend wire to carry `revoked bool` in register) is REJECTED for now:**

Approach (b) is more robust in general (single idempotent RPC, no two-RPC gap) and may be
the right direction for a future story. However, it requires changing Ruling 1/2 wire encoding,
the register handler on the router, and `AdmittedKeySet.RegisterKey`'s call contract. Given
that the simpler approach (c) closes the gap without any wire format change and without
touching the router handler, approach (b) is unnecessary complexity for this story. It is
recorded as a FOLLOW-ON option if operational needs (e.g., audit trail of revoked keys on
router snapshot) justify it.

### Precise obligations

#### (i) BC-2.05.009 — Invariant 6 and PC-7 directive for PO

The PO MUST amend BC-2.05.009 as follows:

1. **PC-7c** (revoked entries in full-snapshot push): Replace the current text
   "Issue `internal.admission.revoke` AFTER the `internal.admission.register` in step (a)"
   with:

   > For REVOKED entries: do NOT issue `internal.admission.register`. A fresh router has
   > no entry for the revoked key, which is the correct non-admissible state. If the router
   > already holds the entry (partial prior push or pre-revocation registration), issue
   > `internal.admission.revoke` to mark it revoked. In either case, the router MUST NOT
   > be left with the key registered-as-active. The register+revoke two-RPC pattern is
   > PROHIBITED for revoked entries because a partial failure (register succeeds, revoke
   > fails) leaves the router in a less-restrictive state, violating Invariant 6.

2. **Invariant 6**: Add a clarifying sentence after the existing text:

   > PC-5's advisory-push carve-out covers "router is missing an update" (stale, still
   > not less restrictive than before the push attempt). It does NOT cover "router was
   > actively made less restrictive by a partial push." The two states are distinct.
   > A router missing a revoked key is non-admissible for that key (correct). A router
   > holding a revoked key as active after a partial push is admissible for that key
   > (incorrect). Invariant 6 forbids the latter absolutely; PC-5 is silent on it.

3. **EC-009** (revoked key in snapshot on restart): Replace the current expected behavior:

   > `PushFullSnapshot` MUST NOT issue `internal.admission.register` for the revoked entry.
   > If the router already has the entry, issue `internal.admission.revoke`. If the router
   > has no entry for the key (fresh router), issue nothing — absent = non-admissible =
   > correct terminal state. The router's resulting state is either "absent" or "revoked" —
   > both are non-admissible and satisfy Invariant 6.

These amendments do NOT require a PC-5 change — PC-5 remains valid as written once
Invariant 6 is clarified to draw the "stale/missing vs actively less-restrictive"
distinction explicitly.

#### (ii) Story — directive for story-writer

The story-writer MUST amend S-BL.ADMISSION-SYNC-WIRE as follows:

1. **AC-009 PC-3c** (full-snapshot push, revoked entries): Replace
   "for REVOKED entries, issues `internal.admission.revoke` AFTER register" with:

   > (c) for REVOKED entries: do NOT issue `internal.admission.register`. Issue
   > `internal.admission.revoke` ONLY if the router already holds an entry for this
   > key (i.e., a prior push partially registered it or the key was registered before
   > revocation). If the router has no entry, issue nothing. A missing entry is
   > non-admissible — the correct terminal state for a revoked key. This avoids the
   > register+revoke two-RPC partial-failure window that would leave the key
   > active on a fresh router (Ruling 13 / Invariant 6).

2. **Test name change**: `TestAdmissionSync_PushFullSnapshot_RevokedKeyStaysRevoked`
   is RETAINED but its assertion is strengthened: the test MUST verify that
   `internal.admission.register` is NOT sent for a revoked entry (either by inspecting
   the router's keyset state or by capturing the RPCs issued). A router that starts with
   no entry for the key must end with no entry (not revoked-but-present, not active).

3. **Add new test name**: `TestAdmissionSync_PushFullSnapshot_RevokedKey_RegisterNotSent`
   — asserts that `PushFullSnapshot` for a control keyset containing a revoked entry
   does NOT issue `internal.admission.register` for that entry. Verify via a spy/mock
   `admissionSyncer` or by checking the router's keyset: on a fresh router, after
   `PushFullSnapshot` with a revoked entry, the router's keyset has NO entry for that key.

4. **Add implementation task** (after task 17):
   > 17d. [ ] PushFullSnapshot revoked-entry handling (Ruling 13): for entries where
   > `entry.IsRevoked()` is true, skip `PushRegisterKey`; issue `PushRevokeKey` ONLY
   > if the router already holds the entry (check via prior knowledge of whether the
   > entry was registered, or attempt revoke and treat ErrNotFound on router as success
   > — absent is the correct state). The simpler and safe implementation: skip
   > PushRegisterKey unconditionally for revoked entries; always attempt PushRevokeKey
   > (a revoke against a not-present entry is a no-op on the router — the router handler
   > should treat "key not found" as success since the desired state is "not active").
   > Document the router handler's idempotent behavior for this case. — implementer

#### (iii) Implementer — exact behavior change

**In `PushFullSnapshot` (admission_sync_client.go), the loop body for each entry changes from:**

```
register(entry)
if entry.expiry != 0 { expire(entry) }
if entry.IsRevoked() { revoke(entry) }
```

**To:**

```
if entry.IsRevoked() {
    // For revoked entries: skip register entirely.
    // Attempt revoke in case the router already holds an active entry.
    // If the router has no entry, the revoke is a no-op (absent = correct terminal state).
    revoke(entry)  // advisory; failure logged at WARN
    continue
}
// Active (non-revoked) entry: register, then set expiry if present.
register(entry)
if entry.expiry != 0 { expire(entry) }
```

**In the router-side `internal.admission.revoke` handler (admission_sync_wire.go):**

The handler MUST treat "key not found" as a success response (or at minimum a non-error
advisory). If the revoke RPC arrives for a key that is not in the router's keyset (because the
router never received a register for it — correct for Ruling 13's skip-register path), the
handler must not return an error. It should return success or a logged-WARN advisory, NOT an
error that causes the push client to treat the push as failed. This ensures that for a fresh
router receiving a revoke for an unregistered key, the push completes successfully with the
router in the correct "absent" state.

**No wire format change.** `internal.admission.revoke` args struct and the
`internal.admission.register` args struct are UNCHANGED. Only the call sequence in
`PushFullSnapshot` changes.

### F-P6-01 confirmation

**F-P6-01 (router-side writeSnapshotAtomic concurrent lost-update race) is a PURE
IMPLEMENTATION FIX requiring NO spec change.**

BC-2.05.010 already mandates correct VLR-local snapshot state — PC-1 ("after each successful
push-handler invocation, write the updated snapshot") and the atomic-write invariant (BC-2.05.010
Invariant 1) together require correct serialized snapshot writes. The concurrent lost-update race
(two concurrent push RPCs writing the snapshot without serialization, last-write-wins on a stale
keyset capture) is a code-level concurrency bug: adding a per-snapshot-write mutex in
`wireAdmissionSyncHandlers` (or serializing snapshot writes via a channel or sequential
post-handler callback) fully resolves it without any spec text change. The spec already says
"write the snapshot"; the bug is in the implementation not serializing concurrent writes. No
BC amendment required.

### Downstream propagation list (Ruling 13)

| Artifact | Edit needed |
|----------|-------------|
| `BC-2.05.009.md` | Amend PC-7c (revoked entries: skip register, revoke-only); amend Invariant 6 (add PC-5 vs Invariant 6 distinction); amend EC-009 (revoked key → no register, revoke-only or nothing). PO executes. |
| `S-BL.ADMISSION-SYNC-WIRE.md` | Amend AC-009 PC-3c; strengthen `RevokedKeyStaysRevoked` test assertion; add `RevokedKey_RegisterNotSent` test name; add impl task 17d. Story-writer executes. |
| `admission_sync_client.go` | `PushFullSnapshot` loop: revoked entries skip `PushRegisterKey`, issue `PushRevokeKey` only. Implementer executes. |
| `admission_sync_wire.go` | `internal.admission.revoke` handler: treat "key not found" as no-error (absent = correct terminal state). Implementer executes. |

