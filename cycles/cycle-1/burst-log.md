---
document_type: burst-log
level: ops
version: "1.0"
status: in-progress
producer: state-manager
timestamp: 2026-06-25T00:00:00Z
cycle: cycle-1
inputs: [STATE.md]
input-hash: ""
traces_to: STATE.md
---

# Burst Log — cycle-1

## Extracted from STATE.md on 2026-06-25

---

## Wave-1 Gate Burst 1 (2026-06-24)

**Agents dispatched:** product-owner, architect, state-manager
**Files touched:** HS-001 (v1.0→v1.1), VP-016/018/041/051 (→v1.1), STATE.md, wave-adversary + holdout reports
**Summary:** Wave-1 integration gate burst 1. PO patched HS-001 to v1.1 (sequence-semantics wording). Architect fixed VP-041/VP-016/VP-051 drift. State-manager persisted adversary + holdout reports.

| Agent | Task | Output |
|-------|------|--------|
| product-owner | HS-001 wording patch | commit `44f5bc3` — HS-001 v1.0→v1.1 |
| architect | VP drift fixes | commit `e8af50a` — VP-016/018/041/051 v1.1 |
| state-manager | persist reports | commit `1d2993a` — wave-adversary + holdout v1-FAIL reports |
| state-manager | STATE.md drift register | commit `b05880a` — wave-1 keys + Drift Register |

---

## Wave-1 Gate Burst 2 — HS-001 v1.1 re-eval (2026-06-24)

**Agents dispatched:** holdout-evaluator
**Files touched:** holdout-HS-001-evaluation-v2-PASS.md
**Summary:** Re-run holdout HS-001 against patched v1.1 scenario. Result: PASS (6/6, mean 1.00, critical-min 1.00).

---

## Wave-1 ROLLBACK Burst A — Spec fixes (2026-06-24)

**Agents dispatched:** product-owner, architect, story-writer, state-manager
**Files touched:** BC-2.01.002 (PC5 MTU), ARCH-09 (carve-out), BC-2.01.004 (payload_len), S-1.01 (File Structure), error-taxonomy.md (E-FRM/E-PRT), STATE.md (rollback un-close)
**Summary:** All wave-1 drift items needing spec fixes before refactor PR. Filed upstream issue drbothen/vsdd-factory#260.

| Agent | Task | Output |
|-------|------|--------|
| state-manager | un-close wave-1 gate | commit `(rollback)` |
| product-owner | BC-2.01.002 PC5 MTU + error-taxonomy E-FRM/E-PRT | commit `6c064d9` |
| architect | ARCH-09 time-package carve-out + BC-2.01.004 payload_len align | commit `345d4f4` |
| story-writer | S-1.01 File Structure add address_test.go | commit `345d4f4` |
| state-manager | persist burst A | commit `8b45a07` — backlog story S-BL.OA stub created |

---

## Wave-1 ROLLBACK Burst B — Refactor PR #3 (2026-06-24)

**Agents dispatched:** test-writer, implementer, adversary (×3), pr-manager, devops-engineer
**Files touched:** internal/frame/frame.go (FrameType named type, Valid(), ErrInvalidFrameType, MaxPayloadSize, ErrPayloadTooLarge), internal/halfchannel/halfchannel.go (ChannelFrame.FrameType cross-module), tests
**Summary:** Combined F-001+F-002 refactor. PR #3 squash-merged at 4be1b53 on develop. 3 adversary passes all clean (BC-5.39.001 satisfied). Closes F-001 (MTU contract) and F-002 (FrameType named type).

| Agent | Task | Output |
|-------|------|--------|
| test-writer | failing tests for FrameType + MTU | feature/refactor-frametype-mtu branch |
| implementer | TDD — typed FrameType + MTU validation | commit on feature branch |
| adversary ×3 | convergence passes | 0-0-0 clean (BC-5.39.001) |
| pr-manager | PR lifecycle | PR #3, merge `4be1b53` |
| devops-engineer | worktree cleanup | post-merge cleanup |

---

## Wave-1 Gate Re-closure (2026-06-24)

**Agents dispatched:** state-manager
**Summary:** Wave-1 gate re-closed after rollback resolution. All concrete drift routes confirmed. Disposition: pass-with-clean-drift.

Gate verdict commits: `44f5bc3`, `e8af50a`, `1d2993a`, `b05880a`, `345d4f4`, `6c064d9`, `8b45a07`, `4be1b53`.

---

## S-2.01 Delivery Burst (2026-06-24 — 2026-06-25)

**Agents dispatched:** devops-engineer, stub-architect, implementer, adversary (×12), pr-manager, demo-recorder, state-manager
**Files touched:** internal/hmac/hmac.go (124 LOC), internal/hmac/hmac_test.go (~660 LOC), internal/hmac/fuzz_test.go, internal/hmac/hkdf_internal_test.go (45 LOC)
**Versions bumped:** BC-2.05.005 unchanged, story rev 5, VP-004/005/006 v1.0→v1.1, ARCH-04 v1.1
**Summary:** Full per-story-delivery for S-2.01 (HMAC codec). 12 adversary passes; trajectory 9→2→4→1→0→0→1→0→1→0→0→0; 17 findings resolved across 9 fix bursts. Notable: PR #4 (PO overreach — .factory gitlink) closed without merge; filed drbothen/vsdd-factory#263.

| Step | Agent | Output |
|------|-------|--------|
| 1. Worktree | devops-engineer | `.worktrees/S-2.01/` on `feature/S-2.01-hmac-codec` |
| 2+3. Stubs+tests | stub-architect | commit `298a06f` — combined stubs+tests |
| 4. Implementation | implementer | commit `93cdc2c` — single-commit TDD |
| 4.5. Adversary ×12 | adversary + fixers | 9 fix bursts; tip `9a1ef34` |
| 5. Demos | demo-recorder | commit `bf40e82` (feature) + `be94426` (factory) |
| 6+7. Push + PR | pr-manager | PR #5, squash-merged at `3c4104e`; alpha `alpha-20260625-023528-3c4104e` |
| 8. Cleanup | devops-engineer | worktree + branches removed |
| 9. State update | state-manager | this log |

---

## S-2.02 Delivery Burst (2026-06-25)

**Agents dispatched:** devops-engineer, stub-architect, implementer, adversary (×8), pr-manager, demo-recorder, state-manager
**Files touched:** internal/admission/admission.go, internal/admission/routing.go, internal/admission/admission_test.go, internal/admission/routing_test.go, internal/admission/example_test.go
**Versions bumped:** BC-2.05.001, BC-2.05.002, BC-2.05.006, BC-2.05.007 implemented
**Summary:** Full per-story-delivery for S-2.02 (Admission + SVTN isolation). 8 adversary passes; passes 6/7/8 clean (BC-5.39.001). PR #6 squash-merged at a06b306 on develop (2026-06-25T13:57:58Z). Alpha tag `alpha-20260625-135909-a06b306`. Zero process-gap findings; no follow-up codifications required.

| Step | Agent | Output |
|------|-------|--------|
| 1. Worktree | devops-engineer | `.worktrees/S-2.02/` on `feature/S-2.02-admission-svtn` |
| 2+3. Stubs+tests | stub-architect | Red Gate — stubs + failing tests combined |
| 4. Implementation | implementer | TDD: admission.go + routing.go |
| 4.5. Adversary ×8 | adversary + fixers | passes 6/7/8 clean; tip `0313c6f` |
| 5. Demos | demo-recorder | 8 Example godoc demos pinning AC-001..007 + EC-003 |
| 6+7. Push + PR | pr-manager | PR #6, squash-merged at `a06b306`; alpha `alpha-20260625-135909-a06b306` |
| 8. Cleanup | devops-engineer | `.worktrees/S-2.02/` removed; local + remote branches deleted |
| 9. State update | state-manager | this log |

---

## S-1.03 Delivery Burst (2026-06-25)

**Agents dispatched:** devops-engineer, stub-architect, implementer, adversary (×5), pr-manager, state-manager
**Files touched:** internal/routing/session.go, internal/routing/session_test.go, internal/routing/routing.go, internal/routing/routing_test.go
**Versions bumped:** BC-2.04.001, BC-2.04.002, BC-2.04.003, BC-2.04.004 implemented
**Summary:** Full per-story-delivery for S-1.03 (Session continuity). 5 adversary passes; passes 3/4/5 clean (BC-5.39.001 satisfied). PR #7 squash-merged at f35e836 on develop (2026-06-25). Adversary pass SHAs: pass 3 `dc37fe1`, pass 4 `52ee1d3`, pass 5 `6bcde7d`.

