---
artifact_id: BC-2.05.008
document_type: behavioral-contract
level: L3
version: "1.3"
status: draft
producer: product-owner
timestamp: 2026-06-27T00:00:00
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
modified:
  - '2026-06-27: v1.2 — EC-006 replaced with concrete mechanism spec: RouteFrame calls admission.RecordHMACFailure(srcAddr) on each ErrHMACVerificationFailed; cross-reference to BC-2.05.005 PC-3 admission-layer contract; FIX-NOW per Wave 3 gate F-2 adjudication'
  - '2026-06-27: v1.3 — per-story adversarial convergence adjudication: (M-2) ratify hmacFailureRecorder interface seam; amend PC-5, EC-006, and injection description; (item-3) ratify src_addr hex rendering; update E-ADM-016/E-ADM-017 message format to embed code literal; version bump'
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

`RouteFrame` in `internal/routing` MUST call `verifyFrameHMAC` against the sending node's `FrameAuthKey` (retrieved from the forwarding-table entry) BEFORE the admitted-set check and BEFORE `SVTNRoute`. Any frame whose HMAC tag does not match is rejected with `ErrHMACVerificationFailed` — a sentinel defined in `internal/routing` that maps to E-ADM-016 in the error taxonomy. The verification is fail-closed: if the forwarding-table entry is absent (so the auth key is unavailable), the frame is also rejected as unverifiable.

This BC specifies the **wire integration** of the HMAC primitive (BC-2.05.005 / `internal/hmac`) into the router dispatch path (`internal/routing`). BC-2.05.005 defines the crypto contract; this BC defines where and when that contract is enforced in the call stack.

Note: `verifyFrameHMAC` currently carries `//nolint:unused` in `internal/routing/routing.go` (line 153 on develop). The wire-up implemented by S-3.04 satisfies this BC's enforcement obligation and removes that nolint annotation.

## Preconditions

1. A frame has arrived at the router; `RouteFrame` has been called with the frame's outer header and payload.
2. The forwarding table contains an entry for `(hdr.SVTNID, hdr.SrcAddr)` — specifically, the entry's `FrameAuthKey` is the per-(node, SVTN) key derived per ADR-001 (HKDF-SHA256, info=`switchboard-frame-auth`).
3. `hdr.HMACTag` carries the 8-byte truncated HMAC-SHA256 tag placed there by the sending node (per BC-2.05.005).

## Postconditions

1. **Valid HMAC:** `verifyFrameHMAC(hdr, payload, entry.FrameAuthKey)` returns `true` → execution continues to the admitted-set check (`r.admittedKeySet.IsAdmitted`) and then to `SVTNRoute`. The frame is processed normally.
2. **Invalid HMAC:** `verifyFrameHMAC(hdr, payload, entry.FrameAuthKey)` returns `false` → `RouteFrame` returns `ErrHMACVerificationFailed` immediately. The frame is dropped. E-ADM-016 "wire HMAC verification failed at RouteFrame: tag mismatch for SVTN `<svtn_id>` from src `<src_addr>`" is logged at the router before return.
3. **HMAC verification occurs BEFORE the admitted-set check.** The ordering is: HMAC first → admitted-set second → SVTNRoute third. This ensures forged frames are rejected before touching the admitted-set data structure (fail-fast on forgery).
4. **Auth key unavailable (no forwarding-table entry for src):** `RouteFrame` returns `ErrHMACVerificationFailed` — the frame is treated as unverifiable and is dropped. Rationale: a frame from a node with no forwarding-table entry has no derivable auth key; admitting such a frame would bypass HMAC verification entirely.
5. **Failure event forwarded to admission layer (per BC-2.05.005 PC-3):** On every `ErrHMACVerificationFailed` return path (postconditions 2 and 4), `RouteFrame` calls `router.failureCounter.RecordHMACFailure(srcAddrHex)` BEFORE returning, where `srcAddrHex` is the lowercase hex encoding of the 8-byte `hdr.SrcAddr` field (e.g. `fmt.Sprintf("%x", hdr.SrcAddr)`). This call is the seam between the routing layer and the admission-layer aggregate-alert mechanism. No call is made on a successful HMAC verification (postcondition 1). `RouteFrame` does NOT evaluate the threshold or emit the alert — that is entirely the responsibility of the `hmacFailureRecorder` implementation (see BC-2.05.005 PC-3 for full contract). The `failureCounter` field is typed as the unexported `hmacFailureRecorder` interface (`RecordHMACFailure(string)`); `*admission.FailureCounter` is the production implementation injected via `WithFailureCounter`. Test doubles may inject any satisfying type.

## Invariants

