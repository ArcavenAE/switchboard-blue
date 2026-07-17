---
artifact_id: BC-2.09.003
document_type: behavioral-contract
level: L3
version: "2.2"
status: draft
producer: product-owner
timestamp: 2026-06-28T00:00:00
phase: 1a
inputs:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/domain-spec/failure-modes.md'
  - '_bmad-output/planning-artifacts/prd.md'
input-hash: "5da12a8"
extracted_from: null
bc_id: BC-2.09.003
subsystem: SS-09
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
    version: "2.1"
    change: >
      Identity-cluster BC groundwork consolidated amendment (items N3+A3+A4):
      PC-12 added (admission_key_file: non-empty when present, no I/O in Validate, E-CFG-014);
      PC-13 added (admission_state_file: non-empty when present, parent-dir stat when set, E-CFG-015);
      PC-14 added (router_management_endpoints: each addr validated as host:port per E-CFG-003
      pattern, E-CFG-016, NO loopback restriction per Ruling 9, control-mode-only field);
      E-CFG-014/015/016 added to Error Codes table; EC-015 through EC-020 added;
      new Canonical Test Vectors for all three fields; test-as-evidence VP rows added.
      Added inputs/input-hash/extracted_from frontmatter fields (template conformance).
  - date: 2026-07-06
    version: "2.0"
    change: >
      Governance-only: narrow Verification Properties table to correct pre-existing
      authoring drift identified in S-6.04 disposition ruling (S-6.04-disposition-ruling.md
      §"Spec Note: BC-2.09.003 Verification Properties Table Drift"). VP-028 and VP-029 are
      strictly scoped to Config.Validate startup-validation behavior (out-of-range
      tick_interval; missing required fields listen_addr/tick_interval). They do NOT cover
      host:port parsing, drain/keepalive validation, management_socket, authorized_operator_keys,
      config application to subsystems, or the SIGHUP reload integration path. Table rows
      corrected: only the two startup-validation properties covered by VP-028/VP-029 retain
      those citations; all other rows now cite "test-as-evidence" with the appropriate owning
      story/AC. Reload-integration fail-closed property (Inv-3/EC-004) annotated as covered
      by S-7.04-FU-SIGHUP-RELOAD AC-002 integration test (no formal VP assigned). No PC/EC/Inv
      semantic changes; no runtime behavior implied. Governance-leaf change per PO ruling path.
  - date: 2026-07-02
    version: "1.9"
    change: >
      Phase 5 Pass 3 remediation Path B follow-up: error-taxonomy v4.3 reconciled
      E-CFG-002 and E-CFG-006 collisions (private-key-export → E-CFG-011; sbctl
      --yes → E-CFG-012). Collision-flag annotation row in the Error Codes table
      removed — the pre-existing inconsistency has been resolved in error-taxonomy
      v4.3, so the flag no longer applies. Closes DRIFT-P5P3-A007-ECFG-COLLISION
      (BC-2.09.003 side). Refs F-P5P3-A-007.
  - date: 2026-07-02
    version: "1.8"
    change: >
      Reconcile listen_addr DEFERRED-APPLICATION row — previously said "No current
      owner story — a network-listener introduction story is needed (flagged for
      STORY-INDEX)" contradicting STORY-INDEX line 135 which already lists S-BL.NI
      as the owner. Row now correctly names S-BL.NI. Closes Phase 5 Pass 2
      F-P5P2-B-002.
  - date: 2026-06-28
    version: "1.7"
    change: >
      Traceability refresh (Wave-5 consistency audit F-002): Stories field updated
      to add S-W5.01 alongside S-6.01. S-W5.01 implements PC-10 (management_socket
      validation, E-CFG-008, AC-011) and PC-11 (authorized_operator_keys PEM
      validation, E-CFG-009, AC-012) added in v1.6. S-6.01 retains ownership of
      AC-001 through AC-009 (all earlier postconditions). E-CFG-002 collision flag
      added: error-taxonomy.md defines E-CFG-002 as "private key export not
      supported" (BC-2.05.007), but this BC (v1.2) uses E-CFG-002 for listen_addr
      invalid host:port validation. Pre-existing inconsistency flagged for
      maintenance-pass resolution; no renumbering in this pass.
  - date: 2026-06-28
    version: "1.6"
    change: >
      Wave-5 management plane config additions (ARCH-12): added PC-10 (management_socket
      validation: non-empty path, not whitespace-only; E-CFG-008) and PC-11
      (authorized_operator_keys: each entry must be a valid PEM PUBLIC KEY block
      containing a 32-byte Ed25519 key; E-CFG-009). Added error codes E-CFG-008 and
      E-CFG-009 to Error Codes table. Added edge cases EC-011 (management_socket
      whitespace) and EC-012 (malformed PEM entry in authorized_operator_keys).
      Updated Canonical Test Vectors with two new rows. Config-schema impact note
      added: interface-definitions.md §Config Schema requires two new fields
      (management_socket: string, authorized_operator_keys: []string). E-CFG collision
      flag: error-taxonomy.md E-CFG-006 is "sbctl admin --yes/--confirm conflict"
      (sbctl flag validation), but BC-2.09.003 v1.4 uses E-CFG-006 for drain_timeout
      negative. This is a pre-existing inconsistency in the error taxonomy that
      predates this pass. The new codes E-CFG-008 and E-CFG-009 are free in both
      documents. Taxonomy reconciliation is flagged for a dedicated maintenance pass.
  - date: 2026-06-28
    version: "1.5"
    change: >
      Traceability refresh (KNOWN-STALE audit): Stories/Story-Anchor rows updated from
      "AC-001 through AC-006" to "AC-001 through AC-009" to reflect the S-6.01 v1.5
      expansion (SP-003/SP-004/SP-005 added AC-007, AC-008, AC-009; the BC's PC-5..PC-9
      already defined the postconditions those ACs trace to). EC canonical numbering
      affirmed as authoritative (BC EC-NNN is source of truth per VSDD); story-writer
      must reconcile story S-6.01 EC IDs to match BC EC IDs (drift item S601-NITPICK-B):
      story EC-012 should be EC-009, story EC-011 should be EC-008 (drain_timeout:0s
      accepted), story EC-013 should be EC-010. Story EC numbering is cosmetic drift only
      — no behavior change.
  - date: 2026-06-28
    version: "1.2"
    change: >
      S-6.01 scope expansion: (a) deep field validation postconditions added
      (PC-5 through PC-8) — listen_addr host:port parse, upstream_routers
      host:port parse, DrainTimeout/KeepaliveInterval positive-value
      enforcement; new error codes E-CFG-002, E-CFG-003, E-CFG-006, E-CFG-007;
      (b) config-application postcondition added (PC-9) — daemon MUST use
      the validated config struct to configure subsystems, not discarded values
      or hardcoded constants. Edge cases EC-005 through EC-009 added.
  - date: 2026-06-28
    version: "1.3"
    change: >
      Right-sized PC-9 and Inv-5 per fresh-eyes verification (2026-06-28) and
      human ruling "apply what exists now, track the rest as concrete
      dependencies." PC-9 now requires application ONLY of tick_interval, whose
      target subsystem (halfchannel.New tick cadence in cmd/switchboard/access.go)
      exists today. listen_addr binding, drain_timeout, and upstream_routers
      application are deferred with named owning stories (listener introduction:
      no current owner — flagged; drain/PE: S-7.04). DEFERRED-APPLICATION note
      added. Inv-5 narrowed to "applicable fields" so legitimately-deferred
      fields do not constitute a violation. EC-010 and the PC-9 canonical test
      vector updated to match.
  - date: 2026-06-28
    version: "1.4"
    change: >
      Resolved 3-way contradiction (BC PC-7/PC-8 vs. config.go implementation vs.
      ARCH-06 defaults) per human ruling "optional with defaults, align to ARCH-06."
      PC-7 and PC-8 now specify drain_timeout and keepalive_interval as OPTIONAL
      fields: Validate() rejects ONLY a negative value (E-CFG-006 / E-CFG-007);
      zero or absent is accepted and means "use daemon default" (10s / 1s per
      ARCH-06). E-CFG-006 and E-CFG-007 trigger conditions and message templates
      updated from "zero or negative" / "must be > 0" to "negative" / "must be
      >= 0 (use 0 to apply daemon default)." EC-008 corrected: drain_timeout: 0s
      is now ACCEPTED (daemon default 10s). Canonical test vector for drain_timeout:
      0s updated to reflect accepted behaviour. Default application remains deferred
      to S-7.04 (DEFERRED-APPLICATION note unchanged).
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
traces_to: [CAP-028]
kos_anchors:
  - elem-single-binary-three-modes
