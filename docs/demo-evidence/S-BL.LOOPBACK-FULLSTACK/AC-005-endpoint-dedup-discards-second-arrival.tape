# AC-005: endpoint checksum dedup discards the second-arriving duplicate
# Evidence: upstream: SendKeystroke called exactly once despite two deliverUpstream calls; downstream: Receive dedup gates deliverDownstream, not OnAck (BC-2.02.002)
# Test: TestLoopbackDriver_EndpointDedupDiscardsSecondArrival

Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-005-endpoint-dedup-discards-second-arrival.gif
Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-005-endpoint-dedup-discards-second-arrival.webm

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

Type "# AC-005: endpoint checksum dedup discards the second-arriving duplicate"
Enter
Sleep 300ms
Type "# upstream: SendKeystroke called exactly once despite two deliverUpstream calls; downstream: Receive dedup gates deliverDownstream, not OnAck (BC-2.02.002)"
Enter
Sleep 300ms

Type `go test -v -race -run "^TestLoopbackDriver_EndpointDedupDiscardsSecondArrival$" ./internal/testenv/`
Enter
Wait+Screen@60s /PASS/
Sleep 2000ms
