# S-BL.NI AC-001 — network ingress accepts a TCP connection, reads a
# self-delimiting outer-header frame, and dispatches it to RouteFrame.
#
# Demonstrates:
#   - The router's data-plane listener binds cfg.ListenAddr (BC-2.09.003 PC-9
#     no longer deferred).
#   - Frames from an unregistered source drop fail-closed via the routing
#     PATH-A "auth key unavailable" branch (BC-2.05.005 PC-3, BC-2.05.008
#     invariant 5) — E-ADM-016 log lines appear per drop.
#   - The E-ADM-017 per-source alert fires once at threshold=5 through the
#     live network path (S-W3.05 AC-009 — was gated on S-BL.NI).
#   - Append-skip: the sixth frame from the same source does NOT re-fire.
#
# Traces: S-BL.NI (AC-001 ingress→RouteFrame); C-1-W3P1-defer (network-ingress
# listener); PROCESS-GAP-W4 (cross-component lock-ordering — see the
# TestIntegration_ConcurrentRegisterAndRouteRaceClean go test); BC-2.09.003
# PC-9 (cfg.ListenAddr application); S-W3.05 AC-009 (E-ADM-017 live-path).
#
# Usage: vhs AC-001-ingress-routeframe.tape
# Per POL-004 rendered artifacts are gitignored; only this .tape script is
# checked in.

Output docs/demo-evidence/S-BL.NI/AC-001-ingress-routeframe.gif
Output docs/demo-evidence/S-BL.NI/AC-001-ingress-routeframe.webm

Require go

Set Shell "zsh"
Set FontFamily "Menlo"
Set FontSize 14
Set Width 1200
Set Height 720
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

# Show that the previously-skeleton runRouter is now wired end-to-end:
# a real net.Listen is spawned by netingress and every accepted frame is
# handed to routing.RouteFrame.
Type "grep -n 'netingress.Serve\\|dataLn, err := net.Listen' cmd/switchboard/mgmt_wire.go"
Enter
Sleep 3s

# Show the ARCH-09 boundary-layer package that owns the ingress.
Type "sed -n '1,30p' internal/netingress/netingress.go"
Enter
Sleep 4s

# Run the live-path integration test that proves the wire (unregistered
# source → PATH-A drop → E-ADM-016 log → E-ADM-017 fires once).
# This is the assertion listed in S-W3.05 AC-009 that was gated on S-BL.NI.
Type "go test -race -run TestIntegration_EADM017_FiresThroughLiveIngress -v ./internal/netingress/"
Enter
Sleep 8s

# Run the -race concurrency check that consumes PROCESS-GAP-W4:
# 4 register writers × 4 ingress dialers, 200ms burst, no race reports.
Type "go test -race -run TestIntegration_ConcurrentRegisterAndRouteRaceClean -v ./internal/netingress/"
Enter
Sleep 6s

# Unit-level framing coverage — self-delimiting via OuterHeader.PayloadLen,
# clean EOF at header boundary, fail-closed on truncation.
Type "go test -race -run 'TestReadFrame_|TestServeConn_|TestServe_' -v ./internal/netingress/ 2>&1 | tail -20"
Enter
Sleep 6s
