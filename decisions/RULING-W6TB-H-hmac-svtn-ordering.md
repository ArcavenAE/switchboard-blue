---
artifact_id: RULING-W6TB-H-hmac-svtn-ordering
document_type: decision
level: ops
version: "1.0"
status: final
producer: product-owner
timestamp: 2026-07-01T00:00:00
updated: 2026-07-01T00:00:00
cycle: v1.0.0-greenfield
stories_in_scope: [S-7.02]
closes_findings: [M-3]
referenced_by:
  - .factory/stories/S-7.02-session-discovery.md
  - .factory/specs/behavioral-contracts/ss-03/BC-2.03.001.md
  - .factory/decisions/RULING-W6TB-D-discovery-scope.md
  - .factory/decisions/wave-6-tranche-a-scope-rulings.md
---

# Ruling W6TB-H — S-7.02 HMAC-vs-SVTN Check Ordering in ReceiveAdvertisement

**Adjudicator:** product-owner
**Date:** 2026-07-01
**Trigger:** S-7.02 Pass-3 L1 finding M-3 (MEDIUM)

---

## Finding Summary

`ReceiveAdvertisement` in
`.worktrees/S-7.02/internal/discovery/discovery.go` lines 250–263 processes
advertisement frames in this order:

1. Extract wire HMAC tag from the first 8 bytes.
2. Decode body (without HMAC verification) to read the `SVTN_ID` field.
3. Check `payload.SVTNID != d.cfg.LocalSVTNID` → return `ErrSVTNMismatch`.
4. Verify HMAC → return `ErrInvalidHMACTag` on failure.

The ordering is: **decode body → SVTN check → HMAC verify**.

BC-2.03.001 PC-5 states: "advertisement frames are **authenticated** (HMAC in
outer header) so receivers can verify it is from an admitted node." This implies
a fail-closed posture: HMAC verification is the authentication gate, and
processing of unauthenticated content before that gate is reached constitutes a
pre-authentication information leak.

Under the current ordering, an unauthenticated attacker can forge the SVTN bytes
(the first field decoded from the body) to force `ErrSVTNMismatch` before HMAC
verification runs. The attacker learns that the receiver is on a specific SVTN
(by observing that non-matching SVTN bytes produce `ErrSVTNMismatch` rather than
`ErrInvalidHMACTag`). With `advertisementKey(svtnID) = svtnID`
(DRIFT-W6TBD-001 scoping placeholder), both error paths are functionally
identical in key derivation — a "wrong SVTN" and "wrong HMAC" are the same
cryptographic failure — but the error code returned differs, leaking a
distinguishing oracle.

---

## Options Considered

### Option A — Flip to HMAC-first

Change `ReceiveAdvertisement` to:

1. Extract wire HMAC tag from first 8 bytes.
2. Derive HMAC key from the **declared** SVTN ID in the first field of the
   encoded body (read the SVTN ID bytes without full decode, or use a fixed key
   derivation over the raw body).
3. Verify HMAC → return `ErrInvalidHMACTag` on failure.
4. Decode body fully.
5. Check `payload.SVTNID != d.cfg.LocalSVTNID` → return `ErrSVTNMismatch`.

Both sentinel errors are preserved for legitimate callers (cross-scope
advertisements from admitted nodes on other SVTNs will fail the HMAC check since
they use a different key). The attacker who forges SVTN bytes receives only
`ErrInvalidHMACTag` regardless of what SVTN bytes they write, because HMAC
verification runs first with the full body.

Pros: strictly fail-closed; aligns with BC-2.03.001 PC-5 "authenticated"
posture; one reorder of ~11 lines; all sentinels preserved.

Cons: the HMAC key derivation must occur before full body decode. Under the
current DRIFT-W6TBD-001 scoping placeholder (`advertisementKey(svtnID) =
svtnID`), the key is derived from the raw SVTN ID bytes in the body — which are
available before full struct decode via a partial read.

### Option B — Keep SVTN-first, amend BC-2.03.001 PC-5

Add a note to BC-2.03.001 PC-5 permitting SVTN dispatch before HMAC under the
scope-split rationale of RULING-W6TB-D: "in the unit-scope registry model
(S-7.02), the receiver MAY perform SVTN-scope dispatch before HMAC verification
because the in-process model has no unauthenticated-attacker threat model."

Pros: no code change; no partial-decode complexity.

Cons: weakens the fail-closed posture specified in PC-5 for all future
implementations including S-BL.DISCOVERY-WIRE, where real network attackers
exist. Embedding a scope-split exception in a BC postcondition creates technical
debt that must be explicitly removed when multicast wire is implemented. The
leaking oracle (distinguishable `ErrSVTNMismatch` vs. `ErrInvalidHMACTag`) is
observable today and does not require multicast for exploitation in a future
integration test environment.

### Option C — Defer to S-BL.DISCOVERY-WIRE, add DRIFT note

