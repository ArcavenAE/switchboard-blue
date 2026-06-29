---
artifact_id: ARCH-03-routing-engine
document_type: architecture-section
level: L3
version: "1.6"
status: draft
producer: architect
timestamp: 2026-06-23T00:00:00
phase: 1b
traces_to: ARCH-INDEX.md
inputDocuments:
  - '.factory/specs/prd.md'
  - '.factory/specs/behavioral-contracts/ss-02/BC-2.02.001.md'
  - '.factory/specs/behavioral-contracts/ss-02/BC-2.02.002.md'
  - '.factory/specs/behavioral-contracts/ss-02/BC-2.02.003.md'
  - '.factory/specs/behavioral-contracts/ss-02/BC-2.02.004.md'
  - '.factory/specs/behavioral-contracts/ss-02/BC-2.02.005.md'
  - '.factory/specs/behavioral-contracts/ss-02/BC-2.02.006.md'
  - '.factory/specs/behavioral-contracts/ss-02/BC-2.02.007.md'
  - '.factory/specs/behavioral-contracts/ss-02/BC-2.02.008.md'
  - '.factory/specs/behavioral-contracts/ss-02/BC-2.02.009.md'
  - '.factory/specs/behavioral-contracts/ss-06/BC-2.06.003.md'
  - '.factory/specs/domain-spec/invariants.md'
kos_anchors:
  - elem-dual-fastest-path-forwarding
  - elem-asymmetric-half-channels
modified:
  - 2026-06-23T00:00:00
  - 2026-06-27T00:00:00
  - 2026-06-28T00:00:00
  - 2026-06-28T12:00:00
  - 2026-06-28T14:00:00
changelog:
  - version: "1.1"
    date: 2026-06-27T00:00:00
    adjudication: S-4.01 pass1-spec-rulings
    changes:
      - "Correction 1 (§Duplicate-and-Race, F-006 block): fixed endpoint dedup key description. The destination endpoint deduplicates by checksum alone (BC-2.02.002); the compound (checksum, arrival_interface_id) key applies only to the router-level drop cache (BC-2.02.009). Also corrected the OnFrameArrival pseudo-code comment citation from BC-2.02.002 to BC-2.02.009."
      - "Correction 2 (§Path Selection, degenerate single-path case): replaced 'both copies go to the same path (degenerate case)' with single-path no-duplication language per BC-2.02.001 postcondition 3 and EC-001."
  - version: "1.2"
    date: 2026-06-28T00:00:00
    adjudication: S-4.03 pass2-adjudication
    changes:
      - "Correction (§Downstream ARQ, delivery contract): OnAck returns deliverable frames synchronously as [][]byte. The caller's tick loop forwards them to the terminal. There is no internal DeliveredFrames channel or goroutine for frame delivery (pure-core constraint; S-4.03 pass-2 adjudication ruling 1)."
  - version: "1.3"
    date: 2026-06-28T00:00:00
    adjudication: S-4.03 RULING-003 ackseq-dos-ruling
    changes:
      - "Addition (§Downstream ARQ, input validation): OnAck validates ackSeq is within sackWindowSize (64) of nextExpected before iterating. Out-of-window ackSeq returns ErrAckOutOfWindow without state mutation. Unsigned subtraction handles stale-ACK case. Traces to BC-2.02.005 EC-005."
  - version: "1.4"
    date: 2026-06-28T12:00:00
    adjudication: Wave 4 fresh-context consistency audit — DRIFT-S4.03-001 owner correction
    changes:
      - "Addition (§ADR-005, implementation ownership note): DRIFT-S4.03-001 resolution — ADR-005 resync-on-reconnect wire-mechanics are owned by S-BL.NI (network-ingress), not S-5.01. S-5.01 is scoped only to internal/metrics quality indicator (BC-2.06.001/002). Drift entry owner field corrected to S-BL.NI."
  - version: "1.5"
    date: 2026-06-28T14:00:00
    adjudication: Wave 5 design notes — S-5.03 degraded-path flag
    changes:
      - "Addition (§Degraded-Path Flag Design, S-5.03): IsDegraded() accessor on PathTracker; degraded bool field set atomically under existing mu on each RTT update; threshold constant DegradedRTTThresholdMS=200; PathSnapshot value type for lock-compliant reads; internal/metrics consumes via PathSnapshot.Degraded. Traces to BC-2.02.003 PC-5."
  - version: "1.6"
    date: 2026-06-28T14:00:00
    adjudication: Wave 5 design notes — S-5.02 p99 RTT accumulator
    changes:
      - "Addition (§p99 RTT Accumulator Design, S-5.02): fixed-bucket latency histogram owned by PathTracker in internal/paths; 16 buckets covering 0–2000ms; O(1) update on each RTT probe; P99() query method on PathTracker; PathSnapshot carries P99RTTMs float64; internal/metrics reads via PathSnapshot. Traces to BC-2.06.003 rtt_p99_ms. VP for accumulator accuracy deferred to S-BL.BENCH."
