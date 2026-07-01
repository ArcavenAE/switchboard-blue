# AC-002 — sbctl router metrics --svtn=<id> outputs BC-2.06.003 PC-2 schema
# Fields: frame_count, hmac_fail_count, drop_cache_hits, path_distribution
# Test: TestSbctlRouterMetrics_OutputsSVTNMetrics

Output docs/demo-evidence/S-5.02/AC-002-router-metrics-svtn.gif
Output docs/demo-evidence/S-5.02/AC-002-router-metrics-svtn.webm

Set FontFamily "Menlo"
Set FontSize 13
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 60ms

Hide
Type "/tmp/stub-daemon-s502 router-metrics /tmp/demo-ac002.sock &"
Enter
Sleep 600ms
Show

Type "# AC-002: sbctl router metrics --svtn=<id> — BC-2.06.003 PC-2"
Enter
Sleep 300ms
Type "# Outputs: frame_count, hmac_fail_count, drop_cache_hits, path_distribution"
Enter
Sleep 300ms

Type "/tmp/sbctl-s502 --target=/tmp/demo-ac002.sock --key=/tmp/demo-ed25519-key --json router metrics --svtn=abc123 | python3 -m json.tool"
Enter
Wait+Screen@8s /frame_count/
Sleep 2000ms

Hide
Type "pkill -f 'stub-daemon-s502' 2>/dev/null; rm -f /tmp/demo-ac002.sock"
Enter
Sleep 300ms
Show
