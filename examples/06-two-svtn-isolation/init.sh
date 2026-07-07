#!/usr/bin/env bash
# init.sh — one-shot: three identities (one network operator, two SVTN
# operators) + configs for one shared router and two per-team node sets.
#
# Role model (spec ubiquitous language, carrier-grade content
# separation: "the router provides infrastructure, the customer holds
# the data keys"):
#   netop   — the network operator: provides + administers the router.
#   team-a  — an SVTN operator (tenant): administers team a's nodes.
#   team-b  — an SVTN operator (tenant): administers team b's nodes.
set -euo pipefail

GEN=/usr/local/lib/switchboard-examples/gen-identity.sh
"${GEN}" netop /keys
"${GEN}" team-a /keys
"${GEN}" team-b /keys

PEM_NETOP="$(sed 's/^/      /' /keys/netop.pem)"
PEM_A="$(sed 's/^/      /' /keys/team-a.pem)"
PEM_B="$(sed 's/^/      /' /keys/team-b.pem)"

# The router is the network operator's infrastructure: ONLY the netop
# key is authorized on its management socket. Tenants share the
# router's data plane (transport), not its management plane.
cat > /etc/switchboard/router.yaml <<EOF
listen_addr: "0.0.0.0:9090"
management_socket: "/run/switchboard/router.sock"
tick_interval: 10ms
upstream_routers: []
authorized_operator_keys:
  - |
${PEM_NETOP}
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

echo "init: wrote router (netop-managed) + 2x2 team node configs"
