---
pass: 35
lane: A
lane_focus: public-surface + operator-UX
verdict: HAS_FINDINGS
findings_count:
  critical: 0
  high: 0
  medium: 1
  low: 0
observations: 2
reviewed_at: 2026-07-04
preflight:
  develop_head: 6deda15
  factory_head: 39b24f2
  state_phase_step: phase-5-taxonomy-remediation-complete
  awaiting: phase-5-pass-35-dispatch
  status: verified
streak_effect: hold-at-0/3 (HAS_FINDINGS)
---

# Pass 35 Adv-A — Public-Surface + Operator-UX Adversarial Review

## Verdict

HAS_FINDINGS. One MEDIUM governance-provenance finding — `wave-6-tranche-a-scope-rulings.md §10 (Ruling-14)` still preserves a factually-wrong historical premise about E-RPC-002 catalog membership at ruling authorship time. Burst 82 remediated the taxonomy but did NOT amend the ruling body, leaving a governance record that lies to any future reader.

Pass 34's REMEDIATED items all hold under fresh verification:
- E-RPC-002 and E-RPC-003 catalog rows are minted at `error-taxonomy.md` L194–L195 with prose matching emission sites byte-for-byte.
- E-RPC-010 forbidden clause at L196 is factually correct — `internal/mgmt/mgmt.go` grep returns zero E-RPC-002 emissions.
- `interface-definitions.md` v1.29 §JSON Output Schema L233 encloses a closed-set enumeration of `error.code` values.
- OBS-P5P34-A-001 (client.go:215 Authenticate bare-error path) verified stamped: L214-216 wraps `io.ErrUnexpectedEOF` as `"E-RPC-002: message too large: %w"`.

Streak arithmetic: 0/3 → 0/3 (one MEDIUM finding, at least one blocking finding present).

## Lens

Adv-A perimeter — operator-visible surfaces only. In scope: taxonomy rows visible via sbctl output, wire envelope semantics, CLI error paths, mgmt/metrics handler emissions that surface to operators, closed-set output schema. Governance documents (`decisions/*.md`) are in scope ONLY where a defect would mislead an operator reading them for troubleshooting or a future reviewer reconstructing the operator contract from the ruling record. The user's dispatch explicitly named `wave-6-tranche-a-scope-rulings.md §10` as an Adv-A novelty candidate under the "governance-text-vs-taxonomy" drift class — so it is in-scope for this pass.

Out of scope (Adv-B lane): symbol-graph purity, dependency-ordering, internal-only test-comment drift, structural code hygiene not touching operator output.

## Sweep receipts

1. Preflight verified against `.factory/STATE.md`: `phase_step: phase-5-taxonomy-remediation-complete`, `awaiting: phase-5-pass-35-dispatch`, `develop_head: 6deda15`, `factory_head: 39b24f2`. Match.
2. Fresh-read `error-taxonomy.md` v4.7:
   - L194 E-RPC-002 row: "Malformed or undecodable RPC args (JSON parse failure at handler-args decode) OR client-side transport-decode failure (bounded read exceeded via `io.ErrUnexpectedEOF`, ADR-012 §6)." Scope matches emission sites.
   - L195 E-RPC-003 row: "Structurally valid JSON decoded successfully, but a required semantic parameter is missing or invalid (e.g., empty `svtn_id`)." Scope matches emission sites.
   - L196 E-RPC-010 forbidden clause: narrows to reserve E-RPC-002 for client transport + handler-args-decode via E-RPC-011 wrap, forbids `internal/mgmt` server dispatch direct emission.
   - L233 v4.7 changelog row present, dated 2026-07-04.
3. Fresh-read `interface-definitions.md` v1.29 L208–L233 §JSON Output Schema: closed-set enumeration lines up with catalog. For sbctl top-level: E-NET-001, E-ADM-010, E-CFG-010, E-RPC-001, E-RPC-002. In-band via E-RPC-001 prefix: E-RPC-010, E-RPC-011. For router.metrics/router.status: E-RPC-002, E-RPC-003. Cross-checked against catalog membership — every enumerated code has a defined taxonomy row.
4. Fresh-read `internal/metrics/handlers.go`:
   - L26: `ErrDecodeArgs = errors.New("E-RPC-002: decode args")` — inspected via `errors.Is`.
   - L33: `ErrInvalidParams = errors.New("E-RPC-003: invalid params")`.
   - L79–L98 `RouterMetrics`: emits E-RPC-002 on `json.Unmarshal` failure (L87–L89) and E-RPC-003 on empty `svtn_id` (L95–L96). Prose comments explicitly cite the E-RPC-003 semantic-validation-vs-transport-decode distinction (L93–L94).
