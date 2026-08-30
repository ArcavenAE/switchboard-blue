# AC-012: both ticker goroutines join deterministically on Env.Close()
# Evidence: upstream and downstream ticker goroutines register on the existing Env.wg/closeCh; Close() blocks until both have exited (bounded-timeout join check)
# Test: TestLoopbackEnv_TickerGoroutines_JoinOnClose

Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-012-ticker-goroutines-join-on-close.gif
Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-012-ticker-goroutines-join-on-close.webm

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

Type "# AC-012: both ticker goroutines join deterministically on Env.Close()"
Enter
Sleep 300ms
Type "# upstream and downstream ticker goroutines register on the existing Env.wg/closeCh; Close() blocks until both have exited (bounded-timeout join check)"
Enter
Sleep 300ms

Type `go test -v -race -run "^TestLoopbackEnv_TickerGoroutines_JoinOnClose$" ./internal/testenv/`
Enter
Wait+Screen@60s /PASS/
Sleep 2000ms
