# AC-006 — register-before-serve, wireMetricsHandlers, VP-047 end-to-end integration
# Tests: TestVP047_SbctlPathsList_EndToEnd — handler→source→response code path
# Also covers: TestDaemonRouterStatus_HandlerRegistered, TestDaemonPathsList_HandlerRegistered
# VP-047; BC-2.06.003 v1.14 PC-1

Output docs/demo-evidence/S-W5.04/AC-006-register-and-vp047-e2e.gif
Output docs/demo-evidence/S-W5.04/AC-006-register-and-vp047-e2e.webm

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

Type "# AC-006: VP-047 e2e — handler→source→response; router_addr key present (empty string accepted)"
Enter
Sleep 300ms

Type "go test ./internal/metrics/... -v -run 'TestVP047_SbctlPathsList_EndToEnd|TestDaemonRouterStatus_HandlerRegistered' 2>&1"
Enter
Wait+Screen@30s /PASS/
Sleep 2000ms
