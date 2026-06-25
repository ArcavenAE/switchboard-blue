---
artifact_id: adv-S-2.01-pass-04
review_target: S-2.01-hmac-codec
producer: adversary
pass: 4
fresh_context: true
branch: feature/S-2.01-hmac-codec
base: develop @ 4be1b53
tip: 93959cb
findings_count: 1
findings_by_severity: {critical: 0, high: 0, medium: 1, low: 0, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-24
---

# Adversary Pass 4 — S-2.01 (HMAC codec)

## Medium

### F-001 — Story AC↔BC traces are systematically mis-anchored; AC-005 cites non-existent postcondition 4
- Location: `.factory/stories/S-2.01-hmac-codec.md:47-66` vs `.factory/specs/behavioral-contracts/ss-05/BC-2.05.005.md:50-54`
- Evidence: BC-2.05.005 defines exactly 3 postconditions:
  - pc1 — "HMAC verification succeeds: frame forwarded to destination"
  - pc2 — "HMAC verification fails: frame dropped; E-ADM-002 logged"
  - pc3 — "Repeated HMAC failures … trigger an admission alert"
  
  Story AC trace columns map incorrectly:
  
  | AC | Trace declared | Should anchor to |
  |----|---|---|
  | AC-001 (8-byte tag) | pc1 | preconditions §2 (defines tag = first 8 bytes of HMAC-SHA256) |
  | AC-002 (verify returns true) | pc2 | **pc1** (success branch) |
  | AC-003 (bit-flip rejection) | pc3 | **pc2** (failure branch) |
  | AC-004 (wrong-key rejection) | pre1 | pc2 or invariant DI-006 |
  | AC-005 (HKDF deterministic) | **"postcondition 4" — DOES NOT EXIST** | preconditions §2 (defines HKDF-SHA256 keying) |

- Impact: BC, code, and tests are all correct; only the story's AC→BC pointer column is wrong. Phase-5 traceability matrix tooling and any BC-change impact analysis will mis-route from these dangling/swapped pointers. The AC-005 → "postcondition 4" pointer is a dangling reference (BC has only 3 PCs).
- Route: product-owner
- Fix: Update story spec AC trace texts to anchor correctly:
  - AC-001 → BC-2.05.005 preconditions §2 (tag = first 8 bytes)
  - AC-002 → BC-2.05.005 pc1 (success branch)
  - AC-003 → BC-2.05.005 pc2 (failure branch on tamper)
  - AC-004 → BC-2.05.005 pc2 (failure semantics on wrong-key)
  - AC-005 → BC-2.05.005 preconditions §2 (HKDF-SHA256 keying definition)
  
  Drop the "postcondition 4" reference entirely.

## Observations (carry over from prior passes — all clean)

- KAT verified: RFC 4231 §4.2 (HMAC) + RFC 5869 §A.1 (HKDF), both byte-correct.
- Constant-time compare verified (`crypto/hmac.Equal`).
- ARCH-08/09 compliant; no secret leakage.
- 5 ACs + 3 VPs + 3 ECs all covered.
- Dual-fuzz coverage (frame-payload + tag-bit, all 64 positions).
- HKDF multi-iter expand works through T(2) for L=42 RFC §A.1 vector.
- Distinctness tests pin per-(node, SVTN) forge-resistance.
- HKDF length guard prevents pathological internal-caller misuse.

## Convergence

Streak reset to 0/3. Pass 5 follows PO story AC-trace fix.

This finding is novel (passes 1-3 did not catch it) — fresh-context BC↔story side-by-side reading surfaced the trace mis-anchoring. Per the Semantic Anchoring Audit rubric: mis-anchoring always blocks convergence regardless of code correctness.
