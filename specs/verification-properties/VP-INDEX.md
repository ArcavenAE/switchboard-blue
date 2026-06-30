---
artifact_id: VP-INDEX
document_type: verification-property-index
level: L4
version: "2.4"
status: draft
producer: architect
timestamp: 2026-06-29T00:00:00
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
| VP-043 | XOR FEC: single loss in group recoverable | BC-2.02.007 | internal/arq | proptest | P1 | draft | VP-043.md |
| VP-044 | Presence advertisement includes required fields | BC-2.03.001, BC-2.03.003 | internal/discovery | integration | P1 | draft | VP-044.md |
| VP-045 | Console session enumeration without hostnames | BC-2.03.002 | internal/discovery | e2e | P1 | draft | VP-045.md |
| VP-046 | Key lifecycle: register/revoke/expire | BC-2.05.004 | internal/svtnmgmt | integration | P1 | draft | VP-046.md |
| VP-047 | Per-path metrics queryable via sbctl | BC-2.06.003 | internal/metrics | integration | P1 | draft | VP-047.md |
| VP-048 | Control node creates/destroys SVTNs (PC-1 create + PC-2 bootstrap: S-6.02; PC-3 destroy: S-6.05; handler+CLI RPC-reachable: S-6.07) | BC-2.07.001 | internal/svtnmgmt | integration | P2 | draft | VP-048.md |
| VP-049 | sbctl unified CLI with OpenSSH auth | BC-2.07.002 | cmd/sbctl | e2e | P2 | draft | VP-049.md |
| VP-050 | Console remotely controllable via sbctl | BC-2.08.001 | cmd/sbctl | e2e | P1 | draft | VP-050.md |
| VP-051 | HalfChannel independence: B unaffected by A's frame production | BC-2.01.003 | internal/halfchannel | proptest | P0 | draft | VP-051.md |
| VP-052 | N consecutive OnMissingFrame calls → indicator downgrade (one level) | BC-2.06.002 | internal/metrics | integration | P1 | draft | VP-052.md |
| VP-053 | Empty-tick frame sequence: K ticks → K frames with contiguous seq nums | BC-2.01.002 | internal/halfchannel | proptest | P0 | draft | VP-053.md |
| VP-054 | Receiver dedup: first-arriving copy delivered, duplicate discarded silently | BC-2.02.002 | internal/multipath | integration | P0 | draft | VP-054.md |
| VP-055 | Presence advertisement payload round-trip: required fields present and stable | BC-2.03.003 | internal/discovery | proptest | P1 | draft | VP-055.md |
| VP-056 | Console detach releases session without closing it; observers unaffected | BC-2.04.004 | internal/session | integration | P1 | draft | VP-056.md |
| VP-057 | Node private key bytes absent from all emitted frame types (sampling + HKDF sketch) | BC-2.05.007 | internal/admission | proptest | P0 | implemented | VP-057.md |
| VP-058 | RouteFrame calls verifyFrameHMAC before IsAdmitted and SVTNRoute | BC-2.05.008 | internal/routing | code-audit | P0 | implemented | VP-058.md |
| VP-059 | FailureCounter.RecordHMACFailure fires E-ADM-017 at threshold (≥5 in 60s) and not before | BC-2.05.005, BC-2.05.008 | internal/admission | proptest | P0 | draft | VP-059.md |
| VP-060 | Daemon lifecycle: connect-failure exits non-zero (E-SYS-002, no relay goroutines); SIGTERM/SIGINT triggers clean shutdown (all goroutines drain, exit 0, no leak, no panic) | BC-2.04.007 | cmd/switchboard | integration | P0 | draft | VP-060.md |
| VP-061 | Metrics output contains no session content or keystroke data (DI-001 enforcement) | BC-2.06.003 | internal/metrics | code-audit | P1 | draft | VP-061.md |
| VP-062 | JSON output is valid JSON for all sbctl metrics CLI input combinations (paths list, router metrics, router status alias) | BC-2.06.003 | cmd/sbctl | fuzz | P1 | draft | VP-062.md |
| VP-063 | PathTracker.IsDegraded() is true iff EWMA-smoothed RTT exceeds DegradedRTTThresholdMS (200.0 ms); recovery below threshold clears the flag | BC-2.02.003 | internal/paths | proptest | P0 | draft | VP-063.md |
| VP-064 | Management server rejects unauthenticated connections (no CHALLENGE_RESPONSE, wrong key, or bad signature) → AUTH_FAIL + close; no RPC dispatched | BC-2.07.004 | internal/mgmt | integration | P0 | draft | VP-064.md |
| VP-065 | Management server rejects replayed challenge nonce within a connection | BC-2.07.004 | internal/mgmt | integration | P1 | draft | VP-065.md |
| VP-066 | Management server enforces bounded read: message > 64 KiB → error + close, no OOM (CWE-400) | BC-2.07.004 | internal/mgmt | unit+fuzz | P0 | draft | VP-066.md |
| VP-067 | sbctl Authenticate() is fail-closed — returns nil only on verified AUTH_OK; all other outcomes return non-nil error | BC-2.07.002 | cmd/sbctl | integration | P0 | draft | VP-067.md |
| VP-068 | mgmt.NewServer panics at construction if len(daemonKey) != ed25519.PrivateKeySize | BC-2.07.004 | internal/mgmt | unit | P0 | draft | VP-068.md |
| VP-069 | mgmt.Server.Serve Returns nil on Intentional Shutdown or Context Cancellation; Non-nil on Unexpected Listener Failure | BC-2.07.004 | internal/mgmt | integration | P0 | draft | VP-069.md |
| VP-070 | Unregistered RPC command → E-RPC-010 in-band response ok:false; connection NOT closed | BC-2.07.004 | internal/mgmt | integration | P0 | draft | VP-070.md |
| VP-071 | Handler execution error → E-RPC-011 in-band response ok:false with verbatim error message; connection NOT closed | BC-2.07.004 | internal/mgmt | integration | P0 | draft | VP-071.md |
| VP-072 | mgmt.Server sets write deadline before every sendJSON (HandshakeTimeout for handshake sends, RPCIdleTimeout for RPC responses); clears after each send — closes CWE-400 write-side slowloris | BC-2.07.004 | internal/mgmt | integration | P0 | draft | VP-072.md |
| VP-073 | Console-Mode TCP Bound to Non-Loopback Address Aborts Startup with E-CFG-008 (error-taxonomy.md E-CFG-008 Variant 2 / buildMgmtListener canonical message; Ruling L) | BC-2.07.004 | cmd/switchboard | integration | P0 | draft | VP-073.md |
| VP-074 | QualityIndicator threshold classification maps (RTT, loss) → {Green, Yellow, Red} correctly; enum cardinality = 3; all 8 boundary values correct | BC-2.06.001 | internal/metrics | unit | P1 | draft | VP-074.md |
| VP-TBD-ACC | p99 accumulator approximation accuracy bound: `rtt_p99_ms ≤ true_p99 + max_bucket_width` | BC-2.06.003 | internal/metrics | benchmark | S-BL.BENCH | deferred | (pending) |

