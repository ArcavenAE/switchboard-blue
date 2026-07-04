---
pass: 39
lane: A
scope: public-surface + operator-UX
develop_head: 6deda15
factory_head_pre_review: e51d4aa
adversary_id: adv-a-pass-39
dispatched_at: 2026-07-04T21:00:00Z
concluded_at: 2026-07-04T21:00:00Z
verdict: NO_FINDINGS
findings_count:
  critical: 0
  high: 0
  medium: 0
  low: 0
  process_gap: 0
observations: 1
anti_findings: 9
novelty: LOW
reviewed_at: 2026-07-04
---

# Phase 5 Pass 39 — Adv-A Lane (public-surface + operator-UX)

## Verdict

**NO_FINDINGS** — Fresh-context re-derivation of the operator-visible perimeter surfaces zero defects. Third consecutive Adv-A clean pass (P37 → P38 → P39). Novelty LOW.

## Lens

Adv-A operates from the SRE-operator / third-party integrator perspective: what does an operator actually observe when they run `sbctl`, hit an error, or integrate against the JSON envelope? Only artifacts an outside consumer can observe are in scope. Governance-internal drift is Adv-B territory and was not read.

## Anti-Findings

- AF-1 [preflight-tuple-git-ref-reconciliation] `.git/refs/heads/develop` = `6deda15def9326f28e96f133e237aff5ecb74d7b` matches dispatched `develop_head=6deda15`; `.git/refs/heads/factory-artifacts` = `e51d4aa560b38e921fadd0a9c134ae21c6ccdfae` matches dispatched `factory_head_pre_review=e51d4aa`. Adopted O-P5P38-META-001 recommended remediation pattern (git-ref cat rather than STATE.md frontmatter read); reconciled clean on first attempt.
- AF-2 [F-P5P36-A-001 phantom-code redirect persistent] `.factory/decisions/wave-6-tranche-a-scope-rulings.md` v1.14 §Ruling-12 §1 L1119-L1120: E-RPC-004 → E-RPC-010 phantom-code redirect remains in place; taxonomy catalog `.factory/specs/prd-supplements/error-taxonomy.md` v4.7 RPC row L193-L197 anchors E-RPC-010 as canonical "unknown command" server-side dispatch code emitted directly on wire without E-RPC-011 wrap; server-side emission `internal/mgmt/mgmt.go` L678-L680 confirmed. No regression.
- AF-3 [F-P5P36-A-002 sibling-sweep persistent across four sites] Combined-footnote pattern preserved at Ruling-11 §1 L1021, Ruling-11 AC-004 L1035, Ruling-12 §1 L1120 (combined with F-P5P36-A-001), Ruling-12 transport-exception sentence L1129. Story-body Universality amendment S-6.07 L78 v1.14 preserved. POL-002 story-index-row-sync STORY-INDEX v3.80 L74 preserved (title includes "(v1.14)").
- AF-4 [CLI exit-code discrimination baseline] `cmd/sbctl/main.go` L100-L111 `errors.As(err, &ue)` usageError → exit 2; else exit 1. Bare `sbctl` → subcommand list + exit 2. `--help/-h` → stdout + exit 0. `paths` bare vs unknown sub-verb → exit 2. `sessions.list` etc. → E-BL routing + exit 2. All confirmed against interface-definitions.md v1.29 §174.
- AF-5 [jsonEnvelope shape stable] `cmd/sbctl/client.go` L95-L101 jsonEnvelope struct: `ok`, `error{code,message}`, `data`. No `$schema` field. Matches interface-def v1.29 §JSON envelope exemplar.
- AF-6 [confirm-gate 5-path holds] `cmd/sbctl/admin.go` L362-L403 `runDestroyConfirmGate` covers: flag-matches, TTY-prompt, non-TTY-without-flag → E-CFG-013, `--yes` bypass, `--yes` + `--confirm` mutex → E-CFG-012. Register uses gate with `targetFlag="--svtn"`; destroy with `--name`; revoke has F-P5P11-A-001 boolean-confirm carve-out via `boolStringFlag`.
- AF-7 [Registered Verbs table byte-parallel] interface-def v1.29 §397-§411 six admin verbs + `paths.list`, `router.metrics`, `router.status`, `sessions.list`, `ping`, `version` — row-for-row matched against `cmd/switchboard/admin_handlers.go` BuildAdminHandlers dispatch.
- AF-8 [handler emissions byte-identical against taxonomy v4.7] E-INT-999 admin_handlers.go L458 → mint text canonical; E-ADM-009 L483/L550/L564/L571/L599/L621/L625 → "has role unregistered" canonical (bootstrap-only pre-check + any-role variants); E-SVTN-001 L642 no stutter; E-SVTN-003 no stutter; E-ADM-013 no stutter; E-CFG-001 canonical. Decode-limit E-RPC-002 stamp at `cmd/sbctl/client.go` L198-L265 + L278-L309 verified.
- AF-9 [factory_head verification hygiene — new anti-finding this pass] `.git/refs/heads/factory-artifacts` = `e51d4aa560b38e921fadd0a9c134ae21c6ccdfae` correctly resolved via git ref file, NOT via STATE.md frontmatter `develop_head:` line (which anchors develop, not factory). O-P5P38-META-001 remediation pattern effective — no metadata slip risk this pass.

## Observations (non-findings)

- O-P5P39-A-001 (informational, non-blocking): **Persistence re-confirmation of O-P5P37-A-001 / O-P5P38-A-001** — the combined-footnote coupling at Ruling-12 §1 L1120 (single footnote covering both F-P5P36-A-001 phantom-code redirect and F-P5P36-A-002 sibling-authorship-premise) remains structurally coupled. Fresh-context re-observation confirms the pattern (non-defective — combined footnote is deliberately co-located per Burst 87 remediation) and does not surface new novelty. No action needed.

## Novelty Assessment

**LOW** — Third consecutive Adv-A clean pass under fresh context. No new axes of concern surface. All Burst-87 remediation vectors (F-P5P36-A-001 phantom-code redirect + F-P5P36-A-002 four-site sibling-sweep) persist. All persistent Adv-A baselines (CLI exit-code, jsonEnvelope, confirm-gate 5-path, Registered Verbs, taxonomy-catalog byte-parallel, handler emissions, S-7.01 sibling-sweep axis) hold. Assuming Adv-B concurrent lane also NO_FINDINGS, this is the closing pass of the three-consecutive-clean-pass window and BC-5.39.001 CONVERGES.

## Scope-Conformance Attestation

Reviewed strictly within the public-surface + operator-UX perimeter per BC-5.39.002. Adv-B current-pass sidecar was NOT read (concurrent-lane isolation). Governance artifacts (VP-INDEX, ARCH-11, policies.yaml, sprint-state.yaml, STATE.md drift table, BC/VP frontmatter versions) inspected only for operator-observable surface impact.
