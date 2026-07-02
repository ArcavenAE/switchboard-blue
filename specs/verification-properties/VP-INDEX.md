---
artifact_id: VP-INDEX
document_type: verification-property-index
level: L4
version: "2.34"
status: draft
producer: product-owner
timestamp: 2026-07-02T00:00:00
phase: 1b
inputDocuments:
  - '.factory/specs/behavioral-contracts/BC-INDEX.md'
  - '.factory/specs/architecture/ARCH-07-verification-architecture.md'
  - '.factory/specs/architecture/ARCH-11-verification-coverage-matrix.md'
traces_to: '.factory/specs/architecture/ARCH-INDEX.md'
---

# VP-INDEX: Verification Properties

> Master index. One row per VP. Full contracts in individual VP-NNN files.
> Source of truth for VP catalog, IDs, modules, tools, phases, and counts.
> VP-INDEX total MUST equal: sum of per-tool counts = VP row count.

## Index

| VP ID | Title | BC(s) | Module | Method | Phase | Status | File |
|-------|-------|-------|--------|--------|-------|--------|------|
| VP-001 | ParseOuterHeader / EncodeOuterHeader round-trip | BC-2.01.004 | internal/frame | proptest | P0 | draft | VP-001.md |
| VP-002 | ParseOuterHeader rejects version mismatch | BC-2.01.004 | internal/frame | proptest | P0 | draft | VP-002.md |
| VP-003 | EncodeOuterHeader produces exactly 44 bytes | BC-2.01.004 | internal/frame | proptest | P0 | draft | VP-003.md |
| VP-004 | ComputeHMAC / VerifyHMAC consistency | BC-2.05.005 | internal/hmac | proptest | P0 | draft | VP-004.md |
| VP-005 | VerifyHMAC rejects single-bit flips | BC-2.05.005 | internal/hmac | fuzz | P0 | draft | VP-005.md |
| VP-006 | VerifyHMAC rejects wrong key | BC-2.05.005 | internal/hmac | proptest | P0 | draft | VP-006.md |
| VP-007 | Admission: private key never in wire structs | BC-2.05.001, BC-2.05.007 | internal/admission | proptest | P0 | implemented | VP-007.md |
| VP-008 | Admission fails for unregistered key | BC-2.05.001, BC-2.05.002 | internal/admission | proptest | P0 | implemented | VP-008.md |
| VP-009 | Admission rejects replayed nonce | BC-2.05.001 | internal/admission | proptest | P0 | implemented | VP-009.md |
| VP-010 | SVTNRoute never delivers to wrong SVTN | BC-2.05.006 | internal/routing | proptest | P0 | implemented | VP-010.md |
| VP-011 | Split-horizon: no forward toward arrival interface | BC-2.02.008 | internal/routing | proptest | P0 | draft | VP-011.md |
| VP-012 | SessionAuth rejects unauthorized console key | BC-2.05.003, BC-2.04.003 | internal/session | proptest | P0 | draft | VP-012.md |
| VP-013 | SessionAuth rejects upstream from read-only key | BC-2.04.005, BC-2.05.003 | internal/session | proptest | P0 | draft | VP-013.md |
| VP-014 | DeriveNodeAddress is deterministic | BC-2.01.006 | internal/frame | proptest | P0 | draft | VP-014.md |
| VP-015 | Router code never parses channel header payload | BC-2.01.005 | internal/routing | fuzz | P0 | draft | VP-015.md |
| VP-016 | HalfChannel.Tick emits exactly one frame per tick | BC-2.01.001, BC-2.01.003 | internal/halfchannel | proptest | P0 | draft | VP-016.md |
| VP-017 | HalfChannel sequence increments by exactly 1 | BC-2.01.003 | internal/halfchannel | proptest | P0 | draft | VP-017.md |
| VP-018 | HalfChannel emits empty frame when no payload | BC-2.01.001, BC-2.01.002 | internal/halfchannel | proptest | P0 | draft | VP-018.md |
| VP-019 | ARQ.OnAck never delivers a frame twice | BC-2.02.005 | internal/arq | proptest | P0 | draft | VP-019.md |
| VP-020 | ARQ delivers frames in-order | BC-2.02.005 | internal/arq | proptest | P0 | draft | VP-020.md |
| VP-021 | ARQ.TLPKTDROP triggers only when frame overdue | BC-2.02.006 | internal/arq | proptest | P0 | draft | VP-021.md |
| VP-022 | Replay.OnUpstream never delivers same seq twice | BC-2.02.004 | internal/replay | proptest | P0 | draft | VP-022.md |
| VP-023 | Replay delivers keystrokes in sequence order | BC-2.02.004 | internal/replay | proptest | P0 | draft | VP-023.md |
| VP-024 | Multipath delivers first copy, discards duplicates | BC-2.02.001, BC-2.02.002 | internal/multipath | proptest | P0 | draft | VP-024.md |
| VP-025 | DropCache never exceeds configured capacity | BC-2.02.009 | internal/multipath | proptest | P0 | draft | VP-025.md |
| VP-026 | PathScore ranking is transitive | BC-2.02.003 | internal/paths | proptest | P0 | draft | VP-026.md |
| VP-027 | QualityIndicator transitions: degradation only goes down | BC-2.06.001, BC-2.06.002 | internal/metrics | proptest | P1 | draft | VP-027.md |
| VP-028 | Config.Validate rejects out-of-range tick_interval | BC-2.09.003 | internal/config | proptest | P0 | draft | VP-028.md |
| VP-029 | Config.Validate rejects missing required fields | BC-2.09.003 | internal/config | proptest | P0 | draft | VP-029.md |
| VP-030 | sbctl exits 1 with E-NET-001 on connection refused | BC-2.07.003 | cmd/sbctl | integration | P0 | draft | VP-030.md |
| VP-031 | tmux control mode: 99% output event completeness | BC-2.04.001 | internal/tmux | integration | P1 | draft | VP-031.md |
| VP-032 | PTY fallback activates on control mode failure | BC-2.04.002 | internal/tmux | integration | P0 | draft | VP-032.md |
| VP-033 | Console attach/detach lifecycle | BC-2.04.003, BC-2.04.004 | internal/session | e2e | P1 | draft | VP-033.md |
| VP-034 | Multi-console fan-out: both consoles receive all frames | BC-2.04.006 | internal/session | e2e | P0 | draft | VP-034.md |
| VP-035 | Read-only console: upstream rejected by access node | BC-2.04.005 | internal/session | integration | P1 | draft | VP-035.md |
| VP-036 | Session continuity across IP address change | BC-2.01.007 | internal/admission | e2e | P0 | deferred | VP-036.md |
| VP-037 | Router drain: nodes migrate within 2s | BC-2.09.002 | internal/drain | e2e | P2 | draft | VP-037.md |
| VP-038 | E→PE graduation: config change only | BC-2.09.001 | internal/config | e2e | P2 | draft | VP-038.md |
| VP-039 | SVTN isolation: no cross-SVTN frame delivery | BC-2.05.006 | internal/routing | e2e | P0 | deferred | VP-039.md |
| VP-040 | Multipath failover: recovery < 2s | BC-2.02.003 | internal/multipath | e2e | P1 | draft | VP-040.md |
| VP-041 | Tick regularity: p99 jitter ≤ 2ms | BC-2.01.001 | internal/halfchannel | benchmark | P0 | draft | VP-041.md |
| VP-042 | Keystroke-to-echo: p99 ≤ 100ms | BC-2.01.001, BC-2.02.001 | internal/halfchannel | benchmark | P0 | draft | VP-042.md |
| VP-043 | XOR FEC: single loss in group recoverable (implementing_story: S-7.01) | BC-2.02.007 | internal/arq | proptest | P1 | draft | VP-043.md |
| VP-044 | Presence advertisement includes required fields (implementing_story: S-7.02; coverage partial — in-process registry seam only; multicast wire deferred to S-BL.DISCOVERY-WIRE per RULING-W6TB-D) | BC-2.03.001, BC-2.03.003 | internal/discovery | integration | P1 | draft | VP-044.md |
| VP-045 | Console session enumeration without hostnames (implementing_story: S-7.02) | BC-2.03.002 | internal/discovery | e2e | P1 | draft | VP-045.md |
| VP-046 | Key lifecycle: register/revoke/expire | BC-2.05.004 | internal/svtnmgmt | integration | P1 | draft | VP-046.md |
| VP-047 | Per-path metrics queryable via sbctl | BC-2.06.003 | internal/metrics | integration | P1 | draft | VP-047.md |
| VP-048 | Control node creates/destroys SVTNs (PC-1 create + PC-2 bootstrap: S-6.02; PC-3 destroy: S-6.05; handler+CLI RPC-reachable: S-6.07; Ruling-7 defense-in-depth RoleControl mutation-test: S-6.07) (v1.8: source_bc pin synced to BC-2.07.001 v1.12) | BC-2.07.001 v1.12 | internal/svtnmgmt | integration | P2 | draft | VP-048.md |
| VP-049 | sbctl unified CLI with OpenSSH auth (implementing_story: S-W5.02) | BC-2.07.002 | cmd/sbctl | e2e | P2 | draft | VP-049.md |
| VP-050 | Console remotely controllable via sbctl (implementing_story: S-7.03) | BC-2.08.001 | cmd/sbctl | e2e | P1 | draft | VP-050.md |
| VP-051 | HalfChannel independence: B unaffected by A's frame production | BC-2.01.003 | internal/halfchannel | proptest | P0 | draft | VP-051.md |
| VP-052 | N consecutive OnMissingFrame calls → indicator downgrade (one level) | BC-2.06.002 | internal/metrics | integration | P1 | draft | VP-052.md |
| VP-053 | Empty-tick frame sequence: K ticks → K frames with contiguous seq nums | BC-2.01.002 | internal/halfchannel | proptest | P0 | draft | VP-053.md |
| VP-054 | Receiver dedup: first-arriving copy delivered, duplicate discarded silently | BC-2.02.002 | internal/multipath | integration | P0 | draft | VP-054.md |
| VP-055 | Presence advertisement payload round-trip: required fields present and stable (implementing_story: S-7.02; v1.2: RejectsInvalidName retired → RejectsEmptyOrInvalidUTF8 + TruncatesOversize per RULING-W6TB-J) | BC-2.03.003 | internal/discovery | proptest | P1 | draft | VP-055.md |
| VP-056 | Console detach releases session without closing it; observers unaffected | BC-2.04.004 | internal/session | integration | P1 | draft | VP-056.md |
| VP-057 | Node private key bytes absent from all emitted frame types (sampling + HKDF sketch) | BC-2.05.007 | internal/admission | proptest | P0 | implemented | VP-057.md |
| VP-058 | RouteFrame calls verifyFrameHMAC before IsAdmitted and SVTNRoute | BC-2.05.008 | internal/routing | code-audit | P0 | implemented | VP-058.md |
| VP-059 | FailureCounter.RecordHMACFailure fires E-ADM-017 at threshold (≥5 in 60s) and not before | BC-2.05.005, BC-2.05.008 | internal/admission | proptest | P0 | draft | VP-059.md |
| VP-060 | Daemon lifecycle: connect-failure exits non-zero (E-SYS-002, no relay goroutines); SIGTERM/SIGINT triggers clean shutdown (all goroutines drain, exit 0, no leak, no panic) | BC-2.04.007 | cmd/switchboard | integration | P0 | draft | VP-060.md |
| VP-061 | Metrics output contains no session content or keystroke data (DI-001 enforcement) | BC-2.06.003 | internal/metrics | code-audit | P1 | draft | VP-061.md |
| VP-062 | JSON output is valid JSON for all sbctl metrics CLI input combinations (paths list, router metrics, router status alias); pending-p99 quality sentinel propagation (v1.1); failed+pending precedence ruling (v1.3); module scope expanded to [internal/metrics, cmd/sbctl] (v1.5); BC-2.06.003 body pins corrected v1.10→v1.13 (v1.6) | BC-2.06.003 v1.13 | [internal/metrics, cmd/sbctl] | fuzz | P1 | draft | VP-062.md |
| VP-063 | PathTracker.IsDegraded() is true iff EWMA-smoothed RTT exceeds DegradedRTTThresholdMS (200.0 ms); recovery below threshold clears the flag | BC-2.02.003 | internal/paths | proptest | P0 | implemented | VP-063.md |
| VP-064 | Management server rejects unauthenticated connections (no CHALLENGE_RESPONSE, wrong key, or bad signature) → AUTH_FAIL + close; no RPC dispatched | BC-2.07.004 | internal/mgmt | integration | P0 | implemented | VP-064.md |
| VP-065 | Management server rejects replayed challenge nonce within a connection | BC-2.07.004 | internal/mgmt | integration | P1 | implemented | VP-065.md |
| VP-066 | Management server enforces bounded read: message > 64 KiB → error + close, no OOM (CWE-400) | BC-2.07.004 | internal/mgmt | unit+fuzz | P0 | implemented | VP-066.md |
| VP-067 | sbctl Authenticate() is fail-closed — returns nil only on verified AUTH_OK; all other outcomes return non-nil error | BC-2.07.002 | cmd/sbctl | integration | P0 | implemented | VP-067.md |
| VP-068 | mgmt.NewServer panics at construction if len(daemonKey) != ed25519.PrivateKeySize | BC-2.07.004 | internal/mgmt | unit | P0 | implemented | VP-068.md |
| VP-069 | mgmt.Server.Serve Returns nil on Intentional Shutdown or Context Cancellation; Non-nil on Unexpected Listener Failure | BC-2.07.004 | internal/mgmt | integration | P0 | implemented | VP-069.md |
| VP-070 | Unregistered RPC command → E-RPC-010 in-band response ok:false; connection NOT closed | BC-2.07.004 | internal/mgmt | integration | P0 | implemented | VP-070.md |
| VP-071 | Handler execution error → E-RPC-011 in-band response ok:false with verbatim error message; connection NOT closed | BC-2.07.004 | internal/mgmt | integration | P0 | implemented | VP-071.md |
| VP-072 | mgmt.Server sets write deadline before every sendJSON (HandshakeTimeout for handshake sends, RPCIdleTimeout for RPC responses); clears after each send — closes CWE-400 write-side slowloris | BC-2.07.004 | internal/mgmt | integration | P0 | implemented | VP-072.md |
| VP-073 | Console-Mode TCP Bound to Non-Loopback Address Aborts Startup with E-CFG-008 (error-taxonomy.md E-CFG-008 Variant 2 / buildMgmtListener canonical message; Ruling L) | BC-2.07.004 | cmd/switchboard | integration | P0 | implemented | VP-073.md |
| VP-074 | QualityIndicator threshold classification maps (RTT, loss) → {Green, Yellow, Red} correctly; enum cardinality = 3; all 8 boundary values correct | BC-2.06.001 | internal/metrics | unit | P1 | implemented | VP-074.md |
| VP-075 | admin.key.* handlers reject non-control callers with E-ADM-009; connection kept open; no key store mutation | BC-2.05.004 | cmd/switchboard | integration | P0 | implemented | VP-075.md |
| VP-076 | Bootstrap key non-revocable AND non-expirable invariant: both revoke and expire return their respective forbidden sentinel (E-ADM-020 / E-ADM-021) for any well-formed request; symmetric management-lockout prevention | BC-2.05.004 | cmd/switchboard | integration | P0 | implemented | VP-076.md |
| VP-TBD-ACC | p99 accumulator approximation accuracy bound: `rtt_p99_ms ≤ true_p99 + max_bucket_width` | BC-2.06.003 | internal/metrics | benchmark | deferred | deferred | (pending) [implementing story: S-BL.BENCH — unscheduled] |
| VP-VW6.NN | per-daemon binary wiring: goroutine lifecycle, config parsing, signal handling for runRouter/runConsole/runAccess/runControl — unblocked once runRouter and runConsole exit stub state | BC-2.07.002 | cmd/switchboard | integration | deferred | deferred | (pending) [implementing story: S-W6.NN — unscheduled] |

