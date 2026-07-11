# AC-004 — E-FWD-001 exhaustion discharge under load + no-duplicate-suppression
#
# Demonstrates:
#   1. E-FWD-001 is emitted when a received PE frame exhausts all routing paths
#   2. peWriteFixture injection enables controlled frame injection into receive loop
#   3. No false duplicate suppression: distinct outer headers each produce E-FWD-001
#
# BC anchors: BC-2.02.008 (E-FWD-001 split-horizon drop), E-FWD-001 full discharge
#
# Test coverage:
#   TestRunRouter_PE_EFWD001ExhaustionUnderLoad                    (cmd/switchboard/router_pe_receive_loop_test.go)
#   TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader  (cmd/switchboard/router_pe_receive_loop_test.go)
#
# Usage: vhs docs/demo-evidence/S-BL.PE-RECEIVE-LOOP/AC-004-efwd001-exhaustion.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.PE-RECEIVE-LOOP/AC-004-efwd001-exhaustion.gif
Output docs/demo-evidence/S-BL.PE-RECEIVE-LOOP/AC-004-efwd001-exhaustion.webm

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

# ── AC-004: E-FWD-001 exhaustion + no-duplicate-suppression ──────────────────

Type `echo "=== AC-004: E-FWD-001 exhaustion under load + no duplicate suppression ==="`
Enter
Sleep 300ms

Type `go test ./cmd/switchboard/ -run 'TestRunRouter_PE_EFWD001ExhaustionUnderLoad|TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader' -count=1 -v`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-004 PASS ==="`
Enter
Sleep 2s
