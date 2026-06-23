---
artifact_id: ARCH-05-cli-and-api
document_type: architecture-section
level: L3
version: "1.0"
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
| drain | `internal/drain` | effectful | BC-2.09.002 |
| sbctl | `cmd/sbctl` | effectful | BC-2.07.002, BC-2.07.003, BC-2.08.001 |

## BC → Architecture Module Mapping

This table resolves the `architecture_module:` field for all 42 BCs:

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
| BC-2.05.001 | `internal/admission` | admission |
| BC-2.05.002 | `internal/admission` | admission |
| BC-2.05.003 | `internal/session` | session-auth |
| BC-2.05.004 | `internal/svtnmgmt` | svtn-mgmt |
| BC-2.05.005 | `internal/hmac` | hmac |
| BC-2.05.006 | `internal/routing` | routing |
| BC-2.05.007 | `internal/admission` | admission |
| BC-2.06.001 | `internal/metrics` | metrics |
| BC-2.06.002 | `internal/metrics` | metrics |
| BC-2.06.003 | `internal/metrics` | metrics |
| BC-2.07.001 | `internal/svtnmgmt` | svtn-mgmt |
| BC-2.07.002 | `cmd/sbctl` | sbctl |
| BC-2.07.003 | `cmd/sbctl` | sbctl |
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

- **Router:** Unix socket at `config.ManagementSocket` (default: `/run/switchboard-router.sock`)
- **Access:** `/run/switchboard-access.sock`
- **Console:** `config.MgmtListenAddr` (default: `127.0.0.1:9091`)
- **Control:** `/run/switchboard-control.sock`

If the socket is absent, sbctl falls back to TCP on `--target`.

## PRD Section 7 RTM — Module Column

The Requirements Traceability Matrix (PRD §7) `Module(s)` column is now populated
from the BC→Package mapping table above. This backfill is complete.
