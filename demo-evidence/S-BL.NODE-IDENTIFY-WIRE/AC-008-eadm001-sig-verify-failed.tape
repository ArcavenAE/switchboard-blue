Output AC-008-eadm001-sig-verify-failed.gif
Output AC-008-eadm001-sig-verify-failed.webm

Set Shell "bash"
Set FontSize 14
Set FontFamily "Menlo"
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set Padding 20

Hide
Type "cd /Users/skippy/work/aae-orc/run/switchboard-blue/.worktrees/S-BL.NODE-IDENTIFY-WIRE"
Enter
Type "clear"
Enter
Show

Type "# AC-008: ErrSignatureVerificationFailed → connection closed E-ADM-001 (wrong signing key)"
Enter
Sleep 500ms

Type "go test ./cmd/switchboard/... ./internal/admission/... -run 'TestNodeIdentifyHandshake_ErrSignatureVerificationFailed|TestNodeIdentifyHandshake_ErrSigVerifyFailed|TestAdmitNode_VerifiesAgainstStoredKey|TestAdmitNode_StoredKeyMatches' -v -count=1 -timeout 30s"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
