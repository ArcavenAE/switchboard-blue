---
artifact_id: error-taxonomy
document_type: prd-supplement-error-taxonomy
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-25T00:00:00
phase: 1a
inputs:
  - '.factory/specs/prd.md'
  - '.factory/specs/domain-spec/failure-modes.md'
  - '.factory/specs/domain-spec/edge-cases.md'
input-hash: "[md5-pending]"
traces_to: '.factory/specs/prd.md'
---

# Error Taxonomy: Switchboard

> PRD supplement — extracted from PRD Section 5.
> Referenced by: implementer, test-writer.

## Error Categories

| Category Code | Category | Description |
|--------------|----------|-------------|
| ADM | Admission/Auth | Authentication, admission, authorization failures |
| CFG | Configuration | Config parse, validation, missing field errors |
| NET | Network | Daemon unreachable, connection refused, timeout |
| PRT | Protocol | Frame format, version, encoding errors |
| FWD | Forwarding | Routing, path selection, loop detection errors |
| SES | Session | Session lifecycle, attach, detach errors |
| SVTN | SVTN Management | SVTN create, destroy, lifecycle errors |
| SYS | System | OS-level errors: PTY unavailable, file descriptor limit |

## Severity Definitions

| Severity | Meaning | Exit Code Impact |
|----------|---------|-----------------|
| broken | Operation cannot complete; operator action required | Non-zero exit |
| degraded | Partial operation; reduced functionality; logged clearly | Zero exit with warning (daemon continues) |
| cosmetic | Display or formatting issue; no functional impact | Zero exit |

## Error Catalog

### ADM — Admission/Authorization

| Error Code | Category | Severity | Exit Code | Message Format | FM/DEC Source |
|-----------|----------|----------|-----------|----------------|---------------|
| E-ADM-001 | ADM | broken | 1 | "admission denied: signature verification failed for <node_addr> on SVTN <svtn_id>" | BC-2.05.001 |
| E-ADM-002 | ADM | broken | — (dropped) | "HMAC verification failed: SVTN <svtn_id>, src <src_addr>, type <frame_type>" | FM-006, BC-2.05.005 |
| E-ADM-016 | ADM | broken | 0 | "wire HMAC verification failed at RouteFrame: tag mismatch for SVTN <svtn_id> from src <src_addr>" | BC-2.05.008; mapped to Go sentinel routing.ErrHMACVerificationFailed; distinct from E-ADM-002 which covers HMAC primitive failure in internal/hmac |
| E-ADM-003 | ADM | broken | — (dropped) | "frame from non-admitted source: src <src_addr>, SVTN <svtn_id>" | BC-2.05.002 |
| E-ADM-004 | ADM | broken | 1 | "address collision: node address <addr> already admitted on SVTN <svtn_id>" | BC-2.01.006 |
| E-ADM-005 | ADM | broken | 1 | "key revoked: <key_fingerprint> on SVTN <svtn_id>" | DEC-005, FM-007 |
| E-ADM-006 | ADM | broken | 1 | "session authorization denied: console <key_fingerprint> not authorized for session <session_name> on <node_addr>" | DEC-006, BC-2.05.003 |
| E-ADM-007 | ADM | degraded | 0 (continues) | "upstream rejected: read-only access for console <key_fingerprint> on session <session_name>" | BC-2.04.005 |
| E-ADM-008 | ADM | broken | 1 | "nonce replay: challenge nonce already consumed for <node_addr>" | BC-2.05.001 |
| E-ADM-009 | ADM | broken | 1 | "insufficient authority for operation <operation>: key <key_fingerprint> has role <role>" | BC-2.05.004, BC-2.07.002 |
| E-ADM-010 | ADM | broken | 1 | "authentication failed: key <key_fingerprint> not authorized for daemon at <address>" | BC-2.07.002 |
| E-ADM-011 | ADM | broken | 1 | "permission denied: <role> key cannot revoke <target_role> key (control > console > readonly)" | BC-2.05.004 |
| E-ADM-012 | ADM | broken | 1 | "key already registered: pubkey <key_fingerprint> already exists for SVTN <svtn_id>" | BC-2.05.004 (register-key) |
| E-ADM-013 | ADM | broken | 1 | "key not found: no key with fingerprint <key_fingerprint> registered in SVTN <svtn_id>" | BC-2.05.004 (revoke-key) |
| E-ADM-014 | ADM | broken | 1 | "bootstrap key mismatch: provided key does not match SVTN <svtn_id> bootstrap" | ADR-004 (recover) |
| E-ADM-015 | ADM | broken | 1 | "key expired: <key_fingerprint> on SVTN <svtn_id> (expired at <expiry_time>)" | FM-013, BC-2.01.007 |

