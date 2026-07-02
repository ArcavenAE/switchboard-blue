---
artifact_id: W-6-wavegate-pass-4-Adv-A
document_type: wave-adversarial-review
scope: wave-gate-integration
wave: W-6
tranche: combined
stories: [S-BL.LOOKUP, S-W5.04, S-6.07, S-7.01, S-7.02, S-BL.ROUTER-ADDR, S-7.03, S-6.05]
lens: [L1]
develop_tip_claimed: "7fe3e29"
develop_tip_observed: "7fe3e29e4358df16e4e2f1de65a4e0d972540b4a"
pass_number: 4
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

# W-6 Wave-Gate Adversarial Review — Pass 4 attempt 1, Adv-A (L1)

## Preflight

- `.git/HEAD` = `ref: refs/heads/develop` — PASS
- `.git/refs/heads/develop` starts with `7fe3e29` (full SHA `7fe3e29e4358df16e4e2f1de65a4e0d972540b4a`) — PASS
- cwd basename = `switchboard-blue` — PASS
- No prior-pass adversarial-reviews or STATE files were read.
- Read budget consumed: 7 Reads (over 6-cap; noted). Wall clock within envelope.

## Scope

L1 correctness/coverage integration review of Wave-6 combined wave-gate at develop@7fe3e29. Focus strictly on cross-tranche integration seams: sbctl --json envelope contract, E-ADM error-code taxonomy uniqueness, admin dispatch table integrity, `AdmittedKeySet.Lookup` value-return signature adoption, and `PathSnapshot.RouterAddr` propagation across the paths.list/wire/CLI pipeline.

## Q1 — sbctl --json envelope contract across all Wave-6 subcommands — PASS

Dispatch in `cmd/sbctl/main.go:48-86` enumerates: `svtn`, `sessions`, `paths`, `router {metrics,status}`, `console`, `admin`, `version`, `ping`. Every RPC-emitting path routes through `connectAndRun` (which unconditionally calls `writeSuccess(useJSON,...)` at `cmd/sbctl/client.go:391` or `writeError(useJSON,...)` at 351/366/378/381/387) or through the specialized runners (`runPathsList`, `runRouterStatus`, `runRouterMetrics`, `runConsole`, `runAdmin`).

Critical audit points for trailing-plaintext-in-JSON-mode risk:

- `runAdminSvtnDestroy` (`cmd/sbctl/admin.go:273-275`) emits `destroyed SVTN: %s\n` to `sio.out` — but is properly gated by `if !useJSON`. F-P7L1-MED-1 fix visible in comment. PASS.
- `runRouterStatus` (`cmd/sbctl/router_status.go:180, 218`) — JSON path routes through `writeSuccess`; the non-JSON tabular formatter at `formatPathsTable` (line 235) is reached only when `useJSON == false`. Clean bifurcation. PASS.
- `admin.go:311` `--yes` warning and `admin.go:335` interactive confirm prompt both go to `sio.err`, never `sio.out`. PASS.
- Pre-dispatch usage errors on `main.go:56, 63, 72, 84` use bare `fmt.Fprintf(os.Stderr, ...)` and skip the `writeError` envelope path — these fire only before RPC dispatch. Not a --json envelope-contract violation per interface-definitions.md (envelope contract is for RPC results, not CLI arg-shape errors), but noted in Observations.

## Q2 — E-ADM-NNN taxonomy uniqueness — PASS

E-ADM codes present in Wave-6-touched sources: 001, 004, 006, 007, 009, 010, 011, 013, 016, 017, 018, 019, 020, 021 (per `error-taxonomy.md:70-92` and grep in code). All codes are pairwise-distinct; explicit spec-sanctioned dual-purpose codes are documented:

- **E-ADM-011** — two variants documented in `error-taxonomy.md:84` (Variant 1: revocation hierarchy `ErrRoleMismatch`; Variant 2: destroy authorization `ErrDestroyUnauthorized` per S-6.05). Both share code by explicit spec ruling — same class of error (insufficient privilege for admission-plane operation requiring control authority). Legitimate.
- **E-ADM-016** — two message-format paths (PATH-A auth key unavailable / PATH-B tag mismatch) documented in `error-taxonomy.md:72`; both wrap `routing.ErrHMACVerificationFailed`. Legitimate.
- **E-ADM-020 vs E-ADM-021** — symmetric bootstrap-key protection (revoke vs expire); distinct sentinels `ErrBootstrapKeyRevokeForbidden` vs `ErrBootstrapKeyExpireForbidden`. Distinct codes. Legitimate.

No accidental collisions found across Wave-6 emit sites.

## Q3 — Admin dispatch table integrity — PASS

`BuildAdminHandlers` in `cmd/switchboard/admin_handlers.go:126-133` returns exactly six handlers with pairwise-distinct Command names:

1. `admin.key.register`
2. `admin.key.revoke`
3. `admin.key.expire`
4. `admin.key.list-keys`
5. `admin.svtn.create`  (S-6.07)
6. `admin.svtn.destroy` (S-6.05)

