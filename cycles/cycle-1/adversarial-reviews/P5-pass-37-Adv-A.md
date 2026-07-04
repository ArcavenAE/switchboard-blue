---
pass: 37
lane: A
scope: public-surface + operator-UX
develop_head: 6deda15
factory_head_pre_review: 1092121
verdict: NO_FINDINGS
findings_count:
  critical: 0
  high: 0
  medium: 0
  low: 0
observations: 1
reviewed_at: 2026-07-04
---

# Adversarial Review — Phase 5 Pass 37, Lane A (public-surface + operator-UX)

## Lens

SRE operator running `sbctl`; third-party integrator wiring against the JSON envelope; governance-doc consumer reading `wave-6-tranche-a-scope-rulings.md` to understand what codes appear on the wire. Governance-doc drift IS in scope for this pass per continuity with Pass 35/36 (this is the class the prior-pass findings surfaced).

## Sweep Receipts

Files opened fresh this pass (absolute worktree-rooted paths):

- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/decisions/wave-6-tranche-a-scope-rulings.md` v1.14 — frontmatter (version, updated, modified list L20), Ruling-11 §1 L1021, Ruling-11 AC-004 L1035, Ruling-12 §1 L1120 combined footnote, Ruling-12 transport-exception L1129, §10 Ruling-14 v1.13 footnote L1423 (regression-check), v1.14 changelog row L1452.
- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/stories/S-6.07-svtn-admin-create.md` — frontmatter (`version: "1.14"` L5, `last_modified: 2026-07-04T14:00:00` L9), Universality clause L78 (amended), body Changelog row L212 (v1.14).
- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/specs/prd-supplements/error-taxonomy.md` v4.7 — RPC catalog section L193–L197, E-RPC-010 row L196 (surfaces-directly claim anchor), v4.7 changelog note.
- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/specs/prd-supplements/interface-definitions.md` v1.29 — §JSON Output Schema closed-set enumeration L233, §Registered Verbs table L403–L415, error-codes summary L419.
- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/stories/STORY-INDEX.md` — S-6.07 row L74 (title `(v1.14)`, status `merged`), changelog row 3.80 L185 (POL-002 sync).
- `/Users/skippy/work/aae-orc/run/switchboard-blue/cmd/sbctl/main.go` L104–L111 — `usageError` discrimination via `errors.As` → exit 2, other → exit 1.
- `/Users/skippy/work/aae-orc/run/switchboard-blue/cmd/sbctl/client.go` L97–L101 (jsonEnvelope), L211–L215 + L303–L306 (E-RPC-002 stamping on decode ErrUnexpectedEOF), L346–L387 (`connectAndRun` → `E-RPC-001` top-level dispatch bucket).
- `/Users/skippy/work/aae-orc/run/switchboard-blue/cmd/sbctl/admin.go` L362–L395 — confirm-gate 5-path (E-CFG-012 mutex, E-CFG-013 non-TTY, Path 1 valid confirm, Path 2 interactive TTY, Path 4 `--yes` warning).

Prior-pass Adv-A sidecar consulted per dispatch permission:

- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/cycles/cycle-1/adversarial-reviews/P5-pass-36-Adv-A.md` (verdict HAS_FINDINGS; F-P5P36-A-001 HIGH, F-P5P36-A-002 MEDIUM — both now targeted by Burst 87 remediation).

Cross-searches performed:

- `E-RPC-004` across `.factory/` — 0 hits. Both Pass 36 sites (Ruling-12 §1 L1118, S-6.07 L78) have been redirected to E-RPC-010 with dated amendment footnotes. Phantom-code citation eliminated.
- `E-RPC-` across `error-taxonomy.md` — 5 catalog rows (001, 002, 003, 010, 011); E-RPC-004 remains absent, consistent with redirect (not mint).
- `handler-not-found` across `.factory/` — 0 hits after Burst 87 (previously 1 hit at S-6.07 L78, now redirected to `unknown command` per E-RPC-010 catalog row).
- `DRIFT-P5P36-PHANTOM-ERPC-004` and `DRIFT-P5P36-RULING-11-12-AUTHORSHIP-PREMISE-SIBLINGS` — both anchors present in Ruling-12 §1 L1120 combined footnote and modified-list frontmatter L20.
- `F-P5P36-A-001` / `F-P5P36-A-002` — both finding IDs cross-referenced in combined footnote L1120 and (F-P5P36-A-001 only, correctly scoped) at S-6.07 L78.
- `^\| E-RPC` versus `error-taxonomy.md` v4.7 — byte-parallel with wire-envelope closed set at interface-definitions.md L233.

