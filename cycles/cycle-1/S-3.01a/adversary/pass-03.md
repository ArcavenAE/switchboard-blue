---
artifact_id: adv-S-3.01a-pass-03
review_target: S-3.01a-tmux-control-mode
producer: adversary
pass: 3
fresh_context: true
branch: feature/S-3.01a-tmux-control-mode
base: develop @ d54bf1a
tip: 44a207b
findings_count: 2
findings_by_severity: {critical: 0, high: 0, medium: 1, low: 1, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-26
note: Returned inline by adversary because tool profile is read-only; persisted by orchestrator via state-manager.
---

# Adversarial Review — Pass 3 — S-3.01a

## Critical Findings
None.

## High Findings
None.

## Medium Findings

### F-PASS3-M-001 — Mis-anchored citation in ListSessions docstring

**File:** `internal/session/session.go:105`

```go
// The returned slice is a value copy — mutations do not affect the Publisher's
// internal state (ARCH-08 §6.6 rule 12: no internal pointer leak).
```

ARCH-08 §6.6 is the "Planned positions (Wave 3 prospective)" table — has no "rule 12". The cited rule comes from `.claude/rules/go.md` "Idiomatic Go" section, rule 12 ("Never return internal pointers from a locked accessor"). The citation points an implementer at the wrong document.

Confidence: HIGH. Severity: MEDIUM.

**Fix:** Change to "go.md rule 12" or "CLAUDE.md go rule 12".

## Low Findings

### F-PASS3-L-001 — ErrSessionNotFound annotates wrong source BC (pending intent verification)

**File:** `internal/session/session.go:23-25, 122-123`

```go
// ErrSessionNotFound ... (E-SES-001; BC-2.04.001).
var ErrSessionNotFound = errors.New("session not found")
```

Error-taxonomy.md:127 attributes E-SES-001 to BC-2.04.003 (attach-time session lookup), not BC-2.04.001 (publication lifecycle). BC-2.04.001 does not cite E-SES-001 anywhere.

Error reuse for Publisher.Unpublish/Get is reasonable, but the annotation should cite the source BC.

**Fix:** Change citation to `(E-SES-001; BC-2.04.003)` or `(E-SES-001; BC-2.04.001, BC-2.04.003)`.

## Observations

- unescapeTmuxOutput arithmetic `(next-'0')*64` operates in byte (uint8) and wraps for first-digit > 3. Tmux never emits such; defensive comment + test vector covering "first digit ≤ 3" would harden the contract. Not blocking.
- dispatchLoop does not clear c.proc/c.stdin/c.cancel after stream EOF, so Connect → wait-for-EOF → Connect returns ErrAlreadyConnected without Close. Docstring documents this; behavior is correct, observation only.
- ARCH-08 §6.6 import compliance: tmux={halfchannel,session}; session={admission} (subset of allowed {frame,admission}). No frame import needed; session.go header explains.
- ARCH-09 classification: session=boundary; tmux=effectful. Headers match.
- AC-001/002/003/004 all have non-skipped tests with real assertions against fake stream.
- F-01/H-01/H-02/H-03/H-04/AC-004 octal-unescape paths exercised.
- No tests shell out to real tmux.
- UTC discipline maintained at session.go:79.
- No init(), no globals, no log.Fatal/os.Exit, no panics.
- HalfChannel concurrent-use contract satisfied (dispatchLoop sole writer; test reads ds.Seq() after Err() confirms loop exit).

## Novelty Assessment

Novelty: LOW–MEDIUM. Findings are documentation/annotation defects (mis-anchored citations), not logic gaps. Logic, lock discipline, error sentinels, import compliance, ARCH-09 classification, AC coverage, and concurrency design all pass.

## Resolution decisions (mechanical, no human input required)

- F-PASS3-M-001: change "ARCH-08 §6.6 rule 12" → "go.md rule 12" in session.go:105.
- F-PASS3-L-001: change BC-2.04.001 → BC-2.04.003 (or both) in session.go:23-25, 122-123.
