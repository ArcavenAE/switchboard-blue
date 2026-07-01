# Demo Evidence Report — S-W5.04

**Story:** daemon-side paths.list / router.metrics / router.status RPC handlers and response types
**Story ID:** S-W5.04 (v1.17)
**Convergence:** 3/3 clean fresh 3-lens passes (BC-5.39.001)
**HEAD:** d889094b1af928aa0485da4b787492959ed844e9
**Recorded:** 2026-07-01

## Coverage Summary

| AC | Description | BC Trace | Recording | Status |
|----|-------------|----------|-----------|--------|
| AC-001 | paths.list handler: populated + EC-001 empty source | BC-2.06.003 PC-1 | AC-001-paths-list-handler | PASS |
| AC-002 | rtt_p99_ms union: "pending" (<10 samples) / float64 (>=10) | BC-2.06.003 PC-1, EC-003 | AC-002-rtt-p99-serialization | PASS |
| AC-003 | PathEntry.status derivation: active/degraded from Active+Degraded flags | BC-2.06.003 PC-1, Ruling-9 | AC-003-status-derivation | PASS |
| AC-004 | router.metrics handler: positive path + E-RPC-011 not-found | BC-2.06.003 PC-2 | AC-004-router-metrics-handler | PASS |
| AC-005 / AC-005a | overallQuality precedence: pending > red > yellow > green; empty-paths → pending | BC-2.06.003 PC-3, EC-007, EC-008 | AC-005-overall-quality-precedence | PASS |
| AC-006 | register-before-serve + VP-047 end-to-end handler→source→response | VP-047, BC-2.06.003 PC-1 | AC-006-register-and-vp047-e2e | PASS |

## Recordings

### AC-001 — paths.list handler

Tests: `TestDaemonPathsList_HandlerRegistered` (populated source → PathsListResponse with PathEntry), `TestDaemonPathsList_EmptySource` (EC-001: empty → `{"paths":[],"message":"no active paths"}`).

- [AC-001-paths-list-handler.gif](AC-001-paths-list-handler.gif)
- [AC-001-paths-list-handler.webm](AC-001-paths-list-handler.webm)
- [AC-001-paths-list-handler.tape](AC-001-paths-list-handler.tape)

### AC-002 — rtt_p99_ms union serialization

Tests: `TestPathEntry_RTTValueSerialization` (table-driven: SampleCount=0 → "pending"; SampleCount=9 → "pending"; SampleCount=10 → float64; SampleCount=100 → float64), `TestRTTValue_RoundTrip` (MarshalJSON/UnmarshalJSON round-trip for both PendingKind and FloatKind, including float64(0) discrimination).

- [AC-002-rtt-p99-serialization.gif](AC-002-rtt-p99-serialization.gif)
- [AC-002-rtt-p99-serialization.webm](AC-002-rtt-p99-serialization.webm)
- [AC-002-rtt-p99-serialization.tape](AC-002-rtt-p99-serialization.tape)

### AC-003 — PathEntry.status derivation (Ruling-9)

Tests: `TestPathEntry_StatusFromDegraded` (Degraded=true → "degraded"; Active=false,Degraded=false → "degraded" [active_false_is_degraded, normative per Ruling-9]; Active=true,Degraded=false → "active"), `TestPathEntry_StatusEnumClosed` (closed-enum guard: "failed" cannot be emitted).

- [AC-003-status-derivation.gif](AC-003-status-derivation.gif)
- [AC-003-status-derivation.webm](AC-003-status-derivation.webm)
- [AC-003-status-derivation.tape](AC-003-status-derivation.tape)

### AC-004 — router.metrics handler

Tests: `TestDaemonRouterMetrics_HandlerRegistered` (positive path: svtn_id present → RouterMetricsResponse with frame_count, hmac_fail_count, drop_cache_hits, path_distribution), `TestDaemonRouterMetrics_SVTNNotFound` (E-RPC-011 not-found when SVTN absent from source).

- [AC-004-router-metrics-handler.gif](AC-004-router-metrics-handler.gif)
- [AC-004-router-metrics-handler.webm](AC-004-router-metrics-handler.webm)
- [AC-004-router-metrics-handler.tape](AC-004-router-metrics-handler.tape)

### AC-005 / AC-005a — overallQuality precedence

Tests: `TestDaemonRouterStatus_QualityStatusIndependence` (row b: Degraded=true + SampleCount=10 → quality derived from p99, not pending; row c: status="active" + SampleCount=5 → quality="pending"; row d: empty-paths → quality="pending" [EC-008]), `TestRouterStatus_EmptyPaths_QualityIsPending` (companion for EC-008 empty-paths case), `TestDaemonRouterStatus_RedBand` (red-band quality from high p99 RTT).

- [AC-005-overall-quality-precedence.gif](AC-005-overall-quality-precedence.gif)
- [AC-005-overall-quality-precedence.webm](AC-005-overall-quality-precedence.webm)
- [AC-005-overall-quality-precedence.tape](AC-005-overall-quality-precedence.tape)

### AC-006 — register-before-serve + VP-047 end-to-end

Tests: `TestVP047_SbctlPathsList_EndToEnd` (integration: pathTrackerSource populated with synthetic PathTracker → GET /paths → PathsListResponse with router_addr key present, empty string accepted per DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER), `TestDaemonRouterStatus_HandlerRegistered` (router.status handler registered and returns correct shape).

- [AC-006-register-and-vp047-e2e.gif](AC-006-register-and-vp047-e2e.gif)
- [AC-006-register-and-vp047-e2e.webm](AC-006-register-and-vp047-e2e.webm)
- [AC-006-register-and-vp047-e2e.tape](AC-006-register-and-vp047-e2e.tape)

## Error Paths Covered

| EC | Scenario | Covered By |
|----|----------|------------|
| EC-001 | No active paths → empty list + "no active paths" message | AC-001 (TestDaemonPathsList_EmptySource) |
| EC-003 | SampleCount<10 → rtt_p99_ms="pending" | AC-002 (TestPathEntry_RTTValueSerialization rows a,b) |
| EC-004 | SVTN not found → E-RPC-011 error envelope | AC-004 (TestDaemonRouterMetrics_SVTNNotFound) |
| EC-005 | Degraded=true → PathEntry.status="degraded" | AC-003 (TestPathEntry_StatusFromDegraded row active_degraded_is_degraded) |
| EC-006 | Degraded=true AND SampleCount<10 → status="degraded" AND quality="pending" | AC-005 (TestDaemonRouterStatus_QualityStatusIndependence row a) |
| EC-007 | SampleCount<10 → quality="pending" regardless of status | AC-005 (TestDaemonRouterStatus_QualityStatusIndependence row c) |
| EC-008 | len(paths)==0 → quality="pending" | AC-005 (TestRouterStatus_EmptyPaths_QualityIsPending) |

## Notes

- This is a library story (internal/metrics, internal/mgmt). There is no standalone CLI binary for these handlers — demos use `go test -v -run` to demonstrate each AC's test coverage directly.
- VP-047 integration test (AC-006) exercises the full handler→source→response code path; pathTrackerSource production wiring is deferred to S-BL.PATH-TRACKER-WIRING per Ruling-6.
- router_addr is "" (empty string) in all recordings per DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER; real host:port lands in S-BL.ROUTER-ADDR.
