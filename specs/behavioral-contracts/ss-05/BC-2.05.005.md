---
artifact_id: BC-2.05.005
document_type: behavioral-contract
level: L3
version: "1.5"
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
  - '2026-06-27: v1.4 — per-story adversarial convergence adjudication: (HF-1) ratify periodic-re-fire semantics under sustained attack; (HF-2) mandate dead-key eviction + hard source-cap (max 65536 tracked sources, LRU); (item-5) add WithNow clock-injection to constructor signature; (item-5) add constructor validation clause for threshold<1 and window<=0; amend EC-005, EC-008, add EC-009/EC-010; update PC-3 API contract'
  - '2026-06-27: v1.5 — S-W3.05 per-story adversarial convergence adjudication (M-1, O-1): (M-1 FIX-NOW CWE-770) mandate per-source append-skip policy: when firedAt[srcAddr] is set and re-arm has not triggered, new timestamps are NOT appended — slice is bounded at threshold entries maximum; add EC-011 (high-rate attack, bounded-slice invariant); (O-1) remove "at ERROR level" phrase — Logger seam is level-less (Log(msg string)); severity is taxonomy-owned (degraded per E-ADM-017); amend PC-3 + Window semantics clause accordingly'
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
3. **Per-source HMAC failure rate alert:** When `RouteFrame` returns `ErrHMACVerificationFailed` for a frame from `src_addr`, it MUST call `admission.RecordHMACFailure(srcAddr string)` on the router's failure recorder before returning. The `FailureCounter` maintains a per-`src_addr` sliding-window counter over a 60-second window. When the count for a `src_addr` reaches or exceeds **5** within any trailing 60-second window, the `FailureCounter` emits a structured log event via the injected `Logger` interface (`Log(msg string)`) with code **E-ADM-017** ("E-ADM-017 HMAC failure rate alert: ≥`<threshold>` failures in `<window_seconds>`s from src `<src_addr>`"). The message embeds the code literal "E-ADM-017" for operator grep-ability. Severity is **not** encoded as a logger level (the `Logger` seam is level-less); operator severity is defined by the error taxonomy as `degraded` (daemon continues). Operators may route the message to an error-level sink by checking for the "E-ADM-017" prefix in their log pipeline.

   **Re-fire semantics under sustained attack (ratified):** The alert fires on the threshold crossing AND re-fires under a sustained attack. Specifically: after an alert fires, the alert is suppressed until all in-window entries that were present at the time of the alert have been trimmed away — i.e., the oldest surviving entry after trim is newer than the last-fire timestamp. At that point the counter re-arms. If failures are still arriving at that moment (sustained attack), the next threshold crossing fires another alert. This means: under a sustained attack of exactly N failures/60s (N≥threshold), alerts fire roughly once per window-length. Under a brief attack that drops below threshold after the alert fires, only one alert fires (classic hysteresis). This is operationally correct: security operators receive ongoing alerts during active forgery floods without per-failure spam. See EC-005 (brief attack), EC-009 (sustained attack) for canonical scenarios.

   **Admission-layer API contract** (the seam the implementer builds — no Go code, but the contract is precise):
   - Type: `admission.FailureCounter` in `internal/admission`
   - Method: `RecordHMACFailure(srcAddr string)` — pure in-memory; takes no `context.Context` (no I/O; a mutex-guarded in-memory sliding window qualifies as pure-enough for this call path)
   - Constructor: `admission.NewFailureCounter(threshold int, windowDuration time.Duration, logger Logger, opts ...FailureCounterOption) *FailureCounter` — logger is injected (dependency injection; no package-level global). Optional `FailureCounterOption` values include `WithNow(fn func() time.Time)` for deterministic clock injection in tests.
   - **Constructor validation:** If `threshold < 1`, the constructor MUST panic with a clear message (a threshold of 0 or negative would fire on every single failure, which is always a programmer error, not a configuration error). If `windowDuration <= 0`, the constructor MUST panic. Both panics use `panic(fmt.Sprintf(...))` in `NewFailureCounter`; they are programmer-error guards, not runtime error paths.
   - Internal state: a `map[string][]time.Time` of per-`src_addr` timestamp slices, guarded by `sync.Mutex`; entries are evicted lazily when a `RecordHMACFailure` call trims stale timestamps older than `windowDuration` (no background goroutine required; per go.md rule #12 the map entries are value-copied on read; no internal pointer to a live slice is returned to callers)
   - **Dead-key eviction:** When the post-trim slice for a `srcAddr` is empty (count = 0), the `srcAddr` key MUST be deleted from the `counts` map entirely (not kept as an empty slice). The corresponding `firedAt` entry MUST also be deleted when the window drains to zero. This prevents unbounded map growth from sources that sent a few failures and then disappeared.
   - **Hard source cap (CWE-770 mitigation):** The `FailureCounter` tracks at most **65,536** distinct `srcAddr` keys. When a new `srcAddr` would exceed this cap, the key with the oldest most-recent-failure timestamp (LRU) is evicted from both `counts` and `firedAt` before inserting the new key. This bounds memory under spoofed-source floods. The cap is a compile-time constant `maxTrackedSources = 65536` defined in the package; it is not configurable at runtime via the constructor (security invariant: the cap cannot be disabled).
   - `RouteFrame` receives the failure recorder via constructor injection on the `Router` struct using the `hmacFailureRecorder` interface (`RecordHMACFailure(string)`); `*admission.FailureCounter` is the production implementation. `internal/routing` imports `internal/admission` (position 4→5, consistent with ARCH-08 §6.5).
   - **Per-source slice bound (CWE-770 amplification mitigation — M-1):** After an alert fires for a `srcAddr` (i.e., `firedAt[srcAddr]` is non-zero and the re-arm condition has not yet been met), new timestamps MUST NOT be appended to the slice for that source. The implementation MUST skip the append step for that `srcAddr` until re-arm. This bounds the per-source slice at most `threshold` entries at any time (the entries that were present at or before the alert threshold-crossing; subsequent entries age out without being replaced). Under a high-rate attack (`rate >> threshold/windowDuration`), memory per source is bounded at `threshold × sizeof(time.Time)` regardless of attack rate. See EC-011 for the canonical test scenario.
   - Window semantics: **sliding** (not fixed-bucket) — at each `RecordHMACFailure` call, timestamps older than `now - windowDuration` are trimmed; the count of remaining entries (after trim, and before append if append is permitted — see per-source slice bound above) determines threshold comparison. After trim, if `firedAt[srcAddr]` is non-zero and `len(keep) == 0` OR `keep[0].After(firedAt[srcAddr])`, the counter re-arms (deletes `firedAt[srcAddr]`); at that point the normal append step proceeds and threshold counting resumes.

   **Concurrency contract:** `RecordHMACFailure` is safe for concurrent calls from multiple goroutines. The `sync.Mutex` in `FailureCounter` is held for the duration of the trim+eviction+append+check sequence. Per go.md rule #12: the slice of timestamps is never returned by reference to the caller; if a `Timestamps(srcAddr string) []time.Time` inspector is ever needed (e.g., for tests), it returns a copy.

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
| EC-005 | Alert hysteresis — brief attack: 6 failures in 60s, then 0 failures for 61s, then 5 more failures in 60s | First batch: E-ADM-017 fires on the 5th failure. After the window expires (all 6 entries age out), the counter re-arms. Second batch: E-ADM-017 fires again on the 5th failure of the new window. Exactly 2 alert events total. Dead-key eviction: after both windows drain, both `counts` and `firedAt` entries for the srcAddr are deleted entirely. |
| EC-006 | Exactly 4 failures within 60s, no 5th failure | No E-ADM-017 emitted. Counter holds 4 timestamps; next trimmed-window check returns 4 < threshold. |
| EC-007 | Concurrent HMAC failures from two different src_addrs, each hitting ≥5 threshold | Each src_addr has its own counter slot. Both cross the threshold independently; two separate E-ADM-017 events emitted (one per src_addr). No interference between counters. |
| EC-008 | 5th failure arrives at exactly windowDuration after the 1st failure (boundary) | After trimming entries older than `now - windowDuration`, the 1st entry falls on the boundary. **Correct behavior: trim entries where `timestamp < now - windowDuration` (strictly less-than); boundary entry is kept; post-trim count = 4; after append = 5; alert fires.** An implementation using `<=` (trim-at-boundary) yields count=4 and fails to alert — that is a defect. This test discriminates the two comparisons. |
| EC-009 | Sustained attack: ≥5 failures/60s continuously, window never drains below threshold | **Canonical sustained-attack scenario:** 5 failures at T=0s → alert-1 fires on the 5th. Then 1 more failure per second continuously. After approximately 60s, the entries from T=0..4s age out of the window; at that moment the oldest surviving entry is newer than the last-fire timestamp → counter re-arms. The next failure that brings the post-trim+append count to threshold fires alert-2. Pattern repeats: alerts fire roughly once every windowDuration while the attack is sustained. **The exact count of alerts is not pinned** (it depends on the rate of arrivals relative to the window); the testable property is: under a continuous stream of failures at rate ≥ threshold/window, MORE THAN ONE E-ADM-017 alert fires (i.e., the counter does not go permanently silent after the first alert). Discriminating test: inject 5 failures, advance clock by 61s (all aged out), inject 5 more → must fire a 2nd alert. For the truly continuous case: inject threshold failures, advance clock by windowDuration+ε (so at least one old entry ages out while new ones remain), inject threshold more → must fire a 3rd alert. |
| EC-010 | Memory bound: 65,537 distinct spoofed src_addrs each send 1 failure | After 65,536 insertions the cap is reached. The 65,537th srcAddr evicts the LRU key (the one with the oldest most-recent-failure timestamp) from `counts` and `firedAt`. Live key count remains ≤ 65,536 at all times. No unbounded map growth. Test: after inserting `maxTrackedSources+1` distinct sources, assert `len(counts) <= maxTrackedSources`. |
| EC-011 | High-rate attack: one src_addr sending 1,000,000 failures/second for 60s (append-skip bound) | After the 5th failure, alert fires and `firedAt[srcAddr]` is set. From the 6th failure onward, the append step is skipped (firedAt non-zero, re-arm not yet met). The per-source slice holds exactly `threshold` entries (the 5 timestamps present at the crossing) and does not grow further regardless of subsequent call rate. After 60s + ε, the 5 entries age out (post-trim `len(keep)==0`); re-arm triggers (deletes `firedAt`); the next `threshold` failures re-fill the slice and fire a second alert. Memory for this source is bounded at `threshold × sizeof(time.Time)` = 80 bytes during the attack. Test: inject `threshold` failures, then inject 1,000,000 more without advancing clock; assert `len(counts[srcAddr]) == threshold` (slice did not grow beyond threshold). |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Valid frame with correct HMAC | Frame forwarded; no log event | happy-path |
| Frame with HMAC computed with wrong key | E-ADM-002 logged; frame dropped | error |
| Frame with HMAC field all-zeros | E-ADM-002 logged; frame dropped | error |
| Empty-tick frame with correct HMAC | Frame forwarded normally | happy-path |
| 5 HMAC failures in 30s from same src_addr (RecordHMACFailure called 5 times) | E-ADM-017 emitted exactly once on the 5th call; message is "E-ADM-017 HMAC failure rate alert: ≥5 failures in 60s from src <src_addr>" | alert-threshold |
| 4 HMAC failures in 60s from same src_addr, no 5th | No E-ADM-017 emitted | below-threshold |
| 5 HMAC failures from src_addr A + 5 from src_addr B, interleaved | E-ADM-017 emitted once for A and once for B, independently | multi-source |
| 5 HMAC failures, then 61s pause, then 5 more from same src_addr | Two E-ADM-017 events (one per window crossing), not one; dead-key eviction: map entries deleted after drain | hysteresis |
| 5 failures, advance clock by windowDuration+ε (1 entry ages out, 4 remain), then 1 more failure | Counter re-arms (oldest surviving entry is newer than last-fire); 6th failure does NOT immediately re-fire; fires only when count reaches threshold again (i.e., on the 5th new failure after re-arm) | sustained-attack-rearm |
| threshold=0 (invalid) | NewFailureCounter panics with a clear message | constructor-validation |
| windowDuration=0 (invalid) | NewFailureCounter panics with a clear message | constructor-validation |
| 65,537 distinct src_addrs each send 1 failure | len(counts) ≤ 65,536; LRU key evicted | memory-bound |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-004, VP-005, VP-006 | For all admitted nodes: frames with correct HMAC are forwarded | proptest |
| VP-004, VP-005, VP-006 | For all non-admitted sources: frames are dropped | proptest |
| VP-004, VP-005, VP-006 | HMAC covers outer header bytes 0–35 + channel header + payload | unit |
| VP-059 | For any sequence of RecordHMACFailure calls with injected clock: (a) E-ADM-017 fires exactly on the call that brings the post-trim count to threshold; (b) subsequent calls in the same un-re-armed window do NOT fire E-ADM-017; (c) after re-arm (oldest surviving entry is newer than last-fire timestamp), the next threshold crossing fires E-ADM-017 again; (d) under a continuous stream of failures, alert count is ≥ 2 (counter never goes permanently silent); (e) live key count is always ≤ maxTrackedSources | proptest |

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
