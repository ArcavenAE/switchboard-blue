---
document_type: adversarial-review
artifact_id: P5-pass-5-Adv-A
verdict: HAS_FINDINGS
finding_critical: 0
finding_high: 0
finding_medium: 2
finding_low: 2
observation_count: 1
develop_tip: cbd02728377e0c158f7c8ff489ee076c98173e5b
model: claude-opus-4-7
time_spent_minutes: 6
files_read: 4
read_cap: 6
prior_passes_read: false
timestamp: 2026-07-03T00:00:00Z
lens: A
lens_focus: public-surface-operator-ux
---

# Adversarial Review — P5 Pass 5, Lens A (Public Surface / Operator UX)

## Preflight

- Repo root: `/Users/skippy/work/aae-orc/run/switchboard-blue`
- Branch: `develop`
- Expected tip: `cbd02728377e0c158f7c8ff489ee076c98173e5b`
- Observed tip: `cbd02728377e0c158f7c8ff489ee076c98173e5b`
- Match: YES — review proceeded.

## Scope

Lens A focuses on the operator-facing surface where a first-run user meets the daemon:

1. `sbctl` CLI dispatch tree (`cmd/sbctl/`) — do advertised subcommands reach working daemon handlers, canonical error emission, real values in interactive prompts.
2. Daemon admin RPC surface (`cmd/switchboard/admin_handlers.go`) — wire-field contract consistency, taxonomy-canonical emission text.
3. First-run / smoke-path coherence for the operator.
4. Spec ↔ surface parity between `interface-definitions.md` and the compiled binaries.

Adjudicated-deferral list respected (not re-reported): svtn.list wire handler, version/ping wire handlers, sessions.list wire handler, `internal/adminwire` extraction, router daemon runtime stub, and `DRIFT-P5P4-PROMPT-SHORTID` (static-example confirm prompt is authorized).

## Findings

### F-P5P5-A-001 [MEDIUM]: interface-definitions §116 misstates admin.svtn.create authority as "control-role" while impl, Registered Verbs, and BC-2.07.001 require bootstrap-only

- **What.** The CLI-vocabulary table in `interface-definitions.md` §116 describes the authority of `sbctl admin svtn create` as "Requires control-role key on control-mode daemon." Three authoritative locations disagree and require the caller to be **the daemon bootstrap key**, not any control-role key:
  - Implementation: `cmd/switchboard/admin_handlers.go:686-705` — `if !hasPubkey || !m.IsBootstrapKey(callerPub) { emit E-ADM-009 "insufficient authority: bootstrap key required for admin.svtn.create" }`.
  - Registered Verbs §380: "Bootstrap-only: authenticated caller MUST be the daemon bootstrap key with RoleControl; cross-SVTN control-role keys are not authorized."
  - BC-2.07.001 Inv-3 (line 162) and Ruling-5.
- **Where.** `.factory/specs/prd-supplements/interface-definitions.md` §116, authority column of the `sbctl admin svtn create` row.
- **Why it matters.** This is an operator-facing surface. A reader with a non-bootstrap control-role key will follow §116, attempt the call, and hit E-ADM-009 with no forward-pointer that the *table* was wrong. It also weakens the audit story: the CLI table is the most-cited "what am I allowed to do" reference, and it contradicts the invariant it exists to project.
- **Suggested remediation.** Reword the §116 authority cell to "Bootstrap-only: caller MUST authenticate with the daemon bootstrap key (RoleControl); cross-SVTN control-role keys are not accepted. See §380 and BC-2.07.001 Inv-3." Add a v1.18 changelog line calling out the §116 correction so downstream tooling maintainers notice.

### F-P5P5-A-002 [MEDIUM]: `sbctl admin recover` fully specified in §119-125 but neither CLI nor daemon dispatch it — no PENDING/DRIFT annotation covers the gap

- **What.** `.factory/specs/prd-supplements/interface-definitions.md` §119-123 advertises an emergency-recovery subcommand with a concrete flag surface (`--svtn <id> --bootstrap-key <path> --confirm <svtn-short-id> | --yes`), exit-code column, and confirm-gate coupling; §125 names "recover" among the destructive operations that share the five-path confirm gate. Neither side implements it:
  - `cmd/sbctl/admin.go:157-178` — `runAdmin`'s switch covers only `key | list-keys | svtn`; default arm returns `admin: unknown subcommand %q; expected 'key', 'list-keys', or 'svtn'`. No `recover` case exists anywhere in `cmd/sbctl/`.
  - `cmd/switchboard/admin_handlers.go:127-134` — `BuildAdminHandlers` registers exactly six verbs (admin.key.register/revoke/expire/list-keys, admin.svtn.create/destroy). No `admin.recover` handler is registered on any code path.
