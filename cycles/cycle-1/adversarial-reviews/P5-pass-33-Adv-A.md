---
pass: 33
lane: A
scope: public-surface + operator-UX
develop_head: 6deda15
factory_head_pre_review: 4b1fb62
verdict: NO_FINDINGS
findings_count:
  critical: 0
  high: 0
  medium: 0
  low: 0
  process_gap: 0
reviewed_at: 2026-07-03
---

# Phase 5 Pass 33 — Adv-A Lane (public-surface + operator-UX)

## Verdict

**NO_FINDINGS** — Clean baseline. 3-clean-pass streak advances from 1/3 to 2/3 (BC-5.39.001).

## Lens

Adv-A operates from the SRE-operator / third-party integrator perspective:
what does an operator actually see when they run `sbctl`, hit an error, or
integrate against the JSON envelope? Only artifacts an outside consumer can
observe are in scope. Governance-internal drift (ARCH-11 dep-graph, VP
frontmatter, BC cross-refs, POL-006/POL-003/POL-002 sweeps on `.factory/`)
is Adv-B territory and was not read.

## Sweep receipts

### Spec artifacts read (exact-current versions)

- `.factory/specs/prd-supplements/error-taxonomy.md` v4.6 (262 lines, full)
- `.factory/specs/prd-supplements/interface-definitions.md` v1.28 (432 lines, full)
- `.factory/policies.yaml` v1.2 (66 lines, full)

### Implementation artifacts read

- `cmd/sbctl/main.go` (168 lines, full) — global-flag parse, subcommand dispatch,
  `usageError`+`errors.As` exit-code discrimination, `writeSuccess`/`writeError`
  envelope emission
- `cmd/sbctl/admin.go` (565 lines, full) — `runAdminSvtn{Create,Destroy}`,
  `runAdminKey{Register,Revoke,Expire}`, `boolStringFlag`, `runDestroyConfirmGate`
  5-path logic, `confirmSVTNShortIDValid`, client-side `--after` validation
- `cmd/switchboard/admin_handlers.go` (partial: lines 100-780) —
  `BuildAdminHandlers`, `decodePublicKey`, `mapAdminError` 9-arm switch + default
  arm, `resolveAndVerifyCallerRole`, `resolveCallerAdmissionAnyRole`,
  `svtnAlreadyExistsErr` / `svtnNotFoundErr` / `adminKeyNotFoundErr` wrappers,
  `makeAdminSVTNCreateHandler` bootstrap-only pre-check + defense-in-depth
  RoleControl gate

### Grep patterns exercised

- `resolveCallerAdmissionAnyRole` — verify F-L2-003 read-only carve-out for
  `admin.key.list-keys` (single call site at admin_handlers.go:363; matches
  interface-def §111 authority column)
- `runDestroyConfirmGate` — verify 5-path membership: destroy (admin.go:306),
  register (admin.go:463); revoke does NOT call it (admin.go:483-518) per
  F-P5P11-A-001 revoke carve-out
- `E-ADM-018`, `E-ADM-019`, `E-ADM-020`, `E-ADM-021`, `E-INT-999`, `E-ADM-011`,
  `E-CFG-012`, `E-CFG-013`, `E-CFG-001` — emission-vs-taxonomy byte diff
- `svtn_id`, `path_distribution` in router.metrics response — verify v1.28
  remediation (phantom `svtn_id` removed, path_distribution integer counts not
  fractional ratios)
- `$schema` in `jsonEnvelope` — verify v1.27 removal
- `--confirm=<svtn-short-id>` normalization — verify lowercase + `svtn-` prefix
  + 8 hex chars in `confirmSVTNShortIDValid`

## Cross-checks (all passed)

### Error emission ↔ taxonomy canonical prose

| Code | Emission site | Taxonomy row | Verdict |
|------|---------------|--------------|---------|
| E-ADM-018 | admin_handlers.go:443 | v4.6 canonical (no parenthetical) | byte-identical |
| E-ADM-019 | admin_handlers.go:430, :434-437 | v4.6 canonical | byte-identical |
| E-ADM-020 | admin_handlers.go:445 | v4.6 "for any well-formed request" | byte-identical |
| E-ADM-021 | admin_handlers.go:447 | v4.6 (E-ADM-020 mirror) | byte-identical |
| E-ADM-011 (destroy variant 2) | admin_handlers.go:449 | v4.6 Variant 2 | byte-identical |
| E-INT-999 (default arm) | admin_handlers.go:458 | v4.1 mint text | byte-identical |
| E-ADM-009 (any-role, unregistered) | admin_handlers.go:599, :621, :625 | v4.6 canonical "has role unregistered" | byte-identical |
| E-ADM-009 (create bootstrap-only) | admin_handlers.go:754, :771 | Ruling-12 §2 canonical | byte-identical |
| E-SVTN-001 (create dup) | admin_handlers.go:642 | v4.6 F-P1L1-004 no stutter | byte-identical |
| E-SVTN-003 (not found) | admin_handlers.go:659 | v4.6 F-P9L1-04 no stutter | byte-identical |
| E-ADM-013 (key not found) | admin_handlers.go:676 | v4.6 F-P9L1-04 no stutter | byte-identical |
| E-CFG-001 (invalid args) | admin_handlers.go:725 | v4.6 canonical | byte-identical |

