# AC-002 — Multicast address derivation: static, deterministic, SVTN-scoped
# AC-003 — Sender-side transmission: TTL=1, no group membership required
#
# Demonstrates:
#   1. MulticastAddrFor(svtnID) returns 239.h0.h1.h2 — the first three bytes
#      of SHA-256(svtnID) — deterministic and static for the SVTN's lifetime,
#      computable independently by every admitted node and the router.
#   2. The access node's Advertise path sends to the SVTN-derived multicast
#      address once per UP+multicast-capable local interface via
#      net.ListenUDP + WriteToUDP (each pinned via setsockopt
#      IP_MULTICAST_IF) — no net.ListenMulticastUDP, no group join.
#   3. The outbound socket's multicast TTL is explicitly set to 1.
#
# BC anchors: BC-2.03.001 Precondition 3 (AC-002); BC-2.03.001 Postcondition 1
# delivery-mechanism note, SEC-DW-08 (AC-003)
#
# Test coverage:
#   TestMulticastAddrFor_Deterministic_SHA256Derived
#   TestDiscovery_Advertise_WriteToMulticast_TTL1_NoGroupJoin
#   (both internal/discovery/discovery_test.go)
#
# Usage: vhs docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-002-003-multicast-addr-and-sender-dispatch.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-002-003-multicast-addr-and-sender-dispatch.gif
Output docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-002-003-multicast-addr-and-sender-dispatch.webm

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

# ── AC-002: deterministic SHA-256-derived multicast address ───────────────────

Type `echo "=== AC-002: MulticastAddrFor is deterministic, SHA-256-derived, SVTN-scoped ==="`
Enter
Sleep 300ms

Type `go test -race -run TestMulticastAddrFor_Deterministic_SHA256Derived -count=1 -v ./internal/discovery/`
Enter
Wait+Line /PASS/
Sleep 1s

Type `echo "=== AC-002 PASS ==="`
Enter
Sleep 1s

# ── AC-003: sender-side TTL=1 dispatch, no group join ──────────────────────────

Type `echo "=== AC-003: sender writes to the multicast group with TTL=1, no net.ListenMulticastUDP ==="`
Enter
Sleep 300ms

Type `go test -race -run TestDiscovery_Advertise_WriteToMulticast_TTL1_NoGroupJoin -count=1 -v ./internal/discovery/`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-002/AC-003 PASS ==="`
Enter
Sleep 2s
