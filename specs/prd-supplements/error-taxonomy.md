---
artifact_id: error-taxonomy
document_type: prd-supplement-error-taxonomy
level: L3
version: "4.2"
status: draft
producer: product-owner
timestamp: 2026-06-29T00:00:00
modified:
  - 2026-06-28T00:00:00 # v2.5 — Wave-5 mgmt-plane adversarial review Rulings 1/3/4/5/6/7 (ARCH-12 v1.2): E-CFG-010 added (key load failure); RPC category added with E-RPC-001 (post-auth dispatch failure); E-NET-001 scope-clarification note added (strictly dial/connect-unreachable)
  - 2026-06-29T00:00:00 # v2.6 — ARCH-12 v1.3 Wave-5 Convergence Ruling C: E-RPC-010 added (server unknown command, in-band), E-RPC-011 added (server handler error, in-band); E-RPC-001 clarified as CLIENT-SIDE only; E-RPC-002 explicitly forbidden. Ruling D: E-CFG-008 message format extended with console TCP loopback rejection variant (buildMgmtListener, not config.Validate())
  - 2026-06-29T00:00:00 # v2.7 — ARCH-12 v1.4 Wave-5 Convergence Ruling L: E-CFG-008 Variant 2 canonical string corrected — buildMgmtListener embeds the error code as prefix ("E-CFG-008: management_socket: ..."); Variant 1 (config.Validate empty socket) does NOT embed the code (different error-reporting path); test assertions must use strings.Contains(err.Error(), "E-CFG-008") not full-string match
  - 2026-06-29T00:00:00 # v2.8 — ARCH-12 v1.5 Wave-5 Convergence Ruling Y: E-NET-001 scope extended — now explicitly covers two cases: (a) net.Dial/net.DialContext failure (unchanged); (b) handshake read-deadline timeout (treated as unreachable per BC-2.07.003 Inv-2); message format for case (b) is "daemon unreachable: <address>: connection timed out"; reconciles prior Inv-2 vs Inv-4 conflict in BC-2.07.003
  - 2026-06-29T00:00:00 # v2.9 — S-6.05 PO ruling: E-ADM-011 extended with Variant 2 (destroy authorization); ErrDestroyUnauthorized sentinel maps to E-ADM-011; no new code slot allocated
  - 2026-06-29T00:00:00 # v3.0 — Task 7 reconverge (S-5.01 + S-6.02 Pass-1 adversarial, lens1 F-001): E-ADM-004 KEPT as "address collision" (BC-2.01.006 predates ARCH-04 addendum); E-ADM-014 KEPT as "bootstrap key mismatch" (ADR-004 recover). New slots: E-ADM-018 ("control-to-control revocation requires explicit confirmation", S-6.02 + ARCH-04 HOLD-001); E-ADM-019 ("role mismatch: claimed role does not match registered key role", HOLD-001 cross-check). NOTE: E-ADM-015 (key expired) and E-ADM-016 (wire HMAC mismatch) are occupied — new entries use next free slots E-ADM-018 and E-ADM-019.
  - 2026-06-30T00:00:00 # v3.1 — S-6.06 Pass-4 ruling F-L2-002: E-ADM-011 scope disambiguated — it is returned by SVTNManager.RevokeKey at the Go API layer (unit-test path) for a revocation-hierarchy violation. It is NOT reachable via the mgmt RPC path when the handler-layer authority gate (E-ADM-009) is wired: the gate fires first and rejects non-control callers before SVTNManager.RevokeKey is ever invoked.
  - 2026-06-30T00:00:00 # v3.2 — S-6.06 Pass-6 rulings F-P6L3-001: E-SVTN-003 added (SVTN not found); closes missing code referenced in S-6.06 Error Code Map
  - 2026-06-30T00:00:00 # v3.3 — F-P8L2-003: E-ADM-020 added (bootstrap-key-revoke-forbidden); emitted by admin.key.revoke handler when svtnmgmt.ErrBootstrapKeyRevokeForbidden; source BC-2.05.004 EC-007
  - 2026-06-30T00:00:00 # v3.4 — F-P10L2-004 (MED): E-ADM-018 message format — <svtn-short-id> → <svtn-id> (matches impl and all other ADM messages that take svtn parameter; full svtnName used in impl, not abbreviated form)
  - 2026-06-30T00:00:00 # v3.5 — F-P11L2-002 (MED): E-ADM-018 message — stripped backticks from --confirm=<svtn-id> for byte-identical canonical text
  - 2026-06-30T00:00:00 # v3.6 — F-P14L2-001 (LOW): backfill v3.4 changelog table row (frontmatter had the entry; human-readable Changelog table was missing it)
  - 2026-06-30T00:00:00 # v3.7 — F-P17L2-001 (MED) + F-P17L2-002 (LOW): E-ADM-020 description + canonical message aligned to BC-2.05.004 v1.9 unconditional phrasing (lens-2 pass-17)
  - 2026-06-30T00:00:00 # v3.8 — F-P18L1-001 (MED): E-ADM-021 minted — bootstrap-key-expire-forbidden symmetric counterpart to E-ADM-020 (refs F-P18L1-001 lens-1 pass-18)
  - 2026-06-30T00:00:00 # v3.9 — Pass-22 F-P22L3-002 sibling-fix (4th-iteration narrowing sweep): E-ADM-020 + E-ADM-021 descriptions narrowed from "unconditionally" to "for any well-formed request"; BC citation updated from v1.10 to v1.12; E-CFG-001 handler-layer layering note added to both rows
  - 2026-07-01T00:00:00 # v4.0 — Wave-6 Tranche A Ruling-4 sibling-propagation (S-6.07 F-P2L3-002): E-ADM-009 FM/DEC Source extended with BC-2.07.001 Inv-3; E-INT-001 minted (internal handler error for non-duplicate admin.svtn.create failure, BC-2.07.001 PC-1); INT category added to category table
  - 2026-07-01T00:00:00 # v4.1 — Ruling-12 §1 universality follow-through: E-INT-999 minted as catch-all sentinel for unmapped internal conditions (default arm of mapAdminError or equivalent); wire envelope follows E-RPC-011 pattern; Ruling-12 §7 process policy added to rulings doc
  - 2026-07-02T00:00:00 # v4.2 — Phase 5 Pass 1 remediation: E-NET-006 BC-anchor annotated with PENDING-S-7.04 (emission site absent from cmd/internal, router drain runtime stubbed); closes DRIFT-P5P1-B-H001-ENET006-TAXONOMY-ORPHAN + DRIFT-P5P1-B-L001
