---
pass_id: P5P15-Adv-B
adversary_lens: test-rigor + traceability
prior_passes_read: false
worktree_preflight:
  target_sha: 6deda15def9326f28e96f133e237aff5ecb74d7b
  branch: develop
  HEAD_sha: <not-verified-read-only-no-bash>
  refs/heads/develop_sha: <not-verified-read-only-no-bash>
  origin/develop_sha: <not-verified-read-only-no-bash>
  worktree_root: /Users/skippy/work/aae-orc/run/switchboard-blue
  preflight_result: PASS (paths + artifact contents consistent with claimed tip; adversary is read-only, cannot run git rev-parse; orchestrator-verified out-of-band before dispatch)
budget:
  wall_clock_target: <=6 min
  wall_clock_used: ~5 min
  file_reads_target: <=6
  file_reads_used: 3 full (policies.yaml, BC-2.05.004.md, VP-077.md) + 2 partial (S-6.06, admin_handlers.go) + grep/glob probes
  overage_disclosure: none
verdict: HAS_FINDINGS
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
    - F-P5P13-A-001, F-P5P13-A-002, F-P5P13-B-001 SHIPPED at PR #69
    - F-P5P14-B-002 SHIPPED at .factory 3994fda
    - F-P5P14-B-003 SHIPPED at .factory 426e0fa
    - F-P5P14-B-004, F-P5P14-B-005 SHIPPED at PR #70
delivered_by: p5-pass15-adv-b (2026-07-03T22:55Z)
---

# Phase 5 Pass 15 Adv-B — findings

### F-P5P15-B-001 [MED]: VP-077 Test Evidence cites test-name prefixes that don't exist

Anchor: `.factory/specs/verification-properties/VP-077.md:92-101` at develop tip 6deda15

Class: BC↔VP↔test traceability defect (test-rigor + traceability lens). Newly-minted VP-077 (F-P5P14-B-003 remediation) fabricates test-function names that do not resolve in the codebase, breaking the intended proof-anchor chain.

Symptom: VP-077 §Test Evidence enumerates three "existing tests" verbatim:
- `TestAdminListKeys_RevokedKey_Denied` (~line 208)
- `TestAdminListKeys_OperatorSetMember_AllowedUnconditionally` (~line 286)
- `TestAdminListKeys_TargetSVTNNotFound_ReturnsESVTN003` (~line 398)

All three are prefixed `TestAdminListKeys_`. Grep across the entire repository for `TestAdminListKeys` returns zero matches.

Verify:
- `cmd/switchboard/admin_handlers_list_keys_admission_test.go` actually contains, at the cited region: `TestListKeys_RevokedExpiredRole_DeniedEADM009` (line 205), `TestListKeys_OperatorSetMember_AllowedUnconditionally` (line 279), `TestListKeys_TargetSVTNNotFound_ReturnsESVTN003` (line 437). All three use prefix `TestListKeys_`, not `TestAdminListKeys_`.
- The "revoked" test name in VP-077 (`RevokedKey_Denied`) further understates the actual test's scope — the real test covers both revoked and expired (`RevokedExpiredRole_DeniedEADM009`), which is exactly EC-008 failure-mode (3) as stated in BC-2.05.004:218 (`revoked=true` or `now >= expiry`).
- Line-number drift on the third citation exceeds the "~" tolerance: ~398 cited vs. 437 actual (delta +39).

A reviewer or automated coverage-audit tool grepping VP-077's Test Evidence to confirm the property is actually verified would fail to resolve any of the three cited symbols. The tests do exist under different names, so the underlying coverage is present — but the VP↔test anchor is broken as written, defeating the purpose of the Test Evidence section as a machine-checkable trace link.

Remediation shape: In VP-077 lines 92-101, replace the three cited test names with the actual names and correct the line numbers:
- `TestAdminListKeys_RevokedKey_Denied` → `TestListKeys_RevokedExpiredRole_DeniedEADM009` (line 205); update descriptive text to note it covers both `revoked=true` AND `now >= expiry` per EC-008 failure-mode (3).
- `TestAdminListKeys_OperatorSetMember_AllowedUnconditionally` → `TestListKeys_OperatorSetMember_AllowedUnconditionally` (line 279).
- `TestAdminListKeys_TargetSVTNNotFound_ReturnsESVTN003` → `TestListKeys_TargetSVTNNotFound_ReturnsESVTN003` (line 437).

Consider adding the other 7 admission tests present in the file (`TestListKeys_AdmittedControlRole_Allowed`, `TestListKeys_AdmittedConsoleRole_Allowed`, `TestListKeys_AdmittedAccessRole_Allowed`, `TestListKeys_CrossSVTNEnumeration_DeniedEADM009`, `TestListKeys_BootstrapKey_Allowed`, `TestListKeys_NoCaller_DeniedEADM009`, `TestListKeys_OperatorSetMember_MissingSVTN_ReturnsESVTN003`) for full coverage of the property statement — the file already exercises all three EC-008 failure modes plus all three admission-success paths, but VP-077 currently under-cites this coverage.

Adjudication: **SHIPPED** at `.factory` commit `5120c9e` (Burst 42a) — VP-077 v1.0 → v1.1. Three test citations corrected + 7 additional citations added (grouped into 4 coverage categories).

Follow-on note (non-blocking): steward flagged that VP-077 §Proof Method Coverage cell still reads "Three sub-cases" and was left intact because it documents the proof harness skeleton not the test suite. Not preemptively rev'd; if a future adversary flags as drift it becomes an anchored finding.

## Anti-findings (checked and passing)

- BC-2.05.004 v1.14 modified: entry (lines 125-133) correctly summarizes the F-P5P14-B-003 remediation content (EC-008 references VP-077; VP table extended with VP-077 row). POL-001 satisfied for v1.14 bump.
- S-6.06 v1.25 modified: entry (line 15) correctly documents F-P5P14-B-002 remediation (AC-006 body-prose sync; VP-077 added to vp_traces; anchor at admin_handlers.go:363). POL-001 satisfied.
- S-6.06 v1.25 impl anchor claim (`cmd/switchboard/admin_handlers.go:363 resolveCallerAdmissionAnyRole`) resolves — line 363 is the call site inside `makeListKeysHandler` (definition at line 592). Verified.
- STORY-INDEX.md line 70 correctly shows S-6.06 status cell as `merged (PR #36, 3ee9c38) (v1.25)`. POL-002 satisfied.
- VP-077 frontmatter `source_bc: BC-2.05.004@v14` matches BC-2.05.004 current version 1.14 (line 4). Cross-reference intact.
- VP-077 EC-008 three-mode enumeration (lines 51-60) aligns byte-for-byte with BC-2.05.004:218 EC-008 body. No drift on failure-mode text.
- BC-2.05.004 Verification Properties table (line 242) lists VP-077 row with correct property description, matching VP-077.md §Property Statement. Bidirectional anchor intact.
- S-6.06 vp_traces frontmatter (lines 26-29) lists VP-075, VP-076, VP-077 — matches BC-2.05.004 v1.14 VP table.
- `TestE2E_AdminListKeys_AnyRole` cited in S-6.06 AC-006 body prose exists at `cmd/switchboard/admin_handlers_e2e_test.go:1039`. Verified.
- POL-003 (candidate) not enforced — respected. Adjudicated deferral DRIFT-P5P14-B-001 not re-raised.

---

VERDICT: HAS_FINDINGS
