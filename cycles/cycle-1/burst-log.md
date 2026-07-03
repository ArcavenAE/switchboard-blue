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


---

## S-6.06 Pass-28 — 2026-06-30 (PASS CLEAN — CONVERGENCE-CLOSED; BC-5.39.001 satisfied)

**Dispatch IDs:** 3 fresh-context diverse-lens adversary passes (convergence-close)
**Spec tip dispatched against:** factory-artifacts HEAD (post-Pass-27 closeout, a6cdb88 lineage). **Impl tip:** d3f186c (unchanged since Pass-25).

**Lens-1 (impl-internal):** PASS CLEAN — novelty NONE. All 7 sentinel arms covered, default arm covered, %w wrapping verified, UTC discipline verified, no locked-accessor leaks, no init()/panic violations outside main, no tautological tests, comprehensive negative-path coverage, no hidden allocations, no sentinel-vs-wire drift, race/TOCTOU regression tests intact.

**Lens-2 (spec↔impl drift):** PASS CLEAN — novelty ZERO. Wire-error verbatim consistency verified; layering claim (handler input-validation before bootstrap sentinel) verified at admin_handlers.go:279-284 + svtnmgmt.go:325/334/263/268; all version cites coherent (VP-076 v1.4, ARCH-04 v1.13, BC-2.05.004 v1.12, error-taxonomy v3.9); VP-INDEX arithmetic 76 total; bidirectional traceability confirmed.

