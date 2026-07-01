Output AC-001-parity-frame-emission.gif
Output AC-001-parity-frame-emission.webm

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

Type "# AC-001: FEC.Encode produces parity frame (frame_type=fec=0x05)"
Enter
Sleep 500ms

Type "go test -count=1 -v -run TestFEC_Encode_ProducesParityFrame ./internal/arq/"
Enter
Sleep 8s
