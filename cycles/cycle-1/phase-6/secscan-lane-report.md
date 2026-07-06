---
artifact_id: P6-SECSCAN-LANE-1
document_type: phase-6-evidence
phase: phase-6-formal-hardening
lane: security-scan
producer: formal-verifier (phase6-secscan)
coordinator: team-lead
timestamp: 2026-07-06T02:45:00Z
develop_head_at_scan: 18fd2fe
status: complete
---

# Phase 6 — Security Scan Lane Report (Burst 1)

## Scope

Full-repo scan at develop 18fd2fe: govulncheck + gosec + focused manual pass on
security-sensitive seams + bounded manual mutation sampling (5 mutants × 3 packages).

## govulncheck

- **GO-2026-4971** (stdlib): reachable; Windows-only NUL-byte panic. Verdict:
  **LOW / ACCEPTED-RISK** — deployment targets are macOS/Linux only.
- 2 package-level + 8 module-level vulns: **unreachable** (informational).

## gosec

12 findings, zero true positives:
- G115 (int conversion overflow): FALSE-POSITIVE — bounded upstream.
- G304 (file inclusion): ACCEPTED-RISK — operator-supplied paths (sbctl --config/--key are operator trust surface by design).
- G204 (subprocess): ACCEPTED-RISK — trusted PATH lookup.
- G103 (unsafe): ACCEPTED-RISK — required POSIX pty API.

## Focused manual pass (all clean)

- HMAC verification: constant-time compare confirmed; ADR-009 v1.6 verify-then-lookup ordering confirmed.
- S601-SEC-001 (CWE-117) mitigation confirmed present with **no sibling injection points** (Detail interpolation sweep).
- sbctl key handling: no key material in logs/errors.
- CWE-400/770 allocation and failure-cap bounds present (UpstreamRoutersFailureCap et al.).

## Mutation sampling (manual, 5 × 3 packages)

| Package | Killed | Survived | Invalid | Kill rate |
|---------|--------|----------|---------|-----------|
| internal/netingress (1–5) | 3 | **2 (real test gaps)** | 0 | 3/5 |
| internal/routing (6–10) | 4 | 0 | 1 equivalent-mutant | 4/4 valid |
| internal/admission (11–15) | 4 | 0 | **1 (mutation 15 — lane collision)** | 4/4 valid |

**Surviving mutants → routed as tests into PR #105 (fuzz lane):**
1. MaxFrameBytes wire-bound not asserted (CWE-400 attacker-sized allocation).
2. MaxConcurrentConnections semaphore-shedding cap not asserted.

**Mutation 15** (E-ADM-017 re-arm block → `_ = lastFire`): re-verified in an
isolated worktree (develop @ 18fd2fe) — **SURVIVED, adjudicated PROVEN DEAD
CODE, not a test gap.** All five SW305-M4 fire-once/re-arm/property tests pass
with the mutation applied because Step 3's re-arm effect is fully subsumed by
Step 2's dead-key eviction (failure_counter.go:136-140 deletes firedAt on
window drain before Step 3 reads it). Coordinator independently verified the
remaining path: Step-4 evictLRU also deletes from BOTH maps (lines 215-216),
and Step 6 always pairs firedAt-set with a counts write — so
firedAt-without-counts is unreachable and no reachable state lets Step 3
change observable behavior. SW305-M4's guard is real; Step 3 is
belt-and-suspenders to a belt Step 2 already wears. Final admission table:
4/5 killed + 1 proven-dead-code. Overall lane: **11/15 killed, 2 real gaps
(→ PR #105 tests), 1 equivalent mutant, 1 dead code.** Follow-up cleanup
(delete Step 3 + document Step 2 as THE drain-only re-arm mechanism +
BC-2.05.005 EC-011 wording alignment) recorded as drift row
DRIFT-P6-ADM-STEP3-DEADCODE.

**Diagnostic-only observation (LOW):** routing PATH-A vs PATH-B error paths
share a log-message shape without a discriminator; hinders triage, no
security impact. Recorded here; no story anchor yet.

## Incident: concurrent-lane mutation collision (coordinator lessons entry)

Coordinator dispatched fuzz + secscan lanes concurrently in the SAME checkout;
secscan's live mutation samples (inverted HMAC verify at routing.go:283 +
gutted E-ADM-017 re-arm) were encountered by the fuzz lane, flagged as foreign
security-relevant modifications, and reverted. Reconciliation: tree verified
pristine; damage bounded to mutation-15's measurement via the
Edit-revert-succeeded discriminator (revert old_string matching mutated state
proves the test observed the mutation). **Lesson: mutation sampling MUST run
in an isolated worktree/clone; coordinator dispatch prompts for concurrent
lanes sharing a checkout must forbid working-tree source mutation.**
Session-fixed → factory-internal record only, no upstream filing
(vsdd-factory has no mutation-lane guidance to be defective — candidate
enhancement only if a phase-6 mutation workflow lands upstream).

## Verdict

No CRITICAL/HIGH findings. No fix PR required. Phase 6 security-scan lane
evidence: CLEAN with two MEDIUM test gaps routed to PR #105 and one
measurement pending isolated re-verify.