### CFG — Configuration

| Error Code | Category | Severity | Exit Code | Message Format | FM/DEC Source |
|-----------|----------|----------|-----------|----------------|---------------|
| E-CFG-001 | CFG | broken | 1 | "config error: <field>: <problem>. Fix: <suggestion>" | FM-010, BC-2.09.003 |
| E-CFG-002 | CFG | broken | 1 | "private key export not supported: <reason>" | BC-2.05.007 (defensive: emitted if any attempted private-key extraction path is invoked; BC-2.05.007 requires this path to be unreachable. Presence of this code at runtime would indicate a code defect.) |
| E-CFG-003 | CFG | broken | 1 | "invalid upstream router address: <addr>. Expected format: <ip>:<port>" | BC-2.09.001 |
| E-CFG-004 | CFG | broken | 1 | "config file not found: <path>" | BC-2.09.003 |
| E-CFG-005 | CFG | broken | 1 | "config parse error: invalid YAML at line <N>: <detail>" | FM-010, BC-2.09.003 |
| E-CFG-006 | CFG | broken | 2 | "--yes cannot be combined with --confirm; pick one" | interface-definitions.md sbctl admin |

### NET — Network

| Error Code | Category | Severity | Exit Code | Message Format | FM/DEC Source |
|-----------|----------|----------|-----------|----------------|---------------|
| E-NET-001 | NET | broken | 1 | "daemon unreachable: <address>: <reason>" | FM-012, BC-2.07.003 |
| E-NET-002 | NET | degraded | 0 | "no active paths: all router connections lost for SVTN <svtn_id>" | DEC-004, BC-2.02.001 |
| E-NET-003 | NET | degraded | 0 | "router unreachable after IP change: <router_addr>; retrying" | DEC-001, BC-2.01.007 |
| E-NET-004 | NET | degraded | 0 | "path failed: <router_addr>: 3 consecutive keep-alives missed; removing from active set" | BC-2.02.003 |
| E-NET-005 | NET | broken | 1 | "access node unreachable: <node_addr> for session <session_name>" | BC-2.04.003 |
| E-NET-006 | NET | broken | 1 | "router draining; connect to alternate router at <alternates_list>" | BC-2.09.002 |

### PRT — Protocol

| Error Code | Category | Severity | Exit Code | Message Format | FM/DEC Source |
|-----------|----------|----------|-----------|----------------|---------------|
| E-PRT-001 | PRT | broken | — (dropped) | "unsupported protocol version <N>: expected major version <M>" | DEC-008, BC-2.01.004 |
| E-PRT-002 | PRT | broken | 1 | "header truncated: expected <N> bytes, got <M>" | BC-2.01.004 |
| E-PRT-003 | PRT | broken | 1 | "frame truncated: outer header complete but frame body shorter than indicated length" | BC-2.01.005 |

### Namespace Aliases (informational)

