# Demo Evidence Report — S-BL.DISCOVERY-WIRE

**Story:** Discovery wire boundary: UDP multicast I/O, admitted-node HMAC keys,
multicast address allocation, hop-2 relay dispatch
**Story version:** v2.14
**Branch:** feature/S-BL.DISCOVERY-WIRE
**Code HEAD:** 501db03a99db7a06586988e133279cb43a11e021
**Evidence date:** 2026-07-15

---

## POL-004 Note

Per `docs/DEMO-EVIDENCE-POLICY.md` (ratified 2026-07-04), rendered binaries
(`.gif`, `.webm`, `.mp4`, `.png`) are **not committed**. `.tape` scripts and
this evidence report are the source of truth. To regenerate locally:

```bash
vhs docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-001-router-mode-multicast-membership.tape
vhs docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-002-003-multicast-addr-and-sender-dispatch.tape
vhs docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-004-discovery-key-derivation.tape
vhs docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-005-006-key-selector-hmac-fail-closed.tape
vhs docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-007-node-local-relay-ingest.tape
vhs docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-008-009-010-replay-sequence-gate.tape
vhs docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-011-012-013-bounded-buffer-rate-cap-logging.tape
vhs docs/demo-evidence/S-BL.DISCOVERY-WIRE/AC-014-015-016-relay-frame-assembly.tape
```

## Scope note — internal wire/plumbing code, not a CLI or web surface

This story implements the discovery wire boundary: UDP multicast I/O,
router-side ingest authentication, node-local relay-ingest, and hop-2
`DISCOVERY_RELAY` frame assembly. None of `RouterIngest.Ingest`,
`Encode`/`Decode`, `MulticastAddrFor`, `IngestRelayAdvertisement`,
`assembleDiscoveryRelayFrame`, or `wireDiscoveryListener` has an interactive
CLI command or a production caller wiring it into `runRouter` end-to-end yet
(see the "Coverage gap — AC-017/AC-018, Task 6" section below). The honest,
meaningful evidence for wire code at this stage is the passing test suite
that verifies each AC's stated postconditions directly — that is what each
tape below drives, via `go test -race -run <pattern> -v`, rather than an
interactive CLI/TUI walkthrough.

All test commands below are run from the worktree root. Every test passes
under `go test -race`.

---

## Summary Table — AC → Test(s) → Result → Postcondition(s) demonstrated

