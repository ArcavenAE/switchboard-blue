---
artifact_id: ARCH-05-cli-and-api
document_type: architecture-section
level: L3
version: "1.4"
status: draft
producer: architect
timestamp: 2026-06-23T00:00:00
phase: 1b
traces_to: ARCH-INDEX.md
inputDocuments:
  - '.factory/specs/prd.md'
  - '.factory/specs/prd-supplements/interface-definitions.md'
  - '.factory/specs/behavioral-contracts/ss-07/BC-2.07.002.md'
  - '.factory/specs/behavioral-contracts/ss-07/BC-2.07.003.md'
kos_anchors:
  - elem-single-binary-three-modes
  - elem-node-router-architecture
modified:
  - 2026-06-28T00:00:00 # v1.1 — ADR-012 cross-reference; management plane detail deferred to ARCH-12
  - 2026-06-28T00:00:00 # v1.2 — F-010: BC table updated to 45 BCs (added BC-2.04.007, BC-2.05.008, BC-2.07.004); mgmt row status corrected to active
  - 2026-06-28T00:00:00 # v1.3 — adversarial review Ruling 4: Unix socket permission 0600 required; console TCP loopback-only confirmed; 0.0.0.0 binding forbidden
  - 2026-06-29T00:00:00 # v1.4 — Wave-5 convergence round-2 Ruling D: console TCP loopback validation placement confirmed in buildMgmtListener (cmd/switchboard/mgmt_wire.go), not config.Validate(); rejection predicate and E-CFG-008 variant documented
---

# ARCH-05: CLI & API

## ADR-006: Daemon RPC Protocol

**Decision:** JSON-over-Unix-socket with SSH signature authentication.

**Rationale:**
- gRPC: requires protobuf toolchain, heavier dependency tree. CLI output is already
  JSON (interface-definitions.md); JSON-over-socket avoids dual schema maintenance.
- Custom binary protocol: unnecessary complexity for a management plane that carries
  configuration and status, not session traffic.
- JSON-over-Unix-socket: simple, debuggable with `nc` or `socat`, zero-dependency
  on the client side (sbctl uses standard `net.Dial`). The JSON schema is specified
  in interface-definitions.md.
- Authentication: the caller signs a challenge nonce with their OpenSSH key (same
  mechanism as SVTN admission, BC-2.05.001). The daemon verifies against the admitted
  operator key set. Unauthenticated calls are rejected before processing.
- TCP fallback: if the Unix socket path is absent or `--target=host:port` is specified,
  sbctl uses TCP. Authentication is the same.

> **Wave 5 refinement (ADR-012):** ADR-006 established the conceptual protocol.
> ADR-012 in ARCH-12 specifies the exact message sequence (CHALLENGE / CHALLENGE_RESPONSE /
> AUTH_OK / AUTH_FAIL), the NDJSON framing, the operator key set source, and the
> bounded-read contract. See ARCH-12-daemon-management-plane.md for the full
> management plane design including `internal/mgmt` package, config additions,
> and story decomposition.

## Go Package Layout

The architecture formalizes the indicative package paths from interface-definitions.md:

| Package | Go import path | Purity | Primary BCs |
|---------|---------------|--------|-------------|
| frame | `internal/frame` | pure-core | BC-2.01.004, BC-2.01.005 |
| hmac | `internal/hmac` | pure-core | BC-2.05.005 |
| admission | `internal/admission` | boundary | BC-2.05.001, BC-2.05.002 |
| session | `internal/session` | boundary | BC-2.05.003, BC-2.04.003–006 |
| halfchannel | `internal/halfchannel` | pure-core (state machine) | BC-2.01.001–003 |
| multipath | `internal/multipath` | pure-core | BC-2.02.001, BC-2.02.002, BC-2.02.009 |
| arq | `internal/arq` | pure-core | BC-2.02.005, BC-2.02.006 |
| replay | `internal/replay` | pure-core | BC-2.02.004 |
| paths | `internal/paths` | pure-core (scoring) | BC-2.02.003 |
| routing | `internal/routing` | boundary | BC-2.02.008, BC-2.05.006 |
| discovery | `internal/discovery` | boundary (PE) | BC-2.03.001–003 |
| metrics | `internal/metrics` | pure-core | BC-2.06.001, BC-2.06.002 |
| tmux | `internal/tmux` | effectful | BC-2.04.001, BC-2.04.002 |
| config | `internal/config` | pure-core (parse+validate) | BC-2.09.003, NFR-011 |
| svtnmgmt | `internal/svtnmgmt` | boundary | BC-2.07.001, BC-2.05.004 |
| mgmt | `internal/mgmt` | boundary (effectful: owns listener) | BC-2.07.004 — Wave 5 (active) |
| drain | `internal/drain` | effectful | BC-2.09.002 |
| sbctl | `cmd/sbctl` | effectful | BC-2.07.002, BC-2.07.003, BC-2.08.001 |

## BC → Architecture Module Mapping

This table resolves the `architecture_module:` field for all 45 BCs:

