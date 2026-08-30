# AC-002: tick-interval validation + independent per-direction schedule
# Evidence: NewLoopback validates TickIntervalUpstream/Downstream; both tickers fire Tick() on schedule, independent of Enqueue timing (BC-2.01.001, BC-2.01.003 harness-scope clock-independence)
# Test: TestNewLoopback_RejectsOutOfBoundsTickInterval, TestLoopbackDriver_TicksFireOnSchedule

Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-002-tick-interval-and-schedule.gif
Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-002-tick-interval-and-schedule.webm

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

Type "# AC-002: tick-interval validation + independent per-direction schedule"
Enter
Sleep 300ms
Type "# NewLoopback validates TickIntervalUpstream/Downstream; both tickers fire Tick() on schedule, independent of Enqueue timing (BC-2.01.001, BC-2.01.003 harness-scope clock-independence)"
Enter
Sleep 300ms

Type `go test -v -race -run "^(TestNewLoopback_RejectsOutOfBoundsTickInterval|TestLoopbackDriver_TicksFireOnSchedule)$" ./internal/testenv/`
Enter
Wait+Screen@60s /PASS/
Sleep 2000ms