Accept current ordering as a known defect, add a `DRIFT-W6TBH-001` note to
`ReceiveAdvertisement` and to AC-005, and require S-BL.DISCOVERY-WIRE to correct
the ordering when implementing real multicast.

Pros: no code change now.

Cons: the reorder is one function's 11-line restructuring, not a multi-file
cross-boundary change. Deferring accumulates a known security posture defect into
every subsequent adversarial pass until S-BL.DISCOVERY-WIRE lands. Option C is
the same reasoning that was correctly rejected for Ruling-14 (dispatch() bounded
read symmetry): a narrow, low-risk fix should not be deferred.

---

## Decision: Option A

**Ruling: Option A — flip to HMAC-first. Apply as part of S-7.02 fix-burst.
Story version bumps to v1.3 (combined with RULING-W6TB-G). Document in this
ruling.**

### Rationale

**Option B is rejected.** Embedding a scope-split exception in BC-2.03.001 PC-5
is the wrong level of abstraction. The fail-closed ordering is a security
invariant, not a transport-layer detail. RULING-W6TB-D already established the
correct scope boundary: the in-process registry model is an implementation scope
constraint, not a license to relax authentication ordering. Admitting SVTN-first
processing into the BC would require future spec surgery to remove it before
S-BL.DISCOVERY-WIRE can be written correctly.

