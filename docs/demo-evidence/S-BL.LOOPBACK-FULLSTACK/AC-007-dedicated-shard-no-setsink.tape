# AC-007: loopback driver uses its own dedicated shard, never mutates env.defaultShard
# Evidence: loopbackDriver constructs its own Publisher/SessionAuth/AccessNode triple with WithKeystrokeSink(loopbackSink); session.AccessNode gains no SetSink escape hatch
# Test: TestLoopbackDriver_DedicatedShard_NoDefaultShardMutation, TestSessionAccessNode_NoSetSinkMethod

Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-007-dedicated-shard-no-setsink.gif
Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-007-dedicated-shard-no-setsink.webm

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

Type "# AC-007: loopback driver uses its own dedicated shard, never mutates env.defaultShard"
Enter
Sleep 300ms
Type "# loopbackDriver constructs its own Publisher/SessionAuth/AccessNode triple with WithKeystrokeSink(loopbackSink); session.AccessNode gains no SetSink escape hatch"
Enter
Sleep 300ms

Type `go test -v -race -run "^(TestLoopbackDriver_DedicatedShard_NoDefaultShardMutation|TestSessionAccessNode_NoSetSinkMethod)$" ./internal/testenv/`
Enter
Wait+Screen@60s /PASS/
Sleep 2000ms
