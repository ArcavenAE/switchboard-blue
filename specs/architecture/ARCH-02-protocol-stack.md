---
artifact_id: ARCH-02-protocol-stack
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
---

# ARCH-02: Protocol Stack

## ADR-001: HMAC Algorithm Decision

**Decision:** HMAC-SHA256 with a per-SVTN derived key.

**Constraints considered:**
- Outer header is 44 bytes fixed (BC-2.01.004). HMAC must fit without expanding header.
- The Noise Protocol Framework (for PE router-to-router handshake) derives keys via
  HKDF-SHA256. Using SHA-256 as the HMAC hash function aligns with the key derivation
  algorithm, avoiding dual-hash implementations.
- HMAC-SHA256 produces a 32-byte tag. The outer header HMAC field is 16 bytes (truncated
  to first 16 bytes of HMAC-SHA256 output), consistent with standard truncation practice
  (RFC 2104 §5: "The MAC length can be chosen as 80 bits or more"). 16 bytes = 128 bits,
  well above the 80-bit minimum.
- Per-SVTN key derivation: `hmac_key = HKDF-SHA256(router_master_key, svtn_id || "hmac-frame-auth")`.
  This scopes the key to the SVTN without requiring per-node keys in the router's hot path.

**Rejected alternatives:**
- HMAC-BLAKE2: adds a dependency not already in the crypto stack.
- HMAC-SHA512: 64-byte output, requires 32-byte field; unnecessary header expansion.
- Poly1305: MAC-only, no existing key hierarchy that fits the admission model.

**ADR-001 reference:** `kos_anchors: [elem-ssh-end-to-end-encryption]`

## ADR-008: Tick Interval Range

**Decision:** Tick interval range is 5–50ms, validated as a tuning parameter.
Default upstream 10ms, downstream 50ms. These are config-settable. Phase 3
benchmarks validate whether any tick rate meets NFR-001 (100ms p99 LAN target).
The architecture does not lock in specific tick values — the mechanism is bedrock;
the parameters are tuning.

## Outer Header Format (44 bytes, fixed, BC-2.01.004)

```
Offset  Size  Field
──────  ────  ──────────────────────────────────────
0       1     version (major.minor encoded as 0xMN)
1       1     frame_type (data=0x01, empty=0x02, ctl=0x03, arq=0x04, fec=0x05)
2       2     flags (reserved; must be 0x0000 for v1)
4       16    svtn_id (128-bit SVTN identifier)
20      8     dst_addr (64-bit node address = hash(svtn_id, pubkey)[0:8])
28      8     src_addr (64-bit node address)
36      2     payload_len (length of channel header + payload)
38      2     reserved (must be 0x0000 for v1)
40      4     sequence (32-bit frame sequence number per half-channel)
44...   N     channel_header + payload (router-opaque; DI-001, BC-2.01.005)
```

**DI-007 enforcement:** Header layout is version-locked. Any field position or size
change requires a major version bump. Extension fields are in the channel header TLV
(not the outer header) to allow endpoint evolution without router upgrades.

**NFR-006 enforcement:** Version field is at byte 0; major version check is the
first operation in `internal/frame.ParseOuterHeader`. Mismatch → E-PRT-001 before
any other parsing.

## Channel Header (router-opaque, BC-2.01.005)

The channel header follows the outer header. Its format is defined by the endpoint
protocol and is invisible to routers (they skip `payload_len` bytes). Minimum fields:

```
Offset  Size  Field
──────  ────  ──────────────────────────────────────
0       2     channel_id (identifies upstream vs downstream half-channel)
2       4     chan_seq (32-bit sequence within the half-channel)
6       8     timestamp_us (microseconds since epoch; for RTT measurement)
14      1     fec_flags (FEC group marker; 0x00 in MVP)
15      1     arq_flags (ARQ/SACK present; see ARCH-03)
16...   TLV   extension fields (type-length-value; router-transparent)
```

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

Node address = `blake3(svtn_id || public_key_bytes)[0:8]` (64-bit, encoded as hex).

Blake3 chosen over SHA-256 for address derivation (not HMAC — different function):
- 64-bit output is a truncation of the full hash; Blake3 is faster at short outputs.
- Address derivation is not security-critical (collisions are operational inconvenience,
  not a security break); HMAC-SHA256 is reserved for the trust-critical HMAC path.

**BC-2.01.007 (IP mobility):** On IP address change, node reconnects with same keypair.
Challenge-response re-authenticates the cryptographic identity. Session continues with
same `src_addr` / `dst_addr` pair; the network-layer reconnect is transparent to endpoints.

## Noise Protocol (PE Phase, not MVP)

Router-to-router peering in PE phase uses Noise XX pattern for mutual authentication.
Key derivation from Noise handshake feeds `HKDF-SHA256` consistent with ADR-001.
The Noise dependency is not imported in the MVP binary; gated by `upstream_routers` config.

## Risk Mitigations

| Risk | Mitigation in This Section |
|------|---------------------------|
| R-005 (protocol version incompatibility) | Version byte 0; major version check is first parse op |
| R-001 (content separation) | Channel header defined as router-opaque; inner payload is `[]byte` in router code, never typed |
