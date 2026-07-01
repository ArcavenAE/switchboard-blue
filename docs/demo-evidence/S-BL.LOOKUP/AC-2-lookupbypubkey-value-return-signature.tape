# AC-2 — AdmittedKeySet.LookupByPubkey value-return signature
# Demonstrates: LookupByPubkey(svtnID, pubkey) (AdmittedKey, bool) — convenience
# wrapper over Lookup; hit returns (key, true), miss returns (zero, false).
# Also covers: exhaustive outcome table (unknown-pubkey, wrong-svtnID, revoked).
# Traces: AC-2 (value-return signature), go.md rule 12 (deep-clone via Lookup).
#
# Usage: vhs AC-2-lookupbypubkey-value-return-signature.tape

Output docs/demo-evidence/S-BL.LOOKUP/AC-2-lookupbypubkey-value-return-signature.gif
Output docs/demo-evidence/S-BL.LOOKUP/AC-2-lookupbypubkey-value-return-signature.webm

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

# Success path: hit/miss outcomes for LookupByPubkey
Type "go test ./internal/admission/... -run 'TestLookupByPubkey_ReturnsBoolTrueOnHit|TestLookupByPubkey_ReturnsBoolFalseOnMiss|TestAdmittedKeySet_LookupByPubkey' -v 2>&1 | grep -E 'RUN|PASS|FAIL|^ok|---'"
Enter
Sleep 5s

# Error path: exhaustive table (unknown, wrong-svtn, revoked) + deep-clone fence
Type "go test ./internal/admission/... -run 'TestLookupByPubkey_ExhaustiveLookupOutcomeTable|TestLookupByPubkey_DeepCloneFence_PublicKeyMutationDoesNotLeak' -v 2>&1 | grep -E 'RUN|PASS|FAIL|^ok|---'"
Enter
Sleep 5s

Sleep 2s
