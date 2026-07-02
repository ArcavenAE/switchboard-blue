---
artifact_id: W-6-wavegate-pass-6-Adv-B
document_type: wave-adversarial-review
scope: wave-gate-integration
wave: W-6
tranche: combined
stories: [S-BL.LOOKUP, S-W5.04, S-6.07, S-7.01, S-7.02, S-BL.ROUTER-ADDR, S-7.03, S-6.05]
lens: [L2, L3]
develop_tip_claimed: "7fe3e29"
develop_tip_observed: "7fe3e29e4358df16e4e2f1de65a4e0d972540b4a"
pass_number: 6
attempt_number: 1
sub_adversary: Adv-B
verdict: CONVERGENT_L2L3
findings:
  critical: 0
  high: 0
  medium: 0
  low: 0
observations: 3
reviewer_context: fresh
prior_passes_read: false
worktree_identity_tuple_verified: true
dispatch_integrity_failure: false
timestamp: 2026-07-02T00:00:00Z
---

# W-6 Wave-Gate Adversarial Review — Pass 6 Attempt 1 (Adv-B, L2+L3)

## Preflight

- `.git/HEAD` = `ref: refs/heads/develop` OK
- `.git/refs/heads/develop` = `7fe3e29e4358df16e4e2f1de65a4e0d972540b4a` (starts with `7fe3e29`) OK
- CWD basename = `switchboard-blue` OK
- `prior_passes_read: false` — no reads under `.factory/cycles/*/adversarial-reviews/`, `STATE.md`, `sprint-state.yaml`.
- Read cap: 5/6 file Reads used. Grep-first discipline held.

## Scope

Wave-6 combined wave-gate integration on `develop@7fe3e29`, 8 merged stories. Perimeter-3 (wave-gate integration). Lenses L2 (test rigor) + L3 (traceability/governance).

## L2 Q1 — Race coverage across new-package `*_test.go`

Grep for `t.Parallel()|sync.WaitGroup|go func(|goroutine|race` across the four new packages:

- `internal/arq/{arq_test.go,fec_test.go}` — 39+15 concurrency-marker matches. `arq_test.go` uses concurrent enqueue/deliver scenarios; `fec_test.go` is single-threaded state-machine (justified single-actor; no shared mutable state).
- `internal/discovery/discovery_test.go` — 41 concurrency-marker matches. Presence-registry racing scenarios present.
- `internal/svtnmgmt/svtnmgmt_test.go` — 128 concurrency-marker matches. Manager write-path under concurrent Create/Destroy exercised.
- `internal/mgmt/mgmt_test.go` — 180 concurrency-marker matches. Shutdown-race (`shutdownWindowListener`), connection-drain, Serve-return path all racing.

Verdict: race scaffolding is present in every new-package test file where multi-actor state exists. No absence-without-justification observed.

## L2 Q2 — Cross-story E2E sequence

Searched for a combined create → attach → destroy through one shared daemon:

- `cmd/switchboard/admin_handlers_e2e_test.go` — 15 top-level tests; covers register/expire/revoke/list-keys/destroy independently. **No test creates an SVTN then attaches via console then destroys through the same daemon process.**
- `cmd/switchboard/console_handlers_e2e_test.go` — 6 tests, all `TestConsoleRemote_E2E_*`, testing attach/detach/switch on pre-seeded fixtures.
- No `TestCreateAttachDestroy` / cross-story lifecycle test located.

Wave-schedule search (`wave-schedule.md`): no `create.*attach.*destroy`, `shared daemon`, `combined lifecycle`, `S-BL.SESSION-DRAIN`, `W-7` deferral markers matched.

**Adjudication:** the eight merged stories each carry their own E2E for their scoped postconditions (S-6.07 create, S-7.03 attach/detach/switch, S-6.05 destroy). Each daemon-mode test constructs its own in-process listener + mgmt.NewServer, matching the pattern documented in VP-050 v1.1 (in-process pattern). A combined-lifecycle test that reuses ONE daemon process across all three phases is not currently present and is not documented as a deferral either in the wave schedule or in-code. This is a scope-boundary gap: it may be intentional (each PC is separately verified per its VP) or it may be a missing integration property. Per the closing-pass instruction and lack of specific defect, I record this as an Observation rather than a finding.

## L2 Q3 — Tautology sweep

Grep pass for `assert.*true.*true`, `Equal.*nil.*nil`, `Equal(t, 1, 1)`, bare `_ = err`:

