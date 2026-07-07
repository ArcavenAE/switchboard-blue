#!/usr/bin/env bash
# init.sh — one-shot: identities + console daemon config.
set -euo pipefail

GEN=/usr/local/lib/switchboard-examples/gen-identity.sh
"${GEN}" operator /keys
"${GEN}" rogue /keys

PEM_INDENTED="$(sed 's/^/      /' /keys/operator.pem)"
cat > /etc/switchboard/console.yaml <<EOF
listen_addr: "127.0.0.1:9090"

# Console-mode management sockets are TCP and MUST bind loopback
# (E-CFG-008) — which is why the driver shares this container's network
# namespace instead of using a shared unix-socket volume.
management_socket: "127.0.0.1:9091"
tick_interval: 10ms

authorized_operator_keys:
  - |
${PEM_INDENTED}
EOF

echo "init: wrote /etc/switchboard/console.yaml"
