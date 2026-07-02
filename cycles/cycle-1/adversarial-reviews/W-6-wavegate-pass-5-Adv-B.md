---
artifact_id: W-6-wavegate-pass-5-Adv-B
document_type: wave-adversarial-review
scope: wave-gate-integration
wave: W-6
tranche: combined
stories: [S-BL.LOOKUP, S-W5.04, S-6.07, S-7.01, S-7.02, S-BL.ROUTER-ADDR, S-7.03, S-6.05]
lens: [L2, L3]
develop_tip_claimed: "7fe3e29"
develop_tip_observed: "7fe3e29e4358df16e4e2f1de65a4e0d972540b4a"
pass_number: 5
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

# Adversarial Review — Wave-6 Wave-Gate — Pass 5 attempt 1 (Adv-B, L2/L3)

## Preflight

- `.git/HEAD` → `ref: refs/heads/develop` (PASS)
- `.git/refs/heads/develop` → `7fe3e29e4358df16e4e2f1de65a4e0d972540b4a` (matches claimed `7fe3e29`) (PASS)
- basename of cwd → `switchboard-blue` (PASS)
- prior-pass reviews NOT read
- read cap honored (6 file Reads)

## Scope

Perimeter-3 wave-gate integration review of Wave-6 combined tranche merged to develop@7fe3e29. Lenses: L2 (test rigor) and L3 (traceability/governance).

## L2 Findings

### L2-Q1 — Race coverage across new-package `*_test.go` (PASS)

Grep for `t.Parallel|go func|sync.` across new-package test files:
- `cmd/switchboard/admin_handlers_e2e_test.go` — 21 hits
- `cmd/switchboard/admin_handlers_test.go` — 70 hits
- `internal/arq/arq_test.go` — 32 hits
- `internal/arq/fec_test.go` — 16 hits
- `internal/svtnmgmt/svtnmgmt_test.go` — 57 hits
- `internal/discovery/discovery_test.go` — 40 hits

Every new-package test file that touches concurrent state carries scaffolding. **PASS.**

### L2-Q2 — Cross-story E2E create → attach → destroy (PASS with observation)

No dedicated test exercising the full `admin.svtn.create → console.attach → admin.svtn.destroy` sequence through a single shared daemon was found (grep for `create.*attach.*destroy`, `SVTNCreate.*ConsoleAttach`, etc. returned no matches in `**/*_test.go`).

Deferral IS documented (Q2 acceptance criterion met):
- `.factory/cycles/cycle-1/wave-schedule.md:187-193` — "Deferred cross-story behavior (out-of-scope for W-6.C). Destroy-with-active-console-attach cascade..."
- In-code marker: `internal/svtnmgmt/svtnmgmt.go:770` — `// Session-terminated notification for active sessions is deferred to S-BL.SESSION-DRAIN`
- Backlog story reference: `S-BL.SESSION-DRAIN` cited in both loci

The wave-schedule deferral names specifically the *destroy-with-active-attach cascade*. A general happy-path create→attach→destroy sequence through one shared daemon is not explicitly deferred nor implemented — reported as observation, not blocking. **PASS.**

### L2-Q3 — Tautology sweep (PASS)

- `assert\.Equal\(t,\s*X\s*,\s*X\s*\)` pattern — 0 matches across `**/*_test.go`
- `fmt.Sprintf(...) == fmt.Sprintf(...)` — 0 matches
- identity self-compare `X == X` — 0 matches

**PASS.**

## L3 Findings

### L3-Q1 — BC-anchor version-pin chain for S-6.05 and S-7.03 (PASS with observation)

**BC-2.07.001 (S-6.05 anchor):**
- Frontmatter version: `1.13` ✓
- Terminal changelog row v1.13 present with `[governance_leaf: true — downstream story/VP pins DO NOT need to re-sync per drbothen/vsdd-factory#429 draft policy]` annotation ✓
- Traceability Stories row body pins `S-6.05 v1.8` (line 206)
- STORY-INDEX shows S-6.05 currently at v1.11 (row 3.65)
- VP-INDEX VP-048 row pins `BC-2.07.001 v1.12` and body notes source_bc pinned to v1.12
- POL-003 Exception A annotation is present at v1.13 — story/VP downstream lag is explicitly permitted.

**BC-2.08.001 (S-7.03 anchor):**
- Frontmatter version: `1.5` ✓
- Terminal changelog rows carry appropriate annotations (see L3-Q3/Q4)
- Body Stories row pins `S-7.03 v1.4` (line 127); STORY-INDEX shows S-7.03 at v1.6 (rows 3.61/3.63)
- VP-INDEX VP-050 pins BC-2.08.001; version-pin chain consistent within governance_leaf exception.

**PASS** on the strict Q1 criterion — POL-003 Exception A annotation is present for both terminal governance-only rows.

