Output AC-006-snapshot-json-roundtrip.gif
Output AC-006-snapshot-json-roundtrip.webm

Set Shell "bash"
Set FontSize 14
Set FontFamily "Menlo"
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set Padding 20

Hide
Type "cd /Users/skippy/work/aae-orc/run/switchboard-blue/.worktrees/S-BL.ADMISSION-SYNC-WIRE"
Enter
Type "clear"
Enter
Show

Type "# AC-006: snapshot JSON round-trip: schema_version:1, correct encoding, no FrameAuthKey/nonces"
Enter
Sleep 500ms

Type "go test ./cmd/switchboard/... -run 'TestSnapshot_JSON_FieldEncoding|TestSnapshot_RoundTrip|TestSnapshot_NoFrameAuthKey' -v -count=1"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
