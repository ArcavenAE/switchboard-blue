---
artifact_id: S-BL.NODE-ADMISSION-PROVISIONING
document_type: story
level: ops
story_id: S-BL.NODE-ADMISSION-PROVISIONING
epic_id: E-7
title: "Node admission-identity provisioning: Ed25519 admission keypair at admission_key_file + Discovery.Run wired into access daemon lifecycle"
status: draft
producer: story-writer
timestamp: 2026-07-15T00:00:00Z
modified:
  - date: 2026-07-15
    version: "1.0"
    change: >
      Initial full decomposition — admission keypair provisioning (BC-2.09.004) and
      Discovery.Run daemon-lifecycle wiring (BC-2.04.008) per architect rulings
      decisions/S-BL.NODE-ADMISSION-PROVISIONING-rulings.md v1.0. Leaf prerequisite
      for S-BL.NODE-IDENTIFY-WIRE. 8 ACs, 5 points.
version: "1.0"
phase: 2
epic: E-7
wave: backlog
priority: P1
scope_phase: PE
points: 5
estimated_points: 5
inputs:
  - 'decisions/S-BL.NODE-ADMISSION-PROVISIONING-rulings.md'
  - 'specs/behavioral-contracts/ss-09/BC-2.09.004.md'
  - 'specs/behavioral-contracts/ss-04/BC-2.04.008.md'
  - 'specs/behavioral-contracts/ss-09/BC-2.09.003.md'
input-hash: "05213d5"
traces_to: 'decisions/S-BL.NODE-ADMISSION-PROVISIONING-rulings.md'
behavioral_contracts:
  - BC-2.09.004
  - BC-2.04.008
  - BC-2.09.003
verification_properties: []
bc_traces:
  - BC-2.09.004
  - BC-2.04.008
  - BC-2.09.003
vp_traces: []
subsystems: [deployment-operations, session-access]
target_module: "cmd/switchboard"
architecture_modules:
  - internal/config
  - internal/discovery
  - cmd/switchboard
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: []
blocks: [S-BL.NODE-IDENTIFY-WIRE]
rulings_doc: "decisions/S-BL.NODE-ADMISSION-PROVISIONING-rulings.md"
estimated_days: null
assumption_validations: []
risk_mitigations: []
acceptance_criteria_count: 8
inputDocuments:
  - 'decisions/S-BL.NODE-ADMISSION-PROVISIONING-rulings.md'   # v1.0 — BINDING. Option E mechanism ratified: local self-generation, PKCS#8 PEM, admission_key_file config field, first-run generation semantics, fail-closed on corrupt, file permissions 0600, startup INFO log with base64url pubkey, Discovery.Run Option Y (same WaitGroup in runAccessWithConnector), ARCH-08 compliance, DI-002 compliance.
  - 'specs/behavioral-contracts/ss-09/BC-2.09.004.md'         # v1.0 — admission_key_file provisioning: Validate() postconditions PC-1/PC-2 (E-CFG-014), first-run generation PC-3/PC-4 (atomic write, mode 0600), load PC-5/PC-6 (fail-closed E-KEY-001), startup INFO log PC-7.
  - 'specs/behavioral-contracts/ss-04/BC-2.04.008.md'         # v1.0 — Discovery.Run daemon-lifecycle wiring: WG-tracked per ARCH-01 PC-1/PC-2, ctx.Canceled clean-shutdown PC-3, no goroutine leak PC-4, startup ordering PC-5.
  - 'specs/behavioral-contracts/ss-09/BC-2.09.003.md'         # v2.1 — PC-12: admission_key_file validation (E-CFG-014, no I/O in Validate); context for E-CFG-014 error code shape.
---

# S-BL.NODE-ADMISSION-PROVISIONING: Node Admission-Identity Provisioning — Ed25519 Admission Keypair + Discovery Sender Lifecycle

## Narrative

- **As an** access node in an SVTN
- **I want to** start with a persistent Ed25519 admission identity (a keypair at a configurable path,
  generated atomically on first run and loaded on subsequent runs), with the discovery heartbeat
  sender (`Discovery.Run`) correctly wired into my daemon lifecycle from the same startup sequence
