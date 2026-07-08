# AC-002 — Fail-closed reload on bad config (BC-2.09.003 EC-004)
#
# Demonstrates: when SIGHUP points at an invalid config file, the daemon emits
# the EC-004 log line and continues on the previous config without state change.
#
# Observable signals:
#   1. Startup emits: switchboard router: mode=E (no upstream_routers configured)
#   2. Config overwritten with invalid YAML (empty listen_addr → E-CFG-001).
#   3. After SIGHUP:
#      config reload failed: E-CFG-001: config error: listen_addr: required field
#      missing. ...; continuing with previous config
#   4. No second mode= line appears — state unchanged.
#   5. Router still responds: a second SIGHUP with the fixed (original) config
#      succeeds, proving the daemon continued executing.
#
# Test coverage: TestRunRouter_SIGHUPReload_BadConfig_FailClosed
# (cmd/switchboard/router_sighup_test.go — passes under go test -race)
#
# Usage: vhs AC-002-fail-closed-reload.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this
# .tape script is checked in.
#
# PATH NOTE: all paths are relative to repo root or derived via $PWD —
# never hardcoded to a specific worktree path (drbothen/vsdd-factory#418).

Output docs/demo-evidence/S-7.04-FU-SIGHUP-RELOAD/AC-002-fail-closed-reload.gif
Output docs/demo-evidence/S-7.04-FU-SIGHUP-RELOAD/AC-002-fail-closed-reload.webm

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

Type `go build -ldflags "-X main.version=demo" -o /tmp/sb-demo-ac002 ./cmd/switchboard`
Enter
Sleep 8s

Type `echo "build OK"`
Enter
Sleep 500ms

# ── Step 2: set up temp config ────────────────────────────────────────────────

Type `DEMO_DIR=$(mktemp -d) && echo "working in $DEMO_DIR"`
Enter
Sleep 500ms

Type `DEMO_DATAPORT=29803`
Enter
Sleep 200ms

Type `DEMO_SOCK="$DEMO_DIR/router.sock"`
Enter
Sleep 200ms

Type `DEMO_CFG="$DEMO_DIR/config.yaml"`
Enter
Sleep 200ms

# Write a valid initial config
Type `printf 'listen_addr: "127.0.0.1:%s"\nmanagement_socket: "%s"\ntick_interval: 10ms\n' "$DEMO_DATAPORT" "$DEMO_SOCK" > "$DEMO_CFG"`
Enter
Sleep 300ms

Type `echo "=== Initial config (valid) ==="`
Enter
Sleep 200ms

Type `cat "$DEMO_CFG"`
Enter
Sleep 2s

# ── Step 3: start the router ──────────────────────────────────────────────────

Type `echo "=== Step 2: start router ==="`
Enter
Sleep 300ms

Type `/tmp/sb-demo-ac002 router --config "$DEMO_CFG" > "$DEMO_DIR/router.log" 2>&1 & DEMO_PID=$! && echo "router PID=$DEMO_PID"`
Enter
Sleep 2s

Type `cat "$DEMO_DIR/router.log"`
Enter
Sleep 2s

# ── Step 4: overwrite config with invalid YAML, send SIGHUP ──────────────────

Type `echo "=== Step 3: overwrite config with invalid YAML (no listen_addr) ==="`
Enter
Sleep 300ms

# Invalid: missing required listen_addr → triggers E-CFG-001
Type `printf 'tick_interval: 10ms\n' > "$DEMO_CFG" && echo "invalid config written"`
Enter
Sleep 300ms

Type `cat "$DEMO_CFG"`
Enter
Sleep 1s

Type `kill -HUP "$DEMO_PID" && echo "SIGHUP sent"`
Enter
Sleep 1s

# ── Step 5: show EC-004 log line ──────────────────────────────────────────────

Type `echo "=== Log after SIGHUP with invalid config (expect reload failed + E-CFG-001) ==="`
Enter
Sleep 200ms

Type `cat "$DEMO_DIR/router.log"`
Enter
Sleep 3s

# Highlight the EC-004 line
Type `echo "--- AC-002 check: fail-closed line present ---"`
Enter
Sleep 200ms

Type `grep "config reload failed" "$DEMO_DIR/router.log"`
Enter
Sleep 2s

# Verify no second mode= line (state unchanged)
Type `echo "--- AC-002 check: mode= line count (expect 1, startup only) ---"`
Enter
Sleep 200ms

Type `grep -c "mode=" "$DEMO_DIR/router.log" && echo "lines (expect 1 — no state change)"`
Enter
Sleep 1s

# ── Step 6: prove daemon still responds via second SIGHUP with valid config ───

Type `echo "=== Step 4: restore valid config, send second SIGHUP (daemon still alive) ==="`
Enter
Sleep 300ms

Type `printf 'listen_addr: "127.0.0.1:%s"\nmanagement_socket: "%s"\ntick_interval: 10ms\nupstream_routers:\n  - addr: "127.0.0.1:29804"\n' "$DEMO_DATAPORT" "$DEMO_SOCK" > "$DEMO_CFG"`
Enter
Sleep 300ms

Type `kill -HUP "$DEMO_PID" && echo "second SIGHUP sent (valid PE config)"`
Enter
Sleep 1s

Type `cat "$DEMO_DIR/router.log"`
Enter
Sleep 3s

Type `echo "--- daemon recovered: mode=PE now present after second SIGHUP ---"`
Enter
Sleep 200ms

Type `grep "mode=PE" "$DEMO_DIR/router.log" && echo "SUCCESS"`
Enter
Sleep 2s

# ── Cleanup ───────────────────────────────────────────────────────────────────

Type `kill "$DEMO_PID" 2>/dev/null; wait "$DEMO_PID" 2>/dev/null; rm -rf "$DEMO_DIR" /tmp/sb-demo-ac002 && echo "=== clean ==="`
Enter
Sleep 2s
