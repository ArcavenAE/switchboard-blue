# AC-002 — netingress OnAccept seam: per-node connection registers in send map;
# writer goroutine is sole writer; cleanup deregisters
#
# Demonstrates:
#   1. netingress.Serve owns DATA creation: allocates IfaceID, creates Send/Done, and
#      populates a fully-built NodeHandle BEFORE calling OnAccept for each ADMITTED
#      connection; nodeConnHook records a nodeConnRegistered event.
#   2. On conn close, the behavior-cleanup func deletes the sync.Map entry, fires
#      nodeConnHook(nodeConnRemoved), and closes done via doneOnce — send is NEVER closed.
#
# BC anchors: BC-2.09.002 PC-1; placement note Q-SEAM (ownership split, F-DW-SP3-001)
#
# Test coverage:
#   TestNetingress_OnAccept_RegistersNodeHandle       (cmd/switchboard/router_drain_wire_test.go)
#   TestRunRouter_NodeConnClose_CleansUpSendMap        (cmd/switchboard/router_drain_wire_test.go)
#
# Usage: vhs docs/demo-evidence/S-7.04-FU-DRAIN-WIRE/AC-002-onaccept-seam-registration-cleanup.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-7.04-FU-DRAIN-WIRE/AC-002-onaccept-seam-registration-cleanup.gif
Output docs/demo-evidence/S-7.04-FU-DRAIN-WIRE/AC-002-onaccept-seam-registration-cleanup.webm

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

# ── AC-002: OnAccept seam registration + cleanup ──────────────────────────────

Type `echo "=== AC-002: OnAccept registers NodeHandle; conn close cleans up send map ==="`
Enter
Sleep 300ms

Type `go test -race -run 'TestNetingress_OnAccept_RegistersNodeHandle|TestRunRouter_NodeConnClose_CleansUpSendMap' -count=1 -v ./cmd/switchboard/`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-002 PASS ==="`
Enter
Sleep 2s
