---
artifact_id: wave-3-pre-impl-consistency-report
document_type: validation-report
scope: Wave 3 pre-implementation consistency audit
date: 2026-06-25
verdict: PASS_WITH_OBSERVATIONS
validator: consistency-validator
finding_counts:
  CRITICAL: 0
  HIGH: 3
  MEDIUM: 4
  LOW: 2
  OBSERVATION: 4
---

# Wave 3 Pre-Implementation Consistency Report

**Date:** 2026-06-25  
**Scope:** S-3.01, S-3.02, S-3.03, S-3.04 — Wave 3 stories + supporting specs  
**Verdict:** PASS_WITH_OBSERVATIONS (3 HIGH, 4 MEDIUM, 2 LOW, 4 OBSERVATION)  
**Blocker status:** No CRITICAL findings. Wave 3 worktrees may open after HIGH findings are resolved.

---

## Findings by Severity

### HIGH Findings

---

**F-W3-H-001**  
**Severity:** HIGH  
**Check:** #1 (Story ↔ BC trace integrity)  
**Location:** `.factory/stories/S-3.02-session-attach-detach-fanout.md` §AC-003  

**Description:**  
S-3.02 AC-003 states: "`Session.Attach` returns `E-SES-001` ('session not found: <session_name>') when the named session does not exist." The trace annotation reads `(traces to BC-2.04.003 EC-001)`.

BC-2.04.003 EC-001 is: "Console has Tier 1 (SVTN admission) but not Tier 2 for this session → E-ADM-006 'session authorization denied'."

BC-2.04.003 EC-002 is: "Named session does not exist → E-SES-001 'session not found'."

The AC body describes the EC-002 behavior but cites the EC-001 identifier. This is a classic wrong-clause anchor. An implementer following the trace would test E-ADM-006 behavior against the AC that verifies E-SES-001. A test-writer anchoring to the citation would write the wrong test case.

**Required action:** Change `S-3.02 AC-003` trace from `BC-2.04.003 EC-001` to `BC-2.04.003 EC-002`.

---

**F-W3-H-002**  
**Severity:** HIGH  
**Check:** #1 (Story ↔ BC trace integrity)  
**Location:** `.factory/stories/S-3.02-session-attach-detach-fanout.md` §AC-007  

**Description:**  
S-3.02 AC-007 states: "Keystrokes from multiple full-access consoles attached simultaneously are serialized by the access node before forwarding to tmux — no keystroke interleaving or data corruption under concurrent sends." The trace annotation reads `(traces to BC-2.04.006 PC-3)`.

BC-2.04.006 PC-3 is: "Keystrokes from any full-access console are forwarded to tmux; keystrokes from read-only consoles are rejected (per BC-2.04.005)."

The serialization requirement claimed in the AC is located in **BC-2.04.006 Invariant 3** ("All full-access console keystrokes are serialized by the access node before forwarding to tmux (no keystroke race condition)"), not in PC-3. PC-3 addresses forwarding vs. rejection by role, not concurrent-send serialization order.

This matters because the test named `TestSession_ConcurrentKeystrokes_Serialized` needs to be anchored to the invariant, not the postcondition, to accurately represent the behavioral guarantee being verified.

**Required action:** Change `S-3.02 AC-007` trace from `BC-2.04.006 PC-3` to `BC-2.04.006 Invariant 3`.

---

**F-W3-H-003**  
**Severity:** HIGH  
**Check:** #5 (ADR coverage) and #1 (BC text vs. story text conflict)  
**Location:** `ARCH-01-core-services.md §ADR-010` vs. `BC-2.04.001 EC-002` and `BC-2.04.002 EC-003`  

**Description:**  
ADR-010 (ARCH-01 §ADR-010, added in Wave 3 planning) states: "Fallback is triggered only on initial `TmuxControlMode.Attach` failure. It is NOT triggered if the control mode connection drops mid-session."

However, two BCs that were not updated to reflect this ADR-010 decision directly contradict it:

- **BC-2.04.001 EC-002** (mid-operation drop): "Access node attempts to reconnect to control mode. If reconnect fails within timeout, **falls back to PTY proxy mode**. Sends 'session unavailable' presence update."
- **BC-2.04.002 EC-003** (mid-operation drop): "Access node attempts control mode reconnect; if reconnect fails after 3 attempts, **switches to PTY proxy mode** for existing sessions."