- **So that** I can advertise my SVTN presence over the multicast wire (unblocking
  `S-BL.DISCOVERY-WIRE`'s `LocalNodeAdmissionPubkey`-dependent HMAC path) and my admission private
  key is available for the connect-time challenge-response handshake in `S-BL.NODE-IDENTIFY-WIRE`

## Context

`S-BL.DISCOVERY-WIRE-rulings.md` v1.11 Ruling 5 (F-DWIP1-001) and `decisions/identity-cluster-architecture.md` v1.2
verified two compounding gaps in the codebase:

1. `internal/config.Config` has no admission-keypair field of any kind; `runAccess` generates only
   an ephemeral `daemonPriv` keypair for its mgmt identity — unrelated to admission.
2. `internal/discovery.New` / `Discovery.Run` have zero production callers anywhere in the
   repository — the sender daemon-lifecycle wiring into `runAccess` was never built.

Both gaps gate `S-BL.NODE-IDENTIFY-WIRE`'s `ChallengeResponse` signing step (needs the node's
own admission private key) and `S-BL.DISCOVERY-WIRE`'s production advertisement path (needs
`discovery.Config.LocalNodeAdmissionPubkey` populated from a loaded keypair). This story closes
both gaps in a single delivery.

**Scope boundary.** This story does NOT implement `S-BL.NODE-IDENTIFY-WIRE`'s opcode, wire
codec, or `Router.BindInterface`. It does NOT implement admission-state sync from control to
router (`S-BL.ADMISSION-SYNC-WIRE`). It makes the node's own admission keypair and the discovery
sender available; the downstream stories use them.

## Previous Story Intelligence (MANDATORY)

