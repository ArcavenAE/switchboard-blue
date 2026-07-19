---
artifact_id: BC-2.01.009
document_type: behavioral-contract
level: L3
version: "1.0"
status: draft
producer: product-owner
timestamp: 2026-07-18T00:00:00Z
phase: 1a
inputs:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/domain-spec/failure-modes.md'
  - '_bmad-output/planning-artifacts/prd.md'
  - 'decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md'
  - 'decisions/identity-cluster-architecture.md'
input-hash: "929cfd4"
extracted_from: null
bc_id: BC-2.01.009
subsystem: session-networking
architecture_module: cmd/switchboard
capability: CAP-003
priority: P0
criticality: critical
scope_phase: E
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified:
  - version: "1.0"
    date: 2026-07-18
    author: product-owner
    change: >
      Initial commission — NODE_IDENTIFY three-message handshake wire protocol.
      Authored per S-BL.NODE-IDENTIFY-WIRE-rulings.md §§2–9, §12, §13
      (S-BL.NODE-IDENTIFY-WIRE-rulings.md v1.1, 2026-07-18).
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
  - 'decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md'
  - 'decisions/identity-cluster-architecture.md'
traces_to: [CAP-003]
kos_anchors:
  - elem-ssh-end-to-end-encryption
  - elem-node-router-architecture
---

# Behavioral Contract BC-2.01.009: NODE_IDENTIFY Three-Message Handshake — Wire Protocol and Failure Paths

## Description

When a node connects to a router over TCP, it must prove its identity before any SVTN-scoped session traffic may flow. The admission handshake uses `control_type = 0x04` (`NODE_IDENTIFY`) and a `msg_kind` sub-byte to multiplex three messages — `NodeIdentify` (node → router), `Challenge` (router → node), and `ChallengeResponse` (node → router) — over a single synchronous exchange on the raw TCP connection. All three messages are handled by the `onAccept` closure in `runRouter`, directly on `net.Conn`, before `netingress.ServeConn` starts reading. On success, `Router.BindInterface` records the `(SVTNID, NodeAddr) → IfaceID` binding (BC-2.01.010) and normal frame routing begins. Any error at any step closes the connection immediately — fail-closed.

The three messages consume exactly one opcode registry entry: `NODE_IDENTIFY = 0x04` (BC-2.01.008 Postcondition 2). Sub-protocol differentiation is via `msg_kind` at payload offset `[2]`.

## Preconditions

1. A TCP connection has been accepted by the router's `netingress` listener.
2. The `onAccept` closure has fired as the first act of the per-connection goroutine, before `ServeConn` begins reading.
3. The `Router`'s `AdmittedKeySet` may or may not contain the connecting node's public key — the handshake determines admission.
4. `nodeIdentifyHandshakeTimeout = 10 * time.Second` is set via `conn.SetDeadline` before the first read (implementation reference: `handshakeTimeout` pattern in `admissionSyncClient`).
5. `SVTNID` in the `NodeIdentify` outer header must be non-zero.

## Postconditions

### Success path

1. **Message 1 — `NodeIdentify` (node → router):** The router reads a 44-byte outer header + 36-byte payload. Outer header: `frame_type = 0x03` (FrameTypeCtl), `payload_len = 36`, `svtn_id` = target SVTN (non-zero), `src_addr` and `dst_addr` are zero, `hmac_tag` is zero. Payload: `control_type = 0x04`, `version = 0x01`, `msg_kind = 0x01`, `reserved = 0x00`, `node_pubkey [32 bytes]` (Ed25519 public key, `ed25519.PublicKeySize`). Total frame: 80 bytes.

2. **Router derives NodeAddr and generates Challenge:** On valid `NodeIdentify`, the router derives `nodeAddr = frame.DeriveNodeAddress(hdr.SVTNID, payload[4:36])` and calls `admission.GenerateChallenge(routerPrivKey)` to produce a 32-byte crypto/rand nonce and `RouterSig = ed25519.Sign(routerPrivKey, nonce[:])` (64 bytes).