> Placeholder rows (VP-TBD-* / VP-V*.NN) use Phase=deferred and Status=deferred until scheduled; the tracking story is recorded in the Notes/File column.
> VP-TBD-ACC is bench-deferred per ARCH-03 v1.6 (F-4, S-5.02 lens-3). The p99 estimate is computed from a rolling sample buffer; exact accuracy bound against a true p99 requires a sustained load benchmark that belongs in a dedicated bench story (S-BL.BENCH). This VP will receive a permanent ID when S-BL.BENCH is scheduled. It is registered here as a placeholder to close the F-4 process gap — the property is known and intentionally deferred, not forgotten. Implementing story: S-BL.BENCH (unscheduled).
> VP-VW6.NN is Wave-6 deferred per VP-049 §Feasibility: per-daemon binary entrypoint wiring (goroutine lifecycle, config parsing, signal handling) is out of S-W5.02 scope because runRouter and runConsole remain in "not implemented" stub state. This placeholder will receive a permanent ID when the Wave-6 story is scheduled. Implementing story: S-W6.NN (unscheduled).

## Counts

| Total VPs | Proptest | Fuzz | Integration | E2E | Benchmark | Code-Audit | Unit |
|-----------|---------|------|-------------|-----|-----------|------------|------|
| 76 | 34 | 4 | 22 | 10 | 2 | 2 | 2 |

