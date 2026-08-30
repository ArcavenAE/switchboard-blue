# AC-013: merged bench file updated to token-based RoundTrip API + VP-042 keystroke-to-echo p99 evidence
# Evidence: internal/bench/keystroke_echo_testenv_bench_test.go (develop, PR #121 @ 4c276d9, //go:build integration)
#   drives lb.SendKeystroke/lb.WaitForEcho (token-based) through the full tick-driven loopback stack.
#   AC-013 has no new unit test - verification is (1) the compile-gate that actually reaches the
#   integration-tagged test binary, then (2) the VP-042 benchmark run itself (harness-delivered, not locked).
# Test: no new test (bench-file modification); BenchmarkKeystrokeToEcho_P99

Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-013-vp042-bench.gif
Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-013-vp042-bench.webm

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

Type "# AC-013: compile-gate for the integration-tagged bench binary (go test, not go build)"
Enter
Sleep 300ms
Type `go test -tags integration -run '^$' -count=1 ./internal/bench/`
Enter
Wait+Screen@30s /ok/
Sleep 1500ms

Type "# VP-042: keystroke-to-echo p99 <= 100ms - harness-delivered evidence run (not a verification_lock flip)"
Enter
Sleep 300ms
Type `go test -tags integration -run '^$' -bench 'KeystrokeToEcho' -benchtime 1x -count=1 ./internal/bench/`
Enter
Wait+Screen@60s /PASS/
Sleep 2500ms