5. `cmd/sbctl/client.go`:
   - L214–L216: `Authenticate` stamps `"E-RPC-002: message too large: %w"` on `io.ErrUnexpectedEOF`.
   - L305–L307: `dispatch` stamps `"rpc failed: %s: E-RPC-002: message too large: %w"`.
6. Grep `E-RPC-002` in `internal/mgmt/mgmt.go` — zero hits. Grep `E-RPC-010` — L680. Grep `E-RPC-011` — L703. Validates the narrowed forbidden clause.
7. `wave-6-tranche-a-scope-rulings.md` §10 (Ruling-14) governance-text sweep — see finding.

## Cross-checks

- **Catalog membership per emitted code (Adv-A surface):** every operator-visible code path emits a taxonomy-defined code.
  - `cmd/sbctl/*.go` → E-NET-001, E-ADM-010, E-CFG-010, E-RPC-001, E-RPC-002 — all rowed.
  - `internal/mgmt/mgmt.go` → E-RPC-010, E-RPC-011, plus E-ADM-*/E-SVTN-*/E-CFG-*/E-INT-* handler codes carried in `error.message` per Ruling-11/Ruling-12 wire-envelope contract — all rowed.
  - `internal/metrics/handlers.go` → E-RPC-002 (via `ErrDecodeArgs`), E-RPC-003 (via `ErrInvalidParams`) — both rowed post-Burst-82.
- **Wire-envelope shape from operator PoV:** `{ok, error{code, message, field}, data}`. `$schema` absent from live output per v1.29 closed-set note. Consistent.
- **Ruling-11/12/14 wire-envelope discipline vs runtime:** transport-layer codes (E-RPC-002) surface as `error.code`; handler-layer codes wrap under E-RPC-011. Runtime code matches ruling.
- **Ruling-14 (§10) response-decode symmetry between Authenticate and dispatch:** both L215 and L306 stamp E-RPC-002 with `io.ErrUnexpectedEOF` triggers under bounded reads. Runtime consistent with the ruling's stated outcome.

## Findings

### F-P5P35-A-001 — MEDIUM — Ruling-14 §10 preserves factually-wrong governance premise about E-RPC-002 catalog membership

**Location:** `.factory/decisions/wave-6-tranche-a-scope-rulings.md` §10 (Ruling-14), specifically the "Impact assessment" table row at approx. line 1422 (the ruling was landed at v1.10 dated 2026-07-01; the document is now at v1.12 with subsequent rulings but §10 body is unamended).

**Evidence:** The §10 row reads:

> `| No BC change | E-RPC-002 is already defined in error-taxonomy.md; the fix applies the existing code to a missing branch |`

At the time Ruling-14 was authored (2026-07-01), E-RPC-002 was NOT catalog-defined in `error-taxonomy.md`. The catalog row was first minted in Burst 82's taxonomy remediation on 2026-07-04 (verified via `error-taxonomy.md` v4.7 changelog L233). Prior to that, E-RPC-002 was emitted at multiple emission sites (`cmd/sbctl/client.go:215` and `:306`, `internal/metrics/handlers.go` handler-args-decode) without a catalog row — the DRIFT-P5P34-TAXONOMY-ORPHAN-ERPC-002-003 finding that drove Burst 82.

**Why this matters for Adv-A:** the governance record is the operator contract's provenance trail. A future operator or reviewer reading Ruling-14 to reconstruct why E-RPC-002 has the scope it has today will conclude — incorrectly — that the code was catalog-defined at ruling authorship time and that Ruling-14 was merely a wire-decode-symmetry patch on top of an existing definition. In reality, Ruling-14 introduced the wire-decode-symmetry emission and the taxonomy definition lagged for three days. A reader trying to answer "when did E-RPC-002 become part of the operator-visible surface?" gets a wrong answer from the ruling.

The taxonomy was remediated in v4.7 but the ruling text was not amended by v1.11 (Ruling-1 SUPERSEDED-BY annotations) or v1.12 (Ruling-W6TB-G/H for S-7.02). §10 body preserves the false premise verbatim.

**Novelty class:** governance-text-vs-taxonomy drift — the very class the dispatch prompt named as an explicit novelty-focus candidate. This is the first-seen instance of the class in a Phase-5 pass.

