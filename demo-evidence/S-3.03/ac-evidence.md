# Demo Evidence: S-3.03 — Tier-2 Per-Session Authorization & Read-Only Enforcement

**Story:** S-3.03  
**Branch:** feature/S-3.03-tier2-session-authorization  
**Commit:** 0b9b776  
**Date:** 2026-06-26  
**Recording format:** Test transcripts (VHS/terminal recording intentionally omitted per project standing preference; evidence is verifiable via `go test` output)

---

## Test Suite Summary

```
ok  github.com/arcavenae/switchboard/internal/session   1.429s
```

**70 PASS lines** (top-level tests + subtests), **0 FAIL**, **0 SKIP**.  
Run command: `go test -race -v ./internal/session/`  
Race detector: enabled.

---

## Acceptance Criterion → Test Mapping

### AC-001 (BC-2.05.003 PC-1)
`SessionAuth.Authorize(consoleKey, sessionName)` succeeds when the console's public key is in the session's authorization list; returns nil error and the console's role.

| Test function | Subtests | PASS line |
|---|---|---|
| `TestSessionAuth_Authorize_RegisteredKey_Succeeds` | `full-access_key`, `read-only_key`, `full-access_on_different_session` | `--- PASS: TestSessionAuth_Authorize_RegisteredKey_Succeeds (0.00s)` |

Relevant PASS lines from transcript:
```
--- PASS: TestSessionAuth_Authorize_RegisteredKey_Succeeds (0.00s)
    --- PASS: TestSessionAuth_Authorize_RegisteredKey_Succeeds/full-access_key (0.00s)
    --- PASS: TestSessionAuth_Authorize_RegisteredKey_Succeeds/read-only_key (0.00s)
    --- PASS: TestSessionAuth_Authorize_RegisteredKey_Succeeds/full-access_on_different_session (0.00s)
```

---

### AC-002 (BC-2.05.003 PC-2)
`SessionAuth.Authorize` returns `E-ADM-006` ("session authorization denied") when the console's key is NOT in the session's authorization list.

| Test function | Subtests | PASS line |
|---|---|---|
| `TestSessionAuth_Authorize_UnregisteredKey_ErrAdmSix` | `completely_empty_auth_list`, `different_key_registered`, `key_registered_on_different_session_only` | `--- PASS: TestSessionAuth_Authorize_UnregisteredKey_ErrAdmSix (0.00s)` |
| `TestSessionAuth_ErrorMessages_MatchTaxonomy` | `E-ADM-006:_Authorize_returns_key_and_session_in_error_message` | validates error string content |

Relevant PASS lines from transcript:
```
--- PASS: TestSessionAuth_Authorize_UnregisteredKey_ErrAdmSix (0.00s)
    --- PASS: TestSessionAuth_Authorize_UnregisteredKey_ErrAdmSix/completely_empty_auth_list (0.00s)
    --- PASS: TestSessionAuth_Authorize_UnregisteredKey_ErrAdmSix/different_key_registered (0.00s)
    --- PASS: TestSessionAuth_Authorize_UnregisteredKey_ErrAdmSix/key_registered_on_different_session_only (0.00s)
--- PASS: TestSessionAuth_ErrorMessages_MatchTaxonomy (0.00s)
    --- PASS: TestSessionAuth_ErrorMessages_MatchTaxonomy/E-ADM-006:_Authorize_returns_key_and_session_in_error_message (0.00s)
```

---

### AC-003 (BC-2.05.003 PC-3 + invariant 1 / VP-012 code-audit)
The router (`internal/routing`) has no per-session authorization data structure. Tier-2 enforcement lives exclusively in `internal/session`.

| Test function | Subtests | PASS line |
|---|---|---|
| `TestSessionAuth_RouterHasNoTier2State` | — | `--- PASS: TestSessionAuth_RouterHasNoTier2State (0.00s)` |

Relevant PASS line from transcript:
```
--- PASS: TestSessionAuth_RouterHasNoTier2State (0.00s)
```

---

### AC-004 (BC-2.05.003 PC-4)
Authorization is per-session: a console authorized for `session-A` is NOT automatically authorized for `session-B`.

