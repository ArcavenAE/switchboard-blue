---
artifact_id: S-BL.PE-RECEIVE-LOOP-placement-note
document_type: architect-placement-note
story_id: S-BL.PE-RECEIVE-LOOP
title: "PE-connection receive/forward loop placement, frame-type design, arqsend wiring, and E-FWD-001 discharge for S-BL.PE-RECEIVE-LOOP"
status: final
producer: architect
timestamp: 2026-07-10T00:00:00Z
version: "1.22"
bc_traces:
  - BC-2.02.008   # PC-3/EC-003 E-FWD-001 exhaustion (postcondition 1 re-anchored from S-7.04-FU-PE-CONNECTOR AC-004)
  - BC-2.06.003   # PC-1 Failed-state observable via retransmit-driven path exhaustion
vp_traces: []    # no VP ownership in this story; VP-037 unblock path runs through S-7.04-FU-DRAIN-WIRE
forward_obligations_consumed:
  - FO-PE-LOOP-001   # Define frame.FrameTypePEConnect / adopt FrameTypeCtl; flip dialLoop bootstrap
subsystems: [deployment-operations, transport-layer]
architecture_modules:
  - cmd/switchboard
  - internal/upstreamdial
  - internal/routing
  - internal/frame
  - internal/multipath
  - internal/netingress
  - internal/testenv
---

## Changelog

