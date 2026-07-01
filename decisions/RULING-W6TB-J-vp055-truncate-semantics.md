---
artifact_id: RULING-W6TB-J-vp055-truncate-semantics
document_type: decision
level: ops
version: "1.0"
status: final
producer: product-owner
timestamp: 2026-07-01T00:00:00
updated: 2026-07-01T00:00:00
cycle: v1.0.0-greenfield
stories_in_scope: [S-7.02]
closes_findings: [O-P4L3-01]
referenced_by:
  - .factory/specs/verification-properties/VP-055.md
  - .factory/specs/behavioral-contracts/ss-03/BC-2.03.003.md
  - .factory/stories/S-7.02-session-discovery.md
  - .factory/decisions/RULING-W6TB-I-bc203003-utf8-truncation-vector.md
---

# Ruling W6TB-J — VP-055 Truncate vs. Reject Semantics

**Adjudicator:** product-owner
**Date:** 2026-07-01
**Trigger:** S-7.02 Pass-4 finding O-P4L3-01 (OBSERVATION)

---

## Finding Summary

VP-055 v1.1 Property Statement contains:

```
∀ SessionName s where len(s) == 0 or len(s) > 255 or !utf8.Valid(s):
  adv.SessionName = s → Encode returns validation error
```

Story S-7.02 AC-004b (Pass-3 M-2 resolution) changed `Encode` semantics: names
exceeding 255 bytes are now **truncated** to a valid UTF-8 rune boundary at or
before byte 252, then appended with "…" (3 bytes), rather than rejected with a
validation error. The current VP-055 property `TestPropPresenceAdvertisement_RejectsInvalidName`
generates names >255 bytes and asserts `err != nil`. Under the truncation
implementation `err == nil`, so the test would fail on the new code, creating a
false test failure.

Empty names (len == 0) and non-UTF-8 inputs remain validation errors — the
split is: oversize names are truncated to a valid representation; structurally
invalid inputs (empty, non-UTF-8) are rejected.

---

## Options Considered

**Option A (adopted):** Bump VP-055 to v1.2. Split the combined "reject invalid
name" property into two separate properties:
  (i) `TestPropPresenceAdvertisement_RejectsEmptyOrInvalidUTF8` — asserts `err != nil`
      for names that are empty (`len == 0`) or not valid UTF-8.
  (ii) `TestPropPresenceAdvertisement_TruncatesOversize` — asserts `err == nil` for
       names >255 bytes, resulting encoded name is ≤255 bytes of valid UTF-8, ends
       with "…" (U+2026), and the non-ellipsis prefix matches the input's byte prefix
       up to the truncation rune boundary.

This is the unified property statement approach: one proptest file, two named
properties, clear preconditions for each. The round-trip property is unchanged
(it only generates names already in 1..255 byte range, so it is unaffected).

**Option B:** Simply narrow the reject property to empty-only, add a new property
for truncation. Functionally equivalent to A; Option A is preferred because it
makes the structural split explicit in the test name ("RejectsEmptyOrInvalidUTF8"
vs. "RejectsInvalidName"), reducing the risk that a future reader narrows the
generator back to include oversize names.

**Option C:** Retain the existing property, add a new VP for truncation. Rejected:
VP proliferation for a single codec function is unwarranted; the two behaviors are
complementary aspects of the same contract (BC-2.03.003 postcondition 2) and
belong in one VP.

---

## Decision: Option A

**Ruling: Bump VP-055 to v1.2. Replace `TestPropPresenceAdvertisement_RejectsInvalidName`
with two separate properties as specified below. Round-trip property is unchanged.**

---

## New VP-055 Property Statement (v1.2)

Replace the stanza:

```
∀ SessionName s where len(s) == 0 or len(s) > 255 or !utf8.Valid(s):
  adv.SessionName = s → Encode returns validation error
```

with:

```
∀ SessionName s where len(s) == 0 or !utf8.Valid(s):
  adv.SessionName = s → Encode returns validation error (reject-empty-or-invalid-utf8)

∀ SessionName s where len(s) > 255:
  let encoded, err = adv with SessionName = s → Encode()
  assert err == nil                                          (truncation, not rejection)
  assert len(encoded_name) <= 255                           (output within bounds)
  assert utf8.Valid(encoded_name)                           (output is valid UTF-8)
  assert strings.HasSuffix(encoded_name, "…")              (ellipsis appended)
  let prefix = encoded_name[:len(encoded_name)-3]           (strip 3-byte ellipsis)
  assert utf8.Valid([]byte(prefix))                         (prefix rune-boundary safe)
  assert strings.HasPrefix(s, prefix)                       (prefix is a true prefix of input)
```

---

## New Proof Harness Skeleton (v1.2 delta)

Replace `TestPropPresenceAdvertisement_RejectsInvalidName` with:

