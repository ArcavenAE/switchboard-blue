---
artifact_id: W-6-wavegate-pass-2-Adv-B
document_type: wave-adversarial-review
scope: wave-gate-integration
wave: W-6
tranche: combined
stories: [S-BL.LOOKUP, S-W5.04, S-6.07, S-7.01, S-7.02, S-BL.ROUTER-ADDR, S-7.03, S-6.05]
lens: [L2, L3]
develop_tip_claimed: "7fe3e29"
develop_tip_observed: "7fe3e29e4358df16e4e2f1de65a4e0d972540b4a"
pass_number: 2
attempt_number: 1
sub_adversary: Adv-B
verdict: CONVERGENT_L2L3
findings:
  critical: 0
  high: 0
  medium: 0
  low: 0
observations: 2
reviewer_context: fresh
prior_passes_read: false
worktree_identity_tuple_verified: true
dispatch_integrity_failure: false
timestamp: 2026-07-02T00:00:00Z
---

# W-6 Combined Wave-Gate — Pass 2, Adv-B (L2 + L3)

## Preflight

- `.git/HEAD` = `ref: refs/heads/develop` — OK
- `.git/refs/heads/develop` starts with `7fe3e29` (observed full SHA `7fe3e29e4358df16e4e2f1de65a4e0d972540b4a`) — OK
- cwd basename = `switchboard-blue` — OK
- Prior-pass sidecars NOT read (fresh context maintained)
- Read cap discipline observed (6 targeted Reads used)

## Scope

Perimeter-3 wave-gate integration review of 8 Wave-6 stories merged on `develop@7fe3e29`. L2 (test rigor) and L3 (traceability/governance) lenses only. Per-story surfaces assumed converged at Perimeter-1.

## L2 — Test Rigor

### L2-Q1 — Concurrency scaffolding across new-package `*_test.go` — PASS

Grep of `t\.Parallel|go func|sync\.` occurrences per new-package test file:

| File | Matches | Package new to W-6 |
|---|---:|---|
| `internal/arq/arq_test.go` | 32 | yes (S-7.01) |
| `internal/svtnmgmt/svtnmgmt_test.go` | 57 | yes (S-6.05, S-6.07) |
| `internal/discovery/discovery_test.go` | 40 | yes (S-7.02) |
| `internal/mgmt/mgmt_test.go` | 61 | yes (S-W5.04 predecessor / S-7.03 mgmt-plane) |
| `cmd/switchboard/admin_handlers_test.go` | 70 | yes (S-6.05, S-6.07) |
| `cmd/switchboard/admin_handlers_e2e_test.go` | 21 | yes (S-6.05 e2e) |

Every new-package test file for Wave-6 carries substantial concurrency scaffolding. No packages surfaced as "silent" on race territory. PASS.

### L2-Q2 — Cross-story E2E create→attach→destroy — PASS (deferral path documented)

No single Go test wires `admin.svtn.create` (S-6.07 handler) → session attach (S-7.03 console remote) → `admin.svtn.destroy` (S-6.05 handler) end-to-end through one server. The e2e file `cmd/switchboard/admin_handlers_e2e_test.go` (grep result: 5 matches for `admin.svtn.destroy`, 0 for `admin.svtn.create` and 0 for `admin.session.attach`) exercises destroy alone via in-process `SVTNMgr.CreateSVTN` scaffolding.

Deferral is explicitly documented in `.factory/cycles/cycle-1/wave-schedule.md:187–193`:

> "Destroy-with-active-console-attach cascade (SVTN destruction propagating a detach to active console attaches on sessions inside the destroyed SVTN) is deferred to `S-BL.SESSION-DRAIN` per S-6.05 v1.5 AC-002 out-of-scope note and the in-code deferral marker at `internal/svtnmgmt/svtnmgmt.go:770-771`. Wave-6 holdout HS-006 does not exercise this cascade. Manual-eval-only for W-6.C; full boundary coverage will land with `S-BL.SESSION-DRAIN`."

Deferral is anchored to a scheduled backlog story (`S-BL.SESSION-DRAIN`) and there is an in-code deferral marker. PASS.

### L2-Q3 — Tautology sweep — PASS

`fmt.Sprintf(...) == fmt.Sprintf` and `assert.Equal(t, X, X)` patterns: zero matches across all Wave-6 `*_test.go` files. PASS.

## L3 — Traceability / Governance

### L3-Q1 — BC-anchor version-pin chain — PASS with Observation

**S-6.05 chain (BC-2.07.001):**
- `.factory/stories/STORY-INDEX.md:73` — S-6.05 row cites `BC-2.07.001` (BC column, no explicit version pin in index — VP anchor row carries pin)
- `.factory/specs/behavioral-contracts/ss-07/BC-2.07.001.md:5` — frontmatter `version: "1.13"`
- `.factory/specs/verification-properties/VP-INDEX.md:74` — VP-048 `source_bc: BC-2.07.001 v1.12`
- BC-2.07.001 changelog row 1.13 (line 218):
  > "F-P4L3-MED-2 (POL-002): Traceability Stories row cite S-6.05 v1.5 → v1.7 (this fix-burst bumps story to v1.7). Governance-only. **[governance_leaf: true — downstream story/VP pins DO NOT need to re-sync per drbothen/vsdd-factory#429 draft policy]**"

