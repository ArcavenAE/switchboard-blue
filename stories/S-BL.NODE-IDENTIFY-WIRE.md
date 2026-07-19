---
artifact_id: S-BL.NODE-IDENTIFY-WIRE
document_type: story
level: ops
story_id: S-BL.NODE-IDENTIFY-WIRE
version: "1.6"
title: "NODE_IDENTIFY wire: connect-time identify handshake binding (SVTNID, NodeAddr) → IfaceID for hop-2 fan-out target resolution"
status: ready
producer: story-writer
timestamp: 2026-07-14T00:00:00Z
modified: 2026-07-18T00:00:00Z
phase: 2
epic: E-7
wave: backlog
priority: P2
scope_phase: E
estimated_points: 10
points: 10
inputs:
  - 'decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md'
  - 'specs/behavioral-contracts/ss-01/BC-2.01.009.md'
  - 'specs/behavioral-contracts/ss-01/BC-2.01.010.md'
  - 'specs/behavioral-contracts/ss-05/BC-2.05.001.md'
  - 'specs/behavioral-contracts/ss-01/BC-2.01.008.md'
input-hash: "a252659"
traces_to: "decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md"
epic_id: E-7
behavioral_contracts:
  - BC-2.01.008
  - BC-2.01.009
  - BC-2.01.010
  - BC-2.05.001
bc_traces:
  - BC-2.01.008
  - BC-2.01.009
  - BC-2.01.010
  - BC-2.05.001
verification_properties: []
subsystems: [session-networking, admission-security]
target_module: "cmd/switchboard"
architecture_modules:
  - internal/routing
  - internal/admission
  - cmd/switchboard
tdd_mode: strict
cycle: v1.0.0-greenfield
estimated_days: null
assumption_validations: []
risk_mitigations: []
depends_on: [S-BL.ADMISSION-SYNC-WIRE, S-BL.NODE-ADMISSION-PROVISIONING]
blocks: []
acceptance_criteria_count: 13
rulings_doc: "decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md"
provenance:
  origin: "S-BL.DISCOVERY-WIRE Ruling 3(f) Forward Obligation — story-ready human gate disposition, 2026-07-14"
  spec_annotation: "S-BL.DISCOVERY-WIRE-rulings.md v1.9, Ruling 3(f) subsection item (j) — the human gate disposition naming and scoping this story"
  adjudication: "S-BL.DISCOVERY-WIRE-fanout-options.md v1.1 Option 1 selected at the story-ready human gate — Option 1's NODE_IDENTIFY handshake mechanism delivered via Option 3's name-and-schedule-now shape"
inputDocuments:
  - 'decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md'   # v1.1 — BINDING. All obligations resolved. Wire format (§§2–9): control_type=0x04 with msg_kind sub-byte at payload[2]; NodeIdentify(80B)/Challenge(144B)/ChallengeResponse(112B) frame layouts; outer header fields; handshake sequence (§7); Router.BindInterface/LookupInterface/UnbindInterface signatures (§8); bounds guards (§9). Obligation 3 (§12): LWW overwrite on reconnect; prior connection self-removing via stale cleanup guard; second NodeIdentify on same connection = hard error E-ADM-023. Obligation 4 (§13): nodeIdentifyHandshakeTimeout=10s (conn.SetDeadline in onAccept); failure path table with E-ADM-022/-023 new codes; eventual-consistency race → ErrNotAdmitted+retry. Obligations 5/6 resolved-by-delivery (§14): PR #126 (admission-sync router keyset populated) + PR #125 (node keypair provisioned). O-1 (§15): AdmitNode MUST gain expiry check (Option A, mirrors ReAuthenticate); BC-2.05.001 amendment required; human-ratified 2026-07-18.
  - 'specs/behavioral-contracts/ss-01/BC-2.01.009.md'  # v1.0 — NODE_IDENTIFY three-message handshake wire protocol; all failure paths (PC/Invariants/Error Codes/Edge Cases/Test Vectors); single opcode/msg_kind model; zero HMACTag in all frames; SVTNID mandatory non-zero; exact payload lengths enforced; second NodeIdentify Invariant 7; eventual-consistency EC-001.
  - 'specs/behavioral-contracts/ss-01/BC-2.01.010.md'  # v1.0 — BindInterface binding lifecycle: BindInterface(LWW on reconnect, write-lock), LookupInterface((InterfaceID,bool) value return, read-lock), UnbindInterface(stale cleanup guard). identityIfaceMap field on Router. All three methods protected by r.mu. Prior connection NOT actively torn down on LWW overwrite.
  - 'specs/behavioral-contracts/ss-05/BC-2.05.001.md'  # v1.2 — AdmitNode admission postconditions including NEW Postcondition 6 (ErrKeyExpired/E-ADM-015 when expiry set and past) + Invariant 5 (symmetric expiry enforcement across AdmitNode and ReAuthenticate). O-1 ruling human-ratified 2026-07-18. Implementation anchor: expiry check after snap.revoked read, before write-lock acquire, mirroring ReAuthenticate.
  - 'specs/behavioral-contracts/ss-01/BC-2.01.008.md'  # v1.3 — NODE_IDENTIFY=0x04 row already in PC-2 registry (Obligation 1 RESOLVED). No further edits to this BC required by this story.
---

# S-BL.NODE-IDENTIFY-WIRE: NODE_IDENTIFY Wire — Connect-Time Identify Handshake, Fan-Out Target Resolution

