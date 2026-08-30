# AC-016: downstream OnAck error surfaces loud, never a silent timeout
# Evidence: fault-injection via the onDownstreamTick() seam drives OnAck into ErrAckOutOfWindow; driver.failLoud (t.Errorf) fires exactly once
# Test: TestLoopbackDriver_OnAckError_SurfacesLoud

Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-016-onack-error-surfaces-loud.gif
Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-016-onack-error-surfaces-loud.webm

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

Type "# AC-016: downstream OnAck error surfaces loud, never a silent timeout"
Enter
Sleep 300ms
Type "# fault-injection via the onDownstreamTick() seam drives OnAck into ErrAckOutOfWindow; driver.failLoud (t.Errorf) fires exactly once"
Enter
Sleep 300ms

Type `go test -v -race -run "^TestLoopbackDriver_OnAckError_SurfacesLoud$" ./internal/testenv/`
Enter
Wait+Screen@60s /PASS/
Sleep 2000ms
