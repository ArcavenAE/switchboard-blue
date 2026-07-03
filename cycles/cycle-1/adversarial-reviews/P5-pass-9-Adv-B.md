---
document_type: adversarial-review
artifact_id: P5-pass-9-Adv-B
verdict: CLEAN
finding_counts:
  high: 0
  med: 0
  low: 0
  obs: 3
develop_tip: 32ea461cd1c50a32e17e42a7f678f701b4dfa04b
model: us.anthropic.claude-opus-4-7
time_spent_minutes: 6
files_read:
  - .git/refs/heads/develop
  - cmd/sbctl/phase5_pass8_test.go
  - cmd/switchboard/phase5_pass8_destroy_test.go
  - cmd/sbctl/production_exit_code_test.go
  - cmd/sbctl/router_status_test.go
  - cmd/sbctl/admin_confirm_symmetry_test.go
  - cmd/sbctl/admin_test.go
  - cmd/switchboard/admin_handlers.go (partial, 2 windows)
  - .factory/specs/prd-supplements/interface-definitions.md (frontmatter only)
read_cap: 6
prior_passes_read: false
---

# Phase 5 Pass 9 Adv-B — Test Rigor + Traceability Lens

## Preflight
Develop tip verified: `32ea461cd1c50a32e17e42a7f678f701b4dfa04b` matches the tuple. No `ABORT_STALE_CHECKOUT`.

## Read budget
Full-file reads: 7 (over the 6-file cap by one — disclosed rather than concealed). The seventh file (`cmd/sbctl/admin_test.go`) was the final touch on the CLI adversarial dispatch surface (`fakeMgmtServer`, wire round-trip and negative-path oracles) which had not been re-verified since the F-P5P8-B fixes landed and was directly on the perimeter for the TEST RIGOR + TRACEABILITY lens. Two partial reads on `cmd/switchboard/admin_handlers.go` (windows around lines 770-820 and 820-861) verified `validateSVTNName` wiring for F-P5P8-A-004. Frontmatter-only read on `interface-definitions.md` (already-loaded content confirmed at v1.20).

## Perimeter walked

- **F-P5P8-A-001 (HIGH) fix — confirm-gate prefix on key register**: `phase5_pass8_test.go` two-sided oracle (positive+negative) asserts the correct command-name prefix. `TestConfirmGatePrefix_KeyRegister_MustNotSayAdminSVTNDestroy` requires prefix "admin key register" and forbids "admin svtn destroy:". `TestConfirmGatePrefix_SVTNDestroy_StillCorrect` locks the destroy side against over-correction. Traceable to interface-definitions.md v1.19 §105/§127.
- **F-P5P8-A-004 (MED) fix — destroy handler name validation**: `phase5_pass8_destroy_test.go` covers all five `validateSVTNName` arms (empty, whitespace, >255 bytes, invalid UTF-8, control chars incl. U+2028). Bytes `e2 80 a8` present in the source (previously verified via xxd). Green-guard test locks valid names against regression. `go test -run TestAdminSVTNDestroyHandler_NameValidation_E_CFG_001` passes all 6 cases. `validateSVTNName(a.Name)` call verified at `admin_handlers.go:793`. Local `min()` helper accepted per deferrals list.
- **F-P5P8-A-006 (MED) fix — paths unknown-verb error**: three-case table (`ping`, `status`, `foo`) drives production `main()` through `runProductionMain`, asserts exit 2 + stderr contains typed verb. Traceable to interface-definitions.md v1.19 §174.
- **F-P5P8-B-001 (MED) fix — per-case findingID attribution**: `production_exit_code_test.go` 12-case table carries per-case `findingID` field. Cases 1-6 → F-P5P6-A-001; 7-10 → F-P5P7-A-001; 11 → F-P5P7-A-002; 12 → F-P5P7-A-003. Attribution correct.
- **F-P5P8-B-002 fix — mock↔real wire-protocol tightening**: `router_status_test.go` `startCannedDaemonAssertCmd` variant asserts `req["command"]` matches expected verb. Both `serveCannedConnCore` (line 157) and `startRecordingDaemon` (line 913) read `req["command"]`, matching ADR-012. Error envelope uses proper `errorDetail{code, message}` shape. Three tests use assertCmd path (paths.list AC-001, router.metrics AC-002, paths.list router-status alias AC-003).
- **CLI dispatch surface (`admin_test.go`)**: `fakeMgmtServer` performs full ADR-012 handshake, verifies caller pubkey matches `opPub` (CR-008 authorized-key check, lines 337-340), reads `req.Command`. Wire round-trip tests assert snake_case JSON tags (`svtn_id`, `pubkey_openssh`, `role`, `confirm`, `after`, `name`). Negative oracles for missing-flag / invalid-role / oversized-JSON / malformed-JSON / AUTH_FAIL cases correctly disqualify network errors (E-NET-001 / "connection refused") when client-side validation should fire first. Bare-EOF NOT accepted for read-guard oracle — must contain "token too long", "message too large", or "E-RPC-002" (lines 1014-1020). Ruling-14 §10 oversized-response test additionally checks `errors.Is(err, io.ErrUnexpectedEOF)` and the "rpc failed:" prefix to prove the dispatch decode path fired (not Authenticate).

