---
artifact_id: ARCH-01-core-services
document_type: architecture-section
level: L3
version: "1.1"
status: draft
producer: architect
timestamp: 2026-06-23T00:00:00
modified: 2026-06-25T00:00:00
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

## ADR-010: Terminal Session Backend — Tmux Control Mode Primary, PTY Proxy Fallback (S-3.01)

**Decision:** `internal/tmux` uses tmux control mode (`tmux -C`) as the primary
terminal session backend. PTY proxy mode is an automatic fallback triggered only at
initial connection time when the control mode connection fails.

**Why tmux control mode is preferred:**
1. Machine-readable event stream: `%output`, `%session-changed`, `%window-add`, and
   `%exit` events arrive as structured lines — no screen-scraping required.
2. Named session addressing: consoles connect by session name (`tmux attach -t NAME`);
   control mode natively enumerates sessions.
3. Session persistence: the tmux server persists sessions independently of the access
   node process. If the access node restarts, it reconnects to the existing tmux
   server rather than losing session state.
4. Fan-out compatibility: `ConsoleSet` fan-out (S-3.02) distributes the event stream
   to multiple consoles from a single tmux control mode connection, avoiding N×tmux
   connections for N attached consoles.

**Why PTY fallback is included:**
- tmux may not be installed on the target host. PTY proxy provides degraded-mode
  operation so the access node does not hard-fail.
- PTY mode does not support named sessions; the fallback is a single-session proxy.
- PTY mode provides functionally equivalent keystroke-to-echo behavior (AC-004) but
  lacks session listing, named session attach, and persistence.

**Fallback semantics (BC-2.04.002):**
- Fallback is triggered only on initial `TmuxControlMode.Attach` failure. It is NOT
  triggered if the control mode connection drops mid-session (EC-002 in S-3.01).
- Once in PTY fallback mode, the access node stays in PTY mode for the lifetime of
  that session. There is no automatic upgrade to control mode.
- At next daemon restart, `TmuxControlMode.Attach` is retried before falling back.

**Rejected alternatives:**
- PTY-only mode: loses session naming, persistence, and efficient fan-out.
- screen as alternative: no structured event protocol; screen-scraping required.
  Adds fragile parsing, not a clean boundary.
- libvterm embedding: complex C dependency; not justified for MVP LAN target.

**References:** BC-2.04.001 (control mode attach), BC-2.04.002 (PTY fallback),
S-3.01 EC-002 (mid-session drop does not trigger fallback).

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-06-23 | Initial core services architecture |
| 1.1 | 2026-06-25 | Added ADR-010: tmux control mode primary, PTY proxy fallback (Wave 3 / S-3.01) |
