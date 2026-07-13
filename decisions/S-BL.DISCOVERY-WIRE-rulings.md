---
artifact_id: S-BL.DISCOVERY-WIRE-rulings
document_type: decision
level: ops
version: "1.3"
status: final
producer: architect
timestamp: 2026-07-13T00:00:00Z
updated: 2026-07-13T22:00:00Z
cycle: cycle-1
stories_in_scope: [S-BL.DISCOVERY-WIRE]
bc_traces:
  - BC-2.03.001
  - BC-2.03.002
  - BC-2.05.005
  - BC-2.01.008
closes_findings: []
resolves: [DRIFT-W6TBD-001]
---

# Ruling: S-BL.DISCOVERY-WIRE — Two Open Design Obligations

All factual claims below are grep/read-verified against the tree at commit
`1f25677d00a3f6bc5f96f1a0a0571033ade9eb6a` (develop HEAD, post
S-BL.CLI-SURFACE-COMPLETION merge). File:line anchors are cited per claim.

This ruling resolves the two Open Design Obligations blocking story
decomposition of `S-BL.DISCOVERY-WIRE`. It does not edit the story, the
anchored BCs, `ARCH-03`, or any other artifact — those edits belong to the
product-owner / story-writer / spec-steward, and are enumerated as explicit
follow-on actions at the end of each ruling and in the summary table.

