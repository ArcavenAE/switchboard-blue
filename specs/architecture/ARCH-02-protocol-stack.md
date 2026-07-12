---
artifact_id: ARCH-02-protocol-stack
document_type: architecture-section
level: L3
version: "1.2"
status: draft
producer: architect
timestamp: 2026-06-23T00:00:00
phase: 1b
traces_to: ARCH-INDEX.md
inputDocuments:
  - '.factory/specs/prd.md'
  - '.factory/specs/behavioral-contracts/ss-01/BC-2.01.001.md'
  - '.factory/specs/behavioral-contracts/ss-01/BC-2.01.002.md'
  - '.factory/specs/behavioral-contracts/ss-01/BC-2.01.003.md'
  - '.factory/specs/behavioral-contracts/ss-01/BC-2.01.004.md'
  - '.factory/specs/behavioral-contracts/ss-01/BC-2.01.005.md'
  - '.factory/specs/behavioral-contracts/ss-01/BC-2.01.006.md'
  - '.factory/specs/behavioral-contracts/ss-01/BC-2.01.007.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '_bmad-output/planning-artifacts/prd.md'
kos_anchors:
  - elem-timeslice-framing
  - elem-asymmetric-half-channels
  - elem-ssh-end-to-end-encryption
modified:
  - 2026-07-11T00:00:00 # v1.2 — S-7.04-FU-DRAIN-WIRE: §"Outer Header Format" frame_type table gains a note that ctl (0x03) payloads carry a control_type byte discriminator (opcodes 0x01=DRAIN, 0x02=RESYNC reserved); BC-2.01.008 is the schema home. Refs: S-7.04-FU-DRAIN-WIRE + code branch feature/S-7.04-FU-DRAIN-WIRE@e7614d7 — the code lands on the feature branch, structurally a different branch from this spec-doc commit, so the wire-format spec-pair obligation is satisfied as same-delivery-burst rather than same-commit.
  - 2026-07-11T00:00:00 # v1.1 — S-BL.PE-RECEIVE-LOOP: §"Outer Header Format" frame_type row amended to add pe_connect=0x06 (same-commit parallel obligation with FrameTypePEConnect definition in frame.go). Refs: S-BL.PE-RECEIVE-LOOP + c316aed.
  - 2026-06-23T00:00:00
---

# ARCH-02: Protocol Stack

## ADR-001: HMAC Algorithm Decision

**Decision:** HMAC-SHA256 with a per-node, per-SVTN derived key (see ARCH-04
for key derivation). The hash function is SHA-256 from Go stdlib `crypto/sha256`.
No Blake3 — no new transitive dependencies.

**Constraints considered:**
- Outer header is 44 bytes fixed (BC-2.01.004, DI-007). HMAC tag must fit within
  this constraint. An 8-byte truncated HMAC tag (first 8 bytes of HMAC-SHA256
  output) fits the 44-byte layout exactly.
- HMAC-SHA256 produces a 32-byte tag. The outer header carries the first 8 bytes
  as the integrity-check tag for router-path authentication. This is standard
  truncation practice (RFC 2104 §5 permits truncation to 80 bits minimum; 64 bits
  is used here as a router-path integrity signal — not a standalone security
  primitive. Full frame authentication binds per-node keying per ARCH-04 F-003).
- Per-node key derivation: see ARCH-04 HMAC Keying section. The router derives a
  unique `frame_auth_key` per admitted node per SVTN using HKDF-SHA256.

**Rejected alternatives:**
- HMAC-BLAKE3: adds a dependency not already in the crypto stack.
- HMAC-SHA512: 64-byte output, requires 32-byte field; unnecessary header expansion.
- Poly1305: MAC-only, no existing key hierarchy that fits the admission model.
- 16-byte HMAC field: does not fit 44-byte outer header sum (see layout below).

**ADR-001 reference:** `kos_anchors: [elem-ssh-end-to-end-encryption]`

## ADR-008: Tick Interval Range

**Decision:** Tick interval range is 5–50ms, validated as a tuning parameter.
Default upstream 10ms, downstream 50ms. These are config-settable. Phase 3
benchmarks validate whether any tick rate meets NFR-001 (100ms p99 LAN target).
The architecture does not lock in specific tick values — the mechanism is bedrock;
the parameters are tuning.

