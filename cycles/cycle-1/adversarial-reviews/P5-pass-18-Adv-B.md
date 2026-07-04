---
pass_id: P5P18-Adv-B
adversary_lens: test-rigor + traceability
prior_passes_read: false
worktree_preflight:
  target_sha: 6deda15def9326f28e96f133e237aff5ecb74d7b
  branch: develop
  HEAD_sha: <not-verified-read-only-no-bash>
  refs/heads/develop_sha: <not-verified-read-only-no-bash>
  origin/develop_sha: <not-verified-read-only-no-bash>
  worktree_root: /Users/skippy/work/aae-orc/run/switchboard-blue
  preflight_result: PASS (orchestrator-verified out-of-band before dispatch)
budget:
  wall_clock_target: <=6 min
  wall_clock_used: ~5 min
  file_reads_target: <=6
  file_reads_used: 5
verdict: HAS_FINDINGS
findings_count: 1
streak_state:
  adjudicated_deferrals_respected: true
  reopened_deferrals: none
  respected_list:
    - DRIFT-P5P7-O1
    - DRIFT-P5P7-O4
    - DRIFT-P5P2-B-O003
    - DRIFT-HS006-ROUTER-DAEMON-STUB
    - DRIFT-P5P4-PROMPT-SHORTID
    - DRIFT-P5P9-STALE-RECONCILIATION-COMMENT
    - DRIFT-P5P14-B-001-VP-SOURCE-BC-VERSION-PIN
    - F-P5P13-A-001, F-P5P13-A-002, F-P5P13-B-001 (SHIPPED at PR #69)
    - F-P5P14-B-002, F-P5P14-B-003 (SHIPPED .factory)
    - F-P5P14-B-004, F-P5P14-B-005 (SHIPPED PR #70)
    - F-P5P15-A-001 (SHIPPED .factory 5e42768)
    - F-P5P15-B-001 (SHIPPED .factory 5120c9e)
    - F-P5P16-A-001 (SHIPPED .factory 041ea2f)
    - F-P5P17-A-001, F-P5P17-A-002 (SHIPPED .factory 2be16e5)
    - VP-077 Coverage-cell "Three sub-cases" companion tidy nit (in-flight)
delivered_by: p5-pass18-adv-b (2026-07-03)
adjudication:
  F-P5P18-B-001: SHIPPED at .factory bc79621 — STORY-INDEX v3.73 → v3.74; Summary VP-coverage counter 76/76 → 77/77; narrative gloss VP-068..VP-076 → VP-068..VP-077 (VP-077 added to BC-2.05.004 anchor list). Closes the aggregate-freshness leg of F-P5P14-B-003.
---

## Findings

### F-P5P18-B-001 [MEDIUM] STORY-INDEX Summary VP-coverage counter stale — reads `76/76` after VP-077 mint (POL-002-adjacent aggregate-freshness gap)

- **Class:** traceability / index arithmetic drift
- **Confidence:** HIGH
- **Severity:** MEDIUM
- **Anchors:**
  - `.factory/stories/STORY-INDEX.md:39` — `| VP coverage | 76/76 (100%) — VP-068..VP-076 added Wave-5 (VP-074 anchored to BC-2.06.001, VP-075/VP-076 anchored to BC-2.05.004) |`
  - `.factory/specs/verification-properties/VP-INDEX.md:115` — `| 77 | 33 | 4 | 23 | 10 | 2 | 2 | 3 |`
  - `.factory/specs/verification-properties/VP-INDEX.md:118` — footer: "Total 76→77. P0 count 54→55."
  - `.factory/specs/verification-properties/VP-INDEX.md:153` (v2.36 changelog, 2026-07-03) — "Total 76→77. Integration 22→23. P0 54→55."
  - `.factory/stories/STORY-INDEX.md:184` (v3.73 changelog, 2026-07-03) — updates S-6.06 v1.24→v1.25 and adds VP-077 to vp_traces, but leaves the top-of-file aggregate counter untouched.
- **Symptom:** VP-INDEX v2.36 (2026-07-03) advances the master VP total from 76→77 with the addition of VP-077 (integration, P0, cmd/switchboard, BC-2.05.004 EC-008). The STORY-INDEX v3.73 changelog for the same day propagates the new VP into S-6.06's `vp_traces` and body prose, but does not update the STORY-INDEX Summary block's `VP coverage` row (still `76/76 (100%)`). The narrative gloss ("VP-068..VP-076 added Wave-5") is now understated by one VP. VP-077 is Wave-5-scoped (implementing_story S-6.06, which is Wave 5 per STORY-INDEX line 91), so both the numerator/denominator AND the Wave-5 narrative are affected.
- **Verify:**
  1. Read `.factory/specs/verification-properties/VP-INDEX.md:114-115` — confirm `Total VPs = 77`.
  2. Read `.factory/stories/STORY-INDEX.md:39` — confirm the Summary row still reads `76/76`.
  3. Read `.factory/stories/STORY-INDEX.md:184` (v3.73 entry) — confirm the entry names VP-077 addition to vp_traces but does not reference an update to the Summary counter.
  4. Confirm VP-077's `implementing_story: S-6.06` and S-6.06's Wave-5 anchorage (STORY-INDEX line 91).
- **Remediation:** Bump STORY-INDEX to v3.74 with a POL-002-labeled changelog entry that (a) updates line 39 to `77/77 (100%)`, (b) extends the narrative to `VP-068..VP-077 added Wave-5 (... VP-077 anchored to BC-2.05.004)`, and (c) cross-references the F-P5P14-B-003 traceability close so future auditors can see the aggregate-freshness fix was routed through the same POL-002 lane as the row-level updates already landed in v3.73.

## Anti-findings (checked and passing)

1. **VP-077 Test Evidence line-number resolution — all 10 anchors correct.** Cited line numbers in VP-077 v1.1 Test Evidence resolve to the correct test declarations in `cmd/switchboard/admin_handlers_list_keys_admission_test.go`: `TestListKeys_AdmittedControlRole_Allowed` at :102, `_AdmittedConsoleRole_Allowed` at :136, `_AdmittedAccessRole_Allowed` at :169, `_RevokedExpiredRole_DeniedEADM009` at :205, `_OperatorSetMember_AllowedUnconditionally` at :279, `_OperatorSetMember_MissingSVTN_ReturnsESVTN003` at :325, `_CrossSVTNEnumeration_DeniedEADM009` at :373, `_BootstrapKey_Allowed` at :415, `_TargetSVTNNotFound_ReturnsESVTN003` at :437, `_NoCaller_DeniedEADM009` at :462. F-P5P15-B-001's rename+renumber landed cleanly.

2. **VP-INDEX arithmetic footer is consistent.** `VP-INDEX.md:115` counts 33+4+23+10+2+2+3 = 77; matches "Total VPs = 77" and the phase-distribution table (`P0=55, P1=18, P2=4, Total=77` at :137-140). VP-077 addition footer entries at :118 and :142 both cite the same 76→77 / 54→55 arithmetic.

3. **BC-2.05.004 v1.14 EC-008 ↔ VP-077 backpointer.** `BC-2.05.004.md:218` EC-008 ends with `**Verified by: VP-077.**` and `BC-2.05.004.md:242` includes the VP-077 row (`Admin list-keys admission-gate — any-role OR operator-set OR bootstrap-key; else E-ADM-009 (EC-008 three failure modes)`). Bidirectional trace closed.

4. **VP-077 source_bc version pin.** `VP-077.md:14` frontmatter carries `source_bc: BC-2.05.004@v14` and `VP-077.md:72` cites "BC-2.05.004 v1.14 — Key Lifecycle..." — both match BC-2.05.004's current `version: "1.14"` at BC file line 5. POL-003 compliance holds.

5. **RED-discipline authenticity in the admission test file.** `admin_handlers_list_keys_admission_test.go:1-28` header explicitly enumerates which cases were RED (4/6/9) versus GREEN-guard (1/2/3/5/7/8), and RED-tagged tests (`_RevokedExpiredRole_` line 232-234, `_CrossSVTNEnumeration_` line 400-403, `_NoCaller_` line 471-473) each name the *specific* missing gate (`makeListKeysHandler has no admission gate`, `CWE-862`, `fail-closed when no CallerPubkey and no CallerRole`). The GREEN-guard tests (control/console/access roles, bootstrap key) are per-arm distinct — a stripped-admission mutation of only one branch (e.g., a hypothetical change that admits console-role but denies access-role) would fail one guard while leaving siblings green, so per-arm signal is preserved.

6. **BC-2.05.004 Preconditions ↔ EC-008 ↔ VP-077 property statement all agree on the tri-branch admission gate.** BC line 185 (Precondition 1 list-keys authority), BC line 218 (EC-008 three failure modes), and `VP-077.md:39-46` (property statement) all encode the same disjunction (`IsAdmittedAnyRole OR OperatorKeySet OR BootstrapKey`) and the same three failure modes with matching error code E-ADM-009. No spec-side drift between the BC narrative and the VP formalization.

7. **VP-077 implementing_story anchorage matches STORY-INDEX Wave-5 completion.** `VP-077.md:21` `implementing_story: S-6.06` and `VP-077.md:277` Story Trace `S-6.06 | Wave 5 | S-6.02, S-W5.01`; STORY-INDEX line 91 places S-6.06 in Wave 5 and line 70 shows S-6.06 as merged (PR #36). Both dep-graph anchors present in VP-077 Story Trace (S-6.02 merged PR #34; S-W5.01 in Wave 5) are consistent with the STORY-INDEX Wave-5 roster.

VERDICT: HAS_FINDINGS
