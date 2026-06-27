---
artifact_id: BC-2.05.005
document_type: behavioral-contract
level: L3
version: "1.3"
status: draft
producer: product-owner
timestamp: 2026-06-27T00:00:00
phase: 1a
bc_id: BC-2.05.005
subsystem: admission-security
architecture_module: internal/hmac
capability: CAP-020
priority: P0
criticality: critical
scope_phase: E
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified:
  - '2026-06-25: added Related BCs entry for BC-2.05.008 (Wave 3 wire-up)'
  - '2026-06-27: v1.3 — PC-3 made concrete and testable: specify admission-layer API (RecordHMACFailure), sliding-window semantics, threshold, alert event code E-ADM-017, concurrency contract; remove [DEFERRED] ambiguity; FIX-NOW per Wave 3 gate F-2 adjudication'
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
  - '_bmad-output/planning-artifacts/prd.md'
traces_to: [CAP-020]
kos_anchors:
  - elem-ssh-end-to-end-encryption
  - elem-node-router-architecture
---

# Behavioral Contract BC-2.05.005: HMAC Frame Authentication at First Router Boundary

## Description

Every SVTN-scoped frame carries an 8-byte HMAC tag in the outer header, computed by the sending node using its `frame_auth_key` (derived per `(node_admission_pubkey, svtn_id)` via HKDF-SHA256). The tag is the first 8 bytes of the 32-byte HMAC-SHA256 output, computed over the full frame (outer header bytes 0–35 || channel header || payload), with hmac_tag bytes treated as zeros during computation. See ARCH-02 §HMAC tag. The first router that receives the frame verifies the tag before forwarding. Frames with invalid tags are rejected before forwarding — fail-closed. This ensures every forwarded frame originated from an admitted node holding the expected private key. See ADR-001 (amended) for the HKDF derivation details.

## Preconditions

1. The sending node is admitted to the SVTN and has a valid admission key.
2. The `frame_auth_key` is derived per `(node_admission_pubkey, svtn_id)` via HKDF-SHA256 with info=`switchboard-frame-auth` and length=32 (see ADR-001 in ARCH-04 §HMAC keying). The HMAC tag is computed over the full frame (outer header bytes 0–35 || channel header || payload), with hmac_tag bytes treated as zeros during computation, using HMAC-SHA256 with `frame_auth_key`; the tag is the first 8 bytes of the 32-byte HMAC-SHA256 output.
3. The first router has the sending node's public key in its admitted key set.

## Postconditions

