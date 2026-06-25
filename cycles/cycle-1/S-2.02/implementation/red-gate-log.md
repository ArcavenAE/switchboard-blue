# S-2.02 Red Gate Log

Generated: 2026-06-25
Worktree: `/Users/skippy/work/switchboard-blue/.worktrees/S-2.02`
Branch: `feature/S-2.02-admission-svtn-isolation`

## Summary

- `go build ./internal/admission/... ./internal/routing/...`: PASS
- `go vet ./...`: PASS
- `just fmt`: PASS (gofumpt clean)
- `just lint`: PASS (0 issues)
- `go test ./internal/admission/... ./internal/routing/...`: RED GATE HOLDS

## Test Results

```
--- FAIL: TestAdmitNode_ValidChallenge (panic: not implemented: S-2.02 AdmittedKeySet.RegisterKey)
--- FAIL: TestAdmitNode_InvalidSignature (panic: not implemented: S-2.02 AdmittedKeySet.RegisterKey)
--- FAIL: TestAdmitNode_ReplayedNonce (panic: not implemented: S-2.02 AdmittedKeySet.RegisterKey)
--- FAIL: TestAdmitNode_KeyNotRegisteredForSVTN (panic: not implemented: S-2.02 AdmittedKeySet.RegisterKey)
--- FAIL: TestDuplicateKeyRegistration_LastWriteWins (panic: not implemented: S-2.02 AdmittedKeySet.RegisterKey)
--- FAIL: TestAdmitNode_RevokedKey (panic: not implemented: S-2.02 AdmittedKeySet.RegisterKey)
--- FAIL: TestGenerateChallenge_NonceUniqueness (panic: not implemented: S-2.02 GenerateChallenge)
--- FAIL: TestGenerateChallenge_NoChallengeContainsPrivateKey (panic: not implemented: S-2.02 GenerateChallenge)
--- FAIL: TestIsAdmitted_FailClosed (panic: not implemented: S-2.02 AdmittedKeySet.IsAdmitted)
--- PASS: TestAdmission_PrivateKeyAbsentFromWireStructs (GREEN-BY-DESIGN — see below)
--- FAIL: TestSVTNRoute_NoCrossContamination (panic: not implemented: S-2.02 AdmittedKeySet.RegisterKey)
--- FAIL: TestSVTNRoute_SVTNPartitionBoundary (panic: not implemented: S-2.02 AdmittedKeySet.RegisterKey)
--- FAIL: TestSVTNRoute_AdmittedFrameForwardedToCorrectSVTN (panic: not implemented: S-2.02 AdmittedKeySet.RegisterKey)
--- FAIL: TestRouteFrame_DropsUnadmitted (panic: not implemented: S-2.02 AdmittedKeySet.RegisterKey)
--- FAIL: TestRouteFrame_AdmittedSetCheckPrecedesForwarding (panic: not implemented: S-2.02 AdmittedKeySet.RegisterKey)
FAIL github.com/arcavenae/switchboard/internal/admission
FAIL github.com/arcavenae/switchboard/internal/routing
```

## GREEN-BY-DESIGN Tests

| Test | Justification |
|------|--------------|
| `TestAdmission_PrivateKeyAbsentFromWireStructs` | Pure structural check on zero-value structs. Asserts `Challenge.Nonce` is `[32]byte` (len check) and `ChallengeResponse.NonceSig` zero value is nil (len check). Zero branching, no I/O, no non-trivial helpers, 4 lines of assertion logic. Cannot fail against any stub — the type system enforces the invariant. Traces to BC-2.05.007 invariant 1 (VP-007). |

## Panic Pattern

All failing tests panic with messages matching: `not implemented: S-2.02 <Symbol>`

Specific symbols:
- `AdmittedKeySet.RegisterKey`
- `AdmittedKeySet.IsAdmitted`
- `AdmittedKeySet.RevokeKey`
- `AdmittedKeySet.Lookup`
- `GenerateChallenge`
- `AdmitNode`
- `Router.RegisterForwardingEntry`
- `RouteFrame`
- `SVTNRoute`
