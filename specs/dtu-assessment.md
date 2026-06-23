---
artifact_id: dtu-assessment
document_type: dtu-assessment
level: L3
version: "1.0"
status: draft
producer: architect
timestamp: 2026-06-23T00:00:00
phase: 1b
DTU_REQUIRED: false
inputDocuments:
  - '.factory/specs/architecture/ARCH-05-cli-and-api.md'
  - '.factory/specs/architecture/ARCH-08-dependency-graph.md'
  - '.factory/specs/prd-supplements/interface-definitions.md'
  - '.factory/specs/prd.md'
traces_to: '.factory/specs/architecture/ARCH-INDEX.md'
---

# DTU Assessment: Switchboard

## Summary

| Metric | Value |
|--------|-------|
| External dependencies identified | 1 (tmux process, local only) |
| DTU clones recommended | 0 |
| Total clone story points | 0 |
| Estimated Wave 1 capacity needed | 0 points |
| DTU_REQUIRED | false |

## DTU_REQUIRED Justification

Switchboard MVP is a **pure local infrastructure binary** — a Go program that:
1. Opens network sockets (UDP/TCP) to other Switchboard processes.
2. Connects to the local `tmux` process via a Unix socket (control mode).
3. Reads and writes YAML config files and SSH key files from disk.
4. Exposes a Unix socket for `sbctl` (another local process).

There are **no third-party SaaS APIs**, no cloud provider services, no external
databases, no messaging queues, no identity providers, and no HTTP APIs to external
services. The entire system is self-contained on the operator's infrastructure.

This means:
- No DTU behavioral clones are needed.
- Integration tests can run against real local processes.
- The `tmux` dependency (below) is a process on the test machine, not a cloud service.

The pre-Phase-3 DTU clone existence check passes because `DTU_REQUIRED: false` with
this explicit justification document constitutes a deliberate architectural decision,
not an oversight.

## Integration Surface Inventory

### Inbound Data Sources (External → Product)

None identified — rationale: Switchboard receives frames from other Switchboard nodes
(not external services) over network sockets. The nodes are other instances of the
same binary. There is no external data feed, webhook, or third-party API that
Switchboard polls.

### Outbound Operations (Product → External)

None identified — rationale: Switchboard does not push data to any external service.
It does not send notifications, trigger payments, write to external storage, or call
any external API. All outbound traffic is to other Switchboard nodes or to the local
tmux process.

### Identity & Access (Bidirectional — auth flow)

| # | Service | Protocol | Fidelity | DTU? | Justification |
|---|---------|----------|----------|------|---------------|
| 1 | OpenSSH key files (local disk) | File I/O | L1 (read-only) | No | SSH keys are stored as local files. The "service" is the file system, not an external identity provider. No API, no network call, no credential exchange with a third party. Tests use locally generated test keypairs. |

Rationale for no DTU: OpenSSH key handling uses `golang.org/x/crypto/ssh` to parse
standard PEM key files. This is a well-tested library with no external API dependency.
Test keypairs are generated inline via `ssh-keygen` equivalent in test setup.

### Persistence & State (Product ↔ Storage)

| # | Service | Protocol | Fidelity | DTU? | Justification |
|---|---------|----------|----------|------|---------------|
| 1 | Local YAML config files | File I/O | L1 (read-only at runtime) | No | Config files are local. Integration tests write temp config files directly. No database, no external store. |

Rationale for no DTU: Config file parsing is a pure in-process operation. Tests
create temp files in `t.TempDir()`.

### Observability & Export (Product → Monitoring)

None identified — rationale: Switchboard has no telemetry, no metrics export to
external systems (Prometheus, Datadog, etc.) in MVP. Structured log output goes to
stdout/stderr only. No log aggregator, no tracing backend, no analytics platform.

Per the product brief: "No usage telemetry without operator consent." MVP has no
telemetry at all. If future versions add Prometheus metrics export, this section
will be updated.

### Enrichment & Lookup (External → Product, on-demand)

None identified — rationale: Switchboard does not query any external enrichment
service. There is no threat intelligence feed, no geocoding, no NVD lookup, no
license database. All routing decisions are based on locally held state (admitted
key set, forwarding table, path metrics).

### Special Case: tmux Process (Local IPC, Not External)

The access node connects to the local `tmux` process via a Unix socket (`tmux -CC`
control mode). This is local inter-process communication on the same machine, not
an external service.

| # | Service | Protocol | Fidelity | DTU? | Justification |
|---|---------|----------|----------|------|---------------|
| 1 | tmux (local process) | Unix socket / control mode | L2 (stateful) | No | tmux is a local process. Integration tests install tmux via the CI environment and run real tmux sessions. A behavioral clone is not needed and would not improve test fidelity — the tests work better with a real tmux process. PTY fallback (BC-2.04.002) is tested by not starting tmux. |

## Dependency Summary

| # | Service | Category | Fidelity | DTU? | Points | Justification |
|---|---------|----------|----------|------|--------|---------------|
| 1 | OpenSSH key files | Identity & Access | L1 | No | 0 | Local file I/O; test keypairs generated inline |
| 2 | Local YAML config files | Persistence & State | L1 | No | 0 | Temp files in tests |
| 3 | tmux (local process) | Inbound Data Source (local) | L2 | No | 0 | Real process in CI; PTY fallback for absent case |

## Services NOT Requiring DTU

| # | Service | Reason |
|---|---------|--------|
| 1 | OpenSSH key files | Local file I/O; standard crypto library; test keypairs generated in-process |
| 2 | tmux | Local process; real tmux used in CI; behavioral clone would decrease fidelity |
| 3 | All network peers | Other Switchboard instances; tested by running multiple instances in integration tests |

## DTU Architecture

Not applicable. DTU_REQUIRED: false.

## Clone Development Approach

Not applicable. No DTU clones required for Switchboard MVP.

## PE Phase Note

The PE router phase may introduce external dependencies (STUN/TURN servers for NAT
traversal, ASM-007). If Switchboard integrates with a STUN/TURN provider in PE phase,
this DTU assessment must be updated and a DTU clone for the STUN/TURN server created.
That is out of scope for the current E router MVP.