**Option C is rejected.** The reorder is low-cost (11 lines restructured in one
function) and zero-risk (same sentinels, same behavior for legitimate callers,
only the attacker's distinguishing oracle closes). Deferring a minimal fix with
a clear security rationale sets a poor precedent given the project's established
pattern (Rulings 14, 5, 7) of taking the narrow fix over the drift entry.

**Option A is correct and minimal.** The key point: with
`advertisementKey(svtnID) = svtnID`, deriving the HMAC key requires knowing the
SVTN ID — which is encoded in the body. The current decode-first approach reads
the SVTN ID to derive the key, then verifies. HMAC-first does the same derivation
in the same place; it simply moves the `VerifyAdvertisementHMAC` call before the
`payload.SVTNID != d.cfg.LocalSVTNID` guard. No information is lost. The SVTN
mismatch path still returns `ErrSVTNMismatch` for admitted-node cross-scope
advertisements (which pass HMAC with their own SVTN's key, then fail the local
SVTN check). The `ErrInvalidHMACTag` path covers all unauthenticated frames.

The result: an unauthenticated attacker who forges SVTN bytes receives
`ErrInvalidHMACTag` (HMAC fails because body is forged). An admitted node on a
different SVTN receives `ErrSVTNMismatch` (HMAC passes for its own SVTN key, but
local SVTN check rejects). Both sentinel codes remain available; neither leaks
before authentication.

### Production Code Change Specification

In `internal/discovery/discovery.go`, `ReceiveAdvertisement`, restructure the
body from the current order:

```
// CURRENT (SVTN-first — REJECTED by RULING-W6TB-H)
1. extract wireTag from raw[:AdvertisementHMACTagSize]
2. body = raw[AdvertisementHMACTagSize:]
3. payload, err = decodeBody(body)  // unauthenticated decode
4. if err != nil { return ErrInvalidHMACTag }
5. if payload.SVTNID != d.cfg.LocalSVTNID { return ErrSVTNMismatch }
6. hmacKey = advertisementKey(d.cfg.LocalSVTNID)
7. if !VerifyAdvertisementHMAC(hmacKey[:], body, wireTag) { return ErrInvalidHMACTag }
8. store sessions
```

To the HMAC-first order:

```
// REQUIRED (HMAC-first — RULING-W6TB-H)
1. extract wireTag from raw[:AdvertisementHMACTagSize]
2. body = raw[AdvertisementHMACTagSize:]
3. payload, err = decodeBody(body)        // minimal decode to read SVTN ID for key derivation
4. if err != nil { return ErrInvalidHMACTag }
5. hmacKey = advertisementKey(payload.SVTNID)  // key derived from declared SVTN ID
6. if !VerifyAdvertisementHMAC(hmacKey[:], body, wireTag) { return ErrInvalidHMACTag }
7. if payload.SVTNID != d.cfg.LocalSVTNID { return ErrSVTNMismatch }
8. store sessions
```

Key observations:

- Step 3 (`decodeBody`) is still called before HMAC. This is acceptable because
  the decode is needed to read the declared SVTN ID for key derivation. The decode
  result is not trusted until after HMAC passes — the decoded `payload.SVTNID` is
  used only to derive the key (step 5), not to make any security decision.
  After HMAC passes (step 6), the full payload is trusted and the SVTN check
  (step 7) compares the now-authenticated SVTN ID against the local SVTN.

- `advertisementKey` now takes `payload.SVTNID` (the declared SVTN) rather than
  `d.cfg.LocalSVTNID`. Under DRIFT-W6TBD-001 (`advertisementKey(x) = x`), this
  is identical when `payload.SVTNID == d.cfg.LocalSVTNID`. For a cross-SVTN
  admitted node, it uses the node's own SVTN key — which will successfully verify
  the node's own advertisement, then fail the local SVTN check in step 7.
  This is the correct behavior: cross-SVTN admitted nodes are authenticated but
  out-of-scope.

- The forged-SVTN-bytes attacker (finding M-3) now receives `ErrInvalidHMACTag`
  at step 6 regardless of what SVTN bytes they wrote, because they cannot forge
  a valid HMAC for any key (they don't know any admitted-node key material). The
  distinguishing oracle is closed.

The comment block at the old step 5 ("SVTN cross-scope isolation check BEFORE
HMAC") MUST be removed. The new step 7 gets:
```go
// SVTN cross-scope check (RULING-W6TB-H): payload is authenticated; verify
// it belongs to our SVTN. Admitted nodes on other SVTNs pass HMAC (their own
// key) but fail here — ErrSVTNMismatch is returned for legitimate cross-scope
// rejection, not as a pre-authentication oracle.
```

### AC-005 Clarification (S-7.02 v1.3)

Append to AC-005 acceptance criterion text:

> **Ordering note (RULING-W6TB-H):** `ReceiveAdvertisement` MUST verify HMAC
> before the SVTN cross-scope check (`payload.SVTNID != LocalSVTNID`). The HMAC
> key is derived from the declared `payload.SVTNID` (not from `LocalSVTNID`) so
> that admitted nodes on other SVTNs are authenticated with their own key and then
> correctly rejected by the SVTN check. Unauthenticated frames MUST return
> `ErrInvalidHMACTag` before any SVTN comparison occurs.

### Test Verification

`TestDiscovery_Advertise_HMACAuthenticated` (AC-005) already asserts that a
frame with a wrong HMAC tag is rejected with `ErrInvalidHMACTag`. No test change
is required for this ruling: the existing test covers the post-flip behavior
identically (wrong tag → `ErrInvalidHMACTag`). The ordering change is an
implementation invariant, not a behavior change observable from outside the
function in the existing test vectors.

If the test-writer wishes to add a distinguishing oracle test verifying that a
forged-SVTN frame returns `ErrInvalidHMACTag` (not `ErrSVTNMismatch`), that
would be a strengthening beyond the current test set. It is RECOMMENDED but not
required for this fix-burst.

---

## BC-2.03.001 Note (No Version Bump Required)

RULING-W6TB-D already annotated PC-5 with the key-placeholder note
(DRIFT-W6TBD-001). No additional BC amendment is required for RULING-W6TB-H: the
ordering fix aligns the implementation to the "authenticated" language already
in PC-5. The BC does not need to describe internal implementation order; it
specifies that HMAC authentication occurs (which it does, before the SVTN check,
after the fix).

---

## S-7.02 Story Delta (combined with RULING-W6TB-G: v1.2 → v1.3)

The v1.3 version bump is shared with RULING-W6TB-G. The single combined
changelog entry covers both rulings:

```
| v1.3 | 2026-07-01 | product-owner | RULING-W6TB-G + RULING-W6TB-H: (G) AC-001b
oracle split: ExactN deterministic test + Config.TickSource seam; (H) AC-005
ordering note: ReceiveAdvertisement MUST verify HMAC before SVTN cross-scope
check; key derived from payload.SVTNID; forged-SVTN oracle closed. |
```

### Frontmatter delta (S-7.02)

| Field | v1.2 | v1.3 |
|-------|------|------|
| `version` | `"1.2"` | `"1.3"` |
| `changed_by_rulings` | `[RULING-W6TB-D]` | `[RULING-W6TB-D, RULING-W6TB-G, RULING-W6TB-H]` |

---

## Downstream Dispatch Table

| Artifact | Change | Agent | When |
|----------|--------|-------|------|
| `.factory/stories/S-7.02-session-discovery.md` | AC-005 ordering note + v1.2→v1.3 + changelog (combined with RULING-W6TB-G) | story-writer | Same burst as ruling |
| `wave-6-tranche-a-scope-rulings.md` | Add §12 changelog entry for RULING-W6TB-G + RULING-W6TB-H | spec-steward | Same burst as ruling |
| `internal/discovery/discovery.go` (worktree) | Flip HMAC-first order in `ReceiveAdvertisement`; update `advertisementKey` call to use `payload.SVTNID`; replace old ordering comment with new RULING-W6TB-H comment | implementer | S-7.02 fix-burst |

---

## Decision Log

| Date | Actor | Entry |
|------|-------|-------|
| 2026-07-01 | product-owner | Option A adopted. HMAC-first ordering is the correct fail-closed posture per BC-2.03.001 PC-5. Cost is minimal (reorder 11 lines, update one `advertisementKey` argument). Option B rejected: spec retreat on a security invariant. Option C rejected: precedent (Rulings 7, 14) is narrow fix, not drift entry. Key derivation change (`d.cfg.LocalSVTNID` → `payload.SVTNID`) is deliberate: it closes the forged-SVTN distinguishing oracle while preserving `ErrSVTNMismatch` for legitimate cross-SVTN admitted nodes. |