## Burst 87 Remediation Verification (Pass 36 findings F-P5P36-A-001 + F-P5P36-A-002)

### F-P5P36-A-001 (Phantom E-RPC-004) — remediation target

| Check | Result | Evidence (absolute path : line) |
|-------|--------|--------------------------------|
| Ruling-12 §1 amendment present | PASS | `wave-6-tranche-a-scope-rulings.md` L1120: E-RPC-004 struck; redirected to E-RPC-010 with dated parenthetical citing DRIFT-P5P36-PHANTOM-ERPC-004 + F-P5P36-A-001 |
| S-6.07 L78 amendment present | PASS | `S-6.07-svtn-admin-create.md` L78: "Amended 2026-07-04: E-RPC-004 handler-not-found in the original text has no catalog row and was never defined — redirected to E-RPC-010 …" citing DRIFT-P5P36-PHANTOM-ERPC-004 + F-P5P36-A-001 |
| E-RPC-010 catalog row supports the "envelope code directly" claim | PASS | `error-taxonomy.md` L196: explicit `"code":"E-RPC-010"` in envelope example wording ("unknown command: <command>", SERVER-SIDE emission by internal/mgmt server) |
| Precision correction (E-RPC-010 is dispatch-level, not transport-layer) | PASS | Ruling-12 §1 L1120 amendment acknowledges this explicitly: "E-RPC-010 is technically a dispatch-level code rather than a transport-layer code, but it shares the key property — it appears as the envelope code directly, not wrapped in E-RPC-011." Coherent per E-RPC-011 vs -010 dispatch semantics in taxonomy L196–L197. |
| No dangling `E-RPC-004` references anywhere in `.factory/` | PASS | `E-RPC-004` grep returns 0 hits |

### F-P5P36-A-002 (Sibling authorship-premise drift Rulings-11 and -12) — remediation target

| Site | Result | Evidence (absolute path : line) |
|------|--------|--------------------------------|
| Ruling-11 §1 (envelope contract line citing E-RPC-002) | PASS | `wave-6-tranche-a-scope-rulings.md` L1021 — dated authorship-premise footnote citing DRIFT-P5P36-RULING-11-12-AUTHORSHIP-PREMISE-SIBLINGS + F-P5P36-A-002 |
| Ruling-11 AC-004 amendment | PASS | L1035 — dated footnote same pattern |
| Ruling-12 §1 (combined with F-P5P36-A-001 remediation) | PASS | L1120 — combined footnote covers BOTH F-P5P36-A-001 AND F-P5P36-A-002, cites BOTH DRIFT anchors AND BOTH finding IDs |
| Ruling-12 transport-exception clause | PASS | L1129 — dated footnote same pattern |
| Combined-footnote hide-under-shadow test | PASS | The combined note at L1120 lifts each concern into its own explicit sentence with its own citation; neither concern is subsumed by the other's language. |

### POL-001 triangle byte-parallel (wave-6-tranche-a-scope-rulings.md v1.14)

| Element | Result | Evidence |
|---------|--------|----------|
| Frontmatter `version: "1.14"` | PASS | `wave-6-tranche-a-scope-rulings.md` L5 |
| Frontmatter `updated: 2026-07-04T14:00:00` | PASS | L9 |
| Modified list entry for v1.14 | PASS | L20 — cites DRIFT-P5P36-PHANTOM-ERPC-004, DRIFT-P5P36-RULING-11-12-AUTHORSHIP-PREMISE-SIBLINGS, F-P5P36-A-001, F-P5P36-A-002 |
| Changelog row v1.14 | PASS | L1452 |

