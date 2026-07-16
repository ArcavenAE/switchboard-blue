---
artifact_id: interface-definitions
document_type: prd-supplement-interface-definitions
level: L3
version: "1.32"
status: draft
producer: product-owner
timestamp: 2026-07-12T00:00:00
phase: 1a
inputs:
  - '.factory/specs/prd.md'
  - '.factory/specs/domain-spec/capabilities.md'
  - '_bmad-output/planning-artifacts/prd.md'
input-hash: "bf87d3a"
traces_to: '.factory/specs/prd.md'
---

# Interface Definitions: Switchboard

> PRD supplement — extracted from PRD Section 3.
> Referenced by: implementer, test-writer, devops-engineer.

## CLI Interface

The `switchboard` binary is the single daemon binary. It runs in different modes by subcommand.
`sbctl` is the operator CLI. It communicates with running daemons.

### switchboard daemon subcommands

```
switchboard router    [--config=<path>]   # Start router daemon (E or PE based on config)
switchboard access    [--config=<path>]   # Start access node daemon
switchboard console   [--config=<path>]   # Start console daemon
switchboard control   [--config=<path>]   # Start control node daemon
switchboard version                       # Print version and exit
switchboard help                          # Print help and exit

Global flags:
  --config=<path>   Path to YAML config file (default: /etc/switchboard/<mode>.yaml or
                    ./switchboard-<mode>.yaml)
  --log-level=<level>  debug|info|warn|error (default: info)
  --log-format=<fmt>   text|json (default: text)
```

### sbctl operator CLI

```
sbctl [--target=<addr>] [--key=<path>] [--json] [--timeout=<dur>] <subcommand> [args...]

Global flags:
  --target=<addr>   Daemon address (host:port or unix socket path) (default: /run/switchboard-router.sock); when the default socket is absent and --target is not specified, sbctl exits with E-NET-001 (exit 1)
  --key=<path>      Path to operator private key file (default: ~/.ssh/id_ed25519)
  --json            Machine-readable JSON output
  --timeout=<dur>   Connection timeout (default: 5s)

Subcommands:

# SVTN Management
sbctl svtn create [--name=<name>]               # [REMOVED] Alias removed as of Phase 5 Pass 3 Path B remediation (PR #62). Was deprecated in v1.2 (S-6.07). Migration target: 'sbctl admin svtn create'. Invoking today returns exit 2 (unknown subcommand: svtn).
sbctl svtn destroy                              # [WONT-FIX-MIGRATION-SHIM, S-BL.CLI-SURFACE-COMPLETION Ruling 3: `--id=<svtn_id>` never implemented — SVTNManager is exclusively name-keyed, no hex-ID reverse index exists; implementing `--id` literally would require a new data-structure change disproportionate to this story, and silently reinterpreting it as name would be a footgun on a destructive command. `svtn destroy` recognizes the sub-verb and returns a usage error redirecting to the canonical form: `svtn destroy: use 'sbctl admin svtn destroy --name=<svtn-name> [--confirm=<svtn-short-id>|--yes]'`, exit 2. Mirrors the `svtn create` REMOVED disposition (line above) — no RPC dispatch, no confirm-gate duplication. Implementation pending under S-BL.CLI-SURFACE-COMPLETION; current develop tip still returns the generic unknown-subcommand form until it lands.]
sbctl svtn list                                  # [PENDING: won't-fix (S-BL.SVTN-LIST-WIRE, surface removed); returns unknown-subcommand usage error, exit 2 (verified post-#65)]
sbctl svtn status --name=<svtn-name>            # [ADJUDICATED-S-BL.CLI-SURFACE-COMPLETION Ruling 2: wire verb `admin.svtn.status` (BC-2.07.001 v1.14 PC-4); bare top-level dispatch, NOT `sbctl admin`-prefixed (read-only, no confirm-gate risk — same shape as `paths list`/`router status`); flag corrected `--id` → `--name` (SVTNManager is exclusively name-keyed, same defect class as §60). Authority: any admitted role via `resolveCallerAdmissionAnyRole` (list-keys precedent). Implementation pending under S-BL.CLI-SURFACE-COMPLETION; current develop tip returns unknown-subcommand usage error, exit 2 (verified post-#65) until it lands.]

# Key Management (read-only; destructive ops exclusively via sbctl admin)
sbctl svtn keys list [--svtn=<id>]              # [SUPERSEDED by §108 sbctl admin list-keys + §384 admin.key.list-keys (S-6.06, implemented); no svtn case arm today, exit 2 (verified post-#65)]
# NOTE: Destructive key operations (register/revoke/expire) are exclusively via
# `sbctl admin` (operator-only with --confirm gating). See `sbctl admin` table below.

# Session Operations
sbctl sessions list [--svtn=<id>]               # List all SVTN sessions (bare `sbctl sessions` also dispatches sessions.list RPC; verified post-#65)
sbctl sessions attach <session-name> [--svtn=<id>]  # [PENDING-S-BL.DISCOVERY-WIRE: not-implemented usage error, exit 2 (verified post-#65)]
sbctl sessions detach [--session=<name>] [--svtn=<id>]  # [PENDING-S-BL.DISCOVERY-WIRE: not-implemented usage error, exit 2 (verified post-#65)]
sbctl sessions status [--session=<name>]        # [PENDING-S-BL.DISCOVERY-WIRE: not-implemented usage error, exit 2 (verified post-#65)]

# Path / Quality Metrics
sbctl paths list [--svtn=<id>]                  # Per-path RTT (rtt_ms, rtt_p99_ms), loss, status (implemented, S-5.02)
sbctl paths ping --router=<addr>                # [ADJUDICATED-S-BL.CLI-SURFACE-COMPLETION Ruling 1: new wire verb `paths.ping` (BC-2.06.004 PC-1) — one-shot bodyless RTT probe, distinct from `paths.list`'s accumulated metrics; dials `<addr>` directly (overrides `--target`), empty request args, `{"pong": true}` response, sbctl reports client-measured `rtt_ms`. Authority: Tier-1 operator-key auth only. Implementation pending under S-BL.CLI-SURFACE-COMPLETION; current develop tip: `paths` case arm only dispatches `list`; `ping` returns usage error, exit 2 (verified post-#65) until it lands.]

# Router Management
sbctl router status --target <router>            # Alias for sbctl paths list + quality column (BC-2.06.003 v1.13 PC-3; S-5.02 v1.9 AC-003/AC-008) — implemented
sbctl router metrics --svtn=<id>                # Frame counts, HMAC failures, drop cache hits — implemented (S-5.02)
sbctl router reload                             # [ADJUDICATED-S-BL.CLI-SURFACE-COMPLETION Ruling 4: new wire verb `router.reload`, router-mode-exclusive (governance-only addendum, BC-2.09.001 v1.2 PC-1) — bridges into the already-shipped SIGHUP-reload code path via `sighupCh` reuse; resolves DRIFT-HS006-DRAIN-CLI-MISSING (reload half). Authority: Tier-1 operator-key auth only. Implementation pending under S-BL.CLI-SURFACE-COMPLETION; current develop tip: router case arm dispatches only `metrics`/`status`; `reload` returns unknown-subcommand usage error, exit 2 (verified post-#65) until it lands.]
sbctl router drain                              # [ADJUDICATED-S-BL.CLI-SURFACE-COMPLETION Ruling 4: new wire verb `router.drain`, router-mode-exclusive (governance-only addendum, BC-2.09.002 v1.3 Trigger/PC-1) — bridges into the already-shipped SIGTERM/drain shutdown sequence via new `drainRequestCh`; resolves DRIFT-HS006-DRAIN-CLI-MISSING (drain half). Authority: Tier-1 operator-key auth only. Connection is expected to be severed as the daemon exits (best-effort, no wire ACK). Implementation pending under S-BL.CLI-SURFACE-COMPLETION; current develop tip: router case arm dispatches only `metrics`/`status`; `drain` returns unknown-subcommand usage error, exit 2 (verified post-#65) until it lands.]

# Console Control (remote)
# NOTE (F-P5P6-A-004 adjudication, verified against cmd/sbctl/console.go post-#65):
# The converged S-7.03 implementation registers only --session; --console and --svtn flags are NOT present.
# Spec amended below to match the converged flag set (authority: F-P5P6-A-004 + S-7.03 convergence record 3/3 clean passes).
sbctl console attach --session=<name>           # Attach to named tmux session (wire: console.attach; BC-2.08.001 PC-1; S-7.03 AC-001)
sbctl console detach                             # Detach from current session (wire: console.detach; BC-2.08.001 PC-2; S-7.03 AC-002)
sbctl console switch --session=<name>           # Atomically detach + attach to named session (wire: console.switch; BC-2.08.001 PC-3; S-7.03 AC-003)

# Diagnostics
sbctl version                                   # [PENDING-S-BL.PING-VERSION-WIRE: no 'version' case arm in main.go switch; returns unknown-subcommand usage error, exit 2 (verified post-#65)]
sbctl ping                                      # [PENDING-S-BL.PING-VERSION-WIRE: no 'ping' case arm in main.go switch; returns unknown-subcommand usage error, exit 2 (verified post-#65)]
```

