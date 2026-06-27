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
