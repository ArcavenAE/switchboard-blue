---
artifact_id: W-6-wavegate-pass-1-Adv-A
document_type: wave-adversarial-review
scope: wave-gate-integration
wave: W-6
tranche: combined
stories: [S-BL.LOOKUP, S-W5.04, S-6.07, S-7.01, S-7.02, S-BL.ROUTER-ADDR, S-7.03, S-6.05]
lens: [L1]
develop_tip_claimed: "7fe3e29"
develop_tip_observed: "7fe3e29e4358df16e4e2f1de65a4e0d972540b4a"
pass_number: 1
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

# Wave-6 Wave-Gate L1 Adversarial Review — Pass 1 (Adv-A)

## Preflight

- `.git/HEAD` → `ref: refs/heads/develop`
- `.git/refs/heads/develop` → `7fe3e29e4358df16e4e2f1de65a4e0d972540b4a`
- worktree basename → `switchboard-blue`
- Worktree-identity tuple verified.

## Scope

L1 cross-tranche integration across 8 Wave-6 stories on `develop@7fe3e29`. Only cross-story
seams — not intra-tranche re-review.

## Q1 — Shared `--json` envelope contract

**Grounding evidence (`cmd/sbctl/main.go`):**
- Lines 93–106: `writeSuccess(useJSON, data, sio)` — canonical envelope.
- Lines 109–122: `writeError(useJSON, code, message, sio)` — canonical error envelope.

**Wave-6 subcommand routes through envelope:**
- `admin.svtn.create` (S-6.07) → routes via `connectAndRun` → `writeSuccess`/`writeError`
  (`cmd/sbctl/client.go:351, 366, 378, 381, 387, 391`).
- `admin.svtn.destroy` (S-6.05) → same path (`cmd/sbctl/admin.go:262`); post-print at
  `cmd/sbctl/admin.go:274` is correctly gated on `!useJSON` (line 273) with the
  interface-definitions.md contract-violation comment inline.
- `paths.list` (S-W5.04) → `writeSuccess`/`writeError` (via `router_status.go:180, 218`;
  human-readable branch line 235 fires only when `useJSON == false`).
- `router.metrics` (S-W5.04) → `writeError` for arg validation (`router_metrics.go:46`),
  then dispatches through the shared client pathway.
- `router.status` (S-W5.04) → `writeError` on every error branch (`router_status.go:126,
  138, 146, 159, 165, 171, 180, 189, 200, 208, 215, 218, 227`); the human-readable
  `formatPathsTable` call (line 235) is inside the `!useJSON` branch.
- `console.attach/detach/switch` (S-7.03) → tests confirm envelope conformance
  (`cmd/sbctl/console_test.go:118–140, 332–354`).

**Warnings / prompts / interactive I/O** in `admin.go:311, 335` correctly go to `sio.err`
(not `sio.out`) as required for `--json` mode.

**No trailing plain-text emission after a JSON envelope in any Wave-6 subcommand.** The
shared envelope contract holds.

## Q2 — Error-taxonomy collisions (E-ADM-011)

Grep of the entire tree for `E-ADM-011` yields exactly **four** files, and only **one
runtime emit site**:

- `cmd/switchboard/admin_handlers.go:419` — `E-ADM-011: destroy unauthorized`
  (S-6.05 destroy path, mapped from `svtnmgmt.ErrDestroyUnauthorized`).
- `internal/svtnmgmt/svtnmgmt.go:72, 746, 759` — documentation and sentinel definition
  referring to "E-ADM-011 Variant 2".

Grep for `Variant 1`, `ErrCreateUnauthorized`, or `create unauthorized` across the
entire repo yields **zero matches**.

**Actual S-6.07 create-side authorization path:**
`cmd/switchboard/admin_handlers.go:680, 697` — the create handler emits `E-ADM-009:
insufficient authority for operation admin.svtn.create`, NOT E-ADM-011.

**Conclusion:** No collision. E-ADM-011 has a single semantic (destroy-unauthorized).
The task premise ("both variants of E-ADM-011") is not realized in code. The
"Variant 2" phrasing in `svtnmgmt.go:72` implies a "Variant 1" that does not exist — see
Observations.

## Q3 — Admin dispatch table integrity

`cmd/switchboard/admin_handlers.go:119–134` — `BuildAdminHandlers` returns six unique
`mgmt.Handler{Command: ..., Fn: ...}` entries:
- `admin.key.register`
- `admin.key.revoke`
- `admin.key.expire`
- `admin.key.list-keys`
- `admin.svtn.create` (S-6.07)
- `admin.svtn.destroy` (S-6.05)

