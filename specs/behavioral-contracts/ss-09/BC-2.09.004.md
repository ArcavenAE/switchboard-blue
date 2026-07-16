---
artifact_id: BC-2.09.004
document_type: behavioral-contract
level: L3
version: "1.0"
status: draft
producer: product-owner
timestamp: 2026-07-15T00:00:00Z
phase: 1a
inputs:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/domain-spec/failure-modes.md'
  - '_bmad-output/planning-artifacts/prd.md'
  - 'decisions/S-BL.NODE-ADMISSION-PROVISIONING-rulings.md'
  - 'decisions/identity-cluster-architecture.md'
input-hash: "893d434"
extracted_from: null
bc_id: BC-2.09.004
subsystem: deployment-operations
architecture_module: internal/config
capability: CAP-028
priority: P0
criticality: critical
scope_phase: E
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified:
  - date: 2026-07-15
    version: "1.0"
    change: >
      Initial draft — admission_key_file config field: first-run keypair generation,
      fail-closed on corrupt, Validate() no-I/O contract, file permissions 0600, startup
      INFO log with base64url pubkey. Authored per S-BL.NODE-ADMISSION-PROVISIONING
      BC groundwork list item N1 (rulings.md v1.0 §6, identity-cluster-architecture.md §5).
deprecated: null
deprecated_by: null
replacement: null
retired: null
removed: null
removal_reason: null
inputDocuments:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/domain-spec/failure-modes.md'
  - '_bmad-output/planning-artifacts/prd.md'
  - 'decisions/S-BL.NODE-ADMISSION-PROVISIONING-rulings.md'
  - 'decisions/identity-cluster-architecture.md'
traces_to: [CAP-028]
kos_anchors:
  - elem-single-binary-three-modes
---

# Behavioral Contract BC-2.09.004: Access Daemon Admission Keypair — Provisioning, Loading, and Config Validation for `admission_key_file`

## Description

The access-mode daemon (`switchboard access`) requires a persistent Ed25519 admission keypair to
participate in SVTN discovery and the `NODE_IDENTIFY` challenge-response handshake
(`S-BL.NODE-IDENTIFY-WIRE`). The keypair is stored as a PKCS#8 PEM file at a configurable path.
On first start the daemon generates and writes the keypair atomically; on subsequent starts it
loads the keypair from the file. A corrupt or missing-but-mandatory file causes the daemon to
refuse to start (fail-closed). `Config.Validate()` checks the field shape only — it performs no
file I/O (ARCH-06 §Config purity contract).

This BC covers three concerns: (1) the `admission_key_file` config field validation rule, (2)
the first-run generation and subsequent load semantics, and (3) the startup INFO log that surfaces
the public key to the operator so they can run `admin.key.register`.

## Preconditions

**Validate() path:**
1. `Config.Validate()` is called (daemon startup or SIGHUP reload).

**Daemon startup / key provisioning path:**
1. The access daemon is starting (`runAccess` entry).
2. The `admission_key_file` config field has been validated by `Config.Validate()`.
3. The effective key file path is resolved: `cfg.AdmissionKeyFile` if non-empty, else the
   default path `/var/lib/switchboard/access-admission-identity.pem`.

## Postconditions

### Config.Validate() postconditions

1. If `admission_key_file` is absent or empty string: `Validate()` **accepts** the value. Absence
   means "use the default path at startup time" — this is not a validation error. No I/O is
   performed.

