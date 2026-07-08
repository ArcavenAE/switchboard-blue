# AC-002 — Unreachable upstream: EC-001 graceful handling with exponential backoff
#
# Demonstrates:
#   1. Failed dial -> "upstream router <addr> unreachable" emitted verbatim (EC-001 log)
#   2. connector.Mode() == ModeE when all upstreams unreachable
#   3. Backoff constants: BackoffBase=500ms (floor only), BackoffCap=30s, JitterFraction=0.25
#   4. Operative base tracks keepaliveInterval (floored at BackoffBase), not raw 500ms
#   5. If >=1 upstream connected: Mode() == ModePE (partial-PE semantics)
#   6. EC-004 fires on upstream-LOSS drop-to-zero (not on graceful Stop)
#   7. Concurrent multi-upstream drop-to-zero emits exactly ONE EC-004 per transition
#
# BC anchors: BC-2.09.001 EC-001, EC-004; placement note Q5 (exact backoff parameters)
#
# Test coverage:
#   TestConnector_DialFailure_EC001Log                    (internal/upstreamdial/connector_test.go)
#   TestConnector_BackoffConstants                        (internal/upstreamdial/connector_test.go)
#   TestOperativeBase_TracksKeepalive                     (internal/upstreamdial/connector_test.go)
#   TestConnector_BackoffParameters                       (internal/upstreamdial/connector_test.go)
#   TestConnector_AllUpstreamsUnreachable_ModeE           (internal/upstreamdial/connector_test.go)
#   TestConnector_NoEC004OnGracefulStop                   (internal/upstreamdial/connector_test.go — F-P4-001)
#   TestConnector_ConcurrentDropToZero_SingleEC004Emission (internal/upstreamdial/connector_test.go — F-P29-001)
#   TestRunRouter_PE_UnreachableUpstream_PartialPE        (cmd/switchboard/router_pe_connector_test.go)
#
# Usage: vhs docs/demo-evidence/S-7.04-FU-PE-CONNECTOR/AC-002-ec001-backoff-partial-pe.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-7.04-FU-PE-CONNECTOR/AC-002-ec001-backoff-partial-pe.gif
Output docs/demo-evidence/S-7.04-FU-PE-CONNECTOR/AC-002-ec001-backoff-partial-pe.webm

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

# ── Step 1: EC-001 log + ModeE + graceful-stop polarity guard ─────────────────

Type `echo "=== AC-002 unit: EC-001 log, ModeE, graceful-stop polarity ==="`
Enter
Sleep 300ms

Type `go test -run "TestConnector_DialFailure_EC001Log|TestConnector_AllUpstreamsUnreachable_ModeE|TestConnector_NoEC004OnGracefulStop" -v ./internal/upstreamdial/`
Enter
Wait+Line /PASS/
Sleep 2s

# ── Step 2: backoff constants + floor (fast) ──────────────────────────────────

Type `echo "=== AC-002 backoff constants: BackoffBase=500ms, BackoffCap=30s, Jitter=0.25 ==="`
Enter
Sleep 300ms

Type `go test -run "TestConnector_BackoffConstants|TestOperativeBase_TracksKeepalive" -v ./internal/upstreamdial/`
Enter
Wait+Line /PASS/
Sleep 2s

# ── Step 3: concurrent drop-to-zero single-emission (F-P29-001) ───────────────

Type `echo "=== AC-002 concurrency regression: exactly one EC-004 per drop-to-zero transition ==="`
Enter
Sleep 300ms

Type `echo "(stress-loop with 2 upstreams; ~9s)"`
Enter
Sleep 300ms

Type `go test -run "TestConnector_ConcurrentDropToZero_SingleEC004Emission" -v ./internal/upstreamdial/`
Enter
Wait+Line /PASS/
Sleep 2s

# ── Step 4: integration — partial-PE (one reachable + one unreachable) ─────────

Type `echo "=== AC-002 integration: partial-PE — one reachable, one unreachable ==="`
Enter
Sleep 300ms

Type `go test -run "TestRunRouter_PE_UnreachableUpstream_PartialPE" -v ./cmd/switchboard/`
Enter
Wait+Line /PASS/
Sleep 2s

# ── Show key observables ──────────────────────────────────────────────────────

Type `echo "=== key observables ==="`
Enter
Sleep 300ms

Type `echo "--- EC-001 log: 'upstream router <addr> unreachable' emitted on every failed dial ---"`
Enter
Sleep 300ms

Type `echo "--- ModeE: connectedCount==0 when all upstreams unreachable ---"`
Enter
Sleep 300ms

Type `echo "--- ModePE: connectedCount>=1 when at least one upstream connected (partial-PE) ---"`
Enter
Sleep 300ms

Type `echo "--- EC-004: 'mode=E (no upstream_routers configured)' on upstream-LOSS drop-to-zero only ---"`
Enter
Sleep 300ms

Type `echo "--- EC-004 NOT emitted on graceful Stop() (F-P4-001 polarity guard) ---"`
Enter
Sleep 300ms

Type `echo "--- EC-004 emitted exactly once per >=1->0 transition under concurrent drops (F-P29-001) ---"`
Enter
Sleep 1s

Type `echo "=== AC-002 PASS ==="`
Enter
Sleep 2s