---

# Behavioral Contract BC-2.09.003: Router Startup Fails Cleanly on Malformed Config with Actionable Error Message; Validated Config Is Applied to Applicable Subsystems

## Description

When the router daemon starts with a malformed, incomplete, or invalid configuration file, it exits immediately with a non-zero exit code and prints a clear, actionable error message identifying the specific problem (field name, line number, value). The daemon does not start in a partially-configured state. No sessions are affected (the daemon was not running). When startup succeeds, the daemon MUST use the validated config struct to configure all subsystems whose implementation exists — it MUST NOT fall back to hardcoded defaults or discard the validated config for fields whose target subsystems are built. Fields whose target subsystems are not yet implemented are validated but their application is explicitly deferred (see DEFERRED-APPLICATION note in PC-9).

## Preconditions

1. The router daemon process is starting.
2. A `--config <path>` flag (or equivalent) has been supplied.
3. The configuration file exists.

## Postconditions

### Failure path postconditions (any validation error)

1. The daemon exits with a non-zero exit code before accepting any connections.
2. stderr contains at least one error message in E-CFG-001 format: `"config error: <field>: <problem>. Fix: <suggestion>"`.
3. stdout is empty.
4. No leftover state, lock files, or partial network bindings.

### Deep field validation postconditions (v1.2 additions)

5. `listen_addr` is parsed as a valid `host:port` (net.ResolveTCPAddr or equivalent); if invalid, exits with E-CFG-002: `"config error: listen_addr: '<value>' is not a valid host:port. Fix: use '<ip>:<port>' format, e.g. '0.0.0.0:9090'"`.
6. Each entry in `upstream_routers[].addr` is parsed as a valid `host:port`; if any entry is invalid, exits with E-CFG-003: `"config error: upstream_routers[<N>].addr: '<value>' is not a valid host:port. Fix: use '<ip>:<port>' format, e.g. '10.0.0.1:9090'"`.
7. `drain_timeout` is an **optional** config field. When absent or zero (Go yaml unmarshalling cannot distinguish absent from explicit-zero for `time.Duration` without a pointer — zero is treated as absent), Validate() **accepts** the value; the daemon applies the documented default of **10s** at application time (deferred to S-7.04; see DEFERRED-APPLICATION note in PC-9). When present and **negative**, Validate() exits with E-CFG-006: `"config error: drain_timeout: must not be negative; got '<value>'. Fix: remove the field to use the daemon default (10s), or set to a positive duration, e.g. '10s'"`. Cross-reference: ARCH-06 §Graceful Drain documents the 10s default.
8. `keepalive_interval` is an **optional** config field. When absent or zero (same Go yaml / `time.Duration` zero-value semantics as PC-7), Validate() **accepts** the value; the daemon applies the documented default of **1s** at application time (deferred to S-7.04; see DEFERRED-APPLICATION note in PC-9). When present and **negative**, Validate() exits with E-CFG-007: `"config error: keepalive_interval: must not be negative; got '<value>'. Fix: remove the field to use the daemon default (1s), or set to a positive duration, e.g. '1s'"`. Cross-reference: ARCH-06 §Graceful Drain (FM-009) documents the 1s default.

### Management plane config field validation postconditions (v1.6 additions)

10. `management_socket` is an **optional** config field (daemon mode startup code applies
    mode-specific defaults when absent; see ARCH-05 §Daemon Management Socket). When
    **present**, it must be a non-empty string that is not entirely whitespace. If
    present and blank or whitespace-only, Validate() exits with E-CFG-008:
    `"config error: management_socket: must not be empty. Fix: set to a valid Unix socket path, e.g. '/run/switchboard-router.sock', or remove the field to use the daemon default"`.
    A non-whitespace string is accepted at validation time regardless of whether the
    path is reachable — path accessibility is a runtime concern, not a config-validation
    concern.

