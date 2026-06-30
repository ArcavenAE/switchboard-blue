---
artifact_id: STORY-INDEX
document_type: story-index
level: ops
version: "3.11"
status: draft
producer: product-owner
timestamp: 2026-06-30T00:00:00
phase: 2
cycle: v1.0.0-greenfield
inputDocuments:
  - '.factory/stories/dependency-graph.md'
  - '.factory/cycles/cycle-1/wave-schedule.md'
---

# Story Index: Switchboard Cycle 1

## Summary

| Metric | Value |
|--------|-------|
| Total stories | 43 (34 master-table stories + 1 draft stub S-6.04 + 4 backlog S-BL.ARQ-TX/S-BL.OA/S-BL.LOOKUP/S-BL.NI + 2 hardening S-HRD.01/S-HRD.02 + 2 maintenance S-M.01/S-M.02) |
| Complete | 18 (S-0.01, S-1.01, S-1.02, S-2.01, S-2.02, S-1.03, S-3.04, S-3.01a, S-3.01b, S-3.02, S-3.03, S-W3.04, S-W3.05, S-4.01, S-4.02, S-4.03, S-4.04, S-6.01) |
| Pending | 9 |
| Draft/unscheduled | 9 (S-M.01, S-M.02, S-6.04, S-6.05, S-W5.03, S-6.07, S-HRD.01, S-HRD.02, S-W5.04) |
| E-phase | 28 (waves 0–5 + Wave 3 fix-now additions + Wave-5 net-new + S-6.07 + S-W5.04) |
| PE-phase | 4 (wave 6 PE stories) |
| Maintenance (draft/unscheduled) | 2 (S-M.01, S-M.02) |
| Total points (waves 0–6) | 192 |
| Total points (incl. S-M.01 + S-M.02) | 202 |
| Waves | 7 (Wave 0–6) + maintenance sweep (unscheduled) |
| Backlog | 4 (S-BL.OA, S-BL.ARQ-TX, S-BL.LOOKUP, S-BL.NI) |
| Draft stubs | 1 (S-6.04) |
| BC coverage | 45/45 (100%) — BC-2.07.004 added Wave-5 |
| VP coverage | 76/76 (100%) — VP-068..VP-076 added Wave-5 (VP-074 anchored to BC-2.06.001, VP-075/VP-076 anchored to BC-2.05.004) |

## Master Story Index

