Output AC-011-control-keyset-persistence.gif
Output AC-011-control-keyset-persistence.webm

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

Type "# AC-011: control-side keyset persistence via control_admission_state_file"
Enter
Type "#  — validate, persist-on-mutation, load-on-startup"
Enter
Sleep 500ms

Type "go test ./cmd/switchboard/... ./internal/config/... -run 'TestControlAdmission_PersistOnMutation|TestControlAdmission_LoadAndPushFullSnapshot|TestControlAdmission_MissingFileEmptyKeyset|TestConfig_Validate_ControlAdmissionStateFile' -v -count=1"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
