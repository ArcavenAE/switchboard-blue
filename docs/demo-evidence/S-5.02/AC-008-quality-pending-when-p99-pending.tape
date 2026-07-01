# AC-008 — sbctl router status emits quality: "pending" when rtt_p99_ms == "pending"
# Client-side field derivation in qualityFromPathEntry (BC-2.06.003 v1.7 PC-3 F-M3 + EC-006)
# Test: TestSbctlRouterStatus_QualityPending_WhenP99Pending

Output docs/demo-evidence/S-5.02/AC-008-quality-pending-when-p99-pending.gif
Output docs/demo-evidence/S-5.02/AC-008-quality-pending-when-p99-pending.webm

Set FontFamily "Menlo"
Set FontSize 13
Set Width 1200
Set Height 700
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 60ms

Hide
Type "/tmp/stub-daemon-s502 router-status-pending /tmp/demo-ac008.sock &"
Enter
Sleep 600ms
Show

Type "# AC-008: quality=pending when rtt_p99_ms=pending (BC-2.06.003 v1.7 EC-006)"
Enter
Sleep 300ms

Type "# Human-readable table: QUALITY column shows 'pending'"
Enter
Sleep 200ms
Type "/tmp/sbctl-s502 --target=/tmp/demo-ac008.sock --key=/tmp/demo-ed25519-key router status"
Enter
Wait+Screen@8s /QUALITY/
Sleep 600ms

Type "# JSON output: both rtt_p99_ms and quality are string 'pending'"
Enter
Sleep 200ms
Type "/tmp/sbctl-s502 --target=/tmp/demo-ac008.sock --key=/tmp/demo-ed25519-key --json router status | python3 -m json.tool"
Enter
Wait+Screen@8s /quality/
Sleep 2000ms

Hide
Type "pkill -f 'stub-daemon-s502' 2>/dev/null; rm -f /tmp/demo-ac008.sock"
Enter
Sleep 300ms
Show
