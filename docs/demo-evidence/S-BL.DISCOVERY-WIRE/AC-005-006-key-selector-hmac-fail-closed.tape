# AC-005 — Fixed-offset key-selector extraction precedes full body decode
# AC-006 — HMAC-first fail-closed verification with unified reject sentinel
#
# Demonstrates:
#   1. The router-side ingest path extracts SVTNID/NodeAddr via direct
#      byte-slice indexing (body[0:16], body[16:24]) — never via
#      decodeBody() — before HMAC verification runs.
#   2. The HMAC computation covers the complete raw body, not merely the
#      24-byte key-selector prefix — tampering the session list beneath an
#      otherwise-valid tag is detected.
#   3. A raw datagram shorter than the 32-byte tag+key-selector minimum is
#      rejected before any key lookup; the full valid-frame minimum with
#      the widened uint64 Sequence field is 42 bytes.
#   4. A lookup-miss (unknown NodeAddr/SVTN) and an HMAC-tag mismatch
#      (known NodeAddr, wrong key) are indistinguishable to an external
#      observer — both resolve to ErrInvalidHMACTag, no distinguishing
#      signal, no registry mutation, no relay — fail-closed.
#
# BC anchors: BC-2.03.001 Postcondition 5; SEC-DW-01 (HIGH, MANDATORY), SEC-DW-05
#
# Test coverage (internal/discovery/discovery_wire_test.go):
#   TestRouterIngest_KeySelectorExtraction_FixedOffset_NoFullDecodeBeforeAuth
#   TestRouterIngest_HMACCoversFullBody_TamperInSessionListDetected
#   TestRouterIngest_ShortDatagram_RejectedBeforeLookup
#   TestRouterIngest_FullValidFrameMinimum_42Bytes
#   TestRouterIngest_LookupMissAndTagMismatch_IndistinguishableRejection
#
# Usage: vhs docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-005-006-key-selector-hmac-fail-closed.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-005-006-key-selector-hmac-fail-closed.gif
Output docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-005-006-key-selector-hmac-fail-closed.webm

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

# ── AC-005: fixed-offset key-selector extraction, HMAC covers full body ───────

Type `echo "=== AC-005: fixed-offset key-selector extraction precedes decodeBody(); HMAC covers full raw body ==="`
Enter
Sleep 300ms

Type `go test -race -run 'TestRouterIngest_KeySelectorExtraction_FixedOffset_NoFullDecodeBeforeAuth|TestRouterIngest_HMACCoversFullBody_TamperInSessionListDetected' -count=1 -v ./internal/discovery/`
Enter
Wait+Line /PASS/
Sleep 1s

Type `echo "=== AC-005: short datagram rejected before key lookup; 42-byte full-valid-frame minimum ==="`
Enter
Sleep 300ms

Type `go test -race -run 'TestRouterIngest_ShortDatagram_RejectedBeforeLookup|TestRouterIngest_FullValidFrameMinimum_42Bytes' -count=1 -v ./internal/discovery/`
Enter
Wait+Line /PASS/
Sleep 1s

Type `echo "=== AC-006: lookup-miss and HMAC-tag-mismatch are indistinguishable — both fail closed ==="`
Enter
Sleep 300ms

Type `go test -race -run TestRouterIngest_LookupMissAndTagMismatch_IndistinguishableRejection -count=1 -v ./internal/discovery/`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-005/AC-006 PASS ==="`
Enter
Sleep 2s
