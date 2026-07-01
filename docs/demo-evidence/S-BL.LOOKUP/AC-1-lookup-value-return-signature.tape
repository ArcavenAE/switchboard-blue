# AC-1 — AdmittedKeySet.Lookup value-return signature
# Demonstrates: Lookup(svtnID, nodeAddr) (AdmittedKey, bool) — hit returns (key, true),
# miss returns (zero-value, false). Deep-clone fence: returned PublicKey is independent.
# Traces: AC-1 (value-return signature), go.md rule 12 (deep-clone).
#
# Usage: vhs AC-1-lookup-value-return-signature.tape

Output docs/demo-evidence/S-BL.LOOKUP/AC-1-lookup-value-return-signature.gif
Output docs/demo-evidence/S-BL.LOOKUP/AC-1-lookup-value-return-signature.webm

Require go

Set Shell "zsh"
Set FontFamily "Menlo"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful
Set BorderRadius 8
Set Padding 20
Set Framerate 30
Set TypingSpeed 30ms

Hide
Type "cd /Users/skippy/work/switchboard-blue/.worktrees/S-BL.LOOKUP && clear"
Enter
Sleep 1s
Show

# Success path: hit returns (AdmittedKey, true); miss returns (zero, false)
Type "go test ./internal/admission/... -run 'TestLookup_ReturnsBoolTrueOnHit|TestLookup_ReturnsBoolFalseOnMiss|TestLookup_Miss_ReturnsZeroAdmittedKey_AllFields|TestLookup_Hit_AllFieldsMatchRegistration' -v 2>&1 | grep -E 'RUN|PASS|FAIL|^ok|---'"
Enter
Sleep 6s

# Deep-clone fence: mutation of returned PublicKey does NOT leak into store
Type "go test ./internal/admission/... -run 'TestLookup_DeepCloneFence_PublicKeyMutationDoesNotLeak' -v 2>&1 | grep -E 'RUN|PASS|FAIL|^ok|---'"
Enter
Sleep 4s

Sleep 2s
