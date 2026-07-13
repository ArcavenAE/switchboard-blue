# AC-013 — `router.reload`/`router.drain` registration: router-mode-exclusive,
# register-before-serve
#
# Demonstrates:
#   1. wireRouterControlHandlers is called from runRouter at the same phase as
#      wireMetricsHandlers, BEFORE serveMgmtServer starts the Serve goroutine
#      (register-before-serve invariant, F-P2L1-001)
#   2. runAccess, runConsole, runControl never call wireRouterControlHandlers —
#      router.reload and router.drain return E-RPC-010 on those modes
#   3. runRouter's sighupCh widens to bidirectional; a new trailing
#      drainRequestCh chan struct{} parameter reaches the select loop's third
#      arm and drives the same shutdown sequence + exit parity as SIGTERM
#
# BC anchors: Decision 4 (registration point); F-P2L1-001
#
# Test coverage:
#   TestWireRouterControlHandlers_RegisterBeforeServe                              (cmd/switchboard/router_control_wire_test.go)
#   TestWireRouterControlHandlers_RouterModeExclusive_OtherModesERPC010            (cmd/switchboard/router_control_wire_test.go)
#   TestRunRouter_DrainRequestChThirdSelectArm_ReachesShutdown_SameExitParityAsSIGTERM (cmd/switchboard/mgmt_wire_test.go)
#
# Usage: vhs docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-013-router-control-registration.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-013-router-control-registration.gif
Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-013-router-control-registration.webm

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

# ── AC-013: router-mode-exclusive registration, register-before-serve ─────────

Type `echo "=== AC-013: wireRouterControlHandlers — router-mode-exclusive, register-before-serve ==="`
Enter
Sleep 300ms

Type `go test -race -run 'TestWireRouterControlHandlers_RegisterBeforeServe|TestWireRouterControlHandlers_RouterModeExclusive_OtherModesERPC010|TestRunRouter_DrainRequestChThirdSelectArm_ReachesShutdown_SameExitParityAsSIGTERM' -count=1 -v ./cmd/switchboard/...`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-013 PASS ==="`
Enter
Sleep 2s
