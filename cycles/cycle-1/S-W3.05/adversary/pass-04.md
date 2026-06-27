---
artifact_id: adv-S-W3.05-pass-04
review_target: S-W3.05-hmac-failure-counter-e-adm-017-alert
producer: adversary
pass: 4
lens: concurrency/memory
fresh_context: true
branch: feat/S-W3.05-hmac-failure-counter
impl_commit: 3af388c
findings_count: 8
findings_by_severity: {critical: 0, high: 1, medium: 2, low: 1, observations: 4}
verdict: NOT_CONVERGED
streak_after_pass: 0
streak_reset_reason: high and medium findings present
timestamp: 2026-06-27
note: Returned inline by adversary; persisted via state-manager.
---

# Adversarial Review — Pass 4 — S-W3.05

**Lens:** Concurrency/Memory
**Verdict:** NOT_CONVERGED
**Counts:** 0C / 1H / 2M / 1L / 4 VERIFIED CLEAN

---

## High Findings

### H-1 (HIGH) — E-ADM-017 message format missing mandatory "HMAC failure rate alert:" phrase

**Files:** `internal/admission/failure_counter.go:168-173`

Impl emits `"E-ADM-017 ≥%d failures in %.0fs from src %s"` — MISSING the mandatory `"HMAC failure rate alert:"` phrase. Canonical authorities ALL include it:
- `error-taxonomy.md:53` (v1.9)
- `BC-2.05.005:57` (PC-3) + `:107` (canonical test vector)
- `BC-2.05.008:86` (EC-006)

Tests `failure_counter_adversarial_test.go` AC-015 and `failure_counter_test.go` AC-003 were massaged to FORBID the phrase — tautological (test pins impl, both diverge from spec). Content defect; spec is authoritative; code + tests + story must conform.

**Confidence:** HIGH
**Owner:** implementer + test-writer

---

## Medium Findings

### M-1 (MED) — Per-source timestamp slice unbounded within window (CWE-770)

**File:** `internal/admission/failure_counter.go:159`

Appends one `time.Time` per failure; trimming only bounds to `rate×window`; `maxTrackedSources` caps source COUNT not per-source slice length. Single source at high rate → unbounded memory within window.

**Owner:** product-owner (adjudication: acceptable given rate constraint, or add per-source cap)

### M-2 (MED) — VP-059.md references fc.TrackedSourceCount(); impl provides SourceCount()

**Files:** `VP-059.md:238`, `VP-059.md:311`; `internal/admission/failure_counter.go:229`

VP harness won't compile as written. Content defect in VP-059.

**Owner:** implementer (align naming in VP harness or rename method)

---

## Low Findings

### L-1 (LOW) — AC-015 inline test comment falsely claims "HMAC failure rate alert:" not in taxonomy v1.9

Resolves with H-1: the phrase IS present at `error-taxonomy.md:53`. The comment is factually incorrect.

**Owner:** test-writer

---

## Verified Clean

- **O-1:** Lock discipline CORRECT — lock held through trim→evict→append→check→firedAt-write; `logger.Log` called after `Unlock`; no lock-during-IO.
- **O-2:** Source-count cap holds — 65537-distinct-source pattern triggers evict-before-insert, keeping `len ≤ 65536`.
- **O-3:** `evictLRU` correct; dead-key eviction keeps map free of empty slices; strict-less-than trim boundary (EC-008); strictly-after re-arm (keep[0].After(lastFire)).
- **O-4:** `Timestamps()` / `SourceCount()` return copies (go.md rule 12); constructor panics on `threshold<1` / `windowDuration<=0`; no goroutine leaks.
