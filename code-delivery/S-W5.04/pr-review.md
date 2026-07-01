# PR #41 Review — S-W5.04 daemon paths/metrics/router handlers

**Verdict: APPROVE**

**Branch:** `feat/S-W5.04-daemon-paths-metrics-handlers` → `develop`
**Story:** S-W5.04 v1.17 · Wave 6 · BC-2.06.001, BC-2.06.003, VP-047, VP-062
**Diff:** +3133 / −? across `internal/metrics/`, `internal/mgmt/`, `cmd/switchboard/`, and demo evidence

## Summary

Delivers the daemon-side half of `sbctl paths list`, `sbctl router metrics`, and
`sbctl router status`. Adds three new RPC handlers on the management server,
pure-core response types with a Kind-discriminated RTTValue union, an adapter
interface for path snapshot sources, and register-before-serve ordering to close
a documented data race. The PR is the direct output of 3/3 clean fresh 3-lens
adversarial passes and reflects extensive convergence work (v1.17 changelog
records 17 revisions).

## AC Coverage — all 6 covered

| AC | Covered by | Notes |
|----|-----------|-------|
| AC-001 | `TestDaemonPathsList_HandlerRegistered`, `TestDaemonPathsList_EmptySource`, `TestMetricsWire_PathsListRegistered` | EC-001 empty message asserted; wire-level test verifies production `wireMetricsHandlers` path (not just unit-level handler). |
| AC-002 | `TestPathEntry_RTTValueSerialization` (rows 0/9/10/100), `TestRTTValue_RoundTrip`, `TestRTTValue_JSONShapeExact`, `TestRTTValue_UnmarshalJSON_NullRejected`, `TestRTTValue_UnmarshalRejectsNegative`, `TestRTTValue_UnmarshalRejectsNonPendingStringTokens_{NaN,Inf}` | Kind enum (`PendingKind`/`FloatKind`) discriminates float64(0) from `"pending"`. Round-trip stability enforced at byte level (guards against 42.0 → 42.000...1 lossy decode). |
| AC-003 | `TestPathEntry_StatusFromDegraded` (incl. `active_false_is_degraded` row per Ruling-9), `TestPathEntry_StatusEnumClosed` (all 4 Active×Degraded combos), `TestPathsList_DiscriminatingStatusOracle` | Defensive `panic()` in `PathEntryFromSnapshot` if status escapes {active, degraded}. `"failed"` never emitted. |
| AC-004 | `TestDaemonRouterMetrics_HandlerRegistered`, `TestDaemonRouterMetrics_SVTNNotFound`, `TestRouterMetrics_MalformedArgsDecode`, `TestRouterMetrics_MissingRequiredSVTN` | E-RPC-011 sentinel with `errors.Is` chain (no string matching). Correct E-RPC-002 vs E-RPC-003 split for decode vs semantic-validation errors. |
| AC-005 / AC-005a | `TestDaemonRouterStatus_HandlerRegistered`, `TestDaemonRouterStatus_RedBand`, `TestDaemonRouterStatus_QualityStatusIndependence` (4 rows), `TestQualityFromEntry_*` (pending precedence, never-emits-failed), `TestRouterStatus_EmptyPaths_QualityIsPending` (EC-008), `TestEC006_DegradedAndPendingRow`, `TestEC002_AllPathsPending`, `TestOverallQuality_MixedPathsPrecedence` (30× iterations) | Status/quality independence proven per Ruling-4. EC-007/008 pending precedence enforced. Multi-path precedence test defeats Go map-iteration randomization. |
| AC-006 / VP-047 | `TestVP047_SbctlPathsList_EndToEnd`, `TestVP047_FieldSwapOracle` | Real `mgmt.Server` + ADR-012 Ed25519 handshake + full dispatch loop. Uses actual `paths.PathTracker.Snapshot()` (not synthetic map). `router_addr` key-presence asserted, `""` accepted. Field-swap oracle rules out path_id ↔ router_addr collision. |

## Story-boundary compliance — clean

- `cmd/sbctl/` — unchanged
- `internal/paths/` — unchanged
- `internal/mgmt/server.go` transport core — unchanged; only `Register` method added to `mgmt.go` and a new `register_metrics.go` with the metrics-handler wiring
- `internal/metrics/types.go`, `handlers.go`, `handlers_test.go`, `integration_test.go`, `types_test.go` — all new
- `internal/mgmt/register_metrics.go` — new
- `cmd/switchboard/metrics_wire.go`, `metrics_wire_test.go` — new; `access.go` / `mgmt_wire.go` refactored to a two-phase `newMgmtServer` → `wireMetricsHandlers` → `serveMgmtServer` sequence

