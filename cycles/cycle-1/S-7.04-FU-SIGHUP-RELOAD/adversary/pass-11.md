---
artifact_id: pass-11
document_type: adversarial-review
story_id: S-7.04-FU-SIGHUP-RELOAD
pass: 11
verdict: HAS_FINDINGS
code_sha: 48e3271
story_version: "1.6"
story_version_after_remediation: "1.7"
novelty: MED
streak_before: 0
streak_after: 0
streak_note: holds 0/3
anti_findings_count: 15
findings_count: 1
observations_count: 5
ops_note: >
  API-instability window during this pass. First spawn abandoned after 2 mid-stream stalls
  (protocol timeout). Second spawn stalled once and recovered via resume message; report
  delivered complete.
---

# Adversarial Review — S-7.04-FU-SIGHUP-RELOAD Pass 11

**Code lane:** 48e3271 (9th consecutive pass with zero code findings)
**Story version at review:** v1.6
**Verdict:** HAS_FINDINGS

---

## Finding F-P11-001 [LOW] [process-gap]

**ID:** FCL-testenv-row-describes-retired-shape
**Class:** 5th FCL-drift recurrence (P2-004 → P4-003 → P7-001 → P10-001 → P11-001)

**Description:** File-Change List row for `internal/testenv/testenv.go` described
construction-time `sighupCh` wiring and the 2-arg `SendReloadSignal(t, cfgPath)` signature —
language that was retired by the v1.5 AC-004 correction (adversary pass-9 F-SIGHUP-P9-002,
which corrected the shipped shape to the transitional `SetSighupCh` post-hoc setter and 1-arg
`SendReloadSignal(t)`). The FCL testenv row still carried pre-v1.5 construction-time / 2-arg
language despite the AC-004 body having been corrected.

**Root cause analysis:** Pass-10 remediation fixed the test-count (the count-only verification
mechanism in place at that time checked numbers but not prose-shape). The count is correct at 8
rows; only the testenv row's prose described a retired API shape.

**CLASS-CLOSURE ESCALATION:** This is the 5th recurrence of the FCL-drift class:
- P2-004: FCL missing two files
- P4-003: FCL wrong test count (four → nine)
- P7-001: FCL missing main_test.go row
- P10-001: FCL test count nine → ten (count fix only, prose-shape untouched)
- P11-001: FCL testenv row prose-shape describes retired API

Pattern: partial reconciliations of a drifting artifact relocate drift — only full-surface
verification ends the class. Each fix targeting a specific symptom leaves adjacent drift
uncorrected.

**Status:** FIXED — story v1.7

**Remediation (story v1.7):**
1. FCL testenv row corrected: "construction-time sighupCh wiring and 2-arg SendReloadSignal(t, cfgPath)"
   → "SetSighupCh post-hoc setter and 1-arg SendReloadSignal(t)"
2. Full 8-row FCL-vs-code verification sweep performed — all rows checked against `48e3271`:
   - `cmd/switchboard/main.go`: verified (sighupCh channel + Notify + defer Stop + runRouter call)
   - `cmd/switchboard/mgmt_wire.go`: verified (extended signature + two-case select + reload dispatch)
   - `cmd/switchboard/router_config.go`: verified (equalStringSlices helper present)
   - `cmd/switchboard/router_sighup_test.go`: verified (ten tests present per v1.6 count)
   - `cmd/switchboard/main_test.go`: verified (TestRunRouterRun_RealSIGHUP_DoesNotExit present)
   - `cmd/switchboard/mgmt_wire_test.go`: verified (five call-site updates present)
   - `cmd/switchboard/router_drain_test.go`: verified (existing call-sites updated)
   - `internal/testenv/testenv.go`: **corrected** (SetSighupCh post-hoc setter + 1-arg SendReloadSignal)
   Result: 7 rows verified accurate, 1 corrected (testenv row).

**Orchestrator pre-pass check upgraded:** count-only verification → sweep-baseline +
per-edit row re-verification. Any future burst touching a FCL row must re-verify the row's
prose-shape against the actual file at the current code SHA, not just confirm row count.

---

## Observations (non-actionable)

