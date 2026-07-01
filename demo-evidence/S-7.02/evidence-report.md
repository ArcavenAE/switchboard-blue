# S-7.02 Demo Evidence Report

**Story:** S-7.02 v1.6 — SVTN-Scoped Multicast Session Discovery  
**Module:** `internal/discovery`  
**HEAD:** a9bf936  
**Status:** CONVERGED under BC-5.39.001 (Pass-8/9/10 all clean 3-lens)  
**Recorded:** 2026-07-01

---

## Coverage Summary

| Recording | AC | BC/VP | Tests Demonstrated | Paths |
|-----------|----|-------|--------------------|-------|
| AC-001a-advertise-on-state-change | AC-001a | BC-2.03.001 PC-3 | TestDiscovery_Advertise_OnStateChange, TestDiscovery_Advertise_OnStateChange_DetachTriggersAdvert | success + error (detach EC-001) |
| AC-001b-periodic-heartbeat | AC-001b | BC-2.03.001 PC-4, RULING-W6TB-G | TestDiscovery_Advertise_PeriodicHeartbeat_ExactN, TestDiscovery_HeartbeatCount_MonotonicallyIncreases | success (exact-N oracle) + observability counter |
| AC-002-enumerate-no-hostname | AC-002 | BC-2.03.002 PC-1 PC-3 Inv-1 | TestDiscovery_Enumerate_NoHostnameRequired, TestDiscovery_Enumerate_EmptyWithoutAdvertisements | success (>=2 distinct addrs) + error (empty) |
| AC-003-advertisement-required-fields | AC-003 | BC-2.03.003 PC-1 | TestDiscovery_Advertisement_RequiredFields, TestDiscovery_Advertisement_QualityUnknownOnStartup | success (all fields present) + EC-002 (unknown quality at startup) |
| AC-004-advertisement-round-trip | AC-004 | BC-2.03.003 Inv-1, VP-055 | TestDiscovery_AdvertisementRoundTrip, TestPropPresenceAdvertisement_RoundTrip | success (deterministic) + property (1–255 byte names) |
| AC-004b-utf8-rune-boundary-truncation | AC-004b | BC-2.03.003 PC-2 EC-001, VP-055 v1.2, RULING-W6TB-I, RULING-W6TB-J | TestDiscovery_Encode_SessionName255ByteCap, TestPropPresenceAdvertisement_TruncatesOversize, TestPropPresenceAdvertisement_RejectsEmptyOrInvalidUTF8 | success (255 accept, 256 truncated) + error (empty/invalid UTF-8 rejected) |
| AC-005-hmac-first-authentication | AC-005 | BC-2.03.001 PC-5, RULING-W6TB-H | TestDiscovery_Advertise_HMACAuthenticated, TestDiscovery_Enumerate_SVTNIsolation_ForgedSVTN | success (valid HMAC) + error (forged SVTN -> ErrInvalidHMACTag before SVTN check) |
| AC-006-svtn-cross-scope-isolation | AC-006 | BC-2.03.002 Inv-1, VP-045 | TestDiscovery_Enumerate_SVTNIsolation, TestDiscovery_VP045_SVTNIsolation_MultipleScopes | success (zero SVTN-B sessions in SVTN-A) + VP-045 e2e multi-scope |

---

## Recordings

### AC-001a: State-Change Advertisement Trigger

- `AC-001a-advertise-on-state-change.gif`
- `AC-001a-advertise-on-state-change.webm`
- `AC-001a-advertise-on-state-change.tape`

Traces to **BC-2.03.001 PC-3**. Demonstrates `Discovery.Advertise` fires within 1 tick on session state change (add/remove/attach status). Both the primary trigger and the detach EC-001 path are shown.

### AC-001b: Periodic Heartbeat (RULING-W6TB-G)

- `AC-001b-periodic-heartbeat.gif`
- `AC-001b-periodic-heartbeat.webm`
- `AC-001b-periodic-heartbeat.tape`

Traces to **BC-2.03.001 PC-4**. Primary oracle uses `Config.TickSource` injection (exact-N tick count, no wall-clock sensitivity) per RULING-W6TB-G. `HeartbeatCount()` atomic accessor demonstrates unconditional production observability counter (Pass-3 L1 M-1).

### AC-002: Enumerate Without Hostnames

