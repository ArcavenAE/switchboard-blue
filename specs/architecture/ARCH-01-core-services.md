---
artifact_id: ARCH-01-core-services
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
  - '.factory/specs/behavioral-contracts/ss-04/BC-2.04.001.md'
  - '.factory/specs/behavioral-contracts/ss-09/BC-2.09.003.md'
kos_anchors:
  - elem-single-binary-three-modes
  - elem-node-router-architecture
---

# ARCH-01: Core Services

## Single Binary, Six Subcommands

Per elem-single-binary-three-modes, one binary serves all deployment roles.
Mode is selected by subcommand, not by build flags (except P router).

| Subcommand | Runtime Mode | Primary Role | Phase |
|------------|-------------|-------------|-------|
| `switchboard router` | E or PE router | Frame forwarding, HMAC auth, admission | E (MVP) |
| `switchboard access` | Access node | tmux publishing, session I/O, Tier 2 auth | E (MVP) |
| `switchboard console` | Console node | Session attach/detach, downstream render | E (MVP) |
| `switchboard control` | Control node | SVTN lifecycle, key registration | E (MVP) |
| `switchboard version` | Version query | Print version, exit | E (MVP) |
| `switchboard help` | Help | Print usage, exit | E (MVP) |

The E vs PE router distinction is purely config-driven: `upstream_routers: []` = E,
any entries = PE. The binary contains both code paths; the config selects.

## cmd/switchboard Package Layout

```
cmd/
  switchboard/         # main package
    main.go            # entrypoint: run(stdout, args) function pattern
    main_test.go       # integration smoke tests
    router.go          # router subcommand handler
    access.go          # access node subcommand handler
    console.go         # console node subcommand handler
    control.go         # control node subcommand handler

cmd/
  sbctl/               # operator CLI (separate binary)
    main.go
    commands/          # one file per subcommand group
```

The existing `main.go` stub (wave-0) establishes the `run(stdout io.Writer, args []string) error`
pattern — the real implementation replaces the stub body while preserving this signature.

## Daemon Lifecycle

```
main() → run(stdout, os.Args)
  → parse global flags (--config, --log-level, --log-format)
  → dispatch to mode handler (router/access/console/control)
  → mode handler:
      1. loadConfig(path) → validate → fail with actionable error if invalid (BC-2.09.003)
      2. initLogger(level, format)
      3. buildDependencies() → pure-core modules initialized first
      4. startServices() → bind/listen (after config validated — NFR-011)
      5. installSignalHandlers(SIGTERM → graceful drain, SIGHUP → reload)
      6. serve() → event loop until shutdown
      7. shutdown() → drain active sessions, close listeners
```

## Signal Handling

| Signal | Router | Access | Console | Control |
|--------|--------|--------|---------|---------|
| SIGTERM | graceful drain (BC-2.09.002) | close sessions | detach | close |
| SIGHUP | reload config | reload config | — | reload config |
| SIGINT | same as SIGTERM | same | same | same |
| SIGKILL | immediate exit (FM-009) | immediate | immediate | immediate |

`sbctl router drain` sends the equivalent of SIGTERM over the management socket.

## Supervision and Error Handling

- No `log.Fatal` or `os.Exit` outside `main()`. All errors propagate via `error` return.
- Config validation must complete before `bind`/`listen` call (NFR-011). Any config
  error exits with code 1 and a human-readable message identifying the field.
- Panics in the daemon are recovered at the event loop boundary and reported as
  exit code 3 (internal error).
- No `init()` functions. All dependencies are passed explicitly via constructors.

## Mode Multiplexing for BCs

| BC | Mode | Package |
|----|------|---------|
| BC-2.04.001, BC-2.04.002 | access node | internal/tmux |
| BC-2.04.003–006 | console node | internal/session |
| BC-2.08.001 | console node + sbctl | cmd/sbctl, internal/session |
| BC-2.09.003 | all modes | internal/config |
| BC-2.07.001 | control node | internal/svtnmgmt |

## Concurrency Model

Each daemon uses a single-threaded event loop per logical connection, with a
shared-memory pool for frame buffers. The goroutine model for 1,000 concurrent
sessions is an open question (NFR-004 notes in ARCH-INDEX Open Frontier Questions).
Initial design: one goroutine pair (reader + writer) per connection. Profiling gates
refactoring to an event-loop model before PE phase.