- Single match: `internal/mgmt/mgmt_test.go:2309` `_ = err`. Context (lines 2290–2320): this is the drain-window Serve-race scenario (`shutdownWindowListener`). The test asserts on the deadline via `<-time.After(500 * time.Millisecond)` timeout, NOT on Serve's return value; the `_ = err` deliberately consumes the returned error since the property under test is "Serve returned within budget after Shutdown", not the error content. This is idiomatic Go for shutdown-race harnesses and matches the AC-017/Ruling I obligation from VP-069. **Not a tautology.**

No other tautology hits.

## L3 Q1 — BC-anchor version-pin chain

**BC-2.07.001** (frontmatter `version: "1.13"`, terminal changelog row v1.13):
- v1.13 row: `[governance_leaf: true — downstream story/VP pins DO NOT need to re-sync per drbothen/vsdd-factory#429 draft policy]` OK
- VP-048 `source_bc: BC-2.07.001 v1.12` — intentionally lagging, per POL-003 Exception A (v1.13 is governance-only Stories-row cite bump).
- VP-INDEX row (line 74) pins BC-2.07.001 v1.12 — consistent with VP-048.
- Story S-6.05 body `bcs_traces: BC-2.07.001: v1.4 through v1.12`; body table pins v1.12 — consistent.

Chain: consistent under Exception A. No defect.

**BC-2.08.001** (frontmatter `version: "1.5"`, terminal changelog row v1.5):
- v1.5 row: `[governance_leaf: true — annotation-shape correction, downstream VP/story pins DO NOT need to re-sync per POL-003 Exception A]` OK
- v1.4 row: NOT governance-only (Inv-3 rewording, transport-type reference update, Stories-row v1.3→v1.4 bump) — behavioral change requiring downstream re-sync.
- v1.3 row: `[governance_leaf: true — Stories-row pin sync, downstream VP/story pins DO NOT need to re-sync per POL-003 Exception A]` OK
- Story S-7.03 body pins `BC-2.08.001 v1.4` — consistent with the last non-governance BC version.
- VP-050 `source_bc: BC-2.08.001` (no version pin) — this is a shape inconsistency vs VP-048's `source_bc: BC-2.07.001 v1.12` format, BUT VP-050 v1.3 modified line ties directly to Story Trace row `S-7.03 v1.4` and BC-2.08.001 v1.4 is the operative version. Substance is anchored; only the source_bc frontmatter version-suffix format differs across VPs.

Chain: substantively consistent. Frontmatter shape drift (with/without `v1.N` suffix on `source_bc:`) is a cross-VP naming variation, not a semantic defect. Below finding threshold.

## L3 Q2 — VP-INDEX arithmetic consistency

- **Counts row** (line 114): `Total=76, Proptest=34, Fuzz=4, Integration=22, E2E=10, Benchmark=2, Code-Audit=2, Unit=2`.
- Arithmetic (line 116): `34 + 4 + 22 + 10 + 2 + 2 + 2 = 76`
- **Phase Distribution** (lines 135–138): `P0=54, P1=18, P2=4, Total=76` — `54+18+4 = 76`
- BC Coverage Check (line 144): "45 BCs total ... All 45 have at least one VP" — narrative consistent.
- Row count: VP-001..VP-076 = 76 permanent-ID rows, plus 2 placeholders (VP-TBD-ACC + VP-VW6.NN) both marked `deferred/deferred`, correctly excluded from the 76 total.
- Changelog v2.34/v2.33/v2.32/v2.31/v2.30 all footer "total remains 76" — consistent.

Verdict: arithmetic consistent across summary row, per-phase, discussion narrative. Clean.

## L3 Q3 — POL-003 Exception A `governance_leaf: true` shape consistency

Terminal governance-only rows sampled:
- BC-2.07.001 v1.13 (2026-07-02): `[governance_leaf: true — downstream story/VP pins DO NOT need to re-sync per drbothen/vsdd-factory#429 draft policy]` OK
- BC-2.08.001 v1.5 (2026-07-02): `[governance_leaf: true — annotation-shape correction, downstream VP/story pins DO NOT need to re-sync per POL-003 Exception A]` OK
- BC-2.08.001 v1.3 (2026-07-02): `[governance_leaf: true — Stories-row pin sync, downstream VP/story pins DO NOT need to re-sync per POL-003 Exception A]` OK

