# AC-001 — PE graduation: outbound dial, session bootstrap, and set-equal reconciliation
#
# Demonstrates:
#   1. Connector constructed with initial upstream address list dials TCP connections
#   2. Each dial completes three-step bootstrap: net.Dial + outerassembler.Assemble + conn.Write
#   3. connector.Mode() == ModePE when connected-count >= 1
#   4. SIGHUP reload with same addresses in reversed order: no teardown, no new dials
#
# BC anchors: BC-2.09.001 PC-2/PC-3, EC-002; placement note Q1, Q6
#
# Test coverage:
#   TestConnector_DialSuccess_ModePE       (internal/upstreamdial/connector_test.go)
#   TestConnector_ReorderReuse_NoTeardown  (internal/upstreamdial/connector_test.go)
#   TestRunRouter_PE_DialAndConnect_UpstreamReachable          (cmd/switchboard/router_pe_connector_test.go)
#   TestRunRouter_PE_SetEqualReconciliation_NoTeardownOnReorder (cmd/switchboard/router_pe_connector_test.go)
#
# Declared divergences (D-1, D-2 per evidence-report.md):
#   - Bootstrap frame type uses halfchannel.FrameTypeData placeholder (FrameTypePEConnect deferred
#     to S-BL.PE-RECEIVE-LOOP per story v1.24/F-P26-001)
#   - Envelope fields (SrcAddr/DstAddr/SVTNID/FrameAuthKey) are zero-valued; full node-identity
#     derivation deferred to S-BL.PE-RECEIVE-LOOP as not-core
#
# Usage: vhs docs/demo-evidence/S-7.04-FU-PE-CONNECTOR/AC-001-pe-dial-connect.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-7.04-FU-PE-CONNECTOR/AC-001-pe-dial-connect.gif
Output docs/demo-evidence/S-7.04-FU-PE-CONNECTOR/AC-001-pe-dial-connect.webm

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

# ── Step 1: run the unit tests ────────────────────────────────────────────────

Type `echo "=== AC-001 unit: dial success + set-equal reconciliation ==="`
Enter
Sleep 300ms

Type `go test -run "TestConnector_DialSuccess_ModePE|TestConnector_ReorderReuse_NoTeardown" -v ./internal/upstreamdial/`
Enter
Wait+Line /PASS/
Sleep 2s

# ── Step 2: run the integration tests ────────────────────────────────────────

Type `echo "=== AC-001 integration: runRouter PE dial + reorder no-teardown ==="`
Enter
Sleep 300ms

Type `go test -run "TestRunRouter_PE_DialAndConnect_UpstreamReachable|TestRunRouter_PE_SetEqualReconciliation_NoTeardownOnReorder" -v ./cmd/switchboard/`
Enter
Wait+Line /PASS/
Sleep 2s

# ── Step 3: show expected Mode() transition ───────────────────────────────────

Type `echo "=== key observable: connector.Mode() == ModePE after dial ==="`
Enter
Sleep 500ms

Type `echo "--- TestConnector_DialSuccess_ModePE: TCP dial + Assemble + Write -> connectedCount++ -> Mode()==ModePE ---"`
Enter
Sleep 500ms

Type `echo "--- TestConnector_ReorderReuse_NoTeardown: reload same addrs reversed -> no teardown, Mode() still ModePE ---"`
Enter
Sleep 1s

# ── Cleanup ───────────────────────────────────────────────────────────────────

Type `echo "=== AC-001 PASS ==="`
Enter
Sleep 2s
