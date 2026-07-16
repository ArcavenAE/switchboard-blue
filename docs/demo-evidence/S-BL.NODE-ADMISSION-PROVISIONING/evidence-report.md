# Demo Evidence Report — S-BL.NODE-ADMISSION-PROVISIONING

**Story:** Node admission-identity provisioning: Ed25519 admission keypair at admission_key_file + Discovery.Run wired into access daemon lifecycle
**Story version:** v1.0
**Branch:** feature/S-BL.NODE-ADMISSION-PROVISIONING
**Code HEAD:** 7e130129972ed792f30d76c7ac50adcfcacb9fdb
**Evidence date:** 2026-07-16

---

## POL-004 Note

Per `docs/DEMO-EVIDENCE-POLICY.md` (ratified 2026-07-04), rendered binaries
(`.gif`, `.webm`, `.mp4`, `.png`) are **not committed**. `.tape` scripts and
this evidence report are the source of truth. To regenerate locally:

```bash
vhs docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-001-config-validation-e-cfg-014.tape
vhs docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-002-003-first-run-and-subsequent-load.tape
vhs docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-004-fail-closed-corrupt-and-non-ed25519.tape
vhs docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-005-permissions-warning.tape
vhs docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-006-startup-info-log-base64url-pubkey.tape
vhs docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-007-local-node-admission-pubkey-wired.tape
vhs docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-008-discovery-run-wg-tracked-clean-shutdown.tape
vhs docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/M3-tilde-expansion-admission-key-path.tape
```

## Scope note — daemon-internal keypair provisioning + lifecycle wiring

This story implements admission-identity provisioning (`loadOrGenerateAdmissionKeypair`
in `cmd/switchboard/access.go`) and `Discovery.Run` lifecycle wiring into
`runAccessWithConnector`. Neither of these surfaces directly through a CLI command
for direct interactive demonstration (no `sbctl` command retrieves the admission pubkey
from a running daemon — that is an explicit non-goal of this story per the story spec).

The honest, meaningful evidence for this plumbing story is the passing test suite that
directly verifies each AC's stated postconditions. All tape commands below drive
`go test -race -run <pattern> -v` targeting the specific AC test functions. Every test
passes under `go test -race`.

---

## Summary Table — AC → Tape → Test(s) → Result → Postconditions demonstrated

