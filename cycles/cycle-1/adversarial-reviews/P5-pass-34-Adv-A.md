---
pass: 34
lane: A
scope: public-surface + operator-UX
develop_head: 6deda15
factory_head_pre_review: 1c5be1f
verdict: HAS_FINDINGS
findings_count:
  critical: 0
  high: 2
  medium: 0
  low: 0
  process_gap: 0
observations: 2
reviewed_at: 2026-07-04
---

# Adversarial Review — Phase 5 Pass 34 (Adv-A, public-surface + operator-UX)

## Verdict

**HAS_FINDINGS — 2 HIGH.** Both findings are taxonomy-orphan defects: production code emits
error codes E-RPC-002 and E-RPC-003 on observable operator paths, but neither code has a
catalog row in error-taxonomy.md v4.6. This is not a documentation-only gap — it is a
governance inconsistency with operator-facing consequences. Ruling-14 §10
(wave-6-tranche-a-scope-rulings.md v1.10, 2026-07-01) explicitly authorized E-RPC-002
emission on the premise "E-RPC-002 is already defined in error-taxonomy.md; the fix applies
the existing code to a missing branch." That premise is factually wrong: no E-RPC-002 catalog
row exists in v4.6. The taxonomy contains E-RPC-010 (the "unknown command" catch-all) with an
explicit "undefined and forbidden" prohibition on E-RPC-002 and other reserved codes —
which directly contradicts the Ruling-14 §10 authorization. The ruling was ratified against a
false factual premise and must be remediated by minting both rows and updating the E-RPC-010
forbidden-clause scope.

## Lens

SRE-operator (reading error codes from structured JSON output), third-party integrator
(parsing --json error envelopes against a documented taxonomy), public-surface (every code
emitted in production must be cataloged per S-W5.04 §Wire Envelope Contract).

## Sweep Receipts

**Full-file reads:**
- `.factory/specs/` error-taxonomy.md v4.6 — full catalog scan
- `.factory/specs/` interface-definitions.md v1.28 — §JSON Output Schema, §Wire Envelope
- `cmd/sbctl/main.go` — entry point, dispatch routing
- `cmd/sbctl/client.go` — full read; L215 (Authenticate path), L306 (another client error path)
- `cmd/sbctl/admin.go` — admin command dispatch, confirm-gate paths
- `internal/metrics/handlers.go` — full read; L26 sentinel (E-RPC-002), L33 sentinel (E-RPC-003)

**Partial reads:**
- `cmd/sbctl/admin_test.go:1490-1629` — test suite error-code discrimination; handlers_test.go:892-941
- `P5-pass-9-Adv-B.md:30-62` — prior sweep receipts for transport-decode layer
- `policies.yaml` — POL-001 through POL-008 current state

**Grep receipts:**
- `grep -rn "E-RPC-002" cmd/ internal/` → 3 hits: client.go:215, client.go:306, internal/metrics/handlers.go:26
- `grep -rn "E-RPC-003" cmd/ internal/` → 1 hit: internal/metrics/handlers.go:33
- `grep -n "E-RPC-002\|E-RPC-003" .factory/specs/behavioral-contracts/ss-*/BC-2.*.md` → E-RPC-003 appears in BC-2.07.004 as a test vector discriminator; E-RPC-002 appears in Ruling-14 §10 authorization text only
- `grep -n "E-RPC-002\|E-RPC-003" error-taxonomy.md` → ZERO catalog rows for either code; only E-RPC-010 row with "undefined and forbidden" language covering the reserved range

## Cross-Checks

| Perimeter | Status |
|-----------|--------|
| CLI dispatch paths | CLEAN — no new dispatch findings |
| Confirm-gate paths (5-path family) | CLEAN — exit-code discrimination correct |
| JSON envelope shape (§Wire Envelope Contract) | CLEAN — outer structure correct |
| Exit-code discipline | CLEAN — exit 1 on operator-error paths as specified |
| Wire-side E-RPC-010 (unknown command) | CLEAN — emission site correct |
| Wire-side E-RPC-011 (JSON envelope wrapper) | CLEAN — wrapper structure intact |
| Wire-side E-RPC-001 (internal server error) | CLEAN — emission site correct, catalog row present |
| Transport-decode E-RPC-002 | **DEFECT** — emitted at 3 sites, NO catalog row (F-P5P34-A-001) |
| Transport-decode E-RPC-003 | **DEFECT** — emitted at 1 site, NO catalog row (F-P5P34-A-002) |

## Findings

### F-P5P34-A-001 [HIGH] — E-RPC-002 taxonomy-orphan: emitted from 3 production paths, no catalog row

**Emission sites:**
- `cmd/sbctl/client.go:215` — Authenticate path; transport-layer decode failure
- `cmd/sbctl/client.go:306` — secondary client error path
- `internal/metrics/handlers.go:26` — ErrRPCTransportDecode sentinel, JSON error envelope via E-RPC-011 wrapper

**Taxonomy state in error-taxonomy.md v4.6:**
- E-RPC-002 has ZERO catalog rows. No description, no operator message, no class, no emission site list.
- E-RPC-010 row explicitly states: "All codes in the E-RPC-NNN reserved range not listed above (including E-RPC-002 through E-RPC-009) are undefined and forbidden."

**Ruling contradiction:**
Ruling-14 §10 (wave-6-tranche-a-scope-rulings.md v1.10, 2026-07-01) authorized E-RPC-002
emission with the stated premise: "E-RPC-002 is already defined in error-taxonomy.md; the
fix applies the existing code to a missing branch." This premise is factually wrong. The
taxonomy contains NO E-RPC-002 row. The E-RPC-010 "undefined and forbidden" clause directly
prohibits the emission Ruling-14 §10 authorized.

