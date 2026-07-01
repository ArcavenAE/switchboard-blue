---
artifact_id: S-BL.LOOKUP
document_type: story
level: ops
version: "1.5"
status: draft
producer: product-owner
timestamp: 2026-07-01T00:00:00
phase: 2
cycle: v1.0.0-greenfield
epic: E-6
wave: 6
points: 1
priority: P2
scope: E
depends_on: [S-6.02]
bc_traces: []
vp_traces: []
subsystems: [admission-security]
closes_drift: [DRIFT-F005-LOOKUP-CONVENTION]
---

# S-BL.LOOKUP: Migrate `AdmittedKeySet.Lookup` / `LookupByPubkey` to `(AdmittedKey, bool)` Value-Return Form

## Summary

Draft story (Wave 6 Tranche A). Migrates admitted-keyset lookup to the shared convention. Was blocked on S-6.02 (merged PR #34). Migrates `AdmittedKeySet.Lookup` and `AdmittedKeySet.LookupByPubkey` from pointer-return (`*AdmittedKey`) to value-return `(AdmittedKey, bool)` form per go.md rule 12 (locked-accessor convention).

## Context

Source: DRIFT-F005-LOOKUP-CONVENTION (tech-debt-register.md). Both accessor methods currently return `*AdmittedKey`. The project-wide locked-accessor convention (go.md rule 12; every other locked accessor in the codebase returns values) mandates `(T, bool)` — value + present-flag. Callers currently use `if stored == nil` nil-check pattern; migration changes to `stored, ok := ...; if !ok`.

Implementation note from ARCH-04 §F-005 Ruling:
- `LookupByPubkey` delegates to `Lookup` — both change together
- Callers (develop head, post-S-6.06 merge): `SVTNManager.ExpireKey` (line 352), `SVTNManager.CallerKeyRole` (line 404), `SVTNManager.CallerKeyRoleActive` (line 425), `SVTNManager.IsRegisteredAnyState` (line 526)
- `SVTNManager.RevokeKey` does NOT call `LookupByPubkey` — it was refactored to `RevokeKeyIfRoleMatches` per ADR-004 Addendum H2 (ARCH-04 v1.14); the original story stub predated S-6.06
- `ed25519.PublicKey` is `[]byte`; struct copy shares backing array — deep-clone still required: `cp.PublicKey = append([]byte(nil), src.PublicKey...)`
- One-line change per call site, no logic change

## Acceptance Criteria

- Migrate `AdmittedKeySet.Lookup(svtnID [16]byte, nodeAddr [8]byte) (AdmittedKey, bool)` from pointer return
- Migrate `AdmittedKeySet.LookupByPubkey(svtnID [16]byte, pubkey ed25519.PublicKey) (AdmittedKey, bool)` from pointer return
- Update all 4 current call sites to use `stored, ok := ...; if !ok` pattern: `SVTNManager.ExpireKey`, `SVTNManager.CallerKeyRole`, `SVTNManager.CallerKeyRoleActive`, `SVTNManager.IsRegisteredAnyState` (note: `SVTNManager.RevokeKey` was superseded by `RevokeKeyIfRoleMatches` per ADR-004 Addendum H2 and is not a call site)
- Deep-clone `PublicKey` field in the value copy
- `go test -race ./internal/svtnmgmt/...` passes with no race detector findings
- `just lint` passes with zero warnings

## Earliest Wave

Wave 6+ (unblocked after S-6.02 merges to develop).

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.5 | 2026-07-01 | F-P12L3-01: H1 title expanded to name both `Lookup` and `LookupByPubkey`; aligns with ARCH-04 v1.14 §F-005 heading, STORY-INDEX row, and sprint-state title. |
| 1.4 | 2026-07-01 | F-P9L3-03: ARCH-04 pin in §Context corrected v1.13→v1.14 (ARCH-04 was bumped to v1.14 by S-6.06 Pass-25 sibling sweep; prior pin was stale). No AC or scope change. |
| 1.3 | 2026-07-01 | F-P5L3-01: §Summary prose updated — replaced stale "Backlog stub. Blocked until S-6.02 merges" with "Draft story (Wave 6 Tranche A). Migrates admitted-keyset lookup to the shared convention. Was blocked on S-6.02 (merged PR #34)." No AC or scope change. |
| 1.2 | 2026-07-01 | Pass-4 L3 F-L3-Med-01 governance: epic frontmatter confirmed E-6 (story depends_on S-6.02, Wave 6 Tranche A alongside S-6.07/S-6.05; STORY-INDEX row was the erroneous artifact and has been corrected to E-6). No AC or scope change. Closes F-L3-Med-01. |
| 1.1 | 2026-07-01 | Pass-1 lens-3 fix-burst: F-P1L3-001 — AC-1 signature corrected to `Lookup(svtnID [16]byte, nodeAddr [8]byte)` and `LookupByPubkey(svtnID [16]byte, pubkey ed25519.PublicKey)`; F-P1L3-002 — frontmatter `status: backlog→draft`, `wave: unscheduled→6` per sprint-state.yaml lines 41-58 and STORY-INDEX line 77; F-P1L3-003 — callsite list refreshed vs develop head post-S-6.06: RevokeKey superseded by RevokeKeyIfRoleMatches (ADR-004 H2), 4 actual callsites enumerated (ExpireKey, CallerKeyRole, CallerKeyRoleActive, IsRegisteredAnyState); F-P1L3-006 — stub parenthetical "(To be filled by story-writer...)" removed. |
| 1.0 | 2026-06-29 | Minted per S-6.02 lens3 F-006 ([process-gap]): DRIFT-F005-LOOKUP-CONVENTION has no draft story stub. Closes task #66. |