Result: VP-048 pin at v1.12 (one behind BC's v1.13) is INTENTIONAL under POL-003 Exception A governance-leaf annotation. Shape correct. PASS.

**S-7.03 chain (BC-2.08.001):**
- `.factory/stories/STORY-INDEX.md:77` — S-7.03 row cites `BC-2.08.001`
- `.factory/specs/behavioral-contracts/ss-08/BC-2.08.001.md:5` — frontmatter `version: "1.4"`
- Changelog v1.4 (line 141) — behavioral rewording of Inv-3 (Unix-socket → BC-2.07.004 EC-013 defer). Substantive spec change, NOT governance-only — correctly LACKS `governance_leaf` annotation. Shape correct.

PASS. (See Obs-1 for the tangential v1.3 shape gap on BC-2.08.001.)

### L3-Q2 — VP coverage count — PASS

`.factory/specs/verification-properties/VP-INDEX.md:114` — Total VPs = **76** (Proptest 34 + Fuzz 4 + Integration 22 + E2E 10 + Benchmark 2 + Code-Audit 2 + Unit 2 = 76). Arithmetic verified line 116. Meets sprint-state target `76+`. No Wave-6-anchored VP gaps: BC-2.07.001 covered by VP-048 (integration); BC-2.08.001 covered by VP-050. PASS.

### L3-Q3 — POL-003 Exception A annotation presence — PASS

`governance_leaf` annotation present on BC-2.07.001 v1.13 changelog (line 218 verified above). BC-2.08.001 v1.4 is a behavioral change (not governance-leaf) and correctly lacks the annotation. Annotation shape matches drbothen/vsdd-factory#429 draft policy phrasing on BC-2.07.001. PASS.

## Findings

None. 0 critical / 0 high / 0 medium / 0 low at wave-gate integration perimeter for L2 + L3.

## Observations

### Obs-1 — [process-gap] BC-2.08.001 v1.3 governance-only bump lacks `governance_leaf` annotation

**Evidence:** `.factory/specs/behavioral-contracts/ss-08/BC-2.08.001.md:142`

> "1.3 | 2026-07-02 | spec-steward | F-P3L3-MED-001: bump Stories row cell reference to S-7.03 v1.3 (POL-003 candidate sync — story v1.2→v1.3 landed 2026-07-02, this row was stale). **No behavioral changes.**"

Contrast: BC-2.07.001 v1.13 (a functionally identical Stories-row cite bump) carries the explicit `[governance_leaf: true — downstream story/VP pins DO NOT need to re-sync per drbothen/vsdd-factory#429 draft policy]` annotation on line 218. BC-2.08.001 v1.3 declares itself governance-only ("POL-003 candidate sync", "No behavioral changes") but omits the annotation.

**Impact:** Non-blocking at wave-gate. This is a governance annotation-shape inconsistency, not a spec-content defect. However, it means a mechanized POL-003 Exception A audit that greps for `governance_leaf` annotations would miss BC-2.08.001 v1.3 as a governance-only sync — leaving downstream VP-050 pin-sync ambiguously interpretable (is v1.2→v1.3 a leaf-drift-safe bump, or must VP-050 re-anchor?). VP-INDEX entry 2.31 line 153 declares VP-050 v1.1→v1.2 driven by BC-2.08.001 v1.2→v1.3 sync, so the sync did propagate — but the annotation would formalize the leaf-drift-permitted status.

**Recommendation:** Retro-annotate BC-2.08.001 v1.3 with the `governance_leaf: true` shape. Confirm via product-owner whether the draft POL-003 Exception A policy has an intent-classifier for retroactive annotation of pre-policy governance-only rows. Tagged `[process-gap]` because the pattern (governance-only bumps missing the annotation) will recur across BCs unless the annotation-shape becomes a template obligation on the spec-steward path.

Severity: LOW (pending intent verification — the annotation gap may be historical residue from before the drbothen/vsdd-factory#429 draft policy landed on 2026-07-02).

### Obs-2 — Cross-story lifecycle E2E is not exercised in Wave-6 tests (deferral path exists and is well-documented)

**Evidence:** 
- `cmd/switchboard/admin_handlers_e2e_test.go` — 5 matches for `admin.svtn.destroy`, 0 for `admin.svtn.create` at handler-invocation call sites, 0 for `admin.session.attach`. Destroy tests set up SVTN state via in-process `SVTNMgr.CreateSVTN` (test scaffolding) rather than round-tripping through the S-6.07 handler.
- `.factory/cycles/cycle-1/wave-schedule.md:187–193` documents deferral to `S-BL.SESSION-DRAIN`.
- In-code deferral marker at `internal/svtnmgmt/svtnmgmt.go:770-771` (per wave-schedule citation; not independently opened this pass due to read cap).

**Impact:** Non-blocking. The lifecycle handler path (create RPC → destroy RPC through the same mgmt server session) is defensible-by-composition: each RPC is independently e2e-tested via `sendAdminRPC` against a real `mgmt.Server`, and the wave-schedule cross-tranche synthesis (Adv-A Q2, Q4 references in the schedule) documents that shared error taxonomies do not collide. However, no single test exercises the combined path, so a regression that only surfaces when create-then-destroy execute against the same live server instance (e.g., handler-registration ordering, shared table-state contention, RPC ID collision) is not covered by a machine gate.

Not `[process-gap]` because the deferral is anchored to a scheduled backlog story with a code marker. Marking as Obs so the sequencing (S-BL.SESSION-DRAIN before Wave-7 SVTN-lifecycle-consuming work) remains visible.

Severity: LOW.

## Verdict

**CONVERGENT_L2L3.**

- L2 Q1/Q2/Q3: PASS
- L3 Q1/Q2/Q3: PASS

0 critical, 0 high, 0 medium, 0 low findings. 2 observations, both LOW severity, one tagged `[process-gap]` (Obs-1) — non-blocking at wave-gate but merits orchestrator visibility during cycle-closing checklist.