The one internal `internal/metrics/metrics_prop_test.go` diff removes a 4-line
"vacuously true" bailout from `TestProp_BC_2_06_001_GreenToRedSingleStep` —
tightening an existing property test, not weakening it. Non-scope-violating
cleanup.

## Key design choices that read as strong

1. **Kind enum over sentinel discrimination (F-P2L1-004).** `RTTValue` uses
   `PendingKind`/`FloatKind` rather than treating `SampleCount==0` as pending.
   This resolves the `float64(0)` vs `nil` ambiguity that a naive sentinel-based
   union cannot handle after a JSON round-trip. `SampleCount` on `RTTValue` is
   correctly documented as producer-side-only.

2. **Register-before-serve fence (F-P2L1-001).** `mgmt.Server.serving atomic.Bool`
   is stored at the top of `Serve`; `Register` reads it and returns an error if
   called post-`Serve`. This is enforced at the type level and verified by
   `TestRegister_AfterServeReturnsError`. The runControl / runAccess paths use
   the explicit three-phase `newMgmtServer` → `wireMetricsHandlers` →
   `serveMgmtServer` sequence.

3. **Adapter-source pattern for future wiring.** `PathsListSource` and
   `RouterMetricsSource` are narrow interfaces; production `pathTrackerSource`
   is a map-backed adapter with the `#DEFERRED: S-BL.PATH-TRACKER-WIRING`
   comment explicitly at the field site as the story requires. Handlers never
   see `paths.PathTracker` directly — only `paths.PathSnapshot` value copies.

4. **E-RPC error taxonomy.** Distinct sentinels for decode failures
   (`ErrDecodeArgs`/E-RPC-002), semantic validation (`ErrInvalidParams`/E-RPC-003),
   and SVTN-not-found (`ErrRouterSVTNNotFound`/E-RPC-011). All tests use
   `errors.Is` per go.md rule 3 — no string-matching.

5. **Ordered oracle discipline.** Tests pin exact expected values (not just
   "any valid quality" or "any non-empty status"), which kills mutation classes
   like active↔degraded swap and pending↔green misclassification.

## Non-blocking observations

- `emptyRouterMetricsSource` in `metrics_wire.go` returns E-RPC-011 for every
  SVTN by design until the counter store lands (deferred to
  S-BL.ROUTER-METRICS-STORE). Documented in comments; consistent with the
  pathTrackerSource deferred pattern.
- `PathEntryFromSnapshot` panics if the derived status escapes {active,
  degraded}. This is a defensive invariant per go.md — normally I'd flag a
  panic in a pure-core function, but here it's specifically guarding the
  reserved-`failed` regression path that the story spec explicitly forbids
  emitting until S-BL.PATH-FAILED-STATUS. `TestPathEntry_StatusEnumClosed`
  exercises all four Active×Degraded combos; the panic branch is unreachable
  under the current derivation and serves as a mutation-testing tripwire.
- `pathTrackerSource` in metrics_wire.go intentionally has no `sync.RWMutex`
  — comment cites `#DEFERRED: S-BL.PATH-TRACKER-WRITER` because the current
  wave has no writer path (the map is constructed once and read-only after
  Serve starts). This is consistent with the register-before-serve invariant
  and go.md rule 12; when the writer story lands the mutex must return.
- `startMgmtServer` is retained as a thin wrapper over the two-phase pair for
  legacy test callers. `//nolint:unparam` is annotated with a justifying
  comment. Clean.

## CI

Quality Gate, CodeQL, Analyze (go), Dependency Review, StepSecurity —
**all green**.

## Demo evidence

`docs/demo-evidence/S-W5.04/` contains 6 `.gif` + 6 `.webm` recordings (one
per AC) plus `evidence-report.md` mapping each AC to its BC trace, tests, and
EC coverage. Includes both positive-path and error-path recordings (EC-001,
EC-003, EC-004, EC-005, EC-006, EC-007, EC-008).

---

Nothing blocking. Strong story delivery — the convergence work shows in the
depth of test oracles and the care taken around the RTTValue union and the
register-before-serve invariant. Ship it.
