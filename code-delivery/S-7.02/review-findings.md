# S-7.02 PR Review Convergence Tracking

**Story:** S-7.02 v1.6 — SVTN-Scoped Multicast Session Discovery
**PR:** #55
**Merge SHA:** c54a8ad06f3f0d253180645a7eeb4ac6216dc004
**Converged:** 2 review cycles

## Convergence Table

| Cycle | Findings | Blocking | Fixed in Cycle | Remaining Blocking |
|-------|----------|----------|----------------|--------------------|
| 1 | 12 (code-reviewer) + 15 (pr-reviewer) | 5 | 5 (commit a785945) | 0 |
| 2 | 4 nits (pr-reviewer) | 0 | 0 (all nits deferred) | 0 → APPROVE |

## Cycle 1 Blocking Findings — All Fixed in a785945

| ID | Source | Severity | Finding | Resolution |
|----|--------|----------|---------|------------|
| CR-001 | code-reviewer | HIGH | `Advertise` accepted invalid session names (empty/non-UTF-8) without validation | Added `encodedSessionName` call in `Advertise` before registry write |
| CR-005 | code-reviewer | HIGH | `append(b[:cut:cut], ellipsis...)` aliasing hazard in `encodedSessionName` | Replaced with explicit `make([]byte, 0, cut+len(ellipsis)) + append` |
| CR-012 / F-5 | both | MED/HIGH | `Advertise` doc comment falsely claimed HMAC signing and multicast transmission | Corrected doc comment; added `NOTE(S-BL.DISCOVERY-WIRE)` deferral annotation |
| F-1 / CR-009 | both | HIGH | `ReceiveAdvertisement` and `Decode` min-length guard checked 8 bytes but doc said 34 | Raised guard to `AdvertisementHMACTagSize+26` = 34 bytes in both functions |
| F-3 / SEC-003 | pr-reviewer | HIGH (elevated from LOW) | `decodeBody` had no upper bound on session count from wire (up to 65535) | Added `maxSessionsPerAdvertisement=1024` constant and guard before loop |

## Cycle 2 Non-Blocking Nits (Deferred)

| ID | Source | Severity | Finding | Disposition |
|----|--------|----------|---------|-------------|
| N-1 | pr-reviewer | NIT | `validatedSession.name` redundant field | Deferred — cosmetic |
| N-2 | pr-reviewer | NIT | `encodedSessionName` aliasing comment slightly off | Deferred — cosmetic |
| N-3 | pr-reviewer | SUGGESTION | `decodeBody` errors not sentinel vars | Deferred to follow-up |
| N-4 | pr-reviewer | NIT | `HeartbeatInterval` const/field name ambiguity | Deferred — cosmetic |

## Deferred Findings (from both cycles)

| ID | Source | Severity | Finding | Deferred To |
|----|--------|----------|---------|-------------|
| F-2 | pr-reviewer | HIGH | `ReceiveAdvertisement` exported; HMAC key = SVTN ID bypassable | S-BL.DISCOVERY-WIRE (DRIFT-W6TBD-001) |
| CR-007 | code-reviewer | HIGH | gopter as external test dependency | APPROVED — gopter is in go.mod, used in prior stories; "no testify" rule targets assertion frameworks |
| SEC-001 | security-reviewer | MEDIUM | `advertisementKey(svtnID)=svtnID` weak HMAC key | S-BL.DISCOVERY-WIRE (DRIFT-W6TBD-001) |
| SEC-002 / F-8 | security+pr | LOW | Enum bytes not range-checked in `decodeBody` | Pre-S-BL.DISCOVERY-WIRE cleanup story |
| SEC-004 / F-9 | security+pr | LOW | Non-UTF-8 names accepted in `decodeBody` from wire | Pre-S-BL.DISCOVERY-WIRE cleanup story |
| SEC-005 | security-reviewer | LOW | `Advertise` previously lacked session name validation | FIXED in a785945 (CR-001) |
| CR-002 / F-10 | both | MED/LOW | ctx cancellation not checked in `Advertise`/`Enumerate` | Deferred — in-process sync path |
| CR-004 | code-reviewer | MED | `advertisementKey` naming implies derivation | Deferred |
| CR-003/006/008/010/011 | code-reviewer | LOW/NIT | Various style/robustness notes | Deferred |
| F-6/7/11..15 | pr-reviewer | LOW/NIT | Various robustness/clarity notes | Deferred |
