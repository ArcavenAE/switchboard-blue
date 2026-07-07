# AC-001 — SIGHUP capture and conditional mode re-emission
#
# Demonstrates: operator adds upstream_routers to config file, sends SIGHUP,
# and runRouter emits "mode=PE" without a daemon restart (BC-2.09.001 PC-1).
#
# Observable signals:
#   1. Startup emits: switchboard router: mode=E (no upstream_routers configured)
#   2. After SIGHUP (config rewritten with upstream_routers):
#      switchboard router: mode=PE upstream_routers=[127.0.0.1:<port>]
#   3. Router PID unchanged — no process restart (daemon keeps running).
#
# Test coverage: TestRunRouter_SIGHUPReload_EtoPE
# (cmd/switchboard/router_sighup_test.go — passes under go test -race)
#
# Usage: vhs AC-001-sighup-e-to-pe.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this
# .tape script is checked in.
#
# PATH NOTE: all paths are relative to repo root or derived via $PWD —
# never hardcoded to a specific worktree path (drbothen/vsdd-factory#418).

Output docs/demo-evidence/S-7.04-FU-SIGHUP-RELOAD/AC-001-sighup-e-to-pe.gif
Output docs/demo-evidence/S-7.04-FU-SIGHUP-RELOAD/AC-001-sighup-e-to-pe.webm

Require go

Set Shell "zsh"
Set FontFamily "Menlo"
Set FontSize 14
Set Width 1200
Set Height 720
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful
Set BorderRadius 8
Set Padding 20
Set Framerate 30
Set TypingSpeed 30ms

Hide
Type "clear"
Enter
Sleep 1s
Show

# ── Step 1: build the binary ──────────────────────────────────────────────────

Type `echo "=== Step 1: build switchboard ==="`
Enter
Sleep 500ms

Type `go build -ldflags "-X main.version=demo" -o /tmp/sb-demo-ac001 ./cmd/switchboard`
Enter
Sleep 8s

Type `echo "build OK"`
Enter
Sleep 500ms

# ── Step 2: set up temp config files ─────────────────────────────────────────

Type `DEMO_DIR=$(mktemp -d) && echo "working in $DEMO_DIR"`
Enter
Sleep 500ms

Type `DEMO_DATAPORT=29801`
Enter
Sleep 200ms

Type `DEMO_SOCK="$DEMO_DIR/router.sock"`
Enter
Sleep 200ms

Type `DEMO_CFG="$DEMO_DIR/config.yaml"`
Enter
Sleep 200ms

# Write the initial E-mode config (no upstream_routers)
Type `printf 'listen_addr: "127.0.0.1:%s"\nmanagement_socket: "%s"\ntick_interval: 10ms\n' "$DEMO_DATAPORT" "$DEMO_SOCK" > "$DEMO_CFG"`
Enter
Sleep 300ms

Type `echo "=== Initial config (E mode — no upstream_routers) ==="`
Enter
Sleep 200ms

Type `cat "$DEMO_CFG"`
Enter
Sleep 2s

# ── Step 3: start the router ──────────────────────────────────────────────────

Type `echo "=== Step 2: start router in background ==="`
Enter
Sleep 300ms

Type `/tmp/sb-demo-ac001 router --config "$DEMO_CFG" > "$DEMO_DIR/router.log" 2>&1 & DEMO_PID=$! && echo "router PID=$DEMO_PID"`
Enter
Sleep 2s

Type `echo "=== Startup log (expect mode=E) ==="`
Enter
Sleep 200ms

Type `cat "$DEMO_DIR/router.log"`
Enter
Sleep 3s

# ── Step 4: rewrite config with upstream_routers, send SIGHUP ────────────────

Type `echo "=== Step 3: add upstream_routers to config, then SIGHUP ==="`
Enter
Sleep 300ms

# Overwrite config with PE-capable version (adds upstream_routers entry)
Type `printf 'listen_addr: "127.0.0.1:%s"\nmanagement_socket: "%s"\ntick_interval: 10ms\nupstream_routers:\n  - addr: "127.0.0.1:29802"\n' "$DEMO_DATAPORT" "$DEMO_SOCK" > "$DEMO_CFG"`
Enter
Sleep 300ms

Type `echo "=== Rewritten config (PE mode — upstream_routers present) ==="`
Enter
Sleep 200ms

Type `cat "$DEMO_CFG"`
Enter
Sleep 2s

Type `kill -HUP "$DEMO_PID" && echo "SIGHUP sent to PID=$DEMO_PID"`
Enter
Sleep 1s

# ── Step 5: show mode=PE emission ────────────────────────────────────────────

Type `echo "=== Log after SIGHUP (expect mode=PE) ==="`
Enter
Sleep 200ms

Type `cat "$DEMO_DIR/router.log"`
Enter
Sleep 3s

# The key observable: mode=PE line appeared without a new PID
Type `echo "--- AC-001 check: mode=PE line present ---"`
Enter
Sleep 200ms

Type `grep "mode=PE" "$DEMO_DIR/router.log"`
Enter
Sleep 2s

Type `echo "--- router PID unchanged: $DEMO_PID still running ---"`
Enter
Sleep 200ms

Type `kill -0 "$DEMO_PID" 2>&1 && echo "PID $DEMO_PID still alive (no restart)" || echo "FAIL: PID exited"`
Enter
Sleep 1s

# ── Cleanup ───────────────────────────────────────────────────────────────────

Type `kill "$DEMO_PID" 2>/dev/null; wait "$DEMO_PID" 2>/dev/null; rm -rf "$DEMO_DIR" /tmp/sb-demo-ac001 && echo "=== clean ==="`
Enter
Sleep 2s