| AC | Tape | Test(s) | Result | Postcondition(s) demonstrated |
|----|------|---------|--------|-------------------------------|
| AC-001 | `AC-001-router-mode-multicast-membership.tape` | `TestRunRouter_DiscoveryListener_JoinsGroup_RouterModeOnly` | PASS | Only router-mode joins the SVTN multicast group; access/console/control never join or receive directly. Accepted at function level per Ruling 4 scope note — see gap note below. |
| AC-002 | `AC-002-003-multicast-addr-and-sender-dispatch.tape` | `TestMulticastAddrFor_Deterministic_SHA256Derived` | PASS | `MulticastAddrFor` returns `239.h0.h1.h2` (first 3 bytes of SHA-256(svtnID)), deterministic and static, no coordination step. |
| AC-003 | `AC-002-003-multicast-addr-and-sender-dispatch.tape` | `TestDiscovery_Advertise_WriteToMulticast_TTL1_NoGroupJoin` | PASS | Sender-side dispatch via `net.ListenUDP`+`WriteToUDP` per UP+multicast-capable interface, TTL=1, no `net.ListenMulticastUDP`/group join. |
| AC-004 | `AC-004-discovery-key-derivation.tape` | `TestDeriveDiscoveryKey_DomainSeparatedFromFrameAuthKey`, `TestDiscoveryAuthKeyFor_LookupSuccessAndMiss`, `TestDeriveDiscoveryKey_SenderRouterAgree`, `TestDiscovery_EncodeThenRouterIngest_AcceptsRealAdmittedNode`, `TestDiscovery_Advertise_MissingNodeAdmissionPubkey_FailsClosed` | PASS | `DeriveDiscoveryKey` domain-separated from `DeriveKey` (distinct HKDF info label); `DiscoveryAuthKeyFor` lookup success/miss; sender/router key agreement; real `Encode`→router-`Ingest` round-trip for a genuinely admitted node (F-DWIP1-001 regression fix); fail-closed `ErrMissingNodeAdmissionPubkey` guard. |
| AC-005 | `AC-005-006-key-selector-hmac-fail-closed.tape` | `TestRouterIngest_KeySelectorExtraction_FixedOffset_NoFullDecodeBeforeAuth`, `TestRouterIngest_HMACCoversFullBody_TamperInSessionListDetected`, `TestRouterIngest_ShortDatagram_RejectedBeforeLookup`, `TestRouterIngest_FullValidFrameMinimum_42Bytes` | PASS | Fixed-offset `SVTNID`/`NodeAddr` extraction precedes `decodeBody()`; HMAC covers the complete raw body (session-list tampering detected); raw datagrams shorter than the 32-byte tag+key-selector minimum are rejected before lookup; 42-byte full-valid-frame minimum confirmed with the widened `uint64 Sequence`. |
| AC-006 | `AC-005-006-key-selector-hmac-fail-closed.tape` | `TestRouterIngest_LookupMissAndTagMismatch_IndistinguishableRejection` | PASS | Lookup-miss and HMAC-tag-mismatch resolve to the identical `ErrInvalidHMACTag` sentinel; no registry mutation, no relay, fail-closed continue. |
| AC-007 | `AC-007-node-local-relay-ingest.tape` | `TestDiscovery_IngestRelayAdvertisement_SVTNMismatch_ErrSVTNMismatch`, `TestDiscovery_IngestRelayAdvertisement_NoHMACRequired`, `TestDiscovery_IngestRelayAdvertisement_Success_RegistryReplaceOnWrite` | PASS | `ReceiveAdvertisement` retired; new node-side relay-ingest decodes hop-2 payload with no per-frame HMAC; `ErrSVTNMismatch` relocated to a direct `OuterHeader.SVTNID` equality check; registry replace-on-write on success. |
| AC-008 | `AC-008-009-010-replay-sequence-gate.tape` | `TestVP080_DiscoveryIngest_ColdStartAcceptance` | PASS | First datagram for a fresh `(SVTNID, NodeAddr)` pair (no prior `lastSeen`) is accepted for any declared `Sequence`, including 0. |
| AC-009 | `AC-008-009-010-replay-sequence-gate.tape` | `TestVP080_DiscoveryIngest_ReplayDiscard_ExactSequence`, `TestVP080_DiscoveryIngest_ReplayDiscard_LowerSequence`, `TestVP080_DiscoveryIngest_ReplayDiscard_NoRelaySideEffect` | PASS | A second HMAC-verified datagram declaring `Sequence <= N` is discarded (exact-replay and lower-sequence cases); no registry update, no relay, `lastSeen` unchanged. |
| AC-010 | `AC-008-009-010-replay-sequence-gate.tape` | `TestVP080_DiscoveryIngest_ForwardAcceptance_AdvancesState`, `TestVP080_DiscoveryIngest_RestartForwardProgress` | PASS | Strictly-increasing `Sequence` is accepted, registry updates, relay triggers, `lastSeen` advances; a restarted node's freshly-epoch-qualified `Sequence` is accepted via forward acceptance (F-DWSP4-001), not cold-start. |
| AC-011 | `AC-011-012-013-bounded-buffer-rate-cap-logging.tape` | `TestRouterIngest_OversizedDatagram_RejectedNoPartialParse` | PASS | Datagram exceeding the sized read buffer is rejected without partial-parse and without reallocation-to-fit. |
| AC-012 | `AC-011-012-013-bounded-buffer-rate-cap-logging.tape` | `TestRouterIngest_AggregateRateCap_NotPerSource`, `TestRouterIngest_FailureCounter_VisibilityOnly_NeverGates` | PASS | Aggregate (not per-source) token-bucket cap rejects once exceeded regardless of declared `NodeAddr`; `FailureCounter` fires on HMAC-rejection for visibility only, never gates admission on attacker-controlled `NodeAddr`. |
| AC-013 | `AC-011-012-013-bounded-buffer-rate-cap-logging.tape` | `TestRouterIngest_FailureLogging_ThresholdCrossingOnly_NotPerPacket` | PASS | HMAC-rejection logging fires only on `FailureCounter`'s threshold-crossing emission, not per rejected packet — distinct from BC-2.05.008's per-packet policy. |
| AC-014 | `AC-014-015-016-relay-frame-assembly.tape` | `TestAssembleDiscoveryRelayFrame_PayloadLayout` | PASS | `DISCOVERY_RELAY` payload layout: `control_type=0x03`/`version=0x01`/`reserved=0x0000` at bytes 0-3; `NodeAddr`/`Sequence`/count/sessions at their specified offsets; `SVTNID` not repeated (lives in `OuterHeader.SVTNID`); pure function, no live connection required. |
| AC-015 | `AC-014-015-016-relay-frame-assembly.tape` | `TestAssembleDiscoveryRelayFrame_ZeroHMACTag` | PASS | Relay frame's `OuterHeader.HMACTag` is the zero value (matches the DRAIN precedent); no per-frame HMAC computed for hop-2 — trust derives from the admitted TCP connection. |
| AC-016 | `AC-014-015-016-relay-frame-assembly.tape` | `TestAssembleDiscoveryRelayFrame_NotRawHop1Bytes` | PASS | Relay payload is freshly constructed from decoded fields, never a byte-for-byte copy of hop-1's raw UDP datagram; hop-1's original HMAC tag never appears in the relay frame. |
| AC-017 | — | — | **GATED** | `depends_on S-BL.NODE-IDENTIFY-WIRE` (fan-out target-resolution companion story, not yet landed). Not implemented, not tested, not demoed — see gap note below. |
| AC-018 | — | — | **GATED** | Same gate as AC-017 (rate-cap decision is meaningless without a live dispatch mechanism to suppress). Not implemented, not tested, not demoed — see gap note below. |

