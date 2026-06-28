---
story: S-4.01
pass: 2
reviewed_commit: cce35a3
verdict: NOT_CONVERGED
critical: 0
high: 2
date: 2026-06-27
---

# S-4.01 Adversarial Review — Pass 2

VERDICT: NOT_CONVERGED 0C/2H (streak reset to 0/3; Pass-1 critical/high fixes all held — no regressions)

## CRITICAL

none

## HIGH

- F-H1 — spec-conformance/dead-code — multipath.go:188,200,297-305 — Multipath.dropCache (compound-key router loop-suppression per BC-2.02.009) is constructed but NEVER read/written by any Multipath method; there is no router OnFrameArrival/Forward method. BC-2.02.009 router behavior is unreachable through the orchestrator (only the DropCache unit type is tested). Receive also ignores its arrival_interface_id param (`_ uint64`). Either wire the router path or remove dead field + explicit deferral.
- F-H2 — spec-conformance (missing postcondition) — multipath.go (DropCache) — BC-2.02.009 postcondition 2 mandates a drop-cache HIT COUNTER (operator diagnostics) and EC-005 mandates collision-event logging. Neither DropCache nor Multipath exposes a hit counter or collision hook. Operators cannot detect loop-storms (EC-002/FM-003) or collisions (EC-005).

## MEDIUM

- F-M1 — paths.go:122-128 — reactivation branch and first-probe branch duplicate "set RTT outright" logic with no shared invariant; firstProbe flag and !active flag encode overlapping "use raw RTT" semantics — maintenance trap (benign now).
- F-M2 — multipath.go:243-266 — Send does not implement BC-2.02.001 EC-003/postcondition-4 queue-with-timeout (E-NET-002); returns error immediately/no queue. Defensible pure-core simplification BUT no explicit deferral note in the story for the queue+E-NET-002.
- F-M3 — multipath_test.go:263-294 — no test drives concurrent Send vs UpdatePaths on shared pathSet; the m.mu lock protecting pathSet is never race-tested (snapshot test mutates single-threaded inside fn). Lock could be dropped and all tests still pass.
- F-M4 — multipath.go:297 — Receive(f, _ uint64) advertises an unused interface-id param (YAGNI/go.md); remove it or (if router forwarding is wired per F-H1) actually use it.

## LOW / OBSERVATIONS

- F-L1 — paths.go:84-92 — NewPathTracker doesn't validate alpha ∈ (0,1] despite doc precondition (garbage-in/garbage-out; alpha=0 freezes EWMA).
- F-L2 — multipath.go:71-77 — NewDropCache doesn't validate capacity>=1; capacity=0 degenerate (pins at 1 entry, violates contract).
- F-L3 — [process-gap] — story:141 names test TestDropCache_KeyedOnChecksumAndInterface but actual is TestBC_2_02_009_DropCache_KeyedOnChecksumAndInterface — story Enforcement-column test name won't match a grep (traceability automation gap).
- F-L4 — multipath.go:294-296 — VP-054 "no ACK side-effects" cannot be verified in this pure-core package; deferred to integration harness — note for wave-gate VP-054 unsatisfied by S-4.01 alone.

## PROCESS GAPS

- F-L3 tagged [process-gap] (story test-name traceability vs grep).
