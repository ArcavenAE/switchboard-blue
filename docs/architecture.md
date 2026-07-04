# Switchboard Architecture

A high-level operator-oriented tour of how Switchboard is built. For
the CLI surface, see [docs/sbctl.md](sbctl.md); for the error taxonomy
see [docs/errors.md](errors.md).

---

## The problem

Two people (or an operator and an unattended agent) want to share a
tmux session over the network with:

- **Low latency** for keystrokes;
- **Multi-path resilience** — the connection survives one link going
  bad;
- **End-to-end encryption** — the transport intermediary cannot read
  the content;
- **Explicit trust** — who can see, drive, or manage the session is
  spelled out in named roles, not implicit from network position.

Switchboard is the transport plane. tmux is the session substrate; SSH
is the encryption; Switchboard glues them together with routing and
admission.

---

## The pieces

There are two kinds of process, both from the same `switchboard` binary:

### Nodes

**Access node** — publishes a local tmux session over the network.
Terminates SSH on both sides; the access node owns the PTY that tmux is
attached to.

**Console** — the interactive terminal endpoint. Runs on the operator's
laptop (or in a container, or under a headless agent runtime); receives
the terminal stream, sends keystrokes upstream.

**Control node** — runs the admission plane: registering keys,
creating SVTNs, revoking access. Purely a management surface — it does
not carry session traffic.

### Routers

Routers relay encrypted frames between nodes. They are **blind relays**:
they see frame envelopes and HMAC tags, but never the SSH-encrypted
payload. A single router binary supports three deployment modes:

| Mode | Role |
|------|------|
| **E** — Edge-local | Runs alongside a node on a single LAN. Fast path for two-machine setups. |
| **PE** — Provider Edge | Production router: connects nodes and peers with other routers. Runs on a jump host or in a datacenter. |
| **P** — Provider Core | Router-to-router only forwarding. Theoretical for now; not yet built. |

The mode is inferred at startup from the config file's
`upstream_routers:` field — an empty list means E, any entry means PE.

Nodes never talk to each other directly. Every frame passes through a
router — even in the E deployment, where the router happens to live on
the same LAN.

---

## SVTNs — Switched Virtual Networks

An **SVTN** is Switchboard's unit of trust and routing scope. It owns:

- A **bootstrap key** — set at creation, a permanent control-role trust
  anchor. Cannot be revoked or expired (`E-ADM-020`, `E-ADM-021`).
- An **admitted-key set** — the keys that are currently allowed to
  admit, drive, or observe traffic in this SVTN.
- A **namespace** for sessions, node addresses, and paths.

