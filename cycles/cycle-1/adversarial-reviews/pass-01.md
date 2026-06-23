---
artifact_id: adv-p1-pass-01
review_target: phase-1-spec-crystallization
producer: adversary
pass: 1
fresh_context: true
findings_count: 27
findings_by_severity: {critical: 5, high: 11, medium: 9, low: 2}
findings_with_process_gap: 3
verdict: NOT_CONVERGED
timestamp: 2026-06-23
---

# Adversarial Review — Pass 1 (Fresh Context)

## Summary

27 findings (5 critical, 11 high, 9 medium, 2 low), 3 tagged `[process-gap]`.

Critical cluster is concentrated in wire-format and security semantics. The
44-byte outer header math does not work (declared fields sum to 56 bytes).
ARCH-02 and ARCH-04 give incompatible HMAC field locations, with ARCH-04
mid-document acknowledging the contradiction. HMAC keying model is per-SVTN
in ARCH-04 but per-node in three BCs whose security guarantees only hold
under the latter. Version field has three different encodings across BC /
ARCH / VP. SVTN ID size is 8 bytes in entities.md but 16 bytes everywhere
else.

Important findings include a dup-and-race / drop-cache semantic conflict at
intermediate routers (F-006), threshold value divergence between BC-2.06.001
and ARCH-03 (F-008), stale BC bodies that still hedge on architecture
decisions ADR-003/004 have already resolved (F-009), and a Go slice
aliasing bug in the VP-005 fuzz harness skeleton that would silently turn
the 128-bit-flip security test into a no-op (F-013). All 42 BC bodies
retain `VP-TBD` and `[filled by architect]` placeholders even though
VP-INDEX and `architecture_module:` frontmatter are populated (F-018,
F-025).

**Verdict: NOT_CONVERGED**

## Critical Findings

### F-001 — Wire format: 44-byte outer header field sizes sum to 56 bytes, not 44
- **Severity:** critical  •  **Category:** wire-format  •  **Confidence:** high
- **Location:** `.factory/specs/behavioral-contracts/ss-01/BC-2.01.004.md:52`; cross-refs ARCH-02-protocol-stack.md:61–76, VP-001.md:47.
- **Finding:** BC-2.01.004 postcondition 2 declares: `Field layout: [version: 2B][frame_type: 2B][svtn_id: 16B][dst_addr: 8B][src_addr: 8B][length: 4B][hmac: 16B] — total 44 bytes`. The arithmetic is 2+2+16+8+8+4+16 = **56 bytes**. VP-001.md:47 admits "16+8+8+4+16 = 52 bytes of fields" while still claiming 44. The number "44" is repeated throughout L2/L3/L4 (entities.md:115, capabilities.md:51, invariants.md:93, ubiquitous-language.md:117, ARCH-02:61, ARCH-00:124) as a load-bearing protocol-stability assertion (DI-007).
- **Route:** product-owner + architect.
- **Fix:** Choose the actual layout and align BC + ARCH + L2 entities + VPs.

### F-002 — Outer header HMAC location is internally contradictory across L2/L3/L4
- **Severity:** critical  •  **Category:** wire-format  •  **Confidence:** high
- **Location:** ARCH-02-protocol-stack.md:63–76 vs ARCH-04-admission-security.md:122–130, invariants.md:41, BC-2.01.004.md:52.
- **Finding:** ARCH-02 outer header has no HMAC field; ARCH-04:122 says HMAC is in outer header bytes 40..43 then mid-sentence reverses: "Wait — re-reading ARCH-02: ... Correction: the full 44-byte layout as designed places HMAC in a dedicated field. The exact byte offsets are specified in `internal/frame`". L2 invariants/entities, BC-2.01.004 list HMAC as a field; ARCH-02 has none.
- **Route:** architect.
- **Fix:** Replace ARCH-04's hedge with an authoritative byte-by-byte outer header diagram; fix ARCH-02's table; propagate.

