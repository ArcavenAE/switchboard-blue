# Switchboard Examples

A ladder of docker-compose proofs of functionality, from a single router
to multi-team topologies. Each example is self-contained: a README with
setup, use, and things to try; a compose file; and a driver container
whose exit code is the verdict (`docker compose up --build
--exit-code-from driver`).

**The examples install the published alpha binaries** from GitHub
Releases inside the containers — they prove the shipped artifacts, not
the working tree. Pin a release with
`SWITCHBOARD_RELEASE=<tag> docker compose build` (default is the tag the
examples were last verified against; see `_shared/Dockerfile`).

## The ladder

| Example | Topology | Proves |
|---|---|---|
| [01-hello-router](01-hello-router/) | router + driver | daemon lifecycle, cross-namespace data plane, authenticated mgmt RPC, fail-closed auth, role exclusion |
| [02-admin-fails-closed](02-admin-fails-closed/) | control + driver | two-layer authority model; `svtn.create` bootstrap-only; stable denial taxonomy |
| [03-tmux-access-node](03-tmux-access-node/) | access + tmux(`top`) + driver | access daemon survives with a live session backend — **not testable on macOS dev machines** |
| [04-console-surface](04-console-surface/) | console + driver (shared netns) | session-plane RPC surface, two-tier admission, loopback-only console mgmt |
| [05-four-nodes-one-svtn](05-four-nodes-one-svtn/) | router + 4 nodes (top/htop/watch/vmstat) + console + driver | the target single-SVTN topology at full width; gated SVTN lifecycle |
| [06-two-svtn-isolation](06-two-svtn-isolation/) | router + 2×2 team nodes + driver | teams with disjoint keys cannot operate each other; gated SVTN-level isolation |

## Current-alpha honesty: gated checks

The getting-started tutorial targets v0.1.0-rc.1. Two pieces of the
distributed story are not wired in the current alpha:

1. **External SVTN bootstrap** — `admin svtn create` is bootstrap-only
   and the daemon's bootstrap key is ephemeral/in-process (persistent
   key wiring is S-6.02), so no external caller can create an SVTN yet.
2. **The network connector** — no daemon dials another daemon yet.
   Access nodes connect to *local* tmux; routers listen; consoles idle.
   Sessions cannot traverse access→router→console.

Assertions that depend on those pieces are **gated checks**
(`check_gated` in `_shared/harness.sh`): they report `GATE-PENDING`
today, flip to `GATE-PASS` when the feature lands, and become hard
failures under `GATED=1`. The topology examples (05, 06) are designed to
turn into the acceptance tests for the connector milestone without
changing shape.

## Layout

```
examples/
  _shared/          Dockerfile (fetches release binaries), gen-identity.sh,
                    harness.sh (check / check_gated / summary)
  NN-<name>/        README.md, docker-compose.yml, init.sh (keys+configs),
                    assert.sh (the driver's proof), node-entry.sh (where nodes exist)
```

Conventions:

- **Identities** are generated per run into a compose volume by
  `_shared/gen-identity.sh` — an OpenSSH-format private key for
  `sbctl --key` and an SPKI PEM for `authorized_operator_keys` (the two
  formats are not interchangeable in this alpha; see the script header).
- **Unix management sockets** are shared with the driver via a `run:`
  volume; console mgmt is loopback-TCP-only, so console drivers use
  `network_mode: "service:console"` instead.
- **Assertions** are behavioral only — exit codes and substrings
  (taxonomy codes like `E-ADM-010`), never byte-exact output, matching
  the discipline in `test/smoke/`.
- **Teardown** is `docker compose down -v`; the `-v` clears generated
  keys and configs.
