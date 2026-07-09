# 05 — four-nodes-one-svtn

The target topology of the getting-started tutorial, at full width: one
router, **four access nodes** each hosting a different live program in
tmux, and **one console** — seven containers on one compose network.

| Node | Program | Why |
|---|---|---|
| node1 | `top` | classic full-screen TUI, constant redraws |
| node2 | `htop` | heavier TUI (colors, meters, per-cell updates) |
| node3 | `watch -n1 date` | full-screen refresh on an interval |
| node4 | `vmstat 1` | scrolling line output (non-TUI contrast case) |

## Topology

### The network view

This is the point of the whole architecture, at full width: **one
console driving sessions on four machines through one blind carrier.**
The SVTN's session directory works like a routing table, except the
routes are tmux sessions — each access node publishes what it hosts,
and the console asks one place "what can I attach to?" and gets an
answer spanning every machine in the SVTN.

```mermaid
graph LR
    subgraph svtn["one SVTN — one trust + routing scope"]
        direction LR
        CN["console — live<br/>one screen + keyboard<br/>for every session in the table"]
        DIR["session directory —<br/>a routing table of tmux sessions:<br/>top@node1 · htop@node2<br/>watch@node3 · vmstat@node4"]
        R["router — blind relay<br/>carries every circuit,<br/>reads none"]
        subgraph work["four machines hosting the work — all live"]
            direction LR
            A1["access node1 — tmux: top"]
            A2["access node2 — tmux: htop"]
            A3["access node3 — tmux: watch date"]
            A4["access node4 — tmux: vmstat 1"]
        end
        CN -. "sessions list —<br/>reads the table" .-> DIR
        CN -. "attach / switch by name,<br/>keystrokes out" .-> R
        R -. "terminal output back" .-> CN
        DIR -. "maintained from<br/>node publications" .- R
        R -. circuit .- A1
        R -. circuit .- A2
        R -. circuit .- A3
        R -. circuit .- A4
    end
```

Every process in this drawing runs today — four access daemons each
holding a live tmux backend, the router, the console. The joins are
dotted because they are the gated milestone: the connector that lets
daemons dial each other is unshipped, so the frames don't traverse yet.
This compose file is built to become the connector's acceptance test
without changing shape — the day the dotted lines go solid, the gated
checks flip to `GATE-PASS`.

### Ground level — the compose plumbing

What the assertions actually drive today: an authenticated management
round-trip to all five daemons, from one operator container.

```mermaid
graph LR
    subgraph net["compose network — one SVTN (target)"]
        D["operator (sbctl)"]
        R["router<br/>data plane :9090<br/>mgmt router.sock"]
        subgraph n1["node1"]
            A1["access daemon"] --- T1["tmux: top"]
        end
        subgraph n2["node2"]
            A2["access daemon"] --- T2["tmux: htop"]
        end
        subgraph n3["node3"]
            A3["access daemon"] --- T3["tmux: watch date"]
        end
        subgraph n4["node4"]
            A4["access daemon"] --- T4["tmux: vmstat 1"]
        end
        C["console<br/>mgmt 127.0.0.1:9091<br/>(idle until the connector)"]
    end
    D -- "mgmt: router.sock" --> R
    D -- "data plane: TCP :9090" --> R
    D -- "mgmt: node1.sock" --> A1
    D -- "mgmt: node2.sock" --> A2
    D -- "mgmt: node3.sock" --> A3
    D -- "mgmt: node4.sock" --> A4
```

The operator initiates everything from the left: five authenticated
management round-trips (one per daemon) plus a raw data-plane reach.
The management sockets (`nodeN.sock`, `router.sock`) share one `run:`
volume with the operator; the console's mgmt is loopback-TCP-only, so
it runs unprobed here (that surface is example 04's lab).

## Transaction under test

```mermaid
sequenceDiagram
    participant D as operator (sbctl)
    participant R as router
    participant N as node1..node4<br/>(4 access daemons)

    Note over D,N: alpha slice — every daemon individually proven
    D->>R: TCP connect :9090 (data plane)
    R-->>D: accept
    D->>R: router.status (operator.key)
    R-->>D: "no active paths" (exit 0)
    loop for each node 1..4
        D->>N: paths.list via nodeN.sock (challenge-response)
        N-->>D: authenticated RPC answer (exit 0)
    end

    Note over D,R: TARGET flow (gated) — the SVTN lifecycle
    D->>R: admin.svtn.create --name=hello-svtn ⊘ GATE-PENDING
    R-->>D: unknown command (admin handlers not on router)
    D->>R: sessions.list --svtn=hello-svtn ⊘ GATE-PENDING
    R-->>D: unknown command (connector unshipped)
```

## What it proves today

- All seven services come up and **stay** up: per-node compose
  healthchecks require the tmux session alive *and* the access daemon's
  management socket present, and the operator only starts when every
  daemon is healthy. Four access daemons each holding a live session
  backend is the widest access-mode exercise the alpha has had.
- The operator completes an authenticated management round-trip to the
  router and **each of the four nodes** (five key-based
  challenge-responses against five separate daemons).
- The router's data plane is reachable from the operator's namespace.

## What's gated (the point of the example)

The SVTN lifecycle this topology exists for — `admin svtn create`,
registering node/console keys, `sessions list` showing `node1..node4`,
console attach/switch across nodes — needs two pieces the alpha hasn't
shipped: an external bootstrap path for `svtn.create` (S-6.02) and the
access→router→console connector. Those assertions run as *gated checks*:
`GATE-PENDING` today, `GATE-PASS` the day the wiring lands, hard
failures under `GATED=1` (CI mode for the post-connector world). This
compose file is intended to become the acceptance test for that
milestone without changing shape.

## Setup + run

```bash
cd examples/05-four-nodes-one-svtn
docker compose up --build --exit-code-from operator
docker compose down -v
```

## Things to try

- **Tour the fleet:** `docker compose exec node2 tmux attach -t node2`
  (htop; detach with Ctrl-b d), then the same for node1/3/4 — four
  different programs running under four different access daemons.
- **Simulate a node failure:** `docker compose exec node3 tmux
  kill-server` and watch node3's healthcheck flip to unhealthy while
  the other six services stay green — per-node blast radius.
- **Ask every daemon who it is:** loop `sbctl --target=/run/switchboard/nodeN.sock
  --key=/keys/operator.key paths list` from `docker compose run --rm operator bash`.
- **Preview the future:** run with `GATED=1 docker compose up ...` to
  see exactly which target behaviors the alpha still refuses — the
  same list a release manager would check before calling the connector
  milestone done.
