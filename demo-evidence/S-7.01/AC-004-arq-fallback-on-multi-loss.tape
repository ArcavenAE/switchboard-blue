Output AC-004-arq-fallback-on-multi-loss.gif
Output AC-004-arq-fallback-on-multi-loss.webm

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

Type "# AC-004: ErrTooManyLosses triggers ARQ retransmit (no silent drop)"
Enter
Sleep 500ms

Type "go test -count=1 -v -run TestFEC_FallbackToARQ_OnMultiLoss ./internal/arq/"
Enter
Sleep 8s
