---
artifact_id: IDENTITY-CLUSTER-ARCH
document_type: architecture-design
version: "1.2"
status: draft
producer: architect
timestamp: 2026-07-15T00:00:00Z
cycle: cycle-1
modified:
  - 2026-07-15T00:00:00Z # v1.1 — Disposition: both mechanism decisions ratified (NODE-ADMISSION-PROVISIONING → Option E; ADMISSION-SYNC-WIRE → Option A near-term stepping stone). Near-term ADMISSION-SYNC-WIRE scope updated: Option A push RPC + router-side VLR-local admitted-state snapshot (write on receive, load on startup); ruling that router-side persistence is IN SCOPE based on constraint (b) HARD REQUIREMENT. Three binding forward constraints recorded. Section 7 Disposition and Section 8 Forward Architecture (HLR/VLR) added. Section 5 plan updated; Summary updated.
  - 2026-07-15T00:00:00Z # v1.2 — ODO-5 resolved: Section 9 added (multi-router connection model ruling). Dial direction: control dials routers (TCP). Endpoint addressing: static list in config.Config (RouterManagementEndpoints []RouterManagementEndpoint, same structural shape as existing UpstreamRouters). Reconnect/detachment tolerance: retry-with-backoff on control side; router is unaffected by control absence. HLR/VLR compat: static-list near-term is forward-compatible with registration-protocol future — the config field provides a bootstrap list that a future router-registration protocol can augment or replace. No new human flag needed. Summary row added.
related_stories:
  - S-BL.ADMISSION-SYNC-WIRE
  - S-BL.NODE-ADMISSION-PROVISIONING
  - S-BL.NODE-IDENTIFY-WIRE
related_rulings:
  - S-BL.DISCOVERY-WIRE-rulings.md v1.11 (Rulings 4 and 5)
related_specs:
  - specs/architecture/ARCH-04-admission-security.md
  - specs/architecture/ARCH-08-dependency-graph.md
  - specs/behavioral-contracts/ss-01/BC-2.01.008.md
  - specs/behavioral-contracts/ss-03/BC-2.03.001.md
  - specs/behavioral-contracts/ss-03/BC-2.03.002.md
related_code:
  - internal/admission/admission.go
  - internal/config/config.go
  - cmd/switchboard/access.go
  - cmd/switchboard/mgmt_wire.go
  - cmd/switchboard/admin_handlers.go
  - cmd/switchboard/router_control_wire.go
  - cmd/switchboard/main.go
---

# Identity-Cluster Architecture: ADMISSION-SYNC-WIRE, NODE-ADMISSION-PROVISIONING, NODE-IDENTIFY-WIRE

## 1. Cluster Overview

### Root Cause

All three follow-on stories trace to a single structural fact, verified in
`S-BL.DISCOVERY-WIRE-rulings.md` Rulings 4 and 5 against committed code at
`feature/S-BL.DISCOVERY-WIRE` SHA `20b5493`:

> **This codebase has no operational admission-key distribution or provisioning
> system beyond the single control-mode process that received one manual
> `admin.key.register` call.** Everything downstream of that call (challenge-response
> verification in `admission.AdmitNode`, discovery-key derivation via
> `routing.DiscoveryAuthKeyFor`) was built assuming the material would already be
> present where needed. Nothing yet makes it present anywhere except inside the
> one process that first received it.

Concretely:

- `main.go`'s mode switch (`case "router": ...; case "control": ...`) makes router,
  control, console, and access **separate OS processes**, each constructing their own
  `admission.NewAdmittedKeySet()` — independent, permanently disconnected from every
  other mode's admission state (`admission.go` confirms no cross-process sync path
  exists anywhere in `cmd/` or `internal/`).
- `admin.key.register`'s handler (`makeRegisterHandler`, `admin_handlers.go`) is
  wired exclusively into `runControl`; router mode never calls it and structurally
  cannot.
- `admission.AdmitNode` is verification-only against a local `AdmittedKeySet` — it
  looks up `ks.keys[svtnID]` and returns `ErrNotAdmitted` if the entry is absent.
  Called against the router process's always-empty `ks`, it fails unconditionally.
- `runAccess` generates only an ephemeral Ed25519 keypair for its mgmt identity
  (`daemonPriv`). `internal/config.Config` has no admission-keypair field of any
  kind — not even a placeholder. `internal/discovery.New` / `Discovery.Run` (the
  sender/heartbeat loop) have zero production callers anywhere in the repository.

### The Three Stories

| Story | Direction | Root-cause half it closes |
|---|---|---|
| **S-BL.ADMISSION-SYNC-WIRE** | control-mode → router / console / (access if applicable) | Non-control daemons learn OTHER nodes' admitted identities, so they can verify claimed identities against a real, populated keyset |
| **S-BL.NODE-ADMISSION-PROVISIONING** | (external) → access-node process, at node startup | A running access-node obtains and holds ITS OWN admission keypair (pubkey for `Config.LocalNodeAdmissionPubkey`, private key for signing `ChallengeResponse`); the `Discovery.Run()` sender daemon is also wired here since neither can work without the other |
| **S-BL.NODE-IDENTIFY-WIRE** | access-node → router, at connect-time | The wire handshake that VERIFIES a claimed identity: `NODE_IDENTIFY` opcode + `Challenge`/`ChallengeResponse` + `admission.AdmitNode` + `Router.BindInterface` recording `(SVTNID, NodeAddr) → IfaceID` |

The table entries in the column "presumes" and "delivers" (from `rulings.md` v1.11
Ruling 5(c)) are authoritative:

| Story | Presumes | Delivers |
|---|---|---|
| S-BL.ADMISSION-SYNC-WIRE | Nothing new | Non-control daemons learn OTHER nodes' admitted identities |
| S-BL.NODE-ADMISSION-PROVISIONING | Nothing new | A node obtains and holds ITS OWN identity; the sender daemon-lifecycle exists to use it |
| S-BL.NODE-IDENTIFY-WIRE | Both of the above already solved | The connect-time wire handshake that VERIFIES a claimed identity |

### Dependency DAG

```
S-BL.ADMISSION-SYNC-WIRE ───┐
                             ├──► S-BL.NODE-IDENTIFY-WIRE
S-BL.NODE-ADMISSION-       ─┘
  PROVISIONING
```

**At spec (elaboration) level:**
- Both `S-BL.ADMISSION-SYNC-WIRE` and `S-BL.NODE-ADMISSION-PROVISIONING` depend on
  nothing new — they are blockers, not blocked.
- `S-BL.NODE-IDENTIFY-WIRE` depends on BOTH predecessors. Its `depends_on` frontmatter
  must include both once both have story IDs. (Currently `[]` per the stub's own
  changelog note; story-writer action deferred until IDs are allocated.)

**At implementation level:**
- Neither blocker depends on the other — they close different halves of the same
  root-cause gap and can be spec'd and decomposed in any order, or in parallel.
- The `NODE_IDENTIFY` wire mechanics themselves are largely independent of how the
  node obtained its key (the challenge-response wire format, opcode registry row, and
  `BindInterface` map are specifiable now — see Section 4). The only thing that cannot
  be integration-tested end-to-end until both prerequisites land is the full live
  handshake against a real admitted keyset.

---

## 2. ADMISSION-SYNC-WIRE Mechanism Option Analysis

### What the story must deliver

The router-mode (and console-mode) daemon process must have its `AdmittedKeySet`
populated with admitted-key entries before `admission.AdmitNode` and
`routing.DiscoveryAuthKeyFor` can function. Concretely:

- `wireDiscoveryListener` needs to know which SVTN IDs the router serves (presence
  signal — multicast group join addresses).
- `admission.AdmitNode` called from `S-BL.NODE-IDENTIFY-WIRE`'s handshake needs
  the connecting node's `RegisterKey`-written entry to be present and admittable.
- `routing.DiscoveryAuthKeyFor` needs a populated `admittedKeySet` for per-datagram
  HMAC lookup.