---

## AC-001 — Router-mode-exclusive multicast group membership

**Tape:** `AC-001-router-mode-multicast-membership.tape`
**BC anchors:** BC-2.03.001 Postcondition 1 (delivery-mechanism note), Invariant 1 (DI-004)
**Test file:** `cmd/switchboard/discovery_wire_test.go`

Only the router-mode daemon calls `net.ListenMulticastUDP` and joins the
SVTN-scoped multicast group on its LAN-facing interface(s); `runAccess`,
`runConsole`, and `runControl` never join any multicast group and never
receive advertisements directly from another node's socket.

**Scope note (Ruling 4, v1.10, 2026-07-15):** `wireDiscoveryListener` is
fully implemented and independently tested at function level (below), but
is not yet called from `runRouter` — the router process has no source of
"which SVTN(s) am I serving" today (Forward Obligation (e),
`S-BL.ADMISSION-SYNC-WIRE`, not yet built). This test verifies the function
in isolation; it does not, and given the stated gap currently cannot, verify
daemon-lifecycle behavior. Accepted at function level per the story's own
disposition — not a defect in this evidence.

**Evidence command:**

```
go test -race -run TestRunRouter_DiscoveryListener_JoinsGroup_RouterModeOnly -count=1 -v ./cmd/switchboard/
```

**Captured output:**

```
=== RUN   TestRunRouter_DiscoveryListener_JoinsGroup_RouterModeOnly
--- PASS: TestRunRouter_DiscoveryListener_JoinsGroup_RouterModeOnly (0.51s)
PASS
ok  	github.com/arcavenae/switchboard/cmd/switchboard	0.851s
```

**Discharge status:** FULL at function level (per Ruling 4 scope note).

---

## AC-002 — Multicast address derivation: static, deterministic, SVTN-scoped

**Tape:** `AC-002-003-multicast-addr-and-sender-dispatch.tape`
**BC anchor:** BC-2.03.001 Precondition 3
**Test file:** `internal/discovery/discovery_test.go`

`MulticastAddrFor(svtnID [16]byte) net.IP` returns `239.h0.h1.h2`, the first
three bytes of SHA-256(svtnID) — deterministic and computable independently
by every admitted node and the router, with no coordination step, no
allocation bookkeeping, and no release step.

**Evidence command:**

```
go test -race -run TestMulticastAddrFor_Deterministic_SHA256Derived -count=1 -v ./internal/discovery/
```

**Captured output:**

```
=== RUN   TestMulticastAddrFor_Deterministic_SHA256Derived
--- PASS: TestMulticastAddrFor_Deterministic_SHA256Derived (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/discovery	0.363s
```

**Discharge status:** FULL.

---

## AC-003 — Sender-side transmission: TTL=1, no group membership required

**Tape:** `AC-002-003-multicast-addr-and-sender-dispatch.tape`
**BC anchor:** BC-2.03.001 Postcondition 1 (delivery-mechanism note); SEC-DW-08
**Test file:** `internal/discovery/discovery_test.go`

The access node's `Run()`/`Advertise` path sends to the SVTN-derived
multicast address once per UP+multicast-capable local interface
(`net.ListenUDP` + `WriteToUDP`, each pinned via `setsockopt
IP_MULTICAST_IF`) — no `net.ListenMulticastUDP`, no group join, on any
interface. The outbound socket's multicast TTL is explicitly set to 1
before the first send.

**Evidence command:**

```
go test -race -run TestDiscovery_Advertise_WriteToMulticast_TTL1_NoGroupJoin -count=1 -v ./internal/discovery/
```

**Captured output:**

```
=== RUN   TestDiscovery_Advertise_WriteToMulticast_TTL1_NoGroupJoin
--- PASS: TestDiscovery_Advertise_WriteToMulticast_TTL1_NoGroupJoin (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/discovery	0.363s
```

**Discharge status:** FULL.

---

## AC-004 — DiscoveryAuthKey derivation: domain-separated from FrameAuthKey

**Tape:** `AC-004-discovery-key-derivation.tape`
**BC anchor:** BC-2.03.001 Postcondition 5; SEC-DW-06
**Test files:** `internal/hmac/hmac_test.go`, `internal/routing/advertisement_hmac_test.go`, `internal/discovery/discovery_wire_test.go`, `internal/discovery/discovery_test.go`

