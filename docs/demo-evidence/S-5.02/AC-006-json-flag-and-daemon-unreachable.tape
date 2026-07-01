# AC-006 — --json respected on both commands; daemon unreachable -> E-NET-001 exit code 1
# Tests: TestSbctlMetrics_JSONEnvelope, TestSbctlMetrics_DaemonUnreachable

Output docs/demo-evidence/S-5.02/AC-006-json-flag-and-daemon-unreachable.gif
Output docs/demo-evidence/S-5.02/AC-006-json-flag-and-daemon-unreachable.webm

Set FontFamily "Menlo"
Set FontSize 13
Set Width 1200
Set Height 700
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 60ms

Hide
Type "/tmp/stub-daemon-s502 paths-full /tmp/demo-ac006.sock &"
Enter
Sleep 600ms
Show

Type "# AC-006: --json envelope + E-NET-001 on unreachable daemon"
Enter
Sleep 300ms

Type "# Success: JSON envelope with ok, error (null), data fields"
Enter
Sleep 200ms
Type "/tmp/demo-ac006-success.sh"
Enter
Wait+Screen@8s /rtt_p99_ms/
Sleep 800ms

Type "# Error path: daemon unreachable -> E-NET-001, exit code 1"
Enter
Sleep 200ms
Type "/tmp/demo-ac006-error.sh"
Enter
Wait+Screen@6s /exit code/
Sleep 2000ms

Hide
Type "pkill -f 'stub-daemon-s502' 2>/dev/null; rm -f /tmp/demo-ac006.sock"
Enter
Sleep 300ms
Show
