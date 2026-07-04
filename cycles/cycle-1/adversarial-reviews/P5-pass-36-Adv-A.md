---
pass: 36
lane: A
scope: public-surface + operator-UX
develop_head: 6deda15
factory_head_pre_review: d666607
verdict: HAS_FINDINGS
findings_count:
  critical: 0
  high: 1
  medium: 1
  low: 0
observations: 2
reviewed_at: 2026-07-04
---

# Adversarial Review — Phase 5 Pass 36, Lane A (public-surface + operator-UX)

## Lens

SRE operator running `sbctl`; third-party integrator wiring against the JSON envelope; governance-doc consumer reading `wave-6-tranche-a-scope-rulings.md` to understand what codes appear on the wire. Governance-doc drift IS in scope for this pass because the Pass 35 finding was in that class (per dispatch).

## Sweep Receipts

Files opened fresh this pass:

- `.factory/decisions/wave-6-tranche-a-scope-rulings.md` v1.13 — frontmatter, §1 Contract Table, Rulings-1 through W6TB-H bodies, §10 Ruling-14 remediation footnote (L1423), v1.13 changelog row (L1450).
- `.factory/specs/prd-supplements/error-taxonomy.md` v4.7 — RPC section L193–L197 (catalog rows E-RPC-001/002/003/010/011), v4.7 changelog note.
- `.factory/specs/prd-supplements/interface-definitions.md` v1.29 — §JSON Output Schema closed-set enumeration L233, v1.29 changelog note L143.
- `.factory/stories/S-6.07-svtn-admin-create.md` — Universality clause L78.
- `cmd/sbctl/client.go` L280–L330 — `dispatch()` E-RPC-002 emission (Ruling-14 runtime application).

Cross-searches:

- `^\| E-RPC` over `.factory/specs/prd-supplements/error-taxonomy.md` — 5 catalog rows (001/002/003/010/011); E-RPC-004 absent.
- `E-RPC-004` over `.factory/` — 2 hits: `decisions/wave-6-tranche-a-scope-rulings.md` L1118 and `stories/S-6.07-svtn-admin-create.md` L78.
- `handler-not-found` over `.factory/` — 1 hit: `S-6.07-svtn-admin-create.md` L78.
- `E-RPC-002` cross-doc — governance references at wave-6 ruling lines 1020, 1033, 1118, 1128 (all authored 2026-07-01, Ruling-11 v1.6 and Ruling-12 v1.0); catalog row minted Burst 82 (2026-07-04, error-taxonomy.md v4.7).

## Burst 85 Remediation Verification Checklist

Target: v1.12 → v1.13 remediation of F-P5P35-A-001 (Ruling-14 §10 authorship-premise footnote).

| Check | Result | Evidence |
|-------|--------|----------|
| Frontmatter `version: "1.13"` | PASS | L1 (frontmatter block) |
| Frontmatter `updated: 2026-07-04T12:00:00` | PASS | frontmatter |
| Modified list entry for v1.13 | PASS | L19 well-formed, cites DRIFT-P5P34-TAXONOMY-ORPHAN-ERPC-002-003 and F-P5P35-A-001 |
| §10 Ruling-14 Impact Assessment footnote | PASS | L1423, dated 2026-07-04, factually correct — acknowledges the runtime application was sound, corrects only the catalog-membership premise |
| Cross-reference to drift item + finding | PASS | inline in footnote |
| Changelog row for v1.13 (POL-001) | PASS | L1450, well-formed |
| Runtime impact | none (governance-only) | Confirmed against `cmd/sbctl/client.go` L300–L309 — E-RPC-002 emission on `dispatch()` `io.ErrUnexpectedEOF` is correct per Ruling-14 |

Verdict on Burst 85 target scope: **CLEAN.** The state-manager fused-burst pattern (Burst 84 + 85 consolidation) held under Adv-A scrutiny for the specific §10 remediation.

## Cross-Checks

