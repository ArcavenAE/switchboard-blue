---
artifact_id: adv-S-W3.05-pass-06
review_target: S-W3.05-hmac-failure-counter-e-adm-017-alert
producer: adversary
pass: 6
lens: integration/wiring
fresh_context: true
branch: feat/S-W3.05-hmac-failure-counter
impl_commit: 3af388c
findings_count: 1
findings_by_severity: {critical: 0, high: 1, medium: 0, low: 0, observations: 0}
verdict: NOT_CONVERGED
streak_after_pass: 0
streak_reset_reason: high finding present
timestamp: 2026-06-27
note: Returned inline by adversary; persisted via state-manager.
---

# Adversarial Review — Pass 6 — S-W3.05

**Lens:** Integration / Wiring
**Verdict:** NOT_CONVERGED
**Counts:** 0C / 1H / 0M / 0L

---

## High Findings

### F-1 (HIGH) — E-ADM-017 message format contradiction: spec authoritative, impl non-conforming

**Files:** `internal/admission/failure_counter.go:168-173`

Spec requires "HMAC failure rate alert:" prefix per error-taxonomy.md (v1.9), BC-2.05.005 PC-3, and BC-2.05.008 EC-006. Impl omits it. On strict reading impl is non-conforming.

**Confidence:** HIGH
**Owner:** implementer + test-writer

---

## Verified Clean (Extensive Integration Checks)

- **Interface seam:** `hmacFailureRecorder` unexported, consumer-defined, no import cycle — `admission` imports only `fmt`/`sync`/`time`. CLEAN.
- **Nil-guard:** `failureCounter` defaults nil, both call sites guard, nil path returns `ErrHMACVerificationFailed` cleanly. CLEAN.
- **RecordHMACFailure call sites:** Called on BOTH failure paths (PATH-A `routing.go:205`, PATH-B `:220`), NOT on success. CLEAN.
- **Exactly-once per failed frame:** PATH-A returns before PATH-B reachable — no double-counting. CLEAN.
- **SrcAddr encoding:** Lowercase-hex `%x` consistent end-to-end into counter + message. CLEAN.
- **Lock ordering:** Router `RLock` released `:194` BEFORE logging/`RecordHMACFailure`; `FailureCounter.mu` released before `logger.Log`; two locks never nested — no deadlock, no lock-during-IO. CLEAN.
- **EC-006 end-to-end:** 5 failures → exactly 1 E-ADM-017 (sole blocker is message-text alignment, not fire count). CLEAN.
- **Trim boundary, LRU cap, constructor panics:** CLEAN.
- **Concurrency design under -race:** Sound. CLEAN.
- **ARCH-08 position-4 import constraint:** Holds. CLEAN.

---

## Consolidated Summary (Passes 4–6)

Streak resets to 0. Blocking (C/H findings): E-ADM-017 message-format (3/3 passes), missing VP-059 proptest harness (C-2 pass-05), AC-012 dead-key `delete(counts)` untested (H-1 pass-05).

Non-blocking (PO/adjudication): M-1 per-source slice unbounded within window (CWE-770, PO), O-1 ERROR-level vs level-less Logger (PO), M-2/O-2 VP-059 `TrackedSourceCount` vs `SourceCount` naming, O-3 EC-005 6-failure vector.