> VP-TBD-ACC is bench-deferred per ARCH-03 v1.6 (F-4, S-5.02 lens-3). The p99 estimate is computed from a rolling sample buffer; exact accuracy bound against a true p99 requires a sustained load benchmark that belongs in a dedicated bench story (S-BL.BENCH). This VP will receive a permanent ID when S-BL.BENCH is scheduled. It is registered here as a placeholder to close the F-4 process gap — the property is known and intentionally deferred, not forgotten. Implementing story: S-BL.BENCH (unscheduled).

## Counts

| Total VPs | Proptest | Fuzz | Integration | E2E | Benchmark | Code-Audit | Unit |
|-----------|---------|------|-------------|-----|-----------|------------|------|
| 74 | 34 | 4 | 20 | 10 | 2 | 2 | 2 |

> Arithmetic check: 34 + 4 + 20 + 10 + 2 + 2 + 2 = 74. Consistent.
> VP-068 (unit) — pure constructor panic-guard (no I/O). VP-074 (unit) — QualityIndicator threshold classification; 14 table-driven cases covering all 6 nominal regions + 8 boundary values.
> Integration count increased from 15 to 20: VP-069, VP-070, VP-071, VP-072, VP-073 added.
> VP-060 added 2026-06-27 for BC-2.04.007 (daemon startup/shutdown lifecycle; integration/subprocess).
> VP-061 added 2026-06-28 for BC-2.06.003 (metrics content-absence code-audit; DI-001 enforcement).
> VP-062 added 2026-06-28 for BC-2.06.003 (JSON well-formedness fuzz across all CLI forms including alias).
> VP-063 added 2026-06-28 for BC-2.02.003 PC-5 (degraded-flag boolean: IsDegraded() tracks EWMA vs DegradedRTTThresholdMS; proptest).
> VP-064 added 2026-06-28 for BC-2.07.004 (management server rejects unauthenticated; integration). Wave-5.
> VP-065 added 2026-06-28 for BC-2.07.004 (management server rejects replayed nonce; integration). Wave-5.
> VP-066 added 2026-06-28 for BC-2.07.004 (management server bounded read CWE-400; proof_method=unit+fuzz — counted under Fuzz bucket only; the unit component is NOT double-counted in the Proptest/Unit bucket). Wave-5. F-012: VP-066 is a compound method (unit+fuzz); it is counted once, in the Fuzz bucket (Fuzz=4 includes VP-066). The unit boundary-check component is subordinate to the fuzz harness and does not add a separate Proptest tally row.
> VP-067 added 2026-06-28 for BC-2.07.002 (Authenticate() fail-closed; integration via net.Pipe). Wave-5. F-006: proof_method corrected from `unit` to `integration` (2026-06-28); Authenticate() tests the two-party wire protocol over a net.Conn pair — integration-style. Method column updated; Integration=15 tally was already correct.

