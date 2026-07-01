Output AC-005-incomplete-last-group-no-parity.gif
Output AC-005-incomplete-last-group-no-parity.webm

Require go

Set FontFamily "Menlo"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set Padding 20
Set PlaybackSpeed 1

Hide
Type "cd /Users/skippy/work/switchboard-blue/.worktrees/S-7.01"
Enter
Sleep 500ms
Show

Type "# AC-005: FEC.Encode does not emit parity for incomplete last group"
Enter
Sleep 500ms

Type "go test -count=1 -v -run TestFEC_Encode_IncompleteLastGroup_NoParity ./internal/arq/"
Enter
Sleep 8s
