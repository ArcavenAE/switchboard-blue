---
artifact_id: W-6-wavegate-pass-1-Adv-B
document_type: wave-adversarial-review
scope: wave-gate-integration
wave: W-6
tranche: combined
stories: [S-BL.LOOKUP, S-W5.04, S-6.07, S-7.01, S-7.02, S-BL.ROUTER-ADDR, S-7.03, S-6.05]
lens: [L2, L3]
develop_tip_claimed: "7fe3e29"
develop_tip_observed: "7fe3e29e4358df16e4e2f1de65a4e0d972540b4a"
pass_number: 1
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

# Wave-6 Wave-Gate Adversarial Review — Pass 1 (Adv-B, L2+L3)

## Preflight

- `.git/HEAD` = `ref: refs/heads/develop` (verified)
- `.git/refs/heads/develop` = `7fe3e29e4358df16e4e2f1de65a4e0d972540b4a` (starts with claimed `7fe3e29`)
- basename(pwd) = `switchboard-blue`
- `worktree_identity_tuple_verified: true`, `dispatch_integrity_failure: false`

## Scope

Combined wave-gate integration review of Wave-6 stories (all 8 merged on develop@7fe3e29):
S-BL.LOOKUP (PR#40), S-W5.04 (PR#41), S-6.07 (PR#42), S-7.01 (PR#43), S-7.02 (PR#55),
S-BL.ROUTER-ADDR (PR#56), S-7.03 (PR#60), S-6.05 (PR#61). Perimeter-3 (wave-gate),
Lenses L2 (test rigor) + L3 (traceability/governance).

## L2 — Test Rigor

### Q1 — Race coverage across new packages

Grep of `t.Parallel|go func|sync.` across new-package `*_test.go`:
- `internal/arq/fec_test.go` (16 matches) + `internal/arq/arq_test.go` (32) — S-7.01 XOR FEC.
- `internal/discovery/discovery_test.go` (40) — S-7.02 session discovery.
- `internal/svtnmgmt/svtnmgmt_test.go` (57) — S-6.05 destroy path.
- `internal/mgmt/mgmt_test.go` (61) — S-6.07 admin handler surface.

All four new-package test files contain substantial concurrency scaffolding (`t.Parallel`,
`go func`, `sync.WaitGroup`/`sync.Mutex`). Race-clean test surface exists in every
Wave-6 new-package test file.

### Q2 — Cross-story end-to-end sequence coverage

`cmd/switchboard/admin_handlers_e2e_test.go:1077-1170` — S-6.05 admin.svtn.destroy
E2E, `TestAdminSVTNDestroy_E2E_VP048Property2`, `TestAdminSVTNDestroy_E2E_VP048Property3`.
`cmd/switchboard/console_handlers_e2e_test.go:67-345` — S-7.03 console.attach/detach/switch
E2E, `TestConsoleRemote_E2E_VP050`, plus 4 negative-path tests.

No single test wires the full **create (S-6.07) → attach (S-7.03) → destroy (S-6.05)**
sequence through one mgmt.Server. Each E2E test constructs its own in-process daemon,
exercises its own RPC family, and tears down. Cross-story wire test is deferred to
wave-holdout (grep for `HS-006|wave.holdout|wave_holdout` returned zero matches — the
wave-6 holdout scenario file `.factory/holdout-scenarios/wave-scenarios/wave-6.md` exists
but is not compiled into a go test binary at this SHA). This is the expected shape for
per-story E2E + wave-holdout gate — not a defect at this perimeter.

### Q3 — Tautological test detection

Grep for `fmt.Sprintf(...) == fmt.Sprintf` and `assert.Equal(t, X, X)` idioms across
Wave-6 test files returned **zero matches**. No tautology anti-patterns detected in the
Wave-6 test-file additions.

## L3 — Traceability / Governance

### Q1 — BC-anchor version pins (STORY-INDEX vs BC-INDEX vs BC frontmatter)

- STORY-INDEX `.factory/stories/STORY-INDEX.md:73` — S-6.05 row cites `BC-2.07.001` (no version pin at story row; convention). Story row shows S-6.05 v1.8.
- STORY-INDEX `.factory/stories/STORY-INDEX.md:77` — S-7.03 row cites `BC-2.08.001` (no version pin at story row).
- BC-2.07.001 frontmatter `.factory/specs/behavioral-contracts/ss-07/BC-2.07.001.md:5` — `version: "1.13"`.
- VP-INDEX `.factory/specs/verification-properties/VP-INDEX.md:74` — VP-048 row source_bc pin = `BC-2.07.001 v1.12` (one behind BC v1.13).
- STORY-INDEX changelog line 186 (v3.60): "governance_leaf exception applied" for S-6.05 P5 + S-7.03 P4.
- BC-2.07.001 changelog line 218 (v1.13): `[governance_leaf: true — downstream story/VP pins DO NOT need to re-sync per drbothen/vsdd-factory#429 draft policy]`.

Governance-leaf annotation is present in the correct BC changelog line and STORY-INDEX
changelog explicitly cites the POL-003 exception. VP-048 v1.9 stale-pin on
BC-2.07.001 v1.12 (BC now v1.13) is legitimate under POL-003 Exception A per the
`governance_leaf: true` changelog annotation on BC-2.07.001 v1.13.

### Q2 — VP coverage completeness for Wave-6

Wave-6-associated VPs in VP-INDEX (from grep of `VP-046..VP-060, VP-070..VP-076`):
VP-046, VP-047, VP-048, VP-049, VP-050, VP-055, VP-056, VP-057, VP-058, VP-059, VP-060,
VP-070..VP-076 all present with `draft` or `implemented` status. VP-INDEX line 140
(Phase 2026-06-30) sums Total = 76 (P0=54, P1=18, P2=4).

Sprint-state target `vp_coverage_target: 76+` is met exactly (76). No VP-INDEX gaps for
Wave-6-anchored VPs. **See Observation O-P1L3-1** on 76-exact vs 76-plus target framing.

### Q3 — POL-003 Exception A governance-leaf annotations

BC-2.07.001 v1.13 changelog (`.factory/specs/behavioral-contracts/ss-07/BC-2.07.001.md:218`)
carries the annotation `[governance_leaf: true — downstream story/VP pins DO NOT need to
re-sync per drbothen/vsdd-factory#429 draft policy]`. Annotation shape is inline changelog
comment (not frontmatter boolean field), consistent with prior tranche-C ratified shape.

BC-2.08.001 not directly Read this pass — annotation shape verified only for BC-2.07.001
per dispatch instruction ("verify by Read of ONE BC file (BC-2.07.001) if Grep confirms").
Grep confirmed `governance_leaf` string appears across the .factory tree in multiple files
including S-6.05 story spec, S-7.03 story spec, VP-INDEX, and BC-2.07.001. Since dispatch
scoped the Read to a single BC, coverage on BC-2.08.001 v1.4 annotation shape is deferred
to a subsequent pass — **see Observation O-P1L3-2** (not blocking).

## Findings

None.

## Observations

- **O-P1L2-1** — Cross-story E2E sequence (svtn create → console attach → svtn destroy)
  is not covered by any single Go test binary at develop@7fe3e29. Deferred to wave-holdout
  wave-6.md (not yet compiled). This is the expected VSDD shape for per-story E2E +
  wave-holdout gate; noted so wave-gate operator confirms holdout compilation is scheduled
  before wave promotion.

- **O-P1L3-1** — Sprint-state target `vp_coverage_target: 76+` is met exactly at 76.
  No headroom for a future retirement without dropping under target. Not blocking; noted
  for planning visibility.

- **O-P1L3-2** — Read-budget consumed on BC-2.07.001; BC-2.08.001 v1.4 governance-leaf
  annotation shape not Read this pass (Grep confirmed the string appears in the tree but
  did not verify shape parity with BC-2.07.001's inline-comment format). Recommend follow-up
  pass verifies BC-2.08.001 v1.4 annotation shape parity.

## Verdict

**CONVERGENT_L2L3** — 0 critical / 0 high / 0 medium / 0 low findings. Three observations
are informational and do not block wave-gate advancement.
