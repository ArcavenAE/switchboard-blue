# AC-003: confirm-gate semantics — all five paths (interface-definitions.md §125/§127/§129)
# BC-2.07.001 postcondition 3; ADR-004 destructive-op guard
# Tests: TestSbctlAdmin_SVTNDestroy_ConfirmGate (7-case table, all five paths)
#
# Path 1 (valid --confirm flag)       → static shape-check passes, RPC dispatched
# Path 1 (invalid --confirm flag)     → static shape-check fails, no RPC, "invalid --confirm"
# Path 2 (omitted, TTY, match)        → interactive prompt, match → RPC dispatched
# Path 2 (omitted, TTY, mismatch)     → interactive prompt, mismatch → no RPC
# Path 3 (omitted, non-TTY)           → hostile-scripting error ("non-interactive session")
# Path 4 (--yes alone)                → bypass with stderr WARNING, RPC dispatched
# Path 5 (--yes + --confirm)          → E-CFG-006 usage error (exit 2), no RPC

Output AC-003-confirm-gate.gif
Output AC-003-confirm-gate.webm

Require go

Set FontFamily "Menlo"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set Padding 20
Set PlaybackSpeed 1

Hide
Type "cd /Users/skippy/work/aae-orc/run/switchboard-blue/.worktrees/S-6.05"
Enter
Sleep 500ms
Show

Type "# AC-003: confirm-gate — 5-path semantics (§125/§127/§129 + ADR-004)"
Enter
Sleep 300ms

Type "# Runs all 7 subtests: valid/invalid flag, TTY match/mismatch, non-TTY, --yes, --yes+--confirm"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlAdmin_SVTNDestroy_ConfirmGate ./cmd/sbctl/..."
Enter
Sleep 12s

Type "# All 7 paths PASS: P1-valid, P1-invalid, P2-match, P2-mismatch, P3-non-TTY, P4-yes, P5-conflict"
Enter
Sleep 2s
