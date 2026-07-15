# AC-004 — DiscoveryAuthKey derivation: domain-separated from FrameAuthKey
#
# Demonstrates:
#   1. hmac.DeriveDiscoveryKey(nodeAdmissionPubkey, svtnID) computes
#      HKDF-SHA256 with a distinct info label (HKDFInfoDiscovery) from
#      DeriveKey's HKDFInfo — the two derived keys are cryptographically
#      independent for the same (pubkey, svtnID) pair.
#   2. (*routing.Router).DiscoveryAuthKeyFor(svtnID, nodeAddr) returns
#      (key, true) on an admitted lookup and (zero, false) on a miss.
#   3. routing.DeriveDiscoveryKey (sender-side symmetric wrapper) produces
#      the identical output DiscoveryAuthKeyFor computes for the same node's
#      own admitted key — sender and router independently agree.
#   4. (F-DWIP1-001, v1.11) A real Encode -> router-Ingest round-trip
#      succeeds for a genuinely admitted node — closing the gap where a
#      prior regression derived the discovery HMAC key from cleartext
#      SVTNID alone (the anti-pattern DRIFT-W6TBD-001/Ruling 1 rejected).
#   5. (F-DWIP1-001, v1.11) transmitAdvertisement fails closed with
#      ErrMissingNodeAdmissionPubkey when the local node's admission pubkey
#      is empty — no silent fallback to an unauthenticated derivation.
#
# BC anchors: BC-2.03.001 Postcondition 5; SEC-DW-06
#
# Test coverage:
#   TestDeriveDiscoveryKey_DomainSeparatedFromFrameAuthKey (internal/hmac/hmac_test.go)
#   TestDiscoveryAuthKeyFor_LookupSuccessAndMiss (internal/routing/advertisement_hmac_test.go)
#   TestDeriveDiscoveryKey_SenderRouterAgree (internal/routing/advertisement_hmac_test.go)
#   TestDiscovery_EncodeThenRouterIngest_AcceptsRealAdmittedNode (internal/discovery/discovery_wire_test.go)
#   TestDiscovery_Advertise_MissingNodeAdmissionPubkey_FailsClosed (internal/discovery/discovery_test.go)
#
# Usage: vhs docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-004-discovery-key-derivation.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-004-discovery-key-derivation.gif
Output docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-004-discovery-key-derivation.webm

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

# ── AC-004: domain-separated key derivation, sender/router agreement ──────────

Type `echo "=== AC-004: DeriveDiscoveryKey is domain-separated from DeriveKey (distinct HKDF info label) ==="`
Enter
Sleep 300ms

Type `go test -race -run TestDeriveDiscoveryKey_DomainSeparatedFromFrameAuthKey -count=1 -v ./internal/hmac/`
Enter
Wait+Line /PASS/
Sleep 1s

Type `echo "=== AC-004: DiscoveryAuthKeyFor lookup success/miss + sender-router agreement ==="`
Enter
Sleep 300ms

Type `go test -race -run 'TestDiscoveryAuthKeyFor_LookupSuccessAndMiss|TestDeriveDiscoveryKey_SenderRouterAgree' -count=1 -v ./internal/routing/`
Enter
Wait+Line /PASS/
Sleep 1s

Type `echo "=== AC-004 (F-DWIP1-001, v1.11): real Encode -> router-Ingest round-trip for a genuinely admitted node ==="`
Enter
Sleep 300ms

Type `go test -race -run TestDiscovery_EncodeThenRouterIngest_AcceptsRealAdmittedNode -count=1 -v ./internal/discovery/`
Enter
Wait+Line /PASS/
Sleep 1s

Type `echo "=== AC-004 (F-DWIP1-001, v1.11): fail-closed guard on missing local admission pubkey ==="`
Enter
Sleep 300ms

Type `go test -race -run TestDiscovery_Advertise_MissingNodeAdmissionPubkey_FailsClosed -count=1 -v ./internal/discovery/`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-004 PASS ==="`
Enter
Sleep 2s