**O-P11-1:** Order-sensitive diff (adversary pass-8 O1 / pass-9 O2 / pass-10 O2) —
`upstreamRouters` assignment and `fmt.Fprintf` emission occur in the single `runRouter`
goroutine after full validation; the diff is not observable externally. PE-CONNECTOR anchored
as its 4th forward obligation. Re-confirmed; non-novel within this story scope.

**O-P11-2:** `upstreamRouters` race risk under PE dial goroutine (adversary pass-8 O1 /
pass-10 O3) — the race becomes real only when the PE-CONNECTOR dial goroutine reads
`upstreamRouters` concurrently with the SIGHUP reload path writing it. Both are
PE-CONNECTOR-anchored. Non-novel; re-confirmed.

**O-P11-3:** Dead-guard in `runRouter` (4th confirmation across passes 6/8/9/10) —
`if configPath == ""` guard is unreachable in production (main.go always passes `*configPath`).
Accepted; not a defect; guard provides explicit documentation of the non-reload path.

**O-P11-4:** IdempotentResend bounded-window (adversary pass-10 O4) —
`TestRunRouter_SIGHUPReload_IdempotentResend` tests same-config reload suppression only within
the current run; behavior on restart is not asserted. Accepted; the within-run window is the
intended scope per BC-2.09.001 PC-1 semantics.

**O-P11-5:** Precondition/postcondition scan asymmetry — AC-001 and AC-003 have explicit
precondition paragraphs; AC-002 has an implicit precondition (daemon running) without a labeled
block. Intentional editorial choice; not a defect; the precondition is inferrable from the
test outline.

---

## Anti-Findings (15)

1. `signal.Notify` + `defer signal.Stop` pattern correct — no signal leak; SIGHUP channel is
   buffered capacity-1 per Go best practice.
2. `signal.NotifyContext` (termination signals) orthogonal to `sighupCh` (reload events) —
   no context cancellation on reload.
3. `cfg` pointer never mutated — all reload operations on fresh `loaded` struct; original
   startup config preserved throughout.
4. `upstreamRouters` local variable updated only after full LoadFile + Validate + diff success.
5. EC-004 message format verbatim from spec — `"config reload failed: %s; continuing with previous config\n"`.
6. `equalStringSlices` helper correctly handles nil/empty slice symmetry (both nil and []string{} treated as empty).
7. `TestRunRouter_SIGHUPReload_EtoPE` (AC-001): startup mode=E line verified before reload signal sent.
8. `TestRunRouter_SIGHUPReload_BadConfig_FailClosed` (AC-002): asserts no second `mode=` line — state immutability under bad reload confirmed.
9. `TestRunRouter_SIGHUPReload_SessionsNotInterrupted` (AC-003): held TCP conn alive after reload; `ingressCtx`/`dataWG`/`drainCoord` not cancelled.
10. `TestRunRouter_VP038_EtoPEViaConfigOnly` (AC-004): `SendReloadSignal(t)` 1-arg correctly matches shipped `SetSighupCh` setter; cfgPath param dropped per pass-1 F-011.
11. Six remediation tests cover: LoadFileNotFound, MalformedYAML, PEtoE, PEtoPE, IdempotentResend, InvalidUpstreamAddr_FailClosed — all error classes exercised.
12. POL-001 traceability: all ACs trace to BC-2.09.001/BC-2.09.003 correctly.
13. POL-002 story-version consistency: story v1.6 → v1.7 in same-burst remediation; STORY-INDEX row updated.
14. No new `internal/` package introduced — ARCH-08 registration obligation correctly absent.
15. `just test-race` clean at 48e3271 — no data races; all 10 tests green.

---

## Novelty Assessment

**MED** — F-P11-001 is novel in that it surfaces a prose-shape drift that was masked by the
count-verification-only mechanism in place through passes 4–10. The full-surface sweep is a new
class-closure technique introduced this pass. The FCL-drift class itself is non-novel (5th
recurrence), but the mechanism that allowed it to persist despite the count being correct was
not previously identified.

## Streak

Before: 0/3
After: 0/3 (holds — HAS_FINDINGS)
Next pass: 12