`hmac.DeriveDiscoveryKey(nodeAdmissionPubkey, svtnID)` computes HKDF-SHA256
over the same inputs `DeriveKey` uses but with a distinct info label
(`HKDFInfoDiscovery`), producing a cryptographically independent key.
`(*routing.Router).DiscoveryAuthKeyFor` returns `(key, true)`/`(zero,
false)` on lookup success/miss. `routing.DeriveDiscoveryKey` (the
sender-side symmetric wrapper) agrees with the router's own computation.

**Qualifying note (F-DWIP1-001, v1.11):** a Step-4.5 pass-1 fix burst found
the shipped `Encode`/`Decode` had regressed to deriving the discovery HMAC
key from cleartext `SVTNID` alone (the exact anti-pattern
DRIFT-W6TBD-001/Ruling 1 already rejected), breaking sender↔router interop
undetected because no prior test exercised a real `Encode`→router-`Ingest`
round-trip. Both `Encode`/`Decode` now take an explicit
`nodeAdmissionPubkey []byte` parameter routed through
`routing.DeriveDiscoveryKey`; `discovery.Config` gained
`LocalNodeAdmissionPubkey []byte`; `transmitAdvertisement` fails closed
with `ErrMissingNodeAdmissionPubkey` when empty. The two new tests below
cover exactly this fix and its fail-closed guard.

**Evidence commands:**

```
go test -race -run TestDeriveDiscoveryKey_DomainSeparatedFromFrameAuthKey -count=1 -v ./internal/hmac/
go test -race -run 'TestDiscoveryAuthKeyFor_LookupSuccessAndMiss|TestDeriveDiscoveryKey_SenderRouterAgree' -count=1 -v ./internal/routing/
go test -race -run TestDiscovery_EncodeThenRouterIngest_AcceptsRealAdmittedNode -count=1 -v ./internal/discovery/
go test -race -run TestDiscovery_Advertise_MissingNodeAdmissionPubkey_FailsClosed -count=1 -v ./internal/discovery/
```

**Captured output:**

```
=== RUN   TestDeriveDiscoveryKey_DomainSeparatedFromFrameAuthKey
--- PASS: TestDeriveDiscoveryKey_DomainSeparatedFromFrameAuthKey (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/hmac	0.303s

=== RUN   TestDiscoveryAuthKeyFor_LookupSuccessAndMiss
=== RUN   TestDiscoveryAuthKeyFor_LookupSuccessAndMiss/lookup_success
=== RUN   TestDiscoveryAuthKeyFor_LookupSuccessAndMiss/lookup_miss:_unregistered_nodeAddr
=== RUN   TestDiscoveryAuthKeyFor_LookupSuccessAndMiss/lookup_miss:_unregistered_svtnID
--- PASS: TestDiscoveryAuthKeyFor_LookupSuccessAndMiss (0.00s)
    --- PASS: TestDiscoveryAuthKeyFor_LookupSuccessAndMiss/lookup_success (0.00s)
    --- PASS: TestDiscoveryAuthKeyFor_LookupSuccessAndMiss/lookup_miss:_unregistered_nodeAddr (0.00s)
    --- PASS: TestDiscoveryAuthKeyFor_LookupSuccessAndMiss/lookup_miss:_unregistered_svtnID (0.00s)
--- PASS: TestDeriveDiscoveryKey_SenderRouterAgree (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/routing	0.546s

=== RUN   TestDiscovery_EncodeThenRouterIngest_AcceptsRealAdmittedNode
--- PASS: TestDiscovery_EncodeThenRouterIngest_AcceptsRealAdmittedNode (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/discovery	0.259s

=== RUN   TestDiscovery_Advertise_MissingNodeAdmissionPubkey_FailsClosed
--- PASS: TestDiscovery_Advertise_MissingNodeAdmissionPubkey_FailsClosed (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/discovery	0.259s
```

**Discharge status:** FULL, including the v1.11 F-DWIP1-001 regression fix
and its fail-closed guard.

---

## AC-005 — Fixed-offset key-selector extraction precedes full body decode

**Tape:** `AC-005-006-key-selector-hmac-fail-closed.tape`
**BC anchor:** BC-2.03.001 Postcondition 5; SEC-DW-01 (HIGH, MANDATORY)
**Test file:** `internal/discovery/discovery_wire_test.go`

The router-side ingest path extracts `SVTNID` from raw bytes `body[0:16]`
and `NodeAddr` from raw bytes `body[16:24]` via direct byte-slice indexing
— never via `decodeBody()` — to select the verification key. The HMAC
computation covers the complete raw body, not merely the 24-byte
key-selector prefix, so a forger cannot leave `SVTNID`/`NodeAddr` untouched
while corrupting the session list beneath an otherwise-valid tag. A raw
datagram shorter than 32 bytes (8-byte tag + 24-byte key selector) is
rejected before any key lookup; with the widened `uint64 Sequence`, the
full valid-frame minimum is 42 bytes.

**Evidence commands (success and failure paths):**

