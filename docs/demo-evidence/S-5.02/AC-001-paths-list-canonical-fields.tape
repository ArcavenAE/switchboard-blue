# AC-001 — sbctl paths list outputs BC-2.06.003 PC-1 schema
# Fields: path_id, router_addr, rtt_ms, rtt_p99_ms (float64 >=10 samples), loss_pct, status
# Test: TestSbctlPathsList_OutputsCanonicalFields

Output docs/demo-evidence/S-5.02/AC-001-paths-list-canonical-fields.gif
Output docs/demo-evidence/S-5.02/AC-001-paths-list-canonical-fields.webm

Set FontFamily "Menlo"
Set FontSize 13
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 60ms

Hide
Type "/tmp/stub-daemon-s502 paths-full /tmp/demo-ac001.sock &"
Enter
Sleep 600ms
Show

Type "# AC-001: sbctl paths list — BC-2.06.003 PC-1 canonical schema"
Enter
Sleep 300ms
Type "# rtt_p99_ms is float64 when >=10 samples"
Enter
Sleep 300ms

Type "/tmp/sbctl-s502 --target=/tmp/demo-ac001.sock --key=/tmp/demo-ed25519-key --json paths list | python3 -m json.tool"
Enter
Wait+Screen@8s /rtt_p99_ms/
Sleep 2000ms

Hide
Type "pkill -f 'stub-daemon-s502' 2>/dev/null; rm -f /tmp/demo-ac001.sock"
Enter
Sleep 300ms
Show
