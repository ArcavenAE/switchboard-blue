# S-BL.PATH-FAILED-STATUS AC-001 — admit "failed" to the path status
# enum and derive it from PathSnapshot.Failed via the precedence
# Failed > Degraded > Active, preserving Ruling-9 fall-through and
# S502-DEFER-3 / EC-007 quality-status orthogonality.
#
# Demonstrates:
#   - Liveness latch: PathTracker.failed sets only when a previously-
#     alive path (!firstProbe) crosses consecutiveMissThreshold (3);
#     a never-alive path stays !Active but !Failed (BC-2.06.003 v1.15
#     PC-1 AC-001).
#   - Reactivation: resetRTT clears failed alongside active on a
#     successful probe, so a recovered path returns to status="active"
#     without operator intervention.
#   - Precedence: PathEntryFromSnapshot maps Failed > Degraded > Active.
#     Ruling-9's !Active fall-through is preserved beneath the new
#     Failed branch (case snap.Degraded || !snap.Active -> "degraded").
#   - Enum-closed regression fence: TestStatusEnumClosed accepts
#     exactly {active, degraded, failed}; the Wave-6 panic guard is
#     removed now that "failed" is legitimate.
#   - Orthogonality: status="failed" coexists with quality="pending"
#     (S502-DEFER-3 / EC-007). "failed" NEVER appears as a quality
#     value; quality mapping in sbctl treats failed as red only via
#     an explicit conditional.
#
# Traces: S-BL.PATH-FAILED-STATUS AC-001; BC-2.06.003 v1.15 PC-1;
# BC-2.06.001 AC-002/AC-003; Wave-6 Ruling-4 (lifted); Wave-6 Ruling-9
# (preserved); S502-DEFER-3 / EC-007 (fenced).
#
# Usage: vhs AC-001-failed-status-precedence-orthogonality.tape
# Per POL-004 rendered artifacts are gitignored; only this .tape script is
# checked in.

Output docs/demo-evidence/S-BL.PATH-FAILED-STATUS/AC-001-failed-status-precedence-orthogonality.gif
Output docs/demo-evidence/S-BL.PATH-FAILED-STATUS/AC-001-failed-status-precedence-orthogonality.webm

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

# The precedence chain in metrics: Failed > Degraded > Active,
# preserving Ruling-9's !Active fall-through.
Type "grep -n 'case snap.Failed\\|case snap.Degraded' internal/metrics/handlers.go"
Enter
Sleep 4s

# The liveness latch in paths: PathTracker.failed sets only when
# !firstProbe crosses consecutiveMissThreshold (paths.go:28).
Type "grep -n 'consecutiveMissThreshold\\|t.failed\\|firstProbe' internal/paths/paths.go | head -10"
Enter
Sleep 4s

# Never-alive protection: a path that has never received a successful
# probe stays !Active but !Failed. Failed is a "was alive, went away"
# signal — not a "never came up" signal.
Type "go test -race -run 'TestPathTracker.*Fail\\|TestNever\\|TestReactivation' -v ./internal/paths/ 2>&1 | tail -25"
Enter
Sleep 6s

# Precedence: Failed dominates Degraded dominates Active.
Type "go test -race -run TestStatusFromDegraded -v ./internal/metrics/ 2>&1 | tail -18"
Enter
Sleep 5s

# Enum-closed regression fence: exactly {active, degraded, failed}.
# Wave-6 Ruling-4 lifted; the pre-story panic guard removed.
Type "go test -race -run TestStatusEnumClosed -v ./internal/metrics/ 2>&1 | tail -12"
Enter
Sleep 5s

# Orthogonality: status="failed" coexists with quality="pending".
# S502-DEFER-3 / EC-007 fenced — quality band derives from RTT + loss,
# not from status.
Type "go test -race -run TestPathEntryFromSnapshot_QualityStatusIndependence -v ./internal/metrics/ 2>&1 | tail -12"
Enter
Sleep 5s

# sbctl round-trip: paths_list.go documents status enum as
# active|degraded|failed and router_status.go quality mapping treats
# failed as red only via an explicit conditional.
Type "grep -n 'active.degraded.failed\\|Status ..\\\"failed\\\"' cmd/sbctl/paths_list.go cmd/sbctl/router_status.go | head -6"
Enter
Sleep 5s

# Full package tests — 3 rounds under -race on both touched packages.
Type "go test -race -count=3 ./internal/paths/ ./internal/metrics/ 2>&1 | tail -5"
Enter
Sleep 6s

Sleep 2s
