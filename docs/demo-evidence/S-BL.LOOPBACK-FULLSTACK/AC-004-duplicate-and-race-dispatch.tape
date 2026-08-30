# AC-004: single enqueued payload dispatched over both synthetic paths
# Evidence: multipath.Send fires deliverUpstream/deliverDownstream exactly twice per ticked data frame (BC-2.02.001; incidental BC-2.02.003 paths.Rank() harness-scope aspect)
# Test: TestLoopbackDriver_DuplicateAndRaceDispatch

Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-004-duplicate-and-race-dispatch.gif
Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-004-duplicate-and-race-dispatch.webm

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

Type "# AC-004: single enqueued payload dispatched over both synthetic paths"
Enter
Sleep 300ms
Type "# multipath.Send fires deliverUpstream/deliverDownstream exactly twice per ticked data frame (BC-2.02.001; incidental BC-2.02.003 paths.Rank() harness-scope aspect)"
Enter
Sleep 300ms

Type `go test -v -race -run "^TestLoopbackDriver_DuplicateAndRaceDispatch$" ./internal/testenv/`
Enter
Wait+Screen@60s /PASS/
Sleep 2000ms