S-3.01 EC-002 correctly follows ADR-010 ("NOT an automatic PTY fallback") but the BCs it references remain in conflict. An implementer reading BC-2.04.001 EC-002 would implement PTY fallback on mid-session drop; an implementer reading S-3.01 EC-002 or ADR-010 would not.

This creates an unresolved specification conflict that must be settled before S-3.01 implementation begins.

**Required action:** Update BC-2.04.001 EC-002 and BC-2.04.002 EC-003 to align with ADR-010. The preferred resolution (per the ADR-010 rationale) is that mid-session control mode drop marks the session unavailable and attempts reconnect; PTY fallback is NOT triggered. If the product owner intends a different behavior, ADR-010 must be revised first.

---

### MEDIUM Findings

---

**F-W3-M-001**  
**Severity:** MEDIUM  
**Check:** #4 (VP-058 coverage)  
**Location:** `.factory/specs/behavioral-contracts/ss-05/BC-2.05.003.md §Verification Properties`  

**Description:**  
BC-2.05.003 lists VP-012 three times as three distinct rows, each attributing a different property to the same VP number:

1. "Router code has no per-session authorization data structure" (code-audit)
2. "Tier 2 check is performed before upstream channel is opened" (integration)
3. "Tier 2 authorization is per-session: different sessions require separate authorization" (integration)

VP-INDEX.md defines VP-012 with exactly one property: "SessionAuth rejects unauthorized console key" (proptest). The BC-2.05.003 VP table is inflating VP-012 to cover behavior that either needs its own VP IDs or is already covered by VP-013 and VP-035.

Additionally, VP-013 (traces to BC-2.04.005, BC-2.05.003 per VP-INDEX) and VP-035 (traces to BC-2.04.005) are absent from the BC-2.05.003 VP table despite the VP-INDEX listing BC-2.05.003 as a source contract for VP-013.

S-3.03 `vp_traces: [VP-012, VP-013, VP-035]` is internally consistent with VP-INDEX but inconsistent with BC-2.05.003's VP table.

**Required action:** Rewrite the BC-2.05.003 Verification Properties table to list VP-012, VP-013, and VP-035 with their correct single-property definitions per VP-INDEX. Remove the two duplicate VP-012 rows.

---

**F-W3-M-002**  
**Severity:** MEDIUM  
**Check:** #2 (ARCH-08 §6.5/§6.6 package compliance)  
**Location:** `ARCH-04-admission-security.md §ADR-009 "Implementation note"`  

**Description:**  
ARCH-04 §ADR-009 includes this implementation note about the `verifyFrameHMAC` function:

```
func verifyFrameHMAC(header *frame.OuterHeader, rest []byte, key []byte) error
```

The actual function in `internal/routing/routing.go` (line 154) has a different signature:

```go
func verifyFrameHMAC(hdr frame.OuterHeader, payload []byte, authKey [hmac.KeySize]byte) bool
```

Three discrepancies:
1. Receiver type: `*frame.OuterHeader` (pointer) vs. `frame.OuterHeader` (value)
2. Parameter name: `rest` vs. `payload`
3. Return type: `error` vs. `bool`

BC-2.05.008 and S-3.04 both correctly describe `verifyFrameHMAC` as returning `bool` (e.g., "returns `false`", "returns `true`"), consistent with the actual code. ARCH-04 is the outlier.

While this is a spec-vs-code drift in documentation rather than an implementer blocker (S-3.04 wires an existing function), an implementer reading ARCH-04 first might attempt to change the function signature to match the ADR — which would break existing tests.

**Required action:** Update ARCH-04 §ADR-009 implementation note to reflect the actual signature: value receiver, `payload []byte` parameter, `bool` return type.

---

**F-W3-M-003**  
**Severity:** MEDIUM  
**Check:** #7 (Index synchronization)  
**Location:** `.factory/STATE.md` lines `l3_bc_count: 42` and `l4_vp_count: 57`  

**Description:**  
STATE.md carries stale counts from Wave 2 close:

- `l3_bc_count: 42` — BC-2.05.008 was minted during Wave 3 planning, bringing the count to 43. BC-INDEX confirms "42 original + BC-2.05.008 minted Wave 3."
- `l4_vp_count: 57` — VP-058 was created during Wave 3 planning. VP-INDEX §Counts now shows 58 VPs total and has been updated accordingly.
- `phase_2_bc_coverage: "42/42"` — stale; should be 43/43 post-Wave 3 planning additions.

These counters feed the wave gate machinery and convergence tracking. Stale values will produce incorrect summaries in the next gate report.

**Required action:** Update STATE.md: `l3_bc_count: 43`, `l4_vp_count: 58`. Consider whether `phase_2_bc_coverage` should be updated to `43/43` or left as historical.

---

**F-W3-M-004**  
**Severity:** MEDIUM  
**Check:** #8 (ARCH-08 §6.5 vs. actual filesystem)  
**Location:** `ARCH-08-dependency-graph.md §§1–5 (Mermaid diagram)`  

**Description:**  
The Mermaid diagram at the top of ARCH-08 (§§1–5) shows `internal/session` in "Layer 1: Security" alongside `internal/admission` and `internal/routing`. In the same diagram, `internal/tmux` appears in "Layer 4: Integration."

However, per ARCH-08's own §6.5 (authoritative for current state), `internal/session` does NOT exist on develop; it is a Wave 3 planned package at position 6 (§6.6). The Mermaid diagram presents the full target architecture, which is explicitly declared as planned/aspirational per the §1 scope callout added in v1.4.

The §1 scope callout handles this correctly in prose. The issue is that the Mermaid diagram positions `session` in "Layer 1" but §6.6 assigns it position 6 (after `halfchannel` at position 3, `admission` at position 4, and `routing` at position 5). The diagram layer label "Layer 1: Security" groups session with admission and routing — but session is actually topologically above admission (it imports admission), not a peer.

This is not a new finding (it reflects an inherent mismatch between the target-architecture diagram and the layered position table), but the ARCH-08 v1.4 scope callout does not explicitly note the layer-label discrepancy for `session`. An implementer might assume session is a peer of admission rather than depending on it.

**Required action:** Add a note to the Mermaid diagram comment (or to the §1 scope callout) clarifying that `internal/session` is depicted in "Layer 1" for structural simplicity but its actual topological position (§6.6 position 6) is above `internal/admission` (position 4). No functional change required.

---

### LOW Findings

---

**F-W3-L-001**  
**Severity:** LOW  
**Check:** #3 (Error code consistency)  
**Location:** `.factory/specs/prd-supplements/error-taxonomy.md §ADM error catalog`  

**Description:**  
The ADM error catalog lists errors in a non-sequential order: E-ADM-001, E-ADM-002, E-ADM-016, E-ADM-003, E-ADM-004 ... The E-ADM-016 row appears as the third entry (between E-ADM-002 and E-ADM-003), interrupting the numeric sequence. This was likely inserted at the "relationally logical" position near E-ADM-002 (both are HMAC-related) but creates navigation confusion.

E-ADM-016 anchor and content are correct: `BC-2.05.008`, correct message format, correct distinction from E-ADM-002.

**Required action:** Reorder the ADM catalog to place E-ADM-016 in numeric order (after E-ADM-015). No content change required.

---

**F-W3-L-002**  
**Severity:** LOW  
**Check:** #9 (Frontmatter compliance)  
**Location:** `.factory/stories/S-3.04-hmac-routeframe-wireup.md` frontmatter  

**Description:**  
S-3.04 `blocks: []` is an empty list. Per the wave dependency graph, S-3.04 has no downstream blockers in Wave 3 (it stands alone as the HMAC wire-up). However, S-3.04 WAVE-3-DEP-001 is classified as a critical-path item in STATE.md. The empty `blocks` is correct — no story is blocked on S-3.04 per STORY-INDEX — but the absence of any downstream blocker for a critical-path security story should be explicitly documented as intentional (rather than a missing link). The field is populated per spec; this is a documentation clarity note.

**Required action:** Optional — add a comment or note in S-3.04 body that `blocks: []` is intentional: no story depends on S-3.04 completing; it is a security hardening fix to the existing `internal/routing` package.

---

