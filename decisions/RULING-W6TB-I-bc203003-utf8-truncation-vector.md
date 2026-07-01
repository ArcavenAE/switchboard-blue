---
artifact_id: RULING-W6TB-I-bc203003-utf8-truncation-vector
document_type: decision
level: ops
version: "1.0"
status: final
producer: product-owner
timestamp: 2026-07-01T00:00:00
updated: 2026-07-01T00:00:00
cycle: v1.0.0-greenfield
stories_in_scope: [S-7.02]
closes_findings: [F-P4L3-04]
referenced_by:
  - .factory/specs/behavioral-contracts/ss-03/BC-2.03.003.md
  - .factory/stories/S-7.02-session-discovery.md
---

# Ruling W6TB-I — BC-2.03.003 UTF-8 Truncation Test Vector (chars vs. bytes)

**Adjudicator:** product-owner
**Date:** 2026-07-01
**Trigger:** S-7.02 Pass-4 finding F-P4L3-04 (LOW-3)

---

## Finding Summary

BC-2.03.003 EC-001 canonical test vector table describes the oversize-name case as:
> `{name: first 252 chars + "…", attached:false, quality:green}`

Story S-7.02 AC-004b (added in Pass-3 as M-2 resolution) says:
> "truncates to 252 bytes and appends '…' (U+2026 HORIZONTAL ELLIPSIS, 3 UTF-8 bytes = 255 bytes total)"

For session names consisting entirely of ASCII, "252 chars" and "252 bytes" are
identical. For session names containing multi-byte UTF-8 codepoints (e.g.,
Japanese, emoji), they differ. If the 252-byte boundary lands in the middle of a
multi-byte codepoint, appending "…" would produce invalid UTF-8. The story AC is
more precise. The BC canonical vector is ambiguous and, for non-ASCII names,
potentially incorrect.

---

## Options Considered

**Option A (adopted):** Bump BC-2.03.003 to v1.3. Correct the canonical test
vector to state "first 252 bytes on a valid UTF-8 rune boundary + '…'", with
explicit handling for mid-codepoint cuts. This is the source-of-truth fix: the BC
should be precise enough to be tested independently of the story.

**Option B:** Leave the BC vague and rely solely on story AC-004b + implementation
for precision. Rejected: a BC that requires external story context to resolve
ambiguity is untestable in isolation, violates the SMART requirement, and would
fail the next adversarial pass.

---

## Decision: Option A

**Ruling: Bump BC-2.03.003 to v1.3. Replace the ambiguous "252 chars" canonical
test vector with the byte-precise, rune-safe wording from AC-004b.**

---

## Rationale

The story AC (AC-004b, v1.3) already contains the correct specification:
252 bytes + "…" (3 bytes) = 255 bytes total, and the truncation point must align
to a valid UTF-8 rune boundary (walking backward from byte 252 if the cut lands
mid-codepoint). The BC must reflect this to be independently testable by
test-writer and verifiable by VP-055. "Chars" is a Go source-level concept
(`rune` count) and is not a byte count; using it in a wire-format contract is a
specification defect.

The BC postcondition 2 ("Session names are UTF-8 encoded, maximum 255 bytes")
is correct. Only the canonical test vector in EC-001 and the test vector table
use "chars" — those two sites require correction.

---

## Files to Modify

### `.factory/specs/behavioral-contracts/ss-03/BC-2.03.003.md`

Bump `version` frontmatter from `"1.2"` to `"1.3"`.

**EC-001 row in the Edge Cases table** — replace:

```
| EC-001 | Session name contains non-ASCII characters (e.g., Japanese) | UTF-8 encoded; max 255 bytes enforced. If tmux session name exceeds 255 bytes (unusual), it is truncated with "…" indicator. |
```

with:

```
| EC-001 | Session name contains non-ASCII characters (e.g., Japanese) | UTF-8 encoded; max 255 bytes enforced. If tmux session name exceeds 255 bytes, `Encode` truncates to a valid UTF-8 rune boundary at or before byte 252, then appends "…" (U+2026, 3 UTF-8 bytes), yielding a total of ≤255 bytes of valid UTF-8. If byte 252 falls mid-codepoint, the truncation walks back to the preceding rune start. `Decode` does not truncate. |
```

**Canonical Test Vectors table** — replace the oversize row:

```
| Session name 256 bytes long | {name: first 252 chars + "…", attached:false, quality:green} | edge-case |
```

with:

```
| Session name 256 bytes of ASCII (e.g., 256 × "x") | {name: first 252 bytes + "…" = 255 bytes total, attached:false, quality:green} | edge-case |
| Session name with multi-byte codepoints totalling 260 bytes; byte 252 falls mid-codepoint | {name: truncated at last rune boundary ≤252 bytes + "…" = ≤255 bytes valid UTF-8, attached:false, quality:green} | edge-case |
```

**Changelog** — append:

```
| v1.3 | 2026-07-01 | product-owner | RULING-W6TB-I (F-P4L3-04): EC-001 and canonical test vector corrected from "252 chars" to "252 bytes on a valid UTF-8 rune boundary". Aligns BC with S-7.02 AC-004b byte-precise semantics. Two edge-case test vector rows replace the single ambiguous row. |
```

---

## Cross-References

| Artifact | Relationship |
|----------|-------------|
| S-7.02 AC-004b | Source of the byte-precise semantics adopted here |
| VP-055 | Must regenerate `genSessionName` to include multi-byte names with byte-boundary edge; the corrected BC is the input to VP-055 v1.2 (see RULING-W6TB-J) |
| RULING-W6TB-J | Companion ruling: VP-055 semantics update (truncate vs. reject) |

---

## Downstream Dispatch Table

| Artifact | Change | Agent | When |
|----------|--------|-------|------|
| `.factory/specs/behavioral-contracts/ss-03/BC-2.03.003.md` | v1.2→v1.3; EC-001 + test vector table as specified above | product-owner (this ruling) | This burst |
| `VP-055.md` | v1.1→v1.2 per RULING-W6TB-J | architect | After this ruling |

---

## Decision Log

| Date | Actor | Entry |
|------|-------|-------|
| 2026-07-01 | product-owner | Option A adopted. "252 chars" is a specification defect for any non-ASCII name. The byte-precise, rune-safe wording from AC-004b is correct and must be the BC source of truth. Two test vector rows replace one to cover both ASCII and multi-byte cases independently. |
