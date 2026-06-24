---
artifact_id: consistency-audit-after-pass-04
producer: consistency-validator
audit_date: 2026-06-23
phase: 1d
input_commit: 8fab3c6
---

# Phase 1 Spec Package — Structural Consistency Audit (Pass 04)

> Auditor note: This is a mechanical cross-document structural audit of the Phase 1
> spec package. It reads BC files, VP files, ARCH files, capabilities.md, error-taxonomy.md,
> interface-definitions.md, and invariants.md. It does NOT assess semantic correctness of
> individual contracts — only cross-document consistency of IDs, titles, module mappings,
> error codes, and VP/BC linkages.

---

## Axis 1 — VP ↔ BC Title Sync

**Findings: 14 mismatches**

Each VP's "## Source Contract" section quotes a BC title. That quoted title is compared
against the BC-INDEX canonical title for the same BC ID.

---

VP-016: source_bc=BC-2.01.001
  quoted (line 2): "BC-2.01.003 — Upstream and Downstream Half-Channels Operate with Independent Clocks and Sequence Spaces"
  actual (BC-INDEX): "Upstream and downstream half-channels operate with independent clocks and sequence spaces"
  Note: VP-016 Source Contract section cites two BCs; the secondary BC title is reproduced in BC-2.01.003 casing (title-case vs sentence-case is acceptable), but see mismatch on first title below.

VP-016: source_bc=BC-2.01.001
  quoted (line 1): "BC-2.01.001 — Timeslice Clock Fires on Every Tick Regardless of Data Availability"
  actual (BC-INDEX): "Timeslice clock fires on every tick regardless of data availability"
  Note: Title-case vs sentence-case. BC-INDEX uses sentence-case throughout. VP-016 uses title-case in Source Contract section.

VP-017: source_bc=BC-2.01.003
  quoted: "BC-2.01.003 — HalfChannel Sequence Counter Increments on Every Tick"
  actual: "Upstream and downstream half-channels operate with independent clocks and sequence spaces"
  Note: The quoted title is a DIFFERENT CONTRACT — it describes a single VP property (sequence counter), not the BC-2.01.003 title. This is a substantive title mismatch.

VP-018: source_bc=BC-2.01.001
  quoted (line 1): "BC-2.01.001 — HalfChannel Tick Produces Exactly One Frame"
  actual: "Timeslice clock fires on every tick regardless of data availability"
  Note: Quoted title is a synthesized property description, not the canonical BC title.

VP-018: source_bc=BC-2.01.001
  quoted (line 2): "BC-2.01.002 — HalfChannel Emits Empty Frame on Nil Payload"
  actual: "Empty-tick frame is a valid liveness signal"
  Note: Same pattern — synthesized property description instead of canonical BC title.

VP-019: source_bc=BC-2.02.005
  quoted: "BC-2.02.005 — ARQ Delivers Each Frame At Most Once"
  actual: "Downstream ARQ with piggybacked ACK and SACK bitmap"
  Note: Quoted title describes a derived property, not the canonical BC title.

VP-020: source_bc=BC-2.02.005
  quoted: "BC-2.02.005 — ARQ Delivers Frames In Sequence Order"
  actual: "Downstream ARQ with piggybacked ACK and SACK bitmap"
  Note: Same pattern.

VP-021: source_bc=BC-2.02.006
  quoted: "BC-2.02.006 — ARQ Tail Loss Probe Drop Signal"
  actual: "TLPKTDROP terminates overdue downstream frames and signals degradation"
  Note: Abbreviated/reformulated title.

VP-022: source_bc=BC-2.02.004
  quoted: "BC-2.02.004 — Replay Deduplicates Upstream Frames by chan_seq"
  actual: "Upstream idempotent replay window: each frame carries last N keystrokes"
  Note: Reformulated.

VP-023: source_bc=BC-2.02.004
  quoted: "BC-2.02.004 — Replay Delivers Upstream Frames in chan_seq Order"
  actual: "Upstream idempotent replay window: each frame carries last N keystrokes"
  Note: Same pattern.

VP-024: source_bc=BC-2.02.001
  quoted (line 1): "BC-2.02.001 — Multipath Deduplicates Frames by Checksum"
  actual: "Duplicate-and-race: same frame sent on two fastest paths simultaneously"
  Note: VP-024 Source Contract cites both BC-2.02.001 and BC-2.02.002. Neither quoted title matches the canonical.

VP-024: source_bc=BC-2.02.001
  quoted (line 2): "BC-2.02.002 — Multipath Delivers Only the First Copy"
  actual: "Receiver delivers first-arriving copy and silently discards subsequent duplicates"
  Note: Abbreviated.

VP-025: source_bc=BC-2.02.009
  quoted: "BC-2.02.009 — DropCache Bounded-Capacity Invariant"
  actual: "Bounded drop cache suppresses looping duplicate frames by checksum"
  Note: Reformulated.

