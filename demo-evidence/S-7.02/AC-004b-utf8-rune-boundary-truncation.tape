# AC-004b: 255-byte session name cap with UTF-8 rune-boundary truncation
# BC-2.03.003 postcondition 2 + EC-001 (RULING-W6TB-I, RULING-W6TB-J)
# Tests: TestDiscovery_Encode_SessionName255ByteCap (255 accept, 256 truncated)
#        TestPropPresenceAdvertisement_TruncatesOversize (maximality + rune-boundary)
#        TestPropPresenceAdvertisement_RejectsEmptyOrInvalidUTF8 (error path)

Output AC-004b-utf8-rune-boundary-truncation.gif
Output AC-004b-utf8-rune-boundary-truncation.webm

Require go

Set FontFamily "Menlo"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set Padding 20
Set PlaybackSpeed 1

Hide
Type "cd /Users/skippy/work/switchboard-blue/.worktrees/S-7.02"
Enter
Sleep 500ms
Show

Type "# AC-004b: 255 bytes accepted; 256 bytes truncated to 252 + ellipsis (U+2026)"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestDiscovery_Encode_SessionName255ByteCap ./internal/discovery/..."
Enter
Sleep 8s

Type "# VP-055: oversize names truncated at rune boundary; result ends with ellipsis"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestPropPresenceAdvertisement_TruncatesOversize ./internal/discovery/..."
Enter
Sleep 10s

Type "# error path: empty or non-UTF-8 names are rejected (err != nil)"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestPropPresenceAdvertisement_RejectsEmptyOrInvalidUTF8 ./internal/discovery/..."
Enter
Sleep 8s