The control-mode process is the sole production writer of `RegisterKey` today
(`admin.key.register` → `makeRegisterHandler` → `BuildAdminHandlers` → `runControl`).
Any solution must close the gap between that write and the router's (and console's)
empty keysets without violating ARCH-04/ARCH-08 import-direction rules or SOUL.md §1
(no hidden dependencies, no phone-home).

### Options

---

#### Option A — Push RPC: control pushes to router on every `admin.key.register`

**Mechanism.** `admin.key.register`'s handler, after calling `RegisterKey` on
control-mode's own `AdmittedKeySet`, also sends a lightweight RPC (over the existing
Unix-domain management socket at `/run/switchboard-router.sock`) to the router process
to replicate the `RegisterKey` write. Symmetric calls needed for `RevokeKey`,
`RemoveSVTN`, and `SetKeyExpiry` (the full mutation surface of `AdmittedKeySet`).

**Trust boundary.** The management socket is already authenticated via
`resolveAndVerifyCallerRole`; control calling router over it would need a new
server-side handler on the router process (a new internal admin RPC, not a new CLI
verb). Authentication is straightforward: the caller is control-mode, which the router
can verify via its existing mgmt-plane credentials model. The router's mgmt socket
binds to localhost (`/run/switchboard-router.sock`); control-mode is always co-located.

**Fail-closed behavior.** If the push RPC fails, control should return an error to the
caller of `admin.key.register` — the write is atomic-or-nothing from the operator's
perspective. A silent-succeed-control/fail-router split-brain is worse than a clean
caller-visible error.

**Complexity.** Moderate. Requires a new internal RPC handler on the router process
(new Go file in `cmd/switchboard/`, analogous to `wireRouterControlHandlers`), a new
client call in `admin_handlers.go`, and serialization for the `RegisterKey` arguments
(`svtnID [16]byte`, `pubkey ed25519.PublicKey`, `role KeyRole`). Does not require any
new `internal/` package or ARCH-08 position; this is entirely `cmd/switchboard` scope.

**ARCH-08 import-direction impact.** Clean. `cmd/switchboard` (position 18, the top)
may import everything. No position-14 (`internal/discovery`) or lower-position package
is touched. The new handler calls `admission.RegisterKey` on the router's own
`AdmittedKeySet` — same package already imported.

**Interaction with `admin.key.register`.** The existing `makeRegisterHandler` call
chain gains one additional action; the public API contract for `admin.key.register`
(sbctl verb, BC trace) is unchanged.

**SVTN-presence for `wireDiscoveryListener`.** On every `admin.key.register` call,
control pushes a `RegisterKey` write to router. After the push, the router's
`admittedKeySet` has at least one entry for that SVTN. The router can derive the
multicast address using `discovery.MulticastAddrFor(svtnID)` — the presence signal
`wireDiscoveryListener` needs. This is a natural, no-extra-step side-effect: the first
`admin.key.register` for a given SVTN causes the router to join that SVTN's multicast
group on its next wire-up.

**Stale state risk.** If the router restarts after key registration, its new empty
`AdmittedKeySet` will not automatically repopulate until the next `admin.key.register`
call without a persistence mechanism. A replay/re-send mechanism was initially flagged
as a follow-on concern in v1.0; see Section 7 (Disposition) for the ruling that
elevates router-side persistence to near-term-in-scope via constraint (b) (HARD
REQUIREMENT: network must function when control is detached).

**Architect's read.** Simplest path that preserves the existing model (control is the
authority, router is a consumer). Push is the canonical distributed-systems shape for
"primary notifies replicas on write." With the VLR-local snapshot addition (Section 7
ruling), the stale-on-restart gap is closed within the near-term story's scope —
consistent with the HLR/VLR target end-state (Section 8).

---

#### Option B — Pull/poll: router periodically queries control for admitted-key state

**Mechanism.** The router process, at startup (and on a periodic interval), connects
to the control process's management socket and requests the full admitted-key snapshot
for all SVTNs. Control exposes a new read-only handler (e.g., `admin.keystate.export`)
that returns the full `AdmittedKeySet` contents. The router deserializes and loads it
into its own `AdmittedKeySet`, replacing prior contents.

**Trust boundary.** Control's management socket is already authenticated; the router
authenticating to control is a new direction (router-as-client) that requires the
router to hold credentials the control socket will accept. In the current model,
credentials are operator-supplied via `AuthorizedOperatorKeys` in `config.Config` or
via the daemon's own bootstrap keypair. The router would need one of these, which
currently it does not need (it is a server, not a management-plane client).

**Fail-closed behavior.** On startup, if control is unreachable, the router starts
with an empty keyset — fail-open if it accepts connections, fail-closed if it blocks
connection acceptance until sync completes. The choice is a design decision; fail-open
is unsafe (the router would accept connections it cannot verify), so fail-closed is
correct. This blocks router startup on control availability — an operational dependency
that does not exist today.

**Complexity.** High. Requires: a new read-only export handler on control; a
pull-client in the router; credential management for the router-as-mgmt-client role;
startup sequencing (control must be up before router can start); and a poll loop with
a meaningful interval. Polling also means admission state is eventually consistent, not
immediately consistent — a key registered in control may take up to one poll interval
to appear in the router's keyset.

**ARCH-08 impact.** Clean at the package level — still `cmd/switchboard` scope. But
it introduces a new runtime topology dependency (router requires control to be reachable
at startup and periodically) that has no equivalent today.

**Interaction with `admin.key.register`.** No change to the register handler. But the
eventual-consistency window means `admin.key.register` succeeds from the operator's
perspective before the router can verify the newly-registered node — a user-visible
race condition the operator must work around.

**Architect's read.** Operationally more fragile than Option A (adds a startup
ordering dependency and eventual-consistency window), and more complex to implement.
Not recommended.

---

#### Option C — Shared persistent store (external key-value store or shared file)

**Mechanism.** Control writes `RegisterKey`/`RevokeKey` events to an external
persistent store (e.g., a key-value file, SQLite database, or bolt/bbolt store at a
well-known path). Router (and console and access) reads from the same path on startup
and subscribes to a filesystem watch or polls periodically.

**Trust boundary.** File-system permissions control access; the store is local to the
machine. No new network socket needed. However, concurrent write safety requires file
locking or a database with proper transaction semantics — a new dependency class.

**Fail-closed behavior.** If the store file is missing or corrupted on router startup,
the router starts with an empty keyset — same fail-closed implication as Option B.

**Complexity.** High. Requires choosing and vendoring a persistence library (or
implementing lock-safe file I/O), defining a serialization format for admitted keys,
and implementing the watch/polling loop. This is materially more infrastructure than
Option A.

**ARCH-08 impact.** A new `internal/` package for the persistence layer would need
a DAG position, potentially touching the dependency graph. Alternatively, the
persistence could live entirely in `cmd/switchboard` — but then it cannot be shared
with a future external router binary. Introduces a stored-credential attack surface
(private key material at rest, if the store ever carries private keys — admission
pubkeys are safe, but the design must be careful not to drift toward storing private
keys here).

**Architect's read.** Highest complexity and most new attack surface. The persistence
need (surviving router restarts) is real but can be addressed as an incremental
improvement to Option A (a router-local snapshot write/read) rather than as a
ground-up shared store. Not recommended as the first story's mechanism.

---

#### Option D — In-process shared state (access + router run in the same process, mode-separation removed for these two modes)

**Mechanism.** Collapse router and control into a single process, or at least share the
`AdmittedKeySet` instance between the two modes without cross-process serialization.

**Trust boundary.** Trivial — same process, same heap. No serialization.

**Complexity.** Very high, and invasive. `main.go`'s mode-switch design is a
load-bearing architectural decision (ARCH-01, process model); reversing it requires
rearchitecting the daemon lifecycle, the management socket per-mode design (ARCH-12),
and the CLI interface. This is a multi-story refactor, not a single follow-on story.

