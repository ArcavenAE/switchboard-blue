# AC-003 — PathEntry.status derivation: active/degraded from Active+Degraded flags (Ruling-9)
# Tests: TestPathEntry_StatusFromDegraded (3 rows) + TestPathEntry_StatusEnumClosed
# BC-2.06.003 v1.14 PC-1; wave-6-tranche-a-scope-rulings.md Ruling-9

Output docs/demo-evidence/S-W5.04/AC-003-status-derivation.gif
Output docs/demo-evidence/S-W5.04/AC-003-status-derivation.webm

Set FontFamily "Menlo"
Set FontSize 13
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 60ms

Hide
Type "cd /Users/skippy/work/switchboard-blue/.worktrees/S-W5.04 && clear"
Enter
Sleep 300ms
Show

Type "# AC-003: status derivation — active iff Active=true AND Degraded=false; degraded otherwise"
Enter
Sleep 300ms

Type "go test ./internal/metrics/... -v -run 'TestPathEntry_StatusFromDegraded|TestPathEntry_StatusEnumClosed' 2>&1"
Enter
Wait+Screen@30s /PASS/
Sleep 2000ms