```
go test -race -run 'TestRouterIngest_KeySelectorExtraction_FixedOffset_NoFullDecodeBeforeAuth|TestRouterIngest_HMACCoversFullBody_TamperInSessionListDetected' -count=1 -v ./internal/discovery/
go test -race -run 'TestRouterIngest_ShortDatagram_RejectedBeforeLookup|TestRouterIngest_FullValidFrameMinimum_42Bytes' -count=1 -v ./internal/discovery/
```

**Captured output:**

```
=== RUN   TestRouterIngest_KeySelectorExtraction_FixedOffset_NoFullDecodeBeforeAuth
--- PASS: TestRouterIngest_KeySelectorExtraction_FixedOffset_NoFullDecodeBeforeAuth (0.00s)
=== RUN   TestRouterIngest_HMACCoversFullBody_TamperInSessionListDetected
--- PASS: TestRouterIngest_HMACCoversFullBody_TamperInSessionListDetected (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/discovery	0.264s

=== RUN   TestRouterIngest_ShortDatagram_RejectedBeforeLookup
=== RUN   TestRouterIngest_ShortDatagram_RejectedBeforeLookup/raw=0_bytes
=== RUN   TestRouterIngest_ShortDatagram_RejectedBeforeLookup/raw=31_bytes_(one_short_of_the_32-byte_key-selector_minimum)
--- PASS: TestRouterIngest_ShortDatagram_RejectedBeforeLookup (0.00s)
    --- PASS: TestRouterIngest_ShortDatagram_RejectedBeforeLookup/raw=0_bytes (0.00s)
    --- PASS: TestRouterIngest_ShortDatagram_RejectedBeforeLookup/raw=31_bytes_(one_short_of_the_32-byte_key-selector_minimum) (0.00s)
--- PASS: TestRouterIngest_FullValidFrameMinimum_42Bytes (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/discovery	0.264s
```

**Discharge status:** FULL. Failure path (`TestRouterIngest_ShortDatagram_RejectedBeforeLookup`, table-driven at raw=0 and raw=31 bytes) exercised alongside the success path.

---

## AC-006 — HMAC-first fail-closed verification with unified reject sentinel

**Tape:** `AC-005-006-key-selector-hmac-fail-closed.tape`
**BC anchor:** BC-2.03.001 Postcondition 5; SEC-DW-01, SEC-DW-05
**Test file:** `internal/discovery/discovery_wire_test.go`

A lookup-miss (`DiscoveryAuthKeyFor` returns `ok=false`) and an HMAC-tag
mismatch both resolve to the identical `ErrInvalidHMACTag` sentinel, with
no distinguishing return value, log line, or other externally observable
signal. No datagram is relayed and no registry state is mutated on either
rejection path; the read loop continues serving subsequent datagrams
fail-closed.

**Evidence command:**

```
go test -race -run TestRouterIngest_LookupMissAndTagMismatch_IndistinguishableRejection -count=1 -v ./internal/discovery/
```

**Captured output:**

```
=== RUN   TestRouterIngest_LookupMissAndTagMismatch_IndistinguishableRejection
=== RUN   TestRouterIngest_LookupMissAndTagMismatch_IndistinguishableRejection/lookup_miss:_unknown_NodeAddr
=== RUN   TestRouterIngest_LookupMissAndTagMismatch_IndistinguishableRejection/HMAC_tag_mismatch:_known_NodeAddr,_wrong_key
--- PASS: TestRouterIngest_LookupMissAndTagMismatch_IndistinguishableRejection (0.00s)
    --- PASS: TestRouterIngest_LookupMissAndTagMismatch_IndistinguishableRejection/lookup_miss:_unknown_NodeAddr (0.00s)
    --- PASS: TestRouterIngest_LookupMissAndTagMismatch_IndistinguishableRejection/HMAC_tag_mismatch:_known_NodeAddr,_wrong_key (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/discovery	0.264s
```

**Discharge status:** FULL. Both fail-closed rejection paths (lookup-miss, tag-mismatch) exercised as subtests, confirmed indistinguishable.

---

## AC-007 — Node-local relay-ingest: ReceiveAdvertisement retired, ErrSVTNMismatch relocated

**Tape:** `AC-007-node-local-relay-ingest.tape`
**BC anchor:** BC-2.03.001 Postcondition 5 (Ruling 1 point 3, corrected by F-DWSP8-001)
**Test file:** `internal/discovery/discovery_wire_test.go`

`Discovery.ReceiveAdvertisement`, as shipped, is deleted — no caller in the
shipped topology. The new node-side relay-ingest function decodes the hop-2
`DISCOVERY_RELAY` payload with no per-frame HMAC — trust derives from the
admitted connection. `ErrSVTNMismatch` survives, relocated: it compares the
relay frame's `OuterHeader.SVTNID` against `d.cfg.LocalSVTNID` directly
(defense-in-depth, not a crypto check). On success it performs the same
registry replace-on-write update `ReceiveAdvertisement` previously did.

