Output AC-004-bc-annotation-cleanup.gif
Output AC-004-bc-annotation-cleanup.webm

Set Shell "bash"
Set FontSize 14
Set FontFamily "Menlo"
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set Padding 20

Hide
Type "cd /Users/skippy/work/switchboard-blue/.worktrees/S-BL.ROUTER-ADDR"
Enter
Type "clear"
Enter
Show

Type "# AC-004: BC-2.06.003 v1.15 — DRIFT-SW504 annotation removed"
Enter
Sleep 500ms

Type "grep -c 'DRIFT-SW504' .factory/specs/behavioral-contracts/ss-06/BC-2.06.003.md && echo 'DRIFT-SW504 count (expect 0)'"
Enter
Sleep 2s

Type "grep -n 'version.*1.15\\|v1.15\\|NewPathTrackerWithAddr' .factory/specs/behavioral-contracts/ss-06/BC-2.06.003.md | head -5"
Enter
Sleep 3s
