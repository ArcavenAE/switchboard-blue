---
document_type: burst-log
level: ops
version: "1.0"
status: complete
producer: state-manager
timestamp: 2026-07-13T23:45:00Z
cycle: cycle-1
inputs: [STATE.md]
input-hash: "de76655"
traces_to: STATE.md
---

# Burst Log — S-BL.DISCOVERY-WIRE Elaboration

## Burst 1 (2026-07-13) — Spec-adversarial convergence: rulings v1.3, story v1.1→v2.0

**Agents dispatched:** architect (rulings + ARCH sync), security-reviewer (SEC-DW consult), product-owner (BC-2.03.001, BC-2.01.008), story-writer (S-BL.DISCOVERY-WIRE story + STORY-INDEX)

**Files touched:**
- `decisions/S-BL.DISCOVERY-WIRE-rulings.md` (new, v1.0→v1.3)
- `specs/architecture/ARCH-03-routing-engine.md` (v1.7→v1.8)
- `specs/architecture/ARCH-INDEX.md` (v1.10→v1.12, via two index-sync entries: v1.11 Ruling 2, v1.12 Ruling 3)
- `specs/behavioral-contracts/ss-03/BC-2.03.001.md` (v1.4→v1.5)
- `specs/behavioral-contracts/ss-01/BC-2.01.008.md` (v1.1→v1.2)
- `specs/verification-properties/VP-080.md` (new, v1.0)
- `specs/verification-properties/VP-INDEX.md` (v2.40→v2.41)
- `stories/S-BL.DISCOVERY-WIRE.md` (v1.1→v2.0)
- `stories/STORY-INDEX.md` (v4.96→v4.97)

**Versions bumped:** ARCH-03 v1.7→v1.8; ARCH-INDEX v1.10→v1.12; BC-2.03.001 v1.4→v1.5; BC-2.01.008 v1.1→v1.2; VP-080 minted v1.0; VP-INDEX v2.40→v2.41; STORY-INDEX v4.96→v4.97; story S-BL.DISCOVERY-WIRE v1.1→v2.0 (input-hash `b4e0a5f`).

### Summary

Elaborated `S-BL.DISCOVERY-WIRE` from a 0-AC backlog stub (v1.1) to an 18-AC, 8-point sprint-ready draft (v2.0), closing DRIFT-W6TBD-001 and a spec conflict the stub inherited from `ARCH-03`'s prior discovery sketch (direct node-to-node multicast, which violated ratified DI-004 and BC-2.03.001's own Invariant 1). Story stays `status: draft` — deliberately NOT promoted to `ready` — pending three human-gate sign-offs at the story-ready gate.

**Architect — Rulings v1.0→v1.3 (`decisions/S-BL.DISCOVERY-WIRE-rulings.md`):**
- **Ruling 1** — `DiscoveryAuthKey := hmac.DeriveDiscoveryKey(nodeAdmissionPubkey, svtnID)`, a domain-separated sibling derivation of the shipped `FrameAuthKey` construction, with a distinct `HKDFInfoDiscovery` info label (SEC-DW-06) so the two derived keys are cryptographically independent. Verified exclusively at the router (never independently by access/console nodes). Includes a new monotonic `Sequence uint32` replay-defense field (SEC-DW-07, architect's own adjudication — DI-003's router-compromise scope does not already cover this passive-capture threat model; cold-start always accepts the first frame per `(SVTNID,NodeAddr)`, bounded to ≤1 heartbeat interval, precedented by `admission.nonceTTL`).
- **Ruling 2** — router-only multicast group membership: only the router-mode daemon calls `net.ListenMulticastUDP`; access/console nodes send-only via plain UDP write, never join the group. This is DI-004 enforcement against a stale `ARCH-03` sketch, not a new architectural principle. Address derivation: `239.h0.h1.h2` = first 3 bytes of SHA-256(svtnID), RFC 2365 administratively-scoped range. Senders set outbound multicast TTL=1 (SEC-DW-08). HMAC remains the sole security boundary regardless of actual multicast-routing scope.
- **Ruling 3** (added same day, v1.3, after Ruling 2 surfaced the hop-2 gap) — hop-2 relay rides the existing `FrameTypeCtl` `control_type=0x03` discriminator (`DISCOVERY_RELAY`), zero `HMACTag` (connection-trust boundary, matching the `S-7.04-FU-DRAIN-WIRE` DRAIN precedent), SVTN-scoped exclude-originator best-effort fan-out, ~1/sec per-`(SVTNID,NodeAddr)` rate cap (SEC-DW-09). Fan-out **target resolution** (which live connections belong to a given admitted node) is verified absent from production code (`admission.AdmitNode` has zero production call sites) — flagged as Forward Obligation (f), with two resolution paths named (see Human-Gate item 3 below).
- Rulings v1.0→v1.3 fold in a completed security-reviewer consult across all three rulings: verdict **RULING-1-SOUND-WITH-CONSTRAINTS**, **RULING-2-SOUND-WITH-CONSTRAINTS** — 9 findings (SEC-DW-01 through SEC-DW-09), all additive, none overturning either ruling's core decision. SEC-DW-01 (HIGH, CWE-770/CWE-400, fixed-offset key-selector extraction before full body decode) and SEC-DW-07 (MED, CWE-294, replay) are the two load-bearing adoptions; SEC-DW-02/03/04/05/08/09 are hardening constraints folded into the story's Implementation Constraints and ACs.

