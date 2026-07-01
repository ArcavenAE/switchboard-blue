---
artifact_id: RULING-W6TB-F-s-bl-router-addr-vp047
document_type: decision
level: ops
version: "1.0"
status: final
producer: product-owner
timestamp: 2026-07-01T00:00:00
updated: 2026-07-01T00:00:00
cycle: v1.0.0-greenfield
stories_in_scope: [S-BL.ROUTER-ADDR]
closes_findings: [F-L3-001, F-LENS2-01]
supersedes: []
referenced_by:
  - .factory/stories/S-BL.ROUTER-ADDR.md
  - .factory/specs/verification-properties/VP-047.md
  - .factory/decisions/wave-6-tranche-a-scope-rulings.md
---

# Ruling W6TB-F — VP-047 Oracle Flip and Field-Swap Seed Correction (S-BL.ROUTER-ADDR Pass-1)

**Adjudicator:** product-owner
**Date:** 2026-07-01
**Trigger:** S-BL.ROUTER-ADDR Pass-1 LENS-3 finding F-L3-001 (HIGH) and LENS-2
finding F-LENS2-01 (MEDIUM)

---

## Finding Summary

### F-L3-001 (HIGH)
VP-047 v1.3 at lines 38, 47, and 89 retains Ruling-1 interim language permitting
`router_addr == ""` and citing `DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER`. BC-2.06.003
is already at v1.15 (DRIFT annotation removed; `""` sentinel-permission clause
retracted; per changelog entry 2026-07-01 stub-architect). Story S-BL.ROUTER-ADDR
AC-005 explicitly commits to flipping the VP-047 oracle from `router_addr == ""`
to `router_addr == <stub-addr>` for `NewPathTrackerWithAddr`-constructed paths.
The story's File Structure MODIFY table (lines 189–196) does NOT include VP-047.md,
creating a gap: implementer is instructed (Task step 12, AC-005) to flip the
oracle but has no spec target authorizing the VP-047 file change.

### F-LENS2-01 (MEDIUM)
`TestVP047_FieldSwapOracle` in `internal/metrics/handlers_test.go` (line 872)
seeds `routerAddr := "abcdefghi"` — an alpha-only string with no colon or digits.
This is a structurally-invalid `host:port`. The field-swap oracle is logically
sound (non-overlapping character sets achieve distinguishability), but using an
invalid address creates breakage risk if address validation is ever added to
`PathEntryFromSnapshot` or its callers.

---

## Ruling 1 — VP-047 Oracle Flip: Posture A (FLIP IN-STORY)

**Decision: Posture A. VP-047 MUST be updated to v1.4 as part of S-BL.ROUTER-ADDR.**

### Rationale

Posture B (defer to S-BL.PATH-TRACKER-WIRING) is rejected on two grounds:

**Ground 1 — Contradicted source of truth.** BC-2.06.003 is already at v1.15 with
the DRIFT annotation removed and the `""` sentinel-permission clause retracted. VP-047
v1.3 now actively contradicts the BC it traces. A VP that contradicts its source BC
is a consistency failure, not a deferred obligation. The gap is not "VP not yet
updated" — it is "VP says one thing, BC says the opposite."

**Ground 2 — Story already owns the flip.** S-BL.ROUTER-ADDR Task step 12 reads:
"Flip VP-047 AC-006 oracle: `router_addr == ""` → `router_addr == <stub-addr>` for
`NewPathTrackerWithAddr`-constructed paths (AC-005)." AC-005 names
`TestVP047_RouterAddrNonEmpty`. The Token Budget table includes "VP-047 (AC-006
oracle flip) ~400 tokens." The story intends to flip the oracle; the spec target
file (VP-047.md) is simply missing from the File Structure table. Correcting this
is a story frontmatter delta (add one MODIFY row), not a scope change.

**Scope boundary preserved.** The unit-scope constraint of RULING-W6TB-B is not
altered by this ruling. VP-047 v1.4 will continue to state that end-to-end
observability (non-empty `router_addr` from a live daemon via `sbctl paths list`)
requires production `PathTracker` wiring via S-BL.PATH-TRACKER-WIRING. The flip
applies only to the unit-test oracle: paths constructed via `NewPathTrackerWithAddr`
in the `PathsListSource` stub MUST yield a non-empty `router_addr`. Paths
constructed via addr-less `NewPathTracker` (including all current production paths)
continue to yield `router_addr: ""`.

### VP-047 v1.3 → v1.4 Change Specification

Spec-steward applies the following changes when producing VP-047 v1.4:

1. **Property Statement (line 38):** Replace:
   ```
   p.router_addr is present (key MUST be present; value MAY be `""` in Wave 6
   interim state pending `PathSnapshot.RouterAddr` enrichment —
   DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER)
   ```
   With:
   ```
   p.router_addr is present and equals `PathSnapshot.RouterAddr`;
   `""` only for paths constructed via addr-less `NewPathTracker`
   (end-to-end observability with non-empty value requires
   S-BL.PATH-TRACKER-WIRING)
   ```