| AC | Tape | Test(s) | Result | Postconditions demonstrated |
|----|------|---------|--------|-----------------------------|
| AC-001 | `AC-001-config-validation-e-cfg-014.tape` | `TestConfig_Validate_AdmissionKeyFile_AbsentAccepted`, `TestConfig_Validate_AdmissionKeyFile_ValidPathAccepted`, `TestConfig_Validate_AdmissionKeyFile_WhitespaceOnlyRejectsE_CFG_014`, `TestConfig_Validate_AdmissionKeyFile_NoIOPerformed`, `TestConfig_Validate_AdmissionKeyFile_ExhaustiveErrorCollection` | PASS | E-CFG-014 fires for whitespace-only `admission_key_file`; absent/empty accepted; non-whitespace path accepted without I/O; exhaustive error collection preserved alongside other fields. |
| AC-002 | `AC-002-003-first-run-and-subsequent-load.tape` | `TestAdmissionKeypair_FirstRun_FileAbsent_KeypairGeneratedAtomically`, `TestAdmissionKeypair_FirstRun_ParentDirAbsent_MkdirAll`, `TestAdmissionKeypair_FirstRun_GeneratedPKCS8PEMParseable`, `TestAdmissionKeypair_FirstRun_Mode0600`, `TestAdmissionKeypair_FirstRun_OrphanTmpDoesNotWedgeStartup` | PASS | Absent key file → Ed25519 keypair generated atomically (`CreateTemp` → `os.Rename`); mode 0600 enforced; PKCS#8 PEM parseable; parent dir created with MkdirAll; orphaned `.tmp` from a prior interrupted run does not wedge next start (B-1 fix). |
| AC-003 | `AC-002-003-first-run-and-subsequent-load.tape` | `TestAdmissionKeypair_SubsequentStart_LoadedKeyMatchesGenerated`, `TestAdmissionKeypair_SubsequentStart_PublicKeyStableAcrossRestarts` | PASS | File present and valid → same public key loaded as on first run; public key is stable across multiple simulated restarts. |
| AC-004 | `AC-004-fail-closed-corrupt-and-non-ed25519.tape` | `TestAdmissionKeypair_FailClosed_CorruptPEM`, `TestAdmissionKeypair_FailClosed_NonEd25519Key`, `TestRunAccess_KeypairLoadFailure_MgmtGoroutineNotLeaked` | PASS | Corrupt/truncated PEM → error containing "PEM decode failed" + path; PKCS#8 RSA key → "not an Ed25519 key" + path; daemon refuses to start in both cases; mgmt goroutine joined before `runAccess` returns (no partial startup, adversary F-2 fix). |
| AC-005 | `AC-005-permissions-warning.tape` | `TestAdmissionKeypair_PermissionsWarning_BroaderThan0600`, `TestAdmissionKeypair_NoPermissionsWarning_Exactly0600` | PASS | Mode 0644, 0444, 0440 → WARNING containing octal mode and "permissions" (F-1 fix: predicate is `perm&0o077 != 0`, not `perm > 0o600` — catches world-readable 0444 which is numerically less than 0600); mode 0600 → no warning; daemon starts in both cases (advisory-not-fatal). |
| AC-006 | `AC-006-startup-info-log-base64url-pubkey.tape` | `TestAdmissionKeypair_StartupInfoLog_FirstRun_ContainsBase64UrlPubkey`, `TestAdmissionKeypair_StartupInfoLog_SubsequentStart_ContainsBase64UrlPubkey` | PASS | INFO log "access: admission identity pubkey (register with admin.key.register): \<base64url\>" emitted on both first-run and subsequent-start; base64url value decodes to the exact 32-byte public key. |
| AC-007 | `AC-007-local-node-admission-pubkey-wired.tape` | `TestAccessDaemon_LocalNodeAdmissionPubkey_PopulatedFrom_LoadedKeypair`, `TestAccessDaemon_LocalNodeAdmissionPubkey_NonNilLength32`, `TestRunAccess_WiresLocalNodeAdmissionPubkey_FromLoadedKeypair` | PASS | `LocalNodeAdmissionPubkey` non-nil, length 32; `disc.Run` with pre-cancelled context returns `context.Canceled` (not `ErrMissingNodeAdmissionPubkey`); through-`runAccess` wiring confirmed via `newDiscovery` seam — captured `discovery.Config.LocalNodeAdmissionPubkey` byte-equals the known keypair's public key (adversary F-6). |
| AC-008 | `AC-008-discovery-run-wg-tracked-clean-shutdown.tape` | `TestDiscoveryRun_WGAddBeforeGoStatement`, `TestDiscoveryRun_SameWaitGroupAsSweepTickers`, `TestDiscoveryRun_CtxCanceled_NotInternalFailure`, `TestDiscoveryRun_NoGoroutineLeak_AfterCtxCancel`, `TestDiscoveryRun_StartupOrdering_AfterKeypairAndMgmtServer` | PASS | `disc.Run` started in WG-tracked goroutine (`wg.Add(1)` before `go`, `defer wg.Done()` inside); shares same `sync.WaitGroup` as sweep/frames-dropped tickers; `ctx.Canceled` from `disc.Run` → `runAccessWithConnector` returns `nil` (not internal failure, Decision 7); no goroutine leak after ctx cancel (blocking HeartbeatObserver discriminator — `wg.Wait` blocks until disc released); disc starts after keypair load + mgmt server (startup ordering). |
| M3 (tilde) | `M3-tilde-expansion-admission-key-path.tape` | `TestRunAccess_TildeExpansion_TildeSlashPrefix`, `TestRunAccess_TildeExpansion_TildeOnly`, `TestRunAccess_TildeExpansion_AbsolutePathUnchanged`, `TestRunAccess_TildeExpansion_TildeUsername_Literal` | PASS | `~/...` expands to `$HOME/<remainder>`; `~` alone expands to `$HOME`; absolute paths used as-is; `~username` treated as literal (matches sbctl EC-007 semantics). |

---

## AC-001 — Config.Validate() E-CFG-014 admission_key_file validation

