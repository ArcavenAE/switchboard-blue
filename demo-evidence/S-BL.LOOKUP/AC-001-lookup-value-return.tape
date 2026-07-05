# S-BL.LOOKUP: AdmittedKeySet.Lookup / LookupByPubkey value-return form
# go.md rule 12 (locked-accessor convention)
# Tests: TestLookup_ReturnsBoolTrueOnHit (hit returns value + true)
#        TestLookup_ReturnsBoolFalseOnMiss (miss returns zero-value + false)
#        TestLookupByPubkey_ReturnsBoolTrueOnHit
#        TestLookupByPubkey_ReturnsBoolFalseOnMiss

Output AC-001-lookup-value-return.gif
Output AC-001-lookup-value-return.webm

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

Type "# S-BL.LOOKUP: Lookup/LookupByPubkey return (AdmittedKey, bool) value form"
Enter
Sleep 300ms

Type "# hit returns value + true; miss returns zero-value + false"
Enter
Sleep 300ms

Type "go test -count=1 -v -run 'TestLookup_ReturnsBoolTrueOnHit|TestLookup_ReturnsBoolFalseOnMiss|TestLookupByPubkey_Returns' ./internal/admission/..."
Enter
Sleep 6s
Sleep 2s

Type "# deep-clone fence: public key mutation in returned copy does not leak to store"
Enter
Sleep 300ms

Type "go test -count=1 -v -run 'TestLookup_DeepCloneFence|TestLookupByPubkey_DeepCloneFence' ./internal/admission/..."
Enter
Sleep 6s
Sleep 2s
