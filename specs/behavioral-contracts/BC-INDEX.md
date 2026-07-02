---
artifact_id: BC-INDEX
document_type: behavioral-contract-index
level: L3
version: "2.7"
status: draft
producer: product-owner
timestamp: 2026-07-02T00:00:00
phase: 1a
inputDocuments:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/prd.md'
traces_to: '.factory/specs/prd.md'
---

# Behavioral Contract Index: Switchboard

> Master index of all BCs. One row per contract.
> Full contracts live in `ss-NN/` shard directories.
> Architecture module mapping finalized in `.factory/specs/architecture/ARCH-INDEX.md` (Phase 1b complete). Every BC carries an `architecture_module:` frontmatter field pointing to its formal Go package.

## Index

| BC ID | Title | Subsystem | CAP(s) | Priority | Scope Phase | Status | File |
|-------|-------|-----------|--------|----------|-------------|--------|------|
| BC-2.01.001 | Timeslice clock fires on every tick regardless of data availability | session-networking | CAP-001 | P0 | E | active | ss-01/BC-2.01.001.md |
| BC-2.01.002 | Empty-tick frame is a valid liveness signal | session-networking | CAP-001 | P0 | E | active | ss-01/BC-2.01.002.md |
| BC-2.01.003 | Upstream and downstream half-channels operate with independent clocks and sequence spaces | session-networking | CAP-002 | P0 | E | active | ss-01/BC-2.01.003.md |
| BC-2.01.004 | Frame outer-header encoding and decoding at 44-byte fixed layout | session-networking | CAP-003 | P0 | E | active | ss-01/BC-2.01.004.md |
| BC-2.01.005 | Channel header is opaque to routers — parseable only by endpoints | session-networking | CAP-003 | P0 | E | active | ss-01/BC-2.01.005.md |
| BC-2.01.006 | Session identity is cryptographic: node address derived from hash(SVTN-ID, public-key) | session-networking | CAP-004 | P0 | E | active | ss-01/BC-2.01.006.md |
| BC-2.01.007 | Session continuity survives IP address change via cryptographic re-authentication | session-networking | CAP-004 | P0 | E | implemented (S-1.03 / PR #7) | ss-01/BC-2.01.007.md |
| BC-2.02.001 | Duplicate-and-race: same frame sent on two fastest paths simultaneously | multipath-forwarding | CAP-005 | P0 | E | implemented (S-4.01 / PR #24) | ss-02/BC-2.02.001.md |
| BC-2.02.002 | Receiver delivers first-arriving copy and silently discards subsequent duplicates | multipath-forwarding | CAP-005 | P0 | E | implemented (S-4.01 / PR #24) | ss-02/BC-2.02.002.md |
| BC-2.02.003 | Per-path RTT and loss tracked via keep-alive probes; paths ranked by quality | multipath-forwarding | CAP-006 | P0 | E | implemented (S-4.01 / PR #24) | ss-02/BC-2.02.003.md |
| BC-2.02.004 | Upstream idempotent replay window: each frame carries last N keystrokes | multipath-forwarding | CAP-007 | P0 | E | implemented (S-4.02 / PR #25) | ss-02/BC-2.02.004.md |
| BC-2.02.005 | Downstream ARQ with piggybacked ACK and SACK bitmap | multipath-forwarding | CAP-008 | P0 | E | implemented (S-4.03 / PR #26) | ss-02/BC-2.02.005.md |
| BC-2.02.006 | TLPKTDROP terminates overdue downstream frames and signals degradation | multipath-forwarding | CAP-008 | P0 | E | implemented (S-4.03 / PR #26) | ss-02/BC-2.02.006.md |
| BC-2.02.007 | XOR parity FEC covers frame groups; single loss in group recoverable without retransmit | multipath-forwarding | CAP-009 | P1 | PE | active | ss-02/BC-2.02.007.md |
| BC-2.02.008 | Router split-horizon prevents frames being forwarded back toward arrival interface | multipath-forwarding | CAP-010 | P0 | E | implemented (S-4.04 / PR #27) | ss-02/BC-2.02.008.md |
| BC-2.02.009 | Bounded drop cache suppresses looping duplicate frames by checksum | multipath-forwarding | CAP-010 | P0 | E | implemented (S-4.04 / PR #27) | ss-02/BC-2.02.009.md |
| BC-2.03.001 | Access node advertises session presence via SVTN-scoped multicast on state change and periodic heartbeat | session-discovery | CAP-011 | P1 | PE | active | ss-03/BC-2.03.001.md |
| BC-2.03.002 | Console enumerates all SVTN sessions without specifying hostnames or IP addresses | session-discovery | CAP-012 | P1 | PE | active | ss-03/BC-2.03.002.md |
| BC-2.03.003 | Presence advertisement includes session name, attachment status, and quality indicator | session-discovery | CAP-011, CAP-012 | P1 | PE | active | ss-03/BC-2.03.003.md |
| BC-2.04.001 | Access node connects to local tmux via control mode and publishes sessions over SVTN | session-access | CAP-013 | P0 | E | implemented (S-3.01a / PR #11) | ss-04/BC-2.04.001.md |
| BC-2.04.002 | Access node falls back to PTY proxy when tmux control mode unavailable | session-access | CAP-013 | P0 | E | implemented (S-3.01b / PR #12) | ss-04/BC-2.04.002.md |
| BC-2.04.003 | Console attaches to session by name; receives downstream stream and sends upstream keystrokes | session-access | CAP-014 | P0 | E | implemented (S-3.02 / PR #13) | ss-04/BC-2.04.003.md |
| BC-2.04.004 | Console detach releases session without closing it; session continues on access node | session-access | CAP-014 | P0 | E | implemented (S-3.02 / PR #13) | ss-04/BC-2.04.004.md |
| BC-2.04.005 | Read-only console receives downstream stream; upstream keystrokes are rejected by access node | session-access | CAP-015 | P0 | E | implemented (S-3.03 / PR #14) | ss-04/BC-2.04.005.md |
| BC-2.04.006 | Two or more consoles may subscribe to the same session output simultaneously | session-access | CAP-016 | P0 | E | implemented (S-3.02 / PR #13) | ss-04/BC-2.04.006.md |
| BC-2.04.007 | Access node daemon startup succeeds or exits non-zero; SIGTERM/SIGINT triggers clean shutdown | session-access | CAP-013 | P0 | E | implemented (S-W3.04 / PR #17) | ss-04/BC-2.04.007.md |
| BC-2.05.001 | Tier 1 SVTN admission via signed key challenge | admission-security | CAP-017 | P0 | E | implemented (S-2.02 / PR #6) | ss-05/BC-2.05.001.md |
| BC-2.05.002 | Router rejects non-admitted nodes before forwarding — fail-closed | admission-security | CAP-017 | P0 | E | implemented (S-2.02 / PR #6) | ss-05/BC-2.05.002.md |
| BC-2.05.003 | Per-session Tier 2 authorization enforced by access node, not router | admission-security | CAP-018 | P0 | E | implemented (S-3.03 / PR #14) | ss-05/BC-2.05.003.md |
| BC-2.05.004 | Key lifecycle: register, revoke, and expire admission and session-authorization keys | admission-security | CAP-019 | P0 | E | active | ss-05/BC-2.05.004.md |
| BC-2.05.005 | HMAC frame authentication at first router boundary | admission-security | CAP-020 | P0 | E | implemented (S-W3.05 / PR #16) | ss-05/BC-2.05.005.md |
| BC-2.05.006 | SVTN cryptographic isolation: admitted node on SVTN-A cannot see SVTN-B traffic | admission-security | CAP-020b | P0 | E | implemented (S-2.02 / PR #6) | ss-05/BC-2.05.006.md |
| BC-2.05.007 | Node private keys never transit the network under any condition | admission-security | CAP-020a | P0 | E | implemented (S-2.02 / PR #6) | ss-05/BC-2.05.007.md |
| BC-2.05.008 | RouteFrame wire-layer HMAC enforcement (Fail-Closed for Writes) | admission-security | CAP-020 | P0 | E | implemented (S-3.04 / PR #9) | ss-05/BC-2.05.008.md |
| BC-2.06.001 | Quality indicator (green/yellow/red) derived from measured path latency and loss | quality-observability | CAP-021 | P1 | E | active | ss-06/BC-2.06.001.md |
| BC-2.06.002 | Missing expected frame is a degradation signal triggering indicator downgrade | quality-observability | CAP-021 | P1 | E | active | ss-06/BC-2.06.002.md |
| BC-2.06.003 | Per-path RTT and loss metrics queryable via sbctl | quality-observability | CAP-022 | P1 | E | active | ss-06/BC-2.06.003.md |
| BC-2.07.001 | Control node creates and destroys SVTNs; first control key bootstrapped locally | network-management | CAP-023 | P2 | E | active | ss-07/BC-2.07.001.md |
| BC-2.07.002 | sbctl unified CLI for all four daemon types with OpenSSH key authentication | network-management | CAP-024 | P2 | E | active | ss-07/BC-2.07.002.md |
| BC-2.07.003 | sbctl reports clear connection error when target daemon is unreachable | network-management | CAP-024 | P0 | E | active | ss-07/BC-2.07.003.md |
| BC-2.07.004 | Daemon management server authenticates all connections via Ed25519 challenge-response (fail-closed) | network-management | CAP-024 | P0 | E | active | ss-07/BC-2.07.004.md |
| BC-2.08.001 | Console remotely controllable via sbctl: attach, detach, switch session, navigate | console-operations | CAP-025 | P1 | PE | active | ss-08/BC-2.08.001.md |
| BC-2.09.001 | E router graduates to PE mode by adding upstream router connections in config | deployment-operations | CAP-026 | P2 | PE | active | ss-09/BC-2.09.001.md |
| BC-2.09.002 | Router sends drain signal before shutdown; nodes migrate to alternate routers | deployment-operations | CAP-027 | P2 | PE | active | ss-09/BC-2.09.002.md |
| BC-2.09.003 | Router Startup Fails Cleanly on Malformed Config with Actionable Error Message; Validated Config Is Applied to the Daemon | deployment-operations | CAP-028 | P0 | E | implemented (S-6.01 / PR #28) | ss-09/BC-2.09.003.md |

## Coverage Summary

| Subsystem | CAPs Covered | BC Count | Scope E | Scope PE | Scope P |
|-----------|-------------|----------|---------|---------|---------|
| session-networking | CAP-001–004 | 7 | 7 | 0 | 0 |
| multipath-forwarding | CAP-005–010 | 9 | 8 | 1 | 0 |
| session-discovery | CAP-011–012 | 3 | 0 | 3 | 0 |
| session-access | CAP-013–016 | 7 | 7 | 0 | 0 |
| admission-security | CAP-017–020, CAP-020a, CAP-020b | 8 | 8 | 0 | 0 |
| quality-observability | CAP-021–022 | 3 | 3 | 0 | 0 |
| network-management | CAP-023–024 | 4 | 4 | 0 | 0 |
| console-operations | CAP-025 | 1 | 0 | 1 | 0 |
| deployment-operations | CAP-026–028 | 3 | 1 | 2 | 0 |
| **Total** | **CAP-001–028 + CAP-020a, CAP-020b** | **45** | **38** | **7** | **0** |

## CAP Coverage Verification

| CAP | Realizing BCs | Status |
|-----|--------------|--------|
| CAP-001 | BC-2.01.001, BC-2.01.002 | covered |
| CAP-002 | BC-2.01.003 | covered |
| CAP-003 | BC-2.01.004, BC-2.01.005 | covered |
| CAP-004 | BC-2.01.006, BC-2.01.007 | covered |
| CAP-005 | BC-2.02.001, BC-2.02.002 | covered |
| CAP-006 | BC-2.02.003 | covered |
| CAP-007 | BC-2.02.004 | covered |
| CAP-008 | BC-2.02.005, BC-2.02.006 | covered |
| CAP-009 | BC-2.02.007 | covered |
| CAP-010 | BC-2.02.008, BC-2.02.009 | covered |
| CAP-011 | BC-2.03.001, BC-2.03.003 | covered |
| CAP-012 | BC-2.03.002, BC-2.03.003 | covered |
| CAP-013 | BC-2.04.001, BC-2.04.002, BC-2.04.007 | covered |
| CAP-014 | BC-2.04.003, BC-2.04.004 | covered |
| CAP-015 | BC-2.04.005 | covered |
| CAP-016 | BC-2.04.006 | covered |
| CAP-017 | BC-2.05.001, BC-2.05.002 | covered |
| CAP-018 | BC-2.05.003 | covered |
| CAP-019 | BC-2.05.004 | covered |
| CAP-020 | BC-2.05.005, BC-2.05.008 | covered |
| CAP-020a | BC-2.05.007 | covered |
| CAP-020b | BC-2.05.006 | covered |
| CAP-021 | BC-2.06.001, BC-2.06.002 | covered |
| CAP-022 | BC-2.06.003 | covered |
| CAP-023 | BC-2.07.001 | covered |
| CAP-024 | BC-2.07.002, BC-2.07.003, BC-2.07.004 | covered |
| CAP-025 | BC-2.08.001 | covered |
| CAP-026 | BC-2.09.001 | covered |
| CAP-027 | BC-2.09.002 | covered |
| CAP-028 | BC-2.09.003 | covered |

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 2.6 | 2026-07-02 | F-P2L3-02 reverse-trace retirement (RULING-W6TB-C §2): BC-2.06.001 v1.6→v1.7 (Stories cell: S-7.03→S-BL.CONSOLE-OBS; PC-5 console-half anchor moved per DRIFT-001b); BC-2.06.002 v1.3→v1.4 (Stories cell: S-7.03→S-BL.CONSOLE-OBS; PC-3 missCount anchor moved per DRIFT-002). BC count unchanged at 45. |
| 2.5 | 2026-07-01 | Pass-2 L3 fix-burst (RULING-W6TB-D bidirectional-trace closure): BC-2.03.001 v1.3→v1.4 (Stories row adds S-BL.DISCOVERY-WIRE with deferred PC-1/PC-3/PC-4 wire delivery annotation); BC-2.03.002 v1.2→v1.3 (Stories row adds S-BL.DISCOVERY-WIRE with deferred real-socket PC-3 aggregation annotation). BC count unchanged at 45. |
| 2.4 | 2026-07-01 | S-7.02 LENS-3 traceability backfill (RULING-W6TB-D): BC-2.03.001 v1.2→v1.3 (Traceability.Stories filled: S-7.02); BC-2.03.002 v1.1→v1.2 (Stories filled: S-7.02; Changelog section added); BC-2.03.003 v1.1→v1.2 (Stories filled: S-7.02; Changelog section added). BC count unchanged at 45. |
| 2.3 | 2026-07-01 | RULING-W6TB-A + W6TB-C architect rulings (commit 103853b): BC-2.07.001 v1.10→v1.11 (Inv-3 destroy authority clarification: admin.svtn.destroy uses resolveAndVerifyCallerRole general control-role gate, not bootstrap-only; E-ADM-009 at RPC handler layer / E-ADM-011 Variant 2 as Go-API DiD; genesis re-open after last-SVTN destroy is permitted recovery; two new canonical test vectors; Traceability Stories updated to S-6.05 v1.3); BC-2.08.001 v1.1→v1.2 (Inv-3 retracted: "same SVTN channel" replaced with management-plane Unix-socket transport requirement per ADR-006/ADR-012; ARCH-08 §6.6 forbidden-import constraint documented; S-7.03 v1.2 Traceability Stories anchored; Changelog section added). BC count unchanged at 45. |
| 2.2 | 2026-07-01 | Pass-3 L3 fix-burst: BC-2.06.003 v1.10→v1.11 (PC-3 S502-DEFER-3 rewritten — retract `status: "failed"` reference; `{active, degraded}` normative Wave-6 vocab; Wave-7 forward-looking note for S-BL.PATH-FAILED-STATUS added; Traceability section updated with Wave-7 Backlog Stories). BC count unchanged at 45. |
| 2.1 | 2026-07-01 | Wave-6 Tranche A Ruling-4 sibling-propagation sweep: BC-2.06.003 v1.9→v1.10 (status enum retracted from `{active,degraded,failed}` → `{active,degraded}`; `failed` reserved for S-BL.PATH-FAILED-STATUS Wave-7; EC-007 updated); BC-2.07.001 v1.4→v1.5 (cross-SVTN control-role key test vector added per F-P2L3-003). BC count unchanged at 45. |
| 2.0 | 2026-07-01 | Wave-6 Tranche A Pass-1 fix-burst: BC-2.06.003 v1.8→v1.9 (PC-1 interim router_addr empty-string allowance); BC-2.07.001 v1.3→v1.4 (Inv-3 tightened: admin.svtn.create bootstrap-only). BC count unchanged at 45. |
| 1.9 | 2026-06-30 | BC-2.06.003 bumped v1.7→v1.8 — S502-DEFER-3 closure: failed+pending precedence ruling added to PC-3; EC-007 added. BC title unchanged. |
| 1.8 | 2026-06-30 | BC-2.05.004 bumped v1.11→v1.12 — Pass-20 lens-3 F-P20L3-001 (MEDIUM) Option B ruling: EC-007 narrowed; "unconditionally" claim removed; handler-layer input-validation ordering clarified; VP-076 property #3 narrowed in parallel. |
| 1.7 | 2026-06-30 | BC-2.05.004 bumped v1.10→v1.11 — Pass-19 sibling-fix propagation: VP-076 added to Verification Properties table; Traceability Stories row updated to cite S-6.06/EC-007 bootstrap-protection coverage; modified: list reordered to monotonic chronological order. |
| 1.6 | 2026-06-30 | BC-2.05.004 bumped to v1.10 — EC-007 extended to cover expire symmetrically; E-ADM-021 minted; VP-076 minted; refs F-P18L1-001 lens-1 pass-18. |
| 1.5 | 2026-06-30 | BC-2.05.004 bumped to v1.9 — EC-007 narrative tightened (bootstrap key unconditionally non-revocable, refs F-P15L1-002). |
| 1.4 | 2026-06-28 | BC-2.07.004 (Daemon management server authenticates all connections via Ed25519 challenge-response) added; network-management count 3→4; total 44→45. |
| 1.3 | 2026-06-28 | Wave-5 management plane BCs: BC-2.07.002 expanded with OpenSSH key authentication; BC-2.07.003 confirmed active (S-6.03 owner). network-management subsystem updated. |
| 1.2 | 2026-06-28 | BC-2.09.003 updated to implemented (S-6.01 / PR #28); BC-2.04.007 updated to implemented (S-W3.04 / PR #17); BC-2.05.008 updated to implemented (S-3.04 / PR #9). |
| 1.1 | 2026-06-27 | Wave-3 and Wave-4 implementation status updates. |