| Test function | Subtests | PASS line |
|---|---|---|
| `TestSessionAuth_Authorize_PerSession_NoSpillover` | — | `--- PASS: TestSessionAuth_Authorize_PerSession_NoSpillover (0.00s)` |
| `TestSessionAuth_CrossSession_Rejected` | — | `--- PASS: TestSessionAuth_CrossSession_Rejected (0.00s)` |
| `TestSessionAuth_Allow_DecisionMatrix` | `cross-session_(registered_on_sess,_queried_on_other)_→_ErrSessionAuthDenied` | validates `Allow()` path |

Relevant PASS lines from transcript:
```
--- PASS: TestSessionAuth_Authorize_PerSession_NoSpillover (0.00s)
--- PASS: TestSessionAuth_CrossSession_Rejected (0.00s)
--- PASS: TestSessionAuth_Allow_DecisionMatrix (0.00s)
    --- PASS: TestSessionAuth_Allow_DecisionMatrix/cross-session_(registered_on_sess,_queried_on_other)_→_ErrSessionAuthDenied (0.00s)
```

---

### AC-005 (BC-2.04.005 PC-1 + PC-3)
A read-only console can attach and receive all downstream frames. Any upstream keystroke frame (payload-bearing) is rejected with `E-ADM-007`. The rejection does NOT terminate the downstream subscription.

| Test function | Subtests | PASS line |
|---|---|---|
| `TestReadOnlyConsole_UpstreamRejected_DownstreamContinues` | — | `--- PASS: TestReadOnlyConsole_UpstreamRejected_DownstreamContinues (0.00s)` |
| `TestReadOnlyConsole_FullAndReadOnly_BothAttached` | — | `--- PASS: TestReadOnlyConsole_FullAndReadOnly_BothAttached (0.00s)` |
| `TestSessionAuth_Allow_DecisionMatrix` | `read-only_+_payload_→_ErrUpstreamReadOnly` | validates E-ADM-007 error path |
| `TestSessionAuth_ErrorMessages_MatchTaxonomy` | `E-ADM-007:_Allow_returns_key_and_session_in_error_message_for_read-only_console` | validates error string |

Relevant PASS lines from transcript:
```
--- PASS: TestReadOnlyConsole_UpstreamRejected_DownstreamContinues (0.00s)
--- PASS: TestReadOnlyConsole_FullAndReadOnly_BothAttached (0.00s)
--- PASS: TestSessionAuth_Allow_DecisionMatrix (0.00s)
    --- PASS: TestSessionAuth_Allow_DecisionMatrix/read-only_+_payload_→_ErrUpstreamReadOnly (0.00s)
--- PASS: TestSessionAuth_ErrorMessages_MatchTaxonomy (0.00s)
    --- PASS: TestSessionAuth_ErrorMessages_MatchTaxonomy/E-ADM-007:_Allow_returns_key_and_session_in_error_message_for_read-only_console (0.00s)
```

---

### AC-006 (BC-2.04.005 PC-3 + EC-004)
Empty-tick frames (liveness probes, zero-length payload) from a read-only console are accepted; only payload-bearing upstream frames are rejected.

| Test function | Subtests | PASS line |
|---|---|---|
| `TestReadOnlyConsole_EmptyTickAccepted` | — | `--- PASS: TestReadOnlyConsole_EmptyTickAccepted (0.00s)` |
| `TestReadOnlyConsole_EmptyTickForwarded_WithCaptureSink` | — | `--- PASS: TestReadOnlyConsole_EmptyTickForwarded_WithCaptureSink (0.00s)` |
| `TestSessionAuth_Allow_DecisionMatrix` | `read-only_+_empty-tick_→_accepted`, `read-only_+_nil_payload_→_accepted` | validates both zero-len and nil payload paths |

Relevant PASS lines from transcript:
```
--- PASS: TestReadOnlyConsole_EmptyTickAccepted (0.00s)
--- PASS: TestReadOnlyConsole_EmptyTickForwarded_WithCaptureSink (0.00s)
--- PASS: TestSessionAuth_Allow_DecisionMatrix (0.00s)
    --- PASS: TestSessionAuth_Allow_DecisionMatrix/read-only_+_empty-tick_→_accepted (0.00s)
    --- PASS: TestSessionAuth_Allow_DecisionMatrix/read-only_+_nil_payload_→_accepted (0.00s)
```

---

## Additional Coverage (Task 7 — Authorizer wiring)

These tests verify `SessionAuth` is wired as the live `Authorizer` in the upstream-receive path (task 7 in the story task list).

