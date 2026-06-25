---
artifact_id: adv-S-1.03-pass-03
review_target: S-1.03-node-identity-session-continuity
producer: adversary
pass: 3
fresh_context: true
branch: feature/S-1.03-node-identity-session-continuity
base: develop @ a06b306
tip: 7a4a6c5
findings_count: 0
findings_by_severity: {critical: 0, high: 0, medium: 0, low: 0, nitpick: 0}
verdict: CONVERGED
timestamp: 2026-06-25
note: Returned inline by adversary because tool profile is read-only; persisted by orchestrator via state-manager.
---

# Adversarial Review — Pass 3 — S-1.03

## Critical Findings
None.

## Important Findings
None.

## Observations
None.

## Audit Notes (verification record)

- **A. AC↔BC↔test trace correctness:** AC-001 → BC-2.01.007 PC3+PC4 (story line 50, test docstring reauth_test.go:31-32). AC-002 → Pre3 (story 54, test 121). AC-003 → Inv3 (story 58, test 170). All aligned with BC v1.3.
- **B. Error code correctness:** ErrKeyExpired → E-ADM-015 (reauth.go:22-25); taxonomy line 64 confirms; BC EC-005 line 81, test trace line 233. Consistent.
- **C/D. Lock discipline & locked-accessor contract:** Step 1 RLock snapshot, release; Step 3 write lock re-checks admitted/revoked/Expiry; explicit unlock before rs.setSourceAddr to avoid double-lock (reauth.go:158-160, 186-190). CurrentSourceAddr returns netip.Addr by value (immutable). No internal pointer leak.
- **E/F. UTC + constant-time:** time.Now().UTC() at reauth.go:147; ed25519.Verify (stdlib constant-time).
- **G. State machine / LWW reset:** ADR-003 amended LWW reset (admitted=false on RegisterKey) is enforced via Step 3 liveEntry.admitted re-check at line 163. Re-auth never sets admitted=true (correct — node already admitted).
- **L. Nonce ordering:** Nonce recorded before sig verify (lines 177→182), mirroring AdmitNode (admission.go:337→344). BC-2.05.001 invariant 3 honored.
- **M. EC-006 anchoring:** Old-path eviction anchored to BC-2.01.007 EC-006 in code (reauth.go:69, 107, 117, 189), tests (reauth_test.go:287, 436), and BC body (line 82).
- **N. Step numbering coherent:** Steps 1 (126), 2 (146), 3 (152), 4 (188) — sequential.
- **J. VP-036 deferral:** t.Skip at reauth_test.go:541-546 with grep-discoverable docstring.
- **I. No init / no globals:** confirmed.
- **K. Spec drift:** Story v1.3 / BC v1.3 / ARCH-04 v1.3 mutually consistent. Taxonomy E-ADM-015 source field references BC-2.01.007. EC-001 (story) ↔ EC-005 (BC) ↔ E-ADM-015 (taxonomy) ↔ test all aligned.

## Novelty Assessment

Novelty: LOW — no findings; specs and implementation fully consistent.

## Streak status

Pass 1: NOT_CONVERGED (6 findings: 2H/3M/1L)
Pass 2: NOT_CONVERGED (3 findings: 2M/1L)
Pass 3: CONVERGED (0 findings)

**Streak: 1/3 toward BC-5.39.001.**
