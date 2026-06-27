## Problem

Wave 3 integration gate finding F-1 (rated HIGH by two independent adversary passes,
passes 2 and 3): `RouteFrame` correctly dropped forged/unverifiable HMAC frames and
returned `ErrHMACVerificationFailed`, but never emitted the mandatory E-ADM-016 log
record. This left BC-2.05.008 PC-2's observability postcondition unimplemented and
untested — a P0 security contract gap on the wire-HMAC path.

**Affected behavioral contract:** BC-2.05.008 PC-2 (and PC-4 conservative extension)
**Error taxonomy entry:** E-ADM-016 — wire HMAC verification failed at RouteFrame
**Gate:** Wave 3 integration gate, F-1
**Severity:** HIGH (confirmed by adversary passes 2 and 3)

## Fix

Two signed commits off `develop` @ b68e498:

### c5d1c17 — feat(routing): add Logger injection and E-ADM-016 logging to RouteFrame

- Added `Logger` interface and `RouterOption`/`WithLogger` functional option to the
  `routing` package, mirroring the `tmux.Logger`/`tmux.WithLogger` pattern from
  `internal/tmux/pty_fallback.go`.
- `NewRouter` is now variadic-options (`NewRouter(ks, opts ...RouterOption)`) —
  fully backward compatible; existing callers pass no options and get `nopLogger`.
- `RouteFrame` now emits the canonical E-ADM-016 message before returning
  `ErrHMACVerificationFailed` on both failure paths:
  - **PATH-A** (no forwarding-table entry, auth key unavailable): `"wire HMAC
    verification failed at RouteFrame: auth key unavailable for SVTN <hex> from src
    <hex> (E-ADM-016)"`
  - **PATH-B** (tag mismatch from `verifyFrameHMAC`): `"wire HMAC verification
    failed at RouteFrame: tag mismatch for SVTN <hex> from src <hex> (E-ADM-016)"`
- Control flow unchanged. Success path does not log. Returned sentinel
  (`ErrHMACVerificationFailed`) unchanged.

### 3abe0b9 — test(routing): assert E-ADM-016 logged at RouteFrame on HMAC failure (BC-2.05.008 PC-2)

New test file `internal/routing/routing_log_test.go` with 4 tests:

| Test | Path | Assertion |
|------|------|-----------|
| `Test_BC_2_05_008_hmac_verify_fail_logs_eadm016` | PATH-B (tag mismatch) | 1 log record, contains E-ADM-016 + canonical message + svtn_id hex + src_addr hex |
| `Test_BC_2_05_008_zero_tag_logs_eadm016` | PATH-B EC-001 (zero tag) | same assertions |
| `Test_BC_2_05_008_no_entry_path_logs_eadm016` | PATH-A (no forwarding entry) | same assertions |
| `Test_BC_2_05_008_no_log_on_hmac_success` | success path | E-ADM-016 NOT logged (mutation resistance) |

## Test Evidence

All checks green on branch `fix/wave3-router-eadm016-logging`:

```
just test   — PASS (8 packages)
just lint   — 0 issues
go test -race ./internal/routing/... ./internal/session/... — PASS (race detector clean)
```

The 4 new tests in `routing_log_test.go` all pass. The mutation-resistance test
(`Test_BC_2_05_008_no_log_on_hmac_success`) confirms the success path emits no
spurious E-ADM-016 records.

> **Demo evidence policy:** per operator standing preference, VHS/terminal recordings
> are waived. Demo evidence for this fix is the test transcript above (new
> `routing_log_test.go` assertions passing under `just test` and `go test -race`).

## Traceability

```mermaid
flowchart LR
    BC["BC-2.05.008\n(wire HMAC enforcement)"]
    PC2["PC-2\n(log E-ADM-016 on\ntag mismatch)"]
    PC4["PC-4\n(log E-ADM-016 on\nno-entry path)"]
    E016["E-ADM-016\n(error taxonomy)"]
    PATHB["RouteFrame PATH-B\n(verifyFrameHMAC false)"]
    PATHA["RouteFrame PATH-A\n(entry == nil)"]
    T1["Test: hmac_verify_fail_logs_eadm016"]
    T2["Test: zero_tag_logs_eadm016"]
    T3["Test: no_entry_path_logs_eadm016"]
    T4["Test: no_log_on_hmac_success\n(mutation resistance)"]

    BC --> PC2
    BC --> PC4
    PC2 --> E016
    PC4 --> E016
    E016 --> PATHB
    E016 --> PATHA
    PATHB --> T1
    PATHB --> T2
    PATHA --> T3
    PATHB --> T4
```

```mermaid
graph LR
    F1["Wave 3 Gate F-1\n(HIGH — adversary passes 2+3)"]
    FIX["fix/wave3-router-eadm016-logging"]
    S304["S-3.04 (HMAC wire-up, merged #9)"]

    S304 -->|"depends on"| FIX
    F1 -->|"resolved by"| FIX
```

## Architecture Changes

The `Router` struct gains one field (`logger Logger`) and `NewRouter` accepts
variadic `RouterOption`. No exported type signatures removed. No control-flow
changes. No new dependencies (all stdlib + internal/frame, internal/hmac,
internal/admission — already imported).

```mermaid
graph TD
    ROUTER["Router\n+ logger Logger (NEW)\n+ NewRouter variadic opts (NEW)\n+ WithLogger(Logger) RouterOption (NEW)"]
    ROUTEFRAME["RouteFrame\n(unchanged signature)\n+ log E-ADM-016 on PATH-A (NEW)\n+ log E-ADM-016 on PATH-B (NEW)"]
    NOPLOG["nopLogger{}\n(default, silent — NEW)"]
    ADMKS["admission.AdmittedKeySet\n(unchanged)"]

    ROUTER --> ROUTEFRAME
    ROUTER --> NOPLOG
    ROUTEFRAME --> ADMKS
```

## Risk Assessment

- **Blast radius:** `internal/routing` only. No public API changes. `NewRouter`
  signature is backward compatible (variadic opts). All existing callers unaffected.
- **Performance:** nopLogger is a no-op struct with no heap allocation. Success path
  unchanged. HMAC-failure path (already returning an error) gains a single
  `fmt.Sprintf` call — negligible for an error path.
- **Security posture:** strictly additive. Improves observability on the HMAC
  rejection path with no change to the fail-closed enforcement logic.

## Pre-Merge Checklist

- [x] `just fmt` — gofumpt clean
- [x] `just lint` — 0 issues
- [x] `just test` — all 8 packages pass
- [x] `go test -race` — routing + session race-clean
- [x] No AI attribution in commits or PR body
- [x] All commits signed
- [x] BC-2.05.008 PC-2 postcondition covered by test
- [x] BC-2.05.008 PC-4 postcondition covered by test (conservative)
- [x] Mutation-resistance test confirms no spurious logging on success path
- [x] `NewRouter` backward compatible (variadic opts, existing callers unaffected)
- [ ] CI green (pending)
- [ ] PR review approved (pending)
