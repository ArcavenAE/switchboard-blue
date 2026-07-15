# AC-008 — Cold-start acceptance: first datagram for a (SVTNID, NodeAddr) pair always accepted
# AC-009 — Replay/stale discard: non-increasing Sequence rejected post-HMAC
# AC-010 — Forward acceptance: strictly-increasing Sequence updates state and triggers relay
#
# Demonstrates:
#   1. AC-008: with no prior lastSeen[svtnID, nodeAddr] entry, an
#      HMAC-verified datagram is accepted for any declared Sequence
#      (including 0) — fresh router start / first-ever admission.
#   2. AC-009: a second HMAC-verified datagram declaring Sequence <= N
#      (including the exact-replay case Sequence == N) is discarded even
#      though its HMAC passes — no registry mutation, no relay,
#      lastSeen unchanged.
#   3. AC-010: a datagram declaring Sequence > N is accepted, the registry
#      updates, relay is triggered, and lastSeen advances. A restarted
#      access node's freshly-epoch-qualified Sequence is accepted via this
#      forward-acceptance path (F-DWSP4-001 restart-liveness amendment,
#      VP-080 property 5), not the cold-start path.
#
# BC anchors: BC-2.03.001 Postcondition 2; VP-080 properties 1-3, 5; SEC-DW-07
#
# Test coverage (internal/discovery/discovery_wire_test.go):
#   TestVP080_DiscoveryIngest_ColdStartAcceptance
#   TestVP080_DiscoveryIngest_ReplayDiscard_ExactSequence
#   TestVP080_DiscoveryIngest_ReplayDiscard_LowerSequence
#   TestVP080_DiscoveryIngest_ReplayDiscard_NoRelaySideEffect
#   TestVP080_DiscoveryIngest_ForwardAcceptance_AdvancesState
#   TestVP080_DiscoveryIngest_RestartForwardProgress
#
# Usage: vhs docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-008-009-010-replay-sequence-gate.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-008-009-010-replay-sequence-gate.gif
Output docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-008-009-010-replay-sequence-gate.webm

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

# ── AC-008: cold-start acceptance ──────────────────────────────────────────────

Type `echo "=== AC-008: first datagram for a fresh (SVTNID, NodeAddr) pair is always accepted ==="`
Enter
Sleep 300ms

Type `go test -race -run TestVP080_DiscoveryIngest_ColdStartAcceptance -count=1 -v ./internal/discovery/`
Enter
Wait+Line /PASS/
Sleep 1s

# ── AC-009: replay/stale discard (fail path) ───────────────────────────────────

Type `echo "=== AC-009: non-increasing Sequence discarded post-HMAC — exact replay, lower Sequence, no relay side-effect ==="`
Enter
Sleep 300ms

Type `go test -race -run 'TestVP080_DiscoveryIngest_ReplayDiscard_ExactSequence|TestVP080_DiscoveryIngest_ReplayDiscard_LowerSequence|TestVP080_DiscoveryIngest_ReplayDiscard_NoRelaySideEffect' -count=1 -v ./internal/discovery/`
Enter
Wait+Line /PASS/
Sleep 1s

# ── AC-010: forward acceptance, including post-restart epoch-qualified Sequence ─

Type `echo "=== AC-010: strictly-increasing Sequence advances state + triggers relay; restart forward-progress ==="`
Enter
Sleep 300ms

Type `go test -race -run 'TestVP080_DiscoveryIngest_ForwardAcceptance_AdvancesState|TestVP080_DiscoveryIngest_RestartForwardProgress' -count=1 -v ./internal/discovery/`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-008/AC-009/AC-010 PASS ==="`
Enter
Sleep 2s
