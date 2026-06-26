---
artifact_id: adv-S-3.02-pass-04
review_target: S-3.02-session-attach-detach-fanout
producer: adversary
pass: 4
fresh_context: true
branch: feature/S-3.02-session-attach-detach-fanout
base: develop @ 56ec9c7
tip: 94443db
findings_count: 23
findings_by_severity: {critical: 3, high: 6, medium: 8, low: 4, process_gap: 2}
verdict: NOT_CONVERGED
timestamp: 2026-06-26
note: Pass-3 fix burst (10 commits, 5 spec patches) addressed prior findings. Fresh-context surfaces new defects in clock-test correctness (F-C-1/F-C-2), channel-lifecycle asymmetry (F-C-3), sessionName validation (F-H-2), frame-drop silent failure (F-H-5), and TOCTOU race (F-H-6).
---

# Adversarial Review — Pass 4 — S-3.02

## Critical Findings

### F-C-1 — AC-008 test does not verify "remaining consoles continue unaffected"
**Files:** `internal/session/session_test.go:519-619`

AC-008 states: "When a console's channel closes unexpectedly … evicts the console from the fan-out set. **Remaining consoles continue unaffected.**" Test attaches two consoles (`crash-victim`, `survivor`), heartbeats `survivor`, calls `Sweep(-time.Second)`. With a negative deadline, `cutoff = time.Now().UTC().Add(1*time.Second)` (a future time) — every entry's `lastHeartbeat.Before(cutoff) == true` — BOTH consoles evicted. Test asserts `evicted == 2` and `survivorDownstream` is CLOSED. The selective-eviction property is structurally unverifiable with this approach.

### F-C-2 — `Sweep(-time.Second)` exploits an undocumented edge case; does NOT exercise production `now.Sub(lastHB) > deadline` semantics
**Files:** `internal/session/fanout.go:182-208`; `internal/session/session_test.go:589`; `internal/session/fanout_test.go:240`

Production callers pass POSITIVE deadlines (e.g., `30 * time.Second`). The test path with negative deadline makes every entry unconditionally Before(cutoff) — no heartbeat arithmetic exercised. A bug comparing the wrong field would be invisible since no fresh heartbeat is established. No test covers: heartbeated 100ms ago, sweep deadline=50ms → evicted; sweep deadline=200ms → survives.

### F-C-3 — `ConsoleSet.Remove` closes only `downstream`; `EvictStale` closes BOTH — asymmetric channel lifecycle
**Files:** `internal/session/fanout.go:111-124` (Remove); `:182-208` (EvictStale); `internal/session/upstream.go:204-225` (Detach doc)

Detach godoc says "ConsoleSet.Remove closes the downstream channel … and EvictStale **also** closes the upstream channel outside the lock." Two channels owned, two close points, one path closes both, one path closes one. Goroutine-leak vector for any caller draining `upstream` after Detach (the documented test-harness path).

## High Findings

### F-H-1 — Error format STILL mismatches error-taxonomy v1.3 E-SES-003 (Detach omits session_name; SendKeystroke uses "in" instead of "for")
**Files:** `internal/session/upstream.go:217-225` (Detach); `:248-258` (SendKeystroke); `.factory/specs/prd-supplements/error-taxonomy.md:129`

Taxonomy: `"session: console <id> not found **for** session <name>"`. Code: `"session: console %s not found"` (Detach, no session_name) and `"session: console %s not found **in** session %s"` (SendKeystroke). Story v1.3 patches claim alignment achieved; alignment incomplete.

### F-H-2 — `SendKeystroke` does NOT validate `sessionName` against the console's attached session
**Files:** `internal/session/upstream.go:248-266`

`SendKeystroke(key, sessionName, payload)` accepts sessionName, passes it to authorizer and error message, but never cross-references against the session `key` attached to. Wrong-session keystroke is silently forwarded.

### F-H-3 — 50-line block-comment in test signals undertested code path; chosen approach (`-time.Second`) bypasses actual EvictStale semantics
**Files:** `internal/session/session_test.go:540-589`

Implementer's confessional documents that without clock injection or sleeps the AC cannot be tested honestly. Chose option (c) `-time.Second` because it is hermetic; same option cannot demonstrate selective-eviction (F-C-1 / F-C-2). Test should either inject clock OR be deferred to system-test.

### F-H-4 — `TestControlMode_SendInput_HappyPath` uses `strings.Contains` to assert payload — does not disambiguate from `list-sessions` init write
**Files:** `internal/tmux/control_test.go:795-829`

`cm.Connect` writes `"list-sessions"` to stdin. Test then SendInputs `"hello\r"` and asserts `strings.Contains(string(got), "hello\r")`. A regression that routed SendInput's writes elsewhere would be invisible if the captured buffer still contained Connect's bytes.