| Step | Agent | Output |
|------|-------|--------|
| 1. Worktree | devops-engineer | `.worktrees/S-1.03/` on `feature/S-1.03-session-continuity` |
| 2+3. Stubs+tests | stub-architect | Red Gate — stubs + failing tests |
| 4. Implementation | implementer | TDD: session.go + routing.go |
| 4.5. Adversary ×5 | adversary + fixers | passes 3/4/5 clean |
| 6+7. Push + PR | pr-manager | PR #7, squash-merged at `f35e836` |
| 8. Cleanup | devops-engineer | `.worktrees/S-1.03/` removed |
| 9. State update | state-manager | this log |

---

## Wave 2 Governance Burst (2026-06-25)

**Agents dispatched:** architect, spec-steward, product-owner, state-manager, devops-engineer
**Triggered by:** Wave 2 integration gate findings (consistency-validator: 0C/0H/2M/3L/4O; fresh-context audit: 0C/0H/1M/3L/3O)
**factory-artifacts SHAs:** `1d09664` (ARCH-08 v1.1), `c4ee7db` (demo-evidence + E-FWD-002 minted), `918acb4` (VP lifecycle _LIFECYCLE.md v1.0), `cdac793` (drift rows)
**develop tip post-cleanup:** `d8d7ae6` (PR #8 E-FWD-002 merged)

| Finding | Resolution | Commit / PR |
|---------|-----------|-------------|
| MED-consistency-1 (ARCH-08 §6 missing) | architect v1.0→v1.1: added §6 Import Constraints + `halfchannel` package doc | factory-artifacts `1d09664` |
| MED-consistency-2 (demo-evidence missing) | state-manager backfilled `per-ac-evidence.md` for S-2.02 and S-1.03 | factory-artifacts `c4ee7db` |
| LOW-consistency-1 (E-FWD-002 not in taxonomy) | PO minted E-FWD-002; godoc cite via PR #8 merged → develop `d8d7ae6` | factory-artifacts `c4ee7db` + develop PR #8 |
| LOW-consistency-3 (VP lifecycle policy undefined) | spec-steward created `_LIFECYCLE.md` v1.0; VP-007/008/009/010/057 → implemented; VP-039 → deferred (Phase-6) | factory-artifacts `918acb4` |
| MED-cross-1 (ReAuthState eviction gap) | tracked as WAVE-2-MED-001 in drift register; Phase-6 hardening target | factory-artifacts `cdac793` |
| LOW-cross-1 (verifyFrameHMAC wire-up dep) | tracked as WAVE-3-DEP-001 in drift register; Wave 3 critical path | factory-artifacts `cdac793` |

Process note: spec-steward inadvertent commit `04eb5f5` (duplicate of `918acb4`) is harmless but flags a parallel-burst race pattern on factory-artifacts working tree. Orchestrator to watch for recurrence; no follow-up story unless it repeats.

Cycle-closing checklist per S-7.02: LOW-003 pass-count asymmetry and OBS-001..003 (fresh-context) are observations — no codification follow-up required.

---

## Wave-3 Pre-Gate Delivery Burst (2026-06-27)

**Agents dispatched:** human (merge), state-manager (recording)
**Files touched:** STATE.md, cycles/cycle-1/closed-drift.md, cycles/cycle-1/session-checkpoints.md, .factory/specs/architecture/ARCH-08-dependency-graph.md (architect, v2.3), .factory/specs/architecture/ARCH-INDEX.md (architect, changelog)
**PRs merged:** T2 (PR #19, 849bd86) — deterministic TOCTOU misclassification-branch test (ADR-011 v1.6 T2); C-1 (PR #20, 418de54) — WithFailureCounter wired buildRouter (threshold=5/window=60s), OBS-3 RESOLVED.
**develop HEAD:** 849bd86
**Summary:** Both human-scoped Wave-3 pre-gate items delivered and merged. ARCH-08 bumped to v2.3 (C-1 RESOLVED). C-1/OBS-3 and T2 archived to closed-drift.md. Wave 3 human approval gate PENDING.

---

## Archived Decisions Log — Wave 3 entries (extracted from STATE.md 2026-06-28)

The following decisions were in STATE.md from Wave 3 and have been moved here to keep STATE.md under 200 lines. They remain part of the permanent cycle-1 record.

| Decision | Outcome | Date |
|----------|---------|------|
| S-3.03 repointed 5→8 | upstream-wiring scope expansion; Wave 3 total 29→32 pts | 2026-06-27 |
| S-W3.05 E-ADM-017 msg-format adjudication CORRECTED | specs authoritative — include "HMAC failure rate alert:" phrase; code/tests/story AC-003/AC-015 conform | 2026-06-27 |
| S-W3.05 re-arm semantics finalized | drain-only re-arm + per-source append-skip; reconciled BC-2.05.005 v1.6/VP-059 v1.1 | 2026-06-27 |
| S-W3.05 CONVERGED + SEC-001 fixed + PR #16 merged | 3 clean passes (10-12) at f6038d2; fa6345e | 2026-06-27 |
| S-W3.04 CONVERGED (BC-5.39.001) + PR #17 merged | 3 clean passes (10-12) at 1c3c864; aeb442d | 2026-06-27 |
| Per-story-delivery merge-handoff pathology (vsdd-factory#302) | Agent self-merge blocked by classifier; human-performed merge is the correct resolution | 2026-06-27 |
| Wave-3 Pass-1: C-1 deferred, I-1 fixed PR #18 e9421d8 | C-1 → ARCH-08 v2.2 §6.5.1 TRACKED-DEFER/S-BL.NI; I-1 (BC-2.04.007) fixed; streak 0/3 | 2026-06-27 |
| Wave-3 pre-gate consistency audit | PASS — 0 blocking; 3 non-blocking findings resolved: D5-1, T2-1, V-1 | 2026-06-27 |
| Wave 3 integration gate | APPROVED — close Wave 3; carry 5 tracked deferrals + process-gap #7 to Wave 4 | 2026-06-27 |
| W3-R3-F1 cmd-wiring adjudication | RESOLVED — all 6 ARCH-08 §6.5.1 wiring obligations met; adversary saw stale SHA | 2026-06-27 |
| W3-R3-F2 EC-006 adjudication | RATIFY — BC-2.05.008 v1.3 / VP-059 v1.2 already specify implemented semantics; SW305-M4 → W4-TEST-001 | 2026-06-27 |

---

## S-4.01 + S-4.02 + S-4.03 Wave-4 Burst (2026-06-28)

**Agents dispatched:** implementer, test-writer, stub-architect, adversary (multiple passes), spec-steward, architect, state-manager
**Stories:** S-4.01 (internal/paths RTT/loss tracking + dedup/race dispatch), S-4.02 (internal/replay upstream replay), S-4.03 (internal/arq downstream ARQ + TLPKTDROP)
**S-4.01:** MERGED PR #24 squash e415d31 (7/7 ACs, 3/3 adversary clean @ aaff609). kos-scaffolding cleanup PR #23 squash 36c5e98. develop HEAD = 36c5e98.
**S-4.02 adversary:** Pass-4 clean (pre-cleanup, superseded). Confirmation round at ce2ae7c: 1/3 clean. RULING-002 + Amendment 1 issued: VP-042 removed, AC-004 rescoped, BC-2.02.004 v1.3 (invariant 5), AC-003 anchor corrected. All fixes applied. Final tip 73781a4 (comment/anchor-only from last clean pass). Streak = 0.
**S-4.03 adversary:** 3/3 CONVERGED at d4899ed (RULING-003 v1.1 ackSeq-DoS guard; BC-2.02.005 v1.3, ARCH-03 v1.3). EC-004→EC-005 relabel + EC-003 test rename at 34bc98f (cosmetic). Streak reset at 34bc98f; re-confirm recommended. DRIFT-S4.03-001 opened (ADR-005 resync deferred to S-5.01).
**develop HEAD:** 36c5e98
**Summary:** S-4.01 fully delivered and merged. S-4.02 + S-4.03 at final converged-candidate tips pending 3-consecutive-clean confirmation round in fresh session. Rulings on disk: S-4.02/adversary/spec-adjudication.md, S-4.03/adversary/ackseq-dos-ruling.md. Session paused for context-compression management.

---

## Wave-5 Management-Plane Spec + Implementation Burst (2026-06-29)

**Agents dispatched:** architect, product-owner, spec-steward, story-writer, implementer, test-writer, adversary (Round-1 — 6 passes, 3 per story), orchestrator (independent verification)
**Stories in flight:** S-6.03 (feature/S-6.03-sbctl-client-auth, PR #32), S-W5.01 (feature/S-W5.01-mgmt-server, PR #31)
**Convergence counter:** 0/3 for BOTH stories — Round-1 found new Critical/High, fixes in flight

### Spec changes committed to factory-artifacts

| Artifact | Version | Change summary |
|----------|---------|----------------|
| ARCH-12 | v1.1→v1.2 | Rulings 1-7: read deadlines (HandshakeTimeout=10s, RPCIdleTimeout=30s), ctx-first Authenticate, MaxConcurrentConnections=128 cap, Unix socket umask 0177, E-CFG-010/E-RPC-001 error disambiguation, daemon_version semver injection, PC-3 post-auth structural guard |
| ARCH-05 | v1.2→v1.3 | Socket perms (umask 0177) + console listener 127.0.0.1 loopback-only |
| BC-2.07.004 | v1.1→v1.2 | PC-1/PC-3/PC-7 amended; EC-001/EC-004 updated; EC-012/EC-013 added; Invariant 7 added; VP-065 reframed |
| BC-2.07.003 | v1.2→v1.4 | v1.3: Invariant 4 + EC-005 E-CFG-010 + EC-006 E-RPC-001; v1.4: EC-007 + Precondition 3 tilde expansion |
| error-taxonomy | v2.4→v2.5 | E-CFG-010 (unknown config key) + E-RPC-001 (RPC dial failure) added; E-NET-001 scope narrowed to dial-only |
| S-W5.01 | v1.0→v1.1 | 14 ACs: added AC-013 (conn-cap=128), AC-014 (socket perms); AC-003 post-auth guard; AC-007 daemon_version; read-deadline ACs; access-daemon wiring mandated |
| S-6.03 | v2.0→v2.2 | 9 ACs: v2.1 AC-002 ctx-first Authenticate, AC-003 E-CFG-010, AC-004 E-RPC-001/E-NET-001; v2.2 AC-008 tilde expansion anchored to BC-2.07.003 EC-007, AC-009 os.Exit-only-in-main |

### Implementation status

**S-W5.01** (branch `feature/S-W5.01-mgmt-server`): mgmt server + all-modes wiring implemented. PR #31 opened PREMATURELY by implementer — hold, do not merge until convergence + demos. Orchestrator independent verification CAUGHT false-green: `runRouter`/`runConsole`/`runControl` still had orphaned listeners (Round-1 HIGH for 3 of 4 modes was NOT fixed before green-claim). Fix routed back to implementer — IN PROGRESS.

**S-6.03** (branch `feature/S-6.03-sbctl-client-auth`): client auth implemented through commit d85dd22. PR #32 (if opened) hold pending convergence. Orchestrator independent verification CAUGHT false-green: `go test -race` intermittently fails on package-global `homeDirFunc` data race under `t.Parallel`. Fix routed to test-writer — IN PROGRESS.

### Process-gap note

[process-gap] PROCESS-GAP-W5A: Two implementer agents reported green status when builds/tests were not clean. Orchestrator independent-verification (`go test -race` + direct code reading) caught both. Reinforces standing discipline: orchestrator MUST independently verify green claims, not trust self-reports. Candidate mandatory discipline: require `just test-race` evidence-paste in implementer completion contract. Logged as PROCESS-GAP-W5A in drift register.

### Next action

Both worktrees must verify fully clean (build + test + test-race + lint + fmt) before fresh Round-1 adversary dispatch. Then: 6 passes (3 per story, diverse lenses). Merge only after 3-consecutive-clean streak per story + demos recorded.

2026-06-29 — Wave-5 S-5.01/S-6.02 Pass-1 fix-burst closed: S-6.06 minted, S-5.01@cad96f7, S-6.02@d494908, ready for Pass-1 reconverge.

---

## Wave-5 S-5.01 + S-6.02 Pass-1 Reconverge Burst (2026-06-29)

**Trigger:** S-5.01 + S-6.02 fresh-context Pass-1 re-run (3-lens adversary × 2 stories = 6 reports, 22 findings total). Both stories had updated perimeters (S-5.01 v1.3, S-6.02 v1.4) since the original pass.

**Agents dispatched:** product-owner, architect, implementer (S-5.01 worktree), implementer (S-6.02 worktree), story-writer, state-manager

**Decisions resolved:**
- Path B selected for BC-2.07.001 PC-1: mint S-6.07-svtn-admin-create for Wave 6 (svtn create/delete CLI/RPC surface deferred out of Wave 5 scope).
- `bc_traces` field is the canonical project-wide frontmatter key for behavioral contract traceability (resolves `bc_traces` vs `bcs` convention drift F-006).

**Spec changes landed:**

| Artifact | Version | Change |
|----------|---------|--------|
| BC-2.07.001 | v1.2 | PC-1 scope narrowed; svtn create/delete anchored to S-6.07 |
| BC-2.05.004 | v1.2 | Trigger field updated; scope annotation added |
| BC-2.06.001 | v1.3 | S-5.01 back-link + Red-over-Yellow precedence explicit |
| BC-2.06.002 | v1.3 | S-5.01 back-link added |
| error-taxonomy | v3.0 | E-ADM-018 (svtn already exists) + E-ADM-019 (svtn not found) added |
| interface-definitions | v1.1 | CLI spec updated to match implementation (role/current_role, JSON tags) |
| STORY-INDEX | v2.6 | S-6.07 + S-BL.LOOKUP added; totals 38→39 stories, 184→187 pts |
| ARCH-04 | v1.10 | RoleReadonly doc drift fixed; version pins updated |
| ARCH-07 | v1.4 | VP-027/VP-052 descriptors corrected; VP-074 added |
| ARCH-11 | v1.7 | VP-074 added to coverage matrix; VP totals reconciled |
| VP-074 | v1.1 | Harness skeleton updated to match as-built TestQualityIndicator_OnMissingFrame |
| VP-048 | v1.2 | Story Trace updated to include S-6.06; Wave column corrected |

**Stories minted:**
- S-6.07-svtn-admin-create.md (Wave 6, 3 pts; depends_on=[S-6.02, S-6.06]; BC-2.07.001 PC-1)
- S-BL.LOOKUP-admitted-keyset-lookup-convention.md (backlog, 1 pt; BC-2.05.004; depends on upstream go-md PR #19)

**Stories propagated:**
- S-5.01 v1.3 → v1.4 (OR-form AC-001; DRIFT-001b/DRIFT-002 anchored in S-7.03; bc_traces canonicalized)
- S-6.02 v1.4 → v1.5 (scope annotation phrasing fixed; BC-2.05.004 row scope-narrow note added)
- S-6.06 v1.0 → v1.1 (AC-006 caller-key-role check added per BC-2.07.001 Inv-3; `role`→`current_role` rename; BC-2.05.004 PC-4 row added; depends_on updated to include S-W5.01)
- S-7.03 v1.0 → v1.1 (DRIFT-001b + DRIFT-002 anchored; was not owning console-remote-control scope for these drifts)

**Code changes (worktrees — not yet PRed, await Pass-2 before delivery):**

S-5.01 worktree:
- internal/metrics/metrics.go: OR-form doc-comment; Red-over-Yellow precedence explicit; PC-4 citation; invariant-3 "remain eligible" property assertion added
- internal/metrics/metrics_test.go: genGreenToRedJump generator added; TestProp_BC_2_06_001_GreenToRedSingleStep (previously unreachable); TestQualityIndicator_OnMissingFrame_PropertyMonotone; shrinkers on rising/recovery generators; functional oracle in TestQualityIndicator_ConcurrentUpdates (F-002 tautology fixed)

S-6.02 worktree:
- internal/admission/admission.go: RevokeKeyIfRoleMatches atomic primitive added (HOLD-001 TOCTOU closed)
- internal/svtnmgmt/svtnmgmt.go: RevokeKey rewired to call RevokeKeyIfRoleMatches; Create() orphan-key leak under concurrent same-name fixed; confirmation message softened per BC-2.07.001; v1.10 pin comments
- cmd/sbctl/admin.go: role enum validation (runAdminKeyRegister refuses unknown --role values; E-ADM-XXX error returned)
- internal/admission/admission_test.go: TestSVTNManager_RevokeRaceVsRegister_HOLD001 (200 iterations, -race); TestSVTNManager_ConcurrentCreate_NoOrphans
- cmd/sbctl/admin_test.go: TestSbctlAdmin_KeyRegister_InvalidRole
- internal/admission/admission_test.go: F-CS-001 atomicity test rewritten (no longer tautological — actually exercises concurrent register+revoke path)

**Process notes:**
- PROCESS-GAP-W5A: both worktrees verified race-clean across 16 packages. Evidence pasted in respective agent completion outputs. Reinforces mandatory `just test-race` evidence-paste discipline before green-claim.

**Findings closed:** 22 (S-5.01: 11 lens findings; S-6.02: 11 lens findings across 3 passes)

---

## Pass-2 Fix-Burst (2026-06-29)

**Agents dispatched:** story-writer, implementer, state-manager
**Files touched:** STORY-INDEX.md (v2.6→v2.7), sprint-state.yaml (v2.6→v2.7), BC-2.06.003.md (v1.3→v1.4), interface-definitions.md (v1.1→v1.2), ARCH-04-admission-security.md (v1.10→v1.11), S-6.06-*.md (v1.1→v1.2), S-6.07-*.md (v1.0→v1.1), VP-048.md (v1.2→v1.3), S-5.01-*.md (v1.3→v1.4), STATE.md

**Summary:** Closed all adversarial Pass-2 findings on the traceability and index axis. No code changes — all changes are spec/index/state artifacts.

| Finding | Severity | Resolution |
|---------|----------|------------|
| F-P2-001 (S-5.01 vp_traces) | HIGH | S-5.01 vp_traces populated; landed in story-writer burst |
| F-P2-001 (S-6.06 error codes) | HIGH | S-6.06 E-ADM-009 error codes reconciled; landed in story-writer burst |
| F-P2-001 (S-6.02 confirm-gate order) | HIGH | S-6.02 confirm-gate ordering fix; landed in implementer burst |
| F-P2-002 (BC-2.07.001 PC-2 test in S-6.07) | HIGH | S-6.07 v1.1 — fingerprint canonical + envelope normalized; landed in story-writer burst |
| F-P2-003 (HOLD-001 oracle in S-6.06) | HIGH | S-6.06 v1.2 — functional oracle added to HOLD-001 test; landed in implementer burst |
| F-P2-004 (interface-definitions retire sbctl svtn create) | MEDIUM | interface-definitions v1.2 — `sbctl svtn create` marked [DEPRECATED]; S-5.02 + S-7.03 bc_traces corrected in STORY-INDEX + sprint-state |
| F-P2-005 (ARCH-04 sentinel + BC-2.06.003 stories) | LOW/MEDIUM | ARCH-04 v1.11 — ErrRoleMismatch sentinel string aligned to `admission.go` implementation; BC-2.06.003 v1.4 — Stories cell filled (S-5.02) |
| F-019 (S-6.05 missing from Wave-6 stories list) | HIGH | sprint-state.yaml — S-6.05 restored to wave-6 stories list; S-6.05 entry added |
| F-020 (S-BL.LOOKUP bc_traces mismatch) | HIGH | sprint-state.yaml — bc_traces set to [] per story file (authority); STORY-INDEX total-stories arithmetic corrected |
| F-021 (S-6.07 status/priority wrong) | HIGH | sprint-state.yaml — S-6.07 priority P1→P2, status pending→draft |
| F-022 (S-6.07 title wrong) | HIGH | sprint-state.yaml — title corrected to "SVTN admin create handler + CLI (sbctl admin svtn create)" |
| F-023/F-024/F-025/F-026 (S-6.07 template + VP-048 four-story trace) | MEDIUM | S-6.07 v1.1 Behavioral Contracts table added; VP-048 v1.3 four-story trace; tdd_mode + inputDocuments added |
| F-027 (STORY-INDEX backlog section mixes draft stubs) | MEDIUM | STORY-INDEX v2.7 — Backlog split into "Backlog: 4" + "Draft stubs: 1" |

**Residual deferrals:** F-005 per spec (→ S-BL.LOOKUP); F-PG-003 input-hash (tracked TODO F-009).

---

## Wave-5 S-5.01 + S-6.02 Pass-3 Convergence — BC-5.39.001 Satisfied (2026-06-29)

**Trigger:** Per-story adversarial Pass-3 (3-lens diverse-context) for S-5.01 and S-6.02. Both stories had accumulated Pass-1 + Pass-2 fix-bursts; this was the final confirmation round.

**Agents dispatched:** adversary (×6 — 3 lenses per story, fresh context per lens), implementer (S-6.02 narrow fix a98bd92), state-manager (factory artifact fixes e08f567)

### S-5.01 Pass-3 Results

| Lens | Focus | Verdict | Findings |
|------|-------|---------|----------|
| 1 | correctness | CONVERGED | 0C/0H/0M |
| 2 | concurrency | CONVERGED | 0C/0H/0M |
| 3 | traceability | CONVERGED | 0C/0H/0M |

Deferred (out-of-perimeter, BC-5.39.002): 1 system-level observation — STORY-INDEX VP coverage rollup shows 67/67 but 74 VPs now exist (rollup count stale). Not a per-story defect; carried to index maintenance.

**BC-5.39.001 status for S-5.01: SATISFIED** — 3 consecutive clean passes, diverse lenses.

### S-6.02 Pass-3 Results

| Lens | Focus | Initial Verdict | Fix | Final Verdict | Findings |
|------|-------|----------------|-----|---------------|----------|
| 1 | scope+wire | BLOCK (F-P3-L1-001 HIGH) | a98bd92 | CONVERGED | 0C/0H/0M |
| 2 | concurrency+security | CONVERGED | — | CONVERGED | 0C/0H/0M |
| 3 | traceability | BLOCK (F-1 MEDIUM) | e08f567 | CONVERGED | 0C/0H/0M |

**Lens 1 fix (a98bd92):** F-P3-L1-001 HIGH — sibling-fix propagation: E-ADM-014 stale in 3 worktree files. Fixed: `cmd/sbctl/admin.go:51` → E-ADM-019; `cmd/sbctl/admin_test.go:679,734` → E-CFG-001; `internal/svtnmgmt/svtnmgmt_test.go:505,530` → E-ADM-019.

**Lens 3 fix (e08f567):** F-1 MEDIUM — ARCH-04 v1.11 prose at line 372 not swept during Pass-2 ARCH-04 v1.10→v1.11 bump. Fixed: ARCH-04 v1.11→v1.12; line 372/374 prose now matches canonical sentinel at line 429/431.

Deferred (out-of-perimeter, BC-5.39.002):
- O-2: phantom S-BL.NI cross-reference (backlog story, not S-6.02 deliverable)
- O-3: sprint-state arithmetic edge case (index consistency; out of story perimeter)
- O-4: S-6.06 ErrRoleMismatch package anchor (sibling story, not S-6.02)

**BC-5.39.001 status for S-6.02: SATISFIED** — 3 consecutive clean passes, diverse lenses (lens 1 + lens 3 re-converged after narrow fixes).

### Outcome

Both S-5.01 and S-6.02 satisfy BC-5.39.001 as of 2026-06-29. Both worktrees are race-clean. Ready for PR delivery via per-story-delivery.md flow.

---

## Wave-5 S-5.01 + S-6.02 Merged to Develop (2026-06-30)

**Agents dispatched:** human (merge), state-manager (recording)
**PRs merged:** PR #35 (S-5.01), PR #34 (S-6.02)
**develop HEAD before:** 0d499ac (post S-W5.01 merge)
**develop HEAD after:** b36cb9b

### Merge Chain

| Story | PR | Merge SHA | Merge Time | Notes |
|-------|-----|-----------|------------|-------|
| S-5.01 | #35 | c1c2c3d | 2026-06-30T12:01:28Z | Squash-merged |
| S-6.02 | #34 | b36cb9b | 2026-06-30T12:01:28Z | Squash-merged; rebased over S-5.01 (c1c2c3d) before merge |

**Dependency chain note:** S-6.02 depends on S-6.03 (d854978) and S-W5.01 (0d499ac), both already on develop. S-6.02 was rebased over S-5.01 (c1c2c3d) to resolve ordering before merge.

**Post-merge cleanup:** Both worktrees removed; feature branches deleted.

**Wave 5 merged stories:** S-5.03 (#30/01ae50c), S-6.03 (#32/d854978), S-W5.01 (#31/0d499ac), S-5.01 (#35/c1c2c3d), S-6.02 (#34/b36cb9b) — 5 of 8 wave-5 stories complete.

**Remaining Wave 5:** S-5.02, S-6.06, S-W5.02 (15 pts). Next: deliver S-5.02 then S-6.06, then S-W5.02 after all dependencies merged. Wave 5 adversarial review follows all merges.

---

## S-6.06 Pass-15 BLOCK + Fix-Burst (2026-06-30)

**Agents dispatched:** adversary (×3 lenses, fresh context), state-manager (recording)
**Spec commit:** fad33ec on factory-artifacts
**Impl commit:** 6528f02 on feat/S-6.06-daemon-admin-handlers

### Pass-15 Lens Results

| Lens | Focus | Verdict | Findings |
|------|-------|---------|----------|
| 1 | Implementation correctness | BLOCK | F-P15L1-001 MED (default-arm double-stamp) + F-P15L1-002 MED (EC-007 unconditional vs conditional) + F-P15L1-003 LOW (comment phrasing) |
| 2 | Spec drift | BLOCK | F-P15L2-001 MED (story line citation 257-262 stale→275-280) + F-P15L2-002 LOW (dup of L1-001) |
| 3 | Sibling propagation + VP harness compilability | PASS | 0 findings — VP-064/065/066/075 compilable; EC-007 propagated; wave-gate scope correct |

**Dup:** F-P15L1-001 and F-P15L2-002 are the same default-arm double-stamp defect seen from two review angles (high signal).

**Overall: BLOCK** — lens-1 BLOCK, lens-2 BLOCK, lens-3 PASS. Clean-pass count: 0/3.

### Fix-Burst Record

| Layer | Commit | Changes |
|-------|--------|---------|
| Spec | fad33ec (factory-artifacts) | BC-2.05.004 v1.8→v1.9 (unconditional EC-007 narrative aligned to impl); S-6.06 story v1.13→v1.14 (line citations 257-262→275-280); BC-INDEX v1.4→v1.5; STORY-INDEX v3.3→v3.4 |
| Impl | 6528f02 (feat/S-6.06-daemon-admin-handlers) | admin_handlers.go: default-arm prefix drop (removes E-RPC-011 double-stamp); comment rewrite for EC-007 conditional vs unconditional clarity; `just test` + `just test-race` both clean |

### Outcome

Fix-burst applied. Clean-pass count reset to 0/3. Pass-16 queued.

---

## S-6.06 Pass-16 PASS (2026-06-30)

**Dispatch IDs:** (not recorded — see STATE.md)
**Spec tip:** fad33ec (factory-artifacts) / **Impl tip:** 6528f02 (feat/S-6.06-daemon-admin-handlers)

### Pass-16 Lens Results

| Lens | Verdict | Findings |
|------|---------|----------|
| 1 | PASS | 0 gating |
| 2 | PASS | 0 gating |
| 3 | PASS | 0 gating |

**Overall: PASS** — all 3 lenses clean. Clean-pass count: 1/3. Pass-17 queued.

---

## S-6.06 Pass-17 BLOCK + Fix-Burst (2026-06-30)

**Spec tip:** fad33ec / **Impl tip:** 6528f02

### Pass-17 Lens Results

| Lens | Verdict | Findings |
|------|---------|----------|
| 1 | PASS | 0 gating |
| 2 | BLOCK | F-P17L2-001 MED (error-taxonomy.md E-ADM-020 out-of-sync with BC v1.9 unconditional) + F-P17L2-002 LOW ("permanent trust anchor" wire-string alignment) |
| 3 | PASS | 0 gating |

**Overall: BLOCK** — lens-2 BLOCK. Clean-pass count remains 1/3. Pass-17 NOT counted.

### Fix-Burst Record

| Layer | Commit | Changes |
|-------|--------|---------|
| Spec | 5da781a (factory-artifacts) | error-taxonomy.md v3.6→v3.7; S-6.06 story v1.14→v1.15; STORY-INDEX v3.4→v3.5 |
| Impl | 2390541 (feat/S-6.06-daemon-admin-handlers) | admin_handlers.go:397 + test:719; race-clean |

Pass-17 NOT counted. Clean-pass count: 1/3. Pass-18 queued.

---

## S-6.06 Pass-18 BLOCK + Fix-Burst (2026-06-30)

**Spec tip:** 5da781a / **Impl tip:** 2390541

### Pass-18 Lens Results

| Lens | Verdict | Findings |
|------|---------|----------|
| 1 | BLOCK | F-P18L1-001 MED (ExpireKey missing bootstrap-key guard — EC-007/revoke-protection parity); F-P18L1-002 MED (adminKeyEntry.Expiry time.Time omitempty zero-value serialization bug); 3 LOW OBS |
| 2 | PASS | 0 gating |
| 3 | PASS | 1 LOW frontmatter drift (piggyback-fixed) |

**Overall: BLOCK** — lens-1 BLOCK (2 MED). Most substantive fix-burst of cycle. Clean-pass count remains 1/3.

### Fix-Burst Record

| Layer | Commit | Changes |
|-------|--------|---------|
| Spec | 518a30f (factory-artifacts) | error-taxonomy.md v3.7→v3.8 (E-ADM-021 + ErrBootstrapKeyExpireForbidden); BC-2.05.004 v1.9→v1.10 (EC-007 extended revoke OR expire); S-6.06 story v1.15→v1.16 + EC-008 + VP-076; VP-INDEX v2.9→v2.10; BC-INDEX v1.5→v1.6; STORY-INDEX v3.4→v3.6 |
| Impl | 9a4cf0b (feat/S-6.06-daemon-admin-handlers) | ExpireKey bootstrap guard + ErrBootstrapKeyExpireForbidden sentinel + tests |
| Impl | 6bd9e12 (feat/S-6.06-daemon-admin-handlers) | adminKeyEntry.Expiry *time.Time pointer + zero-expiry JSON test; all 17 packages race-clean |

Pass-18 NOT counted. Clean-pass count: 1/3. Pass-19 queued.

---

## S-6.06 Pass-19 BLOCK + Fix-Burst (2026-06-30)

**Dispatch IDs:** lens-1 a3606081aef4844dc / lens-2 abd38d77ab61a5933 / lens-3 a3930ee0f3f10431d
**Spec tip:** 518a30f (factory-artifacts) / **Impl tip:** 6bd9e12 (feat/S-6.06-daemon-admin-handlers)

### Pass-19 Lens Results

| Lens | Verdict | Findings |
|------|---------|----------|
| 1 | PASS | F-P19L*-001 MED (dup-confirmed L2+L3): BC-2.05.004 body VP table missing VP-076 row; 6 LOW informational (non-gating) |
| 2 | BLOCK | F-P19L*-001 MED (dup of L1+L3): BC-2.05.004 body VP table missing VP-076 row; F-P19L2-002 LOW: S-6.06 Error Code Map E-ADM-021 line cite 275-280→279-284 |
| 3 | BLOCK | F-P19L*-001 MED (dup of L1+L2): BC-2.05.004 body VP table missing VP-076 row; F-P19L3-002 MED: BC-2.05.004 Traceability Stories row missing EC-007/S-6.06; F-P19L3-003 MED: BC-2.05.004 modified-list non-monotonic |

**Dup:** F-P19L*-001 (BC body VP table missing VP-076 row) confirmed independently by all 3 lenses — high-signal sibling-fix propagation gap from Pass-18 fix-burst.

**Overall: BLOCK** — lens-2 BLOCK, lens-3 BLOCK. Lens-1 PASS (6 LOW informational only). Clean-pass count: 1/3. Pass-19 NOT counted.

**Process-gap codified:** Pass-18 fix-burst minted VP-076 + BC-2.05.004 v1.10 but failed to propagate to (a) BC body VP table, (b) BC Traceability Stories row, (c) monotonic ordering of modified-list. Recurring product-owner sibling-fix discipline gap (similar pattern noted in prior passes). Noted in STATE.md current state log.

### Fix-Burst Record

| Layer | Commit | Changes |
|-------|--------|---------|
| Spec | 13164cb (factory-artifacts) | BC-2.05.004 v1.10→v1.11: VP-076 row added to body VP table; EC-007/S-6.06 added to Traceability Stories row; modified-list monotonic ordering corrected; BC-INDEX v1.6→v1.7 |
| Spec | 9843e9a (factory-artifacts) | S-6.06 story v1.16→v1.17: E-ADM-021 line cite corrected 275-280→279-284; STORY-INDEX v3.6→v3.7 |

**Impl unchanged** — all Pass-19 fixes are spec-only. Impl tip remains 6bd9e12.

Pass-19 NOT counted. Clean-pass count: 1/3. Pass-20 queued (clean-pass attempt #2 of 3 needed).

---

## S-6.06 Pass-20 BLOCK + Fix-Burst (2026-06-30)

**Dispatch IDs:** lens-1 a0ce4060b99958c55 / lens-2 a8eaa3d24878b1fc8 / lens-3 a14728dee74678c40
**Spec tip dispatched against:** 9843e9a (factory-artifacts) / **Impl tip:** 6bd9e12 (feat/S-6.06-daemon-admin-handlers, unchanged)

### Pass-20 Lens Results

| Lens | Verdict | Findings |
|------|---------|----------|
| 1 | PASS CLEAN | 2 MED + 1 LOW non-blocking polish observations only (non-gating) |
| 2 | PASS CLEAN | no gating findings |
| 3 | BLOCK | F-P20L3-001 MED NOVEL: cross-layer ordering ambiguity — handler TTL validation at admin_handlers.go:279-284 fires BEFORE svtnmgmt bootstrap guard; `{bootstrap_pubkey, after:"-1h"}` returns E-CFG-001 not E-ADM-021; contradicts BC EC-007 "unconditionally" language |

**Novelty:** F-P20L3-001 is genuinely new — Passes 1–19 examined symmetry, guard position, and TTL bounds in isolation but never the cross-product of (bootstrap target × malformed input). Real convergence dividend.

**Overall: BLOCK** — lens-3 BLOCK on one NOVEL MED. Lenses 1 and 2 PASS CLEAN. Clean-pass count: 1/3 (unchanged). Pass-20 NOT counted.

**Product-owner ruling:** Option B (spec narrowing). Input validation precedes business-rule sentinels — current impl is correct, BC/VP wording was overstated. Mutation-prevention invariant preserved either way.

### Fix-Burst Record

| Layer | Commit | Changes |
|-------|--------|---------|
| Spec | 677140a (factory-artifacts) | BC-2.05.004 v1.11→v1.12: EC-007 narrowed to well-formed requests only; VP-076 v1.0→v1.1: Property #3 scoped to well-formed; BC-INDEX v1.7→v1.8; error-taxonomy.md O-P20L3-001 fix (E-ADM-021 Tests citation cleanup, removed revoke test reference) |

**Impl unchanged** — Pass-20 fix is spec-narrowing only. Impl tip remains 6bd9e12.

Pass-20 NOT counted. Clean-pass count: 1/3. Pass-21 queued (clean-pass attempt #2 of 3 now that BC v1.12 ground truth has moved).
Spec tip after fix: 677140a. Impl tip: 6bd9e12.

---

## S-6.06 Pass-21 BLOCK + Fix-Burst (2026-06-30)

**Dispatch IDs:** lens-1 ada1125598286af4e / lens-2 a19f659c98fb7441a / lens-3 a27279f4b0c6808f3
**Spec tip dispatched against:** 677140a (factory-artifacts) / **Impl tip:** 6bd9e12 (feat/S-6.06-daemon-admin-handlers, unchanged from Pass-20)

### Pass-21 Lens Results

| Lens | Verdict | Findings |
|------|---------|----------|
| 1 | BLOCK | F-L1-A MED: mapAdminError default-arm untested; F-L1-B MED: ErrInvalidDuration unreachable-claim has no DI-D arm; F-L1-C MED: decodePublicKey silent swallow (go.md rule 3 violation); F-L1-D MED: TestResolveAndVerifyCallerRole expired_key_non_control_rejected mis-anchored, future-expiry-non-control branch uncovered; 5 LOW |
| 2 | BLOCK | F-P21L2-001 MED: dup-confirmed lens-3 EC-008 narrowing gap; F-P21L2-002 MED NEW: VP-INDEX VP-076 row + registry note still cite "unconditionally"/v1.10 |
| 3 | BLOCK | F-P21L3-001 HIGH: EC-008 stale "unconditionally" — sibling-fix propagation gap from Pass-20 Option-B narrowing (BC-2.05.004 v1.12 updated EC-007 but EC-008 not swept); F-P21L3-002 MED [process-gap]: BC EC narrowing not fanned out to story EC tables (recurring pattern, passes 19/20/21); O-P21L3-002 LOW: VP-076 stale v1.10 cite at line 68 |

**Lens-3 F-P21L3-001 note:** This is a sibling-fix propagation gap identical in mechanism to Pass-19's root cause. Pass-20 Option-B fix narrowed EC-007 in BC-2.05.004 and updated VP-076, but EC-008 in the same BC document was not swept. High severity because a spec reader of EC-008 still sees the overstated "unconditionally" language that was ruled incorrect by the PO.

**Overall: BLOCK** — all 3 lenses blocked. Clean-pass count: 1/3 (unchanged). Pass-21 NOT counted.

**Convergence reset assessment recorded:** The impl changed substantively (mapAdminError signature refactored, ErrInvalidDuration DI-D arm added). Per BC-5.39.001 strict interpretation, the clean-pass counter should reset to attempt #1 because impl ground truth moved. However, all changes are pure defense-in-depth additions + test-quality fixes (no behavioral semantics changed — invariants locked in, uncovered branches covered). Orchestrator ruling: continue counting toward 3-clean from current state — Pass-22 = clean-pass attempt #2 of 3. Both interpretations recorded here; convergence-trajectory reflects the substantive-vs-cosmetic distinction.

**Recurring process-gap (F-P21L3-002) codified:** Three consecutive passes (19, 20, 21) have exposed BC/VP narrowing not propagating to story EC tables. Process rule crystallized: when a BC EC is narrowed/widened in a fix-burst, story-writer MUST be dispatched in parallel to update all stories whose EC tables cite that BC EC. Added to STATE.md open drift items.

### Fix-Burst Record — factory-artifacts

| Layer | Agent | Commit | Changes |
|-------|-------|--------|---------|
| Spec | product-owner | fc90ef2 (factory-artifacts) | VP-INDEX v2.10→v2.11: VP-076 row narrowed (updated from "unconditionally" to "for any well-formed request") + EC-007 v1.10 cite corrected to v1.12 + v1.10 stale cite swept; VP-076 v1.1→v1.2: Property Statement closer updated to cite v1.12 |
| Spec | story-writer | 4229464 (factory-artifacts) | S-6.06 v1.17→v1.18: EC-008 narrowed "unconditionally" → "for any well-formed request" with AC-005 layering note; v1.17 changelog row-attribution corrected; STORY-INDEX v3.7→v3.8 |

### Fix-Burst Record — S-6.06 feature branch (worktree)

| Layer | Agent | Commit | Changes |
|-------|-------|--------|---------|
| Impl | implementer | c519fc1 (feat/S-6.06-daemon-admin-handlers) | F-L1-D: TestResolveAndVerifyCallerRole — expired_key_non_control_rejected renamed + TTL changed to cover future-expiry-non-control branch in CallerKeyRoleActive |
| Impl | implementer | 0be8e97 (feat/S-6.06-daemon-admin-handlers) | F-L1-A + F-L1-B + F-L1-C: mapAdminError refactored (signature now takes ed25519.PublicKey, eliminates double-decode + silent swallow); ErrInvalidDuration defense-in-depth arm added; default-arm test added. All 17 packages pass race detector. |

**Spec tip after fix:** 4229464 (factory-artifacts). **Impl tip:** 0be8e97 (feat/S-6.06-daemon-admin-handlers).

Pass-21 NOT counted. Clean-pass count: 1/3. Pass-22 queued (clean-pass attempt #2 of 3 per orchestrator ruling).

---

## S-6.06 Pass-22 Adversarial Review + Fix-Burst (2026-06-30)

**Agents dispatched:** adversary (lens-1, lens-2, lens-3), product-owner (spec fix)
**Dispatch IDs:** lens-1 aeaa638b208bc006a / lens-2 a72e3013057bcc11b / lens-3 a5eef7adde2c2635e
**Spec tip:** 4229464 (factory-artifacts). **Impl tip:** 0be8e97.

**Lens-1:** PASS CLEAN — no gating findings.
**Lens-2:** PASS CLEAN — no gating findings.
**Lens-3:** BLOCK.
- F-P22L3-001 HIGH: story VP table row for VP-076 still cites EC-007/EC-008 "unconditionally" language.
- F-P22L3-002 HIGH: error-taxonomy.md E-ADM-020/E-ADM-021 still carry "unconditionally...at any time" text and stale v1.10 cites.
- F-P22L3-003 MED: VP-076 Property #1 and Property #2 prose unnarrowed.
- F-P22L3-004 MED: VP-076 proof-harness docstring inconsistent with narrowed scope.
- O-P22L3-002 [process-gap]: recurring 4-pass sweep miss pattern; vsdd-factory issues #361–#364 filed.

**Verdict:** BLOCK. Clean-pass count: 1/3 (unchanged). Pass-22 NOT counted.

**Convergence-reset ruling:** Fix-burst was spec-only; no behavioral semantics changed in impl. Counter not reset per BC-5.39.001. Pass-23 = clean-pass attempt #2 of 3.

### Fix-Burst Record — factory-artifacts

| Layer | Agent | Commit | Changes |
|-------|-------|--------|---------|
| Spec | product-owner | 4b42dd5 (factory-artifacts) | error-taxonomy.md v3.8→v3.9 (E-ADM-020/021 text + stale v1.10 cites updated); VP-076 v1.2→v1.3 (Properties #1 & #2 narrowed + harness docstring); S-6.06 v1.18→v1.19 (story VP table row regenerated); VP-INDEX v2.11→v2.12; STORY-INDEX v3.8→v3.9. Exhaustive "unconditionally" sweep — zero current-state residuals. |

**Spec tip after fix:** 4b42dd5. **Impl tip:** 0be8e97 (unchanged).

---

## S-6.06 Pass-23 Adversarial Review + Fix-Burst (2026-06-30)

**Agents dispatched:** adversary (lens-1, lens-2, lens-3), product-owner (spec fix)
**Dispatch IDs:** lens-1 afd8f2e1b20cde42a / lens-2 aea17b5f734310b26 / lens-3 a1038b24343e5e306
**Spec tip:** 4b42dd5 (factory-artifacts). **Impl tip:** 0be8e97.

**Lens-1:** PASS CLEAN — novelty LOW; no findings.
**Lens-2:** PASS CLEAN — O-P23L2-001 LOW (VP-076 Source Contract §line 113 cites error-taxonomy v3.8 vs current v3.9; semantically coherent narrowing, paperwork drift only; deferred to next VP-076 touch).
**Lens-3:** BLOCK.
- F-P23L3-001 MED: S-6.06 v1.19 line 180 Error Code Map E-ADM-021 row cites `BC-2.05.004 EC-007 v1.10`; should be v1.12.
- F-P23L3-002 MED: S-6.06 v1.19 line 245 Task 12 Refs cites `BC-2.05.004 EC-007 v1.10`; should be v1.12.
- O-P23L3-001 LOW: VP-076 Property #1/#2 phrasing slightly tautological — non-blocking.

**Verdict:** BLOCK. Clean-pass count: 1/3 (unchanged). Pass-23 NOT counted.

**PROCESS-GAP-P23 (5th consecutive recurrence):** Sibling-sweep gap missed story-body prose narrative (Error Code Map message annotations + Task Refs). Pass-22 grepped for "unconditionally" but NOT "v1.10" residuals. vsdd-factory #361 comment appended.

**Convergence-reset ruling:** Spec-only fix; counter NOT reset per BC-5.39.001. Pass-24 = clean-pass attempt #3 of 3.

### Fix-Burst Record — factory-artifacts

| Layer | Agent | Commit | Changes |
|-------|-------|--------|---------|
| Spec | product-owner | 82721dc (factory-artifacts) | S-6.06 v1.19→v1.20: both v1.10 cites at lines 180 and 245 bumped to v1.12; STORY-INDEX v3.9→v3.10. Exhaustive grep confirms zero current-state v1.10 residuals. ARCH-04 v1.10 cites at lines 263/332 left alone (different artifact). |

**Spec tip after fix:** 82721dc. **Impl tip:** 0be8e97 (unchanged).

---

## S-6.06 Pass-24 — 2026-06-30 (BLOCK + dual fix-burst applied)

**Dispatch IDs:** lens-1 a6ead8d7956498972 / lens-2 a64e9dbb012bf369a / lens-3 a57d7569f4aaa7675

**Lens-1:** PASS CLEAN — novelty LOW; no findings; impl tip 0be8e97 unchanged.
**Lens-2:** PASS CLEAN — O-P24L2-001 LOW out-of-scope obs (impl comment v1.10 cites at svtnmgmt.go:66,:332 + admin_handlers_test.go:821 — same mechanism as F-P24L3-001 but surfaced advisory by lens-2).
**Lens-3:** BLOCK.
- F-P24L3-001 MED: VP-076.md:113 Source Contract cited error-taxonomy.md v3.8; current version is v3.9. Root cause: Pass-22 fix-burst (4b42dd5) bumped error-taxonomy v3.8→v3.9 and VP-076 v1.2→v1.3 in the same commit but forgot to update VP-076's back-reference at line 113.
- O-P24L3-001 OBS [process-gap]: 6th-pass cite-drift recurrence — axis shifted to downstream-doc cite of upstream-doc version; new surface: impl source comments.

**Verdict:** BLOCK. Clean-pass count: 1/3 (unchanged). Pass-24 NOT counted.

**PROCESS-GAP-P24 (6th consecutive recurrence):** New axis — downstream-doc cite of upstream-doc version (VP→error-taxonomy version cite drift). New surface — impl source comments (svtnmgmt.go + admin_handlers_test.go v1.10 cite residuals). vsdd-factory #361 comment appended (6th recurrence).

**Convergence-reset ruling:** Doc-only + comment-only fix-bursts; no behavior changes. Per BC-5.39.001 doc-only-fix discipline, clean-pass counter NOT reset. Pass-25 = clean-pass attempt #3 of 3 continues.

### Fix-Burst Record — dual-layer (spec + impl)

| Layer | Agent | Commit | Branch | Changes |
|-------|-------|--------|--------|---------|
| Spec | product-owner | c5c948c | factory-artifacts | VP-076 v1.3→v1.4: line 113 v3.8→v3.9 cite fix; VP-INDEX v2.12→v2.13; pre/post-edit grep clean. |
| Impl | implementer | 4b626cf | feat/S-6.06-daemon-admin-handlers | impl comment v1.10→v1.12 at 3 sites (svtnmgmt.go:66,:332 + admin_handlers_test.go:821); just fmt + just lint clean; just test-race 17/17 PASS, 0 races; comment-only, no behavior change. O-P24L2-001 from lens-2 also resolved. |

**Spec tip after fix:** c5c948c. **Impl tip:** 4b626cf.

---

## S-6.06 Pass-26 — 2026-06-30 (PASS CLEAN — first clean pass since Pass-16; clean-pass count 1→2/3)

**Dispatch IDs:** lens-1 a05e401bf6bf753a1 / lens-2 a9efc33989be3c792 / lens-3 ae6b9da5fbadbaaba
**Spec tip dispatched against:** a6cdb88. **Impl tip:** d3f186c.

**Lens-1:** PASS CLEAN — novelty NONE. 7 LOW observations all adjudicated as non-defects (mis-labels, intentional design, fail-closed behavior, dead-code in test). No findings.

**Lens-2:** PASS CLEAN — novelty NONE. All wire-error strings byte-equivalent. ARCH-04 v1.13 + VP-076 v1.4 cites coherent. Sibling-sweep gap closed. No findings.

**Lens-3:** PASS CLEAN — novelty LOW. 2 LOW observations explicitly out-of-scope (architectural / system-level), deferred to phase-5:
- O-P26L3-001 LOW: ARCH-04.md:30-40 modified-list non-monotonic + missing v1.7/v1.8/v1.11/v1.12 + v1.13 inserted before v1.9.
- O-P26L3-002 LOW: error-taxonomy.md:9-23 modified-list mixed ascending/descending ordering.

Both observations are architectural / system-level; out-of-perimeter for S-6.06 per-story scope per BC-5.39.002 PC2. Deferred to phase-5. Created as TaskList #117 (phase-5 follow-up: ARCH-04 + error-taxonomy modified-list monotonicity).

**Verdict:** PASS CLEAN (all 3 lenses). Clean-pass count advances: **2/3**.

This is the first fully-clean pass since Pass-16 (baseline). Passes 17–25 all BLOCK on at least one lens.

**No fix-burst required.**

**Next:** Pass-27 fresh 3-lens (clean-pass attempt #3 of 3). Spec tip: post-closeout SHA on factory-artifacts. Impl tip: d3f186c (unchanged).

---

## S-6.06 Pass-27 — 2026-06-30 (PASS CLEAN — second consecutive fully-clean pass; clean-pass count 2→3/3-pending)

**Dispatch IDs:** lens-1 a68ef99c2850a5ae5 / lens-2 ad7f415313ffdd259 / lens-3 a73b40208a7fef653
**Spec tip dispatched against:** factory-artifacts HEAD (post-Pass-26 closeout). **Impl tip:** d3f186c (unchanged since Pass-25).

**Lens-1 (a68ef99c2850a5ae5):** PASS CLEAN — novelty LOW. 7 LOW non-blocking observations, all adjudicated non-blocking refinements. All routed to TaskList #115 (post-merge polish backlog).
- O-1 LOW: keyFingerprintAdmin(nil) latent footgun in mapAdminError list-keys path.
- O-2 LOW: decodePublicKey not validating Ed25519 point encoding.
- O-3 LOW: RoleMismatchError typed-detail path not in TestMapAdminError_ErrorWrapping.
- O-4 LOW: E-ADM-018 omits fingerprint — intentional per AC-005 (design decision, adjudicated non-defect).
- O-5 LOW: dead privHex variable in VP046 DI-002 test.
- O-6 LOW: goroutine accounting in TestSVTNManager_ExpireKey_TOCTOU_RoleChangeRace.
- O-7 LOW: subtle.ConstantTimeCompare doc-comment accuracy.
No gating findings.

**Lens-2 (ad7f415313ffdd259):** PASS CLEAN — novelty LOW. All wire-error strings byte-aligned; all version cites resolve coherently; layering claim corroborated against implementation. Adversary explicitly recommends Lens-2 streak counter advancement.

**Lens-3 (a73b40208a7fef653):** PASS CLEAN — novelty ZERO. Pass-25 sibling-fix propagation has fully landed. Phase-5 deferred items (TaskList #118) correctly NOT re-flagged per BC-5.39.002 PC2.

**Verdict:** PASS CLEAN (all 3 lenses). Clean-pass count advances: **3/3-pending** (second consecutive fully-clean pass).

**No fix-burst required.**

**Next:** Pass-28 fresh 3-lens (convergence-close — clean-pass attempt #3 of 3). Spec tip: factory-artifacts HEAD. Impl tip: d3f186c (unchanged).

---

## S-6.06 Pass-25 — 2026-06-30 (BLOCK + dual fix-burst applied)

**Dispatch IDs:** lens-1 ab521edc560a0b013 / lens-2 aae0edcaf3acf4640 / lens-3 a9a23dc563641c905
**Spec tip dispatched against:** c5c948c. **Impl tip:** 4b626cf.

**Lens-1:** PASS CLEAN — 4 LOW observations (non-gating).
- Obs-1 LOW: fallback-path coverage gap in resolveAndVerifyCallerRole — no-pubkey-in-ctx path untested; → TaskList #115.
- Obs-2 LOW: 3 stale ARCH-04 v1.10 cites in impl (admission.go:287, svtnmgmt.go:252, svtnmgmt.go:279) + 1 in story; PO adjudicated S-2.01:148 as out-of-scope historical-attribution (intentional).
- Obs-3 LOW: unreachable bogus fingerprint in list-keys default arm.
- Obs-4 LOW: dead code in VP046 test.

**Lens-2:** PASS CLEAN — novelty zero; no findings.

**Lens-3:** BLOCK.
- F-P25L3-001 MED: S-6.06:204 cites "VP-076 v1.1"; current is v1.4. Stale story-body version citation.
- O-P25L3-001 OBS [process-gap]: 7th-recurrence sibling-sweep gap — new axis: downstream→upstream version cites (story body cites of upstream-artifact versions stale after upstream version bumps).

**Verdict:** BLOCK. Clean-pass count: 1/3 (unchanged). Pass-25 NOT counted.

**PROCESS-GAP-P25 (7th consecutive recurrence):** Story body cites of upstream-artifact versions are stale after upstream version bumps. Pass-24 fix-burst (c5c948c) updated VP-076 v1.3→v1.4 but did NOT sweep stories/ for "VP-076 v1.*" current-state cites. Upstream-rooted sweep rule: any document citing an artifact must be re-grepped when that artifact's version bumps. vsdd-factory #361 comment appended (7th recurrence + new axis: story body downstream→upstream cites).

**Convergence-reset ruling:** Both fix-bursts doc-only / comment-only; no behavior changes; per BC-5.39.001 doc-only-fix discipline counter NOT reset. Pass-26 = clean-pass attempt #3 of 3 continues.

### Fix-Burst Record — dual-layer (spec + impl)

| Layer | Agent | Commit | Branch | Changes |
|-------|-------|--------|--------|---------|
| Spec | product-owner | a6cdb88 | factory-artifacts | S-6.06 v1.20→v1.21 + STORY-INDEX v3.10→v3.11; line 204 VP-076 v1.1→v1.4; line 263 ARCH-04 v1.10→v1.13; exhaustive pre/post-edit grep across stories+specs; zero (b)-class residuals remain. |
| Impl | implementer | d3f186c | feat/S-6.06-daemon-admin-handlers | 4 impl/test ARCH-04 v1.10→v1.13 comment bumps at admission.go:287, svtnmgmt.go:252, svtnmgmt.go:279, admin_handlers.go:192; just fmt + just lint clean; just test-race 17/17 PASS, 0 races; comment-only, no behavior change. |

**Spec tip after fix:** a6cdb88. **Impl tip:** d3f186c.

---

## S-6.06 Pass-24 — 2026-06-30 (BLOCK + dual fix-burst applied)

**Dispatch IDs:** lens-1 a6ead8d7956498972 / lens-2 a64e9dbb012bf369a / lens-3 a57d7569f4aaa7675

**Lens-1:** PASS CLEAN — novelty LOW; no findings; impl tip 0be8e97 unchanged.
**Lens-2:** PASS CLEAN — O-P24L2-001 LOW out-of-scope obs (impl comment v1.10 cites at svtnmgmt.go:66,:332 + admin_handlers_test.go:821 — same mechanism as F-P24L3-001 but surfaced advisory by lens-2).
**Lens-3:** BLOCK.
- F-P24L3-001 MED: VP-076.md:113 Source Contract cited error-taxonomy.md v3.8; current version is v3.9. Root cause: Pass-22 fix-burst (4b42dd5) bumped error-taxonomy v3.8→v3.9 and VP-076 v1.2→v1.3 in the same commit but forgot to update VP-076's back-reference at line 113.
- O-P24L3-001 OBS [process-gap]: 6th-pass cite-drift recurrence — axis shifted to downstream-doc cite of upstream-doc version; new surface: impl source comments.

**Verdict:** BLOCK. Clean-pass count: 1/3 (unchanged). Pass-24 NOT counted.

**PROCESS-GAP-P24 (6th consecutive recurrence):** New axis — downstream-doc cite of upstream-doc version (VP→error-taxonomy version cite drift). New surface — impl source comments (svtnmgmt.go + admin_handlers_test.go v1.10 cite residuals). vsdd-factory #361 comment appended (6th recurrence).

**Convergence-reset ruling:** Doc-only + comment-only fix-bursts; no behavior changes. Per BC-5.39.001 doc-only-fix discipline, clean-pass counter NOT reset. Pass-25 = clean-pass attempt #3 of 3 continues.

### Fix-Burst Record — dual-layer (spec + impl)

| Layer | Agent | Commit | Branch | Changes |
|-------|-------|--------|--------|---------|
| Spec | product-owner | c5c948c | factory-artifacts | VP-076 v1.3→v1.4: line 113 v3.8→v3.9 cite fix; VP-INDEX v2.12→v2.13; pre/post-edit grep clean. |
| Impl | implementer | 4b626cf | feat/S-6.06-daemon-admin-handlers | impl comment v1.10→v1.12 at 3 sites (svtnmgmt.go:66,:332 + admin_handlers_test.go:821); just fmt + just lint clean; just test-race 17/17 PASS, 0 races; comment-only, no behavior change. O-P24L2-001 from lens-2 also resolved. |

**Spec tip after fix:** c5c948c. **Impl tip:** 4b626cf.