**Security profile.** Collapsing the process boundary between the control authority
(which accepts `admin.key.register` over an authenticated management socket) and the
router (which accepts untrusted node connections over the data-plane socket) removes a
privilege-separation layer that the current architecture intentionally maintains.

**Architect's read.** Not a viable option for this cluster. Ruled out — the process
separation is a feature, not a limitation.

### Recommended Mechanism: Option A (push RPC)

Option A is the **recommended mechanism** for `S-BL.ADMISSION-SYNC-WIRE`. It is the
only option that:
- Keeps the control process as the sole admission authority (preserving the existing
  security model).
- Makes admission state immediately consistent at the router on every `admin.key.register`
  call (no eventual-consistency window).
- Does not require a new external dependency class (file storage, external KV store).
- Does not add a startup ordering dependency between daemon modes.
- Is narrow enough to be a single, well-scoped story of similar complexity to prior
  `cmd/switchboard` wiring stories.

**RATIFIED (2026-07-15): Option A selected — near-term stepping stone.** See Section 7
(Disposition) for the full ratified decision record, near-term scope refinement (Option A
+ router-side VLR-local admitted-state snapshot), the detached-restart persistence ruling,
and the three binding forward constraints. The HLR/VLR target end-state is documented in
Section 8.

**Scoping note (Ruling 4(d)):** Task 3 of `S-BL.DISCOVERY-WIRE` requires only the
SVTN-presence signal (which SVTN IDs does this router serve?) to pick multicast groups
to join. The full admitted-key material (pubkeys, roles, `FrameAuthKey`) is needed by
`S-BL.NODE-IDENTIFY-WIRE`'s `AdmitNode` call. Whether `S-BL.ADMISSION-SYNC-WIRE`
delivers both (full material sync, satisfying both needs in one story) or is split into
a narrower "SVTN-presence-only" story plus a "full material sync" successor is a
PO/architect scoping call — flagged, not resolved here.

---

## 3. NODE-ADMISSION-PROVISIONING Mechanism Option Analysis

### What the story must deliver

A running access-mode process (`runAccess`) must have, at runtime:

1. **Its own Ed25519 admission keypair** — the public half for
   `Config.LocalNodeAdmissionPubkey` (used in `Encode`/`transmitAdvertisement`), the
   private half for signing `ChallengeResponse` in `S-BL.NODE-IDENTIFY-WIRE`.
2. **A live `Discovery.Run()` goroutine** — the sender/heartbeat loop that calls
   `Advertise`/`transmitAdvertisement` on a schedule. Currently, `discovery.New` and
   `Discovery.Run` have zero production callers anywhere in the repository (`grep`
   across all non-test files returns no matches). Neither facet can be delivered alone:
   a `Discovery.Run()` caller without an identity cannot derive `DiscoveryAuthKey`
   (fails with `ErrMissingNodeAdmissionPubkey`); a provisioned keypair without
   `Discovery.Run()` being wired into `runAccess` never advertises.

The two coupled facets are (per Ruling 5(b)):
- **(i)** How a running access-node process obtains and holds its own admission Ed25519
  keypair (pubkey for `Config.LocalNodeAdmissionPubkey`; private key for
  `ChallengeResponse` signing in `S-BL.NODE-IDENTIFY-WIRE`).
- **(ii)** The daemon-lifecycle wiring of `discovery.New`/`Discovery.Run()` into
  `runAccess`, absent today for the reason above and for the basic absence of a caller.