1. HMAC verification succeeds: frame forwarded to destination.
2. HMAC verification fails: frame dropped; E-ADM-002 "HMAC verification failed: <svtn_id>, <src_addr>, <frame_type>" logged at the router; the sending node receives no delivery confirmation.
3. **Per-source HMAC failure rate alert:** When `RouteFrame` returns `ErrHMACVerificationFailed` for a frame from `src_addr`, it MUST call `admission.RecordHMACFailure(srcAddr string)` on the router's `*admission.FailureCounter` before returning. The `FailureCounter` maintains a per-`src_addr` sliding-window counter over a 60-second window. When the count for a `src_addr` reaches or exceeds **5** within any trailing 60-second window, the `FailureCounter` emits a structured log event at ERROR level with code **E-ADM-017** ("HMAC failure rate alert: ≥5 failures in 60s from src `<src_addr>`"). The alert fires once per threshold crossing (i.e., on the 5th failure and again only if the count drops below 5 and then rises to 5 again — see EC-005 for hysteresis semantics). Subsequent failures within the same window do NOT re-emit the alert (fire-once-per-crossing).

   **Admission-layer API contract** (the seam the implementer builds — no Go code, but the contract is precise):
   - Type: `admission.FailureCounter` in `internal/admission`
   - Method: `RecordHMACFailure(srcAddr string)` — pure in-memory; takes no `context.Context` (no I/O; a mutex-guarded in-memory sliding window qualifies as pure-enough for this call path)
   - Constructor: `admission.NewFailureCounter(threshold int, windowDuration time.Duration, logger Logger) *FailureCounter` — logger is injected (dependency injection; no package-level global)
   - Internal state: a `map[string][]time.Time` of per-`src_addr` timestamp slices, guarded by `sync.Mutex`; entries are evicted lazily when a `RecordHMACFailure` call trims stale timestamps older than `windowDuration` (no background goroutine required; per go.md rule #12 the map entries are value-copied on read; no internal pointer to a live slice is returned to callers)
   - `RouteFrame` receives the `*admission.FailureCounter` via constructor injection on the `Router` struct; `internal/routing` imports `internal/admission` (position 4→5, consistent with ARCH-08 §6.5)
   - Window semantics: **sliding** (not fixed-bucket) — at each `RecordHMACFailure` call, timestamps older than `now - windowDuration` are trimmed; the count of remaining entries is compared against `threshold`

   **Concurrency contract:** `RecordHMACFailure` is safe for concurrent calls from multiple goroutines. The `sync.Mutex` in `FailureCounter` is held for the duration of the trim+append+check sequence. Per go.md rule #12: the slice of timestamps is never returned by reference to the caller; if a `Timestamps(srcAddr string) []time.Time` inspector is ever needed (e.g., for tests), it returns a copy.

## Invariants

1. **DI-006**: Every frame carrying SVTN-scoped traffic is verified against the admitted key set by the first router that receives it. No exceptions.
2. **DI-003**: HMAC authentication proves identity (admitted node) but does not protect content confidentiality at the router (content is SSH-encrypted separately).
3. The HMAC is recomputed fresh by the router for verification — there is no HMAC caching.

## Trigger

Frame arrival at the first router after transmission from the source node.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 (FM-006) | Frame arrives with non-member HMAC | E-ADM-002 logged; frame dropped silently (no rejection sent to source). |
| EC-002 | Frame corruption in transit causes HMAC mismatch | Same as non-member HMAC — E-ADM-002 logged; frame dropped. Sending node will retransmit. |
| EC-003 | Key rotation: node uses new key, router has old key | HMAC verification fails until router receives new key propagation. Node retransmits; after key propagation, HMAC succeeds. |
| EC-004 | Empty-tick frame has no payload | HMAC computed over outer header fields + zero-length payload. This is valid; verification proceeds normally. |
| EC-005 | Alert hysteresis: 6 failures in 60s, then 0 failures for 61s, then 5 more failures in 60s | First batch: E-ADM-017 fires on the 5th failure. After the window expires, the counter resets (stale entries trimmed). Second batch: E-ADM-017 fires again on the 5th failure of the new window. Exactly 2 alert events total. |
| EC-006 | Exactly 4 failures within 60s, no 5th failure | No E-ADM-017 emitted. Counter holds 4 timestamps; next trimmed-window check returns 4 < threshold. |
| EC-007 | Concurrent HMAC failures from two different src_addrs, each hitting ≥5 threshold | Each src_addr has its own counter slot. Both cross the threshold independently; two separate E-ADM-017 events emitted (one per src_addr). No interference between counters. |
| EC-008 | 5th failure arrives at exactly windowDuration after the 1st failure (boundary) | After trimming entries older than `now - windowDuration`, the 1st entry falls on the boundary. Implementation MUST treat the boundary as exclusive (trim if `age >= windowDuration`). If implementation trims the boundary entry, count is 4 and no alert; if it keeps it, count is 5 and alert fires. **Correct behavior: trim entries where `timestamp < now - windowDuration` (strictly less); boundary entry is kept; count = 5; alert fires.** |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Valid frame with correct HMAC | Frame forwarded; no log event | happy-path |
| Frame with HMAC computed with wrong key | E-ADM-002 logged; frame dropped | error |
| Frame with HMAC field all-zeros | E-ADM-002 logged; frame dropped | error |
| Empty-tick frame with correct HMAC | Frame forwarded normally | happy-path |
| 5 HMAC failures in 30s from same src_addr (RecordHMACFailure called 5 times) | E-ADM-017 emitted exactly once on the 5th call | alert-threshold |
| 4 HMAC failures in 60s from same src_addr, no 5th | No E-ADM-017 emitted | below-threshold |
| 5 HMAC failures from src_addr A + 5 from src_addr B, interleaved | E-ADM-017 emitted once for A and once for B, independently | multi-source |
| 5 HMAC failures, then 61s pause, then 5 more from same src_addr | Two E-ADM-017 events (one per window crossing), not one | hysteresis |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-004, VP-005, VP-006 | For all admitted nodes: frames with correct HMAC are forwarded | proptest |
| VP-004, VP-005, VP-006 | For all non-admitted sources: frames are dropped | proptest |
| VP-004, VP-005, VP-006 | HMAC covers outer header bytes 0–35 + channel header + payload | unit |
| VP-059 | FailureCounter.RecordHMACFailure fires E-ADM-017 at exactly ≥5 calls in 60s window and not before | proptest |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-020 ("HMAC frame authentication at router boundary") per capabilities.md §CAP-020 |
| L2 Domain Invariants | DI-006 (HMAC frame authentication at first router), DI-003 (router compromise → availability/quality, not content) |
| Architecture Module | internal/hmac (crypto primitive), internal/admission (FailureCounter type — PC-3), internal/routing (RouteFrame caller — BC-2.05.008) |
| Stories | S-2.01 (crypto primitive only); per-source alert obligation (PC-3) is FIX-NOW Wave 3 gate → new story S-W3.05 (see Story Skeleton section below) |
| Architecture Decision | ADR-001 (amended): frame_auth_key derived per (node_admission_pubkey, svtn_id) via HKDF-SHA256; tag = first 8 bytes of HMAC-SHA256 output |
| Error Codes | E-ADM-002 (per-failure primitive log), E-ADM-016 (per-failure wire-layer log at RouteFrame), E-ADM-017 (aggregate alert: ≥5 failures in 60s from same src_addr — PC-3) |
| Capability Anchor Justification | CAP-020 ("HMAC frame authentication at router boundary") per capabilities.md §CAP-020 — this BC is the direct behavioral specification of the HMAC verification that CAP-020 defines as "The first router verifies and rejects frames from non-admitted sources before forwarding" |

## Related BCs

- BC-2.05.002 — composes with: admitted-set check + HMAC together enforce the SVTN boundary
- BC-2.01.004 — depends on: HMAC field is in the outer header defined by BC-2.01.004
- BC-2.05.008 — composes with: wire-layer integration in RouteFrame (internal/routing); this BC defines the HMAC primitive and the FailureCounter API, BC-2.05.008 defines where both are called

## Story Skeleton (for story-writer — DO NOT expand body here)

Story-writer MUST produce the full body/ACs for this skeleton. Product-owner provides structure only.

```
Story ID:    S-W3.05
Title:       Per-source HMAC failure counter and admission alert (BC-2.05.005 PC-3)
Epic:        E-2 (Admission & Security)
Wave:        3 (FIX-NOW — Wave 3 gate blocker F-2)
Points:      5
Priority:    P0
Scope:       E
BCs:         BC-2.05.005 (PC-3, EC-005–EC-008), BC-2.05.008 (EC-006)
VP:          VP-059
Dependencies:
  - S-2.01 (HMAC primitive — internal/hmac already built)
  - S-3.04 (RouteFrame wire-up in internal/routing — already built; this story adds
            the call to admission.RecordHMACFailure within RouteFrame's failure path)
  - internal/admission package (must exist; FailureCounter is a new exported type)
  - internal/routing/routing.go (RouteFrame; add RecordHMACFailure call in the
            ErrHMACVerificationFailed return path)

Acceptance Criteria (summary for story-writer to expand into full ACs):
  AC-1: FailureCounter type defined in internal/admission with NewFailureCounter
        constructor and RecordHMACFailure(srcAddr string) method.
  AC-2: RecordHMACFailure uses a sliding window (60s default); trims stale entries
        strictly before now-windowDuration on every call.
  AC-3: E-ADM-017 structured log event emitted when count reaches threshold (5).
        Format: "HMAC failure rate alert: ≥5 failures in 60s from src <src_addr>"
        Severity: ERROR. Emitted via the injected logger; no global state.
  AC-4: Alert fires exactly once per threshold crossing (fire-once-per-crossing);
        does NOT re-emit for the 6th, 7th, … failure in the same window.
  AC-5: After window expires (all prior entries trimmed), a new batch of ≥5 failures
        fires the alert again. Test: 5 failures at T=0, 5 failures at T=61s → 2 alerts.
  AC-6: RouteFrame in internal/routing calls router.failureCounter.RecordHMACFailure(
        hdr.SrcAddr) immediately before returning ErrHMACVerificationFailed. No call
        on successful HMAC verification.
  AC-7: FailureCounter is concurrency-safe (sync.Mutex). go test -race MUST pass.
  AC-8: Boundary test: 4 failures → no E-ADM-017. 5th failure → exactly 1 E-ADM-017.
  AC-9: Multi-source test: 5 failures from addr-A + 5 from addr-B → 2 E-ADM-017
        events, one per source.
  AC-10: Exact-boundary test (EC-008): 5th failure timestamp = exactly windowDuration
        after 1st → alert fires (boundary is kept, not trimmed).

Files affected (for story-writer to enumerate):
  - internal/admission/failure_counter.go (new)
  - internal/admission/failure_counter_test.go (new)
  - internal/routing/routing.go (add RecordHMACFailure call; add failureCounter field
    to Router struct; wire via constructor)
  - internal/routing/routing_test.go (add EC-006 test for alert call)
```
