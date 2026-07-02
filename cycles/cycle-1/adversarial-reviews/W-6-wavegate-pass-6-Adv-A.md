---
artifact_id: W-6-wavegate-pass-6-Adv-A
document_type: wave-adversarial-review
scope: wave-gate-integration
wave: W-6
tranche: combined
stories: [S-BL.LOOKUP, S-W5.04, S-6.07, S-7.01, S-7.02, S-BL.ROUTER-ADDR, S-7.03, S-6.05]
lens: [L1]
develop_tip_claimed: "7fe3e29"
develop_tip_observed: "7fe3e29e4358df16e4e2f1de65a4e0d972540b4a"
pass_number: 6
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

# Adversarial Review — Wave-6 combined wave-gate — Pass 6 attempt 1, Adv-A

## Preflight

- `.git/HEAD` = `ref: refs/heads/develop` — OK
- `.git/refs/heads/develop` = `7fe3e29e4358df16e4e2f1de65a4e0d972540b4a` — matches claimed prefix `7fe3e29`
- basename cwd = `switchboard-blue` — OK
- Prior-pass reviews NOT read (`prior_passes_read: false`)
- Reads: 5 files (main.go, admin.go, admin_handlers.go, paths.go, admission.go excerpt). Greps: 12. Under caps.

## Scope

L1 correctness/coverage integration review of Wave-6 combined wave-gate at develop@7fe3e29e. 8 stories: S-BL.LOOKUP, S-W5.04, S-6.07, S-7.01, S-7.02, S-BL.ROUTER-ADDR, S-7.03, S-6.05.

## Q1 — sbctl `--json` envelope contract PASS

All RPC-emitting paths under `cmd/sbctl/` route through `writeSuccess`/`writeError` (`main.go:95-123`) or via `connectAndRun` (`client.go:391`) which itself delegates to `writeSuccess`. Human-readable stdout emissions are all `!useJSON`-gated:

- `admin.go:274` `fmt.Fprintf(sio.out, "destroyed SVTN: %s\n", ...)` — gated on `!useJSON` (line 273). Confirmed spec-conformant per `interface-definitions.md:164` universal envelope contract.
- `router_status.go:235 formatPathsTable(sio.out, ...)` — reached only from `!useJSON` branch (line 227 else); JSON path returns at line 218 via `writeSuccess`.
- `router_metrics.go` — pure `connectAndRun`; no post-print.
- `console.go:66-151` — attach/detach/switch all return `connectAndRun` directly.

No stdout stray plaintext under `--json`. No dual-write regressions.

## Q2 — E-ADM-NNN taxonomy uniqueness PASS

Emit-site inventory (across `cmd/switchboard/admin_handlers.go`, `console_handlers.go`, `cmd/sbctl/client.go`):

| Code | Purpose | Emit-sites | Spec |
|------|---------|------------|------|
| E-ADM-006 | session/console authz | console_handlers.go:68/75/79 | error-taxonomy.md:77 |
| E-ADM-009 | insufficient authority | admin_handlers.go:453/520/534/541/680/697 | error-taxonomy.md:82 |
| E-ADM-010 | authentication failed | client.go:261, router_status.go:165 | mgmt-layer |
| E-ADM-011 | destroy unauthorized (Variant 2) + revocation-hierarchy | admin_handlers.go:419 | error-taxonomy.md:v2.9 (dual-purpose spec-sanctioned) |
| E-ADM-013 | key not found | admin_handlers.go:602 | error-taxonomy.md:86 |
| E-ADM-018 | control-to-control revoke requires confirm | admin_handlers.go:413 | error-taxonomy.md:v3.5 |
| E-ADM-019 | role mismatch | admin_handlers.go:400/405 | mgmt-layer |
| E-ADM-020 | bootstrap-revoke-forbidden | admin_handlers.go:415 | error-taxonomy.md:v3.3/v3.7 |
| E-ADM-021 | bootstrap-expire-forbidden | admin_handlers.go:417 | error-taxonomy.md:v3.8 |

E-ADM-011 (destroy + revocation-hierarchy) and E-ADM-016 (PATH-A/PATH-B HMAC) are the only dual-purpose codes and both are explicitly documented (error-taxonomy.md:v2.9 and :207 respectively). All other codes are pairwise-unique per purpose. Message-format single-source-of-truth held for every code.

## Q3 — admin dispatch table integrity PASS

`BuildAdminHandlers` (`cmd/switchboard/admin_handlers.go:119-134`) enumerates exactly 6 handlers with pairwise-distinct `Command` values:

