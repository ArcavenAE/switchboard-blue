---
document_type: lessons-learned
level: ops
version: "1.0"
status: in-progress
producer: state-manager
timestamp: 2026-06-27T00:00:00Z
cycle: cycle-1
inputs: [STATE.md]
traces_to: STATE.md
---

# Lessons Learned — cycle-1

## Process-Level

1. **Adversarial lenses must include explicit nil-safety / panic-path sweep for constructor-injected dependencies** — S-W3.05 passes 07-09 achieved CONVERGED at 5c3d7ea, but all three lenses (spec-conformance/anti-tautology, concurrency/memory-bounds, integration/RouteFrame) missed a reachable nil-logger deref panic in `NewFailureCounter` (CWE-476 / SEC-001, HIGH). The finding was caught by the security-reviewer during PR #16 review — not by the per-story adversary passes. Root cause: none of the three lenses explicitly targeted constructor precondition completeness or nil-safety of injected dependencies. The streak reset required three additional passes (10-12) at the fixed tip f6038d2. Going forward, at least one adversary pass per story should apply a dedicated nil-safety / panic-path lens covering every constructor-injected parameter.
   _Discovered: PR #16 security review, 2026-06-27. Streak reset: passes 07-09 superseded; re-converged at passes 10-12._

## Policy Candidates

| Lesson | Proposed Policy | Scope | Status |
|--------|----------------|-------|--------|
| 1 | Require nil-safety / panic-path adversary lens for constructor-injected dependencies | Per-story adversarial review lens selection | proposed |

## Phase-5 Deferred Items

### TaskList #117 — ARCH-04 + error-taxonomy modified-list monotonicity (routed 2026-06-30)

**Source:** S-6.06 Pass-26, lens-3 observations O-P26L3-001 + O-P26L3-002. Adjudicated out-of-perimeter for S-6.06 per-story scope per BC-5.39.002 PC2 (architectural / system-level).

**O-P26L3-001:** ARCH-04.md lines 30-40 modified-list non-monotonic — missing version entries v1.7, v1.8, v1.11, v1.12 and v1.13 appears before v1.9 in the list.

**O-P26L3-002:** error-taxonomy.md lines 9-23 modified-list mixed ascending/descending ordering.

**Action:** Phase-5 adversarial refinement should sweep ARCH-04 and error-taxonomy modified-list ordering as part of the spec consistency pass. Both documents have accumulated version entries out of order through successive narrowing fix-bursts. Recommend architect agent sorts modified-list entries by version number ascending in both files during the next spec-evolution burst.
