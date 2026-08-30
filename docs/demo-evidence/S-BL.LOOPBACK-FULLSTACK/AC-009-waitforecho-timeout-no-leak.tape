# AC-009: RoundTrip.done buffered send does not block or leak on timeout
# Evidence: downstream ticker completion path unconditionally drains driver.pending and sends into the buffered done channel even after WaitForEcho has timed out
# Test: TestLoopbackEnv_WaitForEcho_TimeoutThenLateArrival_NoLeak

Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-009-waitforecho-timeout-no-leak.gif
Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-009-waitforecho-timeout-no-leak.webm

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

Type "# AC-009: RoundTrip.done buffered send does not block or leak on timeout"
Enter
Sleep 300ms
Type "# downstream ticker completion path unconditionally drains driver.pending and sends into the buffered done channel even after WaitForEcho has timed out"
Enter
Sleep 300ms

Type `go test -v -race -run "^TestLoopbackEnv_WaitForEcho_TimeoutThenLateArrival_NoLeak$" ./internal/testenv/`
Enter
Wait+Screen@60s /PASS/
Sleep 2000ms
