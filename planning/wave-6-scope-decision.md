---
artifact_id: wave-6-scope-decision
document_type: planning-decision
level: ops
version: "1.0"
status: draft
producer: product-owner
timestamp: 2026-06-30T00:00:00
phase: 3
cycle: v1.0.0-greenfield
wave: 6
inputs_read:
  - path: .factory/STATE.md
    note: wave_5_gate CONVERGED, phase_3_active_wave 5, 25 completed stories
  - path: .factory/stories/STORY-INDEX.md
    version: "3.23"
  - path: .factory/stories/dependency-graph.md
    version: "1.6"
  - path: .factory/stories/ (all candidate frontmatter)
---

# Wave-6 Scope Decision

**Date:** 2026-06-30
**Author:** product-owner
**Status:** DRAFT — awaiting orchestrator approval before story-writer/state-manager follow-up burst

## 1. Wave-6 Stories

The STORY-INDEX v3.23 already lists Wave 6 as 7 stories / 40 points. This decision
confirms that composition, adjusts one story (S-BL.LOOKUP promoted from backlog), and
documents the final serialization plan.

| Story ID | Title | Points | Priority | Deps (all merged?) | Parallel Group | Notes |
|----------|-------|--------|----------|--------------------|----------------|-------|
| S-W5.04 | daemon-side paths.list / router.metrics / router.status RPC handlers | 5 | P1 | S-5.02 ✓, S-W5.01 ✓ | A (first tranche) | Re-scheduled from Wave 5 per F-W5P1-004; unblocked; no cmd/sbctl conflict |
| S-6.05 | SVTN destroy lifecycle: SVTNManager.Destroy + sbctl admin svtn destroy | 3 | P2 | S-6.02 ✓ | A (first tranche) | Deferred from S-6.02 per CR-009; adds cmd/sbctl/admin.go extension |
| S-6.07 | Register admin.svtn.create handler + sbctl admin svtn create CLI | 3 | P2 | S-6.02 ✓, S-6.06 ✓ | A (first tranche) | Human-approved Path B; deps both merged |
| S-BL.LOOKUP | Migrate AdmittedKeySet.Lookup/LookupByPubkey to (AdmittedKey, bool) | 1 | P2 | S-6.02 ✓ | A (first tranche) | Unblocked since PR #34 per O-W5P1-02; go.md rule 12 compliance; 1 pt |
| S-7.01 | XOR parity FEC for single-loss recovery (internal/arq) | 8 | P1 | S-4.03 ✓ | B (second tranche) | PE-phase; no cmd/sbctl or cmd/switchboard conflict |
| S-7.02 | SVTN-scoped multicast session discovery (internal/discovery) | 8 | P1 | S-2.02 ✓, S-3.02 ✓ | B (second tranche) | PE-phase; entirely new package |
| S-7.03 | Console remote control via sbctl attach/detach/switch | 5 | P1 | S-3.02 ✓, S-6.03 ✓ | B (second tranche) | PE-phase; touches internal/session + cmd/sbctl |
| **Total** | | **33** | | | | |

**Note:** S-7.04 (E-to-PE graduation, 8 pts, P2) is in the STORY-INDEX Wave-6 baseline (total 40 pts) but is
explicitly deferred below. With S-7.04 deferred and S-BL.LOOKUP (1 pt) added, Wave-6 total is 33 pts.

Wave 5 delivered 43 pts. Target was 35-50 pts. 33 pts is at the low end of target range; see
Section 5 (risk register) for why deferring S-7.04 is the right call.

## 2. Deferred to Later Waves / Hardening