2. **Note (Ruling-1) block (lines 47–47):** Retract entirely. Replace with:
   ```
   **Note (post-S-BL.ROUTER-ADDR):** `router_addr` MUST equal `PathSnapshot.RouterAddr`.
   `""` is only valid for paths whose `PathTracker` was constructed via addr-less
   `NewPathTracker`. Integration tests using a `PathsListSource` stub with
   `NewPathTrackerWithAddr("127.0.0.1:9000", ...)` MUST assert `router_addr ==
   "127.0.0.1:9000"`, not accept `""`. DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER is closed
   (BC-2.06.003 v1.15; S-BL.ROUTER-ADDR).
   ```

3. **Proof Harness Skeleton — `pathEntry` struct comment (line 88–90):** Replace:
   ```go
   // RouterAddr: key MUST be present; value may be "" in Wave-6 interim state
   // (DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER) until PathSnapshot.RouterAddr is populated.
   // The test asserts key presence and accepts "" as a valid value.
   RouterAddr *string      `json:"router_addr"`
   ```
   With:
   ```go
   // RouterAddr: MUST equal PathSnapshot.RouterAddr. Non-empty when PathTracker
   // constructed via NewPathTrackerWithAddr. Empty string only for addr-less
   // NewPathTracker-constructed paths. DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER closed.
   RouterAddr *string      `json:"router_addr"`
   ```

4. **Integration test assertion block (lines 156–159):** Replace:
   ```go
   // router_addr key must be present; empty string is valid in Wave-6 interim state
   // (DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER). Do NOT reject "" as an error.
   if p.RouterAddr == nil {
       t.Errorf("path[%d]: missing router_addr field (key must be present; value may be empty)", i)
   }
   ```
   With:
   ```go
   // router_addr must equal PathSnapshot.RouterAddr. Non-empty for paths constructed
   // via NewPathTrackerWithAddr. Assert key presence; value check depends on stub.
   if p.RouterAddr == nil {
       t.Errorf("path[%d]: missing router_addr field", i)
   }
   ```

5. **Frontmatter:** Bump `version` to `"1.4"`. Add `2026-07-01T00:00:00` to `modified:`.

6. **Changelog:** Add entry:
   ```
   | 1.4 | 2026-07-01 | spec-steward | F-L3-001 (S-BL.ROUTER-ADDR Pass-1 LENS-3):
   retract Ruling-1 interim clauses (DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER). Property
   Statement updated: router_addr MUST equal PathSnapshot.RouterAddr; "" valid only
   for addr-less NewPathTracker paths. Proof harness comment updated. Integration test
   assertion updated. DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER closed (BC-2.06.003 v1.15).
   Ruling authority: RULING-W6TB-F §Ruling 1. |
   ```

### wave-6-tranche-a-scope-rulings.md Ruling-1 Annotation

Spec-steward MUST add the following annotation to the Ruling-1 section of
`wave-6-tranche-a-scope-rulings.md`:

At the end of the Ruling-1 section (after "### Follow-on Stories"), add:

```markdown
### Ruling-1 Status (post-S-BL.ROUTER-ADDR)

SUPERSEDED IN PART by RULING-W6TB-B (2026-07-01) and RULING-W6TB-F (2026-07-01).

- The `""` sentinel-permission clause is retracted. BC-2.06.003 v1.15 removed the
  interim annotation. VP-047 v1.4 removes the Ruling-1 note.
- DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER is closed. STATE.md updated.
- Ruling-1's scope-A decision (ship empty string as interim) and follow-on story
  stub (S-BL.ROUTER-ADDR) were the correct outcome; that story is now
  ready-for-red-gate and owns both the field enrichment and the VP oracle flip.
- The integration observability deferral (non-empty router_addr from live daemon
  deferred to S-BL.PATH-TRACKER-WIRING) remains active per RULING-W6TB-B.

SUPERSEDED-BY: RULING-W6TB-B (seam decision), RULING-W6TB-F (VP oracle + seed).
```

---

## Ruling 2 — F-LENS2-01: Field-Swap Oracle Seed (STRENGTHEN IN-STORY)

**Decision: Strengthen `routerAddr` seed to a valid `host:port`. Apply as part of
S-BL.ROUTER-ADDR, file `internal/metrics/handlers_test.go`.**

### Rationale

`"abcdefghi"` is valid for the test's present purpose (non-overlapping character
set distinguishability) but is not a valid `host:port`. The test comment on line
868 states: "path_id uses only digits; router_addr uses only alpha chars." This
achieves field-swap detection but at the cost of structural invalidity.

