#!/usr/bin/env bash
# init.sh — one-shot: identities + configs for router, 4 access nodes,
# and 1 console — the target single-SVTN topology.
set -euo pipefail

GEN=/usr/local/lib/switchboard-examples/gen-identity.sh
"${GEN}" operator /keys
"${GEN}" rogue /keys

PEM_INDENTED="$(sed 's/^/      /' /keys/operator.pem)"

cat > /etc/switchboard/router.yaml <<EOF
listen_addr: "0.0.0.0:9090"
management_socket: "/run/switchboard/router.sock"
tick_interval: 10ms
upstream_routers: []
authorized_operator_keys:
  - |
${PEM_INDENTED}
EOF

for n in 1 2 3 4; do
  cat > "/etc/switchboard/access-node${n}.yaml" <<EOF
listen_addr: "127.0.0.1:9090"
management_socket: "/run/switchboard/node${n}.sock"
tick_interval: 10ms
authorized_operator_keys:
  - |
${PEM_INDENTED}
EOF
done

cat > /etc/switchboard/console.yaml <<EOF
listen_addr: "127.0.0.1:9089"
management_socket: "127.0.0.1:9091"
tick_interval: 10ms
authorized_operator_keys:
  - |
${PEM_INDENTED}
EOF

echo "init: wrote router + 4 node + console configs"