### OBSERVATION

---

**F-W3-O-001**  
**Severity:** OBSERVATION  
**Check:** #10 (Tier-2 auth package location consistency)  
**Location:** All 4 Wave 3 stories  

**Description:**  
Tier-2 auth (`SessionAuth`) is correctly and consistently placed in `internal/session` across all Wave 3 stories (S-3.03 creates it; S-3.01 and S-3.02 reference `internal/session` as the boundary layer). No story puts Tier-2 auth in `internal/admission` or `internal/routing`. Package placement is consistent.

---

**F-W3-O-002**  
**Severity:** OBSERVATION  
**Check:** #6 (depends_on graph)  
**Location:** All 4 Wave 3 stories  

**Description:**  
All dependency references resolve correctly in STORY-INDEX:
- S-3.01 depends_on [S-1.02 ✓, S-2.02 ✓, S-2.01 ✓] — all completed
- S-3.02 depends_on [S-3.01 ✓] — pending (expected)
- S-3.03 depends_on [S-3.02 ✓, S-2.02 ✓] — S-3.02 pending (expected)
- S-3.04 depends_on [S-2.01 ✓, S-2.02 ✓] — both completed

No phantom story IDs. No missing depends_on links.

---

**F-W3-O-003**  
**Severity:** OBSERVATION  
**Check:** #9 (Frontmatter compliance)  
**Location:** All 4 Wave 3 stories  

**Description:**  
All four stories pass frontmatter compliance:
- `wave: 3` ✓ (all)
- `status: pending` ✓ (all)
- `tdd_mode: strict` ✓ (all)
- `phase: 2` ✓ (all)
- `cycle: v1.0.0-greenfield` ✓ (all)
- `estimated_points`: 8 (S-3.01), 8 (S-3.02), 5 (S-3.03), 3 (S-3.04) ✓ — matches STORY-INDEX wave summary (24 pts total)
- `bc_traces` present and reference real IDs ✓ (all)
- `vp_traces` present and reference real IDs ✓ (all VPs exist in VP-INDEX as active/draft)

---

**F-W3-O-004**  
**Severity:** OBSERVATION  
**Check:** #2 and #8 (ARCH-08 §6.5 vs filesystem)  
**Location:** `ARCH-08-dependency-graph.md §6.5` vs `ls internal/`  

**Description:**  
ARCH-08 §6.5 lists exactly 5 packages: `frame`, `hmac`, `halfchannel`, `admission`, `routing`. The actual `ls internal/` output shows exactly the same 5 packages: `admission`, `frame`, `halfchannel`, `hmac`, `routing`. No divergence. §6.5 is accurate.

All 4 Wave 3 stories reference packages from §6.6 (PLANNED): `internal/session` (position 6) and `internal/tmux` (position 7). No story references an undeclared package. Architecture compliance is clean.

---

## Check Summary Table

| Check # | Description | Status | Finding(s) |
|---------|-------------|--------|-----------|
| 1 | Story ↔ BC trace integrity | FAIL | F-W3-H-001 (S-3.02 AC-003 wrong EC), F-W3-H-002 (S-3.02 AC-007 wrong clause) |
| 2 | ARCH-08 §6.5/§6.6 package compliance | PASS with obs | F-W3-M-002 (ARCH-04 sig drift), F-W3-O-004 (§6.5 matches fs) |
| 3 | Error code consistency | PASS | F-W3-L-001 (taxonomy ordering cosmetic); all required error codes present in taxonomy |
| 4 | VP-058 coverage | PASS | VP-058 active, in VP-INDEX, cited in S-3.04 and BC-2.05.008; F-W3-M-001 (BC-2.05.003 VP table inaccuracy is adjacent, not VP-058-specific) |
| 5 | ADR coverage | FAIL | F-W3-H-003 (ADR-010 vs BC-2.04.001/002 EC conflict); ARCH-01 ADR-010 exists and S-3.01 cites it |
| 6 | depends_on graph | PASS | F-W3-O-002 (all IDs resolve) |
| 7 | Index synchronization | PARTIAL | F-W3-M-003 (STATE.md stale counts); BC-INDEX and STORY-INDEX rows are correct |
| 8 | ARCH-08 §6.5 vs. actual filesystem | PASS | F-W3-O-004 (exact match) |
| 9 | Frontmatter compliance | PASS | F-W3-O-003 (all pass); F-W3-L-002 (minor observation on S-3.04 blocks: []) |
| 10 | Tier-2 auth package location | PASS | F-W3-O-001 (consistent placement in internal/session) |