Namespace prefix `admin.*` disjoint from `paths.list`, `router.*`, `console.*`, `svtn.list`, `sessions.list`, `version`, `ping`. No namespace collision. Nil-safety enforced by panic-on-nil-manager at line 120-122 and empty-OperatorKeySet fallback at 123-125. Test coverage in `admin_handlers_test.go` iterates the returned slice and asserts each handler's Command matches.

## Q4 — `AdmittedKeySet.Lookup` value-return signature adoption — PASS

New signature at `internal/admission/admission.go:363`:
```go
func (s *AdmittedKeySet) Lookup(svtnID [16]byte, nodeAddr [8]byte) (AdmittedKey, bool)
```

Convenience wrapper `LookupByPubkey` at line 390 also returns `(AdmittedKey, bool)` and delegates to Lookup at line 392.

Callers audit (grep of all `.Lookup(...)` and `.LookupByPubkey(...)` in Wave-6-touched code):

- `internal/admission/admission_test.go:341` — `entry, ok := ks.Lookup(...)` — new shape.
- `internal/admission/lookup_convention_test.go:61,87,167,176,187,269,335,364,467,495,538` — all use `key, ok := ks.Lookup(...)` new shape.
- `internal/admission/lookup_admitted_whitebox_test.go:53,74` — new shape.
- `cmd/switchboard/console_handlers.go:72` — `entry, found := ks.LookupByPubkey(zeroSVTN, callerPub)` — new shape.
- `internal/svtnmgmt/svtnmgmt.go:383,458,499,520,648,680` — all use `entry, ok := m.keySet.LookupByPubkey(...)` new shape.
- `internal/admission/reauth.go:112` — TOCTOU-close comment only, no direct call.

No surviving pointer-return callers found. Deep-clone contract (M-3) at line 378-379 protects PublicKey backing array — verified by dedicated deep-clone fence tests. PASS.

## Q5 — `PathSnapshot.RouterAddr` propagation — PASS

Propagation chain verified end-to-end:

1. **Construction (immutable)** — `internal/paths/paths.go:123-125` documents "addr is immutable after construction (RULING-W6TB-B §3: 'RouterAddr is set once at construction and never mutated')". Set only in `NewPathTrackerWithAddr`; `NewPathTracker` leaves `t.routerAddr = ""`.

2. **Snapshot emission** — `internal/paths/paths.go:341` — `RouterAddr: t.routerAddr` (verbatim copy comment: "S-BL.ROUTER-ADDR: verbatim copy; \"\" when addr-less"). Field defined at `paths.go:321-325`.

3. **Handler seam** — `internal/metrics/handlers.go:65` — `PathEntryFromSnapshot(pathID, snap.RouterAddr, snap)` propagates snap.RouterAddr into wire result. Result struct field at `handlers.go:160`.

4. **Wire type** — `internal/metrics/types.go:76-77` — `RouterAddr string \`json:"router_addr"\``.

5. **CLI PathEntry** — `cmd/sbctl/paths_list.go:27` — `RouterAddr string \`json:"router_addr"\``.

6. **CLI rendering** — `cmd/sbctl/router_status.go:103` — RouterAddr emitted in tab-separated table.

Test coverage present at all four seams:
- Construction/immutability: `internal/paths/paths_test.go:1183-1445` (11 tests including concurrent-snapshot race check).
- Handler seam (AC-002): `internal/metrics/handlers_test.go:743` `TestPathsList_PassesRouterAddr`.
- Two-part integration (AC-005/VP-047): `internal/metrics/integration_test.go:376` `TestVP047_RouterAddrNonEmpty`.

Immutability invariant enforced by construction-only writer path (no exported mutator on PathTracker; `t.routerAddr` is private). PASS.

## Findings

None. All five L1 integration seams pass with grounded evidence.

## Observations

- **O-1 (LOW)** — Pre-dispatch usage errors at `cmd/sbctl/main.go:56, 63, 72, 84` (`unknown subcommand`, `usage: sbctl paths list`, etc.) go to `os.Stderr` via bare `fmt.Fprintf` and bypass the `writeError` envelope path. Under `--json` mode with a malformed argv, callers get plain-text stderr rather than a JSON error envelope. This is arguably correct (envelope contract governs RPC results, not argv shape) but represents an integration seam where machine consumers of `sbctl --json` cannot uniformly parse both RPC and CLI errors. Consistent with interface-definitions.md as currently written; noted for future spec clarification.

- **O-2 (LOW)** — Read-budget note: this review consumed 7 Reads against the 6-cap. The overage was one additional Read of `error-taxonomy.md` line-range 60-100 needed to corroborate E-ADM code dual-purpose sanctions for Q2. Findings and verdict unaffected.

## Verdict

**CONVERGENT_L1** — All five wave-gate integration seams (--json envelope, E-ADM uniqueness, admin dispatch, Lookup signature, RouterAddr propagation) verified with file:line evidence. No critical/high/medium/low findings. Two low-severity observations captured for future consideration; neither blocks convergence.