> Arithmetic check: 34 + 4 + 22 + 10 + 2 + 2 + 2 = 76. Consistent.
> VP-076 (integration, P0, cmd/switchboard) added 2026-06-30 for BC-2.05.004 EC-007 v1.12 (bootstrap-key non-revocable AND non-expirable invariant; symmetric management-lockout prevention; refs F-P18L1-001 lens-1 pass-18). Integration count increased from 21 to 22. Total 75→76. P0 count 53→54.
> VP-075 (integration, cmd/switchboard) added 2026-06-30 for BC-2.05.004 (admin.key.* handler-layer caller-role enforcement; S-6.06 lens-3 F-005 close). Integration count increased from 20 to 21. F-P7L3-001 (2026-06-30): module corrected from internal/mgmt to cmd/switchboard — BuildAdminHandlers and its handler closures reside in cmd/switchboard/admin_handlers.go.
> VP-068 (unit) — pure constructor panic-guard (no I/O). VP-074 (unit) — QualityIndicator threshold classification; 14 table-driven cases covering all 6 nominal regions + 8 boundary values.
> Integration count increased from 15 to 20: VP-069, VP-070, VP-071, VP-072, VP-073 added.
> VP-060 added 2026-06-27 for BC-2.04.007 (daemon startup/shutdown lifecycle; integration/subprocess).
> VP-061 added 2026-06-28 for BC-2.06.003 (metrics content-absence code-audit; DI-001 enforcement).
> VP-062 added 2026-06-28 for BC-2.06.003 (JSON well-formedness fuzz across all CLI forms including alias).
> VP-062 bumped to v1.1 2026-06-30 (S-5.02 Pass-3 F-T3-003): pending-quality sentinel fuzz seed (`rttP99Valid=false`) + assertion added; EC-006 cited in Source Contract; Property 5 added to Property Statement. No count change.
> VP-063 added 2026-06-28 for BC-2.02.003 PC-5 (degraded-flag boolean: IsDegraded() tracks EWMA vs DegradedRTTThresholdMS; proptest).
> VP-064 added 2026-06-28 for BC-2.07.004 (management server rejects unauthenticated; integration). Wave-5.
> VP-065 added 2026-06-28 for BC-2.07.004 (management server rejects replayed nonce; integration). Wave-5.
> VP-066 added 2026-06-28 for BC-2.07.004 (management server bounded read CWE-400; proof_method=unit+fuzz — counted under Fuzz bucket only; the unit component is NOT double-counted in the Proptest/Unit bucket). Wave-5. F-012: VP-066 is a compound method (unit+fuzz); it is counted once, in the Fuzz bucket (Fuzz=4 includes VP-066). The unit boundary-check component is subordinate to the fuzz harness and does not add a separate Proptest tally row.
> VP-067 added 2026-06-28 for BC-2.07.002 (Authenticate() fail-closed; integration via net.Pipe). Wave-5. F-006: proof_method corrected from `unit` to `integration` (2026-06-28); Authenticate() tests the two-party wire protocol over a net.Conn pair — integration-style. Method column updated; Integration=15 tally was already correct.

