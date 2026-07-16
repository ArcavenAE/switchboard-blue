---
artifact_id: S-BL.NODE-ADMISSION-PROVISIONING-RULINGS
document_type: architecture-design
version: "1.0"
status: draft
producer: architect
timestamp: 2026-07-15T00:00:00Z
cycle: cycle-1
related_stories:
  - S-BL.NODE-ADMISSION-PROVISIONING
related_architecture:
  - decisions/identity-cluster-architecture.md v1.2
  - specs/architecture/ARCH-01-core-services.md
  - specs/architecture/ARCH-03-access-node.md
  - specs/architecture/ARCH-08-dependency-graph.md
related_code:
  - cmd/switchboard/access.go
  - internal/config/config.go
  - internal/discovery/discovery.go
  - internal/admission/admission.go
  - cmd/sbctl/client.go
---

# S-BL.NODE-ADMISSION-PROVISIONING: Architecture Rulings v1.0

Ratified mechanism: **Option E** (local self-generation + operator registers
pubkey out-of-band via `admin.key.register`). Ratified 2026-07-15 by human
operator per identity-cluster-architecture.md §7 Disposition table.

This document specifies both coupled facets precisely enough for story
decomposition:

- **Facet (i):** Keypair provisioning — generating/loading the Ed25519 admission
  keypair and surfacing its public half to the operator.
- **Facet (ii):** Daemon-lifecycle wiring — starting `discovery.New` /
  `Discovery.Run()` inside `runAccess`.

---

## 1. Keypair Generation and Persistence

### 1.1 Key File Format

**RULING: PKCS#8 PEM ("PRIVATE KEY" block type).**

Rationale: `sbctl` already accepts PKCS#8 Ed25519 private keys in
`cmd/sbctl/client.go:loadEd25519Key` (E-CFG-010, commit ef1ee1e). That
function handles both OpenSSH `"OPENSSH PRIVATE KEY"` blocks (via
`golang.org/x/crypto/ssh.ParseRawPrivateKey`) and PKCS#8 `"PRIVATE KEY"`
blocks (same parser). The admission key file uses the same format as the
operator key path so that `loadEd25519Key` (or the equivalent primitive
`crypto/x509.ParsePKCS8PrivateKey` + `pem.Decode`) can be reused without
any new parsing library. The config package already imports
`crypto/x509.ParsePKIXPublicKey` for `AuthorizedOperatorKeys`; adding
`crypto/x509.ParsePKCS8PrivateKey` for the private-key load is the same
import.

**On-disk encoding:**

```
-----BEGIN PRIVATE KEY-----
<base64 of PKCS#8 DER encoding of the Ed25519 private key>
-----END PRIVATE KEY-----
```

Generated via `x509.MarshalPKCS8PrivateKey(ed25519PrivKey)` +
`pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8DER})`.

Loaded via `pem.Decode` → `x509.ParsePKCS8PrivateKey` → type-assert to
`ed25519.PrivateKey`.

### 1.2 Config Field

**RULING: New field `AdmissionKeyFile string` with YAML key `admission_key_file`.**

```go
// AdmissionKeyFile is the path to the access daemon's persistent
// Ed25519 admission keypair file (PKCS#8 PEM). Required for access-mode
// operation when the admission identity is needed (S-BL.NODE-ADMISSION-PROVISIONING).
// When absent (empty string), access mode generates a keypair at the
// default path (see below) on first start and persists it there.
// When present, the path is used as-is (tilde expansion follows
// loadEd25519Key conventions).
AdmissionKeyFile string `yaml:"admission_key_file"`
```

**Default path** when `admission_key_file` is absent or empty string:
`/var/lib/switchboard/access-admission-identity.pem`

The default path follows the pattern established by `management_socket`
(mode-specific default applied at daemon startup, not by `Validate()`).
`Validate()` MUST NOT fail if `admission_key_file` is absent — absence
triggers first-run generation to the default path.

**`config.Config.Validate()` extension:** Validate only checks the field
when it is non-empty: if non-empty, it MUST be a non-whitespace string
(same rule as `management_socket`, E-CFG-008 shape). It does NOT attempt
to read or parse the key file — that is daemon startup I/O, not config
validation I/O (ARCH-06 / config purity contract: `Validate()` performs
no I/O).

### 1.3 First-Run vs. Subsequent-Load Semantics

**First run (file absent):**

1. `runAccess` resolves the effective key file path: `cfg.AdmissionKeyFile`
   if non-empty, else the default path.
