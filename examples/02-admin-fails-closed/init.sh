#!/usr/bin/env bash
# init.sh — one-shot: identities + control daemon config.
set -euo pipefail

GEN=/usr/local/lib/switchboard-examples/gen-identity.sh
"${GEN}" operator /keys
"${GEN}" rogue /keys

PEM_INDENTED="$(sed 's/^/      /' /keys/operator.pem)"
cat > /etc/switchboard/control.yaml <<EOF
listen_addr: "127.0.0.1:9090"
management_socket: "/run/switchboard/control.sock"
tick_interval: 10ms

authorized_operator_keys:
  - |
${PEM_INDENTED}
EOF

echo "init: wrote /etc/switchboard/control.yaml"
