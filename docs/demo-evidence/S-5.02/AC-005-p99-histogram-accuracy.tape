# AC-005 — p99 histogram accuracy: P99RTTMs <= true_p99 + max_bucket_width/2
# Evidence: go test -v -run TestBC_2_06_003_P99HistogramAccuracy ./internal/paths/
# Test: TestBC_2_06_003_P99HistogramAccuracy

Output docs/demo-evidence/S-5.02/AC-005-p99-histogram-accuracy.gif
Output docs/demo-evidence/S-5.02/AC-005-p99-histogram-accuracy.webm

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

Type "# AC-005: p99 histogram accuracy test — internal/paths"
Enter
Sleep 300ms
Type "# Verifies: P99RTTMs <= true_p99 + max_bucket_width for all sample distributions"
Enter
Sleep 300ms

Type "go test -v -run TestBC_2_06_003_P99HistogramAccuracy ./internal/paths/"
Enter
Wait+Screen@30s /PASS/
Sleep 2000ms

Hide
Type "cd -"
Enter
Sleep 200ms
Show
