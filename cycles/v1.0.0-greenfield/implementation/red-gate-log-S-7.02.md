# Red Gate Log — S-7.02 SVTN-Scoped Multicast Session Discovery

**Story:** S-7.02 — implement SVTN-scoped multicast session discovery in internal/discovery  
**Date:** 2026-07-01  
**Phase:** TDD (test-writer pass, v1.1)  
**BC-5.38.001 Status:** RED GATE VERIFIED

## Summary

17 tests written in `internal/discovery/discovery_test.go`. All 17 fail against
the current stubs (panic: not implemented). Zero lint issues. Formatter clean.

## Test Files

| File | Tests | Status |
|------|-------|--------|
| `internal/discovery/discovery_test.go` | 17 | FAILING (Red Gate) |

## Per-Test Red Gate Results

| Test Name | AC/BC Trace | Failure Mode |
|-----------|-------------|--------------|
| `TestDiscovery_Advertise_OnStateChange` | AC-001a / BC-2.03.001 PC-3 | `panic: not implemented` — `Advertise` stub |
| `TestDiscovery_Advertise_OnStateChange_DetachTriggersAdvert` | AC-001a / BC-2.03.001 PC-3 + EC-001 | `panic: not implemented` — `Advertise` stub |
| `TestDiscovery_Advertise_PeriodicHeartbeat` | AC-001b / BC-2.03.001 PC-4 | `panic: not implemented` — `Run` stub |
| `TestDiscovery_Advertise_PeriodicHeartbeat_IsIndependent` | AC-001b / BC-2.03.001 PC-4 | `panic: not implemented` — `Run` stub |
| `TestDiscovery_Enumerate_NoHostnameRequired` | AC-002 / BC-2.03.002 PC-3 | `panic: not implemented` — `Encode` stub |
| `TestDiscovery_Enumerate_EmptyWithoutAdvertisements` | AC-002 / BC-2.03.002 EC-002 | `panic: not implemented` — `Enumerate` stub |
| `TestDiscovery_Enumerate_SameSessionNameTwoNodes` | AC-002 / BC-2.03.002 EC-003 | `panic: not implemented` — `Encode` stub |
| `TestDiscovery_Advertisement_RequiredFields` | AC-003 / BC-2.03.003 PC-1 | `panic: not implemented` — `Encode` stub |
| `TestDiscovery_Advertisement_QualityUnknownOnStartup` | AC-003 / BC-2.03.003 EC-002 | `panic: not implemented` — `Encode` stub |
| `TestDiscovery_AdvertisementRoundTrip` | AC-004 / BC-2.03.003 Inv-1 | `panic: not implemented` — `Encode` stub |
| `TestDiscovery_Advertise_HMACAuthenticated` | AC-005 / BC-2.03.001 PC-5 | `panic: not implemented` — `Encode` stub |
| `TestDiscovery_Advertise_HMACAuthenticated_EmptyPayload` | AC-005 / BC-2.03.001 PC-5 | `panic: not implemented` — `ReceiveAdvertisement` stub |
| `TestDiscovery_Enumerate_SVTNIsolation` | AC-006 / BC-2.03.002 Inv-1 | `panic: not implemented` — `Encode` stub |
| `TestDiscovery_Enumerate_SVTNIsolation_ErrSentinel` | AC-006 / BC-2.03.002 Inv-1 | `panic: not implemented` — `Encode` stub |
| `TestDiscovery_VP044_AdvertiseWithinOneTick` | VP-044 / BC-2.03.001 PC-3 (property) | `panic: not implemented` — `Advertise` stub |
| `TestDiscovery_VP045_SVTNIsolation_MultipleScopes` | VP-045 / BC-2.03.002 Inv-1 (property) | `panic: not implemented` — `Encode` stub |
| `TestDiscovery_VP055_RoundTripProperty` | VP-055 / BC-2.03.003 Inv-1 (property) | `panic: not implemented` — `Encode` stub |

## go test output (truncated)

```
--- FAIL: TestDiscovery_Advertise_OnStateChange (0.00s)
panic: not implemented [recovered, repanicked]

goroutine ... github.com/arcavenae/switchboard/internal/discovery.(*Discovery).Advertise(...)
    /internal/discovery/discovery.go:150
...
panic: not implemented [recovered, repanicked]

goroutine ... github.com/arcavenae/switchboard/internal/discovery.Encode(...)
    /internal/discovery/discovery.go:178
...
FAIL    github.com/arcavenae/switchboard/internal/discovery    0.300s
```

## Handoff Notes for Implementer

Make each test pass one at a time with minimum code. Suggested order:

1. `Encode` / `Decode` (unblocks 11 tests that call these first)
2. `ReceiveAdvertisement` + SVTN isolation (AC-006)
3. `ReceiveAdvertisement` + HMAC rejection (AC-005)
4. `Enumerate` (AC-002)
5. `Advertise` state-change path (AC-001a)
6. `Run` heartbeat loop (AC-001b)

Architecture constraints to enforce during implementation:
- `internal/discovery` imports `internal/routing` only (ARCH-08 pos 14→5)
- `internal/discovery` MUST NOT import `internal/hmac` or `internal/frame`
- HMAC authentication uses the Router's HMAC surface exclusively