### F-H-5 — `ConsoleSet.Deliver` silently drops frames on backpressure with no counter/metric — SOUL silent-failure anti-pattern
**Files:** `internal/session/fanout.go:136-148`

`select { case entry.downstream <- hdr: default: /* drop */ }`. If a console's 64-frame buffer fills, frames disappear without trace. BC-2.04.006 PC-1 fan-out completeness is violated for slow consumers. No counter; no metric; no test exercises the drop path. Comment cites NFR-004 — no NFR-004 in BC-2.04.006 v1.2.

### F-H-6 — `IsAttached` → `SendInput` TOCTOU race: in-flight keystroke forwards AFTER Detach returns
**Files:** `internal/session/upstream.go:248-266`; `:217-225`

T1: `SendKeystroke` calls IsAttached → true; T2: `Detach(key)` removes key; T1: acquires sinkMu, calls sink.SendInput. Keystroke forwards for detached console. BC-2.04.004 PC-3 ambiguity on "subsequent" — literal reading forbids; lenient reading permits. Fix: hold sinkMu during IsAttached check.

## Medium Findings

### F-M-1 — `recordingSink.Received()` returns shallow copy; payload bytes uncloned (Rule 12 hygiene)
**Files:** `internal/session/session_test.go:200-207`

### F-M-2 — ConsoleKey construction `"fan-console-"+string(rune('A'+i))` works only for i<26
**Files:** `internal/session/session_test.go:404,460,127`

### F-M-3 — `ConsoleSet.Deliver` comment references non-existent `Evict()` method
**Files:** `internal/session/fanout.go:144-145`

### F-M-4 — `Sweep` per-console count semantics undocumented (passes coverage but worth docstring tightening)
**Files:** `internal/session/upstream.go:280-282`

### F-M-5 — `noSink` value receiver vs `recordingSink` pointer receiver — KeystrokeSink interface contract should document receiver flexibility
**Files:** `internal/session/upstream.go:80-92`; `session_test.go:181-198`

### F-M-6 — `Heartbeat` value-copy-write-back is fragile against future `consoleEntry` field growth
**Files:** `internal/session/fanout.go:156-169`

### F-M-7 — `internal/session` imports `time` — ARCH-08 §6.5 wording could be read to exclude stdlib (verification artifact, not defect)
**Files:** various; ARCH-08-dependency-graph.md:252

### F-M-8 — `ErrSessionAlreadyPublished` and `ErrSessionNotFound` have inconsistent "session:" prefixes
**Files:** `internal/session/session.go:24-31`

## Low Findings

### F-L-1 — Wall-clock-dependent test deadlines (100ms..2s) — flake risk under CI load
**Files:** various

### F-L-2 — `controlModeFailureLogMsg` mixes `errors.Is` and `strings.Contains` — string-matching is dead-code for production but present (S-3.01b scope)
**Files:** `internal/tmux/pty_fallback.go:554-570`

### F-L-3 — Buffer-size divergence (256/64/16) across pipeline; no documented rationale
**Files:** `internal/tmux/control.go:233-235`; `internal/session/fanout.go:60-65`

### F-L-4 — Test uses Go 1.22+ `for i := range numConsoles` syntax (project go.mod is 1.25.4 — OK; verification artifact)
**Files:** `internal/session/session_test.go:403,459,473`

## Process-Gap Findings

### F-PG-1 — Adversarial-review template lacks "Boundary-Sentinel Inputs" axis [process-gap]

The pattern in F-C-1/F-C-2/F-H-3 — a test using out-of-band trick (negative deadline) to bypass wall-clock requirement — is exactly the false-coverage that "AC ↔ test fidelity" axis should catch mechanically. Suggested axis: classify each parameter value as inside/boundary/outside production range; out-of-range determinism tricks MUST have a separate in-range test with clock injection.

### F-PG-2 — Adversarial-review template lacks "Channel-Lifecycle Asymmetry" axis [process-gap]

F-C-3 — graceful path and exceptional path treating resource lifecycle differently — is a class of bug current axes don't explicitly target. Suggested axis: for any type owning multiple resources, every teardown method must teardown the same set in the same order.

## Verdict
NOT_CONVERGED — 3C/6H/8M/4L/2PG

## Novelty Assessment
MEDIUM-HIGH. New findings since pass-3: F-C-1 (selective-eviction unverifiable), F-C-2 (negative-deadline doesn't exercise positive-deadline semantics), F-C-3 (Remove/EvictStale lifecycle asymmetry), F-H-2 (sessionName validation gap), F-H-5 (silent frame drop), F-H-6 (TOCTOU), F-M-3 (dead comment). Remaining are refinements with new file:line citations.
