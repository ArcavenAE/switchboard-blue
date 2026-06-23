---
artifact_id: adv-p1-pass-02
review_target: phase-1-spec-crystallization
producer: adversary
pass: 2
fresh_context: true
findings_count: 18
findings_by_severity: {critical: 3, high: 8, medium: 6, low: 1}
findings_with_process_gap: 2
verdict: NOT_CONVERGED
timestamp: 2026-06-23
---

# Adversarial Review — Pass 2 (Fresh Context)

## Critical Findings

### F-001 — VP-057 specifies HKDF derivation that contradicts ARCH-04 (THREE different specs in one file)
- Severity: critical • Category: consistency / security
- Location: `.factory/specs/verification-properties/VP-057.md` lines 48–50, 188–197 vs `.factory/specs/architecture/ARCH-04-admission-security.md` lines 164–168
- Finding: VP-057 contains three mutually inconsistent statements of HKDF derivation for `frame_auth_key`, none matching ARCH-04 canonical:
  - ARCH-04 lines 165–168 (canonical): `HKDF-Extract(salt=svtn_id, ikm=node_admission_pubkey) → PRK; HKDF-Expand(PRK, info="switchboard-frame-auth", length=32) → frame_auth_key`
  - VP-057 line 49: `frame_auth_key = HKDF-SHA256(svtn_id || session_id, node_admission_pubkey)` — non-existent `session_id`, inverted salt/ikm
  - VP-057 lines 191–196: adds `session_id` to salt AND uses `info="frame-auth/v1"` (vs canonical `"switchboard-frame-auth"`)
- Route: architect
- Fix: Rewrite VP-057 §"Part (ii)" + inline proof sketch to quote ARCH-04 verbatim. Remove all `session_id` references. Use canonical `info` string.

### F-002 — E-ADM-007 is defined TWICE with different semantics
- Severity: critical • Category: consistency / governance
- Location: error-taxonomy.md:56 vs ARCH-04:98
- Finding: Same code `E-ADM-007` — error-taxonomy: "upstream rejected: read-only access" / BC-2.04.005. ARCH-04: "revocation by console-role on control-role rejected" / BC-2.05.004. Production code cannot dispatch deterministically.
- Route: product-owner
- Fix: Allocate E-ADM-011 for the permission-hierarchy rejection in ARCH-04:98. Add row to error-taxonomy.md. Leave E-ADM-007 dedicated to BC-2.04.005.

### F-003 — PRD §7 traceability matrix maps BC-2.05.006/007 to wrong CAPs
- Severity: critical • Category: traceability
- Location: prd.md:327–328 vs BC-INDEX:57–58 and capabilities.md:173–184
- Finding: PRD §7 RTM lines 327–328 both trace to CAP-020. BC-INDEX (canonical) maps BC-2.05.006 → CAP-020b and BC-2.05.007 → CAP-020a.
- Route: product-owner
- Fix: Update PRD §7: BC-2.05.006 → CAP-020b; BC-2.05.007 → CAP-020a. Update §2.05 heading to `(CAP-017–CAP-020, CAP-020a, CAP-020b)`.

## High Findings

### F-004 — ARCH-INDEX changelog says VP total 52; VP-INDEX has 57
- Severity: high • Category: consistency
- Location: ARCH-INDEX:113 vs VP-INDEX:89,100
- Fix: Append changelog row dated 2026-06-23: "Phase 1c-refinement adds VP-053–VP-057. VP total now 57."
- Route: architect

### F-005 — VP-052 references `internal/quality` module which does not exist in ARCH-08 / ARCH-09
- Severity: high • Category: consistency / scope
- Location: VP-052.md:15; VP-INDEX:78; ARCH-11:106 vs ARCH-09:34–54, ARCH-08:25–95, ARCH-05:70–112
- Finding: VP-052 frontmatter says `module: internal/quality`; harness imports `github.com/arcavenae/switchboard/internal/quality`. No such package in dep graph, purity map, or BC→package mapping.
- Route: architect
- Fix: Either add `internal/quality` to ARCH-08/09/05 (pure-core, importing paths+metrics) OR re-home VP-052 to `internal/metrics`. Update VP-052 harness imports accordingly.

