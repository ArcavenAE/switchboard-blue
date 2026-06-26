---
artifact_id: adv-S-3.02-pass-03
review_target: S-3.02-session-attach-detach-fanout
producer: adversary
pass: 3
fresh_context: true
branch: feature/S-3.02-session-attach-detach-fanout
base: develop @ 56ec9c7
tip: 678b13d
findings_count: 26
findings_by_severity: {critical: 5, high: 7, medium: 6, low: 5, process_gap: 3}
verdict: NOT_CONVERGED
timestamp: 2026-06-26
note: Fresh-context post-arch-rework review. Surfaced test-tautology defects + production desync bug (F-C-4) induced by pass-2 EvictStale code.
---

# Adversarial Review — Pass 3 — S-3.02

## Critical Findings

### F-C-1 — AC-002 test is a tautology; upstream-channel write does not exercise keystroke forwarding
**Files:** `internal/session/session_test.go:185-202`; `internal/session/upstream.go:144-150,228-248`
**Confidence:** HIGH

`TestSession_Attach_UpstreamKeystrokesForwarded` claims to verify BC-2.04.003 PC-3 ("upstream half-channel accepted and forwarded to tmux"). Actually does: `upstream <- keystroke` (succeeds because chan is buffered 16) then `<-done`. Nothing on the receive side is checked. The upstream channel returned by `Attach` is **never drained by AccessNode**. The actual forwarding path is `SendKeystroke → sink.SendInput`, a completely different code path the test does not exercise. Passing test gives false confidence PC-3 holds.

### F-C-2 — AC-007 test does not exercise the spec-mandated serialization point; sink is unverified
**Files:** `internal/session/session_test.go:350-377,147-157`

`TestSession_ConcurrentKeystrokes_Serialized` traces to BC-2.04.006 Inv-3 (no keystroke interleaving). `AccessNode` is constructed by `newTestAccessNode` with no `WithKeystrokeSink` → defaults to `noOpSink{}` which does nothing. Test asserts only "no error" — if you deleted `sinkMu.Lock()` the test would still pass. Serialization invariant completely unverified.

### F-C-3 — AC-008 test simulates "crash" by calling Detach; the crash-detection code path is untested
**Files:** `internal/session/session_test.go:379-428`; `internal/session/fanout.go:170-229`; `internal/session/upstream.go:258-267`

Test docstring admits "we attach a console then call Detach to close its channel (equivalent to a crash)". This is wrong: graceful Detach is a different code path. Actual crash detection = Heartbeat + Sweep + EvictStale. `grep '\.Heartbeat(\|\.Sweep(\|\.EvictStale(' internal/` shows NO test calls any of these. Contract unvalidated.

### F-C-4 — `a.upstreams` desynchronizes from `ConsoleSet` after `EvictStale`, allowing forwarded keystrokes from evicted consoles
**Files:** `internal/session/upstream.go:228-248`; `internal/session/fanout.go:202-229`

`AccessNode.SendKeystroke` consults `a.upstreams` (not `cs.consoles`) to check attachment. `EvictStale` deletes from `cs.consoles` and closes channels but does NOT touch `a.upstreams`. After `Sweep` evicts stale console K:
- `cs.consoles[K]` → gone
- `a.upstreams[K]` → still present (closed channel)

Subsequent `SendKeystroke(K, …)` finds K, acquires sinkMu, calls `sink.SendInput(payload)` — forwards keystroke for an EVICTED console. Violates BC-2.04.004 PC-3. Direct write to the stale channel from `a.upstreams` would panic on closed channel.

### F-C-5 — `SendInput` methods on ControlMode, PTYProxy, SessionConnector have zero test coverage
**Files:** `internal/tmux/control.go:638-663`; `internal/tmux/pty_fallback.go:345-361,369-386`

`grep SendInput internal/tmux/*_test.go` → zero. These are the production-path bridge between `session.KeystrokeSink` and the tmux subprocess. AC-002/AC-007 ultimately verify on top of them. Untested.

## High Findings

### F-H-1 — `SessionConnector.SendInput` / `PTYProxy` errors are not wrapping sentinels
**Files:** `internal/tmux/pty_fallback.go:369-386,183,186`

Bare `fmt.Errorf("session connector: closed")` and `"PTY proxy: already closed"` cannot be inspected via `errors.Is`. go.md rule: errors.Is/As only, never string matching.

### F-H-2 — BC-2.04.004 PC-3 (no keystrokes forwarded after Detach) not directly verified
**Files:** `internal/session/session_test.go:218-255`; `internal/session/upstream.go:193-212`

`TestSession_Detach_SessionContinues` verifies downstream closure + session re-attachable. Does NOT call `SendKeystroke` post-Detach to verify rejection. Regression silent.

### F-H-3 — `TestConsoleSet_Evict_RemovesCrashedConsoles` does not test crashed-console removal
**Files:** `internal/session/fanout_test.go:200-232`

Tests `Evict()==0` on healthy set + `Len()==2`. Name promises crash testing; body delivers none.

### F-H-4 — Error message format diverges from error-taxonomy spec for E-SES-002 / E-SES-003
**Files:** `internal/session/fanout.go:17-22`; `internal/session/upstream.go:238-240`; error-taxonomy.md:128-129

Spec format: `"console <id> is already attached to session <name>"`. Code emits `"session: console already attached"` (no id, no name). `SendKeystroke` wraps with only the console key, not session name. Operator-facing diagnostics missing key context.

### F-H-5 — Direct writes to bidirectional upstream channel race with `Detach`'s close
**Files:** `internal/session/upstream.go:163-181`; `internal/session/fanout.go:85`