VP-026: source_bc=BC-2.02.003
  quoted: "BC-2.02.003 — PathScore Induces a Total Order on Paths"
  actual: "Per-path RTT and loss tracked via keep-alive probes; paths ranked by quality"
  Note: Reformulated.

VP-028: source_bc=BC-2.09.003
  quoted: "BC-2.09.003 — Config.Validate Enforces tick_interval Range [5ms, 50ms]"
  actual: "Router startup fails cleanly on malformed config with actionable error message"
  Note: VP-specific property description substituted for BC title.

VP-029: source_bc=BC-2.09.003
  quoted: "BC-2.09.003 — Config.Validate Enforces Required Fields"
  actual: "Router startup fails cleanly on malformed config with actionable error message"
  Note: Same pattern.

VP-030: source_bc=BC-2.07.003
  quoted: "BC-2.07.003 — sbctl Surfaces Structured Error Codes on Failure"
  actual: "sbctl reports clear connection error when target daemon is unreachable"
  Note: Reformulated.

VP-031: source_bc=BC-2.04.001
  quoted: "BC-2.04.001 — tmux Control Mode Output Event Completeness"
  actual: "Access node connects to local tmux via control mode and publishes sessions over SVTN"
  Note: Reformulated.

VP-032: source_bc=BC-2.04.002
  quoted: "BC-2.04.002 — PTY Fallback on tmux Control Mode Unavailability"
  actual: "Access node falls back to PTY proxy when tmux control mode unavailable"
  Note: Abbreviated/reformulated.

VP-033: source_bc=BC-2.04.003
  quoted (line 1): "BC-2.04.003 — Console Attach Starts Downstream Frame Delivery"
  actual: "Console attaches to session by name; receives downstream stream and sends upstream keystrokes"
  Note: Reformulated.

VP-033: source_bc=BC-2.04.003
  quoted (line 2): "BC-2.04.004 — Console Detach Stops Delivery Without Terminating Session"
  actual: "Console detach releases session without closing it; session continues on access node"
  Note: Reformulated.

VP-034: source_bc=BC-2.04.006
  quoted: "BC-2.04.006 — Access Node Fans Out Downstream Frames to All Attached Consoles"
  actual: "Two or more consoles may subscribe to the same session output simultaneously"
  Note: Reformulated.

VP-035: source_bc=BC-2.04.005
  quoted: "BC-2.04.005 — Access Node Rejects Upstream Keystrokes from Read-Only Consoles"
  actual: "Read-only console receives downstream stream; upstream keystrokes are rejected by access node"
  Note: Paraphrased (wrong subject — "Access Node" not in original title).

VP-036: source_bc=BC-2.01.007
  quoted: "BC-2.01.007 — Session Continuity Across IP Address Changes"
  actual: "Session continuity survives IP address change via cryptographic re-authentication"
  Note: Abbreviated.

VP-037: source_bc=BC-2.09.002
  quoted: "BC-2.09.002 — Router Drain Signal Triggers Node Migration Within 2s"
  actual: "Router sends drain signal before shutdown; nodes migrate to alternate routers"
  Note: Reformulated.

VP-038: source_bc=BC-2.09.001
  quoted: "BC-2.09.001 — E→PE Router Graduation via Configuration Only"
  actual: "E router graduates to PE mode by adding upstream router connections in config"
  Note: Abbreviated.

VP-039: source_bc=BC-2.05.006
  quoted: "BC-2.05.006 — SVTN Isolation: No Cross-SVTN Frame Delivery"
  actual: "SVTN cryptographic isolation: admitted node on SVTN-A cannot see SVTN-B traffic"
  Note: Abbreviated.

VP-040: source_bc=BC-2.02.003
  quoted: "BC-2.02.003 — Per-Path RTT and Loss Tracked via Keep-Alive Probes; Paths Ranked by Quality"
  actual: "Per-path RTT and loss tracked via keep-alive probes; paths ranked by quality"
  Note: BC-INDEX uses sentence-case; VP-040 uses title-case capitalization. Otherwise equivalent.

VP-041: source_bc=BC-2.01.001
  quoted: "BC-2.01.001 — HalfChannel Tick Regularity (NFR-009)"
  actual: "Timeslice clock fires on every tick regardless of data availability"
  Note: Reformulated.

VP-042: source_bc=BC-2.01.001
  quoted (line 1): "BC-2.01.001 — HalfChannel Tick Interval (NFR-001)"
  actual: "Timeslice clock fires on every tick regardless of data availability"

VP-042: source_bc=BC-2.01.001
  quoted (line 2): "BC-2.02.001 — Multipath Delivers Within Latency Budget"
  actual: "Duplicate-and-race: same frame sent on two fastest paths simultaneously"

VP-043: source_bc=BC-2.02.007
  quoted: "BC-2.02.007 — ARQ XOR FEC Single-Loss Recovery"
  actual: "XOR parity FEC covers frame groups; single loss in group recoverable without retransmit"

