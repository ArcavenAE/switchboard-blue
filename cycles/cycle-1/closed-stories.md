---
document_type: closed-stories
level: ops
version: "1.0"
status: archive
producer: state-manager
timestamp: 2026-06-25T00:00:00Z
cycle: cycle-1
inputs: [STATE.md]
input-hash: ""
traces_to: STATE.md
---

# Closed Stories — cycle-1

## Extracted from STATE.md on 2026-06-25

---

## S-1.02 Closed — 2026-06-24

- PR #2 merged via squash at `9e9a98a` on `develop`; alpha tag `alpha-20260624-193019-9e9a98a` cut automatically
- 9 commits on feature branch: stubs → failing tests → implementation → 5 fix commits → demos
- 9 adversary passes; finding trajectory 9 → 11 → 7 → 5 → 4 → 3 → 0 → 0 → 0; 39 findings resolved across passes 1-6
- Spec versions advanced: story S-1.02 rev 1.0 → 1.5; BC-2.01.001 v1.0 → 1.1; BC-2.01.002 v1.1 → 1.3; VP-053 v1.0 → 1.2
- Final tree: halfchannel.go (171 LOC), halfchannel_test.go (612 LOC), wraparound_test.go (35 LOC); lint clean; race-free
- Worktree `.worktrees/S-1.02/` removed; local + remote `feature/S-1.02-halfchannel-clock` branches deleted
- Unblocks: S-3.01, S-4.02, S-4.03

---

## S-2.01 Closed — 2026-06-24

- PR #5 merged via squash at `3c4104e` on `develop`; alpha tag `alpha-20260625-023528-3c4104e` cut automatically
- 7 commits on feature branch: stubs+tests / impl / pass-2 helper / pass-2 KAT / pass-3 length guard / pass-7 doc fix / Step 5 example
- 12 adversary passes; finding trajectory 9 → 2 → 4 → 1 → 0 → 0 → 1 → 0 → 1 → 0 → 0 → 0; 17 findings resolved across 9 fix bursts
- AI code review: APPROVE, 2 informational notes
- Security review: CLEARED on 9 axes; 1 LOW (SEC-001: unreachable nil-OKM in hkdfSHA256, defensive-coding nit)
- Spec versions at closure: BC-2.05.005 (no change), story rev 5, VP-004/005/006 v1.1, ARCH-04 v1.1
- Notable mid-cycle: filed drbothen/vsdd-factory#263 (PO agent overreach — PR #4 closed without merge)
- Unblocks: S-2.02 (Admission + SVTN isolation), S-4.04 (Split-horizon loop prevention)

---

## S-2.02 Closed — 2026-06-25

