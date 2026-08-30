# AC-011: teardown diagnostic reports any undrained driver.pending entry
# Evidence: t.Cleanup-registered diagnostic fires (Logf, non-fatal) for an injected stale pending entry; stays silent when the map is fully drained
# Test: TestLoopbackEnv_Cleanup_DiagnosticOnPendingLeak, TestLoopbackEnv_Cleanup_SilentWhenDrained

Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-011-cleanup-diagnostic-on-pending-leak.gif
Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-011-cleanup-diagnostic-on-pending-leak.webm

Set FontFamily "Menlo"
Set FontSize 13
Set Width 1200
Set Height 700
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 60ms

Hide
Type "clear"
Enter
Sleep 200ms
Show

Type "# AC-011: teardown diagnostic reports any undrained driver.pending entry"
Enter
Sleep 300ms
Type "# t.Cleanup-registered diagnostic fires (Logf, non-fatal) for an injected stale pending entry; stays silent when the map is fully drained"
Enter
Sleep 300ms

Type `go test -v -race -run "^(TestLoopbackEnv_Cleanup_DiagnosticOnPendingLeak|TestLoopbackEnv_Cleanup_SilentWhenDrained)$" ./internal/testenv/`
Enter
Wait+Screen@60s /PASS/
Sleep 2000ms