**Tape:** `AC-001-config-validation-e-cfg-014.tape`
**BC anchors:** BC-2.09.003 v2.1 PC-12; BC-2.09.004 PC-1/PC-2; E-CFG-014
**Test file:** `internal/config/config_test.go`

**Success path:** absent or non-whitespace `admission_key_file` → `Config.Validate()` returns no error for this field (no I/O performed — a non-existent path is accepted per ARCH-06 §Config purity).

**Error path:** `admission_key_file: "   "` (whitespace-only) → canonical E-CFG-014 message returned:
`"config error: admission_key_file: must not be empty. Fix: set to a valid file path, e.g. '/var/lib/switchboard/access-admission-identity.pem', or remove the field to use the daemon default"`.
Exhaustive error collection: when `tick_interval` is also invalid, both errors appear before `Validate()` returns.

---

## AC-002 — First-run keypair generation (atomic write, mode 0600, PKCS#8 PEM, parent dir)

**Tape:** `AC-002-003-first-run-and-subsequent-load.tape`
**BC anchors:** BC-2.09.004 PC-3a–3f
**Test file:** `cmd/switchboard/access_admission_test.go`

**Success path (first run):** file absent → keypair generated; PKCS#8 PEM written atomically via `os.CreateTemp` + `os.Rename` to final path; file mode 0600; parent dir created with `MkdirAll(parentDir, 0700)` if absent; INFO log `"admission identity: generated new keypair at <path>"` in stderr; generated file parseable via `pem.Decode` → `x509.ParsePKCS8PrivateKey` → `ed25519.PrivateKey`.

**B-1 fix (adversary pass-2):** an orphaned `<path>.tmp` from a prior interrupted run does not wedge next startup — the new `os.CreateTemp` code uses a random suffix, so the fixed-name orphan is irrelevant.

---

## AC-003 — Subsequent start: same public key loaded; stable across restarts

**Tape:** `AC-002-003-first-run-and-subsequent-load.tape`
**BC anchors:** BC-2.09.004 PC-5
**Test file:** `cmd/switchboard/access_admission_test.go`

**Success path:** file present and valid → `pem.Decode` → `x509.ParsePKCS8PrivateKey` → `ed25519.PrivateKey`; loaded public key byte-equal to first-run public key; three simulated "restarts" all return the same public key.

---

## AC-004 — Fail-closed on corrupt PEM or non-Ed25519 key (E-KEY-001)

**Tape:** `AC-004-fail-closed-corrupt-and-non-ed25519.tape`
**BC anchors:** BC-2.09.004 PC-6; E-KEY-001
**Test file:** `cmd/switchboard/access_admission_test.go`

**Error path 1 — corrupt PEM:** truncated/non-PEM data → error `"access: load admission keypair: <path>: PEM decode failed"`.

**Error path 2 — non-Ed25519 key:** valid PKCS#8 PEM with RSA key → error `"access: load admission keypair: <path>: not an Ed25519 key"`.

**No partial startup (adversary F-2):** when keypair load fails after Phase (c) (mgmt server started), `runAccess` joins the mgmt goroutine before returning — no goroutine leak. Verified by goroutine-count baseline check.

---

## AC-005 — File permissions > 0600 → WARNING logged; daemon starts

**Tape:** `AC-005-permissions-warning.tape`
**BC anchors:** BC-2.09.004 PC-4; rulings §1.4
**Test file:** `cmd/switchboard/access_admission_test.go`

**Warning path (F-1 fix):** mode 0644, 0444 (world-readable), 0440 (group-readable) → WARNING in stderr containing both the octal mode string (e.g., `"0444"`) and `"permissions"`. The F-1 fix corrects the predicate from `perm > 0o600` (which silently skipped 0444 since 292 < 384) to `perm&0o077 != 0` (any group or other bit → warn). Daemon continues to start (advisory-not-fatal, OpenSSH posture).

**No-warning path:** mode 0600 → no permissions warning in stderr.

---

## AC-006 — Startup INFO log with base64url pubkey on every start

**Tape:** `AC-006-startup-info-log-base64url-pubkey.tape`
**BC anchors:** BC-2.09.004 PC-7; rulings Decision 4
**Test file:** `cmd/switchboard/access_admission_test.go`

**First-run path:** after keypair generation, stderr contains `"access: admission identity pubkey (register with admin.key.register): <base64url>"` where `<base64url>` is `base64.RawURLEncoding.EncodeToString([]byte(pub))` of the 32-byte public key.