- `AC-002-enumerate-no-hostname.gif`
- `AC-002-enumerate-no-hostname.webm`
- `AC-002-enumerate-no-hostname.tape`

Traces to **BC-2.03.002 PC-1, PC-3, Inv-1**. Oracle asserts `len(distinctNodeAddrs(result)) >= 2`. Error path shows empty result when no advertisements have been received.

### AC-003: Advertisement Required Fields

- `AC-003-advertisement-required-fields.gif`
- `AC-003-advertisement-required-fields.webm`
- `AC-003-advertisement-required-fields.tape`

Traces to **BC-2.03.003 PC-1**. Verifies presence of `session_name`, `attachment_status`, `quality_indicator`. EC-002 (quality unknown at startup) path confirmed.

### AC-004: Advertisement Round-Trip Stability

- `AC-004-advertisement-round-trip.gif`
- `AC-004-advertisement-round-trip.webm`
- `AC-004-advertisement-round-trip.tape`

Traces to **BC-2.03.003 Inv-1** and **VP-055**. `Encode(Decode(payload)) == payload` for deterministic cases and verified by property test over all valid UTF-8 names 1–255 bytes.

### AC-004b: UTF-8 Rune-Boundary Truncation

- `AC-004b-utf8-rune-boundary-truncation.gif`
- `AC-004b-utf8-rune-boundary-truncation.webm`
- `AC-004b-utf8-rune-boundary-truncation.tape`

Traces to **BC-2.03.003 PC-2 + EC-001** and **VP-055 v1.2** (RULING-W6TB-I, RULING-W6TB-J). Boundary cases: 255-byte name accepted without truncation; 256-byte name truncated to 252 bytes + U+2026. Property test verifies maximality and rune-boundary correctness. Error path confirms empty and non-UTF-8 names return `err != nil`.

### AC-005: HMAC-First Authentication

- `AC-005-hmac-first-authentication.gif`
- `AC-005-hmac-first-authentication.webm`
- `AC-005-hmac-first-authentication.tape`

Traces to **BC-2.03.001 PC-5** (RULING-W6TB-H ordering). Valid HMAC accepted and sessions stored. Forged-SVTN attack path shows `ErrInvalidHMACTag` returned before any SVTN comparison — confirming HMAC-first ordering with key derived from `payload.SVTNID`.

### AC-006: SVTN Cross-Scope Isolation

- `AC-006-svtn-cross-scope-isolation.gif`
- `AC-006-svtn-cross-scope-isolation.webm`
- `AC-006-svtn-cross-scope-isolation.tape`

Traces to **BC-2.03.002 Inv-1** and **VP-045**. Oracle asserts `len(sessionsFromSVTNB(svtnAResult)) == 0`. VP-045 e2e demonstrates isolation holds across multiple independent SVTN scopes.

---

## Race-Detector Transcript

`race-test-transcript.txt` — `go test -count=1 -race ./internal/discovery/...`  
Exit code: 0 (no data races detected, all tests PASS)

---

## BC/VP Traceability

| Behavioral Contract | Postcondition/Invariant | AC | Status |
|---------------------|------------------------|----|--------|
| BC-2.03.001 | PC-3 (state-change trigger) | AC-001a | PASS |
| BC-2.03.001 | PC-4 (30s periodic heartbeat) | AC-001b | PASS |
| BC-2.03.001 | PC-5 (HMAC authentication, HMAC-first ordering) | AC-005 | PASS |
| BC-2.03.002 | PC-1 (enumerate without hostnames) | AC-002 | PASS |
| BC-2.03.002 | PC-3 (>=2 distinct advertisers) | AC-002 | PASS |
| BC-2.03.002 | Inv-1 (SVTN cross-scope negative) | AC-006 | PASS |
| BC-2.03.003 | PC-1 (required payload fields) | AC-003 | PASS |
| BC-2.03.003 | PC-2 + EC-001 (255-byte name cap + truncation) | AC-004b | PASS |
| BC-2.03.003 | Inv-1 (round-trip stability) | AC-004 | PASS |
| VP-044 | Advertisement within 1 tick | AC-001a | PASS |
| VP-045 | SVTN isolation multi-scope e2e | AC-006 | PASS |
| VP-055 v1.2 | Round-trip, truncation, rejection properties | AC-004 + AC-004b | PASS |
