---
artifact_id: S-BL.LOOKUP
document_type: story
level: ops
version: "1.0"
status: backlog
producer: product-owner
timestamp: 2026-06-29T00:00:00
phase: 2
cycle: v1.0.0-greenfield
epic: E-6
wave: unscheduled
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
- Callers: `SVTNManager.RevokeKey` and `SVTNManager.ExpireKey` (S-6.02 worktree)
- `ed25519.PublicKey` is `[]byte`; struct copy shares backing array — deep-clone still required: `cp.PublicKey = append([]byte(nil), src.PublicKey...)`
- One-line change per call site, no logic change

## Acceptance Criteria

(To be filled by story-writer when this stub is promoted to a wave story.)

- Migrate `AdmittedKeySet.Lookup(id string) (AdmittedKey, bool)` from pointer return
- Migrate `AdmittedKeySet.LookupByPubkey(pub ed25519.PublicKey) (AdmittedKey, bool)` from pointer return
- Update all call sites (at minimum: `SVTNManager.RevokeKey`, `SVTNManager.ExpireKey`) to use `stored, ok := ...; if !ok`
- Deep-clone `PublicKey` field in the value copy
- `go test -race ./internal/svtnmgmt/...` passes with no race detector findings
- `just lint` passes with zero warnings

## Earliest Wave

Wave 6+ (unblocked after S-6.02 merges to develop).

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-06-29 | Minted per S-6.02 lens3 F-006 ([process-gap]): DRIFT-F005-LOOKUP-CONVENTION has no draft story stub. Closes task #66. |