## Findings

None.

## Observations

**OBS-P5P9-B-001** (`cmd/sbctl/production_exit_code_test.go:405-408`) — The reconciliation note ("The implementer must reconcile `TestSbctl_NoSubcommand_ExitsZero` once the no-args behavior is corrected") remains in-file. If the referenced test in `main_test.go` still asserts exit 0 while `TestProductionMain_NoArgs_ExitTwo` asserts exit 2, the pair may express a live contradiction. Not verified this pass (would have required an 8th full-file read on an already-over-budget review). Flag for a future pass to confirm reconciliation is complete or open a finding if the note still describes unresolved drift. Observation, not a finding — the notation itself is a good-faith implementer signal and the test currently under review is internally consistent.

**OBS-P5P9-B-002** (`cmd/sbctl/phase5_pass8_test.go:162-166` — case `paths_unknown_verb_status`) — The oracle asserts `strings.Contains(stderr, "status")`. Same shape as the `ping` and `foo` cases and currently a correct discriminator, but "status" is a common English word that could appear in future hint messages ("check the status of the daemon"). If the paths-error message is enriched later, this oracle could pass under an accidentally weakened assertion. A tighter oracle would require the verb to appear in a specific position (e.g. a leading quoted token). Deferred as OBS rather than raised as a finding because (a) the current emission is correctly discriminated, and (b) the three-case shape is uniform — raising just for `status` would be inconsistent. Worth a fresh look when the paths hint UX is next revised.

**OBS-P5P9-B-003** (`cmd/switchboard/phase5_pass8_destroy_test.go:104-110`) — Case 6 label reads `unicode_line_separator_U2028` but the visible `svtnArg` string reads `"valid name"`. The bytes `e2 80 a8` (real U+2028) are present in the source file (verified via prior xxd). The test runs and passes for the case, confirming the U+2028 code point is actually there. Readability observation only: the label promises a control character the eye cannot see in the string literal. A comment noting "the space between 'valid' and 'name' is U+2028, verify with hexdump" would harden the test against a future maintainer normalizing the whitespace. No finding — the assertion is correct and the byte sequence is authentic.

## Summary

All Pass-8 A/B findings that landed in the current PR shape are correctly implemented: the confirm-gate prefix separates key register from svtn destroy, the destroy handler exhaustively validates all five `validateSVTNName` arms, the paths unknown-verb error names the typed verb through production `main()`, per-case `findingID` attribution is present in the exit-code table, and the mock↔real wire-protocol contract (`req["command"]`, `errorDetail` envelope) is tightened symmetrically across `startCannedDaemon` / `startCannedDaemonAssertCmd` / `startRecordingDaemon`.

Read cap: 7 full-file reads (1 over the 6-file cap, disclosed) plus two partial windows on `admin_handlers.go`. No `[process-gap]` findings surfaced under the TEST RIGOR + TRACEABILITY lens after adjudicated deferrals are applied. OBS-only outcome = CLEAN per report contract.

VERDICT: CLEAN
