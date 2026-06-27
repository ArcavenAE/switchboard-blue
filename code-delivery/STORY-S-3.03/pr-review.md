# PR #14 Review — feat(S-3.03): tier-2 per-session authorization and read-only enforcement

**Verdict: APPROVE**

Fresh-eyes review of the diff only (`feature/S-3.03-tier2-session-authorization` → `develop`).
No blocking findings. All 8 checklist items pass. Two NITs noted below — neither blocks merge.

---

## What I verified (no rubber-stamp)

I reviewed all 4 changed files:

- `internal/session/auth.go` (new, 156 lines)
- `internal/session/auth_test.go` (new, 1434 lines)
- `internal/session/session.go` (modified, comment-only)
- `internal/session/upstream.go` (modified, +10/-1)

I also read the surrounding `upstream.go` context on the branch (the `Authorizer`
interface, `NoOpAuthorizer`, `NewAccessNode`, and `SendKeystroke`) because the diff
alone does not show that read-only enforcement is wired into the keystroke path —
I needed to confirm the claim holds.

### Correctness

- **`Authorize`** — per-session two-level map lookup
  (`map[sessionName]map[ConsoleKey]authEntry`). Returns `ErrSessionAuthDenied`
  (E-ADM-006) wrapped with `%w` on both the session-miss and key-miss branches.
  Per-session isolation is structural: a key under session-A's submap is invisible
  to a session-B lookup. Correct (BC-2.05.003 PC-1/PC-2/PC-4).
- **`Allow`** — delegates to `Authorize`, then rejects only
  `role == RoleReadOnly && len(payload) > 0`. Empty and nil payloads pass for
  read-only consoles. Matches BC-2.04.005 EC-004 (empty-tick liveness accepted).
- **`Attach` gate** (the load-bearing new wiring) — calls
  `a.authorizer.Allow(key, sessionName, nil)` BEFORE `a.consoles.Add(...)`. An
  unregistered key is denied with nil channels and no partial fan-out state; a
  read-only key passes via the empty-payload exemption so it can still receive
  downstream frames. Ordering is correct: denial leaves no resource to leak.
- **Read-only keystroke enforcement** — `SendKeystroke` already called
  `a.authorizer.Allow(key, sessionName, payload)` (shipped in S-3.02 with
  `NoOpAuthorizer`). Wiring `SessionAuth` as the live `Authorizer` activates
  enforcement on the real payload-bearing path. The PR description's claim is
  accurate; the enforcement is not a no-op.
- **`RegisterKey`** — last-write-wins, lazy submap init, full write lock. Correct.

### Concurrency / lock ordering

`SessionAuth.mu` is an `RWMutex`: reads (`Authorize`/`Allow`) take `RLock`, writes
(`RegisterKey`) take `Lock`. I checked for nested-lock deadlock risk against the
existing `sinkMu` / `ConsoleSet.mu`:

- In `Attach`, `Allow` (→ `SessionAuth.mu`) is a standalone call BEFORE
  `consoles.Add` (→ `ConsoleSet.mu`); the two locks are never held simultaneously.
- In `SendKeystroke`, `sinkMu` is held, then `consoles.Session` acquires/releases
  `ConsoleSet.mu`, then `Allow` acquires/releases `SessionAuth.mu` — sequential,
  no simultaneous nesting of `ConsoleSet.mu` and `SessionAuth.mu`.

No lock-ordering inversion. `go test -race ./internal/session/` is clean on my run.

### Go idioms (go.md compliance)

- Errors wrapped with `%w`, sentinel identity preserved (`errors.Is` works).
- Error strings have no trailing punctuation (ST1005).
- No nil-check-before-`len`; uses `len(payload) > 0` directly.
- All error returns handled; no `data, _ :=` swallowing.
- Value-copy returns from locked accessors (`Role` is an int value) — no internal
  pointer leak (go.md rule 12). `authEntry` is unexported and never escapes.
- No `init()`, no global mutable state, constructor-based (`NewSessionAuth`).
- Tests: stdlib `testing` only (no testify), table-driven, `t.Helper()` in
  helpers, `t.Parallel()` throughout.

### Test coverage (meaningful, regression-catching)

20 top-level test functions covering AC-001..AC-006 plus edge cases (EC-002/003/004),
the full `Allow` decision matrix, last-write-wins, sentinel distinctness, the
compile-time `Authorizer` guard, the routing code-audit (VP-012), positive
read-only attach, and a concurrent register/authorize race net.

The AC-005 and AC-006 tests use a deliberate two-assertion structure
(reject-payload-first, then accept-empty-tick / prove-downstream-liveness) that is
genuinely mutation-resistant: removing or inverting the enforcement check fails one
of the paired assertions rather than passing vacuously. This is good adversarial
test design.

### PR description accuracy / traceability

Description matches the diff. Every AC maps to a named test, the dependency on #13
is stated, deferred items (`S-3.03-L1-REVOKE`, `S-3.03-O1-VPSKEL`) are disclosed,
and the FM1 drift resolution (replacing the vestigial no-op authorizer) is reflected
in the actual code change. Commit history is conventional-format with story IDs.

### Diff size

~1606 additions, but 1434 of those are tests (single new test file). Production code
is 156 new lines + an 11-line wiring change. The >500-line flag is explained entirely
by test volume; production surface is small and reviewable. Not a concern.

---

## Findings

| Severity | Category | Finding | Suggestion |
|----------|----------|---------|------------|
| NIT | coverage | `TestSessionAuth_ImplementsAuthorizer` ends with `_ = fmt.Sprintf(...)` whose result is discarded — a no-op "belt-and-suspenders" line that adds nothing over the compile-time `var _ session.Authorizer = ...` assertion above it. | Drop the discarded `fmt.Sprintf` line; the compile-time assertion is the real guard. Harmless as-is. |
| NIT | coherence | `auth.go` doc comment still lists "S-3.03 implementer tasks: Task 5..8" as a TODO-style checklist. Now that the story is implemented, this reads as stale planning scaffolding in shipped code. | Trim the task list to a one-line description of the component's role; keep the BC references. Cosmetic. |

Neither finding affects correctness, security, or behavior. Both are optional cleanups.

---

## Demo evidence note (information-wall caveat)

Per checklist item 4, this repo's standing preference (stated in the PR) is Go test
transcripts rather than VHS/`.gif` recordings, since the changed surface is a pure
in-process Go boundary layer with no UI or terminal output. I cannot independently
confirm the transcript artifact from the diff alone, but the test suite is present in
the diff, is meaningful, and passes race-clean — which is the substantive evidence
for a non-UI change. I am not flagging this as blocking.

---

## Summary

Clean, well-tested, idiomatic implementation of Tier-2 per-session authorization and
read-only enforcement. The attach-time gate ordering is correct (denial before any
state mutation), the read-only enforcement is genuinely wired (not a no-op), lock
ordering is sound, and the tests are mutation-resistant rather than vacuous. The two
NITs are cosmetic.

**APPROVE.**
