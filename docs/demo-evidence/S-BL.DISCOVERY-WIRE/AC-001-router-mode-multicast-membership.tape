# AC-001 — Router-mode-exclusive multicast group membership
#
# Demonstrates:
#   1. Only the router-mode daemon calls net.ListenMulticastUDP and joins the
#      SVTN-scoped multicast group on its LAN-facing interface(s).
#   2. runAccess, runConsole, and runControl never join any multicast group
#      and never receive advertisements directly from another node's socket.
#
# Scope note (Ruling 4, v1.10, 2026-07-15): this AC's test verifies
# wireDiscoveryListener at function level. Daemon-lifecycle wiring into
# runRouter is a separate, not-yet-built concern (Forward Obligation (e),
# S-BL.ADMISSION-SYNC-WIRE) — out of scope for this test and this tape.
#
# BC anchors: BC-2.03.001 Postcondition 1 (delivery-mechanism note), Invariant 1 (DI-004)
#
# Test coverage:
#   TestRunRouter_DiscoveryListener_JoinsGroup_RouterModeOnly (cmd/switchboard/discovery_wire_test.go)
#
# Usage: vhs docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-001-router-mode-multicast-membership.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-001-router-mode-multicast-membership.gif
Output docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-001-router-mode-multicast-membership.webm

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

# ── AC-001: router-mode-exclusive multicast group membership ──────────────────

Type `echo "=== AC-001: router-mode daemon joins the SVTN multicast group; access/console/control never do ==="`
Enter
Sleep 300ms

Type `go test -race -run TestRunRouter_DiscoveryListener_JoinsGroup_RouterModeOnly -count=1 -v ./cmd/switchboard/`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-001 PASS ==="`
Enter
Sleep 2s
