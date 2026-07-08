#!/usr/bin/env bash
# assert.sh — example 06 proof: role separation on shared infrastructure.
#
# Three identities, two human roles:
#   netop           — network operator: administers the router
#   team-a / team-b — two SVTN operators (tenants): administer their nodes
set -euo pipefail
source /usr/local/lib/switchboard-examples/harness.sh

NETOP=/keys/netop.key
A=/keys/team-a.key
B=/keys/team-b.key

echo "example 06 — two-svtn-isolation"
wait_for_socket /run/switchboard/router.sock
wait_for_tcp router 9090

# The router is the NETWORK operator's infrastructure: only netop
# manages it. Tenants are denied on the router's management plane...
check NETOP-MANAGES-ROUTER 0 "" -- \
  sbctl --target=/run/switchboard/router.sock --key="${NETOP}" router status
check TEAM-A-DENIED-ROUTER-MGMT 1 "E-ADM-010" -- \
  sbctl --target=/run/switchboard/router.sock --key="${A}" router status
check TEAM-B-DENIED-ROUTER-MGMT 1 "E-ADM-010" -- \
  sbctl --target=/run/switchboard/router.sock --key="${B}" router status

# ...while the router's DATA plane (transport) is shared by everyone.
check DATA-PLANE-SHARED 0 "" -- bash -c 'exec 3<>/dev/tcp/router/9090 && exec 3>&- 3<&-'

# Conversely, the network operator holds no authority over tenant nodes.
check NETOP-DENIED-ON-A1 1 "E-ADM-010" -- sbctl --target=/run/switchboard/a1.sock --key="${NETOP}" paths list
check NETOP-DENIED-ON-B1 1 "E-ADM-010" -- sbctl --target=/run/switchboard/b1.sock --key="${NETOP}" paths list

# Each SVTN operator operates its own nodes...
check A-OPERATES-A1 0 "" -- sbctl --target=/run/switchboard/a1.sock --key="${A}" paths list
check A-OPERATES-A2 0 "" -- sbctl --target=/run/switchboard/a2.sock --key="${A}" paths list
check B-OPERATES-B1 0 "" -- sbctl --target=/run/switchboard/b1.sock --key="${B}" paths list
check B-OPERATES-B2 0 "" -- sbctl --target=/run/switchboard/b2.sock --key="${B}" paths list

# ...and is REFUSED by the other tenant's, with the stable taxonomy code.
# This is the isolation matrix: same commands, different key, hard denial.
check A-DENIED-ON-B1 1 "E-ADM-010" -- sbctl --target=/run/switchboard/b1.sock --key="${A}" paths list
check A-DENIED-ON-B2 1 "E-ADM-010" -- sbctl --target=/run/switchboard/b2.sock --key="${A}" paths list
check B-DENIED-ON-A1 1 "E-ADM-010" -- sbctl --target=/run/switchboard/a1.sock --key="${B}" paths list
check B-DENIED-ON-A2 1 "E-ADM-010" -- sbctl --target=/run/switchboard/a2.sock --key="${B}" paths list

# ── TARGET: SVTN-level isolation on the shared router, gated ──────────
# Two SVTNs on one router with per-SVTN session visibility (team-a's
# console cannot even see team-b's sessions, E-ADM-006 on cross-SVTN
# access). Requires external svtn.create + the network connector.
# CROSS-SVTN-DENIED requires E-ADM-006 SPECIFICALLY: today the same
# call fails earlier with E-ADM-010 (tenants aren't Tier-1 admitted on
# the router's mgmt plane at all), which is the right behavior for the
# alpha but not the SVTN-scoped denial this gate is waiting for.
check_gated SVTN-A-CREATE 0 "svtn_id" -- \
  sbctl --target=/run/switchboard/router.sock --key="${A}" admin svtn create --name=team-a
check_gated SVTN-B-CREATE 0 "svtn_id" -- \
  sbctl --target=/run/switchboard/router.sock --key="${B}" admin svtn create --name=team-b
check_gated CROSS-SVTN-DENIED 1 "E-ADM-006" -- \
  sbctl --target=/run/switchboard/router.sock --key="${A}" sessions list --svtn=team-b

summary