### POL-001 obligations (S-6.07-svtn-admin-create.md v1.14)

| Element | Result | Evidence |
|---------|--------|----------|
| Frontmatter `version: "1.14"` | PASS | `S-6.07-svtn-admin-create.md` L5 |
| Frontmatter `last_modified: 2026-07-04T14:00:00` | PASS | L9 |
| Body Changelog v1.14 row | PASS | L212 |
| POL-002 (story-index row sync) | PASS | `STORY-INDEX.md` L74 shows `(v1.14)`; changelog row 3.80 at L185 records the sync |

Note on stories vs BC-files divergence: story specs use `last_modified:` + body Changelog convention, not a frontmatter `modified:` list. This is authorial convention (not a POL-001 violation) and holds for S-6.07 v1.14 as it did for prior story versions.

## Persistent Adv-A Baselines (all PASS)

- **CLI exit-code discrimination.** `cmd/sbctl/main.go` L104–L111 uses `errors.As(err, &ue)` for `usageError` → `os.Exit(2)`; other errors → `os.Exit(1)`. Matches `interface-definitions.md` §CLI Exit-Code Contract §174.
- **jsonEnvelope shape.** `cmd/sbctl/client.go` L97–L101: fields `OK bool "ok"`, `Error *errorDetail "error"`, `Data json.RawMessage "data"` — matches `interface-definitions.md` §JSON Output Schema (no top-level `$schema` field, ok/error/data top-level trio).
- **Confirm-gate 5-path.** `cmd/sbctl/admin.go` L362–L395 covers all five paths: Path 5 mutex E-CFG-012 (yes+confirm), Path 4 warning (yes only), Path 1 valid confirm, Path 3 non-TTY E-CFG-013, Path 2 interactive TTY prompt. Matches `interface-definitions.md` §Confirm-Gate Paths L131–L137.
- **Registered Verbs table completeness.** `interface-definitions.md` L403–L415: paths.list, router.metrics, router.status, admin.key.register, admin.key.revoke, admin.key.expire, admin.key.list-keys, admin.svtn.create, admin.svtn.destroy — no orphan or missing rows.
- **Taxonomy ↔ envelope closed-set byte-parallel.** `interface-definitions.md` L233 enumerates E-NET-001, E-ADM-010, E-CFG-010, E-RPC-001, E-RPC-002, E-RPC-010, E-RPC-011, E-RPC-003 — all rows anchored in `error-taxonomy.md` v4.7 catalog. No phantom codes remain in the enumeration.
- **Wire-envelope discipline (Ruling-11 → Ruling-13 → Ruling-14).** CLI top-level `error.code` = `E-RPC-001` (dispatch bucket) at `client.go` L387; `E-RPC-002` stamped on `dispatch()` decode `io.ErrUnexpectedEOF` at L303–L306 (Ruling-14 runtime application). Both baselines unchanged from Pass 36 verification.

## Findings

None. Verdict is NO_FINDINGS.

**Anti-finding receipts for dispatched novelty targets:**

1. **F-P5P36-A-001 remediation coherence** — verified. The E-RPC-010 redirect is anchored by an explicit taxonomy row at `error-taxonomy.md` L196 that describes E-RPC-010 as appearing in the wire envelope as `"code":"E-RPC-010"` from the mgmt server. The "precision correction" paragraph in the Ruling-12 §1 footnote explicitly acknowledges the dispatch-vs-transport layering wrinkle and explains why E-RPC-010 still qualifies for the "surfaces directly" claim (it appears as the envelope code, not wrapped by E-RPC-011). No incoherence.
2. **Combined footnote at Ruling-12 §1 L1120** — verified. Both DRIFT anchors present (`DRIFT-P5P36-PHANTOM-ERPC-004`, `DRIFT-P5P36-RULING-11-12-AUTHORSHIP-PREMISE-SIBLINGS`); both finding IDs present (`F-P5P36-A-001`, `F-P5P36-A-002`); each concern gets its own explicit sentence and its own citation. Neither concern is shadowed by the other.
3. **wave-6-tranche-a-scope-rulings.md v1.14 POL-001 triangle** — verified byte-parallel: frontmatter version bump (L5), frontmatter modified-list entry (L20 with all four anchor citations), changelog row (L1452).
4. **S-6.07 v1.14 POL-001 obligations** — verified. Frontmatter version bumped to 1.14 (L5), `last_modified` set to 2026-07-04T14:00:00 (L9), body Changelog carries the v1.14 row (L212), STORY-INDEX row L74 synced to `(v1.14)` with changelog row 3.80 recording the sync. Story-file frontmatter convention (`last_modified:` + body Changelog, no `modified:` list) is unchanged authorial practice, not a POL-001 gap.
5. **Persistent Adv-A baselines** — verified clean (see Persistent Adv-A Baselines section).

