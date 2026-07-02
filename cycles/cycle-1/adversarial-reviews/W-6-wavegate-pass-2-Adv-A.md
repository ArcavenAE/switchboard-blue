---
artifact_id: W-6-wavegate-pass-2-Adv-A
document_type: wave-adversarial-review
scope: wave-gate-integration
wave: W-6
tranche: combined
stories: [S-BL.LOOKUP, S-W5.04, S-6.07, S-7.01, S-7.02, S-BL.ROUTER-ADDR, S-7.03, S-6.05]
lens: [L1]
develop_tip_claimed: "7fe3e29"
develop_tip_observed: "7fe3e29e4358df16e4e2f1de65a4e0d972540b4a"
pass_number: 2
attempt_number: 1
sub_adversary: Adv-A
verdict: CONVERGENT_L1
findings:
  critical: 0
  high: 0
  medium: 0
  low: 0
observations: 3
reviewer_context: fresh
prior_passes_read: false
worktree_identity_tuple_verified: true
dispatch_integrity_failure: false
timestamp: 2026-07-02T00:00:00Z
---

# W-6 Wave-Gate Adversarial Review — Pass 2 — Adv-A (L1)

## Preflight

- HEAD → `ref: refs/heads/develop` (verified via `.git/HEAD`).
- develop tip → `7fe3e29e4358df16e4e2f1de65a4e0d972540b4a` (matches claimed `7fe3e29`).
- Worktree basename → `switchboard-blue`.
- `prior_passes_read: false` — no `.factory/cycles/*/adversarial-reviews/`, `.factory/STATE.md`, or sidecar files opened.
- Read count: 6 / 6 cap. Grep-first discipline honored.

## Scope

Cross-tranche integration seams introduced by the 8 Wave-6 stories on `develop@7fe3e29`.
L1 lens only: correctness/coverage. Intra-story defects out of scope.

## Q1 — `--json` envelope contract across ALL Wave-6 subcommands

**Status: PASS**

Dispatch enumerated at `cmd/sbctl/main.go:48-86`:
- `svtn`, `sessions`, `paths list`, `router metrics`, `router status`, `console`, `admin`, `version`, `ping`.

Emission routing:
- `writeSuccess` (`cmd/sbctl/main.go:95-107`) — writes envelope to `sio.out`; plain data path only reached when `!useJSON`.
- `writeError` (`cmd/sbctl/main.go:111-123`) — writes envelope to `sio.err` on JSON, plain-text to `sio.err` otherwise. No stdout leak.
- All admin/console/paths/router paths funnel through `connectAndRun` (`cmd/sbctl/client.go:346-393`) which calls `writeSuccess`/`writeError`.
- `router_status.go:193-220` `useJSON` branch returns after `writeSuccess`; the plaintext `formatPathsTable(sio.out, …)` sink at `router_status.go:235` is only reachable in the `!useJSON` branch after the `if useJSON { … return nil }` early-return at line 193.
- `admin.go:273-275` `runAdminSvtnDestroy` explicitly guards its post-print with `if !useJSON`, keeping the JSON envelope pure (comment at 267-272 cites F-P7L1-MED-1).
- Confirm-gate warning + interactive prompt go to `sio.err` (`admin.go:311`, `admin.go:335`) — never stdout.

No trailing plaintext leaks in `--json` mode across the enumerated dispatch surface.

## Q2 — E-ADM taxonomy uniqueness (E-ADM-006, E-ADM-009, E-ADM-011)

**Status: PASS**

- **E-ADM-006** — single semantic (session authorization denied), attested at `.factory/specs/prd-supplements/error-taxonomy.md:77`. Consumers `internal/session/auth.go:43`, `cmd/switchboard/console_handlers.go` (attach/detach/switch), CLI trace text — all align.
- **E-ADM-009** — single semantic (insufficient authority for operation). Wave-6 extension in v4.0 broadened the FM/DEC Source to include BC-2.07.001 Inv-3 for `admin.svtn.create` (`.factory/specs/prd-supplements/error-taxonomy.md:25,82`). All 20+ emission sites in `cmd/switchboard/admin_handlers.go` (lines 453, 520, 534, 541, 680, 697) share the canonical prefix "insufficient authority for operation …".
- **E-ADM-011** — dual-semantic BUT explicitly ratified in the taxonomy: v2.9 changelog line at `error-taxonomy.md:14` records "E-ADM-011 extended with Variant 2 (destroy authorization); no new code slot allocated"; v3.1 at line 16 disambiguates the two contexts (SVTNManager.RevokeKey Go-API vs handler-gated destroy). Emission at `cmd/switchboard/admin_handlers.go:419` matches Variant 2. This is a spec-sanctioned reuse, not a collision.

No cross-code collisions detected. Cross-tranche additions (destroy variant, svtn.create Inv-3 source) are recorded in the taxonomy changelog.

## Q3 — Admin dispatch table integrity

**Status: PASS**

