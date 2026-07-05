# S-BL.PATH-TRACKER-WIRING AC-001 — pathTrackerSource observes every
# routing-registry admission via a typed forwarding-entry hook, without
# routing importing paths, and stays race-clean under concurrent
# register+read pressure.
#
# Demonstrates:
#   - Hook contract: routing.Router.ForwardingEntryHook fires exactly
#     once per RegisterForwardingEntry, receives the correct (SVTN,
#     endpoint) tuple, and is a safe no-op when nil (BC-2.06.003 v1.15
#     PC-1; S-W5.04 Ruling-6).
#   - Composition: newPathTrackerSourceFromRouter installs the hook
#     end-to-end so AllSnapshots on the source reflects admissions on
#     the router.
#   - Ruling-11 fold: pathTrackerSource.mu (RWMutex) guards Register +
#     AllSnapshots; the writer story merges into this construct.
#   - Race clean: 4 writers x 4 readers x 128 writes each under -race
#     produces no data-race report and no torn snapshots (go.md rule
#     12 — no leaked pointers into locked state).
#
# Traces: S-BL.PATH-TRACKER-WIRING AC-001 (hook wire); BC-2.06.003
# v1.15 PC-1; S-W5.04 Ruling-6; wave-6-tranche-a-scope-rulings Ruling-11;
# ARCH-08 §6 (routing pos 5 MUST NOT import paths pos 8); go.md rule 12.
#
# Usage: vhs AC-001-hook-wire-race-clean.tape
# Per POL-004 rendered artifacts are gitignored; only this .tape script is
# checked in.

Output docs/demo-evidence/S-BL.PATH-TRACKER-WIRING/AC-001-hook-wire-race-clean.gif
Output docs/demo-evidence/S-BL.PATH-TRACKER-WIRING/AC-001-hook-wire-race-clean.webm

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

# The forwarding-entry hook — a typed callback stored on routing.Router
# that fires under the write lock. Zero paths import — DAG preserved.
Type "grep -n ForwardingEntryHook internal/routing/routing.go | head -8"
Enter
Sleep 4s

# Hook contract tests: fires exactly once, receives the tuple, nil is
# a safe no-op, concurrent Register+hook races clean.
Type "go test -race -run TestForwardingEntryHook -v ./internal/routing/ 2>&1 | tail -30"
Enter
Sleep 6s

# cmd/switchboard side: pathTrackerSource carries the RWMutex (Ruling-11
# fold) and returns value-copy snapshots (go.md rule 12).
Type "grep -n 'pathTrackerSource\\|sync.RWMutex\\|AllSnapshots' cmd/switchboard/metrics_wire.go | head -10"
Enter
Sleep 4s

# Idempotent identity across re-Register — RTT witness proves the same
# tracker instance is reused, not overwritten.
Type "go test -race -run TestRegister_IdempotentIdentityAcrossReRegister -v ./cmd/switchboard/ 2>&1 | tail -12"
Enter
Sleep 5s

# AllSnapshots decouples returned state from internal state.
Type "go test -race -run TestAllSnapshots_ReturnsValueCopies -v ./cmd/switchboard/ 2>&1 | tail -12"
Enter
Sleep 5s

# The race-clean load-bearing test: 4 writers x 4 readers x 128 writes
# each, run under -race. This is the Ruling-11 (folded writer story)
# regression fence.
Type "go test -race -run TestConcurrentRegisterAndAllSnapshots_RaceClean -v ./cmd/switchboard/ 2>&1 | tail -12"
Enter
Sleep 6s

# End-to-end composition: newPathTrackerSourceFromRouter installs the
# hook so router admissions surface on the source.
Type "go test -race -run TestNewPathTrackerSourceFromRouter_InstallsHookEndToEnd -v ./cmd/switchboard/ 2>&1 | tail -12"
Enter
Sleep 5s

# Full package tests — 3 rounds under -race for concurrency clean on
# both touched packages.
Type "go test -race -count=3 ./internal/routing/ ./cmd/switchboard/ 2>&1 | tail -6"
Enter
Sleep 6s

Sleep 2s