**Security consult — SEC-DW-01..09:** all nine findings adjudicated inside the rulings doc's Security Consult Addendum (table with severity/CWE/disposition/landing-site per finding). No blocking disagreement; SEC-DW-06 (domain-separated HKDF label) adopted with no counter-rationale found.

**Product-owner — BC-2.03.001 v1.4→v1.5, BC-2.01.008 v1.1→v1.2:**
- BC-2.03.001: Precondition 3 gains the concrete multicast-address derivation rule; Postcondition 1 gains the router-mediated relay delivery-mechanism note; Postcondition 2 gains the new monotonic `sequence` field + router-side non-increasing discard rule (cites VP-080); Postcondition 5's DRIFT-W6TBD-001 placeholder replaced with the concrete `DiscoveryAuthKey` derivation rule; Invariant 1 (DI-004) reviewed and confirmed already-correct, no change; Verification Properties table gains a VP-080 row.
- BC-2.01.008: Postcondition 2's `control_type` registry gains `DISCOVERY_RELAY = 0x03`; unassigned range narrowed `0x03–0xFF` → `0x04–0xFF`; Postcondition 3's lead sentence rescoped to name DRAIN/RESYNC specifically (was mis-generalized to "all currently-defined opcodes," which broke once a longer-than-4-byte opcode entered the registry) with a citation-level pointer to BC-2.03.001 v1.5/Ruling 3(c) for DISCOVERY_RELAY's own payload layout — schema not duplicated, Invariant 5 (DI-007, "future control messages MAY extend beyond byte 3") not altered.

**Orchestrator dispositions (disk-verified before this burst-record write):**
- Two BC-2.03.001 citation-only wording deviations from product-owner **APPROVED** as correct referent fixes: PC-2's replay qualification now cites "BC-2.03.002 Postcondition 5" (was a looser cross-ref); PC-5's cross-ref corrected to point at PC-2's own in-file replay note.
- BC-INDEX narrative-only version bump **DECLINED** — no BC-INDEX version change this burst; this is a spec-steward convention question, deferred to session close (BC-2.03.001/BC-2.01.008 are in-place amendments to existing BCs, not new BCs, so `l3_bc_count: 45` in STATE.md frontmatter is unaffected).
- PC-3 fixed-length rescope (BC-2.01.008) **APPROVED** — truth-preservation grounded in existing Invariant 5/DI-007, which already permitted future opcodes to extend beyond byte 3; the rescope corrects PC-3's lead sentence rather than altering the invariant.
- Traceability-table extras (Stories row + Related BCs gaining DISCOVERY_RELAY/BC-2.03.001 entries) **APPROVED** — mirrors the existing DRAIN/BC-2.09.002 precedent structurally.

**Story-writer — story v1.1→v2.0 (18 ACs, 8 points, input-hash `b4e0a5f`):**
18 ACs traced to BC-2.03.001 v1.5 (PC 1-3, PC 1-5, Inv 1-3), BC-2.03.002 v1.4 PC-5, BC-2.01.008 v1.2 (PC 2-3, Inv 3/5), and VP-080 v1.0's four property clauses. AC-017/AC-018 (hop-2 fan-out dispatch + rate cap) explicitly GATED on the fan-out target-resolution Forward Obligation; Tasks 1-5 (hop-1 ingest + hop-2 frame construction, AC-001..AC-016) independently deliverable without it. Corrected the v1.1 stub's mislabeled citation (multicast-address-allocation is BC-2.03.001 **Precondition 3**, not Postcondition 1 as the stub stated). Status stays `draft`, `wave` stays `backlog` — NOT promoted to `ready`. STORY-INDEX v4.96→v4.97: Backlog/Deferred Stories table row rewritten (`backlog (v1.1)` → `draft (v2.0, 18 ACs, 8 pts) — elaborated, NOT ready`).

### Human-Gate Items Pending (story-ready gate)

1. **SEC-DW-07 monotonic-`Sequence`-field adjudication** — architect already ruled it in and flagged it prominently for explicit human/PO sign-off before `ready` promotion (adds new wire-format field + router-held state).
2. **Discovery UDP port 49201 bikeshed** — explicit placeholder, needs a human pick.
3. **Hop-2 fan-out target-resolution Forward Obligation** — two options: (i) sequencing dependency on a future node-identity-to-connection-binding story (architect's recommended default); (ii) a narrow story-local `Router.BindInterface` seam, with the architect's own caveat that (ii) only relocates the gap rather than eliminating it (still requires some connection-time identity signal). AC-017/AC-018 are explicitly GATED on this item; Tasks 1-5 / AC-001..016 are independently deliverable regardless of its resolution.

### Infrastructure Notes

- **story-writer dropped twice mid-flight** — once on a 600s stream-watchdog stall, once on an API connection-closed error at the read→write boundary. Both recovered cleanly via orchestrator resume: zero partial writes, zero rework. Third and fourth instances of the mid-stream-drop class observed this cycle (following `adv-cs-i5` in the `S-BL.CLI-SURFACE-COMPLETION` arc).
- **PostToolUse dispatcher false block** — a fail-closed "plugin timed out" block fired during story-writer's `stories/S-BL.DISCOVERY-WIRE.md` Write (validate-factory-path-root, validate-input-hash, validate-template-compliance all timed out mid-dispatch). The write itself persisted correctly and content is valid — confirmed by this burst's disk read of the file. Logged to the orchestrator's gitignored upstream-defect tracker for the session-close batch; not re-litigated here.

---