### F-003 — HMAC keying contradicts itself: per-SVTN (ARCH-04) vs per-node (BC-2.05.002, BC-2.05.006, BC-2.01.006)
- **Severity:** critical  •  **Category:** security  •  **Confidence:** high
- **Location:** ARCH-04-admission-security.md:136; BC-2.05.002.md:70 EC-002; BC-2.05.006.md:59; BC-2.01.006.md:74 EC-004; BC-2.05.005.md:42, 47.
- **Finding:** ARCH-04:136 makes the HMAC key a **per-SVTN shared secret** derived from router master key + svtn_id. Under that model, any admitted node can forge frames bearing another admitted node's source address. But BC-2.05.002 EC-002, BC-2.01.006 EC-004, and BC-2.05.006 invariant 3 state that HMAC keys are scoped per-node (using the node's admission key). The security claims (no peer-to-peer forgery within an SVTN) are unsatisfiable under the ARCH-04 keying scheme.
- **Route:** architect + product-owner.
- **Fix:** Decide whether HMAC trust boundary is SVTN or node. Affects R-001/R-010 mitigations. Reopen ADR-001 if needed.

### F-004 — Outer header version field has three incompatible encodings across BC / ARCH / VP
- **Severity:** critical  •  **Category:** wire-format  •  **Confidence:** high
- **Location:** BC-2.01.004.md:52; ARCH-02:66; VP-002.md:30, 76.
- **Finding:** ARCH-02:66 = 1 byte, packed `0xMN`. BC-2.01.004:52 = `[version: 2B]`. VP-002:30 = "bytes 0–1 (big-endian uint16) … or full first byte depending on the wire encoding" — the spec doesn't commit. Implementers cannot serialize a single byte.
- **Route:** architect.
- **Fix:** Make ARCH-02 the single source of truth, bit-precise. Update BC-2.01.004 and VP-002 to match.

### F-005 — SVTN ID size contradicts itself: 8 bytes (entities.md) vs 16 bytes (everywhere else)
- **Severity:** critical  •  **Category:** wire-format  •  **Confidence:** high
- **Location:** entities.md:90 vs BC-2.01.004.md:46, BC-2.01.006.md:47, ARCH-02:69, ARCH-04:136, VP-001.md:33, VP-014.md:30, interface-definitions.md:202.
- **Finding:** entities.md:90 says "8-byte hash-derived"; every other artifact says 16-byte / 128-bit. interface-definitions.md:202 says `svtn_id: "..." # 16-byte hex SVTN identifier`. entities.md is internally contradictory — line 125 correctly describes the 8-byte node address.
- **Route:** business-analyst (L2 fix).
- **Fix:** Change entities.md:90 to 16-byte (128-bit) identifier.

## High Findings

### F-006 — Drop cache + duplicate-and-race semantics contradict at intermediate routers
- **Severity:** high  •  **Category:** consistency  •  **Confidence:** medium
- **Location:** BC-2.02.001.md, BC-2.02.009.md:42–55; ARCH-03-routing-engine.md:54–67.
- **Finding:** Dup-and-race requires both frame copies to survive multi-hop; drop cache uses `crc32(outer_header || payload)`; identical bytes → identical checksums → second copy dropped at any shared intermediate router. Mitigation is non-obvious.
- **Route:** architect.
- **Fix:** Include arrival interface in checksum, or make drop cache per-interface-pair, or document edge-disjoint-path requirement.

### F-007 — Hash function for node address differs between BC-2.01.006 (SHA-256) and ARCH-02 (Blake3)
- **Severity:** high  •  **Category:** consistency  •  **Confidence:** high
- **Location:** BC-2.01.006.md:47, 80; ARCH-02:135.
- **Finding:** BC says SHA-256, ARCH says Blake3. Different functions → different addresses. Blake3 also introduces a new dependency not in stdlib.
- **Route:** architect.
- **Fix:** Choose one; if Blake3, justify the dep.

### F-008 — Threshold values in BC-2.06.001 don't match ARCH-03 (2-5x divergence)
- **Severity:** high  •  **Category:** consistency  •  **Confidence:** high
- **Location:** BC-2.06.001.md:48, 53–55; ARCH-03:165–167.
- **Finding:** BC says green <100ms p99, yellow 100–500ms, red >500ms; 5% / 20% loss. ARCH says 50ms/200ms/2%/10%. Numbers differ by factors of 2-5x. NFR-001 (100ms p99 LAN) implies 100ms is the actual budget.
- **Route:** architect + product-owner.
- **Fix:** Reconcile; update one side.

