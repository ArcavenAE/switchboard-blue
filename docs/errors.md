# Switchboard Error Reference

Every operational or usage error surfaced by a Switchboard daemon or by
`sbctl` carries a stable taxonomy code. Codes are emitted in both
human-readable output and in the JSON envelope's `error.code` field
(see [docs/sbctl.md](sbctl.md#json-output-envelope)).

Parse `error.code` for programmatic control flow. Do **not** grep the
human-readable message — those strings may be reworded across releases.

---

## Categories

| Prefix | Category | Meaning |
|--------|----------|---------|
| `E-ADM-` | Admission / Auth | Authentication, admission, authorization failures. |
| `E-CFG-` | Configuration | Config parse, validation, flag-shape errors. |
| `E-NET-` | Network | Daemon unreachable, connection refused, dial timeout. |
| `E-PRT-` | Protocol | Frame format, version, encoding errors. |
| `E-FWD-` | Forwarding | Routing, path selection, loop detection. |
| `E-SES-` | Session | Session lifecycle, attach, detach. |
| `E-SVTN-` | SVTN management | SVTN create, destroy, lookup. |
| `E-SYS-` | System | OS-level errors (PTY unavailable, fd exhaustion). |
| `E-RPC-` | Remote procedure call | Post-auth RPC dispatch failures. |
| `E-INT-` | Internal | Unexpected internal errors surfaced to the operator. |

## Severity

| Severity | Meaning | Exit-code impact |
|----------|---------|------------------|
| `broken` | Operation cannot complete; operator action required. | Non-zero exit. |
| `degraded` | Partial operation; reduced functionality; logged clearly. | Zero exit; daemon continues. |
| `cosmetic` | Display or formatting issue; no functional impact. | Zero exit. |

---

## ADM — Admission / Authorization

Errors originating from the admission tier (challenge / signature),
authority checks, or key-lifecycle operations.

| Code | Severity | Exit | Summary |
|------|----------|------|---------|
| `E-ADM-001` | broken | 1 | Admission denied: signature verification failed for the caller node address on this SVTN. |
| `E-ADM-002` | broken | dropped | HMAC verification of a data frame failed at the primitive layer. Frame is dropped; no operator response. |
| `E-ADM-003` | broken | dropped | Frame received from a non-admitted source address. Frame is dropped. |
| `E-ADM-004` | broken | 1 | Address collision — the requested node address is already admitted in the SVTN. |
| `E-ADM-005` | broken | 1 | Key revoked — the key fingerprint has been revoked in the SVTN. |
| `E-ADM-006` | broken | 1 | Session authorization denied — the console key is not authorized for this session on this node. |
| `E-ADM-007` | degraded | 0 | Upstream rejected read-only access for the console. Session continues with reduced privilege. |
| `E-ADM-008` | broken | 1 | Nonce replay — a challenge nonce was already consumed for this node address. |
| `E-ADM-009` | broken | 1 | Insufficient authority for the requested operation — caller's role is too low. |
| `E-ADM-010` | broken | 1 | Authentication failed — operator key not authorized for this daemon. Server-side wire response omits key fingerprint to prevent enumeration. |
| `E-ADM-011` | broken | 1 | Permission denied at the Go API layer — role hierarchy violation (revocation) or destroy authorization violation. Not reachable via the mgmt RPC path (`E-ADM-009` fires first). |
| `E-ADM-012` | broken | 1 | Key already registered — pubkey fingerprint already exists in the SVTN. |
| `E-ADM-013` | broken | 1 | Key not found — no key with this fingerprint is registered in the SVTN. |
| `E-ADM-014` | broken | 1 | Bootstrap key mismatch — provided key does not match the SVTN's bootstrap key (`sbctl admin recover`). |
| `E-ADM-015` | broken | 1 | Key expired — the key's `expiry` has passed. |
| `E-ADM-016` | broken | dropped | Wire HMAC verification failed at `RouteFrame` (auth key unavailable, or tag mismatch). Frame dropped. |
| `E-ADM-017` | degraded | 0 | HMAC failure rate alert — sliding-window failure count exceeded threshold for a source address. Router continues. |
| `E-ADM-018` | broken | 1 | Control-to-control revocation requires explicit confirmation. Retry with `--confirm`. |
| `E-ADM-019` | broken | 1 | Role mismatch — the claimed role in the RPC does not match the role stored for the registered key. Prevents privilege-escalation via role spoofing. |
| `E-ADM-020` | broken | 1 | Bootstrap key revoke forbidden — the SVTN's bootstrap key is a permanent trust anchor and cannot be revoked. |
| `E-ADM-021` | broken | 1 | Bootstrap key expire forbidden — the SVTN's bootstrap key cannot be expired (mirror of E-ADM-020). |

---

## CFG — Configuration

Errors from the daemon config loader or from `sbctl` flag-shape validation.

| Code | Severity | Exit | Summary |
|------|----------|------|---------|
| `E-CFG-001` | broken | 1 or 2 | Generic config field error — invalid value or missing required flag. Exit 1 for daemon-side, exit 2 for CLI-side. Message format: `"config error: <field>: <problem>. Fix: <suggestion>"`. |
| `E-CFG-002` | broken | 1 | Invalid `listen_addr` — not a valid `host:port`. |
| `E-CFG-003` | broken | 1 | Invalid upstream router address. |
| `E-CFG-004` | broken | 1 | Config file not found at the given path. |
| `E-CFG-005` | broken | 1 | YAML parse error — malformed config file. |
| `E-CFG-006` | broken | 1 | Invalid `drain_timeout` — must not be negative. |
| `E-CFG-008` | broken | 1 | Management socket error — empty `management_socket`, or console-mode TCP bind to a non-loopback address. |
| `E-CFG-009` | broken | 1 | Invalid `authorized_operator_keys` entry — not a valid Ed25519 PEM PUBLIC KEY block. Emitted per invalid entry. |
| `E-CFG-010` | broken | 1 | `sbctl --key` file load failed — missing, oversized (>64 KiB), not OpenSSH PEM, or not an Ed25519 key. No connection attempt is made. |
| `E-CFG-011` | broken | 1 | Private key export not supported (defensive; unreachable under normal operation). |
| `E-CFG-012` | broken | 2 | `--yes` cannot be combined with `--confirm` on the same command. Pick one. |
| `E-CFG-013` | broken | 2 | Non-interactive session: `--confirm` is required for scripted use. Use `--confirm=<svtn-short-id>` or `--yes`. |

---

## NET — Network

| Code | Severity | Exit | Summary |
|------|----------|------|---------|
| `E-NET-001` | broken | 1 | Daemon unreachable — dial failed, or handshake read-deadline timeout. Message: `"daemon unreachable: <address>: <detail>"`. |
| `E-NET-006` | broken | 1 | Path drain in progress — router refuses new frames while draining. (Emission site pending `S-7.04`.) |

---

## RPC — Post-auth dispatch

Server-side handler failures after authentication has succeeded. These
travel back to the client as JSON envelope errors with `ok: false` and
`error.code` populated.

| Code | Severity | Exit | Summary |
|------|----------|------|---------|
| `E-RPC-001` | broken | 1 | Client-side RPC dispatch failure — e.g. wire encoding error before send. |
| `E-RPC-002` | broken | 1 | Server-side handler error — reserved for structural handler failures. |
| `E-RPC-003` | broken | 1 | Server-side handler error — reserved variant. |
| `E-RPC-010` | broken | 1 | Server reports an unknown command. |
| `E-RPC-011` | broken | 1 | Server handler returned an error — generic server-side handler failure surfaced in-band. |

---

## SVTN — SVTN management

| Code | Severity | Exit | Summary |
|------|----------|------|---------|
| `E-SVTN-001` | broken | 1 | SVTN already exists. |
| `E-SVTN-003` | broken | 1 | SVTN not found. |

---

## INT — Internal

Catch-all for internal daemon errors that leak to the operator.

| Code | Severity | Exit | Summary |
|------|----------|------|---------|
| `E-INT-001` | broken | 1 | Internal handler error — non-duplicate SVTN create failure. |
| `E-INT-999` | broken | 1 | Catch-all sentinel for unmapped internal conditions. Presence at runtime indicates a code defect worth filing an issue about. |

---

## Programmatic use

- **Match on `error.code`**, not on the message. Message text is
  intentionally informative and may reword between minor releases;
  codes are stable.
- **Retry safety** — `E-NET-001` and daemon-side `E-INT-*` errors are
  safe to retry idempotently. `E-ADM-*` and `E-CFG-*` errors are not —
  the caller must fix input before retrying.
- **Wire format** — the operator-auth failure response
  (`E-ADM-010`) is deliberately identical for "unrecognized key" and
  "wrong signature". Do not attempt to distinguish; the daemon returns
  the same envelope in both cases to prevent key enumeration.

---

## Where these come from

The taxonomy is authored and versioned in
`.factory/specs/prd-supplements/error-taxonomy.md`; that document is
the source of truth. This reference is a curated projection for
operators. If a code you observe is not in this document, consult the
taxonomy source or file an issue against the release.
