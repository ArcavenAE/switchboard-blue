# AC-001 — paths.list handler: registered, reads PathSnapshots, returns PathsListResponse
# Tests: TestDaemonPathsList_HandlerRegistered (populated) + TestDaemonPathsList_EmptySource (EC-001)
# BC-2.06.003 v1.14 PC-1

Output docs/demo-evidence/S-W5.04/AC-001-paths-list-handler.gif
Output docs/demo-evidence/S-W5.04/AC-001-paths-list-handler.webm

Set FontFamily "Menlo"
Set FontSize 13
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 60ms

Hide
Type "clear"
Enter
Sleep 300ms
Show

Type "# AC-001: paths.list handler — populated source and empty source (EC-001)"
Enter
Sleep 300ms

Type "go test ./internal/metrics/... -v -run 'TestDaemonPathsList_HandlerRegistered|TestDaemonPathsList_EmptySource' 2>&1"
Enter
Wait+Screen@30s /PASS/
Sleep 2000ms