---

# ARCH-03: Routing Engine

## Path Selection and Quality Tracking (internal/paths)

Path quality is tracked as EWMA (Exponentially Weighted Moving Average) of RTT and
loss rate per connected router. Keepalive probes are sent at the configured interval
(default 1s); RTT is measured as round-trip time for the probe frame.

**Path ranking** (BC-2.02.003): Paths are ranked by a composite score:
```
score = rtt_ewma_ms * (1 + loss_ewma_fraction * loss_weight)
```
Lower score = better path. `loss_weight` = 10 (configurable). This penalizes
lossy paths more than pure latency measurements.

**Top-2 paths** are selected for duplicate-and-race (BC-2.02.001). If only one
path exists, the frame is sent on that single path with no duplication
(BC-2.02.001 postcondition 3, EC-001). Single-path mode is noted in the quality
indicator.

## Duplicate-and-Race (internal/multipath)

Per elem-dual-fastest-path-forwarding, frames are sent simultaneously on the two
fastest paths. The receiver (internal/multipath on the destination node) implements:

```
DropCache: bounded LRU map of (checksum, arrival_interface_id) → arrival_time
  capacity: config.DropCacheSize (default 10,000 entries)

OnFrameArrival(frame, arrival_interface_id):
  checksum = crc32(frame.outer_header || frame.payload)
  key = (checksum, arrival_interface_id)
  if DropCache.contains(key):
    silently discard (BC-2.02.009, DI-009)
    return
  DropCache.add(key)
  deliver(frame)
```

**Drop cache key (F-006):** The drop cache key is `(checksum, arrival_interface_id)`,
not `(checksum)` alone. This ensures two copies of a frame arriving on different
interfaces are both kept — multipath delivery requires both copies to survive
intermediate hops so the fastest arrives first. Deduplication per-destination
happens in the forwarding stage: if both copies arrive at the same destination
interface, the second is discarded. If they arrive on different interfaces (as
expected in dup-and-race), both are forwarded. The destination node receives at
most two copies. The endpoint receiver deduplicates by checksum alone
(BC-2.02.002): the first-arriving copy is delivered; the second is silently
discarded regardless of arrival interface. Router-side compound keying ensures
both copies reach the destination; endpoint-side checksum-only keying ensures
only one is delivered to the application.

**BC-2.02.009 (bounded drop cache):** LRU eviction ensures the cache never exceeds
capacity. When the cache is full, the oldest entry is evicted. This means very old
checksums may be re-admitted, but this is acceptable because duplicate-and-race
operates in the sub-second regime.

**BC-2.02.008 (split-horizon):** The router maintains per-frame the arrival interface.
A frame is never forwarded back toward its arrival interface. Implemented as: before
forwarding, check `dst_interface != arrival_interface`. This prevents loops in
multi-path topologies.

## ADR-002: FEC Group Size

**Decision:** Default N=4 (4 data frames + 1 XOR parity frame per group). Tunable
via `fec_group_size` config field. Range: 2–16.