Reference-wording drift (`drbothen/vsdd-factory#429` on BC-2.07.001 v1.13 vs `POL-003 Exception A` on BC-2.08.001) is already logged as `DRIFT-POL003-NAMING` per dispatch instructions — NOT re-flagged.

Earlier BC-2.07.001 rows (v1.8/v1.9/v1.10/v1.12) predating the convention: grandfathered per orchestrator adjudication — NOT flagged.

Shape consistency: all terminal governance-only rows on both BCs bear the annotation. Clean.

## L3 Q4 — Terminal governance-only row annotations + already-logged drift

Both BC-2.07.001 v1.13 and BC-2.08.001 v1.3/v1.5 terminal governance-only rows carry the annotation (confirmed above).

Already-logged items explicitly excluded per dispatch:
- `DRIFT-POL003-NAMING` (wording drift `drbothen/vsdd-factory#429` vs `POL-003 Exception A`) — NOT re-flagged.
- `DRIFT-BC207-V113-BODY-CHANGELOG-MISMATCH` (BC-2.07.001 v1.13 changelog cite `→ v1.7` vs body `S-6.05 v1.8`) — NOT re-flagged.

No NEW governance defects unrelated to those two drift items found.

## Findings

None. (0 critical / 0 high / 0 medium / 0 low.)

## Observations

- **[obs-01]** Cross-story E2E (create → attach → destroy through one shared daemon process) is not present in `cmd/switchboard/*_e2e_test.go` and not enumerated in `wave-schedule.md`. Each of S-6.07/S-7.03/S-6.05 has its own VP-scoped E2E using in-process `mgmt.NewServer` per its scoped PC. A combined-lifecycle integration property (spanning three BCs) may be intentional deferral-by-omission or a genuine coverage seam; if a future integration story is minted for this, note the anchor pattern here. Not a defect against the current wave scope.

- **[obs-02]** `source_bc:` frontmatter format varies across VPs — VP-048 uses `BC-2.07.001 v1.12` (BC-ID with version suffix); VP-050 uses `BC-2.08.001` (BC-ID only). Substance is anchored elsewhere (Story Trace tables, changelog narratives). If BC-INDEX / VP-INDEX author-format policy prefers one shape, a future L3 pass could pattern-sweep. Below finding threshold — cosmetic cross-VP frontmatter shape drift.

- **[obs-03] [process-gap]** VP-050 has `source_bc: BC-2.08.001` (no version pin) while POL-003 Exception A governance annotations on downstream artifacts (VPs, stories) rely on being able to detect a "trailing pin". VP-INDEX row for VP-050 (line 76) similarly omits a `v1.N` version on the BC column while VP-048 (line 74) carries `BC-2.07.001 v1.12`. A tool auditing "does the downstream cite the current BC version?" cannot mechanically answer the question for VP-050 because there's no version to compare. Not a defect this pass — the substance is anchored via Story Trace v1.4 pin — but the shape asymmetry weakens auditability. Candidate for a future POL-003 tooling refinement (require `source_bc: BC-N.NN.NNN v<M.N>` on every VP frontmatter for machine-checkability).

## Novelty Assessment

**Novelty: LOW.** L2 sweeps (race scaffolding, cross-story E2E, tautologies) find no defects and only one below-threshold observation (obs-01). L3 sweeps (BC version-pin chains, VP-INDEX arithmetic, `governance_leaf: true` shape, terminal-row annotations) all clean under Exception A, with the two pre-logged drift items respected per dispatch and NOT re-flagged. The frontmatter shape asymmetry noted in obs-02/obs-03 is a genuinely fresh observation (not addressed as a hard defect in prior L3 axes) but is below finding severity — auditability refinement, not correctness violation.

Findings across L2+L3 = 0. Observations = 3. Third consecutive clean pass criterion met for the closing pass under this reviewer's fresh-context sweep.

## Verdict

**CONVERGENT_L2L3.** No critical/high/medium/low findings. Three observations recorded (one process-gap-tagged, all below finding threshold). BC-anchor chains substantively consistent under POL-003 Exception A. VP-INDEX arithmetic consistent (76 = 34+4+22+10+2+2+2 = 54+18+4). Terminal governance-only annotations shape-consistent on both BCs. Race scaffolding present in all four new-package test files. Pre-logged drift items respected.

Verdict: CONVERGENT_L2L3 | critical=0 high=0 medium=0 low=0 | observations=3