---

## BC Trace Table

| Story | AC | Cited BC + Clause | Clause Exists? | Text Matches? | Finding |
|-------|----|--------------------|----------------|---------------|---------|
| S-3.01 | AC-001 | BC-2.04.001 PC-1 | ✓ | ✓ ("has active control mode connection") | — |
| S-3.01 | AC-002 | BC-2.04.001 PC-2 | ✓ | ✓ ("all current sessions enumerated and published") | — |
| S-3.01 | AC-003 | BC-2.04.001 PC-3 + PC-4 | ✓ | ✓ ("new sessions auto-discovered; closed auto-unpublished") | — |
| S-3.01 | AC-004 | BC-2.04.001 PC-5 | ✓ | ✓ ("output events feed downstream half-channel") | — |
| S-3.01 | AC-005 | BC-2.04.002 PC-1 | ✓ | ✓ ("enters PTY proxy mode") | — |
| S-3.01 | AC-006 | BC-2.04.002 PC-2 + PC-3 | ✓ | ✓ ("synthetic name" + "log entry written") | — |
| S-3.01 | AC-007 | BC-2.04.002 EC-004 | ✓ | ✓ ("E-SYS-001; non-zero exit") | — |
| S-3.02 | AC-001 | BC-2.04.003 PC-1 | ✓ | ✓ ("bidirectional channel established") | — |
| S-3.02 | AC-002 | BC-2.04.003 PC-3 | ✓ | ✓ ("upstream keystrokes accepted and forwarded to tmux") | — |
| S-3.02 | AC-003 | BC-2.04.003 EC-001 | WRONG — should be EC-002 | MISMATCH (AC describes E-SES-001 session not found; EC-001 is Tier-2 auth failure → E-ADM-006) | F-W3-H-001 |
| S-3.02 | AC-004 | BC-2.04.004 PC-1 + PC-2 | ✓ | ✓ ("channel closed cleanly; tmux session continues") | — |
| S-3.02 | AC-005 | BC-2.04.004 PC-5 | ✓ | ✓ ("read-only observers continue") | — |
| S-3.02 | AC-006 | BC-2.04.006 PC-1 | ✓ | ✓ ("two or more consoles receive all downstream frames") | — |
| S-3.02 | AC-007 | BC-2.04.006 PC-3 | ✓ exists | MISMATCH: PC-3 covers role-based forward/reject; serialization is Invariant 3 | F-W3-H-002 |
| S-3.02 | AC-008 | BC-2.04.004 EC-002 / BC-2.04.006 | ✓ | ✓ ("channel close detected; evicts from fan-out") | — |
| S-3.03 | AC-001 | BC-2.05.003 PC-1 | ✓ | ✓ ("key in list → attach proceeds") | — |
| S-3.03 | AC-002 | BC-2.05.003 PC-2 | ✓ | ✓ ("key not in list → E-ADM-006") | — |
| S-3.03 | AC-003 | BC-2.05.003 PC-3 + Invariant 1 | ✓ | ✓ ("router has no per-session auth data structure") | — |
| S-3.03 | AC-004 | BC-2.05.003 PC-4 | ✓ | ✓ ("per-session, no cross-session spillover") | — |
| S-3.03 | AC-005 | BC-2.04.005 PC-1 + PC-3 | ✓ | ✓ ("attach + downstream; upstream rejected E-ADM-007") | — |
| S-3.03 | AC-006 | BC-2.04.005 PC-3 + EC-004 | ✓ | ✓ ("empty-tick accepted; payload-bearing rejected") | — |
| S-3.04 | AC-001 | BC-2.05.008 PC-1 | ✓ | ✓ ("valid HMAC proceeds to admission and SVTNRoute") | — |
| S-3.04 | AC-002 | BC-2.05.008 PC-2 | ✓ | ✓ ("invalid HMAC → ErrHMACVerificationFailed immediately; E-ADM-016 logged") | — |
| S-3.04 | AC-003 | BC-2.05.008 PC-3 / VP-058 property 1+2 | ✓ | ✓ ("HMAC verified before admitted-set check") | — |
| S-3.04 | AC-004 | BC-2.05.008 PC-4 / VP-058 property 4 | ✓ | ✓ ("no forwarding entry → ErrHMACVerificationFailed") | — |
| S-3.04 | AC-005 | BC-2.05.008 EC-005 / Invariant 2 | ✓ | ✓ ("ErrHMACVerificationFailed distinct from ErrNotAdmitted") | — |