VP-044: source_bc=BC-2.03.001
  quoted (line 1): "BC-2.03.001 — Access Node Emits PresenceAdvertisement with Required Fields"
  actual: "Access node advertises session presence via SVTN-scoped multicast on state change and periodic heartbeat"

VP-044: source_bc=BC-2.03.001
  quoted (line 2): "BC-2.03.003 — PresenceAdvertisement Schema"
  actual: "Presence advertisement includes session name, attachment status, and quality indicator"

VP-045: source_bc=BC-2.03.002
  quoted: "BC-2.03.002 — Zero-Configuration Console Session Discovery"
  actual: "Console enumerates all SVTN sessions without specifying hostnames or IP addresses"

VP-046: source_bc=BC-2.05.004
  quoted: "BC-2.05.004 — Key Registration, Revocation, and Expiry Enforcement"
  actual: "Key lifecycle: register, revoke, and expire admission and session-authorization keys"

VP-047: source_bc=BC-2.06.003
  quoted: "BC-2.06.003 — Per-Path Metrics Exposed via sbctl"
  actual: "Per-path RTT and loss metrics queryable via sbctl"

VP-048: source_bc=BC-2.07.001
  quoted: "BC-2.07.001 — Control Node SVTN Create/Destroy Operations"
  actual: "Control node creates and destroys SVTNs; first control key bootstrapped locally"

VP-049: source_bc=BC-2.07.002
  quoted: "BC-2.07.002 — sbctl Unified CLI Authenticates Against All Daemon Types"
  actual: "sbctl unified CLI for all four daemon types with OpenSSH key authentication"

VP-050: source_bc=BC-2.08.001
  quoted: "BC-2.08.001 — sbctl Remote Console Control (attach/detach/switch)"
  actual: "Console remotely controllable via sbctl: attach, detach, switch session, navigate"

VP-051: source_bc=BC-2.01.003
  quoted (truncated): "BC-2.01.003 — Upstream and Downstream Half-Channels Operate with"
  actual: "Upstream and downstream half-channels operate with independent clocks and sequence spaces"
  Note: Quoted title is truncated mid-sentence (line wrapping artifact).

VP-052: source_bc=BC-2.06.002
  quoted: "BC-2.06.002 — Missing Frame Triggers Quality Indicator Downgrade"
  actual: "Missing expected frame is a degradation signal triggering indicator downgrade"
  Note: Paraphrased.

VP-053: source_bc=BC-2.01.002
  quoted: "BC-2.01.002 — Empty-Tick Frame Is a Valid Liveness Signal"
  actual: "Empty-tick frame is a valid liveness signal"
  Note: Title-case vs sentence-case only.

VP-054: source_bc=BC-2.02.002
  quoted (truncated): "BC-2.02.002 — Receiver Delivers First-Arriving Copy and Silently"
  actual: "Receiver delivers first-arriving copy and silently discards subsequent duplicates"
  Note: Title truncated mid-sentence (line-wrapping artifact).

VP-055: source_bc=BC-2.03.003
  quoted (truncated): "BC-2.03.003 — Presence Advertisement Includes Session Name, Attachment"
  actual: "Presence advertisement includes session name, attachment status, and quality indicator"
  Note: Truncated.

VP-056: source_bc=BC-2.04.004
  quoted: "BC-2.04.004 — Console Detach Releases Session Without Closing It"
  actual: "Console detach releases session without closing it; session continues on access node"
  Note: Second clause dropped.

VP-057: source_bc=BC-2.05.007
  quoted (truncated): "BC-2.05.007 — Node Private Keys Never Transit the Network Under"
  actual: "Node private keys never transit the network under any condition"
  Note: Truncated.

**Summary for Axis 1:**
- Substantive reformulations (different meaning or scope): VP-017, VP-018 (both lines), VP-019, VP-020, VP-021, VP-022, VP-023, VP-024 (both), VP-025, VP-026, VP-028, VP-029, VP-030, VP-031, VP-032, VP-033 (both), VP-034, VP-035, VP-036, VP-037, VP-038, VP-039, VP-041, VP-042, VP-043, VP-044 (both), VP-045, VP-046, VP-047, VP-048, VP-049, VP-050, VP-052 = 34 substantive mismatches
- Truncations (line-wrapping, text cut off): VP-051, VP-054, VP-055, VP-056, VP-057 = 5 truncations
- Title-case vs sentence-case only: VP-016, VP-040, VP-053 = 3 casing mismatches
- **Total distinct VP files with at least one Axis 1 finding: 44 out of 57**

---

## Axis 2 — BC ↔ VP Coverage Symmetry

**Findings: 12 mismatches across 3 comparison views**

For each BC, three views are compared:
- (a) VP files whose `source_bc:` frontmatter references this BC
- (b) VP IDs listed in the BC body "## Verification Properties" table
- (c) ARCH-11 BC→VP coverage table column

