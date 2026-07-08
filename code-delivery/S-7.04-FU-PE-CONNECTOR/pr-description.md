## Summary

Delivers the `internal/upstreamdial` package — the outbound TCP dial loop that closes the
connect-half deferred by S-7.04. When `runRouter` starts in PE mode, a `Connector` now
dials every address returned by `upstreamRoutersFor(cfg)`, performs a three-step session
bootstrap (TCP dial + `outerassembler.Assemble` + `conn.Write`), and maintains the
connected-count atomic that drives the live `Mode()` accessor (`ModeE` / `ModePE`).
Reconnect uses exponential backoff (base = `keepaliveInterval` floored at 500 ms, cap
30 s, jitter ±25 %). Address-list reconciliation is set-equal (reorders are no-ops). The
transitional testenv seams left by S-7.04-FU-SIGHUP-RELOAD (`SetSighupCh` /
`SendReloadSignal`) are retired; `RouterHandle.Mode()` now delegates to the live connector.
VP-038 (`E→PE via config-only`) verification lock flipped true. VP-037
(`drain-within-window`) partial discharge — harness present, skipped pending
`S-7.04-FU-DRAIN-WIRE`.

32-pass adversarial convergence cycle; 39 findings across 32 passes, streak P30/P31/P32
clean per BC-5.39.001.

Delivery doc: `.factory/stories/S-7.04-FU-PE-CONNECTOR-DELIVERY.md` v1.0

## Blast Radius

**1. Operator-visible surfaces touched:**

No change to the `sbctl` CLI surface, `--help`, `--version` output, config schema,
`paths.list` / `router.status` RPC schema, or wire protocol frame layout observable by
clients. `runRouter` now establishes outbound TCP connections to configured upstream
routers when started in PE mode; the resulting log emissions are internal diagnostic
lines — `"upstream router <addr> unreachable"` on EC-001 dial failure and
`"mode=E (no upstream_routers configured)"` on EC-004 all-upstreams-lost transition.
The `internal/testenv` package removes `SetSighupCh` and `SendReloadSignal`; all internal
callers are migrated in this PR. No change to `docs/getting-started.md` or the operator
error taxonomy. Bootstrap frames use `halfchannel.FrameTypeData` as a placeholder until
`S-BL.PE-RECEIVE-LOOP` defines the distinct `FrameTypePEConnect` constant (forward
obligation FO-PE-LOOP-001; no currently-connected receiver parses these frames).

**2. Silent-failure risk:**

Zero-Envelope deferral: `Connector` is constructed with zeroed `outerassembler.Envelope`
fields (`SrcAddr`, `DstAddr`, `SVTNID`, `FrameAuthKey`); upstream peers receive
unauthenticated bootstrap frames. This is documented in a comment in `mgmt_wire.go` and
tracked as Declared Divergence 2 (class: not-core). Full node-identity derivation (Ed25519
key material, HMAC key derivation) is deferred to the session-bootstrap follow-on story.
`FrameTypeData` bootstrap placeholder (Declared Divergence 3, FO-PE-LOOP-001): bootstrap
frames are structurally indistinguishable from session data at the receiver until
`S-BL.PE-RECEIVE-LOOP` defines the distinct frame-type constant. VP-037 drain harness is
present but `t.Skip`-ed; drain-and-migrate cannot be proven until `S-7.04-FU-DRAIN-WIRE`
ships the DRAIN broadcast. EC-004 duplicate-emission race under concurrent multi-upstream
drop-to-zero (F-P29-001) is fixed via single-atomic transition ownership
(`newCount := Add(-1)`); regression `TestConnector_ConcurrentDropToZero_SingleEC004Emission`
caught the race at 40–50% over 180 unfixed iterations and passes deterministically post-fix
under `go test -race`. No class of defect currently reachable by the test suite is left
uncovered within the shipped scope.

**3. Smoke gate touched:**

No. `just smoke-quick` sentinel invariants unchanged; no new `INV-*` id required, no
`test/smoke/invariants.sh` entry. The 7 VP e2e harnesses updated for the ctx-first
parameter-order migration (commit `670c64b`, go.md rule 7) include the currently
smoke-registered VPs (`vp033_034`, `vp036`, `vp037`, `vp038`, `vp039`, `vp040`,
`vp046`); all call-site changes are mechanical signature reorders with no assertion
change. Confirmed no new operator-boundary sentinel is needed.

