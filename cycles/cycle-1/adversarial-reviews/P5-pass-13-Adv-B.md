---
artifact_id: P5-pass-13-Adv-B
document_type: adversarial-review
phase: 5
pass: 13
adversary: B
verdict: HAS_FINDINGS
findings_count:
  high: 0
  medium: 0
  low: 1
  obs: 2
timestamp: 2026-07-03T00:00:00Z
reviewed_develop_head: 66e9ddc
reviewed_spec_version: "1.24"
---

# Phase 5 Pass 13 — Adversary B Review

**Verdict:** HAS_FINDINGS

**Develop HEAD reviewed:** 66e9ddc
**Spec version reviewed:** interface-definitions.md v1.24

## Findings

### F-P5P13-B-001 [LOW] — e2e stub name misalignment: admin.key.list vs admin.key.list-keys

**Anchor:** `cmd/sbctl/e2e_helpers_test.go:191`

**Description:** The e2e test helper at line 191 registers a mock for the wire command `admin.key.list`, but the production surface uses `admin.key.list-keys` (verified: §384 Registered Verbs table, `cmd/switchboard/admin_handlers.go` handler registration). The mock registration with the wrong name means the e2e test helper would fail to intercept actual `admin.key.list-keys` calls, producing a false negative (the mock is never triggered, the real handler — or an error — runs instead).

**Remediation shape:** Update `cmd/sbctl/e2e_helpers_test.go:191` to register the mock for `admin.key.list-keys` (not `admin.key.list`).

**Adjudication:** CODE track (test-writer fix).

---

### OBS-P5P13-B-001 [OBS] — Admission gate restoration may have incomplete test coverage for the cross-SVTN denial path

**Description:** The admission gate fix (F-P5P13-A-001) adds enforcement for cross-SVTN callers. It is unclear whether the existing test suite includes a sub-case for a caller that is validly admitted to SVTN-A calling list-keys on SVTN-B (cross-SVTN denial). Most existing tests focus on the caller's role within the target SVTN. A missing cross-SVTN denial test would leave the CWE-862 defense unverified by automation.

**Recommendation:** Test-writer follow-on — add a cross-SVTN denial sub-case to `TestListKeysAdmissionGate` (or equivalent) verifying that a caller admitted to SVTN-A receives E-ADM-009 when calling list-keys on SVTN-B.

---

### OBS-P5P13-B-002 [OBS] — E-CFG-001 token pattern consistency: other admin surface gaps may exist

**Description:** The F-P5P13-A-002 finding identified that the list-keys `usageErrf` at line 168 lacks the E-CFG-001 token. A sibling-sweep was not performed on the broader admin surface. Other `usageErrf` callsites for missing required flags (e.g., `--key`, `--role` on key register/revoke, `--name` on svtn destroy) may have the same gap. A targeted grep sweep for `usageErrf` calls that should emit E-CFG-001 but do not include the token string would close this.

**Recommendation:** Post-remediation sibling sweep — verify all admin `usageErrf` callsites that correspond to E-CFG-001 exit-code column entries carry the token.
