---
artifact_id: adv-S-3.03-pass-02
review_target: S-3.03-tier2-session-authorization
producer: adversary
pass: 2
fresh_context: true
branch: feature/S-3.03-tier2-session-authorization
findings_count: 2
findings_by_severity: {critical: 0, high: 0, medium: 2, low: 0, observations: 7}
verdict: CONVERGED
streak_after_pass: 1
streak_reset_reason: null
timestamp: 2026-06-26
note: Returned inline by adversary; persisted via state-manager.
---

# Adversarial Review — Pass 2 — S-3.03

## Verified Fixed from Pass 1

- **C-1** — attach-time auth gate confirmed correct and complete. On denial, `Attach` returns `(nil, nil, err)` with `errors.Is(ErrSessionAuthDenied)`; no partial state, no console added, no channel. Non-vacuous fan-out proof confirmed (see O-7 below).
- **H-1 / H-2** — error format conformance confirmed. E-ADM-006 / E-ADM-007 messages now interpolate required fields; taxonomy-format test asserts interpolated fields are present in error output.

## Clean Areas (independently verified)

- **O-4 (TOCTOU):** TOCTOU between `Attach` `Allow()` and `consoles.Add()` is spec-acceptable — no revoke method exists; BC-2.05.003 EC-004 states "existing sessions continue until next re-auth." Not a defect.
- **O-5 (lock ordering):** Lock ordering `sinkMu → {ConsoleSet.mu | SessionAuth.mu}` is sound. Inner locks never simultaneously held. No go.md rule-12 internal-pointer leak — `Session` returns a string copy; `Authorize` returns a `Role` int.
- **O-6 (VP-012 / AC-003):** Router-has-no-Tier-2-state genuinely enforced. Routing grep clean; import boundary `{frame, admission}` only.
- **O-7 (attach-denial state):** Attach denial leaves no partial state and no resource leak confirmed.
- **Test integrity:** All new `Attach` tests, empty-tick-forwarded test, taxonomy-format test, and per-session-isolation test verified mutation-resistant — each would fail if the gate were removed or inverted.
- **AC-001..AC-006 + Task 7:** All PASS. S-3.02-FM1 resolved — `SessionAuth` wired as live `Authorizer` at attach-time.

## Medium Findings

### M-1 — Empty-tick from read-only console forwarded to tmux KeystrokeSink

**Spec reference:** BC-2.04.005 EC-004

An empty tick originating from a read-only console is forwarded to the tmux `KeystrokeSink`. The spec's phrase "liveness probe credited" was ambiguous: it could mean forwarding a zero-length frame to `KeystrokeSink`, or it could mean the `ConsoleSet` keepalive heartbeat.

**Adjudication (architect):** CORRECT as-is. "Liveness probe credited" = forwarding the zero-length frame to `KeystrokeSink`, NOT the `ConsoleSet` keepalive heartbeat. Code unchanged. BC-2.04.005 EC-004 clarified to v1.3 to resolve the ambiguity.

**Status:** CLOSED — spec clarification only; no code change.

### M-2 — `Attach` checks session existence before auth gate

**Files:** `internal/session/upstream.go`

`Attach` calls `pub.Get` (session existence check) before the auth gate, disclosing session existence to an unauthorized console via a distinct error path.

**Adjudication (architect):** Intentional and spec-acceptable. BC-2.05.003 EC-001 explicitly permits unauthorized consoles to list sessions. Ordering is correct. An `// intentional ordering: BC-2.05.003 EC-001 permits unauthorized consoles to list sessions` comment added for future reviewers (implementer commit c354251).

**Status:** CLOSED — intentional; comment added.

## Observations

### O-1 — `SendKeystroke` "console not found in session" violated error-taxonomy E-SES-003

Error-taxonomy E-SES-003 specifies: "preposition is 'for' … any use of 'in' is a defect." The phrase "console not found in session" used the wrong preposition.

**Resolution:** Changed to "for" (implementer commit c354251).

### O-2 — E-ADM-006 omits `<node_addr>` (SessionAuth lacks node identity)

`SessionAuth` does not carry node identity, so the `<node_addr>` field mandated by E-ADM-006 cannot be populated at this layer. The caller that owns node identity must supply it.

**Resolution:** Deferred to integration perimeter / wave-gate. No action at S-3.03 boundary.

### O-3 — Error messages interpolate `string(key)` directly

`ConsoleKey` is an opaque non-secret identifier today. If it ever carries a real credential, direct interpolation into error strings could leak it.

**Resolution:** Deferred. If `ConsoleKey` gains credential semantics, switch to a hashed fingerprint. No action required now.

### O-4 — TOCTOU verified clean (see Clean Areas above)

### O-5 — Lock ordering verified sound (see Clean Areas above)

### O-6 — VP-012 / AC-003 router state verified clean (see Clean Areas above)

### O-7 — Attach-denial state verified clean (see Clean Areas above)

## Fix Commits This Pass

- Implementer: c354251 (O-1 preposition fix + M-2 intentional-ordering comment)

## Spec Edits This Pass

- `BC-2.04.005.md` v1.3 — EC-004 clarified: "liveness probe credited" = forwarding zero-length frame to `KeystrokeSink` (not `ConsoleSet` keepalive heartbeat)

## Deferred

- O-2 — `<node_addr>` field in E-ADM-006: deferred to integration perimeter / wave-gate
- O-3 — `ConsoleKey` fingerprint representation: deferred; no action until key gains credential semantics

## Novelty Assessment

Novelty: LOW. Pass 2 following full remediation of C-1 / H-1 / H-2. Remaining findings (M-1, M-2) required architect adjudication on spec interpretation and intentional design decisions, not implementation defects. Both resolved without code changes. Convergence streak advances to 1/3 (first clean pass).