## Changes

- **`internal/upstreamdial/connector.go` (NEW)** — `Connector` type, `Handle` interface
  (`ReloadAddrs([]string)`, `Mode() ConnMode`, `Stop()`), `ConnMode` enum; `dialLoop`
  goroutine per address; reconnect backoff with `operativeBase` pure function
  (keepaliveInterval floored at `BackoffBase`=500 ms); `atomic.Int32` connected-count;
  set-equal reconciliation inline in `Connector.reconcile`; `stopOnce sync.Once`
  idempotent `Stop()`; fast-path + non-blocking drain + non-blocking resend
  `ReloadAddrs`; EC-004 guarded by `ctx.Err() == nil`; single-atomic drop-to-zero
  ownership.
- **`cmd/switchboard/mgmt_wire.go`** — `runRouter` constructs `upstreamdial.New` at
  startup; SIGHUP case calls `connector.ReloadAddrs`; `connector.Stop()` in cleanup;
  `peConnectorHook` stub removed; `#DEFERRED` block split to `#SHIPPED` (PE-CONNECTOR)
  and preserved (DRAIN-WIRE).
- **`internal/testenv/testenv.go`** — Retires `SetSighupCh` / `SendReloadSignal` /
  `sighupCh` field / `os` + `syscall` imports; adds `SetConnector(upstreamdial.Handle)`;
  `RouterHandle.Mode()` delegates to live `connector.Mode()`; `Restart()` teardown-recreate
  contract; dynamic PE fixture listener replacing stub `"127.0.0.1:9999"`.
- **`internal/upstreamdial/connector_test.go` (NEW)** — 19 unit tests (AC-001..AC-006,
  backoff suite, EC-004 polarity, Stop idempotency, ReloadAddrs storm, concurrent drop-to-zero).
- **`cmd/switchboard/router_pe_connector_test.go` (NEW)** — 9 integration tests
  (AC-001..AC-006, VP-037 harness/skip, VP-038 harness).
- **`internal/testenv/testenv_test.go`** — +1 test `TestRouterHandle_Restart_TwicePE`
  (F-P2-001 Stop-idempotency regression).
- **`cmd/switchboard/router_sighup_test.go`** — `TestRunRouter_VP038_EtoPEViaConfigOnly`
  migrated from `SetSighupCh`/`SendReloadSignal` to inline `rawSighupCh <- syscall.SIGHUP`.
- **7 VP e2e test files** — Mechanical ctx-first signature migration per go.md rule 7
  (commit `670c64b`); 33 call sites; no behavioral change.
- **`cmd/switchboard/router_config.go`** — `upstreamRoutersAsSet` helper deleted
  (F-P1-007; set-equal logic moved inline to `Connector.reconcile`).
- **`.factory/specs/verification-properties/VP-038.md`** — `verification_lock` flipped
  `true`.
- **`.factory/specs/verification-properties/VP-037.md`** — `lifecycle_status`
  partial-discharge note (blocked on `S-7.04-FU-DRAIN-WIRE`).
- **`.factory/specs/architecture/ARCH-08-dependency-graph.md`** — `internal/upstreamdial`
  registered at DAG position 19; positions 19–22 renumbered to 20–23.
- **Demo evidence** — `demos/S-7.04-FU-PE-CONNECTOR/` (AC-001.tape, AC-002.tape,
  evidence-report.md; POL-004-compliant, no rendered binaries).

## Checklist

- [x] Tests added/updated (29 net-new + 1 migrated)
- [x] `just fmt` -- code is formatted
- [x] `just lint` -- zero warnings (golangci-lint 0 issues)
- [x] `just test` -- all tests pass (go test -race ./... green; 1 sanctioned skip VP-037)
- [x] `just smoke-quick` -- sentinel invariants pass locally (no sentinel change)
- [x] Commit messages follow conventional commits format
- [x] Blast radius block above answers all three questions (not "TBD")

## Testing