### `sbctl admin`

Operator-only subcommand requiring `--confirm` token. Used for SVTN lifecycle management, key management, and emergency recovery.

#### Key management (`sbctl admin key`)

Nested form — all destructive key operations use `sbctl admin key <verb>`:

| Subcommand | Purpose | Auth | Exit codes |
|-----------|---------|------|-----------|
| `sbctl admin key register --svtn <svtn-name> --key <openssh-pubkey> [--role <control\|console\|access>] [--confirm=<svtn-short-id>\|--yes]` | Register a new admission key; duplicate registration is last-write-wins role update per ADR-003 (no error) | Requires existing control-role key + interactive `--confirm` token | 0=ok, E-CFG-001 (invalid/missing inputs: missing pubkey, pubkey decode failure, invalid role enum, missing svtn_id), E-ADM-009 (insufficient authority), E-CFG-012 (--yes + --confirm combined), E-CFG-013 (non-interactive session with neither --confirm nor --yes), E-SVTN-003 (SVTN not found) |
| `sbctl admin key revoke --svtn <svtn-name> --key <openssh-pubkey> --role <control\|console\|access> [--confirm]` | Revoke admission key | Requires existing control-role key; `--confirm` is a boolean wire flag (bare `--confirm` or `--confirm=true`), enforced daemon-side as a conditional gate for control-to-control revocation (E-ADM-018, exit 1); no interactive prompt, no `--yes`, no E-CFG-012/E-CFG-013 on the revoke path (see §131 carve-out note). Per ADR-004 console cannot revoke control. | 0=ok, E-ADM-019 (role mismatch: claimed role does not match stored role — `"E-ADM-019: role mismatch: claimed role <role> does not match registered key role <role> for key <fp>"`), E-ADM-018 (control-to-control revocation requires confirmation — `"E-ADM-018: control-to-control revocation requires explicit confirmation: use --confirm to proceed"`), E-ADM-013 (not found) |
| `sbctl admin key expire --svtn <svtn-name> --key <openssh-pubkey> --after <duration>` | Set automatic expiry on an admission key. The operator supplies a Go duration string (e.g. `"24h"`) as `--after`; the CLI sends the string verbatim as the `after` wire field without any transformation (verified: `cmd/sbctl/admin.go`; wire struct `adminKeyExpireArgs.After string \`json:"after"\`` at `cmd/sbctl/admin.go`). Client-side pre-validation rejects zero or negative durations before dialing (`cmd/sbctl/admin.go`, `usageErrf` → exit 2). Server independently validates the `after` field: positive and ≤100 years (`cmd/switchboard/admin_handlers.go`). **Correction note (v1.22):** The v1.6 changelog entry (line 168) described an `--at <RFC3339-timestamp>` flag with CLI-side RFC3339→duration translation — that design was never implemented. The converged CLI shape is `--after <duration>` (Go duration syntax). The v1.6 changelog line is preserved (history is immutable); this v1.22 entry is the authoritative correction. | Requires existing control-role key; no `--confirm` required (non-destructive scheduling) | 0=ok, E-ADM-013 (key not found), E-CFG-001 split by layer: zero/negative duration → client-side, exit 2, token present in stderr (`cmd/sbctl/admin.go`); >100 years → daemon-side, exit 1 (`cmd/switchboard/admin_handlers.go`), E-ADM-021 (bootstrap-key-expire-forbidden — `mapAdminError` arm for `ErrBootstrapKeyExpireForbidden`, `admin_handlers.go`; the bootstrap key is a permanent trust anchor and cannot be expired), E-ADM-009 (insufficient authority — `resolveAndVerifyCallerRole`, `admin_handlers.go`; caller key not active in SVTN's admitted key set), E-SVTN-003 (SVTN not found — `mapAdminError` arm for `ErrSVTNNotFound`, `admin_handlers.go`; SVTN does not exist in registry) |
| `sbctl admin list-keys --svtn <svtn-name>` | List all admission keys with role, fingerprint, expiry | Any admitted role (any active role in target SVTN) OR operator-set member OR daemon bootstrap key — the AUTHORITY gate (F-L2-003) is bypassed (no control-only requirement) but the ADMISSION gate still applies (see BC-2.05.004:155 and EC-008) | 0=ok, E-SVTN-003 (SVTN not found, daemon-side, exit 1 — `makeListKeysHandler` calls `m.ListKeys(a.SVTNName)` at `admin_handlers.go`; on error routes through `mapAdminError` whose `ErrSVTNNotFound` arm at `admin_handlers.go` returns `svtnNotFoundErr` → wire `"E-SVTN-003: SVTN not found: <name>"`), E-CFG-001 (missing `--svtn`, client-side, exit 2 — `cmd/sbctl/admin.go` rejects empty flag via `usageErrf`) |

> **`--svtn` placeholder semantics (F-P5P12-A-002):** The `--svtn` flag in all `sbctl admin key` and `sbctl admin list-keys` commands takes the **SVTN NAME** — the value passed to `sbctl admin svtn create --name=...`. It does NOT take the hex `svtn_id` returned in the create response. The wire field is named `svtn_id` for historical wire-compat (Burst 19 alignment; wire structs at `cmd/switchboard/admin_handlers.go` carry `json:"svtn_id"` but Go field `SVTNName`), but all `admin.key.*` requests carry the name in that field (see §396-399 Registered Verbs — `{"svtn_id": "<svtn-name>"}`). The daemon SVTN lookup is name-keyed: `internal/svtnmgmt/svtnmgmt.go` accesses `m.svtns[svtnName]` map (verified at `:254`, `:300`, `:370`); there is no hex-id lookup path for key operations. Pasting the hex `svtn_id` from a create response into `--svtn` will produce `E-SVTN-003: SVTN not found: <the-hex>`. Use the name you supplied to `--name=` at creation time.

`--key <openssh-pubkey>` — Replaces the former `--key-fingerprint <fp>` flag. Accepts an OpenSSH-format public key string (e.g. `ssh-ed25519 AAAA... comment`). The CLI marshals this as the `pubkey_openssh` wire field sent to `internal/svtnmgmt`. Previously accepted a raw hex-encoded public key (`pubkey_hex`); `pubkey_openssh` is the canonical wire field name as of interface-definitions v1.13. The daemon-side `decodePublicKey()` accepts both OpenSSH format (primary path, via `ssh.ParseAuthorizedKey`) and raw base64-encoded 32-byte Ed25519 key material (fallback, for backward compatibility with clients that have not yet migrated to OpenSSH format). New clients MUST send OpenSSH format; the base64 fallback is deprecated and may be removed in a future version.

#### SVTN lifecycle (`sbctl admin svtn`)

| Subcommand | Purpose | Auth | Exit codes |
|-----------|---------|------|-----------|
| `sbctl admin svtn create --name=<svtn-name>` | Create a new SVTN; returns `svtn_id` and bootstrap fingerprint | Bootstrap-only: caller MUST authenticate with the daemon bootstrap key (RoleControl); cross-SVTN control-role keys are not accepted. See §380 and BC-2.07.001 Inv-3. | 0=ok, E-SVTN-001 (already exists), E-ADM-009 (insufficient authority: bootstrap key required), E-CFG-001 (invalid SVTN name: empty / whitespace-only / >255 bytes / invalid UTF-8 / control chars), E-INT-001 (internal error wrap on non-duplicate Create failure) |
| `sbctl admin svtn destroy --name=<svtn-name> [--confirm=<svtn-short-id>\|--yes]` | Destroy an SVTN and all admitted keys; terminates active sessions | Requires control-role key + `--confirm` | 0=ok, E-ADM-011 (unauthorized), E-ADM-009 (insufficient role), E-CFG-001 (invalid SVTN name: empty / whitespace-only / >255 bytes / invalid UTF-8 / control chars), E-SVTN-003 (SVTN not found — `mapAdminError` arm for `ErrSVTNNotFound` via `resolveAndVerifyCallerRole` → `m.Destroy` path, `admin_handlers.go`) |

#### Emergency recovery

> **PENDING-S-BL.ADMIN-RECOVER-WIRE:** Spec-complete; implementation deferred to backlog story `S-BL.ADMIN-RECOVER-WIRE`. Neither `cmd/sbctl` nor the daemon dispatch `admin recover` today — the `runAdmin` switch covers only `key | list-keys | svtn`; the default arm returns `admin: unknown subcommand %q`. Operators invoking `sbctl admin recover` on current builds receive exit 2 (usage error, verified against PR #65 — `runAdmin` default arm returns `usageErrf`, main.go maps `*usageError` → `os.Exit(2)` via `errors.As`).

| Subcommand | Purpose | Auth | Exit codes |
|-----------|---------|------|-----------|
| `sbctl admin recover --svtn <svtn-name> --bootstrap-key <path> --confirm <svtn-short-id>\|--yes` | Emergency recovery when all control keys are lost | Requires bootstrap key (set at SVTN creation per BC-2.07.001) + interactive `--confirm` token | 0=ok, E-ADM-014 (bootstrap mismatch) |
> **PENDING-S-BL.ADMIN-RECOVER-WIRE:** Row above is spec-complete; CLI and daemon dispatch not yet implemented. See annotation above.

**`--confirm=<svtn-short-id>`** — Required on all destructive admin operations that use `runDestroyConfirmGate` (key register, svtn destroy, recover — **not key revoke**). Accepts the SVTN short ID (the first 8 hex characters of the SVTN ID, formatted as `SVTN-<short-id>`) as the confirmation token. Prevents accidental mass-revocation by requiring the operator to name the target SVTN explicitly. When the flag is omitted, the command enters interactive mode and prompts `Type SVTN-<short-id> to confirm:` before proceeding. Per ADR-004 split-brain mitigation.
> **Interim rendering (DRIFT-P5P4-PROMPT-SHORTID):** Until the CLI can resolve the actual SVTN short-id from the daemon response, the prompt MAY render as a static-example form (e.g. `"Type the SVTN short-ID (e.g. SVTN-abcd1234) to confirm: "`). Both forms satisfy the confirmation gate; the substitution form is the canonical long-term target.
> **`key revoke` carve-out (F-P5P11-A-001):** `sbctl admin key revoke` does NOT use this interactive-confirm family. Its `--confirm` flag is a boolean wire flag (bare `--confirm` or `--confirm=true`; registered as `boolStringFlag` at `cmd/sbctl/admin.go`; `isTrue()` at `admin.go`). Enforcement is daemon-side and conditional: E-ADM-018 is emitted by `SVTNManager.RevokeKey` only when a control-role key revokes another control-role key without `--confirm` (error-taxonomy.md E-ADM-018; taxonomy v4.4 adjudication is the authority for this shape). There is no interactive prompt, no `--yes` flag, and no E-CFG-012/E-CFG-013 on the revoke path (confirmed: `runAdminKeyRevoke` at `cmd/sbctl/admin.go` does not call `runDestroyConfirmGate`; contrast with destroy at `admin.go` and register at `admin.go`).

**`--yes`** — Bypasses the `--confirm` interactive prompt for scripted use. Emits a warning to stderr: `"WARNING: --yes bypasses confirmation; ensure correct --name target before scripting"`. Cannot be combined with `--confirm` (usage error, exit 2). Applies to the `runDestroyConfirmGate` family (destroy, register; recover pending) — **not applicable to key revoke** (see carve-out note above).
> **Note (F-P5P9-A-006):** The flag name interpolated in the `--yes` warning is command-specific: `sbctl admin svtn destroy` emits `--name` (`runDestroyConfirmGate` called with `targetFlag="--name"`, `admin.go`); `sbctl admin key register` emits `--svtn` (`runDestroyConfirmGate` called with `targetFlag="--svtn"`, `admin.go`). The impl-quoted string above (`--name`) is the destroy form and MUST NOT be changed; register's `--svtn` form is distinct.

Confirmation flow summary (applies to the `runDestroyConfirmGate` family: svtn destroy, key register, recover pending): interactive commands prompt for `Type SVTN-<short-id> to confirm:` when `--confirm` is not supplied on the command line. Providing `--confirm=<svtn-short-id>` satisfies the check non-interactively. `--yes` bypasses the check entirely with a stderr warning. Combining `--yes` with `--confirm` is a usage error (E-CFG-012, exit 2). In a non-interactive session (no TTY) where neither `--confirm` nor `--yes` is supplied, the command exits with E-CFG-013 (exit 2) — use `--confirm=<svtn-short-id>` or `--yes` for scripted invocations. **`sbctl admin key revoke` is not part of this family** — see `key revoke` carve-out note under §131.
> **Interim rendering (DRIFT-P5P4-PROMPT-SHORTID):** Until the CLI can resolve the actual SVTN short-id from the daemon response, the prompt MAY render as a static-example form (e.g. `"Type the SVTN short-ID (e.g. SVTN-abcd1234) to confirm: "`). Both forms satisfy the confirmation gate; the substitution form is the canonical long-term target.

> **v1.32 changelog note (2026-07-16):** M2 remediation (S-BL.NODE-ADMISSION-PROVISIONING, BC-2.09.004 PC-2). Added `admission_key_file` to the Access node config YAML block (§Access node config). Field is optional (string); default path `/var/lib/switchboard/access-admission-identity.pem` applied at daemon startup by `runAccess`, not by `Config.Validate`. Semantics: path to the node's persistent Ed25519 admission keypair (PKCS#8 PEM); tilde expanded per `loadEd25519Key` conventions (BC-2.07.003 EC-007); first-run generates atomically at mode 0600; subsequent runs load the existing key; fail-closed on corrupt/non-Ed25519 file. Validation: E-CFG-014 — non-whitespace when present; no file I/O in `Validate` (ARCH-06 §Config purity contract). Access-mode only. Cross-ref: BC-2.09.004 PC-1/PC-2, BC-2.09.003 PC-12. Input-hash updated (bf87d3a; was 226ea28 — pre-existing drift from inputs unchanged since v1.31).

> **v1.31 changelog note (2026-07-12):** F-CS-SP1-002 (MED, spec-adversarial pass 1) remediation. §60 `svtn destroy` redirect literal corrected — removed a spurious `usage: ` prefix from the quoted error text. The authoritative sources agree without the prefix: story AC-009 PC-1 mandates the exact text `svtn destroy: use 'sbctl admin svtn destroy --name=<svtn-name> [--confirm=<svtn-short-id>|--yes]'`; the Ruling 3 code snippet in `.factory/decisions/S-BL.CLI-SURFACE-COMPLETION-rulings.md` agrees; `cmd/sbctl/main.go` prints the error verbatim with no framework `usage:` prefix. No change to the command's exit code (2) or any other CLI listing row. Refs: F-CS-SP1-002.

> **v1.30 changelog note (2026-07-12):** S-BL.CLI-SURFACE-COMPLETION architect rulings executed (`.factory/decisions/S-BL.CLI-SURFACE-COMPLETION-rulings.md`). Four Registered Verbs rows added: `admin.svtn.status` (Ruling 2, BC-2.07.001 v1.14 PC-4, any-admitted-role authority via `resolveCallerAdmissionAnyRole`), `paths.ping` (Ruling 1, new BC-2.06.004 PC-1, Tier-1-only, bodyless RTT probe distinct from `paths.list`), `router.reload` and `router.drain` (Ruling 4, governance-only addenda to BC-2.09.001/BC-2.09.002, router-mode-exclusive, bridge into the already-shipped SIGHUP-reload/SIGTERM-drain code paths — resolves `DRIFT-HS006-DRAIN-CLI-MISSING`). CLI listing corrections: §60 `svtn destroy` reclassified `PENDING` → won't-fix migration shim (Ruling 3 — `--id` never implementable, `SVTNManager` is exclusively name-keyed; redirects to `sbctl admin svtn destroy --name=...`, mirrors the `svtn create` REMOVED precedent); §62 `svtn status` flag corrected `--id` → `--name`, reclassified `PENDING` → adjudicated (bare top-level dispatch, not `sbctl admin`-prefixed); §77 `paths ping` and §82-83 `router reload`/`router drain` reclassified `PENDING-S-BL.CLI-SURFACE-COMPLETION` → adjudicated (design settled; CLI/RPC implementation still pending, exit-2 behavior unchanged until it lands — the PENDING marker retired per the ruling's summary table, not because the code shipped). No change to any already-implemented verb's contract. Refs: architect ruling 2026-07-12.

> **v1.29 changelog note (2026-07-04):** Burst 82 taxonomy remediation (F-P5P34-A-001 + F-P5P34-A-002, OBS-P5P34-A-002). §JSON Output Schema extended with `error.code` closed-set enumeration note cross-referencing `error-taxonomy.md` v4.7. Enumerates: top-level sbctl codes (E-NET-001, E-ADM-010, E-CFG-010, E-RPC-001, E-RPC-002); in-band handler codes (E-RPC-010, E-RPC-011, handler-specific families); `router.metrics`/`router.status`-specific in-band codes (E-RPC-002 via `metrics.ErrDecodeArgs` E-RPC-011 wrap; E-RPC-003 via `metrics.ErrInvalidParams` E-RPC-011 wrap). No substantive shape change — the enumeration is a reference-by-cross-reference addition (POL-003 Exception A governance-only bump). Frontend of POL-002 sibling-sweep: error-taxonomy.md v4.7 + interface-definitions.md v1.29 committed together. S-6.07 §Wire Envelope Contract already enumerates E-RPC-002 as transport exception; no edit required on S-6.07 (noted in Burst 82 report). Refs: F-P5P34-A-001, F-P5P34-A-002.

> **v1.28 changelog note (2026-07-03):** POL-001. Phase 5 Pass 17 adversarial remediation. (a) Remove `svtn_id` phantom field from `router.metrics` response — envelope example at §Router-metrics-response and Registered Verbs table Response Data column both listed a `svtn_id` echo field the daemon never emits. Wire response type is `RouterMetricsResponse{FrameCount, HMACFailCount, DropCacheHits, PathDistribution}` per `internal/metrics/types.go`; CLI decode struct matches at `cmd/sbctl/router_metrics.go`; canonical BC-2.06.003 test vector at `.factory/specs/behavioral-contracts/ss-06/BC-2.06.003.md:109` omits svtn_id. Request payload correctly retains `svtn_id` (unchanged). (b) Correct `path_distribution` example values from fractional ratios (0.52, 0.48) to integer frame counts (900000, 334567) — wire type is `map[string]uint64` per `types.go` and `router_metrics.go`; BC-2.06.003 PC-2 documents per-path frame distribution as count, not ratio; demo-evidence `stub_daemon.go` emits integer counts. Ratio example would not deserialize into the actual wire type under any typed consumer. Addresses DRIFT-P5P17-A-001 (svtn_id phantom field) + DRIFT-P5P17-A-002 (fractional path_distribution ratios). Refs: F-P5P17-A-001, F-P5P17-A-002.

> **v1.27 changelog note (2026-07-03):** POL-001. Remove undocumented `$schema` field from JSON envelope success example; correct "share a common envelope" prose to reflect that sbctl emits a schemaless envelope. Normalizes spec to observable sbctl behavior at `cmd/sbctl/client.go` (jsonEnvelope struct has no Schema field) and `cmd/sbctl/main.go` (constructors never populate one). Addresses F-P5P16-A-001. No production domain exists to host the referenced schema URL.

> **v1.26 changelog note (2026-07-03):** Phase 5 Pass 15 adversarial remediation (F-P5P15-A-001 [MED]). Registered Verbs Response Data column corrections: (1) `admin.key.register` row: `{"ok": true}` → `{"key_fingerprint": "SHA256:<base64>", "timestamp": "<RFC3339>"}` — wire response is `adminKeyResult{Fingerprint, At}` per `cmd/switchboard/admin_handlers.go` (struct) and return sites at `:215-218`; (2) `admin.key.revoke` row: same correction, return site `:257-260`; (3) `admin.key.expire` row: same correction, return site `:336-339`; (4) `admin.svtn.destroy` row: `{"ok": true, "status": "destroyed"}` → `{"status": "destroyed"}` — outer envelope `ok/error` is documented in the response envelope section and does not belong in the Response Data column; wire return is `struct{Status string \`json:"status"\`}{Status: "destroyed"}` per `admin_handlers.go`. All corrections verified against BC-2.05.004 PC-4 (fingerprint + timestamp postcondition for key lifecycle operations) and develop tip `6deda15`. Refs: F-P5P15-A-001.

> **v1.25 changelog note (2026-07-03):** Phase 5 Pass 13 sharpening (F-L2-003 admission gate distinction). BC-2.05.004:155 sharpened: F-L2-003 removes the CONTROL-only AUTHORITY gate but NOT the ADMISSION gate; cross-SVTN callers denied with E-ADM-009 (CWE-862 defense against cross-SVTN roster enumeration). BC-2.05.004 EC-008 added: enumerates the three reachable list-keys admission failure modes — (1) missing CallerPubkey / no ambient bootstrap identity → E-ADM-009; (2) CallerPubkey present but not registered on target SVTN AND not in operator-set AND not bootstrap → E-ADM-009; (3) CallerPubkey present, registered on target SVTN, but revoked/expired (registered-any-state true, active false) → E-ADM-009. VP-075:114 Scope exclusion paragraph expanded: F-L2-003 scope exclusion clarified — authority gate only, not admission gate; BC-2.05.004:155 and EC-008 cross-referenced; CWE-862 cited. §111 auth column sharpened from "Any admitted role" to enumerate the full admission requirement (any active role OR operator-set OR bootstrap key) with explicit note that AUTHORITY gate is bypassed but ADMISSION gate is not. BC-2.05.004 bumped v1.12 → v1.13; VP-075 bumped v1.6 → v1.7. Ref: PR #69 (merged squash 03ce8e7).

> **v1.24 changelog note (2026-07-03):** Phase 5 Pass 12 spec-side remediation (Burst 35). F-P5P12-A-001 [MED]: §111 `sbctl admin list-keys` exit-code column extended from "0=ok" to include E-SVTN-003 (SVTN not found, daemon-side, exit 1 — `makeListKeysHandler` calls `m.ListKeys(a.SVTNName)` at `cmd/switchboard/admin_handlers.go`; on error routes through `mapAdminError` whose `ErrSVTNNotFound` arm at `admin_handlers.go` returns `svtnNotFoundErr` → wire `"E-SVTN-003: SVTN not found: <name>"`) and E-CFG-001 (missing `--svtn`, client-side, exit 2 — `cmd/sbctl/admin.go` rejects empty flag via `usageErrf`). F-P5P12-A-002 [MED]: `--svtn <id>` placeholder corrected to `--svtn <svtn-name>` in all four admin key rows (§108 register, §109 revoke, §110 expire, §111 list-keys); `--svtn` takes the SVTN NAME (the value from `--name=` at creation time), not the hex `svtn_id` from the create response. Wire field is `svtn_id` for historical compat (Burst 19; wire structs at `cmd/switchboard/admin_handlers.go` carry `json:"svtn_id"` with Go field `SVTNName`); daemon lookup is name-keyed (`internal/svtnmgmt/svtnmgmt.go` `m.svtns[svtnName]` at `:254`, `:300`, `:370`). Placeholder-semantics note added after §111 row citing §396-399 and verified name-lookup anchors. OBS-P5P12-A-002 (OBS-driven consistency touch): `[--confirm=<svtn-short-id>|--yes]` added to §108 register and §120 destroy syntax cells to match §109 revoke `[--confirm]` — all three are in the `runDestroyConfirmGate` family (destroy at `cmd/sbctl/admin.go`, register at `cmd/sbctl/admin.go`; E-CFG-012/E-CFG-013 in their exit-code columns); §131-§137 prose remains authoritative for semantics. Refs: F-P5P12-A-001, F-P5P12-A-002, OBS-P5P12-A-002. Class sweep: §130 recover row (PENDING-S-BL.ADMIN-RECOVER-WIRE) placeholder likewise corrected to `<svtn-name>`.

> **v1.23 changelog note (2026-07-03):** Phase 5 Pass 11 spec-side remediation (Burst 33). F-P5P11-A-001 [HIGH]: §131 `--confirm=<svtn-short-id>` prose corrected to remove `key revoke` from the `runDestroyConfirmGate` interactive-confirm family — the list now reads "key register, svtn destroy, recover" with an explicit carve-out note for `key revoke`. The carve-out documents: `key revoke --confirm` is a boolean wire flag (bare `--confirm` or `--confirm=true`; `boolStringFlag` registered at `cmd/sbctl/admin.go`; `isTrue()` at `admin.go`); enforcement is daemon-side and conditional (`SVTNManager.RevokeKey` emits E-ADM-018 only on control-to-control revocation without `--confirm`; error-taxonomy.md v4.4 is the adjudication authority); no interactive prompt, no `--yes`, no E-CFG-012/E-CFG-013 on the revoke path (confirmed: `runAdminKeyRevoke` at `cmd/sbctl/admin.go` does not call `runDestroyConfirmGate`; contrast destroy at `admin.go`, register at `admin.go`). §137 confirmation flow summary scoped to the `runDestroyConfirmGate` family (destroy, register; recover pending) with a closing reference to the §131 carve-out. F-P5P11-A-002 [MED]: §109 `sbctl admin key revoke` syntax corrected — `--role <control|console|access>` added as REQUIRED with no default (registered at `cmd/sbctl/admin.go`; required-check at `admin.go`; enum validation at `admin.go`); distinct from `key register` §108 where `--role` is optional defaulting to `console`. `[--confirm]` shown as optional boolean in syntax cell (aligned with Fix 1 shape). Auth cell updated to document boolean wire-flag confirm and daemon-side enforcement. Refs: F-P5P11-A-001, F-P5P11-A-002.

> **v1.22 changelog note (2026-07-03):** Phase 5 Pass 10 spec-side remediation (Burst 31). F-P5P10-A-001 [HIGH]: §110 `sbctl admin key expire` syntax corrected from phantom `--at <RFC3339-timestamp>` to `--after <duration>` (Go duration syntax, e.g. `"24h"`; flag registered at `cmd/sbctl/admin.go`). CLI-side prose rewritten to the actual semantics: the operator supplies a Go duration string that is sent verbatim as the `after` wire field with no transformation (verified: `adminKeyExpireArgs.After` at `cmd/sbctl/admin.go`; `connectAndRun` call at `cmd/sbctl/admin.go`); daemon parses duration independently at `cmd/switchboard/admin_handlers.go`. Correction note added to §110 body: the v1.6 changelog `--at` RFC3339 design was never implemented; the converged CLI shape is `--after`. The v1.6 changelog line (line 168) is preserved (history immutable). F-P5P10-A-002 [MED]: §110 E-CFG-001 exit-class split documented — zero/negative duration → client-side validation (`cmd/sbctl/admin.go`), `usageErrf` → exit 2, token present in stderr; >100 years → daemon-side validation (`cmd/switchboard/admin_handlers.go`), server error → exit 1. §186 exit-2 row extended to include E-CFG-001 (client-side flag-value validation) alongside E-CFG-012/E-CFG-013. Refs: PR #68, F-P5P10-A-001, F-P5P10-A-002.

> **v1.21 changelog note (2026-07-03):** Phase 5 Pass 9 spec-side remediation (Burst 29). F-P5P9-A-001 [HIGH]: §94-95 `sbctl version` + `sbctl ping` annotated PENDING-S-BL.PING-VERSION-WIRE — neither has a case arm in the `main.go` switch (`cmd/sbctl/main.go`); both route to the `default` arm and return unknown-subcommand `usageErrf`, exit 2 (verified post-#65). F-P5P9-A-002 [MED]: `--target` flag entry §51 extended with default value `/run/switchboard-router.sock` (`cmd/sbctl/main.go`) and E-NET-001 (exit 1) consequence when the default socket is absent; §370 flag-interactions table row for `--target` updated to match. F-P5P9-A-003 [MED]: §110 expire exit-code column extended with E-ADM-021 (bootstrap-key-expire-forbidden — `mapAdminError` arm for `ErrBootstrapKeyExpireForbidden`, `admin_handlers.go`), E-ADM-009 (insufficient authority — `resolveAndVerifyCallerRole`, `admin_handlers.go`), E-SVTN-003 (SVTN not found — `mapAdminError` arm for `ErrSVTNNotFound`, `admin_handlers.go`); E-CFG-012/E-CFG-013 NOT added (expire has no confirm gate — `runAdminKeyExpire` never calls `runDestroyConfirmGate`; confirmed `cmd/sbctl/admin.go`). F-P5P9-A-004 [LOW]: §120 destroy exit-code column extended with E-SVTN-003 (reachable via `mapAdminError` after `resolveAndVerifyCallerRole` → `m.Destroy` path, `admin_handlers.go`). F-P5P9-A-005 [LOW]: §48 synopsis reflowed to match impl usage line `main.go` verbatim: added `[--timeout=<dur>]` and `[args...]`. F-P5P9-A-006 [LOW]: §128 `--yes` warning text carries `--name` (destroy form); footnote added clarifying flag-name interpolation is command-specific — destroy uses `--name` (`admin.go`), register uses `--svtn` (`admin.go`); impl-quoted destroy string unchanged.

> **v1.20 changelog note (2026-07-03):** Phase 5 Pass 8 spec-side remediation (Burst 27). F-P5P8-A-002 [HIGH]: §108 register exit-code column replaced — E-ADM-012 (no such sentinel on the register path; `RegisterKey` is unconditional last-write-wins per ADR-003, `internal/svtnmgmt/svtnmgmt.go`) and E-ADM-018 (belongs only on the revoke path; `ErrControlRevocationRequiresConfirm` is returned by `RevokeKeyIfRoleMatches`, not `RegisterKey`) removed; actual register errors documented (E-CFG-001 for invalid/missing inputs including pubkey decode failure at `admin_handlers.go`, E-ADM-009 for insufficient authority, E-CFG-012/E-CFG-013 for confirm-gate misuse, E-SVTN-003 for missing SVTN); LWW note added per ADR-003. F-P5P8-A-005 [MED]: §109 revoke hierarchy error corrected from E-ADM-011 to E-ADM-019 (`mapAdminError` maps `ErrRoleMismatch` → E-ADM-019 at `admin_handlers.go`); emission text added verbatim; E-ADM-011 maps only from `ErrDestroyUnauthorized` on the destroy path (`admin_handlers.go`); E-ADM-018 added to §109 enumeration (control-to-control revocation gate, reachable via `RevokeKeyIfRoleMatches` per `svtnmgmt.go`, emission text added verbatim). F-P5P8-A-007 [LOW]: `<hex-pubkey>` placeholders in §108 and §109 row headers corrected to `<openssh-pubkey>` (impl primary path is `ssh.ParseAuthorizedKey` per `admin_handlers.go`; §113 already uses openssh). F-P5P8-A-003 [MED] (adjudicated spec-side): §108 syntax corrected from implied-required `--role` to optional `[--role <control|console|access>]` with explicit note "defaults to `console` when omitted" (verified: `cmd/sbctl/admin.go` `fs.String("role", "console", ...)`). Authority note §395 corrected: E-ADM-012 removed (no-error LWW noted), E-ADM-011 scope qualified (destroy path only, not revoke), E-ADM-019 added for revoke hierarchy violation, E-ADM-018 corrected scope to revoke confirm gate. PR #67 reference.

> **v1.19 changelog note (2026-07-03):** Phase 5 Pass 6 spec-side remediation (Burst 23). F-P5P6-A-002: §121 PENDING-S-BL.ADMIN-RECOVER-WIRE exit-2 claim verified against merged main.go/admin.go (PR #65) — `runAdmin` default arm returns `usageErrf`, main() discriminates via `errors.As(err, &*usageError)` → `os.Exit(2)`; annotation updated to "verified against PR #65". F-P5P6-A-005: seven unimplemented CLI verbs annotated with PENDING markers and verified current behavior (exit 2, usage error, post-#65): §77 `paths ping`, §82 `router reload`, §83 `router drain`, §60 `svtn destroy`, §62 `svtn status` (collective marker PENDING-S-BL.CLI-SURFACE-COMPLETION); §61 `svtn list` (won't-fix S-BL.SVTN-LIST-WIRE); §70-73 `sessions attach/detach/status` (PENDING-S-BL.DISCOVERY-WIRE, matching existing backlog story). F-P5P6-A-005 secondary: §65 `sbctl svtn keys list` annotated as superseded by §108 `sbctl admin list-keys` (canonical, implemented S-6.06) and §384 `admin.key.list-keys` Registered Verbs row. F-P5P6-A-001/A-006 spec side: §174 exit-code table — exit-2 row extended to name bare invocation (no subcommand) explicitly; `--help`/`-h` added to exit-0 row. F-P5P6-A-004 adjudication (spec-side amendment): §86-88 `console attach/detach/switch` amended — prior spec mandated `--console=<addr>` and `--svtn` flags not present in the converged S-7.03 implementation; amended to the real flag set (`--session` only for attach/switch; no flags for detach), with changelog note citing F-P5P6-A-004 + S-7.03 convergence record 3/3 clean passes as authority (verified against `cmd/sbctl/console.go` runConsoleAttach/Detach/Switch post-#65).

> **v1.18 changelog note (2026-07-03):** Phase 5 Pass 5 adversarial remediation (Burst 21). F-P5P5-A-001: §116 authority cell corrected from "Requires control-role key on control-mode daemon" to bootstrap-only language aligned with impl (`admin_handlers.go`), Registered Verbs §380, and BC-2.07.001 Inv-3/Ruling-5. F-P5P5-A-002: §119-123 Emergency recovery section annotated with `PENDING-S-BL.ADMIN-RECOVER-WIRE` (spec-complete, implementation deferred; operators get exit 2 unknown-subcommand today); §125 confirm-gate mention carries sibling annotation. F-P5P5-A-003: §116 exit-codes column extended with E-CFG-001 (five `validateSVTNName` validation arms) and E-INT-001 (non-duplicate Create failure wrap); §117 exit-codes column extended with E-CFG-001 for caller-supplied-name validation path. F-P5P5-A-004: §59 `sbctl svtn create` alias status changed from DEPRECATED/retained to REMOVED as of Phase 5 Pass 3 Path B remediation (PR #62); migration target and exit-2 behaviour noted.

> **v1.17 changelog note (2026-07-02):** Pass-11 adversarial corrections: §375/v1.15 changelog corrects role field description (target-key role for HOLD-001 E-ADM-019 cross-check, not caller authorization); §128 --yes warning corrects --svtn→--name (destroy uses --name); §129/§130 add E-CFG-013 reference for non-interactive scripting guard. Refs: F-11A-2/3/4.

> **v1.16 changelog note (2026-07-02):** Pass-8 adversarial corrections: §129 cross-reference updated from obsolete E-CFG-006 to E-CFG-012 (taxonomy realignment from v4.3); §125/§129 interactive prompt spec annotated to authorize static-example interim rendering pending DRIFT-P5P4-PROMPT-SHORTID resolution. Refs: F-8A-M1/M2.

> **v1.15 changelog note (2026-07-02):** Pass-7 adversarial correction: admin.key.revoke Registered Verbs row adds wire fields `role` and `confirm`. `role` is the target key's role for the HOLD-001 E-ADM-019 cross-check (not the caller's authorization role, which is resolved independently by `resolveAndVerifyCallerRole`); `confirm` is load-bearing for BC-2.05.004 PC-2 control-revocation gate. Refs: F-7A-M1.

> **v1.14 changelog note (2026-07-02):** Pass-6 adversarial correction: admin.svtn.destroy Registered Verbs row corrects wire field from svtn_id to name (impl-spec sync); admin.key.* Registered Verbs svtn_id placeholder corrected from <hex> to <svtn-name> (field carries SVTN name string, not hex ID); base64 fallback documented in decodePublicKey description. Refs: F-6A-M1/M2/M3.

> **v1.13 changelog note (2026-07-02):** Pass-4 wire-contract remediation (Burst 19 Phase 2b): renamed `pubkey_hex` → `pubkey_openssh` in all admin.key.* Registered Verbs request schemas (wire field now carries OpenSSH format string, e.g. `ssh-ed25519 AAAA... comment`); wire field name `svtn_id` confirmed throughout (no bare `svtn` field present). Formally added `admin.key.expire`, `admin.key.list-keys`, and `admin.svtn.destroy` to Registered Verbs table; retired `admin.key.role` row (read-only role-lookup verb superseded by `admin.key.list-keys`; no BC postcondition owned exclusively by `admin.key.role`). Cross-package integration test added code-side (no spec change needed). Refs: Burst 19 Phase 2b adjudicated remediation shape.

> **v1.11 changelog note (2026-07-01):** F-P5L3R-04 (Pass-6 L3): 6-site sweep — `paths.list`, `router.metrics`, `router.status` Registered Verbs row BC pins advanced v1.11 → v1.13; path-list-response intro BC pin advanced v1.11 → v1.13; pending-response intro BC pin advanced v1.11 → v1.13; `sbctl router status` CLI comment BC pin advanced v1.11 → v1.13. No structural or semantic change; BC-2.06.003 is now at v1.13.

> **v1.10 changelog note (2026-07-01):** F-L3-002 (Pass-3 L3): `router.metrics` Registered Verbs row — story trace column updated to include both `S-5.02` and `S-W5.04`; BC pin bumped to `BC-2.06.003 v1.11 PC-2`. F-L3-003 (Pass-3 L3): `router.status` Registered Verbs row — story trace column updated to include both `S-5.02` and `S-W5.04`; BC pin confirmed at `BC-2.06.003 v1.11 PC-3`. `paths.list` BC pin also bumped to v1.11. `sbctl router status` CLI comment BC pin bumped to v1.11.

> **v1.9 changelog note (2026-07-01):** Wave-6 Tranche A Ruling-4 pin-sweep: BC-2.06.003 pins updated v1.7→v1.10 at four locations (§Path list response intro, pending-response intro, `sbctl router status` CLI comment, `paths.list`/`router.status` Registered Verbs rows). Added `failed`-reserved note to paths.list row and path-list-response section. F-P2L3-001 admin.key.* re-anchoring: `admin.key.register` owning BC changed `BC-2.07.001` → `BC-2.05.004 PC-1`; `admin.key.revoke` changed → `BC-2.05.004 PC-2`; `admin.key.role` changed → `BC-2.05.004 PC-1`. `paths.list` row also updated to add S-W5.04 story trace.

> **v1.8 changelog note (2026-07-01):** F-P1L3-003: `## Daemon RPC Surface` section expanded from prose-only into an enumerated verb table listing `paths.list`, `router.metrics`, `router.status`, `admin.key.register`, `admin.key.revoke`, `admin.key.role`, and `admin.svtn.create` — each row specifies owning BC/PC, authority requirement, request/response shape, and story trace. Authority note and error-code summary added.

> **v1.7 changelog note (2026-06-30):** S-5.02 Pass-8 F-P8L3-002 sibling-propagation sweep: updated all live-content BC-2.06.003 version pins from v1.5 to v1.7 (line-80 `sbctl router status` comment, §Path list response intro, §pending-response JSON example intro). Also updated S-5.02 story pin on line-80 from v1.5 to v1.9. BC v1.7 introduced no behavioral change vs v1.5 for the cited clauses (PC-1 field schema, PC-3 alias, EC-003/EC-006 pending sentinel). Historical changelog rows citing v1.5 are preserved unchanged.

> **v1.6 changelog note (2026-06-30):** S-6.06 Pass-4 ruling F-L2-007: `sbctl admin key expire` row updated — added wire-field translation note (`--at <RFC3339>` → `after` Go duration string), noted server-side duration validation (positive, ≤100 years), added E-CFG-001 to exit codes.

> **v1.5 changelog note (2026-06-30):** F-T3-004: propagate BC-2.06.003 v1.5 version pins. Line-80 `sbctl router status` comment updated v1.2→v1.5 (now cites S-5.02 v1.5 AC-003/AC-008). §Path list response description updated v1.2→v1.5. Added pending-response JSON example showing `"rtt_p99_ms": "pending"` and note that alias emits `"quality": "pending"` when p99 pending (BC-2.06.003 v1.5 EC-003/EC-006 + PC-3).

> **v1.4 changelog note:** F-T5: `sbctl router metrics --svtn=<id>` — removed surrounding brackets to mark `--svtn=<id>` as required (not optional). BC-2.06.003 PC-2 presents this flag as required and defines no semantics for the omitted-flag case; this supplement now matches that intent.

> **v1.3 changelog note:** S-5.02 lens-3 drift sync — F-1 (HIGH): `sbctl router status` flag corrected from `[--svtn=<id>]` to `--target <router>` (BC-2.06.003 v1.2 PC-3; S-5.02 v1.3 AC-003). F-2 (MEDIUM): `last_probe_at` removed from path list JSON example (field not defined in BC-2.06.003 PC-1). F-3 (MEDIUM): `paths list` description extended to name `rtt_p99_ms` explicitly; JSON schema note added documenting `"pending"` string variant per BC-2.06.003 PC-1 / EC-003 and S-5.02 AC-004. All changes bring this supplement into alignment with BC-2.06.003 v1.2 as canonical authority.

> **v1.2 changelog note:** F-P2-004: `sbctl svtn create` marked `[DEPRECATED]` — superseded by `sbctl admin svtn create` (S-6.07; Wave 6). Retained as alias until vMINOR+1 deprecation cycle completes.

> **v1.1 changelog note:** `register-key` → `admin key register`; `revoke-key` → `admin key revoke`; `expire` subcommand added (was absent). `--key-fingerprint <fp>` replaced by `--key <hex-pubkey>` throughout. `sbctl admin svtn create --name=<svtn-name>` and `sbctl admin svtn destroy --name=<svtn-name>` added (S-6.07 Wave 6). Per Task 8 reconverge (S-6.02 lens1 F-003, interface-definitions.md CLI spec stale vs implementation).

## Exit Code Semantics

| Code | Meaning | When |
|------|---------|------|
| 0 | Success | Command completed normally; `--help` / `-h` |
| 1 | Operational error | Admission denied, session not found, daemon unreachable, config error |
| 2 | Usage error | No subcommand supplied (bare `sbctl`); invalid subcommand; missing required flags; type constraint violation; E-CFG-001 (client-side flag-value validation: zero or negative `--after` duration on `admin key expire`, `cmd/sbctl/admin.go`); E-CFG-012; E-CFG-013 (verified against merged main.go/admin.go post-#65) |
| 3 | Internal error | Unexpected panic or unrecoverable internal state (should be rare) |

## JSON Output Schema

JSON output is produced when `--json` flag is present on any sbctl command. All JSON responses share the same envelope shape (`ok`, `error`, `data` — no top-level schema field is emitted):

```json
{
  "ok": true,
  "error": null,
  "data": { ... }
}
```

On error:
```json
{
  "ok": false,
  "error": {
    "code": "E-NET-001",
    "message": "daemon unreachable: localhost:9090: connection refused",
    "field": null
  },
  "data": null
}
```

> **`error.code` closed set (v1.29, cross-ref error-taxonomy.md v4.7):** The `error.code` field carries a value from the closed set of defined error codes in `error-taxonomy.md` v4.7 RPC section. For sbctl top-level errors the relevant codes are: `E-NET-001` (unreachable before connection), `E-ADM-010` (authentication failure), `E-CFG-010` (key load failure), `E-RPC-001` (post-auth dispatch failure — CLI-level bucket; see Ruling-13), `E-RPC-002` (transport-decode failure: oversized message from server; client-side emission at `cmd/sbctl/client.go`). For daemon in-band response codes surfaced via `E-RPC-001` message prefix, the inner codes are `E-RPC-010` (unknown command), `E-RPC-011` (handler error), and handler-specific codes from ADM/SVTN/CFG/INT families. For `router.metrics` and `router.status` in-band handler responses specifically: `E-RPC-002` (args decode failure, `metrics.ErrDecodeArgs` via E-RPC-011 wrap) and `E-RPC-003` (invalid params after decode, `metrics.ErrInvalidParams` via E-RPC-011 wrap). Do NOT extend the `error.code` set without a corresponding row in `error-taxonomy.md`. See `error-taxonomy.md` v4.7 for the full authoritative catalog.

### Session list response (`sbctl sessions list --json`)

```json
{
  "ok": true,
  "data": {
    "sessions": [
      {
        "name": "agent-01",
        "node_addr": "a1b2c3d4e5f60708",
        "svtn_id": "0102030405060708090a0b0c0d0e0f10",
        "attached": true,
        "quality": "green",
        "access_mode": "full"
      }
    ]
  }
}
```

### Path list response (`sbctl paths list --json`)

Fields per BC-2.06.003 v1.13 PC-1. `rtt_p99_ms` is a float64 when ≥10 RTT samples have been collected; it is the string `"pending"` when fewer than 10 samples exist (EC-003). `last_probe_at` is NOT part of the schema (removed per BC-2.06.003 PC-1; was never defined in the canonical BC). Note: `status` is `active` or `degraded`; `failed` is reserved for `S-BL.PATH-FAILED-STATUS` (Wave-7) and MUST NOT appear in Wave-6 responses.

Example — normal response (≥10 samples):

```json
{
  "ok": true,
  "data": {
    "paths": [
      {
        "path_id": "path-001",
        "router_addr": "192.168.1.1:9090",
        "rtt_ms": 15,
        "rtt_p99_ms": 22,
        "loss_pct": 0.1,
        "status": "active"
      }
    ]
  }
}
```

Example — pending response (<10 samples; BC-2.06.003 v1.13 EC-003/EC-006). The alias `sbctl router status` also emits `"quality": "pending"` in this case (PC-3):

```json
{
  "ok": true,
  "data": {
    "paths": [
      {
        "path_id": "path-001",
        "router_addr": "192.168.1.1:9090",
        "rtt_ms": 8,
        "rtt_p99_ms": "pending",
        "loss_pct": 0.0,
        "status": "active"
      }
    ]
  }
}
```

### Router metrics response (`sbctl router metrics --json`)

```json
{
  "ok": true,
  "data": {
    "frame_count": 1234567,
    "hmac_fail_count": 3,
    "drop_cache_hits": 12,
    "path_distribution": {
      "path-001": 900000,
      "path-002": 334567
    }
  }
}
```

## Config File Schema

All daemons use YAML configuration. Config file path defaults:
- `/etc/switchboard/<mode>.yaml` (system install)
- `./switchboard-<mode>.yaml` (local run)
- Override: `--config=<path>`

### Router config (`switchboard-router.yaml`)

```yaml
# Required
listen_addr: "0.0.0.0:9090"       # Address to listen for node connections
svtn_id: "..."                     # 16-byte hex SVTN identifier (or auto-generate)
key_file: "~/.ssh/switchboard_router"  # Router keypair for admission

# Optional - determines E vs PE mode
upstream_routers: []               # Empty = E router; entries = PE router
  # - addr: "10.0.1.1:9090"
  # - addr: "10.0.1.2:9090"

# Optional tuning
keepalive_interval: "1s"
drain_timeout: "10s"
drop_cache_size: 10000
log_level: "info"
log_format: "text"
```

### Access node config (`switchboard-access.yaml`)

```yaml
# Required
router_addrs:
  - "192.168.1.1:9090"
key_file: "~/.ssh/switchboard_access"
svtn_id: "..."

# Optional
admission_key_file: ""             # Path to the node's persistent Ed25519 admission keypair
                                   # (PKCS#8 PEM, "PRIVATE KEY" block type). When absent or
                                   # empty string, the daemon uses the default path:
                                   #   /var/lib/switchboard/access-admission-identity.pem
                                   # (applied at daemon startup by runAccess, not by Config.Validate).
                                   # When present, must be a non-whitespace path; tilde (~) is
                                   # expanded per loadEd25519Key conventions (BC-2.07.003 EC-007).
                                   # On first run the keypair is generated atomically (mode 0600);
                                   # on subsequent runs the existing key is loaded. Daemon fails
                                   # closed if the file is corrupt or not an Ed25519 key.
                                   # Validation: E-CFG-014 — non-whitespace when present; no file
                                   # I/O in Validate (ARCH-06 §Config purity contract).
                                   # Access-mode only — ignored in router/console/control modes.
                                   # Cross-ref: BC-2.09.004 PC-1/PC-2, BC-2.09.003 PC-12.
tmux_socket: ""                    # Empty = default tmux socket
pty_fallback: true                 # Allow PTY fallback if tmux unavailable
tick_interval_upstream: "10ms"
tick_interval_downstream: "50ms"
log_level: "info"
```

### Console config (`switchboard-console.yaml`)

```yaml
# Required
router_addrs:
  - "192.168.1.1:9090"
key_file: "~/.ssh/switchboard_console"
svtn_id: "..."

# Optional
mgmt_listen_addr: "127.0.0.1:9091"  # Port for sbctl console control
log_level: "info"
```

### Control node config (`switchboard-control.yaml`)

```yaml
# Required
router_addrs:
  - "192.168.1.1:9090"
key_file: "~/.ssh/switchboard_control"
svtn_id: "..."

log_level: "info"
```

## Flag Interactions

| Flag A | Flag B | Interaction | Resolution |
|--------|--------|-------------|------------|
| `--json` | `--log-format=text` | No conflict; --json affects sbctl output, --log-format affects daemon log output | Both apply independently |
| `--target=<addr>` | config `daemon.address` | `--target` overrides config value; default is `/run/switchboard-router.sock` (cmd/sbctl/main.go); when `--target` is absent and the default socket is absent, sbctl exits E-NET-001 (exit 1) | `--target` wins |
| `--key=<path>` | SSH agent | If --key specified, file key used; SSH agent ignored | `--key` wins |
| `upstream_routers: []` | `upstream_routers: [...]` | Empty list = E router; any entry = PE router | Presence of entries determines mode |
| `--log-level=debug` | `log_level: info` in config | CLI flag overrides config | `--log-level` flag wins |

## Daemon RPC Surface

Daemons expose a local management API that sbctl connects to. RPC protocol is JSON-over-Unix-socket per ADR-006 (see `.factory/specs/architecture/ARCH-05-cli-and-api.md`). TCP fallback engaged when `--target=host:port` is specified.

All RPC endpoints require authentication: the caller presents its OpenSSH key signature (same mechanism as SVTN admission). Unauthenticated calls are rejected before processing.

### Registered Verbs

| Verb | Owning BC / PC | Authority Required | Request Args | Response Data | Story Trace |
|------|----------------|--------------------|--------------|---------------|-------------|
| `paths.list` | BC-2.06.003 v1.13 PC-1 | Any admitted key (bootstrap or control-role) | `{"svtn_id": "<hex>"}` (optional; omit for all paths) | `{"paths": [{path_id, router_addr, rtt_ms, rtt_p99_ms, loss_pct, status}]}` (`rtt_p99_ms` is float64 or `"pending"` per EC-003; `status` is `active` or `degraded`; `failed` reserved for Wave-7 S-BL.PATH-FAILED-STATUS) | S-5.02, S-W5.04 |
| `router.metrics` | BC-2.06.003 v1.13 PC-2 | Any admitted key | `{"svtn_id": "<hex>"}` (required) | `{"frame_count", "hmac_fail_count", "drop_cache_hits", "path_distribution"}` | S-5.02, S-W5.04 |
| `router.status` | BC-2.06.003 v1.13 PC-3 | Any admitted key | `{"target": "<router-addr>"}` | Alias for `paths.list` response + `quality` field; `"quality": "pending"` when p99 not yet available (EC-006) | S-5.02, S-W5.04 |
| `admin.key.register` | BC-2.05.004 PC-1 | Control-role key + `--confirm` token (or operator-set bootstrap grant for first-register into fresh SVTN per BC-2.05.004 EC-005) | `{"svtn_id": "<svtn-name>", "pubkey_openssh": "<OpenSSH-format string, e.g. ssh-ed25519 AAAA... comment>", "role": "control\|console\|access"}` | `{"key_fingerprint": "SHA256:<base64>", "timestamp": "<RFC3339>"}` | S-6.06 |
| `admin.key.revoke` | BC-2.05.004 PC-2 | Control-role key + `--confirm` token; console-role keys may not revoke control-role keys (ADR-004 Inv-3) | `{"svtn_id": "<svtn-name>", "pubkey_openssh": "<OpenSSH-format string>", "role": "<role>", "confirm": <bool>}` — `role`: the target key's role (e.g. `"control"`, `"console"`, `"access"`); passed to RevokeKey for the HOLD-001 E-ADM-019 cross-check that validates the caller's registered role matches the claimed role for the key being revoked; NOT the caller's authorization role (which is resolved independently by `resolveAndVerifyCallerRole` from the authenticated pubkey); `confirm`: boolean, required `true` when revoking a control-role key (BC-2.05.004 PC-2 control-revocation gate); `false` or absent is equivalent to `false` | `{"key_fingerprint": "SHA256:<base64>", "timestamp": "<RFC3339>"}` | S-6.06 |
| `admin.key.expire` | BC-2.05.004 PC-3 | Control-role key; no `--confirm` required (non-destructive scheduling) | `{"svtn_id": "<svtn-name>", "pubkey_openssh": "<OpenSSH-format string>", "after": "<Go duration string>"}` | `{"key_fingerprint": "SHA256:<base64>", "timestamp": "<RFC3339>"}` | S-6.06 |
| `admin.key.list-keys` | BC-2.05.004 Precondition 1 | Any admitted role | `{"svtn_id": "<svtn-name>"}` | `{"keys": [{fingerprint, role, expiry}]}` | S-6.06 |
| `admin.svtn.create` | BC-2.07.001 PC-1, Inv-3 | Bootstrap-only: authenticated caller MUST be the daemon bootstrap key with `RoleControl`; cross-SVTN control-role keys are not authorized | `{"name": "<svtn-name>"}` | `{"svtn_id": "<hex>", "bootstrap_fingerprint": "SHA256:<base64>"}` | S-6.07 |
| `admin.svtn.destroy` | BC-2.07.001 PC-3 | Control-role key via `resolveAndVerifyCallerRole` gate (general control-role, NOT bootstrap-only) + `--confirm` token | `{"name": "<svtn-name>"}` | `{"status": "destroyed"}` | S-6.05 |
| `admin.svtn.status` | BC-2.07.001 v1.14 PC-4 | Any admitted role (control, console, or access) in the target SVTN, OR operator-set member, OR bootstrap key — `resolveCallerAdmissionAnyRole` (list-keys precedent, F-L2-003); AUTHORITY gate bypassed, ADMISSION gate retained | `{"name": "<svtn-name>"}` | `{"svtn_id": "<hex>", "name": "<svtn-name>", "created_at": "<RFC3339>", "key_counts": {"control": <n>, "console": <n>, "access": <n>}}` — deliberately excludes session/health data (ARCH-09 purity boundary; `internal/session` is a forbidden import for `cmd/switchboard/admin_handlers.go`) | S-BL.CLI-SURFACE-COMPLETION |
| `paths.ping` | BC-2.06.004 PC-1 | Tier-1 operator-key auth only (same bar as `paths.list`/`router.metrics`/`router.status`; no additional Tier-2 role gate) | `{}` (empty — the daemon being dialed via `--router=<addr>`/`--target` IS the probe target by construction) | `{"pong": true}` (RTT is measured and reported client-side by `cmd/sbctl`, not on the wire) | S-BL.CLI-SURFACE-COMPLETION |
| `router.reload` | BC-2.09.001 v1.2 PC-1 (governance-only addendum) | Tier-1 operator-key auth only; router mode has no `SVTNManager`/`RoleControl` concept, so no stricter Tier-2 gate is available to reuse | `{}` | `{"accepted": true}` (fire-and-forget, matching raw-signal UX parity; adds E-CFG-004 if the daemon was started without `--config` — reload not applicable) | S-BL.CLI-SURFACE-COMPLETION |
| `router.drain` | BC-2.09.002 v1.3 Trigger/PC-1 (governance-only addendum) | Tier-1 operator-key auth only; same reasoning as `router.reload` | `{}` | `{"accepted": true}` (fire-and-forget; connection is expected to be severed as the daemon exits — best-effort, consistent with PC-3's no-wire-ACK framing) | S-BL.CLI-SURFACE-COMPLETION |

> **Authority note:** "bootstrap-only" verbs (`admin.svtn.create`) require that the authenticated caller's public key matches the daemon bootstrap key AND that the key's role is `RoleControl`. Regular cross-SVTN control-role keys are explicitly rejected (BC-2.07.001 Inv-3 / S-6.07 AC-003). "Control-role" verbs require any key with `RoleControl` in the target SVTN's `AdmittedKeySet`.

> **Error codes:** Insufficient authority → `E-ADM-009`. Duplicate key registration is last-write-wins role update (no error; ADR-003). Role mismatch on revoke (claimed role ≠ stored role, HOLD-001 hybrid) → `E-ADM-019`. Control-to-control revocation without confirmation → `E-ADM-018`. Key not found → `E-ADM-013`. Permission denied on SVTN destroy (non-control caller) → `E-ADM-011`. SVTN already exists → `E-SVTN-001`. Unregistered command → `E-RPC-010` (server-side, in-band). Handler error → `E-RPC-011` (server-side, in-band). See `error-taxonomy.md` for full table.

## Library Exports (Go Package Boundaries)

The following internal packages are the primary code boundaries (paths are indicative; exact structure is architecture-scoped):

| Package | Description | Primary Consumer |
|---------|-------------|-----------------|
| `internal/frame` | Frame encoding/decoding, outer header, channel header | router, access, console |
| `internal/hmac` | HMAC computation and verification | router, all nodes |
| `internal/admission` | Tier 1 challenge/response, admitted key set | router |
| `internal/session` | Tier 2 session authorization, session lifecycle | access node |
| `internal/halfchannel` | Timeslice clock, upstream/downstream half-channel state machines | access node, console |
| `internal/paths` | Path ranking, RTT/loss metrics, keep-alive | all nodes |
| `internal/multipath` | Duplicate-and-race dispatch, receiver deduplication | access node, console |
| `internal/discovery` | Presence advertisement, session enumeration | access node, console |
| `internal/config` | Config file parsing, validation, reload | all daemons |
| `internal/metrics` | Quality indicator computation, path metrics storage | all nodes |