**Observation (governance hygiene, not a finding per POL-003 Exception A):** BC-2.07.001 terminal changelog v1.13 description text says the fix-burst bumped the anchor to "S-6.05 v1.7" ("Stories row cite S-6.05 v1.5 → v1.7 (this fix-burst bumps story to v1.7)"), but the body Traceability Stories row (line 206) actually reads `S-6.05 v1.8`. STORY-INDEX row 3.60 shows a subsequent v1.7→v1.8 bump on the same day. The body appears to have been updated to v1.8 without an accompanying changelog row. Per the governance_leaf exception this lag/drift is not blocking, but the self-inconsistency between v1.13's stated `→ v1.7` and the body's `v1.8` is worth noting for cycle-close hygiene.

### L3-Q2 — VP-INDEX total count arithmetic consistency (PASS)

- Summary row: `76`
- Per-tool sum: `34 + 4 + 22 + 10 + 2 + 2 + 2 = 76` ✓
- Phase sum: `54 + 18 + 4 = 76` ✓
- Explicit arithmetic-check line at 116: "34 + 4 + 22 + 10 + 2 + 2 + 2 = 76. Consistent." ✓
- Row count grep (`^| VP-`) returns 78; two rows are placeholders (`VP-TBD-ACC`, `VP-VW6.NN`) explicitly footnoted as Phase=deferred/Status=deferred and excluded from active tallies → 78 − 2 = 76 active rows matching total ✓
- Discussion narrative (rows 117–129) numerically consistent with counts.

**PASS.**

### L3-Q3 — POL-003 Exception A annotation shape consistency across NEW rows (PASS)

Only NEW governance-only rows without annotation would be defects (grandfathered rows explicitly excluded per task guidance).

- **BC-2.07.001 v1.13** (terminal, governance-only): annotation present ✓
- **BC-2.08.001 v1.5** (terminal, annotation-shape correction): annotation present ✓
- **BC-2.08.001 v1.3** (governance-only Stories-row pin sync): annotation present ✓
- **BC-2.08.001 v1.4**: substantive Inv-3 rewording (not governance-only) — no annotation required ✓
- Earlier BC-2.07.001 rows v1.8/v1.9/v1.10/v1.12 grandfathered per task guidance — not flagged.

**PASS.**

### L3-Q4 — Annotation shape consistency between BCs (PASS with observation)

Terminal annotation shapes:
- BC-2.07.001 v1.13: `[governance_leaf: true — downstream story/VP pins DO NOT need to re-sync per drbothen/vsdd-factory#429 draft policy]`
- BC-2.08.001 v1.3: `[governance_leaf: true — Stories-row pin sync, downstream VP/story pins DO NOT need to re-sync per POL-003 Exception A]`
- BC-2.08.001 v1.5: `[governance_leaf: true — annotation-shape correction, downstream VP/story pins DO NOT need to re-sync per POL-003 Exception A]`

Core substance is invariant (`governance_leaf: true`, em-dash separator, `DO NOT need to re-sync` clause). Reference wording drifts: BC-2.07.001 cites `drbothen/vsdd-factory#429 draft policy`; BC-2.08.001 cites `POL-003 Exception A`. Both refer to the same policy but under different names. Task guidance classifies this as observation, not finding.

**PASS.**

## Findings

None (0 critical / 0 high / 0 medium / 0 low).

## Observations

- **Obs-1 [L2-Q2]:** No cross-story create→attach→destroy happy-path E2E through one shared daemon was located; wave-schedule.md:187-193 documents the destroy-with-active-attach cascade deferral to `S-BL.SESSION-DRAIN` (in-code marker at `internal/svtnmgmt/svtnmgmt.go:770`), but a general happy-path integration test through one daemon is neither present nor named as a deferred item. Not blocking.
- **Obs-2 [L3-Q4 / L3-Q1]:** Two governance-hygiene items worth noting for cycle close: (a) POL-003 Exception A annotation cites `drbothen/vsdd-factory#429 draft policy` in BC-2.07.001 v1.13 but `POL-003 Exception A` in BC-2.08.001 v1.3/v1.5 — naming convention should converge for future rows; (b) BC-2.07.001 v1.13 changelog description says `→ v1.7` but body Stories row (line 206) reads `S-6.05 v1.8`, suggesting an undocumented body edit — permitted under governance_leaf exception but internally self-inconsistent.

## Novelty Assessment

Novelty: LOW. Both observations are governance/hygiene items that do not affect merged-code behavior. All L2 axes pass cleanly (concurrency scaffolding present in all new-package tests, zero tautology patterns detected). L3 arithmetic and traceability chains are internally consistent under POL-003 Exception A. Fresh-context review consistent with wave-gate convergence.

## Verdict

**CONVERGENT_L2L3** — L2 test rigor: all three axes PASS (race coverage present, cross-story deferral documented, zero tautology patterns). L3 traceability/governance: all four axes PASS under POL-003 Exception A (BC pins present with governance_leaf annotations where required, VP-INDEX arithmetic consistent 76=76=76, annotation shape substantively consistent between BCs). Two governance-hygiene observations logged for cycle-close consideration but no blocking findings.