**Operator impact:**
An SRE or integrator parsing structured JSON error output from `sbctl` or the metrics daemon
will receive `"code": "E-RPC-002"` with no catalog entry to consult. Operator runbooks and
monitoring tooling that try to enumerate valid E-RPC codes from the taxonomy will miss this
code entirely. The E-RPC-010 "undefined and forbidden" language actively misdirects operators
who consult the taxonomy when encountering E-RPC-002.

**Remediation shape:**
1. Mint E-RPC-002 catalog row in error-taxonomy.md (description: transport-layer decode
   failure; class: RPC transport; emission sites: client.go:215, client.go:306,
   metrics/handlers.go:26 sentinel).
2. Update E-RPC-010 "undefined and forbidden" clause to scope-narrow the prohibited range,
   explicitly carving out E-RPC-002 and E-RPC-003 as now-defined codes.
3. Update interface-definitions.md §JSON Output Schema to enumerate the closed set of
   valid error.code values (OBS-P5P34-A-002 cross-reference).

---

### F-P5P34-A-002 [HIGH] — E-RPC-003 taxonomy-orphan: emitted from production path, no catalog row

**Emission site:**
- `internal/metrics/handlers.go:33` — ErrInvalidParams sentinel, JSON error envelope via E-RPC-011 wrapper

**Taxonomy state in error-taxonomy.md v4.6:**
- E-RPC-003 has ZERO references of any kind in error-taxonomy.md v4.6. Unlike E-RPC-002 (which
  at least appears in the Ruling-14 §10 text in the rulings document), E-RPC-003 is completely
  absent — not even a "forbidden" cross-reference.

**Test-suite signal:**
`cmd/sbctl/admin_test.go` (handlers_test.go:892-941) discriminates on E-RPC-002 vs E-RPC-003
as spec-intent, confirming both codes are treated as distinct observable outputs. The test
suite encodes the intent without the taxonomy ratifying it.

**Operator impact:**
Same class as F-P5P34-A-001. Operators receiving `"code": "E-RPC-003"` from the metrics
handler find zero documentation in the error catalog. The operator cannot distinguish
transport-decode (E-RPC-002) from invalid-params (E-RPC-003) without reading source code.

**Remediation shape:**
1. Mint E-RPC-003 catalog row in error-taxonomy.md (description: invalid request parameters;
   class: RPC protocol; emission site: metrics/handlers.go:33 sentinel).
2. Same E-RPC-010 scope-narrow as F-P5P34-A-001 (single edit covers both codes).
3. Same interface-definitions.md §JSON Output Schema closed-set enumeration.

---

## Observations

### OBS-P5P34-A-001 — client.go:215 Authenticate bypasses writeError under --json [LOW, deferred]

`cmd/sbctl/client.go:215` (Authenticate path) emits E-RPC-002 but does NOT route through
the `writeError` helper used by other JSON-output paths. Under `--json` mode, this path
writes its own envelope shape rather than using the shared formatter. The outer shape is
consistent with the §Wire Envelope Contract, but the pathway bypasses any future centralized
formatter evolution. Deferred — not a spec violation today, but a maintainability concern.

### OBS-P5P34-A-002 — interface-definitions.md §JSON Output Schema does not enumerate error.code closed set [LOW, deferred]

§JSON Output Schema in interface-definitions.md v1.28 describes the `error` object shape
(`code`, `message`, `details`) but does not enumerate the closed set of valid `code` values.
An integrator reading only the interface spec cannot determine which error codes are valid
without also reading the error taxonomy. The two documents are cross-referenced by the spec
but not joined by an explicit closed-set enumeration. Deferred pending Burst 82 taxonomy
remediation — the closed-set enumeration depends on the taxonomy being complete first.

---

## Novelty Assessment

**HIGH.** This finding class is genuinely novel after 34 adversarial passes. The specific
defect — production code emitting error codes not cataloged in the taxonomy — has not appeared
in any prior pass. More significant: Ruling-14 §10 (2026-07-01) was ratified with a factually
wrong premise ("E-RPC-002 is already defined"). The ruling was adopted 3 days before this
pass. Passes 32 and 33 (both clean) missed it because the specific cross-check
"does a ruling's 'already defined' premise match the actual taxonomy state" was not in the
Adv-A sweep protocol. This is a governance-verification gap, not a sweep-scope gap — the
taxonomy was read in prior passes, but the cross-check was not done against ruling premises.

Adv-A streak resets 2/3 → 0/3. The overall Phase 5 convergence criterion (BC-5.39.001:
three consecutive two-lane clean passes) resets; the streak cannot continue from a pass with
HIGH findings.

---

## Referenced BCs and Rulings

- error-taxonomy.md v4.6 — E-RPC-010 "undefined and forbidden" prohibition; absence of E-RPC-002/E-RPC-003 rows
- interface-definitions.md v1.28 — §JSON Output Schema, §Wire Envelope Contract
- wave-6-tranche-a-scope-rulings.md v1.10 §10 (Ruling-14, 2026-07-01) — E-RPC-002 authorization on factually wrong premise
- S-6.07 §Wire Envelope Contract — canonical wire-side error code emission requirement
- S-W5.04 — daemon paths error taxonomy anchor
- BC-2.07.004 — E-RPC-003 appears as test vector discriminator; no taxonomy anchor
- ADR-012 §6 — error taxonomy governance; all emitted codes must have catalog rows
