---
artifact_id: adv-S-W3.05-pass-02
review_target: S-W3.05-hmac-failure-counter-e-adm-017-alert
producer: adversary
pass: 2
lens: concurrency-security-resource-exhaustion
fresh_context: true
branch: feat/S-W3.05-hmac-failure-counter
impl_commit: 1e74d76
findings_count: 9
findings_by_severity: {critical: 0, high: 2, medium: 0, low: 0, observations: 7}
verdict: NOT_CONVERGED
streak_after_pass: 0
streak_reset_reason: high findings present
timestamp: 2026-06-27
note: Returned inline by adversary; persisted via state-manager.
---

# Adversarial Review — Pass 2 — S-W3.05

**Lens:** Concurrency + Security + Resource-Exhaustion
**Verdict:** NOT_CONVERGED
**Counts:** 0C / 2H / 0M / 0L / 7 OBSERVATION

---

## High Findings

### F-1 (HIGH) [process-gap on spec side] — Unbounded attacker-keyed map memory DoS (CWE-770)

**Files:** `internal/admission/failure_counter.go:86-130`, `internal/routing/routing.go:205,220`

`counts` and `firedAt` maps are keyed by attacker-controlled 8-byte `SrcAddr` (`fmt.Sprintf("%x", hdr.SrcAddr)` from wire header). Map keys are NEVER deleted:

- Line 96: `keep := existing[:0]` — in-place slice trim, NOT map delete
- Line 117: unconditional append; line 118: re-store → entry always survives with ≥1 element
- No `delete(counts, ...)` path anywhere
- `firedAt` only deleted on re-arm, not on eviction/quiescence

An attacker spoofing distinct `SrcAddr` values (2^64 keyspace) at 1 frame each creates unbounded monotonic map growth. The detector amplifies the flood it is meant to detect.

BC-2.05.005 PC-3 "lazy eviction" covers only per-source slice trimming, NOT dead-key removal from the map. The spec does not cap or bound map cardinality.

**Owner:** product-owner (define eviction/cap policy + amend PC-3) → implementer

### F-2 (HIGH) — No test bounds memory or exercises many-distinct-sources path

AC-010 `-race` detects data races only; it cannot catch the logical "never-shrinks" defect. The test suite is GREEN while F-1 is open.

**Owner:** test-writer (after PO defines eviction policy)

---

## Observations

- **O-1:** Router `RLock` correctly RELEASED before `logger.Log` AND `RecordHMACFailure` calls (`routing.go:194` RUnlock before lines 199/214 log, 205/220 counter). E-ADM-016 lock-during-log stall class NOT regressed; fix held. Secondary note: `FailureCounter.mu` IS held during `logger.Log` (`failure_counter.go:123`) — lower severity (fires at most once per crossing per source), recorded as OBSERVATION not finding.
- **O-2:** go.md rule 12 CLEAN — no internal pointer leakage from locked accessor.
- **O-3:** Sliding-window atomicity CLEAN — trim + rearm + append + check all occur under one `Lock`.
- **O-4:** `now` func() injection thread-safe — set once at construction, read under `mu`.
- **O-5:** `threshold <= 0` unguarded — first failure fires immediately; no constructor validation. Minor (operator-set not attacker-set). Recorded as OBSERVATION.
- **O-6:** E-ADM-017 message prepends code prefix vs taxonomy format (matches pass-1 I2).
- **O-7 [process-gap]:** Implementation uses `firedAt map[string]time.Time` rather than the spec-suggested `fired map[string]bool`; more correct for hysteresis but AC-004 prose now mis-describes the shipped mechanism. **Owner:** product-owner reconcile.