**Subsequent-start path:** same INFO log emitted unconditionally — operators can recover the pubkey from logs after any restart.

---

## AC-007 — discovery.Config.LocalNodeAdmissionPubkey wired from loaded/generated keypair

**Tape:** `AC-007-local-node-admission-pubkey-wired.tape`
**BC anchors:** BC-2.09.004 PC-3e; BC-2.04.008 Precondition 3; rulings Decision 5
**Test file:** `cmd/switchboard/access_admission_test.go`

**Wiring verification:** `[]byte(admissionPrivKey.Public().(ed25519.PublicKey))` — non-nil, length 32 (`ed25519.PublicKeySize`). `disc.Run` with a pre-cancelled context returns `context.Canceled`, not `ErrMissingNodeAdmissionPubkey`, confirming `LocalNodeAdmissionPubkey` was populated before `discovery.New` was called.

**Through-runAccess regression guard (adversary F-6):** the `newDiscovery` package-level seam is overridden in the test to capture the `discovery.Config` argument. The captured `LocalNodeAdmissionPubkey` is asserted byte-equal to the known keypair's public key. This discriminates against the regression of wiring nil or a different key.

---

## AC-008 — Discovery.Run goroutine WG-tracked; ctx.Canceled clean shutdown; no goroutine leak

**Tape:** `AC-008-discovery-run-wg-tracked-clean-shutdown.tape`
**BC anchors:** BC-2.04.008 PC-1 through PC-5; ARCH-01 v1.7 §Goroutine WaitGroup Contract; BC-2.04.008 Invariant 2; rulings Decision 7
**Test file:** `cmd/switchboard/access_admission_test.go`

**WG-tracked goroutine:** `disc.Run` is started inside `runAccessWithConnector` with `wg.Add(1)` before the `go` statement. Verified by HeartbeatObserver channel handshake: a tick to the `TickSource` channel fires the observer, confirming `disc.Run` is live. If `disc.Run` is not started, the observer never fires and the test fails within 300ms.

**Same WaitGroup as sweep/tickers:** `disc.Run` shares `wg` with sweep and frames-dropped ticker goroutines. `wg.Wait()` joins all goroutines. Verified by the same tick/cancel flow plus clean function return.

**ctx.Canceled → nil return:** `context.Canceled` from `disc.Run` does not set `internalFailure`. `runAccessWithConnector` returns `nil` on clean context cancellation (Decision 7 / BC-2.04.008 Invariant 2). Verified: after tick confirms disc started, ctx cancel is issued, return value asserted nil.

**No goroutine leak (blocking HeartbeatObserver discriminator):** disc is constructed with an observer that blocks on a `release` channel. A tick parks `disc.Run` inside the observer. `ctx` is then cancelled. On **stub code** (disc not in wg), `wg.Wait()` returns without disc → `done` closes while disc is parked → `t.Fatal`. On **fixed code** (disc in wg), `wg.Wait()` blocks waiting for disc → `done` stays open for 150ms → bounded-wait branch passes. After the 150ms window, `release` is closed, disc unblocks, `wg.Done()` fires, `wg.Wait()` returns, `done` closes within 2s.

**Startup ordering:** `disc.Run` starts after keypair load and mgmt server start (phases (d)–(f) per rulings §3.1). Verified by the same tick handshake + clean nil return.

---

## M3 — Tilde expansion in admission_key_file path

**Tape:** `M3-tilde-expansion-admission-key-path.tape`
**BC anchors:** M3 / BC-2.07.003 EC-007; rulings §1.2; Decision 2
**Test file:** `cmd/switchboard/access_admission_test.go`

**`~/subdir/adm.pem`:** expands to `$HOME/subdir/adm.pem`. Key file is created under the real `$HOME` (temporarily set via `t.Setenv`). No literal `~` directory created.

**`~` alone:** expands to `$HOME`. Key file is written to `$HOME` itself (directory exists, no new dir created).

**Absolute path:** used as-is with no modification.

**`~username` path:** treated as a literal path (not expanded). The code does not attempt `os.UserHomeDir()` for `~username` patterns — consistent with sbctl's EC-007 semantics. No `"home directory unavailable"` message in stderr.