2. If `admission_key_file` is present (non-empty string): `Validate()` accepts any non-whitespace
   string value, regardless of whether the path exists on disk. It MUST NOT attempt to read or
   parse the key file — file accessibility is a daemon-startup I/O concern, not a config-validation
   concern (ARCH-06 §Config purity). If the value is present but whitespace-only, `Validate()`
   returns E-CFG-014 (same rule as PC-10 / E-CFG-008 for `management_socket`):
   `"config error: admission_key_file: must not be empty. Fix: set to a valid file path, e.g. '/var/lib/switchboard/access-admission-identity.pem', or remove the field to use the daemon default"`.

   **Config-schema impact note:** A new field is added to `internal/config.Config`:
   ```go
   // AdmissionKeyFile is the path to the access daemon's persistent Ed25519 admission keypair
   // file (PKCS#8 PEM, "PRIVATE KEY" block type). Required for access-mode operation.
   // When absent (empty string), the daemon uses the default path:
   //   /var/lib/switchboard/access-admission-identity.pem
   // When present, the path is used as-is (tilde expansion follows loadEd25519Key conventions).
   // Validate() only checks that a non-empty value is not whitespace-only; it does not read
   // or stat the file (ARCH-06 §Config purity contract).
   AdmissionKeyFile string `yaml:"admission_key_file"`
   ```
   The YAML config schema in `interface-definitions.md §Config Schema` must be updated to
   document this field. This is an implementer responsibility for `S-BL.NODE-ADMISSION-PROVISIONING`.

### First-run keypair generation postconditions

3. If the effective key file path does not exist (`os.ErrNotExist`):
   a. A new Ed25519 keypair is generated via `ed25519.GenerateKey(rand.Reader)`.
   b. The private key is written atomically to the key file path: write PKCS#8 PEM to
      `<path>.tmp` with mode `0600`, then `os.Rename(<path>.tmp, <path>)`. If the parent
      directory does not exist, `os.MkdirAll` with mode `0700` is called first.
   c. The file's final permissions are `0600` (owner read-write; no group or other access).
   d. A structured INFO log is emitted:
      `"admission identity: generated new keypair at <path>"`.
   e. The derived public key (`admissionPrivKey.Public().(ed25519.PublicKey)`) is available for
      wiring into `discovery.Config.LocalNodeAdmissionPubkey`.

4. If the effective key file path exists but the permissions are broader than `0600`, the daemon
   logs a structured WARNING (not fatal):
   `"admission identity key file <path> has permissions <mode>: expected 0600; private key may be exposed"`.
   The daemon continues to start (matching OpenSSH's advisory-not-fatal posture for loose key
   permissions).

### Subsequent-start keypair load postconditions

5. If the effective key file path exists: the file is read, PEM-decoded, and parsed via
   `x509.ParsePKCS8PrivateKey`. The result is type-asserted to `ed25519.PrivateKey`.

6. If any step in PC-5 fails (file unreadable, PEM malformed, non-Ed25519 key type, parse error):
   **fail-closed** — `runAccess` returns a non-nil error wrapping:
   `"access: load admission keypair: <path>: <reason>"`.
   The daemon refuses to start. This is the same fail-closed posture as a bad `listen_addr`.

### Startup INFO log postcondition

7. On every start (first-run generation AND subsequent load), after the keypair is
   available, `runAccess` logs a structured INFO message containing the admission public key
   encoded as base64url (no padding) of the raw 32-byte Ed25519 public key:
   ```
   access: admission identity pubkey (register with admin.key.register): <base64url-no-pad>
   ```
   This is emitted unconditionally so the operator can recover the pubkey from logs after
   a restart without needing to access the key file directly.

## Invariants

1. `Config.Validate()` performs no I/O for `admission_key_file` — path accessibility is checked
   only at daemon startup (`runAccess`), not at config parse time (ARCH-06 §Config purity).
2. The admission private key is never logged, transmitted over a network, or stored in any struct
   that crosses a package boundary. It is held in the local scope of `runAccess` and passed as a
   parameter to functions that need it (DI-002).
3. First-run key generation is atomic: a partial write (crash, disk-full) does not leave a corrupt
   key file because the write-to-temp + rename pattern is used.
4. The `admission_key_file` field is **access-mode only**. Other daemon modes (router, control,
   console) do not read or act on this field. Validate() does not enforce mode-specific restrictions
   at parse time; the application-level check is performed by `runAccess` during startup.
5. Absence of `admission_key_file` in the config is not an error — the daemon uses the default
   path. This default is applied at daemon startup, NOT by `Config.Validate()`.

