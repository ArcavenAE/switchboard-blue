---
artifact_id: W-6-wavegate-pass-4-Adv-B
document_type: wave-adversarial-review
scope: wave-gate-integration
wave: W-6
tranche: combined
stories: [S-BL.LOOKUP, S-W5.04, S-6.07, S-7.01, S-7.02, S-BL.ROUTER-ADDR, S-7.03, S-6.05]
lens: [L2, L3]
develop_tip_claimed: "7fe3e29"
develop_tip_observed: "7fe3e29e4358df16e4e2f1de65a4e0d972540b4a"
pass_number: 4
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

# Wave-6 Combined Wave-Gate Adversarial Review — Pass 4 attempt 1 — Adv-B (L2/L3)

## Preflight

- `.git/HEAD` → `ref: refs/heads/develop` ✓
- `.git/refs/heads/develop` → `7fe3e29e4358df16e4e2f1de65a4e0d972540b4a` — starts with `7fe3e29` ✓
- cwd basename `switchboard-blue` ✓
- Prior-pass artifacts: NOT read (grep-metadata surfaced filenames only via governance_leaf term-search; contents not opened)
- Read budget used: 5 Reads (BC-2.07.001, BC-2.08.001 twice, policies.yaml, wave-schedule.md fragments). Under 6.

## Scope

L2 (test rigor) + L3 (traceability/governance) integration review of Wave-6 combined wave-gate on develop@7fe3e29e4358df16e4e2f1de65a4e0d972540b4a — 8 stories merged (S-BL.LOOKUP, S-W5.04, S-6.07, S-7.01, S-7.02, S-BL.ROUTER-ADDR, S-7.03, S-6.05).

## L2 Q1 — Race-coverage across new-package test files: PASS

Grep of `t.Parallel|go func\(|sync\.` across the new Wave-6 packages returned dense concurrency scaffolding:

- `internal/arq/fec_test.go`: 16 occurrences (`fec_test.go:110,151,168,193,206,241,256,285,307,353,449,465,513,574,585` t.Parallel usage; explicit sequential-subtest comment at 579-585 documenting count_verify ordering constraint — non-parallel absence justified)
- `internal/arq/arq_test.go`: 32 occurrences (t.Parallel at 115,151,182,236, …)
- `internal/discovery/discovery_test.go`: 40 occurrences (t.Parallel at 97,136,188,245,301,373,433,476,494,545,589,625,660 + sync.Mutex + goroutines at 195,214,251,271,305,326,390 — real concurrency exercise, not just annotation)
- `internal/svtnmgmt/svtnmgmt_test.go`: 57 occurrences (t.Parallel across 80,116,143,166,211,244,258,307,326,372,416,438,459,477,488,522,552,565,598 …); svtnmgmt.go itself uses `sync.RWMutex` at line 121, confirmed race-tested
- `internal/mgmt/mgmt_test.go`: 61 occurrences with concurrent-scenario goroutines at 144,645,869,964,1059,1158,1294 and explicit `// NOT t.Parallel(): measures goroutine counts for leak detection` justifications at 854, 1237

New-package concurrency scaffolding is present and justified where absent. PASS.

## L2 Q2 — Cross-story E2E sequence (create → attach → destroy): PASS with deferral

Grepped `cmd/switchboard/` and `cmd/sbctl/` for any test wiring create+attach+destroy end-to-end through one server: no such combined test exists. Deferral is documented:

- `.factory/cycles/cycle-1/wave-schedule.md:187-193` explicitly states: "Deferred cross-story behavior (out-of-scope for W-6.C). Destroy-with-active-console-attach cascade (SVTN destruction propagating a detach to active console attaches on sessions inside the destroyed SVTN) is deferred to `S-BL.SESSION-DRAIN` per S-6.05 v1.5 AC-002 out-of-scope note and the in-code deferral marker at `internal/svtnmgmt/svtnmgmt.go:770-771`."
- Wave-schedule further records "Wave-6 holdout HS-006 does not exercise this cascade. Manual-eval-only for W-6.C; full boundary coverage will land with `S-BL.SESSION-DRAIN`."

Deferral is well-anchored (schedule + in-code marker + backlog story). PASS.

## L2 Q3 — Tautology sweep: PASS

