---
artifact_id: ARCH-03-routing-engine
document_type: architecture-section
level: L3
version: "1.0"
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
  - '.factory/specs/domain-spec/invariants.md'
kos_anchors:
  - elem-dual-fastest-path-forwarding
  - elem-asymmetric-half-channels
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
path exists (E router MVP), both copies go to the same path (degenerate case).

## Duplicate-and-Race (internal/multipath)

Per elem-dual-fastest-path-forwarding, frames are sent simultaneously on the two
fastest paths. The receiver (internal/multipath on the destination node) implements:

```
DropCache: bounded LRU map of (checksum → arrival_time)
  capacity: config.DropCacheSize (default 10,000 entries)

OnFrameArrival(frame):
  checksum = crc32(frame.outer_header || frame.payload)
  if DropCache.contains(checksum):
    silently discard (BC-2.02.002, DI-009)
    return
  DropCache.add(checksum)
  deliver(frame)
```

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

**FEC is not implemented in E router MVP** (P1, PE scope). The outer header
`fec_flags` field is reserved (0x00) in MVP; the FEC code path is behind a
`upstream_routers != nil` guard.

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
  Piggyback ACK: cumulative acknowledgment + SACK bitmap (64-bit, 64 frames max)
  On timeout: retransmit unACK'd frames

Receiver (console):
  RecvBuffer: reorder buffer, delivers in sequence
  Piggybacked ACK sent on every downstream tick
  SACK bitmap marks out-of-order received frames
```

**BC-2.02.006 (TLPKTDROP):** When a downstream frame is overdue beyond
`tlpktdrop_timeout` (default: 2 × tick_interval), the frame is dropped with a
TLPKTDROP signal. This prevents head-of-line blocking at the cost of terminal
display corruption. The quality indicator degrades on TLPKTDROP.

**TLPKTDROP timeout formula:** `2 × tick_interval` is the architectural default.
The 2x factor ensures we tolerate normal jitter (≤ 1 tick per DI-008) before
dropping. Configurable via `arq_drop_timeout_multiplier`.

## ADR-005: Downstream ARQ Continuity Under Router Failover

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

## Quality Indicator (internal/metrics, BC-2.06.001–002)

```
QualityState: green | yellow | red

Transitions:
  green → yellow: any path RTT > yellow_rtt_threshold OR loss > yellow_loss_threshold
  yellow → red:   all paths RTT > red_rtt_threshold OR a TLPKTDROP event
  red → yellow:   all paths RTT back under red_rtt_threshold AND no recent TLPKTDROP
  yellow → green: all paths RTT under yellow_rtt_threshold AND loss under threshold

Default thresholds:
  yellow_rtt_ms = 50, red_rtt_ms = 200
  yellow_loss = 0.02 (2%), red_loss = 0.10 (10%)
```

**NFR-014:** Quality indicator must update within 2 tick cycles of path quality change.
The `internal/metrics` package is called from the path scoring loop; there is no
batching between path measurement and indicator update.

## Session Discovery (internal/discovery, BC-2.03.001–003, Phase PE)

Presence advertisement is out of E router MVP scope. Architecture sketch for PE:
- Access nodes send `PRESENCE_ADV` frames to a well-known SVTN multicast address.
- Consoles subscribe to multicast and maintain a local session list.
- Heartbeat interval: 30s default (tuning parameter; see ARCH-INDEX).
- State-change advertisements are immediate (attach/detach events).

The multicast address is scoped to the SVTN (uses `svtn_id` as part of the multicast
group derivation) to enforce DI-005 (SVTN isolation).
