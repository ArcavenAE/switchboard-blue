# 06 — two-svtn-isolation

One shared router, two teams with two access nodes each, and **disjoint
operator keys**. The claim under test: *team A cannot operate team B's
infrastructure, and vice versa* — while both share the same transport
plane.

## Topology

```mermaid
graph TB
    R["router — shared transport plane<br/>authorized: team-a AND team-b keys"]
    subgraph ta["team a (key: team-a only)"]
        A1["node-a1<br/>tmux: top"]
        A2["node-a2<br/>tmux: watch date"]
    end
    subgraph tb["team b (key: team-b only)"]
        B1["node-b1<br/>tmux: htop"]
        B2["node-b2<br/>tmux: vmstat 1"]
    end
    D["operator (sbctl)<br/>holds BOTH keys, plays both roles"]
    D -- "team-a key ✓" --> A1 & A2
    D -- "team-b key ✓" --> B1 & B2
    D -- "either key ✓" --> R
    D -. "team-a key on b* → E-ADM-010<br/>team-b key on a* → E-ADM-010" .-> R
```

## Transaction under test — the isolation matrix

```mermaid
sequenceDiagram
    participant D as operator (sbctl)
    participant R as router
    participant A as node-a1 / node-a2
    participant B as node-b1 / node-b2

    Note over D,R: shared transport — both teams authorized on the router
    D->>R: paths.list (team-a.key)
    R-->>D: ok (exit 0)
    D->>R: paths.list (team-b.key)
    R-->>D: ok (exit 0)

    Note over D,B: own-team operation succeeds
    D->>A: paths.list (team-a.key)
    A-->>D: ok (exit 0)
    D->>B: paths.list (team-b.key)
    B-->>D: ok (exit 0)

    Note over D,B: cross-team operation is REFUSED — same command, other key
    D->>B: paths.list (team-a.key)
    B-->>D: E-ADM-010 authentication failed (exit 1)
    D->>A: paths.list (team-b.key)
    A-->>D: E-ADM-010 authentication failed (exit 1)

    Note over D,R: TARGET (gated) — SVTN-level isolation on the shared router
    D->>R: admin.svtn.create --name=team-a / team-b ⊘ GATE-PENDING
    D->>R: sessions.list --svtn=team-b (team-a.key) ⊘ GATE-PENDING<br/>target: E-ADM-006 cross-SVTN denial
```

## What it proves today — the isolation matrix

The operator runs the *same command* against all four nodes with both
keys. Own-team calls succeed; cross-team calls are refused with
`E-ADM-010 authentication failed` — a hard, taxonomy-coded denial at the
Ed25519 challenge-response layer, not an unadvertised absence. The
router accepts both teams, demonstrating that sharing transport does not
imply sharing operational authority.

This is isolation *by key configuration*, per daemon. It is the
mechanism the alpha actually ships.

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

- **Be team A for a while:** `docker compose run --rm operator bash`, then
  walk the matrix by hand: `sbctl --target=/run/switchboard/b1.sock
  --key=/keys/team-a.key paths list` — watch the denial; swap the key
  and watch it pass.
- **Verify the denial is auth-layer, not transport-layer:** the
  connection *opens* (no E-NET-001) and then authentication fails —
  the daemon is reachable but refuses you. Different failure depth than
  a firewall.
- **Grant cross-team access deliberately:** add team-b's PEM to
  `access-a1.yaml` in `init.sh`, re-up, and watch `B-DENIED-ON-A1`
  fail — the isolation is exactly as strong as the key list, which is
  the point.
