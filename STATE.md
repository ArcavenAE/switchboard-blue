---
pipeline: IN_PROGRESS
phase: phase-1-spec-crystallization
phase_step: pending-adversary-pass-3
refinement_round_2_complete: true
product: switchboard
mode: greenfield
anchor_strategy: reference-via-frontmatter
l2_complete: true
l2_artifact_count: 11
l2_subsystems: [session-networking, multipath-forwarding, session-discovery, session-access, admission-security, quality-observability, network-management, console-operations, deployment-operations]
l3_complete: true
l3_bc_count: 42
l3_cap_coverage: "29/29"
l3_cap_count: 29
l3_error_codes: 31
l3_bc_id_scheme: "BC-2.SS.NNN — S=2 stable L3-PRD prefix, SS=subsystem 01-09, NNN=sequence"
l3_subsystem_field_status: "patched — all 42 BCs have canonical subsystem + architecture_module fields"
l4_complete: true
l4_vp_count: 57
l4_bc_coverage: "42/42"
refinement_round_1_complete: true
arch_sections: 13
arch_adrs: 8
dtu_required: false
dtu_justification: "MVP single-LAN; no third-party SaaS deps. PE phase may need STUN/TURN DTU."
dtu_assessment: 2026-06-23
dtu_clones_built: n/a
dtu_services: []
feasibility_status: "all-feasible"
cicd_setup_complete: true
cicd_workflow_count: 6
cicd_p0_gaps: 3
cicd_p1_gaps: 2
cicd_p2_gaps: 5
internal_packages: 18
purity_distribution: {pure_core: 9, boundary: 5, effectful: 4}
go_verification_toolchain: ["go test", "go test -race", "go test -fuzz", "golangci-lint", "staticcheck", "go-mutesting"]
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

- RESOLVED: **HMAC algorithm** — HMAC-SHA256 with 16-byte truncated tag, HKDF-SHA256 per-SVTN key derivation (ADR-001, ARCH-02/04)
- RESOLVED: **FEC group size** — N=4 default (20% overhead); tunable (ADR-002, ARCH-03). Phase 3 validates default empirically.
- RESOLVED: **Duplicate key registration** — last-write-wins (ADR-003, ARCH-04). Operator controls last write.
- RESOLVED: **Console/access key permissions** — control > console > access; only control nodes register keys (ADR-004, ARCH-04)
- RESOLVED: **Downstream ARQ failover** — resync from last ACK; in-flight frames during failover are lost (ADR-005, ARCH-03). Stateful transfer deferred to PE.
- **Tick interval range [5ms, 50ms]** — still empirical (ADR-008 keeps as tuning parameter). Validates in Phase 3.
- **Presence heartbeat 30s** — discovery is scope_phase PE, not MVP. Defer.
- **Marvel integration** — `_kos/nodes/frontier/question-marvel-integration.yaml` is acknowledged in `bounded-contexts.md` as out of scope. Now explicitly deferred — no MVP integration, no PE-phase integration. Re-evaluate post-MVP if marvel project publishes a stable interface. (resolves adversary F-024)
- ✓ **HMAC keying** → RE-RESOLVED with amended ADR-001: per-(node, svtn) HKDF derivation using node_admission_pubkey as IKM (was per-SVTN). Restores per-node trust boundary the BCs require.
- ✓ **Outer header layout** → AUTHORITATIVE (ARCH-02): 44 bytes exactly: version(1), frame_type(1), payload_len(2), svtn_id(16), src_addr(8), dst_addr(8), hmac_tag(8). Sequence lives in channel header only.
- ✓ **HMAC tag size** → 8 bytes (truncated from 32-byte HMAC-SHA256). 64-bit MAC sufficient for the rate-limited threat model; document for next adversary pass to verify.
- ✓ **Hash function** → SHA-256 stdlib (no Blake3 transitive dep).
- ✓ **Drop cache** → keyed on (checksum, arrival_interface_id) — fixes dup-and-race conflict.
- ✓ **Quality thresholds canonical** → 100/500ms RTT, 5%/20% loss, hysteresis 3.
- ✓ **OQ-003 permission hierarchy** → ADR-004 expanded: console cannot revoke control; control-to-control revocation requires `sbctl admin` human authorization.

## KoS frontier questions surfaced in Phase 1b

- Q: Does router-to-router PE phase need Noise XX mutual auth in addition to HMAC?
- Q: Should SACK bitmap window be configurable (64-bit default may be too narrow for PE high-latency links)?
- Q: Goroutine model for 1k concurrent sessions — per-session pair vs event-loop (NFR-004)?
- Q: Drop cache — TTL eviction in addition to LRU to prevent suppression after wraparound?
- Q: PE router-to-router Noise — share node admission keypair, or separate router identity?
- F-027 [process-gap] — 4 of 6 kos frontier files have empty `content:` blocks (`question-asymmetric-channels`, `question-encryption-model`, `question-marvel-integration`, `question-timeslice-framing`). Lint at kos-edge creation time should disallow empty content. Filed upstream.

## Phase 3 blockers (must resolve before TDD implementation)

- **P0-001 — Branch protection missing on `develop`.** `ci.yml` runs but is not a required check. PR with failing tests can merge. Undermines TDD. Fix: enable branch protection requiring `ci` check + 1 approving review + dismiss-stale-reviews + restrict-push.
- **P0-002 — Branch protection missing on `main`.** Stable release branch unprotected; force-push possible. Fix: same as P0-001 plus restrict-push to release tags only.
- **P0-003 — Commit signature enforcement absent at repo level.** Global gitconfig enforces signing locally, but GitHub does not reject unsigned bot commits. Fix: after enabling branch protection, set `required_signatures: true` on both branches.

Full CI/CD inventory, P0 remediation steps, and P1/P2 gaps: `.factory/specs/cicd-setup.md`.

## Non-blocking debt

- `.factory/.gitignore` not bootstrapped (drbothen/vsdd-factory#230 + this-session comment).

## Adversary cycle-1 metrics

- Pass 1 findings: 27 (5 critical, 11 high, 9 medium, 2 low; 3 process-gap tagged)
- Cycle 1 refinement: 5 critical + 11 high + 7 medium + 1 low addressed = 24 in-cycle; 2 process-gap deferred to upstream (F-025, F-027); 1 low deferred (covered by BA sweep).
- Convergence target: 3 consecutive zero-findings passes per FACTORY rules.
- Full findings: `.factory/cycles/cycle-1/adversarial-reviews/pass-01.md`
- Pass 2 findings: 18 (3 critical, 8 high, 6 medium, 1 low; 2 process-gap)
- Cycle 1 round-2 refinement: 17 in-cycle (3 critical + 8 high + 6 medium addressed); F-019 (1 low) by-design at Phase 1d, deferred to Phase 2 backfill rule.
- Trajectory: 27 → 18 → ? (Pass 3 pending; convergence target = 3 consecutive zero-findings passes)
- Full findings: `.factory/cycles/cycle-1/adversarial-reviews/pass-02.md`
