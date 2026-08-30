# AC-014: full round trip completes on the single shared *arq.ARQ instance (regression guard)
# Evidence: guards against reintroducing the arqServer/arqClient two-instance split (AC-001 Addendum) - asserts non-empty, correctly-correlated delivery, not merely that WaitForEcho returned
# Test: TestLoopbackEnv_RoundTripCompletes_SingleSharedARQInstance

Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-014-roundtrip-completes-single-arq-instance.gif
Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-014-roundtrip-completes-single-arq-instance.webm

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

Type "# AC-014: full round trip completes on the single shared *arq.ARQ instance (regression guard)"
Enter
Sleep 300ms
Type "# guards against reintroducing the arqServer/arqClient two-instance split (AC-001 Addendum) - asserts non-empty, correctly-correlated delivery, not merely that WaitForEcho returned"
Enter
Sleep 300ms

Type `go test -v -race -run "^TestLoopbackEnv_RoundTripCompletes_SingleSharedARQInstance$" ./internal/testenv/`
Enter
Wait+Screen@60s /PASS/
Sleep 2000ms
