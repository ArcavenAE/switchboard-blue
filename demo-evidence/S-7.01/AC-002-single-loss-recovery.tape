Output AC-002-single-loss-recovery.gif
Output AC-002-single-loss-recovery.webm

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

Type "# AC-002: FEC.Recover recovers missing frame via XOR (single loss)"
Enter
Sleep 500ms

Type "go test -count=1 -v -run TestFEC_Recover_SingleLoss ./internal/arq/"
Enter
Sleep 8s
