---
document_type: red-gate-log
level: ops
version: "1.0"
status: complete
producer: state-manager
timestamp: 2026-07-15T01:56:48Z
phase: 3
inputs:
  - stories/S-BL.DISCOVERY-WIRE.md
input-hash: "915d2b4"
traces_to: stories/S-BL.DISCOVERY-WIRE.md
stub_architect_agent: stub-architect
stub_compile_verified: true
test_writer_agent: tw-dw-stubs
red_gate_verified: true
cycle: cycle-1
---

# Red Gate Log: S-BL.DISCOVERY-WIRE

Delivery branch: `feature/S-BL.DISCOVERY-WIRE`, branched from `develop`.
Red Gate discipline per BC-5.38.001: stubs compile and pass registration
tests, then a failing test suite is written against those stubs before any
real implementation lands.

## Summary

| Story | Tests Written | All Fail (Red)? | Gate |
|-------|----------------|------------------|------|
| S-BL.DISCOVERY-WIRE | 28 | Yes (28/28) | PASSED |

## Stubs Created

### S-BL.DISCOVERY-WIRE

Commit `7f36e73` (signed), 7 files, +359 lines. All bodies
`panic("not implemented: ...")` except pure data constants. `go build` /
`go vet` / `gofumpt` / `golangci-lint` clean at `develop@1f25677` base;
existing test compilation unaffected.

**Task 1 (AC-004, key derivation):**
- `internal/hmac/hmac.go`: `HKDFInfoDiscovery` constant (real) +
  `DeriveDiscoveryKey(...)` function -- stub
- `internal/routing/advertisement_hmac.go`: `(*Router).DiscoveryAuthKeyFor`
  and package-level `DeriveDiscoveryKey` -- stub thin wrappers per Ruling 1
  Implementation Constraints 1 and 4

**Task 2 (AC-005..013, router-side ingest):**
- `internal/discovery/discovery_wire.go` (new): `RouterIngestConfig`,
  `RouterIngestDecision`, `NewRouterIngest` constructor -- real;
  `Ingest` method -- stub. No `lastSeen`/mutex fields yet (Task 2's
  Green-step addition).

**Task 3 (AC-001..003, multicast listener + sender dispatch):**
- `internal/discovery/discovery.go`: `DiscoveryPort` constant (real,
  49201 per the 2026-07-14 human gate disposition) + `MulticastAddrFor`
  function -- stub
- `cmd/switchboard/discovery_wire.go` (new): `wireDiscoveryListener` --
  stub, deliberately not called from `runRouter` yet (wiring is Task 3's
  Green-step action)
- `internal/testenv/multicast_loopback.go` (new):
  `MulticastLoopbackInterface` -- stub, purpose-built loopback helper,
  explicitly not an extension of `NewLoopback` per Decision 2(e)