`ConsoleSet.Add` returns `chan []byte` (bidirectional). Attach narrows to `chan<- []byte` for caller. Detach calls `close(us)` outside `a.mu`. Concurrent sender on buffered channel can race with close → "send on closed channel" panic.

### F-H-6 — AC-005 test claims "read-only observer" but no role distinction exists
**Files:** `internal/session/session_test.go:257-294`; `internal/session/upstream.go` (no role concept)

Test verifies fan-out, not read-only-vs-full-access. BC-2.04.006 PC-3 (read-only keystrokes rejected) unimplemented + untested. Defer to S-3.03 (Tier-2 auth + read-only role).

### F-H-7 — Story BCs anchored to `ss-04/` per filesystem; review prompt referenced `ss-02/`. Audit needed [process-gap]
**Files:** `.factory/specs/behavioral-contracts/ss-04/BC-2.04.003.md` (actual location)

Subsystem numbering convention inconsistent. `ss-04 = session-access`. The orchestrator's prompt referenced `ss-02/`. Source: BC-INDEX or product-docs SS numbering. Audit upstream.

## Medium Findings

### F-M-1 — `evictQueue` slice in ConsoleSet is functionally an unused side-channel
**Files:** `internal/session/fanout.go:47-53,118-122,160-168,202-218`

Appended by Remove/EvictStale, drained by Evict (count discarded), called from DeliverFrame ignoring count. Dead state.

### F-M-2 — `consoleEntry` stored by value; Heartbeat write-back fragility
**Files:** `internal/session/fanout.go:33-37,176-189`

`consoles map[ConsoleKey]consoleEntry` — Heartbeat reads, mutates, writes back. Future maintainer omitting write-back silently corrupts. Either comment, refactor to `*consoleEntry`, or extract a mutation method.

### F-M-3 — `DeliverFrame` calls `Evict()` on every delivery — allocation churn on hot path
**Files:** `internal/session/upstream.go:250-256`; `internal/session/fanout.go:135-168`

Per-frame WLock + slice reset for a queue that carries no information. Pure overhead.

### F-M-4 — Story narrative claims SVTN multicast; implementation is in-process fan-out only
**Files:** `BC-2.04.006.md:50-51 (PC-2 "router fans out via SVTN multicast")`; `internal/session/fanout.go:135-147`

Access-node-side fan-out only. Router multicast deferred. BC trace should note PC-2 is partial. Defer router half to S-3.03 or later.

### F-M-5 — `WithKeystrokeSink` option defined but never used in any test
**Files:** `internal/session/upstream.go:80-87`

Public option with zero test usage. Combined with F-C-1/F-C-2, the wiring path has zero coverage.

### F-M-6 — BC postconditions PC-4 (advertisement attached=true/false), PC-5 (screen refresh) untested
**Files:** `BC-2.04.003.md:52-56`; `BC-2.04.004.md:52-54`

Story trace includes these PCs but story acceptance criteria do not assert them. Either exclude with rationale or implement. Defer to advertisement-update story (Wave 4+).

## Low Findings

### F-L-1 — TestSession_CrashDetach_EvictsFromFanOut docstring admits the simulation is wrong
**Files:** `internal/session/session_test.go:388-393`

Says "implementer must handle crash detection inside DeliverFrame via recover on send-to-closed" — but channels can only be closed from the SEND side, not receive. Doc is incoherent.

### F-L-2 — `noOpSink` default makes production callers silently discard keystrokes
**Files:** `internal/session/upstream.go:71-75,125-140`

`NewAccessNode` without `WithKeystrokeSink` → silent discard. SOUL anti-pattern (silent failure). Either require sink at construction or default to error-returning sink.

### F-L-3 — Send-only `chan<- []byte` already prevents close at compile time; docstring is partly redundant
**Files:** `internal/session/upstream.go:163-181`

Note for self-validation. No action.

### F-L-4 — Magic numbers documented but not configurable
**Files:** `internal/session/fanout.go:62-70`; `internal/tmux/control.go:233-235`

`downstreamBufSize=64`, `upstreamBufSize=16`, `framesBufferSize=256`. P0 fan-out target (50+ observers) warrants verification.

### F-L-5 — `noOpSink` (unexported) vs `NoOpAuthorizer` (exported) — minor public-surface asymmetry
**Files:** `internal/session/upstream.go:73-75`

OK in context. NoOpAuthorizer is allow-all useful API; noOpSink is test-default.

## Process-Gap Findings

### F-PG-1 — Story can declare 8 ACs and ship with multiple tautological tests passing [process-gap]
**Files:** S-3.02 story; F-C-1/F-C-2/F-C-3 above

Existing gate checks "all tests pass." Does not check that each AC's test actually asserts the BC postcondition. Need test-behavior policy at TDD-RG layer.

### F-PG-2 — KeystrokeSink interface + WithKeystrokeSink option added without BC anchor [process-gap]
**Files:** `internal/session/upstream.go:51-87`; BC-2.04.003 / BC-2.04.006

New public abstraction added to production code without corresponding spec patch. Spec-drift.

### F-PG-3 — Heartbeat/Sweep/EvictStale exported, undocumented at BC level, unreferenced [process-gap]
**Files:** `internal/session/fanout.go:170-229`; `internal/session/upstream.go:258-267`

Public surface with zero tests, zero callers, no BC anchor. Untested public API ships as a correctness defect (F-C-4).

## Verdict
NOT_CONVERGED — 5C/7H/6M/5L/3PG

## Novelty Assessment
HIGH. Findings centered on three layered issues: (1) AC test tautologies — tests named correctly, bodies vacuous; (2) F-C-4 production desync (NEW defect induced by pass-2 EvictStale code); (3) tmux SendInput zero coverage. Process-gap findings point at structural drift class. Likely novel vs prior passes.
