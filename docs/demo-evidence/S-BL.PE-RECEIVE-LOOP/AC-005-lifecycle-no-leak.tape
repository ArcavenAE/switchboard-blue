# AC-005 — Receive goroutine lifecycle: exit on close/error, flap-cycle no-leak
#
# Demonstrates:
#   1. Receive goroutine exits cleanly when connection is closed
#   2. Receive goroutine exits on read error (non-close error)
#   3. Receive goroutine exits on version mismatch (protocol error)
#   4. Flap cycle (connect → disconnect → reconnect) produces no goroutine leak
#      (per-reconnect join before new goroutine spawn)
#
# BC anchors: BC-2.09.001 (goroutine lifecycle), goroutine leak prevention
#
# Test coverage:
#   TestConnector_ReceiveLoop_ExitsOnConnClose        (internal/upstreamdial/connector_test.go)
#   TestConnector_ReceiveLoop_ExitsOnReadError         (internal/upstreamdial/connector_test.go)
#   TestConnector_ReceiveLoop_ExitsOnVersionMismatch   (internal/upstreamdial/connector_test.go)
#   TestConnector_ReceiveLoop_FlapCycleJoin_NoLeak     (internal/upstreamdial/connector_test.go)
#
# Usage: vhs docs/demo-evidence/S-BL.PE-RECEIVE-LOOP/AC-005-lifecycle-no-leak.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.PE-RECEIVE-LOOP/AC-005-lifecycle-no-leak.gif
Output docs/demo-evidence/S-BL.PE-RECEIVE-LOOP/AC-005-lifecycle-no-leak.webm

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

# ── AC-005: receive goroutine lifecycle + flap-cycle no-leak ─────────────────

Type `echo "=== AC-005: receive goroutine exits on close/error, flap-cycle no-leak ==="`
Enter
Sleep 300ms

Type `go test ./internal/upstreamdial/ -run 'TestConnector_ReceiveLoop_ExitsOnConnClose|TestConnector_ReceiveLoop_ExitsOnReadError|TestConnector_ReceiveLoop_ExitsOnVersionMismatch|TestConnector_ReceiveLoop_FlapCycleJoin_NoLeak' -count=1 -v`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-005 PASS ==="`
Enter
Sleep 2s