| Story ID | Title | Points | Reason | Target |
|----------|-------|--------|--------|--------|
| S-7.04 | E-to-PE router graduation and graceful drain with node migration | 8 | P2 priority; introduces a new `internal/drain` package + drain-timeout/upstream_routers/keepalive_interval config application (BC-2.09.001/002/003) that requires deep integration with cmd/switchboard routing wiring not yet established. Wave 6 is already adding S-7.01 (arq FEC) + S-7.02 (discovery, new package) concurrently; adding a third complex multi-package story in the same tranche elevates merge-conflict risk and wave-adversarial convergence cost. Deferring to Wave 7 does not block any BC (BC-2.09.001/002 are standalone graduation contracts). | Wave 7 |
| S-HRD.01 | Add conn.SetWriteDeadline to client write paths (CWE-400 defense-in-depth) | TBD | MEDIUM severity but explicitly tagged defense-in-depth; Rulings V/Y in ARCH-12 v1.5 scoped client deadlines to read+dial only. Server-side close bounds practical risk. Phase-6 hardening candidate per PO constraint in scope document. | Phase 6 hardening |
| S-HRD.02 | daemon logging infrastructure + security-event emission (BC-2.07.004 PC-3/EC-004) | TBD | MEDIUM severity; requires establishing daemon-wide slog seam before any log calls. S-W5.01 AC-003 deferred this because no structured logging infra exists. Phase-6 hardening candidate. Depends on S-W5.01 (merged). Could be slotted into Wave 6 if S-W5.04 ADR establishes a slog pattern first — but that creates an intra-wave ordering dependency that would force serialization of S-W5.04 → S-HRD.02, adding risk. Defer to Phase 6. | Phase 6 hardening |
| S-BL.OA | outer-assembler — compose ChannelFrame + OuterHeader into wire frames | TBD | Backlog stub, no ACs, no BC traces. The composed wire-format gap (wave-adv F-003/F-004) has not been promoted to a scheduled story. Requires story-writer expansion pass before scheduling. | Wave 7+ / backlog |
| S-BL.ARQ-TX | Wire ARQ retransmit-SEND path into router/multipath dispatch (BC-2.02.005 PC-3) | TBD | Backlog stub, no ACs. Requires router wiring that is not established yet; S-7.01 (FEC in arq) would be a natural predecessor. Schedule after S-7.01 merges. | Wave 7+ |
| S-BL.NI | network-ingress listener + live-path integration test + cfg.ListenAddr binding | TBD | Backlog draft; BC-2.09.003 PC-9 listen_addr deferral anchored here. Architecturally significant (first real inbound network connection). Not Wave 6 material; needs its own wave with explicit cross-component lock-ordering review (PROCESS-GAP-W4). | Wave 7+ |
| S-6.04 | SIGHUP config reload with fail-closed safety | TBD | Draft stub, no ACs. BC-2.09.003 Inv-3/EC-004. Low urgency; Wave 6+ scope but not ready for implementation. | Wave 6+ (but post-Wave-6 if not fleshed out before Wave-6 kick-off) |

## 3. Serialization Plan

Wave 6 splits into two tranches. Within each tranche, stories can run concurrently on separate
branches. Tranche A must substantially complete (all stories merged to develop) before Tranche B
stories enter final adversarial convergence and merge.

### Tranche A — Management-Plane Completion (run concurrently, serialize on cmd/sbctl)

**Stories:** S-W5.04, S-6.05, S-6.07, S-BL.LOOKUP

```
S-W5.04  ─── touches internal/metrics + internal/mgmt only — no cmd/sbctl conflict
S-BL.LOOKUP ─ touches internal/admission only — no conflicts
S-6.05   ─┐
S-6.07   ─┘  BOTH touch cmd/sbctl/admin.go — MUST SERIALIZE
```

**Intra-Tranche-A serialization constraint:**
- S-6.05 and S-6.07 both extend `cmd/sbctl/admin.go` (adding `svtn destroy` and `svtn create`
  subcommands). They MUST NOT run in parallel. Recommended order: S-6.07 first (Create is
  the natural predecessor to Destroy in the SVTN lifecycle), then S-6.05. Alternatively, assign
  one branch and have the implementer do both sequentially. Either way — one branch at a time
  for the admin.go extension work.
- S-W5.04 and S-BL.LOOKUP have zero file conflict with each other or with S-6.05/S-6.07;
  they can run concurrently on separate branches.

