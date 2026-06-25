---
artifact_id: BC-2.05.008
document_type: behavioral-contract
level: L3
version: "1.0"
status: draft
producer: product-owner
timestamp: 2026-06-25T00:00:00
phase: 1a
bc_id: BC-2.05.008
subsystem: SS-05
architecture_module: internal/routing
capability: CAP-020
priority: P0
criticality: critical
scope_phase: E
origin: greenfield
lifecycle_status: active
introduced: v0.3.0
modified: []
deprecated: null
deprecated_by: null
replacement: null
retired: null
removed: null
removal_reason: null
inputDocuments:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/domain-spec/failure-modes.md'
  - '.factory/specs/behavioral-contracts/ss-05/BC-2.05.005.md'
  - 'internal/routing/routing.go'
traces_to: [CAP-020]
kos_anchors:
  - elem-ssh-end-to-end-encryption
  - elem-node-router-architecture
---

# BC-2.05.008: RouteFrame Wire-Layer HMAC Enforcement (Fail-Closed for Writes)

## Description

`RouteFrame` in `internal/routing` MUST call `verifyFrameHMAC` against the sending node's `FrameAuthKey` (retrieved from the forwarding-table entry) BEFORE the admitted-set check and BEFORE `SVTNRoute`. Any frame whose HMAC tag does not match is rejected with `ErrHMACVerificationFailed` — a sentinel defined in `internal/routing` that maps to E-ADM-002 in the error taxonomy. The verification is fail-closed: if the forwarding-table entry is absent (so the auth key is unavailable), the frame is also rejected as unverifiable.

This BC specifies the **wire integration** of the HMAC primitive (BC-2.05.005 / `internal/hmac`) into the router dispatch path (`internal/routing`). BC-2.05.005 defines the crypto contract; this BC defines where and when that contract is enforced in the call stack.

Note: `verifyFrameHMAC` currently carries `//nolint:unused` in `internal/routing/routing.go` (line 153 on develop). The wire-up implemented by S-3.04 satisfies this BC's enforcement obligation and removes that nolint annotation.

## Preconditions

1. A frame has arrived at the router; `RouteFrame` has been called with the frame's outer header and payload.
2. The forwarding table contains an entry for `(hdr.SVTNID, hdr.SrcAddr)` — specifically, the entry's `FrameAuthKey` is the per-(node, SVTN) key derived per ADR-001 (HKDF-SHA256, info=`switchboard-frame-auth`).
3. `hdr.HMACTag` carries the 8-byte truncated HMAC-SHA256 tag placed there by the sending node (per BC-2.05.005).

## Postconditions

1. **Valid HMAC:** `verifyFrameHMAC(hdr, payload, entry.FrameAuthKey)` returns `true` → execution continues to the admitted-set check (`r.admittedKeySet.IsAdmitted`) and then to `SVTNRoute`. The frame is processed normally.
2. **Invalid HMAC:** `verifyFrameHMAC(hdr, payload, entry.FrameAuthKey)` returns `false` → `RouteFrame` returns `ErrHMACVerificationFailed` immediately. The frame is dropped. E-ADM-002 "HMAC verification failed: SVTN `<svtn_id>`, src `<src_addr>`, type `<frame_type>`" is logged at the router before return.
3. **HMAC verification occurs BEFORE the admitted-set check.** The ordering is: HMAC first → admitted-set second → SVTNRoute third. This ensures forged frames are rejected before touching the admitted-set data structure (fail-fast on forgery).
4. **Auth key unavailable (no forwarding-table entry for src):** `RouteFrame` returns `ErrHMACVerificationFailed` — the frame is treated as unverifiable and is dropped. Rationale: a frame from a node with no forwarding-table entry has no derivable auth key; admitting such a frame would bypass HMAC verification entirely.

## Invariants

1. **DI-006**: Every frame carrying SVTN-scoped traffic is HMAC-verified before forwarding. No frame reaches `SVTNRoute` without a successful `verifyFrameHMAC` call. This invariant is enforceable by code audit (VP-058).
2. `ErrHMACVerificationFailed` is distinct from `admission.ErrNotAdmitted` (E-ADM-003). Callers use `errors.Is` to distinguish forgery rejection from admission rejection.
3. The HMAC tag wire value is read BEFORE the outer header fields are zeroed for MAC computation — this prevents the tautological-verify defect (already guarded in `verifyFrameHMAC` implementation).
4. Verification uses `hmac.VerifyHMAC` (constant-time comparison) — no timing oracle.

