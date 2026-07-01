# AC-002 — rtt_p99_ms union serialization: "pending" when SampleCount<10, float64 when >=10
# Tests: TestPathEntry_RTTValueSerialization (4 rows: count=0,9,10,100) + TestRTTValue_RoundTrip
# BC-2.06.003 v1.14 PC-1, EC-003

Output docs/demo-evidence/S-W5.04/AC-002-rtt-p99-serialization.gif
Output docs/demo-evidence/S-W5.04/AC-002-rtt-p99-serialization.webm

Set FontFamily "Menlo"
Set FontSize 13
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 60ms

Hide
Type "cd /Users/skippy/work/switchboard-blue/.worktrees/S-W5.04 && clear"
Enter
Sleep 300ms
Show

Type "# AC-002: rtt_p99_ms — pending string (<10 samples) vs float64 (>=10 samples)"
Enter
Sleep 300ms

Type "go test ./internal/metrics/... -v -run 'TestPathEntry_RTTValueSerialization|TestRTTValue_RoundTrip' 2>&1"
Enter
Wait+Screen@30s /PASS/
Sleep 2000ms
