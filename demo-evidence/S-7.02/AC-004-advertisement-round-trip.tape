# AC-004: Advertisement round-trip stability — Encode(Decode(payload)) == payload
# BC-2.03.003 invariant 1
# Tests: TestDiscovery_AdvertisementRoundTrip
#        TestPropPresenceAdvertisement_RoundTrip (VP-055 property test)

Output AC-004-advertisement-round-trip.gif
Output AC-004-advertisement-round-trip.webm

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

Type "# AC-004: Encode(Decode(payload)) == payload for all valid advertisements"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestDiscovery_AdvertisementRoundTrip ./internal/discovery/..."
Enter
Sleep 8s

Type "# VP-055 property: round-trip holds for all valid UTF-8 names 1-255 bytes"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestPropPresenceAdvertisement_RoundTrip ./internal/discovery/..."
Enter
Sleep 10s
