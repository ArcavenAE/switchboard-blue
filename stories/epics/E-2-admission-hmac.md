---
artifact_id: E-2-admission-hmac
document_type: epic
level: ops
epic_id: E-2
version: "1.0"
status: pending
producer: story-writer
timestamp: 2026-06-24T00:00:00
phase: 2
scope_phase: E
priority: P0
bc_traces:
  - BC-2.05.001
  - BC-2.05.002
  - BC-2.05.005
  - BC-2.05.006
  - BC-2.05.007
subsystems: [admission-security]
architecture_modules: [internal/hmac, internal/admission, internal/routing]
inputDocuments:
  - '.factory/specs/behavioral-contracts/BC-INDEX.md'
  - '.factory/specs/architecture/ARCH-04-admission-security.md'
---

# E-2: Admission + HMAC (Security Foundation)

## Goal

Establish the SVTN trust boundary. Deliver HMAC-SHA256 frame authentication
with per-(node, SVTN) HKDF-derived keys, tier-1 signed-challenge admission, and
SVTN cryptographic isolation. Routers must fail-closed: no forwarding without admission.

## BCs

| BC | Title | Priority |
|----|-------|---------|
| BC-2.05.001 | Tier 1 SVTN admission via signed key challenge | P0 |
| BC-2.05.002 | Router rejects non-admitted nodes before forwarding — fail-closed | P0 |
| BC-2.05.005 | HMAC frame authentication at first router boundary | P0 |
| BC-2.05.006 | SVTN cryptographic isolation: admitted node on SVTN-A cannot see SVTN-B traffic | P0 |
| BC-2.05.007 | Node private keys never transit the network under any condition | P0 |

## Subsystems Touched

- SS-05 admission-security (primary)

## Estimated Stories

2 stories: S-2.01 (HMAC codec), S-2.02 (admission + SVTN isolation)