**Rationale:** Tranche A closes the management plane: daemon-side RPC handlers for
paths/metrics (S-W5.04), the SVTN lifecycle create/destroy CLI round-trip (S-6.07 + S-6.05),
and the go.md rule-12 Lookup refactor (S-BL.LOOKUP). These together complete Epic E-6
and E-5 daemon-side obligations. Running them first ensures the management plane is fully
exercised before the PE-phase network protocol stories (Tranche B) add new packages.

### Tranche B — PE-Phase Network Features (run concurrently, no shared packages)

**Stories:** S-7.01, S-7.02, S-7.03

```
S-7.01  ─── internal/arq only — no conflict with S-7.02 or S-7.03
S-7.02  ─── internal/discovery (new package) — no conflict
S-7.03  ─── internal/session + cmd/sbctl (new subcommands) — no cmd/sbctl/admin.go conflict
```

All three Tranche-B stories can run concurrently on separate branches. The only potential
conflict is if S-7.03 and any Tranche-A story both touch `cmd/sbctl/main.go` command
registration. S-7.03 adds `sbctl attach`, `sbctl detach`, `sbctl switch` — new top-level
commands, not admin subcommands. If Tranche A is fully merged before Tranche B begins, there
is no conflict. If there is overlap, S-7.03 and any open Tranche-A story with cmd/sbctl
changes should coordinate on the main.go registration commit (carry a rebase obligation).

**Rationale:** S-7.01 (FEC) and S-7.02 (discovery) are entirely new packages with no shared
state. S-7.03 requires S-6.03 (merged) but no Wave-6 predecessor. Running all three in
parallel maximizes throughput and keeps wave calendar short.

### Wave-Gate Sequence

```
Tranche A: [S-W5.04 ∥ S-BL.LOOKUP ∥ (S-6.07 → S-6.05)] → Tranche-A wave-adversarial
Tranche B: [S-7.01 ∥ S-7.02 ∥ S-7.03]                  → Tranche-B wave-adversarial
Wave-6 gate: combined wave-adversarial on full develop branch after all 7 merged
```

Whether to run tranches sequentially or allow Tranche B to start while Tranche A is still
in adversarial convergence is an orchestrator call. The safe default is sequential tranches.
Parallel tranches are acceptable if Tranche-A stories are at the PR-open stage (merge
imminent) and Tranche-B branches carry no cmd/sbctl/admin.go conflict.

## 4. BC / VP Coverage Delta Added by Wave 6

Wave 6 does not mint new BCs. Coverage delta is implementation reachability for existing
unimplemented BCs and VP obligations.

| Story | BCs activated | VPs satisfied | Net new coverage |
|-------|--------------|---------------|------------------|
| S-W5.04 | BC-2.06.003 (daemon-side PC-1/PC-2/PC-3) | VP-047, VP-062 | Daemon-side paths.list/router.metrics/router.status RPC types + integration test |
| S-6.05 | BC-2.07.001 PC-3 (Destroy) | VP-048 (property 2) | SVTN destroy lifecycle completeness |
| S-6.07 | BC-2.07.001 PC-1 (Create via RPC+CLI) | VP-048 (property 1 — admin.svtn.create reachability) | CLI/RPC reachability gap closed; BC-2.07.001 PC-1 was unverifiable pre-Wave-6 (S-6.02 Pass-1 F-003/F-010 HIGH findings) |
| S-BL.LOOKUP | (no BC) | (no VP) | go.md rule-12 compliance; closes DRIFT-F005-LOOKUP-CONVENTION |
| S-7.01 | BC-2.02.007 (XOR FEC) | VP-043 | Full FEC encode/decode path |
| S-7.02 | BC-2.03.001, BC-2.03.002, BC-2.03.003 (session discovery) | VP-044, VP-045, VP-055 | All session-discovery BCs activated |
| S-7.03 | BC-2.08.001 (console remote), BC-2.06.001, BC-2.06.002 | VP-050 | Console remote-control surface via sbctl |