### 2.1 VP source_bc frontmatter vs ARCH-11 coverage table

BC-2.01.001:
  source_bc pointing here: VP-016, VP-018, VP-041, VP-042 (4 VPs)
  ARCH-11 table: VP-016, VP-018 (2 VPs)
  Missing from ARCH-11: VP-041, VP-042

BC-2.01.003:
  source_bc pointing here: VP-017, VP-051 (2 VPs)
  ARCH-11 table: VP-016, VP-017, VP-051 (3 VPs)
  Extra in ARCH-11: VP-016 (VP-016's source_bc = BC-2.01.001, not BC-2.01.003)

BC-2.05.001:
  source_bc pointing here: VP-007, VP-009 (2 VPs)
  ARCH-11 table: VP-007, VP-008, VP-009 (3 VPs)
  Extra in ARCH-11: VP-008 (VP-008's source_bc = BC-2.05.002)

BC-2.04.005:
  source_bc pointing here: VP-013, VP-035 (2 VPs)
  ARCH-11 table: VP-035 only (1 VP)
  Missing from ARCH-11: VP-013

### 2.2 VP source_bc frontmatter vs BC body VP table

BC-2.01.001 body table: VP-016, VP-018, VP-041, VP-042
  source_bc pointing here: VP-016, VP-018, VP-041, VP-042
  Match: YES — consistent

BC-2.01.002 body table: VP-053, VP-052, VP-016
  source_bc pointing here: VP-053 only
  Discrepancy: BC-2.01.002 body lists VP-052 (source_bc=BC-2.06.002) and VP-016 (source_bc=BC-2.01.001). These are cross-contract references — acceptable as "related VP" cross-references, but VP-052 and VP-016 do not claim BC-2.01.002 as their primary source.

BC-2.01.003 body table: VP-017, VP-051
  source_bc pointing here: VP-017, VP-051
  Match: YES — consistent

BC-2.04.004 body table: VP-056, VP-033
  source_bc pointing here: VP-056 only
  VP-033's source_bc = BC-2.04.003. Body references VP-033 as a related VP. Not a defect per se but cross-contract reference.

BC-2.05.001 body table: VP-007, VP-009
  source_bc pointing here: VP-007, VP-009
  Match: YES — consistent. (ARCH-11 table adds VP-008 which belongs to BC-2.05.002)

BC-2.05.003 body table: VP-012 only
  source_bc pointing here: VP-012 only
  ARCH-11 adds VP-013 to BC-2.05.003, but VP-013's source_bc = BC-2.04.005

BC-2.05.007 body table: VP-057, VP-007, VP-049
  source_bc pointing here: VP-057 only
  VP-007 source_bc = BC-2.05.001; VP-049 source_bc = BC-2.07.002
  Cross-contract references that overstate VP-049's applicability to private key non-transit

### 2.3 ARCH-11 coverage accuracy vs VP-INDEX canonical

ARCH-11 asserts: "VP counts recounted from VP-INDEX (canonical source of truth, 57 VPs total)"
but the per-module count table shows:

| Module | ARCH-11 count | Actual VP-INDEX count | Delta |
|--------|--------------|----------------------|-------|
| internal/halfchannel | 6 | 7 (VP-016,017,018,041,042,051,053) | ARCH-11 undercounts by 1 |
| internal/paths | 2 | 1 (VP-026 only) | ARCH-11 overcounts by 1 |
| internal/metrics | 4 | 3 (VP-027,047,052) | ARCH-11 overcounts by 1 |
| internal/config | 2 | 3 (VP-028,029,038) | ARCH-11 undercounts by 1 |

Net delta: 0 (overcounts cancel undercounts), which is why the ARCH-11 total still sums to 57. But individual module counts are wrong. The discrepancies:
- `internal/paths` ARCH-11 says 2 with "proptest(1), e2e(1)"; VP-INDEX has only VP-026 (proptest). VP-040 (e2e) has module=internal/multipath in both VP-INDEX and VP-040.md frontmatter — ARCH-11 is counting VP-040 in paths but its canonical module is multipath.
- `internal/metrics` ARCH-11 says 4 with "proptest(2), integration(2)"; VP-INDEX has VP-027 (proptest), VP-047 (integration), VP-052 (integration) = 3. The second proptest entry cannot be identified from VP-INDEX.
- `internal/halfchannel` ARCH-11 says 6 with "proptest(5), benchmark(1)"; VP-INDEX has VP-016,017,018,051,053 (proptest=5) + VP-041,042 (benchmark=2) = 7. ARCH-11 says benchmark=1 but there are 2 benchmark VPs.
- `internal/config` ARCH-11 says 2 with "proptest(2)"; VP-INDEX has VP-028, VP-029 (both proptest) + VP-038 (e2e) = 3. VP-038 not counted.

---

## Axis 3 — BC ↔ CAP Traceability Sync

**Findings: 0 mismatches**

For all 42 BCs, the following four views are consistent:
- (a) `capability:` frontmatter field
- (b) `traces_to:` frontmatter field
- (c) Body "## Traceability" L2 Capability row (where present)
- (d) BC-INDEX CAP(s) column

All 42 BCs: `capability:` == `traces_to:` == body traceability row == BC-INDEX.

Special cases verified:
- BC-2.03.003: `capability: CAP-011`, `traces_to: [CAP-011, CAP-012]` — BC-INDEX lists "CAP-011, CAP-012"; the primary capability field (CAP-011) is consistent; traces_to correctly adds CAP-012 as secondary.
- BC-2.05.006: `capability: CAP-020b`, consistent throughout.
- BC-2.05.007: `capability: CAP-020a`, consistent throughout.

No Axis 3 defects.

---

## Axis 4 — CAP ↔ Realization Symmetry

**Findings: 0 mismatches**

All CAPs in capabilities.md have consistent reverse listings:
- capabilities.md "Realized by:" annotation (where present) matches BC-INDEX CAP Coverage Verification table
- BCs with matching `capability:` frontmatter agree with BC-INDEX
- Spot-checked: CAP-020 realized by BC-2.05.005 (capabilities.md and BC-INDEX agree); CAP-020a realized by BC-2.05.007; CAP-020b by BC-2.05.006

Note: Most CAPs in capabilities.md do not have an explicit "Realized by:" annotation — only CAP-020, CAP-020a, CAP-020b, and CAP-028 carry this. The absence of the annotation for other CAPs is not a defect (BC-INDEX carries the authoritative mapping).

No Axis 4 defects.

---

## Axis 5 — Wire Format Consistency

**Canonical values (from ARCH-02):**
- HMAC tag: 8 bytes (first 8 bytes of HMAC-SHA256 output)
- SVTN ID: 16 bytes
- Outer header: 44 bytes total

**Findings: 0 mismatches**

All spec files consistently use the ARCH-02 canonical values. Verification:

| Value | Files checked | Result |
|-------|--------------|--------|
| HMAC tag = 8 bytes | BC-2.01.004, BC-2.05.005, BC-2.05.006, BC-2.05.007, ARCH-02, ARCH-04, invariants.md (DI-006), capabilities.md (CAP-003, CAP-020) | All say 8 bytes / "first 8 bytes of HMAC-SHA256" |
| SVTN ID = 16 bytes | BC-2.01.004 (precondition 2), BC-2.01.006 (precondition 2), capabilities.md (CAP-003), ARCH-02, interface-definitions.md config schema | All say 16-byte / 128-bit |
| Outer header = 44 bytes | BC-2.01.004, BC-2.01.005, BC-INDEX, ARCH-02, ARCH-11, invariants.md (DI-007) | All say 44 bytes |
| Node address = 8 bytes | BC-2.01.006, ARCH-02 session identity section, capabilities.md (CAP-003) | All say 8 bytes / 64-bit |

No Axis 5 defects.

---

## Axis 6 — CLI Subcommand Reference vs Definition

**Findings: 1 defect**

CLI subcommands referenced in spec files were compared against the defined subcommands in interface-definitions.md.

Defined `sbctl` subcommands in interface-definitions.md:
- `sbctl svtn create`, `sbctl svtn destroy`, `sbctl svtn list`, `sbctl svtn status`
- `sbctl svtn keys register`, `sbctl svtn keys revoke`, `sbctl svtn keys list`, `sbctl svtn keys expire`
- `sbctl sessions list`, `sbctl sessions attach`, `sbctl sessions detach`, `sbctl sessions status`
- `sbctl paths list`, `sbctl paths ping`
- `sbctl router status`, `sbctl router metrics`, `sbctl router reload`, `sbctl router drain`
- `sbctl console attach`, `sbctl console detach`, `sbctl console switch`
- `sbctl admin register-key`, `sbctl admin revoke-key`, `sbctl admin recover`, `sbctl admin list-keys`
- `sbctl version`, `sbctl ping`

**Defect 6.1:** ARCH-04 (ADR-004) references `sbctl admin` with a "confirmation token" flow and `--confirm` flag, and mentions `sbctl admin recover` for bootstrap key emergency recovery. The `sbctl admin` table in interface-definitions.md shows different flag names:
- interface-definitions.md uses `--svtn <id> --pubkey <path>` flags  
- ARCH-04 narrative refers to "confirmation token from an offline operator key" for split-brain recovery, which corresponds to `sbctl admin recover --svtn <id> --bootstrap-key <path>`
- These are consistent in subcommand names but ARCH-04 uses `--confirm` flag terminology not defined in interface-definitions.md

**Minor references (no definition gap):** `sbctl router drain`, `sbctl paths list`, `sbctl sessions list` are all defined. The reference "sbctl status against" (extracted as a fragment) is from narrative text, not a subcommand invocation. `sbctl switch` fragment is from `sbctl console switch` — defined.

**Net: 1 minor defect** (ARCH-04 mentions `--confirm` flag not defined in interface-definitions.md). All named subcommands are defined.

---

## Axis 7 — Error Code Reference vs Definition

**Findings: 2 defects**

Error codes defined in error-taxonomy.md: E-ADM-001 through E-ADM-011, E-CFG-001 through E-CFG-005, E-NET-001 through E-NET-006, E-PRT-001 through E-PRT-003, E-FWD-001, E-SES-001, E-SVTN-001 through E-SVTN-002, E-SYS-001.

Total defined: 30 error codes.

All error codes referenced across all spec files: E-ADM-001–011, E-CFG-001–005, E-NET-001–006, E-PRT-001–003, E-FWD-001, E-SES-001, E-SVTN-001–002, E-SYS-001 = 30 codes.

Set comparison: referenced set == defined set. No codes referenced but not defined. No codes defined but never referenced.

**Defect 7.1 — E-ADM-007 semantic contradiction:**
- error-taxonomy.md defines E-ADM-007 as: "upstream rejected: read-only access for console `<key_fingerprint>` on session `<session_name>`" — sourced from BC-2.04.005 (read-only upstream rejection)
- ARCH-04 (ADR-004) uses E-ADM-007 in a different semantic context: "Any revocation operation by a console-role key on a control-role key is rejected with E-ADM-007"
- The correct code for the hierarchy violation (console cannot revoke control) is **E-ADM-011**, defined as: "permission denied: `<role>` key cannot revoke `<target_role>` key (control > console > readonly)"
- **ARCH-04 cites the wrong error code for the revocation hierarchy violation.** E-ADM-007 = upstream rejection; E-ADM-011 = permission hierarchy violation for revocation.

**Defect 7.2 — Error codes only in "never referenced" pass:**
All 30 defined codes are referenced at least once in the spec files. However, E-CFG-002 appears only in error-taxonomy.md itself (sourced from BC-2.05.007) and does not appear in any BC body, VP, or ARCH file as a raised condition. The taxonomy entry for E-CFG-002 says "private key export not supported: `<reason>`" — it is a valid code but has no tracing back to a BC precondition or edge case that would trigger it. This is a coverage gap, not an error-code/taxonomy defect.

---

## Axis 8 — Module ↔ Subsystem Coherence

**Findings: 1 defect**

ARCH-INDEX Subsystem Registry maps subsystems to implementing modules. BC frontmatter has `subsystem:` and `architecture_module:` fields. ARCH-05 provides the authoritative BC→package mapping table.

All 42 BCs: `subsystem:` frontmatter matches the BC-INDEX subsystem column exactly. All 42 BCs: `architecture_module:` frontmatter matches the ARCH-05 BC→Architecture Module table, with one exception:

**Defect 8.1 — BC-2.01.005 architecture_module inconsistency:**
- BC-2.01.005 frontmatter: `architecture_module: internal/frame`
- ARCH-05 BC→module table: BC-2.01.005 → `internal/frame` (module name "frame")
- ARCH-11 BC→VP coverage table module column: BC-2.01.005 → `internal/routing`

The BC itself is about channel header opacity to routers. The verification target (VP-015: "Router code never parses channel header payload") lives in `internal/routing`. The BC contract is authored from the `internal/frame` perspective (it defines the channel header format), but the primary verification target is the router-side enforcement. ARCH-11 and ARCH-05 disagree on which module owns this BC.

VP-015 frontmatter (`module: internal/routing`) is consistent with ARCH-11 and with what VP-015 actually tests.

All other BCs: subsystem and architecture_module are consistent across all views. All subsystem names in BC frontmatter are valid SS-NN identifiers from the ARCH-INDEX Subsystem Registry. All `architecture_module` values correspond to Go packages owned by the BC's subsystem (per ARCH-INDEX).

---

## Axis 9 — Open Questions / ADR Resolution

**Findings: 1 partial finding**

Open questions from invariants.md:
- OQ-001: Console node key registration scope
- OQ-002: Access node key management capability
- OQ-003: Permission hierarchy among key roles
- OQ-004: Downstream half-channel state continuity on path failover

ADR resolutions found in ARCH files:
- OQ-001: Resolved in ARCH-04, ADR-004 section: "OQ-001 resolution: Console nodes cannot register new Tier 1 admission keys."
- OQ-002: Resolved in ARCH-04, ADR-004 section: "OQ-002 resolution: Access nodes have no key management capability whatsoever."
- OQ-003: **Partially resolved.** ARCH-04 ADR-004 section header reads "Console Key Registration Model and Permission Hierarchy (OQ-001, OQ-002, OQ-003, F-010)" and defines the `control > console > readonly` hierarchy. However, there is no explicit "OQ-003 resolution:" paragraph in the body — only an "OQ-002 note:" paragraph and a description of the hierarchy. The resolution of OQ-003 ("Is there a permission hierarchy among key roles?") is implicit in the hierarchy definition but lacks the explicit resolution marker used for OQ-001 and OQ-002.
- OQ-004: Resolved in ARCH-03, ADR-005 section: "OQ-004 resolution: Resolves the open question in invariants.md OQ-004 — downstream switchover continuity."

**Defect 9.1:** OQ-003 has no explicit "OQ-003 resolution:" paragraph in ARCH-04 despite the section header claiming to resolve it. The resolution is implied by the hierarchy definition but should be made explicit to close the invariants.md open question.

No ADR cites a non-existent OQ ID. All four OQs (OQ-001 through OQ-004) are addressed in ARCH files.

---

## Axis 10 — ARCH-11 / ARCH-07 VP Count Accuracy

**Findings: 7 count discrepancies**

### 10.1 ARCH-11 per-module count table vs VP-INDEX canonical

VP-INDEX is the authoritative count (57 total).

| Module | ARCH-11 count | ARCH-11 method breakdown | VP-INDEX actual count | VP-INDEX actual methods | Delta |
|--------|--------------|--------------------------|----------------------|------------------------|-------|
| internal/halfchannel | 6 | proptest(5), benchmark(1) | 7 | proptest(5), benchmark(2) | ARCH-11 undercounts by 1; also methods wrong (benchmark=1 should be benchmark=2) |
| internal/paths | 2 | proptest(1), e2e(1) | 1 | proptest(1) | ARCH-11 overcounts by 1; VP-040 (e2e) belongs to internal/multipath not paths |
| internal/metrics | 4 | proptest(2), integration(2) | 3 | proptest(1), integration(2) | ARCH-11 overcounts by 1; one proptest entry unaccountable |
| internal/config | 2 | proptest(2) | 3 | proptest(2), e2e(1) | ARCH-11 undercounts by 1; VP-038 (e2e) omitted |

The net sum stays 57 because the over/under-counts cancel (+2 overcounts, +2 undercounts).

### 10.2 ARCH-07 VP catalog vs VP-INDEX

ARCH-07 P0 catalog lists VP-001 through VP-015 (15 VPs).
ARCH-07 P1 catalog lists VP-016 through VP-030 (15 VPs).
ARCH-07 Test-Sufficient catalog lists VP-031 through VP-042 (12 VPs).
Phase 1c additions: VP-043–VP-050, VP-052–VP-056, VP-053, VP-057.

**Discrepancy 10.2.1:** ARCH-07 P0 table lists VP-005 method as "proptest/fuzz". VP-INDEX lists VP-005 method as "fuzz". The VP-005.md frontmatter should be the arbiter — not checked here as source, but VP-INDEX is canonical per its own header.

**Discrepancy 10.2.2:** ARCH-07 P1 table lists VP-028 method as "unit". VP-INDEX lists VP-028 method as "proptest". This is a method category disagreement between ARCH-07 and VP-INDEX.

**Discrepancy 10.2.3:** ARCH-07 P0 table lists VP-015 method as "fuzz + code audit". VP-INDEX lists VP-015 method as "fuzz". Method label disagrees (ARCH-07 more detailed).

**Discrepancy 10.2.4:** ARCH-07 Test-Sufficient table stops at VP-042, then continues in a separate "Phase 1c-refinement" section with VP-043–VP-057. This creates an inconsistent catalog structure where VP-043–VP-057 have mixed phase assignments:
- VP-043 is in ARCH-07 "Test-Sufficient Properties Added in Phase 1c-refinement" but VP-043.md frontmatter (source_bc=BC-2.02.007) and VP-INDEX classify it as `proptest` P1.
- ARCH-07 classifies VP-053 and VP-057 as "P0 Properties Added in Phase 1c-refinement" but lists them under a separate section header rather than in the main P0 table.
- ARCH-07 classifies VP-051, VP-054, VP-055 as "P1 Properties Added in Phase 1c-refinement" but VP-054 (integration, source_bc=BC-2.02.002) is also listed in VP-INDEX as integration (consistent).

**Discrepancy 10.2.5:** ARCH-11 Coverage Summary table:

| ARCH-11 says | VP-INDEX canonical |
|-------------|-------------------|
| P0 VPs: 39 | 39 |
| P1 VPs: 14 | 14 |
| P2+ VPs: 4 | 4 |
| Total: 57 | 57 |

Phase distribution counts match. No defect here.

**Discrepancy 10.2.6:** ARCH-11 BC-2.01.001 row shows VP-016, VP-018 only. VP-INDEX shows BC-2.01.001 covering VP-016, VP-018, VP-041, VP-042 (four VPs). ARCH-11 omits VP-041 and VP-042 from BC-2.01.001's row despite VP-041 and VP-042 having source_bc=BC-2.01.001.

**Discrepancy 10.2.7:** ARCH-11 BC-2.05.001 row shows VP-007, VP-008, VP-009. But VP-008's source_bc=BC-2.05.002, not BC-2.05.001. ARCH-11 lists VP-008 under BC-2.05.001 because both BCs are semantically related (admission chain), but this misrepresents the formal VP→BC traceability as established by source_bc frontmatter.

---

## Aggregate

### 1. Count of unique defects per axis

| Axis | Description | Finding Count |
|------|-------------|--------------|
| Axis 1 | VP ↔ BC title sync | 44 VP files with at least 1 mismatch; ~42 distinct VP-level mismatches (34 substantive, 5 truncations, 3 casing) |
| Axis 2 | BC ↔ VP coverage symmetry | 8 distinct mismatches (4 source_bc vs ARCH-11, 3 BC body cross-refs, 4 per-module count errors) |
| Axis 3 | BC ↔ CAP traceability | 0 |
| Axis 4 | CAP ↔ realization symmetry | 0 |
| Axis 5 | Wire format consistency | 0 |
| Axis 6 | CLI subcommand reference vs definition | 1 (ARCH-04 uses undefined `--confirm` flag) |
| Axis 7 | Error code reference vs definition | 2 (E-ADM-007 wrong code in ARCH-04; E-CFG-002 unreferenced in BC bodies) |
| Axis 8 | Module ↔ subsystem coherence | 1 (BC-2.01.005 ARCH-05 vs ARCH-11 module disagreement) |
| Axis 9 | OQ resolution | 1 (OQ-003 implicit but not explicitly marked resolved) |
| Axis 10 | ARCH-11/ARCH-07 VP count accuracy | 7 (4 module count errors, 3 method label discrepancies) |
| **Total** | | **~64 distinct defects** |

### 2. Highest-leverage fix targets

**Fix A — Regenerate all VP Source Contract sections from BC-INDEX canonical titles (resolves ~42 Axis 1 defects)**
Every VP file's "## Source Contract" BC title was authored independently rather than pulled from BC-INDEX. A mechanical pass updating all VP Source Contract BC title citations to exactly match the BC-INDEX `Title` column would close all 42 Axis 1 findings in one operation.

**Fix B — Correct ARCH-11 per-module VP count table (resolves 4 Axis 2 + 4 Axis 10 count defects)**
Four module counts are wrong: halfchannel (6→7), paths (2→1), metrics (4→3), config (2→3). A single recount pass on ARCH-11 closes 8 defects. Additionally, VP-041 and VP-042 should be added to the ARCH-11 BC-2.01.001 row.

**Fix C — Correct ARCH-04 E-ADM-007 to E-ADM-011 (resolves 1 Axis 7 defect)**
One-line change in ARCH-04 ADR-004 section: "rejected with E-ADM-007" → "rejected with E-ADM-011". This is a semantic correctness fix — wrong error code cited for permission hierarchy violation.

**Fix D — Add OQ-003 explicit resolution paragraph to ARCH-04 (resolves 1 Axis 9 defect)**
Add "**OQ-003 resolution:** A permission hierarchy exists among key roles: control > console > readonly. Lower-tier roles cannot revoke higher-tier roles." to ARCH-04 ADR-004 section.

**Fix E — Resolve BC-2.01.005 module in ARCH-11 or ARCH-05 (resolves 1 Axis 8 defect)**
Either: update ARCH-11 BC-2.01.005 module column to `internal/frame` (matching BC frontmatter and ARCH-05), or update BC-2.01.005 frontmatter architecture_module to `internal/routing` (matching what VP-015 actually tests). Decision required: is BC-2.01.005's "primary module" the format definer (frame) or the enforcement target (routing)?

### 3. Structural vs individual defect ratio

- **Structural defects** (a single root cause produces N individual findings): Axis 1 (42 findings from one root cause: VP Source Contract sections not bound to BC-INDEX canonical titles) + Axis 10 module count table (4 findings from stale count table) = ~46 structural findings from 2 structural root causes.
- **Individual defects** (one-off per-document errors): Axis 7 E-ADM-007 semantic contradiction (1), Axis 8 BC-2.01.005 module disagreement (1), Axis 9 OQ-003 implicit resolution (1), Axis 6 --confirm flag (1), Axis 7 E-CFG-002 unreferenced (1), Axis 10 method label discrepancies (3), Axis 2 VP-008 in wrong BC row (1) = ~9 individual defects.

**Ratio:** ~46 structural / ~18 individual ≈ 72% of defects stem from structural (systemic) root causes. Fixing 2 structural root causes (VP Source Contract generation policy + ARCH-11 recount) would close ~72% of all findings.

---

*End of audit. Total spec files read: 57 VP files + 42 BC files + 11 ARCH files + 4 domain-spec/prd-supplement files + BC-INDEX + VP-INDEX + ARCH-INDEX = 117 files.*
