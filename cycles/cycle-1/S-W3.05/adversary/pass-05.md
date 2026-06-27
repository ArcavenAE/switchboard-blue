---
artifact_id: adv-S-W3.05-pass-05
review_target: S-W3.05-hmac-failure-counter-e-adm-017-alert
producer: adversary
pass: 5
lens: spec-conformance/test-quality
fresh_context: true
branch: feat/S-W3.05-hmac-failure-counter
impl_commit: 3af388c
findings_count: 8
findings_by_severity: {critical: 2, high: 1, medium: 0, low: 3, observations: 2}
verdict: NOT_CONVERGED
streak_after_pass: 0
streak_reset_reason: critical findings present
timestamp: 2026-06-27
note: Returned inline by adversary; persisted via state-manager.
---

# Adversarial Review — Pass 5 — S-W3.05

**Lens:** Spec-Conformance / Test-Quality
**Verdict:** NOT_CONVERGED
**Counts:** 2C / 1H / 0M / 3L / 2 VERIFIED DISCRIMINATING

---

## Critical Findings

### C-1 (CRITICAL) — E-ADM-017 message format tautology: code + tests diverge from all four spec authorities

**Files:** `internal/admission/failure_counter.go:168-173`, `failure_counter_adversarial_test.go` AC-015, `failure_counter_test.go` AC-003

Code + AC-015 + AC-003 agree against four spec authorities (`error-taxonomy.md:53` v1.9, `BC-2.05.005:57` PC-3, `BC-2.05.005:107` canonical test vector, `BC-2.05.008:86` EC-006). AC-015 test comment makes a factually-false claim about error-taxonomy v1.9 — the phrase "HMAC failure rate alert:" IS present at `:53`. Test pins impl, not spec. Content defect; specs are authoritative.

**Owner:** implementer + test-writer

### C-2 (CRITICAL) — VP-059 proof_method=proptest: no property-based test exists

**Files:** `VP-059.md:17`, `:61`, `:64`; `internal/admission/`

`VP-059.md` specifies `proof_method: proptest` — stateful model checker over arbitrary call sequences, properties a–e. No property-based test harness exists (`testing/quick`, `rapid`, or stateful generators absent from `internal/admission`). `TestFailureCounter_PropertiesABCD` / `_PropertyE_MemoryBound` harnesses were never ported. Proof method UNMET. Content defect; blocks convergence.

**Owner:** test-writer (port VP-059 proptest harness)

---

## High Findings

### H-1 (HIGH) — AC-012 dead-key drain-to-zero delete path has no discriminating test

**File:** `internal/admission/failure_counter.go:132-136`; `failure_counter_adversarial_test.go:294-324`

`TestFailureCounter_DeadKeyEvictedAfterDrain` author-admits it cannot observe `SourceCount()` drop — final assertion only checks `SourceCount()>=1`, true regardless. An impl that skipped the `delete(counts, srcAddr)` call would still pass. Mandated path untested. Content defect.

**Owner:** test-writer

---

## Low Findings

### O-1 (LOW) — "ERROR level" (PC-3 :57, AC-003) unsatisfiable through admission.Logger

`Logger.Log(msg string)` has no severity parameter. "ERROR level" constraint is unenforceable at this interface.

**Owner:** product-owner (adjudicate: document as arch limitation or add severity param)

### O-2 (LOW) — VP-059 TrackedSourceCount vs SourceCount naming mismatch

Duplicate of pass-04 M-2. VP harness won't compile.

**Owner:** implementer

### O-3 (LOW) — EC-005 canonical scenario uses 6-failure first batch; tests use only 5-then-5

No test exercises the 6-failure first-batch scenario from EC-005.

**Owner:** test-writer

---

## Verified Discriminating

- **AC-004:** Re-arm boundary test (threshold=1, no re-arm at entry==firedAt, re-arm at strictly-after) genuinely pins `>` vs `>=`. Discriminating.
- **pass-04 O-1..O-4:** Lock discipline, source-count cap, evict/trim boundary, copy-return — all verified CLEAN in prior pass; not re-examined here.
