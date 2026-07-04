# Getting Started with Switchboard

A ten-minute walkthrough: install Switchboard, start a router in E-mode,
create an SVTN, publish a tmux session from one machine, and connect a
console from another.

**Audience:** operators bringing up their first Switchboard deployment.
**Target release:** v0.1.0-rc.1.

If you're looking for the full CLI surface, jump to
[docs/sbctl.md](sbctl.md); for the concepts behind the pieces you're
about to install, see [docs/architecture.md](architecture.md).

---

## Prerequisites

You need:

- **Go 1.25+** to build from source. Homebrew installs will be published
  from tagged releases when this milestone stabilizes; for now, build from
  the tag.
- **[just](https://github.com/casey/just)** — the task runner used by
  the build. `brew install just` on macOS.
- **tmux** on the machine that will host the session.
- **An Ed25519 SSH key pair** for each participant (operator, access
  node, console). `ssh-keygen -t ed25519` produces one.

Two machines make the tutorial more interesting, but everything works on
one machine with two terminals if that's what you have.

---

## 1. Build

Clone and build:

```bash
git clone https://github.com/arcavenae/switchboard.git
cd switchboard
just build
```

This produces `bin/switchboard` (the daemon) and `bin/sbctl` (the
operator CLI). Both binaries are single-file — copy or symlink them
onto `$PATH` on each machine that needs them:

```bash
sudo install bin/switchboard bin/sbctl /usr/local/bin/
```

Confirm:

```bash
switchboard --version
```

---

## 2. Start a router

The router is the transport plane. In E-mode (edge-local, single-LAN
deployment) it needs almost no config.

Write `switchboard-router.yaml`:

```yaml
listen_addr: "0.0.0.0:9090"
management_socket: "/run/switchboard-router.sock"

# E-mode: no upstream routers
upstream_routers: []
```

Start the daemon:

```bash
sudo switchboard router --config switchboard-router.yaml
```

The router logs its listen address and management socket path on stdout.
Leave it running.

---

## 3. Bootstrap: create your first SVTN

The very first management call to a fresh daemon must use the daemon's
**bootstrap key**. The daemon prints a bootstrap public key on first
startup; store its matching private key in a safe place — you will
need it to create SVTNs.

From an operator machine that can reach the router:

```bash
sbctl \
  --target=/run/switchboard-router.sock \
  --key=~/.ssh/switchboard-bootstrap \
  admin svtn create --name=hello-svtn
```

You should see:

```
SVTN created:
  svtn_id: a1b2c3d4e5f60102
  bootstrap_fingerprint: SHA256:...
```

Save the `svtn_id`; you will paste its short-id prefix into confirmation
prompts later. The `bootstrap_fingerprint` is what SVTN control keys
verify against for emergency recovery.

Add your day-to-day operator key as a control-role key in the SVTN:

```bash
sbctl \
  --key=~/.ssh/switchboard-bootstrap \
  admin key register \
    --svtn=hello-svtn \
    --key="$(cat ~/.ssh/id_ed25519.pub)" \
    --role=control \
    --confirm=<paste svtn short-id>
```

From now on you can use `~/.ssh/id_ed25519` for admin work; keep the
bootstrap key offline as your recovery credential.

---

## 4. Add an access node key and a console key

The **access node** publishes tmux sessions. The **console** attaches to
them. Each needs its own key registered in the SVTN with the appropriate
role:

```bash
# Access node
sbctl \
  --key=~/.ssh/id_ed25519 \
  admin key register \
    --svtn=hello-svtn \
    --key="$(ssh-keygen -y -f ~/.ssh/switchboard-access)" \
    --role=access \
    --confirm=<svtn short-id>

# Console (operator laptop)
sbctl \
  --key=~/.ssh/id_ed25519 \
  admin key register \
    --svtn=hello-svtn \
    --key="$(ssh-keygen -y -f ~/.ssh/switchboard-console)" \
    --role=console \
    --confirm=<svtn short-id>
```

Confirm the key set:

```bash
sbctl --key=~/.ssh/id_ed25519 admin list-keys --svtn=hello-svtn
```

You should see three entries: `control`, `access`, `console`.

---

## 5. Publish a tmux session (access node side)

On the machine that will host tmux, start a session:

```bash
tmux new -s work
# ...do some work in the session...
```

In another terminal on the same machine, start the access daemon:

```yaml
# switchboard-access.yaml
upstream_router: "10.0.0.1:9090"        # the router's listen addr
node_key: "/etc/switchboard/access.key"
svtn: "hello-svtn"
```

```bash
switchboard access --config switchboard-access.yaml
```

The access node authenticates to the router (using
`/etc/switchboard/access.key`), attaches to the running tmux server, and
advertises its published sessions. `work` will now appear in the SVTN's
session list.

Verify from the operator machine:

```bash
sbctl --key=~/.ssh/id_ed25519 sessions list --svtn=hello-svtn
```

You should see `work` listed.

---

## 6. Connect a console

On the operator laptop:

```yaml
# switchboard-console.yaml
upstream_router: "10.0.0.1:9090"
node_key: "/home/me/.ssh/switchboard-console"
svtn: "hello-svtn"
```

```bash
switchboard console --config switchboard-console.yaml
```

The console daemon dials the router, authenticates, and idles. In
another terminal, attach to the remote session:

```bash
sbctl --key=~/.ssh/switchboard-console console attach --session=work
```

Your terminal is now driving the remote tmux session. Detach with:

```bash
sbctl console detach
```

Congratulations — you have a working Switchboard SVTN.

---

## 7. Look at what the network sees

Observe path health from the operator side:

```bash
sbctl --key=~/.ssh/id_ed25519 paths list --svtn=hello-svtn
sbctl --key=~/.ssh/id_ed25519 router metrics --svtn=hello-svtn
```

`rtt_p99_ms` may show `"pending"` for the first few seconds — that's
expected until ten RTT samples have been collected. See
[docs/architecture.md — Multi-path routing](architecture.md#multi-path-routing).

---

## Tearing down

Revoke the console key when you're done:

```bash
sbctl --key=~/.ssh/id_ed25519 admin key revoke \
  --svtn=hello-svtn \
  --key="$(ssh-keygen -y -f ~/.ssh/switchboard-console)" \
  --role=console
```

Or destroy the whole SVTN (requires the confirmation gate):

```bash
sbctl --key=~/.ssh/id_ed25519 admin svtn destroy \
  --name=hello-svtn \
  --confirm=<svtn short-id>
```

`--confirm=<svtn short-id>` guards against typos; a non-interactive
script can pass `--yes` instead, but never both — see
[docs/sbctl.md — Confirmation and non-interactive use](sbctl.md#confirmation-and-non-interactive-use).

---

## Common pitfalls

- **`E-NET-001` on the first sbctl command** — the router isn't
  listening, or `--target` doesn't point where you think. Check
  `management_socket` in the router config.
- **`E-ADM-010`** — the operator key is not registered in the SVTN.
  Confirm with `sbctl admin list-keys`.
- **`E-CFG-013`** — a scripted invocation reached a confirmation gate.
  Either pass `--confirm=<svtn short-id>` (the safe form) or `--yes`
  (bypass with warning).
- **`E-CFG-008` on console-mode startup** — a console-mode management
  socket bound to a non-loopback TCP address. Use a Unix socket or a
  loopback bind (`127.0.0.1:<port>`).

Every error carries a stable taxonomy code — see [docs/errors.md](errors.md)
for the full catalog and their handling recommendations.

---

## Next steps

- Skim [docs/architecture.md](architecture.md) to understand SVTNs,
  timeslice framing, half-channels, and multi-path routing.
- Read the full [docs/sbctl.md](sbctl.md) reference for every verb, flag,
  and JSON schema.
- Contribute — [CONTRIBUTING.md](../CONTRIBUTING.md) covers dev workflow,
  commit conventions, and CI.
