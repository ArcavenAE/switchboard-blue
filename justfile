# switchboard justfile
set dotenv-load
set dotenv-filename := ".env.local"

VERSION := env("VERSION", "dev")

# ─── Core recipes ─────────────────────────────────────────────

# Build the switchboard binary
build:
    go build -ldflags "-X main.version={{VERSION}}" -o bin/switchboard ./cmd/switchboard

# Build the sbctl binary
build-sbctl:
    go build -ldflags "-X main.version={{VERSION}}" -o bin/sbctl ./cmd/sbctl

# Build both binaries (switchboard + sbctl)
build-both: build build-sbctl

# Build and run
run: build
    ./bin/switchboard

# Remove build artifacts
clean:
    rm -rf bin/

# Format with gofumpt
fmt:
    gofumpt -w .

# Run golangci-lint
lint:
    golangci-lint run ./...

# Run all tests
test:
    go test ./... -v

# Run tests with race detector
test-race:
    go test -race ./... -v

# Run tests in Docker
test-docker:
    @command -v docker >/dev/null 2>&1 || { echo "Error: Docker is required but not found."; exit 1; }
    @docker info >/dev/null 2>&1 || { echo "Error: Docker daemon is not running."; exit 1; }
    @mkdir -p test-results
    docker compose -f docker-compose.test.yml run --rm test

# ─── Smoke ────────────────────────────────────────────────────

# Plan B stratified smoke harness (task #176). Three tiers:
#
#   smoke-quick     Tier 1 (INV-1..INV-10). <10s. Runs on every PR.
#   smoke           Tier 2 daemon lifecycle. ~30s. Nightly + on-demand.
#   smoke-tutorial  Tier 3 tutorial walk. ~10s. Nightly + on-demand.
#                   (S-BL.ROUTER-RUNTIME landed — the T3-2-router
#                    expected-fail is gone; tier 3 is a clean pass gate now.)
#
# See test/smoke/invariants.sh, tier2-daemon.sh, tier3-tutorial.sh and
# docs/architecture.md §Smoke Invariants for the assertion catalog.

# Sentinel invariants (Tier 1): build both binaries with a stamped VERSION
# token, then run the operator-boundary smoke gate. Runs in <10 seconds.
#
# VERSION token is deliberately time-stamped so INV-8 (ldflags wiring)
# asserts the injected value flows through both banners. Local devs
# running `test/smoke/invariants.sh` bareback (without VERSION set) get
# INV-8 SKIP; this recipe forces the assertion so CI and pre-push checks
# behave identically.
#
# Builds are inlined (not a `build-both` dependency) so the shell-computed
# STAMP flows into both ldflags invocations under a single env.
smoke-quick:
    #!/usr/bin/env bash
    set -euo pipefail
    STAMP="smoke-$(date -u +%Y%m%dT%H%M%SZ)"
    go build -ldflags "-X main.version=${STAMP}" -o bin/switchboard ./cmd/switchboard
    go build -ldflags "-X main.version=${STAMP}" -o bin/sbctl ./cmd/sbctl
    VERSION="${STAMP}" ./test/smoke/invariants.sh

# Daemon lifecycle (Tier 2): start/SIGTERM/restart/error-taxonomy against a
# live daemon. Slower than Tier 1 (~30s) — do not gate PRs; nightly CI job.
#
# Uses `switchboard control` mode (Unix socket) because access mode
# requires a PTY unavailable on hosted CI runners. Both modes share the
# same signal-handling and config-load path, so this still exercises the
# daemon skeleton.
smoke: smoke-quick
    ./test/smoke/tier2-daemon.sh

# Tutorial smoke (Tier 3): extract fenced bash blocks from
# docs/getting-started.md and assert exit codes + substring presence.
# S-BL.ROUTER-RUNTIME landed — T3-2-router now passes; exit 0 is the
# steady-state result. Safe to wire into nightly CI.
smoke-tutorial: smoke-quick
    ./test/smoke/tier3-tutorial.sh

# Spec assertions (Plan D, task #178): data-driven projection of spec ACs
# into behavioral assertions. Catalog: test/smoke/spec-assertions.json;
# generic runner: test/smoke/spec-runner.sh. Adding spec coverage = adding
# a JSON entry, no new bash. Expected-fail entries reference an open issue
# and flip to required-pass (XPASS fails the run) when the defect is fixed.
# Runs on every PR alongside Tier 1 (<5s).
smoke-spec:
    #!/usr/bin/env bash
    set -euo pipefail
    go build -o bin/switchboard ./cmd/switchboard
    go build -o bin/sbctl ./cmd/sbctl
    ./test/smoke/spec-runner.sh

# ─── Cross-compile ────────────────────────────────────────────

# Build for all release targets
build-all:
    GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version={{VERSION}}" -o bin/switchboard-darwin-arm64 ./cmd/switchboard
    GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version={{VERSION}}" -o bin/switchboard-darwin-amd64 ./cmd/switchboard
    GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version={{VERSION}}" -o bin/switchboard-linux-amd64 ./cmd/switchboard

# ─── Signing & packaging ─────────────────────────────────────

# Codesign the binary
sign: (_require-var "APPLE_SIGNING_IDENTITY" env("APPLE_SIGNING_IDENTITY", ""))
    codesign --force --options runtime --sign "${APPLE_SIGNING_IDENTITY}" --timestamp bin/switchboard

# Verify the codesign signature
verify:
    codesign --verify --verbose=2 bin/switchboard

# ─── Internal helpers ─────────────────────────────────────────

# Check that an env var is set
[private]
_require-var name value:
    #!/usr/bin/env bash
    if [ -z "{{value}}" ]; then
        echo "ERROR: {{name}} is not set."
        echo ""
        echo "To set up local signing:"
        echo "  1. Copy .env.local.example to .env.local"
        echo "  2. Fill in your values"
        echo "  3. Run 'just sign-check' to find your signing identities"
        echo ""
        echo ".env.local is gitignored (*.env.* pattern)."
        exit 1
    fi