### F-009 — ADR-003 / ADR-004 resolved but BC-2.05.001 / BC-2.05.004 still hedge as "TBD architecture decision"  [partial-fix-regression]
- **Severity:** high  •  **Category:** consistency  •  **Confidence:** high
- **Location:** BC-2.05.001.md:74 EC-002; BC-2.05.004.md:46, 74 EC-003.
- **Finding:** ADRs resolved (LWW for duplicates; key management exclusive to control node). BC bodies still say "must be defined before implementation" and "this BC defers to the architecture decision." Stale.
- **Route:** product-owner.
- **Fix:** Update BC bodies to reference ADR-003 / ADR-004 explicitly.

### F-010 — OQ-003 permission hierarchy not actually resolved by ADR-004 (peer revocation case)
- **Severity:** high  •  **Category:** coverage  •  **Confidence:** medium
- **Location:** invariants.md:158; ARCH-04:43, 60.
- **Finding:** ADR-003 mis-cites "OQ-003 resolution" (OQ-003 is permission hierarchy, not LWW). ADR-004 establishes control > console > access but doesn't answer: can a console revoke a control key? Can a control revoke another control's key (split-brain)?
- **Route:** architect.
- **Fix:** ADR-004 explicit on console→control and control→control revocation.

### F-011 — Frame field set in ARCH-02 outer header table differs from BC-2.01.004
- **Severity:** high  •  **Category:** consistency  •  **Confidence:** high
- **Location:** ARCH-02:64–76; BC-2.01.004.md:52.
- **Finding:** ARCH-02 has 9 fields including `flags, reserved, sequence`; BC has 7 fields including `hmac`. Sets are not equal. ARCH-02 also has `sequence` in outer header AND BC-2.01.005:122 has `chan_seq` in channel header — two sequence numbers per frame.
- **Route:** architect.
- **Fix:** Authoritative field set; likely remove outer-header `sequence` (belongs in channel header).

### F-012 — Channel header layout missing SACK location for BC-2.02.005 ARQ
- **Severity:** high  •  **Category:** consistency  •  **Confidence:** medium
- **Location:** ARCH-02:86–100.
- **Finding:** Channel header has FEC/ARQ flag bytes but no 64-bit SACK bitmap that BC-2.02.005 requires. If SACK is in payload (SSH-opaque), access node can only ACK with payload-bearing upstream frames — but EC-002 talks about standalone ACKs.
- **Route:** architect.
- **Fix:** Specify SACK location in channel header, or define dedicated ACK channel.

### F-013 — VP-005 fuzz harness has Go slice aliasing bug → false-passes
- **Severity:** high  •  **Category:** verification-feasibility  •  **Confidence:** high
- **Location:** VP-005.md:81–93.
- **Finding:** `flipped := tag; flipped[byteIdx] ^= 1 << uint(bitIdx)`. If `tag` is `[]byte`, the slice shares backing array → mutation propagates back to `tag`. After 128 iterations, every bit is XORed multiple times, not tested against the original. Property "128 single-bit flips all rejected" is not actually proven. Formal verifier copying this skeleton ships a no-op test.
- **Route:** architect.
- **Fix:** Use `tag [16]byte` explicitly, or `flipped := append([]byte(nil), tag...)`.

### F-014 — VP-INDEX total (50) vs ARCH-11 sum (46)
- **Severity:** high  •  **Category:** verification  •  **Confidence:** high
- **Location:** VP-INDEX.md:80–84; ARCH-11.md:84–102.
- **Finding:** VP-INDEX declares 50 VPs (28+2+12+6+2=50). ARCH-11 per-module table sums to 46. VP-040's module is "integration" (not a real Go package).
- **Route:** architect.
- **Fix:** Recount; fix VP-040 module assignment.