1. **DI-006**: Every frame carrying SVTN-scoped traffic is HMAC-verified before forwarding. No frame reaches `SVTNRoute` without a successful `verifyFrameHMAC` call. This invariant is enforceable by code audit (VP-058).
2. `ErrHMACVerificationFailed` is distinct from `admission.ErrNotAdmitted` (E-ADM-003). Callers use `errors.Is` to distinguish forgery rejection from admission rejection.
3. The HMAC tag wire value is read BEFORE the outer header fields are zeroed for MAC computation — this prevents the tautological-verify defect (already guarded in `verifyFrameHMAC` implementation).
4. Verification uses `hmac.VerifyHMAC` (constant-time comparison) — no timing oracle.
5. `RecordHMACFailure` is called on ALL `ErrHMACVerificationFailed` paths — including when the forwarding-table entry is absent (postcondition 4). The absence of an auth key is itself a potential forgery signal; the counter counts it.

## Trigger

Frame arrival at the router; `RouteFrame` entry point.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Frame with `HMACTag` field all-zeros | `verifyFrameHMAC` returns false; `ErrHMACVerificationFailed`; E-ADM-016 logged. |
| EC-002 | Frame with HMAC computed under a different node's key (cross-node forgery attempt) | `verifyFrameHMAC` returns false; same rejection path as EC-001 (E-ADM-016 logged). |
| EC-003 | Frame from a node that IS in the admitted set but whose forwarding-table entry has been purged (auth key unavailable) | No auth key → frame treated as unverifiable → `ErrHMACVerificationFailed`. The admitted-set check is never reached. |
| EC-004 | Empty-tick frame (zero-length payload) with correct HMAC | `verifyFrameHMAC` returns true; frame proceeds normally. HMAC over empty payload is valid per BC-2.05.005 EC-004. |
| EC-005 | Frame with correct HMAC but from a node not in the admitted set | HMAC passes (postcondition 1); admitted-set check then rejects with `admission.ErrNotAdmitted` (E-ADM-003). Two distinct errors for two distinct conditions. |
| EC-006 | Repeated HMAC failures (≥5 in 60 s) from same `src_addr` | `RouteFrame` calls `router.failureCounter.RecordHMACFailure(fmt.Sprintf("%x", hdr.SrcAddr))` immediately before returning `ErrHMACVerificationFailed`. The `failureCounter` field is typed as `hmacFailureRecorder` (unexported interface: `RecordHMACFailure(string)`); `*admission.FailureCounter` is the production implementation injected via `WithFailureCounter`. Tests may inject a fake satisfying the same interface. When `RecordHMACFailure` detects that the sliding-window count for the srcAddr hex string has reached ≥5 within 60 seconds, it emits E-ADM-017 ("E-ADM-017 HMAC failure rate alert: ≥5 failures in 60s from src `<src_addr>`") at ERROR level. The message embeds the code literal for grep-ability. This mechanism is fully specified in BC-2.05.005 PC-3 (sliding window semantics, periodic-re-fire under sustained attack, concurrency contract, dead-key eviction, hard source cap). `RouteFrame` itself has no alert logic — it is a call-through. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Frame with valid HMAC, admitted src, forwarding entry present | Frame forwarded; no error | happy-path |
| Frame with all-zero HMAC tag | `ErrHMACVerificationFailed` returned; E-ADM-016 logged | error |
| Frame with HMAC computed under wrong key | `ErrHMACVerificationFailed` returned; E-ADM-016 logged | error |
| Frame from admitted node, no forwarding-table entry for src | `ErrHMACVerificationFailed` returned (auth key unavailable); `RecordHMACFailure` called | edge-case |
| Empty-tick frame with correct HMAC, admitted src | Forwarded normally; `RecordHMACFailure` NOT called | happy-path |
| Frame with valid HMAC, src NOT in admitted set | `admission.ErrNotAdmitted` returned (E-ADM-003); HMAC check passes but admission fails; `RecordHMACFailure` NOT called (HMAC passed) | edge-case |
| 5 consecutive HMAC failures from same src_addr within 60s | After 5th: E-ADM-017 emitted by `FailureCounter`; `RouteFrame` called `RecordHMACFailure` 5 times | alert-threshold |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-058 | `RouteFrame` calls `verifyFrameHMAC` before `IsAdmitted` and `SVTNRoute` | code-audit / proptest |
| VP-004, VP-005, VP-006 | `verifyFrameHMAC` rejects forged tags (HMAC primitive correctness — from BC-2.05.005) | proptest / fuzz |
| VP-059 | `FailureCounter.RecordHMACFailure` fires E-ADM-017 at exactly the 5th call within 60s and not before | proptest |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-020 ("HMAC frame authentication at router boundary") per capabilities.go §CAP-020 |
| L2 Domain Invariants | DI-006 (HMAC frame authentication at first router — every frame verified before forwarding) |
| Architecture Module | internal/routing |
| Stories | S-3.04 (Wave 3) |
| Architecture Decision | ADR-001 (amended): frame_auth_key derived per (node_admission_pubkey, svtn_id) via HKDF-SHA256 |
| Error Sentinel | `ErrHMACVerificationFailed` in `internal/routing`; maps to E-ADM-016 (wire-layer log event, dropped frame); E-ADM-002 covers HMAC primitive failure in `internal/hmac` (BC-2.05.005) |
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
