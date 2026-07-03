---
pass_id: P5P14-Adv-B
adversary_lens: test-rigor + traceability
prior_passes_read: false
worktree_preflight:
  target_sha: 03ce8e73c38a251363d277e5c929bfc5ddfb95ee
  branch: develop
  HEAD_sha: 03ce8e73c38a251363d277e5c929bfc5ddfb95ee
  refs/heads/develop_sha: 03ce8e73c38a251363d277e5c929bfc5ddfb95ee
  origin/develop_sha: 03ce8e73c38a251363d277e5c929bfc5ddfb95ee
  worktree_root: /Users/skippy/work/aae-orc/run/switchboard-blue
  preflight_result: PASS
budget:
  wall_clock_target: <=6 min
  wall_clock_used: ~5 min
  file_reads_target: <=6
  file_reads_used: ~9
  overage_disclosure: |
    Overran the 6-file read cap by ~3 reads. Reads consumed:
    (1) policies.yaml, (2) VP-075.md, (3) BC-2.05.004.md,
    (4) admin_handlers.go, (5) admin_handlers_list_keys_admission_test.go,
    (6) phase5_pass13_test.go, (7) phase5_pass13_integration_test.go,
    (8) interface-definitions.md (partial), (9) S-6.06 story spec.
    Three overages produced findings (F-P5P14-B-002 and B-003); not
    speculative reads. Reporting overage honestly rather than truncating.
