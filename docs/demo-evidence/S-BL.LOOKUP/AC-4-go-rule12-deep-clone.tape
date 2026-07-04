# AC-4 — go.md rule 12 compliance: deep-clone of ed25519.PublicKey ([]byte) field
# Demonstrates: returned AdmittedKey has PublicKey backed by an independent array;
# mutation of the returned copy does NOT corrupt the store's live entry.
# Both Lookup and LookupByPubkey inherit the deep-clone guarantee.
# Traces: AC-4 (deep-clone), go.md rule 12.
#
# Usage: vhs AC-4-go-rule12-deep-clone.tape

Output docs/demo-evidence/S-BL.LOOKUP/AC-4-go-rule12-deep-clone.gif
Output docs/demo-evidence/S-BL.LOOKUP/AC-4-go-rule12-deep-clone.webm

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
Type "clear"
Enter
Sleep 1s
Show

# Show the deep-clone implementation in admission.go
Type "grep -n 'deep.clone\|PublicKey.*append\|cp\\.PublicKey' internal/admission/admission.go"
Enter
Sleep 3s

# Run deep-clone fence tests for both Lookup and LookupByPubkey
Type "go test ./internal/admission/... -run 'TestLookup_DeepCloneFence_PublicKeyMutationDoesNotLeak|TestLookupByPubkey_DeepCloneFence_PublicKeyMutationDoesNotLeak' -v -race 2>&1 | grep -E 'RUN|PASS|FAIL|^ok|---'"
Enter
Sleep 6s

# Error path: race detector on concurrent register + lookup (no data race expected)
Type "go test ./internal/admission/... -run 'TestLookup_ConcurrentRegisterRace|TestLookupByPubkey_ConcurrentSameEntryRegistration' -v -race 2>&1 | grep -E 'RUN|PASS|FAIL|^ok|DATA RACE|---'"
Enter
Sleep 8s

Sleep 2s
