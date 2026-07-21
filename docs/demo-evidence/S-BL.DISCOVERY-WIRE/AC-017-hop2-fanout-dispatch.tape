# AC-017 — Hop-2 relay fan-out dispatch: SVTN-scoped, exclude-originator,
#           best-effort non-blocking
#
# Demonstrates:
#   1. Target resolution: Router.InterfacesForSVTN(svtnID, excludeNodeAddr)
#      returns every admitted interface on the SVTN except the originating
#      node's — the exclude-one property, SVTN isolation, unknown-SVTN
#      non-nil-empty guarantee, unbound-addr passthrough, and race safety
#      under concurrent Bind/Unbind.  (Task 6a)
#   2. Dispatch: relayDiscoveryFrame fans out the assembled DISCOVERY_RELAY
#      frame to every OTHER admitted node on the SVTN via best-effort
#      non-blocking channel sends; missing send-map entries are silently
#      skipped; full channels do not block the caller; frame bytes equal
#      assembleDiscoveryRelayFrame output.  (Task 6b)
#   3. Split-horizon: ForwardOnAllOtherInterfaces verifies the semantic
#      property in isolation — forward to every interface except the one
#      the datagram arrived on.
#
# BC anchors: BC-2.01.008 Precondition 2 (SVTN-scoped routing),
#             Precondition 3 (exclude-originator);
#             SEC-DW-08 (hop-2 relay dispatch correctness)
#
# Test coverage:
#   internal/routing/:
#     TestInterfacesForSVTN_ExcludesOriginator_ReturnsRest
#     TestInterfacesForSVTN_ExcludesOnlyOriginator
#     TestInterfacesForSVTN_SVTNIsolation
#     TestInterfacesForSVTN_UnknownSVTN_ReturnsNonNilEmptySlice
#     TestInterfacesForSVTN_AllExcluded_ReturnsNonNilEmptySlice
#     TestInterfacesForSVTN_ExcludeUnboundNodeAddr_ReturnsAll
#     TestInterfacesForSVTN_ConcurrentBindUnbind_Race
#     TestSplitHorizon_ForwardOnAllOtherInterfaces
#   cmd/switchboard/:
#     TestRelayDispatch_SVTNScoped_ExcludeOriginator_BestEffortNonBlocking
#
# Usage: vhs docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-017-hop2-fanout-dispatch.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-017-hop2-fanout-dispatch.gif
Output docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-017-hop2-fanout-dispatch.webm

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

# ── Target resolution: exclude-originator property ────────────────────────────

Type `echo "=== AC-017 (1/4): InterfacesForSVTN — exclude-originator and exclude-only-originator ==="`
Enter
Sleep 300ms

Type `go test -race -run 'TestInterfacesForSVTN_ExcludesOriginator_ReturnsRest|TestInterfacesForSVTN_ExcludesOnlyOriginator' -count=1 -v ./internal/routing/`
Enter
Wait+Line /PASS/
Sleep 1s

# ── Target resolution: SVTN isolation and empty-slice guarantees ──────────────

Type `echo "=== AC-017 (2/4): InterfacesForSVTN — SVTN isolation, unknown-SVTN, all-excluded, unbound-addr ==="`
Enter
Sleep 300ms

Type `go test -race -run 'TestInterfacesForSVTN_SVTNIsolation|TestInterfacesForSVTN_UnknownSVTN_ReturnsNonNilEmptySlice|TestInterfacesForSVTN_AllExcluded_ReturnsNonNilEmptySlice|TestInterfacesForSVTN_ExcludeUnboundNodeAddr_ReturnsAll' -count=1 -v ./internal/routing/`
Enter
Wait+Line /PASS/
Sleep 1s

# ── Target resolution: concurrent Bind/Unbind race safety ────────────────────

Type `echo "=== AC-017 (3/4): InterfacesForSVTN — concurrent Bind/Unbind race safety ==="`
Enter
Sleep 300ms

Type `go test -race -run TestInterfacesForSVTN_ConcurrentBindUnbind_Race -count=1 -v ./internal/routing/`
Enter
Wait+Line /PASS/
Sleep 1s

# ── Best-effort non-blocking dispatch + split-horizon ─────────────────────────

Type `echo "=== AC-017 (4/4): relay dispatch (SVTN-scoped, exclude-originator, best-effort non-blocking) + split horizon ==="`
Enter
Sleep 300ms

Type `go test -race -run 'TestRelayDispatch_SVTNScoped_ExcludeOriginator_BestEffortNonBlocking' -count=1 -v ./cmd/switchboard/`
Enter
Wait+Line /PASS/
Sleep 500ms

Type `go test -race -run TestSplitHorizon_ForwardOnAllOtherInterfaces -count=1 -v ./internal/routing/`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-017 PASS ==="`
Enter
Sleep 2s
