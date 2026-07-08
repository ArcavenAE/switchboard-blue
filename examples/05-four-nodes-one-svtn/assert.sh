#!/usr/bin/env bash
# assert.sh — example 05 proof: the full single-SVTN topology, alpha slice.
set -euo pipefail
source /usr/local/lib/switchboard-examples/harness.sh

OP=/keys/operator.key

echo "example 05 — four-nodes-one-svtn"
wait_for_socket /run/switchboard/router.sock
wait_for_tcp router 9090

# Six daemons, one compose network. Compose healthchecks already gate this
# operator on "4 tmux sessions alive + all sockets present + console TCP up";
# these checks add the authenticated round-trip to every unix-socket daemon.
check ROUTER-STATUS 0 "no active paths" -- \
  sbctl --target=/run/switchboard/router.sock --key="${OP}" router status
for n in 1 2 3 4; do
  check "NODE${n}-MGMT" 0 "" -- \
    sbctl --target="/run/switchboard/node${n}.sock" --key="${OP}" paths list
done

# Data plane reachable from the operator's namespace.
check DATA-PLANE-TCP 0 "" -- bash -c 'exec 3<>/dev/tcp/router/9090 && exec 3>&- 3<&-'

# ── TARGET flow (docs/getting-started.md §3-§6), gated ────────────────
# Everything below is the SVTN lifecycle this topology exists for. It
# requires (a) an external bootstrap path for svtn.create and (b) the
# access→router→console connector. Both are unshipped in this alpha.
# Run with GATED=1 to turn pending gates into failures once they ship.
check_gated SVTN-CREATE 0 "svtn_id" -- \
  sbctl --target=/run/switchboard/router.sock --key="${OP}" admin svtn create --name=hello-svtn
check_gated SESSIONS-ON-ROUTER 0 "node1" -- \
  sbctl --target=/run/switchboard/router.sock --key="${OP}" sessions list --svtn=hello-svtn

summary