**Evidence command (success and failure paths):**

```
go test -race -run 'TestDiscovery_IngestRelayAdvertisement_SVTNMismatch_ErrSVTNMismatch|TestDiscovery_IngestRelayAdvertisement_NoHMACRequired|TestDiscovery_IngestRelayAdvertisement_Success_RegistryReplaceOnWrite' -count=1 -v ./internal/discovery/
```

**Captured output:**

```
=== RUN   TestDiscovery_IngestRelayAdvertisement_SVTNMismatch_ErrSVTNMismatch
--- PASS: TestDiscovery_IngestRelayAdvertisement_SVTNMismatch_ErrSVTNMismatch (0.00s)
=== RUN   TestDiscovery_IngestRelayAdvertisement_NoHMACRequired
--- PASS: TestDiscovery_IngestRelayAdvertisement_NoHMACRequired (0.00s)
=== RUN   TestDiscovery_IngestRelayAdvertisement_Success_RegistryReplaceOnWrite
--- PASS: TestDiscovery_IngestRelayAdvertisement_Success_RegistryReplaceOnWrite (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/discovery	0.277s
```

**Discharge status:** FULL. Failure path (`ErrSVTNMismatch`) and success path (registry replace-on-write) both exercised.

---

## AC-008 — Cold-start acceptance: first datagram for a (SVTNID, NodeAddr) pair always accepted

**Tape:** `AC-008-009-010-replay-sequence-gate.tape`
**BC anchor:** BC-2.03.001 Postcondition 2 (replay-resistance field note); VP-080
**Test file:** `internal/discovery/discovery_wire_test.go`

With no prior `lastSeen[svtnID, nodeAddr]` entry (fresh router start, or
the first frame from a newly-admitted node), an HMAC-verified datagram with
any declared `Sequence` value (including 0) is accepted; the registry
updates and `lastSeen` is set to the accepted `Sequence`.

**Evidence command:**

```
go test -race -run TestVP080_DiscoveryIngest_ColdStartAcceptance -count=1 -v ./internal/discovery/
```

**Captured output:**

```
=== RUN   TestVP080_DiscoveryIngest_ColdStartAcceptance
--- PASS: TestVP080_DiscoveryIngest_ColdStartAcceptance (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/discovery	0.269s
```

**Discharge status:** FULL.

---

## AC-009 — Replay/stale discard: non-increasing Sequence rejected post-HMAC

**Tape:** `AC-008-009-010-replay-sequence-gate.tape`
**BC anchor:** BC-2.03.001 Postcondition 2; BC-2.03.002 Postcondition 5; VP-080; SEC-DW-07
**Test file:** `internal/discovery/discovery_wire_test.go`

A second HMAC-verified datagram for the same `(SVTNID, NodeAddr)` declaring
`Sequence <= N` (including the exact-replay case `Sequence == N`) is
discarded even though its HMAC passes — the registry is not updated, the
datagram is not relayed, and `lastSeen` is unchanged.

**Evidence command (failure/discard path):**

```
go test -race -run 'TestVP080_DiscoveryIngest_ReplayDiscard_ExactSequence|TestVP080_DiscoveryIngest_ReplayDiscard_LowerSequence|TestVP080_DiscoveryIngest_ReplayDiscard_NoRelaySideEffect' -count=1 -v ./internal/discovery/
```

**Captured output:**

```
=== RUN   TestVP080_DiscoveryIngest_ReplayDiscard_ExactSequence
--- PASS: TestVP080_DiscoveryIngest_ReplayDiscard_ExactSequence (0.00s)
=== RUN   TestVP080_DiscoveryIngest_ReplayDiscard_LowerSequence
--- PASS: TestVP080_DiscoveryIngest_ReplayDiscard_LowerSequence (0.00s)
=== RUN   TestVP080_DiscoveryIngest_ReplayDiscard_NoRelaySideEffect
--- PASS: TestVP080_DiscoveryIngest_ReplayDiscard_NoRelaySideEffect (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/discovery	0.269s
```

**Discharge status:** FULL. All three discard-path variants (exact-replay, lower-sequence, no-relay-side-effect) exercised.

---

## AC-010 — Forward acceptance: strictly-increasing Sequence updates state and triggers relay

**Tape:** `AC-008-009-010-replay-sequence-gate.tape`
**BC anchor:** BC-2.03.001 Postcondition 2; VP-080
**Test file:** `internal/discovery/discovery_wire_test.go`

An HMAC-verified datagram declaring `Sequence = N+1` (or any value `> N`)
is accepted; the registry updates, the accept+relay decision is emitted,
and `lastSeen` advances. A restarted access node's first post-restart
datagram — freshly-sampled `epoch`, low `counter` — is accepted via this
forward-acceptance path (F-DWSP4-001 restart-liveness amendment), not
AC-008's cold-start path.

**Evidence command:**