- PR #6 merged via squash at `a06b306` on `develop` (2026-06-25T13:57:58Z); alpha tag `alpha-20260625-135909-a06b306` cut automatically
- 13 commits on feature branch: stubs+tests / impl / 5 adversary fix commits / Step 5 example godoc
- 8 adversary passes; finding trajectory converged at passes 6/7/8 (3 consecutive clean); BC-5.39.001 satisfied
- 8 Example godoc demos pinning AC-001..007 + EC-003
- Post-merge: `go test -race` PASS, `just lint` 0 issues on develop main worktree
- Cycle-closing checklist: zero process-gap findings; no follow-up codifications required
- Spec versions at closure: BC-2.05.001, BC-2.05.002, BC-2.05.006, BC-2.05.007 (implemented via S-2.02 / PR #6)
- Unblocks: S-1.03 (Session continuity, 5pts)

---

## Wave-1 Closed — 2026-06-24

Wave 1 of Phase 3 (TDD implementation) is fully closed. Re-closure after rollback per drbothen/vsdd-factory#260.

### Stories Merged

| Story | PR | Merge SHA | Points |
|-------|-----|----------|--------|
| S-1.01 frame-codec | #1 | `1c76160` | 8 |
| S-1.02 halfchannel-clock | #2 | `9e9a98a` | 5 |
| Wave-1 refactor F-001+F-002 | #3 | `4be1b53` | (drift closure, no points) |

### Gate Disposition

`pass-with-clean-drift` — every wave-1 finding has either landed or been routed to a concrete backlog story (S-BL.OA).

### Spec Versions at Wave-1 Closure

- BC-2.01.001 v1.1
- BC-2.01.002 v1.4
- BC-2.01.004 (post burst-A invariant-3 alignment with ARCH-02)
- BC-2.01.005 v1.1
- VP-016 v1.1, VP-017 v1.1, VP-018 v1.1, VP-041 v1.1, VP-051 v1.1, VP-053 v1.2
- Story S-1.01 rev 1.1, S-1.02 rev 1.5, S-BL.OA rev 0.1-backlog-stub
- ARCH-09 v1.1 (post-burst-A time-package carve-out clarification)

### Adversary Cycle Metrics (Wave-1)

- S-1.02 per-story passes: 9 (trajectory 9 → 11 → 7 → 5 → 4 → 3 → 0 → 0 → 0)
- Wave-1 gate adversary: 1 pass, CONVERGED with 4 findings (2 MED deferrable, 2 LOW)
- Refactor F-001+F-002 PR #3 adversary: 3 passes, all 0 findings (BC-5.39.001 satisfied)

### Convergence Streaks

| Cycle | Streak | Final |
|-------|--------|-------|
| S-1.02 | 3/3 | CONVERGED 2026-06-24 |
| Wave-1 gate | 1/1 (single-pass closure) | CONVERGED with deferrable mediums 2026-06-24 |
| Refactor PR #3 | 3/3 | CONVERGED 2026-06-24 |

### Wave-1 Drift Register

All items resolved or routed to concrete backlog targets:

- **wave-adv F-001 (MED) — RESOLVED.** Spec side: BC-2.01.002 v1.4 PC5 (burst A commit `6c064d9`). Code side: PR #3 (merge `4be1b53`) — `MaxPayloadSize = 65515` + `ErrPayloadTooLarge` sentinel + Enqueue MTU validation.
- **wave-adv F-002 (MED) — RESOLVED.** PR #3 (merge `4be1b53`) — `type FrameType byte` named type in `internal/frame`; `Valid()` method; `ErrInvalidFrameType` sentinel; `ParseOuterHeader` enum validation.
- **wave-adv F-003 (LOW) — DEFERRED to S-BL.OA.** Composed wire-format test belongs to the outer-assembler story.
- **wave-adv F-004 (LOW, per-story-scope) — DEFERRED to S-BL.OA.** ARCH-02 channel-header serializer is the outer-assembler's responsibility.
- **consistency F-003 (MED, process-gap) — RESOLVED.** ARCH-09 Purity Rule 1 carve-out clarified (burst A commit `345d4f4`).
- **consistency F-005 (MED) — RESOLVED.** VP-018 stale-API harness fixed in burst-1 commit `e8af50a`.
- **consistency F-006 (LOW) — RESOLVED.** BC-2.01.004 invariant 3 aligned with ARCH-02 normative payload_len definition (burst A commit `345d4f4`).
- **consistency F-007 (LOW) — RESOLVED.** S-1.01 File Structure table now enumerates `internal/frame/address_test.go` (burst A commit `345d4f4`; story revision 1.0 → 1.1).
- **E-FRM ↔ E-PRT namespace (informational) — RESOLVED.** Cross-reference subsection added to `.factory/specs/prd-supplements/error-taxonomy.md` (burst A commit `6c064d9`). Canonical is E-PRT-*.

See `.factory/cycles/cycle-1/wave-1/consistency-validation.md` for the detailed consistency-validator report.

---

## Wave 2 Closure — 2026-06-25

**Gate disposition:** PASS_WITH_OBSERVATIONS
**Stories merged:** S-2.01 (5 pts, PR #5, `3c4104e`), S-2.02 (8 pts, PR #6, `a06b306`), S-1.03 (5 pts, PR #7, `f35e836`)
**Total points:** 18 pts
**Wave 2 total:** 3/3 stories merged

### S-1.03 — Session Continuity (5 pts, PR #7)

HMAC-authenticated session continuity for ReAuthenticate path. Adversary passes 3/4/5 clean (BC-5.39.001 satisfied). Merged at `f35e836` (2026-06-25).

Open carry-forwards:
- WAVE-2-MED-001: ReAuthState not evicted on RevokeKey/RegisterKey reset (Phase-6 hardening).
- VP-036: property test deferred (needs `internal/testenv.ConnectWithSourceIP`).
- VP-039-test-skip: t.Skip placeholder needed in `internal/routing/*_test.go`.
- SEC-003: sub-microsecond TOCTOU on `now` in ReAuthenticate (ACCEPTED, Phase-6).

### Wave 2 Gate Reports

- Consistency-validator: `cycles/cycle-1/wave-2/consistency-report.md` (0C/0H/2M/3L/4O)
- Fresh-context audit: `cycles/cycle-1/wave-2/fresh-context-audit.md` (0C/0H/1M/3L/3O)

### Wave 2 Drift Register

- **WAVE-2-MED-001 (OPEN, Phase-6):** ReAuthState not evicted on RevokeKey or RegisterKey reset; stale source-IP survives via CurrentSourceAddr.
- **WAVE-3-DEP-001 (OPEN, Wave 3 critical path):** verifyFrameHMAC is //nolint:unused on develop; Wave-2 router has zero frame-forgery defense until wired into RouteFrame.

See `cycles/cycle-1/wave-2/` for detailed gate reports.
