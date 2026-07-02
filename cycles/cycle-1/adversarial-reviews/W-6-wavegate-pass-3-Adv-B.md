---
artifact_id: W-6-wavegate-pass-3-Adv-B
document_type: wave-adversarial-review
scope: wave-gate-integration
wave: W-6
tranche: combined
stories: [S-BL.LOOKUP, S-W5.04, S-6.07, S-7.01, S-7.02, S-BL.ROUTER-ADDR, S-7.03, S-6.05]
lens: [L2, L3]
develop_tip_claimed: "7fe3e29"
develop_tip_observed: "7fe3e29e4358df16e4e2f1de65a4e0d972540b4a"
pass_number: 3
attempt_number: 1
sub_adversary: Adv-B
verdict: CONVERGENT_L2L3
findings:
  critical: 0
  high: 0
  medium: 1
  low: 0
observations: 2
reviewer_context: fresh
prior_passes_read: false
worktree_identity_tuple_verified: true
dispatch_integrity_failure: false
timestamp: 2026-07-02T00:00:00Z
---

# Wave-6 Wave-Gate Adversarial Review — Pass 3 — Adv-B (L2 + L3)

## Preflight

- `.git/HEAD` → `ref: refs/heads/develop`
- `.git/refs/heads/develop` → `7fe3e29e4358df16e4e2f1de65a4e0d972540b4a`
- basename(pwd) → `switchboard-blue`
- Prior-pass sidecars not read; fresh context.

## Scope

Perimeter-3 wave-gate integration review, L2 (test rigor) + L3 (traceability/governance).

## L2 — Test Rigor

### L2-Q1 Race-coverage — PASS

Grep `t.Parallel|go func|sync.` per new-package Wave-6 test file:
- `internal/svtnmgmt/svtnmgmt_test.go` — 57
- `internal/discovery/discovery_test.go` — 40
- `internal/arq/arq_test.go` — 32
- `internal/arq/fec_test.go` — 16
- `admin_handlers_e2e_test.go`, `console_handlers_e2e_test.go` — full in-process daemon spin-up per test

Every new-package test file carries substantive concurrency scaffolding.

### L2-Q2 Cross-story E2E create→attach→destroy — Deferred (Obs O-1)

E2E split across admin_handlers_e2e_test.go (create+destroy) + console_handlers_e2e_test.go (attach+detach). Three-way seam not covered end-to-end in one test body. Deferral to `S-BL.CONSOLE-OBS` scheduled per STORY-INDEX:137. Not a finding.

### L2-Q3 Tautology sweep — PASS

Zero matches for self-comparing idioms.

## L3 — Traceability / Governance

### L3-Q1 BC-anchor version-pin chain

**S-6.05 → BC-2.07.001:** version 1.13 (line 5), v1.13 changelog line 218 carries `[governance_leaf: true — downstream story/VP pins DO NOT need to re-sync per drbothen/vsdd-factory#429 draft policy]`. VP-048 pin at v1.12 correctly terminates via governance-leaf. PASS.

**S-7.03 → BC-2.08.001:** version 1.4 (line 5), v1.4 changelog (line 141) — behavioral rewording of Inv-3, governance_leaf correctly absent. v1.3 changelog (line 142) — declared "POL-003 candidate sync… No behavioral changes" — governance-only sync — but LACKS `governance_leaf: true` annotation. See F1.

### L3-Q2 VP coverage count — PASS

VP-INDEX.md:138 Total = 76. L140 recount 2026-06-30: P0=54, P1=18, P2=4. Meets target 76+.

### L3-Q3 POL-003 Exception A annotation-shape consistency — SEE F1

`governance_leaf` present on BC-2.07.001 v1.13 (line 218), BC-INDEX v3.0 (line 127), STORY-INDEX v3.60 (line 186), sprint-state.yaml 346/357/358. ABSENT from BC-2.08.001 v1.3 (line 142) despite identical governance-only intent.

## Findings

### F1 — MEDIUM — Governance annotation-shape drift: BC-2.08.001 v1.3 governance-only bump lacks `governance_leaf` annotation

**File:** `.factory/specs/behavioral-contracts/ss-08/BC-2.08.001.md:142`

**Evidence.** v1.3 changelog cell: "F-P3L3-MED-001: bump Stories row cell reference to S-7.03 v1.3 (POL-003 candidate sync — story v1.2→v1.3 landed 2026-07-02, this row was stale). No behavioral changes." Textually identical intent to BC-2.07.001 v1.13 (line 218), which carries the explicit `[governance_leaf: true — downstream story/VP pins DO NOT need to re-sync per drbothen/vsdd-factory#429 draft policy]` annotation.

**Impact.** No behavioral defect on develop. Risk is process-level: mechanized POL-003 cascade check cannot programmatically distinguish "governance-only, cascade terminates" from "content bump, cascade required" without the machine-readable annotation.

**Confidence.** HIGH — file/line unambiguous.

**Suggested remediation.** Retro-annotate v1.3 changelog with `[governance_leaf: true — Stories-row pin sync, downstream VP/story pins DO NOT need to re-sync per POL-003 Exception A]`.

## Observations

### O-1 — Cross-tranche integration E2E deferred to backlog (S-BL.CONSOLE-OBS)

Well-documented; not a defect.

### O-2 — [process-gap] `governance_leaf` annotation authored by hand, not enforced

Second occurrence of same authoring gap crosses process-gap threshold. Suggest pre-commit or CI check that flags BC changelog row whose text contains "No behavioral changes" or "governance-only" but lacks `governance_leaf:` token.

## Verdict

**CONVERGENT_L2L3** (adversary judgment: MEDIUM is governance-hygiene, non-blocking behaviorally). Under strict BC-5.39.001 clean-pass criterion, MEDIUM finding breaks the streak — remediation + fresh pass required.
