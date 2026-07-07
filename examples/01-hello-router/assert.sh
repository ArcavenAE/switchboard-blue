#!/usr/bin/env bash
# assert.sh — example 01 proof: router daemon + authenticated mgmt plane.
set -euo pipefail
source /usr/local/lib/switchboard-examples/harness.sh

SOCK=/run/switchboard/router.sock
OP=/keys/operator.key
ROGUE=/keys/rogue.key

echo "example 01 — hello-router"
switchboard --version
sbctl --version
echo

wait_for_socket "${SOCK}"
wait_for_tcp router 9090

# The driver runs in a different network namespace than the router — this
# TCP reachability check is the "second machine" proof a single-host smoke
# can't give you.
check DATA-PLANE-TCP 0 "" -- bash -c 'exec 3<>/dev/tcp/router/9090 && exec 3>&- 3<&-'

# Ed25519 challenge-response with the configured operator key, then a real
# RPC round-trip. "no active paths" is the documented empty-state answer.
check MGMT-AUTH-STATUS 0 "no active paths" -- \
  sbctl --target="${SOCK}" --key="${OP}" router status
check MGMT-PATHS-LIST 0 "" -- \
  sbctl --target="${SOCK}" --key="${OP}" paths list
check MGMT-JSON 0 "" -- \
  bash -c "sbctl --json --target='${SOCK}' --key='${OP}' router status | jq -e type"

# Auth fails closed: a key NOT in authorized_operator_keys is rejected with
# the stable taxonomy code, not a panic or a bare transport error.
check AUTH-ROGUE-DENIED 1 "E-ADM-010" -- \
  sbctl --target="${SOCK}" --key="${ROGUE}" router status

# Role exclusion (ADR-004): router daemons do not register admin handlers.
check ADMIN-NOT-ON-ROUTER 1 "unknown command" -- \
  sbctl --target="${SOCK}" --key="${OP}" admin svtn create --name=nope

summary
