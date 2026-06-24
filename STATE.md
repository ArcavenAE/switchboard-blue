---
pipeline: IN_PROGRESS
phase: phase-3-tdd-implementation
phase_step: wave-1-in-progress (S-1.02 step-2 stubs complete; pause point for context clear)
phase_2_gate: APPROVED
phase_2_gate_date: 2026-06-24
phase_2_gate_disposition: approve-proceed-to-wave-1
phase_3_active_wave: 1
phase_3_completed_stories: [S-1.01]
phase_3_active_stories: [S-1.02]
phase_3_active_story_status: "S-1.02: in-progress, Step 2 of 9 complete, Step 3 (test-writer failing tests) pending"
phase_3_pause_point: "S-1.02 Step 2/9 (stubs) complete at 63f12f4 on feature/S-1.02-halfchannel-clock; resume by dispatching test-writer for Step 3"
s_1_01_merge_sha: 1c76160
s_1_01_pr_number: 1
phase_1_gate: APPROVED
phase_1_gate_date: 2026-06-24
phase_1_gate_disposition: approve-with-drift
phase_1_final_trajectory: "27 → 18 → 17 → 21 → 17 → 14 → 7 → 9"
phase_1_passes: 8
phase_1_refinement_rounds: 8
phase_1_commits: 18
phase_1_open_drift: 9
refinement_round_7_complete: true
refinement_round_8_complete: true
refinement_round_2_complete: true
refinement_round_3_complete: true
refinement_round_4_complete: true
refinement_round_5_complete: true
refinement_round_6_complete: true
structural_audit_complete: true
product: switchboard
mode: greenfield
anchor_strategy: reference-via-frontmatter
l2_complete: true
l2_artifact_count: 11
l2_subsystems: [session-networking, multipath-forwarding, session-discovery, session-access, admission-security, quality-observability, network-management, console-operations, deployment-operations]
l3_complete: true
l3_bc_count: 42
l3_cap_coverage: "30/30"
l3_cap_count: 30
l3_error_codes: 31
l3_bc_id_scheme: "BC-2.SS.NNN — S=2 stable L3-PRD prefix, SS=subsystem 01-09, NNN=sequence"
l3_subsystem_field_status: "patched — all 42 BCs have canonical subsystem + architecture_module fields"
l4_complete: true
l4_vp_count: 57
l4_bc_coverage: "42/42"
refinement_round_1_complete: true
arch_sections: 13
arch_adrs: 8
dtu_required: false
dtu_justification: "MVP single-LAN; no third-party SaaS deps. PE phase may need STUN/TURN DTU."
dtu_assessment: 2026-06-23
dtu_clones_built: n/a
dtu_services: []
feasibility_status: "all-feasible"
cicd_setup_complete: true
cicd_workflow_count: 6
cicd_p0_gaps: 3
cicd_p1_gaps: 2
cicd_p2_gaps: 5
internal_packages: 18
purity_distribution: {pure_core: 9, boundary: 5, effectful: 4}
go_verification_toolchain: ["go test", "go test -race", "go test -fuzz", "golangci-lint", "staticcheck", "go-mutesting"]
phase_2_complete: true
phase_2_epics: 8
phase_2_stories: 21
phase_2_holdouts: 6
phase_2_waves: 7
phase_2_total_points: 132
phase_2_bc_coverage: "42/42"
phase_2_drift_addressed: ["F-P8-001", "F-P8-002", "F-P8-003", "F-P8-006", "F-P8-007", "F-P8-008"]
phase_2_drift_deferred: ["F-P8-004", "F-P8-005", "F-P8-009"]
s_1_02_adversary_pass_01: 9_findings_not_converged
s_1_02_adversary_pass_02: 11_findings_not_converged
s_1_02_adversary_pass_03: 7_findings_not_converged
s_1_02_adversary_pass_04: 5_findings_not_converged
s_1_02_adversary_pass_05: 4_findings_not_converged
timestamp: 2026-06-24T00:00:00Z
last_update: 2026-06-24