### F-006 — BC-2.05.007 architecture_module = `internal/hmac` in BC/ARCH-05; `internal/admission` in ARCH-11 and VPs
- Severity: high • Category: traceability
- Location: BC-2.05.007.md:12,95; ARCH-05:102 vs ARCH-11:60; VP-007:16; VP-057:15
- Route: architect
- Fix: Adjudicate. Recommended: implementation is `internal/admission` (where keys live). Update BC-2.05.007 frontmatter + Traceability row + ARCH-05:102.

### F-007 — BC-2.01.001 jitter ±1ms is tighter than NFR-009 ≤2ms / VP-041
- Severity: high • Category: consistency
- Location: BC-2.01.001.md:53 vs nfr-catalog.md:37, VP-041:33,40
- Route: product-owner
- Fix: Update BC-2.01.001 postcondition 4 to "±2ms p99 jitter (NFR-009 budget)".

### F-008 — BC-2.02.009 drop-cache uses checksum-only key; ARCH-03 says (checksum, arrival_interface_id) compound
- Severity: high • Category: consistency / wire-format
- Location: BC-2.02.009.md:42, 50–55 vs ARCH-03:54–75
- Finding: Implementing BC literally suppresses legitimate multipath copies (the F-006 of pass 1 fix is in ARCH-03 but not propagated to BC).
- Route: product-owner
- Fix: Update BC-2.02.009 description + postconditions + EC-001/EC-003 to use `(checksum, arrival_interface_id)`. Cross-reference ARCH-03.

### F-009 — BC-2.01.004 test vector for version mismatch is bit-wrong
- Severity: high • Category: ambiguity / wire-format
- Location: BC-2.01.004.md:81, 93
- Finding: Version byte is `bits[7:4]=major, bits[3:0]=minor`. Test vector `version=2` is byte `0x02` = major 0 minor 2 = MINOR mismatch, not major.
- Route: product-owner
- Fix: Replace with `version=0x20` (major=2, minor=0). Clarify EC-001 to "major nibble (bits 7–4) > current major version."

### F-010 — Stale Round-2 propagation reminder notes in ARCH-03
- Severity: high • Category: governance / [process-gap]
- Location: ARCH-03:153–154 (BC-2.02.005 EC-003), ARCH-03:212–214 (NFR-014)
- Finding: Notes say "must be updated in Round 2." BC-2.02.005 and NFR-014 ARE updated. Notes are stale fossils.
- Route: architect
- Fix: Delete the two stale notes.

### F-011 — PRD line 93 is a stale Phase-1a placeholder note
- Severity: high • Category: governance
- Location: prd.md:93
- Finding: Says "Architecture subsystem IDs (SS-NN) are placeholders pending ARCH-INDEX (Phase 1b)." ARCH-INDEX exists with canonical Subsystem Registry.
- Route: product-owner
- Fix: Replace with positive pointer: "SS-NN canonical in `.factory/specs/architecture/ARCH-INDEX.md` Subsystem Registry."

### F-012 — interface-definitions.md:273 says RPC protocol "implementation-defined"; ADR-006 has decided JSON-over-Unix-socket
- Severity: high • Category: consistency / governance
- Route: product-owner
- Fix: Update interface-definitions.md:273 to: "RPC protocol is JSON-over-Unix-socket per ADR-006 (ARCH-05). TCP fallback when `--target=host:port` specified."

## Medium Findings

### F-013 — ARCH-11 per-module sum 56 vs VP-INDEX total 57; reconciliation note self-contradictory
- Severity: medium • Category: consistency
- Location: ARCH-11.md:88–113
- Finding: Per-module rows sum to 56; reconciliation names TWO VPs (VP-040, VP-042) but claims they account for "the remaining 1 VP."
- Route: architect
- Fix: Recount per-module assignments. Rewrite reconciliation to identify exactly ONE off-table VP (likely VP-042 with module `integration`).

