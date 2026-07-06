---
artifact_id: S-BL.POLICY-SCHEMA-VALIDATOR
document_type: story
level: ops
story_id: S-BL.POLICY-SCHEMA-VALIDATOR
title: "policies.yaml schema linter — validate canonical POL-NNN field schema"
status: backlog
producer: story-writer
timestamp: 2026-07-06T00:00:00Z
version: "0.1-backlog-stub"
phase: 2
epic: E-6
wave: unscheduled
priority: P3
scope_phase: E
estimated_points: 2
bc_traces: []
vp_traces: []
subsystems: [network-management]
architecture_modules:
  - tools/policy-lint   # candidate location; or cmd/sbctl subcommand; TBD at scheduling
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: []
blocks: []
inputDocuments:
  - '.factory/policies.yaml'
acceptance_criteria_count: 0
backlog_origin:
  source: Ruling-12
  ruling: Ruling-12 §6 (F-P7L3R2-03 POL-002 schema drift)
  drift_items_consumed:
    - F-P7L3R2-03   # POL-002 schema drift: non-standard name/description fields instead of canonical schema
  notes: >
    Ruling-12 §6 minted this story in response to F-P7L3R2-03 (Pass-7 Lens-3 Round-2):
    POL-002 had non-standard `name:` and `description:` fields rather than the canonical
    POL-001 schema (id, title, severity, scope, rule, rationale, enforcement, examples).
    The ruling added a schema-linter story to catch future drift.

    The linter validates that every entry in .factory/policies.yaml conforms to the
    canonical field schema documented in POL-001. No BC or VP traces — this is a
    governance tooling story, not a product behavioral story.

    Candidate implementations:
    (a) Standalone Go script in tools/policy-lint (simplest; no daemon dependency)
    (b) `sbctl policy lint` subcommand (integrates with existing CLI surface)
    (c) CI check called by the lefthook pre-commit hook

    All three options are valid; choice is deferred to scheduling/architect.
---

# S-BL.POLICY-SCHEMA-VALIDATOR: Policies.yaml Schema Linter

> **STATUS: BACKLOG STUB.** This story was minted per Ruling-12 §6. Acceptance criteria,
> implementation approach (tool vs. CLI subcommand vs. CI hook), and task list will be
> fleshed out when the story is scheduled.

## Narrative

- **As a** spec-steward running governance sweeps
- **I want** an automated linter that validates every policy entry in `.factory/policies.yaml`
  against the canonical POL-001 field schema
- **So that** schema drift (missing fields, non-standard keys) is caught mechanically
  before human review

## Context

`.factory/policies.yaml` is the governance policy registry (currently v1.3, 4 policies:
POL-001, POL-002, POL-004, plus POL-003 candidate). The canonical schema (from POL-001) is:

```
id: <string>
title: <string>
severity: HIGH | MED | LOW
scope: <string>
rule: <string>
rationale: <string>
enforcement: <string>
examples:
  violation: <string>
  compliant: <string>
```

F-P7L3R2-03 found that POL-002 used non-standard `name:` and `description:` fields.
Ruling-12 §4 restructured POL-002 to the canonical schema; §6 minted this story to
prevent future drift.

## Anchors Consumed

| Anchor | Verbatim ID | Source |
|--------|-------------|--------|
| Ruling-12 §6 — policy-schema-validator story | F-P7L3R2-03 | Pass-7 Lens-3 Round-2 finding |

## Sketched Acceptance Criteria

> ACs are illustrative. No BC/VP traces — this is tooling, not behavioral product.

**AC-001:** The linter reads `.factory/policies.yaml` and validates that every entry under
`policies:` contains at minimum the required fields: `id`, `title`, `severity`, `scope`,
`rule`, `rationale`, `enforcement`. Missing required fields produce a linting error naming
the policy `id` and the missing field.

**AC-002:** The linter validates `severity` is one of `HIGH`, `MED`, or `LOW`. An
unrecognized value produces an error.

**AC-003:** Unknown top-level keys in a policy entry (e.g., `name:`, `description:`) are
flagged as warnings (not errors) to allow forward-compatible extension while alerting
about drift.

**AC-004:** Exit code 0 on clean; exit code 1 if any errors (not just warnings). Suitable
for use as a CI gate or pre-commit hook.

**AC-005:** `make lint` or the project's lefthook pre-commit configuration includes a
call to the linter against `.factory/policies.yaml`.

## Non-Goals

- Does not validate policy *content* (whether the rule is well-reasoned). Structural
  schema compliance only.
- Does not validate `.factory/policies.yaml` YAML syntax (yaml.Unmarshal handles that).
- Does not auto-fix schema drift — report only.

## When to Schedule

Unscheduled. Small story (2 points), no dependencies. Can be pulled into any wave with
spare capacity during spec-steward governance work.

## Backlog Status

| Field | Value |
|-------|-------|
| Created | 2026-07-06 |
| Origin | Ruling-12 §6; F-P7L3R2-03 POL-002 schema drift |
| BC/VP traces | none (governance tooling) |
| Status transitions | (none yet) |