- **Persistent-focus baseline E-RPC-002/003** — error-taxonomy.md L193–L195 catalog rows present. PASS.
- **Persistent-focus baseline E-RPC-010 forbidden clause scoping** — L196 "mgmt-server direct emission still forbidden" correctly scoped (does not spill into client CLI dispatch). PASS.
- **Persistent-focus baseline interface-definitions.md §JSON Output Schema closed set** — v1.29 L233 enumerates E-NET-001, E-ADM-010, E-CFG-010, E-RPC-001, E-RPC-002, E-RPC-010, E-RPC-011, E-RPC-003. Consistent with taxonomy. PASS.

## Findings

### F-P5P36-A-001 (HIGH) — Phantom E-RPC-004 citation with no catalog anchor

**Confidence:** HIGH — grounded in file:line evidence at three sites.

**Sites:**

1. `.factory/decisions/wave-6-tranche-a-scope-rulings.md` Ruling-12 §1 (~L1118): `Only transport-layer codes (E-RPC-002, E-RPC-004, etc.) surface as the wire envelope code directly.`
2. `.factory/stories/S-6.07-svtn-admin-create.md` L78: `Only transport-layer codes (E-RPC-002 args-decode, E-RPC-004 handler-not-found, etc.) surface as the envelope code directly without a prefix in message.`

**Defect:** E-RPC-004 (allegedly "handler-not-found") is cited in Ruling-12 §1 and the S-6.07 story spec as a transport-layer example, but **E-RPC-004 has never been catalog-defined in `error-taxonomy.md`**. v4.7 RPC section (L193–L197) contains only E-RPC-001, E-RPC-002, E-RPC-003, E-RPC-010, E-RPC-011. A fresh grep of the entire error taxonomy for `^\| E-RPC-004` returns zero matches. Neither the interface-definitions.md §JSON Output Schema closed-set enumeration (v1.29 L233) nor the S-6.07 enumerated table (L82–L89) includes it.

