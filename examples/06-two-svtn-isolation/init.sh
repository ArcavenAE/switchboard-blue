#!/usr/bin/env bash
# init.sh — one-shot: two disjoint team identities + configs for one
# shared router and two per-team node sets.
set -euo pipefail

GEN=/usr/local/lib/switchboard-examples/gen-identity.sh
"${GEN}" team-a /keys
"${GEN}" team-b /keys

PEM_A="$(sed 's/^/      /' /keys/team-a.pem)"
PEM_B="$(sed 's/^/      /' /keys/team-b.pem)"

# The router is the shared transport plane: BOTH team keys are authorized
# on its management socket. Isolation is a per-daemon property, not a
# transport property.
cat > /etc/switchboard/router.yaml <<EOF
listen_addr: "0.0.0.0:9090"
management_socket: "/run/switchboard/router.sock"
tick_interval: 10ms
upstream_routers: []
authorized_operator_keys:
  - |
${PEM_A}
  - |
${PEM_B}
EOF

for team in a b; do
  pem_var="PEM_$(echo "${team}" | tr '[:lower:]' '[:upper:]')"
  for n in 1 2; do
    cat > "/etc/switchboard/access-${team}${n}.yaml" <<EOF
listen_addr: "127.0.0.1:9090"
management_socket: "/run/switchboard/${team}${n}.sock"
tick_interval: 10ms
authorized_operator_keys:
  - |
${!pem_var}
EOF
  done
done

echo "init: wrote router + 2x2 team node configs"