---

# Switchboard Factory State

## Current phase

**Phase 3 — TDD Implementation** (entered 2026-06-24 after Phase 2 gate APPROVED).

Phase 2 gate: `approve-proceed-to-wave-1` (2026-06-24). Wave 1 active (S-1.01, S-1.02).

Phase 1 closed: 8 adversarial passes, 8 refinement rounds, 18 commits.
Trajectory: 27 → 18 → 17 → 21 → 17 → 14 → 7 → 9. Gate disposition: approve-with-drift.

## Pause/Resume Bookmark — 2026-06-24

**Pipeline paused mid per-story-delivery for S-1.02.**

### Where we are

- Wave 1: S-1.01 ✅ merged (PR #1, develop tip 1c76160)
- Wave 1: S-1.02 🔄 in-progress, Step 2 of 9 complete

### S-1.02 per-story-delivery progress

| Step | Agent | Status | Artifact |
|------|-------|--------|----------|
| 1. Worktree | devops-engineer | ✅ done | `.worktrees/S-1.02/` on `feature/S-1.02-halfchannel-clock` from `origin/develop` (1c76160) |
| 2. Stubs | stub-architect | ✅ done | commit `63f12f4` — internal/halfchannel/halfchannel.go (5 panic stubs: New, Tick, Enqueue, Seq, TickInterval) + Direction enum + ChannelFrame type + interval constants |
| 3. Failing tests | test-writer | ⏸ pending | Will write tests for 6 ACs (TestHalfChannelTick_OneFramePerCall etc.) |
| 4. Implementation | implementer | ⏸ pending | TDD per failing test |
| 4.5. Adversary convergence | adversary loop | ⏸ pending | BC-5.39.001 ≥3 clean passes |
| 5. Demos | demo-recorder | ⏸ pending | Per-AC evidence logs + Example godoc test |
| 6. Push | implementer/devops | ⏸ pending | (Stubs already pushed at this bookmark) |
| 7. PR lifecycle | pr-manager | ⏸ pending | 9-step process; target develop |
| 8. Worktree cleanup | devops-engineer | ⏸ pending | Remove `.worktrees/S-1.02/`, delete feature branch |
| 9. State update | state-manager | ⏸ pending | Mark S-1.02 completed in STORY-INDEX + sprint-state |

- Adversary pass 1 complete: 9 findings (2 critical, 3 high, 2 medium, 2 low); routing to PO + architect + test-writer + implementer.
- Adversary pass 2 complete: 11 findings (0 crit, 4 high, 4 med, 3 low); routing.
- Adversary pass 3 complete: 7 findings (0 crit, 1 high, 3 med, 2 low, 1 nitpick); convergence not yet reached; routing.
- Adversary pass 4 complete: 5 findings; convergence not yet reached; routing remaining 5 findings.
- Adversary pass 5 complete: 4 findings (1 high AC↔BC mis-anchor + 3 low test-quality nits); routing.

### Resume instructions

1. New session reads `.factory/STATE.md` (this section).
2. Verify worktree state: `git worktree list` shows `.worktrees/S-1.02` on `feature/S-1.02-halfchannel-clock`.
3. Verify branch state: `git -C .worktrees/S-1.02 log --oneline develop..HEAD` shows `63f12f4 feat(S-1.02): add internal/halfchannel stubs (Red Gate)`.
4. Dispatch `vsdd-factory:test-writer` for Step 3 (failing tests) — see `.factory/stories/S-1.02-halfchannel-clock.md` ACs.
5. After tests written, independently verify Red Gate (tests panic per stub), record `.factory/cycles/cycle-1/S-1.02/implementation/red-gate-log.md`, then dispatch `vsdd-factory:implementer` for Step 4.
6. After implementer: Step 4.5 adversary convergence loop until ≥3 clean passes.
7. Continue per `agents/orchestrator/per-story-delivery.md`.

### Lessons-learned guardrails to inject into S-1.02 dispatches (from S-1.01)

- DO NOT modify `.golangci.yml` or any project-wide lint/format config to silence findings.
- DO NOT use language-agnostic placeholders (`todo!()` etc.) — Go uses `panic("not implemented: S-1.02 <name>")`.
- For SA4006-prone test patterns (calling `len()` on a fixed-size array return), use `encoded[:]` slicing or byte-offset assertions, not bare `len(arr) != N` (compile-time tautology).
- If error codes are needed, USE ONLY codes that exist in `.factory/specs/prd-supplements/error-taxonomy.md` (don't fabricate). S-1.01 needed `E-PRT-001/002`.
- VP Source Contract titles in any new VP files must match `BC-INDEX.md` canonical titles verbatim (sentence-case).
- Adversary returns chat text not files (per #211) — orchestrator must Write findings to disk after each pass.

### Open items (independent of S-1.02)

- Wave 1 integration gate (after S-1.02 merges): consistency-validator + wave-1 holdout HS-001 + wave-adversary on the merged S-1.01+S-1.02 diff.
- Phase 3 Wave 4 will be first multi-story fan-out opportunity (4 stories: S-4.01, S-4.02, S-4.03, S-6.01 in parallel after Wave 2+3 complete).

## Phase 3 — TDD Implementation (active)

**Wave 1 active** — frame format + half-channel clock foundation.

| Story | Title | Status | Points | Module |
|---|---|---|---|---|
| S-1.01 | Frame codec | completed (PR #1 merged 1c76160) | 8 | internal/frame |
| S-1.02 | Half-channel clock | pending | 5 | internal/halfchannel |

Wave 1 dependencies: none (pure-core foundation; both stories independent).

Wave 1 holdout: `.factory/holdout-scenarios/wave-scenarios/wave-1.md` (HS-001).

### Phase 3 prerequisites (BEING ADDRESSED)

- P0-001: branch protection on `develop` — devops-engineer dispatched.
- P0-002: branch protection on `main` — devops-engineer dispatched.
- P0-003: required-signatures on protected branches — devops-engineer dispatched.

### Per-story delivery flow

Each story follows `/vsdd-factory:deliver-story`:
1. devops creates worktree at `.worktrees/<story-id>/`
2. stub-architect: compilable stubs (todo!() bodies — Red Gate)
3. test-writer: failing tests from BCs/ACs
4. implementer: TDD (red → green → refactor)
5. demo-recorder: per-AC visual demos
6. devops: push feature branch
7. pr-manager: full PR lifecycle (open → CI gate → AI review → fix loop → merge)
8. devops: worktree cleanup

## Phase 2 — Story Decomposition Output

| Artifact | Count |
|---|---|
| Epics | 8 (E-0 through E-7) |
| Stories | 21 (S-0.01 through S-7.04) |
| Waves | 7 (Wave 0 complete; Waves 1-6 pending) |
| Holdout scenarios | 6 (one per wave 1-6) |
| Total story points | 132 |
| BC coverage | 42/42 (100%) |

Wave distribution:
- Wave 0: 1 story, 1 pt (S-0.01 BMAD scaffolding — already complete)
- Wave 1: 2 stories, 13 pts (frame codec + half-channel clock)
- Wave 2: 3 stories, 18 pts (security foundation + session continuity)
- Wave 3: 3 stories, 21 pts (session access MVP)
- Wave 4: 5 stories, 29 pts (reliability layer + config) — max parallelism
- Wave 5: 4 stories, 21 pts (observability + CLI)
- Wave 6: 4 stories, 29 pts (PE-phase features)

Phase 1 DRIFT items addressed during decomposition:
- F-P8-001: BC-2.05.004 CLI surface → S-6.02 uses canonical `sbctl admin`
- F-P8-002: VP-030/VP-049 `sbctl status` → S-6.03 notes canonical `sbctl router status`
- F-P8-003: BC-2.08.001 module → S-7.03 targets `internal/session`
- F-P8-006: BC-2.05.007 list cmd → S-6.02 notes canonical `sbctl admin list-keys`
- F-P8-007: BC-2.02.005 SACK location → S-4.03 AC explicitly requires channel header
- F-P8-008: BC-2.02.007 frame_type → S-7.01 uses canonical `fec=0x05`

Remaining Phase 1 DRIFT (3 items deferred to Phase 3 architect/test-writer):
- F-P8-004 VP-026 transitivity invariant
- F-P8-005 VP-027 title/harness direction mismatch
- F-P8-009 architecture-feasibility-report deployment-operations CAP range text

## Source-of-truth inputs

Reference-via-frontmatter strategy. BMAD docs and KoS nodes remain
authoritative; `.factory/specs/` will derive from them via
`inputDocuments:` frontmatter.

- `_bmad-output/planning-artifacts/product-brief-switchboard-2026-03-31.md` — L1 brief
- `_bmad-output/planning-artifacts/prd.md` — L2/L3 source material (BMAD format)
- `_bmad-output/brainstorming/*` — 3 sessions (architecture, naming, session cache)
- `_kos/nodes/bedrock/` — 7 architectural bedrock nodes
- `_kos/nodes/frontier/` — open questions

## Discovery artifacts

- `.factory/planning/artifact-inventory.md`
- `.factory/planning/gap-analysis.md`
- `.factory/planning/routing-decision.md`

## Deferred decisions

- RESOLVED: **HMAC algorithm** — HMAC-SHA256 with 16-byte truncated tag, HKDF-SHA256 per-SVTN key derivation (ADR-001, ARCH-02/04)
- RESOLVED: **FEC group size** — N=4 default (20% overhead); tunable (ADR-002, ARCH-03). Phase 3 validates default empirically.
- RESOLVED: **Duplicate key registration** — last-write-wins (ADR-003, ARCH-04). Operator controls last write.
- RESOLVED: **Console/access key permissions** — control > console > access; only control nodes register keys (ADR-004, ARCH-04)
- RESOLVED: **Downstream ARQ failover** — resync from last ACK; in-flight frames during failover are lost (ADR-005, ARCH-03). Stateful transfer deferred to PE.
- **Tick interval range [5ms, 50ms]** — still empirical (ADR-008 keeps as tuning parameter). Validates in Phase 3.
- **Presence heartbeat 30s** — discovery is scope_phase PE, not MVP. Defer.
- **Marvel integration** — `_kos/nodes/frontier/question-marvel-integration.yaml` is acknowledged in `bounded-contexts.md` as out of scope. Now explicitly deferred — no MVP integration, no PE-phase integration. Re-evaluate post-MVP if marvel project publishes a stable interface. (resolves adversary F-024)
- ✓ **HMAC keying** → RE-RESOLVED with amended ADR-001: per-(node, svtn) HKDF derivation using node_admission_pubkey as IKM (was per-SVTN). Restores per-node trust boundary the BCs require.
- ✓ **Outer header layout** → AUTHORITATIVE (ARCH-02): 44 bytes exactly: version(1), frame_type(1), payload_len(2), svtn_id(16), src_addr(8), dst_addr(8), hmac_tag(8). Sequence lives in channel header only.
- ✓ **HMAC tag size** → 8 bytes (truncated from 32-byte HMAC-SHA256). 64-bit MAC sufficient for the rate-limited threat model; document for next adversary pass to verify.
- ✓ **Hash function** → SHA-256 stdlib (no Blake3 transitive dep).
- ✓ **Drop cache** → keyed on (checksum, arrival_interface_id) — fixes dup-and-race conflict.
- ✓ **Quality thresholds canonical** → 100/500ms RTT, 5%/20% loss, hysteresis 3.
- ✓ **OQ-003 permission hierarchy** → ADR-004 expanded: console cannot revoke control; control-to-control revocation requires `sbctl admin` human authorization.

## KoS frontier questions surfaced in Phase 1b

- Q: Does router-to-router PE phase need Noise XX mutual auth in addition to HMAC?
- Q: Should SACK bitmap window be configurable (64-bit default may be too narrow for PE high-latency links)?
- Q: Goroutine model for 1k concurrent sessions — per-session pair vs event-loop (NFR-004)?
- Q: Drop cache — TTL eviction in addition to LRU to prevent suppression after wraparound?
- Q: PE router-to-router Noise — share node admission keypair, or separate router identity?
- F-027 [process-gap] — 4 of 6 kos frontier files have empty `content:` blocks (`question-asymmetric-channels`, `question-encryption-model`, `question-marvel-integration`, `question-timeslice-framing`). Lint at kos-edge creation time should disallow empty content. Filed upstream.


## Non-blocking debt

- `.factory/.gitignore` not bootstrapped (drbothen/vsdd-factory#230 + this-session comment).

## Adversary cycle-1 metrics

- Pass 1 findings: 27 (5 critical, 11 high, 9 medium, 2 low; 3 process-gap tagged)
- Cycle 1 refinement: 5 critical + 11 high + 7 medium + 1 low addressed = 24 in-cycle; 2 process-gap deferred to upstream (F-025, F-027); 1 low deferred (covered by BA sweep).
- Convergence target: 3 consecutive zero-findings passes per FACTORY rules.
- Full findings: `.factory/cycles/cycle-1/adversarial-reviews/pass-01.md`
- Pass 2 findings: 18 (3 critical, 8 high, 6 medium, 1 low; 2 process-gap)
- Cycle 1 round-2 refinement: 17 in-cycle (3 critical + 8 high + 6 medium addressed); F-019 (1 low) by-design at Phase 1d, deferred to Phase 2 backfill rule.
- Full findings: `.factory/cycles/cycle-1/adversarial-reviews/pass-02.md`
- Pass 3 findings: 17 (4 critical, 9 high, 3 medium, 1 low; 1 process-gap)
- Cycle 1 round-3 refinement: all 17 in-cycle addressed (4 critical + 9 high + 3 medium + 1 low); F-P3-018 [process-gap] VP↔BC title-sync check filed upstream.
- Pass 4 findings: 21 (4 critical, 9 high, 6 medium, 2 low; 1 process-gap)
- Structural consistency audit (post-pass-4): 64 defects across 10 axes; 51 structural (closeable by 2 mechanical sweeps), 13 individual
- Cycle 1 round-4 refinement: 64 audit defects addressed mechanically + pass-4 findings F-P4-002, F-P4-008–013 covered by mechanical sweep (E-ADM-007→011, ARCH-11 counts, VP titles, --confirm flag, BC-2.01.005 module). F-P4-001 (PRD §7 BC-2.09.003→CAP-028) NOT yet addressed; F-P4-004 (best/any path quality) NOT yet addressed; F-P4-006 (VP-028/029 BC postcondition gap) NOT yet addressed; F-P4-014 (VP-001 uint32 vs u16) NOT yet addressed; F-P4-017 (module-criticality row count) NOT yet addressed.
- Cycle 1 round-5 refinement: all remaining pass-4 findings closed (F-P4-001, F-P4-004, F-P4-006, F-P4-014, F-P4-017, F-P4-018). Total pass-4 in-cycle resolution: 20 of 21 (F-P4-019 = stale CAP range in feasibility-report, deferred — Sweep 2 closed broader bug).
- Pass 5 findings: 17 (0 critical, 8 high, 7 medium, 2 low)
- Cycle 1 round-6 refinement: all 17 pass-5 findings closed across architect + PO refinement (split into 4 small bursts due to API connection drops).
- Pass 6 findings: 14 (0 critical, 7 high, 6 medium, 1 low)
- Cycle 1 round-7 refinement: all 14 pass-6 findings closed across 3 PO bursts + 1 architect burst. Priority drift (4 BCs P1→P0), BC contradiction fixes (BC-2.05.004, BC-2.06.001), error-taxonomy exit codes (E-ADM-011/012/013/014, E-CFG-006), interface-definitions --yes attribution + destructive sbctl svtn ops removal, module-criticality drop-cache placement, BC-2.09.003 DI-007 trace removal, 5 BCs missing VP rows added, BC-2.05.007 phantom sbctl debug removed, ARCH-11 module counts corrected.
- Pass 7 findings: 7 (0 critical, 2 high, 4 medium, 1 low)
- Cycle 8 refinement: all 7 pass-7 findings closed.
- Pass 8 findings: 9 (0 critical, 3 high, 5 medium, 1 low)
- Trajectory: 27 → 18 → 17 → 21 → 17 → 14 → 7 → 9 — GATE APPROVED with drift (approve-with-drift disposition; 9 items carried into Phase 2)
- Full findings: `.factory/cycles/cycle-1/adversarial-reviews/pass-03.md`

## Phase 1 Drift (carried into Phase 2)

- **HIGH** F-P8-001 — BC-2.05.004 trigger + test vectors still reference removed `sbctl svtn keys register|revoke|expire` (canonical is `sbctl admin`). Route: product-owner during Phase 2 story-writing for ss-05.
- **HIGH** F-P8-002 — VP-030 and VP-049 harness code uses non-existent `sbctl status` (canonical: `sbctl router status`). Route: architect or test-writer in Phase 3 when implementing these tests.
- **HIGH** F-P8-003 — BC-2.08.001 architecture_module pass-5 decision (internal/session) not propagated to ARCH-05:109, ARCH-11:67, VP-050:16. Route: architect Phase 2.
- **MEDIUM** F-P8-004 — VP-026 cites "transitivity" invariant that doesn't exist in BC-2.02.003. Route: architect during Phase 3 test-writing for BC-2.02.003.
- **MEDIUM** F-P8-005 — VP-027 title "degradation goes down" but harness tests recovery direction (red→green skip). Route: architect during Phase 3 test-writing.
- **MEDIUM** F-P8-006 — BC-2.05.007 test vector uses `sbctl keys list` (canonical: `sbctl svtn keys list` or `sbctl admin list-keys`). Route: product-owner Phase 2.
- **MEDIUM** F-P8-007 — BC-2.02.005 says SACK in upstream "payload" — ARCH-02 places it in channel header. Route: product-owner Phase 2.
- **MEDIUM** F-P8-008 — BC-2.02.007 references `FRAME_TYPE=PARITY`; canonical enum value is `fec=0x05`. Route: product-owner Phase 2.
- **LOW** F-P8-009 — architecture-feasibility-report:61 deployment-operations range still "(CAP-026–027)"; should be (CAP-026–028). Route: architect Phase 2.

## Phase 1 Carry-forward gaps (scope decisions deferred)

- **Content-type-aware loss recovery (interactive/streaming/bulk)** — BMAD PRD references but only interactive (BC-2.06.002) covered in BCs. Deferred to PE-phase BC if needed.
- **Audit log / session recording** — explicit out-of-scope decision: not part of switchboard MVP or PE phase. Re-evaluate post-MVP.
- **Multi-platform daemon behavior** — BC-2.09.x covers startup; Linux/macOS-specific behavior is implementation concern handled in Phase 3 implementer adaptive logic. No new BC needed.

## Phase 3 prerequisites (must resolve before TDD implementation)

- **P0-001** — Enable branch protection on `develop` (require ci.yml check, 1 approving review, dismiss-stale-reviews, restrict push)
- **P0-002** — Enable branch protection on `main` (same + restrict push to release tags)
- **P0-003** — Enable required_signatures on both protected branches (matches user's local gitconfig enforcement)
- Pass-8 P-codes (F-P8-001 through F-P8-009) should be resolved opportunistically by Phase 2 story-writer
