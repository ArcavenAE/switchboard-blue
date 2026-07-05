# S-6.07 AC-003/AC-005/AC-006: bootstrap-only authority check + error paths
# BC-2.07.001 Inv-3 + EC-001
# Tests: TestSbctlAdmin_SvtnCreate_NonControlDenied (AC-003: non-control → E-ADM-009)
#        TestSbctlAdmin_SvtnCreate_DuplicateName (AC-005: duplicate → E-SVTN-001)
#        TestSbctlAdmin_SvtnCreate_MissingName (AC-006: missing --name flag → error)

Output AC-003-005-006-authority-errors.gif
Output AC-003-005-006-authority-errors.webm

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

Type "# S-6.07 error paths: authority + duplicate + missing-name"
Enter
Sleep 300ms

Type "# AC-003: non-control-role key receives E-ADM-009"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlAdmin_SvtnCreate_NonControlDenied ./cmd/sbctl/..."
Enter
Sleep 6s
Sleep 1s

Type "# AC-005: duplicate SVTN name → E-SVTN-001 (no stutter)"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlAdmin_SvtnCreate_DuplicateName ./cmd/sbctl/..."
Enter
Sleep 6s
Sleep 1s

Type "# AC-006: missing --name flag exits with error"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlAdmin_SvtnCreate_MissingName ./cmd/sbctl/..."
Enter
Sleep 6s
Sleep 2s