**Unit tests** (`internal/upstreamdial/connector_test.go`, 19 tests): dial success /
failure, EC-001 log verbatim, set-equal reorder no teardown, backoff constants + operative
base tracking (3 variants), backoff parameters reset-on-success wiring, all-upstreams-ModeE,
keepalive ticker health probe, reload adds/removes, `TestNextBackoff_*` arithmetic suite
(4 tests), EC-004 drop-to-zero emission, EC-004 graceful-Stop polarity guard, Stop
idempotency (sync.Once), ReloadAddrs storm (200 k iterations, 10 s watchdog),
concurrent drop-to-zero exactly-one-emission (F-P29-001, 40–50 % catch rate pre-fix).

**Integration tests** (`cmd/switchboard/router_pe_connector_test.go`, 9 tests):
AC-001 dial-and-connect; AC-001 set-equal reconciliation on reorder; AC-002 partial-PE
(one reachable + one unreachable); AC-003 keepaliveInterval reaches Connector;
AC-004 no-spurious-E-FWD-001 under normal load + F-P11-001 mutation pin; AC-006
RouterHandle.Mode() live-state tracking (F-P15-001 mutation-pinned); VP-038
E→PE-via-config-change (live Mode()); VP-037 drain harness (skipped, partial discharge).

**Migrated** (`router_sighup_test.go`): `TestRunRouter_VP038_EtoPEViaConfigOnly` — seam
retired to inline `rawSighupCh <- syscall.SIGHUP`; emission-based observable preserved.

All tests run under `go test -race ./...`. VP-037 skip is the single sanctioned exception
(partial discharge, blocked on `S-7.04-FU-DRAIN-WIRE`).

## Notes

**Declared Divergences** (from DELIVERY doc):

1. **AC-004 partial discharge (unmet-deps)** — E-FWD-001 exhaustion (postcondition 1) and
   S404-OBS-F / S404-LOW-1 re-anchored to `S-BL.PE-RECEIVE-LOOP`. Postcondition 2
   (no spurious E-FWD-001 under normal load) discharged here.

2. **Zero-Envelope Q6 deferral (not-core)** — Full node-identity derivation (Ed25519,
   HMAC) deferred to session-bootstrap follow-on. Three-step "connection established"
   definition is satisfied with zero envelope.

3. **FrameTypeData bootstrap placeholder (not-core, FO-PE-LOOP-001)** — Distinct
   `FrameTypePEConnect` constant deferred to `S-BL.PE-RECEIVE-LOOP` (consumer story).

4. **VP-037 partial discharge (unmet-deps)** — Drain-within-window harness present and
   skipped. VP-038 verification_lock flipped true.

**Forward obligation FO-PE-LOOP-001** registered in `S-BL.PE-RECEIVE-LOOP`: define
`frame.FrameTypePEConnect`; flip `dialLoop` bootstrap from `FrameTypeData` placeholder.

**Adversarial convergence:** 32 passes, 39 findings (doc-drift 14, process-gap 11,
test-fidelity 8, impl-defect 6), streak P30/P31/P32 clean (BC-5.39.001). Two notable
impl findings: F-P5-001 ReloadAddrs deadlock (blocking inner-receive in drop-oldest
pattern corrected to fast-path + non-blocking drain + non-blocking resend);
F-P29-001 EC-004 concurrent-drop race (single-atomic transition ownership).

**Upstream process issues filed:** drbothen/vsdd-factory #573 (normative-AC symbol
fidelity), #574 (placement-note derivation verification), #575 (concurrent-transition
coverage blindspot).

**Base advance:** `develop` advanced from merge-base `950285c` to `0fcd240` (PR #114 —
examples/ docker-compose ladder + CI workflow tweak; zero file overlap with this branch).
`gh pr update-branch` will be run after PR creation.

**VP-037/VP-038 anchor true-up** will be applied (VP-037.md / VP-038.md merge-SHA fields
updated) after merge.

**Dependency graph:**
- Depends on: S-7.04 (MERGED), S-7.04-FU-SIGHUP-RELOAD (MERGED), S-BL.OA (MERGED
  PR #96), S-BL.ARQ-TX (MERGED PR #98), S-BL.TESTENV (MERGED PR #110)
- Blocks: S-7.04-FU-DRAIN-WIRE (via S-BL.PE-RECEIVE-LOOP), S-BL.PE-RECEIVE-LOOP
