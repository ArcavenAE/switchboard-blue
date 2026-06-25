---
artifact_id: ARCH-08-dependency-graph
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
  - '.factory/specs/module-criticality.md'
  - '.factory/specs/architecture/ARCH-05-cli-and-api.md'
kos_anchors:
  - elem-node-router-architecture
---

# ARCH-08: Dependency Graph

## Module Dependency DAG

Import direction convention: `A → B` means package A imports package B (A depends on B).
**No cycles.** Any cycle is an architecture violation per SOUL.md #11.

```mermaid
graph LR
    %% Layer 0: Foundation (no internal imports)
    frame["internal/frame\n(pure-core)"]
    hmac["internal/hmac\n(pure-core)"]
    config["internal/config\n(pure-core)"]

    %% Layer 1: Security (imports foundation)
    admission["internal/admission\n(boundary)"]
    session["internal/session\n(boundary)"]
    routing["internal/routing\n(boundary)"]

    %% Layer 2: Protocol (imports foundation + security)
    halfchannel["internal/halfchannel\n(pure-core)"]
    paths["internal/paths\n(pure-core)"]

    %% Layer 3: Reliability (imports protocol)
    arq["internal/arq\n(pure-core)"]
    replay["internal/replay\n(pure-core)"]
    multipath["internal/multipath\n(pure-core)"]
    metrics["internal/metrics\n(pure-core)"]

    %% Layer 4: Integration (imports reliability)
    tmux["internal/tmux\n(effectful)"]
    discovery["internal/discovery\n(boundary)"]
    svtnmgmt["internal/svtnmgmt\n(boundary)"]
    drain["internal/drain\n(effectful)"]

    %% Layer 5: Command layer (imports all)
    sbctl["cmd/sbctl\n(effectful)"]
    main["cmd/switchboard\n(effectful)"]

    %% Edges
    admission --> frame
    admission --> hmac
    session --> frame
    routing --> frame
    routing --> hmac
    routing --> admission
    halfchannel --> frame
    paths --> frame
    arq --> frame
    arq --> halfchannel
    replay --> frame
    replay --> halfchannel
    multipath --> frame
    multipath --> paths
    metrics --> paths
    tmux --> halfchannel
    tmux --> session
    discovery --> routing
    svtnmgmt --> admission
    svtnmgmt --> config
    drain --> routing
    sbctl --> metrics
    sbctl --> discovery
    sbctl --> svtnmgmt
    sbctl --> config
    main --> admission
    main --> routing
    main --> halfchannel
    main --> arq
    main --> replay
    main --> multipath
    main --> tmux
    main --> discovery
    main --> svtnmgmt
    main --> drain
    main --> config
    main --> metrics
```

## Topological Order (root → leaf)

Packages listed root-first. Any package may only import packages earlier in this list.

```
1.  internal/config         (no internal imports)
2.  internal/frame          (no internal imports)
3.  internal/hmac           (no internal imports)
4.  internal/admission      (imports: frame, hmac)
5.  internal/session        (imports: frame)
6.  internal/routing        (imports: frame, hmac, admission)
7.  internal/halfchannel    (imports: frame)
8.  internal/paths          (imports: frame)
9.  internal/arq            (imports: frame, halfchannel)
10. internal/replay         (imports: frame, halfchannel)
11. internal/multipath      (imports: frame, paths)
12. internal/metrics        (imports: paths)
13. internal/tmux           (imports: halfchannel, session)
14. internal/discovery      (imports: routing)
15. internal/svtnmgmt       (imports: admission, config)
16. internal/drain          (imports: routing)
17. cmd/sbctl               (imports: metrics, discovery, svtnmgmt, config)
18. cmd/switchboard         (imports: all above)
```

## Cycle-Freeness Verification

Mental topological sort: no package in positions 1–16 imports any package at a higher
position. Verification:

