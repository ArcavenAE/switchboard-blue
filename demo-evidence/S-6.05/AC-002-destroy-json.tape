# AC-002: sbctl admin svtn destroy --json — single canonical JSON envelope
# BC-2.07.001 postcondition 3; interface-definitions.md §162/§164
# F-P7L1-MED-1 CLOSURE: --json mode emits exactly one envelope, no trailing plain-text line
# Tests: TestSbctlAdmin_SVTNDestroy_HappyPath_JSON (single-envelope integrity + no trailing text)

Output AC-002-destroy-json.gif
Output AC-002-destroy-json.webm

Require go

Set FontFamily "Menlo"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set Padding 20
Set PlaybackSpeed 1

Hide
Type "cd /Users/skippy/work/aae-orc/run/switchboard-blue/.worktrees/S-6.05"
Enter
Sleep 500ms
Show

Type "# AC-002: --json mode emits exactly one canonical JSON envelope (F-P7L1-MED-1 closure)"
Enter
Sleep 300ms

Type "# No trailing plain-text line after the envelope (interface-definitions.md §164)"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlAdmin_SVTNDestroy_HappyPath_JSON ./cmd/sbctl/..."
Enter
Sleep 8s

Type "# Verify: second json.Decode returns io.EOF — confirms exactly one envelope on stdout"
Enter
Sleep 2s