## Trigger

Frame arrival at the router; `RouteFrame` entry point.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Frame with `HMACTag` field all-zeros | `verifyFrameHMAC` returns false; `ErrHMACVerificationFailed`; E-ADM-002 logged. |
| EC-002 | Frame with HMAC computed under a different node's key (cross-node forgery attempt) | `verifyFrameHMAC` returns false; same rejection path as EC-001. |
| EC-003 | Frame from a node that IS in the admitted set but whose forwarding-table entry has been purged (auth key unavailable) | No auth key → frame treated as unverifiable → `ErrHMACVerificationFailed`. The admitted-set check is never reached. |
| EC-004 | Empty-tick frame (zero-length payload) with correct HMAC | `verifyFrameHMAC` returns true; frame proceeds normally. HMAC over empty payload is valid per BC-2.05.005 EC-004. |
| EC-005 | Frame with correct HMAC but from a node not in the admitted set | HMAC passes (postcondition 1); admitted-set check then rejects with `admission.ErrNotAdmitted` (E-ADM-003). Two distinct errors for two distinct conditions. |
| EC-006 | Repeated HMAC failures (≥5 in 60 s) from same `src_addr` | Failure counter incremented per existing alert logic (BC-2.05.005 postcondition 3); admission alert triggered. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Frame with valid HMAC, admitted src, forwarding entry present | Frame forwarded; no error | happy-path |
| Frame with all-zero HMAC tag | `ErrHMACVerificationFailed` returned; E-ADM-002 logged | error |
| Frame with HMAC computed under wrong key | `ErrHMACVerificationFailed` returned; E-ADM-002 logged | error |
| Frame from admitted node, no forwarding-table entry for src | `ErrHMACVerificationFailed` returned (auth key unavailable) | edge-case |
| Empty-tick frame with correct HMAC, admitted src | Forwarded normally | happy-path |
| Frame with valid HMAC, src NOT in admitted set | `admission.ErrNotAdmitted` returned (E-ADM-003); HMAC check passes but admission fails | edge-case |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-058 | `RouteFrame` calls `verifyFrameHMAC` before `IsAdmitted` and `SVTNRoute` | code-audit / proptest |
| VP-004, VP-005, VP-006 | `verifyFrameHMAC` rejects forged tags (HMAC primitive correctness — from BC-2.05.005) | proptest / fuzz |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-020 ("HMAC frame authentication at router boundary") per capabilities.go §CAP-020 |
| L2 Domain Invariants | DI-006 (HMAC frame authentication at first router — every frame verified before forwarding) |
| Architecture Module | internal/routing |
| Stories | S-3.04 (Wave 3) |
| Architecture Decision | ADR-001 (amended): frame_auth_key derived per (node_admission_pubkey, svtn_id) via HKDF-SHA256 |
| Error Sentinel | `ErrHMACVerificationFailed` in `internal/routing`; maps to E-ADM-002 (log event, dropped frame) |
| Capability Anchor Justification | CAP-020 ("HMAC frame authentication at router boundary") per capabilities.md §CAP-020 — this BC specifies the wire integration point where CAP-020's enforcement obligation lands: `RouteFrame` in `internal/routing` must call the HMAC primitive (BC-2.05.005 / `internal/hmac`) and reject on failure before forwarding |

## Related BCs

- BC-2.05.005 — composes with: HMAC crypto primitive; this BC specifies where that primitive is called in the routing dispatch path
- BC-2.05.002 — composes with: admitted-set check occurs after HMAC verification; both are preconditions to forwarding
- BC-2.05.006 — composes with: SVTN isolation enforced in SVTNRoute after HMAC and admission pass

## Architecture Anchors

- ARCH-04 §HMAC keying — FrameAuthKey derivation
- ARCH-04 §SVTN Cryptographic Isolation — ordering of enforcement checks at the router
- ARCH-09 v1.1 §boundary classification — `internal/routing` is a boundary package

## Story Anchor

S-3.04 (Wave 3) — wires `verifyFrameHMAC` into `RouteFrame`; removes `//nolint:unused` annotation from routing.go:153

## VP Anchors

VP-058 (new — RouteFrame calls verifyFrameHMAC before IsAdmitted and SVTNRoute)