3. **Message 2 — `Challenge` (router → node):** The router writes a 44-byte outer header + 100-byte payload. Outer header: `frame_type = 0x03`, `payload_len = 100`, `svtn_id` echoed from `NodeIdentify`, `src_addr` and `dst_addr` are zero, `hmac_tag` is zero. Payload: `control_type = 0x04`, `version = 0x01`, `msg_kind = 0x02`, `reserved = 0x00`, `nonce [32 bytes]`, `router_sig [64 bytes]`. Total frame: 144 bytes.

4. **Message 3 — `ChallengeResponse` (node → router):** The router reads a 44-byte outer header + 68-byte payload. Outer header: `frame_type = 0x03`, `payload_len = 68`, `svtn_id` unchanged, `src_addr` and `dst_addr` are zero, `hmac_tag` is zero. Payload: `control_type = 0x04`, `version = 0x01`, `msg_kind = 0x03`, `reserved = 0x00`, `nonce_sig [64 bytes]` = `ed25519.Sign(nodePrivKey, challenge.Nonce[:])`. Total frame: 112 bytes.

5. **`AdmitNode` called:** The router calls `admission.AdmitNode(challenge, resp, pubKey, hdr.SVTNID, ks)` which verifies `resp.NonceSig` = `ed25519.Verify(pubKey, challenge.Nonce[:], resp.NonceSig)` and checks that the key is registered, not revoked, and not expired (BC-2.05.001 Postconditions 3–7). On `nil` return, admission is granted.

6. **`Router.BindInterface` called:** On `AdmitNode` success, the `onAccept` closure calls `Router.BindInterface(hdr.SVTNID, nodeAddr, h.IfaceID)` (BC-2.01.010). The `(SVTNID, NodeAddr) → IfaceID` binding is recorded.

7. **Handshake deadline cleared:** `conn.SetDeadline(time.Time{})` clears the per-connection deadline set in Precondition 4. Normal frame routing via `ServeConn` begins.

8. **Connection lifecycle invariant:** After `onAccept` returns, the connection is in one of two states: (a) fully bound — `Router.BindInterface` called, `sendMap` entry live, `ServeConn` running; or (b) closed. There is no "unbound but open" state.

### Failure paths

9. **Failure posture:** Any error at any step during the three-message exchange closes the connection immediately. The error code / log message for each failure path is listed in the Error Codes section below. No error or status is returned to the connecting node beyond connection closure.

## Invariants

1. **Single opcode, sub-byte discrimination:** All three messages share `control_type = 0x04` (`NODE_IDENTIFY`). Message type is identified by `msg_kind` at payload offset `[2]`. This is the sub-protocol model defined by BC-2.01.008 Postcondition 2 and Ruling §2.

2. **Zero HMACTag in all three messages:** There is no `FrameAuthKey` available before the handshake completes — none has been established yet. The challenge-response IS the authentication mechanism. Zero `HMACTag` is correct and consistent with the DRAIN / DISCOVERY_RELAY precedent (BC-2.01.008 Verified Premises).

3. **SVTNID must be set in all three messages:** Unlike DRAIN (global broadcast), `NODE_IDENTIFY` is SVTN-scoped. The SVTN ID in the outer header is required for `AdmitNode` keyset scoping. A zero SVTN ID is rejected at the `NodeIdentify` decoding step.

4. **SrcAddr and DstAddr are zero in all three messages:** Addresses in the outer header identify nodes on the data plane; control frames do not route through the forwarding table. The node's identity is in the payload (`node_pubkey`), not the outer header.

5. **Exact payload lengths are enforced:** All three messages have fixed payload sizes (36, 100, 68 bytes respectively). Any deviation is treated as a malformed frame and closes the connection. The `reserved` byte at payload offset `[3]` MUST be `0x00`; a non-zero reserved byte is a hard decoder error.

6. **Handshake dispatcher is `onAccept`, not `route`:** The three messages are handled before `ServeConn` starts. The `route` closure (stateless frame dispatcher) does not see these messages. This is consistent with the `onAccept` architecture established by `netingress.Serve`.

7. **Second `NodeIdentify` on the same connection is a hard error:** If an already-admitted connection sends a second `NODE_IDENTIFY` frame, the router closes the connection and logs E-ADM-023. A well-behaved node never does this; this is a fail-closed guard against application-layer protocol violations. This is distinct from the reconnect case (new TCP connection), which is governed by BC-2.01.010 LWW rebind semantics.