**Rationale:** FEC (BC-2.02.007, Phase PE) applies in multi-path topologies where
single-frame loss in a group is expected. Group size 4 provides a 20% FEC overhead,
recoverable from any single loss. Group size 2 would provide 50% overhead (high);
group size 8 would delay recovery by 7 frames.

The default N=4 is consistent with MOSH's FEC defaults and established in practice
for interactive terminal sessions. Phase 3 benchmarks validate this against measured
loss rates in PE topologies.

**FEC is not implemented in E router MVP** (P1, PE scope). The `FEC_present` flag
in the channel header (bit 0 of `flags`) is 0x00 in MVP; the FEC code path is behind
a `upstream_routers != nil` guard.

## Upstream Idempotent Replay (internal/replay, BC-2.02.004)

Each upstream frame carries the last N keystrokes as a replay window (U-C sliding
window). The receiver deduplicates by `chan_seq`. The replay window size N is
configurable (default 3 keystrokes).

```
ReplayWindow: ring buffer of size N
  Each upstream frame: payload = [current_keystroke | prev_N-1_keystrokes]
  Receiver: on delivery, check chan_seq against seen set; deduplicate
```

This means keystroke loss is self-healing without explicit ARQ, at the cost of
`N * avg_keystroke_bytes` per frame overhead. For N=3 and typical keystrokes (1–4
bytes each), overhead is ≤ 12 bytes/frame — acceptable.

## Downstream ARQ (internal/arq, BC-2.02.005)

```
Sender (access node):
  SendBuffer: sliding window of unacknowledged frames
  Each frame: chan_seq in channel header
  Piggyback ACK: cumulative acknowledgment + SACK bitmap in channel header (flags bit 2 set)
  On timeout: retransmit unACK'd frames

Receiver (console):
  RecvBuffer: reorder buffer, delivers in sequence
  SACK bitmap sent in channel header (SACK_present=1) on every downstream tick
  SACK bitmap marks out-of-order received frames
```

**SACK location:** The SACK bitmap is embedded in the channel header as a conditional
8-byte field (see ARCH-02 Channel Header). When SACK_present=1 (flags bit 2), the
channel header is 20 bytes. This eliminates the need for a standalone ACK channel.

**Read-only console ACK (F-023):** Read-only consoles do NOT use a dedicated standalone
ACK channel. All consoles — including read-only consoles — have a degenerate upstream
half-channel that produces empty-tick frames (BC-2.01.002: all half-channels produce
empty-tick frames; BC-2.01.003: upstream shutdown does not stop downstream). The
degenerate upstream half-channel carries SACK in its channel header (`SACK_present=1`)
even when carrying no data payload. This means:
- The upstream half-channel of a read-only console emits empty-tick frames at its
  configured tick interval.
- Each such frame has `SACK_present=1` in the channel header flags.
- The 8-byte SACK bitmap in the channel header carries the ACK state for the
  downstream half-channel.
- No dedicated ACK channel is needed, defined, or implemented.

**Delivery contract (S-4.03 pass-2 adjudication):** `OnAck` returns deliverable
frames synchronously as `[][]byte`. The caller's tick loop forwards them to the
terminal. There is no internal `DeliveredFrames` channel or goroutine for frame
delivery — `internal/arq` is pure-core and may not spawn goroutines (same
constraint as `internal/halfchannel`). `TLPKTDROP` returns the degradation event
as a value and additionally sends it to the buffered `DegradationEvents` channel
for the metrics layer.

**Input validation (RULING-003):** `OnAck` validates that `ackSeq` lies within one
SACK window (64 positions) of `nextExpected` before executing the Step-1 iteration
loop. An out-of-window cumulative ACK is a protocol-illegal frame; `OnAck` returns
`ErrAckOutOfWindow` without iterating and without mutating ARQ state. Callers must
check this error and discard the frame. The guard uses unsigned subtraction:
stale ACKs (`ackSeq < nextExpected`) also trigger rejection via uint32 wrap, with
no separate comparison required. Traces to BC-2.02.005 EC-005.

