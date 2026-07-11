# AC-002 — FrameCallback wired to OnFrameArrival + import perimeter
#
# Demonstrates:
#   1. runRouter wires Connector.SetFrameCallback to routing.OnFrameArrival
#   2. Frames received on the PE connection are forwarded to OnFrameArrival
#   3. upstreamdial package import perimeter is clean (no routing import allowed)
#
# BC anchors: BC-2.09.001 (frame forwarding), import perimeter enforcement
#
# Test coverage:
#   TestRunRouter_PE_FrameCallback_WiredToOnFrameArrival (cmd/switchboard/router_pe_receive_loop_test.go)
#   TestUpstreamdialImportPerimeter                      (internal/upstreamdial/import_perimeter_test.go)
#
# Usage: vhs docs/demo-evidence/S-BL.PE-RECEIVE-LOOP/AC-002-framecallback-wired.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.PE-RECEIVE-LOOP/AC-002-framecallback-wired.gif
Output docs/demo-evidence/S-BL.PE-RECEIVE-LOOP/AC-002-framecallback-wired.webm

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

# ── Step 1: FrameCallback wired to OnFrameArrival (integration) ───────────────

Type `echo "=== AC-002: FrameCallback wired to OnFrameArrival ==="`
Enter
Sleep 300ms

Type `go test ./cmd/switchboard/ -run TestRunRouter_PE_FrameCallback_WiredToOnFrameArrival -count=1 -v`
Enter
Wait+Line /PASS/
Sleep 2s

# ── Step 2: import perimeter (no routing import in upstreamdial) ───────────────

Type `echo "=== AC-002: upstreamdial import perimeter ==="`
Enter
Sleep 300ms

Type `go test ./internal/upstreamdial/ -run TestUpstreamdialImportPerimeter -count=1 -v`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-002 PASS ==="`
Enter
Sleep 2s
