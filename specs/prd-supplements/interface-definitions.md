---
artifact_id: interface-definitions
document_type: prd-supplement-interface-definitions
level: L3
version: "1.0"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
inputs:
  - '.factory/specs/prd.md'
  - '.factory/specs/domain-spec/capabilities.md'
  - '_bmad-output/planning-artifacts/prd.md'
input-hash: "[md5-pending]"
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
sbctl svtn create [--name=<name>]               # Create a new SVTN
sbctl svtn destroy --id=<svtn_id>               # Destroy an SVTN
sbctl svtn list                                  # List known SVTNs
sbctl svtn status --id=<svtn_id>                # SVTN status and admitted node count

# Key Management
sbctl svtn keys register --key=<pubkey_path> --role=<control|console|access> [--svtn=<id>]
sbctl svtn keys revoke --key=<pubkey_path> [--svtn=<id>]
sbctl svtn keys list [--svtn=<id>]
sbctl svtn keys expire --key=<pubkey_path> --at=<timestamp> [--svtn=<id>]

# Session Operations
sbctl sessions list [--svtn=<id>]               # List all SVTN sessions
sbctl sessions attach <session-name> [--svtn=<id>]
sbctl sessions detach [--session=<name>] [--svtn=<id>]
sbctl sessions status [--session=<name>]        # Quality indicator + path metrics

# Path / Quality Metrics
sbctl paths list [--svtn=<id>]                  # Per-path RTT, loss, status
sbctl paths ping --router=<addr>                # One-shot RTT probe

# Router Management
sbctl router status [--svtn=<id>]               # Router health and forwarding stats
sbctl router metrics [--svtn=<id>]              # Frame counts, HMAC failures, drop cache hits
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

Operator-only subcommand requiring `--confirm` token. Used for SVTN key management and emergency recovery.

| Subcommand | Purpose | Auth | Exit codes |
|-----------|---------|------|-----------|
| `sbctl admin register-key --svtn <id> --pubkey <path> --role <control\|console\|access>` | Register a new admission key | Requires existing control-role key + interactive `--confirm` token | 0=ok, E-ADM-012 (already registered) |
| `sbctl admin revoke-key --svtn <id> --key-fingerprint <fp>` | Revoke admission key | Requires existing control-role key + `--confirm`; per ADR-004 console cannot revoke control | 0=ok, E-ADM-011 (hierarchy violation), E-ADM-013 (not found) |
| `sbctl admin recover --svtn <id> --bootstrap-key <path> --confirm <svtn-short-id>\|--yes` | Emergency recovery when all control keys are lost | Requires bootstrap key (set at SVTN creation per BC-2.07.001) + interactive `--confirm` token | 0=ok, E-ADM-014 (bootstrap mismatch) |
| `sbctl admin list-keys --svtn <id>` | List admission keys | Any admitted role | 0=ok |

**`--confirm=<svtn-short-id>`** — Required on all destructive admin operations (register-key, revoke-key, recover). Accepts the SVTN short ID (the first 8 hex characters of the SVTN ID, formatted as `SVTN-<short-id>`) as the confirmation token. Prevents accidental mass-revocation by requiring the operator to name the target SVTN explicitly. When the flag is omitted, the command enters interactive mode and prompts `Type SVTN-<short-id> to confirm:` before proceeding. Per ADR-004 split-brain mitigation.

**`--yes`** — Bypasses the `--confirm` interactive prompt for scripted use. Emits a warning to stderr: `"WARNING: --yes bypasses confirmation; ensure correct --svtn target before scripting"`. Cannot be combined with `--confirm` (usage error, exit 2).

Confirmation flow summary: interactive commands prompt for `Type SVTN-<short-id> to confirm:` when `--confirm` is not supplied on the command line. Providing `--confirm=<svtn-short-id>` satisfies the check non-interactively. `--yes` bypasses the check entirely with a stderr warning (E-CFG-006).

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
        "status": "active",
        "last_probe_at": "2026-06-23T10:00:00Z"
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