**Task 4 (AC-007, node-local relay-ingest):**
- `internal/discovery/discovery.go`: `IngestRelayAdvertisement` method --
  stub, added alongside the still-intact `ReceiveAdvertisement`
  (retirement is Task 4's Green-step action)

**Task 5 (AC-014..016, hop-2 frame assembly; not gated):**
- `cmd/switchboard/discovery_relay_wire.go` (new):
  `assembleDiscoveryRelayFrame` -- stub, pure function

Deliberate Green-deferrals per the story's Task Breakdown: `runRouter`
listener wiring (Task 3 Green), `ReceiveAdvertisement`/`advertisementKey`
deletion (Task 4 Green). AC-017/AC-018/Task 6 GATED on
`S-BL.NODE-IDENTIFY-WIRE` per story v2.12.

## Red Gate Verification

Commit `4bc578c` (signed): 28 new/extended test functions across 5 files
(3 new: `internal/discovery/discovery_wire_test.go`,
`cmd/switchboard/discovery_relay_wire_test.go` [AC-014/015/016 only,
AC-017/018 portion GATED], `internal/testenv/multicast_loopback_test.go`;
2 extended: `internal/hmac/hmac_test.go`,
`internal/routing/advertisement_hmac_test.go`).

### S-BL.DISCOVERY-WIRE

- AC-001 (BC-2.03.001 PC-1/Inv-1): `TestRunRouter_DiscoveryListener_JoinsGroup_RouterModeOnly` -- FAIL (expected)
- AC-002 (BC-2.03.001 PC-3): `TestMulticastAddrFor_Deterministic_SHA256Derived` -- FAIL (expected)
- AC-002 (BC-2.03.001 PC-3): `TestMulticastLoopbackInterface_ResolvesLoopback` -- FAIL (expected)
- AC-003 (BC-2.03.001 PC-1): `TestDiscovery_Advertise_WriteToMulticast_TTL1_NoGroupJoin` -- FAIL (expected)
- AC-004 (BC-2.03.001 PC-5): `TestDeriveDiscoveryKey_DomainSeparatedFromFrameAuthKey` -- FAIL (expected)
- AC-004 (BC-2.03.001 PC-5): `TestDeriveDiscoveryKey_SenderRouterAgree` -- FAIL (expected)
- AC-004 (BC-2.03.001 PC-5): `TestDiscoveryAuthKeyFor_LookupSuccessAndMiss` -- FAIL (expected)
- AC-005 (BC-2.03.001 PC-5): `TestRouterIngest_KeySelectorExtraction_FixedOffset_NoFullDecodeBeforeAuth` -- FAIL (expected)
- AC-006 (BC-2.03.001 PC-5): `TestRouterIngest_HMACCoversFullBody_TamperInSessionListDetected` -- FAIL (expected)
- AC-006 (BC-2.03.001 PC-5): `TestRouterIngest_LookupMissAndTagMismatch_IndistinguishableRejection` -- FAIL (expected)
- AC-007 (BC-2.03.001 PC-5): `TestDiscovery_IngestRelayAdvertisement_SVTNMismatch_ErrSVTNMismatch` -- FAIL (expected)
- AC-007 (BC-2.03.001 PC-5): `TestDiscovery_IngestRelayAdvertisement_NoHMACRequired` -- FAIL (expected)
- AC-007 (BC-2.03.001 PC-5): `TestDiscovery_IngestRelayAdvertisement_Success_RegistryReplaceOnWrite` -- FAIL (expected)
- AC-008 (BC-2.03.001 PC-2; VP-080 property 1): `TestVP080_DiscoveryIngest_ColdStartAcceptance` -- FAIL (expected)
- AC-009 (BC-2.03.001 PC-2; VP-080 property 2): `TestVP080_DiscoveryIngest_ReplayDiscard_ExactSequence` -- FAIL (expected)
- AC-009 (BC-2.03.001 PC-2; VP-080 property 2): `TestVP080_DiscoveryIngest_ReplayDiscard_LowerSequence` -- FAIL (expected)
- AC-009 (BC-2.03.001 PC-2; VP-080 property 2): `TestVP080_DiscoveryIngest_ReplayDiscard_NoRelaySideEffect` -- FAIL (expected)
- AC-010 (BC-2.03.001 PC-2; VP-080 property 3): `TestVP080_DiscoveryIngest_ForwardAcceptance_AdvancesState` -- FAIL (expected)
- AC-010 (BC-2.03.001 PC-2; VP-080 property 3): `TestVP080_DiscoveryIngest_RestartForwardProgress` -- FAIL (expected)
- AC-011 (BC-2.03.001 PC-5): `TestRouterIngest_FullValidFrameMinimum_42Bytes` -- FAIL (expected)
- AC-011 (BC-2.03.001 PC-5): `TestRouterIngest_OversizedDatagram_RejectedNoPartialParse` -- FAIL (expected)
- AC-011 (BC-2.03.001 PC-5): `TestRouterIngest_ShortDatagram_RejectedBeforeLookup` -- FAIL (expected)
- AC-012 (BC-2.03.001 PC-5): `TestRouterIngest_AggregateRateCap_NotPerSource` -- FAIL (expected)
- AC-012 (BC-2.03.001 PC-5): `TestRouterIngest_FailureCounter_VisibilityOnly_NeverGates` -- FAIL (expected)
- AC-013 (BC-2.03.001 PC-5): `TestRouterIngest_FailureLogging_ThresholdCrossingOnly_NotPerPacket` -- FAIL (expected)
- AC-014 (BC-2.01.008 PC-2/PC-3/Inv-5; BC-2.03.001 PC-5): `TestAssembleDiscoveryRelayFrame_PayloadLayout` -- FAIL (expected)
- AC-015 (BC-2.03.001 PC-1 delivery note): `TestAssembleDiscoveryRelayFrame_ZeroHMACTag` -- FAIL (expected)
- AC-016 (BC-2.03.001 PC-5 relay/connection-trust note): `TestAssembleDiscoveryRelayFrame_NotRawHop1Bytes` -- FAIL (expected)

**Orchestrator-verified independently:** `go test ./...` -> exactly 28
FAIL, every failure via the "red gate: stub not yet implemented"
sentinel; 354 pre-existing tests PASS; lint clean.

**Coverage notes (flagged, not silently narrowed):**
- AC-001 PC-2 verified by import-inspection (absence not
  runtime-observable without 3 extra daemons)
- AC-003 PC-2 (TTL=1) not wire-verified -- needs `x/net`, violates the
  zero-new-deps constraint, deferred to Green code review
- AC-004 PC-5 + AC-007 PC-1/PC-5 are structural Green-step facts

**Test-writer stub extension approved by the orchestrator:**
`RouterIngestConfig` gained an optional `Logger` field (AC-013
observation seam; ARCH-08 §6.5 import direction respected).

**Scope:** AC-001..AC-016 (Tasks 1-5). AC-017/AC-018/Task 6 GATED on
`S-BL.NODE-IDENTIFY-WIRE` per story v2.12.

## Regression Check

| Existing Tests | Status |
|-----------------|--------|
| 354 pre-existing tests | all pass |

`go build` / `go vet` / `gofumpt` / `golangci-lint` clean on the full
tree at both the stub commit (`7f36e73`) and the Red test commit
(`4bc578c`).

## Hand-Off to Implementer

- Stories ready for implementation: S-BL.DISCOVERY-WIRE (Tasks 1-5,
  AC-001..AC-016)
- Implementation guidance: Green phase proceeds task-by-task per the
  story's Task Breakdown ordering (Task 1 key derivation -> Task 2
  router-side ingest -> Task 3 multicast listener/sender wiring ->
  Task 4 node-local relay-ingest retirement of `ReceiveAdvertisement`
  -> Task 5 hop-2 frame assembly). AC-003 PC-2 (TTL=1) wire-level
  verification is explicitly deferred to Green code review per the
  zero-new-deps constraint. AC-017/AC-018/Task 6 (fan-out dispatch,
  rate cap) remain GATED on `S-BL.NODE-IDENTIFY-WIRE` and are out of
  scope for this Green phase.

## Verdict

RED GATE PASSED 2026-07-14 (orchestrator-verified). Implementer
authorized.
