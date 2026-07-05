# S-BL.LOOKUP: race detector clean on admission/svtnmgmt packages
# All callers migrated to (T, bool) form; no pointer aliasing
# Tests: go test -race ./internal/svtnmgmt/... + ./internal/admission/...

Output AC-002-race-clean.gif
Output AC-002-race-clean.webm

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

Type "# S-BL.LOOKUP: race-detector clean after value-return migration"
Enter
Sleep 300ms

Type "# error path: confirm deep-clone keeps store isolation under concurrent reads"
Enter
Sleep 300ms

Type "go test -race -count=1 -v -run TestLookup_Hit_AdmittedFieldTrackedAfterAdmit_Whitebox ./internal/admission/..."
Enter
Sleep 6s
Sleep 2s

Type "# svtnmgmt callers: ExpireKey, CallerKeyRole, CallerKeyRoleActive, IsRegisteredAnyState"
Enter
Sleep 300ms

Type "go test -race -count=1 -run TestSVTNManager_LookupMigration ./internal/svtnmgmt/..."
Enter
Sleep 6s
Sleep 2s
