---
artifact_id: interface-definitions
document_type: prd-supplement-interface-definitions
level: L3
version: "1.18"
status: draft
producer: product-owner
timestamp: 2026-07-02T12:00:00
phase: 1a
inputs:
  - '.factory/specs/prd.md'
  - '.factory/specs/domain-spec/capabilities.md'
  - '_bmad-output/planning-artifacts/prd.md'
input-hash: "bc4367a"
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
sbctl [--target=<addr>] [--key=<path>] [--json] <subcommand>

Global flags:
  --target=<addr>   Daemon address (host:port or unix socket path)
  --key=<path>      Path to operator private key file (default: ~/.ssh/id_ed25519)
  --json            Machine-readable JSON output
  --timeout=<dur>   Connection timeout (default: 5s)

Subcommands:

# SVTN Management
sbctl svtn create [--name=<name>]               # [REMOVED] Alias removed as of Phase 5 Pass 3 Path B remediation (PR #62). Was deprecated in v1.2 (S-6.07). Migration target: 'sbctl admin svtn create'. Invoking today returns exit 2 (unknown subcommand: svtn).
sbctl svtn destroy --id=<svtn_id>               # Destroy an SVTN
sbctl svtn list                                  # List known SVTNs
sbctl svtn status --id=<svtn_id>                # SVTN status and admitted node count

# Key Management (read-only; destructive ops exclusively via sbctl admin)
sbctl svtn keys list [--svtn=<id>]
# NOTE: Destructive key operations (register/revoke/expire) are exclusively via
# `sbctl admin` (operator-only with --confirm gating). See `sbctl admin` table below.

# Session Operations
sbctl sessions list [--svtn=<id>]               # List all SVTN sessions
sbctl sessions attach <session-name> [--svtn=<id>]
sbctl sessions detach [--session=<name>] [--svtn=<id>]
sbctl sessions status [--session=<name>]        # Quality indicator + path metrics

# Path / Quality Metrics
sbctl paths list [--svtn=<id>]                  # Per-path RTT (rtt_ms, rtt_p99_ms), loss, status
sbctl paths ping --router=<addr>                # One-shot RTT probe

# Router Management
sbctl router status --target <router>            # Alias for sbctl paths list + quality column (BC-2.06.003 v1.13 PC-3; S-5.02 v1.9 AC-003/AC-008)
sbctl router metrics --svtn=<id>                # Frame counts, HMAC failures, drop cache hits
sbctl router reload                             # Reload config (SIGHUP equivalent)
sbctl router drain                              # Graceful drain (SIGTERM equivalent)

# Console Control (remote)
sbctl console attach --session=<name> --console=<addr> [--svtn=<id>]
sbctl console detach --console=<addr>
sbctl console switch --session=<name> --console=<addr>

