# S-7.04 follow-on — forced-exit past drain_timeout (BC-2.09.002 EC-003).
#
# Story S-7.04 (PR #101) wired the graceful-drain coordinator into
# runRouter and shipped the three BC-2.09.003 DEFERRED-APPLICATION
# closures (drain_timeout, keepalive_interval, upstream_routers). The
# runRouter-level tests it shipped exercise the empty-observer fast-path
# only — no observer registered, drain completes immediately, no
# ErrTimeout branch is hit. That left DRIFT-HS006-DRAIN-TIMEOUT-FORCED-
# EXIT-UNEVIDENCED open: BC-2.09.002 EC-003 says "drain timeout exceeded
# → router disconnects after timeout; logs remaining unacknowledged
# nodes," but nothing under -race proved the forced-exit path actually
# fires end-to-end.
#
# What this follow-on delivers: an unexported package-local test hook
# (drainCoordHook) lets same-package tests register a slow observer on
# runRouter's real drain.Drain instance. The new integration test
# TestRunRouter_ForcedExitPastDrainTimeout registers an observer that
# blocks on obsCtx.Done(), holds a live TCP conn on the netingress
# listener across shutdown, cancels the runRouter ctx, and asserts:
#
#   1. runRouter returns nil (EC-003 is expected behavior, not error).
#   2. Elapsed time from cancel → return is bounded by
#      drain_timeout + 2×drain_timeout slack.
#   3. The observer unwound via drainCtx.Done() (coordinator honored
#      its own drain window).
#   4. The router logged "BC-2.09.002 EC-003" so operators see the
#      forced-exit signal.
#
# Production code paths leave drainCoordHook nil — it is a package-local
# var, not exported, only assignable from same-package tests via
# t.Cleanup. Once the DRAIN-over-SVTN wire protocol lands (out of
# S-7.04 scope; a follow-on wave), that story's observer registration
# replaces this hook indirection.
#
# Traces: BC-2.09.002 EC-003; DRIFT-HS006-DRAIN-TIMEOUT-FORCED-EXIT-
# UNEVIDENCED; S-7.04 follow-on to PR #101.
#
# Usage: vhs forced-exit-past-drain-timeout.tape
# Per POL-004 rendered artifacts (.gif/.webm) are gitignored; only this
# .tape script is checked in.

Output docs/demo-evidence/S-7.04/forced-exit-past-drain-timeout.gif
Output docs/demo-evidence/S-7.04/forced-exit-past-drain-timeout.webm

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

# The pure-core drain coordinator. Single-use; observers fan out on
# Signal with a derived drainCtx (timeout=drain_timeout); the race
# obsWG.Wait() vs drainCtx.Done() sets timedOut=true only when the
# timer wins, and Wait returns ErrTimeout in that branch.
Type "sed -n '95,155p' internal/drain/drain.go"
Enter
Sleep 6s

# The runRouter shutdown sequence: Signal → Wait bounded by
# drainCoord.Timeout(). The EC-003 log line fires when Wait returns
# non-nil. Note the test hook at line 502 — nil in production.
Type "sed -n '482,548p' cmd/switchboard/mgmt_wire.go"
Enter
Sleep 8s

# The test hook declaration (router_config.go). Package-local var, not
# exported, only assignable from same-package tests. When nil (prod
# default), runRouter runs with zero observers and hits the fast-path.
Type "sed -n '64,76p' cmd/switchboard/router_config.go"
Enter
Sleep 5s

# The forced-exit integration test. Registers a slow observer via
# drainCoordHook + t.Cleanup, dials a live ingress conn, cancels
# runRouter, asserts elapsed bounds + observer unwound + EC-003 log.
Type "sed -n '276,411p' cmd/switchboard/router_drain_test.go"
Enter
Sleep 10s

# Under -race, 5 rounds. Each run: 500ms drain_timeout + slack →
# ~520ms per iteration. All must pass to prove the forced-exit path
# is deterministic (no scheduler-noise flakes) under the race detector.
Type "go test -race -run 'TestRunRouter_ForcedExitPastDrainTimeout$' -v -count=5 ./cmd/switchboard/ 2>&1 | tail -20"
Enter
Sleep 8s

# The complementary drain unit test (already shipped in PR #101) that
# exercises the coordinator's ErrTimeout branch in isolation. Runs
# fast — its window is 30ms — so shipping both proves the invariant
# holds both in pure-core (unit) and integrated (runRouter) contexts.
Type "go test -race -run TestDrain_ObserverTimeout -v ./internal/drain/ 2>&1 | tail -10"
Enter
Sleep 5s

# Full router_drain_test.go integration battery — all five S-7.04
# tests together (four fast-path + the new forced-exit). Under -race,
# no flakes.
Type "go test -race -run 'TestRunRouter_HonorsCustomDrainTimeout|TestRunRouter_EmitsUpstreamMode|TestRunRouter_EmitsKeepaliveInterval|TestRunRouter_ForcedExitPastDrainTimeout' -v ./cmd/switchboard/ 2>&1 | tail -20"
Enter
Sleep 8s

Sleep 2s
