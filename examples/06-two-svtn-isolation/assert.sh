#!/usr/bin/env bash
# assert.sh — example 06 proof: two teams cannot operate one another.
set -euo pipefail
source /usr/local/lib/switchboard-examples/harness.sh

A=/keys/team-a.key
B=/keys/team-b.key

echo "example 06 — two-svtn-isolation"
wait_for_socket /run/switchboard/router.sock

# Shared transport plane: BOTH teams are authorized on the router.
check ROUTER-ACCEPTS-A 0 "" -- \
  sbctl --target=/run/switchboard/router.sock --key="${A}" paths list
check ROUTER-ACCEPTS-B 0 "" -- \
  sbctl --target=/run/switchboard/router.sock --key="${B}" paths list

# Each team operates its own nodes...
check A-OPERATES-A1 0 "" -- sbctl --target=/run/switchboard/a1.sock --key="${A}" paths list
check A-OPERATES-A2 0 "" -- sbctl --target=/run/switchboard/a2.sock --key="${A}" paths list
check B-OPERATES-B1 0 "" -- sbctl --target=/run/switchboard/b1.sock --key="${B}" paths list
check B-OPERATES-B2 0 "" -- sbctl --target=/run/switchboard/b2.sock --key="${B}" paths list

# ...and is REFUSED by the other team's, with the stable taxonomy code.
# This is the isolation matrix: same commands, different key, hard denial.
check A-DENIED-ON-B1 1 "E-ADM-010" -- sbctl --target=/run/switchboard/b1.sock --key="${A}" paths list
check A-DENIED-ON-B2 1 "E-ADM-010" -- sbctl --target=/run/switchboard/b2.sock --key="${A}" paths list
check B-DENIED-ON-A1 1 "E-ADM-010" -- sbctl --target=/run/switchboard/a1.sock --key="${B}" paths list
check B-DENIED-ON-A2 1 "E-ADM-010" -- sbctl --target=/run/switchboard/a2.sock --key="${B}" paths list

# ── TARGET: SVTN-level isolation on the shared router, gated ──────────
# Two SVTNs on one router with per-SVTN session visibility (team-a's
# console cannot even see team-b's sessions, E-ADM-006 on cross-SVTN
# access). Requires external svtn.create + the network connector.
check_gated SVTN-A-CREATE 0 "svtn_id" -- \
  sbctl --target=/run/switchboard/router.sock --key="${A}" admin svtn create --name=team-a
check_gated SVTN-B-CREATE 0 "svtn_id" -- \
  sbctl --target=/run/switchboard/router.sock --key="${B}" admin svtn create --name=team-b
check_gated CROSS-SVTN-DENIED 1 "E-ADM" -- \
  sbctl --target=/run/switchboard/router.sock --key="${A}" sessions list --svtn=team-b

summary
