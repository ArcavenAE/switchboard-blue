# Evidence Report — S-1.03: Session Continuity via Cryptographic Re-Authentication

**Story:** S-1.03
**Branch:** feature/S-1.03-node-identity-session-continuity
**Recording method:** Example godoc tests (Go library — no CLI surface; same precedent as S-2.02)

## AC/EC Coverage

| ID | Description | Example function | File | Expected output snippet | Result |
|----|-------------|-----------------|------|------------------------|--------|
| AC-001 | Session resumes on IP change (BC-2.01.007 PC3+PC4) | `ExampleAdmittedKeySet_reAuthenticateOnIPChange` | `internal/admission/example_test.go` | `reauth error: <nil>` / `still admitted: true` / `source addr: 192.0.2.42` | PASS |
| AC-002 | Wrong keypair rejected (BC-2.01.007 Pre3) | `ExampleAdmittedKeySet_reAuthenticateWrongKeyRejected` | `internal/admission/example_test.go` | `is ErrSignatureVerificationFailed: true` | PASS |
| AC-003 | Node address stable after re-auth (BC-2.01.007 Inv3) | `ExampleAdmittedKeySet_reAuthenticateNodeAddressStable` | `internal/admission/example_test.go` | `addr stable: true` / `still admitted with same addr: true` | PASS |
| EC-001 | Expired key → ErrKeyExpired / E-ADM-015 (BC-2.01.007 EC-005) | `ExampleAdmittedKeySet_reAuthenticateExpiredKey` | `internal/admission/example_test.go` | `is ErrKeyExpired: true` | PASS |
| EC-002 | Old path evicted on new re-auth (BC-2.01.007 EC-006) | `ExampleAdmittedKeySet_reAuthenticateEvictsOldPath` | `internal/admission/example_test.go` | `after second reauth: 198.51.100.20` | PASS |
| EC-003 | Last write wins — sequential demo (ADR-003; concurrent variant: TestReAuthenticate_NoRace) | `ExampleAdmittedKeySet_reAuthenticateLastWriteWins` | `internal/admission/example_test.go` | `last write wins: 10.10.10.2` | PASS |

## Test run summary

```
go test -run "^Example" ./internal/admission/... -v

=== RUN   ExampleAdmittedKeySet_reAuthenticateOnIPChange     --- PASS
=== RUN   ExampleAdmittedKeySet_reAuthenticateWrongKeyRejected --- PASS
=== RUN   ExampleAdmittedKeySet_reAuthenticateNodeAddressStable --- PASS
=== RUN   ExampleAdmittedKeySet_reAuthenticateExpiredKey     --- PASS
=== RUN   ExampleAdmittedKeySet_reAuthenticateEvictsOldPath  --- PASS
=== RUN   ExampleAdmittedKeySet_reAuthenticateLastWriteWins  --- PASS

(plus all 6 S-2.02 examples — PASS)
ok  github.com/arcavenae/switchboard/internal/admission  0.498s

go test ./internal/admission/... ./internal/routing/... -race -count=1
ok  github.com/arcavenae/switchboard/internal/admission  2.364s
ok  github.com/arcavenae/switchboard/internal/routing    1.364s

just lint
0 issues.
```

## EC-003 LWW note

EC-003 requires last-write-wins semantics under concurrent re-authentication.
The Example function demonstrates the sequential shape (two `ReAuthenticate` calls
in order; second IP wins). The concurrent variant — two goroutines racing to claim
the same `(svtnID, nodeAddr)` — is covered by `TestReAuthenticate_NoRace` in
`internal/admission/reauth_test.go`, which runs clean under `go test -race`.
A concurrent `// Output:` block is not feasible because the winning goroutine is
non-deterministic; the sequential Example is the appropriate godoc evidence.

## Traceability

| Example | BC anchor | Error code | Story AC/EC |
|---------|-----------|------------|-------------|
| `reAuthenticateOnIPChange` | BC-2.01.007 PC3+PC4 | — | AC-001 |
| `reAuthenticateWrongKeyRejected` | BC-2.01.007 Pre3 | E-ADM-001 | AC-002 |
| `reAuthenticateNodeAddressStable` | BC-2.01.007 Inv3 | — | AC-003 |
| `reAuthenticateExpiredKey` | BC-2.01.007 EC-005 | E-ADM-015 | EC-001 |
| `reAuthenticateEvictsOldPath` | BC-2.01.007 EC-006 | — | EC-002 |
| `reAuthenticateLastWriteWins` | BC-2.01.007 EC-003 / ADR-003 | — | EC-003 |