- `internal/routing` imports `admission` (position 4) — OK (routing is 6, admission is 4).
- `internal/tmux` imports `session` (position 5) — OK (tmux is 13, session is 5).
- `internal/discovery` imports `routing` (position 6) — OK (discovery is 14, routing is 6).
- `cmd/sbctl` imports `svtnmgmt` (position 15) — OK (sbctl is 17, svtnmgmt is 15).
- No back-edges. DAG is acyclic.

## Boundary Violation Rules

The following import patterns are **forbidden**:

| Forbidden Pattern | Reason |
|------------------|--------|
| `internal/routing` → `internal/tmux` | Router must not import session-content code |
| `internal/frame` → any other internal | Frame is a leaf; importing would create a cycle |
| `internal/hmac` → any other internal | HMAC is a leaf |
| Any package → `cmd/sbctl` | Commands are effectful tops; never imported by library code |
| Any package → `cmd/switchboard` | main is the top; never imported |

These are enforced by `go vet` (import cycle detection) and lint rules. Any CI
failure from import cycles is a P0 blocker.

## Notes on Deliberate Coupling

- `internal/routing` imports `internal/admission` because routing decisions depend
  on the admitted node set (SVTN partition). This is intentional — routing and
  admission are tightly coupled at the router boundary.
- `internal/session` is imported by both `internal/tmux` (access node enforces
  Tier 2) and `cmd/sbctl` (console control). The session package is a pure
  authorization boundary, not an I/O package, so this coupling is clean.

## §6 Import Constraints

The dependency graph in §§1–5 is a hard contract on import direction. The
following constraints apply to every Go file under `internal/`. This section
codifies what the compiler and `go vet` already enforce structurally and what
the consistency-validator audits at every wave gate.

### §6.1 Topological ordering (live packages, Wave-2 state)

Each package occupies a fixed position in the DAG. A package at position N may
only import packages at positions 1..N-1. The table below covers all `internal/`
packages present on `develop` at Wave-2 close (f35e836).

| Position | Package | Allowed imports | Classification |
|----------|---------|-----------------|----------------|
| 1 | `internal/frame` | ∅ (stdlib only) | pure-core |
| 2 | `internal/hmac` | ∅ (stdlib only) | pure-core |
| 3 | `internal/halfchannel` | {frame} | pure-core |
| 4 | `internal/admission` | {frame, hmac} | boundary |
| 5 | `internal/routing` | {frame, hmac, admission} | boundary |

Positions 6 and above are reserved for packages introduced in later waves; they
must be declared here before their first commit (see §6.4).

Verified against `grep -rn "switchboard/internal" --include="*.go" internal/ | grep -v _test.go`
at f35e836. No deviations found.

### §6.2 Forbidden edges

- `internal/frame` MUST NOT import any other `internal/` package.
- `internal/hmac` MUST NOT import any other `internal/` package.
- `internal/halfchannel` MUST NOT import `internal/admission` or `internal/routing`.
- `internal/admission` MUST NOT import `internal/routing`.
- No package may import a package at a higher position than itself.

### §6.3 Enforcement

- `go vet ./...` (run via `just lint`) catches cyclic imports at build time.
  Any import-cycle failure is a P0 CI blocker.
- The consistency-validator audits positional drift at every wave gate, verifying
  that no import edge exists outside the allowed set declared in §6.1.
- The adversary will flag any new import edge not declared in §6.1 as a finding
  requiring an explicit §6.4 declaration before the wave gate passes.

### §6.4 Adding a new internal package

New packages must, before their first commit to any branch:

1. Declare their position (1..N) in this section, extending the §6.1 table.
2. Declare their classification (pure-core vs boundary) per ARCH-09.
3. List their allowed imports explicitly in the §6.1 table.
4. Pass the consistency-validator check at the wave gate.

Undeclared packages discovered at the wave gate are an architecture violation.

---

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-06-23 | Initial dependency graph, topological order, and boundary violation rules |
| 1.1 | 2026-06-25 | Added §6 Import Constraints (§§6.1–6.4) — explicit codification of DAG positions, forbidden edges, enforcement mechanism, and new-package protocol; prompted by Wave-2 gate audit finding WAVE-2-MED-001 |