**Lens-3 (within-doc/sibling-prop):** PASS CLEAN — novelty ZERO. All five mandatory sweeps clean; Pass-25 sibling-fix propagation fully landed; known phase-5-deferred items (TaskList #118) correctly not re-flagged per BC-5.39.002 PC2.

**Verdict:** PASS CLEAN — THIRD consecutive fully-clean pass. **BC-5.39.001 CONVERGENCE-CLOSED.**

**Trajectory:** 16:PASS(1/3) → 17:BLOCK → 18:BLOCK → 19:BLOCK → 20:BLOCK → 21:BLOCK → 22:BLOCK → 23:BLOCK → 24:BLOCK → 25:BLOCK → 26:PASS(2/3) → 27:PASS(3/3-pending) → **28:PASS(3/3✓CLOSED)**

**No fix-burst required.** Spec tip at convergence: factory-artifacts HEAD. Impl tip at convergence: d3f186c.

---

## Wave-6 Tranche B Pass-6 — 2026-07-01 (BLOCK — S-BL.ROUTER-ADDR L2 blocked; S-7.01/S-7.02 CLEAN 1/3)

**Dispatch:** 9-lens aggregate (S-7.01 × 3, S-7.02 × 3, S-BL.ROUTER-ADDR × 3). Clean-attempt #1/3 reset for all three stories.

**S-7.02 (all 3 lenses):** CLEAN 1/3. All lens results clean.

**S-7.01 (all 3 lenses):** CLEAN 1/3. All lens results clean.

**S-BL.ROUTER-ADDR:** L1/L2/L3 aggregate — L2 FAILED. Finding F-P6L2-01 STALE RED-GATE: integration_test.go Part B contained a stale RED-GATE recover-guard (lines 456-469) referencing the old `paths.NewPathTracker` 3-arg signature that no longer exists after the S-7.01 partial-fix propagation. L2 finding blocked the story; S-7.01 partial-fix propagation gap exposed. Clean-pass counter reset to 0/3 for S-BL.ROUTER-ADDR.

**Pass-6 fix-burst:** test-writer dispatched for S-BL.ROUTER-ADDR. Fix: removed lines 456-469 (stale RED-GATE guard), replaced with direct `tracker := paths.NewPathTrackerWithAddr(stubAddr, 50.0, 0.125)`. Fix landed at commit **b3c93b5**. F-P6L2-01 CLOSED.

**Counter state after Pass-6:** S-7.01 1/3, S-7.02 1/3, S-BL.ROUTER-ADDR 0/3 (reset).

---

## Wave-6 Tranche B Pass-7 — 2026-07-01 (BLOCK — S-7.02 L2 blocked with 3 novel MEDIUM findings)

**Dispatch:** S-7.01 × 3 lenses (clean-attempt #2/3); S-7.02 × 3 lenses (clean-attempt #2/3); S-BL.ROUTER-ADDR pending fresh dispatch (post-b3c93b5 fix — not run this pass).

**S-7.01 (all 3 lenses):** CLEAN 2/3. All 3 lenses clean. Counter advances to 2/3.

**S-7.02:** L1 CLEAN, L3 CLEAN. L2 FAILED — 3 novel MEDIUM findings:
- F-P7L2-MED-01: tautological HMAC-first oracle (test structure validates HMAC before content, masking oracle-order sensitivity)
- F-P7L2-MED-02: TruncatesOversize maximality (boundary test does not verify maximum truncation behavior precisely)
- F-P7L2-MED-03: mid-rune exact-content (UTF-8 multi-byte boundary not tested for exact-content contract)
L2 BLOCK resets S-7.02 counter to 0/3.

**S-BL.ROUTER-ADDR:** NOT RUN this pass. Was still at 0/3 pending fresh dispatch after b3c93b5 fix. Awaiting Pass-8 dispatch.

**Pass-7 fix-burst:** test-writer dispatched for S-7.02 (F-P7L2-MED-01/02/03). SHA not yet reported — in flight.

**Counter state after Pass-7:** S-7.01 2/3, S-7.02 0/3 (reset), S-BL.ROUTER-ADDR 0/3 (pending fresh dispatch).

---

## Wave-6 Tranche B Pass-8/9 aggregate — 2026-07-01

**S-7.01:** MERGED to develop. PR #43, merge SHA 5c658e7. First Tranche B story to converge under BC-5.39.001. Worktree removed, local branch deleted. Follow-up issues CR-001/004/005/006/007 filed in parallel.

**Pass-8:** S-7.02 and S-BL.ROUTER-ADDR dispatched. S-7.02 pass-8 fix-burst addressed F-P7L2-MED-01/02/03 (test-writer). Impl HEAD at pass-8 close: a9bf936 (S-7.02), dffc27e (S-BL.ROUTER-ADDR).

**Pass-9:** S-7.02 CLEAN 2/3 at HEAD a9bf936. All 3 lenses (L1/L2/L3) clean. Novelty LOW across all lenses. No process-gap findings. S-BL.ROUTER-ADDR CLEAN 2/3 at HEAD dffc27e. All 3 lenses clean. Two LOW observations documented and non-blocking: PathEntryFromSnapshot parameter redundancy (cosmetic) + VP-047 end-to-end non-empty deferred to S-BL.PATH-TRACKER-WIRING per RULING-W6TB-B.

**Counter state after Pass-9:** S-7.01 MERGED (5c658e7 PR #43), S-7.02 2/3 (HEAD a9bf936), S-BL.ROUTER-ADDR 2/3 (HEAD dffc27e). Pass-10 dispatched for convergence-close (3/3 attempt).

---

## Wave-6 Tranche B Pass-10 + CLOSURE — 2026-07-01

**Agents dispatched:** adversary (S-7.02 × 3 lenses, S-BL.ROUTER-ADDR × 3 lenses), pr-manager (×2), devops-engineer (cleanup), state-manager (recording)

### Pass-10 Aggregate — CONVERGENCE-CLOSED (3/3 both stories)

**S-7.02 (HEAD a9bf936):** All 3 lenses CLEAN. Novelty ZERO/LOW. No gating findings. Third consecutive fully-clean pass — BC-5.39.001 SATISFIED.

**S-BL.ROUTER-ADDR (HEAD dffc27e):** All 3 lenses CLEAN. Novelty ZERO/LOW. No gating findings. Third consecutive fully-clean pass — BC-5.39.001 SATISFIED. Non-blocking LOW obs (PathEntryFromSnapshot parameter redundancy; VP-047 end-to-end deferred per RULING-W6TB-B) reclassified as out-of-perimeter and not re-flagged per BC-5.39.002 PC2.

### Merge Chain — Tranche B

| Story | PR | Merge SHA | Merge Time | Notes |
|-------|-----|-----------|------------|-------|
| S-7.01 | #43 | 5c658e7 | 2026-07-02 | Squash-merged (first to converge) |
| S-7.02 | #55 | c54a8ad | 2026-07-01 | Squash-merged |
| S-BL.ROUTER-ADDR | #56 | 91d5675 | 2026-07-01 | Squash-merged; required gh pr update-branch base catch-up |

### Force-Push Introspection

During S-BL.ROUTER-ADDR PR #56 delivery, after S-7.02 PR #55 merged, the repository's "require branches up to date" protection rule rejected PR #56's merge attempt (base SHA had advanced). The pr-manager agent reached for `git rebase` + `git push --force-with-lease` — the common fallback — but that is the wrong tool for this situation. The correct non-destructive tool is `gh pr update-branch`, which performs a base-commit-merge without rewriting history.

Auto-mode classifier correctly blocked the force-push attempt. The error was caught in real time. `gh pr update-branch` was invoked successfully on the second attempt, and PR #56 merged cleanly.

Two issues filed as a result:

- **drbothen/vsdd-factory#408** (HIGH): `pr-manager: prefer gh pr update-branch over rebase+force-push when PR base advances during convergence`. Affects the pr-manager playbook and per-story-delivery skill.
- **ArcavenAE/switchboard-blue#57** (LOW): `Tranche/parallel-worktree delivery hits merge-serialization hazard under "require branches up to date"`. Governance observation; Option A (accept gh pr update-branch as standard) adopted.

This is an own-dogfood observation: vsdd-factory#408 was filed, and `gh pr update-branch` was immediately used as the documented fix on the same delivery that surfaced the gap.

### Post-Merge Cleanup

- Worktree `.worktrees/S-BL.ROUTER-ADDR` removed (was clean before removal)
- Local branch `feat/S-BL.ROUTER-ADDR` deleted (was `122a927`)
- Remote branch `feat/S-BL.ROUTER-ADDR` deleted (via `gh pr merge --delete-branch` at merge time)
- S-7.02 worktree and branch: removed and deleted in earlier burst (per prior session)

### Follow-Up Issues Filed This Cycle

**switchboard-blue issues (filed directly):** #44–#54 (code-level LOW/nit observations from Pass-10), #57 (merge-serialization hazard governance).

**drbothen/vsdd-factory issues:** #407 (POL-001 scope unclear for INDEX artifacts; LOW), #408 (pr-manager force-push vs update-branch; HIGH).

### Tranche B Summary

| Story | BC-5.39.001 | PR | Merge SHA | Adversary Passes |
|-------|-------------|-----|-----------|-----------------|
| S-7.01 | SATISFIED | #43 | 5c658e7 | P6(1/3)→P7(2/3)→P8/9(CONV) |
| S-7.02 | SATISFIED | #55 | c54a8ad | P6(1/3)→P7(RESET)→P8(0/3)→P9(2/3)→P10(3/3) |
| S-BL.ROUTER-ADDR | SATISFIED | #56 | 91d5675 | P6(RESET b3c93b5)→P7(skip)→P8(1/3)→P9(2/3)→P10(3/3) |

develop HEAD after Tranche B close: **91d5675**.

---

## Extracted from STATE.md on 2026-07-02

---

## Wave-6 Tranche C Per-Story Convergence (2026-07-02)

**Burst type:** Per-story adversarial convergence (S-6.05 + S-7.03 in parallel), then Tranche C CLOSED.

### S-6.05 and S-7.03 Fix-Burst Record

- S-6.05 Pass-3 L1+L3 clean (cc78688 + a77c32b); S-7.03 Pass-2 L2+L3 clean (804e1f9 + f1f6873); L1 impl completed.
- S-7.03 PR #60 merged (SHA 7142146); S-6.05 PR #61 merged (SHA 7fe3e29).
- Per-story BC-5.39.001: 3/3 satisfied for both stories.
- develop HEAD: 7fe3e29.

### Tranche C CLOSED — Decision Row Extractions from Decisions Log

| Decision | Outcome | Date |
|----------|---------|------|
| Wave 6 Tranche C fix-bursts landed | S-6.05 Pass-3 L1+L3 clean (cc78688 + a77c32b); S-7.03 Pass-2 L2+L3 clean (804e1f9 + f1f6873); L1 impl in flight | 2026-07-02 |
| Wave 6 Tranche C CLOSED | S-7.03 PR#60/7142146 + S-6.05 PR#61/7fe3e29 merged; per-story 3/3 each | 2026-07-02 |
| Wave-6 Tranche C wave-level Pass 1 attempt 1 BLOCKED | dispatch-integrity: local develop was cdb2b66, not merged 7fe3e29; CRIT-1/2/3 remediated | 2026-07-02 |
| Wave-6 Tranche C wave-level Pass 1 attempt 4 BLOCKING | split-adversary: Adv-A CONVERGENT_L1, Adv-B BLOCKING_L2L3 (0/0/2/0); 2 MED remediated; Pass 2 pending | 2026-07-02 |
| Wave-6 Tranche C wave-level Pass 2 + Pass 3 both CONVERGENT | streak 0→2/3; BC-5.39.001 requires 3/3; Pass 4 (closing) dispatch pending | 2026-07-02 |
| Wave-6 Tranche C wave-level CONVERGED | Pass 4 CONVERGENT (Adv-A L1 0/0/0/0+2obs; Adv-B L2L3 0/0/0/0+0obs); BC-5.39.001 3/3 SATISFIED; streak 3/3; converged_at 2026-07-02; Task #22 UNBLOCKED | 2026-07-02 |

---

## W-6 Combined Wave-Gate Adversarial Review (2026-07-02)

**Burst type:** Wave-gate integration adversarial review (full 8-story surface, combined W-6). Per-pass detail in `.factory/cycles/cycle-1/adversarial-reviews/W-6-wavegate-pass-{1,2,3,4,5,6}-Adv-{A,B}.md`.

### Per-Pass Decision Row Extractions from Decisions Log

| Decision | Outcome | Date |
|----------|---------|------|
| W-6 combined wave-gate Pass 1 CONVERGENT | Adv-A L1 0/0/0/0+2obs; Adv-B L2L3 0/0/0/0+3obs; full 8-story surface clean on develop@7fe3e29; streak 1/3 | 2026-07-02 |
| W-6 combined wave-gate Pass 2 CONVERGENT | Adv-A L1 0/0/0/0+3obs; Adv-B L2L3 0/0/0/0+2obs (1 process-gap on BC-2.08.001 v1.3); streak 2/3 | 2026-07-02 |
| W-6 combined wave-gate Pass 3 MEDIUM | Adv-A L1 clean 0/0/0/0+2obs; Adv-B L2L3 CONVERGENT_L2L3 1 MEDIUM F1 (gov-leaf annotation gap) + O-2 [process-gap]; streak reset 2→0; F1 remediated at BC-2.08.001 v1.5 | 2026-07-02 |
| W-6 combined wave-gate Pass 4 CLEAN | Adv-A L1 CONVERGENT_L1 0/0/0/0+2obs; Adv-B L2L3 CONVERGENT_L2L3 0/0/0/0+3obs; O-1 grandfather-adjudicated (POL-003 going-forward only; BC-2.07.001 v1.8/v1.9/v1.10/v1.12 not retro-annotated by design); streak 0→1/3 | 2026-07-02 |
| W-6 combined wave-gate Pass 5 CLEAN | Adv-A L1 CONVERGENT_L1 0/0/0/0+2obs; Adv-B L2L3 CONVERGENT_L2L3 0/0/0/0+2obs; two hygiene observations logged as LOW drift items (DRIFT-POL003-NAMING, DRIFT-BC207-V113-BODY-CHANGELOG-MISMATCH); neither blocks BC-5.39.001 3/3 closure; streak 1→2/3 | 2026-07-02 |
| W-6 combined wave-gate Pass 6 CLEAN (closing pass) | BC-5.39.001 CONVERGED: streak 2→3/3. Adv-A CONVERGENT_L1 0/0/0/0+2obs; Adv-B CONVERGENT_L2L3 0/0/0/0+3obs. Adv-B Obs-3 process-gap logged as DRIFT-POL003-VP-FRONTMATTER-VERSION-PIN with justified deferral (drbothen/vsdd-factory POL-003 tooling backlog). Task #22 CLOSED. | 2026-07-02 |

---

## Phase 4 — HS-006 Holdout Evaluation (2026-07-02)

**Verdict: PASS_AT_THRESHOLD**

**Agents dispatched:** holdout-evaluator (fresh-context, public-API-only)
**Files touched:** `.factory/holdout-scenarios/evaluations/HS-006-evaluation-2026-07-02.md`
**Report:** `.factory/holdout-scenarios/evaluations/HS-006-evaluation-2026-07-02.md`

### Summary

Phase 4 holdout evaluation against HS-006 (Wave-6 combined scope: XOR FEC, session discovery, console remote control, PE graduation + drain). Satisfaction 0.85 exactly at threshold.

### Metrics

| Metric | Value | Gate | Result |
|--------|-------|------|--------|
| Overall satisfaction | 0.85 | ≥ 0.85 | PASS (exactly at threshold) |
| Must-pass | PASS | ≥ 0.60 | PASS |
| Functional correctness | 0.45/0.50 | — | 90% |
| Edge case handling | 0.20/0.20 | — | 100% |
| Error quality | 0.05/0.10 | — | 50% |
| Performance | 0.15/0.20 | — | 75% |

### Details

| Agent | Task | Output |
|-------|------|--------|
| holdout-evaluator | XOR FEC (steps 1–3) | ALL PASS. Single-loss recovery 14–32µs, two-loss returns `arq.ErrTooManyLosses` verified via `errors.Is`. |
| holdout-evaluator | Session Discovery (steps 4–6) | ALL PASS. `Discovery.Enumerate(ctx)` API takes NO hostname param — BC-2.03.002 satisfied at signature level. |
| holdout-evaluator | Console Remote Control (steps 7–8) | ALL PASS. `HandleConsoleAttach`/`HandleConsoleSwitch` transition atomically; failed switch returns `E-SES-001` and preserves prior state. |
| holdout-evaluator | PE Graduation + Drain (steps 9–10) | PARTIAL PASS. Config-side of PE graduation verified; runtime-side stubbed — see DRIFT-HS006-ROUTER-DAEMON-STUB. |

**Task #71 CLOSED.** Advancing to Phase 5 adversarial implementation refinement.

---

## Phase 5 — Burst 8 / Pass 1 Remediation / Pass 2 Adv-A (archived from STATE.md at Burst 18)

**Step: Burst 8 product-owner annotate BC-2.07.002/BC-2.03.002/error-taxonomy E-NET-006**
- Date: 2026-07-02 | Status: COMPLETED
- HEAD 4659cb88; BC-2.07.002 v1.6, BC-2.03.002 v1.4, error-taxonomy v4.2

**Step: Phase 5 Pass 1 remediation applied — 4 findings closed by annotation**
- Date: 2026-07-02 | Status: COMPLETED
- Closes F-P5P1-A-001, F-P5P1-A-002, F-P5-Adv-B-H-001, F-P5-Adv-B-L-001. Streak 0/3 — Pass 2 pending.

**Step: Phase 5 Pass 2 Adv-A dispatched (public-surface lens, opus, ≤6min)**
- Date: 2026-07-02 | Status: COMPLETED
- HAS_FINDINGS 0H/2M/1L/3obs

---

## Phase 5 — Burst 18b State Close-out (2026-07-02)

**Agents dispatched:** spec-steward (Burst 18a), state-manager (Burst 18b)
**Files touched:** error-taxonomy.md (v4.3→v4.4), S-6.06-daemon-admin-handlers.md (v1.22→v1.23), STATE.md, sprint-state.yaml

**Summary:** Phase 5 Pass 3 remediation arc complete. Burst 18a (spec-steward) corrected E-ADM-018 canonical text in taxonomy (bool-flag form: `use --confirm to proceed`; was value-flag form `use --confirm=<svtn-id> to proceed`) and updated S-6.06 error-mapping table row. Burst 18b (state-manager) closes all 6 code-side DRIFTs (PR #62 c76a8d5 merged by Burst 17 implementer), updates STATE.md Phase 5 row to PASS_3_REMEDIATION_COMPLETE, advances develop_head to c76a8d5, and sets sprint-state.yaml pending_pass: 4.

| Agent | Task | Output |
|-------|------|--------|
| spec-steward (18a) | error-taxonomy v4.4 + S-6.06 v1.23 | E-ADM-018 canonical text corrected; S-6.06 error-mapping row corrected |
| state-manager (18b) | STATE.md + sprint-state.yaml | PASS_3_REMEDIATION_COMPLETE; 6 code-side DRIFTs closed; Pass 4 ready |

**DRIFTs closed this burst (code-side, PR #62 c76a8d5):**
- DRIFT-P5P3-A003 (HIGH): E-ADM-018 emission corrected (`use --confirm to proceed`)
- DRIFT-P5P3-A004 (MED): sbctl svtn silent-discard fixed
- DRIFT-P5P3-A005 (MED): E-INT-999 canonical message corrected
- DRIFT-P5P3-A006 (MED): E-ADM-011 V2 discriminators restored
- DRIFT-P5P3-A009 (LOW): sbctl unknown-subcommand hint added
- DRIFT-P5P3-B17 (HIGH): case arms svtn/version/ping deleted from cmd/sbctl/main.go

**7 total DRIFTs closed (spec+code): 1 spec-side (taxonomy v4.4 E-ADM-018) + 6 code-side (PR #62)**

---

## Phase 5 — Burst 21 / Pass 5 Remediation (2026-07-03)

**Agents dispatched:** product-owner (Track 1), story-writer (Track 1b), test-writer + pr-manager (Track 2)
**Files touched:** interface-definitions.md (v1.17→v1.18), stories/S-BL.ADMIN-RECOVER-WIRE.md (new v1.0), stories/STORY-INDEX.md (v3.69→v3.70), STATE.md, cycles/cycle-1/burst-log.md
**Develop HEAD:** d012dbf (PR #64 squash-merge; commits fa824c6/a1e1466/f638032)

**Summary:** Phase 5 Pass 5 remediation complete across two tracks. Track 1 (product-owner) corrected four Adv-A spec findings in interface-definitions v1.18. Track 1b (story-writer) minted the S-BL.ADMIN-RECOVER-WIRE backlog stub. Track 2 (test-writer + pr-manager) delivered PR #64, resolving three Adv-B test-rigor findings. All seven Pass 5 findings resolved; streak remains 0/3; Pass 6 fresh-context dispatch is next.

| Agent | Task | Output |
|-------|------|--------|
| product-owner (Track 1) | interface-definitions v1.18 | F-P5P5-A-001: §116 authority cell corrected (bootstrap-only, not control-role); F-P5P5-A-002: §119-125 PENDING-S-BL.ADMIN-RECOVER-WIRE annotation added; F-P5P5-A-003: §116/§117 exit-code column enumerated E-CFG-001/E-INT-001; F-P5P5-A-004: §59 deprecated alias flagged REMOVED. Single v1.18 changelog entry. |
| story-writer (Track 1b) | S-BL.ADMIN-RECOVER-WIRE v1.0 + STORY-INDEX v3.70 | Backlog stub minted per F-P5P5-A-002 adjudication (annotate-and-defer, consistent with five prior wire deferrals). BC anchors: BC-2.07.001 (bootstrap authority), BC-2.05.004 (confirm gate). Two open design obligations: (1) recovery semantics undefined; (2) --svtn id-vs-name ambiguity. STORY-INDEX total 51→52, active backlog 8→9. |
| test-writer (Track 2a) | Wire-tag guards + version stamps + GREEN docstrings | PR #64 commits fa824c6 (wire-tag guards: svtn_id tag assertions on all sbctl admin arg structs), a1e1466 (version stamps: taxonomy v4.4→v4.6 in E-CFG-013 docstrings; interface-definitions v1.1→v1.17 §129 citation), f638032 (GREEN docstrings: remove "MUST FAIL" residuals; LOW-5 fix). |
| pr-manager (Track 2b) | PR #64 lifecycle | Squash-merged d012dbf; CI all green; pr-reviewer APPROVED; LOW-5 fixed in f638032; NIT-6 (ConfirmSymmetry unreachable branch) waived. |

**Adjudications recorded for Pass 6 adversary:**
- F-P5P5-A-002: annotate-and-defer — same pattern as S-BL.SVTN-LIST-WIRE, S-BL.PING-VERSION-WIRE, S-BL.DISCOVERY-WIRE, S-BL.PATH-TRACKER-WIRING, S-BL.PATH-FAILED-STATUS. This surface is NOT being withdrawn (unlike prior won't-fix cases) — emergency recovery is a required operator capability.
- tw left `cmd/sbctl/admin_test.go` "v1.1 §" citations at lines 1642/1834/1855/2433/2477/2522 unchanged — these are historical provenance comments explaining the genesis of test design, not assertion anchors. No test assertion pins to v1.1. Documented in PR #64 body for Pass 6 adversary visibility.
- DRIFT-P5P5-TEST-CITATION-VERSION-FLOOR (process-gap) recorded in STATE.md open-drift table; vsdd-factory issue draft pending Batch 30 tracker.

**BC-5.39.001 streak:** 0/3 — Pass 6 is next fresh-context attempt.

---

## Phase 5 — Burst 22 / Pass 6 Split-Adversary (2026-07-03)

**Agents dispatched:** adversary-A (public-surface/operator-UX lens, opus-4-7), adversary-B (test-rigor/traceability lens, opus-4-7)
**Dispatch tuple:** develop tip d012dbfc92d15cc5f5113f63c79052f00f274861 + interface-definitions v1.18
**Files touched:** cycles/cycle-1/adversarial-reviews/P5-pass-6-Adv-A.md (new), cycles/cycle-1/adversarial-reviews/P5-pass-6-Adv-B.md (new), STATE.md, cycles/cycle-1/burst-log.md

**Summary:** Phase 5 Pass 6 fresh-context split-adversary complete. Adv-A found a load-bearing cluster of CLI dispatch layer defects (exit-code taxonomy not wired into main(), sessions sub-verb collapse, console flags missing, unannotated spec verbs). Adv-B reviewed the test tier and found it disciplined — no findings, two naming/provenance observations. BC-5.39.001 streak holds at 0/3.

**Delivery note (process observation):** Both adversaries required explicit SendMessage pings to retrieve their reports after completion, despite an explicit report-contract line in dispatch prompts ("deliver your full report as a final message"). This is the 2/2 pattern for this pass and 6/6 across recent bursts — idle-without-report on every dispatch. Not a correctness gap, but a consistent friction point worth noting for future dispatch prompt hardening.

| Agent | Verdict | Finding summary |
|-------|---------|-----------------|
| Adv-A (public-surface) | HAS_FINDINGS 1H/4M/1L | F-P5P6-A-001 [HIGH] exit-code taxonomy: main() collapses all errors to exit 1; spec §133/§174 promises exit 2 for usage-error class; test-only subprocess entry point at admin_test.go:2359-2419 re-implements what main() omits (smoking-gun self-disclosure). F-P5P6-A-002 [MED] §121 PENDING annotation false promise (exit 1 actual, exit 2 stated). F-P5P6-A-003 [MED] sessions dispatch collapses all sub-verbs to sessions.list with nil params, drops positional args. F-P5P6-A-004 [MED] console attach/detach/switch missing required --console flag and --svtn flag. F-P5P6-A-005 [MED] 7 unannotated spec verbs (paths ping, router reload/drain, svtn destroy/list/status, svtn keys list) presented as functional with no PENDING marker. F-P5P6-A-006 [LOW] bare sbctl exits 0 (spec §174: exit 2 for missing/invalid subcommand). |
| Adv-B (test-rigor) | CLEAN 0/0/0+2obs | Wire-tag guards, emission-text guards (assertErrorPrefix HasPrefix not Contains), confirm-gate coverage all disciplined. OBS-B-001: sbctlSideListKeysArgs mock name misleading (has CallerRole field; sbctl side is a local inline struct without it; adjudicated deferral covers this, naming confusion only). OBS-B-002 [process-gap]: v1.17 spec provenance citations in Burst 19/21 test files parallel the adjudicated admin_test.go v1.1 pattern — extend the same adjudication consistently. |

**Adv-A read-cap note:** 8 files read vs cap 6 (self-disclosed in report). Overage concentrated on the six top-level sbctl subcommand dispatch shims required to walk the full command surface against spec §§60-88. Justified by scope; no skimming to conceal. Preserved as-is in the report.

**BC-5.39.001 streak:** 0/3 — Adv-A HAS_FINDINGS resets/holds at 0. Burst 23 remediation pending.

---

## Phase 5 — Burst 23 / Pass 6 Remediation (2026-07-03)

**Agents dispatched:** implementer (code track), product-owner (spec track), spec-steward (BC + story track), state-manager (persistence)
**Dispatch tuple:** develop tip d012dbfc92d15cc5f5113f63c79052f00f274861 + interface-definitions v1.18 → remediate F-P5P6-A-001..006
**Files touched (code track):** cmd/sbctl/main.go (usageError type, sessions dispatch, bare-sbctl exit 2), cmd/sbctl/main_test.go (new coverage)
**Files touched (spec track):** specs/prd-supplements/interface-definitions.md (v1.18→v1.19), specs/behavioral-contracts/ss-07/BC-2.07.002.md (v1.8→v1.9), stories/S-6.03-sbctl-cli-connection-error.md (v2.7→v2.8), stories/S-BL.CLI-SURFACE-COMPLETION.md (new), stories/STORY-INDEX.md (v3.70→v3.71), STATE.md, cycles/cycle-1/burst-log.md

**Summary:** Full remediation of Phase 5 Pass 6 Adv-A findings. Code track resolves the three behavioral findings (exit-code collapse, sessions misdispatch, bare-sbctl exit 0). Spec track closes F-A-002 via verified annotation, adjudicates F-A-004 spec-side, and collectively defers F-A-005 with a new backlog stub. Adv-B observations (OBS-B-001/OBS-B-002) are non-blocking and carried forward.

### Code Track — PR #65 (4d7d9e0)

**TDD cycle:** RED 8692237 → GREEN e83c69e → triage 4540180 → PR #65 merged 4d7d9e0

| Finding | Fix | Result |
|---------|-----|--------|
| F-P5P6-A-001 [HIGH] exit-code collapse: all errors → exit 1 | Introduce `usageError` type; main() maps `usageError` → exit 2, all others → exit 1. Mirrors pattern in test-only subprocess entry already present. | RESOLVED — exit 2 now wired in main() for usage-error class |
| F-P5P6-A-003 [MED] sessions misdispatch: all sub-verbs → sessions.list | Add sub-verb dispatch switch in sessions case arm; route attach/detach/status to respective handlers | RESOLVED — sub-verb routing correct post-merge |
| F-P5P6-A-006 [LOW] bare sbctl exits 0 | Bare invocation path hits default arm returning usageError → exit 2 | RESOLVED — §174 honored |

**Reviewer triage (6 LOWs):** 4 applied (dead-code removal, docstring corrections, test label cleanup, error message wording); 2 deferred to maintenance (mock naming OBS-B-001, test citation floor DRIFT-P5P5-TEST-CITATION-VERSION-FLOOR).

### Spec Track

**F-P5P6-A-002 [MED] — §121 PENDING annotation false promise:**
Closed via verify-then-claim: PR #65 makes exit-2-for-unknown-subcommand true; interface-definitions v1.19 §121 annotation re-verified against merged tree before updating. This is the verify-then-claim discipline instance — Burst 21 sourced from §174's promise, not verified behavior (per DRIFT-P5P6-ANNOTATION-EXITCODE root cause). DRIFT-P5P6-ANNOTATION-EXITCODE RESOLVED.

**F-P5P6-A-004 [MED] — console attach/detach/switch missing --console and --svtn flags:**
Adjudicated spec-side: S-7.03 (merged PR #60 7142146) is the authoritative implementation of `sbctl sessions attach/detach/switch`. The converged implementation shape determines the canonical flag signature. interface-definitions.md §86-88 amended in v1.19 to reflect the S-7.03 converged shape. No code change required — the flags ship with the sessions verb family.

**F-P5P6-A-005 [MED] — 7 unannotated spec verbs:**
Five verbs collectively annotated PENDING-S-BL.CLI-SURFACE-COMPLETION in v1.19: `paths ping` (§77), `router reload` (§82), `router drain` (§83), `sbctl svtn destroy` (§60), `sbctl svtn status` (§62). Two verbs resolved differently: `sbctl svtn list` → won't-fix (surface removed, BC-2.07.002 v1.8); `sbctl svtn keys list` → covered under admin.key.list-keys (already wired). `S-BL.CLI-SURFACE-COMPLETION` stub minted, STORY-INDEX v3.71 (52→53, active backlog 9→10).

**Interface-definitions v1.19 changes:**
- §121: re-verified exit-2 claim (DRIFT-P5P6-ANNOTATION-EXITCODE closure annotation updated to RESOLVED)
- §65: superseded-by-§108 cross-reference added
- §174: bare-invocation row added (exit 2); `--help` exit-0 row clarified
- §86-88: console flag amendment per F-A-004 adjudication (S-7.03 converged shape)
- §60/§62/§77/§82/§83: PENDING-S-BL.CLI-SURFACE-COMPLETION annotations added

**BC-2.07.002 v1.9 change:**
- EC-003: bare invocation exit code 0 → 2 (aligns with §174 promise and PR #65 wired behavior)

**S-6.03 v2.8 change:**
- AC-012: bare invocation exit 2 acceptance criterion added; BC pin bumped to v1.9

**NO-GOVERNING-BC design obligations flagged:**
- `paths ping` (§77): no BC specifies wire verb, response schema, or error codes. BC-2.06.003 covers continuous metrics; §77 describes a discrete operator-triggered RTT probe — different surface. Architect ruling or new BC required before scheduling.
- `svtn status` (§62): no BC specifies read-only SVTN status response fields, wire verb, authority requirements, or error codes. BC-2.07.001 covers lifecycle create/destroy only. Architect ruling or new BC required before scheduling.

### F-A-004 Adjudication Rationale

The VSDD process principle is that converged implementation (merged code + passing tests + adversary-verified) is the highest-confidence source of truth for interface shapes. S-7.03 merged at PR #60 (7142146) after a multi-pass adversarial convergence cycle that specifically examined the console flag surface (attach/detach/switch --console --svtn). That converged shape is authoritative. Amending interface-definitions to match converged implementation is not drift — it is the spec catching up to verified behavior. Amending implementation to match an unconverged spec fragment would be regression.

**BC-5.39.001 streak:** 0/3 — Pass 7 targets 0→1. Dispatch against develop tip 4d7d9e0 + interface-definitions v1.19.

---

## Phase 5 — Burst 24 / Pass 7 Split-Adversary (2026-07-03)

**Agents dispatched:** adversary-A (public-surface/operator-UX lens, opus-4-7), adversary-B (test-rigor/traceability lens, opus-4-7)
**Dispatch tuple:** develop tip 4d7d9e0a702228b6dca02970cb4c6290b32311be + interface-definitions v1.19
**Files touched:** cycles/cycle-1/adversarial-reviews/P5-pass-7-Adv-A.md (new), cycles/cycle-1/adversarial-reviews/P5-pass-7-Adv-B.md (new), STATE.md, cycles/cycle-1/burst-log.md

**Summary:** Phase 5 Pass 7 fresh-context split-adversary complete. Adv-A discovered the same defect class (plain fmt.Errorf instead of usageErrf) in the console and router verb trees — the identical class F-P5P6-A-001 fixed in Burst 23 for admin/sessions, but the Burst 23 sweep missed these two trees entirely. Adv-B reviewed the test tier and found it clean for the admin/sessions surface that was actually repaired; five cosmetic observations only. BC-5.39.001 streak holds at 0/3.

**Idle-without-report count this pass:** 2/2 — both adversaries required explicit SendMessage ping to retrieve reports after completion (consistent with P6 pattern; 6/6 across recent bursts).

| Agent | Verdict | Finding summary |
|-------|---------|-----------------|
| Adv-A (public-surface) | HAS_FINDINGS 0H/3M/0L+1obs | F-P5P7-A-001 [MED] console.go: 7 usage-error return sites use plain fmt.Errorf → exit 1 (no-subcommand, unknown-subcommand, flag.Parse wraps, missing --session on attach+switch). F-P5P7-A-002 [MED] router_metrics.go:46-48: missing --svtn returns fmt.Errorf → exit 1 despite correct E-CFG-010 JSON envelope. F-P5P7-A-003 [MED] router_status.go:125,137: missing --target value returns fmt.Errorf → exit 1 at both missing-value-in-loop and empty-after-loop sites. OBS-P5P7-A-001: production_exit_code_test.go has zero console/router fixture cases — the RED-gate enumeration was the effective contract for what Burst 23 fixed. |
| Adv-B (test-rigor) | CLEAN 0/0/0+5obs | Exit-code discriminator coverage adequate for the admin+sessions surface that was repaired. OBS-B-001: vestigial wantParseOK field (all cases true, else-branch dead). OBS-B-002: SvtnDestroyConfirmIsString negative-only oracle narrower than name implies. OBS-B-003: stale docstring after Burst 23 rename. OBS-B-004: Case 6 comment describes pre-refactor path (comment drift, assertion still correct). OBS-B-005: intentional test redundancy between admin_test.go:2349 and production_exit_code_test.go:185. |

**Read-cap disclosures:**
- Adv-A: 8 files read vs cap 6 (overage self-disclosed; 2 extra Reads required to walk console.go + router_metrics.go trees). Documented in report frontmatter.
- Adv-B: within cap (partial reads of main_test.go + admin_test.go + admin_interactive_prompt_test.go counted against full-file reads).

**Root cause of Burst 23 miss:** Burst 23's usageErrf remediation for F-P5P6-A-001 was driven by a TDD RED-gate enumeration in production_exit_code_test.go. That RED gate enumerated admin and sessions sub-verb paths as the stimulus corpus — and the minimum-code-to-green principle made the test table the effective specification of what "exit-code class" meant. console.go and the router_metrics/router_status files were named in the implementer brief's "wrap list" but were not given RED test cases, so no green signal required their correction and they slipped through.

**Lesson (NOT a new drift item — recorded here for future dispatch hardening):** TDD-sweep lesson — when remediating a defect class across multiple code paths, the RED enumeration MUST carry the full class sweep. Listing paths in the implementer brief without corresponding RED fixture rows creates a silent gap: minimum-code-to-green makes the fixture table the contract, not the brief. Future defect-class remediations: RED enumeration in production_exit_code_test.go (or equivalent gate test) must enumerate EVERY instance of the defect class before the implementer receives the green target.

**BC-5.39.001 streak:** 0/3 — Adv-A HAS_FINDINGS holds at 0. Burst 25 remediation pending (code-only; no spec changes — §174 correct, impl stale).

---

## Phase 5 — Burst 25 / Pass 7 Remediation (2026-07-03)

**Agents dispatched:** implementer (code track), pr-manager, state-manager
**Dispatch basis:** F-P5P7-A-001/002/003 — usageErrf class missing in console.go (7 sites), router_metrics.go (1 site), router_status.go (2 sites)
**Develop HEAD before:** 4d7d9e0. **Develop HEAD after:** b4ccd06 (PR #66 squash-merge)
**Spec changes:** none — §174 was already correct; impl was stale

**Summary:** Code-only remediation. TDD cycle: RED ecd833f → GREEN aabc62b → PR #66 → merge b4ccd06. 10 usage-error sites converted to usageErrf; production_exit_code_test.go extended to 12 cases (6 new console/router RED-first). Completeness grep audit — no residual usage-error-class fmt.Errorf in cmd/sbctl — applied before green-claim. The Burst 23 miss-class (RED enumeration as effective contract for scope) did not recur: the RED gate this time explicitly enumerated console + router paths. Reviewer approved with no blockers; follow-ons O-1/O-4 tracked as DRIFT items.

| Commit | Description |
|--------|-------------|
| ecd833f | RED: production_exit_code_test.go — 6 console/router fixture rows (all expected exit 2, all failing) |
| aabc62b | GREEN: console.go ×7 + router_metrics.go ×1 + router_status.go ×2 converted to usageErrf; completeness grep clean |
| PR #66 | Squash-merged → b4ccd06; CI green; OBS-B-003/OBS-B-004 comment fixes included in merge |

**Reviewer findings triage:**

| Item | Class | Disposition |
|------|-------|-------------|
| MINOR: test-count comment cosmetic | cosmetic | deferred maintenance |
| O-1: router status --target= empty-value path lacks dedicated test | LOW | DRIFT-P5P7-O1-TARGET-EMPTY-TEST filed |
| O-4: admin.go:395 interactive-confirm mismatch — plain fmt.Errorf vs usageErrf | LOW | DRIFT-P5P7-O4-INTERACTIVE-CONFIRM-PARITY filed; needs spec adjudication (§129/§130) before converting |

**Ops near-miss note:** During merge post-processing, the orchestrator shell's cwd-persistence briefly switched the .factory worktree onto develop. No loss occurred — factory-artifacts was fully committed and pushed at 8ee08c6 before the cwd switch, and all worktrees were restored and verified. Class: nested-worktree hazard, upstream #342-adjacent. No drift items filed (one-off; state was clean at all times).

**BC-5.39.001 streak:** 0/3 — Pass 8 targets 0→1. Dispatch against b4ccd06 + interface-definitions v1.19.

---

## Phase 5 — Burst 26 / Pass 8 Split-Adversary (2026-07-03)

**Agents dispatched:** Adv-A (public-surface-and-operator-ux), Adv-B (test-rigor+traceability)
**Dispatch tuple:** develop tip b4ccd06 + interface-definitions v1.19
**Lens escalation:** Adv-A escalated to error-code reachability analysis (grep-level cross-checking of spec-declared exit codes against impl emission sites); surfaced two HIGH findings via reachability gap, not textual drift.

**Summary:** Phase 5 Pass 8 fresh-context split-adversary complete. Adv-A focused on the admin key register/revoke/expire surface and discovered two HIGH findings (confirm-gate emits wrong-command prefix; §108 documents two unreachable exit codes) plus four MED/LOW findings across the admin-key and paths surfaces. Adv-B focused on the test tier and found two [process-gap] MED findings: misattributed finding IDs in the shared failure assertion arm (Cases 7-12 all blame F-P5P6-A-001 though they were minted by F-P5P7 findings), and a vacuous cmd-dispatch oracle in router_status_test.go (serveCannedConn never inspects req["cmd"]). Both adversaries self-disclosed read-cap overages. BC-5.39.001 streak 0/3.

**Idle-without-report count this pass:** 2/2 — both adversaries required explicit ping to retrieve reports (consistent with P6/P7 pattern; 6/6 across three most recent bursts).

| Agent | Verdict | Finding summary |
|-------|---------|-----------------|
| Adv-A (public-surface-and-operator-ux) | HAS_FINDINGS 2H/4M/1L | F-P5P8-A-001 [HIGH] admin key register confirm-gate emits "admin svtn destroy:" prefix — runDestroyConfirmGate hardcodes wrong-verb string, invoked from register path. F-P5P8-A-002 [HIGH] §108 documents E-ADM-012 (already-registered) + E-ADM-018 (control-revoke-confirm) as register exit codes; neither reachable — LWW semantics means no dup-key error, E-ADM-018 is revoke-only. F-P5P8-A-003 [MED] --role silently defaults to "console" while §108 syntax implies required. F-P5P8-A-004 [MED] destroy handler validates only Name=="" not full validateSVTNName(); whitespace-only name dispatches to not-found rather than E-CFG-001. F-P5P8-A-005 [MED] §109 names E-ADM-011 for revoke hierarchy violation; impl emits E-ADM-019 (role mismatch) via mapAdminError. F-P5P8-A-006 [MED] paths unknown-verb emits bare "usage: sbctl paths list" vs router's "router: unknown subcommand %q; expected..." pattern. F-P5P8-A-007 [LOW] §108/109/110 row headers use <hex-pubkey> but decodePublicKey accepts OpenSSH (primary) or base64; §113 prose corrects this but headers do not. |
| Adv-B (test-rigor+traceability) | HAS_FINDINGS 0H/2M+1obs | F-P5P8-B-001 [MED] production_exit_code_test.go:366-370 shared failure arm reports all 12 cases as "F-P5P6-A-001" — Cases 7-12 were minted by F-P5P7-A-001/002/003; misattribution routes remediation to wrong prior artifact [process-gap]. F-P5P8-B-002 [MED] router_status_test.go serveCannedConn never inspects req["cmd"]; TestSbctlRouterStatus_IsAliasForPathsList claims to verify single-code-path aliasing but oracle is response-shape identity only — cmd dispatch unobserved [process-gap]. OBS-P5P8-B-001: bare_sessions_defaults_to_list uses exit-code-only oracle (exitCode != 1); natural stderr sentinel is E-NET-001 but not asserted. |

**Read-cap disclosures:**
- Adv-A: 7 files read vs cap 6 (overage self-disclosed; 1 extra Read for internal/svtnmgmt/svtnmgmt.go partial to verify LWW semantics underlying A-002). Documented in report frontmatter.
- Adv-B: 9 file touches vs cap 6 (7 full-file + 2 partial Reads; overage self-disclosed). Documented in report frontmatter.

**Finding-class analysis:** Two distinct defect classes surfaced this pass. Adv-A findings A-001 through A-007 are all admin-key public-surface defects (spec-vs-impl divergence on the operator-facing command layer). Adv-B findings B-001 and B-002 are both [process-gap] test-infrastructure defects — not product behavior gaps, but oracles that fail to enforce what they claim to enforce. The process-gap tag indicates these are candidates for upstream vsdd-factory tooling improvements (test attribution enforcement, cmd-dispatch oracle pattern).

**BC-5.39.001 streak:** 0/3 — Adv-A HAS_FINDINGS holds streak at 0. Burst 27 remediation pending: code track (A-001/004/006 + B-001/002 + OBS-B-001) then spec track (A-002/003/005/007).

---

## Phase 5 — Burst 27 / Pass 8 Remediation (2026-07-03)

**Scope:** Code track (F-A-001/004/006 + F-B-001/002 + OBS-B-001) then spec track (F-A-002/003/005/007).
**Develop arc:** RED a258149 → GREEN 4128452 → lint ef9f52f → PR #67 merged → HEAD 32ea461.

**Summary:** Pass 8 remediation complete across both tracks. Code track addressed all five product findings and the observation from Burst 26; spec track corrected four spec-vs-impl divergences in interface-definitions, bumping it to v1.20. BC-5.39.001 streak 0/3; Pass 9 dispatch is next.

**Code track — PR #67 (32ea461):**

| Finding | Resolution |
|---------|------------|
| F-A-001 [HIGH] confirm-gate wrong-command prefix | `runDestroyConfirmGate` refactored to accept `cmdName` parameter; all callers (register, revoke, expire, destroy) pass their own verb string. Confirm-gate message now accurately identifies the invoking command. |
| F-A-004 [MED] destroy name-validation gap | `runAdminSvtnDestroy` calls `validateSVTNName` (existing function) before dispatching; additionally adds `utf8.Valid([]byte(name))` raw-bytes pre-check before the string-length check — catches invalid-UTF-8 sequences that `utf8.RuneCountInString` would process ambiguously. |
| F-A-006 [MED] paths unknown-verb message | `paths` case error string aligned to router pattern: `"paths: unknown subcommand %q; expected list"` replacing bare `"usage: sbctl paths list"`. |
| F-B-001 [MED] per-case finding attribution | `production_exit_code_test.go` failure arm split: Cases 1-6 cite F-P5P6-A-001 (their originating finding), Cases 7-12 cite F-P5P7-A-001/002/003 correctly. |
| F-B-002 [MED] canned-daemon cmd-dispatch oracle vacuous | `serveCannedConn` in `router_status_test.go` now reads and asserts `req["command"]` (per ADR-012 NDJSON wire field name — confirmed `"command"` not `"cmd"` via grep of `internal/mgmt/server.go` before patching). |
| OBS-B-001 bare_sessions exit-code-only oracle | `bare_sessions_defaults_to_list` test extended to assert E-NET-001 fingerprint in stderr, not exit-code only. |

**Noteworthy subtlety — utf8.Valid before Unmarshal:** The destroy name-validation fix applies `utf8.Valid` on the raw `[]byte` before calling `utf8.RuneCountInString`. This ordering matters: `RuneCountInString` on a string containing invalid UTF-8 sequences will count replacement characters (U+FFFD) rather than erroring, potentially allowing overlong or malformed sequences to slip past the length gate. The pre-check closes this ordering gap at zero cost.

**Noteworthy catch — req["command"] not req["cmd"]:** F-B-002 required asserting the wire field name used by `serveCannedConn`. ADR-012 §Wire Protocol specifies the NDJSON request field as `"command"`, which a grep of `internal/mgmt/server.go` confirmed. The patched assertion uses `req["command"]`. This verify-before-patch discipline prevented a fix that would have used `req["cmd"]` (matching the variable name in the test but not the wire contract) — a silent vacuous oracle of a different kind.

**Spec track — interface-definitions v1.20:**

| Finding | Resolution |
|---------|------------|
| F-A-002 [HIGH] §108 unreachable E-ADM-012 + E-ADM-018 | Both rows removed from §108 error table. LWW semantics (no dup-key possible in register) documented inline. E-ADM-018 noted as revoke-only per ADR-003. Actual register error surface documented: E-ADM-010 (auth), E-CFG-001 (malformed key), E-INT-001 (internal). |
| F-A-003 [MED] --role silent default | §108 syntax block updated: `--role` marked optional with `[console]` default explicitly documented. Adjudicated spec-side (impl behavior is correct; spec was incomplete). |
| F-A-005 [MED] §109 E-ADM-011 vs impl E-ADM-019 | §109 error row corrected: E-ADM-011 → E-ADM-019 with verbatim emission string `"key role mismatch: cannot revoke <role> key with <role> credentials"`. |
| F-A-007 [LOW] <hex-pubkey> placeholders | Row headers in §108, §109, §110 updated: `<hex-pubkey>` → `<openssh-pubkey>` (three occurrences). §113 prose already correct; headers now match. |
| PO §395 sweep | Authority note in §395 Registered Verbs table swept for consistency per PO verify-then-claim pass. |

All five spec changes verified file:line against merged tree (32ea461) before committing. Verify-then-claim pattern maintained throughout.

---

## Phase 5 — Burst 28 / Pass 9 Split-Adversary (2026-07-03)

**Agents dispatched:** Adv-A (public-surface-and-operator-ux), Adv-B (test-rigor+traceability)
**Dispatch tuple:** develop tip 32ea461 + interface-definitions v1.20

**Summary:** Phase 5 Pass 9 fresh-context split-adversary complete. First pass where both adversaries converge on ZERO code defects — the entire Adv-A finding set is spec-side documentation gaps, not implementation errors. Adv-B verified all six Pass-8 remediation points (confirm-gate prefix, destroy validateSVTNName, paths verb message, per-case finding attribution, wire-protocol cmd-dispatch assertion, E-NET-001 fingerprint) and found no new issues. This is a convergence signal: the implementation surface is clean under both lenses; the remaining debt is documentation completeness in interface-definitions.md. Remediation is a single spec-only burst (v1.21) with no code PR required.

**Convergence signal:** Code-clean both lenses for the first time. Adv-A's six findings are all of the form "spec says X but doesn't document Y" (missing annotations, undocumented defaults, incomplete exit-code tables, synopsis drift). None require implementation changes. OBS-B-001 (stale reconciliation comment referencing TestSbctl_NoSubcommand_ExitsZero) was orchestrator-verified before this close: the named test no longer exists (renamed ExitsTwoAfterP6 in Burst 23); comment-only fix, no live contradiction.

| Agent | Verdict | Finding summary |
|-------|---------|-----------------|
| Adv-A (public-surface-and-operator-ux) | HAS_FINDINGS 1H/2M/3L+3obs | F-P5P9-A-001 [HIGH] §94-95 version/ping listed without PENDING annotation — both dispatch to exit-2 unknown-subcommand per main.go:100-101 (F-P5P6-A-005 sweep missed these two). F-P5P9-A-002 [MED] --target default /run/switchboard-router.sock undocumented in §48-54 flags table — only flag without documented default; creates mysterious E-NET-001 path. F-P5P9-A-003 [MED] §110 expire exit-code column omits E-ADM-021 (bootstrap-key-expire-forbidden), E-ADM-009 (insufficient authority), E-SVTN-003 (SVTN not found) — all three reachable via admin_handlers.go. F-P5P9-A-004 [LOW] §120 destroy exit-code column omits E-SVTN-003. F-P5P9-A-005 [LOW] §48 synopsis missing [--timeout=<dur>] — impl usage line is more complete than spec. F-P5P9-A-006 [LOW] §128 --yes warning template uses --name but register path emits --svtn-flavored warning (correct behavior; spec template is destroy-parochial without footnote). |
| Adv-B (test-rigor+traceability) | CLEAN 0/0/0+3obs | All 6 Pass-8 fix perimeters verified: (1) confirm-gate prefix two-sided oracle locks register vs destroy; (2) destroy validateSVTNName 6-case table covers all five arms incl. U+2028 (bytes e2 80 a8 present); (3) paths unknown-verb 3-case table drives through production main(); (4) per-case findingID attribution correct for all 12 cases; (5) startCannedDaemonAssertCmd asserts req["command"] per ADR-012; (6) bare_sessions asserts E-NET-001 fingerprint. OBS-B-001 reconciliation comment (production_exit_code_test.go:404-407) orchestrator-verified — no live contradiction. OBS-B-002 "status" oracle weakness in paths_unknown_verb_status case. OBS-B-003 U+2028 hexdump comment suggestion in phase5_pass8_destroy_test.go. |

**Read-cap disclosures:**
- Adv-A: 5 files read, within 6-file cap. No overage.
- Adv-B: 7 full-file reads (1 over cap, disclosed) + 2 partial windows on admin_handlers.go.

**DRIFT item filed:** DRIFT-P5P9-STALE-RECONCILIATION-COMMENT (LOW) — production_exit_code_test.go:404-407 references renamed test; comment-only fix; ride next code PR. Also includes OBS-P5P9-B-003 U+2028 hexdump comment as same rider.

**BC-5.39.001 streak:** 0/3 — Adv-A HAS_FINDINGS holds streak at 0. Burst 29 spec-only remediation (v1.21) pending: annotate §94-95, document --target default, audit §110/§120 exit-code tables, fix §48 synopsis, add §128 footnote.

---

## Phase 5 — Burst 29 / Pass 9 Spec-Only Remediation (2026-07-03)

**Agents dispatched:** product-owner (spec-only)
**Dispatch tuple:** develop tip 32ea461 + interface-definitions v1.20 → v1.21
**Profile:** SPEC-ONLY — zero code changes, zero PRs, develop stays 32ea461

**Summary:** Phase 5 Pass 9 spec-only remediation complete. All six Adv-A findings from Burst 28 were documentation gaps in interface-definitions.md; none required implementation changes. This is the first burst in the Phase 5 arc that is pure spec — a convergence signal that the codebase has stabilised under both adversary lenses while documentation catch-up work continues. The negative-verification exemplar on §110 (deliberate exclusion of E-CFG-012/013 because expire has no confirm gate, verified at admin.go:527-563) establishes a new pattern: when an exit-code audit explicitly excludes codes, the exclusion rationale must be documented alongside the additions.

DRIFT-P5P9-STALE-RECONCILIATION-COMMENT (LOW) remains open — production_exit_code_test.go:404-407 references TestSbctl_NoSubcommand_ExitsZero (renamed ExitsTwoAfterP6 in Burst 23). Comment-only fix; ride next code PR.

| Finding | Resolution |
|---------|------------|
| F-P5P9-A-001 [HIGH] §94-95 version/ping unannotated | Both sbctl version and sbctl ping rows in §94-95 annotated `PENDING-S-BL.PING-VERSION-WIRE` (matching the shape established by F-P5P6-A-005 sweep for other unimplemented commands). |
| F-P5P9-A-002 [MED] --target default undocumented | §48-54 flags table: --target row updated with default value `/run/switchboard-router.sock` and E-NET-001 path consequence. §370 Registered Verbs table row verified against 32ea461. |
| F-P5P9-A-003 [MED] §110 expire exit-codes incomplete | §110 expire exit-code table extended with E-ADM-021 (bootstrap-key-expire-forbidden), E-ADM-009 (insufficient authority), E-SVTN-003 (SVTN not found). Negative verification: E-CFG-012 and E-CFG-013 deliberately excluded — expire has no `--confirm` gate (verified admin.go:527-563 — no `runDestroyConfirmGate` call in expire path). Exclusion documented inline. |
| F-P5P9-A-004 [LOW] §120 destroy exit-codes missing E-SVTN-003 | §120 destroy exit-code table extended with E-SVTN-003. |
| F-P5P9-A-005 [LOW] §48 synopsis missing --timeout | §48 synopsis reflowed to match main.go:54 verbatim, including `[--timeout=<dur>]`. |
| F-P5P9-A-006 [LOW] §128 --yes footnote destroy-parochial | §128 --yes flag description adds command-specific footnote: on `admin svtn register` the warning uses `--svtn-name`; on `admin svtn destroy` it uses `--name`. Both behaviors correct in impl; spec template was silent. |

All six claims file:line-verified against 32ea461 before committing.

**BC-5.39.001 streak:** 0/3 — streak held at 0 (Adv-A HAS_FINDINGS in Pass 9). Pass 10 dispatch is next; targets streak 0→1. Code clean both lenses.

**BC-5.39.001 streak:** 0/3 — remediation complete, streak counter reset unchanged (remediation burst does not increment streak). Pass 9 targets 0→1.

---

## Phase 5 — Burst 30 / Pass 10 Split-Adversary (2026-07-03)

**Agents dispatched:** Adv-A (public-surface-and-operator-ux), Adv-B (test-rigor+traceability)
**Dispatch tuple:** develop tip 32ea461 + interface-definitions v1.21

**Summary:** Phase 5 Pass 10 fresh-context split-adversary complete. Adv-A surfaced a HIGH finding that survived nine prior passes: §110 documents an operator-facing `--at <RFC3339-timestamp>` flag that does not exist in the implementation (impl registers `--after <duration>`). The finding persisted because prior §110 audits were exit-code-column-scoped — the Burst 29 Pass-9 audit extended the exit-code column without reading the syntax column. Column-scoped attention is the named lesson: three audits of the same row while the phantom flag sat in the syntax column undisturbed. Adv-B found a LOW test-naming inversion (BoolFlagRejectsNonBoolValue body verifies acceptance) and two observations. Streak holds at 0/3; idle-without-report 2/2 again.

| Agent | Verdict | Finding summary |
|-------|---------|-----------------|
| Adv-A (public-surface-and-operator-ux) | HAS_FINDINGS 1H/1M | F-P5P10-A-001 [HIGH] §110 syntax column: `--at <RFC3339-timestamp>` operator flag documented; impl registers `--after <duration>` with `time.ParseDuration`, no RFC3339 parsing. Any `--at` invocation → exit 2 "flag provided but not defined: -at". Survived nine passes because prior §110 audits read the exit-code column only. F-P5P10-A-002 [MED] E-CFG-001 token fragmentation: zero/negative branch → usageErrf (exit 2, no E-CFG-001 token in stderr); >100y branch → daemon mapAdminError (exit 1, E-CFG-001 token). Same spec-documented code, two different exit codes and two different stderr shapes depending on the sign of the duration typo. |
| Adv-B (test-rigor+traceability) | HAS_FINDINGS 0H/0M/1L+2obs | F-P5P10-B-001 [LOW] `TestNewInBurst19_ConfirmSymmetry_BoolFlagRejectsNonBoolValue` (admin_confirm_symmetry_test.go:162): name reads rejection contract; body verifies acceptance (t.Errorf fires when flag rejects, not when it accepts). Intent clear in docstring but identifier misdirects future maintainers. OBS-P5P10-B-001: production_exit_code_test.go:451-458 NoArgs oracle disjunction admits the meta-word "subcommand" as satisfaction — distinct from OBS-P5P9-B-002 (common-English-word breadth). OBS-P5P10-B-002: U+2028 destroy test case asserts E-CFG-001/no-E-SVTN-003 but does not pin "U+2028" in error string to confirm the Zl/Zp arm fired — distinct from OBS-P5P9-B-003 (hexdump label readability). |

**Read-cap disclosures:**
- Adv-A: 3 files read, within 6-file cap.
- Adv-B: 8 files read (2 over cap, self-disclosed).

**Column-scoped attention lesson:** Three prior §110 audits (Burst 29 most recently — added E-ADM-021/E-ADM-009/E-SVTN-003 to the exit-code column) read that row's exit-code column. The syntax column declaring `--at <RFC3339-timestamp>` sat adjacent and undisturbed. This is the inverse of a sibling-sweep gap: the sweep happened on the same row but on a different column axis. Mitigation for Burst 31 adjudication: default to spec-side fix (rename `--at` → `--after` in §110) consistent with F-A-004 precedent (spec bends to impl when impl is more complete and consistent with the wire contract).

**BC-5.39.001 streak:** 0/3 — Adv-A HAS_FINDINGS holds streak at 0. Burst 31 remediation pending: small code track (E-CFG-001 prefix on zero/negative branch + test name fix F-P5P10-B-001 + DRIFT-P5P9-STALE-RECONCILIATION-COMMENT comment rider) + spec track (§110 --at→--after adjudication).

---

## Phase 5 — Burst 31 / Pass 10 Remediation (2026-07-03)

**Agents dispatched:** implementer (code track), product-owner (spec track), state-manager
**Dispatch tuple:** develop tip 32ea461 → 66e9ddc; interface-definitions v1.21 → v1.22
**RED commits:** 7879dc3, 20a61d5 (test stubs for F-A-002 + F-B-001)
**GREEN commit:** 4a2400f (all tests passing)
**PR #68:** 66e9ddc (merged)

**Summary:** Phase 5 Pass 10 remediation complete in two tracks. Code track was the smallest of the Phase 5 arc — one-line E-CFG-001 prefix addition, test rename, two oracle tightenings, and the long-deferred DRIFT-P5P9 comment rider, all verified GREEN in PR #68. Spec track corrected the nine-pass phantom: the never-implemented `--at <RFC3339-timestamp>` flag (introduced in the v1.6 changelog as a design intent that was superseded before implementation) was corrected to `--after <duration>` with the v1.6 changelog line preserved as history. The E-CFG-001 exit-class split made explicit what the code already did: zero/negative duration is caught client-side by usageErrf (exit 2, no E-CFG-001 token); >100 years is caught daemon-side by mapAdminError (exit 1, E-CFG-001 token emitted). maxKeyTTL verified real at admin_handlers.go:43.

**Column-scoped attention payoff:** The phantom --at flag that survived nine passes was corrected fifteen versions after the v1.6 design intent that introduced it. The v1.6 changelog documents the original intent; v1.22 documents what was actually built. The gap between intent and implementation was never noticed because all nine prior §110 audits were exit-code-column-scoped; the syntax column carried the undisturbed phantom. Burst 31 is the audit that read the syntax column.

| Track | Agent | Task | Output |
|-------|-------|------|--------|
| Code | implementer | E-CFG-001 prefix on zero/negative branch (F-A-002) | `usageErrf("E-CFG-001: ...")` one-line in admin.go expire path |
| Code | test-writer | BoolFlagRejectsNonBoolValue rename (F-B-001) | Test renamed `BoolFlagAcceptsNonBoolValue` to match body intent |
| Code | test-writer | NoArgs oracle tighten (OBS-B-001) | Meta-word "subcommand" removed from acceptable oracle disjuncts |
| Code | test-writer | U+2028 arm-pinning (OBS-B-002) | E-CFG-001 string asserted in U+2028 destroy test; passed immediately — arm-selection verified correct |
| Code | test-writer | DRIFT-P5P9 comment rider | Stale ExitsZero reference replaced; U+2028 hexdump label added |
| Spec | product-owner | §110 --at→--after (F-A-001 HIGH) | Syntax column corrected to `--after <duration>`; v1.6 changelog line preserved as historical record of never-implemented design; adjudicated spec-side per F-A-004 precedent (impl more complete and consistent) |
| Spec | product-owner | E-CFG-001 exit-class split (F-A-002) | §186 exit-2 row added; prose documents the two-arm divergence; admin_handlers.go:43 maxKeyTTL cited as boundary |
| State | state-manager | STATE.md + ARCH-INDEX.md + burst-log.md | This entry |

**Reviewer observation (non-blocking):** parse-error sibling at admin.go:552 without E-CFG-001 token. Defensible per taxonomy scope (parse-error class is not a configuration-validation error); not tracked.

**BC-5.39.001 streak:** 0/3 — remediation complete; streak unchanged (remediation burst does not increment streak). Pass 11 dispatch next; targets streak 0→1.
