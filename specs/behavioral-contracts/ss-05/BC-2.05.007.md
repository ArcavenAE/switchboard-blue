---
artifact_id: BC-2.05.007
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.05.007
subsystem: SS-TBD
capability: CAP-020
priority: P0
criticality: critical
scope_phase: E
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
traces_to: [CAP-020]
kos_anchors:
  - elem-ssh-end-to-end-encryption
---

# Behavioral Contract BC-2.05.007: Node Private Keys Never Transit the Network Under Any Condition

## Description

A node's private SSH key is never serialized, transmitted, or included in any wire-format message, frame, log entry, diagnostic output, or error response. This invariant holds under all conditions: normal operation, error states, diagnostic modes, key management operations. Public keys transit as required (for admission challenges, key registration). The private key is used only for local signing operations.

## Preconditions

1. Any node operation that involves cryptographic authentication.
2. Any key management operation (register, revoke, export diagnostics).

## Postconditions

1. The private key bytes are not present in any outgoing network frame.
2. The private key bytes are not present in any log output.
3. The private key bytes are not present in any sbctl command output.
4. Any code path that reads the private key must not pass it to any network I/O function, serializer, or logger.

## Invariants

1. **DI-002**: This invariant is unconditional. There is no mode, debug flag, or operator command that causes private keys to transit.
2. The private key file path may appear in logs (for error diagnosis); the private key content does not.
3. Public keys (fingerprints, full public key) may appear in diagnostics and logs; only the private key is protected by this invariant.

## Trigger

Any operation involving the private key: admission challenge signing, HMAC computation, key management.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Operator runs `sbctl debug --export-keys` | Command succeeds for public key export only. Private key is never included in any export output. If private key export is requested, E-CFG-002 "private key export not supported". |
| EC-002 | Crash dump / core dump | Implementation must not include private key material in crash reports. Private key material should be kept in memory regions marked non-dumpable (OS-specific; implementation detail). |
| EC-003 | Error in HMAC computation; error logged | Log entry includes error type and SVTN ID; never includes the key bytes. |
| EC-004 | Diagnostic trace mode enabled | Even in maximum verbosity trace mode, private key bytes are never output. This must be enforced by code review, not configuration. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Node sends admission challenge response | Wire capture: challenge nonce + signature only; no private key material | happy-path |
| Operator runs `sbctl keys list` | Output: key fingerprints, roles, expiry dates. No private key bytes. | happy-path |
| Simulated private key exfiltration path (code audit) | No code path exists that could serialize private key to network or log | property |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-TBD | Private key bytes do not appear in any network frame | code-audit/fuzzing |
| VP-TBD | Private key bytes do not appear in any log output | code-audit |
| VP-TBD | No API, CLI, or diagnostic mode returns private key material | code-audit |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-020 ("HMAC frame authentication at router boundary") per capabilities.md §CAP-020 |
| L2 Domain Invariants | DI-002 (node private keys never transit the network) |
| Architecture Module | [filled by architect] |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-020 ("HMAC frame authentication at router boundary") per capabilities.md §CAP-020 — private key non-transit is the key management invariant that underlies the HMAC trust model; also directly enforces DI-002 which is grounded in PRD FR39 and the cryptographic standards domain |

## Related BCs

- BC-2.05.001 — related to: admission challenge signing is the operation that uses the private key
- BC-2.05.005 — related to: HMAC computation uses the admission key; this BC ensures the key stays local
