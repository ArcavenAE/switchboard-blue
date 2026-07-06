# S-BL.CONSOLE-OBS — sessions.status RPC surfaces per-session quality
# (green|yellow|red|pending) and lifetime miss_count via the console
# daemon's management plane, closing DRIFT-001b (quality) + DRIFT-002
# (missCount) and satisfying BC-2.06.001 v1.7 PC-5 console-half plus
# BC-2.06.002 v1.4 PC-3.
#
# Demonstrates:
#   - QualityIndicator.MissCount lifetime accessor (AC-005 / DRIFT-002).
#     Never resets; increments once per OnMissingFrame call regardless
#     of hysteresis state.
#   - Publisher-level per-session quality wiring — parallel qualities
#     map keyed by session name, lockstep with the sessions map in
#     Publish/Unpublish. Fresh sessions render as "pending" until an
#     observation lands.
#   - Publisher.HandleSessionsStatus transport-agnostic handler with
#     the {name, published_at, quality, miss_count} tuple, sorted by
#     name for the all-sessions query, E-SES-001 for unknown by-name.
#   - cmd/switchboard BuildSessionsHandlers Tier-2 admission shim —
#     RoleControl / RoleConsole callers only, else E-ADM-006 (L1-C4).
#   - Full sbctl↔daemon e2e: RoleConsole admission passes, unknown
#     session returns E-SES-001 envelope, admission-denied returns
#     E-ADM-006, and the primary path publishes 3 misses on a green
#     baseline to observe {quality: yellow, miss_count: 3}.
#
# Traces: S-BL.CONSOLE-OBS; BC-2.06.001 v1.7 PC-5 console-half;
# BC-2.06.002 v1.4 PC-3; RULING-W6TB-C AC-004 + AC-005;
# DRIFT-001b + DRIFT-002.
#
# Usage: vhs sessions-status-quality-and-misscount.tape
# Per POL-004 rendered artifacts are gitignored; only this .tape script is
# checked in.

Output docs/demo-evidence/S-BL.CONSOLE-OBS/sessions-status-quality-and-misscount.gif
Output docs/demo-evidence/S-BL.CONSOLE-OBS/sessions-status-quality-and-misscount.webm

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

# AC-005 / DRIFT-002: QualityIndicator.MissCount is a LIFETIME counter,
# distinct from the private missingFrameCount consecutive-counter used
# for hysteresis. Never resets; increments once per OnMissingFrame call.
Type "grep -n 'MissCount\\|missCount' internal/metrics/metrics.go | head -8"
Enter
Sleep 4s

# The lifetime-counter tests exercise: zero on construct, increments
# per call, NOT reset by Update, monotonic across a mixed workload,
# concurrent-safe under -race.
Type "go test -race -run 'TestQualityIndicator_MissCount' -v ./internal/metrics/ 2>&1 | tail -20"
Enter
Sleep 6s

# AC-004 / DRIFT-001b: Publisher owns a per-session qualities map
# keyed by session name, created in Publish and dropped in Unpublish.
# Fresh sessions render as quality="pending" (no observation yet)
# until OnSessionMeasurement or OnSessionMissingFrame lands.
Type "grep -n 'p.qualities\\|newSessionQuality\\|qualityString' internal/session/session.go internal/session/session_quality.go | head -12"
Enter
Sleep 5s

# The Publisher-level tests (11) cover: empty on startup, published
# sessions render pending, green measurement produces green, misses
# drive yellow, unknown session returns ErrSessionNotFound, unpublish
# drops the indicator, snapshots are sorted, value-copy semantics,
# concurrent observations.
Type "go test -race -run 'TestPublisher_.*Quality\\|TestPublisher_.*Snapshot\\|TestPublisher_.*Measurement\\|TestPublisher_.*Missing' ./internal/session/ 2>&1 | tail -8"
Enter
Sleep 5s

# The transport-agnostic handler surface — sessions.status RPC:
#   Request:  {session_name?: string}
#   Response: {sessions: [{name, published_at, quality, miss_count}]}
#   Errors:   E-SES-001 (unknown), E-ADM-006 (Tier-2 denied via shim)
Type "grep -n 'HandleSessionsStatus\\|SessionsStatusRequest\\|SessionsStatusResponse' internal/session/sessions_status.go | head -8"
Enter
Sleep 4s

# The 7 handler tests exercise the wire contract including the JSON
# round-trip: sessions[], name, published_at, quality (green|yellow|
# red|pending), miss_count. Verifies E-SES-001 wraps ErrSessionNotFound.
Type "go test -race -run 'TestPublisher_HandleSessionsStatus' -v ./internal/session/ 2>&1 | tail -22"
Enter
Sleep 7s

# The daemon-side wiring in cmd/switchboard registers sessions.status
# alongside BuildConsoleHandlers on the console-mode daemon at
# construction time — before Serve starts (F-P2L1-001 invariant).
Type "grep -n 'BuildSessionsHandlers\\|sessions.status' cmd/switchboard/sessions_handlers.go cmd/switchboard/mgmt_wire.go | head -8"
Enter
Sleep 4s

# The 4 e2e tests drive a REAL mgmt.Server: challenge-response
# handshake, Tier-2 admission via verifyConsoleCallerRole, and the
# full wire contract for all-sessions + by-name + unknown + denied.
Type "go test -race -run 'TestSessionsStatus_E2E' -v ./cmd/switchboard/ 2>&1 | tail -14"
Enter
Sleep 6s

# The sbctl CLI dispatches sessions status → sessions.status RPC.
# Optional positional selects one session; empty selects all.
Type "grep -n 'case .status.\\|sessions.status\\|session_name' cmd/sbctl/main.go | head -8"
Enter
Sleep 4s

# The CLI dispatch tests over net.Pipe verify the RPC envelope shape:
# command=\"sessions.status\", args={session_name:<name>} for by-name,
# args=null for the all-sessions query. Guards against misdispatch
# to sessions.list and against leaking an empty session_name.
Type "go test -race -run 'TestSessionsStatus_CLI' -v ./cmd/sbctl/ 2>&1 | tail -10"
Enter
Sleep 5s

# F-P5P6-A-003 sub-verb dispatch: `sessions status` now dispatches
# to the daemon (exit 1 E-NET-001 with no listener), attach/detach
# still exit 2 not-implemented (deferred to S-BL.DISCOVERY-WIRE).
Type "go test -race -run 'TestProductionMain_Sessions_SubVerbValidation' -v ./cmd/sbctl/ 2>&1 | tail -14"
Enter
Sleep 6s

# Full-tree health under -race, targeted at the touched packages.
Type "go test -race -count=1 ./internal/metrics/ ./internal/session/ ./cmd/switchboard/ ./cmd/sbctl/ 2>&1 | tail -5"
Enter
Sleep 6s

Sleep 2s