**Previously unverifiable BCs closed by Wave 6:**
- BC-2.07.001 PC-1 (svtn.create RPC reachability) — was a HIGH finding in S-6.02 adversarial.
  S-6.07 closes this gap.
- BC-2.07.001 PC-3 (svtn.destroy) — deferred from S-6.02 per CR-009 ruling. S-6.05 closes.
- BC-2.06.003 daemon-side (VP-047 integration test, VP-062 fuzz harness) — deferred from
  S-5.02 per Pass-4 Ruling 1. S-W5.04 closes.

After Wave 6: all 45 BCs will be fully activated (implementation reachable). Remaining
Phase-6 hardening obligations (S-HRD.01, S-HRD.02) are defense-in-depth, not BC gaps.

**S502-DEFER-1 cross-note:** S-W5.04 activates BC-2.06.003 daemon-side PC-1/PC-2/PC-3.
S502-DEFER-3 (BC-2.06.003 PC-3 F-M3 failed+pending precedence ambiguity) should be
re-examined once S-W5.04 is drafted for convergence — the daemon-side implementation will
make the precedence rule concrete.

## 5. Risk Register (Top 3)

### RISK-W6-001: S-6.05 / S-6.07 cmd/sbctl/admin.go serialization conflict
**Likelihood:** HIGH (both stories must extend the same file)
**Impact:** MEDIUM (merge conflict, rebase overhead, potential test breakage if registration order differs)
**Mitigation:** Enforce one-branch-at-a-time rule for the admin.go work: deliver S-6.07 first
(Create), merge it, then S-6.05 (Destroy) branches off the updated develop. Document this
constraint in the wave dispatch order. Alternatively, assign both stories to the same branch
(single worktree) and let the implementer deliver them sequentially without a merge in between.
**Owner:** orchestrator (dispatch scheduling)

