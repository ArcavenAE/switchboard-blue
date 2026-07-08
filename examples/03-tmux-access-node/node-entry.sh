#!/usr/bin/env bash
# node-entry.sh — access-node container entrypoint.
#
# Starts a tmux server hosting a session named "work" running `top`
# (a real TUI program producing continuous screen updates), then starts
# the access daemon, which connects to that tmux server via control
# mode (`tmux -CC`) — or falls back to its PTY proxy if control mode is
# unavailable.
set -euo pipefail

tmux new-session -d -s work 'top'
echo "node-entry: tmux session 'work' started ($(tmux list-sessions -F '#{session_name}: #{session_windows} windows'))"

exec switchboard access --config /etc/switchboard/access.yaml
