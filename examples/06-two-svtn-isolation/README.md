# 06 — two-svtn-isolation

One shared router, two teams with two access nodes each, and **three
disjoint identities**. The claims under test:

1. *Tenants cannot operate each other's infrastructure* — team A's key
   opens nothing of team B's, and vice versa.
2. *Tenant and network-operator authority don't overlap* — the teams
   cannot manage the router; the network operator cannot manage the
   teams' nodes. Sharing transport does not mean sharing control.

## The roles in play

Role vocabulary per the domain spec's ubiquitous language (see
[docs/architecture.md — Who runs what](../../docs/architecture.md#who-runs-what--the-two-sides-of-the-trust-boundary)):

| Identity | Role | Administers |
|---|---|---|
| `netop` | **network operator** — provides the router infrastructure | the router |
| `team-a` | **SVTN operator** for team a | node-a1, node-a2 |
| `team-b` | **SVTN operator** for team b | node-b1, node-b2 |

This is the spec's trust boundary made runnable — *"the network
operator provides infrastructure; the customer holds the data keys"*
(carrier-grade content separation). The router is `netop`'s machine;
the tenants get its **data plane** (frame transport) but are strangers
to its **management plane**.

## Topology

```mermaid
graph TB
    R["router — netop's infrastructure<br/>mgmt: netop key ONLY<br/>data plane :9090: shared transport"]
    subgraph ta["team a — SVTN operator 'team-a'"]
        A1["node-a1<br/>tmux: top"]
        A2["node-a2<br/>tmux: watch date"]
    end
    subgraph tb["team b — SVTN operator 'team-b'"]
        B1["node-b1<br/>tmux: htop"]
        B2["node-b2<br/>tmux: vmstat 1"]
    end
    OP["operator container<br/>plays all three roles in turn"]
    OP -- "netop key → router mgmt ✓" --> R
    OP -- "team-a key → a-nodes ✓" --> A1 & A2
    OP -- "team-b key → b-nodes ✓" --> B1 & B2
    OP -. "team keys → router mgmt: E-ADM-010<br/>netop key → any node: E-ADM-010<br/>cross-team: E-ADM-010" .-> R
```

## Transaction under test — the authority matrix

```mermaid
sequenceDiagram
    participant O as operator container<br/>(three keys, three hats)
    participant R as router (netop's)
    participant A as node-a1 / node-a2
    participant B as node-b1 / node-b2

    Note over O,R: network operator's plane
    O->>R: router.status (netop.key)
    R-->>O: ok (exit 0)
    O->>R: router.status (team-a.key)
    R-->>O: E-ADM-010 (tenants don't manage routers)

    Note over O,B: transport is shared, control is not
    O->>R: TCP connect :9090 (data plane)
    R-->>O: accept — any tenant's frames may transit
    O->>A: paths.list (netop.key)
    A-->>O: E-ADM-010 (netop doesn't manage tenant nodes)

    Note over O,B: tenant vs tenant — the isolation matrix
    O->>A: paths.list (team-a.key)
    A-->>O: ok (exit 0)
    O->>B: paths.list (team-a.key)
    B-->>O: E-ADM-010 authentication failed (exit 1)
    O->>A: paths.list (team-b.key)
    A-->>O: E-ADM-010 authentication failed (exit 1)

    Note over O,R: TARGET (gated) — SVTN-level isolation on the shared router
    O->>R: admin.svtn.create --name=team-a / team-b ⊘ GATE-PENDING
    O->>R: sessions.list --svtn=team-b (team-a.key) ⊘ GATE-PENDING<br/>target: E-ADM-006 cross-SVTN denial
```

## What it proves today — authority by key, per plane

The operator container runs the *same commands* wearing each of the
three identities. Every off-role call is refused with
`E-ADM-010 authentication failed` — a hard, taxonomy-coded denial at
the Ed25519 challenge-response layer. The full matrix:

| key \ target | router mgmt | a-nodes | b-nodes | router data plane |
|---|---|---|---|---|
| `netop` | ✓ | ✗ | ✗ | ✓ (transport) |
| `team-a` | ✗ | ✓ | ✗ | ✓ (transport) |
| `team-b` | ✗ | ✗ | ✓ | ✓ (transport) |

This is isolation *by key configuration*, per daemon — the mechanism
the alpha actually ships.

## What's gated — SVTN-level isolation

The stronger claim in the example's name — two **SVTNs** on one router,
where team A's console cannot even *see* team B's sessions
(`E-ADM-006` on cross-SVTN access) — needs external `svtn.create` and
the network connector, both unshipped. Encoded as gated checks
(`GATE-PENDING` today; `GATED=1` makes them hard failures once the
milestone lands). When they flip, this compose file becomes the
acceptance test for multi-tenancy on a shared router.

## Setup + run

```bash
cd examples/06-two-svtn-isolation
docker compose up --build --exit-code-from operator
docker compose down -v
```

## Things to try

- **Wear one hat at a time:** `docker compose run --rm operator bash`,
  then walk the matrix by hand: `sbctl --target=/run/switchboard/b1.sock
  --key=/keys/team-a.key paths list` — watch the denial; swap the key
  and watch it pass. Try `netop.key` against the router, then against
  a node.
- **Verify the denial is auth-layer, not transport-layer:** the
  connection *opens* (no E-NET-001) and then authentication fails —
  the daemon is reachable but refuses you. Different failure depth than
  a firewall.
- **Grant cross-team access deliberately:** add team-b's PEM to
  `access-a1.yaml` in `init.sh`, re-up, and watch `B-DENIED-ON-A1`
  fail — the isolation is exactly as strong as the key list, which is
  the point.
