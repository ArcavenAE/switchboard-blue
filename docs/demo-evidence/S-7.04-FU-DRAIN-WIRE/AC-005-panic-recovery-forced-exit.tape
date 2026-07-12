# AC-005 — EC-003 forced-exit via TestRunRouter_ForcedExitPastDrainTimeout (pre-existing,
# no change expected) + drain.Signal panic recovery
#
# Demonstrates:
#   1. drain.Signal's fan-out goroutine gains a defer recover() wrapper: a panicking
#      observer does not crash the coordinator and Wait still returns within the
#      window. Verified via a subprocess-isolated re-exec of the test binary (the
#      RED-GATE mechanism: a pre-fix panic would otherwise take the whole test
#      binary down with it).
#   2. EC-003 (unresponsive registered observer forces exit past cfg.DrainTimeout)
#      is evidenced by the pre-existing TestRunRouter_ForcedExitPastDrainTimeout,
#      unchanged by this story per Q-AC005/F-DW-SP3-007.
#
# BC anchors: BC-2.09.002 EC-003; placement note Q5, Q-AC005
#
# Test coverage:
#   TestDrain_ObserverPanicRecovery          (internal/drain/drain_test.go, subprocess-isolated)
#   TestRunRouter_ForcedExitPastDrainTimeout  (cmd/switchboard/router_drain_test.go, pre-existing)
#
# Usage: vhs docs/demo-evidence/S-7.04-FU-DRAIN-WIRE/AC-005-panic-recovery-forced-exit.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-7.04-FU-DRAIN-WIRE/AC-005-panic-recovery-forced-exit.gif
Output docs/demo-evidence/S-7.04-FU-DRAIN-WIRE/AC-005-panic-recovery-forced-exit.webm

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

# ── AC-005: observer panic recovery ───────────────────────────────────────────

Type `echo "=== AC-005: drain.Signal recovers from a panicking observer ==="`
Enter
Sleep 300ms

Type `go test -race -run TestDrain_ObserverPanicRecovery -count=1 -v ./internal/drain/`
Enter
Wait+Line /PASS/
Sleep 2s

# ── AC-005: EC-003 forced-exit past drain timeout ─────────────────────────────

Type `echo "=== AC-005: EC-003 — unresponsive observer forces exit past DrainTimeout ==="`
Enter
Sleep 300ms

Type `go test -race -run TestRunRouter_ForcedExitPastDrainTimeout -count=1 -v ./cmd/switchboard/`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-005 PASS ==="`
Enter
Sleep 2s