## Phase Distribution

| Phase | Count |
|-------|-------|
| P0 | 54 |
| P1 | 18 |
| P2 | 4 |
| **Total** | **76** |

> Phase recounted 2026-06-30: VP-076 (P0, integration) added for BC-2.05.004 EC-007 v1.12 (bootstrap-key non-revocable AND non-expirable invariant). P0 = 54. P1 = 18. P2 = 4. Total = 76.

## BC Coverage Check

45 BCs total (44 prior + BC-2.07.004 added Wave-5). All 45 have at least one VP. VP-076 added 2026-06-30 for BC-2.05.004 EC-007 v1.12 (bootstrap-key non-revocable AND non-expirable invariant; symmetric management-lockout prevention; refs F-P18L1-001). VP-075 added 2026-06-30 for BC-2.05.004 (handler-layer caller-role enforcement; S-6.06 lens-3 F-005). VP-061 and VP-062 added for BC-2.06.003 (Phase 6 hardening). VP-063 added for BC-2.02.003 PC-5 (proptest). VP-064, VP-065, VP-066 added for BC-2.07.004 (Wave-5 management server). VP-067 added for BC-2.07.002 (Authenticate() fail-closed; Wave-5). VP-068–VP-073 added 2026-06-29 for BC-2.07.004 v1.3 Wave-5 Convergence Rulings A–E (Invariant 8, PC-10, PC-11, PC-12, PC-1 write deadline, EC-013 loopback). VP-074 added 2026-06-29 for BC-2.06.001 threshold classification (unit; L-001 disambiguation). Zero coverage gaps.

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 2.34 | 2026-07-02 | VP-050 bumped v1.2→v1.3 (F-P4L3-MED-002 propagation): Story Trace row transport clause updated — "mgmt-plane Unix socket" → "mgmt-plane transport (BC-2.07.004 EC-013)"; Story row bumped S-7.03 v1.3 → v1.4. Governance-only. No count or method changes; total remains 76. |
| 2.33 | 2026-07-02 | VP-048 bumped v1.8→v1.9 (F-P5L3-LOW-1): Story Trace P3 sub-test name corrected — "destroy absent from list and blocks admission" → "TestAdminSVTNDestroy_E2E_VP048Property3" (top-level test in admin_handlers_e2e_test.go:1170). Governance-only; no property text change. No count or method changes; total remains 76. |
| 2.32 | 2026-07-02 | VP-048 bumped v1.7→v1.8 (F-P4L3-MED-1, POL-003): source_bc pin sync BC-2.07.001 v1.11→v1.12. Property text unchanged. Governance-only. No count or method changes; total remains 76. |
| 2.31 | 2026-07-02 | VP-050 bumped v1.1→v1.2 (F-P3L3-MED-001): bump Story trace row S-7.03 v1.2→v1.3 (POL-003 candidate sync). No count or method changes; total remains 76. |
| 2.30 | 2026-07-02 | VP-050 bumped v1.0→v1.1 (F-P2L3-06/07): implementing_story S-7.03 added to frontmatter; Story Trace section added; phantom testenv.NewFull replaced with in-process mgmt.NewServer + net.Listen skeleton (no testenv package in codebase; pattern from cmd/sbctl/e2e_test.go S-W5.02); catalog row title annotated with implementing_story. No count or method changes; total remains 76. |
| 2.29 | 2026-07-02 | VP-048 bumped v1.6→v1.7 (F-P3L3-M-03): source_bc pin corrected v1.7→v1.11 (BC-2.07.001 Inv-3 destroy-authority clarification per RULING-W6TB-A). No property text changes; no count or method changes; total remains 76. |
| 2.28 | 2026-07-01 | VP-055 bumped v1.1→v1.2 (RULING-W6TB-J, O-P4L3-01): retire TestPropPresenceAdvertisement_RejectsInvalidName; add TestPropPresenceAdvertisement_RejectsEmptyOrInvalidUTF8 (empty and invalid-UTF-8 inputs → error) and TestPropPresenceAdvertisement_TruncatesOversize (>255-byte valid UTF-8 → truncated ≤255 bytes with "…" suffix, err == nil). Aligns with S-7.02 AC-004b and BC-2.03.003 v1.3 EC-001. Round-trip property unchanged. No count or method changes; total remains 76. |
| 2.27 | 2026-07-01 | VP-045 bumped v1.1→v1.2 (Pass-3 L3 F-P3L3-H1 sibling propagation per RULING-W6TB-D): partial-coverage note added (BC-2.03.002 PC-1/PC-2/PC-4/PC-5 in-process verified; real-socket PC-3 deferred to S-BL.DISCOVERY-WIRE). No count or method changes; total remains 76. |
| 2.26 | 2026-07-01 | VP-044 bumped v1.0→v1.1, VP-045 bumped v1.0→v1.1, VP-055 bumped v1.0→v1.1 (S-7.02 LENS-3 traceability backfill per RULING-W6TB-D): implementing_story S-7.02 added to frontmatter of all three; Story Trace sections added; catalog row titles annotated. VP-044 partial-coverage note: in-process registry seam only; multicast wire deferred to S-BL.DISCOVERY-WIRE. No count or method changes; total remains 76. |
| 2.25 | 2026-07-01 | VP-047 bumped v1.3→v1.4 (RULING-W6TB-F §Ruling 1, F-L3-001): Ruling-1 interim clauses retracted — Property Statement updated (router_addr MUST equal PathSnapshot.RouterAddr; "" valid only for addr-less NewPathTracker paths), proof harness struct comment updated, integration test assertion updated. DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER closed (BC-2.06.003 v1.15; S-BL.ROUTER-ADDR). No count or method changes; total remains 76. |
| 2.24 | 2026-07-01 | VP-043 bumped v1.0→v1.1 (S-7.01 LENS-3 traceability backfill): implementing_story S-7.01 added to frontmatter; Story Trace section added; catalog row title annotated with implementing_story. No count or method changes; total remains 76. |
| 2.23 | 2026-07-01 | VP-062 bumped v1.5→v1.6 (F-P5L3R-02 Pass-6 L3): BC-2.06.003 body pins corrected v1.10→v1.13 at 7 sites; catalog row BC column annotated v1.13. Prior v1.11 changelog entry was aspirational — body sweep actually completed at v1.6. No count or method changes; total remains 76. |
| 2.22 | 2026-07-01 | VP-048 bumped v1.3→v1.4 (Ruling-7 defense-in-depth, Pass-3 L3 handoff): property (3) and mutation-test invariant added — handler MUST check `caller.role == RoleControl` explicitly after `IsBootstrapKey(caller)`; non-bootstrap-role caller rejected E-ADM-009 before SVTN state consulted; row 5 added to Story Trace (S-6.07); BC-2.07.001 bumped v1.5→v1.6 with Inv-3 defense-in-depth note. No count or method changes; total remains 76. |
| 2.21 | 2026-07-01 | VP-062 bumped v1.4→v1.5 (F-L3-005 Pass-3 L3): module scope expanded from `cmd/sbctl` to `[internal/metrics, cmd/sbctl]`; catalog row Module column updated; BC-2.06.003 pin updated to v1.11. No count or method changes; total remains 76. |
| 2.20 | 2026-06-30 | VP-062 version pin bumped v1.2→v1.3 in catalog row description: added "failed+pending precedence ruling (v1.3)" annotation (S502-DEFER-3 closure, commit 7ee5b82). No count or method changes; total remains 76. |
| 2.19 | 2026-06-30 | Wave-5 merge closure: VP-063 through VP-076 status flipped draft → implemented. All 14 Wave-5 VPs belong to stories merged in Wave 5 (S-5.03 PR#30, S-W5.01 PR#31, S-6.03 PR#32, S-6.02 PR#34, S-5.01 PR#35, S-6.06 PR#36, S-5.02 PR#37, S-W5.02 PR#38). Pre-Wave-6 hygiene sweep. |
| 2.18 | 2026-06-30 | Pass-6 L3 fix F-P6L3-001: normalize placeholder-row Status column — VP-VW6.NN Status changed from "draft" to "deferred" (both placeholders now Phase=deferred, Status=deferred). Added footnote above placeholder footers explaining placeholder-row conventions. Ref F-P6L3-001. |
| 2.17 | 2026-06-30 | Pass-5 L3 fix F-P5L3-004: normalize placeholder-row Phase column — VP-TBD-ACC and VP-VW6.NN Phase column changed from story-ID strings to "deferred"; implementing-story identifiers moved to Notes column. No count changes to active VP tallies. VP-049 §Story Trace pin bumped v1.3→v1.4 (F-P5L3-001 sibling propagation after story v1.3→v1.4). |
| 2.16 | 2026-06-30 | S-W5.02 Pass-2 fix F-P2L2-005: VP-049 bumped v1.1→v1.2 — proof harness skeleton API drift corrected (missing "fmt" import, NewServer 5-arg signature with ln+daemonVersion, Serve(ctx) lifecycle, Shutdown wiring, Authenticate cross-package note). VP-VW6.NN Wave-6 placeholder stub registered (draft, unscheduled) per VP-049 §Feasibility. No count changes to active VP tallies. |
| 2.15 | 2026-06-30 | S-W5.02 Pass-1 fix-burst: VP-049 bumped v1.0→v1.1 — implementing_story updated S-6.03→S-W5.02 per dep-graph v1.3 anchor propagation; proof harness skeleton rewritten to in-process mgmt.NewServer pattern (four daemon instances, distinct handler tables); §Story Trace and §Feasibility added. Row title updated with implementing_story annotation. No count changes. |
| 2.14 | 2026-06-30 | Pass-6 fix-burst F-P6L3-002/003: VP-062 bumped v1.1→v1.2 — (1) stale BC-2.06.003 v1.5 pin swept to v1.7 at 4 locations (stale-pin only; EC-006 semantics unchanged); (2) implementing_story S-5.02 → S-W5.04 per VP-047 Pass-4 Ruling-3 precedent (daemon-side types deferred from S-5.02 to S-W5.04). No count or catalog-row changes. |
| 2.13 | 2026-06-30 | Pass-24 lens-3 F-P24L3-001: VP-076 bumped v1.3→v1.4 — Source Contract cite error-taxonomy.md v3.8→v3.9 (stale taxonomy version carryover; v3.9 authoritative since Pass-22 commit 4b42dd5). No count or catalog-row changes. |
| 2.12 | 2026-06-30 | Pass-22 F-P22L3-003 + F-P22L3-004 sibling-fix propagation (4th-iteration narrowing sweep): VP-076 bumped v1.2→v1.3 — Properties #1 and #2 "unconditionally" narrowed to "for any well-formed request that reaches SVTNManager"; proof-harness comment narrowed to "sentinel for any well-formed bootstrap-key request". No count or catalog-row changes (row description already correct from v2.11). |
| 2.11 | 2026-06-30 | Pass-21 F-P21L2-002 sibling-fix: VP-076 row description narrowed from "unconditionally" to "for any well-formed request" to mirror BC-2.05.004 EC-007 v1.12 + VP-076 v1.2; all v1.10 BC citation annotations updated to v1.12. No count changes. |
| 2.10 | 2026-06-30 | VP-076 minted (integration, P0, cmd/switchboard) — bootstrap-key non-revocable AND non-expirable invariant; symmetric management-lockout prevention per BC-2.05.004 EC-007 v1.12 (E-ADM-021 symmetric counterpart to E-ADM-020); refs F-P18L1-001 lens-1 pass-18. Total: 75→76. Integration: 21→22. P0: 53→54. |
| 2.9 | 2026-06-30 | Pass-12 lens-3 (F-P12L3-001/002/003): VP-065 v1.3 (add missing mgmt import, drop dead encoding/json); VP-066 v1.3 (add missing "io" import for fuzz harness io.Copy/io.Discard/io.LimitReader); VP-064 v1.3 (Handler.Fn first param interface{}→context.Context, return (interface{},error)→(any,error)). Imports + Handler.Fn signature alignment — closes Pass-11 partial-fix gap on harness compilability. No VP count changes. |
| 2.8 | 2026-06-30 | Pass-11 backfill (F-P11L3-003/004): VP-075 v1.3 (F-P8L2-004 Source Contract correction), v1.4 (F-P9L2-001 NewServer 4→5 arg + O-P9L2-002 SVTN registration wiring), v1.5 (F-P10L2-001 net.Pipe→net.Listen + F-P10L2-003 helper citation), v1.6 (F-P11L2-003 consolePub/accessPub redundancy); VP-064/065/066 v1.1 (F-P10L3-001 NewServer arg count); VP-064/065/066 v1.2 (F-P11L3-001 net.Pipe→net.Listen sibling-fix propagation from VP-075 v1.5). No VP count changes. |
| 2.7 | 2026-06-30 | PO Ruling 3 (S-5.02 Pass-4 scope ruling, decisions/S-5.02-pass4-scope-ruling.md): VP-047 `implementing_story` transferred S-5.02 → S-W5.04 per `vp_index_is_vp_catalog_source_of_truth` policy. No count changes; VP property/invariant content unchanged. VP-047.md bumped to v1.2. |
| 2.6 | 2026-06-30 | S-5.02 Pass-3 F-T3-003: VP-062 bumped to v1.1 — pending-quality sentinel coverage added (BC-2.06.003 v1.5 EC-006). Fuzz corpus seed 7 (`rttP99Valid=false`, assert `quality=="pending"`); PropTest case `"pending p99 path"` added; Property 5 added. No VP count change (existing VP, behavioral extension only). |
| 2.5 | 2026-06-30 | S-6.06 lens-3 F-005 close: VP-075 (integration, P0) minted — admin.key.* handler-layer caller-role enforcement; server-side role lookup; E-ADM-009 rejection for non-control callers; connection kept open. source_bc: BC-2.05.004 (DI-001 / PC-1 admission-control authority). Implementing story: S-6.06. Counts: Total=75, Integration=21, P0=53. F-P7L3-001 (2026-06-30): module col corrected internal/mgmt → cmd/switchboard; per-module rollup: cmd/switchboard +1 (3→4 VPs), internal/mgmt -1 (9→8 VPs). |
| 2.4 | 2026-06-30 | F-4 (S-5.02 lens-3): VP-TBD-ACC placeholder registered — p99 accumulator accuracy bound (`rtt_p99_ms ≤ true_p99 + max_bucket_width`) bench-deferred per ARCH-03 v1.6; implementing story S-BL.BENCH (unscheduled). No count change to active VP tallies (deferred placeholder; bucket TBD at scheduling time). |
| 2.3 | 2026-06-29 | CR-009 ruling: VP-048 ownership split — PC-1 (create) + PC-2 (bootstrap) remain owned by S-6.02; PC-3 (destroy + admission-rejection) transferred to new story S-6.05-svtn-destroy (Wave 6, depends_on S-6.02). VP-048 row Title column updated to document the split. VP-048.md Story Trace section updated. BC-2.07.001 Stories row updated. No count or method changes. |
| 2.2 | 2026-06-29 | H-001/H-002 remediation (S-5.01 API reconciliation): VP-052 title updated from "Missing expected tick within deadline → indicator downgrade" to "N consecutive OnMissingFrame calls → indicator downgrade (one level)" — reflects count-based API (no Clock injection). VP-027 and VP-052 proof harness skeletons reconciled with as-built internal/metrics API. No count or method changes. |
| 2.1 | 2026-06-29 | BC-2.06.001 VP table disambiguation (L-001): VP-074 (unit) added for threshold classification (PC-2/PC-3/PC-4); BC-2.06.001 VP table updated to two clean rows — VP-074 (unit) and VP-027 (proptest). Counts: Total=74, Unit=2, P1=18. |
| 2.0 | 2026-06-29 | ARCH-12 v1.5 Ruling P VP propagation: VP-069 bumped to v1.2 — fatal-accept-error drain obligation added (closeAllConns() before connWG.Wait() on unexpected-close path; drain budget ≤200ms; TestServe_FatalAcceptErrorDrainsQuickly test obligation); source-contract citation extended to Rulings B/G/I/P; error-taxonomy v2.7→v2.8 (E-NET-001 two-case). No count changes. |
| 1.9 | 2026-06-29 | ARCH-12 v1.4 Rulings G and L VP propagation: VP-069 title updated to match v1.1 H1 (adds "Non-nil on Unexpected Listener Failure"; coverage expanded to 3 paths: Shutdown, ctx-cancel, unexpected-close per Ruling G). VP-073 title updated to match v1.1 H1 (Console-Mode TCP Bound to Non-Loopback Address Aborts Startup with E-CFG-008) and source-contract annotation extended to cite error-taxonomy.md E-CFG-008 Variant 2 / buildMgmtListener canonical message per Ruling L. No count changes. |
| 1.8 | 2026-06-29 | BC-2.07.004 v1.3 Wave-5 Convergence Rulings A–E VP assignment: VP-068 (unit, Invariant 8 — NewServer key-size panic), VP-069 (integration, PC-10 — Serve nil on shutdown), VP-070 (integration, PC-11 — E-RPC-010 unknown command in-band), VP-071 (integration, PC-12 — E-RPC-011 handler error in-band), VP-072 (integration, PC-1 write-deadline / Ruling E — slowloris write defense CWE-400), VP-073 (integration, EC-013 Ruling D — console TCP loopback rejection E-CFG-008). Counts: Total=73, Proptest=34, Fuzz=4, Integration=20, E2E=10, Benchmark=2, Code-Audit=2, Unit=1. Phase: P0=52, P1=17, P2=4. |
| 1.7 | 2026-06-28 | Wave-5 consistency audit F-006/F-012: (1) VP-067 Method column corrected from `unit` to `integration` — Authenticate() tests two-party wire protocol over net.Pipe (net.Conn pair); Integration=15 tally was already correct. (2) VP-066 bucket disambiguation added: proof_method=unit+fuzz is counted once in the Fuzz bucket only; the unit component is not double-counted. Tally 34+4+15+10+2+2=67 verified and unchanged. |
| 1.6 | 2026-06-28 | Wave-5 management plane VPs added: VP-064 (management server rejects unauthenticated; integration), VP-065 (replayed nonce rejection; integration), VP-066 (bounded read CWE-400; unit+fuzz → fuzz bucket), VP-067 (Authenticate() fail-closed; integration via net.Pipe). BC-2.07.004 coverage complete. Phase counts updated: P0=46, P1=17, P2=4. |