Some scenario documents (e.g., HS-001 v1.1) use the prefix `E-FRM-*` for
protocol-layer framing errors synonymously with `E-PRT-*`. The canonical names
use `E-PRT-*`. No renaming is planned. The aliases arose because the holdout
scenario was authored before the category-code table above was finalized; the
`errors.Is` identity checks in the scenario still passed because the underlying
sentinel values are the same.

Mapping for cross-reference:

| Alias (non-canonical) | Canonical | Notes |
|-----------------------|-----------|-------|
| E-FRM-001 | E-PRT-001 | Unsupported protocol version |
| E-FRM-002 | E-PRT-002 | Header truncated |

All new scenarios and BCs MUST use the canonical `E-PRT-*` names.
This note added per drbothen/vsdd-factory#260 rollback (holdout-discovered, 2026-06-24).

### FWD — Forwarding

| Error Code | Category | Severity | Exit Code | Message Format | FM/DEC Source |
|-----------|----------|----------|-----------|----------------|---------------|
| E-FWD-001 | FWD | degraded | 0 | "split-horizon: no non-arrival interface available for dst <dst_addr>; frame dropped" | BC-2.02.008 |
| E-FWD-002 | FWD | degraded | — (dropped) | "routing: no forwarding entry for destination <dst_addr> in SVTN <svtn_id>" | BC-2.05.006; distinguishes forwarding-table miss from admission failure (E-ADM-003); callers use errors.Is to separate the two conditions |

### SES — Session

| Error Code | Category | Severity | Exit Code | Message Format | FM/DEC Source |
|-----------|----------|----------|-----------|----------------|---------------|
| E-SES-001 | SES | broken | 1 | "session not found: <session_name> on SVTN <svtn_id>" | BC-2.04.003 |

### SVTN — SVTN Management

| Error Code | Category | Severity | Exit Code | Message Format | FM/DEC Source |
|-----------|----------|----------|-----------|----------------|---------------|
| E-SVTN-001 | SVTN | broken | 1 | "SVTN already exists: <svtn_id>" | BC-2.07.001 |
| E-SVTN-002 | SVTN | broken | 1 | "SVTN bootstrap already complete: <svtn_id>" | BC-2.07.001 |

### SYS — System

| Error Code | Category | Severity | Exit Code | Message Format | FM/DEC Source |
|-----------|----------|----------|-----------|----------------|---------------|
| E-SYS-001 | SYS | broken | 1 | "PTY device unavailable: cannot start access node. Install 'openpty' or check device permissions." | FM-011, BC-2.04.002 |

## Failure Mode to Error Code Mapping

| FM-NNN | Failure Mode | Relevant Error Codes |
|--------|-------------|---------------------|
| FM-001 | Single router failure (E phase) | E-NET-002, E-NET-004 |
| FM-002 | All paths degrade | E-NET-002, quality indicator red |
| FM-003 | Frame duplication storm | E-FWD-001 (drop cache metric), operator diagnostic |
| FM-004 | Access node loses tmux control mode | No error code — degradation signal + log |
| FM-005 | Presence message lost/stale | No error code — eventual consistency |
| FM-006 | HMAC verification failure (primitive layer) | E-ADM-002 |
| FM-014 | Wire-layer HMAC mismatch at RouteFrame (tag mismatch from admitted node) | E-ADM-016 |
| FM-007 | Key revocation propagation delay | Acknowledged gap; no error code |
| FM-008 | Quality indicator stuck green | Bug in DI-008 implementation |
| FM-009 | Router crashes without drain | E-NET-004 (detected by nodes) |
| FM-010 | Config error on startup | E-CFG-001, E-CFG-004, E-CFG-005 |
| FM-011 | tmux not present | E-SYS-001 (if PTY also fails); log message on PTY fallback |
| FM-012 | sbctl cannot connect | E-NET-001 |
| FM-013 | Key expired at re-authentication time | E-ADM-015 |
| (no FM) | Forwarding-table miss for (svtnID, dstAddr) — distinct from admission failure | E-FWD-002 |