`BuildAdminHandlers` at `cmd/switchboard/admin_handlers.go:119-134` returns 6 entries:
1. `admin.key.register`
2. `admin.key.revoke`
3. `admin.key.expire`
4. `admin.key.list-keys`
5. `admin.svtn.create`
6. `admin.svtn.destroy`

Pairwise-distinct. Namespace collisions checked:
- `internal/mgmt/register_metrics.go:46-52` registers `paths.list`, `router.metrics`, `router.status` — disjoint namespace (`paths.*`, `router.*`).
- `cmd/switchboard/console_handlers.go:49-51` registers `console.attach`, `console.detach`, `console.switch` — disjoint (`console.*`).

No admin/paths/router/console overlaps.

## Q4 — `AdmittedKeySet.Lookup` new value-return signature adoption

**Status: PASS**

New signature at `internal/admission/admission.go:363`:
```
func (s *AdmittedKeySet) Lookup(svtnID [16]byte, nodeAddr [8]byte) (AdmittedKey, bool)
```
Deep-clones `PublicKey` (line 379) so returned copy is fully decoupled per go.md rule 12.

Callers grep-audited (`\.Lookup\(` restricted to `AdmittedKeySet` shape):
- `internal/admission/admission.go:392` — `LookupByPubkey` wraps and returns `(AdmittedKey, bool)`.
- `internal/admission/admission_test.go:341`, `lookup_convention_test.go` (10 call sites), `lookup_admitted_whitebox_test.go:53,74` — all use `k, ok := ks.Lookup(...)` form.

No surviving pointer-return callers.

## Q5 — `PathSnapshot.RouterAddr` propagation

**Status: PASS**

- **Constructor site (single):** `internal/paths/paths.go:65` — `routerAddr` field on `PathTracker` set only by `NewPathTrackerWithAddr` (BC-2.06.003 PC-1, RULING-W6TB-B §3).
- **Immutability documented:** `paths.go:62-64` — "Immutable after construction; set only by NewPathTrackerWithAddr."
- **Snapshot():** `paths.go:341` — `RouterAddr: t.routerAddr` (verbatim copy under `t.mu` lock).
- **PathSnapshot struct:** `paths.go:321-325` — field documented.
- **Handler:** `internal/metrics/handlers.go:65` — `PathEntryFromSnapshot(pathID, snap.RouterAddr, snap)` forwards `snap.RouterAddr` into `PathEntry.RouterAddr` (see `handlers.go:158-166`).
- **Wire type:** `internal/metrics/types.go:77` — `RouterAddr string \`json:"router_addr"\``.
- **CLI type:** `cmd/sbctl/paths_list.go:27` — `RouterAddr string \`json:"router_addr"\``.
- **Human-readable render:** `cmd/sbctl/router_status.go:99-103` — table column `ROUTER_ADDR`.

Immutability + concurrent-safe reads exercised by `internal/paths/paths_test.go:1372-1413` (post-probe/loss non-mutation + concurrent-snapshot tests). Handler-seam and end-to-end oracle at `internal/metrics/integration_test.go:376-497` (TestVP047_RouterAddrNonEmpty Parts A and B) validate ^[^:]+:[0-9]+$ shape from tracker through wire.

## Findings

None. No blocking or non-blocking L1 correctness defects surfaced.

## Observations

1. **[LOW / non-blocking]** `cmd/sbctl/e2e_helpers_test.go:191` registers a mock server handler named `"admin.key.list"` — this name does not match the production RPC name `admin.key.list-keys` registered at `cmd/switchboard/admin_handlers.go:130`. Because this is a helper file used only by tests and no shipped CLI dispatch uses `admin.key.list`, the mismatch does not affect production behavior. Worth confirming whether the helper reflects intent or is stale test-only shorthand. No fix required for wave-gate convergence.

2. **[LOW / non-blocking]** The sbctl-side `PathEntry` (`cmd/sbctl/paths_list.go:25-34`) uses field name `P99RTTMs any` while the daemon-side `PathEntry` (`internal/metrics/types.go:73-89`) uses `RTTP99Ms RTTValue`. Both marshal to the same JSON key (`rtt_p99_ms`), so the wire contract holds; the Go field-name divergence is cosmetic. Cross-tranche integration correctness is preserved via the JSON tag.

3. **[LOW / non-blocking]** `E-ADM-011` is a spec-sanctioned dual-semantic code (revocation-hierarchy Go-API path vs. destroy-authorization handler path). The taxonomy changelog (`error-taxonomy.md:14,16`) explicitly documents both variants and asserts non-overlap at runtime (E-ADM-009 handler gate fires before RevokeKey Go-API reaches its E-ADM-011 emission path). This is compliant with the current Q2 criterion, but the dual-purpose slot warrants monitoring in future wave-gates for accidental cross-variant leakage on new admin operations.

## Verdict

**CONVERGENT_L1**

All five L1 questions PASS with grounded file:line evidence. Three low-severity, non-blocking observations recorded. No cross-tranche integration defects surfaced against develop@7fe3e29.
