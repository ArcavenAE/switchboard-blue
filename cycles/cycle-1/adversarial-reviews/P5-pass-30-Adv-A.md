---
pass_id: P5-pass-30-Adv-A
lane: A
phase: 5
cycle: cycle-1
timestamp: 2026-07-03T00:00:00Z
worktree_identity_tuple:
  factory_state_step: phase-5-pass-29-concluded-has-findings
  develop_tip_sha: 6deda15def9326f28e96f133e237aff5ecb74d7b
  factory_head_sha: 84133b2
  go_module: github.com/arcavenae/switchboard
  readme_title: Switchboard
  note: preflight-fail; orchestrator verified SHAs out-of-band before dispatch; factory_head_sha is Burst-75b SHA-token-patch commit (84133b2); mismatch with STATE.md L196 which records ba943bf (Burst 75) — this mismatch is the preflight-fail trigger
verdict: PREFLIGHT_FAIL
outcome: preflight-fail-becomes-finding
findings_count: 1
critical: 0
high: 1
medium: 0
low: 0
observations: 0
findings: [F-P5P30-A-001]
reconstructed_from_orchestrator_adjudication: false
# note: direct adversary preflight output authored by dispatch-halt agent (not orchestrator-reconstructed)
---

# Phase 5 Pass 30 — Adversary A Preflight Failure

**Lens:** Spec-completeness + traceability + POL-002 sibling-sweep
**Perimeter:** `.factory/` artifacts only — no source code reads
**Anti-findings checked (pre-flight):** Pass-29 adjudicated remediations:
- F-P5P29-A-001 SHIPPED Burst 75 (P5-pass-27-Adv-B.md + P5-pass-28-Adv-B.md authored)
- F-P5P29-A-002 SHIPPED Burst 75 (STATE.md Session Resume Checkpoint refreshed)

> **Preflight-fail verdict.** Pass 30 Adv-A dispatch was halted at preflight when the
> worktree identity tuple check detected a mismatch: STATE.md L196 records factory HEAD
> `ba943bf` (Burst 75) but the actual factory-artifacts branch tip is `84133b2` (Burst 75b).
> This constitutes a POL-002 partial-fix regression of the STATE-MANAGER-SIBLING-SWEEP class
> and the fifth consecutive recurrence of that pattern.
>
> Per pipeline protocol, a preflight-fail terminates the pass and becomes a finding for the
> orchestrator to remediate before re-dispatch. Pass 30 Adv-A proper review deferred until
> Burst 76 remediation is committed and the worktree identity tuple is consistent.

---

## F-P5P30-A-001 — HIGH — POL-002 — STATE.md L196 Stale Factory HEAD After Burst 75b

**Finding class:** POL-002 — cross-artifact freshness; fifth-consecutive-pass STATE-MANAGER-SIBLING-SWEEP recurrence. First recurrence WITHIN the codification burst itself.

**Description:** Preflight worktree identity tuple verification checks that `STATE.md`'s recorded `factory_head_sha` matches the actual factory-artifacts branch tip. The check found:

- **STATE.md L196 records:** `ba943bf` (Burst 75 — P5-pass-27-Adv-B.md + P5-pass-28-Adv-B.md sidecars authored + STATE.md checkpoint refresh + sprint-state v1.56)
- **Actual factory-artifacts tip:** `84133b2` (Burst 75b — SHA-token patch substituting `ba943bf` for placeholder tokens throughout pass_29 block, remediation_commits, and sidecar paths in sprint-state.yaml)

The mismatch is exactly one commit: Burst 75b advanced the factory HEAD from `ba943bf` to `84133b2` by patching placeholder SHA tokens, but the commit that performed that patch did not update STATE.md L196 to reflect the new HEAD. The Session Resume Checkpoint still reads "Burst 75" as the terminal burst despite Burst 75b having superseded it.

**Self-reference paradox:** This finding exposes a structural limitation in the "record tip-SHA in L196" pattern. Any commit that updates L196 changes the factory HEAD in the same operation, making the SHA it records immediately stale — the SHA is unknowable at commit-authorship time. The Burst 75 → Burst 75b arc demonstrates the exact failure mode: Burst 75 authored L196 with `ba943bf` (its own SHA), then Burst 75b patched tokens throughout the artifacts without the ability to record Burst 75b's own forthcoming SHA (`84133b2`) in L196 at authorship time.

**Pattern (fifth instance):**
| Instance | Pass | Burst | What was shipped | What was missed | Caught at |
|----------|------|-------|-----------------|-----------------|-----------|
| 1 | P27 | Burst 71a | Observable content (sprint-state v1.53 + STATE.md narrative) | Metadata layer: phase5 stanza, pass_counter, phase_step | P28 Adv-A |
| 2 | P28 | Burst 73a | Metadata layer (sprint-state v1.54 + STATE.md Status column) | Audit-trail layer: missing Adv-B sidecars for P27+P28 | P29 Adv-A |
| 3 | P29 | Burst 73c arc | Audit-trail layer (Adv-B sidecars) + checkpoint refresh | Cross-artifact freshness: checkpoint still read Burst-73a only | P29 Adv-A |
| 4 | P29-followup | Burst 75b | SHA-token patch (substituted ba943bf throughout) | L196 not updated to 84133b2 after Burst 75b advanced HEAD | P30 Adv-A (this finding) |
| 5 | P30-preflight | — | — | L196 staleness caught at preflight (self-reference paradox surfaced) | This sidecar |

**Severity escalation note:** The fourth instance (Burst 75b) is qualitatively different from instances 1–3: it is a recurrence WITHIN the very burst that was codifying the STATE-MANAGER-SIBLING-SWEEP drift item. The codification burst (Burst 75) added the drift item to STATE.md's Open Drift Items table; Burst 75b (the SHA-token patch that immediately followed) then re-exhibited the pattern the drift item describes. This is the first instance where the drift item and its recurrence are causally adjacent.

**Blast radius:** 1 STATE.md field (L196) → HIGH (worktree identity tuple used as pass preflight; mismatch blocks adversary dispatch protocol).

**Remediation:** Burst 76 — switch L196 from tip-SHA pattern to burst-arc-name pattern (no SHA recorded, readers directed to `git log --oneline -3`). Permanently resolves the self-reference paradox. STATE-MANAGER-SIBLING-SWEEP drift item escalated to fifth-instance severity with `proposed_upstream_fix` note (pre-commit sibling-sweep step in state-manager task template). Upstream vsdd-factory issue status advanced from "draft candidate" to "READY FOR FILING".

---

VERDICT: PREFLIGHT_FAIL
OUTCOME: preflight-fail-becomes-finding — F-P5P30-A-001 routed to orchestrator for Burst 76 remediation before Pass 30 re-dispatch
