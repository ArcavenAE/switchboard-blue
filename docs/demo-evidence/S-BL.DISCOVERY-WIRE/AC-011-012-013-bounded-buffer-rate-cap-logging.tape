# AC-011 — Bounded, fixed-size UDP read buffer sized to realistic usage
# AC-012 — Aggregate rate cap at ingest; FailureCounter reused visibility-only
# AC-013 — Rate-limited, counter-based failure logging
#
# Demonstrates:
#   1. AC-011: a datagram exceeding the sized read buffer is rejected
#      without partial-parse and without reallocation-to-fit.
#   2. AC-012: an aggregate (not per-source) token-bucket cap rejects
#      datagrams once the aggregate rate is exceeded, regardless of
#      declared NodeAddr — rotating the declared NodeAddr across forged
#      datagrams does not evade the cap. FailureCounter fires on
#      HMAC-rejection for operator visibility only; it never gates
#      admission on the attacker-controlled NodeAddr.
#   3. AC-013: HMAC-rejection logging fires only on FailureCounter's own
#      threshold-crossing emission, not unconditionally per rejected
#      packet — distinct from BC-2.05.008's per-packet TCP policy.
#
# BC anchors: BC-2.03.001 Postcondition 5; SEC-DW-02 (MED), SEC-DW-03 (MED), SEC-DW-04 (MED)
#
# Test coverage (internal/discovery/discovery_wire_test.go):
#   TestRouterIngest_OversizedDatagram_RejectedNoPartialParse
#   TestRouterIngest_AggregateRateCap_NotPerSource
#   TestRouterIngest_FailureCounter_VisibilityOnly_NeverGates
#   TestRouterIngest_FailureLogging_ThresholdCrossingOnly_NotPerPacket
#
# Usage: vhs docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-011-012-013-bounded-buffer-rate-cap-logging.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-011-012-013-bounded-buffer-rate-cap-logging.gif
Output docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-011-012-013-bounded-buffer-rate-cap-logging.webm

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

# ── AC-011: oversized datagram rejected, no partial parse ─────────────────────

Type `echo "=== AC-011: datagram exceeding the sized read buffer is rejected, no partial-parse/reallocation ==="`
Enter
Sleep 300ms

Type `go test -race -run TestRouterIngest_OversizedDatagram_RejectedNoPartialParse -count=1 -v ./internal/discovery/`
Enter
Wait+Line /PASS/
Sleep 1s

# ── AC-012: aggregate rate cap, FailureCounter visibility-only ────────────────

Type `echo "=== AC-012: aggregate (not per-source) rate cap; FailureCounter visibility-only, never gates ==="`
Enter
Sleep 300ms

Type `go test -race -run 'TestRouterIngest_AggregateRateCap_NotPerSource|TestRouterIngest_FailureCounter_VisibilityOnly_NeverGates' -count=1 -v ./internal/discovery/`
Enter
Wait+Line /PASS/
Sleep 1s

# ── AC-013: threshold-crossing-only failure logging ────────────────────────────

Type `echo "=== AC-013: HMAC-rejection logging fires only on FailureCounter threshold-crossing, not per-packet ==="`
Enter
Sleep 300ms

Type `go test -race -run TestRouterIngest_FailureLogging_ThresholdCrossingOnly_NotPerPacket -count=1 -v ./internal/discovery/`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-011/AC-012/AC-013 PASS ==="`
Enter
Sleep 2s
