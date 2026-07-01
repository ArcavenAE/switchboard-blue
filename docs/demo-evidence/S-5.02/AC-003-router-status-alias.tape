# AC-003 — sbctl router status --target <router> is CLI shim for paths list
# JSON structure identical to sbctl paths list + added quality column (green/yellow/red/pending)
# Test: TestSbctlRouterStatus_IsAliasForPathsList

Output docs/demo-evidence/S-5.02/AC-003-router-status-alias.gif
Output docs/demo-evidence/S-5.02/AC-003-router-status-alias.webm

Set FontFamily "Menlo"
Set FontSize 13
Set Width 1200
Set Height 700
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 60ms

Hide
Type "/tmp/stub-daemon-s502 router-status /tmp/demo-ac003.sock &"
Enter
Sleep 600ms
Show

Type "# AC-003: sbctl router status — alias for paths list, adds quality column"
Enter
Sleep 300ms

Type "# Human-readable table with quality column (green/yellow/red/pending)"
Enter
Sleep 200ms
Type "/tmp/sbctl-s502 --target=/tmp/demo-ac003.sock --key=/tmp/demo-ed25519-key router status"
Enter
Wait+Screen@8s /QUALITY/
Sleep 800ms

Type "# JSON output — same fields as paths list + quality field"
Enter
Sleep 200ms
Type "/tmp/sbctl-s502 --target=/tmp/demo-ac003.sock --key=/tmp/demo-ed25519-key --json router status | python3 -m json.tool"
Enter
Wait+Screen@8s /quality/
Sleep 2000ms

Hide
Type "pkill -f 'stub-daemon-s502' 2>/dev/null; rm -f /tmp/demo-ac003.sock"
Enter
Sleep 300ms
Show
