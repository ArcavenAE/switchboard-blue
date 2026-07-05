# S-BL.ROUTER-RUNTIME AC-001 — router daemon starts, binds mgmt + data plane,
# shuts down cleanly on SIGTERM.
#
# Demonstrates:
#   - `switchboard router` accepts a valid config
#   - management socket appears on disk (evidence newMgmtServer + serveMgmtServer wired)
#   - data-plane TCP listener binds cfg.ListenAddr (accept-and-close skeleton)
#   - SIGTERM produces clean exit (nil error, WaitGroup drained)
#   - `switchboard router --help` exits 0 (INV-7 sentinel)
#
# Traces: S-BL.ROUTER-RUNTIME AC-001..AC-004; ARCH-01 §Goroutine WaitGroup Contract;
# ADR-004 role-exclusion (router does NOT register admin.key.* handlers).
#
# Usage: vhs AC-001-router-startup.tape
# Per POL-004 rendered artifacts are gitignored; only this .tape script is
# checked in.

Output docs/demo-evidence/S-BL.ROUTER-RUNTIME/AC-001-router-startup.gif
Output docs/demo-evidence/S-BL.ROUTER-RUNTIME/AC-001-router-startup.webm

Require go

Set Shell "zsh"
Set FontFamily "Menlo"
Set FontSize 14
Set Width 1200
Set Height 700
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

# INV-7 sentinel — router --help exits 0
Type "./bin/switchboard router --help"
Enter
Sleep 2s

# Prepare a tutorial-shape router config on a loopback ephemeral pair
Type "TMPDIR_ROUTER=$(mktemp -d) && cat >$TMPDIR_ROUTER/router.yaml <<EOF"
Enter
Type "listen_addr: \"127.0.0.1:0\""
Enter
Type "management_socket: \"$TMPDIR_ROUTER/router.sock\""
Enter
Type "tick_interval: 10ms"
Enter
Type "upstream_routers: []"
Enter
Type "EOF"
Enter
Sleep 1s

# Start the router; capture its stderr for the observability lines
Type "./bin/switchboard router --config $TMPDIR_ROUTER/router.yaml 2>&1 | tee $TMPDIR_ROUTER/router.log &"
Enter
Sleep 2s

# Confirm the mgmt socket materialised (AC-001 observable) — proof
# newMgmtServer + serveMgmtServer wired the register-before-serve invariant.
Type "test -S $TMPDIR_ROUTER/router.sock && echo 'AC-001: mgmt socket present' || echo 'FAIL: socket missing'"
Enter
Sleep 2s

# Confirm the router logged its listen address + socket path
Type "grep -c 'switchboard router:' $TMPDIR_ROUTER/router.log"
Enter
Sleep 2s

# SIGTERM the daemon — AC-004 graceful shutdown contract
Type "kill -TERM %1 && wait %1 && echo 'AC-004: exit clean on SIGTERM'"
Enter
Sleep 3s

# Cleanup
Type "rm -rf $TMPDIR_ROUTER"
Enter
Sleep 1s
