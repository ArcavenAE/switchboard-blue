# S-6.07 AC-001/AC-002: sbctl admin svtn create — happy path + output fields
# BC-2.07.001 postcondition 1 + PC-1/PC-2
# Tests: TestSbctlAdmin_SvtnCreate_CLI (happy-path CLI dispatch)
#        TestSbctlAdmin_SvtnCreate_SuccessOutputsSVTNIDAndFingerprint (output fields)

Output AC-001-002-svtn-create-cli.gif
Output AC-001-002-svtn-create-cli.webm

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

Type "# S-6.07 AC-001/AC-002: sbctl admin svtn create — CLI dispatch + output fields"
Enter
Sleep 300ms

Type "# AC-001: BuildAdminHandlers registers admin.svtn.create handler"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlAdmin_SvtnCreate_CLI ./cmd/sbctl/..."
Enter
Sleep 6s
Sleep 2s

Type "# AC-002: success output contains svtn_id + bootstrap_fingerprint"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlAdmin_SvtnCreate_SuccessOutputsSVTNIDAndFingerprint ./cmd/sbctl/..."
Enter
Sleep 6s
Sleep 2s
