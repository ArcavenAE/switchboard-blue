---
artifact_id: adv-p1-pass-08
review_target: phase-1-spec-crystallization
producer: adversary
pass: 8
fresh_context: true
findings_count: 9
findings_by_severity: {critical: 0, high: 3, medium: 5, low: 1}
findings_with_process_gap: 0
verdict: NOT_CONVERGED
timestamp: 2026-06-23
---

# Adversarial Review — Pass 8

**Trajectory:** 27 → 18 → 17 → 21 → 17 → 14 → 7 → **9**. Non-monotonic; partial-fix regression dominates.

## High

### F-P8-001 — BC-2.05.004 still uses removed `sbctl svtn keys register|revoke|expire` CLI surface
- BC-2.05.004:66 Trigger: `sbctl svtn keys register|revoke|expire or equivalent API call`
- BC-2.05.004:81-83 three test vectors use `sbctl svtn keys register/revoke`
- interface-definitions.md:64-67 deleted those subcommands; canonical is `sbctl admin register-key/revoke-key`
- Route: product-owner. Fix: rewrite BC-2.05.004 Trigger + 3 test vectors to use `sbctl admin` surface.

### F-P8-002 — VP-030 and VP-049 invoke non-existent `sbctl status` subcommand
- VP-030.md:70 `exec.Command("sbctl", "--target", target, "status")`
- VP-049.md:49,96-100 same pattern
- interface-definitions.md defines `sbctl svtn status` / `sbctl router status` / `sbctl sessions status` but no bare `sbctl status`
- Route: architect. Fix: change VP harnesses to `sbctl router status` (or appropriate subcommand).

### F-P8-003 — BC-2.08.001 architecture_module drift unresolved
- BC-2.08.001:12 `architecture_module: internal/session` (set in pass-5 F-P5-015)
- ARCH-05:109 still says `cmd/sbctl`
- ARCH-11:67 still says `cmd/sbctl`
- VP-050.md:16 `module: cmd/sbctl`
- Route: architect. Fix: propagate the pass-5 decision (internal/session) to ARCH-05, ARCH-11, VP-050.

## Medium

### F-P8-004 — VP-026 cites "transitivity" invariant that doesn't exist in BC-2.02.003
- VP-026:42 cites "Invariant — Score ordering is transitive"
- BC-2.02.003:57-61 invariants are per-path scope, atomic ranking, router-quality — no transitivity
- Route: architect. Fix: either add transitivity invariant to BC-2.02.003 or rewrite VP-026 to test an actual postcondition (path ranking by RTT ascending).

### F-P8-005 — VP-027 title vs harness direction mismatch
- Title (VP-INDEX:53): "degradation only goes down"
- ARCH-07:72: "transitions monotonic under sustained degradation: green→yellow→red only"
- VP-027 harness:102-105 actually checks `if prevState == QualityRed && cur == QualityGreen { return false }` — recovery direction
- An implementer who deletes green→red direct transition (real bug) would pass this test
- Route: architect. Fix: rewrite either title (recovery direction) or harness (degradation direction). Recommend harness fix — title intent (no degradation skip) is the safer property.

### F-P8-006 — BC-2.05.007 test vector uses non-existent `sbctl keys list` form
- BC-2.05.007:78 vector: `sbctl keys list`
- interface-definitions.md has `sbctl svtn keys list` or `sbctl admin list-keys` — no bare `sbctl keys`
- Route: product-owner. Fix: change to `sbctl svtn keys list` or `sbctl admin list-keys`.

### F-P8-007 — BC-2.02.005 says SACK is in upstream payload; ARCH-02 says channel header
- BC-2.02.005:50 Postcondition 1: "cumulative ACK and SACK bitmap in every upstream frame payload"
- ARCH-02:115-122, 126-131 explicitly places SACK bitmap in channel header (sack_bitmap at offset 12, conditional on flags bit 2)
- Route: product-owner. Fix: update BC-2.02.005 to "in every upstream frame channel header (sack_bitmap field; flags bit 2 SACK_present set)".

### F-P8-008 — BC-2.02.007 says `FRAME_TYPE=PARITY`; canonical enum is `fec=0x05`
- BC-2.02.007:54 references `FRAME_TYPE=PARITY`
- BC-2.01.004:57 + ARCH-02:74 canonical: `frame_type | u8 enum: data=0x01, empty_tick=0x02, ctl=0x03, arq=0x04, fec=0x05`
- Route: product-owner. Fix: BC-2.02.007 → `frame_type=fec (0x05)`.

## Low

### F-P8-009 — architecture-feasibility-report:61 still says "deployment-operations (CAP-026–027)"
- Same row names BC-2.09.003 which traces to CAP-028 — internally inconsistent
- Route: architect. Fix: → "(CAP-026–028)".

## Verdict

**NOT_CONVERGED.** Trajectory 27→18→17→21→17→14→7→**9** is non-monotonic. Fresh-context review keeps surfacing partial-fix propagation drift each cycle. The spec corpus has ~120 cross-referencing files; mechanical sync requires CI-level enforcement, not human-loop iteration.

This is the asymptotic plateau. Either accept-with-debt and gate, or implement a mechanical CI gate (e.g., a script that walks BC/VP/ARCH/interface cross-references) before more refinement rounds.
