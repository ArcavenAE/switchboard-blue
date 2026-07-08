#!/usr/bin/env bash
# node-entry.sh <team><n> <program...> — tmux session + access daemon for
# one team node.
set -euo pipefail

id="$1"
shift

tmux new-session -d -s "${id}" "$*"
echo "node-entry: tmux session '${id}' running: $*"

exec switchboard access --config "/etc/switchboard/access-${id}.yaml"