11. `authorized_operator_keys` is an **optional** config field (empty list = bootstrap
    mode: daemon's own key is the authorized operator key per ADR-012 §2). When
    **present and non-empty**, each entry must be a valid PEM block of type `"PUBLIC KEY"`
    (PKIX/SubjectPublicKeyInfo encoding) containing a 32-byte Ed25519 public key. For
    each entry that fails this validation, Validate() records an error in E-CFG-009
    format (all errors collected before exit, exhaustive reporting):
    `"config error: authorized_operator_keys[<N>]: entry is not a valid Ed25519 PEM PUBLIC KEY block. Fix: provide a PEM-encoded Ed25519 public key (type 'PUBLIC KEY', 32-byte key length)"`.
    An empty list (`authorized_operator_keys: []` or field absent) is accepted.

    **Config-schema impact note (for interface-definitions.md and story-writer):**
    Two new fields are added to `internal/config.Config`:
    - `management_socket string` (yaml: `management_socket`) — optional, validated by PC-10.
    - `authorized_operator_keys []string` (yaml: `authorized_operator_keys`) — optional,
      validated by PC-11.
    The YAML config schema section in interface-definitions.md §Config Schema must be
    updated to document these fields. This is flagged as a story-writer / implementer
    responsibility for S-W5.01.

11b. **Mgmt listener TCP loopback restriction (Ruling 12, v2.2):** When `management_socket`
    resolves to a TCP `host:port` (i.e., `net.SplitHostPort` succeeds), the daemon's
    `buildMgmtListener` MUST enforce a loopback-only restriction for **console, control, and
    access** modes: a non-loopback host (anything other than `127.0.0.1`, `[::1]`, or
    `localhost`) causes daemon startup to fail with **E-CFG-008**:
    `"config error: management_socket: <mode> mode requires a loopback address (127.0.0.1, [::1], or localhost); got: <value>"`.
    A loopback TCP address (e.g., `127.0.0.1:9091`) is accepted. On successful TCP bind,
    the daemon MUST emit an INFO log: `"<mode> management listener bound to <address>"`.
    **Router mode is the sole exemption** (Ruling 9, unchanged): control→router push is
    inherently cross-host; ADR-012 challenge-response authentication is the security
    boundary; the loopback restriction does NOT apply to router-mode TCP management
    listeners. This postcondition replaces the prior console-only loopback guard scope —
    previously `buildMgmtListener` checked `if mode == "console"`; the correct check is
    `if mode != "router"`.

    **Scope note:** The loopback restriction is enforced at listener bind time (daemon startup),
    not at `Config.Validate()` time — binding behavior cannot be validated without actually
    attempting to parse and bind the address. This postcondition is a daemon-startup concern
    orthogonal to the blank-string validation in PC-10.

### Identity-cluster config field validation postconditions (v2.1 additions; extended in v2.2)

12. `admission_key_file` is an **optional** config field (access-mode daemon applies the
    default path `/var/lib/switchboard/access-admission-identity.pem` when absent; see
    BC-2.09.004 §Keypair provisioning). When **present**, it must be a non-empty string
    that is not entirely whitespace. If present and blank or whitespace-only, Validate()
    exits with E-CFG-014:
    `"config error: admission_key_file: must not be empty. Fix: set to a valid file path, e.g. '/var/lib/switchboard/access-admission-identity.pem', or remove the field to use the daemon default"`.
    A non-whitespace string is accepted at validation time regardless of whether the path
    exists on disk — file accessibility is a daemon-startup concern, not a
    config-validation concern (ARCH-06 §Config purity). PC-12 is **access-mode only**;
    other modes do not use this field. Validate() does not enforce mode restrictions at
    parse time.

    **Config-schema impact note:** A new field is added to `internal/config.Config`:
    - `admission_key_file string` (yaml: `admission_key_file`) — optional, validated by PC-12.
    This is a S-BL.NODE-ADMISSION-PROVISIONING implementer responsibility.

13. `admission_state_file` is an **optional** config field (router-mode daemon starts with
    empty keyset when absent; see BC-2.05.010 §Load-on-startup). When **present**, it must
    be a non-empty string that is not entirely whitespace. If present and blank or
    whitespace-only, Validate() exits with E-CFG-015:
    `"config error: admission_state_file: must not be empty. Fix: set to a valid writable file path, e.g. '/var/lib/switchboard/admission-state.json', or remove the field to start with an empty keyset"`.
    A non-whitespace string is accepted at validation time; the parent directory
    accessibility check is a daemon-startup concern, not a config-validation concern.
    PC-13 is **router-mode only**; other modes do not use this field.

    **Config-schema impact note:** A new field is added to `internal/config.Config`:
    - `admission_state_file string` (yaml: `admission_state_file`) — optional, validated by PC-13.
    This is a S-BL.ADMISSION-SYNC-WIRE implementer responsibility.

15. `control_admission_state_file` is an **optional** config field, **control-mode only**.
    When **present**, it must be a non-empty string that is not entirely whitespace. If present
    and blank or whitespace-only, Validate() exits with E-CFG-017:
    `"config error: control_admission_state_file: must not be empty. Fix: set to a valid writable file path, e.g. '/var/lib/switchboard/control-admission-state.json', or remove the field to disable control-side persistence"`.
    A non-whitespace string is accepted at validation time; path accessibility and parent-directory
    existence are daemon-startup concerns, not config-validation concerns (ARCH-06 §Config purity).
    PC-15 is **control-mode only**; router/console/access modes do not read this field.
    When absent or empty, control does not persist admission state and `PushFullSnapshot` on
    startup pushes an empty keyset — the EC-007 resync guarantee (BC-2.05.009) does NOT apply.
    Operators who require EC-007 MUST configure this field. No file I/O in `Validate()`.

    **Config-schema impact note:** A new field is added to `internal/config.Config`:
    ```go
    // ControlAdmissionStateFile is the path where the control-mode daemon persists its
    // authoritative AdmittedKeySet (S-BL.ADMISSION-SYNC-WIRE Ruling 11).
    ControlAdmissionStateFile string `yaml:"control_admission_state_file"`
    ```
    This is a S-BL.ADMISSION-SYNC-WIRE implementer responsibility.

14. `router_management_endpoints` is an **optional** config field (empty list means no
    admission-state push replication; see BC-2.05.009). When **present and non-empty**,
    each entry's `addr` field must be a valid `host:port` string. For each entry whose
    `addr` fails `validateHostPort`, Validate() records an error in E-CFG-016 format
    (exhaustive reporting — all errors collected before exit):
    `"config error: router_management_endpoints[<N>].addr: '<value>' is not a valid host:port. Fix: use '<ip>:<port>' or '<hostname>:<port>' format, e.g. '10.0.0.2:9093'"`.
    **NO loopback restriction:** any valid `host:port` is accepted, including
    `0.0.0.0:PORT` and non-loopback addresses. The ADR-012 challenge-response handshake
    is the authentication boundary; network-level access restriction is the operator's
    firewall-policy responsibility (Ruling 9, S-BL.ADMISSION-SYNC-WIRE-rulings.md).
    An empty list (`router_management_endpoints: []` or field absent) is accepted.
    PC-14 is **control-mode only**; router/console/access modes do not read this field.

    **Config-schema impact note:** A new field is added to `internal/config.Config`:
    ```go
    RouterManagementEndpoints []RouterManagementEndpoint `yaml:"router_management_endpoints"`
    type RouterManagementEndpoint struct { Addr string `yaml:"addr"` }
    ```
    This is structurally identical to the existing `UpstreamRouters []UpstreamRouter` pattern.
    This is a S-BL.ADMISSION-SYNC-WIRE implementer responsibility.

### Config application postcondition (v1.3 right-sized)

9. When `--config` is supplied and validation passes, the daemon initializes all subsystems using the validated config struct for fields whose target subsystems exist today. Specifically, the following field IS applied immediately:

   - **`tick_interval`** — the half-channel tick cadence is sourced from `cfg.TickInterval` (passed to `halfchannel.New`), not the hardcoded `10ms` default. (Target subsystem: `internal/halfchannel` via `cmd/switchboard/access.go`. Exists on develop.)

   The daemon MUST NOT silently ignore `tick_interval` and fall back to the hardcoded `10*time.Millisecond` constant when a config file is supplied.

   **DEFERRED-APPLICATION note (v1.3):** The following config fields are fully VALIDATED by PC-5 through PC-8 but their APPLICATION to daemon subsystems is explicitly deferred because the target subsystems do not yet exist on the develop branch. This is not a spec gap — it is a tracked forward dependency. Each deferred field is owned by a named story:

   | Field | Reason for Deferral | Owning Story / Flag |
   |-------|---------------------|---------------------|
   | `listen_addr` | No TCP listener (`net.Listen` / `.Accept`) exists anywhere in the codebase; the daemon is a PTY/tmux relay with no network-ingress listener today. | **S-BL.NI (backlog, network-ingress listener).** S-BL.NI is the owning story per STORY-INDEX Backlog table row for `S-BL.NI`: "Also owns cfg.ListenAddr application — must wire `cfg.ListenAddr` to `net.Listen`/`.Accept` at this story's implementation time (BC-2.09.003 PC-9 DEFERRED-APPLICATION; S-6.01 v1.4 deferred listen_addr binding depends on this story)." The application point is `net.Listen(cfg.ListenAddr)` inside the new `internal/listener` (or equivalent) subsystem this story introduces. |
   | `drain_timeout` | `internal/drain` does not exist on develop. | S-7.04 (Wave 7, PE graduation and graceful drain). **S-7.04 obligation (L-5):** Must wire `cfg.DrainTimeout` into the graceful-drain coordinator (the `internal/drain` package it introduces), replacing any hardcoded drain-timeout constant. The application point is the drain sequence initiated on graceful shutdown / PE graduation, specifically the deadline passed to the drain context or equivalent timeout mechanism. Default: 10s per ARCH-06 §Graceful Drain when field is zero/absent. |
   | `upstream_routers` | PE-mode upstream connection logic is owned by the PE graduation work; `internal/drain` does not exist. | S-7.04 (Wave 7). **S-7.04 obligation (L-5):** Must wire `cfg.UpstreamRouters` (slice of `{addr string}`) into the PE-mode upstream connector, replacing any hardcoded upstream address list. The application point is the upstream connection pool / peer-list initialization in the PE graduation path. |
   | `keepalive_interval` | The `sweepDeadline` constant (60s, console eviction window) is architecturally distinct from the node reconnect keepalive interval described by FM-009 ("after `keepalive_interval`, default 1s, nodes attempt reconnect"). Wiring `cfg.KeepaliveInterval` to `sweepDeadline` would misrepresent their semantics. The correct keepalive mechanism is part of the drain/node-keepalive subsystem work. | S-7.04 (Wave 7). **S-7.04 obligation (L-5):** Must wire `cfg.KeepaliveInterval` into the node-reconnect keepalive ticker (the mechanism described by FM-009), replacing any hardcoded 1s constant or equivalent. The application point is the keepalive goroutine / ticker that fires reconnect attempts from a drained or disconnected PE node. Default: 1s per ARCH-06 §Graceful Drain (FM-009) when field is zero/absent. MUST NOT wire to `sweepDeadline` (console eviction window — semantically unrelated). |

   These deferred fields are validated at startup (PC-6, PC-7, PC-8) — a bad value still causes an actionable error and exit 1. Only their APPLICATION to running subsystems is deferred.

## Invariants

1. No daemon starts in a degraded-config state — it's all-or-nothing.
2. Error messages name the specific field (and index for array fields) and provide a fix suggestion.
3. This applies equally to initial startup and config reload (SIGHUP): a bad config reload leaves the daemon running on the previous config.
4. All validation errors are collected and reported together (exhaustive reporting), not just the first.
5. The validated config is the single source of truth for subsystem configuration of **applicable fields** — those whose target subsystem exists on develop. Hardcoded fallback values for applicable fields are prohibited when a config file is supplied. Fields whose target subsystems are not yet built (see DEFERRED-APPLICATION note in PC-9) are excluded from this invariant until their owning stories deliver the subsystem.

## Trigger

Daemon startup config parsing failure; config reload with invalid config.

## Error Codes

| Code | Condition | Severity | Exit Code | Message Template |
|------|-----------|----------|-----------|-----------------|
| E-CFG-001 | Required field missing or generic validation failure | broken | 1 | `"config error: <field>: <problem>. Fix: <suggestion>"` |
| E-CFG-002 | `listen_addr` is not a valid `host:port` | broken | 1 | `"config error: listen_addr: '<value>' is not a valid host:port. Fix: use '<ip>:<port>' format, e.g. '0.0.0.0:9090'"` |
| E-CFG-003 | `upstream_routers[N].addr` is not a valid `host:port` | broken | 1 | `"config error: upstream_routers[<N>].addr: '<value>' is not a valid host:port. Fix: use '<ip>:<port>' format, e.g. '10.0.0.1:9090'"` |
| E-CFG-004 | Config file not found at the supplied path | broken | 1 | `"config file not found: <path>"` |
| E-CFG-005 | Config file present but malformed YAML (syntax error) | broken | 1 | `"config parse error: invalid YAML at line <N>: <detail>"` |
| E-CFG-006 | `drain_timeout` is present and negative | broken | 1 | `"config error: drain_timeout: must not be negative; got '<value>'. Fix: remove the field to use the daemon default (10s), or set to a positive duration, e.g. '10s'"` |
| E-CFG-007 | `keepalive_interval` is present and negative | broken | 1 | `"config error: keepalive_interval: must not be negative; got '<value>'. Fix: remove the field to use the daemon default (1s), or set to a positive duration, e.g. '1s'"` |
| E-CFG-008 | `management_socket` is present but empty or whitespace-only | broken | 1 | `"config error: management_socket: must not be empty. Fix: set to a valid Unix socket path, e.g. '/run/switchboard-router.sock', or remove the field to use the daemon default"` |
| E-CFG-009 | `authorized_operator_keys[N]` is not a valid PEM PUBLIC KEY block containing an Ed25519 key | broken | 1 | `"config error: authorized_operator_keys[<N>]: entry is not a valid Ed25519 PEM PUBLIC KEY block. Fix: provide a PEM-encoded Ed25519 public key (type 'PUBLIC KEY', 32-byte key length)"` |
| E-CFG-014 | `admission_key_file` is present but empty or whitespace-only | broken | 1 | `"config error: admission_key_file: must not be empty. Fix: set to a valid file path, e.g. '/var/lib/switchboard/access-admission-identity.pem', or remove the field to use the daemon default"` |
| E-CFG-015 | `admission_state_file` is present but empty or whitespace-only | broken | 1 | `"config error: admission_state_file: must not be empty. Fix: set to a valid writable file path, e.g. '/var/lib/switchboard/admission-state.json', or remove the field to start with an empty keyset"` |
| E-CFG-016 | `router_management_endpoints[N].addr` is not a valid `host:port` | broken | 1 | `"config error: router_management_endpoints[<N>].addr: '<value>' is not a valid host:port. Fix: use '<ip>:<port>' or '<hostname>:<port>' format, e.g. '10.0.0.2:9093'"` |
| E-CFG-017 | `control_admission_state_file` is present but empty or whitespace-only | broken | 1 | `"config error: control_admission_state_file: must not be empty. Fix: set to a valid writable file path, e.g. '/var/lib/switchboard/control-admission-state.json', or remove the field to disable control-side persistence"` |

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Config file missing entirely | E-CFG-004 "config file not found: <path>"; exit 1. |
| EC-002 | Config file present but empty | E-CFG-001 "config error: required field 'listen_addr' missing"; exit 1. |
| EC-003 (FM-010) | Malformed YAML (syntax error) | E-CFG-005 "config parse error: invalid YAML at line N: <detail>"; exit 1. |
| EC-004 | Config reload (SIGHUP) with bad new config | Daemon logs: "config reload failed: <error>; continuing with previous config". Previous config remains active. |
| EC-005 | `listen_addr` present but missing port (e.g. `"0.0.0.0"`) | E-CFG-002 with value `"0.0.0.0"`; exit 1. |
| EC-006 | `listen_addr` with non-numeric port (e.g. `"0.0.0.0:notaport"`) | E-CFG-002 with value `"0.0.0.0:notaport"`; exit 1. |
| EC-007 | `upstream_routers` has two entries; first is valid, second is invalid | E-CFG-003 naming index 1 (0-based); all errors collected before exit 1 (exhaustive reporting). |
| EC-008 | `drain_timeout: 0s` (or field absent) | Validate() **accepts** the value (zero == absent per Go yaml / `time.Duration` zero-value semantics). Daemon applies default 10s at application time (S-7.04). No error; daemon starts normally. |
| EC-009 | `keepalive_interval: -1s` | E-CFG-007 with value `"-1s"`; exit 1. |
| EC-010 | Config file supplied and valid; daemon starts | PC-9 (v1.3): `tick_interval` from config is passed to `halfchannel.New` (not the hardcoded `10ms` fallback). `listen_addr`, `drain_timeout`, `keepalive_interval`, and `upstream_routers` application deferred to their owning stories (see DEFERRED-APPLICATION note). |
| EC-011 | `management_socket: "   "` (whitespace-only value) | E-CFG-008 "config error: management_socket: must not be empty..."; exit 1. Exhaustive error collection: if other errors also exist, all are reported together. |
| EC-012 | `authorized_operator_keys: ["not-pem-data", "-----BEGIN PUBLIC KEY-----\nAAA\n-----END PUBLIC KEY-----\n"]` (first entry invalid, second invalid key type/length) | E-CFG-009 reported for each invalid entry with its index (0-based). All errors collected before exit 1. |
| EC-013 | `authorized_operator_keys: []` (empty list) | Validate() accepts; daemon starts in bootstrap mode (daemon's own key is the authorized operator key). No error. |
| EC-014 | `management_socket` field absent entirely | Validate() accepts; daemon startup code applies mode-specific default path (e.g., `/run/switchboard-router.sock` for router). No validation error. |
| EC-015 | `admission_key_file: "   "` (whitespace-only value) | E-CFG-014 "config error: admission_key_file: must not be empty..."; exit 1. Exhaustive error collection with other errors. |
| EC-016 | `admission_key_file` absent entirely | Validate() accepts; access-mode daemon startup applies default path `/var/lib/switchboard/access-admission-identity.pem`. No validation error. |
| EC-017 | `admission_key_file` set to a valid non-whitespace path string | Validate() accepts regardless of whether the file exists (no I/O). |
| EC-018 | `admission_state_file: "   "` (whitespace-only value) | E-CFG-015 "config error: admission_state_file: must not be empty..."; exit 1. |
| EC-019 | `router_management_endpoints` has two entries; first valid, second `addr: "notvalid"` | E-CFG-016 naming index 1 (0-based); all errors collected before exit 1 (exhaustive reporting). |
| EC-020 | `router_management_endpoints: []` or field absent | Validate() accepts; no push replication; control writes succeed locally only. |
| EC-021 | `control_admission_state_file: "   "` (whitespace-only value) | E-CFG-017 "config error: control_admission_state_file: must not be empty..."; exit 1. Exhaustive error collection with other errors. |
| EC-022 | `control_admission_state_file` absent entirely | Validate() accepts; control starts without persistence; EC-007 resync guarantee does not apply; PushFullSnapshot on startup pushes an empty keyset. No validation error. |
| EC-023 | `control_admission_state_file: "/var/lib/switchboard/control-state.json"` (non-whitespace, file may not exist) | Validate() accepts (no I/O). At daemon startup, loadSnapshotFromFile handles missing→empty, corrupt→E-KEY-002+exit-1. |
| EC-024 | `management_socket: "0.0.0.0:9091"` in control or access mode config | `buildMgmtListener` returns E-CFG-008 at bind time: "management_socket: control mode requires a loopback address..."; daemon exits 1. |
| EC-025 | `management_socket: "127.0.0.1:9091"` in control or access mode config | `buildMgmtListener` accepts; daemon binds TCP loopback listener; INFO log "control management listener bound to 127.0.0.1:9091"; starts normally. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Missing required field `listen_addr` | E-CFG-001 "config error: listen_addr: required field missing. Fix: add 'listen_addr: <ip>:<port>' to config"; exit 1 | happy-path |
| `listen_addr: "0.0.0.0"` (no port) | E-CFG-002 "config error: listen_addr: '0.0.0.0' is not a valid host:port..."; exit 1 | error |
| `upstream_routers: [{addr: "notvalid"}]` | E-CFG-003 "config error: upstream_routers[0].addr: 'notvalid' is not a valid host:port..."; exit 1 | error |
| `drain_timeout: 0s` (or field absent) | Validate() accepts; daemon applies default 10s at startup (S-7.04); exit 0 | happy-path (optional field) |
| `drain_timeout: -5s` | E-CFG-006 "config error: drain_timeout: must not be negative; got '-5s'..."; exit 1 | error |
| `keepalive_interval: -1s` | E-CFG-007 "config error: keepalive_interval: must not be negative; got '-1s'..."; exit 1 | error |
| Invalid YAML syntax | E-CFG-005 "config parse error: invalid YAML at line 5: unexpected token"; exit 1 | error |
| Config file not found | E-CFG-004 "config file not found: /etc/switchboard/router.yaml"; exit 1 | error |
| Config reload with bad config | Daemon logs "config reload failed"; continues on previous config; exits 0 (daemon still running) | edge-case |
| Valid config supplied with `tick_interval: 20ms` | `halfchannel.New` receives `20ms` tick interval (not hardcoded `10ms`); daemon starts normally | happy-path (PC-9 v1.3) |
| `management_socket: "   "` (whitespace-only) | E-CFG-008 "config error: management_socket: must not be empty..."; exit 1 | error (PC-10, v1.6) |
| `management_socket` field absent | Validate() accepts; daemon default applied at startup; exit 0 | happy-path (PC-10, optional field) |
| `authorized_operator_keys: ["not-pem"]` | E-CFG-009 "config error: authorized_operator_keys[0]: entry is not a valid Ed25519 PEM PUBLIC KEY block..."; exit 1 | error (PC-11, v1.6) |
| `authorized_operator_keys: []` or field absent | Validate() accepts; bootstrap mode; exit 0 | happy-path (PC-11, bootstrap) |
| `admission_key_file: "   "` (whitespace) | E-CFG-014 "config error: admission_key_file: must not be empty..."; exit 1 | error (PC-12, v2.1) |
| `admission_key_file` absent | Validate() accepts; access daemon uses default path at startup; exit 0 | happy-path (PC-12, optional field) |
| `admission_key_file: "/var/lib/switchboard/foo.pem"` (non-whitespace, file may not exist) | Validate() accepts (no I/O); exit 0 | happy-path (PC-12, no-I/O rule) |
| `admission_state_file: "   "` (whitespace) | E-CFG-015 "config error: admission_state_file: must not be empty..."; exit 1 | error (PC-13, v2.1) |
| `admission_state_file` absent | Validate() accepts; router starts with empty keyset; exit 0 | happy-path (PC-13, optional field) |
| `router_management_endpoints: [{addr: "notvalid"}]` | E-CFG-016 "config error: router_management_endpoints[0].addr: 'notvalid' is not a valid host:port..."; exit 1 | error (PC-14, v2.1) |
| `router_management_endpoints: [{addr: "10.0.0.2:9093"}]` | Validate() accepts; exit 0 | happy-path (PC-14, v2.1) |
| `router_management_endpoints: [{addr: "0.0.0.0:9093"}]` | Validate() accepts (no loopback restriction); exit 0 | happy-path (PC-14, Ruling 9) |
| `router_management_endpoints: []` or field absent | Validate() accepts; no push replication; exit 0 | happy-path (PC-14, empty list) |
| `control_admission_state_file: "   "` (whitespace) | E-CFG-017 "config error: control_admission_state_file: must not be empty..."; exit 1 | error (PC-15, v2.2) |
| `control_admission_state_file` absent | Validate() accepts; control starts without persistence; EC-007 resync does not apply; exit 0 | happy-path (PC-15, optional field) |
| `control_admission_state_file: "/var/lib/switchboard/control-state.json"` (non-whitespace) | Validate() accepts (no I/O); exit 0 | happy-path (PC-15, no-I/O rule) |
| `management_socket: "0.0.0.0:9091"` in control-mode config | `buildMgmtListener` fails at bind time with E-CFG-008 "management_socket: control mode requires a loopback address..."; exit 1 | error (PC-11b, Ruling 12, v2.2) |
| `management_socket: "0.0.0.0:9091"` in access-mode config | `buildMgmtListener` fails at bind time with E-CFG-008 "management_socket: access mode requires a loopback address..."; exit 1 | error (PC-11b, Ruling 12, v2.2) |
| `management_socket: "127.0.0.1:9091"` in control-mode config | `buildMgmtListener` accepts; TCP loopback bind succeeds; INFO log emitted; daemon starts normally | happy-path (PC-11b, Ruling 12, v2.2) |
| `management_socket: "0.0.0.0:9093"` in router-mode config | `buildMgmtListener` accepts (no loopback restriction for router per Ruling 9); daemon starts normally | happy-path (Ruling 9 router-mode exemption, unchanged) |

## Verification Properties

| VP-NNN | Property | Proof Method | Notes |
|--------|----------|-------------|-------|
| VP-028 | Out-of-range tick_interval (d < 5ms or d > 50ms) causes Config.Validate to return non-nil error with code E-CFG-001 naming field 'tick_interval' | unit (table-driven) | Startup-scope Config.Validate property only. Proven: verification_lock true, 2026-07-06. |
| VP-029 | Zeroing any required field in {listen_addr, tick_interval} causes Config.Validate to return non-nil E-CFG-001 error naming the absent field | unit (table-driven) | Startup-scope Config.Validate property only. Proven: verification_lock true, 2026-07-06. |
| test-as-evidence | Startup with any config validation error always exits non-zero with E-CFG-001 message naming the field and providing a fix suggestion | unit (test suite; no formal VP assigned) | General exit-nonzero guarantee derived from VP-028/VP-029 for tick_interval/listen_addr; extended to other fields (E-CFG-002 through E-CFG-009) by analogous unit tests in S-6.01/S-W5.01. |
| test-as-evidence | listen_addr host:port parse enforced at validation (E-CFG-002) | unit (S-6.01 AC-003; no formal VP assigned) | |
| test-as-evidence | upstream_routers[N].addr host:port parse enforced (E-CFG-003) | unit (S-6.01 AC-004; no formal VP assigned) | |
| test-as-evidence | drain_timeout and keepalive_interval: zero/absent accepted; negative rejected (E-CFG-006/E-CFG-007) | unit (S-6.01 AC-007/AC-008; no formal VP assigned) | |
| test-as-evidence | management_socket present-and-blank rejected with E-CFG-008; absent accepted | unit (S-W5.01 AC-011; no formal VP assigned) | |
| test-as-evidence | authorized_operator_keys invalid PEM entry rejected with E-CFG-009 (per-index); empty list accepted | unit (S-W5.01 AC-012; no formal VP assigned) | |
| test-as-evidence | Validated config applied to daemon subsystems (tick_interval → halfchannel.New; not hardcoded default) | unit/integration (S-6.01 AC-009; no formal VP assigned) | |
| test-as-evidence | Config reload failure leaves daemon on previous config (Inv-3/EC-004: fail-closed reload) | integration (S-7.04-FU-SIGHUP-RELOAD AC-002; no formal VP assigned) | VP-028/VP-029 do NOT cover the SIGHUP reload integration path. This property is a new integration-test obligation for S-7.04-FU-SIGHUP-RELOAD. |
| test-as-evidence | `admission_key_file` present-and-blank rejected with E-CFG-014; absent accepted (no I/O in Validate) | unit (S-BL.NODE-ADMISSION-PROVISIONING AC; no formal VP assigned) | |
| test-as-evidence | `admission_state_file` present-and-blank rejected with E-CFG-015; absent accepted | unit (S-BL.ADMISSION-SYNC-WIRE AC; no formal VP assigned) | |
| test-as-evidence | `router_management_endpoints[N].addr` invalid host:port rejected with E-CFG-016 (exhaustive); empty list and valid host:port accepted; no loopback restriction (any bind accepted) | unit (S-BL.ADMISSION-SYNC-WIRE AC; no formal VP assigned) | |
| test-as-evidence | `control_admission_state_file` present-and-blank rejected with E-CFG-017; absent accepted (no I/O in Validate) | unit (S-BL.ADMISSION-SYNC-WIRE AC; no formal VP assigned) | PC-15, v2.2 |
| test-as-evidence | Control and access modes with non-loopback TCP `management_socket` rejected with E-CFG-008 at bind time; loopback TCP accepted; INFO log emitted on successful bind | integration (S-BL.ADMISSION-SYNC-WIRE AC; no formal VP assigned) | PC-11b, Ruling 12, v2.2 |
| test-as-evidence | Router mode with non-loopback TCP `management_socket` accepted (no loopback guard; Ruling 9 exemption unchanged) | integration (S-BL.ADMISSION-SYNC-WIRE AC; no formal VP assigned) | PC-11b, Ruling 9 router-mode exemption |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-028 ("Daemon startup config validation") per capabilities.md §CAP-028 |
| L2 Domain Invariants | (none directly; anchored to FM-010 via capability CAP-028) |
| Architecture Module | internal/config |
| Stories | S-6.01 (AC-001 through AC-009); S-W5.01 (PC-10 → AC-011: management_socket validation; PC-11 → AC-012: authorized_operator_keys PEM validation); S-BL.NODE-ADMISSION-PROVISIONING (PC-12: admission_key_file validation); S-BL.ADMISSION-SYNC-WIRE (PC-13: admission_state_file validation; PC-14: router_management_endpoints validation; PC-15: control_admission_state_file validation; PC-11b: loopback guard scope correction for control+access modes) |
| Capability Anchor Justification | CAP-028 ("Daemon startup config validation") per capabilities.md §CAP-028 — this BC directly realizes the guarantee that a daemon exits non-zero with an actionable error message before accepting any connections, which is exactly the scope of CAP-028. The config-application postcondition (PC-9) is a necessary corollary: validation is meaningless if the validated config is then discarded. Anchored to FM-010 (deployment misconfig). |

## Related BCs

- BC-2.09.001 — related to: config errors discovered on reload (including upstream_routers address validation) use the same E-CFG-* error mechanism
- BC-2.04.007 — parallel: access node daemon startup/shutdown lifecycle (same class of lifecycle contract, different daemon, different subsystem); BC-2.04.007 does not own config validation

## Architecture Anchors

- ARCH-06-deployment-and-ops.md §Config File Validation (BC-2.09.003, NFR-011) — binding sequence (loadConfigFile → Validate → bindListenSocket) is authoritative
- ARCH-INDEX.md §SS-09 (deployment-operations, internal/config)

## Story Anchor

S-6.01 — AC-001 through AC-009 trace to postconditions in this BC.
S-W5.01 — AC-011 (PC-10: management_socket) and AC-012 (PC-11: authorized_operator_keys) trace to v1.6 postconditions in this BC.
S-BL.NODE-ADMISSION-PROVISIONING — ACs for PC-12 (admission_key_file validation, E-CFG-014) trace to v2.1 postconditions in this BC.
S-BL.ADMISSION-SYNC-WIRE — ACs for PC-13 (admission_state_file validation, E-CFG-015), PC-14 (router_management_endpoints validation, E-CFG-016), PC-15 (control_admission_state_file validation, E-CFG-017), and PC-11b (loopback guard scope for control+access modes, Ruling 12) trace to postconditions in this BC.

## VP Anchors

VP-028 — startup-scope: Config.Validate rejects out-of-range tick_interval (d < 5ms or d > 50ms); proven, verification_lock true.
VP-029 — startup-scope: Config.Validate rejects missing required fields (listen_addr, tick_interval); proven, verification_lock true.
Both VPs scope strictly to Config.Validate behavior in internal/config. They do NOT cover the SIGHUP reload integration path, host:port parsing properties, drain/keepalive/management/keys properties, or config application to subsystems. Those properties are covered by test-as-evidence (story ACs) as documented in the Verification Properties table above.

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 2.2 | 2026-07-17 | add PC-15 `control_admission_state_file` / E-CFG-017 per Ruling 11 (F-P3-01); extend mgmt-listener loopback restriction to control+access modes (router-only exemption) per Ruling 12 (F-P3-02) — PC-11b added; EC-021..EC-025 added; test-vector rows + VP rows added for both rulings. |
| 2.1 | 2026-07-15 | Identity-cluster BC groundwork consolidated amendment (items N3+A3+A4): PC-12 added (`admission_key_file`: non-empty when present, no file I/O in Validate, E-CFG-014); PC-13 added (`admission_state_file`: non-empty when present, E-CFG-015); PC-14 added (`router_management_endpoints`: each `addr` validated as `host:port` per E-CFG-016, NO loopback restriction per Ruling 9 — ADR-012 is the auth boundary, control-mode-only field); E-CFG-014/015/016 added to Error Codes table; EC-015 through EC-020 added; Canonical Test Vectors for all three fields added; test-as-evidence VP rows added. Added `inputs`/`input-hash`/`extracted_from` frontmatter fields (template conformance). |
| 2.0 | 2026-07-06 | Governance-only: narrowed Verification Properties table to correct pre-existing authoring drift (PO ruling S-6.04-disposition-ruling.md §"Spec Note: BC-2.09.003 Verification Properties Table Drift"). VP-028 and VP-029 were overstated as covering all 9 VP-table rows including "Config reload failure leaves daemon on previous config" and host:port/drain/keepalive/management/keys properties. Both VPs scope strictly to Config.Validate startup-validation behavior (tick_interval out-of-range; missing required fields). Table rebuilt: two rows retain VP-028/VP-029 citations for the properties they actually prove; all other rows reclassified as test-as-evidence with owning story/AC citations. Reload-integration fail-closed property (Inv-3/EC-004) explicitly annotated as S-7.04-FU-SIGHUP-RELOAD AC-002 obligation with no formal VP. No PC/EC/Inv semantic changes; no runtime behavior implied. Governance-leaf change. |
| 1.9 | 2026-07-02 | Phase 5 Pass 3 remediation Path B follow-up: error-taxonomy v4.3 reconciled E-CFG-002 (private-key-export → E-CFG-011) and E-CFG-006 (sbctl --yes → E-CFG-012). Collision-flag annotation row removed from Error Codes table — pre-existing inconsistency resolved. Closes DRIFT-P5P3-A007-ECFG-COLLISION (BC-2.09.003 side). Refs F-P5P3-A-007. |
| 1.8 | 2026-07-02 | Reconcile listen_addr DEFERRED-APPLICATION row — previously said "No current owner story" contradicting STORY-INDEX line 135 which lists S-BL.NI as the owner. Row now correctly names S-BL.NI. Closes Phase 5 Pass 2 F-P5P2-B-002. |
| 1.7 | 2026-06-28 | Traceability refresh (Wave-5 consistency audit F-002/F-003): Stories field updated — S-W5.01 added for PC-10/PC-11 (AC-011: management_socket, AC-012: authorized_operator_keys); S-6.01 retains AC-001..AC-009. Story Anchor updated to include S-W5.01. E-CFG-002 collision flag added to Error Codes table: error-taxonomy.md E-CFG-002 = "private key export not supported" (BC-2.05.007); BC-2.09.003 v1.2 E-CFG-002 = listen_addr invalid host:port — pre-existing inconsistency, flagged for maintenance-pass resolution. |
| 1.6 | 2026-06-28 | Wave-5 management plane config additions (ARCH-12): PC-10 (management_socket: non-empty when present; E-CFG-008) and PC-11 (authorized_operator_keys: valid PEM Ed25519 PUBLIC KEY per entry; E-CFG-009) added. Error codes E-CFG-008 and E-CFG-009 added to Error Codes table. Edge cases EC-011 through EC-014 added. Canonical test vectors for both new fields added. VP coverage rows added to Verification Properties table. Config-schema impact note in PC-11: interface-definitions.md §Config Schema requires update for management_socket and authorized_operator_keys fields (S-W5.01 responsibility). Pre-existing E-CFG-006 collision flagged (error-taxonomy.md E-CFG-006 = sbctl admin flag conflict; BC-2.09.003 v1.4 E-CFG-006 = drain_timeout negative) — reconciliation deferred to maintenance pass. New codes E-CFG-008/E-CFG-009 are free in both documents. |
| 1.5 | 2026-06-28 | Traceability refresh (KNOWN-STALE / Wave 4 audit): Stories row and Story Anchor updated from "AC-001 through AC-006" to "AC-001 through AC-009". S-6.01 gained AC-007 (drain_timeout negative rejection, PC-7), AC-008 (keepalive_interval negative rejection, PC-8), and AC-009 (tick_interval config application, PC-9) via SP-003/SP-004/SP-005 — the BC's postconditions PC-5..PC-9 were already fully defined; only the human-readable anchor summary was stale. Canonical EC numbering reaffirmed: BC EC-NNN is authoritative (VSDD policy); story-writer must reconcile S-6.01 EC IDs to BC EC IDs per S601-NITPICK-B: story EC-009 → BC EC-008 (drain_timeout:0s accepted), story EC-010 → BC EC-009 (drain_timeout:-5s → E-CFG-006, already correct in BC), story EC-011 → new (keepalive_interval:0s accepted — not yet in BC; story-writer to add BC EC-011 for symmetry if desired), story EC-012 → BC EC-009 (keepalive_interval:-1s → E-CFG-007), story EC-013 → BC EC-010 (valid config/daemon start). |
| 1.4 | 2026-06-28 | Resolved 3-way contradiction: BC PC-7/PC-8 said "if present, must be > 0" (implying optional but rejecting zero), config.go rejected zero/absent as required fields (E-CFG-006/007), and ARCH-06 documented defaults (drain_timeout 10s, keepalive_interval 1s). Human ruling: "optional with defaults, align to ARCH-06." PC-7 and PC-8 updated: both fields are optional; Validate() rejects ONLY a negative value; zero/absent is accepted (daemon default applied at startup by S-7.04). E-CFG-006/007 trigger conditions updated from "zero or negative" to "negative"; message templates updated from "must be > 0" to "must not be negative." EC-008 corrected: drain_timeout: 0s is now accepted (daemon default 10s). Canonical test vector updated to match. |
| 1.3 | 2026-06-28 | Right-sized PC-9 and Inv-5. Fresh-eyes verification confirmed that `listen_addr` binding, `drain_timeout`, `upstream_routers`, and `keepalive_interval` APPLICATION targets subsystems that do not exist on develop. Human ruling: "apply what exists now, track the rest as concrete dependencies." PC-9 narrowed: only `tick_interval` is applied now (wired to `halfchannel.New` in `cmd/switchboard/access.go`, currently hardcoded at `10ms`). DEFERRED-APPLICATION note added with named owning stories for each deferred field (listener introduction: no owner yet — flagged for STORY-INDEX; drain/PE/keepalive: S-7.04 Wave 7). Inv-5 narrowed to "applicable fields" so legitimately-deferred fields do not constitute an invariant violation. H1 title updated to "Applicable Subsystems." EC-010 and PC-9 canonical test vector updated. |
| 1.2 | 2026-06-28 | S-6.01 scope expansion to cover (a) deep field validation and (b) config application. Added PC-5 through PC-9; added E-CFG-002, E-CFG-003, E-CFG-006, E-CFG-007 error codes; added EC-005 through EC-010; updated title H1 to reflect both behaviors; added Inv-4 and Inv-5. Fixed `subsystem:` frontmatter to use SS-09 (ARCH-INDEX Subsystem Registry). |
| 1.1 | 2026-06-23 | Initial draft — router startup fails cleanly on malformed config. |