SVTNs are created via `sbctl admin svtn create --name=<...>`. The
returned `svtn_id` (hex, 8 bytes) is what appears in wire frames; the
name is what appears on operator commands (see the
[CLI reference](sbctl.md#sbctl-admin-svtn-create)).

Admission is role-based:

| Role | What it can do |
|------|----------------|
| `control` | Register keys, revoke keys, expire keys, destroy the SVTN, invoke management RPCs. |
| `console` | Attach to a session, receive terminal output, send keystrokes. Cannot register or revoke keys. |
| `access` | Publish a session. |

Role checks happen at two layers — an admission gate (are you in the
key set at all?) and an authority gate (does your role permit this
operation?). Role escalation is prevented by a cross-check against the
stored role at register time (`E-ADM-019`).

---

## Timeslice framing

Two design goals pull in different directions: keystrokes want tiny
frames delivered immediately, terminal output wants large frames
delivered efficiently. Switchboard reconciles them with **timeslice
framing** — "the bus leaves on time, full or not."

- Each direction (upstream, downstream) has its own clock.
- When a tick fires, whatever bytes are ready get bundled into one frame
  and sent.
- If nothing is ready, an empty frame is sent (heartbeat + timing
  witness).

This gives:

- Predictable jitter — the frame cadence is stable regardless of load.
- Cheap heartbeats — an empty frame doubles as liveness probe.
- Symmetric routing — every frame carries the same envelope shape.

---

## Asymmetric half-channels

Upstream and downstream traffic have different profiles:

| Channel | Content | Loss tolerance | Ordering |
|---------|---------|-----------------|----------|
| Upstream | Keystrokes | Loss-intolerant | Strict FIFO |
| Downstream | Terminal output | Bursty; screen state can be re-synced | Loose |

They are handled by **half-channel** state machines with independent
clocks, buffers, and retransmission policies. The console never blocks
its downstream receiver on upstream progress and vice versa.

---

## Multi-path routing

When more than one path is available between two nodes, Switchboard runs
a **duplicate-and-race** strategy:

- The sender emits the same frame on every viable path.
- The receiver deduplicates by frame id.
- Path quality is tracked (RTT, p99 RTT, loss) and surfaced via `sbctl paths list`.

Paths carry a status:

- `active` — currently used for forwarding.
- `degraded` — reachable but under-performing; kept as a backup.
- `failed` — reserved for a future release; MUST NOT appear in v0.1.0-rc.1.

Path metrics stabilize once ≥10 RTT samples have been collected. Before
that, `rtt_p99_ms` is emitted as the sentinel string `"pending"`
(see BC-2.06.003 EC-003 in the source specs).

---

## Admission tiers

Two admission checks happen for every session attach:

**Tier 1 — challenge/response.** The daemon issues a nonce; the caller
signs it with their private key; the daemon verifies the signature
against the SVTN's admitted-key set. Nonce replay is prevented
(`E-ADM-008`).

**Tier 2 — session authorization.** Even a fully-admitted console must
be authorized for the specific `<session-name>` on the specific
`<node-addr>` it wants to attach to. Failure emits `E-ADM-006`.

Read-only console attachment is possible — the upstream can reject
write operations without dropping the session, surfacing `E-ADM-007`
(degraded, session continues).

---

## Wire security

Every frame carries an HMAC tag keyed to the SVTN. Verification lives
inside the router and inside each receiving node. Failures are logged
and dropped (`E-ADM-002`, `E-ADM-016`). A sliding-window rate check
raises `E-ADM-017` (degraded) when a single source hits an HMAC failure
threshold — designed to surface active tampering without opening a
denial-of-service vector.

The SSH end-to-end tunnel is nested inside the Switchboard transport.
Routers verify the HMAC of the outer frame; they never see, decrypt, or
touch the SSH payload.

---

## Where the code lives

Internal packages, roughly by layer:

| Package | Layer |
|---------|-------|
| `internal/frame` | Frame encode/decode, header parsing. |
| `internal/hmac` | HMAC primitive. |
| `internal/admission` | Tier 1 admission gate, admitted-key set. |
| `internal/session` | Tier 2 session authorization, session lifecycle. |
| `internal/halfchannel` | Timeslice clock, upstream/downstream state machines. |
| `internal/paths` | Path ranking, RTT/loss metrics, keep-alive probes. |
| `internal/multipath` | Duplicate-and-race dispatch; receiver dedup. |
| `internal/discovery` | Presence advertisement, session enumeration. |
| `internal/routing` | Router path selection, HMAC gate. |
| `internal/svtnmgmt` | SVTN lifecycle and admitted-key state. |
| `internal/config` | Config file parsing, validation, reload. |
| `internal/metrics` | Quality-indicator computation, path metrics storage. |

The `cmd/switchboard/` entrypoint dispatches to daemon subcommands
(`router`, `access`, `console`, `control`); `cmd/sbctl/` is the
operator CLI.

---

## What's in v0.1.0-rc.1

The current MVP scope is **nodes + E router on a single LAN** —
proving out the edge protocol and user experience before tackling
multi-hop networking. Wire protocol, admission tiers, timeslice
framing, half-channels, path metrics, and the full admin key/svtn
lifecycle are all present. See [docs/sbctl.md — Unimplemented verbs
(PENDING)](sbctl.md#unimplemented-verbs-pending) for the verbs that
are spec-defined but not yet wired.

---

## Further reading

- [docs/getting-started.md](getting-started.md) — spin up an SVTN and connect.
- [docs/sbctl.md](sbctl.md) — full CLI reference.
- [docs/errors.md](errors.md) — error taxonomy.
- `.factory/specs/` — behavioral contracts, verification properties, and PRD supplements (spec-side canonical sources).