### F-014 — VP-042 frontmatter `module: integration` is not a Go package
- Severity: medium • Category: consistency
- Location: VP-042.md:15
- Route: architect
- Fix: Replace `module: integration` with `internal/halfchannel` (matches the BC source).

### F-015 — ARCH-07 verification catalog only covers VP-001–VP-042; VP-043–VP-057 missing
- Severity: medium • Category: coverage / governance
- Location: ARCH-07.md:37–90
- Route: architect
- Fix: Append VP-043 through VP-057 to ARCH-07's P0/P1/Test-Sufficient tables.

### F-016 — ARCH-07 has stale module assignments (VP-033, VP-040)
- Severity: medium • Category: consistency
- Location: ARCH-07.md:81 (VP-033), :88 (VP-040)
- Finding: ARCH-07 says `integration` for both; VP-INDEX and ARCH-INDEX changelog say `internal/session` and `internal/multipath` respectively.
- Route: architect
- Fix: Update ARCH-07 module columns to match VP-INDEX. Audit all rows.

### F-017 — module-criticality.md dependency graph arrows inconsistent with ARCH-08
- Severity: medium • Category: consistency
- Location: module-criticality.md:109–135 vs ARCH-08:25–95
- Finding: Convention `A → B = build A first` breaks for `config → admission` and `config → routing` (admission/routing do not import config).
- Route: product-owner (owns module-criticality)
- Fix: Replace mermaid graph with a copy of ARCH-08's authoritative graph, or replace with a sorted topological list referencing ARCH-08 as canonical.

### F-018 — DEC-007 stale wording: "must be explicitly defined by architecture"; ADR-003 already decided LWW
- Severity: medium • Category: governance
- Location: edge-cases.md:92–99 vs ARCH-04:41–60 (ADR-003)
- Route: business-analyst
- Fix: Update DEC-007 to: "Expected behavior: last-write-wins per ADR-003 (ARCH-04). Most recent authenticated registration supersedes earlier entries for same (node_pubkey, svtn_id) pair."

## Low / Observations

### F-019 — Every BC has `[filled by story-writer]` placeholder in Stories traceability row
- Severity: low (by-design at Phase 1d, backfilled in Phase 2)
- Route: orchestrator (process)
- Fix: Add pipeline rule: "After Phase 2, verify no BC contains `[filled by story-writer]`." `[process-gap]`

## Process Observations

- **[process-gap] propagation-discipline:** Architects leave "Note for PO: must update X in Round 2" reminders that are not deleted after X is fixed. Created F-010, F-011, F-012, F-018. Add rule (S-7.01 territory): the dependent edit MUST include deletion of its triggering announcement.

- **[process-gap] crypto-derivation-source-of-truth:** Multiple files independently restate HKDF derivation (ARCH-02, ARCH-04, VP-057, BC-2.05.005/006/007, capabilities.md). F-001 had three contradictory restatements in one VP file. Crypto derivations should be specified ONCE in an ADR and hyperlinked everywhere else. A lint matching HKDF param strings across files would catch this mechanically.

## Routing Summary

| Agent | Findings |
|---|---|
| architect | F-001, F-004, F-005, F-006, F-010, F-013, F-014, F-015, F-016 |
| product-owner | F-002, F-003, F-007, F-008, F-009, F-011, F-012, F-017 |
| business-analyst | F-018 |
| orchestrator → upstream tracker | F-019 process gap (Phase 2 hook); F-010/F-001 process gaps |

## Verdict

18 findings (down from 27 in pass 1). Pattern: stale propagation notes + incomplete cross-document propagation. Substantive critical defects (HMAC keying derivation, error code collision, RTM mis-mapping). Verdict: **NOT_CONVERGED**. Round-2 refinement required before pass 3.