### F-015 — BC-2.05.007 traces_to: [CAP-020] but subject is private-key non-transit, not HMAC verification  [semantic-anchoring]
- **Severity:** high  •  **Category:** traceability  •  **Confidence:** high
- **Location:** BC-2.05.007.md:31, 96.
- **Finding:** BC-2.05.007's subject is "Node Private Keys Never Transit the Network." Anchored to CAP-020 (HMAC frame auth at router boundary) — mis-fit. BC-INDEX:108 then triple-counts CAP-020 as having BC-2.05.005, BC-2.05.006, BC-2.05.007.
- **Route:** business-analyst + product-owner.
- **Fix:** Add CAP-020a "Private key non-transit"; re-anchor BC-2.05.007.

### F-016 — CAP-020 covered by three BCs with unrelated subjects  [semantic-anchoring]
- **Severity:** high  •  **Category:** coverage  •  **Confidence:** medium
- **Location:** BC-INDEX.md:108; capabilities.md:166–170.
- **Finding:** CAP-020 maps to BC-2.05.005 (HMAC verification), BC-2.05.006 (SVTN cryptographic isolation), BC-2.05.007 (private key non-transit) — three independent security properties under one CAP. SVTN isolation has no CAP.
- **Route:** business-analyst.
- **Fix:** Add CAP-020b "SVTN cryptographic isolation"; retarget BC-2.05.006.

## Medium Findings

### F-017 — BC-2.01.003 independence not verified by VP-016 / VP-017
- **Severity:** medium  •  **Category:** verification  •  **Confidence:** high
- **Location:** ARCH-11.md:28; VP-INDEX.md:42–43.
- **Finding:** BC-2.01.003's core claim is independence of upstream / downstream half-channel clocks and sequences. VP-016 and VP-017 are single-half-channel properties; neither tests independence.
- **Route:** architect.
- **Fix:** Add VP for "two HalfChannels with different tick intervals do not synchronize."

### F-018 — All 42 BCs retain `VP-TBD` and `[filled by architect]` placeholders  [partial-fix-regression]
- **Severity:** medium  •  **Category:** governance  •  **Confidence:** high
- **Location:** Every BC; sampled BC-2.01.001.md:88–90, 98–99.
- **Finding:** Body's Verification Properties table is `VP-TBD` despite VP-INDEX assigning real VPs. Architecture Module body row is `[filled by architect]` despite `architecture_module:` frontmatter being populated.
- **Route:** product-owner (body sweep) + architect (VP back-fill).
- **Fix:** Sweep 42 BCs: replace `VP-TBD` rows with VP IDs from VP-INDEX; replace `[filled by architect]` with `architecture_module` value.

### F-019 — BC criticality vocabulary (`important`, `supportive`) not in module-criticality.md tiers
- **Severity:** medium  •  **Category:** consistency  •  **Confidence:** high
- **Location:** module-criticality.md:26–31; BC frontmatter.
- **Finding:** module-criticality defines CRITICAL / HIGH / MEDIUM / LOW. BCs use `critical / important / supportive`. `supportive` doesn't map; `important` is ambiguous. BC-2.07.003 and BC-2.09.003 are P0 with `criticality: important`.
- **Route:** product-owner.
- **Fix:** Standardize on 4-tier vocabulary; re-evaluate P0 mappings.

### F-020 — BC-2.06.002 missing-frame detection only nominally covered by VP-027
- **Severity:** medium  •  **Category:** verification  •  **Confidence:** medium
- **Location:** ARCH-11.md:59; VP-INDEX.md:53.
- **Finding:** VP-027 is about monotonicity of degradation transitions, not missing-frame detection. BC-2.06.002's core claim isn't exercised.
- **Route:** architect.
- **Fix:** Add VP for "missing expected tick within deadline → indicator downgrade."

### F-021 — Hysteresis count divergent (2 vs 3 vs unspecified)
- **Severity:** medium  •  **Category:** ambiguity  •  **Confidence:** medium
- **Location:** BC-2.01.002.md:70; BC-2.06.001.md:62, 75; ARCH-03; NFR-014.
- **Finding:** BC-2.01.002 EC-001 says ≥3 ticks. BC-2.06.001 invariant 3 says 3-measurement hysteresis. NFR-014 says "within 2 tick cycles." ARCH-03's FSM has no hysteresis count.
- **Route:** product-owner + architect.
- **Fix:** Define one canonical hysteresis; update NFR-014 or BCs to align.

