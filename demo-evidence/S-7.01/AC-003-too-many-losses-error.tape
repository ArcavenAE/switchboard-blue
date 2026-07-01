Output AC-003-too-many-losses-error.gif
Output AC-003-too-many-losses-error.webm

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

Type "# AC-003: FEC.Recover returns ErrTooManyLosses when 2+ frames missing"
Enter
Sleep 500ms

Type "go test -count=1 -v -run TestFEC_Recover_TwoLossesFail ./internal/arq/"
Enter
Sleep 8s