- `assert.Equal\(t, ([a-zA-Z_.]+), \1\)` across `internal/svtnmgmt` and `internal/discovery`: 0 matches.
- `fmt.Sprintf(…) == fmt.Sprintf` across all `*_test.go`: 0 matches.
- Repository does not appear to use `testify/assert` for these packages (grep of `Equal(t,` returned nothing in svtnmgmt/discovery), consistent with `go.md` §Testing stdlib-only guidance.

No tautological assertions detected. PASS.

## L3 Q1 — BC-anchor version-pin chain for S-6.05 (BC-2.07.001) and S-7.03 (BC-2.08.001): PASS

STORY-INDEX (`.factory/stories/STORY-INDEX.md`):
- Line 73 — S-6.05 row lists BC-2.07.001 (no version-pin embedded in the row itself; index does not require inline BC version).
- Line 77 — S-7.03 row lists BC-2.08.001.

BC-2.07.001 frontmatter+changelog (`.factory/specs/behavioral-contracts/ss-07/BC-2.07.001.md`):
- frontmatter version: `"1.13"` (line 5), timestamp 2026-06-30
- Latest changelog row (line 218): v1.13, 2026-07-02, "F-P4L3-MED-2 (POL-002): Traceability Stories row cite S-6.05 v1.5 → v1.7 (this fix-burst bumps story to v1.7). Governance-only. [governance_leaf: true — downstream story/VP pins DO NOT need to re-sync per drbothen/vsdd-factory#429 draft policy]"

BC-2.08.001 frontmatter+changelog (`.factory/specs/behavioral-contracts/ss-08/BC-2.08.001.md`):
- frontmatter version: `"1.5"` (line 5), timestamp 2026-07-02
- Latest changelog row (line 141): v1.5, 2026-07-02, "F1 remediation from W-6 wave-gate Pass-3 Adv-B: retro-annotate v1.3 changelog row with `governance_leaf: true` per POL-003 Exception A audit-tool compatibility. Shape now matches BC-2.07.001 v1.13. No behavioral changes. [governance_leaf: true — annotation-shape correction, downstream VP/story pins DO NOT need to re-sync per POL-003 Exception A]"

VP-048 (`.factory/specs/verification-properties/VP-INDEX.md:74`): "source_bc pin synced to BC-2.07.001 v1.12" (frontmatter draft). Note: BC-2.07.001 is now at v1.13. This is intentional per POL-003 Exception A candidate policy — governance-only bumps (v1.12 was Stories-row narrative sync per policies.yaml candidate POL-003 wording, v1.13 was Traceability Stories row cite) do not require downstream VP re-pin. Documented in the BC-2.07.001 v1.13 changelog row annotation itself.

VP-050 (`.factory/specs/verification-properties/VP-INDEX.md:76`): source_bc pin BC-2.08.001 (no explicit sub-version in row). VP-INDEX changelog v2.34 (line 150) documents propagation of BC-2.08.001 v1.4-era transport-clause update.

Version chain is coherent. VP source_bc pins deliberately lag the BC governance-only tip per candidate POL-003 Exception A — matches the annotation on the terminal changelog row. PASS.

## L3 Q2 — VP-INDEX total count: PASS

VP-INDEX (`.factory/specs/verification-properties/VP-INDEX.md`):
- Line 114 summary row: `76 | 34 | 4 | 22 | 10 | 2 | 2 | 2`
- Line 116: "Arithmetic check: 34 + 4 + 22 + 10 + 2 + 2 + 2 = 76. Consistent."
- Line 138: `**Total** | **76**`
- Line 140: "Phase recounted 2026-06-30: … P0 = 54. P1 = 18. P2 = 4. Total = 76."

Sprint-state target was `76+`. Actual = 76 (meets baseline). No Wave-6-anchored gap.

## L3 Q3 — POL-003 Exception A annotation shape consistency: PASS with observation

POL-003 status: candidate policy pending user ratification (`.factory/policies.yaml:58-65`). The `governance_leaf: true` annotation shape has been adopted informally across BC changelog rows that are pure Stories-row / VP-Story-Trace pin syncs.

BC-2.08.001 changelog (5 rows: v1.5, v1.4, v1.3, v1.2, v1.1):
- v1.5 governance-only (retro-annotation of v1.3): HAS `[governance_leaf: true …]` ✓
- v1.4 behavioral (Inv-3 wording rewrite): correctly NO annotation ✓
- v1.3 governance-only (Stories row pin sync): HAS annotation ✓
- v1.2 behavioral (RULING-W6TB-C retraction): correctly NO annotation ✓
- v1.1 initial draft: correctly NO annotation ✓