### F-022 — CAP-006 "topX" vague vs BC-2.02.001 "exactly two"
- **Severity:** medium  •  **Category:** ambiguity  •  **Confidence:** medium
- **Location:** capabilities.md:73–74; BC-2.02.001.md:50.
- **Finding:** CAP says topX (X undefined); BC says exactly two; ARCH says top-2 but not as an ADR.
- **Route:** business-analyst.
- **Fix:** Update CAP-006 to "top-2"; add ADR-009 if needed.

### F-023 — BC-2.02.005 EC-003 invents dedicated ACK channel with no spec anywhere
- **Severity:** medium  •  **Category:** ambiguity  •  **Confidence:** medium
- **Location:** BC-2.02.005.md:71.
- **Finding:** "Console sends standalone ACK frames on a dedicated ACK channel" — undefined in CAP-008, ARCH-03, or BC-2.01.005 channel layout.
- **Route:** architect.
- **Fix:** Remove read-only standalone-ACK case (rely on empty-tick frames), or define the ACK channel.

### F-024 — KoS `question-marvel-integration` not explicitly deferred in STATE.md
- **Severity:** medium  •  **Category:** governance  •  **Confidence:** high
- **Location:** `_kos/nodes/frontier/question-marvel-integration.yaml`; bounded-contexts.md:152–155.
- **Finding:** bounded-contexts.md says marvel is out of scope; STATE.md doesn't list it under Deferred decisions. Implicit deferral.
- **Route:** state-manager.
- **Fix:** Add "marvel integration deferred to post-MVP / out of scope" to STATE.md.

### F-025 — BC body back-fill missing from pipeline  [process-gap]
- **Severity:** medium  •  **Category:** governance  •  **Confidence:** high
- **Location:** All 42 BCs (same as F-018).
- **Finding:** No pipeline step says "after VP-INDEX is published, sweep BCs to replace VP-TBD with VP IDs." Recurring inconsistency.
- **Route:** orchestrator (process improvement).
- **Fix:** Add Phase 1b checkpoint step "back-fill BC bodies with architecture_module and VP references." File as drbothen/vsdd-factory issue.

## Low Findings

### F-026 — VSN stragglers in L2 docs despite SVTN canonicalization
- **Severity:** low  •  **Category:** consistency  •  **Confidence:** low
- **Location:** L2-INDEX.md:87; entities.md:125; capabilities.md:105.
- **Finding:** ubiquitous-language.md designates SVTN canonical, VSN legacy. Still 4 uses of VSN in L2-INDEX, entities, capabilities.
- **Route:** business-analyst.
- **Fix:** Sweep VSN → SVTN.

### F-027 — Empty `content:` blocks in 4 of 6 KoS frontier files  [process-gap]
- **Severity:** low  •  **Category:** governance  •  **Confidence:** medium
- **Location:** `_kos/nodes/frontier/question-asymmetric-channels.yaml`, `question-encryption-model.yaml`, `question-marvel-integration.yaml`, `question-timeslice-framing.yaml`.
- **Finding:** Spec `kos_anchors:` references rely on KoS graph being self-explanatory; empty frontier files weaken that. Process-gap: kos process should disallow empty content.
- **Route:** business-analyst (KoS process, not VSDD spec).
- **Fix:** Populate the four empty frontier files; or lint at kos-edge creation time.

## Routing summary

| Agent | Findings to fix |
|---|---|
| **architect** | F-001, F-002, F-003 (shared with PO), F-004, F-006, F-007, F-008 (shared with PO), F-010, F-011, F-012, F-013, F-014, F-017, F-020, F-021 (shared with PO), F-023 |
| **product-owner** | F-001 (propagation), F-003 (propagation), F-008 (propagation), F-009, F-015 (BC trace fix), F-018, F-019, F-021 (propagation) |
| **business-analyst** | F-005, F-015 (CAP split), F-016, F-022, F-026, F-027 (kos) |
| **state-manager** | F-024 |
| **orchestrator → upstream issue tracker** | F-025, F-027 (factory pipeline gap) |
