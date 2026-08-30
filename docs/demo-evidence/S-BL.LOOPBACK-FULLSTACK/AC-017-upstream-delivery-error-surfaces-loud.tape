# AC-017: upstream SendKeystroke delivery error surfaces loud, in-place
# Evidence: deliverUpstream checks accessNode.SendKeystroke's error in-place (not via upstreamMP.Send's masked return) and calls driver.failLoud exactly once on failure
# Test: TestLoopbackDriver_UpstreamDeliveryError_SurfacesLoud

Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-017-upstream-delivery-error-surfaces-loud.gif
Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-017-upstream-delivery-error-surfaces-loud.webm

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

Type "# AC-017: upstream SendKeystroke delivery error surfaces loud, in-place"
Enter
Sleep 300ms
Type "# deliverUpstream checks accessNode.SendKeystroke's error in-place (not via upstreamMP.Send's masked return) and calls driver.failLoud exactly once on failure"
Enter
Sleep 300ms

Type `go test -v -race -run "^TestLoopbackDriver_UpstreamDeliveryError_SurfacesLoud$" ./internal/testenv/`
Enter
Wait+Screen@60s /PASS/
Sleep 2000ms
