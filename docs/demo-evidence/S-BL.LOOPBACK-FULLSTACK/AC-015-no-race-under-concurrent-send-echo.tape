# AC-015: loopback driver runs clean under go test -race
# Evidence: per-direction upstreamHCMu/downstreamHCMu mutexes (H2) serialize Enqueue/Tick on each HalfChannel under concurrent SendKeystroke/WaitForEcho pairs - zero DATA RACE
# Test: TestLoopbackDriver_NoRaceUnderConcurrentSendEcho

Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-015-no-race-under-concurrent-send-echo.gif
Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-015-no-race-under-concurrent-send-echo.webm

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

Type "# AC-015: loopback driver runs clean under go test -race"
Enter
Sleep 300ms
Type "# per-direction upstreamHCMu/downstreamHCMu mutexes (H2) serialize Enqueue/Tick on each HalfChannel under concurrent SendKeystroke/WaitForEcho pairs - zero DATA RACE"
Enter
Sleep 300ms

Type `go test -v -race -run "^TestLoopbackDriver_NoRaceUnderConcurrentSendEcho$" ./internal/testenv/`
Enter
Wait+Screen@60s /PASS/
Sleep 2000ms
