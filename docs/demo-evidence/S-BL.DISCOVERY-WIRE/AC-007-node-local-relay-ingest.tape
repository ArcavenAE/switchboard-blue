# AC-007 — Node-local relay-ingest: ReceiveAdvertisement retired,
#          ErrSVTNMismatch relocated as defense-in-depth
#
# Demonstrates:
#   1. The new node-side relay-ingest function decodes the hop-2
#      DISCOVERY_RELAY payload with no per-frame HMAC — trust derives from
#      the admitted TCP connection (AC-015), not a cryptographic check.
#   2. ErrSVTNMismatch survives, relocated: it compares the relay frame's
#      OuterHeader.SVTNID against d.cfg.LocalSVTNID directly — defense in
#      depth against a relay/routing bug, not a crypto boundary.
#   3. On success, the registry is updated via the same replace-on-write
#      semantics the retired Discovery.ReceiveAdvertisement used.
#
# Note (F-DWSP8-001): Discovery.ReceiveAdvertisement, as shipped, is
# deleted outright — it had no caller in the shipped relay topology (the
# router authenticates at hop-1 via AC-005/AC-006; no node receives hop-1
# UDP directly). TestDiscovery_VP045_SVTNIsolation_MultipleScopes is
# retired, not extended.
#
# BC anchor: BC-2.03.001 Postcondition 5 (Ruling 1 point 3, corrected by F-DWSP8-001)
#
# Test coverage (internal/discovery/discovery_wire_test.go):
#   TestDiscovery_IngestRelayAdvertisement_SVTNMismatch_ErrSVTNMismatch
#   TestDiscovery_IngestRelayAdvertisement_NoHMACRequired
#   TestDiscovery_IngestRelayAdvertisement_Success_RegistryReplaceOnWrite
#
# Usage: vhs docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-007-node-local-relay-ingest.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-007-node-local-relay-ingest.gif
Output docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-007-node-local-relay-ingest.webm

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

# ── AC-007: node-local relay-ingest — no per-frame HMAC, relocated SVTN guard ─

Type `echo "=== AC-007: node-side relay-ingest, no per-frame HMAC, ErrSVTNMismatch relocated, registry replace-on-write ==="`
Enter
Sleep 300ms

Type `go test -race -run 'TestDiscovery_IngestRelayAdvertisement_SVTNMismatch_ErrSVTNMismatch|TestDiscovery_IngestRelayAdvertisement_NoHMACRequired|TestDiscovery_IngestRelayAdvertisement_Success_RegistryReplaceOnWrite' -count=1 -v ./internal/discovery/`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-007 PASS ==="`
Enter
Sleep 2s
