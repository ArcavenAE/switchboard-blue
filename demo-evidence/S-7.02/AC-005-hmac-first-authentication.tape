# AC-005: HMAC-first advertisement authentication (RULING-W6TB-H ordering)
# BC-2.03.001 postcondition 5
# Tests: TestDiscovery_Advertise_HMACAuthenticated (valid HMAC accepted)
#        TestDiscovery_Advertise_HMACAuthenticated_TagCorruption (error path)
#        TestDiscovery_Enumerate_SVTNIsolation_ForgedSVTN (forged SVTN -> ErrInvalidHMACTag first)

Output AC-005-hmac-first-authentication.gif
Output AC-005-hmac-first-authentication.webm

Require go

Set FontFamily "Menlo"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set Padding 20
Set PlaybackSpeed 1

Hide
Type "cd /Users/skippy/work/switchboard-blue/.worktrees/S-7.02"
Enter
Sleep 500ms
Show

Type "# AC-005: valid HMAC accepted; invalid HMAC tag -> ErrInvalidHMACTag"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestDiscovery_Advertise_HMACAuthenticated ./internal/discovery/..."
Enter
Sleep 8s

Type "# HMAC-first ordering: forged SVTN gets ErrInvalidHMACTag not ErrSVTNMismatch"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestDiscovery_Enumerate_SVTNIsolation_ForgedSVTN ./internal/discovery/..."
Enter
Sleep 8s