| Test function | Purpose | PASS line |
|---|---|---|
| `TestSessionAuth_ImplementsAuthorizer` | compile-time interface check: `*SessionAuth` satisfies `Authorizer` | `--- PASS: TestSessionAuth_ImplementsAuthorizer (0.00s)` |
| `TestAccessNode_Attach_ReadOnlyKey_Succeeds` | M-1: read-only key can attach via access node with live authorizer | `--- PASS: TestAccessNode_Attach_ReadOnlyKey_Succeeds (0.00s)` |
| `TestSessionAuth_ConcurrentRegisterAndAuthorize` | M-2: no race on concurrent register+authorize (race detector clean) | `--- PASS: TestSessionAuth_ConcurrentRegisterAndAuthorize (0.01s)` |

Relevant PASS lines from transcript:
```
--- PASS: TestSessionAuth_ImplementsAuthorizer (0.00s)
--- PASS: TestAccessNode_Attach_ReadOnlyKey_Succeeds (0.00s)
--- PASS: TestSessionAuth_ConcurrentRegisterAndAuthorize (0.01s)
```

---

## Edge Case Coverage

| EC | Test | PASS |
|---|---|---|
| EC-001/EC-002: empty auth list → all rejected | `TestSessionAuth_EmptyAuthList_AllRejected`, `TestAccessNode_Attach_EmptyAuthList_Rejected` | PASS |
| EC-003: cross-session rejection | `TestSessionAuth_CrossSession_Rejected` | PASS |
| EC-004: full-access and read-only both attached | `TestReadOnlyConsole_FullAndReadOnly_BothAttached` | PASS |
| EC-006: empty-tick from read-only accepted | `TestReadOnlyConsole_EmptyTickAccepted`, `TestReadOnlyConsole_EmptyTickForwarded_WithCaptureSink` | PASS |

---

## Full Test Run Summary

