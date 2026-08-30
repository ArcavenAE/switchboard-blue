# AC-006: downstream ARQ EnqueueSend + OnAck wired once per data tick
# Evidence: EnqueueSend/OnAck run on the SAME shared *arq.ARQ instance, once per downstream data tick, in the tick body after Send returns (BC-2.02.005, AC-001-discharged call contract)
# Test: TestLoopbackDriver_DownstreamARQWiring

Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-006-downstream-arq-wiring.gif
Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-006-downstream-arq-wiring.webm

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

Type "# AC-006: downstream ARQ EnqueueSend + OnAck wired once per data tick"
Enter
Sleep 300ms
Type "# EnqueueSend/OnAck run on the SAME shared *arq.ARQ instance, once per downstream data tick, in the tick body after Send returns (BC-2.02.005, AC-001-discharged call contract)"
Enter
Sleep 300ms

Type `go test -v -race -run "^TestLoopbackDriver_DownstreamARQWiring$" ./internal/testenv/`
Enter
Wait+Screen@60s /PASS/
Sleep 2000ms