| Predecessor | Lesson carried forward |
|-------------|------------------------|
| `S-BL.DISCOVERY-WIRE` (Tasks 1-5 DELIVERED PR #123 @ d249f88) | `discovery.Config.LocalNodeAdmissionPubkey []byte` already exists on disk and `transmitAdvertisement` returns `ErrMissingNodeAdmissionPubkey` when it is empty. This story MUST populate it from the loaded/generated keypair before `disc.Run` starts — not after, and not as a zero-value. |
| `S-W5.01` (merged PR #31) | The existing `Config.Validate()` exhaustive-error-collection pattern (all errors gathered before returning, same as PC-10/PC-11 for `management_socket`/`authorized_operator_keys`) is the required shape for PC-12 (`admission_key_file` E-CFG-014). Match it exactly. |
| `S-7.04-FU-DRAIN-WIRE` (DELIVERED PR #120 @ f73676d) | `wg.Add(1)` MUST be called in the caller scope BEFORE the `go` statement (ARCH-01 §Goroutine WaitGroup Contract). BC-2.04.007's sweep+frames-dropped pattern is the established precedent; the discovery goroutine is the fourth goroutine in this WaitGroup alongside those. |
| `sbctl E-CFG-010` fix (commit ef1ee1e) | `loadEd25519Key` already handles both OpenSSH and PKCS#8 Ed25519 PEM blocks via the same parse path. The admission key file uses PKCS#8 `"PRIVATE KEY"` (consistent with E-CFG-010, not OpenSSH `"OPENSSH PRIVATE KEY"`); the load code can share `x509.ParsePKCS8PrivateKey` from the same stdlib imports already in `internal/config`. |

## Adjudicated Design Decisions

Transcribed from `decisions/S-BL.NODE-ADMISSION-PROVISIONING-rulings.md` v1.0 (binding).

### Decision 1 — Key file format: PKCS#8 PEM

`"PRIVATE KEY"` block (not OpenSSH `"OPENSSH PRIVATE KEY"`). Generated via
`x509.MarshalPKCS8PrivateKey(ed25519PrivKey)` + `pem.EncodeToMemory`. Loaded via `pem.Decode` →
`x509.ParsePKCS8PrivateKey` → type-assert to `ed25519.PrivateKey`. Consistent with `loadEd25519Key`'s
existing PKCS#8 handling.

### Decision 2 — Config field and default path

New field `AdmissionKeyFile string` / YAML `admission_key_file` in `internal/config.Config`.
Default path when absent: `/var/lib/switchboard/access-admission-identity.pem`. The default is
applied at daemon startup (`runAccess`), NOT by `Config.Validate()`. `Config.Validate()` only
checks that a non-empty value is not whitespace-only (E-CFG-014); it performs no file I/O
(ARCH-06 §Config purity contract).

### Decision 3 — First-run and subsequent-load semantics

**First run (file absent):** `os.Open` returns `os.ErrNotExist` → generate via
`ed25519.GenerateKey(rand.Reader)` → write PKCS#8 PEM to `<path>.tmp` with mode `0600` →
`os.Rename(<path>.tmp, <path>)` (atomic). If parent directory does not exist, `os.MkdirAll`
with `0700` first. Log INFO `"admission identity: generated new keypair at <path>"`.

**Subsequent start (file present):** `pem.Decode` → `x509.ParsePKCS8PrivateKey` → type-assert
to `ed25519.PrivateKey`. Any failure is **fail-closed** — `runAccess` returns a non-nil error
wrapping `"access: load admission keypair: <path>: <reason>"`. Daemon refuses to start.

**Permissions check:** if file permissions are broader than `0600`, log a structured WARNING
(not fatal): `"admission identity key file <path> has permissions <mode>: expected 0600; private key may be exposed"`. Match OpenSSH's advisory-not-fatal posture.

### Decision 4 — On-every-start INFO log of pubkey

On every start (first-run AND subsequent load), after the keypair is available, `runAccess`
logs INFO with the admission public key as base64url (no padding) of the raw 32-byte Ed25519
public key: `"access: admission identity pubkey (register with admin.key.register): <base64url-no-pad>"`.

### Decision 5 — LocalNodeAdmissionPubkey wiring

`admissionPubKey := admissionPrivKey.Public().(ed25519.PublicKey)`. Into
`discovery.Config.LocalNodeAdmissionPubkey`: `[]byte(admissionPubKey)` (32-byte raw key).
This is the exact shape `transmitAdvertisement` and `Encode` require.

### Decision 6 — Discovery.Run placement (Option Y, rulings §3.1)

`Discovery.Run` starts inside `runAccessWithConnector`, tracked in the same `sync.WaitGroup`
as sweep and frames-dropped ticker goroutines. `runAccessWithConnector` gains a `disc *discovery.Discovery` parameter (or equivalent — the implementer may also pass `discovery.Config` and construct inside; all three signature shapes are architecturally equivalent per BC-2.04.008 Invariant 5). The WaitGroup contract (ARCH-01 v1.7 §Goroutine WaitGroup Contract): `wg.Add(1)` in caller before `go`; `defer wg.Done()` inside goroutine.

### Decision 7 — Shutdown handling

`Discovery.Run(ctx)` returns `context.Canceled` on normal `ctx` cancellation. This return value
MUST NOT set `internalFailure = true` and MUST NOT call `cancel()` — it is a clean shutdown
signal. Only unexpected non-context errors from `disc.Run` are candidates for `internalFailure`.

### Decision 8 — DI-002 compliance

The admission private key is never logged, never transmitted over the network, and never stored
in any struct that crosses a package boundary. It is held as a local variable in `runAccess`
and passed as a parameter to the future `NODE_IDENTIFY` handshake function in
`S-BL.NODE-IDENTIFY-WIRE`. The public key and signatures computed from it are the only values
that leave the process.

### Decision 9 — ARCH-08 compliance

All new code lives in `internal/config` (position 1, no new imports needed) for the `AdmissionKeyFile` field, and in `cmd/switchboard` (position 18, the top) for the key I/O and lifecycle wiring. No new package introduced; no ARCH-08 position changes needed.

## Acceptance Criteria

### AC-001 — Config.Validate() accepts absent or valid admission_key_file; rejects whitespace-only (BC-2.09.003 v2.1 PC-12 / BC-2.09.004 PC-1 and PC-2 / E-CFG-014)

**BC Anchor:** BC-2.09.003 v2.1 Postcondition 12; BC-2.09.004 Postconditions 1–2.

**Postconditions:**
1. When `admission_key_file` is absent or empty string in config, `Config.Validate()` accepts
   the value and returns no error for this field.
2. When `admission_key_file` is present with a non-whitespace string value, `Config.Validate()`
   accepts it regardless of whether the file exists on disk.
3. When `admission_key_file` is present with a whitespace-only value (e.g., `"   "`),
   `Config.Validate()` returns an error containing E-CFG-014:
   `"config error: admission_key_file: must not be empty. Fix: set to a valid file path, e.g. '/var/lib/switchboard/access-admission-identity.pem', or remove the field to use the daemon default"`.
4. `Config.Validate()` performs no file I/O for `admission_key_file` — the error is returned
   without attempting to `stat`, `open`, or read the file path (ARCH-06 §Config purity).
5. The exhaustive error collection pattern is preserved: if multiple config fields have errors,
   all errors (including E-CFG-014 if applicable) are collected before the function returns.

**Test names:**
- `TestConfig_Validate_AdmissionKeyFile_AbsentAccepted`
- `TestConfig_Validate_AdmissionKeyFile_ValidPathAccepted`
- `TestConfig_Validate_AdmissionKeyFile_WhitespaceOnlyRejectsE_CFG_014`
- `TestConfig_Validate_AdmissionKeyFile_NoIOPerformed`

---

### AC-002 — First-run keypair generation: file absent → atomic write with mode 0600, PKCS#8 PEM, parent dir created (BC-2.09.004 PC-3)

**BC Anchor:** BC-2.09.004 Postconditions 3a–3e.

**Postconditions:**
1. When the effective key file path does not exist (`os.ErrNotExist`), a new Ed25519 keypair
   is generated via `ed25519.GenerateKey(rand.Reader)`.
2. The private key is written atomically: PKCS#8 PEM written to `<path>.tmp` with mode `0600`,
   then `os.Rename(<path>.tmp, <path>)`. The final file's permissions are `0600`.
3. If the parent directory does not exist, `os.MkdirAll(parentDir, 0700)` is called before
   writing.
4. A structured INFO log is emitted: `"admission identity: generated new keypair at <path>"`.
5. After generation, the derived public key (`admissionPrivKey.Public().(ed25519.PublicKey)`)
   is available for wiring into `discovery.Config.LocalNodeAdmissionPubkey`.
6. The generated PKCS#8 PEM file is parseable by `pem.Decode` → `x509.ParsePKCS8PrivateKey`
   → type-assert to `ed25519.PrivateKey` without error.

**Test names:**
- `TestAdmissionKeypair_FirstRun_FileAbsent_KeypairGeneratedAtomically`
- `TestAdmissionKeypair_FirstRun_ParentDirAbsent_MkdirAll`
- `TestAdmissionKeypair_FirstRun_GeneratedPKCS8PEMParseable`
- `TestAdmissionKeypair_FirstRun_Mode0600`

---

### AC-003 — Subsequent start: file present and valid → key loaded; same public key as first run (BC-2.09.004 PC-5)

**BC Anchor:** BC-2.09.004 Postconditions 5.

**Postconditions:**
1. When the effective key file path exists and contains a valid PKCS#8 Ed25519 PEM block,
   `pem.Decode` + `x509.ParsePKCS8PrivateKey` + type-assert to `ed25519.PrivateKey` succeeds.
2. The loaded private key's derived public key (`loadedPriv.Public().(ed25519.PublicKey)`)
   is equal to the public key from the same keypair's first-run generation (i.e., the public
   key is stable across restarts for the same key file).
3. The admission public key is available for wiring into `discovery.Config.LocalNodeAdmissionPubkey`.

**Test names:**
- `TestAdmissionKeypair_SubsequentStart_LoadedKeyMatchesGenerated`
- `TestAdmissionKeypair_SubsequentStart_PublicKeyStableAcrossRestarts`

---

### AC-004 — Fail-closed on corrupt or non-Ed25519 file (BC-2.09.004 PC-6 / E-KEY-001)

**BC Anchor:** BC-2.09.004 Postconditions 6.

**Postconditions:**
1. When the key file exists but contains truncated or non-parseable PEM data, `runAccess`
   returns a non-nil error wrapping `"access: load admission keypair: <path>: PEM decode failed"`.
   The daemon refuses to start.
2. When the key file contains a valid PKCS#8 PEM block but with a non-Ed25519 key type (e.g.,
   RSA), `runAccess` returns a non-nil error wrapping `"access: load admission keypair: <path>:
   not an Ed25519 key"`. The daemon refuses to start.
3. No partial startup occurs — `runAccess` returns the error before any goroutine is launched,
   consistent with the fail-closed posture of a bad `listen_addr`.

**Test names:**
- `TestAdmissionKeypair_FailClosed_CorruptPEM`
- `TestAdmissionKeypair_FailClosed_NonEd25519Key`

---

### AC-005 — File present with permissions > 0600 → WARNING logged; daemon starts (BC-2.09.004 PC-4)

**BC Anchor:** BC-2.09.004 Postcondition 4.

**Postconditions:**
1. When the key file exists, is parseable, and has permissions broader than `0600` (e.g., `0644`),
   a structured WARNING is logged: `"admission identity key file <path> has permissions <mode>:
   expected 0600; private key may be exposed"`.
2. The daemon continues to start normally (advisory-not-fatal). The key is loaded and used.
3. The warning is not emitted when permissions are exactly `0600`.

**Test names:**
- `TestAdmissionKeypair_PermissionsWarning_BroaderThan0600`
- `TestAdmissionKeypair_NoPermissionsWarning_Exactly0600`

---

### AC-006 — Startup INFO log with base64url pubkey on every start (BC-2.09.004 PC-7)

**BC Anchor:** BC-2.09.004 Postcondition 7.

**Postconditions:**
1. On every successful start (whether keypair was generated on first run OR loaded from an
   existing file), after the keypair is available, `runAccess` emits a structured INFO log
   whose message begins with:
   `"access: admission identity pubkey (register with admin.key.register):"` and contains
   the base64url (no padding) encoding of the raw 32-byte Ed25519 public key.
2. The logged base64url value, when decoded, produces a 32-byte slice equal to
   `[]byte(admissionPrivKey.Public().(ed25519.PublicKey))`.
3. The log is emitted unconditionally (not conditioned on first-run vs. subsequent-start) so
   the operator can recover the pubkey from logs after any restart.

**Test names:**
- `TestAdmissionKeypair_StartupInfoLog_FirstRun_ContainsBase64UrlPubkey`
- `TestAdmissionKeypair_StartupInfoLog_SubsequentStart_ContainsBase64UrlPubkey`

---

### AC-007 — discovery.Config.LocalNodeAdmissionPubkey wired from loaded/generated keypair (BC-2.09.004 PC-3e / BC-2.04.008 Precondition 3)

**BC Anchor:** BC-2.09.004 Postcondition 3e; BC-2.04.008 Precondition 3.

**Postconditions:**
1. The `discovery.Config` constructed in `runAccess` has `LocalNodeAdmissionPubkey` set to
   the 32-byte raw Ed25519 public key derived from the admission keypair:
   `[]byte(admissionPrivKey.Public().(ed25519.PublicKey))`.
2. `LocalNodeAdmissionPubkey` is non-nil and has length 32.
3. `discovery.New(discoveryCfg)` is called AFTER the admission keypair is loaded/generated
   (startup-ordering rule: phases (d)–(e) per rulings §3.1 — keypair load before `discovery.New`
   before `disc.Run`).
4. Calling `disc.Run(ctx)` does not return `ErrMissingNodeAdmissionPubkey` (which would mean
   `LocalNodeAdmissionPubkey` was empty at run time).

**Test names:**
- `TestAccessDaemon_LocalNodeAdmissionPubkey_PopulatedFrom_LoadedKeypair`
- `TestAccessDaemon_LocalNodeAdmissionPubkey_NonNilLength32`

---

### AC-008 — Discovery.Run goroutine WG-tracked in runAccessWithConnector; ctx.Canceled is clean shutdown; no goroutine leak (BC-2.04.008 PC-1 through PC-4)

**BC Anchor:** BC-2.04.008 Postconditions 1–4.

**Postconditions:**
1. `wg.Add(1)` is called in the caller scope (i.e., in `runAccessWithConnector` or the
   equivalent dispatch site) BEFORE the `go disc.Run(runCtx)` statement. The goroutine body
   calls `defer wg.Done()` as its first statement after entry. This is the ARCH-01 v1.7
   §Goroutine WaitGroup Contract.
2. The discovery goroutine is tracked in the same `sync.WaitGroup` used by sweep and
   frames-dropped ticker goroutines (no separate WaitGroup introduced for discovery).
3. When `runCtx` is cancelled (`context.Canceled` or `context.DeadlineExceeded` returned by
   `disc.Run`), neither `internalFailure = true` nor `cancel()` is called. These return values
   are clean shutdown signals, not errors.
4. `wg.Wait()` in `runAccessWithConnector` returns cleanly after `runCtx` cancellation with no
   goroutine leak. Test enforcement: `t.Cleanup` + a bounded `wg.Wait()` with a `time.AfterFunc`
   deadline of ≤100ms after cancellation.
5. The `Discovery.Run` goroutine is started AFTER the admission keypair is loaded AND AFTER the
   management server goroutine is started (startup-ordering rule: phases (a)–(f) per rulings §3.1).

**Test names:**
- `TestDiscoveryRun_WGAddBeforeGoStatement`
- `TestDiscoveryRun_SameWaitGroupAsSweepTickers`
- `TestDiscoveryRun_CtxCanceled_NotInternalFailure`
- `TestDiscoveryRun_NoGoroutineLeak_AfterCtxCancel`
- `TestDiscoveryRun_StartupOrdering_AfterKeypairAndMgmtServer`

---

## Non-Goals

- **NODE_IDENTIFY opcode / ChallengeResponse signing** — the admission private key is made
  available in the `runAccess` scope by this story so that `S-BL.NODE-IDENTIFY-WIRE` can use it.
  This story does not implement the wire handshake itself.
- **Admission-state sync from control to router** — that is `S-BL.ADMISSION-SYNC-WIRE`.
- **sbctl admin.node.pubkey command** — the operator recovers the pubkey from the daemon startup
  log (AC-006). A dedicated `sbctl` retrieval command is out of scope.
- **SIGHUP keypair reload** — the admission identity is loaded once at daemon startup and held
  for the process lifetime. A SIGHUP reload of config does not re-load the admission keypair
  (EC-010 in BC-2.09.004).

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | `admission_key_file` absent entirely | `Validate()` accepts; `runAccess` uses default path `/var/lib/switchboard/access-admission-identity.pem`. |
| EC-002 | `admission_key_file: "   "` (whitespace) | E-CFG-014; exit 1. All config errors collected before return. |
| EC-003 | Valid path; file does not exist (first run) | Keypair generated atomically with mode 0600; INFO log; pubkey INFO log; daemon starts. |
| EC-004 | Valid path; file present and valid Ed25519 PKCS#8 | Key loaded; pubkey INFO log; daemon starts. |
| EC-005 | File exists with permissions `0644` | WARNING logged; daemon starts (advisory-not-fatal). |
| EC-006 | File exists; corrupt PEM (truncated) | E-KEY-001 `"access: load admission keypair: <path>: PEM decode failed"`; exit 1. |
| EC-007 | File exists; valid PEM but RSA key (not Ed25519) | E-KEY-001 `"access: load admission keypair: <path>: not an Ed25519 key"`; exit 1. |
| EC-008 | Parent directory for default path does not exist | `os.MkdirAll(parentDir, 0700)`; keypair generated; daemon starts. |
| EC-009 | SIGHUP reload with a different `admission_key_file` path | `Validate()` accepts the non-whitespace string. Keypair is NOT re-loaded on SIGHUP — identity is stable for the process lifetime. |
| EC-010 | `runCtx` already cancelled when `disc.Run` is called | `disc.Run` returns `context.Canceled` immediately; `wg.Done()` fires; `wg.Wait()` returns within 100ms; no `internalFailure`. |
| EC-011 | Crash mid-key-write (simulated by removing .tmp file) | The `.tmp` file never gets renamed; the canonical path remains absent on the next start; fresh keypair generated on next start. |

## File-Change List

| File | Change | Justification |
|------|--------|---------------|
| `internal/config/config.go` | Add `AdmissionKeyFile string \`yaml:"admission_key_file"\`` field to `Config` struct; add E-CFG-014 validation case to `Validate()` (whitespace-only check, no I/O, same shape as E-CFG-008) | BC-2.09.003 PC-12; BC-2.09.004 PC-1/PC-2 |
| `cmd/switchboard/access.go` | Add admission keypair load/generate logic in `runAccess` (phases (d)–(e) per rulings §3.1); add startup INFO + WARNING logs; pass `disc` or `discovery.Config` into `runAccessWithConnector`; extend `runAccessWithConnector` signature | BC-2.09.004 PC-3 through PC-7; BC-2.04.008 PC-1 through PC-5 |
| `cmd/switchboard/access_test.go` (or `access_admission_test.go`) | New test file (or extended) covering all AC-001..AC-008 test cases | All ACs |
| `internal/config/config_test.go` | New or extended table-driven tests for `admission_key_file` validation (AC-001) | AC-001 |

## Token Budget Estimate

| Component | Estimate |
|-----------|---------|
| `internal/config` field + validate extension | ~50 tokens |
| `cmd/switchboard/access.go` keypair I/O + lifecycle wiring | ~200 tokens |
| Unit + integration tests (8 ACs, ~12 test functions) | ~350 tokens |
| **Overall** | ~600 tokens — well within the 1000-token story budget |

## Architecture Compliance Rules

| Rule | Requirement | Enforcement |
|------|-------------|-------------|
| ARCH-06 §Config purity | `Config.Validate()` MUST NOT perform file I/O for `admission_key_file` | Verified by AC-001 test `TestConfig_Validate_AdmissionKeyFile_NoIOPerformed` (mock filesystem or absent file path that does not exist — `Validate()` must return no error) |
| ARCH-01 v1.7 §Goroutine WaitGroup Contract | `wg.Add(1)` in caller before `go disc.Run(runCtx)` | Verified by AC-008; enforced by race detector (`go test -race`) |
| ARCH-08 §Import DAG | No new packages introduced; `internal/config` at position 1 (no new imports); `cmd/switchboard` at position 18 (may import all) | Enforced at compile time; verified by `go list -deps` check in tests |
| DI-002 | Admission private key never logged, never transmitted, never in cross-package struct | Verified by code review; `admissionPrivKey` is a local variable in `runAccess` scope only |
| BC-2.04.008 Invariant 2 | `context.Canceled` from `disc.Run` is NOT `internalFailure` | Verified by AC-008 test `TestDiscoveryRun_CtxCanceled_NotInternalFailure` |

## POL-005 Delivery Plan Note

This story is a leaf prerequisite in the identity-cluster (`depends_on: []`). It does not
depend on `S-BL.ADMISSION-SYNC-WIRE`. Implementations targeting `S-BL.NODE-IDENTIFY-WIRE`
should deliver this story first, then `S-BL.ADMISSION-SYNC-WIRE`, then `S-BL.NODE-IDENTIFY-WIRE`.

TDD discipline: the Red Gate (failing tests) must be established before any implementation
code is written. The `compute-input-hash --check` command should be run before beginning
implementation to verify input files have not changed since `input-hash: "504693c"` was recorded.

## Provenance

- **Origin:** `S-BL.DISCOVERY-WIRE-rulings.md` v1.11 Ruling 5 (Forward Obligation (f)) and
  `decisions/identity-cluster-architecture.md` v1.2 (three-leg cluster design, §4).
- **Rulings:** `decisions/S-BL.NODE-ADMISSION-PROVISIONING-rulings.md` v1.0 (all rulings,
  zero open human flags — fully decomposition-ready as of 2026-07-15).
- **Unblocks:** `S-BL.NODE-IDENTIFY-WIRE`'s ChallengeResponse signing step (Open Design
  Obligation 6) and `S-BL.DISCOVERY-WIRE`'s production `LocalNodeAdmissionPubkey` path.

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-07-15 | Initial full decomposition — 8 ACs, 5 points, leaf prerequisite with `depends_on: []`. Keypair provisioning (BC-2.09.004) + Discovery.Run lifecycle wiring (BC-2.04.008). Per rulings v1.0 Option E + Option Y. |