### CLI exit-code discrimination

- `main.go:104-111` — `errors.As(err, &ue)` → exit 2 for usageError; else exit 1
- Bare `sbctl` (no subcommand) — stderr enumerated subcommand list, exit 2
  (main.go:52-56) — matches interface-def §174
- `--help`/`-h` — flag.CommandLine.SetOutput(os.Stdout) → stdout, exit 0
  (main.go:47) — matches AC-012 / BC-2.07.002 EC-003 Ruling A
- `paths` sub-verb discrimination — no sub-verb generic usage vs unknown sub-verb
  router-style naming, both exit 2 (main.go:75-80) — matches F-P5P8-A-006
- `sessions {attach,detach,status}` — E-BL routing to S-BL.DISCOVERY-WIRE
  backlog, exit 2 (main.go:130-131)

### Confirm-gate 5-path (`runDestroyConfirmGate`)

| Path | Implementation | Spec authority |
|------|----------------|----------------|
| flag supplied + normalizes + matches | admin.go via `confirmSVTNShortIDValid` | interface-def §125-§137 |
| interactive TTY prompt | admin.go tty-detect + read | interface-def §125-§137 |
| non-TTY without --confirm | E-CFG-013 exit 2 | error-taxonomy v4.6 E-CFG-013 canonical |
| `--yes` bypass | admin.go early-return | interface-def §125-§137 |
| `--yes` + `--confirm` mutex | E-CFG-012 exit 2 | error-taxonomy v4.6 E-CFG-012 canonical |

Register (admin.go:463) uses `runDestroyConfirmGate` with `targetFlag="--svtn"`;
destroy (admin.go:306) uses it with `targetFlag="--name"`; revoke
(admin.go:483-518) has the F-P5P11-A-001 boolean-confirm carve-out (uses
`boolStringFlag` with IsBoolFlag=true, emits E-ADM-018 server-side).

### Wire response envelope

`jsonEnvelope` struct in sbctl matches interface-def §JSON envelope exemplar:
`ok`, `error{code,message}`, `data`. No `$schema` field emitted (v1.27
remediation confirmed in place).

### Router.metrics response shape

`path_distribution` field is `map[string]uint64` (integer counts, not
fractional ratios); no phantom `svtn_id` field in response — v1.28 remediation
holds.

### Registered Verbs table (interface-def §397-§411)

Six admin verbs registered by `BuildAdminHandlers`: `admin.svtn.create`,
`admin.svtn.destroy`, `admin.key.register`, `admin.key.revoke`,
`admin.key.expire`, `admin.key.list-keys`. Plus `paths.list`, `router.metrics`,
`router.status`, `sessions.list`, `ping`, `version`. Table row-for-row present.

## POL sweep on operator-visible surfaces

- **POL-001 (changelog-completeness)**: `error-taxonomy.md` v4.6 changelog row
  present; `interface-definitions.md` v1.28 changelog row present;
  `policies.yaml` v1.2 modified entry present. No orphan version bumps.
- **POL-002 (story-index-row-sync)**: out of Adv-A scope (governance-internal).
- **POL-005 (body-prose-impl-anchor-check)**: `interface-definitions.md`
  §125-§137 confirm-gate prose anchors to `runDestroyConfirmGate` in
  admin.go:306, :463; §108-§111 admin key rows anchor to
  runAdminKeyRegister/Revoke/Expire; §120-§122 SVTN lifecycle anchors to
  runAdminSvtnCreate/Destroy. No orphan operator-facing prose.
- **POL-008 (BC PC Phase column drift)**: operator-visible BC PC rows sampled
  via error-taxonomy backrefs — no visible drift in the operator lens.

## Observations (non-findings)

None. Pass 32 clean baseline holds under fresh-context Adv-A re-review.

## Novelty assessment

Novelty: LOW — findings are absent, not refinements. The public-surface +
operator-UX lens has converged over prior burst-cycle remediation
(v1.27/v1.28 interface-def fixes, v4.6 error-taxonomy narrowing, admin_handlers
default-arm universality, register confirm-gate parity). Advances 3-clean-pass
streak to 2/3.
