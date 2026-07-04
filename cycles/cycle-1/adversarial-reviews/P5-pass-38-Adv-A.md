---
pass: 38
lane: A
scope: public-surface + operator-UX
develop_head: 6deda15
factory_head_pre_review: 1ca13b4
verdict: NO_FINDINGS
findings_count:
  critical: 0
  high: 0
  medium: 0
  low: 0
observations: 1
reviewed_at: 2026-07-04
---

# Adversarial Review — Phase 5 Pass 38, Lane A (public-surface + operator-UX)

## Lens

SRE operator running `sbctl`; third-party integrator wiring against the JSON envelope; governance-doc consumer reading `wave-6-tranche-a-scope-rulings.md` and the S-6.07 story to understand what codes appear on the wire and what UX contract applies at the confirm gate. Governance-doc drift remains in scope for this pass under continuity with Passes 35 / 36 / 37 (the class the Burst 87 remediation targeted).

## Sweep Receipts

Files opened fresh this pass (absolute worktree-rooted paths):

- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/STATE.md` — L30 `develop_head: 6deda15`; Session Resume Checkpoint L197–L219 (Burst 89 = state-manager close-out of Pass 37, content-neutral).
- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/decisions/wave-6-tranche-a-scope-rulings.md` v1.14 — frontmatter (L5 version, L9 updated, L20 modified-list with all four DRIFT/F-P5P36 anchor citations), Ruling-11 §1 L1021, Ruling-11 AC-004 L1035, Ruling-12 §1 combined footnote L1120, Ruling-12 transport-exception L1129, Ruling-14 §10 v1.13 footnote L1423 (regression-check), v1.14 changelog row L1452.
- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/stories/S-6.07-svtn-admin-create.md` v1.14 — frontmatter (`version: "1.14"` L5, `last_modified: 2026-07-04T14:00:00` L9), §Universality amendment L78, body Changelog v1.14 row L212.
- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/specs/prd-supplements/error-taxonomy.md` v4.7 — RPC catalog rows L193–L197 (E-RPC-001, -002, -003, -010, -011); E-RPC-010 row L196 anchors the "surfaces directly as envelope code" claim.
- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/specs/prd-supplements/interface-definitions.md` v1.29 — §JSON Output Schema closed-set enumeration L233, §Registered Verbs table L403–L415, error-codes summary L419.
- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/stories/STORY-INDEX.md` — S-6.07 row L74 `(v1.14) merged (PR #42, 446efce)`, changelog row 3.80 L185 (POL-002 row-sync artifact).
- `/Users/skippy/work/aae-orc/run/switchboard-blue/cmd/sbctl/main.go` L100–L111 — `usageError` discrimination via `errors.As` → exit 2, other → exit 1.
- `/Users/skippy/work/aae-orc/run/switchboard-blue/cmd/sbctl/client.go` L95–L101 (jsonEnvelope shape), L198–L265 (Authenticate incl. E-RPC-002 stamp L214–L216, E-ADM-010 auth_fail L261), L278–L309 (dispatch incl. E-RPC-002 decode-limit stamp L300–L305), L385–L388 (E-RPC-001 top-level dispatch bucket).
- `/Users/skippy/work/aae-orc/run/switchboard-blue/cmd/sbctl/admin.go` L362–L403 — `runDestroyConfirmGate` covers all five paths (E-CFG-012 mutex, E-CFG-013 non-TTY, Path 1 valid confirm, Path 2 interactive TTY, Path 4 `--yes` warning).
- `/Users/skippy/work/aae-orc/run/switchboard-blue/internal/mgmt/mgmt.go` L678–L680 — server-side E-RPC-010 emission ("unknown command: <cmd>").
- `/Users/skippy/work/aae-orc/run/switchboard-blue/cmd/switchboard/admin_handlers.go` L458 (E-INT-999 unmapped-catch-all), L483/L550/L564/L571/L599/L621/L625 (E-ADM-009 insufficient-authority sites), L642 (E-SVTN-001 duplicate-name message format).

Prior-pass Adv-A sidecar consulted (per dispatch permission — prior pass only, not current-pass Adv-B):

- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/cycles/cycle-1/adversarial-reviews/P5-pass-37-Adv-A.md` (verdict NO_FINDINGS; 1 observation O-P5P37-A-001 flagging combined-footnote structural coupling at Ruling-12 §1 L1120).

Cross-searches performed:

- `E-RPC-004` across `.factory/` — 0 live spec hits. Both Burst 87 remediation sites (Ruling-12 §1 L1120 and S-6.07 L78) remain redirected to E-RPC-010 with dated amendment footnotes. No regression; no dangling phantom-code citation.
- `handler-not-found` across `.factory/` — 0 hits. Language-level regression check on the F-P5P36-A-001 remediation vector — the phrase does not reappear anywhere in the spec set under `wave-6-tranche-a-*` or the S-6.07 body.
- `DRIFT-P5P36-PHANTOM-ERPC-004` — present at Ruling-12 §1 L1120 (combined footnote), S-6.07 L78 (§Universality amendment), and `wave-6-tranche-a-scope-rulings.md` L20 (frontmatter modified-list).
- `DRIFT-P5P36-RULING-11-12-AUTHORSHIP-PREMISE-SIBLINGS` — present at Ruling-11 §1 L1021, Ruling-11 AC-004 L1035, Ruling-12 §1 L1120 (combined footnote), Ruling-12 transport-exception L1129, and L20 frontmatter modified-list.
- `F-P5P36-A-001` / `F-P5P36-A-002` — F-P5P36-A-001 cross-referenced at Ruling-12 §1 L1120 and S-6.07 L78; F-P5P36-A-002 cross-referenced at all four Ruling-11/12 amendment sites; both IDs cited in Ruling-12 §1's combined footnote and in the frontmatter modified-list.
- `^\| E-RPC` in `error-taxonomy.md` — 5 catalog rows (001, 002, 003, 010, 011); E-RPC-004 absent, consistent with the redirect (not mint) posture. Closed-set enumeration at `interface-definitions.md` L233 byte-parallel with these five rows.

## Burst-89 Continuity Verification

Burst 89 was state-manager close-out of Pass 37 per STATE.md Session Resume Checkpoint (content-neutral: no ruling edits, no story edits, no code edits). No risk vector for Pass-36 remediation regression. Explicit persistence check:

| Site | Result | Evidence |
|------|--------|----------|
| Ruling-11 §1 dated authorship-premise footnote | PASS | `wave-6-tranche-a-scope-rulings.md` L1021 — footnote intact, cites DRIFT-P5P36-RULING-11-12-AUTHORSHIP-PREMISE-SIBLINGS + F-P5P36-A-002 |
| Ruling-11 AC-004 dated authorship-premise footnote | PASS | L1035 — same pattern, intact |
| Ruling-12 §1 combined footnote (F-P5P36-A-001 + F-P5P36-A-002) | PASS | L1120 — both concerns preserved; both DRIFT anchors present; both finding IDs present; each concern lifted into its own explicit sentence with its own citation; dispatch-vs-transport precision paragraph intact |
| Ruling-12 transport-exception dated authorship-premise footnote | PASS | L1129 — footnote intact |
| S-6.07 §Universality amendment | PASS | `S-6.07-svtn-admin-create.md` L78 — E-RPC-004→E-RPC-010 redirect intact, cites DRIFT-P5P36-PHANTOM-ERPC-004 + F-P5P36-A-001 |
| Combined-footnote structural coupling (O-P5P37-A-001 pattern) | PASS | L1120 — both halves still fused as one parenthetical; neither concern silently shadowed by the other's language |

## Persistent Adv-A Baselines (all PASS)

- **CLI exit-code discrimination.** `cmd/sbctl/main.go` L100–L111 uses `errors.As(err, &ue)` to route `usageError` → `os.Exit(2)`, other errors → `os.Exit(1)`. Matches `interface-definitions.md` §CLI Exit-Code Contract.
- **jsonEnvelope shape.** `cmd/sbctl/client.go` L95–L101: `OK bool "ok"`, `Error *errorDetail "error"`, `Data json.RawMessage "data"`. No top-level `$schema`. Byte-parallel with `interface-definitions.md` §JSON Output Schema.
- **Confirm-gate 5-path.** `cmd/sbctl/admin.go` L362–L403: Path 5 (E-CFG-012 mutex, yes+confirm), Path 4 (warning, yes-only), Path 1 (valid confirm string), Path 3 (E-CFG-013 non-TTY), Path 2 (interactive TTY prompt). Matches `interface-definitions.md` §Confirm-Gate Paths.
- **Registered Verbs table.** `interface-definitions.md` L403–L415 lists nine verbs: paths.list, router.metrics, router.status, admin.key.register, admin.key.revoke, admin.key.expire, admin.key.list-keys, admin.svtn.create, admin.svtn.destroy. No orphan; no missing row.
- **Taxonomy ↔ envelope closed-set byte-parallel.** `interface-definitions.md` L233 enumerates E-NET-001, E-ADM-010, E-CFG-010, E-RPC-001, E-RPC-002, E-RPC-010, E-RPC-011, E-RPC-003 — all rows anchored in `error-taxonomy.md` v4.7 catalog. No phantom in the enumeration.
- **Wire-envelope discipline (Ruling-11 → Ruling-13 → Ruling-14).** CLI top-level `error.code` = `E-RPC-001` (dispatch bucket) at `client.go` L387; `E-RPC-002` stamped on `dispatch()` decode `io.ErrUnexpectedEOF` at L300–L305 and on Authenticate CHALLENGE-decode limit at L214–L216 (Ruling-14 runtime application). Server-side E-RPC-010 emission at `internal/mgmt/mgmt.go` L678–L680 supports Ruling-12 §1's "surfaces directly as envelope code" claim.
- **Handler emissions (server-side surface).** `cmd/switchboard/admin_handlers.go` — E-INT-999 unmapped-catch-all at L458; E-ADM-009 insufficient-authority at L483/L550/L564/L571/L599/L621/L625; E-SVTN-001 duplicate-name at L642. All rows present in `error-taxonomy.md` v4.7 catalog.

## Findings

None. Verdict is NO_FINDINGS.

**Anti-finding receipts (dispatched novelty targets):**

1. **Burst 89 transition — content-neutral.** STATE.md Session Resume Checkpoint L197–L219 records Burst 89 as state-manager close-out of Pass 37 with no content edits. Independent verification: every Burst-87 remediation site (four Ruling-11/12 dated footnotes plus S-6.07 L78 amendment plus Ruling-12 §1 combined footnote plus frontmatter L20 modified-list) is byte-consistent with the P5P37 sidecar's evidence. No regression vector.
2. **F-P5P36-A-001 remediation persistence.** Ruling-12 §1 L1120 combined footnote struck E-RPC-004 and redirected to E-RPC-010 with the dispatch-vs-transport precision paragraph preserved intact. S-6.07 L78 amendment intact. E-RPC-004 grep returns 0 live spec hits. `handler-not-found` phrase grep returns 0 hits. E-RPC-010 catalog row at `error-taxonomy.md` L196 continues to anchor the "surfaces directly as envelope code" claim.
3. **F-P5P36-A-002 remediation persistence.** All four sibling-sweep sites (Ruling-11 §1 L1021, Ruling-11 AC-004 L1035, Ruling-12 §1 L1120 combined footnote, Ruling-12 transport-exception L1129) preserve their dated authorship-premise citations. DRIFT-P5P36-RULING-11-12-AUTHORSHIP-PREMISE-SIBLINGS anchor present at all four sites plus frontmatter modified-list.
4. **POL-001 triangle byte-parallel — wave-6 rulings v1.14.** Frontmatter L5 `version: "1.14"`; L9 `updated: 2026-07-04T14:00:00`; L20 modified-list entry citing both DRIFT anchors and both finding IDs; changelog row v1.14 at L1452.
5. **POL-001 obligations — S-6.07 v1.14.** Frontmatter L5 `version: "1.14"`, L9 `last_modified: 2026-07-04T14:00:00`; body Changelog v1.14 row at L212. Story convention (`last_modified:` + body Changelog, no frontmatter `modified:` list) unchanged authorial practice, not a POL-001 gap.
6. **POL-002 story-index row-sync.** STORY-INDEX L74 shows S-6.07 `(v1.14) merged (PR #42, 446efce)`; changelog row 3.80 at L185 records the sync artifact.
7. **CLI-code baselines and emission-vs-taxonomy diff.** All persistent baselines (see section above) hold. Diff between server-side emissions (E-RPC-010 in mgmt server; E-INT-999 / E-ADM-009 / E-SVTN-001 in admin_handlers) and the taxonomy catalog rows shows byte-parallel — no orphan emission, no missing catalog row.
8. **Combined-footnote structural coupling.** L1120 preserves the fused parenthetical with both concerns visible and independently cited. The O-P5P37-A-001 concern (that a future amendment must preserve both halves as a package) is a design-protocol note, not a defect — the current shape is coherent and read-throughable.

## Observations

### O-P5P38-A-001 (persistence re-confirmation of O-P5P37-A-001 — non-defective, no novelty)

The combined-footnote structural coupling at `wave-6-tranche-a-scope-rulings.md` Ruling-12 §1 L1120 remains intact under the Burst-89 close-out (which was content-neutral by design). The same structural note recorded in P5P37 applies: a future adversarial pass working on a Ruling-12 amendment must preserve the fused shape — striking or refactoring one half of the footnote without the other would silently drop cross-references. This observation is carried forward without novelty, purely for continuity; no new information beyond the P5P37 record. Optional upstream capture (state-manager protocol addition to flag combined-footnote sites as "structurally coupled") remains deferred per standing directive.

## Novelty Assessment

**LOW.** Findings are receipts, not gaps. This is the second consecutive clean Adv-A pass. Every Burst-87 remediation vector (F-P5P36-A-001 phantom-code redirect at Ruling-12 §1 L1120 + S-6.07 L78; F-P5P36-A-002 sibling authorship-premise sweep at Ruling-11 §1, Ruling-11 AC-004, Ruling-12 §1 combined footnote, Ruling-12 transport-exception L1129) persists across the Burst-89 state-manager close-out, which by design touched no content. POL-001 triangles remain byte-parallel on `wave-6-tranche-a-scope-rulings.md` v1.14 and `S-6.07` v1.14; POL-002 story-index row-sync is satisfied at STORY-INDEX L74 + changelog row 3.80. All persistent Adv-A baselines (CLI exit-code discrimination, jsonEnvelope shape, confirm-gate 5-path, Registered Verbs table completeness, taxonomy-catalog byte-parallel, handler-emission coherence with catalog rows) hold. The one observation is a persistence-only re-confirmation of P5P37's already-recorded combined-footnote note — no new insight, no new evidence, carried purely for continuity.

Streak status under BC-5.39.001: this is Adv-A's second consecutive clean pass. One more clean pass required for convergence (3-of-3).

## Referenced BCs, Rulings, and Policies

- BC-5.39.001 (3-of-3 clean streak convergence)
- BC-5.39.002 (Adv-A / Adv-B lane isolation) — respected; no VP-INDEX, verification-coverage-matrix, coverage-architecture, or Adv-B current-pass reads
- Ruling-11 (wire envelope contract) — v1.6 + Burst-87 sibling authorship-premise footnotes at L1021, L1035
- Ruling-12 (universality of E-RPC-011 wrapping) — v1.0 + Burst-87 combined footnote at L1120 (F-P5P36-A-001 + F-P5P36-A-002); transport-exception footnote L1129
- Ruling-13 (CLI top-level `error.code` = E-RPC-001)
- Ruling-14 (dispatch response decode E-RPC-002) — regression-checked clean at §10 L1423 v1.13 footnote; no Burst-89 drift
- POL-001 (changelog format) — v1.14 triangle byte-parallel on both files
- POL-002 (story-index row-sync) — satisfied
- Partial-Fix Regression Discipline (S-7.01) — sibling-sweep axis clean across all four F-P5P36-A-002 sites

## Scope-Conformance Attestation

- Perimeter respected: public-surface + operator-UX only. No VP-INDEX arithmetic, no ARCH-11 matrix checks, no STORY-INDEX aggregate arithmetic, no policies.yaml structure reads, no sprint-state.yaml structure reads, no STATE.md frontmatter-transition adjudication, no BC-frontmatter POL-005/006/008 reconciliation.
- Adv-B current-pass sidecar not read. Prior-pass Adv-A sidecar consulted once, per dispatch permission.
- Read-only tools only: Read, Grep, Glob. No Write, Edit, or Bash. No shell mutation.
- Absolute worktree-rooted paths used throughout for feature-code evidence; canonical `.factory/` paths used for spec / ADR / BC / policy evidence.