2. Attempts to `os.Open(path)`. On `os.ErrNotExist`: generate a new
   Ed25519 keypair via `ed25519.GenerateKey(rand.Reader)`.
3. Writes the private key to `path` as PKCS#8 PEM with mode `0600`
   (owner-read-write only). The write MUST be atomic: write to
   `path + ".tmp"`, `os.Chmod(tmpPath, 0600)`, `os.Rename(tmpPath, path)`.
   If the parent directory does not exist, `os.MkdirAll` with `0700`.
4. Derives the public key from the private key for use in
   `discovery.Config.LocalNodeAdmissionPubkey`.
5. Logs a structured message at INFO level: `"admission identity: generated
   new keypair at <path>"` so the operator knows to register it.

**Subsequent start (file present):**

1. `os.Open(path)` succeeds.
2. Read, `pem.Decode`, `x509.ParsePKCS8PrivateKey`, type-assert to
   `ed25519.PrivateKey`.
3. If any step fails → **fail closed**: `runAccess` returns an error
   wrapping the message `"access: load admission keypair: <path>: <reason>"`.
   The daemon refuses to start. This is the same fail-closed posture as a
   bad `listen_addr`.
4. Derives the public key from the private key.

**Corrupt file (present but unparseable):**

Fail closed — same as load failure. The error message explicitly names the
path so the operator can rotate the file.

### 1.4 File Permissions

`0600` (owner read-write, no group/other access). This is enforced at write
time and SHOULD be checked at read time: if the file's permissions are
broader than `0600`, log a WARNING (structured, not fatal): `"admission
identity key file <path> has permissions <mode>: expected 0600; private key
may be exposed"`. This is advisory, not fatal — matching OpenSSH's behavior
for loose key permissions.

### 1.5 Surfacing the Public Key for Operator Registration

The operator needs the public key to run `admin.key.register`. Two paths:

**Path A (primary) — daemon startup log:** On every start (first-run AND
subsequent), `runAccess` logs the admission public key as a base64-encoded
string at INFO level after the key is loaded/generated:

```
access: admission identity pubkey (register with admin.key.register):
  <base64url-no-pad encoding of the 32-byte raw Ed25519 public key>
```

**Path B (future) — sbctl command:** A future `sbctl admin.node.pubkey`
command or similar can retrieve the key from the running daemon over the
management plane. This is OUT OF SCOPE for this story; Path A is sufficient
for the near-term operational workflow.

The log line appears unconditionally so that after a restart, the operator
can recover the pubkey from the daemon log without needing to interact with
the file directly.

---

## 2. Config Wiring

### 2.1 Loaded Private Key and Derived Public Key Flow

After the key file is loaded/generated in `runAccess`:

```
admissionPrivKey  ed25519.PrivateKey    (loaded, never leaves process memory)
admissionPubKey   ed25519.PublicKey     (derived: admissionPrivKey.Public())
```

**Into `discovery.Config.LocalNodeAdmissionPubkey`:**

```go
discoveryCfg := discovery.Config{
    LocalNodeAddr:            localNodeAddr,     // from config or derived
    LocalSVTNID:              svtnID,            // from config
    Router:                   router,            // the shared *routing.Router
    HeartbeatInterval:        0,                 // use default 30s
    LocalNodeAdmissionPubkey: []byte(admissionPubKey),
}
disc := discovery.New(discoveryCfg)
```

`LocalNodeAdmissionPubkey` is the 32-byte raw Ed25519 public key, passed as
`[]byte`. This is the exact shape `transmitAdvertisement` and `Encode` expect
(both accept `[]byte` per `discovery.go:586`).

**Into the `ChallengeResponse` signer (for `S-BL.NODE-IDENTIFY-WIRE`):**

`admissionPrivKey` is retained in the scope of `runAccess` (or a
connect-time goroutine) so that `S-BL.NODE-IDENTIFY-WIRE`'s handshake
can call `ed25519.Sign(admissionPrivKey, challenge.Nonce[:])` to produce
`ChallengeResponse.NonceSig`. This private key is NOT stored in any struct
that crosses a package boundary — it is a local variable in `runAccess`,
passed as a parameter to the future connect-time handshake function.

**Into `newMgmtServer` — no change.** The management identity (`daemonPriv`)
remains the ephemeral keypair generated at line 134 of `access.go`. The
admission identity is distinct from the management identity. They have
separate lifetimes (admission = persistent, management = ephemeral) and
separate trust domains (admission = SVTN network authorization, management =
operator plane authentication).

### 2.2 ARCH-08 Import-Direction Impact