- Unlike the five known-deferred wire items, `admin recover` carries **no** PENDING-S-BL.* / DRIFT / adjudication annotation in the spec — it reads as first-class advertised surface.
- **Where.** Spec: `.factory/specs/prd-supplements/interface-definitions.md` §119-125. Impl gap: `cmd/sbctl/admin.go:157-178`, `cmd/switchboard/admin_handlers.go:127-134`.
- **Why it matters.** An operator hitting a real incident will reach for `sbctl admin recover` as the spec instructs and be met with `admin: unknown subcommand "recover"` (exit 2) with no explanation that the feature is not yet delivered. Recovery flows are the exact case where surface-vs-spec drift most damages trust.
- **Suggested remediation.** One of:
  1. Add `PENDING-S-BL.ADMIN-RECOVER-WIRE` annotation to §119-125, file a backlog story stub, and add a runtime hint to the `runAdmin` default arm ("recover: pending — see PENDING-S-BL.ADMIN-RECOVER-WIRE") so operators are not silently redirected to `unknown subcommand`.
  2. Or, if recover is not planned for the current release train, remove the row and the confirm-gate mention until a delivery story lands.

### F-P5P5-A-003 [LOW]: §116/§117 exit-code column omits canonical error codes actually emitted by admin.svtn.create/destroy

- **What.** The exit-code column of §116 (create) lists top-level result codes but does not enumerate `E-CFG-001` (five validation arms in `validateSVTNName`, `cmd/switchboard/admin_handlers.go:824-849`: empty / whitespace-only / >255 bytes / invalid UTF-8 / control chars including U+2028/U+2029) nor `E-INT-001` (non-duplicate Create failure wrap at `admin_handlers.go:737`: `E-INT-001: internal error: admin.svtn.create: %w`). §117 (destroy) similarly omits `E-CFG-001` for the caller-supplied-name validation path.
- **Where.** `.factory/specs/prd-supplements/interface-definitions.md` §116 exit-codes column; §117 same column.
- **Why it matters.** Operators writing scripts around these calls will not know to branch on E-CFG-001 or E-INT-001; §116 is presented as the authoritative catalog of exit codes. Low severity because the codes themselves are canonical and stable — only the projection is incomplete.
- **Suggested remediation.** Extend §116 and §117 exit-codes cells to enumerate every taxonomy code the compiled handler can emit, cross-referenced to the taxonomy row (E-CFG-001 for validation failures, E-INT-001 for internal-error wrap on Create).

### F-P5P5-A-004 [LOW]: §59 promises `sbctl svtn create` as retained deprecation alias but main.go dispatches it as an unknown subcommand

- **What.** `interface-definitions.md` §59 states: *"`sbctl svtn create [--name=<name>]` — [DEPRECATED] alias for `sbctl admin svtn create`. Retained as alias until vMINOR+1 deprecation cycle completes."* The top-level dispatcher in `cmd/sbctl/main.go:48-80` has cases only for `sessions | paths | router | console | admin`; no `svtn` arm exists. Invoking `sbctl svtn create` today produces `unknown subcommand: svtn` (exit 2), which is behavioural removal, not retention.
- **Where.** Spec: `.factory/specs/prd-supplements/interface-definitions.md` §59. Impl gap: `cmd/sbctl/main.go:48-80`.
- **Why it matters.** Deprecation contract says the alias still works and merely warns; the binary treats it as if the deprecation cycle has already completed. Anyone with muscle memory (or shell history) from a prior release hits an unhelpful exit-2 rather than the promised warn-and-forward. Low severity because the migration target is documented and stable.
- **Suggested remediation.** One of:
  1. Wire a `svtn` case in the top-level dispatcher that emits a stderr deprecation notice and re-dispatches to `runAdmin` with the `svtn` subargv, or
  2. Update §59 to state the alias is removed at this version and record the removal in the changelog.

## Observations

### OBS-P5P5-A-001: CLI vocabulary block (§58-94) advertises a broader command surface than either binary implements

- Beyond the two findings above, the §58-94 block enumerates `sbctl svtn destroy | list | status`, `sbctl svtn keys list`, `sbctl sessions attach | detach | status`, `sbctl paths ping`, `sbctl router reload | drain`, `sbctl console detach | switch` — none of these have CLI dispatch branches in `cmd/sbctl/main.go` or subcommand files. Downgraded from finding to observation because the underlying **wire** deferrals (sessions.list wire, version/ping wire, svtn.list wire, router daemon runtime stub) are on the "do not re-report" list and are the upstream causes of most of the missing CLI arms. Once those wire handlers land, a coordinated CLI-surface pass will need to wire each subcommand; a per-command PENDING-S-BL.<VERB>-CLI annotation on §58-94 rows now would make that pass mechanical rather than archaeological. Flagged for the tracker rather than remediated here.

## Policy Rubric

- **POL-001 (changelog-completeness, MED).** F-P5P5-A-001 and F-P5P5-A-002 both warrant a v1.18 changelog line (authority correction, recover-deferral annotation). No blocker at this pass — findings capture the requirement.
- **POL-002 (story-index-row-sync, MED).** No story-index rows detected as needing sync from this pass (findings are spec-side, not story-side). Clean under POL-002.

VERDICT: HAS_FINDINGS