### RISK-W6-002: S-7.02 discovery package is entirely greenfield — high adversarial convergence cost
**Likelihood:** MEDIUM (new package, 3 BCs, 3 VPs — BC-2.03.001/002/003 cover multicast
session presence, join, leave, and SVTN-scoped isolation which have no existing code to reference)
**Impact:** HIGH (if BC-2.03.* are underdetermined for the implementation, spec clarification
cycles will block the story in adversarial convergence as seen with S-6.06's 28-pass convergence)
**Mitigation:** Before dispatching S-7.02 for implementation, have the implementer do a
read-pass on BC-2.03.001/002/003 and flag any ambiguities to the PO. Consider a pre-implementation
spec-clarification burst (product-owner + architect) to tighten the BC preconditions/postconditions
for the multicast protocol before the TDD cycle starts. S-7.02 should NOT start concurrently
with S-7.01 if BC-2.03.* still have open ambiguities — defer S-7.02 until spec is confirmed clean.
**Owner:** product-owner (spec pre-check), orchestrator (dispatch gating)

### RISK-W6-003: S502-DEFER-1 (BC-2.06.003 PC-3 auth-timeout on runRouterStatus) bleeds into Wave 6
**Likelihood:** MEDIUM (S502-DEFER-1 is an unresolved MEDIUM defer on BC-2.06.003 PC-3 in
the already-merged S-5.02; S-W5.04 activates BC-2.06.003 daemon-side and its adversarial
review will re-examine the PC-3 spec; any residual client-side coverage gap on the alias path
may surface as a HIGH finding against S-W5.04 or against BC-2.06.003 itself)
**Impact:** MEDIUM (could force a spec-tightening pass on BC-2.06.003 mid-wave, requiring
PO ruling and downstream propagation — the exact SIBLINGSWEEP pattern that hit S-6.06 for 28 passes)
**Mitigation:** Before dispatching S-W5.04, the PO should close S502-DEFER-3 (BC-2.06.003
PC-3 F-M3 failed+pending precedence) by issuing a formal ruling and updating BC-2.06.003.
This transforms a latent adversarial ambiguity into a known-closed issue. Do NOT enter
S-W5.04 adversarial convergence with an open BC spec ambiguity on its anchoring BC.
**Owner:** product-owner (pre-wave spec closure)

---

## Appendix A: Dependency Verification

All Wave-6 stories have verified-merged dependencies per dep-graph.md v1.6 and STATE.md:

| Story | Depends On | Merged SHA | Verified |
|-------|-----------|------------|---------|
| S-W5.04 | S-5.02, S-W5.01 | 98eb8b7, 0d499ac | yes |
| S-6.05 | S-6.02 | b36cb9b | yes |
| S-6.07 | S-6.02, S-6.06 | b36cb9b, 3ee9c38 | yes |
| S-BL.LOOKUP | S-6.02 | b36cb9b | yes |
| S-7.01 | S-4.03 | 8d9744f | yes |
| S-7.02 | S-2.02, S-3.02 | a06b306, 1ff74f5 | yes |
| S-7.03 | S-3.02, S-6.03 | 1ff74f5, d854978 | yes |

No story enters Wave 6 with an unmerged dependency.

## Appendix B: Point Budget

| Category | Stories | Points |
|----------|---------|--------|
| Tranche A (management-plane completion) | S-W5.04, S-6.05, S-6.07, S-BL.LOOKUP | 12 |
| Tranche B (PE-phase network features) | S-7.01, S-7.02, S-7.03 | 21 |
| **Wave-6 total** | **7** | **33** |
| Wave-5 (reference) | 8 | 43 |
| Target range | 35–50 pt | 35–50 |

33 pts is 2 points below the 35-pt floor. This is intentional: S-7.04 (8 pts, P2) was
in the STORY-INDEX baseline (40 pts total) but is deferred because:
1. P2 priority — it is not blocking any BC gap.
2. The internal/drain package introduction is architecturally significant; it should not
   share a wave-adversarial run with three other complex stories.
3. Tranche B already has 21 pts of PE-phase work that is entirely new territory. Adding
   S-7.04 would make the wave-adversarial convergence run against 29 pts of new code.

If the orchestrator judges that 33 pts is too light, S-7.04 can be re-added to Tranche B
(making 41 pts). The risk tradeoff is documented in RISK-W6-002 above. PO recommendation:
keep S-7.04 deferred; deliver clean at 33 pts rather than rushed at 41 pts.

## Appendix C: Deferred-Story Carry-Forward

The following open drift items and deferred ACs carry into Wave 6 planning:

| Item ID | Description | Owner | Carry-forward action |
|---------|-------------|-------|---------------------|
| S502-DEFER-1 | runRouterStatus auth-timeout wrap (BC-2.06.003 PC-3 gap) | implementer | Close before S-W5.04 adversarial; PO to issue PC-3 precedence ruling |
| S502-DEFER-3 | BC-2.06.003 PC-3 F-M3 spec-ambiguity (failed+pending precedence) | product-owner | Formal PO ruling pre-Wave-6 dispatch |
| S502-DEFER-2 | writeSuccess os.Exit(3) outside main() (go.md rule violation) | implementer | Phase-5 cleanup; route to hygiene commit pre-Wave-6 |
| SW502-DEFER-1 | closingConn.Read server-shutdown/FIN conflation | implementer | Wave-6 hygiene |
| SW502-DEFER-2 | closingListenerWrapper goroutine WaitGroup gap | implementer | Wave-6 hygiene |
| SW502-DEFER-6 | closed map dead code in closingListenerWrapper | implementer | Wave-6 hygiene |
| DRIFT-SW501-NITPICK | stale "Stub: Red Gate" comments + dead `_ = pub` | implementer | Bundle with Wave-6 kick-off hygiene commit |
| S502-DEFER-4 | ARCH-11 VP total 75 vs actual 76; dep-graph v1.4 VP total 67 vs actual 76 | architect | Architecture doc sweep — pre-Wave-6 state-manager burst |
