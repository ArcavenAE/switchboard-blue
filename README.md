# Switchboard

A low-latency, multi-path, end-to-end encrypted tmux session router. Switchboard establishes virtual switched networks (VSNs) over overlay routers, purpose-built for high-trust remote CLI access.

## Architecture

**Nodes** connect to tmux sessions; **routers** relay encrypted frames between them.

- **Access node** — publishes tmux sessions over the network
- **Console** — connects to remote tmux sessions
- **Control** — manages VSN configuration

Routers are blind relays — they forward SSH-encrypted traffic without seeing content. A single router binary supports three deployment modes:

| Mode | Role |
|------|------|
| **E** (Edge-local) | Runs alongside a node for same-LAN setup between two machines |
| **PE** (Provider Edge) | Production router: connects nodes and peers with other routers |
| **P** (Provider Core) | Router-to-router only forwarding (theoretical — not yet built) |

Nodes communicate end-to-end via SSH. Switchboard adds routing and network admission, not encryption.

## Key Design Principles

- **No direct node-to-node** — all traffic flows through routers
- **Timeslice framing** — "the bus leaves on time, full or not." Each direction has its own clock; frames carry whatever bytes are ready when the tick fires
- **Asymmetric half-channels** — upstream (keystrokes: tiny, ordered, loss-intolerant) and downstream (terminal output: bursty, state-syncable) are handled independently
- **Dual fastest-path forwarding** with latency-based path selection

## Status

**v0.1.0-rc.1** — release candidate. The current MVP scope is
**nodes + E router** on a single LAN, proving out the edge protocol and
user experience before tackling multi-hop networking.

## Documentation

- **[Getting Started](docs/getting-started.md)** — install, bootstrap an SVTN, publish and connect a tmux session (10 minutes).
- **[sbctl CLI Reference](docs/sbctl.md)** — every verb, flag, JSON envelope, and exit code.
- **[Architecture](docs/architecture.md)** — SVTNs, timeslice framing, half-channels, multi-path routing.
- **[Errors](docs/errors.md)** — the full error taxonomy with severity, exit codes, and handling notes.

## Install

Build from source (Go 1.25+ and [just](https://github.com/casey/just) required):

```bash
git clone https://github.com/arcavenae/switchboard.git
cd switchboard
just build
sudo install bin/switchboard bin/sbctl /usr/local/bin/
switchboard --version
```

Then follow [docs/getting-started.md](docs/getting-started.md).

## Build recipes

```bash
just build          # Build binary to bin/switchboard
just test           # Run tests
just test-race      # Run tests with race detector
just fmt            # Format with gofumpt
just lint           # Run golangci-lint
just build-all      # Cross-compile darwin/arm64, darwin/amd64, linux/amd64
just run            # Build and run directly
```

## Project Structure

```
cmd/switchboard/    # Entry point
internal/           # Internal packages (not yet populated)
scripts/            # macOS packaging (app, dmg, pkg)
packaging/          # Info.plist for macOS app bundle
Formula/            # Homebrew formula template
```

## License

MIT
