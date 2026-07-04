# AC-004 — router.metrics handler: positive path + E-RPC-011 not-found error
# Tests: TestDaemonRouterMetrics_HandlerRegistered + TestDaemonRouterMetrics_SVTNNotFound
# BC-2.06.003 v1.14 PC-2; error code E-RPC-011

Output docs/demo-evidence/S-W5.04/AC-004-router-metrics-handler.gif
Output docs/demo-evidence/S-W5.04/AC-004-router-metrics-handler.webm

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

Type "# AC-004: router.metrics handler — positive path and E-RPC-011 not-found error"
Enter
Sleep 300ms

Type "go test ./internal/metrics/... -v -run 'TestDaemonRouterMetrics' 2>&1"
Enter
Wait+Screen@30s /PASS/
Sleep 2000ms
