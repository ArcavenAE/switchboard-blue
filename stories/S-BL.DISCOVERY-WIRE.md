---
artifact_id: S-BL.DISCOVERY-WIRE
document_type: story
level: ops
story_id: S-BL.DISCOVERY-WIRE
epic_id: E-7
title: "Discovery wire boundary: UDP multicast I/O, admitted-node HMAC keys, multicast address allocation, hop-2 relay dispatch"
status: ready
producer: story-writer
timestamp: 2026-07-01T00:00:00
modified:
  - date: 2026-07-15
    version: "2.13"
    change: >
      Pre-adjudication cascade from `S-BL.DISCOVERY-WIRE-rulings.md` v1.10 (two
      Green-step implementation-time findings, dispatched by team-lead ahead of the
      Step-4.5 adversarial loop). **Ruling 4 (new Forward Obligation (e)):**
      `wireDiscoveryListener` is fully implemented and independently tested but not
      called from `runRouter` — the router process has no source of "which SVTN(s) am
      I serving" (`admission.AdmittedKeySet` has no SVTN-enumeration method; the only
      production `RegisterKey` caller runs in a separate, disconnected control-mode OS
      process). AC-001 ACCEPTED at function level (same shape as `ReceiveAdvertisement`'s
      defense-in-depth reframe and AC-017/AC-018's GATED treatment) but Postcondition 1's
      literal text overclaims daemon-runtime behavior that does not exist today — a new
      "Scope note (Ruling 4, v1.10, 2026-07-15)" paragraph inserted after AC-001's BC
      Anchor line, before Postconditions, names the gap and points to Forward Obligation
      (e) and the new follow-on story `S-BL.ADMISSION-SYNC-WIRE` (working name, not yet
      created). Forward Obligations table gained row (e), citing the same gap and its
      distinction from both "S-6.02, rc.1 gate" and `S-BL.NODE-IDENTIFY-WIRE` (neither of
      which is a home for it — `S-BL.NODE-IDENTIFY-WIRE` is itself blocked by the same
      root cause, since `admission.AdmitNode` is verification-only against a router
      process's always-empty `AdmittedKeySet`). Summary sentence below the Forward
      Obligations table updated: "(b)/(c)" → "(b)/(c)/(e)" remain open and non-blocking;
      "(b)/(c)/(d)" → "(b)/(c)/(d)/(e)" never blocked TDD implementation of
      AC-001..AC-016, with a clause noting (e) is a retrospective finding accepted at
      function level, not a `[GATED]` marker. **Ruling 2 addendum (elaboration, sanctioned
      within existing scope, no new decision):** AC-003 postcondition 1 rewritten —
      sender-side multicast egress now fans out once per UP+multicast-capable local
      interface (`net.ListenUDP` + `WriteToUDP`, each pinned via `setsockopt
      IP_MULTICAST_IF`), not a single `net.WriteTo`/`net.DialUDP` call, elaborated during
      Task 3's Green step because `239.0.0.0/8` does not reliably route to every
      interface on multi-homed hosts; postconditions 2/3 unchanged. Mechanical version-pin
      sweep: every live-prose `rulings v1.9` pin → `v1.10` at 15 spots (`inputDocuments:`
      comment, Status-note blockquote, Decision-section intro, Decision 1's
      Node-local-ingest-correction citation, AC-007's qualifying note, both Architecture
      Compliance Rules rows, AC-017's Gate paragraph, Forward Obligations rows (a) and
      (d) — the latter's paren-form AND its "item (l)" citation, the Human Gate intro
      blockquote's subsection citation, items 1/2/3's disposition-blockquote citations
      ("item (k)" ×2, "item (j)"), and Task 6's body); the sanctioned historical
      `inputDocuments:`-comment narration of "the v1.9 story-ready human gate disposition"
      (describing WHEN that disposition landed, in the document's own running history)
      left unchanged, consistent with this story's established historical-preservation
      precedent for point-in-time citations. `input-hash` recomputed via
      `compute-input-hash --update`: rulings changed on disk again (v1.9→v1.10) —
      `f5135e6` → `8bdbc57`. `acceptance_criteria_count` stays 18; points stay 8. `status`
      stays `ready` (Ruling 4/the addendum are retrospective, implementation-time findings
      dispatched ahead of the adversarial loop, not new Human Gate items — they do not
      reopen the story-ready disposition). Frontmatter `version` 2.12 → 2.13, new
      `modified:` entry added.
  - date: 2026-07-14
    version: "2.12"
    change: >
      Story-ready human gate disposition burst, transcribed from
      `S-BL.DISCOVERY-WIRE-rulings.md` v1.9's "Ruling 3(f) Forward Obligation, SEC-DW-07,
      and the discovery port — human gate disposition" subsection (items (j)/(k)/(l)) and
      `S-BL.DISCOVERY-WIRE-fanout-options.md` v1.1's Disposition section. Human Gate
      section gained a top-of-section DISPOSITIONED blockquote plus per-item disposition
      blockquotes appended to items 1 (SEC-DW-07 both residual bounds APPROVED), 2
      (discovery UDP port `49201` ADOPTED, no longer a placeholder), and 3 (fan-out
      target-resolution resolved to named companion story `S-BL.NODE-IDENTIFY-WIRE` —
      `S-BL.DISCOVERY-WIRE-fanout-options.md` v1.0 Option 1, the human rejected both
      originally-offered options and selected this new option after review). Forward
      Obligations table row (a) replaced with a RESOLVED disposition citing
      `S-BL.NODE-IDENTIFY-WIRE`; row (d) also updated to RESOLVED (`S-BL.SESSIONS-LIST-WIRE`,
      already created, `BC-2.03.002.md` v1.5 already re-points PC-1's annotation); the
      table's summary sentence rewritten to reflect both resolutions. AC-017/AC-018 gate
      tags changed from `[GATED — see Forward Obligation (a) / Human Gate item 3]` to
      `[GATED — depends_on S-BL.NODE-IDENTIFY-WIRE]`; AC-017's Gate paragraph rewritten to
      name the story by ID; AC-018's Gate paragraph left as-is (references AC-017's).
      Decision 2(c)'s port sentence changed from "Recommended... bikeshed-level
      placeholder... Flagged for human sign-off" to "Adjudicated: `49201`... adopted by
      human gate disposition 2026-07-14 (no longer a placeholder)". Frontmatter:
      `depends_on` gained `S-BL.NODE-IDENTIFY-WIRE`; `status: draft` → `ready` (all three
      Human Gate items now dispositioned); `version` 2.11 → 2.12. Mechanical sweep: every
      live-prose `rulings v1.8` pin → `v1.9` (7 spots: `inputDocuments:` comment,
      Status-note blockquote, Decision-section intro, Decision 1's Node-local-ingest-
      correction citation, AC-007's qualifying note, both Architecture Compliance Rules
      rows) plus the one paren-form pin inside Forward Obligations row (d); every live-prose
      `BC-2.03.002 v1.4` pin → `v1.5` (the `inputDocuments:` comment, its description text
      updated to note the PC-1 re-point already executed on disk). Task 6 gating text
      re-anchored by name in the three explicitly-authorized sections: Token Budget's
      "Overall" paragraph, the Task Breakdown section intro, Task 6's heading + body
      (option (i)/(ii) branching removed — collapsed to the single adopted mechanism), the
      Architecture Compliance Rules relay-dispatch row's Enforcement cell, and Task 7's PR
      note — all now read "`S-BL.NODE-IDENTIFY-WIRE`" by name instead of "Human Gate item
      3"/"Forward Obligation (a)". Four residual `Forward Obligation (a)`/`Human Gate item
      3` references found OUTSIDE the authorized touch-list (Non-Goals bullet, EC-008,
      File-Change List's `discovery_relay_wire_test.go` row, Anchors Consumed's two hop-2
      rows) are deliberately left unedited this burst — flagged for a follow-on sweep, not
      silently expanded into. `input-hash` recomputed: two of five declared `inputs:`
      changed on disk (rulings v1.8→v1.9, BC-2.03.002 v1.4→v1.5) — `a39b7ad` → `f5135e6`.
      `acceptance_criteria_count` stays 18; points stay 8. New backlog stub created as part
      of this same disposition: `S-BL.NODE-IDENTIFY-WIRE.md` v1.0 (draft, wave backlog, 0
      ACs).
  - date: 2026-07-14
    version: "2.11"
    change: >
      Remediated spec-adversarial pass 13 finding F-DWSP13-001 (LOW): two live-prose spots
      attributed the F-DWSP4-001 restart-liveness amendment (`Sequence` `uint32`→`uint64`
      epoch-qualified widening + offset consequences) to "rulings v1.6". The canonical adoption
      version is rulings v1.5 (rulings.md's "Replay / freshness — restart-liveness amendment
      (F-DWSP4-001 ... — v1.5 adjudication)" section); v1.6 was ONLY the residual-bounds
      precision correction and changed no widths or offsets. Corroborated by BC-2.03.001 PC-2's
      blockquote ("F-DWSP4-001, v1.5"), VP-080's history, and this story's own line 1270
      ("v1.5 update (F-DWSP4-001), residual bounds corrected v1.6"), which directly contradicted
      line 1257 within the same Human Gate item 1. Fixed: AC-005's note (line 909) — "widths
      updated per F-DWSP4-001/rulings v1.6" → "...v1.5"; Human Gate item 1's opening sentence
      (line 1257) — "epoch-qualified per the v1.6 restart-liveness amendment" →
      "...v1.5 restart-liveness amendment". Line 1270's correct dual-version phrasing left
      unchanged. **Exception-set retirement:** prior sweeps (v2.7-v2.10 rows) classified these
      two spots as "sanctioned point-in-time historical `rulings v1.6` citations" and verified
      them unchanged burst after burst — that classification protected the version-at-fix-time
      reading without checking the text's actual semantic claim, which falsely attributed the
      AMENDMENT itself to v1.6. This fix retires that exception class; the historical
      classification in the v2.7 through v2.10 rows themselves is left unedited
      (historical-preservation precedent) — this entry layers the correction forward.
      **Convention going forward:** amendment attributions cite the ADOPTING version (v1.5);
      "residual bounds corrected v1.6" is the correct dual-version formula where the v1.6
      precision correction is also relevant (as line 1270 already does). Mandatory
      multiline-tolerant re-certification sweep (Perl `-0777`, the ratified v2.9/v2.10 pattern
      set: `VP-080\s+v1\.[0-9]+`, `` rulings(\.md)?[`']?\s+v1\.[0-9]+ ``,
      `rulings\s*\(v1\.[0-9]+\)`, `VP-080\s*\(v1\.[0-9]+\)`) found a THIRD spot making the
      identical false attribution: "The architect's fix (rulings v1.6): widen `Sequence` to
      `uint64`, epoch-qualified..." (Human Gate item 1, four lines below the correctly-phrased
      v1.5/v1.6 dual-version paragraph opener) — the widening fix itself credited to v1.6 rather
      than v1.5. Initially flagged out of scope pending disposition; **the orchestrator extended
      this same burst to cover it** (class-sweep principle — fix every instance of a finding's
      class in one burst rather than leaving a guaranteed pass-14 finding). Fixed in place,
      same v2.11 (uncommitted, amended not re-versioned): "The architect's fix (rulings v1.6):"
      → "The architect's fix (rulings v1.5; residual bounds corrected v1.6):" — the same
      dual-version formula line 1270 already uses; nothing else in the sentence changed. **All
      THREE same-class spots (AC-005's note at line 909, Human Gate item 1's opening sentence at
      line 1257, and Human Gate item 1's fix-origin sentence at line 1274) are now fixed in
      v2.11.** Re-ran the full re-certification sweep after this third fix: every live-prose
      `VP-080` hit reads `v1.7` (5 spots); every live-prose `rulings` hit reads `v1.8` (6 spots,
      plus the paren-form at line 1241) or the exempt Provenance "Adjudication:" bullet (`v1.3`,
      line 1643) — **zero live-prose `rulings v1.6` pins remain anywhere in the file, in any
      form.** The exception-set retirement now FULLY holds. `acceptance_criteria_count` stays
      18; points stay 8. None of this story's five declared `inputs:` (rulings, BC-2.03.001,
      BC-2.03.002, BC-2.01.008, ARCH-03) changed this burst: `compute-input-hash --check`
      confirms `a39b7ad` holds unchanged.
  - date: 2026-07-14
    version: "2.10"
    change: >
      Cascade-only pin sweep for F-DWSP12-001 (LOW, spec-adversarial pass 12): the architect
      corrected VP-080's Source Contract process-status paragraph (it had read "drafted, not yet
      executed" since the v1.0 mint even though the BC-2.03.001 PC-2 amendment landed at v1.5 and
      was superseded in place at v1.6) and bumped VP-080 to v1.7 — citation-only, no
      property-substance change (Properties 1-5, Test Scenarios, thresholds unchanged), input-hash
      unchanged (`5d904d5`); VP-INDEX bumped to v2.47. This entry is the story-side pin sweep.
      Updated all six live-prose `VP-080 v1.6` pins to `v1.7`: the `inputDocuments:` comment
      (gained a new v1.7 historical clause; the prior "v1.6 replaces..." clause tense-shifted to
      "v1.6 replaced..." per this entry's own established pattern), AC-009's note ("VP-080 v1.6
      Test Scenario 5"), AC-010 postcondition 5 ("VP-080 v1.6 property 4"), AC-010 postcondition 6
      ("VP-080 v1.6 Property 5"), the Non-Goals `uint64`-composite-wraparound bullet
      (whitespace-tolerant line-wrap edit, the F-DWSP6-001-established pattern), and the
      File-Change List's `discovery_wire_test.go` row ("VP-080 v1.6 cites as the surviving
      lineage" — swept forward to v1.7, since VP-080's v1.7 fix touched only the Source Contract
      paragraph and left the Proof Method table's surviving-lineage citation this row references
      unchanged and still current — the row's claim remains true under the new version). Mandatory
      multiline-tolerant re-certification sweep (Perl `-0777` over the whole file as one buffer,
      the ratified v2.9 pattern set: `VP-080\s+v1\.[0-9]+`, `` rulings(\.md)?[`']?\s+v1\.[0-9]+ ``,
      `rulings\s*\(v1\.[0-9]+\)`, plus a paren-form `VP-080\s*\(v1\.[0-9]+\)` check): every
      live-prose `VP-080` hit now reads `v1.7`; every live-prose `rulings` hit already reads
      `v1.8`, with the same two sanctioned point-in-time historical `rulings v1.6` citations
      (AC-005's F-DWSP1-001 fix-context note; Human Gate item 1's SEC-DW-07-fix-origin note) and
      the Provenance "Adjudication:" bullet (`rulings.md` v1.3, correctly documenting the v2.0
      elaboration's authority set at the time) verified unchanged; zero paren-form `VP-080` hits
      found anywhere in the file. `acceptance_criteria_count` stays 18; points stay 8. None of
      this story's five declared `inputs:` (rulings, BC-2.03.001, BC-2.03.002, BC-2.01.008,
      ARCH-03) changed this burst: `compute-input-hash --check` confirms `a39b7ad` holds
      unchanged.
  - date: 2026-07-14
    version: "2.9"
    change: >
      Remediated spec-adversarial pass 11 finding F-DWSP11-001 (LOW): the Forward Obligations
      table row (d) at line 1175 read "**None of the three rulings (v1.3) adjudicate this**" — a
      straggler from the v2.0 elaboration (rulings was v1.3 at that time; it is v1.8 now). The
      paren-separated form `rulings (v1.3)` is structurally invisible to every prior sweep's
      pattern (`` rulings(\.md)?[`']?\s+v1\.[0-9]+ `` requires the version token immediately
      after the name, not parenthesized) — the third sweep-blind-spot sub-class surfaced on this
      story, after F-DWSP6-001's line-wrap survivor (v2.5) and F-DWSP10-001's retired-test
      exemplar cascade (v2.8). The claim's truth is unaffected — no ruling after v1.3 adjudicates
      the `sessions.list` RPC question; Rulings 1/2/3's scope remains exclusively the
      UDP-multicast advertisement transport — hence LOW, not MED/HIGH. Fixed: `v1.3` → `v1.8` at
      line 1175, cell text otherwise unchanged. **Correction to the v2.8 row's completeness
      claim:** that entry asserted "every live-prose `rulings` hit already reads `v1.8` except the
      two intentional point-in-time historical `rulings v1.6` citations" — true under the sweep
      pattern in force at the time, but that pattern was blind to the paren form, so the claim
      did not in fact cover every live-prose hit. The v2.8 row itself is left unedited per this
      story's historical-preservation precedent; this note layers the correction alongside it.
      **New sweep standard for this story (F-DWSP6-001/F-DWSP10-001/F-DWSP11-001 countermeasure
      lineage):** the mandatory multiline-tolerant Perl `-0777` re-certification sweep now also
      runs a paren-tolerant pattern, `rulings\s*\(v1\.[0-9]+\)`, alongside the existing
      `` rulings(\.md)?[`']?\s+v1\.[0-9]+ `` and `VP-080\s+v1\.[0-9]+` patterns. This burst's
      extended sweep (both rulings patterns plus the VP-080 pattern, whole file as one buffer)
      found exactly one paren-form hit — the line-1175 fix above — and reconfirmed every other
      live-prose `rulings` hit already reads `v1.8` and every live-prose `VP-080` hit already
      reads `v1.6`, with the same two sanctioned point-in-time historical `rulings v1.6` citations
      (AC-005's F-DWSP1-001 fix-context note; Human Gate item 1's SEC-DW-07-fix-origin note) and
      the Provenance "Adjudication:" bullet (`rulings.md` v1.3, correctly documenting the v2.0
      elaboration's authority set at the time) verified unchanged. A broader case-insensitive
      paren check for both `rulings` and `VP-080` near any parenthesized version number confirmed
      no further paren-form hits of either kind exist anywhere in the file.
      `acceptance_criteria_count` stays 18; points stay 8. None of this story's five declared
      `inputs:` (rulings, BC-2.03.001, BC-2.03.002, BC-2.01.008, ARCH-03) changed this burst:
      `compute-input-hash --check` confirms `a39b7ad` holds unchanged.
  - date: 2026-07-14
    version: "2.8"
    change: >
      Remediated spec-adversarial pass 10 finding F-DWSP10-001 (MED): three live spots cited the
      retired `TestDiscovery_VP045_SVTNIsolation_MultipleScopes` as an EXTANT exemplar to extend —
      a propagation gap the F-DWSP8-001 retirement (v2.6) never fully cascaded. Architect fixed
      VP-080's two spots (v1.6: the Proof Method table's Tool cell and the Feasibility Assessment's
      Proof-complexity Notes cell, both re-cited to the surviving router-side
      `DiscoveryAuthKeyFor`-admitted lineage) — citation-only, no property-substance change,
      input-hash unchanged (`5d904d5`, none of VP-080's three declared inputs changed) — and bumped
      VP-INDEX to v2.46. The third spot was this story's own File-Change List line 1353
      (`discovery_wire_test.go` row), which contradicted the adjacent line-1354 row's "retired
      outright, not extended" framing by still describing the row as extending the retired test's
      family. Fixed: line 1353's cell text replaced verbatim with the architect's supplied
      blockquote — now frames AC-005/AC-006 as establishing the admitted-node/SVTN
      `DiscoveryAuthKeyFor`-admitted test-setup pattern that VP-080 v1.6 cites as the surviving
      lineage after the retired test's outright retirement, resolving the 1353/1354 contradiction.
      Story-side pin sweep updated all five live-prose `VP-080 v1.5` pins → `v1.6`: the
      `inputDocuments:` comment (gained a new v1.6 historical clause; the prior "v1.5 is a citation
      re-pin..." clause tense-shifted to "v1.5 was..." per this entry's own established pattern),
      AC-009's note ("VP-080 v1.5 Test Scenario 5"), AC-010 postcondition 5 ("VP-080 v1.5 property
      4"), AC-010 postcondition 6 ("VP-080 v1.5 Property 5"), and the Non-Goals
      `uint64`-composite-wraparound bullet. Mandatory multiline-tolerant re-certification sweep
      (Perl `-0777` over the whole file as one buffer, patterns `VP-080\s+v1\.[0-9]+`,
      `` rulings(\.md)?[`']?\s+v1\.[0-9]+ ``, `TestDiscovery_VP045_SVTNIsolation_MultipleScopes`):
      every live-prose `VP-080` hit now reads `v1.6`; every live-prose `rulings` hit already reads
      `v1.8` except the two intentional point-in-time historical `rulings v1.6` citations (AC-005's
      F-DWSP1-001 fix-context note; Human Gate item 1's SEC-DW-07-fix-origin note) — verified
      correctly unchanged, consistent with every prior sweep this session; every remaining
      retired-test mention (the `inputDocuments:` rulings/VP-045 comments, AC-004 postcondition 5's
      qualifying note, AC-007's test names, Task 4) already correctly frames the test as
      retired/historical — verified, zero further drift found beyond the one line-1353 fix.
      `acceptance_criteria_count` stays 18; points stay 8. VP-080 is NOT one of this story's five
      declared `inputs:` (rulings, BC-2.03.001, BC-2.03.002, BC-2.01.008, ARCH-03), and none of
      those five changed this burst: `compute-input-hash --check` confirms `a39b7ad` holds
      unchanged.
  - date: 2026-07-14
    version: "2.7"
    change: >
      Remediated spec-adversarial pass 9 finding F-DWSP9-001 (MED): fix-burst 6 (pass-8,
      F-DWSP8-001) bumped the rulings doc v1.7→v1.8 but dropped VP-080's established
      re-pin-on-every-rulings-bump cascade (the v2.3→v2.4 / rulings v1.6→v1.7 / VP-080 v1.3→v1.4
      precedent). Architect cascaded VP-080 v1.4→v1.5 (citation re-pin + input-hash refresh only —
      Properties 1-5, Test Scenarios, and thresholds unchanged, no property-substance change) and
      VP-INDEX to v2.45; this entry is the story-side pin sweep. Updated all live-prose `VP-080
      v1.4` pins → `v1.5` at the five spots certified complete by the v2.5 mandatory
      re-certification sweep: the `inputDocuments:` comment, AC-009's note ("VP-080 v1.4 Test
      Scenario 5"), AC-010 postcondition 5 ("VP-080 v1.4 property 4"), AC-010 postcondition 6
      ("VP-080 v1.4 Property 5"), and the Non-Goals `uint64`-composite-wraparound bullet — the
      F-DWSP6-001 line-wrap-survivor spot, re-verified with a whitespace-tolerant edit. Mandatory
      multiline-tolerant re-certification sweep (Perl `-0777` over the whole file as one buffer,
      patterns spanning newlines: `VP-080\s+v1\.[0-9]+`, `rulings(\.md)?['`]?\s+v1\.[0-9]+`,
      `VP-INDEX\s+v2\.[0-9]+`): every live-prose `VP-080` hit now reads `v1.5`; every live-prose
      `rulings` hit already reads `v1.8` (Status-note blockquote, Decision-section intro, Decision
      1's full-rationale citation, both Architecture Compliance Rules rows, AC-004's F-DWSP8-001
      qualifying note, the `inputDocuments:` comment) — no rulings drift found this burst. The two
      point-in-time historical citations pinned to `rulings v1.6` (AC-005's
      F-DWSP1-001/F-DWSP4-001 fix-context note; Human Gate item 1's SEC-DW-07-fix-origin note) and
      the `VP-INDEX v2.43` citation embedded in the `VP-080` `inputDocuments:` comment (documents
      the VP-INDEX version active when Property 5 was added at VP-080 v1.3, not a current-state
      claim) verified as correctly unchanged — consistent with every prior sweep this session.
      `acceptance_criteria_count` stays 18; points stay 8. VP-080 is NOT one of this story's five
      declared `inputs:` (rulings, BC-2.03.001, BC-2.03.002, BC-2.01.008, ARCH-03), and none of
      those five changed this burst: `compute-input-hash --check` confirms `a39b7ad` holds
      unchanged.
  - date: 2026-07-14
    version: "2.6"
    change: >
      Remediated spec-adversarial pass 8 finding F-DWSP8-001 (HIGH): AC-004's `advertisementKey`
      deletion structurally broke the VP-045 test AC-007 mandated "passes unmodified" — rooted in
      rulings Implementation Constraint 3's false claim ("`ReceiveAdvertisement` preserved unchanged
      in shape"), now corrected. Architect adjudication: `Discovery.ReceiveAdvertisement` is RETIRED
      (deleted, not preserved) — it was one of THREE `advertisementKey` call sites (`Encode`,
      `Decode`, `ReceiveAdvertisement`), not the two this story's scoping implied, and a node
      structurally has no key to derive for an arbitrary sender per BC-2.03.001 v1.6 Postcondition
      5. Replaced by a new node-side relay-ingest function: decodes the hop-2 `DISCOVERY_RELAY`
      payload, no per-frame HMAC (trust = the admitted connection, AC-015), and relocates
      `ErrSVTNMismatch` to a direct `OuterHeader.SVTNID` vs. `d.cfg.LocalSVTNID` equality check;
      same registry replace-on-write semantics. `TestDiscovery_VP045_SVTNIsolation_MultipleScopes`
      is RETIRED outright, not extended. Six architect blockquotes applied verbatim: (1) AC-004
      postcondition 5 gained a qualifying note naming the three call sites and clarifying
      `ReceiveAdvertisement`'s retirement — TD-031 deviation: the supplied blockquote cited
      `discovery.go:319`/`:369`/`:399` line-number anchors, converted to symbol-only citations
      (`Encode`/`Decode`/`ReceiveAdvertisement`) per this story's no-volatile-line-number-citation
      posture; (2) AC-007 fully rewritten (title, BC Anchor, all 5 postconditions, test
      names/level/file) describing the new node-local relay-ingest function; (3) File-Change
      List's `discovery_test.go` row replaced — widened from a one-test regression-pin claim to
      the full ten-test disposition (`TestDiscovery_Enumerate_NoHostnameRequired`,
      `..._SameSessionNameTwoNodes`, `TestDiscovery_Advertise_HMACAuthenticated`, `..._EmptyPayload`,
      `..._TagCorruption`, `TestDiscovery_Enumerate_SVTNIsolation`, `..._ForgedSVTN`,
      `..._ErrSentinel`, `TestDiscovery_VP045_SVTNIsolation_MultipleScopes`,
      `TestDiscovery_Decode_RejectsZeroLengthName` — nine rewritten against the router-side
      `DiscoveryAuthKeyFor`-admitted model, the VP045-named test retired outright); (4) Task 4 fully
      rewritten (title + body) from a verification-only checkpoint to a Red/Green implementation
      task retiring `ReceiveAdvertisement` and adding the relocated `ErrSVTNMismatch` guard; (5)
      Decision 1's `ReceiveAdvertisement` bullet replaced in full, correcting the "preserved
      unchanged in shape" claim; (6) the S-7.02 Previous Story Intelligence anchors-table row's
      trailing sentence replaced, correcting the same claim. Mechanical rulings version-pin sweep
      v1.7→v1.8 (whitespace/multiline-tolerant Perl `-0777` procedure, per the F-DWSP6-001
      certification): five live spots fixed — the Status-note blockquote, the Decision-section
      intro sentence, both Architecture Compliance Rules rows citing Ruling 1 IC-2/Ruling 3(b), and
      the `inputDocuments:` rulings comment (which also gained a parenthetical noting the new v1.8
      Node-local ingest correction entry, matching the pattern of its v1.6/v1.7 predecessors).
      VP-045 `v1.3`→`v1.4` fixed at its one live spot, the `inputDocuments:` comment (description
      text unchanged — VP-045 v1.4 corrects only a stale supporting-evidence citation to the now-
      retired test; PARTIAL status and the real-socket PC-3 gap are unaffected, confirmed by
      reading `VP-045.md` v1.4 directly). Two historical parenthetical citations of "rulings v1.6"
      (AC-005's F-DWSP1-001/F-DWSP4-001 fix-context note; Non-Goals' `Sequence`-widening item's
      "v1.5 update... residual bounds corrected v1.6" note) verified as pre-existing historical
      citations of the ruling version active at those specific past fixes, not current-state
      claims — left unchanged, consistent with every prior sweep this session (never included in
      the "five live spots" count in any burst). BC-2.03.001 confirmed by architect to need NO
      amendment this pass. `acceptance_criteria_count` stays 18 (AC-007 rewritten in place, not
      added/removed); points stay 8. `compute-input-hash --update` re-run (rulings v1.7→v1.8 is the
      only declared-input change; VP-045 is not one of this story's five declared `inputs:`):
      `eccbdc4` → `a39b7ad`.
  - date: 2026-07-13
    version: "2.5"
    change: >
      Remediated spec-adversarial pass 6 finding F-DWSP6-001 (MED): the v2.4 sweep's completeness
      claim ("all live-prose `VP-080 v1.3` pins updated to `v1.4`") was FALSE — one instance
      survived because it line-wrapped across the ID/version boundary (`VP-080` at end of one
      line, `v1.3` at the start of the next), and the v2.4 sweep used single-line-based matching
      that cannot see across a wrap. Location: the Non-Goals section's `uint64`-composite-
      wraparound bullet — "out of scope for this story and VP-080\n  v1.3 property 4". Fixed:
      `v1.3` → `v1.4`. This layers a correction onto the v2.4 row's claim rather than editing it
      — the v2.4 historical entry is left untouched (it accurately reports what that burst
      believed it had done; this entry states the burst was incomplete and why, per the
      historical-preservation precedent). **Mandatory re-certification sweep, whitespace/
      multiline-tolerant (the DRAIN story's F-SP19-001 countermeasure, applied here):** ran
      Perl regex over the whole file as one buffer (`\s+` matches newlines, so ID/version pairs
      split across a wrap are caught) for five patterns — `VP-080\s+v1\.[0-9]+`,
      `rulings\.md['\`]?\s+v1\.[0-9]+`, `BC-2\.03\.001\s+v1\.[0-9]+`, `VP-INDEX\s+v2\.[0-9]+`,
      `ARCH-03\s+v1\.[0-9]+` (plus `BC-2\.03\.002\s+v1\.[0-9]+`/`BC-2\.01\.008\s+v1\.[0-9]+` for
      completeness) — then classified every hit by line number as live-prose or history-layer
      (frontmatter `modified:` entries, the Provenance "Adjudication:" bullet, and body
      Changelog rows are exempt, per the established precedent). A second pass allowing markdown
      formatting characters between ID and version (`` VP-080[`*_\s]{1,20}v1\.[0-9]+ ``) found no
      additional hits beyond the whitespace-only pattern. Full result: every live-prose hit
      already reads the current version — `VP-080` → `v1.4` at the inputDocuments comment, AC-009's
      note, AC-010 postconditions 5/6, and this Non-Goals bullet (5 live spots, all v1.4 after
      this fix); `rulings.md` → `v1.7` at the inputDocuments comment, the Status-note blockquote,
      the Decision-section intro sentence, and both Architecture Compliance Rules rows (5 live
      spots, all v1.7); `BC-2.03.001` → `v1.6` at the inputDocuments comment and all 7 Anchors
      Consumed table rows (unchanged this burst — BC-2.03.001 did not bump; still v1.6, correct);
      `VP-INDEX`/`ARCH-03`/`BC-2.03.002`/`BC-2.01.008` citations are all static inputDocuments-
      comment references with no additional live occurrences and no reported version bump this
      burst. Zero further stale live pins found. `acceptance_criteria_count` stays 18; points
      stay 8. `compute-input-hash --check`: no declared `inputs:` file changed this burst
      (rulings/BC-2.03.001/VP-080 on-disk versions unchanged since v2.4) — hash `eccbdc4` holds,
      confirmed clean.
  - date: 2026-07-13
    version: "2.4"
    change: >
      Pass-5 fix-burst cascade: the rulings doc bumped v1.6→v1.7 (F-DWSP5-001, a one-token
      propagation fix to Ruling 3(c)'s trailing prose — `byte[18:]`→`byte[22:]` — no ruling
      content change) and VP-080 bumped v1.3→v1.4 (housekeeping, alongside ARCH-07/ARCH-11
      gaining the VP-078/079/080 rows). No story content changed — this is a citation/hash
      refresh only. Updated the same five live-prose rulings-version-pin spots fixed at v2.3:
      the Status-note blockquote, the "Adjudicated Design Decisions" intro sentence, two
      Architecture Compliance Rules rows, and the `inputDocuments:` comment — all v1.6→v1.7.
      Updated all live-prose `VP-080 v1.3` pins to `v1.4`: the `inputDocuments:` comment, AC-009's
      note ("VP-080 v1.3 Test Scenario 5"), and AC-010 postconditions 5/6 ("VP-080 v1.3 property
      4"/"VP-080 v1.3 Property 5"). Verified (not assumed) that this story's own Decision 3(c)
      diagram and AC-014 postcondition 2 do NOT carry the stale `byte[18:]` sessions offset the
      rulings doc's v1.7 fix corrected upstream — both already read `byte[22:]`/`bytes 22+` from
      the v2.3 mechanical sweep; pass 5 independently confirmed this story clean. Left UNCHANGED
      per the established historical-preservation precedent: all `modified:`/Changelog entries'
      own historical version citations (including this v2.3 entry's own "rulings v1.6"/"VP-080
      v1.3" narrative text, which correctly describes that burst's authority set at the time) and
      the Provenance "Adjudication:" bullet. `acceptance_criteria_count` stays 18; points stay 8.
      `compute-input-hash --update` re-run (rulings v1.7 changed on disk): see hash delta in
      Changelog row below.
  - date: 2026-07-13
    version: "2.3"
    change: >
      Remediated spec-adversarial pass 4 finding F-DWSP4-001 (HIGH): SEC-DW-07's original
      in-memory-counter `Sequence` field, paired against the router's restart-STABLE `lastSeen`
      watermark, produced up to ~8.3h of silent discovery lockout on every ordinary access-node
      restart/redeploy/crash-recover — not a rare edge case, a routine-operations liveness gap.
      Architect fix (rulings v1.6): `Sequence` widened `uint32`→`uint64`, epoch-qualified (high
      32 bits = `uint32(time.Now().UTC().Unix())` sampled at `Discovery`-instance start, low 32
      bits = the original counter) — self-contained at the sender, no router-side logic change,
      no dependency on the still-open fan-out target-resolution Forward Obligation. Authority: 
      rulings v1.6 (Ruling 1 "restart-liveness amendment" subsection + v1.6 precision-correction
      Decision Log entry), BC-2.03.001 v1.6, VP-080 v1.3, VP-INDEX v2.43. Five architect
      blockquotes applied verbatim: (1) AC-008 gained a scope-clarification note distinguishing
      fresh-cold-start from a stable-identity restart; (2) AC-009 gained a note characterizing
      its discard rule as correct for THREE triggers — genuine replay, the ≤1s
      same-epoch-second crash-loop residual (EC-010 case 1), and the N-bounded
      backward-clock-adjustment residual (EC-010 case 2); (3) AC-010 postcondition 5 REPLACED
      (`uint32` wraparound-out-of-scope → `uint64` composite wraparound-out-of-scope with
      per-component wrap bounds) and postcondition 6 ADDED (restart forward-acceptance path) +
      new test name `TestVP080_DiscoveryIngest_RestartForwardProgress`; (4) new EC-010 Edge
      Cases row characterizing the restart-liveness fix and both residual cases with their
      different bounds; (5) Human Gate item 1 gained an appended v1.5/v1.6 update paragraph
      naming both residuals for explicit human sign-off. Deviation flagged per instruction: the
      supplied AC-010 blockquotes cited "VP-080 v1.2" — corrected to "VP-080 v1.3" (the current
      version) in both postcondition 5 and postcondition 6 as applied. Mechanical sweeps
      (rulings v1.6-authoritative): hop-1 wire layout `[4]Sequence`→`[8]Sequence` and 38→42-byte
      full-valid-frame minimum in AC-005's note (raw-min-32/body-min-24 pre-lookup guard left
      UNCHANGED, upstream of `Sequence`, per explicit instruction); hop-2 `DISCOVERY_RELAY`
      payload in Decision 3(c)'s diagram and AC-014 postcondition 2 — `Sequence`
      byte[12:16]→byte[12:20], count byte[16:18]→byte[20:22], sessions byte[18:]→byte[22:],
      fixed pre-session header 18→22 bytes. Additional live-prose sweep instances corrected:
      Human Gate item 1's original "`Sequence uint32`" claim → "`Sequence uint64`
      (epoch-qualified...)"; Non-Goals' "`uint32` `Sequence` wraparound" bullet →
      "`uint64` composite `Sequence` wraparound"; File-Change List's `discovery.go` row
      "`Sequence uint32` field" → "`Sequence uint64` field" with the epoch-qualification detail
      inline. Version-pin sweep: every live-prose citation of the rulings doc and VP-080 with an
      explicit version number updated to v1.6/v1.3 respectively (frontmatter `inputDocuments:`
      comments; the "Status note" blockquote; the "Adjudicated Design Decisions" intro sentence;
      two Architecture Compliance Rules table rows) — extended one step beyond the literal
      instruction (which named only "VP-080"/"rulings") to also update the `inputDocuments:`
      comment for `BC-2.03.001.md` (`v1.5`→`v1.6`), since that comment is input-version metadata,
      not prose. **Amended in place, same v2.3 (post-pass-4, pre-commit):** the orchestrator
      identified this elaboration's own initially-flagged "candidate for a future sweep" as the
      same stale-version-pin class pass 1 flagged on VP-080 (F-DWSP1-002) — pass 5 would have
      re-flagged it. Fixed now, in place: all live-prose `BC-2.03.001 v1.5` citations updated to
      `v1.6` — the Anchors Consumed table's 7 rows, and two Decision 3 prose sentences (3(c)'s
      "BC-2.03.001 PC-5 (v1.5, already landed)" and 3(h)'s "v1.5's PC-5 sentence"). Left
      deliberately UNCHANGED, per the historical-preservation precedent established at v2.1/v2.2:
      the v1.1/v2.0/v2.1/v2.2 `modified:`/Changelog entries' own rulings/VP-080/BC-2.03.001
      version citations (they describe what was true at each entry's own time — the v2.0
      Changelog row's "BC-2.03.001 v1.5" citation was transiently corrupted by an overbroad
      `replace_all` during this same amendment and restored to `v1.5`); the Provenance section's
      "Adjudication:" bullet (explicitly anchored to "This elaboration (v2.0)"). No further
      BC-2.03.001 version-pin instances remain outside historical entries.
      `acceptance_criteria_count` stays 18 (no AC added or removed — two new
      postconditions/notes, not new ACs); `points`/`estimated_points` stay 8.
      `compute-input-hash --update` re-run: rulings AND BC-2.03.001 both changed on disk — see
      hash delta in Changelog row below; this same-v2.3 amendment is prose-only (no `inputs:`
      file touched) and re-verified via `compute-input-hash --check` (unaffected, still clean).
  - date: 2026-07-13
    version: "2.2"
    change: >
      Remediated spec-adversarial pass 3 finding F-DWSP3-001 (MED, sibling of F-DWSP1-001's
      off-by-N-bytes threshold-miscount class). AC-005 Postcondition 3 stated the HMAC
      computation covers "the complete raw body bytes, not merely the 16-byte key-selector
      prefix" — the key-selector region per SEC-DW-01 is `SVTNID` `body[0:16]` (16 bytes) +
      `NodeAddr` `body[16:24]` (8 bytes) = 24 bytes, so naming both fields and then citing
      "16-byte" undercounted the region by the `NodeAddr` contribution, contradicting this
      same AC's own Postcondition 1 and Postcondition 4 (both correctly state the 24-byte
      selector arithmetic). Fixed to: "not merely the 24-byte key-selector prefix (SVTNID 16 +
      NodeAddr 8)". Class sweep performed across the full story for every remaining
      "16-byte"/"24-byte"/"8-byte"/"32-byte"/"38-byte" occurrence and the hop-2
      `DISCOVERY_RELAY` payload arithmetic (4-byte control header + 8-byte `NodeAddr` +
      4-byte `Sequence` + 2-byte count = 18 bytes before the variable session list, at
      lines ~389-395 and ~769-773): no other miscounted field-group arithmetic found — the
      sole "16-byte" occurrence in the entire story was this AC-005 PC-3 sentence, now fixed;
      all other 8-byte mentions correctly refer to `NodeAddr` alone; Architecture Compliance
      Rules line ~1018 already uses count-free phrasing ("not merely the key-selector prefix")
      and was left unchanged per the orchestrator's explicit scoping; the F-DWSP1-001
      changelog/modified-entry history quotes of the old "24 bytes" wording were left
      unchanged as they document a prior finding's text, not a live claim. Sibling fix on the
      architecture side: `.factory/decisions/S-BL.DISCOVERY-WIRE-rulings.md` (declared input
      #1) was independently corrected by the architect to v1.4 for the same error class —
      `compute-input-hash --update` re-run to pick up that change: `b4e0a5f` → `c321c05`. No
      AC added or removed (still 18); points unchanged (8).
  - date: 2026-07-13
    version: "2.1"
    change: >
      Remediated spec-adversarial pass 1 finding F-DWSP1-001 (HIGH, off-by-8-bytes threshold
      bug, CWE-770/400 class). AC-005 Postcondition 4 stated the router-side ingest guard
      rejects a raw datagram "shorter than 24 bytes (insufficient for the fixed key-selector
      fields)" — that threshold omits the 8-byte HMAC tag prefix and contradicts this story's
      own Decision 1/SEC-DW-01 wire layout (`SVTNID` at raw bytes 8-24, `NodeAddr` at raw bytes
      24-32), the rulings doc's Implementation Constraint 2, and the shipped
      `internal/discovery/discovery.go` layout (`[8]HMACTag | [16]SVTNID | [8]NodeAddr | ...`).
      A literal implementation would leave a 24-31-byte raw-datagram window that passes the
      length guard and then indexes out of bounds pre-authentication — exactly the
      resource-exhaustion/pre-auth-crash class SEC-DW-01 exists to close. Fixed to: "A raw
      datagram shorter than 32 bytes (8-byte HMAC tag + the 24-byte SVTNID/NodeAddr key
      selector) — equivalently, whose post-tag body is shorter than 24 bytes — is rejected
      before any key lookup is attempted." Added an explanatory note directly under AC-005
      clarifying the `body` = post-tag-bytes convention this AC and Decision 1 Implementation
      Constraint 2 both use, and noting the full valid-frame minimum with the SEC-DW-07
      `Sequence` field is 38 bytes (8 tag + 16 + 8 + 4 + 2 count) — but the pre-lookup guard
      only needs raw ≥ 32 / body ≥ 24, since `Sequence`/count are parsed by `decodeBody()` only
      after HMAC verification succeeds. Swept the full story for any other occurrence of the
      wrong 24-byte-raw threshold (test names, Task Breakdown, Edge Cases, File-Change List) —
      none found; AC-005 Postcondition 4 was the sole occurrence. No AC added or removed;
      `acceptance_criteria_count` (18) and `points`/`estimated_points` (8) unchanged — this is a
      correctness fix to an existing postcondition's numeric threshold, not new or removed
      scope. `input-hash` unchanged (`b4e0a5f`) — none of the five declared `inputs:` files
      were touched, story-body-only fix; `compute-input-hash --check` confirms no drift.
      Frontmatter `version` 2.0 → 2.1; new `modified:` entry appended (newest-first, per
      POL-001).
  - date: 2026-07-13
    version: "2.0"
    change: >
      Elaborated from backlog stub (v1.1, draft, 0 ACs) to sprint-ready draft (v2.0, 18 ACs,
      8 points) per architect ruling `.factory/decisions/S-BL.DISCOVERY-WIRE-rulings.md` v1.3
      (all three rulings). Ruling 1 resolves DRIFT-W6TBD-001 (admitted-node HMAC key
      derivation — reuse of the shipped `hmac.DeriveKey` shape, domain-separated via a new
      `HKDFInfoDiscovery` label). Ruling 2 resolves the SVTN-scoped multicast address
      obligation AND surfaces a real spec conflict the stub inherited: the stub's
      `net.ListenMulticastUDP` instruction and `ARCH-03`'s prior "consoles subscribe to
      multicast" sketch both violated the already-ratified DI-004 domain invariant and
      BC-2.03.001's own Invariant 1 — resolved via a router-only-multicast-membership design
      (only the router-mode daemon joins the group; access/console nodes send-only, never
      join). Ruling 3 (added same day, v1.3) adjudicates the hop-2 relay transport the stub
      never scoped: rides the existing `FrameTypeCtl` `control_type=0x03` discriminator
      (`DISCOVERY_RELAY`, already landed in `BC-2.01.008` v1.2), zero `HMACTag`
      (connection-trust boundary matching the `S-7.04-FU-DRAIN-WIRE` DRAIN precedent),
      SVTN-scoped exclude-originator best-effort fan-out, `~1/sec` per-`(SVTNID,NodeAddr)`
      rate cap. One item — fan-out **target resolution** (which live connections belong to a
      given SVTN's admitted nodes) — is a verified, not invented, Forward Obligation: the
      node-identity-to-connection binding it requires does not exist in production code today
      (grep-confirmed: `admission.AdmitNode` has zero production call sites). Task Breakdown
      structured so hop-1 ingest (Tasks 1-4) and hop-2 frame construction (Task 5) are
      independently deliverable; hop-2 fan-out dispatch (Task 6, AC-017/AC-018) is explicitly
      GATED on this Forward Obligation pending human disposition (see Human Gate section).
      Corrects the v1.1 stub's mislabeled citation: the stub's Open Design Obligation #2
      quoted "BC-2.03.001 PC-1 requires 'a SVTN-scoped multicast address is allocated...'" —
      that requirement is BC-2.03.001 **Precondition 3**, not Postcondition 1 (PC-1 governs
      delivery — "the advertisement is multicast to all admitted nodes" — a different clause
      the address-allocation text does not appear in). All prose in this elaboration cites
      Precondition 3 correctly. `bc_traces`/`behavioral_contracts` gains `BC-2.01.008`
      (DISCOVERY_RELAY registry row consumer) alongside the stub's existing `BC-2.03.001`/
      `BC-2.03.002`. `vp_traces`/`verification_properties` gains `VP-080` (SEC-DW-07
      replay-rejection, minted by architect ahead of this elaboration) alongside the stub's
      `VP-044`/`VP-045`. Status stays `draft` (NOT promoted to `ready`) — three items require
      explicit human/PO sign-off before wave scheduling: (1) the SEC-DW-07 monotonic-`Sequence`
      field design (architect already ruled it in, but the ruling itself flags it prominently
      for human gate per the orchestrator's own instruction), (2) the discovery UDP port
      number (`49201` recommended, explicitly a bikeshed placeholder), (3) the fan-out
      target-resolution Forward Obligation's two resolution paths — see Human Gate section.
      Frontmatter conformed to the template-mandated superset keys per the
      `S-BL.CLI-SURFACE-COMPLETION`/`S-BL.LOOPBACK-FULLSTACK` precedent (`epic_id`, `inputs`,
      `input-hash`, `traces_to`, `behavioral_contracts`, `verification_properties`,
      `target_module`, `estimated_days`, `assumption_validations`, `risk_mitigations`).
      `input-hash` computed via `compute-input-hash --update`.
version: "2.13"
phase: 2
epic: E-7
wave: backlog
priority: P1
scope_phase: PE
points: 8
estimated_points: 8
inputs:
  - '.factory/decisions/S-BL.DISCOVERY-WIRE-rulings.md'
  - '.factory/specs/behavioral-contracts/ss-03/BC-2.03.001.md'
  - '.factory/specs/behavioral-contracts/ss-03/BC-2.03.002.md'
  - '.factory/specs/behavioral-contracts/ss-01/BC-2.01.008.md'
  - '.factory/specs/architecture/ARCH-03-routing-engine.md'
input-hash: "8bdbc57"
traces_to: '.factory/decisions/S-BL.DISCOVERY-WIRE-rulings.md'
behavioral_contracts:
  - BC-2.03.001
  - BC-2.03.002
  - BC-2.01.008
verification_properties:
  - VP-044
  - VP-045
  - VP-080
bc_traces:
  - BC-2.03.001
  - BC-2.03.002
  - BC-2.01.008
vp_traces:
  - VP-044
  - VP-045
  - VP-080
subsystems: [session-discovery]
target_module: "internal/discovery"
architecture_modules:
  - internal/discovery
  - internal/routing
  - internal/hmac
  - cmd/switchboard
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: [S-7.02, S-2.02, S-BL.NODE-IDENTIFY-WIRE]
blocks: []
estimated_days: null
assumption_validations: []
risk_mitigations: []
changed_by_rulings: [RULING-W6TB-D, RULING-W6TB-H, S-BL.DISCOVERY-WIRE-rulings]
acceptance_criteria_count: 18
inputDocuments:
  - '.factory/decisions/S-BL.DISCOVERY-WIRE-rulings.md'   # v1.10 — BINDING. All three rulings + Security Consult Addendum (SEC-DW-01..09) + Replay/freshness subsection (now incl. the F-DWSP4-001 restart-liveness amendment) + Decision Log (incl. the v1.6 precision-correction entry, the v1.7 one-token propagation fix to Ruling 3(c)'s trailing prose, the v1.8 Node-local ingest correction retiring `ReceiveAdvertisement`/`TestDiscovery_VP045_SVTNIsolation_MultipleScopes`, the v1.9 story-ready human gate disposition — Ruling 3(f)'s fan-out target resolution resolved to named companion story `S-BL.NODE-IDENTIFY-WIRE`, SEC-DW-07/discovery-port sign-off recorded, `sessions.list` Forward Obligation (d) resolved to `S-BL.SESSIONS-LIST-WIRE` — and the v1.10 Ruling 4 (Task-3 router daemon-lifecycle wiring gap, new Forward Obligation (e), `S-BL.ADMISSION-SYNC-WIRE` named as a follow-on) plus a Ruling 2 addendum (sender-side multicast egress elaboration, sanctioned within Ruling 2's existing scope). Where this story and the ruling appear to diverge, the ruling governs.
  - '.factory/specs/behavioral-contracts/ss-03/BC-2.03.001.md'   # v1.6 — Preconditions 1-3, Postconditions 1-5, Invariants 1-3. Ruling 1/2 amendments already executed by product-owner (Precondition 3 address-derivation note, PC-1 relay-delivery note, PC-2 Sequence field, PC-5 DiscoveryAuthKey derivation); v1.6 carries the F-DWSP4-001 restart-liveness amendment to PC-2's Sequence-field description.
  - '.factory/specs/behavioral-contracts/ss-03/BC-2.03.002.md'   # v1.5 — PC-5 is the postcondition SEC-DW-07/VP-080 protects (staleness-expiry guarantee). PC-1's `sessions.list` RPC-exposure annotation is NOT adjudicated by any of the three rulings — flagged, not solved, in Non-Goals; v1.5 re-points the annotation from PENDING-S-BL.DISCOVERY-WIRE to PENDING-S-BL.SESSIONS-LIST-WIRE per Forward Obligation (d)'s resolution.
  - '.factory/specs/behavioral-contracts/ss-01/BC-2.01.008.md'   # v1.2 — DISCOVERY_RELAY=0x03 registry row (Ruling 3(g), already executed by product-owner); PC-3 4-byte control header + DISCOVERY_RELAY extension note; Invariant 3 (append-only) and Invariant 5/DI-007 (extend-beyond-byte-3 allowance) govern the hop-2 payload layout.
  - '.factory/specs/architecture/ARCH-03-routing-engine.md'   # v1.8 — §Session Discovery. Router-relay model, address derivation, hop-2 relay-transport paragraph, and the superseded-language callout are all already executed (Ruling 2/3), not proposed.
  - '.factory/specs/verification-properties/VP-080.md'   # v1.7 — SEC-DW-07 replay-rejection property, draft lifecycle_status pending this story's wave scoping (draft→active transition is this elaboration's job per the VP's own Lifecycle section); v1.3 carried the restart-liveness Property 5 (forward-acceptance-on-restart) added alongside the F-DWSP4-001 fix, per VP-INDEX v2.43; v1.4 was a housekeeping bump alongside ARCH-07/ARCH-11 gaining the VP-078/079/080 rows; v1.5 was a citation re-pin + input-hash refresh alongside rulings v1.8 (F-DWSP9-001) — no property-substance change (Properties 1-5, Test Scenarios, thresholds all unchanged); v1.6 replaced two stale `TestDiscovery_VP045_SVTNIsolation_MultipleScopes`-as-extant-exemplar citations (Proof Method table, Feasibility Assessment) with the surviving router-side `DiscoveryAuthKeyFor`-admitted test lineage (F-DWSP10-001) — no property-substance change; v1.7 corrected the Source Contract's stale process-status paragraph (it had read "drafted, not yet executed" since the v1.0 mint even though the BC-2.03.001 PC-2 amendment landed at v1.5 and was superseded in place at v1.6) and confirmed textual alignment between the reproduced blockquote and the landed BC text (F-DWSP12-001) — no property-substance change, input-hash unchanged (`5d904d5`).
  - '.factory/specs/verification-properties/VP-044.md'   # v1.2 — PARTIAL (RULING-W6TB-D doctrine); multicast wire delivery (PC-1/PC-3/PC-4) is the gap this story closes.
  - '.factory/specs/verification-properties/VP-045.md'   # v1.4 — PARTIAL (unchanged); real-socket PC-3 aggregation over UDP multicast is the gap this story closes. v1.4 corrects a stale supporting-evidence citation only (`TestDiscovery_VP045_SVTNIsolation_MultipleScopes`, retired by rulings v1.8 F-DWSP8-001) — no change to status or the gap this VP names.
  - '.factory/stories/S-7.02-session-discovery.md'   # MERGED PR #55. In-process registry model this story replaces at the wire boundary only — trigger model and payload semantics (BC-2.03.001's non-deferred clauses) are unchanged.
  - '.factory/stories/S-7.04-FU-DRAIN-WIRE.md'   # DELIVERED PR #120 @ f73676d. Direct precedent for the hop-2 relay mechanics this story's Ruling 3 reuses verbatim in shape: control_type discriminator on FrameTypeCtl, sendMap.Range best-effort non-blocking fan-out, zero-HMACTag connection-trust boundary, register-before-serve wiring pattern.
  - '.factory/stories/S-BL.CLI-SURFACE-COMPLETION.md'   # v2.9, DELIVERED PR #122. Structural precedent for this elaboration's section architecture (Adjudicated Design Decisions, File-Change List, Token Budget Estimate, POL-005 Delivery Plan Note, Anchors Consumed).
backlog_origin:
  source: "RULING-W6TB-D (Wave-6 Tranche-B planning) — real-socket wire transport deferred from S-7.02"
  deferred_from: S-7.02
  drift_items_consumed:
    - DRIFT-W6TBD-001
  notes: >
    Backlog stub v1.0/v1.1 deferred two Open Design Obligations blocking scheduling:
    admitted-node HMAC key derivation (DRIFT-W6TBD-001) and SVTN-scoped multicast address
    allocation. Both resolved by architect ruling `S-BL.DISCOVERY-WIRE-rulings.md`
    (v1.0-v1.2, 2026-07-13), which also surfaced and resolved a third, undiscovered
    obligation — the stub's `net.ListenMulticastUDP` instruction and ARCH-03's prior
    "consoles subscribe to multicast" sketch both conflicted with the already-ratified DI-004
    domain invariant. Ruling 3 (v1.3, same day) then adjudicated the resulting hop-2 relay
    transport this conflict's resolution requires, since story-writer decomposition would
    otherwise be blocked on an undecided delivery mechanism. This elaboration transcribes all
    three rulings into sprint-ready ACs; the hop-2 fan-out **target-resolution** item (Ruling
    3(f)) remains a verified Forward Obligation requiring human disposition before wave
    scheduling — see Human Gate section below.
---

# S-BL.DISCOVERY-WIRE: Discovery Wire Boundary — UDP Multicast I/O, Admitted-Node HMAC Keys, Multicast Address Allocation, Hop-2 Relay Dispatch

> **Status note:** All three Open Design Obligations (two from the v1.1 stub, one surfaced
> mid-ruling) are ADJUDICATED (`.factory/decisions/S-BL.DISCOVERY-WIRE-rulings.md` v1.10,
> 2026-07-13). This elaboration is sprint-ready in content but stays `status: draft` — it has
> NOT been promoted to `ready` — because three items require explicit human/PO sign-off before
> wave scheduling (see **Human Gate — Story-Ready Sign-off Required** below). Do not implement
> from the v1.1 stub's "Open Design Obligations"/"Scope"/"Scope Constraints" sections — they are
> superseded below by "Adjudicated Design Decisions."

## Narrative

- **As an** access node publishing tmux sessions on an SVTN
- **I want to** advertise my presence and session state over a real UDP-multicast wire channel,
  authenticated with admitted-node-scoped keys, and have the router relay validated
  advertisements to every other admitted node on the SVTN
- **So that** consoles discover sessions with zero manual configuration (no hostnames, no IP
  addresses) over the network — not just inside a single in-process test harness — while an
  observer who can capture one valid advertisement off the multicast segment cannot replay it
  indefinitely to keep a stale or revoked session looking perpetually alive

## Context

Ruling W6TB-D (`.factory/decisions/RULING-W6TB-D-discovery-scope.md`) established that S-7.02
(MERGED PR #55) delivers an in-process registry model for session discovery — the trigger model
and payload semantics of BC-2.03.001 are fully verified there. Real multicast wire I/O and
admitted-node HMAC key derivation were deferred to this story.

**Corrected citation (fixes a v1.1 stub error).** The v1.1 stub's Open Design Obligation #2 read:
"BC-2.03.001 PC-1 requires 'a SVTN-scoped multicast address is allocated for the SVTN's
discovery channel.'" That quotation is accurate but its BC-clause label was wrong — the
address-allocation requirement is BC-2.03.001 **Precondition 3**, not Postcondition 1.
Postcondition 1 governs a different clause entirely ("the advertisement is multicast to all
admitted nodes on the SVTN" — a delivery guarantee, not an allocation precondition). This
elaboration and every downstream artifact cite Precondition 3 correctly; the stub file itself is
superseded by this version.

**A real spec conflict surfaced during adjudication, not invented by this elaboration.** The
stub's scope item 3 ("replace ... with `net.ListenMulticastUDP` dispatch goroutine") is silent on
*who* calls `ListenMulticastUDP`. `ARCH-03`'s prior discovery sketch ("Access nodes send
`PRESENCE_ADV` frames to a well-known SVTN multicast address. Consoles subscribe to multicast...")
read literally as direct peer-to-peer IP multicast — access nodes and consoles as co-members of
one OS-level multicast group. That reading directly conflicts with the already-ratified **DI-004**
domain invariant ("no direct node-to-node communication") and with BC-2.03.001's own **Invariant
1** ("Advertisements flow node-to-router-to-node via the SVTN; no direct node-to-node
multicast"). Ruling 2 resolved this in favor of the BC/DI-004 (already ratified) over the ARCH-03
sketch (explicitly marked provisional): only the router-mode daemon ever joins the multicast
group; access nodes and consoles send-only or receive-via-relay, never direct peers. `ARCH-03`
§"Session Discovery" is now at v1.8 with the reconciled design live, including an explicit
"Superseded language" callout naming exactly what the old sketch said and why it was wrong.

**Ruling 3 (added same day, v1.3) closes the resulting hop-2 gap.** Once advertisements are
router-relayed rather than peer-multicast, a router-to-node relay transport has to exist and had
never been specified. Ruling 3 adjudicated it directly, since leaving it to story-writer
decomposition would mean scheduling this story against an undecided delivery mechanism.

## Previous Story Intelligence (MANDATORY)

| Predecessor | Lesson carried forward |
|-------------|--------------------------|
| `S-7.02` (MERGED PR #55, `c54a8ad`) | Ships the in-process registry model, trigger conditions (state-change/heartbeat/on-demand), and `AdvertisementPayload`/`encodeBody`/`decodeBody` wire-body layout this story extends rather than replaces. The `advertisementKey` function (`internal/discovery/discovery.go`) is the placeholder this story retires. `ReceiveAdvertisement`'s HMAC-first ordering (`discovery.go`) is RULING-W6TB-H's shipped precedent — **retired, not preserved, by Ruling 1 point 3 as corrected by F-DWSP8-001**: it cannot survive `advertisementKey`'s deletion (three call sites, not one) and a node has no key to derive for an arbitrary sender. A new node-side relay-ingest function takes over its former role (relocated `ErrSVTNMismatch` guard, no per-frame HMAC) — see AC-007. |
| `RULING-W6TB-H` (S-7.02 remediation) | Established the HMAC-first-before-SVTN-check ordering and the `ErrInvalidHMACTag`-before-`ErrSVTNMismatch` sentinel discipline this story's router-side ingest path (new code) must independently re-establish — the two paths are structurally parallel, not the same code. |
| `S-7.04-FU-DRAIN-WIRE` (DELIVERED PR #120 @ `f73676d`) | Direct, shipped precedent for every hop-2 mechanic Ruling 3 specifies: `control_type` discriminator riding `FrameTypeCtl` rather than a new outer `FrameType` byte; zero `HMACTag` on router-to-node control frames (connection-trust boundary, not per-frame auth); `sendMap.Range` best-effort non-blocking fan-out (`select { case nc.send <- frame: default: }`); register-before-serve wiring (`wireXHandlers` called from `runRouter` before `serveMgmtServer`). Reuse these shapes verbatim — do not re-derive them. |
| `S-BL.CLI-SURFACE-COMPLETION` (DELIVERED PR #122 @ `1f25677`) | Direct precedent for this story's own elaboration discipline: "story-writer's job is transcription, not re-derivation" when a binding architect ruling exists (applied identically here to `S-BL.DISCOVERY-WIRE-rulings.md`). Template-mandated frontmatter superset keys, Forward Obligations table shape, Token Budget Estimate section shape, and the POL-005 Delivery Plan Note text are all reused verbatim from that story. |
| `admission.RegisterKey` / `hmac.DeriveKey` (S-2.02, S-1.03) | Direct precedent for Ruling 1's key-reuse decision: `AdmittedKey.FrameAuthKey := hmac.DeriveKey(nodeAdmissionPubkey, svtnID)` is the shipped, audited derivation this story's `DiscoveryAuthKey` domain-separates from (same inputs, distinct HKDF info label) rather than re-inventing. |
| `BC-2.05.005 PC-3` / `FailureCounter` (S-W3.05) | Direct precedent for SEC-DW-03's visibility-only reuse: a per-source `FailureCounter` (threshold=5/60s) already exists and is wired to every router-mode `Router`; this story reuses it for discovery HMAC-failure visibility but explicitly does NOT let it gate on the attacker-controlled pre-auth `NodeAddr` field. |

## Adjudicated Design Decisions

Transcribed from `.factory/decisions/S-BL.DISCOVERY-WIRE-rulings.md` v1.10 (binding — all factual
claims in that ruling are grep/read-verified against `develop@1f25677`). Where this story and the
rulings doc appear to diverge, the rulings doc governs. Each entry below carries the load-bearing
constraints inline — the implementer should not need to re-open the rulings doc for the common
path.

### Decision 1 (Ruling 1) — Admitted-node HMAC key derivation: reuse `hmac.DeriveKey`'s shape, domain-separated

Retire `advertisementKey(svtnID) = svtnID` — a pure function of a value that transits in
cleartext, so any observer can compute it. Replace it with `DiscoveryAuthKey :=
hmac.DeriveDiscoveryKey(nodeAdmissionPubkey, svtnID)` — the identical HKDF-SHA256 construction
(ADR-001) and the identical `(nodeAdmissionPubkey, svtnID)` inputs `admission.RegisterKey` already
uses for the session-data `FrameAuthKey`, but with a distinct HKDF info label
(`HKDFInfoDiscovery = "switchboard-discovery-auth"`, vs. the existing `HKDFInfo =
"switchboard-frame-auth"`) so the two derived keys are cryptographically independent (SEC-DW-06 —
domain-separates the new, more-exposed UDP ingest surface from the existing TCP-handshake-gated
one). No new KDF primitive; `hkdfSHA256` (stdlib `crypto/hmac`+`crypto/sha256`, no external
dependency) is unchanged and shared by both call sites.

- **A per-SVTN group secret was considered and rejected.** It would require new state (an
  `SVTNManager`-held secret, a new post-admission distribution step) with no shipped precedent —
  materially larger and riskier than the finding requires, and unnecessary once Ruling 2
  establishes that advertisements are router-relayed (one verifier, the router, not many peers
  each needing to verify every other peer).
- **Verification happens exclusively at the router** — the "first router" DI-006 gate. Access and
  console nodes never independently look up or re-verify another node's `DiscoveryAuthKey`; they
  receive already-authenticated advertisements only via the router's hop-2 relay.
- **Key location: nowhere new.** `DiscoveryAuthKey` is computed on demand from
  `AdmittedKeySet.Lookup`'s already-returned `PublicKey` field — not cached as a new field on
  `AdmittedKey` (unlike `FrameAuthKey`, precomputed once at `RegisterKey` time). Fully additive to
  `internal/admission`; that package is not touched.
- **New lookup surface:** `(*routing.Router).DiscoveryAuthKeyFor(svtnID [16]byte, nodeAddr [8]byte)
  ([hmac.KeySize]byte, bool)` in `internal/routing/advertisement_hmac.go` — a thin wrapper over
  `admittedKeySet.Lookup` + `hmac.DeriveDiscoveryKey`, same shape as the file's existing
  `ComputeAdvertisementHMAC`/`VerifyAdvertisementHMAC` pass-through pattern. Preserves the ARCH-08
  position-14 boundary: `internal/discovery` may import ONLY `internal/routing`.
- **Sender-side symmetric wrapper:** `routing.DeriveDiscoveryKey(pubkey []byte, svtnID [16]byte)
  [hmac.KeySize]byte` — lets an access node compute its own `DiscoveryAuthKey` locally (both
  inputs are locally known: own public key, own SVTN ID) without querying the router, and without
  `internal/discovery` importing `internal/hmac` directly.
- **`Discovery.ReceiveAdvertisement` is RETIRED (deleted), not preserved — corrected by F-DWSP8-001.**
  It cannot compile once `advertisementKey` is deleted (one of three call sites, not the one this
  AC's scoping implied), and even patched, a node has no key to derive for an arbitrary sender
  (BC-2.03.001 v1.6 PC-5's own rule). A new node-side relay-ingest function replaces it on the
  hop-2 path: no per-frame HMAC (trust = the admitted connection, AC-015), `ErrSVTNMismatch`
  relocated to a direct `OuterHeader.SVTNID` vs. `d.cfg.LocalSVTNID` equality check.
  `TestDiscovery_VP045_SVTNIsolation_MultipleScopes` is retired, not preserved — see AC-007 and the
  File-Change List for the full ten-test disposition. Full rationale:
  `S-BL.DISCOVERY-WIRE-rulings.md` v1.10, "Node-local ingest correction."

**Security-hardening constraints (Security Consult Addendum, SEC-DW-01/02/03/04/05, all ADOPTED as
MANDATORY except SEC-DW-05's optional dummy-HMAC hardening):**

1. **SEC-DW-01 (HIGH, MANDATORY).** Router-side ingest MUST extract the key-selector fields
   (`SVTNID` at raw bytes 8-24/`body[0:16]`, `NodeAddr` at raw bytes 24-32/`body[16:24]`) via
   fixed-offset direct byte-slice indexing and MUST NOT call the full `decodeBody()` (which walks
   the variable-length, attacker-controlled session-entry list) until *after* HMAC verification
   succeeds. HMAC verification itself still runs over the **complete raw body bytes** — only the
   *decode/parse* step is deferred, not the *coverage* of the MAC.
2. **SEC-DW-02 (MED).** Bounded, fixed-size UDP read buffer sized to realistic legitimate usage —
   not the 65,507-byte UDP/IP theoretical maximum. `maxSessionsPerAdvertisement` (currently `1024`,
   sized against the old TCP/length-prefixed assumption) MUST be re-derived at implementation time
   from realistic tmux-sessions-per-access-node scale.
3. **SEC-DW-03 (MED).** Two-layer rate defense at the socket-read loop: (a) an **aggregate**
   (not per-source) token-bucket cap — a per-source cap keyed on the pre-auth, attacker-controlled
   `NodeAddr` is trivially defeated by identity rotation; (b) separately, reuse the existing
   per-source `FailureCounter` (threshold=5/60s, BC-2.05.005 PC-3) for operator **visibility**
   only — never a per-`NodeAddr` admission or rate gate.
4. **SEC-DW-04 (MED).** Discovery HMAC-failure logging MUST be rate-limited/counter-based via
   `FailureCounter`'s own threshold-crossing emission — NOT BC-2.05.008's per-packet TCP logging
   policy (an attacker reaching the discovery multicast group pays no per-attempt cost the way a
   TCP handshake does).
5. **SEC-DW-05 (LOW/INFO, MUST clause).** No wire-visible accept/reject differential. Advertisements
   are one-way fire-and-forget UDP (no ack, so no response-content oracle by construction);
   unifying "lookup-miss" with "HMAC-mismatch" into the single `ErrInvalidHMACTag` sentinel
   (constraint 1 above) is correct and required. Processing-time symmetry between the two
   rejection paths is the residual concern; dummy-HMAC-on-lookup-miss is optional hardening, not
   required for this story.
6. **SEC-DW-06.** `HKDFInfoDiscovery` domain-separation, as described above — ADOPTED, no
   counter-rationale found.

### Decision 2 (Ruling 2) — SVTN-scoped multicast address: administratively-scoped IPv4, router-only listener, relay (not peer) delivery

**(a) Router-mediated relay, not raw peer-to-peer IP multicast.** This is not a new design choice
— it is the literal requirement of DI-004 and BC-2.03.001 Invariant 1. Only the router-mode
daemon calls `net.ListenMulticastUDP`; access/console nodes send addressed to the multicast group
but never join it, and never receive from it directly. Standard IP multicast semantics do not
require a *sender* to join a group — only *receivers* need group membership — giving a design that
is simultaneously "real multicast" and DI-004-compliant.

- Senders set the outbound multicast **TTL to 1** explicitly (SEC-DW-08) — network-layer
  containment as defense-in-depth alongside the application-layer control (only the router process
  joins the group).
- The router authenticates each inbound datagram via Ruling 1's `DiscoveryAuthKeyFor` lookup
  (HMAC-first, fail-closed — the DI-006 "first router" gate) and, on success, relays the
  advertisement onward per Decision 3 below.
- **A multicast-address collision between two different SVTNs is harmless, not merely unlikely.**
  Even if two SVTNs' derived addresses collided, a router serving SVTN-A receiving a stray SVTN-B
  datagram would attempt `DiscoveryAuthKeyFor(payload.SVTNID, payload.NodeAddr)` against SVTN-A's
  admitted state and fail — dropped fail-closed exactly as an unauthenticated frame would be. HMAC
  authentication, not address uniqueness, is the actual security boundary (SEC-DW-08): the
  `239.0.0.0/8` range is addressing hygiene/routing-efficiency, never a security control, and this
  holds regardless of the actual multicast-routing scope realized in any given deployment.

**(b) Address derivation.** `addr = 239.h0.h1.h2` where `h = SHA-256(svtnID)` (first three bytes).
IPv4 `239.0.0.0/8` — RFC 2365 "administratively scoped," explicitly intended for private,
non-globally-routed applications like this one. Deterministic and static — no allocation
bookkeeping, no release step on `admin.svtn.destroy`. A raw truncation of `svtnID` was considered
and rejected purely for domain-separation hygiene (the hash step avoids any accidental structural
correlation between the SVTN ID's bit layout and the derived address); `svtnID` already transits
in cleartext in every outer header, so this is not a secrecy requirement.

**(c) Port.** A single fixed named constant in `internal/discovery` (parallel to the existing
`HeartbeatInterval` constant) in the IANA dynamic/private range (49152–65535).
**Adjudicated: `49201`** — arbitrary, unregistered, adopted by human gate disposition 2026-07-14 (no
longer a placeholder; see Human Gate item 2). One port suffices for all SVTNs because the group
*address*, not the port, provides SVTN scoping.

**(d) IPv6 explicitly out of scope.** No IPv6 data-plane precedent exists anywhere in this
codebase (only mgmt-plane loopback authorization references exist). Flagged as a named forward
obligation for a future story, not a silent gap.

**(e) Loopback testability.** `net.ListenMulticastUDP` works on loopback on both macOS and Linux,
but the loopback interface name differs (`lo0` vs `lo`, per this project's own B13 lesson —
platform-specific behavior requires platform-specific testing) and must be resolved via
`net.InterfaceByName`. `internal/testenv.NewLoopback` is a VP-042-scoped compile-shim and is NOT a
fit for this — do not extend it. A new, purpose-built helper (e.g.
`testenv.MulticastLoopbackInterface(t testing.TB) *net.Interface`) is new test infrastructure this
story's scope must count.

### Decision 3 (Ruling 3) — Hop-2 relay transport: `control_type=0x03`, connection-trust boundary, exclude-originator fan-out, ~1/sec rate cap

**(a) Transport.** Rides the existing `FrameTypeCtl` (`0x03`) outer frame with a new
`control_type = 0x03` (`DISCOVERY_RELAY`) payload discriminator — **not** a new outer `FrameType`
byte (the 6-slot canonical enum is exhausted; `FrameTypePEConnect = 0x06` took the last slot, and
`FrameType.Valid()` hard-rejects anything above it). This is the identical pattern
`S-7.04-FU-DRAIN-WIRE`'s DRAIN broadcast already ships (`control_type=0x01`). `BC-2.01.008` is the
schema home; its Postcondition 2 registry row for `DISCOVERY_RELAY=0x03` is **already landed**
(v1.2, executed by product-owner) — no story-writer or implementer action needed there.

**(b) Connection-trust boundary — zero `HMACTag`, matching the DRAIN precedent exactly.** The
relay frame's `HMACTag` is the zero value; authentication for this hop is the already-completed
Tier-1-admitted, already-open TCP connection itself, not a fresh per-frame HMAC (the router does
not sign outbound control traffic to nodes anywhere in the shipped codebase). This does NOT dilute
SEC-DW-08 — hop 1 (UDP multicast ingest, reachable by anyone on the LAN segment) uses HMAC because
there is no other authentication available there; hop 2 travels over a connection already mutually
authenticated at admission time. Stating both boundaries explicitly is what keeps SEC-DW-08
undiluted.

**(c) Payload — re-serialized, not a raw relay of hop-1's UDP bytes.** Hop-1's HMAC is scoped to
the wire path/key that produced it and has no meaning to a receiving node — forwarding it verbatim
would misleadingly imply the receiving node could re-verify it, which BC-2.03.001 PC-5 (v1.6,
already landed) explicitly forecloses. Layout, respecting `BC-2.01.008` PC-3's fixed 4-byte
control header and its extend-beyond-byte-3 allowance (Invariant 5/DI-007):

```
byte[0]     control_type = 0x03 (DISCOVERY_RELAY)
byte[1]     version = 0x01
byte[2:4]   reserved = 0x0000
byte[4:12]  NodeAddr    — the ORIGINATING access node's 8-byte address
byte[12:20] Sequence    — uint64 BE (epoch-qualified, F-DWSP4-001), the same value hop-1 accepted (SEC-DW-07)
byte[20:22] session count — uint16 BE
byte[22:]   sessions...   — internal/discovery's existing per-session encoding
```

`SVTNID` is deliberately NOT repeated inside this payload — the relay frame's own
`OuterHeader.SVTNID` field already carries the SVTN scope, matching how every other frame type on
this wire already scopes itself.

**(d) Fan-out — SVTN-scoped, exclude-originator, best-effort non-blocking (DRAIN's `sendMap.Range`
pattern), NOT `routing.SplitHorizon.Forward`/`FrameArrivalHandler.OnFrameArrival`.**
`SplitHorizon`/`OnFrameArrival` were evaluated as a real alternative and rejected for two concrete
reasons: (1) their drop-cache half exists to suppress inter-router relay loops in a multi-router
mesh; `BC-2.01.008` Invariant 2 confirms no inter-router relay path is implemented, and this
design has exactly one router terminating a given SVTN's multicast segment (star topology) — no
loop to suppress, and hop-1's own SEC-DW-07 sequence-gate already guarantees freshness; (2)
`SplitHorizon.Forward`'s exclusion parameter is `arrivalIface InterfaceID` — the interface the
frame arrived ON — but hop-1's advertisement arrives via the UDP multicast socket, not any
`netingress`-accepted TCP interface, so there is no real `arrivalIface` to supply without
inventing a fictitious sentinel. What hop-2 actually needs to exclude is "the originating access
node's own admitted TCP connection" — a `NodeAddr`-keyed exclusion, not an `InterfaceID`-keyed
one. Fan-out therefore follows DRAIN's own dispatch shape directly: a purpose-built closure (same
placement as the DRAIN observer, inline in `runRouter`) iterates the live connections of nodes
admitted to the advertisement's SVTN, skips the originating `NodeAddr`, and for each remaining
target does the same `select { case nc.send <- relayFrame: default: }` best-effort non-blocking
send DRAIN already uses.

**(e) SEC-DW-09 rate cap — `~1/sec` per `(SVTNID, NodeAddr)`, silent-drop-first plus a non-gating
visibility counter.** Enforced at the relay-**dispatch** decision point (distinct from Ruling 1's
ingest-side SEC-DW-03 token bucket — a different enforcement point serving a different purpose:
this one protects against amplification by a misbehaving-but-legitimately-admitted sender, HMAC-
valid and sequence-increasing, passing every hop-1 check). Keyed by the *originating*
`(SVTNID, NodeAddr)`: an advertisement arriving faster than ~1/sec from the same sender still
updates the router's own local registry/discard-map state (SEC-DW-07 correctness unaffected) but
is NOT relayed on that excess arrival. Silent drop is the actual backstop; an optional
`FailureCounter`-shaped counter is visibility-only, never a gate.

**(f) Fan-out TARGET RESOLUTION is a genuine, verified Forward Obligation — NOT resolved here.**
Determining "which of the router's live connections currently belong to nodes admitted to SVTN X"
requires binding node identity (`NodeAddr`) to a live connection's `InterfaceID`/`nodeConn`. This
binding **does not exist in production code today** — grep-confirmed, not assumed:
`routing.ForwardingEntry` carries no `InterfaceID` field; `SVTNRoute` never dispatches bytes
anywhere (`_ = payload`, `_ = entry // available for future use` — ordinary DATA-plane relay is
itself still a validation-only stub); `cmd/switchboard/mgmt_wire.go`'s `sendMap` is keyed purely by
`routing.InterfaceID` in accept order, no `NodeAddr` recorded anywhere in `onAccept`;
`admission.AdmitNode` (the handshake that would reveal a connecting node's identity) has **zero
production call sites** anywhere in `cmd/` or `internal/` — the only caller is the test harness in
`internal/testenv/testenv.go`. This is the same gap this project has already
named once, generically, as `FO-DRAIN-WIRE-002` — no successor story exists yet to resolve it. See
**Forward Obligations** and **Human Gate** below for the two resolution paths and why this
elaboration does not pick one unilaterally.

**(g) `BC-2.01.008` registry row — already landed.** v1.2, executed by product-owner between
ruling v1.2 and v1.3. No story-writer or implementer action.

**(h) `BC-2.03.001` needs no further amendment for hop-2 — confirmed, not assumed.** v1.6's PC-5
sentence ("access and console nodes never independently look up or re-verify another node's
`DiscoveryAuthKey`; they receive already-authenticated advertisements via the router's relay over
their own admitted connection") already describes the trust model this ruling formalizes.

## Design Constraint: Router-Mode Discovery Wiring

Two new pieces of router-mode-exclusive wiring, both following the shipped
`wireMetricsHandlers`/`wireRouterControlHandlers`/DRAIN-observer register-before-serve precedent
in `runRouter` (`cmd/switchboard/mgmt_wire.go`):

1. **Discovery multicast listener.** Binds `net.ListenMulticastUDP` on the SVTN-derived group
   address(es) it currently serves, joins the group, and dispatches inbound datagrams into
   `internal/discovery`'s new router-side ingest path (SEC-DW-01..07). Router-mode-exclusive —
   `runAccess`/`runConsole`/`runControl` never call it.
2. **Relay-dispatch closure.** On an accept+relay verdict from the ingest path, assembles the
   `DISCOVERY_RELAY` frame (Decision 3c) and fans it out per Decision 3(d)/(e) — **the live-wiring
   half of this is Task 6, explicitly GATED per the Human Gate section below.**

`internal/discovery`'s router-side ingest path returns an accept/relay **decision** to its caller
rather than performing any relay I/O itself (per the rulings doc touch-list) — this is exactly
what keeps hop-1 (the ingest decision) independently testable and deliverable without the hop-2
dispatch mechanism existing yet.

## Acceptance Criteria

### AC-001 — Router-mode-exclusive multicast group membership (traces to BC-2.03.001 Postcondition 1, Invariant 1)

**BC Anchor:** BC-2.03.001 Postcondition 1 (delivery-mechanism note), Invariant 1 (DI-004)

**Scope note (Ruling 4, v1.10, 2026-07-15):** Postcondition 1 describes this story's target
behavior; router-mode daemon-lifecycle wiring is NOT yet live in production. `wireDiscoveryListener`
is fully implemented and independently tested at function level (this AC's own test, below) but is
not called from `runRouter`. Verified reason: the router process has no source of "which SVTN(s) am
I serving" — `admission.AdmittedKeySet` has no SVTN-enumeration method, and the only production
`RegisterKey` caller runs in a separate, disconnected control-mode OS process. See Forward
Obligation (e) and `S-BL.DISCOVERY-WIRE-rulings.md` v1.10 Ruling 4 for the full adjudication and the
new follow-on story (`S-BL.ADMISSION-SYNC-WIRE`, working name) that will complete this wiring. This
AC's test verifies the function in isolation; it does not, and given the above currently cannot,
verify daemon-lifecycle behavior.

**Postconditions:**

1. Only the router-mode daemon calls `net.ListenMulticastUDP` and joins the SVTN-scoped multicast
   group on its LAN-facing interface(s).
2. `runAccess`, `runConsole`, and `runControl` never join any multicast group and never receive
   advertisements directly from another node's socket.

**Test name:** `TestRunRouter_DiscoveryListener_JoinsGroup_RouterModeOnly`
**Test level:** integration
**Test file:** `cmd/switchboard/discovery_wire_test.go` (new)

---

### AC-002 — Multicast address derivation: static, deterministic, SVTN-scoped (traces to BC-2.03.001 Precondition 3)

**BC Anchor:** BC-2.03.001 Precondition 3

**Postconditions:**

1. `MulticastAddrFor(svtnID [16]byte) net.IP` returns `239.h0.h1.h2`, where `h0..h2` are the first
   three bytes of SHA-256(svtnID) — deterministic, computable independently by every admitted node
   and the router with no coordination step.
2. The address is static for the SVTN's lifetime — no allocation bookkeeping, no release step on
   `admin.svtn.destroy`.
3. The same SVTN ID always produces the same address across repeated calls and across processes.

**Test name:** `TestMulticastAddrFor_Deterministic_SHA256Derived`
**Test level:** unit
**Test file:** `internal/discovery/discovery_test.go` (extended)

---

### AC-003 — Sender-side transmission: TTL=1, no group membership required (traces to BC-2.03.001 Postcondition 1)

**BC Anchor:** BC-2.03.001 Postcondition 1 (delivery-mechanism note); SEC-DW-08

**Precondition:** An access node has an outbound advertisement ready per BC-2.03.001's existing
trigger model (state change / heartbeat / on-demand, all unchanged from S-7.02).

**Postconditions:**

1. The access node's `Run()`/`Advertise` path sends to the SVTN-derived multicast address once per
   UP+multicast-capable local interface (`net.ListenUDP` + `WriteToUDP`, each pinned via `setsockopt
   IP_MULTICAST_IF`) — no `net.ListenMulticastUDP`, no group join, on any interface. Elaborated from
   a single-send design during Task 3's Green step (multi-homed hosts do not reliably route
   `239.0.0.0/8` to every interface a peer may be listening on); sanctioned as within Ruling 2's
   scope, not a new decision — see `S-BL.DISCOVERY-WIRE-rulings.md` v1.10 Ruling 2 addendum
   ("sender-side multicast egress elaboration").
2. The outbound socket's multicast TTL is explicitly set to 1 before the first send.
3. The access node's only target-address knowledge is the deterministic, SVTN-derived group
   address — structurally identical to how it already knows the router's `cfg.ListenAddr` for the
   TCP data plane, just multicast-addressed instead of unicast-addressed.

**Test name:** `TestDiscovery_Advertise_WriteToMulticast_TTL1_NoGroupJoin`
**Test level:** integration (loopback, via `testenv.MulticastLoopbackInterface`)
**Test file:** `internal/discovery/discovery_test.go` (extended)

---

### AC-004 — `DiscoveryAuthKey` derivation: domain-separated from `FrameAuthKey` (traces to BC-2.03.001 Postcondition 5)

**BC Anchor:** BC-2.03.001 Postcondition 5; SEC-DW-06

**Postconditions:**

1. `hmac.DeriveDiscoveryKey(nodeAdmissionPubkey, svtnID)` computes HKDF-SHA256 over the same
   `(nodeAdmissionPubkey, svtnID)` inputs `DeriveKey` uses, with a distinct info label
   `HKDFInfoDiscovery = "switchboard-discovery-auth"` (vs. `HKDFInfo = "switchboard-frame-auth"`).
2. For the same `(nodeAdmissionPubkey, svtnID)` pair, `DeriveDiscoveryKey`'s output differs from
   `DeriveKey`'s output — the two keys are cryptographically independent.
3. `(*routing.Router).DiscoveryAuthKeyFor(svtnID, nodeAddr)` returns `(key, true)` when
   `admittedKeySet.Lookup(svtnID, nodeAddr)` succeeds, and `(zero, false)` otherwise — a thin,
   read-only wrapper adding no new mutable state.
4. `routing.DeriveDiscoveryKey(pubkey, svtnID)` (sender-side symmetric wrapper) produces the
   identical output `DiscoveryAuthKeyFor` would compute for the same node's own admitted key,
   letting a sending access node derive its key locally without querying the router.
5. `advertisementKey(svtnID [16]byte) [16]byte` is deleted; no call site references it.

**Qualifying note (F-DWSP8-001):** `advertisementKey` had three call sites pre-deletion — `Encode`,
`Decode`, and `Discovery.ReceiveAdvertisement` — not only the two sites this AC's Test file list
implies. `ReceiveAdvertisement` is retired (deleted), not merely updated to a new key-derivation
call, per rulings v1.10's Node-local ingest correction. "No call site references it" remains true
post-deletion; no postcondition text change needed, scope clarification only.

**Test names:** `TestDeriveDiscoveryKey_DomainSeparatedFromFrameAuthKey`,
`TestDiscoveryAuthKeyFor_LookupSuccessAndMiss`, `TestDeriveDiscoveryKey_SenderRouterAgree`
**Test level:** unit
**Test file:** `internal/hmac/hmac_test.go` (extended), `internal/routing/advertisement_hmac_test.go` (extended)

---

### AC-005 — Fixed-offset key-selector extraction precedes full body decode (traces to BC-2.03.001 Postcondition 5)

**BC Anchor:** BC-2.03.001 Postcondition 5; SEC-DW-01 (HIGH, MANDATORY)

**Postconditions:**

1. The router-side ingest path extracts `SVTNID` from raw bytes `body[0:16]` and `NodeAddr` from
   raw bytes `body[16:24]` via direct byte-slice indexing — NOT via a call to `decodeBody()` — to
   select the verification key.
2. `decodeBody()` (which walks the variable-length, attacker-controlled session-entry list) is
   never invoked before HMAC verification succeeds.
3. The HMAC computation itself covers the **complete raw body bytes**, not merely the 24-byte
   key-selector prefix (SVTNID 16 + NodeAddr 8) — a forger cannot leave `SVTNID`/`NodeAddr`
   untouched while corrupting the session list beneath an otherwise-valid tag.
4. A raw datagram shorter than 32 bytes (8-byte HMAC tag + the 24-byte SVTNID/NodeAddr key
   selector) — equivalently, whose post-tag body is shorter than 24 bytes — is rejected before
   any key lookup is attempted.

**Note (F-DWSP1-001 fix context, widths updated per F-DWSP4-001/rulings v1.5):** the wire layout is
`[8]HMACTag | [16]SVTNID | [8]NodeAddr | [8]Sequence | [2]count | sessions...` (raw offsets),
matching the shipped `internal/discovery/discovery.go` layout and its own guard comment. `body`
throughout this AC (and Decision 1 Implementation Constraint 2's `body[0:16]`/`body[16:24]`
notation) means the bytes *after* the 8-byte HMAC tag — so `SVTNID`/`NodeAddr` sit at raw bytes
8-24/24-32, not 0-24. With SEC-DW-07's `Sequence` field now widened to `uint64` (epoch-qualified,
F-DWSP4-001 — see AC-010 postcondition 6 and EC-010), the full valid-frame minimum is 42 bytes
(8 tag + 16 SVTNID + 8 NodeAddr + 8 Sequence + 2 count) — but the pre-lookup key-selector guard
in postcondition 4 only needs to enforce raw ≥ 32 / body ≥ 24 (UNCHANGED — this guard sits upstream
of the `Sequence` field and is unaffected by its widening), since `Sequence` and `count` are
parsed by `decodeBody()` *after* HMAC verification succeeds (postcondition 2), not before.

**Test names:** `TestRouterIngest_KeySelectorExtraction_FixedOffset_NoFullDecodeBeforeAuth`,
`TestRouterIngest_HMACCoversFullBody_TamperInSessionListDetected`
**Test level:** unit
**Test file:** `internal/discovery/discovery_wire_test.go` (new)

---

### AC-006 — HMAC-first fail-closed verification with unified reject sentinel (traces to BC-2.03.001 Postcondition 5)

**BC Anchor:** BC-2.03.001 Postcondition 5; SEC-DW-01, SEC-DW-05

**Postconditions:**

1. A lookup-miss (`DiscoveryAuthKeyFor` returns `ok=false` — unknown `NodeAddr` or wrong SVTN) and
   an HMAC-tag mismatch (known `NodeAddr`, wrong key) both resolve to the identical
   `ErrInvalidHMACTag` sentinel, with no distinguishing return value, log line, or other externally
   observable signal.
2. No datagram is relayed, and no discovery registry state is mutated, on either rejection path.
3. Processing continues fail-closed: the datagram is silently dropped, the read loop continues
   serving subsequent datagrams.

**Test name:** `TestRouterIngest_LookupMissAndTagMismatch_IndistinguishableRejection`
**Test level:** unit
**Test file:** `internal/discovery/discovery_wire_test.go`

---

### AC-007 — Node-local relay-ingest: `ReceiveAdvertisement` retired, `ErrSVTNMismatch` relocated as defense-in-depth (traces to BC-2.03.001 Postcondition 5)

**BC Anchor:** BC-2.03.001 Postcondition 5 (Ruling 1 point 3, corrected by F-DWSP8-001)

**Postconditions:**

1. `Discovery.ReceiveAdvertisement`, as shipped, is deleted — no caller in the shipped topology
   (router uses the new `DiscoveryAuthKeyFor` path, AC-005/AC-006; no node receives hop-1 UDP
   directly, Ruling 2).
2. A new node-side relay-ingest function decodes the hop-2 `DISCOVERY_RELAY` payload (AC-014 PC-2
   shape: `NodeAddr | Sequence | count | sessions`, no `SVTNID`) with **no per-frame HMAC** — trust
   derives from the admitted connection (AC-015).
3. `ErrSVTNMismatch` survives, relocated: compares the relay frame's `OuterHeader.SVTNID` against
   `d.cfg.LocalSVTNID`, returns `ErrSVTNMismatch` on mismatch — defense-in-depth against a
   relay/routing bug, not a crypto check.
4. On success, performs the same registry replace-on-write update `ReceiveAdvertisement`
   previously did.
5. Only the router-side ingest path (AC-005/AC-006) performs primary discovery-frame
   authentication under the relay model — unchanged from the original PC-3.

**Test names:** new unit test for the `OuterHeader.SVTNID` equality-check guard (implementer-named).
`TestDiscovery_VP045_SVTNIsolation_MultipleScopes` is retired, not extended.
**Test level:** unit
**Test file:** implementer's choice — `discovery_wire_test.go` (shared with AC-005/AC-006) or
`discovery_test.go`.

---

### AC-008 — Cold-start acceptance: first datagram for a `(SVTNID, NodeAddr)` pair always accepted (traces to BC-2.03.001 Postcondition 2; VP-080 property 1)

**BC Anchor:** BC-2.03.001 Postcondition 2 (replay-resistance field note); VP-080

**Precondition:** The router has no prior `lastSeen[svtnID, nodeAddr]` entry (fresh router start,
or the first frame from a newly-admitted node).

**Postconditions:**

1. An HMAC-verified datagram with any declared `Sequence` value (including 0) is accepted for a
   `(SVTNID, NodeAddr)` pair with no prior recorded `Sequence`.
2. The discovery registry is updated per the accepted content; the accept+relay decision is
   emitted; `lastSeen[svtnID, nodeAddr]` is set to the accepted `Sequence`.

**Note (F-DWSP4-001 scope clarification, v1.5):** this AC's precondition ("no prior `lastSeen`
entry") covers ONLY fresh-router-start and first-ever-admission cold-start — it does NOT rescue a
previously-admitted node that restarts while the router keeps running and still holds a `lastSeen`
watermark for that `(SVTNID,NodeAddr)` pair. That restart case is now closed by the epoch-qualified
`Sequence` construction (AC-010's forward-acceptance path) and characterized separately as EC-010;
do not conflate the two cold-start-shaped preconditions.

**Test name:** `TestVP080_DiscoveryIngest_ColdStartAcceptance`
**Test level:** integration
**Test file:** `internal/discovery/discovery_wire_test.go`

---

### AC-009 — Replay/stale discard: non-increasing `Sequence` rejected post-HMAC (traces to BC-2.03.001 Postcondition 2; protects BC-2.03.002 Postcondition 5; VP-080 property 2)

**BC Anchor:** BC-2.03.001 Postcondition 2; BC-2.03.002 Postcondition 5 (the staleness-expiry
guarantee this discard rule protects); VP-080; SEC-DW-07

**Precondition:** A `lastSeen[svtnID, nodeAddr] = N` entry already exists (established via AC-008
or a prior AC-010 forward acceptance).

**Postconditions:**

1. A second HMAC-verified datagram for the same `(SVTNID, NodeAddr)` declaring `Sequence <= N`
   (including the exact-replay degenerate case `Sequence == N`) is discarded — even though its
   HMAC passes.
2. The discovery registry is NOT updated for the discarded datagram.
3. The discarded datagram is NOT relayed to other admitted nodes on the SVTN.
4. `lastSeen[svtnID, nodeAddr]` is unchanged by the discard.

**Note (F-DWSP4-001 scope clarification, v1.5; residual bounds corrected v1.6):** this AC's discard
rule remains the correct, intended behavior for THREE distinct triggers — (1) genuine replay (an
attacker-captured frame, or a legitimate but out-of-order/stale duplicate); (2) the
same-wall-clock-second crash-loop restart (EC-010 case 1) — bounded to ≤1 second, one or more
discards cleared unconditionally at the next epoch tick; (3) the backward-host-clock-adjustment
restart (EC-010 case 2) — every advertisement from the restarted instance is discarded for a window
of duration ≈N (the clock-adjustment magnitude, NOT ≤1 second) until wall-clock time naturally
re-passes the pre-adjustment epoch. Cases (2) and (3) are accepted, intentional edges of the
F-DWSP4-001 fix — not regressions. VP-080 v1.7 Test Scenario 5 exercises case (2); case (3) is not
yet test-scripted.

**Test names:** `TestVP080_DiscoveryIngest_ReplayDiscard_ExactSequence`,
`TestVP080_DiscoveryIngest_ReplayDiscard_LowerSequence`,
`TestVP080_DiscoveryIngest_ReplayDiscard_NoRelaySideEffect`
**Test level:** integration
**Test file:** `internal/discovery/discovery_wire_test.go`

---

### AC-010 — Forward acceptance: strictly-increasing `Sequence` updates state and triggers relay (traces to BC-2.03.001 Postcondition 2; VP-080 property 3)

**BC Anchor:** BC-2.03.001 Postcondition 2; VP-080

**Precondition:** `lastSeen[svtnID, nodeAddr] = N` (established via AC-008).

**Postconditions:**

1. An HMAC-verified datagram declaring `Sequence = N+1` (or any value `> N`) is accepted.
2. The discovery registry is updated to the new content.
3. The accept+relay decision is emitted for the hop-2 dispatch caller.
4. `lastSeen[svtnID, nodeAddr]` advances to the new `Sequence` value.
5. `uint64` composite wraparound behavior is explicitly OUT OF SCOPE — only relative ordering (`>`)
   is asserted; no wraparound case is tested (VP-080 v1.7 property 4, Non-Goals). The composite's
   `epoch` component wraps only after ~136 years from a 1970 origin; its `counter` component wraps
   only after ~4.29 billion advertisements from a single process instance without restart.
6. **(F-DWSP4-001, v1.5, new)** A restarted access node's first post-restart datagram — declaring a
   freshly-sampled `epoch` and a low `counter` — is accepted via THIS postcondition (forward
   acceptance), not AC-008's cold-start path, because its composite `Sequence` exceeds the router's
   prior `lastSeen` watermark with overwhelming likelihood (VP-080 v1.7 Property 5). See EC-010 for
   the full restart-liveness characterization, including the bounded same-epoch-second residual.

**Test names:** `TestVP080_DiscoveryIngest_ForwardAcceptance_AdvancesState`,
`TestVP080_DiscoveryIngest_RestartForwardProgress`
**Test level:** integration
**Test file:** `internal/discovery/discovery_wire_test.go`

---

### AC-011 — Bounded, fixed-size UDP read buffer sized to realistic usage (traces to BC-2.03.001 Postcondition 5)

**BC Anchor:** BC-2.03.001 Postcondition 5; SEC-DW-02 (MED)

**Postconditions:**

1. The router's socket-read loop reads each datagram into a fixed-size buffer sized to the
   realistic worst-case legitimate advertisement — not the 65,507-byte UDP/IP theoretical maximum.
2. A datagram exceeding the sized buffer is rejected without partial-parse and without
   reallocation-to-fit.
3. `maxSessionsPerAdvertisement` is re-derived at implementation time from realistic
   tmux-sessions-per-access-node scale (implementer task — the real number depends on product
   usage data this ruling cannot derive from the wire format alone; likely low hundreds, not 1024).

**Test name:** `TestRouterIngest_OversizedDatagram_RejectedNoPartialParse`
**Test level:** unit
**Test file:** `internal/discovery/discovery_wire_test.go`

---

### AC-012 — Aggregate rate cap at ingest; `FailureCounter` reused visibility-only (traces to BC-2.03.001 Postcondition 5)

**BC Anchor:** BC-2.03.001 Postcondition 5; SEC-DW-03 (MED)

**Postconditions:**

1. An aggregate (not per-source) token-bucket cap at the socket-read loop rejects datagrams once
   the aggregate rate is exceeded, regardless of declared `NodeAddr`.
2. The existing `FailureCounter` (threshold=5/60s) is invoked on HMAC-rejection events for
   operator visibility only; it never gates admission or ingest based on the declared,
   attacker-controlled `NodeAddr`.
3. A source rotating its declared `NodeAddr` across successive forged datagrams does not evade the
   aggregate cap.

**Test names:** `TestRouterIngest_AggregateRateCap_NotPerSource`,
`TestRouterIngest_FailureCounter_VisibilityOnly_NeverGates`
**Test level:** unit
**Test file:** `internal/discovery/discovery_wire_test.go`

---

### AC-013 — Rate-limited, counter-based failure logging (traces to BC-2.03.001 Postcondition 5)

**BC Anchor:** BC-2.03.001 Postcondition 5; SEC-DW-04 (MED)

**Postconditions:**

1. Discovery HMAC-rejection logging fires only on `FailureCounter`'s own threshold-crossing
   emission, not unconditionally per rejected packet.
2. This is explicitly distinct from BC-2.05.008's per-packet TCP HMAC-failure logging policy —
   discovery's ingest path never adopts the per-packet form.

**Test name:** `TestRouterIngest_FailureLogging_ThresholdCrossingOnly_NotPerPacket`
**Test level:** unit
**Test file:** `internal/discovery/discovery_wire_test.go`

---

### AC-014 — `DISCOVERY_RELAY` frame assembly: `control_type=0x03` payload layout (traces to BC-2.01.008 Postcondition 2, Postcondition 3, Invariant 5; BC-2.03.001 Postcondition 5)

**BC Anchor:** BC-2.01.008 Postcondition 2 (registry row, already landed v1.2), Postcondition 3 +
Invariant 5/DI-007 (4-byte header + extend-beyond-byte-3 allowance); BC-2.03.001 Postcondition 5
relay/connection-trust note

**Precondition:** The router-side ingest path (AC-010) has produced an accept+relay decision for a
`(SVTNID, NodeAddr, Sequence)`-identified advertisement.

**Postconditions:**

1. The relay frame is a `FrameTypeCtl` (`0x03`) outer frame whose payload begins with
   `control_type = 0x03`, `version = 0x01`, `reserved = 0x0000` at bytes 0-3.
2. Bytes 4-11 carry the originating access node's 8-byte `NodeAddr`; bytes 12-19 carry the
   `Sequence` value (uint64 BE, epoch-qualified per F-DWSP4-001) hop-1 accepted; bytes 20-21 carry
   the session count (uint16 BE); bytes 22+ carry the per-session list using
   `internal/discovery`'s existing per-session encoding.
3. `SVTNID` is not repeated in the payload — the relay frame's own `OuterHeader.SVTNID` carries
   SVTN scope.
4. Frame assembly is a pure function testable independent of any live connection or dispatch
   mechanism (no fan-out target resolution required to construct the frame bytes).

**Test name:** `TestAssembleDiscoveryRelayFrame_PayloadLayout`
**Test level:** unit
**Test file:** `cmd/switchboard/discovery_relay_wire_test.go` (new)

---

### AC-015 — Zero `HMACTag` on relay frame: connection-trust boundary (traces to BC-2.03.001 Postcondition 1 delivery-mechanism note)

**BC Anchor:** BC-2.03.001 Postcondition 1 (delivery-mechanism note); SEC-DW-08 (hop-2 half)

**Postconditions:**

1. The `DISCOVERY_RELAY` frame's `OuterHeader.HMACTag` is the zero value — matching the DRAIN
   precedent (`S-7.04-FU-DRAIN-WIRE`) exactly.
2. No per-frame HMAC is computed for hop-2; the receiving node's trust in the relayed content
   derives exclusively from its own already-admitted TCP connection to the router.

**Test name:** `TestAssembleDiscoveryRelayFrame_ZeroHMACTag`
**Test level:** unit
**Test file:** `cmd/switchboard/discovery_relay_wire_test.go`

---

### AC-016 — Payload is re-serialized, never a raw retransmission of hop-1 bytes (traces to BC-2.03.001 Postcondition 5 relay/connection-trust note)

**BC Anchor:** BC-2.03.001 Postcondition 5 (relay/connection-trust note)

**Postconditions:**

1. The relay frame's payload bytes are freshly constructed from the decoded `NodeAddr`,
   `Sequence`, and session-list fields (AC-014) — never a byte-for-byte copy of hop-1's raw UDP
   datagram.
2. Hop-1's original HMAC tag never appears anywhere in the relay frame.

**Test name:** `TestAssembleDiscoveryRelayFrame_NotRawHop1Bytes`
**Test level:** unit
**Test file:** `cmd/switchboard/discovery_relay_wire_test.go`

---

### AC-017 — [GATED — depends_on S-BL.NODE-IDENTIFY-WIRE] SVTN-scoped, exclude-originator, best-effort fan-out dispatch (traces to BC-2.03.001 Postcondition 1 delivery-mechanism note)

**BC Anchor:** BC-2.03.001 Postcondition 1 (delivery-mechanism note)

**Gate:** This AC's Green step is gated on **`S-BL.NODE-IDENTIFY-WIRE`** (the fan-out target-resolution
companion story named by the 2026-07-14 human gate disposition, resolving Ruling 3(f)'s Forward
Obligation — see rulings v1.10 item (j), `S-BL.DISCOVERY-WIRE-fanout-options.md` v1.1). AC text is
fully specified below and does not change once `S-BL.NODE-IDENTIFY-WIRE` lands; only the
target-resolution mechanism it dispatches through does.

**Postconditions:**

1. The router iterates the live connections of nodes admitted to the advertisement's SVTN,
   excluding the originating `NodeAddr`.
2. For each remaining target, dispatch is best-effort and non-blocking:
   `select { case nc.send <- relayFrame: default: }` — matching DRAIN's own dispatch shape
   exactly; a slow/stuck node drops the relay silently rather than blocking the router.
3. The originating node never receives an echo of its own advertisement via hop-2.
4. No queueing, no retry, no wire ACK for the relay send.

**Test name:** `TestRelayDispatch_SVTNScoped_ExcludeOriginator_BestEffortNonBlocking`
**Test level:** integration
**Test file:** `cmd/switchboard/discovery_relay_wire_test.go`

---

### AC-018 — [GATED — depends_on S-BL.NODE-IDENTIFY-WIRE] Relay-dispatch rate cap (traces to BC-2.03.001 Postcondition 5; SEC-DW-09)

**BC Anchor:** BC-2.03.001 Postcondition 5; SEC-DW-09

**Gate:** Same gate as AC-017 — the rate-cap decision is meaningless without a live dispatch
mechanism to suppress.

**Postconditions:**

1. Relay dispatch for a given `(SVTNID, NodeAddr)` pair is capped at `~1/sec`, independent of HMAC
   validity or `Sequence` freshness (both already passed by the time this cap is evaluated).
2. An advertisement arriving faster than the cap from the same sender still updates the router's
   own registry/discard-map state (AC-010's correctness is unaffected) but is NOT relayed on that
   excess arrival — silent drop is the actual backstop.
3. An optional visibility counter (matching `FailureCounter`'s shape) may record cap-triggered
   suppressions but never gates or alters the drop decision itself.

**Test name:** `TestRelayDispatch_RateCap_PerSVTNNodeAddr_SilentDropFirst`
**Test level:** integration
**Test file:** `cmd/switchboard/discovery_relay_wire_test.go`

## Forward Obligations (tracked as story tasks — the adversary MUST police these)

| # | Obligation | Owner | Gate | Status |
|---|-----------|-------|------|--------|
| (a) | Fan-out **target resolution** (Ruling 3(f)): binding node identity (`NodeAddr`) to a live connection's `InterfaceID`/`nodeConn` does not exist in production code today — `admission.AdmitNode` has zero production call sites; `sendMap` carries no `NodeAddr`. Gates AC-017/AC-018 and Task 6. | architect / PO (disposition), then implementer | **RESOLVED — 2026-07-14.** Human gate selected `S-BL.DISCOVERY-WIRE-fanout-options.md` v1.0 Option 1: named companion story **`S-BL.NODE-IDENTIFY-WIRE`** (added to `depends_on`) delivers the `NODE_IDENTIFY` handshake. AC-017/AC-018/Task 6 now gate on that story by name, not on this table row. See rulings v1.10 item (j). | RESOLVED |
| (b) | `VP-044`/`VP-045` are `PARTIAL` per the Phase-6 VP sweep (RULING-W6TB-D doctrine) — full coverage requires this story's real UDP-multicast integration tests to land, superseding the "Blocker: multicast wire transport implementation" notes on both VPs. | formal-verifier | None — discharged once AC-001/AC-003 tests land and pass. | OPEN — non-blocking (post-Task-4 cleanup, not a scheduling gate on any AC) |
| (c) | `VP-080` is `lifecycle_status: draft` (minted ahead of this elaboration per its own Lifecycle section) — the `draft → active` transition is this story's job once it is scoped into a wave. | architect / spec-steward | None — mechanical once this story reaches `ready`/wave-scheduled. | OPEN — non-blocking |
| (d) | `BC-2.03.002` Postcondition 1's `PENDING-S-BL.DISCOVERY-WIRE` annotation anticipates this story also wiring a `sessions.list` RPC handler exposing `discovery.Enumerate()` over the mgmt wire. **None of the three rulings (v1.10) adjudicate this** — Ruling 1/2/3 scope is exclusively the UDP-multicast advertisement transport (hop-1 ingest + hop-2 relay), not a console-facing enumeration RPC. This is a discovered scope question, not resolved here. | PO / architect | **RESOLVED — 2026-07-14.** Human gate resolved this to a new backlog story, **`S-BL.SESSIONS-LIST-WIRE`** (`stories/S-BL.SESSIONS-LIST-WIRE.md` v1.0, draft, wave backlog). `BC-2.03.002` PC-1's annotation is already re-pointed to that story's ID on disk (`BC-2.03.002.md` v1.5). See rulings v1.10 item (l). | RESOLVED |
| (e) | Router daemon-lifecycle wiring for the discovery multicast listener (AC-001 Postcondition 1): `wireDiscoveryListener` is fully implemented and tested but not called from `runRouter` — the router process has no source of "which SVTN(s) am I serving" (`admission.AdmittedKeySet` has no SVTN-enumeration method; the only production `RegisterKey` caller runs in control-mode, a separate, disconnected OS process from router-mode). Same root cause also blocks `S-BL.NODE-IDENTIFY-WIRE`'s own `AdmitNode` call (verification-only; requires the key already be present in the router's own, always-empty `AdmittedKeySet`). | architect (disposition, Ruling 4) / PO (scoping), then implementer | New follow-on story recommended: `S-BL.ADMISSION-SYNC-WIRE` (working name — not yet created; PO/architect to confirm name + scope). `S-BL.NODE-IDENTIFY-WIRE` must add this story as a `depends_on` prerequisite once both exist. See rulings v1.10 Ruling 4. | OBLIGATION NAMED — story not yet created (unlike rows (a)/(d), no stub exists yet) |

**Obligations (a) and (d) are resolved as of 2026-07-14** (see rows above); (b)/(c)/(e) remain open
and non-blocking. None of (b)/(c)/(d)/(e) ever blocked TDD implementation of AC-001..AC-016 —
(e) is a retrospective, implementation-time finding (Ruling 4, v1.10) accepted at function level
with a qualifying Scope note on AC-001, not a `[GATED]` marker. AC-017/AC-018/Task 6 now gate on
**`S-BL.NODE-IDENTIFY-WIRE`** by name (not on an open disposition) — Tasks 1-5 remain independently
deliverable regardless of that story's schedule.

## Human Gate — Story-Ready Sign-off Required

Three items are carried forward from the architect ruling for explicit human/PO disposition before
this story is promoted from `draft` to `ready` and scheduled into a wave. None of them block
writing or landing the code for AC-001 through AC-016; they gate the `ready` transition and, for
item 3, the scheduling of Task 6/AC-017/AC-018 specifically.

> **DISPOSITIONED — 2026-07-14.** All three items below received human sign-off/selection at the
> story-ready gate. See `S-BL.DISCOVERY-WIRE-rulings.md` v1.10, Ruling 3 subsection "Ruling 3(f)
> Forward Obligation, SEC-DW-07, and the discovery port — human gate disposition," items (j)/(k)/(l),
> and `S-BL.DISCOVERY-WIRE-fanout-options.md` v1.1 for the full record. This story may now be
> promoted to `ready` on this axis.

### 1. SEC-DW-07 monotonic-`Sequence`-field adjudication

The architect already ruled on this (Ruling 1, "Replay / freshness" subsection): a new
`Sequence uint64` (epoch-qualified per the v1.5 restart-liveness amendment — see below; originally
specified as `uint32`) wire field is added to `AdvertisementPayload`, with a router-held
per-`(SVTNID,NodeAddr)` last-accepted map, discarding non-increasing sequences even after HMAC
passes. This is a **new wire-format field and new router-held state** that did not exist before
this story — the architect's own ruling flags it "prominently for the human gate at story-ready...
a human sign-off should see it named explicitly before the story is scheduled, not just inherit
silently." Rationale for adopting it now rather than deferring: design-time cost is small (one
field, one small map) vs. retrofit-after-ship (a wire-format version bump); the threat model is
unusually easy to execute specifically because Ruling 2 makes the channel multicast (passive LAN
capture, no compromise required); the harm directly contradicts BC-2.03.002 Postcondition 5's
stated staleness-expiry guarantee. **Ask for sign-off:** confirm the field addition is acceptable
scope for this story (vs. splitting it into a separate wire-format-versioning story).

**v1.5 update (F-DWSP4-001), residual bounds corrected v1.6:** between this item's original framing
and story-ready, spec-adversarial pass 4 found the `Sequence` field as originally specified
(in-memory counter, resets on node restart) paired against the router's restart-STABLE `lastSeen`
watermark, producing up to ~8.3h of silent discovery absence on every ordinary node
restart/redeploy/crash-recover — not just a rare edge case. The architect's fix (rulings v1.5;
residual bounds corrected v1.6): widen `Sequence` to `uint64`, epoch-qualified (high 32 bits = wall-clock seconds sampled at process
start, low 32 bits = the original counter) — self-contained at the sender, no router-side logic
change, no new state, no dependency on the still-missing node-identity-to-connection binding
(Forward Obligation (a)). Two residuals remain, with DIFFERENT bounds — read both before signing: a
same-wall-clock-second crash-loop is bounded to ≤1 second (self-clearing at the next epoch tick); a
backward host-clock adjustment of magnitude N produces a discard window of duration ≈N —
deterministic and finite, but not bounded by a small constant, and could in principle exceed the
original ~8.3h exposure for a large-enough backward step. The second case is accepted not because
it's short, but because its trigger (a restart coinciding with an operator clock misconfiguration)
is rare and absent from the original defect, which fired on every ordinary restart with certainty.
**Ask for sign-off:** confirm the widened field remains acceptable scope, and that BOTH residual
shapes — the ≤1s self-clearing case and the N-bounded clock-misconfiguration case — are acceptable
to ship without further hardening (e.g., a monotonic-clock cross-check, or clamping N at some
maximum) — this ruling explicitly declined that additional hardening as diminishing-returns
complexity, but a human sign-off should see the specific trade-off before it's locked in.

> **Disposition (2026-07-14): APPROVED as documented.** Both residual bounds — Case 1 (≤1s,
> same-wall-clock-second crash-loop) and Case 2 (≈N, backward host-clock adjustment) per rulings
> v1.6's precision-corrected framing — are accepted with no further hardening requested. See
> rulings v1.10 item (k).

### 2. Discovery UDP port number

`49201` is recommended (IANA dynamic/private range 49152–65535, arbitrary, unregistered) but is
explicitly a bikeshed-level placeholder per Decision 2(c) — not gated on any ruling's substance.
**Ask for sign-off:** confirm `49201` or supply an alternative before the constant is committed to
`internal/discovery`.

> **Disposition (2026-07-14): `49201` ADOPTED — no longer a placeholder.** The bikeshed is closed;
> `49201` is the adjudicated discovery UDP port. See rulings v1.10 item (k).

### 3. Fan-out target-resolution — two resolution paths, architect's analysis

Ruling 3(f) verified (not invented) that the node-identity-to-connection binding hop-2 fan-out
needs does not exist in production code today. Two paths were named, with the architect's own
caveat on option (ii):

- **Option (i) — sequencing dependency (architect's recommended default).** Add an explicit
  `depends_on` edge from this story's hop-2 fan-out task to whatever future story delivers
  node-identity-to-connection binding (the same gap this project has already named generically as
  `FO-DRAIN-WIRE-002`, referenced but not yet scheduled as a named story anywhere in
  `.factory/stories/`). This story's frontmatter does NOT currently list such a dependency because
  no successor story exists yet to name — that absence is itself the artifact of this open
  question, not an oversight. If option (i) is chosen, a new story (working name suggestion:
  something in the shape of a session-bootstrap/admission-handshake-on-connect story) must be
  created and added to `depends_on` before Task 6 can be scheduled into a wave.
- **Option (ii) — narrow story-local seam.** Scope a small, story-local
  `Router.BindInterface(svtnID, nodeAddr, ifaceID)`-shaped method (analogous in size to the
  existing `RegisterForwardingEntry`) if blocking on an unscheduled story is unacceptable. **The
  architect's explicit caveat:** this narrow seam still requires SOME connection-time identity
  signal to call it with — which circles back to the same unimplemented
  admission-handshake-on-connect gap; it does not eliminate the dependency, it only relocates
  where it must be resolved (inside this story, as new scope, rather than in a successor story).
  Choosing (ii) means this story's own points estimate would need revisiting — it was not sized
  assuming this seam's implementation.

The architect recommends (i) as "the honest default unless PO/story-writer has visibility into a
scheduled session-bootstrap story I don't have." **Ask for sign-off:** PO/architect selects (i) or
(ii). This elaboration deliberately does not choose — per the orchestrator's instruction not to
invent this binding, and per the general principle that a story-writer transcribes architect
rulings rather than resolving an item the ruling itself left open for a different role.

> **Disposition (2026-07-14): NEITHER option (i) nor option (ii) selected — both rejected.** The
> human asked for better options; `S-BL.DISCOVERY-WIRE-fanout-options.md` v1.0 (six additional
> options, grounded in the shipped code) was produced and reviewed. **Selected: that document's
> Option 1** — a new, immediately-named, immediately-scheduled companion story,
> **`S-BL.NODE-IDENTIFY-WIRE`**, delivers the `control_type=0x04` `NODE_IDENTIFY` handshake (wiring
> `admission.AdmitNode`/`admission.GenerateChallenge` over the live connection, recording
> `(SVTNID, NodeAddr) → IfaceID` via a `Router.BindInterface`-shaped method). Add
> `S-BL.NODE-IDENTIFY-WIRE` to this story's `depends_on`; Task 6/AC-017/AC-018 gate on it by name.
> See rulings v1.10 item (j) for full mechanism and rationale, and
> `S-BL.DISCOVERY-WIRE-fanout-options.md` v1.1 for the evaluation + disposition record.

## Non-Goals

- **Literal IPv6 discovery transport** — no IPv6 data-plane precedent exists anywhere in this
  codebase; scoping it now would require inventing both an IPv6 story and an IPv6-specific
  administratively-scoped derivation (RFC 3306) with zero grounding (Decision 2(d)).
- **`uint64` composite `Sequence` wraparound handling** — out of scope for this story and VP-080
  v1.7 property 4 (see AC-010 postcondition 5 for the composite's per-component wrap bounds); not a
  practical concern at realistic heartbeat rates within any reasonable node uptime.
- **`sbctl sessions list` / `sessions.list` RPC wire exposure** — BC-2.03.002 Postcondition 1's
  `PENDING-S-BL.DISCOVERY-WIRE` annotation anticipates a console-facing enumeration RPC, but none
  of the three rulings adjudicate one. This story's scope is exclusively the advertisement
  transport (hop-1 ingest + hop-2 relay). Flagged as Forward Obligation (d) — a distinct scope
  question for PO/architect, not silently absorbed here.
- **Dynamic multicast address allocation, or a release step on `admin.svtn.destroy`** — the
  derivation is a pure, static function of `svtnID`; there is no allocation state to track,
  collide against, or release (Decision 2(b)).
- **A new outer `FrameType` byte for hop-2** — evaluated and rejected; the 6-slot enum is
  exhausted and the shipped DRAIN precedent already establishes the `control_type`-discriminator
  pattern as this codebase's actual convention for router-terminated control operations
  (Decision 3(a)).
- **`routing.SplitHorizon.Forward`/`FrameArrivalHandler.OnFrameArrival` reuse for hop-2 fan-out** —
  evaluated as a real alternative and rejected: its inter-router loop-prevention machinery doesn't
  fit this story's single-router star topology, and its `arrivalIface`-keyed exclusion has no
  natural value for a UDP-sourced frame (Decision 3(d)).
- **A new per-SVTN broadcast group secret key** — considered and rejected in favor of reusing the
  already-shipped `FrameAuthKey` derivation shape, domain-separated (Decision 1).
- **Dummy-HMAC-on-lookup-miss timing-oracle hardening** — SEC-DW-05's optional hardening layer;
  not required for this story, may be a future hardening-pass candidate.
- **The fan-out target-resolution binding itself (node-identity-to-connection)** — explicitly not
  invented by this story; see Forward Obligation (a) / Human Gate item 3.

## Edge Cases

BC-2.03.001's existing EC-001 (heartbeat lost in transit), EC-002 (session closes mid-flight),
EC-003 (brief SVTN disconnect/resync), and EC-004 (100-session fragmentation) are inherited from
S-7.02's registry model and are unaffected by this story — this story makes their delivery
mechanism real over the wire without changing their expected behavior. New edge cases introduced
by this story:

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-005 | Two different SVTNs derive colliding multicast addresses | Harmless — the receiving router's `DiscoveryAuthKeyFor` lookup fails for the foreign SVTN's datagram and it is dropped fail-closed exactly as an unauthenticated frame would be (Decision 2(a)); HMAC, not address uniqueness, is the security boundary. |
| EC-006 | Router restarts, or a node is newly admitted, with no prior `lastSeen` entry | First frame for that `(SVTNID,NodeAddr)` pair is accepted unconditionally regardless of its `Sequence` (AC-008), bounding the residual replay window to at most one heartbeat interval — the same bounded-not-perfect posture `admission`'s `nonceTTL=60s` already accepts. |
| EC-007 | Oversized or malformed UDP datagram arrives at the router's discovery listener | Rejected without partial-parse, without reallocation-to-fit, before any key lookup (AC-011). |
| EC-008 | An admitted, HMAC-valid, sequence-increasing sender relays advertisements faster than ~1/sec | Registry/discard-map state still updates correctly (AC-010 unaffected); relay dispatch for the excess arrivals is silently suppressed (AC-018, GATED per Human Gate item 3). |
| EC-009 | An attacker with no valid `DiscoveryAuthKey` sends datagrams naming a known, admitted `NodeAddr` | Rejected identically to a datagram naming an unknown `NodeAddr` — `ErrInvalidHMACTag`, no distinguishing signal (AC-006). |
| EC-010 | Access node restarts with a stable admitted identity while the router keeps running (no router restart, no `lastSeen` entry evicted) | The restarted node's `Discovery` instance re-samples `epoch = uint32(time.Now().UTC().Unix())` at instance start; because real wall-clock time has advanced since the prior process's epoch capture, the new composite `Sequence` is with overwhelming likelihood strictly greater than the router's stored `lastSeen` value, so post-restart advertisements are accepted via forward acceptance (AC-010) rather than being locked out — closing F-DWSP4-001's ~8.3h liveness gap. Two distinct residuals remain, with different bounds (do not conflate): **Case 1 — same-wall-clock-second crash-loop:** two-or-more restarts within the same `epoch` tick produce a same-epoch, low-counter `Sequence` that IS correctly discarded once (or a handful of times) by AC-009's existing rule — bounded to at most 1 second (until the epoch advances), a strict improvement over the pre-fix outage and consistent with this project's existing bounded-not-perfect replay posture (`admission.nonceTTL=60s`, EC-006). **Case 2 — backward host-clock adjustment of magnitude N:** every advertisement from the restarted instance is discarded (not just one) for a window of duration ≈N until wall-clock time naturally re-passes the pre-adjustment epoch — deterministic and finite, but NOT bounded by a small constant; N could in principle exceed the pre-fix ~8.3h bound for a sufficiently large backward step. Case 2 is accepted because its trigger (a restart coinciding with an operator clock misconfiguration) is a rare precondition absent from the original defect, which fired unconditionally on every ordinary restart — not because its duration is small. |

## Architecture Mapping

| Component | Module | Pure/Effectful | Notes |
|-----------|--------|-----------------|-------|
| `hmac.DeriveDiscoveryKey`, `HKDFInfoDiscovery` | `internal/hmac` | pure-core | No I/O; stdlib `crypto/hmac`+`crypto/sha256` only, matching the package's existing zero-external-dependency invariant |
| `(*routing.Router).DiscoveryAuthKeyFor`, `routing.DeriveDiscoveryKey` | `internal/routing` | pure-core | Read-only lookup + derivation; preserves ARCH-08 position-14 boundary (`discovery` imports ONLY `routing`) |
| `discovery.MulticastAddrFor` | `internal/discovery` | pure-core | Deterministic SHA-256-derived address, no I/O |
| `AdvertisementPayload.Sequence`, `encodeBody`/`decodeBody` extension | `internal/discovery` | pure-core | Wire encode/decode, no I/O |
| Router-side ingest decision function (new, e.g. `discovery_wire.go`) | `internal/discovery` | pure-core (decision logic) | Fixed-offset extraction, HMAC verify call, sequence-gate decision — returns accept/relay verdict, performs no I/O itself |
| Discovery multicast listener bind/join/read loop | `internal/discovery` + `cmd/switchboard` | effectful-shell | Socket I/O |
| Sender-side `WriteTo`/TTL=1 dispatch | `internal/discovery` | effectful-shell | Socket I/O |
| `DISCOVERY_RELAY` frame assembly (new, e.g. `discovery_relay_wire.go`) | `cmd/switchboard` | pure-core (assembly) | Byte-layout construction, no I/O (AC-014/015/016 are testable without a live connection) |
| Relay-dispatch closure (fan-out + rate cap) | `cmd/switchboard` | effectful-shell | `sendMap`-based socket writes — GATED, Task 6 |
| `runRouter` wiring (listener + relay dispatch registration) | `cmd/switchboard/mgmt_wire.go` | effectful-shell | Register-before-serve, router-mode-exclusive |
| `testenv.MulticastLoopbackInterface` | `internal/testenv` | effectful-shell (test infra) | Platform-appropriate loopback interface resolution for VP-044/VP-045 integration tests |

## Purity Classification

| Module | Classification | Justification |
|--------|---------------|----------------|
| `internal/hmac` | pure-core | Unchanged invariant: stdlib-only, no I/O, deterministic — `DeriveDiscoveryKey` is a second, purpose-bound derivation of the same already-pure function shape. |
| `internal/routing` (new methods only) | pure-core | `DiscoveryAuthKeyFor`/`DeriveDiscoveryKey` are read-only lookups and derivations over already-held state; no new mutation, no I/O. |
| `internal/discovery` (encode/decode, address derivation, ingest decision) | pure-core | Wire-format and decision logic separated from socket I/O by construction — the ingest decision function returns a verdict, the caller performs I/O. |
| `internal/discovery` (listener, sender dispatch) | effectful-shell | UDP socket bind/join/read/write. |
| `cmd/switchboard` (frame assembly) | pure-core | Byte-layout construction only. |
| `cmd/switchboard` (relay dispatch, `runRouter` wiring) | effectful-shell | Socket writes, connection-set iteration, registration side effects. |

## Architecture Compliance Rules (MANDATORY)

| Rule | Source | Enforcement |
|------|--------|--------------|
| `internal/discovery` MUST import ONLY `internal/routing` among `internal/` packages — `internal/admission` and `internal/hmac` are forbidden imports. | ARCH-08 §6.5 position-14 (`ARCH-08-dependency-graph.md` v2.13, line 159/173) | `go list -deps ./internal/discovery/...` boundary check; existing package doc comment in `discovery.go` states the rule explicitly. |
| `internal/hmac` MUST NOT import any other `internal/` package or any external dependency — stdlib `crypto/hmac`+`crypto/sha256` only. | `internal/hmac/hmac.go` package doc comment (existing invariant, unchanged by this story) | Same `go list -deps` check; `DeriveDiscoveryKey`'s addition must not introduce a new import. |
| Router-side ingest MUST perform fixed-offset key-selector extraction before any call to `decodeBody()` (SEC-DW-01, MANDATORY, not optional hardening). | `S-BL.DISCOVERY-WIRE-rulings.md` v1.10, Ruling 1 Implementation Constraint 2 | AC-005 tests; code review. |
| HMAC verification MUST cover the complete raw body bytes, not merely the key-selector prefix. | Same | AC-005 tamper-detection test. |
| New `control_type` opcode allocation MUST be sequential and append-only (`0x03` after `0x01`/`0x02`); existing opcode values are never reassigned. | BC-2.01.008 Invariant 3 | Already satisfied — `DISCOVERY_RELAY=0x03` row landed v1.2; no story-writer/implementer action needed to satisfy this rule, only to not violate it. |
| The `DISCOVERY_RELAY` relay frame's `HMACTag` MUST be zero — hop-2's trust boundary is the admitted connection, never a per-frame HMAC. | `S-BL.DISCOVERY-WIRE-rulings.md` v1.10, Ruling 3(b) | AC-015 test. |
| Relay dispatch MUST be best-effort, non-blocking (`select`/`default`), never a blocking send that could stall the router on a slow/stuck node. | Ruling 3(d), DRAIN precedent | AC-017 test (GATED — depends_on S-BL.NODE-IDENTIFY-WIRE). |

## Forbidden Dependencies

- **`internal/discovery` MUST NOT gain a dependency on `internal/admission` or `internal/hmac`.**
  If this module's import graph gains either dependency, the boundary check (`go list -deps`) MUST
  fail the review. All HMAC and admitted-key operations route through the existing
  `internal/routing.Router` thin-wrapper surface (`ComputeAdvertisementHMAC`/
  `VerifyAdvertisementHMAC` precedent, extended by `DiscoveryAuthKeyFor`/`DeriveDiscoveryKey`).
- **`internal/hmac` MUST NOT gain any external or `internal/` dependency.** Its existing package
  doc comment states this as an invariant; `DeriveDiscoveryKey` and `HKDFInfoDiscovery` must be
  implemented using only `crypto/hmac`+`crypto/sha256`, reusing the existing `hkdfSHA256` helper
  unchanged.
- **`cmd/switchboard`'s relay-dispatch closure MUST NOT route through `routing.SplitHorizon.Forward`
  or `FrameArrivalHandler.OnFrameArrival`.** Explicitly evaluated and rejected (Decision 3(d)) —
  reintroducing this dependency would misapply multi-router loop-prevention machinery to a
  single-router star topology that doesn't need it.

## Library & Framework Requirements (MANDATORY)

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.25.4 | Toolchain version pin, per `go.mod` — unchanged, repo-wide. |
| stdlib `net` | (stdlib, no version pin) | `net.ListenMulticastUDP`, `net.InterfaceByName`, `net.WriteTo`/`net.DialUDP` for all hop-1 UDP transport. No external multicast library introduced. |
| stdlib `crypto/hmac`, `crypto/sha256` | (stdlib, no version pin) | `DeriveDiscoveryKey`'s HKDF-SHA256 construction reuses `internal/hmac`'s existing hand-rolled `hkdfSHA256` helper — no `golang.org/x/crypto/hkdf` dependency introduced. |
| `golang.org/x/crypto` | v0.53.0 (already a `go.mod` dependency, unrelated packages) | Already pinned repo-wide; this story introduces **no new external dependency**. |

_This story introduces zero new third-party dependencies — every new function is stdlib-only,
consistent with `internal/hmac`'s existing "no external dependencies" package invariant and
SOUL.md §7 (build the simplest thing that works)._

## File Structure Requirements / File-Change List

| File | Action | Purpose |
|------|--------|---------|
| `internal/hmac/hmac.go` | modify | Add `HKDFInfoDiscovery` constant + `DeriveDiscoveryKey` function (SEC-DW-06). `DeriveKey`/`HKDFInfo`/`RegisterKey` untouched. |
| `internal/hmac/hmac_test.go` | extend | AC-004 domain-separation tests. |
| `internal/routing/advertisement_hmac.go` | modify | Add `DiscoveryAuthKeyFor` and `DeriveDiscoveryKey` thin wrappers per Ruling 1 Implementation Constraints 1 and 4. |
| `internal/routing/advertisement_hmac_test.go` | extend | AC-004 lookup-success/lookup-miss/sender-router-agreement tests. |
| `internal/discovery/discovery.go` | modify | Delete `advertisementKey`; add `Sequence uint64` field to `AdvertisementPayload`, epoch-qualified per F-DWSP4-001 (high 32 bits = `uint32(time.Now().UTC().Unix())` sampled at `Discovery`-instance start, low 32 bits = the original counter) + `encodeBody`/`decodeBody` extension (SEC-DW-07); add `MulticastAddrFor(svtnID) net.IP`; update `Encode`/`Decode`/sender-side call sites to the new lookup-based key scheme; extend `Run()`/`Advertise` to dispatch over real UDP `WriteTo` with TTL=1 (AC-003). |
| `internal/discovery/discovery_wire.go` | create | Router-side ingest path (new file, per rulings-doc touch-list naming): fixed-offset key-selector extraction (SEC-DW-01), bounded read buffer (SEC-DW-02), aggregate rate limiter + `FailureCounter` visibility reuse (SEC-DW-03), rate-limited failure logging (SEC-DW-04), `lastSeen` replay-discard map (SEC-DW-07). Returns an accept/relay decision to its caller; performs no relay I/O itself. |
| `internal/discovery/discovery_wire_test.go` | create | AC-005, AC-006, AC-008, AC-009, AC-010, AC-011, AC-012, AC-013 tests — new router-side ingest test file; AC-005/AC-006 establish the admitted-node/SVTN `DiscoveryAuthKeyFor`-admitted test-setup pattern that VP-080 v1.7 cites as the surviving lineage after `TestDiscovery_VP045_SVTNIsolation_MultipleScopes`'s outright retirement (see the adjacent `discovery_test.go` row's AC-007 disposition). |
| `internal/discovery/discovery_test.go` | extend | AC-002 (`MulticastAddrFor` determinism), AC-003 (sender-side TTL=1 dispatch). AC-007 (F-DWSP8-001): `ReceiveAdvertisement`'s retirement breaks ten existing test functions — `TestDiscovery_Enumerate_NoHostnameRequired`, `TestDiscovery_Enumerate_SameSessionNameTwoNodes`, `TestDiscovery_Advertise_HMACAuthenticated`, `..._EmptyPayload`, `..._TagCorruption`, `TestDiscovery_Enumerate_SVTNIsolation`, `..._ForgedSVTN`, `..._ErrSentinel`, `TestDiscovery_VP045_SVTNIsolation_MultipleScopes`, `TestDiscovery_Decode_RejectsZeroLengthName` — all need rewriting against the router-side `DiscoveryAuthKeyFor`-admitted model and/or the new node-side relay-ingest model; the VP045-named test is retired outright, not extended. |
| `cmd/switchboard/discovery_wire.go` | create | Router-mode-exclusive multicast listener wiring: bind/join/teardown, calls into `internal/discovery`'s router-side ingest path. |
| `cmd/switchboard/discovery_wire_test.go` | create | AC-001 (router-mode-exclusive group membership) tests. |
| `cmd/switchboard/discovery_relay_wire.go` | create | `DISCOVERY_RELAY` frame assembly (AC-014/015/016, not gated); relay-dispatch closure — fan-out + SEC-DW-09 rate cap (AC-017/018, **GATED**, Task 6). |
| `cmd/switchboard/discovery_relay_wire_test.go` | create | AC-014, AC-015, AC-016 tests (not gated); AC-017, AC-018 tests (**GATED** — written against a stub/injected connection-set until Human Gate item 3 resolves, per Task 6). |
| `cmd/switchboard/mgmt_wire.go` (`runRouter`) | modify | New Phase wiring: discovery multicast listener registration (Task 3) + relay-dispatch closure registration (Task 6, **GATED**) — mirrors the `wireMetricsHandlers`/`wireRouterControlHandlers`/DRAIN-observer register-before-serve precedent. |
| `internal/testenv/multicast_loopback.go` | create | `MulticastLoopbackInterface(t testing.TB) *net.Interface` — new, purpose-built helper; explicitly NOT an extension of `NewLoopback` per Decision 2(e). |
| `internal/testenv/multicast_loopback_test.go` | create | Self-test for the new helper. |

**No ARCH-08 §6.4 registration obligation** — no new `internal/` package position is introduced;
`internal/discovery` and `internal/routing` already occupy their existing positions (14, 5) and
`internal/hmac` is unpositioned/leaf per its own "no internal/ imports" invariant. `cmd/switchboard`
is not a `internal/` package.

## Token Budget Estimate (MANDATORY)

**Methodology:** byte counts below are `wc -c` on the files as they exist at
`develop@1f25677d00a3f6bc5f96f1a0a0571033ade9eb6a` (post-`S-BL.CLI-SURFACE-COMPLETION` merge),
divided by 4 chars/token, for every file that currently exists. Content that does not yet exist
(new-file stub bodies, the 18 ACs' worth of new/extended test code) is a line-count-based estimate,
called out explicitly. Broken out per dispatch pass per the per-story-delivery TDD sequence
(stub-architect → test-writer → implementer); each pass runs in its own fresh-context dispatch.

### Pass 1 — Stub pass (stub-architect)

| Context Source | Estimated Tokens |
|---|---|
| This story spec (full — File-Change List, Architecture Mapping, all 18 ACs' signatures) | ~28k |
| `S-BL.DISCOVERY-WIRE-rulings.md` (full, 87,319 bytes — the binding source for every mechanism this story implements) | ~22k |
| Precedent production files (`discovery.go` 20,998B, `hmac.go` 5,164B, `advertisement_hmac.go` 1,152B, `mgmt_wire.go` 50,785B DRAIN-observer section only ~8k tokens of it relevant) | ~17k |
| BC-2.03.001 (12,392B) + BC-2.01.008 (14,900B) — exact PC/Invariant wording for the wire format | ~7k |
| Tool-call overhead (Read/Glob/Grep envelopes, ~10%) | ~7k |
| **Total** | **~81k** |
| Agent context window | 200K (Sonnet-class) |
| **Budget usage** | **~41%** |

### Pass 2 — Failing-test pass (test-writer)

| Context Source | Estimated Tokens |
|---|---|
| This story spec (full — all 18 ACs' postcondition/test-name blocks, BC anchors) | ~28k |
| BC-2.03.001, BC-2.03.002, BC-2.01.008 (full — exact PC/EC wording to test against) | ~9k |
| ARCH-03 §Session Discovery (v1.8, ~88 lines) + relevant VP-080/VP-044/VP-045 proof-method sections | ~7k |
| Stub code from Pass 1 (8 new-file skeletons/signatures across `internal/hmac`, `internal/routing`, `internal/discovery`, `cmd/switchboard`, `internal/testenv`) | ~5k |
| Tool-call overhead | ~8k |
| **Total** | **~57k** |
| Agent context window | 200K (Sonnet-class) |
| **Budget usage** | **~29%** |

### Pass 3 — TDD implementation pass (implementer)

| Context Source | Estimated Tokens |
|---|---|
| This story spec (full) | ~28k |
| Stub code + failing-test content from Pass 1/2 (8 stub files + ~36 test functions across 8 new/extended test files implied by the story's cited test names — line-count estimate, not yet written) | ~14k |
| Production files being extended, before-state (`discovery.go` full 21k B, `mgmt_wire.go` DRAIN-observer + `wireMetricsHandlers`/`wireRouterControlHandlers` registration sections ~10k B, `advertisement_hmac.go` full, `hmac.go` full, `testenv.go` `NewLoopback` section) | ~13k |
| SEC-DW-01..09 constraint text (rulings doc Implementation Constraints section, re-read during hop-1 hardening tasks) | ~6k |
| BC/error-taxonomy spot-checks during implementation (exact literals already encoded in Pass-2 failing tests; lighter touch) | ~4k |
| Tool-call overhead (heaviest pass — most edits + `just test-race` cycles across 8 new files) | ~10k |
| **Total** | **~75k** |
| Agent context window | 200K (Sonnet-class) |
| **Budget usage** | **~38%** |

**Overall:** no pass breaches the 60% split-discussion threshold. Pass 1 (~41%) is the heaviest —
driven by the rulings doc's own size (87KB, three full rulings plus the Security Consult Addendum
and Decision Log) being unavoidable binding-source context for the stub pass. All three passes stay
comfortably under half the window. No story split required at 8 points; Task 6's GATED status
further reduces effective Pass-3 load if it is deferred to a follow-on burst once
`S-BL.NODE-IDENTIFY-WIRE` lands (Tasks 1-5 alone would be materially lighter than the full 8-point
estimate above).

## Task Breakdown (Strict TDD — Stubs → Red → Green → Gate)

All tasks execute in a single worktree on a feature branch cut from `develop@HEAD`. Each task gate
is `just test-race` green + `just lint` clean before proceeding to the next. **Tasks 1-5 (hop-1
ingest, sender dispatch, hop-2 frame construction) are independently deliverable and do not depend
on `S-BL.NODE-IDENTIFY-WIRE` landing. Task 6 (hop-2 fan-out dispatch) is explicitly GATED — see its
own section below.**

### Task 1 — Key-derivation plumbing (AC-004)

Red: write AC-004 tests against stub/no-op `DeriveDiscoveryKey`/`DiscoveryAuthKeyFor`. Green:
implement `HKDFInfoDiscovery` + `hmac.DeriveDiscoveryKey`, `routing.DiscoveryAuthKeyFor` +
`routing.DeriveDiscoveryKey`; delete `advertisementKey`; update `discovery.go`'s
`Encode`/`Decode` call sites. Gate: `just test-race`, `just lint`.

### Task 2 — Router-side ingest path: authentication + replay gate + hardening (AC-005..AC-013)

Red: write AC-005 through AC-013 tests against a stub ingest function that always rejects. Green:
implement `internal/discovery/discovery_wire.go` — fixed-offset extraction (SEC-DW-01), HMAC-first
verification with unified reject sentinel (SEC-DW-01/05), the `Sequence`/`lastSeen` replay-discard
map (SEC-DW-07), bounded read buffer + re-derived `maxSessionsPerAdvertisement` (SEC-DW-02),
aggregate rate cap + `FailureCounter` visibility reuse (SEC-DW-03), rate-limited failure logging
(SEC-DW-04). Depends on Task 1. Gate: `just test-race`, `just lint`.

### Task 3 — Router-mode multicast listener + sender-side dispatch (AC-001, AC-002, AC-003)

Red: write AC-001/AC-002/AC-003 tests against a stub listener/sender. Green: implement
`discovery.MulticastAddrFor`; implement `cmd/switchboard/discovery_wire.go`'s router-mode-exclusive
listener wiring (register-before-serve, mirroring `wireMetricsHandlers`/
`wireRouterControlHandlers`); extend `discovery.go`'s `Run()`/`Advertise` for real `WriteTo`
dispatch with TTL=1; implement `testenv.MulticastLoopbackInterface`. Depends on Task 2 (the
listener dispatches into the ingest path). Gate: `just test-race`, `just lint`; confirm VP-044/VP-045
integration tests now exercise real loopback multicast (Forward Obligation (b) discharge signal).

### Task 4 — Node-local relay-ingest: retire `ReceiveAdvertisement`, add relocated `ErrSVTNMismatch` guard (AC-007)

Red: write a new unit test for the node-side relay-ingest function's `OuterHeader.SVTNID` vs.
`d.cfg.LocalSVTNID` equality check. Green: delete `Discovery.ReceiveAdvertisement`; implement the
new node-side relay-ingest function (decode hop-2 payload, no HMAC, relocated SVTN check, registry
replace-on-write). Rewrite or delete the ten test functions F-DWSP8-001 identifies (File-Change
List row above) — `TestDiscovery_VP045_SVTNIsolation_MultipleScopes` retired outright; the rest
rewritten against the router-side admitted-node model per AC-005/AC-006's setup pattern. Depends
on Task 2 (router-side decode) and Task 5 (hop-2 frame-assembly payload shape). Gate:
`just test-race`, `just lint` — no regression in the *surviving* test set (the ten identified tests
are expected to change).

### Task 5 — Hop-2 relay frame construction (AC-014, AC-015, AC-016)

Red: write AC-014/AC-015/AC-016 tests against a stub frame-assembly function. Green: implement
`cmd/switchboard/discovery_relay_wire.go`'s frame-assembly half (`control_type=0x03` payload
layout, zero `HMACTag`, re-serialized-not-raw payload). This task is fully independent of Task 6 —
frame assembly is a pure function taking decoded fields and producing bytes; it needs no live
connection or fan-out mechanism to test. Depends on Task 2 (consumes the ingest decision's decoded
fields). Gate: `just test-race`, `just lint`.

### Task 6 — [GATED — depends_on S-BL.NODE-IDENTIFY-WIRE] Hop-2 fan-out dispatch (AC-017, AC-018)

**This task's Green step MUST NOT be scheduled until `S-BL.NODE-IDENTIFY-WIRE` lands** (the fan-out
target-resolution companion story named by the 2026-07-14 human gate disposition — see rulings v1.10
item (j)). Red-step tests (AC-017/AC-018) MAY be written now against an injected connection-set stub
(e.g. a `func(svtnID [16]byte, excludeNodeAddr [8]byte) []nodeConn` seam) so the fan-out
**semantics** (SVTN-scoped, exclude-originator, best-effort non-blocking, ~1/sec rate cap) are
pinned by tests independent of the resolution mechanism — but the Green step that wires a *real*
connection-set lookup into that seam depends on `S-BL.NODE-IDENTIFY-WIRE`'s
`Router.BindInterface`-shaped method landing first; this task's Green step is a follow-on burst
against that story's shipped API, not part of this story's own delivery.

Gate: `just test-race`, `just lint` — deferred until the gate clears.

### Task 7 — Quality gate

```sh
just fmt
just lint
just test-race
```

All packages pass. Zero lint warnings. Then open PR targeting `develop` — Task 6's inclusion in
that PR depends on whether `S-BL.NODE-IDENTIFY-WIRE` has landed by delivery time; if not, Tasks 1-5
alone constitute a complete, mergeable PR delivering hop-1 ingest end-to-end, with Task 6 tracked as
a follow-on.

## Delivery Plan Note — POL-005

Any adversarial or evaluation dispatch for this story (per-story pass, wave-gate Perimeter-2, or
any other evaluation dispatch) **MUST embed the POL-005 (`adversary-dispatch-integrity`, HIGH)
verification tuple** in the dispatch prompt — `{repo path, branch, expected HEAD SHA at dispatch
time, artifact IDs + versions under review}` — per `.factory/policies.yaml` POL-005. The dispatched
agent's first action must verify its observed `git rev-parse HEAD` and artifact versions against
the tuple before proceeding; on mismatch, it must ABORT the pass and report the divergence as the
pass result rather than reviewing stale state.

## Anchors Consumed

| Anchor | Verbatim ID | Source | Disposition |
|--------|-------------|--------|--------------|
| Admitted-node HMAC key derivation (DRIFT-W6TBD-001) | BC-2.03.001 v1.6 Postcondition 5 | Ruling 1 | TO DISCHARGE — AC-004, AC-005, AC-006 |
| SVTN-scoped multicast address derivation | BC-2.03.001 v1.6 Precondition 3 | Ruling 2 | TO DISCHARGE — AC-002 |
| Router-relay delivery model (DI-004 compliance) | BC-2.03.001 v1.6 Postcondition 1, Invariant 1 | Ruling 2 | TO DISCHARGE — AC-001, AC-003 |
| Replay/freshness (SEC-DW-07) | BC-2.03.001 v1.6 Postcondition 2; VP-080 | Ruling 1 | TO DISCHARGE — AC-008, AC-009, AC-010 |
| Ingest resource/rate hardening (SEC-DW-01..05) | BC-2.03.001 v1.6 Postcondition 5 | Ruling 1 (Security Consult Addendum) | TO DISCHARGE — AC-005, AC-006, AC-011, AC-012, AC-013 |
| Hop-2 relay transport, frame layout | BC-2.01.008 v1.2 Postcondition 2, Postcondition 3, Invariant 5 | Ruling 3(a)(c) | TO DISCHARGE — AC-014 |
| Hop-2 connection-trust boundary | BC-2.03.001 v1.6 Postcondition 1 delivery-mechanism note | Ruling 3(b) | TO DISCHARGE — AC-015 |
| Hop-2 fan-out semantics + rate cap | BC-2.03.001 v1.6 Postcondition 1 delivery-mechanism note; SEC-DW-09 | Ruling 3(d)(e) | GATED — AC-017, AC-018 (Human Gate item 3) |
| Fan-out target resolution | Ruling 3(f) | Ruling 3 | NOT DISCHARGEABLE by this story alone — Forward Obligation (a) |
| DRIFT-W6TBD-001 | drift item | `RULING-W6TB-D-discovery-scope.md` | RESOLVED by AC-004/AC-005/AC-006 — tag PR with `Resolves: DRIFT-W6TBD-001` per this repo's non-`closes`/`fixes` convention for prior-architect-note-reported items |

## Provenance

- **Origin:** `RULING-W6TB-D-discovery-scope.md` — real-socket wire transport and admitted-node
  HMAC key derivation deferred from S-7.02 to this backlog stub (v1.0, 2026-07-01).
- **v1.1 fix-burst:** F-P4L3-03 added RULING-W6TB-H's Scope Constraints (HMAC-first ordering,
  `payload.SVTNID`-based key derivation, sentinel ordering) to the stub.
- **Adjudication:** `.factory/decisions/S-BL.DISCOVERY-WIRE-rulings.md` v1.3 (2026-07-13) — all
  three Open Design Obligations resolved (Ruling 1 key derivation, Ruling 2 address derivation +
  router-relay model, Ruling 3 hop-2 relay transport, added same day after Ruling 2 surfaced the
  hop-2 gap). This elaboration (v2.0) is the story-writer transcription of that ruling into
  sprint-ready ACs, per the ruling's own framing ("It does not edit the story... those edits
  belong to the product-owner / story-writer").
- **Status:** stays `draft`, not `ready` — three items require human/PO disposition before wave
  scheduling (see Human Gate section).

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 2.13 | 2026-07-15 | **Pre-adjudication cascade from `S-BL.DISCOVERY-WIRE-rulings.md` v1.10** — two Green-step implementation-time findings, dispatched by team-lead ahead of the Step-4.5 adversarial loop, verified independently against `feature/S-BL.DISCOVERY-WIRE`. **Ruling 4 (new Forward Obligation (e)):** `wireDiscoveryListener` is fully implemented and independently tested but not called from `runRouter` — the router process has no source of "which SVTN(s) am I serving" (`admission.AdmittedKeySet` has no SVTN-enumeration method; the only production `RegisterKey` caller runs in a separate, disconnected control-mode OS process; `runRouter`/`runConsole`/`runControl` each construct their own permanently disconnected, empty `AdmittedKeySet`). AC-001 **ACCEPTED at function level** (same "implemented + tested, integration deferred for a verified reason" shape as `ReceiveAdvertisement`'s defense-in-depth reframe and AC-017/AC-018's GATED treatment) but Postcondition 1's literal text overclaims daemon-runtime behavior that does not exist today — a new "Scope note (Ruling 4, v1.10, 2026-07-15)" paragraph inserted after AC-001's BC Anchor line, before Postconditions, names the gap and points to the new Forward Obligation (e) row and the new follow-on story **`S-BL.ADMISSION-SYNC-WIRE`** (working name — not multi-option-vetted like `S-BL.NODE-IDENTIFY-WIRE` was, not yet created). Forward Obligations table gained row (e): the obligation lands on neither existing candidate — "S-6.02, rc.1 gate" only fixes who can call `admin.key.register` against the control daemon, not how that write reaches router-mode's separate `AdmittedKeySet`; `S-BL.NODE-IDENTIFY-WIRE` is not a home either — worse, it is ITSELF blocked by this same gap, since `admission.AdmitNode` is verification-only (`ks.keys[svtnID]` lookup, `ErrNotAdmitted` if absent) and will fail unconditionally against a router process's always-empty keyset regardless of the NODE_IDENTIFY handshake shipping exactly as v1.9 ruled. Summary sentence below the Forward Obligations table updated: "(b)/(c) remain open" → "(b)/(c)/(e) remain open"; "None of (b)/(c)/(d) ever blocked" → "None of (b)/(c)/(d)/(e) ever blocked", with a clause noting (e) is a retrospective, implementation-time finding accepted at function level with a qualifying Scope note, not a `[GATED]` marker. **Ruling 2 addendum (elaboration, sanctioned within Ruling 2's existing scope — no new decision, no BC change, no points change):** AC-003 postcondition 1 rewritten in place — sender-side multicast egress now fans out once per UP+multicast-capable local interface (`net.ListenUDP` + `WriteToUDP`, each pinned via `setsockopt IP_MULTICAST_IF`), not a single `net.WriteTo`/`net.DialUDP` call as originally specified; elaborated empirically during Task 3's Green step because `239.0.0.0/8` does not reliably route to every interface on multi-homed dev hosts (matches the mDNS/SSDP multi-homed pattern). Postconditions 2 and 3 unchanged. DI-004/Invariant 1 (no group join) and TTL=1 (SEC-DW-08) both preserved per-send-per-interface. Mechanical version-pin sweep: every live-prose `rulings v1.9` pin → `v1.10` at 15 spots (the `inputDocuments:` comment, Status-note blockquote, Decision-section intro, Decision 1's Node-local-ingest-correction citation, AC-007's qualifying note, both Architecture Compliance Rules rows, AC-017's Gate paragraph, Forward Obligations row (a), row (d)'s paren-form AND its "item (l)" citation, the Human Gate intro blockquote's subsection citation, items 1/2/3's disposition-blockquote citations — "item (k)" ×2, "item (j)" — and Task 6's body); the sanctioned historical `inputDocuments:`-comment narration citing "the v1.9 story-ready human gate disposition" (a running-history record of WHEN that disposition landed, alongside the equally-preserved v1.6/v1.7/v1.8 citations in the same sentence) left unchanged, consistent with this story's established historical-preservation precedent. `input-hash` recomputed via `compute-input-hash --update`: rulings changed on disk again since v2.12's computation (v1.9→v1.10) — `f5135e6` → `8bdbc57`. `acceptance_criteria_count` stays 18; points stay 8. **`status` stays `ready`** — Ruling 4 and the Ruling 2 addendum are retrospective, implementation-time findings dispatched ahead of the Step-4.5 adversarial loop, not new Human Gate items; they do not reopen the story-ready disposition (v1.9's human-gate content, including the Option 1 selection for `S-BL.NODE-IDENTIFY-WIRE`, is explicitly not reopened per the ruling's own framing). Frontmatter `version` 2.12 → 2.13, new `modified:` entry added. |
| 2.12 | 2026-07-14 | **Story-ready human gate disposition burst — status promoted `draft` → `ready`.** All three items carried by the Human Gate section received human sign-off/selection on 2026-07-14, transcribed verbatim from `S-BL.DISCOVERY-WIRE-rulings.md` v1.9's new "Ruling 3(f) Forward Obligation, SEC-DW-07, and the discovery port — human gate disposition" subsection (items (j)/(k)/(l)) and `S-BL.DISCOVERY-WIRE-fanout-options.md` v1.1's Disposition section. **Item 1 (SEC-DW-07):** APPROVED as documented — both residual bounds (Case 1 ≤1s same-wall-clock-second crash-loop; Case 2 ≈N backward host-clock adjustment) accepted with no further hardening requested. **Item 2 (discovery UDP port):** `49201` ADOPTED — no longer a placeholder; Decision 2(c)'s port sentence rewritten from "Recommended... bikeshed-level placeholder... Flagged for human sign-off" to "Adjudicated: `49201`... adopted by human gate disposition 2026-07-14". **Item 3 (fan-out target resolution):** NEITHER originally-offered option (i) (unnamed sequencing dependency) nor option (ii) (narrow story-local seam with no identity signal) selected — both rejected; the human asked for better options, `S-BL.DISCOVERY-WIRE-fanout-options.md` v1.0 (six additional options) was produced and reviewed, and the human selected its **Option 1**: a new, immediately-named, immediately-scheduled companion story, **`S-BL.NODE-IDENTIFY-WIRE`** (created this burst, `stories/S-BL.NODE-IDENTIFY-WIRE.md` v1.0, draft, wave backlog, 0 ACs — models the `S-BL.SESSIONS-LIST-WIRE` backlog-stub shape), delivers the `control_type=0x04` `NODE_IDENTIFY` handshake wiring `admission.AdmitNode`/`admission.GenerateChallenge` over the live connection and records `(SVTNID, NodeAddr) → IfaceID` via a new `Router.BindInterface`-shaped method. Frontmatter `depends_on` gained `S-BL.NODE-IDENTIFY-WIRE`; AC-017/AC-018/Task 6 now gate on it by name. Human Gate section gained a top-of-section DISPOSITIONED blockquote plus per-item disposition blockquotes appended to items 1/2/3 (verbatim architect-supplied text). Forward Obligations table: row (a) replaced with a RESOLVED disposition citing `S-BL.NODE-IDENTIFY-WIRE`; row (d) (the `sessions.list` obligation, already resolved to `S-BL.SESSIONS-LIST-WIRE` — see v4.109 STORY-INDEX entry — and `BC-2.03.002.md` already re-pointed to v1.5) also updated to RESOLVED status for consistency; the table's summary sentence rewritten to state both (a) and (d) are resolved, (b)/(c) remain open non-blocking. AC-017's heading tag changed `[GATED — see Forward Obligation (a) / Human Gate item 3]` → `[GATED — depends_on S-BL.NODE-IDENTIFY-WIRE]`, Gate paragraph rewritten to name the story by ID; AC-018's heading tag changed identically, its Gate paragraph left as-is (references AC-017's). Task 6 gating text re-anchored by name across the three explicitly-authorized sections: Token Budget's "Overall" paragraph, the Task Breakdown section intro, Task 6's own heading + body (the option (i)/(ii) branching collapsed to the single adopted mechanism, since the choice is no longer open), the Architecture Compliance Rules relay-dispatch row's Enforcement cell, and Task 7's PR-inclusion note — all now read `S-BL.NODE-IDENTIFY-WIRE` by name instead of "Human Gate item 3"/"Forward Obligation (a)". **Four residual `Forward Obligation (a)`/`Human Gate item 3` references found OUTSIDE this burst's authorized touch-list** (the Non-Goals fan-out-binding bullet, EC-008's edge-case description, the File-Change List's `discovery_relay_wire_test.go` row, and Anchors Consumed's two hop-2 rows) are deliberately left unedited — flagged for a follow-on sweep, not silently expanded into. Mechanical version-pin sweep: every live-prose `rulings v1.8` pin → `v1.9` at 7 spots (`inputDocuments:` comment, Status-note blockquote, Decision-section intro, Decision 1's Node-local-ingest-correction citation, AC-007's qualifying note, both Architecture Compliance Rules rows) plus the one paren-form pin inside Forward Obligations row (d); every live-prose `BC-2.03.002 v1.4` pin → `v1.5` at its one live spot (the `inputDocuments:` comment, description text updated to note the PC-1 re-point to `S-BL.SESSIONS-LIST-WIRE` already executed on disk). `input-hash` recomputed via `compute-input-hash --update`: two of this story's five declared `inputs:` changed on disk since the last computation (rulings v1.8→v1.9, BC-2.03.002 v1.4→v1.5) — `a39b7ad` → `f5135e6`. `acceptance_criteria_count` stays 18; points stay 8. Frontmatter `version` 2.11 → 2.12, new `modified:` entry added. **`status: draft` → `ready`** — the Human Gate section's intro conditioned promotion on exactly these three sign-offs; all three landed. `wave` stays `backlog` (unscheduled — promotion is a readiness signal, not a wave assignment). |
| 2.11 | 2026-07-14 | Remediated spec-adversarial pass 13 finding F-DWSP13-001 (LOW): two live-prose spots attributed the F-DWSP4-001 restart-liveness amendment (`Sequence` `uint32`→`uint64` epoch-qualified widening + offset consequences) to "rulings v1.6" — the canonical adoption version is rulings v1.5 (rulings.md's restart-liveness amendment section is a v1.5 adjudication; v1.6 was only the residual-bounds precision correction, changing no widths or offsets), corroborated by BC-2.03.001 PC-2's blockquote ("F-DWSP4-001, v1.5"), VP-080's history, and this story's own line 1270 ("v1.5 update (F-DWSP4-001), residual bounds corrected v1.6"), which directly contradicted line 1257 within the same Human Gate item 1. Fixed: AC-005's note (line 909) "widths updated per F-DWSP4-001/rulings v1.6" → "...v1.5"; Human Gate item 1's opening sentence (line 1257) "epoch-qualified per the v1.6 restart-liveness amendment" → "...v1.5 restart-liveness amendment". Line 1270's correct dual-version phrasing left unchanged. **Exception-set retirement:** prior sweeps (v2.7-v2.10 rows) classified these two spots as "sanctioned point-in-time historical `rulings v1.6` citations" and verified them unchanged burst after burst — that classification protected the version-at-fix-time reading without checking the text's actual semantic claim, which falsely attributed the AMENDMENT itself to v1.6; this fix retires that exception class. The v2.7-v2.10 rows' own historical classification text is left unedited (historical-preservation precedent) — this entry layers the correction forward. **Convention going forward:** amendment attributions cite the ADOPTING version (v1.5); "residual bounds corrected v1.6" is the correct dual-version formula where the v1.6 precision correction is also relevant (as line 1270 already does). Mandatory multiline-tolerant re-certification sweep (Perl `-0777`, the ratified v2.9/v2.10 pattern set plus the paren-form `VP-080` check) found a THIRD spot making the identical false attribution: "The architect's fix (rulings v1.6): widen `Sequence` to `uint64`, epoch-qualified..." (Human Gate item 1, four lines below the correctly-phrased v1.5/v1.6 dual-version paragraph opener) — the widening fix itself credited to v1.6 rather than v1.5. Initially flagged out of scope pending disposition; **the orchestrator extended this same burst to cover it** (class-sweep principle — fix every instance of a finding's class in one burst rather than leaving a guaranteed pass-14 finding). Fixed in place, same v2.11 (uncommitted, amended not re-versioned): "The architect's fix (rulings v1.6):" → "The architect's fix (rulings v1.5; residual bounds corrected v1.6):" — the same dual-version formula line 1270 already uses; nothing else in the sentence changed. **All THREE same-class spots (AC-005's note at line 909, Human Gate item 1's opening sentence at line 1257, and Human Gate item 1's fix-origin sentence at line 1274) are now fixed in v2.11.** Re-ran the full re-certification sweep after this third fix: every live-prose `VP-080` hit reads `v1.7` (5 spots); every live-prose `rulings` hit reads `v1.8` (6 spots, plus the paren-form at line 1241) or the exempt Provenance "Adjudication:" bullet (`v1.3`, line 1643) — **zero live-prose `rulings v1.6` pins remain anywhere in the file, in any form.** The exception-set retirement now FULLY holds. `acceptance_criteria_count` stays 18; points stay 8. None of this story's five declared `inputs:` (rulings, BC-2.03.001, BC-2.03.002, BC-2.01.008, ARCH-03) changed this burst: `compute-input-hash --check` confirms `a39b7ad` holds unchanged. |
| 2.10 | 2026-07-14 | Cascade-only pin sweep for F-DWSP12-001 (LOW, spec-adversarial pass 12): VP-080's Source Contract process-status paragraph corrected by the architect (it had read "drafted, not yet executed" since the v1.0 mint even though the BC-2.03.001 PC-2 amendment landed at v1.5 and was superseded in place at v1.6) — citation-only, no property-substance change, input-hash unchanged (`5d904d5`) — and bumped to v1.7; VP-INDEX bumped to v2.47. Story-side pin sweep updated all six live-prose `VP-080 v1.6` pins → `v1.7`: the `inputDocuments:` comment, AC-009's note, AC-010 postconditions 5/6, the Non-Goals `uint64`-composite-wraparound bullet, and the File-Change List's `discovery_wire_test.go` row (swept forward — the v1.7 fix touched only the Source Contract paragraph, leaving the surviving-lineage citation this row references unchanged and current). Mandatory multiline-tolerant re-certification sweep (Perl `-0777`, the ratified v2.9 pattern set plus a paren-form `VP-080` check): every live-prose `VP-080` hit now reads `v1.7`; every live-prose `rulings` hit already reads `v1.8` with the two sanctioned historical exceptions (AC-005, Human Gate item 1) and the Provenance bullet (`v1.3`) verified unchanged; zero paren-form `VP-080` hits found. `acceptance_criteria_count` stays 18; points stay 8. None of this story's five declared `inputs:` changed this burst: `compute-input-hash --check` confirms `a39b7ad` holds unchanged. |
| 2.9 | 2026-07-14 | Remediated spec-adversarial pass 11 finding F-DWSP11-001 (LOW): the Forward Obligations table row (d) at line 1175 read "**None of the three rulings (v1.3) adjudicate this**" — a straggler from the v2.0 elaboration (rulings was v1.3 at that time; it is v1.8 now). The paren-separated form `rulings (v1.3)` is structurally invisible to every prior sweep's pattern (`` rulings(\.md)?[`']?\s+v1\.[0-9]+ `` requires the version token immediately after the name, not parenthesized) — the third sweep-blind-spot sub-class surfaced on this story, after F-DWSP6-001's line-wrap survivor (v2.5) and F-DWSP10-001's retired-test exemplar cascade (v2.8). The claim's truth is unaffected — no ruling after v1.3 adjudicates the `sessions.list` RPC question; Rulings 1/2/3's scope remains exclusively the UDP-multicast advertisement transport — hence LOW, not MED/HIGH. Fixed: `v1.3` → `v1.8` at line 1175, cell text otherwise unchanged. **Correction to the v2.8 row's completeness claim:** that entry asserted "every live-prose `rulings` hit already reads `v1.8` except the two intentional point-in-time historical `rulings v1.6` citations" — true under the sweep pattern in force at the time, but that pattern was blind to the paren form, so the claim did not in fact cover every live-prose hit. The v2.8 row itself is left unedited per this story's historical-preservation precedent; this row layers the correction alongside it. **New sweep standard for this story (F-DWSP6-001/F-DWSP10-001/F-DWSP11-001 countermeasure lineage):** the mandatory multiline-tolerant Perl `-0777` re-certification sweep now also runs a paren-tolerant pattern, `rulings\s*\(v1\.[0-9]+\)`, alongside the existing `` rulings(\.md)?[`']?\s+v1\.[0-9]+ `` and `VP-080\s+v1\.[0-9]+` patterns. This burst's extended sweep (both rulings patterns plus the VP-080 pattern, whole file as one buffer) found exactly one paren-form hit — the line-1175 fix above — and reconfirmed every other live-prose `rulings` hit already reads `v1.8` and every live-prose `VP-080` hit already reads `v1.6`, with the same two sanctioned point-in-time historical `rulings v1.6` citations (AC-005's F-DWSP1-001 fix-context note; Human Gate item 1's SEC-DW-07-fix-origin note) and the Provenance "Adjudication:" bullet (`rulings.md` v1.3, correctly documenting the v2.0 elaboration's authority set at the time) verified unchanged. A broader case-insensitive paren check for both `rulings` and `VP-080` near any parenthesized version number confirmed no further paren-form hits of either kind exist anywhere in the file. `acceptance_criteria_count` stays 18; points stay 8. None of this story's five declared `inputs:` (rulings, BC-2.03.001, BC-2.03.002, BC-2.01.008, ARCH-03) changed this burst: `compute-input-hash --check` confirms `a39b7ad` holds unchanged. |
| 2.8 | 2026-07-14 | Remediated spec-adversarial pass 10 finding F-DWSP10-001 (MED): three live spots cited the retired `TestDiscovery_VP045_SVTNIsolation_MultipleScopes` as an EXTANT exemplar to extend — a propagation gap the F-DWSP8-001 retirement (v2.6) never fully cascaded. Architect fixed VP-080's two spots (v1.6: the Proof Method table's Tool cell and the Feasibility Assessment's Proof-complexity Notes cell, both re-cited to the surviving router-side `DiscoveryAuthKeyFor`-admitted lineage) — citation-only, no property-substance change, input-hash unchanged (`5d904d5`, none of VP-080's three declared inputs changed) — and bumped VP-INDEX to v2.46. The third spot was this story's own File-Change List line 1353 (`discovery_wire_test.go` row), which contradicted the adjacent line-1354 row's "retired outright, not extended" framing by still describing the row as extending the retired test's family. Fixed: line 1353's cell text replaced verbatim with the architect's supplied blockquote — now frames AC-005/AC-006 as establishing the admitted-node/SVTN `DiscoveryAuthKeyFor`-admitted test-setup pattern that VP-080 v1.6 cites as the surviving lineage after the retired test's outright retirement, resolving the 1353/1354 contradiction. Story-side pin sweep updated all five live-prose `VP-080 v1.5` pins → `v1.6`: the `inputDocuments:` comment (gained a new v1.6 historical clause; the prior "v1.5 is a citation re-pin..." clause tense-shifted to "v1.5 was..." per this entry's own established pattern), AC-009's note ("VP-080 v1.5 Test Scenario 5"), AC-010 postcondition 5 ("VP-080 v1.5 property 4"), AC-010 postcondition 6 ("VP-080 v1.5 Property 5"), and the Non-Goals `uint64`-composite-wraparound bullet. Mandatory multiline-tolerant re-certification sweep (Perl `-0777` over the whole file as one buffer, patterns `VP-080\s+v1\.[0-9]+`, `` rulings(\.md)?[`']?\s+v1\.[0-9]+ ``, `TestDiscovery_VP045_SVTNIsolation_MultipleScopes`): every live-prose `VP-080` hit now reads `v1.6`; every live-prose `rulings` hit already reads `v1.8` except the two intentional point-in-time historical `rulings v1.6` citations (AC-005's F-DWSP1-001 fix-context note; Human Gate item 1's SEC-DW-07-fix-origin note) — verified correctly unchanged, consistent with every prior sweep this session; every remaining retired-test mention (the `inputDocuments:` rulings/VP-045 comments, AC-004 postcondition 5's qualifying note, AC-007's test names, Task 4) already correctly frames the test as retired/historical — verified, zero further drift found beyond the one line-1353 fix. `acceptance_criteria_count` stays 18; points stay 8. VP-080 is NOT one of this story's five declared `inputs:` (rulings, BC-2.03.001, BC-2.03.002, BC-2.01.008, ARCH-03), and none of those five changed this burst: `compute-input-hash --check` confirms `a39b7ad` holds unchanged. |
| 2.7 | 2026-07-14 | Remediated spec-adversarial pass 9 finding F-DWSP9-001 (MED): fix-burst 6 (pass-8, F-DWSP8-001) bumped the rulings doc v1.7→v1.8 but dropped VP-080's established re-pin-on-every-rulings-bump cascade (the v2.3→v2.4 / rulings v1.6→v1.7 / VP-080 v1.3→v1.4 precedent). Architect cascaded VP-080 v1.4→v1.5 (citation re-pin + input-hash refresh only — Properties 1-5, Test Scenarios, and thresholds unchanged, no property-substance change) and VP-INDEX to v2.45; this row is the story-side pin sweep. Updated all live-prose `VP-080 v1.4` pins → `v1.5` at the five spots certified complete by the v2.5 mandatory re-certification sweep: the `inputDocuments:` comment, AC-009's note ("VP-080 v1.4 Test Scenario 5"), AC-010 postcondition 5 ("VP-080 v1.4 property 4"), AC-010 postcondition 6 ("VP-080 v1.4 Property 5"), and the Non-Goals `uint64`-composite-wraparound bullet (the F-DWSP6-001 line-wrap-survivor spot, re-verified with a whitespace-tolerant edit). Mandatory multiline-tolerant re-certification sweep (Perl `-0777` over the whole file as one buffer, patterns spanning newlines: `VP-080\s+v1\.[0-9]+`, `` rulings(\.md)?[`']?\s+v1\.[0-9]+ ``, `VP-INDEX\s+v2\.[0-9]+`): every live-prose `VP-080` hit now reads `v1.5`; every live-prose `rulings` hit already reads `v1.8` (Status-note blockquote, Decision-section intro, Decision 1's full-rationale citation, both Architecture Compliance Rules rows, AC-004's F-DWSP8-001 qualifying note, the `inputDocuments:` comment) — no rulings drift found this burst. The two point-in-time historical citations pinned to `rulings v1.6` (AC-005's F-DWSP1-001/F-DWSP4-001 fix-context note; Human Gate item 1's SEC-DW-07-fix-origin note) and the `VP-INDEX v2.43` citation embedded in the `VP-080` `inputDocuments:` comment (documents the VP-INDEX version active when Property 5 was added at VP-080 v1.3, not a current-state claim) verified as correctly unchanged — consistent with every prior sweep this session. `acceptance_criteria_count` stays 18; points stay 8. VP-080 is NOT one of this story's five declared `inputs:` (rulings, BC-2.03.001, BC-2.03.002, BC-2.01.008, ARCH-03), and none of those five changed this burst: `compute-input-hash --check` confirms `a39b7ad` holds unchanged. |
| 2.6 | 2026-07-14 | Remediated spec-adversarial pass 8 finding F-DWSP8-001 (HIGH): AC-004's `advertisementKey` deletion structurally broke the VP-045 test AC-007 mandated "passes unmodified" — rooted in rulings Implementation Constraint 3's false claim ("`ReceiveAdvertisement` preserved unchanged in shape"), now corrected. Architect adjudication: `Discovery.ReceiveAdvertisement` is RETIRED (deleted, not preserved) — one of THREE `advertisementKey` call sites (`Encode`/`Decode`/`ReceiveAdvertisement`), not two; a node has no key to derive for an arbitrary sender (BC-2.03.001 v1.6 PC-5). Replaced by a new node-side relay-ingest function: decodes the hop-2 `DISCOVERY_RELAY` payload, no per-frame HMAC (trust = the admitted connection, AC-015), `ErrSVTNMismatch` relocated to a direct `OuterHeader.SVTNID` vs. `d.cfg.LocalSVTNID` equality check, same registry replace-on-write semantics. `TestDiscovery_VP045_SVTNIsolation_MultipleScopes` RETIRED outright, not extended. Six architect blockquotes applied verbatim: (1) AC-004 PC-5 qualifying note — TD-031 deviation: `discovery.go:319`/`:369`/`:399` anchors converted to symbol-only citations (`Encode`/`Decode`/`ReceiveAdvertisement`); (2) AC-007 fully rewritten (title, BC Anchor, all 5 postconditions, test names/level/file); (3) File-Change List's `discovery_test.go` row replaced — widened to the full ten-test disposition, nine rewritten against the router-side `DiscoveryAuthKeyFor`-admitted model plus the VP045-named test retired outright; (4) Task 4 fully rewritten from a verification-only checkpoint to a Red/Green implementation task; (5) Decision 1's `ReceiveAdvertisement` bullet replaced in full; (6) the S-7.02 Previous Story Intelligence anchors-table row's trailing sentence replaced. Mechanical rulings version-pin sweep v1.7→v1.8 (whitespace/multiline-tolerant Perl `-0777` procedure): five live spots fixed — Status-note blockquote, Decision-section intro sentence, both Architecture Compliance Rules rows, `inputDocuments:` rulings comment (gained a parenthetical noting the v1.8 Node-local ingest correction entry). VP-045 `v1.3`→`v1.4` fixed at its one live spot, the `inputDocuments:` comment (description text otherwise unchanged — VP-045 v1.4 corrects only a stale supporting-evidence citation; PARTIAL status and the real-socket PC-3 gap unaffected, confirmed against `VP-045.md` v1.4 directly). Two pre-existing historical "rulings v1.6" parenthetical citations (AC-005's fix-context note; Non-Goals' `Sequence`-widening note) verified as accurate historical citations of the ruling version active at those past fixes, not current-state claims — left unchanged, consistent with every prior sweep this session. BC-2.03.001 confirmed by architect to need NO amendment this pass. `acceptance_criteria_count` stays 18 (AC-007 rewritten in place, not added/removed); points stay 8. `compute-input-hash --update` re-run (rulings v1.7→v1.8 is the only declared-input change; VP-045 is not one of this story's five declared `inputs:`): `eccbdc4` → `a39b7ad`. |
| 2.5 | 2026-07-13 | Remediated spec-adversarial pass 6 finding F-DWSP6-001 (MED): the v2.4 sweep's completeness claim ("all live-prose `VP-080 v1.3` pins updated to `v1.4`") was FALSE — one instance survived because it line-wrapped across the ID/version boundary (`VP-080` at end of one line, `v1.3` at the start of the next), and the v2.4 sweep used single-line-based matching that cannot see across a wrap. Location: the Non-Goals section's `uint64`-composite-wraparound bullet. Fixed: `v1.3` → `v1.4`. Layers a correction onto the v2.4 row's claim rather than editing it — the v2.4 historical entry is left untouched. **Mandatory re-certification sweep, whitespace/multiline-tolerant (DRAIN story's F-SP19-001 countermeasure):** Perl regex over the whole file as one buffer for `VP-080\s+v1\.[0-9]+`, `` rulings\.md['`]?\s+v1\.[0-9]+ ``, `BC-2\.03\.001\s+v1\.[0-9]+`, `VP-INDEX\s+v2\.[0-9]+`, `ARCH-03\s+v1\.[0-9]+`, `BC-2\.03\.002\s+v1\.[0-9]+`, `BC-2\.01\.008\s+v1\.[0-9]+`; classified every hit by line as live-prose or history-layer (frontmatter `modified:`, Provenance "Adjudication:" bullet, body Changelog rows exempt); a second markdown-tolerant pass found no additional hits. Result: every live-prose hit already reads the current version — `VP-080`→`v1.4` at 5 spots (inputDocuments comment, AC-009 note, AC-010 postconditions 5/6, this Non-Goals bullet); `rulings.md`→`v1.7` at 5 spots (inputDocuments comment, Status-note blockquote, Decision-section intro, two Architecture Compliance Rules rows); `BC-2.03.001`→`v1.6` at inputDocuments comment + 7 Anchors Consumed rows (unchanged this burst, no bump reported); `VP-INDEX`/`ARCH-03`/`BC-2.03.002`/`BC-2.01.008` are static inputDocuments-only references, no bump reported. Zero further stale live pins found. `acceptance_criteria_count` stays 18; points stay 8. `compute-input-hash --check`: no declared `inputs:` file changed this burst — `eccbdc4` holds, confirmed clean. |
| 2.4 | 2026-07-13 | Pass-5 fix-burst cascade: the rulings doc bumped v1.6→v1.7 (F-DWSP5-001, a one-token propagation fix to Ruling 3(c)'s trailing prose — `byte[18:]`→`byte[22:]` — no ruling content change) and VP-080 bumped v1.3→v1.4 (housekeeping, alongside ARCH-07/ARCH-11 gaining the VP-078/079/080 rows). No story content changed — citation/hash refresh only. Updated the same five live-prose rulings-version-pin spots fixed at v2.3 (Status-note blockquote, Decision-section intro sentence, two Architecture Compliance Rules rows, `inputDocuments:` comment) v1.6→v1.7. Updated all live-prose `VP-080 v1.3` pins → `v1.4`: `inputDocuments:` comment, AC-009's note, AC-010 postconditions 5/6. Verified (not assumed) this story's own Decision 3(c) diagram and AC-014 postcondition 2 do NOT carry the stale `byte[18:]` sessions offset the rulings v1.7 fix corrected upstream — both already read `byte[22:]`/`bytes 22+` from the v2.3 mechanical sweep. Left UNCHANGED per the historical-preservation precedent: all `modified:`/Changelog entries' own historical version citations (including the v2.3 entry's own "rulings v1.6"/"VP-080 v1.3" narrative, which correctly describes that burst's authority set at the time) and the Provenance "Adjudication:" bullet. `acceptance_criteria_count` stays 18; points stay 8. `compute-input-hash --update` re-run (rulings v1.7 changed on disk): `cd82f7b` → `eccbdc4`. |
| 2.3 | 2026-07-13 | Remediated spec-adversarial pass 4 finding F-DWSP4-001 (HIGH): SEC-DW-07's original in-memory-counter `Sequence` field, paired against the router's restart-STABLE `lastSeen` watermark, produced up to ~8.3h of silent discovery lockout on every ordinary access-node restart/redeploy/crash-recover. Architect fix (rulings v1.6): `Sequence` widened `uint32`→`uint64`, epoch-qualified (high 32 bits = wall-clock seconds sampled at `Discovery`-instance start, low 32 bits = the original counter). Five architect blockquotes applied verbatim: AC-008 gained a scope-clarification note (cold-start vs. stable-identity restart); AC-009 gained a note characterizing its discard rule as correct for genuine replay plus two bounded restart residuals (EC-010 cases 1/2); AC-010 postcondition 5 replaced (`uint32`→`uint64` composite wraparound-out-of-scope) and postcondition 6 added (restart forward-acceptance path) with new test `TestVP080_DiscoveryIngest_RestartForwardProgress`; new EC-010 Edge Cases row; Human Gate item 1 gained an appended v1.6 sign-off paragraph. Deviation flagged: supplied blockquotes cited "VP-080 v1.2" — corrected to "VP-080 v1.3" in both AC-010 postconditions as applied. Mechanical sweeps (rulings v1.6-authoritative): hop-1 `[4]Sequence`→`[8]Sequence`, 38→42-byte full-valid-frame minimum in AC-005's note (raw-min-32/body-min-24 pre-lookup guard left UNCHANGED per explicit instruction); hop-2 `DISCOVERY_RELAY` payload in Decision 3(c) diagram + AC-014 postcondition 2 — `Sequence` byte[12:16]→byte[12:20], count byte[16:18]→byte[20:22], sessions byte[18:]→byte[22:], fixed pre-session header 18→22 bytes. Additional live-prose corrections: Human Gate item 1's original `Sequence uint32` claim, Non-Goals' `uint32` wraparound bullet, File-Change List's `discovery.go` row — all → `uint64`/epoch-qualified. Version-pin sweep: every live-prose citation of rulings and VP-080 with an explicit version number → v1.6/v1.3 (frontmatter `inputDocuments:` comments, Status-note blockquote, Decision-section intro, two Architecture Compliance Rules rows); extended one step beyond the literal instruction to also bump `BC-2.03.001.md`'s `inputDocuments:` comment (v1.5→v1.6, input-version metadata, not prose). **Amended in place, same v2.3 (post-pass-4, pre-commit):** the orchestrator identified this row's own initially-flagged "candidate for a future sweep" as the same stale-version-pin class pass 1 flagged on VP-080 (F-DWSP1-002) — pass 5 would have re-flagged it. Fixed now, in place: all live-prose `BC-2.03.001 v1.5` citations → `v1.6` (Anchors Consumed table's 7 rows; Decision 3(c)'s "BC-2.03.001 PC-5 (v1.5, already landed)"; Decision 3(h)'s "v1.5's PC-5 sentence"). Left UNCHANGED, matching the v2.1/v2.2 historical-preservation precedent: all `modified:`/Changelog entries' own historical version citations (the v2.0 row's "BC-2.03.001 v1.5" citation was transiently corrupted by an overbroad `replace_all` during this same amendment and restored to `v1.5`), and the Provenance "Adjudication:" bullet (anchored to "This elaboration (v2.0)"). No further BC-2.03.001 version-pin instances remain outside historical entries. `acceptance_criteria_count` stays 18 (no AC added/removed); points stay 8. `compute-input-hash --update` re-run (rulings AND BC-2.03.001 both changed on disk): `c321c05` → `cd82f7b`; this same-v2.3 amendment is prose-only and re-verified via `compute-input-hash --check` (unaffected, still clean). |
| 2.2 | 2026-07-13 | Remediated spec-adversarial pass 3 finding F-DWSP3-001 (MED, sibling of F-DWSP1-001's off-by-N-bytes threshold-miscount class). AC-005 Postcondition 3 stated the HMAC computation covers "the complete raw body bytes, not merely the 16-byte key-selector prefix" — the key-selector region per SEC-DW-01 is `SVTNID` `body[0:16]` (16 bytes) + `NodeAddr` `body[16:24]` (8 bytes) = 24 bytes, so naming both fields and then citing "16-byte" undercounted the region by the `NodeAddr` contribution, contradicting this same AC's own Postcondition 1 and Postcondition 4 (both correctly state the 24-byte selector arithmetic). Fixed to: "not merely the 24-byte key-selector prefix (SVTNID 16 + NodeAddr 8)". Class sweep performed across the full story for every remaining "16-byte"/"24-byte"/"8-byte"/"32-byte"/"38-byte" occurrence and the hop-2 `DISCOVERY_RELAY` payload arithmetic (4-byte control header + 8-byte `NodeAddr` + 4-byte `Sequence` + 2-byte count = 18 bytes before the variable session list): no other miscounted field-group arithmetic found — the sole "16-byte" occurrence in the entire story was this AC-005 PC-3 sentence, now fixed; Architecture Compliance Rules row (line ~1018) already uses count-free phrasing and was left unchanged per the orchestrator's explicit scoping; the F-DWSP1-001 changelog/modified-entry history quotes of the old wrong wording were left unchanged as they document a prior finding's text, not a live claim. Sibling fix on the architecture side: `.factory/decisions/S-BL.DISCOVERY-WIRE-rulings.md` (declared input #1) was independently corrected by the architect to v1.4 for the same error class; `compute-input-hash --update` re-run to pick up that change: `b4e0a5f` → `c321c05`. No AC added or removed (still 18); points unchanged (8). |
| 2.1 | 2026-07-13 | Remediated spec-adversarial pass 1 finding F-DWSP1-001 (HIGH, off-by-8-bytes threshold bug, CWE-770/400 class). AC-005 Postcondition 4 stated the router-side ingest guard rejects a raw datagram "shorter than 24 bytes (insufficient for the fixed key-selector fields)" — that threshold omits the 8-byte HMAC tag prefix and contradicts this story's own Decision 1/SEC-DW-01 wire layout (`SVTNID` at raw bytes 8-24, `NodeAddr` at raw bytes 24-32), the rulings doc's Implementation Constraint 2, and the shipped `internal/discovery/discovery.go` layout (`[8]HMACTag \| [16]SVTNID \| [8]NodeAddr \| ...`). A literal implementation would leave a 24-31-byte raw-datagram window that passes the length guard and then indexes out of bounds pre-authentication — exactly the resource-exhaustion/pre-auth-crash class SEC-DW-01 exists to close. Fixed to: "A raw datagram shorter than 32 bytes (8-byte HMAC tag + the 24-byte SVTNID/NodeAddr key selector) — equivalently, whose post-tag body is shorter than 24 bytes — is rejected before any key lookup is attempted." Added an explanatory note directly under AC-005 clarifying the `body` = post-tag-bytes convention this AC and Decision 1 Implementation Constraint 2 both use, and noting the full valid-frame minimum with the SEC-DW-07 `Sequence` field is 38 bytes (8 tag + 16 + 8 + 4 + 2 count) — but the pre-lookup guard only needs raw ≥ 32 / body ≥ 24, since `Sequence`/count are parsed by `decodeBody()` only after HMAC verification succeeds. Swept the full story for any other occurrence of the wrong 24-byte-raw threshold (test names, Task Breakdown, Edge Cases, File-Change List) — none found; AC-005 Postcondition 4 was the sole occurrence. No AC added or removed; `acceptance_criteria_count` (18) and `points`/`estimated_points` (8) unchanged. `input-hash` unchanged (`b4e0a5f`) — none of the five declared `inputs:` files were touched; `compute-input-hash --check` confirms no drift. |
| 2.0 | 2026-07-13 | Elaborated from backlog stub (v1.1, draft, 0 ACs) to sprint-ready draft (v2.0, 18 ACs, 8 points) per architect ruling `S-BL.DISCOVERY-WIRE-rulings.md` v1.3 (all three rulings). Replaced "Open Design Obligations"/"Scope"/"Scope Constraints" with "Adjudicated Design Decisions" (three decisions, one per ruling, load-bearing constraints transcribed inline) plus a new "Design Constraint: Router-Mode Discovery Wiring" section. 18 ACs traced to BC-2.03.001 v1.5 (Preconditions 1-3, Postconditions 1-5, Invariants 1-3), BC-2.03.002 v1.4 Postcondition 5, BC-2.01.008 v1.2 (Postconditions 2-3, Invariants 3/5), and VP-080 v1.0's four property clauses — covering hop-1 ingest (router multicast join, sender TTL=1 dispatch, HMAC verify with SEC-DW-01 fixed-offset extraction, SEC-DW-07 sequence gate incl. cold-start-accepts-first, registry update, SEC-DW-02/03/04 resource/rate hardening) and hop-2 relay (DISCOVERY_RELAY frame assembly, zero-HMACTag connection-trust boundary, fan-out semantics, SEC-DW-09 rate cap). AC-017/AC-018 (hop-2 fan-out dispatch + rate cap) explicitly marked GATED on Ruling 3(f)'s verified fan-out target-resolution Forward Obligation; Task Breakdown structured so Tasks 1-5 (hop-1 + hop-2 frame construction) are independently deliverable and Task 6 (fan-out dispatch) is explicitly gated, deferrable to a follow-on burst. Corrected the v1.1 stub's mislabeled citation: the multicast-address-allocation requirement is BC-2.03.001 **Precondition 3**, not Postcondition 1 as the stub's Open Design Obligation #2 stated — fixed throughout this elaboration's prose. Added dedicated "Human Gate — Story-Ready Sign-off Required" section carrying forward three items flagged by the architect for explicit human/PO disposition before `ready` promotion: (1) the SEC-DW-07 monotonic-`Sequence`-field design (architect already ruled it in, flagged prominently per the ruling's own text), (2) the discovery UDP port number (`49201` recommended, explicit bikeshed), (3) the fan-out target-resolution Forward Obligation's two resolution paths (option (i) sequencing dependency on a future node-identity-to-connection-binding story — architect's recommended default — vs. option (ii) a narrow story-local `Router.BindInterface` seam, with the architect's own caveat that (ii) still requires some connection-time identity signal and only relocates the gap rather than eliminating it). Status stays `draft` — NOT promoted to `ready` — pending these three sign-offs; `wave` stays `backlog`. `bc_traces`/`behavioral_contracts` gained `BC-2.01.008` (hop-2 registry consumer, already landed v1.2); `vp_traces`/`verification_properties` gained `VP-080` (SEC-DW-07 replay-rejection, minted v1.0 ahead of this elaboration). Frontmatter conformed to the `S-BL.CLI-SURFACE-COMPLETION`/`S-BL.LOOPBACK-FULLSTACK` template-mandated superset keys (`epic_id`, `inputs`, `input-hash`, `traces_to`, `behavioral_contracts`, `verification_properties`, `target_module`, `estimated_days`, `assumption_validations`, `risk_mitigations`); both `points`/`estimated_points` set to 8 (this fleet carries both field-name conventions across existing stories; both populated for compatibility). Full File-Change List (14 rows), Architecture Mapping, Purity Classification, Architecture Compliance Rules, Forbidden Dependencies, Library & Framework Requirements (zero new third-party dependencies), Token Budget Estimate (3-pass, none over 41%), Task Breakdown (7 tasks, Task 6 GATED), Forward Obligations table, Anchors Consumed, and POL-005 Delivery Plan Note added. `input-hash` computed via `compute-input-hash --update`. |
| v1.1 | 2026-07-01 | F-P4L3-03: add RULING-W6TB-H to changed_by_rulings; add Scope Constraints section specifying HMAC-first ordering preservation, `payload.SVTNID`-based key derivation, and sentinel ordering requirements when replacing in-process paths with UDP dispatch. |
| v1.0 | 2026-07-01 | Backlog stub created per Ruling W6TB-D. Full decomposition deferred to Wave-7 planning. |
