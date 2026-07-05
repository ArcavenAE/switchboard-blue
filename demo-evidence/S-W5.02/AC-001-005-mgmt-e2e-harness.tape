# S-W5.02: e2e management plane integration harness — all four daemon types
# BC-2.07.002 PC-1/PC-2/PC-3/Inv-1/Inv-2; VP-049
# Note: e2e_test.go uses //go:build integration tag
# Tests demonstrated via: go test -tags integration -v ./cmd/sbctl/...
# Supplemental: non-integration tests that prove auth layer works end-to-end

Output AC-001-005-mgmt-e2e-harness.gif
Output AC-001-005-mgmt-e2e-harness.webm

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

Type "# S-W5.02: e2e management plane across all four daemon types (VP-049)"
Enter
Sleep 300ms

Type "# AC-001-003: connect + auth + execute RPC via integration harness"
Enter
Sleep 300ms

Type "go test -count=1 -v -tags integration -run TestE2E_MgmtPlane ./cmd/sbctl/..."
Enter
Sleep 15s

Type "# AC-004: auth invariant — mgmt server requires HMAC auth; unauthenticated rejected"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlAdmin_UnauthenticatedClient_FailsClosed ./cmd/sbctl/..."
Enter
Sleep 5s
