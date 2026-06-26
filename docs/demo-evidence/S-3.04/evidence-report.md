# Evidence Report — S-3.04: HMAC Wire-Up into RouteFrame

**Story:** S-3.04
**Branch:** feature/S-3.04-hmac-routeframe-wireup
**Recording method:** Example godoc tests (Go library — no CLI surface; same precedent as S-2.01/S-2.02)

## AC Coverage

| AC / EC | Description | Example function | File | Result |
|---------|-------------|-----------------|------|--------|
| AC-001 | `RouteFrame` with valid HMAC tag proceeds to admission check and `SVTNRoute` | `ExampleRouter_validHMACForwarded` | `internal/routing/example_test.go` | PASS |
| AC-002 | `RouteFrame` with invalid HMAC tag returns `ErrHMACVerificationFailed` immediately | `ExampleRouter_invalidHMACRejected` | `internal/routing/example_test.go` | PASS |
| AC-003 | HMAC verification fires BEFORE admitted-set check (ordering invariant) | `ExampleRouter_hmacBeforeAdmission` | `internal/routing/example_test.go` | PASS |
| AC-004 | No forwarding-table entry for source → `ErrHMACVerificationFailed` (auth key unavailable) | `ExampleRouter_noForwardingEntry` | `internal/routing/example_test.go` | PASS |
| AC-005 | `ErrHMACVerificationFailed` distinct from `admission.ErrNotAdmitted`; valid HMAC + unadmitted → `ErrNotAdmitted` | `ExampleRouter_validHMACUnadmittedRejected` | `internal/routing/example_test.go` | PASS |
| EC-001 | All-zero `HMACTag` field rejected with `ErrHMACVerificationFailed` | `ExampleRouter_zeroHMACTagRejected` | `internal/routing/example_test.go` | PASS |
| EC-002 | Tag computed under a different node's key (cross-node forgery) rejected | `ExampleRouter_wrongKeyHMACRejected` | `internal/routing/example_test.go` | PASS |

## Output Anchors

| Example | Pinned `// Output:` |
|---------|---------------------|
| `ExampleRouter_validHMACForwarded` | `routed without error: true` |
| `ExampleRouter_invalidHMACRejected` | `is ErrHMACVerificationFailed: true` |
| `ExampleRouter_hmacBeforeAdmission` | `is ErrHMACVerificationFailed: true` / `is ErrNotAdmitted: false` |
| `ExampleRouter_noForwardingEntry` | `is ErrHMACVerificationFailed: true` |
| `ExampleRouter_validHMACUnadmittedRejected` | `is ErrNotAdmitted: true` / `is ErrHMACVerificationFailed: false` |
| `ExampleRouter_zeroHMACTagRejected` | `is ErrHMACVerificationFailed: true` |
| `ExampleRouter_wrongKeyHMACRejected` | `is ErrHMACVerificationFailed: true` |

## BC Traceability

| Example | BC anchor | ADR / VP |
|---------|-----------|----------|
| `ExampleRouter_validHMACForwarded` | BC-2.05.008 PC-1 | ADR-009 v1.6 ordering |
| `ExampleRouter_invalidHMACRejected` | BC-2.05.008 PC-2 (E-ADM-016) | ADR-009 v1.6 |
| `ExampleRouter_hmacBeforeAdmission` | BC-2.05.008 PC-3 | VP-058 property 1+2; ADR-009 v1.6 |
| `ExampleRouter_noForwardingEntry` | BC-2.05.008 PC-4 | VP-058 property 4 |
| `ExampleRouter_validHMACUnadmittedRejected` | BC-2.05.008 EC-005 / invariant 2 | — |
| `ExampleRouter_zeroHMACTagRejected` | BC-2.05.008 EC-001 | — |
| `ExampleRouter_wrongKeyHMACRejected` | BC-2.05.008 EC-002 | — |

## Test run summary

```
go test -run "^Example" ./internal/routing/... -v

=== RUN   ExampleRouter_dropsUnadmitted           --- PASS  (S-2.02 AC-004, retained)
=== RUN   ExampleRouter_validHMACForwarded        --- PASS  (S-3.04 AC-001)
=== RUN   ExampleRouter_invalidHMACRejected       --- PASS  (S-3.04 AC-002)
=== RUN   ExampleRouter_hmacBeforeAdmission       --- PASS  (S-3.04 AC-003)
=== RUN   ExampleRouter_noForwardingEntry         --- PASS  (S-3.04 AC-004)
=== RUN   ExampleRouter_validHMACUnadmittedRejected --- PASS  (S-3.04 AC-005)
=== RUN   ExampleRouter_zeroHMACTagRejected       --- PASS  (S-3.04 EC-001)
=== RUN   ExampleRouter_wrongKeyHMACRejected      --- PASS  (S-3.04 EC-002)
=== RUN   ExampleRouter_svtnIsolation             --- PASS  (S-2.02 AC-005, retained)

PASS ok  github.com/arcavenae/switchboard/internal/routing
```

Race detector: `go test ./internal/routing/... -race -count=1` — PASS
Lint: `just lint` — 0 issues

## Notes

- All examples use deterministic seeded keypairs (`ed25519.GenerateKey(bytes.NewReader(seed32(...)))`) so `// Output:` blocks are pinned and repeatable across platforms.
- `ExampleRouter_validHMACForwarded` passes a non-nil payload (`[]byte("hello-switchboard")`) to demonstrate that HMAC covers header+payload together, and to satisfy the `unparam` linter which requires `exampleComputeTag`'s `payload` parameter to be exercised with a non-nil value at least once.
- `ExampleRouter_hmacBeforeAdmission` uses a node that IS admitted; the invalid HMAC tag causes rejection before the admitted-set check is reached, proving HMAC-before-admission ordering.
- `ExampleRouter_validHMACUnadmittedRejected` uses `errors.Is` on both sentinels to verify they are truly distinct (AC-005 / BC-2.05.008 invariant 2).
- Pre-existing S-2.02 examples (`ExampleRouter_dropsUnadmitted`, `ExampleRouter_svtnIsolation`) were updated during S-3.04 implementation to provide the forwarding entry and valid HMAC tag required by the new HMAC-first ordering.