## Trigger

- `Config.Validate()` called at daemon startup or SIGHUP reload (PC-1/PC-2 path).
- `runAccess` starting and resolving the admission keypair (PC-3 through PC-7 path).

## Error Codes

| Code | Condition | Severity | Exit Code | Message Template |
|------|-----------|----------|-----------|-----------------|
| E-CFG-014 | `admission_key_file` is present but whitespace-only | broken | 1 | `"config error: admission_key_file: must not be empty. Fix: set to a valid file path, e.g. '/var/lib/switchboard/access-admission-identity.pem', or remove the field to use the daemon default"` |
| E-KEY-001 | Admission keypair file exists but fails to load (unreadable, PEM malformed, non-Ed25519 type) | broken | 1 | `"access: load admission keypair: <path>: <reason>"` |

> **Note on E-KEY-001:** This is a new error family (`KEY`) for keypair provisioning and load
> failures. The message is wrapped from `runAccess` and surfaces to the operator as a startup
> failure log entry. It is distinct from E-CFG-010 (sbctl `--key` flag load failure) because
> it is a daemon-startup error, not a CLI-flag error.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | `admission_key_file` absent entirely | `Validate()` accepts; `runAccess` uses default path `/var/lib/switchboard/access-admission-identity.pem`. No error. |
| EC-002 | `admission_key_file: "   "` (whitespace-only value) | E-CFG-014; exit 1. Exhaustive error collection: if other errors also exist, all are reported together. |
| EC-003 | `admission_key_file` set to a valid path; file does not exist | First-run generation: keypair generated, written atomically with mode 0600, INFO log emitted. Daemon starts normally. |
| EC-004 | `admission_key_file` set to a valid path; file exists and is parseable Ed25519 PKCS#8 | Subsequent load: key loaded, pubkey derived, INFO log emitted. Daemon starts normally. |
| EC-005 | Key file exists but has permissions `0644` (broader than 0600) | WARNING logged: `"admission identity key file <path> has permissions 0644: expected 0600; private key may be exposed"`. Daemon starts normally (advisory-not-fatal). |
| EC-006 | Key file exists but contains RSA PKCS#8 (wrong key type) | E-KEY-001: `"access: load admission keypair: <path>: not an Ed25519 key"`. `runAccess` returns error; daemon exits 1. |
| EC-007 | Key file exists but contains invalid PEM (truncated/corrupt) | E-KEY-001: `"access: load admission keypair: <path>: PEM decode failed"`. `runAccess` returns error; daemon exits 1. |
| EC-008 | Parent directory for default path does not exist | `os.MkdirAll` creates it with mode `0700`; keypair is generated and written to the new directory. Daemon starts normally. |
| EC-009 | Config reload (SIGHUP) with whitespace `admission_key_file` | E-CFG-014 (exhaustive, with any other errors); daemon continues on previous config (Inv-3 of BC-2.09.003). |
| EC-010 | Config reload (SIGHUP) with valid `admission_key_file` path | `Validate()` accepts the non-whitespace string. Keypair is NOT re-loaded on SIGHUP — the admission identity is loaded once at daemon startup and held for the process lifetime. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Config with `admission_key_file` absent | `Validate()` accepts; `runAccess` uses default path; daemon starts | happy-path |
| Config with `admission_key_file: "   "` (whitespace) | E-CFG-014 "config error: admission_key_file: must not be empty..."; exit 1 | error (PC-2) |
| Config with valid path; file absent | Keypair generated; written to path with mode 0600; INFO log at start; pubkey INFO log emitted; daemon starts | happy-path (PC-3) |
| Config with valid path; file present and valid Ed25519 PKCS#8 | Key loaded; pubkey INFO log emitted; daemon starts | happy-path (PC-5) |
| Config with valid path; file present with mode 0755 | WARNING "...has permissions 0755: expected 0600..."; daemon starts normally | edge-case (PC-4) |
| Config with valid path; file is corrupt PEM | E-KEY-001 "access: load admission keypair: <path>: PEM decode failed"; exit 1 | error (PC-6) |
| Config with valid path; file is valid PEM but RSA key | E-KEY-001 "access: load admission keypair: <path>: not an Ed25519 key"; exit 1 | error (PC-6) |
| Any valid start (first-run or subsequent load) | INFO log contains base64url-no-pad 32-byte pubkey; message starts with "access: admission identity pubkey (register with admin.key.register):" | happy-path (PC-7) |

