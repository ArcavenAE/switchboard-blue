Output AC-002-malformed-nodeidentify-rejected.gif
Output AC-002-malformed-nodeidentify-rejected.webm

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

Type "# AC-002: Malformed NodeIdentify frame (wrong payload_len / msg_kind / reserved / control_type / version / frame_type) → connection closed"
Enter
Sleep 500ms

Type "go test ./cmd/switchboard/... -run 'TestNodeIdentifyHandshake_MalformedNodeIdentify|TestNodeIdentifyHandshake_MalformedChallengeResponse|TestNodeIdentifyHandshake_WrongOuterFrameType' -v -count=1 -timeout 60s"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
