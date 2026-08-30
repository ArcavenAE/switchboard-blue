# AC-003: empty ticks fire on schedule but are never wire-dispatched
# Evidence: Tick() fires every interval regardless of data (BC-2.01.002); empty-tick frames never reach multipath.Send
# Test: TestLoopbackDriver_EmptyTicksNotDispatched

Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-003-empty-ticks-not-dispatched.gif
Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-003-empty-ticks-not-dispatched.webm

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

Type "# AC-003: empty ticks fire on schedule but are never wire-dispatched"
Enter
Sleep 300ms
Type "# Tick() fires every interval regardless of data (BC-2.01.002); empty-tick frames never reach multipath.Send"
Enter
Sleep 300ms

Type `go test -v -race -run "^TestLoopbackDriver_EmptyTicksNotDispatched$" ./internal/testenv/`
Enter
Wait+Screen@60s /PASS/
Sleep 2000ms
