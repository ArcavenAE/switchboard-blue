---
artifact_id: adv-S-3.01a-pass-10
review_target: S-3.01a-tmux-control-mode
producer: adversary
pass: 10
fresh_context: true
branch: feature/S-3.01a-tmux-control-mode
base: develop @ d54bf1a
tip: 675705f
findings_count: 1
findings_by_severity: {critical: 0, high: 0, medium: 0, low: 1, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-26
note: Returned inline by adversary because tool profile is read-only; persisted by orchestrator via state-manager.
---

# Adversarial Review — Pass 10 — S-3.01a

## Critical Findings
None.

## High Findings
None.

## Medium Findings
None.

## Low Findings

### F-PASS10-L-001 — NewPublisher/Publisher docstring claims admission gating that code does not implement

**Files:** `internal/session/session.go:43-48` (Publisher struct doc), `internal/session/session.go:55-64` (NewPublisher godoc)

Publisher struct doc at lines 43-48 claims: "manages the set of published tmux sessions and **gates publication against the admission key set** (BC-2.04.001 PC-2; ARCH-08 §6.6 position 6)."

NewPublisher godoc at lines 55-57 claims: "constructs a Publisher that checks publication admission using keys (BC-2.04.001 precondition 3; S-3.01a task 5)."

`Publisher.Publish` (lines 70-84) NEVER reads `p.keys`. No code path consults the AdmittedKeySet during Publish/Unpublish.

Additionally: the cited anchor BC-2.04.001 precondition 3 reads "access node is admitted to an SVTN (or admission in progress)" — this is about the access node's own SVTN admission, not per-session key gating. Per-session admission gating is S-3.03 SessionAuth (Tier 2) territory.

The keys field IS intentional forward infrastructure (session_test.go:17-25 says "available for S-3.03 admission-gated publish tests"). The godoc should say so explicitly.

Confidence: HIGH. Severity: LOW.

## Resolution decision

Fix the docstrings to honestly describe current behavior + forward-scoped role of keys.

## Verification Notes (no defects on these axes)

- ARCH-08 §6.6 imports compliant: session={admission} ⊆ {frame, admission}; tmux={halfchannel, session} exact.
- ARCH-09 classifications match (session boundary, tmux effectful).
- All 4 ACs have hermetic tests.
- Subprocess reaped via cmd.Wait() goroutine.
- Scanner buffer raised to 2 MiB; oversized payloads fragmented to MaxPayloadSize chunks.
- Close() joins dispatchLoop via sync.WaitGroup.
- sync.Once correctly guards closeErrCh and closeFrames against double-close.
- Backpressure protected via non-blocking select on c.frames.
- ErrAlreadyConnected enforces Connect idempotency.
- Octal unescape handles truncated/non-octal escapes safely.
- No internal-pointer leaks from ListSessions (returns value copies).
- time.Now().UTC() used; no init(); no panics in library code.

## Novelty Assessment

Novelty: LOW. Single docstring inaccuracy — same class as pass-6 F-LOW-01 (stale docstring promising unimplemented behavior). Implementation is functionally converged; only doc-level inaccuracies remain.

## Streak status

Pass 1: NOT_CONVERGED (7 findings)
Pass 2: NOT_CONVERGED (5 findings)
Pass 3: NOT_CONVERGED (2 findings)
Pass 4: CONVERGED (0 findings)
Pass 5: NOT_CONVERGED (4 findings)
Pass 6: NOT_CONVERGED (1 finding)
Pass 7: NOT_CONVERGED (3 findings)
Pass 8: CONVERGED (0 findings)
Pass 9: CONVERGED (0 findings)
Pass 10: NOT_CONVERGED (1 finding: 1L)

**Streak: 0/3 toward BC-5.39.001.**