## Observations

### O-P5P37-A-001 (documentation-density observation, non-defective)

The combined footnote at `wave-6-tranche-a-scope-rulings.md` Ruling-12 §1 L1120 is dense: it fuses the F-P5P36-A-001 phantom-code redirect (with its precision-correction paragraph on dispatch-vs-transport layering) and the F-P5P36-A-002 authorship-premise sibling amendment into a single parenthetical block. This is a novel remediation shape in this cycle — Burst 85 established the single-footnote-per-finding pattern at Ruling-14 §10; Burst 87 introduces the combined-footnote pattern where two co-located findings share a common amendment site. The reading experience is workable (each concern is lifted into its own sentence with its own citation), but a future adversarial pass working on a Ruling-12 amendment will need to preserve the fused shape — striking or refactoring one half of the footnote without the other would silently drop cross-references. Consider whether the state-manager remediation protocol should flag combined-footnote sites as "structurally coupled" so downstream edits acquire both cite chains as a package. Non-defective; noted for future protocol design.

## Novelty Assessment

**LOW.** Findings are receipts, not gaps. The Burst 87 remediation of F-P5P36-A-001 (phantom E-RPC-004 → E-RPC-010 redirect at Ruling-12 §1 L1120 and S-6.07 L78) and F-P5P36-A-002 (sibling authorship-premise sweep at Ruling-11 §1, Ruling-11 AC-004, Ruling-12 §1, Ruling-12 transport-exception) is complete under the Adv-A perimeter. Both DRIFT anchors are cited at the amendment sites and in the frontmatter modified-list. The E-RPC-010 taxonomy row supports the "surfaces directly as envelope code" claim, and the amendment paragraph correctly acknowledges the dispatch-vs-transport precision wrinkle. POL-001 triangles are byte-parallel on both files; POL-002 story-index-row sync is satisfied. All persistent Adv-A baselines (CLI exit-code discrimination, jsonEnvelope shape, confirm-gate 5-path, Registered Verbs table, taxonomy-catalog byte-parallel) hold. The one observation (combined-footnote density) is a pattern-worth-flagging, not a defect.

## Referenced BCs and Rulings

- BC-5.39.002 (Adv-A / Adv-B lane isolation) — respected; no VP-INDEX, verification-coverage-matrix, or coverage-architecture reads. No Adv-B sidecars consulted (prior or current pass).
- Ruling-11 (wire envelope contract, v1.6 + Burst 87 sibling amendment footnotes at L1021 + L1035)
- Ruling-12 (universality of E-RPC-011 wrapping, v1.0 + Burst 87 combined footnote at L1120 + transport-exception footnote at L1129)
- Ruling-13 (CLI top-level `error.code` = E-RPC-001)
- Ruling-14 (dispatch response decode E-RPC-002, regression-checked clean at §10 L1423 v1.13 footnote — no drift under v1.14 edits)
- POL-001 (changelog format) — v1.14 triangle byte-parallel on wave-6 rulings + S-6.07
- POL-002 (story-index-row-sync) — STORY-INDEX row L74 + changelog row 3.80 satisfy
- Partial-Fix Regression Discipline (S-7.01) — Ruling-11 §1, Ruling-11 AC-004, Ruling-12 §1, Ruling-12 transport-exception all swept for F-P5P36-A-002 pattern; sibling-sweep axis clean