Note: `access.go`'s own comment already names a broken promise ("persistent key_file
wiring is deferred to S-6.02") that `config.Config`'s field list confirms was never
delivered. The "deferred to S-6.02" comment names a story that shipped (PR #34, Wave 5)
without delivering this; the obligation was dropped, not completed.

### Options

---

#### Option E — Local self-generation + external pubkey registration (operator-assisted)

**Mechanism.** On first startup (or when no keypair file is present), the access-mode
daemon generates a new Ed25519 keypair locally, writes it to a persistent key file
(e.g., `/var/lib/switchboard/access-identity.pem` — a configurable `key_file` path in
`config.Config`). On subsequent startups, it loads the key file. The public half is
registered out-of-band by the operator via `admin.key.register` against the control
daemon. The node's own private key never leaves the file (DI-002).

**Key-file format.** PKCS#8 PEM (consistent with `sbctl`'s existing PKCS#8 Ed25519
acceptance, per `S-BL.CLI-SURFACE-COMPLETION` PR #119 / `E-CFG-010`). No new parsing
library required — `crypto/x509.ParsePKCS8PrivateKey` is already used in the
codebase.

**Trust boundary.** The key file is a local filesystem artifact. Access is controlled
by OS-level file permissions. DI-002 ("private key never transits the wire") is
trivially satisfied — the node never sends its private key anywhere; only
`ChallengeResponse.NonceSig` (the signature) transits. This is the same model
`examples/README.md` uses for the control daemon's own bootstrap keypair.

**Fail-closed behavior.** If the key file is absent and generation fails (disk full,
permission denied), `runAccess` should return an error and refuse to start — same
fail-closed posture as missing `listen_addr`. If the key file exists but is malformed,
same. Never start without a usable identity.

**Complexity.** Low-to-moderate. Requires: a new `AdmissionKeyFile` (or equivalent)
field in `config.Config`; file-generation and file-loading logic in `runAccess` (or a
new helper); and daemon-lifecycle wiring for `discovery.New`/`Discovery.Run()` once
the keypair is available. The `admission.KeyRole` enum is already wired to `RoleAccess`
for access nodes; no new admission logic is needed.

**Interaction with `S-BL.ADMISSION-SYNC-WIRE`.** The node generates its own public key
locally; the operator registers that pubkey against the control daemon via
`admin.key.register`. Once `S-BL.ADMISSION-SYNC-WIRE` distributes that registration
to the router, the router's `AdmittedKeySet` will have the entry the `NODE_IDENTIFY`
handshake needs. The two stories are genuinely independent and can proceed in parallel
up to integration testing.

**ARCH-08 impact.** Clean — `config.Config` is position 1 (DAG root, no imports).
Adding a field there does not change any dependency. `runAccess` (in `cmd/switchboard`,
position 18, the top) reads the file.

**Architect's read.** The most autonomy-preserving option. The node owns its own
identity material; no network call is needed to provision it. Consistent with
SOUL.md §1 (user sovereignty, no hidden dependencies). Consistent with DI-002 (private
key never transits). The operator out-of-band registration step is a genuine UX cost
but is the existing model for all other admitted-key registrations in this system —
no special-casing for the access node.

---

#### Option F — Operator-provisioned key material via `config.Config` (static keypair in config)

**Mechanism.** The operator generates an Ed25519 keypair offline (e.g., `sbctl key gen`)
and places the PEM-encoded private key in the config file under a new
`admission_key_file` field (a path) or directly as `admission_key` (inline PEM). The
node loads this at startup.

**Trust boundary.** Private key material is in a config file — potentially more
accessible than a dedicated key file with tighter permissions, but practically similar.
This is the `AuthorizedOperatorKeys` pattern applied to the node identity side.

**Complexity.** Slightly lower than Option E (no generation step; key file is a
straight load). The `config.Config.Validate()` call must validate the key is a valid
Ed25519 PKCS#8 PEM — same validation as `AuthorizedOperatorKeys` already does for
public keys.

**Interaction with `S-BL.ADMISSION-SYNC-WIRE`.** Same as Option E — operator registers
the pubkey out-of-band.

**Architect's read.** Nearly identical to Option E in security profile and complexity,
but places private key material inside the main config file alongside
`AuthorizedOperatorKeys` — which are public keys. This conflates public and private key
material in one file, making audits more error-prone. Option E's separate key file with
distinct permissions is preferable. The two are not mutually exclusive; Option F can be
offered as an alternative syntax alongside Option E's key-file path.

---

#### Option G — Enrollment/bootstrap protocol: router issues a provisioned keypair to the node on first contact

**Mechanism.** On first connection to the router (before `NODE_IDENTIFY`), a new
`NODE_ENROLL` opcode causes the router to generate a new Ed25519 keypair on the node's
behalf, return the private key to the node, and simultaneously register the public key
in control via `S-BL.ADMISSION-SYNC-WIRE`'s push RPC.

**Trust boundary.** VIOLATES DI-002. DI-002 ("private key never transits the wire")
is an unconditional domain invariant. No amount of TLS wrapping satisfies this — the
invariant's comment text (`admission.go:202`) explicitly states "Only the signature
(a public artefact computed by the node locally) is transmitted." A provisioning
protocol that delivers a private key over a network connection is a fundamental
architectural violation, not a complexity trade-off.

**Architect's read.** Not a viable option. Ruled out unconditionally — this is a DI-002
violation.

---

#### Option H — Derive node identity from existing daemon keypair (reuse `daemonPriv`)

**Mechanism.** `runAccess` already generates an ephemeral Ed25519 keypair (`daemonPriv`)
for its mgmt identity. The admission keypair is set equal to `daemonPriv`.

**Trust boundary.** `daemonPriv` is explicitly documented as ephemeral ("identity
changes across restarts"). The access node's admission identity must be stable across
restarts — a node that re-registers a new pubkey on every restart defeats the
admission model (each restart appears as a new, unadmitted node; the operator must
run `admin.key.register` after every node restart).

**Interaction with `S-BL.ADMISSION-SYNC-WIRE`.** Would require re-registering the new
pubkey on every restart, and `S-BL.ADMISSION-SYNC-WIRE` distributing the updated entry
to the router before the node can reconnect. This is an operational anti-pattern.

**Architect's read.** Not viable as a primary mechanism. The ephemeral-versus-persistent
mismatch makes this a correctness bug, not a simplification. Documented so it is not
rediscovered as a "free" solution.

### Recommended Mechanism: Option E (local self-generation + external pubkey registration)

Option E is the **recommended mechanism** for `S-BL.NODE-ADMISSION-PROVISIONING`. It:
- Satisfies DI-002 unconditionally (private key never leaves the node).
- Produces a stable, restart-persistent identity without any network-call dependency.
- Is consistent with the existing model for other admitted keys (operator runs
  `admin.key.register` to register a pubkey against a SVTN).
- Is the lowest-complexity option that delivers both coupled facets (keypair +
  daemon-lifecycle wiring) without new architectural dependencies.

**RATIFIED (2026-07-15): Option E approved.** See Section 7 (Disposition) for the
ratified decision record. No forward constraints on this decision; proceed to
elaboration.

**Scoping note:** The daemon-lifecycle wiring of `discovery.New`/`Discovery.Run()` into
`runAccess` is a coupled facet of this story (facet (ii)), not a separate story — because
without a stable keypair, the `Discovery.Run()` loop cannot function (`transmitAdvertisement`
fails closed with `ErrMissingNodeAdmissionPubkey`), and without the loop, a provisioned
keypair has no production use. Both facets must land together to make the access node's
advertisement path live end-to-end.

---

## 4. NODE-IDENTIFY-WIRE Readiness

### What can be spec'd now (independent of mechanism decisions)

`S-BL.NODE-IDENTIFY-WIRE`'s core wire mechanics are largely specifiable independent
of how the node obtained its key or how the router received the admitted-key entry.
The following are deterministic given the existing codebase and the fanout-options
document's Option 1 selection:

**Wire format — specifiable now:**
- `control_type = 0x04` (`NODE_IDENTIFY`) is the next free value per BC-2.01.008 v1.2
  Invariant 3 (append-only, sequential assignment after `0x03`). The registry-table row
  addition to BC-2.01.008 Postcondition 2 can be written now — same precedent as the
  `DISCOVERY_RELAY = 0x03` row Ruling 3(g) added.
- The `NODE_IDENTIFY` frame header is fixed 4 bytes (Invariant 5 / DI-007), same as
  every other `control_type`. The payload carries the node's Ed25519 public key (32
  bytes). After the router receives `NODE_IDENTIFY`, it sends a `Challenge` frame
  (opcode TBD in the same registry, or a sub-field); the node responds with a
  `ChallengeResponse` frame carrying `NonceSig`. The exact byte layout (nonce encoding,
  signature encoding, whether `Challenge` and `ChallengeResponse` use additional
  `control_type` discriminators or are part of a sub-protocol multiplexed within the
  established `0x04` control session) is an architect elaboration needed at scheduling
  time — **flagged as Open Design Obligation 2 in the story stub**.
- The `GenerateChallenge` → `Challenge` → `ChallengeResponse` → `AdmitNode` sequence
  is already designed, tested (13 call sites in `_test.go`), and uses only primitives
  in `internal/admission` — zero changes to that package.

**Router-side binding — specifiable now:**
- On `AdmitNode` success, a new `Router.BindInterface(svtnID [16]byte, nodeAddr [8]byte, ifaceID routing.InterfaceID)` method (exact name and file left to decomposition) records `(SVTNID, NodeAddr) → IfaceID` in a new map alongside `nodeConn`. This is the binding that `S-BL.DISCOVERY-WIRE`'s AC-017/AC-018/Task 6 gate on.
- `Router.BindInterface` is a pure addition to `internal/routing` — no ARCH-08
  position change needed. It does not require any `internal/admission` import from
  `internal/routing` (already imports `admission` at position 5 per the DAG).

**Test coverage — specifiable now:**
- Successful handshake (admitted key, correct nonce signature) → binding recorded.
- Wrong-SVTN path → `AdmitNode` returns `ErrNotAdmitted`.
- Revoked-key path → `AdmitNode` returns `ErrKeyRevoked`.
- Replayed-nonce path → `AdmitNode` returns `ErrNonceReplay`.
- The last three paths are already covered by `admission`'s own test suite; the
  wire-transport wrapper needs its own tests for the new opcode codec.

### What must wait for the two prerequisite mechanism decisions

- **Full integration test (handshake against a real admitted keyset):** Until
  `S-BL.ADMISSION-SYNC-WIRE` lands, the router's `AdmittedKeySet` is always empty in
  production, so `AdmitNode` returns `ErrNotAdmitted` unconditionally. The integration
  path cannot be exercised end-to-end before that story.
- **`ChallengeResponse` signing in production code:** Until `S-BL.NODE-ADMISSION-PROVISIONING`
  lands, no production caller has a private key to sign with. The node-side handshake
  logic can be written as a function that accepts the private key as a parameter, but
  its production caller (`runAccess` or a new connect-time goroutine) cannot invoke it
  until the keypair is present.
- **Open Design Obligation 3 (re-identify / rebind semantics):** Whether a second
  `NODE_IDENTIFY` frame from an already-bound connection overwrites or errors; whether
  the prior connection is torn down on reconnect with the same identity. These depend
  in part on what event-driven hooks are available (e.g., connection teardown
  notification) — which itself depends on what `S-BL.ADMISSION-SYNC-WIRE` wires for
  connection-lifecycle events.
- **Open Design Obligation 4 (handshake timeout):** Whether an unbound connection is
  closed after N seconds if `NODE_IDENTIFY` is not received. Requires an architect
  ruling on timeout values and error codes.

### Recommended elaboration ordering

1. **Write Open Design Obligation 1 (BC-2.01.008 opcode registry row)** — this requires
   no mechanism decision. A product-owner can add the `NODE_IDENTIFY = 0x04` row to
   BC-2.01.008 Postcondition 2 today.
2. **Write Open Design Obligation 2 (challenge-transcript wire format)** — the architect
   can elaborate the byte layout for the `NODE_IDENTIFY` opcode and its
   `Challenge`/`ChallengeResponse` sub-frames independent of the mechanism decisions.
   This is a pure wire-format design question (how to encode 32-byte nonce, 64-byte
   signature, 32-byte pubkey inside a `control_type`-framed payload).
3. **Defer Open Design Obligations 3 and 4** (re-identify semantics, handshake timeout)
   until `S-BL.ADMISSION-SYNC-WIRE`'s mechanism is decided — they depend on connection-
   lifecycle event availability.
4. **Decompose into tasks** only after obligations 1 and 2 are resolved and the
   mechanism decisions for the two prerequisites are made.

---

## 5. Recommended Plan

### Story-elaboration order

**Immediate (no gate — both confirmed standing):**

1. **BC-2.01.008 opcode row** — product-owner adds `NODE_IDENTIFY = 0x04` row to
   BC-2.01.008 Postcondition 2. Same precedent as `DISCOVERY_RELAY = 0x03`. This
   unblocks Open Design Obligation 1 for `S-BL.NODE-IDENTIFY-WIRE`. Effort: trivial.

2. **NODE-IDENTIFY-WIRE wire format elaboration** — architect resolves Open Design
   Obligation 2 (challenge-transcript byte layout). Produces a concrete byte spec for
   the `NODE_IDENTIFY` frame and its `Challenge`/`ChallengeResponse` sub-frames. This
   is independent of the mechanism decisions for the two prerequisites. Effort: a
   focused architect ruling, similar in size to a single `S-BL.DISCOVERY-WIRE` ruling
   section.

**Ready to elaborate (both mechanism gates cleared — 2026-07-15):**

3. **Elaborate ADMISSION-SYNC-WIRE** — mechanism ratified as Option A + router-side
   VLR-local admitted-state snapshot (see Section 7 ruling). **Open Design Obligation 5
   is resolved** (Section 9): control dials routers via TCP; endpoint addressing uses a
   new `RouterManagementEndpoints []RouterManagementEndpoint` field in control-mode
   `config.Config` (same structural shape as `UpstreamRouters`); reconnection is
   retry-with-backoff on the control side with the router stateless w.r.t. control
   presence. All pre-conditions for writing acceptance criteria are now satisfied.

4. **Elaborate NODE-ADMISSION-PROVISIONING** — mechanism ratified as Option E (local
   self-generation + operator registers pubkey out-of-band). No forward constraints on
   this decision. Items: new `AdmissionKeyFile` config field, file-generation/load
   logic in `runAccess`, daemon-lifecycle wiring for `discovery.New`/`Discovery.Run()`.

**After both elaborated and delivered:**

5. **Extend S-BL.NODE-IDENTIFY-WIRE `depends_on`** — story-writer adds both
   `S-BL.ADMISSION-SYNC-WIRE` and `S-BL.NODE-ADMISSION-PROVISIONING` to the
   `depends_on` frontmatter once both have story IDs.

6. **Deliver the cluster in dependency order:**
   - Wave N: ADMISSION-SYNC-WIRE and NODE-ADMISSION-PROVISIONING (can parallelize if
     team capacity allows — they are independent).
   - Wave N+1 (or N if both land before wave close): NODE-IDENTIFY-WIRE.

### Can any story proceed to per-story delivery in parallel?

- **ADMISSION-SYNC-WIRE** and **NODE-ADMISSION-PROVISIONING**: Yes, in parallel with
  each other. Both mechanism gates are now cleared. They are structurally independent
  (opposite directions of the same root cause; no shared code surfaces beyond what
  ARCH-08 position 18 already permits).
- **NODE-IDENTIFY-WIRE**: No — must wait for both predecessors to be delivered before
  integration testing is meaningful. The wire codec and opcode registry can be
  spec'd and stubbed before delivery, but the Red Gate → Green Gate cycle is only
  meaningful once `AdmitNode` can succeed against a real router-side keyset and the
  node process holds a real private key.

### PO BC-groundwork checklist

| Action | Story | Owner | Gate |
|---|---|---|---|
| Add `NODE_IDENTIFY = 0x04` row to BC-2.01.008 PC-2 | S-BL.NODE-IDENTIFY-WIRE | product-owner | None — do now |
| Author new BCs for admission-sync RPC + VLR-local snapshot (Option A ratified) | S-BL.ADMISSION-SYNC-WIRE | product-owner | Gate cleared — ready |
| Author new BCs or amend BC-2.09.003 for `key_file` config field (Option E ratified) | S-BL.NODE-ADMISSION-PROVISIONING | product-owner | Gate cleared — ready |
| Extend S-BL.NODE-IDENTIFY-WIRE `depends_on` once both IDs exist | S-BL.NODE-IDENTIFY-WIRE | story-writer | After both predecessors have IDs |

---

## 6. Story Name Confirmation

Both working names coined in the rulings document are **confirmed and recommended**:

- **`S-BL.ADMISSION-SYNC-WIRE`** — accurately names the story's direction (sync of
  admission state to the router's wire-receiving process) and is consistent with the
  `S-BL.*-WIRE` naming convention for wire-protocol stories in this project.
- **`S-BL.NODE-ADMISSION-PROVISIONING`** — accurately names the story's direction
  (provisioning of a node's own admission identity) and is distinct in naming from
  SYNC (inbound from control vs. self-provisioned outbound). "Provisioning" is not
  a wire-protocol story in the same sense; consider whether the suffix `-WIRE` is
  appropriate — the story does wire `Discovery.Run()` into `runAccess`, but its
  primary deliverable is the keypair, not a protocol. Either `S-BL.NODE-ADMISSION-PROVISIONING`
  (current working name, no `-WIRE` suffix) or `S-BL.NODE-ADMISSION-WIRE` (if the
  team prefers uniform `-WIRE` suffixes) is acceptable; flag for PO/architect
  confirmation at scoping time.

Both names carry the disclosed-confidence caveat from the rulings document
("not multi-option-vetted; PO/architect to confirm name + scope"). This document
elevates both to **recommended** (architect-analyzed and mechanism-ratified as of
2026-07-15) — no outstanding human gates on these names.

---

## 7. Disposition

Both mechanism decisions have been ratified by the human operator (2026-07-15).
The Option A selection carries three binding forward constraints and a near-term scope
refinement (ruled below).

### Ratified Decisions

| Story | Decision | Mechanism | Date | Notes |
|---|---|---|---|---|
| S-BL.NODE-ADMISSION-PROVISIONING | **Option E approved** | Local self-generation + operator registers pubkey out-of-band via `admin.key.register` | 2026-07-15 | Architect-recommended mechanism; no modification to scope or constraints. Proceed to elaboration. |
| S-BL.ADMISSION-SYNC-WIRE | **Option A selected — near-term stepping stone** | Push RPC (control→router on each `RegisterKey`/`RevokeKey`/`RemoveSVTN`/`SetKeyExpiry` write) + router-side VLR-local admitted-state snapshot | 2026-07-15 | Stepping stone toward HLR/VLR end-state (Section 8). Near-term scope includes router-side persistence (ruling below). Three binding forward constraints: (a) MANY routers, (b) control-detachment resilience (HARD), (c) HLR/VLR target. |

### Ruling: Router-Side Persistence Scope (ADMISSION-SYNC-WIRE)

**The question.** Pure Option A push-on-write does not satisfy constraint (b) (control-
detachment resilience). A router that restarts while control is detached comes up with an
empty `AdmittedKeySet` and no path to repopulate — the network is dark for that router
until control reattaches. Does the near-term story need router-side persistence of the
last-synced admission state (a VLR-like local cache surviving router restart and control
detachment) IN SCOPE? Or is detached-restart a cleanly-deferrable known limitation,
with only live-push + in-memory cache shipping now?

**Ruling: IN SCOPE for the near-term story. No human flag needed.**

Rationale:

1. **Constraint (b) is stated as a HARD REQUIREMENT.** "The network will need to function
   when the control node is not attached / has detached from the network." Router restart
   during control detachment is not an edge case; it is a routine operational event
   (hardware failure, software updates, OS reboots, rolling deployments). An architecture
   that fails this requirement on the most common restart scenario does not satisfy the
   hard requirement in any meaningful sense — labeling it a "known limitation" defers a
   hard requirement, not an edge case.

2. **The persistence mechanism is additive to Option A, not a new coordination class.**
   The concern with Option C (shared persistent store) was that it introduced an external
   bidirectional read/write coordination protocol between control and router. Router-side
   VLR persistence has neither property: the snapshot file is owned entirely by the router
   (only the router writes it, only the router reads it). Control's push RPC is unchanged.
   No new inter-process coordination is required. The structural analogy is Option E's
   key-file approach — each process owns and persists its own state within its own domain.
   A router-local admitted-key snapshot is a smaller, structurally cleaner addition than
   anything Option C proposed.

3. **The HLR/VLR framing confirms the direction.** A VLR that resets on restart is not
   functioning as a VLR; it is an in-memory relay. Making the router's VLR-role durable
   is precisely what the near-term Option A story should be, consistent with the stated
   target end-state (Section 8). Shipping without persistence would require undoing the
   design constraint in the next story rather than building forward.

4. **Complexity delta is small.** One new `config.Config` field (`admission_state_file`
   path), one write operation in the router-side push handler (serialize and persist the
   `AdmittedKeySet` snapshot on each received push), and one conditional read in `runRouter`
   startup (deserialize and load if file is present and valid). Standard file I/O,
   consistent with Option E's key-file precedent. This remains a single, well-scoped
   `cmd/switchboard` wiring story — the same complexity band as Option A alone.

**Near-term ADMISSION-SYNC-WIRE scope (refined):**

Option A push RPC + router-side VLR-local admitted-state snapshot:

- Control pushes to each router's management endpoint on every `RegisterKey`, `RevokeKey`,
  `RemoveSVTN`, and `SetKeyExpiry` write (full mutation surface of `AdmittedKeySet`).
- Router writes snapshot to `admission_state_file` path (configured in `config.Config`)
  on each received push. Format: serialized representation of the current `AdmittedKeySet`
  contents; format/encoding TBD at elaboration time (JSON, CBOR, or binary — must be
  deterministic and forward-compatible with the HLR/VLR replication model).
- Router loads snapshot from `admission_state_file` on startup (if present and valid).
- **Missing file:** start with empty keyset and wait for a control push to populate
  (fresh install semantics — the router cannot populate itself without control's first
  push). This is correct and expected; it does not violate constraint (b) because a
  fresh-install router has never had state to recover.
- **Corrupt/unreadable file:** fail-closed (refuse to start with corrupt state) — same
  posture as other startup validity checks in this codebase.
- Control's push behavior (push-RPC handler, call sites in `admin_handlers.go`) is
  unchanged by the addition of persistence; the router's handler simply writes the
  snapshot after applying the push.

**Open Design Obligation 5 (multi-router connection model) — flagged for elaboration:**

Constraint (a) (MANY routers) requires the ADMISSION-SYNC-WIRE elaboration to specify
how control discovers and addresses multiple router management sockets. The current
co-location assumption (Unix-domain socket at `/run/switchboard-router.sock`) does not
generalize:

- Control is described as a CLIENT that attaches to the network (constraint (b)), not
  a permanently-resident peer of every router.
- A production network has multiple routers, potentially not co-located with control.

Elaboration must choose among: (a) enumerate router endpoints in `config.Config`
(static list — simplest, no dynamic discovery), (b) define a router-registration
protocol (routers announce their management endpoint to control on attach — dynamic,
consistent with "control is a client"), or (c) an alternative scheme. This is an
**architect ruling at elaboration time** — it does not block beginning elaboration,
but must be resolved before acceptance criteria can be finalized. The near-term
story's scope (write + read handler, serialization, config field) can proceed in
parallel while this sub-question is open.

---

## 8. Forward Architecture: HLR/VLR Admission-State Distribution

### Motivation and Analogy

The human operator explicitly framed Option A (push RPC) as a near-term stepping stone,
not the target architecture. The target is modeled on the cellular network Home Location
Register / Visitor Location Register (HLR/VLR) pattern. This section records that
direction durably so future work is built forward toward it, and so a future agent does
not mistake the near-term Option A + snapshot mechanism for the final design.

**Cellular HLR/VLR mapping to Switchboard:**

| Cellular component | Switchboard component | Role |
|---|---|---|
| **HLR** (Home Location Register) | Control node | Authoritative registry of all admitted identities. Sole write authority for `RegisterKey`, `RevokeKey`, `RemoveSVTN`, `SetKeyExpiry`. Source of truth for the admission-key set. |
| **VLR** (Visitor Location Register) | Each router | Local cache of admitted-key/SVTN state for the SVTNs this router currently serves. Answers `AdmitNode` and `DiscoveryAuthKeyFor` queries without HLR (control) availability. Durable across router restarts and control detachment. |
| HLR→VLR synchronization | Near-term: push RPC + VLR-local snapshot file. Future: distributed replication protocol or admission-state DB. | State propagation from authoritative source to local caches, delivering eventual consistency in the target model. |

### Three Binding Forward Constraints

These constraints were stated by the human as part of the Option A near-term selection.
They are binding on all future design work that touches admission-state distribution.

**(a) MANY routers.** The design must not assume a single router or a fixed
control↔router pairing. A production network has multiple routers. Control communicates
with all of them. Every design artifact at any layer of the admission-state distribution
stack (connection model, sync protocol, state format, failure modes) must be consistent
with arbitrary router count.

**(b) CONTROL-DETACHMENT RESILIENCE (HARD REQUIREMENT).** Control is a CLIENT that
attaches to the network — it is not a permanently-wired peer of every router. Routers
must keep functioning (serving discovery, admitting nodes, enforcing admission state)
across control detachment — both across runtime detachment (in-memory state suffices)
and across router restart during detachment (durable VLR state required). This is a
hard functional requirement, not a quality-of-life improvement. Every story in the
admission-state distribution cluster must be evaluated against this requirement.

**(c) HLR/VLR as the target end-state.** The long-term admission-state distribution
architecture is an HLR/VLR-style system where:
- Control (HLR) is the authoritative admission registry. It is the sole authority for
  all `AdmittedKeySet` mutations, operating as a service rather than a co-resident daemon.
- Each router (VLR) holds a locally durable, eventually-consistent cache of the admitted
  state for the SVTNs it serves. The VLR answers admission queries without HLR
  availability.
- The synchronization mechanism (near-term: push RPC + file snapshot) evolves toward a
  proper replication protocol or distributed admission-state store as the fleet scales.

### Relationship to Option A Near-Term Scope

Option A + router-side VLR-local snapshot is designed to be upgrade-compatible with the
HLR/VLR end-state:

- The router's local snapshot IS the VLR's durable state. The near-term implementation
  uses a local file; the HLR/VLR implementation uses a replicated database — but the
  router's role (maintain local cache, answer queries without HLR) is unchanged between
  the two.
- Control's push-on-write is the near-term analog of HLR→VLR synchronization. The
  push-on-write model is replaced by a replication protocol in the HLR/VLR design, but
  the control→router directionality and the "control is the authoritative writer" invariant
  are preserved throughout.
- The `AdmittedKeySet` abstraction (`internal/admission`) need not change between near-term
  and HLR/VLR — only the population mechanism (push RPC vs. replication protocol) changes.
  This is a clean separation: the data model is stable; the distribution mechanism evolves.

### Future Work Candidate

The HLR/VLR target is a named future work item, not an open question to be resolved in
the current cluster. Future naming candidates:

- **Story**: `S-BL.ADMISSION-HLR-VLR` or `S-BL.ADMISSION-STATE-REPLICATE`
- **Architecture frontier note**: a `_kos/nodes/frontier/` entry (or equivalent in this
  project's tracking system) capturing the distributed-admission-state architecture
  direction, the three binding constraints, and the cellular HLR/VLR mapping

This section does NOT create a story stub or spec for the HLR/VLR work. That
decomposition happens after the near-term story (ADMISSION-SYNC-WIRE with Option A +
VLR-local snapshot) has been delivered and the team has operational evidence about which
replication patterns matter in practice.

---

## 9. ODO-5 Resolution: Multi-Router Connection Model

### Code topology baseline (verified at develop@d249f88)

Before reasoning about the ODO-5 options, three facts about the current code
must be established, because they constrain which connection shapes are
structurally available:

**Fact 1 — Every daemon is a server only, never a client, on the management plane.**
`runRouter`, `runControl`, `runConsole`, and `runAccess` each call
`newMgmtServer`/`serveMgmtServer` to LISTEN on their respective sockets.
Zero daemon-to-daemon management-plane client code exists anywhere in
`cmd/switchboard/`. There is no `mgmt.NewClient`, no `net.Dial` to a peer's
management socket, no cross-mode RPC. The management plane today is a
one-layer fan-out: external callers (sbctl) → daemon management socket.

**Fact 2 — Router management sockets are Unix-domain only in the current code.**
`mgmtNetwork("router")` returns `"unix"` (`mgmt_wire.go:159`). The default
path is `/run/switchboard-router.sock` (`mgmt_wire.go:146`). Unix-domain
sockets are local to a single machine — no remote-machine router can be
reached over a Unix socket. Any control→router push RPC in a multi-machine
topology requires TCP (or an equivalent network transport), meaning the router
management endpoint for Option A must be a network address, not just a socket
path, in the many-routers case.

**Fact 3 — `config.Config` has a structural twin: `UpstreamRouters`.**
The existing `UpstreamRouters []UpstreamRouter` field (`config.go:130–132`)
enumerates a static list of TCP address strings that a router dials for PE
mode. The structural pattern for "control needs to know the addresses of
multiple routers" is exactly this pattern inverted: a `RouterManagementEndpoints
[]RouterManagementEndpoint` list in the control-mode config enumerating the
management addresses of the routers it should push to. The parsing, validation
(`validateHostPort`), and SIGHUP-reload infrastructure for such a list is
already present in `config.go`; no new parsing primitive is required.

### Dial direction ruling

**RULING: Control dials routers. Control is the client on the management push path.**

Reasoning:

The three binding constraints from Section 8 must be taken together:

- Constraint (b) says control is "a CLIENT that attaches to the network." Read
  literally, this describes control's relationship to the data plane — control
  attaches and detaches; the network must survive without it. This language
  does NOT settle the dial direction on the management push path independently.
  Both shapes are compatible with "control is a client on the data plane" while
  taking opposite dial directions on the management plane. The constraint is not
  self-settling on dial direction; it requires explicit reasoning.

- For the **router-registration shape (routers dial control):** routers would
  need to know control's management address at startup and maintain a persistent
  outbound connection to it for the duration that control is attached. When
  control detaches, the router's outbound connection closes; when control
  reattaches, the router must re-register. This creates a persistent
  router→control connection lifecycle that outlasts individual push events. Two
  structural problems arise:
  
  1. **Startup ordering dependency reintroduced.** A router that must dial
     control on startup and cannot proceed until the connection is established
     reintroduces the startup ordering problem that Section 2's Option B analysis
     ruled out (paragraph: "blocks router startup on control availability — an
     operational dependency that does not exist today"). The registration shape
     can be made non-blocking (router starts without connecting to control, and
     registers when convenient), but then the first control-attach after a fresh
     router startup requires the router to initiate the connection — which is
     fine, but means the router must also hold the control endpoint address in
     its config. That is equivalent in static-config terms to control holding
     the router endpoint address.
  
  2. **Persistent connection to a detachable client.** Control can detach at
     any time. A router holding a persistent connection to control must detect
     disconnection and tear it down, then detect reconnection and re-establish
     it. This is a reconnect/reattach loop owned by the router — non-trivial
     state that currently does not exist anywhere in `runRouter` (which has no
     outbound client connections except `upstreamdial.Connector` for the PE
     data plane).

- For the **control-dials-routers shape (control is the management-plane client):**
  Control holds a list of router management endpoints. When control starts (or
  reattaches), it dials each router in its list, establishes an authenticated
  connection, and pushes the current admission state. When a push completes,
  control may either hold the connection idle (for future push events) or close
  it and re-dial on the next push event. Control's detachment (process exit or
  network departure) means the connection closes from the control side;
  routers are completely unaffected — their management socket server continues
  running, waiting for the next connection. Crucially:
  
  - **Routers have no reconnect logic to write.** The router's management
    socket is already a passive listener (from `runRouter`'s call to
    `newMgmtServer`). No new router-side client state is needed.
  - **Detachment tolerance is free.** When control detaches, connections close.
    When control reattaches (new process, same config), it re-dials and re-pushes.
    The router's VLR-local snapshot (Section 7) handles the interval between
    control's last push and re-attachment — exactly as intended.
  - **Startup independence preserved.** A router can start with no control
    present. Its management socket server is ready. When control arrives, it
    dials in and pushes. The router does not need control to be up first.

**Conclusion:** Control dials routers. The control-dials-routers shape satisfies
all three binding constraints cleanly and requires no new state machine in
`runRouter`. The registration shape introduces reconnect complexity at the router,
creates latent startup-ordering risk, and offers no advantage for the near-term
scope. Ruled out for the near-term story.

### Endpoint addressing mechanism ruling

**RULING: Static list in `config.Config` on the control-mode config — a new
`RouterManagementEndpoints []RouterManagementEndpoint` field, same structural
shape as `UpstreamRouters`.**

Options and reasoning:

**(a) Static list in `config.Config`.**
`UpstreamRouters []UpstreamRouter` already establishes this pattern in the
same file (`config.go:130–132`). Adding `RouterManagementEndpoints
[]RouterManagementEndpoint` is a direct structural parallel: each entry carries
an `Addr string` validated as a TCP `host:port`. Validation (`validateHostPort`),
SIGHUP-reload (`config.LoadFile` + `config.Validate`), and zero-value semantics
(empty list = no routers to push to) are all already handled by the existing
infrastructure. From the operator's perspective, this is the same config
discipline as `upstream_routers` — explicit, auditable, no dynamic discovery.

**(b) Router-registration protocol.**
As analyzed above (dial direction), this moves the endpoint-discovery problem
into a registration handshake that requires a persistent connection and
reconnect logic. For the near-term story, where control has a small, known
set of routers, a registration protocol is operationally equivalent to a static
list but structurally more complex. The registration protocol is the correct
direction for large, dynamic fleets — but that is the HLR/VLR end-state work,
not the near-term story.

**(c) Zero-configuration local-only path (Unix socket).**
The existing co-location assumption (`/run/switchboard-router.sock` on the same
machine) works for single-router deployments where control and router share a
host, but structurally cannot scale to many routers on different machines.
Retaining Unix socket support as a zero-config option for single-machine
deployments alongside the TCP list is reasonable — when `RouterManagementEndpoints`
is empty and control and router are co-located, control can fall back to the
local Unix socket path. This zero-config path should be explicitly documented
but is not load-bearing for multi-router support.

**Near-term: static TCP list, with optional Unix socket fallback for
co-located single-router deployments.**

The config field name and wire contract:

```yaml
# In the control-mode config (or a shared config when all modes share one file):
router_management_endpoints:
  - addr: "10.0.0.2:9090-mgmt"  # or the configured management socket TCP address
  - addr: "10.0.0.3:9090-mgmt"
```

Note: "TCP address for the router's management endpoint" means the control-mode
static list enumerates TCP `host:port` strings, NOT Unix paths. For co-located
routers, the operator points the router's management socket at a loopback TCP
address (`management_socket: 127.0.0.1:9093`) and control's list references
that address. This preserves the ability to co-locate control and router on
the same machine without requiring them to always share a process or filesystem.

### Reconnection and detachment-tolerance ruling

**RULING: Retry-with-backoff on the control side; router is stateless with
respect to control presence.**

Behavior specification:

1. **Control startup / reattach.** `runControl` iterates `RouterManagementEndpoints`,
   dials each router's management TCP address, and pushes the current admission
   state snapshot. Dial failures are retried with bounded exponential backoff.
   After N consecutive failures to a given endpoint, control logs a warning and
   stops retrying until the next push event or a SIGHUP reload. Control does NOT
   fail to start if routers are unreachable — it is the authoritative registry
   regardless of whether any router is currently reachable.

2. **Per-write push.** On each `admin.key.register` / `admin.key.revoke` /
   `admin.key.expire` / `admin.svtn.destroy` write, control pushes the delta
   (or the full snapshot, whichever is simpler — TBD at story decomposition)
   to each configured router endpoint. Temporary push failures on individual
   writes are logged and retried; they do NOT roll back the write to control's
   own `AdmittedKeySet`. The admitted-key write and the push are independent
   actions: control is the authority, routers are consumers. A failed push means
   the router is temporarily stale; the router's VLR-local snapshot (Section 7)
   bridges the gap.

3. **Control detachment (process exit or network departure).** Connections to
   routers close from the control side. Routers receive EOF, discard the
   connection, and continue serving from their VLR-local snapshot. No router
   state machine change is needed — the router's management socket server
   passively waits for the next connection.

4. **Router restart during control detachment.** The router loads its VLR-local
   snapshot on startup (Section 7 ruling). It does not need control to be present
   to start. Control's next push will bring the snapshot up to date.

5. **Control reconnect.** When a new control process starts (or the operator
   runs `sbctl admin.key.register` against a restarted control), control
   re-dials all configured router endpoints and pushes the current state. The
   router applies the push, overwrites its VLR-local snapshot, and is current.

**What does NOT need to be implemented in the near-term story:**

- A persistent idle connection from control to each router between push events
  (dial-on-demand per push event is simpler and sufficient for the near-term
  throughput; idle connection with heartbeat is a follow-on if operational
  evidence shows reconnect latency is a problem).
- An explicit "control is attached" / "control is detached" protocol message
  on the router side (the router's behavior is identical in both states).
- A full-sync-on-connect handshake (near-term: push the full current snapshot
  on every control startup; incremental delta sync is a future optimization).

### HLR/VLR forward-compatibility

The static-list near-term mechanism is explicitly designed as a stepping stone
to the router-registration protocol that the HLR/VLR end-state will use:

- **Data model stability.** The `AdmittedKeySet` abstraction is unchanged. The
  VLR-local snapshot format (introduced in this story) is the carrier format
  that the replication protocol will also use. Nothing the near-term story
  touches must be un-done.
- **Config field additive.** `RouterManagementEndpoints` in the control config
  serves as the bootstrap list. A future registration protocol augments or
  replaces this list dynamically at runtime — the config list is never wrong
  to have, it just becomes a seed rather than the sole source of truth.
- **Dial direction preserved.** The HLR/VLR model has control as the
  authoritative registry (HLR) pushing to distributed router caches (VLR). The
  dial direction ruling above (control dials routers) is consistent with
  HLR→VLR replication where the HLR pushes to VLRs. A future protocol where
  VLRs subscribe and pull from the HLR would reverse the dial direction — but
  the current ruling is compatible with either, because the near-term story
  establishes the data model and the snapshot format, not the replication
  protocol shape.
- **No dead-end decisions.** Every near-term choice (static TCP list, dial-on-
  demand per push, full-snapshot push) is the simplest form of its corresponding
  HLR/VLR feature. The upgrade path is: static list → registration protocol,
  dial-on-demand → persistent replication connection, full-snapshot → incremental
  delta. None of these upgrades requires changing the router-side handler or the
  VLR-local snapshot format.

### Human flags

None. The dial-direction choice (control dials routers) and the endpoint-addressing
mechanism (static TCP list in control config) are architect-resolvable questions
fully grounded in the code topology and the binding constraints. No product or
deployment implication requires human input to settle: the static-list approach
is the conservative, operationally obvious near-term choice, forward-compatible
with the dynamic registration protocol the HLR/VLR end-state demands.

### Acceptance criteria prerequisites now satisfied

ODO-5 is resolved. ADMISSION-SYNC-WIRE's acceptance criteria can now be
finalized. The story-writer has all inputs:

1. **Mechanism:** Option A push RPC + VLR-local snapshot (Section 7 ruling).
2. **Dial direction:** Control dials routers.
3. **Endpoint addressing:** `RouterManagementEndpoints []RouterManagementEndpoint`
   in `config.Config`; each entry is a TCP `host:port`; validated by `validateHostPort`;
   config field is CONTROL-MODE ONLY (routers do not read this field).
4. **Reconnection behavior:** retry-with-backoff on the control side; dial-on-demand
   per push event (near-term); router stateless with respect to control presence.
5. **Serialization format:** TBD at story decomposition (JSON, CBOR, or binary;
   must be deterministic and forward-compatible with HLR/VLR replication model).
6. **Unix socket fallback:** control may treat an empty `RouterManagementEndpoints`
   list as a signal to fall back to the local Unix socket path
   (`/run/switchboard-router.sock`) for co-located single-router deployments
   (zero-config convenience path, not load-bearing for multi-router support).

---

## Summary

| Item | Value |
|---|---|
| Root cause | Single control-mode process holds all admission state; no cross-process distribution exists |
| Stories in cluster | S-BL.ADMISSION-SYNC-WIRE, S-BL.NODE-ADMISSION-PROVISIONING, S-BL.NODE-IDENTIFY-WIRE |
| Stories with no prerequisites | S-BL.ADMISSION-SYNC-WIRE, S-BL.NODE-ADMISSION-PROVISIONING |
| ADMISSION-SYNC-WIRE mechanism (near-term) | **Option A ratified** — push RPC (control→router on each `RegisterKey`/`RevokeKey`/`RemoveSVTN`/`SetKeyExpiry` write) + router-side VLR-local admitted-state snapshot (write on receive; load on startup). Near-term stepping stone toward HLR/VLR end-state (Section 8). |
| NODE-ADMISSION-PROVISIONING mechanism | **Option E approved** — local self-generation + operator registers pubkey out-of-band. |
| Near-term constraint (b) addressed by | Router-side VLR-local admitted-state snapshot. IN SCOPE for near-term ADMISSION-SYNC-WIRE story (Section 7 ruling). |
| Forward architecture target | HLR/VLR admission-state distribution — control = HLR (authoritative registry), router = VLR (local cache). Three binding constraints documented in Section 8. |
| Open Design Obligation 5 | **RESOLVED** — Section 9. Dial direction: control dials routers (TCP). Endpoint addressing: static `RouterManagementEndpoints []RouterManagementEndpoint` list in control-mode `config.Config` (same structural shape as `UpstreamRouters`). Reconnection: retry-with-backoff on control side; router stateless w.r.t. control presence. Unix-socket fallback for empty list (co-located single-router, zero-config). HLR/VLR compatible: static list is a stepping stone; upgrade path is static list → registration protocol without changing router-side handler or snapshot format. No human flag needed. |
| NODE-IDENTIFY-WIRE can spec wire format now | Yes — challenge-transcript byte layout + BC-2.01.008 opcode row independent of mechanism decisions |
| NODE-IDENTIFY-WIRE delivery gates on | Both predecessors delivered |
| Can ADMISSION-SYNC and NODE-ADMISSION-PROVISIONING parallelize? | Yes — structurally independent, no shared code surfaces; both mechanism gates now cleared |
| Immediate no-gate items (confirmed standing) | (1) PO adds `NODE_IDENTIFY = 0x04` to BC-2.01.008 PC-2; (2) architect elaborates NODE-IDENTIFY-WIRE challenge-transcript wire format (Open Design Obligation 2) |
| Recommended first elaboration step | Either ADMISSION-SYNC-WIRE or NODE-ADMISSION-PROVISIONING — both gates cleared. All ADMISSION-SYNC-WIRE pre-conditions including ODO-5 are now resolved; acceptance criteria can be finalized. |
| Story names confirmed? | Yes (both working names recommended; NODE-ADMISSION-PROVISIONING -WIRE suffix left to PO/architect preference) |