## Outer Header Format (44 bytes, fixed — DI-007, BC-2.01.004)

This is the canonical single source of truth for the outer header wire format.
Any field position or size change requires a major version bump (DI-007).

| Offset | Size | Field | Notes |
|--------|------|-------|-------|
| 0 | 1 | version | bits[7:4]=major (0–15), bits[3:0]=minor (0–15). For v0.1: 0x01. |
| 1 | 1 | frame_type | u8 enum: data=0x01, empty_tick=0x02, ctl=0x03, arq=0x04, fec=0x05, pe_connect=0x06 |
| 2 | 2 | payload_len | u16 big-endian; byte count of everything after the outer header (channel header + payload) |
| 4 | 16 | svtn_id | 128-bit SVTN identifier |
| 20 | 8 | src_node_addr | 64-bit source node address |
| 28 | 8 | dst_node_addr | 64-bit destination node address |
| 36 | 8 | hmac_tag | first 8 bytes of HMAC-SHA256 output over the full frame |
| **Total** | **44** | | |

**Field arithmetic:** 1 + 1 + 2 + 16 + 8 + 8 + 8 = **44 bytes.** DI-007 satisfied.

**`ctl` payload discriminator (BC-2.01.008):** `ctl` (0x03) payloads carry a
`control_type` byte as the first byte of the payload — a discriminator
distinguishing control-message sub-types. Opcodes: `0x01 = DRAIN`, `0x02 =
RESYNC` (reserved). BC-2.01.008 is the schema home for `control_type`.

**No outer-header sequence field.** Per-half-channel sequence (`chan_seq`) lives
exclusively in the channel header (see below). There is no sequence field in the
outer header.

**Version encoding:** 1-byte packed field. Bits [7:4] encode major version (0–15).
Bits [3:0] encode minor version (0–15). For protocol version 0.1, the version byte
is `0x01` (major=0, minor=1). For protocol version 1.0, the version byte is `0x10`.
`ParseOuterHeader` checks the major nibble first; major version mismatch → E-PRT-001
before any further parsing.

**HMAC tag position:** The `hmac_tag` field occupies bytes 36–43 (8 bytes). It
carries the first 8 bytes of the HMAC-SHA256 output computed over the entire frame
(outer header bytes 0–35 concatenated with the channel header and payload). The
HMAC input excludes bytes 36–43 (the tag field itself, treated as zeros during
computation). Full HMAC key derivation is specified in ARCH-04.

**DI-007 enforcement:** Header layout is version-locked. Extension fields belong
in the channel header TLV, not the outer header. Endpoint evolution does not require
router upgrades.

**NFR-006 enforcement:** Version byte is at offset 0; major version check is the
first operation in `internal/frame.ParseOuterHeader`. Mismatch → E-PRT-001 before
any other parsing.

## Channel Header (router-opaque — BC-2.01.005)

The channel header follows immediately after the outer header. Its format is
defined by the endpoint protocol and is invisible to routers: routers skip
`payload_len` bytes without inspecting content. The channel header is part of
those `payload_len` bytes.

| Offset | Size | Field | Notes |
|--------|------|-------|-------|
| 0 | 4 | chan_id | u32; identifies the half-channel (upstream vs downstream) |
| 4 | 4 | chan_seq | u32 big-endian; per-half-channel sequence number, increments by 1 per tick |
| 8 | 1 | flags | bit 0=FEC_present, bit 1=ARQ_req, bit 2=SACK_present |
| 9 | 3 | reserved | must be zero |
| 12 | 8 | sack_bitmap | **conditional**: present only when flags bit 2 (SACK_present) is set; covers 64 sequence slots |
| **Total** | **12** or **20** | | 12 bytes fixed; +8 bytes when SACK_present=1 |

**Channel header timestamp and FEC metadata decision (canonical):** The channel header does NOT carry a sender timestamp field — timestamping is end-to-end and lives in the SSH-encrypted payload. FEC metadata is signaled by `flags` bit 0 (FEC_present) only; FEC group seq/parity coefficients live in the payload header of FEC frames (frame_type=fec), not in the channel header. This keeps the channel header router-opaque and avoids version coupling between FEC implementation and the channel header format.

