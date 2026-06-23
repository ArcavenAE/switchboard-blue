---
artifact_id: BC-INDEX
document_type: behavioral-contract-index
level: L3
version: "1.0"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
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
| BC-2.01.007 | Session continuity survives IP address change via cryptographic re-authentication | session-networking | CAP-004 | P0 | E | active | ss-01/BC-2.01.007.md |
| BC-2.02.001 | Duplicate-and-race: same frame sent on two fastest paths simultaneously | multipath-forwarding | CAP-005 | P0 | E | active | ss-02/BC-2.02.001.md |
| BC-2.02.002 | Receiver delivers first-arriving copy and silently discards subsequent duplicates | multipath-forwarding | CAP-005 | P0 | E | active | ss-02/BC-2.02.002.md |
| BC-2.02.003 | Per-path RTT and loss tracked via keep-alive probes; paths ranked by quality | multipath-forwarding | CAP-006 | P0 | E | active | ss-02/BC-2.02.003.md |
| BC-2.02.004 | Upstream idempotent replay window: each frame carries last N keystrokes | multipath-forwarding | CAP-007 | P0 | E | active | ss-02/BC-2.02.004.md |
| BC-2.02.005 | Downstream ARQ with piggybacked ACK and SACK bitmap | multipath-forwarding | CAP-008 | P0 | E | active | ss-02/BC-2.02.005.md |
| BC-2.02.006 | TLPKTDROP terminates overdue downstream frames and signals degradation | multipath-forwarding | CAP-008 | P0 | E | active | ss-02/BC-2.02.006.md |
| BC-2.02.007 | XOR parity FEC covers frame groups; single loss in group recoverable without retransmit | multipath-forwarding | CAP-009 | P1 | PE | active | ss-02/BC-2.02.007.md |
| BC-2.02.008 | Router split-horizon prevents frames being forwarded back toward arrival interface | multipath-forwarding | CAP-010 | P0 | E | active | ss-02/BC-2.02.008.md |
| BC-2.02.009 | Bounded drop cache suppresses looping duplicate frames by checksum | multipath-forwarding | CAP-010 | P0 | E | active | ss-02/BC-2.02.009.md |
| BC-2.03.001 | Access node advertises session presence via SVTN-scoped multicast on state change and periodic heartbeat | session-discovery | CAP-011 | P1 | PE | active | ss-03/BC-2.03.001.md |
| BC-2.03.002 | Console enumerates all SVTN sessions without specifying hostnames or IP addresses | session-discovery | CAP-012 | P1 | PE | active | ss-03/BC-2.03.002.md |
| BC-2.03.003 | Presence advertisement includes session name, attachment status, and quality indicator | session-discovery | CAP-011, CAP-012 | P1 | PE | active | ss-03/BC-2.03.003.md |
| BC-2.04.001 | Access node connects to local tmux via control mode and publishes sessions over SVTN | session-access | CAP-013 | P0 | E | active | ss-04/BC-2.04.001.md |
| BC-2.04.002 | Access node falls back to PTY proxy when tmux control mode unavailable | session-access | CAP-013 | P0 | E | active | ss-04/BC-2.04.002.md |
| BC-2.04.003 | Console attaches to session by name; receives downstream stream and sends upstream keystrokes | session-access | CAP-014 | P1 | E | active | ss-04/BC-2.04.003.md |
| BC-2.04.004 | Console detach releases session without closing it; session continues on access node | session-access | CAP-014 | P1 | E | active | ss-04/BC-2.04.004.md |
| BC-2.04.005 | Read-only console receives downstream stream; upstream keystrokes are rejected by access node | session-access | CAP-015 | P1 | E | active | ss-04/BC-2.04.005.md |
| BC-2.04.006 | Two or more consoles may subscribe to the same session output simultaneously | session-access | CAP-016 | P0 | E | active | ss-04/BC-2.04.006.md |
| BC-2.05.001 | Tier 1 SVTN admission via signed key challenge | admission-security | CAP-017 | P0 | E | active | ss-05/BC-2.05.001.md |
| BC-2.05.002 | Router rejects non-admitted nodes before forwarding — fail-closed | admission-security | CAP-017 | P0 | E | active | ss-05/BC-2.05.002.md |
| BC-2.05.003 | Per-session Tier 2 authorization enforced by access node, not router | admission-security | CAP-018 | P0 | E | active | ss-05/BC-2.05.003.md |
| BC-2.05.004 | Key lifecycle: register, revoke, and expire admission and session-authorization keys | admission-security | CAP-019 | P1 | E | active | ss-05/BC-2.05.004.md |
| BC-2.05.005 | HMAC frame authentication at first router boundary | admission-security | CAP-020 | P0 | E | active | ss-05/BC-2.05.005.md |
| BC-2.05.006 | SVTN cryptographic isolation: admitted node on SVTN-A cannot see SVTN-B traffic | admission-security | CAP-020 | P0 | E | active | ss-05/BC-2.05.006.md |
| BC-2.05.007 | Node private keys never transit the network under any condition | admission-security | CAP-020 | P0 | E | active | ss-05/BC-2.05.007.md |
| BC-2.06.001 | Quality indicator (green/yellow/red) derived from measured path latency and loss | quality-observability | CAP-021 | P1 | E | active | ss-06/BC-2.06.001.md |
| BC-2.06.002 | Missing expected frame is a degradation signal triggering indicator downgrade | quality-observability | CAP-021 | P1 | E | active | ss-06/BC-2.06.002.md |
| BC-2.06.003 | Per-path RTT and loss metrics queryable via sbctl | quality-observability | CAP-022 | P1 | E | active | ss-06/BC-2.06.003.md |
| BC-2.07.001 | Control node creates and destroys SVTNs; first control key bootstrapped locally | network-management | CAP-023 | P2 | E | active | ss-07/BC-2.07.001.md |
| BC-2.07.002 | sbctl unified CLI for all four daemon types with OpenSSH key authentication | network-management | CAP-024 | P2 | E | active | ss-07/BC-2.07.002.md |
| BC-2.07.003 | sbctl reports clear connection error when target daemon is unreachable | network-management | CAP-024 | P0 | E | active | ss-07/BC-2.07.003.md |
| BC-2.08.001 | Console remotely controllable via sbctl: attach, detach, switch session, navigate | console-operations | CAP-025 | P1 | PE | active | ss-08/BC-2.08.001.md |
| BC-2.09.001 | E router graduates to PE mode by adding upstream router connections in config | deployment-operations | CAP-026 | P2 | PE | active | ss-09/BC-2.09.001.md |
| BC-2.09.002 | Router sends drain signal before shutdown; nodes migrate to alternate routers | deployment-operations | CAP-027 | P2 | PE | active | ss-09/BC-2.09.002.md |
| BC-2.09.003 | Router startup fails cleanly on malformed config with actionable error message | deployment-operations | CAP-023, CAP-024 | P0 | E | active | ss-09/BC-2.09.003.md |