**BC-2.02.006 (TLPKTDROP):** When a downstream frame is overdue beyond
`tlpktdrop_timeout` (default: 2 × tick_interval), the frame is dropped with a
TLPKTDROP signal. This prevents head-of-line blocking at the cost of terminal
display corruption. The quality indicator degrades on TLPKTDROP.

**TLPKTDROP timeout formula:** `2 × tick_interval` is the architectural default.
The 2x factor ensures we tolerate normal jitter (≤ 1 tick per DI-008) before
dropping. Configurable via `arq_drop_timeout_multiplier`.

## ADR-005: Downstream ARQ Continuity Under Router Failover

**OQ-004 resolution:** Resolves the open question in invariants.md OQ-004 — downstream switchover continuity. The chosen approach is resync-from-last-ACK rather than stateful ARQ state transfer, on grounds of simplicity and correctness within MVP scope.

**Decision:** On path failover (node disconnects from one router and reconnects to
another), the downstream half-channel performs a **resync**: the receiver sends a
`RESYNC` control frame requesting the sender to retransmit from the last
acknowledged sequence number. The sender replays from `last_acked_seq + 1`.

**Rationale:** In-flight frames during failover are lost (the old router connection
is dead). The downstream half-channel cannot guarantee delivery across the failover.
Resync is safe because:
1. The SACK bitmap tells the receiver exactly what it has and hasn't seen.
2. Retransmit carries the original `chan_seq`, so deduplication at the receiver works.
3. Terminal state is recoverable from a retransmit — ARQ is ordered delivery.

**Rejected alternative:** Stateful continuity (transferring ARQ state to the new
router). This requires router state transfer protocol, which is out of MVP scope.
Resync is simpler and correct.

**Open question for KoS:** This sketches the PE-phase design. The exact `RESYNC`
frame format and state machine are deferred to PE implementation. E router has a
single path; failover only occurs on manual restart.

**Implementation ownership (DRIFT-S4.03-001 resolution, 2026-06-28):** The
wire-mechanics of ADR-005 (RESYNC frame emission, reconnect state machine, replay
from `last_acked_seq + 1`) belong to **S-BL.NI** (network-ingress story), not
S-5.01. Rationale: resync fires when the node re-establishes a network connection
to a new router — this is a connection-lifecycle concern owned by the network-ingress
layer (internal/arq + network-ingress wiring). S-5.01 is scoped exclusively to the
quality-indicator state machine in `internal/metrics` (BC-2.06.001/002) and has no
ARQ reconnect scope. The `internal/arq` pure-core state machine (S-4.03) provides the
primitives (last ACK seq, retransmit); the effectful reconnect trigger lives in the
ingress layer. DRIFT-S4.03-001 owner field should read: `S-BL.NI`.

## Quality Indicator (internal/metrics, BC-2.06.001–002)

**Quality indicator semantic (per BC-2.06.001 invariant 4):**

The session-aggregated quality indicator uses the BEST current path's metrics:
- green: best path RTT p99 ≤ 100ms AND best path loss ≤ 5%
- yellow: best path RTT p99 in (100ms, 500ms] OR best path loss in (5%, 20%]
- red: best path RTT p99 > 500ms OR best path loss > 20% OR no paths available

Per-path metric scoring (used for path ranking and forwarding decisions per
CAP-006) uses each path's own metrics independently — a degraded path is ranked
lower but doesn't degrade the session indicator if a healthy path exists.

Transitions are hysteretic over 3 consecutive measurements (per BC-2.06.001
invariant 3, NFR-014).

```
QualityState: green | yellow | red

Transitions (based on BEST path metrics):
  green → yellow: best path RTT p99 > 100ms OR best path loss > 5%
  yellow → red:   best path RTT p99 > 500ms OR best path loss > 20% OR no paths available
  red → yellow:   best path RTT p99 ≤ 500ms AND best path loss ≤ 20% AND ≥1 path available
  yellow → green: best path RTT p99 ≤ 100ms AND best path loss ≤ 5%

Canonical thresholds (NFR-001: 100ms p99 LAN budget):
  green_rtt_ms     = 100   (≤ 100ms: green)
  yellow_rtt_ms    = 500   (100–500ms: yellow)
  red_rtt_ms       = 500   (> 500ms: red)
  green_loss_pct   = 5     (≤ 5%: green)
  yellow_loss_pct  = 20    (5–20%: yellow)
  red_loss_pct     = 20    (> 20%: red)
```

