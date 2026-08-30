# AC-010: synthetic path trackers report IsActive() true with zero probes
# Evidence: paths.NewPathTracker(1.0, 0.125).IsActive() is true immediately at construction, before any OnProbe call
# Test: TestNewLoopbackPaths_TrackersActiveWithoutProbe

Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-010-pathtracker-active-without-probe.gif
Output docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/AC-010-pathtracker-active-without-probe.webm

Set FontFamily "Menlo"
Set FontSize 13
Set Width 1200
Set Height 700
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 60ms

Hide
Type "clear"
Enter
Sleep 200ms
Show

Type "# AC-010: synthetic path trackers report IsActive() true with zero probes"
Enter
Sleep 300ms
Type "# paths.NewPathTracker(1.0, 0.125).IsActive() is true immediately at construction, before any OnProbe call"
Enter
Sleep 300ms

Type `go test -v -race -run "^TestNewLoopbackPaths_TrackersActiveWithoutProbe$" ./internal/testenv/`
Enter
Wait+Screen@60s /PASS/
Sleep 2000ms
