# AC-014 — DISCOVERY_RELAY frame assembly: control_type=0x03 payload layout
# AC-015 — Zero HMACTag on relay frame: connection-trust boundary
# AC-016 — Payload is re-serialized, never a raw retransmission of hop-1 bytes
#
# Demonstrates:
#   1. AC-014: the relay frame is a FrameTypeCtl (0x03) outer frame whose
#      payload begins control_type=0x03, version=0x01, reserved=0x0000;
#      bytes 4-11 carry NodeAddr, bytes 12-19 the epoch-qualified uint64
#      Sequence, bytes 20-21 the session count, bytes 22+ the session
#      list. SVTNID is not repeated — it lives in OuterHeader.SVTNID.
#      Frame assembly is a pure function, independent of any live
#      connection or dispatch mechanism.
#   2. AC-015: the relay frame's OuterHeader.HMACTag is the zero value —
#      matching the DRAIN precedent exactly. No per-frame HMAC is computed
#      for hop-2; trust derives from the already-admitted TCP connection.
#   3. AC-016: the payload bytes are freshly constructed from the decoded
#      fields — never a byte-for-byte copy of hop-1's raw UDP datagram —
#      and hop-1's original HMAC tag never appears in the relay frame.
#
# Also exercised: a full hop-1-decode -> hop-2-assemble round-trip, and the
# oversize-payload guard (panics rather than silently truncating/corrupting
# the frame on a payload that cannot fit the wire format).
#
# BC anchors: BC-2.01.008 Postcondition 2, Postcondition 3, Invariant 5 (DI-007);
# BC-2.03.001 Postcondition 5 (relay/connection-trust note); SEC-DW-08 (hop-2 half)
#
# Test coverage (cmd/switchboard/discovery_relay_wire_test.go):
#   TestAssembleDiscoveryRelayFrame_PayloadLayout
#   TestAssembleDiscoveryRelayFrame_ZeroHMACTag
#   TestAssembleDiscoveryRelayFrame_NotRawHop1Bytes
#   TestAssembleDiscoveryRelayFrame_IngestRelayAdvertisement_RoundTrip
#   TestAssembleDiscoveryRelayFrame_PayloadOversize_Panics
#
# Usage: vhs docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-014-015-016-relay-frame-assembly.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-014-015-016-relay-frame-assembly.gif
Output docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-014-015-016-relay-frame-assembly.webm

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

# ── AC-014: payload layout, control_type=0x03 ──────────────────────────────────

Type `echo "=== AC-014: DISCOVERY_RELAY payload layout — control_type=0x03, NodeAddr/Sequence/count/sessions ==="`
Enter
Sleep 300ms

Type `go test -race -run TestAssembleDiscoveryRelayFrame_PayloadLayout -count=1 -v ./cmd/switchboard/`
Enter
Wait+Line /PASS/
Sleep 1s

# ── AC-015: zero HMACTag on relay frame ────────────────────────────────────────

Type `echo "=== AC-015: relay frame's OuterHeader.HMACTag is the zero value — connection-trust boundary ==="`
Enter
Sleep 300ms

Type `go test -race -run TestAssembleDiscoveryRelayFrame_ZeroHMACTag -count=1 -v ./cmd/switchboard/`
Enter
Wait+Line /PASS/
Sleep 1s

# ── AC-016: re-serialized payload, never raw hop-1 retransmission ─────────────

Type `echo "=== AC-016: payload is freshly re-serialized, never a raw hop-1-byte retransmission ==="`
Enter
Sleep 300ms

Type `go test -race -run TestAssembleDiscoveryRelayFrame_NotRawHop1Bytes -count=1 -v ./cmd/switchboard/`
Enter
Wait+Line /PASS/
Sleep 1s

# ── Supporting: hop-1-decode -> hop-2-assemble round-trip, oversize-payload guard ─

Type `echo "=== Supporting: full round-trip + oversize-payload panic guard ==="`
Enter
Sleep 300ms

Type `go test -race -run 'TestAssembleDiscoveryRelayFrame_IngestRelayAdvertisement_RoundTrip|TestAssembleDiscoveryRelayFrame_PayloadOversize_Panics' -count=1 -v ./cmd/switchboard/`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-014/AC-015/AC-016 PASS ==="`
Enter
Sleep 2s