**SACK bitmap location (F-012):** The SACK bitmap is embedded in the channel header
as a conditional 8-byte field, present when `flags` bit 2 is set. This avoids a
separate ARQ control message for acknowledgement. When SACK_present=0, the channel
header is 12 bytes. When SACK_present=1, the channel header is 20 bytes. The
`payload_len` field in the outer header accounts for the actual channel header size.

**chan_seq:** This is the only sequence field in the protocol. There is no sequence
field in the outer header. Per-half-channel ordering and deduplication are performed
by the endpoint using `chan_seq`. Routers do not read or modify `chan_seq`.

**chan_seq initial value and seq=0 reservation (RULING-001):** chan_seq starts at 1.
The value 0 is reserved and is never a valid wire-frame sequence number. On wrap
from MaxUint32, the next value is 1 (not 0): senders MUST skip 0 on wrap. A
received frame with chan_seq=0 is malformed and MUST be discarded without delivery.
This reservation makes seq=0 a safe "unset/none" sentinel in receiver-side data
structures (e.g., DegradationEvent.DroppedSeq=0 means "no event").

**chan_seq session-lifetime assumption (RULING-001):** 32-bit sequence wraparound
across an active session is not a supported scenario for MVP. At a 10ms tick rate,
the 32-bit space (values 1–MaxUint32) wraps after approximately 497 days; at 100Hz
it wraps after approximately 49 days. Sessions are assumed to terminate before
wraparound. Receiver-side comparison loops (e.g., cumulative-ACK scan) need not
handle the MaxUint32→1 boundary for MVP. This assumption is documented in
BC-2.02.002 (EC-004) and BC-2.02.004 (EC-005).

## Half-Channel Architecture (internal/halfchannel)

Per elem-timeslice-framing and elem-asymmetric-half-channels, each direction is
an independent state machine:

```
Upstream half-channel (keystrokes):
  tick_interval = config.TickIntervalUpstream (default 10ms)
  time.NewTicker → on each tick:
    payload = dequeueUpstream(replay_window)
    frame = buildFrame(payload)
    send(frame)

Downstream half-channel (terminal output):
  tick_interval = config.TickIntervalDownstream (default 50ms)
  time.NewTicker → on each tick:
    payload = dequeueDownstream(arq_state)
    frame = buildFrame(payload)
    send(frame)
```

**DI-008 enforcement:** `time.NewTicker` with compensation for drift. The ticker
fires every tick_interval; frames are sent even if payload is empty. Skipping is
not allowed.

**NFR-009:** Timer deviation target is ≤ 2ms p99. Achieved by:
1. `time.NewTicker` (not `time.Sleep`) — uses OS timer, self-corrects.
2. No heap allocation in the hot path (pre-allocated frame pool).
3. Compensation: if wakeup is late by > half a tick, the missed tick fires immediately;
   the *next* tick is scheduled from the original base, not the late wakeup.

## Session Identity (BC-2.01.006, BC-2.01.007)

Node address = `sha256(svtn_id || public_key_bytes)[0:8]` (64-bit, encoded as hex).

SHA-256 from Go stdlib `crypto/sha256` is used for both address derivation and HMAC.
This eliminates the Blake3 transitive dependency. Address derivation is not
security-critical (collisions are operational inconvenience, not a security break);
consistent use of SHA-256 simplifies the crypto inventory.

**BC-2.01.007 (IP mobility):** On IP address change, node reconnects with same keypair.
Challenge-response re-authenticates the cryptographic identity. Session continues with
same `src_node_addr` / `dst_node_addr` pair; the network-layer reconnect is transparent
to endpoints.

## Noise Protocol (PE Phase, not MVP)

Router-to-router peering in PE phase uses Noise XX pattern for mutual authentication.
Key derivation from Noise handshake feeds `HKDF-SHA256` consistent with ADR-001.
The Noise dependency is not imported in the MVP binary; gated by `upstream_routers` config.

## Risk Mitigations

| Risk | Mitigation in This Section |
|------|---------------------------|
| R-005 (protocol version incompatibility) | Version byte 0; major version check is first parse op |
| R-001 (content separation) | Channel header defined as router-opaque; inner payload is `[]byte` in router code, never typed |