**Remediation shape (for orchestrator/spec-steward, not enacted here):**
- Mint a v1.13 changelog row in `wave-6-tranche-a-scope-rulings.md` documenting the retroactive taxonomy alignment.
- Add a footnote or strike-through annotation on the §10 "Impact assessment" table row: `[Amended 2026-07-04: at ruling authorship (2026-07-01) E-RPC-002 was NOT catalog-defined; the catalog row was minted in Burst 82 (error-taxonomy.md v4.7) subsequent to Ruling-14 taking effect. The ruling's application of E-RPC-002 to the missing dispatch branch preceded the catalog row by three days — see DRIFT-P5P34-TAXONOMY-ORPHAN-ERPC-002-003.]`
- No BC or runtime change required; this is a spec-document-only fix.

**Confidence:** HIGH — file:line evidence for both the false premise (Ruling-14 §10 authored 2026-07-01) and the catalog absence at that time (error-taxonomy.md v4.7 changelog dates the row addition to 2026-07-04). No inference required.

## Observations

- **OBS-P5P35-A-001** (not blocking) — `internal/mgmt/mgmt_test.go` L1610–L1611, L1720–L1721, L1803 still carry stale comments referencing "the defective handler-error code E-RPC-002" as the pre-Ruling-C emission. These are internal test comments, not operator-visible surface — DEFERRED to Adv-B lane per BC-5.39.002 PC2 (internal-structural). No operator-visible defect. Filed here to prevent Adv-A recurrence.
- **OBS-P5P35-A-002** (not blocking) — `interface-definitions.md` v1.29 L233 closed-set enumeration is presented as a prose note. If the same enumeration is authoritatively repeated elsewhere in the doc (e.g., a table further down), the two representations will need sync discipline. Not currently a defect; noted as latent surface for future drift. `[process-gap]` candidate only if the enumeration ever gets duplicated.

## Novelty assessment

**MEDIUM novelty.** F-P5P35-A-001 is the first Phase-5 instance of the **governance-text-vs-taxonomy** drift class — a distinct class from prior taxonomy-orphan findings (which addressed emitted-code-without-catalog-row) and from prior forbidden-clause-scope findings (which addressed catalog-clause-vs-runtime-emission-set). The new class addresses **ruling-authorship-premise-vs-catalog-state-at-authorship-time** — governance provenance rather than runtime consistency.

This class is unlikely to have been detectable by earlier passes because it requires:
1. The taxonomy remediation to have already landed (Burst 82, 2026-07-04),
2. Fresh comparison of the ruling's dated authorship (§10 header 2026-07-01) against the catalog row's dated addition (v4.7 changelog 2026-07-04),
3. Recognition that retroactive taxonomy alignment does not amend historical ruling text.

Recommendation for orchestrator: consider adding a spec-steward review axis "governance-authorship-premise-consistency-with-catalog-state-at-date-of-authorship" for future adversarial cycles. Any ruling that asserts "X is already defined" against a taxonomy needs to be validated against the taxonomy's version-history at ruling date, not at review date.

Beyond this one finding, the Pass 34 remediation set holds cleanly under fresh read. The operator-visible surface — closed-set enumeration in v1.29, catalog membership for all emitted codes, wire-envelope contract discipline — is in strong shape.

## Referenced BCs, ADRs, and rulings

- BC-5.39.001 (loop mechanics), BC-5.39.002 (scope constraints) — perimeter contract.
- ADR-012 §6 — bounded-read semantics that motivate E-RPC-002 client-side transport-decode.
- Ruling-11 (§7) — wire envelope contract (E-RPC-011 wraps handler codes, E-RPC-002 surfaces as `error.code` transport-layer).
- Ruling-12 (§8) — universal handler-code coverage under E-RPC-011 wrap; E-RPC-002 as transport-layer exception.
- Ruling-14 (§10) — dispatch response-decode symmetry with Authenticate; **subject of F-P5P35-A-001**.
- DRIFT-P5P34-TAXONOMY-ORPHAN-ERPC-002-003 — Burst 82 taxonomy remediation providing the retroactive-alignment context for F-P5P35-A-001.

## Files referenced (absolute paths)

- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/STATE.md`
- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/specs/prd-supplements/error-taxonomy.md`
- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/specs/prd-supplements/interface-definitions.md`
- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/decisions/wave-6-tranche-a-scope-rulings.md`
- `/Users/skippy/work/aae-orc/run/switchboard-blue/internal/metrics/handlers.go`
- `/Users/skippy/work/aae-orc/run/switchboard-blue/internal/mgmt/mgmt.go`
- `/Users/skippy/work/aae-orc/run/switchboard-blue/cmd/sbctl/client.go`
