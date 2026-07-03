---
artifact_id: P5-pass-13-Adv-A
document_type: adversarial-review
phase: 5
pass: 13
adversary: A
verdict: HAS_FINDINGS
findings_count:
  high: 1
  medium: 1
  low: 0
  obs: 2
timestamp: 2026-07-03T00:00:00Z
reviewed_develop_head: 66e9ddc
reviewed_spec_version: "1.24"
---

# Phase 5 Pass 13 — Adversary A Review

**Verdict:** HAS_FINDINGS

**Develop HEAD reviewed:** 66e9ddc
**Spec version reviewed:** interface-definitions.md v1.24

## Findings

### F-P5P13-A-001 [HIGH] — list-keys admission gate removed along with authority gate

**Anchor:** BC-2.05.004 Precondition 1 (F-L2-003), `cmd/switchboard/admin_handlers.go` `makeListKeysHandler`

**Description:** F-L2-003 removed the CONTROL-only authority gate for `admin.key.list-keys`, allowing any admitted role to call it. However, the implementation inadvertently also removed the ADMISSION gate, meaning any caller with a valid key — even one admitted to a DIFFERENT SVTN — could call `admin.key.list-keys` on a target SVTN and receive its full admitted key roster. This is a CWE-862 (Missing Authorization) violation: cross-SVTN callers must not be permitted to enumerate another SVTN's admitted key set.

**Remediation shape:** Restore the admission gate in `makeListKeysHandler` to verify that the caller's key is admitted to the target SVTN (in any active role), is a member of the operator-set, or is the daemon bootstrap key. Cross-SVTN callers must receive E-ADM-009. The authority gate (control-only check) must NOT be restored — F-L2-003 ruling stands.

**Adjudication:** CODE track — admission gate must be enforced in implementation.

---

### F-P5P13-A-002 [MED] — E-CFG-001 token absent from sbctl list-keys error output

**Anchor:** interface-definitions.md §111 exit-code column, `cmd/sbctl/admin.go:168`

**Description:** The §111 exit-code column documents E-CFG-001 for missing `--svtn` (client-side, exit 2, via `usageErrf`). However, the actual `usageErrf` call at `cmd/sbctl/admin.go:168` does not include the `E-CFG-001` token in its message string. Other commands that emit E-CFG-001 include the token explicitly in the `usageErrf` call. The token must be present in stderr so operators and tooling can discriminate the error class.

**Remediation shape:** Add the `E-CFG-001` token to the `usageErrf` call at `cmd/sbctl/admin.go:168` for the missing `--svtn` case, consistent with the pattern used elsewhere on the admin surface.

**Adjudication:** CODE track.

---

### OBS-P5P13-A-001 [OBS] — BC-2.05.004 Precondition 1 list-keys authority wording lacks explicit cross-SVTN denial

**Description:** The list-keys authority sentence (F-L2-003) says "any admitted role... or operator-set member" but does not explicitly state that a key admitted to SVTN-A cannot call list-keys on SVTN-B. The admission gate distinction (authority gate vs. admission gate) is implicit in the sentence but not explicitly articulated. Future readers of the spec may interpret "any admitted role" as "any admitted role anywhere" rather than "any admitted role on the TARGET SVTN."

**Recommendation:** Spec-side sharpening — append a clarifying sentence distinguishing the authority gate from the admission gate, and naming the CWE-862 cross-SVTN enumeration defense.

---

### OBS-P5P13-A-002 [OBS] — VP-075 scope exclusion paragraph does not distinguish authority gate from admission gate

**Description:** VP-075:114 scope exclusion reads: "admin.key.list-keys is a read-only operation open to any admitted role and is NOT covered by this property." The phrase "NOT covered by this property" is technically correct (VP-075 covers the control-only authority gate), but could be misread as "list-keys has no admission requirements at all." The scope exclusion should clarify that the AUTHORITY gate is excluded, not the ADMISSION gate.

**Recommendation:** Spec-side sharpening — expand the scope exclusion paragraph to distinguish authority gate exclusion from admission gate applicability.