**Why it matters (public-surface / operator-UX lens):** A third-party integrator reading the governance ruling to build a client-side error handler for `E-RPC-004` (per the ruling's explicit invitation to treat it as a "transport-layer" code that "surfaces as the wire envelope code directly") will write dead code — the wire will never carry that code because no daemon path emits it and no catalog row backs it. This is strictly worse than the F-P5P35-A-001 authorship-premise pattern: F-P5P35-A-001 was a temporal drift (catalog row minted 3 days after ruling authored). E-RPC-004 has NO factual anchor at any point in time.

**Distinct class:** phantom-code citation. Governance references a code that has never existed. Not caught by Burst 85 (which was scoped to Ruling-14 §10). Requires a catalog decision (mint it — if handler-dispatch-miss is a real transport concern — or strike the citations).

**Suggested triage:** file DRIFT-P5P36-PHANTOM-ERPC-004 for state-manager routing. Amendment options: (a) mint E-RPC-004 in error-taxonomy.md if the handler-not-found bucket is architecturally meaningful and distinct from E-RPC-011; (b) strike both citations and let the daemon path for unknown-command continue to surface via existing codes; (c) redirect Ruling-12 §1 and S-6.07 L78 to cite E-RPC-010 (already catalog-defined and semantically closer to "unknown command"). Adjudication belongs to the ruling author.

---

### F-P5P36-A-002 (MEDIUM) — Sibling authorship-premise drift not swept in Burst 85

**Confidence:** HIGH — grounded in file:line evidence at three sites in the same document remediated for the same class in Burst 85.

**Sites (governance-doc, authored 2026-07-01; catalog row minted 2026-07-04 in Burst 82):**

1. `.factory/decisions/wave-6-tranche-a-scope-rulings.md` Ruling-11 v1.6 (~L1020): `- resp.Error.Code = envelope-level code (always E-RPC-011 for handler failures; other codes for transport/decode failures such as E-RPC-002).`
2. Ruling-11 AC-004 amendment (~L1033): `AC-004: same amendment pattern for E-RPC-002 transport errors (which DO surface as wire code).`
3. Ruling-12 §1 (~L1118, ~L1128): `Only transport-layer codes (E-RPC-002, E-RPC-004, etc.) surface as the wire envelope code directly.` / `E-RPC-002 is the transport-layer exception and is NOT wrapped.`

**Defect:** Same class as F-P5P35-A-001. At authorship-time (2026-07-01 per changelog rows for Ruling-11 v1.6 and Ruling-12 v1.0), E-RPC-002 was not yet catalog-defined. The Burst 82 catalog mint dated 2026-07-04 (error-taxonomy.md v4.7 changelog L28) was subsequent. Rulings-11 and -12 both cite E-RPC-002 as an established transport-layer code before the catalog row backed it, exactly parallel to the Ruling-14 §10 case that Burst 85 remediated.

**Why the sweep was incomplete:** Burst 85 remediation was scoped to Ruling-14 §10 (the specific F-P5P35-A-001 site). The sibling instances in Rulings-11 and -12 were not covered. The Partial-Fix Regression Discipline axis (S-7.01) requires siblings in the same architectural layer receive the same fix — Rulings-11, 12, and 14 are same-layer (all in `wave-6-tranche-a-scope-rulings.md`, all authored 2026-07-01, all citing E-RPC-002 pre-catalog). Blast radius: 4 sites (3 sibling ruling lines + AC-004 amendment). Per S-7.01 severity guidance: **HIGH** by blast-radius rubric (≥2 files/sites); reduced to **MEDIUM** here because runtime behavior is unaffected and the corrective pattern is well-established (dated audit-trail footnote — same shape as Ruling-14 §10 v1.13 amendment).

**Suggested triage:** amendment sweep on Rulings-11 and -12, mirroring the v1.13 §10 footnote pattern — inline dated parenthetical citing DRIFT-P5P34-TAXONOMY-ORPHAN-ERPC-002-003, or per-ruling footnotes.

---

## Observations

### O-P5P36-A-001 (protocol-discipline observation)

State-manager marked `DRIFT-P5P35-RULING-14-GOVERNANCE-PREMISE-STALE` CLOSED in STATE.md before the Adv-A verification pass ran. The specific §10 remediation is clean and the closure is retroactively validated for that scope. However, the underlying defect **class** (governance-text-vs-taxonomy authorship-premise) had unswept sibling sites in the same document (F-P5P36-A-002 above). The closure was scoped to the specific §10 instance rather than the class. Consider adding a "sweep-siblings-in-same-doc" gate to the drift-close protocol when the defect is a text/naming pattern rather than a localized code change — analogous to POL-002 (story-index-row-sync) applied to ruling-scoped citations.

### O-P5P36-A-002 (novelty-class observation)

Phantom-code citation (F-P5P36-A-001) is a first-seen class in this cycle. Distinct from authorship-premise drift (has-a-catalog-row-eventually) because the code never had a catalog row at any point. Suggest adding a governance-doc-vs-catalog-completeness axis to future adversarial passes: for every E-XXX-YYY reference in `.factory/decisions/**/*.md` and `.factory/stories/**/*.md`, grep-corroborate against `error-taxonomy.md` catalog rows. This axis would have caught E-RPC-004 in a single ripgrep.

## Novelty Assessment

**HIGH.** Pass 35 caught the F-P5P35-A-001 class (authorship-premise drift, Ruling-14 §10 instance). Pass 36 caught (a) three sibling instances of the same class in Rulings-11 and -12, and (b) a genuinely new class — phantom-code citation with no catalog anchor at any point in time — surfaced at two sites (Ruling-12 §1 and S-6.07 L78). Neither finding overlaps with what Pass 35 found; both are grounded in fresh grep evidence. The fresh-context compounding value axis held.

## Referenced BCs and Rulings

- BC-5.39.002 (Adv-A / Adv-B lane isolation) — respected; no VP-INDEX, verification-coverage-matrix, or coverage-architecture reads.
- Ruling-11 (wire envelope contract, v1.6, 2026-07-01)
- Ruling-12 (universality of E-RPC-011 wrapping, v1.0, 2026-07-01)
- Ruling-14 (dispatch response decode E-RPC-002, remediated v1.13 §10 2026-07-04)
- POL-001 (changelog format) — Burst 85 changelog row conforms
- Partial-Fix Regression Discipline (S-7.01) — sibling-sweep axis applied
