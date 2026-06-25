---
artifact_id: adv-S-2.01-pass-09
review_target: S-2.01-hmac-codec
producer: adversary
pass: 9
fresh_context: true
branch: feature/S-2.01-hmac-codec
base: develop @ 4be1b53
tip: 9a1ef34
findings_count: 1
findings_by_severity: {critical: 0, high: 0, medium: 0, low: 1, nitpick: 0}
verdict_adversary_self: CONVERGED (downgraded Observation O-1 to non-blocking)
verdict_orchestrator: NOT_CONVERGED (user 2026-06-24 elected strict re-classification of soft-flag as LOW finding; consistent with pass-4 F-001 + pass-7 F-001 prior choices)
timestamp: 2026-06-24
---

# Adversary Pass 9 — S-2.01 (HMAC codec)

## Low

### F-001 (reclassified from adversary's "Observation O-1") — Story File Structure Requirements table missing hkdf_internal_test.go
- Location: `.factory/stories/S-2.01-hmac-codec.md:145-147` (File Structure Requirements table); the file `.worktrees/S-2.01/internal/hmac/hkdf_internal_test.go` exists and is the canonical home for `TestDeriveKey_RFC5869_KAT`.
- Evidence: File Structure table lists 3 files (`hmac.go`, `hmac_test.go`, `fuzz_test.go`); AC-005 amended note (line 66) mandates the KAT in `hkdf_internal_test.go` when inline HKDF is taken. The KAT was extracted to a new internal-package test file (pass-2 fix), but the File Structure table was not updated.
- Impact: Story-spec coherence gap. An implementer reading File Structure Requirements would think only 3 files are needed; reading AC-005 they'd discover the 4th. Two clauses of the same story spec disagree.
- Route: product-owner
- Fix: Add a row to the File Structure Requirements table: `| internal/hmac/hkdf_internal_test.go | create | RFC 5869 §A.1 known-answer test for inline HKDF via unexported hkdfSHA256 helper (AC-005, ARCH-04 v1.1 KAT mandate). |`.
- Note: adversary classified as "Observation O-1" / non-blocking; orchestrator+user re-classified per strict BC-5.39.001 zero-findings rule (consistent with pass-4 F-001 and pass-7 F-001 prior choices).

## Observations (carry over)

All other audit axes clean: HMAC + HKDF KATs pinned, constant-time compare, type-safe API, purity, ARCH-08 leaf invariant, AC traces correct (post pass-4), VP-004/005/006 covered, no key-material leakage, forge-resistance proven.

## Convergence

Streak reset to 0/3. Pass 10 follows PO File Structure table patch.
