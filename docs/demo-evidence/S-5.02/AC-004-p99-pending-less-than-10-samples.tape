# AC-004 — <10 RTT samples -> rtt_p99_ms: "pending" (string sentinel per EC-003)
# Test: TestSbctlPathsList_P99Pending_LessThan10Samples

Output docs/demo-evidence/S-5.02/AC-004-p99-pending-less-than-10-samples.gif
Output docs/demo-evidence/S-5.02/AC-004-p99-pending-less-than-10-samples.webm

Set FontFamily "Menlo"
Set FontSize 13
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 60ms

Hide
Type "/tmp/stub-daemon-s502 paths-pending /tmp/demo-ac004.sock &"
Enter
Sleep 600ms
Show

Type "# AC-004: rtt_p99_ms is string 'pending' when < 10 RTT samples (BC-2.06.003 EC-003)"
Enter
Sleep 300ms

Type "/tmp/sbctl-s502 --target=/tmp/demo-ac004.sock --key=/tmp/demo-ed25519-key --json paths list | python3 -m json.tool"
Enter
Wait+Screen@8s /pending/
Sleep 2000ms

Hide
Type "pkill -f 'stub-daemon-s502' 2>/dev/null; rm -f /tmp/demo-ac004.sock"
Enter
Sleep 300ms
Show