## Verification Properties

| VP-NNN | Property | Proof Method | Notes |
|--------|----------|-------------|-------|
| test-as-evidence | `admission_key_file` absent → `Validate()` accepts; default path used at startup | unit | S-BL.NODE-ADMISSION-PROVISIONING AC |
| test-as-evidence | Whitespace-only `admission_key_file` → E-CFG-014; exit 1 (no I/O) | unit | S-BL.NODE-ADMISSION-PROVISIONING AC |
| test-as-evidence | File absent → keypair generated, mode 0600, PKCS#8 PEM parseable, pubkey matches derived key | unit | S-BL.NODE-ADMISSION-PROVISIONING AC |
| test-as-evidence | File present with mode > 0600 → WARNING logged; daemon starts | unit | S-BL.NODE-ADMISSION-PROVISIONING AC |
| test-as-evidence | Corrupt PEM → fail-closed; `runAccess` returns non-nil error | unit | S-BL.NODE-ADMISSION-PROVISIONING AC |
| test-as-evidence | Startup INFO log contains base64url-no-pad pubkey on every start | unit | S-BL.NODE-ADMISSION-PROVISIONING AC |
| test-as-evidence | `discovery.Config.LocalNodeAdmissionPubkey` populated with 32-byte raw pubkey from loaded/generated key | unit | S-BL.NODE-ADMISSION-PROVISIONING AC |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-028 ("Daemon startup config validation") per capabilities.md §CAP-028 |
| L2 Domain Invariants | DI-002 (private keys never transit), none directly for config validation |
| Architecture Module | internal/config (Validate postconditions); cmd/switchboard (keypair I/O and lifecycle) |
| Stories | S-BL.NODE-ADMISSION-PROVISIONING (all postconditions and ACs) |
| Capability Anchor Justification | CAP-028 ("Daemon startup config validation") — this BC specifies the config-field validation rule for `admission_key_file` (subset of CAP-028's startup-validation guarantee) plus the adjacent keypair-provisioning logic that is gated on valid config. CAP-028 is the best available anchor; a future domain-spec revision may introduce a dedicated admission-provisioning CAP. |

## Related BCs

- BC-2.09.003 — parallel: same config-validation shape (PC-10/E-CFG-008 `management_socket` and PC-11/E-CFG-009 `authorized_operator_keys` precedent); `admission_key_file` follows the same optional-field-non-whitespace rule
- BC-2.04.008 — composes with: Discovery.Run() daemon-lifecycle wiring depends on the admission keypair provisioned by this BC
- BC-2.03.001 — downstream: `LocalNodeAdmissionPubkey` wired via this BC feeds the discovery advertisement HMAC key derivation

## Architecture Anchors

- ARCH-06-deployment-and-ops.md §Config File Validation (ARCH-06 / config purity contract: Validate() performs no I/O)
- ARCH-08-dependency-graph.md §SS-09 (deployment-operations, internal/config at position 1)
- decisions/S-BL.NODE-ADMISSION-PROVISIONING-rulings.md §1–§2 (key file format, config field, first-run vs. subsequent-load semantics, file permissions)

## Story Anchor

S-BL.NODE-ADMISSION-PROVISIONING — all postconditions in this BC trace to acceptance criteria for this story.

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-07-15 | Initial draft — `admission_key_file` config field validation (E-CFG-014), first-run keypair generation, subsequent load, fail-closed on corrupt, 0600 permissions + advisory warning for broader, startup INFO pubkey log. Authored per S-BL.NODE-ADMISSION-PROVISIONING BC groundwork item N1. |