```go
// TestPropPresenceAdvertisement_RejectsEmptyOrInvalidUTF8 verifies that Encode
// returns a validation error for session names that are empty or not valid UTF-8.
// (VP-055 v1.2 — RULING-W6TB-J: oversize names are truncated, not rejected.)
func TestPropPresenceAdvertisement_RejectsEmptyOrInvalidUTF8(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Generate names that are empty or contain invalid UTF-8 byte sequences.
	genInvalidName := gen.OneGenOf(
		gen.Const(""), // empty
		gen.SliceOf(gen.UInt8()).Map(func(bs []byte) string {
			// Inject an invalid UTF-8 byte sequence (0xFF is never valid UTF-8).
			return string(append(bs, 0xFF))
		}),
	)

	properties.Property(
		"encode rejects session names that are empty or contain invalid UTF-8",
		prop.ForAll(
			func(name string) bool {
				adv := discovery.PresenceAdvertisement{
					SessionName:      name,
					AttachStatus:     discovery.AttachStatusOpen,
					QualityIndicator: discovery.QualityGreen,
				}
				_, err := adv.Encode()
				return err != nil // must return an error
			},
			genInvalidName,
		),
	)

	properties.TestingRun(t)
}

// TestPropPresenceAdvertisement_TruncatesOversize verifies that Encode truncates
// session names exceeding 255 bytes to a valid UTF-8 rune boundary ≤252 bytes,
// appends "…" (U+2026, 3 UTF-8 bytes), and returns err == nil.
// (VP-055 v1.2 — RULING-W6TB-J; BC-2.03.003 v1.3 EC-001.)
func TestPropPresenceAdvertisement_TruncatesOversize(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Generate valid UTF-8 strings that exceed 255 bytes.
	genOversizeName := gen.AnyString().Map(func(s string) string {
		for len([]byte(s)) <= 255 {
			s += "x"
		}
		return s
	}).SuchThat(func(s string) bool {
		return utf8.ValidString(s) && len([]byte(s)) > 255
	})

	ellipsis := "…" // U+2026, 3 UTF-8 bytes

	properties.Property(
		"encode truncates oversize names to ≤255 bytes of valid UTF-8 with ellipsis",
		prop.ForAll(
			func(name string) bool {
				adv := discovery.PresenceAdvertisement{
					SessionName:      name,
					AttachStatus:     discovery.AttachStatusOpen,
					QualityIndicator: discovery.QualityGreen,
				}
				encoded, err := adv.Encode()
				if err != nil {
					return false // truncation must not error
				}

				decoded, decErr := discovery.Decode(encoded)
				if decErr != nil {
					return false
				}
				result := decoded.SessionName

				// Output must be within the 255-byte limit.
				if len([]byte(result)) > 255 {
					return false
				}
				// Output must be valid UTF-8.
				if !utf8.ValidString(result) {
					return false
				}
				// Output must end with "…".
				if !strings.HasSuffix(result, ellipsis) {
					return false
				}
				// Prefix (minus ellipsis) must be a valid UTF-8 prefix of the input.
				prefix := result[:len(result)-len(ellipsis)]
				if !utf8.ValidString(prefix) {
					return false
				}
				return strings.HasPrefix(name, prefix)
			},
			genOversizeName,
		),
	)

	properties.TestingRun(t)
}
```

---

## Feasibility Assessment Update (v1.2)

| Property | Generator | Bounded? | Complexity |
|----------|-----------|----------|-----------|
| RejectsEmptyOrInvalidUTF8 | empty string + invalid-UTF-8 byte injection | no | low |
| TruncatesOversize | valid UTF-8 strings padded to >255 bytes | no | low–medium |
| RoundTrip (unchanged) | valid UTF-8 names 1–255 bytes | no | low |

---

## Cross-References

| Artifact | Relationship |
|----------|-------------|
| BC-2.03.003 v1.3 | Source contract corrected per RULING-W6TB-I |
| S-7.02 AC-004b | Byte-precise truncation semantics that VP-055 v1.2 now verifies |
| RULING-W6TB-I | Companion ruling: BC-2.03.003 test vector correction |

---

## Files to Modify

### `.factory/specs/verification-properties/VP-055.md`

1. Bump `version` frontmatter from `"1.1"` to `"1.2"`.
2. Replace the reject-invalid stanza in Property Statement as specified above.
3. Replace `TestPropPresenceAdvertisement_RejectsInvalidName` in the Proof Harness
   Skeleton with the two new functions above.
4. Update Source Contract — EC-001 row to cite BC-2.03.003 v1.3 (truncation).
5. Append to Lifecycle table:

```
| v1.2 | 2026-07-01 | product-owner | RULING-W6TB-J (O-P4L3-01): split "reject invalid name" property into (i) RejectsEmptyOrInvalidUTF8 (empty and invalid-UTF-8 inputs → error) and (ii) TruncatesOversize (>255-byte valid UTF-8 inputs → truncated to ≤255 bytes with "…" suffix, err == nil). Aligns with S-7.02 AC-004b truncation semantics and BC-2.03.003 v1.3 EC-001. Round-trip property unchanged. |
```

---

## Downstream Dispatch Table

| Artifact | Change | Agent | When |
|----------|--------|-------|------|
| `.factory/specs/verification-properties/VP-055.md` | v1.1→v1.2 per property spec above | architect | This burst (after BC-2.03.003 v1.3 lands) |
| `BC-2.03.003.md` | v1.2→v1.3 per RULING-W6TB-I | product-owner (RULING-W6TB-I) | Concurrent |

---

## Decision Log

| Date | Actor | Entry |
|------|-------|-------|
| 2026-07-01 | product-owner | Option A adopted. The semantic change from reject→truncate for oversize names is a deliberate design decision (AC-004b, Pass-3 M-2). VP-055 must reflect the actual contract. Splitting into two named properties makes the preconditions unambiguous and prevents generator drift. The round-trip property is unaffected (its generator already constrains to 1–255 bytes). Option C rejected: two properties of one codec belong in one VP. |
