---
artifact_id: BC-2.03.003
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.03.003
subsystem: session-discovery
architecture_module: internal/discovery
capability: CAP-011
priority: P1
criticality: high
scope_phase: PE
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified: []
deprecated: null
deprecated_by: null
replacement: null
retired: null
removed: null
removal_reason: null
inputDocuments:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '_bmad-output/planning-artifacts/prd.md'
traces_to: [CAP-011, CAP-012]
kos_anchors:
  - elem-node-router-architecture
---

# Behavioral Contract BC-2.03.003: Presence Advertisement Includes Session Name, Attachment Status, and Quality Indicator

## Description

Each presence advertisement payload carries structured metadata about each published session: the session name (as known to tmux), the attachment status (attached/unattached), and the quality indicator (green/yellow/red). This is the minimum metadata required for the console to display a useful session list and for operators to make attach/observe decisions without connecting to each session.

## Preconditions

1. Access node has at least one session to advertise.
2. Each session has a name (from tmux session name), attachment status (derived from current console subscriptions), and quality indicator (from per-path metrics).

## Postconditions

1. Each session entry in the advertisement contains exactly: {session_name: string, attached: bool, quality: green|yellow|red}.
2. Session names are UTF-8 encoded, maximum 255 bytes.
3. Quality indicator reflects the current path quality as of the last metric update.
4. Attachment status reflects whether at least one full-access console is currently attached.
5. The advertisement does not contain: IP addresses, hostnames, internal node identifiers, or session content.

## Invariants

1. Session metadata does not include session content (invariant from DI-001).
2. The quality indicator is a derived field — the actual per-path metrics are in the quality-observability subsystem.
3. "Attached" means at least one full-access console is attached. Read-only observers do not change attached status.

## Trigger

Advertisement assembly at the access node before multicast dispatch.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Session name contains non-ASCII characters (e.g., Japanese) | UTF-8 encoded; max 255 bytes enforced. If tmux session name exceeds 255 bytes (unusual), it is truncated with "…" indicator. |
| EC-002 | Session quality indicator not yet computed (first advertisement) | Quality defaults to green (optimistic start); updates on first keep-alive measurement. |
| EC-003 | Session has only read-only observers, no full-access console | attached=false. Read-only observers are subscribers but not "attached" in the session-management sense. |
| EC-004 | Access node loses tmux control mode (FM-004) | Quality indicator moves to yellow or red; attachment status becomes unknown (advertised as "unknown" or last known value). |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Session "agent-01", full-access console attached, path RTT=15ms | {name:"agent-01", attached:true, quality:green} | happy-path |
| Session "agent-02", no console, path RTT=180ms | {name:"agent-02", attached:false, quality:yellow} | happy-path |
| Session name "日本語セッション" (19 UTF-8 bytes) | {name:"日本語セッション", attached:false, quality:green} | edge-case |
| Session name 256 bytes long | {name: first 252 chars + "…", attached:false, quality:green} | edge-case |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-055 | Presence advertisement payload round-trips through Encode/Decode; all 3 fields present and stable; invalid names rejected | proptest |
| VP-045 | Console session enumeration without hostnames (verifies no internal addresses in advertisement) | e2e |
| VP-055 | Quality field is always one of: green, yellow, red (verified in VP-055 proptest over QualityIndicator enum) | proptest |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-011 ("Multicast presence advertisement") per capabilities.md §CAP-011; CAP-012 ("Console session enumeration across SVTN") per capabilities.md §CAP-012 |
| L2 Domain Invariants | DI-001 (carrier-grade content separation — advertisement contains no session content) |
| Architecture Module | internal/discovery |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-011 ("Multicast presence advertisement") per capabilities.md §CAP-011 — this BC specifies the advertisement payload contents that CAP-011 defines as "attachment status and quality indicators"; also CAP-012 which requires "session name, attachment status, and quality indicator" per capabilities.md §CAP-012 |

## Related BCs

- BC-2.03.001 — depends on: this BC defines the payload that BC-2.03.001 broadcasts
- BC-2.06.001 — related to: quality indicator in advertisements is derived from this subsystem
