---
artifact_id: VP-LIFECYCLE
document_type: governance-policy
level: L4
version: "1.0"
status: active
producer: spec-steward
timestamp: 2026-06-25T00:00:00
---

# VP Lifecycle State-Machine

> First codification of the VP lifecycle state-machine for this project.
> VSDD.md (engine-level) defines VP immutability and append-only numbering
> but does not prescribe a named state-machine for per-VP lifecycle status.
> This document fills that gap for the switchboard-blue project.

## States

| State | Meaning |
|-------|---------|
| `draft` | VP exists but no story yet implements or is planned to implement it. May be authored speculatively during architecture or spec-crystallization phases. |
| `active` | VP is in scope for the current or near-future wave. A story exists or is explicitly planned. |
| `implemented` | VP's backing story has merged with passing tests and evidence that exercise the property. The proof harness (or test function) exists and passes in CI. |
| `deferred` | VP scope is acknowledged but pushed to a later wave or phase. Deferral requires a target wave/phase and a stated reason. |
| `superseded` | VP was replaced by a newer VP. The successor VP ID must be cited. No further evidence is required for the superseded VP itself. |

## Transitions

```
draft ──────────────────────────────────────────► active
                  (scoped into a wave)

active ─────────────────────────────────────────► implemented
                  (backing story merges; tests pass)

active ─────────────────────────────────────────► deferred
                  (human/orchestrator decision to push out)

deferred ───────────────────────────────────────► active
                  (re-pulled into a wave)

any state ──────────────────────────────────────► superseded
                  (replaced by a newer VP-NNN)
```

There is no `removed` terminal state for VP documents. Superseded VPs are
retained in-place with updated frontmatter (append-only IDs). See VSDD.md
§ "Append-Only ID and Slug Protection".

## Required Fields per State

### `draft`

```yaml
lifecycle_status: draft
```

No additional required fields. `lifecycle_history` is optional at draft.

### `active`

```yaml
lifecycle_status: active
```

`lifecycle_history` entry recommended when transitioning from `draft`:

```yaml
lifecycle_history:
  - date: YYYY-MM-DD
    from: draft
    to: active
    reason: "Scoped into wave X story S-N.NN"
    agent: <agent-id or human>
```

### `implemented`

```yaml
lifecycle_status: implemented
implementing_stories:
  - S-N.NN  # story ID; cite PR number in the lifecycle_history entry
```

Required `lifecycle_history` entry:

```yaml
lifecycle_history:
  - date: YYYY-MM-DD
    from: active
    to: implemented
    reason: "Backing story S-N.NN merged (PR #N); proof harness passes in CI"
    agent: <agent-id or human>
```

### `deferred`

```yaml
lifecycle_status: deferred
deferred_to: <wave-id or phase-name>   # e.g. "phase-6-hardening", "wave-3"
deferred_reason: "<human-readable explanation>"
deferred_date: YYYY-MM-DD
```

Required `lifecycle_history` entry:

```yaml
lifecycle_history:
  - date: YYYY-MM-DD
    from: active
    to: deferred
    reason: "<same as deferred_reason>"
    agent: <agent-id or human>
```

### `superseded`

```yaml
lifecycle_status: superseded
superseded_by: VP-NNN
```

Required `lifecycle_history` entry:

```yaml
lifecycle_history:
  - date: YYYY-MM-DD
    from: <prior-state>
    to: superseded
    reason: "Replaced by VP-NNN: <brief rationale>"
    agent: <agent-id or human>
```

## Version Bumps on Lifecycle Transitions

Every lifecycle transition requires a version bump on the VP document:

| Change type | Bump |
|-------------|------|
| `draft` → `active` | PATCH (0.0.X) — metadata only |
| `active` → `implemented` | MINOR (0.X.0) — evidence added |
| `active` → `deferred` | MINOR (0.X.0) — scope change |
| `deferred` → `active` | PATCH (0.0.X) — re-activation |
| any → `superseded` | MINOR (0.X.0) — lifecycle closure |

## Immutability Constraint

Once `verification_lock: true` is set on a VP (proof harness committed and
proof completed), the property statement and proof method sections are
immutable. Lifecycle metadata (`lifecycle_status`, `lifecycle_history`,
`implementing_stories`) may still be updated. If a locked VP's source BC
undergoes a MAJOR version bump, the VP must be flagged for re-assessment by
the spec-steward and may only be replaced via a new VP-NNN (append-only).

## Relationship to VP-036 Precedent

VP-036 was the first VP manually transitioned to `deferred` (v1.1, 2026-06-25,
S-1.03 spec patch). This state-machine codifies that precedent and applies it
consistently to all subsequent VPs.