| Story ID | Title | Epic | Wave | BC Traces | Subsystems | Points | Priority | Scope | Status |
|---------|-------|------|------|-----------|-----------|--------|---------|-------|--------|
| S-0.01 | Port BMAD scaffolding as wave-0 baseline | E-0 | 0 | (none) | cmd/switchboard | 1 | P0 | E | complete |
| S-1.01 | Implement 44-byte outer header codec | E-1 | 1 | BC-2.01.004, BC-2.01.005, BC-2.01.006 | session-networking | 5 | P0 | E | completed |
| S-1.02 | Implement timeslice clock state machine | E-1 | 1 | BC-2.01.001, BC-2.01.002, BC-2.01.003 | session-networking | 8 | P0 | E | completed (PR #2, merge 9e9a98a) |
| S-1.03 | Session continuity via cryptographic re-authentication | E-1 | 2 | BC-2.01.007 | session-networking, admission-security | 5 | P0 | E | completed (PR #7, merge f35e836) |
| S-2.01 | Implement HMAC-SHA256 frame authentication | E-2 | 2 | BC-2.05.005 | admission-security | 5 | P0 | E | completed (PR #5, merge 3c4104e) |
| S-2.02 | Tier-1 admission and SVTN isolation | E-2 | 2 | BC-2.05.001, BC-2.05.002, BC-2.05.006, BC-2.05.007 | admission-security | 8 | P0 | E | completed (PR #6, merge a06b306) |
| S-3.01a | Tmux control mode integration | E-3 | 3 | BC-2.04.001 | session-access | 8 | P0 | E | completed (PR #11, merge 43208ab) |
| S-3.01b | PTY proxy fallback | E-3 | 3 | BC-2.04.002 | session-access | 5 | P0 | E | completed (PR #12, merge 56ec9c7) |
| S-3.02 | Console attach/detach and multi-console fan-out | E-3 | 3 | BC-2.04.003, BC-2.04.004, BC-2.04.006 | session-access | 8 | P0 | E | completed (PR #13, merge 1ff74f5) |
| S-3.03 | Tier-2 per-session authorization and read-only | E-3 | 3 | BC-2.04.005, BC-2.05.003 | session-access, admission-security | 8 | P0 | E | completed (PR #14, merge b68e498) |
| S-3.04 | Wire verifyFrameHMAC into RouteFrame (HMAC enforcement at router boundary) | E-2 | 3 | BC-2.05.008 | admission-security | 3 | P0 | E | completed (PR #9, merge d54bf1a) |
| S-W3.04 | Full daemon assembly — wire all Wave-3 subsystems in cmd/switchboard | E-3 | 3 | BC-2.04.001, BC-2.04.002, BC-2.04.003, BC-2.04.004, BC-2.04.005, BC-2.04.006, BC-2.04.007, BC-2.05.008 | session-access, admission-security | 8 | P0 | E | completed (PR #17, merge aeb442d) |
| S-W3.05 | Per-source HMAC failure counter and admission alert (BC-2.05.005 PC-3) | E-2 | 3 | BC-2.05.005, BC-2.05.008 | admission-security | 8 | P0 | E | completed (PR #16, merge fa6345e) |
| S-4.01 | Per-path RTT/loss tracking and dup-and-race | E-4 | 4 | BC-2.02.001, BC-2.02.002, BC-2.02.003, BC-2.02.009 | multipath-forwarding | 8 | P0 | E | completed (PR #24, merge e415d31) |
| S-4.02 | Upstream idempotent replay window | E-4 | 4 | BC-2.02.004 | multipath-forwarding | 5 | P0 | E | completed (PR #25, merge 95729c7) |
| S-4.03 | Downstream ARQ with ACK/SACK and TLPKTDROP | E-4 | 4 | BC-2.02.005, BC-2.02.006 | multipath-forwarding | 8 | P0 | E | completed (PR #26, merge 8d9744f) |
| S-4.04 | Split-horizon loop prevention + drop-cache router wiring | E-4 | 4 | BC-2.02.008, BC-2.02.009 (router wiring) | multipath-forwarding | 5 | P0 | E | completed (PR #27, merge 42c51e2) |
| S-5.01 | Green/yellow/red quality indicator with hysteresis | E-5 | 5 | BC-2.06.001, BC-2.06.002 | quality-observability | 5 | P1 | E | pending |
| S-5.02 | sbctl paths list / router metrics (canonical) + router status alias + p99 | E-5 | 5 | BC-2.06.003 | quality-observability, network-management | 5 | P1 | E | pending |
| S-5.03 | flag paths as degraded when EWMA RTT > 200ms | E-5 | 5 | BC-2.02.003 | multipath-forwarding | 2 | P1 | E | pending |
| S-6.01 | Config parsing and validation | E-6 | 4 | BC-2.09.003 | deployment-operations | 3 | P0 | E | completed (PR #28, merge abeba27) |
| S-6.02 | SVTN lifecycle and key management via sbctl admin | E-6 | 5 | BC-2.05.004, BC-2.07.001 | network-management, admission-security | 8 | P0 | E | pending |
| S-6.03 | sbctl client auth (Authenticate() fail-closed), flag parsing, JSON envelope, connection error | E-6 | 5 | BC-2.07.002, BC-2.07.003 | network-management | 5 | P0 | E | pending |
| S-W5.01 | internal/mgmt server, config E-CFG-008/009, cmd/switchboard wiring (all 4 daemon modes) | E-6 | 5 | BC-2.07.004, BC-2.09.003 | network-management, deployment-operations | 8 | P0 | E | draft |
| S-W5.02 | e2e management plane harness: sbctl auth + RPC across all 4 daemon types (VP-049) | E-6 | 5 | BC-2.07.002 | network-management | 5 | P0 | E | draft |
| S-6.06 | Daemon-side admin RPC handlers (admin.key.register / revoke / expire / list-keys) | E-6 | 5 | BC-2.05.004 | network-management, admission-security | 5 | P1 | E | draft |
| S-W5.03 | Release CI version gate — assert release binary version is semver not "dev" | E-9 | unscheduled | BC-2.07.004 | deployment-operations | 2 | P1 | E | draft |
| S-W5.04 | daemon-side paths.list / router.metrics / router.status RPC handlers and response types | E-5 | 5 | BC-2.06.003 | quality-observability, network-management | 5 | P1 | E | draft |
| S-6.05 | SVTN destroy lifecycle: SVTNManager.Destroy + sbctl admin svtn destroy | E-6 | 6 | BC-2.07.001 | network-management | 3 | P2 | E | draft |
| S-6.07 | Register admin.svtn.create handler + sbctl admin svtn create CLI subcommand | E-6 | 6 | BC-2.07.001 | network-management | 3 | P2 | E | draft |
| S-7.01 | XOR parity FEC for single-loss recovery | E-7 | 6 | BC-2.02.007 | multipath-forwarding | 8 | P1 | PE | pending |
| S-7.02 | SVTN-scoped multicast session discovery | E-7 | 6 | BC-2.03.001, BC-2.03.002, BC-2.03.003 | session-discovery | 8 | P1 | PE | pending |
| S-7.03 | Console remote control via sbctl | E-7 | 6 | BC-2.08.001, BC-2.06.001, BC-2.06.002 | console-operations, network-management | 5 | P1 | PE | pending |
| S-7.04 | E-to-PE router graduation and graceful drain | E-7 | 6 | BC-2.09.001, BC-2.09.002 | deployment-operations | 8 | P2 | PE | pending |

## Wave Summary

| Wave | Stories | Points | Theme |
|------|---------|--------|-------|
| 0 | S-0.01 | 1 | BMAD scaffolding (complete) |
| 1 | S-1.01, S-1.02 + refactor PR #3 | 13 | Frame codec + half-channel clock — **CLOSED 2026-06-24 (pass-with-clean-drift; rollback resolved 2026-06-24)** |
| 2 | S-1.03, S-2.01, S-2.02 | 18 | Security foundation + session continuity — **COMPLETE 2026-06-25 (3/3 merged; integration gate next)** |
| 3 | S-3.01a, S-3.01b, S-3.02, S-3.03, S-3.04, **S-W3.04**, **S-W3.05** | 48 | Session access MVP + HMAC wire-up + Wave 3 fix-now blockers — all 7 stories MERGED |
| 4 | S-4.01, S-4.02, S-4.03, S-4.04, S-6.01 | 29 | Reliability layer + config — **CLOSED 2026-06-28 (all 5 merged: PR #24–#28)** |
| 5 | S-5.01, S-5.02, S-5.03, S-6.02, S-6.03, S-W5.01, S-W5.02, S-6.06, S-W5.04 | 48 | Observability + CLI + Management Plane (S-W5.04 5pts added Pass-4 Ruling 1 — daemon-side paths.list/metrics handlers) |
| 6 | S-6.05, S-6.07, S-7.01, S-7.02, S-7.03, S-7.04 | 35 | SVTN lifecycle (create+destroy) + PE-phase features |
| **Total** | **32** (wave stories) | **187** | (+ S-M.01 + S-M.02 maintenance, 10 pts, unscheduled — grand total 34 stories / 197 pts) |

> Note: Wave 2 includes S-1.03 (depends on S-1.01 + S-2.02). Wave 3 includes S-3.04 (HMAC wire-up into RouteFrame, E-2 epic, P0) and the split of original S-3.01 into S-3.01a (tmux control mode, 8pts) + S-3.01b (PTY fallback, 5pts); S-3.03 repointed 5→8pts. Wave 3 also included two FIX-NOW gate blockers: S-W3.04 (daemon assembly, 8pts, E-3, F-1; merged PR #17 aeb442d) and S-W3.05 (HMAC failure counter, 8pts, E-2, F-2; repointed 5→8 per PO adjudication; merged PR #16 fa6345e). Wave 3 total: 7 stories, 48 pts, all MERGED. Wave 4 total: 5 stories, 29 pts, all MERGED (S-4.01 PR #24 e415d31, S-4.02 PR #25 95729c7, S-4.03 PR #26 8d9744f, S-4.04 PR #27 42c51e2, S-6.01 PR #28 abeba27; closed 2026-06-28). Wave 5 total: 9 stories, 48 pts (S-5.01: 5pts, S-5.02: 5pts, S-5.03: 2pts, S-6.02: 8pts, S-6.03: 5pts [re-scoped v2.0], S-W5.01: 8pts [net-new], S-W5.02: 5pts [net-new], S-6.06: 5pts [CR-W5-SCOPE-SPLIT adversary Pass 1], S-W5.04: 5pts [Pass-4 Ruling 1 — daemon-side metrics handlers]). Wave 6 total: 6 stories, 35 pts (S-6.05: 3pts [deferred from S-6.02, CR-009 ruling]; S-6.07: 3pts [net-new, HUMAN-APPROVED PATH B — admin.svtn.create handler + CLI]; S-7.01: 8pts, S-7.02: 8pts, S-7.03: 5pts, S-7.04: 8pts). Total points including Wave 0: 187. Per-wave counts above use story points from individual story files.

**Wave-5 Serialization Constraints:**
- S-6.03 (creates `cmd/sbctl` scaffold — `main.go`, `client.go`) must merge **before** S-6.02 (adds `cmd/sbctl/admin.go`) and before S-5.02 (adds paths_list.go, router_metrics.go, router_status.go) — same file registration in `cmd/sbctl/main.go`.
- **S-6.02 and S-5.02 MUST NOT run in parallel** — both edit `cmd/sbctl/main.go` command registration. Serialize: S-6.03 → S-6.02 → S-5.02 (or S-6.03 → S-5.02, then S-6.02; either order after S-6.03, but not concurrent).
- S-5.03 (internal/paths only) depends only on S-4.01; **must merge before S-5.01** (S-5.01 depends_on now includes S-5.03 — F-005 fix; S-5.01 reads Snapshot().Degraded which S-5.03 adds).
- S-5.01 (internal/metrics only — no cmd/sbctl edits) depends on S-4.01, S-4.03, and S-5.03; can start once all three are merged.
- **S-W5.01** (internal/mgmt + internal/config + cmd/switchboard) edits **no** cmd/sbctl files — can run **in parallel with S-6.03, S-6.02, S-5.02** on separate branches. No cmd/sbctl conflict.
- **S-6.06** (daemon admin handler wiring) depends on S-6.02; can start once S-6.02 merges. Owns the `TODO(CR-002)` at `mgmt_wire.go:259` — minted per CR-W5-SCOPE-SPLIT adversary Pass 1 ruling.
- **S-W5.02** (e2e harness) depends on **S-6.03, S-W5.01, and S-6.06**; must be the last Wave-5 management-plane story. Gate story for the management plane.
- **S-W5.04** (daemon-side paths.list / metrics handlers) depends on **S-5.02 and S-W5.01**; can start once both are merged. Owns VP-047 integration test. No cmd/sbctl conflict — writes only to `internal/metrics` and `internal/mgmt`.

## BC Coverage Check

All 45 BCs covered (44 prior + BC-2.07.004 minted Wave-5 management plane). BC-2.07.004 is covered by S-W5.01 (server-side auth handshake, PC-1 through PC-10, VP-064/VP-065/VP-066). S-W5.02 provides additional VP-049 e2e coverage for BC-2.07.002 (client+server joint verification across all four daemon types). BC-2.09.003 gains two new postconditions (PC-10, PC-11) covered by S-W5.01 (E-CFG-008, E-CFG-009). BC-2.06.003 daemon-side PC-1/PC-2/PC-3 serialization and VP-047 integration test are owned by S-W5.04 (Pass-4 Ruling 1 deferral from S-5.02). See dependency-graph.md BC-to-Stories matrix for full traceability.

## Backlog / Deferred Stories

Stories created as concrete drift-item targets BEFORE they're scheduled into a wave.
Backlog stubs have minimal frontmatter and no ACs yet. When a wave-N planning cycle
picks one up, story-writer fleshes it out into a normal wave-N story (move out of this
section, add full ACs/tasks/files/architecture).

Backlog convention introduced 2026-06-24 per drbothen/vsdd-factory#260 rollback —
addresses the "deferred to TBD story" anti-pattern.

**Backlog: 4** (no ACs; unscheduled; awaiting wave-planning promotion)

| Story ID | Title | Status | Drift items consumed | Earliest wave |
|----------|-------|--------|----------------------|---------------|
| S-BL.ARQ-TX | wire ARQ retransmit-SEND path into router/multipath dispatch (BC-2.02.005 PC-3) | backlog | S403-H1-DEFER (Wave 4 audit); depends S-4.03 | Wave 5+ |
| S-BL.OA | outer-assembler — compose ChannelFrame + OuterHeader into wire frames | backlog | wave-adv F-001 (spec closed) / F-003 / F-004 | Wave 3+ |
| S-BL.LOOKUP | Migrate `AdmittedKeySet.Lookup`/`LookupByPubkey` to `(AdmittedKey, bool)` value-return per go.md rule 12 | backlog | DRIFT-F005-LOOKUP-CONVENTION (S-6.02 adversary F-005); depends S-6.02 merge | Wave 6+ |
| S-BL.NI | network-ingress: implement network-ingress listener (bind/accept inbound network frames, feed to RouteFrame). `routing.WithFailureCounter(fc)` alongside `routing.WithLogger(rl)` is ALREADY WIRED in `buildRouter` (C-1 RESOLVED, PR #20, ARCH-08 v2.3 §6.5.1). No counter-wiring obligation remains for this story. Remaining obligation: wire a live-data-path ingress listener so real frames from the network traverse `RouteFrame`; include an integration test asserting E-ADM-017 fires through that live data path (frames triggering RouteFrame → FailureCounter → alert), not merely from constructed-but-idle router. **Also owns cfg.ListenAddr application** — must wire `cfg.ListenAddr` to `net.Listen`/`.Accept` at this story's implementation time (BC-2.09.003 PC-9 DEFERRED-APPLICATION; S-6.01 v1.4 deferred listen_addr binding depends on this story). | draft | C-1-W3P1-defer (network-ingress listener; FailureCounter wiring COMPLETED PR #20; ARCH-08 §6.5.1 v2.3 TRACKED-DEFER; BC-2.05.005 PC-3, S-W3.05 AC-009); BC-2.09.003 PC-9 listen_addr deferral (S-6.01 v1.4 SP-004) | Wave 4+ |

**Draft stubs: 1** (has some structure but no full ACs; will be promoted at wave-N planning)

| Story ID | Title | Status | Drift items consumed | Earliest wave |
|----------|-------|--------|----------------------|---------------|
| S-6.04 | SIGHUP config reload with fail-closed safety | draft | S601-DRAFT-STORY (Wave 4 audit) | Wave 6+ |

## Hardening Stories

Stories for security and resilience hardening deferred from wave convergence rounds.
Hardening IDs use the scheme `S-HRD.NN` (introduced 2026-06-29 per ARCH-12 Round-5
DEFER-WITH-FOLLOWUP rulings). Each row records the deferred finding and target release.

| Story ID | Title | Status | Deferred from | Severity | Target release |
|----------|-------|--------|---------------|----------|----------------|
| S-HRD.01 | Add `conn.SetWriteDeadline` to client write paths in `cmd/sbctl/client.go` (CWE-400 defense-in-depth) | draft | S-6.03 Wave-5 convergence Round-5 Finding 2 (ARCH-12 v1.5 DEFER-WITH-FOLLOWUP); client write deadline deferred because server-side close bounds practical risk; Rulings V/Y scoped client deadlines to read+dial only | MEDIUM | next post-Wave-5 hardening pass |
| S-HRD.02 | daemon logging infrastructure + security-event emission (BC-2.07.004 PC-3 / EC-004 deferred slog seam on mgmt.Server) | draft | S-W5.01 AC-003 deferral — Architect Ruling 1 (S-W5.01 mgmt-server convergence); BC-2.07.004 PC-3/EC-004 "logs a security event" deferred from S-W5.01 because daemon has no structured logging infra; connection control (AUTH_FAIL + E-ADM-010 + close) is satisfied and tested via VP-065 | MEDIUM | next post-Wave-5 hardening pass |

## Maintenance Stories

Stories for DX/tooling/infrastructure work that are NOT part of feature waves 1–7.
Maintenance IDs use the scheme `S-M.NN` (introduced 2026-06-27). No BC anchor required.
Execute in a post-Wave-7 maintenance sweep or standalone orchestrator dispatch.

| Story ID | Title | Epic | Wave | BC Traces | Points | Priority | Status |
|----------|-------|------|------|-----------|--------|----------|--------|
| S-M.01 | Migrate toolchain provisioning from Homebrew to mise | E-MAINT | unscheduled | (none — DX/tooling) | 5 | P2 | draft |
| S-M.02 | Formalize Apple code-signing and notarization of release binaries (toggle-gated) | E-MAINT | unscheduled | (none — release infra) | 5 | P2 | draft |

Epic E-MAINT covers maintenance/DX/self-improvement stories. No BC anchor applies to tooling stories.
Drift items MISE-DX-001 and MISE-DOC-002 are consumed by S-M.01.
Drift item SIGN-DX-001 is consumed by S-M.02. S-M.02 is milestone-gated — SIGNING_ENABLED stays OFF until functional-product milestone.

## Files

All story files are in `.factory/stories/S-N.MM-*.md`. Maintenance story files use `.factory/stories/S-M.NN-*.md`. Backlog stubs use `.factory/stories/S-BL.*-*.md`. Epic files are in `.factory/stories/epics/E-N-*.md`.

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 3.11 | 2026-06-30 | Pass-25 lens-3 fix: S-6.06 bumped v1.20 → v1.21 — stale VP-076 v1.1 cite in EC-008 body (line 204) bumped to v1.4; stale ARCH-04 v1.10 cite in Architecture Compliance Rules HOLD-001 row (line 263) bumped to v1.13. Closes F-P25L3-001 + lens-1 Obs-2 (7th-recurrence sibling-sweep gap closeout). |
| 3.10 | 2026-06-30 | Pass-23 lens-3 fix: S-6.06 bumped v1.19 → v1.20 — stale BC-2.05.004 EC-007 v1.10 cites in Error Code Map (line ~180) and Task 12 (line ~245) bumped to v1.12; closes recurring sibling-sweep gap from Pass-22 (4-pass recurrence). Closes F-P23L3-001 + F-P23L3-002. |
| 3.9 | 2026-06-30 | Pass-22 sibling-fix propagation (F-P22L3-001, 4th-iteration narrowing sweep): S-6.06 bumped v1.18 → v1.19 — VP-076 body table row "unconditionally" narrowed to "for any well-formed request" with E-CFG-001 layering note; mirrors BC-2.05.004 EC-007 v1.12 + VP-076 v1.3. |
| 3.8 | 2026-06-30 | Pass-21 sibling-fix propagation: S-6.06 bumped v1.17 → v1.18 — EC-008 narrowed from "unconditionally" to "for any well-formed request" per BC-2.05.004 EC-007 v1.12 (Pass-20 Option-B) and VP-076 v1.1; added E-CFG-001 input-validation layering note (F-P21L3-001, F-P21L2-001 HIGH dup-confirmed). v1.17 changelog row-attribution correction folded as parenthetical (fix was in ErrInvalidDuration row, not E-ADM-021 row). |
| 3.7 | 2026-06-30 | Pass-19 lens-2 fix: S-6.06 bumped v1.16 → v1.17 — correct Error Code Map ErrInvalidDuration row line citation 275-280 → 279-284 (line drift from intervening edits; F-P19L2-002). |
| 3.6 | 2026-06-30 | F-P18L1-001 (MED) + lens-3 LOW version-drift fix: S-6.06 bumped to v1.16 — EC-008 + Error Code Map extended to cover bootstrap-expire-forbidden (E-ADM-021); VP-076 minted; VP coverage updated 75/75 → 76/76. Frontmatter version drift corrected (was "3.4", body showed v3.5; now "3.6" aligned with body changelog). |
| 3.5 | 2026-06-30 | S-6.06 bumped to v1.15 — Error Code Map ErrBootstrapKeyRevokeForbidden canonical message aligned to error-taxonomy.md v3.7 unconditional phrasing (refs F-P17L2-001, F-P17L2-002, lens-2 pass-17). |
| 3.4 | 2026-06-30 | S-6.06 bumped to v1.14 — EC-008 wording aligned to BC-2.05.004 v1.9 unconditional phrasing; admin_handlers.go ttl-guard line citation corrected (refs F-P15L1-002, F-P15L2-001). |
| 3.3 | 2026-06-30 | F-T4L3-001: VP coverage updated 67/67 → 75/75 (100%) — VP-068..VP-075 added Wave-5; VP-074 anchored to BC-2.06.001, VP-075 anchored to BC-2.05.004. |
| 3.2 | 2026-06-30 | Pass-5 lens-3 F-P5-T-001: S-5.02 BC column corrected — drop BC-2.06.001 (no AC traces to it; AC-007 dropped v1.5; PC-5 deferred to S-7.03). BC column now `BC-2.06.003` only. |
| 3.1 | 2026-06-30 | S-5.02 Pass-4 Ruling 1: mint S-W5.04 (5pts, P1, Wave 5 capacity permitting/else 6, draft) — daemon-side paths.list/router.metrics/router.status RPC handlers and VP-047 integration test deferred from S-5.02. Total stories 42→43; Wave-5 8→9 stories, 43→48 pts; Total points (waves 0–6) 187→192, (incl. maintenance) 197→202. Draft/unscheduled 8→9. E-phase 27→28. Add S-W5.04 serialization note and BC coverage note. BC-2.06.003 v1.5→v1.6: F-LO1 PC-1 wording aligned to ARCH-03 v1.6 ("fixed-bucket histogram, never reset" replaces "rolling sample buffer"). |
| 3.0 | 2026-06-30 | Pass-2 lens-3 F-T3-002 propagation: S-6.06 BC column corrected — BC-2.07.001 anchor dropped per S-6.06 v1.3 PO ruling (S-6.06 owns admin.key.* only; BC-2.07.001 Inv-3 is scoped to admin.svtn.* operations). BC column now `BC-2.05.004` only. |
| 2.9 | 2026-06-30 | S-5.02 v1.4→v1.5: propagate Pass-3 PO rulings — drop AC-007 (tautological; sbctl sessions.list quality surfacing deferred to S-7.03 per Ruling 1); add AC-008 (pending-quality sentinel, BC-2.06.003 v1.5 PC-3 + EC-006); BC version pins v1.4→v1.5; §Scope Boundary added. |
| 2.8 | 2026-06-30 | S-6.06 lens-3 F-006 close: Draft/unscheduled rollup corrected — removed S-W5.01 (merged per feat(S-W5.01) commit), S-W5.02 (Wave-5 in-progress pending, not unscheduled), and S-6.06 (Wave-5 in-progress, not unscheduled). Count: 11→8. Remaining draft/unscheduled: S-M.01, S-M.02, S-6.04, S-6.05, S-W5.03, S-6.07, S-HRD.01, S-HRD.02. |
| 2.7 | 2026-06-29 | Pass-2 adversarial fix-burst: F-P2-002 — S-5.02 BC Traces add BC-2.06.001; F-P2-003 — S-7.03 BC Traces add BC-2.06.001 + BC-2.06.002; F-020 — total-stories arithmetic fixed (39→42, enumeration corrected: 33 master + 4 backlog + 1 draft-stub + 2 hardening + 2 maintenance); F-027 — backlog section split into "Backlog: 4 (S-BL.*)" and "Draft stubs: 1 (S-6.04)". Closes F-P2-002, F-P2-003, F-020, F-027. |
| 2.6 | 2026-06-29 | HUMAN-APPROVED PATH B (product-owner + human): add S-6.07 (3pts, P2, Wave 6, draft) — `admin.svtn.create` handler + `sbctl admin svtn create` CLI, closes S-5.01/S-6.02 Pass-1 adversarial F-003/F-010 (BC-2.07.001 PC-1 had no CLI/RPC reachability). Add S-BL.LOOKUP (1pt, backlog) — closes DRIFT-F005-LOOKUP-CONVENTION ([process-gap] F-006). Wave 6 total: 5→6 stories, 32→35 pts. Grand totals: stories 38→39, pts (wave 0–6) 184→187, (incl. maintenance) 194→197. Draft/unscheduled: 9→10. E-phase stories: 26→27. Backlog: 2→3. |
| 2.5 | 2026-06-29 | CR-W5-SCOPE-SPLIT ruling (product-owner + human): add S-6.06 (5pts, P1, Wave 5, draft) — daemon-side admin RPC handlers minted per adversary Pass 1 CRITICAL finding F-001. S-6.02 ships CLI-layer only (deferral note added); S-6.06 wires daemon-side mgmt.Handler registration for admin.key.register / revoke / expire / list-keys on control-mode daemon only. S-W5.02 depends_on updated to include S-6.06. Wave 5 total: 7→8 stories, 38→43 pts. Grand totals: stories 37→38, pts (wave 0–6) 179→184, (incl. maintenance) 189→194. Draft/unscheduled: 8→9. E-phase stories: 25→26. |
| 2.4 | 2026-06-29 | CR-009 ruling (product-owner): add S-6.05 (3pts, P2, Wave 6, draft) — SVTN destroy lifecycle deferred from S-6.02. BC-2.07.001 PC-3 (Destroy) and VP-048 property 2 now owned by S-6.05; PC-1/PC-2 remain S-6.02. Wave 6 total: 4→5 stories, 29→32 pts. Grand totals: stories 36→37, pts (wave 0–6) 176→179, (incl. maintenance) 186→189. Draft/unscheduled: 7→8. BC-2.07.001 Traceability updated with PC split. |
| 2.3 | 2026-06-29 | Add S-HRD.02 (draft, E-HRD hardening epic) — Architect Ruling 1 (S-W5.01 mgmt-server convergence): daemon logging infrastructure + security-event emission. Owns the BC-2.07.004 PC-3 / EC-004 "logs a security event" deferral from S-W5.01 AC-003 v1.6; establishes daemon-wide slog seam on mgmt.Server + access.go wiring. MEDIUM severity. Total stories: 35→36. |
| 2.2 | 2026-06-29 | Add S-HRD.01 (draft, E-HRD hardening epic) — ARCH-12 v1.5 Wave-5 Convergence Round-5 Finding 2 (DEFER-WITH-FOLLOWUP): client write deadlines on `cmd/sbctl/client.go` Authenticate() + dispatch() write paths (CWE-400 defense-in-depth, MEDIUM severity). Introduces S-HRD.NN hardening story scheme. BC-2.07.003 Inv-2 annotation intent recorded in stub. Total stories: 34→35. |
| 2.1 | 2026-06-29 | Add S-W5.03 (2pts, draft, E-9 CI/devops epic) — ARCH-12 v1.5 Ruling S follow-up: CI assertion that release binary version is semver not "dev" (BC-2.07.004 PC-7 enforcement). Non-blocking for Wave 5; must ship before first tagged management-plane release. Total stories: 33→34; Draft/unscheduled: 6→7. |
| 2.0 | 2026-06-28 | Wave-5 management plane net-new stories: add S-W5.01 (8pts, internal/mgmt + config E-CFG-008/009 + cmd/switchboard wiring, BC-2.07.004 + BC-2.09.003 PC-10/11, VP-064/065/066) and S-W5.02 (5pts, e2e harness, BC-2.07.002, VP-049). Re-scope S-6.03 to client-auth-only boundary (v2.0): Authenticate() fail-closed, --key/--target/--json/--timeout, VP-067 + VP-030 only (VP-049 moved to S-W5.02); fix EC-002 E-ADM-001 → E-ADM-010. Wave-5 totals: 5→7 stories, 25→38 pts. Grand totals: stories 31→33, pts 163→176 (wave 0–6), 173→186 (incl. maintenance). BC coverage 44→45. VP coverage 63→67. Serialization note updated: S-W5.01 can run parallel with sbctl-side stories; S-W5.02 gates on S-6.03 + S-W5.01 both merged. |
| 1.9 | 2026-06-28 | Consistency audit: F-005 — update Wave-5 serialization note: S-5.03 must precede S-5.01 (S-5.01.depends_on now includes S-5.03); Wave-5 arithmetic unchanged (5 stories, 25 pts). |
| 1.8 | 2026-06-28 | Wave-5 planning: add S-5.03 (2pts, P1, depends S-4.01, closes drift S401-O3); update S-5.02 title + points 3→5 (p99 scope + canonical surface reconciliation per BC-2.06.003 v1.2); fix S-6.02 dependency inversion (depends_on adds S-6.03; blocks removes S-6.03); update Wave-5 summary 4→5 stories, 21→25pts; update totals (Pending 8→9, Total stories 30→31, points 159→163). Add Wave-5 serialization constraints note. |
| 1.7 | 2026-06-28 | Wave 4 cycle-close: S-4.01 (PR #24, e415d31), S-4.02 (PR #25, 95729c7), S-4.03 (PR #26, 8d9744f), S-4.04 (PR #27, 42c51e2), S-6.01 (PR #28, abeba27) marked completed. Summary counts updated: Complete 13→18, Pending 13→8, Total 28→30. Added S-6.04 (draft, SIGHUP reload, BC-2.09.003 Inv-3/EC-004) and S-BL.ARQ-TX (backlog, retransmit-SEND PC-3 wiring, depends S-4.03) as new stub rows. |
| 1.6 | 2026-06-28 | S-6.01 narrowed to v1.4 per BC-2.09.003 v1.3 right-sizing (commit bc52270): AC-009 scoped to tick_interval application only (TestConfigTickIntervalApplied); listen_addr binding dependency flagged on S-BL.NI (updated backlog row to own cfg.ListenAddr wiring per BC-2.09.003 PC-9 DEFERRED-APPLICATION); drain_timeout/upstream_routers/keepalive_interval application remains S-7.04 Wave 7. |
| 1.5 | 2026-06-27 | S-4.01: AC-007 (Hits() hit counter, BC-2.02.009 postcondition 2) + deferrals section (router wiring → S-4.04; EC-005 logging → S-4.04; BC-2.02.001 EC-003 queue-with-timeout out of scope). S-4.04: added BC-2.02.009 to bc_traces; AC-004 (OnFrameArrival DropCache wiring) + AC-005 (EC-005 collision logging via WithLogger); scope transfer note citing pass-2 ruling. dependency-graph.md BC-2.02.009 row updated. Per pass-2 scope ruling pass2-bc009-scope.md. |
| 1.4 | 2026-06-27 | Post-merge traceability correction: rewrite S-BL.NI backlog row — remove stale FailureCounter-wiring obligation (COMPLETED by PR #20, C-1 RESOLVED per ARCH-08 v2.3 §6.5.1); scope S-BL.NI to remaining network-ingress listener obligation only; update ARCH-08 citation from v2.2 to v2.3. (Wave 3 pre-gate consistency audit Finding D5-1.) |
| 1.3 | 2026-06-27 | Added S-M.01, S-M.02 maintenance stories; introduced E-MAINT epic; updated summary counts. |
