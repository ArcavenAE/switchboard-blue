---
artifact_id: S-BL.LOOKUP
document_type: story
level: ops
version: "1.1"
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

# S-BL.LOOKUP: Migrate `AdmittedKeySet.LookupByPubkey` to `(AdmittedKey, bool)` Value-Return Form

## Summary

Backlog stub. Migrates `AdmittedKeySet.Lookup` and `AdmittedKeySet.LookupByPubkey` from pointer-return (`*AdmittedKey`) to value-return `(AdmittedKey, bool)` form per go.md rule 12 (locked-accessor convention). Blocked until S-6.02 merges to avoid mid-flight collision.

## Context

Source: DRIFT-F005-LOOKUP-CONVENTION (tech-debt-register.md). Both accessor methods currently return `*AdmittedKey`. The project-wide locked-accessor convention (go.md rule 12; every other locked accessor in the codebase returns values) mandates `(T, bool)` — value + present-flag. Callers currently use `if stored == nil` nil-check pattern; migration changes to `stored, ok := ...; if !ok`.

Implementation note from ARCH-04 §F-005 Ruling:
- `LookupByPubkey` delegates to `Lookup` — both change together
- Callers (develop head, post-S-6.06 merge): `SVTNManager.ExpireKey` (line 352), `SVTNManager.CallerKeyRole` (line 404), `SVTNManager.CallerKeyRoleActive` (line 425), `SVTNManager.IsRegisteredAnyState` (line 526)
- `SVTNManager.RevokeKey` does NOT call `LookupByPubkey` — it was refactored to `RevokeKeyIfRoleMatches` per ADR-004 Addendum H2 (ARCH-04 v1.13); the original story stub predated S-6.06
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
| 1.1 | 2026-07-01 | Pass-1 lens-3 fix-burst: F-P1L3-001 — AC-1 signature corrected to `Lookup(svtnID [16]byte, nodeAddr [8]byte)` and `LookupByPubkey(svtnID [16]byte, pubkey ed25519.PublicKey)`; F-P1L3-002 — frontmatter `status: backlog→draft`, `wave: unscheduled→6` per sprint-state.yaml lines 41-58 and STORY-INDEX line 77; F-P1L3-003 — callsite list refreshed vs develop head post-S-6.06: RevokeKey superseded by RevokeKeyIfRoleMatches (ADR-004 H2), 4 actual callsites enumerated (ExpireKey, CallerKeyRole, CallerKeyRoleActive, IsRegisteredAnyState); F-P1L3-006 — stub parenthetical "(To be filled by story-writer...)" removed. |
| 1.0 | 2026-06-29 | Minted per S-6.02 lens3 F-006 ([process-gap]): DRIFT-F005-LOOKUP-CONVENTION has no draft story stub. Closes task #66. |