8. **Eventual-consistency race is not a protocol defect:** If a node connects before its `RegisterKey` push has been processed by the router (BC-2.05.009), `AdmitNode` returns `ErrNotAdmitted`. This is indistinguishable from any other not-admitted path. The correct disposition is connection closure; retry by the node after backoff resolves the race. No special handling is required in the handshake.

## Trigger

A new TCP connection is accepted by the router's `netingress` listener, causing `onAccept` to fire.

## Error Codes

| Code | Name | Trigger | Connection Disposition |
|------|------|---------|----------------------|
| (WARN log) | malformed NodeIdentify frame | `hdr.PayloadLen != 36`, wrong `control_type` / `version` / `msg_kind` / `reserved` byte | Close immediately |
| (WARN log) | zero SVTN ID | All-zero bytes in outer header SVTN ID field at `NodeIdentify` receipt | Close immediately |
| E-ADM-001 | signature verification failed | `NonceSig` does not verify against pubkey in `AdmitNode` | Close immediately |
| E-ADM-003 | not admitted | Node's pubkey not registered for this SVTN in `AdmitNode` | Close immediately |
| E-ADM-005 | key revoked | Node's key has been revoked in `AdmitNode` | Close immediately |
| E-ADM-008 | nonce replay | Challenge nonce already consumed in `AdmitNode` | Close immediately |
| E-ADM-015 | key expired | Key expiry is set and past in `AdmitNode` (BC-2.05.001 Postcondition 6) | Close immediately |
| E-ADM-022 | handshake timeout | 10s elapsed without completing three-message exchange (`conn.SetDeadline` fires) | Close immediately |
| E-ADM-023 | duplicate NodeIdentify | Second `NodeIdentify` frame received on an already-handshaken connection | Close immediately |
| (WARN log) | malformed ChallengeResponse frame | `hdr.PayloadLen != 68`, wrong discriminators at `ChallengeResponse` receipt | Close immediately |