```
go test -race -run 'TestVP080_DiscoveryIngest_ForwardAcceptance_AdvancesState|TestVP080_DiscoveryIngest_RestartForwardProgress' -count=1 -v ./internal/discovery/
```

**Captured output:**

```
=== RUN   TestVP080_DiscoveryIngest_ForwardAcceptance_AdvancesState
--- PASS: TestVP080_DiscoveryIngest_ForwardAcceptance_AdvancesState (0.00s)
=== RUN   TestVP080_DiscoveryIngest_RestartForwardProgress
--- PASS: TestVP080_DiscoveryIngest_RestartForwardProgress (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/discovery	0.269s
```

**Discharge status:** FULL, including the restart-liveness (epoch-qualified `Sequence`) case.

---

## AC-011 — Bounded, fixed-size UDP read buffer sized to realistic usage

**Tape:** `AC-011-012-013-bounded-buffer-rate-cap-logging.tape`
**BC anchor:** BC-2.03.001 Postcondition 5; SEC-DW-02 (MED)
**Test file:** `internal/discovery/discovery_wire_test.go`

The router's socket-read loop reads each datagram into a fixed-size buffer
sized to the realistic worst-case legitimate advertisement. A datagram
exceeding the sized buffer is rejected without partial-parse and without
reallocation-to-fit.

**Evidence command (failure/rejection path):**

```
go test -race -run TestRouterIngest_OversizedDatagram_RejectedNoPartialParse -count=1 -v ./internal/discovery/
```

**Captured output:**

```
=== RUN   TestRouterIngest_OversizedDatagram_RejectedNoPartialParse
--- PASS: TestRouterIngest_OversizedDatagram_RejectedNoPartialParse (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/discovery	0.276s
```

**Discharge status:** FULL.

---

## AC-012 — Aggregate rate cap at ingest; FailureCounter reused visibility-only

**Tape:** `AC-011-012-013-bounded-buffer-rate-cap-logging.tape`
**BC anchor:** BC-2.03.001 Postcondition 5; SEC-DW-03 (MED)
**Test file:** `internal/discovery/discovery_wire_test.go`

An aggregate (not per-source) token-bucket cap at the socket-read loop
rejects datagrams once the aggregate rate is exceeded, regardless of
declared `NodeAddr` — a source rotating its declared `NodeAddr` across
forged datagrams does not evade the cap. The existing `FailureCounter`
(threshold=5/60s) is invoked on HMAC-rejection events for operator
visibility only; it never gates admission or ingest on the declared,
attacker-controlled `NodeAddr`.

**Evidence command:**

```
go test -race -run 'TestRouterIngest_AggregateRateCap_NotPerSource|TestRouterIngest_FailureCounter_VisibilityOnly_NeverGates' -count=1 -v ./internal/discovery/
```

**Captured output:**

```
=== RUN   TestRouterIngest_AggregateRateCap_NotPerSource
--- PASS: TestRouterIngest_AggregateRateCap_NotPerSource (0.01s)
=== RUN   TestRouterIngest_FailureCounter_VisibilityOnly_NeverGates
--- PASS: TestRouterIngest_FailureCounter_VisibilityOnly_NeverGates (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/discovery	0.276s
```

**Discharge status:** FULL.

---

## AC-013 — Rate-limited, counter-based failure logging

**Tape:** `AC-011-012-013-bounded-buffer-rate-cap-logging.tape`
**BC anchor:** BC-2.03.001 Postcondition 5; SEC-DW-04 (MED)
**Test file:** `internal/discovery/discovery_wire_test.go`

Discovery HMAC-rejection logging fires only on `FailureCounter`'s own
threshold-crossing emission, not unconditionally per rejected packet —
explicitly distinct from BC-2.05.008's per-packet TCP HMAC-failure logging
policy.

**Evidence command:**

```
go test -race -run TestRouterIngest_FailureLogging_ThresholdCrossingOnly_NotPerPacket -count=1 -v ./internal/discovery/
```

**Captured output:**

```
=== RUN   TestRouterIngest_FailureLogging_ThresholdCrossingOnly_NotPerPacket
--- PASS: TestRouterIngest_FailureLogging_ThresholdCrossingOnly_NotPerPacket (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/discovery	0.276s
```

**Discharge status:** FULL.

---

## AC-014 — DISCOVERY_RELAY frame assembly: control_type=0x03 payload layout

**Tape:** `AC-014-015-016-relay-frame-assembly.tape`
**BC anchor:** BC-2.01.008 Postcondition 2, Postcondition 3, Invariant 5/DI-007; BC-2.03.001 Postcondition 5
**Test file:** `cmd/switchboard/discovery_relay_wire_test.go`

