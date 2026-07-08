#!/usr/bin/env bash
# assert.sh — example 04 proof: the console daemon's session-plane surface.
set -euo pipefail
source /usr/local/lib/switchboard-examples/harness.sh

TARGET=127.0.0.1:9091
OP=/keys/operator.key
ROGUE=/keys/rogue.key

echo "example 04 — console-surface"
wait_for_tcp 127.0.0.1 9091

# Authenticated operators get session-plane answers with stable taxonomy
# codes. Getting E-SES-* (not E-ADM-010) proves BOTH auth layers passed:
# Tier-1 challenge-response AND Tier-2 console-role admission.
check ATTACH-UNKNOWN-SESSION 1 "E-SES-001" -- \
  sbctl --target="${TARGET}" --key="${OP}" console attach --session=nope
check DETACH-NOT-ATTACHED 1 "E-SES-004" -- \
  sbctl --target="${TARGET}" --key="${OP}" console detach
check SESSIONS-STATUS-UNKNOWN 1 "E-SES-001" -- \
  sbctl --target="${TARGET}" --key="${OP}" sessions status --session=nope

# Unknown keys are refused at Tier 1.
check ROGUE-DENIED 1 "E-ADM-010" -- \
  sbctl --target="${TARGET}" --key="${ROGUE}" console attach --session=nope

# Usage errors are caught client-side (exit 2, no RPC).
check ATTACH-REQUIRES-SESSION 2 "--session is required" -- \
  sbctl --target="${TARGET}" --key="${OP}" console attach

# TARGET behavior (docs/getting-started.md §5-6): sessions published by
# access nodes appear here, and attach succeeds. Requires the
# access→router→console connector, which is not wired in this alpha —
# nothing can populate this console's session registry from outside.
check_gated SESSIONS-LIST-TARGET 0 "" -- \
  sbctl --target="${TARGET}" --key="${OP}" sessions list

summary