## Coverage Summary

| Subsystem | CAPs Covered | BC Count | Scope E | Scope PE | Scope P |
|-----------|-------------|----------|---------|---------|---------|
| session-networking | CAP-001–004 | 7 | 7 | 0 | 0 |
| multipath-forwarding | CAP-005–010 | 9 | 8 | 1 | 0 |
| session-discovery | CAP-011–012 | 3 | 0 | 3 | 0 |
| session-access | CAP-013–016 | 6 | 6 | 0 | 0 |
| admission-security | CAP-017–020 | 7 | 7 | 0 | 0 |
| quality-observability | CAP-021–022 | 3 | 3 | 0 | 0 |
| network-management | CAP-023–024 | 3 | 3 | 0 | 0 |
| console-operations | CAP-025 | 1 | 0 | 1 | 0 |
| deployment-operations | CAP-026–027 | 3 | 1 | 2 | 0 |
| **Total** | **CAP-001–027** | **42** | **35** | **7** | **0** |

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
| CAP-013 | BC-2.04.001, BC-2.04.002 | covered |
| CAP-014 | BC-2.04.003, BC-2.04.004 | covered |
| CAP-015 | BC-2.04.005 | covered |
| CAP-016 | BC-2.04.006 | covered |
| CAP-017 | BC-2.05.001, BC-2.05.002 | covered |
| CAP-018 | BC-2.05.003 | covered |
| CAP-019 | BC-2.05.004 | covered |
| CAP-020 | BC-2.05.005, BC-2.05.006, BC-2.05.007 | covered |
| CAP-021 | BC-2.06.001, BC-2.06.002 | covered |
| CAP-022 | BC-2.06.003 | covered |
| CAP-023 | BC-2.07.001, BC-2.09.003 | covered |
| CAP-024 | BC-2.07.002, BC-2.07.003 | covered |
| CAP-025 | BC-2.08.001 | covered |
| CAP-026 | BC-2.09.001 | covered |
| CAP-027 | BC-2.09.002 | covered |
