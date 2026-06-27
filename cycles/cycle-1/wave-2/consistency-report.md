---
artifact_id: wave-2-consistency-report
document_type: consistency-validation-report
level: ops
version: "1.0"
producer: consistency-validator
timestamp: 2026-06-25T00:00:00
wave: 2
develop_tip: f35e836
stories_validated: [S-2.01, S-2.02, S-1.03]
---

# Wave 2 Consistency Validation Report

**Wave:** 2 — Security Foundation + Session Continuity  
**Stories closed:** S-2.01 (PR #5, 3c4104e), S-2.02 (PR #6, a06b306), S-1.03 (PR #7, f35e836)  
**Develop tip:** f35e8363ebf4ac8119e7edc3358d22bc0c76e885 (short: f35e836)  
**Validated:** 2026-06-25  
**Verdict:** PASS_WITH_OBSERVATIONS

---

## Verification Matrix

| Check | Category | Result | Notes |
|-------|----------|--------|-------|
| 1. Cross-story traceability | BC-INDEX + STORY-INDEX | PASS | All 6 target BCs `implemented`; all 3 stories `completed` |
| 2. Error taxonomy consistency | error-taxonomy.md + code sentinels | PASS with observation | All named sentinels present; ErrNoForwardingEntry missing taxonomy row (observation) |
| 3. ARCH-08 dependency graph integrity | import graph | PASS | All import constraints satisfied |
| 4. ARCH-09 purity boundary classification | purity violations | PASS | admission is classified `boundary` (time.Now correct); hmac is clean |
| 5. ADR-003 LWW reset semantic | RegisterKey + ReAuthenticate | PASS | LWW confirmed; un-revoke path confirmed |
| 6. Concurrent test coverage | race detector tests | PASS | TestAdmitNodeRevokeKey_NoRace + TestReAuthenticate_NoRace present |
| 7. Drift register reality check | STATE.md | PASS with observation | VP-036 + SEC-003 properly recorded; ARCH-08 not bumped to v1.1 |
| 8. Cross-story spec version check | BC/ARCH/VP frontmatter | MEDIUM finding | ARCH-08 is v1.0 but routing.go comment cites v1.1 |
| 9. Demo evidence integrity | cycle directories | MEDIUM finding | S-2.02 + S-1.03 missing per-AC evidence files in standard location |
| 10. CI/tooling sanity | fmt + lint + test -race | PASS | All green: 0 lint issues, 6/6 packages pass race |

---

## Findings by Severity

### CRITICAL (0 findings)

None. No findings that would block a PR.

---

### HIGH (0 findings)

None. No semantic drift requiring pre-Wave-3 fixes.

---

### MEDIUM (2 findings)

#### MEDIUM-001 — ARCH-08 version frozen at v1.0 despite routing.go self-citing v1.1

**Location:** `.factory/specs/architecture/ARCH-08-dependency-graph.md` frontmatter; `internal/routing/routing.go` line 8  
**Observed:**  
- ARCH-08-dependency-graph.md has `version: "1.0"`  
- routing.go package comment reads: `// Import constraints (ARCH-08 §6): this package MAY import internal/frame, internal/hmac, and internal/admission only. No upward imports.` — the ARCH-08 doc does not have a §6 header in v1.0, though the content is present under "Boundary Violation Rules"  
- ARCH-09 correctly bumped to v1.1 (Wave-1 burst fix); ARCH-08 was not explicitly bumped  

**Impact:** Cross-reference integrity weakened. If a new contributor reads `ARCH-08 §6` per routing.go's cite, they land on a section that does not exist under that label in the v1.0 doc.  
**Expected:** ARCH-08 should be at v1.1 with a §6 explicitly named "Import Constraints Per Package" or the routing.go comment updated to match the actual section header.  
**Routing:** Architect — bump ARCH-08 to v1.1 with a §6 heading, or update routing.go comment to cite existing section title.

---

#### MEDIUM-002 — S-2.02 and S-1.03 per-AC demo evidence files missing from standard location

**Location:** `.factory/cycles/cycle-1/S-2.02/` and `.factory/cycles/cycle-1/S-1.03/`  
**Observed:**  
- S-2.01 has `.factory/cycles/cycle-1/S-2.01/demo-evidence/per-ac-evidence.md` — standard location present  
- S-2.02 has only `implementation/` (red-gate-log) and `adversary/` subdirs — no `demo-evidence/per-ac-evidence.md`  
- S-1.03 has only `adversary/` — no `demo-evidence/per-ac-evidence.md`  
**Impact:** If Wave 2 closure artifacts are audited, S-2.02 and S-1.03 will fail the evidence-file check. The STATE.md and STORY-INDEX both list these as `completed`, but there is no per-AC demo artifact on disk.  
**Note:** Demo evidence may have been captured inline in adversary pass files or as godoc Example functions in source. The godoc Examples are present in code. This is a traceability gap, not a correctness gap.  
**Routing:** Demo-recorder or state-manager — create placeholder per-AC evidence stubs for S-2.02 and S-1.03 pointing to godoc Example functions and adversary pass logs.

---

### LOW (3 findings)

#### LOW-001 — ErrNoForwardingEntry has no error-taxonomy row

**Location:** `internal/routing/routing.go:26`; `.factory/specs/prd-supplements/error-taxonomy.md`  
**Observed:** `ErrNoForwardingEntry = errors.New("routing: no forwarding entry for destination in this SVTN")` is a live sentinel in the routing package, but error-taxonomy.md has no corresponding row. The FWD category only has E-FWD-001 (split-horizon drop).  
**Impact:** Operators/test-writers have no taxonomy reference for diagnosing forwarding-table misses. The error code omission means any future scenario doc that references this error will have no canonical ID.  
**Routing:** Product-owner — add E-FWD-002 (or similar) for forwarding-table miss, citing routing.go:ErrNoForwardingEntry.

---

#### LOW-002 — E-ADM-014 (bootstrap key mismatch) not wired into any implementation code

**Location:** `.factory/specs/prd-supplements/error-taxonomy.md` E-ADM-014; `internal/admission/`  
**Observed:** E-ADM-014 `"bootstrap key mismatch: provided key does not match SVTN <svtn_id> bootstrap"` is defined in the taxonomy and traced to ADR-004. Grepping all non-test Go files in `internal/admission/` finds no reference to E-ADM-014, no `ErrBootstrapKeyMismatch` sentinel, and no `sbctl admin recover` code path.  
**Impact:** This is expected — the `svtnmgmt` package (S-6.02, Wave 5) is the implementation home. The taxonomy row is correctly marked as a future contract. No fix required before Wave 3.  
**Classification:** LOW (pre-emptive traceability note, not a defect). The row is a valid forward declaration.

---

#### LOW-003 — VP-007/008/009/010/057 still carry `lifecycle_status: active` (not `implemented` or `deferred`)

**Location:** `.factory/specs/verification-properties/VP-007.md` through VP-010.md, VP-057.md  
**Observed:** All five VPs backing the S-2.02/S-1.03 behavioral contracts have `status: draft` and `lifecycle_status: active`. Now that S-2.02 (BC-2.05.001/002/006/007) and S-1.03 (BC-2.01.007) are merged with passing unit tests, these VPs should transition to `lifecycle_status: implemented` (or remain `active/draft` until Phase-6 formal hardening completes, per the VSDD workflow).  
**Impact:** VP-INDEX row commentary is unclear about whether these are "under test" (active) or "test written, passing" (implemented). This matters for Phase-4 holdout evaluation scoping.  
**Routing:** Spec-steward or state-manager — decide and consistently apply the `lifecycle_status` transition policy post-story-merge.

---

### OBSERVATION (4 items)

#### OBS-001 — VP-036 deferred status properly documented and placeholder present

VP-036.md is v1.1 with `lifecycle_status: deferred`, `deferred_to: phase-6-hardening`. The test file `internal/admission/reauth_test.go` has `TestProperty_VP036_SessionContinuity` with a proper `t.Skip` carrying the grep-discoverable cite `// VP-036 deferred to Phase-6: requires testenv.ConnectWithSourceIP`. The property statement in the spec is unchanged. Deferral is fully traceable. No action needed.

#### OBS-002 — SEC-003 TOCTOU disposition correctly recorded

SEC-003 (sub-microsecond TOCTOU on `now` in `ReAuthenticate` — `now := time.Now().UTC()` captured outside the write lock) is documented in STATE.md as accepted, tracked alongside VP-036 for Phase-6 hardening. The admission boundary classification (ARCH-09 `boundary` bucket) explains why `time.Now()` is present. No action needed.

#### OBS-003 — ARCH-08 + ARCH-09 session footnote correctly cite `internal/session` as shared with cmd/sbctl

ARCH-08 "Notes on Deliberate Coupling" explains that `internal/session` is imported by both `internal/tmux` and `cmd/sbctl`. The actual import graph for implemented packages does not yet include `internal/session` (it is future Wave 3), so there is no actual violation to detect. Confirmed: zero imports of `internal/session` currently exist.

#### OBS-004 — `just fmt` runs gofumpt cleanly with no diff output (no reformatting needed)

`just fmt` (gofumpt -w .) completes without producing diff output, confirming all committed code is properly formatted. `just lint` exits 0 with "0 issues." `go test ./... -race -count=1` passes all 6 packages with no data races detected.

---

## Detailed Check Results

### Check 1: Cross-story traceability

**BC-INDEX.md status for Wave 2 BCs:**

| BC | Expected Status | Actual Status | PR Cite | Result |
|----|----------------|---------------|---------|--------|
| BC-2.05.005 | implemented (S-2.01) | `implemented (S-2.01 / PR #5)` | PR #5 | PASS |
| BC-2.05.001 | implemented (S-2.02) | `implemented (S-2.02 / PR #6)` | PR #6 | PASS |
| BC-2.05.002 | implemented (S-2.02) | `implemented (S-2.02 / PR #6)` | PR #6 | PASS |
| BC-2.05.006 | implemented (S-2.02) | `implemented (S-2.02 / PR #6)` | PR #6 | PASS |
| BC-2.05.007 | implemented (S-2.02) | `implemented (S-2.02 / PR #6)` | PR #6 | PASS |
| BC-2.01.007 | implemented (S-1.03) | `implemented (S-1.03 / PR #7)` | PR #7 | PASS |

**STORY-INDEX.md status:**

| Story | Expected | Actual | Result |
|-------|----------|--------|--------|
| S-2.01 | completed | `completed (PR #5, merge 3c4104e)` | PASS |
| S-2.02 | completed | `completed (PR #6, merge a06b306)` | PASS |
| S-1.03 | completed | `completed (PR #7, merge f35e836)` | PASS |

Summary count: Complete = 6 (S-0.01, S-1.01, S-1.02, S-2.01, S-2.02, S-1.03) — matches Wave 2 closure.

### Check 2: Error taxonomy consistency

**Named sentinel → taxonomy ID mapping:**

| Sentinel | File | Expected ID | Taxonomy Row | Result |
|----------|------|-------------|-------------|--------|
| `ErrSignatureVerificationFailed` | admission/admission.go:28 | E-ADM-001 | Present | PASS |
| `ErrKeyRevoked` | admission/admission.go:32 | E-ADM-005 | Present | PASS |
| `ErrNonceReplay` | admission/admission.go:36 | E-ADM-008 | Present | PASS |
| `ErrNotAdmitted` | admission/admission.go:41 | E-ADM-003 | Present | PASS |
| `ErrKeyNotRegistered` | admission/admission.go:48 | E-ADM-013 | Present | PASS |
| `ErrKeyExpired` | admission/reauth.go:25 | E-ADM-015 | Present (S-1.03 minted) | PASS |
| `ErrNoForwardingEntry` | routing/routing.go:26 | (none) | MISSING — see LOW-001 | OBSERVATION |
| E-ADM-014 | taxonomy | ADR-004 | Row present, no code impl | LOW-002 (expected) |

**No orphan rows found in error-taxonomy.md.** All error code rows have traceable BCs or FM references.

### Check 3: ARCH-08 dependency graph integrity

**Actual imports (non-test .go files):**

| Package | Imports (internal) | Expected (ARCH-08) | Result |
|---------|-------------------|--------------------|--------|
| `internal/admission` | frame, hmac | frame, hmac | PASS |
| `internal/routing` | admission, frame, hmac | frame, hmac, admission | PASS |
| `internal/hmac` | (none internal) | frame only | PASS — hmac imports nothing internal |
| `internal/halfchannel` | frame | frame | PASS |
| `internal/frame` | (none internal) | nothing internal | PASS |

Note: ARCH-08 spec says `internal/hmac` imports `frame`. The code has `internal/hmac` with zero internal imports. This is consistent with `internal/hmac` being a leaf — `frame` is only needed if hmac uses `DeriveNodeAddress`, which it does not. The spec DAG may be conservative. No import cycle; no forbidden edges. **Net result: PASS.**

### Check 4: ARCH-09 purity boundary classification

**`internal/hmac/` purity check:**
- Zero occurrences of `time.Now`, `os.`, or `init()` in non-test .go files. PASS — pure-core confirmed.

**`internal/admission/` purity check:**
- Two occurrences of `time.Now().UTC()`:
  - `admission.go:326` — inside `AdmitNode`
  - `reauth.go:153` — inside `ReAuthenticate`
- ARCH-09 classifies `internal/admission` as **boundary** (not pure-core). `time.Now()` usage is correct and expected in a boundary package. PASS.

### Check 5: ADR-003 LWW reset semantic

**`RegisterKey` LWW behavior:** `admission.go:155-175` — creates a fresh `AdmittedKey` entry (all fields zeroed, `revoked: false`) and overwrites the map entry under Lock. A post-revoke `RegisterKey` produces a new entry with `revoked=false`, clearing the revoked flag. LWW un-revoke confirmed.

**`ReAuthenticate` LWW interaction:** `reauth.go` — re-fetches the live entry under write lock at L173 to detect concurrent `RegisterKey` replacement. Comments cite ADR-003 LWW explicitly. Confirmed correct.

**Test coverage:** `TestRegisterKey_AfterRevoke_ClearsRevokedFlag` at admission_test.go:511 pins the ADR-003 LWW un-revoke path. Test present and passes under `-race`.

### Check 6: Concurrent test coverage

**Race-annotated tests:**

| Test | File | Description |
|------|------|-------------|
| `TestAdmitNodeRevokeKey_NoRace` | admission_test.go:470 | H-1 regression: concurrent AdmitNode + RevokeKey |
| `TestReAuthenticate_NoRace` | reauth_test.go:439 | Concurrent re-authentication from two goroutines |
| `TestRegisterKey_AfterRevoke_ClearsRevokedFlag` | admission_test.go:511 | ADR-003 LWW un-revoke pin |

All three tests pass under `go test -race`. No `-race` flag usage inside test files themselves (tests are designed to be run by `just test-race` or `go test -race ./...`). This is correct — test files do not embed `-race` directives.

### Check 7: Drift register reality check

**STATE.md open drift items verified:**

| Item | Severity | Status in STATE.md | Correctness |
|------|----------|--------------------|-------------|
| F-P8-004 | MED | open — Phase 3 test-writing for BC-2.02.003 | Consistent |
| F-P8-005 | MED | open — Phase 3 test-writing | Consistent |
| F-P8-009 | LOW | open — Phase 2 deferred | Consistent |
| F-003 | LOW | deferred to outer-assembler story (S-BL.OA) | Consistent |
| F-004 | LOW | deferred to outer-assembler story (S-BL.OA) | Consistent |
| VP-036 testenv | Phase-6 hardening | deferred, placeholder in code | Consistent with VP-036.md v1.1 |
| SEC-003 | Phase-6 hardening | accepted, track with VP-036 | Consistent |

No stale or missing drift items. ARCH-08 not having a v1.1 bump is an oversight noted in MEDIUM-001, not present as a drift item yet.

### Check 8: Cross-story spec patches version check

| Spec File | Expected Version | Actual Version | Result |
|-----------|-----------------|----------------|--------|
| BC-2.01.007.md | v1.3 | `"1.3"` | PASS |
| BC-2.05.001.md | v1.0 | `"1.1"` | PASS (v1.1 ≥ v1.0, implemented status correct) |
| BC-2.05.002.md | v1.0 | `"1.1"` | PASS (v1.1 ≥ v1.0) |
| BC-2.05.005.md | v1.0 | `"1.1"` | PASS (v1.1 ≥ v1.0) |
| BC-2.05.006.md | v1.0 | `"1.1"` | PASS (v1.1 ≥ v1.0) |
| BC-2.05.007.md | v1.0 | `"1.1"` | PASS (v1.1 ≥ v1.0) |
| ARCH-04.md | v1.3 | `"1.3"` | PASS |
| ARCH-08.md | v1.1 | `"1.0"` | FAIL → MEDIUM-001 |
| ARCH-09.md | v1.1 | `"1.1"` | PASS |
| VP-036.md | v1.1 with deferred status | v1.1, `lifecycle_status: deferred` | PASS |
| error-taxonomy.md | E-ADM-015 row present | Present (FM-013, BC-2.01.007) | PASS |

Note: All BC-2.05.* files are at v1.1, which supersedes the expected v1.0 — this is expected given the burst-fix that updated these specs during Wave 2 delivery.

### Check 9: Demo evidence integrity

| Story | Evidence Location | Status |
|-------|------------------|--------|
| S-2.01 | `.factory/cycles/cycle-1/S-2.01/demo-evidence/per-ac-evidence.md` | EXISTS |
| S-2.02 | `.factory/cycles/cycle-1/S-2.02/demo-evidence/per-ac-evidence.md` | MISSING → MEDIUM-002 |
| S-1.03 | `.factory/cycles/cycle-1/S-1.03/demo-evidence/per-ac-evidence.md` | MISSING → MEDIUM-002 |

S-2.02 has adversary passes 1–8 and a red-gate-log. S-1.03 has adversary passes 1–5. Both stories have godoc `Example*` functions in source (`ExampleAdmitNode_*` etc. for S-2.02, similar for S-1.03). The per-AC evidence artifact linking these together is missing.

### Check 10: CI/tooling sanity

| Tool | Result | Notes |
|------|--------|-------|
| `just fmt` (gofumpt) | PASS — no reformatting | No diff produced |
| `just lint` (golangci-lint) | PASS — 0 issues | Clean exit |
| `go test ./... -race -count=1` | PASS — 6/6 packages | All pass with race detector |

Package breakdown:
- `cmd/switchboard`: ok (1.485s)
- `internal/admission`: ok (3.023s)
- `internal/frame`: ok (1.739s)
- `internal/halfchannel`: ok (1.995s)
- `internal/hmac`: ok (2.561s)
- `internal/routing`: ok (2.842s)

---

## Summary

### Findings Count by Severity

| Severity | Count |
|----------|-------|
| CRITICAL | 0 |
| HIGH | 0 |
| MEDIUM | 2 |
| LOW | 3 |
| OBSERVATION | 4 |
| **Total** | **9** |

### Verdict

**PASS_WITH_OBSERVATIONS**

The Wave 2 security foundation (HMAC frame authentication, Tier-1 admission, SVTN isolation, session continuity) is internally consistent. All targeted behavioral contracts are correctly marked `implemented` in the index. The dependency graph is acyclic and satisfies ARCH-08. The purity boundary map is correct. The error taxonomy is complete for all implemented sentinels. All tests pass with the race detector. No CRITICAL or HIGH findings.

### Items Requiring Orchestrator Routing

| Finding ID | Action Required | Route To |
|-----------|----------------|----------|
| MEDIUM-001 | Bump ARCH-08 to v1.1 with explicit §6 heading, or update routing.go cite | architect |
| MEDIUM-002 | Create per-AC evidence stubs for S-2.02 and S-1.03 | demo-recorder or state-manager |
| LOW-001 | Add E-FWD-002 (forwarding-table miss) to error-taxonomy.md | product-owner (can defer to S-4.01 wave) |
| LOW-003 | Define and apply `lifecycle_status: implemented` transition policy for VP-007/008/009/010/057 | spec-steward |
