# S-6.01 AC-001/AC-003/AC-004: config validation — tick_interval, required fields, validate before socket
# BC-2.05.001 postconditions
# Tests: TestConfigValidate_RejectsOutOfRangeTickInterval (AC-001)
#        TestConfigValidate_RejectsMissingRequiredFields (AC-002)
#        TestConfigValidate_BeforeSocketOpen (AC-004: validate before binding socket)

Output AC-001-004-config-validation.gif
Output AC-001-004-config-validation.webm

Require go

Set FontFamily "Menlo"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set Padding 20
Set PlaybackSpeed 1

Hide
Type "cd /Users/skippy/work/aae-orc/run/switchboard-blue"
Enter
Sleep 500ms
Show

Type "# S-6.01: config validation — rejects invalid ranges + required fields"
Enter
Sleep 300ms

Type "# AC-001: tick_interval outside [100ms, 10s] → E-CFG-001"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestConfigValidate_RejectsOutOfRangeTickInterval ./internal/config/..."
Enter
Sleep 6s
Sleep 1s

Type "# AC-002: missing required fields → E-CFG-001"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestConfigValidate_RejectsMissingRequiredFields ./internal/config/..."
Enter
Sleep 6s
Sleep 1s

Type "# AC-004: validation runs BEFORE socket is opened (fail early)"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestConfigValidate_BeforeSocketOpen ./internal/config/..."
Enter
Sleep 6s
Sleep 2s