**Headline finding surfaced while adjudicating Ruling 2:** the story stub's
scope item 3 ("replace ... with `net.ListenMulticastUDP` dispatch goroutine")
and `ARCH-03`'s discovery sketch ("consoles subscribe to multicast") are in
direct conflict with an already-ratified domain invariant (**DI-004** — "no
direct node-to-node communication") and with **BC-2.03.001's own Invariant 1**
("Advertisements flow node-to-router-to-node via the SVTN; no direct
node-to-node multicast"). This is not a new judgment call I am inventing — the
BC already decided this — but the story stub and `ARCH-03` were drafted from
an earlier, unreconciled sketch and need correction. See Ruling 2.

**v1.1 (this revision):** folds in a completed security-reviewer consult
(RECOMMENDED, not blocking, per v1.0) covering both rulings. Verdict:
**RULING-1-SOUND-WITH-CONSTRAINTS**, **RULING-2-SOUND-WITH-CONSTRAINTS**,
9 findings (SEC-DW-01..09), all additive — neither ruling's core decision
was overturned. Full disposition in the Security Consult Addendum inside
each ruling. Since this document has never been committed, Implementation
Constraint prose is rewritten in place to its final v1.1 shape rather than
carrying stale v1.0 text forward under a strikethrough; the Decision Log
and each Addendum preserve the v1.0→v1.1 delta for audit purposes. Text
that stated a *decision* (the concrete key-derivation rule, the BC
amendment drafts) is treated as superseded-and-noted, not silently
overwritten — see each Addendum for the explicit "supersedes" callout.

---

## Verified Premises

| Premise | File:Line | Evidence |
|---|---|---|
| S-7.02 placeholder: `advertisementKey(svtnID) = svtnID`, retained after RULING-W6TB-H's reorder | `internal/discovery/discovery.go:415-417` | `func advertisementKey(svtnID [16]byte) [16]byte { return svtnID }` |
| `ReceiveAdvertisement` is HMAC-first per RULING-W6TB-H: decode → derive key from `payload.SVTNID` → verify HMAC → SVTN check | `internal/discovery/discovery.go:297-330` | Comment block at :292-296 + code at :310-330 |
| DI-004 is an unconditional domain invariant: all inter-node traffic passes through a router; a node has no mechanism to discover/contact another node's address directly | `.factory/specs/domain-spec/invariants.md:66-72` | "**DI-004 — No direct node-to-node communication** ... A node has no mechanism to discover or contact another node's network address directly. Admission control is enforced at the router; bypassing the router bypasses admission control." |
| BC-2.03.001 Invariant 1 restates DI-004 specifically for advertisements | `.factory/specs/behavioral-contracts/ss-03/BC-2.03.001.md:61` | "**DI-004**: Advertisements flow node-to-router-to-node via the SVTN; no direct node-to-node multicast." |
| `ARCH-03`'s discovery sketch contradicts DI-004/BC-2.03.001 as currently worded | `.factory/specs/architecture/ARCH-03-routing-engine.md:299-308` | "Access nodes send `PRESENCE_ADV` frames to a well-known SVTN multicast address. Consoles subscribe to multicast..." — no router-mediation stated; marked "architecture sketch for PE," not finalized |
| DI-006: every SVTN-scoped frame is HMAC-verified against the admitted key set by the **first router**, before forwarding | `.factory/specs/domain-spec/invariants.md:81-86` | "A frame that passes HMAC verification was originated by a node holding a private key registered against that SVTN." |
| The admitted-node HMAC key vocabulary DRIFT-W6TBD-001 asks for already exists, shipped, for point-to-point frame auth | `internal/hmac/hmac.go:110-126`; `internal/admission/admission.go:154-156,241-242` | `hmac.DeriveKey(nodeAdmissionPubkey []byte, svtnID [16]byte) [32]byte` — HKDF-SHA256(IKM=admission pubkey, salt=svtnID, info="switchboard-frame-auth"); `AdmittedKey.FrameAuthKey` computed at `RegisterKey` time via this exact call |
| The 8-byte outer-header HMAC tag is explicitly documented as **not a standalone security primitive** — it is a router-path integrity signal layered on top of Tier-1 admission | `.factory/specs/architecture/ARCH-02-protocol-stack.md:44-48` (ADR-001) | "64 bits is used here as a router-path integrity signal — not a standalone security primitive. Full frame authentication binds per-node keying per ARCH-04 F-003." |
| `AdmittedKeySet.Lookup(svtnID, nodeAddr)` already returns a value-copy `AdmittedKey` (incl. `FrameAuthKey`) — existing, audited accessor | `internal/admission/admission.go:363-381` | Deep-clones `PublicKey`; safe for concurrent read (go.md rule 12) |
| `internal/discovery` is ARCH-08-position-14, boundary-classified, permitted to import ONLY `internal/routing` — `internal/admission` and `internal/hmac` are forbidden imports | `.factory/specs/architecture/ARCH-08-dependency-graph.md:159,173,330`; `internal/discovery/discovery.go:1-9,19-30` | "14. internal/discovery (imports: routing)"; package doc: "discovery→routing is legal; discovery→hmac and discovery→frame are forbidden" |
| `internal/routing.Router` already holds a reference to the SAME `*admission.AdmittedKeySet` that stores `FrameAuthKey` | `internal/routing/routing.go:150-152,163-166` | `Router{ admittedKeySet *admission.AdmittedKeySet, ... }`; `NewRouter(ks *admission.AdmittedKeySet, ...)` |
| `discovery.Config` already carries a `*routing.Router` field, currently used only for the two thin HMAC-wrapper calls | `internal/discovery/discovery.go:117-120` | `Router *routing.Router // used for HMAC authentication...` |
| `routing.RouteFrame`'s existing verification ordering (forwarding-table lookup by declared `hdr.SVTNID`+`hdr.SrcAddr` → HMAC verify → admitted-set check) is the direct precedent for "wire-declared SVTN ID plus locally-held admitted-node material" | `internal/routing/routing.go:227-307` | Step 1 comment: "Look up forwarding-table entry for (hdr.SVTNID, hdr.SrcAddr). If absent → no auth key → ... fail-closed." |
| The data plane's only existing socket transport is TCP (`net.Listen("tcp", cfg.ListenAddr)`), bound in router mode; there is no existing UDP surface anywhere in `cmd/switchboard` | `cmd/switchboard/mgmt_wire.go:510-513` | Phase (d) of `runRouter` |
| Four daemon modes exist today: router, access, console, control — each with its own `run*` entry point in `cmd/switchboard` | `cmd/switchboard/mgmt_wire.go:459,973,1076`; `cmd/switchboard/access.go:128` | `runRouter`, `runConsole`, `runControl`, `runAccess` |
| The router-mode-exclusive wiring pattern (new `wireXHandlers` function, called from `runRouter` at Phase (c)/(c2), before `serveMgmtServer`) is already shipped precedent, not a hypothetical | `cmd/switchboard/mgmt_wire.go:493-505` | `wireMetricsHandlers` (Phase c) + `wireRouterControlHandlers` (Phase c2), both register-before-serve |
| SVTN IDs are 128-bit `crypto/rand`-generated, not sequential or predictable | `internal/svtnmgmt/svtnmgmt.go:129-140,192-195,694` | `NewSVTNManager` uses `rand.Reader`; `Create` generates the ID from `m.randSource` |
| No IPv6 data-plane precedent exists anywhere in the codebase; the only IPv6 references are mgmt-plane loopback authorization (`[::1]`) | grep across `.factory/specs/architecture/*.md`, `internal/`, `cmd/` | `ARCH-05-cli-and-api.md:174,188`; `ARCH-12-daemon-management-plane.md:1178-1187` — both scoped to mgmt-plane loopback, not data plane |
| `internal/testenv.NewLoopback` is a VP-042-scoped compile-shim, unrelated to multicast; `internal/testenv.Env` already models multi-router topologies (`nRouters`, `routers []*RouterHandle`) | `internal/testenv/testenv.go:363-387,392-437` | No multicast fixture exists today |
| Story `scope_phase: PE` denotes the router-to-router **peering** phase (multi-router topology), a phase beyond the current single-router MVP | `.factory/specs/architecture/ARCH-INDEX.md:157,161`; `.factory/specs/architecture/ARCH-02-protocol-stack.md:205` | "Router-to-router peering in PE phase uses Noise XX..."; "does router-to-router Noise handshake reuse the same keypair as node admission..." |
| A per-source `FailureCounter` (threshold=5 failures / 60s window) already exists and is wired to every router-mode `Router` for TCP HMAC-failure alerting (E-ADM-017, BC-2.05.005 PC-3) — precedent for SEC-DW-03's visibility-only reuse | `cmd/switchboard/mgmt_wire.go:482-491` (`buildRouter` comment); `internal/routing/routing.go:276-279,293-296` (`r.failureCounter.RecordHMACFailure(...)`) | "buildRouter ... constructs a routing.Router with ... a FailureCounter at threshold=5, window=60s per BC-2.05.005 PC-3" |
| `admission.Create`/`nonceTTL` already accepts a *bounded, not perfect* replay-prevention window (60s) as this project's precedent for anti-replay design in the same subsystem family — direct grounding for SEC-DW-07's cold-start accept-window | `internal/admission/admission.go:142-145` | `const nonceTTL = 60 * time.Second` |
| DI-003 is scoped to **router-compromise** threat models specifically ("When a router is compromised, the attacker gains..."), not to plain network-eavesdropper replay of legitimately-captured, still-valid-HMAC traffic — relevant to SEC-DW-07's adjudication | `.factory/specs/domain-spec/invariants.md:53-60` | "**DI-003 — Router compromise degrades availability, not confidentiality** ... The attacker cannot gain: ... the ability to inject content that a legitimate node will accept as authentic." |

---

## Ruling 1 — Admitted-node HMAC key derivation: reuse the shipped per-(node,SVTN) `FrameAuthKey`, verified at the router

**DECISION (v1.1 — folds in a completed security consult; see Security
Consult Addendum below): Retire `advertisementKey(svtnID) = svtnID`.
Replace it with a lookup of a domain-separated discovery key,
`DiscoveryAuthKey := hmac.DeriveDiscoveryKey(nodeAdmissionPubkey, svtnID)`
— the same HKDF-SHA256 construction (ADR-001) and the same
`(nodeAdmissionPubkey, svtnID)` inputs `internal/admission.RegisterKey`
already uses for the session-data `FrameAuthKey`, but with a distinct,
discovery-specific HKDF info label so the two derived keys are
cryptographically independent (SEC-DW-06). No new KDF primitive, no new
secret-material class — a second, purpose-bound derivation of the same
already-shipped function shape. Verification happens exclusively at the
router (Ruling 2's router-relay model), gated by fixed-offset key-selector
extraction (SEC-DW-01) and a per-(SVTN,node) monotonic sequence check
against replay (SEC-DW-07). Access/console nodes never independently
re-derive or re-verify this key.**

### Rationale

**Why not invent a new key-derivation primitive.** DRIFT-W6TBD-001's own
complaint is precise: `svtnID` is "not admitted-node-scoped secret
material — any observer of one advertisement can compute the key." The
fix implied by that complaint is "derive the key from something that only
an admitted node (post-Tier-1) possesses" — and that vocabulary already
exists, shipped, audited, and reused three times in this codebase
(`AdmittedKey.FrameAuthKey`, `routing.ForwardingEntry.FrameAuthKey`,
`outerassembler.Envelope.FrameAuthKey` — all the same `[32]byte` value per
`outerassembler/assemble.go:30-32`). Inventing a second, discovery-specific
KDF or secret-material class when this one already satisfies "keying
material established at Tier-1 admission" (the RULING-W6TB-D framing,
`RULING-W6TB-D-discovery-scope.md:61-63`) would be duplicate machinery with
no offsetting benefit, and would need its own from-scratch security
argument. Reuse is the minimal, most-precedented, lowest-risk option.

**An alternative was considered and rejected: a new per-SVTN broadcast
group secret.** Because `advertisementKey` as RULING-W6TB-H shipped it is a
function of `svtnID` alone (not `svtnID` + a node address), one might read
DRIFT-W6TBD-001 as calling for a per-SVTN *group* key (a single shared
secret for the whole SVTN's discovery channel, generated at
`admin.svtn.create` time and distributed to each node post-admission).
This was considered and rejected for two reasons: (1) it requires new
state (an `SVTNManager`-held secret, a new distribution step riding on the
post-challenge-response admission response) that does not exist today and
has no shipped precedent — a materially larger and riskier change than
Ruling 2's finding makes necessary; (2) it is unnecessary once Ruling 2
establishes that advertisements are **router-relayed, not
peer-multicast** — a per-SVTN group key is the right primitive for
"any member can verify any other member's raw broadcast," but under the
router-relay model there is only ever ONE verifier (the router), talking to
one sender at a time, which is exactly the point-to-point shape
`FrameAuthKey` + `RouteFrame` already solve. The group-key shape is not a
security improvement here — it is unneeded generality with a bigger
implementation and review surface for zero behavioral benefit.

**Why the reuse is safe, not merely convenient.** ADR-001 (`ARCH-02
:44-48`) already documents that the 8-byte outer-header HMAC tag is "a
router-path integrity signal — not a standalone security primitive," with
the *real* authentication gate being Tier-1 admission (Ed25519
challenge-response, DI-002 private-key-never-transits) plus the
router-side admitted/forwarding-table check (DI-006). Reusing
`FrameAuthKey` for discovery frames does not change that risk model — it
extends an already-accepted layered design to a new frame class, rather
than introducing a new one. `FrameAuthKey`'s IKM (`nodeAdmissionPubkey`) is
not itself required to be secret for this design to hold: forging a valid
tag requires knowing (a) a specific admitted node's registered public key
and (b) the target SVTN ID, and even then a forged frame is only accepted
if a live forwarding-table/admitted-key entry exists for that
`(SVTNID, NodeAddr)` pair — i.e., a real admission event already happened
for that identity. This is the identical trust boundary `RouteFrame`
already operates under for every other SVTN frame type; nothing new is
being asked of it.

**Why verification happens only at the router.** This falls directly out
of Ruling 2: DI-004 forbids direct node-to-node delivery, so the only
process that ever authenticates raw advertisement bytes off the wire is
the router (the "first router" DI-006 names). Access/console nodes receive
advertisements only via the router's relay, over their own
already-authenticated channel to the router — the same trust model
ordinary routed frames already use (a destination node does not
re-verify the *original sender's* HMAC tag on relayed traffic; it trusts
what its own authenticated connection delivers). This means
`ReceiveAdvertisement`'s raw-bytes HMAC-verification code path is, after
this story, **router-side only** — see Implementation Constraints below
for what that means for the existing function's callers.

### Concrete derivation rule (for BC-2.03.001 PC-5 amendment)

**Supersedes the v1.0 draft below** (SEC-DW-06, adopted — see Addendum).
The v1.0 text proposed reusing `hmac.DeriveKey`/`FrameAuthKey` verbatim
(same derived key for session-data frames and discovery frames). The
security consult flagged that discovery's ingest surface is now
pre-authentication-reachable in a way session-data's TCP-handshake-gated
surface is not, and recommended domain-separating the two derived keys so
a weakness specific to the new, more-exposed surface cannot be leveraged
against the existing session-data key. Cost is one new HKDF info-label
constant and one new sibling function — no change to any already-shipped
code path (`DeriveKey`, `RegisterKey`, `AdmittedKey.FrameAuthKey` are
untouched). Replace the PC-5 key-placeholder note with:

> **Key derivation (Ruling S-BL.DISCOVERY-WIRE-1, v1.1):** The HMAC key
> authenticating an advertisement frame is `DiscoveryAuthKey :=
> hmac.DeriveDiscoveryKey(nodeAdmissionPubkey, svtnID)` — HKDF-SHA256 over
> the same `(nodeAdmissionPubkey, svtnID)` inputs `internal/admission`
> already uses for the session-data `frame_auth_key` (ADR-001), but with a
> distinct info label (`HKDFInfoDiscovery = "switchboard-discovery-auth"`,
> vs. the existing `HKDFInfo = "switchboard-frame-auth"`) so the two keys
> are cryptographically independent — a compromise of one does not imply
> the other. No new KDF primitive is introduced; `hkdfSHA256` (the
> underlying HKDF implementation) is unchanged and shared by both call
> sites. The key is verified exclusively by the router that receives the
> advertisement off the discovery multicast socket (the "first router" per
> DI-006), using fixed-offset extraction of only the key-selector fields
> (`SVTNID`, `NodeAddr`) before any variable-length body content is parsed
> (SEC-DW-01) — access and console nodes never independently look up or
> re-verify another node's `DiscoveryAuthKey`; they receive
> already-authenticated advertisements via the router's relay over their
> own admitted connection. The router additionally enforces a
> per-(SVTN,NodeAddr) monotonic sequence check to reject replayed frames
> (SEC-DW-07 — see the dedicated subsection below).

### Where the key lives

Nowhere new. `DiscoveryAuthKey` is computed on demand from data already in
`AdmittedKeySet` — specifically the `PublicKey` field `Lookup` already
returns (`internal/admission/admission.go:363-381`), fed into
`hmac.DeriveDiscoveryKey`. It is **not** cached as a new field on
`admission.AdmittedKey` (unlike `FrameAuthKey`, which is precomputed once
at `RegisterKey` time) — recomputing on demand from the already-returned
`PublicKey` is cheap (one HKDF call per verified datagram) and avoids
touching the `AdmittedKey` struct or `RegisterKey` at all, keeping this
change fully additive to `internal/admission`. No new storage, no new
distribution step.

### Implementation constraints (v1.1 — supersedes the v1.0 draft; see Security Consult Addendum)

1. **New lookup surface on `*routing.Router`**, preserving the ARCH-08
   §6.5 position-14 boundary (`discovery→routing` legal,
   `discovery→admission` and `discovery→hmac` forbidden). Named
   `DiscoveryAuthKeyFor` (renamed from the v1.0 draft's `FrameAuthKeyFor`
   per SEC-DW-06 — it derives a distinct key, not the shared `FrameAuthKey`):

   ```go
   // internal/routing/advertisement_hmac.go (extend, don't create a new file)
   func (r *Router) DiscoveryAuthKeyFor(svtnID [16]byte, nodeAddr [8]byte) ([hmac.KeySize]byte, bool) {
       ak, ok := r.admittedKeySet.Lookup(svtnID, nodeAddr) // existing method, admission.go:363
       if !ok {
           return [hmac.KeySize]byte{}, false
       }
       return hmac.DeriveDiscoveryKey([]byte(ak.PublicKey), svtnID), true
   }
   ```

   `Router` already stores `admittedKeySet` (unexported); this is a thin
   read-only wrapper, same shape as `ComputeAdvertisementHMAC`/
   `VerifyAdvertisementHMAC`'s existing pass-through pattern
   (`internal/routing/advertisement_hmac.go:18-29`).

2. **`advertisementKey(svtnID [16]byte) [16]byte` is deleted. Key-selector
   extraction MUST use fixed offsets, never a full body decode, before
   HMAC verification (SEC-DW-01, HIGH, CWE-770/CWE-400 — MANDATORY, not
   optional hardening).** Per the wire layout `encodeBody` already
   establishes (raw wire bytes = `[8]tag | [16]SVTNID | [8]NodeAddr |
   uint16 count | sessions...`, i.e. `SVTNID` at raw bytes 8-24 / `body[0:16]`
   and `NodeAddr` at raw bytes 24-32 / `body[16:24]`), the router-side
   ingest path (new code, see Ruling 2 — this is NOT the existing
   `Discovery.ReceiveAdvertisement` method; see point 3) reads **only**
   `body[0:16]` as the declared SVTN ID and `body[16:24]` as the declared
   node address via direct byte-slice indexing to select the verification
   key — it MUST NOT call the full `decodeBody()` (which walks the
   variable-length, attacker-controlled session-entry list, up to
   `maxSessionsPerAdvertisement` per-session heap allocations) until
   *after* HMAC verification succeeds. **HMAC verification itself still
   runs over the complete raw body bytes** (required for integrity — the
   MAC must cover every byte an attacker could tamper with, not merely the
   16-byte key-selector prefix, or a forger could leave the declared
   `SVTNID`/`NodeAddr` untouched while corrupting the session list beneath
   an otherwise-valid tag); only the *decode/parse* step is deferred, not
   the *coverage* of the MAC. This mirrors the property `RouteFrame`
   already has on the TCP path — content-blind key selection (its
   selectors live in the fixed 44-byte `OuterHeader`, R-001 "payload is
   never parsed here") — achieved here via fixed-offset extraction instead
   of a fixed-size header, since the SVTNID/NodeAddr key selectors happen
   to precede the variable-length region in this wire format. Full
   `decodeBody()` — sessions, names, statuses, quality — is the
   trusted-content parse and runs only once authentication has passed.
   This is a correction to the v1.0 draft, which said "decodes the body to
   get `payload.SVTNID` and `payload.NodeAddr`" without specifying *how* —
   ambiguous enough to permit an unsafe full-decode-before-auth reading.
   Rationale: running the session-entry parse before authentication turns
   every unauthenticated UDP datagram into free CPU/allocation work for an
   attacker, which the prior TCP-only surface never exposed (a TCP
   connection costs the attacker a completed handshake first; a UDP send
   costs nothing). `router.DiscoveryAuthKeyFor(svtnID, nodeAddr)` is then
   called with the fixed-offset-extracted values, and a `false` return (no
   entry — not admitted, or wrong SVTN) is treated identically to an HMAC
   mismatch: `ErrInvalidHMACTag`, fail-closed, **before** any
   distinguishing signal is returned — a strict tightening of the
   RULING-W6TB-H oracle-closing property (Ruling 2's model removes even
   the possibility of computing a foreign SVTN's key, since the lookup is
   keyed by the exact declared `(SVTNID, NodeAddr)` pair against real
   admission state, not a pure function of `SVTNID` alone).

3. **`Discovery.ReceiveAdvertisement` (node-local, single-SVTN-scoped,
   `d.cfg.LocalSVTNID`-based) is preserved unchanged in shape** — including
   RULING-W6TB-H's HMAC-first ordering and the `ErrSVTNMismatch` sentinel —
   but its role changes to **defense-in-depth**, exactly the pattern the
   architect ruled for AC-011 PC-3 in
   `S-BL.CLI-SURFACE-COMPLETION-rulings.md` (v1.1 addendum, `E-CFG-011`
   shape): under a correctly-functioning router relay, a node only ever
   receives advertisements for SVTNs it is legitimately admitted to, so
   `ErrSVTNMismatch` should be unreachable via any live topology. It stays
   in the code as an explicit guard against a future relay bug that
   mis-routes across SVTN boundaries, and the existing test
   `TestDiscovery_VP045_SVTNIsolation_MultipleScopes`
   (`internal/discovery/discovery_test.go:1121`) continues to pass
   unchanged — **no test regression, no story-writer action needed on that
   test.** Whether hop-2 delivery re-invokes `ReceiveAdvertisement` verbatim
   (feeding it router-relayed bytes still carrying the original sender's
   tag) or a lighter trusted-ingest path is used is a decomposition-time
   implementation choice — see Forward Obligation (b).

4. **`Encode`/package-level `Decode`** (used for round-trip tests, AC-004)
   need their own `advertisementKey` call sites updated to the new
   lookup-based scheme where they are used to construct outbound wire
   bytes on the sending (access) node's side. On the sender side, the node
   computes its own `DiscoveryAuthKey` the same way the router's lookup
   does — `hmac.DeriveDiscoveryKey(ownPubkey, ownSVTNID)` — which it can do
   without querying the router (both inputs are locally known to the node:
   its own public key and its own SVTN ID). This requires the same
   `internal/routing` pass-through treatment as `ComputeAdvertisementHMAC`
   already gets; add a `routing.DeriveDiscoveryKey(pubkey []byte, svtnID
   [16]byte) [hmac.KeySize]byte` thin wrapper over `hmac.DeriveDiscoveryKey`
   so `internal/discovery` never imports `internal/hmac` directly
   (preserves the ARCH-08 boundary).

5. **Bounded, fixed-size UDP read buffer, sized to realistic legitimate
   usage — not the UDP/IP maximum (SEC-DW-02, MED, CWE-770/CWE-400).** The
   router's socket-read loop MUST read each datagram into a fixed buffer
   sized to the realistic worst-case *legitimate* advertisement, not the
   65,507-byte UDP/IP theoretical maximum — any datagram exceeding the
   sized buffer is rejected without further processing (no partial-parse,
   no reallocation-to-fit). `maxSessionsPerAdvertisement` (currently
   `1024`, `discovery.go:528`) was sized against the old TCP/length-prefixed
   framing assumption with no grounding in real usage; it MUST be
   re-derived at decomposition time from the actual expected
   tmux-sessions-per-access-node scale — likely low hundreds, not 1024.
   This is a usage-scale sizing question (what does a legitimate access
   node actually publish?), not a network-MTU question — flagged as an
   implementer task since the real number depends on product usage data
   not yet gathered, not on anything this ruling can derive from the wire
   format alone.

6. **Two-layer rate defense at the socket-read loop (SEC-DW-03, MED,
   CWE-406/CWE-770).** (a) A separate **aggregate** (not per-source)
   token-bucket cap at the discovery listener's socket-read loop as the
   actual backstop — aggregate, because a per-source cap keyed on the
   declared `NodeAddr` is trivially defeated by an attacker rotating
   spoofed source identities before HMAC has run; an aggregate cap cannot
   be defeated that way. (b) Separately, reuse the existing per-source
   `FailureCounter` (threshold=5/60s, BC-2.05.005 PC-3, already wired via
   `buildRouter`, `cmd/switchboard/access.go`), keyed by declared
   `NodeAddr`, for **operator visibility** into sustained forgery attempts
   against a specific identity — but document explicitly that this is a
   visibility signal, not a DoS-prevention control, and **never** a
   per-NodeAddr admission or rate gate: the declared `NodeAddr` in a
   not-yet-authenticated datagram is attacker-controlled, so gating on it
   pre-auth would let an attacker spoof a legitimate node's address to
   trip the counter and effectively block that node.

7. **Rate-limited, counter-based failure logging, reusing FailureCounter's
   own threshold-crossing emission — does NOT inherit BC-2.05.008's
   per-packet TCP logging policy (SEC-DW-04, MED, CWE-400/CWE-770/CWE-779).**
   `RouteFrame`'s existing pattern logs every HMAC failure individually
   (`internal/routing/routing.go:272-279,289-296`), which is safe on TCP
   because each attempt already cost the attacker a completed handshake —
   an attacker who can reach the discovery multicast group has no
   equivalent cost and must not be able to drive unbounded log volume.
   Discovery's router-side ingest MUST reuse the same `FailureCounter`
   mechanism's threshold-crossing emission as the logging trigger (not
   unconditional per-packet logging, and not a separately-invented
   rate-limiter) — a log line fires when the counter crosses its
   threshold, not on every individual rejected datagram.

8. **No wire-visible accept/reject differential (SEC-DW-05, LOW/INFO,
   CWE-208, MUST).** Advertisements are one-way, fire-and-forget UDP with
   no acknowledgment, so there is no *response-content* oracle to close by
   construction — the security consult explicitly endorses the already-shipped
   design choices here: silent-drop-for-all-failures is correct, and
   unifying "lookup-miss" with "HMAC-mismatch" into the single
   `ErrInvalidHMACTag` sentinel (Implementation Constraint 2 above) is
   correct. The residual is narrower: **processing-time**. Datagram
   inter-arrival timing is observable to anyone on the shared multicast
   segment, so the router's processing time for a "lookup miss" (unknown
   `NodeAddr`) MUST NOT be observably different from a "lookup hit, tag
   mismatch" (known `NodeAddr`, wrong key) — the ingest path MUST NOT emit
   any wire-visible reply, counter, or other externally observable signal
   that differs between accept/reject outcomes. Optional hardening, not
   required for this story: on lookup-miss, compute a dummy HMAC
   verification against a fixed placeholder key before returning, so
   per-packet processing time does not vary detectably between the two
   rejection paths.

### Replay / freshness (SEC-DW-07, MED, CWE-294 — my adjudication, flagged for the human gate)

**Finding, confirmed against the shipped registry-update code.**
Neither `AdvertisementPayload` nor the HMAC scheme (v1.0 or v1.1) carries
any timestamp, nonce, or sequence field — the payload is `SVTNID +
NodeAddr + count + sessions` only. `ReceiveAdvertisement`'s registry
update is **unconditional replace-on-write**: on every accepted
advertisement it deletes all prior registry entries for `payload.NodeAddr`
and inserts whatever the new payload says
(`internal/discovery/discovery.go:334-348`), with no ordering or
freshness check. An attacker who captures one valid, HMAC-authenticated
advertisement datagram (trivial on a multicast segment — no admission
bypass or router compromise needed to *observe* traffic, only to inject
it usefully) can replay it indefinitely; because node revocation does not
rotate `DiscoveryAuthKey` material, a captured frame from a node later
revoked would still verify. Sustained replay extends BC-2.03.001 EC-001
("heartbeat advertisement lost in transit... consoles may show stale data
for one heartbeat interval — acceptable per FM-005") and EC-002 ("tmux
session closes while advertisement is in flight") from a bounded
one-heartbeat staleness window into an attacker-controlled, unbounded
window. Impact is availability/integrity-of-presence, not confidentiality
— discovery carries no session content (BC-2.03.001 Invariant 3,
BC-2.03.003 PC-5) and no session-content or session-access compromise
follows — but it directly defeats a **stated postcondition**: BC-2.03.002
PC-5 promises "sessions no longer advertised... do not appear after the
next heartbeat cycle," and periodic replay of a stale frame can keep a
phantom session looking perpetually fresh, indefinitely, well past the
~30-60s staleness window the BC otherwise guarantees. This is a **new**
capability this story introduces — no comparable unauthenticated-broadcast
surface exists on develop today.

**My read on DI-003: the harm category (availability/integrity-of-presence,
not confidentiality) is the same category DI-003 discusses, but I do not
read DI-003's literal scope as pre-accepting this specific residual.**
DI-003 ("Router compromise degrades availability, not confidentiality")
is a statement about what a *compromised router* threat model can and
cannot achieve. Discovery replay requires no router compromise at all —
only passive capture of multicast traffic, a strictly lower attacker bar,
and one this story itself introduces by making the channel multicast.
DI-003's *posture* (this project already accepts availability-class
degradation as tolerable in some threat models, and does not gate
everything on preventing it) is a reasonable analogy in favor of accepting
the residual, and the consult correctly offered it as one legitimate
option — but the posture is not the same as DI-003's stated *scope*
literally covering this attacker, who needs no compromise. I weigh the
analogy as available-but-not-dispositive, and it does not on its own
justify shipping this new capability with an unbounded replay window
undocumented.

**Decision: add a monotonic sequence field now, not later.** This story
is establishing the discovery wire format for the first time — the design-time
cost of one new field is small (a few lines, one new small map on the
router), while retrofitting it after the format ships means a wire-format
version bump and back-compat handling across every implementation that
already speaks the old format. Given the threat model is unusually easy to
execute here specifically *because* Ruling 2 makes this channel multicast
(passive LAN capture, no compromise required — a materially lower bar than
most attacks this system otherwise defends against), and the harm directly
contradicts a stated BC postcondition rather than being a vague
"worse availability," I adopt the field rather than accept the residual.

**Concrete spec, for BC-2.03.001 Postcondition 2 amendment (new envelope
field, alongside `NodeAddr` — does not touch BC-2.03.003 PC-1's per-session
`{session_name, attached, quality}` contract):**

- **Field:** `Sequence uint32`, positioned immediately after `NodeAddr` in
  the wire body — `SVTNID[0:16] | NodeAddr[16:24] | Sequence[24:28] |
  uint16 count[28:30] | sessions...`. This does not disturb SEC-DW-01's
  fixed-offset key-selector extraction (`[0:16]`/`[16:24]`), since
  `Sequence` sits after both key-selector fields.
- **Sender-side:** each `Discovery` instance maintains an in-memory
  monotonic counter (same pattern as the existing `heartbeatCount
  atomic.Uint64`, `discovery.go:152`), incremented on every outbound
  advertisement — whether triggered by state-change (AC-001a) or heartbeat
  (AC-001b) — and embedded as `Sequence` in that frame.
- **Receiver (router) discard rule:** the router tracks, per
  `(SVTNID, NodeAddr)`, the last-accepted `Sequence` value (a small map,
  bounded by the number of admitted nodes — no unbounded growth, no TTL
  sweep needed, unlike the admission layer's nonce set). On a
  HMAC-verified frame: if `incoming.Sequence <=
  lastSeen[svtnID,nodeAddr]`, discard as a replay (do not relay to other
  nodes) even though HMAC passed; otherwise accept, relay, and update
  `lastSeen`.
- **Cold-start behavior:** on router restart, or the first-ever frame from
  a newly-admitted node, there is no prior `lastSeen` entry — the first
  frame received for a given `(SVTNID,NodeAddr)` is always accepted
  regardless of its `Sequence` value, bootstrapping the tracking state.
  This reopens a narrow, bounded replay window (an attacker could replay
  an old captured frame immediately after a router restart, before the
  real node's next heartbeat) — bounded to at most one heartbeat interval
  (≤30s) after restart, not indefinite. This is not a new class of
  accepted risk: it directly parallels this codebase's own precedent for
  bounded-not-perfect replay protection, `internal/admission`'s
  `nonceTTL = 60 * time.Second` (`admission.go:142`).
- **Wraparound:** `uint32` wraps at ~4.29 billion advertisements per node;
  at realistic rates (heartbeat every 30s plus occasional state-change
  bursts) this is not a practical concern within any reasonable node
  uptime. No wraparound-tolerant comparison logic is proposed — plain
  strictly-greater-than is sufficient for this story's scope.

**This choice is flagged prominently for the human gate at story-ready**,
per the orchestrator's instruction — it adds a small amount of new wire
format and router-held state that a human sign-off should see named
explicitly before the story is scheduled, not just inherit silently from
this ruling.

### Security Consult Addendum (v1.1) — SEC-DW-01 through SEC-DW-09

A security-reviewer consult was dispatched per the v1.0 "RECOMMENDED, not
blocking" note above. Verdict: **RULING-1-SOUND-WITH-CONSTRAINTS** — the
core reuse-and-domain-separate decision stands; nine findings, all
additive constraints, none overturning the ruling.

| # | Severity | CWE | Finding | Disposition | Where it landed |
|---|---|---|---|---|---|
| SEC-DW-01 | HIGH | CWE-770, CWE-400 | Full `decodeBody()` before HMAC verify is a pre-auth parsing/DoS surface on the new unauthenticated UDP ingest | ADOPTED as MANDATORY | Implementation Constraint 2 (rewritten) |
| SEC-DW-02 | MED | CWE-770, CWE-400 | UDP read buffer must be bounded/fixed, sized to realistic legitimate usage — not the UDP/IP theoretical maximum; `maxSessionsPerAdvertisement` needs re-derivation against real usage scale, not a network-MTU figure | ADOPTED | Implementation Constraint 5 (new) |
| SEC-DW-03 | MED | CWE-406, CWE-770 | Aggregate token-bucket backstop at the read loop; reuse `FailureCounter`/E-ADM-017 as visibility-only, never a per-`NodeAddr` gate | ADOPTED | Implementation Constraint 6 (new) |
| SEC-DW-04 | MED | CWE-400, CWE-770, CWE-779 | Discovery failure logging must be rate-limited/counter-based, not BC-2.05.008's per-packet TCP policy | ADOPTED | Implementation Constraint 7 (new) |
| SEC-DW-05 | LOW/INFO | CWE-208 | No wire-visible accept/reject differential (MUST); dummy-HMAC-on-miss stays optional hardening | ADOPTED (MUST clause) | Implementation Constraint 8 (new) |
| SEC-DW-06 | — | — | Purpose-bound HKDF label (`HKDFInfoDiscovery`) to cap blast radius of the new ingest surface | **ADOPTED** — no counter-rationale found; satisfies my own v1.0 rejection criteria for the group-secret alternative (cheap, doesn't reopen that design) while capping exposure of the new surface | Concrete derivation rule (rewritten); Implementation Constraints 1 and 4 (rewritten) |
| SEC-DW-07 | MED | CWE-294 | Replay: no freshness signal in the wire format | **My adjudication:** add a monotonic `Sequence` field now (design-time cheap vs. retrofit-expensive; DI-003 does not already cover this threat model) | New "Replay / freshness" subsection above; flagged for human gate |
| SEC-DW-08 | — | — | Senders should set multicast TTL=1 explicitly; scope-language clarification that 239/8 is hygiene, HMAC is the sole boundary, regardless of actual multicast-routing scope in a given deployment | ADOPTED | Ruling 2 (see below) |
| SEC-DW-09 | — | — | Relay rate cap per `(SVTNID,NodeAddr)`, independent of HMAC validity, against a misbehaving-but-admitted sender | ADOPTED | Folded into Ruling 2's Forward Obligation |

I found no counter-rationale for SEC-DW-06 and adopted it as instructed;
nothing here required stopping to report a disagreement.

---

## Ruling 2 — SVTN-scoped multicast address derivation: administratively-scoped IPv4, router-only listener, relay (not peer) delivery

**DECISION: (a) The wire mechanism is router-mediated relay, not raw
peer-to-peer IP multicast — this is not a new design choice, it is the
literal requirement of DI-004 and BC-2.03.001 Invariant 1, which the story
stub and `ARCH-03`'s sketch language failed to reconcile. Only the
router-mode daemon calls `net.ListenMulticastUDP`; access/console nodes
send advertisements addressed to the multicast group but never join it,
and never receive from it directly. (b) The multicast address is
IPv4-only for this story, deterministically derived from the SVTN ID into
the RFC 2365 administratively-scoped range `239.0.0.0/8`, on a fixed
well-known port. (c) This lands as amendments to `ARCH-03` §"Session
Discovery" and `BC-2.03.001` (Precondition 3, Postcondition 1) — not a new
ADR, because the underlying rule (router-mediated delivery) is already an
ADR-level decision (DI-004) with an existing home; what's missing is the
concrete derivation formula, which is architecture-detail, not a new
architectural principle.**

### The finding, restated plainly

`ARCH-03-routing-engine.md:299-308` (an explicitly-marked "architecture
sketch for PE," not yet reconciled) reads: "Access nodes send `PRESENCE_ADV`
frames to a well-known SVTN multicast address. Consoles subscribe to
multicast and maintain a local session list." Read literally, this is
peer-to-peer IP multicast — access nodes and consoles as co-members of the
same OS-level multicast group, receiving each other's datagrams directly.
That is precisely what **DI-004** forbids: "All traffic between nodes
passes through at least one router. A node has no mechanism to discover or
contact another node's network address directly."
(`invariants.md:66-72`). BC-2.03.001 Invariant 1 — the BC this very story
is chartered to fully implement — already resolves the tension in the
router's favor: "Advertisements flow node-to-router-to-node via the SVTN;
no direct node-to-node multicast." The story stub's scope item 3 ("replace
... with `net.ListenMulticastUDP` dispatch goroutine") is agnostic about
*who* calls `ListenMulticastUDP`, and was evidently drafted against the
`ARCH-03` sketch rather than against the BC's own invariant text. This
ruling makes the already-decided answer concrete and flags `ARCH-03` for
correction — it is not overriding DI-004 or the BC; it is enforcing them
against a stale architecture sketch.

### Why router-only multicast membership satisfies DI-004 without abandoning "multicast" as a mechanism

Standard IP multicast semantics do not require a *sender* to join a
multicast group — only *receivers* need group membership
(`IGMP`/`MLD` join). This gives a clean, standards-compliant design that is
simultaneously "real multicast" (satisfying the story's `net.
ListenMulticastUDP` instruction and `ARCH-03`'s "well-known SVTN multicast
address" framing) and DI-004-compliant:

- **Only the router-mode daemon calls `net.ListenMulticastUDP`** and joins
  the SVTN-scoped group on its LAN-facing interface(s). It is the sole
  subscriber.
- **Access nodes send** `PRESENCE_ADV` datagrams via plain `net.
  DialUDP`/`WriteTo` to the well-known multicast address — this requires
  no group membership and no knowledge of any other node's address,
  satisfying DI-004's "a node has no mechanism to ... contact another
  node's network address directly." The access node's only target
  knowledge is the deterministic, SVTN-derived group address — structurally
  identical to how it already knows the router's `cfg.ListenAddr` for the
  TCP data plane, just multicast-addressed instead of unicast-addressed.
  **Senders set the outbound multicast TTL to 1 explicitly** (SEC-DW-08,
  adopted): network-layer containment (the datagram never survives a
  router-hardware hop beyond the local link) as defense-in-depth alongside
  the application-layer control (only the switchboard router process
  joins the group) — belt-and-suspenders against a misconfigured LAN
  switch/router that might otherwise forward multicast traffic further
  than intended.
- **The router authenticates** each inbound datagram via Ruling 1's
  `DiscoveryAuthKeyFor` lookup (HMAC-first, fail-closed — DI-006's "first
  router" gate) and, on success, **relays** the advertisement onward over
  each admitted node's own already-authenticated connection to the router
  — the same trust boundary every other SVTN frame type already crosses.
  Consoles never join the multicast group; "consoles subscribe to
  multicast" in `ARCH-03`'s current text should be corrected to "consoles
  receive relayed advertisements via the router" (see Downstream Touch-List).
- **A multicast-address collision between two different SVTNs is
  harmless**, not merely unlikely: even if two SVTNs' derived addresses
  collided, a router serving SVTN-A would receive a stray SVTN-B datagram,
  attempt the `DiscoveryAuthKeyFor(payload.SVTNID, payload.NodeAddr)` lookup
  against SVTN-A's forwarding/admitted state, and fail — the datagram is
  dropped fail-closed exactly as an unauthenticated frame would be. HMAC
  authentication, not multicast-address uniqueness, is the actual security
  boundary (consistent with DI-005's framing: cross-SVTN isolation
  "requires possession of keys," not merely non-overlapping addressing).
  This means address collision only needs to be *rare enough for routing
  efficiency*, not cryptographically improbable. **More generally
  (SEC-DW-08, adopted):** the `239.0.0.0/8` administratively-scoped range
  is a hygiene/naming choice, not a security control — HMAC authentication
  remains the sole security boundary *regardless of the actual
  multicast-routing scope realized in a given deployment*. A deployment
  whose network fabric does not honor administrative scoping as expected
  (misconfigured switches, unusual VLAN topology, etc.) degrades routing
  efficiency and collision exposure, never authentication.

### Concrete address-derivation scheme

- **Range:** IPv4 `239.0.0.0/8` — RFC 2365 "administratively scoped"
  block, explicitly intended for private, non-globally-routed multicast
  applications exactly like this one (as opposed to `224.0.0.0/24`,
  reserved for link-local infrastructure protocols, or the globally-routed
  `233.0.0.0/8` GLOP space, neither of which fit). 16,777,216 distinct
  group addresses.
- **Derivation:** `addr = 239.h[0].h[1].h[2]` where `h = SHA-256(svtnID)`
  (first three bytes of the digest). Deterministic: same SVTN ID always
  produces the same address, computable independently by every admitted
  node and the router with no coordination step (mirroring Ruling 1's
  "no distribution needed" property). A raw truncation of `svtnID` itself
  was considered and rejected in favor of a hash step purely for domain
  separation hygiene (avoiding any accidental structural correlation
  between the SVTN ID's bit layout and the derived address) — `svtnID`
  already transits in cleartext in every outer header
  (`ARCH-02-protocol-stack.md:78`), so this is not a secrecy requirement,
  just cheap good practice.
- **Port:** a single fixed, named constant in `internal/discovery`
  (parallel to the existing `HeartbeatInterval` constant,
  `discovery.go:32-34`) in the IANA dynamic/private range (49152–65535) to
  avoid registered-port collision — **the exact number is a bikeshed-level
  choice, not gated on this ruling; recommend `49201`** (arbitrary,
  unregistered) as a placeholder for architect/PO sign-off. One port
  suffices for all SVTNs because the group *address* — not the port —
  provides SVTN scoping.
- **Static, not dynamic, allocation.** The address is a pure function of
  `svtnID`; there is no allocation state to track, collide against, or
  release on `admin.svtn.destroy`. This is simpler than dynamic allocation
  and has no failure mode requiring cleanup — consistent with SOUL.md §7
  (gradual elaboration: build the simplest thing that works).
- **IPv6: explicitly out of scope for this story.** No IPv6 data-plane
  precedent exists anywhere in this codebase (Verified Premises); the only
  IPv6 references are mgmt-plane loopback authorization. Scoping IPv6
  in now would require inventing both an IPv6 data-plane story and an
  IPv6-specific administratively-scoped derivation (RFC 3306) with zero
  existing precedent to ground it against — disproportionate to this
  story. Flag as a named forward obligation, not a silent gap.

### Loopback testability for VP-044 / VP-045

`net.ListenMulticastUDP` works on loopback on both macOS and Linux, but —
per this project's own **B13** lesson (platform-specific behavior requires
platform-specific testing; `tmux-cmc` needed five fixes across five
platform-behavior surprises) — the loopback interface name differs
(`lo0` on macOS vs `lo` on Linux) and must be resolved via
`net.InterfaceByName` rather than assumed. `internal/testenv.NewLoopback`
(`testenv.go:378-387`) is a **VP-042-scoped compile-shim** for
keystroke-echo benchmarking and is not a fit for this — do not extend it.
Recommend a new, purpose-built helper (e.g.
`testenv.MulticastLoopbackInterface(t testing.TB) *net.Interface`,
resolving the platform-appropriate loopback interface name once, with a
clear skip/fail message if the CI runner's loopback doesn't support
multicast) to unblock VP-044/VP-045 integration tests without coupling to
the unrelated VP-042 fixture. This is new test infrastructure, not present
today, and should be counted in the story's implementation scope.

### `ARCH-03` / `BC-2.03.001` amendment content

**`ARCH-03` §"Session Discovery" — EXECUTED this session (Task 2), not
merely proposed.** `ARCH-03-routing-engine.md` v1.6→v1.7; the section is
now live with the router-relay design, `DiscoveryAuthKey` naming
(corrected from an earlier `frame_auth_key` draft label), TTL=1, the
routing-scope-independence scope note, and a "Superseded language"
callout naming exactly what the old sketch said. `ARCH-INDEX.md`
v1.10→v1.11 synced with a matching changelog row. See that file directly
for the final text; not reproduced here to avoid a second copy drifting
from the source of truth.

**`BC-2.03.001` Precondition 3 amendment (still PENDING — product-owner)**
(currently: "A SVTN-scoped multicast address is allocated for the SVTN's
discovery channel"):

> **Derivation rule (Ruling S-BL.DISCOVERY-WIRE-2):** the multicast address
> is `239.h0.h1.h2` where `h0..h2` are the first three bytes of
> SHA-256(svtnID) — deterministic, static, requiring no allocation
> bookkeeping. Only the router-mode daemon joins this multicast group.

**`BC-2.03.001` Postcondition 1 amendment (still PENDING — product-owner)**
(currently: "The advertisement is multicast to all admitted nodes on the
SVTN"):

> **Delivery mechanism note (Ruling S-BL.DISCOVERY-WIRE-2):** "multicast"
> here denotes SVTN-wide fan-out semantics, not direct peer-to-peer IP
> multicast. Delivery is router-mediated: the access node sends one UDP
> datagram to the SVTN-scoped multicast address (received only by the
> router); the router authenticates it and relays it to each admitted node
> over that node's own connection. This satisfies DI-004 (no direct
> node-to-node communication) and DI-006 (HMAC verified at first router).
> The `239.0.0.0/8` range is addressing hygiene, not a security boundary —
> HMAC authentication remains the sole security boundary regardless of the
> actual multicast-routing scope realized in a given deployment
> (SEC-DW-08).

**`BC-2.03.001` Postcondition 2 amendment — new field (still PENDING —
product-owner)** (currently: "Each advertisement includes: access node
address, list of session names, per-session attachment status, per-session
quality indicator"):

> **Replay-resistance field (SEC-DW-07, Ruling S-BL.DISCOVERY-WIRE-1):**
> each advertisement additionally includes a monotonic `sequence` value,
> unique-and-increasing per (access node, SVTN), incremented on every
> outbound advertisement (state-change or heartbeat-triggered). The router
> discards any HMAC-verified advertisement whose `sequence` is not strictly
> greater than the last-accepted value for that (SVTN, node) pair, even
> though HMAC passed — closing the replay window that would otherwise let
> a captured, still-valid frame be re-injected indefinitely and defeat
> Postcondition 5's staleness-expiry guarantee. Cold-start (router
> restart, or first frame from a newly-admitted node) accepts
> unconditionally, bounding the residual replay window to at most one
> heartbeat interval — the same bounded-not-perfect posture this project
> already accepts for admission-layer nonce replay (`nonceTTL=60s`).

### Forward Obligation: hop-2 relay transport — RESOLVED by Ruling 3 (v1.3)

**Superseded status.** This subsection originally deferred the hop-2 wire
mechanism to decomposition. Ruling 3 below (v1.3) adjudicates it directly —
story-writer decomposition was blocked on it (spec-adversarial pass 1 would
flag an undecided delivery transport). The SEC-DW-09 rate-cap constraint
this subsection introduced is restated and concretized inside Ruling 3; it
is not duplicated here. Kept as a historical pointer only — see Ruling 3.

---

## Ruling 3 — Hop-2 relay transport: `FrameTypeCtl` `control_type` discriminator, connection-trust boundary, SVTN-scoped exclude-originator fan-out, ~1/sec rate cap

**DECISION:**

**(a) Transport — ride the existing `FrameTypeCtl` (0x03) `control_type`
discriminator, NOT a new outer `FrameType` byte.** `internal/frame.go`'s
canonical `FrameType` enum is `{0x01..0x06}` — fully populated;
`FrameTypePEConnect = 0x06` (S-BL.PE-RECEIVE-LOOP) took the last slot, and
`FrameType.Valid()` hard-rejects anything above it. Minting a 7th outer
frame type would require editing `frame.go`'s `Valid()` bound, amending
ARCH-02 §"Outer Header Format" (the canonical registry), and touching every
`ParseOuterHeader` call site's implicit assumptions — a materially larger,
riskier change than this story's scope warrants, and not what the shipped
precedent actually did. The real precedent (S-7.04-FU-DRAIN-WIRE,
`cmd/switchboard/mgmt_wire.go`, develop `f73676d`) rides `FrameTypeCtl`
with a `control_type` discriminator byte in the payload
(`payload[0]`) — `control_type=0x01` for DRAIN. `BC-2.01.008` (the
authoritative `control_type` schema home) already reserves `0x02` for
RESYNC and states in Invariant 3: "Schema growth is append-only: New
`control_type` opcodes are assigned sequentially (0x03, 0x04, …)." Ruling
3 allocates **`control_type = 0x03` (`DISCOVERY_RELAY`)** — the next free
value, per that rule.

**(b) Connection-trust boundary — zero `HMACTag`, matching the DRAIN
precedent exactly; no dilution of SEC-DW-08's "HMAC is the sole security
boundary" framing.** The DRAIN broadcast (`mgmt_wire.go`, drain observer
closure) constructs its outer header via `frame.EncodeOuterHeader` and
never sets `HMACTag` — it ships as the zero value. Authentication for that
hop is the already-completed Tier-1-admitted, already-open TCP connection
itself, not a fresh per-frame HMAC (the router does not sign outbound
control traffic to nodes anywhere in the shipped codebase). Hop-2 discovery
relay follows the identical pattern: the relay frame's `HMACTag` is zero;
the connection is the trust boundary. This does **not** dilute SEC-DW-08 —
that framing governs hop 1 (the UDP multicast ingest point, reachable by
anyone on the LAN segment, where HMAC-over-`DiscoveryAuthKey` is
load-bearing because there is no other authentication available). Hop 2
travels over a connection that was already mutually authenticated at
admission time, before any discovery traffic exists — a structurally
different trust context, the same one every other router-to-node control
frame in this codebase already relies on. Stating both boundaries
explicitly (hop 1 = HMAC; hop 2 = connection identity) is what keeps
SEC-DW-08 undiluted, not weakens it.

**(c) Payload shape — re-serialized, not a raw relay of hop-1's UDP
bytes.** Hop-1's UDP datagram HMAC is scoped to the wire path and key that
produced it (`DiscoveryAuthKey`, verified exclusively by the router per
Ruling 1) and has no meaning to a receiving node — forwarding it verbatim
would misleadingly imply the receiving node could or should re-verify it,
which BC-2.03.001 PC-5 (v1.5, already landed) explicitly forecloses:
"access and console nodes never independently look up or re-verify another
node's `DiscoveryAuthKey`; they receive already-authenticated
advertisements via the router's relay over their own admitted connection."
That sentence already anticipates and describes exactly the trust model
this ruling formalizes — strong textual support that BC-2.03.001 needs no
further amendment for hop-2 (see (e) below). Concrete layout, respecting
`BC-2.01.008` Postcondition 3's fixed 4-byte control-message header and
Invariant 5/DI-007's explicit allowance to extend beyond byte 3:

```
byte[0]    control_type = 0x03 (DISCOVERY_RELAY)
byte[1]    version = 0x01
byte[2:4]  reserved = 0x0000
byte[4:12] NodeAddr    — the ORIGINATING access node's 8-byte address
byte[12:16] Sequence   — uint32 BE, the same value hop-1 accepted (SEC-DW-07)
byte[16:18] session count — uint16 BE
byte[18:]  sessions...  — same per-session encoding internal/discovery's
                           existing encodeBody already produces
```

`SVTNID` is deliberately NOT repeated inside this payload — the relay
frame's own `OuterHeader.SVTNID` field (present on every frame type, per
the 44-byte layout `frame.go` already defines) carries the SVTN scope,
consistent with how every other frame type on this wire already scopes
itself. Reusing `internal/discovery`'s existing per-session serialization
for `byte[18:]` avoids inventing a second session-list wire format.

**(d) Fan-out semantics — SVTN-scoped, exclude-originator, best-effort
non-blocking (DRAIN's `sendMap.Range` pattern), NOT
`routing.SplitHorizon.Forward`/`FrameArrivalHandler.OnFrameArrival`.**
I evaluated the latter — a real, already-shipped alternative (BC-2.02.008
split-horizon, BC-2.02.009 drop-cache, `internal/routing/split_horizon.go`
+ `on_frame_arrival.go`) — and reject it for two concrete reasons, not
convenience:
  1. Its drop-cache half exists to suppress inter-router relay loops in a
     multi-router mesh. `BC-2.01.008` Invariant 2 confirms "no inter-router
     relay path is implemented in this codebase" — Ruling 2's design has
     exactly one router terminating a given SVTN's multicast segment (a
     star topology, node leaves only). There is no loop to suppress, and
     hop-1's own SEC-DW-07 sequence-gate already guarantees a frame
     reaching the relay step is fresh (not a replay) — layering
     crc32-checksum drop-cache dedup on top is redundant machinery
     misapplied to a topology it wasn't built for.
  2. `SplitHorizon.Forward`'s exclusion parameter is `arrivalIface
     InterfaceID` — the interface the frame arrived ON. Hop-1's
     advertisement arrives via the UDP multicast socket, not via any
     `netingress`-accepted TCP interface at all; there is no real
     `arrivalIface` value to supply without inventing a fictitious
     sentinel. What hop-2 actually needs to exclude is "the originating
     access node's own admitted TCP connection" (so a node doesn't receive
     an echo of its own advertisement) — a `NodeAddr`-keyed exclusion, not
     an `InterfaceID`-keyed one in the split-horizon package's literal
     sense.

  Fan-out therefore follows DRAIN's own dispatch shape directly: a
  purpose-built closure (same placement as the DRAIN observer, inline in
  `runRouter`) iterates the live connections of nodes admitted to the
  advertisement's SVTN, skips the originating `NodeAddr`, and for each
  remaining target does the same `select { case nc.send <- relayFrame:
  default: }` best-effort non-blocking send DRAIN already uses — full
  backpressure alignment with the shipped precedent (a slow/stuck node
  drops the relay silently rather than blocking the router; `nc.send` is
  never closed, so the send itself cannot panic). No queueing, no retry,
  no wire ACK — matching Q3.P1's "best-effort delivery BINDING" ruling
  already established for DRAIN (`S-7.04-FU-DRAIN-WIRE-placement-note.md`).

**(e) SEC-DW-09 rate cap — concretized at ~1/sec per `(SVTNID, NodeAddr)`,
enforced at the relay-dispatch decision point, silent-drop-first plus a
non-gating visibility counter.** This is the *fan-out amplification* cap,
distinct from Ruling 1 Implementation Constraint 6's *ingest* token-bucket
(SEC-DW-03) — a different enforcement point serving a different purpose.
Without it, one misbehaving-but-legitimately-admitted sender (HMAC-valid,
sequence-increasing — passes every hop-1 check) could force the router
into O(N) relay writes per received frame at unbounded rate, N = admitted
node count on that SVTN. Enforcement: keyed by the *originating*
`(SVTNID, NodeAddr)` — an advertisement arriving faster than ~1/sec from
the same sender still updates the router's own local registry/discard-map
state (SEC-DW-07 correctness is unaffected) but is NOT relayed on that
excess arrival. Discard behavior follows the SEC-DW-03 philosophy exactly
(Implementation Constraint 6): **silent drop is the actual backstop**;
an optional counter (reusing the `FailureCounter`-style shape, keyed the
same way) is visibility-only, never a gate, never promoted to a rate
decision itself. The ~1/sec figure gives generous headroom above
legitimate traffic (state-change bursts plus the 30s-default heartbeat)
while bounding amplification; decomposition may tune the exact value.

**(f) Fan-out TARGET RESOLUTION is a genuine, verified Forward
Obligation — not resolved here.** Determining "which of the router's live
connections currently belong to nodes admitted to SVTN X" requires binding
node identity (`NodeAddr`) to a live connection's `InterfaceID`/`nodeConn`.
This binding **does not exist in production code today.** Verified
directly, not assumed:
  - `routing.ForwardingEntry` (routing.go:130-139) carries `NodeAddr`,
    `SVTNID`, `FrameAuthKey` — no `InterfaceID` field.
  - `SVTNRoute` (routing.go:325-343) looks up `ForwardingEntry` but never
    dispatches bytes anywhere — `_ = payload`, `_ = entry // available for
    future use`. Ordinary DATA-plane relay delivery is itself still a
    validation-only stub; hop-2 discovery relay inherits this gap rather
    than introducing a new one.
  - `cmd/switchboard/mgmt_wire.go`'s `sendMap` is keyed purely by
    `routing.InterfaceID`, assigned in accept order (`IfaceIDSeed`) — no
    `NodeAddr` is recorded anywhere in the `onAccept` closure.
  - `admission.AdmitNode` (the Tier-1 signed-challenge handshake that
    would reveal a connecting node's identity) has **zero production call
    sites** — grepped directly: the only caller anywhere in `cmd/` or
    `internal/` is `internal/testenv/testenv.go:942` (test harness). It is
    not wired into `runRouter`'s connection-accept path at all yet.
  - This is the same gap this project has already named once, generically:
    `FO-DRAIN-WIRE-002` (`S-7.04-FU-DRAIN-WIRE.md`): "The drain observer
    assembles DRAIN frames using a per-node `Envelope` with zero
    SrcAddr/DstAddr/FrameAuthKey... The full bootstrap with Ed25519 key
    material is a session-bootstrap follow-on." No story named
    `S-BL.SESSION-BOOTSTRAP` (or similar) exists yet in `.factory/stories/`
    — it is referenced only generically, not yet scheduled.

  I am not proposing a workaround that assumes this binding exists — it
  doesn't, and inventing one would misrepresent the codebase's actual
  state. **Recommendation for story-writer:** either (i) add an explicit
  `depends-on`/sequencing edge from `S-BL.DISCOVERY-WIRE`'s hop-2 task to
  whatever story eventually delivers node-identity-to-connection binding
  (blocks the fan-out *target resolution* only — frame format, payload
  shape, rate cap, and the exclude-originator *principle* are all fully
  specified above and unblocked), or (ii) scope a narrow,
  story-local `Router.BindInterface(svtnID, nodeAddr, ifaceID)`-shaped seam
  (small, analogous in size to `RegisterForwardingEntry`) if blocking on an
  unscheduled story is unacceptable — but note this narrow seam still
  requires SOME connection-time identity signal to call it with, which
  circles back to the same unimplemented admission-handshake-on-connect
  gap; it does not eliminate the dependency, only relocates where it must
  be resolved. I recommend (i) as the honest default unless PO/story-writer
  has visibility into a scheduled session-bootstrap story I don't have.

**(g) `BC-2.01.008` needs a registry-row amendment — flagged, not
executed.** `control_type = 0x03` must be formally allocated in
`BC-2.01.008` Postcondition 2's table (currently: DRAIN=0x01, RESYNC=0x02
reserved, 0x03–0xFF unassigned) per that BC's own Invariant 3
(append-only, sequential assignment). This is a **different BC** from the
one this task asked me to assess (BC-2.03.001) — I was not authorized to
touch BC files this pass (unlike ARCH-03, which was explicitly
reassigned to me), so I flag this as a new touch-list item for
product-owner rather than executing it. See touch-list below.

**(h) Does `BC-2.03.001` need further amendment? NO — confirmed, not just
assumed.** BC-2.03.001 v1.5 PC-1's delivery-mechanism note and PC-5's key
derivation note already describe the relay/connection-trust model in
BC-appropriate generality (see (c) above — PC-5's own sentence already
states the model this ruling formalizes). Frame-type, `control_type`
allocation, payload byte layout, and rate-cap enforcement point are
architecture/story-decomposition-level detail, not BC-level — consistent
with how Ruling 1/2's HMAC and address-derivation mechanics landed in
`ARCH-03`, not further BC-2.03.001 edits, beyond what v1.5 already
captured.

**(i) Does `ARCH-03` v1.7 need a relay-transport sentence? YES — applied
directly as v1.8 below (executed, not proposed), per the same
architect-owns-ARCH-03 pattern this session already established for
Ruling 2.**

---

## Estimated Points: 8 (top of the stub's 5–8 range; unchanged by v1.1 and v1.3)

Ruling 3 fully specifies hop-2's wire mechanics (frame type, `control_type`
allocation, payload layout, connection-trust boundary, fan-out principle,
rate-cap value and discard philosophy) — removing ambiguity that could
have caused implementation rework, and each piece is a bounded, narrow
addition comparable in size to what SEC-DW-01..09 already added (a new
`control_type` opcode, one relay-dispatch closure modeled directly on
DRAIN's, one rate-cap map). The one item Ruling 3 does NOT resolve — fan-out
target resolution, blocked on node-identity-to-connection binding — is
named as an explicit Forward Obligation with a recommended resolution path
((f) above), not an open-ended unknown; it does not by itself justify
raising the estimate, since the *decision* is made (sequence or scope a
narrow seam), only the *scheduling* is deferred to story-writer/PO. Holds
at 8.

The v1.1 security-hardening constraints (SEC-DW-01..09) were weighed
against this estimate and do not move it: each is a narrow, bounded
addition (a handful of constants, one small `lastSeen` map, one new HKDF
label/function, one rate-limit knob) layered onto scope this ruling's v1.0
already counted, not a new architectural hop. If decomposition finds the
combined hardening work larger than expected, that is a signal to revisit
the estimate then, not a reason to inflate it speculatively here.

Rationale: the stub's own uncertainty ("5–8 TBD pending admitted-node key
vocabulary complexity") resolved toward the *simple* end for Ruling 1 (pure
reuse, no new KDF, no new distribution step) but Ruling 2's finding adds
genuinely new scope the stub did not anticipate: a real router-mode-only
UDP multicast listener lifecycle (bind/join/teardown, wired into `runRouter`
alongside the shipped `wireMetricsHandlers`/`wireRouterControlHandlers`
precedent), a new relay/fan-out path from router to every admitted node
(hop 2, Forward Obligation above), a new `routing.Router` lookup surface
(Ruling 1, small), and new multicast-capable loopback test infrastructure
for VP-044/VP-045 (none exists today — `testenv.NewLoopback` doesn't cover
this). This is comparable in shape to `S-BL.CLI-SURFACE-COMPLETION`'s
Ruling 4 (`router.reload`/`router.drain`, the single largest piece of that
story) but with an added new-protocol-surface dimension (UDP, not
RPC-over-existing-mgmt-transport) that Ruling 4 didn't have. Recommend 8,
not 5.

---

## Downstream Artifact Touch-List (for PO / story-writer / spec-steward — none executed by this ruling)

| Artifact | Change | Owner |
|---|---|---|
| `.factory/stories/S-BL.DISCOVERY-WIRE.md` | Resolve both Open Design Obligations (point to this ruling v1.3); set `estimated_points: 8`; correct §2's mislabeled "PC-1" citation to "Precondition 3"; add `changed_by_rulings: [..., S-BL.DISCOVERY-WIRE-rulings]`; decompose scope items 3/4 into router-listener, relay-hop, and sender-side tasks per this ruling (incl. bounded-buffer, rate-cap, and replay-counter tasks from the Security Consult Addendum); decompose Ruling 3's hop-2 relay design into concrete tasks (`control_type=0x03` dispatch, relay-frame assembly, `~1/sec` rate-cap map); add Ruling 3(f)'s fan-out-target-resolution Forward Obligation as an explicit dependency/task per the recommended (i)/(ii) choice; flag the SEC-DW-07 monotonic-`Sequence`-field decision at the human gate per the orchestrator's instruction | product-owner / story-writer |
| `BC-2.03.001` | **DONE** — v1.5 on disk, PO-executed (all four v1.2 blockquotes applied: PC-1 relay-delivery note, PC-2 `Sequence` field + SEC-DW-07 discard rule citing VP-080, PC-5 `DiscoveryAuthKey` derivation). Ruling 3(h): confirmed NO further BC-2.03.001 amendment needed for hop-2 — PC-5's existing "already-authenticated... via the router's relay" language already covers the connection-trust model Ruling 3 formalizes at the architecture level. | product-owner (done) |
| **`BC-2.01.008`** (new item, Ruling 3(g)) | Add a table row to Postcondition 2's `control_type` registry: `DISCOVERY_RELAY \| 0x03 \| S-BL.DISCOVERY-WIRE \| Router relays a validated, sequence-fresh advertisement to admitted SVTN peers`. Per that BC's own Invariant 3 (append-only, sequential assignment) — `0x03` is the next free value after DRAIN=0x01/RESYNC=0x02(reserved). Not executed by this ruling — a different BC than the one this task authorized me to assess. | product-owner |
| `.factory/specs/architecture/ARCH-03-routing-engine.md` | v1.7→**v1.8, executed by this session (Ruling 3)**: hop-2 relay-transport paragraph added to §"Session Discovery" — see ARCH-03 changelog | architect (done) |
| `.factory/specs/architecture/ARCH-INDEX.md` | Version/changelog sync for ARCH-03 v1.8 — **checked this pass; see below** | architect (done/checked) |
| `internal/routing/advertisement_hmac.go` | Add `DiscoveryAuthKeyFor` and `DeriveDiscoveryKey` per Ruling 1 Implementation Constraints 1 and 4 (v1.1 names — renamed from the v1.0 draft's `FrameAuthKeyFor`/`DeriveFrameAuthKey` per SEC-DW-06) | implementer |
| `internal/hmac/hmac.go` | Add `HKDFInfoDiscovery` constant + `DeriveDiscoveryKey` function (SEC-DW-06); `DeriveKey`/`HKDFInfo`/`RegisterKey` untouched | implementer |
| `internal/discovery/discovery.go` | Delete `advertisementKey`; update `Encode`/`Decode`/sender-side call sites to the domain-separated key; add the new `Sequence uint32` field to `AdvertisementPayload` + `encodeBody`/`decodeBody` (SEC-DW-07); add router-side ingest path (new file, e.g. `discovery_wire.go`) with fixed-offset key-selector extraction (SEC-DW-01), bounded read buffer (SEC-DW-02), rate limiting (SEC-DW-03), rate-limited logging (SEC-DW-04), and the `lastSeen` replay-discard map (SEC-DW-07); returns an accept/relay decision (incl. the SEC-DW-09 rate-cap verdict, Ruling 3(e)) to its caller rather than performing I/O itself; `ReceiveAdvertisement` shape unchanged (defense-in-depth reframe, doc comment update only) | implementer |
| `cmd/switchboard/mgmt_wire.go` (`runRouter`) | New Phase wiring: (1) the discovery multicast listener calling into `internal/discovery`'s router-side ingest; (2) on an accept+relay verdict, a hop-2 relay-dispatch closure modeled directly on the DRAIN observer (`control_type=0x03` `DISCOVERY_RELAY` frame, zero `HMACTag`, `sendMap`-based best-effort non-blocking send, SVTN-scoped exclude-originator fan-out per Ruling 3(d)) — mirrors the `wireMetricsHandlers`/`wireRouterControlHandlers`/DRAIN-observer register-before-serve precedents. Fan-out TARGET RESOLUTION depends on Ruling 3(f)'s Forward Obligation (node-identity-to-connection binding) — see story row above. | implementer |
| `internal/testenv/testenv.go` | New multicast-loopback test helper (not an extension of `NewLoopback`) | test-writer |
| `VP-044.md`, `VP-045.md` | Update from `partial` (RULING-W6TB-D doctrine) toward full coverage once the wire integration tests land; supersede the "Blocker: multicast wire transport implementation" notes | formal-verifier |
| `VP-080.md` (minted this session, `status: draft`) | Replay-rejection verification property covering SEC-DW-07's `Sequence`/`lastSeen` discard rule, anchored to BC-2.03.001. Already cited in BC-2.03.001 v1.5 (PO-executed). `VP-INDEX.md` v2.41 already carries the index row/counts. Transition `draft → active` belongs to story-writer once this VP is scoped into a wave. | product-owner (done) / story-writer (scope) |
| New VP recommended (not filed by this ruling) — hop-2 relay fan-out | A verification property covering Ruling 3(d)/(e): SVTN-scoped fan-out excludes the originator; the SEC-DW-09 rate cap suppresses relay (not registry-update) on excess-rate arrivals from the same `(SVTNID,NodeAddr)`. Left unminted deliberately — depends on Ruling 3(f)'s fan-out-target-resolution mechanism landing first, unlike VP-080 which had no such dependency. | architect / formal-verifier, once (f) resolves |
| No new `E-*` taxonomy codes are anticipated | `DiscoveryAuthKeyFor` lookup-miss reuses the existing `ErrInvalidHMACTag`/`ErrSVTNMismatch` sentinels (E-ADM-family already covers HMAC failures via `routing`'s existing codes); the hop-2 `control_type=0x03` unrecognized-opcode path reuses BC-2.01.008 PC-4's existing silent-ignore rule — no new code needed there either | — |
| No `interface-definitions.md` / CLI-surface changes | This story is wire/transport only; no new `sbctl` verb is implied by any of the three rulings | — |

---

## Summary Table

| # | Obligation | Decision | Key mechanism | BC action |
|---|---|---|---|---|
| 1 | Admitted-node HMAC key derivation | Reuse the shipped `hmac.DeriveKey` shape with a domain-separated info label (`DiscoveryAuthKey`, ADR-001 + SEC-DW-06); verified router-only, fixed-offset key-selector extraction (SEC-DW-01), replay-checked (SEC-DW-07) | New `Router.DiscoveryAuthKeyFor`/`DeriveDiscoveryKey` thin wrappers | BC-2.03.001 PC-2 (new `Sequence` field, cites VP-080) + PC-5 — **done, v1.5** |
| 2 | SVTN-scoped multicast address derivation | Router-relay model (DI-004-mandated, not new); IPv4 `239.0.0.0/8`, `SHA-256(svtnID)`-derived, static, fixed port, sender TTL=1 (SEC-DW-08) | Router-only `net.ListenMulticastUDP`; sender-only `WriteTo`, no group join | BC-2.03.001 Precondition 3 + PC-1 — **done, v1.5**; ARCH-03 §Session Discovery — **done, v1.7** |
| 3 | Hop-2 relay transport (decomposition-blocking Forward Obligation from Ruling 2) | `FrameTypeCtl` + new `control_type=0x03` (`DISCOVERY_RELAY`); zero `HMACTag` (connection-trust boundary, DRAIN precedent); SVTN-scoped exclude-originator best-effort fan-out (NOT `SplitHorizon`/`OnFrameArrival` — evaluated, rejected); `~1/sec` per-`(SVTNID,NodeAddr)` rate cap, silent-drop-first (SEC-DW-09) | New relay-dispatch closure in `runRouter`, modeled on the DRAIN observer; fan-out **target resolution** left as a verified Forward Obligation (node-identity-to-connection binding does not exist in production code today) | **No BC-2.03.001 change needed — confirmed.** `BC-2.01.008` needs one new registry row (`control_type=0x03`) — flagged, not executed. ARCH-03 §Session Discovery — **done, v1.8** |

**Nothing in this ruling descopes the story.** Ruling 2's DI-004 finding
adds scope the stub did not anticipate (a real relay hop) rather than
removing any — hence the points recommendation moving to the top of the
stub's stated range, not below it. Ruling 3 fully specifies that hop's
wire mechanics and does not add scope beyond what Ruling 2 already
counted; its one open item (fan-out target resolution) is a named,
recommended-resolution Forward Obligation, not an unscoped unknown.

---

## Decision Log

| Date | Actor | Entry |
|------|-------|-------|
| 2026-07-13 (v1.3) | architect | **Ruling 3 — hop-2 relay transport, adjudicated per orchestrator request** (story-writer decomposition was blocked on it). Transport: `FrameTypeCtl` + new `control_type=0x03` (`DISCOVERY_RELAY`), NOT a new outer `FrameType` — the 6-slot outer enum is exhausted (`FrameTypePEConnect=0x06` filled the last slot); the real DRAIN-over-SVTN precedent (`f73676d`) rides the `control_type` discriminator, not a new outer type, correcting the framing in the orchestrator's own request. Connection-trust boundary: zero `HMACTag` on the relay frame, matching DRAIN exactly — SEC-DW-08 stays undiluted (hop 1 = HMAC boundary; hop 2 = already-admitted-connection boundary, stated explicitly as two distinct boundaries, not blurred into one). Payload: re-serialized (NodeAddr + Sequence + session list after the 4-byte control header), not a raw relay of hop-1's UDP bytes — BC-2.03.001 PC-5 (v1.5, already landed) already anticipates this trust model in prose, confirming (h) below. Fan-out: evaluated and REJECTED `routing.SplitHorizon.Forward`/`FrameArrivalHandler.OnFrameArrival` as a real alternative (not a strawman) — its drop-cache half targets inter-router loop prevention in a mesh topology that doesn't exist here (BC-2.01.008 Inv-2: no inter-router relay path implemented), and its `arrivalIface` exclusion parameter has no natural value for a UDP-sourced frame; adopted a purpose-built closure modeled directly on DRAIN's `sendMap.Range` best-effort non-blocking send instead, filtered to SVTN-admitted nodes excluding the originating `NodeAddr`. Rate cap: concretized SEC-DW-09 at `~1/sec` per `(SVTNID,NodeAddr)`, enforced at the relay-dispatch point (distinct from Ruling 1's ingest-side SEC-DW-03 token bucket), silent-drop-first plus a non-gating visibility counter — same philosophy as SEC-DW-03, not a new doctrine. **(f) Verified, not invented, Forward Obligation:** fan-out target resolution (which live connections belong to SVTN-admitted nodes) requires node-identity-to-connection binding that does NOT exist in production code today — confirmed by direct grep, not assumption: `admission.AdmitNode` has zero production call sites (only `internal/testenv/testenv.go:942`); `routing.ForwardingEntry` carries no `InterfaceID`; `SVTNRoute` never dispatches bytes (`_ = payload`, `_ = entry`); `sendMap` carries no `NodeAddr`. Same gap this project already named once as `FO-DRAIN-WIRE-002` ("session-bootstrap follow-on," no story yet scheduled). Recommended story-writer add an explicit sequencing dependency rather than inventing a workaround that assumes infrastructure that doesn't exist. **(g)** `BC-2.01.008` needs one new `control_type` registry row (`0x03`) — flagged as a new touch-list item for product-owner, not executed (different BC than this task authorized me to touch). **(h)** Confirmed, not assumed: `BC-2.03.001` needs NO further amendment for hop-2 — its v1.5 PC-5 text already states the relay/connection-trust model in BC-appropriate generality. **(i)** `ARCH-03` v1.7→v1.8 executed directly (architect-owns-ARCH-03 precedent from Ruling 2 continues) — hop-2 relay-transport paragraph added to §Session Discovery. Points estimate holds at 8 — the one open item (f) is a named decision-with-a-recommended-path, not an unscoped unknown. Superseded the old "Forward Obligation: hop-2 relay transport left to decomposition" subsection in place with a pointer to this ruling. Touch-list, Summary Table (new row 3), and `bc_traces` frontmatter (added `BC-2.01.008`) updated accordingly. `BC-2.03.001` touch-list row corrected to DONE (v1.5 landed by product-owner between v1.2 and v1.3 of this document, per the orchestrator's status update) — not executed by this entry, just reflecting already-completed work. |
| 2026-07-13 (v1.2) | architect | Minted `VP-080` (integration, P1, `internal/discovery`, `status: draft`) for the SEC-DW-07 replay-rejection property, closing the touch-list's own "New VP recommended (not filed by this ruling)" placeholder-language row so product-owner can cite a real ID in the forthcoming BC-2.03.001 Postcondition 2 amendment instead of a `VP-TBD` sentinel — the exact placeholder pattern that cost `S-BL.CLI-SURFACE-COMPLETION` a Forward Obligation and a remediation burst (VP-078/VP-079's own `VP-TBD-PING-A/B` history). `VP-INDEX.md` v2.40→v2.41 (index row, Counts, Phase Distribution, BC Coverage Check, changelog). Downstream touch-list rows for `BC-2.03.001`, the new-VP placeholder, and Summary Table row 1 updated to cite `VP-080` by ID. Version bump rationale: this is genuinely new body content added after the v1.1 pass was already reported to the team-lead as delivered/final — not a continuation of the pre-delivery "rewrite in place" allowance the v1.1 entry below invoked for its own uncommitted-draft correction pass. `lifecycle_status: draft` (not `active`) per `_LIFECYCLE.md`: no wave assignment or story-writer decomposition exists yet for `S-BL.DISCOVERY-WIRE`; full reasoning in `VP-080.md`'s own Lifecycle section. No BC, story, or VP-044/045 file touched by this ruling itself — PO/story-writer/formal-verifier own those per the touch-list. |
| 2026-07-13 (v1.1) | architect | Folded in a completed security-reviewer consult per the v1.0 RECOMMENDED note. Verdict: RULING-1-SOUND-WITH-CONSTRAINTS, RULING-2-SOUND-WITH-CONSTRAINTS, 9 findings (SEC-DW-01..09), all additive, neither core decision overturned. **SEC-DW-01 (HIGH, MANDATORY):** router-side ingest MUST extract key-selector fields (`SVTNID`, `NodeAddr`) via fixed offsets (`body[0:16]`/`body[16:24]`) and MUST NOT run full `decodeBody()` before HMAC verification — closes a pre-auth parsing/DoS surface the v1.0 draft's ambiguous "decodes the body" wording permitted. **SEC-DW-02/03/04:** bounded fixed-size UDP read buffer sized to realistic legitimate usage (not the UDP/IP theoretical maximum) + re-derive `maxSessionsPerAdvertisement` against real usage scale, not a network-MTU figure, at decomposition; two-layer rate defense (aggregate token-bucket at the read loop + the existing `FailureCounter`/E-ADM-017 reused strictly as a visibility-only signal, never a per-`NodeAddr` gate, since that address is attacker-controlled pre-auth); discovery HMAC-failure logging must be rate-limited/counter-based, explicitly NOT inheriting BC-2.05.008's per-packet TCP logging policy (UDP has no per-attempt cost to an attacker the way a TCP handshake does). **SEC-DW-05 (MUST):** no wire-visible accept/reject differential — trivially true for responses (advertisements are one-way, no ack) but binding on processing-time symmetry between lookup-miss and tag-mismatch paths; dummy-HMAC-on-miss remains optional hardening. **SEC-DW-06:** adopted a domain-separated HKDF info label (`HKDFInfoDiscovery`, new `hmac.DeriveDiscoveryKey`) so the new, more-exposed discovery ingest surface cannot be leveraged against the existing session-data `FrameAuthKey` — found no counter-rationale, did not need to stop and report disagreement; this supersedes the v1.0 derivation-rule text (`DeriveKey`/`FrameAuthKey` reuse verbatim), which is now corrected throughout Ruling 1. **SEC-DW-07 (my adjudication):** added a new monotonic `Sequence uint32` envelope field (BC-2.03.001 PC-2 amendment) and a router-held per-`(SVTNID,NodeAddr)` last-seen replay-discard map, rather than accepting the residual — DI-003 does not already cover this threat model (it is scoped to router-compromise, not passive multicast-capture replay, a materially lower attacker bar); design-time cost is small and retrofit-after-ship would be materially larger; the harm directly contradicts BC-2.03.002 PC-5's stated staleness-expiry postcondition, not just a vague availability concern. Cold-start accept-window is bounded (~1 heartbeat interval) and precedented by `admission`'s existing `nonceTTL=60s` bounded-not-perfect replay model. Flagged prominently for the human gate at story-ready per the orchestrator's instruction. **SEC-DW-08:** adopted sender-side multicast TTL=1 plus an ARCH-03 scope-clarifying sentence that `239.0.0.0/8` is addressing hygiene, not a security boundary — HMAC alone is. **SEC-DW-09:** folded into the hop-2 Forward Obligation as a mandatory per-`(SVTNID,NodeAddr)` relay rate cap independent of HMAC validity, protecting against a misbehaving-but-legitimately-admitted sender. Points estimate unchanged at 8 — all nine findings are bounded, narrow additions to already-counted scope, not a new architectural hop. Convention applied: since this document was never committed, Implementation Constraint and derivation-rule prose was rewritten in place to its final v1.1 shape (not left as stale v1.0 text under strikethrough); each Addendum/subsection states explicitly what it supersedes and why, preserving the audit trail without carrying dead text forward. |
| 2026-07-13 (v1.0) | architect | Initial ruling on both Open Design Obligations for `S-BL.DISCOVERY-WIRE`. Ruling 1: admitted-node HMAC key derivation reuses the already-shipped per-(node,SVTN) `FrameAuthKey` (`hmac.DeriveKey`, ADR-001) rather than inventing a new KDF or a new per-SVTN group-secret class (considered and rejected — unnecessary once Ruling 2 establishes router-only verification); verification moves exclusively to the router, with node-local `ReceiveAdvertisement`/`ErrSVTNMismatch` reframed as defense-in-depth (precedent: CLI-SURFACE-COMPLETION Ruling 4 addendum, `E-CFG-011` shape) — no test regression. Security-reviewer consult RECOMMENDED (not blocking) on UDP-ingest resource-exhaustion and lookup-timing-oracle risk, since the reuse itself is fully grounded in already-ratified ADR-001 language ("not a standalone security primitive"). Ruling 2: discovered and resolved a real spec conflict — the story stub's `net.ListenMulticastUDP` instruction and `ARCH-03`'s discovery sketch ("consoles subscribe to multicast") contradict the already-ratified DI-004 domain invariant and BC-2.03.001's own Invariant 1 ("no direct node-to-node multicast"); resolved in favor of the BC/DI-004 (higher precedence, already ratified) over the ARCH-03 sketch (explicitly marked provisional) via a router-only-multicast-membership design that is simultaneously real IP multicast and DI-004-compliant (senders never join the group; only the router does). Address derivation: IPv4 `239.0.0.0/8` (RFC 2365 administratively-scoped), `SHA-256(svtnID)`-derived, static allocation, fixed port (exact number left to architect/PO sign-off). IPv6 explicitly deferred (zero existing data-plane precedent). Lands as ARCH-03 + BC-2.03.001 amendments, not a new ADR, since the governing principle (DI-004) already has a ratified home. Points: 8 (top of stub range), driven by Ruling 2's newly-surfaced relay-hop scope, not by Ruling 1 (which resolved simply). Forward Obligation named (not resolved): exact hop-2 relay wire transport left to decomposition. |
