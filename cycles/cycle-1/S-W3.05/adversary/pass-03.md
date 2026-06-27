---
artifact_id: adv-S-W3.05-pass-03
review_target: S-W3.05-hmac-failure-counter-e-adm-017-alert
producer: adversary
pass: 3
lens: test-quality-integration-api-design
fresh_context: true
branch: feat/S-W3.05-hmac-failure-counter
impl_commit: 1e74d76
findings_count: 11
findings_by_severity: {critical: 0, high: 3, medium: 3, low: 3, observations: 2}
verdict: NOT_CONVERGED
streak_after_pass: 0
streak_reset_reason: high findings present
timestamp: 2026-06-27
note: Returned inline by adversary; persisted via state-manager.
---

# Adversarial Review — Pass 3 — S-W3.05

**Lens:** Test-Quality + Integration + API Design
**Verdict:** NOT_CONVERGED
**Counts:** 0C / 3H / 3M / 3L / 2 OBSERVATION

---

## High Findings

### H-1 (HIGH) — Confirmed tautology: re-arm rule derived to pass tests, contradicts AC-004 literal

**Files:** `internal/admission/failure_counter.go`, `internal/admission/failure_counter_test.go`

CONFIRMED from pass-1 C1. The re-arm rule was derived to pass the existing tests, yet it contradicts AC-004 literal ("drops below threshold"). AC-005 asserts only `Count==2` at end of sequence, never WHICH call fires — therefore the derived rule AND a mutant both pass. The tests are tautological with respect to re-arm semantics.

**Owner:** product-owner → test-writer

### H-2 (HIGH) — Re-arm off-by-one not caught by test suite

**File:** `internal/admission/failure_counter.go:109`

Mutating the re-arm condition from `keep[0].After(lastFire)` to `!keep[0].Before(lastFire)` (i.e., `>=`) still yields exactly 2 alerts in AC-005. The re-arm boundary case (entry timestamp == fire timestamp) is untested. The mutation survives.

**Owner:** test-writer (add boundary discriminator test)

### H-3 (HIGH) [process-gap] — VP-059 missing spec file AND no proptest exists

VP-059 is the story's named P0 verification property (cited in `BC-2.05.005:111` and `BC-2.05.008:105` as proptest). VP-059.md does NOT exist (VP-058 and VP-060 exist; VP-059 is absent from the filesystem). Additionally, no property-based test exists — all tests are example-based. The property that would defend H-1 and H-2 is missing at both the spec and code levels.

**Owner:** architect (author VP-059.md) + test-writer/implementer (proptest)

---

## Medium Findings

### M-1 (MED) — Integration test does not pin fire-once end-to-end

**File:** `internal/routing/routing_hmac_counter_test.go:378-437`

The integration test is genuine end-to-end (real counter + Router) but asserts only `Count==1` after exactly 5 frames. It never drives a 6th or 7th frame through `RouteFrame` to confirm fire-once semantics end-to-end, nor does it drive 0–4 to confirm no spurious fire.

**Owner:** test-writer

### M-2 (MED) — WithFailureCounter uses unexported interface, not spec-pinned `*admission.FailureCounter`

**File:** `internal/routing/routing.go:35-62`

`WithFailureCounter` accepts unexported interface `hmacFailureRecorder` rather than the spec-pinned `*admission.FailureCounter` (BC-2.05.008 EC-006 + AC-009 + Red-Gate contract). Structurally sound for S-W3.04 daemon wiring (the real type satisfies the interface), idiomatic Go, but diverges from the pinned seam and relaxes the guarantee — any `RecordHMACFailure(string)` implementation is injectable.

**Owner:** product-owner — ratify or revert

### M-3 (MED) — `srcAddr` key format not pinned by BC

**File:** `internal/routing/routing.go:205,220`

`srcAddr` key `fmt.Sprintf("%x", hdr.SrcAddr)` is an implementer-chosen encoding, never pinned by BC. E-ADM-017 shows hex in its message, but the canonical `<src_addr>` rendering is unspecified.

**Owner:** product-owner — define canonical `<src_addr>` rendering

---

## Low Findings

- **L-1:** E-ADM-017 format vs canonical taxonomy (duplicate of pass-1 I2 / pass-2 O-6).
- **L-2:** AC-001 `TestNewFailureCounter_ConstructorFields` over-claims — only checks `!=nil` + 0 logs; cannot see internal fields (black-box). A swapped threshold/window constructor argument would pass.
- **L-3:** AC-009 success subtest tolerates any non-HMAC error; does not assert `err==nil`/forward. However, it DOES correctly pin PC-5 negative: a spurious `RecordHMACFailure` on success WOULD be caught. Net: weak positive claim, adequate negative claim.

---

## Observations

- **O-1:** `keep := existing[:0]` in-place trim is safe — entry overwritten; `Timestamps()` copies. No aliasing hazard.
- **O-2:** Import-cycle avoidance correct — `admission` defines its own `Logger` interface; no `routing` import in `admission`.
