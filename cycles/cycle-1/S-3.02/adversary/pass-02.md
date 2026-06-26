---
artifact_id: adv-S-3.02-pass-02
review_target: S-3.02-session-attach-detach-fanout
producer: adversary
pass: 2
fresh_context: true
branch: feature/S-3.02-session-attach-detach-fanout
base: develop @ 56ec9c7
tip: 0740642
findings_count: 7
findings_by_severity: {critical: 2, high: 3, medium: 2, low: 0, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-26
note: Returned inline by adversary; persisted via state-manager.
---

# Adversarial Review — Pass 2 — S-3.02

## Critical Findings

### F-C-001 — Send-on-closed-channel panic: SendKeystroke vs Detach race

**Files:** `internal/session/upstream.go:185-216` (SendKeystroke); `:154-173` (Detach)

SendKeystroke reads us from map under a.mu (line 191-193), releases a.mu, acquires a.upstreamMu (line 201), executes `case us <- payload` (line 209). Detach acquires a.mu (line 161), deletes from map, releases a.mu, calls close(us) (line 169) — NEVER acquires a.upstreamMu.

Race: T1 SendKeystroke reads us, unlocks a.mu. T2 Detach locks a.mu, deletes, unlocks, close(us). T1 locks a.upstreamMu, sends to closed channel → panic.

upstreamMu does NOT serialize against Detach. Realistic operator pattern (console process exits while buffered keystroke mid-flight) crashes access-node process.

NEW defect induced by pass-1 F-02 fix. Confidence: HIGH.

### F-C-002 — Consumer goroutine leak: blocking UpstreamSink send + no shutdown

**Files:** `internal/session/upstream.go:132-138` (consumer goroutine); `:71` (consumerWg)

Per-console consumer goroutine performs `a.UpstreamSink <- payload` as blocking send. If UpstreamSink (cap 256) fills, goroutine blocks. Subsequent Detach calls close(us) — goroutine parked on UpstreamSink send, not on range us. Never observes close. Never exits. Goroutine leak.

consumerWg incremented per Attach but never Wait()'d. AccessNode has no Close/Shutdown method. Tests Attach but never Detach; UpstreamSink never drained. Each test exits with live blocked goroutines.

NEW defect induced by pass-1 F-06 fix.

## High Findings

### F-H-001 — Keystroke serialization invariant at WRONG boundary

**Files:** `internal/session/upstream.go:199-213` (SendKeystroke); `:132-138` (consumer goroutines); BC-2.04.006 Invariant 3

upstreamMu serializes producer→channel hop. But each console has its own us, and each console's consumer goroutine writes to shared UpstreamSink unsynchronized. The spec's serialization point is "before forwarding to tmux" = UpstreamSink boundary. Test TestSession_ConcurrentKeystrokes_Serialized asserts only nil error returns; doesn't verify ordering.

### F-H-002 — AC-008/BC-2.04.004 EC-002 crash detection STILL unimplemented

**Files:** `internal/session/fanout.go:144-163` (Evict only drains Remove queue); BC-2.04.004 EC-002

BC says "Access node detects channel closure on next keepalive timeout. Session released." Implementation: Evict() doesn't scan channels; only resets slice populated by Remove. No keepalive timer, no probe, no out-of-band close signal. Story task 12 (VP-056 integration test with keepalive) absent.

AC-008 passes only because test simulates crash via Detach. BC postcondition unfulfilled for real-world crash path.

### F-H-003 — AC-002/BC-2.04.003 PC-3 test verifies "accepted" but not "forwarded"

**File:** `internal/session/session_test.go:185-202` (TestSession_Attach_UpstreamKeystrokesForwarded)

Test writes one keystroke into upstream channel and asserts only send goroutine completes. Does NOT read from UpstreamSink, does NOT assert delivery, does NOT invoke SendKeystroke. BC's "forwarded" half of PC-3 has no acceptance evidence.

## Medium Findings

### F-M-001 — UpstreamSink is EXPORTED, allowing external mutation/close

**File:** `internal/session/upstream.go:77`

UpstreamSink chan []byte is exported. External `close(an.UpstreamSink)` panics consumer goroutines. Violates CLAUDE.md Go rule 12.

### F-M-002 — ARCH-09 boundary contract self-contradiction

**Files:** `internal/session/upstream.go:132-138` (goroutine spawn); `session.go:5-7` (godoc says no goroutines)

session.go godoc: "no I/O and does not spawn goroutines itself — those are the responsibility of the effectful layer (internal/tmux)". Attach now spawns per-console goroutine. Self-contradiction. ARCH-09 boundary classification mandates no goroutines for boundary packages.

## Novelty Assessment

Novelty: HIGH. Pass-1's fixes induced TWO critical defects (F-C-001 panic race, F-C-002 leak) + introduced ARCH-09 boundary violation. F-H-001 (serialization at wrong boundary) is deeper than pass-1's original "_ = payload" discard. F-M-002 (package classification self-contradiction) is exactly the architectural drift fresh-context surfaces.

## Resolution decisions (from human review)

- F-M-002: move consumer goroutine to internal/tmux. internal/session stays boundary. Use existing tmux infrastructure (ControlMode has c.stdin; PTYProxy has master) by adding SendInput methods to both + SessionConnector.SendKeystroke that dispatches based on active.
- F-C-001, F-C-002, F-H-001, F-H-003 — all flow from architectural relocation: F-C-001 (Detach race) closed by ownership move; F-C-002 (leak) addressed by AccessNode lifecycle in tmux; F-H-001 (serialization boundary) addressed by serializing at tmux-sink (where spec says); F-H-003 (test gap) addressed by adding sink-drain assertion.
- F-H-002 crash detection: implement keepalive-driven crash detection NOW. Per-console keepalive timestamp + periodic sweep + Heartbeat() method + AC-008 test drives no-heartbeat scenario.
- F-M-001: un-export UpstreamSink; expose via accessor method instead.
- F-04 sentinels already minted by PO in pass-1 (E-SES-002/003/004/005).

## Architectural correction (user-surfaced)

The existing tmux infrastructure (ControlMode + PTYProxy + SessionConnector) is the right home for keystroke-forwarding. ControlMode already has c.stdin writable pipe; PTYProxy has master ReadWriteCloser. AccessNode in internal/session is pure state + dispatch surface. SessionConnector in internal/tmux owns the upstream consumer goroutine + writes to active subsystem (ctrl OR pty).