# Diagnostics
sbctl version                                   # Print daemon version
sbctl ping                                      # Connectivity check to daemon
```

### `sbctl admin`

Operator-only subcommand requiring `--confirm` token. Used for SVTN lifecycle management, key management, and emergency recovery.

#### Key management (`sbctl admin key`)

Nested form — all destructive key operations use `sbctl admin key <verb>`:

| Subcommand | Purpose | Auth | Exit codes |
|-----------|---------|------|-----------|
| `sbctl admin key register --svtn <id> --key <hex-pubkey> --role <control\|console\|access>` | Register a new admission key | Requires existing control-role key + interactive `--confirm` token | 0=ok, E-ADM-012 (already registered), E-ADM-018 (control-to-control confirmation required) |
| `sbctl admin key revoke --svtn <id> --key <hex-pubkey>` | Revoke admission key | Requires existing control-role key + `--confirm`; per ADR-004 console cannot revoke control | 0=ok, E-ADM-011 (hierarchy violation), E-ADM-013 (not found), E-ADM-018 (control-to-control confirmation required) |
| `sbctl admin key expire --svtn <id> --key <hex-pubkey> --at <RFC3339-timestamp>` | Set automatic expiry on an admission key. CLI translates `--at <RFC3339-timestamp>` to a Go duration string (`after` wire field) before sending: `after = timestamp - time.Now()`. Server validates `after` is positive and ≤100 years. | Requires existing control-role key; no `--confirm` required (non-destructive scheduling) | 0=ok, E-ADM-013 (key not found), E-CFG-001 (invalid `after` duration: zero, negative, or >100 years) |
| `sbctl admin list-keys --svtn <id>` | List all admission keys with role, fingerprint, expiry | Any admitted role | 0=ok |

`--key <openssh-pubkey>` — Replaces the former `--key-fingerprint <fp>` flag. Accepts an OpenSSH-format public key string (e.g. `ssh-ed25519 AAAA... comment`). The CLI marshals this as the `pubkey_openssh` wire field sent to `internal/svtnmgmt`. Previously accepted a raw hex-encoded public key (`pubkey_hex`); `pubkey_openssh` is the canonical wire field name as of interface-definitions v1.13. The daemon-side `decodePublicKey()` accepts both OpenSSH format (primary path, via `ssh.ParseAuthorizedKey`) and raw base64-encoded 32-byte Ed25519 key material (fallback, for backward compatibility with clients that have not yet migrated to OpenSSH format). New clients MUST send OpenSSH format; the base64 fallback is deprecated and may be removed in a future version.

#### SVTN lifecycle (`sbctl admin svtn`)

| Subcommand | Purpose | Auth | Exit codes |
|-----------|---------|------|-----------|
| `sbctl admin svtn create --name=<svtn-name>` | Create a new SVTN; returns `svtn_id` and bootstrap fingerprint | Bootstrap-only: caller MUST authenticate with the daemon bootstrap key (RoleControl); cross-SVTN control-role keys are not accepted. See §380 and BC-2.07.001 Inv-3. | 0=ok, E-SVTN-001 (already exists), E-ADM-009 (insufficient authority: bootstrap key required), E-CFG-001 (invalid SVTN name: empty / whitespace-only / >255 bytes / invalid UTF-8 / control chars), E-INT-001 (internal error wrap on non-duplicate Create failure) |
| `sbctl admin svtn destroy --name=<svtn-name>` | Destroy an SVTN and all admitted keys; terminates active sessions | Requires control-role key + `--confirm` | 0=ok, E-ADM-011 (unauthorized), E-ADM-009 (insufficient role), E-CFG-001 (invalid SVTN name: empty / whitespace-only / >255 bytes / invalid UTF-8 / control chars) |

#### Emergency recovery

> **PENDING-S-BL.ADMIN-RECOVER-WIRE:** Spec-complete; implementation deferred to backlog story `S-BL.ADMIN-RECOVER-WIRE`. Neither `cmd/sbctl` nor the daemon dispatch `admin recover` today — the `runAdmin` switch covers only `key | list-keys | svtn`; the default arm returns `admin: unknown subcommand %q`. Operators invoking `sbctl admin recover` on current builds receive exit 2 (unknown-subcommand).

| Subcommand | Purpose | Auth | Exit codes |
|-----------|---------|------|-----------|
| `sbctl admin recover --svtn <id> --bootstrap-key <path> --confirm <svtn-short-id>\|--yes` | Emergency recovery when all control keys are lost | Requires bootstrap key (set at SVTN creation per BC-2.07.001) + interactive `--confirm` token | 0=ok, E-ADM-014 (bootstrap mismatch) |
> **PENDING-S-BL.ADMIN-RECOVER-WIRE:** Row above is spec-complete; CLI and daemon dispatch not yet implemented. See annotation above.

**`--confirm=<svtn-short-id>`** — Required on all destructive admin operations (key register, key revoke, svtn destroy, recover). Accepts the SVTN short ID (the first 8 hex characters of the SVTN ID, formatted as `SVTN-<short-id>`) as the confirmation token. Prevents accidental mass-revocation by requiring the operator to name the target SVTN explicitly. When the flag is omitted, the command enters interactive mode and prompts `Type SVTN-<short-id> to confirm:` before proceeding. Per ADR-004 split-brain mitigation.
> **Interim rendering (DRIFT-P5P4-PROMPT-SHORTID):** Until the CLI can resolve the actual SVTN short-id from the daemon response, the prompt MAY render as a static-example form (e.g. `"Type the SVTN short-ID (e.g. SVTN-abcd1234) to confirm: "`). Both forms satisfy the confirmation gate; the substitution form is the canonical long-term target.

**`--yes`** — Bypasses the `--confirm` interactive prompt for scripted use. Emits a warning to stderr: `"WARNING: --yes bypasses confirmation; ensure correct --name target before scripting"`. Cannot be combined with `--confirm` (usage error, exit 2).

Confirmation flow summary: interactive commands prompt for `Type SVTN-<short-id> to confirm:` when `--confirm` is not supplied on the command line. Providing `--confirm=<svtn-short-id>` satisfies the check non-interactively. `--yes` bypasses the check entirely with a stderr warning. Combining `--yes` with `--confirm` is a usage error (E-CFG-012, exit 2). In a non-interactive session (no TTY) where neither `--confirm` nor `--yes` is supplied, the command exits with E-CFG-013 (exit 2) — use `--confirm=<svtn-short-id>` or `--yes` for scripted invocations.
> **Interim rendering (DRIFT-P5P4-PROMPT-SHORTID):** Until the CLI can resolve the actual SVTN short-id from the daemon response, the prompt MAY render as a static-example form (e.g. `"Type the SVTN short-ID (e.g. SVTN-abcd1234) to confirm: "`). Both forms satisfy the confirmation gate; the substitution form is the canonical long-term target.

> **v1.18 changelog note (2026-07-03):** Phase 5 Pass 5 adversarial remediation (Burst 21). F-P5P5-A-001: §116 authority cell corrected from "Requires control-role key on control-mode daemon" to bootstrap-only language aligned with impl (`admin_handlers.go:686-705`), Registered Verbs §380, and BC-2.07.001 Inv-3/Ruling-5. F-P5P5-A-002: §119-123 Emergency recovery section annotated with `PENDING-S-BL.ADMIN-RECOVER-WIRE` (spec-complete, implementation deferred; operators get exit 2 unknown-subcommand today); §125 confirm-gate mention carries sibling annotation. F-P5P5-A-003: §116 exit-codes column extended with E-CFG-001 (five `validateSVTNName` validation arms) and E-INT-001 (non-duplicate Create failure wrap); §117 exit-codes column extended with E-CFG-001 for caller-supplied-name validation path. F-P5P5-A-004: §59 `sbctl svtn create` alias status changed from DEPRECATED/retained to REMOVED as of Phase 5 Pass 3 Path B remediation (PR #62); migration target and exit-2 behaviour noted.

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
| 0 | Success | Command completed normally |
| 1 | Operational error | Admission denied, session not found, daemon unreachable, config error |
| 2 | Usage error | Invalid subcommand, missing required flags, type constraint violation |
| 3 | Internal error | Unexpected panic or unrecoverable internal state (should be rare) |

## JSON Output Schema

JSON output is produced when `--json` flag is present on any sbctl command. All JSON responses share a common envelope:

```json
{
  "$schema": "https://switchboard.example/schemas/v1/response.json",
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
    "svtn_id": "...",
    "frame_count": 1234567,
    "hmac_fail_count": 3,
    "drop_cache_hits": 12,
    "path_distribution": {
      "path-001": 0.52,
      "path-002": 0.48
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
| `--target=<addr>` | config `daemon.address` | `--target` overrides config value | `--target` wins |
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
| `router.metrics` | BC-2.06.003 v1.13 PC-2 | Any admitted key | `{"svtn_id": "<hex>"}` (required) | `{"svtn_id", "frame_count", "hmac_fail_count", "drop_cache_hits", "path_distribution"}` | S-5.02, S-W5.04 |
| `router.status` | BC-2.06.003 v1.13 PC-3 | Any admitted key | `{"target": "<router-addr>"}` | Alias for `paths.list` response + `quality` field; `"quality": "pending"` when p99 not yet available (EC-006) | S-5.02, S-W5.04 |
| `admin.key.register` | BC-2.05.004 PC-1 | Control-role key + `--confirm` token (or operator-set bootstrap grant for first-register into fresh SVTN per BC-2.05.004 EC-005) | `{"svtn_id": "<svtn-name>", "pubkey_openssh": "<OpenSSH-format string, e.g. ssh-ed25519 AAAA... comment>", "role": "control\|console\|access"}` | `{"ok": true}` | S-6.06 |
| `admin.key.revoke` | BC-2.05.004 PC-2 | Control-role key + `--confirm` token; console-role keys may not revoke control-role keys (ADR-004 Inv-3) | `{"svtn_id": "<svtn-name>", "pubkey_openssh": "<OpenSSH-format string>", "role": "<role>", "confirm": <bool>}` — `role`: the target key's role (e.g. `"control"`, `"console"`, `"access"`); passed to RevokeKey for the HOLD-001 E-ADM-019 cross-check that validates the caller's registered role matches the claimed role for the key being revoked; NOT the caller's authorization role (which is resolved independently by `resolveAndVerifyCallerRole` from the authenticated pubkey); `confirm`: boolean, required `true` when revoking a control-role key (BC-2.05.004 PC-2 control-revocation gate); `false` or absent is equivalent to `false` | `{"ok": true}` | S-6.06 |
| `admin.key.expire` | BC-2.05.004 PC-3 | Control-role key; no `--confirm` required (non-destructive scheduling) | `{"svtn_id": "<svtn-name>", "pubkey_openssh": "<OpenSSH-format string>", "after": "<Go duration string>"}` | `{"ok": true}` | S-6.06 |
| `admin.key.list-keys` | BC-2.05.004 Precondition 1 | Any admitted role | `{"svtn_id": "<svtn-name>"}` | `{"keys": [{fingerprint, role, expiry}]}` | S-6.06 |
| `admin.svtn.create` | BC-2.07.001 PC-1, Inv-3 | Bootstrap-only: authenticated caller MUST be the daemon bootstrap key with `RoleControl`; cross-SVTN control-role keys are not authorized | `{"name": "<svtn-name>"}` | `{"svtn_id": "<hex>", "bootstrap_fingerprint": "SHA256:<base64>"}` | S-6.07 |
| `admin.svtn.destroy` | BC-2.07.001 PC-3 | Control-role key via `resolveAndVerifyCallerRole` gate (general control-role, NOT bootstrap-only) + `--confirm` token | `{"name": "<svtn-name>"}` | `{"ok": true, "status": "destroyed"}` | S-6.05 |

> **Authority note:** "bootstrap-only" verbs (`admin.svtn.create`) require that the authenticated caller's public key matches the daemon bootstrap key AND that the key's role is `RoleControl`. Regular cross-SVTN control-role keys are explicitly rejected (BC-2.07.001 Inv-3 / S-6.07 AC-003). "Control-role" verbs require any key with `RoleControl` in the target SVTN's `AdmittedKeySet`.

> **Error codes:** Insufficient authority → `E-ADM-009`. Key already registered → `E-ADM-012`. Hierarchy violation (e.g. console revokes control) → `E-ADM-011`. Key not found → `E-ADM-013`. SVTN already exists → `E-SVTN-001`. Unregistered command → `E-RPC-010` (server-side, in-band). Handler error → `E-RPC-011` (server-side, in-band). See `error-taxonomy.md` for full table.

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
