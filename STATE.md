---
pipeline: IN_PROGRESS
phase: phase-1-spec-crystallization
phase_step: pending-create-architecture
product: switchboard
mode: greenfield
anchor_strategy: reference-via-frontmatter
l2_complete: true
l2_artifact_count: 11
l2_subsystems: [session-networking, multipath-forwarding, session-discovery, session-access, admission-security, quality-observability, network-management, console-operations, deployment-operations]
l3_complete: true
l3_bc_count: 42
l3_cap_coverage: "27/27"
l3_error_codes: 31
l3_bc_id_scheme: "BC-2.SS.NNN — S=2 stable L3-PRD prefix, SS=subsystem 01-09, NNN=sequence"
l3_deferred_decisions: ["tick-interval-range", "fec-group-size", "presence-heartbeat", "hmac-algorithm", "e-router-no-discovery-ux"]
l3_subsystem_field_status: "SS-TBD pending architect formalization"
timestamp: 2026-06-23T19:25:54Z
last_update: 2026-06-23
---

# Switchboard Factory State

## Current phase

**Phase 1 — Spec Crystallization** (entered 2026-06-23 after artifact-detection
discovery).

Next step: `/vsdd-factory:create-domain-spec` (L2 domain spec) →
`/vsdd-factory:create-prd` (L3 BC-S.SS.NNN) → `/vsdd-factory:create-architecture`
→ Phase 1d adversarial spec review → human approval gate.

## Source-of-truth inputs

Reference-via-frontmatter strategy. BMAD docs and KoS nodes remain
authoritative; `.factory/specs/` will derive from them via
`inputDocuments:` frontmatter.

- `_bmad-output/planning-artifacts/product-brief-switchboard-2026-03-31.md` — L1 brief
- `_bmad-output/planning-artifacts/prd.md` — L2/L3 source material (BMAD format)
- `_bmad-output/brainstorming/*` — 3 sessions (architecture, naming, session cache)
- `_kos/nodes/bedrock/` — 7 architectural bedrock nodes
- `_kos/nodes/frontier/` — open questions

## Discovery artifacts

- `.factory/planning/artifact-inventory.md`
- `.factory/planning/gap-analysis.md`
- `.factory/planning/routing-decision.md`

## Deferred decisions

- **Tick interval range [5ms, 50ms]** — derived from BMAD ASM-001. Needs empirical validation before hardening; E-CFG-001 fires outside range. Surfaces at Phase 1 gate.
- **FEC group size N=4** — XOR parity default; not measurement-grounded. Architecture phase should treat as tunable.
- **Presence heartbeat 30s** — chosen as reasonable default; no hard requirement. Confirm against discovery freshness SLA.
- **HMAC algorithm SHA-256** — may need alignment with Noise handshake key material; architecture decision.
- **E-router has no session discovery (scope_phase: PE)** — MVP consoles must specify access node address manually. Confirm acceptable.

## Non-blocking debt

- `.factory/.gitignore` not bootstrapped (drbothen/vsdd-factory#230 + this-session comment).
