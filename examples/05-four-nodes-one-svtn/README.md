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

```mermaid
graph TB
    subgraph net["compose network — one SVTN (target)"]
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
        C["console<br/>mgmt 127.0.0.1:9091"]
        D["operator (sbctl)"]
    end
    D -- "router.sock" --> R
    D -- "node1..4.sock" --> A1 & A2 & A3 & A4
    D -- "TCP :9090" --> R
    A1 & A2 & A3 & A4 -. "TARGET: publish sessions<br/>(connector unshipped)" .-> R
    C -. "TARGET: attach/switch<br/>(connector unshipped)" .-> R
```

Solid lines run today; dashed lines are the target data flow the gated
checks wait for. The management sockets (`nodeN.sock`, `router.sock`)
share one `run:` volume with the operator.

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