1. `admin.key.register`
2. `admin.key.revoke`
3. `admin.key.expire`
4. `admin.key.list-keys`
5. `admin.svtn.create`
6. `admin.svtn.destroy`

No duplicates, no missing. Nil-safety:
- `m == nil` → `panic("BuildAdminHandlers: SVTNManager must not be nil (EC-004)")` fail-fast at wiring (line 121). Confirmed by `admin_handlers_e2e_test.go:721 TestControlMode_AdminHandlersRegistered`.
- `ops == nil` → normalized to `mgmt.NewOperatorKeySet(nil)` empty set (line 124). Prevents nil deref in `resolveAndVerifyCallerRole` operator-key bootstrap-grant path.

## Q4 — `AdmittedKeySet.Lookup` value-return signature PASS

Signature confirmed at `internal/admission/admission.go:363`:
```
func (s *AdmittedKeySet) Lookup(svtnID [16]byte, nodeAddr [8]byte) (AdmittedKey, bool)
```
Returns value copy with `PublicKey` deep-cloned via `append(ed25519.PublicKey(nil), entry.PublicKey...)` (line 379) — no aliasing into internal state. LookupByPubkey (line 390) delegates unchanged.

Call-site sweep across the tree (grep pattern `\.Lookup\(` under module scope) shows 15 call sites — all consume `(key, ok := ks.Lookup(...))` value-tuple destructuring. No survivor uses `key.Field` before checking `ok`; no survivor takes `&key`. Zero pointer-return legacy callers.

## Q5 — PathSnapshot.RouterAddr propagation and immutability PASS

Construction (single assignment site):
- `internal/paths/paths.go:130` — `t.routerAddr = addr` inside `NewPathTrackerWithAddr`. No exported mutator; no other `t.routerAddr =` assignment in the tree.
- `PathSnapshot` struct field `RouterAddr string` (line 325) is populated only in `Snapshot()` (line 341: `RouterAddr: t.routerAddr`). Immutability held per RULING-W6TB-B §3.

Propagation chain (verified end-to-end):
1. Construction — `NewPathTrackerWithAddr(addr, ...)` → `t.routerAddr = addr`
2. Snapshot emission — `PathTracker.Snapshot()` copies `t.routerAddr` verbatim to `PathSnapshot.RouterAddr`
3. Handler seam — `internal/metrics/handlers.go:65` calls `PathEntryFromSnapshot(pathID, snap.RouterAddr, snap)` and `:160` assigns `RouterAddr: routerAddr` into wire type
4. Wire type — `internal/metrics/types.go:77` `RouterAddr string \`json:"router_addr"\``
5. CLI type — `cmd/sbctl/paths_list.go:27` `RouterAddr string \`json:"router_addr"\`` (matching struct tag)
6. CLI rendering — `cmd/sbctl/router_status.go:103` includes `e.RouterAddr` in table row

Test corroboration: `internal/metrics/integration_test.go:376 TestVP047_RouterAddrNonEmpty` covers both Part A (handler-seam pass-through) and end-to-end enforcement via `routerAddrPattern.MatchString(entry.RouterAddr)`. Regexp-format oracle `^[^:]+:[0-9]+$` structurally validates host:port shape.

## Findings

None.

## Observations

- Verbatim propagation of `snap.RouterAddr` from `PathTracker` → wire → CLI is the only path where an empty string can surface at the CLI, and only when `NewPathTracker` (rather than `NewPathTrackerWithAddr`) is used — deliberate per paths.go:104-105 comment. Non-defect.
- `mapAdminError` default arm returns `E-INT-999: unmapped admin error` (admin_handlers.go:428) rather than crashing — belt-and-braces catchall consistent with Ruling-12 §1 (every handler error carries an `E-*` prefix). Non-defect.

## Verdict

CONVERGENT_L1. All 5 L1 questions PASS. Zero findings across critical/high/medium/low. Integration seams across S-BL.LOOKUP (value-return signature), S-BL.ROUTER-ADDR (immutable RouterAddr propagation), S-6.05/S-6.07 (admin dispatch), S-6.05 (E-ADM-011 dual-purpose destroy-authz), S-7.03 (console --json), and S-7.01/S-7.02/S-W5.04 (RPC envelope contract, error taxonomy) hold. Third consecutive clean pass eligibility: consistent with L1 correctness/coverage lens.

Verdict: CONVERGENT_L1 | critical=0 high=0 medium=0 low=0 | observations=2
