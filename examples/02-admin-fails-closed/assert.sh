#!/usr/bin/env bash
# assert.sh — example 02 proof: the admin plane's authority model.
set -euo pipefail
source /usr/local/lib/switchboard-examples/harness.sh

SOCK=/run/switchboard/control.sock
OP=/keys/operator.key
ROGUE=/keys/rogue.key

echo "example 02 — admin-fails-closed"
wait_for_socket "${SOCK}"

# Layer 1 (mgmt auth): the configured operator key passes the Ed25519
# challenge-response — proven by getting a LAYER-2 answer (E-ADM-009),
# not an authentication failure.
check SVTN-CREATE-BOOTSTRAP-ONLY 1 "E-ADM-009" -- \
  sbctl --target="${SOCK}" --key="${OP}" admin svtn create --name=hello
check SVTN-CREATE-ROLE-REPORTED 1 "has role unregistered" -- \
  sbctl --target="${SOCK}" --key="${OP}" admin svtn create --name=hello
check LIST-KEYS-DENIED 1 "E-ADM-009" -- \
  sbctl --target="${SOCK}" --key="${OP}" admin list-keys --svtn=hello

# Layer 1 fails closed for unknown keys: stable code, no panic.
check ROGUE-DENIED 1 "E-ADM-010" -- \
  sbctl --target="${SOCK}" --key="${ROGUE}" admin svtn create --name=hello

# TARGET behavior (docs/getting-started.md §3): an operator bootstrap path
# for SVTN creation. In this alpha the sole svtn.create authority is the
# daemon's own ephemeral in-process key, so no external caller can create
# an SVTN. When persistent bootstrap-key wiring ships (S-6.02), this
# check flips to GATE-PASS and should be promoted to a hard check.
check_gated SVTN-CREATE-TARGET 0 "svtn_id" -- \
  sbctl --target="${SOCK}" --key="${OP}" admin svtn create --name=hello

summary
