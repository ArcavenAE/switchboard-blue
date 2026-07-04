# sbctl — Switchboard Operator CLI Reference

`sbctl` is the operator command-line interface to a running Switchboard
daemon. It manages SVTNs (Switched Virtual Networks), admission keys,
sessions, and observes router state via the daemon's management RPC.

This reference describes the CLI as it exists in **v0.1.0-rc.1**.
Verbs that are spec-defined but not yet wired are called out with a
`PENDING` marker at the end of their entry.

---

## Contents

- [Global flags](#global-flags)
- [Exit codes](#exit-codes)
- [JSON output envelope](#json-output-envelope)
- [Admission plane](#admission-plane)
  - [`sbctl admin key register`](#sbctl-admin-key-register)
  - [`sbctl admin key revoke`](#sbctl-admin-key-revoke)
  - [`sbctl admin key expire`](#sbctl-admin-key-expire)
  - [`sbctl admin list-keys`](#sbctl-admin-list-keys)
  - [`sbctl admin svtn create`](#sbctl-admin-svtn-create)
  - [`sbctl admin svtn destroy`](#sbctl-admin-svtn-destroy)
  - [`sbctl admin recover`](#sbctl-admin-recover) — PENDING
- [Path plane](#path-plane)
  - [`sbctl paths list`](#sbctl-paths-list)
  - [`sbctl router metrics`](#sbctl-router-metrics)
  - [`sbctl router status`](#sbctl-router-status)
- [Session plane](#session-plane)
  - [`sbctl sessions list`](#sbctl-sessions-list)
  - [`sbctl console attach`](#sbctl-console-attach)
  - [`sbctl console detach`](#sbctl-console-detach)
  - [`sbctl console switch`](#sbctl-console-switch)
- [Confirmation and non-interactive use](#confirmation-and-non-interactive-use)
- [Unimplemented verbs (PENDING)](#unimplemented-verbs-pending)

---

## Global flags

All `sbctl` invocations accept the following global flags:

| Flag | Default | Meaning |
|------|---------|---------|
| `--target=<addr>` | `/run/switchboard-router.sock` | Daemon management endpoint. Unix socket path or `host:port` for TCP. When the default socket is absent and no `--target` is given, sbctl exits with `E-NET-001`. |
| `--key=<path>` | — | Operator private key (OpenSSH format, Ed25519). Required for any authenticated verb. If omitted, sbctl attempts SSH agent auth. |
| `--json` | off | Emit machine-readable JSON to stdout instead of human-readable text. See [envelope schema](#json-output-envelope). |
| `--timeout=<duration>` | daemon default | Per-request timeout (Go duration, e.g. `30s`, `2m`). |

Precedence rules:

- `--target` beats config file `daemon.address`.
- `--key` beats SSH agent (agent is ignored when `--key` is present).
- `--log-level` CLI flag beats `log_level` in the daemon config file (daemon-side).

---

## Exit codes

| Code | Meaning | Typical trigger |
|------|---------|-----------------|
| `0` | Success | Command completed; also emitted by `--help` / `-h`. |
| `1` | Operational error | Admission denied, daemon unreachable, config error, SVTN-not-found, wire error. |
| `2` | Usage error | Missing/invalid flags, unknown subcommand, TTY gate failures (`E-CFG-012`, `E-CFG-013`). |
| `3` | Internal error | Unexpected panic or unrecoverable internal state (should be rare). |

Every operational and usage error carries a taxonomy code (see
[docs/errors.md](errors.md)) both in text output and in the JSON
envelope's `error.code` field.

---

## JSON output envelope

When `--json` is passed, every response is a single JSON object with
the shape:

```json
{
  "ok": true,
  "error": null,
  "data": { ... }
}
```

or, on failure:

```json
{
  "ok": false,
  "error": {
    "code": "E-ADM-009",
    "message": "insufficient authority for operation admin.key.revoke: key SHA256:... has role console",
    "field": null
  },
  "data": null
}
```

- `ok` — boolean success flag; matches exit code 0.
- `error.code` — a taxonomy code from [docs/errors.md](errors.md). Guaranteed present on `ok: false`.
- `error.message` — human-readable diagnostic. Do not parse for control flow — parse `error.code`.
- `error.field` — populated on config/usage errors that pinpoint a single field.
- `data` — the response payload; shape depends on the verb.

There is no top-level `$schema` field. Schemas are described per-verb below.

---

## Admission plane

### `sbctl admin key register`

Register a new admission key in an SVTN.

**Syntax:**
```
sbctl admin key register --svtn <svtn-name> --key <openssh-pubkey> \
    [--role <control|console|access>] \
    [--confirm=<svtn-short-id>|--yes]
```

**Args:**
- `--svtn` — target SVTN name (from `admin svtn create --name=`).
- `--key` — OpenSSH-format public key literal (e.g. `ssh-ed25519 AAAA... comment`).
- `--role` — `control`, `console`, or `access`. Defaults to `console`.
- `--confirm=<svtn-short-id>` — interactive confirmation token; matched against the target SVTN's short id.
- `--yes` — bypass confirmation for scripted use (warns to stderr).

**Auth:** Requires a control-role key. Duplicate registration is
last-write-wins on role (no error) per ADR-003.

**Exit codes:** `0`, `E-CFG-001`, `E-ADM-009`, `E-CFG-012`,
`E-CFG-013`, `E-SVTN-003`.

**Response (`data`):**
```json
{
  "key_fingerprint": "SHA256:...",
  "timestamp": "2026-07-04T12:34:56Z"
}
```

---

### `sbctl admin key revoke`

Revoke an admission key.

**Syntax:**
```
sbctl admin key revoke --svtn <svtn-name> --key <openssh-pubkey> \
    --role <control|console|access> [--confirm]
```

**Args:**
- `--svtn` — target SVTN name.
- `--key` — OpenSSH-format public key literal.
- `--role` — the **target key's** role (used for role cross-check, `E-ADM-019`). Required.
- `--confirm` — **boolean** wire flag (bare `--confirm` or `--confirm=true`). Required only when revoking a control-role key; enforced daemon-side (`E-ADM-018`).

**Auth:** Requires a control-role key. A console-role key cannot revoke a
control-role key (ADR-004 Inv-3). Revoking the bootstrap key is
forbidden (`E-ADM-020`).

**Note:** `key revoke` is **not** part of the `runDestroyConfirmGate`
family — there is no interactive prompt, no `--yes`, and neither
`E-CFG-012` nor `E-CFG-013` applies here. The `--confirm` flag on this
verb is a plain daemon-side boolean, distinct from the destroy/register
confirmation gate.

**Exit codes:** `0`, `E-ADM-009`, `E-ADM-011`, `E-ADM-013`,
`E-ADM-018`, `E-ADM-019`, `E-ADM-020`.

**Response (`data`):**
```json
{
  "key_fingerprint": "SHA256:...",
  "timestamp": "2026-07-04T12:34:56Z"
}
```

---

### `sbctl admin key expire`

Schedule automatic expiry on an admission key. Non-destructive; no
`--confirm` gate.

**Syntax:**
```
sbctl admin key expire --svtn <svtn-name> --key <openssh-pubkey> \
    --after <duration>
```

**Args:**
- `--svtn` — target SVTN name.
- `--key` — OpenSSH-format public key literal.
- `--after` — Go duration string (`24h`, `168h`, etc.). Must be positive; upper bound of 100 years is enforced daemon-side.

**Auth:** Requires a control-role key. Expiring the bootstrap key is
forbidden (`E-ADM-021`).

**Exit codes:** `0`, `E-CFG-001` (invalid `--after`), `E-ADM-009`,
`E-ADM-013`, `E-ADM-021`, `E-SVTN-003`.

---

### `sbctl admin list-keys`

List all admission keys in an SVTN.

**Syntax:**
```
sbctl admin list-keys --svtn <svtn-name>
```

**Auth:** Any admitted role in the target SVTN, or the operator-set
membership, or the daemon bootstrap key. The authority gate is bypassed
here but the admission gate still applies.

**Exit codes:** `0`, `E-CFG-001` (missing `--svtn`), `E-SVTN-003`.

**Response (`data`):**
```json
{
  "keys": [
    {"fingerprint": "SHA256:...", "role": "control", "expiry": "2027-01-01T00:00:00Z"},
    {"fingerprint": "SHA256:...", "role": "console", "expiry": null}
  ]
}
```

---

### `sbctl admin svtn create`

Create a new SVTN. The caller must authenticate with the **daemon
bootstrap key** — cross-SVTN control-role keys are not accepted here.

**Syntax:**
```
sbctl admin svtn create --name=<svtn-name>
```

**Args:**
- `--name` — SVTN name (non-empty, ≤255 bytes, valid UTF-8, no control chars).

**Auth:** Bootstrap-only (see BC-2.07.001 Inv-3).

**Exit codes:** `0`, `E-CFG-001`, `E-ADM-009`, `E-SVTN-001` (already
exists), `E-INT-001`.

**Response (`data`):**
```json
{
  "svtn_id": "a1b2c3d4e5f60102",
  "bootstrap_fingerprint": "SHA256:..."
}
```

Note the returned `svtn_id` (hex) — subsequent commands accept the
**name** (`--svtn <svtn-name>`), not this hex id. The wire format
retains the field name `svtn_id` for historical compatibility, but
daemon-side lookup is name-keyed.

---

### `sbctl admin svtn destroy`

Destroy an SVTN and all admitted keys. Terminates active sessions.

**Syntax:**
```
sbctl admin svtn destroy --name=<svtn-name> [--confirm=<svtn-short-id>|--yes]
```

**Auth:** Requires a control-role key and the confirmation gate
(`--confirm=<svtn-short-id>` interactive, or `--yes` scripted).

**Exit codes:** `0`, `E-ADM-009`, `E-ADM-011`, `E-CFG-001`,
`E-CFG-012`, `E-CFG-013`, `E-SVTN-003`.

**Response (`data`):**
```json
{"status": "destroyed"}
```

---

### `sbctl admin recover`

Emergency recovery when all control keys are lost. Uses the SVTN's
bootstrap key to re-establish control access.

**Syntax:**
```
sbctl admin recover --svtn <svtn-name> --bootstrap-key <path> \
    --confirm <svtn-short-id>|--yes
```

**Auth:** Requires the bootstrap key set at SVTN creation.

**Exit codes (spec):** `0`, `E-ADM-014`.

> **PENDING-S-BL.ADMIN-RECOVER-WIRE:** This verb is spec-complete but
> not wired in v0.1.0-rc.1. Invoking it returns exit `2` with an
> unknown-subcommand usage error until the backlog story ships.

---

## Path plane

### `sbctl paths list`

List all router paths for one SVTN or (with `--svtn` omitted) all
SVTNs the caller can see.

**Syntax:**
```
sbctl paths list [--svtn <svtn-name>]
```

**Auth:** Any admitted key (bootstrap or control-role).

**Exit codes:** `0`, `E-ADM-*`, `E-NET-001`.

**Response (`data`, `--json`):**
```json
{
  "paths": [
    {
      "path_id": "p-01",
      "router_addr": "10.0.0.1:9090",
      "rtt_ms": 4.2,
      "rtt_p99_ms": 12.7,
      "loss_pct": 0.0,
      "status": "active"
    }
  ]
}
```

- `rtt_p99_ms` is a float64 once ≥10 RTT samples have been collected; it is the string `"pending"` when fewer than 10 samples exist (EC-003 per BC-2.06.003).
- `status` is `active` or `degraded`. `failed` is reserved for a future release (`S-BL.PATH-FAILED-STATUS`) and MUST NOT appear in v0.1.0-rc.1 responses.

---

### `sbctl router metrics`

Fetch aggregate router metrics for one SVTN.

**Syntax:**
```
sbctl router metrics --svtn <svtn-name>
```

**Response (`data`):**
```json
{
  "frame_count": 12345,
  "hmac_fail_count": 0,
  "drop_cache_hits": 7,
  "path_distribution": {"p-01": 0.62, "p-02": 0.38}
}
```

---

### `sbctl router status`

Alias for `paths list` that additionally emits a `quality` field
summarizing the router's overall health.

**Syntax:**
```
sbctl router status --target <router-addr>
```

When p99 RTT is still `pending`, `quality` is emitted as the string
`"pending"` (per BC-2.06.003 PC-3, EC-006).

---

## Session plane

### `sbctl sessions list`

List tmux sessions currently published to an SVTN.

**Syntax:**
```
sbctl sessions list [--svtn <svtn-name>]
```

**Auth:** Any admitted role.

Response shape:

```json
{
  "sessions": [
    {"name": "work", "node_addr": "0102030405060708", "svtn": "team-a"}
  ]
}
```

---

### `sbctl console attach`

Attach an interactive terminal to a remote tmux session.

**Syntax:**
```
sbctl console attach --session <session-name>
```

The console dials the daemon at `--target`, authenticates with
`--key`, and switches the local terminal to raw mode until the
session detaches.

---

### `sbctl console detach`

Detach the currently attached console (interactive).

**Syntax:**
```
sbctl console detach
```

---

### `sbctl console switch`

Switch attachment to a different session without exiting the console
process.

**Syntax:**
```
sbctl console switch --session <session-name>
```

---

## Confirmation and non-interactive use

Several verbs are guarded by the `runDestroyConfirmGate` family:

- `admin svtn destroy`
- `admin key register`
- `admin recover` (PENDING)

Behaviour:

1. **Interactive TTY, no `--confirm`, no `--yes`** — sbctl prompts:
   ```
   Type SVTN-<short-id> to confirm:
   ```
   Reads a line; matches the expected SVTN short id; on match, proceeds.
2. **`--confirm=<svtn-short-id>`** — satisfies the gate non-interactively.
3. **`--yes`** — bypasses the gate; emits a stderr warning:
   ```
   WARNING: --yes bypasses confirmation; ensure correct --name target before scripting
   ```
4. **`--yes` and `--confirm=<...>` combined** — usage error, exit `2`, `E-CFG-012`.
5. **Non-TTY session, neither `--confirm` nor `--yes`** — exit `2`, `E-CFG-013`. Use `--confirm=<svtn-short-id>` or `--yes` for scripted invocations.

**`sbctl admin key revoke` is not part of this family** — see the
"Note" under [`admin key revoke`](#sbctl-admin-key-revoke). Its
`--confirm` is a plain boolean daemon-side gate for control-to-control
revocation only, and the E-CFG-012 / E-CFG-013 constraints do not
apply.

---

## Unimplemented verbs (PENDING)

The following verbs are spec-defined but not wired in **v0.1.0-rc.1**.
Invoking any of them produces exit `2` with an unknown-subcommand
usage error until the corresponding backlog story ships.

| Verb | Backlog story |
|------|--------------|
| `sbctl svtn destroy` | `S-BL.CLI-SURFACE-COMPLETION` |
| `sbctl svtn status` | `S-BL.CLI-SURFACE-COMPLETION` |
| `sbctl svtn list` | won't-fix (surface removed) |
| `sbctl paths ping` | `S-BL.CLI-SURFACE-COMPLETION` |
| `sbctl router reload` | `S-BL.CLI-SURFACE-COMPLETION` |
| `sbctl router drain` | `S-BL.CLI-SURFACE-COMPLETION` |
| `sbctl sessions attach` | `S-BL.DISCOVERY-WIRE` |
| `sbctl sessions detach` | `S-BL.DISCOVERY-WIRE` |
| `sbctl sessions status` | `S-BL.DISCOVERY-WIRE` |
| `sbctl version` | `S-BL.PING-VERSION-WIRE` |
| `sbctl ping` | `S-BL.PING-VERSION-WIRE` |
| `sbctl admin recover` | `S-BL.ADMIN-RECOVER-WIRE` |

The canonical CLI surface (implemented) is what appears in the sections
above. The `admin` verb family uses `sbctl admin key {register,revoke,expire}`
and `sbctl admin list-keys` — an earlier form (`sbctl svtn keys list`)
is superseded.

---

## Configuration files

Each daemon mode reads a YAML config file at startup. Only the
management surface is described here; see the router charter for
network-layer fields.

- **Router** — `switchboard-router.yaml` (listen addr, upstream routers, management socket).
- **Access node** — `switchboard-access.yaml` (upstream router, node key).
- **Console** — `switchboard-console.yaml` (upstream router, node key, management socket).
- **Control node** — `switchboard-control.yaml` (upstream router, control key, management socket).

Management sockets default to `/run/switchboard-router.sock` (Unix).
Console-mode daemons may bind management to a TCP loopback address
(`127.0.0.1:<port>` or `[::1]:<port>`); binding a console-mode management
listener to a non-loopback address is refused with `E-CFG-008`.

See `docs/getting-started.md` for a minimal example config.

---

## See also

- [docs/getting-started.md](getting-started.md) — quickstart walkthrough
- [docs/architecture.md](architecture.md) — Switchboard architecture overview
- [docs/errors.md](errors.md) — full error taxonomy
- [CONTRIBUTING.md](../CONTRIBUTING.md) — how to contribute
