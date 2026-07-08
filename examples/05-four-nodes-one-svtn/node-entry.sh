#!/usr/bin/env bash
# node-entry.sh <n> <program...> — start tmux running the given program in
# a session named after the node, then start that node's access daemon.
set -euo pipefail

n="$1"
shift

tmux new-session -d -s "node${n}" "$*"
echo "node-entry: tmux session 'node${n}' running: $*"

exec switchboard access --config "/etc/switchboard/access-node${n}.yaml"
