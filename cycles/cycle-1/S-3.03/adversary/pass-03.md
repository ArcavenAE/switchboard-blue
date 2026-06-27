---
artifact_id: adv-S-3.03-pass-03
review_target: S-3.03-tier2-session-authorization
producer: adversary
pass: 3
fresh_context: true
branch: feature/S-3.03-tier2-session-authorization
findings_count: 3
findings_by_severity: {critical: 0, high: 0, medium: 1, low: 2, observations: 8}
verdict: CONVERGED
streak_after_pass: 2
streak_reset_reason: null
timestamp: 2026-06-26
note: Returned inline by adversary; persisted via state-manager.
---

# Adversarial Review — Pass 3 — S-3.03

## Disposition

CONVERGED — zero CRITICAL, zero HIGH. Streak: 2/3 (second consecutive clean pass).

Fresh-context adversary made a sustained effort to construct an unauthorized/read-only data-leak path and a race that drops the auth check; could construct NEITHER.

## Verified-Clean (independently re-derived)

- **O-1:** Attach-time gate precedes fan-out membership; `(nil, nil, err)` on denial, `errors.Is(ErrSessionAuthDenied)`, non-vacuous nil-channel + non-membership tests.
- **O-2:** Lock ordering `sinkMu → ConsoleSet.mu → SessionAuth.mu` consistent, no inversion/deadlock; `sinkMu` serialization closes attach-vs-send TOCTOU.
- **O-3:** No go.md rule-12 internal-pointer leak from any accessor (`Authorize` → `Role` value, `Snapshot` → `[]ConsoleKey` copy, `Session` → string, `ListSessions`/`Get` → `Info` value).
- **O-4:** VP-012 router-no-Tier-2-state holds (routing grep clean; import boundary `{frame, admission}` only; audit test is a real regression guard).
- **O-5:** Read-only empty-tick forwarded to `KeystrokeSink` per BC-2.04.005 v1.3 EC-004.
- **O-6:** Enforcement tests mutation-resistant incl. decision-matrix row proving unregistered-key empty-tick is STILL denied (exemption does not bypass auth).
- **O-7:** Per-session isolation / no SVTN spillover (PC-4, VP-013).
- **O-8:** AC-001..AC-006 + Task 7 satisfied; S-3.02-FM1 upstream channel dangling-but-documented (production routes via `SendKeystroke` → sink, channel test-only) — acceptable.

## Findings (none blocking)

### M-1 (MEDIUM, recurring from pass-2 O-2/O-3) — E-ADM-006/007 opaque `ConsoleKey` vs `<key_fingerprint>` / missing `<node_addr>`

**Spec reference:** `error-taxonomy.md` E-ADM-006, E-ADM-007

E-ADM-006/007 render a full opaque `ConsoleKey` rather than `<key_fingerprint>` and omit `<node_addr>`. Adversary notes this is largely a layering reality: `internal/session` has no node identity; `ConsoleKey` is a non-secret opaque id.

**Adjudication (architect):** `error-taxonomy.md` clarified (v1.6) with a layering note — opaque `ConsoleKey` serves as fingerprint at the session layer; `node_addr` is supplied by the transport/admission boundary. No code change required. `errors.Is` holds. Not a security or identity gap.

**Status:** CLOSED — `error-taxonomy.md` v1.6 layering note.

### L-1 (LOW) — BC-2.05.003 EC-004 "revoke" half has no enforcement surface

**Spec reference:** BC-2.05.003 EC-004

`SessionAuth` exposes `RegisterKey` (add/overwrite) but no `RevokeKey` or `Remove`. The "revoke" half of BC-2.05.003 EC-004 has no enforcement surface.

**Adjudication:** Out of S-3.03 AC scope. Deferred to operator-provisioning story (Wave 4+). Tracked as drift item S-3.03-L1-REVOKE.

**Status:** DEFERRED — out of scope for S-3.03.

### L-2 (LOW) — Stale `Publisher` doc comment in `session.go`

**File:** `internal/session/session.go`

Stale `Publisher` doc comment implied `SessionAuth` draws from `AdmittedKeySet`; Tier-2 keys are independent (DI-011).

**Resolution:** Comment corrected (implementer commit, pass-3).

**Status:** CLOSED — implementer fix.

## Observations

### O-1 — Attach-gate + fan-out membership verified (see Verified-Clean above)

### O-2 — Lock ordering verified sound (see Verified-Clean above)

### O-3 — No internal-pointer leaks (see Verified-Clean above)

### O-4 — VP-012 router state verified clean (see Verified-Clean above)

### O-5 — Read-only empty-tick forwarding verified (see Verified-Clean above)

### O-6 — Enforcement tests mutation-resistant (see Verified-Clean above)

### O-7 — Per-session isolation verified (see Verified-Clean above)

### O-8 — AC-001..AC-006 + Task 7 + S-3.02-FM1 disposition (see Verified-Clean above)

## Fix Commits This Pass

- Implementer: L-2 stale doc comment correction

## Spec Edits This Pass

- `error-taxonomy.md` v1.6 — M-1 layering note: opaque `ConsoleKey` as session-layer fingerprint; `node_addr` supplied at transport/admission boundary

## Deferred

- L-1 — `RevokeKey`/`Remove` unenforced: deferred to operator-provisioning story (Wave 4+), tracked as drift item S-3.03-L1-REVOKE

## Novelty Assessment

Novelty: VERY LOW. Pass 3 following two prior clean passes. Adversary made sustained effort to construct unauthorized-read or auth-drop race — neither exploitable. M-1 is a recurring layering observation fully resolved by spec clarification. L-1 is a known out-of-scope gap. L-2 is a trivial stale comment. Convergence streak advances to 2/3 (second consecutive clean pass). Explicitly CONVERGED.