The new `AdmissionKeyFile` field lives in `internal/config` (DAG position 1,
no imports). No new import. The key-file read/write logic lives in
`cmd/switchboard/access.go` (DAG position 18, the top — may import
everything). The `discovery.Config.LocalNodeAdmissionPubkey` field already
exists (`internal/discovery`, position 14). No new packages, no position
changes, no ARCH-08 violation.

---

## 3. Daemon-Lifecycle Wiring

### 3.1 Where `discovery.New` and `Discovery.Run()` Are Invoked

Both are called in `runAccess`, after the management server is started
(ARCH-12 §Daemon Mode Startup) and after the admission keypair is
loaded/generated.

The lifecycle ordering in `runAccess` expands as follows:

```
[existing Phase (a)] newMgmtServer(...)
[existing Phase (b)] wireMetricsHandlers(...)
[existing Phase (c)] serveMgmtServer(ctx, &mgmtWG, mgmtSrv)

[NEW Phase (d)] load/generate admission keypair → admissionPrivKey, admissionPubKey
[NEW Phase (e)] disc := discovery.New(discoveryCfg)
[NEW Phase (f)] wg.Add(1) before go disc.Run(runCtx) — see §3.2
```

Phase (d) happens in `runAccess` before `runAccessWithConnector` is called,
consistent with the pattern used for `buildAccessComponents` — all
pre-connector setup happens in `runAccess`, all post-connect goroutines in
`runAccessWithConnector`.

**Design choice — where to start the Run() goroutine:**

Option X: start in `runAccess` before calling `runAccessWithConnector` (with
its own WaitGroup, joined before/after `runErr`).

Option Y: pass `disc` into `runAccessWithConnector` and start there, tracked
by the same `wg` as the other goroutines, so all goroutines drain together
on ctx cancellation.

**RULING: Option Y — pass `disc` into `runAccessWithConnector`, start and
track with the same `wg`.**

Rationale: `Discovery.Run(ctx)` blocks until ctx is cancelled and returns
`ctx.Err()`. It has the same lifecycle as the other goroutines in
`runAccessWithConnector` (all drain on `<-runCtx.Done()`). Tracking in the
same `wg` means one clean `wg.Wait()` accounts for all goroutines, consistent
with ARCH-01 §Goroutine WaitGroup Contract and BC-2.04.007 PC-2 postcondition
6 (no goroutine leak). Splitting into a separate WaitGroup would require a
second `wg.Wait()` and a second shutdown scope.

`runAccessWithConnector`'s signature gains a `disc *discovery.Discovery`
parameter (or the `discovery.Config` + construction happens inside it — the
story-writer may choose). The key point is that the `Discovery.Run` goroutine
is WG-tracked with the same WaitGroup.

### 3.2 Goroutine WaitGroup Contract (ARCH-01 v1.7 §Goroutine WaitGroup Contract)

Per the contract: `wg.Add(1)` MUST be called in the **caller** BEFORE the
`go` statement. The goroutine body calls `defer wg.Done()`.

```go
// In runAccessWithConnector (or the equivalent):
wg.Add(1)
go func() {
    defer wg.Done()
    if err := disc.Run(runCtx); err != nil && !errors.Is(err, context.Canceled) {
        // Discovery.Run returns ctx.Err() on normal shutdown; log unexpected errors.
        // Do NOT call cancel() or set internalFailure for ctx.Canceled —
        // that is normal shutdown, not a failure.
    }
}()
```

`Discovery.Run` returns `context.Canceled` or `context.DeadlineExceeded` on
normal `ctx` cancellation. These MUST NOT set `internalFailure` or call
`cancel()` — they are clean shutdown signals.

### 3.3 Shutdown/ctx Handling

`Discovery.Run(ctx)` respects `ctx.Done()` and returns `ctx.Err()`. When
`runCtx` is cancelled (SIGTERM/SIGINT or mid-session failure), the discovery
goroutine exits on the next `<-ctx.Done()` select. The same `wg.Wait()` that
joins all other goroutines joins the discovery goroutine. No separate teardown
needed.

### 3.4 Interaction with Existing `runAccess` Lifecycle

The discovery goroutine is additive to the existing lifecycle. The existing
management server WaitGroup (`mgmtWG`) handles the management goroutine
separately and is joined AFTER `runErr` (post-`runAccessWithConnector`). The
discovery goroutine is joined as part of `runAccessWithConnector`'s own `wg`
— consistent with the sweep and frames-dropped tickers.

