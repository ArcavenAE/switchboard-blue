# AC-018 — Per-(SVTNID, NodeAddr) ~1/sec relay rate cap at the dispatch-decision
#           point: first datagram relayed, excess silently dropped  (SEC-DW-09)
#
# Demonstrates:
#   1. First relay for a given (SVTNID, NodeAddr) key is forwarded; a second
#      relay within the ~1-second window for the same key is silently dropped
#      (no error, no panic).
#   2. A distinct (SVTNID, NodeAddr) key is NOT capped by a sibling key's
#      window — per-key isolation confirmed.
#   3. A key whose rate-cap window has elapsed is allowed again after the
#      interval passes.
#   4. Suppression counter is non-gating (counter increments on drop but
#      does not change the relay/drop decision for future datagrams).
#   5. No error or panic on a burst of datagrams for the same key.
#
# BC anchor: SEC-DW-09 (relay rate cap — silent drop, per (SVTNID, NodeAddr),
#            ~1/second, at dispatch-decision point)
#
# Test coverage:
#   cmd/switchboard/:
#     TestRelayDispatch_RateCap_PerSVTNNodeAddr_SilentDropFirst
#
# Usage: vhs docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-018-relay-rate-cap.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-018-relay-rate-cap.gif
Output docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-018-relay-rate-cap.webm

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

# ── Per-(SVTNID, NodeAddr) rate cap: first relayed, burst dropped ─────────────

Type `echo "=== AC-018: per-(SVTNID, NodeAddr) relay rate cap — first allowed, burst silently dropped ==="`
Enter
Sleep 300ms

Type `go test -race -run TestRelayDispatch_RateCap_PerSVTNNodeAddr_SilentDropFirst -count=1 -v ./cmd/switchboard/`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-018 PASS ==="`
Enter
Sleep 2s