The relay frame is a `FrameTypeCtl` (`0x03`) outer frame whose payload
begins with `control_type=0x03, version=0x01, reserved=0x0000` at bytes
0-3; bytes 4-11 carry the originating access node's 8-byte `NodeAddr`;
bytes 12-19 carry the `Sequence` value (uint64 BE, epoch-qualified); bytes
20-21 carry the session count (uint16 BE); bytes 22+ carry the per-session
list. `SVTNID` is not repeated in the payload. Frame assembly is a pure
function testable independent of any live connection or dispatch mechanism.

**Evidence command:**

```
go test -race -run TestAssembleDiscoveryRelayFrame_PayloadLayout -count=1 -v ./cmd/switchboard/
```

**Captured output:**

```
=== RUN   TestAssembleDiscoveryRelayFrame_PayloadLayout
--- PASS: TestAssembleDiscoveryRelayFrame_PayloadLayout (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/cmd/switchboard	0.300s
```

**Discharge status:** FULL. Also exercised in this tape: the full hop-1-decode→hop-2-assemble round-trip (`TestAssembleDiscoveryRelayFrame_IngestRelayAdvertisement_RoundTrip`) and the oversize-payload panic guard (`TestAssembleDiscoveryRelayFrame_PayloadOversize_Panics`), both PASS.

---

## AC-015 — Zero HMACTag on relay frame: connection-trust boundary

**Tape:** `AC-014-015-016-relay-frame-assembly.tape`
**BC anchor:** BC-2.03.001 Postcondition 1 (delivery-mechanism note); SEC-DW-08 (hop-2 half)
**Test file:** `cmd/switchboard/discovery_relay_wire_test.go`

The `DISCOVERY_RELAY` frame's `OuterHeader.HMACTag` is the zero value —
matching the DRAIN precedent (`S-7.04-FU-DRAIN-WIRE`) exactly. No
per-frame HMAC is computed for hop-2; the receiving node's trust in the
relayed content derives exclusively from its own already-admitted TCP
connection to the router.

**Evidence command:**

```
go test -race -run TestAssembleDiscoveryRelayFrame_ZeroHMACTag -count=1 -v ./cmd/switchboard/
```

**Captured output:**

```
=== RUN   TestAssembleDiscoveryRelayFrame_ZeroHMACTag
--- PASS: TestAssembleDiscoveryRelayFrame_ZeroHMACTag (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/cmd/switchboard	0.300s
```

**Discharge status:** FULL.

---

## AC-016 — Payload is re-serialized, never a raw retransmission of hop-1 bytes

**Tape:** `AC-014-015-016-relay-frame-assembly.tape`
**BC anchor:** BC-2.03.001 Postcondition 5 (relay/connection-trust note)
**Test file:** `cmd/switchboard/discovery_relay_wire_test.go`

The relay frame's payload bytes are freshly constructed from the decoded
`NodeAddr`, `Sequence`, and session-list fields — never a byte-for-byte
copy of hop-1's raw UDP datagram. Hop-1's original HMAC tag never appears
anywhere in the relay frame.

**Evidence command:**

```
go test -race -run TestAssembleDiscoveryRelayFrame_NotRawHop1Bytes -count=1 -v ./cmd/switchboard/
```

**Captured output:**

```
=== RUN   TestAssembleDiscoveryRelayFrame_NotRawHop1Bytes
--- PASS: TestAssembleDiscoveryRelayFrame_NotRawHop1Bytes (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/cmd/switchboard	0.300s
```

**Discharge status:** FULL.

---

## Coverage gap — AC-017/AC-018, Task 6 (intentional, documented)

`AC-017` (SVTN-scoped, exclude-originator, best-effort fan-out dispatch)
and `AC-018` (relay-dispatch rate cap) are marked
`[GATED — depends_on S-BL.NODE-IDENTIFY-WIRE]` in the story spec. Task 6
(hop-2 fan-out dispatch) is explicitly out of scope for this story's
Tasks 1-5 delivery. Neither AC has an implementation, a test, or a demo in
this evidence set — `grep -rn "TestRelayDispatch" --include="*.go" .`
returns zero matches in this worktree, confirming no stub or partial test
exists yet. This is the story's own documented scope boundary (Human Gate
item 3 / Forward Obligations table), not an evidence gap introduced by this
report — the story's Task Breakdown states plainly: "Tasks 1-5 (hop-1
ingest, sender dispatch, hop-2 frame construction) are independently
deliverable and do not depend on `S-BL.NODE-IDENTIFY-WIRE` landing. Task 6
(hop-2 fan-out dispatch) is explicitly GATED."

---

## Test-suite confirmation

Full package-level confirmation that every test cited above passes under
`go test -race` alongside the rest of the suite (no isolated-run-only
passes):

```
go test ./cmd/switchboard/... ./internal/discovery/... ./internal/hmac/... ./internal/routing/... ./internal/testenv/... -v
```

All relevant packages report `ok` with zero `FAIL` lines; every test named
in the Summary Table above appears with `--- PASS`.

**Code HEAD:** `501db03a99db7a06586988e133279cb43a11e021` on
`feature/S-BL.DISCOVERY-WIRE`.