BC-2.07.001 changelog (13 rows). Terminal row (v1.13) is governance-only and HAS the annotation. Earlier governance-only rows lack it — see observation O-1.

## Findings

None. All L2/L3 gates PASS.

## Observations

### O-1 (LOW, [process-gap], pending intent verification)

**Annotation-shape drift within BC-2.07.001 governance-only rows.** Pass-3 Adv-B triggered a retro-annotation on BC-2.08.001 v1.3 (yielding v1.5) to bring its shape into parity with BC-2.07.001 v1.13. However, BC-2.07.001 itself contains earlier governance-only rows that predate the annotation convention and were NOT retro-annotated in the same sweep — most notably BC-2.07.001 v1.12 (line 219: "F-P3L3-M-05: Sync Stories-row narrative — S-6.05 anchor updated v1.3 → v1.5 … No behavioral changes."), which is the same class of Stories-row pin sync as v1.13 but carries no `[governance_leaf: true …]` bracket. v1.8, v1.9, v1.10 are also purely hygiene/backfill and lack the annotation.

Blast radius = 1 file (BC-2.07.001 alone within the wave-gate scope). Per partial-fix regression discipline (S-7.01): sibling-file symmetry within the same subsystem — the F1 remediation applied the annotation asymmetrically. **Intent-adjudication note:** the annotation convention itself is a candidate (POL-003 not yet ratified per `policies.yaml:58-65`), and it may be intentional to annotate only rows landed AFTER Pass-3 rather than retro-annotating all historical governance-only rows. If the intent is "annotation-shape is a going-forward convention, historical rows are grandfathered," this is not a defect. If the intent is "annotation-shape is authoritative across all governance-only rows," BC-2.07.001 v1.8/v1.9/v1.10/v1.12 need retro-annotation. Orchestrator or spec-steward to adjudicate. No behavioral impact either way. Evidence: `.factory/specs/behavioral-contracts/ss-07/BC-2.07.001.md:219-223`.

**ORCHESTRATOR ADJUDICATION (2026-07-02):** GRANDFATHER. POL-003 is a candidate policy adopted mid-session; the `governance_leaf` annotation convention applies going-forward only. Earlier governance-only rows in BC-2.07.001 (v1.8/v1.9/v1.10/v1.12) predate the convention and are intentionally left as-is. This observation is CLOSED as not-a-defect. Future governance-only bumps must carry the annotation.

### O-2 (LOW)

**In-code deferral marker location is unverified against current develop tip.** wave-schedule.md:191 cites `internal/svtnmgmt/svtnmgmt.go:770-771` as the site of the SESSION-DRAIN cascade deferral marker. I did not open svtnmgmt.go in this pass (Read budget conservation) — a future pass or the orchestrator may want to verify the line-number reference still resolves at 7fe3e29e. Deferral is otherwise well-anchored via wave-schedule text + S-BL.SESSION-DRAIN backlog entry.

### O-3 (LOW)

**Cross-story E2E coverage strategy is deferral-based, not integrated.** All eight Wave-6 stories have per-story E2E tests (e.g., `cmd/switchboard/admin_handlers_e2e_test.go` for S-6.05/S-6.07, `cmd/switchboard/console_handlers_e2e_test.go` for S-7.03 with 6 test functions) but no single test wires the create → attach → destroy sequence through one shared daemon process. The deferral to `S-BL.SESSION-DRAIN` is documented and legitimate; this observation is a note that Wave-7+ should ensure S-BL.SESSION-DRAIN carries the multi-story E2E test that this wave-gate deferred, not just a single-cascade unit test. No blocking concern for W-6.

## Novelty Assessment

Novelty: LOW-MEDIUM. O-1 (annotation-shape drift within BC-2.07.001) is a novel sibling-symmetry angle on the F1 remediation shipped in Pass-3 Adv-B — the fix applied to BC-2.08.001 was not mirrored to the peer BC's earlier governance-only rows. This is the partial-fix regression discipline pattern applied to a governance/annotation surface, not a content defect. O-2/O-3 are structural notes for future work.

## Verdict

**CONVERGENT_L2L3** — 0 CRIT / 0 HIGH / 0 MED / 0 LOW; 3 Observations. L2 and L3 gates all PASS.
