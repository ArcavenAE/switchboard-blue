# SEC-DW-10 — Both per-(SVTNID, NodeAddr) process-lifetime maps are bounded:
#   relayRateCap.last via prune-by-age; RouterIngest.lastSeen via LRU-by-lowest-sequence
#
# Demonstrates:
#   relayRateCap map (cmd/switchboard):
#     1. Map is bounded after stale entries accumulate — after the staleness
#        window, Prune() evicts expired entries and the map shrinks to capacity.
#     2. A stale-pruned key is re-allowed (not permanently banned) — the eviction
#        resets the key to cold-start state.
#     3. Active keys (recently used, still within their rate-cap window) are
#        NOT pruned — prune targets only stale entries.
#
#   RouterIngest.lastSeen map (internal/discovery):
#     4. lastSeen map is bounded at cap — once the cap is reached, the next
#        new (SVTNID, NodeAddr) pair triggers an eviction.
#     5. A replayed sequence for an evicted key is STILL rejected (cold-start
#        acceptance is for the first datagram from a new key, not for replay;
#        a re-admitted cold-start key begins accepting from any Sequence — but
#        if the attacker replays the EXACT sequence the evicted entry held,
#        cold-start acceptance lets it through: EC-006 accepted residual).
#     6. An evicted key's cold-start is accepted — the new (SVTNID, NodeAddr)
#        pair starts fresh and its first datagram passes regardless of Sequence.
#     7. LRU eviction targets the entry with the lowest Sequence — not random,
#        not FIFO, the oldest-by-sequence-progress entry is evicted first.
#
# BC anchor: SEC-DW-10 (process-lifetime map bounding),
#            map-bounding-ruling.md v1.2 Decisions 1 (prune-by-age for relayRateCap),
#            Decision 2 (LRU-by-lowest-sequence for lastSeen), Decision 8 (both bounded)
#            EC-006 (accepted residual: cold-start acceptance after LRU eviction),
#            SEC-DW-07 (lastSeen replay gate)
#
# Test coverage:
#   cmd/switchboard/:
#     TestRelayRateCap_MapBounded_AfterStaleEntries
#     TestRelayRateCap_StalePrunedKey_ReAllowed
#     TestRelayRateCap_ActiveKeys_NotPruned
#   internal/discovery/:
#     TestRouterIngest_LastSeenMap_BoundedAtCap
#     TestRouterIngest_ReplayRejected_AfterCapEviction
#     TestRouterIngest_EvictedKey_ColdStartAccepted
#     TestRouterIngest_LastSeen_LRU_EvictsLowestSequence
#
# Note: TestRelayRateCap_MapBounded_AfterStaleEntries sleeps for the prune
# staleness window (~20s) to verify time-based eviction; the tape uses
# Wait+Line /PASS/ which will wait as long as needed rather than a fixed Sleep.
#
# Usage: vhs docs/demo-evidence/S-BL.DISCOVERY-WIRE/SEC-DW-10-map-bounding.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.DISCOVERY-WIRE/SEC-DW-10-map-bounding.gif
Output docs/demo-evidence/S-BL.DISCOVERY-WIRE/SEC-DW-10-map-bounding.webm

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

# ── relayRateCap: prune-by-age bound ─────────────────────────────────────────

Type `echo "=== SEC-DW-10 (1/4): relayRateCap prune-by-age — map bounded after stale entries ==="`
Enter
Sleep 300ms

Type `go test -run TestRelayRateCap_MapBounded_AfterStaleEntries -count=1 -v ./cmd/switchboard/`
Enter
Wait+Line /PASS/
Sleep 1s

# ── relayRateCap: stale-pruned key re-allowed + active keys not pruned ────────

Type `echo "=== SEC-DW-10 (2/4): relayRateCap — stale-pruned key re-allowed; active keys not pruned ==="`
Enter
Sleep 300ms

Type `go test -race -run 'TestRelayRateCap_StalePrunedKey_ReAllowed|TestRelayRateCap_ActiveKeys_NotPruned' -count=1 -v ./cmd/switchboard/`
Enter
Wait+Line /PASS/
Sleep 1s

# ── lastSeen LRU: map bounded at cap + replay still rejected ─────────────────

Type `echo "=== SEC-DW-10 (3/4): lastSeen LRU — map bounded at cap; replay rejected after eviction ==="`
Enter
Sleep 300ms

Type `go test -race -run 'TestRouterIngest_LastSeenMap_BoundedAtCap|TestRouterIngest_ReplayRejected_AfterCapEviction' -count=1 -v ./internal/discovery/`
Enter
Wait+Line /PASS/
Sleep 1s

# ── lastSeen LRU: cold-start accepted after eviction + LRU targets lowest seq ─

Type `echo "=== SEC-DW-10 (4/4): lastSeen LRU — evicted key cold-start accepted (EC-006); LRU targets lowest-sequence ==="`
Enter
Sleep 300ms

Type `go test -race -run 'TestRouterIngest_EvictedKey_ColdStartAccepted|TestRouterIngest_LastSeen_LRU_EvictsLowestSequence' -count=1 -v ./internal/discovery/`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== SEC-DW-10 PASS ==="`
Enter
Sleep 2s