> **STATUS: READY FOR IMPLEMENTATION.** All six Open Design Obligations and O-1 resolved.
> Both prerequisite stories delivered: `S-BL.ADMISSION-SYNC-WIRE` (PR #126 @ 92a2c65) and
> `S-BL.NODE-ADMISSION-PROVISIONING` (PR #125 @ ce06f6a). 13 ACs, 10 points, TDD strict.

> **SCOPE NOTE:** The original story stub stated "introduces zero changes to `internal/admission`."
> This assertion is **superseded** by the O-1 ruling (rulings §15, human-ratified 2026-07-18):
> `internal/admission/admission.go`'s `AdmitNode` must gain an expiry check (4 lines, mirroring
> `ReAuthenticate`). See Task 16 and AC-013.

## Narrative

- **As a** router-mode daemon serving an SVTN
- **I want to** verify the identity of a connecting node via a `NODE_IDENTIFY` (control_type=0x04)
  challenge-response handshake that executes in the `onAccept` closure before `ServeConn` starts,
  then record the binding `(SVTNID, NodeAddr) → IfaceID` in `Router.identityIfaceMap`
- **So that** hop-2 fan-out target resolution (`S-BL.DISCOVERY-WIRE` AC-017/AC-018/Task 6) can
  resolve a node's cryptographic address to its live connection's send-map key

## Context

`S-BL.DISCOVERY-WIRE`'s Ruling 3(f) verified that hop-2 fan-out target resolution has no
production implementation today: binding a connecting node's identity (`NodeAddr`) to its live
connection's `InterfaceID`/`nodeConn` does not exist anywhere in `cmd/switchboard`. This gap
gates `S-BL.DISCOVERY-WIRE`'s AC-017 (SVTN-scoped, exclude-originator fan-out dispatch), AC-018
(relay-dispatch rate cap), and Task 6 (hop-2 fan-out dispatch).

This story delivers the full `NODE_IDENTIFY` handshake in three layers:
1. **Wire codec** (`cmd/switchboard/node_identify_wire.go`): encode/decode for the three
   messages (NodeIdentify=0x01, Challenge=0x02, ChallengeResponse=0x03 within `control_type=0x04`).
2. **Handshake driver** (extended `onAccept` closure in `runRouter`): 10s deadline, three-message
   synchronous exchange, fail-closed on any error.
3. **Identity binding** (`internal/routing/identity.go`): `Router.BindInterface`,
   `LookupInterface`, `UnbindInterface` backed by `identityIfaceMap` under `r.mu`.

Plus a small correction to **`internal/admission`** (O-1 ruling): `AdmitNode` gains the same
expiry check `ReAuthenticate` already has.

**Scope boundary.** This story does NOT implement `S-BL.DISCOVERY-WIRE`'s fan-out dispatch
(AC-017/AC-018/Task 6) — it provides the `LookupInterface` primitive those tasks consume. It
does NOT add any new config fields or management-plane RPCs. The admission keyset is populated
by `S-BL.ADMISSION-SYNC-WIRE` (delivered); the node's signing key is provisioned by
`S-BL.NODE-ADMISSION-PROVISIONING` (delivered).

## BC Anchors

| BC | Why anchored |
|----|-------------|
| BC-2.01.008 | `NODE_IDENTIFY=0x04` is already registered in PC-2 (Obligation 1 RESOLVED). No further edits. Invariant 3 (append-only sequential assignment) is satisfied. |
| BC-2.01.009 | Wire protocol for the three-message handshake: frame layouts, handshake sequence, all failure paths, error codes E-ADM-022/-023. Primary AC source for ACs 001–011. |
| BC-2.01.010 | Binding lifecycle: `BindInterface` (LWW), `LookupInterface` (value return), `UnbindInterface` (stale cleanup guard). Primary AC source for ACs 010 and 012. |
| BC-2.05.001 | `AdmitNode` postcondition PC-6 (ErrKeyExpired/E-ADM-015) and Invariant 5 (symmetric expiry). Primary AC source for AC-013. |

## Previous Story Intelligence (MANDATORY)

| Predecessor | Key Decisions | Patterns Established | Lessons Carried Forward |
|-------------|--------------|---------------------|------------------------|
| `S-BL.NODE-ADMISSION-PROVISIONING` (PR #125 @ ce06f6a) | `loadOrGenerateAdmissionKeypair` generates/loads the access node's stable Ed25519 private key at `cfg.AdmissionKeyFile`; public key in `discovery.Config.LocalNodeAdmissionPubkey` | PKCS#8 PEM load/generate pattern in `cmd/switchboard/access.go`; `runAccessWithConnector` calls `d.Run(runCtx)` in a goroutine | The connecting node's `ChallengeResponse` signing key is the private key loaded by `loadOrGenerateAdmissionKeypair`. On the router side, the challenge is verified against the public key retrieved from the admitted keyset (not the pubkey in the NodeIdentify frame — that is derived, not trusted directly). |
| `S-BL.ADMISSION-SYNC-WIRE` (PR #126 @ 92a2c65) | `wireAdmissionSyncHandlers` populates the router's `AdmittedKeySet` via four `internal.admission.*` push RPCs from control; `routerPersister` persists VLR-local JSON snapshot | `wireAdmissionSyncHandlers` called after `newMgmtServer`, before `serveMgmtServer` (F-P2L1-001 register-before-serve invariant) | The router's `AdmittedKeySet` is now non-empty after admission pushes land. `AdmitNode` will succeed when the node's key is registered for the SVTN. The eventual-consistency race (node connects before push arrives) is `ErrNotAdmitted` → close → retry; no special handling needed. |
| `S-BL.DISCOVERY-WIRE` (PR #123 @ d249f88) | `control_type=0x03` DISCOVERY_RELAY precedent; zero-HMACTag connection-trust boundary; `wireXHandlers` registration pattern | `sendMap` is `routing.InterfaceID → *nodeConn`; SVTNID is set in ctl outer headers for SVTN-scoped operations | AC-017/AC-018/Task 6 in `S-BL.DISCOVERY-WIRE` gate on `LookupInterface` being available from this story. The `sendMap.Store` call in `onAccept` must be preceded by a successful handshake — do not store an unverified node in the routing map. |
| `S-7.04-FU-DRAIN-WIRE` (PR #120 @ f73676d) | Per-node observer registration pattern; `onAccept` cleanup func is `func()` returned to `netingress.Serve`; cleanup does `sendMap.Delete(h.IfaceID)` + drain-done close | `onAccept` fires as FIRST ACT of per-conn goroutine, before `ServeConn` | The cleanup func must NOW ALSO call `r.UnbindInterface(svtnID, nodeAddr)` (BC-2.01.010 PC-8). The stale cleanup guard (PC-9: check `identityIfaceMap[svtnID][nodeAddr] == myIfaceID` before deleting) prevents a LWW-overwritten binding from being removed by the prior connection's cleanup. |

## Adjudicated Design Decisions

Transcribed from `decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md` v1.1 (binding — all obligations resolved).

### Decision 1 — Single control_type=0x04 with msg_kind sub-byte (Ruling §2)

Three messages share `control_type=0x04` (`NODE_IDENTIFY`), distinguished by `msg_kind` at
payload offset `[2]`. One opcode registry entry. The `reserved` byte at offset `[3]` is
hard-reserved (`0x00` required; non-zero is a hard decoder error).

| msg_kind | Name | Direction | Payload after 4-byte header |
|----------|------|-----------|------------------------------|
| `0x01` | NodeIdentify | node→router | `node_pubkey [32 bytes]` |
| `0x02` | Challenge | router→node | `nonce [32 bytes]` + `router_sig [64 bytes]` |
| `0x03` | ChallengeResponse | node→router | `nonce_sig [64 bytes]` |

### Decision 2 — Frame sizes and outer header (Rulings §§3–6)

| Message | Total | Header | Payload | HMACTag | SVTNID |
|---------|-------|--------|---------|---------|--------|
| NodeIdentify | 80B | 44B | 36B | zero | set |
| Challenge | 144B | 44B | 100B | zero | echoed |
| ChallengeResponse | 112B | 44B | 68B | zero | same |

`SrcAddr` and `DstAddr` are zero in all three messages. HMACTag is zero (no `FrameAuthKey`
exists before handshake completes; challenge-response IS the auth mechanism). SVTNID must be
set (non-zero) for `AdmitNode` keyset scoping.

### Decision 3 — Handshake dispatcher is onAccept (Ruling §7)

All three messages are handled in the `onAccept` closure in `runRouter`, on `net.Conn` directly,
**before** `netingress.ServeConn` starts reading. The `route` closure does NOT see these messages.
A `control_type=0x04` frame that reaches `route()` after ServeConn starts is a protocol violation
(second NodeIdentify on the same connection) → E-ADM-023 → connection closed.

### Decision 4 — Router.BindInterface method signatures (Ruling §8)

New field on `Router` in `internal/routing` (no new imports, no ARCH-08 DAG changes):
```go
identityIfaceMap map[[16]byte]map[[8]byte]InterfaceID  // protected by r.mu
```

Three methods:
```go
func (r *Router) BindInterface(svtnID [16]byte, nodeAddr [8]byte, ifaceID InterfaceID)
func (r *Router) LookupInterface(svtnID [16]byte, nodeAddr [8]byte) (InterfaceID, bool)
func (r *Router) UnbindInterface(svtnID [16]byte, nodeAddr [8]byte)
```
`BindInterface`/`UnbindInterface` hold `r.mu` write lock. `LookupInterface` holds read lock.
Return type is `(InterfaceID, bool)` value — not a pointer (go.md rule 12).

### Decision 5 — LWW rebind on reconnect; same-connection second NodeIdentify = hard error (Ruling §12)

A node reconnecting (new TCP, same admitted identity) triggers LWW overwrite via `BindInterface`
— the prior binding is overwritten. The prior TCP connection is NOT actively torn down. Its cleanup
func calls `UnbindInterface` with its own `ifaceID`; the stale cleanup guard fires if the stored
ifaceID no longer matches, preventing the new binding from being removed.

A second `NodeIdentify` arriving on the SAME already-admitted connection is a hard error: log
E-ADM-023 and close the connection.

### Decision 6 — Handshake timeout 10s; all failure paths close connection (Ruling §13)

`const nodeIdentifyHandshakeTimeout = 10 * time.Second` (matches `handshakeTimeout` in
`admission_sync_client.go:154`). Set via `conn.SetDeadline` before first read; cleared on
success. All eleven failure paths in BC-2.01.009's error table → close connection immediately.
No "unbound but open" state after `onAccept` returns.

### Decision 7 — AdmitNode expiry check (O-1, Ruling §15, human-ratified 2026-07-18)

`internal/admission.AdmitNode` gains an expiry check after the `snap.revoked` read and before
the write-lock acquire, mirroring `ReAuthenticate`:
```go
if !snap.expiry.IsZero() && time.Now().UTC().After(snap.expiry) {
    return ErrKeyExpired
}
```
This closes the gap where an expired key could be admitted at connect time even though
`ReAuthenticate` would reject it. BC-2.05.001 Postcondition 6 / Invariant 5 governs both gates.

## Acceptance Criteria

### AC-001 — Successful three-message handshake: admitted key + valid signature → binding recorded, ServeConn begins (BC-2.01.009 PC-1 through PC-7; BC-2.01.010 PC-1)

**BC Anchor:** BC-2.01.009 Postconditions 1–7 (full success path); BC-2.01.010 Postcondition 1 (binding created).

**Postconditions:**
1. A node with an admitted, non-revoked, non-expired key sends a valid `NodeIdentify` frame
   (80 bytes: `control_type=0x04`, `msg_kind=0x01`, 32-byte pubkey, non-zero SVTNID, exact sizes).
2. The router derives `nodeAddr = frame.DeriveNodeAddress(svtnID, pubkey)`, calls
   `admission.GenerateChallenge(routerPrivKey)`, and writes the `Challenge` frame (144 bytes:
   `msg_kind=0x02`, 32-byte nonce, 64-byte RouterSig).
3. The node sends a `ChallengeResponse` frame (112 bytes: `msg_kind=0x03`, 64-byte NonceSig =
   `ed25519.Sign(nodePrivKey, challenge.Nonce[:])`).
4. The router calls `admission.AdmitNode(challenge, resp, pubKey, svtnID, ks)` which returns `nil`.
5. `Router.BindInterface(svtnID, nodeAddr, h.IfaceID)` is called. `LookupInterface(svtnID, nodeAddr)`
   returns `(h.IfaceID, true)`.
6. `conn.SetDeadline(time.Time{})` clears the 10s deadline. `sendMap.Store(h.IfaceID, ...)` is
   called. `ServeConn` begins — normal frame routing is active.
7. After `onAccept` returns, the connection is in the fully-bound state (BC-2.01.009 Postcondition 8).

**Test names:**
- `TestNodeIdentifyHandshake_Success_BindingRecorded` (unit/integration: in-process router with
  populated keyset, simulated node connection; verify LookupInterface returns correct IfaceID)
- `TestNodeIdentifyHandshake_Success_ServeConnBegins` (integration: verify normal frame routing
  works after handshake completes — send a data frame; router processes it)

---

### AC-002 — Malformed NodeIdentify frame → connection closed (BC-2.01.009 Invariant 5; error code table)

**BC Anchor:** BC-2.01.009 Invariant 5 (exact payload lengths enforced); Error Code table (malformed NodeIdentify → close).

**Postconditions:**
1. If `hdr.PayloadLen != 36` on the NodeIdentify frame, the connection is closed immediately.
2. If `payload[0] != 0x04` (wrong control_type) or `payload[1] != 0x01` (wrong version) or
   `payload[2] != 0x01` (wrong msg_kind for NodeIdentify) on a 36-byte payload, the connection
   is closed immediately.
3. If `payload[3] != 0x00` (non-zero reserved byte), the connection is closed immediately.
   (BC-2.01.009 Invariant 5: reserved byte MUST be `0x00`; non-zero is a hard decoder error.)
4. In all malformed cases, a WARN log is emitted (no E-ADM-* code; wire-format violation category).

**Test names:**
- `TestNodeIdentifyHandshake_MalformedNodeIdentify_WrongPayloadLen` (payload_len=20 in outer header)
- `TestNodeIdentifyHandshake_MalformedNodeIdentify_WrongMsgKind` (msg_kind=0x02 instead of 0x01)
- `TestNodeIdentifyHandshake_MalformedNodeIdentify_NonZeroReservedByte` (reserved=0x01)

---

### AC-003 — Zero SVTN ID in NodeIdentify outer header → connection closed (BC-2.01.009 Precondition 5; error code table)

**BC Anchor:** BC-2.01.009 Precondition 5 (SVTNID must be non-zero); Error Code table (zero SVTN ID → close).

**Postconditions:**
1. If the outer header's `svtn_id` field is all-zero bytes in the NodeIdentify frame, the router
   closes the connection immediately.
2. A WARN log is emitted: `"node_identify: zero SVTN ID rejected"`.
3. A frame with a non-zero SVTN ID proceeds to the next validation step (does not trigger this path).

**Test names:**
- `TestNodeIdentifyHandshake_ZeroSVTNID_Rejected`

---

### AC-004 — ErrNotAdmitted → connection closed E-ADM-003 (BC-2.01.009 error code E-ADM-003)

**BC Anchor:** BC-2.01.009 Error Code E-ADM-003 (not admitted → close immediately).

**Postconditions:**
1. If `admission.AdmitNode` returns `ErrNotAdmitted` (node's pubkey not registered for this
   SVTN in the router's `AdmittedKeySet`), the connection is closed immediately.
2. A WARN log is emitted containing `"E-ADM-003"` and the SVTN ID.
3. `Router.BindInterface` is NOT called. No binding is recorded.

**Test names:**
- `TestNodeIdentifyHandshake_ErrNotAdmitted_ConnectionClosed` (use a fresh AdmittedKeySet with no
  keys registered; node sends valid NodeIdentify; verify connection is closed, no binding exists)

---

### AC-005 — ErrKeyRevoked → connection closed E-ADM-005 (BC-2.01.009 error code E-ADM-005)

**BC Anchor:** BC-2.01.009 Error Code E-ADM-005 (key revoked → close immediately).

**Postconditions:**
1. If `admission.AdmitNode` returns `ErrKeyRevoked` (node's key has been revoked in the keyset),
   the connection is closed immediately.
2. A WARN log is emitted containing `"E-ADM-005"` and the SVTN ID.
3. `Router.BindInterface` is NOT called.

**Test names:**
- `TestNodeIdentifyHandshake_ErrKeyRevoked_ConnectionClosed` (register key, then revoke it;
  node attempts handshake; verify connection closed)

---

### AC-006 — ErrKeyExpired → connection closed E-ADM-015 (BC-2.01.009 error code E-ADM-015; BC-2.05.001 PC-6)

**BC Anchor:** BC-2.01.009 Error Code E-ADM-015 (key expired → close immediately); BC-2.05.001 Postcondition 6 (AdmitNode returns ErrKeyExpired when expiry set and past). Traces to BC-2.05.001 PC-6 and BC-2.01.009 EC-005.

**Postconditions:**
1. If `admission.AdmitNode` returns `ErrKeyExpired` (the key has a non-zero expiry timestamp and
   `time.Now().UTC()` is after that timestamp — enforced by the O-1 expiry check added to
   `AdmitNode`), the connection is closed immediately.
2. A WARN log is emitted containing `"E-ADM-015"` and the SVTN ID.
3. `Router.BindInterface` is NOT called.
4. This path is reachable because `AdmitNode` now enforces expiry (O-1 ruling §15 of rulings doc;
   BC-2.05.001 Postcondition 6, human-ratified 2026-07-18).

**Test names:**
- `TestNodeIdentifyHandshake_ErrKeyExpired_ConnectionClosed` (register key with a past-expiry
  timestamp; node attempts handshake; verify connection closed — REQUIRES Task 16 to be
  implemented first, as AdmitNode must check expiry)

---

### AC-007 — ErrNonceReplay → connection closed E-ADM-008 (BC-2.01.009 error code E-ADM-008)

**BC Anchor:** BC-2.01.009 Error Code E-ADM-008 (nonce replay → close immediately).

**Postconditions:**
1. If `admission.AdmitNode` returns `ErrNonceReplay` (the challenge nonce was already consumed
   within `admission.nonceTTL = 60s`), the connection is closed immediately.
2. A WARN log is emitted containing `"E-ADM-008"` and the SVTN ID.
3. `Router.BindInterface` is NOT called.

**Test names:**
- `TestNodeIdentifyHandshake_ErrNonceReplay_ConnectionClosed` (construct a stale
  ChallengeResponse with a nonce already consumed; verify connection closed)

---

### AC-008 — ErrSignatureVerificationFailed → connection closed E-ADM-001 (BC-2.01.009 error code E-ADM-001)

**BC Anchor:** BC-2.01.009 Error Code E-ADM-001 (signature verification failed → close immediately).

**Postconditions:**
1. If `admission.AdmitNode` returns `ErrSignatureVerificationFailed` (the `NonceSig` in the
   `ChallengeResponse` does not verify against the node's registered public key), the connection
   is closed immediately.
2. A WARN log is emitted containing `"E-ADM-001"` and the SVTN ID.
3. `Router.BindInterface` is NOT called.

**Test names:**
- `TestNodeIdentifyHandshake_ErrSignatureVerificationFailed_ConnectionClosed` (register key K1;
  node signs with different key K2; verify connection closed)

---

### AC-009 — Handshake timeout (10s) → connection closed E-ADM-022 (BC-2.01.009 Precondition 4; error code E-ADM-022)

**BC Anchor:** BC-2.01.009 Precondition 4 (`nodeIdentifyHandshakeTimeout = 10 * time.Second` set
via `conn.SetDeadline` before first read); Error Code E-ADM-022 (timeout → close immediately).

**Postconditions:**
1. `conn.SetDeadline(time.Now().Add(nodeIdentifyHandshakeTimeout))` is called at the start of the
   handshake, before the first `io.ReadFull`. The deadline is 10 seconds.
2. If the deadline fires before the three-message exchange completes (simulated by a slow client
   or mock deadline), `io.ReadFull` returns a deadline-exceeded error, the connection is closed.
3. A WARN log is emitted containing `"E-ADM-022"` and `"handshake timeout"`.
4. On successful handshake completion, `conn.SetDeadline(time.Time{})` clears the deadline (BC-2.01.009 PC-7).

**Test names:**
- `TestNodeIdentifyHandshake_Timeout_E_ADM_022` (use a `net.Pipe` with a mock deadline that fires
  immediately; verify connection is closed with timeout warning)

---

### AC-010 — LWW rebind: reconnecting node overwrites prior binding; stale cleanup guard protects new binding (BC-2.01.010 PC-2; BC-2.01.010 PC-9)

**BC Anchor:** BC-2.01.010 Postcondition 2 (LWW overwrite on reconnect); BC-2.01.010 Postcondition 9 (stale cleanup guard). Traces to BC-2.01.010 Invariant 1 (r.mu governs all three methods).

**Postconditions:**
1. When a node completes a successful handshake on a second TCP connection (same admitted identity,
   new `h.IfaceID`), `Router.BindInterface(svtnID, nodeAddr, newIfaceID)` is called. `LookupInterface`
   now returns `(newIfaceID, true)`.
2. The prior TCP connection is NOT actively torn down by the router. The prior `sendMap` entry
   remains live until the prior connection's cleanup func fires.
3. When the prior connection's cleanup func eventually fires and calls `UnbindInterface(svtnID,
   nodeAddr)` with the old `ifaceID`, the stale cleanup guard fires: `identityIfaceMap[svtnID][nodeAddr]`
   maps to the NEW `ifaceID`, not the old one. The delete is suppressed. `LookupInterface` still
   returns `(newIfaceID, true)`.
4. A clean disconnect (first connection closes BEFORE second reconnects) is also tested: after
   `UnbindInterface` with the current (matching) `ifaceID`, `LookupInterface` returns `(0, false)`;
   a subsequent `BindInterface` for the reconnect re-inserts correctly.

**Test names:**
- `TestBindInterface_LWW_Reconnect_OverwritesPriorBinding` (unit: bind ifaceID=1, then bind
  ifaceID=2 for same (svtnID,nodeAddr); LookupInterface returns (2, true))
- `TestBindInterface_StaleCleanupGuard_DoesNotRemoveNewBinding` (unit: bind 1, bind 2 LWW,
  call UnbindInterface with old ifaceID=1; LookupInterface still returns (2, true))
- `TestBindInterface_CleanDisconnect_ThenReconnect` (unit: bind 1, UnbindInterface with ifaceID=1,
  LookupInterface returns (0,false); bind 2; LookupInterface returns (2,true))

---

### AC-011 — Second NodeIdentify on same already-admitted connection → hard error E-ADM-023 → connection closed (BC-2.01.009 Invariant 7; error code E-ADM-023)

**BC Anchor:** BC-2.01.009 Invariant 7 (second NodeIdentify on same connection is hard error);
Error Code E-ADM-023 (duplicate NodeIdentify → close immediately).

**Postconditions:**
1. After the handshake completes on a connection and `ServeConn` is running, if the node sends
   another frame with `control_type=0x04` (`NODE_IDENTIFY`), the router's `route()` function
   processes it as a protocol violation.
2. The connection is closed immediately. A WARN log is emitted: `"node_identify: duplicate NodeIdentify on established connection"` (or equivalent containing E-ADM-023).
3. A second `NodeIdentify` arriving BEFORE `ServeConn` starts (within the `onAccept` handler
   itself, which is not possible with the synchronous three-message exchange) is also handled fail-closed by the local handshake-completed guard, but the primary test is the post-ServeConn path.

**Test names:**
- `TestNodeIdentifyHandshake_DuplicateNodeIdentify_E_ADM_023` (integration: complete a successful
  handshake on a connection; then send a second NodeIdentify frame via the normal frame-sending
  path; verify connection is closed with E-ADM-023 warning)

---

### AC-012 — Cleanup func calls UnbindInterface on connection close; binding removed (BC-2.01.010 PC-8)

**BC Anchor:** BC-2.01.010 Postcondition 8 (binding removed on connection close via cleanup func).

**Postconditions:**
1. The `func()` returned by `onAccept` to `netingress.Serve` (the per-connection cleanup func)
   calls `r.UnbindInterface(svtnID, nodeAddr)` in addition to `sendMap.Delete(h.IfaceID)`.
2. After the cleanup func fires (simulated by calling it directly in a test), `LookupInterface(svtnID,
   nodeAddr)` returns `(0, false)`.
3. The `ifaceID` passed to `UnbindInterface` is the connection's own `h.IfaceID`. When the stale
   cleanup guard is in effect (LWW overwrite already occurred), the delete is suppressed (AC-010).

**Test names:**
- `TestNodeIdentifyHandshake_CleanupFunc_UnbindInterface_Called` (integration: complete
  handshake; close the connection; verify cleanup func fires and LookupInterface returns (0,false))
- `TestUnbindInterface_RemovesBinding` (unit: bind (svtnID,nodeAddr,ifaceID); call UnbindInterface;
  LookupInterface returns (0,false))

---

### AC-013 — AdmitNode expiry check in internal/admission: ErrKeyExpired for past-expiry key (BC-2.05.001 PC-6; BC-2.05.001 Invariant 5)

**BC Anchor:** BC-2.05.001 Postcondition 6 (AdmitNode returns ErrKeyExpired when expiry set and
`time.Now().UTC()` is after expiry); BC-2.05.001 Invariant 5 (symmetric expiry enforcement across
AdmitNode and ReAuthenticate). Traces to BC-2.05.001 PC-6 / Invariant 5 / O-1 ruling §15.

**Postconditions:**
1. When `admission.AdmitNode` is called with a key whose `expiry` is non-zero and whose expiry
   timestamp is in the past (`time.Now().UTC().After(key.expiry)`), `AdmitNode` returns `ErrKeyExpired`.
2. The check occurs after the `snap.revoked` read (Step 1) and before the write-lock acquire
   (Step 3), mirroring `ReAuthenticate`'s expiry-check placement.
3. When called with a key whose `expiry` is in the future (not yet past), `AdmitNode` does NOT
   return `ErrKeyExpired` for the expiry reason (normal admission proceeds through the remaining checks).
4. When called with a key whose `expiry` is zero (no expiry set), `AdmitNode` does NOT return
   `ErrKeyExpired`.
5. Existing `AdmitNode` tests (revoke, nonce-replay, sig-verify, happy-path) are unaffected —
   those tests register keys without setting expiry, so the new check has no impact on them.

**Test names:**
- `TestAdmitNode_ExpiredKey_ReturnsErrKeyExpired` (unit in `internal/admission/*_test.go`:
  register a key with `SetKeyExpiry` using a past timestamp; call `AdmitNode` with valid challenge
  and response; assert `ErrKeyExpired` is returned — analog of `TestReAuthenticate_ExpiredKey_*`
  tests in `reauth_test.go`)
- `TestAdmitNode_FutureExpiry_Succeeds` (unit: register key with a future expiry; `AdmitNode`
  with valid challenge/response returns nil — expiry check does not false-positive)
- `TestAdmitNode_NoExpiry_Succeeds` (unit: register key with zero expiry; `AdmitNode` returns
  nil — guard does not fire when expiry is unset)

---

## Architecture Mapping

| Component | Module | Pure/Effectful | Justification |
|-----------|--------|---------------|---------------|
| NODE_IDENTIFY frame codec (encode/decode functions) | `cmd/switchboard/node_identify_wire.go` | pure-core | No I/O; encodes/decodes byte slices; deterministic |
| `nodeIdentifyHandshake` driver | `cmd/switchboard/node_identify_wire.go` | effectful-shell | TCP I/O via `net.Conn`; reads/writes live connection; calls `conn.SetDeadline` |
| `onAccept` wiring (calling handshake + sendMap) | `cmd/switchboard/mgmt_wire.go` | effectful-shell | Existing effectful shell; this story adds handshake call before `sendMap.Store` |
| `Router.BindInterface` / `LookupInterface` / `UnbindInterface` | `internal/routing/identity.go` | pure-core | No I/O; mutex-protected map operations only; deterministic given lock discipline |
| `Router.identityIfaceMap` field | `internal/routing/routing.go` | pure-core | New map field; initialized with `Router`; no I/O |
| `admission.AdmitNode` expiry check (4 lines) | `internal/admission/admission.go` | pure-core | Existing pure-core function; expiry check is a pure time comparison; no I/O |

## Non-Goals

- **`S-BL.DISCOVERY-WIRE` fan-out dispatch (AC-017/AC-018/Task 6)** — those AC items gate on
  this story's `LookupInterface` being available; they are NOT delivered here.
- **Key rotation UX** — out of scope; this story wires the existing static-admitted-key handshake.
- **Mid-connection re-admission** — not needed; reconnect uses a full new TCP handshake (LWW).
- **RouterSig binding to `(svtnID, nodeAddr)`** — pre-existing property of `internal/admission`;
  this story introduces no changes to `GenerateChallenge`. See rulings §11 for the security analysis.
- **New config fields or management RPCs** — the handshake uses existing config (`routerPrivKey`,
  `AdmittedKeySet`); no new config fields are needed.

## Edge Cases

| ID | Scenario | Expected Behavior |
|----|----------|-------------------|
| EC-001 | Eventual-consistency race: node connects before its `RegisterKey` push arrives at the router | `AdmitNode` returns `ErrNotAdmitted` (E-ADM-003); connection closed. Node retries after backoff. Self-resolving when push arrives. Not a protocol defect (BC-2.01.009 EC-001 / Invariant 8). |
| EC-002 | Handshake timeout fires during `io.ReadFull` | Deadline-exceeded error; connection closed; WARN E-ADM-022 logged (BC-2.01.009 EC-002). |
| EC-003 | Non-zero reserved byte at `payload[3]` | Hard decoder error; connection closed; WARN logged (BC-2.01.009 EC-003 / Invariant 5). |
| EC-004 | Second NodeIdentify on established (already-admitted) connection | Hard error E-ADM-023; connection closed immediately (BC-2.01.009 EC-004 / Invariant 7). Detected via `case 0x04` in `route()`. |
| EC-005 | Expired key at connect time | `AdmitNode` returns `ErrKeyExpired` (E-ADM-015); connection closed. BC-2.05.001 Postcondition 6. AC-006 covers this. (BC-2.01.009 EC-005). |
| EC-006 | Node's key not registered for the SVTN in the outer header SVTNID | `ErrNotAdmitted` (E-ADM-003); connection closed (BC-2.01.009 EC-006). |
| EC-007 | Two successive reconnects (new TCP each time); key admitted | Each reconnect runs full three-message handshake. `BindInterface` uses LWW. Rebind requires `AdmitNode` nil return — full re-handshake required (BC-2.01.009 EC-007). |
| EC-008 | Prior connection's cleanup func fires AFTER LWW overwrite | Stale cleanup guard fires (BC-2.01.010 EC-001): stored ifaceID != caller's own ifaceID; UnbindInterface delete suppressed; new binding preserved. AC-010 covers this. |
| EC-009 | `LookupInterface` for unbound `(svtnID, nodeAddr)` | Returns `(0, false)`. Caller MUST check bool flag (BC-2.01.010 EC-003). AC-012 covers this. |
| EC-010 | `identityIfaceMap` nested map for svtnID absent when `BindInterface` called | `BindInterface` allocates the nested `map[[8]byte]InterfaceID` if absent (standard Go nil-map init pattern). |

## Purity Classification

| Module | Classification | Justification |
|--------|---------------|---------------|
| `internal/admission` (AdmitNode, GenerateChallenge) | pure-core | Existing classification; expiry check (O-1) adds pure time comparison; no I/O |
| `internal/routing` (BindInterface, LookupInterface, UnbindInterface) | pure-core | Mutex-protected map operations; no I/O; deterministic |
| `cmd/switchboard` (handshake driver, onAccept wiring) | effectful-shell | TCP I/O, live connection reads/writes, conn.SetDeadline |

## Token Budget Estimate (MANDATORY)

| Context Source | Estimated Tokens |
|---------------|-----------------|
| This story spec | ~700 |
| `cmd/switchboard/mgmt_wire.go` (onAccept closure, route(), sendMap) | ~300 |
| `internal/routing/routing.go` (Router struct, r.mu pattern) | ~150 |
| `internal/admission/admission.go` (AdmitNode, ReAuthenticate expiry pattern) | ~200 |
| New `cmd/switchboard/node_identify_wire.go` (codec + handshake driver, ~200 lines) | ~200 |
| New `internal/routing/identity.go` (3 methods, ~80 lines) | ~80 |
| BC files (5 BCs) | ~300 |
| Rulings doc (relevant sections) | ~200 |
| Test files (13 ACs, ~25 test functions) | ~600 |
| Tool outputs overhead | ~150 |
| **Total** | **~2,880 tokens — well under 20% of agent context window** |

## Tasks (MANDATORY)

Red Gate discipline: all test functions must be written FIRST (test-writer step) and FAIL
before any implementation code is written (implementer step). Do NOT write any production
code until `go test ./...` shows compile errors or test failures for ALL ACs.

1. [ ] Write failing tests for AC-001 (handshake success: BindInterface called, ServeConn begins) — test-writer
2. [ ] Write failing tests for AC-002 (malformed NodeIdentify: wrong payload_len, wrong msg_kind, non-zero reserved byte) — test-writer
3. [ ] Write failing tests for AC-003 (zero SVTN ID rejected) — test-writer
4. [ ] Write failing tests for AC-004 (ErrNotAdmitted closes connection) — test-writer
5. [ ] Write failing tests for AC-005 (ErrKeyRevoked closes connection) — test-writer
6. [ ] Write failing tests for AC-006 (ErrKeyExpired closes connection — depends on Task 16) — test-writer
7. [ ] Write failing tests for AC-007 (ErrNonceReplay closes connection) — test-writer
8. [ ] Write failing tests for AC-008 (ErrSignatureVerificationFailed closes connection) — test-writer
9. [ ] Write failing tests for AC-009 (handshake timeout 10s → E-ADM-022) — test-writer
10. [ ] Write failing tests for AC-010 (LWW rebind: OverwritesPriorBinding + StaleCleanupGuard + CleanDisconnect) — test-writer
11. [ ] Write failing tests for AC-011 (second NodeIdentify post-handshake → E-ADM-023) — test-writer
12. [ ] Write failing tests for AC-012 (cleanup func calls UnbindInterface; binding removed) — test-writer
13. [ ] Write failing tests for AC-013 (AdmitNode expiry check: ErrKeyExpired for past-expiry key; FutureExpiry_Succeeds; NoExpiry_Succeeds) — test-writer [in `internal/admission/*_test.go`]
14. [ ] Verify Red Gate: `go test ./...` fails with compile or test failures for all 13 ACs
15. [ ] Add `identityIfaceMap map[[16]byte]map[[8]byte]InterfaceID` field to `Router` struct in `internal/routing/routing.go` (initialized in `NewRouter`); implement `BindInterface`, `LookupInterface`, `UnbindInterface` in `internal/routing/identity.go` with `r.mu` write/read lock discipline; stale cleanup guard in `UnbindInterface` (check `identityIfaceMap[svtnID][nodeAddr] == callerIfaceID` before deleting) — implementer [BC-2.01.010 PC-1/PC-4/PC-8/PC-9; §8 ruling]
16. [ ] Add expiry check to `admission.AdmitNode` in `internal/admission/admission.go`: after `snap.revoked` check, before write-lock acquire — `if !snap.expiry.IsZero() && time.Now().UTC().After(snap.expiry) { return ErrKeyExpired }` — mirrors `ReAuthenticate` expiry-check pattern at `reauth.go:196`; re-check under write lock following the same two-step pattern — implementer [O-1 ruling §15; BC-2.05.001 PC-6 / Invariant 5]
17. [ ] Implement `const nodeIdentifyHandshakeTimeout = 10 * time.Second` and NODE_IDENTIFY frame codec in `cmd/switchboard/node_identify_wire.go`: `encodeNodeIdentify`, `encodeChallenge`, `encodeChallengeResponse` (pure encode functions); `decodeNodeIdentify`, `decodeChallengeResponse` (pure decode functions with all per-message size guards and reserved-byte check); all fixed sizes enforced (36/100/68 bytes) — implementer [BC-2.01.009 Postconditions 1/3/4; Invariant 5; §§4–6 of rulings]
18. [ ] Implement `nodeIdentifyHandshake(conn net.Conn, r *routing.Router, routerPrivKey ed25519.PrivateKey, ks *admission.AdmittedKeySet, h netingress.ConnHandle) (svtnID [16]byte, nodeAddr [8]byte, err error)` in `cmd/switchboard/node_identify_wire.go`: `conn.SetDeadline(time.Now().Add(nodeIdentifyHandshakeTimeout))` on entry; `io.ReadFull` outer header + payload for NodeIdentify; validate SVTNID non-zero; decode NodeIdentify; derive `nodeAddr = frame.DeriveNodeAddress(hdr.SVTNID, pubkey)`; call `GenerateChallenge(routerPrivKey)`; encode + write Challenge; `io.ReadFull` outer header + payload for ChallengeResponse; decode ChallengeResponse; call `AdmitNode`; on success call `r.BindInterface(hdr.SVTNID, nodeAddr, h.IfaceID)` then `conn.SetDeadline(time.Time{})`; on any error path close conn and return error — implementer [BC-2.01.009 PC-1 through PC-8; §7 of rulings; failure table §13]
19. [ ] Add `case 0x04:` to `route()` switch in `cmd/switchboard/mgmt_wire.go`: log WARN `"node_identify: duplicate NodeIdentify on established connection"` (E-ADM-023) and return non-nil error to cause connection teardown via `netingress.ServeConn` error path — implementer [BC-2.01.009 Invariant 7; §12 of rulings; AC-011]
20. [ ] Wire `nodeIdentifyHandshake` into `onAccept` in `runRouter` (`cmd/switchboard/mgmt_wire.go`): call `nodeIdentifyHandshake(conn, r, routerPrivKey, ks, h)` at the start of the per-conn goroutine BEFORE `sendMap.Store`; on failure return a no-op cleanup func (connection already closed by handshake driver); on success proceed with `sendMap.Store(h.IfaceID, ...)` and existing drain setup; extend cleanup func to also call `r.UnbindInterface(svtnID, nodeAddr)` — the stale cleanup guard in `UnbindInterface` safely suppresses the delete if a LWW overwrite occurred — implementer [BC-2.01.009 PC-6; BC-2.01.010 PC-8; §7 and §12 of rulings]
21. [ ] Run `go test ./... -race`; confirm all 13 AC test functions pass
22. [ ] Update STATE.md (state-manager)

## Architecture Compliance Rules (MANDATORY)

| Rule | Requirement | Enforcement |
|------|-------------|-------------|
| ARCH-08 §Import DAG — all new cmd/ code at position 18 | `cmd/switchboard` is position 18 (the top). New files `node_identify_wire.go` and any `onAccept` edits in `mgmt_wire.go` stay in `cmd/switchboard`. New `internal/routing/identity.go` adds no new imports to `internal/routing` (already imports `internal/admission`). | `go list -deps` verification recommended in test suite |
| ARCH-08 position 5: `internal/routing` already imports `internal/admission` | `identityIfaceMap` and the three new methods require zero new imports for `internal/routing`. Confirmed via verified premises in rulings §8. | Compile-time |
| F-P2L1-001 register-before-serve | `nodeIdentifyHandshake` is called in `onAccept` BEFORE `sendMap.Store` — unverified nodes never appear in the routing map. Analogous to `wireAdmissionSyncHandlers` being called after `newMgmtServer` / before `serveMgmtServer`. | AC-001 happy-path test verifies no binding exists pre-handshake |
| ADR-003 LWW semantics | `BindInterface` overwrites prior bindings (LWW) — consistent with `forwardingTable` mutation semantics and keyset mutation semantics in `AdmittedKeySet`. | AC-010 tests LWW overwrite path |
| go.md rule 12 — return value copies from locked accessors | `LookupInterface` returns `(InterfaceID, bool)` — a value type, not a pointer into internal state. | Code review; `go test -race` |
| go.md rule 12 — write lock for mutations | `BindInterface` and `UnbindInterface` acquire `r.mu` write lock; `LookupInterface` acquires read lock. Identical to `RegisterForwardingEntry` / `LookupForwardingEntry` discipline. | `go test -race` |
| BC-2.01.009 Invariant 8 — eventual-consistency race is NOT a protocol defect | `ErrNotAdmitted` on the race path returns the same error as any unregistered key — no special handling. | AC-004 covers the non-race `ErrNotAdmitted` path; EC-001 documents the race |
| DI-002 — private keys never transit | The `ChallengeResponse.NonceSig` is the node's signature, not its private key. The router never sees the node's private key. The `routerPrivKey` used in `GenerateChallenge` stays within the router process. | Code review; BC-2.05.001 Invariant 1 |
| Forbidden dependency: cmd/switchboard MUST NOT gain `internal/netingress` import | `netingress` is position 8, downstream of `internal/routing` (5) but upstream of `cmd/switchboard` (18). The `onAccept` closure is wired in `runRouter` which already has access to `net.Conn` directly. No new `netingress` import needed. | `go list -deps ./cmd/switchboard` |

## Library & Framework Requirements (MANDATORY)

| Tool / Package | Version | Purpose |
|----------------|---------|---------|
| Go | 1.25.4 (per `go.mod`) | Language runtime — all new files are Go |
| `crypto/ed25519` | stdlib | `PublicKeySize = 32`, `SignatureSize = 64`; used in codec size constants and signature verify |
| `io` | stdlib | `io.ReadFull` for fixed-size frame reads in handshake driver |
| `net` | stdlib | `net.Conn` interface for handshake driver; `conn.SetDeadline` |
| `time` | stdlib | `nodeIdentifyHandshakeTimeout = 10 * time.Second`; expiry check in `AdmitNode` |
| `sync` | stdlib | `sync.RWMutex` via existing `r.mu` in `internal/routing.Router` |
| `internal/admission` | project-local | `AdmitNode`, `GenerateChallenge`, `Challenge`, `ChallengeResponse`, `AdmittedKeySet`, `ErrKeyExpired`, `ErrKeyRevoked`, `ErrNotAdmitted`, `ErrNonceReplay`, `ErrSignatureVerificationFailed` |
| `internal/frame` | project-local | `DeriveNodeAddress`, `FrameTypeCtl = 0x03`, `VersionByte = 0x01`, `OuterHeader`, `OuterHeaderSize = 44` |
| `internal/routing` | project-local | `InterfaceID`, `Router` (new methods added here) |

## File Structure Requirements (MANDATORY)

| File | Action | Purpose |
|------|--------|---------|
| `cmd/switchboard/node_identify_wire.go` | **create** | `const nodeIdentifyHandshakeTimeout = 10 * time.Second`; NODE_IDENTIFY frame codec (`encodeNodeIdentify`, `encodeChallenge`, `encodeChallengeResponse`, `decodeNodeIdentify`, `decodeChallengeResponse`); `nodeIdentifyHandshake` driver (full three-message exchange with SetDeadline and all fail-closed error paths) |
| `cmd/switchboard/node_identify_wire_test.go` | **create** | Handshake integration tests (AC-001 through AC-009, AC-011, AC-012) — uses `net.Pipe()` or loopback TCP to simulate node connections; in-process `Router` + populated `AdmittedKeySet` |
| `cmd/switchboard/mgmt_wire.go` | **modify** | (1) Add `case 0x04:` to `route()` switch with E-ADM-023 warning + error return (AC-011); (2) Wire `nodeIdentifyHandshake` call at start of `onAccept` before `sendMap.Store`; extend cleanup func to call `r.UnbindInterface` (Tasks 19+20) |
| `internal/routing/identity.go` | **create** | `BindInterface`, `LookupInterface`, `UnbindInterface` methods on `*Router`; `identityIfaceMap map[[16]byte]map[[8]byte]InterfaceID` — new field (initialized in `NewRouter` or first use); full `r.mu` lock discipline; stale cleanup guard in `UnbindInterface` |
| `internal/routing/routing.go` | **modify** | Add `identityIfaceMap map[[16]byte]map[[8]byte]InterfaceID` field to `Router` struct; initialize in `NewRouter` (or add `sync.Once` init in `BindInterface` — prefer explicit init in `NewRouter` for clarity) |
| `internal/routing/identity_test.go` | **create** | Unit tests for `BindInterface`, `LookupInterface`, `UnbindInterface`: happy-path (AC-012), LWW overwrite (AC-010), stale cleanup guard (AC-010), `LookupInterface` for unbound key returns `(0, false)` |
| `internal/admission/admission.go` | **modify** | Add expiry check to `AdmitNode`: `if !snap.expiry.IsZero() && time.Now().UTC().After(snap.expiry) { return ErrKeyExpired }` after `snap.revoked` read, before write-lock acquire; add re-check under write lock per `ReAuthenticate` pattern (Task 16) |
| `internal/admission/admission_test.go` (or `admitnode_test.go`) | **modify** | Add `TestAdmitNode_ExpiredKey_ReturnsErrKeyExpired`, `TestAdmitNode_FutureExpiry_Succeeds`, `TestAdmitNode_NoExpiry_Succeeds` (AC-013) |

> **CORRECTION from stub v1.4:** The File-Change List previously listed
> `specs/behavioral-contracts/ss-01/BC-2.01.008.md` as "modify — Add NODE_IDENTIFY=0x04 row."
> This edit is **already done** (BC-2.01.008 v1.3, 2026-07-15, Obligation 1 RESOLVED). No further
> edit to BC-2.01.008 is required by this story.

## Provenance

- **Origin:** `S-BL.DISCOVERY-WIRE.md` Forward Obligations table, row (a) — Ruling 3(f)
  verified that hop-2 fan-out target resolution does not exist in production code.
- **Disposition:** story-ready human gate for `S-BL.DISCOVERY-WIRE`, 2026-07-14 —
  human selected `S-BL.DISCOVERY-WIRE-fanout-options.md` v1.0 Option 1.
- **Wire-format ruling:** `decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md` v1.0 — Obligation 2.
- **Obligations 3/4:** `decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md` v1.1 §§12-13.
- **Obligations 5/6:** RESOLVED-BY-DELIVERY — PR #126 (`S-BL.ADMISSION-SYNC-WIRE`) + PR #125 (`S-BL.NODE-ADMISSION-PROVISIONING`).
- **O-1:** `decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md` v1.1 §15; human-ratified 2026-07-18; BC-2.05.001 v1.2 amended.
- **Cluster design:** `decisions/identity-cluster-architecture.md` v1.1.
- **Unblocks:** `S-BL.DISCOVERY-WIRE`'s AC-017, AC-018, and Task 6.

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.6 | 2026-07-18 | Consistency-audit Finding 2 cascade: BC-2.01.010 PC renumber (duplicate PC-4 fixed; LookupInterface 4→5/5→6/6→7, UnbindInterface 7→8/8→9/9→10). Citations updated: AC-010 heading/BC Anchor PC-8→PC-9 (stale cleanup guard); AC-012 heading/BC Anchor PC-7→PC-8 (binding removed on close); Prev-Story-Intel S-7.04 row PC-7→PC-8 / PC-8→PC-9; Task 15 PC-7/PC-8→PC-8/PC-9; Task 20 PC-7→PC-8. input-hash recomputed. |
| 1.5 | 2026-07-18 | Full decomposition: 13 ACs covering all §15 AC areas + wire-format error paths; frontmatter updated (`behavioral_contracts`, `bc_traces`, `inputs`, `points=10`, `status=ready`); all [TODO] sections populated; O-1 AdmitNode expiry check added to scope (supersedes "zero changes to internal/admission" claim); BC-2.01.009, BC-2.01.010, BC-2.05.001 added to bc_traces; File-Change List corrected (BC-2.01.008 edit is DONE — removed from list); Previous Story Intelligence populated with #125/#126 lessons; Architecture Compliance Rules expanded; `inputDocuments:` added; stale claim superseded. input-hash updated to reflect new inputs list. |
| 1.4 | 2026-07-15 | `depends_on` updated from `[]` to `[S-BL.ADMISSION-SYNC-WIRE, S-BL.NODE-ADMISSION-PROVISIONING]` — both prerequisite stories now exist. Template conformance scaffolding added. |
| 1.3 | 2026-07-15 | Obligations 1+2 RESOLVED — wire format in `decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md` v1.0. `rulings_doc` frontmatter added. |
| 1.2 | 2026-07-15 | Added Open Design Obligation 6 — second BLOCKER: no production path provisions node admission keypair. |
| 1.1 | 2026-07-15 | Added Open Design Obligation 5 — BLOCKER: router-mode `AdmittedKeySet` always empty. |
| 1.0 | 2026-07-14 | Backlog stub created per `S-BL.DISCOVERY-WIRE` Ruling 3(f) Forward Obligation. |