phase: 1a
inputs:
  - '.factory/specs/prd.md'
  - '.factory/specs/domain-spec/failure-modes.md'
  - '.factory/specs/domain-spec/edge-cases.md'
input-hash: "bc4367a"
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
| RPC | Remote Procedure Call | Post-auth RPC dispatch failures: server error after AUTH_OK |
| INT | Internal | Unexpected internal errors surfaced to the operator from handler or service layer |

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
| E-ADM-016 | ADM | broken | 0 | **PATH-A (auth key unavailable):** `"wire HMAC verification failed at RouteFrame: auth key unavailable for SVTN <svtn_id> from src <src_addr> (E-ADM-016)"` — emitted when the routing table has no HMAC key for the SVTN (key lookup returns ok=false); fail-closed: frame is dropped. **PATH-B (tag mismatch):** `"wire HMAC verification failed at RouteFrame: tag mismatch for SVTN <svtn_id> from src <src_addr> (E-ADM-016)"` — emitted when a key is present but the computed HMAC tag does not match the frame's tag. Both paths return the same Go sentinel `routing.ErrHMACVerificationFailed` and are indistinguishable at the `errors.Is` level; the distinct message variants carry the operator-visible discriminator. `<svtn_id>` and `<src_addr>` are the lowercase hex encodings of the respective 8-byte fields (e.g. "a1b2c3d4e5f60102"); code literal "(E-ADM-016)" embedded in both messages for grep-ability. Distinct from E-ADM-002 (HMAC primitive failure in internal/hmac) and E-ADM-017 (aggregate rate alert). Both variants verified in internal/routing/routing.go (Wave-3 audit F-1.2). | BC-2.05.008; mapped to Go sentinel routing.ErrHMACVerificationFailed; both PATH-A and PATH-B verified in internal/routing/routing.go lines 200 and 215 |
| E-ADM-017 | ADM | degraded | 0 (daemon continues) | "E-ADM-017 HMAC failure rate alert: ≥<threshold> failures in <window_seconds>s from src <src_addr>" | BC-2.05.005 PC-3, BC-2.05.008 EC-006; emitted by admission.FailureCounter when the per-src_addr sliding-window count reaches the configured threshold; `<threshold>` and `<window_seconds>` are the FailureCounter's configured values (default: 5 and 60); `<src_addr>` is the lowercase hex encoding of the 8-byte SrcAddr field; code literal "E-ADM-017" embedded at message start for grep-ability; re-arm semantics: alert fires on threshold crossing; the counter re-arms only when the sliding window fully drains (len(keep)==0 after trim, i.e. no in-window entries remain); while fired, new timestamps are not appended to the per-source slice (append-skip), bounding it at threshold (drain-only re-arm per BC-2.05.005 v1.6 PC-3, VP-059 v1.1 property (c)); severity=degraded because the router continues operating; distinct from E-ADM-016 (per-failure wire log) and E-ADM-002 (per-failure primitive log) |
| E-ADM-003 | ADM | broken | — (dropped) | "frame from non-admitted source: src <src_addr>, SVTN <svtn_id>" | BC-2.05.002 |
| E-ADM-004 | ADM | broken | 1 | "address collision: node address <addr> already admitted on SVTN <svtn_id>" | BC-2.01.006 |
| E-ADM-005 | ADM | broken | 1 | "key revoked: <key_fingerprint> on SVTN <svtn_id>" | DEC-005, FM-007 |
| E-ADM-006 | ADM | broken | 1 | "session authorization denied: console <key_fingerprint> not authorized for session <session_name> on <node_addr>" | DEC-006, BC-2.05.003 |
| _(layering note)_ | | | | **internal/session layer:** `ConsoleKey` serves as `<key_fingerprint>` and `<node_addr>` is omitted — the session layer has no node identity (ARCH-08 §6.6, position-6). The transport/admission boundary caller, which owns node identity, supplies `<node_addr>` when re-surfacing this error to the operator. `errors.Is` identity is preserved via `%w` wrapping. | |
| E-ADM-007 | ADM | degraded | 0 (continues) | "upstream rejected: read-only access for console <key_fingerprint> on session <session_name>" | BC-2.04.005 |
| _(layering note)_ | | | | **internal/session layer:** same layering applies as E-ADM-006 — `ConsoleKey` serves as `<key_fingerprint>`; `<node_addr>` is omitted at this layer and supplied by the transport/admission boundary caller when re-surfacing. **Static sentinel + caller-wrapping pattern (Wave-3 audit F-1.3):** `ErrUpstreamReadOnly` (internal/session/auth.go:50) is a static `errors.New` sentinel — `"session: upstream rejected: read-only access (E-ADM-007)"` — intentionally omitting `<key_fingerprint>` and `<session_name>` because those values are not available at sentinel-declaration time. The caller (`SessionAuth.Allow`, auth.go:146) adds parametric context via `fmt.Errorf("upstream rejected: read-only access for console %s on session %s: %w", key, sessionName, ErrUpstreamReadOnly)`, producing the full canonical message format above. `errors.Is` identity is preserved via `%w`-wrapping. The sentinel's `"session:"` prefix and `"(E-ADM-007)"` suffix are implementation artifacts not present in the operator-visible canonical format; grep patterns targeting E-ADM-007 should match on the `"(E-ADM-007)"` literal in the sentinel OR on `"read-only access for console"` in wrapped messages. | |
| E-ADM-008 | ADM | broken | 1 | "nonce replay: challenge nonce already consumed for <node_addr>" | BC-2.05.001 |
| E-ADM-009 | ADM | broken | 1 | "insufficient authority for operation <operation>: key <key_fingerprint> has role <role>" | BC-2.05.004, BC-2.07.002, BC-2.07.001 Inv-3 |
| E-ADM-010 | ADM | broken | 1 | "authentication failed: key <key_fingerprint> not authorized for daemon at <address>" | BC-2.07.002 (client-side: operator CLI auth failure); BC-2.07.004 (server-side: wire format `{"type":"auth_fail","code":"E-ADM-010","message":"authentication failed"}` — response is identical for unrecognized key and wrong signature to prevent oracle; no key_fingerprint in wire message to prevent enumeration). E-ADM-010 is the **canonical operator-auth-failure code** for both sbctl (client) and internal/mgmt (server). Distinct from E-ADM-001 (SVTN node admission failure). |
| E-ADM-011 | ADM | broken | 1 | E-ADM-011 has two message variants depending on the operation being denied. **Variant 1 (revocation hierarchy — existing):** `"permission denied: <role> key cannot revoke <target_role> key (control > console > readonly)"` — emitted by `SVTNManager.RevokeKey` when the caller's key role is insufficient to revoke the target key's role (e.g., console attempting to revoke a control key). **Variant 2 (destroy authorization — S-6.05):** `"permission denied: <role> key cannot destroy SVTN <svtn_name>"` — emitted by `SVTNManager.Destroy` when the caller's key is not a control-role key (BC-2.07.001 Invariant 3). Both variants use the Go sentinel `ErrRoleMismatch` (existing, Variant 1) or the new `ErrDestroyUnauthorized` (Variant 2, S-6.05) and share the E-ADM-011 code because both represent the same class of error: insufficient privilege for an admission-plane operation requiring control authority. `errors.Is` identity uses the respective sentinel for each path. **Scope disambiguation (S-6.06 Pass-4 F-L2-002):** E-ADM-011 is a Go API-layer code returned by `SVTNManager.RevokeKey` or `SVTNManager.Destroy` directly. It is NOT reachable via the `admin.key.*` mgmt RPC path when the handler-layer authority gate is correctly wired — the gate returns E-ADM-009 to non-control callers before `SVTNManager.RevokeKey` is ever invoked. E-ADM-011 is reachable only via direct Go API calls (unit-test path). | BC-2.05.004 (Variant 1); BC-2.07.001 Inv-3 (Variant 2, S-6.05) |
| E-ADM-012 | ADM | broken | 1 | "key already registered: pubkey <key_fingerprint> already exists for SVTN <svtn_id>" | BC-2.05.004 (register-key) |
| E-ADM-013 | ADM | broken | 1 | "key not found: no key with fingerprint <key_fingerprint> registered in SVTN <svtn_id>" | BC-2.05.004 (revoke-key) |
| E-ADM-014 | ADM | broken | 1 | "bootstrap key mismatch: provided key does not match SVTN <svtn_id> bootstrap" | ADR-004 (recover) |
| E-ADM-015 | ADM | broken | 1 | "key expired: <key_fingerprint> on SVTN <svtn_id> (expired at <expiry_time>)" | FM-013, BC-2.01.007 |
| E-ADM-018 | ADM | broken | 1 | "control-to-control revocation requires explicit confirmation: use --confirm=<svtn-id> to proceed" | S-6.02 + ARCH-04 HOLD-001 (split-brain mitigation); emitted by `SVTNManager.RevokeKey` when a control-role key attempts to revoke another control-role key without the explicit `--confirm` token. The `ErrControlRevocationRequiresConfirm` sentinel maps to this code. Distinct from E-ADM-011 (hierarchy violation: console/readonly cannot revoke control at all — that is an unconditional deny). E-ADM-018 is a conditional deny: the operation is permitted for control-to-control but requires explicit confirmation. |
| E-ADM-019 | ADM | broken | 1 | "role mismatch: claimed role <claimed_role> does not match registered key role <registered_role> for key <key_fingerprint>" | ARCH-04 HOLD-001 cross-check; emitted when the key-role asserted in an RPC request's auth metadata does not match the role stored in the admitted-key set for that key's fingerprint. This prevents a caller from escalating privileges by claiming a higher role than their registered key actually has. Distinct from E-ADM-009 (insufficient authority: correct role known but too low) and E-ADM-010 (auth failure: key not recognized at all). |
| E-ADM-020 | ADM | broken | 1 | `"bootstrap-key-revoke-forbidden: cannot revoke the bootstrap key in SVTN <svtn_id> (permanent trust anchor)"` | BC-2.05.004 EC-007 (bootstrap-key revoke protection); emitted by the `admin.key.revoke` handler in `cmd/switchboard/admin_handlers.go` when `svtnmgmt.ErrBootstrapKeyRevokeForbidden` is returned. Per BC-2.05.004 EC-007 (v1.12), the bootstrap key is non-revocable for any well-formed request, regardless of whether other control keys have been registered. Malformed requests (invalid fields, parse errors) are rejected at the handler with E-CFG-001 before SVTNManager is invoked. Prevents management lockout. Distinct from E-ADM-011 (hierarchy violation: lower-tier role cannot revoke higher-tier key), E-ADM-018 (control-to-control revocation requires explicit confirmation), and E-ADM-009 (caller lacks control authority). `<svtn_id>` is the target SVTN identifier. The Go sentinel `svtnmgmt.ErrBootstrapKeyRevokeForbidden` maps exclusively to this code. |
| E-ADM-021 | ADM | broken | 1 | `"bootstrap-key-expire-forbidden: cannot expire the bootstrap key in SVTN <svtn_id> (permanent trust anchor)"` | BC-2.05.004 EC-007 (bootstrap-key expire protection; v1.12); emitted by the `admin.key.expire` handler in `cmd/switchboard/admin_handlers.go` (mapAdminError arm) when `svtnmgmt.ErrBootstrapKeyExpireForbidden` is returned. Per BC-2.05.004 EC-007 (v1.12), the bootstrap key is non-expirable for any well-formed request, regardless of whether other control keys have been registered. Malformed requests (invalid fields, parse errors) are rejected at the handler with E-CFG-001 before SVTNManager is invoked. Mirrors E-ADM-020 (revoke) for symmetric management-lockout prevention; expiry has the same functional lockout effect as revocation (per BC-2.05.004 EC-004: key expires while session active = same behavior as revocation). The Go sentinel `svtnmgmt.ErrBootstrapKeyExpireForbidden` maps exclusively to this code. `<svtn_id>` is the target SVTN identifier. Tests: `TestMapAdminError_ErrorWrapping/ErrBootstrapKeyExpireForbidden`. (O-P20L3-001: cite only the expire test; revoke sentinel is covered by E-ADM-020's test citation.) |

### CFG — Configuration

| Error Code | Category | Severity | Exit Code | Message Format | FM/DEC Source |
|-----------|----------|----------|-----------|----------------|---------------|
| E-CFG-001 | CFG | broken | 1 | "config error: <field>: <problem>. Fix: <suggestion>" | FM-010, BC-2.09.003 |
| E-CFG-002 | CFG | broken | 1 | "private key export not supported: <reason>" | BC-2.05.007 (defensive: emitted if any attempted private-key extraction path is invoked; BC-2.05.007 requires this path to be unreachable. Presence of this code at runtime would indicate a code defect.) |
| _(collision flag)_ | | | | **KNOWN INCONSISTENCY (F-003):** BC-2.09.003 v1.2 uses E-CFG-002 for `listen_addr` invalid host:port validation — a different failure mode from this table's definition ("private key export not supported"). This pre-existing collision predates Wave 5 and was not flagged at the time of authorship (compare: the E-CFG-006 collision was flagged explicitly). Reconciliation is needed in a maintenance pass: either (a) BC-2.09.003's listen_addr code should be renumbered (e.g., to E-CFG-010), or (b) this table's E-CFG-002 (private-key export) should be renumbered. No renumbering in this pass — the collision is now tracked. | BC-2.09.003 v1.2 vs. error-taxonomy.md |
| E-CFG-003 | CFG | broken | 1 | "invalid upstream router address: <addr>. Expected format: <ip>:<port>" | BC-2.09.001 |
| E-CFG-004 | CFG | broken | 1 | "config file not found: <path>" | BC-2.09.003 |
| E-CFG-005 | CFG | broken | 1 | "config parse error: invalid YAML at line <N>: <detail>" | FM-010, BC-2.09.003 |
| E-CFG-006 | CFG | broken | 2 | "--yes cannot be combined with --confirm; pick one" | interface-definitions.md sbctl admin |
| _(collision flag)_ | | | | **KNOWN INCONSISTENCY:** BC-2.09.003 v1.4 uses E-CFG-006 for `drain_timeout` negative validation and E-CFG-007 for `keepalive_interval` negative validation — these codes are absent from this taxonomy table (which uses E-CFG-006 for the sbctl admin flag conflict). This pre-existing discrepancy predates Wave 5. Reconciliation is needed in a maintenance pass: either (a) the BC-2.09.003 codes (drain/keepalive negative) should be renumbered (note: E-CFG-010 is now assigned to sbctl key-load failure per Wave-5 Ruling 5; renumber to E-CFG-011/E-CFG-012 or other free slots), or (b) the taxonomy E-CFG-006 (sbctl admin) should be renumbered. The codes E-CFG-008 and E-CFG-009 (Wave-5 management plane) are free in both documents and are used as-is. | BC-2.09.003 v1.4 vs. this table |
| E-CFG-008 | CFG | broken | 1 | E-CFG-008 has two distinct message formats depending on originating site. **Variant 1 (empty/whitespace management_socket — `config.Validate()`):** `"config error: management_socket: must not be empty. Fix: set to a valid Unix socket path, e.g. '/run/switchboard-router.sock', or remove the field to use the daemon default"` — emitted by `config.Validate()` when the `management_socket` field is present but empty or whitespace-only (BC-2.09.003 PC-10, Wave-5). The error code is returned as a structured field by `Validate()`; it is NOT embedded in the message string. **Variant 2 (console TCP non-loopback — `buildMgmtListener`, Ruling D/L):** `"E-CFG-008: management_socket: console mode requires a loopback address (127.0.0.1, [::1], or localhost); got: <address>"` — emitted by `buildMgmtListener` (`cmd/switchboard/mgmt_wire.go`) when the console-mode TCP management socket address has a non-loopback host (0.0.0.0, ::, bare port, or any non-loopback IP). The error code `E-CFG-008` IS embedded as a prefix in the format string for grep-ability in logs. This variant is generated in the wiring layer, NOT by `config.Validate()`. Daemon startup aborts. BC-2.07.004 EC-013. Both variants share the E-CFG-008 code for taxonomy lookup. **Test assertions MUST use `strings.Contains(err.Error(), "E-CFG-008")` rather than a full-string match** to remain correct across both variants and tolerate minor message rewording. |
| E-CFG-009 | CFG | broken | 1 | "config error: authorized_operator_keys[<N>]: entry is not a valid Ed25519 PEM PUBLIC KEY block. Fix: provide a PEM-encoded Ed25519 public key (type 'PUBLIC KEY', 32-byte key length)" | BC-2.09.003 PC-11 (Wave-5); emitted per-entry when `authorized_operator_keys` contains a malformed or wrong-type PEM block. Exhaustive: all invalid entries reported before exit. |
| E-CFG-010 | CFG | broken | 1 | "key load failed: \<path\>: \<reason\>" | BC-2.07.003 EC-005 (Wave-5 Ruling 5); emitted by sbctl when the `--key` file is absent, oversized (> 64 KiB), not valid OpenSSH PEM, or contains a non-Ed25519 key. `<path>` is the value of the `--key` flag. `<reason>` is the specific sub-error (e.g. "no such file or directory", "file exceeds 64 KiB limit", "not an Ed25519 key"). Distinct from E-CFG-008 (management_socket empty) and E-CFG-009 (authorized_operator_keys PEM). Next free slot in the CFG family (E-CFG-010). No connection attempt is made when this code is emitted. |

### NET — Network

| Error Code | Category | Severity | Exit Code | Message Format | FM/DEC Source |
|-----------|----------|----------|-----------|----------------|---------------|
| E-NET-001 | NET | broken | 1 | "daemon unreachable: <address>: <reason>" | FM-012, BC-2.07.003. **Scope clarification (Wave-5 Ruling 5; extended by Ruling Y / ARCH-12 v1.5):** E-NET-001 is emitted for two cases: **(a) Dial failure:** `net.Dial`/`net.DialContext` failure — the daemon truly cannot be reached (connection refused, DNS failure, etc.). **(b) Handshake timeout (Ruling Y):** The daemon accepted the TCP connection but did not complete the ADR-012 challenge-response handshake within the timeout budget. From the operator's perspective, the daemon is effectively unreachable for management purposes — the same corrective action applies (check daemon health). The E-NET-001 message for case (b) is `"daemon unreachable: <address>: connection timed out"`. Key-load failures (before dial) are E-CFG-010; post-auth (post-AUTH_OK) dispatch failures are E-RPC-001. These failure modes MUST NOT share codes. See BC-2.07.003 Invariant 4 (v1.5) for the authoritative spec. |
| E-NET-002 | NET | degraded | 0 | "no active paths: all router connections lost for SVTN <svtn_id>" | DEC-004, BC-2.02.001 |
| E-NET-003 | NET | degraded | 0 | "router unreachable after IP change: <router_addr>; retrying" | DEC-001, BC-2.01.007 |
| E-NET-004 | NET | degraded | 0 | "path failed: <router_addr>: 3 consecutive keep-alives missed; removing from active set" | BC-2.02.003 |
| E-NET-005 | NET | broken | 1 | "access node unreachable: <node_addr> for session <session_name>" | BC-2.04.003 |
| E-NET-006 | NET | broken | 1 | "router draining; connect to alternate router at <alternates_list>" | BC-2.09.002 (PENDING-S-7.04: emission site not yet present in `cmd/` or `internal/` as of develop@7fe3e29e — router drain runtime is stubbed pending S-7.04 delivery; taxonomy row documents intended operator-facing message shape.) |

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
| E-SES-002 | SES | broken | 0 | "session: console <console_id> already attached to session <session_name>" | BC-2.04.003; mapped to Go sentinel session.ErrConsoleAlreadyAttached (S-3.02); prefix "session:" follows Go package-name idiomatic error prefix convention |
| E-SES-003 | SES | broken | 0 | "session: console <console_id> not found for session <session_name>" | BC-2.04.004; mapped to Go sentinel session.ErrConsoleNotFound (S-3.02); prefix "session:" follows Go package-name idiomatic error prefix convention. **Both Detach and SendKeystroke emit this format when the console key is not found in the fan-out set. Detach MUST receive session_name in its signature to satisfy this format — `Detach(key string, sessionName string) error`.** The preposition is "for" throughout — any use of "in" is a defect. |
| E-SES-004 | SES | broken | 0 | "console <console_id> not attached for command" | BC-2.08.001; S-7.03 detach-when-not-attached edge case |
| E-SES-005 | SES | RETIRED | — | RETIRED — duplicate of E-ADM-007; read-only upstream rejection is an ADM-category authorization event. See E-ADM-007. | BC-2.04.005; S-3.03 scope — superseded by E-ADM-007 |
| E-SES-006 | SES | broken | 0 | "session: console <console_id> attached to session <attached_session_name>, not <requested_session_name>" | BC-2.04.003 Inv-4 (session validation in SendKeystroke); mapped to Go sentinel session.ErrSessionMismatch (S-3.02). Emitted when `SendKeystroke(key, sessionName, payload)` is called but the console's recorded attached session_name does not match the `sessionName` argument. This is a client-error (the caller routed to the wrong access node or has stale session state). Severity: client-error / broken. Anchored to: BC-2.04.003 + session validation invariant. |

### SVTN — SVTN Management

| Error Code | Category | Severity | Exit Code | Message Format | FM/DEC Source |
|-----------|----------|----------|-----------|----------------|---------------|
| E-SVTN-001 | SVTN | broken | 1 | "SVTN already exists: <svtn_id>" | BC-2.07.001 |
| E-SVTN-002 | SVTN | broken | 1 | "SVTN bootstrap already complete: <svtn_id>" | BC-2.07.001 |
| E-SVTN-003 | SVTN | broken | 1 | "SVTN not found: <svtn_id>" | S-6.06 Error Code Map (F-P6L3-001); emitted by the `admin.key.*` handler layer when `svtnmgmt.ErrSVTNNotFound` is returned (e.g., the SVTN specified in a key register/revoke/expire/list-keys request does not exist in the SVTNManager). Distinct from E-SVTN-001 (SVTN already exists at create time) and E-SVTN-002 (bootstrap already complete). |

### SYS — System

| Error Code | Category | Severity | Exit Code | Message Format | FM/DEC Source |
|-----------|----------|----------|-----------|----------------|---------------|
| E-SYS-001 | SYS | broken | 1 | "PTY device unavailable: cannot start access node. Install 'openpty' or check device permissions." | FM-011, BC-2.04.002 |
| E-SYS-002 | SYS | broken | 1 | "fatal: cannot connect to session backend: <reason>" | BC-2.04.007 PC-1; emitted when `SessionConnector.Connect(ctx)` returns non-nil (both tmux control mode and PTY fallback exhausted). Distinct from E-SYS-001 (OS-level PTY device unavailable) — E-SYS-002 is the aggregate connect-failure at the `SessionConnector` level after all fallbacks are tried. Also emitted on E-SYS-003 (PTY-source EOF mid-session) because the `<reason>` is interpolated from `ErrPTYSourceEOF`; the wire-visible message format is always E-SYS-002 in both cases. |
| E-SYS-003 | SYS | broken | 1 | "session connector: PTY source EOF" | BC-2.04.002 invariant 3 (never silent); ARCH-01 ADR-011 v1.5 §HIGH-A. Emitted as the sentinel `ErrPTYSourceEOF` on `sc.Err()` when the active PTY source reaches EOF (shell process exits normally) without a prior `sc.Close()` call. The `forwardFrames` relay detects `srcCh==prevSrcCh` in PTY mode and sends this sentinel rather than hot-spinning. Operator-visible as E-SYS-002 format — `"fatal: cannot connect to session backend: session connector: PTY source EOF"` — because it flows through the existing `runAccess` sc.Err() drain path. E-SYS-003 is the taxonomy cross-reference for the sentinel; E-SYS-002 is the operator message format. Exit code: 1 (non-zero; same as PC-2.6 path). |

### RPC — Remote Procedure Call

| Error Code | Category | Severity | Exit Code | Message Format | FM/DEC Source |
|-----------|----------|----------|-----------|----------------|---------------|
| E-RPC-001 | RPC | broken | 1 | "rpc failed: \<command\>: \<reason\>" | BC-2.07.003 EC-006 (Wave-5 Ruling 5); **CLIENT-SIDE ONLY** — emitted by `cmd/sbctl` to stderr when an authenticated RPC request fails: the server returns `"ok":false`, the response cannot be decoded, or the connection drops after AUTH_OK. `<command>` is the RPC command name (e.g. "router.status"). `<reason>` is the specific sub-error from the server error object or the decode/connection error. Distinct from E-NET-001 (unreachable before connection), E-ADM-010 (authentication failure), and E-CFG-010 (key load failure). **E-RPC-001 MUST NOT appear in `internal/mgmt` server code** — the server uses E-RPC-010 and E-RPC-011 for its in-band response errors (Wave-5 Convergence Ruling C). |
| E-RPC-010 | RPC | broken | — (in-band response) | "unknown command: \<command\>" | BC-2.07.004 PC-11 (Wave-5 Convergence Ruling C); **SERVER-SIDE** — emitted by `internal/mgmt` server in the JSON response envelope (`"ok":false,"error":{"code":"E-RPC-010","message":"unknown command: <cmd>"}`) when an authenticated RPC request names a command that is not registered in the handler slice. The connection is NOT closed after this response. `<command>` is the unregistered command name from the request. Distinct from client-side E-RPC-001 (sbctl stderr) and server-side E-RPC-011 (handler error). The undefined `E-RPC-002` is forbidden — any occurrence in `internal/mgmt` is a defect. |
| E-RPC-011 | RPC | broken | — (in-band response) | "\<handler error message\>" | BC-2.07.004 PC-12 (Wave-5 Convergence Ruling C); **SERVER-SIDE** — emitted by `internal/mgmt` server in the JSON response envelope (`"ok":false,"error":{"code":"E-RPC-011","message":"<err>"}`) when a registered handler's `Fn` returns a non-nil error. The message is the handler's error string verbatim (not wrapped further by `internal/mgmt`). The connection is NOT closed after this response. Distinct from E-RPC-001 (client-side sbctl code) and E-RPC-010 (server unknown command). |

### INT — Internal

| Error Code | Category | Severity | Exit Code | Message Format | FM/DEC Source |
|-----------|----------|----------|-----------|----------------|---------------|
| E-INT-001 | INT | broken | 1 | `"internal error: <operation>: <cause>"` | BC-2.07.001 PC-1; emitted by the `admin.svtn.create` handler (and other admin handlers) in `cmd/switchboard/admin_handlers.go` when `SVTNManager.Create()` or another internal operation returns a non-duplicate, non-authorization error that does not map to a defined E-ADM-* or E-SVTN-* code. `<operation>` is the RPC operation name (e.g. `"admin.svtn.create"`). `<cause>` is the wrapped error string. Distinct from E-SVTN-001 (SVTN already exists), E-ADM-009 (insufficient authority), and E-RPC-011 (generic server handler error). Use this code for unexpected internal failures to avoid masking them behind E-RPC-011. Registered per S-6.07 v1.3 non-duplicate Create() failure code-stamp (Ruling-5 amendment). |
| E-INT-999 | INT | broken | 1 | `"unmapped internal condition, programmer error, please report"` | Ruling-12 §1 universality catch-all sentinel (v4.1). Emitted by the **default arm** of `mapAdminError` (or any equivalent handler-side taxonomy fallback) when a returned error does not match any named sentinel in the INT, ADM, SVTN, CFG, or other family. Existence of this code in production logs is a programmer error — every reachable error path SHOULD map to a named code. Wire envelope follows the standard E-RPC-011 pattern: `{code: "E-RPC-011", message: "E-INT-999: unmapped internal condition, programmer error, please report: <wrapped>"}`. `<wrapped>` is `err.Error()` verbatim. Distinct from E-INT-001 (named internal handler error with operation context). Introducing a new handler that can reach the default arm without first defining a specific code is a defect — see Ruling-12 §7 for the required three-part update process. |

## Failure Mode to Error Code Mapping

| FM-NNN | Failure Mode | Relevant Error Codes |
|--------|-------------|---------------------|
| FM-001 | Single router failure (E phase) | E-NET-002, E-NET-004 |
| FM-002 | All paths degrade | E-NET-002, quality indicator red |
| FM-003 | Frame duplication storm | E-FWD-001 (drop cache metric), operator diagnostic |
| FM-004 | Access node loses tmux control mode | No error code — degradation signal + log |
| FM-005 | Presence message lost/stale | No error code — eventual consistency |
| FM-006 | HMAC verification failure (primitive layer) | E-ADM-002 |
| FM-014 | Wire-layer HMAC mismatch at RouteFrame (tag mismatch from admitted node, or auth key unavailable) | E-ADM-016 (PATH-A auth-key-unavailable and PATH-B tag-mismatch) |
| (no FM) | Per-source HMAC failure rate alert: sustained forgery or misconfiguration from same src_addr | E-ADM-017 |
| FM-007 | Key revocation propagation delay | Acknowledged gap; no error code |
| FM-008 | Quality indicator stuck green | Bug in DI-008 implementation |
| FM-009 | Router crashes without drain | E-NET-004 (detected by nodes) |
| FM-010 | Config error on startup | E-CFG-001, E-CFG-004, E-CFG-005 |
| FM-011 | tmux not present | E-SYS-001 (if PTY also fails); E-SYS-002 (if both tmux and PTY fail at SessionConnector level); log message on PTY fallback |
| (no FM) | PTY shell exits (normal end-of-session or crash) while access node is running | E-SYS-003 (`ErrPTYSourceEOF`) delivered on `sc.Err()`; surfaces to operator as E-SYS-002 format message; triggers PC-2.6 exit-1 path |
| FM-012 | sbctl cannot connect | E-NET-001 |
| FM-013 | Key expired at re-authentication time | E-ADM-015 |
| (no FM) | Forwarding-table miss for (svtnID, dstAddr) — distinct from admission failure | E-FWD-002 |
| (no FM) | SendKeystroke session_name mismatch — caller routed to wrong access node or has stale session state | E-SES-006 |

## Changelog

| Version | Date | Change |
|---------|------|--------|
| v4.2 | 2026-07-02 | Add PENDING-S-7.04 annotation to E-NET-006 row mimicking E-CFG-002 defensive-annotation shape. Documents that emission site is not yet present in `cmd/` or `internal/`; router drain runtime stubbed pending S-7.04. Closes DRIFT-P5P1-B-H001-ENET006-TAXONOMY-ORPHAN, DRIFT-P5P1-B-L001 (annotation-shape inconsistency). Refs Phase 5 Pass 1 Adv-B F-P5-Adv-B-H-001, F-P5-Adv-B-L-001. |
| v4.1 | 2026-07-01 | Ruling-12 §1 universality follow-through: E-INT-999 minted as catch-all default-arm sentinel for `mapAdminError` and equivalent handler-side taxonomy fallbacks. Canonical message: `"unmapped internal condition, programmer error, please report"`. Wire envelope: `{code: "E-RPC-011", message: "E-INT-999: unmapped internal condition, programmer error, please report: <wrapped>"}`. Ruling-12 §7 process policy added to wave-6-tranche-a-scope-rulings.md v1.8: introducing a new handler-code family requires (a) new error-taxonomy row, (b) §Universality row in anchor story spec, (c) amendment to Ruling-12 §1 enumeration — all in the same fix-burst. |
| v4.0 | 2026-07-01 | Wave-6 Tranche A Ruling-4 sibling-propagation (S-6.07 F-P2L3-002): E-ADM-009 FM/DEC Source appended with `, BC-2.07.001 Inv-3` (cross-SVTN control-role key → E-ADM-009 for admin.svtn.create). E-INT-001 minted — `"internal error: <operation>: <cause>"`; source BC-2.07.001 PC-1; registered for non-duplicate Create() failure code-stamp per S-6.07 v1.3 Ruling-5 amendment. INT category added to category table. |
| v3.9 | 2026-06-30 | Pass-22 F-P22L3-002 sibling-fix (4th-iteration narrowing sweep): E-ADM-020 description — BC citation updated v1.10→v1.12; "unconditionally non-revocable at any time" narrowed to "non-revocable for any well-formed request"; E-CFG-001 handler-layer gate note added. E-ADM-021 description — same pattern: v1.10→v1.12; "unconditionally non-expirable at any time" narrowed to "non-expirable for any well-formed request"; E-CFG-001 handler-layer gate note added. Source-of-truth: BC-2.05.004 EC-007 v1.12 (Pass-20 Option-B). |
| v3.8 | 2026-06-30 | E-ADM-021 minted: bootstrap-key-expire-forbidden symmetric counterpart to E-ADM-020 (refs F-P18L1-001 lens-1 pass-18). Sentinel: `svtnmgmt.ErrBootstrapKeyExpireForbidden`. Emitted by `admin.key.expire` handler (mapAdminError arm) when the bootstrap pubkey is targeted. Mirrors revoke protection (EC-004: expire = same lockout effect as revoke). E-ADM-020 description updated to cite BC-2.05.004 EC-007 v1.10 (was v1.9). |
| v3.7 | 2026-06-30 | F-P17L2-001 (MED) + F-P17L2-002 (LOW): E-ADM-020 description rewritten to unconditional phrasing per BC-2.05.004 EC-007 v1.9 ("unconditionally non-revocable at any time, regardless of whether other control keys have been registered"). Canonical message updated from "cannot revoke the last bootstrap key in SVTN <svtn_id>" to "cannot revoke the bootstrap key in SVTN <svtn_id> (permanent trust anchor)". Eliminates false conditionality in both description and message format. |
| v3.6 | 2026-06-30 | F-P14L2-001 (LOW): backfill v3.4 changelog table row (was present in frontmatter modified: list but absent from human-readable Changelog table). No catalog changes. |
| v3.5 | 2026-06-30 | F-P11L2-002 (MED): E-ADM-018 message format — stripped backticks from `--confirm=<svtn-id>` so canonical text, impl, and story are byte-identical: "use --confirm=<svtn-id> to proceed". |
| v3.4 | 2026-06-30 | F-P10L2-004 (MED): E-ADM-018 message format — &lt;svtn-short-id&gt; → &lt;svtn-id&gt; (matches impl and all other ADM messages that take svtn parameter; full svtnName used in impl, not abbreviated form). |
| v3.3 | 2026-06-30 | F-P8L2-003: E-ADM-020 added — `bootstrap-key-revoke-forbidden: cannot revoke the last bootstrap key in SVTN <svtn_id>`; emitted by admin.key.revoke handler when `svtnmgmt.ErrBootstrapKeyRevokeForbidden`; source BC-2.05.004 EC-007. |
| v3.2 | 2026-06-30 | S-6.06 Pass-6 finding F-P6L3-001: E-SVTN-003 added — "SVTN not found: <svtn_id>"; emitted by admin.key.* handler layer when svtnmgmt.ErrSVTNNotFound is returned. Closes missing error code referenced in S-6.06 Error Code Map. |
| v3.1 | 2026-06-30 | S-6.06 Pass-4 ruling F-L2-002: E-ADM-011 scope disambiguation note added — it is a Go API-layer code from `SVTNManager.RevokeKey`/`Destroy`; NOT reachable via `admin.key.*` mgmt RPC path when handler-layer authority gate (E-ADM-009) is wired. Gate fires first; SVTNManager is never invoked for non-control callers on mutating ops. |
| v3.0 | 2026-06-29 | Task 7 reconverge (S-5.01 + S-6.02 Pass-1 adversarial, lens1 F-001): E-ADM-004 KEPT as "address collision" (BC-2.01.006); E-ADM-014 KEPT as "bootstrap key mismatch" (ADR-004 recover). New E-ADM-018 ("control-to-control revocation requires explicit confirmation", `ErrControlRevocationRequiresConfirm`, S-6.02 + ARCH-04 HOLD-001). New E-ADM-019 ("role mismatch: claimed role does not match registered key role", ARCH-04 HOLD-001 cross-check). NOTE: Task spec requested E-ADM-015 + E-ADM-016 but those slots are occupied (E-ADM-015 = key expired, E-ADM-016 = wire HMAC mismatch); new entries use next free slots E-ADM-018 and E-ADM-019 per append-only-numbering policy. |
| v2.9 | 2026-06-29 | S-6.05 PO ruling: E-ADM-011 extended with Variant 2 (destroy authorization). New `ErrDestroyUnauthorized` sentinel maps to E-ADM-011; message format: `"permission denied: <role> key cannot destroy SVTN <svtn_name>"`. No new code slot allocated — destroy authorization is the same class of error as revocation hierarchy violation (insufficient privilege for admission-plane operation requiring control authority). BC-2.07.001 Inv-3 source. |
| v2.8 | 2026-06-29 | ARCH-12 v1.5 Wave-5 Convergence Ruling Y: E-NET-001 scope extended to cover two explicit cases: (a) `net.Dial`/`net.DialContext` failure (existing); (b) handshake read-deadline timeout — daemon accepted TCP connection but did not complete ADR-012 handshake within timeout budget; treated as unreachable per BC-2.07.003 Inv-2; message format for case (b): `"daemon unreachable: <address>: connection timed out"`. Reconciles BC-2.07.003 Inv-2 vs Inv-4 conflict (Ruling Y). |
| v2.7 | 2026-06-29 | ARCH-12 v1.4 Wave-5 Convergence Ruling L: E-CFG-008 Variant 2 canonical message string corrected — `buildMgmtListener` embeds the error code as prefix: `"E-CFG-008: management_socket: console mode requires a loopback address (127.0.0.1, [::1], or localhost); got: <address>"`. Variant 1 (`config.Validate()`) does NOT embed the code in the string (different error-reporting path). Both variants distinguished clearly. Test assertion guidance added: `strings.Contains(err.Error(), "E-CFG-008")` required (not full-string match). VP-073 property description updated to cite canonical `buildMgmtListener` format. |
| v2.6 | 2026-06-29 | ARCH-12 v1.3 Wave-5 Convergence Rulings C and D: (1) E-RPC-010 added — server-side unknown-command in-band response (`internal/mgmt`); exit code "—" (in-band); connection not closed; BC-2.07.004 PC-12. (2) E-RPC-011 added — server-side handler-error in-band response; message is handler error string verbatim; connection not closed; BC-2.07.004 PC-13. (3) E-RPC-001 clarified as CLIENT-SIDE ONLY (`cmd/sbctl` stderr); must not appear in `internal/mgmt`; E-RPC-002 forbidden. (4) E-CFG-008 row extended with Variant 2 (console TCP loopback rejection) — message: "config error: management_socket: console mode requires a loopback address (127.0.0.1, [::1], or localhost); got: \<address\>"; emitted by `buildMgmtListener`, not `config.Validate()`; BC-2.07.004 EC-013 (Ruling D). |
| v2.5 | 2026-06-28 | Wave-5 mgmt-plane adversarial review Rulings 1/3/4/5/6/7 (ARCH-12 v1.2): (1) E-CFG-010 added (CFG table) — sbctl key load failure (absent/oversized/malformed/wrong-type key at `--key` path), exit 1, message "key load failed: \<path\>: \<reason\>", no connection attempt made; free slot in CFG family. (2) RPC category added to category table; E-RPC-001 added in new RPC section — post-auth dispatch failure after AUTH_OK, exit 1, message "rpc failed: \<command\>: \<reason\>"; opens RPC family. (3) E-NET-001 row updated with scope-clarification note: strictly dial/connect-unreachable; key-load failures → E-CFG-010; post-auth failures → E-RPC-001. |
| v2.4 | 2026-06-28 | Wave-5 consistency audit F-003: E-CFG-002 collision flag added. BC-2.09.003 v1.2 uses E-CFG-002 for listen_addr invalid host:port; this taxonomy defines E-CFG-002 as "private key export not supported" (BC-2.05.007). Pre-existing collision now explicitly surfaced (the E-CFG-006 collision was already flagged; E-CFG-002 was not). Reconciliation deferred to maintenance pass; no renumbering in this pass. |
| v2.3 | 2026-06-28 | Wave-5 management plane: (1) E-ADM-010 note extended — now explicitly cites BC-2.07.004 (server-side) in addition to BC-2.07.002 (client-side); establishes E-ADM-010 as the canonical operator-auth-failure code for both sbctl and internal/mgmt; anti-oracle note added (identical message for unrecognized key and wrong signature). (2) E-CFG-008 added (management_socket present-and-blank; BC-2.09.003 PC-10). (3) E-CFG-009 added (authorized_operator_keys malformed PEM; BC-2.09.003 PC-11). (4) Collision flag added for pre-existing E-CFG-006 discrepancy (taxonomy: sbctl admin flag; BC-2.09.003 v1.4: drain_timeout negative) — reconciliation deferred to maintenance pass. EC-002 error-code bug in S-6.03 noted: story EC-002 cites E-ADM-001 (SVTN node admission failure); correct code is E-ADM-010 (operator-auth-failure); story-writer to fix S-6.03 EC-002. |
| v2.2 | 2026-06-27 | Wave-3 consistency audit F-1.2 (E-ADM-016 PATH-A variant documented) and F-1.3 (E-ADM-007 layering-note extended for static session-layer sentinel). E-ADM-016 now documents both PATH-A (auth key unavailable, routing.go:200) and PATH-B (tag mismatch, routing.go:215). E-ADM-007 layering note now explicitly acknowledges ErrUpstreamReadOnly as a static errors.New sentinel with parametric context added by callers via %w-wrapping. FM-014 mapping updated to reference both E-ADM-016 paths. |
| v2.1 | 2026-06-27 | (HIGH-A) Added E-SYS-003 (`ErrPTYSourceEOF`): sentinel for PTY-source EOF mid-session detected by forwardFrames relay (ARCH-01 ADR-011 v1.5 §HIGH-A). Updated E-SYS-002 note to clarify it is also the wire-visible message format for E-SYS-003 events. Per S-W3.04 adversarial convergence pass-2. |
| v2.0 | 2026-06-27 | Align E-ADM-017 re-fire annotation with drain-only re-arm (BC-2.05.005 v1.6 PC-3, VP-059 v1.1 property (c), S-W3.05 AC-004/AC-016): replaced stale "re-fires when the oldest surviving in-window entry is newer than the last-fire timestamp" prose with drain-only re-arm + append-skip semantics; message format unchanged. |
| v1.9 | 2026-06-27 | Per-story adversarial convergence adjudication: (1) E-ADM-016 message format updated — added "(E-ADM-016)" literal suffix for grep-ability; clarified `<src_addr>` is lowercase hex of 8-byte SrcAddr. (2) E-ADM-017 message format updated — parameterized `<threshold>` and `<window_seconds>` (not hardcoded ≥5/60s); added "E-ADM-017" literal prefix; clarified `<src_addr>` hex rendering; added re-fire semantics description (periodic re-arm under sustained attack). Closes HF-1 (hysteresis), item-3 (format + src_addr rendering) adversary findings. |
| v1.8 | 2026-06-27 | Added E-SYS-002 (SessionConnector aggregate connect failure: both tmux ctrl and PTY exhausted, exit 1). Registered per BC-2.04.007 authorship (daemon lifecycle contract). Updated FM-011 mapping row. |
| v1.7 | 2026-06-27 | Added E-ADM-017 (aggregate HMAC failure rate alert: ≥5 failures in 60s from same src_addr, severity=degraded, emitted by admission.FailureCounter); updated E-ADM-016 note to distinguish from E-ADM-017; added FM mapping row for E-ADM-017. Closes Wave 3 gate F-2 (product-owner adjudication: FIX-NOW, not deferred). |
| v1.6 | 2026-06-26 | Added layering notes to E-ADM-006 and E-ADM-007: clarifies that at the internal/session layer `ConsoleKey` serves as `<key_fingerprint>` and `<node_addr>` is legitimately omitted (session layer has no node identity per ARCH-08 §6.6); transport/admission boundary supplies `<node_addr>` when re-surfacing. No behavioral change; closes recurring M-1 adversarial finding from S-3.03 passes. |
| v1.5 | 2026-06-26 | Added E-ADM-016 (wire HMAC mismatch at RouteFrame), E-SES-004, E-SES-005 (RETIRED), E-SES-006; retired E-SES-005 duplicate; added namespace-aliases note for E-FRM-* → E-PRT-*; added FM-014 mapping; updated E-SES-002 and E-SES-003 anchors. |