## Phase Distribution

| Phase | Count |
|-------|-------|
| P0 | 52 |
| P1 | 18 |
| P2 | 4 |
| **Total** | **74** |

> Phase recounted 2026-06-29: VP-074 (P1, unit) added for BC-2.06.001 threshold classification. P0 = 52. P1 = 18. P2 = 4. Total = 74.

## BC Coverage Check

45 BCs total (44 prior + BC-2.07.004 added Wave-5). All 45 have at least one VP. VP-061 and VP-062 added for BC-2.06.003 (Phase 6 hardening). VP-063 added for BC-2.02.003 PC-5 (proptest). VP-064, VP-065, VP-066 added for BC-2.07.004 (Wave-5 management server). VP-067 added for BC-2.07.002 (Authenticate() fail-closed; Wave-5). VP-068–VP-073 added 2026-06-29 for BC-2.07.004 v1.3 Wave-5 Convergence Rulings A–E (Invariant 8, PC-10, PC-11, PC-12, PC-1 write deadline, EC-013 loopback). VP-074 added 2026-06-29 for BC-2.06.001 threshold classification (unit; L-001 disambiguation). Zero coverage gaps.

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 2.4 | 2026-06-30 | F-4 (S-5.02 lens-3): VP-TBD-ACC placeholder registered — p99 accumulator accuracy bound (`rtt_p99_ms ≤ true_p99 + max_bucket_width`) bench-deferred per ARCH-03 v1.6; implementing story S-BL.BENCH (unscheduled). No count change to active VP tallies (deferred placeholder; bucket TBD at scheduling time). |
| 2.3 | 2026-06-29 | CR-009 ruling: VP-048 ownership split — PC-1 (create) + PC-2 (bootstrap) remain owned by S-6.02; PC-3 (destroy + admission-rejection) transferred to new story S-6.05-svtn-destroy (Wave 6, depends_on S-6.02). VP-048 row Title column updated to document the split. VP-048.md Story Trace section updated. BC-2.07.001 Stories row updated. No count or method changes. |
| 2.2 | 2026-06-29 | H-001/H-002 remediation (S-5.01 API reconciliation): VP-052 title updated from "Missing expected tick within deadline → indicator downgrade" to "N consecutive OnMissingFrame calls → indicator downgrade (one level)" — reflects count-based API (no Clock injection). VP-027 and VP-052 proof harness skeletons reconciled with as-built internal/metrics API. No count or method changes. |
| 2.1 | 2026-06-29 | BC-2.06.001 VP table disambiguation (L-001): VP-074 (unit) added for threshold classification (PC-2/PC-3/PC-4); BC-2.06.001 VP table updated to two clean rows — VP-074 (unit) and VP-027 (proptest). Counts: Total=74, Unit=2, P1=18. |
| 2.0 | 2026-06-29 | ARCH-12 v1.5 Ruling P VP propagation: VP-069 bumped to v1.2 — fatal-accept-error drain obligation added (closeAllConns() before connWG.Wait() on unexpected-close path; drain budget ≤200ms; TestServe_FatalAcceptErrorDrainsQuickly test obligation); source-contract citation extended to Rulings B/G/I/P; error-taxonomy v2.7→v2.8 (E-NET-001 two-case). No count changes. |
| 1.9 | 2026-06-29 | ARCH-12 v1.4 Rulings G and L VP propagation: VP-069 title updated to match v1.1 H1 (adds "Non-nil on Unexpected Listener Failure"; coverage expanded to 3 paths: Shutdown, ctx-cancel, unexpected-close per Ruling G). VP-073 title updated to match v1.1 H1 (Console-Mode TCP Bound to Non-Loopback Address Aborts Startup with E-CFG-008) and source-contract annotation extended to cite error-taxonomy.md E-CFG-008 Variant 2 / buildMgmtListener canonical message per Ruling L. No count changes. |
| 1.8 | 2026-06-29 | BC-2.07.004 v1.3 Wave-5 Convergence Rulings A–E VP assignment: VP-068 (unit, Invariant 8 — NewServer key-size panic), VP-069 (integration, PC-10 — Serve nil on shutdown), VP-070 (integration, PC-11 — E-RPC-010 unknown command in-band), VP-071 (integration, PC-12 — E-RPC-011 handler error in-band), VP-072 (integration, PC-1 write-deadline / Ruling E — slowloris write defense CWE-400), VP-073 (integration, EC-013 Ruling D — console TCP loopback rejection E-CFG-008). Counts: Total=73, Proptest=34, Fuzz=4, Integration=20, E2E=10, Benchmark=2, Code-Audit=2, Unit=1. Phase: P0=52, P1=17, P2=4. |
| 1.7 | 2026-06-28 | Wave-5 consistency audit F-006/F-012: (1) VP-067 Method column corrected from `unit` to `integration` — Authenticate() tests two-party wire protocol over net.Pipe (net.Conn pair); Integration=15 tally was already correct. (2) VP-066 bucket disambiguation added: proof_method=unit+fuzz is counted once in the Fuzz bucket only; the unit component is not double-counted. Tally 34+4+15+10+2+2=67 verified and unchanged. |
| 1.6 | 2026-06-28 | Wave-5 management plane VPs added: VP-064 (management server rejects unauthenticated; integration), VP-065 (replayed nonce rejection; integration), VP-066 (bounded read CWE-400; unit+fuzz → fuzz bucket), VP-067 (Authenticate() fail-closed; integration via net.Pipe). BC-2.07.004 coverage complete. Phase counts updated: P0=46, P1=17, P2=4. |