| BC ID | Go Package | Module Name |
|-------|-----------|-------------|
| BC-2.01.001 | `internal/halfchannel` | halfchannel |
| BC-2.01.002 | `internal/halfchannel` | halfchannel |
| BC-2.01.003 | `internal/halfchannel` | halfchannel |
| BC-2.01.004 | `internal/frame` | frame |
| BC-2.01.005 | `internal/frame` | frame |
| BC-2.01.006 | `internal/frame` | frame |
| BC-2.01.007 | `internal/admission` | admission |
| BC-2.02.001 | `internal/multipath` | multipath |
| BC-2.02.002 | `internal/multipath` | multipath |
| BC-2.02.003 | `internal/paths` | paths |
| BC-2.02.004 | `internal/replay` | replay |
| BC-2.02.005 | `internal/arq` | arq |
| BC-2.02.006 | `internal/arq` | arq |
| BC-2.02.007 | `internal/arq` | arq (FEC extension) |
| BC-2.02.008 | `internal/routing` | routing |
| BC-2.02.009 | `internal/multipath` | multipath |
| BC-2.03.001 | `internal/discovery` | discovery |
| BC-2.03.002 | `internal/discovery` | discovery |
| BC-2.03.003 | `internal/discovery` | discovery |
| BC-2.04.001 | `internal/tmux` | tmux-control |
| BC-2.04.002 | `internal/tmux` | tmux-control |
| BC-2.04.003 | `internal/session` | session-auth |
| BC-2.04.004 | `internal/session` | session-auth |
| BC-2.04.005 | `internal/session` | session-auth |
| BC-2.04.006 | `internal/session` | session-auth |
| BC-2.04.007 | `cmd/switchboard` | switchboard-daemon |
| BC-2.05.001 | `internal/admission` | admission |
| BC-2.05.002 | `internal/admission` | admission |
| BC-2.05.003 | `internal/session` | session-auth |
| BC-2.05.004 | `internal/svtnmgmt` | svtn-mgmt |
| BC-2.05.005 | `internal/hmac` | hmac |
| BC-2.05.006 | `internal/routing` | routing |
| BC-2.05.007 | `internal/admission` | admission |
| BC-2.05.008 | `internal/routing` | routing |
| BC-2.06.001 | `internal/metrics` | metrics |
| BC-2.06.002 | `internal/metrics` | metrics |
| BC-2.06.003 | `internal/metrics` | metrics |
| BC-2.07.001 | `internal/svtnmgmt` | svtn-mgmt |
| BC-2.07.002 | `cmd/sbctl` | sbctl |
| BC-2.07.003 | `cmd/sbctl` | sbctl |
| BC-2.07.004 | `internal/mgmt` | mgmt |
| BC-2.08.001 | `cmd/sbctl` | sbctl |
| BC-2.09.001 | `internal/config` | config |
| BC-2.09.002 | `internal/drain` | drain |
| BC-2.09.003 | `internal/config` | config |

## sbctl CLI Design

The CLI is a separate binary (`cmd/sbctl`) but shares all `internal/` packages with
the daemon. It communicates with daemons over the management socket.

**BC-2.07.003 (connection error reporting):** sbctl detects `dial` failures and
emits `E-NET-001` with the attempted address. It never produces successful-looking
output when the daemon is unreachable. Exit code 1.

**BC-2.07.002 (unified CLI):** All daemon types are reachable via `--target`. The
same `sbctl` binary can target router, access, console, or control daemons.

**JSON output:** `--json` flag switches all output to the common envelope format
defined in interface-definitions.md. Error objects include the E-code.

## Daemon Management Socket

| Daemon Mode | Socket Type | Default Address | Notes |
|-------------|-------------|-----------------|-------|
| router | Unix | `/run/switchboard-router.sock` | Permissions: 0600 (see below) |
| access | Unix | `/run/switchboard-access.sock` | Permissions: 0600 |
| console | TCP | `127.0.0.1:9091` | Loopback-only (see below) |
| control | Unix | `/run/switchboard-control.sock` | Permissions: 0600 |

If the socket is absent, sbctl falls back to TCP on `--target`.

### Unix Socket Permissions (CWE-276 — adversarial review Ruling 4)

All Unix management sockets MUST be created with permissions `0600`. This is
achieved by setting `syscall.Umask(0177)` immediately before `net.Listen("unix", ...)`
and restoring the previous umask immediately after. Relying on the system umask
is forbidden — the daemon may run in environments where the umask is 0022 or 0000.

The `chmod`-after-create approach MUST NOT be used: it introduces a TOCTOU window
between socket creation (world-accessible) and the chmod call.

The full rationale and implementation pattern are in
ARCH-12-daemon-management-plane.md §Unix Socket Permissions.

### Console TCP Loopback Binding (CWE-276 — adversarial review Ruling 4; placement refined Ruling D / v1.4)

The console daemon's management TCP listener MUST bind to a loopback address only.
Binding to `0.0.0.0`, `::`, bare port (`:9091`), or any non-loopback IP is FORBIDDEN.

**Authorized loopback hosts:** `127.0.0.1`, `[::1]` (IPv6 loopback), `localhost`.

**Enforcement placement (Wave-5 convergence round-2 Ruling D):** The loopback-only
check is enforced in `buildMgmtListener` (`cmd/switchboard/mgmt_wire.go`), in the TCP
branch, before `net.Listen` is called. It is NOT enforced in `config.Validate()`
because `Validate()` has no mode parameter and cannot distinguish console mode from
other modes without violating the config package's purity-boundary classification
(pure-core parse+validate — see §Go Package Layout).

**Rejection:** Any console-mode TCP `management_socket` whose host is not in the
authorized loopback set causes `buildMgmtListener` to return an error with code
`E-CFG-008`: `"config error: management_socket: console mode requires a loopback
address (127.0.0.1, [::1], or localhost); got: <address>"`. Daemon startup aborts.

**IPv6 note:** `[::1]` is included to support IPv6-only hosts without requiring
a future ADR. The policy (loopback-only management plane) is unchanged.

The full rationale, rejection predicate, and implementation pattern are in
ARCH-12-daemon-management-plane.md §Ruling D.

