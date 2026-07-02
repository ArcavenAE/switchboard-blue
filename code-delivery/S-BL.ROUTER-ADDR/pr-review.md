# PR #56 Review — S-BL.ROUTER-ADDR

**Verdict: APPROVE**

Independent fresh-eyes review of the diff, PR description, and test evidence for
`feat(S-BL.ROUTER-ADDR): populate PathSnapshot.RouterAddr with resolved host:port (BC-2.06.003 PC-1)`.

## Summary

The change is minimal, additive, and correct against the stated contract:

- `PathTracker` gains an immutable `routerAddr` field (constructor-write-only,
  read under the existing `mu`).
- `NewPathTrackerWithAddr(addr, initialRTTMs, alpha)` delegates to
  `NewPathTracker` for the alpha guard + shared init, then assigns
  `t.routerAddr = addr`. `NewPathTracker` itself is untouched — backward
  compatibility for all existing (addr-less) call sites is preserved.
- `PathSnapshot.RouterAddr` is populated verbatim from `t.routerAddr` under the
  existing `Snapshot()` lock; no new lock, no new allocation beyond a string
  copy.
- `internal/metrics.PathsList` replaces the hard-coded `""` and the
  `DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER` comment with `snap.RouterAddr`.
- BC version bumped `v1.14 → v1.15` in every doc-comment and the
  `.factory/specs/behavioral-contracts/ss-06/BC-2.06.003.md` changelog. The
  active DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER sentinel-permission annotation is
  retracted; the remaining occurrences of the string are historical (v1.9
  changelog entry) and closure-status references (line 60, "closed by
  S-BL.ROUTER-ADDR"), which are appropriate.
- `TestVP047_SbctlPathsList_EndToEnd` oracle comment updated to reflect the new
  semantics (`""` still expected here because the integration paths still use
  `NewPathTracker`; non-empty coverage is provided by the new
  `TestVP047_RouterAddrNonEmpty`).

## Checklist Results

| # | Item | Status |
|---|------|--------|
| 1 | Diff Coherence | PASS — all 5 files scope-appropriate |
| 2 | Description Accuracy | PASS — PR body matches diff |
| 3 | Test Coverage | PASS — every new code path exercised (constructor happy/invalid-alpha, immutability, concurrent, snapshot propagation, addr-less compat, handler pass-through, JSON key presence, VP-047 A+B, field-swap regex) |
| 4 | Demo Evidence | Not visible in diff; PR body references `.factory/demo-evidence/S-BL.ROUTER-ADDR/` with 17 files on orphan branch — accepted on faith of description |
| 5 | Commit Quality | PASS — conventional-commit format, story ID present, no AI attribution |
| 6 | Diff Size | PASS — 598 additions / 19 deletions; large portion is test scaffolding |
| 7 | Missing Changes | PASS — all 5 ACs traced to test symbols present in diff |
| 8 | Dependency Status | PASS — S-W5.04 (PR #41) is merged per PR body |

## Correctness Findings

### `internal/paths/paths.go`

- `NewPathTrackerWithAddr` delegates to `NewPathTracker`. The alpha guard fires
  before any allocation, so an invalid alpha still panics from the shared code
  path — the reason the strengthened oracle in
  `TestBC_2_06_003_NewPathTrackerWithAddr_RejectsInvalidAlpha` correctly
  matches on the substring `"alpha"`.
- `t.routerAddr = addr` is written **after** `NewPathTracker` returns, before
  the tracker is exposed to any goroutine. The field is thereafter only read,
  and always under `t.mu` in `Snapshot()`. This satisfies the Go memory model
  because the write happens-before any subsequent goroutine's mutex acquisition
  (assuming the caller publishes the tracker safely). No data race under
  `-race` per the PR body evidence.
- Doc comment on `PathTracker` and both constructor doc-comments are accurate
  and explicit about the addr-less sentinel semantics.

### `internal/metrics/handlers.go`

- The single-line substitution
  `PathEntryFromSnapshot(pathID, snap.RouterAddr, snap)` is exactly the seam
  described in RULING-W6TB-B Option A. No other logic changes.
- `PathEntryFromSnapshot` signature is unchanged; the second parameter still
  accepts `routerAddr string`. This preserves the existing test surface.
- Doc-comment BC version bumps are consistent (v1.14 → v1.15 across all four
  affected functions).

### Test files

- `TestBC_2_06_003_RouterAddr_ConcurrentSnapshot` correctly exercises the
  `Snapshot()`/`OnProbe()` interleave; the RouterAddr equality check is a
  stable oracle (immutable post-construction), and the comment properly
  distinguishes vehicle-vs-target per RULING-W6TB-K F-P4L2-02.
- `TestVP047_RouterAddrNonEmpty` Part A adds the host:port regex oracle
  (`^[^:]+:[0-9]+$`) that `TestPathsList_PassesRouterAddr` lacks, justifying
  the split. Part B exercises the constructor path end-to-end through
  `PathsList`.
- `TestVP047_FieldSwapOracle` seed change from `"abcdefghi"` → `"127.0.0.1:9000"`
  is correct per RULING-W6TB-F Ruling 2. The `pathID = "000111222"` remains
  digit-only, so the swap oracle (host:port contains `.` and `:`, pathID has
  neither) is preserved and strengthened by the added regex assertion.
- `mustNewPathTrackerWithAddr` is cleanly reduced to a direct constructor call;
  no stale recover-guard scaffolding remains after Pass-5/Pass-6 cleanup.

### Spec

- `BC-2.06.003.md` v1.15 changelog entry explicitly names the DRIFT closure
  and retracts the sentinel-permission clause. Remaining mentions of
  `DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER` in the file are historical (v1.9 row)
  and closure-status (line 60), which is correct.

## Non-Blocking Observations

**NIT (not blocking) — `internal/paths/paths.go:130`, `NewPathTrackerWithAddr`:**
Consistent with the SEC-001 LOW in the PR body, this constructor accepts any
string as `addr` — including an empty string, whitespace, or something
structurally invalid like `"not-a-host:port"`. Callers relying on the
`^[^:]+:[0-9]+$` shape in tests won't get that guarantee from the constructor
itself; production wiring will need to validate before calling. The PR body
correctly defers this to `S-BL.PATH-TRACKER-WIRING` — flagging only for
awareness.

**NIT (not blocking) — `internal/paths/paths.go`, `NewPathTrackerWithAddr`
implementation:**
`t.routerAddr = addr` is written without holding `t.mu`. This is safe because
the tracker has not escaped yet (still on the constructor's stack), but a
future refactor that (e.g.) registered the tracker inside the constructor
before this assignment would silently introduce a race the tests would not
catch. A one-line `// safe: not yet published to other goroutines` comment on
the assignment would future-proof against that regression. Not required.

**NIT (not blocking) — `TestPathsList_RouterAddrEmptyForAddrLessSnapshot`
naming:**
The name says "for addr-less snapshot" but the test constructs the snapshot
literal with `RouterAddr: ""` rather than going through `NewPathTracker`. The
name is slightly aspirational; the assertion is nonetheless correct
(empty-in → empty-out through the handler + JSON). Non-blocking.

## Verification of Concerns

- **Regression risk to `NewPathTracker` call sites:** `grep -rn "NewPathTracker\b"
  internal/ cmd/ --include="*.go" | grep -v "_test.go\|WithAddr"` returns only
  the definition and doc-comment in `paths.go` itself. No production caller
  breaks.
- **DRIFT annotation removal:** confirmed via grep on the spec — only historical
  (v1.9 changelog row) and closure-status references remain. The active
  sentinel-permission clause on PC-1 is retracted.
- **VP-047 oracle flip:** the field-swap seed change and the new
  `TestVP047_RouterAddrNonEmpty` Parts A+B together implement RULING-W6TB-F
  Rulings 1 and 2 correctly.
- **`NewPathTracker` unchanged:** the diff shows zero deletions in the original
  constructor body; only an appended paragraph in the doc-comment. Confirmed.

## Decision

APPROVE. No blocking findings. Three NITs above are non-blocking and can be
addressed in follow-ons or ignored at author discretion.
