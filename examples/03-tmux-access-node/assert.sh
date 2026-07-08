#!/usr/bin/env bash
# assert.sh — example 03 proof: access daemon alongside a live tmux+top.
set -euo pipefail
source /usr/local/lib/switchboard-examples/harness.sh

SOCK=/run/switchboard/access.sock
OP=/keys/operator.key

echo "example 03 — tmux-access-node"
wait_for_socket "${SOCK}"

# The compose healthcheck already gates on "socket exists AND tmux session
# alive". Hold the claim for a few seconds: the daemon must SURVIVE with a
# session backend connected, not just bind and die (the macOS failure mode
# is bind → PTY error → exit within ~1s).
sleep 3
check DAEMON-SURVIVES 0 "" -- bash -c "test -S '${SOCK}'"

# Management plane over the access daemon's own socket.
check MGMT-PATHS-LIST 0 "" -- \
  sbctl --target="${SOCK}" --key="${OP}" paths list

# Role exclusion (ADR-004): access daemons never register admin handlers.
check ADMIN-NOT-ON-ACCESS 1 "unknown command" -- \
  sbctl --target="${SOCK}" --key="${OP}" admin svtn create --name=nope

# TARGET behavior: the published tmux session becomes visible to the
# management plane (per docs/getting-started.md §5 "sessions list").
# The sessions.* surface is console-mode-only in this alpha and the
# access→router upstream connector is not wired yet, so there is no
# RPC through which to observe the publication. Flips when session
# observability lands on the access daemon.
check_gated SESSIONS-VISIBLE 0 "work" -- \
  sbctl --target="${SOCK}" --key="${OP}" sessions status --session=work

summary
