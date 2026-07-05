# S-W5.04 AC-002/AC-003: rtt_p99_ms union serialization + PathEntry.status derivation
# BC-2.06.003 EC-003 + postcondition 1 Ruling-9
# Tests: TestPathEntry_RTTValueSerialization (AC-002: "pending" sentinel serialization)
#        TestPathEntry_StatusFromDegraded (AC-003: status derived from degraded flag)
#        TestRTTValue_RoundTrip (AC-002: round-trip through JSON)

Output AC-002-003-rtt-p99-status.gif
Output AC-002-003-rtt-p99-status.webm

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

Type "# S-W5.04 AC-002/003: rtt_p99_ms union type + PathEntry status derivation"
Enter
Sleep 300ms

Type "# AC-002: rtt_p99_ms serializes as float or 'pending' string sentinel"
Enter
Sleep 300ms

Type "go test -count=1 -v -run 'TestPathEntry_RTTValueSerialization|TestRTTValue_RoundTrip' ./internal/metrics/..."
Enter
Sleep 6s
Sleep 2s

Type "# AC-003: PathEntry.status derived from degraded flag (Ruling-9)"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestPathEntry_StatusFromDegraded ./internal/metrics/..."
Enter
Sleep 6s
Sleep 2s
