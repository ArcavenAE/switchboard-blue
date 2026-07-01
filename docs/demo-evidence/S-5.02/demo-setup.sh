#!/usr/bin/env bash
# demo-setup.sh — build demo binaries for S-5.02 VHS recordings.
# Run once before recording tapes.
set -euo pipefail

WORKTREE="$(cd "$(dirname "$0")/../../../" && pwd)"
EVIDENCE_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "Building sbctl..."
cd "$WORKTREE"
go build -o /tmp/sbctl-s502 ./cmd/sbctl/

echo "Building stub daemon..."
cd "$EVIDENCE_DIR"
go build -o /tmp/stub-daemon-s502 ./stub_daemon.go

echo "Copying test key..."
cp "$WORKTREE/cmd/sbctl/testdata/test_ed25519_key" /tmp/demo-ed25519-key

echo "Done. Binaries: /tmp/sbctl-s502  /tmp/stub-daemon-s502"
echo "Key: /tmp/demo-ed25519-key"
