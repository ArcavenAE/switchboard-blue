## Summary

Comment-only documentation hygiene fixes for Wave 4, closing three adversarial
findings surfaced during the wave-level convergence pass (C=0 / H=0 / M=0 final
verdict). No behavioral change; zero code, logic, or test-assertion modifications.

**Findings addressed:**

| ID | File | Finding |
|----|------|---------|
| L-1 | `cmd/switchboard/access.go` | Stale "FORBIDDEN imports" header falsely claimed `internal/config` was not-yet-existing, when S-6.01 had already wired it |
| S403-COS1 | `internal/arq/arq.go` | Doc comment claimed `SACKPopCount` uses `encoding/binary`; the file does not import that package — it uses `math/bits.OnesCount64` directly |
| S403-COS2 | `internal/arq/arq_test.go` | Dangling "per stub notes" reference in a traceability comment (no stubs remain; the helper is green-by-design) |

## Architecture Changes

No architectural changes. Comment corrections only.

```mermaid
graph TD
    A[docs/wave4-comment-hygiene] -->|comment-only patch| B[develop]
    B --> C[No structural change]
```

## Story Dependencies

This branch is a standalone cycle-close hygiene patch, not tied to a numbered
story. It is a follow-on to the Wave 4 adversarial convergence pass.

```mermaid
graph LR
    W4ADV[Wave 4 Adversarial Convergence<br/>C=0/H=0/M=0] -->|generated findings| HYGI[docs/wave4-comment-hygiene]
    S601[S-6.01 config wiring] -->|made internal/config PERMITTED| HYGI
    S403[S-4.03 ARQ impl] -->|SACKPopCount is the subject| HYGI
```

## Spec Traceability

```mermaid
flowchart LR
    L1[Finding L-1<br/>stale FORBIDDEN header] --> FIX1[access.go: reword<br/>PERMITTED/deferred boundary]
    COS1[Finding S403-COS1<br/>false encoding/binary claim] --> FIX2[arq.go: remove<br/>encoding/binary mention]
    COS2[Finding S403-COS2<br/>dangling stub notes ref] --> FIX3[arq_test.go: clean<br/>traceability comment]
```

## Test Evidence

No tests changed. Build and lint confirmed clean prior to push.

- `just build` — pass
- `just lint` — 0 issues
- `just test` (arq + cmd/switchboard) — green

Wave 4 adversarial convergence result (pre-branch): 6/6 diverse-lens passes, C=0 / H=0 / M=0.

## Holdout Evaluation

N/A — evaluated at wave gate.

## Adversarial Review

N/A — evaluated at Phase 5. Wave-level adversarial convergence produced these
findings; this PR closes them.

## Security Review

N/A — comment-only changes. No code paths, data flows, inputs, outputs, or
access-control logic modified.

## Risk Assessment

- **Blast radius:** Documentation only. No runtime behavior change possible from
  `//` comment edits.
- **Performance impact:** None.
- **Rollback:** Not meaningful; comment corrections carry no runtime risk.

## AI Pipeline Metadata

- Pipeline mode: feature hygiene (manual cycle-close)
- Models: us.anthropic.claude-sonnet-4-6
- Story: wave4-comment-hygiene (no story ID — standalone hygiene patch)

## Pre-Merge Checklist

- [x] Comment-only diff verified by orchestrator (zero code/logic/assertion changes)
- [x] Findings L-1, S403-COS1, S403-COS2 addressed
- [x] Build clean
- [x] Lint 0 issues
- [x] Tests green (arq + cmd/switchboard)
- [x] Wave 4 adversarial convergence: C=0/H=0/M=0
- [x] No AI attribution in PR body
- [x] Base branch: develop (gitflow)