The `Discovery.Advertise` call site (the session-change trigger, BC-2.03.001
PC-3) is OUT OF SCOPE for this story. This story wires the heartbeat loop
(`Discovery.Run`) and makes the keypair available; the state-change trigger
path (`sess.Publisher` → `Discovery.Advertise`) is a follow-on wiring step
for a future story.

---

## 4. DI-002 Compliance Statement

**The private key never transits the network in this design.**

Specifically:

1. `admissionPrivKey` is generated by `ed25519.GenerateKey(rand.Reader)` on
   the access node's local machine and written to the local filesystem only.
2. The key file is read on subsequent starts and held in process memory as
   `admissionPrivKey ed25519.PrivateKey`. It is not serialized to any wire
   format, not logged, and not passed to any function that transmits to a
   network connection.
3. `discovery.Config.LocalNodeAdmissionPubkey` receives `[]byte(admissionPubKey)`
   — the PUBLIC half only. `transmitAdvertisement` and `Encode` use this to
   compute the HMAC key `routing.DeriveDiscoveryKey(nodeAdmissionPubkey, svtnID)`
   — a derived symmetric key, not the private key.
4. For `S-BL.NODE-IDENTIFY-WIRE` (future): `ChallengeResponse.NonceSig` =
   `ed25519.Sign(admissionPrivKey, challenge.Nonce[:])`. Only the SIGNATURE
   transits — the private key is never serialized. This is the DI-002 invariant
   per `admission.go:202` comment: "Only the signature (a public artefact
   computed by the node locally) is transmitted."

DI-002 is satisfied unconditionally.

---

## 5. Test Strategy Sketch

### 5.1 Unit-Testable Now (no external dependencies)

**Keypair generation:**
- `admission_key_file` absent → key written to default path, mode 0600,
  PKCS#8 PEM parseable.