```
=== RUN   TestSessionAuth_Authorize_RegisteredKey_Succeeds
=== RUN   TestSessionAuth_Authorize_UnregisteredKey_ErrAdmSix
=== RUN   TestSessionAuth_RouterHasNoTier2State
=== RUN   TestSessionAuth_Authorize_PerSession_NoSpillover
=== RUN   TestSessionAuth_EmptyAuthList_AllRejected
=== RUN   TestSessionAuth_CrossSession_Rejected
=== RUN   TestReadOnlyConsole_UpstreamRejected_DownstreamContinues
=== RUN   TestReadOnlyConsole_EmptyTickAccepted
=== RUN   TestReadOnlyConsole_FullAndReadOnly_BothAttached
=== RUN   TestSessionAuth_Allow_DecisionMatrix
=== RUN   TestSessionAuth_RegisterKey_LastWriteWins
=== RUN   TestSessionAuth_SentinelErrors_Distinct
=== RUN   TestSessionAuth_ImplementsAuthorizer
=== RUN   TestAccessNode_Attach_UnauthorizedKey_Rejected
=== RUN   TestAccessNode_Attach_AuthorizedKey_Succeeds
=== RUN   TestAccessNode_Attach_EmptyAuthList_Rejected
=== RUN   TestSessionAuth_ErrorMessages_MatchTaxonomy
=== RUN   TestReadOnlyConsole_EmptyTickForwarded_WithCaptureSink
=== RUN   TestAccessNode_Attach_ReadOnlyKey_Succeeds
=== RUN   TestSessionAuth_ConcurrentRegisterAndAuthorize
[... ConsoleSet, Publisher, Session tests ...]

--- PASS: TestSessionAuth_Authorize_RegisteredKey_Succeeds (0.00s)
    --- PASS: TestSessionAuth_Authorize_RegisteredKey_Succeeds/full-access_key (0.00s)
    --- PASS: TestSessionAuth_Authorize_RegisteredKey_Succeeds/read-only_key (0.00s)
    --- PASS: TestSessionAuth_Authorize_RegisteredKey_Succeeds/full-access_on_different_session (0.00s)
--- PASS: TestSessionAuth_Authorize_UnregisteredKey_ErrAdmSix (0.00s)
    --- PASS: TestSessionAuth_Authorize_UnregisteredKey_ErrAdmSix/completely_empty_auth_list (0.00s)
    --- PASS: TestSessionAuth_Authorize_UnregisteredKey_ErrAdmSix/different_key_registered (0.00s)
    --- PASS: TestSessionAuth_Authorize_UnregisteredKey_ErrAdmSix/key_registered_on_different_session_only (0.00s)
--- PASS: TestSessionAuth_RouterHasNoTier2State (0.00s)
--- PASS: TestSessionAuth_Authorize_PerSession_NoSpillover (0.00s)
--- PASS: TestSessionAuth_EmptyAuthList_AllRejected (0.00s)
    --- PASS: TestSessionAuth_EmptyAuthList_AllRejected/console-a (0.00s)
    --- PASS: TestSessionAuth_EmptyAuthList_AllRejected/console-b (0.00s)
    --- PASS: TestSessionAuth_EmptyAuthList_AllRejected/console-c (0.00s)
--- PASS: TestSessionAuth_CrossSession_Rejected (0.00s)
--- PASS: TestReadOnlyConsole_UpstreamRejected_DownstreamContinues (0.00s)
--- PASS: TestReadOnlyConsole_EmptyTickAccepted (0.00s)
--- PASS: TestReadOnlyConsole_FullAndReadOnly_BothAttached (0.00s)
--- PASS: TestSessionAuth_Allow_DecisionMatrix (0.00s)
    --- PASS: TestSessionAuth_Allow_DecisionMatrix/full-access_+_payload_→_accepted (0.00s)
    --- PASS: TestSessionAuth_Allow_DecisionMatrix/full-access_+_empty-tick_→_accepted (0.00s)
    --- PASS: TestSessionAuth_Allow_DecisionMatrix/read-only_+_payload_→_ErrUpstreamReadOnly (0.00s)
    --- PASS: TestSessionAuth_Allow_DecisionMatrix/read-only_+_empty-tick_→_accepted (0.00s)
    --- PASS: TestSessionAuth_Allow_DecisionMatrix/read-only_+_nil_payload_→_accepted (0.00s)
    --- PASS: TestSessionAuth_Allow_DecisionMatrix/unregistered_key_+_payload_→_ErrSessionAuthDenied (0.00s)
    --- PASS: TestSessionAuth_Allow_DecisionMatrix/unregistered_key_+_empty-tick_→_ErrSessionAuthDenied (0.00s)
    --- PASS: TestSessionAuth_Allow_DecisionMatrix/cross-session_(registered_on_sess,_queried_on_other)_→_ErrSessionAuthDenied (0.00s)
--- PASS: TestSessionAuth_RegisterKey_LastWriteWins (0.00s)
--- PASS: TestSessionAuth_SentinelErrors_Distinct (0.00s)
--- PASS: TestSessionAuth_ImplementsAuthorizer (0.00s)
--- PASS: TestAccessNode_Attach_UnauthorizedKey_Rejected (0.00s)
--- PASS: TestAccessNode_Attach_AuthorizedKey_Succeeds (0.00s)
--- PASS: TestAccessNode_Attach_EmptyAuthList_Rejected (0.00s)
    --- PASS: TestAccessNode_Attach_EmptyAuthList_Rejected/console-empty-a (0.00s)
    --- PASS: TestAccessNode_Attach_EmptyAuthList_Rejected/console-empty-b (0.00s)
    --- PASS: TestAccessNode_Attach_EmptyAuthList_Rejected/console-empty-c (0.00s)
--- PASS: TestSessionAuth_ErrorMessages_MatchTaxonomy (0.00s)
    --- PASS: TestSessionAuth_ErrorMessages_MatchTaxonomy/E-ADM-006:_Authorize_returns_key_and_session_in_error_message (0.00s)
    --- PASS: TestSessionAuth_ErrorMessages_MatchTaxonomy/E-ADM-007:_Allow_returns_key_and_session_in_error_message_for_read-only_console (0.00s)
--- PASS: TestReadOnlyConsole_EmptyTickForwarded_WithCaptureSink (0.00s)
--- PASS: TestAccessNode_Attach_ReadOnlyKey_Succeeds (0.00s)
--- PASS: TestSessionAuth_ConcurrentRegisterAndAuthorize (0.01s)
--- PASS: ExamplePublisher_publishUnpublish (0.00s)
PASS
ok  github.com/arcavenae/switchboard/internal/session   1.429s
```

---

*No VHS tape files, GIF recordings, or asciinema captures were created. This project uses Go test transcripts as the accepted demo-evidence format.*
