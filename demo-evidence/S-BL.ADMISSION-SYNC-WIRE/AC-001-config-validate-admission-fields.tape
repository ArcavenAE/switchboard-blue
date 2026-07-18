Output AC-001-config-validate-admission-fields.gif
Output AC-001-config-validate-admission-fields.webm

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

Type "# AC-001: Config.Validate() for admission_state_file (E-CFG-015) + router_management_endpoints (E-CFG-016)"
Enter
Sleep 500ms

Type "go test ./internal/config/... -run 'TestConfig_Validate_AdmissionStateFile|TestConfig_Validate_RouterManagement|TestConfig_Validate_ControlAdmission' -v -count=1"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
