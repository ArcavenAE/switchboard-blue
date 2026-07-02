---
artifact_id: W-6-wavegate-pass-5-Adv-A
document_type: wave-adversarial-review
scope: wave-gate-integration
wave: W-6
tranche: combined
stories: [S-BL.LOOKUP, S-W5.04, S-6.07, S-7.01, S-7.02, S-BL.ROUTER-ADDR, S-7.03, S-6.05]
lens: [L1]
develop_tip_claimed: "7fe3e29"
develop_tip_observed: "7fe3e29e4358df16e4e2f1de65a4e0d972540b4a"
pass_number: 5
attempt_number: 1
sub_adversary: Adv-A
verdict: CONVERGENT_L1
findings:
  critical: 0
  high: 0
  medium: 0
  low: 0
observations: 2
reviewer_context: fresh
prior_passes_read: false
worktree_identity_tuple_verified: true
dispatch_integrity_failure: false
timestamp: 2026-07-02T00:00:00Z
---

# Adversarial Review — Wave-6 Wave-Gate — Pass 5 attempt 1 (Adv-A, L1)

## Preflight

- `.git/HEAD` → `ref: refs/heads/develop` — PASS
- `.git/refs/heads/develop` → `7fe3e29e4358df16e4e2f1de65a4e0d972540b4a` (matches claimed `7fe3e29`) — PASS
- `basename(pwd)` = `switchboard-blue` — PASS

Preflight passed; proceeded with review. Read cap: 6 (used 6). Wall clock well within budget.

## Scope

Perimeter-3 wave-gate integration cross-story seams for Wave-6 combined tranche (8 stories). L1 correctness/coverage only. Did NOT read prior review sidecars, STATE.md, or sprint-state.yaml.

## Q1 — sbctl `--json` envelope contract across all Wave-6 CLI paths

**Verdict: PASS.**

Dispatch table `cmd/sbctl/main.go:48-86` routes to five paths: `connectAndRun` (svtn/sessions/version/ping), `runPathsList` (delegates to `connectAndRun`), `runRouterMetrics` (delegates to `connectAndRun`), `runRouterStatus` (owns its own dispatch + injection path with explicit `writeSuccess`/`writeError` at `router_status.go:180, 218, 235`), `runConsole`, `runAdmin`. All specialized paths route through `writeSuccess`/`writeError` (`main.go:95-123`).

Argument-parse errors and pre-flight usage errors emit only to stderr (`main.go:35, 56, 63, 72, 84`; `admin.go:125, 136, 152, 161, 182-185`; `console.go:61, 72, 95, 98, 117, 143, 146`) which is acceptable — no stdout writes on the error path.

**One anomaly worth flagging as observation (not finding):** `cmd/sbctl/admin.go:273-275` in `runAdminSvtnDestroy` — after `connectAndRun` has already called `writeSuccess(false, data, sio)` (which writes the RPC response to stdout at `main.go:106`), the destroy path additionally writes `"destroyed SVTN: <name>\n"` to `sio.out` gated on `!useJSON`. The `--json` gate is present and correct (envelope is NOT corrupted). But in non-JSON mode this produces two stdout lines (raw daemon response + confirmation line). Peer commands (`svtn create`, `key register/revoke/expire`) do NOT do this. See Observation-1.

## Q2 — E-ADM-NNN error code taxonomy uniqueness

**Verdict: PASS.**

`.factory/specs/prd-supplements/error-taxonomy.md:70-92` enumerates E-ADM-001 through E-ADM-021 (contiguous, no gaps). Each row carries a distinct semantics. The two documented dual-purpose codes are explicit and load-bearing:

- **E-ADM-011** (line 84) — dual variant: "revocation hierarchy" (`ErrRoleMismatch`) and "destroy authorization" (`ErrDestroyUnauthorized`) — both marked as spec-sanctioned insufficient-privilege class; scope-disambiguated in body text.
- **E-ADM-016** (line 72) — dual variant: PATH-A "auth key unavailable" and PATH-B "tag mismatch" — both return `routing.ErrHMACVerificationFailed`; variant discrimination via message substring.

Both are the exact "legitimate spec-sanctioned dual-purpose codes" enumerated in the scope. No un-documented collisions found across E-ADM-NNN space.

## Q3 — Admin dispatch table pairwise-distinct + nil-safety

**Verdict: PASS.**

`cmd/switchboard/admin_handlers.go:119-134` — `BuildAdminHandlers` returns exactly 6 handlers:

| Command | Handler |
|---|---|
| `admin.key.register` | `makeRegisterHandler(m, ops)` |
| `admin.key.revoke` | `makeRevokeHandler(m, ops)` |
| `admin.key.expire` | `makeExpireHandler(m, ops)` |
| `admin.key.list-keys` | `makeListKeysHandler(m)` |
| `admin.svtn.create` | `makeAdminSVTNCreateHandler(m, ops)` |
| `admin.svtn.destroy` | `makeAdminSVTNDestroyHandler(m, ops)` |

All command strings pairwise-distinct. Nil-safety per go.md rule:

- Line 120-122 — `if m == nil { panic("BuildAdminHandlers: SVTNManager must not be nil (EC-004)") }` — fail-fast on misconfiguration.
- Line 123-125 — `if ops == nil { ops = mgmt.NewOperatorKeySet(nil) }` — graceful default for the OperatorKeySet.

## Q4 — `AdmittedKeySet.Lookup` value-return signature adoption

**Verdict: PASS.**

`internal/admission/admission.go:363` — `func (s *AdmittedKeySet) Lookup(svtnID [16]byte, nodeAddr [8]byte) (AdmittedKey, bool)` — value-return per go.md rule 12 (finding-032 store-sync contract leak).

The Grep of `\.Lookup\(` across `internal/admission/*` (18+ matches) shows all call sites use the value-return idiom `key, ok := ks.Lookup(...)` (e.g., `lookup_convention_test.go:61, 87, 167, 176, 187, 269, 335, 364, 467, 495, 538`; `lookup_admitted_whitebox_test.go:53, 74`; `admission_test.go:341`). `LookupByPubkey` at line 390-393 delegates to `Lookup` and inherits the value-return contract. No surviving pointer-return callers.

## Q5 — `PathSnapshot.RouterAddr` propagation chain + immutability

**Verdict: PASS.**

Chain traced end-to-end:

1. **Construction** (`internal/paths/paths.go:65, 128-132`) — `PathTracker.routerAddr` is a lowercase (unexported) field, set exclusively by `NewPathTrackerWithAddr` (line 130). `NewPathTracker` (line 106-117) leaves it as `""`. **Grep for `SetRouterAddr` / `SetAddr` returned zero matches** — no exported mutator exists. RULING-W6TB-B §3 immutability invariant confirmed at code level.
2. **Snapshot emission** (`internal/paths/paths.go:325`) — `RouterAddr string` field on `PathSnapshot`.
3. **Handler seam** (`internal/metrics/integration_test.go:376+`) — `TestVP047_RouterAddrNonEmpty` Part A verifies `PathsList` forwards `RouterAddr` from injected `PathSnapshot` through to JSON response.
4. **Wire type** — JSON tag `router_addr` on `PathEntry.RouterAddr` at `cmd/sbctl/paths_list.go:27`.
5. **CLI rendering** — `router_status.go:99-104` tabular output includes `ROUTER_ADDR` column; `e.RouterAddr` formatted with `%-22s`.

Immutability defense-in-depth confirmed via `internal/paths/paths_test.go:1183+` `TestBC_2_06_003_RouterAddr_ImmutableAfterConstruction` (line 1192) referenced in test file.

## Findings

**None at L1.** No critical, high, medium, or low findings. Wave-6 wave-gate integration seams are correctness-clean at the L1 lens.

## Observations

**Observation-1 (LOW — non-blocking UX):**

`cmd/sbctl/admin.go:273-275` — `runAdminSvtnDestroy` emits a client-side confirmation line `"destroyed SVTN: <name>\n"` to `sio.out` AFTER `connectAndRun` has already written the daemon RPC response (raw JSON bytes) to `sio.out` via `writeSuccess(useJSON=false, data, sio)` at `main.go:106`. In non-JSON mode this produces two stdout lines (daemon-response JSON + confirmation). Gated on `!useJSON`, so `--json` envelope is not corrupted — the invariant Q1 tests holds. Peer admin commands (`svtn create`, `key register`, `key revoke`, `key expire`) all return `connectAndRun` directly with no post-print. Comment at line 268-272 acknowledges this asymmetry as an F-P7L1-MED-1 fix. Not a defect; a UX asymmetry worth flagging for consistency review outside L1.

**Observation-2 (LOW — informational):**

Error-taxonomy documents two pre-existing E-CFG code collisions unrelated to Wave-6 stories (`error-taxonomy.md:100, 105`): E-CFG-002 collision (private-key export vs. BC-2.09.003 v1.2 listen_addr) and E-CFG-006 collision (sbctl admin `--yes`/`--confirm` vs. BC-2.09.003 v1.4 drain_timeout). Both are explicitly flagged as pre-Wave-5 legacy discrepancies with reconciliation tracked. Not in L1 Wave-6 scope; noted for completeness.

## Verdict

Wave-6 wave-gate integration seams are correctness-clean at the L1 lens. No critical, high, medium, or low findings.
