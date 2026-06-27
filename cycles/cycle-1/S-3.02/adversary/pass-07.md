---
artifact_id: adv-S-3.02-pass-07
review_target: S-3.02-session-attach-detach-fanout
producer: adversary
pass: 7
fresh_context: true
branch: feature/S-3.02-session-attach-detach-fanout
base: develop @ 1bde664
tip: d857bf2
findings_count: 5
findings_by_severity: {critical: 0, high: 0, medium: 2, low: 3}
verdict: CONVERGED
timestamp: 2026-06-26
note: Fresh-context pass-7. Zero critical, zero high. F-M-1 vestigial upstream channel (deferred to S-3.03 — drift item). F-M-2 missing multi-console continuity test + F-L-3 untested ErrNoKeystrokeSink default — both fixed pass-7 (commits 2fa5738, 4017192). Streak 2/3 at this pass.
---

# Adversarial Review — Pass 7 — S-3.02

## Critical Findings
None.

## High Findings
None.

## Medium Findings

### F-M-1 (MEDIUM) — Upstream channel returned by Attach is vestigial in production
upstream.go:203-219 Attach returns upstream chan<- []byte, but AccessNode never drains it; SendKeystroke (upstream.go:267-292) forwards payload directly to a.sink.SendInput and never reads the upstream channel. The "CLOSE-RACE CONTRACT" (fanout.go:99-105, upstream.go:182-189) guards a channel with no production writer. Divergence between returned API surface and actual data path realizing BC-2.04.003 PC-3. DISPOSITION: deferred to S-3.03 (draining consumer) / next spec touch — tracked as drift item. Not blocking (MEDIUM).

### F-M-2 (MEDIUM) — Missing multi-console fan-out continuity test (NFR-004/EC-005)
fanout.go:186-200 drops on full buffer and increments aggregate framesDropped. Existing test (fanout_test.go:278-299) used ONE console, so could not prove that a draining sibling console still receives all frames while another stalls. FIXED pass-7: TestConsoleSet_Deliver_StalledConsoleDoesNotBlockOthers (commit 2fa5738).

## Low Findings

### F-L-1 (LOW) — EvictStale computes cutoff via nowFn() before acquiring lock
fanout.go:235. Benign — cutoff reflects clock at call entry (correct keepalive semantic). Correct as-is; load-bearing lock boundary, do not "tidy" inside lock.

### F-L-2 (LOW) — Heartbeat does value-copy-then-reassign on consoleEntry
fanout.go:208-221. Intentional (value map avoids internal-pointer-leak per go.md rule 12). Correct as-is; do not "optimize" to pointer map.

### F-L-3 (LOW) [process-gap] — Untested fail-loud ErrNoKeystrokeSink default
upstream.go:81-86 noSink{} returns ErrNoKeystrokeSink but no test proved it fires. Recurring "guard shipped without a test that proves it fires" anti-pattern. FIXED pass-7: TestAccessNode_SendKeystroke_NoSink_ReturnsError (commit 4017192).

## Confirmed Invariants
RWMutex Deliver(RLock full-loop) mutually exclusive with Remove/EvictStale(WLock close+delete) — no close-during-send. SendKeystroke takes sinkMu before consoles.Session() — TOCTOU closed. AC-007 contentionSink test mutation-sensitive. Detach signature includes sessionName. session pkg does not import tmux (ARCH-08 import direction). WithClock mutates in-place. AC-008 fake-clock eviction deterministic.

## Novelty Assessment
Novelty LOW. Zero critical/high. Two MEDIUMs are new framings (API-vs-datapath gap; NFR-004 test coverage gap), three LOWs are coverage/maintenance observations. Production concurrency core correct and well-tested.

## Verdict
CONVERGED (zero critical AND zero high).
