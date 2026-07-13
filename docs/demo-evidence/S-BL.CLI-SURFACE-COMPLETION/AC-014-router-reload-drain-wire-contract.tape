# AC-014 — `router.reload`/`router.drain` wire contract
#
# Demonstrates:
#   1. Both verbs require Tier-1 operator-key authentication only — no stricter
#      Tier-2 gate is available or introduced
#   2. Request args for both: {}. Response data for both: {"accepted": true} —
#      fire-and-forget, no synchronous completion confirmation
#   3. Standard shared connection-error codes: E-NET-001 (unreachable),
#      E-ADM-010 (auth failure) — observed client-side by cmd/sbctl, re-homed
#      per F-CS-SP6-001 since router_control_wire_test.go is server-side and
#      structurally cannot exercise cmd/sbctl's dispatch
#
# BC anchors: Decision 4 (wire contract); BC-2.09.001 v1.2, BC-2.09.002 v1.3
#
# Test coverage:
#   TestRouterReloadDrain_TierOneAuthOnly_FireAndForgetAcceptedTrue (cmd/switchboard/router_control_wire_test.go) — PC-1, PC-2
#   TestRouterReloadDrain_Unreachable_ENET001                        (cmd/sbctl/router_control_test.go) — PC-3
#   TestRouterReloadDrain_AuthFailure_EADM010                        (cmd/sbctl/router_control_test.go) — PC-3
#
# Usage: vhs docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-014-router-reload-drain-wire-contract.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-014-router-reload-drain-wire-contract.gif
Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-014-router-reload-drain-wire-contract.webm

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

# ── Step 1: PC-1/PC-2 — Tier-1-only auth, fire-and-forget {"accepted":true} ───

Type `echo "=== AC-014 PC-1/PC-2: Tier-1-only auth, fire-and-forget accepted:true (server-side) ==="`
Enter
Sleep 300ms

Type `go test -race -run TestRouterReloadDrain_TierOneAuthOnly_FireAndForgetAcceptedTrue -count=1 -v ./cmd/switchboard/...`
Enter
Wait+Line /PASS/
Sleep 2s

# ── Step 2: PC-3 — client-observed E-NET-001 / E-ADM-010 ──────────────────────

Type `echo "=== AC-014 PC-3: client-observed E-NET-001 unreachable, E-ADM-010 auth failure ==="`
Enter
Sleep 300ms

Type `go test -race -run 'TestRouterReloadDrain_Unreachable_ENET001|TestRouterReloadDrain_AuthFailure_EADM010' -count=1 -v ./cmd/sbctl/...`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-014 PASS ==="`
Enter
Sleep 2s