- `admission_key_file` present but file absent → key written to that path.
- Subsequent start → same key loaded (public key matches first run's pubkey).

**Load fail-closed:**
- Corrupt PEM → error returned, daemon does not start.
- Non-Ed25519 key (e.g., RSA) in PKCS#8 PEM → error returned.
- File readable but mode broader than 0600 → WARNING logged, daemon starts.

**Config wiring:**
- `discovery.Config.LocalNodeAdmissionPubkey` is populated with the 32-byte
  raw pubkey (non-empty, correct length).
- `admissionPubKey` is `admissionPrivKey.Public().(ed25519.PublicKey)`.

**Config.Validate() extension:**
- Non-empty `admission_key_file` with whitespace-only value → validation error.
- Non-empty `admission_key_file` with valid path string → validation passes
  (no file I/O in Validate).

**Goroutine WaitGroup:**
- After `runAccessWithConnector` returns, `wg.Wait()` returns cleanly with
  no leak (same test pattern as sweep/frames-dropped tickers).
- `Discovery.Run` goroutine is joined before the function returns.

### 5.2 Gated on Future Stories

**Full advertisement round-trip** (Discovery.Advertise → multicast → router
receives): requires a live router with a populated `AdmittedKeySet`. This is
the `S-BL.ADMISSION-SYNC-WIRE` + `S-BL.NODE-IDENTIFY-WIRE` integration gate.
Do not attempt to integration-test the full path in this story.

**`ChallengeResponse` signing in the `NODE_IDENTIFY` handshake**: the signing
call is unit-testable (pass `admissionPrivKey` as a parameter to a helper
function), but the production caller (connect-time goroutine) cannot be
exercised end-to-end until `S-BL.NODE-IDENTIFY-WIRE` is delivered.

---

## 6. BC Groundwork for the PO

The PO must author or amend the following behavioral contracts before the
story-writer can write acceptance criteria. This is a groundwork list, not a
writing instruction — the architect is NOT authoring these BCs.

| Action | Target BC(s) | Notes |
|---|---|---|
| Author new BC for `admission_key_file` config field | New BC (suggest BC-2.09.XXX, sequential after current highest in ss-09) | Must cover: field absent → default path, first-run generation semantics, fail-closed on corrupt file, Validate() behavior (no I/O), file permissions 0600, startup INFO log with pubkey |
| Author new BC for `Discovery.Run()` daemon-lifecycle wiring | New BC (suggest BC-2.04.XXX, sequential after current highest in ss-04, or a new BC-2.03.XXX in discovery) | Must cover: goroutine WG-tracked per ARCH-01 §Goroutine WaitGroup Contract, exits on ctx cancel, ctx.Canceled is NOT an internal failure, startup ordering (after mgmt server) |
| Amend BC-2.09.003 or add a new VALIDATE BC | BC-2.09.003 (config validation) or new | Add `admission_key_file` validation rule: non-empty value must be non-whitespace; file is NOT read during Validate |
| Confirm `discovery.Config.LocalNodeAdmissionPubkey` contract | Existing field has a comment in discovery.go but no formal BC | If no BC exists for this field, PO should add one or confirm it is covered under BC-2.03.001/002 |

---

## 7. Story Name Confirmation

**RULING: `S-BL.NODE-ADMISSION-PROVISIONING` — confirmed, no `-WIRE` suffix.**

Rationale: The primary deliverable of this story is the keypair (identity
material), not a wire protocol. While the story does wire `Discovery.Run()`
into `runAccess`, the `-WIRE` suffix in this project consistently identifies
stories whose PRIMARY deliverable is a wire-protocol opcode or frame
exchange (e.g., `S-BL.DISCOVERY-WIRE` = multicast UDP, `S-BL.NODE-IDENTIFY-WIRE`
= challenge-response wire handshake). Keypair provisioning + discovery-sender
lifecycle is not a wire protocol story. The name `S-BL.NODE-ADMISSION-PROVISIONING`
is unambiguous and already established in the identity-cluster-architecture.md
dependency DAG.

---

## 8. Human Flags

**None.** All design choices in this document are architect-resolvable:

- Key file format (PKCS#8 PEM): established precedent from sbctl / E-CFG-010.
- Config field name (`admission_key_file`): consistent with existing field
  naming convention (`management_socket`, `authorized_operator_keys`).
- Default path (`/var/lib/switchboard/access-admission-identity.pem`): follows
  FHS; consistent with the existing Unix socket default path convention.
- Goroutine lifetime (tracked with main `wg` in `runAccessWithConnector`):
  architecturally grounded in ARCH-01 §Goroutine WaitGroup Contract.
- Story name (`S-BL.NODE-ADMISSION-PROVISIONING`): confirmed per §7.
- DI-002: confirmed per §4.

One **design detail left to story decomposition** (not a human flag, but
named for the story-writer):

> The exact signature extension to `runAccessWithConnector` (pass `disc
> *discovery.Discovery` as a parameter, vs. pass `discovery.Config` and
> construct inside, vs. pass `admissionPrivKey ed25519.PrivateKey` and let
> the function build both). The story-writer should choose the form that
> best fits the existing seam pattern (the `connectorIface` injection in
> ARCH-01 ADR-011 v1.5 §HIGH-B). All three are architecturally equivalent;
> this is an implementation detail, not a design decision.

---

## Summary

| Item | Ruling |
|---|---|
| Key file format | PKCS#8 PEM ("PRIVATE KEY" block), consistent with E-CFG-010 / sbctl |
| Config field name | `AdmissionKeyFile string` / YAML `admission_key_file` |
| Default path | `/var/lib/switchboard/access-admission-identity.pem` |
| First-run generation | `ed25519.GenerateKey` → atomic write to key file with mode 0600 |
| Subsequent load | `pem.Decode` + `x509.ParsePKCS8PrivateKey` + type-assert |
| Fail-closed | Missing-or-corrupt key file → daemon refuses to start (non-nil error) |
| Operator pubkey surface | INFO log on every start: base64url-encoded 32-byte raw Ed25519 pubkey |
| `LocalNodeAdmissionPubkey` source | `[]byte(admissionPrivKey.Public().(ed25519.PublicKey))` |
| `ChallengeResponse` signer | `admissionPrivKey` retained in `runAccess` scope, passed as param to future NODE-IDENTIFY-WIRE handshake function |
| `Discovery.Run` placement | `runAccessWithConnector`, tracked in same `wg` as sweep/frames-dropped goroutines |
| WaitGroup contract | `wg.Add(1)` in caller BEFORE `go`, `defer wg.Done()` inside goroutine, per ARCH-01 v1.7 |
| Shutdown | `ctx.Canceled` from `disc.Run` is NOT an internal failure — clean shutdown |
| DI-002 | Private key never transits network — confirmed; only pubkey and signatures transmitted |
| ARCH-08 impact | None — all changes in `config` (position 1, no imports) and `cmd/switchboard` (position 18, top) |
| Story name | `S-BL.NODE-ADMISSION-PROVISIONING` confirmed — no `-WIRE` suffix |
| Human flags | None — all choices architect-resolvable from code and ratified constraints |
| BC groundwork for PO | Four items: new admission_key_file config BC, new discovery-lifecycle daemon BC, BC-2.09.003 amendment, discovery.Config.LocalNodeAdmissionPubkey BC audit |
