---
artifact_id: adv-S-3.01a-pass-05
review_target: S-3.01a-tmux-control-mode
producer: adversary
pass: 5
fresh_context: true
branch: feature/S-3.01a-tmux-control-mode
base: develop @ d54bf1a
tip: 73de969
findings_count: 4
findings_by_severity: {critical: 0, high: 1, medium: 2, low: 1, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-26
note: Returned inline by adversary because tool profile is read-only; persisted by orchestrator via state-manager.
---

# Adversarial Review — Pass 5 — S-3.01a

## Critical Findings
None.

## High Findings

### H-1 — bufio.Scanner default token limit (64KB) causes false-drop on large %output bursts

**File:** `internal/tmux/control.go:271`

`scanner := bufio.NewScanner(r)` — no `scanner.Buffer(...)` call. Default `MaxScanTokenSize = 64 * 1024`. Terminal apps can dump arbitrary-size output in one write (cat largefile, ANSI cursor bursts, large paste). When a single `%output` line exceeds 64KB, `scanner.Scan()` returns false; `scanner.Err()` returns `bufio.ErrTooLong`.

Post-loop logic at lines 318-342 cannot distinguish `ErrTooLong` from EOF — falls through to default arm (line 325), signals `ErrControlModeDropped`, exits dispatchLoop. `scanner.Err()` never inspected.

Per ADR-010 v1.2: PTY mode is sticky for connection lifetime → single large burst permanently degrades the access node. Violates BC-2.04.001 PC-5; 99% delivery target in AC-004 unattainable under realistic workloads.

Confidence: HIGH. Severity: HIGH.

## Medium Findings

### M-1 — Silent drop of %output payloads when Enqueue fails

**File:** `internal/tmux/control.go:371-372`

```go
data := unescapeTmuxOutput(parts[1])
_ = c.downstream.Enqueue(data)
c.downstream.Tick()
```

`halfchannel.Enqueue` returns `ErrPayloadTooLarge` when `len(data) > MaxPayloadSize` (65515 bytes). Error silently discarded; Tick still fires (empty-tick frame downstream). Combined with H-1, ≤65KB lines with octal-escape expansion may still exceed MaxPayloadSize.

Violates BC-2.04.001 PC-5 silently with no observability. Conflicts with go.md rule 3 (always check errors) and SOUL.md #4 (silent failures).

### M-2 — Close does not join dispatchLoop goroutine; lifecycle contract gap

**File:** `internal/tmux/control.go:227-253`

Close cancels ctx, closes stdin/stdout/errCh (via sync.Once), returns. No `sync.WaitGroup` join. After Close returns, dispatchLoop may still be in `scanner.Scan()` or post-loop logic, calling `c.publisher.Publish/Unpublish`.

Publisher is locked (data race mitigated by internal/session) but lifecycle race remains. Contract on ControlMode (line 56-57): "ControlMode spawns exactly one internal goroutine when Connect succeeds." Close offers no completion semantics — callers cannot reason about when it's safe to drop publisher/downstream references.

Current consumer pattern (Err() then Close) sidesteps. Future `Close(); pub=nil` is unsafe.

## Low Findings

### L-1 — session.go package comment narrows allowed imports vs ARCH-08

**File:** `internal/session/session.go:9-11`

```go
// Allowed internal imports: {admission} per ARCH-08 §6.6 (frame import removed
// when FrameTypeData re-export was deleted — no remaining consumer).
```

ARCH-08 §6.6 authoritatively declares `internal/session | {frame, admission}`. Package comment narrows to `{admission}` based on implementation state, not architecture spec. Future maintainer adding `frame` import would be misled into thinking they've violated the contract when in fact architecture permits it.

## Observations

- ErrAlreadyConnected idempotency guard is correct defensive design not specified by BC-2.04.001.
- unescapeTmuxOutput correctly handles \NNN octal and \\ literal-backslash escapes; unrecognized-escape pass-through conservative but documented.
- Test hermeticity well-enforced — all tests use fakeExecFunc; VP-031 (real-tmux) correctly deferred.
- Race detector status: lifecycle gap M-2 means future `cm.Close(); pub = nil` is unsafe. Current consumer pattern sidesteps.

## Novelty Assessment

Novelty: MEDIUM. H-1 (scanner buffer) and M-1 (silent Enqueue drop) are concrete payload-size defects with operational consequences. M-2 (Close lifecycle) is a contract gap affecting downstream consumer correctness. L-1 is documentation drift.

These are real operational defects against realistic tmux workloads — pass 4 missed them entirely, demonstrating the value of multi-pass fresh-context review.

## Resolution decisions (from human review)

- H-1: scanner.Buffer 2 MiB cap; inspect scanner.Err() post-loop to distinguish ErrTooLong (WARN log, continue) from EOF.
- M-1: fragment payload into <=MaxPayloadSize chunks; each chunk gets own Enqueue+Tick. Preserves PC-5 delivery.
- M-2: add sync.WaitGroup; Close waits for dispatchLoop to exit. Documents "after Close, dispatchLoop exited and downstream no longer accessed."
- L-1: restore {frame, admission} in comment with parenthetical "current code imports only admission; frame is permitted but unused."
