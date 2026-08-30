# AC-008: RoundTrip token API isolates concurrent/stale round trips
# Evidence: WaitForEcho consumes exactly one token's own completion channel; never reads Env.CollectFrames' accumulating buffer
# Test: TestLoopbackEnv_WaitForEcho_DoesNotConsumeOtherRoundTrips, TestLoopbackEnv_WaitForEcho_IgnoresStaleCollectFramesBuffer

Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-008-roundtrip-token-isolation.gif
Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-008-roundtrip-token-isolation.webm

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

Type "# AC-008: RoundTrip token API isolates concurrent/stale round trips"
Enter
Sleep 300ms
Type "# WaitForEcho consumes exactly one token's own completion channel; never reads Env.CollectFrames' accumulating buffer"
Enter
Sleep 300ms

Type `go test -v -race -run "^(TestLoopbackEnv_WaitForEcho_DoesNotConsumeOtherRoundTrips|TestLoopbackEnv_WaitForEcho_IgnoresStaleCollectFramesBuffer)$" ./internal/testenv/`
Enter
Wait+Screen@60s /PASS/
Sleep 2000ms