> **Note:** E-ADM-022 and E-ADM-023 are `cmd/switchboard`-scope wire-protocol error codes; they describe wire-handler violations, not `internal/admission` keyset semantics. E-ADM-001, -003, -005, -008, -015 are re-used from `internal/admission` (same codes as `ReAuthenticate`).

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Eventual-consistency race: node connects before its `RegisterKey` push is processed | `AdmitNode` returns `ErrNotAdmitted` (E-ADM-003); connection closed. Node retries after backoff. Self-resolving when push arrives. Not a protocol defect. |
| EC-002 | Handshake timeout: connection stalls for 10s without completing | `conn.SetDeadline` fires; `io.ReadFull` returns a deadline error; connection closed with E-ADM-022 warning log. |
| EC-003 | Reserved byte non-zero (`payload[3] != 0x00`) | Hard decoder error; connection closed. A pre-admission connection with ambiguous handshake state is not tolerated. |
| EC-004 | `NodeIdentify` arrives on an already-admitted connection | Hard error (E-ADM-023); connection closed immediately. Second handshake on same TCP conn is an application-level protocol violation. |
| EC-005 | Node's key is expired at connect time | `AdmitNode` returns `ErrKeyExpired` (E-ADM-015); connection closed. See BC-2.05.001 Postcondition 6 / Invariant 5. |
| EC-006 | Node's key is not registered for the SVTN in the outer header `SVTNID` | `AdmitNode` returns `ErrNotAdmitted` (E-ADM-003); connection closed. |
| EC-007 | Two successive reconnects from the same node (new TCP each time) | Each reconnect completes a full three-message handshake on its new TCP connection. `Router.BindInterface` uses LWW semantics (BC-2.01.010 PC-2). Rebind is only possible after `AdmitNode` returns nil — full re-handshake required. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Valid admitted key; correct `NonceSig`; non-zero `SVTNID` | Handshake succeeds; `BindInterface` called; `ServeConn` begins | happy-path |
| Valid `NodeIdentify` frame but key not registered (`ErrNotAdmitted`) | Connection closed; E-ADM-003 logged | error |
| Valid `NodeIdentify` frame; key revoked (`ErrKeyRevoked`) | Connection closed; E-ADM-005 logged | error |
| Valid `NodeIdentify` frame; key expired (`ErrKeyExpired`, Postcondition 6) | Connection closed; E-ADM-015 logged | error |
| `NodeIdentify` with wrong `NonceSig` (signature mismatch) | Connection closed; E-ADM-001 logged | error |
| `NodeIdentify` with previously-consumed nonce (`ErrNonceReplay`) | Connection closed; E-ADM-008 logged | error |
| `NodeIdentify` with `payload_len != 36` | Connection closed; malformed frame warning logged | error |
| `NodeIdentify` with `payload[3] != 0x00` (non-zero reserved byte) | Connection closed; malformed frame warning logged | error |
| `NodeIdentify` with zero SVTN ID in outer header | Connection closed; zero SVTN ID warning logged | error |
| 10s elapses before `ChallengeResponse` arrives | Connection closed; E-ADM-022 timeout warning logged | error |
| Second `NodeIdentify` on already-admitted connection | Connection closed; E-ADM-023 duplicate warning logged | error |
| Node connects twice (new TCP each time); key admitted | Both handshakes succeed; second `BindInterface` call LWW-overwrites first binding | rebind |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| test-as-evidence | Three-message exchange completes; `BindInterface` called on admitted key | integration (in-process router + node stubs) |
| test-as-evidence | All enumerated failure paths (EC-001 through EC-007) close the connection | unit / integration per path |
| test-as-evidence | After `onAccept` returns, connection is either fully bound or closed — no unbound-open state | integration |
| test-as-evidence | Second `NodeIdentify` on same connection triggers E-ADM-023 and closes connection | unit |
| test-as-evidence | Handshake timeout (10s) closes connection with E-ADM-022 | unit (mock deadline) |
| test-as-evidence | `payload[3] != 0x00` (non-zero reserved byte) closes connection | unit |
| test-as-evidence | Zero SVTN ID in outer header closes connection | unit |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-003 ("Frame envelope encoding and decoding") per capabilities.md §CAP-003 |
| L2 Domain Invariants | DI-001 (carrier-grade content separation — terminal-consumer carve-out), DI-002 (private keys never transit), DI-007 (layout stability within major version) |
| Architecture Module | cmd/switchboard (`onAccept` closure in `runRouter`; NODE_IDENTIFY handler) |
| Stories | S-BL.NODE-IDENTIFY-WIRE |
| Capability Anchor Justification | CAP-003 ("Frame envelope encoding and decoding") per capabilities.md §CAP-003 — this BC specifies the wire protocol for `control_type = 0x04` frames, which are a sub-class of the ctl frame envelope defined by CAP-003 / BC-2.01.008 |

## Related BCs

- BC-2.01.008 — authority for: `control_type = 0x04` opcode registry row; outer header layout; silent-ignore / fail-closed rules for ctl frames
- BC-2.05.001 — composes with: `AdmitNode` is invoked at Postcondition 5 of this BC; Postcondition 6 (ErrKeyExpired) applies here
- BC-2.01.010 — composes with: on success (Postcondition 6), `Router.BindInterface` is called to record the `(SVTNID, NodeAddr) → IfaceID` binding; BC-2.01.010 governs that binding's lifecycle
- BC-2.01.007 — related to: re-authentication (`ReAuthenticate`) uses a similar challenge-response pattern; both gates enforce expiry symmetrically per BC-2.05.001 Invariant 5

## Architecture Anchors

- decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md §§2–9 (wire format for all three messages, outer header, payload layouts, msg_kind sub-byte)
- decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md §7 (handshake sequence diagram, `onAccept` dispatcher)
- decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md §12 (Obligation 3 — rebind / same-connection second NodeIdentify)
- decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md §13 (Obligation 4 — timeout value, failure path table, eventual-consistency race)
- decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md §15 (O-1 — AdmitNode expiry check)

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-07-18 | Initial commission — NODE_IDENTIFY three-message handshake: wire format (§§2–9), handshake sequence (§7), failure paths (§13), timeout (§13), second-NodeIdentify hard error (§12), eventual-consistency race disposition (§13), `AdmitNode` expiry check (§15). All sourced from S-BL.NODE-IDENTIFY-WIRE-rulings.md v1.1. |
