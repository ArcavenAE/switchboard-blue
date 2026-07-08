#!/usr/bin/env bash
# init.sh — one-shot: generate operator + rogue identities, render the
# router config with the operator public key authorized.
set -euo pipefail

GEN=/usr/local/lib/switchboard-examples/gen-identity.sh
"${GEN}" operator /keys
"${GEN}" rogue /keys

PEM_INDENTED="$(sed 's/^/      /' /keys/operator.pem)"
cat > /etc/switchboard/router.yaml <<EOF
listen_addr: "0.0.0.0:9090"
management_socket: "/run/switchboard/router.sock"

# Timeslice tick — required. Allowed range: [5ms, 50ms].
tick_interval: 10ms

# E-mode: no upstream routers
upstream_routers: []

# Operator keys authorized on the management plane (SPKI PEM).
authorized_operator_keys:
  - |
${PEM_INDENTED}
EOF

echo "init: wrote /etc/switchboard/router.yaml"
