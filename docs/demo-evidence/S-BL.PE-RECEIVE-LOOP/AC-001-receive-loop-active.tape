# AC-001 — Receive loop active after connect (E-FWD-001 liveness via runRouter)
#
# Demonstrates:
#   1. After PE connect, runRouter starts a receive goroutine on the upstream connection
#   2. The receive loop is active (liveness verified via integration test)
#   3. E-FWD-001 is emitted when incoming frames exhaust all paths (receive → OnFrameArrival)
#
# BC anchors: BC-2.09.001 (receive/forward loop), E-FWD-001 liveness
#
# Test coverage:
#   TestRunRouter_PE_ReceiveLoop_ActiveAfterConnect (cmd/switchboard/router_pe_receive_loop_test.go)
#
# Usage: vhs docs/demo-evidence/S-BL.PE-RECEIVE-LOOP/AC-001-receive-loop-active.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.PE-RECEIVE-LOOP/AC-001-receive-loop-active.gif
Output docs/demo-evidence/S-BL.PE-RECEIVE-LOOP/AC-001-receive-loop-active.webm

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

# ── AC-001: receive loop liveness after connect ───────────────────────────────

Type `echo "=== AC-001: receive loop active after PE connect ==="`
Enter
Sleep 300ms

Type `go test ./cmd/switchboard/ -run TestRunRouter_PE_ReceiveLoop_ActiveAfterConnect -count=1 -v`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-001 PASS ==="`
Enter
Sleep 2s
