# Evidence Report — S-2.02: Tier-1 Admission and SVTN Isolation

**Story:** S-2.02  
**Branch:** feature/S-2.02-admission-svtn-isolation  
**Recording method:** Example godoc tests (Go library — no CLI surface; same precedent as S-2.01)

## AC Coverage

| AC | Description | Example function | File | Result |
|----|-------------|-----------------|------|--------|
| AC-001 | `AdmitNode` succeeds with valid challenge-response signature | `ExampleAdmittedKeySet_admitNode` | `internal/admission/example_test.go` | PASS |
| AC-002 | `AdmitNode` returns `ErrSignatureVerificationFailed` (E-ADM-001) for invalid signature | `ExampleAdmittedKeySet_invalidSignature` | `internal/admission/example_test.go` | PASS |
| AC-003 | `AdmitNode` returns `ErrNonceReplay` (E-ADM-008) for replayed nonce | `ExampleAdmittedKeySet_replayDetection` | `internal/admission/example_test.go` | PASS |
| AC-004 | `RouteFrame` drops frame and returns `ErrNotAdmitted` (E-ADM-003) for unadmitted source | `ExampleRouter_dropsUnadmitted` | `internal/routing/example_test.go` | PASS |
| AC-005 | `SVTNRoute` never delivers frame to node on different SVTN | `ExampleRouter_svtnIsolation` | `internal/routing/example_test.go` | PASS |
| AC-006 | No private key bytes in wire structs (`Challenge`, `ChallengeResponse`) | `ExampleGenerateChallenge_privateKeyAbsent` | `internal/admission/example_test.go` | PASS |
| AC-007 | `GenerateChallenge` produces nonce without transmitting private key | `ExampleGenerateChallenge_privateKeyAbsent` | `internal/admission/example_test.go` | PASS |

## Test run summary

```
go test -run "^Example" ./internal/admission/... ./internal/routing/... -v

=== RUN   ExampleAdmittedKeySet_admitNode         --- PASS
=== RUN   ExampleAdmittedKeySet_invalidSignature  --- PASS
=== RUN   ExampleAdmittedKeySet_replayDetection   --- PASS
=== RUN   ExampleAdmittedKeySet_revokedKey        --- PASS  (EC-003 edge case)
=== RUN   ExampleAdmittedKeySet_isAdmitted        --- PASS
=== RUN   ExampleGenerateChallenge_privateKeyAbsent --- PASS
=== RUN   ExampleRouter_dropsUnadmitted           --- PASS
=== RUN   ExampleRouter_svtnIsolation             --- PASS

PASS ok  github.com/arcavenae/switchboard/internal/admission
PASS ok  github.com/arcavenae/switchboard/internal/routing
```

Race detector: `go test ./internal/admission/... ./internal/routing/... -race` — PASS  
Lint: `just lint` — 0 issues

## Notes

- AC-006 and AC-007 are both covered by `ExampleGenerateChallenge_privateKeyAbsent`, which verifies that neither `Challenge.Nonce` nor `Challenge.RouterSig` appears as a substring of the router's raw private key bytes.
- An additional example `ExampleAdmittedKeySet_revokedKey` covers edge case EC-003 (revoked key returns `ErrKeyRevoked`).
- All examples use deterministic seeded keypairs (`ed25519.GenerateKey(bytes.NewReader(seed32(...)))`) so `// Output:` blocks are pinned and repeatable across platforms.