---

## Detailed Notes by Check

### Check 3 — Error Code Consistency

- **E-ADM-016** row exists in error-taxonomy.md with correct anchor (BC-2.05.008, distinct from E-ADM-002). ✓
- **S-3.04** cites E-ADM-016 for HMAC wire-layer failure (not E-ADM-002). ✓
- **S-3.03** cites E-ADM-006 (Tier-2 auth) and E-ADM-007 (read-only rejection). Both exist in taxonomy. ✓
- **S-3.01** cites E-SYS-001 only. E-ADM-006 and E-SES-001 are not applicable to tmux control mode connection — the original check expectation was overly broad. E-SYS-001 exists in taxonomy. ✓
- **S-3.02** cites E-ADM-006 (in EC edge case table, deferred to S-3.03), E-SES-001 (AC-003), E-NET-005 (EC-003 edge case). All three exist in taxonomy. ✓
- No story cites E-ADM-002 for the wire-layer HMAC case. ✓

### Check 5 — ADR Coverage

- **ADR-009** in ARCH-04 cites E-ADM-016 (not E-ADM-002). ✓
- **ADR-010** in ARCH-01 covers tmux primary + PTY fallback. ✓
- **S-3.01** body cites ADR-010 in multiple places (edge cases, architecture compliance rules, tasks list). ✓
- BC-2.04.001 EC-002 and BC-2.04.002 EC-003 have NOT been updated to reflect ADR-010's "no mid-session fallback" decision. HIGH finding F-W3-H-003.

### Check 4 — VP-058 Coverage

- VP-058.md exists, `lifecycle_status: active`. ✓
- VP-INDEX has a row for VP-058 listed as `draft` status (expected — not yet verified against implementation). ✓
- S-3.04 frontmatter `vp_traces: [VP-058]`. ✓
- BC-2.05.008 body cites VP-058 in Verification Properties and in "VP Anchors" section. ✓

---

## Required Actions Summary

| Priority | Finding | Action |
|----------|---------|--------|
| HIGH — fix before S-3.02 worktree opens | F-W3-H-001 | Change S-3.02 AC-003 trace from `BC-2.04.003 EC-001` to `BC-2.04.003 EC-002` |
| HIGH — fix before S-3.02 worktree opens | F-W3-H-002 | Change S-3.02 AC-007 trace from `BC-2.04.006 PC-3` to `BC-2.04.006 Invariant 3` |
| HIGH — fix before S-3.01 worktree opens | F-W3-H-003 | Update BC-2.04.001 EC-002 and BC-2.04.002 EC-003 to align with ADR-010 (no PTY fallback on mid-session drop) |
| MEDIUM — fix before story picked up | F-W3-M-001 | Rewrite BC-2.05.003 VP table: remove duplicate VP-012 rows; add VP-013 and VP-035 |
| MEDIUM — fix before S-3.04 worktree opens | F-W3-M-002 | Update ARCH-04 §ADR-009 implementation note signature to match actual routing.go (value receiver, `payload []byte`, `bool` return) |
| MEDIUM — fix before wave gate | F-W3-M-003 | Update STATE.md: `l3_bc_count: 43`, `l4_vp_count: 58` |
| MEDIUM — fix before wave gate | F-W3-M-004 | Add note to ARCH-08 §1 or Mermaid comment that `internal/session` Layer 1 placement is aspirational; actual topological position is 6 (above admission at 4) |
| LOW | F-W3-L-001 | Reorder error-taxonomy.md ADM catalog to place E-ADM-016 in numeric sequence |
| LOW | F-W3-L-002 | Optional: document `blocks: []` in S-3.04 as intentional |