A valid `host:port` — specifically `"127.0.0.1:9000"` — satisfies the same oracle
invariant: it does not consist solely of decimal digits (contains `.` and `:`
characters); it is non-overlapping with `"000111222"` (digit-only path_id); any
field swap would place the IP:port string in `path_id` and the digit string in
`router_addr`, both of which are detectable. Additionally, `"127.0.0.1:9000"` is
the canonical stub address used throughout the story's AC set (AC-001, AC-002,
AC-003), providing test-suite consistency.

The strengthening is low-risk: `PathEntryFromSnapshot` performs no address
validation; it is pure JSON serialization. The oracle logic is identical; only the
seed value changes.

### Change Specification

In `internal/metrics/handlers_test.go`, the `TestVP047_FieldSwapOracle` function:

1. **Line 868 (comment):** Replace:
   ```go
   // path_id uses only digits; router_addr uses only alpha chars.
   // If the fields were swapped, the digit-only string would appear in
   // router_addr and the alpha-only string in path_id.
   ```
   With:
   ```go
   // path_id uses only digits; router_addr uses a valid host:port (contains
   // '.' and ':' characters not present in path_id). If the fields were swapped,
   // the digit-only string would appear in router_addr and the host:port string
   // in path_id — both are detectable.
   ```

2. **Line 872:** Replace:
   ```go
   routerAddr := "abcdefghi"
   ```
   With:
   ```go
   routerAddr := "127.0.0.1:9000"
   ```

3. **Line 903 (comment):** Replace:
   ```go
   // router_addr must contain only alpha chars.
   ```
   With:
   ```go
   // router_addr must equal the canonical stub host:port (contains '.' and ':').
   ```

This change falls within S-BL.ROUTER-ADDR's existing MODIFY scope on
`internal/metrics/handlers_test.go` (File Structure table line 194).

---

## Story S-BL.ROUTER-ADDR Frontmatter Delta

Story version bumps from `1.0-ready-for-red-gate` to `1.1-ready-for-red-gate`.

### File Structure MODIFY Row Addition

Add the following row to the File Structure Requirements table (insert after line 195,
before the BC-2.06.003.md row):

```markdown
| .factory/specs/verification-properties/VP-047.md | MODIFY | v1.3→v1.4: retract Ruling-1 interim clauses; property statement says router_addr MUST equal PathSnapshot.RouterAddr; "" valid only for addr-less NewPathTracker; DRIFT closed (Ruling RULING-W6TB-F §Ruling 1) |
```

### Frontmatter Delta

```yaml
revision: "1.1-ready-for-red-gate"
```

Add to `inputDocuments:`:
```yaml
  - '.factory/decisions/RULING-W6TB-F-s-bl-router-addr-vp047.md'
```

### Changelog Addition

```markdown
| 1.1-ready-for-red-gate | 2026-07-01 | product-owner | RULING-W6TB-F: add VP-047.md
MODIFY row to File Structure (spec target for AC-005 oracle flip was missing — F-L3-001).
Strengthen TestVP047_FieldSwapOracle seed: routerAddr "abcdefghi" → "127.0.0.1:9000"
(valid host:port; non-overlapping oracle preserved — F-LENS2-01). Bump revision. |
```

---

## STATE.md Delta

Spec-steward MUST update STATE.md as follows:

In the open drift items table, change row for `DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER`:
```
| DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER | ... | backlog |
```
To:
```
| ~~DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER~~ | closed | BC-2.06.003 v1.15 (S-BL.ROUTER-ADDR). PathSnapshot.RouterAddr field + NewPathTrackerWithAddr constructor added; PathsList pass-through replaced hard-coded "". DRIFT closed when S-BL.ROUTER-ADDR merges. VP-047 v1.4 Ruling-1 interim language retracted (RULING-W6TB-F). | closed |
```

Note: the STATE.md entry flags closure as contingent on S-BL.ROUTER-ADDR merge,
consistent with RULING-W6TB-B §Observable Coverage Matrix row "BC-2.06.003 PC-1
`""` sentinel annotation removed — Yes."

---

## Downstream Dispatch Table

| Artifact | Change | Agent | When |
|----------|--------|-------|------|
| `VP-047.md` | v1.3 → v1.4 per §Ruling 1 change spec | spec-steward | Same burst as ruling |
| `S-BL.ROUTER-ADDR.md` | v1.0 → v1.1 per §Story Frontmatter Delta | story-writer | Same burst as ruling |
| `wave-6-tranche-a-scope-rulings.md` | Add Ruling-1 SUPERSEDED-BY annotation | spec-steward | Same burst as ruling |
| `STATE.md` | DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER → closed-pending-merge | state-manager | Same burst as ruling |

---

## Decision Log

| Date | Actor | Entry |
|------|-------|-------|
| 2026-07-01 | product-owner | Initial ruling. Posture A (VP-047 flip in-story) on F-L3-001. Seed strengthening on F-LENS2-01. Story v1.0 → v1.1 frontmatter delta. VP-047 v1.3 → v1.4 change spec issued to spec-steward. |
