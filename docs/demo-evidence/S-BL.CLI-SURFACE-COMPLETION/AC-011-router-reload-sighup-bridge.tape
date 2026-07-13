# AC-011 — `router.reload` bridges into the shipped SIGHUP-reload path
#
# Demonstrates:
#   1. The router.reload handler synthesizes a signal onto the (now-bidirectional)
#      sighupCh, coalescing exactly like signal.Notify's own semantics
#   2. From that synthesis point forward, the RPC-triggered and SIGHUP-OS-signal
#      -triggered reload paths are code-path-identical
#   3. Defense-in-depth guard: router.reload's handler checks configPath == ""
#      before synthesizing onto sighupCh, returning E-CFG-004 "reload not
#      applicable: daemon started without --config" synchronously if that
#      (normally-unreachable) invariant is ever violated
#
# BC anchors: BC-2.09.001 v1.2 PC-1 (RPC-trigger note)
#
# Test coverage:
#   TestRouterReload_BridgesToSighupCh_CodePathIdentical (cmd/switchboard/router_control_wire_test.go)
#   TestRouterReload_NoConfigLoaded_ECFG004              (cmd/switchboard/router_control_wire_test.go)
#
# Usage: vhs docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-011-router-reload-sighup-bridge.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-011-router-reload-sighup-bridge.gif
Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-011-router-reload-sighup-bridge.webm

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

# ── AC-011: router.reload bridges into SIGHUP path; defense-in-depth guard ────

Type `echo "=== AC-011: router.reload bridges to sighupCh; E-CFG-004 defense-in-depth guard ==="`
Enter
Sleep 300ms

Type `go test -race -run 'TestRouterReload_BridgesToSighupCh_CodePathIdentical|TestRouterReload_NoConfigLoaded_ECFG004' -count=1 -v ./cmd/switchboard/...`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-011 PASS ==="`
Enter
Sleep 2s