**Hysteresis (F-021):** Transitions require 3 consecutive measurements before
firing. This is the canonical hysteresis value, derived from BC-2.01.002 EC-001
("≥3 consecutive missed ticks") and BC-2.06.001 invariant 3 ("3-consecutive-
measurement hysteresis"). A single spike does not trigger a state change.

**Quality indicator update:** The `internal/metrics` package is called from the
path scoring loop; there is no batching between path measurement and indicator
update. Quality indicator updates within 3 tick cycles of sustained path quality
change.

## Session Discovery (internal/discovery, BC-2.03.001–003, Phase PE)

Presence advertisement is out of E router MVP scope. Architecture sketch for PE:
- Access nodes send `PRESENCE_ADV` frames to a well-known SVTN multicast address.
- Consoles subscribe to multicast and maintain a local session list.
- Heartbeat interval: 30s default (tuning parameter; see ARCH-INDEX).
- State-change advertisements are immediate (attach/detach events).

The multicast address is scoped to the SVTN (uses `svtn_id` as part of the multicast
group derivation) to enforce DI-005 (SVTN isolation).

## Degraded-Path Flag Design (internal/paths, S-5.03)

**Traces to:** BC-2.02.003 PC-5 — "a path whose RTT exceeds the degradation threshold
(implementation: >200ms) is flagged as degraded."

### Threshold constant

```go
// DegradedRTTThresholdMS is the RTT (milliseconds) above which a path is
// considered degraded. Traces to BC-2.02.003 postcondition 5.
const DegradedRTTThresholdMS = 200.0
```

No config exposure in Wave 5. The threshold is architectural (NFR-001 100ms LAN
budget with a 2x headroom); it should not be tunable without a separate ADR.

### Field addition to PathTracker

Add a single `degraded bool` field alongside the existing `active bool`. Both
fields share the existing `mu sync.Mutex` — no new locking primitive is needed.

```go
type PathTracker struct {
    mu sync.Mutex
    // ... existing fields unchanged ...
    degraded bool  // true when ewmaRTTMS > DegradedRTTThresholdMS
}
```

### Update path: inside OnProbe

The degraded flag is re-evaluated at the same point where `ewmaRTTMS` is written,
still under `t.mu`. This is the only place the EWMA is mutated.

```go
// After updating t.ewmaRTTMS (successful probe branch):
t.degraded = t.ewmaRTTMS > DegradedRTTThresholdMS
```

On a loss event the EWMA RTT is not updated (only loss EWMA moves), so `degraded`
is not re-evaluated on miss — consistent with existing semantics: RTT is only
known from successful round-trips.

On `resetRTT` (first probe and reactivation): set `t.degraded` from the reset value:
```go
t.degraded = arrivalRTTMS > DegradedRTTThresholdMS
```

### Accessor

```go
// IsDegraded reports whether the path's EWMA RTT currently exceeds
// DegradedRTTThresholdMS. A degraded path remains in the active set and
// continues to be ranked; it is not removed unless consecutive misses
// also accumulate. Traces to BC-2.02.003 PC-5.
func (t *PathTracker) IsDegraded() bool {
    t.mu.Lock()
    defer t.mu.Unlock()
    return t.degraded
}
```

### PathSnapshot value type (go.md rule 12 compliance)

The existing `RTT()`, `LossPct()`, `IsActive()` accessors each take the lock
separately, which is safe for individual reads but can produce inconsistent
snapshots when multiple fields are needed together (e.g., by `internal/metrics`).
For S-5.03 and S-5.02 (p99, below) introduce a single consistent snapshot type:

```go
// PathSnapshot is a point-in-time copy of all PathTracker fields that
// consumers need. It is a value type; callers never hold a pointer into
// PathTracker's internal state (go.md rule 12).
type PathSnapshot struct {
    EWMARTS    float64 // current EWMA RTT, milliseconds
    LossPct    float64 // current EWMA loss percentage (0–100)
    Active     bool
    Degraded   bool    // EWMA RTT > DegradedRTTThresholdMS
    P99RTTMs   float64 // p99 RTT from histogram (see §p99 accumulator)
}

// Snapshot returns a consistent point-in-time copy of all tracker fields
// under a single lock acquisition. Callers must not retain pointers to the
// returned value's fields — the value is already a safe copy.
func (t *PathTracker) Snapshot() PathSnapshot {
    t.mu.Lock()
    defer t.mu.Unlock()
    return PathSnapshot{
        EWMARTS:  t.ewmaRTTMS,
        LossPct:  t.ewmaLossPct,
        Active:   t.active,
        Degraded: t.degraded,
        P99RTTMs: t.hist.p99(),
    }
}
```

The individual `RTT()`, `LossPct()`, `IsActive()`, `IsDegraded()` single-field
accessors remain on the type for test legibility and for callers that only need
one field. `internal/metrics` and `sbctl` serialization MUST use `Snapshot()` to
avoid split-lock inconsistency across fields.

### Interaction with quality indicator (S-5.01, internal/metrics)

`internal/metrics` currently calls individual path accessors to build the quality
state machine input. With S-5.03 merged, it switches to `Snapshot()` so that the
`Degraded` field is read atomically with `Active` and `P99RTTMs`. The quality
indicator does not change its state machine thresholds as a result of S-5.03 —
the degraded flag is a per-path property that feeds the `status` field in the
`sbctl paths list` output (BC-2.06.003 PC-1: `status: active|degraded|failed`)
and is an input to the yellow/red quality state machine (RTT p99 > 100ms feeds
yellow; RTT p99 > 500ms feeds red — the 200ms degraded threshold is a distinct,
coarser per-path label, not a quality-indicator threshold).

**Composition contract:** S-5.01 (internal/metrics) reads `Snapshot().Degraded`
to set the per-path `status` field in the metrics snapshot it exposes. S-5.01
does not need to know `DegradedRTTThresholdMS`; it only reads the pre-computed
boolean. The threshold evaluation stays entirely in `internal/paths`.

### Concurrency contract summary

- One lock (`t.mu`) guards all PathTracker state including `degraded`.
- `degraded` is written only in `OnProbe` and `resetRTT`, both under `t.mu`.
- `IsDegraded()` and `Snapshot()` read under `t.mu`.
- No goroutines are spawned; PathTracker remains pure-core.

---

## p99 RTT Accumulator Design (internal/paths, S-5.02)

**Traces to:** BC-2.06.003 PC-1 — `rtt_p99_ms` field in `sbctl paths list` output.

### Why a histogram, not a ring buffer

Two candidate approaches:

| Approach | Memory per path | Update cost | Query cost | Accuracy |
|----------|----------------|-------------|------------|---------|
| Ring buffer of last N samples (N=100) | 800 bytes (100 × float64) | O(1) amortized | O(N) sort | Exact over window |
| Fixed-bucket histogram | 128 bytes (16 × uint64) | O(1) | O(buckets) = O(1) | Bucketed approximation |

For an interactive terminal session with 1s probe interval, 100-sample ring =
100s window, which is reasonable, but the O(N) sort on every `sbctl paths list`
query is unnecessary overhead on the hot path. The histogram uses 6.25× less
memory and has O(1) query cost. The approximation error is ≤ bucket width (chosen
to be ≤ 25ms for the 0–200ms range most relevant to NFR-001). **Decision: histogram.**

The ring buffer would be the right choice if exact percentiles were needed for
formal verification purposes — flag this as VP-deferred (see below).

### Bucket layout

16 buckets covering 0–2000ms:

```
Bucket  0:    0 –   25ms  (width 25ms)
Bucket  1:   25 –   50ms
Bucket  2:   50 –   75ms
Bucket  3:   75 –  100ms
Bucket  4:  100 –  150ms  (width 50ms from here)
Bucket  5:  150 –  200ms
Bucket  6:  200 –  300ms  (width 100ms from here)
Bucket  7:  300 –  400ms
Bucket  8:  400 –  500ms
Bucket  9:  500 –  750ms  (width 250ms)
Bucket 10:  750 – 1000ms
Bucket 11: 1000 – 1250ms
Bucket 12: 1250 – 1500ms
Bucket 13: 1500 – 1750ms
Bucket 14: 1750 – 2000ms
Bucket 15: 2000ms+        (overflow)
```

Rationale: finer buckets in the 0–200ms range (NFR-001: 100ms p99 LAN budget)
where the quality thresholds live; coarser buckets at high latency where only
the red/failed distinction matters. Total: 16 × uint64 = 128 bytes per path.

### Data structure (private to PathTracker)

```go
// rttHistogram is a fixed-bucket count histogram for per-path RTT samples.
// All fields are read/written under PathTracker.mu — no independent locking.
type rttHistogram struct {
    counts [16]uint64
    total  uint64 // total samples recorded
}

// bucketFor returns the bucket index for a given RTT in milliseconds.
func bucketFor(rttMS float64) int { /* bucket boundary lookup */ }

// record adds one RTT sample to the histogram.
func (h *rttHistogram) record(rttMS float64) {
    h.counts[bucketFor(rttMS)]++
    h.total++
}

// p99 returns the p99 RTT estimate in milliseconds.
// Returns 0 if no samples have been recorded.
// The returned value is the upper bound of the bucket containing the 99th
// percentile count (conservative: rounds up, never down).
func (h *rttHistogram) p99() float64 { /* walk counts until 99% reached */ }
```

The histogram is embedded directly in `PathTracker` as `hist rttHistogram`.
No heap allocation; the PathTracker struct grows by 136 bytes (16×8 + 8).

### Update path

`rttHistogram.record(arrivalRTTMS)` is called from `OnProbe` on every successful
probe (lossEvent=false), after the EWMA update, still under `t.mu`. Loss events
do not contribute a sample (no RTT is measured on a miss).

`resetRTT` also calls `h.record(arrivalRTTMS)` so the reactivating probe seeds
the histogram before the path re-enters the ranked set.

Histogram counts are never reset (unlike EWMA which resets on reactivation).
Rationale: the path's historical latency distribution is still valid after
reactivation; the reset RTT EWMA handles the "fresh start" for scoring purposes.
If the path has been dormant for a long time the histogram will represent stale
data — acceptable for MVP given the E-phase single-LAN scope where path RTT is
stable. Revisit for PE-phase if roaming is needed.

### Query path

`P99RTTMs float64` is included in `PathSnapshot` (see §Degraded-Path Flag Design
above). The `Snapshot()` method calls `t.hist.p99()` under the same `t.mu` lock
hold. No additional locking. The `sbctl` serialization layer reads
`PathSnapshot.P99RTTMs` and emits it as `rtt_p99_ms` in the JSON output, satisfying
BC-2.06.003 PC-1.

### Memory bound

Per-path histogram cost: 136 bytes. A node with 8 active paths = 1,088 bytes total.
This is negligible and does not require pooling or budgeting.

### Concurrency contract

Identical to the degraded-flag contract: `hist` is read and written only under
`t.mu`. `Snapshot()` is the only external read surface. No goroutines.

### VP note — accumulator accuracy

The histogram approximation (p99 ≤ true p99 + bucket_width) is not formally
verified in S-5.02. This is acceptable for the CLI diagnostic use case; operators
read it as "approximately 22ms" not "exactly 22ms". A formal accuracy VP
(property: `p99() ≤ true_p99 + max_bucket_width`) belongs in S-BL.BENCH when
benchmark harnesses are available for calibration. Story-writer should note this
as a deferred VP when writing S-5.02; test-writer should write a table-driven
accuracy test with synthetic sample distributions to bound the approximation error.
