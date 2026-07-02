---
artifact_id: W-6-wavegate-pass-3-Adv-A
document_type: wave-adversarial-review
scope: wave-gate-integration
wave: W-6
tranche: combined
stories: [S-BL.LOOKUP, S-W5.04, S-6.07, S-7.01, S-7.02, S-BL.ROUTER-ADDR, S-7.03, S-6.05]
lens: [L1]
develop_tip_claimed: "7fe3e29"
develop_tip_observed: "7fe3e29e4358df16e4e2f1de65a4e0d972540b4a"
pass_number: 3
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

# Adversarial Review — Wave-6 Combined Wave-Gate — Pass 3 — Adv-A (L1)

## Preflight

- `.git/HEAD` → `ref: refs/heads/develop`
- `.git/refs/heads/develop` → `7fe3e29e4358df16e4e2f1de65a4e0d972540b4a`
- basename(pwd) → `switchboard-blue`
- Prior-pass sidecars not read; fresh context.

## Scope

L1 correctness/coverage integration lens across 8 Wave-6 stories merged on develop@7fe3e29.

## Q1 — `--json` envelope contract — PASS

`cmd/sbctl/main.go:48-86` dispatch: svtn, sessions, paths, router, console, admin, version, ping. All route via `connectAndRun` → `writeSuccess`/`writeError` or through `router_status.go:119-236` own envelope path. `admin.go:273-275` post-print gated by `if !useJSON`. `admin.go:311,335` warnings/prompts to `sio.err`. `router_metrics.go:46`, `router_status.go:125-140` use `writeError` (JSON envelope on stderr). No stdout leaks under `--json`.

## Q2 — E-ADM code uniqueness — PASS

- E-ADM-006 → `cmd/switchboard/console_handlers.go:68,75,79`; `internal/session/auth.go:43`
- E-ADM-007 → `internal/session/auth.go:50`
- E-ADM-009 → `cmd/switchboard/admin_handlers.go:453,520,534,541,680,697`
- E-ADM-011 → `cmd/switchboard/admin_handlers.go:419`
- E-ADM-013 → `cmd/switchboard/admin_handlers.go:602`
- E-ADM-016 → `internal/routing/routing.go:89,200,215`
- E-ADM-018/019/020/021 → admin_handlers.go:400/405/413/415/417

Each code bound to single semantic. No accidental dual-use.

## Q3 — Admin dispatch table integrity — PASS

`BuildAdminHandlers` at `admin_handlers.go:119-134` — 6 pairwise-distinct commands (register/revoke/expire/list-keys/svtn.create/svtn.destroy). Sibling namespaces disjoint: console.* (3), paths.* (1), router.* (2). 12 commands total, no collisions.

## Q4 — AdmittedKeySet.Lookup value-return adoption — PASS

`admission.go:363-381` returns `(AdmittedKey, bool)` with deep-clone at 379. Callers via `LookupByPubkey`:
- `console_handlers.go:72` — `entry, found := ...`
- `svtnmgmt.go:383,458,499,520,648,680` — `entry, ok := ...`

No surviving pointer-return callers.

## Q5 — PathSnapshot.RouterAddr propagation — PASS

Chain: `NewPathTrackerWithAddr` (paths.go:128-132) → `t.routerAddr` (immutable per paths.go:60-65,123-125) → `Snapshot()` (paths.go:341) → `PathSnapshot.RouterAddr` (paths.go:321-325) → `PathEntryFromSnapshot` (handlers.go:65) → `PathEntry.RouterAddr` (handlers.go:160) → wire tag (types.go:76-77) → CLI (paths_list.go:27) → human render (router_status.go:99-103).

## Findings

None.

## Observations

**O-1** [LOW] — `cmd/switchboard/metrics_wire.go:59-63` `newPathTrackerSource` empty tracker map; deferred to `S-BL.PATH-TRACKER-WIRING` (Wave-7). RouterAddr chain (Q5) exercised only by integration tests until that story lands. Well-documented deferral, non-blocking.

**O-2** [LOW] — `admin_handlers.go:428` uses `E-INT-999` catch-all for unmapped SVTNManager sentinels; `E-INT-001` used at line 713 for non-duplicate Create failures. Semantically differentiated; numeric gap invites future audit confusion. Taxonomy hygiene note.

## Verdict

**CONVERGENT_L1.** 0/0/0/0 findings, 2 informational observations.