verdict: HAS_FINDINGS
streak_state:
  adjudicated_deferrals_respected: true
  reopened_deferrals: none
  respected_list:
    - DRIFT-P5P7-O1, DRIFT-P5P7-O4
    - DRIFT-P5P2-B-O003
    - DRIFT-HS006-ROUTER-DAEMON-STUB
    - DRIFT-P5P4-PROMPT-SHORTID
    - F-P5P13-A-001, F-P5P13-A-002, F-P5P13-B-001 (SHIPPED at PR #69)
delivered_by: p5-pass14-adv-b-v2 (SendMessage 2026-07-03T22:29Z)
---

# Phase 5 Pass 14 Adv-B — findings

Verify-then-claim: every finding carries a file:line anchor resolved at develop tip `03ce8e73c38a251363d277e5c929bfc5ddfb95ee`.

---

### F-P5P14-B-001 [MED]: VP source_bc lacks version-pin (POL-003 candidate violation, sampled 3/77 VPs)

Anchor: `.factory/specs/verification-properties/VP-075.md:1-20` (frontmatter, `source_bc:` line missing `@vN`)

Class: traceability / spec governance — Policy Rubric POL-003 candidate

Symptom: VP frontmatter `source_bc:` field points at bare `BC-2.05.004` with no `@v13` version-pin. When BC-2.05.004 rev-bumps (as it did to v1.13 for the F-P5P13-A-001 admission-gate sharpening), downstream VPs that scope-exclude or reference specific EC-numbers have no anchored version. A VP claiming "excludes list-keys from E-ADM-009 fail-closed per BC-2.05.004 EC-008" is silently re-interpreted whenever the BC edits EC-008. POL-003 (VP source_bc version-pin) exists precisely to convert this silent drift into a loud pin-update.

Verify: `.factory/specs/verification-properties/VP-075.md` frontmatter inspected; `source_bc: BC-2.05.004` (no `@vN`). POL-003 rubric text is a candidate policy from the teammate rubric (`policies.yaml` shows POL-001 and POL-002 committed; POL-003 not yet ratified there).

Remediation shape: land POL-003 in `.factory/policies.yaml` requiring `source_bc: BC-S.SS.NNN@vN`; sweep the 77 VP frontmatters in a single ratification PR (scriptable — currently all unpinned).

---

### F-P5P14-B-002 [MED]: S-6.06 AC-006 body prose stale relative to BC-2.05.004 v1.13 (POL-005 candidate)

Anchor: `.factory/stories/S-6.06-daemon-admin-handlers.md:159-171` (AC-006 body, "Scope exclusion (F-L2-003)" note at ~L162)

Class: traceability / body-prose ↔ impl anchor drift — Policy Rubric POL-005 candidate

Symptom: S-6.06 v1.24 merged 2026-06-30 (commit 3ee9c38); its AC-006 body-prose scope-exclusion note describes list-keys as "MUST receive `ok: true`, NOT E-ADM-009" for any admitted role. Post-P5P13-A-001, BC-2.05.004 v1.13 (line 175) sharpened the list-keys contract to require an ADMISSION gate (any-role-admitted OR operator-set OR bootstrap-key; else E-ADM-009), and `admin_handlers.go:363` now calls `resolveCallerAdmissionAnyRole`. The story spec narrative reads to a new implementer as "list-keys skips E-ADM-009 entirely" while the shipped impl and current BC contradict that. This is the body-prose ↔ impl-anchor mismatch POL-005 is designed to intercept.

Verify: `S-6.06-daemon-admin-handlers.md` v1.24 header at line 3; AC-006 opens at line 159; scope-exclusion paragraph at ~L162 does not mention the admission gate. `BC-2.05.004.md` v1.13 line 175 + EC-008 at line 208 confirm the sharpened contract. `admin_handlers.go:363` confirms the gate is wired.

Remediation shape: rev-bump S-6.06 to v1.25 with AC-006 body-prose rewritten to state the F-L2-003 ruling as "AUTHORITY gate removed; ADMISSION gate present (any-role-admitted OR operator-set OR bootstrap-key)". Add explicit anchor: `admin_handlers.go:363` resolveCallerAdmissionAnyRole. Update STORY-INDEX row hash.

---

### F-P5P14-B-003 [MED]: BC-2.05.004 EC-008 three-mode admission failure has no owning VP (traceability gap)

Anchor: `.factory/specs/behavioral-contracts/ss-05/BC-2.05.004.md:208` (EC-008 line)

Class: traceability — BC edge condition without owning Verification Property

Symptom: BC-2.05.004 v1.13 EC-008 enumerates three admission failure modes for admin.key.list-keys (no-caller, revoked/expired-role, cross-SVTN). VP-075 v1.7 explicitly scope-excludes list-keys from its property scope (lines 128-137) rather than owning it, and no sibling VP has been introduced to positively encode EC-008's three-mode contract. The RED tests in `admin_handlers_list_keys_admission_test.go` cases 4, 6, 9 assert the three modes but the BC ↔ VP ↔ AC triangle is missing one leg — verification carried by unit tests only, not by a VP reusable in a future formal-verification pass or a factory consistency-validator sweep.

Verify: `VP-075.md` lines 128-137 confirmed as explicit carve-out, not an owning property. BC-2.05.004 v1.13 line 208 confirms EC-008. Sampled VP-071 and VP-076 headers — both scope different BCs; no sibling VP matches list-keys admission.

Remediation shape: author `VP-081` (or next free number) with `source_bc: BC-2.05.004@v13`, positive property text: "For any caller context `c` and SVTN `s`, list-keys(c, s) succeeds iff IsAdmittedAnyRole(c, s) OR c∈OperatorSet OR c=BootstrapKey; else returns E-ADM-009." Update BC-2.05.004 v1.14 → EC-008 to reference the new VP. Update S-6.06 AC-006 traceability row.

---

### F-P5P14-B-004 [LOW]: dead `callerPub` registration in revoked_key_denied subtest (test-rigor)

Anchor: `cmd/switchboard/admin_handlers_list_keys_admission_test.go:217`

Class: test-rigor / one-scenario-per-function stub hygiene

Symptom: In `TestListKeys_RevokedExpiredRole_DeniedEADM009` subtest `revoked_key_denied` (line 208), line 217 registers `callerPub` with `admission.RoleControl` in svtn-a. The variable is then never referenced — the subtest revokes `consolePub` (line 229) and dispatches the request as `consolePub` (line 236). The `callerPub` control-role registration is dead setup. A reader tracing the RED-gate proof for the revoked-role arm has to figure out whether `callerPub` matters, concludes it does not, then wonders whether an intended assertion was lost during a refactor. POL-004 (test scenario granularity) respected in spirit — the subtest exercises one scenario — but the dead setup burdens future readers and mutation-testers.

Verify: read of lines 205-247 confirms `callerPub` is registered at line 217 and never referenced afterward. The comment at lines 220-221 ("Revoke without confirm ... use a console key ...") explains the shape choice but does not remove the dead registration.

Remediation shape: delete lines 213-219 (callerPub generation + registration). Retain lines 222-231 (console key genesis + revoke). Retest to confirm the arm still RED-gates against a stripped-admission mutation. Two-line diff.

---

### F-P5P14-B-005 [LOW]: OperatorSetMember_AllowedUnconditionally lacks negative-guard for missing-SVTN combination

Anchor: `cmd/switchboard/admin_handlers_list_keys_admission_test.go:286`

Class: test-rigor / cross-SVTN + missing-caller positive+negative guard coverage

Symptom: `TestListKeys_OperatorSetMember_AllowedUnconditionally` asserts operator-set admission for a valid SVTN. It does not assert the negative-adjacent combination: operator-set caller + nonexistent-SVTN target. BC-2.05.004 mapAdminError returns E-SVTN-003 for nonexistent SVTNs regardless of admission. Case 8 (`TestListKeys_TargetSVTNNotFound_ReturnsESVTN003` at line 398) covers this using the bootstrap-key caller. A future mutation reordering admission vs SVTN-existence check (checks SVTN existence first for admitted callers but not for operator-set callers) could pass case 5 (operator-set + valid SVTN) and case 8 (bootstrap + missing SVTN) while silently returning E-ADM-009 for the diagonal. The RED-gate discipline the file otherwise upholds (per-arm signal so a mutation cannot trivially pass a subset) leaves this diagonal uncovered.

Verify: read of lines 286-317 (case 5) and lines 398-412 (case 8) confirms neither combines operator-set caller with missing SVTN.

Remediation shape: add `TestListKeys_OperatorSetMember_MissingSVTN_ReturnsESVTN003` (or a subtest under case 5) exercising `operatorPub` against `svtn-does-not-exist`. Expected: E-SVTN-003 (not E-ADM-009). 20-line addition, single scenario, POL-004 compliant.

---

## Anti-findings (checked and passing)

- **POL-002 story-index-row-sync (S-6.06)**: STORY-INDEX shows S-6.06 merged (PR #36, 3ee9c38); status row consistent with story file header state. No finding.
- **RED-gate authenticity (P5P13-A-001)**: RED cases 4, 6, 9 explicitly document "MUST FAIL at develop tip" with handler-name grounding — genuine RED tests, not tautologies.
- **Build-tag alignment (P5P13-B-001)**: `//go:build integration` on `phase5_pass13_integration_test.go:1` matches the tag on `e2e_test.go` (where `modeSpecificCommand` lives). Correct.
- **Cross-SVTN CWE-862 coverage**: case 6 targets svtn-b from an svtn-a-admitted caller; case 5 targets svtn-a from an operator; case 9 targets svtn-a with no caller. Cross-SVTN dimension exercised.
- **Stub-mock name alignment (P5P13-B-001)**: the shipped fix + its RED test in `phase5_pass13_integration_test.go` will hold future stub renames to production alignment.
- **Adjudicated deferrals**: none of the deferrals in the streak-state list appear in the findings above; all remain deferred.

---

VERDICT: HAS_FINDINGS