| Version | Change |
|---------|--------|
| 1.0 | Initial release. Full backtick-symbol sweep (Appendix A) performed prior to publication; all symbols verified against tree at `8eb54a5` (S-7.04-FU-PE-CONNECTOR merge SHA). |
| 1.1 | Remediate five spec-adversarial pass-1 findings: F-SP1-001 (HIGH [spec-defect]) — new Q8 ruling specifies FrameArrivalHandler-based wiring with full dependency construction; F-SP1-002 (HIGH [spec-gap]) — Q3 blast-radius enumeration for Valid() widening (test amendments + doc-comment updates); F-SP1-003 (HIGH [spec-gap]) — Q3 adds ARCH-02 frame_type table amendment obligation; F-SP1-005 (MED [spec-gap]) — Q6 strengthened with explicit per-reconnect-iteration join requirement; F-SP1-006 (MED [doc-drift]) — Q1 contradiction with Q2 annotated with explicit supersession. Appendix A updated with new symbols from Q8. |
| 1.2 | Remediate four spec-adversarial pass-2 findings: F-SP2-001 (CRITICAL [spec-defect]) — new Q9 ruling supersedes Q4/Q5 injection topology: arqsend `Dispatch` must NOT dial `ListenAddr`; the upstream fixture MUST write directly to the accepted PE connection; option (b) ruled (fixture assembles + writes frame directly; arqsend obligation audited and narrowed); F-SP2-002 (HIGH [spec-gap]) — Q9 specifies write-capable upstream fixture shape, placement (test-local, same file as other runRouter integration tests), and exact API (accepted-conn handle + `WriteFrame(wire []byte) error` method); F-SP2-003 (MED [spec-defect]) — Q9 mandates harness rule: every AC asserting OnFrameArrival must use the real runRouter goroutine pattern (not testenv.Restart which bypasses SetFrameCallback); F-SP2-004 (MED [doc-drift]) — Q3 blast-radius amended with two missed frame_test.go locations (`TestParseOuterHeader_AcceptsAllValidFrameTypes` "all five" comment and 5-element `valid` slice). Adjudicated-clean section added for five pass-2 non-findings (per F-SP2-001 report). Appendix A delta added for new fixture symbols. |
| 1.3 | Remediate three spec-adversarial pass-3 findings: F-SP3-001 (HIGH [spec-defect]) — Q2 byte-contract contradiction resolved: `frame.ReadOuterFrame` MUST return payload-only (consistent with `netingress.ReadFrame` precedent); receive goroutine reconstructs full frame via `frame.EncodeOuterHeader`+append before invoking `FrameFn`; Q2 false claim about netingress.ReadFrame retracted; FrameFn `raw` parameter pinned as full outer-header+payload; AC-002/AC-004 false-duplicate pinning test shape specified. F-SP3-002 (HIGH [spec-gap]) — AC-005 harness re-attributed to hand-rolled flap harness in `connector_test.go` per existing heldConn+Close() pattern; `peWriteFixture` de-attributed from AC-005 in FCL row 7. F-SP3-003 (MED [doc-drift]) — `OuterHeader.FrameType` field comment ("data, ctl, arq, fec, empty-tick") adjudicated item-8; Q3 blast-radius table updated to 8 locations; extended sweep transcript included. Appendix A delta for `frame.EncodeOuterHeader` (reuse from v1.0 verification). Pass-3 adjudicated-clean section added. |
| 1.4 | Remediate two spec-adversarial pass-4 findings: F-SP4-001 (HIGH [spec-gap]) — Q2 FrameFn discrimination contract amended with binding return-value rule: non-nil FrameFn return MUST NOT terminate the receive loop; discard-and-continue (`_ = frameFn(hdr, raw)` idiom) mandated, mirroring ServeConn's `continue` pattern; exit-on-error form explicitly forbidden; receive-goroutine sketch updated. F-SP4-002 (HIGH [spec-gap]) — Q1/Q8 amended with SetFrameCallback ordering contract: MUST be called before Start(); `frameFn` field is set-once pre-launch (goroutine-creation happens-before covers visibility; no field synchronization required); production wiring order in runRouter = construct → SetFrameCallback → Start; receive goroutine MAY assume non-nil under this ordering; nil-guard posture: defense-in-depth silent discard added as optional belt-and-suspenders; post-Start callback mutation forbidden. Appendix A delta for v1.4 (no new symbols; prior symbol table remains complete). Pass-4 adjudicated-clean section added. |
| 1.5 | Remediate three spec-adversarial pass-5 findings: F-SP5-001 (HIGH [spec-gap]) — Q2 receive-loop READ-error disposition specified: on ANY non-nil error from `frame.ReadOuterFrame` the receive goroutine MUST exit the loop (return); `continue`-on-read-error FORBIDDEN (exact mirror of v1.4 callback-error return-FORBIDDEN rule; per-site disposition follows `netingress.ServeConn` precedent — read error → exit, callback error → continue); receive-goroutine sketch updated with explicit `if err != nil { return }` branch; logging disposition specified; AC-005 read-error-exit pin test added (name: `TestConnector_ReceiveLoop_ExitsOnReadError`; +1 test to connector_test.go estimated count). F-SP5-OBS-1 (LOW [spec-divergence]) — bounded-read/read-deadline divergence from netingress precedent: accepted with rationale (`PayloadLen` is `uint16` ≤64 KB/frame allocation bound; upstream is configured/semi-trusted; LimitReader would add overhead without meaningful DoS protection on a point-to-point dialed connection). F-SP5-OBS-2 (LOW [spec-completeness]) — connector_test.go frame-injection mechanism clarified: the AC-005 hand-rolled flap harness (`heldConn`+`Close()`) established in connector_test.go provides the write pattern; AC-001/AC-003 unit tests reuse the same in-package fixture pattern (`outerassembler.Assemble` usage as shown); no new shared helper is created. Appendix A delta for v1.5 (no new symbols introduced). Pass-5 adjudicated-clean section added. |
| 1.6 | Remediate four spec-adversarial pass-6 findings: F-SP6-001 (HIGH [spec-defect]) — v1.5 "exit → dialLoop's existing teardown/reconnect path" claim corrected: `maintainConn` returns only on write failure / keepalive-probe-assembly failure / `SetWriteDeadline` failure / `stopAddr` close — it never reads the conn and cannot observe receive-goroutine exit; receive goroutine MUST call `_ = conn.Close()` on read-error exit to convert the failure into a write-side event that causes `maintainConn` to return; lifecycle contract amended to add conn.Close() as second receive-goroutine output (alongside FrameFn callback); double-close is safe/idempotent on net.Conn; reconnect latency bound stated; backoff behaviour after established-then-failed cited; pin-test `TestConnector_ReceiveLoop_ExitsOnReadError` timeout guidance added; malformed-frame reconnect-storm risk assessed as bounded by keepaliveInterval only (backoff resets on success). F-SP6-002 (HIGH [spec-gap]) — Handle-interface blast radius for `SetFrameCallback`: RULED Option A — `SetFrameCallback` stays OFF the `upstreamdial.Handle` interface; `runRouter` calls it on the concrete `*Connector` between `New()` and `Start()` (concrete type available there); `fakeConnectorHandle` (router_pe_connector_test.go:75–81) is NOT affected; router_pe_connector_test.go remains "existing, unmodified"; Q1 Q-block text updated. F-SP6-003 (MED [spec-defect]) — AC-001 PC-3 `connector.Mode()` and AC-004 precondition `ModePE` poll unassertable under real-runRouter harness (runRouter holds connector as unexported local): amended to use observable substitutes — `peWriteFixture.accepted` channel receipt confirms PE establishment; `"mode=PE"` writer-output line (existing `waitForConnections`/`scanForLine` pattern) is the poll substitute; `connector.Mode()` assertions remain valid only in connector_test.go unit tests (in-package, concrete type). F-SP6-004 (LOW [doc-drift]) — blast-radius table undercounts: `frame_test.go:501` and `:540` stale-comment locations added as items 9 and 10; count corrected 8 → 10; `:540` edit specified to cover both the range `{0x01..0x05}`→`{0x01..0x06}` AND the "canonical five"→"canonical six" text. Appendix A delta for v1.6 (no new symbols). Pass-6 adjudicated-clean section added. |
| 1.7 | Remediate five spec-adversarial pass-7 findings (2026-07-09). F-SP7-001 (HIGH [spec-defect]) — RETRACT v1.6 F-SP6-003 claims that `"mode=PE"` "fires after connectedCount.Add(1)" and is "the stronger guarantee ... for a strict ModePE assertion"; `"mode=PE"` is emitted in `runRouter`'s startup writer block gated on `len(upstreamRouters)>0` (verified `mgmt_wire.go` :548) and on SIGHUP re-emit (:587), synchronously after `connector.Start()` returns (Start only launches goroutines); no dependency on `connectedCount` or any established connection; correct establishment observables specified for AC-001 PC-3 and AC-004 precondition. F-SP7-002 (MED [spec-divergence]) — parenthetical in observable-substitute item 1 (AC-001 PC-3) incorrectly stated `accepted` = "completed step 3 (atomically incrementing connectedCount)"; corrected: `accepted` fires at TCP-accept time, strictly BEFORE `connectedCount.Add(1)` (bootstrap Write at :350 precedes Add(1) at :365); single-correct-semantics statement added, folded into F-SP7-001 corrected-observables block. F-SP7-003 (MED [spec-divergence]) — Candidate-FCL connector.go row (:1677) and Summary-of-Rulings Q1 row (:1692) retained stale "to `Handle` interface" / "`Handle` gains `SetFrameCallback(fn FrameFn)` seam" wording contradicting the binding F-SP6-002 Option A ruling; both swept to Option A language (method on concrete `*Connector` ONLY; Handle interface unchanged; `fakeConnectorHandle` unaffected); full grep sweep performed with patterns `"Handle gains"`, `"to Handle"`, `"Handle.*SetFrame"`, `"Add FrameFn type.*Handle"` — grep patterns and hit counts recorded in body. F-SP7-004 (LOW [doc-drift]) — story Task 1 cites note "v1.2"; story-writer propagation item noted; cross-reference version-pin policy ruled. F-SP7-005 (LOW [spec-completeness]) — transient stale-ModePE window (after receive-goroutine `conn.Close()` exit, before `maintainConn` write failure decrements `connectedCount`) acknowledged; bounded by `keepaliveInterval`; no AC obligation. Pass-7 adjudicated-clean section added. Appendix A delta for v1.7 (no new symbols). F-SP7-003 sweep completed on audit: two additional Q1-body residuals (:76 'gains a method', :90 'gains a setter') struck — initial 4-pattern transcript was insufficient; expanded pattern set and corrected hit counts recorded in the sweep-transcript section. |
| 1.8 | Remediate two spec-adversarial pass-10 findings (2026-07-10). F-SP10-001 (MED [doc-drift]) — Q4 and Q5 supersession banners added at the top of each section body; both were the only superseded sections lacking in-place annotation; amendment is annotation-only (no ruling content changed; Q9 and the corrected-observables block from F-SP7-001 already governed — this makes the supersession visible at point-of-read). F-SP10-002 (LOW [doc-drift]) — note frontmatter `architecture_modules` corrected: added `internal/frame` and `internal/multipath` (the modules Q2/Q3/Q8 centre on); dropped `internal/arqsend` (removed from the story's touch-list by Q9.4). Story v1.9 untouched. |
| 1.9 | Remediate two spec-adversarial pass-11 findings (2026-07-10). F-SP11-001 (HIGH [spec-defect]) — Q2 AC-005 `TestConnector_ReceiveLoop_ExitsOnReadError` injection recipe replaced: the v1.5-era "single byte 0xFF as FrameType" recipe is physically unrealizable (io.ReadFull blocks on < 44 bytes) and mis-attributes the error (0xFF at byte[0] triggers ErrVersionMismatch, not ErrInvalidFrameType); corrected recipe mandates a complete 44-byte outer header with byte[0]=0x01 (valid version byte, VersionMajor=0, VersionByte=0x01, verified frame.go :21/:23), byte[1]=0x07 (out-of-range frame_type, one above FrameTypePEConnect=0x06 upper bound), PayloadLen=0x0000 at bytes[2:4] big-endian (verified frame.go EncodeOuterHeader :90), remaining bytes zero; conn NOT closed; io.ReadFull completes; ParseOuterHeader returns ErrInvalidFrameType at byte[1]; receive goroutine exits via read-error branch → conn.Close() → maintainConn write failure → reconnect. Optional variant (adjudicated: ADD as separate pin) for ErrVersionMismatch path: byte[0]=0xFF (major nibble 0xF ≠ VersionMajor 0) → ErrVersionMismatch → same exit contract; named `TestConnector_ReceiveLoop_ExitsOnVersionMismatch`. F-SP11-002 (LOW [token-budget]) — story-side, handled by story-writer; not touched in this note. F-SP11-003 (LOW [doc-drift]) — §8.2 dangling "see elaboration note below" clause struck; production interface-set population is out of scope for this story; §8.5 governs the test-scoped set. |
| 1.10 | Remediate one spec-adversarial pass-12 finding (2026-07-10). F-SP12-001 (MED [spec-completeness]) — Q2 ARCH-08 obligation block extended with a second explicit edit obligation: in the same §6.5 row edit, the ARCH-08 row's parenthetical "frame is NOT imported directly; reachable transitively through outerassembler and halfchannel. Corrected from v2.6 {frame, outerassembler} per adversary pass-1 F-P1-001." MUST be reconciled — the F-P1-001 correction was accurate for its time (no direct frame import existed then) and is now partially superseded by this story's legitimate direct edge; replacement wording specified in Q2. Import-set count reclassified: the ARCH-08 parenthetical is a distinct import-edge-prose location; blast-radius count extended from 10 to 11 (unified count). Pass-12 confirmations recorded in new adjudicated-clean section. |
| 1.11 | Remediate one spec-adversarial pass-13 finding (2026-07-10). F-SP13-001 (MED [spec-completeness]) — §6.6.2 sibling of F-SP12-001: the §6.6.2 upstreamdial forbidden-edges bullet carries the same three stale import-set claims as the §6.5 parenthetical remediated in v1.10. Q2 ARCH-08 obligation extended with a THIRD edit target (§6.6.2 upstreamdial bullet, three sub-edits in the same commit as the §6.5 edits); binding replacement bullet text specified in Q2. Blast-radius count ruling amended: 11 → 12 (10 frame sweep + 2 ARCH-08 locations: §6.5 row parenthetical + §6.6.2 forbidden-edges bullet). Class-closure grep performed (patterns `"halfchannel, outerassembler"` and `"F-P1-001"`): 0 further occurrences beyond §6.5 (line 325) and §6.6.2 (lines 458–465) edit targets plus 2 changelog rows (history-preserved, not edited). Pass-13 confirmations recorded in new adjudicated-clean section; transcript corrected on audit: `"halfchannel, outerassembler"` has 4 hits (line 316 = `internal/arqsend` row, benign substring match on a different package's own import set — initially uncounted). |
| 1.12 | Remediate one spec-adversarial pass-14 finding (2026-07-10). F-SP14-001 (MED [spec-completeness]) — BC-2.01.004:61 (Postcondition 2 outer-header layout table, frame_type row) is cited zero times in story or note despite carrying the byte-identical enum row as ARCH-02:74; post-ship, BC-2.01.004 would enumerate 5 frame types while ARCH-02 + frame.go Valid() accept 6. Remediation option (a): Q3 ARCH-02 amendment obligation extended with a binding sibling obligation for BC-2.01.004:61 — must be amended to `, pe_connect=0x06` in the SAME commit as ARCH-02:74 and FrameTypePEConnect. Before/after rows quoted verbatim. Rationale cites F-P8-008 co-canonical precedent (BC-2.01.004:57 + ARCH-02:74 named canonical pair in pass-8) and BC-2.01.004 v1.2 sync-practice. Blast-radius arithmetic: BC-2.01.004:61 is a wire-format spec pair partner to ARCH-02:74 (both same-commit parallel obligations, sibling of but not inside the unified-12 count); total stated as "unified 12 (10 frame sweep + 2 ARCH-08 locations) + wire-format spec pair (ARCH-02:74 + BC-2.01.004:61, same-commit parallel obligations)". Class-closure grep performed (patterns `"arq=0x04, fec=0x05"` and `"empty_tick=0x02"`): exactly 2 canonical locations each (BC-2.01.004:61 and ARCH-02:74); transcript recorded with disposition of all hits. Pass-14 adjudicated-clean section added. |
| 1.13 | Remediate one spec-adversarial pass-17 finding (2026-07-10). F-SP17-001 (MED [spec-gap / test-set underdetermination]) — AC-003 discrimination contract (discard PEConnect, forward everything else) pinned at only two test points: forward side tested with FrameTypeData only, discard side with FrameTypePEConnect only; a whitelist-data-only implementation passes all ~11 named tests while silently dropping FrameTypeCtl frames required by Non-Goals (S-BL.RESYNC-FRAME consumer). New BINDING unit test added to Q2: `TestConnector_ReceiveLoop_CtlFrameForwardedToCallback` in `internal/upstreamdial/connector_test.go` — assembles a complete valid frame with `FrameType: frame.FrameTypeCtl`, uses same in-package accept-and-write fixture family as `PEConnectFrameDiscarded`, asserts FrameFn IS invoked and `hdr.FrameType == frame.FrameTypeCtl`; kills the whitelist-data-only malicious implementation. Companion cosmetic fix recorded: story's discrimination-sketch else-branch comment enumerates `{data, ctl, arq, fec}` but empty_tick also traverses the forward branch — comment must gain empty_tick (story-writer applies). Test counts updated: connector tests 6 → 7 (minimum; with optional ExitsOnVersionMismatch: 7); total net-new ~11 → ~12 (1 frame_test + 7 connector_test + 4 integration). Pass-17 adjudicated section added: F-SP17-001 accepted (one pin test + comment enumeration fix + counts 7/~12); P1b concurrency clean (OnFrameArrival hitCountMu + DropCache mu verified thread-safe, ReloadAddrs set-diff isolation, Stop() stopOnce idempotent), P1c DRAIN-WIRE seam clean (backlog story, illustrative ACs, no concrete API expectation), P1d VP traceability clean (no VP pins a 5-type enum or Valid() bound; vp_traces:[] correct), P2 POL pass, P3 DataFrameForwarded + FlapCycleJoin re-executed realizable. |
| 1.14 | Remediate one spec-adversarial pass-18 finding (2026-07-10). F-SP18-001 (MED [spec-gap / test-set underdetermination]) — discard-side loop-continuation unpinned: `TestConnector_ReceiveLoop_PEConnectFrameDiscarded` asserts only "FrameFn NOT invoked"; nothing asserts the connection stays open and reading continues after the discard. Malicious implementation `if hdr.FrameType == FrameTypePEConnect { _ = conn.Close(); return }` passes every named test while converting every bootstrap frame into teardown+reconnect. Remediation: EXTEND `TestConnector_ReceiveLoop_PEConnectFrameDiscarded` (extend-not-add; counts unchanged at 7 connector / ~12 total) — on the SAME connection, fixture writes a `FrameTypePEConnect` frame FOLLOWED by a `FrameTypeData` frame; assert (a) FrameFn NOT invoked for the bootstrap frame, (b) FrameFn IS invoked for the subsequent data frame (`hdr.FrameType == frame.FrameTypeData` at the call site). Kills discard-as-close: the close tears down the conn before the data frame is read, failing (b). New BINDING ruling block `AC-003 discard-continuation pin (v1.14 — F-SP18-001)` added in Q2 immediately after the F-SP17-001 block; realizability note included (two frames back-to-back on one conn, `frame.ReadOuterFrame` loops on `io.ReadFull(44)` + PayloadLen reads, length-delimited, segment-boundary-independent). Pass-18 Adjudicated section added: F-SP18-001 accepted (extend-not-add, counts unchanged); P1a Ctl-pin realizability clean (Assemble :102 FrameType passthrough, Valid() 0x03 true, no Ctl special-case before frameFn); P1b kill transcript updated; P1c AC-002/004 count-tolerance clean; P1d note-ruling/story coherence confirmed; P2 POL pass; P3 ExitsOnReadError re-traced realizable. |
| 1.15 | Remediate one spec-adversarial pass-19 finding (2026-07-10). F-SP19-001 (MED [doc-drift / incompletely-discharged prior remediation]) — v1.1 supersession note :110-111: live unannotated Option-B claim ("Q2 also rules that `upstreamdial.Handle` gains `SetFrameCallback(fn FrameFn)`") spans a line break; survived F-SP7-003 sweep because single-line grep patterns cannot match cross-line token pairs; contradicts F-SP6-002 Option A binding ruling and falsely attributes Handle placement to Q2. Residual struck and annotated in the v1.1 supersession note using the standard ~~strikethrough~~ + `*(amended v1.15 — ...)*` pattern. F-SP7-003 sweep transcript extended with v1.15 addendum: root cause recorded (cross-line token pair unreachable by single-line grep), NEW canonical multi-line-tolerant pattern documented (`tr '\n' ' ' \| grep -o "Handle. gains .SetFrameCallback"`), post-fix hit count (7 hits; 2 struck historical, 5 meta-references in documentation) with per-hit dispositions recorded, sweep re-certified zero live unannotated Option-B claims. Pass-19 Adjudicated section added. This is the 6th incomplete-sweep-class instance and the 2nd false sweep-completeness certification. |
| 1.19 | Implementation-phase adversarial adjudication (2026-07-11): F-IP1-001 (MED [missing regression guard + false enforcement claim]) — AC-002 `go list -deps` assertion promised but undelivered; Architecture Compliance Rules "build MUST fail" sentence is factually wrong (upstreamdial→routing is acyclic; Go build does NOT fail). Ruling: standalone perimeter test `TestUpstreamdialImportPerimeter` in `internal/upstreamdial/connector_test.go` with positive-coverage guard (exec `go list -deps`, assert non-empty AND contains `internal/frame`, then assert `internal/routing` absent). Corrected Architecture Compliance wording specified. Forward obligation recorded for mgmt_wire.go:549–551 nil ForwardFunc. |
| 1.21 | Per-story adversarial adjudication round 3 (2026-07-11): F-IP3-001 (MED) — note-side F-IP2-001 Option-b propagation performed: :194-199 implementation-obligation block struck with ~~strikethrough~~ and annotated with caller-obligation wording per :3108; class-closure sweep (4 patterns, multi-line-tolerant) found 0 additional live unannotated occurrences; this is the 9th incomplete-sweep-class instance. OBS-1 (LOW [test-coverage]) — FlapCycleJoin_NoLeak recvWg.Wait() pin gap: ACCEPTED as documented pin-limitation; deterministic recipe impractical without deadlock/flake risk. OBS-2 (LOW [process-gap]) — remediation workflow missing mandatory in-place annotation step: recorded as [process-gap]; countermeasure now binding for remaining passes. |
| 1.22 | Per-story adversarial adjudication round 4 (2026-07-11): F-IP4-001 (MED) — outgoing bootstrap frame_type pin test: RULING (a) REMEDIATE NOW. New test `TestConnector_BootstrapFrameTypePEConnect` added to `internal/upstreamdial/connector_test.go`; recipe: accept one connection on in-package fixture, `io.ReadFull` the 44-byte outer header, `frame.ParseOuterHeader`, assert `hdr.FrameType == frame.FrameTypePEConnect`; positive guard that ReadFull returned nil error; precedent F-SP17-001/F-SP18-001 (same-session remediation of MED receive-side holes); remediation cost minimal; deferral would leave a named deliverable (FO-PE-LOOP-001) revertible for the entire DRAIN-WIRE gap. Connector tests 8→9, total ~13→~14. Checkbox observation: Tasks 1–16 (`:1018–:1033`) marked `[ ]` despite all deliverables verifiably complete at `c3fca02` (commits `c316aed`, `a3d5117`, `e85c9df`, `8e8296c`, `5274cf1`, spec-side `9792605`) — RULING: mark `[x]` in story bump v1.25 citing delivering commits; deliberate convention not identified; simple correct fix. |
| 1.20 | Per-story adversarial adjudication round 2 (2026-07-11): F-IP2-001 (MED) — SetFrameCallback post-Start mutation guard: OPTION (b) RULED — caller-ordering contract alone sufficient; guard not implemented; note records rejected-option rationale (panic vs ignore hides vs surfaces; one production caller with correct ordering; goroutine-happens-before already covers the set-once field; implementing a guard adds complexity without eliminating the data race on concurrent callers). F-IP2-002 (MED) — residual false attribution in `router_pe_receive_test.go:212–217` doc comment: mechanical replacement text specified. F-IP2-003 (LOW) — ARCH-08 v2.11 Changelog table missing 2.11 row: fixed in-place in ARCH-08 (row added below 2.10, no version bump). |
| 1.18 | GREEN-phase adjudication (2026-07-11): F-GP1-001 (HIGH [green-phase contract conflict]) — `TestConnector_BackoffParameters` breaks 3/3 deterministically under the binding F-SP5-001/F-SP6-001 unconditional-close contract. Root cause: silent `SetWriteDeadline` failure path in `maintainConn` emits no EC-001 stamp; test's stamp[0] assumption is violated; measured gap captures doubled backoff (~2 s) not operative base (~1 s). Decision: OPTION (b) — keep unconditional close (production code unchanged), fix test stamp-collection to be robust to both teardown paths via Mode-drop poll before stamp collection. Options (a) (re-opens half-close hole, REJECTED) and (c) (misleading EC-001 log in production, REJECTED) evaluated and rejected. Story propagation: story-writer adds task to apply `TestConnector_BackoffParameters` fix. |
| 1.17 | Remediate one spec-adversarial pass-21 finding (2026-07-10). F-SP21-001 (MED [doc-drift / incomplete sweep-completeness certification]) — v1.16 class-closure sweep table certified "17 blocks … complete" but missed four versioned binding-block headers whose bold text does not match the recorded grep patterns (`binding.:`, `BINDING`, `[Ss]ketch`): :262 `**\`FrameFn\` byte-contract (binding — F-SP3-001 correction):**` (v1.3/F-SP3-001); :511 `**Test shape (binding for story-writer and implementer):**` (v1.13/F-SP17-001 sub-block); :1812 `**Pin test shape (binding for story-writer):**` (Q9/F-SP3-001 byte-contract pinning obligation); :1928 `**Binding harness rule:**` (Q9.3/F-SP2-003 runRouter mandate). All four verified CURRENT (no supersession needed). Sweep table extended to rows 18–21; grep transcript replaced with canonical pattern `grep -nE '\*\*[^*]*[Bb]inding'` (21 hits); v1.17 addendum block added to sweep section re-certifying over all 21 blocks. This is the 8th incomplete-sweep-class instance and the 3rd false completeness certification. Pass-21 Adjudicated section added. |
| 1.16 | Remediate one spec-adversarial pass-20 finding (2026-07-10). F-SP20-001 (MED [doc-drift / incompletely-discharged prior remediation]) — READ-error disposition contract block (:365-421): three-part defect: (1) header :365 lacked the "amended v1.6" supersession marker present on the STORY's equivalent header; (2) prose :386-387 stated the retracted mechanism verbatim ("Exit → dialLoop's existing teardown/reconnect path closes the conn and re-dials, which is the ONLY correct resync") — false per ground truth (maintainConn at connector.go:399 is write-only, never observes receive-goroutine exit); (3) v1.5 sketch :404-421 had a bare `return` with no `_ = conn.Close()` and no in-place warning. Three-part annotation applied: (1) header extended with supersession marker pointing to F-SP6-001 section; (2) prose struck with ~~strikethrough~~ + `*(amended v1.16 — F-SP20-001: RETRACTED ...)*` annotation; (3) banner blockquote inserted above sketch fence directing implementers to the v1.6 binding sketch. Sketch body preserved unmodified (history preservation). Class-closure sweep performed: 17 versioned binding blocks enumerated; 2 newly remediated (rows 4-5), 2 previously annotated (rows 6, 10), 13 fully current; zero unannotated stale binding blocks remain. This is the 7th incomplete-sweep-class instance. Pass-20 Adjudicated section added. |

# Architect Placement Note: PE-Connection Receive/Forward Loop
## Story: S-BL.PE-RECEIVE-LOOP

This note answers seven design questions required to unblock story elaboration
and scheduling. All file anchors refer to the `develop` branch at HEAD `8eb54a5`
(S-7.04-FU-PE-CONNECTOR merge SHA). Rulings are binding for the story-writer
and implementer. Every API derivation block is grep-verified against the tree at
this SHA; see Appendix A for the symbol-sweep disposition table.

---

## Q1 — Receive goroutine ownership: inside `upstreamdial.Connector` vs wiring in `cmd/switchboard`

**Ruling: the receive goroutine lives inside `internal/upstreamdial.Connector`,
one goroutine per established connection, started after step-3 success in
`dialLoop`. `cmd/switchboard/mgmt_wire.go` is not modified for the receive-loop
itself — `runRouter` receives only a callback or channel seam added to the
`upstreamdial.Handle` interface. This keeps `internal/upstreamdial` routing-free
per the forbidden-edge constraint.**

**Derivation:**

ARCH-08 §6.6.2 at `8eb54a5` defines the forbidden edges for `internal/upstreamdial`:

> Forbidden: `drain`, `routing`, `testenv`, and any package at positions 20–23.

`internal/routing` is at position 17. A direct import of `routing` by
`upstreamdial` would be a NEW edge between positions 19 → 17 — numerically
acyclic but **functionally forbidden** because ARCH-08 explicitly lists
`routing` in the forbidden set. The placement note for S-7.04-FU-PE-CONNECTOR
(Q4, §"Forbidden edges") is unambiguous:

> `internal/upstreamdial` → `internal/routing` (routing is a boundary; connector
> does not participate in frame forwarding)

Two implementation shapes preserve this constraint:

**(a) Callback seam:** ~~`Handle` gains a method
`SetFrameCallback(fn func([]byte) error)`~~. *(amended v1.7 — F-SP7-003: superseded by F-SP6-002 Option A; `SetFrameCallback` lives on concrete `*Connector` only, `Handle` interface unchanged)* After step-3 success in `dialLoop`,
the receive goroutine calls `fn` for each raw frame. `runRouter` wires a closure
at construction, passing it to the connector. This is the same pattern
`netingress.ServeConn` already uses for `RouteFn`.

**(b) Channel seam:** the `Connector` exposes a `chan []byte` of received frames;
`runRouter` drains it in its own goroutine. This doubles the goroutine count and
buffers unconsumed frames, adding backpressure complexity for no benefit.

**Ruling: option (a) — callback seam.** It mirrors the existing `netingress.RouteFn`
pattern exactly and keeps the `Connector` as the goroutine owner without importing
`routing`.

**ARCH-08 obligation:** `internal/upstreamdial`'s allowed-import set DOES NOT
change — the callback receives `[]byte` raw frames at the connector boundary,
which requires only stdlib at the connector layer. ~~The `Handle` interface gains a
setter~~; *(amended v1.7 — F-SP7-003: superseded by F-SP6-002 Option A; `SetFrameCallback` lives on concrete `*Connector` only, `Handle` interface unchanged)* the `Connector` struct gains the callback field. Both changes are internal
to the existing registered package. No §6.4 registration is required by this
story.

> **[v1.1 supersession note — F-SP1-006]** The sentence in this Q1 derivation
> that read "No new import row is needed" and described the callback signature as
> `func([]byte) error` is superseded by Q2's ruling. Q2 rules that the connector
> callback signature is `type FrameFn func(hdr frame.OuterHeader, raw []byte) error`
> (not `func([]byte) error`) and that `upstreamdial` gains a direct `frame` import
> (ARCH-08 §6.5 amendment required). ~~Q2 also rules that `upstreamdial.Handle`
> gains `SetFrameCallback(fn FrameFn)` (not `SetFrameCallback(fn func([]byte) error)`).~~
> *(amended v1.15 — F-SP19-001, completing F-SP7-003: F-SP6-002 Option A is binding — `SetFrameCallback` is a method on the concrete `*upstreamdial.Connector` ONLY; the `Handle` interface is unchanged. This clause's Option-B attribution to Q2 is also retracted — Q2 rules the framing primitive (`ReadOuterFrame`, FrameFn signature, frame import), not Handle placement, which is F-SP6-002's domain.)*
> Q1's routing-free constraint, goroutine placement decision, and option (a)
> callback-seam choice remain operative. The specific import and signature details
> are determined by Q2, which is authoritative. The v1.0 Q1 text is preserved here
> per factory history-preservation policy; read Q2 as the controlling specification
> for all type and import details.

**SetFrameCallback interface placement (v1.6 — F-SP6-002, BINDING):**

Pass-6 adversarial review identified that if `SetFrameCallback` were added to the
`upstreamdial.Handle` interface, `fakeConnectorHandle` at
`cmd/switchboard/router_pe_connector_test.go` lines 75–81 (implements only
`ReloadAddrs`/`Mode`/`Stop`) would no longer compile, breaking the two `SetConnector`
call sites at lines ~493 and ~503, and contradicting the story's declaration that
`router_pe_connector_test.go` is "existing, unmodified."

**Ruling: Option A — `SetFrameCallback` is NOT added to the `upstreamdial.Handle`
interface.** The method exists only on the concrete `*upstreamdial.Connector` type.
`runRouter` in `cmd/switchboard/mgmt_wire.go` holds the connector as a concrete
`*Connector` between `New()` and `Start()` and calls `SetFrameCallback` there (on the
concrete type, before the value is ever used as a `Handle`). Post-Start, the connector
is accessible via the `Handle` interface only, which does not expose `SetFrameCallback`
— this is the desired encapsulation: post-construction callback mutation is already
forbidden by the ordering contract.

The `upstreamdial.Handle` interface therefore remains:
```go
type Handle interface {
    ReloadAddrs(addrs []string)
    Mode() ConnMode
    Stop()
}
```
unchanged from its v1.5 shape. `fakeConnectorHandle` in `router_pe_connector_test.go`
compiles without modification. No FCL row is added for `router_pe_connector_test.go`.

**SetFrameCallback ordering contract (v1.4 — F-SP4-002):**

`SetFrameCallback` MUST be called before `Start()`. The `frameFn` field on
`Connector` is set-once pre-launch. The `happens-before` edge created by
`sync` at goroutine creation (Go memory model §"Goroutine creation") guarantees
that any `frameFn` value written before `Start()` is visible to all goroutines
launched by `Start()`. No additional field synchronization (mutex, atomic) is
required for this field, because it is written exactly once before any reader
goroutine is created.

Production wiring order in `runRouter` (binding):

```
construct → SetFrameCallback → Start
```

Concretely in `mgmt_wire.go` (verified: current code has `connector := upstreamdial.New(...)`
immediately followed by `connector.Start()` at `8eb54a5`; this story inserts
`connector.SetFrameCallback(frameFn)` between those two lines):

```go
connector := upstreamdial.New(w, outerassembler.Envelope{}, keepaliveInterval, upstreamRouters)
connector.SetFrameCallback(frameFn)  // MUST precede Start
connector.Start()
```

The receive goroutine MAY assume `frameFn` is non-nil under this ordering contract
(the field is guaranteed visible and non-nil before the goroutine is created).

Nil-guard posture (defense-in-depth): as a belt-and-suspenders guard against
future callers that construct a `Connector` without calling `SetFrameCallback`
before `Start`, the receive goroutine SHOULD apply a nil check before invoking
the callback and silently discard the frame if `frameFn` is nil. This does not
replace the ordering obligation — a nil `frameFn` at receive time is a
programming error, not an expected condition — but it prevents a nil-deref panic
in the face of that error. The discard is silent (no log emission) because a
nil callback implies the caller did not wire up routing; logging every discarded
frame would be noise without context. The nil check has no effect on the
production path, where the ordering contract holds.

~~Post-Start mutation of the callback is forbidden. Any call to `SetFrameCallback`
after `Start()` returns is a data race (dial goroutines are already reading
`frameFn`); the `Connector` implementation MUST NOT permit it. If the
`SetFrameCallback` setter is called post-Start, it may panic or be silently
ignored — the implementer's choice — but it MUST NOT proceed with an unsynchronized
field write.~~ *(amended v1.21 — F-IP3-001: note-side F-IP2-001 Option-b propagation, mandated at :3106 but not performed in v1.20. The implementation obligation is replaced by a caller obligation only:* `SetFrameCallback` *MUST be called before* `Start()`*. Calling it after* `Start()` *returns is a **data race** (dial goroutines are already reading* `frameFn`*); the caller is solely responsible for the ordering. The implementation does not detect or guard against post-Start mutation — the field is set-once and the goroutine-creation happens-before already covers visibility to all goroutines launched by* `Start()`*. See the F-IP2-001 ruling at :3104 for full rationale.)*

**Cite:** ARCH-08 §6.6.2 forbidden-edges note at `8eb54a5`;
`internal/upstreamdial/connector.go` `dialLoop` structure (verified at `8eb54a5`);
`internal/netingress/netingress.go` `RouteFn` pattern (verified at `8eb54a5`);
`cmd/switchboard/mgmt_wire.go` lines 525–526 (`connector := upstreamdial.New(...)`
immediately followed by `connector.Start()` at `8eb54a5` — this story inserts
`SetFrameCallback` between them); Go memory model goroutine-creation happens-before
guarantee.

---

## Q2 — Frame path: framing/deframing mechanism for incoming bytes on a PE connection

**Ruling (v1.3 — supersedes v1.2 byte-contract text): a new function
`frame.ReadOuterFrame(r io.Reader) (OuterHeader, []byte, error)` is added to
`internal/frame/frame.go` (position 2). Like `netingress.ReadFrame`, it returns
`(parsed header, payload-only slice, error)` — the `[]byte` return is
PAYLOAD-ONLY, not a full-frame slice. The receive goroutine then reconstructs the
full frame (outer header + payload) via `frame.EncodeOuterHeader(hdr)` + append
before invoking the `FrameFn` callback. The `FrameFn raw` parameter is therefore
ALWAYS the full frame bytes (outer header + payload), as required by
`OnFrameArrival`'s contract.**

> **[v1.2 retraction — F-SP3-001]** The v1.2 Q2 text contained two false claims.
> First, the sentence describing the `Connector`'s receive goroutine as calling
> `netingress.ReadFrame(conn)` then `frameFn(hdr, payload)` implied `raw` was
> payload-only at the callback boundary. Second, the v1.2 option-3 ruling
> characterised `frame.ReadOuterFrame`'s `[]byte` return as "the same
> read-header-then-payload logic as `netingress.ReadFrame`" and allowed the
> receive goroutine to pass that slice directly to `FrameFn`. Both are retracted.
> The correction: `netingress.ReadFrame` is verified at `8eb54a5` to read the
> 44-byte header into a discarded `hdrBuf [frame.OuterHeaderSize]byte` and return
> `(hdr, payload, nil)` — `payload` is the slice allocated for `hdr.PayloadLen`
> bytes only. There is no full-frame combined slice anywhere in `netingress.ReadFrame`.
> `frame.ReadOuterFrame` MUST have the same signature and semantics (payload-only
> return) so that `netingress.ReadFrame` can optionally delegate to it without
> changing its own return contract. The full-frame reconstruction obligation
> belongs to the receive goroutine at the single call site, not to
> `frame.ReadOuterFrame`. Q2 (v1.1 supersession annotation regarding import/signature)
> remains operative for import and callback-type decisions; the specific
> byte-contract detail is now governed by this v1.3 correction.

**Derivation from the `netingress` API (verified at `8eb54a5`):**

```go
// internal/netingress/netingress.go
// ReadFrame reads exactly one framed message from r: OuterHeaderSize bytes
// followed by hdr.PayloadLen bytes of payload. Returns the parsed header
// and payload slice. The []byte return is payload-only; it does NOT include
// the outer header bytes.
func ReadFrame(r io.Reader) (frame.OuterHeader, []byte, error)
```

`ReadFrame` implementation (verified at `8eb54a5`): reads header bytes into
`var hdrBuf [frame.OuterHeaderSize]byte` (discarded after parsing), then allocates
`payload := make([]byte, int(hdr.PayloadLen))` and reads exactly that many bytes.
Return value is `(hdr, payload, nil)` — the header bytes are not included in the
`[]byte` return.

`ReadFrame` is self-delimiting via `OuterHeader.PayloadLen` (44-byte outer header
carries a `uint16` payload length). It is the canonical framing primitive used by
`netingress.ServeConn` for all incoming connections. The wire format on PE upstream
connections is identical — `outerassembler.Assemble` on the sending side produces a
wire frame consumed byte-for-byte by `ReadFrame` on the receiving side (documented
in `assemble.go`'s package comment at `8eb54a5`).

**`FrameFn` byte-contract (binding — F-SP3-001 correction):**

The `FrameFn` callback parameter `raw []byte` MUST be the full wire frame:
outer header (44 bytes) + payload. This is required because:

1. `OnFrameArrival(frameBytes []byte, ...)` computes its drop-cache key as
   `crc32.ChecksumIEEE(frameBytes)` (verified at `8eb54a5` in
   `internal/routing/on_frame_arrival.go` line ~197). If `frameBytes` is
   payload-only, two frames that differ only in their outer header (e.g.
   different `SrcAddr` or `FrameAuthKey`) produce the same checksum and the
   second frame is silently suppressed as a false loop duplicate.
2. `SplitHorizon.Forward(frameBytes, ...)` forwards `frameBytes` verbatim to
   output interfaces. A payload-only `frameBytes` would produce malformed
   frames at recipients.
3. `OnFrameArrival`'s doc comment (verified at `8eb54a5`) states explicitly:
   "frameBytes is the raw frame (outer header + payload)."

The callback signature remains:

```go
// In internal/upstreamdial, passed to SetFrameCallback:
type FrameFn func(hdr frame.OuterHeader, raw []byte) error
// raw is the full wire frame: outer header (OuterHeaderSize bytes) + payload.
```

**Reconstruction in the receive goroutine:**

`frame.EncodeOuterHeader` exists at `8eb54a5` (verified in `internal/frame/frame.go`):

```go
// EncodeOuterHeader serialises h into exactly OuterHeaderSize (44) bytes
// using the ARCH-02 big-endian wire layout.
func EncodeOuterHeader(h OuterHeader) [OuterHeaderSize]byte
```

The receive goroutine reconstructs the full frame at the single call site:

```go
hdr, payload, err := frame.ReadOuterFrame(conn)
if err != nil { ... }
ehdr := frame.EncodeOuterHeader(hdr)
raw := append(ehdr[:], payload...)
_ = frameFn(hdr, raw)  // discard-and-continue; see FrameFn return-value contract below
```

**FrameFn return-value contract (v1.4 — F-SP4-001, binding):**

A non-nil return value from `frameFn(hdr, raw)` MUST NOT terminate the receive
loop or close the connection. The receive goroutine MUST discard the error
and continue reading the next frame (discard-and-continue semantics).

Rationale: `OnFrameArrival` returns non-nil on exactly two normal-operation paths:
`ErrAllPathsSplitHorizon` (E-FWD-001, when every forwarding candidate is
split-horizon blocked) and `ErrDropCacheHit` (when the drop cache identifies the
frame as a loop duplicate). Neither is a fatal condition. If the receive loop
exits on the first `ErrAllPathsSplitHorizon` return, the pin test
`TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader` fails (frame B is
never read and the second E-FWD-001 emission never fires), defeating the
byte-contract validation.

The normative precedent is `netingress.ServeConn` (verified at `8eb54a5` in
`internal/netingress/netingress.go`):

```go
if err := route(hdr, payload); err != nil {
    // Drop-and-continue: routing already logs E-ADM-016/017 per BC-2.05.008.
    continue
}
```

The `RouteFn` doc comment (verified at `8eb54a5`, lines 61–65) states:

> RouteFn returning a non-nil error is NOT a signal to close the connection
> [...] The error is logged and dropped. [...] would double-count.

`OnFrameArrival` already logs E-FWD-001 and EC-005 internally (verified at
`8eb54a5` in `internal/routing/on_frame_arrival.go`). The receive goroutine
MUST NOT log the error again (double-count rationale applies here too). The
correct idiom is a plain discard:

```go
_ = frameFn(hdr, raw)
```

This satisfies `errcheck` (the unchecked-error linter enabled in `.golangci.yml`
at `8eb54a5`) without the `//nolint` directive: a bare `_ =` assignment is a
legitimate explicit discard, not an ignored error. The `//nolint:errcheck`
directive MUST NOT be used; it suppresses errcheck for the whole line and masks
genuine unhandled-error bugs that may be introduced in future edits.

The exit-on-error form is explicitly forbidden:

```go
// FORBIDDEN — exits the loop on E-FWD-001, defeating TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader
if err := frameFn(hdr, raw); err != nil {
    return
}
```

This places the allocation at the single call site, mirrors the `netingress.ReadFrame`
precedent (payload-only return), and keeps `frame.ReadOuterFrame`'s signature
consistent with `netingress.ReadFrame` so delegation is possible without
changing the return contract.

**READ-error disposition contract (v1.5 — F-SP5-001, binding); superseded in part by v1.6 — F-SP6-001 (conn.Close() teardown wiring): see the 'Read-error conn.Close() teardown wiring' section below:**

On ANY non-nil return from `frame.ReadOuterFrame`, the receive goroutine MUST exit the loop
(`return`). `continue`-on-read-error is FORBIDDEN — this is the exact mirror of the v1.4
callback-error return-FORBIDDEN rule. The per-site disposition follows the
`netingress.ServeConn` precedent (verified at `8eb54a5` in
`internal/netingress/netingress.go`, read-error branch at lines 134–143, route-error branch at lines 145–147):

> read error → **exit** the loop (return nil on EOF/ctx-cancel, return err otherwise)
> callback error → **continue** (discard-and-continue)

Rationale: `continue`-on-read-error produces one of two failure modes:
1. **Busy-loop on conn-close EOF** — `frame.ReadOuterFrame` returns `io.EOF` on every
   iteration; the goroutine never exits; `Connector.Stop()` blocks on the per-reconnect
   join forever; AC-005 leak tests hang.
2. **Permanent framing desync on malformed frame** — if a semi-trusted upstream sends a
   malformed frame (`ErrInvalidFrameType` or truncation `io.ErrUnexpectedEOF`) WITHOUT
   closing the conn, every subsequent 44-byte header read consumes mid-frame garbage.
   `maintainConn` keepalive writes still succeed (full-duplex), so the conn is never
   torn down and never reconnected. The connection is permanently desynced.

~~Exit → `dialLoop`'s existing teardown/reconnect path closes the conn and re-dials, which
is the ONLY correct resync for a byte-misaligned stream.~~ *(amended v1.16 — F-SP20-001: RETRACTED. `maintainConn` is write-only (connector.go:399) and never observes receive-goroutine exit — dialLoop's teardown path does NOT fire on read-goroutine exit alone. The receive goroutine MUST itself call `_ = conn.Close()` before returning, converting the read-side failure into a write-side teardown. See the F-SP6-001 wiring section (binding).)*

**Logging disposition:** Two cases:
- **Clean exit** (io.EOF at a frame boundary, or any read error when `ctx.Err() != nil`
  — conn-close during `Stop()`/reconnect teardown): **silent exit, no log**. These are
  expected lifecycle events, not errors; logging them would produce noise in normal
  operation and does not double-count because `OnFrameArrival` never saw the frame.
- **Abnormal read error** (parse error such as `ErrInvalidFrameType`, truncation
  `io.ErrUnexpectedEOF`, or net error other than context cancellation): **one log line
  permitted**, at the implementer's discretion, before returning. The v1.4 double-count
  constraint does NOT apply here because `OnFrameArrival` never received the frame;
  there is no routing-layer event to double-count. Logging is optional — a silent exit is
  also acceptable given that `dialLoop` will log EC-001 on the subsequent redial failure
  if the connection is truly broken. The implementer MUST NOT log on the clean-exit path.

**Receive-goroutine sketch (updated — replaces the elided `{ ... }` from v1.4):**

> **SUPERSEDED (v1.16 — F-SP20-001):** this v1.5 sketch omits the binding `_ = conn.Close()` before `return` in the read-error branch (F-SP6-001). Do NOT implement from this sketch — the v1.6 'Updated receive-goroutine sketch' below (with `_ = conn.Close()`) is the binding version.

```go
for {
    hdr, payload, err := frame.ReadOuterFrame(conn)
    if err != nil {
        // READ error: exit the loop regardless of error type.
        // continue-on-read-error is FORBIDDEN (framing desync / busy-loop).
        // ctx.Err() != nil → conn-close during Stop()/reconnect: silent exit.
        return
    }
    ehdr := frame.EncodeOuterHeader(hdr)
    raw := append(ehdr[:], payload...)
    if hdr.FrameType == frame.FrameTypePEConnect {
        // bootstrap acknowledgment: silent discard
        continue
    }
    _ = frameFn(hdr, raw)  // discard-and-continue; non-nil return MUST NOT terminate loop (F-SP4-001)
}
```

**AC-005 read-error-exit pin test (v1.5 — binding for story-writer):**

A read-error-exit pin test is REQUIRED. The existing `TestConnector_ReceiveLoop_ExitsOnConnClose`
covers path (a) — the goroutine exits when the server closes the conn (EOF at frame boundary).
It does NOT cover path (b) — malformed frame without conn-close. Path (b) is the critical
framing-desync scenario where `continue` would produce a permanently broken connection.

**Pin test name:** `TestConnector_ReceiveLoop_ExitsOnReadError`

**Test shape:** ~~Inject garbage / malformed bytes (e.g. write a single byte `0xFF` as
`FrameType`, which `ParseOuterHeader` rejects as `ErrInvalidFrameType`) to the upstream
fixture connection WITHOUT closing the conn.~~ *(amended v1.9 — F-SP11-001: the v1.5 recipe
is PHYSICALLY UNREALIZABLE and contains a WRONG ERROR ATTRIBUTION. Two independent defects:
(1) UNREALIZABLE — `frame.ReadOuterFrame` mirrors `netingress.ReadFrame` → `io.ReadFull` over
the full 44-byte header (verified `internal/netingress/netingress.go` :79 — `io.ReadFull(r,
hdrBuf[:])`). One byte written, conn held open → ReadFull blocks awaiting bytes 2–44 → the
goroutine parks INSIDE the read, never reaches the error branch, never calls conn.Close(),
never exits; keepalives keep succeeding; the test hangs at RED forever against any
implementation. (2) WRONG ERROR — even writing 44 bytes of 0xFF would fail at byte[0] as
`ErrVersionMismatch` (major nibble `(0xFF >> 4) & 0x0F = 0xF`, compared against `VersionMajor
= 0`, verified `internal/frame/frame.go` :21, :106–108). The frame_type byte[1] is never
reached. "Single byte 0xFF as FrameType" conflates the version byte at byte[0] with the
frame_type byte at byte[1].*

**BINDING corrected recipe (v1.9 — F-SP11-001):** Write a COMPLETE 44-byte outer header to
the upstream fixture connection WITHOUT closing the conn. Concrete values (all verified against
`internal/frame/frame.go`):

- **byte[0] = 0x01** — valid version byte (VersionByte=0x01, verified frame.go :23; major
  nibble `(0x01 >> 4) & 0x0F = 0x00 == VersionMajor=0`, verified frame.go :21/:106–108;
  passes the version check, allowing ParseOuterHeader to reach the frame_type check)
- **byte[1] = 0x07** — out-of-range frame_type, one above the story's new upper bound
  `FrameTypePEConnect=0x06`; `FrameType(0x07).Valid()` returns false because `Valid()` checks
  `f >= FrameTypeData && f <= FrameTypePEConnect` after this story's amendment; ParseOuterHeader
  returns `ErrInvalidFrameType` at byte[1]
- **bytes[2:4] = 0x00, 0x00** — `PayloadLen = 0` big-endian (verified EncodeOuterHeader
  frame.go :90: `binary.BigEndian.PutUint16(b[2:4], h.PayloadLen)`); zero PayloadLen means
  `frame.ReadOuterFrame` does not attempt any payload read after the 44-byte header completes
- **bytes[4:44] = 0x00...** — remaining fields zero

With this 44-byte write, `io.ReadFull` completes deterministically (the pending 44-byte read
is satisfied by a single write — no timing gymnastics required). `ParseOuterHeader` returns
`ErrInvalidFrameType` at byte[1]. The receive goroutine takes the read-error branch →
`_ = conn.Close()` → exit → `maintainConn` write failure → reconnect. This IS the
F-SP5-001 case-2 scenario (malformed-without-close) realized correctly.

**Why the old recipe fails (two counts — for clarity in the codebase):**
1. Partial-header injection (< 44 bytes, conn held open) tests the BLOCKING path, not the
   ERROR path. `io.ReadFull` blocks — the goroutine never reaches the error branch.
2. `0xFF` at byte[0] tests VERSION REJECTION (`ErrVersionMismatch`), not frame-type rejection
   (`ErrInvalidFrameType`) — the frame_type byte at position 1 is never evaluated.

Assert that: (a) the receive goroutine exits (via the per-connection done channel or
`goleak.VerifyNone`), AND (b) the connector initiates a reconnect cycle (dials the fixture
again within the reconnect timeout). This proves exit-on-read-error is wired; a `continue`
implementation would busy-loop and the done channel would never close.

**OPTIONAL additional variant pin (v1.9 — F-SP11-001 optional ruling: ADJUDICATED — ADD):**
A second pin `TestConnector_ReceiveLoop_ExitsOnVersionMismatch` is worth adding: write a
complete 44-byte header with byte[0] = 0xFF (major nibble 0xF ≠ VersionMajor 0 →
`ErrVersionMismatch`), PayloadLen=0, conn NOT closed. Same exit contract (read-error branch
→ conn.Close() → reconnect). Rationale: ErrVersionMismatch is the OTHER common malformed-frame
path via ParseOuterHeader and exercises the same error-branch code; adding it costs one test
and fully documents the version-rejection path. It does NOT test a new code branch (same
read-error `if err != nil { _ = conn.Close(); return }` as ErrInvalidFrameType), but pins the
surface. Story-writer SHOULD add it as a companion to `TestConnector_ReceiveLoop_ExitsOnReadError`.

This test lives in `internal/upstreamdial/connector_test.go` (same file as AC-005).
The `in-package` fixture pattern for writing bytes to the upstream side is established
by the AC-005 flap harness (`heldConn`+`conn.Write` — see `TestConnector_BackoffParameters`
pattern, verified at `8eb54a5`).

**AC-003 forwarding-completeness pin test (v1.13 — F-SP17-001, BINDING):**

The discrimination contract (discard `FrameTypePEConnect`, forward everything else) is
pinned by tests at only two points: the forward side with `FrameTypeData` only
(`TestConnector_ReceiveLoop_DataFrameForwardedToCallback`) and the discard side with
`FrameTypePEConnect` only (`TestConnector_ReceiveLoop_PEConnectFrameDiscarded`). A
whitelist implementation `if hdr.FrameType == frame.FrameTypeData { _ = frameFn(hdr, raw) }`
passes all named tests while silently dropping `FrameTypeCtl` frames — which the story's
Non-Goals require to reach the callback for the S-BL.RESYNC-FRAME consumer. Under strict
TDD the RED test set IS the contract; the prose sketch does not gate.

**Pin test name:** `TestConnector_ReceiveLoop_CtlFrameForwardedToCallback`

**Test shape (binding for story-writer and implementer):** Assemble a complete valid frame
with `FrameType: frame.FrameTypeCtl` via `outerassembler.Assemble`. Use the in-package
accept-and-write fixture — same harness family as
`TestConnector_ReceiveLoop_PEConnectFrameDiscarded`: local `net.Listen`, accept the
connector's dialed connection, assemble + `conn.Write` from the server side. Assert that
`FrameFn` IS invoked (inverted assertion of `PEConnectFrameDiscarded` — the callback MUST
be called); assert `hdr.FrameType == frame.FrameTypeCtl` at the `FrameFn` call site.

**Rationale:** `FrameTypeCtl` chosen because the story's Non-Goals explicitly name the
RESYNC-over-PE path (the S-BL.RESYNC-FRAME consumer); RESYNC frames traverse the else-branch
as `FrameTypeCtl`. This test kills the whitelist-data-only malicious implementation:
`if hdr.FrameType == frame.FrameTypeData` passes `DataFrameForwardedToCallback` but fails
`CtlFrameForwardedToCallback`, closing the test-set underdetermination gap named by
F-SP17-001.

**Companion cosmetic fix (story-writer will apply; ruling recorded here):** The story's
receive-goroutine discrimination-sketch else-branch comment currently enumerates
`{data, ctl, arq, fec}` as the frame types that traverse the forward path. `empty_tick`
frames also traverse the else-branch and are forwarded to the callback. The comment
enumeration MUST gain `empty_tick`. The correct characterisation of the forward branch is
type-agnostic-except-pe_connect: EVERY valid frame type other than `pe_connect` is
forwarded, including `empty_tick`. Story-writer applies this comment amendment in the same
commit that implements the story.

**Test file:** `internal/upstreamdial/connector_test.go`.

**Estimated connector_test.go test count update (amended v1.13 — F-SP17-001):** +1 test
(`TestConnector_ReceiveLoop_ExitsOnReadError`) above the v1.4 forecast of 4 unit tests in
connector_test.go, plus +1 optional variant `TestConnector_ReceiveLoop_ExitsOnVersionMismatch`
(v1.9), plus +1 new forwarding-completeness pin `TestConnector_ReceiveLoop_CtlFrameForwardedToCallback`
(v1.13, F-SP17-001). New totals: **6 unit tests minimum** (+1 optional = **7**) in
`internal/upstreamdial/connector_test.go`. The total net-new test count for the story rises
from ~11 to **~12**: 1 `frame_test` amendment + 7 `connector_test` unit tests +
4 integration tests (`router_pe_receive_test.go`).

**AC-003 discard-continuation pin (v1.14 — F-SP18-001, BINDING):**

Pass-18 adversarial review identified that `TestConnector_ReceiveLoop_PEConnectFrameDiscarded`
asserts only "FrameFn NOT invoked" — it does not assert that the connection stays open and
reading continues after the discard. A malicious implementation:

```go
if hdr.FrameType == frame.FrameTypePEConnect {
    _ = conn.Close()
    return
}
```

passes every named test (silently discards and closes the conn; no other test sends a
`FrameTypePEConnect` frame and then checks for continued reading), while converting every
bootstrap frame into teardown+reconnect — producing a reconnect storm and dropping all frames
queued behind the bootstrap. This is the symmetric sibling of the F-SP17-001 forward-side
gap: the forward-side continuation is pinned by `NoDuplicateSuppression` (≥2 frames,
F-SP4-001 axis); the discard-side has no analogue.

**Binding remediation: EXTEND `TestConnector_ReceiveLoop_PEConnectFrameDiscarded` (do NOT
add a new test; counts are UNCHANGED at 7 connector / ~12 total).**

**Extended two-frame recipe (binding for story-writer and implementer):** On the SAME
connection that the test has already established, the fixture writes a `FrameTypePEConnect`
frame FOLLOWED by a `FrameTypeData` frame (both via `outerassembler.Assemble` + server-side
`conn.Write`, same in-package accept-and-write fixture pattern). Assert:

- **(a)** `FrameFn` is NOT invoked for the `FrameTypePEConnect` frame (the existing discard
  assertion — preserved verbatim).
- **(b)** `FrameFn` IS invoked for the subsequent `FrameTypeData` frame; `hdr.FrameType ==
  frame.FrameTypeData` at the call site.

**What (b) kills:** The discard-as-close implementation calls `_ = conn.Close()` and returns
on the `FrameTypePEConnect` frame. The conn is then closed before the `FrameTypeData` frame
is read. `frame.ReadOuterFrame` returns an error on the closed conn; the receive goroutine
exits. `FrameFn` is never invoked for the data frame. Assertion (b) fails. The test now kills
both the discard-and-exit form and the discard-as-close form.

**Rationale:** Extending (rather than adding) preserves the discard test's identity as THE
discrimination pin for the discard branch and avoids another count sweep. The two-frame
sequence on one conn is the minimum recipe that distinguishes discard-and-continue from
discard-and-exit. This is symmetric to the forward-side ≥2 continuation pin
(`NoDuplicateSuppression`, F-SP4-001 axis): that test sends two frames to verify the forward
branch continues; this extension sends one `FrameTypePEConnect` + one `FrameTypeData` to
verify the discard branch continues.

**Realizability note:** Two frames written back-to-back on one conn arrive as one or two TCP
segments — the framing is length-delimited and segment-boundary-independent. `frame.ReadOuterFrame`
mirrors `netingress.ReadFrame`: it loops on `io.ReadFull(44)` to read the outer header (44 bytes
exactly), then reads `hdr.PayloadLen` bytes of payload (verified `internal/frame/frame.go`
`ReadOuterFrame` semantics, same as `netingress.ReadFrame` at `8eb54a5`). Whether the two
frames arrive as one TCP segment or two, `ReadOuterFrame` completes each frame read atomically
by `io.ReadFull` contract. No timing gymnastics required; the recipe is deterministically
realizable.

**Test file:** `internal/upstreamdial/connector_test.go` (same file as the existing
`TestConnector_ReceiveLoop_PEConnectFrameDiscarded`).

**Counts unchanged:** This is an extension of an existing test, NOT a new test. The connector
test count remains **7 minimum** (+ optional `ExitsOnVersionMismatch` = 7). The total
net-new story test count remains **~12**. No count-sweep of the note is required.

**AC-001 PC-3 / AC-004 precondition observable substitutes (v1.6 — F-SP6-003, binding for story-writer):**

Pass-6 adversarial review identified that AC-001 PC-3 ("connector.Mode() returns
`upstreamdial.ModePE`") and AC-004's precondition poll ("polls for
`upstreamdial.ModePE`") are unassertable under the Q9.3 real-`runRouter` harness: the
harness launches `runRouter` as a goroutine and `runRouter` holds the connector as an
unexported local. The test has no handle to call `connector.Mode()` on.

**Ruling: substitute the following observables (binding for story-writer):**

1. **PE establishment confirmed via `peWriteFixture.accepted`**: When `peWriteFixture`'s
   `accepted` channel receives a value, the connector has dialed and the PE connection is
   established (i.e. ~~the connector has completed step 3, which atomically increments
   `connectedCount` and is the same event that causes `Mode()` to become `ModePE`~~).
   *(amended v1.7 — F-SP7-001/F-SP7-002: the parenthetical above is RETRACTED — see v1.7
   corrected-observables block below.)*
   The `accepted` receive IS the `connector.Mode() == ModePE` observable in tests using the
   `runRouter` goroutine pattern.
2. **`"mode=PE"` writer-output line**: The existing `waitForConnections`/`scanForLine`
   pattern in `router_pe_connector_test.go` (verified at `8eb54a5`) polls the router's
   writer output for the `"mode=PE"` line emitted by `runRouter` on the PE-mode
   ~~transition. This pattern is available as an alternative or supplementary poll.~~
   *(amended v1.7 — F-SP7-001: the characterisation of `"mode=PE"` as signalling a
   "PE-mode transition" is RETRACTED — see v1.7 corrected-observables block below.)*

`connector.Mode()` assertions — calling the method directly — are only valid in
`internal/upstreamdial/connector_test.go` (in-package access, concrete `*Connector`
type). AC-001 PC-3 text and AC-004 precondition text MUST be reworded by the
story-writer to use one of the two observable substitutes above.

---

**Corrected observable semantics for AC-001 PC-3 and AC-004 precondition (v1.7 — F-SP7-001 + F-SP7-002, BINDING):**

Pass-7 adversarial review (F-SP7-001) established that the v1.6 ruling conflated two
distinct observables with incorrect semantics. Ground-truth verification at `8eb54a5`:

**`"mode=PE"` — CORRECT SEMANTICS: PE-CONFIG PRESENCE only, NOT an establishment observable.**

`"mode=PE"` (full string: `"switchboard router: mode=PE upstream_routers=%v"`) is emitted
in `runRouter`'s startup writer block (verified `cmd/switchboard/mgmt_wire.go` :548) gated
ONLY on `len(upstreamRouters) > 0`, synchronously after `connector.Start()` returns.
`connector.Start()` merely launches goroutines — no dial, no TCP accept, no
`connectedCount.Add(1)` has yet occurred. The second emission (:587) fires on SIGHUP reload
when the upstream-router list changes to non-empty. Neither emission has any dependency on
`connectedCount` or on any established connection.

The existing test `TestRunRouter_PE_UnreachableUpstream_PartialPE` (verified at `8eb54a5`
in `cmd/switchboard/router_pe_connector_test.go`) proves this: `"mode=PE"` fires even when
the upstream is unreachable (the test verifies EC-001 log fires alongside `"mode=PE"`).

**CONCLUSION: `"mode=PE"` proves only that `len(upstreamRouters) > 0` at startup (or SIGHUP).
It is NOT an establishment observable. A test gating AC-001 PC-3 or AC-004's precondition
on `"mode=PE"` receives a false-green — mode=PE will have fired before any connection
attempt completes.**

**`peWriteFixture.accepted` — CORRECT SEMANTICS: TCP-accept-level establishment, strictly BEFORE `connectedCount.Add(1)`.**

When `peWriteFixture.accepted` receives, the connector's `net.Dial` to the fixture has
succeeded and the fixture's `net.Accept()` has returned a `net.Conn`. Verified timing
sequence in `internal/upstreamdial/connector.go`:
- bootstrap `Write` at :350 (after `DialContext` returns)
- `connectedCount.Add(1)` at :365 (after the bootstrap Write succeeds)
- TCP accept on fixture side fires at `DialContext` success — strictly before :350 (bootstrap Write) and thus strictly before :365 (Add(1))

`peWriteFixture.accepted` is therefore an **early / approximate** establishment observable:
it fires when the TCP session is open but before the three-step "established" definition
(DialContext + bootstrap Write + Add(1)) is complete. It is sufficient for asserting
"connection is being established" but NOT for asserting `Mode() == ModePE`.

**Corrected ruling per observable, binding for story-writer:**

| Observable | What it proves | Correct use |
|---|---|---|
| `"mode=PE"` in writer output | PE-CONFIG PRESENCE: `len(upstreamRouters) > 0` at startup/SIGHUP only | Use ONLY to assert PE config was applied. Do NOT use as an establishment gate. |
| `peWriteFixture.accepted` receive | TCP-accept-level establishment — TCP session open, strictly before `connectedCount.Add(1)` | Use as an early/approximate establishment gate for AC-001 PC-3 and AC-004 precondition. Sufficient for "connector has dialed the upstream". |
| Frame arrival on `FrameFn` / E-FWD-001 emission | Receive-goroutine is live and forwarding frames | The ONLY true establishment + liveness observable. Required for ACs that assert the receive loop is active (e.g. AC-001 receive-loop-active assertion, AC-002 E-FWD-001 emission). |

**AC-001 PC-3 corrected precondition (binding for story-writer):**
Use `peWriteFixture.accepted` receive to gate on "PE connection established at TCP level";
then the AC's own frame-forward assertion (E-FWD-001 or `FrameFn`-driven effect) serves as
the receive-goroutine liveness signal. The story text for AC-001 PC-3 MUST be amended to
replace any `connector.Mode() == ModePE` or `"mode=PE"` establishment poll with
`peWriteFixture.accepted` receipt as the establishment gate.

**AC-004 precondition corrected (binding for story-writer):**
Replace any poll for `"mode=PE"` or `connector.Mode() == ModePE` with `peWriteFixture.accepted`
receive as the precondition gate. This is the correct signal that the PE conn is open and
the receive goroutine will shortly be accepting frames. The story text for AC-004's
precondition MUST be updated accordingly.

**Pass-7 adjudicated item (not a separate finding — folded into F-SP7-001 corrected block):**
The v1.6 pass-6-adjudicated-clean row at the end of this note (line ~1925) already correctly
stated "accepted receives at TCP accept time, which is strictly before Add(1)" and characterised
`accepted` as "slightly EARLY relative to `ModePE`". The v1.6 body text at the observable-substitute
block was inconsistent with that adjudicated-clean row. This v1.7 correction makes the body
text consistent with the pass-6 adjudication's own correct analysis.

---

**Read-error conn.Close() teardown wiring (v1.6 — F-SP6-001, BINDING):**

The v1.5 contract stated "exit → dialLoop's existing teardown/reconnect path closes the conn
and re-dials" without specifying HOW that teardown is triggered. Ground-truth verification at
`8eb54a5` reveals the gap:

`maintainConn` (verified at `8eb54a5`, lines ~399–430) is a **write-only loop** — it selects on
`stopAddr` (closed by `addrCancel.cancel()` from `reconcile` / `Stop()`) and on `keepaliveTick.C`
(sends keepalive write, checks `SetWriteDeadline` + write errors). It never calls `conn.Read`
and has no knowledge of the receive goroutine. If a malformed frame arrives without the
upstream closing the connection, keepalive WRITES continue succeeding (full-duplex TCP),
`maintainConn` never returns, and `dialLoop` never closes/re-dials — the permanent-desync
failure mode the v1.5 rationale claimed to cure.

**BINDING: on read-error exit, the receive goroutine MUST call `_ = conn.Close()` before
returning.** This is the mechanism that wires the read-side failure into the write-side
teardown path:

1. Receive goroutine calls `_ = conn.Close()` on non-nil `frame.ReadOuterFrame` return.
2. `maintainConn`'s next `conn.SetWriteDeadline` call (or the subsequent `conn.Write`) fails
   because the conn is closed — `maintainConn` returns.
3. `dialLoop` falls through to the post-`maintainConn` teardown sequence
   (decrement connectedCount, log EC-004 if warranted, loop to reconnect with backoff).

`_ = conn.Close()` after the receive goroutine has already exited due to `dialLoop` teardown
(the normal path, where `dialLoop` closes the conn) produces a second `Close()` call on an
already-closed `net.Conn`. Go's `net.Conn.Close()` is **safe to call multiple times**: the
second call returns an error of the form `use of closed network connection` which is discarded
by the `_ =` assignment. This idempotency is guaranteed by the `net` package's `poll.FD.Close()`
implementation. State it explicitly: **double-close is safe; the receive goroutine MUST always
call `_ = conn.Close()` on exit, regardless of how the exit was triggered.**

**Updated receive-goroutine sketch (v1.6 — replaces v1.5 sketch):**

```go
for {
    hdr, payload, err := frame.ReadOuterFrame(conn)
    if err != nil {
        // READ error: exit the loop regardless of error type.
        // continue-on-read-error is FORBIDDEN (framing desync / busy-loop).
        // BINDING (v1.6 — F-SP6-001): close the conn to trigger maintainConn
        // write failure → dialLoop teardown → backoff → redial.
        // Double-close is safe/idempotent on net.Conn.
        _ = conn.Close()
        return
    }
    ehdr := frame.EncodeOuterHeader(hdr)
    raw := append(ehdr[:], payload...)
    if hdr.FrameType == frame.FrameTypePEConnect {
        // bootstrap acknowledgment: silent discard
        continue
    }
    _ = frameFn(hdr, raw)  // discard-and-continue; non-nil return MUST NOT terminate loop (F-SP4-001)
}
```

**Reconnect latency bound:** After the receive goroutine calls `_ = conn.Close()`, redial is
initiated within ≤ `keepaliveInterval` (the next keepalive tick's `SetWriteDeadline` + `Write`
fails) + backoff. The pin test `TestConnector_ReceiveLoop_ExitsOnReadError` timeout MUST
accommodate `keepaliveInterval` + `operativeBase` backoff; tests should use a short
`keepaliveInterval` (e.g. `10ms` or `20ms`, consistent with the existing `connector_test.go`
pattern at `8eb54a5`).

**Backoff after established-then-failed (v1.6 — F-SP6-001 reconnect-storm concern):**
Verified at `8eb54a5` in `dialLoop`: backoff resets to `operativeBase(c.keepaliveInterval)` on
each successful connect (line `backoff = operativeBase(c.keepaliveInterval)` immediately after
`c.connectedCount.Add(1)`). After a successful connection that subsequently fails (including
the malformed-frame-then-close scenario), the next redial begins with `operativeBase` delay
(keepaliveInterval, floored at `BackoffBase = 500ms`), then applies `nextBackoff` on each
successive failure. The reconnect-storm concern from pass-6 is answered: a repeatedly
malformed upstream produces at most one reconnect per `operativeBase` interval, not a tight
spin. The dial-failure path (`ctx.Done()` → `time.After(backoff)` → `backoff = nextBackoff`)
is entered ONLY after `maintainConn` returns and the loop iterates; the reset-on-success
line in the current iteration's first successful path is not re-executed on reconnect until
the new conn succeeds.

**Lifecycle contract amendment (v1.6 — F-SP6-001):**

The `conn.Close()` ownership statement is amended. Prior versions stated dialLoop owns
`conn.Close()` exclusively (step 8 of the goroutine ordering contract). The corrected contract:

- **Normal teardown** (dialLoop calls `conn.Close()` after `maintainConn` returns due to
  keepalive write failure or `stopAddr` close): `conn.Close()` is called at step 8 by
  `dialLoop`. The receive goroutine has already exited or will exit shortly after; its
  subsequent `_ = conn.Close()` is a harmless idempotent second close.
- **Abnormal teardown** (receive goroutine exits on read error, calls `_ = conn.Close()`):
  `maintainConn`'s next write fails, `maintainConn` returns, then `dialLoop` calls
  `conn.Close()` again at step 8 — also idempotent.

In both cases the conn is eventually closed exactly once from the perspective of the
operating system (TCP RST/FIN), with the second `Close()` call discarded. The per-address
`done chan struct{}` ordering (step 9) is unchanged: `dialLoop` MUST still join the receive
goroutine before looping to reconnect.

**Bounded-read / read-deadline divergence from netingress precedent (v1.5 — F-SP5-OBS-1, accepted with rationale):**

`netingress.ServeConn` (verified at `8eb54a5`) wraps every `ReadFrame` call in
`io.LimitReader(conn, MaxFrameBytes)` (CWE-400, VP-066, line 132). The receive-goroutine
sketch calls `frame.ReadOuterFrame(conn)` directly with no `LimitReader` and no read
deadline.

**Ruling: accept with rationale — the divergence is acceptable for the PE receive path.**

Mitigations present that are absent on the data-plane netingress path:
1. `PayloadLen` is `uint16` — maximum frame allocation is 44 + 65 535 = 65 579 bytes per
   frame. This is a hard codec-level bound, not a configurable limit. A malformed
   `PayloadLen = 0xFFFF` allocates at most ~64 KB, which is a bounded, acceptable per-frame
   cost with no amplification possible.
2. The PE upstream connection is a DIALED connection to a configured, semi-trusted upstream
   router — not an arbitrary accepted connection from an unknown client (which is the netingress
   threat model). The upstream router address is operator-controlled; the adversary model for
   this path is a misconfigured or compromised upstream, not an unauthenticated attacker.
3. The READ-error exit contract (F-SP5-001) ensures any malformed frame causes an
   immediate connection teardown and reconnect — the allocation is bounded per connection,
   not per-attack-loop.

A `LimitReader` would add wrapper indirection on every `ReadOuterFrame` call on a
point-to-point dialed connection where the frame-size bound is already enforced by the
`uint16 PayloadLen` codec. The benefit does not justify the overhead divergence from the
`frame.ReadOuterFrame` interface contract. No read deadline is set because
`maintainConn`'s keepalive ticker already detects dead connections via write failures
(verified at `8eb54a5` in `maintainConn` — `conn.SetWriteDeadline` + write failure → return).

No implementation change required. Observation recorded.

**Import note:** `netingress` is at position 18 (ARCH-08 §6.5). The
`Connector` at position 19 may NOT import `netingress` (that would be a
back-edge: 19 → 18). The legal option is:

**Option 3 (ruling):** A new function `frame.ReadOuterFrame(r io.Reader) (OuterHeader, []byte, error)` is added to `internal/frame/frame.go` (position 2). It implements the same read-header-then-payload logic as `netingress.ReadFrame`, returning payload-only. `netingress.ReadFrame` may delegate to it (reducing duplication) or retain its own copy with a cross-reference comment — the implementer's choice.

The `upstreamdial` receive goroutine calls `frame.ReadOuterFrame(conn)`,
reconstructs the full frame via `EncodeOuterHeader`+append, and passes the
result to the `FrameFn` callback. No import-graph change: `upstreamdial`
already has a transitive path to `frame` through `halfchannel` and `outerassembler`;
a direct `frame` import at position 19 is lawful (frame is position 2).

**ARCH-08 obligation:** `internal/upstreamdial` gains a direct import edge to
`internal/frame` (position 2). The allowed-import row must be updated in §6.5:
`{halfchannel, outerassembler}` → `{frame, halfchannel, outerassembler}`.
This is a §6.4 amendment (import-set extension of an existing package, not a
new package). The story implementer must update ARCH-08 §6.5 in the same commit
that introduces the `frame.ReadOuterFrame` import.

**ARCH-08 parenthetical reconciliation obligation (v1.10 — F-SP12-001, BINDING):**
The same §6.5 row edit MUST also reconcile the parenthetical that currently reads:

> "frame is NOT imported directly; reachable transitively through outerassembler
> and halfchannel. Corrected from v2.6 {frame, outerassembler} per adversary
> pass-1 F-P1-001."

This story reverses the F-P1-001 correction for `internal/frame`: frame becomes
a DIRECT import again (legitimately — `frame.ReadOuterFrame` and
`frame.FrameTypePEConnect` in `connector.go`). An implementer following the
import-set amendment literally without updating the parenthetical would leave
ARCH-08 §6.5 self-contradictory: the import set would include `frame` while the
parenthetical asserts `frame` is NOT imported directly, with an actively
misleading historical rationale.

**Concrete replacement wording for the parenthetical (binding for implementer):**
Replace the stale parenthetical with:

> "frame direct import added by S-BL.PE-RECEIVE-LOOP (pos 2 → pos 19, forward
> edge, no cycle; frame.ReadOuterFrame + frame.FrameTypePEConnect in
> connector.go). Historical note: v2.6 had listed {frame, outerassembler}
> prematurely; adversary pass-1 F-P1-001 corrected that (no direct import
> existed at that time); the direct frame edge is now real as of this story."

This replacement preserves the historical context of the F-P1-001 correction
while accurately stating the post-story position. The PROSPECTIVE and
pre-merge machine-verification qualifiers in the surrounding row text should
be updated per the normal merge-time procedure.

**Blast-radius count ruling (v1.10 — F-SP12-001; amended v1.11 — F-SP13-001; amended v1.12 — F-SP14-001):**
The ARCH-08 parenthetical is a DISTINCT import-edge-prose location from the ten
FrameTypePEConnect/Valid() sweep locations enumerated in Q3. It belongs under the
§6.5 ARCH-08 obligation (Q2), not the frame.go/frame_test.go sweep (Q3). The
v1.10 ruling counted **11**: 10 FrameTypePEConnect/Valid() locations (Q3 table,
unchanged) + 1 ARCH-08 §6.5 parenthetical. *(amended v1.11 — F-SP13-001:* the
§6.6.2 forbidden-edges bullet is a SECOND ARCH-08 import-edge-prose location
carrying identical stale claims; the total blast-radius count is therefore **12**:
10 frame sweep + 2 ARCH-08 locations (§6.5 row parenthetical + §6.6.2
forbidden-edges bullet). The Q3 table's "10 locations" summary remains accurate
for the FrameTypePEConnect sweep; the Summary-of-Rulings Q3 row "10 locations"
count applies to that sweep only. Both ARCH-08 edit targets (§6.5 and §6.6.2)
MUST be made in the same commit as the frame-sweep edits.*) *(amended v1.12 — F-SP14-001:*
`BC-2.01.004:61` is a wire-format spec pair partner to `ARCH-02:74` — both are
same-commit parallel obligations and both belong under the Q3 ARCH-02 amendment
obligation (Q3), not inside the unified-12 count. The ONE consistent arithmetic
sentence the story MUST carry verbatim is: **"The total blast radius is: unified 12
(10 frame sweep locations + 2 ARCH-08 import-edge-prose locations) + wire-format
spec pair (ARCH-02:74 + BC-2.01.004:61, same-commit parallel obligations alongside
the frame-sweep commit)."** The unified-12 count remains the frame sweep + ARCH-08
obligation count; BC-2.01.004:61 is a sibling parallel obligation paired with
ARCH-02:74 and not folded into the unified-12 numeral.)

**§6.6.2 upstreamdial forbidden-edges bullet — ARCH-08 edit obligation (v1.11 — F-SP13-001, BINDING):**

The §6.6.2 upstreamdial bullet (ARCH-08 lines 456–466 at `8eb54a5`) currently reads:

> `internal/upstreamdial` MUST NOT import `internal/drain`, `internal/routing`,
> `internal/testenv`, or any package at positions 20–23. Allowed imports are
> `{halfchannel, outerassembler}` only (positions 5 and 8). Nothing may import
> `internal/upstreamdial` except `cmd/switchboard`, `internal/testenv` (the _test-only
> composition root at position 23), and `_test` files — it is an effectful leaf in
> the connectivity layer. Cycle-freeness: all allowed imports (halfchannel pos 5,
> outerassembler pos 8) are below position 19; no back-edges. `internal/testenv` at
> position 23 importing upstreamdial at position 19 is lawful (23 > 19). (Per
> placement note Q4 forbidden edges and ARCH-08 §6.4 constraint requirement; import
> set corrected from v2.6 {frame, outerassembler} per adversary pass-1 F-P1-001;
> permitted-importers updated per adversary pass-7 F-P7-002.)

This bullet carries the same three stale claims as the §6.5 parenthetical remediated in v1.10:
(1) `"Allowed imports are {halfchannel, outerassembler} only (positions 5 and 8)"`;
(2) cycle-freeness enumeration `"all allowed imports (halfchannel pos 5, outerassembler pos 8)"`;
(3) F-P1-001 import-correction rationale.

In the same commit that edits the §6.5 row, the implementer MUST also replace the
§6.6.2 bullet with the following text (the three sub-edits are indicated inline):

**Binding replacement bullet text (v1.11 — implementer edits mechanically from this):**

> `internal/upstreamdial` MUST NOT import `internal/drain`, `internal/routing`,
> `internal/testenv`, or any package at positions 20–23. Allowed imports are
> `{frame, halfchannel, outerassembler}` only (positions 2, 5 and 8). Nothing may import
> `internal/upstreamdial` except `cmd/switchboard`, `internal/testenv` (the _test-only
> composition root at position 23), and `_test` files — it is an effectful leaf in
> the connectivity layer. Cycle-freeness: all allowed imports (frame pos 2, halfchannel pos 5,
> outerassembler pos 8) are below position 19; no back-edges. `internal/testenv` at
> position 23 importing upstreamdial at position 19 is lawful (23 > 19). (Per
> placement note Q4 forbidden edges and ARCH-08 §6.4 constraint requirement; import
> set corrected from v2.6 {frame, outerassembler} per adversary pass-1 F-P1-001
> (no direct frame import existed then); frame direct import re-added by
> S-BL.PE-RECEIVE-LOOP (frame.ReadOuterFrame + frame.FrameTypePEConnect in
> connector.go); permitted-importers updated per adversary pass-7 F-P7-002.)

The three sub-edits in detail:
(a) `"Allowed imports are {halfchannel, outerassembler} only (positions 5 and 8)"` →
    `"Allowed imports are {frame, halfchannel, outerassembler} only (positions 2, 5 and 8)"`;
(b) `"all allowed imports (halfchannel pos 5, outerassembler pos 8) are below position 19; no back-edges"` →
    `"all allowed imports (frame pos 2, halfchannel pos 5, outerassembler pos 8) are below position 19; no back-edges"`;
(c) `"import set corrected from v2.6 {frame, outerassembler} per adversary pass-1 F-P1-001;"` →
    `"import set corrected from v2.6 {frame, outerassembler} per adversary pass-1 F-P1-001 (no direct frame import existed then); frame direct import re-added by S-BL.PE-RECEIVE-LOOP (frame.ReadOuterFrame + frame.FrameTypePEConnect in connector.go);"` — the F-P7-002 clause (`"permitted-importers updated per adversary pass-7 F-P7-002."`) is PRESERVED UNTOUCHED at end.

**Class-closure grep transcript (v1.11 — sweep-pattern discipline):**

To verify no §6.6.3 sibling exists, the following patterns were grepped against
`.factory/specs/architecture/ARCH-08-dependency-graph.md`:

| Pattern | Hits | Lines | Disposition |
|---------|------|-------|-------------|
| `halfchannel, outerassembler` | 4 | 13, 316, 325, 458 | Line 13: changelog row v2.7 (history-preserved, not an edit target). Line 316: position-10 `internal/arqsend` row (`{arq, frame, halfchannel, outerassembler}`) — benign substring match on a different package's own import set; not an upstreamdial claim; no edit. *(initially uncounted; caught on orchestrator audit — same F-SP7-003 defect class)* Line 325: §6.5 table row (pass-12 edit target). Line 458: §6.6.2 bullet (pass-13 edit target — this amendment). |
| `F-P1-001` | 4 | 13, 325, 465, 502 | Line 13: changelog row v2.7 (history-preserved). Line 325: §6.5 table row (pass-12 edit target). Line 465: §6.6.2 bullet (pass-13 edit target). Line 502: changelog body row v2.7 (history-preserved, not an edit target). |

**Conclusion:** beyond the two edit targets (§6.5 line 325, §6.6.2 lines 456–466) and
four changelog rows (lines 13, 502 — history-preserved, immutable), there are NO
further occurrences of the stale import-set claim or the F-P1-001 rationale in ARCH-08.
Pass-14 cannot find a §6.6.3 sibling.

**Cite:** `internal/frame/frame.go` `ParseOuterHeader`, `EncodeOuterHeader` (verified at `8eb54a5`);
`internal/netingress/netingress.go` `ReadFrame` — payload-only return, `hdrBuf` discarded (verified at `8eb54a5`);
`internal/netingress/netingress.go` `RouteFn` doc comment — "NOT a signal to close the connection … error is logged and dropped" (verified at `8eb54a5`, lines 61–65);
`internal/netingress/netingress.go` `ServeConn` — drop-and-continue `continue` with double-count-avoidance rationale (verified at `8eb54a5`, lines 145–147);
`internal/routing/on_frame_arrival.go` `OnFrameArrival` — `crc32.ChecksumIEEE(frameBytes)` (verified at `8eb54a5`);
`internal/routing/on_frame_arrival.go` `ErrAllPathsSplitHorizon`, `ErrDropCacheHit` — two non-fatal non-nil return paths (verified at `8eb54a5`);
`.golangci.yml` — `errcheck` enabled (verified at `8eb54a5`);
ARCH-08 §6.5 position table.

---

## Q3 — FO-PE-LOOP-001 discharge: define `frame.FrameTypePEConnect` vs adopt `frame.FrameTypeCtl`

**Ruling: define a new constant `frame.FrameTypePEConnect` at the next available
value. The current five canonical values are `0x01`–`0x05` (verified at `8eb54a5`).
`FrameTypePEConnect` MUST be assigned value `0x06`. The receive loop discriminates
bootstrap frames from session data by checking `hdr.FrameType == frame.FrameTypePEConnect`
after `frame.ReadOuterFrame`. Bootstrap frames are consumed (dropped or ACK'd)
by the receiver; data frames are forwarded through the callback.**

**Why not `FrameTypeCtl` (0x03)?**

`FrameTypeCtl` (0x03) is defined in `internal/frame/frame.go` (verified at `8eb54a5`)
as a generic control-plane frame type. The placement note for S-7.04-FU-PE-CONNECTOR
(Q6, F-P28-001 correction) cites it as the "control-category constant" — but that
story deferred the specific PE-CONNECT constant to this story with rationale:

> "using `halfchannel.FrameTypeData` as placeholder makes bootstrap frames
> indistinguishable from session data at the receiver."

Adopting `FrameTypeCtl` would disambiguate bootstrap from data, but would conflate
PE-CONNECT with other future control-plane messages (keepalive ACKs, RESYNC,
DRAIN). A distinct `FrameTypePEConnect` is needed so the receive loop can apply
the right handler without a secondary discriminator field in the channel header.

**`frame.FrameType.Valid()` update obligation — full blast radius (F-SP1-002 + F-SP1-003):**

`internal/frame/frame.go` (verified at `8eb54a5`) currently defines:

```go
func (f FrameType) Valid() bool {
    return f >= FrameTypeData && f <= FrameTypeFec
}
```

With `FrameTypeFec = 0x05`, this accepts `0x01`–`0x05` and rejects `0x06`.
Adding `FrameTypePEConnect = 0x06` REQUIRES updating `Valid()` to
`return f >= FrameTypeData && f <= FrameTypePEConnect` (or the widened upper bound).
Failing to update `Valid()` will cause `frame.ParseOuterHeader` to return
`ErrInvalidFrameType` for every PE-CONNECT bootstrap frame received, silently
dropping all upstream bootstraps.

The `Valid()` widening has a full blast radius that the implementer MUST sweep and
remediate. Grep-verified against `8eb54a5`:

**Test amendments required (F-SP1-002):**

1. `internal/frame/frame_test.go` — `TestFrameType_Valid` table (verified at `8eb54a5`,
   lines containing the `just_above_max` case):
   - Current: `{"just_above_max", frame.FrameType(0x06), false}` — this case MUST be
     changed to `{"just_above_max", frame.FrameType(0x07), false}` because `0x06` will
     become `FrameTypePEConnect` (valid). The test name "just_above_max" remains accurate
     for `0x07` (one above the new max `0x06`). Verified: `frame_test.go` contains
     `{"just_above_max", frame.FrameType(0x06), false}` at `8eb54a5`.

2. `internal/frame/frame_test.go` — `TestParseOuterHeader_RejectsInvalidFrameType`
   (verified at `8eb54a5`):
   - Current: `invalids := []byte{0x00, 0x06, 0x77, 0xFF}` — the `0x06` entry MUST be
     changed to `0x07` (or any value `>= 0x07`) because `0x06` will no longer be invalid.
     Verified: `frame_test.go` contains `invalids := []byte{0x00, 0x06, 0x77, 0xFF}` at `8eb54a5`.

**Doc-comment updates required (F-SP1-002):**

3. `internal/frame/frame.go` `FrameType` type comment (verified at `8eb54a5`):
   - Current: `"Only five values are canonical; all others are reserved."` — MUST be
     updated to reflect six canonical values. Verified: `frame.go` line 27 contains
     `"Only five values are canonical; all others are reserved."` at `8eb54a5`.

4. `internal/frame/frame.go` `Valid()` doc comment (verified at `8eb54a5`):
   - Current: `"Valid reports whether the FrameType byte is one of the five canonical
     enum values defined in ARCH-02 §3.1. Returns false for 0x00 and 0x06..0xFF."` —
     MUST be updated: "six canonical enum values" and "Returns false for 0x00 and
     0x07..0xFF". Verified: `frame.go` lines 40–41 contain this text at `8eb54a5`.

5. `internal/frame/frame.go` `ErrInvalidFrameType` doc comment (verified at `8eb54a5`):
   - Current: `"not one of the five canonical FrameType values (per ARCH-02 §3.1)"` —
     MUST be updated to "six canonical" or "not in {0x01..0x06}". Verified: `frame.go`
     lines 47–48 contain this text at `8eb54a5`.

**Amended blast-radius (F-SP2-004 — two locations missed in v1.1):**

The v1.1 Q3 sweep was incomplete. Two additional `frame_test.go` locations require amendment
(grep-verified against `8eb54a5`):

6. `internal/frame/frame_test.go` — `TestParseOuterHeader_AcceptsAllValidFrameTypes` function
   doc comment (verified at `8eb54a5`, located at the comment block beginning "TestParseOuterHeader_AcceptsAllValidFrameTypes
   asserts that all five canonical FrameType values pass ParseOuterHeader's enum validation"):
   - Current: `"all five canonical FrameType values"` — MUST be updated to `"all six canonical
     FrameType values"`. Verified: this comment exists immediately above the function at `8eb54a5`.

7. `internal/frame/frame_test.go` — `TestParseOuterHeader_AcceptsAllValidFrameTypes` `valid`
   slice (verified at `8eb54a5`):
   - Current: 5-element slice `{FrameTypeData, FrameTypeEmptyTick, FrameTypeCtl, FrameTypeArq, FrameTypeFec}` — MUST have `frame.FrameTypePEConnect` appended as the sixth element. This is the regression guard that `Valid()` accepts the new constant; without it, a future narrowing of `Valid()` could silently break PE-CONNECT bootstrap parsing. Verified: the function body contains this exact 5-element slice at `8eb54a5`.

**Extended sweep transcript (F-SP2-004 re-sweep, broader patterns):**

The following grep patterns were run against `*.go` files at `8eb54a5` to satisfy the F-SP2-004
re-sweep requirement:

- `grep -rn "five" --include="*.go" internal/frame/` → hits at `frame_test.go` lines 501, 560 (both now enumerated above), and no other `frame/` files.
- `grep -rn "0x05" --include="*.go" internal/frame/` → hits at `frame.go` (FrameTypeFec constant definition) and `frame_test.go` (test data bytes — not FrameType assumptions; these are payload bytes in round-trip tests, not Valid() range bounds). No additional Valid()-range assumptions found.
- `grep -rn "FrameTypeFec" --include="*.go" .` (excluding `.factory/`) → hits: `internal/frame/frame.go` (constant definition), `internal/frame/frame_test.go` (test data), `internal/outerassembler/fuzz_test.go`. The `fuzz_test.go` hit is at the `ft.Valid()` gate pattern — it does NOT hard-code a range bound; `ft.Valid()` auto-adjusts when `Valid()` is widened. Verified: `fuzz_test.go` line 128 reads `if !ft.Valid() { return }` (adversary pass-2 confirmed self-adjusting; recorded as swept-clean per F-SP2-001 adjudication section below).
- `grep -rni "five" --include="*.md" .factory/specs/` → hits in ARCH-02 are the `fec=0x05` value in the `frame_type` table row, which is a value description not a count claim, and is already covered by the ARCH-02 amendment obligation in Q3. No additional count-five claims found in spec docs.

**All ten blast-radius locations now enumerated (complete list, updated v1.6 — F-SP6-004; prior count of eight was incomplete):**

| # | Location | Required change |
|---|----------|-----------------|
| 1 | `frame_test.go` `TestFrameType_Valid` `just_above_max` case | `FrameType(0x06) → FrameType(0x07)` |
| 2 | `frame_test.go` `TestParseOuterHeader_RejectsInvalidFrameType` `invalids` slice | `0x06 → 0x07` in slice |
| 3 | `frame.go` `FrameType` type doc comment | `"Only five values"` → `"Only six values"` |
| 4 | `frame.go` `Valid()` doc comment | `"five canonical…0x06..0xFF"` → `"six canonical…0x07..0xFF"` |
| 5 | `frame.go` `ErrInvalidFrameType` doc comment | `"five canonical"` → `"six canonical / not in {0x01..0x06}"` |
| 6 | `frame_test.go` `TestParseOuterHeader_AcceptsAllValidFrameTypes` doc comment | `"all five canonical"` → `"all six canonical"` (F-SP2-004) |
| 7 | `frame_test.go` `TestParseOuterHeader_AcceptsAllValidFrameTypes` `valid` slice | Append `frame.FrameTypePEConnect` as sixth element (F-SP2-004) |
| 8 | `frame.go` `OuterHeader.FrameType` field comment | `"identifies the frame kind (data, ctl, arq, fec, empty-tick)"` → `"identifies the frame kind (data, ctl, arq, fec, empty-tick, pe_connect)"` (F-SP3-003) |
| 9 | `frame_test.go` `TestFrameType_Valid` function-body comment at ~:501 | `"five canonical enum values"` → `"six canonical enum values"` (F-SP6-004; verified at `8eb54a5`: line 501 contains this stale count claim in the `TestFrameType_Valid` function description comment) |
| 10 | `frame_test.go` `TestParseOuterHeader_RejectsInvalidFrameType` inline comment at ~:540 | Change BOTH `"{0x01..0x05}"` → `"{0x01..0x06}"` AND `"canonical five enum values"` → `"canonical six enum values"` in the same edit (F-SP6-004; the v1.5 specification of this edit only covered the range update; the "canonical five" count claim on the same line is also stale) |

**F-SP3-003 adjudication (v1.3): item-8.** The `OuterHeader.FrameType` field
comment at `internal/frame/frame.go` line 68 reads:

```go
// FrameType identifies the frame kind (data, ctl, arq, fec, empty-tick).
```

This enumerates all canonical frame types by name. The adversary's finding is
upheld: this is not an illustrative example but a completeness claim — it lists
every kind in the same exhaustive form as the parallel doc comments and the
`frame_type` table row in ARCH-02. It follows the same pattern as items 3–5
(explicit enumeration / count claims in doc comments). Adding `FrameTypePEConnect`
without updating this comment leaves it incorrectly claiming five kinds. The
claim that "three consecutive incomplete-sweep instances" occurred in this file is
accurate: items 6 and 7 were missed in v1.1 (corrected in v1.2 by F-SP2-004),
and item 8 is now added by F-SP3-003. Rule: every sweep that adds a canonical
frame type constant MUST be accompanied by a grep for the type-name enumeration
comment in the `OuterHeader` struct, not only for the count-claim patterns
(`"five"`, `"0x05"`, `"FrameTypeFec"`). Record for future sweeps: the binding
pattern for field-comment enumerations is `grep -n "data, ctl"` or
`grep -n "frame kind"` on `internal/frame/frame.go`.

**Extended v1.3 sweep transcript (F-SP3-003 re-sweep, enumeration-aware patterns):**

Patterns run against `internal/frame/frame.go` and `internal/frame/frame_test.go`
at `8eb54a5`:

- `grep -n "data, ctl"` on `internal/frame/frame.go` → hit at line 68 (now item 8, enumerated above). No other matches.
- `grep -n "empty.tick"` on `internal/frame/*.go` → hits: `frame.go:68` (item 8). No additional locations.
- `grep -n "frame kind"` on `internal/frame/*.go` → hit at `frame.go:68` (same line, item 8 only).
- `grep -rn "data, ctl\|empty.tick\|frame kind" --include="*.go"` across all `internal/` → additional matches outside `internal/frame/`: `internal/session/auth.go:33` (`"empty-tick frames are accepted"`) and `internal/session/upstream.go:212` (`"empty-tick exemption"`). These are prose descriptions of frame semantics in the session package, not enumeration-completeness claims; they do not enumerate all frame types and are not updated by adding a new type. Adjudicated NOT items.
- `grep -n "FrameType identifies\|frame.FrameType\|type FrameType" --include="*.go"` on `internal/frame/frame.go` → `type FrameType byte` declaration (no enumeration claim); `// FrameType identifies the frame kind` at line 68 (item 8, already enumerated). No additional locations.

Sweep complete. Eight locations total. No further enumeration-completeness claims found.

No other files in the tree at `8eb54a5` contain "five values" or "five canonical"
assumptions anchored to the `0x05` upper bound per the extended sweeps above. The
`outerassembler/fuzz_test.go` `ft.Valid()` gate is self-adjusting and requires no
change (swept-clean; see adjudicated-clean section below).

**ARCH-02 amendment obligation (F-SP1-003):**

ARCH-02 §"Outer Header Format" (at `.factory/specs/architecture/ARCH-02-protocol-stack.md`,
verified at `8eb54a5`) contains the canonical single source of truth for the wire
frame types. The `frame_type` row currently reads:

```
| 1 | 1 | frame_type | u8 enum: data=0x01, empty_tick=0x02, ctl=0x03, arq=0x04, fec=0x05 |
```

`FrameTypePEConnect = 0x06` goes on the wire — it is the bootstrap frame type
the PE upstream connection uses. ARCH-02 is declared the "canonical single source
of truth for the outer header wire format" (verified at `8eb54a5` in ARCH-02 preamble).
The implementer MUST amend the `frame_type` row in the same commit that defines
`FrameTypePEConnect`:

```
| 1 | 1 | frame_type | u8 enum: data=0x01, empty_tick=0x02, ctl=0x03, arq=0x04, fec=0x05, pe_connect=0x06 |
```

This is a parallel obligation to the ARCH-08 §6.5 amendment required by Q2 — both
spec documents require the same commit. Additionally, `internal/frame/frame.go`
line 31 contains the comment `"Frame type constants (ARCH-02 §3.1)"` — this comment
remains accurate and does not require amendment, but the new constant MUST appear
beneath it with an `(ARCH-02 §3.1)` inline citation.

**BC-2.01.004:61 sibling amendment obligation (v1.12 — F-SP14-001, BINDING):**

`BC-2.01.004` (`.factory/specs/behavioral-contracts/ss-01/BC-2.01.004.md`) Postcondition 2
carries an outer-header layout table. Line 61 contains the `frame_type` row with the
byte-identical enum text:

```
| 1      | 1    | frame_type     | u8 enum: data=0x01, empty_tick=0x02, ctl=0x03, arq=0x04, fec=0x05 |
```

This row MUST be amended to:

```
| 1      | 1    | frame_type     | u8 enum: data=0x01, empty_tick=0x02, ctl=0x03, arq=0x04, fec=0x05, pe_connect=0x06 |
```

in the **SAME commit** that defines `FrameTypePEConnect` in `internal/frame/frame.go` —
exactly parallel to the ARCH-02:74 obligation above.

**Rationale for option (a) (add to obligation) over scope-out note:** Pass-8 adversarial
review (F-P8-008) named `BC-2.01.004:57 + ARCH-02:74` as the co-canonical pair for the
outer-header layout spec — they are not independent documents but twin faces of the same
wire-format contract. BC-2.01.004 v1.2 shows active BC↔ARCH sync practice: the two
documents are expected to move in lockstep on wire-format changes. If `FrameTypePEConnect`
is added to ARCH-02:74 and `frame.go Valid()` without updating BC-2.01.004:61, the BC would
enumerate 5 frame types while the code and ARCH-02 accept 6 — an observable post-ship
discrepancy that cannot be deferred to a maintenance note.

**Class-closure grep transcript (v1.12 — F-SP14-001 sweep; mandatory per 4th incomplete-sweep-class instance):**

This is the 4th instance of the incomplete-sweep-class (F-SP7-003, F-SP10-001, F-SP13-001,
F-SP14-001). Patterns run against `.factory/specs/` to find all wire-format spec locations
carrying the frame_type enum that require the same amendment:

| Pattern | Command | Hits | Files:Lines | Disposition |
|---------|---------|------|-------------|-------------|
| `arq=0x04, fec=0x05` | `grep -rn "arq=0x04, fec=0x05" .factory/specs/` | **2** | `BC-2.01.004:61` and `ARCH-02-protocol-stack.md:74` | BC-2.01.004:61 = Postcondition 2 outer-header layout table, frame_type row — **this amendment's edit target**. ARCH-02:74 = §"Outer Header Format" frame_type table row — already the Q3 ARCH-02 edit target. No third location. |
| `empty_tick=0x02` | `grep -rn "empty_tick=0x02" .factory/specs/` | **2** | `BC-2.01.004:61` and `ARCH-02-protocol-stack.md:74` | Same two locations as above pattern — confirms no paraphrased enum copies elsewhere in `.factory/specs/`. Both benign contexts identified; both are the canonical wire-format rows. No third sibling. |

**Conclusion:** Exactly 2 canonical locations for the wire-format frame_type enum row in
`.factory/specs/`: `BC-2.01.004:61` and `ARCH-02:74`. Both are now explicitly named as
same-commit parallel amendment obligations. Pass-15 cannot find a third wire-format spec
sibling.

**`dialLoop` bootstrap flip obligation (FO-PE-LOOP-001):** `internal/upstreamdial/connector.go`
`dialLoop` (verified at `8eb54a5`) currently sets:

```go
cf := halfchannel.ChannelFrame{
    FrameType: halfchannel.FrameTypeData,
}
```

This story flips it to:

```go
cf := halfchannel.ChannelFrame{
    FrameType: frame.FrameTypePEConnect,
}
```

`halfchannel` aliases only `FrameTypeData` and `FrameTypeEmptyTick` (verified at
`8eb54a5`). `frame.FrameTypePEConnect` must be imported directly from
`internal/frame`. This introduces a direct `frame` import in `upstreamdial` —
consistent with the Q2 ruling (same import-set extension, covered by the same
§6.4 amendment).

**Receive loop discrimination contract:** after `frame.ReadOuterFrame`, the
receive goroutine applies:

```
if hdr.FrameType == frame.FrameTypePEConnect {
    // bootstrap acknowledgment path (or silent discard if no reply needed)
} else {
    // data/ctl/arq/fec frame — pass to FrameFn callback
}
```

The exact bootstrap acknowledgment protocol is determined by the story-writer at
elaboration time. If no reply is defined in this story's scope, bootstrap frames
are silently discarded; the upstream router's PE-CONNECT is treated as a
registration event only.

**Cite:** `internal/frame/frame.go` (FrameType constants, Valid(), doc comments, verified at `8eb54a5`);
`internal/frame/frame_test.go` (`TestFrameType_Valid` just_above_max case, `TestParseOuterHeader_RejectsInvalidFrameType` invalids slice, verified at `8eb54a5`);
`internal/halfchannel/halfchannel.go` (FrameType aliases, verified at `8eb54a5`);
`internal/upstreamdial/connector.go` `dialLoop` bootstrap construction (verified at `8eb54a5`);
`.factory/specs/architecture/ARCH-02-protocol-stack.md` frame_type row (verified at `8eb54a5`).

---

## Q4 — `arqsend.Retransmitter` wiring into `runRouter`

> **[v1.8 supersession annotation — F-SP10-001]** Q4's test-role content below is SUPERSEDED: the arqsend/net.Dial(ListenAddr) dispatch shape was replaced by peWriteFixture injection at the accepted PE conn (Q9 §9.1/§9.4, F-SP2-001). Q4's PRODUCTION-wiring ruling (arqsend.New placement) remains historically accurate but is NOT part of this story's test topology. Do not implement from this section — Q9 governs.

**Ruling: `arqsend.New` is called inside `runRouter` after the connector is
constructed and started (Phase g, after Phase f). The `Retransmitter` is used
only in the integration test that exercises E-FWD-001 under sustained load; it
is NOT wired into the production `runRouter` datapath for this story. A
per-test construction inside the test body is the correct shape.**

**Derivation from the `arqsend` API (verified at `8eb54a5`):**

```go
// internal/arqsend/arqsend.go
func New(a *arq.ARQ, env outerassembler.Envelope, opts ...Option) *Retransmitter
func (r *Retransmitter) Retransmit(oldSeq, newSeq uint32, now time.Time, dispatch Dispatch) error
type Dispatch func(wire []byte) error
```

`arqsend.Retransmitter` is pure-core (no goroutines, no I/O). Its `Retransmit`
method requires an `*arq.ARQ` and an `outerassembler.Envelope`. In the integration
test context:

- A test-internal `*arq.ARQ` (constructed via `arq.New`) tracks retransmit state.
- The `Dispatch` callback sends wire bytes to the test router's data-plane listener
  address via `net.Dial` + `conn.Write` — the same loopback pattern
  `TestRunRouter_PE_EFWD001ReconfirmationUnderLoad` used in S-7.04-FU-PE-CONNECTOR
  (verified at `8eb54a5`).

The production `runRouter` does NOT need a persistent `Retransmitter` instance:
the production ARQ retransmit path is node-side (access nodes retransmit), not
router-side. Wiring a `Retransmitter` into `runRouter` would be production-dead
code outside this test's scope.

**Test construction point:** inside `TestRunRouter_PE_EFWD001ExhaustionUnderLoad`
(or equivalent name at elaboration), after the PE router is started and the
upstream fixture connection is established:

```go
a := arq.New(arq.Config{...})
rt := arqsend.New(a, outerassembler.Envelope{}, arqsend.WithChanID(1))
// dispatch: write wire bytes to the router's ListenAddr
dispatch := func(wire []byte) error {
    conn, err := net.Dial("tcp", routerListenAddr)
    if err != nil { return err }
    defer conn.Close()
    _, err = conn.Write(wire)
    return err
}
```

**Lifecycle:** the `Retransmitter` has no Stop/Close method (pure-core, no
goroutines). Its lifecycle is bounded to the test function.

**`cmd/switchboard` import impact:** `arqsend` is already imported transitively
through the test binary; no new production import is needed.

**Cite:** `internal/arqsend/arqsend.go` `New`, `Retransmit`, `Dispatch`
(verified at `8eb54a5`); `internal/arq/arq.go` `New` (verified at `8eb54a5`);
`cmd/switchboard/router_pe_connector_test.go` `TestRunRouter_PE_EFWD001ReconfirmationUnderLoad`
(loopback pattern, verified at `8eb54a5`).

---

## Q5 — E-FWD-001 exhaustion integration test shape

> **[v1.8 supersession annotation — F-SP10-001]** Q5's test shape below is SUPERSEDED in three respects: (1) arqsend/ListenAddr dispatch → peWriteFixture at the accepted PE conn (Q9 §9.1/§9.4); (2) testenv.New/RouterHandle.Restart harness → real runRouter goroutine pattern (Q9.3 — testenv.Restart never calls SetFrameCallback, nil FrameFn, vacuous assertion); (3) RouterHandle.Mode() ModePE poll → peWriteFixture.accepted establishment gate per the F-SP7-001 binding three-observable semantics. Do not implement from this section — Q9 + the corrected-observables block govern.

**Ruling: the test wires a real PE connection between a test router and an
upstream fixture, sends frames via `arqsend.Retransmitter.Retransmit` (dispatch
writes to the router's data-plane `ListenAddr`), and asserts `"E-FWD-001"` appears
in the router's writer output. Path exhaustion is achieved by setting the
`interfaceSet` to `[]routing.InterfaceID{arrivalIface}` only — i.e., the arrival
interface is the only forwarding candidate — which causes
`FrameArrivalHandler.OnFrameArrival` to call `SplitHorizon.Forward` with no
eligible output interface and emit E-FWD-001.**

**Exact E-FWD-001 emission string (verified at `8eb54a5`, from
`internal/routing/on_frame_arrival.go`):**

```
"all paths split-horizon-blocked: frame dropped (checksum=0x%08x iface=%d) (BC-2.02.008 E-FWD-001)"
```

**Assertion key:** `"E-FWD-001"` — the spec-anchored event code, NOT
`"split-horizon-blocked"` or `"all paths split-horizon"`. This is the lesson
from F-P11-001 (adversary pass-11 of S-7.04-FU-PE-CONNECTOR, committed at
`8eb54a5`): space vs hyphen mismatches make a vacuous negative assertion. Use the
event code tag that is stable across prose rewording of the emission text.
The mutation pin test `TestScanForLine_DetectsEFWD001ProductionEmission`
(verified at `8eb54a5` in `router_pe_connector_test.go`) validates this key.

**Test infrastructure required:**

The `testenv` package (position 23, verified at `8eb54a5`) provides
`testenv.NewWithRouters(ctx, t, n int)` which starts `n` in-process routers.
`testenv.New(ctx, t)` starts a single-router environment. For the E-FWD-001
exhaustion test, a single router with one PE upstream fixture is sufficient:

1. Start the test router via `testenv.New(ctx, t)`.
2. The test PE upstream is the existing `peLn` fixture listener already
   created inside `newEnv` (verified at `8eb54a5` in `testenv.go` — it is
   a `net.Listen("tcp", "127.0.0.1:0")` that accepts and drains connections,
   available via `e.PERouterAddr(t)`).
3. Restart the test router with `UpstreamRouters: []string{e.PERouterAddr(t)}`
   via `RouterHandle.Restart(t, cfg)` so the connector dials the fixture.
4. Wait for the receive loop to be active (poll `RouterHandle.Mode()` for `ModePE`).
5. Send frames via `arqsend.Retransmitter` dispatching to the router's
   `cfg.ListenAddr` (the data-plane TCP listener). Frames must be
   well-formed `outerassembler.Assemble` output to pass `ParseOuterHeader` and
   reach `FrameArrivalHandler.OnFrameArrival`.
6. Path exhaustion requires the forwarding table to have only the arrival
   interface as the eligible interface. Achieving this in `testenv` requires
   the story-writer to assess: does `testenv`'s routing table naturally have
   only one registered interface (the incoming node connection), making any
   frame from the upstream fixture arrive with the same `InterfaceID` as its
   only forwarding entry? If so, no special setup is needed. If not, a
   dedicated loopback fixture must pre-register a forwarding entry pointing
   back to the arrival interface. This is an elaboration decision.

**No new `testenv` API beyond what already exists is required** for Q5. The
existing seams (`NewWithRouters`, `PERouterAddr`, `RouterHandle.Restart`,
`RouterHandle.Mode`, `Env.StartRouter`) are sufficient.

**Cite:** `internal/testenv/testenv.go` `NewWithRouters`, `New`, `PERouterAddr`,
`StartRouter`, `RouterHandle.Restart`, `RouterHandle.Mode` (verified at `8eb54a5`);
`internal/routing/on_frame_arrival.go` E-FWD-001 emission line (verified at `8eb54a5`);
`cmd/switchboard/router_pe_connector_test.go` `TestScanForLine_DetectsEFWD001ProductionEmission`
(F-P11-001 mutation pin, verified at `8eb54a5`).

---

## Q6 — Concurrency contract: receive goroutine lifecycle vs `Connector.Stop`/`ReloadAddrs`

**Ruling: the receive goroutine is owned by `dialLoop` and exits when the
per-address context (`ctx context.Context` in `dialLoop`) is cancelled. It MUST
NOT hold any shared mutable state beyond the `net.Conn` passed to it by `dialLoop`.
`conn.Close()` (called by `dialLoop` after `maintainConn` returns) signals the
receive goroutine's `frame.ReadOuterFrame` loop to exit via `io.EOF` or
`net.Error`. No separate stop channel is needed for the receive goroutine; it
drains naturally when the connection closes.**

**Derivation:**

The shipped `dialLoop` in `internal/upstreamdial/connector.go` (verified at
`8eb54a5`) follows this pattern for each established connection:

1. `conn, err := dialer.DialContext(ctx, "tcp", addr)` — dial
2. `outerassembler.Assemble` + `conn.Write` — bootstrap
3. `c.connectedCount.Add(1)` — increment
4. `c.maintainConn(addr, conn, ctx.Done(), keepaliveTick.C)` — blocks until connection dead or stop
5. `newCount := c.connectedCount.Add(-1)` + `_ = conn.Close()` — teardown

The receive goroutine must be started between steps 3 and 4, and must use the
same per-address `ctx.Done()` channel as `maintainConn`. When `ctx` is cancelled
(via `addrCancel.cancel()` from `reconcile` or via `stopCh` close from `Stop()`),
the per-address context is cancelled, `DialContext` returns, and any ongoing
`frame.ReadOuterFrame(conn)` returns because the underlying `net.Conn` is closed.

**Exactly-once semantics:** the concurrent-transition lesson from F-P29-001
(EC-004 concurrent-drop race, S-7.04-FU-PE-CONNECTOR) applies symmetrically to
the receive loop. The receive goroutine MUST NOT access `c.connectedCount` or any
other shared state. The `connectedCount` lifecycle is owned by `dialLoop` alone
(increment after step 3, decrement via `Add(-1)` return value after step 5). The
receive goroutine's only output is calling the `FrameFn` callback with received
bytes — a stateless action from the concurrency perspective. **(amended v1.6 —
F-SP6-001):** The receive goroutine has exactly TWO outputs: (1) the `FrameFn`
callback (data path), and (2) `_ = conn.Close()` on read-error exit (teardown
signal). The "only output is FrameFn" characterisation of prior versions is
retracted for the abnormal-exit path; it remains accurate on the happy path where
the goroutine exits because `conn.Close()` was called by `dialLoop` teardown
(which has already closed the conn before the goroutine exits).

**Goroutine lifecycle contract:**

```
dialLoop goroutine:
    1. dial
    2. bootstrap
    3. connectedCount.Add(+1)
    4. START receive goroutine (owns conn, ctx.Done())
    5. maintainConn(addr, conn, ctx.Done(), tick)  ← blocks
    6. receive goroutine exits (conn closed or ctx done)
    7. connectedCount.Add(-1) — must occur AFTER receive goroutine exits
       OR be independent of receive goroutine state (no shared write)
    8. conn.Close()
    9. [if reconnecting] WAIT for receive goroutine from previous iteration to
       fully exit before beginning step 1 of the next dial iteration
```

The ordering between steps 6 and 7 is not constrained by shared state — the
receive goroutine does not modify `connectedCount`. But `dialLoop` MUST wait
for the receive goroutine to exit before looping to reconnect. This is a
**per-reconnect-iteration join requirement (F-SP1-005):**

> **Q6 per-reconnect join (binding):** Before `dialLoop` begins dialing a new
> connection for the same address (step 1 of a reconnect iteration), it MUST
> join — that is, block until completion of — the receive goroutine from the
> previous iteration. A `sync.WaitGroup` (Add(1) before step 4, Done() in the
> receive goroutine's deferred return) or a per-connection `done chan struct{}`
> (closed by the receive goroutine on exit) satisfies this requirement. The
> join MUST occur at the end of each dial iteration, before the reconnect
> backoff sleep and before the next dial attempt. Failure to join creates a
> goroutine-leak vector: a "flapping" upstream (rapid connect/disconnect) can
> accumulate O(N) receive goroutines reading from closed or recycled connections.

The AC-005 race test (covering `Connector.Stop()` teardown) MUST also cover a
**flap cycle** — that is, at least one complete connect-then-disconnect-then-reconnect
cycle — not only final teardown. A test that only exercises `Stop()` after one
successful connection does not exercise the per-iteration join path.

> **[v1.3 AC-005 harness placement — F-SP3-002]** The Q6 flap-cycle requirement
> belongs to `internal/upstreamdial/connector_test.go`, NOT to `router_pe_receive_test.go`
> or the `peWriteFixture`. See Q9 §9.2 and FCL row 4 (F-SP3-002 correction). The
> flap-cycle shape already established in `connector_test.go` at `8eb54a5`
> (`TestConnector_BackoffParameters` phases 2→3: `heldConn` accept-and-drain +
> server-side `conn.Close()` to trigger reconnect, verified at `8eb54a5`) is the
> correct template. The new AC-005 test calls `SetFrameCallback` on the connector
> with a counting `FrameFn`, runs a held-connection phase, closes the server-side
> conn to force reconnect, opens a second listener to produce a second connection,
> and asserts (a) no goroutine leak (via `goleak.VerifyNone` or equivalent), (b) the
> `FrameFn` is called for frames arriving on both connections, (c) `Connector.Stop()`
> after both connections have been held-then-dropped completes without hang. This
> test lives entirely in `connector_test.go` and requires no `runRouter` or
> `peWriteFixture` involvement.

**`Stop()` teardown correctness:** `Connector.Stop()` calls `stopOnce.Do(close(c.stopCh))`
then `<-c.doneCh`. `c.doneCh` is closed by `reconcileLoop` after all
`addrCancel.done` channels are drained. For this to cover receive goroutines,
each `addrCancel.done` channel must not be closed until the receive goroutine for
that address has exited. The implementer must ensure the per-address `done chan struct{}`
is closed only after both `maintainConn` AND the receive goroutine have returned.
The per-iteration join (above) is a prerequisite for the teardown join to be
sound: without it, the `doneCh` close can race against a goroutine from a prior
iteration that was never joined.

**Cite:** `internal/upstreamdial/connector.go` `dialLoop`, `reconcile`,
`addrCancel` (verified at `8eb54a5`); F-P29-001 concurrent-drop race ruling
(S-7.04-FU-PE-CONNECTOR adversary pass-29, noted in DELIVERY at `8eb54a5`).

---

## Q7 — BC-2.06.003 PC-1 Failed-state observable: emission point and integration assertion key

**Ruling: BC-2.06.003 PC-1 "failed" status is emitted by
`internal/metrics.PathEntryFromSnapshot` in `internal/metrics/handlers.go` when
`PathSnapshot.Failed == true`. The `FrameArrivalHandler.OnFrameArrival` path that
emits E-FWD-001 does NOT directly emit BC-2.06.003 PC-1 — the two observables are
orthogonal. This story's BC-2.06.003 obligation is to demonstrate that the
send+forward path traversal (arqsend → PE receive loop → OnFrameArrival) is live
and exercised; the "failed" status emission is a downstream metrics concern gated
on path liveness failures, not on split-horizon drops.**

**Derivation:**

BC-2.06.003 PC-1 (verified at `8eb54a5`) defines `status: "failed"` as deriving
from `PathSnapshot.Failed == true`, which is set only when a previously-alive path
stops responding (`!firstProbe` liveness check in the `paths` package). The
`metrics.PathEntryFromSnapshot` function (verified at `8eb54a5` in
`internal/metrics/handlers.go`) implements:

```go
case snap.Failed:
    status = "failed"
```

This path is triggered by keepalive liveness failures — a path that was active
and went silent. It is NOT triggered by split-horizon drops (E-FWD-001). The two
are independent:

- E-FWD-001 fires because a frame's only forwarding option is the arrival interface.
  This is a topology condition, not a liveness failure.
- `status: "failed"` fires because keepalive probes stop receiving replies.
  This is a liveness condition.

**Consequence for this story's BC-2.06.003 discharge:**

The stub story's BC-2.06.003 PC-1 trace ("Failed-state via retransmit-driven path
exhaustion") conflates two independent mechanisms. The story-writer must clarify at
elaboration time which mechanism is being asserted:

- **Option A (E-FWD-001 path):** Assert that `"E-FWD-001"` fires under sustained
  retransmit load when the routing table has only the arrival interface. This is
  the AC-004 postcondition 1 discharge (the primary obligation re-anchored here).
  It does NOT require `status: "failed"` from `metrics`.

- **Option B (path-failed status path):** Assert that after a PE upstream
  connection drops, `sbctl paths list` returns `status: "failed"`. This is
  a follow-on behavioral property owned by `S-BL.PATH-FAILED-STATUS` infrastructure
  (already shipped at `8eb54a5`) and does not require the receive loop per se.

**Ruling: Option A is the operative discharge for this story.** The
BC-2.06.003 PC-1 trace in the stub is inherited from the original AC-004
framing and does not add a separate `status: "failed"` integration assertion
obligation in this story beyond what E-FWD-001 already covers.

**BC ambiguity flag (do not resolve unilaterally):** The stub story's
`bc_traces` section lists `BC-2.06.003` with the description "PC-1 Failed-state
via retransmit-driven path exhaustion." Reading BC-2.06.003 PC-1 literally, the
"failed" status field is about path liveness (`PathSnapshot.Failed`), not about
split-horizon drops. The PO should confirm whether (a) BC-2.06.003 PC-1 is
traced here to document that the full send+forward path enables future
`status: "failed"` path liveness testing, or (b) there is a spec-level linkage
between E-FWD-001 exhaustion and `status: "failed"` that I did not find. This
ambiguity does not block implementation of Q5 but must be resolved before AC
finalization.

**Integration assertion key for BC-2.06.003 (if Option A):** assert that the
writer output contains `"E-FWD-001"` (same key as Q5). No `"failed"` string
assertion is required by this story under Option A.

**Cite:** `internal/metrics/handlers.go` `PathEntryFromSnapshot` (verified at
`8eb54a5`); `internal/paths/paths.go` `PathSnapshot.Failed` field (verified at
`8eb54a5`); `internal/routing/on_frame_arrival.go` E-FWD-001 emission (verified
at `8eb54a5`); BC-2.06.003 v1.16 PC-1.

---

## Q8 — Production wiring: making E-FWD-001 reachable via the PE receive callback (F-SP1-001)

**Context (finding F-SP1-001):** The v1.0 note's Q1/Q2 rulings wired the PE
receive callback to `routing.RouteFrame`. The adversarial pass established that
this wiring cannot emit E-FWD-001: `RouteFrame` delegates to `SVTNRoute` (verified
at `8eb54a5` in `internal/routing/routing.go` — `RouteFrame` returns `SVTNRoute(hdr, payload, r)`);
`SVTNRoute` performs admission + forwarding-table lookup and returns `ErrNoForwardingEntry`
on miss, but NEVER calls `FrameArrivalHandler.OnFrameArrival` (verified: zero
production callers of `OnFrameArrival` or `NewFrameArrivalHandler` exist in `cmd/`
at `8eb54a5` — grep confirmed). `ErrAllPathsSplitHorizon` (the source of E-FWD-001)
is emitted exclusively by `SplitHorizon.Forward`, which is called only from
`OnFrameArrival` (verified at `8eb54a5` in `internal/routing/on_frame_arrival.go`).
Additionally, `runRouter` at `8eb54a5` constructs the `router` via
`buildRouter(admission.NewAdmittedKeySet(), routerLogger)` with an empty forwarding
table — AC-004's arqsend frames would die at admission (`ErrNotAdmitted`) before
reaching any forwarding decision.

**Ruling: the PE receive `FrameFn` callback in `runRouter` MUST route through a
properly-constructed `routing.FrameArrivalHandler` rather than calling
`routing.RouteFrame` directly. This is wiring option (a): `runRouter` constructs
a `*routing.FrameArrivalHandler` at startup (after Phase b), passes a closure
wrapping `handler.OnFrameArrival(...)` as the `FrameFn` to `connector.SetFrameCallback`,
and does NOT change the `netingress.Serve` path (which retains its existing
`routing.RouteFrame` closure).**

**Q8 wiring specification:**

### 8.1 — FrameArrivalHandler construction

`runRouter` constructs a `*routing.FrameArrivalHandler` immediately after the
router is built (after Phase b, before Phase c). Construction requires a
`*multipath.DropCache`:

```go
// After router := buildRouter(...):
dc := multipath.NewDropCache(multipath.DefaultDropCacheSize)
arrivalHandler := routing.NewFrameArrivalHandler(dc)
routing.WithFrameArrivalLogger(routerLogger)(arrivalHandler)
```

Verified at `8eb54a5`:
- `multipath.NewDropCache(capacity int) *DropCache` — `internal/multipath/multipath.go`
- `multipath.DefaultDropCacheSize = 10_000` — `internal/multipath/multipath.go`
- `routing.NewFrameArrivalHandler(dc *multipath.DropCache) *FrameArrivalHandler` — `internal/routing/on_frame_arrival.go`
- `routing.WithFrameArrivalLogger(l Logger) func(*FrameArrivalHandler)` — `internal/routing/on_frame_arrival.go`

### 8.2 — FrameFn closure wired to connector

The `FrameFn` callback set on the connector routes through `OnFrameArrival`. The
full signature of `OnFrameArrival` is (verified at `8eb54a5`):

```go
func (h *FrameArrivalHandler) OnFrameArrival(
    frameBytes []byte,
    arrivalIface InterfaceID,
    interfaceSet []InterfaceID,
    fn ForwardFunc,
) error
```

The `FrameFn` closure must supply:
- `arrivalIface` — the PE connection's logical `routing.InterfaceID`. A fixed
  constant (e.g. `routing.InterfaceID(1)` or a named PE-interface ID) is
  acceptable for this story; the value uniquely identifies the PE upstream path.
  The implementer assigns this at construction time; the exact value is an
  elaboration decision.
- `interfaceSet` — the set of forwarding candidates. For the E-FWD-001 exhaustion
  test, the interface set MUST be `[]routing.InterfaceID{peIfaceID}` only (the
  arrival interface is the sole candidate), which guarantees `SplitHorizon.Forward`
  returns `ErrAllPathsSplitHorizon`. In production, the interface set is populated
  from the router's forwarding table or a registry of connected data-plane nodes.
  *(amended v1.9 — F-SP11-003: dangling pointer removed; production interface-set
  population is out of scope for this story — §8.5 governs the test-scoped set)*
- `fn ForwardFunc` — the forward function that actually sends bytes to an interface.
  In production this dials the destination. In the integration test a no-op or
  capture function is acceptable (the E-FWD-001 path never calls `fn` because all
  paths are split-horizon blocked).

**Skeleton (illustrative):**

```go
peIfaceID := routing.InterfaceID(1) // PE upstream arrival interface
frameFn := upstreamdial.FrameFn(func(hdr frame.OuterHeader, raw []byte) error {
    // interfaceSet: for test = [peIfaceID] only (exhaustion); in production,
    // consult the forwarding table or registered interface registry.
    return arrivalHandler.OnFrameArrival(
        raw,
        peIfaceID,
        []routing.InterfaceID{peIfaceID}, // single-interface = guaranteed exhaustion in test
        func(iface routing.InterfaceID, frameBytes []byte) error {
            // production: forward to iface's connection; test: capture or discard
            return nil
        },
    )
})
connector.SetFrameCallback(frameFn)
```

### 8.3 — Import graph impact

`cmd/switchboard/mgmt_wire.go` already imports `internal/routing` (verified at
`8eb54a5`) and `internal/netingress`. Adding `internal/multipath` is a new import
at the `cmd/switchboard` layer. `cmd/switchboard` is at the top of the DAG (no
position constraint applies to the binary); this import is unconditionally legal.
No ARCH-08 §6.4 registration is required for `cmd/switchboard` imports.

Verified at `8eb54a5`: `cmd/switchboard/mgmt_wire.go` imports list includes
`internal/routing` and does NOT include `internal/multipath` — adding it is the
only import change in `cmd/switchboard`.

### 8.4 — `netingress.Serve` path unaffected

The `netingress.Serve` data-plane accept loop in `runRouter` retains its existing
wiring `routing.RouteFrame(hdr, payload, router)` unchanged (verified at
`8eb54a5`). That path is for frames arriving from connected access nodes on the
data-plane TCP listener. The `FrameArrivalHandler` path is strictly the PE
upstream receive goroutine. The two paths are disjoint and do not share router
state beyond the `*routing.Router` itself (which is safe for concurrent use via
its internal `sync.RWMutex`).

### 8.5 — Forwarding table + admission state for the integration test

AC-004's arqsend retransmit frames must survive admission and reach the forward
decision. The `runRouter` construction at `8eb54a5` uses
`admission.NewAdmittedKeySet()` (empty set — fail-closed). Frames from the test
would die at `ErrNotAdmitted` before `OnFrameArrival` is even invoked.

**The integration test MUST:**

1. Call `router.RegisterForwardingEntry(svtnID, nodeAddr, authKey)` with an entry
   that matches the test frame's `hdr.SVTNID` and `hdr.DstAddr` — so `SVTNRoute`
   does not return `ErrNoForwardingEntry` before the frame even reaches the
   handler. (Note: with the `FrameArrivalHandler` wiring, the frame goes directly
   to `arrivalHandler.OnFrameArrival`; `SVTNRoute` is NOT called on this path.
   `OnFrameArrival` does not consult the forwarding table — it operates on raw
   bytes and the drop cache only. The forwarding table constraint above applies to
   the `netingress` path, not the PE receive path.)
2. Ensure the outerassembler `Envelope` used to construct arqsend frames carries
   an `FrameAuthKey` matching a key the test supplies. Because the PE receive
   `FrameFn` goes directly to `OnFrameArrival` (bypassing `RouteFrame`'s HMAC
   check), HMAC admission is NOT checked on the PE receive path in this design.
   The test frames therefore do not need a valid HMAC. The story-writer must
   confirm this is acceptable for the E-FWD-001 exhaustion test or elect to add
   an explicit HMAC-verify step in the `FrameFn` closure.
3. Set `interfaceSet = []routing.InterfaceID{peIfaceID}` in the `FrameFn` closure
   to guarantee all paths are split-horizon blocked and E-FWD-001 fires on every
   non-bootstrap frame.

### 8.6 — Blast radius on existing RouteFrame callers

`routing.RouteFrame` callers at `8eb54a5` (grep-verified):
- `cmd/switchboard/mgmt_wire.go` `runRouter` Phase f ingress closure: UNAFFECTED — this path stays as-is.
- `internal/netingress/integration_test.go`: UNAFFECTED — test code, not modified by this story.
- `internal/arqsend/integration_test.go`: UNAFFECTED — test code using `RouteFrame` directly.
- `internal/admission/failure_counter_adversarial_test.go` `TestRouteFrame_EndToEnd_EADMAlertMessageFormat`: UNAFFECTED — test code.
- `internal/routing/example_test.go`: UNAFFECTED — example tests.
- `internal/routing/routing_internal_test.go`: UNAFFECTED — internal tests.

No production caller of `routing.RouteFrame` is modified. The `netingress` path
is explicitly preserved. `RouteFrame`'s signature and semantics are unchanged.

**Cite:** `internal/routing/routing.go` `RouteFrame`, `SVTNRoute` (verified at `8eb54a5` — `RouteFrame` returns `SVTNRoute(...)`, no `OnFrameArrival` call);
`internal/routing/on_frame_arrival.go` `OnFrameArrival`, `NewFrameArrivalHandler`, `WithFrameArrivalLogger` (verified at `8eb54a5`);
`internal/multipath/multipath.go` `NewDropCache`, `DefaultDropCacheSize` (verified at `8eb54a5`);
`internal/routing/split_horizon.go` `SplitHorizon.Forward`, `ErrAllPathsSplitHorizon` (verified at `8eb54a5`);
`cmd/switchboard/mgmt_wire.go` `runRouter` import list (verified at `8eb54a5` — `routing` present, `multipath` absent).

---

## Q9 — E-FWD-001 injection topology: upstream fixture write path and harness rule (F-SP2-001, F-SP2-002, F-SP2-003)

**Context:** Pass-2 adversarial review established three interlocking defects in the Q4/Q5
injection model:

- **F-SP2-001 (CRITICAL [spec-defect]):** Q8 wires production emission onto the PE receive
  path (`FrameFn → OnFrameArrival` on frames arriving over the DIALED upstream conn). But
  Q4/Q5's AC-004 test-injection vector has `arqsend.Dispatch` dial `cfg.ListenAddr` (the
  data-plane TCP listener) and write wire bytes there. Those bytes enter via
  `netingress.Serve → RouteFrame` — a physically disjoint socket from the dialed PE conn.
  `RouteFrame` does NOT call `OnFrameArrival` (verified at `8eb54a5`; zero production
  callers of `OnFrameArrival` from the netingress path). AC-004 as specified in v1.1 is
  undischargeable: the frame never reaches the FrameFn callback.

- **F-SP2-002 (HIGH [spec-gap]):** No write-capable upstream fixture exists. Both
  `startPEListenerFixture` (in `cmd/switchboard/router_pe_connector_test.go`, verified at
  `8eb54a5` — accept loop reads-and-drains, zero `Write` calls) and testenv's `peLn`
  (in `internal/testenv/testenv.go`, verified at `8eb54a5` — "Drain and close: we just
  need the connection to be accepted", zero `Write` calls) are read-only drains.

- **F-SP2-003 (MED [spec-defect]):** AC-004's precondition starts the router via
  `testenv.New`/`Restart`. `testenv.Restart` builds a bare `upstreamdial.New` and
  NEVER calls `SetFrameCallback` (verified at `8eb54a5` in `testenv.go` `Restart` —
  it calls `upstreamdial.New(...).Start()` with no callback wiring). `runRouter`
  (the real production function in `mgmt_wire.go`) is where `SetFrameCallback` will
  be called per the Q8 ruling; `testenv.Restart` bypasses this entirely. Any AC
  asserting that `OnFrameArrival` is reached therefore CANNOT use `testenv.Restart`;
  it must use the real `runRouter` goroutine pattern.

### 9.1 — Injection topology ruling (supersedes Q4 arqsend Dispatch and Q5 test shape)

**Ruling: option (b) — the upstream fixture assembles and writes one assembled outer frame
directly to the accepted PE connection. `arqsend.Retransmitter` is NOT used as the frame
producer in the AC-004 exhaustion integration test.**

The correct injection topology for AC-004:

```
PE upstream fixture (accepted conn)
    ──WRITES assembled outer frame──►  dialed PE conn in runRouter
                                           │
                                       PE receive goroutine in upstreamdial.Connector
                                           │ (via frame.ReadOuterFrame)
                                       FrameFn callback in runRouter
                                           │ (arrivalHandler.OnFrameArrival)
                                       SplitHorizon.Forward → ErrAllPathsSplitHorizon
                                           │
                                       E-FWD-001 logged in writer output  ◄── assert here
```

The frame the fixture writes is a valid `outerassembler.Assemble` output — the same
wire format that `frame.ReadOuterFrame` (new, defined by this story) expects. The fixture
uses `outerassembler.Assemble(cf, sackBitmap, env)` with a non-bootstrap `FrameType`
(e.g. `frame.FrameTypeData`) so it passes the `FrameTypePEConnect` discard check in the
receive goroutine and reaches `OnFrameArrival`. A zero `outerassembler.Envelope` is
sufficient; HMAC is not checked on the PE receive path (Q8 §8.5 ruling confirmed clean
per adjudicated-clean section below).

### 9.1a — Byte-contract pinning test obligation (F-SP3-001)

**Ruling: the AC-002/AC-004 test suite MUST include a false-duplicate-collision
pin test that directly validates the full-frame reconstruction is wired correctly.**

The failure mode if `FrameFn raw` is payload-only: two frames that differ only in
outer-header bytes (e.g. different `SrcAddr` fields in the `OuterHeader`) produce
the same `crc32.ChecksumIEEE(payload)` checksum — the second frame is silently
suppressed as a loop duplicate by `OnFrameArrival`'s drop-cache check. AC-004 as
specified does not catch this because it injects only one frame (cache always misses
on the first injection). A payload-only-wired `FrameFn` would produce a green AC-004
with the silent-failure mode undetected.

**Pin test shape (binding for story-writer):**

```
Pin test: TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader
  Precondition: inject two frames via peWriteFixture.WriteFrame:
    - Frame A: assembled with OuterHeader.SrcAddr = [8]byte{0x01, ...}
    - Frame B: assembled with OuterHeader.SrcAddr = [8]byte{0x02, ...}
    - Both frames have identical payload content
    - Both frames use FrameTypeData (non-bootstrap; pass PE-CONNECT discard check)
  Assertion: E-FWD-001 fires TWICE in the writer output
             (i.e. both frames independently reach OnFrameArrival and independently
             exhaust split-horizon; the second frame is NOT suppressed as a
             false-duplicate cache hit)
  What it proves: crc32 was computed over full-frame bytes (header+payload),
                  not over payload-only — if payload-only, identical payloads would
                  produce crc32 collision on Frame B and suppress it as a duplicate,
                  yielding only ONE E-FWD-001 emission.
```

This test is placed in `router_pe_receive_test.go` alongside AC-002. It is the
spec-level assertion obligation that pins the byte-contract at the observable
behaviour level: no test of the internal reconstruction path is needed; the
observable (two vs one E-FWD-001 emissions) is sufficient.

**Cite:** `internal/routing/on_frame_arrival.go` `OnFrameArrival` — `crc32.ChecksumIEEE(frameBytes)` drop-cache key (verified at `8eb54a5`); `internal/multipath/multipath.go` `AddIfAbsent` — compound key `(checksum, arrivalInterfaceID)` (verified at `8eb54a5`); `outerassembler.Assemble` — `Envelope.SrcAddr` feeds into the serialised outer header bytes (verified at `8eb54a5` in `internal/outerassembler/assemble.go`).

### 9.2 — Write-capable upstream fixture specification (F-SP2-002)

**Fixture placement: test-local, same file as other `runRouter` integration tests —
`cmd/switchboard/router_pe_receive_test.go` (NEW, per FCL row 7).**

A testenv seam is NOT required and would incur ARCH-08 position-23 implications
(testenv imports `outerassembler` at position 8; that edge is already present per
ARCH-08 §6.5 v2.8, so adding a `WriteFrame` helper would not add a new import edge —
but it would couple testenv's API surface to PE-frame-injection concerns that are local
to this story's test file). The lightest lawful option is a test-local fixture struct,
consistent with the pattern already used in `router_pe_connector_test.go`
(`startPEListenerFixture`).

**Fixture shape:**

```go
// peWriteFixture is a test-local upstream fixture that accepts one connection
// and exposes WriteFrame so the test can inject assembled outer frames into the
// PE receive goroutine.
type peWriteFixture struct {
    addr     string
    accepted chan net.Conn // buffered(1); receives the accepted conn
    ln       net.Listener
}

func startPEWriteFixture(t *testing.T) *peWriteFixture {
    t.Helper()
    ln, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil {
        t.Fatalf("startPEWriteFixture: Listen: %v", err)
    }
    t.Cleanup(func() { _ = ln.Close() })
    f := &peWriteFixture{addr: ln.Addr().String(), accepted: make(chan net.Conn, 1), ln: ln}
    go func() {
        conn, err := ln.Accept()
        if err != nil {
            return
        }
        // Drain incoming bytes (connector writes bootstrap + keepalives).
        go func(c net.Conn) {
            buf := make([]byte, 4096)
            for {
                if _, err := c.Read(buf); err != nil {
                    return
                }
            }
        }(conn)
        f.accepted <- conn
    }()
    return f
}

// WriteFrame writes a pre-assembled wire frame to the accepted conn.
// Blocks until the connection is accepted (or t fails).
func (f *peWriteFixture) WriteFrame(t *testing.T, wire []byte) {
    t.Helper()
    var conn net.Conn
    select {
    case conn = <-f.accepted:
        f.accepted <- conn // put back for subsequent calls
    case <-time.After(3 * time.Second):
        t.Fatal("peWriteFixture.WriteFrame: timed out waiting for accepted conn")
    }
    if _, err := conn.Write(wire); err != nil {
        t.Fatalf("peWriteFixture.WriteFrame: Write: %v", err)
    }
}
```

**Frame assembly in the test:**

```go
wire, err := outerassembler.Assemble(
    halfchannel.ChannelFrame{
        FrameType: frame.FrameTypeData,   // non-bootstrap → reaches OnFrameArrival
        ChanID:    1,
        ChanSeq:   1,
        Payload:   []byte{0x01},
    },
    [outerassembler.SACKBitmapSize]byte{},
    outerassembler.Envelope{},            // zero env — HMAC bypass per Q8 §8.5
)
// outerassembler.Assemble, outerassembler.SACKBitmapSize verified at 8eb54a5
// halfchannel.ChannelFrame, frame.FrameTypeData verified at 8eb54a5
if err != nil { t.Fatalf("Assemble: %v", err) }
fixture.WriteFrame(t, wire)
```

### 9.3 — Harness rule (F-SP2-003): runRouter goroutine pattern is mandatory for OnFrameArrival ACs

**Binding harness rule:** Every AC that asserts `OnFrameArrival` is reached — specifically
AC-001, AC-002, and AC-004 — MUST use the real `runRouter` goroutine pattern, not
`testenv.Restart`. The real pattern:

```go
buf := &syncBuffer{}
ctx, cancel := context.WithCancel(context.Background())
errCh := make(chan error, 1)
go func() {
    errCh <- runRouter(ctx, buf, cfg, cfgPath, nil)
}()
t.Cleanup(func() {
    cancel()
    select { case <-errCh: case <-time.After(3 * time.Second): }
})
```

`runRouter` is the code path that constructs the `FrameArrivalHandler` and calls
`connector.SetFrameCallback(frameFn)` per the Q8 ruling. `testenv.Restart` builds a bare
`upstreamdial.New` without calling `SetFrameCallback` — verified at `8eb54a5` in
`testenv.go` `Restart` implementation. This means any test using `testenv.Restart` will
have an unregistered `FrameFn` (nil); `OnFrameArrival` is never called; E-FWD-001 never
fires. Such a test would pass trivially for the wrong reason.

**Rationale for no testenv seam:** Adding a `SetFrameCallback` seam to `testenv` would
require testenv to import `routing` (or accept a `routing.FrameArrivalHandler`) — a
position-23 package importing position-17, which is lawful (23 > 17), but imports
`routing` into the test composition root unnecessarily. The real `runRouter` goroutine
pattern is already established in `router_pe_connector_test.go` (AC-001 through AC-004
in `TestRunRouter_PE_DialAndConnect_UpstreamReachable` et al., all verified at `8eb54a5`);
the new `router_pe_receive_test.go` file MUST follow the same pattern. No testenv API
change is required or permitted for this story.

### 9.4 — arqsend obligation audit and disposition (Q4 supersession accounting)

Option (b) rules `arqsend.Retransmitter` out of the E-FWD-001 integration test. The
Q4 ruling that arqsend is "test-internal only, not wired into production `runRouter`"
remains correct. What changes is the test's use of arqsend:

- **Q4's arqsend production-wiring ruling** (arqsend NOT in `runRouter`) — **RETAINED.**
  The production `runRouter` does not need a persistent `Retransmitter` instance; the
  production ARQ retransmit path is node-side. This ruling is unaffected.

- **Q4's test-internal arqsend construction** (the `Dispatch → net.Dial(ListenAddr)`
  shape) — **SUPERSEDED by Q9.** The `Dispatch` closure that dials `ListenAddr` is
  the injection path that F-SP2-001 identifies as physically disjoint from the PE
  receive goroutine. The entire arqsend frame-production role in AC-004 is replaced
  by the `peWriteFixture.WriteFrame` path.

- **S404-OBS-F "sustained send+forward" re-confirmation framing** — Q9 rules this
  is discharged through the `peWriteFixture` injection path. The "send" is
  `peWriteFixture.WriteFrame`; the "forward" attempt is `OnFrameArrival` routing
  through the split-horizon path. The S404-OBS-F obligation does NOT require
  `arqsend.Retransmitter` specifically; it requires a live frame traversing the full
  send+forward path. The `peWriteFixture` path satisfies this obligation.

- **S404-LOW-1 "live-egress re-confirmation"** — same disposition as S404-OBS-F.
  Both drift anchors are discharged by AC-004 using the `peWriteFixture` injection
  path.

**arqsend in the FCL:** `internal/arqsend` is removed from the `architecture_modules`
list of files touched by this story. The existing `arqsend` integration test
(`internal/arqsend/integration_test.go`) remains unmodified; it tests arqsend's own
`RouteFrame`-dispatch path and is unaffected by this story's injection topology change.

### 9.5 — Story propagation obligations (binding for story-writer)

The story-writer MUST propagate the following changes to `S-BL.PE-RECEIVE-LOOP.md`:

1. **Q4 dispatch closure** (v1.2) — remove the `net.Dial(cfg.ListenAddr)` dispatch shape from
   AC-004; replace with `peWriteFixture.WriteFrame` injection path.
2. **Q5 test infrastructure** (v1.2) — replace "dispatch writes to router's data-plane
   `ListenAddr`" with "upstream fixture writes assembled frame to the accepted PE
   connection via `peWriteFixture.WriteFrame`".
3. **AC-004 precondition** (v1.2) — the test uses the `runRouter` goroutine pattern with a
   `peWriteFixture` as the upstream. The `peWriteFixture` replaces both the precondition
   note about `arqsend.Retransmitter` construction and the `dispatch` closure body.
4. **FCL row 7** (v1.2 + v1.3) — `cmd/switchboard/router_pe_receive_test.go`: add `peWriteFixture` type definition AND the byte-contract pin test `TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader` per Q9 §9.1a (F-SP3-001). AC-005 flap-cycle is NOT in this file (F-SP3-002).
5. **FCL row 4** (v1.3, F-SP3-002) — `internal/upstreamdial/connector_test.go`: add AC-005 flap-cycle test per Q6 annotation.
6. **Q3 blast-radius** (v1.2 + v1.3) — add items 6 and 7 from v1.2 and item 8 from v1.3 to the story's implementation-obligation list.
7. **Q2 byte-contract** (v1.3, F-SP3-001) — specify that the receive goroutine reconstructs full frame bytes via `frame.EncodeOuterHeader`+append before invoking `FrameFn`; `FrameFn raw` is full outer-header+payload; amend any story text that implies `raw` is payload-only.
8. **Remove `internal/arqsend`** (v1.2) from the `architecture_modules` header.
9. **Q2 FrameFn return-value contract** (v1.4, F-SP4-001) — specify that the receive goroutine MUST use `_ = frameFn(hdr, raw)` (discard-and-continue); the exit-on-error form `if err := frameFn(...); err != nil { return }` is forbidden; amend any sketch code that shows the error being acted upon.
10. **Q1/Q8 SetFrameCallback ordering contract** (v1.4, F-SP4-002) — specify that `SetFrameCallback` MUST be called before `Start()`; wiring order in `runRouter` is construct → `SetFrameCallback` → `Start`; receive goroutine MAY assume non-nil `frameFn`; include nil-guard silent-discard as optional defense-in-depth; post-Start mutation is forbidden.

### 9.6 — connector_test.go frame-injection mechanism for AC-001/AC-003 (v1.5 — F-SP5-OBS-2)

**Clarification (no implementation change required):**

The AC-005 hand-rolled flap harness (`heldConn`+`Close()`) in `internal/upstreamdial/connector_test.go`
establishes the write pattern — the server-side accepted conn is held in the harness goroutine
and can call `conn.Write` to inject bytes before or after close. The AC-001/AC-003 unit tests
(`TestConnector_ReceiveLoop_DataFrameForwardedToCallback` and
`TestConnector_ReceiveLoop_PEConnectFrameDiscarded`) reuse this same in-package fixture
pattern: each test starts a local `net.Listen` listener, accepts the connector's dialed
connection, and uses `outerassembler.Assemble` + `conn.Write` to inject frames (same
`outerassembler.Assemble` usage already shown in Q9 §9.1 and the story's Design Constraints).
No new shared helper is created — the pattern is duplicated per-test or extracted into a
test-local helper at the implementer's discretion.

This clarification is needed because the v1.4 story text specified `connector_test.go` unit
tests for AC-001/AC-003 but left the write mechanism implicit ("sends a data frame on the
upstream fixture side"). The write mechanism is: `outerassembler.Assemble(cf, sackBitmap, env)`
+ server-side `conn.Write(wire)` from a goroutine that accepted the connector's dial. This is
the same pattern used by the F-SP5-001 `TestConnector_ReceiveLoop_ExitsOnReadError` test (write
malformed bytes without close) and is consistent with the existing `TestConnector_BackoffParameters`
held-conn harness (verified at `8eb54a5`).

**Cite:** Pass-2 adversarial report (F-SP2-001 CRITICAL, F-SP2-002 HIGH, F-SP2-003 MED,
F-SP2-004 MED); `cmd/switchboard/router_pe_connector_test.go` `startPEListenerFixture`
(accept-and-drain, zero Write calls, verified at `8eb54a5`); `internal/testenv/testenv.go`
`peLn` goroutine (accept-and-drain, zero Write calls, verified at `8eb54a5`);
`internal/testenv/testenv.go` `Restart` (bare `upstreamdial.New` without
`SetFrameCallback`, verified at `8eb54a5`); `outerassembler.Assemble` (verified at
`8eb54a5`); `halfchannel.ChannelFrame` (verified at `8eb54a5`).

---

## Pass-2 Adjudicated-Clean (non-findings, per adversarial pass-2 report)

The following five items were raised by the pass-2 adversary but adjudicated clean.
They are recorded here per the "adjudicated-clean: cite pass-2 report, do not re-derive"
instruction.

| Item | Adversary concern | Ruling |
|------|-------------------|--------|
| `fn ForwardFunc` no-op consistent | `SplitHorizon` may not call `fn` if no eligible path — is this a vacuous test? | Clean. `SplitHorizon.Forward` returns `ErrAllPathsSplitHorizon` BEFORE calling `fn` on the empty-eligible path (verified at `8eb54a5` in `internal/routing/split_horizon.go`). E-FWD-001 fires on the return path regardless; the no-op `fn` is never invoked. The test is not vacuous. |
| Duplicate-frame drop-cache semantics | `DropCache` may suppress the second frame if two identical frames are injected, preventing E-FWD-001 from firing twice | Clean. `arqsend.Retransmit` creates a fresh `ChanSeq` per `Retransmit` call (verified at `8eb54a5`). With the Q9 ruling replacing arqsend with `peWriteFixture`, a single injected frame is sufficient — the test asserts `"E-FWD-001"` fires once. `DropCache` has no effect on the first unique frame (fresh checksum). |
| HMAC bypass vs BC-2.02.008 preconditions | PE receive path bypasses `RouteFrame` HMAC check — does BC-2.02.008 assume admission is enforced before `OnFrameArrival`? | Clean. BC-2.02.008 carries no admission assumption (verified at `8eb54a5` in `.factory/specs/`); it postconditions on the split-horizon event itself. `OnFrameArrival` treats `frameBytes` as opaque — no HMAC field in `on_frame_arrival.go`. The bypass is acceptable for this story; a SEC follow-on revisit is noted in Q8 §8.5. |
| `peIfaceID = InterfaceID(1)` collision | Could `InterfaceID(1)` collide with a data-plane interface ID already registered by `netingress`? | Clean. The data-plane listener in `runRouter` uses `netingress.Serve`, which does NOT register `InterfaceID` values with the router (verified at `8eb54a5`); `routing.InterfaceID` values are assigned by the `FrameFn` closure, not by `netingress`. No pre-existing `InterfaceID(1)` registration exists at construction time. The PE iface ID is assigned exclusively by the wiring in Q8. |
| `routerLogger` satisfies `routing.Logger` | Does `routerLogger` (constructed in `runRouter`) implement `routing.Logger` without a shim? | Clean. `routerLogger` is constructed via `newStdLogger(w)` (verified at `8eb54a5` in `mgmt_wire.go`); `routing.Logger` is the single-method `Log(string)` interface (verified at `8eb54a5` in `internal/routing/`); `newStdLogger` produces a concrete type that satisfies `Log(string)` (verified at `8eb54a5`). No shim is needed. |

---

## Scope Boundary vs S-7.04-FU-DRAIN-WIRE

| This story (PE-RECEIVE-LOOP) | S-7.04-FU-DRAIN-WIRE |
|---|---|
| Receive goroutine per PE connection; routes incoming frames to `FrameFn` callback | Broadcasts DRAIN signal to connected nodes via SVTN channel |
| Defines `frame.FrameTypePEConnect`; flips `dialLoop` bootstrap from placeholder | Registers observers on `drainCoord` to send DRAIN frames to nodes |
| E-FWD-001 exhaustion discharge (AC-004 postcondition 1) | VP-037 `verification_lock` flip (blocked on DRAIN broadcast) |

This story provides the receive loop that makes DRAIN broadcast meaningful — a DRAIN frame sent over a PE connection must be received and forwarded. `S-7.04-FU-DRAIN-WIRE` cannot be scheduled before this story merges.

## Scope Boundary vs S-BL.RESYNC-FRAME

`S-BL.RESYNC-FRAME` owns the RESYNC control-frame exchange initiated after a node migrates to a new upstream router. This story's receive loop handles the raw frame arrival and routing; it does not implement RESYNC semantics. A RESYNC frame arriving on a PE connection will be passed to the `FrameFn` callback as a normal `FrameTypeCtl` frame (if the RESYNC frame type is `FrameTypeCtl`); further dispatch is the RESYNC story's concern.

---

## Files-Changed Forecast (Candidate FCL)

| File | Change |
|------|--------|
| `internal/frame/frame.go` (MODIFIED) | Add `FrameTypePEConnect FrameType = 0x06` (ARCH-02 §3.1 citation); update `Valid()` upper bound to `<= FrameTypePEConnect`; update doc comments: FrameType type ("Only five" → "Only six"), Valid() ("five canonical…0x06..0xFF" → "six canonical…0x07..0xFF"), ErrInvalidFrameType ("five canonical" → "six canonical or not in {0x01..0x06}") |
| `internal/frame/frame_test.go` (MODIFIED) | Add `TestFrameType_Valid_PEConnect` asserting `FrameTypePEConnect.Valid() == true`; change `just_above_max` case from `FrameType(0x06)` to `FrameType(0x07)`; change `invalids` slice `0x06` entry to `0x07`; update `"five canonical enum values"` description comment and `"Bytes not in {0x01..0x05}"` comment |
| `internal/upstreamdial/connector.go` (MODIFIED) | ~~Add `FrameFn` type + `SetFrameCallback(fn FrameFn)` to `Handle` interface~~; *(amended v1.7 — F-SP7-003: "to `Handle` interface" RETRACTED — Option A ruling (F-SP6-002) places `SetFrameCallback` on concrete `*Connector` ONLY; `Handle` interface unchanged; `fakeConnectorHandle` unaffected)* Add `FrameFn` type + `SetFrameCallback(fn FrameFn)` method on concrete `*Connector` only (NOT on `Handle` interface); add `frameFn` field to `Connector`; add receive goroutine in `dialLoop` after step-3 success with per-connection `WaitGroup` join before reconnect; flip bootstrap `ChannelFrame.FrameType` from `halfchannel.FrameTypeData` to `frame.FrameTypePEConnect` (FO-PE-LOOP-001); add direct `internal/frame` import |
| `internal/upstreamdial/connector_test.go` (MODIFIED) | Unit tests: receive goroutine exits on conn close, `FrameTypePEConnect` bootstrap frame is discarded, data frames passed to `FrameFn`; **AC-005 flap-cycle test** (per F-SP3-002 ruling): held-connection accept-and-drain + server-side `conn.Close()` to force reconnect + second listener for second connection; asserts no goroutine leak, `FrameFn` called on both connections, `Stop()` completes without hang; follows existing `heldConn`+`Close()` pattern from `TestConnector_BackoffParameters` (verified at `8eb54a5`) |
| `cmd/switchboard/mgmt_wire.go` (MODIFIED) | Construct `multipath.NewDropCache` + `routing.NewFrameArrivalHandler` after Phase b; wire `SetFrameCallback` on the connector with `FrameFn` closure routing through `arrivalHandler.OnFrameArrival`; add `internal/multipath` import |
| `cmd/switchboard/router_pe_receive_test.go` (NEW) | Integration tests: AC-001 (receive loop active after PE connection), AC-002 (E-FWD-001 fires under path exhaustion via `OnFrameArrival` with single-interface set), AC-003 (bootstrap frame discarded, not forwarded), **byte-contract pin test** `TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader` per Q9 §9.1a (two frames with differing `SrcAddr` both produce E-FWD-001; proves full-frame reconstruction). AC-005 flap-cycle test is in `connector_test.go` per F-SP3-002 ruling — `peWriteFixture` is NOT used by AC-005. |
| `.factory/specs/architecture/ARCH-02-protocol-stack.md` (MODIFIED) | §"Outer Header Format" `frame_type` table row: add `pe_connect=0x06` |
| `.factory/specs/architecture/ARCH-08-dependency-graph.md` (MODIFIED) | §6.5 update: `internal/upstreamdial` allowed imports `{halfchannel, outerassembler}` → `{frame, halfchannel, outerassembler}` |

**Estimated AC count:** 3–5 ACs. See §"Estimated AC count for story-writer" below.

---

## Summary of Rulings (Q1–Q9)

| Q | Ruling (one-line) |
|---|---|
| Q1 | Receive goroutine lives in `upstreamdial.Connector` (per-connection, spawned after step-3 success); ~~`Handle` gains `SetFrameCallback(fn FrameFn)` seam~~; *(amended v1.7 — F-SP7-003: "`Handle` gains" RETRACTED — Option A ruling (F-SP6-002) means `SetFrameCallback` exists ONLY on concrete `*Connector`; `Handle` interface unchanged)* `SetFrameCallback(fn FrameFn)` seam exists on concrete `*Connector` ONLY (NOT on `Handle` interface; `fakeConnectorHandle` unaffected per F-SP6-002 Option A); `upstreamdial` stays routing-free. (v1.0 import/signature details superseded by Q2 — see v1.1 supersession annotation.) **SetFrameCallback ordering contract (v1.4):** MUST be called before `Start()`; `frameFn` is set-once pre-launch; receive goroutine MAY assume non-nil under this ordering; nil-guard silent-discard added as defense-in-depth; post-Start mutation forbidden. |
| Q2 | Framing via new `frame.ReadOuterFrame(io.Reader) (OuterHeader, []byte, error)` at position 2 — returns payload-only (consistent with `netingress.ReadFrame` precedent); receive goroutine reconstructs full frame via `frame.EncodeOuterHeader(hdr)`+append before invoking `FrameFn`; `FrameFn raw` parameter is ALWAYS full outer-header+payload; `upstreamdial` gains direct `frame` import (ARCH-08 §6.5 amendment required); callback signature `type FrameFn func(hdr frame.OuterHeader, raw []byte) error`. (v1.2 false claim that `FrameFn raw` is payload-only retracted; see v1.3 retraction annotation.) **FrameFn return-value contract (v1.4):** non-nil return MUST NOT terminate the receive loop; discard-and-continue `_ = frameFn(hdr, raw)` is the only permitted form; exit-on-error form is forbidden. |
| Q3 | Define `frame.FrameTypePEConnect = 0x06`; update `Valid()` upper bound to `<= FrameTypePEConnect`; full blast radius (10 locations — corrected v1.6 F-SP6-004): amend `just_above_max` test (0x06→0x07), invalids slice (0x06→0x07), five doc-comment occurrences in `frame.go`/`frame_test.go`, ARCH-02 §"Outer Header Format" `frame_type` table row, `OuterHeader.FrameType` field comment (item 8, F-SP3-003), `TestFrameType_Valid` function-body "five canonical enum values" comment (item 9, F-SP6-004), AND `TestParseOuterHeader_RejectsInvalidFrameType` "canonical five enum values" + range comment (item 10, F-SP6-004 — both the range AND the count must be updated in the same edit). |
| Q4 | `arqsend.New` is NOT wired into production `runRouter` (retained). Arqsend's test-internal `Dispatch → net.Dial(ListenAddr)` injection shape is **superseded by Q9** — that shape dispatches to the data-plane socket (netingress path), not the PE receive goroutine. arqsend is NOT the frame producer in AC-004. |
| Q5 | E-FWD-001 test uses the real `runRouter` goroutine pattern (not `testenv.Restart` — F-SP2-003 harness rule); upstream fixture is `peWriteFixture` which writes assembled outer frames to the accepted PE connection; asserts key `"E-FWD-001"` in writer output (F-P11-001 lesson retained). Injection topology fully specified by Q9. |
| Q6 | Receive goroutine exits naturally when `conn.Close()` called by `dialLoop` teardown; per-connection `WaitGroup`/`done chan struct{}` MUST be joined at end of each dial iteration before reconnect (F-SP1-005 per-reconnect join requirement); AC-005 flap-cycle test lives in `connector_test.go` (per F-SP3-002 — `peWriteFixture` is NOT used by AC-005; flap harness follows `heldConn`+`Close()` pattern, verified in `TestConnector_BackoffParameters` at `8eb54a5`). |
| Q7 | BC-2.06.003 PC-1 "failed" status is from `metrics.PathEntryFromSnapshot` (path liveness), NOT from E-FWD-001 (split-horizon); BC ambiguity flagged for PO confirmation; operative assertion is `"E-FWD-001"` key (Option A). |
| Q8 | PE receive `FrameFn` MUST route through `routing.NewFrameArrivalHandler`+`OnFrameArrival` (not `RouteFrame`) to make E-FWD-001 reachable; `runRouter` constructs `multipath.NewDropCache` + `routing.NewFrameArrivalHandler` after Phase b; `netingress.Serve` path unchanged; `cmd/switchboard` gains `internal/multipath` import. |
| Q9 | **Injection topology ruling** (supersedes Q4 dispatch + Q5 injection shape): option (b) — upstream PE fixture (`peWriteFixture`, test-local in `router_pe_receive_test.go`) writes assembled outer frame directly to the accepted PE connection; `arqsend.Retransmitter` is NOT used as frame producer in AC-004; harness rule: every AC asserting `OnFrameArrival` MUST use the real `runRouter` goroutine pattern (not `testenv.Restart`); S404-OBS-F and S404-LOW-1 discharged via `peWriteFixture` injection path; Q4 production-wiring ruling (arqsend not in `runRouter`) retained. **AC-005 flap-cycle is NOT in `peWriteFixture` scope** (F-SP3-002 ruling; see FCL row 4 and Q6 annotation). Byte-contract pin test §9.1a added: two frames differing in `OuterHeader.SrcAddr` both produce E-FWD-001, proving full-frame reconstruction is wired (F-SP3-001). |

---

## Estimated AC Count for Story-Writer

Based on the rulings above and the 5-point estimated scope in the stub story:

| AC | Trace | Description |
|----|-------|-------------|
| AC-001 | BC-2.09.001 PC-2/PC-3 | Receive loop is active after PE connection established; incoming frame from upstream is passed to `FrameArrivalHandler.OnFrameArrival` callback |
| AC-002 | BC-2.02.008 PC-3/EC-003, S404-OBS-F | E-FWD-001 fires under sustained path-exhaustion load via live PE connection + arqsend retransmit |
| AC-003 | FO-PE-LOOP-001 | Bootstrap frame with `FrameTypePEConnect` is discarded at receiver; NOT forwarded through routing callback |
| AC-004 | BC-2.06.003 PC-1 (Option A) | Live send+forward path traversal is exercised; BC-2.06.003 trace confirmed or clarified by PO per Q7 flag |
| AC-005 | S404-LOW-1 | Live egress re-confirmation: full send→forward path is demonstrated end-to-end |

Estimated: **5 ACs** (may collapse AC-004/AC-005 into AC-002 at elaboration).

---

## Appendix A: Backtick-Symbol Sweep

All `CamelCase/pkg.Symbol` tokens used in this note. Sweep performed against the
tree at `8eb54a5` using `grep` on the verified file paths.

| Symbol | File verified | Status |
|--------|--------------|--------|
| `frame.FrameTypeData` | `internal/frame/frame.go` | VERIFIED — `FrameTypeData FrameType = 0x01` |
| `frame.FrameTypeEmptyTick` | `internal/frame/frame.go` | VERIFIED — `FrameTypeEmptyTick FrameType = 0x02` |
| `frame.FrameTypeCtl` | `internal/frame/frame.go` | VERIFIED — `FrameTypeCtl FrameType = 0x03` |
| `frame.FrameTypeArq` | `internal/frame/frame.go` | VERIFIED — `FrameTypeArq FrameType = 0x04` |
| `frame.FrameTypeFec` | `internal/frame/frame.go` | VERIFIED — `FrameTypeFec FrameType = 0x05` |
| `frame.FrameTypePEConnect` | N/A | NEW CONSTANT — value `0x06`; to be defined by this story in `internal/frame/frame.go` |
| `frame.FrameType.Valid()` | `internal/frame/frame.go` | VERIFIED — `func (f FrameType) Valid() bool { return f >= FrameTypeData && f <= FrameTypeFec }` |
| `frame.ParseOuterHeader` | `internal/frame/frame.go` | VERIFIED — `func ParseOuterHeader(b []byte) (OuterHeader, error)` |
| `frame.OuterHeader` | `internal/frame/frame.go` | VERIFIED — `type OuterHeader struct { ... FrameType FrameType; PayloadLen uint16; ... }` |
| `frame.OuterHeaderSize` | `internal/frame/frame.go` | VERIFIED — `const OuterHeaderSize = 44` |
| `frame.ReadOuterFrame` | N/A | NEW FUNCTION — to be added to `internal/frame/frame.go` by this story |
| `netingress.ReadFrame` | `internal/netingress/netingress.go` | VERIFIED — `func ReadFrame(r io.Reader) (frame.OuterHeader, []byte, error)` |
| `netingress.RouteFn` | `internal/netingress/netingress.go` | VERIFIED — `type RouteFn func(hdr frame.OuterHeader, payload []byte) error` |
| `netingress.ServeConn` | `internal/netingress/netingress.go` | VERIFIED — `func ServeConn(ctx context.Context, conn net.Conn, route RouteFn, logger Logger) error` |
| `netingress.Serve` | `internal/netingress/netingress.go` | VERIFIED — `func Serve(ctx context.Context, ln net.Listener, route RouteFn, logger Logger) error` |
| `halfchannel.FrameTypeData` | `internal/halfchannel/halfchannel.go` | VERIFIED — alias of `frame.FrameTypeData` |
| `halfchannel.FrameTypeEmptyTick` | `internal/halfchannel/halfchannel.go` | VERIFIED — alias of `frame.FrameTypeEmptyTick` |
| `halfchannel.ChannelFrame` | `internal/halfchannel/halfchannel.go` | VERIFIED — `type ChannelFrame struct { ChanID uint32; ChanSeq uint32; FrameType frame.FrameType; Flags byte; Payload []byte }` |
| `outerassembler.Assemble` | `internal/outerassembler/assemble.go` | VERIFIED — `func Assemble(cf halfchannel.ChannelFrame, sackBitmap [SACKBitmapSize]byte, env Envelope) ([]byte, error)` |
| `outerassembler.Envelope` | `internal/outerassembler/assemble.go` | VERIFIED — `type Envelope struct { SVTNID [16]byte; SrcAddr [8]byte; DstAddr [8]byte; FrameAuthKey [hmac.KeySize]byte }` |
| `outerassembler.SACKBitmapSize` | `internal/outerassembler/channelheader.go` | VERIFIED — `const SACKBitmapSize = 8` |
| `routing.FrameArrivalHandler` | `internal/routing/on_frame_arrival.go` | VERIFIED — `type FrameArrivalHandler struct { ... }` |
| `routing.NewFrameArrivalHandler` | `internal/routing/on_frame_arrival.go` | VERIFIED — `func NewFrameArrivalHandler(dc *multipath.DropCache) *FrameArrivalHandler` |
| `routing.WithFrameArrivalLogger` | `internal/routing/on_frame_arrival.go` | VERIFIED — `func WithFrameArrivalLogger(l Logger) func(*FrameArrivalHandler)` |
| `routing.FrameArrivalHandler.OnFrameArrival` | `internal/routing/on_frame_arrival.go` | VERIFIED — `func (h *FrameArrivalHandler) OnFrameArrival(frameBytes []byte, arrivalIface InterfaceID, interfaceSet []InterfaceID, fn ForwardFunc) error` |
| `routing.ErrAllPathsSplitHorizon` | `internal/routing/split_horizon.go` | VERIFIED — `var ErrAllPathsSplitHorizon = errors.New("routing: split-horizon: no eligible output interface (E-FWD-001)")` |
| `routing.InterfaceID` | `internal/routing/split_horizon.go` | VERIFIED — `type InterfaceID uint64` |
| `routing.ForwardFunc` | `internal/routing/split_horizon.go` | VERIFIED — `type ForwardFunc func(iface InterfaceID, frameBytes []byte) error` |
| `routing.RouteFrame` | `internal/routing/routing.go` | VERIFIED — `func RouteFrame(hdr frame.OuterHeader, payload []byte, r *Router) error` |
| `arqsend.New` | `internal/arqsend/arqsend.go` | VERIFIED — `func New(a *arq.ARQ, env outerassembler.Envelope, opts ...Option) *Retransmitter` |
| `arqsend.Retransmitter` | `internal/arqsend/arqsend.go` | VERIFIED — `type Retransmitter struct { ... }` |
| `arqsend.Retransmitter.Retransmit` | `internal/arqsend/arqsend.go` | VERIFIED — `func (r *Retransmitter) Retransmit(oldSeq, newSeq uint32, now time.Time, dispatch Dispatch) error` |
| `arqsend.Dispatch` | `internal/arqsend/arqsend.go` | VERIFIED — `type Dispatch func(wire []byte) error` |
| `arqsend.WithChanID` | `internal/arqsend/arqsend.go` | VERIFIED — `func WithChanID(chanID uint32) Option` |
| `arqsend.ErrSequenceNotInFlight` | `internal/arqsend/arqsend.go` | VERIFIED — `var ErrSequenceNotInFlight = errors.New("arqsend: oldSeq not in flight")` |
| `arq.New` | `internal/arq/arq.go` | VERIFIED — `func New(cfg Config) *ARQ` |
| `arq.ARQ` | `internal/arq/arq.go` | VERIFIED — `type ARQ struct { ... }` |
| `upstreamdial.New` | `internal/upstreamdial/connector.go` | VERIFIED — `func New(w io.Writer, env outerassembler.Envelope, keepaliveInterval time.Duration, initialAddrs []string) *Connector` |
| `upstreamdial.Connector` | `internal/upstreamdial/connector.go` | VERIFIED — `type Connector struct { ... }` |
| `upstreamdial.Handle` | `internal/upstreamdial/connector.go` | VERIFIED — `type Handle interface { ReloadAddrs(addrs []string); Mode() ConnMode; Stop() }` |
| `upstreamdial.ConnMode` | `internal/upstreamdial/connector.go` | VERIFIED — `type ConnMode int32` |
| `upstreamdial.ModeE` | `internal/upstreamdial/connector.go` | VERIFIED — `ModeE ConnMode = 0` |
| `upstreamdial.ModePE` | `internal/upstreamdial/connector.go` | VERIFIED — `ModePE ConnMode = 1` |
| `upstreamdial.BackoffBase` | `internal/upstreamdial/connector.go` | VERIFIED — `const BackoffBase = 500 * time.Millisecond` |
| `testenv.New` | `internal/testenv/testenv.go` | VERIFIED — `func New(ctx context.Context, t testing.TB) *Env` |
| `testenv.NewWithRouters` | `internal/testenv/testenv.go` | VERIFIED — `func NewWithRouters(ctx context.Context, t testing.TB, n int) *Env` |
| `testenv.RouterHandle` | `internal/testenv/testenv.go` | VERIFIED — `type RouterHandle struct { ... }` |
| `testenv.RouterMode` | `internal/testenv/testenv.go` | VERIFIED — `type RouterMode int` |
| `testenv.ModeE` | `internal/testenv/testenv.go` | VERIFIED — `ModeE RouterMode = iota` |
| `testenv.ModePE` | `internal/testenv/testenv.go` | VERIFIED — `ModePE` (second iota value) |
| `testenv.RouterHandle.SetConnector` | `internal/testenv/testenv.go` | VERIFIED — `func (r *RouterHandle) SetConnector(h upstreamdial.Handle)` |
| `testenv.RouterHandle.Mode` | `internal/testenv/testenv.go` | VERIFIED — delegates to `connector.Mode()` when connector non-nil |
| `testenv.Env.StartRouter` | `internal/testenv/testenv.go` | VERIFIED — `func (e *Env) StartRouter(t testing.TB, cfg RouterConfig) *RouterHandle` |
| `testenv.Env.PERouterAddr` | `internal/testenv/testenv.go` | VERIFIED — `func (e *Env) PERouterAddr(t testing.TB) string` |
| `testenv.Env.SendDrainSignal` | `internal/testenv/testenv.go` | VERIFIED — `func (e *Env) SendDrainSignal(t testing.TB, idx int)` |
| `metrics.PathEntryFromSnapshot` | `internal/metrics/handlers.go` | VERIFIED — `func PathEntryFromSnapshot(pathID string, snap paths.PathSnapshot) PathEntry` (produces `status: "failed"` when `snap.Failed`) |
| `paths.PathSnapshot.Failed` | `internal/paths/paths.go` | VERIFIED — `type PathSnapshot struct { ... Failed bool ... }` |
| `multipath.NewDropCache` | `internal/multipath/multipath.go` | VERIFIED — `func NewDropCache(capacity int) *DropCache` (panics if capacity < 1) |
| `multipath.DropCache` | `internal/multipath/multipath.go` | VERIFIED — `type DropCache struct { ... }` (zero value not usable; construct via NewDropCache) |
| `multipath.DefaultDropCacheSize` | `internal/multipath/multipath.go` | VERIFIED — `const DefaultDropCacheSize = 10_000` |
| `routing.SVTNRoute` | `internal/routing/routing.go` | VERIFIED — `func SVTNRoute(hdr frame.OuterHeader, payload []byte, r *Router) error` — called by RouteFrame; does NOT call OnFrameArrival |
| `routing.ErrDropCacheHit` | `internal/routing/on_frame_arrival.go` | VERIFIED — `var ErrDropCacheHit = errors.New("routing: drop cache hit — frame suppressed as loop duplicate (BC-2.02.009)")` |

### Appendix A Delta (v1.2 additions — Q9 fixture symbols)

New symbols introduced by Q9 (test-local; NEW definitions in `cmd/switchboard/router_pe_receive_test.go`):

| Symbol | File | Status |
|--------|------|--------|
| `peWriteFixture` | `cmd/switchboard/router_pe_receive_test.go` | NEW TYPE — test-local upstream fixture struct with `addr string`, `accepted chan net.Conn`, `ln net.Listener` |
| `startPEWriteFixture` | `cmd/switchboard/router_pe_receive_test.go` | NEW FUNCTION — `func startPEWriteFixture(t *testing.T) *peWriteFixture`; starts loopback TCP listener, accepts one conn (draining read loop on connector's bootstrap/keepalive writes), exposes it via `accepted` channel |
| `peWriteFixture.WriteFrame` | `cmd/switchboard/router_pe_receive_test.go` | NEW METHOD — `func (f *peWriteFixture) WriteFrame(t *testing.T, wire []byte)` — writes pre-assembled outer frame to the accepted PE connection |

Previously verified symbols reused by Q9 (no re-verification required):

| Symbol | Prior verification | Q9 usage |
|--------|--------------------|----------|
| `outerassembler.Assemble` | Appendix A v1.0 | Used in test to assemble `FrameTypeData` frame for fixture injection |
| `outerassembler.SACKBitmapSize` | Appendix A v1.0 | Used in zero-value SACK bitmap for test frame |
| `outerassembler.Envelope` | Appendix A v1.0 | Zero envelope (HMAC bypass per Q8 §8.5) |
| `halfchannel.ChannelFrame` | Appendix A v1.0 | Test frame struct with `FrameType: frame.FrameTypeData` |
| `frame.FrameTypeData` | Appendix A v1.0 | Non-bootstrap type to pass PE-CONNECT discard check |

### Appendix A Delta (v1.4 additions — F-SP4-001 + F-SP4-002)

No new symbols are introduced by the v1.4 rulings. All symbols cited in the
new contracts are already verified in the main table or prior deltas:

| Symbol | Prior verification | v1.4 usage |
|--------|--------------------|------------|
| `netingress.RouteFn` | Appendix A v1.0 | F-SP4-001: normative precedent for discard-and-continue semantics ("NOT a signal to close the connection") |
| `netingress.ServeConn` | Appendix A v1.0 | F-SP4-001: `continue` drop-and-continue pattern; double-count-avoidance rationale |
| `routing.ErrAllPathsSplitHorizon` | Appendix A v1.0 | F-SP4-001: one of the two non-fatal non-nil `FrameFn` return paths |
| `routing.ErrDropCacheHit` | Appendix A v1.0 | F-SP4-001: second non-fatal non-nil `FrameFn` return path |
| `upstreamdial.Connector.Start` | Appendix A v1.0 (via `upstreamdial.New`) | F-SP4-002: ordering constraint — `SetFrameCallback` before `Start()` |

The `.golangci.yml` errcheck configuration is a build-configuration file, not
a Go symbol; no Appendix A row is required.

### Appendix A Delta (v1.3 additions — Q2 reconstruction + F-SP3-003)

New or clarified symbols introduced by v1.3 rulings:

| Symbol | File verified | Status |
|--------|--------------|--------|
| `frame.EncodeOuterHeader` | `internal/frame/frame.go` | VERIFIED — `func EncodeOuterHeader(h OuterHeader) [OuterHeaderSize]byte` (line ~84 at `8eb54a5`; serialises h into exactly 44 bytes per ARCH-02 big-endian wire layout). Used in Q2 receive-goroutine reconstruction: `ehdr := frame.EncodeOuterHeader(hdr)` then `append(ehdr[:], payload...)`. Already in the position-2 package; no new import required beyond Q2's existing `frame` import ruling. |

Previously verified symbols reused by v1.3 (no re-verification required):

| Symbol | Prior verification | v1.3 usage |
|--------|--------------------|------------|
| `outerassembler.Envelope.SrcAddr` | `outerassembler.Envelope` verified Appendix A v1.0 | Q9 §9.1a pin test: two frames differing in `SrcAddr` bytes; `SrcAddr [8]byte` field in `Envelope` feeds into assembled outer header |
| `frame.OuterHeaderSize` | Appendix A v1.0 | Q2 reconstruction: `ehdr [OuterHeaderSize]byte` allocation size |

### Appendix A Delta (v1.5 additions — F-SP5-001 + F-SP5-OBS-1 + F-SP5-OBS-2)

No new symbols are introduced by the v1.5 rulings. All symbols cited in the new contracts
are already verified in the main table or prior deltas:

| Symbol | Prior verification | v1.5 usage |
|--------|--------------------|------------|
| `frame.ReadOuterFrame` | Appendix A v1.0 (marked NEW — defined by this story) | F-SP5-001: READ-error exit contract — any non-nil return MUST trigger `return` |
| `netingress.ServeConn` | Appendix A v1.0 | F-SP5-001: normative precedent for read-error-exit vs callback-continue per-site disposition (lines 134–143 / 145–147 at `8eb54a5`) |
| `frame.ErrInvalidFrameType` | `internal/frame/frame.go` — `var ErrInvalidFrameType = errors.New("frame: invalid frame_type")` (verified at `8eb54a5`, line 49) | F-SP5-001: canonical example of abnormal read error (malformed frame type → reconnect); also example target byte for `TestConnector_ReceiveLoop_ExitsOnReadError` |
| `outerassembler.Assemble` | Appendix A v1.0 | F-SP5-OBS-2: write-side frame assembly for AC-001/AC-003 connector_test.go unit fixtures |
| `halfchannel.ChannelFrame` | Appendix A v1.0 | F-SP5-OBS-2: same |

`frame.ErrInvalidFrameType` is returned by `frame.ParseOuterHeader` (and thus by
`frame.ReadOuterFrame`) when `b[1]` is not a canonical `FrameType` value. Verified at
`8eb54a5` in `internal/frame/frame.go` line 49: `var ErrInvalidFrameType = errors.New("frame: invalid frame_type")`.

### Appendix A Delta (v1.6 additions — F-SP6-001 through F-SP6-004)

No new symbols are introduced by the v1.6 rulings. All symbols cited in the amendments
are already verified in the main table or prior deltas:

| Symbol | Prior verification | v1.6 usage |
|--------|--------------------|------------|
| `upstreamdial.Handle` | Appendix A v1.0 (`type Handle interface { ReloadAddrs; Mode; Stop }`) | F-SP6-002: confirmed interface does NOT gain `SetFrameCallback` (Option A ruling) |
| `upstreamdial.Connector` | Appendix A v1.0 | F-SP6-002: `SetFrameCallback` called on concrete `*Connector` between `New()` and `Start()` |
| `net.Conn.Close()` | stdlib — idempotent/safe for multiple calls | F-SP6-001: receive goroutine calls `_ = conn.Close()` on read-error exit; double-close is safe |
| `maintainConn` | `internal/upstreamdial/connector.go` (verified at `8eb54a5`, lines ~399–430) | F-SP6-001: write-only loop; returns only on write failure / SetWriteDeadline failure / stopAddr close; never reads conn |
| `testenv.RouterHandle.SetConnector` | Appendix A v1.0 | F-SP6-002: `SetConnector` call sites (:493, :503) use `fakeConnectorHandle` which does NOT need `SetFrameCallback`; Option A preserves compile integrity |

### Appendix A Delta (v1.7 additions — F-SP7-001 through F-SP7-005)

No new symbols are introduced by the v1.7 rulings. All code citations are already verified
in the main table or prior deltas. Ground-truth verifications performed for v1.7:

| Symbol / location | Verified at | v1.7 usage |
|---|---|---|
| `mgmt_wire.go` :548 startup writer block `if len(upstreamRouters) == 0 / else mode=PE` | `cmd/switchboard/mgmt_wire.go` (verified at `8eb54a5` — lines 545–549) | F-SP7-001: `"mode=PE"` gated on `len(upstreamRouters)>0` only; no connection dependency |
| `mgmt_wire.go` :587 SIGHUP re-emit `mode=PE` | `cmd/switchboard/mgmt_wire.go` (verified at `8eb54a5` — line 587 in SIGHUP branch) | F-SP7-001: second emission also gated on `len(upstreamRouters)>0` after reload |
| `connector.go` :350 bootstrap Write | `internal/upstreamdial/connector.go` (verified at `8eb54a5` — `n, wErr := conn.Write(wire)` in step-3 block) | F-SP7-002: bootstrap Write precedes Add(1); TCP accept on fixture side fires at DialContext success, before Write |
| `connector.go` :365 `connectedCount.Add(1)` | `internal/upstreamdial/connector.go` (verified at `8eb54a5` — `c.connectedCount.Add(1)`) | F-SP7-002: Add(1) strictly after bootstrap Write; accepted fires before both |
| `connector.go` :390 `"mode=E"` log-only emission | `internal/upstreamdial/connector.go` (verified at `8eb54a5` — `c.logf("mode=E (no upstream_routers configured)\n")` in transition-ownership block) | F-SP7-001 / Pass-7 adjudicated-clean: confirms connector.go emits only `mode=E`, not `mode=PE`; no confusion with mgmt_wire.go emissions |
| `TestRunRouter_PE_UnreachableUpstream_PartialPE` | `cmd/switchboard/router_pe_connector_test.go` (verified at `8eb54a5`) | F-SP7-001 existing-test precedent: proves `"mode=PE"` fires even when upstream is unreachable |

---

## Pass-3 Adjudicated-Clean (non-findings, per adversarial pass-3 report)

The following items were raised by the pass-3 adversary but adjudicated clean.
Recorded per "adjudicated-clean: cite pass, do not re-derive" instruction.

| Item | Adversary concern | Ruling |
|------|-------------------|--------|
| Fixture drain-vs-WriteFrame concurrency | `peWriteFixture` uses a perpetual drain goroutine (`c.Read(buf)` in a loop) reading from the same `net.Conn` that `WriteFrame` writes to — is there a concurrency hazard? | Clean. The fixture drain reads from the connector's WRITE direction (bootstrap + keepalive bytes the connector sends to the fixture's server side). `WriteFrame` writes to the connector's READ direction (upstream-originated frames the connector's receive goroutine reads). TCP connections are full-duplex: read and write paths are independent. Single reader (drain goroutine) and single writer (WriteFrame) per direction — no races. |
| `Assemble`→`ReadOuterFrame` framing self-delimited | Does `frame.ReadOuterFrame` correctly delimit messages encoded by `outerassembler.Assemble`? | Clean. `outerassembler.Assemble` (verified at `8eb54a5`) encodes `hdr.PayloadLen` in the outer header per ARCH-02 big-endian layout; `ReadOuterFrame` (mirroring `netingress.ReadFrame`) reads exactly `OuterHeaderSize` bytes then `hdr.PayloadLen` bytes. Self-delimiting by construction. |
| Zero-Envelope passes all guards to split-horizon | Does a zero `outerassembler.Envelope{}` (zero SVTNID, zero SrcAddr, zero FrameAuthKey) reach `OnFrameArrival` without being rejected by any guard in the PE receive path? | Clean. The PE receive `FrameFn` closure (per Q8 ruling) does NOT call `RouteFrame` or any HMAC-check function — it goes directly to `arrivalHandler.OnFrameArrival`. `OnFrameArrival` treats `frameBytes` as opaque (BC-2.01.005 / VP-015) — no field inspection. A zero envelope produces a well-formed assembled frame (non-zero `PayloadLen` from the `Payload: []byte{0x01}` field); `ParseOuterHeader` accepts it (version byte `0x01`, valid `FrameTypeData`, valid `PayloadLen`). No guard rejects it before `OnFrameArrival`. |
| Bootstrap discrimination direction | The fixture writes `FrameTypeData` (0x01); the connector writes `FrameTypePEConnect` (0x06). Is the discrimination direction correct (fixture-to-connector is DATA, connector-to-upstream is PE-CONNECT)? | Clean. The PE-CONNECT bootstrap frame is sent UPSTREAM by the connector (in `dialLoop`) to announce itself to the upstream router. Frames arriving ON the PE connection from upstream are data/ctl/arq/fec — the test fixture correctly uses `FrameTypeData` (0x01) so the receive goroutine's `FrameTypePEConnect` discard check does NOT fire and the frame reaches `OnFrameArrival`. |
| Keepalive interference with E-FWD-001 | Could keepalive frames (FrameTypeEmptyTick, sent by `maintainConn`) arrive on the accepted conn and fire E-FWD-001 before the test-injected DATA frame, causing timing sensitivity? | Clean. `maintainConn` sends keepalives on the DIALED conn (connector → upstream); the fixture's accepted conn is the server side of that connection — it receives keepalive bytes in the drain goroutine's Read loop. These bytes are NOT relayed back to the connector's receive goroutine (the drain only reads and discards). The connector's receive goroutine reads from the DIALED conn (data flowing upstream → connector). No keepalive interference. |

---

## Pass-4 Adjudicated-Clean (non-findings, per adversarial pass-4 report)

The following items were raised by the pass-4 adversary but adjudicated clean.
Recorded per "adjudicated-clean: cite pass-4 report, do not re-derive" instruction.

| Item | Adversary concern | Ruling |
|------|-------------------|--------|
| Byte-contract round-trip exact | Is the `EncodeOuterHeader`/`ParseOuterHeader` pair lossless over all 44 bytes, including any fields not represented in `OuterHeader`? | Clean (per pass-4 report). The pair is a total lossless round-trip over all 44 wire bytes: `crc32(EncodeOuterHeader(ParseOuterHeader(b))) == crc32(b)` for all valid 44-byte inputs. No outer-header checksum field; no reserved bits that are masked on parse. Reconstruction path `EncodeOuterHeader(hdr)` + append is exact. |
| Pin test valid and distinguishing | Does `TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader` actually distinguish the full-frame vs payload-only wiring? Does `Envelope.SrcAddr` flow to outer-header bytes `[20:28]`? | Clean (per pass-4 report). `Envelope.SrcAddr` feeds outer-header bytes `[20:28]` via `assemble.go` (verified at `8eb54a5`). `SrcAddr` is absent from the channel header. The drop-cache key `crc32.ChecksumIEEE(frameBytes)` is computed over the full frame including outer-header bytes; with identical payloads and differing `SrcAddr`, payload-only wiring produces the same checksum (false dup suppression), while full-frame wiring produces different checksums (both frames reach `OnFrameArrival` independently). The test is valid and strongly distinguishing. |
| E-FWD-001 emission rate-limiting | Is the E-FWD-001 emission rate-limited in any tier-1 or tier-2 sampler that would prevent the second emission from appearing in the writer output? | Clean (per pass-4 report). The tier-1/tier-2 sampler gates only EC-005 cache-hit log emissions (per `on_frame_arrival.go` at `8eb54a5`). E-FWD-001 is emitted unconditionally on every split-horizon exhaustion event — it is not subject to any rate limiter or sampler. Both frame A and frame B independently fire E-FWD-001 without suppression. |
| Q3 8-location blast radius completeness | Did the fresh grep find any ninth blast-radius location beyond the eight enumerated in Q3? | Clean (per pass-4 report). Fresh grep found no ninth location. All eight were correctly enumerated at the time of pass-4. **Note (v1.6 — F-SP6-004):** pass-6 adversarial review subsequently found two additional stale-comment locations (items 9 and 10 — `frame_test.go` lines ~501 and ~540 count-claim comments) that the pass-4 grep did not surface because the search patterns used did not match those specific comment texts. The Q3 blast-radius table is corrected to 10 locations in v1.6; this pass-4 adjudication row is preserved as an accurate record of the state at pass-4. |
| `TestConnector_BackoffParameters` heldConn precedent | Is the `heldConn`+`Close()` flap pattern in `TestConnector_BackoffParameters` a real, verified shape at `8eb54a5`? | Clean (per pass-4 report). Confirmed real at `8eb54a5`. `TestConnector_BackoffParameters` uses a `heldConn` that accepts a connection and holds it, then calls `conn.Close()` on the server side to trigger a reconnect. This is the exact template for the AC-005 flap-cycle test (Q6 annotation). |

---

## Pass-5 Adjudicated-Clean (non-findings, per adversarial pass-5 report)

The following items were raised by the pass-5 adversary but adjudicated clean.
Recorded per "adjudicated-clean: cite pass-5 report, do not re-derive" instruction.

| Item | Adversary concern | Ruling |
|------|-------------------|--------|
| READ-error exit vs ctx-cancel ordering | When `ctx` is cancelled, `conn.Close()` is called by the `makeAddrContext` bridge goroutine. Could the receive goroutine's `frame.ReadOuterFrame` return a net error (not `io.EOF`) that the implementer might mistake for an abnormal error and log? | Clean. The F-SP5-001 logging disposition specifies that any read error when `ctx.Err() != nil` is treated as a clean exit (silent, no log). The implementer checks `ctx.Err()` after any non-nil `ReadOuterFrame` error to distinguish the two paths. This mirrors the `netingress.ServeConn` pattern (verified at `8eb54a5`, lines 138–142: `if ctx.Err() != nil { return nil }`). The ruling is self-consistent with the existing teardown model. |
| OBS-1: uint16 bound sufficient for DoS protection | Is `PayloadLen uint16` truly an adequate mitigation against CWE-400 on a PE-to-PE connection, given that PE routers can be compromised? | Clean per OBS-1 ruling. The threat model for the PE receive path is a CONFIGURED upstream router, not an unauthenticated attacker. A compromised upstream can exhaust memory at most at ~64 KB/frame × goroutine concurrency; there is no amplification. This is a qualitatively different risk than the data-plane netingress path (unauthenticated clients). The accept-the-divergence ruling in OBS-1 stands. If the threat model changes (e.g. PE routers become user-configurable without operator vetting), a future pass can add `LimitReader`. |
| OBS-2: per-test duplication vs shared helper in connector_test.go | Is duplicating the accept-and-write fixture in each connector_test.go unit test a maintenance risk? | Clean per OBS-2 ruling. The pattern is simple enough (≤20 lines per test) that per-test duplication does not create significant maintenance burden. The implementer may extract a test-local helper function within `connector_test.go` if it improves readability, but this is an implementer choice, not a spec obligation. No exported or cross-package helper is created. |

---

## F-SP7-003 Sweep Transcript (v1.7 — Option A compliance sweep)

F-SP7-003 required a full sweep of the placement note for residual "to Handle interface" /
"Handle gains" / "Handle.*SetFrame" / "Add FrameFn type.*Handle" text that contradicts the
binding F-SP6-002 Option A ruling.

**Initial sweep (4-pattern set):** Executed at v1.7 publication with patterns `"Handle gains"`,
`"to Handle"`, `"Handle.*SetFrame"`, `"Add FrameFn type.*Handle"`. Found and swept the FCL
connector.go row and Q1 Summary-of-Rulings row. **The initial sweep was insufficient**: two
binding-claim occurrences in the Q1 v1.0 body text were missed — the `"Handle gains"` pattern
matched the Q1-body occurrence (line :76) but the count was recorded as 2, not 3, and the body
hit was not actioned; the `"interface gains"` phrasing at line :90 used different vocabulary
not covered by any of the four initial patterns.

**Expanded sweep (8-pattern set):** Executed on orchestrator disk-audit post-publication.
Found and swept the two missed Q1-body residuals. Both struck and annotated in the same v1.7
revision (no version bump — completion of the v1.7 F-SP7-003 sweep).

**Grep patterns used and hit counts (initial vs expanded, with dispositions):**

| Pattern | Initial sweep hits | Expanded sweep hits | Disposition |
|---|---|---|---|
| `"Handle gains"` | **2** (Changelog v1.0 row; Q1 Summary-of-Rulings row — Q1 body :76 hit **initially missed, swept on transcript audit**) | 3 (+ Q1 body option-(a) :76) | Q1 body :76: struck + annotated v1.7 (initially missed residual). Q1 Summary-of-Rulings row: struck + corrected. Changelog v1.0 row: history-preserved verbatim (pre-Option-A; labelled superseded). |
| `"gains a method"` | **not in initial pattern set** | 1 (Q1 body :76: `Handle gains a method SetFrameCallback(fn func([]byte) error)`) | Struck + annotated v1.7 (same line as "Handle gains" :76 residual). |
| `"gains a setter"` | **not in initial pattern set** | 1 (Q1 body :90: `The Handle interface gains a setter`) | Struck + annotated v1.7 (F-SP7-003 residual — "interface gains" phrasing not covered by any initial pattern). |
| `"interface gains"` | **not in initial pattern set** | 1 (Q1 body :90 — same occurrence as "gains a setter") | Struck + annotated v1.7 above. |
| `"to Handle"` | 1 (FCL connector.go row) | 1 | FCL connector.go row: struck + corrected. |
| `"Handle.*SetFrame"` | 0 (no binding-claim hits beyond already-swept rows) | 0 | None. |
| `"Add FrameFn type.*Handle"` | 1 (FCL connector.go row — same as "to Handle" hit) | 1 | FCL connector.go row: amended. |
| `"Handle interface"` (broader catch-all) | not run in initial sweep | Multiple; filtered | Filtered: (a) Q1 body :90 "The `Handle` interface gains a setter" — struck v1.7 (captured above). (b) Appendix A delta v1.6: correctly states interface unchanged (Option A). (c) Pass-6 adjudicated-clean rows: correctly describe Option A. No additional stale Option-B claims. |

**Post-sweep conclusion (final — after expanded pattern audit):** The initial 4-pattern sweep
missed two Q1-body binding-claim occurrences: line :76 (`Handle gains a method`, present in the
`"Handle gains"` pattern but not actioned in the initial pass) and line :90 (`interface gains a
setter`, vocabulary not covered by any initial pattern). Both were identified by the orchestrator
disk-audit and swept in the same v1.7 revision. After the expanded sweep, all stale
"Handle gains" / "gains a method" / "gains a setter" / "interface gains" / "to Handle interface"
binding-claim text has been struck and annotated. The only remaining occurrences are
(a) history-preserved v1.0 changelog row (labelled superseded), (b) v1.1 supersession
annotations explaining the evolution from Option-B intent to Option-A ruling,
(c) this sweep-transcript itself (meta-references, not binding claims), and
(d) Option-A-consistent text in adjudicated-clean sections and Appendix A.
No residual Option-B violations remain.

**v1.15 Addendum — F-SP7-003 Sweep Re-Certification (F-SP19-001):**

The original F-SP7-003 sweep was conducted with four single-line grep patterns
(`"Handle gains"`, `"to Handle"`, `"Handle.*SetFrame"`, `"Add FrameFn type.*Handle"`) and
the expanded 8-pattern set applied at v1.7 post-publication audit. Both sweep sets share a
common limitation: they cannot match tokens that span a line break. The F-SP19-001
residual at lines :110-111 (v1.14 numbering) contains the token pair `"Handle\`" on
one line and `"gains \`SetFrameCallback"` on the next — no single-line grep pattern
can match both tokens simultaneously.

**NEW canonical multi-line-tolerant pattern:** `tr '\n' ' ' < FILE | grep -o "Handle. gains .SetFrameCallback"`

This pattern joins all lines into a single stream before matching and reliably catches
the cross-line token pair regardless of where the line break falls.

**Post-fix execution (run at v1.15):**

```
tr '\n' ' ' < S-BL.PE-RECEIVE-LOOP-placement-note.md | grep -o "Handle\` gains \`SetFrameCallback"
```

**Hit count:** 7 hits (post-fix). Note: the v1.15 addendum text itself introduces
meta-reference occurrences of the pattern (in the canonical-pattern line, the changelog
row, the Pass-19 adjudicated table, and this addendum). The strikethrough on the
:110-111 residual preserves the token pair in the file's byte stream (markdown
strikethrough does not remove text from `tr | grep` matching). All 7 hits are
enumerated below with dispositions:

| Hit | Location | Disposition |
|-----|----------|-------------|
| 1 | v1.1 supersession note (:111-112 in v1.15 numbering) | Struck and annotated v1.15 (F-SP19-001) — the strikethrough markdown `~~Q2 also rules that \`upstreamdial.Handle\` gains \`SetFrameCallback...~~` preserves the token pair in the byte stream; the claim is retracted; not a live assertion |
| 2 | Summary-of-Rulings Q1 row (~:2096 in v1.15 numbering) | Already struck and annotated v1.7 (F-SP7-003) — `~~\`Handle\` gains \`SetFrameCallback(fn FrameFn)\` seam~~` with `*(amended v1.7 — F-SP7-003: ...)` annotation; not a live claim |
| 3 | Changelog v1.7 row (~:38) | History-preserved verbatim — quotes the pre-Option-A wording as the subject of the F-SP7-003 remediation description; not a live claim |
| 4 | Changelog v1.15 row (~:46) | Meta-reference — the v1.15 changelog entry quotes the F-SP19-001 residual text as the defect being remediated; not a live claim |
| 5 | v1.15 Addendum canonical-pattern line (~:2388) | Meta-reference — the `grep -o` argument string in the documented pattern command; not a binding claim |
| 6 | Pass-19 Adjudicated table row (~:2588) | Meta-reference — the F-SP19-001 finding description in the adjudicated table quotes the residual as the defect; not a live claim |
| 7 | This addendum hit-count table (this row) | Meta-reference — the disposition text in the table above; not a live claim |

**Re-certification:** Zero live unannotated Option-B claims remain. All 7 joined-line hits
are either struck historical text with explicit retraction annotations (hits 1-2) or
meta-references in changelog/adjudicated-section/addendum documentation (hits 3-7).
The sweep-incompleteness root cause (single-line patterns cannot match cross-line token
pairs) is the 6th instance of incomplete-sweep class: F-SP7-003, F-SP10-001, F-SP13-001,
F-SP14-001, F-SP15-001, F-SP19-001. This is also the 2nd instance of a false
sweep-completeness certification (the first was the F-SP7-003 initial 4-pattern
certification that missed two Q1-body residuals; this is the expanded-8-pattern
certification that missed the cross-line residual). The canonical multi-line-tolerant
pattern above supersedes single-line pattern sets for any future Handle-placement sweep.

---

## F-SP7-004 Cross-Reference Version-Pin Policy (v1.7)

Story `S-BL.PE-RECEIVE-LOOP.md` Task 1 cites this note as "v1.2". **This is a story-writer
propagation item:** the story-writer must update "v1.2" → "v1.7" when elaborating the story.
This note does not own the story file.

**Version-pin policy ruling (v1.7):** Cross-references from story files to this placement
note SHOULD cite "current version per frontmatter" rather than a hardcoded version string.
Hardcoded version citations become stale on every amendment. The story-writer may use either
"current version" language or a hardcoded version updated to "v1.7" at elaboration time.
This note's own internal cross-references (e.g. "v1.3 retraction", "v1.4 F-SP4-001") are
amendment-history labels and are intentionally version-anchored; they do not require updates.

---

## F-SP7-005 Transient Stale-ModePE Window (v1.7 — accepted with rationale)

After the receive goroutine calls `_ = conn.Close()` on read-error exit (F-SP6-001 teardown
wiring), `connectedCount.Add(-1)` has NOT yet fired. `maintainConn` must observe the write
failure first (via its next `conn.SetWriteDeadline` or `conn.Write` call), then return to
`dialLoop`, which then calls `connectedCount.Add(-1)`. This means `Mode()` transiently
reports `ModePE` (connectedCount ≥ 1) for up to `keepaliveInterval` after the receive
goroutine exits.

**This transient is bounded by `keepaliveInterval` and is accepted with no AC obligation:**

1. No AC in this story asserts `Mode()` in the window between receive-goroutine exit and
   `maintainConn` write failure.
2. No `FrameFn` consumer runs during this window (the receive goroutine has exited; no new
   frames are delivered).
3. The transient is an inherent consequence of the F-SP6-001 teardown wiring — the receive
   goroutine triggering teardown via `conn.Close()` rather than waiting for `dialLoop` to
   detect liveness independently.
4. The bound is `≤ keepaliveInterval` (the maximum time until the next keepalive write
   attempt fails) plus processing latency.

**No implementation change required.** Observation recorded for completeness; the transient
is not a spec defect. Future stories that assert `Mode()` for liveness checks after
deliberate teardown MUST account for this window.

---

## Pass-6 Adjudicated-Clean (non-findings, per adversarial pass-6 report)

The following items were raised by the pass-6 adversary but adjudicated clean.
Recorded per "adjudicated-clean: cite pass-6 report, do not re-derive" instruction.
All four actionable findings (F-SP6-001 through F-SP6-004) are remediated above.

| Item | Adversary concern | Ruling |
|------|-------------------|--------|
| conn.Close() ordering under normal teardown | When `dialLoop` teardown causes `maintainConn` to return (keepalive write failure due to upstream-initiated close), `dialLoop` calls `conn.Close()` at step 8. The receive goroutine also calls `_ = conn.Close()` on its read-error exit. Is this a race on the conn? | Clean. The double-close is safe and idempotent per Go's `net.Conn` contract. `dialLoop`'s `conn.Close()` at step 8 and the receive goroutine's `_ = conn.Close()` may execute concurrently or in any order; the net package's `poll.FD.Close()` implements the close as a CAS on a closed flag — concurrent closes are safe. The second call returns an error (discarded). No unsafe state. |
| `peWriteFixture.accepted` as ModePE substitute — race with connector bootstrap delay | `peWriteFixture.accepted` receives the conn when the connector's `net.Accept()` fires. Could this precede `connectedCount.Add(1)` (step 3 of dialLoop), meaning the fixture receives the conn before the connector has incremented ModePE? | Clean (per pass-6 analysis, which correctly identified the "strictly before Add(1)" timing). *Note (v1.7 — F-SP7-001/F-SP7-002):* this adjudicated-clean row's analysis ("accepted receives at TCP accept time, which is strictly before Add(1)") was already correct; the v1.6 body text at the observable-substitute block was inconsistent with it. The v1.7 amendment of the body text brings the note into alignment with this row's correct analysis. The "rely on the `\"mode=PE\"` writer-output line" recommendation in the original row is superseded by F-SP7-001: `"mode=PE"` is a config-presence signal, not a stronger establishment guarantee. See v1.7 corrected-observables block above. |
| `SetFrameCallback` concrete-type call in runRouter — is `*Connector` still in scope at the call site? | After `connector.Start()`, does `runRouter` hold a `*Connector` or only a `Handle`? | Clean. At `8eb54a5` `mgmt_wire.go` line ~525–526 declares `connector := upstreamdial.New(...)` (concrete type `*Connector`) then immediately calls `connector.Start()`. The concrete-type variable is still in scope on the next line. Option A (F-SP6-002 ruling) inserts `connector.SetFrameCallback(frameFn)` between `New(...)` and `Start()` — the concrete type is unambiguously available at the call site. |

---

## Pass-7 Adjudicated-Clean (non-findings, per adversarial pass-7 report)

The following items were raised by or considered during pass-7 review but adjudicated
non-actionable (either clean, or a propagation item for the story-writer rather than
an amendment obligation for this note).

| Item | Concern | Ruling |
|------|---------|--------|
| v1.6 pass-6-adjudicated-clean row already had correct `accepted` timing | The pass-6 adjudicated-clean row for "peWriteFixture.accepted as ModePE substitute" (retained at end of this note) correctly stated `accepted` fires "strictly before Add(1)" and characterised it as "slightly EARLY relative to ModePE". This is consistent with F-SP7-001/F-SP7-002. The defect was only in the v1.6 body text being inconsistent with its own adjudication row. | Clean (no separate amendment beyond the F-SP7-001/F-SP7-002 body correction). The adjudicated-clean row is annotated v1.7 above for cross-reference. |
| `"mode=PE"` emission in `connector.go` | Does `connector.go` emit a `"mode=PE"` line that might be confused with `mgmt_wire.go`'s `"mode=PE"` line? | Clean. Verified: `connector.go` emits only `"mode=E (no upstream_routers configured)"` at :390 (via `c.logf`) when `connectedCount.Add(-1)` returns 0. There is no `"mode=PE"` emission from `connector.go`. The sole `"mode=PE"` sources are `mgmt_wire.go` :548 (startup writer block) and :587 (SIGHUP re-emit). This is consistent with F-SP7-001's ruling that `"mode=PE"` is a config-presence signal emitted by `runRouter`, not by `connector`. |
| F-SP7-004 story propagation — who is responsible for the "v1.2" fix? | Story file `S-BL.PE-RECEIVE-LOOP.md` Task 1 cites "v1.2". This note cannot amend the story (BARS: do NOT touch the story). | Story-writer propagation item only. This note records the obligation at F-SP7-004 above; the story-writer updates "v1.2" → "v1.7" at elaboration time. No action in this note beyond the F-SP7-004 record. |
| F-SP7-005 transient affects `Mode()` callers in testenv | `testenv.RouterHandle.Mode()` delegates to `connector.Mode()`. During the transient window, does this cause test false-greens? | Non-actionable for this story. No current story AC polls `testenv.RouterHandle.Mode()` after teardown; the transient is bounded by `keepaliveInterval` (typically 10ms in tests). Recorded at F-SP7-005. Future stories with tight Mode()-based teardown assertions must account for the bound. |

---

## Pass-11 Adjudicated-Clean (non-findings and pass-through items, per adversarial pass-11 report)

The following items were raised by the pass-11 adversary; F-SP11-001 and F-SP11-003 are
remediated above. F-SP11-002 is story-side and is not touched by this note.

| Item | Classification | Ruling |
|------|----------------|--------|
| F-SP11-001 — ExitsOnReadError injection recipe unrealizable + wrong error attribution | HIGH [spec-defect] — REMEDIATED in Q2 above | Corrected recipe mandates complete 44-byte header: byte[0]=0x01 (VersionByte, verified frame.go :23), byte[1]=0x07 (ErrInvalidFrameType path), PayloadLen=0x0000 at bytes[2:4] big-endian (verified frame.go :90), remaining bytes zero, conn NOT closed. io.ReadFull completes deterministically; ParseOuterHeader returns ErrInvalidFrameType at byte[1]; receive goroutine exits via read-error branch → conn.Close() → reconnect. Both defects in the v1.5 recipe named explicitly: (1) partial-header blocks io.ReadFull, (2) 0xFF at byte[0] triggers ErrVersionMismatch not ErrInvalidFrameType. Optional ErrVersionMismatch variant adjudicated ADD. |
| F-SP11-002 — story-side token budget line | LOW [token-budget] — STORY-SIDE ONLY | Not touched in this placement note. Story-writer handles this independently. |
| F-SP11-003 — §8.2 dangling "see elaboration note below" pointer | LOW [doc-drift] — REMEDIATED in §8.2 above | Dangling clause struck and annotated: production interface-set population is out of scope for this story; §8.5 governs the test-scoped set. |

---

## Pass-12 Adjudicated-Clean (non-findings and pass-12 confirmations, per adversarial pass-12 report)

F-SP12-001 is the sole actionable finding from pass-12 and is remediated in Q2 above.
The following pass-12 confirmations are recorded per the pass-12 report.

| Item | Classification | Ruling |
|------|----------------|--------|
| F-SP12-001 — ARCH-08 §6.5 parenthetical contradiction (11th blast-radius location) | MED [spec-completeness] — REMEDIATED in Q2 above | Q2 ARCH-08 obligation extended with explicit second edit: parenthetical reconciliation wording specified verbatim; blast-radius count ruled 11 (unified: 10 Q3 frame sweep + 1 Q2 ARCH-08 parenthetical, separate enumeration kept under §6.5 obligation). *(count amended v1.11 — F-SP13-001: 11 → 12)* |
| Both corrected recipes (ErrInvalidFrameType + ErrVersionMismatch) are realizable | Pass-12 confirmation | Confirmed: the 44-byte corrected recipe mandated in v1.9 (byte[0]=0x01, byte[1]=0x07, PayloadLen=0x0000, remaining bytes zero, conn held open) and the optional ErrVersionMismatch variant (byte[0]=0xFF, PayloadLen=0x0000, conn held open) are both physically realizable — io.ReadFull completes on a single 44-byte write, no timing gymnastics required. |
| All four recipe copies byte-identical | Pass-12 confirmation | Confirmed: the corrected recipe appears in four locations in this note (Q2 pin-test shape, Q2 "BINDING corrected recipe" block, Q2 "Why the old recipe fails" clarification, Pass-11 adjudicated-clean table). All four are byte-identical on the wire-value spec (byte[0]=0x01, byte[1]=0x07, bytes[2:4]=0x00 0x00, bytes[4:44]=0x00). |
| 10 frame blast-radius locations byte-exact in Q3 table | Pass-12 confirmation | Confirmed: all 10 locations in the Q3 blast-radius table are correctly enumerated and the required changes are precisely specified. The FrameTypePEConnect/Valid() sweep locations remain 10; the ARCH-08 parenthetical is a distinct 11th location under Q2, not a Q3 item. *(11th location count stands; 12th added by v1.11 — F-SP13-001)* |
| ARCH-02 frame_type table amendment target exact | Pass-12 confirmation | Confirmed: ARCH-02 §"Outer Header Format" `frame_type` row amendment is correctly specified in Q3 ("add `pe_connect=0x06`") and the FCL row for ARCH-02 is accurate. No change to this obligation. |

---

## Pass-13 Adjudicated-Clean (non-findings and pass-13 confirmations, per adversarial pass-13 report)

F-SP13-001 is the sole actionable finding from pass-13 and is remediated in Q2 above.
The following pass-13 confirmations are recorded per the pass-13 report.

| Item | Classification | Ruling |
|------|----------------|--------|
| F-SP13-001 — ARCH-08 §6.6.2 forbidden-edges bullet: §6.5 sibling with identical stale import-set claims (12th blast-radius location) | MED [spec-completeness] — REMEDIATED in Q2 above | Q2 ARCH-08 obligation extended with a third edit target: §6.6.2 upstreamdial forbidden-edges bullet, three sub-edits in same commit as §6.5 edits; binding replacement bullet text quoted verbatim (sub-edits (a) allowed-import set, (b) cycle-freeness enumeration, (c) F-P1-001 rationale with frame re-add note); F-P7-002 clause preserved untouched; blast-radius count updated 11 → 12; class-closure grep transcript recorded (0 further hits beyond the two edit targets and changelog rows). |
| All ~11 test recipes realizable including AC-002 exhaustion, AC-003 discard, AC-004 byte-contract pin | Pass-13 confirmation | Confirmed: all test shapes specified in this note are physically realizable against the verified tree. AC-002 E-FWD-001 exhaustion: `peWriteFixture.WriteFrame` injects assembled `FrameTypeData` frame via accepted PE conn → receive goroutine → `OnFrameArrival` → split-horizon exhaustion with single-interface set → E-FWD-001. AC-003 discard: fixture writes `FrameTypePEConnect` frame → receive goroutine's discard branch fires → `OnFrameArrival` never called → E-FWD-001 never appears. AC-004 byte-contract pin (`TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader`): two frames with differing `Envelope.SrcAddr` both produce independent `crc32` keys → both reach `OnFrameArrival` independently → two E-FWD-001 emissions confirmed. |
| ARCH-02 single-row amendment adequate | Pass-13 confirmation | Confirmed: ARCH-02 §"Outer Header Format" `frame_type` table row is the sole location in ARCH-02 requiring amendment; no other ARCH-02 section enumerates frame type constants by value in a way that would be stale. The single-row amendment obligation specified in Q3 is complete. |
| §6.4 registration adequacy — no additional registration rows needed | Pass-13 confirmation | Confirmed: the §6.4 prospective-positions table in ARCH-08 registers packages before their first commit. The `internal/upstreamdial` package is already registered at position 19 (Wave 7, S-7.04-FU-PE-CONNECTOR). Adding a direct `frame` import edge to an already-registered package is a §6.5 amendment (allowed-import-set extension), not a new §6.4 registration. No new row in the §6.4 prospective table is required for this story. |
| PROSPECTIVE and pre-merge machine-verification qualifiers — deferred to merge-time | Pass-13 confirmation | Confirmed per v1.10 ruling: the PROSPECTIVE marker on the §6.5 row and the "final machine-verification at merge" qualifier are updated by the implementer at merge time (when `cee8e8b` reference is replaced by the actual merge SHA). This story's placement note specifies the textual edits; the implementer strips the PROSPECTIVE qualifier after merging. No additional obligation in this note. |

---

## Pass-14 Adjudicated-Clean (non-findings and pass-14 confirmations, per adversarial pass-14 report)

F-SP14-001 is the sole actionable finding from pass-14 and is remediated in Q3 above.
The following pass-14 confirmations are recorded per the pass-14 report.

| Item | Classification | Ruling |
|------|----------------|--------|
| F-SP14-001 — BC-2.01.004:61 (Postcondition 2 outer-header layout table, frame_type row) cited zero times despite being the co-canonical wire-format pair to ARCH-02:74 | MED [spec-completeness] — REMEDIATED in Q3 above | BC-2.01.004:61 amendment obligation added to Q3 ARCH-02 region: frame_type row must append `, pe_connect=0x06` in the SAME commit as ARCH-02:74 and FrameTypePEConnect; before/after rows quoted verbatim; rationale cites F-P8-008 co-canonical precedent and BC-2.01.004 v1.2 sync-practice; class-closure grep transcript recorded (2 hits each for `"arq=0x04, fec=0x05"` and `"empty_tick=0x02"` — exactly BC-2.01.004:61 and ARCH-02:74, no third sibling). Blast-radius arithmetic updated: BC-2.01.004:61 is placed as wire-format spec pair partner to ARCH-02:74 (both same-commit parallel obligations, sibling of but not inside the unified-12 count). Arithmetic sentence for story propagation stated verbatim. |
| All existing recipes realizable — ARCH-08 fence holds | Pass-14 confirmation | Confirmed: ARCH-08 §6.5 and §6.6.2 obligations specified in v1.11 remain unchanged and realizable. The BC-2.01.004 amendment is a spec-document edit only; it does not alter any implementation obligation or test recipe. |
| FCL↔Task bijection holds | Pass-14 confirmation | Confirmed: the 9-row FCL table (rows 1–9) and the Q3 obligation structure remain internally consistent after the BC-2.01.004 addition. BC-2.01.004 is a spec file and its amendment is captured under the Q3 ARCH-02 obligation (same-commit discipline). No new FCL row is needed: story v1.14 will EXTEND existing FCL row 9 (the ARCH-02 row) and Task 3 to carry BC-2.01.004:61 as the second half of the wire-format spec pair — row count stays 9, and the "same commit" discipline already governs both edits. |
| POL-001 / POL-002 pass | Pass-14 confirmation | Confirmed: no new POL-001 (canonical-source violation) or POL-002 (sweep-incomplete) defect classes introduced by v1.12. The class-closure grep transcript demonstrates sweep completeness for the frame_type enum row across `.factory/specs/`. |
| Cosmetic FCL item-numbering double-listing observation | Observation — NOT a finding | Lines ~501/~540 in the FCL table appear under two historical item numbers due to v1.2 + v1.3 multi-pass annotations; total of 10 locations in the Q3 blast-radius table is correct. The double-listing is presentation-cosmetic, within the adjudicated fence, and does NOT affect any implementation obligation. No action. |

---

## Pass-17 Adjudicated (per adversarial pass-17 report)

F-SP17-001 is the sole actionable finding from pass-17 and is remediated in Q2 above.
All other pass-17 items are adjudicated below.

| Item | Classification | Ruling |
|------|----------------|--------|
| F-SP17-001 — AC-003 discrimination contract test-set underdetermination: whitelist-data-only implementation passes all ~11 named tests while silently dropping FrameTypeCtl | MED [spec-gap / test-set underdetermination] — ACCEPTED; REMEDIATED in Q2 above | New BINDING unit test `TestConnector_ReceiveLoop_CtlFrameForwardedToCallback` added in Q2; test descriptor, harness family, and inverted-assertion shape specified verbatim. Companion cosmetic fix (else-branch comment gains empty_tick) recorded as story-writer obligation. Test counts updated: connector minimum 6 → 7; total net-new ~11 → ~12. |
| P1b — OnFrameArrival concurrency: hitCountMu, DropCache mu | Pass-17 P1b confirmation | Clean. `OnFrameArrival`'s internal hit-count state is protected by `hitCountMu` (verified at `8eb54a5` in `internal/routing/on_frame_arrival.go`); `DropCache.AddIfAbsent` is protected by its own internal mu (verified at `8eb54a5` in `internal/multipath/multipath.go`). Concurrent calls from the single receive goroutine per connection and any other callers are safe. No implementation change required. |
| P1b — ReloadAddrs set-diff isolation | Pass-17 P1b confirmation | Clean. `Connector.ReloadAddrs` computes a set-diff of added/removed addresses and fires `addrCancel` entries and new-dial goroutines without sharing mutable state with the receive goroutine. The receive goroutine holds only the `net.Conn` passed to it at dial time; it does not read `c.addrs` or any `addrCancel` map. No race. |
| P1b — Stop() stopOnce idempotent | Pass-17 P1b confirmation | Clean. `Connector.Stop()` is guarded by `c.stopOnce.Do(...)` (verified at `8eb54a5`). Concurrent or repeated calls are safe; only the first call closes `c.stopCh` and blocks on `<-c.doneCh`. No implementation change required. |
| P1c — DRAIN-WIRE seam: no concrete API expectation | Pass-17 P1c confirmation | Clean. The S-7.04-FU-DRAIN-WIRE scope boundary (§"Scope Boundary vs S-7.04-FU-DRAIN-WIRE") describes the dependency at the level of "receive loop provides the path DRAIN broadcast traverses." No concrete API (method signature, channel type, or interface addition) is committed to in this note for the DRAIN-WIRE story. The boundary table is illustrative; the DRAIN-WIRE story is a backlog story. No obligation locked. |
| P1d — VP traceability: no VP pins a 5-type enum or Valid() bound | Pass-17 P1d confirmation | Clean. The `vp_traces: []` frontmatter is correct: no VP in the VP index pins a 5-type FrameType enum or a specific `Valid()` upper bound. The Valid() widening to 6 types does not violate any VP. VP-037 (DRAIN-WIRE unblock path) is traced through S-7.04-FU-DRAIN-WIRE per this note's frontmatter — correct and unchanged. |
| P2 — POL-001 / POL-002 pass | Pass-17 P2 confirmation | Confirmed: no new canonical-source violation (POL-001) or incomplete-sweep (POL-002) defect class introduced by v1.13. The new test adds no spec-document amendment obligation; the companion comment fix is scoped to the story's discrimination sketch (story-writer propagation item only). |
| P3 — DataFrameForwarded + FlapCycleJoin re-executed realizable | Pass-17 P3 confirmation | Confirmed: `TestConnector_ReceiveLoop_DataFrameForwardedToCallback` (AC-001 unit) and the AC-005 FlapCycleJoin test remain physically realizable against the verified tree at `8eb54a5`. The new `TestConnector_ReceiveLoop_CtlFrameForwardedToCallback` test uses the same harness family and is realizable by the same argument (in-package accept-and-write fixture, `outerassembler.Assemble` with `FrameTypeCtl`, server-side `conn.Write`). No new import edge or API required beyond what v1.12 already mandates. |

---

## Pass-18 Adjudicated (per adversarial pass-18 report)

F-SP18-001 is the sole actionable finding from pass-18 and is remediated in Q2 above.
All other pass-18 items are adjudicated below.

| Item | Classification | Ruling |
|------|----------------|--------|
| F-SP18-001 — AC-003 discard-side loop-continuation unpinned: `PEConnectFrameDiscarded` asserts only "FrameFn NOT invoked"; discard-as-close passes every named test while producing reconnect storm | MED [spec-gap / test-set underdetermination] — ACCEPTED; REMEDIATED in Q2 above | Binding remediation: EXTEND `TestConnector_ReceiveLoop_PEConnectFrameDiscarded` (extend-not-add; counts UNCHANGED at 7 connector / ~12 total). Two-frame recipe on same conn: `FrameTypePEConnect` frame THEN `FrameTypeData` frame; assert (a) FrameFn NOT invoked for bootstrap frame, (b) FrameFn IS invoked for subsequent data frame (`hdr.FrameType == frame.FrameTypeData`). Kills discard-as-close: close tears down conn before data frame read, failing (b). Symmetric to forward-side NoDuplicateSuppression continuation pin (F-SP4-001 axis). Realizability confirmed: `frame.ReadOuterFrame` loops on `io.ReadFull(44)` + PayloadLen reads; length-delimited; segment-boundary-independent. |
| P1a — Ctl-pin realizability: CtlFrameForwardedToCallback remains realizable with the extended PEConnectFrameDiscarded | Pass-18 P1a confirmation | Clean. `FrameTypeCtl = 0x03` passes `Valid()` (verified `internal/frame/frame.go`: `Valid()` checks `f >= FrameTypeData && f <= FrameTypePEConnect`; 0x03 is in range). `outerassembler.Assemble` passes `FrameType` field through to the outer header byte[1] unchanged (verified `internal/outerassembler/assemble.go` :102 — `ChannelFrame.FrameType` written directly to the wire header). No special-case for `FrameTypeCtl` in the receive goroutine before `frameFn` call. The `CtlFrameForwardedToCallback` test is unaffected by the `PEConnectFrameDiscarded` extension. |
| P1b — Kill transcript: does the two-frame extension also kill other previously-identified malicious implementations? | Pass-18 P1b confirmation | Confirmed cumulative kill coverage: payload-only reconstruction killed by `NoDuplicateSuppression` full-frame crc32 (F-SP4-001 axis); header-only reconstruction judged within ledger item 1 byte-contract coverage (44-byte io.ReadFull reads full header; no partial-header path reachable); callback-before-check killed by `PEConnectFrameDiscarded` assertion (a) (FrameFn not called for bootstrap); reconnect-skip killed by `ExitsOnReadError` PC (b) (connector re-dials after error exit). The new extension adds: discard-as-close killed by assertion (b) (data frame not received after close). |
| P1c — AC-002/AC-004 count-tolerance: ~12 count is presence-assertion + ≥2 precise; extension does not alter this | Pass-18 P1c confirmation | Clean. The ~12 total is a presence-assertion count (approximately 12 net-new tests minimum); the precise counts are: 1 `frame_test` amendment + 7 `connector_test` unit tests + 4 integration tests. The `PEConnectFrameDiscarded` extension is a within-test edit, not an additional test. The 7 connector count and ~12 total count are accurate and unchanged. |
| P1d — Note-ruling/story coherence: extended test shape recorded in ruling block; story-writer obligation scoped to "extend, not add" | Pass-18 P1d confirmation | Confirmed. The v1.14 ruling block in Q2 specifies the two-frame recipe verbatim (recipe text binding for story-writer and implementer). Story propagation obligation: story-writer updates `TestConnector_ReceiveLoop_PEConnectFrameDiscarded` per the binding recipe; no new test name or count change to propagate. |
| P2 — POL-001 / POL-002 pass | Pass-18 P2 confirmation | Confirmed: no new canonical-source violation (POL-001) or incomplete-sweep (POL-002) defect class introduced by v1.14. The extension adds no spec-document amendment obligation; all existing blast-radius counts (frame sweep unified-12, wire-format spec pair) are unaffected. |
| P3 — ExitsOnReadError re-traced realizable with two-frame extension | Pass-18 P3 confirmation | Confirmed. `TestConnector_ReceiveLoop_ExitsOnReadError` (v1.9 corrected recipe: complete 44-byte header, byte[0]=0x01, byte[1]=0x07, PayloadLen=0x0000, conn NOT closed) remains realizable. The `PEConnectFrameDiscarded` extension does not change the error-exit recipe or its timing contract. The two tests are independent: `ExitsOnReadError` uses a malformed header (triggers read-error branch); the extended `PEConnectFrameDiscarded` uses two well-formed frames (exercises discard-continue-then-forward). No interaction. |

---

## Pass-19 Adjudicated (per adversarial pass-19 report)

F-SP19-001 is the sole actionable finding from pass-19 and is remediated in the v1.1 supersession note above (lines :110-111) and the F-SP7-003 sweep transcript addendum. All other pass-19 items are adjudicated below.

| Item | Classification | Ruling |
|------|----------------|--------|
| F-SP19-001 — v1.1 supersession note :110-111: live unannotated Option-B claim spanning a line break ("Q2 also rules that `upstreamdial.Handle` gains `SetFrameCallback(fn FrameFn)`") contradicts F-SP6-002 Option A ruling and falsely attributes Handle placement to Q2 | MED [doc-drift / incompletely-discharged prior remediation] — ACCEPTED; REMEDIATED above | Residual struck and annotated in v1.1 supersession note using the same ~~strikethrough~~ + `*(amended vN — ...)` pattern as all prior Option-B-retraction annotations. Retraction states: (a) F-SP6-002 Option A is binding — `SetFrameCallback` is on concrete `*Connector` ONLY, `Handle` interface unchanged; (b) Q2 rules the framing primitive (`ReadOuterFrame`, FrameFn signature, frame import), not Handle placement — the Option-B attribution to Q2 is retracted. F-SP7-003 sweep transcript extended with a v1.15 addendum recording the root cause (single-line and expanded-8-pattern sets cannot match cross-line token pairs), the NEW canonical multi-line-tolerant pattern (`tr '\n' ' ' < FILE \| grep -o "Handle. gains .SetFrameCallback"`), the post-fix hit count (7 hits; 2 struck historical, 5 meta-references in documentation) with all per-hit dispositions. Sweep re-certified: zero live unannotated Option-B claims remain. This is the 6th incomplete-sweep-class instance (F-SP7-003, F-SP10-001, F-SP13-001, F-SP14-001, F-SP15-001, F-SP19-001) and the 2nd false sweep-completeness certification. |
| P1a — Two-frame extension (F-SP18-001) realizable; P1b round-3 archetypes all killed/non-observable | Pass-19 P1a + P1b confirmation | Clean. P1a: two-frame extension in `PEConnectFrameDiscarded` remains realizable (confirmed pass-18). P1b kill transcript: hdr-mutation archetype killed by call-site assertions (verified pass-17 / pass-18 ledger); double-invoke-masked-idempotent archetype: DropCache `AddIfAbsent` is idempotent under duplicate keys — no AC forbids idempotent double-invocation, and raw aliasing is impossible under the mandated fresh-allocation idiom (`outerassembler.Assemble` + synchronous consumption before next `io.ReadFull`); no new archetype introduced by v1.14. |
| P1c — Cross-layer triples coherent | Pass-19 P1c confirmation | Clean. The three cross-layer triples (AC-001/AC-002/AC-004: PE connection → receive goroutine → `OnFrameArrival` → E-FWD-001; AC-003: bootstrap frame → discard branch; AC-005: read-error → exit → conn.Close() → reconnect) remain internally coherent. The v1.15 amendment is annotation-only within the v1.1 supersession note; it does not alter any implementation obligation or test recipe. |
| P2 — POL-001 / POL-002 pass | Pass-19 P2 confirmation | Confirmed: no new canonical-source violation (POL-001) or incomplete-sweep (POL-002) defect class introduced by v1.15. The v1.15 change is a note-only doc-drift correction and sweep re-certification; all blast-radius counts (frame sweep unified-12, wire-format spec pair) are unaffected. |
| P3 — CtlPin + NoDuplicateSuppression re-traced realizable | Pass-19 P3 confirmation | Confirmed. `TestConnector_ReceiveLoop_CtlFrameForwardedToCallback` (v1.13 — F-SP17-001): uses same in-package accept-and-write fixture family, `FrameTypeCtl = 0x03` passes `Valid()`, `outerassembler.Assemble` passthrough verified — realizable unchanged. `TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader` (AC-004 byte-contract pin): two frames with differing `Envelope.SrcAddr` produce independent `crc32` keys → both reach `OnFrameArrival` independently — realizable unchanged. Neither test has any dependency on the Handle-placement question amended in v1.15. |
| Ledger passes 1–18 — all hold | Pass-19 ledger confirmation | Confirmed. No ruling in passes 1–18 is affected by the v1.15 note-only amendment. The entire pass-1 through pass-18 adjudicated ledger stands as published. |

---

## Pass-20 Adjudicated (per adversarial pass-20 report)

F-SP20-001 is the sole actionable finding from pass-20 and is remediated in the three-part annotation at the READ-error disposition contract block (:365), the prose retraction (:386-387), and the v1.5 sketch banner (:402 region). All other pass-20 items are adjudicated below.

### v1.16 Class-Closure Sweep: versioned binding blocks and sketches

This is the 7th instance of the incomplete-sweep class ("later version supersedes earlier binding block without in-place annotation"). F-SP20-001 triggered a full enumeration of every versioned binding block and sketch header in the note to verify each carries either (a) confirmed current status or (b) an in-place supersession marker pointing to its successor.

**Grep commands used:**

```
grep -n 'binding.:' note.md          (binding headers)
grep -n 'BINDING' note.md            (BINDING headers)
grep -n '[Ss]ketch' note.md          (sketch headers)
grep -n 'SUPERSEDED\|superseded\|amended v1\.' note.md   (existing annotations)
```

Multi-line-tolerant patterns applied where token pairs could span newlines (per F-SP19-001 lesson).

**Enumeration (17 blocks, complete):**

| # | Line | Block / Header | Version | Status | Disposition |
|---|------|---------------|---------|--------|-------------|
| 1 | 120 | `SetFrameCallback interface placement (v1.6 — F-SP6-002, BINDING)` | v1.6 | Current | No supersession — Option A ruling (concrete `*Connector` only; Handle unchanged) remains binding. |
| 2 | 159 | `Production wiring order in runRouter (binding)` | v1.4 | Current | No supersession — construct → SetFrameCallback → Start ordering contract is the live requirement. |
| 3 | 306 | `FrameFn return-value contract (v1.4 — F-SP4-001, binding)` | v1.4 | Current | No supersession — discard-and-continue (`_ = frameFn(...)`) mandate is still binding. |
| 4 | 365 | `READ-error disposition contract (v1.5 — F-SP5-001, binding)` | v1.5 | **Superseded in part** | **REMEDIATED v1.16 — F-SP20-001**: header now carries "; superseded in part by v1.6 — F-SP6-001 (conn.Close() teardown wiring): see the 'Read-error conn.Close() teardown wiring' section below". Prose :386-387 struck and annotated. v1.5 sketch banner inserted. |
| 5 | 402 | `Receive-goroutine sketch (updated — replaces the elided { ... } from v1.4)` | v1.5 | **Superseded** | **REMEDIATED v1.16 — F-SP20-001**: banner blockquote inserted immediately above sketch fence: "SUPERSEDED (v1.16 — F-SP20-001): this v1.5 sketch omits the binding `_ = conn.Close()` before `return`…". Sketch body preserved unmodified (history). |
| 6 | 423 | `AC-005 read-error-exit pin test (v1.5 — binding for story-writer)` | v1.5 | Current (amended) | Test name still binding; the v1.5 single-byte injection recipe was already struck and annotated at :434 by v1.9 (F-SP11-001). In-place annotation present; compliant. |
| 7 | 447 | `BINDING corrected recipe (v1.9 — F-SP11-001)` | v1.9 | Current | No supersession — 44-byte complete header recipe is the live binding recipe. |
| 8 | 495 | `AC-003 forwarding-completeness pin test (v1.13 — F-SP17-001, BINDING)` | v1.13 | Current | No supersession — CtlFrameForwardedToCallback test requirement is the live binding. |
| 9 | 543 | `AC-003 discard-continuation pin (v1.14 — F-SP18-001, BINDING)` | v1.14 | Current | No supersession — two-frame PEConnectFrameDiscarded extension is the live binding. |
| 10 | 606 | `AC-001 PC-3 / AC-004 precondition observable substitutes (v1.6 — F-SP6-003, binding for story-writer)` | v1.6 | Superseded in part (annotated) | v1.7 F-SP7-001/F-SP7-002 retraction annotations already present at :620 and :628 ("amended v1.7 — F-SP7-001/F-SP7-002: … RETRACTED"). In-place annotations present; compliant. |
| 11 | 638 | `Corrected observable semantics for AC-001 PC-3 and AC-004 precondition (v1.7 — F-SP7-001 + F-SP7-002, BINDING)` | v1.7 | Current | No supersession — three-observable semantics (peWriteFixture.accepted, "mode=PE", connectedCount observation constraints) are the live binding. |
| 12 | 706 | `Read-error conn.Close() teardown wiring (v1.6 — F-SP6-001, BINDING)` | v1.6 | Current | No supersession — this section IS the superseding binding wiring that F-SP20-001 remediation points to. |
| 13 | 738 | `Updated receive-goroutine sketch (v1.6 — replaces v1.5 sketch)` | v1.6 | Current | No supersession — this is the binding v1.6 sketch (with `_ = conn.Close()`). The v1.5 sketch at :402 now carries an in-place banner pointing here. |
| 14 | 850 | `ARCH-08 parenthetical reconciliation obligation (v1.10 — F-SP12-001, BINDING)` | v1.10 | Current | No supersession — §6.5 parenthetical reconciliation wording remains the live obligation. |
| 15 | 902 | `§6.6.2 upstreamdial forbidden-edges bullet — ARCH-08 edit obligation (v1.11 — F-SP13-001, BINDING)` | v1.11 | Current | No supersession — §6.6.2 forbidden-edges replacement bullet remains the live obligation. |
| 16 | 1160 | `BC-2.01.004:61 sibling amendment obligation (v1.12 — F-SP14-001, BINDING)` | v1.12 | Current | No supersession — BC-2.01.004:61 same-commit amendment obligation remains the live binding. |
| 17 | 1446 | `Q6 per-reconnect join (binding)` | v1.4+ | Current | No supersession — per-reconnect-iteration join requirement remains the live binding. |
| 18 | 262 | `` **`FrameFn` byte-contract (binding — F-SP3-001 correction):** `` | v1.3 / F-SP3-001 | Current | Verified at :262–305. Content: `FrameFn raw` parameter MUST be full wire frame (outer header + payload); three-point rationale (crc32 key, SplitHorizon.Forward verbatim forwarding, OnFrameArrival doc comment); reconstruction idiom (`EncodeOuterHeader` + append). No supersession annotation present; none needed — no later version overrides this contract. This block's header form (`binding — F-SP3-001 correction`) does not match any of the v1.16 recorded patterns (`binding.:`, `BINDING`, `[Ss]ketch`); that mismatch is the root cause of the miss. |
| 19 | 511 | `` **Test shape (binding for story-writer and implementer):** `` | v1.13 / F-SP17-001 sub-block | Current | Verified at :511–517. Content: test shape for `TestConnector_ReceiveLoop_CtlFrameForwardedToCallback` — assemble a FrameTypeCtl frame, use in-package accept-and-write fixture, assert FrameFn IS invoked and `hdr.FrameType == frame.FrameTypeCtl`. This block is a binding sub-element of the AC-003 forwarding-completeness pin test block (row 8, :498 / `AC-003 forwarding-completeness pin test (v1.13 — F-SP17-001, BINDING)`). No supersession; row 8 in this table already certifies the outer block Current; this sub-header is a companion. Header form (`binding for story-writer and implementer`) not matched by recorded patterns. |
| 20 | 1812 | `` **Pin test shape (binding for story-writer):** `` | Q9 region / F-SP3-001 byte-contract pinning obligation | Current | Verified at :1812–1836. Content: `TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader` pin test — inject two frames with differing `OuterHeader.SrcAddr` and identical payload; assert E-FWD-001 fires TWICE (proves crc32 computed over full frame bytes, not payload-only). Located in Q9 §9.1a "Byte-contract pinning test obligation (F-SP3-001)". No supersession; this block has not been amended or superseded by any subsequent version. Header form (`binding for story-writer`) not matched by recorded patterns. |
| 21 | 1928 | `` **Binding harness rule:** `` | Q9.3 / F-SP2-003 runRouter mandate | Current | Verified at :1928–1955. Content: every AC asserting OnFrameArrival (AC-001, AC-002, AC-004) MUST use the real `runRouter` goroutine pattern, NOT `testenv.Restart`; rationale: testenv.Restart builds a bare `upstreamdial.New` without calling `SetFrameCallback` — nil FrameFn; vacuous assertion. Located in Q9.3 "Harness rule (F-SP2-003)". No supersession; block is the canonical live Q9.3 ruling. Header form (`Binding harness rule`) not matched by recorded patterns. |

**Sweep result (v1.16 original):** 17 blocks enumerated. 2 defective (rows 4–5, both remediated in v1.16). 1 previously remediated (row 6, v1.9 annotation present). 1 previously remediated in part (row 10, v1.7 annotations present). 13 fully current with no supersession needed. Zero unannotated stale binding blocks remain after v1.16.

**v1.17 Addendum — F-SP21-001 Sweep Extension and Re-Certification:**

The v1.16 sweep certified completeness based on three grep patterns (`grep -n 'binding.:'`, `grep -n 'BINDING'`, `grep -n '[Ss]ketch'`). F-SP21-001 established that these patterns cannot match bold binding-block headers whose text does not contain a bare `binding:` or `BINDING` token. The canonical pattern that enumerates ALL bold binding headers is:

```
grep -nE '\*\*[^*]*[Bb]inding' S-BL.PE-RECEIVE-LOOP-placement-note.md
```

**Canonical grep execution (run at v1.17):**

```
:262:**`FrameFn` byte-contract (binding — F-SP3-001 correction):**
:307:**FrameFn return-value contract (v1.4 — F-SP4-001, binding):**
:367:**READ-error disposition contract (v1.5 — F-SP5-001, binding); superseded in part...:**
:406:> **SUPERSEDED (v1.16 — F-SP20-001):** this v1.5 sketch omits the binding `_ = conn.Close()`...
:427:**AC-005 read-error-exit pin test (v1.5 — binding for story-writer):**
:512:**Test shape (binding for story-writer and implementer):** Assemble a complete valid frame
:567:**Binding remediation: EXTEND `TestConnector_ReceiveLoop_PEConnectFrameDiscarded`...
:570:**Extended two-frame recipe (binding for story-writer and implementer):** On the SAME
:610:**AC-001 PC-3 / AC-004 precondition observable substitutes (v1.6 — F-SP6-003, binding for story-writer):**
:618:**Ruling: substitute the following observables (binding for story-writer):**
:680:**Corrected ruling per observable, binding for story-writer:**
:688:**AC-001 PC-3 corrected precondition (binding for story-writer):**
:695:**AC-004 precondition corrected (binding for story-writer):**
:869:**Concrete replacement wording for the parenthetical (binding for implementer):**
:930:**Binding replacement bullet text (v1.11 — implementer edits mechanically from this):**
:1321:> **[v1.8 supersession annotation — F-SP10-001]** Q5's test shape below is SUPERSEDED...Do not implement from this section — Q9 + the corrected-observables block govern.
:1450:> **Q6 per-reconnect join (binding):** Before `dialLoop` begins dialing a new
:1813:**Pin test shape (binding for story-writer):**
:1929:**Binding harness rule:** Every AC that asserts `OnFrameArrival` is reached...
:2657:| 5 | 402 | ... **Superseded** | **REMEDIATED v1.16 — F-SP20-001**...binding...
:2671:**Sweep result (v1.16 original):** 17 blocks enumerated...binding blocks remain after v1.16.
```

(Note: line numbers above are approximate post-v1.17-edit values; the canonical grep was run on the v1.16 content before the addendum edits. The count of 21 hits is correct for the pre-addendum file. The addendum text itself introduces additional meta-reference hits; those are dispositioned below.)

**Hit count (pre-addendum v1.16 file): 21 hits.** Per-hit dispositions reconciled against the extended table:

| Hit | Line (approx v1.16) | Block / context | Table row | Disposition |
|-----|---------------------|-----------------|-----------|-------------|
| 1 | :262 | `` **`FrameFn` byte-contract (binding — F-SP3-001 correction):** `` | Row 18 (added v1.17) | Versioned binding block — CURRENT, no supersession |
| 2 | :307 | `**FrameFn return-value contract (v1.4 — F-SP4-001, binding):**` | Row 3 | Already in v1.16 table — Current |
| 3 | :366 | `**READ-error disposition contract (v1.5 — F-SP5-001, binding); superseded in part...:**` | Row 4 | Already in v1.16 table — Superseded in part (annotated v1.16) |
| 4 | :405 | `> **SUPERSEDED (v1.16 — F-SP20-001):** this v1.5 sketch omits the binding...` | Row 5 | Banner text inside row 5 blockquote — meta-reference, not a binding block header; row 5 already in v1.16 table |
| 5 | :426 | `**AC-005 read-error-exit pin test (v1.5 — binding for story-writer):**` | Row 6 | Already in v1.16 table — Current (amended) |
| 6 | :511 | `**Test shape (binding for story-writer and implementer):**` | Row 19 (added v1.17) | Binding sub-block of row 8 — CURRENT, no supersession |
| 7 | :566 | `**Binding remediation: EXTEND \`TestConnector_ReceiveLoop_PEConnectFrameDiscarded\`...**` | Row 9 (v1.16 table) | Header of row 9 binding block — already in v1.16 table (row 9 line :543 is the outer block header; :566 is its disposition text, same block) |
| 8 | :569 | `**Extended two-frame recipe (binding for story-writer and implementer):**` | Row 9 sub-item | Binding detail within row 9 — companion to row 9; same block |
| 9 | :609 | `**AC-001 PC-3 / AC-004 precondition observable substitutes (v1.6 — F-SP6-003, binding for story-writer):**` | Row 10 | Already in v1.16 table — Superseded in part (annotated v1.7) |
| 10 | :617 | `**Ruling: substitute the following observables (binding for story-writer):**` | Row 10 sub-heading | Sub-heading within row 10 block; row 10 covers the outer header; not a separate block |
| 11 | :679 | `**Corrected ruling per observable, binding for story-writer:**` | Row 11 sub-heading | Sub-heading within row 11 block; row 11 (`:638`) covers the outer BINDING header |
| 12 | :687 | `**AC-001 PC-3 corrected precondition (binding for story-writer):**` | Row 11 sub-heading | Sub-heading within row 11 block |
| 13 | :694 | `**AC-004 precondition corrected (binding for story-writer):**` | Row 11 sub-heading | Sub-heading within row 11 block |
| 14 | :868 | `**Concrete replacement wording for the parenthetical (binding for implementer):**` | Row 14 sub-heading | Sub-heading within row 14 block (`:850`); row 14 covers the outer BINDING header |
| 15 | :929 | `**Binding replacement bullet text (v1.11 — implementer edits mechanically from this):**` | Row 15 sub-heading | Sub-heading within row 15 block (`:905`); row 15 covers the outer BINDING header |
| 16 | :1320 | `> **[v1.8 supersession annotation — F-SP10-001]** Q5's test shape below is SUPERSEDED...binding...` | N/A | Supersession banner inside Q5 blockquote (meta-reference, history-preserved annotation text); not a binding block header |
| 17 | :1449 | `` > **Q6 per-reconnect join (binding):** `` | Row 17 | Already in v1.16 table — Current |
| 18 | :1812 | `**Pin test shape (binding for story-writer):**` | Row 20 (added v1.17) | Versioned binding block — CURRENT, no supersession |
| 19 | :1928 | `**Binding harness rule:**` | Row 21 (added v1.17) | Versioned binding block — CURRENT, no supersession |
| 20 | :2638 | Table cell in v1.16 sweep table row 5 | N/A | Meta-reference — table cell text citing F-SP20-001 remediation; not a binding block header |
| 21 | :2652 | Sweep-result summary line | N/A | Meta-reference — v1.16 sweep result narrative; not a binding block header |

**Re-certification:** 21 grep hits total. 17 binding block headers (hits 1–3, 5–6, 7 outer block, 9, 10, 11, 17–19 map to table rows 3–11, 14–15, 17–21) — all covered in the extended 21-row table. 4 non-block-header hits (hits 4, 16, 20, 21) are meta-references or banner text, each dispositioned. After the v1.17 extension: **21 blocks total (17 from v1.16 + 4 added rows 18–21). All 21 are either CURRENT or carry in-place supersession annotations. Zero unannotated stale binding blocks remain after v1.17.** This is the 8th incomplete-sweep-class instance (F-SP7-003, F-SP10-001, F-SP13-001, F-SP14-001, F-SP15-001, F-SP19-001, F-SP20-001, F-SP21-001) and the 3rd false completeness certification (F-SP7-003 initial 4-pattern; F-SP19-001 expanded-8-pattern cross-line miss; F-SP21-001 this one).

**Post-edit meta-hit note (v1.17):** The 21-hit transcript above was captured BEFORE this addendum was inserted. Running the canonical pattern on the post-v1.17 file returns a larger count (67 at v1.17, verified by fresh grep run) because the pattern also matches documentation echoes introduced by this remediation itself: the transcript block's quoted hit lines, sweep-table rows 18–21, the v1.17 changelog row, and the Pass-21 Adjudicated section all contain bold text with "binding" or "Binding". These are meta-references, not binding blocks. The binding-block universe remains the 21 enumerated blocks (rows 1–21). Future auditors should either run the pattern and subtract documentation-region hits (changelog table, sweep section, Adjudicated sections), or verify that no new hit falls OUTSIDE those regions — a new out-of-region hit is a candidate unenumerated block.

| Item | Classification | Ruling |
|------|----------------|--------|
| F-SP20-001 — READ-error block :365-421: three-part defect: (1) header :365 lacks "amended v1.6" marker; (2) prose :386-387 states retracted mechanism verbatim ("Exit → dialLoop's existing teardown/reconnect path closes the conn and re-dials, which is the ONLY correct resync"); (3) v1.5 sketch :404-421 has bare `return` with no `_ = conn.Close()` and no banner | MED [doc-drift / incompletely-discharged prior remediation] — ACCEPTED; REMEDIATED above | Three-part annotation applied: (1) header :365 extended with "; superseded in part by v1.6 — F-SP6-001 (conn.Close() teardown wiring): see the 'Read-error conn.Close() teardown wiring' section below"; (2) prose :386-387 struck with ~~strikethrough~~ and annotated "*(amended v1.16 — F-SP20-001: RETRACTED. `maintainConn` is write-only (connector.go:399) and never observes receive-goroutine exit — dialLoop's teardown path does NOT fire on read-goroutine exit alone. The receive goroutine MUST itself call `_ = conn.Close()` before returning, converting the read-side failure into a write-side teardown. See the F-SP6-001 wiring section (binding).)*"; (3) banner blockquote inserted immediately above v1.5 sketch fence: "SUPERSEDED (v1.16 — F-SP20-001): this v1.5 sketch omits the binding `_ = conn.Close()` before `return` in the read-error branch (F-SP6-001). Do NOT implement from this sketch — the v1.6 'Updated receive-goroutine sketch' below (with `_ = conn.Close()`) is the binding version." Sketch body preserved unmodified (history preservation policy). Class-closure sweep performed (17 blocks): zero unannotated stale binding blocks remain. This is the 7th incomplete-sweep-class instance (F-SP7-003, F-SP10-001, F-SP13-001, F-SP14-001, F-SP15-001, F-SP19-001, F-SP20-001). |
| P1a — v1.15 strikethrough well-formed + canonical pattern reconciled 7/7 + story v1.18 metadata-only verified at diff level | Pass-20 P1a confirmation | Clean. All v1.15 changes hold: the v1.15 Option-B retraction in the v1.1 supersession note is well-formed and annotation-complete. The F-SP7-003 canonical multi-line pattern (`tr '\n' ' ' \| grep -o "Handle. gains .SetFrameCallback"`) 7-hit reconciliation is verified. Story v1.18 is metadata-only (no implementation obligation changed). |
| P1b — Other retracted mechanisms confirmed properly bannered: mode=PE (F-SP7-001), arqsend Q4/Q5 (F-SP10-001), single-byte recipe (F-SP11-001) | Pass-20 P1b confirmation | Clean. mode=PE: struck+annotated at v1.7 F-SP7-001/F-SP7-002, compliant (class-closure sweep rows 10-11). arqsend Q4: v1.8 supersession banner present ("Q4's test-role content below is SUPERSEDED"). arqsend Q5: v1.8 supersession banner present ("Q5's test shape below is SUPERSEDED"). Single-byte recipe: struck+annotated at v1.9 F-SP11-001 (:434). All four retracted mechanism blocks carry in-place annotation; none defective. |
| P1c — All five ACs testable/constructible/observable/single-reading | Pass-20 P1c confirmation | Clean. AC-001 (DataFrameForwardedToCallback): observable via FrameFn call-count assertion, constructible with outerassembler.Assemble + in-package fixture. AC-002 (byte-contract NoDuplicateSuppression): observable via crc32 key discrimination, constructible with two differing Envelope.SrcAddr frames. AC-003 (PEConnectFrameDiscarded + CtlFrameForwardedToCallback + discard-continuation): observable via FrameFn call presence/absence + two-frame sequence, constructible with real runRouter. AC-004 (ModePE precondition + ForwardedToCallback): observable via peWriteFixture.accepted gate + FrameFn assertion, constructible with real runRouter. AC-005 (ExitsOnReadError + ExitsOnVersionMismatch): observable via connector redial within timeout + goroutine exit, constructible with 44-byte malformed header injection. All five ACs have single unambiguous readings. |
| P1d — 10 note→story claims verified zero divergence | Pass-20 P1d confirmation | Confirmed. The v1.16 changes are note-side annotation only; no implementation obligation, test recipe, test name, or test count is altered. The story body carries the binding obligations unchanged. Zero divergence introduced by v1.16. |
| P2 — POL-001 / POL-002 pass | Pass-20 P2 confirmation | Confirmed: no new canonical-source violation (POL-001) or incomplete-sweep (POL-002) defect class introduced by v1.16. The three-part annotation is additive-plus-strikethrough (history preservation); all blast-radius counts (frame sweep unified-12, wire-format spec pair) are unaffected. Class-closure sweep (17 blocks) discharges POL-002 for the versioned-binding-block scope. |
| P3 — CtlPin + ExitsOnVersionMismatch re-traced realizable | Pass-20 P3 confirmation | Confirmed. `TestConnector_ReceiveLoop_CtlFrameForwardedToCallback` (v1.13 — F-SP17-001): `FrameTypeCtl = 0x03` passes `Valid()`, `outerassembler.Assemble` passthrough verified — realizable unchanged. `TestConnector_ReceiveLoop_ExitsOnVersionMismatch` (v1.9 optional — F-SP11-001): byte[0]=0xFF triggers ErrVersionMismatch → same read-error exit contract → conn.Close() → maintainConn write failure → reconnect — realizable unchanged. Neither test has any dependency on the v1.16 annotation changes. |
| Ledger passes 1–19 — all hold | Pass-20 ledger confirmation | Confirmed. No ruling in passes 1–19 is affected by the v1.16 note-only annotation amendments. The entire pass-1 through pass-19 adjudicated ledger stands as published. |

---

## Pass-21 Adjudicated (per adversarial pass-21 report)

F-SP21-001 is the sole actionable finding from pass-21 and is remediated in the v1.17 sweep-table extension (rows 18–21), the replacement of the recorded grep transcript with the canonical `grep -nE '\*\*[^*]*[Bb]inding'` pattern, and the v1.17 addendum block with per-hit reconciliation above.

| Item | Classification | Ruling |
|------|----------------|--------|
| F-SP21-001 — v1.16 sweep table (:2632-2650) certifies "every versioned binding block … 17 blocks … complete" but the enumeration missed four bold binding-block headers: :262 `` **`FrameFn` byte-contract (binding — F-SP3-001 correction):** `` (v1.3/F-SP3-001); :511 `**Test shape (binding for story-writer and implementer):**` (v1.13/F-SP17-001 sub-block inside the AC-003 forwarding-completeness pin test); :1812 `**Pin test shape (binding for story-writer):**` (Q9 §9.1a byte-contract pinning obligation); :1928 `**Binding harness rule:**` (Q9.3/F-SP2-003 runRouter mandate). Root cause: recorded grep patterns (`binding.:`, `BINDING`, `[Ss]ketch`) cannot match these header forms; manual pass also missed them. | MED [doc-drift / incomplete sweep-completeness certification] — ACCEPTED; REMEDIATED above | Four rows (18–21) added to the sweep table. Each verified CURRENT: row 18 (:262) — `FrameFn` byte-contract (full-wire-frame MUST; crc32/SplitHorizon rationale; reconstruction idiom) — no supersession, live requirement; row 19 (:511) — `Test shape` for `TestConnector_ReceiveLoop_CtlFrameForwardedToCallback` — sub-binding within row 8 block, live test specification; row 20 (:1812) — `Pin test shape` for `TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader` — Q9 §9.1a byte-contract pin, live test obligation; row 21 (:1928) — `Binding harness rule` for runRouter goroutine pattern (Q9.3/F-SP2-003) — live harness mandate. Grep transcript replaced with canonical pattern `grep -nE '\*\*[^*]*[Bb]inding'`; 21 hits enumerated and reconciled against extended table in v1.17 addendum. Sweep re-certified: 21 blocks total; zero unannotated stale binding blocks remain. This is the 8th incomplete-sweep-class instance and the 3rd false completeness certification. |
| P1a — Three-part F-SP20-001 annotation well-formed; 9 table dispositions from pass-20 audited TRUE | Pass-21 P1a confirmation | Clean. The v1.16 three-part annotation (header supersession marker, prose retraction, sketch banner) is well-formed and complete. All 9 pass-20 adjudication table rows verified unchanged and unaffected by the v1.17 sweep extension. |
| P1b — Story historiography clean: all sketches F-SP6-001-consistent, all amendment markers accurate | Pass-21 P1b confirmation | Clean. All sketches carry accurate version attribution and amendment markers. No new historiography inconsistency introduced by the v1.17 addendum (addendum is note-internal; no story obligation is altered). |
| P1c — Task 1–16 dry-run: no blocking contradictions. Task-8 RED-gate-ordering observation adjudicated | Pass-21 P1c confirmation | Clean. No blocking contradictions across Tasks 1–16. Task-8 observation: the task description does not explicitly require RED gate before step 4 (the `FrameTypePEConnect` discrimination logic). This is NOT a finding — under strict TDD the implementer writes the failing test (RED gate) before any implementation step; the order is enforced by TDD discipline, not by explicit step numbering. The assertion in question is a trivial pure-function comparison; grouping the test with other connector unit tests (step 4) rather than placing a separate RED-gate bullet before step 4 is intentional sequencing. PO MAY add "RED gate before step 4" language for consistency with Tasks 12 and 14 at a future natural edit opportunity; this is a style note, not a spec defect. |
| P1d — Notes-chain last-five audit clean | Pass-21 P1d confirmation | Clean. The last five pass-adjudicated sections (passes 15–20) are internally consistent; no ruling chain contradiction introduced by the v1.17 addendum. |
| P2 — POL-001 / POL-002 pass | Pass-21 P2 confirmation | Confirmed: no new canonical-source violation (POL-001) introduced by v1.17. POL-002 (incomplete-sweep): F-SP21-001 accepted and discharged — sweep extended to 21 blocks with canonical grep pattern and per-hit reconciliation; POL-002 is re-satisfied for the versioned-binding-block scope. |
| P3 — NoDuplicateSuppression + AC-005 lifecycle re-traced realizable | Pass-21 P3 confirmation | Confirmed. `TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader` (row 20 / Q9 §9.1a): two frames with differing `Envelope.SrcAddr` → independent crc32 keys → both reach `OnFrameArrival` independently → two E-FWD-001 emissions — realizable unchanged. AC-005 lifecycle: 44-byte malformed header → ErrInvalidFrameType → receive goroutine exits → `conn.Close()` → `maintainConn` write failure → reconnect — realizable unchanged. Neither test has any dependency on the v1.17 sweep extension. |
| Ledger passes 1–20 — all hold | Pass-21 ledger confirmation | Confirmed. No ruling in passes 1–20 is affected by the v1.17 note-only sweep-extension amendments. The entire pass-1 through pass-20 adjudicated ledger stands as published. |

---

## GREEN-phase adjudication (v1.18 — F-GP1-001, BINDING)

**Context:** During GREEN-phase TDD implementation of S-BL.PE-RECEIVE-LOOP, the implementer discovered that
the F-SP5-001/F-SP6-001 binding — `_ = conn.Close()` on ANY non-nil `frame.ReadOuterFrame` return — breaks
the pre-existing predecessor test `TestConnector_BackoffParameters` (S-7.04-FU-PE-CONNECTOR, merged) in a
deterministic, mechanism-traceable manner. This is a VSDD feedback-loop event: GREEN-phase implementation
discovery → architect adjudication. The adjudication below is binding.

### Empirical evidence

**Failure:** 3/3 deterministic. With the binding unconditional close implemented, `TestConnector_BackoffParameters`
reports post-reset retry gap 2.0–2.5 s; want [700 ms, 1300 ms].

**Mechanism (verified against ground truth at working tree HEAD):**

1. Server-side `conn.Close()` in the test's Phase 3 causes the PE upstream connection to receive EOF at the
   next `frame.ReadOuterFrame` call. The binding implementation calls `_ = conn.Close()` on this EOF return.

2. `maintainConn` (verified `connector.go` lines 484–486) on the next keepalive tick calls
   `conn.SetWriteDeadline(...)`. Because the conn is now closed (by the receive goroutine in step 1),
   `SetWriteDeadline` returns a non-nil error. The code at lines 484–486 reads:
   ```go
   if err := conn.SetWriteDeadline(time.Now().UTC().Add(c.keepaliveInterval)); err != nil {
       return   // SILENT return — no log, no "unreachable" emission
   }
   ```
   **This is a silent return** — no `logf("upstream router %s unreachable\n", addr)` is emitted.

3. The test's stamp-collection mechanism (lines 677–693) collects stamps matching
   `"upstream router %s unreachable"`. The `SetWriteDeadline`-failure exit emits no such stamp.
   Therefore the test's implicit assumption that stamp[0] is the write-failure log from
   `maintainConn` is violated: the connection teardown path that fires after unconditional close
   produces a silent `maintainConn` return, not a write-failure log.

4. Stamp[0] is consequently the first REDIAL failure (dial on the now-closed listener) rather than
   the write failure. All subsequent stamps are shifted: stamp[1] is the second dial failure, stamp[2]
   is the third. The measured gap `stamps[2].t - stamps[1].t` captures the DOUBLED backoff (~2 s,
   after the first retry's `nextBackoff` grows the base) rather than the operative-base backoff (~1 s).
   The test fails because the gap exceeds `hiWindow = 1300 ms`.

5. **Counter-check:** the implementer's carve-out (skip `conn.Close()` on clean `io.EOF` /
   `io.ErrUnexpectedEOF`) makes the test pass — but opens the TCP half-close hole: if the PE peer calls
   `CloseWrite()` (FIN in one direction only), the receive goroutine exits on `io.EOF` WITHOUT closing
   the conn, keepalive writes continue succeeding (peer ACKs the write channel), `maintainConn` never
   returns, the connection is permanently read-dead, and no reconnect occurs. This is the exact failure
   class F-SP6-001 was enacted to close.

### Options considered

**(a) AMEND the contract** — permit skip-close on clean `io.EOF`/`io.ErrUnexpectedEOF`.
Consequence: the TCP half-close hole is re-opened with no adequate substitute. The half-close
scenario is NOT out of contract: it occurs under real network conditions and with buggy or
asymmetrically-failing PE peers. F-SP6-001 explicitly addresses it. **REJECTED.**

**(b) ADJUST the predecessor test** — keep unconditional close; fix `TestConnector_BackoffParameters`
so its stamp-collection is robust to both the write-failure path and the silent-SetWriteDeadline-failure
path. The test's intent — "backoff resets to operative base after a successful connection" — is
unchanged; only the observability mechanism is made robust to teardown-path differences.

**(c) ALIGN maintainConn observability** — add the "unreachable" log line to the `SetWriteDeadline`
failure return path. Concern: logging "upstream router X unreachable" when `SetWriteDeadline` fails
on a conn we ourselves just closed is semantically misleading — the peer is not unreachable; we
initiated the close. This corrupts operator-visible EC-001 semantics and would embed a misleading
log in production code to satisfy a test's stamp-matching assumption. **REJECTED.**

### Decision: Option (b) — adjust the predecessor test

**Rationale:**

1. Option (b) is the only option that preserves BOTH the F-SP6-001 safety property (no permanently
   read-dead connections under any TCP teardown scenario including half-close) AND the predecessor
   test's behavioural intent (backoff resets to operative base after connection loss).

2. The test's intent is robust and correct. Its stamp-collection implementation is brittle — it
   depends on a specific `maintainConn` exit path (write failure) producing an EC-001 log, but the
   unconditional-close contract changes which exit path fires in the normal teardown-from-EOF case.
   Making the test robust to this is a legitimate test fix, not a semantic change.

3. Option (a) re-opens a documented, load-bearing safety property (F-SP6-001) with no adequate
   compensating mechanism. The half-close scenario is real and within the adversary model for a
   semi-trusted configured upstream. This note already records the risk at lines 719–784; the
   binding was enacted precisely to close it.

4. Option (c) introduces a semantically misleading production log line ("unreachable" on a
   locally-closed conn) that would corrupt EC-001 signal quality for operators. A test whose
   correctness depends on a misleading log would itself be a spec defect. Rejected on first
   principles.

### Half-close analysis (surviving the adversarial pass)

With unconditional `_ = conn.Close()` on any `frame.ReadOuterFrame` non-nil error:

- **Normal server-close EOF** (peer calls `Close()`): receive goroutine gets `io.EOF` → calls
  `_ = conn.Close()` (idempotent, the peer's FIN already closed the remote end) → `maintainConn`'s
  next `SetWriteDeadline` fails silently → `maintainConn` returns → dialLoop tears down → reconnect.
  SAFE: full teardown, reconnect within keepaliveInterval.

- **TCP half-close** (peer calls `CloseWrite()`, sends FIN in receive direction only): receive
  goroutine gets `io.EOF` → calls `_ = conn.Close()` → write channel is now closed from our side →
  `maintainConn`'s next keepalive `SetWriteDeadline` fails silently OR `conn.Write` fails → `maintainConn`
  returns → dialLoop tears down → reconnect. SAFE: half-close is converted to full teardown. Without
  `_ = conn.Close()`, keepalive writes would keep succeeding indefinitely (peer is still ACKing the
  write channel) and the connection would be permanently read-dead with no reconnect trigger.

- **Malformed frame without server-close** (ErrInvalidFrameType, ErrVersionMismatch): receive
  goroutine returns a parse error → calls `_ = conn.Close()` → `maintainConn` tears down → reconnect.
  SAFE: permanent framing-desync prevented.

The unconditional-close contract eliminates all three failure scenarios. There is no adversarially
survivable scenario where an exemption for `io.EOF` is safe.

### Implementation / test-writer directive (BINDING)

**Production code:** NO CHANGE to `connector.go`. The binding `_ = conn.Close()` on ANY non-nil
`frame.ReadOuterFrame` return (as already implemented in the working tree) MUST be preserved exactly.

**Test change — `internal/upstreamdial/connector_test.go`, `TestConnector_BackoffParameters`:**

The stamp-collection in Phase 3 (lines 668–706 at working tree HEAD) MUST be made robust to the
silent-SetWriteDeadline-failure teardown path. The fix is:

After the server-side `conn.Close()` at line 675 (and `ln2.Close()` at line 675), wait for the
connector to drop out of `ModePE` — meaning the unconditional-close-triggered teardown has
completed and the connector is now in the redial loop. Only then begin collecting stamps. Concretely:

1. After `_ = conn.Close()` and `_ = ln2.Close()` (the server-side drop), insert a `pollForNoModePE`
   call (or equivalent: poll `c.Mode() != ModePE` with a tight timeout, e.g. `2 * testKeepalive`).
   This synchronises the test to the teardown completion — regardless of whether the teardown was
   triggered by a write failure (old path) or a silent SetWriteDeadline failure (new path with
   unconditional close), Mode drops when `connectedCount.Add(-1)` returns 0.

2. After the Mode-drop poll succeeds, drain the stamp channel (as the existing `drainLoop` does
   after Phase 2). This discards any stamps from the teardown phase itself.

3. Collect 2 stamps from the redial phase (not 3). The gap between stamp[0] (first dial failure)
   and stamp[1] (second dial failure after operative-base backoff sleep) measures exactly the
   post-reset backoff delay. Assert `gap = stamp[1].t - stamp[0].t` in `[loWindow, hiWindow]`.

This change makes the test's observability mechanism invariant to teardown-path differences:
whether `maintainConn` exits via write failure (old path, with EC-001 stamp) or via silent
SetWriteDeadline failure (new path, no stamp), the Mode-drop poll correctly identifies
teardown completion before stamp collection begins. The measured gap is always the clean
operative-base backoff between redial attempt 1 and redial attempt 2.

**Story propagation requirements:** The story-writer MUST add a task (or extend Task 14 / Task 15)
to apply the above `TestConnector_BackoffParameters` fix as part of the GREEN-phase implementation
of S-BL.PE-RECEIVE-LOOP. This fix may be applied in the same commit as the receive-goroutine
implementation (since that implementation is what exposes the brittleness), or in a standalone
fixup commit before the story's PR is raised. The story's acceptance criteria, test count, and
all other implementation obligations are unchanged — this is a GREEN-phase pre-condition fix only.

---

## Per-story adversarial adjudication (v1.19 — F-IP1-001, BINDING)

**Finding:** F-IP1-001 (MED [missing regression guard + false enforcement claim]) — adversary implementation-phase pass-1.

### Four disk-verified legs

1. **Story :680 (AC-002 test descriptor) — promised, not delivered.** The test name block reads:
   `"...confirms no routing import in \`internal/upstreamdial\` via \`go list -deps\`"`.
   The delivered test `TestRunRouter_PE_FrameCallback_WiredToOnFrameArrival` (worktree
   `cmd/switchboard/router_pe_receive_test.go` :219–258) asserts ONLY `scanForLine("E-FWD-001")`.
   Zero `go list` / `go/packages` / depguard invocations exist anywhere in the test file or
   the worktree test corpus.

2. **Story :984 (Estimated Test Surface row) — claims verification that does not exist.**
   The row for `TestRunRouter_PE_FrameCallback_WiredToOnFrameArrival` reads:
   `"no routing import in \`internal/upstreamdial\` (\`go list -deps\` verified)"`.
   This is a false claim: no such verification is present in the delivered code.

3. **Story :907–909 (Architecture Compliance Rules) — factually wrong enforcement claim.**
   The sentence reads:
   > "Build-time violation: if `internal/upstreamdial` gains a `routing` import, the build
   > MUST fail (enforced by `ARCH-08 §6.6.2` and `go list -deps` verification in the
   > integration test)."
   This is wrong on two independent counts:
   - The edge `upstreamdial` (position 19) → `routing` (position 17) is **acyclic** (19 > 17).
     Go's toolchain rejects only cyclic imports. Adding `internal/routing` to `internal/upstreamdial`
     compiles cleanly; the build does NOT fail.
   - ARCH-08 §6.6.2 is a **documented perimeter constraint**, not a build-time enforcement
     mechanism. It establishes the forbidden-edge rule; it does not cause the compiler to reject
     the import.

4. **Current code is compliant — this is a missing regression guard, not a live violation.**
   `internal/upstreamdial/connector.go` imports `{frame, halfchannel, outerassembler}` only
   (verified at working-tree HEAD). The adversary's point stands: a hostile implementation
   that imports `routing` directly and bypasses the `FrameFn` callback seam would pass the
   entire delivered suite. The gap is the absence of a perimeter test, not a production defect.

### PART A ruling — enforcement mechanism

**Rejected options:**

**(a1 — inline in `TestRunRouter_PE_FrameCallback_WiredToOnFrameArrival`):** The story
promised this shape and the adversary proposed it. It is rejected on **single-concern test
design** grounds. `TestRunRouter_PE_FrameCallback_WiredToOnFrameArrival` already asserts
the functional wiring contract (SetFrameCallback → OnFrameArrival → E-FWD-001). Embedding
a structural import-perimeter check inside a functional wiring test conflates two independent
concerns: "does the callback seam work?" (behavioural) vs "does upstreamdial respect its
import perimeter?" (structural/architectural). These concerns have different failure modes,
different maintainers (functional tests break when behaviour changes; perimeter tests break
when the dependency graph changes), and different debugging contexts. Overloading one test
with both responsibilities produces a test that fails for two unrelated reasons and whose
failure message is ambiguous.

**Accepted option (a2 — standalone dependency test):** A dedicated test
`TestUpstreamdialImportPerimeter` in `internal/upstreamdial/connector_test.go` (the
package's own test file, in-package scope) using `os/exec` to invoke `go list -deps`.
This is the correct home: it is a structural property of the `upstreamdial` package, it
belongs alongside the package's own unit tests, and it runs under the same `go test ./internal/upstreamdial/...`
invocation that exercises the package's other obligations.

**Rejected option (a3 — depguard lint rule):** `.golangci.yml` in the worktree does not
enable `depguard` (verified — enabled linters: errcheck, govet, ineffassign, staticcheck,
unused, misspell, unconvert, unparam; depguard is absent). A lint-layer enforcement rule
that no CI gate executes is not enforcement. Adding depguard would require a non-trivial
`.golangci.yml` change with its own import-path rule syntax, and it would shift the
enforcement from test-time (visible in `go test` output) to lint-time (a separate CI step
that may not run in all contexts). The test-time path is more reliable and already
consistent with how the rest of the codebase pins invariants.

**Combination:** The ruling is **a2 only**. The AC-002 test descriptor in the story MUST
also be corrected (story-writer propagation) to remove the false `go list -deps` attribution
from `TestRunRouter_PE_FrameCallback_WiredToOnFrameArrival` and to point to the standalone
`TestUpstreamdialImportPerimeter` test as the perimeter's enforcement locus.

### PART B — corrected Architecture Compliance Rules wording

The sentence at story :907–909 MUST be replaced verbatim with:

> **ARCH-08 §6.6.2 import perimeter for `internal/upstreamdial`:** `drain`, `routing`,
> `testenv`, and packages at positions 20–23 MUST NOT be imported. The callback seam
> preserves this: `upstreamdial` imports `frame` (position 2) but not `routing` (position 17).
> Note: the `upstreamdial` → `routing` edge is acyclic (position 19 > 17); Go's toolchain
> does NOT reject it at build time. The perimeter is enforced by the architectural constraint
> in ARCH-08 §6.6.2 (documented forbidden-edge rule) and by the test-time regression guard
> `TestUpstreamdialImportPerimeter` in `internal/upstreamdial/connector_test.go`, which uses
> `go list -deps` to assert `internal/routing` is absent from the transitive dependency set.

The existing sentence "Build-time violation: if `internal/upstreamdial` gains a `routing`
import, the build MUST fail (enforced by `ARCH-08 §6.6.2` and `go list -deps` verification
in the integration test)" is RETRACTED IN FULL — it contains two false claims (acyclic build
does not fail; `go list` was in the integration test, not in a dedicated perimeter test) and
MUST NOT appear in the story's Architecture Compliance Rules section.

### Test recipe — `TestUpstreamdialImportPerimeter` (binding for test-writer)

**File:** `internal/upstreamdial/connector_test.go` (same file as `TestConnector_BackoffParameters`
et al. — in-package, no new file required).

**Test name:** `TestUpstreamdialImportPerimeter`

**Full recipe (binding):**

```go
func TestUpstreamdialImportPerimeter(t *testing.T) {
    // Exec go list -deps to obtain the transitive dependency set of
    // internal/upstreamdial and verify the routing-import forbidden edge
    // is absent. This is the test-time enforcement of ARCH-08 §6.6.2.
    cmd := exec.Command("go", "list", "-deps",
        "github.com/arcavenae/switchboard/internal/upstreamdial")
    out, err := cmd.Output()
    if err != nil {
        t.Fatalf("TestUpstreamdialImportPerimeter: go list -deps failed: %v", err)
    }
    deps := string(out)

    // Positive-coverage guard: the output MUST be non-empty AND contain a
    // known-present dependency (internal/frame) before we trust the absence
    // assertion. A broken exec or empty output would otherwise silently pass
    // the absence check.
    if len(deps) == 0 {
        t.Fatal("TestUpstreamdialImportPerimeter: go list -deps returned empty output")
    }
    if !strings.Contains(deps, "github.com/arcavenae/switchboard/internal/frame") {
        t.Fatalf("TestUpstreamdialImportPerimeter: positive-coverage guard failed — "+
            "internal/frame not found in deps (exec may be broken or working directory wrong);\n"+
            "got:\n%s", deps)
    }

    // Perimeter assertion: internal/routing MUST NOT appear.
    if strings.Contains(deps, "github.com/arcavenae/switchboard/internal/routing") {
        t.Errorf("TestUpstreamdialImportPerimeter: ARCH-08 §6.6.2 violation — "+
            "internal/routing is in the transitive deps of internal/upstreamdial;\n"+
            "full deps:\n%s", deps)
    }
}
```

**Notes for test-writer:**

- `exec.Command("go", "list", "-deps", ...)` uses the `go` binary already on PATH in every
  `go test` invocation — no additional tooling is required. The working directory is the
  module root (`go test` sets cwd to the package under test; `internal/upstreamdial` is within
  the module, so the module root is resolvable via the `go` tool automatically).
- The positive-coverage guard (`internal/frame` present) is MANDATORY — it prevents a broken
  exec from producing a false-green on the absence assertion. `internal/frame` is a direct
  import of `internal/upstreamdial` as of this story (verified: `connector.go` imports
  `github.com/arcavenae/switchboard/internal/frame` at HEAD).
- This test requires `"os/exec"` and `"strings"` imports in `connector_test.go`. Both are
  stdlib; no new external dependency is introduced.
- Test count impact: +1 unit test to `internal/upstreamdial/connector_test.go`. The story's
  connector test count (7 minimum + 1 optional = 7 or 8) rises by 1: **8 minimum** (+ optional
  `ExitsOnVersionMismatch` = 8). The total net-new story test count rises from ~12 to **~13**.
  Story-writer MUST update the Estimated Test Surface table and the test-count summaries.

### Disposition of adversary's forward-looking observation

The adversary identified `mgmt_wire.go` lines 549–551 (the `FrameFn` closure in `runRouter`)
as passing a nil `ForwardFunc` to `arrivalHandler.OnFrameArrival`. Ground truth: at working-tree
HEAD, `runRouter` wires a single-interface set containing only `peIfaceID` (the PE arrival
interface). `OnFrameArrival` calls `SplitHorizon.Forward`, which discovers all paths are
split-horizon blocked before it ever attempts to invoke `fn`. The nil `ForwardFunc` is therefore
never called on the single-interface set; no nil-deref occurs; this is **correct for the
single-interface set that this story constructs**.

**This is NOT a defect in the current story.** Q8 §8.2 explicitly notes: "In production, the
interface set is populated from the router's forwarding table or a registry of connected
data-plane nodes." The nil `ForwardFunc` is a known placeholder appropriate for the
single-interface exhaustion case this story exercises (guaranteed split-horizon block →
`fn` never called).

**Forward obligation recorded (not a current defect):** When a future story (in the
S-7.04-FU-DRAIN-WIRE / session-bootstrap era) grows the PE interface set beyond one entry —
i.e., adds a second `InterfaceID` to the `interfaceSet` slice so that `SplitHorizon.Forward`
has a non-blocked candidate path and actually invokes `fn` — the nil `ForwardFunc` MUST be
replaced with a real forwarding implementation before that story ships. A nil `fn` call on a
non-blocked path produces a nil-pointer dereference in production. The story that widens the
interface set owns this obligation. Story-writer of that future story: check this note.

This observation does not create any obligation for the current story's implementer, test-writer,
or story-writer beyond the forward-obligation documentation above.

---

## Per-story adversarial adjudication round 2 (v1.20 — F-IP2-001/002/003, BINDING)

**Source:** Implementation-phase adversarial pass-2 against S-BL.PE-RECEIVE-LOOP at commit e397157 (story v1.22, note v1.19). Disk-verified by orchestrator prior to dispatch.

---

### F-IP2-001 (MED) — SetFrameCallback post-Start mutation guard unimplemented

**Finding restated:** The story (:491–494) and note (:195–198) both carry the binding clause: "The Connector implementation MUST NOT permit [post-Start mutation via SetFrameCallback] — it may panic or silently ignore the call, but MUST NOT proceed with an unsynchronized field write." The delivered `SetFrameCallback` (worktree `connector.go:228–230`) is an unconditional unguarded write: `c.frameFn = fn`. No `started`/`startOnce` atomic field exists in the struct. All current callers obey the pre-Start ordering, so the race detector never fires and no test exercises the forbidden path.

**RULING: OPTION (b) — caller-ordering contract alone sufficient; downgrade the spec clause.**

The note and story MUST be updated (story-writer propagation) to replace the implementation obligation with a caller obligation only:

> `SetFrameCallback` MUST be called before `Start()`. Calling it after `Start()` returns is a **data race** (dial goroutines are already reading `frameFn`); the caller is solely responsible for the ordering. The implementation does not detect or guard against post-Start mutation — the field is set-once and the goroutine-creation happens-before already covers visibility to all goroutines launched by `Start()`.

**Rejected option — (a) implement the guard:**

(a1) **Panic on post-Start call.** An atomic `started int32` flag (or reuse of `startOnce`) set at the top of `Start()` before goroutine launch; `SetFrameCallback` reads it and panics if set. Advantages: the programming error surfaces loudly and immediately; consistent with Go's nil-pointer-panic philosophy for misuse of concrete types. Disadvantages: the guard is itself a concurrent read of `started` against a write in `Start()` — the two operations are not synchronised by the same mechanism. Setting `started=1` in `Start()` before goroutine launch does NOT give `SetFrameCallback` a happens-before guarantee on `started`; a concurrent `SetFrameCallback` call could race the `started` store and miss it, defeating the guard. The only race-safe implementation would use `sync/atomic`, which costs a new field and an acquire/release load-store pair. A panic in a production goroutine (if `SetFrameCallback` were ever called from one) would crash the process; for a guard on programmer misuse the cost/benefit is poor.

(a2) **Silent ignore on post-Start call.** Same field, same race risk on the guard itself, but instead of panicking the call is a no-op. This matches the nil-guard silent-discard philosophy cited in the finding but has a worse diagnostic profile: a misuse that silently does nothing produces exactly the symptom described in the spec — "an unsynchronized field write" — without the programmer ever knowing. The nil-guard precedent (`:485–489`) applies to a nil *callback*, not to a race on the *callback field itself*. The analogy does not hold.

**Why (b) is correct here:**

1. **One production caller, correct ordering.** `runRouter` in `mgmt_wire.go` is the sole production caller (verified grep). It calls `New()` → `SetFrameCallback()` → `Start()` in strict sequence. The goroutine-creation happens-before guarantee in Go's memory model makes `frameFn` visible to all goroutines launched by `Start()` without any additional synchronization. This is already documented in the `frameFn` field comment and in v1.4's Q2/Q8 amendment. The guard protects against a caller pattern that does not exist and cannot occur via the concrete-type-only API.

2. **The guard is not race-safe without a new synchronisation primitive.** Any `started` flag readable by `SetFrameCallback` and writable by `Start()` is itself a potential data race unless sequenced by a mutex or an atomic with the appropriate memory ordering. Adding this complexity to protect against a call that cannot legally occur through the `*Connector` public surface is cost without benefit.

3. **The spec clause survived 24 passes because it was aspirationally correct, not because it adds observable safety.** The spec was written before the single-production-caller constraint was fully elaborated. Re-reading it now: the binding obligation was to not proceed with an unsynchronized write. The existing implementation is not unsafe because the race never occurs; the caller contract is the correct layer at which to enforce it.

4. **No TDD obligation flows from this ruling.** There is no failing test to write, because (b) removes the implementation obligation. Story-writer MUST update the clause wording in the story; no test-writer or implementer action required for this finding.

---

### F-IP2-002 (MED) — Residual false attribution in `router_pe_receive_test.go:212–217` doc comment

**Finding restated:** The doc comment for `TestRunRouter_PE_FrameCallback_WiredToOnFrameArrival` (worktree `cmd/switchboard/router_pe_receive_test.go:212–217`) includes the bullet:

```
//   - No routing import in internal/upstreamdial (ARCH-08 §6.6.2 preserved).
```

The F-IP1-001 remediation (v1.19) corrected the story prose but missed this sibling comment in the test file. The test asserts only `E-FWD-001`; the import perimeter is enforced by `TestUpstreamdialImportPerimeter` (ruled in v1.19) in a separate file. The attribution is false.

**RULING: mechanical replacement; test-writer applies.**

The bullet at `router_pe_receive_test.go:215–216` (the line reading `//   - No routing import in internal/upstreamdial (ARCH-08 §6.6.2 preserved).`) MUST be replaced verbatim with:

```
//   - Import perimeter enforced separately by TestUpstreamdialImportPerimeter (internal/upstreamdial/connector_test.go, F-IP1-001).
```

The replacement is a single-line, single-bullet substitution. No other changes to the doc comment or the test body are required. Story-writer notes this as a propagation item; test-writer applies it in the same commit as `TestUpstreamdialImportPerimeter`.

---

### F-IP2-003 (LOW) — ARCH-08 dual-changelog drift

**Finding restated:** `ARCH-08-dependency-graph.md` carries `version: "2.11"` in its frontmatter and a `modified:` entry for v2.11 (line 10), but the `## Changelog` table's newest row was `2.10`. Every prior version 2.4–2.10 appeared in both frontmatter history and Changelog table. The v2.11 row was missing from the table (POL-001 parity violation).

**Ruling: fixed in-place in ARCH-08 (no version bump — this completes the v2.11 edit).**

The `| 2.11 | ... |` row has been added to the ARCH-08 Changelog table immediately after the `2.10` row. Content derived from the `modified:` frontmatter entry at line 10:

> `| 2.11 | 2026-07-11 | S-BL.PE-RECEIVE-LOOP: §6.5 pos-19 import set updated `{halfchannel, outerassembler}` → `{frame, halfchannel, outerassembler}` (direct frame edge re-added per F-SP12-001); §6.5 parenthetical reconciled (historical F-P1-001 "frame not imported directly" note superseded with preservation of context); §6.6.2 upstreamdial forbidden-edges bullet updated at positions 2, 5, 8 per F-SP13-001 (cycle-freeness gains frame pos 2; F-P1-001 clause updated; F-P7-002 clause preserved). Refs: S-BL.PE-RECEIVE-LOOP + F-SP12-001/F-SP13-001 + code commit c316aed. |`

ARCH-08 frontmatter (`version: "2.11"`, `modified:` line 10) is **untouched** — this is completion of the v2.11 edit, not a new version.

---

## Per-story adversarial adjudication round 3 (v1.21 — F-IP3-001 + observations, BINDING)

**Source:** Implementation-phase adversarial pass-3 against S-BL.PE-RECEIVE-LOOP. Disk-verified by orchestrator prior to dispatch.

---

### F-IP3-001 (MED) — note-side F-IP2-001 Option-b propagation not performed in v1.20

**Finding restated:** The v1.20 ruling (F-IP2-001, :3104–3124) mandated: "The note and story MUST be updated … to replace the implementation obligation with a caller obligation only." The story was updated to v1.23 (:491–498). The note's Q1 "SetFrameCallback Ordering Contract" block at :194–199 was NOT updated — it still carried verbatim: "the `Connector` implementation MUST NOT permit it. If the `SetFrameCallback` setter is called post-Start, it may panic or be silently ignored — the implementer's choice — but it MUST NOT proceed with an unsynchronized field write." No supersession marker was present. A top-to-bottom reader hitting this block ~2900 lines before the F-IP2-001 ruling would wrongly conclude the delivered unguarded setter violates the spec.

**Annotation applied (v1.21):** The block at :194–199 has been struck with `~~strikethrough~~` and annotated inline with the Option-b caller-obligation wording from :3108:

> ~~Post-Start mutation of the callback is forbidden. Any call to `SetFrameCallback` after `Start()` returns is a data race (dial goroutines are already reading `frameFn`); the `Connector` implementation MUST NOT permit it. If the `SetFrameCallback` setter is called post-Start, it may panic or be silently ignored — the implementer's choice — but it MUST NOT proceed with an unsynchronized field write.~~ *(amended v1.21 — F-IP3-001: note-side F-IP2-001 Option-b propagation, mandated at :3106 but not performed in v1.20. The implementation obligation is replaced by a caller obligation only:* `SetFrameCallback` *MUST be called before* `Start()`*. Calling it after* `Start()` *returns is a **data race** (dial goroutines are already reading* `frameFn`*); the caller is solely responsible for the ordering. The implementation does not detect or guard against post-Start mutation — the field is set-once and the goroutine-creation happens-before already covers visibility to all goroutines launched by* `Start()`*. See the F-IP2-001 ruling at :3104 for full rationale.)*

**Class-closure sweep (mandatory — 9th incomplete-sweep-class instance):**

The following patterns were applied to the ENTIRE note body using multi-line-tolerant technique (`cat <file> | tr '\n' ' ' | grep -o "<pattern>"`) to find any OTHER location asserting the pre-Option-b implementation obligation outside changelog rows and the F-IP2-001/F-IP3-001 adjudication sections:

| # | Pattern | Hits | Disposition |
|---|---------|------|-------------|
| 1 | `MUST NOT permit` | 2 | :196 — TARGET, struck above. :3102 — F-IP2-001 finding-restated historical text, acceptable. |
| 2 | `panic or be silently ignored` (multi-line: `panic or` across line break at :197–198) | 2 | :197–198 — TARGET, struck above (same block as hit 1). :3102 — F-IP2-001 finding-restated historical text, acceptable. |
| 3 | `unsynchronized field write` | 2 | :198–199 — TARGET, struck above (same block). :3102 — F-IP2-001 finding-restated historical text, acceptable. |
| 4 | `implementer's choice` | 2 | :198 — TARGET, struck above (same block). :842 — unrelated ruling ("netingress.ReadFrame may delegate to it … the implementer's choice"), not the post-Start obligation. |

**Sweep result: zero additional live unannotated occurrences of the pre-Option-b implementation obligation outside the already-annotated :194–199 block and acceptable historical/adjudication restatements.** Sweep certified over the full note body with multi-line-tolerant patterns.

**This is the 9th incomplete-sweep-class instance** (F-SP19/20/21 were 6th/7th/8th). Root cause: the F-IP2-001 ruling at :3106 said "note and story MUST be updated" but did not name a specific line range; the story-writer update path was executed; the note-side path was not. The OBS-2 countermeasure below addresses the process gap.

---

### OBS-1 (LOW [test-coverage]) — FlapCycleJoin_NoLeak does not pin recvWg.Wait() against deletion

**Observation restated:** Every lifecycle test in `connector_test.go` closes the connection before asserting goroutine cleanup. When the conn is closed, the receive goroutine self-exits via EOF (the `io.ReadFull` call inside `frame.ReadOuterFrame` returns `io.EOF`). Because the goroutine exits on its own, `recvWg.Wait()` in `dialLoop` returns promptly regardless of whether the WaitGroup increment/Done pair are present or not — a malicious implementation that omits `recvWg.Add(1)` / `recvWg.Done()` entirely still passes all existing tests, because the conn-close guarantees the goroutine exits before `Stop()` completes the join. The join is required for correctness on the reconnect path (Q6 per-reconnect join, :1453–1462), but no test forces the code path where `recvWg.Wait()` is load-bearing.

**RULING: (i) ACCEPT as documented pin-limitation.**

**Rationale:**

1. **The join is present and required-for-correctness.** The Q6 per-reconnect-iteration join contract (:1453–1462) is binding and verified in implementation review. The per-connection WaitGroup (or equivalent `done chan struct{}`) is specified in the note; its presence is not in dispute.

2. **A deterministic pin would require holding the conn open through the WaitGroup wait.** The test harness would need to: (a) accept a connection on the server side, (b) NOT close it, (c) let the receive goroutine block inside `io.ReadFull` on the open conn, (d) trigger a reconnect from the `dialLoop` side (e.g., via `Stop()` or `ReloadAddrs([])`), (e) assert `Stop()` blocks until the receive goroutine exits, (f) then allow the conn to close. The specific concern is step (d)/(e): `Stop()` closes `stopCh`, which causes `reconcileLoop` to cancel the dial contexts. The per-address `addrCancel.done` channel is closed only after both `maintainConn` AND the receive goroutine have returned (:1484–1492). If the receive goroutine is blocked in `io.ReadFull` on an open conn, and the dial context is cancelled, the goroutine may not exit until the conn is closed externally — which means the test must close the conn from the server side to unblock the goroutine, immediately creating the same "conn-close causes self-exit" situation the pin was meant to avoid.

3. **The resulting test cannot distinguish "goroutine exited because WaitGroup.Wait() blocked Stop()" from "goroutine exited because conn-close unblocked io.ReadFull".** Any observable pin (timeout on `Stop()`, goroutine count assertion) would also pass if the WaitGroup were absent, because the conn-close always unblocks the goroutine before the timeout. A done-channel observable (`make(chan struct{})`, closed by the receive goroutine) could track goroutine exit independently, but cannot force the goroutine to stay alive long enough to require the WaitGroup wait — the only thing keeping the goroutine alive is an open conn with data to read, and closing that conn is the only way to let `Stop()` proceed.

4. **Risk assessment: LOW.** The WaitGroup is present in the implemented code (verified in adversarial pass-3); a future refactor that deletes it would require explicitly removing `recvWg.Add(1)` / `recvWg.Done()` calls — this is not an accidental omission. The flap-cycle test does validate the reconnect path and goroutine-leak assertion via the `runtime.NumGoroutine()` before/after comparison (connector_test.go:1858/:1924 — the "or equivalent" arm of the Q6 recipe; goleak is not imported). The specific WaitGroup-dependency path is not independently pinned, but the goroutine-leak gate provides a net that catches the failure mode the WaitGroup prevents (accumulated goroutines from rapid flap cycles).

**No test-writer or implementer action required from this ruling.**

---

### OBS-2 (LOW [process-gap]) — remediation workflow missing mandatory in-place annotation step

**Observation restated:** The F-IP2-001 ruling mandated a note-side propagation but did not include a step requiring the architect to annotate the cited stale line-range in-place in the same remediation burst. The propagation was deferred to story-writer scope, and the note-side annotation was never performed. This is the 9th instance of the incomplete-sweep class across this story's adversarial history (F-SP19/F-SP20/F-SP21 were 6th/7th/8th; F-IP3-001 is the 9th).

**[process-gap]** Recorded per the cycle-closing checklist (S-7.02). The orchestrator will route the codification follow-up at cycle close.

**Countermeasure now binding for remaining passes of S-BL.PE-RECEIVE-LOOP:**

Any future finding by any adversarial pass that cites a note line-range as stale or as carrying superseded binding text **REQUIRES** the in-place annotation of that exact line-range in the same remediation burst as the ruling, before the adjudication section is appended. The annotation follows the established class-closure pattern: ~~strikethrough~~ the stale text + `*(amended vN.NN — FXXX: <brief rationale>)*` inline marker. The ruling section MUST then record the verbatim new annotation text (as done for F-IP3-001 above) so the adversary can verify in the next pass. A ruling that says "note-side propagation required" without performing the annotation in the same burst is procedurally incomplete and MUST be treated as an open finding by the next adversarial pass.

---

## Per-story adversarial adjudication round 4 (v1.22 — F-IP4-001 + checkbox observation, BINDING)

**Source:** Implementation-phase adversarial pass-4 against S-BL.PE-RECEIVE-LOOP at commit c3fca02. Disk-verified by orchestrator prior to dispatch.

---

### F-IP4-001 (MED) — outgoing bootstrap FrameTypePEConnect unpinned

**Finding restated:** AC-003 PC-3 / Task 11 / FO-PE-LOOP-001 mandate the dialLoop bootstrap `ChannelFrame.FrameType` flip from `halfchannel.FrameTypeData` to `frame.FrameTypePEConnect`. The worktree delivers this correctly at `connector.go:359–361` (verified at pass-4): `cf := halfchannel.ChannelFrame{ FrameType: frame.FrameTypePEConnect }`. However, no test parses the connector's OUTGOING bootstrap frame and asserts `frame_type == 0x06`. Every existing fixture drains the first write unparsed: `TestConnector_KeepaliveTickerDrivesHealthProbe` reads it at `connector_test.go:811–815` but checks only `n > 0`. Reverting `:360` to `halfchannel.FrameTypeData` leaves the full suite green. This is the same test-set-underdetermination class as F-SP17-001 and F-SP18-001, both of which were remediated on the receive side in-story. The property is behaviorally inert within this story (all in-story consumers drain the bootstrap frame without inspecting its `frame_type`), but it is a named deliverable of FO-PE-LOOP-001, consumed by the S-7.04-FU-DRAIN-WIRE / session-bootstrap era — a silent revert would surface far downstream from its origin.

**RULING: OPTION (a) — REMEDIATE NOW with a pin test.**

**Rejected option — (b) defer to consuming story with recorded forward obligation:**

Deferral is structurally available (note already carries the FO-PE-LOOP-001 forward-obligation pattern, see the nil-ForwardFunc obligation at round-3 boundary). However, the forward obligation here differs materially from the nil-ForwardFunc case. The nil-ForwardFunc obligation defers a NEW capability (a real forwarding implementation) to a story that expands the interface set — deferral is appropriate because the capability does not yet exist and cannot be tested in isolation. The bootstrap frame_type flip is different: the capability EXISTS in the current commit, is testable now with a trivial in-package fixture, and represents a named invariant of the current story's own acceptance criteria (AC-003 PC-3 "bootstrap frame discriminated from data frames by type"). Deferring a test for an AC of the current story to a future story leaves the AC's implementation revert-able for the entire gap between merge and DRAIN-WIRE delivery — a gap that could span multiple sprints. The F-SP17-001 and F-SP18-001 precedent establishes that MED underdetermination findings of this class are remediated in-story; there is no basis for treating the outgoing-direction hole differently from the receive-side holes that were remediated immediately. Remediation cost is minimal (one test, one accept-and-read fixture). The reject-and-defer cost is high: an undetectable revert of a named deliverable.

**Test recipe (binding):**

**Test name:** `TestConnector_BootstrapFrameTypePEConnect`

**Rationale for new test vs extending existing:** `TestConnector_KeepaliveTickerDrivesHealthProbe` already accepts one connection and reads the bootstrap frame, but its fixture-goroutine reads via `conn.Read(buf)` and checks only `n > 0`; the fixture contract is "drain and proceed." Reusing it would require restructuring that fixture's error path and adding a blocking assertion on a parsed field — two incompatible concerns in one test body. A dedicated test is cleaner, follows the single-concern test design rationale established by F-IP1-001 (separate functional-wiring from perimeter-checking), and is the pattern used for every other pin test in this story (`ExitsOnReadError`, `ExitsOnVersionMismatch`, `CtlFrameForwardedToCallback`). A NEW dedicated test is ruled; do NOT extend `TestConnector_KeepaliveTickerDrivesHealthProbe`.

**Fixture pattern:** in-package accept-and-read, per `connector_test.go` conventions. Use `newLoopbackListener(t)` to obtain a loopback listener. Accept one connection in a goroutine (or use a buffered channel for the accepted conn). The test starts the connector with `New(lw, zeroEnv(), testKeepalive, []string{addr})` + `c.Start()` + `t.Cleanup(c.Stop)`. The fixture accepts the dialed connection.

**Exact assertions (in order):**

1. `io.ReadFull(conn, buf[:frame.OuterHeaderSize])` — read exactly `frame.OuterHeaderSize` (44) bytes from the accepted connection into a local `[frame.OuterHeaderSize]byte` buffer (or equivalent `[]byte`).
2. **Positive guard:** assert that `io.ReadFull` returned `nil` error and `n == frame.OuterHeaderSize`. The assertion `n == frame.OuterHeaderSize` is the positive guard that the read actually completed (guards against a broken fixture that returns 0 bytes and a nil error — impossible with `io.ReadFull` semantics, but documents intent).
3. `hdr, err := frame.ParseOuterHeader(buf[:frame.OuterHeaderSize])` — parse the 44-byte header.
4. Assert `err == nil` (the bootstrap frame MUST be a valid outer header; a parse error here indicates a wire-format regression, not just a frame_type regression).
5. Assert `hdr.FrameType == frame.FrameTypePEConnect` — the primary pin assertion.

**Timeout / establishment wait:** Use `pollForMode(c, 2*time.Second)` before reading the bootstrap frame to ensure the connector has dialed. Alternatively, wait on the fixture's accepted-conn channel with a 2s deadline. Either pattern is consistent with existing `connector_test.go` conventions.

**Test-surface table impact:**

- `internal/upstreamdial/connector_test.go` connector tests: **8 → 9** (adding `TestConnector_BootstrapFrameTypePEConnect`)
- Total net-new estimated: **~13 → ~14** (1 `frame_test` + 9 `connector_test` + 4 integration)

**Story-writer propagation items (for story bump v1.25):**

1. FCL row 5 (`connector_test.go`): append `TestConnector_BootstrapFrameTypePEConnect` to the Change cell (F-IP4-001 bootstrap frame_type pin); append `F-IP4-001` to the Anchor cell. Update the test count 8 → 9.
2. Estimated Test Surface connector table: add row for `TestConnector_BootstrapFrameTypePEConnect` (unit, `connector_test.go` — accepts dialed conn; `io.ReadFull` 44 bytes; `frame.ParseOuterHeader`; asserts `hdr.FrameType == frame.FrameTypePEConnect`; kills `halfchannel.FrameTypeData` regression; F-IP4-001).
3. File Structure Requirements `connector_test.go` count: 8 → 9 (this test added).
4. Estimated new test count: `~13 → ~14` net-new.
5. Task 11 (`[ ] Flip dialLoop bootstrap FrameType…`): append `; **[v1.22 F-IP4-001]** pin test `TestConnector_BootstrapFrameTypePEConnect` added to verify the flip is observable on the wire`.
6. Add Task 21 for this pin test (marked `[x]` if delivered with this bump, `[ ]` if test-writer task).
7. Checkbox hygiene (see observation below): mark Tasks 1–16 `[x]`.

---

### Checkbox observation (disk-verified)

**Observation restated:** Story Tasks 1–16 (`:1018–:1033`) are marked `[ ]` while Tasks 17–20 are `[x]`, yet all Tasks 1–16 deliverables are verifiably complete at `c3fca02` (pass-4 audited each; delivering commits: `c316aed` stubs, `a3d5117` RED, `e85c9df` / `8e8296c` / `5274cf1` GREEN, spec-side `9792605`).

**RULING: Mark Tasks 1–16 `[x]` in story bump v1.25** citing the delivering commits listed above. No deliberate convention for leaving complete tasks unchecked has been identified in this story's history; the earlier `[x]` tasks (17–20) demonstrate the convention IS to mark complete tasks checked. The simple correct fix is to apply consistent `[x]` marking. The story-writer owns this in the v1.25 bump alongside the F-IP4-001 propagation items above.