Names are pairwise distinct. `paths.list`, `router.metrics`, `router.status`, and
`console.*` register through separate builders (they are not in the admin family) and
are namespaced with a distinct top-level segment (`paths.`, `router.`, `console.`),
avoiding collision with the `admin.*` namespace.

## Q4 — AdmittedKeySet.Lookup value-return migration consumers

New signature (`internal/admission/admission.go:363`):
`func (s *AdmittedKeySet) Lookup(svtnID [16]byte, nodeAddr [8]byte) (AdmittedKey, bool)`
and `LookupByPubkey` at line 390 delegates.

**All Wave-6-touched consumers use the new value-return shape:**
- S-7.03 console handler: `cmd/switchboard/console_handlers.go:72` — `entry, found := ks.LookupByPubkey(zeroSVTN, callerPub)` ✓
- SVTN manager (touched by S-6.05 destroy + S-6.07 create): `internal/svtnmgmt/svtnmgmt.go:383, 458, 499, 520, 648, 680` — all use `entry, ok :=` or `_, ok :=` ✓
- Convention test enforcement: `internal/admission/lookup_convention_test.go` (multiple sites).

No surviving `*AdmittedKey`-single-return caller detected.

## Q5 — PathSnapshot.RouterAddr population across Wave-6

- `internal/paths/paths.go:121–125, 321–325, 341` — `PathSnapshot.RouterAddr` field
  populated verbatim from `t.routerAddr` in `Snapshot()`; documented invariant
  "RouterAddr is set at construction, immutable thereafter" (RULING-W6TB-B §3).
- `internal/metrics/handlers.go:65` — `paths.list` handler consumes
  `snap.RouterAddr` and passes it through `PathEntryFromSnapshot(pathID, snap.RouterAddr, snap)`.
- `internal/metrics/handlers.go:160` — direct `RouterAddr: routerAddr` field
  construction in metrics wire type.
- `cmd/sbctl/paths_list.go:27` — CLI PathEntry declares
  `RouterAddr string json:"router_addr"`, matching the wire contract.
- `cmd/sbctl/router_status.go:103` — `formatPathsTable` renders `e.RouterAddr` in the
  ROUTER_ADDR column.

**S-7.02 discovery:** RouterAddr does NOT appear in `internal/discovery/`. Session
discovery advertisements are semantically a different artifact class than PathSnapshots
(session names / HMAC advertisements, not per-path metrics). No integration seam here
requires RouterAddr in discovery frames. INSUFFICIENT_EVIDENCE downgrade avoided: the
task premise appears to have conflated advertisement classes — the actual scope of
S-BL.ROUTER-ADDR is the paths/router metrics surface, and every consumer in that
surface consumes RouterAddr correctly.

## Findings

_None. Zero critical / high / medium / low findings._

## Observations

- **O-1 (stale doc reference — svtnmgmt.go)**: `internal/svtnmgmt/svtnmgmt.go:72, 746, 759`
  refer to `ErrDestroyUnauthorized` as "E-ADM-011 Variant 2". This phrasing implies a
  "Variant 1" (create-unauthorized) that does not exist in the current codebase — the
  admin.svtn.create handler emits E-ADM-009, not E-ADM-011 (see
  `cmd/switchboard/admin_handlers.go:680, 697`). No runtime consequence; the wire
  taxonomy is single-semantic. Doc drift only. Do not block wave-gate; consider
  simplifying "Variant 2" to just "destroy unauthorized" in a future doc harvest.

- **O-2 (pre-parse usage errors bypass --json envelope — pre-Wave-6 pattern)**:
  `cmd/sbctl/main.go:35, 56, 63, 72, 84` emit usage errors via plain
  `fmt.Println` / `fmt.Fprintf(os.Stderr, ...)` before subcommand dispatch. In `--json`
  mode a scripted caller invoking an unknown subcommand or a subcommand missing its
  required positional arg receives non-JSON plaintext on stderr. This pattern is NOT
  introduced by any Wave-6 story — it predates the wave. Noted for future hygiene, not
  a wave-gate blocker.

## Verdict

**CONVERGENT_L1.** Zero blocking findings across Q1–Q5. All cross-tranche seams (envelope
contract, error-taxonomy, admin dispatch table, Lookup migration, RouterAddr propagation)
hold on `develop@7fe3e29`.

Novelty: LOW — the wave-gate integration surface is clean; observations are cosmetic /
predate the wave.
