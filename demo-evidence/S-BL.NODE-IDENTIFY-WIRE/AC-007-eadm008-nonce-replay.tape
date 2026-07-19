Output AC-007-eadm008-nonce-replay.gif
Output AC-007-eadm008-nonce-replay.webm

Set Shell "bash"
Set FontSize 14
Set FontFamily "Menlo"
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set Padding 20

Hide
Type "cd /Users/skippy/work/aae-orc/run/switchboard-blue/.worktrees/S-BL.NODE-IDENTIFY-WIRE"
Enter
Type "clear"
Enter
Show

Type "# AC-007: ErrNonceReplay → connection closed E-ADM-008 (nonce already consumed)"
Enter
Sleep 500ms

Type "go test ./cmd/switchboard/... -run 'TestNodeIdentifyHandshake_ErrNonceReplay|TestNodeIdentifyHandshake_NonceReplay_E_ADM_008_Logged' -v -count=1 -timeout 30s"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
